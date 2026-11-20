package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/adapter/notificacao"
	"financas/domain"
	"financas/repository"
	"financas/service"
)

// M-T31 (spec mfa-opcional, R35): a pessoa e avisada pelo canal que JA tem ligado quando o MFA muda. Duas frases fixas, so o fato.

// vinculo grava o vinculo da pessoa com um espaco, com o Telegram (texto) e/ou os aparelhos (conjunto de textos).
func vinculo(f *bancoFake, sub, sid, chat string, tokens ...string) {
	it := map[string]types.AttributeValue{
		"PK":   &types.AttributeValueMemberS{Value: "USER#" + sub},
		"SK":   &types.AttributeValueMemberS{Value: "SPACE#" + sid},
		"role": &types.AttributeValueMemberS{Value: "member"},
	}
	if chat != "" {
		it["telegramChatId"] = &types.AttributeValueMemberS{Value: chat}
	}
	if len(tokens) > 0 {
		it["pushTokens"] = &types.AttributeValueMemberSS{Value: tokens}
	}
	f.itens[chave("USER#"+sub, "SPACE#"+sid)] = it
}

func zerarCanais() {
	canalTeste.enviados, canalTeste.falhaCom = nil, nil
	canalPushTeste.enviados, canalPushTeste.falhaCom = nil, nil
}

func todosOsEnviados() []envio {
	return append(append([]envio(nil), canalTeste.enviados...), canalPushTeste.enviados...)
}

const tokenPush = "token-de-aparelho-12345678901234567890"

