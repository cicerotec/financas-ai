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
		quer string
	}{
		{"GET", []string{"spaces", "s1", "tx"}, string(auth.AcaoLerTx)},
		{"POST", []string{"spaces", "s1", "tx"}, string(auth.AcaoCriarTx)},
		{"GET", []string{"me"}, acaoMe},
		{"POST", []string{"me"}, acaoDesconhecida},
		{"GET", []string{"nada"}, acaoDesconhecida},
		{"GET", []string{""}, acaoDesconhecida},
		{"GET", []string{"spaces", "s1"}, acaoDesconhecida},
		{"PATCH", []string{"spaces", "s1", "tx"}, acaoDesconhecida},
	}
	for _, c := range casos {
		if got := rotular(c.m, c.p); got != c.quer {
			t.Errorf("rotular(%s, %v) = %q, esperado %q", c.m, c.p, got, c.quer)
		}
	}
}

func TestRotularNaoLevaId(t *testing.T) { // L-T5
	got := rotular("PUT", []string{"spaces", "ESPACO-SECRETO", "tx", "ID-SECRETO"})
	if strings.Contains(got, "SECRETO") {
		t.Errorf("a acao vazou um id: %q", got)
	}
}

func TestLinhaDeLog(t *testing.T) { // L-T8
	d := 12*time.Millisecond + 900*time.Microsecond
	got := linhaDeLog("abc-123", "GET", "tx.ler", 200, d)
	quer := "req rid=abc-123 m=GET acao=tx.ler status=200 ms=12"
	if got != quer {
		t.Errorf("linha = %q, esperado %q", got, quer)
	}
	if linhaDeLog("abc-123", "GET", "tx.ler", 200, d) != got {
		t.Error("a mesma entrada deu linhas diferentes")
	}
}

func TestLinhaDeLogValoresSemEspaco(t *testing.T) { // R2
	got := linhaDeLog("", "GET", "a b", 200, 0)
	quer := "req rid=- m=GET acao=- status=200 ms=0"
	if got != quer {
		t.Errorf("linha = %q, esperado %q", got, quer)
	}
}

func TestRegistrarUmaLinhaPorRequisicao(t *testing.T) { // L-T2, L-T4, L-T6
	casos := []struct {
		nome string
		acao string
		res  resp
		err  error
		quer string
	}{
		{"sem token", acaoSemToken, resp{StatusCode: 401}, nil, "acao=sem-token status=401"},
		{"sem permissao", string(auth.AcaoExcluirTx), resp{StatusCode: 403}, nil, "acao=tx.excluir status=403"},
		{"erro go", string(auth.AcaoLerTx), resp{}, errors.New("falhou"), "acao=tx.ler status=500"},
	}
	for _, c := range casos {
		b := capturarLog(t)
		registrar("rid1", "DELETE", c.acao, c.res, c.err, 5*time.Millisecond)
		linhas := strings.Split(strings.TrimSpace(b.String()), "\n")
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
