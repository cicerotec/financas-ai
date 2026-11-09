package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Item 58 (.ai/specs/meios-de-pagamento/spec.md, Fase C): desativar e encerrar a conta. So a partir de hoje, com saldo zero e sem evento; depois disso nada
// novo entra no meio (importar so antes da dataInicio). Hoje e 2026-11-03.

// cenarioEncerrar parte do cenarioMeios e acrescenta:
//   - ENCERRADO: meio desativado desde 2026-10-31, com um lancamento antigo (de 2020, antes da dataInicio 2021-01-01) e saldo inicial que o zera;
//   - SEM-INICIO: meio desativado sem dataInicio em CFG#SALDOS.
func cenarioEncerrar(t *testing.T) *bancoFake {
	t.Helper()
	f := cenarioMeios(t)
	d := f.ler(t, pkEspaco, "CFG#LISTAS")
	d["banco"] = append(d["banco"].([]any), "ENCERRADO", "SEM-INICIO")
	d["ativoAte"] = map[string]any{"ENCERRADO": "2026-10-31", "SEM-INICIO": "2026-10-31"}
	f.colocar(t, d)
	// todo meio de teste tem inicio do controle (sem ele a conferencia recusa, sem ler nada), menos SEM-INICIO
	por := map[string]any{}
	for _, m := range []string{"ENCERRADO", "ITAU", "VR-NOVO", "CITIBANK", "SO-EXCLUIDO", "SO-PREVISTO", "SODEXO"} {
		por[m] = map[string]any{"saldoInicial": 0.0, "dataInicio": "2021-01-01"}
	}
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 1.0, "porBanco": por})
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2020-01-01T12:00:00.000Z#des1", "id": "des1", "dataEvento": "2020-01-01T12:00:00.000Z",
		"status": "PAGO", "tipo": "saida", "valor": 5.0, "banco": "ENCERRADO", "descricao": "antigo", "nota": "", "tags": []any{}, "criadoPor": "own"})
	return f
}

func codMeio(t *testing.T, d doc) string {
	t.Helper()
	c, _ := d["codigo"].(string)
	return c
}

func pendenciasDe(d doc) string {
	var partes []string
	for _, x := range d["pendencias"].([]any) {
		partes = append(partes, x.(string))
	}
	return strings.Join(partes, " | ")
}

// M-T18: desativar com saldo diferente de zero e recusado, com a frase do saldo; nada e gravado.
func TestDesativarComSaldoEPendencia(t *testing.T) {
	f := cenarioEncerrar(t)
	st, d := chamar(t, "own", "PUT", "spaces/s1/meios/ITAU", nil, `{"ativoAte":"2026-11-03"}`)
	esperar(t, "ITAU tem saldo", st, 409)
	if codMeio(t, d) != "MEIO_COM_PENDENCIA" || !strings.Contains(pendenciasDe(d), "saldo") {
		t.Errorf("deveria citar o saldo: %v", d)
	}
	if ate, _ := f.ler(t, pkEspaco, "CFG#LISTAS")["ativoAte"].(map[string]any); ate["ITAU"] != nil {
		t.Errorf("a recusa nao grava nada: %v", ate)
	}
}

// M-T19: so o futuro impede (previsto no passado nao existe, e a conferencia nao le o passado); so o dono desativa.
func TestDesativarComPrevistoImpede(t *testing.T) {
	f := cenarioEncerrar(t)
	// um previsto de 2019 e um de 2026-09-10 (SO-PREVISTO) ja ficaram para tras: nao impedem
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2019-05-05T12:00:00.000Z#pv19", "id": "pv19", "dataEvento": "2019-05-05T12:00:00.000Z",
		"status": "PAGO", "tipo": "saida", "valor": 3.0, "banco": "SODEXO", "descricao": "p", "tags": []any{}, "criadoPor": "own", "previsto": true, "oculto": true})
	st, d := chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, `{"ativoAte":"hoje"}`)
	esperar(t, "previsto de 2019 nao impede", st, 200)
	st, _ = chamar(t, "own", "PUT", "spaces/s1/meios/SO-PREVISTO", nil, `{"ativoAte":"hoje"}`)
	esperar(t, "previsto do passado nao impede", st, 200)
	// criar um lancamento de data futura num meio livre vira previsto e impede
	c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-12-20T12:00:00.000Z", "VR-NOVO", 7))
	esperar(t, "lancar no futuro", c, 201)
	st, d = chamar(t, "own", "PUT", "spaces/s1/meios/VR-NOVO", nil, `{"ativoAte":"hoje"}`)
	esperar(t, "data futura", st, 409)
	if !strings.Contains(pendenciasDe(d), "previsto") && !strings.Contains(pendenciasDe(d), "futura") {
		t.Errorf("deveria citar o lancamento futuro: %v", d)
	}
}

