package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// Fatia 1 do item 32 (.ai/specs/arquitetura-camadas/spec-fatia-1.md): espacos e leitura de lancamentos migraram para as
// camadas sem mudar o contrato HTTP.

// F1-T1: GET /me com META, sem META, sem vinculo e com erro de leitura.
func TestF1GetMe(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": "USER#own", "SK": "SPACE#s2", "role": "member", "apelido": "Ana"}) // s2 nao tem META
	st, d := chamar(t, "own", "GET", "me", nil, "")
	esperar(t, "me", st, 200)
	if d["sub"] != "own" {
		t.Errorf("sub = %v", d["sub"])
	}
	nomes := map[string]string{}
	for _, e := range d["espacos"].([]any) {
		m := e.(map[string]any)
		nomes[m["id"].(string)] = m["nome"].(string)
		if _, ok := m["telegramChatId"]; !ok {
			t.Errorf("a resposta perdeu a chave telegramChatId: %v", m)
		}
	}
	if nomes["s1"] != "Familia" || nomes["s2"] != "s2" || len(nomes) != 2 {
		t.Errorf("nomes = %v (com META usa o nome; sem META usa o id)", nomes)
	}

	st, d = chamar(t, "ninguem", "GET", "me", nil, "")
	esperar(t, "sem vinculo", st, 200)
	if l, ok := d["espacos"].([]any); !ok || len(l) != 0 {
		t.Errorf("sem vinculo: espacos = %v, esperado []", d["espacos"])
	}

	f.falharLeitura = errors.New("dynamodb fora do ar")
	st, d = chamar(t, "own", "GET", "me", nil, "")
	esperar(t, "erro de leitura", st, 500)
	if d["erro"] != "erro interno" {
		t.Errorf("erro = %v", d["erro"])
	}
}

// F1-T2: filtros, limite e paginacao completa por cursor, sem repetir nem perder item.
func TestF1ListarTxFiltrosEPaginacao(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "TX#2026-01-04T10:00:00.000Z#extra", "id": "extra", "dataEvento": "2026-01-04T10:00:00.000Z",
		"status": "PREVISTO", "tipo": "saida", "valor": 5.0, "banco": "NUBANK", "descricao": "Mercado do bairro", "nota": "Compra GRANDE"})

	st, d := chamar(t, "own", "GET", "spaces/s1/tx", nil, "")
	esperar(t, "lista", st, 200)
	if got := idsDe(d); got != "extra,idlegado,idmembro,iddono" {
		t.Errorf("ordem desc = %s", got)
	}
	if d["proximo"] != "" {
		t.Errorf("proximo = %v, esperado vazio quando acabou", d["proximo"])
	}

	casos := []struct {
		nome string
		qs   map[string]string
		quer string
	}{
		{"asc", map[string]string{"ordem": "asc"}, "iddono,idmembro,idlegado,extra"},
		{"de", map[string]string{"de": "2026-01-03"}, "extra,idlegado"},
		{"ate", map[string]string{"ate": "2026-01-02T99"}, "idmembro,iddono"},
		{"banco", map[string]string{"banco": "NUBANK"}, "extra"},
		{"status", map[string]string{"status": "PREVISTO"}, "extra"},
		{"texto na descricao", map[string]string{"q": "BAIRRO"}, "extra"},
		{"texto na nota", map[string]string{"q": "grande"}, "extra"},
		{"texto sem resultado", map[string]string{"q": "nada"}, ""},
	}
	for _, c := range casos {
		st, d := chamar(t, "own", "GET", "spaces/s1/tx", c.qs, "")
		esperar(t, c.nome, st, 200)
		if got := idsDe(d); got != c.quer {
			t.Errorf("%s: %q, esperado %q", c.nome, got, c.quer)
		}
	}

	// limite: 1 a 1000 vale; fora disso (ou nao numerico) volta ao padrao 200
	for _, lim := range []string{"0", "1001", "abc", "-3"} {
		_, d := chamar(t, "own", "GET", "spaces/s1/tx", map[string]string{"limite": lim}, "")
		if len(itensDe(d)) != 4 {
			t.Errorf("limite=%s deveria cair no padrao e devolver os 4", lim)
		}
	}

	// paginacao: uma por pagina, todas as paginas, cada item uma vez
	var vistos []string
	cursor := ""
	for pagina := 0; pagina < 10; pagina++ {
		qs := map[string]string{"limite": "1", "ordem": "asc"}
		if cursor != "" {
			qs["cursor"] = cursor
		}
		_, d := chamar(t, "own", "GET", "spaces/s1/tx", qs, "")
		for _, it := range itensDe(d) {
			vistos = append(vistos, it["id"].(string))
		}
		cursor, _ = d["proximo"].(string)
		if cursor == "" {
			break
		}
	}
	if got := strings.Join(vistos, ","); got != "iddono,idmembro,idlegado,extra" {
		t.Errorf("paginacao = %s", got)
	}
}

