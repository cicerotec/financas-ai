package domain

import (
	"fmt"
	"testing"
)

func txCombo(tags ...string) Lancamento {
	l := Lancamento{}
	ts := make([]any, len(tags))
	for i, t := range tags {
		ts[i] = t
	}
	l["tags"] = ts
	return l
}

func vezes(n int, l func() Lancamento) []Lancamento {
	var r []Lancamento
	for i := 0; i < n; i++ {
		r = append(r, l())
	}
	return r
}

func TestCombosOrdemDasTagsNaoImporta(t *testing.T) { // R2
	ls := []Lancamento{txCombo("mercado", "casa"), txCombo("casa", "mercado"), txCombo("mercado", "casa")}
	r := CombosSugeridos(ls)
	if len(r) != 1 || r[0]["contagem"] != 3 {
		t.Fatalf("esperava 1 conjunto com contagem 3, veio %v", r)
	}
	if tags := r[0]["tags"].([]any); tags[0] != "mercado" || tags[1] != "casa" {
		t.Fatalf("as tags saem na ordem do primeiro lancamento: %v", tags)
	}
}

func TestCombosMinimos(t *testing.T) { // R2
	if r := CombosSugeridos(vezes(2, func() Lancamento { return txCombo("a", "b") })); len(r) != 0 {
		t.Fatalf("2 usos nao entram: %v", r)
	}
	if r := CombosSugeridos(vezes(3, func() Lancamento { return txCombo("a", "b") })); len(r) != 1 {
		t.Fatalf("3 usos entram: %v", r)
	}
	if r := CombosSugeridos(vezes(5, func() Lancamento { return txCombo("a") })); len(r) != 0 {
		t.Fatalf("1 tag nao entra: %v", r)
	}
	if r := CombosSugeridos(vezes(5, func() Lancamento { return txCombo("a", "a") })); len(r) != 0 {
		t.Fatalf("tag repetida conta uma vez, entao 1 tag nao entra: %v", r)
	}
}

func TestCombosIgnoraExcluidosEPrevistos(t *testing.T) { // R2
	ex := txCombo("a", "b")
	ex["excluidoEm"] = "2026-11-01T10:00:00Z"
	pv := txCombo("a", "b")
	pv["previsto"] = true
	ls := []Lancamento{txCombo("a", "b"), txCombo("a", "b"), ex, pv}
	if r := CombosSugeridos(ls); len(r) != 0 {
		t.Fatalf("excluido e previsto nao contam: %v", r)
	}
}

func TestCombosOrdemELimite(t *testing.T) { // R4
	var ls []Lancamento
	for i := 0; i < 12; i++ { // 12 conjuntos, cada um com 3 usos
		ls = append(ls, vezes(3, func() Lancamento { return txCombo(fmt.Sprintf("t%02d", i), "x") })...)
	}
	ls = append(ls, vezes(2, func() Lancamento { return txCombo("t11", "x") })...) // t11 sobe para 5
	r := CombosSugeridos(ls)
	if len(r) != CombosMax {
		t.Fatalf("guarda no maximo %d: %d", CombosMax, len(r))
	}
	if r[0]["contagem"] != 5 || r[0]["tags"].([]any)[0] != "t11" {
		t.Fatalf("o mais usado vem primeiro: %v", r[0])
	}
	if r[1]["tags"].([]any)[0] != "t00" || r[2]["tags"].([]any)[0] != "t01" { // empate: pela chave
		t.Fatalf("empate pela chave: %v %v", r[1], r[2])
	}
}
