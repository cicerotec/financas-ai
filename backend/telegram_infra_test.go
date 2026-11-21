package main

import (
	"strings"
	"testing"
)

// V-T20 (spec telegram-vinculo, R23 e R24): o template ganha so as duas variaveis e a politica de leitura so desses dois parametros, por
// ambiente; os scripts do dono nao imprimem segredo, nao aceitam o token na linha de comando, recusam ambiente invalido e o de registro recusa um
// bot que ja tem outro webhook. A IA nunca roda os scripts: este teste so le o texto deles.

func TestTemplateTemOSegredoEOUsuarioDoBotPorAmbiente(t *testing.T) {
	tpl := lerDaRaiz(t, "template.yaml")
	for _, quer := range []string{
		"TELEGRAM_WEBHOOK_SECRET_PARAM: !If [EhDev, /financas/dev/telegram-webhook-secret, /financas/prod/telegram-webhook-secret]",
		"TELEGRAM_BOT_USUARIO_PARAM: !If [EhDev, /financas/dev/telegram-bot-usuario, /financas/prod/telegram-bot-usuario]",
		"SSMParameterReadPolicy: { ParameterName: !If [EhDev, financas/dev/telegram-webhook-secret, financas/prod/telegram-webhook-secret] }",
		"SSMParameterReadPolicy: { ParameterName: !If [EhDev, financas/dev/telegram-bot-usuario, financas/prod/telegram-bot-usuario] }",
		// o que ja existia segue igual
		"TELEGRAM_TOKEN_PARAM: !If [EhDev, /financas/dev/telegram-token, /financas/prod/telegram-token]",
		"SSMParameterReadPolicy: { ParameterName: !If [EhDev, financas/dev/telegram-token, financas/prod/telegram-token] }",
		"FCM_CREDENCIAL_PARAM: !If [EhDev, /financas/dev/fcm-credencial, /financas/prod/fcm-credencial]",
	} {
		if strings.Count(tpl, quer) != 1 {
			t.Errorf("template.yaml deveria ter exatamente uma vez: %s", quer)
		}
	}
	if strings.Contains(tpl, "financas/*") || strings.Contains(tpl, "ssm:*") || strings.Contains(tpl, "telegram-*") {
		t.Error("a politica de leitura nao pode ser mais larga que os parametros do Telegram")
	}
	// nenhuma regra nova do Events: so as duas de sempre (avisos diarios e combos semanais)
	if n := strings.Count(tpl, "Type: Schedule"); n != 2 {
		t.Errorf("o template tem %d regras de agendamento, esperadas as 2 de sempre", n)
	}
}

func TestScriptsDoTelegramNaoImprimemSegredoNemAceitamOTokenNaLinhaDeComando(t *testing.T) {
	guardar := lerDaRaiz(t, "scripts/guardar-telegram.ps1")
	registrar := lerDaRaiz(t, "scripts/registrar-webhook-telegram.ps1")
	for nome, sc := range map[string]string{"guardar-telegram.ps1": guardar, "registrar-webhook-telegram.ps1": registrar} {
		if !strings.Contains(sc, "[ValidateSet('dev', 'prod')]") {
			t.Errorf("%s nao recusa um ambiente que nao seja dev ou prod", nome)
		}
		for _, proibido := range []string{"[string]$Token", "[string]$Segredo", "[securestring]$Token", "Write-Host $token", "Write-Host $segredo", "Write-Output $token",
			"Write-Output $segredo", "Write-Host \"$token", "Write-Host \"$segredo", "Out-Host", "$token | Write"} {
			if strings.Contains(sc, proibido) {
				t.Errorf("%s nao pode ter %q: token e segredo nunca sao impressos nem recebidos como parametro", nome, proibido)
			}
		}
		if !strings.Contains(sc, "SecureString") || !strings.Contains(sc, `"file://`) {
			t.Errorf("%s deveria gravar como SecureString por um arquivo (file://), sem o valor na linha de comando", nome)
		}
	}
	for _, quer := range []string{"Read-Host", "-AsSecureString", "/financas/$Ambiente/telegram-token"} {
		if !strings.Contains(guardar, quer) {
			t.Errorf("guardar-telegram.ps1 sem %q", quer)
		}
	}
	for _, quer := range []string{"getMe", "getWebhookInfo", "setWebhook", `allowed_updates = @('message')`, "-Forcar", "RandomNumberGenerator",
		"/financas/$Ambiente/telegram-webhook-secret", "/financas/$Ambiente/telegram-bot-usuario"} {
		if !strings.Contains(registrar, quer) {
			t.Errorf("registrar-webhook-telegram.ps1 sem %q", quer)
		}
	}
	// o webhook ja registrado em outra URL so cede com -Forcar, e o usuario do bot e gravado por ultimo, depois do segredo
	if !strings.Contains(registrar, "$atual -ne $urlDoWebhook -and -not $Forcar") {
		t.Error("o script de registro tem de recusar um bot que ja tem outro webhook, sem -Forcar")
	}
	if i, j := strings.Index(registrar, "telegram-webhook-secret\" 'SecureString'"), strings.Index(registrar, "telegram-bot-usuario\" 'String'"); i < 0 || j < 0 || i > j {
		t.Error("o usuario do bot deve ser gravado por ultimo, depois do segredo")
	}
	if i, j := strings.Index(registrar, "'setWebhook'"), strings.Index(registrar, "telegram-webhook-secret\" 'SecureString'"); i < 0 || j < 0 || i > j {
		t.Error("o segredo so e gravado depois de o setWebhook responder ok")
	}
}
