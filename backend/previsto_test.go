package main

import (
	"fmt"
	"strings"
	"testing"

	"financas/domain"
)

// Item 41, fase A (.ai/specs/futuros-previsto/spec.md): o marcador previsto, o catalogo de status, Futuros, confirmar e transferencias.
// Hoje e 24/10/2026 (12:00 em Sao Paulo): setembro ja fechou por data, outubro esta aberto.

func relogioPrev(t *testing.T) { t.Helper(); relogioEm(t, 2026, 10, 24, 15, 0) }

func corpo(data, status string, valor float64, extra string) string {
	return fmt.Sprintf(`{"dataEvento":%q,"status":%q,"tipo":"saida","valor":%v,"banco":"ITAU","descricao":"x"%s}`, data, status, valor, extra)
}

func gravado(t *testing.T, f *bancoFake, data, id string) doc {
	t.Helper()
	return f.ler(t, pkEspaco, dePrefixoTx+data+"#"+id)
}

const (
	hoje24   = "2026-10-24T15:00:00.000Z"
	amanha25 = "2026-10-25T15:00:00.000Z"
	ontem23  = "2026-10-23T15:00:00.000Z"
	nov05    = "2026-11-05T15:00:00.000Z"
	set10    = "2026-09-10T15:00:00.000Z"
)

// previsto grava direto no banco falso um previsto de ITAU.
func previsto(t *testing.T, f *bancoFake, id, data string, extra doc) {
	t.Helper()
	d := doc{"PK": pkEspaco, "SK": dePrefixoTx + data + "#" + id, "id": id, "dataEvento": data, "status": "PAGO", "tipo": "saida", "valor": 100.0,
		"banco": "ITAU", "descricao": "previsto " + id, "tags": []any{}, "criadoPor": "own", "previsto": true}
	for k, v := range extra {
		d[k] = v
	}
	f.colocar(t, d)
}

// P-T4, P-T5, P-T6 e P-T30: quem liga o marcador e o servidor.
func TestCriarLigaOMarcadorPelaDataEDescartaOQueVemDoCliente(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo(nov05, "PAGO", 10, ""))
	esperar(t, "data futura", c, 201)
	if d["previsto"] != true || gravado(t, f, nov05, d["id"].(string))["previsto"] != true {
		t.Errorf("data futura deve ligar o marcador: %v", d)
	}
	// hoje e o corpo com previsto: true, descartado
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo(hoje24, "PAGO", 10, `,"previsto":true`))
	esperar(t, "hoje", c, 201)
	if _, tem := d["previsto"]; tem {
		t.Errorf("o previsto do corpo e descartado: %v", d)
	}
	if _, tem := gravado(t, f, hoje24, d["id"].(string))["previsto"]; tem {
		t.Error("nada de previsto gravado para hoje")
	}
	// previsto: false numa data futura tambem e descartado, e o excluirDoTotal do cliente e recalculado
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo(nov05, "PAGO", 10, `,"previsto":false,"excluirDoTotal":true,"desfazer":{"versao":9}`))
	esperar(t, "forjar", c, 201)
	g := gravado(t, f, nov05, d["id"].(string))
	if g["previsto"] != true || g["excluirDoTotal"] != false {
		t.Errorf("o servidor decide: %v", g)
	}
}

// P-T6, P-T32 e P-T36: o status precisa ser selecionavel.
func TestCriarRecusaStatusDescontinuadoEInvalido(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	n := contaTx(f)
	for _, st := range []string{"PREVISTO", "TRANSFERINDO", "CREDITANDO", "CREDITO IN", "CREDITO EX", "CONTAS"} {
		c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo(hoje24, st, 1, ""))
		esperar(t, "criar com "+st, c, 400)
		if d["erro"] != "status descontinuado" {
			t.Errorf("%s: %v", st, d)
		}
	}
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo(hoje24, "FOO", 1, ""))
	esperar(t, "fora do catalogo", c, 400)
	if d["erro"] != "status invalido" {
		t.Errorf("FOO: %v", d)
	}
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo(hoje24, "pago", 1, ""))
	esperar(t, "texto diferente do catalogo", c, 400)
	if contaTx(f) != n {
		t.Error("nada pode ser gravado")
	}
	for _, ok := range []string{"PAGO", "CREDITO"} {
		c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo(hoje24, ok, 1, ""))
		esperar(t, "criar com "+ok, c, 201)
	}
	// o lote (backup) aceita qualquer status do catalogo
	lote := `{"itens":[{"id":"antigo00001","dataEvento":"2019-03-10T15:00:00.000Z","status":"CREDITO IN","tipo":"saida","valor":9,"banco":"CARTAO-A","descricao":"antigo"}]}`
	c, d = chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "lote", c, 200)
	if d["gravados"] != 1.0 {
		t.Errorf("o backup volta como era: %v", d)
	}
	lote = `{"itens":[{"id":"antigo00002","dataEvento":"2019-03-10T15:00:00.000Z","status":"FOO","tipo":"saida","valor":9,"banco":"CARTAO-A","descricao":"x"}]}`
	_, d = chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	if d["gravados"] != 0.0 || len(d["rejeitados"].([]any)) != 1 {
		t.Errorf("fora do catalogo e rejeitado: %v", d)
	}
}

