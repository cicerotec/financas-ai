package main

import (
	"context"
	"encoding/json"
	mrand "math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"financas/auth"
	"financas/domain"
	"financas/service"
)

// Spec mfa-opcional, AD-2 (a lista nasce pendente e so vale depois de confirmada): M-T21, M-T21a, M-T21c a f, R26, R36 e M-T32.

const (
	pkPessoa      = "USER#ana"
	skPendenteMFA = "MFACODIGOS#PENDENTE"
	skAtivaMFA    = "MFACODIGOS#ATIVA"
	rotaGerar     = "me/mfa/codigos"
	rotaConfirmar = "me/mfa/codigos/confirmar"
)

func mfaGerar(t *testing.T, sub string) (int, doc) {
	t.Helper()
	return chamar(t, sub, "POST", rotaGerar, nil, "")
}

func mfaConfirmar(t *testing.T, sub, listaID string) (int, doc) {
	t.Helper()
	return chamar(t, sub, "POST", rotaConfirmar, nil, `{"listaId":"`+listaID+`"}`)
}

func mfaListaID(d doc) string { s, _ := d["listaId"].(string); return s }

func mfaCodigosDe(t *testing.T, d doc) []string {
	t.Helper()
	var out []string
	for _, x := range d["codigos"].([]any) {
		out = append(out, x.(string))
	}
	return out
}

func mfaItemTexto(f *bancoFake, pk, sk string) string {
	it, ok := f.itens[chave(pk, sk)]
	if !ok {
		return ""
	}
	b, _ := json.Marshal(it)
	return string(b)
}

// M-T21: gerar devolve a lista PENDENTE uma vez (listaId e 10 codigos de 20 simbolos) e grava so hashes; a ativa nao nasce.
func TestGerarDevolveAListaPendenteEGravaSoHashes(t *testing.T) {
	f := cenario(t)
	c, d := mfaGerar(t, "ana")
	esperar(t, "gerar", c, 200)
	cods := mfaCodigosDe(t, d)
	if len(cods) != domain.QuantidadeCodigos || !domain.ListaIDValido(mfaListaID(d)) {
		t.Fatalf("esperava %d codigos e um listaId valido: %v", domain.QuantidadeCodigos, d)
	}
	vistos := map[string]bool{}
	for _, c := range cods {
		n, ok := domain.NormalizarCodigo(c)
		if !ok || len(n) != domain.TamanhoCodigo || strings.Count(c, "-") != 4 || vistos[n] {
			t.Errorf("codigo %q fora do formato (20 simbolos em 5 grupos) ou repetido", c)
		}
		vistos[n] = true
	}
	if f.ler(t, pkPessoa, skAtivaMFA) != nil {
		t.Error("gerar nao pode criar a lista ativa")
	}
	pend := mfaItemTexto(f, pkPessoa, skPendenteMFA)
	if pend == "" || !strings.Contains(pend, mfaListaID(d)) {
		t.Fatalf("a pendente nao foi gravada com o listaId: %s", pend)
	}
	for n := range vistos { // R26: nunca o codigo em texto, nem sem hifens
		if strings.Contains(pend, n) || strings.Contains(pend, domain.FormatarCodigo(n)) {
			t.Fatalf("o codigo %s foi gravado em texto: %s", n, pend)
		}
	}
	for n := range vistos { // so o hash, com o listaId como chave
		if !strings.Contains(pend, domain.HashCodigo(mfaListaID(d), n)) {
			t.Errorf("falta o hash do codigo %s", n)
		}
	}
	pendItem := f.ler(t, pkPessoa, skPendenteMFA)
	if exp, _ := pendItem["expiraEm"].(float64); int64(exp) != relogioTeste.Agora().Add(domain.ValidadePendente).Unix() {
		t.Errorf("expiraEm = %v, esperado agora + 1 hora", pendItem["expiraEm"])
	}
	// a rota nao grava mais nada
	for k := range f.itens {
		if strings.HasPrefix(k, pkPessoa+"|") && !strings.HasSuffix(k, skPendenteMFA) {
			t.Errorf("gerar gravou um item a mais: %s", k)
		}
	}
}

