package domain

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// MaxApelido e o tamanho maximo, em letras, do nome de exibicao.
const MaxApelido = 30

var reChatID = regexp.MustCompile(`^-?[0-9]{5,20}$`)

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

// ChatIDValido diz se o texto e um chat id do Telegram (so numeros, com sinal opcional, de 5 a 20 digitos).
func ChatIDValido(s string) bool { return reChatID.MatchString(s) }
