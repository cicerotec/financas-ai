package main

import (
	"fmt"
	"strings"
	"testing"
)

// Itens 33b e 35, fase A (.ai/specs/pagar-fatura/spec.md): pagador, vencimento, a marca pagaFatura, as faturas em Futuros e a trava da fatura conferida.
// Hoje e 24/10/2026 (cenarioCartao): o CARTAO-A fecha no dia 28; ITAU e BANCO-B nao sao cartoes.

func putCartao(t *testing.T, meio, corpo string) (int, doc) {
	t.Helper()
	return chamar(t, "own", "PUT", "spaces/s1/cartoes/"+meio, nil, corpo)
}

func cartaoGravado(t *testing.T, f *bancoFake, meio string) map[string]any {
	t.Helper()
	return f.ler(t, pkEspaco, "CFG#CARTOES")["porBanco"].(map[string]any)[meio].(map[string]any)
}

func pagamento(banco string, valor float64, data, marca string) string {
	return fmt.Sprintf(`{"dataEvento":%q,"status":"PAGO","tipo":"saida","valor":%v,"banco":%q,"descricao":"Pagamento da fatura"%s}`, data, valor, banco, marca)
}

const marcaSet = `,"pagaFatura":{"meio":"CARTAO-A","ym":"2026-09"}`

// G-T4: pagador e dia do vencimento do cartao.
func TestCartaoPagadorEDiaDeVencimento(t *testing.T) {
	f := cenarioCartao(t)
	for nome, corpo := range map[string]string{
		"pagador fora da lista":   `{"pagador":"NAO-EXISTE"}`,
		"pagador que nao e texto": `{"pagador":5}`,
		"dia 0":                   `{"diaVencimento":0}`,
		"dia 32":                  `{"diaVencimento":32}`,
		"dia quebrado":            `{"diaVencimento":5.5}`,
		"dia como texto":          `{"diaVencimento":"5"}`,
	} {
		c, _ := putCartao(t, "CARTAO-A", corpo)
		esperar(t, nome, c, 400)
	}
	if _, tem := cartaoGravado(t, f, "CARTAO-A")["pagador"]; tem {
		t.Fatal("nada pode ser gravado")
	}
	c, _ := putCartao(t, "CARTAO-A", `{"pagador":"BANCO-A","diaVencimento":5}`)
	esperar(t, "pagador fora da lista (BANCO-A nao existe)", c, 400)
	c, _ = putCartao(t, "CARTAO-A", `{"pagador":"BANCO-B","diaVencimento":5}`)
	esperar(t, "gravar", c, 200)
	if g := cartaoGravado(t, f, "CARTAO-A"); g["pagador"] != "BANCO-B" || g["diaVencimento"] != 5.0 || g["inicio"] != "2026-01" {
		t.Errorf("gravado (o inicio fica): %v", g)
	}
	c, _ = putCartao(t, "CARTAO-A", `{"pagador":null,"diaVencimento":null}`)
	esperar(t, "null remove", c, 200)
	if g := cartaoGravado(t, f, "CARTAO-A"); g["pagador"] != nil || g["diaVencimento"] != nil {
		t.Errorf("campos removidos: %v", g)
	}
	c, _ = putCartao(t, "CARTAO-A", `{}`)
	esperar(t, "sem nada para mudar", c, 400)
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/cartoes/CARTAO-A", nil, `{"diaVencimento":5}`)
	esperar(t, "member nao configura", c, 403)
}

