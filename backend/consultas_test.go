package main

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"financas/domain"
)

// Item 53 (.ai/specs/tendencias-consultas/spec.md): consultas salvas da Tendencias, compartilhadas no espaco. Cenario: s1 com "own"
// (owner) e "mem" (member); o outro espaco, s2, tem o dono "estranho".

const consultaBase = `{"nome":"Mercado sem presente","modo":"grupos","visao":"gastos","grupos":[{"tem":["mercado"],"nao":["presente"]}]}`

func jsonUnmarshal(s string, v any) error { return json.Unmarshal([]byte(s), v) }
func itoa(n int) string                   { return strconv.Itoa(n) }

func comApelidos(t *testing.T, f *bancoFake) {
	t.Helper()
	f.colocar(t, doc{"PK": "USER#own", "SK": "SPACE#s1", "role": "owner", "apelido": "Cicero"})
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member", "apelido": "Ana"})
}

// nova cria uma consulta como `sub` e devolve o id.
func nova(t *testing.T, sub, corpo string) string {
	t.Helper()
	c, d := chamar(t, sub, "POST", "spaces/s1/consultas", nil, corpo)
	esperar(t, "criar consulta", c, 201)
	id, _ := d["id"].(string)
	if len(id) != 20 {
		t.Fatalf("o id deveria ter 20 caracteres: %q", id)
	}
	return id
}

func comNome(nome string) string {
	return strings.Replace(consultaBase, "Mercado sem presente", nome, 1)
}

func listaConsultas(t *testing.T, sub string) []any {
	t.Helper()
	r, _ := srvTeste.processar(context.Background(), &user{Sub: sub}, "GET", []string{"spaces", "s1", "consultas"}, nil, "")
	var l []any
	if err := jsonUnmarshal(r.Body, &l); err != nil {
		t.Fatalf("GET /consultas nao devolveu uma lista: %v (%s)", err, r.Body)
	}
	return l
}

func itemConsulta(f *bancoFake, t *testing.T, id string) doc {
	return f.ler(t, pkEspaco, "CONSULTA#"+id)
}

func TestConsultaCriarGravaOItemEODevolveAVisao(t *testing.T) { // C-T5, C-T11
	f := cenario(t)
	comApelidos(t, f)
	c, d := chamar(t, "mem", "POST", "spaces/s1/consultas", nil, consultaBase)
	esperar(t, "member cria", c, 201)
	if d["nome"] != "Mercado sem presente" || d["autor"] != "Ana" || d["minha"] != true || d["podeAlterar"] != true || d["version"] != 1.0 {
		t.Errorf("visao da consulta criada: %v", d)
	}
	for _, k := range []string{"criadoPor", "PK", "SK", "excluidoEm", "atualizadoPor"} {
		if _, tem := d[k]; tem {
			t.Errorf("a visao nao pode trazer %q (o sub nunca sai): %v", k, d)
		}
	}
	it := itemConsulta(f, t, d["id"].(string))
	if it["criadoPor"] != "mem" || it["atualizadoPor"] != "mem" || it["criadoEm"] == nil || it["criadoEm"] != it["atualizadoEm"] {
		t.Errorf("item gravado: %v", it)
	}
}

func TestConsultaCriadoPorVemSoDoToken(t *testing.T) { // C-T5
	f := cenario(t)
	c, _ := chamar(t, "mem", "POST", "spaces/s1/consultas", nil, strings.Replace(consultaBase, `"nome"`, `"criadoPor":"own","nome"`, 1))
	esperar(t, "criadoPor no corpo", c, 400)
	c, _ = chamar(t, "mem", "POST", "spaces/s1/consultas", nil, strings.Replace(consultaBase, `"nome"`, `"id":"forjado","nome"`, 1))
	esperar(t, "id no corpo", c, 400)
	for k := range f.itens {
		if strings.Contains(k, "CONSULTA#") {
			t.Errorf("nada deveria ter sido gravado: %s", k)
		}
	}
}

func TestConsultaRecusasDeFormatoEDeNome(t *testing.T) { // C-T6, C-T9
	cenario(t)
	nova(t, "mem", consultaBase)
	casos := map[string]string{
		"json com total":    strings.Replace(consultaBase, `"nome"`, `"total":10,"nome"`, 1),
		"modo invalido":     strings.Replace(consultaBase, `"grupos","visao"`, `"x","visao"`, 1),
		"sem tag":           `{"nome":"Vazia","modo":"grupos","visao":"gastos","grupos":[{"tem":[],"nao":[]}]}`,
		"nome repetido":     consultaBase,
		"nome em maiuscula": comNome("MERCADO SEM PRESENTE"),
	}
	for nome, corpo := range casos {
		c, d := chamar(t, "own", "POST", "spaces/s1/consultas", nil, corpo)
		esperar(t, nome, c, 400)
		if d["erro"] == nil {
			t.Errorf("%s: a resposta deveria ter a mensagem", nome)
		}
	}
}

