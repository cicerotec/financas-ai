package domain

import (
	"testing"
	"time"
)

func TestNormalizarTiraAcentoEMaiuscula(t *testing.T) {
	for entrada, quer := range map[string]string{"Açúcar e CAFÉ": "acucar e cafe", "Saúde": "saude", "ÃÕÂÊÔ ñ ü": "aoaeo n u", "plain": "plain", "": ""} {
		if got := Normalizar(entrada); got != quer {
			t.Errorf("Normalizar(%q) = %q, esperado %q", entrada, got, quer)
		}
	}
}

func TestTermoBuscaTiraHashEEspacos(t *testing.T) {
	for entrada, quer := range map[string]string{"#Saúde": "saude", "  Farmácia   Popular ": "farmacia popular", "##x": "x", "#": "", "   ": ""} {
		if got := TermoBusca(entrada); got != quer {
			t.Errorf("TermoBusca(%q) = %q, esperado %q", entrada, got, quer)
		}
	}
}

func TestEmDentroDeDiasComecaNoInicioDoDiaEmSaoPaulo(t *testing.T) {
	agora := time.Date(2026, 10, 19, 15, 0, 0, 0, time.UTC) // 12h em Sao Paulo
	casos := []struct {
		em   string
		dias int
		quer bool
	}{
		{"2026-09-19T03:00:00.000Z", 30, true},  // 00h de 19/09 em Sao Paulo: o primeiro instante da janela
		{"2026-09-19T02:59:59.000Z", 30, false}, // 23h59 de 18/09
		{"2026-10-19T15:00:00.000Z", 30, true},  // agora
		{"2026-10-19T15:00:01.000Z", 30, false}, // no futuro
		{"2025-03-01T12:00:00.000Z", 365, false},
		{"lixo", 30, false},
	}
	for _, c := range casos {
		if got := EmDentroDeDias(c.em, agora, c.dias); got != c.quer {
			t.Errorf("EmDentroDeDias(%q, %d) = %v, esperado %v", c.em, c.dias, got, c.quer)
		}
	}
}
