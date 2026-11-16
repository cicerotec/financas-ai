package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"financas/auth"
	"financas/domain"
	"financas/repository"
)

// Aparelhos que recebem aviso pelo app (fase 4 do item 54; .ai/specs/app-android-push/spec.md).

func tokenFicticio(n int) string {
	return "tok-ficticio-do-aparelho-" + strings.Repeat("x", 10) + string(rune('a'+n))
}

func tokensDoVinculo(t *testing.T, f *bancoFake, sub, sid string) []string {
	t.Helper()
	d := f.ler(t, "USER#"+sub, "SPACE#"+sid)
	l, _ := d["pushTokens"].([]string) // conjunto de textos (SS) volta como []string
	return l
}

// P-T1: registra no vinculo de quem chama, idempotente, devolve so a contagem; o sub vem do token, nunca do corpo.
func TestRegistrarAparelhoGravaNoVinculoDeQuemChama(t *testing.T) {
	f := cenario(t)
	tok := tokenFicticio(0)
	st, d := chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+tok+`","sub":"own"}`)
	esperar(t, "registrar", st, 200)
	if d["aparelhos"] != 1.0 || len(d) != 1 {
		t.Errorf("resposta = %v (so a contagem, nunca o token)", d)
	}
	if got := tokensDoVinculo(t, f, "mem", "s1"); len(got) != 1 || got[0] != tok {
		t.Errorf("vinculo do membro = %v", got)
	}
	if got := tokensDoVinculo(t, f, "own", "s1"); len(got) != 0 {
		t.Errorf("o sub do corpo nao pode valer: vinculo do dono = %v", got)
	}
	st, d = chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+tok+`"}`)
	if st != 200 || d["aparelhos"] != 1.0 {
		t.Errorf("registrar de novo e idempotente: %d %v", st, d)
	}
	st, _ = chamar(t, "estranho", "PUT", "spaces/s1/push", nil, `{"token":"`+tok+`"}`)
	esperar(t, "sem vinculo com o espaco", st, 403)
	if _, existe := f.itens[chave("USER#estranho", "SPACE#s1")]; existe {
		t.Error("nada pode ser criado para quem nao tem vinculo")
	}
}

// P-T2
func TestAparelhoComTokenInvalidoE400(t *testing.T) {
	cenario(t)
	for _, ruim := range []string{`{}`, `{"token":""}`, `{"token":"com espaco no meio do token 12345"}`, `{"token":"curto"}`,
		`{"token":"` + strings.Repeat("a", 1025) + `"}`, `{"token":"tem/barra/no/meio/do/token/1234"}`, `{"token":12345678901234567890}`} {
		if st, _ := chamar(t, "mem", "PUT", "spaces/s1/push", nil, ruim); st != 400 {
			t.Errorf("%.40s deveria ser 400, foi %d", ruim, st)
		}
	}
}

// P-T3: o quinto entra, o sexto diferente e 409 com a frase; um que ja esta continua aceito.
func TestLimiteDeCincoAparelhos(t *testing.T) {
	f := cenario(t)
	for i := 0; i < 5; i++ {
		if st, _ := chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+tokenFicticio(i)+`"}`); st != 200 {
			t.Fatalf("aparelho %d: status %d", i, st)
		}
	}
	st, d := chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+tokenFicticio(5)+`"}`)
	esperar(t, "sexto", st, 409)
	if d["erro"] != "Você já tem 5 aparelhos neste espaço. Desligue um deles para ligar este." {
		t.Errorf("frase = %v", d["erro"])
	}
	if st, _ := chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+tokenFicticio(2)+`"}`); st != 200 {
		t.Errorf("o que ja esta no conjunto continua aceito: %d", st)
	}
	if got := tokensDoVinculo(t, f, "mem", "s1"); len(got) != 5 {
		t.Errorf("conjunto = %d tokens", len(got))
	}
}

// P-T4: remover tira o token, token ausente nao e erro, e o atributo some quando o conjunto esvazia.
func TestRemoverAparelhoDoEspaco(t *testing.T) {
	f := cenario(t)
	a, b := tokenFicticio(0), tokenFicticio(1)
	chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+a+`"}`)
	chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+b+`"}`)
	st, d := chamar(t, "mem", "POST", "spaces/s1/push/remover", nil, `{"token":"`+a+`"}`)
	if st != 200 || d["aparelhos"] != 1.0 {
		t.Errorf("remover: %d %v", st, d)
	}
	if st, _ := chamar(t, "mem", "POST", "spaces/s1/push/remover", nil, `{"token":"`+a+`"}`); st != 200 {
		t.Errorf("token ausente nao e erro: %d", st)
	}
	chamar(t, "mem", "POST", "spaces/s1/push/remover", nil, `{"token":"`+b+`"}`)
	if _, tem := f.ler(t, "USER#mem", "SPACE#s1")["pushTokens"]; tem {
		t.Error("o atributo deveria sumir quando o conjunto esvazia")
	}
}