// P-T7 a P-T11 e P-T39: o marcador na edicao.
func TestEditarMantemELigaOMarcador(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	// um antigo (status PREVISTO) e um previsto marcado
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + nov05 + "#antigo", "id": "antigo", "dataEvento": nov05, "status": "PREVISTO", "tipo": "saida",
		"valor": 50.0, "banco": "ITAU", "descricao": "antigo", "tags": []any{}, "criadoPor": "own"})
	previsto(t, f, "marcado", nov05, nil)
	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/antigo", map[string]string{"de": nov05}, `{"descricao":"outra"}`)
	esperar(t, "so a descricao", c, 200)
	if d["status"] != "PREVISTO" || d["previsto"] != true {
		t.Errorf("antigo segue previsto (derivado) e o status intacto: %v", d)
	}
	if _, tem := gravado(t, f, nov05, "antigo")["previsto"]; tem {
		t.Error("o derivado nao e gravado")
	}
	// trocar o status do antigo para PAGO (com a data ainda futura liga o marcador); levando a data para hoje deixa de ser previsto
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/antigo", map[string]string{"de": nov05}, `{"status":"PAGO","dataEvento":"`+hoje24+`"}`)
	esperar(t, "antigo para PAGO e para hoje", c, 200)
	if _, tem := d["previsto"]; tem {
		t.Errorf("deixa de ser previsto: %v", d)
	}
	// o marcado nao perde o marcador por mandar previsto: false, nem ao ir para ontem (continua previsto, agora atrasado)
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/marcado", map[string]string{"de": nov05}, `{"previsto":false,"dataEvento":"`+ontem23+`"}`)
	esperar(t, "marcado para ontem", c, 200)
	if d["previsto"] != true {
		t.Errorf("continua previsto: %v", d)
	}
	// um lancamento real que vai para o mes seguinte vira previsto
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo(hoje24, "PAGO", 10, ""))
	esperar(t, "real", c, 201)
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/"+d["id"].(string), map[string]string{"de": hoje24}, `{"dataEvento":"`+nov05+`"}`)
	esperar(t, "real para o mes seguinte", c, 200)
	if d["previsto"] != true {
		t.Errorf("vira previsto: %v", d)
	}
	// editar so o texto de um lancamento com status que nao e selecionavel: aceito, sem trava (P-T39)
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + set10 + "#credin", "id": "credin", "dataEvento": set10, "status": "CREDITO IN", "tipo": "saida",
		"valor": 20.0, "banco": "CARTAO-A", "descricao": "antigo", "tags": []any{}, "criadoPor": "own"})
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/credin", map[string]string{"de": set10}, `{"descricao":"novo texto"}`)
	esperar(t, "texto de CREDITO IN", c, 200)
	if d["status"] != "CREDITO IN" {
		t.Errorf("status intacto: %v", d)
	}
	// mudar o valor de um CREDITO IN em mes fechado: nao conta no saldo, entao nao passa pela trava (P-T37)
	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/credin", map[string]string{"de": set10}, `{"valor":25}`)
	esperar(t, "valor de CREDITO IN em mes fechado", c, 200)
	// mas trocar para um status descontinuado e recusado
	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/credin", map[string]string{"de": set10}, `{"status":"CREDITO EX"}`)
	esperar(t, "trocar para descontinuado", c, 400)
}

// P-T12 e P-T8: o lote.
func TestLoteLigaMantemENaoDesliga(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	lote := fmt.Sprintf(`{"itens":[
		{"id":"loteaaaaaa","dataEvento":%q,"status":"PAGO","tipo":"saida","valor":1,"banco":"ITAU","descricao":"futuro"},
		{"id":"lotebbbbbb","dataEvento":%q,"status":"PAGO","tipo":"saida","valor":1,"banco":"ITAU","descricao":"antigo marcado","previsto":true},
		{"id":"loteccccc1","dataEvento":%q,"status":"PAGO","tipo":"saida","valor":1,"banco":"ITAU","descricao":"comum","previsto":false}]}`, nov05, ontem23, ontem23)
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "lote", c, 200)
	if d["gravados"] != 3.0 {
		t.Fatalf("lote: %v", d)
	}
	if gravado(t, f, nov05, "loteaaaaaa")["previsto"] != true || gravado(t, f, ontem23, "lotebbbbbb")["previsto"] != true {
		t.Error("o futuro liga e o do backup mantem")
	}
	if _, tem := gravado(t, f, ontem23, "loteccccc1")["previsto"]; tem {
		t.Error("o comum nao ganha o marcador")
	}
}

