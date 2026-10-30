package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Recorrencias liga as rotas da recorrencia automatica (item 51) ao service.
type Recorrencias struct{ svc *service.Recorrencias }

func NewRecorrencias(svc *service.Recorrencias) *Recorrencias { return &Recorrencias{svc: svc} }

// Post: POST /spaces/{sid}/recorrencias?de=<dataEvento atual> { id, intervalo, aviso? }. Marca o lancamento, cria a primeira copia e o item.
// restrito: so quem criou o lancamento pode marcar (member).
func (c *Recorrencias) Post(ctx context.Context, sid, sub, de string, body domain.Doc, restrito bool) (Resp, error) {
	id, _ := body["id"].(string)
	f, _ := body["intervalo"].(float64)
	var aviso domain.Doc
	if bruto, tem := body["aviso"]; tem && bruto == nil {
		aviso = domain.Doc{} // aviso: null = a primeira copia sai sem aviso, mesmo que o original tenha
	} else if tem { // aviso opcional da primeira copia, validado como o de qualquer lancamento
		tmp := domain.Doc{"aviso": bruto}
		if err := domain.ValidarAviso(tmp); err != nil {
			return Erro(400, err.Error())
		}
		aviso, _ = tmp["aviso"].(domain.Doc)
	}
	d, err := c.svc.Marcar(ctx, sid, sub, id, de, int(f), aviso, restrito)
	if err != nil {
		return respostaErro(err)
	}
	return Out(201, d)
}

// Delete: DELETE /spaces/{sid}/recorrencias/{id}. Desliga a recorrencia (as copias futuras ja criadas ficam).
func (c *Recorrencias) Delete(ctx context.Context, sid, sub, id string, restrito bool) (Resp, error) {
	if err := c.svc.Desligar(ctx, sid, sub, id, restrito); err != nil {
		return respostaErro(err)
	}
	return Out(200, domain.Doc{"ok": true})
}

// Get: GET /spaces/{sid}/recorrencias. As recorrencias ativas do espaco.
func (c *Recorrencias) Get(ctx context.Context, sid string) (Resp, error) {
	l, err := c.svc.Listar(ctx, sid)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, l)
}
