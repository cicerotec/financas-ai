package domain

import (
	"fmt"
	"time"
	_ "time/tzdata" // fuso de Sao Paulo mesmo onde o sistema nao tem /usr/share/zoneinfo
)

// FusoApp e o fuso do app: o mes de um lancamento e sempre o de Sao Paulo.
var FusoApp = func() *time.Location {
	l, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("BRT", -3*3600)
	}
	return l
}()

// MesDe devolve o mes (AAAA-MM, no fuso do app) de uma dataEvento; "" se a data for invalida.
func MesDe(dataEvento string) string {
	t, err := time.Parse(time.RFC3339Nano, dataEvento)
	if err != nil {
		return ""
	}
	return t.In(FusoApp).Format("2006-01")
}

// Alvo e o par banco e mes que um lancamento afeta no fechamento mensal.
type Alvo struct{ Banco, Mes string }

// AlvoDe devolve o alvo de um lancamento; ok=false se faltar banco ou data valida.
func AlvoDe(d Doc) (Alvo, bool) {
	b, _ := d["banco"].(string)
	s, _ := d["dataEvento"].(string)
	m := MesDe(s)
	return Alvo{b, m}, b != "" && m != ""
}

// ErroMesConferido: o mes do banco foi conferido com o extrato e esta travado. Fechamento conferido e absoluto.
type ErroMesConferido struct{ Banco, Mes, Ate string }

func (e *ErroMesConferido) Error() string {
	return fmt.Sprintf("o mes %s do banco %s esta conferido (ate %s) e travado: reabra o mes na aba Saldos para alterar", e.Mes, e.Banco, e.Ate)
}

// MesAtual devolve o mes (AAAA-MM) do instante dado, no fuso do app.
func MesAtual(agora time.Time) string { return agora.In(FusoApp).Format("2006-01") }

// UltimoConferido devolve o mes mais recente conferido de um item SALDO#<banco>; "" se nenhum.
func UltimoConferido(d Doc) string {
	m, _ := d["conferidos"].(map[string]any)
	u := ""
	for k := range m {
		if k > u {
			u = k
		}
	}
	return u
}

// MesDeInicio devolve o mes da data de inicio do controle de um banco, lida da configuracao de saldos (CFG#SALDOS);
// "" se o banco nao tem data de inicio valida.
func MesDeInicio(cfgSaldos Doc, banco string) string {
	por, _ := cfgSaldos["porBanco"].(map[string]any)
	if b, ok := por[banco].(map[string]any); ok {
		if di, _ := b["dataInicio"].(string); len(di) >= 7 && MesValido(di[:7]) {
			return di[:7]
		}
	}
	return ""
}
