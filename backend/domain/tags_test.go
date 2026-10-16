package domain

import (
	"reflect"
	"testing"
)

func TestNormalizarTags(t *testing.T) {
	longa := ""
	for i := 0; i < 61; i++ {
		longa += "a"
	}
	got := NormalizarTags([]string{" Lazer ", "#Viagem", "", "   ", "#", "Lazer", longa, "##x"})
	want := []string{"Lazer", "Viagem", "#x"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NormalizarTags = %v, esperado %v", got, want)
	}
	muitas := make([]string, 150)
	for i := range muitas {
		muitas[i] = "t" + string(rune('A'+i%26)) + string(rune('a'+i/26))
	}
	if n := len(NormalizarTags(muitas)); n != 100 {
		t.Errorf("limite de 100 tags por chamada: veio %d", n)
	}
}

func TestFaltantes(t *testing.T) {
	got := Faltantes([]string{"a", "b", "c"}, []string{"b", "x"})
	if !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Errorf("Faltantes = %v", got)
	}
	if Faltantes([]string{"a"}, []string{"a"}) != nil {
		t.Error("nada deveria faltar")
	}
	// maiusculas contam como tags diferentes (o app trata tags como texto exato)
	if got := Faltantes([]string{"Lazer"}, []string{"lazer"}); !reflect.DeepEqual(got, []string{"Lazer"}) {
		t.Errorf("Lazer e lazer sao tags diferentes: %v", got)
	}
}

func TestTagsDe(t *testing.T) {
	d := Doc{"tags": []any{"a", 1, "b", nil}}
	if got := TagsDe(d); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("TagsDe = %v", got)
	}
	if TagsDe(Doc{}) != nil || TagsDe(Doc{"tags": "x"}) != nil {
		t.Error("sem lista de tags deve devolver nil")
	}
}