// M-T21f: gerar aceita corpo vazio; confirmar exige JSON (400 sem corpo); as duas rotas sao reconhecidas fora do espaco.
func TestGerarAceitaCorpoVazioEConfirmarExigeJSON(t *testing.T) {
	cenario(t)
	c, _ := chamar(t, "ana", "POST", rotaGerar, nil, "")
	esperar(t, "gerar sem corpo", c, 200)
	c, _ = chamar(t, "ana", "POST", rotaGerar, nil, `{"qualquer":"coisa"}`)
	esperar(t, "gerar com corpo (ignorado)", c, 200)
	c, _ = chamar(t, "ana", "POST", rotaConfirmar, nil, "")
	esperar(t, "confirmar sem corpo", c, 400)
	c, _ = chamar(t, "ana", "POST", rotaConfirmar, nil, "nao e json")
	esperar(t, "confirmar com lixo", c, 400)
	c, _ = chamar(t, "ana", "POST", rotaConfirmar, nil, `{}`)
	esperar(t, "confirmar sem listaId", c, 400)
	c, _ = chamar(t, "ana", "POST", rotaConfirmar, nil, `{"listaId":"../x"}`)
	esperar(t, "confirmar com listaId de formato invalido", c, 400)
	for _, m := range []string{"GET", "PUT", "DELETE"} {
		if a, ok := auth.RotaSemEspaco(m, []string{"me", "mfa", "codigos"}); ok {
			t.Errorf("%s /me/mfa/codigos nao e rota (acao %q)", m, a)
		}
	}
	if a, ok := auth.RotaSemEspaco("POST", []string{"me", "mfa", "codigos"}); !ok || a != auth.AcaoGerarMFA {
		t.Errorf("POST /me/mfa/codigos = %q %v", a, ok)
	}
	if a, ok := auth.RotaSemEspaco("POST", []string{"me", "mfa", "codigos", "confirmar"}); !ok || a != auth.AcaoConfirmarMFA {
		t.Errorf("POST /me/mfa/codigos/confirmar = %q %v", a, ok)
	}
}

// M-T21c: confirmar com o listaId certo promove a pendente e apaga a ativa anterior; o codigo antigo deixa de valer e o novo vale.
func TestConfirmarPromoveAPendenteEApagaAAtivaAnterior(t *testing.T) {
	f := cenario(t)
	_, d1 := mfaGerar(t, "ana")
	c, _ := mfaConfirmar(t, "ana", mfaListaID(d1))
	esperar(t, "confirmar 1", c, 200)
	if f.ler(t, pkPessoa, skPendenteMFA) != nil {
		t.Error("a pendente deveria sair depois de confirmada")
	}
	at1 := f.ler(t, pkPessoa, skAtivaMFA)
	if at1 == nil || at1["listaId"] != mfaListaID(d1) {
		t.Fatalf("a ativa deveria ser a lista 1: %v", at1)
	}
	if _, tem := at1["expiraEm"]; tem {
		t.Error("a ativa nao tem expiraEm")
	}
	_, d2 := mfaGerar(t, "ana")
	c, _ = mfaConfirmar(t, "ana", mfaListaID(d2))
	esperar(t, "confirmar 2", c, 200)
	at2 := f.ler(t, pkPessoa, skAtivaMFA)
	if at2["listaId"] != mfaListaID(d2) || mfaListaID(d1) == mfaListaID(d2) {
		t.Fatalf("a ativa deveria ser a lista 2: %v", at2)
	}
	txt := mfaItemTexto(f, pkPessoa, skAtivaMFA)
	n1, _ := domain.NormalizarCodigo(mfaCodigosDe(t, d1)[0])
	n2, _ := domain.NormalizarCodigo(mfaCodigosDe(t, d2)[0])
	if strings.Contains(txt, domain.HashCodigo(mfaListaID(d1), n1)) {
		t.Error("o hash de um codigo da lista antiga ainda esta na ativa")
	}
	if !strings.Contains(txt, domain.HashCodigo(mfaListaID(d2), n2)) {
		t.Error("o hash de um codigo da lista nova nao esta na ativa")
	}
}

// M-T21a: gerar com uma lista ativa existente NAO a invalida nem a toca.
func TestGerarNaoTocaNaAtiva(t *testing.T) {
	f := cenario(t)
	_, d1 := mfaGerar(t, "ana")
	mfaConfirmar(t, "ana", mfaListaID(d1))
	antes := mfaItemTexto(f, pkPessoa, skAtivaMFA)
	for i := 0; i < 3; i++ {
		c, _ := mfaGerar(t, "ana")
		esperar(t, "gerar de novo", c, 200)
	}
	if depois := mfaItemTexto(f, pkPessoa, skAtivaMFA); depois != antes || depois == "" {
		t.Errorf("gerar alterou a lista ativa:\nantes  %s\ndepois %s", antes, depois)
	}
}

