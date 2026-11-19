package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Spec mfa-opcional, R27 a R31 e R34: o uso de um codigo da lista ativa desliga o MFA da pessoa. Ponta a ponta pelo roteador, com um Cognito
// falso e o DynamoDB em memoria (que aqui tambem conta leituras e escritas).

// ---------- o Cognito falso ----------

type identidadeFalsa struct {
	pessoas         map[string]domain.PessoaMFA // por e-mail
	buscas          []string                    // e-mails recebidos (ja normalizados)
	desligados      []string                    // subs desligados
	falharBusca     error
	falharDesligar  error
	antesDeDesligar func()
}

func (i *identidadeFalsa) BuscarPessoa(_ context.Context, email string) (domain.PessoaMFA, bool, error) {
	i.buscas = append(i.buscas, email)
	if i.falharBusca != nil {
		return domain.PessoaMFA{}, false, i.falharBusca
	}
	p, ok := i.pessoas[email]
	return p, ok, nil
}

func (i *identidadeFalsa) DesligarMFA(_ context.Context, sub string) error {
	if i.antesDeDesligar != nil {
		i.antesDeDesligar()
	}
	if i.falharDesligar != nil {
		return i.falharDesligar
	}
	i.desligados = append(i.desligados, sub)
	return nil
}

// ---------- o DynamoDB que conta ----------

type contaBanco struct {
	*bancoFake
	leituras, escritas int
	antesDeAtualizar   func()
	falharLeitura2     error
}

func (c *contaBanco) GetItem(ctx context.Context, in *dynamodb.GetItemInput, o ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	c.leituras++
	if c.falharLeitura2 != nil {
		return nil, c.falharLeitura2
	}
	return c.bancoFake.GetItem(ctx, in, o...)
}

func (c *contaBanco) PutItem(ctx context.Context, in *dynamodb.PutItemInput, o ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	c.escritas++
	return c.bancoFake.PutItem(ctx, in, o...)
}

func (c *contaBanco) DeleteItem(ctx context.Context, in *dynamodb.DeleteItemInput, o ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	c.escritas++
	return c.bancoFake.DeleteItem(ctx, in, o...)
}

func (c *contaBanco) UpdateItem(ctx context.Context, in *dynamodb.UpdateItemInput, o ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	if h := c.antesDeAtualizar; h != nil {
		c.antesDeAtualizar = nil
		h()
	}
	c.escritas++
	return c.bancoFake.UpdateItem(ctx, in, o...)
}

func (c *contaBanco) TransactWriteItems(ctx context.Context, in *dynamodb.TransactWriteItemsInput, o ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	c.escritas++
	return c.bancoFake.TransactWriteItems(ctx, in, o...)
}

func (c *contaBanco) Query(ctx context.Context, in *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	c.leituras++
	return c.bancoFake.Query(ctx, in, o...)
}

func (c *contaBanco) zerar() { c.leituras, c.escritas = 0, 0 }

// ---------- o cenario ----------

type cenarioRec struct {
	f   *bancoFake
	db  *contaBanco
	id  *identidadeFalsa
	srv *Servidor
}

const ipA = "203.0.113.9"

func novoRec(t *testing.T) *cenarioRec {
	t.Helper()
	f := cenario(t) // monta srvTeste sobre o mesmo banco: mfaGerar e mfaConfirmar o usam
	db := &contaBanco{bancoFake: f}
	id := &identidadeFalsa{pessoas: map[string]domain.PessoaMFA{
		"ana@exemplo.com": {Sub: "ana", Habilitada: true, TemMFA: true},
		"bia@exemplo.com": {Sub: "bia", Habilitada: true, TemMFA: true},
		"sem@exemplo.com": {Sub: "sem", Habilitada: true, TemMFA: false},
		"off@exemplo.com": {Sub: "off", Habilitada: false, TemMFA: true},
	}}
	srv := novoServidor(db, "", relogioTeste, notifTeste, "")
	srv.usarIdentidade(id)
	return &cenarioRec{f: f, db: db, id: id, srv: srv}
}

