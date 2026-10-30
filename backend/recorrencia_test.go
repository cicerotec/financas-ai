package main

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"

	"financas/auth"
	"financas/domain"
)

// Item 51 (.ai/specs/recorrencia/spec.md): o lancamento marcado com recorrente: { intervalo } e copiado sozinho. A copia nasce na hora de
// marcar; a rotina diaria dos avisos cria a seguinte quando a ultima vence. O relogio padrao dos testes e 05/02/2026.

const pkAvisosAtivos = "AVISOS#ATIVOS"

// copiasDe devolve os lancamentos do espaco s1 com a descricao dada, do mais antigo para o mais novo; vivos=true tira os excluidos.
func copiasDe(t *testing.T, f *bancoFake, descricao string, vivos bool) []doc {
	t.Helper()
	var r []doc
	for k, it := range f.itens {
		if !strings.HasPrefix(k, pkEspaco+"|"+dePrefixoTx) {
			continue
		}
		var d doc
		if err := attributevalue.UnmarshalMap(it, &d); err != nil {
			t.Fatal(err)
		}
		if d["descricao"] != descricao || (vivos && d["excluidoEm"] != nil) {
			continue
		}
		r = append(r, d)
	}
	sort.Slice(r, func(i, j int) bool { return r[i]["dataEvento"].(string) < r[j]["dataEvento"].(string) })
	return r
}

func versoesNoBanco(f *bancoFake) int {
	n := 0
	for k := range f.itens {
		if strings.HasPrefix(k, "HIST#s1|") {
			n++
		}
	}
	return n
}

func itemRecorrencia(t *testing.T, f *bancoFake, primeiro string) doc {
	t.Helper()
	return f.ler(t, pkAvisosAtivos, "s1#recorrencia#"+primeiro)
}

func rodarRotina(t *testing.T) {
	t.Helper()
	if _, err := srvTeste.avisos.Rodar(context.Background()); err != nil {
		t.Fatal(err)
	}
}

const corpoColegio = `{"dataEvento":"2026-02-10T10:00:00.000Z","valor":900,"tipo":"saida","status":"PAGO","banco":"ITAU","descricao":"Colegio","nota":"mensalidade","tags":["Lazer"],"aviso":{"dias":[3],"insistir":false},"recorrente":{"intervalo":%d}}`

func criarColegio(t *testing.T, intervalo string) (primeiro string) {
	t.Helper()
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, strings.Replace(corpoColegio, "%d", intervalo, 1))
	esperar(t, "criar com recorrente", c, 201)
	return d["id"].(string)
}

// RC-T2, RC-T4: criar ja marcado cria na hora a primeira copia, sem versao no historico e sem serie.
func TestRecorrenciaNaCriacaoCriaACopiaNaHora(t *testing.T) {
	f := cenario(t)
	primeiro := criarColegio(t, "6")

	l := copiasDe(t, f, "Colegio", true)
	if len(l) != 2 {
		t.Fatalf("esperava o lancamento e a copia, veio %d", len(l))
	}
	origem, copia := l[0], l[1]
	if rec, _ := origem["recorrente"].(map[string]any); rec["intervalo"] != 6.0 {
		t.Errorf("o lancamento deveria levar recorrente {intervalo: 6}: %v", origem["recorrente"])
	}
	if copia["dataEvento"] != "2026-08-10T10:00:00.000Z" {
		t.Errorf("semestral: a copia deveria nascer em 10/08, veio %v", copia["dataEvento"])
	}
	for _, k := range []string{"status", "tipo", "valor", "descricao", "nota", "banco", "tags", "aviso", "recorrente"} {
		if copia[k] == nil {
			t.Errorf("a copia deveria levar %s", k)
		}
	}
	if copia["serie"] != nil {
		t.Error("a copia da recorrencia nao leva serie")
	}
	if copia["previsto"] != true {
		t.Error("a copia e de data futura: o servidor a marca como prevista")
	}
	if copia["criadoPor"] != "own" {
		t.Errorf("criadoPor da copia deveria ser o do registro de origem: %v", copia["criadoPor"])
	}
	if n := versoesNoBanco(f); n != 0 {
		t.Errorf("criar nao grava versao, nem a da copia: %d versao(oes)", n)
	}
	if f.ler(t, pkAvisosAtivos, "s1#"+copia["id"].(string)) == nil {
		t.Error("o aviso da copia deveria ter sido sincronizado")
	}
	it := itemRecorrencia(t, f, primeiro)
	if it == nil || it["tipo"] != "recorrencia" || it["ultimoId"] != copia["id"] || it["dataUltimo"] != copia["dataEvento"] ||
		it["intervalo"] != 6.0 || it["dia"] != 10.0 {
		t.Errorf("item da recorrencia inesperado: %v", it)
	}
}