// P-T13 e P-T41 a P-T43: GET /tx sem previstos, com rotulo, filtros e sem total.
func TestGetTxSemPrevistosComRotuloEFiltros(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	mk := func(id, data, status string, extra doc) {
		d := doc{"PK": pkEspaco, "SK": dePrefixoTx + data + "#" + id, "id": id, "dataEvento": data, "status": status, "tipo": "saida", "valor": 5.0,
			"banco": "ITAU", "descricao": id, "tags": []any{}, "criadoPor": "own"}
		for k, v := range extra {
			d[k] = v
		}
		f.colocar(t, d)
	}
	mk("a1", "2019-03-10T15:00:00.000Z", "CREDITO IN", nil)
	mk("a2", "2019-03-11T15:00:00.000Z", "CREDITO EX", nil)
	mk("a3", "2019-03-12T15:00:00.000Z", "PAGO", nil)
	mk("a4", "2019-03-13T15:00:00.000Z", "FOO", nil)
	mk("p1", "2019-03-14T15:00:00.000Z", "PREVISTO", nil)               // antigo: previsto pelo nome
	mk("p2", "2019-03-15T15:00:00.000Z", "PAGO", doc{"previsto": true}) // previsto pelo marcador
	c, d := chamar(t, "own", "GET", "spaces/s1/tx", map[string]string{"de": "2019-03-01", "ate": "2019-03-31", "filtros": "1"}, "")
	esperar(t, "periodo de 2019", c, 200)
	ids := idsDe(d)
	if strings.Contains(ids, "p1") || strings.Contains(ids, "p2") || len(itensDe(d)) != 4 {
		t.Errorf("sem previstos: %s", ids)
	}
	for _, it := range itensDe(d) {
		if it["id"] == "a1" && (it["rotulo"] != "CREDITO IN" || it["corPadrao"] != "#378ADD") {
			t.Errorf("rotulo e cor do catalogo: %v", it)
		}
		if it["id"] == "a4" && (it["rotulo"] != "FOO" || it["corPadrao"] != "#888780") {
			t.Errorf("fora do catalogo: o gravado e a cor neutra: %v", it)
		}
	}
	filtros := d["filtros"].(map[string]any)["status"].([]any)
	if fmt.Sprint(filtros) != "[PAGO CREDITO IN CREDITO EX FOO]" {
		t.Errorf("filtros do periodo (so o que aparece, na ordem do catalogo, o de fora no fim): %v", filtros)
	}
	if _, tem := d["resumo"]; tem {
		t.Error("sem resumo")
	}
	if _, tem := d["meses"]; tem {
		t.Error("sem meses")
	}
	// com o filtro de status ativo, a lista de filtros nao encolhe
	_, d = chamar(t, "own", "GET", "spaces/s1/tx", map[string]string{"de": "2019-03-01", "ate": "2019-03-31", "status": "PAGO", "filtros": "1"}, "")
	if idsDe(d) != "a3" || len(d["filtros"].(map[string]any)["status"].([]any)) != 4 {
		t.Errorf("status=PAGO: %v", d)
	}
	// um periodo recente nao traz os de 2019
	_, d = chamar(t, "own", "GET", "spaces/s1/tx", map[string]string{"de": "2026-10-01", "filtros": "1"}, "")
	for _, x := range d["filtros"].(map[string]any)["status"].([]any) {
		if x == "CREDITO IN" || x == "FOO" {
			t.Errorf("o periodo recente nao tem %v", x)
		}
	}
}

// P-T43: muitas paginas.
func TestGetTxComVariasPaginas(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	for i := 0; i < 1200; i++ {
		data := fmt.Sprintf("2025-05-%02dT%02d:%02d:00.000Z", 1+i%28, (i/28)%24, i%60)
		f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + data + fmt.Sprintf("#v%04d", i), "id": fmt.Sprintf("v%04d", i), "dataEvento": data, "status": "PAGO",
			"tipo": "saida", "valor": 1.0, "banco": "ITAU", "descricao": "v", "tags": []any{}})
	}
	total, cursor := 0, ""
	for pg := 0; pg < 10; pg++ {
		qs := map[string]string{"de": "2025-05-01", "ate": "2025-05-31", "limite": "500", "filtros": "1"}
		if cursor != "" {
			qs["cursor"] = cursor
		}
		_, d := chamar(t, "own", "GET", "spaces/s1/tx", qs, "")
		total += len(itensDe(d))
		if cursor == "" && d["filtros"] == nil {
			t.Error("a primeira pagina traz os filtros")
		}
		cursor, _ = d["proximo"].(string)
		if cursor == "" {
			break
		}
	}
	if total != 1200 {
		t.Errorf("paginas somam %d, esperado 1200", total)
	}
}

