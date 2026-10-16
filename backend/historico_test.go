package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/repository"
	"financas/service"
)

// Fase 1 do historico (.ai/specs/historico/spec.md): toda escrita de lancamento grava a versao junto, e excluir e
// logico. Os ids F1-T* sao os do spec.

const pkHist = "HIST#s1"

// versoesDe devolve as versoes do id gravadas no banco falso, da primeira para a ultima.
func versoesDe(t *testing.T, f *bancoFake, id string) []doc {
	t.Helper()
	prefixo := pkHist + "|TX#" + id + "#"
	var chaves []string
	for k := range f.itens {
		if strings.HasPrefix(k, prefixo) {
			chaves = append(chaves, k)
		}
	}
	sort.Strings(chaves)
	var vs []doc
	for _, k := range chaves {
		var d doc
		if err := attributevalue.UnmarshalMap(f.itens[k], &d); err != nil {
			t.Fatal(err)
		}
		vs = append(vs, d)
	}
	return vs
}

func registroDe(v doc) doc { r, _ := v["registro"].(map[string]any); return r }

func confereVersao(t *testing.T, v doc, numero float64, acao, porSub string) {
	t.Helper()
	if v["versao"] != numero || v["acao"] != acao || v["porSub"] != porSub {
		t.Errorf("versao %v acao %v porSub %q; esperado %v %s %q", v["versao"], v["acao"], v["porSub"], numero, acao, porSub)
	}
	r := registroDe(v)
	if r == nil || r["PK"] != nil || r["SK"] != nil {
		t.Errorf("o registro deve ser o lancamento sem as chaves do banco: %v", r)
	}
}

func chaveTx(data, id string) string { return dePrefixoTx + data + "#" + id }

// F1-T1
func TestCriarGravaOLancamentoEAVersao1(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "mem", "POST", "spaces/s1/tx", nil,
		`{"dataEvento":"2026-02-01T10:00:00.000Z","status":"PAGO","tipo":"saida","valor":5,"banco":"ITAU","descricao":"novo"}`)
	esperar(t, "criar", c, 201)
	id, _ := d["id"].(string)
	if f.ler(t, pkEspaco, chaveTx("2026-02-01T10:00:00.000Z", id)) == nil {
		t.Fatal("o lancamento vivo deveria existir")
	}
	vs := versoesDe(t, f, id)
	if len(vs) != 1 {
		t.Fatalf("esperava 1 versao, vieram %d", len(vs))
	}
	confereVersao(t, vs[0], 1, "insert", "mem")
	if r := registroDe(vs[0]); r["id"] != id || r["valor"] != 5.0 || r["criadoPor"] != "mem" || vs[0]["em"] != r["criadoEm"] {
		t.Errorf("registro da versao 1 = %v (em %v)", r, vs[0]["em"])
	}
}

// F1-T2
func TestEditarComHistoricoGravaAProximaVersao(t *testing.T) {
	f := cenario(t)
	_, d := chamar(t, "own", "POST", "spaces/s1/tx", nil,
		`{"dataEvento":"2026-02-01T10:00:00.000Z","status":"PAGO","tipo":"saida","valor":5,"banco":"ITAU","descricao":"v1"}`)
	id, _ := d["id"].(string)
	qs := map[string]string{"de": "2026-02-01T10:00:00.000Z"}
	c, _ := chamar(t, "own", "PUT", "spaces/s1/tx/"+id, qs, `{"descricao":"v2"}`)
	esperar(t, "editar 1", c, 200)
	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/"+id, qs, `{"descricao":"v3"}`)
	esperar(t, "editar 2", c, 200)
	vs := versoesDe(t, f, id)
	if len(vs) != 3 {
		t.Fatalf("esperava 3 versoes, vieram %d", len(vs))
	}
	confereVersao(t, vs[0], 1, "insert", "own")
	confereVersao(t, vs[1], 2, "update", "own")
	confereVersao(t, vs[2], 3, "update", "own")
	for i, quer := range []string{"v1", "v2", "v3"} {
		if registroDe(vs[i])["descricao"] != quer {
			t.Errorf("versao %d: descricao %v, esperado %s (o registro e o estado depois da acao)", i+1, registroDe(vs[i])["descricao"], quer)
		}
	}
}

