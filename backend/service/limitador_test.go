package service

import (
	"fmt"
	"testing"
	"time"
)

// Limites em memoria da recuperacao do MFA (spec mfa-opcional, R29): M-T24 e M-T24c no nivel do limitador.

var t0 = time.Date(2026, 11, 19, 12, 0, 30, 0, time.UTC)

func TestParBloqueiaNaDecimaFalhaPorUmaHora(t *testing.T) {
	l := NewLimitadorRecuperacao()
	par := "sub|203.0.113.9"
	for i := 1; i <= MaxFalhasPar; i++ {
		if l.ParBloqueado(par, t0) {
			t.Fatalf("o par nao devia estar bloqueado antes da falha %d", i)
		}
		l.RegistrarFalhaPar(par, t0)
	}
	if !l.ParBloqueado(par, t0) {
		t.Fatal("a 10a falha devia bloquear o par")
	}
	if !l.ParBloqueado(par, t0.Add(BloqueioPar-time.Second)) {
		t.Error("o bloqueio dura 1 hora")
	}
	if l.ParBloqueado(par, t0.Add(BloqueioPar)) {
		t.Error("passada 1 hora o par volta")
	}
	if l.ParBloqueado("sub|198.51.100.1", t0) || l.ParBloqueado("outra|203.0.113.9", t0) {
		t.Error("outra origem ou outra pessoa nao e afetada (o par e (pessoa, origem))")
	}
}

func TestFalhasForaDaJanelaNaoSeSomam(t *testing.T) {
	l := NewLimitadorRecuperacao()
	par := "sub|ip"
	for i := 0; i < MaxFalhasPar-1; i++ {
		l.RegistrarFalhaPar(par, t0)
	}
	l.RegistrarFalhaPar(par, t0.Add(JanelaPar+time.Minute)) // a janela de 1 hora recomecou
	if l.ParBloqueado(par, t0.Add(JanelaPar+time.Minute)) {
		t.Error("9 falhas numa janela e 1 na outra nao bloqueiam")
	}
}

func TestSucessoZeraOContadorDoPar(t *testing.T) {
	l := NewLimitadorRecuperacao()
	par := "sub|ip"
	for i := 0; i < MaxFalhasPar-1; i++ {
		l.RegistrarFalhaPar(par, t0)
	}
	l.ZerarPar(par)
	for i := 0; i < MaxFalhasPar-1; i++ {
		l.RegistrarFalhaPar(par, t0)
	}
	if l.ParBloqueado(par, t0) {
		t.Error("o sucesso zera o contador: 9 + 9 falhas nao bloqueiam")
	}
}

func TestOrigemTrintaPorMinuto(t *testing.T) {
	l := NewLimitadorRecuperacao()
	for i := 1; i <= MaxPorMinutoOrigem; i++ {
		if !l.PermiteOrigem("203.0.113.9", t0) {
			t.Fatalf("a chamada %d devia passar", i)
		}
	}
	if l.PermiteOrigem("203.0.113.9", t0.Add(10*time.Second)) {
		t.Error("a 31a chamada no mesmo minuto devia ser barrada")
	}
	if !l.PermiteOrigem("198.51.100.1", t0) {
		t.Error("outra origem nao e afetada")
	}
	if !l.PermiteOrigem("203.0.113.9", t0.Add(time.Minute)) {
		t.Error("no minuto seguinte a origem volta")
	}
}

// O limitador nao cresce sem limite: com muitas chaves, ele se limpa.
func TestLimitadorTemTetoDeEntradas(t *testing.T) {
	l := NewLimitadorRecuperacao()
	for i := 0; i < tetoEntradas*2; i++ {
		l.PermiteOrigem(fmt.Sprintf("ip-%d", i), t0)
		l.RegistrarFalhaPar(fmt.Sprintf("par-%d", i), t0)
	}
	if len(l.origens) > tetoEntradas || len(l.pares) > tetoEntradas {
		t.Errorf("o limitador cresceu alem do teto: %d origens, %d pares", len(l.origens), len(l.pares))
	}
}
