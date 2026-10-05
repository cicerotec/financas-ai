package main

import (
	"fmt"
	"testing"
	"time"
)

// relogio fixo: hoje e 05/10/2026, entao o mes atual e 2026-10
func relogioFixo(t *testing.T) {
	t.Helper()
	velho := agora
	agora = func() time.Time { return time.Date(2026, 10, 5, 15, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { agora = velho })
}

func corpoTx(data, banco string, valor float64) string {
	return fmt.Sprintf(`{"dataEvento":"%s","status":"PAGO","tipo":"saida","valor":%v,"banco":"%s","descricao":"t"}`, data, valor, banco)
}

func conferir(t *testing.T, quem, banco, mes string, ver float64) (int, doc) {
	return chamar(t, quem, "POST", "spaces/s1/fechamentos/conferir", nil,
		fmt.Sprintf(`{"banco":"%s","mes":"%s","version":%v,"saldo":100,"real":100}`, banco, mes, ver))
}

func TestMesPeloFusoDeSaoPaulo(t *testing.T) {
	for entrada, quer := range map[string]string{
		"2026-04-01T01:30:00.000Z": "2026-03", // 22:30 de 31/03 em Sao Paulo
		"2026-03-31T23:00:00.000Z": "2026-03",
		"2026-04-01T03:00:00.000Z": "2026-04", // 00:00 de 01/04 em Sao Paulo
		"2026-01-01T10:00:00.000Z": "2026-01",
		"lixo":                     "",
	} {
		if got := mesDe(entrada); got != quer {
			t.Errorf("mesDe(%q) = %q, esperado %q", entrada, got, quer)
		}
	}
}

func TestConferirTravaOMesEOsAnteriores(t *testing.T) {
	relogioFixo(t)
	cenario(t)
	c, d := conferir(t, "own", "ITAU", "2026-03", 0)
	esperar(t, "owner confere marco", c, 200)
	if d["version"] != 1.0 {
		t.Errorf("version deveria ir para 1, veio %v", d["version"])
	}

	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-02-10T12:00:00.000Z", "ITAU", 5))
	esperar(t, "criar em fevereiro (antes do conferido)", c, 409)
	if d["codigo"] != "mes_conferido" || d["conferidoAte"] != "2026-03" {
		t.Errorf("resposta da trava inesperada: %v", d)
	}
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-03-15T12:00:00.000Z", "ITAU", 5))
	esperar(t, "criar no proprio mes conferido", c, 409)
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-04-02T12:00:00.000Z", "ITAU", 5))
	esperar(t, "criar em abril (depois)", c, 201)
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-02-10T12:00:00.000Z", "NUBANK", 5))
	esperar(t, "outro banco nao e afetado", c, 201)
	c, _ = chamar(t, "mem", "POST", "spaces/s1/tx", nil, corpoTx("2026-03-10T12:00:00.000Z", "ITAU", 5))
	esperar(t, "member tambem e barrado", c, 409)
}

func TestMesConferidoSoPermiteEditarOQueNaoMexeNoSaldo(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	conferir(t, "own", "ITAU", "2026-03", 0)
	de := map[string]string{"de": dataDono}
	const caminho = "spaces/s1/tx/iddono"

	c, _ := chamar(t, "own", "PUT", caminho, de, `{"descricao":"so texto","nota":"n","tags":["Lazer"]}`)
	esperar(t, "editar texto e tags", c, 200)
	if f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono")["descricao"] != "so texto" {
		t.Error("a edicao de texto deveria ter sido gravada")
	}
	for nome, corpo := range map[string]string{
		"valor":  `{"valor":11}`,
		"oculto": `{"oculto":true}`,
		"status": `{"status":"PREVISTO"}`,
		"tipo":   `{"tipo":"entrada"}`,
		"banco":  `{"banco":"NUBANK"}`,
		"data":   `{"dataEvento":"2026-06-01T10:00:00.000Z"}`,
	} {
		c, _ = chamar(t, "own", "PUT", caminho, de, corpo)
		esperar(t, "alterar "+nome, c, 409)
	}
	c, _ = chamar(t, "own", "DELETE", caminho, de, "")
	esperar(t, "excluir em mes conferido", c, 409)
	if f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono")["valor"] != 10.0 {
		t.Error("o lancamento nao pode ter mudado")
	}

	// mover um lancamento de fora PARA dentro do mes conferido tambem e barrado
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-05-01T12:00:00.000Z", "ITAU", 7))
	esperar(t, "criar em maio", c, 201)
}