// RC-T1, RC-T2: marcar um lancamento que ja existe grava versao e cria a copia na primeira data depois de hoje.
func TestRecorrenciaMarcarLancamentoExistente(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "own", "POST", "spaces/s1/recorrencias", map[string]string{"de": dataDono}, `{"id":"iddono","intervalo":1}`)
	esperar(t, "marcar", c, 201)
	// 01/01 mais um mes e 01/02, que ja passou (hoje e 05/02): a primeira data depois de hoje e 01/03
	if d["id"] != "iddono" || d["intervalo"] != 1.0 || d["dataUltimo"] != "2026-03-01T10:00:00.000Z" {
		t.Errorf("resposta inesperada: %v", d)
	}
	origem := f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono")
	if origem["recorrente"] == nil {
		t.Error("o lancamento deveria ter sido marcado")
	}
	if versoesNoBanco(f) == 0 {
		t.Error("marcar um lancamento que ja existe e uma alteracao: deveria gravar versao")
	}
	copia := f.ler(t, pkEspaco, dePrefixoTx+"2026-03-01T10:00:00.000Z#"+domain.IDCopiaRecorrencia("iddono", "2026-03-01T10:00:00.000Z"))
	if copia == nil || copia["recorrente"] == nil || copia["descricao"] != "original" || copia["previsto"] != true {
		t.Errorf("deveria haver a copia de 01/03: %v", copia)
	}
	// ja e recorrente
	c, _ = chamar(t, "own", "POST", "spaces/s1/recorrencias", map[string]string{"de": dataDono}, `{"id":"iddono","intervalo":1}`)
	esperar(t, "marcar de novo", c, 400)
}

func TestRecorrenciaRecusas(t *testing.T) {
	f := cenario(t)
	de := map[string]string{"de": dataDono}
	for _, corpo := range []string{`{"id":"iddono","intervalo":0}`, `{"id":"iddono","intervalo":121}`, `{"id":"iddono"}`} {
		c, _ := chamar(t, "own", "POST", "spaces/s1/recorrencias", de, corpo)
		esperar(t, "intervalo invalido "+corpo, c, 400)
	}
	c, _ := chamar(t, "own", "POST", "spaces/s1/recorrencias", de, `{"id":"naoexiste","intervalo":1}`)
	esperar(t, "lancamento inexistente", c, 400)
	chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", de, "")
	c, _ = chamar(t, "own", "POST", "spaces/s1/recorrencias", de, `{"id":"iddono","intervalo":1}`)
	esperar(t, "lancamento excluido", c, 400)

	// transferencia nao e recorrente
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + dataMembro + "#idtransf", "id": "idtransf", "dataEvento": dataMembro,
		"status": "PAGO", "tipo": "saida", "valor": 5.0, "banco": "ITAU", "transferParId": "par1", "criadoPor": "own"})
	c, _ = chamar(t, "own", "POST", "spaces/s1/recorrencias", map[string]string{"de": dataMembro}, `{"id":"idtransf","intervalo":1}`)
	esperar(t, "transferencia", c, 400)
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, `{"dataEvento":"2026-02-10T10:00:00.000Z","valor":5,"tipo":"saida","status":"PAGO","banco":"ITAU","transferParId":"par1","recorrente":{"intervalo":1}}`)
	esperar(t, "criar transferencia recorrente", c, 400)
	c, _ = chamar(t, "own", "POST", "spaces/s1/tx", nil, `{"dataEvento":"2026-02-10T10:00:00.000Z","valor":5,"tipo":"saida","status":"PAGO","banco":"ITAU","recorrente":{"intervalo":0}}`)
	esperar(t, "criar com intervalo invalido", c, 400)

	// member: so o que criou (RC-T18)
	c, _ = chamar(t, "mem", "POST", "spaces/s1/recorrencias", map[string]string{"de": dataLegado}, `{"id":"idlegado","intervalo":1}`)
	esperar(t, "member marca o de outra pessoa", c, 403)
	c, _ = chamar(t, "mem", "POST", "spaces/s1/recorrencias", map[string]string{"de": dataMembro}, `{"id":"idmembro","intervalo":1}`)
	esperar(t, "member marca o proprio", c, 201)
	c, _ = chamar(t, "estranho", "POST", "spaces/s1/recorrencias", de, `{"id":"iddono","intervalo":1}`)
	esperar(t, "quem nao e do espaco", c, 403)
}

