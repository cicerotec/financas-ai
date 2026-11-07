package service

import (
	"context"
	"log"

	"financas/domain"
)

// Combos sugeridos (item 57): uma rotina semanal conta os conjuntos de tags dos 1000 lancamentos mais recentes e guarda o
// resultado em CFG#COMBOS (sugeridos, geradoEm). O dono liga e desliga a rotina em Listas; ligar grava um item auxiliar no
// AVISOS#ATIVOS (tipo "combos", id "combos"), que a rotina le para saber de quais espacos calcular. Nenhuma tela conta combos.

// idItemCombos e o id do item auxiliar da rotina de um espaco (um por espaco).
const idItemCombos = "combos"

// RepositorioCombos e o que a rotina precisa do banco: o item auxiliar no AVISOS#ATIVOS e a gravacao parcial de CFG#COMBOS.
type RepositorioCombos interface {
	Sincronizar(ctx context.Context, sid string, item domain.Doc) error
	Remover(ctx context.Context, sid, id string) error
	BuscarItem(ctx context.Context, sid, id string) (domain.Doc, error)
	Ativos(ctx context.Context) ([]domain.Doc, error)
}

// ConfigDeCombos e o que a rotina precisa de CFG#COMBOS.
type ConfigDeCombos interface {
	Ler(ctx context.Context, sid, nome string) (domain.Doc, error)
	Salvar(ctx context.Context, sid, nome string, body domain.Doc, esperada float64) (domain.Doc, error)
	// AtualizarCombos grava so sugeridos e geradoEm, sem subir a version nem tocar na lista.
	AtualizarCombos(ctx context.Context, sid string, sugeridos []domain.Doc, geradoEm string) error
}

// RecentesDoEspaco le os lancamentos de um espaco, do mais novo para tras (o repository de lancamentos implementa).
type RecentesDoEspaco interface {
	Listar(ctx context.Context, sid string, f domain.FiltroTx) ([]domain.Doc, string, error)
}

type Combos struct {
	avisos RepositorioCombos
	cfg    ConfigDeCombos
	lanc   RecentesDoEspaco
	rel    Relogio
}

func NewCombos(avisos RepositorioCombos, cfg ConfigDeCombos, lanc RecentesDoEspaco, rel Relogio) *Combos {
	return &Combos{avisos: avisos, cfg: cfg, lanc: lanc, rel: rel}
}

// RotinaAtiva diz se a rotina esta ligada no espaco (existe o item auxiliar).
func (s *Combos) RotinaAtiva(ctx context.Context, sid string) (bool, error) {
	it, err := s.avisos.BuscarItem(ctx, sid, idItemCombos)
	if err != nil {
		log.Println("combos: ler item da rotina:", err)
		return false, ErrInterno
	}
	return it != nil, nil
}

// Definir liga ou desliga a rotina do espaco ({"ativa": true|false}). Desligar mantem os sugeridos ja calculados.
func (s *Combos) Definir(ctx context.Context, sid string, body domain.Doc) (domain.Doc, error) {
	ativa, ok := body["ativa"].(bool)
	if !ok {
		return nil, &ErroValidacao{"informe ativa: true ou false"}
	}
	var err error
	if ativa {
		err = s.avisos.Sincronizar(ctx, sid, domain.Doc{"id": idItemCombos, "tipo": domain.TipoItemCombos})
	} else {
		err = s.avisos.Remover(ctx, sid, idItemCombos)
	}
	if err != nil {
		log.Println("combos: gravar item da rotina:", err)
		return nil, ErrSalvar
	}
	return domain.Doc{"rotinaAtiva": ativa}, nil
}

// Rodar e o evento semanal ({"acao":"combos"}): para cada espaco com a rotina ligada, conta e grava os sugeridos. Um espaco
// com erro nao impede os outros. Devolve quantos espacos foram calculados e quantos sugeridos ficaram guardados.
func (s *Combos) Rodar(ctx context.Context) (domain.Doc, error) {
	itens, err := s.avisos.Ativos(ctx)
	if err != nil {
		log.Println("combos: ler itens ativos:", err)
		return nil, ErrInterno
	}
	espacos, sugeridos := 0, 0
	feitos := map[string]bool{}
	for _, it := range itens {
		if tipo, _ := it["tipo"].(string); tipo != domain.TipoItemCombos {
			continue
		}
		sid, _ := it["sid"].(string)
		if sid == "" || feitos[sid] {
			continue
		}
		feitos[sid] = true
		n, err := s.calcular(ctx, sid)
		if err != nil {
			log.Printf("combos: espaco %s: %v", sid, err)
			continue
		}
		espacos++
		sugeridos += n
	}
	return domain.Doc{"espacos": espacos, "sugeridos": sugeridos}, nil
}

// calcular conta os combos de um espaco e grava. Sem o item CFG#COMBOS, cria com a lista vazia (como uma gravacao normal).
func (s *Combos) calcular(ctx context.Context, sid string) (int, error) {
	recentes, _, err := s.lanc.Listar(ctx, sid, domain.FiltroTx{SemPrevistos: true, Limite: domain.CombosAmostra})
	if err != nil {
		return 0, err
	}
	sugeridos := domain.CombosSugeridos(recentes)
	hoje := domain.HojeDe(s.rel.Agora())
	atual, err := s.cfg.Ler(ctx, sid, "combos")
	if err != nil {
		return 0, err
	}
	if atual == nil {
		_, err = s.cfg.Salvar(ctx, sid, "combos", domain.Doc{"lista": []any{}, "sugeridos": sugeridos, "geradoEm": hoje}, 0)
	} else {
		err = s.cfg.AtualizarCombos(ctx, sid, sugeridos, hoje)
	}
	if err != nil {
		return 0, err
	}
	return len(sugeridos), nil
}