// M-T21d: confirmar e idempotente; pendente substituida ou vencida da 409 e nao muda a ativa.
func TestConfirmarIdempotenteE409(t *testing.T) {
	f := cenario(t)
	_, d1 := mfaGerar(t, "ana")
	c, _ := mfaConfirmar(t, "ana", mfaListaID(d1))
	esperar(t, "primeiro confirmar", c, 200)
	c, _ = mfaConfirmar(t, "ana", mfaListaID(d1))
	esperar(t, "confirmar de novo o mesmo listaId (idempotente)", c, 200)

	// substituida: outra geracao (outra aba) troca a pendente; o confirmar da primeira da 409 e a ativa fica
	_, d2 := mfaGerar(t, "ana")
	_, d3 := mfaGerar(t, "ana")
	ativa := mfaItemTexto(f, pkPessoa, skAtivaMFA)
	c, corpo := mfaConfirmar(t, "ana", mfaListaID(d2))
	esperar(t, "confirmar pendente substituida", c, 409)
	if corpo["erro"] != "lista_substituida" {
		t.Errorf("corpo do 409 = %v", corpo)
	}
	if mfaItemTexto(f, pkPessoa, skAtivaMFA) != ativa {
		t.Error("o 409 mudou a ativa")
	}
	c, _ = mfaConfirmar(t, "ana", "0123456789abcdef0123456789abcdef") // listaId que nunca existiu
	esperar(t, "confirmar listaId desconhecido", c, 409)

	// vencida: passada 1 hora da geracao, a pendente nao pode mais ser confirmada
	depois := relogioTeste.Agora().Add(domain.ValidadePendente + time.Second)
	antes := relogioTeste.Definir(depois)
	defer relogioTeste.Definir(antes)
	c, corpo = mfaConfirmar(t, "ana", mfaListaID(d3))
	esperar(t, "confirmar pendente vencida", c, 409)
	if corpo["erro"] != "lista_substituida" {
		t.Errorf("corpo do 409 (vencida) = %v", corpo)
	}
	if mfaItemTexto(f, pkPessoa, skAtivaMFA) != ativa {
		t.Error("confirmar vencida mudou a ativa")
	}
	// ate o ultimo segundo ainda vale
	relogioTeste.Definir(antes)
	_, d4 := mfaGerar(t, "ana")
	relogioTeste.Definir(antes.Add(domain.ValidadePendente - time.Second))
	c, _ = mfaConfirmar(t, "ana", mfaListaID(d4))
	esperar(t, "confirmar antes de vencer", c, 200)
}

// M-T21e: sem token e 401; com o token de outra pessoa nada muda na lista desta (o sub vem so do token).
func TestConfirmarSoMexeNaListaDeQuemChama(t *testing.T) {
	f := cenario(t)
	_, da := mfaGerar(t, "ana")
	// outra pessoa tenta confirmar o listaId da Ana: 409 e a pendente da Ana continua
	c, _ := mfaConfirmar(t, "bia", mfaListaID(da))
	esperar(t, "bia confirma a lista da ana", c, 409)
	if f.ler(t, pkPessoa, skPendenteMFA) == nil || f.ler(t, pkPessoa, skAtivaMFA) != nil {
		t.Error("a lista da Ana mudou por causa de outra pessoa")
	}
	if f.ler(t, "USER#bia", skAtivaMFA) != nil {
		t.Error("a Bia ganhou uma ativa que nao era dela")
	}
	// o sub do corpo e ignorado
	c, _ = chamar(t, "bia", "POST", rotaConfirmar, nil, `{"listaId":"`+mfaListaID(da)+`","sub":"ana"}`)
	esperar(t, "sub no corpo", c, 409)
	// cada pessoa tem a sua
	_, db := mfaGerar(t, "bia")
	if f.ler(t, "USER#bia", skPendenteMFA) == nil || mfaListaID(da) == mfaListaID(db) {
		t.Error("a Bia deveria ter a propria pendente")
	}
	// sem token: 401 (as duas rotas)
	s := novoServidor(f, "", relogioTeste, notifTeste, "")
	for _, caminho := range []string{"/me/mfa/codigos", "/me/mfa/codigos/confirmar"} {
		var r req
		r.RawPath = caminho
		r.Body = `{"listaId":"0123456789abcdef"}`
		r.RequestContext.HTTP.Method = "POST"
		var rt rotulo
		if res, _ := s.atender(context.Background(), r, &rt); res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s sem token respondeu %d, esperado 401", caminho, res.StatusCode)
		}
	}
}

