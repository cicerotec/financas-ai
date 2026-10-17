package service

import (
	"context"
	"log"

	"financas/domain"
)

// RepositorioContador e o que o service precisa do banco para o contador de alteracoes do espaco.
type RepositorioContador interface {
	// Proximo soma 1 (criando o contador se preciso) e devolve o valor novo.
	Proximo(ctx context.Context, sid string) (float64, error)
	// Atual devolve o valor sem mexer; 0 se ainda nao existe.
	Atual(ctx context.Context, sid string) (float64, error)
}

// Contador conta as alteracoes do espaco. Cada escrita de lancamento (criar, editar, excluir, importar, restaurar) soma 1
// e devolve o valor novo em "_seq". A tela guarda o ultimo que viu: para saber se outra pessoa gravou, le so este valor em
// vez de reler os lancamentos. Se o que volta de uma escrita propria e o anterior + 1, ninguem gravou no meio.
type Contador struct{ repo RepositorioContador }

func NewContador(repo RepositorioContador) *Contador { return &Contador{repo: repo} }

// ComSeq soma 1 ao contador e anexa o valor novo na resposta. A escrita ja foi feita: se o contador falhar, so registra
// no log e a resposta sai sem "_seq" (a tela entende como "nao sei" e recarrega).
func (s *Contador) ComSeq(ctx context.Context, sid string, d domain.Doc) domain.Doc {
	n, err := s.repo.Proximo(ctx, sid)
	if err != nil {
		log.Printf("ERRO seq sid=%s: %v", sid, err)
		return d
	}
	d["_seq"] = n
	return d
}

// Atual: GET /spaces/{sid}/seq.
func (s *Contador) Atual(ctx context.Context, sid string) (float64, error) {
	n, err := s.repo.Atual(ctx, sid)
	if err != nil {
		log.Println("seq:", err)
		return 0, ErrInterno
	}
	return n, nil
}
