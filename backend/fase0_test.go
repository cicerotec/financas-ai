package main

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Fase 0 do historico (.ai/specs/historico/spec.md): um unico caminho de escrita, sem mudar comportamento.

func itemFake(pk, sk string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: pk}, "SK": &types.AttributeValueMemberS{Value: sk}}
}

// F0-T2: o banco falso e atomico e honra attribute_not_exists, como o DynamoDB.
func TestFakeTransacaoETudoOuNada(t *testing.T) {
	f := novoBancoFake()
	f.colocar(t, doc{"PK": "P", "SK": "existe", "v": 1.0})
	f.colocar(t, doc{"PK": "P", "SK": "apagar", "v": 1.0})

	_, err := f.TransactWriteItems(context.Background(), &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Delete: &types.Delete{Key: itemFake("P", "apagar")}},
		{Put: &types.Put{Item: itemFake("P", "novo")}},
		{Put: &types.Put{Item: itemFake("P", "existe"), ConditionExpression: aws.String("attribute_not_exists(SK)")}},
	}})
	var cancelada *types.TransactionCanceledException
	if !errors.As(err, &cancelada) {
		t.Fatalf("a transacao deveria ser cancelada pela condicao, veio %v", err)
	}
	if f.ler(t, "P", "apagar") == nil {
		t.Error("o Delete nao pode ter sido aplicado: a transacao foi cancelada")
	}
	if f.ler(t, "P", "novo") != nil {
		t.Error("o Put nao pode ter sido aplicado: a transacao foi cancelada")
	}

	// sem violacao, tudo entra
	_, err = f.TransactWriteItems(context.Background(), &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Delete: &types.Delete{Key: itemFake("P", "apagar")}},
		{Put: &types.Put{Item: itemFake("P", "novo"), ConditionExpression: aws.String("attribute_not_exists(SK)")}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if f.ler(t, "P", "apagar") != nil || f.ler(t, "P", "novo") == nil {
		t.Error("a transacao valida deveria apagar e gravar")
	}

	// dois itens com a mesma chave na mesma transacao: o DynamoDB recusa
	_, err = f.TransactWriteItems(context.Background(), &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Put: &types.Put{Item: itemFake("P", "dup")}},
		{Put: &types.Put{Item: itemFake("P", "dup")}},
	}})
	if !errors.As(err, &cancelada) {
		t.Errorf("chave repetida na transacao deveria ser recusada, veio %v", err)
	}
}

// F0-T3: Query do banco falso respeita ScanIndexForward, Limit e ExclusiveStartKey.
func TestFakeQueryOrdemLimiteEContinuacao(t *testing.T) {
	f := novoBancoFake()
	for _, sk := range []string{"TX#a", "TX#b", "TX#c", "TX#d"} {
		f.colocar(t, doc{"PK": "P", "SK": sk})
	}
	f.colocar(t, doc{"PK": "OUTRA", "SK": "TX#z"})
	consulta := func(asc bool, limite int32, depoisDe string) ([]string, bool) {
		in := &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :s)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "P"}, ":s": &types.AttributeValueMemberS{Value: "TX#"}},
			ScanIndexForward: aws.Bool(asc),
		}
		if limite > 0 {
			in.Limit = aws.Int32(limite)
		}
		if depoisDe != "" {
			in.ExclusiveStartKey = itemFake("P", depoisDe)
		}
		out, err := f.Query(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		var sks []string
		for _, it := range out.Items {
			sks = append(sks, sv(it["SK"]))
		}
		return sks, out.LastEvaluatedKey != nil
	}
	if got, _ := consulta(false, 1, ""); strings.Join(got, ",") != "TX#d" {
		t.Errorf("o ultimo por SK (ScanIndexForward=false, Limit=1) deveria ser TX#d, veio %v", got)
	}
	got, mais := consulta(true, 2, "")
	if strings.Join(got, ",") != "TX#a,TX#b" || !mais {
		t.Errorf("primeira pagina: %v (mais=%v)", got, mais)
	}
	got, mais = consulta(true, 2, "TX#b")
	if strings.Join(got, ",") != "TX#c,TX#d" || mais {
		t.Errorf("segunda pagina: %v (mais=%v)", got, mais)
	}
	if got, _ := consulta(false, 0, "TX#c"); strings.Join(got, ",") != "TX#b,TX#a" {
		t.Errorf("continuacao em ordem decrescente: %v", got)
	}
}

