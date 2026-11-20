package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// CodigosDeRecuperacao liga as rotas autenticadas da lista de codigos de recuperacao ao service (spec mfa-opcional, R25 e R25a).
// O sub e sempre o do token, nunca o do corpo.
type CodigosDeRecuperacao struct{ svc *service.CodigosDeRecuperacao }

func NewCodigosDeRecuperacao(svc *service.CodigosDeRecuperacao) *CodigosDeRecuperacao {
	return &CodigosDeRecuperacao{svc: svc}
}

// PostGerar: POST /me/mfa/codigos (corpo vazio aceito). 200 {listaId, codigos}.
func (c *CodigosDeRecuperacao) PostGerar(ctx context.Context, sub string) (Resp, error) {
	r, err := c.svc.Gerar(ctx, sub)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, r)
}

// PostConfirmar: POST /me/mfa/codigos/confirmar com {"listaId"}. 200 {ok}, 409 lista_substituida.
func (c *CodigosDeRecuperacao) PostConfirmar(ctx context.Context, sub string, body domain.Doc) (Resp, error) {
	r, err := c.svc.Confirmar(ctx, sub, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, r)
}
