package domain

import "errors"

// MaxAparelhos e o limite de aparelhos que uma pessoa registra por espaco para receber avisos.
const MaxAparelhos = 5

// ErrLimiteAparelhos: a pessoa ja tem MaxAparelhos aparelhos neste espaco e o novo e outro (409). O texto e a frase da API.
var ErrLimiteAparelhos = errors.New("Você já tem 5 aparelhos neste espaço. Desligue um deles para ligar este.")

// TokenAparelhoValido diz se o texto tem o formato de um token de aparelho: sem espacos, so letras, numeros e _ : . -, entre 20 e 1024
// caracteres. Nao diz se o token existe: isso so o servico de entrega sabe.
func TokenAparelhoValido(t string) bool {
	if len(t) < 20 || len(t) > 1024 {
		return false
	}
	for _, r := range t {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == ':' || r == '.' || r == '-'
		if !ok {
			return false
		}
	}
	return true
}