// G-T5: pagador e vencimento de uma fatura.
func TestFaturaPagadorEVencimento(t *testing.T) {
	f := cenarioCartao(t)
	for nome, corpo := range map[string]string{
		"vencimento antes do fechamento":  `{"vencimento":"2026-10-20"}`,
		"vencimento no dia do fechamento": `{"vencimento":"2026-10-28"}`,
		"46 dias depois":                  `{"vencimento":"2026-12-13"}`,
		"data inexistente":                `{"vencimento":"2026-11-31"}`,
		"vencimento como numero":          `{"vencimento":5}`,
		"pagador fora da lista":           `{"pagador":"NAO-EXISTE"}`,
	} {
		c, _ := putFatura(t, "2026-10", corpo)
		esperar(t, nome, c, 400)
	}
	if faturaGravada(t, f, "2026-10") != nil {
		t.Fatal("nada pode ser gravado")
	}
	c, _ := putFatura(t, "2026-10", `{"vencimento":"2026-11-05","pagador":"BANCO-B"}`)
	esperar(t, "valido", c, 200)
	if m := faturaGravada(t, f, "2026-10"); m["vencimento"] != "2026-11-05" || m["pagador"] != "BANCO-B" {
		t.Errorf("gravado: %v", m)
	}
	c, _ = putFatura(t, "2026-10", `{"vencimento":null,"pagador":null}`)
	esperar(t, "null remove", c, 200)
	if faturaGravada(t, f, "2026-10") != nil {
		t.Error("a fatura vazia some da configuracao")
	}
	c, _ = putFatura(t, "2025-12", `{"vencimento":"2026-01-05"}`)
	esperar(t, "fatura de historico nao aceita vencimento", c, 400)
	c, _ = putFatura(t, "2025-12", `{"valorBanco":10}`)
	esperar(t, "mas aceita o valor do banco", c, 200)
}

// G-T6: a marca precisa ser bem formada, em POST, PUT e lote.
func TestMarcaPagaFaturaInvalida(t *testing.T) {
	f := cenarioCartao(t)
	n := contaTx(f)
	ruins := map[string]string{
		"meio que nao e cartao": `,"pagaFatura":{"meio":"BANCO-B","ym":"2026-09"}`,
		"mes invalido":          `,"pagaFatura":{"meio":"CARTAO-A","ym":"2026-13"}`,
		"sem o mes":             `,"pagaFatura":{"meio":"CARTAO-A"}`,
		"campo a mais":          `,"pagaFatura":{"meio":"CARTAO-A","ym":"2026-09","x":1}`,
		"nao e objeto":          `,"pagaFatura":"CARTAO-A"`,
	}
	for nome, marca := range ruins {
		c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 10, hoje24, marca))
		esperar(t, "POST com "+nome, c, 400)
		c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, "{"+strings.TrimPrefix(marca, ",")+"}")
		esperar(t, "PUT com "+nome, c, 400)
	}
	c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, `{"dataEvento":"`+hoje24+`","status":"CREDITO","tipo":"saida","valor":10,"banco":"ITAU","descricao":"x"`+marcaSet+`}`)
	esperar(t, "compra de credito nao paga fatura", c, 400)
	lote := `{"itens":[{"id":"marcarui001","dataEvento":"` + hoje24 + `","status":"PAGO","tipo":"saida","valor":10,"banco":"ITAU","descricao":"x","pagaFatura":{"meio":"BANCO-B","ym":"2026-09"}}]}`
	_, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	if d["gravados"] != 0.0 || len(d["rejeitados"].([]any)) != 1 {
		t.Errorf("o lote rejeita a marca ruim: %v", d)
	}
	if contaTx(f) != n {
		t.Error("nada pode ser gravado")
	}
}

