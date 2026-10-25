package domain

import (
	"testing"
	"time"
)

func em(ano int, mes time.Month, dia, hora, min int) time.Time {
	return time.Date(ano, mes, dia, hora, min, 0, 0, FusoBR)
}

// P-T1
func TestEhPrevisto(t *testing.T) {
	casos := []struct {
		l    Lancamento
		quer bool
	}{
		{Lancamento{"previsto": true, "status": "PAGO"}, true},
		{Lancamento{"status": "PREVISTO"}, true},
		{Lancamento{"status": "TRANSFERINDO"}, true},
		{Lancamento{"status": "CREDITANDO"}, true},
		{Lancamento{"status": "PAGO"}, false},
		{Lancamento{"status": "PAGO", "previsto": false}, false},
	}
	for i, c := range casos {
		if got := EhPrevisto(c.l); got != c.quer {
			t.Errorf("caso %d: %v, esperado %v", i, got, c.quer)
		}
	}
}

// P-T2
func TestPrevistoPorData(t *testing.T) {
	agora := em(2026, 10, 24, 12, 0)
	casos := []struct {
		data string
		quer bool
	}{
		{"2026-10-24T03:01:00.000Z", false}, // 00:01 em Sao Paulo
		{"2026-10-25T02:59:00.000Z", false}, // 23:59 em Sao Paulo
		{"2026-10-25T03:00:00.000Z", true},  // 00:00 de amanha
		{"2026-10-23T15:00:00.000Z", false},
		{"nao e data", false},
	}
	for _, c := range casos {
		if got := PrevistoPorData(c.data, agora); got != c.quer {
			t.Errorf("%s: %v, esperado %v", c.data, got, c.quer)
		}
	}
	// virada de ano: 31/12 vs 01/01
	ano := em(2026, 12, 31, 23, 30)
	if PrevistoPorData("2027-01-01T02:00:00.000Z", ano) { // 31/12 23:00 em SP: ainda hoje
		t.Error("31/12 23:00 em Sao Paulo ainda e hoje")
	}
	if !PrevistoPorData("2027-01-01T03:00:00.000Z", ano) {
		t.Error("01/01 00:00 em Sao Paulo e amanha")
	}
}

// P-T3
func TestAtrasado(t *testing.T) {
	agora := em(2026, 10, 24, 12, 0)
	prev := func(data string) Lancamento { return Lancamento{"previsto": true, "dataEvento": data} }
	if !Atrasado(prev("2026-10-23T15:00:00.000Z"), agora) || Atrasado(prev("2026-10-24T15:00:00.000Z"), agora) || Atrasado(prev("2026-10-25T15:00:00.000Z"), agora) {
		t.Error("atrasado so quando o dia e anterior a hoje")
	}
	if Atrasado(Lancamento{"status": "PAGO", "dataEvento": "2026-10-01T15:00:00.000Z"}, agora) {
		t.Error("o que nao e previsto nunca e atrasado")
	}
}

