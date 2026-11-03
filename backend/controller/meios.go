package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Meios liga as rotas de meio de pagamento (item 43) ao service.
type Meios struct{ svc *service.Meios }

func NewMeios(svc *service.Meios) *Meios { return &Meios{svc: svc} }

// Put: PUT /spaces/{sid}/meios/{meio} (so o dono). {"ativoAte": "AAAA-MM-DD"} desativa a partir da data; {"ativoAte": null} reativa.
func (c *Meios) Put(ctx context.Context, sid, meio string, body domain.Doc) (Resp, error) {
	d, err := c.svc.Definir(ctx, sid, meio, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// Delete: DELETE /spaces/{sid}/meios/{meio} (so o dono). Remove o meio que nunca foi usado.
func (c *Meios) Delete(ctx context.Context, sid, meio string) (Resp, error) {
	if err := c.svc.Remover(ctx, sid, meio); err != nil {
		return respostaErro(err)
	}
	return Resp{StatusCode: 204}, nil
}
