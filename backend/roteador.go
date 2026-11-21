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
	m := r.RequestContext.HTTP.Method
	p := strings.Split(strings.Trim(r.RawPath, "/"), "/")
	// A unica excecao ao "sem token e 401" (spec mfa-opcional, R27): a rota publica e reconhecida ANTES do token, por metodo e caminho
	// exatos (auth.RotaPublica). O cabecalho authorization nem e lido: um token invalido nela nao muda a resposta. VerificarToken nao muda.
	if a, ok := auth.RotaPublica(m, p); ok {
		*rt = rotular(m, p) // o rotulo e gravado antes de entregar ao controller: o log nao diz "sem-token" numa chamada atendida
		return s.publica(ctx, a, r)
	}
	u, err := auth.VerificarToken(strings.TrimPrefix(r.Headers["authorization"], "Bearer "))
	if err != nil {
		return erro(http.StatusUnauthorized, "nao autenticado")
	}
	*rt = rotular(m, p)

	return s.processar(ctx, u, m, p, r.QueryStringParameters, r.Body)
}

// publica entrega ao controller uma rota que vale sem token. A origem vem so de RequestContext.HTTP.SourceIP, nunca de cabecalho.
func (s *Servidor) publica(ctx context.Context, a auth.Acao, r req) (resp, error) {
	switch a {
	case auth.AcaoRecuperarMFA:
		return s.recup.Post(ctx, r.RequestContext.HTTP.SourceIP, r.Body)
	case auth.AcaoWebhookTelegram: // so o cabecalho do segredo e lido; o authorization nem entra aqui
		return s.telegram.PostWebhook(ctx, r.Headers, r.Body)
	}
	return erro(http.StatusNotFound, "rota inexistente")
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
	if a, ok := auth.RotaPublica(m, p); ok { // a rota publica tem o proprio rotulo: acao=mfa.recuperar rota=mfa
		return rotulo{string(a), p[0]}
	}
	if len(p) == 1 && p[0] == "me" && m == "GET" {
		return rotulo{acaoMe, "me"}
	}
	if a, ok := auth.RotaSemEspaco(m, p); ok {
		return rotulo{string(a), p[0]}
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
	if a, ok := auth.RotaSemEspaco(m, p); ok { // sem espaco no caminho: so o sub do token, nunca o do corpo
		if a == auth.AcaoLerMFA { // leitura: sem corpo; so o sub do token
			return s.estadoMFA.Get(ctx, u.Sub)
		}
		if a == auth.AcaoGerarMFA { // gerar aceita corpo vazio: nao le nada do corpo
			return s.codigos.PostGerar(ctx, u.Sub)
		}
		var body domain.Doc
		if err := json.Unmarshal([]byte(rawBody), &body); err != nil || body == nil { // as demais exigem JSON
			return erro(http.StatusBadRequest, "json invalido")
		}
		switch a {
		case auth.AcaoSairPush:
			return s.aparelhos.PostPushSair(ctx, u.Sub, body)
		case auth.AcaoConfirmarMFA:
			return s.codigos.PostConfirmar(ctx, u.Sub, body)
		}
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
	} else if (m == "POST" || m == "PUT") && acaoReq != auth.AcaoRestaurar && acaoReq != auth.AcaoVincularTelegram && !(acaoReq == auth.AcaoEditarConsulta && m == "POST") { // restaurar (versao e consulta) e o link do Telegram nao tem corpo
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
		case "recorrencias":
			return s.rec.Get(ctx, sid)
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
		if p[2] == "recorrencias" {
			if m == "DELETE" {
				return s.rec.Delete(ctx, sid, u.Sub, p[3], dec == auth.SoDono)
			}
			return s.rec.Post(ctx, sid, u.Sub, qs["de"], body, dec == auth.SoDono)
		}
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
	case auth.AcaoVincularTelegram: // o sub e o do token; nada do corpo
		return s.telegram.PostVinculo(ctx, sid, u.Sub)
	case auth.AcaoRegistrarPush:
		return s.aparelhos.PutPush(ctx, sid, u.Sub, body)
	case auth.AcaoRemoverPush:
		return s.aparelhos.PostPushRemover(ctx, sid, u.Sub, body)
	case auth.AcaoTestarAviso:
		return s.avisos.PostAvisoTeste(ctx, sid, u.Sub, body)
	case auth.AcaoEscreverAparenc:
		return s.cfg.PutAparencia(ctx, sid, u.Sub, body)
	case auth.AcaoEscreverMeio:
		if m == "DELETE" {
			return s.meios.Delete(ctx, sid, parte(p, 3))
		}
		return s.meios.Put(ctx, sid, parte(p, 3), body)
	case auth.AcaoRotinaCombos:
		return s.combos.Put(ctx, sid, body)
	case auth.AcaoLerConsulta:
		return s.consultas.Get(ctx, sid, u.Sub, role == auth.PapelOwner)
	case auth.AcaoCriarConsulta:
		return s.consultas.Post(ctx, sid, u.Sub, role == auth.PapelOwner, body)
	case auth.AcaoEditarConsulta:
		switch m {
		case "PUT":
			return s.consultas.Put(ctx, sid, p[3], u.Sub, dec == auth.SoDono, body)
		case "DELETE":
			return s.consultas.Delete(ctx, sid, p[3], u.Sub, dec == auth.SoDono)
		}
		return s.consultas.Restaurar(ctx, sid, p[3], u.Sub, dec == auth.SoDono)
	}
	return erro(http.StatusNotFound, "rota inexistente")
}

// entrada separa os dois tipos de evento que a Lambda recebe: a chamada HTTP da Function URL e o agendamento diario
// do EventBridge ({"acao":"avisos"} ou {"acao":"combos"}, enviado como entrada fixa da regra).
func (s *Servidor) entrada(ctx context.Context, raw json.RawMessage) (any, error) {
	var ev struct {
		Acao string `json:"acao"`
	}
	if json.Unmarshal(raw, &ev) == nil && ev.Acao == "avisos" {
		return s.avisos.Rodar(ctx)
	}
	if ev.Acao == "combos" { // semanal (item 57): calcula os combos sugeridos dos espacos com a rotina ligada
		return s.combos.Rodar(ctx)
	}
	var r req
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return s.handler(ctx, r)
}
