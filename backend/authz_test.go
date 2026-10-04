package main

import (
	"reflect"
	"testing"
)

// Matriz completa: toda acao x todo papel. Se alguem acrescentar uma acao e esquecer de
// decidir o papel, o teste de "acoes conhecidas" abaixo falha.
func TestMatrizDePermissoes(t *testing.T) {
	casos := []struct {
		papel string
		a     acao
		quer  decisao
	}{
		// owner: tudo
		{papelOwner, acaoLerTx, liberado},
		{papelOwner, acaoCriarTx, liberado},
		{papelOwner, acaoEditarTx, liberado},
		{papelOwner, acaoExcluirTx, liberado},
		{papelOwner, acaoImportarTx, liberado},
		{papelOwner, acaoLerCfg, liberado},
		{papelOwner, acaoEscreverCfg, liberado},
		{papelOwner, acaoAdicionarTags, liberado},
		{papelOwner, acaoLerAparencia, liberado},
		{papelOwner, acaoEscreverAparenc, liberado},
		{papelOwner, acaoEditarPerfil, liberado},
		// member: ler, criar, tags novas e a propria aparencia; editar so o que criou
		{papelMember, acaoLerTx, liberado},
		{papelMember, acaoCriarTx, liberado},
		{papelMember, acaoEditarTx, soDono},
		{papelMember, acaoExcluirTx, negado},
		{papelMember, acaoImportarTx, negado},
		{papelMember, acaoLerCfg, liberado},
		{papelMember, acaoEscreverCfg, negado},
		{papelMember, acaoAdicionarTags, liberado},
		{papelMember, acaoLerAparencia, liberado},
		{papelMember, acaoEscreverAparenc, liberado},
		{papelMember, acaoEditarPerfil, liberado},
	}
	for _, c := range casos {
		if got := permitido(c.papel, c.a); got != c.quer {
			t.Errorf("permitido(%q, %q) = %v, esperado %v", c.papel, c.a, got, c.quer)
		}
	}
}

func TestTodaAcaoConhecidaTemDecisaoParaOwnerEMember(t *testing.T) {
	todas := []acao{acaoLerTx, acaoCriarTx, acaoEditarTx, acaoExcluirTx, acaoImportarTx,
		acaoLerCfg, acaoEscreverCfg, acaoAdicionarTags, acaoLerAparencia, acaoEscreverAparenc, acaoEditarPerfil}
	for _, a := range todas {
		if permitido(papelOwner, a) != liberado {
			t.Errorf("owner deveria poder %q", a)
		}
	}
	// toda acao que as rotas produzem precisa estar nesta lista
	vistas := map[acao]bool{}
	for _, a := range todas {
		vistas[a] = true
	}
	rotas := []struct {
		m    string
		rest []string
	}{
		{"GET", []string{"tx"}}, {"POST", []string{"tx"}}, {"POST", []string{"tx", "batch"}},
		{"PUT", []string{"tx", "abc"}}, {"DELETE", []string{"tx", "abc"}},
		{"GET", []string{"cfg", "listas"}}, {"PUT", []string{"cfg", "listas"}},
		{"POST", []string{"tags"}}, {"GET", []string{"aparencia"}}, {"PUT", []string{"aparencia"}},
		{"PUT", []string{"perfil"}},
	}
	for _, r := range rotas {
		a, ok := rotaParaAcao(r.m, r.rest)
		if !ok {
			t.Errorf("rota %s %v deveria existir", r.m, r.rest)
			continue
		}
		if !vistas[a] {
			t.Errorf("acao %q da rota %s %v nao esta na lista de acoes conhecidas", a, r.m, r.rest)
		}
	}
}

func TestPapelDesconhecidoOuVazioNaoPodeNada(t *testing.T) {
	for _, papel := range []string{"", "viewer", "OWNER", "Member", "admin"} {
		for _, a := range []acao{acaoLerTx, acaoCriarTx, acaoEditarTx, acaoExcluirTx, acaoImportarTx,
			acaoLerCfg, acaoEscreverCfg, acaoAdicionarTags, acaoLerAparencia, acaoEscreverAparenc, acaoEditarPerfil, "inexistente"} {
			if got := permitido(papel, a); got != negado {
				t.Errorf("papel %q, acao %q: esperado negado, veio %v", papel, a, got)
			}
		}
	}
	if permitido(papelOwner, "inexistente") != negado || permitido(papelMember, "inexistente") != negado {
		t.Error("acao desconhecida deve ser negada, ate para o owner")
	}
}

