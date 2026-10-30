package domain

import "testing"

// RC-T5, RC-T6: a conta das datas parte sempre do dia original, no mesmo horario local (13:00Z = 10:00 em Sao Paulo).
func TestSomaMesesRecorrencia(t *testing.T) {
	casos := []struct {
		base      string
		dia, meses int
		quer      string
	}{
		{"2026-01-31T13:00:00.000Z", 31, 1, "2026-02-28T13:00:00.000Z"},
		{"2026-01-31T13:00:00.000Z", 31, 2, "2026-03-31T13:00:00.000Z"},
		{"2026-01-31T13:00:00.000Z", 31, 3, "2026-04-30T13:00:00.000Z"},
		{"2028-02-29T13:00:00.000Z", 29, 12, "2029-02-28T13:00:00.000Z"},
		{"2028-02-29T13:00:00.000Z", 29, 48, "2032-02-29T13:00:00.000Z"},
		{"2026-10-28T13:00:00.000Z", 28, 2, "2026-12-28T13:00:00.000Z"},
		{"2026-10-28T13:00:00.000Z", 28, 6, "2027-04-28T13:00:00.000Z"},
		{"2026-10-28T13:00:00.000Z", 28, 12, "2027-10-28T13:00:00.000Z"},
		// 02:00Z de 01/11 ainda e 31/10 em Sao Paulo: o dia do mes e o local
		{"2026-11-01T02:00:00.000Z", 31, 1, "2026-12-01T02:00:00.000Z"},
	}
	for _, c := range casos {
		got, err := SomaMesesRecorrencia(c.base, c.dia, c.meses)
		if err != nil || got != c.quer {
			t.Errorf("%s +%d meses (dia %d): %s %v, esperado %s", c.base, c.meses, c.dia, got, err, c.quer)
		}
	}
}

func TestProximaCopiaRecorrenciaSempreDepoisDeHoje(t *testing.T) {
	// a base e de janeiro; hoje e 05/02: 01/02 ja passou, a primeira depois de hoje e 01/03
	got, err := ProximaCopiaRecorrencia("2026-01-01T13:00:00.000Z", 1, 1, "2026-02-05")
	if err != nil || got != "2026-03-01T13:00:00.000Z" {
		t.Errorf("mensal: %s %v", got, err)
	}
	// semestral marcado em janeiro: julho, nao depois
	got, _ = ProximaCopiaRecorrencia("2026-01-15T13:00:00.000Z", 15, 6, "2026-02-05")
	if got != "2026-07-15T13:00:00.000Z" {
		t.Errorf("semestral: %s", got)
	}
	// a copia no proprio dia de hoje ainda nao e futura
	got, _ = ProximaCopiaRecorrencia("2026-01-05T13:00:00.000Z", 5, 1, "2026-02-05")
	if got != "2026-03-05T13:00:00.000Z" {
		t.Errorf("hoje nao conta: %s", got)
	}
}

func TestValidarRecorrente(t *testing.T) {
	d := Doc{"recorrente": map[string]any{"intervalo": 3.0, "lixo": 1.0}}
	if err := ValidarRecorrente(d); err != nil {
		t.Fatal(err)
	}
	if r := d["recorrente"].(Doc); len(r) != 1 || r["intervalo"] != 3.0 {
		t.Errorf("normalizado = %v", d["recorrente"])
	}
	n := Doc{"recorrente": nil}
	if err := ValidarRecorrente(n); err != nil {
		t.Fatal(err)
	}
	if _, tem := n["recorrente"]; tem {
		t.Error("null remove o campo")
	}
	for _, ruim := range []any{"x", 3.0, map[string]any{}, map[string]any{"intervalo": 0.0}, map[string]any{"intervalo": 121.0}, map[string]any{"intervalo": 1.5}, map[string]any{"intervalo": "2"}, true} {
		if err := ValidarRecorrente(Doc{"recorrente": ruim}); err == nil {
			t.Errorf("%v deveria ser recusado", ruim)
		}
	}
	if err := ValidarRecorrente(Doc{"recorrente": map[string]any{"intervalo": 1.0}, "transferParId": "p"}); err == nil {
		t.Error("transferencia nao e recorrente")
	}
}

func TestIDCopiaRecorrenciaEIdempotenteEValido(t *testing.T) {
	a := IDCopiaRecorrencia("abc123", "2026-03-10T10:00:00.000Z")
	if a != IDCopiaRecorrencia("abc123", "2026-03-10T10:00:00.000Z") {
		t.Error("o id deve ser derivado, igual nas duas rodadas")
	}
	if a == IDCopiaRecorrencia("abc123", "2026-04-10T10:00:00.000Z") || a == IDCopiaRecorrencia("outra1", "2026-03-10T10:00:00.000Z") {
		t.Error("recorrencias e datas diferentes dao ids diferentes")
	}
	if !ReID.MatchString(a) {
		t.Errorf("id fora do formato: %s", a)
	}
}
