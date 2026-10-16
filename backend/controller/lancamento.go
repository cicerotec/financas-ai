package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Lancamentos liga as rotas de lancamento ao service.
type Lancamentos struct{ svc *service.Lancamentos }

func NewLancamentos(svc *service.Lancamentos) *Lancamentos { return &Lancamentos{svc: svc} }

// PostTx: POST /spaces/{sid}/tx
func (c *Lancamentos) PostTx(ctx context.Context, sid, sub string, body domain.Doc) (Resp, error) {
	d, err := c.svc.Criar(ctx, sid, sub, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(201, d)
}

// PostTxBatch: POST /spaces/{sid}/tx/batch
func (c *Lancamentos) PostTxBatch(ctx context.Context, sid, sub string, body domain.Doc) (Resp, error) {
	d, err := c.svc.Importar(ctx, sid, sub, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PutTx: PUT /spaces/{sid}/tx/{id}?de=<dataEvento atual>. restrito: so quem criou pode editar (member).
func (c *Lancamentos) PutTx(ctx context.Context, sid, id, de string, body domain.Doc, sub string, restrito bool) (Resp, error) {
	d, err := c.svc.Editar(ctx, sid, id, de, body, sub, restrito)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// DelTx: DELETE /spaces/{sid}/tx/{id}?de=<dataEvento atual>. A exclusao e logica.
func (c *Lancamentos) DelTx(ctx context.Context, sid, id, de, sub string) (Resp, error) {
	d, err := c.svc.Excluir(ctx, sid, id, de, sub)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}