func TestRotaParaAcao(t *testing.T) {
	casos := []struct {
		m    string
		rest []string
		a    acao
		ok   bool
	}{
		{"GET", []string{"tx"}, acaoLerTx, true},
		{"POST", []string{"tx"}, acaoCriarTx, true},
		{"POST", []string{"tx", "batch"}, acaoImportarTx, true},
		{"PUT", []string{"tx", "abc123"}, acaoEditarTx, true},
		{"DELETE", []string{"tx", "abc123"}, acaoExcluirTx, true},
		{"GET", []string{"cfg", "saldos"}, acaoLerCfg, true},
		{"PUT", []string{"cfg", "cartoes"}, acaoEscreverCfg, true},
		{"PUT", []string{"cfg", "tags"}, acaoEscreverCfg, true}, // reescrever a lista de tags e do owner
		{"POST", []string{"tags"}, acaoAdicionarTags, true},
		{"GET", []string{"aparencia"}, acaoLerAparencia, true},
		{"PUT", []string{"aparencia"}, acaoEscreverAparenc, true},
		{"PUT", []string{"perfil"}, acaoEditarPerfil, true},
		// nao existem
		{"GET", []string{"perfil"}, "", false},
		{"POST", []string{"perfil"}, "", false},
		{"DELETE", []string{"perfil"}, "", false},
		{"GET", []string{"tx", "abc"}, "", false},
		{"DELETE", []string{"tx"}, "", false},
		{"PATCH", []string{"tx", "abc"}, "", false},
		{"GET", []string{"tx", "batch"}, "", false},
		{"GET", []string{"cfg", "outra"}, "", false},
		{"PUT", []string{"cfg", "outra"}, "", false},
		{"GET", []string{"tags"}, "", false},
		{"DELETE", []string{"tags"}, "", false},
		{"POST", []string{"aparencia"}, "", false},
		{"GET", []string{}, "", false},
		{"GET", []string{"qualquer"}, "", false},
	}
	for _, c := range casos {
		a, ok := rotaParaAcao(c.m, c.rest)
		if ok != c.ok || a != c.a {
			t.Errorf("rotaParaAcao(%s, %v) = (%q, %v), esperado (%q, %v)", c.m, c.rest, a, ok, c.a, c.ok)
		}
	}
}

// PUT /tx/batch cai como edicao de um lancamento chamado "batch" (que nao existe): nunca importa.
func TestPutEmBatchNaoEImportacao(t *testing.T) {
	a, ok := rotaParaAcao("PUT", []string{"tx", "batch"})
	if !ok || a != acaoEditarTx {
		t.Fatalf("PUT /tx/batch deveria ser edicao, veio (%q, %v)", a, ok)
	}
	if permitido(papelMember, acaoImportarTx) != negado {
		t.Error("member nao pode importar")
	}
}

func TestEhDono(t *testing.T) {
	reg := doc{"criadoPor": "sub-ela"}
	if !ehDono(reg, "sub-ela") {
		t.Error("o criador deve ser dono")
	}
	if ehDono(reg, "sub-outro") {
		t.Error("outro usuario nao e dono")
	}
	if ehDono(doc{}, "sub-ela") {
		t.Error("registro sem criadoPor nao e de ninguem")
	}
	if ehDono(doc{"criadoPor": ""}, "") {
		t.Error("sub vazio nunca e dono, nem de registro com criadoPor vazio")
	}
	if ehDono(doc{"criadoPor": 123}, "123") {
		t.Error("criadoPor com tipo errado nao conta")
	}
}

func TestNormalizarTags(t *testing.T) {
	longa := ""
	for i := 0; i < 61; i++ {
		longa += "a"
	}
	got := normalizarTags([]string{" Lazer ", "#Viagem", "", "   ", "#", "Lazer", longa, "##x"})
	want := []string{"Lazer", "Viagem", "#x"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizarTags = %v, esperado %v", got, want)
	}
	muitas := make([]string, 150)
	for i := range muitas {
		muitas[i] = "t" + string(rune('A'+i%26)) + string(rune('a'+i/26))
	}
	if n := len(normalizarTags(muitas)); n != 100 {
		t.Errorf("limite de 100 tags por chamada: veio %d", n)
	}
}

func TestFaltantes(t *testing.T) {
	got := faltantes([]string{"a", "b", "c"}, []string{"b", "x"})
	if !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Errorf("faltantes = %v", got)
	}
	if faltantes([]string{"a"}, []string{"a"}) != nil {
		t.Error("nada deveria faltar")
	}
	// maiusculas contam como tags diferentes (o app trata tags como texto exato)
	if got := faltantes([]string{"Lazer"}, []string{"lazer"}); !reflect.DeepEqual(got, []string{"Lazer"}) {
		t.Errorf("Lazer e lazer sao tags diferentes: %v", got)
	}
}

func TestTagsDe(t *testing.T) {
	d := doc{"tags": []any{"a", 1, "b", nil}}
	if got := tagsDe(d); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("tagsDe = %v", got)
	}
	if tagsDe(doc{}) != nil || tagsDe(doc{"tags": "x"}) != nil {
		t.Error("sem lista de tags deve devolver nil")
	}
}
