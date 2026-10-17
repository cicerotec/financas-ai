package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Fechamento liga as rotas do fechamento mensal por banco ao service.
type Fechamento struct{ svc *service.Fechamento }

func NewFechamento(svc *service.Fechamento) *Fechamento { return &Fechamento{svc: svc} }

// GetFechamentos: GET /spaces/{sid}/fechamentos
func (c *Fechamento) GetFechamentos(ctx context.Context, sid string) (Resp, error) {
	itens, mesAtual, err := c.svc.Listar(ctx, sid)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, domain.Doc{"itens": itens, "mesAtual": mesAtual})
}

// PutFechamentos: PUT /spaces/{sid}/fechamentos (cache dos meses encerrados)
func (c *Fechamento) PutFechamentos(ctx context.Context, sid string, body domain.Doc) (Resp, error) {
	d, err := c.svc.SalvarCaches(ctx, sid, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PostConferir: POST /spaces/{sid}/fechamentos/conferir (so o dono)
func (c *Fechamento) PostConferir(ctx context.Context, sid, sub string, body domain.Doc) (Resp, error) {
	d, err := c.svc.Conferir(ctx, sid, sub, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PostReabrir: POST /spaces/{sid}/fechamentos/reabrir (so o dono)
func (c *Fechamento) PostReabrir(ctx context.Context, sid string, body domain.Doc) (Resp, error) {
	d, err := c.svc.Reabrir(ctx, sid, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}
