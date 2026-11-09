package main

import (
	"errors"
	"testing"
)

// Item 60: o meio que entra na lista nasce com inicio do controle = hoje e saldo inicial 0, gravado pelo servidor em CFG#SALDOS.porBanco (sem campo novo).

func putListas(t *testing.T, bancos string) (int, doc) {
	t.Helper()
	return chamar(t, "own", "PUT", "spaces/s1/cfg/listas", nil, `{"version":1,"banco":`+bancos+`}`)
}

// N-T1: nome novo na lista cria a entrada em Saldos com dataInicio = hoje (fuso do servidor), saldoInicial 0, e a resposta diz quais meios foram iniciados.
func TestMeioNovoNasceComInicioHoje(t *testing.T) {
	relogioFixo(t) // 2026-10-05
	f := cenario(t)
	st, d := putListas(t, `["ITAU","NOVO"]`)
	esperar(t, "adicionar NOVO", st, 200)
	novos, _ := d["meiosNovos"].([]any)
	if len(novos) != 1 || novos[0].(map[string]any)["nome"] != "NOVO" || novos[0].(map[string]any)["dataInicio"] != "2026-10-05" {
		t.Fatalf("meiosNovos = %v", d["meiosNovos"])
	}
	por := f.ler(t, pkEspaco, "CFG#SALDOS")["porBanco"].(map[string]any)
	n, _ := por["NOVO"].(map[string]any)
	if n["dataInicio"] != "2026-10-05" || n["saldoInicial"] != 0.0 {
		t.Errorf("NOVO em Saldos = %v", por["NOVO"])
	}
	if it, _ := por["ITAU"].(map[string]any); it["dataInicio"] != "2020-01-01" {
		t.Errorf("o meio que ja existia nao e tocado: %v", por["ITAU"])
	}
	// o /saldos calcula o meio novo (saldo 0) sem ler o passado
	f.falharQuerySemInicio = errors.New("leu o passado inteiro")
	c, s := saldos(t, "own")
	esperar(t, "saldos", c, 200)
	if m := itemDoMeio(t, s, "NOVO"); m["naoCalculado"] != nil || m["semInicio"] != false || !quase(m["saldo"].(float64), 0) {
		t.Errorf("NOVO calculado com saldo 0: %v", m)
	}
}

// N-T2: gravar a lista de novo, sem nome novo, nao devolve meiosNovos nem mexe em Saldos.
func TestListasSemNomeNovoNaoMexeEmSaldos(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	v := f.ler(t, pkEspaco, "CFG#SALDOS")["version"]
	st, d := putListas(t, `["ITAU"]`)
	esperar(t, "gravar sem novo", st, 200)
	if d["meiosNovos"] != nil {
		t.Errorf("meiosNovos = %v", d["meiosNovos"])
	}
	if f.ler(t, pkEspaco, "CFG#SALDOS")["version"] != v {
		t.Error("Saldos nao pode ser regravado")
	}
}

// N-T3: um nome que ja tem entrada em Saldos (voltou depois de um Remover, por exemplo) mantem o inicio que tinha.
func TestMeioNovoComEntradaAntigaMantemOInicio(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 2.0, "porBanco": map[string]any{
		"ITAU":   map[string]any{"saldoInicial": 0.0, "dataInicio": "2020-01-01"},
		"VOLTOU": map[string]any{"saldoInicial": 12.0, "dataInicio": "2025-03-01"},
	}})
	st, d := putListas(t, `["ITAU","VOLTOU"]`)
	esperar(t, "adicionar VOLTOU", st, 200)
	if n, _ := d["meiosNovos"].([]any); len(n) != 0 {
		t.Errorf("meiosNovos = %v", d["meiosNovos"])
	}
	if v := f.ler(t, pkEspaco, "CFG#SALDOS")["porBanco"].(map[string]any)["VOLTOU"].(map[string]any); v["dataInicio"] != "2025-03-01" || v["saldoInicial"] != 12.0 {
		t.Errorf("VOLTOU = %v", v)
	}
}

// N-T4: varios nomes novos de uma vez (importacao de planilha, IA do Lancar) e nome repetido na mesma lista: um item por nome.
func TestVariosMeiosNovosDeUmaVez(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	st, d := putListas(t, `["ITAU","A","B","A"]`)
	esperar(t, "adicionar A e B", st, 200)
	if n, _ := d["meiosNovos"].([]any); len(n) != 2 {
		t.Fatalf("meiosNovos = %v", d["meiosNovos"])
	}
	por := f.ler(t, pkEspaco, "CFG#SALDOS")["porBanco"].(map[string]any)
	for _, m := range []string{"A", "B"} {
		if e, _ := por[m].(map[string]any); e["dataInicio"] != "2026-10-05" {
			t.Errorf("%s = %v", m, por[m])
		}
	}
}

// N-T5: sem nenhuma configuracao de Saldos ainda (espaco antigo), o primeiro meio novo cria CFG#SALDOS.
func TestMeioNovoSemCfgSaldosCria(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	delete(f.itens, chave(pkEspaco, "CFG#SALDOS"))
	st, _ := putListas(t, `["ITAU","NOVO"]`)
	esperar(t, "adicionar NOVO", st, 200)
	por := f.ler(t, pkEspaco, "CFG#SALDOS")["porBanco"].(map[string]any)
	if e, _ := por["NOVO"].(map[string]any); e["dataInicio"] != "2026-10-05" {
		t.Errorf("porBanco = %v", por)
	}
}
