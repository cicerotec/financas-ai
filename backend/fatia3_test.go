package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fatia 3 do item 32 (.ai/specs/arquitetura-camadas/spec-fatia-3.md): o fechamento e a trava de mes migraram para as
// camadas sem mudar o contrato HTTP. Os testes de fechamento_test.go e fechamento_auto_test.go seguem como a rede da trava.

func fecha(banco string, ver float64, saldo, real float64) string {
	return fmt.Sprintf(`{"banco":"%s","mes":"2026-02","version":%v,"saldo":%v,"real":%v}`, banco, ver, saldo, real)
}

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

// F3-T2: conflito de versao nas tres escritas e a ordem de conferir/reabrir.
func TestF3ConflitosOrdemEMesNaoConferido(t *testing.T) {
	cenario(t)
	c, d := chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":7,"caches":{}}`)
	esperar(t, "PUT com version errada (sem item)", c, 409)
	atual, _ := d["atual"].(map[string]any)
	if d["erro"] != "alterado por outra pessoa" || atual["version"] != 0.0 {
		t.Errorf("409 sem item deve trazer atual {version:0}: %v", d)
	}
	c, d = chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":0,"caches":{"2026-01":10}}`)
	esperar(t, "primeiro PUT", c, 200)
	c, _ = chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":0,"caches":{}}`)
	esperar(t, "PUT com version velha", c, 409)
	c, _ = chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil, fecha("ITAU", 0, 100, 100))
	esperar(t, "conferir com version velha", c, 409)

	c, _ = chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil, fecha("ITAU", 1, 100, 100))
	esperar(t, "conferir fevereiro", c, 200)
	c, d = chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil, `{"banco":"ITAU","mes":"2026-01","version":2,"saldo":1,"real":1}`)
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

// F3-T3: validacoes, com a borda da diferenca de 0,005 e o descarte do mes em andamento e dos futuros.
func TestF3ValidacoesEBordas(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"version":0,"caches":{}}`)
	esperar(t, "sem banco", c, 400)
	if d["erro"] != "banco obrigatorio" {
		t.Errorf("erro = %v", d["erro"])
	}
	for _, corpo := range []string{`{"banco":"ITAU","caches":{"2026-13":1}}`, `{"banco":"ITAU","caches":{"x":1}}`, `{"banco":"ITAU","caches":{"2026-01":"a"}}`} {
		c, d = chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, corpo)
		esperar(t, "cache invalido "+corpo, c, 400)
		if d["erro"] != "caches invalidos" {
			t.Errorf("erro = %v", d["erro"])
		}
	}
	// hoje e 05/02/2026: janeiro entra no cache; fevereiro (em andamento) e marco (futuro) sao descartados
	c, d = chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":0,"base":"2026-01","caches":{"2026-01":10,"2026-02":20,"2026-03":30}}`)
	esperar(t, "gravar cache", c, 200)
	caches, _ := d["caches"].(map[string]any)
	if len(caches) != 1 || caches["2026-01"] != 10.0 || d["base"] != "2026-01" {
		t.Errorf("so janeiro deveria entrar: %v", d)
	}
	if got := f.ler(t, pkEspaco, "SALDO#ITAU"); got == nil || got["PK"] != pkEspaco {
		t.Errorf("item gravado = %v", got)
	}

	ver := 1.0
	for _, c := range []struct {
		nome       string
		saldo, rel float64
		quer       int
	}{{"diferenca 0,004 passa", 100, 100.004, 200}, {"diferenca 0,006 nao passa", 100, 100.006, 400}} {
		// (a borda exata de 0,005 depende do ponto flutuante: 100,005 - 100 da 0,00499...; por isso 0,004 e 0,006)
		st, dd := chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil, fecha("ITAU", ver, c.saldo, c.rel))
		esperar(t, c.nome, st, c.quer)
		if st == 400 && dd["erro"] != "o saldo calculado nao bate com o extrato: lance o ajuste antes de conferir" {
			t.Errorf("%s: %v", c.nome, dd["erro"])
		}
	}
	c2, d2 := chamar(t, "own", "POST", "spaces/s1/fechamentos/conferir", nil, `{"banco":"ITAU","mes":"2026-03","version":2,"saldo":1,"real":1}`)
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
