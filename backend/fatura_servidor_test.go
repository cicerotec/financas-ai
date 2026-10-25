package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Item 33a, fase A (.ai/specs/fatura-servidor/spec.md): as faturas do cartao calculadas e validadas no servidor.
// Hoje e 24/10/2026 (12:00 em Sao Paulo). O CARTAO-A fecha no dia 28: a fatura atual e 2026-10 (29/09 a 28/10).

func relogioCartao(t *testing.T) { t.Helper(); relogioEm(t, 2026, 10, 24, 15, 0) }

// compra grava um lancamento de cartao no banco falso.
func compra(t *testing.T, f *bancoFake, id, dia, status, tipo string, valor float64, extra doc) {
	t.Helper()
	data := dia + "T15:00:00.000Z"
	d := doc{"PK": pkEspaco, "SK": dePrefixoTx + data + "#" + id, "id": id, "dataEvento": data, "status": status, "tipo": tipo,
		"valor": valor, "banco": "CARTAO-A", "descricao": "compra " + id, "tags": []any{}, "criadoPor": "own"}
	for k, v := range extra {
		d[k] = v
	}
	f.colocar(t, d)
}

// cenarioCartao: CARTAO-A (fecha dia 28, controle desde 2026-01) e CARTAO-B sao cartoes; ITAU e BANCO-B nao. Fatura 2026-10 do CARTAO-A:
// 100 + 50,50 - 20 (estorno) + 5 (CREDITO IN, da fatura seguinte) = 135,50; a 2026-11 tem 10 (CREDITO EX) + 40 = 50.
func cenarioCartao(t *testing.T) *bancoFake {
	t.Helper()
	relogioCartao(t)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#LISTAS", "version": 1.0,
		"banco":  []any{"ITAU", "CARTAO-A", "CARTAO-B", "BANCO-B"},
		"status": []any{"PAGO", "CREDITO", "CREDITO EX", "CREDITO IN", "PREVISTO", "CREDITANDO"}})
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#CARTOES", "version": 3.0, "porBanco": map[string]any{
		"CARTAO-A": map[string]any{"diaPadrao": 28.0, "inicio": "2026-01", "faturas": map[string]any{}},
		"CARTAO-B": map[string]any{"diaPadrao": 5.0, "faturas": map[string]any{}},
	}})
	compra(t, f, "c1", "2026-09-30", "CREDITO", "saida", 100, nil)
	compra(t, f, "c2", "2026-10-10", "CREDITO", "saida", 50.50, nil)
	compra(t, f, "c3", "2026-10-15", "CREDITO", "entrada", 20, nil)
	compra(t, f, "c4", "2026-10-27", "CREDITO EX", "saida", 10, nil)
	compra(t, f, "c5", "2026-10-29", "CREDITO", "saida", 40, nil)
	compra(t, f, "c6", "2026-10-29", "CREDITO IN", "saida", 5, nil)
	compra(t, f, "c7", "2026-10-10", "CREDITO", "saida", 999, doc{"oculto": true})
	compra(t, f, "c8", "2026-10-10", "PAGO", "saida", 777, nil)
	compra(t, f, "c9", "2026-10-10", "CREDITANDO", "saida", 555, nil)
	// o BANCO-B tem compra no credito (sem ser cartao ainda); o ITAU nao
	compra(t, f, "b1", "2026-10-10", "CREDITO", "saida", 1, doc{"banco": "BANCO-B"})
	return f
}

func fatura(t *testing.T, quem, banco, ym string) (int, doc) {
	t.Helper()
	caminho := "spaces/s1/cartoes/" + banco + "/faturas"
	if ym != "" {
		caminho += "/" + ym
	}
	return chamar(t, quem, "GET", caminho, nil, "")
}

func numero(t *testing.T, d doc, campo string) float64 {
	t.Helper()
	n, ok := d[campo].(float64)
	if !ok {
		t.Fatalf("%s nao e numero: %v", campo, d)
	}
	return n
}

