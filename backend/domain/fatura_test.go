package domain

import "testing"

func cartaoTeste(dia int, fech map[string]string) Cartao {
	c := Cartao{DiaPadrao: dia, Faturas: map[string]FaturaCfg{}}
	for ym, f := range fech {
		c.Faturas[ym] = FaturaCfg{Fechamento: f}
	}
	return c
}

// F-T1: o fechamento da fatura.
func TestFechamentoDe(t *testing.T) {
	casos := []struct {
		nome string
		c    Cartao
		ym   string
		quer string
	}{
		{"padrao", cartaoTeste(12, nil), "2026-10", "2026-10-12"},
		{"sem dia padrao vale 1", cartaoTeste(0, nil), "2026-10", "2026-10-01"},
		{"escolhido no mes", cartaoTeste(12, map[string]string{"2026-10": "2026-10-15"}), "2026-10", "2026-10-15"},
		{"herda o dia do ultimo escolhido", cartaoTeste(12, map[string]string{"2026-07": "2026-07-20"}), "2026-10", "2026-10-20"},
		{"o escolhido de um mes nao vale para o anterior", cartaoTeste(12, map[string]string{"2026-07": "2026-07-20"}), "2026-06", "2026-06-12"},
		{"dia 31 em fevereiro", cartaoTeste(31, nil), "2026-02", "2026-02-28"},
		{"dia 31 em fevereiro bissexto", cartaoTeste(31, nil), "2028-02", "2028-02-29"},
		{"dia 31 em abril", cartaoTeste(31, nil), "2026-04", "2026-04-30"},
		{"herdado dia 31 em abril", cartaoTeste(12, map[string]string{"2026-03": "2026-03-31"}), "2026-04", "2026-04-30"},
		{"mais de 60 meses atras nao herda", cartaoTeste(12, map[string]string{"2020-01": "2020-01-25"}), "2026-10", "2026-10-12"},
		{"exatamente 60 meses atras herda", cartaoTeste(12, map[string]string{"2021-10": "2021-10-25"}), "2026-10", "2026-10-25"},
	}
	for _, x := range casos {
		if got := FechamentoDe(x.c, x.ym); got != x.quer {
			t.Errorf("%s: %s, esperado %s", x.nome, got, x.quer)
		}
	}
}

// F-T2: periodo e fatura base nas bordas.
func TestPeriodoEFaturaBase(t *testing.T) {
	c := cartaoTeste(12, nil)
	ini, fim := PeriodoFatura(c, "2026-10")
	if ini != "2026-09-13" || fim != "2026-10-12" {
		t.Errorf("periodo = %s a %s", ini, fim)
	}
	ini, fim = PeriodoFatura(c, "2027-01")
	if ini != "2026-12-13" || fim != "2027-01-12" {
		t.Errorf("periodo na virada de ano = %s a %s", ini, fim)
	}
	for dia, quer := range map[string]string{
		"2026-10-12": "2026-10", // no dia do fechamento ainda e desta fatura
		"2026-10-13": "2026-11", // no dia seguinte ja e da proxima
		"2026-09-13": "2026-10", // primeiro dia do periodo
		"2026-12-13": "2027-01", // virada de ano
		"2026-12-31": "2027-01",
	} {
		if got := FaturaBase(c, dia); got != quer {
			t.Errorf("FaturaBase(%s) = %s, esperado %s", dia, got, quer)
		}
	}
}

// F-T3: a fatura atual antes, no dia e depois do fechamento.
func TestFaturaAtual(t *testing.T) {
	c := cartaoTeste(12, nil)
	for hoje, quer := range map[string]string{"2026-10-11": "2026-10", "2026-10-12": "2026-10", "2026-10-13": "2026-11"} {
		if got := FaturaAtual(c, hoje); got != quer {
			t.Errorf("FaturaAtual(%s) = %s, esperado %s", hoje, got, quer)
		}
	}
}

// F-T4: o ajuste efetivo.
func TestAjusteEfetivo(t *testing.T) {
	casos := []struct {
		l    Lancamento
		quer int
	}{
		{Lancamento{"status": "CREDITO EX", "faturaAjuste": 0.0}, 0},
		{Lancamento{"status": "CREDITO", "faturaAjuste": 1.0}, 1},
		{Lancamento{"status": "CREDITO", "faturaAjuste": -1.0}, -1},
		{Lancamento{"status": "CREDITO EX"}, 1},
		{Lancamento{"status": "CREDITO IN"}, -1},
		{Lancamento{"status": "credito  in"}, -1},
		{Lancamento{"status": "CRÉDITO EX"}, 1},
		{Lancamento{"status": "CREDITO"}, 0},
		{Lancamento{"status": "CREDITO IN", "faturaAjuste": "1"}, -1}, // texto nao e numero: vale o padrao
	}
	for i, x := range casos {
		if got := AjusteEfetivo(x.l); got != x.quer {
			t.Errorf("caso %d: %d, esperado %d", i, got, x.quer)
		}
	}
}

