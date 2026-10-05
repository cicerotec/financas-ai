package main

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ---------- DynamoDB em memoria (so o que o codigo usa) ----------

type bancoFake struct {
	itens map[string]map[string]types.AttributeValue
}

func novoBancoFake() *bancoFake {
	return &bancoFake{itens: map[string]map[string]types.AttributeValue{}}
}

func sv(a types.AttributeValue) string { return a.(*types.AttributeValueMemberS).Value }
func chave(pk, sk string) string       { return pk + "|" + sk }

func (f *bancoFake) GetItem(_ context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	it, ok := f.itens[chave(sv(in.Key["PK"]), sv(in.Key["SK"]))]
	if !ok {
		return &dynamodb.GetItemOutput{}, nil
	}
	return &dynamodb.GetItemOutput{Item: it}, nil
}
func (f *bancoFake) PutItem(_ context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	k := chave(sv(in.Item["PK"]), sv(in.Item["SK"]))
	atual, existe := f.itens[k]
	switch aws.ToString(in.ConditionExpression) {
	case "attribute_not_exists(PK)":
		if existe {
			return nil, &types.ConditionalCheckFailedException{}
		}
	case "attribute_not_exists(PK) OR attribute_not_exists(version)":
		if existe && atual["version"] != nil {
			return nil, &types.ConditionalCheckFailedException{}
		}
	case "version = :v":
		quer, _ := strconv.ParseFloat(in.ExpressionAttributeValues[":v"].(*types.AttributeValueMemberN).Value, 64)
		var tem float64
		if n, ok := atual["version"].(*types.AttributeValueMemberN); ok {
			tem, _ = strconv.ParseFloat(n.Value, 64)
		}
		if !existe || tem != quer {
			return nil, &types.ConditionalCheckFailedException{}
		}
	}
	f.itens[k] = in.Item
	return &dynamodb.PutItemOutput{}, nil
}
func (f *bancoFake) DeleteItem(_ context.Context, in *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	delete(f.itens, chave(sv(in.Key["PK"]), sv(in.Key["SK"])))
	return &dynamodb.DeleteItemOutput{}, nil
}
func (f *bancoFake) TransactWriteItems(_ context.Context, in *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	for _, t := range in.TransactItems {
		if t.Delete != nil {
			delete(f.itens, chave(sv(t.Delete.Key["PK"]), sv(t.Delete.Key["SK"])))
		}
		if t.Put != nil {
			f.itens[chave(sv(t.Put.Item["PK"]), sv(t.Put.Item["SK"]))] = t.Put.Item
		}
	}
	return &dynamodb.TransactWriteItemsOutput{}, nil
}
func (f *bancoFake) BatchWriteItem(_ context.Context, in *dynamodb.BatchWriteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	for _, reqs := range in.RequestItems {
		for _, w := range reqs {
			if w.PutRequest != nil {
				f.itens[chave(sv(w.PutRequest.Item["PK"]), sv(w.PutRequest.Item["SK"]))] = w.PutRequest.Item
			}
		}
	}
	return &dynamodb.BatchWriteItemOutput{}, nil
}

// UpdateItem: so o que o perfil usa. Sem o item, falha como a condicao attribute_exists(PK) do DynamoDB.
func (f *bancoFake) UpdateItem(_ context.Context, in *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	k := chave(sv(in.Key["PK"]), sv(in.Key["SK"]))
	it, ok := f.itens[k]
	if !ok {
		return nil, &types.ConditionalCheckFailedException{}
	}
	switch aws.ToString(in.UpdateExpression) {
	case "SET apelido = :a":
		it["apelido"] = in.ExpressionAttributeValues[":a"]
	case "REMOVE apelido":
		delete(it, "apelido")
	default:
		return nil, errors.New("UpdateExpression nao suportada no banco simulado")
	}
	return &dynamodb.UpdateItemOutput{}, nil
}