// O recorrente nao se escreve pelo PUT: so a recorrencia o liga e desliga.
func TestRecorrenteNaoSeEscrevePeloPut(t *testing.T) {
	f := cenario(t)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"recorrente":{"intervalo":2},"descricao":"editado"}`)
	esperar(t, "put", c, 200)
	if d := f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono"); d["recorrente"] != nil || d["descricao"] != "editado" {
		t.Errorf("o PUT nao pode ligar o recorrente: %v", d)
	}
}

// RC-T3b, RC-T7: a rotina cria a seguinte quando a ultima vence, uma vez so.
func TestRecorrenciaRotinaCriaASeguinteQuandoAUltimaVence(t *testing.T) {
	f := cenario(t)
	primeiro := criarColegio(t, "1") // 10/02 e a copia de 10/03
	relogioEm(t, 2026, 3, 9, 15, 0)
	rodarRotina(t)
	if n := len(copiasDe(t, f, "Colegio", true)); n != 2 {
		t.Fatalf("a ultima (10/03) ainda nao venceu: esperava 2 lancamentos, veio %d", n)
	}
	relogioEm(t, 2026, 3, 10, 15, 0)
	rodarRotina(t)
	l := copiasDe(t, f, "Colegio", true)
	if len(l) != 3 || l[2]["dataEvento"] != "2026-04-10T10:00:00.000Z" {
		t.Fatalf("no dia 10/03 deveria nascer a de 10/04: %v", l)
	}
	if it := itemRecorrencia(t, f, primeiro); it["ultimoId"] != l[2]["id"] || it["dataUltimo"] != "2026-04-10T10:00:00.000Z" {
		t.Errorf("o item deveria ter avancado: %v", it)
	}
	if versoesNoBanco(f) != 0 {
		t.Error("a copia da rotina nao grava versao")
	}
	rodarRotina(t)
	if n := len(copiasDe(t, f, "Colegio", true)); n != 3 {
		t.Errorf("rodar de novo no mesmo dia nao duplica: %d", n)
	}

	// RC-T8: a copia foi criada e o item nao avancou (caiu no meio): a rodada seguinte so avanca, sem duplicar
	item := itemRecorrencia(t, f, primeiro)
	item["ultimoId"], item["dataUltimo"] = l[1]["id"], l[1]["dataEvento"]
	f.colocar(t, item)
	rodarRotina(t)
	if n := len(copiasDe(t, f, "Colegio", true)); n != 3 {
		t.Errorf("depois da falha no meio, nao deve duplicar: %d", n)
	}
	if it := itemRecorrencia(t, f, primeiro); it["ultimoId"] != l[2]["id"] {
		t.Errorf("o item deveria ter avancado de novo: %v", it)
	}
}

// RC-T9: apagar a copia pula o ciclo: a rotina cria a seguinte na data e nao recria a apagada.
func TestRecorrenciaApagarACopiaPulaOCiclo(t *testing.T) {
	f := cenario(t)
	criarColegio(t, "1")
	l := copiasDe(t, f, "Colegio", true)
	c, _ := chamar(t, "own", "DELETE", "spaces/s1/tx/"+l[1]["id"].(string), map[string]string{"de": l[1]["dataEvento"].(string)}, "")
	esperar(t, "apagar a copia", c, 200)
	relogioEm(t, 2026, 3, 10, 15, 0)
	rodarRotina(t)
	vivos := copiasDe(t, f, "Colegio", true)
	if len(vivos) != 2 || vivos[1]["dataEvento"] != "2026-04-10T10:00:00.000Z" {
		t.Errorf("a de 10/03 foi apagada e fica pulada; a de 10/04 nasce: %v", vivos)
	}
	if todos := copiasDe(t, f, "Colegio", false); len(todos) != 3 {
		t.Errorf("a apagada continua na tabela (exclusao logica) e nao e recriada: %d", len(todos))
	}
}

// RC-T10: excluir pelo app a ultima nao para a recorrencia: a copia seguinte sai com os dados dela.
func TestRecorrenciaUltimaExcluidaContinua(t *testing.T) {
	f := cenario(t)
	primeiro := criarColegio(t, "1")
	if itemRecorrencia(t, f, primeiro) == nil {
		t.Fatal("sem item")
	}
	l := copiasDe(t, f, "Colegio", true)
	chamar(t, "own", "DELETE", "spaces/s1/tx/"+l[0]["id"].(string), map[string]string{"de": l[0]["dataEvento"].(string)}, "")
	if itemRecorrencia(t, f, primeiro) == nil {
		t.Error("excluir o primeiro registro nao apaga o item da recorrencia")
	}
}

// R9: rotina parada ou lancamento antigo: uma copia so, depois de hoje, sem recuperar os ciclos perdidos.
func TestRecorrenciaSempreParaFrente(t *testing.T) {
	f := cenario(t)
	criarColegio(t, "1")
	relogioEm(t, 2026, 6, 20, 15, 0) // a ponta e 10/03; 10/04, 10/05 e 10/06 ja passaram
	rodarRotina(t)
	l := copiasDe(t, f, "Colegio", true)
	if len(l) != 3 || l[2]["dataEvento"] != "2026-07-10T10:00:00.000Z" {
		t.Errorf("esperava uma copia so, em 10/07: %v", l)
	}
}

// RC-T11, RC-T12, RC-T19: desligar, listar e religar.
func TestRecorrenciaDesligarListarReligar(t *testing.T) {
	f := cenario(t)
	primeiro := criarColegio(t, "1")
	r, _ := srvTeste.rec.Get(context.Background(), "s1")
	if r.StatusCode != 200 || !strings.Contains(r.Body, primeiro) || !strings.Contains(r.Body, `"intervalo":1`) {
		t.Errorf("GET /recorrencias deveria listar a ativa: %s", r.Body)
	}
	pontaAntes := copiasDe(t, f, "Colegio", true)[1]

	c, _ := chamar(t, "own", "DELETE", "spaces/s1/recorrencias/"+primeiro, nil, "")
	esperar(t, "desligar", c, 200)
	if itemRecorrencia(t, f, primeiro) != nil {
		t.Error("desligar apaga o item")
	}
	ponta := f.ler(t, pkEspaco, dePrefixoTx+pontaAntes["dataEvento"].(string)+"#"+pontaAntes["id"].(string))
	if ponta == nil || ponta["recorrente"] != nil {
		t.Errorf("a copia a vista fica, sem a marca: %v", ponta)
	}
	c, _ = chamar(t, "own", "DELETE", "spaces/s1/recorrencias/"+primeiro, nil, "")
	esperar(t, "desligar o que ja esta desligado", c, 200)
	relogioEm(t, 2026, 3, 10, 15, 0)
	rodarRotina(t)
	if n := len(copiasDe(t, f, "Colegio", true)); n != 2 {
		t.Errorf("desligada, a rotina nao cria nada: %d", n)
	}
	r, _ = srvTeste.rec.Get(context.Background(), "s1")
	if strings.Contains(r.Body, primeiro) {
		t.Errorf("desligada nao aparece na lista: %s", r.Body)
	}

	// religar: marca a copia a vista de novo; a proxima nasce depois de hoje (10/03), em 10/04
	c, d := chamar(t, "own", "POST", "spaces/s1/recorrencias", map[string]string{"de": pontaAntes["dataEvento"].(string)},
		`{"id":"`+pontaAntes["id"].(string)+`","intervalo":1}`)
	esperar(t, "religar", c, 201)
	if d["dataUltimo"] != "2026-04-10T10:00:00.000Z" {
		t.Errorf("religar cria a proxima depois de hoje: %v", d)
	}
}

// R9: no maximo 50 copias por rodada; o que sobra fica para a seguinte (RC-T15). Ponta que sumiu do banco encerra a recorrencia (RC-T14).
func TestRecorrenciaLimiteEPontaPerdida(t *testing.T) {
	f := cenario(t)
	for i := 0; i < 51; i++ {
		id := "ponta" + string(rune('a'+i/26)) + string(rune('a'+i%26))
		f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2026-02-01T10:00:00.000Z#" + id, "id": id, "dataEvento": "2026-02-01T10:00:00.000Z",
			"status": "PAGO", "tipo": "saida", "valor": 1.0, "banco": "ITAU", "descricao": "fixa" + id, "criadoPor": "own", "recorrente": doc{"intervalo": 1.0}})
		f.colocar(t, doc{"PK": pkAvisosAtivos, "SK": "s1#recorrencia#" + id, "sid": "s1", "id": "recorrencia#" + id, "tipo": "recorrencia",
			"ultimoId": id, "dataUltimo": "2026-02-01T10:00:00.000Z", "intervalo": 1.0, "dia": 1.0})
	}
	f.colocar(t, doc{"PK": pkAvisosAtivos, "SK": "s1#recorrencia#sumiu", "sid": "s1", "id": "recorrencia#sumiu", "tipo": "recorrencia",
		"ultimoId": "pontasumiu", "dataUltimo": "2026-02-01T10:00:00.000Z", "intervalo": 1.0, "dia": 1.0})
	contar := func() int {
		n := 0
		for k := range f.itens {
			if strings.HasPrefix(k, pkEspaco+"|"+dePrefixoTx+"2026-03-01") {
				n++
			}
		}
		return n
	}
	rodarRotina(t)
	if n := contar(); n != service50 {
		t.Errorf("uma rodada cria no maximo %d copias: %d", service50, n)
	}
	rodarRotina(t)
	if n := contar(); n != 51 {
		t.Errorf("a rodada seguinte cria o que sobrou: %d de 51", n)
	}
	if f.ler(t, pkAvisosAtivos, "s1#recorrencia#sumiu") != nil {
		t.Error("a recorrencia sem ponta deveria ter sido removida")
	}
}

const service50 = 50

// RC-T17: cada copia vai para o espaco do item.
func TestRecorrenciaIsolamentoEntreEspacos(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": "SPACE#s2", "SK": dePrefixoTx + "2026-02-01T10:00:00.000Z#idoutro", "id": "idoutro", "dataEvento": "2026-02-01T10:00:00.000Z",
		"status": "PAGO", "tipo": "saida", "valor": 1.0, "banco": "ITAU", "descricao": "doutro", "criadoPor": "x", "recorrente": doc{"intervalo": 1.0}})
	f.colocar(t, doc{"PK": pkAvisosAtivos, "SK": "s2#recorrencia#idoutro", "sid": "s2", "id": "recorrencia#idoutro", "tipo": "recorrencia",
		"ultimoId": "idoutro", "dataUltimo": "2026-02-01T10:00:00.000Z", "intervalo": 1.0, "dia": 1.0})
	rodarRotina(t)
	if n := len(copiasDe(t, f, "doutro", true)); n != 0 {
		t.Errorf("a copia do espaco s2 nao pode cair no s1: %d", n)
	}
	n := 0
	for k := range f.itens {
		if strings.HasPrefix(k, "SPACE#s2|"+dePrefixoTx+"2026-03-01") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("a copia deveria ter nascido no s2: %d", n)
	}
}

// As rotas da recorrencia e o que cada papel pode (RC-T18).
func TestRecorrenciaRotasEAutorizacao(t *testing.T) {
	casos := []struct {
		m    string
		rest []string
		quer auth.Acao
	}{
		{"GET", []string{"recorrencias"}, auth.AcaoLerTx},
		{"POST", []string{"recorrencias"}, auth.AcaoEditarTx},
		{"DELETE", []string{"recorrencias", "abc"}, auth.AcaoEditarTx},
	}
	for _, c := range casos {
		if a, ok := auth.RotaParaAcao(c.m, c.rest); !ok || a != c.quer {
			t.Errorf("%s %v: %v %v, esperado %v", c.m, c.rest, a, ok, c.quer)
		}
	}
	if _, ok := auth.RotaParaAcao("PUT", []string{"recorrencias"}); ok {
		t.Error("PUT /recorrencias nao existe")
	}
	if auth.Permitido(auth.PapelMember, auth.AcaoEditarTx) != auth.SoDono {
		t.Error("o member marca so o que criou")
	}
}

// RC-T20: nada existente e tocado.
func TestRecorrenciaNaoTocaOPassado(t *testing.T) {
	f := cenario(t)
	relogioEm(t, 2026, 3, 10, 15, 0)
	rodarRotina(t)
	for _, k := range []string{dePrefixoTx + dataDono + "#iddono", dePrefixoTx + dataMembro + "#idmembro"} {
		if d := f.ler(t, pkEspaco, k); d["recorrente"] != nil {
			t.Errorf("%s ganhou recorrente sem o dono marcar", k)
		}
	}
	if versoesNoBanco(f) != 0 {
		t.Error("a rotina nao grava versao")
	}
}


// Marcar um lancamento ja existente com aviso: o aviso vai na primeira copia (com o item no AVISOS#ATIVOS) e o original nao muda;
// um aviso invalido e recusado e nada e criado.
func TestRecorrenciaMarcarComAviso(t *testing.T) {
	f := cenario(t)
	de := map[string]string{"de": dataDono}
	c, _ := chamar(t, "own", "POST", "spaces/s1/recorrencias", de, `{"id":"iddono","intervalo":1,"aviso":{"dias":[0,1],"insistir":true}}`)
	esperar(t, "marcar com aviso", c, 201)
	copia := f.ler(t, pkEspaco, dePrefixoTx+"2026-03-01T10:00:00.000Z#"+domain.IDCopiaRecorrencia("iddono", "2026-03-01T10:00:00.000Z"))
	av, _ := copia["aviso"].(map[string]any)
	if av == nil || av["insistir"] != true || len(av["dias"].([]any)) != 2 {
		t.Fatalf("a copia deveria levar o aviso escolhido: %v", copia["aviso"])
	}
	if f.ler(t, pkAvisosAtivos, "s1#"+copia["id"].(string)) == nil {
		t.Error("o aviso da copia deveria estar no AVISOS#ATIVOS")
	}
	if f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono")["aviso"] != nil {
		t.Error("o lancamento original nao pode ganhar aviso")
	}

	// aviso invalido: 400 e nenhuma recorrencia
	c, _ = chamar(t, "own", "POST", "spaces/s1/recorrencias", map[string]string{"de": dataMembro}, `{"id":"idmembro","intervalo":1,"aviso":{"dias":[99],"insistir":false}}`)
	esperar(t, "aviso invalido", c, 400)
	if f.ler(t, pkAvisosAtivos, "s1#recorrencia#idmembro") != nil {
		t.Error("com aviso invalido nada deveria ser criado")
	}
}

// aviso: null ao marcar = a primeira copia sai sem aviso, mesmo que o lancamento marcado tenha um.
func TestRecorrenciaMarcarSemAvisoLimpaDaCopia(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + dataDono + "#idcomaviso", "id": "idcomaviso", "dataEvento": dataDono,
		"status": "PAGO", "tipo": "saida", "valor": 5.0, "banco": "ITAU", "criadoPor": "own", "aviso": doc{"dias": []any{1.0}, "insistir": false}})
	c, _ := chamar(t, "own", "POST", "spaces/s1/recorrencias", map[string]string{"de": dataDono}, `{"id":"idcomaviso","intervalo":1,"aviso":null}`)
	esperar(t, "marcar sem aviso", c, 201)
	copia := f.ler(t, pkEspaco, dePrefixoTx+"2026-03-01T10:00:00.000Z#"+domain.IDCopiaRecorrencia("idcomaviso", "2026-03-01T10:00:00.000Z"))
	if copia == nil || copia["aviso"] != nil {
		t.Errorf("a copia deveria sair sem aviso: %v", copia)
	}
}