// F-T5: o que entra na fatura.
func TestContaNaFatura(t *testing.T) {
	listas := Doc{}
	base := func(m Lancamento) Lancamento {
		l := Lancamento{"banco": "CARTAO-A", "status": "CREDITO", "tipo": "saida", "valor": 10.0}
		for k, v := range m {
			l[k] = v
		}
		return l
	}
	casos := []struct {
		nome string
		l    Lancamento
		quer bool
	}{
		{"credito normal", base(nil), true},
		{"oculto", base(Lancamento{"oculto": true}), false},
		{"excluido", base(Lancamento{"excluidoEm": "2026-10-01T00:00:00.000Z"}), false},
		{"evento futuro", base(Lancamento{"status": "CREDITANDO"}), false},
		{"outro banco", base(Lancamento{"banco": "OUTRO"}), false},
		{"status PAGO", base(Lancamento{"status": "PAGO"}), false},
		{"estorno entra", base(Lancamento{"tipo": "entrada"}), true},
		{"credito com acento", base(Lancamento{"status": "Crédito IN"}), true},
	}
	for _, x := range casos {
		if got := ContaNaFatura(x.l, "CARTAO-A", listas); got != x.quer {
			t.Errorf("%s: %v, esperado %v", x.nome, got, x.quer)
		}
	}
	// um status marcado como nao real na configuracao tambem fica de fora
	l := base(Lancamento{"status": "CREDITO X"})
	if !ContaNaFatura(l, "CARTAO-A", listas) {
		t.Error("sem configuracao, entra")
	}
	if ContaNaFatura(l, "CARTAO-A", Doc{"statusReal": map[string]any{"CREDITO X": false}}) {
		t.Error("status nao real nao entra")
	}
}

func lanc(id, dataEvento, status, tipo string, valor float64, extra Lancamento) Doc {
	l := Doc{"id": id, "dataEvento": dataEvento, "status": status, "tipo": tipo, "valor": valor, "banco": "C", "descricao": id}
	for k, v := range extra {
		l[k] = v
	}
	return l
}

// F-T6 e F-T7: total com tres faturas, um estorno, um ajuste +1 e o fuso de Sao Paulo.
func TestTotalFaturaEFuso(t *testing.T) {
	c := cartaoTeste(12, nil)
	itens := []Doc{
		lanc("a", "2026-09-20T15:00:00.000Z", "CREDITO", "saida", 100.10, nil),
		lanc("b", "2026-10-05T15:00:00.000Z", "CREDITO", "saida", 50.20, nil),
		lanc("c", "2026-10-06T15:00:00.000Z", "CREDITO", "entrada", 20.00, nil),
		lanc("d", "2026-10-07T15:00:00.000Z", "CREDITO EX", "saida", 30.00, nil),
		lanc("e", "2026-10-20T15:00:00.000Z", "CREDITO", "saida", 40.00, nil),
		lanc("f", "2026-10-08T15:00:00.000Z", "CREDITO", "saida", 5.00, Lancamento{"faturaAjuste": 1.0}),
		lanc("g", "2026-09-01T15:00:00.000Z", "CREDITO", "saida", 999, Lancamento{"oculto": true}),
	}
	rs := RegistrosDoCartao(c, "C", itens, Doc{})
	// fatura 2026-10: a 100.10 + b 50.20 - c 20 = 130.30
	if got := TotalFatura(rs, "2026-10"); got != 130.30 {
		t.Errorf("total 2026-10 = %v, esperado 130.3", got)
	}
	// fatura 2026-11: d 30 (CREDITO EX) + e 40 + f 5 (ajuste explicito +1) = 75
	if got := TotalFatura(rs, "2026-11"); got != 75.00 {
		t.Errorf("total 2026-11 = %v, esperado 75", got)
	}
	// F-T7: 02:30 UTC de 01/11 e 23:30 de 31/10 em Sao Paulo
	rs = RegistrosDoCartao(c, "C", []Doc{lanc("x", "2026-11-01T02:30:00.000Z", "CREDITO", "saida", 1, nil)}, Doc{})
	if rs[0].Dia != "2026-10-31" {
		t.Errorf("02:30 UTC de 01/11 e 31/10 em Sao Paulo, veio %s", rs[0].Dia)
	}
}

// F-T8: fechada e historica.
func TestEstadoDaFatura(t *testing.T) {
	c := cartaoTeste(12, nil)
	c.Inicio = "2026-08"
	if !FaturaFechada(c, "2026-09", "2026-10-13") || FaturaFechada(c, "2026-10", "2026-10-12") || !FaturaFechada(c, "2026-10", "2026-10-13") {
		t.Error("fechada so depois do dia do fechamento")
	}
	if !FaturaHistorica(c, "2026-07") || FaturaHistorica(c, "2026-08") {
		t.Error("historica so antes do inicio")
	}
	if FaturaHistorica(cartaoTeste(12, nil), "2000-01") {
		t.Error("sem inicio nada e historico")
	}
}

