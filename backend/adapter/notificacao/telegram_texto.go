package notificacao

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"financas/domain"
)

// Texto, limite e divisao da mensagem do Telegram. Tudo aqui e formato do canal: o domain so guarda o conteudo
// (domain.ResumoAvisos) e a ordem das linhas (domain.OrdenarLinhas).

// LimiteTelegram e o tamanho, em letras, de cada mensagem do Telegram: abaixo dos 4096 que ele aceita, para sobrar margem
// (o limite deles pode contar em outra unidade que nao a letra).
const LimiteTelegram = 4000

// TextoDeTeste e a mensagem do botao "Testar" dos avisos.
func TextoDeTeste(marca string) string {
	return "🔔 " + marca + "Teste do financas.ai: se você recebeu esta mensagem, os avisos vão chegar aqui."
}

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

// dataBR mostra a data com o ano (dd/mm/aaaa): um aviso de "10/01" nao diz de que ano e.
func dataBR(dia string) string {
	t, err := domain.DiaParaTempo(dia)
	if err != nil {
		return dia
	}
	return t.Format("02/01/2006")
}

// textoDaLinha e a linha do resumo ("• Vence em 3 dias (13/10/2026): Condominio - R$ 480,00 (ITAU)").
func textoDaLinha(l domain.LinhaAviso) string {
	saida := l.Tipo != "entrada"
	var quando string
	switch l.Kind {
	case domain.AvisoAtrasado:
		if saida {
			quando = "Atrasado há " + plural(l.N, "dia", "dias")
		} else {
			quando = "Recebimento atrasado há " + plural(l.N, "dia", "dias")
		}
	case domain.AvisoHoje:
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
	return fmt.Sprintf("• %s (%s): %s — %s (%s)", quando, dataBR(l.Venc), desc, Moeda(l.Valor), l.Banco)
}

// MontarResumo junta as linhas de uma pessoa: atrasados primeiro, depois hoje, depois os que ainda vao vencer. marca e a
// domain.MarcaAmbiente do servidor. A lista de quem chamou nao e reordenada.
func MontarResumo(marca, hoje string, linhas []domain.LinhaAviso) string {
	partes := []string{"🔔 " + marca + "Avisos de " + dataBR(hoje)}
	for _, l := range domain.OrdenarLinhas(linhas) {
		partes = append(partes, textoDaLinha(l))
	}
	return strings.Join(partes, "\n")
}

func tamanho(s string) int { return len([]rune(s)) }

// PartesDoResumo devolve o texto do Telegram em uma ou mais mensagens de ate limite letras. Se o resumo cabe, e uma
// mensagem so, identica a MontarResumo. Se nao, cada parte repete o cabecalho com " (i/n)", as linhas seguem a ordem de
// MontarResumo, uma linha nunca e cortada ao meio (uma linha sozinha acima do limite e truncada com "…") e nenhuma se perde.
// A lista de quem chamou nao e reordenada.
func PartesDoResumo(r domain.ResumoAvisos, limite int) []string {
	if r.Teste {
		return []string{TextoDeTeste(r.Marca)}
	}
	inteiro := MontarResumo(r.Marca, r.Hoje, r.Linhas)
	if tamanho(inteiro) <= limite {
		return []string{inteiro}
	}
	corpo := strings.Split(inteiro, "\n")
	base, textos := corpo[0], corpo[1:]

	digitos := 1
	for {
		reserva := tamanho(" (" + strings.Repeat("9", digitos) + "/" + strings.Repeat("9", digitos) + ")")
		disponivel := limite - tamanho(base) - reserva // o que sobra para as linhas de uma parte (cada uma com seu "\n")
		var blocos [][]string
		var atual []string
		usado := 0
		for _, t := range textos {
			if max := disponivel - 1; tamanho(t) > max {
				t = string([]rune(t)[:max-1]) + "…"
			}
			if usado+1+tamanho(t) > disponivel && len(atual) > 0 {
				blocos = append(blocos, atual)
				atual, usado = nil, 0
			}
			atual = append(atual, t)
			usado += 1 + tamanho(t)
		}
		if len(atual) > 0 {
			blocos = append(blocos, atual)
		}
		if n := len(strconv.Itoa(len(blocos))); n > digitos {
			digitos = n
			continue
		}
		partes := make([]string, len(blocos))
		for i, b := range blocos {
			cab := base
			if len(blocos) > 1 {
				cab += " (" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(blocos)) + ")"
			}
			partes[i] = cab + "\n" + strings.Join(b, "\n")
		}
		return partes
	}
}
