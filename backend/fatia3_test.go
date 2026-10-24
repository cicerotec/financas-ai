package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fatia 3 do item 32 (.ai/specs/arquitetura-camadas/spec-fatia-3.md): o fechamento e a trava de mes migraram para as
// camadas sem mudar o contrato HTTP. Os testes de fechamento_test.go e fechamento_auto_test.go seguem como a rede da trava.

// F3-T1: ler os fechamentos devolve os itens sem chaves internas, o mes atual do relogio e [] sem itens.
func TestF3ListarFechamentos(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "own", "GET", "spaces/s1/fechamentos", nil, "")
	esperar(t, "sem itens", c, 200)
	if l, ok := d["itens"].([]any); !ok || len(l) != 0 || d["mesAtual"] != "2026-02" {
		t.Errorf("sem itens: %v", d)
	}
	f.colocar(t, doc{"PK": pkEspaco, "SK": "SALDO#ITAU", "banco": "ITAU", "version": 2.0, "caches": map[string]any{"2026-01": 10.0}})
	relogioEm(t, 2026, 10, 5, 15, 0)
	_, d = chamar(t, "mem", "GET", "spaces/s1/fechamentos", nil, "")
	itens := d["itens"].([]any)
	if len(itens) != 1 || itens[0].(map[string]any)["PK"] != nil || itens[0].(map[string]any)["SK"] != nil || d["mesAtual"] != "2026-10" {
		t.Errorf("com item: %v", d)
	}
	f.falharLeitura = errors.New("fora do ar")
	c, _ = chamar(t, "own", "GET", "spaces/s1/fechamentos", nil, "") // a leitura do papel falha primeiro
	esperar(t, "erro de leitura", c, 500)
}

// F3-T2: a ordem de conferir/reabrir. (Item 40: o PUT do cache nao grava mais nada e nao ha mais conflito de version na conferencia,
// porque o servidor compara o extrato com o saldo que ele calculou.)
func TestF3ConflitosOrdemEMesNaoConferido(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":7,"caches":{"2026-01":10}}`)
	esperar(t, "PUT do cache e ignorado, sem conflito", c, 200)
	if d["version"] != 0.0 {
		t.Errorf("sem item, a resposta e {version:0}: %v", d)
	}
	if f.ler(t, pkEspaco, "SALDO#ITAU") != nil {
		t.Error("o PUT do cache nao pode gravar nada")
	}

	c, _ = conferir(t, "own", "ITAU", "2026-02", 0)
	esperar(t, "conferir fevereiro", c, 200)
	c, d = chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil, `{"meio":"ITAU","mes":"2026-01","real":1}`)
	esperar(t, "conferir mes anterior ao ultimo", c, 409)
	if d["codigo"] != "ordem" || d["erro"] != "o mes 2026-01 nao pode ser conferido: o ultimo conferido e 2026-02" {
		t.Errorf("409 de ordem = %v", d)
	}
	c, d = chamar(t, "own", "POST", "spaces/s1/fechamentos/reabrir", nil, `{"banco":"ITAU","mes":"2026-01"}`)
	esperar(t, "reabrir mes que nao e o ultimo", c, 409)
	if d["codigo"] != "ordem" || d["erro"] != "reabra primeiro o mes 2026-02 (o mais recente conferido)" {
		t.Errorf("409 de ordem ao reabrir = %v", d)
	}
	c, d = chamar(t, "own", "POST", "spaces/s1/fechamentos/reabrir", nil, `{"banco":"NUBANK","mes":"2026-02"}`)
	esperar(t, "reabrir sem conferencia", c, 404)
	if d["erro"] != "esse mes nao esta conferido" {
		t.Errorf("404 = %v", d)
	}
	c, d = chamar(t, "own", "POST", "spaces/s1/fechamentos/reabrir", nil, `{"banco":"ITAU","mes":"2026-02"}`)
	esperar(t, "reabrir o ultimo", c, 200)
	if conf, _ := d["conferidos"].(map[string]any); len(conf) != 0 {
		t.Errorf("o mes deveria sair de conferidos: %v", d)
	}
}

