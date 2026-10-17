package main

import (
	"os"
	"testing"
	"time"

	"financas/utils/relogio"
)

// relogioTeste e o relogio de todos os cenarios: um relogio.Fixo compartilhado com o Servidor montado em cenario().
// Padrao: 05/02/2026. Nesse dia fevereiro (o mes atual) e janeiro (o anterior, ate o dia 7) estao abertos, entao os testes
// que escrevem em janeiro e fevereiro de 2026 nao esbarram no fechamento por data (.ai/specs/fechamento-automatico).
// Quem testa outra data usa relogioEm, que a devolve ao padrao no fim do teste.
var relogioTeste = relogio.NewFixo(dataPadraoDosTestes)

var dataPadraoDosTestes = time.Date(2026, 2, 5, 15, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	relogioTeste.Definir(dataPadraoDosTestes)
	os.Exit(m.Run())
}

// relogioEm fixa "agora" (em UTC) so durante o teste.
func relogioEm(t *testing.T, ano int, mes time.Month, dia, hora, min int) {
	t.Helper()
	antes := relogioTeste.Definir(time.Date(ano, mes, dia, hora, min, 0, 0, time.UTC))
	t.Cleanup(func() { relogioTeste.Definir(antes) })
}
