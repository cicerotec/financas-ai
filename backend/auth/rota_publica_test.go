package auth

import "testing"

// M-T27a (spec mfa-opcional): RotaPublica devolve true SO para POST ["mfa","recuperar"]; a acao nao esta liberada para nenhum papel.
func TestRotaPublicaSoMetodoECaminhoExatos(t *testing.T) {
	if a, ok := RotaPublica("POST", []string{"mfa", "recuperar"}); !ok || a != AcaoRecuperarMFA {
		t.Fatalf("POST /mfa/recuperar deve ser publica com a acao %q, veio %q %v", AcaoRecuperarMFA, a, ok)
	}
	nao := []struct {
		m string
		p []string
	}{
		{"GET", []string{"mfa", "recuperar"}},
		{"PUT", []string{"mfa", "recuperar"}},
		{"DELETE", []string{"mfa", "recuperar"}},
		{"post", []string{"mfa", "recuperar"}}, // caixa do metodo
		{"POST", []string{"mfa", "recuperar", "x"}},
		{"POST", []string{"mfa"}},
		{"POST", []string{"me"}},
		{"POST", []string{"push", "sair"}},
		{"POST", []string{"spaces", "s1", "tx"}},
		{"POST", []string{"MFA", "recuperar"}}, // caixa do caminho
		{"POST", []string{"mfa", "Recuperar"}},
		{"POST", []string{"mfa", "", "recuperar"}},
		{"POST", []string{"", "mfa", "recuperar"}},
		{"POST", []string{}},
		{"POST", nil},
	}
	for _, c := range nao {
		if a, ok := RotaPublica(c.m, c.p); ok || a != "" {
			t.Errorf("RotaPublica(%q, %v) = %q, %v; so POST [mfa recuperar] e publica", c.m, c.p, a, ok)
		}
	}
}

// A acao publica nao passa por papel: nenhum papel (nem um desconhecido) a tem liberada em Permitido.
func TestAcaoRecuperarMFANaoEstaLiberadaParaNenhumPapel(t *testing.T) {
	for _, papel := range []string{PapelOwner, PapelMember, "", "admin", "publico"} {
		if d := Permitido(papel, AcaoRecuperarMFA); d != Negado {
			t.Errorf("Permitido(%q, %s) = %v, esperado Negado: a rota publica so existe por RotaPublica", papel, AcaoRecuperarMFA, d)
		}
	}
	if a, ok := RotaSemEspaco("POST", []string{"mfa", "recuperar"}); ok {
		t.Errorf("a recuperacao nao e uma rota autenticada sem espaco, veio %q", a)
	}
	if a, ok := RotaParaAcao("POST", []string{"mfa", "recuperar"}); ok {
		t.Errorf("a recuperacao nao e uma rota de espaco, veio %q", a)
	}
}