// F1-T3
func TestEditarLegadoGravaV1DeAntesEV2DeDepois(t *testing.T) {
	f := cenario(t)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/tx/idmembro", map[string]string{"de": dataMembro}, `{"descricao":"depois"}`)
	esperar(t, "editar legado", c, 200)
	vs := versoesDe(t, f, "idmembro")
	if len(vs) != 2 {
		t.Fatalf("esperava 2 versoes (v1 preguicosa e v2), vieram %d", len(vs))
	}
	confereVersao(t, vs[0], 1, "insert", "") // autor desconhecido
	confereVersao(t, vs[1], 2, "update", "own")
	if registroDe(vs[0])["descricao"] != "original" || registroDe(vs[1])["descricao"] != "depois" {
		t.Errorf("v1 deve ter o estado de antes e v2 o de depois: %v / %v", registroDe(vs[0])["descricao"], registroDe(vs[1])["descricao"])
	}
	if vs[0]["em"] != vs[1]["em"] {
		t.Error("as duas versoes nascem na mesma transacao, com o mesmo instante")
	}
}

// F1-T4
func TestOcultarEAjustarFaturaSaoEdicoes(t *testing.T) {
	f := cenario(t)
	qs := map[string]string{"de": dataDono}
	c, _ := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", qs, `{"oculto":true}`)
	esperar(t, "ocultar", c, 200)
	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/iddono", qs, `{"faturaAjuste":1}`)
	esperar(t, "ajustar fatura", c, 200)
	vs := versoesDe(t, f, "iddono")
	if len(vs) != 3 {
		t.Fatalf("v1 preguicosa e uma versao por edicao: esperava 3, vieram %d", len(vs))
	}
	confereVersao(t, vs[1], 2, "update", "own")
	confereVersao(t, vs[2], 3, "update", "own")
	if registroDe(vs[1])["oculto"] != true || registroDe(vs[2])["faturaAjuste"] != 1.0 {
		t.Errorf("os registros deveriam mostrar oculto e faturaAjuste: %v / %v", registroDe(vs[1]), registroDe(vs[2]))
	}
}

