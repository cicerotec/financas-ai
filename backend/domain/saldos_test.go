package domain

import (
	"math"
	"testing"
	"time"
)

// Testes do saldo calculado no servidor (item 40, specs/saldo-servidor): S-T1 a S-T8, S-T11 e S-T24.

func tx(data string, tipo string, valor float64, extra Doc) Lancamento {
	l := Lancamento{"dataEvento": data, "tipo": tipo, "valor": valor, "status": "PAGO", "banco": "ITAU"}
	for k, v := range extra {
		l[k] = v
	}
	return l
}

func iguais(a, b float64) bool { return math.Abs(a-b) < 0.0001 }

// S-T1: o que conta no saldo (pelo catalogo de status do servidor, item 41).
func TestContaNoSaldo(t *testing.T) {
	casos := []struct {
		nome string
		l    Lancamento
		quer bool
	}{
		{"PAGO normal", tx("2026-01-10T12:00:00.000Z", "saida", 10, nil), true},
		{"CONTAS conta", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"status": "CONTAS"}), true},
		{"oculto", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"oculto": true}), false},
		{"excluido", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"excluidoEm": "2026-01-11T00:00:00.000Z"}), false},
		{"CREDITO nao afeta o saldo", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"status": "CREDITO"}), false},
		{"status fora do catalogo", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"status": "OUTRO"}), false},
		{"PREVISTO (antigo)", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"status": "PREVISTO"}), false},
		{"previsto pelo marcador, mesmo PAGO", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"previsto": true}), false},
	}
	for _, c := range casos {
		if got := ContaNoSaldo(c.l); got != c.quer {
			t.Errorf("%s: ContaNoSaldo = %v, esperado %v", c.nome, got, c.quer)
		}
	}
}

func somar(t *testing.T, _ Doc, de *time.Time, ls ...Lancamento) map[string]float64 {
	t.Helper()
	a := NovoAcumulador(de)
	for _, l := range ls {
		a.Somar(l)
	}
	return a.PorMes()
}

// S-T2: cascata de tres meses com entradas e saidas e saldo inicial.
func TestCascataTresMeses(t *testing.T) {
	porMes := somar(t, Doc{}, nil,
		tx("2026-01-10T12:00:00.000Z", "entrada", 1000.10, nil),
		tx("2026-01-20T12:00:00.000Z", "saida", 250.05, nil),
		tx("2026-02-03T12:00:00.000Z", "saida", 400, nil),
		tx("2026-03-15T12:00:00.000Z", "entrada", 0.30, nil),
		tx("2026-03-16T12:00:00.000Z", "saida", 0.10, nil),
	)
	cfg := ConfigBanco{Existe: true, SaldoInicial: 100, DataInicio: "2026-01-01"}
	c := Cascata(porMes, cfg, nil, "2026-04")
	quer := map[string]float64{"2026-01": 850.05, "2026-02": 450.05, "2026-03": 450.25, "2026-04": 450.25}
	for m, v := range quer {
		if got, ok := c.Fechamentos[m]; m != "2026-04" && (!ok || !iguais(got, v)) {
			t.Errorf("fechamento de %s = %v, esperado %v", m, got, v)
		}
	}
	if !iguais(c.Saldo, 450.25) {
		t.Errorf("saldo final = %v, esperado 450.25", c.Saldo)
	}
	if _, ok := c.Fechamentos["2026-04"]; ok {
		t.Error("o mes atual nunca entra nos fechamentos de cache")
	}
	if len(c.Meses) != 4 || c.Meses[3].Mes != "2026-04" {
		t.Errorf("a cascata deve ir do inicio ate o mes atual: %v", c.Meses)
	}
}

// S-T3: a data de inicio vale a partir de 00:00 em Sao Paulo.
func TestDataDeInicioEmSaoPaulo(t *testing.T) {
	de, ok := InicioDoControle("2026-01-05")
	if !ok {
		t.Fatal("data de inicio valida deveria dar um instante")
	}
	// 00:00 de 05/01 em Sao Paulo = 03:00 UTC
	if got := de.UTC().Format(time.RFC3339); got != "2026-01-05T03:00:00Z" {
		t.Fatalf("inicio em UTC = %s", got)
	}
	porMes := somar(t, Doc{}, &de,
		tx("2026-01-05T02:59:59.999Z", "saida", 7, nil), // 23:59 de 04/01: historico
		tx("2026-01-05T03:00:00.000Z", "saida", 5, nil), // 00:00 de 05/01: conta
	)
	if !iguais(porMes["2026-01"], -5) {
		t.Errorf("so o de 00:00 conta: %v", porMes)
	}
	if _, ok := InicioDoControle("lixo"); ok {
		t.Error("data de inicio invalida nao vale")
	}
}

// S-T4: o mes e o de Sao Paulo, nao o de UTC.
func TestMesPelaDataDeSaoPaulo(t *testing.T) {
	porMes := somar(t, Doc{}, nil, tx("2026-11-01T02:30:00.000Z", "saida", 3, nil)) // 23:30 de 31/10 em SP
	if !iguais(porMes["2026-10"], -3) || len(porMes) != 1 {
		t.Errorf("deveria contar em outubro: %v", porMes)
	}
}

// S-T5 e S-T6: mes sem movimento repete o anterior; mes futuro com movimento entra no saldo atual.
func TestCascataMesSemMovimentoEMesFuturo(t *testing.T) {
	porMes := map[string]float64{"2026-01": 10, "2026-04": -3}
	cfg := ConfigBanco{Existe: true, SaldoInicial: 0, DataInicio: "2026-01-01"}
	c := Cascata(porMes, cfg, nil, "2026-03")
	if !iguais(c.Fechamentos["2026-02"], 10) {
		t.Errorf("fevereiro sem movimento repete janeiro: %v", c.Fechamentos)
	}
	if !iguais(c.Saldo, 7) || c.Meses[len(c.Meses)-1].Mes != "2026-04" {
		t.Errorf("a cascata vai ate o ultimo mes com movimento e o saldo o inclui: saldo=%v meses=%v", c.Saldo, c.Meses)
	}
}

