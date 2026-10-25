package service

import (
	"context"
	"errors"
	"log"
	"strings"

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
	out := semItem(d)
	if strings.EqualFold(nome, "listas") {
		if err := s.juntarMeios(ctx, sid, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// juntarMeios acrescenta a lista de meios (item 33a): cada nome de banco com a marca de cartao (e chave de CFG#CARTOES.porBanco) e
// ativo (sempre verdadeiro ate o item 43). O servidor junta os dois itens; o front nao faz chamada extra nem olha status.
func (s *Config) juntarMeios(ctx context.Context, sid string, listas domain.Doc) error {
	nomes, ok := listas["banco"].([]any)
	if !ok {
		return nil
	}
	cartoes, err := s.repo.Ler(ctx, sid, "cartoes")
	if err != nil {
		log.Println("cfg: cartoes:", err)
		return ErrInterno
	}
	por, _ := cartoes["porBanco"].(map[string]any)
	meios := make([]domain.Doc, 0, len(nomes))
	for _, x := range nomes {
		nome, ok := x.(string)
		if !ok {
			continue
		}
		_, cartao := por[nome].(map[string]any)
		meios = append(meios, domain.Doc{"nome": nome, "cartao": cartao, "ativo": true})
	}
	listas["meios"] = meios
	return nil
}

// Gravar grava a configuracao. O "version" do corpo e o que o cliente leu; os campos protegidos nao entram. No conflito,
// devolve *ErroConflitoConfig com o estado atual para o cliente mesclar.
func (s *Config) Gravar(ctx context.Context, sid, nome string, body domain.Doc) (domain.Doc, error) {
	esperada, _ := body["version"].(float64)
	delete(body, "version")
	domain.Protegidos(body)
	switch strings.ToLower(nome) {
	case "listas":
		delete(body, "meios") // derivado pelo servidor na leitura, nunca gravado
	case "cartoes":
		s.preservarConferidas(ctx, sid, body) // a conferencia da fatura so o servidor grava (R28)
	}
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

// preservarConferidas mantem, em cada fatura do corpo, o `conferida` ja gravado e descarta o que o cliente mandou (item 33a, R28): o total
// conferido e do servidor (POST .../conferir). A rota generica some na fase C.
func (s *Config) preservarConferidas(ctx context.Context, sid string, body domain.Doc) {
	atual, _ := s.repo.Ler(ctx, sid, "cartoes")
	gravadas := map[string]any{}
	porAtual, _ := atual["porBanco"].(map[string]any)
	for banco, c := range porAtual {
		fs, _ := c.(map[string]any)["faturas"].(map[string]any)
		for ym, f := range fs {
			if conf, ok := f.(map[string]any)["conferida"]; ok {
				gravadas[banco+"|"+ym] = conf
			}
		}
	}
	por, _ := body["porBanco"].(map[string]any)
	for banco, c := range por {
		cm, _ := c.(map[string]any)
		fs, _ := cm["faturas"].(map[string]any)
		for ym, f := range fs {
			fm, ok := f.(map[string]any)
			if !ok {
				continue
			}
			delete(fm, "conferida")
			if conf, ok := gravadas[banco+"|"+ym]; ok {
				fm["conferida"] = conf
			}
		}
	}
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