// F-T9
func TestCartoesListaAtualTotalEDisponiveis(t *testing.T) {
	cenarioCartao(t)
	c, d := chamar(t, "own", "GET", "spaces/s1/cartoes", nil, "")
	esperar(t, "listar", c, 200)
	cartoes := d["cartoes"].([]any)
	if len(cartoes) != 2 {
		t.Fatalf("cartoes: %v", d)
	}
	a := cartoes[0].(map[string]any)
	fa := a["fatura"].(map[string]any)
	if a["meio"] != "CARTAO-A" || fa["ym"] != "2026-10" || fa["fechamento"] != "2026-10-28" || fa["diasAteFechar"] != 4.0 || fa["total"] != 135.50 {
		t.Errorf("CARTAO-A: %v", a)
	}
	b := cartoes[1].(map[string]any)["fatura"].(map[string]any)
	if b["ym"] != "2026-11" || b["fechamento"] != "2026-11-05" || b["total"] != 0.0 { // dia 24 > 5: a fatura atual e a de novembro
		t.Errorf("CARTAO-B: %v", b)
	}
	if d["total"] != 135.50 || d["proximoFechamentoEmDias"] != 4.0 {
		t.Errorf("total e proximo fechamento: %v", d)
	}
	disp := d["disponiveis"].([]any)
	if len(disp) != 2 || disp[0].(map[string]any)["meio"] != "BANCO-B" || disp[0].(map[string]any)["temComprasNoCredito"] != true ||
		disp[1].(map[string]any)["meio"] != "ITAU" || disp[1].(map[string]any)["temComprasNoCredito"] != false {
		t.Errorf("disponiveis (os que tem compras no credito primeiro): %v", disp)
	}
	// o member tambem le
	c, _ = chamar(t, "mem", "GET", "spaces/s1/cartoes", nil, "")
	esperar(t, "member le", c, 200)
}

// F-T10 e F-T11
func TestFaturaAtualFuturaEChips(t *testing.T) {
	cenarioCartao(t)
	c, d := fatura(t, "own", "CARTAO-A", "")
	esperar(t, "sem ym", c, 200)
	if d["ym"] != "2026-10" || d["rotulo"] != "Atual" || d["total"] != 135.50 || d["fechada"] != false || d["historico"] != false || d["inicio"] != "2026-01" || d["incompleto"] != false {
		t.Errorf("fatura atual: %v", d)
	}
	per := d["periodo"].(map[string]any)
	if per["ini"] != "2026-09-29" || per["fim"] != "2026-10-28" {
		t.Errorf("periodo: %v", per)
	}
	if d["anterior"] != "2026-09" || d["proxima"] != "2026-11" {
		t.Errorf("anterior/proxima: %v", d)
	}
	chips := d["chips"].([]any)
	if len(chips) != 3 || chips[0].(map[string]any)["rotulo"] != "Anterior" || chips[1].(map[string]any)["rotulo"] != "Atual" || chips[2].(map[string]any)["rotulo"] != "Próxima" || chips[2].(map[string]any)["ym"] != "2026-11" {
		t.Errorf("chips: %v", chips)
	}
	// ym futuro e valido; os rotulos distantes sao mmm/aa e a virada de ano anda certo
	c, d = fatura(t, "own", "CARTAO-A", "2027-01")
	esperar(t, "ym futuro", c, 200)
	if d["rotulo"] != "jan/27" || d["anterior"] != "2026-12" || d["total"] != 0.0 {
		t.Errorf("fatura de janeiro: %v", d)
	}
	chips = d["chips"].([]any)
	if chips[0].(map[string]any)["rotulo"] != "dez/26" || chips[2].(map[string]any)["rotulo"] != "fev/27" {
		t.Errorf("chips da virada de ano: %v", chips)
	}
	c, _ = fatura(t, "own", "ITAU", "") // banco sem cartao
	esperar(t, "meio que nao e cartao", c, 404)
	c, _ = fatura(t, "own", "CARTAO-A", "2026-13")
	esperar(t, "ym invalido", c, 400)
}