// F0-T4: os controllers de lancamento nao chegam ao banco, e o service nao importa o SDK da AWS.
func TestControllersDeLancamentoNaoChamamOBanco(t *testing.T) {
	proibidos := map[string]bool{"db": true, "dynamodb": true, "types": true, "attributevalue": true,
		"table": true, "getItem": true, "marshal": true, "key": true}
	fset := token.NewFileSet()
	arq, err := parser.ParseFile(fset, "controller_lancamento.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	achados := map[string]bool{}
	for _, d := range arq.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		achados[fn.Name.Name] = true
		ast.Inspect(fn, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && proibidos[id.Name] {
				t.Errorf("%s usa %q: controller nao toca o banco, so o service", fn.Name.Name, id.Name)
			}
			return true
		})
	}
	for _, nome := range []string{"postTx", "putTx", "delTx", "postTxBatch"} {
		if !achados[nome] {
			t.Errorf("controller %s nao encontrado em controller_lancamento.go (o teste nao pode passar vazio)", nome)
		}
	}

	srv, err := parser.ParseFile(fset, "servico_lancamento.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, im := range srv.Imports {
		if p, _ := strconv.Unquote(im.Path.Value); strings.Contains(p, "aws") {
			t.Errorf("servico_lancamento.go importa %s: o service nao conhece o SDK da AWS", p)
		}
	}
}

// ---------- o comportamento que a refatoracao nao pode mudar ----------

func TestEditarMudandoADataTrocaAChave(t *testing.T) {
	f := cenario(t)
	nova := "2026-01-05T10:00:00.000Z"
	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"dataEvento":"`+nova+`"}`)
	esperar(t, "owner muda a data", c, 200)
	if d["dataEvento"] != nova {
		t.Errorf("dataEvento na resposta = %v", d["dataEvento"])
	}
	if f.ler(t, pkEspaco, dePrefixoTx+dataDono+"#iddono") != nil {
		t.Error("a chave antiga deveria ter sumido")
	}
	if novo := f.ler(t, pkEspaco, dePrefixoTx+nova+"#iddono"); novo == nil || novo["descricao"] != "original" {
		t.Errorf("a chave nova deveria existir com o resto do lancamento: %v", novo)
	}
}

func TestEditarSemParametroDeOuInexistente(t *testing.T) {
	cenario(t)
	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", nil, `{"descricao":"x"}`)
	esperar(t, "PUT sem de", c, 400)
	if d["erro"] != "parametro de (dataEvento atual) obrigatorio" {
		t.Errorf("mensagem = %v", d["erro"])
	}
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/naoexiste", map[string]string{"de": dataDono}, `{"descricao":"x"}`)
	esperar(t, "PUT de lancamento inexistente", c, 404)
	if d["erro"] != "lancamento nao encontrado" {
		t.Errorf("mensagem = %v", d["erro"])
	}
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"valor":"nao e numero"}`)
	esperar(t, "PUT com valor invalido", c, 400)
	if d["erro"] != "valor deve ser numerico" {
		t.Errorf("mensagem = %v", d["erro"])
	}
}

func TestExcluirInexistenteRespondeOkSemSeq(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "own", "DELETE", "spaces/s1/tx/naoexiste", map[string]string{"de": dataDono}, "")
	esperar(t, "DELETE de lancamento inexistente", c, 200)
	if d["ok"] != true || d["_seq"] != nil {
		t.Errorf("deveria responder so {ok:true}, veio %v", d)
	}
	if f.ler(t, "SPACE#s1", "SEQ") != nil {
		t.Error("nada foi excluido: o contador de alteracoes nao pode andar")
	}
	c, d = chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", nil, "")
	esperar(t, "DELETE sem de", c, 400)
	_ = d
}

func TestImportarRejeitaDuplicadoEItemInvalido(t *testing.T) {
	f := cenario(t)
	item := `{"id":"idimportado1","dataEvento":"2026-03-01T10:00:00.000Z","status":"PAGO","tipo":"saida","valor":1,"banco":"ITAU","descricao":"imp"}`
	lote := `{"itens":[` + item + `,` + item + `,{"valor":1},"texto"]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "importa lote", c, 200)
	if d["gravados"] != 1.0 {
		t.Errorf("gravados = %v, esperado 1", d["gravados"])
	}
	rej, _ := d["rejeitados"].([]any)
	if len(rej) != 3 {
		t.Fatalf("rejeitados = %v, esperado 3 (duplicado, sem campos, nao-objeto)", d["rejeitados"])
	}
	if m := rej[0].(map[string]any)["motivo"]; m != "duplicado no mesmo lote" {
		t.Errorf("motivo do primeiro rejeitado = %v", m)
	}
	if p, _ := d["pendentes"].([]any); len(p) != 0 {
		t.Errorf("pendentes = %v", d["pendentes"])
	}
	if f.ler(t, pkEspaco, dePrefixoTx+"2026-03-01T10:00:00.000Z#idimportado1") == nil {
		t.Error("o item valido deveria ter sido gravado com o id preservado")
	}
	if d["_seq"] == nil {
		t.Error("algo foi gravado: a resposta deve trazer _seq")
	}
	c, d = chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, `{"itens":[]}`)
	esperar(t, "lote vazio", c, 400)
	if d["erro"] != "itens deve ter de 1 a 25 lancamentos" {
		t.Errorf("mensagem = %v", d["erro"])
	}
}
