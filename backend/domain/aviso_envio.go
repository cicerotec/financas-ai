package domain

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Avisos de lancamentos futuros (a validacao do campo "aviso" esta em aviso.go). O lancamento leva um campo opcional
//
//	aviso: { dias: [3, 1, 0], insistir: true }
//
// "dias" = quantos dias antes do vencimento avisar (0 = no dia); "insistir" = avisa no dia e todo dia depois, ate o
// lancamento deixar de ser evento futuro. O aviso so e enviado se o status for de evento futuro (configuracao
// statusFuturo) e o lancamento nao estiver oculto. Este arquivo tem a decisao e o texto, puros: quem le o banco, o
// relogio e o canal de envio e o service.

// MaxAtrasoAviso: depois de tantos dias de atraso o item auxiliar sai e o aviso para de insistir.
const MaxAtrasoAviso = 60

// ErrCanalNaoConfigurado: o canal de envio (Telegram, e-mail...) nao tem o que precisa para enviar (token, parametro).
var ErrCanalNaoConfigurado = errors.New("canal de aviso nao configurado")

// FusoBR: o Brasil nao tem horario de verao desde 2019, entao um fuso fixo basta (e dispensa o banco de fusos na Lambda).
var FusoBR = time.FixedZone("BRT", -3*3600)

// MarcaAmbiente: os dois ambientes usam o mesmo bot e o mesmo chat, entao o dev marca as mensagens com [DEV] para nao
// parecerem avisos reais.
func MarcaAmbiente(ambiente string) string {
	if ambiente == "dev" {
		return "[DEV] "
	}
	return ""
}

// LerAviso le o aviso ja gravado (numeros voltam como float64 do DynamoDB).
func LerAviso(d Doc) (dias []int, insistir bool, ok bool) {
	m, tem := d["aviso"].(map[string]any)
	if !tem {
		return nil, false, false
	}
	if arr, _ := m["dias"].([]any); arr != nil {
		for _, x := range arr {
			if f, isNum := x.(float64); isNum {
				dias = append(dias, int(f))
			}
		}
	}
	insistir, _ = m["insistir"].(bool)
	return dias, insistir, len(dias) > 0 || insistir
}

// DiaBR devolve o dia (AAAA-MM-DD) no fuso de Sao Paulo de uma dataEvento (UTC, como guardada).
func DiaBR(dataEvento string) (string, error) {
	t, err := time.Parse(time.RFC3339Nano, dataEvento)
	if err != nil {
		return "", err
	}
	return t.In(FusoBR).Format("2006-01-02"), nil
}

// DiaParaTempo converte AAAA-MM-DD em meia-noite UTC (so para somar e subtrair dias).
func DiaParaTempo(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.UTC)
}

// KindAviso diz em que situacao o vencimento esta em relacao a hoje.
type KindAviso int

const (
	SemAviso KindAviso = iota
	AvisoAntes
	AvisoHoje
	AvisoAtrasado
)

// DecidirAviso diz se hoje (AAAA-MM-DD) e dia de avisar um vencimento, e quantos dias faltam ou passaram.
func DecidirAviso(hoje, venc string, dias []int, insistir bool) (KindAviso, int) {
	h, err1 := DiaParaTempo(hoje)
	v, err2 := DiaParaTempo(venc)
	if err1 != nil || err2 != nil {
		return SemAviso, 0
	}
	diff := int(math.Round(v.Sub(h).Hours() / 24)) // >0 falta, 0 hoje, <0 atrasado
	switch {
	case diff == 0 && (insistir || contem(dias, 0)):
		return AvisoHoje, 0
	case diff > 0 && contem(dias, diff):
		return AvisoAntes, diff
	case diff < 0 && insistir && -diff <= MaxAtrasoAviso:
		return AvisoAtrasado, -diff
	}
	return SemAviso, 0
}

func contem(l []int, n int) bool {
	for _, x := range l {
		if x == n {
			return true
		}
	}
	return false
}

// ---------- texto ----------

// Moeda formata em reais (R$ 1.234,56).
func Moeda(v float64) string {
	s := strconv.FormatFloat(math.Abs(v), 'f', 2, 64)
	inteiro, frac, _ := strings.Cut(s, ".")
	var partes []string
	for len(inteiro) > 3 {
		partes = append([]string{inteiro[len(inteiro)-3:]}, partes...)
		inteiro = inteiro[:len(inteiro)-3]
	}
	partes = append([]string{inteiro}, partes...)
	r := "R$ " + strings.Join(partes, ".") + "," + frac
	if v < 0 {
		r = "-" + r
	}
	return r
}

func plural(n int, um, varios string) string {
	if n == 1 {
		return "1 " + um
	}
	return fmt.Sprintf("%d %s", n, varios)
}

// DataBR mostra a data com o ano (dd/mm/aaaa): um aviso de "10/01" nao diz de que ano e.
func DataBR(dia string) string {
	t, err := DiaParaTempo(dia)
	if err != nil {
		return dia
	}
	return t.Format("02/01/2006")
}

// LinhaAviso e um lancamento a avisar, ja com a situacao do vencimento.
type LinhaAviso struct {
	Kind      KindAviso
	N         int
	Venc      string
	Tipo      string
	Descricao string
	Valor     float64
	Banco     string
}

// Texto e a linha do resumo ("• Vence em 3 dias (13/10/2026): Condominio - R$ 480,00 (ITAU)").
func (l LinhaAviso) Texto() string {
	saida := l.Tipo != "entrada"
	var quando string
	switch l.Kind {
	case AvisoAtrasado:
		if saida {
			quando = "Atrasado há " + plural(l.N, "dia", "dias")
		} else {
			quando = "Recebimento atrasado há " + plural(l.N, "dia", "dias")
		}
	case AvisoHoje:
		if saida {
			quando = "Vence hoje"
		} else {
			quando = "A receber hoje"
		}
	default:
		if saida {
			quando = "Vence em " + plural(l.N, "dia", "dias")
		} else {
			quando = "A receber em " + plural(l.N, "dia", "dias")
		}
	}
	desc := strings.TrimSpace(l.Descricao)
	if desc == "" {
		desc = "(sem descrição)"
	}
	return fmt.Sprintf("• %s (%s): %s — %s (%s)", quando, DataBR(l.Venc), desc, Moeda(l.Valor), l.Banco)
}

// MontarResumo junta as linhas de uma pessoa: atrasados primeiro, depois hoje, depois os que ainda vao vencer. marca e a
// MarcaAmbiente do servidor.
func MontarResumo(marca, hoje string, linhas []LinhaAviso) string {
	sort.SliceStable(linhas, func(i, j int) bool {
		a, b := linhas[i], linhas[j]
		if a.Kind != b.Kind {
			return a.Kind > b.Kind // atrasado > hoje > antes
		}
		return a.Venc < b.Venc
	})
	partes := []string{"🔔 " + marca + "Avisos de " + DataBR(hoje)}
	for _, l := range linhas {
		partes = append(partes, l.Texto())
	}
	return strings.Join(partes, "\n")
}

// StatusFuturoPadrao: os status que contam como evento futuro quando a configuracao do espaco nao diz.
var StatusFuturoPadrao = map[string]bool{"PREVISTO": true, "TRANSFERINDO": true, "CREDITANDO": true}

// EhStatusFuturo diz se o status e de evento futuro (a configuracao statusFuturo de CFG#LISTAS, ou o padrao).
func EhStatusFuturo(listas Doc, status string) bool {
	if m, ok := listas["statusFuturo"].(map[string]any); ok {
		v, _ := m[status].(bool)
		return v
	}
	return StatusFuturoPadrao[status]
}