// S-T5 (sem dataInicio): comeca no primeiro movimento e nao grava fechamento nenhum.
func TestCascataSemDataDeInicio(t *testing.T) {
	porMes := map[string]float64{"2026-02": 5, "2026-03": 6}
	c := Cascata(porMes, ConfigBanco{}, nil, "2026-03")
	if !iguais(c.Saldo, 11) || len(c.Fechamentos) != 0 || c.Meses[0].Mes != "2026-02" {
		t.Errorf("sem dataInicio: saldo=%v fechamentos=%v meses=%v", c.Saldo, c.Fechamentos, c.Meses)
	}
}

// S-T7: o ponto de partida (conferido ou cache valido) nao muda o resultado.
func TestPontoDePartidaNaoMudaOResultado(t *testing.T) {
	ls := []Lancamento{
		tx("2026-01-10T12:00:00.000Z", "entrada", 500, nil),
		tx("2026-02-10T12:00:00.000Z", "saida", 120.5, nil),
		tx("2026-03-10T12:00:00.000Z", "saida", 80.25, nil),
		tx("2026-04-10T12:00:00.000Z", "entrada", 10, nil),
	}
	cfg := ConfigBanco{Existe: true, SaldoInicial: 50, DataInicio: "2026-01-01"}
	atual := "2026-04"
	cheio := Cascata(somar(t, Doc{}, nil, ls...), cfg, nil, atual)

	// parte do fechamento de fevereiro (conferido): so os lancamentos de marco em diante
	de, _ := LimiteAPartirDe(cfg, &Partida{Mes: "2026-02", Saldo: cheio.Fechamentos["2026-02"]})
	parcial := Cascata(somar(t, Doc{}, &de, ls...), cfg, &Partida{Mes: "2026-02", Saldo: cheio.Fechamentos["2026-02"]}, atual)
	if !iguais(parcial.Saldo, cheio.Saldo) {
		t.Errorf("com ponto de partida o saldo deveria ser igual: %v x %v", parcial.Saldo, cheio.Saldo)
	}
	if !iguais(parcial.Fechamentos["2026-03"], cheio.Fechamentos["2026-03"]) {
		t.Errorf("fechamento de marco difere: %v x %v", parcial.Fechamentos, cheio.Fechamentos)
	}
}

func TestCachesValidosEPontoDePartida(t *testing.T) {
	cfg := ConfigBanco{Existe: true, SaldoInicial: 0, DataInicio: "2026-02-01"}
	base := BaseCache(cfg)
	item := Doc{
		"base":       base,
		"invalidoDe": "2026-05",
		"caches":     map[string]any{"2026-01": 1.0, "2026-02": 2.0, "2026-03": 3.0, "2026-05": 5.0, "2026-06": 6.0, "2026-07": 7.0},
		"conferidos": map[string]any{"2026-03": map[string]any{"saldo": 33.0, "real": 33.0}},
	}
	c := CachesValidos(item, cfg, base, "2026-07")
	// 2026-01 e anterior ao inicio; 2026-05 em diante e invalido; so 02 e 03
	if len(c) != 2 || c["2026-02"] != 2 || c["2026-03"] != 3 {
		t.Errorf("caches validos inesperados: %v", c)
	}
	p, ok := PontoDePartida(item, cfg, base, "2026-07")
	if !ok || p.Mes != "2026-03" || p.Saldo != 33 {
		t.Errorf("o conferido vence o cache do mesmo mes: %v %v", p, ok)
	}
	// base diferente: nenhum cache vale, mas o conferido continua valendo
	if c := CachesValidos(item, cfg, "outra", "2026-07"); len(c) != 0 {
		t.Errorf("base diferente invalida os caches: %v", c)
	}
	// cache do mes atual nunca vale
	if c := CachesValidos(Doc{"base": base, "caches": map[string]any{"2026-07": 9.0}}, cfg, base, "2026-07"); len(c) != 0 {
		t.Errorf("o mes atual nunca e cache: %v", c)
	}
	// sem dataInicio nenhum cache vale
	if c := CachesValidos(item, ConfigBanco{Existe: true}, base, "2026-07"); len(c) != 0 {
		t.Errorf("sem dataInicio nao ha cache: %v", c)
	}
}

// S-T8: a base do cache muda com a configuracao e a versao antiga nao casa.
func TestBaseDoCache(t *testing.T) {
	cfg := ConfigBanco{Existe: true, SaldoInicial: 100.5, DataInicio: "2026-01-01"}
	b := BaseCache(cfg)
	if b != "v3|100.5|2026-01-01|PAGO,CONTAS" {
		t.Errorf("base = %q", b)
	}
	if BaseCache(ConfigBanco{Existe: true, SaldoInicial: 100, DataInicio: "2026-01-01"}) == b {
		t.Error("mudar o saldo inicial muda a base")
	}
	// mudar o catalogo (um status passa a contar no saldo) muda a base: o cache antigo nao vale
	antes := Catalogo
	defer func() { Catalogo = antes }()
	Catalogo = append(append([]EntradaStatus{}, antes...), EntradaStatus{Nome: "PIX", Rotulo: "PIX", ContaNoSaldo: true, Selecionavel: true})
	if BaseCache(cfg) == b {
		t.Error("mudar o que conta no saldo muda a base")
	}
}