// P-T14 e P-T15: Futuros.
func TestFuturosGruposEResumo(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	previsto(t, f, "atrasado", set10, doc{"valor": 540.0})
	previsto(t, f, "hoje", hoje24, doc{"valor": 10.0})
	previsto(t, f, "novembro", nov05, doc{"valor": 7200.0, "tipo": "entrada"})
	previsto(t, f, "dezembro", "2026-12-05T15:00:00.000Z", doc{"valor": 30.0})
	previsto(t, f, "oculto", nov05, doc{"valor": 1000.0, "oculto": true})
	previsto(t, f, "excluido", nov05, doc{"valor": 2000.0, "excluidoEm": "2026-10-01T00:00:00.000Z"})
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + ontem23 + "#antigo", "id": "antigo", "dataEvento": ontem23, "status": "TRANSFERINDO", "tipo": "saida", "valor": 119.9,
		"banco": "ITAU", "descricao": "antigo", "tags": []any{}, "criadoPor": "own"})
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + hoje24 + "#comum", "id": "comum", "dataEvento": hoje24, "status": "PAGO", "tipo": "saida", "valor": 3.0,
		"banco": "ITAU", "descricao": "comum", "tags": []any{}, "criadoPor": "own"})
	c, d := chamar(t, "mem", "GET", "spaces/s1/futuros", nil, "")
	esperar(t, "futuros", c, 200)
	r := d["resumo"].(map[string]any)
	// pagar: 540 + 10 + 30 + 119,9 ; receber: 7200 ; atrasados: set10 e ontem (2) = 659,9
	if r["pagar"] != 699.9 || r["receber"] != 7200.0 || r["atrasados"].(map[string]any)["n"] != 2.0 || r["atrasados"].(map[string]any)["valor"] != 659.9 {
		t.Errorf("resumo: %v", r)
	}
	atras := d["atrasados"].([]any)
	if len(atras) != 2 || atras[0].(map[string]any)["id"] != "atrasado" || atras[1].(map[string]any)["id"] != "antigo" {
		t.Errorf("atrasados, mais antigos primeiro: %v", atras)
	}
	meses := d["meses"].([]any)
	if len(meses) != 3 || meses[0].(map[string]any)["mes"] != "2026-10" || meses[1].(map[string]any)["mes"] != "2026-11" || meses[2].(map[string]any)["mes"] != "2026-12" {
		t.Fatalf("meses: %v", meses)
	}
	nov := meses[1].(map[string]any)["itens"].([]any)
	if len(nov) != 1 || nov[0].(map[string]any)["id"] != "novembro" || nov[0].(map[string]any)["previsto"] != true || nov[0].(map[string]any)["rotulo"] != "PAGO" {
		t.Errorf("novembro (o oculto e o excluido ficam de fora): %v", nov)
	}
	if strings.Contains(fmt.Sprint(d), "comum") || d["truncado"] != false {
		t.Error("o lancamento comum nao e previsto")
	}
	// com ocultos=1 o oculto entra na lista e nao no resumo
	_, d = chamar(t, "own", "GET", "spaces/s1/futuros", map[string]string{"ocultos": "1"}, "")
	if !strings.Contains(fmt.Sprint(d["meses"]), "oculto") || d["resumo"].(map[string]any)["receber"] != 7200.0 || d["resumo"].(map[string]any)["pagar"] != 699.9 {
		t.Errorf("ocultos=1: %v", d)
	}
}

// Futuros so olha do mes passado em diante: um previsto mais antigo (como um PREVISTO esquecido de anos atras) nao entra, e nao e lido.
func TestFuturosIgnoraOsPrevistosAnterioresAoMesPassado(t *testing.T) {
	relogioPrev(t) // 24/10/2026: a janela comeca em 01/09/2026
	f := cenario(t)
	previsto(t, f, "agosto", "2026-08-31T15:00:00.000Z", nil)
	previsto(t, f, "velho", "2018-03-10T15:00:00.000Z", nil)
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2019-05-05T15:00:00.000Z#antigo", "id": "antigo", "dataEvento": "2019-05-05T15:00:00.000Z", "status": "PREVISTO",
		"tipo": "saida", "valor": 9.0, "banco": "ITAU", "descricao": "esquecido", "tags": []any{}})
	previsto(t, f, "setembro", "2026-09-01T15:00:00.000Z", nil)
	previsto(t, f, "outubro", hoje24, nil)
	_, d := chamar(t, "own", "GET", "spaces/s1/futuros", nil, "")
	ids := fmt.Sprint(d["atrasados"], d["meses"])
	for _, fora := range []string{"agosto", "velho", "antigo"} {
		if strings.Contains(ids, fora) {
			t.Errorf("%s e anterior a setembro e fica de fora: %s", fora, ids)
		}
	}
	if !strings.Contains(ids, "setembro") || !strings.Contains(ids, "outubro") {
		t.Errorf("do mes passado em diante entra: %s", ids)
	}
	if d["resumo"].(map[string]any)["atrasados"].(map[string]any)["n"] != 1.0 {
		t.Errorf("so setembro esta atrasado: %v", d["resumo"])
	}
}

// P-T16
func TestFuturosTruncaEm1000(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	for i := 0; i < 1100; i++ {
		data := fmt.Sprintf("2027-%02d-%02dT%02d:%02d:00.000Z", 1+i%12, 1+i%28, (i/28)%24, i%60)
		previsto(t, f, fmt.Sprintf("f%04d", i), data, nil)
	}
	_, d := chamar(t, "own", "GET", "spaces/s1/futuros", nil, "")
	n := 0
	for _, m := range d["meses"].([]any) {
		n += len(m.(map[string]any)["itens"].([]any))
	}
	if n != 1000 || d["truncado"] != true {
		t.Errorf("itens %d, truncado %v", n, d["truncado"])
	}
}

