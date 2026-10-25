package main

import (
	"context"
	"net/http"
	"net/url"

	"financas/controller"
	"financas/domain"
)

// rotasCartao liga as rotas /spaces/{sid}/cartoes... ao controller, pelo metodo e pelo tamanho do caminho (a autorizacao por papel ja foi feita).
// p e o caminho inteiro: spaces / sid / cartoes / banco / faturas / ym / acao.
type rotasCartao struct{ c *controller.Cartoes }

func parte(p []string, i int) string {
	if i >= len(p) {
		return ""
	}
	s, err := url.PathUnescape(p[i])
	if err != nil {
		return p[i]
	}
	return s
}

func (r *rotasCartao) lerCartao(ctx context.Context, sid, sub string, dono bool, p []string) (resp, error) {
	if len(p) == 3 {
		return r.c.GetCartoes(ctx, sid)
	}
	return r.c.GetFatura(ctx, sid, sub, dono, parte(p, 3), parte(p, 5))
}

func (r *rotasCartao) escreverCartao(ctx context.Context, sid, sub, m string, p []string, body domain.Doc) (resp, error) {
	banco, ym := parte(p, 3), parte(p, 5)
	switch {
	case len(p) == 3 && m == "POST":
		return r.c.PostCartao(ctx, sid, body)
	case len(p) == 4 && m == "DELETE":
		return r.c.DelCartao(ctx, sid, banco)
	case len(p) == 4 && m == "PUT":
		return r.c.PutCartao(ctx, sid, banco, body)
	case len(p) == 6 && m == "PUT":
		return r.c.PutFatura(ctx, sid, banco, ym, body)
	case len(p) == 7 && p[6] == "conferir" && m == "POST":
		return r.c.PostConferirFatura(ctx, sid, sub, banco, ym, body)
	case len(p) == 7 && p[6] == "conferir" && m == "DELETE":
		return r.c.DelConferirFatura(ctx, sid, banco, ym)
	case len(p) == 7 && p[6] == "diferenca" && m == "POST":
		return r.c.PostDiferenca(ctx, sid, sub, banco, ym)
	}
	return erro(http.StatusNotFound, "rota inexistente")
}
