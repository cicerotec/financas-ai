package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Combos liga a rotina semanal dos combos sugeridos (item 57) ao service.
type Combos struct{ svc *service.Combos }

func NewCombos(svc *service.Combos) *Combos { return &Combos{svc: svc} }

// Put: PUT /spaces/{sid}/combos/rotina (so o dono). {"ativa": true|false} liga ou desliga a rotina. Devolve {"rotinaAtiva": bool}.
func (c *Combos) Put(ctx context.Context, sid string, body domain.Doc) (Resp, error) {
	d, err := c.svc.Definir(ctx, sid, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// Rodar e o evento semanal do agendamento ({"acao":"combos"}). Devolve {"espacos": n, "sugeridos": n}.
func (c *Combos) Rodar(ctx context.Context) (domain.Doc, error) {
	return c.svc.Rodar(ctx)
}