func confirmar(t *testing.T, quem, id, de, corpo string, extra map[string]string) (int, doc) {
	t.Helper()
	qs := map[string]string{"de": de}
	for k, v := range extra {
		qs[k] = v
	}
	return chamar(t, quem, "POST", "spaces/s1/tx/"+id+"/confirmar", qs, corpo)
}

// P-T17, P-T18, P-T19 e P-T23: confirmar, os 400, 404, 409 e o Desfazer.
func TestConfirmarPrevisto(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	previsto(t, f, "p1", nov05, doc{"valor": 548.9})
	for nome, c := range map[string]string{
		"valor zero":       `{"valor":0,"dataEvento":"` + hoje24 + `","status":"PAGO"}`,
		"data futura":      `{"valor":10,"dataEvento":"` + amanha25 + `","status":"PAGO"}`,
		"status invalido":  `{"valor":10,"dataEvento":"` + hoje24 + `","status":"FOO"}`,
		"status antigo":    `{"valor":10,"dataEvento":"` + hoje24 + `","status":"PREVISTO"}`,
		"CREDITO IN":       `{"valor":10,"dataEvento":"` + hoje24 + `","status":"CREDITO IN"}`,
		"corpo sem campos": `{}`,
	} {
		st, _ := confirmar(t, "own", "p1", nov05, c, nil)
		esperar(t, nome, st, 400)
	}
	if gravado(t, f, nov05, "p1")["previsto"] != true {
		t.Fatal("nada pode ser gravado")
	}
	st, d := confirmar(t, "own", "p1", nov05, `{"valor":548.9,"dataEvento":"`+hoje24+`","status":"PAGO"}`, nil)
	esperar(t, "confirmar", st, 200)
	if _, tem := d["previsto"]; tem || d["dataEvento"] != hoje24 || d["_seq"] == nil {
		t.Errorf("confirmado: %v", d)
	}
	desfazer, _ := d["desfazer"].(map[string]any)
	if gravado(t, f, nov05, "p1") != nil || gravado(t, f, hoje24, "p1") == nil {
		t.Error("a chave mudou com a data")
	}
	if _, tem := gravado(t, f, hoje24, "p1")["previsto"]; tem {
		t.Error("o marcador sai")
	}
	vs := versoesDe(t, f, "p1")
	confereVersao(t, vs[len(vs)-1], float64(len(vs)), "update", "own")
	// desfazer: restaurar a versao de antes volta previsto, com valor, data e status de antes
	n := int(desfazer["versao"].(float64))
	st, d = restaurar(t, "own", "p1", n)
	esperar(t, "desfazer", st, 200)
	if d["previsto"] != true || d["dataEvento"] != nov05 || d["valor"] != 548.9 {
		t.Errorf("voltou previsto: %v", d)
	}
	// 404 e 409
	st, _ = confirmar(t, "own", "naoexiste", nov05, `{"valor":1,"dataEvento":"`+hoje24+`","status":"PAGO"}`, nil)
	esperar(t, "inexistente", st, 404)
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + hoje24 + "#real", "id": "real", "dataEvento": hoje24, "status": "PAGO", "tipo": "saida", "valor": 5.0, "banco": "ITAU", "descricao": "r", "tags": []any{}, "criadoPor": "own"})
	st, d = confirmar(t, "own", "real", hoje24, `{"valor":1,"dataEvento":"`+hoje24+`","status":"PAGO"}`, nil)
	esperar(t, "nao e previsto", st, 409)
	if d["codigo"] != "nao_previsto" {
		t.Errorf("409: %v", d)
	}
	previsto(t, f, "ex", nov05, doc{"excluidoEm": "2026-10-01T00:00:00.000Z"})
	st, _ = confirmar(t, "own", "ex", nov05, `{"valor":1,"dataEvento":"`+hoje24+`","status":"PAGO"}`, nil)
	esperar(t, "excluido", st, 404)
}

