package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"financas/auth"
	"financas/controller"
	"financas/domain"
)

// O roteador: recebe o evento da Function URL (ou do agendamento), autentica, classifica a rota, confere o papel e chama o
// controller do assunto. Nao tem regra de negocio: so liga a rota ao controller (a montagem do Servidor esta em servidor.go).

type resp = events.LambdaFunctionURLResponse
type req = events.LambdaFunctionURLRequest

func out(status int, v any) (resp, error)        { return controller.Out(status, v) }
func erro(status int, msg string) (resp, error) { return controller.Erro(status, msg) }

// handler atende uma chamada HTTP e deixa uma linha de log ao terminar (so o que foi chamado, nunca o conteudo).
func (s *Servidor) handler(ctx context.Context, r req) (resp, error) {
	t0 := time.Now()
	rt := rotulo{Acao: acaoSemToken, Rota: "-"}
	res, err := s.atender(ctx, r, &rt)
	registrar(r.RequestContext.RequestID, r.RequestContext.HTTP.Method, rt, res, err, time.Since(t0))
	return res, err
}

// atender autentica e roda a requisicao; grava em rt o nome da rota assim que ela e classificada.
func (s *Servidor) atender(ctx context.Context, r req, rt *rotulo) (resp, error) {
	u, err := auth.VerificarToken(strings.TrimPrefix(r.Headers["authorization"], "Bearer "))
	if err != nil {
		return erro(http.StatusUnauthorized, "nao autenticado")
	}
	m := r.RequestContext.HTTP.Method
	p := strings.Split(strings.Trim(r.RawPath, "/"), "/")
	*rt = rotular(m, p)

	return s.processar(ctx, u, m, p, r.QueryStringParameters, r.Body)
}

// Nomes de acao das saidas que nao chegam a classificacao da rota (o log nunca leva o caminho, que tem ids).
const (
	acaoSemToken     = "sem-token"
	acaoMe           = "me"
	acaoDesconhecida = "desconhecida"
)

// rotulo e o que o log diz de uma rota: a acao (a mesma de auth.RotaParaAcao) e a rota sem ids, como "cfg/listas".
type rotulo struct{ Acao, Rota string }

// rotular classifica uma rota para o log, sem ids. A rota e o primeiro trecho depois do espaco (um nome fixo da API) e,
// em /cfg, o nome da configuracao, que auth.RotaParaAcao so aceita de uma lista fixa.
func rotular(m string, p []string) rotulo {
	if len(p) == 1 && p[0] == "me" && m == "GET" {
		return rotulo{acaoMe, "me"}
	}
	if len(p) < 3 || p[0] != "spaces" {
		return rotulo{acaoDesconhecida, "-"}
	}
	a, ok := auth.RotaParaAcao(m, p[2:])
	if !ok {
		return rotulo{acaoDesconhecida, "-"}
	}
	rota := p[2]
	if rota == "cfg" && len(p) > 3 {
		rota += "/" + p[3]
	}
	return rotulo{string(a), rota}
}

// linhaDeLog monta a linha de uma requisicao: campos fixos, sem espaco nos valores e sem nenhum dado da chamada.
func linhaDeLog(rid, m string, rt rotulo, status int, d time.Duration) string {
	semVazio := func(v string) string {
		if v == "" || strings.ContainsAny(v, " \t\r\n") {
			return "-"
		}
		return v
	}
	return fmt.Sprintf("req rid=%s m=%s acao=%s rota=%s status=%d ms=%d",
		semVazio(rid), semVazio(m), semVazio(rt.Acao), semVazio(rt.Rota), status, d.Milliseconds())
}

// registrar escreve a linha da requisicao; erro Go da execucao conta como 500.
func registrar(rid, m string, rt rotulo, res resp, err error, d time.Duration) {
	status := res.StatusCode
	if err != nil {
		status = http.StatusInternalServerError
	}
	log.Print(linhaDeLog(rid, m, rt, status, d))
}

// processar roda uma requisicao ja autenticada: classifica a rota, confere o papel e executa.
func (s *Servidor) processar(ctx context.Context, u *auth.User, m string, p []string, qs map[string]string, rawBody string) (resp, error) {
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
	var body domain.Doc
	if rawBody == "" && acaoReq == auth.AcaoEscreverCartao { // os comandos do cartao sem opcao aceitam corpo vazio
		body = domain.Doc{}
	} else if (m == "POST" || m == "PUT") && acaoReq != auth.AcaoRestaurar { // restaurar nao tem corpo
		if err := json.Unmarshal([]byte(rawBody), &body); err != nil || body == nil {
			return erro(http.StatusBadRequest, "json invalido")
		}
	}
	confirma := qs["confirmaMesFechado"] == "1" // a pessoa confirmou mexer em mes fechado por data (spec do item 38)
	switch acaoReq {
	case auth.AcaoLerTx:
		switch p[2] {
		case "futuros":
			return s.lanc.GetFuturos(ctx, sid, qs)
		case "status":
			return s.lanc.GetStatus(qs)
		}
		return s.lanc.GetTxs(ctx, sid, qs)
	case auth.AcaoCriarTx:
		if p[2] == "transferencias" {
			return s.lanc.PostTransferencia(ctx, sid, u.Sub, body, confirma)
		}
		return s.lanc.PostTx(ctx, sid, u.Sub, body, confirma)
	case auth.AcaoConfirmarTx:
		return s.lanc.PostConfirmar(ctx, sid, p[3], qs["de"], body, u.Sub, dec == auth.SoDono, confirma)
	case auth.AcaoImportarTx:
		return s.lanc.PostTxBatch(ctx, sid, u.Sub, body, confirma)
	case auth.AcaoEditarTx:
		return s.lanc.PutTx(ctx, sid, p[3], qs["de"], body, u.Sub, dec == auth.SoDono, confirma)
	case auth.AcaoExcluirTx:
		return s.lanc.DelTx(ctx, sid, p[3], qs["de"], u.Sub, confirma)
	case auth.AcaoLerHistorico:
		if p[2] == "historico" {
			return s.lanc.GetHistorico(ctx, sid, qs, u.Sub, dec == auth.SoDono)
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
		return s.fech.GetSaldos(ctx, sid)
	case auth.AcaoConferirMes:
		return s.fech.PostConferir(ctx, sid, u.Sub, body)
	case auth.AcaoReabrirMes:
		return s.fech.PostReabrir(ctx, sid, body)
	case auth.AcaoLerCartao:
		return s.cartoes.lerCartao(ctx, sid, u.Sub, role == auth.PapelOwner, p)
	case auth.AcaoEscreverCartao:
		return s.cartoes.escreverCartao(ctx, sid, u.Sub, m, p, body)
	case auth.AcaoLerAparencia:
		return s.cfg.GetAparencia(ctx, sid, u.Sub)
	case auth.AcaoEditarPerfil:
		return s.perfil.PutPerfil(ctx, sid, u.Sub, body)
	case auth.AcaoTestarAviso:
		return s.avisos.PostAvisoTeste(ctx, sid, u.Sub)
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
		return s.avisos.Rodar(ctx)
	}
	var r req
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return s.handler(ctx, r)
}
