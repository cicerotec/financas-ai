package domain

import (
	"strings"
	"testing"
)

func TestTokenAparelhoValido(t *testing.T) {
	ok := []string{strings.Repeat("a", 20), "dXyZ_12-34:ab.cd" + strings.Repeat("9", 10), strings.Repeat("a", 1024)}
	ruim := []string{"", strings.Repeat("a", 19), strings.Repeat("a", 1025), strings.Repeat("a", 10) + " " + strings.Repeat("a", 10),
		strings.Repeat("a", 20) + "/", strings.Repeat("a", 20) + "ç", strings.Repeat("a", 20) + "\n"}
	for _, t1 := range ok {
		if !TokenAparelhoValido(t1) {
			t.Errorf("%.30q deveria ser valido", t1)
		}
	}
	for _, t1 := range ruim {
		if TokenAparelhoValido(t1) {
			t.Errorf("%.30q deveria ser invalido", t1)
		}
	}
}

func TestContagensSoTemInteiros(t *testing.T) {
	r := ResumoAvisos{Marca: "[DEV] ", Linhas: []LinhaAviso{
		{Kind: AvisoAtrasado}, {Kind: AvisoAtrasado, Tipo: "entrada"}, {Kind: AvisoHoje}, {Kind: AvisoAntes}, {Kind: AvisoAntes},
	}}
	c := Contagens(r)
	if c.Atrasados != 2 || c.Hoje != 1 || c.AVencer != 2 || c.Marca != "[DEV] " || c.Teste {
		t.Errorf("contagens = %+v", c)
	}
	if c := Contagens(ResumoAvisos{Teste: true}); !c.Teste || c.Atrasados+c.Hoje+c.AVencer != 0 {
		t.Errorf("teste = %+v", c)
	}
}