// P-T20, P-T21 e P-T22: as travas rodam sobre o alvo novo.
func TestConfirmarRespeitaAsTravasPeloAlvoNovo(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	previsto(t, f, "setembro", set10, doc{"valor": 100.0})
	// manter a data de setembro (fechado por data) pede confirmacao
	corpoSet := `{"valor":100,"dataEvento":"` + set10 + `","status":"PAGO"}`
	st, d := confirmar(t, "own", "setembro", set10, corpoSet, nil)
	esperar(t, "mes fechado", st, 409)
	if d["codigo"] != "mes_fechado" {
		t.Errorf("409: %v", d)
	}
	st, _ = confirmar(t, "own", "setembro", set10, corpoSet, map[string]string{"confirmaMesFechado": "1"})
	esperar(t, "com a confirmacao", st, 200)
	vs := versoesDe(t, f, "setembro")
	if vs[len(vs)-1]["confirmouMesFechado"] != true {
		t.Errorf("a versao leva confirmouMesFechado: %v", vs[len(vs)-1])
	}
	// com a data de hoje nao pede nada (o efeito de antes era nulo e o de agora e em outubro, aberto)
	previsto(t, f, "setembro2", set10, doc{"valor": 100.0})
	st, _ = confirmar(t, "own", "setembro2", set10, `{"valor":100,"dataEvento":"`+hoje24+`","status":"PAGO"}`, nil)
	esperar(t, "data de hoje", st, 200)
	// mes conferido: recusa sempre, mesmo com a confirmacao
	ver, _ := f.ler(t, pkEspaco, "SALDO#ITAU")["version"].(float64)
	c, _ := conferir(t, "own", "ITAU", "2026-08", ver)
	esperar(t, "conferir agosto", c, 200)
	ago := "2026-08-10T15:00:00.000Z"
	previsto(t, f, "agosto", ago, nil)
	st, d = confirmar(t, "own", "agosto", ago, `{"valor":100,"dataEvento":"`+ago+`","status":"PAGO"}`, map[string]string{"confirmaMesFechado": "1"})
	esperar(t, "mes conferido", st, 409)
	if d["codigo"] != "mes_conferido" {
		t.Errorf("409: %v", d)
	}
}

// P-T25 e P-T26
func TestConfirmarPermissao(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	previsto(t, f, "meu", nov05, doc{"criadoPor": "mem"})
	previsto(t, f, "dele", nov05, doc{"criadoPor": "own"})
	body := `{"valor":10,"dataEvento":"` + hoje24 + `","status":"PAGO"}`
	st, _ := confirmar(t, "mem", "dele", nov05, body, nil)
	esperar(t, "member no de outra pessoa", st, 403)
	st, _ = confirmar(t, "mem", "meu", nov05, body, nil)
	esperar(t, "member no que criou", st, 200)
	st, _ = confirmar(t, "estranho", "dele", nov05, body, nil)
	esperar(t, "estranho", st, 403)
	st, _ = confirmar(t, "own", "dele", nov05, body, nil)
	esperar(t, "owner", st, 200)
}

// P-T28 e P-T29: a lista de status e do servidor.
func TestListasTemStatusDoServidorEIgnoraAsCaixas(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#LISTAS", "version": 4.0, "banco": []any{"ITAU"},
		"status":          []any{"PAGO", "PREVISTO", "CREDITO IN", "CREDITO EX", "CONTAS", "TRANSFERINDO", "CREDITANDO"},
		"statusFuturo":    map[string]any{"PREVISTO": true},
		"statusReal":      map[string]any{"PREVISTO": false},
		"afetaSaldoBanco": map[string]any{"PAGO": true}, "statusCor": map[string]any{"PAGO": "#111111"}})
	c, d := chamar(t, "mem", "GET", "spaces/s1/cfg/listas", nil, "")
	esperar(t, "ler", c, 200)
	if fmt.Sprint(d["status"]) != "[PAGO CREDITO]" {
		t.Errorf("status do servidor: %v", d["status"])
	}
	for _, k := range []string{"statusFuturo", "statusReal", "afetaSaldoBanco"} {
		if _, tem := d[k]; tem {
			t.Errorf("%s nao sai mais", k)
		}
	}
	if d["statusCor"] == nil {
		t.Error("statusCor continua")
	}
	// PUT com status e caixas alteradas: ignorados; o resto grava
	corpoPut := `{"version":4,"banco":["ITAU","NUBANK"],"status":["X"],"statusFuturo":{"X":true},"statusReal":{},"afetaSaldoBanco":{},"statusCor":{"PAGO":"#222222"}}`
	c, _ = chamar(t, "own", "PUT", "spaces/s1/cfg/listas", nil, corpoPut)
	esperar(t, "gravar", c, 200)
	g := f.ler(t, pkEspaco, "CFG#LISTAS")
	if fmt.Sprint(g["status"]) != "[PAGO PREVISTO CREDITO IN CREDITO EX CONTAS TRANSFERINDO CREDITANDO]" || g["statusFuturo"].(map[string]any)["PREVISTO"] != true {
		t.Errorf("o gravado fica como estava: %v", g)
	}
	if fmt.Sprint(g["banco"]) != "[ITAU NUBANK]" || g["statusCor"].(map[string]any)["PAGO"] != "#222222" {
		t.Errorf("o resto grava: %v", g)
	}
}