// M-T20: a recorrencia ativa cuja ponta e do meio impede desativar.
func TestDesativarComRecorrenciaImpede(t *testing.T) {
	cenarioEncerrar(t)
	c, l := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-11-01T12:00:00.000Z", "VR-NOVO", 7))
	esperar(t, "lancar", c, 201)
	c, _ = chamar(t, "own", "POST", "spaces/s1/recorrencias", map[string]string{"de": l["dataEvento"].(string)}, fmt.Sprintf(`{"id":"%s","intervalo":1}`, l["id"]))
	esperar(t, "tornar recorrente", c, 201)
	st, d := chamar(t, "own", "PUT", "spaces/s1/meios/VR-NOVO", nil, `{"ativoAte":"2026-11-03"}`)
	esperar(t, "recorrencia ativa", st, 409)
	if !strings.Contains(pendenciasDe(d), "recorrência") {
		t.Errorf("deveria citar a recorrencia: %v", d)
	}
}

// M-T22: meio zerado e sem evento desativa; depois, a leitura mostra inativo.
func TestDesativarMeioLimpo(t *testing.T) {
	f := cenarioEncerrar(t)
	st, d := chamar(t, "own", "PUT", "spaces/s1/meios/VR-NOVO", nil, `{"ativoAte":"2026-11-03"}`)
	esperar(t, "meio limpo", st, 200)
	if d["ativo"] != false {
		t.Errorf("deveria ficar inativo: %v", d)
	}
	if ate, _ := f.ler(t, pkEspaco, "CFG#LISTAS")["ativoAte"].(map[string]any); ate["VR-NOVO"] != "2026-11-03" {
		t.Errorf("gravado: %v", ate)
	}
}

