package main

import (
	"fmt"
	"testing"

	"financas/domain"
)

// Item 56 (.ai/specs/formas-de-pagamento/spec.md): PIX, DEBITO, DINHEIRO e TED entram no catalogo, em ordem de uso, com o PAGO no fim.
// Todas valem para entrada e saida e contam no saldo como o PAGO.

var formasNovas = []string{"PIX", "DEBITO", "DINHEIRO", "TED"}

func corpoTipo(data, status, tipo string, valor float64) string {
	return fmt.Sprintf(`{"dataEvento":%q,"status":%q,"tipo":%q,"valor":%v,"banco":"ITAU","descricao":"x"}`, data, status, tipo, valor)
}

// F-T1 e F-T2: a lista na ordem de uso, com o PIX como padrao e o PAGO no fim.
func TestFormasDePagamentoNaOrdemDeUso(t *testing.T) {
	relogioPrev(t)
	cenario(t)
	c, d := chamar(t, "mem", "GET", "spaces/s1/status", nil, "")
	esperar(t, "status", c, 200)
	if fmt.Sprint(d["status"]) != "[PIX DEBITO CREDITO DINHEIRO TED PAGO]" {
		t.Errorf("lista: %v", d["status"])
	}
	if lista := domain.Selecionaveis(); lista[0] != "PIX" || lista[len(lista)-1] != "PAGO" {
		t.Errorf("o padrao e o PIX e o PAGO fica no fim: %v", lista)
	}
}

// F-T3, F-T4 e F-T5: todas valem para saida e entrada (estorno no debito, estorno no cartao como entrada de CREDITO).
func TestFormasDePagamentoValemParaEntradaESaida(t *testing.T) {
	relogioPrev(t)
	cenario(t)
	for _, tipo := range []string{"saida", "entrada"} {
		for _, s := range append(append([]string{}, formasNovas...), "PAGO", "CREDITO") {
			c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTipo(hoje24, s, tipo, 10))
			esperar(t, tipo+" "+s, c, 201)
			if d["status"] != s || d["tipo"] != tipo {
				t.Errorf("%s %s: %v", tipo, s, d)
			}
		}
	}
}

// F-T6: o saldo de um meio com um lancamento de cada forma e o mesmo dos cinco como PAGO.
func TestFormasDePagamentoContamNoSaldoComoOPago(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#LISTAS", "banco": []any{"ITAU"}, "version": 1.0})
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 1.0, "porBanco": map[string]any{"ITAU": map[string]any{"saldoInicial": 100.0, "dataInicio": "2026-01-01"}}})
	for _, s := range append(append([]string{}, formasNovas...), "PAGO") {
		c, _ := chamar(t, "own", "POST", "spaces/s1/tx", conf, corpoTipo("2026-02-10T12:00:00.000Z", s, "saida", 10))
		esperar(t, s, c, 201)
	}
	c, _ := chamar(t, "own", "POST", "spaces/s1/tx", conf, corpoTipo("2026-02-11T12:00:00.000Z", "TED", "entrada", 30))
	esperar(t, "entrada por TED", c, 201)
	c, d := saldos(t, "own")
	esperar(t, "saldos", c, 200)
	if itau := itemDoMeio(t, d, "ITAU"); !quase(itau["saldo"].(float64), 50) {
		t.Errorf("100 - 30 (do cenario) - 5 x 10 + 30 = 50: %v", itau)
	}
}

