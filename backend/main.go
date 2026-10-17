package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"financas/auth"
	"financas/controller"
	"financas/relogio"
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
		return s.cfg.GetCfg(ctx, sid, p[3])
	case auth.AcaoEscreverCfg:
		return s.cfg.PutCfg(ctx, sid, p[3], body)
	case auth.AcaoAdicionarTags:
		return s.cfg.PostTags(ctx, sid, body)
	case auth.AcaoLerSeq:
		return s.cfg.GetSeq(ctx, sid)
	case auth.AcaoLerFech:
		return s.fech.GetFechamentos(ctx, sid)
	case auth.AcaoEscreverFech:
		return s.fech.PutFechamentos(ctx, sid, body)
	case auth.AcaoConferirMes:
		return s.fech.PostConferir(ctx, sid, u.Sub, body)
	case auth.AcaoReabrirMes:
		return s.fech.PostReabrir(ctx, sid, body)
	case auth.AcaoLerAparencia:
		return s.cfg.GetAparencia(ctx, sid, u.Sub)
	case auth.AcaoEditarPerfil:
		return s.perfil.PutPerfil(ctx, sid, u.Sub, body)
	case auth.AcaoTestarAviso:
		return postAvisoTeste(ctx, sid, u)
	case auth.AcaoEscreverAparenc:
		return s.cfg.PutAparencia(ctx, sid, u.Sub, body)
	}
	return erro(http.StatusNotFound, "rota inexistente")
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
	s := novoServidor(db, table, relogio.Real{})
	if porta := os.Getenv("LOCAL_PORT"); porta != "" { // so no computador do desenvolvedor (local.go)
		rodarLocal(porta, s)
		return
	}
	lambda.Start(s.entrada)
}
