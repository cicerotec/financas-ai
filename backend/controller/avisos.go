package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Avisos liga o teste do aviso e a rotina diaria ao service.
type Avisos struct{ svc *service.Avisos }

func NewAvisos(svc *service.Avisos) *Avisos { return &Avisos{svc: svc} }

// PostAvisoTeste: POST /spaces/{sid}/avisos/teste. Manda uma mensagem de teste para o Telegram de quem fez a chamada.
func (c *Avisos) PostAvisoTeste(ctx context.Context, sid, sub string) (Resp, error) {
	if err := c.svc.Testar(ctx, sid, sub); err != nil {
		return respostaErro(err)
	}
	return Out(200, domain.Doc{"ok": true})
}

// Rodar e o evento diario do agendamento ({"acao":"avisos"}). Devolve {"enviados": n}.
func (c *Avisos) Rodar(ctx context.Context) (domain.Doc, error) {
	n, err := c.svc.Rodar(ctx)
	return domain.Doc{"enviados": n}, err
}
