package domain

import (
	"testing"
	"time"
)

func TestDataValida(t *testing.T) {
	for s, quer := range map[string]bool{"2026-12-31": true, "2026-02-29": false, "2026-2-1": false, "": false, "2026-12-31T00:00:00Z": false, "31/12/2026": false} {
		if got := DataValida(s); got != quer {
			t.Errorf("DataValida(%q) = %v, esperado %v", s, got, quer)
		}
	}
}

func TestMeioAtivo(t *testing.T) {
	// ativo enquanto hoje < ativoAte: no proprio dia da data o meio ja deixou de valer
	for _, c := range []struct {
		ate, hoje string
		quer      bool
	}{{"", "2026-11-03", true}, {"2026-12-31", "2026-11-03", true}, {"2026-11-03", "2026-11-03", false}, {"2026-10-31", "2026-11-03", false}} {
		if got := MeioAtivo(c.ate, c.hoje); got != c.quer {
			t.Errorf("MeioAtivo(%q, %q) = %v, esperado %v", c.ate, c.hoje, got, c.quer)
		}
	}
}

func TestHojeDeUsaOFusoDoApp(t *testing.T) {
	// 01/11 02:00 UTC ainda e 31/10 em Sao Paulo
	if got := HojeDe(time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC)); got != "2026-10-31" {
		t.Errorf("HojeDe = %q", got)
	}
}

// Item 58: o meio desativado.
func TestMeioDesativadoPelaData(t *testing.T) {
	listas := Doc{"banco": []any{"A", "B"}, "ativoAte": map[string]any{"A": "2026-11-03"}}
	if !MeioDesativado(listas, "A", "2026-11-03") || !MeioDesativado(listas, "A", "2026-12-01") {
		t.Error("desativado a partir da data (inclusive)")
	}
	if MeioDesativado(listas, "A", "2026-11-02") || MeioDesativado(listas, "B", "2030-01-01") {
		t.Error("antes da data, ou sem ativoAte, o meio vale")
	}
	if !MeioNaLista(listas, "A") || MeioNaLista(listas, "C") {
		t.Error("MeioNaLista")
	}
}

func TestSoMudaCamposLivres(t *testing.T) {
	antes := Lancamento{"dataEvento": "2020-01-01T12:00:00.000Z", "valor": 5.0, "banco": "A", "status": "PAGO", "tipo": "saida", "descricao": "x", "tags": []any{}, "excluirDoTotal": false}
	livre := Lancamento{"dataEvento": "2020-01-01T12:00:00.000Z", "valor": 5.0, "banco": "A", "status": "PAGO", "tipo": "saida", "descricao": "y", "nota": "n", "tags": []any{"T"},
		"pagaFatura": nil, "aviso": nil, "faturaAjuste": nil, "oculto": false, "excluirDoTotal": false}
	if !SoMudaCamposLivres(antes, livre) {
		t.Error("descricao, nota e tags mudam; null, vazio e a mesma data sem os milissegundos nao contam")
	}
	livre["dataEvento"] = "2020-01-01T12:00:00Z"
	if !SoMudaCamposLivres(antes, livre) {
		t.Error("a mesma data em outro formato e a mesma data")
	}
	for k, v := range map[string]any{"valor": 6.0, "banco": "B", "status": "CREDITO", "tipo": "entrada", "oculto": true, "dataEvento": "2020-01-02T12:00:00.000Z", "previsto": true, "faturaAjuste": 1.0} {
		c := Lancamento{}
		for kk, vv := range antes {
			c[kk] = vv
		}
		c[k] = v
		if SoMudaCamposLivres(antes, c) {
			t.Errorf("mudar %s mexe no saldo ou na fatura", k)
		}
	}
	comRec := Lancamento{"valor": 5.0, "recorrente": map[string]any{"intervalo": 1.0}}
	if !SoMudaCamposLivres(comRec, Lancamento{"valor": 5.0, "recorrente": nil}) || SoMudaCamposLivres(Lancamento{"valor": 5.0}, comRec) {
		t.Error("tirar a marca recorrente pode; po-la, nao")
	}
}

func TestImportavelEmMeioDesativado(t *testing.T) {
	if !ImportavelEmMeioDesativado("2020-12-31T23:00:00.000Z", "2021-01-01") {
		t.Error("antes do inicio entra")
	}
	if ImportavelEmMeioDesativado("2021-01-01T03:00:00.000Z", "2021-01-01") || ImportavelEmMeioDesativado("2022-01-01T12:00:00.000Z", "2021-01-01") {
		t.Error("no instante do inicio (00:00 em Sao Paulo) e depois, nao")
	}
	if ImportavelEmMeioDesativado("2020-01-01T12:00:00.000Z", "") || ImportavelEmMeioDesativado("lixo", "2021-01-01") {
		t.Error("sem dataInicio, ou data invalida, nao")
	}
}
