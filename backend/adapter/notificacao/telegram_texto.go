package notificacao

import (
	"financas/domain"
)

// Texto da mensagem do Telegram. E formato do canal: o domain so guarda o conteudo (domain.ResumoAvisos) e a contagem
// (domain.Contagens), que e a unica coisa do resumo que este arquivo le.

// TextoDeTeste e a mensagem do botao "Testar" dos avisos.
func TextoDeTeste(marca string) string {
	return "🔔 " + marca + "Teste do financas.ai: se você recebeu esta mensagem, os avisos vão chegar aqui."
}

// dataBR mostra a data com o ano (dd/mm/aaaa): um aviso de "10/01" nao diz de que ano e.
func dataBR(dia string) string {
	t, err := domain.DiaParaTempo(dia)
	if err != nil {
		return dia
	}
	return t.Format("02/01/2006")
}

// TextoDoTelegram e a mensagem de um resumo, numa so: a contagem ("2 atrasados · 1 vence hoje · 3 a vencer") e o convite para abrir o
// app. Descricao, valor, meio de pagamento, data de vencimento e nome de espaco ou de pessoa nunca entram, mesmo que as linhas os tenham
// (spec telegram-vinculo, R11). O aviso de seguranca e a mensagem de teste seguem com o texto fixo de sempre.
func TextoDoTelegram(r domain.ResumoAvisos) string {
	if r.Mensagem != "" { // aviso de seguranca (texto fixo, sem linhas)
		return "🔐 " + r.Marca + r.Mensagem
	}
	if r.Teste {
		return TextoDeTeste(r.Marca)
	}
	frase := FraseDaContagem(domain.Contagens(r))
	if frase == "" {
		frase = "você tem avisos"
	}
	return "🔔 " + r.Marca + "Avisos de " + dataBR(r.Hoje) + ": " + frase + ". Abra o app para ver o que é cada um."
}