// F-T8 e F-T12: fechada, historica, as compras da fatura e de onde vieram.
func TestFaturaCompras(t *testing.T) {
	cenarioCartao(t)
	_, d := fatura(t, "own", "CARTAO-A", "2026-10")
	itens := d["lancamentos"].([]any)
	por := map[string]map[string]any{}
	for _, x := range itens {
		m := x.(map[string]any)
		por[m["id"].(string)] = m
	}
	if _, tem := por["c5"]; tem {
		t.Errorf("c5 e da fatura 2026-11: %v", por["c5"])
	}
	if _, tem := por["c7"]; tem {
		t.Error("oculto nao entra")
	}
	if c4 := por["c4"]; c4 == nil || c4["origem"] != "foi" || c4["fatura"] != "2026-11" || c4["base"] != "2026-10" || c4["ajuste"] != 1.0 || c4["ocupaOutraFatura"] != "2026-11" {
		t.Errorf("c4 (CREDITO EX foi para a proxima): %v", c4)
	}
	if c6 := por["c6"]; c6 == nil || c6["origem"] != "veio" || c6["fatura"] != "2026-10" || c6["base"] != "2026-11" || c6["ajuste"] != -1.0 || c6["ocupaOutraFatura"] != "2026-11" {
		t.Errorf("c6 (CREDITO IN veio da proxima): %v", c6)
	}
	if c3 := por["c3"]; c3 == nil || c3["valor"] != -20.0 || c3["origem"] != nil || c3["dia"] != "2026-10-15" || c3["podeMexer"] != true {
		t.Errorf("c3 (estorno): %v", c3)
	}
	// o member so mexe no que criou (todas foram criadas por "own")
	_, d = fatura(t, "mem", "CARTAO-A", "2026-10")
	for _, x := range d["lancamentos"].([]any) {
		if x.(map[string]any)["podeMexer"] != false {
			t.Errorf("member nao mexe no que o dono criou: %v", x)
		}
	}
	// ordem por dia
	prev := ""
	for _, x := range d["lancamentos"].([]any) {
		dia := x.(map[string]any)["dia"].(string)
		if dia < prev {
			t.Error("lancamentos fora de ordem")
		}
		prev = dia
	}
	// fechada: a 2026-09 ja fechou; historico: antes do inicio (2026-01)
	_, d = fatura(t, "own", "CARTAO-A", "2026-09")
	if d["fechada"] != true || d["historico"] != false {
		t.Errorf("2026-09: %v", d)
	}
	_, d = fatura(t, "own", "CARTAO-A", "2025-12")
	if d["historico"] != true {
		t.Errorf("2025-12: %v", d)
	}
}

func putFatura(t *testing.T, ym, corpo string) (int, doc) {
	t.Helper()
	return chamar(t, "own", "PUT", "spaces/s1/cartoes/CARTAO-A/faturas/"+ym, nil, corpo)
}

func conferirFatura(t *testing.T, ym, corpo string) (int, doc) {
	t.Helper()
	return chamar(t, "own", "POST", "spaces/s1/cartoes/CARTAO-A/faturas/"+ym+"/conferir", nil, corpo)
}

func faturaGravada(t *testing.T, f *bancoFake, ym string) map[string]any {
	t.Helper()
	cfg := f.ler(t, pkEspaco, "CFG#CARTOES")
	a := cfg["porBanco"].(map[string]any)["CARTAO-A"].(map[string]any)
	fs, _ := a["faturas"].(map[string]any)
	m, _ := fs[ym].(map[string]any)
	return m
}

// F-T13, F-T15 e F-T18: conferir, mudou desde a conferencia e a diferenca.
func TestConferirFaturaEMudouDesdeAConferencia(t *testing.T) {
	f := cenarioCartao(t)
	// sem valorBanco: 409 com o total, e nada gravado
	c, d := conferirFatura(t, "2026-10", `{}`)
	esperar(t, "sem valor do banco", c, 409)
	if d["codigo"] != "diferenca" || d["total"] != 135.5 || d["valorBanco"] != nil {
		t.Errorf("409 sem valorBanco: %v", d)
	}
	if faturaGravada(t, f, "2026-10")["conferida"] != nil {
		t.Error("nada pode ser gravado")
	}
	// com valorBanco diferente: 409 com a diferenca
	c, _ = putFatura(t, "2026-10", `{"valorBanco":110}`)
	esperar(t, "valor do banco", c, 200)
	c, d = conferirFatura(t, "2026-10", `{}`)
	esperar(t, "com diferenca", c, 409)
	if d["total"] != 135.5 || d["valorBanco"] != 110.0 || d["diferenca"] != 25.5 {
		t.Errorf("409 com diferenca: %v", d)
	}
	// aceitando a diferenca, grava o total DO SERVIDOR
	c, d = conferirFatura(t, "2026-10", `{"aceitarDiferenca":true,"total":99999}`)
	esperar(t, "aceitar", c, 200)
	conf := faturaGravada(t, f, "2026-10")["conferida"].(map[string]any)
	if conf["total"] != 135.5 || conf["por"] != "own" || conf["em"] == nil {
		t.Errorf("conferida: %v", conf)
	}
	// batendo, grava sem pedir nada (a fatura 2026-11 tem 50)
	c, _ = putFatura(t, "2026-11", `{"valorBanco":50}`)
	esperar(t, "valor", c, 200)
	c, _ = conferirFatura(t, "2026-11", `{}`)
	esperar(t, "batendo", c, 200)
	// editar o valor de uma compra conferida: mudou desde a conferencia (e a contagem do cartao)
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2026-10-10T15:00:00.000Z#c2", "id": "c2", "dataEvento": "2026-10-10T15:00:00.000Z",
		"status": "CREDITO", "tipo": "saida", "valor": 60.50, "banco": "CARTAO-A", "descricao": "compra c2", "tags": []any{}, "criadoPor": "own"})
	_, d = fatura(t, "own", "CARTAO-A", "2026-10")
	if d["mudouDesdeConferencia"] != true || d["total"] != 145.50 {
		t.Errorf("mudou: %v", d)
	}
	_, d = chamar(t, "own", "GET", "spaces/s1/cartoes", nil, "")
	if d["cartoes"].([]any)[0].(map[string]any)["conferidasQueMudaram"] != 1.0 {
		t.Errorf("conferidasQueMudaram: %v", d["cartoes"])
	}
}

