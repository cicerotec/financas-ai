package domain

import "errors"

var (
	// ErrConflitoConfig: outra pessoa gravou a configuracao depois da leitura (controle otimista por "version").
	ErrConflitoConfig = errors.New("alterado por outra pessoa")
	// ErrVinculoNaoEncontrado: a pessoa nao tem vinculo com o espaco (USER#<sub> / SPACE#<sid>).
	ErrVinculoNaoEncontrado = errors.New("vinculo com o espaco nao encontrado")
)

// MudancaPerfil e o que se altera no vinculo da pessoa com o espaco. nil = nao mexe; ponteiro para "" = remove o campo.
type MudancaPerfil struct {
	Apelido *string
	ChatID  *string
}
