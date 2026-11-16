package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Aparelhos liga as rotas dos aparelhos que recebem aviso pelo app ao service.
type Aparelhos struct{ svc *service.Aparelhos }

func NewAparelhos(svc *service.Aparelhos) *Aparelhos { return &Aparelhos{svc: svc} }

// PutPush: PUT /spaces/{sid}/push com {"token"}. Registra o aparelho de quem fez a chamada neste espaco.
func (c *Aparelhos) PutPush(ctx context.Context, sid, sub string, body domain.Doc) (Resp, error) {
	r, err := c.svc.Registrar(ctx, sid, sub, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, r)
}

// PostPushRemover: POST /spaces/{sid}/push/remover com {"token"}. Tira o aparelho deste espaco.
func (c *Aparelhos) PostPushRemover(ctx context.Context, sid, sub string, body domain.Doc) (Resp, error) {
	r, err := c.svc.Remover(ctx, sid, sub, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, r)
}

// PostPushSair: POST /push/sair com {"token"}. Tira o aparelho de todos os espacos de quem fez a chamada.
func (c *Aparelhos) PostPushSair(ctx context.Context, sub string, body domain.Doc) (Resp, error) {
	r, err := c.svc.Sair(ctx, sub, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, r)
}