// G-T7 e G-T8: a marca e uma label; nao recusa nada e so muda o excluirDoTotal.
func TestMarcaPagaFaturaNaoFazRegra(t *testing.T) {
	f := cenarioCartao(t)
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 600, hoje24, marcaSet))
	esperar(t, "primeiro pagamento", c, 201)
	id1 := d["id"].(string)
	if g := gravado(t, f, hoje24, id1); g["pagaFatura"].(map[string]any)["ym"] != "2026-09" || g["excluirDoTotal"] != true {
		t.Errorf("a marca fica e o excluirDoTotal e derivado: %v", g)
	}
	// o segundo meio, o mesmo meio de novo e um valor qualquer: tudo vale
	for _, banco := range []string{"BANCO-B", "ITAU", "ITAU"} {
		c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento(banco, 1900, hoje24, marcaSet))
		esperar(t, "mais um pagamento em "+banco, c, 201)
	}
	// o cliente nao decide o excluirDoTotal
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 5, hoje24, marcaSet+`,"excluirDoTotal":false`))
	esperar(t, "excluirDoTotal do corpo", c, 201)
	if gravado(t, f, hoje24, d["id"].(string))["excluirDoTotal"] != true {
		t.Error("o servidor decide")
	}
	// um lancamento comum nao sai do total
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 5, hoje24, ""))
	esperar(t, "sem marca", c, 201)
	if gravado(t, f, hoje24, d["id"].(string))["excluirDoTotal"] != false {
		t.Error("sem a marca, fica no total")
	}
	// trocar e remover a marca
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/"+id1, map[string]string{"de": hoje24}, `{"pagaFatura":{"meio":"CARTAO-B","ym":"2026-08"}}`)
	esperar(t, "trocar", c, 200)
	if d["pagaFatura"].(map[string]any)["meio"] != "CARTAO-B" {
		t.Errorf("trocou: %v", d)
	}
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/"+id1, map[string]string{"de": hoje24}, `{"pagaFatura":null}`)
	esperar(t, "remover", c, 200)
	if g := gravado(t, f, hoje24, id1); g["pagaFatura"] != nil || g["excluirDoTotal"] != false {
		t.Errorf("sem a marca o lancamento volta ao normal: %v", g)
	}
	// o lote de backup mantem a marca
	lote := `{"itens":[{"id":"marcabkp001","dataEvento":"` + hoje24 + `","status":"PAGO","tipo":"saida","valor":10,"banco":"ITAU","descricao":"x","pagaFatura":{"meio":"CARTAO-A","ym":"2026-09"}}]}`
	_, d = chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	if d["gravados"] != 1.0 || gravado(t, f, hoje24, "marcabkp001")["pagaFatura"] == nil {
		t.Errorf("o backup mantem a marca: %v", d)
	}
}

// compras de setembro para ter uma fatura fechada com total: 3000 (29/08 a 28/09)
func comprasDeSetembro(t *testing.T, f *bancoFake) {
	t.Helper()
	compra(t, f, "s1", "2026-09-10", "CREDITO", "saida", 2000, nil)
	compra(t, f, "s2", "2026-09-20", "CREDITO", "saida", 1000, nil)
}

// G-T9: a fatura mostra os pagamentos marcados, a soma e a diferenca.
func TestFaturaMostraOsPagamentosMarcados(t *testing.T) {
	f := cenarioCartao(t)
	comprasDeSetembro(t, f)
	putFatura(t, "2026-09", `{"valorBanco":2500,"vencimento":"2026-10-05","pagador":"BANCO-B"}`)
	_, d := fatura(t, "own", "CARTAO-A", "2026-09")
	if d["vencimento"] != "2026-10-05" || d["pagador"] != "BANCO-B" || d["totalPago"] != 0.0 || len(d["pagamentos"].([]any)) != 0 {
		t.Errorf("sem pagamentos: %v", d)
	}
	_, a := chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 600, "2026-10-05T15:00:00.000Z", marcaSet))
	chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("BANCO-B", 1900, "2026-10-06T15:00:00.000Z", marcaSet))
	_, ex := chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 77, "2026-10-07T15:00:00.000Z", marcaSet))
	chamar(t, "own", "DELETE", "spaces/s1/tx/"+ex["id"].(string), map[string]string{"de": "2026-10-07T15:00:00.000Z"}, "")
	chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 33, "2027-09-01T15:00:00.000Z", marcaSet))                                           // fora da janela de leitura
	chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 44, "2026-10-08T15:00:00.000Z", `,"pagaFatura":{"meio":"CARTAO-A","ym":"2026-08"}`)) // de outra fatura
	_, d = fatura(t, "own", "CARTAO-A", "2026-09")
	pgs := d["pagamentos"].([]any)
	if len(pgs) != 2 || d["totalPago"] != 2500.0 || d["diferencaPaga"] != 0.0 {
		t.Fatalf("pagamentos: %v", d)
	}
	p0 := pgs[0].(map[string]any)
	if p0["id"] != a["id"] || p0["dia"] != "2026-10-05" || p0["meio"] != "ITAU" || p0["valor"] != 600.0 || p0["status"] != "PAGO" {
		t.Errorf("primeiro pagamento: %v", p0)
	}
	if pgs[1].(map[string]any)["meio"] != "BANCO-B" {
		t.Errorf("por data: %v", pgs)
	}
	// sem o valor do banco, a diferenca e contra o total calculado
	putFatura(t, "2026-09", `{"valorBanco":null}`)
	_, d = fatura(t, "own", "CARTAO-A", "2026-09")
	if d["diferencaPaga"] != -500.0 {
		t.Errorf("diferenca contra o total (3000): %v", d["diferencaPaga"])
	}
	// GET /cartoes traz o padrao
	putCartao(t, "CARTAO-A", `{"pagador":"BANCO-B","diaVencimento":5}`)
	_, l := chamar(t, "own", "GET", "spaces/s1/cartoes", nil, "")
	c0 := l["cartoes"].([]any)[0].(map[string]any)
	if c0["pagador"] != "BANCO-B" || c0["diaVencimento"] != 5.0 {
		t.Errorf("padrao do cartao: %v", c0)
	}
}

