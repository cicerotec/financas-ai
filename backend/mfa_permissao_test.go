package main

import (
	"regexp"
	"strings"
	"testing"
)

// M-T30 (spec mfa-opcional, R32): a permissao nova da Lambda no Cognito e SO AdminGetUser e AdminSetUserMFAPreference, sobre o proprio pool.
func TestLambdaTemSoDuasAcoesDoCognitoNoProprioPool(t *testing.T) {
	tpl := lerDaRaiz(t, "template.yaml")
	ini := strings.Index(tpl, "  ApiFunction:")
	if ini < 0 {
		t.Fatal("nao achei o recurso ApiFunction no template.yaml")
	}
	fim := strings.Index(tpl[ini:], "\n  # ----")
	if fim < 0 {
		t.Fatal("nao achei o fim do recurso ApiFunction")
	}
	fn := tpl[ini : ini+fim]
	acoes := regexp.MustCompile(`cognito-idp:[A-Za-z*]+`).FindAllString(fn, -1)
	quer := map[string]bool{"cognito-idp:AdminGetUser": true, "cognito-idp:AdminSetUserMFAPreference": true}
	if len(acoes) != 2 {
		t.Fatalf("a Lambda deve ter exatamente 2 acoes do Cognito, tem %v", acoes)
	}
	for _, a := range acoes {
		if !quer[a] {
			t.Errorf("acao do Cognito inesperada na Lambda: %s", a)
		}
	}
	bloco := regexp.MustCompile(`(?s)Sid: RecuperacaoMFA.*?Resource: ([^\n]+)`).FindStringSubmatch(fn)
	if bloco == nil || strings.TrimSpace(bloco[1]) != "!GetAtt UserPool.Arn" {
		t.Errorf("a permissao precisa valer so no proprio pool (Resource: !GetAtt UserPool.Arn), veio %v", bloco)
	}
	if strings.Contains(fn, "cognito-idp:*") || strings.Contains(fn, "Resource: '*'") || strings.Contains(fn, `Resource: "*"`) {
		t.Error("nenhum coringa e permitido na policy da Lambda")
	}
}
