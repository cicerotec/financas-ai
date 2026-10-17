package main

import (
	"context"

	"financas/domain"
	"financas/service"
)

// efeitosRaiz liga o service de lancamentos aos efeitos: tags, contador e saldos ja sao services; os avisos ainda moram na raiz.
// Cada um vira um adaptador proprio quando a sua fatia migrar de pasta (item 32 do BACKLOG).
type efeitosRaiz struct {
	tags     *service.Tags
	contador *service.Contador
	fech     *service.Fechamento
}

func (e efeitosRaiz) Travas(ctx context.Context, sid string) service.Travas { return e.fech.Travas(ctx, sid) }
func (e efeitosRaiz) InvalidarSaldos(ctx context.Context, sid string, alvos []domain.Alvo) {
	e.fech.InvalidarSaldos(ctx, sid, alvos)
}
func (e efeitosRaiz) GarantirTags(ctx context.Context, sid string, tags []string) {
	e.tags.Garantir(ctx, sid, tags)
}
func (efeitosRaiz) SincronizarAviso(ctx context.Context, sid string, l domain.Lancamento) {
	sincronizarAviso(ctx, sid, l)
}
func (efeitosRaiz) RemoverAviso(ctx context.Context, sid, id string) { removerAviso(ctx, sid, id) }
func (e efeitosRaiz) ComSeq(ctx context.Context, sid string, d domain.Doc) domain.Doc {
	return e.contador.ComSeq(ctx, sid, d)
}
