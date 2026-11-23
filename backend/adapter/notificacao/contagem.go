package notificacao

import (
	"fmt"
	"strings"

	"financas/domain"
)

// Frase da contagem dos avisos, a mesma no push e no Telegram: so inteiros, nunca descricao, valor, meio de pagamento ou data
// (decisao do dono, 2026-11-13 para o push e 2026-11-20 para o Telegram: a mensagem aparece na tela bloqueada e passa por um terceiro).

func plural(n int, um, varios string) string {
	if n == 1 {
		return "1 " + um
	}
	return fmt.Sprintf("%d %s", n, varios)
}

// vencem diz quantos vencem e quando: "1 vence amanhã", "3 vencem nos próximos dias".
func vencem(n int, quando string) string {
	if n == 1 {
		return "1 vence " + quando
	}
	return fmt.Sprintf("%d vencem %s", n, quando)
}

// FraseDaContagem diz quantos avisos ha em cada situacao, atrasados primeiro, depois os de hoje e os a vencer, separados por " · ".
// O que for zero some; sem nenhum, devolve "".
func FraseDaContagem(c domain.ContagemAvisos) string {
	var partes []string
	if c.Atrasados > 0 {
		partes = append(partes, plural(c.Atrasados, "atrasado", "atrasados"))
	}
	if c.Hoje > 0 {
		partes = append(partes, vencem(c.Hoje, "hoje"))
	}
	if c.Amanha > 0 {
		partes = append(partes, vencem(c.Amanha, "amanhã"))
	}
	if c.Proximos > 0 {
		partes = append(partes, vencem(c.Proximos, "nos próximos dias"))
	}
	return strings.Join(partes, " · ")
}
