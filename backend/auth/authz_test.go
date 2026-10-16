package auth

import "testing"

// Matriz completa: toda Acao x todo papel. Se alguem acrescentar uma Acao e esquecer de
// decidir o papel, o teste de "acoes conhecidas" abaixo falha.
func TestMatrizDePermissoes(t *testing.T) {
	casos := []struct {
		papel string
		a     Acao
		quer  Decisao
	}{
		// owner: tudo
		{PapelOwner, AcaoLerTx, Liberado},
		{PapelOwner, AcaoCriarTx, Liberado},
		{PapelOwner, AcaoEditarTx, Liberado},
		{PapelOwner, AcaoExcluirTx, Liberado},
		{PapelOwner, AcaoImportarTx, Liberado},
		{PapelOwner, AcaoLerCfg, Liberado},
		{PapelOwner, AcaoEscreverCfg, Liberado},
		{PapelOwner, AcaoAdicionarTags, Liberado},
		{PapelOwner, AcaoLerAparencia, Liberado},
		{PapelOwner, AcaoEscreverAparenc, Liberado},
		{PapelOwner, AcaoEditarPerfil, Liberado},
		{PapelOwner, AcaoLerFech, Liberado},
		{PapelOwner, AcaoLerSeq, Liberado},
		{PapelOwner, AcaoEscreverFech, Liberado},
		{PapelOwner, AcaoConferirMes, Liberado},
		{PapelOwner, AcaoReabrirMes, Liberado},
		// member: ler, criar, tags novas e a propria aparencia; editar so o que criou
		{PapelMember, AcaoLerTx, Liberado},
		{PapelMember, AcaoCriarTx, Liberado},
		{PapelMember, AcaoEditarTx, SoDono},
		{PapelMember, AcaoExcluirTx, Negado},
		{PapelMember, AcaoImportarTx, Negado},
		{PapelMember, AcaoLerCfg, Liberado},
		{PapelMember, AcaoEscreverCfg, Negado},
		{PapelMember, AcaoAdicionarTags, Liberado},
		{PapelMember, AcaoLerAparencia, Liberado},
		{PapelMember, AcaoEscreverAparenc, Liberado},
		{PapelMember, AcaoEditarPerfil, Liberado},
		{PapelMember, AcaoLerFech, Liberado},
		{PapelMember, AcaoLerSeq, Liberado},
		{PapelMember, AcaoEscreverFech, Negado},
		{PapelMember, AcaoConferirMes, Negado},
		{PapelMember, AcaoReabrirMes, Negado},
	}
	for _, c := range casos {
		if got := Permitido(c.papel, c.a); got != c.quer {
			t.Errorf("Permitido(%q, %q) = %v, esperado %v", c.papel, c.a, got, c.quer)
		}
	}
}

func TestTodaAcaoConhecidaTemDecisaoParaOwnerEMember(t *testing.T) {
	todas := []Acao{AcaoLerTx, AcaoCriarTx, AcaoEditarTx, AcaoExcluirTx, AcaoImportarTx,
		AcaoLerCfg, AcaoEscreverCfg, AcaoAdicionarTags, AcaoLerAparencia, AcaoEscreverAparenc, AcaoEditarPerfil,
		AcaoLerFech, AcaoEscreverFech, AcaoConferirMes, AcaoReabrirMes, AcaoLerSeq}
	for _, a := range todas {
		if Permitido(PapelOwner, a) != Liberado {
			t.Errorf("owner deveria poder %q", a)
		}
	}
	// toda Acao que as rotas produzem precisa estar nesta lista
	vistas := map[Acao]bool{}
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
		{"GET", []string{"seq"}},
		{"GET", []string{"fechamentos"}}, {"PUT", []string{"fechamentos"}},
		{"POST", []string{"fechamentos", "conferir"}}, {"POST", []string{"fechamentos", "reabrir"}},
	}
	for _, r := range rotas {
		a, ok := RotaParaAcao(r.m, r.rest)
		if !ok {
			t.Errorf("rota %s %v deveria existir", r.m, r.rest)
			continue
		}
		if !vistas[a] {
			t.Errorf("Acao %q da rota %s %v nao esta na lista de acoes conhecidas", a, r.m, r.rest)
		}
	}
}

