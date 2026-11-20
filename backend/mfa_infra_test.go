package main

import (
	"regexp"
	"strings"
	"testing"
)

// Infraestrutura do MFA (spec mfa-opcional, Fase 2): o template e as roles do Actions.

func bloco(t *testing.T, tpl, recurso string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^  ` + recurso + `:\n(?:(?:    .*|)\n)+`)
	m := re.FindString(tpl)
	if m == "" {
		t.Fatalf("recurso %s nao encontrado no template.yaml", recurso)
	}
	return m
}

// M-T1 e M-T20 (R1, R6): MFA opcional so por aplicativo autenticador; nunca ON, nunca SMS.
func TestPoolTemMfaOpcionalSoPorTOTP(t *testing.T) {
	pool := bloco(t, lerDaRaiz(t, "template.yaml"), "UserPool")
	for _, quer := range []string{"MfaConfiguration: OPTIONAL", "EnabledMfas: [SOFTWARE_TOKEN_MFA]"} {
		if !strings.Contains(pool, quer) {
			t.Errorf("UserPool sem %q", quer)
		}
	}
	for _, proibido := range []string{"MfaConfiguration: ON", "SMS_MFA", "EMAIL_OTP", "SmsConfiguration"} {
		if strings.Contains(pool, proibido) {
			t.Errorf("UserPool nao pode ter %q", proibido)
		}
	}
}

// M-T1a (R2a): Managed login v2 com estilo de marca ligado ao cliente; o nome do dominio nao muda.
func TestDominioUsaManagedLogin(t *testing.T) {
	tpl := lerDaRaiz(t, "template.yaml")
	dom := bloco(t, tpl, "UserPoolDomain")
	for _, quer := range []string{"ManagedLoginVersion: 2", "Domain: !Ref CognitoDomainPrefix"} {
		if !strings.Contains(dom, quer) {
			t.Errorf("UserPoolDomain sem %q", quer)
		}
	}
	marca := bloco(t, tpl, "UserPoolBranding")
	for _, quer := range []string{"Type: AWS::Cognito::ManagedLoginBranding", "UserPoolId: !Ref UserPool", "ClientId: !Ref UserPoolClient"} {
		if !strings.Contains(marca, quer) {
			t.Errorf("UserPoolBranding sem %q", quer)
		}
	}
}

// M-T2 (R2): o cliente permite o escopo novo, nao ganha fluxo de senha e nao perde as URLs de retorno.
func TestClientePermiteEscopoSemFluxoDeSenha(t *testing.T) {
	cli := bloco(t, lerDaRaiz(t, "template.yaml"), "UserPoolClient")
	for _, quer := range []string{
		"AllowedOAuthScopes: [openid, email, aws.cognito.signin.user.admin]",
		"ExplicitAuthFlows: [ALLOW_REFRESH_TOKEN_AUTH]",
		"CallbackURLs:", "LogoutURLs:", "AllowedOAuthFlows: [code]", "GenerateSecret: false",
	} {
		if !strings.Contains(cli, quer) {
			t.Errorf("UserPoolClient sem %q", quer)
		}
	}
	if strings.Contains(cli, "ALLOW_USER_PASSWORD_AUTH") || strings.Contains(cli, "ALLOW_USER_SRP_AUTH") {
		t.Error("UserPoolClient nao pode ter fluxo de senha")
	}
}

// R5: a role do Actions de cada ambiente precisa das acoes novas do Cognito antes do merge.
func TestRolesDoActionsTemAcoesDoManagedLogin(t *testing.T) {
	oidc := lerDaRaiz(t, "infra/github-oidc.yaml")
	for _, acao := range []string{
		"cognito-idp:SetUserPoolMfaConfig", "cognito-idp:UpdateUserPoolDomain",
		"cognito-idp:CreateManagedLoginBranding", "cognito-idp:UpdateManagedLoginBranding",
		"cognito-idp:DeleteManagedLoginBranding", "cognito-idp:DescribeManagedLoginBranding",
		"cognito-idp:DescribeManagedLoginBrandingByClient",
	} {
		if !strings.Contains(oidc, acao) {
			t.Errorf("infra/github-oidc.yaml sem %s", acao)
		}
	}
}

// R3: o changeset que substitui ou remove pool, cliente ou dominio de login e barrado.
func TestChangesetBarraSubstituicaoDoLogin(t *testing.T) {
	sc := lerDaRaiz(t, "scripts/verificar-changeset.sh")
	for _, tipo := range []string{"AWS::Cognito::UserPool", "AWS::Cognito::UserPoolClient", "AWS::Cognito::UserPoolDomain"} {
		if !strings.Contains(sc, `"`+tipo+`"`) {
			t.Errorf("verificar-changeset.sh nao barra %s", tipo)
		}
	}
}
