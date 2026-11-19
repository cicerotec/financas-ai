package main

// Prova que .ai/api.md e a lista de rotas do codigo andam juntos (ver .ai/api.md, secao "Como este indice se mantem"). Arquivo: backend/api_doc_test.go.
// A classificacao e a de verdade do servidor: rotular (roteador.go), a mesma que da o rotulo do log. O teste nao tem lista propria de rotas:
//  1. IDA: cada linha do indice tem de ser reconhecida pelo codigo, com a mesma acao e as mesmas decisoes de papel de auth.Permitido;
//  2. VOLTA: toda rota que o codigo reconhece tem de estar no indice. O universo testado nasce do proprio codigo (auth/authz.go e roteador.go):
//     para cada posicao do caminho, os literais com que o codigo compara (rest[0] == "tx", switch p[2] { case "futuros": }, as chaves de cfgNomes)
//     mais um valor qualquer ("ID"). Rota nova que o codigo passe a reconhecer aparece sozinha, sem ninguem lembrar de listar.
// Limite conhecido: classificacao feita por outro mecanismo que nao a comparacao de rest[i] ou p[i] com um literal (HasPrefix, regex, tabela)
// nao entra na VOLTA. Quem introduzir um mecanismo assim estende apiColher() no mesmo PR.

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"financas/auth"
)

const (
	apiArquivoIndice = "../.ai/api.md"
	apiSentinela     = "ID" // qualquer segmento que nao seja um literal conhecido
	apiMaxSegmentos  = 8    // /spaces/{sid}/cartoes/{banco}/faturas/{ym}/conferir tem 7; sobra uma
)

var apiMetodos = []string{"GET", "POST", "PUT", "DELETE"}

// apiEhPublica diz se a rota vale sem token. Enquanto nao existe rota publica, e sempre falso.
// Na entrega da rota publica do MFA (AD-1) passa a ser: return ok com auth.RotaPublica(m, p).
func apiEhPublica(m string, p []string) bool { return false }

// classificar e a unica porta para o codigo: a mesma do log.
func apiClassificar(m string, p []string) (string, bool) {
	rt := rotular(m, p)
	if rt.Acao == acaoDesconhecida {
		return "", false
	}
	return rt.Acao, true
}

type apiLinha struct {
	metodo, caminho, acao, owner, member, token, spec string
	n                                                 int // linha no arquivo
}

// lerIndice devolve as linhas de tabela do indice cuja primeira coluna e um metodo HTTP.
func apiLerIndice(t *testing.T) []apiLinha {
	t.Helper()
	f, err := os.Open(apiArquivoIndice)
	if err != nil {
		t.Fatalf("o indice %s nao existe ou nao abre: %v", apiArquivoIndice, err)
	}
	defer f.Close()
	var out []apiLinha
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		l := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(l, "|") {
			continue
		}
		c := strings.Split(strings.Trim(strings.ReplaceAll(l, `\|`, "§"), "|"), "|") // \| dentro de uma celula e uma alternativa (a|b), nao outra coluna
		for i := range c {
			c[i] = strings.ReplaceAll(strings.Trim(strings.TrimSpace(c[i]), "`"), "§", "|")
		}
		if len(c) != 7 || !apiContem(apiMetodos, c[0]) {
			continue // cabecalho, separador e as tabelas de outro assunto
		}
		out = append(out, apiLinha{c[0], c[1], c[2], c[3], c[4], c[5], c[6], n})
	}
	return out
}

func apiContem(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// expandir transforma o caminho do indice em caminhos concretos: {x} vira o valor qualquer e a|b|c vira tres caminhos.
func apiExpandir(caminho string) [][]string {
	segs := strings.Split(strings.Trim(caminho, "/"), "/")
	out := [][]string{{}}
	for _, s := range segs {
		var alt []string
		switch {
		case strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}"):
			alt = []string{apiSentinela}
		default:
			alt = strings.Split(s, "|")
		}
		var prox [][]string
		for _, base := range out {
			for _, a := range alt {
				prox = append(prox, append(append([]string{}, base...), a))
			}
		}
		out = prox
	}
	return out
}

func apiChave(m string, p []string) string { return m + " /" + strings.Join(p, "/") }