func TestConsultaNomeLiberadoDepoisDeExcluir(t *testing.T) { // C-T9, C-T14
	cenario(t)
	id := nova(t, "mem", consultaBase)
	c, _ := chamar(t, "mem", "DELETE", "spaces/s1/consultas/"+id, nil, "")
	esperar(t, "excluir", c, 200)
	nova(t, "own", consultaBase) // o nome de uma excluida pode ser reaproveitado
}

func TestConsultaLimiteDeCinquentaVivas(t *testing.T) { // C-T10
	cenario(t)
	var primeiro string
	for i := 0; i < domain.MaxConsultas; i++ {
		id := nova(t, "mem", comNome("c"+itoa(i)))
		if i == 0 {
			primeiro = id
		}
	}
	c, d := chamar(t, "mem", "POST", "spaces/s1/consultas", nil, comNome("a51"))
	esperar(t, "51a consulta", c, 400)
	if msg, _ := d["erro"].(string); !strings.Contains(msg, "limite") {
		t.Errorf("a mensagem deveria falar do limite: %v", d["erro"])
	}
	c, _ = chamar(t, "mem", "DELETE", "spaces/s1/consultas/"+primeiro, nil, "")
	esperar(t, "excluir libera a vaga", c, 200)
	nova(t, "mem", comNome("a51"))
}

func TestConsultaListaViveOrdenadaESemSub(t *testing.T) { // C-T11
	f := cenario(t)
	comApelidos(t, f)
	relogioEm(t, 2026, 2, 5, 10, 0)
	velha := nova(t, "own", comNome("Velha"))
	relogioEm(t, 2026, 2, 6, 10, 0)
	nova(t, "mem", comNome("Nova"))
	relogioEm(t, 2026, 2, 7, 10, 0)
	apagada := nova(t, "mem", comNome("Apagada"))
	chamar(t, "mem", "DELETE", "spaces/s1/consultas/"+apagada, nil, "")
	l := listaConsultas(t, "mem")
	if len(l) != 2 || l[0].(map[string]any)["nome"] != "Nova" || l[1].(map[string]any)["nome"] != "Velha" {
		t.Fatalf("so as vivas, da mais nova para a mais antiga: %v", l)
	}
	nova_, antiga := l[0].(map[string]any), l[1].(map[string]any)
	if nova_["autor"] != "Ana" || nova_["minha"] != true || nova_["podeAlterar"] != true {
		t.Errorf("a dela: %v", nova_)
	}
	if antiga["autor"] != "Cicero" || antiga["minha"] != false || antiga["podeAlterar"] != false || antiga["id"] != velha {
		t.Errorf("a do dono, vista pela member: %v", antiga)
	}
	for _, x := range l {
		if _, tem := x.(map[string]any)["criadoPor"]; tem {
			t.Error("o sub nao pode sair na lista")
		}
	}
	// o dono pode alterar todas
	for _, x := range listaConsultas(t, "own") {
		if x.(map[string]any)["podeAlterar"] != true {
			t.Errorf("o dono altera todas: %v", x)
		}
	}
}

func TestConsultaAutorQueSaiuDoEspaco(t *testing.T) { // R5
	f := cenario(t)
	id := nova(t, "mem", consultaBase)
	delete(f.itens, chave("USER#mem", "SPACE#s1"))
	l := listaConsultas(t, "own")
	if len(l) != 1 || l[0].(map[string]any)["autor"] != "alguem do espaco" {
		t.Errorf("sem vinculo, o autor aparece como 'alguem do espaco': %v", l)
	}
	_ = id
}

