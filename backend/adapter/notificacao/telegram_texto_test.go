package notificacao

import (
	"strings"
	"testing"

	"financas/domain"
)

// linhasCheias traz tudo o que um lancamento tem (descricao, valor, meio de pagamento, vencimento): nada disso pode chegar ao Telegram.
func linhasCheias() []domain.LinhaAviso {
	return []domain.LinhaAviso{
		{Kind: domain.AvisoAtrasado, N: 2, Venc: "2026-10-05", Tipo: "saida", Descricao: "Internet da casa", Valor: 120.5, Banco: "NUBANK"},
		{Kind: domain.AvisoAtrasado, N: 9, Venc: "2026-09-28", Tipo: "saida", Descricao: "Seguro do carro", Valor: 987.65, Banco: "BRADESCO"},
		{Kind: domain.AvisoHoje, Venc: "2026-10-07", Tipo: "entrada", Descricao: "Aluguel recebido", Valor: 2500, Banco: "ITAU"},
		{Kind: domain.AvisoAntes, N: 3, Venc: "2026-10-10", Tipo: "saida", Descricao: "Condominio", Valor: 800, Banco: "ITAU"},
	}
}

// V-T11: o texto e a contagem, e so ela.
func TestTextoDoTelegramEhSoAContagem(t *testing.T) {
	r := domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: linhasCheias()}
	got := TextoDoTelegram(r)
	want := "🔔 Avisos de 07/10/2026: 2 atrasados · 1 vence hoje · 1 a vencer. Abra o app para ver o que é cada um."
	if got != want {
		t.Errorf("texto = %q, esperado %q", got, want)
	}
	for _, proibido := range []string{"Internet", "Seguro", "Aluguel", "Condominio", "NUBANK", "BRADESCO", "ITAU", "120", "987", "2500", "800", "R$",
		"05/10/2026", "28/09/2026", "10/10/2026", "•", "\n"} {
		if strings.Contains(got, proibido) {
			t.Errorf("o texto leva %q: %q", proibido, got)
		}
	}
}

func TestTextoDoTelegramSingularPluralEZeroSome(t *testing.T) {
	casos := map[string][]domain.LinhaAviso{
		"1 atrasado":     {{Kind: domain.AvisoAtrasado}},
		"2 atrasados":    {{Kind: domain.AvisoAtrasado}, {Kind: domain.AvisoAtrasado}},
		"1 vence hoje":   {{Kind: domain.AvisoHoje}},
		"3 vencem hoje":  {{Kind: domain.AvisoHoje}, {Kind: domain.AvisoHoje}, {Kind: domain.AvisoHoje}},
		"1 a vencer":     {{Kind: domain.AvisoAntes}},
		"4 a vencer":     {{Kind: domain.AvisoAntes}, {Kind: domain.AvisoAntes}, {Kind: domain.AvisoAntes}, {Kind: domain.AvisoAntes}},
		"1 vence hoje · 1 a vencer": {{Kind: domain.AvisoAntes}, {Kind: domain.AvisoHoje}},
	}
	for contagem, linhas := range casos {
		got := TextoDoTelegram(domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: linhas})
		want := "🔔 Avisos de 07/10/2026: " + contagem + ". Abra o app para ver o que é cada um."
		if got != want {
			t.Errorf("texto = %q, esperado %q", got, want)
		}
	}
	// o que e zero nao aparece
	got := TextoDoTelegram(domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: []domain.LinhaAviso{{Kind: domain.AvisoHoje}}})
	if strings.Contains(got, "atrasado") || strings.Contains(got, "a vencer") || strings.Contains(got, "0 ") {
		t.Errorf("zero deveria sumir: %q", got)
	}
}

func TestTextoDoTelegramEntradasContamNaSituacaoDoVencimento(t *testing.T) {
	linhas := []domain.LinhaAviso{{Kind: domain.AvisoAtrasado, Tipo: "entrada"}, {Kind: domain.AvisoHoje, Tipo: "entrada"}, {Kind: domain.AvisoAntes, Tipo: "entrada"}}
	got := TextoDoTelegram(domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: linhas})
	if !strings.Contains(got, "1 atrasado · 1 vence hoje · 1 a vencer") {
		t.Errorf("as entradas contam como as saidas: %q", got)
	}
}

func TestTextoDoTelegramMarcaDoDevSoNoDev(t *testing.T) {
	linhas := []domain.LinhaAviso{{Kind: domain.AvisoHoje}}
	if got := TextoDoTelegram(domain.ResumoAvisos{Hoje: "2026-10-07", Marca: domain.MarcaAmbiente("dev"), Linhas: linhas}); !strings.HasPrefix(got, "🔔 [DEV] Avisos de 07/10/2026: ") {
		t.Errorf("o dev leva [DEV]: %q", got)
	}
	for _, amb := range []string{"prod", ""} {
		if got := TextoDoTelegram(domain.ResumoAvisos{Hoje: "2026-10-07", Marca: domain.MarcaAmbiente(amb), Linhas: linhas}); strings.Contains(got, "[DEV]") || !strings.HasPrefix(got, "🔔 Avisos de 07/10/2026: ") {
			t.Errorf("ambiente %q nao leva [DEV]: %q", amb, got)
		}
	}
}

// A parte da contagem e a mesma do push: uma funcao so para os dois canais.
func TestTextoDoTelegramUsaAMesmaContagemDoPush(t *testing.T) {
	r := domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: linhasCheias()}
	_, corpoDoPush := TextoDoPush(domain.Contagens(r))
	if !strings.Contains(TextoDoTelegram(r), corpoDoPush) {
		t.Errorf("o texto do Telegram %q nao leva a contagem do push %q", TextoDoTelegram(r), corpoDoPush)
	}
}

// Uma mensagem so, curta, por maior que seja a lista: nao ha mais divisao em partes nem limite de 4000 letras.
func TestTextoDoTelegramEhUmaMensagemCurtaPorMaiorQueSejaALista(t *testing.T) {
	var linhas []domain.LinhaAviso
	for i := 0; i < 500; i++ {
		linhas = append(linhas, domain.LinhaAviso{Kind: domain.AvisoAntes, N: 3, Venc: "2026-10-10", Tipo: "saida", Descricao: strings.Repeat("x", 80), Valor: 10, Banco: "ITAU"})
	}
	got := TextoDoTelegram(domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: linhas})
	if n := len([]rune(got)); n > 200 || strings.Contains(got, "\n") || strings.Contains(got, "(1/") {
		t.Errorf("deveria ser uma mensagem curta e sem partes (%d letras): %q", n, got)
	}
	if !strings.Contains(got, "500 a vencer") {
		t.Errorf("a contagem some: %q", got)
	}
}

// O texto de teste e o aviso de seguranca (texto fixo) nao mudam.
func TestTextoDoTelegramTesteESegurancaSeguemFixos(t *testing.T) {
	if got := TextoDoTelegram(domain.ResumoAvisos{Marca: "[DEV] ", Teste: true}); !strings.HasPrefix(got, "🔔 [DEV] Teste do financas.ai") {
		t.Errorf("texto de teste = %q", got)
	}
	if got := TextoDoTelegram(domain.ResumoAvisos{Marca: "[DEV] ", Mensagem: domain.MensagemMFADesligado}); got != "🔐 [DEV] "+domain.MensagemMFADesligado {
		t.Errorf("aviso de seguranca = %q", got)
	}
}