func TestImportacaoRejeitaMesConferido(t *testing.T) {
	relogioFixo(t)
	cenario(t)
	conferir(t, "own", "ITAU", "2026-03", 0)
	lote := `{"itens":[` + corpoTx("2026-02-01T10:00:00.000Z", "ITAU", 1) + `,` + corpoTx("2026-06-01T10:00:00.000Z", "ITAU", 1) + `]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "importar", c, 200)
	if d["gravados"] != 1.0 {
		t.Errorf("so o de junho deveria entrar, gravados=%v", d["gravados"])
	}
	if rej, _ := d["rejeitados"].([]any); len(rej) != 1 {
		t.Errorf("o de fevereiro deveria ser rejeitado: %v", d["rejeitados"])
	}
}

func TestEscritaInvalidaCacheEBarraCacheVelho(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	c, d := chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil,
		`{"banco":"ITAU","version":0,"base":"100|2026-01-01","caches":{"2026-01":90,"2026-02":80,"2026-10":70}}`)
	esperar(t, "gravar cache", c, 200)
	caches, _ := d["caches"].(map[string]any)
	if len(caches) != 2 || caches["2026-10"] != nil {
		t.Errorf("o mes em andamento nao pode virar cache: %v", caches)
	}

	// um lancamento novo em fevereiro: cache de fevereiro em diante deixa de valer e a version muda
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-02-20T12:00:00.000Z", "ITAU", 3))
	esperar(t, "lancar em fevereiro", c, 201)
	st := f.ler(t, pkEspaco, "SALDO#ITAU")
	if st["invalidoDe"] != "2026-02" || st["version"] != 2.0 {
		t.Errorf("esperado invalidoDe=2026-02 e version=2, veio %v / %v", st["invalidoDe"], st["version"])
	}
	// uma escrita mais antiga puxa o limite para tras; uma mais nova nao o empurra para frente
	chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-01-20T12:00:00.000Z", "ITAU", 3))
	chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-07-20T12:00:00.000Z", "ITAU", 3))
	if got := f.ler(t, pkEspaco, "SALDO#ITAU")["invalidoDe"]; got != "2026-01" {
		t.Errorf("invalidoDe deveria ser o mes mais antigo alterado, veio %v", got)
	}

	// cache calculado antes da escrita (version 1) nao pode ser gravado
	c, d = chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":1,"base":"x","caches":{"2026-01":1}}`)
	esperar(t, "cache velho", c, 409)
	if atual, _ := d["atual"].(map[string]any); atual["version"] != 4.0 {
		t.Errorf("o 409 deveria trazer o estado atual: %v", d["atual"])
	}
	// com a version certa, grava e limpa o invalidoDe
	c, _ = chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":4,"base":"x","caches":{"2026-01":1}}`)
	esperar(t, "cache atual", c, 200)
	if f.ler(t, pkEspaco, "SALDO#ITAU")["invalidoDe"] != nil {
		t.Error("gravar o cache novo deveria limpar o invalidoDe")
	}
}

func TestEditarTextoNaoInvalidaCache(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":0,"base":"x","caches":{"2026-01":90}}`)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"descricao":"outro texto"}`)
	esperar(t, "editar texto", c, 200)
	if st := f.ler(t, pkEspaco, "SALDO#ITAU"); st["invalidoDe"] != nil || st["version"] != 1.0 {
		t.Errorf("editar texto nao deveria mexer no cache: %v", st)
	}
}

func TestValidacoesDaConferencia(t *testing.T) {
	relogioFixo(t)
	cenario(t)
	c, _ := conferir(t, "mem", "ITAU", "2026-03", 0)
	esperar(t, "member nao confere", c, 403)
	c, _ = conferir(t, "own", "ITAU", "2026-11", 0)
	esperar(t, "mes futuro", c, 400)
	c, _ = conferir(t, "own", "ITAU", "03/2026", 0)
	esperar(t, "mes mal formado", c, 400)
	c, _ = chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil,
		`{"banco":"ITAU","mes":"2026-03","version":0,"saldo":100,"real":99.5}`)
	esperar(t, "nao bate com o extrato", c, 400)
	c, _ = conferir(t, "own", "ITAU", "2026-10", 0)
	esperar(t, "o mes atual pode ser conferido", c, 200)
	c, _ = conferir(t, "own", "ITAU", "2026-03", 1)
	esperar(t, "conferir mes anterior ao ultimo conferido", c, 409)
	c, d := conferir(t, "own", "ITAU", "2026-09", 0)
	esperar(t, "version velha", c, 409)
	if d["atual"] == nil {
		t.Error("o 409 deveria trazer o estado atual")
	}
}

