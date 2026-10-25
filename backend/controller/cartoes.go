package controller

import (
	"context"
	"net/http"

	"financas/domain"
	"financas/service"
)

// Cartoes liga as rotas das faturas do cartao ao service (item 33a). Nao tem regra: so traduz rota em chamada.
type Cartoes struct{ svc *service.Cartoes }

func NewCartoes(svc *service.Cartoes) *Cartoes { return &Cartoes{svc: svc} }

func resposta(d domain.Doc, err error) (Resp, error) {
	if err != nil {
		return respostaErro(err)
	}
	return Out(http.StatusOK, d)
}

// GetCartoes: GET /spaces/{sid}/cartoes
func (c *Cartoes) GetCartoes(ctx context.Context, sid string) (Resp, error) {
	return resposta(c.svc.Listar(ctx, sid))
}

// GetFatura: GET /spaces/{sid}/cartoes/{banco}/faturas[/{ym}]
func (c *Cartoes) GetFatura(ctx context.Context, sid, sub string, dono bool, banco, ym string) (Resp, error) {
	return resposta(c.svc.Fatura(ctx, sid, sub, dono, banco, ym))
}

// PostCartao: POST /spaces/{sid}/cartoes (so o dono)
func (c *Cartoes) PostCartao(ctx context.Context, sid string, body domain.Doc) (Resp, error) {
	return resposta(c.svc.Adicionar(ctx, sid, body))
}

// DelCartao: DELETE /spaces/{sid}/cartoes/{banco} (so o dono)
func (c *Cartoes) DelCartao(ctx context.Context, sid, banco string) (Resp, error) {
	return resposta(c.svc.Remover(ctx, sid, banco))
}

// PutCartao: PUT /spaces/{sid}/cartoes/{banco} (so o dono)
func (c *Cartoes) PutCartao(ctx context.Context, sid, banco string, body domain.Doc) (Resp, error) {
	return resposta(c.svc.Inicio(ctx, sid, banco, body))
}

// PutFatura: PUT /spaces/{sid}/cartoes/{banco}/faturas/{ym} (so o dono)
func (c *Cartoes) PutFatura(ctx context.Context, sid, banco, ym string, body domain.Doc) (Resp, error) {
	return resposta(c.svc.Fechamento(ctx, sid, banco, ym, body))
}

// PostConferirFatura: POST /spaces/{sid}/cartoes/{banco}/faturas/{ym}/conferir (so o dono)
func (c *Cartoes) PostConferirFatura(ctx context.Context, sid, sub, banco, ym string, body domain.Doc) (Resp, error) {
	return resposta(c.svc.Conferir(ctx, sid, sub, banco, ym, body))
}

// DelConferirFatura: DELETE /spaces/{sid}/cartoes/{banco}/faturas/{ym}/conferir (so o dono)
func (c *Cartoes) DelConferirFatura(ctx context.Context, sid, banco, ym string) (Resp, error) {
	return resposta(c.svc.Desconferir(ctx, sid, banco, ym))
}

// PostDiferenca: POST /spaces/{sid}/cartoes/{banco}/faturas/{ym}/diferenca (so o dono)
func (c *Cartoes) PostDiferenca(ctx context.Context, sid, sub, banco, ym string) (Resp, error) {
	return resposta(c.svc.RegistrarDiferenca(ctx, sid, sub, banco, ym))
}
