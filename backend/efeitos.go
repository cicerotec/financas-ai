package main

import (
	"context"

	"financas/domain"
	"financas/service"
)

// efeitosRaiz liga o service de lancamentos aos efeitos que ainda moram na raiz (saldos, tags, avisos, contador).
// Cada um vira um adaptador proprio quando a sua fatia migrar de pasta (item 32 do BACKLOG).
type efeitosRaiz struct{}

type travasRaiz struct{ t *travas }

func (a travasRaiz) Checar(al domain.Alvo, confirmado bool) (bool, error) {
	return a.t.checar(al, confirmado, agora())
}

func (efeitosRaiz) Travas(ctx context.Context, sid string) service.Travas {
	return travasRaiz{novasTravas(ctx, sid)}
}
func (efeitosRaiz) InvalidarSaldos(ctx context.Context, sid string, alvos []domain.Alvo) {
	invalidarSaldos(ctx, sid, alvos)
}
func (efeitosRaiz) GarantirTags(ctx context.Context, sid string, tags []string) {
	garantirTagsServidor(ctx, sid, tags)
}
func (efeitosRaiz) SincronizarAviso(ctx context.Context, sid string, l domain.Lancamento) {
	sincronizarAviso(ctx, sid, l)
}
func (efeitosRaiz) RemoverAviso(ctx context.Context, sid, id string) { removerAviso(ctx, sid, id) }
func (efeitosRaiz) ComSeq(ctx context.Context, sid string, d domain.Doc) domain.Doc {
	return comSeq(ctx, sid, d)
}
