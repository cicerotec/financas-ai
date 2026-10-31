package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Consultas liga as rotas das consultas salvas da Tendencias (item 53) ao service.
type Consultas struct{ svc *service.Consultas }

func NewConsultas(svc *service.Consultas) *Consultas { return &Consultas{svc: svc} }

// Get: GET /spaces/{sid}/consultas. As consultas vivas do espaco, com autor, minha e podeAlterar ja decididos pelo servidor.
func (c *Consultas) Get(ctx context.Context, sid, sub string, owner bool) (Resp, error) {
	l, err := c.svc.Listar(ctx, sid, sub, owner)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, l)
}

// Post: POST /spaces/{sid}/consultas { nome, modo, grupos | expressao, visao }.
func (c *Consultas) Post(ctx context.Context, sid, sub string, owner bool, body domain.Doc) (Resp, error) {
	d, err := c.svc.Criar(ctx, sid, sub, owner, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(201, d)
}

// Put: PUT /spaces/{sid}/consultas/{id} { version, nome?, (modo, grupos | expressao, visao)? }. restrito: member, so a propria.
func (c *Consultas) Put(ctx context.Context, sid, id, sub string, restrito bool, body domain.Doc) (Resp, error) {
	d, err := c.svc.Atualizar(ctx, sid, id, sub, restrito, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// Delete: DELETE /spaces/{sid}/consultas/{id}. Exclusao logica; o Desfazer e o Restaurar.
func (c *Consultas) Delete(ctx context.Context, sid, id, sub string, restrito bool) (Resp, error) {
	if err := c.svc.Excluir(ctx, sid, id, sub, restrito); err != nil {
		return respostaErro(err)
	}
	return Out(200, domain.Doc{"id": id})
}

// Restaurar: POST /spaces/{sid}/consultas/{id}/restaurar (sem corpo). Devolve a consulta de volta, com o mesmo autor.
func (c *Consultas) Restaurar(ctx context.Context, sid, id, sub string, restrito bool) (Resp, error) {
	d, err := c.svc.Restaurar(ctx, sid, id, sub, restrito)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}