// Query: entende "PK = :pk AND SK BETWEEN :a AND :b" e "PK = :pk AND begins_with(SK, :s)".
func (f *bancoFake) Query(_ context.Context, in *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	pk := sv(in.ExpressionAttributeValues[":pk"])
	var sks []string
	for k, it := range f.itens {
		if sv(it["PK"]) != pk {
			continue
		}
		sk := sv(it["SK"])
		if a, ok := in.ExpressionAttributeValues[":a"]; ok {
			if sk < sv(a) || sk > sv(in.ExpressionAttributeValues[":b"]) {
				continue
			}
		}
		if s, ok := in.ExpressionAttributeValues[":s"]; ok && !strings.HasPrefix(sk, sv(s)) {
			continue
		}
		sks = append(sks, k)
	}
	sort.Strings(sks)
	out := &dynamodb.QueryOutput{}
	for _, k := range sks {
		out.Items = append(out.Items, f.itens[k])
	}
	return out, nil
}

func (f *bancoFake) colocar(t *testing.T, d doc) {
	t.Helper()
	it, err := attributevalue.MarshalMap(d)
	if err != nil {
		t.Fatal(err)
	}
	f.itens[chave(d["PK"].(string), d["SK"].(string))] = it
}

func (f *bancoFake) ler(t *testing.T, pk, sk string) doc {
	t.Helper()
	it, ok := f.itens[chave(pk, sk)]
	if !ok {
		return nil
	}
	var d doc
	if err := attributevalue.UnmarshalMap(it, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// ---------- cenario ----------
// espaco s1: "own" (owner), "mem" (member). "estranho" nao pertence a nenhum espaco.

const (
	dataDono    = "2026-01-01T10:00:00.000Z"
	dataMembro  = "2026-01-02T10:00:00.000Z"
	dataLegado  = "2026-01-03T10:00:00.000Z"
	pkEspaco    = "SPACE#s1"
	dePrefixoTx = "TX#"
)

func cenario(t *testing.T) *bancoFake {
	f := novoBancoFake()
	db = f
	f.colocar(t, doc{"PK": "USER#own", "SK": "SPACE#s1", "role": "owner"})
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member"})
	f.colocar(t, doc{"PK": pkEspaco, "SK": "META", "nome": "Familia"})
	tx := func(data, id, criadoPor string) doc {
		d := doc{"PK": pkEspaco, "SK": dePrefixoTx + data + "#" + id, "id": id, "dataEvento": data,
			"status": "PAGO", "tipo": "saida", "valor": 10.0, "banco": "ITAU", "descricao": "original", "tags": []any{"Lazer"}}
		if criadoPor != "" {
			d["criadoPor"] = criadoPor
		}
		return d
	}
	f.colocar(t, tx(dataDono, "iddono", "own"))
	f.colocar(t, tx(dataMembro, "idmembro", "mem"))
	f.colocar(t, tx(dataLegado, "idlegado", "")) // sem criadoPor
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#TAGS", "usadas": []any{"Lazer"}, "version": 1.0})
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#LISTAS", "banco": []any{"ITAU"}, "version": 1.0})
	return f
}

func chamar(t *testing.T, sub, metodo, caminho string, qs map[string]string, corpo string) (int, doc) {
	t.Helper()
	r, _ := processar(context.Background(), &user{Sub: sub}, metodo, strings.Split(caminho, "/"), qs, corpo)
	var d doc
	_ = json.Unmarshal([]byte(r.Body), &d)
	return r.StatusCode, d
}

func esperar(t *testing.T, nome string, got, quer int) {
	t.Helper()
	if got != quer {
		t.Errorf("%s: status %d, esperado %d", nome, got, quer)
	}
}

func tagsDoBanco(t *testing.T, f *bancoFake) []string {
	d := f.ler(t, pkEspaco, "CFG#TAGS")
	var r []string
	for _, x := range d["usadas"].([]any) {
		r = append(r, x.(string))
	}
	return r
}

// ---------- testes ----------

func TestMemberEditaSoOQueCriou(t *testing.T) {
	f := cenario(t)
	c, _ := chamar(t, "mem", "PUT", "spaces/s1/tx/idmembro", map[string]string{"de": dataMembro}, `{"descricao":"editado por ela"}`)
	esperar(t, "member edita o proprio", c, 200)
	if d := f.ler(t, pkEspaco, dePrefixoTx+dataMembro+"#idmembro"); d["descricao"] != "editado por ela" {
		t.Errorf("a edicao do proprio registro deveria ter sido gravada: %v", d["descricao"])
	}

	c, _ = chamar(t, "mem", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"descricao":"invasao"}`)
	esperar(t, "member edita registro do owner", c, 403)
	if d := f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono"); d["descricao"] != "original" {
		t.Errorf("registro do owner foi alterado pelo member: %v", d["descricao"])
	}

	c, _ = chamar(t, "mem", "PUT", "spaces/s1/tx/idlegado", map[string]string{"de": dataLegado}, `{"oculto":true}`)
	esperar(t, "member oculta registro sem criadoPor", c, 403)
	if d := f.ler(t, pkEspaco, dePrefixoTx+dataLegado+"#idlegado"); d["oculto"] != nil {
		t.Error("registro legado nao pode ser ocultado pelo member")
	}
}

func TestMemberPodeOcultarOProprio(t *testing.T) {
	f := cenario(t)
	c, _ := chamar(t, "mem", "PUT", "spaces/s1/tx/idmembro", map[string]string{"de": dataMembro}, `{"oculto":true}`)
	esperar(t, "member oculta o proprio", c, 200)
	if d := f.ler(t, pkEspaco, dePrefixoTx+dataMembro+"#idmembro"); d["oculto"] != true {
		t.Error("o registro dela deveria estar oculto")
	}
}

func TestOwnerEditaEExcluiQualquerUm(t *testing.T) {
	f := cenario(t)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/tx/idmembro", map[string]string{"de": dataMembro}, `{"descricao":"owner mexeu"}`)
	esperar(t, "owner edita registro do member", c, 200)
	c, _ = chamar(t, "own", "DELETE", "spaces/s1/tx/idlegado", map[string]string{"de": dataLegado}, "")
	esperar(t, "owner exclui", c, 200)
	if f.ler(t, pkEspaco, dePrefixoTx+dataLegado+"#idlegado") != nil {
		t.Error("registro deveria ter sido excluido pelo owner")
	}
}

func TestMemberNaoExclui(t *testing.T) {
	f := cenario(t)
	// nem o que ela mesma criou
	c, _ := chamar(t, "mem", "DELETE", "spaces/s1/tx/idmembro", map[string]string{"de": dataMembro}, "")
	esperar(t, "member exclui o proprio", c, 403)
	c, _ = chamar(t, "mem", "DELETE", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, "")
	esperar(t, "member exclui do owner", c, 403)
	if f.ler(t, pkEspaco, dePrefixoTx+dataMembro+"#idmembro") == nil || f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono") == nil {
		t.Error("nenhum registro deveria ter sido excluido")
	}
}

func TestMemberNaoForjaCriadoPor(t *testing.T) {
	f := cenario(t)
	// cria dizendo que quem criou foi o owner
	c, d := chamar(t, "mem", "POST", "spaces/s1/tx", nil,
		`{"dataEvento":"2026-02-01T10:00:00.000Z","status":"PAGO","tipo":"saida","valor":5,"banco":"ITAU","descricao":"x","criadoPor":"own","id":"idforjado"}`)
	esperar(t, "member cria lancamento", c, 201)
	if d["criadoPor"] != "mem" {
		t.Errorf("criadoPor deve ser quem fez a chamada, veio %v", d["criadoPor"])
	}
	if d["id"] == "idforjado" {
		t.Error("o id e gerado pelo servidor")
	}
	// edita o proprio tentando passar a posse para o owner
	c, d = chamar(t, "mem", "PUT", "spaces/s1/tx/idmembro", map[string]string{"de": dataMembro}, `{"criadoPor":"own"}`)
	esperar(t, "member edita o proprio", c, 200)
	if d["criadoPor"] != "mem" {
		t.Errorf("criadoPor nao pode ser trocado, veio %v", d["criadoPor"])
	}
	_ = f
}

func TestMemberCriaSoTagsNovasEntramNaLista(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "mem", "POST", "spaces/s1/tags", nil, `{"tags":["Lazer","#Viagem"," Nova "]}`)
	esperar(t, "member adiciona tags", c, 200)
	add, _ := d["adicionadas"].([]any)
	if len(add) != 2 {
		t.Errorf("so as 2 novas deviam ser adicionadas, veio %v", d["adicionadas"])
	}
	got := tagsDoBanco(t, f)
	if strings.Join(got, ",") != "Lazer,Viagem,Nova" {
		t.Errorf("lista de tags = %v", got)
	}
	// repetir nao muda nada
	_, d = chamar(t, "mem", "POST", "spaces/s1/tags", nil, `{"tags":["Viagem"]}`)
	if add, _ := d["adicionadas"].([]any); len(add) != 0 {
		t.Errorf("nada novo para adicionar: %v", d["adicionadas"])
	}
	c, _ = chamar(t, "mem", "POST", "spaces/s1/tags", nil, `{"tags":["", "#"]}`)
	esperar(t, "lista vazia", c, 400)
}

func TestMemberNaoReescreveAListaDeTags(t *testing.T) {
	f := cenario(t)
	// tentativa de apagar tags reescrevendo a lista
	c, _ := chamar(t, "mem", "PUT", "spaces/s1/cfg/tags", nil, `{"usadas":[],"version":1}`)
	esperar(t, "member reescreve cfg/tags", c, 403)
	if got := tagsDoBanco(t, f); strings.Join(got, ",") != "Lazer" {
		t.Errorf("a lista nao pode ter mudado: %v", got)
	}
	c, _ = chamar(t, "own", "PUT", "spaces/s1/cfg/tags", nil, `{"usadas":["Lazer","Outra"],"version":1}`)
	esperar(t, "owner reescreve cfg/tags", c, 200)
}

func TestTagNovaEmLancamentoEntraNaLista(t *testing.T) {
	f := cenario(t)
	c, _ := chamar(t, "mem", "POST", "spaces/s1/tx", nil,
		`{"dataEvento":"2026-02-01T10:00:00.000Z","status":"PAGO","tipo":"saida","valor":5,"banco":"ITAU","descricao":"x","tags":["Lazer","Praia"]}`)
	esperar(t, "member cria lancamento com tag nova", c, 201)
	if got := tagsDoBanco(t, f); strings.Join(got, ",") != "Lazer,Praia" {
		t.Errorf("a tag nova deveria entrar na lista: %v", got)
	}
}

func TestMemberNaoMexeEmConfiguracoesCompartilhadas(t *testing.T) {
	f := cenario(t)
	for _, nome := range []string{"listas", "saldos", "cartoes", "combos", "tags"} {
		c, _ := chamar(t, "mem", "PUT", "spaces/s1/cfg/"+nome, nil, `{"version":1,"x":1}`)
		esperar(t, "member PUT cfg/"+nome, c, 403)
	}
	if d := f.ler(t, pkEspaco, "CFG#LISTAS"); d["x"] != nil {
		t.Error("CFG#LISTAS nao pode ser alterado pelo member")
	}
	// mas pode ler
	for _, nome := range []string{"listas", "saldos", "cartoes", "combos", "tags"} {
		c, _ := chamar(t, "mem", "GET", "spaces/s1/cfg/"+nome, nil, "")
		esperar(t, "member GET cfg/"+nome, c, 200)
	}
	c, _ := chamar(t, "own", "PUT", "spaces/s1/cfg/saldos", nil, `{"porBanco":{}}`)
	esperar(t, "owner PUT cfg/saldos", c, 200)
}

func TestMemberNaoImporta(t *testing.T) {
	f := cenario(t)
	lote := `{"itens":[{"dataEvento":"2026-03-01T10:00:00.000Z","status":"PAGO","tipo":"saida","valor":1,"banco":"ITAU","descricao":"imp"}]}`
	c, _ := chamar(t, "mem", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "member importa em lote", c, 403)
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "owner importa em lote", c, 200)
	if d["gravados"] != 1.0 {
		t.Errorf("owner deveria gravar 1, veio %v", d["gravados"])
	}
	_ = f
}

func TestAparenciaEPorUsuario(t *testing.T) {
	f := cenario(t)
	c, _ := chamar(t, "mem", "PUT", "spaces/s1/aparencia", nil, `{"estiloCor":"faixa"}`)
	esperar(t, "member grava a propria aparencia", c, 200)
	if f.ler(t, pkEspaco, "USER#mem#CFG#APARENCIA")["estiloCor"] != "faixa" {
		t.Error("a aparencia deveria ficar no item do member")
	}
	if f.ler(t, pkEspaco, "USER#own#CFG#APARENCIA") != nil {
		t.Error("a aparencia do owner nao pode ser criada nem alterada pelo member")
	}
	c, d := chamar(t, "own", "GET", "spaces/s1/aparencia", nil, "")
	esperar(t, "owner le a propria aparencia", c, 200)
	if d["estiloCor"] != nil {
		t.Error("o owner nao deve ver a aparencia do member")
	}
}

func TestMemberLeLancamentos(t *testing.T) {
	cenario(t)
	c, d := chamar(t, "mem", "GET", "spaces/s1/tx", nil, "")
	esperar(t, "member lista lancamentos", c, 200)
	if itens, _ := d["itens"].([]any); len(itens) != 3 {
		t.Errorf("deveria ler os 3 lancamentos, veio %d", len(itens))
	}
}

func TestQuemNaoEDoEspacoNaoFazNada(t *testing.T) {
	f := cenario(t)
	casos := []struct{ m, caminho, corpo string }{
		{"GET", "spaces/s1/tx", ""},
		{"POST", "spaces/s1/tx", `{"dataEvento":"2026-02-01T10:00:00.000Z","status":"PAGO","tipo":"saida","valor":5,"banco":"ITAU"}`},
		{"PUT", "spaces/s1/tx/iddono", `{"descricao":"x"}`},
		{"DELETE", "spaces/s1/tx/iddono", ""},
		{"GET", "spaces/s1/cfg/listas", ""},
		{"PUT", "spaces/s1/cfg/listas", `{"version":1}`},
		{"POST", "spaces/s1/tags", `{"tags":["X"]}`},
		{"GET", "spaces/s1/aparencia", ""},
		{"POST", "spaces/s1/tx/batch", `{"itens":[{}]}`},
	}
	for _, c := range casos {
		st, _ := chamar(t, "estranho", c.m, c.caminho, map[string]string{"de": dataDono}, c.corpo)
		esperar(t, "estranho "+c.m+" "+c.caminho, st, 403)
	}
	if d := f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono"); d["descricao"] != "original" {
		t.Error("o estranho alterou dados")
	}
	if got := tagsDoBanco(t, f); strings.Join(got, ",") != "Lazer" {
		t.Errorf("o estranho alterou as tags: %v", got)
	}
}

func TestMemberDeUmEspacoNaoAcessaOutro(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": "SPACE#s2", "SK": "CFG#LISTAS", "banco": []any{"SEGREDO"}, "version": 1.0})
	c, _ := chamar(t, "mem", "GET", "spaces/s2/cfg/listas", nil, "")
	esperar(t, "member le espaco que nao e dele", c, 403)
	c, _ = chamar(t, "own", "GET", "spaces/s2/cfg/listas", nil, "")
	esperar(t, "owner de s1 le s2", c, 403)
}

func TestRotaInexistente(t *testing.T) {
	cenario(t)
	for _, c := range []struct{ m, caminho string }{
		{"GET", "spaces/s1/nada"}, {"GET", "nada"}, {"GET", "spaces/s1"}, {"PATCH", "spaces/s1/tx/iddono"},
	} {
		st, _ := chamar(t, "own", c.m, c.caminho, nil, "")
		esperar(t, c.m+" "+c.caminho, st, 404)
	}
}