// P-T44: a lista sugerida para uma data.
func TestStatusSugeridoPorData(t *testing.T) {
	relogioPrev(t)
	cenario(t)
	c, d := chamar(t, "mem", "GET", "spaces/s1/status", map[string]string{"data": "2019-03-10T15:00:00.000Z"}, "")
	esperar(t, "status", c, 200)
	if fmt.Sprint(d["status"]) != "[PAGO CREDITO]" || d["todos"] != false {
		t.Errorf("2019: %v", d)
	}
	_, d = chamar(t, "mem", "GET", "spaces/s1/status", map[string]string{"todos": "1"}, "")
	if fmt.Sprint(d["status"]) != "[PAGO CREDITO]" || d["todos"] != true {
		t.Errorf("todos: %v", d)
	}
	// uma entrada com desde no futuro some da sugestao e aparece em todos=1
	antes := domain.Catalogo
	defer func() { domain.Catalogo = antes }()
	domain.Catalogo = append(append([]domain.EntradaStatus{}, antes...), domain.EntradaStatus{Nome: "PIX", Rotulo: "PIX", ContaNoSaldo: true, Selecionavel: true, Desde: "2027-01-01"})
	_, d = chamar(t, "mem", "GET", "spaces/s1/status", nil, "")
	if fmt.Sprint(d["status"]) != "[PAGO CREDITO]" {
		t.Errorf("PIX ainda nao vale: %v", d)
	}
	_, d = chamar(t, "mem", "GET", "spaces/s1/status", map[string]string{"data": "2027-02-01T15:00:00.000Z"}, "")
	if fmt.Sprint(d["status"]) != "[PAGO CREDITO PIX]" {
		t.Errorf("PIX vale em 2027: %v", d)
	}
	_, d = chamar(t, "mem", "GET", "spaces/s1/status", map[string]string{"todos": "1"}, "")
	if fmt.Sprint(d["status"]) != "[PAGO CREDITO PIX]" {
		t.Errorf("todos: %v", d)
	}
}

// P-T33 e P-T34: transferencia entre meios.
func TestTransferencia(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	n := contaTx(f)
	for nome, c := range map[string]string{
		"origem igual ao destino": `{"origem":"ITAU","destino":"ITAU","valor":10}`,
		"valor zero":              `{"origem":"ITAU","destino":"NUBANK","valor":0}`,
		"origem vazia":            `{"origem":"","destino":"NUBANK","valor":10}`,
		"sem destino":             `{"origem":"ITAU","valor":10}`,
	} {
		st, _ := chamar(t, "own", "POST", "spaces/s1/transferencias", nil, c)
		esperar(t, nome, st, 400)
	}
	if contaTx(f) != n {
		t.Fatal("nada pode ser gravado")
	}
	st, d := chamar(t, "own", "POST", "spaces/s1/transferencias", nil, `{"origem":"ITAU","destino":"NUBANK","valor":300,"dataEvento":"`+hoje24+`"}`)
	esperar(t, "transferir", st, 201)
	ids := d["ids"].([]any)
	if d["ok"] != true || len(ids) != 2 || d["transferParId"] == nil || d["_seq"] == nil || contaTx(f) != n+2 {
		t.Fatalf("resposta: %v", d)
	}
	a, b := gravado(t, f, hoje24, ids[0].(string)), gravado(t, f, hoje24, ids[1].(string))
	if a["banco"] != "ITAU" || a["tipo"] != "saida" || b["banco"] != "NUBANK" || b["tipo"] != "entrada" || a["transferParId"] != b["transferParId"] ||
		a["status"] != "CONTAS" || a["excluirDoTotal"] != true || b["excluirDoTotal"] != true || a["valor"] != 300.0 {
		t.Errorf("pontas: %v / %v", a, b)
	}
	// a trava de mes vale nos dois meios: setembro fechado pede confirmacao e nada e gravado
	n = contaTx(f)
	st, d = chamar(t, "own", "POST", "spaces/s1/transferencias", nil, `{"origem":"ITAU","destino":"NUBANK","valor":5,"dataEvento":"`+set10+`"}`)
	esperar(t, "mes fechado", st, 409)
	if contaTx(f) != n {
		t.Error("nada gravado com a trava")
	}
	st, _ = chamar(t, "own", "POST", "spaces/s1/transferencias", map[string]string{"confirmaMesFechado": "1"}, `{"origem":"ITAU","destino":"NUBANK","valor":5,"dataEvento":"`+set10+`"}`)
	esperar(t, "com a confirmacao", st, 201)
}

// P-T36 e P-T38: a trava depende do efeito no saldo.
func TestTravaPeloEfeitoNoSaldo(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	set := func(id, data string) {
		f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + data + "#" + id, "id": id, "dataEvento": data, "status": "PAGO", "tipo": "saida", "valor": 10.0,
			"banco": "ITAU", "descricao": id, "tags": []any{}, "criadoPor": "own"})
	}
	ago := "2026-08-10T15:00:00.000Z"
	set("fechado", set10)
	set("conferido", ago)
	set("antes", "2026-01-10T15:00:00.000Z")
	ver, _ := f.ler(t, pkEspaco, "SALDO#ITAU")["version"].(float64)
	c, _ := conferir(t, "own", "ITAU", "2026-08", ver)
	esperar(t, "conferir agosto", c, 200)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 1.0, "porBanco": map[string]any{"ITAU": map[string]any{"saldoInicial": 0.0, "dataInicio": "2026-04-01"}}})
	// trocar PAGO por CREDITO muda o efeito: conferido recusa, fechado por data pede confirmacao
	st, d := chamar(t, "own", "PUT", "spaces/s1/tx/conferido", map[string]string{"de": ago}, `{"status":"CREDITO"}`)
	esperar(t, "mes conferido", st, 409)
	if d["codigo"] != "mes_conferido" {
		t.Errorf("409: %v", d)
	}
	st, d = chamar(t, "own", "PUT", "spaces/s1/tx/fechado", map[string]string{"de": set10}, `{"status":"CREDITO"}`)
	esperar(t, "mes fechado", st, 409)
	if d["codigo"] != "mes_fechado" {
		t.Errorf("409: %v", d)
	}
}

