package service

import (
	"context"
	"errors"
	"log"

	"financas/domain"
)

// RepositorioConfig e o que o service precisa do banco para as configuracoes do espaco.
type RepositorioConfig interface {
	// Ler devolve a configuracao compartilhada, ou nil se ainda nao existe.
	Ler(ctx context.Context, sid, nome string) (domain.Doc, error)
	// Salvar grava com controle otimista (esperada = version lida). domain.ErrConflitoConfig se alguem gravou antes.
	Salvar(ctx context.Context, sid, nome string, body domain.Doc, esperada float64) (domain.Doc, error)
	LerAparencia(ctx context.Context, sid, sub string) (domain.Doc, error)
	GravarAparencia(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error)
}

// ErroConflitoConfig e o 409 da configuracao: outra pessoa gravou antes; Atual e o estado que ficou no banco.
type ErroConflitoConfig struct{ Atual domain.Doc }

func (e *ErroConflitoConfig) Error() string { return domain.ErrConflitoConfig.Error() }

// Config le e grava as configuracoes compartilhadas (listas, tags, combos, saldos, cartoes) e a aparencia de cada pessoa.
type Config struct{ repo RepositorioConfig }

func NewConfig(repo RepositorioConfig) *Config { return &Config{repo: repo} }

// semItem e o que se devolve quando a configuracao ainda nao existe.
func semItem(d domain.Doc) domain.Doc {
	if d == nil {
		return domain.Doc{"version": 0}
	}
	return domain.Publico(d)
}

// Ler devolve a configuracao compartilhada; sem item, so {"version": 0}.
func (s *Config) Ler(ctx context.Context, sid, nome string) (domain.Doc, error) {
	d, err := s.repo.Ler(ctx, sid, nome)
	if err != nil {
		log.Println("cfg:", err)
		return nil, ErrInterno
	}
	return semItem(d), nil
}

// Gravar grava a configuracao. O "version" do corpo e o que o cliente leu; os campos protegidos nao entram. No conflito,
// devolve *ErroConflitoConfig com o estado atual para o cliente mesclar.
func (s *Config) Gravar(ctx context.Context, sid, nome string, body domain.Doc) (domain.Doc, error) {
	esperada, _ := body["version"].(float64)
	delete(body, "version")
	domain.Protegidos(body)
	res, err := s.repo.Salvar(ctx, sid, nome, body, esperada)
	switch {
	case errors.Is(err, domain.ErrDadosInvalidos):
		return nil, err
	case errors.Is(err, domain.ErrConflitoConfig):
		atual, _ := s.repo.Ler(ctx, sid, nome)
		return nil, &ErroConflitoConfig{Atual: domain.Publico(atual)}
	case err != nil:
		log.Println("put cfg:", err)
		return nil, ErrSalvar
	}
	return res, nil
}

// LerAparencia devolve a aparencia da pessoa; sem item, so {"version": 0}.
func (s *Config) LerAparencia(ctx context.Context, sid, sub string) (domain.Doc, error) {
	d, err := s.repo.LerAparencia(ctx, sid, sub)
	if err != nil {
		log.Println("aparencia:", err)
		return nil, ErrInterno
	}
	return semItem(d), nil
}

// GravarAparencia substitui a aparencia da pessoa (so a dela).
func (s *Config) GravarAparencia(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error) {
	domain.Protegidos(body)
	res, err := s.repo.GravarAparencia(ctx, sid, sub, body)
	switch {
	case errors.Is(err, domain.ErrDadosInvalidos):
		return nil, err
	case err != nil:
		log.Println("put aparencia:", err)
		return nil, ErrSalvar
	}
	return res, nil
}
