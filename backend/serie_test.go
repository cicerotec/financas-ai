package main

import (
	"strings"
	"testing"
)

func TestValidarSerie(t *testing.T) {
	d := doc{"serie": map[string]any{"id": "abc123def456", "indice": 3.0, "intervalo": 6.0, "lixo": "x"}}
	if err := validarSerie(d); err != nil {
		t.Fatal(err)
	}
	s := d["serie"].(doc)
	if s["id"] != "abc123def456" || s["indice"] != 3.0 || s["intervalo"] != 6.0 || len(s) != 3 {
		t.Errorf("serie normalizada = %v (campos extras devem sair)", s)
	}
	n := doc{"serie": nil}
	if err := validarSerie(n); err != nil {
		t.Fatal(err)
	}
	if _, tem := n["serie"]; tem {
		t.Error("serie:null deveria remover o campo")
	}
	if err := validarSerie(doc{}); err != nil {
		t.Errorf("sem o campo nada muda: %v", err)
	}
	ok := func(id string, indice, intervalo any) map[string]any {
		return map[string]any{"id": id, "indice": indice, "intervalo": intervalo}
	}
	ruins := []any{
		"texto",
		ok("ABC123def456", 1.0, 1.0),                    // maiuscula
		ok("curto", 1.0, 1.0),                           // menos de 6
		ok(strings.Repeat("a", 41), 1.0, 1.0),           // mais de 40
		ok("abc 123 def", 1.0, 1.0),                     // espaco
		ok("abc123def456", 0.0, 1.0),                    // indice 0
		ok("abc123def456", 1000.0, 1.0),                 // indice alto
		ok("abc123def456", 1.5, 1.0),                    // indice quebrado
		ok("abc123def456", "1", 1.0),                    // indice texto
		ok("abc123def456", 1.0, 0.0),                    // intervalo 0
		ok("abc123def456", 1.0, 121.0),                  // intervalo alto
		map[string]any{"indice": 1.0, "intervalo": 1.0}, // sem id
		map[string]any{"id": "abc123def456", "intervalo": 1.0},
	}
	for i, r := range ruins {
		if err := validarSerie(doc{"serie": r}); err == nil {
			t.Errorf("caso %d (%v) deveria ser recusado", i, r)
		}
	}
}

func TestSerieNaCriacaoEdicaoELote(t *testing.T) {
	f := cenario(t)
	corpo := `{"dataEvento":"2026-10-10T12:00:00.000Z","valor":50,"tipo":"saida","status":"PREVISTO","banco":"ITAU","descricao":"Seguro",
		"serie":{"id":"serie0000abc","indice":2,"intervalo":12}}`
	st, d := chamar(t, "mem", "POST", "spaces/s1/tx", nil, corpo)
	esperar(t, "criar com serie", st, 201)
	id := d["id"].(string)
	tx := f.ler(t, pkEspaco, "TX#2026-10-10T12:00:00.000Z#"+id)
	s, _ := tx["serie"].(map[string]any)
	if s["id"] != "serie0000abc" || s["indice"] != 2.0 || s["intervalo"] != 12.0 {
		t.Errorf("serie gravada = %v", tx["serie"])
	}

	// editar outro campo preserva a serie (o PUT mescla)
	st, _ = chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-10T12:00:00.000Z"}, `{"descricao":"Seguro auto"}`)
	esperar(t, "editar", st, 200)
	if tx = f.ler(t, pkEspaco, "TX#2026-10-10T12:00:00.000Z#"+id); tx["serie"] == nil || tx["descricao"] != "Seguro auto" {
		t.Errorf("a edicao perdeu a serie: %v", tx)
	}

	// serie invalida e recusada, no POST e no PUT
	ruim := strings.Replace(corpo, `"indice":2`, `"indice":0`, 1)
	st, _ = chamar(t, "mem", "POST", "spaces/s1/tx", nil, ruim)
	esperar(t, "serie invalida no POST", st, 400)
	st, _ = chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-10T12:00:00.000Z"}, `{"serie":{"id":"x"}}`)
	esperar(t, "serie invalida no PUT", st, 400)

	// serie:null desliga
	chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-10T12:00:00.000Z"}, `{"serie":null}`)
	if _, tem := f.ler(t, pkEspaco, "TX#2026-10-10T12:00:00.000Z#"+id)["serie"]; tem {
		t.Error("serie:null deveria remover o campo")
	}

	// importacao em lote valida a serie como qualquer lancamento
	lote := `{"itens":[
		{"id":"lotes11111","dataEvento":"2026-10-11T12:00:00.000Z","valor":1,"tipo":"saida","status":"PREVISTO","banco":"ITAU","serie":{"id":"serie0000abc","indice":1,"intervalo":1}},
		{"id":"lotes22222","dataEvento":"2026-10-12T12:00:00.000Z","valor":1,"tipo":"saida","status":"PREVISTO","banco":"ITAU","serie":{"id":"X","indice":1,"intervalo":1}}]}`
	st, r := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "lote", st, 200)
	if r["gravados"] != 1.0 || len(r["rejeitados"].([]any)) != 1 {
		t.Errorf("lote = %v, esperado 1 gravado e 1 rejeitado", r)
	}
}