// F1-T5
func TestMudarADataGeraUmaVersaoETrocaAChave(t *testing.T) {
	f := cenario(t)
	nova := "2026-01-09T10:00:00.000Z"
	c, _ := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"dataEvento":"`+nova+`"}`)
	esperar(t, "mudar data", c, 200)
	if f.ler(t, pkEspaco, chaveTx(dataDono, "iddono")) != nil || f.ler(t, pkEspaco, chaveTx(nova, "iddono")) == nil {
		t.Error("a chave antiga deveria sair e a nova entrar")
	}
	vs := versoesDe(t, f, "iddono")
	atualizacoes := 0
	for _, v := range vs {
		if v["acao"] == "update" {
			atualizacoes++
		}
	}
	if len(vs) != 2 || atualizacoes != 1 {
		t.Errorf("mudar a data e uma edicao: v1 preguicosa e uma update; veio %d versoes, %d update", len(vs), atualizacoes)
	}
	if registroDe(vs[1])["dataEvento"] != nova || registroDe(vs[0])["dataEvento"] != dataDono {
		t.Errorf("as versoes deveriam registrar a data de antes e a de depois")
	}
}

// F1-T6 e F1-T8
func TestExcluirELogicoEGravaAVersaoDelete(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": "AVISOS#ATIVOS", "SK": "s1#iddono", "sid": "s1", "id": "iddono"})
	c, d := chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, "")
	esperar(t, "excluir", c, 200)
	if d["ok"] != true || d["_seq"] != 1.0 {
		t.Errorf("resposta = %v, esperado ok e _seq 1", d)
	}
	vivo := f.ler(t, pkEspaco, chaveTx(dataDono, "iddono"))
	excluidoEm, _ := vivo["excluidoEm"].(string)
	if vivo == nil || excluidoEm == "" || vivo["descricao"] != "original" {
		t.Fatalf("o lancamento deve continuar na tabela, marcado com excluidoEm: %v", vivo)
	}
	if f.ler(t, "AVISOS#ATIVOS", "s1#iddono") != nil {
		t.Error("o aviso do lancamento excluido deveria ter sido removido")
	}
	vs := versoesDe(t, f, "iddono")
	if len(vs) != 2 {
		t.Fatalf("legado: v1 (estado de antes) e v2 (delete); vieram %d", len(vs))
	}
	confereVersao(t, vs[0], 1, "insert", "")
	confereVersao(t, vs[1], 2, "delete", "own")
	if registroDe(vs[0])["excluidoEm"] != nil || registroDe(vs[1])["excluidoEm"] != excluidoEm || vs[1]["em"] != excluidoEm {
		t.Errorf("a v1 e o estado de antes (sem excluidoEm) e a delete guarda excluidoEm = em: %v / %v", registroDe(vs[0]), registroDe(vs[1]))
	}
}

// F1-T7
func TestExcluirDeNovoOuOQueNaoExisteNaoGravaVersao(t *testing.T) {
	f := cenario(t)
	qs := map[string]string{"de": dataDono}
	chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", qs, "")
	antes := len(versoesDe(t, f, "iddono"))
	c, d := chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", qs, "")
	esperar(t, "excluir de novo", c, 200)
	if d["ok"] != true || d["_seq"] != nil {
		t.Errorf("deveria responder so {ok:true}, veio %v", d)
	}
	c, _ = chamar(t, "own", "DELETE", "spaces/s1/tx/naoexiste", qs, "")
	esperar(t, "excluir o que nao existe", c, 200)
	if depois := len(versoesDe(t, f, "iddono")); depois != antes || len(versoesDe(t, f, "naoexiste")) != 0 {
		t.Errorf("nao deveria gravar versao: antes %d, depois %d", antes, depois)
	}
}

// F1-T9
func TestFalhaNaTransacaoAbortaAEdicaoInteira(t *testing.T) {
	f := cenario(t)
	f.falharTransacao = errors.New("falha injetada")
	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"descricao":"nao pode gravar"}`)
	esperar(t, "edicao com falha", c, 500)
	if d["erro"] != "erro ao salvar" {
		t.Errorf("mensagem = %v", d["erro"])
	}
	if viv := f.ler(t, pkEspaco, chaveTx(dataDono, "iddono")); viv["descricao"] != "original" {
		t.Errorf("o lancamento nao pode mudar sem a versao: %v", viv["descricao"])
	}
	if n := len(versoesDe(t, f, "iddono")); n != 0 {
		t.Errorf("nenhuma versao deveria ter ficado: %d", n)
	}
	f.falharTransacao = errors.New("falha injetada")
	c, _ = chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, "")
	esperar(t, "exclusao com falha", c, 500)
	if f.ler(t, pkEspaco, chaveTx(dataDono, "iddono"))["excluidoEm"] != nil {
		t.Error("a exclusao tambem e tudo ou nada")
	}
}

