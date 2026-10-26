package domain

import "testing"

func cartaoV(dia, venc int, fat map[string]FaturaCfg) Cartao {
	return Cartao{DiaPadrao: dia, DiaVencimento: venc, Faturas: fat}
}

// G-T1: o vencimento da fatura.
func TestVencimentoDe(t *testing.T) {
	casos := []struct {
		nome string
		c    Cartao
		ym   string
		quer string
	}{
		{"escolhido na fatura", cartaoV(28, 5, map[string]FaturaCfg{"2026-10": {Vencimento: "2026-11-09"}}), "2026-10", "2026-11-09"},
		{"o dia do cartao nao vale para a fatura", cartaoV(28, 5, nil), "2026-10", ""},
		{"nem para outra fatura", cartaoV(28, 5, map[string]FaturaCfg{"2026-10": {Vencimento: "2026-11-09"}}), "2026-09", ""},
		{"sem nenhum dos dois", cartaoV(28, 0, nil), "2026-10", ""},
	}
	for _, x := range casos {
		if got := VencimentoDe(x.c, x.ym); got != x.quer {
			t.Errorf("%s: %q, esperado %q", x.nome, got, x.quer)
		}
	}
}

// A sugestao do vencimento (so para preencher o campo) segue o dia do cartao.
func TestVencimentoSugerido(t *testing.T) {
	casos := []struct {
		nome string
		c    Cartao
		ym   string
		quer string
	}{
		{"dia menor que o do fechamento: mes seguinte", cartaoV(28, 5, nil), "2026-10", "2026-11-05"},
		{"dia maior que o do fechamento: no mesmo mes", cartaoV(10, 20, nil), "2026-10", "2026-10-20"},
		{"dia igual ao do fechamento: o mes seguinte (tem que ser depois)", cartaoV(10, 10, nil), "2026-10", "2026-11-10"},
		{"dia 31 em abril", cartaoV(20, 31, nil), "2026-04", "2026-04-30"},
		{"virada de ano", cartaoV(28, 5, nil), "2026-12", "2027-01-05"},
		{"sem dia", cartaoV(28, 0, nil), "2026-10", ""},
	}
	for _, x := range casos {
		if got := VencimentoSugerido(x.c, x.ym); got != x.quer {
			t.Errorf("%s: %q, esperado %q", x.nome, got, x.quer)
		}
	}
}

// G-T2: a precedencia do pagador.
func TestPagadorDe(t *testing.T) {
	c := Cartao{Pagador: "BANCO-A", Faturas: map[string]FaturaCfg{"2026-10": {Pagador: "BANCO-B"}}}
	if PagadorDe(c, "2026-10") != "BANCO-B" || PagadorDe(c, "2026-09") != "" || PagadorDe(Cartao{}, "2026-09") != "" {
		t.Error("pagador: so o da fatura; o padrao do cartao nao vale para ela")
	}
	if PagadorSugerido(c, "2026-10") != "BANCO-B" || PagadorSugerido(c, "2026-09") != "BANCO-A" || PagadorSugerido(Cartao{}, "2026-09") != "" {
		t.Error("sugestao: da fatura, depois do cartao, depois nenhum")
	}
}

func cartoesDe(nomes ...string) Doc {
	por := map[string]any{}
	for _, n := range nomes {
		por[n] = map[string]any{"diaPadrao": 28.0, "faturas": map[string]any{}}
	}
	return Doc{"porBanco": por}
}