// F-T7, F-T8 e F-T9: trocar uma forma por outra que conta igual nao passa pela trava; trocar para uma que nao conta, sim.
func TestReclassificarEntreFormasNaoPassaPelaTrava(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	mk := func(id, data, status string) {
		f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + data + "#" + id, "id": id, "dataEvento": data, "status": status, "tipo": "saida", "valor": 10.0,
			"banco": "ITAU", "descricao": id, "tags": []any{}, "criadoPor": "own"})
	}
	ago := "2026-08-10T15:00:00.000Z"
	mk("conf", ago, "PIX")
	mk("fech", set10, "PIX")
	ver, _ := f.ler(t, pkEspaco, "SALDO#ITAU")["version"].(float64)
	c, _ := conferir(t, "own", "ITAU", "2026-08", ver)
	esperar(t, "conferir", c, 200)
	st, d := chamar(t, "own", "PUT", "spaces/s1/tx/fech", map[string]string{"de": set10}, `{"status":"TED"}`)
	esperar(t, "PIX para TED em mes fechado, sem a confirmacao extra", st, 200)
	if d["status"] != "TED" {
		t.Errorf("fech: %v", d)
	}
	st, d = chamar(t, "own", "PUT", "spaces/s1/tx/conf", map[string]string{"de": ago}, `{"status":"DINHEIRO"}`)
	esperar(t, "PIX para DINHEIRO em mes conferido", st, 200)
	st, d = chamar(t, "own", "PUT", "spaces/s1/tx/conf", map[string]string{"de": ago}, `{"status":"CREDITO"}`)
	esperar(t, "DINHEIRO para CREDITO em mes conferido", st, 409)
	if d["codigo"] != "mes_conferido" {
		t.Errorf("409: %v", d)
	}
}

// F-T11: RECEBIMENTO e BOLETO nao fazem parte do catalogo.
func TestRecebimentoEBoletoNaoSaoFormasDePagamento(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	for _, s := range []string{"RECEBIMENTO", "BOLETO"} {
		c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTipo(hoje24, s, "entrada", 10))
		esperar(t, s, c, 400)
	}
	previsto(t, f, "p1", nov05, nil)
	st, _ := confirmar(t, "own", "p1", nov05, `{"valor":10,"dataEvento":"`+hoje24+`","status":"RECEBIMENTO"}`, nil)
	esperar(t, "confirmar com RECEBIMENTO", st, 400)
}

// F-T12 e F-T13: importar e confirmar aceitam as formas novas, e o PAGO antigo continua entrando.
func TestImportarEConfirmarComAsFormasNovas(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	lote := `{"itens":[` +
		`{"id":"imp0000001","dataEvento":"2026-03-05T10:00:00.000Z","status":"PAGO","tipo":"saida","valor":7,"banco":"ITAU","descricao":"a"},` +
		`{"id":"imp0000002","dataEvento":"2026-03-06T10:00:00.000Z","status":"PIX","tipo":"saida","valor":7,"banco":"ITAU","descricao":"b"}]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", conf, lote)
	esperar(t, "importar", c, 200)
	if d["gravados"] != 2.0 || len(d["rejeitados"].([]any)) != 0 {
		t.Errorf("importar: %v", d)
	}
	previsto(t, f, "p1", nov05, nil)
	st, d := confirmar(t, "own", "p1", nov05, `{"valor":10,"dataEvento":"`+hoje24+`","status":"TED"}`, nil)
	esperar(t, "confirmar com TED", st, 200)
	if d["status"] != "TED" {
		t.Errorf("confirmado: %v", d)
	}
}

// F-T14: rotulo e cor de cada forma, e o ajuste da conferencia continua PAGO (o generico), nunca o primeiro da lista.
func TestRotuloCorEAjusteDasFormas(t *testing.T) {
	cores := map[string]string{"PIX": "#1D9E75", "DEBITO": "#D4537E", "DINHEIRO": "#EF9F27", "TED": "#5DCAA5", "PAGO": "#639922"}
	for nome, cor := range cores {
		if r, c := domain.RotuloECor(nome); r != nome || c != cor {
			t.Errorf("%s: %q %q", nome, r, c)
		}
	}
	if domain.StatusDeAjuste() != "PAGO" {
		t.Errorf("o ajuste da conferencia e PAGO: %q", domain.StatusDeAjuste())
	}
}
