package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Espacos liga GET /me e a consulta de papel ao service.
type Espacos struct{ svc *service.Espacos }

func NewEspacos(svc *service.Espacos) *Espacos { return &Espacos{svc: svc} }

// Papel e o papel da pessoa no espaco; "" = sem acesso.
func (c *Espacos) Papel(ctx context.Context, sid, sub string) (string, error) {
	return c.svc.Papel(ctx, sid, sub)
}

// GetMe: GET /me. A resposta tem as mesmas chaves de sempre (sub, email, espacos[id, nome, role, apelido, telegramChatId]).
func (c *Espacos) GetMe(ctx context.Context, sub, email string) (Resp, error) {
	es, err := c.svc.Meus(ctx, sub)
	if err != nil {
		return respostaErro(err)
	}
	lista := make([]domain.Doc, 0, len(es))
	for _, e := range es {
		lista = append(lista, domain.Doc{"id": e.ID, "nome": e.Nome, "role": e.Role, "apelido": e.Apelido, "telegramChatId": e.TelegramChatID})
	}
	return Out(200, domain.Doc{"sub": sub, "email": email, "espacos": lista})
}
