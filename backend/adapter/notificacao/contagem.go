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

// FraseDaContagem diz quantos avisos ha em cada situacao, atrasados primeiro, depois os de hoje e os a vencer, separados por " · ".
// O que for zero some; sem nenhum, devolve "".
func FraseDaContagem(c domain.ContagemAvisos) string {
	var partes []string
	if c.Atrasados > 0 {
		partes = append(partes, plural(c.Atrasados, "atrasado", "atrasados"))
	}
	if c.Hoje > 0 {
		if c.Hoje == 1 {
			partes = append(partes, "1 vence hoje")
		} else {
			partes = append(partes, fmt.Sprintf("%d vencem hoje", c.Hoje))
		}
	}
	if c.AVencer > 0 {
		partes = append(partes, fmt.Sprintf("%d a vencer", c.AVencer))
	}
	return strings.Join(partes, " · ")
}