func TestMemberAlteraSoAsPropriasEODonoTodas(t *testing.T) { // C-T3, C-T4
	f := cenario(t)
	propria := nova(t, "mem", consultaBase)
	doDono := nova(t, "own", comNome("Do dono"))
	corpo := func(v int) string { return `{"version":` + string(rune('0'+v)) + `,"nome":"Mudou"}` }

	c, _ := chamar(t, "mem", "PUT", "spaces/s1/consultas/"+propria, nil, corpo(1))
	esperar(t, "member renomeia a propria", c, 200)
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/consultas/"+doDono, nil, corpo(1))
	esperar(t, "member renomeia a do dono", c, 403)
	c, _ = chamar(t, "mem", "DELETE", "spaces/s1/consultas/"+doDono, nil, "")
	esperar(t, "member exclui a do dono", c, 403)
	c, _ = chamar(t, "mem", "POST", "spaces/s1/consultas/"+doDono+"/restaurar", nil, "")
	esperar(t, "member restaura a do dono", c, 403)
	if it := itemConsulta(f, t, doDono); it["nome"] != "Do dono" || it["excluidoEm"] != nil {
		t.Errorf("a consulta do dono nao pode mudar: %v", it)
	}

	// o dono altera, exclui e restaura a da member; o autor nao muda e atualizadoPor registra quem mexeu
	c, d := chamar(t, "own", "PUT", "spaces/s1/consultas/"+propria, nil, `{"version":2,"nome":"Pelo dono"}`)
	esperar(t, "dono renomeia a da member", c, 200)
	if d["minha"] != false || d["podeAlterar"] != true {
		t.Errorf("visao do dono sobre a da member: %v", d)
	}
	if it := itemConsulta(f, t, propria); it["criadoPor"] != "mem" || it["atualizadoPor"] != "own" || it["version"] != 3.0 {
		t.Errorf("criadoPor fica, atualizadoPor e version mudam: %v", it)
	}
	c, _ = chamar(t, "own", "DELETE", "spaces/s1/consultas/"+propria, nil, "")
	esperar(t, "dono exclui a da member", c, 200)
	c, _ = chamar(t, "own", "POST", "spaces/s1/consultas/"+propria+"/restaurar", nil, "")
	esperar(t, "dono restaura a da member", c, 200)
	if it := itemConsulta(f, t, propria); it["criadoPor"] != "mem" || it["excluidoEm"] != nil {
		t.Errorf("restaurar devolve o mesmo item com o mesmo autor: %v", it)
	}
}

func TestConsultaPutVersaoEConflito(t *testing.T) { // C-T12, C-T13
	f := cenario(t)
	id := nova(t, "mem", consultaBase)
	outra := nova(t, "mem", comNome("Outra"))

	c, d := chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil, `{"version":1,"nome":"Primeira"}`)
	esperar(t, "version atual", c, 200)
	if d["version"] != 2.0 {
		t.Errorf("a version sobe: %v", d["version"])
	}
	c, d = chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil, `{"version":1,"nome":"Atrasada"}`)
	esperar(t, "version antiga", c, 409)
	if d["codigo"] != "conflito_versao" {
		t.Errorf("o 409 deveria dizer conflito_versao: %v", d)
	}
	if it := itemConsulta(f, t, id); it["nome"] != "Primeira" {
		t.Errorf("o 409 nao altera nada: %v", it)
	}
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil, `{"version":2,"nome":"primeira"}`)
	esperar(t, "manter o proprio nome (so a caixa muda)", c, 200)
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil, `{"version":3,"nome":"OUTRA"}`)
	esperar(t, "nome de outra consulta viva", c, 400)
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil, `{"version":3}`)
	esperar(t, "sem nome nem consulta", c, 400)
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil, `{"nome":"Sem version"}`)
	esperar(t, "sem version", c, 400)
	_ = outra
}

func TestConsultaPutTrocaAConsultaEmBloco(t *testing.T) { // R11
	f := cenario(t)
	id := nova(t, "mem", consultaBase)
	c, d := chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil,
		`{"version":1,"modo":"avancado","visao":"saldo","expressao":[{"k":"tag","v":"casa"},{"k":"op","v":"OU"},{"k":"tag","v":"carro"}]}`)
	esperar(t, "trocar a consulta", c, 200)
	it := itemConsulta(f, t, id)
	if it["modo"] != "avancado" || it["visao"] != "saldo" || it["grupos"] != nil || it["expressao"] == nil || it["nome"] != "Mercado sem presente" {
		t.Errorf("trocou o bloco, ficou o nome e saiu o grupos: %v", it)
	}
	if d["expressao"] == nil || d["grupos"] != nil {
		t.Errorf("visao: %v", d)
	}
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil, `{"version":2,"modo":"grupos"}`)
	esperar(t, "bloco pela metade", c, 400)
}

