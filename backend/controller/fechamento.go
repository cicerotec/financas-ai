package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Fechamento liga as rotas do fechamento mensal e dos saldos ao service.
type Fechamento struct {
	svc    *service.Fechamento
	saldos *service.Saldos
	conf   *service.Conferencia
}

func NewFechamento(svc *service.Fechamento, saldos *service.Saldos, conf *service.Conferencia) *Fechamento {
	return &Fechamento{svc: svc, saldos: saldos, conf: conf}
}

// GetSaldos: GET /spaces/{sid}/saldos. O saldo de cada meio, os ultimos meses e o total, calculados pelo servidor.
func (c *Fechamento) GetSaldos(ctx context.Context, sid string) (Resp, error) {
	d, err := c.saldos.Listar(ctx, sid)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// GetFechamentos: GET /spaces/{sid}/fechamentos
func (c *Fechamento) GetFechamentos(ctx context.Context, sid string) (Resp, error) {
	itens, mesAtual, err := c.svc.Listar(ctx, sid)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, domain.Doc{"itens": itens, "mesAtual": mesAtual})
}

// PutFechamentos: PUT /spaces/{sid}/fechamentos (o cache do cliente e ignorado; so o servidor grava o cache)
func (c *Fechamento) PutFechamentos(ctx context.Context, sid string, body domain.Doc) (Resp, error) {
	d, err := c.svc.SalvarCaches(ctx, sid, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PostConferir: POST /spaces/{sid}/fechamentos/conferir (so o dono). O servidor compara o extrato com o saldo que calculou.
func (c *Fechamento) PostConferir(ctx context.Context, sid, sub string, body domain.Doc) (Resp, error) {
	d, err := c.conf.Conferir(ctx, sid, sub, body)
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
