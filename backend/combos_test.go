package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Item 57 (.ai/specs/combos-sugeridos/spec.md): o servidor conta os combos sugeridos uma vez por semana e guarda em CFG#COMBOS;
// o dono liga e desliga a rotina em Listas (item auxiliar tipo "combos" no AVISOS#ATIVOS). O relogio padrao dos testes e 05/02/2026.

func lancComTags(f *bancoFake, t *testing.T, sid, data, id string, tags ...any) {
	f.colocar(t, doc{"PK": "SPACE#" + sid, "SK": "TX#" + data + "#" + id, "id": id, "dataEvento": data,
		"status": "PAGO", "tipo": "saida", "valor": 10.0, "banco": "ITAU", "descricao": "x", "tags": tags})
}

func ligarRotina(t *testing.T, f *bancoFake, sid string) {
	t.Helper()
	f.colocar(t, doc{"PK": "AVISOS#ATIVOS", "SK": sid + "#combos", "sid": sid, "id": "combos", "tipo": "combos"})
}

func rodarCombos(t *testing.T) doc {
	t.Helper()
	r, err := srvTeste.combos.Rodar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return doc(r)
}

func sugeridosDe(d doc) []any {
	l, _ := d["sugeridos"].([]any)
	return l
}

func TestCombosRotinaGravaSemMudarVersionNemLista(t *testing.T) { // R6
	f := cenario(t)
	for i := 0; i < 3; i++ {
		lancComTags(f, t, "s1", "2026-02-01T10:00:00", fmt.Sprintf("m%d", i), "mercado", "mercado-mensal")
	}
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#COMBOS", "lista": []any{doc{"tags": []any{"a", "b"}}}, "version": 3.0})
	ligarRotina(t, f, "s1")
	r := rodarCombos(t)
	if r["espacos"] != 1 || r["sugeridos"] != 1 {
		t.Fatalf("rotina = %v", r)
	}
	c := f.ler(t, pkEspaco, "CFG#COMBOS")
	if c["version"] != 3.0 || len(c["lista"].([]any)) != 1 {
		t.Fatalf("version e lista nao mudam: %v", c)
	}
	s := sugeridosDe(c)
	if len(s) != 1 || s[0].(doc)["contagem"] != 3.0 || c["geradoEm"] != "2026-02-05" {
		t.Fatalf("sugeridos = %v geradoEm = %v", s, c["geradoEm"])
	}
}

func TestCombosRotinaCriaOItemQuandoNaoExiste(t *testing.T) { // R6
	f := cenario(t)
	for i := 0; i < 3; i++ {
		lancComTags(f, t, "s1", "2026-02-01T10:00:00", fmt.Sprintf("m%d", i), "a", "b")
	}
	ligarRotina(t, f, "s1")
	rodarCombos(t)
	c := f.ler(t, pkEspaco, "CFG#COMBOS")
	if c == nil || len(sugeridosDe(c)) != 1 || len(c["lista"].([]any)) != 0 {
		t.Fatalf("item criado = %v", c)
	}
}

func TestCombosSemItemAtivoNaoFazNada(t *testing.T) { // R5
	f := cenario(t)
	for i := 0; i < 3; i++ {
		lancComTags(f, t, "s1", "2026-02-01T10:00:00", fmt.Sprintf("m%d", i), "a", "b")
	}
	r := rodarCombos(t)
	if r["espacos"] != 0 || r["sugeridos"] != 0 || f.ler(t, pkEspaco, "CFG#COMBOS") != nil {
		t.Fatalf("sem item ativo nada muda: %v", r)
	}
}

func TestCombosSoOsMil_MaisRecentes(t *testing.T) { // R3
	f := cenario(t)
	for i := 0; i < 3; i++ {
		lancComTags(f, t, "s1", "2025-01-01T10:00:00", fmt.Sprintf("v%d", i), "x", "y") // antigos: ficam fora dos 1000
	}
	for i := 0; i < 1000; i++ {
		lancComTags(f, t, "s1", "2026-01-15T10:00:00", fmt.Sprintf("n%04d", i), "a", "b")
	}
	ligarRotina(t, f, "s1")
	rodarCombos(t)
	s := sugeridosDe(f.ler(t, pkEspaco, "CFG#COMBOS"))
	if len(s) != 1 || s[0].(doc)["contagem"] != 1000.0 {
		t.Fatalf("so o conjunto dos 1000 mais recentes: %v", s)
	}
}

// bancoQueFalhaNoEspaco faz a leitura de um espaco falhar, para ver que os outros seguem.
type bancoQueFalhaNoEspaco struct {
	*bancoFake
	pk string
}

