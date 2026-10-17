package main

import (
	"os"
	"testing"
	"time"
)

// Relogio padrao dos testes: 05/02/2026. Nesse dia fevereiro (o mes atual) e janeiro (o anterior, ate o dia 7) estao abertos,
// entao os testes que escrevem em janeiro e fevereiro de 2026 nao esbarram no fechamento por data (.ai/specs/fechamento-automatico).
// Quem testa outra data troca o relogio (relogioFixo, ou relogioEm) e o devolve no fim do teste.
func TestMain(m *testing.M) {
	agora = func() time.Time { return time.Date(2026, 2, 5, 15, 0, 0, 0, time.UTC) }
	os.Exit(m.Run())
}

// relogioEm fixa "agora" (em UTC) so durante o teste.
func relogioEm(t *testing.T, ano int, mes time.Month, dia, hora, min int) {
	t.Helper()
	velho := agora
	agora = func() time.Time { return time.Date(ano, mes, dia, hora, min, 0, 0, time.UTC) }
	t.Cleanup(func() { agora = velho })
}
