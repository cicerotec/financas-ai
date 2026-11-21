package auth

import "testing"

// V-T18 (spec telegram-vinculo, R25): telegram.vincular e de owner e member, so no proprio vinculo; telegram.webhook nao esta liberada a papel nenhum.
func TestTelegramVincularEDeOwnerEMember(t *testing.T) {
	for _, papel := range []string{PapelOwner, PapelMember} {
		if d := Permitido(papel, AcaoVincularTelegram); d != Liberado {
			t.Errorf("Permitido(%q, %s) = %v, esperado Liberado", papel, AcaoVincularTelegram, d)
		}
	}
	for _, papel := range []string{"", "viewer", "OWNER", "admin", "publico"} {
		if d := Permitido(papel, AcaoVincularTelegram); d != Negado {
			t.Errorf("Permitido(%q, %s) = %v, esperado Negado", papel, AcaoVincularTelegram, d)
		}
	}
	if a, ok := RotaParaAcao("POST", []string{"telegram", "vinculo"}); !ok || a != AcaoVincularTelegram {
		t.Errorf("POST /spaces/{sid}/telegram/vinculo = %q, %v", a, ok)
	}
	for _, c := range []struct {
		m    string
		rest []string
	}{
		{"GET", []string{"telegram", "vinculo"}}, {"PUT", []string{"telegram", "vinculo"}}, {"DELETE", []string{"telegram", "vinculo"}},
		{"POST", []string{"telegram"}}, {"POST", []string{"telegram", "vinculo", "x"}}, {"POST", []string{"Telegram", "vinculo"}},
		{"POST", []string{"telegram", "webhook"}}, // o webhook nao e uma rota de espaco
	} {
		if a, ok := RotaParaAcao(c.m, c.rest); ok {
			t.Errorf("RotaParaAcao(%s, %v) = %q: nao e uma rota", c.m, c.rest, a)
		}
	}
}

func TestTelegramWebhookNaoEstaLiberadoParaNenhumPapel(t *testing.T) {
	for _, papel := range []string{PapelOwner, PapelMember, "", "admin", "publico"} {
		if d := Permitido(papel, AcaoWebhookTelegram); d != Negado {
			t.Errorf("Permitido(%q, %s) = %v, esperado Negado: a rota publica so existe por RotaPublica", papel, AcaoWebhookTelegram, d)
		}
	}
	if a, ok := RotaSemEspaco("POST", []string{"telegram", "webhook"}); ok {
		t.Errorf("o webhook nao e uma rota autenticada sem espaco, veio %q", a)
	}
}

// V-T4: RotaPublica devolve true SO para POST [mfa recuperar] e POST [telegram webhook].
func TestRotaPublicaTelegramSoMetodoECaminhoExatos(t *testing.T) {
	if a, ok := RotaPublica("POST", []string{"telegram", "webhook"}); !ok || a != AcaoWebhookTelegram {
		t.Fatalf("POST /telegram/webhook deve ser publica com a acao %q, veio %q %v", AcaoWebhookTelegram, a, ok)
	}
	if a, ok := RotaPublica("POST", []string{"mfa", "recuperar"}); !ok || a != AcaoRecuperarMFA {
		t.Fatalf("POST /mfa/recuperar continua publica, veio %q %v", a, ok)
	}
	nao := []struct {
		m string
		p []string
	}{
		{"GET", []string{"telegram", "webhook"}},
		{"PUT", []string{"telegram", "webhook"}},
		{"DELETE", []string{"telegram", "webhook"}},
		{"post", []string{"telegram", "webhook"}},
		{"POST", []string{"telegram", "webhook", "x"}},
		{"POST", []string{"telegram"}},
		{"POST", []string{"telegram", "vinculo"}},
		{"POST", []string{"Telegram", "webhook"}},
		{"POST", []string{"telegram", "Webhook"}},
		{"POST", []string{"telegram", "", "webhook"}},
		{"POST", []string{"", "telegram", "webhook"}},
		{"POST", []string{"spaces", "s1", "telegram", "webhook"}},
		{"POST", []string{"spaces", "s1", "telegram", "vinculo"}},
		{"POST", []string{"webhook"}},
		{"POST", []string{}},
		{"POST", nil},
	}
	for _, c := range nao {
		if a, ok := RotaPublica(c.m, c.p); ok || a != "" {
			t.Errorf("RotaPublica(%q, %v) = %q, %v; so POST [mfa recuperar] e POST [telegram webhook] sao publicas", c.m, c.p, a, ok)
		}
	}
}
