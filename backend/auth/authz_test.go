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
		{PapelOwner, AcaoConferirMes, Liberado},
		{PapelOwner, AcaoReabrirMes, Liberado},
		{PapelOwner, AcaoLerHistorico, Liberado},
		{PapelOwner, AcaoRestaurar, Liberado},
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
		{PapelMember, AcaoConferirMes, Negado},
		{PapelMember, AcaoReabrirMes, Negado},
		{PapelMember, AcaoLerHistorico, SoDono},
		{PapelMember, AcaoRestaurar, SoDono},
		{PapelOwner, AcaoLerCartao, Liberado},
		{PapelOwner, AcaoEscreverCartao, Liberado},
		{PapelMember, AcaoLerCartao, Liberado},
		{PapelMember, AcaoEscreverCartao, Negado},
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
		AcaoLerFech, AcaoConferirMes, AcaoReabrirMes, AcaoLerSeq, AcaoLerHistorico, AcaoRestaurar, AcaoLerCartao, AcaoEscreverCartao}
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
		{"GET", []string{"seq"}}, {"GET", []string{"historico"}},
		{"GET", []string{"tx", "abc", "versoes"}}, {"POST", []string{"tx", "abc", "versoes", "2", "restaurar"}},
		{"GET", []string{"saldos"}},
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
			AcaoLerFech, AcaoConferirMes, AcaoReabrirMes, AcaoLerSeq, AcaoLerHistorico, AcaoRestaurar, AcaoLerCartao, AcaoEscreverCartao, "inexistente"} {
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
		{"GET", []string{"saldos"}, AcaoLerFech, true},
		{"GET", []string{"fechamentos"}, "", false}, // saiu na fase C do item 40
		{"GET", []string{"saldos"}, AcaoLerFech, true}, // o saldo de cada meio, calculado pelo servidor (item 40)
		{"POST", []string{"saldos"}, "", false},
		{"PUT", []string{"saldos"}, "", false},
		{"DELETE", []string{"saldos"}, "", false},
		{"GET", []string{"saldos", "x"}, "", false},
		{"PUT", []string{"fechamentos"}, "", false}, // saiu na fase C do item 40
		{"POST", []string{"fechamentos", "conferir"}, AcaoConferirMes, true},
		{"POST", []string{"fechamentos", "reabrir"}, AcaoReabrirMes, true},
		{"DELETE", []string{"fechamentos"}, "", false},
		{"GET", []string{"historico"}, AcaoLerHistorico, true},
		{"GET", []string{"tx", "abc123", "versoes"}, AcaoLerHistorico, true},
		{"POST", []string{"tx", "abc123", "versoes", "2", "restaurar"}, AcaoRestaurar, true},
		// historico: metodo errado ou caminho extra nao existe
		{"POST", []string{"historico"}, "", false},
		{"PUT", []string{"historico"}, "", false},
		{"DELETE", []string{"historico"}, "", false},
		{"GET", []string{"historico", "x"}, "", false},
		{"POST", []string{"tx", "abc123", "versoes"}, "", false},
		{"PUT", []string{"tx", "abc123", "versoes"}, "", false},
		{"DELETE", []string{"tx", "abc123", "versoes"}, "", false},
		{"GET", []string{"tx", "abc123", "versoes", "2"}, "", false},
		{"GET", []string{"tx", "abc123", "versoes", "2", "restaurar"}, "", false},
		{"PUT", []string{"tx", "abc123", "versoes", "2", "restaurar"}, "", false},
		{"POST", []string{"tx", "abc123", "versoes", "2", "restaurar", "x"}, "", false},
		{"POST", []string{"tx", "abc123", "outra", "2", "restaurar"}, "", false},
		// cartoes (item 33a): as leituras e os comandos
		{"GET", []string{"cartoes"}, AcaoLerCartao, true},
		{"POST", []string{"cartoes"}, AcaoEscreverCartao, true},
		{"PUT", []string{"cartoes"}, "", false},
		{"DELETE", []string{"cartoes"}, "", false},
		{"GET", []string{"cartoes", "X"}, "", false},
		{"PUT", []string{"cartoes", "X"}, AcaoEscreverCartao, true},
		{"DELETE", []string{"cartoes", "X"}, AcaoEscreverCartao, true},
		{"POST", []string{"cartoes", "X"}, "", false},
		{"GET", []string{"cartoes", "X", "faturas"}, AcaoLerCartao, true},
		{"GET", []string{"cartoes", "X", "faturas", "2026-10"}, AcaoLerCartao, true},
		{"PUT", []string{"cartoes", "X", "faturas", "2026-10"}, AcaoEscreverCartao, true},
		{"PUT", []string{"cartoes", "X", "faturas"}, "", false},
		{"POST", []string{"cartoes", "X", "faturas", "2026-10"}, "", false},
		{"POST", []string{"cartoes", "X", "faturas", "2026-10", "conferir"}, AcaoEscreverCartao, true},
		{"DELETE", []string{"cartoes", "X", "faturas", "2026-10", "conferir"}, AcaoEscreverCartao, true},
		{"GET", []string{"cartoes", "X", "faturas", "2026-10", "conferir"}, "", false},
		{"POST", []string{"cartoes", "X", "faturas", "2026-10", "diferenca"}, AcaoEscreverCartao, true},
		{"DELETE", []string{"cartoes", "X", "faturas", "2026-10", "diferenca"}, "", false},
		{"POST", []string{"cartoes", "X", "faturas", "2026-10", "outra"}, "", false},
		{"POST", []string{"cartoes", "X", "faturas", "2026-10", "conferir", "y"}, "", false},
		{"GET", []string{"cartoes", "X", "outra", "2026-10"}, "", false},
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