// F1-T10
func TestDuasEdicoesConcorrentesSegundaRecebe409(t *testing.T) {
	f := cenario(t)
	// uma edicao concorrente grava a versao 1 depois que esta leu o historico vazio e antes de a transacao rodar
	f.antesDaTransacao = func() {
		f.colocar(t, doc{"PK": pkHist, "SK": "TX#iddono#00000001", "versao": 1.0, "acao": "update", "porSub": "outro",
			"em": "2026-01-02T00:00:00.000Z", "registro": map[string]any{"id": "iddono"}})
	}
	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"descricao":"perdeu a corrida"}`)
	esperar(t, "edicao concorrente", c, 409)
	if d["codigo"] != "conflito_versao" || d["erro"] != "alterado por outra pessoa" {
		t.Errorf("resposta = %v", d)
	}
	if viv := f.ler(t, pkEspaco, chaveTx(dataDono, "iddono")); viv["descricao"] != "original" {
		t.Errorf("a edicao que perdeu nao pode gravar: %v", viv["descricao"])
	}
	if n := len(versoesDe(t, f, "iddono")); n != 1 {
		t.Errorf("so a versao da edicao que chegou antes deveria existir, vieram %d", n)
	}
}

// F1-T11
func TestMesConferidoBloqueiaSemGravarNada(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":0,"base":"x","caches":{}}`)
	c, _ := conferir(t, "own", "ITAU", "2026-03", 1)
	esperar(t, "conferir marco", c, 200)

	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"valor":99}`)
	esperar(t, "editar valor em mes conferido", c, 409)
	if d["codigo"] != "mes_conferido" {
		t.Errorf("resposta = %v", d)
	}
	c, d = chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, "")
	esperar(t, "excluir em mes conferido", c, 409)
	if d["codigo"] != "mes_conferido" {
		t.Errorf("resposta = %v", d)
	}
	viv := f.ler(t, pkEspaco, chaveTx(dataDono, "iddono"))
	if viv["valor"] != 10.0 || viv["excluidoEm"] != nil || len(versoesDe(t, f, "iddono")) != 0 {
		t.Errorf("sem escrita nao ha versao: %v", viv)
	}
}

// F1-T12
func TestGetTxNaoMostraOExcluidoNemOExcluidoEm(t *testing.T) {
	cenario(t)
	chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, "")
	c, d := chamar(t, "mem", "GET", "spaces/s1/tx", nil, "")
	esperar(t, "listar", c, 200)
	itens, _ := d["itens"].([]any)
	if len(itens) != 2 {
		t.Errorf("deveriam sobrar 2 lancamentos, vieram %d", len(itens))
	}
	if strings.Contains(fmt.Sprint(d), "excluidoEm") || strings.Contains(fmt.Sprint(d), "iddono") {
		t.Errorf("o excluido nao pode aparecer: %v", d)
	}
	// editar o excluido: some das telas, entao nao existe para edicao
	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"descricao":"x"}`)
	esperar(t, "editar o excluido", c, 404)
}

func corpoImport(id, data, descricao string) string {
	return fmt.Sprintf(`{"id":%q,"dataEvento":%q,"status":"PAGO","tipo":"saida","valor":7,"banco":"ITAU","descricao":%q}`, id, data, descricao)
}

// F1-T13
func TestImportarGravaInsertEUpdateNumaTransacao(t *testing.T) {
	f := cenario(t)
	lote := `{"itens":[` + corpoImport("idnovo0001", "2026-03-05T10:00:00.000Z", "novo") + `,` + corpoImport("iddono", dataDono, "reimportado") + `]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "importar", c, 200)
	if d["gravados"] != 2.0 || len(d["pendentes"].([]any)) != 0 || len(d["rejeitados"].([]any)) != 0 {
		t.Fatalf("resposta = %v", d)
	}
	novo := versoesDe(t, f, "idnovo0001")
	if len(novo) != 1 {
		t.Fatalf("id novo: 1 versao insert, vieram %d", len(novo))
	}
	confereVersao(t, novo[0], 1, "insert", "own")
	velho := versoesDe(t, f, "iddono")
	if len(velho) != 2 {
		t.Fatalf("id existente e legado: v1 de antes e v2 update, vieram %d", len(velho))
	}
	confereVersao(t, velho[0], 1, "insert", "")
	confereVersao(t, velho[1], 2, "update", "own")
	if registroDe(velho[0])["descricao"] != "original" || registroDe(velho[1])["descricao"] != "reimportado" {
		t.Errorf("a reimportacao deve guardar o estado anterior: %v / %v", registroDe(velho[0])["descricao"], registroDe(velho[1])["descricao"])
	}
}

// F1-T14
func TestImportarRejeitaItemEmMesConferidoEGravaOResto(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":0,"base":"x","caches":{}}`)
	conferir(t, "own", "ITAU", "2026-03", 1)
	lote := `{"itens":[` + corpoImport("idtravado1", "2026-01-20T10:00:00.000Z", "travado") + `,` + corpoImport("idlivre001", "2026-06-20T10:00:00.000Z", "livre") + `]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "importar", c, 200)
	rej, _ := d["rejeitados"].([]any)
	if d["gravados"] != 1.0 || len(rej) != 1 || !strings.Contains(fmt.Sprint(rej[0]), "mes conferido e travado") {
		t.Fatalf("resposta = %v", d)
	}
	if len(versoesDe(t, f, "idtravado1")) != 0 || len(versoesDe(t, f, "idlivre001")) != 1 {
		t.Error("so o item aceito deve ter versao")
	}
}

func TestImportarComFalhaDevolveTudoEmPendentes(t *testing.T) {
	f := cenario(t)
	f.falharTransacao = errors.New("falha injetada")
	lote := `{"itens":[` + corpoImport("idnovo0001", "2026-03-05T10:00:00.000Z", "a") + `,` + corpoImport("idnovo0002", "2026-03-06T10:00:00.000Z", "b") + `]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "importar com falha", c, 200)
	if d["gravados"] != 0.0 || len(d["pendentes"].([]any)) != 2 || d["_seq"] != nil {
		t.Errorf("lote que falha inteiro volta em pendentes: %v", d)
	}
	if f.ler(t, pkEspaco, chaveTx("2026-03-05T10:00:00.000Z", "idnovo0001")) != nil || len(versoesDe(t, f, "idnovo0001")) != 0 {
		t.Error("nada deveria ter sido gravado")
	}
}

