package domain

import (
	"strconv"
	"strings"
)

// ResumoAvisos e o conteudo de um aviso, sem formato: o dia, a marca do ambiente e as linhas (ou a marca de "mensagem de
// teste"). O service entrega isto ao despachante; cada canal escolhe o texto (o Telegram, o resumo com cada linha; o push da
// fase 4 do item 54, so a contagem).
type ResumoAvisos struct {
	Hoje   string // AAAA-MM-DD
	Marca  string // MarcaAmbiente do servidor
	Linhas []LinhaAviso
	Teste  bool // a mensagem de teste do botao "Testar" (sem linhas)
}

// LimiteTelegram e o tamanho, em letras, de cada mensagem do Telegram: abaixo dos 4096 que ele aceita, para sobrar margem
// (o limite deles pode contar em outra unidade que nao a letra).
const LimiteTelegram = 4000

// TextoDeTeste e a mensagem do botao "Testar" dos avisos.
func TextoDeTeste(marca string) string {
	return "🔔 " + marca + "Teste do financas.ai: se você recebeu esta mensagem, os avisos vão chegar aqui."
}

func tamanho(s string) int { return len([]rune(s)) }

// PartesDoResumo devolve o texto do Telegram em uma ou mais mensagens de ate limite letras. Se o resumo cabe, e uma
// mensagem so, identica a MontarResumo. Se nao, cada parte repete o cabecalho com " (i/n)", as linhas seguem a ordem de
// MontarResumo, uma linha nunca e cortada ao meio (uma linha sozinha acima do limite e truncada com "…") e nenhuma se perde.
// A lista de quem chamou nao e reordenada.
func PartesDoResumo(r ResumoAvisos, limite int) []string {
	if r.Teste {
		return []string{TextoDeTeste(r.Marca)}
	}
	linhas := append([]LinhaAviso(nil), r.Linhas...)
	inteiro := MontarResumo(r.Marca, r.Hoje, linhas) // ordena a copia
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
