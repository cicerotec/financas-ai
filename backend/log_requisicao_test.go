package main

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"financas/auth"
)

// Spec .ai/specs/log-de-requisicoes/spec.md (L-T1 a L-T8).

func capturarLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var b bytes.Buffer
	antes, flags := log.Writer(), log.Flags()
	log.SetOutput(&b)
	log.SetFlags(0)
	t.Cleanup(func() { log.SetOutput(antes); log.SetFlags(flags) })
	return &b
}

func TestRotular(t *testing.T) { // L-T1, L-T3
	casos := []struct {
		m    string
		p    []string
		quer rotulo
	}{
		{"GET", []string{"spaces", "s1", "tx"}, rotulo{string(auth.AcaoLerTx), "tx"}},
		{"POST", []string{"spaces", "s1", "tx"}, rotulo{string(auth.AcaoCriarTx), "tx"}},
		{"GET", []string{"spaces", "s1", "cfg", "listas"}, rotulo{string(auth.AcaoLerCfg), "cfg/listas"}},
		{"GET", []string{"spaces", "s1", "cfg", "combos"}, rotulo{string(auth.AcaoLerCfg), "cfg/combos"}},
		{"GET", []string{"spaces", "s1", "cfg", "tags"}, rotulo{string(auth.AcaoLerCfg), "cfg/tags"}},
		{"GET", []string{"spaces", "s1", "aparencia"}, rotulo{string(auth.AcaoLerAparencia), "aparencia"}},
		{"GET", []string{"me"}, rotulo{acaoMe, "me"}},
		{"POST", []string{"me"}, rotulo{acaoDesconhecida, "-"}},
		{"GET", []string{"nada"}, rotulo{acaoDesconhecida, "-"}},
		{"GET", []string{""}, rotulo{acaoDesconhecida, "-"}},
		{"GET", []string{"spaces", "s1"}, rotulo{acaoDesconhecida, "-"}},
		{"GET", []string{"spaces", "s1", "cfg", "inventada"}, rotulo{acaoDesconhecida, "-"}},
		{"PATCH", []string{"spaces", "s1", "tx"}, rotulo{acaoDesconhecida, "-"}},
	}
	for _, c := range casos {
		if got := rotular(c.m, c.p); got != c.quer {
			t.Errorf("rotular(%s, %v) = %+v, esperado %+v", c.m, c.p, got, c.quer)
		}
	}
}

func TestRotularNaoLevaId(t *testing.T) { // L-T5
	for _, p := range [][]string{
		{"spaces", "ESPACO-SECRETO", "tx", "ID-SECRETO"},
		{"spaces", "ESPACO-SECRETO", "tx", "ID-SECRETO", "versoes", "3", "restaurar"},
		{"spaces", "ESPACO-SECRETO", "cfg", "ID-SECRETO"},
		{"spaces", "ESPACO-SECRETO", "cartoes", "ID-SECRETO", "faturas", "2026-10"},
	} {
		for _, m := range []string{"GET", "POST", "PUT", "DELETE"} {
			rt := rotular(m, p)
			if strings.Contains(rt.Acao+rt.Rota, "SECRETO") {
				t.Errorf("rotular(%s, %v) vazou um id: %+v", m, p, rt)
			}
		}
	}
}

func TestLinhaDeLog(t *testing.T) { // L-T8
	d := 12*time.Millisecond + 900*time.Microsecond
	rt := rotulo{"cfg.ler", "cfg/listas"}
	got := linhaDeLog("abc-123", "GET", rt, 200, d)
	quer := "req rid=abc-123 m=GET acao=cfg.ler rota=cfg/listas status=200 ms=12"
	if got != quer {
		t.Errorf("linha = %q, esperado %q", got, quer)
	}
	if linhaDeLog("abc-123", "GET", rt, 200, d) != got {
		t.Error("a mesma entrada deu linhas diferentes")
	}
}

func TestLinhaDeLogValoresSemEspaco(t *testing.T) { // R2
	got := linhaDeLog("", "GET", rotulo{"a b", "x y"}, 200, 0)
	quer := "req rid=- m=GET acao=- rota=- status=200 ms=0"
	if got != quer {
		t.Errorf("linha = %q, esperado %q", got, quer)
	}
}

func TestRegistrarUmaLinhaPorRequisicao(t *testing.T) { // L-T2, L-T4, L-T6
	casos := []struct {
		nome string
		rt   rotulo
		res  resp
		err  error
		quer string
	}{
		{"sem token", rotulo{acaoSemToken, "-"}, resp{StatusCode: 401}, nil, "acao=sem-token rota=- status=401"},
		{"sem permissao", rotulo{string(auth.AcaoExcluirTx), "tx"}, resp{StatusCode: 403}, nil, "acao=tx.excluir rota=tx status=403"},
		{"erro go", rotulo{string(auth.AcaoLerTx), "tx"}, resp{}, errors.New("falhou"), "acao=tx.ler rota=tx status=500"},
	}
	for _, c := range casos {
		b := capturarLog(t)
		registrar("rid1", "DELETE", c.rt, c.res, c.err, 5*time.Millisecond)
		linhas := strings.Split(strings.TrimSpace(b.String()), string(rune(10)))
		if len(linhas) != 1 {
			t.Fatalf("%s: %d linhas, esperada 1: %q", c.nome, len(linhas), b.String())
		}
		if !strings.Contains(linhas[0], c.quer) || !strings.Contains(linhas[0], "ms=5") {
			t.Errorf("%s: linha %q sem %q", c.nome, linhas[0], c.quer)
		}
	}
}

func TestEntradaDoAgendamentoNaoLogaRequisicao(t *testing.T) { // L-T7
	// O evento {"acao":"avisos"} sai em entrada antes do handler, que e quem escreve a linha: o log so existe em handler.
	b := capturarLog(t)
	if strings.Contains(b.String(), "req rid=") {
		t.Fatal("log inesperado antes de qualquer requisicao")
	}
}