// P-T24 e P-T47: o efeito no saldo e o que muda a trava.
func TestEfeitoEMudaSaldo(t *testing.T) {
	base := func() Lancamento {
		return Lancamento{"valor": 10.0, "tipo": "saida", "status": "PAGO", "banco": "ITAU", "dataEvento": "2026-10-01T12:00:00.000Z"}
	}
	com := func(k string, v any) Lancamento { l := base(); l[k] = v; return l }
	if e := EfeitoSaldo(base()); !e.Existe || e.Valor != -10 || e.Alvo != (Alvo{"ITAU", "2026-10"}) {
		t.Errorf("efeito: %+v", e)
	}
	if e := EfeitoSaldo(com("tipo", "entrada")); e.Valor != 10 {
		t.Errorf("entrada soma: %+v", e)
	}
	muda := []struct {
		nome string
		a, b Lancamento
		quer bool
	}{
		{"so a descricao", base(), com("descricao", "x"), false},
		{"tags", base(), com("tags", []any{"a"}), false},
		{"valor", base(), com("valor", 11.0), true},
		{"tipo", base(), com("tipo", "entrada"), true},
		{"meio", base(), com("banco", "NUBANK"), true},
		{"data em outro mes", base(), com("dataEvento", "2026-11-01T12:00:00.000Z"), true},
		{"data no mesmo mes", base(), com("dataEvento", "2026-10-09T12:00:00.000Z"), false},
		{"oculto ligado", base(), com("oculto", true), true},
		{"oculto desligado", com("oculto", true), base(), true},
		{"previsto ligado", base(), com("previsto", true), true},
		{"previsto desligado", com("previsto", true), base(), true},
		{"editar o valor de um previsto", com("previsto", true), func() Lancamento { l := com("previsto", true); l["valor"] = 99.0; return l }(), false},
		{"PAGO para CONTAS (os dois contam)", base(), com("status", "CONTAS"), false},
		{"PAGO para CREDITO (deixa de contar)", base(), com("status", "CREDITO"), true},
		{"CREDITO para CREDITO IN (nenhum conta)", com("status", "CREDITO"), com("status", "CREDITO IN"), false},
		{"valor de CREDITO IN", com("status", "CREDITO IN"), func() Lancamento { l := com("status", "CREDITO IN"); l["valor"] = 99.0; return l }(), false},
	}
	for _, c := range muda {
		if got := MudaSaldo(c.a, c.b); got != c.quer {
			t.Errorf("%s: MudaSaldo = %v, esperado %v", c.nome, got, c.quer)
		}
	}
	// os alvos: so os efeitos nao nulos, sem repetir
	a := AlvosDoSaldo(com("previsto", true), base())
	if len(a) != 1 || a[0] != (Alvo{"ITAU", "2026-10"}) {
		t.Errorf("alvos de previsto -> real: %v", a)
	}
	if a := AlvosDoSaldo(base(), com("dataEvento", "2026-11-05T12:00:00.000Z")); len(a) != 2 {
		t.Errorf("os dois meses: %v", a)
	}
	if a := AlvosDoSaldo(base(), com("descricao", "x")); len(a) != 1 {
		t.Errorf("sem repetir: %v", a)
	}
}

// P-T35 e P-T45: o catalogo contra os padroes de hoje.
func TestCatalogoDoCorte(t *testing.T) {
	if got := Selecionaveis(); len(got) != 2 || got[0] != "PAGO" || got[1] != "CREDITO" {
		t.Errorf("selecionaveis: %v", got)
	}
	quer := map[string]struct {
		saldo, fora, credito, previsto bool
		ajuste                          int
	}{
		"PAGO":         {saldo: true},
		"CREDITO":      {credito: true},
		"CONTAS":       {saldo: true, fora: true},
		"CREDITO IN":   {credito: true, ajuste: -1},
		"CREDITO EX":   {credito: true, ajuste: 1},
		"PREVISTO":     {previsto: true},
		"TRANSFERINDO": {fora: true, previsto: true},
		"CREDITANDO":   {previsto: true},
	}
	if len(Catalogo) != len(quer) {
		t.Errorf("o catalogo tem %d entradas, esperado %d", len(Catalogo), len(quer))
	}
	for nome, q := range quer {
		e, ok := EntradaDe(nome)
		if !ok || e.ContaNoSaldo != q.saldo || e.ForaDoTotal != q.fora || e.Credito != q.credito || e.Previsto != q.previsto || e.AjustePadrao != q.ajuste {
			t.Errorf("%s: %+v", nome, e)
		}
		if e.Selecionavel != (nome == "PAGO" || nome == "CREDITO") {
			t.Errorf("%s: selecionavel = %v", nome, e.Selecionavel)
		}
	}
	// os nomes antigos de previsto sao os do catalogo com previsto: true
	for _, nome := range StatusAntigosPrevistos {
		if e, _ := EntradaDe(nome); !e.Previsto {
			t.Errorf("%s deveria ser previsto no catalogo", nome)
		}
	}
	if r, c := RotuloECor("FOO"); r != "FOO" || c != CorNeutra {
		t.Errorf("fora do catalogo: %s %s", r, c)
	}
}
