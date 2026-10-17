package domain

import (
	"fmt"
	"time"
)

// DiaLimiteMesAnterior e o ultimo dia do mes em que o mes anterior ainda e livre: ate o dia 7 (fuso de Sao Paulo) o mes
// anterior pode ser alterado sem confirmacao; a partir do dia 8 ele fecha. E uma constante (nao uma variavel): ninguem a
// muda em tempo de execucao e este e o unico lugar onde o numero aparece. Veja .ai/specs/fechamento-automatico.
const DiaLimiteMesAnterior = 7

// MesFechadoPorData diz se o mes (AAAA-MM) esta fechado por data no instante dado: o mes atual e os futuros estao abertos, o
// anterior esta aberto ate o dia DiaLimiteMesAnterior, e qualquer outro esta fechado. Funcao pura: o instante vem de quem
// chama (o relogio do servidor), nunca do cliente. Mes invalido nao esta fechado (quem chama so passa meses validos).
func MesFechadoPorData(mes string, agora time.Time) bool {
	if !MesValido(mes) {
		return false
	}
	a := agora.In(FusoApp)
	if mes >= a.Format("2006-01") {
		return false
	}
	// o dia 1 do mes atual menos um mes: AddDate no dia 31 pularia para o mes seguinte
	anterior := time.Date(a.Year(), a.Month(), 1, 0, 0, 0, 0, FusoApp).AddDate(0, -1, 0).Format("2006-01")
	return !(mes == anterior && a.Day() <= DiaLimiteMesAnterior)
}

// ErroMesFechado: o mes do banco esta fechado por data. Diferente do mes conferido, este e confirmavel: com a confirmacao
// explicita a escrita segue.
type ErroMesFechado struct{ Banco, Mes string }

func (e *ErroMesFechado) Error() string {
	return fmt.Sprintf("o mes %s do banco %s ja fechou (so o mes atual e o anterior, ate o dia %d, ficam livres): confirme para alterar",
		e.Mes, e.Banco, DiaLimiteMesAnterior)
}
