package service

import (
	"context"
	"log"

	"financas/domain"
)

// RepositorioEspacos e o que o service precisa do banco para saber a quem a pessoa pertence.
type RepositorioEspacos interface {
	// Papel devolve o papel da pessoa no espaco; "" = sem vinculo.
	Papel(ctx context.Context, sid, sub string) (string, error)
	// EspacosDe devolve os espacos da pessoa, com o nome de cada um.
	EspacosDe(ctx context.Context, sub string) ([]domain.Espaco, error)
}

// Espacos responde "quem e esta pessoa neste espaco" e "a que espacos ela pertence".
type Espacos struct{ repo RepositorioEspacos }

func NewEspacos(repo RepositorioEspacos) *Espacos { return &Espacos{repo: repo} }

// Papel devolve o papel da pessoa no espaco ("" = sem acesso). Falha de leitura vira ErrInterno.
func (s *Espacos) Papel(ctx context.Context, sid, sub string) (string, error) {
	role, err := s.repo.Papel(ctx, sid, sub)
	if err != nil {
		log.Println("papel:", err)
		return "", ErrInterno
	}
	return role, nil
}

// Meus devolve os espacos da pessoa. Falha de leitura vira ErrInterno.
func (s *Espacos) Meus(ctx context.Context, sub string) ([]domain.Espaco, error) {
	es, err := s.repo.EspacosDe(ctx, sub)
	if err != nil {
		log.Println("me:", err)
		return nil, ErrInterno
	}
	return es, nil
}
