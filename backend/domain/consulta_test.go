package domain

import (
	"strings"
	"testing"
)

// Item 53 (.ai/specs/tendencias-consultas/spec.md): o servidor valida so o formato da consulta (R3, R4, R5).

func tagsN(n int) []any {
	l := make([]any, n)
	for i := range l {
		l[i] = "t" + strings.Repeat("x", i)
	}
	return l
}

func fichas(n int) []any {
	l := make([]any, 0, n)
	for len(l) < n {
		if len(l) > 0 {
			l = append(l, Doc{"k": "op", "v": "OU"})
		}
		if len(l) < n {
			l = append(l, Doc{"k": "tag", "v": "a"})
		}
	}
	return l[:n]
}

func corpoGrupos() Doc {
	return Doc{"nome": "Mercado", "modo": "grupos", "visao": "gastos",
		"grupos": []any{Doc{"tem": []any{"mercado"}, "nao": []any{"presente"}}}}
}

func corpoExpr(f []any) Doc {
	return Doc{"nome": "Casa", "modo": "avancado", "visao": "saldo", "expressao": f}
}

func TestCorpoConsultaValido(t *testing.T) { // C-T6 (o lado que passa)
	c, err := ValidarCorpoConsulta(corpoGrupos(), true)
	if err != nil || c.Nome != "Mercado" || c.Consulta["modo"] != "grupos" || c.Consulta["visao"] != "gastos" {
		t.Fatalf("corpo valido recusado: %v %+v", err, c)
	}
	if _, ok := c.Consulta["expressao"]; ok {
		t.Error("no modo grupos nao deve haver expressao")
	}
	c, err = ValidarCorpoConsulta(corpoExpr([]any{Doc{"k": "tag", "v": "casa"}, Doc{"k": "op", "v": "E"}, Doc{"k": "("}, Doc{"k": "tag", "v": "fixo"}, Doc{"k": ")"}}), true)
	if err != nil || c.Consulta["modo"] != "avancado" {
		t.Fatalf("expressao valida recusada: %v", err)
	}
}

func TestCorpoConsultaRecusas(t *testing.T) { // C-T6, C-T5
	mut := func(f func(Doc)) Doc { d := corpoGrupos(); f(d); return d }
	casos := map[string]Doc{
		"campo desconhecido":    mut(func(d Doc) { d["total"] = 10.0 }),
		"criadoPor no corpo":    mut(func(d Doc) { d["criadoPor"] = "own" }),
		"version no POST":       mut(func(d Doc) { d["version"] = 1.0 }),
		"modo invalido":         mut(func(d Doc) { d["modo"] = "x" }),
		"visao invalida":        mut(func(d Doc) { d["visao"] = "tudo" }),
		"grupos com expressao":  mut(func(d Doc) { d["expressao"] = fichas(1) }),
		"sem nome":              mut(func(d Doc) { delete(d, "nome") }),
		"nome so espacos":       mut(func(d Doc) { d["nome"] = "   " }),
		"grupo sem tag":         mut(func(d Doc) { d["grupos"] = []any{Doc{"tem": []any{}, "nao": []any{}}} }),
		"grupo com campo extra": mut(func(d Doc) { d["grupos"] = []any{Doc{"tem": []any{"a"}, "x": 1.0}} }),
		"tag nao textual":       mut(func(d Doc) { d["grupos"] = []any{Doc{"tem": []any{1.0}}} }),
		"tag vazia":             mut(func(d Doc) { d["grupos"] = []any{Doc{"tem": []any{" "}}} }),
		"tag de 61 caracteres":  mut(func(d Doc) { d["grupos"] = []any{Doc{"tem": []any{strings.Repeat("a", 61)}}} }),
		"expressao vazia":       corpoExpr(nil),
		"ficha desconhecida":    corpoExpr([]any{Doc{"k": "tag", "v": "a"}, Doc{"k": "zzz"}}),
		"operador invalido":     corpoExpr([]any{Doc{"k": "tag", "v": "a"}, Doc{"k": "op", "v": "XOR"}}),
		"parentese com valor":   corpoExpr([]any{Doc{"k": "(", "v": "x"}, Doc{"k": "tag", "v": "a"}}),
		"ficha com campo extra": corpoExpr([]any{Doc{"k": "tag", "v": "a", "x": 1.0}}),
		"so operadores":         corpoExpr([]any{Doc{"k": "op", "v": "E"}}),
		"grupos no avancado":    func() Doc { d := corpoExpr(fichas(1)); d["grupos"] = []any{}; return d }(),
	}
	for nome, c := range casos {
		if _, err := ValidarCorpoConsulta(c, true); err == nil {
			t.Errorf("%s: deveria ser recusado", nome)
		}
	}
}