// F1-T16: nao existe metodo para editar nem apagar versao. A lista e exata de proposito: quem acrescentar um metodo
// precisa mexer aqui e pensar se ele respeita a regra.
func TestNaoExisteMetodoParaEditarOuApagarVersao(t *testing.T) {
	nomes := func(tp reflect.Type) []string {
		var ns []string
		for i := 0; i < tp.NumMethod(); i++ {
			ns = append(ns, tp.Method(i).Name)
		}
		sort.Strings(ns)
		return ns
	}
	quer := []string{"Buscar", "Salvar", "SalvarLote", "UltimaVersao"}
	if got := nomes(reflect.TypeOf((*service.Repositorio)(nil)).Elem()); !reflect.DeepEqual(got, quer) {
		t.Errorf("service.Repositorio = %v, esperado %v", got, quer)
	}
	if got := nomes(reflect.TypeOf(&repository.Lancamentos{})); !reflect.DeepEqual(got, quer) {
		t.Errorf("repository.Lancamentos = %v, esperado %v", got, quer)
	}
}

// F1-T17: so o repository conhece HIST#. Nenhuma rota monta a chave do historico com valor do cliente.
func TestSoORepositoryConheceAChaveDoHistorico(t *testing.T) {
	filepath.Walk(".", func(caminho string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(caminho, ".go") || strings.HasSuffix(caminho, "_test.go") {
			return nil
		}
		if strings.HasPrefix(filepath.ToSlash(caminho), "repository/") {
			return nil
		}
		fonte, _ := os.ReadFile(caminho)
		if strings.Contains(string(fonte), "HIST#") {
			t.Errorf("%s menciona HIST#: so repository/ monta a chave do historico", caminho)
		}
		return nil
	})
}

// A rotina diaria de avisos trata o lancamento excluido logicamente como excluido: nao avisa e limpa o auxiliar.
func TestRotinaDeAvisosIgnoraLancamentoExcluidoLogicamente(t *testing.T) {
	f, id := cenarioAvisos(t)
	env := trocaEnvio(t, nil)
	f.itens[chave(pkEspaco, "TX#2026-10-10T12:00:00.000Z#"+id)]["excluidoEm"] = &types.AttributeValueMemberS{Value: "2026-10-09T10:00:00.000Z"}
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-10")); n != 0 || len(*env) != 0 {
		t.Errorf("excluido logicamente nao deve avisar, enviou %d", n)
	}
	if auxDe(t, f, id) != nil {
		t.Error("o auxiliar do excluido deveria ter sido limpo pela rotina")
	}
}
