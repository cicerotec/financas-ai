package controller

import (
	"context"

	"financas/service"
)

// EstadoMFA liga GET /me/mfa ao service (spec mfa-opcional, R37). So le; o sub e o do token.
type EstadoMFA struct{ svc *service.EstadoMFA }

func NewEstadoMFA(svc *service.EstadoMFA) *EstadoMFA { return &EstadoMFA{svc: svc} }

// Get: GET /me/mfa. 200 {"ligado", "lista": {"ativa", "geradaEm"?}}.
func (c *EstadoMFA) Get(ctx context.Context, sub string) (Resp, error) {
	r, err := c.svc.Ler(ctx, sub)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, r)
}
