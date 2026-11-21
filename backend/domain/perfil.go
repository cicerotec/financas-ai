package domain

import (
	"errors"
	"strings"
	"unicode"
)

// MaxApelido e o tamanho maximo, em letras, do nome de exibicao.
const MaxApelido = 30

// NormalizarApelido tira espacos sobrando e recusa nomes longos ou com caracteres de controle.
// Vazio e valido: significa "sem apelido".
func NormalizarApelido(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) > MaxApelido {
		return "", errors.New("o nome pode ter no maximo 30 caracteres")
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return "", errors.New("o nome tem caracteres invalidos")
		}
	}
	return s, nil
}