// M-T23: meio desativado nao recebe lancamento novo: criar, transferir (as duas pontas), confirmar e trocar o banco.
func TestMeioDesativadoNaoRecebeLancamento(t *testing.T) {
	f := cenarioEncerrar(t)
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-11-02T12:00:00.000Z", "ENCERRADO", 7))
	esperar(t, "criar", c, 409)
	if codMeio(t, d) != "MEIO_DESATIVADO" || d["meio"] != "ENCERRADO" {
		t.Errorf("criar: %v", d)
	}
	for _, corpo := range []string{`{"origem":"ENCERRADO","destino":"ITAU","valor":3}`, `{"origem":"ITAU","destino":"ENCERRADO","valor":3}`} {
		c, d = chamar(t, "own", "POST", "spaces/s1/transferencias", nil, corpo)
		esperar(t, "transferir "+corpo, c, 409)
		if codMeio(t, d) != "MEIO_DESATIVADO" {
			t.Errorf("transferir: %v", d)
		}
	}
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"banco":"ENCERRADO"}`)
	esperar(t, "trocar o banco para o desativado", c, 409)
	if codMeio(t, d) != "MEIO_DESATIVADO" {
		t.Errorf("trocar banco: %v", d)
	}
	if f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono")["banco"] != "ITAU" {
		t.Error("o lancamento continua no ITAU")
	}
	// confirmar um previsto que esta num meio encerrado (dado antigo) tambem e recusado
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2026-10-20T12:00:00.000Z#pvenc", "id": "pvenc", "dataEvento": "2026-10-20T12:00:00.000Z",
		"status": "PAGO", "tipo": "saida", "valor": 4.0, "banco": "ENCERRADO", "descricao": "p", "tags": []any{}, "criadoPor": "own", "previsto": true})
	c, d = chamar(t, "own", "POST", "spaces/s1/tx/pvenc/confirmar", map[string]string{"de": "2026-10-20T12:00:00.000Z"}, `{"valor":4,"dataEvento":"2026-11-02T12:00:00.000Z","status":"PAGO"}`)
	esperar(t, "confirmar", c, 409)
	if codMeio(t, d) != "MEIO_DESATIVADO" {
		t.Errorf("confirmar: %v", d)
	}
}

// M-T24: num lancamento de meio desativado so descricao, nota e tags mudam; o formulario inteiro, com null e data sem os milissegundos, nao e falso positivo.
func TestEditarLancamentoDeMeioDesativado(t *testing.T) {
	cenarioEncerrar(t)
	de := map[string]string{"de": "2020-01-01T12:00:00.000Z"}
	formulario := `{"dataEvento":"2020-01-01T12:00:00Z","status":"PAGO","tipo":"saida","valor":5,"banco":"ENCERRADO","descricao":"novo","nota":"anotei",` +
		`"tags":["Lazer"],"pagaFatura":null,"aviso":null,"faturaAjuste":null,"oculto":false}`
	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/des1", de, formulario)
	esperar(t, "descricao, nota e tags", c, 200)
	if d["descricao"] != "novo" {
		t.Errorf("editou: %v", d)
	}
	for _, ruim := range []string{`{"valor":9}`, `{"status":"CREDITO"}`, `{"oculto":true}`, `{"tipo":"entrada"}`, `{"dataEvento":"2020-01-02T12:00:00.000Z"}`, `{"banco":"ITAU"}`} {
		c, d = chamar(t, "own", "PUT", "spaces/s1/tx/des1", de, ruim)
		esperar(t, "recusa "+ruim, c, 409)
		if codMeio(t, d) != "MEIO_DESATIVADO" {
			t.Errorf("%s: %v", ruim, d)
		}
	}
	c, d = chamar(t, "own", "DELETE", "spaces/s1/tx/des1", de, "")
	esperar(t, "excluir", c, 409)
	if codMeio(t, d) != "MEIO_DESATIVADO" {
		t.Errorf("excluir: %v", d)
	}
	// restaurar a versao 1 (antes da edicao) tambem e recusado: o meio esta encerrado
	c, d = chamar(t, "own", "POST", "spaces/s1/tx/des1/versoes/1/restaurar", nil, "")
	esperar(t, "restaurar", c, 409)
	if codMeio(t, d) != "MEIO_DESATIVADO" {
		t.Errorf("restaurar: %v", d)
	}
}

// M-T26: restaurar uma versao cujo meio foi removido da lista e recusado, e o historico nao muda.
func TestRestaurarParaMeioRemovido(t *testing.T) {
	f := cenarioEncerrar(t)
	// o lancamento iddono do ITAU e editado para o VR-NOVO, que depois e removido (nunca teve outro uso)
	deConf := map[string]string{"de": dataDono, "confirmaMesFechado": "1"}
	c, _ := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", deConf, `{"banco":"VR-NOVO"}`)
	esperar(t, "mover para VR-NOVO", c, 200)
	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/iddono", deConf, `{"banco":"ITAU"}`)
	esperar(t, "voltar para ITAU", c, 200)
	// remover o VR-NOVO nao e possivel enquanto algum lancamento (mesmo antigo) o tiver; o nome sai da lista por aqui para o teste:
	d := f.ler(t, pkEspaco, "CFG#LISTAS")
	var resto []any
	for _, n := range d["banco"].([]any) {
		if n != "VR-NOVO" {
			resto = append(resto, n)
		}
	}
	d["banco"] = resto
	f.colocar(t, d)
	// a versao 2 tem banco VR-NOVO
	c, dd := chamar(t, "own", "POST", "spaces/s1/tx/iddono/versoes/2/restaurar", conf, "")
	esperar(t, "restaurar para o meio removido", c, 409)
	if codMeio(t, dd) != "MEIO_DESATIVADO" || !strings.Contains(fmt.Sprint(dd["erro"]), "removido") {
		t.Errorf("a mensagem diz que o meio foi removido: %v", dd)
	}
	if f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono")["banco"] != "ITAU" {
		t.Error("nada foi restaurado")
	}
}

// M-T25: importar num meio desativado so aceita o registro anterior a dataInicio; sem dataInicio, rejeita; em meio ativo, entra; e nao sobrescreve registro posterior.
func TestImportarEmMeioDesativado(t *testing.T) {
	f := cenarioEncerrar(t)
	item := func(id, data, banco string) string {
		return fmt.Sprintf(`{"id":"%s","dataEvento":"%s","status":"PAGO","tipo":"saida","valor":1,"banco":"%s","descricao":"imp"}`, id, data, banco)
	}
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2022-02-02T12:00:00.000Z#posterior", "id": "posterior", "dataEvento": "2022-02-02T12:00:00.000Z",
		"status": "PAGO", "tipo": "saida", "valor": 9.0, "banco": "ENCERRADO", "descricao": "gravado", "tags": []any{}, "criadoPor": "own"})
	lote := `{"itens":[` + strings.Join([]string{
		item("idantes0001", "2020-06-01T12:00:00.000Z", "ENCERRADO"),     // entra: antes da dataInicio
		item("iddepois001", "2022-01-01T12:00:00.000Z", "ENCERRADO"),     // rejeitado: depois
		item("idigual0001", "2021-01-01T03:00:00.000Z", "ENCERRADO"),     // rejeitado: no instante do inicio (00:00 em Sao Paulo) ja conta no saldo
		item("idsemini001", "2020-06-01T12:00:00.000Z", "SEM-INICIO"),    // rejeitado: o meio nao tem dataInicio
		item("idativo0001", "2026-11-02T12:00:00.000Z", "ITAU"),          // entra: meio ativo
		item("posterior", "2022-02-02T12:00:00.000Z", "ITAU"),            // rejeitado: tiraria um registro de dentro do meio encerrado
	}, ",") + `]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", conf, lote)
	esperar(t, "importar", c, 200)
	if d["gravados"] != 2.0 {
		t.Errorf("gravados = %v, esperado 2: %v", d["gravados"], d)
	}
	rej, _ := d["rejeitados"].([]any)
	if len(rej) != 4 {
		t.Fatalf("rejeitados = %v, esperado 4", d["rejeitados"])
	}
	for _, r := range rej {
		if r.(map[string]any)["codigo"] != "meio_desativado" {
			t.Errorf("codigo do rejeitado: %v", r)
		}
	}
	if f.ler(t, pkEspaco, dePrefixoTx+"2020-06-01T12:00:00.000Z#idantes0001") == nil || f.ler(t, pkEspaco, dePrefixoTx+"2026-11-02T12:00:00.000Z#idativo0001") == nil {
		t.Error("os dois aceitos deveriam estar gravados")
	}
	if f.ler(t, pkEspaco, dePrefixoTx+"2022-02-02T12:00:00.000Z#posterior")["banco"] != "ENCERRADO" {
		t.Error("o registro posterior ao inicio continua no meio encerrado")
	}
}