// F-T19: forjar total e conferida.
func TestConferirFaturaIgnoraTotalEConferidaDoCliente(t *testing.T) {
	f := cenarioCartao(t)
	putFatura(t, "2026-10", `{"valorBanco":135.5}`)
	c, _ := conferirFatura(t, "2026-10", `{"total":1,"conferida":{"total":1,"por":"outro"}}`)
	esperar(t, "conferir", c, 200)
	conf := faturaGravada(t, f, "2026-10")["conferida"].(map[string]any)
	if conf["total"] != 135.5 || conf["por"] != "own" {
		t.Errorf("o servidor grava o dele: %v", conf)
	}
	// o PUT generico de cfg/cartoes nao existe mais (fase C): nada do corpo e gravado
	cfg := f.ler(t, pkEspaco, "CFG#CARTOES")
	corpo := fmt.Sprintf(`{"version":%v,"porBanco":{"CARTAO-A":{"diaPadrao":28,"faturas":{"2026-08":{"conferida":{"total":7,"em":"y"}}}}}}`, cfg["version"])
	c, _ = chamar(t, "own", "PUT", "spaces/s1/cfg/cartoes", nil, corpo)
	esperar(t, "PUT generico", c, 404)
	a := f.ler(t, pkEspaco, "CFG#CARTOES")["porBanco"].(map[string]any)["CARTAO-A"].(map[string]any)["faturas"].(map[string]any)
	if _, tem := a["2026-08"]; tem || a["2026-10"].(map[string]any)["conferida"].(map[string]any)["total"] != 135.5 {
		t.Errorf("a configuracao nao pode mudar: %v", a)
	}
}