func faturasDeFuturos(t *testing.T) []any {
	t.Helper()
	_, d := chamar(t, "own", "GET", "spaces/s1/futuros", nil, "")
	fs, _ := d["faturas"].([]any)
	return fs
}

func umaFatura(t *testing.T, fs []any, meio, ym string) map[string]any {
	t.Helper()
	for _, x := range fs {
		m := x.(map[string]any)
		if m["meio"] == meio && m["ym"] == ym {
			return m
		}
	}
	return nil
}

// G-T10 e G-T11: as faturas em Futuros.
func TestFuturosTrazAsFaturas(t *testing.T) {
	f := cenarioCartao(t)
	comprasDeSetembro(t, f)
	if len(faturasDeFuturos(t)) != 0 {
		t.Fatal("sem vencimento nenhuma fatura entra")
	}
	putCartao(t, "CARTAO-A", `{"pagador":"BANCO-B","diaVencimento":5}`)
	if len(faturasDeFuturos(t)) != 0 {
		t.Fatal("o dia e o pagador do cartao nao fazem fatura nenhuma entrar")
	}
	putFatura(t, "2026-09", `{"valorBanco":2500,"vencimento":"2026-10-05","pagador":"BANCO-B"}`)
	putFatura(t, "2026-10", `{"vencimento":"2026-11-05"}`)
	fs := faturasDeFuturos(t)
	set, out := umaFatura(t, fs, "CARTAO-A", "2026-09"), umaFatura(t, fs, "CARTAO-A", "2026-10")
	if set == nil || out == nil || len(fs) != 2 {
		t.Fatalf("faturas: %v", fs)
	}
	if set["estado"] != "fechada" || set["valor"] != 2500.0 || set["total"] != 3000.0 || set["valorBanco"] != 2500.0 || set["vencimento"] != "2026-10-05" || set["atrasada"] != true || set["pagador"] != "BANCO-B" || set["rotulo"] != "set/26" {
		t.Errorf("set/26: %v", set)
	}
	sg := set["sugestao"].(map[string]any)
	if sg["descricao"] != "Pagamento da fatura set/26 CARTAO-A" || sg["valor"] != 2500.0 || sg["pagador"] != "BANCO-B" {
		t.Errorf("sugestao: %v", sg)
	}
	if out["estado"] != "aberta" || out["valor"] != 135.5 || out["vencimento"] != "2026-11-05" || out["atrasada"] != false {
		t.Errorf("out/26: %v", out)
	}
	// entram tambem em atrasados e nos meses, e o resumo soma
	_, d := chamar(t, "own", "GET", "spaces/s1/futuros", nil, "")
	if !strings.Contains(fmt.Sprint(d["atrasados"]), "fatura") || !strings.Contains(fmt.Sprint(d["meses"]), "2026-11") {
		t.Errorf("atrasados e meses: %v / %v", d["atrasados"], d["meses"])
	}
	r := d["resumo"].(map[string]any)
	// o cenario tem um previsto antigo de 555 (c9, atrasado): 555 + 2500 + 135,50 a pagar; 2 atrasados somando 3055
	if r["pagar"] != 3190.5 || r["atrasados"].(map[string]any)["n"] != 2.0 || r["atrasados"].(map[string]any)["valor"] != 3055.0 {
		t.Errorf("resumo: %v", r)
	}
	// um lancamento marcado tira a fatura da lista; excluir o lancamento a devolve
	_, p := chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 2500, hoje24, marcaSet))
	if umaFatura(t, faturasDeFuturos(t), "CARTAO-A", "2026-09") != nil {
		t.Error("a fatura paga some")
	}
	chamar(t, "own", "DELETE", "spaces/s1/tx/"+p["id"].(string), map[string]string{"de": hoje24}, "")
	if umaFatura(t, faturasDeFuturos(t), "CARTAO-A", "2026-09") == nil {
		t.Error("sem o lancamento marcado a fatura volta")
	}
	// a fatura de CARTAO-B (sem vencimento) nao entra; mudar o vencimento muda a lista (o cache nao esconde)
	putFatura(t, "2026-09", `{"vencimento":"2026-10-10"}`)
	if v := umaFatura(t, faturasDeFuturos(t), "CARTAO-A", "2026-09")["vencimento"]; v != "2026-10-10" {
		t.Errorf("vencimento novo: %v", v)
	}
	// o membro tambem ve as faturas (Futuros e leitura)
	_, dm := chamar(t, "mem", "GET", "spaces/s1/futuros", nil, "")
	if len(dm["faturas"].([]any)) != 2 {
		t.Errorf("member ve: %v", dm["faturas"])
	}
}