// M-T27: o meio so com lancamentos excluidos logicamente pode ser removido; com um vivo ou previsto, nao.
func TestRemoverMeioSoComExcluidos(t *testing.T) {
	f := cenarioMeios(t)
	st, _ := chamar(t, "own", "DELETE", "spaces/s1/meios/SO-EXCLUIDO", nil, "")
	esperar(t, "so excluidos", st, 204)
	for _, b := range f.ler(t, pkEspaco, "CFG#LISTAS")["banco"].([]any) {
		if b == "SO-EXCLUIDO" {
			t.Error("o meio saiu da lista")
		}
	}
	st, d := chamar(t, "own", "DELETE", "spaces/s1/meios/SO-PREVISTO", nil, "")
	esperar(t, "com previsto", st, 409)
	if codMeio(t, d) != "MEIO_EM_USO" {
		t.Errorf("previsto: %v", d)
	}
}

// M-T28: o pagador de um cartao precisa ser um meio ativo.
func TestPagadorDesativadoERecusado(t *testing.T) {
	cenarioEncerrar(t)
	st, d := chamar(t, "own", "PUT", "spaces/s1/cartoes/CITIBANK", nil, `{"pagador":"ENCERRADO"}`)
	esperar(t, "pagador desativado", st, 400)
	if !strings.Contains(fmt.Sprint(d["erro"]), "desativado") {
		t.Errorf("a mensagem diz que o pagador esta desativado: %v", d)
	}
	st, _ = chamar(t, "own", "PUT", "spaces/s1/cartoes/CITIBANK", nil, `{"pagador":"ITAU"}`)
	esperar(t, "pagador ativo", st, 200)
}