// A recuperacao com sucesso avisa pelo canal ligado, com a frase fixa e SEM nenhum dado.
func TestRecuperacaoComSucessoAvisaPeloCanalLigado(t *testing.T) {
	zerarCanais()
	c := novoRec(t)
	vinculo(c.f, "ana", "s1", "111222333", tokenPush)
	codigos := c.lista(t, "ana")
	zerarCanais() // o confirmar acima ja avisou da lista nova: aqui so conta o aviso da recuperacao
	ok(t, "recuperar", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	env := todosOsEnviados()
	if len(env) != 2 {
		t.Fatalf("esperava 1 aviso no Telegram e 1 no aparelho, vieram %d: %v", len(env), env)
	}
	for _, e := range env {
		if !strings.Contains(e.texto, domain.MensagemMFADesligado) {
			t.Errorf("o aviso devia ter a frase fixa %q, veio %q", domain.MensagemMFADesligado, e.texto)
		}
		for _, proibido := range []string{"ana@exemplo.com", "exemplo.com", codigos[0], codigos[1], "ana", ipA} {
			if strings.Contains(e.texto, proibido) {
				t.Errorf("o aviso leva %q: %q", proibido, e.texto)
			}
		}
	}
	if len(canalTeste.enviados) != 1 || canalTeste.enviados[0].chat != "111222333" || len(canalPushTeste.enviados) != 1 || canalPushTeste.enviados[0].chat != tokenPush {
		t.Errorf("cada canal recebe so o proprio destino: telegram %v, aparelho %v", canalTeste.enviados, canalPushTeste.enviados)
	}
}

// O 200 do confirmar avisa "uma nova lista foi gerada" e NUNCA diz que o MFA foi ligado.
func TestConfirmarAvisaListaNovaENuncaDizQueLigou(t *testing.T) {
	zerarCanais()
	c := novoRec(t)
	vinculo(c.f, "ana", "s1", "111222333")
	_, d := mfaGerar(t, "ana")
	if len(todosOsEnviados()) != 0 {
		t.Error("gerar (a lista ainda pendente) nao avisa")
	}
	mfaConfirmar(t, "ana", mfaListaID(d))
	env := todosOsEnviados()
	if len(env) != 1 || !strings.Contains(env[0].texto, domain.MensagemListaGerada) {
		t.Fatalf("o confirmar devia avisar a lista nova: %v", env)
	}
	for _, proibido := range []string{"ligado", "ligou", "ativado", "ativou"} {
		if strings.Contains(strings.ToLower(env[0].texto), proibido) {
			t.Errorf("o aviso do confirmar nao pode dizer que o MFA foi ligado: %q", env[0].texto)
		}
	}
	if !strings.Contains(domain.MensagemMFADesligado, "desligado com um código de recuperação") || !strings.Contains(domain.MensagemListaGerada, "nova lista de códigos de recuperação foi gerada") {
		t.Error("as frases fixas do spec mudaram")
	}
	// confirmar com 409 (lista substituida) nao avisa
	zerarCanais()
	mfaConfirmar(t, "ana", "0123456789abcdef0123456789abcdef")
	if len(todosOsEnviados()) != 0 {
		t.Error("um confirmar que falhou nao avisa")
	}
}

// Destinos iguais em varios espacos recebem UM aviso so.
func TestDestinosIguaisEmVariosEspacosRecebemUmAviso(t *testing.T) {
	zerarCanais()
	c := novoRec(t)
	vinculo(c.f, "ana", "s1", "111222333", tokenPush)
	vinculo(c.f, "ana", "s2", "111222333", tokenPush)
	vinculo(c.f, "ana", "s3", "444555666") // outro chat
	_, d := mfaGerar(t, "ana")
	mfaConfirmar(t, "ana", mfaListaID(d))
	if len(canalTeste.enviados) != 2 || len(canalPushTeste.enviados) != 1 {
		t.Errorf("esperava 2 chats distintos e 1 aparelho, veio telegram %v, aparelho %v", canalTeste.enviados, canalPushTeste.enviados)
	}
}

// Sem canal ligado nada e enviado e a operacao NAO falha.
func TestSemCanalLigadoNadaEEnviadoEAOperacaoNaoFalha(t *testing.T) {
	zerarCanais()
	c := novoRec(t)
	vinculo(c.f, "ana", "s1", "")
	codigos := c.lista(t, "ana")
	ok(t, "recuperar sem canal", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	if len(todosOsEnviados()) != 0 {
		t.Errorf("sem canal ligado nada deve ser enviado: %v", todosOsEnviados())
	}
	// pessoa sem nenhum espaco: igual
	c2 := novoRec(t)
	codigos2 := c2.lista(t, "bia")
	ok(t, "recuperar sem espaco", c2.recuperar("bia@exemplo.com", codigos2[0], ipA))
}

// A falha do canal nao muda a resposta nem o que ja foi feito.
func TestFalhaDoCanalNaoMudaARespostaDaRecuperacao(t *testing.T) {
	zerarCanais()
	c := novoRec(t)
	vinculo(c.f, "ana", "s1", "111222333", tokenPush)
	codigos := c.lista(t, "ana")
	canalTeste.falhaCom, canalPushTeste.falhaCom = errors.New("canal fora do ar com 111222333"), domain.ErrDestinoInvalido
	defer zerarCanais()
	buf := capturarLog(t)
	ok(t, "recuperar com os canais fora do ar", c.recuperar("ana@exemplo.com", codigos[0], ipA))
	if len(c.id.desligados) != 1 || c.f.ler(t, pkPessoa, skAtivaMFA) != nil {
		t.Error("a falha do canal nao pode desfazer a recuperacao")
	}
	if log := buf.String(); strings.Contains(log, "111222333") || strings.Contains(log, tokenPush) || strings.Contains(log, domain.MensagemMFADesligado) {
		t.Errorf("o log leva destino ou texto:\n%s", log)
	}
}

// A falha ao confirmar nao e escondida pelo aviso: o confirmar responde 200 mesmo se o canal falhar.
func TestFalhaDoCanalNaoMudaOConfirmar(t *testing.T) {
	zerarCanais()
	c := novoRec(t)
	vinculo(c.f, "ana", "s1", "111222333")
	canalTeste.falhaCom = errors.New("fora do ar")
	defer zerarCanais()
	_, d := mfaGerar(t, "ana")
	cod, _ := mfaConfirmar(t, "ana", mfaListaID(d))
	esperar(t, "confirmar com o canal fora do ar", cod, 200)
}

// O aviso nunca atrasa a resposta alem do teto, mesmo com um canal que trava.
func TestAvisoNaoPrendeAResposta(t *testing.T) {
	trava := make(chan struct{})
	defer close(trava)
	lento := &canalQueTrava{trava: trava}
	desp, _ := service.NovoDespachante(lento)
	f := novoBancoFake()
	vinculo(f, "ana", "s1", "111222333")
	seg := service.NewAvisosDeSeguranca(repoEspacosDe(f), repoContatosDe(f), desp, "")
	t0 := time.Now()
	seg.Avisar(context.Background(), "ana", domain.MensagemMFADesligado)
	if d := time.Since(t0); d > 3*time.Second {
		t.Errorf("o aviso prendeu a resposta por %v", d)
	}
}

type canalQueTrava struct{ trava chan struct{} }

func (c *canalQueTrava) Nome() string { return "telegram" }
func (c *canalQueTrava) Enviar(ctx context.Context, _ string, _ domain.ResumoAvisos) error {
	<-c.trava
	return nil
}

// O texto do Telegram do aviso de seguranca e o do push: so a frase (com a marca do dev).
func TestTextoDoAvisoDeSeguranca(t *testing.T) {
	r := domain.ResumoAvisos{Marca: "[DEV] ", Mensagem: domain.MensagemMFADesligado}
	partes := notifTelegramTexto(r)
	if len(partes) != 1 || partes[0] != "🔐 [DEV] "+domain.MensagemMFADesligado {
		t.Errorf("texto do Telegram = %q", partes)
	}
}

func repoEspacosDe(f *bancoFake) *repository.Espacos  { return repository.NewEspacos(f, "") }
func repoContatosDe(f *bancoFake) *repository.Pessoas { return repository.NewPessoas(f, "") }
func notifTelegramTexto(r domain.ResumoAvisos) []string {
	return notificacao.PartesDoResumo(r, notificacao.LimiteTelegram)
}
