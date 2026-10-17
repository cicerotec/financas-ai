package domain

import (
	"testing"
	"time"
)

func TestMesAtualPeloFusoDeSaoPaulo(t *testing.T) {
	casos := []struct {
		agora time.Time
		quer  string
	}{
		{time.Date(2026, 10, 5, 15, 0, 0, 0, time.UTC), "2026-10"},
		{time.Date(2026, 11, 1, 2, 59, 0, 0, time.UTC), "2026-10"}, // 23h59 de 31/10 em Sao Paulo
		{time.Date(2026, 11, 1, 3, 0, 0, 0, time.UTC), "2026-11"},  // 00h00 de 01/11 em Sao Paulo
		{time.Date(2027, 1, 1, 2, 0, 0, 0, time.UTC), "2026-12"},   // virada de ano
	}
	for _, c := range casos {
		if got := MesAtual(c.agora); got != c.quer {
			t.Errorf("MesAtual(%v) = %s, esperado %s", c.agora, got, c.quer)
		}
	}
}

func TestUltimoConferido(t *testing.T) {
	if got := UltimoConferido(nil); got != "" {
		t.Errorf("sem item = %q", got)
	}
	if got := UltimoConferido(Doc{"conferidos": map[string]any{}}); got != "" {
		t.Errorf("sem conferidos = %q", got)
	}
	d := Doc{"conferidos": map[string]any{"2026-03": 1, "2026-09": 2, "2026-07": 3}}
	if got := UltimoConferido(d); got != "2026-09" {
		t.Errorf("ultimo = %q", got)
	}
}

func TestMesDeInicio(t *testing.T) {
	cfg := Doc{"porBanco": map[string]any{
		"ITAU":   map[string]any{"dataInicio": "2026-01-15"},
		"NUBANK": map[string]any{"dataInicio": "2026-1"},
		"C6":     map[string]any{"dataInicio": "lixo-de-data"},
		"BB":     map[string]any{},
	}}
	casos := map[string]string{"ITAU": "2026-01", "NUBANK": "", "C6": "", "BB": "", "OUTRO": ""}
	for banco, quer := range casos {
		if got := MesDeInicio(cfg, banco); got != quer {
			t.Errorf("MesDeInicio(%s) = %q, esperado %q", banco, got, quer)
		}
	}
	if got := MesDeInicio(nil, "ITAU"); got != "" {
		t.Errorf("sem configuracao = %q", got)
	}
}