// M-T29: /saldos marca encerrado so o desativado de saldo zero; o desativado com saldo continua visivel (o servidor nunca esconde dinheiro).
func TestSaldosMarcaEncerrado(t *testing.T) {
	f := cenarioEncerrar(t)
	d := f.ler(t, pkEspaco, "CFG#LISTAS")
	d["ativoAte"] = map[string]any{"ENCERRADO": "2026-10-31", "SEM-INICIO": "2026-10-31", "SODEXO": "2026-10-31"}
	f.colocar(t, d)
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2026-10-10T12:00:00.000Z#comsaldo", "id": "comsaldo", "dataEvento": "2026-10-10T12:00:00.000Z",
		"status": "PAGO", "tipo": "entrada", "valor": 50.0, "banco": "SODEXO", "descricao": "s", "tags": []any{}, "criadoPor": "own"})
	st, resp := chamar(t, "own", "GET", "spaces/s1/saldos", nil, "")
	esperar(t, "saldos", st, 200)
	enc := map[string]bool{}
	for _, x := range resp["meios"].([]any) {
		m := x.(map[string]any)
		enc[m["meio"].(string)], _ = m["encerrado"].(bool)
	}
	// item 60: SEM-INICIO nao e calculado (nao se sabe o saldo), entao nunca vira encerrado
	if !enc["ENCERRADO"] || enc["SEM-INICIO"] {
		t.Errorf("desativado e zerado fica encerrado; o nao calculado nao: %v", enc)
	}
	if enc["SODEXO"] || enc["ITAU"] {
		t.Errorf("desativado com saldo e meio ativo continuam visiveis: %v", enc)
	}
}

// compraCredito semeia uma compra de credito no CITIBANK (cartao que fecha no dia 5; hoje 2026-11-03, a fatura atual e 2026-11).
func compraCredito(t *testing.T, f *bancoFake, id, dia string, valor float64) {
	t.Helper()
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + dia + "T12:00:00.000Z#" + id, "id": id, "dataEvento": dia + "T12:00:00.000Z",
		"status": "CREDITO", "tipo": "saida", "valor": valor, "banco": "CITIBANK", "descricao": id, "tags": []any{}, "criadoPor": "own"})
}

// pagamentoDaFatura semeia o lancamento que registra o pagamento da fatura ym do CITIBANK (a marca pagaFatura).
func pagamentoDaFatura(t *testing.T, f *bancoFake, id, dia, ym string, valor float64) {
	t.Helper()
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + dia + "T12:00:00.000Z#" + id, "id": id, "dataEvento": dia + "T12:00:00.000Z",
		"status": "PAGO", "tipo": "saida", "valor": valor, "banco": "ITAU", "descricao": id, "tags": []any{}, "criadoPor": "own",
		"pagaFatura": map[string]any{"meio": "CITIBANK", "ym": ym}})
}