func TestConsultaExcluirELogicoEIdempotente(t *testing.T) { // C-T14
	f := cenario(t)
	id := nova(t, "mem", consultaBase)
	c, d := chamar(t, "mem", "DELETE", "spaces/s1/consultas/"+id, nil, "")
	esperar(t, "excluir", c, 200)
	if d["id"] != id {
		t.Errorf("DELETE devolve o id: %v", d)
	}
	it := itemConsulta(f, t, id)
	if it == nil || it["excluidoEm"] == nil || it["criadoPor"] != "mem" {
		t.Fatalf("o item continua na tabela, marcado: %v", it)
	}
	c, _ = chamar(t, "mem", "DELETE", "spaces/s1/consultas/"+id, nil, "")
	esperar(t, "excluir de novo", c, 200)
	if again := itemConsulta(f, t, id); again["version"] != it["version"] {
		t.Error("excluir o que ja esta excluido nao grava")
	}
	if len(listaConsultas(t, "mem")) != 0 {
		t.Error("a excluida some da lista")
	}
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil, `{"version":2,"nome":"x"}`)
	esperar(t, "editar uma excluida", c, 404)
}

func TestConsultaRestaurarRecusaNomeTomadoELimite(t *testing.T) { // C-T15
	cenario(t)
	id := nova(t, "mem", consultaBase)
	chamar(t, "mem", "DELETE", "spaces/s1/consultas/"+id, nil, "")
	nova(t, "own", consultaBase) // o nome foi tomado
	c, _ := chamar(t, "mem", "POST", "spaces/s1/consultas/"+id+"/restaurar", nil, "")
	esperar(t, "nome tomado", c, 400)

	cenario(t)
	id = nova(t, "mem", consultaBase)
	chamar(t, "mem", "DELETE", "spaces/s1/consultas/"+id, nil, "")
	for i := 0; i < domain.MaxConsultas; i++ {
		nova(t, "own", comNome("n"+itoa(i)))
	}
	c, _ = chamar(t, "mem", "POST", "spaces/s1/consultas/"+id+"/restaurar", nil, "")
	esperar(t, "limite atingido", c, 400)

	cenario(t)
	id = nova(t, "mem", consultaBase)
	c, d := chamar(t, "mem", "POST", "spaces/s1/consultas/"+id+"/restaurar", nil, "")
	esperar(t, "restaurar a que nao esta excluida", c, 200)
	if d["id"] != id {
		t.Errorf("devolve a mesma consulta: %v", d)
	}
}

func TestConsultaOutroEspacoOuInexistenteE404(t *testing.T) { // C-T16, C-T17
	f := cenario(t)
	f.colocar(t, doc{"PK": "USER#estranho", "SK": "SPACE#s2", "role": "owner"})
	id := nova(t, "own", consultaBase)
	for _, rota := range []struct{ m, p, corpo string }{
		{"PUT", "spaces/s2/consultas/" + id, `{"version":1,"nome":"x"}`},
		{"DELETE", "spaces/s2/consultas/" + id, ""},
		{"POST", "spaces/s2/consultas/" + id + "/restaurar", ""},
		{"PUT", "spaces/s1/consultas/naoexiste", `{"version":1,"nome":"x"}`},
		{"DELETE", "spaces/s1/consultas/naoexiste", ""},
		{"POST", "spaces/s1/consultas/naoexiste/restaurar", ""},
	} {
		sub := "estranho"
		if strings.Contains(rota.p, "s1") {
			sub = "own"
		}
		c, _ := chamar(t, sub, rota.m, rota.p, nil, rota.corpo)
		esperar(t, rota.m+" "+rota.p, c, 404)
	}
	if it := itemConsulta(f, t, id); it["nome"] != "Mercado sem presente" || it["excluidoEm"] != nil {
		t.Errorf("a consulta do s1 nao pode mudar: %v", it)
	}
	// o dono de outro espaco nao ve nem a lista, nem entra no s1
	c, _ := chamar(t, "estranho", "GET", "spaces/s1/consultas", nil, "")
	esperar(t, "estranho lista o s1", c, 403)
	c, _ = chamar(t, "estranho", "POST", "spaces/s1/consultas", nil, consultaBase)
	esperar(t, "estranho cria no s1", c, 403)
	if l := func() []any {
		r, _ := srvTeste.processar(context.Background(), &user{Sub: "estranho"}, "GET", []string{"spaces", "s2", "consultas"}, nil, "")
		var l []any
		_ = jsonUnmarshal(r.Body, &l)
		return l
	}(); len(l) != 0 {
		t.Errorf("o s2 nao ve as consultas do s1: %v", l)
	}
}

