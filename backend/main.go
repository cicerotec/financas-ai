package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type resp = events.LambdaFunctionURLResponse
type req = events.LambdaFunctionURLRequest

var cfgNomes = map[string]bool{"listas": true, "tags": true, "combos": true, "saldos": true, "cartoes": true}

func out(status int, v any) (resp, error) {
	b, _ := json.Marshal(v)
	return resp{StatusCode: status, Headers: map[string]string{"content-type": "application/json"}, Body: string(b)}, nil
}
func erro(status int, msg string) (resp, error) { return out(status, doc{"erro": msg}) }

func handler(ctx context.Context, r req) (resp, error) {
	u, err := verificarToken(strings.TrimPrefix(r.Headers["authorization"], "Bearer "))
	if err != nil {
		return erro(http.StatusUnauthorized, "nao autenticado")
	}
	m := r.RequestContext.HTTP.Method
	p := strings.Split(strings.Trim(r.RawPath, "/"), "/")

	if len(p) == 1 && p[0] == "me" && m == "GET" {
		return getMe(ctx, u)
	}
	if len(p) < 3 || p[0] != "spaces" {
		return erro(http.StatusNotFound, "rota inexistente")
	}
	sid := p[1]
	role, err := papel(ctx, u, sid)
	if err != nil {
		log.Println("papel:", err)
		return erro(http.StatusInternalServerError, "erro interno")
	}
	if role == "" {
		return erro(http.StatusForbidden, "sem acesso a esse espaco")
	}
	var body doc
	if m == "POST" || m == "PUT" {
		if err := json.Unmarshal([]byte(r.Body), &body); err != nil || body == nil {
			return erro(http.StatusBadRequest, "json invalido")
		}
	}
	switch {
	case p[2] == "tx" && len(p) == 3 && m == "GET":
		return getTxs(ctx, sid, r.QueryStringParameters)
	case p[2] == "tx" && len(p) == 3 && m == "POST":
		return postTx(ctx, sid, u, body)
	case p[2] == "tx" && len(p) == 4 && m == "PUT":
		return putTx(ctx, sid, p[3], r.QueryStringParameters["de"], body)
	case p[2] == "tx" && len(p) == 4 && m == "DELETE":
		return delTx(ctx, sid, p[3], r.QueryStringParameters["de"])
	case p[2] == "cfg" && len(p) == 4 && cfgNomes[p[3]] && m == "GET":
		return getCfg(ctx, spacePK(sid), "CFG#"+strings.ToUpper(p[3]))
	case p[2] == "cfg" && len(p) == 4 && cfgNomes[p[3]] && m == "PUT":
		return putCfg(ctx, spacePK(sid), "CFG#"+strings.ToUpper(p[3]), body)
	case p[2] == "aparencia" && len(p) == 3 && m == "GET":
		return getCfg(ctx, spacePK(sid), "USER#"+u.Sub+"#CFG#APARENCIA")
	case p[2] == "aparencia" && len(p) == 3 && m == "PUT":
		return putAparencia(ctx, spacePK(sid), "USER#"+u.Sub+"#CFG#APARENCIA", body)
	}
	return erro(http.StatusNotFound, "rota inexistente")
}

func getMe(ctx context.Context, u *user) (resp, error) {
	q, err := db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :s)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "USER#" + u.Sub},
			":s":  &types.AttributeValueMemberS{Value: "SPACE#"},
		},
	})
	if err != nil {
		log.Println("me:", err)
		return erro(500, "erro interno")
	}
	espacos := []doc{}
	for _, it := range q.Items {
		var d doc
		if attributevalue.UnmarshalMap(it, &d) != nil {
			continue
		}
		id := strings.TrimPrefix(d["SK"].(string), "SPACE#")
		meta, _ := getItem(ctx, spacePK(id), "META")
		nome := id
		if meta != nil {
			if n, ok := meta["nome"].(string); ok {
				nome = n
			}
		}
		espacos = append(espacos, doc{"id": id, "nome": nome, "role": d["role"]})
	}
	return out(200, doc{"sub": u.Sub, "email": u.Email, "espacos": espacos})
}

func getTxs(ctx context.Context, sid string, qs map[string]string) (resp, error) {
	f := filtroTx{De: qs["de"], Ate: qs["ate"], Banco: qs["banco"], Q: qs["q"], Status: qs["status"],
		Asc: qs["ordem"] == "asc", Limite: 200, Cursor: qs["cursor"]}
	if n, err := strconv.Atoi(qs["limite"]); err == nil && n > 0 && n <= 1000 {
		f.Limite = n
	}
	itens, cursor, err := listarTx(ctx, sid, f)
	if err != nil {
		log.Println("listar:", err)
		return erro(500, "erro ao listar")
	}
	if itens == nil {
		itens = []doc{}
	}
	return out(200, doc{"itens": itens, "proximo": cursor})
}

// valida e normaliza os campos obrigatorios de um lancamento
func validarTx(d doc) error {
	s, _ := d["dataEvento"].(string)
	if s == "" {
		return errors.New("dataEvento obrigatoria")
	}
	n, err := normData(s)
	if err != nil {
		return err
	}
	d["dataEvento"] = n
	if v, ok := d["valor"].(float64); !ok || v <= 0 {
		return errors.New("valor deve ser > 0")
	}
	if t := d["tipo"]; t != "entrada" && t != "saida" {
		return errors.New("tipo deve ser entrada ou saida")
	}
	for _, c := range []string{"status", "banco"} {
		if s, _ := d[c].(string); s == "" {
			return errors.New(c + " obrigatorio")
		}
	}
	return nil
}