// M-T32: sair da conta (logout) tira o aparelho mas NUNCA apaga a lista de codigos: ela e a saida de quem perdeu o autenticador.
func TestLogoutNaoApagaAListaAtiva(t *testing.T) {
	f := cenario(t)
	_, d := mfaGerar(t, "ana")
	mfaConfirmar(t, "ana", mfaListaID(d))
	mfaGerar(t, "ana") // e uma pendente em aberto
	ativa, pend := mfaItemTexto(f, pkPessoa, skAtivaMFA), mfaItemTexto(f, pkPessoa, skPendenteMFA)
	c, _ := chamar(t, "ana", "POST", "push/sair", nil, `{"token":"token-de-aparelho-12345678901234567890"}`)
	esperar(t, "push/sair", c, 200)
	if mfaItemTexto(f, pkPessoa, skAtivaMFA) != ativa || ativa == "" {
		t.Error("o logout apagou ou mudou a lista ativa")
	}
	if mfaItemTexto(f, pkPessoa, skPendenteMFA) != pend {
		t.Error("o logout mexeu na pendente")
	}
}

// R36: os itens de MFA nao aparecem na listagem de espacos da pessoa (a consulta filtra begins_with(SK, "SPACE#")).
func TestItensDeMFANaoApareceNosEspacos(t *testing.T) {
	cenario(t)
	_, d := mfaGerar(t, "own")
	mfaConfirmar(t, "own", mfaListaID(d))
	mfaGerar(t, "own")
	c, me := chamar(t, "own", "GET", "me", nil, "")
	esperar(t, "GET /me", c, 200)
	b, _ := json.Marshal(me)
	if strings.Contains(string(b), "MFACODIGOS") {
		t.Errorf("a listagem de espacos mostrou um item de MFA: %s", b)
	}
}

// Falha do banco: a resposta e 500 e nada vaza (nem o codigo, nem o sub).
func TestGerarComFalhaNoBanco(t *testing.T) {
	f := cenario(t)
	f.falharPut = http.ErrAbortHandler
	r, _ := srvTeste.processar(context.Background(), &user{Sub: "ana"}, "POST", strings.Split(rotaGerar, "/"), nil, "")
	if r.StatusCode != 500 {
		t.Errorf("status %d, esperado 500", r.StatusCode)
	}
	if strings.Contains(r.Body, "ana") || strings.Contains(r.Body, "abort") {
		t.Errorf("o corpo do erro vaza detalhe: %s", r.Body)
	}
}

// O service nao importa a AWS (arquitetura_test cobre) e a fonte aleatoria pode ser trocada: com a mesma fonte, a mesma lista.
func TestGerarComFonteFixaEDeterministico(t *testing.T) {
	f := cenario(t)
	_ = f
	mk := func() doc {
		svc := service.NewCodigosDeRecuperacao(&repoCodigosMudo{}, relogioTeste)
		svc.UsarFonte(mrand.New(mrand.NewSource(7)))
		r, err := svc.Gerar(context.Background(), "ana")
		if err != nil {
			t.Fatal(err)
		}
		return doc(r)
	}
	a, b := mk(), mk()
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Errorf("a mesma fonte deu listas diferentes:\n%s\n%s", ja, jb)
	}
}

type repoCodigosMudo struct{}

func (repoCodigosMudo) SalvarPendente(context.Context, string, domain.ListaCodigos) error { return nil }
func (repoCodigosMudo) Confirmar(context.Context, string, string, time.Time) error        { return nil }

// uma fonte degenerada (so repete) nao prende a rota num laco: da 500 e nao grava nada.
type fonteQueRepete struct{}

func (fonteQueRepete) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 7
	}
	return len(p), nil
}

func TestGerarComFonteQueRepeteNaoPrende(t *testing.T) {
	svc := service.NewCodigosDeRecuperacao(&repoCodigosMudo{}, relogioTeste)
	svc.UsarFonte(fonteQueRepete{})
	feito := make(chan error, 1)
	go func() { _, err := svc.Gerar(context.Background(), "ana"); feito <- err }()
	select {
	case err := <-feito:
		if err == nil {
			t.Error("uma fonte que so repete deveria dar erro")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Gerar prendeu num laco com uma fonte que repete")
	}
}
