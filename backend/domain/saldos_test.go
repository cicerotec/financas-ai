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

// S-T1: o que conta no saldo.
func TestContaNoSaldo(t *testing.T) {
	semConfig := Doc{}
	comConfig := Doc{
		"afetaSaldoBanco": map[string]any{"PAGO": true, "CREDITO": false, "PIX": true},
		"statusFuturo":    map[string]any{"PREVISTO": true},
	}
	casos := []struct {
		nome   string
		l      Lancamento
		listas Doc
		quer   bool
	}{
		{"PAGO normal", tx("2026-01-10T12:00:00.000Z", "saida", 10, nil), semConfig, true},
		{"CONTAS conta pelo padrao", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"status": "CONTAS"}), semConfig, true},
		{"oculto", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"oculto": true}), semConfig, false},
		{"excluido", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"excluidoEm": "2026-01-11T00:00:00.000Z"}), semConfig, false},
		{"status que nao afeta o saldo", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"status": "CREDITO"}), comConfig, false},
		{"status fora da lista de afeta saldo", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"status": "OUTRO"}), comConfig, false},
		{"evento futuro mesmo afetando", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"status": "PREVISTO"}), Doc{
			"afetaSaldoBanco": map[string]any{"PREVISTO": true}, "statusFuturo": map[string]any{"PREVISTO": true}}, false},
		{"PREVISTO pelo padrao sem configuracao", tx("2026-01-10T12:00:00.000Z", "saida", 10, Doc{"status": "PREVISTO"}), semConfig, false},
		{"status configurado depois", tx("2026-01-10T12:00:00.000Z", "entrada", 10, Doc{"status": "PIX"}), comConfig, true},
	}
	for _, c := range casos {
		if got := ContaNoSaldo(c.l, c.listas); got != c.quer {
			t.Errorf("%s: ContaNoSaldo = %v, esperado %v", c.nome, got, c.quer)
		}
	}
}

func somar(t *testing.T, listas Doc, de *time.Time, ls ...Lancamento) map[string]float64 {
	t.Helper()
	a := NovoAcumulador(listas, de)
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
	base := BaseCache(cfg, Doc{})
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
	b := BaseCache(cfg, Doc{})
	if b != "v2|100.5|2026-01-01|CONTAS,PAGO/CREDITANDO,PREVISTO,TRANSFERINDO" {
		t.Errorf("base = %q", b)
	}
	if BaseCache(ConfigBanco{Existe: true, SaldoInicial: 100.5, DataInicio: "2026-01-01"}, Doc{"afetaSaldoBanco": map[string]any{"PAGO": true}}) == b {
		t.Error("mudar o que afeta o saldo muda a base")
	}
	if BaseCache(ConfigBanco{Existe: true, SaldoInicial: 100, DataInicio: "2026-01-01"}, Doc{}) == b {
		t.Error("mudar o saldo inicial muda a base")
	}
}

// S-T11: os doze ultimos meses e o estado de cada um.
func TestMesesDoPainel(t *testing.T) {
	mapa := map[string]float64{}
	for i, m := range []string{"2025-08", "2025-09", "2025-10", "2025-11", "2025-12", "2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06", "2026-07", "2026-08", "2026-09", "2026-10", "2026-11"} {
		mapa[m] = float64(i)
	}
	conf := map[string]Doc{"2026-03": {"saldo": 77.0, "real": 77.0, "ajuste": 0.0, "em": "2026-04-03T10:00:00.000Z", "por": "sub-1"}}
	ms := MesesDoPainel(mapa, conf, "2026-10")
	if len(ms) != 12 || ms[0].Mes != "2026-10" || ms[11].Mes != "2025-11" {
		t.Fatalf("12 meses do atual para tras, sem os futuros: %v", ms)
	}
	estado := map[string]string{}
	for _, m := range ms {
		estado[m.Mes] = m.Estado
	}
	if estado["2026-10"] != "aberto" || estado["2026-09"] != "calculado" || estado["2026-03"] != "conferido" || estado["2026-02"] != "travado" || estado["2026-04"] != "calculado" {
		t.Errorf("estados inesperados: %v", estado)
	}
	for _, m := range ms {
		if m.Mes == "2026-03" && (m.Valor != 77 || m.Conferido == nil || m.Conferido["real"] != 77.0) {
			t.Errorf("o mes conferido mostra o saldo conferido: %v", m)
		}
	}
}

// S-T24: o resultado com cache e conferido e o mesmo de calcular tudo, numa implementacao de referencia escrita aqui.
func TestEquivalenciaComReferencia(t *testing.T) {
	var ls []Lancamento
	valor := 13.17
	for m := 1; m <= 9; m++ {
		for d := 1; d <= 28; d += 9 {
			tipo := "saida"
			if (m+d)%3 == 0 {
				tipo = "entrada"
			}
			valor = math.Mod(valor*1.37+3.11, 400) + 0.01
			ls = append(ls, tx(time.Date(2026, time.Month(m), d, 14, 0, 0, 0, time.UTC).Format("2006-01-02T15:04:05.000Z"), tipo, math.Round(valor*100)/100, nil))
		}
	}
	cfg := ConfigBanco{Existe: true, SaldoInicial: 321.45, DataInicio: "2026-02-01"}
	atual := "2026-09"
	got := Cascata(somar(t, Doc{}, mustInicio(t, "2026-02-01"), ls...), cfg, nil, atual)

	// referencia: soma simples dia a dia, sem agrupar por mes de uma vez
	acc := cfg.SaldoInicial
	porMes := map[string]float64{}
	for _, l := range ls {
		d, _ := time.Parse(time.RFC3339Nano, l["dataEvento"].(string))
		if d.Before(*mustInicio(t, "2026-02-01")) {
			continue
		}
		v := l["valor"].(float64)
		if l["tipo"] == "saida" {
			v = -v
		}
		porMes[d.In(FusoApp).Format("2006-01")] += v
	}
	for m := 2; m <= 9; m++ {
		k := time.Date(2026, time.Month(m), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
		acc = math.Floor((acc+porMes[k])*100+0.5) / 100
		if m < 9 && !iguais(got.Fechamentos[k], acc) {
			t.Errorf("fechamento de %s: %v, referencia %v", k, got.Fechamentos[k], acc)
		}
	}
	if !iguais(got.Saldo, acc) {
		t.Errorf("saldo %v, referencia %v", got.Saldo, acc)
	}
}

func mustInicio(t *testing.T, s string) *time.Time {
	t.Helper()
	de, ok := InicioDoControle(s)
	if !ok {
		t.Fatalf("inicio invalido: %s", s)
	}
	return &de
}
