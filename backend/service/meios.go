package service

import (
	"context"
	"errors"
	"log"
	"time"

	"financas/domain"
)

// UsoDeMeio e o que o service precisa dos lancamentos para saber se um meio ja foi usado (item 43).
type UsoDeMeio interface {
	// UsaMeio diz se algum lancamento (vivo, excluido logicamente ou previsto) tem o meio; para no primeiro achado. paginas e itens dizem o custo.
	UsaMeio(ctx context.Context, sid, meio string) (usa bool, paginas, itens int, err error)
}

var (
	ErrMeioNaoEncontrado = errors.New("meio de pagamento nao encontrado")
	ErrMeioEmUso         = errors.New("esse meio tem lancamentos: desative o meio em vez de remover")
	ErrMeioECartao       = errors.New("esse meio e um cartao: remova o cartao na aba Cartao antes")
)

// Meios desativa, reativa e remove os meios de pagamento (CFG#LISTAS.banco e CFG#LISTAS.ativoAte). Adicionar continua pelo PUT de Listas.
type Meios struct {
	cfg RepositorioConfig
	uso UsoDeMeio
	rel Relogio
}

func NewMeios(cfg RepositorioConfig, uso UsoDeMeio, rel Relogio) *Meios {
	return &Meios{cfg: cfg, uso: uso, rel: rel}
}

// nomesDeBanco devolve os nomes da lista banco de CFG#LISTAS.
func nomesDeBanco(listas domain.Doc) []string {
	var nomes []string
	l, _ := listas["banco"].([]any)
	for _, x := range l {
		if s, ok := x.(string); ok {
			nomes = append(nomes, s)
		}
	}
	return nomes
}

func temNome(nomes []string, nome string) bool {
	for _, n := range nomes {
		if n == nome {
			return true
		}
	}
	return false
}

// lerListas devolve CFG#LISTAS, sem as chaves internas, e a versao lida; ErrMeioNaoEncontrado se o meio nao esta em banco.
func (s *Meios) lerListas(ctx context.Context, sid, meio string) (domain.Doc, float64, error) {
	atual, err := s.cfg.Ler(ctx, sid, "listas")
	if err != nil {
		log.Println("meios: ler listas:", err)
		return nil, 0, ErrInterno
	}
	if !temNome(nomesDeBanco(atual), meio) {
		return nil, 0, ErrMeioNaoEncontrado
	}
	versao, _ := atual["version"].(float64)
	corpo := domain.Doc{}
	for k, v := range atual {
		if k != "PK" && k != "SK" && k != "version" {
			corpo[k] = v
		}
	}
	return corpo, versao, nil
}

func (s *Meios) gravar(ctx context.Context, sid string, corpo domain.Doc, versao float64) error {
	if _, err := s.cfg.Salvar(ctx, sid, "listas", corpo, versao); err != nil {
		if errors.Is(err, domain.ErrConflitoConfig) {
			atual, _ := s.cfg.Ler(ctx, sid, "listas")
			return &ErroConflitoConfig{Atual: domain.Publico(atual)}
		}
		log.Println("meios: salvar listas:", err)
		return ErrSalvar
	}
	return nil
}

// Definir: PUT /spaces/{sid}/meios/{meio}. Corpo {"ativoAte": "AAAA-MM-DD"} desativa a partir da data; {"ativoAte": null} reativa. Nao toca em lancamento.
func (s *Meios) Definir(ctx context.Context, sid, meio string, body domain.Doc) (domain.Doc, error) {
	v, tem := body["ativoAte"]
	if !tem {
		return nil, &ErroValidacao{"informe ativoAte (AAAA-MM-DD) ou null para reativar"}
	}
	data, ehTexto := v.(string)
	if v != nil && (!ehTexto || !domain.DataValida(data)) {
		return nil, &ErroValidacao{"ativoAte deve ser uma data AAAA-MM-DD ou null"}
	}
	corpo, versao, err := s.lerListas(ctx, sid, meio)
	if err != nil {
		return nil, err
	}
	mapa := map[string]any{}
	if m, ok := corpo["ativoAte"].(map[string]any); ok {
		for k, x := range m {
			mapa[k] = x
		}
	}
	if v == nil {
		delete(mapa, meio)
	} else {
		mapa[meio] = data
	}
	if len(mapa) == 0 {
		delete(corpo, "ativoAte")
	} else {
		corpo["ativoAte"] = mapa
	}
	if err := s.gravar(ctx, sid, corpo, versao); err != nil {
		return nil, err
	}
	ate, _ := mapa[meio].(string)
	res := domain.Doc{"meio": meio, "ativo": domain.MeioAtivo(ate, domain.HojeDe(s.rel.Agora()))}
	if ate != "" {
		res["ativoAte"] = ate
	} else {
		res["ativoAte"] = nil
	}
	return res, nil
}

// Remover: DELETE /spaces/{sid}/meios/{meio}. So remove o meio que nunca foi usado; a conferencia e uma varredura que so roda aqui e para no primeiro achado.
func (s *Meios) Remover(ctx context.Context, sid, meio string) error {
	corpo, versao, err := s.lerListas(ctx, sid, meio)
	if err != nil {
		return err
	}
	cartoes, err := s.cfg.Ler(ctx, sid, "cartoes")
	if err != nil {
		log.Println("meios: ler cartoes:", err)
		return ErrInterno
	}
	if por, _ := cartoes["porBanco"].(map[string]any); por[meio] != nil {
		log.Printf("meios: remover paginas=0 itens=0 ms=0 resultado=cartao")
		return ErrMeioECartao
	}
	t0 := time.Now()
	usa, paginas, itens, err := s.uso.UsaMeio(ctx, sid, meio)
	resultado := "ok"
	switch {
	case err != nil:
		resultado = "erro"
	case usa:
		resultado = "em_uso"
	}
	// so os numeros e o resultado: o nome do meio e dado do dono e nao vai ao log
	log.Printf("meios: remover paginas=%d itens=%d ms=%d resultado=%s", paginas, itens, time.Since(t0).Milliseconds(), resultado)
	if err != nil {
		log.Println("meios: varredura:", err)
		return ErrListar
	}
	if usa {
		return ErrMeioEmUso
	}
	resto := []any{}
	for _, n := range nomesDeBanco(corpo) {
		if n != meio {
			resto = append(resto, n)
		}
	}
	corpo["banco"] = resto
	if m, ok := corpo["ativoAte"].(map[string]any); ok {
		novo := map[string]any{}
		for k, x := range m {
			if k != meio {
				novo[k] = x
			}
		}
		if len(novo) == 0 {
			delete(corpo, "ativoAte")
		} else {
			corpo["ativoAte"] = novo
		}
	}
	return s.gravar(ctx, sid, corpo, versao)
}