// lista gera e confirma uma lista ativa para a pessoa e devolve os codigos como a tela os mostra.
func (c *cenarioRec) lista(t *testing.T, sub string) []string {
	t.Helper()
	cod, d := mfaGerar(t, sub)
	esperar(t, "gerar", cod, 200)
	cod, _ = mfaConfirmar(t, sub, mfaListaID(d))
	esperar(t, "confirmar", cod, 200)
	return mfaCodigosDe(t, d)
}

// pendente gera uma lista e NAO a confirma.
func (c *cenarioRec) pendente(t *testing.T, sub string) []string {
	t.Helper()
	cod, d := mfaGerar(t, sub)
	esperar(t, "gerar", cod, 200)
	return mfaCodigosDe(t, d)
}

func (c *cenarioRec) recuperar(email, codigo, ip string) resp {
	body := `{"email":"` + email + `","codigo":"` + codigo + `"}`
	var rt rotulo
	res, _ := c.srv.atender(context.Background(), eventoPublico(body, nil, ip), &rt)
	return res
}

func ok(t *testing.T, nome string, r resp) {
	t.Helper()
	if r.StatusCode != 200 || r.Body != `{"ok":true}` {
		t.Errorf("%s: esperava 200 {\"ok\":true}, veio %d %s", nome, r.StatusCode, r.Body)
	}
}

func falhou(t *testing.T, nome string, r resp) {
	t.Helper()
	if r.StatusCode != http.StatusBadRequest || r.Body != mfaRecuperarFalha {
		t.Errorf("%s: esperava a falha unica 400 %s, veio %d %s", nome, mfaRecuperarFalha, r.StatusCode, r.Body)
	}
}

// ---------- o caminho feliz ----------

// R27 e R31 (M-T26): um codigo valido da lista ativa desliga o MFA DELA, devolve so {"ok":true} (sem token, sem sessao) e apaga a lista ativa.
func TestCodigoValidoDesligaOMFAEApagaALista(t *testing.T) {
	c := novoRec(t)
	codigos := c.lista(t, "ana")
	c.lista(t, "bia") // a lista de outra pessoa
	res := c.recuperar("ana@exemplo.com", codigos[3], ipA)
	ok(t, "recuperar", res)
	if len(c.id.desligados) != 1 || c.id.desligados[0] != "ana" {
		t.Fatalf("o MFA desligado devia ser so o da ana: %v", c.id.desligados)
	}
	if c.f.ler(t, pkPessoa, skAtivaMFA) != nil {
		t.Error("usar um codigo apaga a lista ativa inteira")
	}
	if c.f.ler(t, "USER#bia", skAtivaMFA) == nil {
		t.Error("a lista da Bia nao pode ser tocada")
	}
	if len(res.Headers) != 1 || res.Headers["content-type"] != "application/json" || len(res.Cookies) != 0 {
		t.Errorf("a resposta nao pode levar token, cookie nem cabecalho extra: %v %v", res.Headers, res.Cookies)
	}
	for _, proibido := range []string{"token", "Token", "senha", "sessao", "access", "id_token", "refresh"} {
		if strings.Contains(res.Body, proibido) {
			t.Errorf("a resposta leva %q: %s", proibido, res.Body)
		}
	}
	// o codigo aceita minusculas, espacos e sem hifens
	c2 := novoRec(t)
	outros := c2.lista(t, "ana")
	ok(t, "codigo digitado em minusculas e com espacos", c2.recuperar("ana@exemplo.com", strings.ToLower(strings.ReplaceAll(outros[0], "-", " ")), ipA))
}

// M-T21b: um codigo da lista PENDENTE nao recupera nada.
func TestCodigoDaListaPendenteNaoRecupera(t *testing.T) {
	c := novoRec(t)
	pend := c.pendente(t, "ana")
	falhou(t, "sem lista ativa", c.recuperar("ana@exemplo.com", pend[0], ipA))
	ativa := c.lista(t, "ana")
	nova := c.pendente(t, "ana") // outra pendente, com a ativa existindo
	falhou(t, "codigo da pendente com ativa existente", c.recuperar("ana@exemplo.com", nova[0], ipA))
	if len(c.id.desligados) != 0 {
		t.Errorf("nada devia ter sido desligado: %v", c.id.desligados)
	}
	ok(t, "o codigo da ativa continua valendo", c.recuperar("ana@exemplo.com", ativa[0], ipA))
}