// M-T31: cartao com fatura (aberta ou fechada, das tres ultimas) com valor e sem pagamento registrado nao desativa, mesmo sem vencimento; registrado o pagamento, desativa.
func TestDesativarCartaoComFaturaSemPagamento(t *testing.T) {
	f := cenarioEncerrar(t)
	compraCredito(t, f, "fc1", "2026-09-20", 40) // fatura 2026-10 (fechada, 06/09 a 05/10)
	compraCredito(t, f, "fc2", "2026-10-20", 100) // fatura 2026-11 (a atual, 06/10 a 05/11)
	st, d := chamar(t, "own", "PUT", "spaces/s1/meios/CITIBANK", nil, `{"ativoAte":"2026-11-03"}`)
	esperar(t, "duas faturas sem pagamento", st, 409)
	p := pendenciasDe(d)
	if !strings.Contains(p, "R$ 40,00") || !strings.Contains(p, "R$ 100,00") || !strings.Contains(p, "não tem pagamento registrado") {
		t.Errorf("deveria citar as duas faturas e o valor de cada uma: %s", p)
	}
	pagamentoDaFatura(t, f, "pg1", "2026-10-06", "2026-10", 40)
	st, d = chamar(t, "own", "PUT", "spaces/s1/meios/CITIBANK", nil, `{"ativoAte":"2026-11-03"}`)
	esperar(t, "falta a fatura atual", st, 409)
	if p := pendenciasDe(d); strings.Contains(p, "R$ 40,00") || !strings.Contains(p, "R$ 100,00") {
		t.Errorf("so a fatura atual continua pendente: %s", p)
	}
	pagamentoDaFatura(t, f, "pg2", "2026-11-03", "2026-11", 100)
	st, d = chamar(t, "own", "PUT", "spaces/s1/meios/CITIBANK", nil, `{"ativoAte":"2026-11-03"}`)
	esperar(t, "pagamentos registrados", st, 200)
	if d["ativoAte"] != "2026-11-03" {
		t.Errorf("deveria desativar: %v", d)
	}
}

// M-T32: o PUT de saldos nao muda saldo inicial nem dataInicio de meio desativado; o resto continua gravando.
func TestPutDeSaldosNaoMudaMeioDesativado(t *testing.T) {
	f := cenarioEncerrar(t)
	cor := func(ini string, extra string) string {
		v := f.ler(t, pkEspaco, "CFG#SALDOS")["version"].(float64)
		return fmt.Sprintf(`{"version":%v,"porBanco":{"ENCERRADO":{"saldoInicial":0,"dataInicio":"%s"}%s}}`, v, ini, extra)
	}
	st, d := chamar(t, "own", "PUT", "spaces/s1/cfg/saldos", nil, cor("2022-01-01", ""))
	esperar(t, "mudar a dataInicio", st, 409)
	if codMeio(t, d) != "MEIO_DESATIVADO" {
		t.Errorf("deveria ser MEIO_DESATIVADO: %v", d)
	}
	st, _ = chamar(t, "own", "PUT", "spaces/s1/cfg/saldos", nil, strings.Replace(cor("2021-01-01", ""), `"saldoInicial":0`, `"saldoInicial":10`, 1))
	esperar(t, "mudar o saldo inicial", st, 409)
	if b := f.ler(t, pkEspaco, "CFG#SALDOS")["porBanco"].(map[string]any)["ENCERRADO"].(map[string]any); b["dataInicio"] != "2021-01-01" || b["saldoInicial"] != 0.0 {
		t.Errorf("a recusa nao grava nada: %v", b)
	}
	st, _ = chamar(t, "own", "PUT", "spaces/s1/cfg/saldos", nil, cor("2021-01-01", `,"ITAU":{"saldoInicial":5,"dataInicio":"2026-01-01"}`))
	esperar(t, "mudar so outro meio", st, 200)
}

// M-T33: meio desativado nao vira cartao; meio ativo vira.
func TestMeioDesativadoNaoViraCartao(t *testing.T) {
	cenarioEncerrar(t)
	st, d := chamar(t, "own", "POST", "spaces/s1/cartoes", nil, `{"meio":"ENCERRADO","diaPadrao":10}`)
	esperar(t, "cartao desativado", st, 409)
	if codMeio(t, d) != "MEIO_DESATIVADO" {
		t.Errorf("deveria ser MEIO_DESATIVADO: %v", d)
	}
	st, d = chamar(t, "own", "GET", "spaces/s1/cartoes", nil, "")
	esperar(t, "listar cartoes", st, 200)
	for _, x := range d["disponiveis"].([]any) {
		if x.(map[string]any)["meio"] == "ENCERRADO" {
			t.Errorf("a lista para adicionar cartao nao oferece meio desativado: %v", d["disponiveis"])
		}
	}
	st, _ = chamar(t, "own", "POST", "spaces/s1/cartoes", nil, `{"meio":"SODEXO","diaPadrao":10}`)
	esperar(t, "cartao ativo", st, 200)
}

