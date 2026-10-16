package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
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

	return processar(ctx, u, m, p, r.QueryStringParameters, r.Body)
}

// processar roda uma requisicao ja autenticada: classifica a rota, confere o papel e executa.
func processar(ctx context.Context, u *user, m string, p []string, qs map[string]string, rawBody string) (resp, error) {
	if len(p) == 1 && p[0] == "me" && m == "GET" {
		return getMe(ctx, u)
	}
	if len(p) < 3 || p[0] != "spaces" {
		return erro(http.StatusNotFound, "rota inexistente")
	}
	sid := p[1]
	acaoReq, ok := rotaParaAcao(m, p[2:])
	if !ok {
		return erro(http.StatusNotFound, "rota inexistente")
	}
	role, err := papel(ctx, u, sid)
	if err != nil {
		log.Println("papel:", err)
		return erro(http.StatusInternalServerError, "erro interno")
	}
	if role == "" {
		return erro(http.StatusForbidden, "sem acesso a esse espaco")
	}
	dec := permitido(role, acaoReq)
	if dec == negado {
		log.Printf("negado: sub=%s papel=%s acao=%s", u.Sub, role, acaoReq)
		return erro(http.StatusForbidden, "sem permissao para esta acao")
	}
	var body doc
	if m == "POST" || m == "PUT" {
		if err := json.Unmarshal([]byte(rawBody), &body); err != nil || body == nil {
			return erro(http.StatusBadRequest, "json invalido")
		}
	}
	lancamentos := novoServicoLancamentos(novoRepoLancamentos()) // a ligacao fica aqui ate o main montar as pecas (item 32)
	switch acaoReq {
	case acaoLerTx:
		return getTxs(ctx, sid, qs)
	case acaoCriarTx:
		return postTx(ctx, lancamentos, sid, u, body)
	case acaoImportarTx:
		return postTxBatch(ctx, lancamentos, sid, u, body)
	case acaoEditarTx:
		return putTx(ctx, lancamentos, sid, p[3], qs["de"], body, u.Sub, dec == soDono)
	case acaoExcluirTx:
		return delTx(ctx, lancamentos, sid, p[3], qs["de"])
	case acaoLerCfg:
		return getCfg(ctx, spacePK(sid), "CFG#"+strings.ToUpper(p[3]))
	case acaoEscreverCfg:
		return putCfg(ctx, spacePK(sid), "CFG#"+strings.ToUpper(p[3]), body)
	case acaoAdicionarTags:
		return postTags(ctx, sid, body)
	case acaoLerSeq:
		return getSeq(ctx, sid)
	case acaoLerFech:
		return getFechamentos(ctx, sid)
	case acaoEscreverFech:
		return putFechamentos(ctx, sid, body)
	case acaoConferirMes:
		return conferirMes(ctx, sid, u.Sub, body)
	case acaoReabrirMes:
		return reabrirMes(ctx, sid, body)
	case acaoLerAparencia:
		return getCfg(ctx, spacePK(sid), "USER#"+u.Sub+"#CFG#APARENCIA")
	case acaoEditarPerfil:
		return putPerfil(ctx, sid, u, body)
	case acaoTestarAviso:
		return postAvisoTeste(ctx, sid, u)
	case acaoEscreverAparenc:
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
		espacos = append(espacos, doc{"id": id, "nome": nome, "role": d["role"], "apelido": d["apelido"], "telegramChatId": d["telegramChatId"]})
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
	// zero e negativo sao aceitos de proposito (lancamentos so de nota, estornos)
	if _, ok := d["valor"].(float64); !ok {
		return errors.New("valor deve ser numerico")
	}
	if t := d["tipo"]; t != "entrada" && t != "saida" {
		return errors.New("tipo deve ser entrada ou saida")
	}
	for _, c := range []string{"status", "banco"} {
		if s, _ := d[c].(string); s == "" {
			return errors.New(c + " obrigatorio")
		}
	}
	if err := validarAviso(d); err != nil {
		return err
	}
	return validarSerie(d)
}

func protegidos(d doc) {
	for _, k := range []string{"PK", "SK", "id", "criadoEm", "criadoPor"} {
		delete(d, k)
	}
}

var reID = regexp.MustCompile(`^[a-z0-9]{6,40}$`)

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
var (
	errConflito = errors.New("alterado por outra pessoa")
	errDados    = errors.New("dados invalidos")
)

// salvarCfg grava com controle otimista: so grava se a versao no banco ainda e a que o cliente leu.
func salvarCfg(ctx context.Context, pk, sk string, body doc, esperada float64) (doc, error) {
	body["PK"], body["SK"], body["version"] = pk, sk, esperada+1
	item, err := marshal(body)
	if err != nil {
		return nil, errDados
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
			return nil, errConflito
		}
		return nil, err
	}
	return publico(body), nil
}

func putCfg(ctx context.Context, pk, sk string, body doc) (resp, error) {
	esperada, _ := body["version"].(float64)
	delete(body, "version")
	protegidos(body)
	res, err := salvarCfg(ctx, pk, sk, body, esperada)
	switch {
	case errors.Is(err, errDados):
		return erro(400, "dados invalidos")
	case errors.Is(err, errConflito):
		atual, _ := getItem(ctx, pk, sk)
		return out(http.StatusConflict, doc{"erro": "alterado por outra pessoa", "atual": publico(atual)})
	case err != nil:
		log.Println("put cfg:", err)
		return erro(500, "erro ao salvar")
	}
	return out(200, res)
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

// entrada separa os dois tipos de evento que a Lambda recebe: a chamada HTTP da Function URL e o agendamento diario
// do EventBridge ({"acao":"avisos"}, enviado como entrada fixa da regra).
func entrada(ctx context.Context, raw json.RawMessage) (any, error) {
	var ev struct {
		Acao string `json:"acao"`
	}
	if json.Unmarshal(raw, &ev) == nil && ev.Acao == "avisos" {
		n, err := rodarAvisos(ctx, time.Now())
		return doc{"enviados": n}, err
	}
	var r req
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return handler(ctx, r)
}

func main() {
	if err := initDB(context.Background()); err != nil {
		log.Fatal(err)
	}
	lambda.Start(entrada)
}
