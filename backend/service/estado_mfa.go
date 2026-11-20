package service

import (
	"context"
	"log"
	"time"

	"financas/domain"
)

// EstadoMFA responde "o MFA desta pessoa esta ligado e ha lista de codigos de recuperacao ativa?" (spec mfa-opcional, R37). O estado do MFA vem
// do login (o Cognito) e a lista, do banco; o front so exibe. A pessoa e SEMPRE a do token (o sub), nunca a do corpo nem da URL, e a resposta nao
// leva hash, listaId, e-mail nem codigo.
type EstadoMFA struct {
	id   Identidade
	repo RepositorioRecuperacao
}

func NewEstadoMFA(id Identidade, repo RepositorioRecuperacao) *EstadoMFA {
	return &EstadoMFA{id: id, repo: repo}
}

// Ler devolve {"ligado": bool, "lista": {"ativa": bool, "geradaEm": "AAAA-MM-DDThh:mm:ssZ" (so se ha lista)}}. ErrInterno se o login ou o
// banco falharem: sem estado pela metade.
func (s *EstadoMFA) Ler(ctx context.Context, sub string) (domain.Doc, error) {
	p, achou, err := s.id.BuscarPessoa(ctx, sub)
	if err != nil || !achou {
		log.Println("estado mfa: falha ao consultar o login")
		return nil, ErrInterno
	}
	lista, err := s.repo.LerAtiva(ctx, sub)
	if err != nil {
		log.Println("estado mfa: falha ao ler a lista")
		return nil, ErrInterno
	}
	l := domain.Doc{"ativa": lista != nil}
	if lista != nil && !lista.GeradaEm.IsZero() {
		l["geradaEm"] = lista.GeradaEm.UTC().Format(time.RFC3339)
	}
	return domain.Doc{"ligado": p.TemMFA, "lista": l}, nil
}