// decisao escreve a decisao de papel como o indice a escreve.
func apiDecisao(papel, acao string) string {
	switch auth.Permitido(papel, auth.Acao(acao)) {
	case auth.Liberado:
		return "sim"
	case auth.SoDono:
		return "só o que criou"
	}
	return "não"
}

// esperada monta as colunas que o codigo manda para uma rota (para conferir e para sugerir a linha a colar).
func apiEsperada(m string, p []string, acao string) (owner, member, tok string) {
	tok = "ID token"
	if apiEhPublica(m, p) {
		tok = "nenhum"
	}
	if len(p) >= 3 && p[0] == "spaces" {
		return apiDecisao(auth.PapelOwner, acao), apiDecisao(auth.PapelMember, acao), tok
	}
	return "—", "—", tok // fora de /spaces/{sid} nao ha papel: vale o sub do token (ou ninguem, se for publica)
}

func apiSugestao(m string, p []string, acao string) string {
	segs := append([]string{}, p...)
	for i, s := range segs {
		if s == apiSentinela {
			segs[i] = "{id}"
			if i == 1 && segs[0] == "spaces" {
				segs[i] = "{sid}"
			}
		}
	}
	o, mb, tk := apiEsperada(m, p, acao)
	return fmt.Sprintf("| %s | `/%s` | `%s` | %s | %s | %s | (spec de origem) |", m, strings.Join(segs, "/"), acao, o, mb, tk)
}

// ---- universo da VOLTA, colhido do proprio codigo ----

type apiVocab map[int]map[string]bool

func (v apiVocab) add(i int, s string) {
	if s == "" {
		return
	}
	if v[i] == nil {
		v[i] = map[string]bool{}
	}
	v[i][s] = true
}

// colher le um arquivo Go e anota, por posicao do caminho inteiro, cada literal comparado com <nome>[i].
// desloc diz quanto somar ao indice de cada variavel: rest (o que vem depois de /spaces/{sid}) soma 2; p (o caminho inteiro) soma 0.
func apiColher(t *testing.T, arquivo string, desloc map[string]int, v apiVocab) {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), arquivo, nil, 0)
	if err != nil {
		t.Fatalf("nao leu %s: %v", arquivo, err)
	}
	indice := func(e ast.Expr) (int, bool) { // rest[0], p[1]
		ix, ok := e.(*ast.IndexExpr)
		if !ok {
			return 0, false
		}
		id, ok := ix.X.(*ast.Ident)
		if !ok {
			return 0, false
		}
		d, ok := desloc[id.Name]
		lit, ok2 := ix.Index.(*ast.BasicLit)
		if !ok || !ok2 || lit.Kind != token.INT {
			return 0, false
		}
		n, _ := strconv.Atoi(lit.Value)
		return n + d, true
	}
	texto := func(e ast.Expr) (string, bool) {
		lit, ok := e.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(lit.Value)
		return s, err == nil
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BinaryExpr: // rest[0] == "tx"   |   p[0] != "spaces"
			if x.Op != token.EQL && x.Op != token.NEQ {
				return true
			}
			for _, par := range [][2]ast.Expr{{x.X, x.Y}, {x.Y, x.X}} {
				if i, ok := indice(par[0]); ok {
					if s, ok := texto(par[1]); ok {
						v.add(i, s)
					}
				}
			}
		case *ast.SwitchStmt: // switch p[2] { case "futuros", "status": }
			if x.Tag == nil {
				return true
			}
			i, ok := indice(x.Tag)
			if !ok {
				return true
			}
			for _, c := range x.Body.List {
				for _, e := range c.(*ast.CaseClause).List {
					if s, ok := texto(e); ok {
						v.add(i, s)
					}
				}
			}
		}
		return true
	})
	// as chaves de cfgNomes (cfgNomes[rest[1]]) sao literais da posicao 3
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "cfgNomes" || len(vs.Values) != 1 {
			return true
		}
		if cl, ok := vs.Values[0].(*ast.CompositeLit); ok {
			for _, e := range cl.Elts {
				if kv, ok := e.(*ast.KeyValueExpr); ok {
					if s, ok := texto(kv.Key); ok {
						v.add(3, s)
					}
				}
			}
		}
		return false
	})
}

