package service

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"log"
	"time"

	"financas/domain"
)

// RepositorioCodigos e o que o service precisa do banco para as listas de codigos de recuperacao (spec mfa-opcional, R25, R25a e R36).
// So o repository sabe onde elas ficam.
type RepositorioCodigos interface {
	// SalvarPendente grava a lista pendente da pessoa, substituindo a anterior. Nao toca na ativa.
	SalvarPendente(ctx context.Context, sub string, l domain.ListaCodigos) error
	// Confirmar promove a pendente listaID a ativa e apaga a ativa anterior, de forma condicional e idempotente.
	// domain.ErrListaSubstituida se a pendente nao existe, mudou ou venceu (e a ativa nao e a pedida).
	Confirmar(ctx context.Context, sub, listaID string, agora time.Time) error
}

// CodigosDeRecuperacao gera e confirma a lista de codigos de recuperacao. O sub vem SEMPRE do token de quem chama, nunca do corpo.
// O codigo em texto so existe na resposta de Gerar: nunca e gravado nem logado.
type CodigosDeRecuperacao struct {
	repo   RepositorioCodigos
	avisos Avisador // avisa a pessoa pelo canal que ja tem ligado (R35); pode ser nil
	rel    Relogio
	fonte  io.Reader // bytes aleatorios; crypto/rand em producao
}

func NewCodigosDeRecuperacao(repo RepositorioCodigos, avisos Avisador, rel Relogio) *CodigosDeRecuperacao {
	return &CodigosDeRecuperacao{repo: repo, avisos: avisos, rel: rel, fonte: rand.Reader}
}

// UsarFonte troca a fonte de bytes aleatorios (so para os testes serem deterministicos).
func (s *CodigosDeRecuperacao) UsarFonte(f io.Reader) { s.fonte = f }

// Gerar: POST /me/mfa/codigos. Cria uma lista PENDENTE (substituindo a pendente anterior) e devolve {listaId, codigos} uma unica vez.
// Nao toca na lista ativa e nao a invalida: a pessoa nunca fica sem saida.
func (s *CodigosDeRecuperacao) Gerar(ctx context.Context, sub string) (domain.Doc, error) {
	id, err := domain.NovoListaID(s.fonte)
	if err != nil {
		log.Println("mfa codigos: sorteio:", err)
		return nil, ErrInterno
	}
	agora := s.rel.Agora()
	lista := domain.ListaCodigos{ListaID: id, GeradaEm: agora, ExpiraEm: agora.Add(domain.ValidadePendente)}
	mostrar := make([]string, 0, domain.QuantidadeCodigos)
	vistos := map[string]bool{}
	for tentativas := 0; len(mostrar) < domain.QuantidadeCodigos; tentativas++ {
		if tentativas >= 3*domain.QuantidadeCodigos { // com 100 bits nunca chega aqui; uma fonte quebrada nao pode prender a rota num laco
			log.Println("mfa codigos: a fonte aleatoria repete codigos")
			return nil, ErrInterno
		}
		c, err := domain.NovoCodigo(s.fonte)
		if err != nil {
			log.Println("mfa codigos: sorteio:", err)
			return nil, ErrInterno
		}
		if vistos[c] {
			continue
		}
		vistos[c] = true
		lista.Codigos = append(lista.Codigos, domain.CodigoGuardado{Hash: domain.HashCodigo(id, c)})
		mostrar = append(mostrar, domain.FormatarCodigo(c))
	}
	if err := s.repo.SalvarPendente(ctx, sub, lista); err != nil {
		log.Println("mfa codigos: salvar pendente:", err) // sem sub, sem codigo
		return nil, ErrSalvar
	}
	return domain.Doc{"listaId": id, "codigos": mostrar}, nil
}

// Confirmar: POST /me/mfa/codigos/confirmar com {"listaId"}. Promove a pendente a ativa e apaga a ativa anterior; idempotente.
// domain.ErrListaSubstituida (409) se a pendente foi substituida por outra geracao ou venceu.
func (s *CodigosDeRecuperacao) Confirmar(ctx context.Context, sub string, body domain.Doc) (domain.Doc, error) {
	id, ok := body["listaId"].(string)
	if !ok || !domain.ListaIDValido(id) {
		return nil, &ErroValidacao{"informe o listaId"}
	}
	err := s.repo.Confirmar(ctx, sub, id, s.rel.Agora())
	switch {
	case errors.Is(err, domain.ErrListaSubstituida):
		return nil, err
	case err != nil:
		log.Println("mfa codigos: confirmar:", err)
		return nil, ErrSalvar
	}
	if s.avisos != nil { // o 200 do confirmar: "uma nova lista foi gerada" (NAO diz que o MFA foi ligado: ele so liga depois, na tela)
		s.avisos.Avisar(ctx, sub, domain.MensagemListaGerada)
	}
	return domain.Doc{"ok": true}, nil
}