func TestPapelDesconhecidoOuVazioNaoPodeNada(t *testing.T) {
	for _, papel := range []string{"", "viewer", "OWNER", "Member", "admin"} {
		for _, a := range []Acao{AcaoLerTx, AcaoCriarTx, AcaoEditarTx, AcaoExcluirTx, AcaoImportarTx,
			AcaoLerCfg, AcaoEscreverCfg, AcaoAdicionarTags, AcaoLerAparencia, AcaoEscreverAparenc, AcaoEditarPerfil,
			AcaoLerFech, AcaoEscreverFech, AcaoConferirMes, AcaoReabrirMes, AcaoLerSeq, "inexistente"} {
			if got := Permitido(papel, a); got != Negado {
				t.Errorf("papel %q, Acao %q: esperado Negado, veio %v", papel, a, got)
			}
		}
	}
	if Permitido(PapelOwner, "inexistente") != Negado || Permitido(PapelMember, "inexistente") != Negado {
		t.Error("Acao desconhecida deve ser negada, ate para o owner")
	}
}

func TestRotaParaAcao(t *testing.T) {
	casos := []struct {
		m    string
		rest []string
		a    Acao
		ok   bool
	}{
		{"GET", []string{"tx"}, AcaoLerTx, true},
		{"POST", []string{"tx"}, AcaoCriarTx, true},
		{"POST", []string{"tx", "batch"}, AcaoImportarTx, true},
		{"PUT", []string{"tx", "abc123"}, AcaoEditarTx, true},
		{"DELETE", []string{"tx", "abc123"}, AcaoExcluirTx, true},
		{"GET", []string{"cfg", "saldos"}, AcaoLerCfg, true},
		{"PUT", []string{"cfg", "cartoes"}, AcaoEscreverCfg, true},
		{"PUT", []string{"cfg", "tags"}, AcaoEscreverCfg, true}, // reescrever a lista de tags e do owner
		{"POST", []string{"tags"}, AcaoAdicionarTags, true},
		{"GET", []string{"aparencia"}, AcaoLerAparencia, true},
		{"PUT", []string{"aparencia"}, AcaoEscreverAparenc, true},
		{"PUT", []string{"perfil"}, AcaoEditarPerfil, true},
		{"GET", []string{"seq"}, AcaoLerSeq, true},
		{"POST", []string{"seq"}, "", false},
		{"GET", []string{"fechamentos"}, AcaoLerFech, true},
		{"PUT", []string{"fechamentos"}, AcaoEscreverFech, true},
		{"POST", []string{"fechamentos", "conferir"}, AcaoConferirMes, true},
		{"POST", []string{"fechamentos", "reabrir"}, AcaoReabrirMes, true},
		{"DELETE", []string{"fechamentos"}, "", false},
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
		a, ok := RotaParaAcao(c.m, c.rest)
		if ok != c.ok || a != c.a {
			t.Errorf("RotaParaAcao(%s, %v) = (%q, %v), esperado (%q, %v)", c.m, c.rest, a, ok, c.a, c.ok)
		}
	}
}

// PUT /tx/batch cai como edicao de um lancamento chamado "batch" (que nao existe): nunca importa.
func TestPutEmBatchNaoEImportacao(t *testing.T) {
	a, ok := RotaParaAcao("PUT", []string{"tx", "batch"})
	if !ok || a != AcaoEditarTx {
		t.Fatalf("PUT /tx/batch deveria ser edicao, veio (%q, %v)", a, ok)
	}
	if Permitido(PapelMember, AcaoImportarTx) != Negado {
		t.Error("member nao pode importar")
	}
}
