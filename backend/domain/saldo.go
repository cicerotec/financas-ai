package domain

import (
	"fmt"
	"reflect"
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

// CamposSaldo sao os campos que mudam o saldo de um banco; o resto (descricao, nota, tags...) pode ser editado em
// mes conferido.
var CamposSaldo = []string{"valor", "tipo", "status", "banco", "dataEvento"}

// MudaSaldo diz se a edicao de antigo para novo mexe no saldo.
func MudaSaldo(antigo, novo Doc) bool {
	for _, c := range CamposSaldo {
		if !reflect.DeepEqual(antigo[c], novo[c]) {
			return true
		}
	}
	a, _ := antigo["oculto"].(bool)
	n, _ := novo["oculto"].(bool)
	return a != n
}

// ErroMesConferido: o mes do banco foi conferido com o extrato e esta travado. Fechamento conferido e absoluto.
type ErroMesConferido struct{ Banco, Mes, Ate string }

func (e *ErroMesConferido) Error() string {
	return fmt.Sprintf("o mes %s do banco %s esta conferido (ate %s) e travado: reabra o mes na aba Saldos para alterar", e.Mes, e.Banco, e.Ate)
}
