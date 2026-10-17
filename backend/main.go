package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/auth"
	"financas/controller"
)

type resp = events.LambdaFunctionURLResponse
type req = events.LambdaFunctionURLRequest

func out(status int, v any) (resp, error)        { return controller.Out(status, v) }
func erro(status int, msg string) (resp, error) { return controller.Erro(status, msg) }

func (s *Servidor) handler(ctx context.Context, r req) (resp, error) {
	u, err := auth.VerificarToken(strings.TrimPrefix(r.Headers["authorization"], "Bearer "))
	if err != nil {
		return erro(http.StatusUnauthorized, "nao autenticado")
	}
	m := r.RequestContext.HTTP.Method
	p := strings.Split(strings.Trim(r.RawPath, "/"), "/")

	return s.processar(ctx, u, m, p, r.QueryStringParameters, r.Body)
}

// processar roda uma requisicao ja autenticada: classifica a rota, confere o papel e executa.
func (s *Servidor) processar(ctx context.Context, u *user, m string, p []string, qs map[string]string, rawBody string) (resp, error) {
	if len(p) == 1 && p[0] == "me" && m == "GET" {
		return s.esp.GetMe(ctx, u.Sub, u.Email)
	}
	if len(p) < 3 || p[0] != "spaces" {
		return erro(http.StatusNotFound, "rota inexistente")
	}
	sid := p[1]
	acaoReq, ok := auth.RotaParaAcao(m, p[2:])
	if !ok {
		return erro(http.StatusNotFound, "rota inexistente")
	}
	role, err := s.esp.Papel(ctx, sid, u.Sub)
	if err != nil {
		return erro(http.StatusInternalServerError, "erro interno")
	}
	if role == "" {
		return erro(http.StatusForbidden, "sem acesso a esse espaco")
	}
	dec := auth.Permitido(role, acaoReq)
	if dec == auth.Negado {
		log.Printf("negado: sub=%s papel=%s acao=%s", u.Sub, role, acaoReq)
		return erro(http.StatusForbidden, "sem permissao para esta acao")
	}
	var body doc
	if (m == "POST" || m == "PUT") && acaoReq != auth.AcaoRestaurar { // restaurar nao tem corpo
		if err := json.Unmarshal([]byte(rawBody), &body); err != nil || body == nil {
			return erro(http.StatusBadRequest, "json invalido")
		}
	}
	confirma := qs["confirmaMesFechado"] == "1" // a pessoa confirmou mexer em mes fechado por data (spec do item 38)
	switch acaoReq {
	case auth.AcaoLerTx:
		return s.lanc.GetTxs(ctx, sid, qs)
	case auth.AcaoCriarTx:
		return s.lanc.PostTx(ctx, sid, u.Sub, body, confirma)
	case auth.AcaoImportarTx:
		return s.lanc.PostTxBatch(ctx, sid, u.Sub, body, confirma)
	case auth.AcaoEditarTx:
		return s.lanc.PutTx(ctx, sid, p[3], qs["de"], body, u.Sub, dec == auth.SoDono, confirma)
	case auth.AcaoExcluirTx:
		return s.lanc.DelTx(ctx, sid, p[3], qs["de"], u.Sub, confirma)
	case auth.AcaoLerHistorico:
		if p[2] == "historico" {
			return s.lanc.GetHistorico(ctx, sid, qs["mes"], qs["q"], u.Sub, dec == auth.SoDono)
		}
		return s.lanc.GetVersoes(ctx, sid, p[3], qs["de"], u.Sub, dec == auth.SoDono)
	case auth.AcaoRestaurar:
		return s.lanc.PostRestaurar(ctx, sid, p[3], p[5], u.Sub, dec == auth.SoDono, confirma)
	case auth.AcaoLerCfg:
		return getCfg(ctx, spacePK(sid), "CFG#"+strings.ToUpper(p[3]))
	case auth.AcaoEscreverCfg:
		return putCfg(ctx, spacePK(sid), "CFG#"+strings.ToUpper(p[3]), body)
	case auth.AcaoAdicionarTags:
		return postTags(ctx, sid, body)
	case auth.AcaoLerSeq:
		return getSeq(ctx, sid)
	case auth.AcaoLerFech:
		return getFechamentos(ctx, sid)
	case auth.AcaoEscreverFech:
		return putFechamentos(ctx, sid, body)
	case auth.AcaoConferirMes:
		return conferirMes(ctx, sid, u.Sub, body)
	case auth.AcaoReabrirMes:
		return reabrirMes(ctx, sid, body)
	case auth.AcaoLerAparencia:
		return getCfg(ctx, spacePK(sid), "USER#"+u.Sub+"#CFG#APARENCIA")
	case auth.AcaoEditarPerfil:
		return putPerfil(ctx, sid, u, body)
	case auth.AcaoTestarAviso:
		return postAvisoTeste(ctx, sid, u)
	case auth.AcaoEscreverAparenc:
		return putAparencia(ctx, spacePK(sid), "USER#"+u.Sub+"#CFG#APARENCIA", body)
	}
	return erro(http.StatusNotFound, "rota inexistente")
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
func (s *Servidor) entrada(ctx context.Context, raw json.RawMessage) (any, error) {
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
	return s.handler(ctx, r)
}

func main() {
	if err := initDB(context.Background()); err != nil {
		log.Fatal(err)
	}
	s := novoServidor(db, table)
	if porta := os.Getenv("LOCAL_PORT"); porta != "" { // so no computador do desenvolvedor (local.go)
		rodarLocal(porta, s)
		return
	}
	lambda.Start(s.entrada)
}