// F3-T3: validacoes, com a borda da diferenca de 0,005. (Item 40: o PUT do cache so exige o banco; o resto do corpo e ignorado.)
func TestF3ValidacoesEBordas(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"version":0,"caches":{}}`)
	esperar(t, "sem banco", c, 400)
	if d["erro"] != "banco obrigatorio" {
		t.Errorf("erro = %v", d["erro"])
	}
	for _, corpo := range []string{`{"banco":"ITAU","caches":{"2026-13":1}}`, `{"banco":"ITAU","caches":{"x":1}}`, `{"banco":"ITAU","caches":{"2026-01":"a"}}`} {
		c, _ = chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, corpo)
		esperar(t, "o corpo do cache e ignorado "+corpo, c, 200)
	}
	if f.ler(t, pkEspaco, "SALDO#ITAU") != nil {
		t.Error("nada pode ser gravado pelo PUT do cache")
	}

	// o saldo do ITAU em fevereiro e -30 (tres saidas de 10 em janeiro); a borda e 0,005 de diferenca
	// (a borda exata de 0,005 depende do ponto flutuante: por isso 0,006 e 0,004)
	c, d = chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil, `{"meio":"ITAU","mes":"2026-02","real":-30.006}`)
	esperar(t, "diferenca 0,006 nao passa", c, 409)
	if d["codigo"] != "diferenca" {
		t.Errorf("diferenca 0,006: %v", d)
	}
	c, _ = chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil, `{"meio":"ITAU","mes":"2026-02","real":-30.004}`)
	esperar(t, "diferenca 0,004 passa", c, 200)

	c2, d2 := chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil, `{"meio":"ITAU","mes":"2026-03","real":1}`)
	esperar(t, "mes futuro", c2, 400)
	if d2["erro"] != "so da para conferir o mes atual ou meses anteriores" {
		t.Errorf("erro = %v", d2["erro"])
	}
	c2, _ = chamar(t, "own", "POST", "spaces/s1/fechamentos/reabrir", nil, `{"banco":"ITAU","mes":"fev"}`)
	esperar(t, "reabrir com mes invalido", c2, 400)
}

// F3-T5: a falha ao invalidar o cache nao derruba o lancamento (so vai ao log).
func TestF3FalhaAoInvalidarNaoDerrubaOLancamento(t *testing.T) {
	f := cenario(t)
	f.falharPut = errors.New("fora do ar") // o PutItem do SALDO#ITAU falha; o lancamento e gravado por transacao
	c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-02-10T12:00:00.000Z", "ITAU", 5))
	esperar(t, "criar mesmo sem conseguir invalidar", c, 201)
}

// F3-T7: o relogio e injetado; nao existe a global agora e o service de fechamento nao chama time.Now. (O service de
// lancamentos ainda carimba o "em" das versoes com time.Now; injetar o Relogio nele e uma melhoria separada.)
func TestF3RelogioInjetado(t *testing.T) {
	filepath.Walk(".", func(caminho string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(caminho, ".go") {
			return nil
		}
		p := filepath.ToSlash(caminho)
		b, _ := os.ReadFile(caminho)
		fonte := string(b)
		if p == "service/fechamento.go" && strings.Contains(fonte, "time.Now(") {
			t.Errorf("%s chama time.Now: o service recebe o instante pelo Relogio", p)
		}
		if !strings.Contains(p, "/") && !strings.HasSuffix(p, "_test.go") && strings.Contains(fonte, "agora = time.Now") {
			t.Errorf("%s declara a global agora: o relogio e injetado", p)
		}
		return nil
	})
}

// F3-T8: a raiz perdeu o fechamento.
func TestF3RaizSemOFechamento(t *testing.T) {
	if _, err := os.Stat("fechamento.go"); err == nil {
		t.Error("fechamento.go ainda existe na raiz: migrou para service/, repository/ e controller/")
	}
	for _, nome := range []string{"main.go", "roteador.go", "efeitos.go", "dominio_compat.go"} {
		b, _ := os.ReadFile(nome)
		for _, p := range []string{"salvarCfg", "novasTravas", "invalidarSaldos", "errConflito", "lerSaldo"} {
			if strings.Contains(string(b), p) {
				t.Errorf("%s ainda tem %q", nome, p)
			}
		}
	}
}
