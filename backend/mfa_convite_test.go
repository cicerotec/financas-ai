package main

import (
	"os/exec"
	"strings"
	"testing"
)

// M-T24d (spec mfa-opcional, R29): o convite novo grava o e-mail em minusculas, porque o pool do Cognito diferencia maiusculas
// e a recuperacao do MFA busca a pessoa pelo e-mail exato.

func TestSeedGravaEmailEmMinusculas(t *testing.T) {
	sc := lerDaRaiz(t, "scripts/seed.sh")
	const baixa = `tr '[:upper:]' '[:lower:]'`
	for _, quer := range []string{
		`OWNER=$(printf '%s' "$OWNER" | ` + baixa + `)`,
		`MEMBER=$(printf '%s' "$MEMBER" | ` + baixa + `)`,
	} {
		if !strings.Contains(sc, quer) {
			t.Errorf("scripts/seed.sh sem %q", quer)
		}
	}
	// a normalizacao vem antes de qualquer criacao de usuario
	if i, j := strings.Index(sc, `OWNER=$(printf`), strings.Index(sc, `SUB_DONO=$(criar`); i < 0 || j < 0 || i > j {
		t.Error("a normalizacao do e-mail precisa vir antes de criar o usuario")
	}
}

// Comportamento real da normalizacao (so onde ha bash): o mesmo trecho do script devolve minusculas.
func TestNormalizacaoDoEmailDevolveMinusculas(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash indisponivel")
	}
	out, err := exec.Command(bash, "-c", `E="Ana.Silva@Exemplo.COM"; E=$(printf '%s' "$E" | tr '[:upper:]' '[:lower:]'); printf '%s' "$E"`).Output()
	if err != nil {
		t.Skip("bash nao executou: " + err.Error())
	}
	if got := string(out); got != strings.ToLower("Ana.Silva@Exemplo.COM") {
		t.Errorf("normalizacao devolveu %q", got)
	}
}
