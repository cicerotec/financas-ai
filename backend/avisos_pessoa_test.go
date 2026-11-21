package main

import (
	"context"
	"strings"
	"testing"
)

// Agrupamento por pessoa (spec telegram-vinculo, R13 e D7): a chave da rotina e (destino, sub). Spaces da mesma pessoa no mesmo
// destino viram uma mensagem; pessoas diferentes nunca se misturam, mesmo dividindo o chat.

// segundoEspaco poe o espaco s2, em que "mem" e dono, com um previsto de "mem" vencendo em 2026-10-10 (avisa 3 dias antes), ligado ao chat dado.
func segundoEspaco(t *testing.T, f *bancoFake, chatDoMem string) {
	t.Helper()
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s2", "role": "owner", "telegramChatId": chatDoMem})
	f.colocar(t, doc{"PK": "SPACE#s2", "SK": "META", "nome": "Sitio", "ownerSub": "mem"})
	f.colocar(t, doc{"PK": "SPACE#s2", "SK": "CFG#LISTAS", "banco": []any{"ITAU"}, "statusFuturo": map[string]any{"PREVISTO": true}, "version": 1.0})
	f.colocar(t, doc{"PK": "SPACE#s2", "SK": "CFG#SALDOS", "version": 1.0, "porBanco": map[string]any{"ITAU": map[string]any{"saldoInicial": 0.0, "dataInicio": "2020-01-01"}}})
	corpo := `{"dataEvento":"2026-10-10T12:00:00.000Z","valor":99,"tipo":"saida","status":"PAGO","banco":"ITAU",
		"descricao":"Racao","aviso":{"dias":[3,0],"insistir":false}}`
	st, _ := chamar(t, "mem", "POST", "spaces/s2/tx", nil, corpo)
	esperar(t, "criar no s2", st, 201)
}

func mensagensPorChat(env []envio) map[string][]string {
	m := map[string][]string{}
	for _, e := range env {
		m[e.chat] = append(m[e.chat], e.texto)
	}
	return m
}

// V-T12a: dois spaces da MESMA pessoa no mesmo chat: uma mensagem, a contagem somada e nada que diga de qual space.
func TestRotinaSpacesDaMesmaPessoaNoMesmoChatViramUmaMensagem(t *testing.T) {
	f, _ := cenarioAvisos(t) // s1: "mem" cria, "own" e dono; own=111111, mem=222222
	segundoEspaco(t, f, "222222")
	env := trocaEnvio(t, nil)

	n, err := rodarAvisos(context.Background(), agoraBR("2026-10-07"))
	if err != nil || n != 2 {
		t.Fatalf("rodarAvisos = (%d, %v), esperado 2 resumos (um por pessoa)", n, err)
	}
	porChat := mensagensPorChat(*env)
	if got := porChat["222222"]; len(got) != 1 || !strings.Contains(got[0], "2 a vencer") {
		t.Errorf("mem tem dois spaces no mesmo chat: uma mensagem com 2 a vencer, recebeu %q", got)
	}
	if got := porChat["111111"]; len(got) != 1 || !strings.Contains(got[0], "1 a vencer") {
		t.Errorf("own so tem o s1: 1 a vencer, recebeu %q", got)
	}
	for _, e := range *env {
		for _, proibido := range []string{"Familia", "Sitio", "s1", "s2", "Racao", "Conta de luz"} {
			if strings.Contains(e.texto, proibido) {
				t.Errorf("a mensagem diz de que space ou lancamento veio (%q): %q", proibido, e.texto)
			}
		}
	}
}

// V-T12a: o mesmo chat dividido por duas pessoas: duas mensagens, e nenhuma soma as linhas da outra.
func TestRotinaPessoasDiferentesNoMesmoChatNuncaSeMisturam(t *testing.T) {
	f, _ := cenarioAvisos(t)
	segundoEspaco(t, f, "111111") // o chat do dono do s1 e agora tambem o do "mem" no s2
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member", "telegramChatId": "111111"})
	env := trocaEnvio(t, nil)

	n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-07"))
	got := mensagensPorChat(*env)["111111"]
	if n != 2 || len(got) != 2 {
		t.Fatalf("enviados=%d, mensagens no chat=%d: esperado 2 e 2 (uma por pessoa)", n, len(got))
	}
	somaDeAVencer := 0
	for _, m := range got {
		switch {
		case strings.Contains(m, "2 a vencer"):
			somaDeAVencer += 2
		case strings.Contains(m, "1 a vencer"):
			somaDeAVencer++
		}
	}
	if somaDeAVencer != 3 { // own: 1 (s1); mem: 2 (s1 criado por ele e s2 dele), nunca 3 numa so mensagem
		t.Errorf("mensagens = %q: esperada uma com 1 e outra com 2 a vencer", got)
	}
	for _, m := range got {
		if strings.Contains(m, "3 a vencer") {
			t.Errorf("uma mensagem somou as linhas das duas pessoas: %q", m)
		}
	}
}

// V-T12a e V-T14: destino recusado pelo canal apaga o destino de TODOS os vinculos de onde ele saiu, e so dessa pessoa; a rotina nao
// devolve erro, o grupo recusado nao conta como entregue e os outros destinos entregam.
func TestRotinaDestinoRecusadoApagaOsVinculosDaPessoaEOsOutrosEntregam(t *testing.T) {
	f, _ := cenarioAvisos(t)
	segundoEspaco(t, f, "222222")
	env := trocaEnvio(t, nil)
	canalTeste.recusados = map[string]bool{"222222": true} // o "mem" bloqueou o bot

	n, err := rodarAvisos(context.Background(), agoraBR("2026-10-07"))
	if err != nil {
		t.Fatalf("a rotina nao pode devolver erro: %v", err)
	}
	if n != 1 || len(*env) != 1 || (*env)[0].chat != "111111" {
		t.Errorf("so o dono entrega: enviados=%d mensagens=%v", n, *env)
	}
	for _, sid := range []string{"s1", "s2"} {
		if _, tem := f.ler(t, "USER#mem", "SPACE#"+sid)["telegramChatId"]; tem {
			t.Errorf("o chat recusado deveria sair do vinculo do mem no %s", sid)
		}
	}
	if got := f.ler(t, "USER#own", "SPACE#s1")["telegramChatId"]; got != "111111" {
		t.Errorf("o chat do dono nao pode ser apagado: %v", got)
	}
}