// F-T20
func TestDesconferirFatura(t *testing.T) {
	f := cenarioCartao(t)
	c, _ := chamar(t, "own", "DELETE", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10/conferir", nil, "")
	esperar(t, "sem conferida", c, 404)
	putFatura(t, "2026-10", `{"valorBanco":135.5}`)
	conferirFatura(t, "2026-10", `{}`)
	c, _ = chamar(t, "own", "DELETE", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10/conferir", nil, "")
	esperar(t, "desconferir", c, 200)
	if m := faturaGravada(t, f, "2026-10"); m["conferida"] != nil || m["valorBanco"] != 135.5 {
		t.Errorf("fica o valorBanco e sai a conferida: %v", m)
	}
}

// Conferir fatura de historico: 400.
func TestFaturaDeHistoricoNaoConfereNemMudaFechamento(t *testing.T) {
	f := cenarioCartao(t)
	c, _ := conferirFatura(t, "2025-12", `{"aceitarDiferenca":true}`)
	esperar(t, "historico nao confere", c, 400)
	c, _ = putFatura(t, "2025-12", `{"fechamento":"2025-12-20"}`)
	esperar(t, "historico nao aceita fechamento", c, 400)
	if faturaGravada(t, f, "2025-12") != nil {
		t.Error("nada gravado")
	}
	c, _ = putFatura(t, "2025-12", `{"valorBanco":50}`)
	esperar(t, "o valor do banco pode", c, 200)
}

// F-T15 e F-T21: sem sugestao e registrar a diferenca.
func TestRegistrarDiferencaDaFatura(t *testing.T) {
	f := cenarioCartao(t)
	relogioEm(t, 2026, 10, 30, 15, 0) // a fatura 2026-10 fechou no dia 28; total 135,50; o app diz 165,50: falta 30 e ninguem perto do corte explica
	putFatura(t, "2026-10", `{"valorBanco":165.5}`)
	_, d := fatura(t, "own", "CARTAO-A", "2026-10")
	if d["diferenca"] != -30.0 || d["podeRegistrarDiferenca"] != true || d["semSugestao"] == nil || len(d["sugestoes"].([]any)) != 0 {
		t.Errorf("sem sugestao: %v", d)
	}
	antes := contaTx(f)
	c, d := chamar(t, "own", "POST", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10/diferenca", nil, "")
	esperar(t, "registrar diferenca", c, 200)
	if contaTx(f) != antes+1 || d["total"] != 165.5 {
		t.Errorf("criou o lancamento e devolveu a fatura de novo: %v", d)
	}
	var novo doc
	for k := range f.itens {
		if strings.HasPrefix(k, pkEspaco+"|"+dePrefixoTx+"2026-10-28") {
			novo = f.ler(t, pkEspaco, strings.TrimPrefix(k, pkEspaco+"|"))
		}
	}
	if novo == nil || novo["valor"] != 30.0 || novo["tipo"] != "saida" || novo["status"] != "CREDITO" || novo["banco"] != "CARTAO-A" ||
		novo["faturaAjuste"] != 0.0 || novo["dataEvento"] != "2026-10-28T15:00:00.000Z" || novo["descricao"] != "Ajuste da fatura out/26 (juros, anuidade ou IOF)" {
		t.Errorf("lancamento da diferenca: %v", novo)
	}
	// agora bate: nao ha mais o que registrar
	c, _ = chamar(t, "own", "POST", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10/diferenca", nil, "")
	esperar(t, "zero", c, 400)
	// sem valorBanco
	c, _ = chamar(t, "own", "POST", "spaces/s1/cartoes/CARTAO-A/faturas/2026-08/diferenca", nil, "")
	esperar(t, "sem valorBanco", c, 400)
}

// F-T14: as sugestoes chegam com a acao e o rotulo, so com a fatura fechada.
func TestFaturaTrazSugestoes(t *testing.T) {
	f := cenarioCartao(t)
	// fatura 2026-09 (29/08 a 28/09, ja fechada): s1 de 40 em 27/09 e s2 de 25 em 29/09 (base 2026-10)
	compra(t, f, "s1", "2026-09-27", "CREDITO", "saida", 40, nil)
	compra(t, f, "s2", "2026-09-29", "CREDITO", "saida", 25, nil)
	putFatura(t, "2026-09", `{"valorBanco":0}`) // o calculado (40) e maior em 40: s1 passa para a proxima fatura
	_, d := fatura(t, "own", "CARTAO-A", "2026-09")
	sug := d["sugestoes"].([]any)
	if d["total"] != 40.0 || d["diferenca"] != 40.0 || len(sug) != 1 || d["semSugestao"] != nil || d["podeRegistrarDiferenca"] != false {
		t.Fatalf("calculado maior: %v", d)
	}
	s0 := sug[0].(map[string]any)
	acao := s0["acoes"].([]any)[0].(map[string]any)
	if s0["rotulo"] != "Passar para a próxima fatura" || acao["id"] != "s1" || acao["faturaAjuste"] != 1.0 {
		t.Errorf("sugestao: %v", s0)
	}
	// o app diz 65: o calculado e menor em 25 e s2 (da proxima fatura, 1 dia depois do corte) volta para esta
	putFatura(t, "2026-09", `{"valorBanco":65}`)
	_, d = fatura(t, "own", "CARTAO-A", "2026-09")
	sug = d["sugestoes"].([]any)
	if len(sug) != 1 {
		t.Fatalf("calculado menor: %v", d)
	}
	s0 = sug[0].(map[string]any)
	acao = s0["acoes"].([]any)[0].(map[string]any)
	if s0["rotulo"] != "Trazer para esta fatura" || acao["id"] != "s2" || acao["faturaAjuste"] != -1.0 {
		t.Errorf("sugestao: %v", s0)
	}
	// fatura aberta nao sugere
	putFatura(t, "2026-10", `{"valorBanco":35.5}`)
	_, d = fatura(t, "own", "CARTAO-A", "2026-10")
	if d["fechada"] != false || len(d["sugestoes"].([]any)) != 0 {
		t.Errorf("fatura aberta nao sugere: %v", d)
	}
}

// F-T16
func TestAdicionarCartaoValida(t *testing.T) {
	f := cenarioCartao(t)
	antes := f.ler(t, pkEspaco, "CFG#CARTOES")["version"]
	for nome, corpo := range map[string]string{
		"ja e cartao":    `{"meio":"CARTAO-A","diaPadrao":10}`,
		"fora da lista":  `{"meio":"NAO-EXISTE","diaPadrao":10}`,
		"dia 0":          `{"meio":"BANCO-B","diaPadrao":0}`,
		"dia 32":         `{"meio":"BANCO-B","diaPadrao":32}`,
		"dia quebrado":   `{"meio":"BANCO-B","diaPadrao":10.5}`,
		"sem dia":        `{"meio":"BANCO-B"}`,
		"sem meio":       `{"diaPadrao":10}`,
		"dia como texto": `{"meio":"BANCO-B","diaPadrao":"10"}`,
	} {
		c, _ := chamar(t, "own", "POST", "spaces/s1/cartoes", nil, corpo)
		esperar(t, nome, c, 400)
	}
	if f.ler(t, pkEspaco, "CFG#CARTOES")["version"] != antes {
		t.Error("nada pode ser gravado")
	}
	c, _ := chamar(t, "own", "POST", "spaces/s1/cartoes", nil, `{"meio":"BANCO-B","diaPadrao":10}`)
	esperar(t, "adicionar", c, 200)
	b := f.ler(t, pkEspaco, "CFG#CARTOES")["porBanco"].(map[string]any)["BANCO-B"].(map[string]any)
	if b["diaPadrao"] != 10.0 || b["faturas"] == nil {
		t.Errorf("cartao gravado: %v", b)
	}
	// aceita `banco` no lugar de `meio` (de-para na entrada)
	c, _ = chamar(t, "own", "POST", "spaces/s1/cartoes", nil, `{"banco":"ITAU","diaPadrao":3}`)
	esperar(t, "banco no lugar de meio", c, 200)
}

// R18 e R19: remover e o inicio do controle.
func TestRemoverCartaoEInicio(t *testing.T) {
	f := cenarioCartao(t)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/cartoes/CARTAO-B", nil, `{"inicio":"2026-5"}`)
	esperar(t, "inicio invalido", c, 400)
	c, _ = chamar(t, "own", "PUT", "spaces/s1/cartoes/CARTAO-B", nil, `{"inicio":"2026-05"}`)
	esperar(t, "inicio", c, 200)
	b := f.ler(t, pkEspaco, "CFG#CARTOES")["porBanco"].(map[string]any)["CARTAO-B"].(map[string]any)
	if b["inicio"] != "2026-05" {
		t.Errorf("inicio: %v", b)
	}
	c, _ = chamar(t, "own", "PUT", "spaces/s1/cartoes/CARTAO-B", nil, `{"inicio":null}`)
	esperar(t, "limpar", c, 200)
	b = f.ler(t, pkEspaco, "CFG#CARTOES")["porBanco"].(map[string]any)["CARTAO-B"].(map[string]any)
	if _, tem := b["inicio"]; tem {
		t.Errorf("inicio deveria sair: %v", b)
	}
	c, _ = chamar(t, "own", "PUT", "spaces/s1/cartoes/ITAU", nil, `{"inicio":"2026-05"}`)
	esperar(t, "meio que nao e cartao", c, 404)
	n := contaTx(f)
	c, _ = chamar(t, "own", "DELETE", "spaces/s1/cartoes/CARTAO-B", nil, "")
	esperar(t, "remover", c, 200)
	if _, tem := f.ler(t, pkEspaco, "CFG#CARTOES")["porBanco"].(map[string]any)["CARTAO-B"]; tem || contaTx(f) != n {
		t.Error("remove so a configuracao; os lancamentos ficam")
	}
	c, _ = chamar(t, "own", "DELETE", "spaces/s1/cartoes/CARTAO-B", nil, "")
	esperar(t, "remover de novo", c, 404)
}

// F-T17
func TestFechamentoEValorDaFaturaValidam(t *testing.T) {
	f := cenarioCartao(t)
	for nome, corpo := range map[string]string{
		"fechamento de outro mes": `{"fechamento":"2026-11-10"}`,
		"data inexistente":        `{"fechamento":"2026-10-32"}`,
		"formato errado":          `{"fechamento":"2026-10-3"}`,
		"valor negativo":          `{"valorBanco":-1}`,
		"valor como texto":        `{"valorBanco":"10"}`,
		"vazio":                   `{}`,
	} {
		c, _ := putFatura(t, "2026-10", corpo)
		esperar(t, nome, c, 400)
	}
	if faturaGravada(t, f, "2026-10") != nil {
		t.Error("nada pode ser gravado")
	}
	c, _ := putFatura(t, "2026-10", `{"fechamento":"2026-10-25","valorBanco":12.5}`)
	esperar(t, "gravar", c, 200)
	if m := faturaGravada(t, f, "2026-10"); m["fechamento"] != "2026-10-25" || m["valorBanco"] != 12.5 {
		t.Errorf("gravado: %v", m)
	}
	// o fechamento escolhido muda o periodo (e o total) da fatura
	_, d := fatura(t, "own", "CARTAO-A", "2026-10")
	if d["periodo"].(map[string]any)["fim"] != "2026-10-25" {
		t.Errorf("periodo: %v", d["periodo"])
	}
	c, _ = putFatura(t, "2026-10", `{"fechamento":null,"valorBanco":null}`)
	esperar(t, "null remove", c, 200)
	if faturaGravada(t, f, "2026-10") != nil {
		t.Error("a fatura vazia some da configuracao")
	}
}

// F-T22: o faturaAjuste so aceita -1, 0 e +1, e so vale em credito.
func TestFaturaAjusteValidaEmCriarEditarEImportar(t *testing.T) {
	f := cenarioCartao(t)
	tx := func(ajuste, status string) string {
		return fmt.Sprintf(`{"dataEvento":"2026-10-20T15:00:00.000Z","status":"%s","tipo":"saida","valor":9,"banco":"CARTAO-A","descricao":"x","faturaAjuste":%s}`, status, ajuste)
	}
	n0 := contaTx(f)
	for _, ruim := range []string{"2", `"1"`, "-2", "0.5"} {
		c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, tx(ruim, "CREDITO"))
		esperar(t, "criar com "+ruim, c, 400)
		c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/c2", map[string]string{"de": "2026-10-10T15:00:00.000Z"}, `{"faturaAjuste":`+ruim+`}`)
		esperar(t, "editar com "+ruim, c, 400)
		c, _ = chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, `{"itens":[`+tx(ruim, "CREDITO")+`]}`)
		if c == 200 && contaTx(f) != n0 {
			t.Errorf("o lote com faturaAjuste %s gravou algo", ruim)
		}
	}
	if contaTx(f) != n0 {
		t.Fatal("nenhum lancamento com faturaAjuste invalido pode ser gravado")
	}
	n := contaTx(f)
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, tx("null", "CREDITO"))
	esperar(t, "null vale como ausente", c, 201)
	id := d["id"].(string)
	g := f.ler(t, pkEspaco, dePrefixoTx+"2026-10-20T15:00:00.000Z#"+id)
	if _, tem := g["faturaAjuste"]; tem || contaTx(f) != n+1 {
		t.Errorf("faturaAjuste null nao e gravado: %v", g)
	}
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, tx("1", "PAGO"))
	esperar(t, "em lancamento que nao e de credito", c, 201)
	g = f.ler(t, pkEspaco, dePrefixoTx+"2026-10-20T15:00:00.000Z#"+d["id"].(string))
	if _, tem := g["faturaAjuste"]; tem {
		t.Errorf("faturaAjuste ignorado fora do credito: %v", g)
	}
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, tx("1", "CREDITO"))
	esperar(t, "em credito", c, 201)
	g = f.ler(t, pkEspaco, dePrefixoTx+"2026-10-20T15:00:00.000Z#"+d["id"].(string))
	if g["faturaAjuste"] != 1.0 {
		t.Errorf("faturaAjuste em credito e gravado: %v", g)
	}
}

