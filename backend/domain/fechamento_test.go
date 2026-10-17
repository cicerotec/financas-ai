package domain

import (
	"strings"
	"testing"
	"time"
)

// A-T1: o mes fechado por data (spec .ai/specs/fechamento-automatico, R2).
func TestMesFechadoPorData(t *testing.T) {
	utc := func(s string) time.Time {
		x, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	casos := []struct {
		agora string
		mes   string
		quer  bool
		nota  string
	}{
		// dia 7 as 23h59 em Sao Paulo (02h59 de 08/10 em UTC): setembro ainda e livre
		{"2026-10-08T02:59:00Z", "2026-10", false, "mes atual"},
		{"2026-10-08T02:59:00Z", "2026-09", false, "anterior, dia 7 as 23h59 em SP"},
		{"2026-10-08T02:59:00Z", "2026-11", false, "futuro"},
		{"2026-10-08T02:59:00Z", "2026-08", true, "mais antigo que o anterior"},
		// dia 8 as 0h em Sao Paulo (03h00 UTC): setembro fecha
		{"2026-10-08T03:00:00Z", "2026-09", true, "anterior, dia 8 as 0h em SP"},
		{"2026-10-08T03:00:00Z", "2026-10", false, "mes atual segue livre"},
		// o instante em UTC nao decide: 08/10 02h30 UTC ainda e dia 7 em SP
		{"2026-10-08T02:30:00Z", "2026-09", false, "UTC ja diz dia 8, SP diz dia 7"},
		// o primeiro dia do mes
		{"2026-10-01T03:00:00Z", "2026-09", false, "dia 1: o anterior esta aberto"},
		{"2026-10-01T03:00:00Z", "2026-08", true, "dia 1: dois meses atras fechado"},
		// virada de ano
		{"2027-01-05T15:00:00Z", "2026-12", false, "dezembro em 05/01"},
		{"2027-01-05T15:00:00Z", "2026-11", true, "novembro em 05/01"},
		{"2027-01-08T15:00:00Z", "2026-12", true, "dezembro em 08/01"},
		// dia 31: o mes anterior nao pode "pular" por causa do AddDate
		{"2026-03-31T15:00:00Z", "2026-02", true, "fevereiro em 31/03"},
		{"2026-03-31T15:00:00Z", "2026-03", false, "marco em 31/03"},
		{"2026-03-05T15:00:00Z", "2026-02", false, "fevereiro em 05/03"},
		// entrada invalida nao fecha nada
		{"2026-10-20T15:00:00Z", "", false, "vazio"},
		{"2026-10-20T15:00:00Z", "lixo", false, "texto"},
		{"2026-10-20T15:00:00Z", "2026-9", false, "formato errado"},
	}
	for _, c := range casos {
		if got := MesFechadoPorData(c.mes, utc(c.agora)); got != c.quer {
			t.Errorf("MesFechadoPorData(%q, %s) = %v, esperado %v (%s)", c.mes, c.agora, got, c.quer, c.nota)
		}
	}
}

func TestDiaLimiteEUmaConstante(t *testing.T) {
	// "dia 7" e o unico numero: a mensagem do erro o usa, entao mudar a constante muda o texto junto
	if !strings.Contains((&ErroMesFechado{Banco: "ITAU", Mes: "2026-08"}).Error(), "ate o dia 7") || DiaLimiteMesAnterior != 7 {
		t.Errorf("a mensagem deve citar o dia limite: %q", (&ErroMesFechado{Banco: "ITAU", Mes: "2026-08"}).Error())
	}
	msg := (&ErroMesFechado{Banco: "ITAU", Mes: "2026-08"}).Error()
	for _, parte := range []string{"2026-08", "ITAU", "confirme"} {
		if !strings.Contains(msg, parte) {
			t.Errorf("mensagem sem %q: %s", parte, msg)
		}
	}
}
