package notificacao

import (
	"strings"
	"testing"

	"financas/domain"
)

// Aviso de seguranca do MFA (spec mfa-opcional, R35): cada canal mostra o texto fixo, sem nenhum dado.

func TestTelegramMostraSoAFraseDoAvisoDeSeguranca(t *testing.T) {
	texto := TextoDoTelegram(domain.ResumoAvisos{Marca: "[DEV] ", Mensagem: domain.MensagemMFADesligado})
	if texto != "🔐 [DEV] "+domain.MensagemMFADesligado {
		t.Errorf("texto = %q", texto)
	}
	texto = TextoDoTelegram(domain.ResumoAvisos{Mensagem: domain.MensagemListaGerada})
	if !strings.HasSuffix(texto, domain.MensagemListaGerada) {
		t.Errorf("texto = %q", texto)
	}
}

func TestPushMostraSoAFraseDoAvisoDeSeguranca(t *testing.T) {
	titulo, corpo := textoDaNotificacao(domain.ResumoAvisos{Marca: "[DEV] ", Mensagem: domain.MensagemListaGerada})
	if titulo != "[DEV] Finanças" || corpo != domain.MensagemListaGerada {
		t.Errorf("titulo=%q corpo=%q", titulo, corpo)
	}
	// um resumo comum continua so com as contagens
	_, corpo = textoDaNotificacao(domain.ResumoAvisos{Linhas: []domain.LinhaAviso{{Kind: domain.AvisoHoje, Descricao: "Segredo", Valor: 10}}})
	if strings.Contains(corpo, "Segredo") || !strings.Contains(corpo, "1 vence hoje") {
		t.Errorf("o resumo comum mudou: %q", corpo)
	}
}