func TestConsultaNaoMexeNoSeqNemNoHistorico(t *testing.T) { // C-T18
	f := cenario(t)
	id := nova(t, "mem", consultaBase)
	chamar(t, "mem", "PUT", "spaces/s1/consultas/"+id, nil, `{"version":1,"nome":"Outro"}`)
	chamar(t, "mem", "DELETE", "spaces/s1/consultas/"+id, nil, "")
	chamar(t, "mem", "POST", "spaces/s1/consultas/"+id+"/restaurar", nil, "")
	for k := range f.itens {
		if strings.HasPrefix(k, "HIST#") || k == pkEspaco+"|SEQ" {
			t.Errorf("as consultas nao gravam em %s", k)
		}
	}
}

// contandoBanco soma as chamadas ao banco, para provar o custo de cada rota (R17).
type contandoBanco struct {
	*bancoFake
	get, put, query, upd int
}

func (c *contandoBanco) GetItem(ctx context.Context, in *dynamodb.GetItemInput, o ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	c.get++
	return c.bancoFake.GetItem(ctx, in, o...)
}
func (c *contandoBanco) PutItem(ctx context.Context, in *dynamodb.PutItemInput, o ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	c.put++
	return c.bancoFake.PutItem(ctx, in, o...)
}
func (c *contandoBanco) Query(ctx context.Context, in *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	c.query++
	return c.bancoFake.Query(ctx, in, o...)
}
func (c *contandoBanco) UpdateItem(ctx context.Context, in *dynamodb.UpdateItemInput, o ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	c.upd++
	return c.bancoFake.UpdateItem(ctx, in, o...)
}

func TestConsultaCustoDeCadaRota(t *testing.T) { // C-T19
	f := cenario(t)
	comApelidos(t, f)
	cb := &contandoBanco{bancoFake: f}
	srvTeste = novoServidor(cb, "", relogioTeste, notifTeste, "")
	conta := func(nome, m, p, corpo string, quer string) {
		t.Helper()
		cb.get, cb.put, cb.query, cb.upd = 0, 0, 0, 0
		c, _ := chamar(t, "own", m, p, nil, corpo)
		if c >= 400 {
			t.Fatalf("%s: status %d", nome, c)
		}
		// get inclui a leitura do papel e do apelido (1 por pessoa), que nao sao da consulta: so Query, Put e Update contam aqui
		if got := strings.Join([]string{itoa(cb.query), itoa(cb.put), itoa(cb.upd)}, "/"); got != quer {
			t.Errorf("%s: query/put/update = %s, esperado %s", nome, got, quer)
		}
	}
	conta("POST", "POST", "spaces/s1/consultas", consultaBase, "1/1/0")
	id := listaConsultas(t, "own")[0].(map[string]any)["id"].(string)
	conta("GET", "GET", "spaces/s1/consultas", "", "1/0/0")
	conta("PUT renomeia", "PUT", "spaces/s1/consultas/"+id, `{"version":1,"nome":"B"}`, "1/1/0")
	conta("PUT troca a consulta", "PUT", "spaces/s1/consultas/"+id, `{"version":2,"modo":"grupos","visao":"gastos","grupos":[{"tem":["a"]}]}`, "0/1/0")
	conta("DELETE", "DELETE", "spaces/s1/consultas/"+id, "", "0/1/0")
	conta("restaurar", "POST", "spaces/s1/consultas/"+id+"/restaurar", "", "1/1/0")
}

func TestConsultaLogDaRotaNaoTrazIdNemNome(t *testing.T) { // C-T20
	casos := []struct{ m, caminho, acao string }{
		{"GET", "spaces/s1/consultas", "consulta.ler"},
		{"POST", "spaces/s1/consultas", "consulta.criar"},
		{"PUT", "spaces/s1/consultas/idsecreto123", "consulta.editar"},
		{"DELETE", "spaces/s1/consultas/idsecreto123", "consulta.editar"},
		{"POST", "spaces/s1/consultas/idsecreto123/restaurar", "consulta.editar"},
	}
	for _, c := range casos {
		rt := rotular(c.m, strings.Split(c.caminho, "/"))
		if rt.Acao != c.acao || rt.Rota != "consultas" {
			t.Errorf("%s %s: rotulo %+v", c.m, c.caminho, rt)
		}
		linha := linhaDeLog("rid1", c.m, rt, 200, 0)
		for _, vaza := range []string{"idsecreto123", "s1", "Mercado"} {
			if strings.Contains(linha, vaza) {
				t.Errorf("a linha de log vaza %q: %s", vaza, linha)
			}
		}
	}
}