// F-T11: rotulos.
func TestRotuloFatura(t *testing.T) {
	if RotuloFatura(-1, "2026-09") != "Anterior" || RotuloFatura(0, "2026-10") != "Atual" || RotuloFatura(1, "2026-11") != "Próxima" || RotuloFatura(2, "2026-12") != "dez/26" || RotuloFatura(-2, "2026-08") != "ago/26" {
		t.Error("rotulos")
	}
	if RotuloMes("2027-01") != "jan/27" {
		t.Error("virada de ano")
	}
}

// F-T14: as sugestoes, os mesmos vetores do candidatosFatura do navegador.
func TestCandidatos(t *testing.T) {
	c := cartaoTeste(12, nil) // fatura 2026-10: 13/09 a 12/10
	itens := []Doc{
		lanc("p1", "2026-10-10T15:00:00.000Z", "CREDITO", "saida", 30.00, nil),
		lanc("p2", "2026-10-12T15:00:00.000Z", "CREDITO", "saida", 70.00, nil),
		lanc("p3", "2026-09-14T15:00:00.000Z", "CREDITO", "saida", 25.50, nil),
		lanc("p4", "2026-09-25T15:00:00.000Z", "CREDITO", "saida", 99.00, nil), // longe do corte
		lanc("v1", "2026-10-14T15:00:00.000Z", "CREDITO", "saida", 40.00, nil), // fatura 2026-11
	}
	rs := RegistrosDoCartao(c, "C", itens, Doc{})

	// o calculado e maior em 100: p1 + p2 passam para a proxima fatura
	sug := Candidatos(c, rs, "2026-10", 100)
	if len(sug) != 1 || len(sug[0].Registros) != 2 || sug[0].Rotulo != "Passar para a próxima fatura" {
		t.Fatalf("100: %+v", sug)
	}
	if sug[0].Acoes[0].Ajuste != 1 || sug[0].Acoes[1].Ajuste != 1 {
		t.Errorf("acoes = %+v", sug[0].Acoes)
	}
	// uma compra so (25.50) perto do inicio volta para a anterior
	sug = Candidatos(c, rs, "2026-10", 25.50)
	if len(sug) != 1 || sug[0].Registros[0].ID != "p3" || sug[0].Rotulo != "Trazer para a anterior" || sug[0].Acoes[0].Ajuste != -1 {
		t.Errorf("25.50: %+v", sug)
	}
	// o calculado e menor em 40: a compra v1, da proxima fatura, vem para esta
	sug = Candidatos(c, rs, "2026-10", -40)
	if len(sug) != 1 || sug[0].Registros[0].ID != "v1" || sug[0].Rotulo != "Trazer para esta fatura" || sug[0].Acoes[0].Ajuste != -1 {
		t.Errorf("-40: %+v", sug)
	}
	// longe do corte nao entra, e valor que ninguem explica nao tem sugestao
	if sug = Candidatos(c, rs, "2026-10", 99); len(sug) != 0 {
		t.Errorf("99 (longe do corte): %+v", sug)
	}
	if sug = Candidatos(c, rs, "2026-10", 1234.56); len(sug) != 0 {
		t.Errorf("sem combinacao: %+v", sug)
	}
	// tres compras: 10 + 20 + 30 = 60, todas perto do corte
	itens = []Doc{
		lanc("a", "2026-10-11T15:00:00.000Z", "CREDITO", "saida", 10, nil),
		lanc("b", "2026-10-12T15:00:00.000Z", "CREDITO", "saida", 20, nil),
		lanc("c2", "2026-10-10T15:00:00.000Z", "CREDITO", "saida", 30, nil),
	}
	rs = RegistrosDoCartao(c, "C", itens, Doc{})
	sug = Candidatos(c, rs, "2026-10", 60)
	if len(sug) != 1 || len(sug[0].Registros) != 3 {
		t.Errorf("tres compras: %+v", sug)
	}
	// o limite de 5 sugestoes: 8 compras de 10 perto do corte, alvo 10
	itens = nil
	for i := 0; i < 8; i++ {
		itens = append(itens, lanc(string(rune('a'+i)), "2026-10-12T15:00:00.000Z", "CREDITO", "saida", 10, nil))
	}
	rs = RegistrosDoCartao(c, "C", itens, Doc{})
	if sug = Candidatos(c, rs, "2026-10", 10); len(sug) != 5 {
		t.Errorf("limite de 5: %d", len(sug))
	}
}

func TestSomaDiasEDiferencaDeMeses(t *testing.T) {
	if SomaDias("2026-12-31", 1) != "2027-01-01" || SomaDias("2026-03-01", -1) != "2026-02-28" {
		t.Error("SomaDias")
	}
	if DiferencaDeMeses("2027-01", "2026-11") != 2 || DiferencaDeMeses("2026-10", "2026-10") != 0 || DiferencaDeMeses("2026-01", "2026-03") != -2 {
		t.Error("DiferencaDeMeses")
	}
}