// G-T12 a G-T15: a trava da fatura conferida.
func TestFaturaConferidaTravaAsEscritas(t *testing.T) {
	f := cenarioCartao(t)
	c, _ := conferirFatura(t, "2026-10", `{"aceitarDiferenca":true}`) // a fatura 2026-10 (29/09 a 28/10)
	esperar(t, "conferir", c, 200)
	n := contaTx(f)
	travado := func(nome string, status int, d doc) {
		t.Helper()
		esperar(t, nome, status, 409)
		if d["codigo"] != "fatura_conferida" || d["meio"] != "CARTAO-A" || d["ym"] != "2026-10" {
			t.Errorf("%s: %v", nome, d)
		}
	}
	credito := func(data string) string {
		return `{"dataEvento":"` + data + `","status":"CREDITO","tipo":"saida","valor":9,"banco":"CARTAO-A","descricao":"nova"}`
	}
	st, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, credito("2026-10-12T15:00:00.000Z"))
	travado("criar compra no periodo", st, d)
	de := map[string]string{"de": "2026-10-10T15:00:00.000Z"} // c2
	for nome, corpo := range map[string]string{
		"mudar o valor":          `{"valor":60}`,
		"mudar a data de fatura": `{"dataEvento":"2026-10-29T15:00:00.000Z"}`,
		"mudar o ajuste":         `{"faturaAjuste":1}`,
		"ocultar":                `{"oculto":true}`,
		"tirar do cartao":        `{"banco":"ITAU"}`,
	} {
		st, d = chamar(t, "own", "PUT", "spaces/s1/tx/c2", de, corpo)
		travado(nome, st, d)
	}
	st, d = chamar(t, "own", "DELETE", "spaces/s1/tx/c2", de, "")
	travado("excluir", st, d)
	lote := `{"itens":[{"id":"loteconf001","dataEvento":"2026-10-12T15:00:00.000Z","status":"CREDITO","tipo":"saida","valor":9,"banco":"CARTAO-A","descricao":"x"}]}`
	_, dl := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	if dl["gravados"] != 0.0 || !strings.Contains(fmt.Sprint(dl["rejeitados"]), "fatura_conferida") {
		t.Errorf("o lote rejeita com o codigo: %v", dl)
	}
	// trazer uma compra de uma fatura livre PARA a conferida (c5 e da 2026-11) tambem e recusado
	st, d = chamar(t, "own", "PUT", "spaces/s1/tx/c5", map[string]string{"de": "2026-10-29T15:00:00.000Z"}, `{"dataEvento":"2026-10-12T15:00:00.000Z"}`)
	travado("trazer para a conferida", st, d)
	if contaTx(f) != n || gravado(t, f, "2026-10-10T15:00:00.000Z", "c2")["valor"] != 50.5 {
		t.Error("nada pode ser gravado")
	}
	// o que NAO muda a fatura passa
	st, _ = chamar(t, "own", "PUT", "spaces/s1/tx/c2", de, `{"descricao":"outra","nota":"x","tags":["a"]}`)
	esperar(t, "texto e tags", st, 200)
	st, _ = chamar(t, "own", "PUT", "spaces/s1/tx/c5", map[string]string{"de": "2026-10-29T15:00:00.000Z"}, `{"valor":41}`)
	esperar(t, "compra de outra fatura (2026-11)", st, 200)
	st, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 135.5, hoje24, `,"pagaFatura":{"meio":"CARTAO-A","ym":"2026-10"}`))
	esperar(t, "o pagamento (nao e compra de credito) passa", st, 201)
	// desfazer a conferencia destrava
	st, _ = chamar(t, "own", "DELETE", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10/conferir", nil, "")
	esperar(t, "desfazer a conferencia", st, 200)
	st, _ = chamar(t, "own", "PUT", "spaces/s1/tx/c2", de, `{"valor":60}`)
	esperar(t, "depois de desfazer", st, 200)
	st, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, credito("2026-10-12T15:00:00.000Z"))
	esperar(t, "criar de novo", st, 201)
}

