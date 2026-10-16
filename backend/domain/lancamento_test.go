package domain

import "testing"

func TestEhDono(t *testing.T) {
	reg := Doc{"criadoPor": "sub-ela"}
	if !EhDono(reg, "sub-ela") {
		t.Error("o criador deve ser dono")
	}
	if EhDono(reg, "sub-outro") {
		t.Error("outro usuario nao e dono")
	}
	if EhDono(Doc{}, "sub-ela") {
		t.Error("registro sem criadoPor nao e de ninguem")
	}
	if EhDono(Doc{"criadoPor": ""}, "") {
		t.Error("sub vazio nunca e dono, nem de registro com criadoPor vazio")
	}
	if EhDono(Doc{"criadoPor": 123}, "123") {
		t.Error("criadoPor com tipo errado nao conta")
	}
}

func TestIdentidadeEProtegidos(t *testing.T) {
	d := Doc{"id": "abc", "dataEvento": "2026-01-01T10:00:00.000Z", "PK": "x", "SK": "y", "criadoPor": "z", "criadoEm": "w", "valor": 1.0}
	if got := Identidade(d); got != "2026-01-01T10:00:00.000Z#abc" {
		t.Errorf("Identidade = %q", got)
	}
	Protegidos(d)
	if len(d) != 2 || d["valor"] != 1.0 || d["dataEvento"] == nil {
		t.Errorf("Protegidos deveria tirar so PK, SK, id, criadoEm e criadoPor: %v", d)
	}
}

func TestAlvoDeEMudaSaldo(t *testing.T) {
	// 01/10 02:00 UTC ainda e 30/09 em Sao Paulo
	a, ok := AlvoDe(Doc{"banco": "ITAU", "dataEvento": "2026-10-01T02:00:00.000Z"})
	if !ok || a.Banco != "ITAU" || a.Mes != "2026-09" {
		t.Errorf("AlvoDe = %+v (ok=%v), esperado ITAU 2026-09", a, ok)
	}
	if _, ok := AlvoDe(Doc{"dataEvento": "2026-10-01T02:00:00.000Z"}); ok {
		t.Error("sem banco nao ha alvo")
	}
	antes := Doc{"valor": 10.0, "tipo": "saida", "status": "PAGO", "banco": "ITAU", "dataEvento": "2026-10-01T12:00:00.000Z"}
	depois := CopiaRasa(antes)
	depois["descricao"] = "so texto"
	if MudaSaldo(antes, depois) {
		t.Error("editar descricao nao mexe no saldo")
	}
	depois["valor"] = 11.0
	if !MudaSaldo(antes, depois) {
		t.Error("mudar o valor mexe no saldo")
	}
}
