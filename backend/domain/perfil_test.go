package domain

import (
	"strings"
	"testing"
)

func TestNormalizarApelido(t *testing.T) {
	ok := []struct{ in, quer string }{
		{"Maria", "Maria"},
		{"  Maria   da  Silva ", "Maria da Silva"},
		{"José André", "José André"},
		{"", ""},
		{"   ", ""},
		{"Zé\tCarlos\n", "Zé Carlos"}, // tab e quebra de linha viram espaco
		{"<b>Eu</b>", "<b>Eu</b>"},    // a tela escapa o HTML; o servidor so guarda texto
		{strings.Repeat("a", 30), strings.Repeat("a", 30)},
		{strings.Repeat("é", 30), strings.Repeat("é", 30)}, // conta caracteres, nao bytes
	}
	for _, c := range ok {
		got, err := NormalizarApelido(c.in)
		if err != nil || got != c.quer {
			t.Errorf("NormalizarApelido(%q) = (%q, %v), esperado %q", c.in, got, err, c.quer)
		}
	}
	for _, in := range []string{strings.Repeat("a", 31), "Ana\x00", "Ana\x07"} {
		if _, err := NormalizarApelido(in); err == nil {
			t.Errorf("NormalizarApelido(%q) deveria recusar", in)
		}
	}
}