// O credito e o debito do mesmo banco tem o mesmo nome: o pagador pode ser o proprio cartao e o debito dele pode levar a marca.
func TestCartaoPagoPeloDebitoDoMesmoBanco(t *testing.T) {
	f := cenarioCartao(t)
	c, _ := putCartao(t, "CARTAO-A", `{"pagador":"CARTAO-A"}`)
	esperar(t, "pagador e o proprio cartao", c, 200)
	if g := cartaoGravado(t, f, "CARTAO-A"); g["pagador"] != "CARTAO-A" {
		t.Errorf("pagador = %v", g["pagador"])
	}
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("CARTAO-A", 10, hoje24, marcaSet))
	esperar(t, "debito do proprio banco do cartao leva a marca", c, 201)
}

// O dia de vencimento e o pagador do cartao so sugerem: nao mudam nenhuma fatura (decisao do dono, 2026-10-25).
func TestPadroesDoCartaoSoSugerem(t *testing.T) {
	cenarioCartao(t)
	putCartao(t, "CARTAO-A", `{"pagador":"BANCO-B","diaVencimento":16}`)
	_, d := fatura(t, "own", "CARTAO-A", "2026-09")
	if d["vencimento"] != nil || d["pagador"] != nil {
		t.Errorf("a fatura nao herda o padrao: %v / %v", d["vencimento"], d["pagador"])
	}
	p := d["preencher"].(map[string]any)
	if p["vencimento"] != "2026-10-16" || p["pagador"] != "BANCO-B" {
		t.Errorf("sugestao para preencher: %v", p)
	}
	putFatura(t, "2026-09", `{"vencimento":"2026-10-16","pagador":"BANCO-B"}`)
	putCartao(t, "CARTAO-A", `{"pagador":"BANCO-A","diaVencimento":30}`)
	_, d = fatura(t, "own", "CARTAO-A", "2026-09")
	if d["vencimento"] != "2026-10-16" || d["pagador"] != "BANCO-B" {
		t.Errorf("mudar o padrao nao mexe na fatura: %v / %v", d["vencimento"], d["pagador"])
	}
	putFatura(t, "2026-09", `{"vencimento":null}`)
	_, d = fatura(t, "own", "CARTAO-A", "2026-09")
	if d["vencimento"] != nil {
		t.Errorf("limpar tira o vencimento da fatura: %v", d["vencimento"])
	}
}