// F1-T3: excluidos logicamente nao saem e nada interno vaza.
func TestF1ListarNaoMostraExcluidosNemChavesInternas(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "TX#2026-01-05T10:00:00.000Z#morto", "id": "morto", "dataEvento": "2026-01-05T10:00:00.000Z",
		"status": "PAGO", "tipo": "saida", "valor": 1.0, "banco": "ITAU", "excluidoEm": "2026-01-06T10:00:00.000Z"})
	st, d := chamar(t, "own", "GET", "spaces/s1/tx", nil, "")
	esperar(t, "lista", st, 200)
	for _, it := range itensDe(d) {
		if it["id"] == "morto" {
			t.Fatal("o excluido apareceu na lista")
		}
	}
	for _, it := range d["itens"].([]any) {
		m := it.(map[string]any)
		for _, k := range []string{"PK", "SK", "excluidoEm"} {
			if _, ok := m[k]; ok {
				t.Errorf("a chave interna %s vazou: %v", k, m)
			}
		}
	}
}

// F1-T4: cursor invalido continua 500 "erro ao listar" (comportamento preservado; o certo seria 400, fica no BACKLOG).
func TestF1CursorInvalidoContinuaDando500(t *testing.T) {
	cenario(t)
	for _, c := range []string{"@@@", "QUJD"} { // nao e base64 / e base64 mas nao e uma SK de lancamento
		st, d := chamar(t, "own", "GET", "spaces/s1/tx", map[string]string{"cursor": c}, "")
		esperar(t, "cursor "+c, st, 500)
		if d["erro"] != "erro ao listar" {
			t.Errorf("erro = %v", d["erro"])
		}
	}
}

// F1-T5: o papel vem do repository: dono, membro, sem vinculo (403) e erro de leitura (500).
func TestF1PapelDoEspaco(t *testing.T) {
	f := cenario(t)
	st, _ := chamar(t, "own", "GET", "spaces/s1/tx", nil, "")
	esperar(t, "dono", st, 200)
	st, _ = chamar(t, "mem", "GET", "spaces/s1/tx", nil, "")
	esperar(t, "membro", st, 200)
	st, d := chamar(t, "estranho", "GET", "spaces/s1/tx", nil, "")
	esperar(t, "sem vinculo", st, 403)
	if d["erro"] != "sem acesso a esse espaco" {
		t.Errorf("erro = %v", d["erro"])
	}
	f.falharLeitura = errors.New("fora do ar")
	st, d = chamar(t, "own", "GET", "spaces/s1/tx", nil, "")
	esperar(t, "erro ao ler o papel", st, 500)
	if d["erro"] != "erro interno" {
		t.Errorf("erro = %v", d["erro"])
	}
}

// F1-T6: o Servidor e montado uma vez e a raiz nao le espacos nem lancamentos no banco.
func TestF1RaizSoLigaAsPecasMigradas(t *testing.T) {
	ler := func(nome string) string {
		b, err := os.ReadFile(nome)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	main, servidor := ler("main.go")+ler("roteador.go"), ler("servidor.go")
	if strings.Contains(main, "NewLancamentos(") || strings.Contains(main, "NewEspacos(") {
		t.Error("main.go monta service a cada requisicao: a montagem e do novoServidor (servidor.go), uma vez")
	}
	if strings.Count(servidor, "func novoServidor(") != 1 {
		t.Error("deve existir um unico montador: novoServidor")
	}
	for _, nome := range []string{"main.go", "roteador.go", "store.go"} {
		if fonte := ler(nome); strings.Contains(fonte, "dynamodb.QueryInput") || strings.Contains(fonte, "func papel(") || strings.Contains(fonte, "func listarTx(") {
			t.Errorf("%s ainda le espacos ou lancamentos direto do banco (migrou para repository/)", nome)
		}
	}
}
