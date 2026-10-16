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

func TestVersoesDaEscrita(t *testing.T) {
	antes := Doc{"id": "a", "valor": 1.0, "PK": "SPACE#s", "SK": "TX#x"}
	depois := Doc{"id": "a", "valor": 2.0}

	// lancamento novo: so a versao pedida, com o numero seguinte
	vs := VersoesDaEscrita(0, nil, depois, AcaoInsert, "quem", "t")
	if len(vs) != 1 || vs[0].Numero != 1 || vs[0].Acao != AcaoInsert || vs[0].PorSub != "quem" {
		t.Errorf("criacao: %+v", vs)
	}
	// antigo, sem historico: v1 de antes (autor desconhecido) e v2 de depois
	vs = VersoesDaEscrita(0, antes, depois, AcaoUpdate, "quem", "t")
	if len(vs) != 2 || vs[0].Numero != 1 || vs[0].Acao != AcaoInsert || vs[0].PorSub != "" || vs[0].Registro["valor"] != 1.0 ||
		vs[1].Numero != 2 || vs[1].Acao != AcaoUpdate || vs[1].PorSub != "quem" || vs[1].Registro["valor"] != 2.0 {
		t.Errorf("legado: %+v", vs)
	}
	if _, tem := vs[0].Registro["PK"]; tem {
		t.Error("o registro nao leva as chaves do banco")
	}
	// com historico: so a proxima
	vs = VersoesDaEscrita(7, antes, depois, AcaoDelete, "quem", "t")
	if len(vs) != 1 || vs[0].Numero != 8 || vs[0].Acao != AcaoDelete {
		t.Errorf("com historico: %+v", vs)
	}
}

func TestExcluidoPublicoEProtegidos(t *testing.T) {
	if Excluido(Doc{}) || Excluido(Doc{"excluidoEm": nil}) || !Excluido(Doc{"excluidoEm": "2026-01-01T00:00:00.000Z"}) {
		t.Error("so quem tem excluidoEm preenchido esta excluido")
	}
	d := Doc{"PK": "x", "SK": "y", "excluidoEm": "z", "valor": 1.0}
	if p := Publico(d); p["excluidoEm"] != nil || p["PK"] != nil || p["valor"] != 1.0 {
		t.Errorf("Publico deve tirar PK, SK e excluidoEm: %v", p)
	}
	corpo := Doc{"excluidoEm": "falso", "descricao": "x"}
	Protegidos(corpo)
	if corpo["excluidoEm"] != nil {
		t.Error("o cliente nao pode gravar nem limpar excluidoEm pelo corpo")
	}
}