// M-T22: um codigo vale uma vez; usar um invalida o resto.
func TestCodigoValeUmaVez(t *testing.T) {
	c := novoRec(t)
	codigos := c.lista(t, "ana")
	ok(t, "primeiro uso", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	falhou(t, "o mesmo codigo de novo", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	falhou(t, "outro codigo da mesma lista (a lista foi apagada)", c.recuperar("ana@exemplo.com", codigos[1], ipA))
	if len(c.id.desligados) != 1 {
		t.Errorf("so o primeiro uso desliga: %v", c.id.desligados)
	}
}

// M-T22b: o Cognito falha: a marca de usado e desfeita e o codigo continua valendo.
func TestFalhaDoCognitoNaoQueimaOCodigo(t *testing.T) {
	c := novoRec(t)
	codigos := c.lista(t, "ana")
	c.id.falharDesligar = errors.New("cognito fora do ar")
	falhou(t, "cognito falha", c.recuperar("ana@exemplo.com", codigos[2], ipA))
	ativa := c.f.ler(t, pkPessoa, skAtivaMFA)
	if ativa == nil {
		t.Fatal("a lista ativa nao pode ser apagada por uma falha do Cognito")
	}
	for i, x := range ativa["codigos"].([]any) {
		if x.(map[string]any)["usado"] != false {
			t.Errorf("o codigo %d ficou marcado como usado depois da falha do Cognito", i)
		}
	}
	c.id.falharDesligar = nil
	ok(t, "o mesmo codigo, depois", c.recuperar("ana@exemplo.com", codigos[2], "198.51.100.7"))
}

// M-T22a: dois usos simultaneos do mesmo codigo: so um passa. Outra chamada marca o codigo entre a leitura da lista e a escrita condicional.
func TestDoisUsosSimultaneosSoUmPassa(t *testing.T) {
	c := novoRec(t)
	codigos := c.lista(t, "ana")
	n, _ := domain.NormalizarCodigo(codigos[1])
	c.db.antesDeAtualizar = func() { // a "outra chamada" ganha a corrida: marca o mesmo codigo antes da escrita condicional desta
		c.f.marcarUsadoMFA(t, pkPessoa, mfaListaIDDe(c.f, t), n)
	}
	falhou(t, "perdeu a corrida", c.recuperar("ana@exemplo.com", codigos[1], ipA))
	if len(c.id.desligados) != 0 {
		t.Errorf("quem perde a corrida nao desliga nada: %v", c.id.desligados)
	}
}

// ---------- uma resposta so ----------

// M-T23: e-mail sem conta, pessoa sem MFA, conta desabilitada, codigo errado, codigo usado e formato invalido respondem IGUAL.
func TestTodasAsFalhasRespondemIgual(t *testing.T) {
	c := novoRec(t)
	codigos := c.lista(t, "ana")
	errado := "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ"
	base := c.recuperar("ana@exemplo.com", errado, ipA)
	falhou(t, "codigo errado", base)
	casos := map[string]resp{
		"e-mail sem conta":      c.recuperar("ninguem@exemplo.com", codigos[0], ipA),
		"pessoa sem MFA":        c.recuperar("sem@exemplo.com", codigos[0], ipA),
		"conta desabilitada":    c.recuperar("off@exemplo.com", codigos[0], ipA),
		"formato invalido":      c.recuperar("ana@exemplo.com", "curto", ipA),
		"formato com simbolo":   c.recuperar("ana@exemplo.com", "K7QM-4TXD-9WPB-2HRC-6NV!", ipA),
		"e-mail so com espacos": c.recuperar("   ", codigos[0], ipA),
	}
	for nome, r := range casos {
		if r.StatusCode != base.StatusCode || r.Body != base.Body || r.Headers["content-type"] != base.Headers["content-type"] {
			t.Errorf("%s respondeu %d %s, esperado igual a codigo errado (%d %s)", nome, r.StatusCode, r.Body, base.StatusCode, base.Body)
		}
	}
	// codigo ja usado: usa um, devolve a marca por baixo dos panos (lista sem apagar) e tenta de novo
	c.f.marcarUsadoMFA(t, pkPessoa, mfaListaIDDe(c.f, t), mustNorm(t, codigos[5]))
	usado := c.recuperar("ana@exemplo.com", codigos[5], ipA)
	if usado.StatusCode != base.StatusCode || usado.Body != base.Body {
		t.Errorf("codigo ja usado respondeu %d %s", usado.StatusCode, usado.Body)
	}
	if len(c.id.desligados) != 0 {
		t.Errorf("nenhuma dessas falhas pode desligar o MFA: %v", c.id.desligados)
	}
}

// Falha do Cognito ou do DynamoDB tambem e a mesma resposta, e nao grava nada.
func TestFalhaInternaRespondeAFalhaUnicaENaoGrava(t *testing.T) {
	c := novoRec(t)
	codigos := c.lista(t, "ana")
	c.db.zerar()
	c.id.falharBusca = errors.New("cognito caiu")
	falhou(t, "cognito caiu", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	if c.db.leituras+c.db.escritas != 0 {
		t.Errorf("com o Cognito fora, o DynamoDB nao pode ser tocado: %d leituras, %d escritas", c.db.leituras, c.db.escritas)
	}
	c.id.falharBusca = nil
	c.db.falharLeitura2 = errors.New("dynamo caiu")
	falhou(t, "dynamo caiu", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	c.db.falharLeitura2 = nil
	if len(c.id.desligados) != 0 || c.f.ler(t, pkPessoa, skAtivaMFA) == nil {
		t.Error("a falha interna nao pode desligar o MFA nem apagar a lista")
	}
}

// ---------- a capacidade da tabela (5 RCU/WCU) ----------

// M-T24b: e-mail sem conta, formato invalido e par bloqueado NAO leem nem gravam nada; codigo errado de conta existente e 1 leitura e nenhuma escrita.
func TestRotaPublicaNaoGastaATabelaSemConta(t *testing.T) {
	c := novoRec(t)
	codigos := c.lista(t, "ana")
	c.db.zerar()
	falhou(t, "sem conta", c.recuperar("ninguem@exemplo.com", codigos[0], ipA))
	falhou(t, "formato invalido", c.recuperar("ana@exemplo.com", "curto", ipA))
	falhou(t, "pessoa sem MFA", c.recuperar("sem@exemplo.com", codigos[0], ipA))
	if c.db.leituras != 0 || c.db.escritas != 0 {
		t.Fatalf("sem conta, sem MFA e formato invalido nao tocam o DynamoDB: %d leituras, %d escritas", c.db.leituras, c.db.escritas)
	}
	falhou(t, "codigo errado", c.recuperar("ana@exemplo.com", "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", ipA))
	if c.db.leituras != 1 || c.db.escritas != 0 {
		t.Errorf("codigo errado de conta existente: esperava 1 leitura e nenhuma escrita, veio %d e %d", c.db.leituras, c.db.escritas)
	}
	// o sucesso grava so o necessario: marcar e apagar
	c.db.zerar()
	ok(t, "sucesso", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	if c.db.leituras != 1 || c.db.escritas != 2 {
		t.Errorf("o sucesso custa 1 leitura e 2 escritas (marcar e apagar), veio %d e %d", c.db.leituras, c.db.escritas)
	}
}

// M-T24: o par (pessoa, origem) e bloqueado depois de 10 falhas; bloqueado, nem o codigo certo e conferido; outra origem segue; passada 1 hora volta.
func TestParBloqueadoDepoisDeDezFalhas(t *testing.T) {
	c := novoRec(t)
	codigos := c.lista(t, "ana")
	for i := 0; i < 10; i++ {
		falhou(t, "falha", c.recuperar("ana@exemplo.com", "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", ipA))
	}
	c.db.zerar()
	falhou(t, "par bloqueado com o codigo certo", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	if c.db.leituras != 0 || len(c.id.desligados) != 0 {
		t.Errorf("bloqueado, o codigo nao e conferido: %d leituras, desligados %v", c.db.leituras, c.id.desligados)
	}
	// passada 1 hora o par volta (o relogio de teste e fixo: avanca e restaura)
	depois := relogioTeste.Agora().Add(time.Hour + time.Second)
	antes := relogioTeste.Definir(depois)
	defer relogioTeste.Definir(antes)
	ok(t, "passada 1 hora o par volta", c.recuperar("ana@exemplo.com", codigos[0], ipA))
}

func TestOutraOrigemNaoEAfetadaPeloBloqueio(t *testing.T) {
	c := novoRec(t)
	codigos := c.lista(t, "ana")
	for i := 0; i < 10; i++ {
		c.recuperar("ana@exemplo.com", "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", ipA)
	}
	falhou(t, "par bloqueado", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	ok(t, "outra origem, mesma pessoa", c.recuperar("ana@exemplo.com", codigos[0], "198.51.100.7"))
}

// O sucesso zera o contador do par: 9 falhas, sucesso, mais 9 falhas e o sucesso seguinte ainda passa.
func TestSucessoZeraAsFalhas(t *testing.T) {
	c := novoRec(t)
	for rodada := 0; rodada < 2; rodada++ {
		codigos := c.lista(t, "ana")
		for i := 0; i < 9; i++ {
			falhou(t, "falha", c.recuperar("ana@exemplo.com", "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", ipA))
		}
		ok(t, "sucesso na rodada", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	}
}

// M-T24a: variacoes de caixa e de espaco nas pontas do e-mail buscam a MESMA pessoa (o e-mail chega ao Cognito em minusculas e aparado).
func TestEmailNormalizadoParaMinusculas(t *testing.T) {
	c := novoRec(t)
	for _, v := range []string{"ana@exemplo.com", "ANA@EXEMPLO.COM", " Ana@Exemplo.com ", "\tana@exemplo.com\n"} {
		c.recuperar(strings.ReplaceAll(strings.ReplaceAll(v, "\t", " "), "\n", " "), "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", ipA)
	}
	for _, b := range c.id.buscas {
		if b != "ana@exemplo.com" {
			t.Errorf("o Cognito recebeu %q, esperado ana@exemplo.com", b)
		}
	}
	if len(c.id.buscas) != 4 {
		t.Errorf("esperava 4 buscas, vieram %d", len(c.id.buscas))
	}
}

// M-T24c: 30 chamadas por minuto por origem; a 31a nem chega ao Cognito; outra origem segue; no minuto seguinte volta.
func TestOrigemTrintaPorMinutoNaRota(t *testing.T) {
	c := novoRec(t)
	for i := 0; i < 30; i++ {
		c.recuperar("ninguem@exemplo.com", "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", ipA)
	}
	if len(c.id.buscas) != 30 {
		t.Fatalf("as 30 primeiras chamadas chegam ao Cognito, chegaram %d", len(c.id.buscas))
	}
	falhou(t, "31a", c.recuperar("ninguem@exemplo.com", "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", ipA))
	if len(c.id.buscas) != 30 {
		t.Error("a 31a chamada da origem nao pode chegar ao Cognito")
	}
	c.recuperar("ninguem@exemplo.com", "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", "198.51.100.7")
	if len(c.id.buscas) != 31 {
		t.Error("outra origem nao e afetada")
	}
	antes := relogioTeste.Definir(relogioTeste.Agora().Add(time.Minute))
	defer relogioTeste.Definir(antes)
	c.recuperar("ninguem@exemplo.com", "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", ipA)
	if len(c.id.buscas) != 32 {
		t.Error("no minuto seguinte a origem volta")
	}
}

// M-T25: o codigo de uma pessoa nunca desliga o MFA de outra, nem com o e-mail da outra.
func TestCodigoDeUmaPessoaNaoDesligaOMFADeOutra(t *testing.T) {
	c := novoRec(t)
	ana := c.lista(t, "ana")
	c.lista(t, "bia")
	falhou(t, "codigo da Ana com o e-mail da Bia", c.recuperar("bia@exemplo.com", ana[0], ipA))
	for _, sub := range c.id.desligados {
		if sub == "bia" {
			t.Fatal("o MFA da Bia foi desligado com um codigo da Ana")
		}
	}
	ok(t, "a Ana com o proprio e-mail", c.recuperar("ana@exemplo.com", ana[0], ipA))
	if len(c.id.desligados) != 1 || c.id.desligados[0] != "ana" {
		t.Errorf("so o MFA da Ana devia ser desligado: %v", c.id.desligados)
	}
}

// ---------- o que nunca vai ao log ----------

// R34 e M-T28: nenhum log (nem das falhas internas) leva e-mail, codigo, sub nem corpo.
func TestRecuperacaoNaoLogaDadosDaPessoa(t *testing.T) {
	buf := capturarLog(t)
	c := novoRec(t)
	c.id.pessoas["segredo@exemplo.com"] = domain.PessoaMFA{Sub: "sub-secreto-123", Habilitada: true, TemMFA: true}
	codigos := c.lista(t, "sub-secreto-123")
	cod := codigos[0]

	c.id.falharDesligar = errors.New("falha com sub-secreto-123 e segredo@exemplo.com")
	c.recuperar("segredo@exemplo.com", cod, ipA) // falha do Cognito ao desligar
	c.id.falharDesligar = nil
	c.id.falharBusca = errors.New("falha com segredo@exemplo.com")
	c.recuperar("segredo@exemplo.com", cod, ipA) // falha do Cognito ao buscar
	c.id.falharBusca = nil
	c.db.falharLeitura2 = errors.New("falha com sub-secreto-123")
	c.recuperar("segredo@exemplo.com", cod, ipA) // falha do DynamoDB
	c.db.falharLeitura2 = nil
	c.recuperar("segredo@exemplo.com", "ZZZZ-ZZZZ-ZZZZ-ZZZZ-ZZZZ", ipA)
	r := eventoPublico(`{"email":"segredo@exemplo.com","codigo":"`+cod+`"}`, nil, ipA)
	r.RequestContext.RequestID = "rid-9"
	_, _ = c.srv.handler(context.Background(), r)

	log := buf.String()
	n, _ := domain.NormalizarCodigo(cod)
	for _, proibido := range []string{"segredo@exemplo.com", "exemplo.com", "sub-secreto-123", cod, n, ipA, "ZZZZ"} {
		if strings.Contains(log, proibido) {
			t.Errorf("o log leva %q:\n%s", proibido, log)
		}
	}
	if !strings.Contains(log, "acao=mfa.recuperar rota=mfa") {
		t.Errorf("a linha da requisicao deveria ser acao=mfa.recuperar rota=mfa:\n%s", log)
	}
}

// Sem identidade ligada (antes de o main religar ao Cognito), toda recuperacao so responde a falha e nao desliga nada.
func TestSemIdentidadeTodaRecuperacaoFalha(t *testing.T) {
	f := cenario(t)
	srv := novoServidor(f, "", relogioTeste, notifTeste, "")
	var rt rotulo
	res, _ := srv.atender(context.Background(), eventoPublico(`{"email":"ana@exemplo.com","codigo":"K7QM-4TXD-9WPB-2HRC-6NVF"}`, nil, ipA), &rt)
	falhou(t, "sem identidade", res)
}

// ---------- auxiliares ----------

func mustNorm(t *testing.T, c string) string {
	t.Helper()
	n, ok := domain.NormalizarCodigo(c)
	if !ok {
		t.Fatalf("codigo fora do formato: %q", c)
	}
	return n
}

// mfaListaIDDe devolve o listaId da lista ativa da pessoa do cenario (ana).
func mfaListaIDDe(f *bancoFake, t *testing.T) string {
	t.Helper()
	d := f.ler(t, pkPessoa, skAtivaMFA)
	if d == nil {
		t.Fatal("nao ha lista ativa")
	}
	return d["listaId"].(string)
}

// marcarUsadoMFA marca direto no banco o codigo (ja normalizado) como usado, como se outra chamada tivesse ganhado a corrida.
func (f *bancoFake) marcarUsadoMFA(t *testing.T, pk, listaID, codigoNormalizado string) {
	t.Helper()
	it := f.itens[chave(pk, skAtivaMFA)]
	h := domain.HashCodigo(listaID, codigoNormalizado)
	for _, x := range it["codigos"].(*types.AttributeValueMemberL).Value {
		m := x.(*types.AttributeValueMemberM).Value
		if m["hash"].(*types.AttributeValueMemberS).Value == h {
			m["usado"] = &types.AttributeValueMemberBOOL{Value: true}
			return
		}
	}
	t.Fatal("o codigo nao esta na lista ativa")
}