// ValidarPagaFatura: a marca e uma label, mas precisa ser bem formada.
func TestValidarPagaFatura(t *testing.T) {
	cart := cartoesDe("CARTAO-A")
	pago := func(m any, extra Doc) Doc {
		d := Doc{"status": "PAGO", "banco": "BANCO-A", "pagaFatura": m}
		for k, v := range extra {
			d[k] = v
		}
		return d
	}
	ok := map[string]any{"meio": "CARTAO-A", "ym": "2026-09"}
	casos := []struct {
		nome string
		d    Doc
		erro bool
	}{
		{"valida", pago(ok, nil), false},
		{"sem a marca", Doc{"status": "PAGO"}, false},
		{"meio que nao e cartao", pago(map[string]any{"meio": "BANCO-B", "ym": "2026-09"}, nil), true},
		{"mes invalido", pago(map[string]any{"meio": "CARTAO-A", "ym": "2026-13"}, nil), true},
		{"sem o mes", pago(map[string]any{"meio": "CARTAO-A"}, nil), true},
		{"campo a mais", pago(map[string]any{"meio": "CARTAO-A", "ym": "2026-09", "x": 1.0}, nil), true},
		{"nao e objeto", pago("CARTAO-A", nil), true},
		{"compra de credito", pago(ok, Doc{"status": "CREDITO"}), true},
		{"debito do proprio banco do cartao", pago(ok, Doc{"banco": "CARTAO-A"}), false},
	}
	for _, c := range casos {
		if err := ValidarPagaFatura(c.d, cart); (err != nil) != c.erro {
			t.Errorf("%s: erro = %v, esperado erro=%v", c.nome, err, c.erro)
		}
	}
	d := pago(nil, nil)
	if err := ValidarPagaFatura(d, cart); err != nil || d["pagaFatura"] != nil {
		t.Error("null remove a marca")
	}
	if _, tem := d["pagaFatura"]; tem {
		t.Error("a chave sai")
	}
}

// G-T3: o efeito na fatura e o que muda a trava.
func TestEfeitoEMudaFatura(t *testing.T) {
	cart := Doc{"porBanco": map[string]any{"C": map[string]any{"diaPadrao": 12.0, "faturas": map[string]any{}}}}
	base := func() Lancamento {
		return Lancamento{"banco": "C", "status": "CREDITO", "tipo": "saida", "valor": 100.0, "dataEvento": "2026-10-05T15:00:00.000Z"}
	}
	com := func(k string, v any) Lancamento { l := base(); l[k] = v; return l }
	if e := EfeitoFatura(base(), cart); !e.Existe || e.Ym != "2026-10" || e.Valor != 100 || e.Meio != "C" {
		t.Errorf("efeito: %+v", e)
	}
	if e := EfeitoFatura(com("dataEvento", "2026-10-20T15:00:00.000Z"), cart); e.Ym != "2026-11" {
		t.Errorf("depois do fechamento cai na proxima: %+v", e)
	}
	if e := EfeitoFatura(com("faturaAjuste", 1.0), cart); e.Ym != "2026-11" {
		t.Errorf("ajuste +1: %+v", e)
	}
	if EfeitoFatura(com("banco", "OUTRO"), cart).Existe || EfeitoFatura(com("status", "PAGO"), cart).Existe || EfeitoFatura(com("oculto", true), cart).Existe || EfeitoFatura(com("previsto", true), cart).Existe {
		t.Error("so compra de credito de um cartao, visivel e real, entra")
	}
	muda := []struct {
		nome string
		a, b Lancamento
		quer bool
	}{
		{"descricao", base(), com("descricao", "x"), false},
		{"tags", base(), com("tags", []any{"a"}), false},
		{"a marca", base(), com("pagaFatura", map[string]any{"meio": "C", "ym": "2026-09"}), false},
		{"valor", base(), com("valor", 101.0), true},
		{"data para outra fatura", base(), com("dataEvento", "2026-10-20T15:00:00.000Z"), true},
		{"data na mesma fatura", base(), com("dataEvento", "2026-10-06T15:00:00.000Z"), false},
		{"faturaAjuste", base(), com("faturaAjuste", -1.0), true},
		{"oculto", base(), com("oculto", true), true},
		{"deixa de ser credito", base(), com("status", "PAGO"), true},
	}
	for _, c := range muda {
		if got := MudaFatura(c.a, c.b, cart); got != c.quer {
			t.Errorf("%s: MudaFatura = %v, esperado %v", c.nome, got, c.quer)
		}
	}
	if f := FaturasAfetadas(base(), com("dataEvento", "2026-10-20T15:00:00.000Z"), cart); len(f) != 2 {
		t.Errorf("as duas faturas: %v", f)
	}
	if f := FaturasAfetadas(base(), com("descricao", "x"), cart); len(f) != 1 {
		t.Errorf("sem repetir: %v", f)
	}
}