// F-T23: a lista de meios pronta, com a marca de cartao.
func TestListasTrazMeiosComMarcaDeCartao(t *testing.T) {
	f := cenarioCartao(t)
	c, d := chamar(t, "mem", "GET", "spaces/s1/cfg/listas", nil, "")
	esperar(t, "ler listas", c, 200)
	meios := d["meios"].([]any)
	quer := []struct {
		nome   string
		cartao bool
	}{{"ITAU", false}, {"CARTAO-A", true}, {"CARTAO-B", true}, {"BANCO-B", false}}
	if len(meios) != len(quer) {
		t.Fatalf("meios: %v", meios)
	}
	for i, q := range quer {
		m := meios[i].(map[string]any)
		if m["nome"] != q.nome || m["cartao"] != q.cartao || m["ativo"] != true {
			t.Errorf("meio %d: %v", i, m)
		}
	}
	if cores, _ := d["statusCorPadrao"].(map[string]any); len(cores) != 8 || cores["PAGO"] != "#639922" || cores["CREDITO IN"] != "#378ADD" {
		t.Errorf("statusCorPadrao (todos os status do catalogo): %v", d["statusCorPadrao"])
	}
	fp, _ := d["faturaPadrao"].(map[string]any)
	if len(fp) != 3 || fp["CREDITO"] != 0.0 || fp["CREDITO EX"] != 1.0 || fp["CREDITO IN"] != -1.0 {
		t.Errorf("faturaPadrao (so os status de credito): %v", d["faturaPadrao"])
	}
	if b := d["banco"].([]any); len(b) != 4 || b[0] != "ITAU" {
		t.Errorf("banco (nomes) continua: %v", d["banco"])
	}
	// o que o front devolve no PUT nao grava `meios`
	v := d["version"].(float64)
	corpo := fmt.Sprintf(`{"version":%v,"banco":["ITAU","CARTAO-A"],"status":["PAGO"],"meios":[{"nome":"X","cartao":true,"ativo":true}],"faturaPadrao":{"X":1},"statusCorPadrao":{"X":"#000000"}}`, v)
	c, _ = chamar(t, "own", "PUT", "spaces/s1/cfg/listas", nil, corpo)
	esperar(t, "gravar listas", c, 200)
	if _, tem := f.ler(t, pkEspaco, "CFG#LISTAS")["meios"]; tem {
		t.Error("meios e derivado: nunca e gravado")
	}
	if _, tem := f.ler(t, pkEspaco, "CFG#LISTAS")["faturaPadrao"]; tem {
		t.Error("faturaPadrao e derivado: nunca e gravado")
	}
	if _, tem := f.ler(t, pkEspaco, "CFG#LISTAS")["statusCorPadrao"]; tem {
		t.Error("statusCorPadrao e derivado: nunca e gravado")
	}
}

