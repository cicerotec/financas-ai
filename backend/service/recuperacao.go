package service

import (
	"context"
	"errors"
)

// ErrRecuperacaoInvalida e a unica falha da recuperacao do MFA: e-mail sem conta, pessoa sem MFA, codigo errado ou usado, limite estourado,
// corpo invalido ou falha interna. O servidor nao distingue (spec mfa-opcional, R28): a rota responde sempre a mesma coisa.
var ErrRecuperacaoInvalida = errors.New("recuperacao invalida")

// Recuperacao e a regra da recuperacao do MFA por um codigo da lista (spec mfa-opcional, R27 a R31).
//
// Primeira fatia da Fase 3 (a rota publica no roteador, AD-1): ainda nao ha lista de codigos, limite nem Cognito, entao TODA chamada e uma
// falha e nada e desligado. As portas (a lista de codigos e o desligador de MFA) entram nas proximas fatias, declaradas aqui.
type Recuperacao struct{}

func NewRecuperacao() *Recuperacao { return &Recuperacao{} }

// Recuperar tenta desligar o MFA da pessoa do e-mail com um codigo da lista ativa. origem e o IP de quem chamou (SourceIP, nunca cabecalho).
func (s *Recuperacao) Recuperar(ctx context.Context, origem, email, codigo string) error {
	return ErrRecuperacaoInvalida
}