// M-T34: o app pede "hoje" e o servidor usa o dia dele (sem depender do relogio do aparelho); outras palavras continuam 400.
func TestDesativarPedindoHoje(t *testing.T) {
	f := cenarioEncerrar(t)
	st, d := chamar(t, "own", "PUT", "spaces/s1/meios/VR-NOVO", nil, `{"ativoAte":"hoje"}`)
	esperar(t, "hoje", st, 200)
	if d["ativoAte"] != "2026-11-03" || d["ativo"] != false {
		t.Errorf("deveria gravar o dia do servidor: %v", d)
	}
	if ate, _ := f.ler(t, pkEspaco, "CFG#LISTAS")["ativoAte"].(map[string]any); ate["VR-NOVO"] != "2026-11-03" {
		t.Errorf("gravado: %v", ate)
	}
	st, _ = chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, `{"ativoAte":"amanha"}`)
	esperar(t, "outra palavra", st, 400)
	st, _ = chamar(t, "own", "PUT", "spaces/s1/meios/ITAU", nil, `{"ativoAte":"hoje"}`)
	esperar(t, "hoje com pendencia", st, 409)
}

// M-T35: meio sem nenhum ponto de partida do saldo (sem mes conferido, cache nem dataInicio) nao desativa: a recusa vem sem ler lancamento algum.
func TestDesativarSemInicioDoControleRecusa(t *testing.T) {
	f := cenarioEncerrar(t)
	d := f.ler(t, pkEspaco, "CFG#SALDOS")
	delete(d["porBanco"].(map[string]any), "VR-NOVO")
	f.colocar(t, d)
	f.falharQuerySemInicio = errors.New("leu o passado inteiro") // qualquer Query a partir do inicio dos tempos falha
	st, resp := chamar(t, "own", "PUT", "spaces/s1/meios/VR-NOVO", nil, `{"ativoAte":"hoje"}`)
	esperar(t, "sem inicio", st, 409)
	if p := pendenciasDe(resp); !strings.Contains(p, "início do controle") || !strings.Contains(p, "Saldos") {
		t.Errorf("deveria pedir o inicio do controle em Saldos: %s", p)
	}
	if ate, _ := f.ler(t, pkEspaco, "CFG#LISTAS")["ativoAte"].(map[string]any); ate["VR-NOVO"] != nil {
		t.Errorf("a recusa nao grava nada: %v", ate)
	}
}

// M-T36: custo. Com inicio do controle, desativar nao le o passado: um Query a partir do inicio dos tempos (SK = TX#) falharia, e a desativacao passa.
func TestDesativarNaoLeOPassado(t *testing.T) {
	f := cenarioEncerrar(t)
	f.falharQuerySemInicio = errors.New("leu o passado inteiro")
	st, resp := chamar(t, "own", "PUT", "spaces/s1/meios/VR-NOVO", nil, `{"ativoAte":"hoje"}`)
	esperar(t, "desativar sem ler o passado", st, 200)
	if resp["ativo"] != false {
		t.Errorf("deveria desativar: %v", resp)
	}
}

// M-T37: sem dataInicio, mas com um mes conferido (o ponto de partida do saldo), desativa e nao le o passado.
func TestDesativarComFechamentoSemDataInicio(t *testing.T) {
	f := cenarioEncerrar(t)
	d := f.ler(t, pkEspaco, "CFG#SALDOS")
	delete(d["porBanco"].(map[string]any), "VR-NOVO")
	f.colocar(t, d)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "SALDO#VR-NOVO", "banco": "VR-NOVO", "version": 1.0,
		"conferidos": map[string]any{"2026-10": map[string]any{"saldo": 0.0, "real": 0.0, "em": "2026-11-01T12:00:00.000Z", "por": "own"}}})
	f.falharQuerySemInicio = errors.New("leu o passado inteiro")
	st, resp := chamar(t, "own", "PUT", "spaces/s1/meios/VR-NOVO", nil, `{"ativoAte":"hoje"}`)
	esperar(t, "com mes conferido", st, 200)
	if resp["ativo"] != false {
		t.Errorf("deveria desativar: %v", resp)
	}
}
