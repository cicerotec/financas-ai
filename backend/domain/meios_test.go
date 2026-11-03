package domain

import (
	"testing"
	"time"
)

func TestDataValida(t *testing.T) {
	for s, quer := range map[string]bool{"2026-12-31": true, "2026-02-29": false, "2026-2-1": false, "": false, "2026-12-31T00:00:00Z": false, "31/12/2026": false} {
		if got := DataValida(s); got != quer {
			t.Errorf("DataValida(%q) = %v, esperado %v", s, got, quer)
		}
	}
}

func TestMeioAtivo(t *testing.T) {
	// ativo enquanto hoje < ativoAte: no proprio dia da data o meio ja deixou de valer
	for _, c := range []struct {
		ate, hoje string
		quer      bool
	}{{"", "2026-11-03", true}, {"2026-12-31", "2026-11-03", true}, {"2026-11-03", "2026-11-03", false}, {"2026-10-31", "2026-11-03", false}} {
		if got := MeioAtivo(c.ate, c.hoje); got != c.quer {
			t.Errorf("MeioAtivo(%q, %q) = %v, esperado %v", c.ate, c.hoje, got, c.quer)
		}
	}
}

func TestHojeDeUsaOFusoDoApp(t *testing.T) {
	// 01/11 02:00 UTC ainda e 31/10 em Sao Paulo
	if got := HojeDe(time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC)); got != "2026-10-31" {
		t.Errorf("HojeDe = %q", got)
	}
}