func TestCorpoConsultaLimites(t *testing.T) { // C-T7: 4, 8, 60 e 40 passam; 5, 9, 61 e 41 nao
	grupos := func(n int) Doc {
		g := make([]any, n)
		for i := range g {
			g[i] = Doc{"tem": []any{"a"}}
		}
		d := corpoGrupos()
		d["grupos"] = g
		return d
	}
	listas := func(n int) Doc {
		d := corpoGrupos()
		d["grupos"] = []any{Doc{"tem": tagsN(n)}}
		return d
	}
	nome := func(n int) Doc { d := corpoGrupos(); d["nome"] = strings.Repeat("ã", n); return d }
	passam := map[string]Doc{"4 grupos": grupos(4), "8 tags": listas(8), "60 fichas": corpoExpr(fichas(60)), "nome de 40": nome(40)}
	for k, c := range passam {
		if _, err := ValidarCorpoConsulta(c, true); err != nil {
			t.Errorf("%s deveria passar: %v", k, err)
		}
	}
	recusam := map[string]Doc{"5 grupos": grupos(5), "9 tags": listas(9), "61 fichas": corpoExpr(fichas(61)), "nome de 41": nome(41)}
	for k, c := range recusam {
		if _, err := ValidarCorpoConsulta(c, true); err == nil {
			t.Errorf("%s deveria ser recusado", k)
		}
	}
}

func TestCorpoConsultaExpressaoMalformadaEAceita(t *testing.T) { // C-T8: decisao A, so o formato
	f := []any{Doc{"k": "("}, Doc{"k": "op", "v": "E"}, Doc{"k": "tag", "v": "a"}, Doc{"k": ")"}, Doc{"k": ")"}}
	if _, err := ValidarCorpoConsulta(corpoExpr(f), true); err != nil {
		t.Fatalf("a gramatica nao e validada no servidor: %v", err)
	}
}

func TestCorpoConsultaNormalizaOQueGrava(t *testing.T) {
	d := corpoGrupos()
	d["nome"] = "  Mercado  "
	d["grupos"] = []any{Doc{"tem": []any{" mercado "}}}
	c, err := ValidarCorpoConsulta(d, true)
	if err != nil || c.Nome != "Mercado" {
		t.Fatalf("nome nao normalizado: %q %v", c.Nome, err)
	}
	g := c.Consulta["grupos"].([]any)[0].(Doc)
	if g["tem"].([]any)[0] != "mercado" || len(g["nao"].([]any)) != 0 {
		t.Errorf("grupo nao normalizado: %v", g)
	}
}

func TestCorpoConsultaPut(t *testing.T) {
	if _, err := ValidarCorpoConsulta(Doc{"nome": "x"}, false); err == nil {
		t.Error("PUT sem version deveria ser recusado")
	}
	if _, err := ValidarCorpoConsulta(Doc{"version": 1.0}, false); err == nil {
		t.Error("PUT sem nome nem consulta deveria ser recusado")
	}
	c, err := ValidarCorpoConsulta(Doc{"version": 2.0, "nome": "Novo"}, false)
	if err != nil || c.Nome != "Novo" || c.Consulta != nil || c.Versao != 2 {
		t.Errorf("renomear: %v %+v", err, c)
	}
	bloco := corpoGrupos()
	delete(bloco, "nome")
	bloco["version"] = 3.0
	if c, err = ValidarCorpoConsulta(bloco, false); err != nil || c.Consulta == nil || c.Nome != "" {
		t.Errorf("trocar a consulta: %v %+v", err, c)
	}
	parcial := Doc{"version": 1.0, "modo": "grupos"} // o bloco vem inteiro ou nao vem
	if _, err = ValidarCorpoConsulta(parcial, false); err == nil {
		t.Error("o bloco da consulta pela metade deveria ser recusado")
	}
}

func TestMesmoNomeConsulta(t *testing.T) {
	if !MesmoNomeConsulta(" Mercado ", "mercado") || MesmoNomeConsulta("a", "b") {
		t.Error("o nome e comparado sem diferenciar maiusculas nem espacos nas pontas")
	}
}