// P-T5: sair tira o token de TODOS os espacos da pessoa e de nenhum de outra.
func TestSairTiraOAparelhoDeTodosOsEspacosDaPessoa(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s2", "role": "owner"})
	f.colocar(t, doc{"PK": "SPACE#s2", "SK": "META", "nome": "Outro"})
	tok := tokenFicticio(3)
	for _, sub := range []string{"mem", "own"} {
		chamar(t, sub, "PUT", "spaces/s1/push", nil, `{"token":"`+tok+`"}`)
	}
	chamar(t, "mem", "PUT", "spaces/s2/push", nil, `{"token":"`+tok+`"}`)

	st, d := chamar(t, "mem", "POST", "push/sair", nil, `{"token":"`+tok+`"}`)
	if st != 200 || d["removidos"] != 2.0 {
		t.Errorf("sair: %d %v", st, d)
	}
	if len(tokensDoVinculo(t, f, "mem", "s1")) != 0 || len(tokensDoVinculo(t, f, "mem", "s2")) != 0 {
		t.Error("o token da pessoa deveria sair dos dois espacos")
	}
	if got := tokensDoVinculo(t, f, "own", "s1"); len(got) != 1 {
		t.Errorf("o vinculo de outra pessoa nao pode ser tocado: %v", got)
	}
	if st, _ := chamar(t, "mem", "POST", "push/sair", nil, `{"token":"curto"}`); st != 400 {
		t.Errorf("token invalido no sair: %d", st)
	}
}

// P-T6: Destinos devolve os tokens do push em ordem, junto do Telegram.
func TestDestinosDevolveOPushOrdenado(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member", "telegramChatId": "222222"})
	chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+tokenFicticio(2)+`"}`)
	chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+tokenFicticio(1)+`"}`)
	m, err := repository.NewPessoas(f, "").Destinos(context.Background(), "s1", "mem")
	if err != nil || len(m["telegram"]) != 1 || len(m["push"]) != 2 || m["push"][0] > m["push"][1] {
		t.Errorf("destinos = %v (%v)", m, err)
	}
	if m, _ := repository.NewPessoas(f, "").Destinos(context.Background(), "s1", "own"); len(m) != 0 {
		t.Errorf("sem destino o mapa vem vazio: %v", m)
	}
}

// P-T13: teste de aviso pelo push vai a todos os aparelhos da pessoa no espaco.
func TestMensagemDeTestePeloPush(t *testing.T) {
	cenario(t)
	trocaEnvio(t, nil)
	st, _ := chamar(t, "mem", "POST", "spaces/s1/avisos/teste", nil, `{"canal":"push"}`)
	esperar(t, "sem aparelho", st, 400)
	chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+tokenFicticio(0)+`"}`)
	chamar(t, "mem", "PUT", "spaces/s1/push", nil, `{"token":"`+tokenFicticio(1)+`"}`)
	st, _ = chamar(t, "mem", "POST", "spaces/s1/avisos/teste", nil, `{"canal":"push"}`)
	esperar(t, "com dois aparelhos", st, 200)
	if len(canalPushTeste.enviados) != 2 || len(canalTeste.enviados) != 0 {
		t.Errorf("push=%v telegram=%v", canalPushTeste.enviados, canalTeste.enviados)
	}
	canalPushTeste.falhaCom = domain.ErrCanalNaoConfigurado
	st, _ = chamar(t, "mem", "POST", "spaces/s1/avisos/teste", nil, `{"canal":"push"}`)
	esperar(t, "nao configurado", st, 503)
	canalPushTeste.falhaCom = errors.New("recusou")
	st, _ = chamar(t, "mem", "POST", "spaces/s1/avisos/teste", nil, `{"canal":"push"}`)
	esperar(t, "recusou", st, 502)
	st, _ = chamar(t, "estranho", "POST", "spaces/s1/avisos/teste", nil, `{"canal":"push"}`)
	esperar(t, "fora do espaco", st, 403)
}

// P-T14: as tres acoes existem, valem para dono e membro (cada um no proprio vinculo) e o que nao esta liberado continua negado.
func TestAcoesDeAparelhoNaAutorizacao(t *testing.T) {
	for _, papel := range []string{auth.PapelOwner, auth.PapelMember} {
		for _, a := range []auth.Acao{auth.AcaoRegistrarPush, auth.AcaoRemoverPush} {
			if auth.Permitido(papel, a) != auth.Liberado {
				t.Errorf("%s deveria poder %s", papel, a)
			}
		}
	}
	if auth.Permitido("estranho", auth.AcaoRegistrarPush) != auth.Negado {
		t.Error("papel desconhecido continua negado")
	}
	for metodo, esperado := range map[string]bool{"PUT": true, "GET": false, "DELETE": false, "POST": false} {
		if a, ok := auth.RotaParaAcao(metodo, []string{"push"}); ok != esperado || (ok && a != auth.AcaoRegistrarPush) {
			t.Errorf("%s spaces/{sid}/push = (%v, %v)", metodo, a, ok)
		}
	}
	if a, ok := auth.RotaParaAcao("POST", []string{"push", "remover"}); !ok || a != auth.AcaoRemoverPush {
		t.Errorf("POST push/remover = (%v, %v)", a, ok)
	}
	if _, ok := auth.RotaParaAcao("PUT", []string{"push", "remover"}); ok {
		t.Error("PUT push/remover nao existe")
	}
	if a, ok := auth.RotaSemEspaco("POST", []string{"push", "sair"}); !ok || a != auth.AcaoSairPush {
		t.Errorf("POST push/sair = (%v, %v)", a, ok)
	}
	for _, ruim := range [][]string{{"push"}, {"push", "outra"}, {"spaces", "s1", "push"}, {"me"}} {
		if _, ok := auth.RotaSemEspaco("POST", ruim); ok {
			t.Errorf("%v nao e rota sem espaco", ruim)
		}
	}
	if _, ok := auth.RotaSemEspaco("GET", []string{"push", "sair"}); ok {
		t.Error("GET push/sair nao existe")
	}
}
