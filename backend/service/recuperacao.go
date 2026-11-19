package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"
	"strings"

	"financas/domain"
)

// ErrRecuperacaoInvalida e a unica falha da recuperacao do MFA: e-mail sem conta, pessoa sem MFA, codigo errado ou usado, limite estourado,
// corpo invalido ou falha interna. O servidor nao distingue (spec mfa-opcional, R28): a rota responde sempre a mesma coisa.
var ErrRecuperacaoInvalida = errors.New("recuperacao invalida")

// Identidade e o que o service precisa do login (o Cognito): achar a pessoa pelo e-mail e desligar o MFA dela. So estas duas coisas
// (R32): nunca cria sessao, nunca devolve token e nunca altera a senha (R31).
type Identidade interface {
	// BuscarPessoa acha a pessoa pelo e-mail ja normalizado. achou=false se nao ha conta; erro so para falha do servico.
	BuscarPessoa(ctx context.Context, email string) (p domain.PessoaMFA, achou bool, err error)
	// DesligarMFA desliga o aplicativo autenticador da pessoa (pelo sub).
	DesligarMFA(ctx context.Context, sub string) error
}

// RepositorioRecuperacao e o que o service precisa do banco para usar um codigo da lista ativa (R27, R36).
type RepositorioRecuperacao interface {
	LerAtiva(ctx context.Context, sub string) (*domain.ListaCodigos, error)
	MarcarUsado(ctx context.Context, sub, listaID string, i int) (bool, error)
	DesfazerUsado(ctx context.Context, sub, listaID string, i int) error
	ApagarAtiva(ctx context.Context, sub, listaID string) error
}

// Recuperacao e a regra da recuperacao do MFA por um codigo da lista (spec mfa-opcional, R27 a R31).
type Recuperacao struct {
	id      Identidade
	repo    RepositorioRecuperacao
	rel     Relogio
	limites *LimitadorRecuperacao
}

func NewRecuperacao(id Identidade, repo RepositorioRecuperacao, rel Relogio) *Recuperacao {
	return &Recuperacao{id: id, repo: repo, rel: rel, limites: NewLimitadorRecuperacao()}
}

// Recuperar tenta desligar o MFA da pessoa do e-mail com um codigo da lista ativa. origem e o IP de quem chamou (SourceIP, nunca cabecalho).
// Toda falha e ErrRecuperacaoInvalida. A ordem vai da checagem mais barata a mais cara (R29): formato do codigo, limite por origem,
// Cognito, limite do par, leitura da lista. Nenhum log leva e-mail, origem, sub nem codigo (R34).
func (s *Recuperacao) Recuperar(ctx context.Context, origem, email, codigo string) error {
	agora := s.rel.Agora()
	cod, ok := domain.NormalizarCodigo(codigo)
	if !ok { // formato invalido: a mesma resposta, sem chamar o Cognito
		return ErrRecuperacaoInvalida
	}
	if !s.limites.PermiteOrigem(origem, agora) {
		return ErrRecuperacaoInvalida
	}
	email = strings.ToLower(strings.TrimSpace(email)) // o pool diferencia maiusculas e os e-mails sao gravados em minusculas (R29)
	if email == "" {
		return ErrRecuperacaoInvalida
	}
	p, achou, err := s.id.BuscarPessoa(ctx, email)
	if err != nil {
		log.Println("recuperacao: falha ao consultar o login")
		return ErrRecuperacaoInvalida
	}
	if !achou || p.Sub == "" { // e-mail sem conta: nao toca o DynamoDB
		return ErrRecuperacaoInvalida
	}
	par := p.Sub + "|" + origem
	if s.limites.ParBloqueado(par, agora) {
		return ErrRecuperacaoInvalida
	}
	falha := func() error {
		s.limites.RegistrarFalhaPar(par, agora)
		return ErrRecuperacaoInvalida
	}
	if !p.Habilitada || !p.TemMFA { // conta desabilitada ou sem MFA: nada a desligar
		return falha()
	}
	lista, err := s.repo.LerAtiva(ctx, p.Sub)
	if err != nil {
		log.Println("recuperacao: falha ao ler a lista")
		return ErrRecuperacaoInvalida
	}
	if lista == nil { // sem lista ativa (so ha pendente, ou nada): a pendente nunca recupera
		return falha()
	}
	// compara com TODOS os hashes, em tempo constante, sem parar no primeiro que casa
	h := []byte(domain.HashCodigo(lista.ListaID, cod))
	idx := -1
	for i, c := range lista.Codigos {
		if subtle.ConstantTimeCompare([]byte(c.Hash), h) == 1 && idx < 0 {
			idx = i
		}
	}
	if idx < 0 || lista.Codigos[idx].Usado {
		return falha()
	}
	marcado, err := s.repo.MarcarUsado(ctx, p.Sub, lista.ListaID, idx) // condicional: dois usos simultaneos, so um passa
	if err != nil {
		log.Println("recuperacao: falha ao marcar o codigo")
		return ErrRecuperacaoInvalida
	}
	if !marcado {
		return falha()
	}
	if err := s.id.DesligarMFA(ctx, p.Sub); err != nil {
		log.Println("recuperacao: falha ao desligar o MFA no login")
		if e := s.repo.DesfazerUsado(ctx, p.Sub, lista.ListaID, idx); e != nil { // um codigo nao e queimado por falha do Cognito
			log.Println("recuperacao: falha ao desfazer a marca do codigo")
		}
		return ErrRecuperacaoInvalida
	}
	if err := s.repo.ApagarAtiva(ctx, p.Sub, lista.ListaID); err != nil { // usar um codigo invalida o resto da lista
		log.Println("recuperacao: falha ao apagar a lista usada")
	}
	s.limites.ZerarPar(par)
	return nil
}