func (b *bancoQueFalhaNoEspaco) Query(ctx context.Context, in *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if pk, ok := in.ExpressionAttributeValues[":pk"]; ok && sv(pk) == b.pk && sv(in.ExpressionAttributeValues[":a"]) == "TX#" {
		return nil, errors.New("falha de leitura")
	}
	return b.bancoFake.Query(ctx, in, o...)
}

func TestCombosUmEspacoComErroNaoImpedeOsOutros(t *testing.T) { // R7, R14
	f := cenario(t)
	srvTeste = novoServidor(&bancoQueFalhaNoEspaco{f, "SPACE#s1"}, "", relogioTeste, notifTeste, "")
	for i := 0; i < 3; i++ {
		lancComTags(f, t, "s1", "2026-02-01T10:00:00", fmt.Sprintf("m%d", i), "a", "b")
		lancComTags(f, t, "s2", "2026-02-01T10:00:00", fmt.Sprintf("m%d", i), "c", "d")
	}
	ligarRotina(t, f, "s1")
	ligarRotina(t, f, "s2")
	r := rodarCombos(t)
	if r["espacos"] != 1 || f.ler(t, "SPACE#s1", "CFG#COMBOS") != nil || len(sugeridosDe(f.ler(t, "SPACE#s2", "CFG#COMBOS"))) != 1 {
		t.Fatalf("s1 falha, s2 calcula so o dele: %v", r)
	}
}

func TestCombosInterruptorDoDono(t *testing.T) { // R5a
	f := cenario(t)
	st, d := chamar(t, "mem", "PUT", "spaces/s1/combos/rotina", nil, `{"ativa":true}`)
	esperar(t, "membro nao liga", st, 403)
	st, d = chamar(t, "own", "PUT", "spaces/s1/combos/rotina", nil, `{"ativa":"sim"}`)
	esperar(t, "corpo invalido", st, 400)
	st, d = chamar(t, "own", "PUT", "spaces/s1/combos/rotina", nil, `{"ativa":true}`)
	esperar(t, "liga", st, 200)
	if d["rotinaAtiva"] != true || f.ler(t, "AVISOS#ATIVOS", "s1#combos")["tipo"] != "combos" {
		t.Fatalf("ligar cria o item: %v", d)
	}
	_, d = chamar(t, "mem", "GET", "spaces/s1/cfg/combos", nil, "")
	if d["rotinaAtiva"] != true {
		t.Fatalf("GET mostra rotinaAtiva: %v", d)
	}
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#COMBOS", "lista": []any{}, "sugeridos": []any{doc{"tags": []any{"a", "b"}, "contagem": 4.0}}, "geradoEm": "2026-02-01", "version": 1.0})
	st, d = chamar(t, "own", "PUT", "spaces/s1/combos/rotina", nil, `{"ativa":false}`)
	esperar(t, "desliga", st, 200)
	if d["rotinaAtiva"] != false || f.ler(t, "AVISOS#ATIVOS", "s1#combos") != nil {
		t.Fatalf("desligar apaga o item: %v", d)
	}
	_, d = chamar(t, "own", "GET", "spaces/s1/cfg/combos", nil, "")
	if len(sugeridosDe(d)) != 1 || d["rotinaAtiva"] != false {
		t.Fatalf("desligar mantem os sugeridos: %v", d)
	}
}

func TestCombosAvisosIgnoramOItemDaRotina(t *testing.T) { // R5b
	f := cenario(t)
	ligarRotina(t, f, "s1")
	if _, err := srvTeste.avisos.Rodar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.ler(t, "AVISOS#ATIVOS", "s1#combos") == nil {
		t.Fatal("a rotina dos avisos nao apaga nem trata o item dos combos")
	}
}

func TestCombosPutManualPreservaSugeridos(t *testing.T) { // R9
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#COMBOS", "lista": []any{}, "sugeridos": []any{doc{"tags": []any{"a", "b"}, "contagem": 4.0}}, "geradoEm": "2026-02-01", "version": 2.0})
	st, _ := chamar(t, "own", "PUT", "spaces/s1/cfg/combos", nil,
		`{"version":2,"lista":[{"tags":["c","d"]}],"sugeridos":[],"geradoEm":"x","rotinaAtiva":true}`)
	esperar(t, "salva", st, 200)
	c := f.ler(t, pkEspaco, "CFG#COMBOS")
	if c["version"] != 3.0 || len(c["lista"].([]any)) != 1 || len(sugeridosDe(c)) != 1 || c["geradoEm"] != "2026-02-01" || c["rotinaAtiva"] != nil {
		t.Fatalf("o PUT manual preserva sugeridos e geradoEm: %v", c)
	}
}
