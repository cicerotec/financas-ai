package service

import (
	"context"

	"financas/domain"
)

// EfeitosLancamento junta os servicos que reagem a uma escrita de lancamento (saldos, tags, avisos e contador) e implementa
// a interface Efeitos que o service de lancamentos declara.
type EfeitosLancamento struct {
	fechamento *Fechamento
	tags       *Tags
	avisos     *Avisos
	contador   *Contador
}

func NewEfeitos(fechamento *Fechamento, tags *Tags, avisos *Avisos, contador *Contador) *EfeitosLancamento {
	return &EfeitosLancamento{fechamento: fechamento, tags: tags, avisos: avisos, contador: contador}
}

func (e *EfeitosLancamento) Travas(ctx context.Context, sid string) Travas {
	return e.fechamento.Travas(ctx, sid)
}
func (e *EfeitosLancamento) InvalidarSaldos(ctx context.Context, sid string, alvos []domain.Alvo) {
	e.fechamento.InvalidarSaldos(ctx, sid, alvos)
}
func (e *EfeitosLancamento) GarantirTags(ctx context.Context, sid string, tags []string) {
	e.tags.Garantir(ctx, sid, tags)
}
func (e *EfeitosLancamento) SincronizarAviso(ctx context.Context, sid string, l domain.Lancamento) {
	e.avisos.Sincronizar(ctx, sid, l)
}
func (e *EfeitosLancamento) RemoverAviso(ctx context.Context, sid, id string) { e.avisos.Remover(ctx, sid, id) }
func (e *EfeitosLancamento) ComSeq(ctx context.Context, sid string, d domain.Doc) domain.Doc {
	return e.contador.ComSeq(ctx, sid, d)
}