// apiCanonica troca por apiSentinela todo segmento que e so um parametro: se com o valor qualquer o codigo reconhece a mesma rota com a mesma acao,
// o literal era um id que por acaso tem o nome de um literal de outra rota (DELETE /tx/batch e o id "batch", nao uma rota propria).
func apiCanonica(m string, p []string, acao string) []string {
	c := append([]string{}, p...)
	for i, s := range c {
		if s == apiSentinela {
			continue
		}
		c[i] = apiSentinela
		if a, ok := apiClassificar(m, c); !ok || a != acao {
			c[i] = s
		}
	}
	return c
}

// reconhecidas devolve toda rota (metodo + caminho com o valor qualquer) que o codigo reconhece dentro do universo colhido.
func apiReconhecidas(t *testing.T) map[string]string {
	t.Helper()
	v := apiVocab{}
	apiColher(t, "auth/authz.go", map[string]int{"rest": 2, "p": 0}, v)
	apiColher(t, "roteador.go", map[string]int{"p": 0}, v)
	universo := make([][]string, apiMaxSegmentos)
	for i := range universo {
		universo[i] = []string{apiSentinela}
		for s := range v[i] {
			universo[i] = append(universo[i], s)
		}
		sort.Strings(universo[i][1:])
	}
	achadas := map[string]string{}
	caminho := make([]string, 0, apiMaxSegmentos)
	var anda func(i int)
	anda = func(i int) {
		if len(caminho) > 0 {
			for _, m := range apiMetodos {
				if a, ok := apiClassificar(m, caminho); ok {
					c := apiCanonica(m, caminho, a)
					achadas[apiChave(m, c)] = a
				}
			}
		}
		if i == apiMaxSegmentos {
			return
		}
		for _, s := range universo[i] {
			caminho = append(caminho, s)
			anda(i + 1)
			caminho = caminho[:len(caminho)-1]
		}
	}
	anda(0)
	return achadas
}

func TestIndiceDeRotas(t *testing.T) {
	doc := apiLerIndice(t)
	achadas := apiReconhecidas(t)

	if len(achadas) == 0 {
		t.Fatal("a varredura nao achou nenhuma rota: o colhedor de literais parou de entender o codigo")
	}

	// IDA: cada linha do indice existe no codigo, com a mesma acao e as mesmas decisoes
	noIndice := map[string]int{}
	for _, l := range doc {
		for _, p := range apiExpandir(l.caminho) {
			k := apiChave(l.metodo, p)
			if outra, dup := noIndice[k]; dup {
				t.Errorf("api.md linha %d: %s ja esta na linha %d", l.n, k, outra)
			}
			noIndice[k] = l.n
			acao, ok := apiClassificar(l.metodo, p)
			if !ok {
				t.Errorf("api.md linha %d: %s nao existe no codigo (rotular nao a reconhece)", l.n, k)
				continue
			}
			if acao != l.acao {
				t.Errorf("api.md linha %d: %s tem a acao %q no codigo, e o indice diz %q", l.n, k, acao, l.acao)
			}
			o, m, tk := apiEsperada(l.metodo, p, acao)
			if l.owner != o || l.member != m || l.token != tk {
				t.Errorf("api.md linha %d: %s tem owner=%q member=%q token=%q no codigo, e o indice diz owner=%q member=%q token=%q",
					l.n, k, o, m, tk, l.owner, l.member, l.token)
			}
			if l.spec != "—" {
				if _, err := os.Stat(filepath.Join("..", ".ai", filepath.FromSlash(l.spec))); err != nil {
					t.Errorf("api.md linha %d: a spec %q nao existe em .ai/", l.n, l.spec)
				}
			}
		}
	}

	// VOLTA: toda rota que o codigo reconhece esta no indice
	var faltam []string
	for k, acao := range achadas {
		if _, ok := noIndice[k]; ok {
			continue
		}
		partes := strings.SplitN(k, " /", 2)
		faltam = append(faltam, apiSugestao(partes[0], strings.Split(partes[1], "/"), acao))
	}
	if len(faltam) > 0 {
		sort.Strings(faltam)
		t.Errorf("%d rota(s) existem no codigo e nao estao em .ai/api.md. Cole as linhas abaixo (ajuste o nome dos {parametros} e a coluna Spec):\n%s",
			len(faltam), strings.Join(faltam, "\n"))
	}
}