// P-T46: reclassificar o passado entre status que contam no saldo nao toca na trava.
func TestReclassificarOPassadoNaoPassaPelaTrava(t *testing.T) {
	relogioPrev(t)
	antes := domain.Catalogo
	defer func() { domain.Catalogo = antes }()
	domain.Catalogo = append(append([]domain.EntradaStatus{}, antes...),
		domain.EntradaStatus{Nome: "PIX", Rotulo: "PIX", CorPadrao: "#111111", ContaNoSaldo: true, Selecionavel: true},
		domain.EntradaStatus{Nome: "DEBITO", Rotulo: "DEBITO", CorPadrao: "#222222", ContaNoSaldo: true, Selecionavel: true})
	f := cenario(t)
	mk := func(id, data string) {
		f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + data + "#" + id, "id": id, "dataEvento": data, "status": "PAGO", "tipo": "saida", "valor": 10.0,
			"banco": "ITAU", "descricao": id, "tags": []any{}, "criadoPor": "own"})
	}
	ago := "2026-08-10T15:00:00.000Z"
	antigo := "2018-03-10T15:00:00.000Z"
	mk("conf", ago)
	mk("fech", set10)
	mk("velho", antigo)
	ver, _ := f.ler(t, pkEspaco, "SALDO#ITAU")["version"].(float64)
	c, _ := conferir(t, "own", "ITAU", "2026-08", ver)
	esperar(t, "conferir", c, 200)
	for id, data := range map[string]string{"conf": ago, "fech": set10, "velho": antigo} {
		st, d := chamar(t, "own", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": data}, `{"status":"PIX"}`)
		esperar(t, "reclassificar "+id, st, 200)
		if d["status"] != "PIX" {
			t.Errorf("%s: %v", id, d)
		}
		vs := versoesDe(t, f, id)
		confereVersao(t, vs[len(vs)-1], float64(len(vs)), "update", "own")
	}
}

// Sem filtros=1, GET /tx nao faz a leitura extra do periodo para os filtros de status.
func TestGetTxSoCalculaFiltrosAPedido(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	_, d := chamar(t, "own", "GET", "spaces/s1/tx", nil, "")
	if _, tem := d["filtros"]; tem {
		t.Error("filtros so a pedido")
	}
	antes := f.lidosNoQuery
	chamar(t, "own", "GET", "spaces/s1/tx", map[string]string{"filtros": "1"}, "")
	umaLeitura := f.lidosNoQuery - antes
	antes = f.lidosNoQuery
	chamar(t, "own", "GET", "spaces/s1/tx", nil, "")
	if f.lidosNoQuery-antes >= umaLeitura {
		t.Errorf("sem filtros deveria ler menos: %d x %d", f.lidosNoQuery-antes, umaLeitura)
	}
}

// Futuros reaproveita a leitura enquanto o contador de alteracoes nao muda; qualquer escrita de lancamento faz ler de novo.
func TestFuturosReaproveitaALeituraAteAlguemGravar(t *testing.T) {
	relogioPrev(t)
	f := cenario(t)
	previsto(t, f, "p1", nov05, nil)
	c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo(hoje24, "PAGO", 1, "")) // uma escrita: o contador passa a existir
	esperar(t, "escrita", c, 201)
	chamar(t, "own", "GET", "spaces/s1/futuros", nil, "")
	antes := f.lidosNoQuery
	_, d := chamar(t, "own", "GET", "spaces/s1/futuros", nil, "")
	if f.lidosNoQuery != antes {
		t.Errorf("a segunda chamada nao deveria ler a particao (leu %d itens)", f.lidosNoQuery-antes)
	}
	if len(d["meses"].([]any)) != 1 {
		t.Errorf("mesmo resultado: %v", d)
	}
	// alguem grava um previsto novo: o contador muda e a leitura e refeita
	c, _ = chamar(t, "mem", "POST", "spaces/s1/tx", nil, corpo(nov05, "PAGO", 5, ""))
	esperar(t, "outra escrita", c, 201)
	_, d = chamar(t, "own", "GET", "spaces/s1/futuros", nil, "")
	if f.lidosNoQuery == antes || len(d["meses"].([]any)[0].(map[string]any)["itens"].([]any)) != 2 {
		t.Errorf("depois de uma escrita le de novo e traz o previsto novo: %v", d)
	}
	// o resultado do cache nao e alterado por quem chama (decorar mexe numa copia)
	_, d = chamar(t, "own", "GET", "spaces/s1/futuros", nil, "")
	_, d = chamar(t, "own", "GET", "spaces/s1/futuros", nil, "")
	if len(d["meses"].([]any)[0].(map[string]any)["itens"].([]any)) != 2 {
		t.Errorf("repetir nao muda nada: %v", d)
	}
}