func protegidos(d doc) {
	for _, k := range []string{"PK", "SK", "id", "criadoEm", "criadoPor"} {
		delete(d, k)
	}
}

func postTx(ctx context.Context, sid string, u *user, body doc) (resp, error) {
	protegidos(body)
	if err := validarTx(body); err != nil {
		return erro(400, err.Error())
	}
	id := novoID()
	body["id"] = id
	body["criadoEm"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	body["criadoPor"] = u.Sub
	body["PK"] = spacePK(sid)
	body["SK"] = txSK(body["dataEvento"].(string), id)
	item, err := marshal(body)
	if err != nil {
		return erro(400, "dados invalidos")
	}
	if _, err := db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &table, Item: item,
		ConditionExpression: aws.String("attribute_not_exists(PK)")}); err != nil {
		log.Println("post tx:", err)
		return erro(500, "erro ao salvar")
	}
	return out(201, publico(body))
}

func putTx(ctx context.Context, sid, id, de string, body doc) (resp, error) {
	de, err := normData(de)
	if err != nil {
		return erro(400, "parametro de (dataEvento atual) obrigatorio")
	}
	antigo, err := getItem(ctx, spacePK(sid), txSK(de, id))
	if err != nil {
		return erro(500, "erro interno")
	}
	if antigo == nil {
		return erro(404, "lancamento nao encontrado")
	}
	protegidos(body)
	novo := antigo
	for k, v := range body {
		novo[k] = v
	}
	if err := validarTx(novo); err != nil {
		return erro(400, err.Error())
	}
	novo["SK"] = txSK(novo["dataEvento"].(string), id)
	item, err := marshal(novo)
	if err != nil {
		return erro(400, "dados invalidos")
	}
	if novo["SK"] == txSK(de, id) {
		_, err = db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &table, Item: item})
	} else { // dataEvento mudou: a chave muda, entao apaga a antiga e grava a nova juntas
		_, err = db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
			{Delete: &types.Delete{TableName: &table, Key: key(spacePK(sid), txSK(de, id))}},
			{Put: &types.Put{TableName: &table, Item: item}},
		}})
	}
	if err != nil {
		log.Println("put tx:", err)
		return erro(500, "erro ao salvar")
	}
	return out(200, publico(novo))
}

func delTx(ctx context.Context, sid, id, de string) (resp, error) {
	de, err := normData(de)
	if err != nil {
		return erro(400, "parametro de (dataEvento atual) obrigatorio")
	}
	if _, err := db.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: &table, Key: key(spacePK(sid), txSK(de, id))}); err != nil {
		log.Println("del tx:", err)
		return erro(500, "erro ao excluir")
	}
	return out(200, doc{"ok": true})
}

func getCfg(ctx context.Context, pk, sk string) (resp, error) {
	d, err := getItem(ctx, pk, sk)
	if err != nil {
		return erro(500, "erro interno")
	}
	if d == nil {
		return out(200, doc{"version": 0})
	}
	return out(200, publico(d))
}

// Configuracao compartilhada: controle otimista. O cliente manda o "version" que leu;
// se alguem gravou antes, responde 409 com o estado atual para o cliente mesclar.
func putCfg(ctx context.Context, pk, sk string, body doc) (resp, error) {
	esperada, _ := body["version"].(float64)
	delete(body, "version")
	protegidos(body)
	body["PK"], body["SK"], body["version"] = pk, sk, esperada+1
	item, err := marshal(body)
	if err != nil {
		return erro(400, "dados invalidos")
	}
	in := &dynamodb.PutItemInput{TableName: &table, Item: item}
	if esperada == 0 {
		in.ConditionExpression = aws.String("attribute_not_exists(PK) OR attribute_not_exists(version)")
	} else {
		in.ConditionExpression = aws.String("version = :v")
		in.ExpressionAttributeValues = map[string]types.AttributeValue{":v": &types.AttributeValueMemberN{Value: strconv.FormatFloat(esperada, 'f', -1, 64)}}
	}
	if _, err := db.PutItem(ctx, in); err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			atual, _ := getItem(ctx, pk, sk)
			return out(http.StatusConflict, doc{"erro": "alterado por outra pessoa", "atual": publico(atual)})
		}
		log.Println("put cfg:", err)
		return erro(500, "erro ao salvar")
	}
	return out(200, publico(body))
}

func putAparencia(ctx context.Context, pk, sk string, body doc) (resp, error) {
	protegidos(body)
	body["PK"], body["SK"] = pk, sk
	item, err := marshal(body)
	if err != nil {
		return erro(400, "dados invalidos")
	}
	if _, err := db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &table, Item: item}); err != nil {
		log.Println("put aparencia:", err)
		return erro(500, "erro ao salvar")
	}
	return out(200, publico(body))
}

func main() {
	if err := initDB(context.Background()); err != nil {
		log.Fatal(err)
	}
	lambda.Start(handler)
}