func TestReabrirSoOMaisRecente(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	conferir(t, "own", "ITAU", "2026-02", 0)
	conferir(t, "own", "ITAU", "2026-03", 1)
	reabrir := func(quem, mes string) int {
		c, _ := chamar(t, quem, "POST", "spaces/s1/fechamentos/reabrir", nil, fmt.Sprintf(`{"banco":"ITAU","mes":"%s"}`, mes))
		return c
	}
	esperar(t, "member nao reabre", reabrir("mem", "2026-03"), 403)
	esperar(t, "reabrir fevereiro com marco conferido", reabrir("own", "2026-02"), 409)
	esperar(t, "reabrir mes nao conferido", reabrir("own", "2026-05"), 409)
	esperar(t, "reabrir marco", reabrir("own", "2026-03"), 200)

	c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-03-10T12:00:00.000Z", "ITAU", 5))
	esperar(t, "marco liberado", c, 201)
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-02-10T12:00:00.000Z", "ITAU", 5))
	esperar(t, "fevereiro continua travado", c, 409)
	esperar(t, "reabrir fevereiro", reabrir("own", "2026-02"), 200)
	esperar(t, "nada mais para reabrir", reabrir("own", "2026-02"), 404)
	if conf, _ := f.ler(t, pkEspaco, "SALDO#ITAU")["conferidos"].(map[string]any); len(conf) != 0 {
		t.Errorf("nao deveria sobrar conferencia: %v", conf)
	}
}

func TestMemberLeFechamentosMasNaoEscreve(t *testing.T) {
	relogioFixo(t)
	cenario(t)
	conferir(t, "own", "ITAU", "2026-03", 0)
	c, d := chamar(t, "mem", "GET", "spaces/s1/fechamentos", nil, "")
	esperar(t, "member le", c, 200)
	itens, _ := d["itens"].([]any)
	if len(itens) != 1 || d["mesAtual"] != "2026-10" {
		t.Errorf("leitura inesperada: %v", d)
	}
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":1,"caches":{}}`)
	esperar(t, "member nao grava cache", c, 403)
	c, _ = chamar(t, "estranho", "GET", "spaces/s1/fechamentos", nil, "")
	esperar(t, "estranho nao le", c, 403)
}

func TestHistoricoAnteriorAoInicioNaoTravaNemInvalida(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 1.0,
		"porBanco": map[string]any{"ITAU": map[string]any{"saldoInicial": 100.0, "dataInicio": "2026-01-05"}}})
	chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":0,"base":"x","caches":{"2026-01":90,"2026-02":80}}`)
	c, _ := conferir(t, "own", "ITAU", "2026-03", 1)
	esperar(t, "conferir marco", c, 200)
	antes := f.ler(t, pkEspaco, "SALDO#ITAU")

	// 2025 e historico: entra mesmo com marco conferido, inclusive importando em lote
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2025-12-20T12:00:00.000Z", "ITAU", 9))
	esperar(t, "criar em 2025", c, 201)
	lote := `{"itens":[` + corpoTx("2025-06-01T10:00:00.000Z", "ITAU", 1) + `,` + corpoTx("2025-07-01T10:00:00.000Z", "ITAU", 1) + `]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "importar 2025", c, 200)
	if d["gravados"] != 2.0 {
		t.Errorf("os dois de 2025 deveriam entrar: %v", d)
	}
	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"descricao":"x"}`)
	esperar(t, "editar texto em janeiro", c, 200)
	depois := f.ler(t, pkEspaco, "SALDO#ITAU")
	if depois["version"] != antes["version"] || depois["invalidoDe"] != nil {
		t.Errorf("historico anterior ao inicio nao pode invalidar o cache: %v -> %v", antes, depois)
	}

	// do mes da data de inicio em diante continua travado
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-01-10T12:00:00.000Z", "ITAU", 9))
	esperar(t, "criar em janeiro de 2026", c, 409)
	// e outro banco sem data de inicio configurada nao ganha excecao
	chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"NUBANK","version":0,"base":"x","caches":{}}`)
	conferir(t, "own", "NUBANK", "2026-03", 1)
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2025-12-20T12:00:00.000Z", "NUBANK", 9))
	esperar(t, "sem data de inicio, tudo ate o conferido trava", c, 409)
}
