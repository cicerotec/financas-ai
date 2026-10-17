package service

import (
	"context"
	"errors"
	"log"

	"financas/domain"
)

// ErrSalvarTags: as tags nao puderam ser gravadas (erro do banco ou conflito persistente).
var ErrSalvarTags = errors.New("erro ao salvar tags")

// Tags e a lista compartilhada de tags (CFG#TAGS, campo "usadas"): so cresce por aqui. O membro nao reescreve a lista
// (PUT em cfg/tags e so do dono), mas pode acrescentar tags novas.
type Tags struct{ repo RepositorioConfig }

func NewTags(repo RepositorioConfig) *Tags { return &Tags{repo: repo} }

// acrescentar grava no CFG#TAGS apenas o que ainda nao existe, tentando de novo se alguem gravou ao mesmo tempo.
// Devolve o que foi adicionado.
func (s *Tags) acrescentar(ctx context.Context, sid string, novas []string) ([]string, error) {
	for tentativa := 0; tentativa < 3; tentativa++ {
		atual, err := s.repo.Ler(ctx, sid, "tags")
		if err != nil {
			return nil, err
		}
		var existentes []string
		versao := 0.0
		if atual != nil {
			if l, ok := atual["usadas"].([]any); ok {
				for _, x := range l {
					if t, ok := x.(string); ok {
						existentes = append(existentes, t)
					}
				}
			}
			versao, _ = atual["version"].(float64)
		}
		falta := domain.Faltantes(novas, existentes)
		if len(falta) == 0 {
			return nil, nil
		}
		todas := make([]any, 0, len(existentes)+len(falta))
		for _, t := range append(existentes, falta...) {
			todas = append(todas, t)
		}
		_, err = s.repo.Salvar(ctx, sid, "tags", domain.Doc{"usadas": todas}, versao)
		if errors.Is(err, domain.ErrConflitoConfig) {
			continue // alguem gravou ao mesmo tempo: le de novo e tenta outra vez
		}
		if err != nil {
			return nil, err
		}
		return falta, nil
	}
	return nil, errors.New("conflito ao gravar tags")
}

// Adicionar: POST /spaces/{sid}/tags. Devolve as tags que foram de fato adicionadas (nunca nil).
func (s *Tags) Adicionar(ctx context.Context, sid string, body domain.Doc) ([]string, error) {
	novas := domain.NormalizarTags(domain.TagsDe(body))
	if len(novas) == 0 {
		return nil, &ErroValidacao{"informe ao menos uma tag"}
	}
	adicionadas, err := s.acrescentar(ctx, sid, novas)
	if err != nil {
		log.Println("post tags:", err)
		return nil, ErrSalvarTags
	}
	if adicionadas == nil {
		adicionadas = []string{}
	}
	return adicionadas, nil
}

// Garantir e chamado ao salvar um lancamento: tags novas entram na lista sozinhas. Falha aqui nao derruba o
// salvamento do lancamento (so registra).
func (s *Tags) Garantir(ctx context.Context, sid string, tags []string) {
	novas := domain.NormalizarTags(tags)
	if len(novas) == 0 {
		return
	}
	if _, err := s.acrescentar(ctx, sid, novas); err != nil {
		log.Println("garantir tags:", err)
	}
}
