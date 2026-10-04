package main

import (
	"strings"
	"testing"
)

func TestNormalizarApelido(t *testing.T) {
	ok := []struct{ in, quer string }{
		{"Maria", "Maria"},
		{"  Maria   da  Silva ", "Maria da Silva"},
		{"José André", "José André"},
		{"", ""},
		{"   ", ""},
		{"Zé\tCarlos\n", "Zé Carlos"}, // tab e quebra de linha viram espaco
		{"<b>Eu</b>", "<b>Eu</b>"},    // a tela escapa o HTML; o servidor so guarda texto
		{strings.Repeat("a", 30), strings.Repeat("a", 30)},
		{strings.Repeat("é", 30), strings.Repeat("é", 30)}, // conta caracteres, nao bytes
	}
	for _, c := range ok {
		got, err := normalizarApelido(c.in)
		if err != nil || got != c.quer {
			t.Errorf("normalizarApelido(%q) = (%q, %v), esperado %q", c.in, got, err, c.quer)
		}
	}
	for _, in := range []string{strings.Repeat("a", 31), "Ana\x00", "Ana\x07"} {
		if _, err := normalizarApelido(in); err == nil {
			t.Errorf("normalizarApelido(%q) deveria recusar", in)
		}
	}
}

func apelidoNoBanco(t *testing.T, f *bancoFake, sub string) any {
	d := f.ler(t, "USER#"+sub, "SPACE#s1")
	if d == nil {
		t.Fatalf("vinculo de %s sumiu", sub)
	}
	return d["apelido"]
}

func TestCadaPessoaDefineOProprioNome(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"apelido":"  Maria  "}`)
	esperar(t, "member define o nome", c, 200)
	if d["apelido"] != "Maria" {
		t.Errorf("resposta = %v", d["apelido"])
	}
	if apelidoNoBanco(t, f, "mem") != "Maria" {
		t.Error("o nome deveria estar no vinculo do member")
	}
	if apelidoNoBanco(t, f, "own") != nil {
		t.Error("o vinculo do owner nao pode mudar quando o member define o nome dele")
	}
	// o papel nao e tocado
	if f.ler(t, "USER#mem", "SPACE#s1")["role"] != "member" {
		t.Error("o papel do member deve continuar member")
	}
	c, _ = chamar(t, "own", "PUT", "spaces/s1/perfil", nil, `{"apelido":"Cícero"}`)
	esperar(t, "owner define o nome", c, 200)
	if apelidoNoBanco(t, f, "own") != "Cícero" || apelidoNoBanco(t, f, "mem") != "Maria" {
		t.Error("cada um com o seu nome")
	}
	if f.ler(t, "USER#own", "SPACE#s1")["role"] != "owner" {
		t.Error("o papel do owner deve continuar owner")
	}
}

func TestNomeVazioRemoveOApelido(t *testing.T) {
	f := cenario(t)
	chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"apelido":"Maria"}`)
	c, d := chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"apelido":"   "}`)
	esperar(t, "limpar o nome", c, 200)
	if d["apelido"] != "" {
		t.Errorf("resposta = %v", d["apelido"])
	}
	if apelidoNoBanco(t, f, "mem") != nil {
		t.Error("o apelido deveria ter sido removido")
	}
}

func TestNomeInvalido(t *testing.T) {
	f := cenario(t)
	casos := map[string]string{
		"longo":       `{"apelido":"` + strings.Repeat("a", 31) + `"}`,
		"controle":    `{"apelido":"Ana\u0000"}`,
		"sem campo":   `{"nome":"x"}`,
		"tipo errado": `{"apelido":123}`,
	}
	for nome, corpo := range casos {
		c, _ := chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, corpo)
		esperar(t, nome, c, 400)
	}
	if apelidoNoBanco(t, f, "mem") != nil {
		t.Error("nenhum nome invalido pode ter sido gravado")
	}
}

func TestEstranhoNaoDefineNome(t *testing.T) {
	f := cenario(t)
	c, _ := chamar(t, "estranho", "PUT", "spaces/s1/perfil", nil, `{"apelido":"Invasor"}`)
	esperar(t, "quem nao e do espaco", c, 403)
	if _, existe := f.itens[chave("USER#estranho", "SPACE#s1")]; existe {
		t.Error("nao pode criar vinculo para quem nao e do espaco")
	}
	if apelidoNoBanco(t, f, "own") != nil || apelidoNoBanco(t, f, "mem") != nil {
		t.Error("ninguem deveria ter ganho apelido")
	}
}

func TestMeDevolveOApelido(t *testing.T) {
	cenario(t)
	chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"apelido":"Maria"}`)
	c, d := chamar(t, "mem", "GET", "me", nil, "")
	esperar(t, "GET /me", c, 200)
	esp, _ := d["espacos"].([]any)
	if len(esp) != 1 {
		t.Fatalf("esperava 1 espaco, veio %v", d["espacos"])
	}
	e := esp[0].(map[string]any)
	if e["apelido"] != "Maria" || e["role"] != "member" || e["id"] != "s1" {
		t.Errorf("espaco = %v", e)
	}
	// o owner ainda nao definiu o nome: vem vazio, sem quebrar
	_, d = chamar(t, "own", "GET", "me", nil, "")
	if e := d["espacos"].([]any)[0].(map[string]any); e["apelido"] != nil {
		t.Errorf("owner sem apelido deveria vir nulo, veio %v", e["apelido"])
	}
}