// F-T24: os comandos so do dono.
func TestComandosDeCartaoSoDoDono(t *testing.T) {
	cenarioCartao(t)
	rotas := []struct{ m, caminho, corpo string }{
		{"POST", "spaces/s1/cartoes", `{"meio":"BANCO-B","diaPadrao":3}`},
		{"DELETE", "spaces/s1/cartoes/CARTAO-B", ""},
		{"PUT", "spaces/s1/cartoes/CARTAO-B", `{"inicio":null}`},
		{"PUT", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10", `{"valorBanco":1}`},
		{"POST", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10/conferir", `{}`},
		{"DELETE", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10/conferir", ""},
		{"POST", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10/diferenca", ""},
	}
	for _, r := range rotas {
		c, _ := chamar(t, "mem", r.m, r.caminho, nil, r.corpo)
		esperar(t, "member "+r.m+" "+r.caminho, c, 403)
		c, _ = chamar(t, "estranho", r.m, r.caminho, nil, r.corpo)
		esperar(t, "estranho "+r.m+" "+r.caminho, c, 403)
	}
	c, _ := chamar(t, "own", "GET", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10/conferir", nil, "")
	esperar(t, "rota que nao existe", c, 404)
}

// F-T25: se outra pessoa grava a configuracao no meio, o comando le de novo e grava sem perder nada.
func TestComandosDeCartaoRepetemNoConflito(t *testing.T) {
	f := cenarioCartao(t)
	vez := 0
	f.antesDoPut = func(in *dynamodb.PutItemInput) {
		vez++
		if vez == 1 { // outra pessoa grava antes: muda a version e acrescenta o inicio de CARTAO-B
			cfg := f.ler(t, pkEspaco, "CFG#CARTOES")
			cfg["porBanco"].(map[string]any)["CARTAO-B"].(map[string]any)["inicio"] = "2026-02"
			cfg["version"] = cfg["version"].(float64) + 1
			f.colocar(t, cfg)
		}
	}
	c, _ := chamar(t, "own", "POST", "spaces/s1/cartoes", nil, `{"meio":"BANCO-B","diaPadrao":7}`)
	esperar(t, "adicionar com conflito no meio", c, 200)
	por := f.ler(t, pkEspaco, "CFG#CARTOES")["porBanco"].(map[string]any)
	if por["BANCO-B"] == nil || por["CARTAO-B"].(map[string]any)["inicio"] != "2026-02" {
		t.Errorf("os dois ficam: %v", por)
	}
}
