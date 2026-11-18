package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Infraestrutura do push (spec app-android-push, R21 e P-T16 a P-T18): segredos fora do git, nenhuma dependencia nova, template por ambiente.

func lerDaRaiz(t *testing.T, caminho string) string {
	t.Helper()
	b, err := os.ReadFile("../" + caminho)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// P-T16: a autenticacao no Firebase e feita a mao com golang-jwt; nenhuma biblioteca nova entra.
func TestPushNaoTrazDependenciaNova(t *testing.T) {
	mod := lerDaRaiz(t, "backend/go.mod")
	for _, proibida := range []string{"golang.org/x/oauth2", "firebase.google.com/go", "google.golang.org/api", "cloud.google.com"} {
		if strings.Contains(mod, proibida) {
			t.Errorf("go.mod traz %s: a spec manda assinar o JWT com golang-jwt, sem biblioteca nova (decisao a parte)", proibida)
		}
	}
}

// P-T17: o .gitignore cobre os dois arquivos, nenhum esta rastreado e o script nao imprime nem recebe o conteudo.
func TestSegredosDoPushFicamForaDoGit(t *testing.T) {
	ign := lerDaRaiz(t, ".gitignore")
	for _, padrao := range []string{"app-android/android/app/google-services.json", "*-adminsdk-*.json"} {
		if !strings.Contains(ign, padrao) {
			t.Errorf(".gitignore sem %q", padrao)
		}
	}
	if out, err := exec.Command("git", "-C", "..", "ls-files").Output(); err == nil {
		for _, arq := range strings.Split(string(out), "\n") {
			if strings.HasSuffix(arq, "google-services.json") || strings.Contains(arq, "-adminsdk-") {
				t.Errorf("%s esta rastreado pelo git", arq)
			}
		}
	} else {
		t.Log("git indisponivel: a checagem dos arquivos rastreados foi pulada")
	}
	sc := lerDaRaiz(t, "scripts/guardar-fcm.ps1")
	for _, quer := range []string{"[ValidateSet('dev', 'prod')]", `"file://$caminho"`, "/financas/$Ambiente/fcm-credencial", "SecureString"} {
		if !strings.Contains(sc, quer) {
			t.Errorf("guardar-fcm.ps1 sem %q", quer)
		}
	}
	for _, proibido := range []string{"[string]$Json", "[string]$Conteudo", "[string]$Chave", "Write-Host $j", "Write-Output $j", "Out-Host", "-Raw | Write"} {
		if strings.Contains(sc, proibido) {
			t.Errorf("guardar-fcm.ps1 nao pode ter %q: o conteudo nunca e impresso nem recebido como parametro", proibido)
		}
	}
}

// P-T18: o template ganha a variavel e a politica de leitura so da credencial do ambiente.
func TestTemplateTemACredencialDoPushPorAmbiente(t *testing.T) {
	tpl := lerDaRaiz(t, "template.yaml")
	for _, quer := range []string{
		"FCM_CREDENCIAL_PARAM: !If [EhDev, /financas/dev/fcm-credencial, /financas/prod/fcm-credencial]",
		"SSMParameterReadPolicy: { ParameterName: !If [EhDev, financas/dev/fcm-credencial, financas/prod/fcm-credencial] }",
	} {
		if strings.Count(tpl, quer) != 1 {
			t.Errorf("template.yaml deveria ter exatamente uma vez: %s", quer)
		}
	}
	if strings.Contains(tpl, "financas/*") || strings.Contains(tpl, "ssm:*") {
		t.Error("a politica de leitura nao pode ser mais larga que o parametro da credencial")
	}
}
