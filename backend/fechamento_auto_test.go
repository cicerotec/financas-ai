package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Item 38, fase A (.ai/specs/fechamento-automatico/spec.md): o mes fecha por data e mexer nele exige confirmacao explicita
// (query confirmaMesFechado=1). Os ids A-T* sao os do spec; A-T1 (a regra pura) esta em domain/fechamento_test.go.
// Nestes testes "hoje" e 16/10/2026: outubro e novembro abertos, setembro e todos os anteriores fechados por data.

func agoraOutubro16(t *testing.T) { relogioEm(t, 2026, 10, 16, 15, 0) }

const (
	dAgosto  = "2026-08-10T12:00:00.000Z"
	dOutubro = "2026-10-10T12:00:00.000Z"
)

func totalDeVersoes(f *bancoFake) int {
	n := 0
	for k := range f.itens {
		if strings.HasPrefix(k, pkHist+"|") {
			n++
		}
	}
	return n
}

func codigoDe(d doc) string { s, _ := d["codigo"].(string); return s }

func confereMesFechado(t *testing.T, d doc, banco, mes string) {
	t.Helper()
	if d["codigo"] != "mes_fechado" || d["banco"] != banco || d["mes"] != mes || d["confirmavel"] != true {
		t.Errorf("resposta do mes fechado = %v", d)
	}
	if e, _ := d["erro"].(string); !strings.Contains(e, "ate o dia 7") || !strings.Contains(e, "confirme") {
		t.Errorf("a mensagem deveria explicar o limite e pedir a confirmacao: %q", e)
	}
}

// A-T2, A-T3 e A-T4
func TestCriarEmMesFechadoExigeConfirmacao(t *testing.T) {
	agoraOutubro16(t)
	f := cenario(t)
	corpo := corpoTx(dAgosto, "ITAU", 5)

	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo)
	esperar(t, "criar em agosto sem confirmar", c, 409)
	confereMesFechado(t, d, "ITAU", "2026-08")
	if totalDeVersoes(f) != 0 {
		t.Error("recusado: nao pode gravar nem versao")
	}

	c, d = chamar(t, "own", "POST", "spaces/s1/tx", conf, corpo)
	esperar(t, "criar em agosto confirmando", c, 201)
	vs := versoesDe(t, f, d["id"].(string))
	if len(vs) != 1 || vs[0]["confirmouMesFechado"] != true || vs[0]["acao"] != "insert" {
		t.Errorf("a versao 1 deveria registrar a confirmacao: %v", vs)
	}

	// em mes aberto a confirmacao enviada sem necessidade nao e registrada
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", conf, corpoTx(dOutubro, "ITAU", 5))
	esperar(t, "criar em outubro com confirmacao desnecessaria", c, 201)
	if vs := versoesDe(t, f, d["id"].(string)); len(vs) != 1 || vs[0]["confirmouMesFechado"] != nil {
		t.Errorf("confirmar sem precisar nao grava nada: %v", vs)
	}
}

// A-T5 e A-T6
func TestEditarEmMesFechado(t *testing.T) {
	agoraOutubro16(t)
	f := cenario(t) // iddono: 2026-01-01, ITAU, valor 10
	qs := map[string]string{"de": dataDono}
	comConf := map[string]string{"de": dataDono, "confirmaMesFechado": "1"}

	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", qs, `{"valor":11}`)
	esperar(t, "mudar o valor sem confirmar", c, 409)
	confereMesFechado(t, d, "ITAU", "2026-01")
	if f.ler(t, pkEspaco, chaveTx(dataDono, "iddono"))["valor"] != 10.0 || totalDeVersoes(f) != 0 {
		t.Error("recusado: nada pode ter mudado")
	}

	// so texto, nota e tags nao mexem no saldo: passam sem confirmar e sem marca
	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/iddono", qs, `{"descricao":"so texto","nota":"n","tags":["Lazer"]}`)
	esperar(t, "editar so o texto", c, 200)
	vs := versoesDe(t, f, "iddono")
	if len(vs) != 2 || vs[1]["confirmouMesFechado"] != nil {
		t.Errorf("texto nao exige confirmacao nem deixa marca: %v", vs)
	}

	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/iddono", comConf, `{"valor":11}`)
	esperar(t, "mudar o valor confirmando", c, 200)
	vs = versoesDe(t, f, "iddono")
	if len(vs) != 3 || vs[2]["confirmouMesFechado"] != true || vs[0]["confirmouMesFechado"] != nil {
		t.Errorf("so a versao da acao confirmada leva a marca: %v", vs)
	}
}

// A-T7
func TestMudarADataParaOuDeMesFechadoExigeConfirmacao(t *testing.T) {
	agoraOutubro16(t)
	f := cenario(t)
	id := criarTx(t, "own", dOutubro, "no mes aberto", 10)
	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": dOutubro}, `{"dataEvento":"`+dAgosto+`"}`)
	esperar(t, "mover de outubro para agosto", c, 409)
	confereMesFechado(t, d, "ITAU", "2026-08")
	if f.ler(t, pkEspaco, chaveTx(dOutubro, id)) == nil {
		t.Error("o lancamento deveria ter ficado em outubro")
	}

	c, _ = chamar(t, "own", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": dOutubro, "confirmaMesFechado": "1"}, `{"dataEvento":"`+dAgosto+`"}`)
	esperar(t, "mover confirmando", c, 200)
	// agora ele esta em agosto: voltar para outubro tambem exige (o alvo de antes e fechado)
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": dAgosto}, `{"dataEvento":"`+dOutubro+`"}`)
	esperar(t, "mover de agosto para outubro", c, 409)
	confereMesFechado(t, d, "ITAU", "2026-08")
}

// A-T8 e A-T15
func TestExcluirEmMesFechadoEODesfazer(t *testing.T) {
	agoraOutubro16(t)
	f := cenario(t)
	qs := map[string]string{"de": dataDono}

	c, d := chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", qs, "")
	esperar(t, "excluir sem confirmar", c, 409)
	confereMesFechado(t, d, "ITAU", "2026-01")
	if f.ler(t, pkEspaco, chaveTx(dataDono, "iddono"))["excluidoEm"] != nil || totalDeVersoes(f) != 0 {
		t.Error("recusado: nada pode ter mudado")
	}
	c, d = chamar(t, "own", "DELETE", "spaces/s1/tx/iddono", map[string]string{"de": dataDono, "confirmaMesFechado": "1"}, "")
	esperar(t, "excluir confirmando", c, 200)
	desf, _ := d["desfazer"].(map[string]any)
	if d["ok"] != true || desf["versao"] != 1.0 {
		t.Errorf("legado: v1 (antes) e v2 (delete); o Desfazer restaura a 1: %v", d)
	}
	vs := versoesDe(t, f, "iddono")
	if len(vs) != 2 || vs[1]["acao"] != "delete" || vs[1]["confirmouMesFechado"] != true || vs[0]["confirmouMesFechado"] != nil {
		t.Errorf("versoes = %v", vs)
	}

	// com historico: o Desfazer aponta a ultima versao de antes; excluir o excluido e o inexistente nao devolvem desfazer
	id := criarTx(t, "own", dOutubro, "com historico", 10)
	editarTx(t, "own", id, dOutubro, `{"valor":11}`)
	editarTx(t, "own", id, dOutubro, `{"valor":12}`)
	_, d = chamar(t, "own", "DELETE", "spaces/s1/tx/"+id, map[string]string{"de": dOutubro}, "")
	if desf, _ := d["desfazer"].(map[string]any); desf["versao"] != 3.0 {
		t.Errorf("tres versoes antes da exclusao: desfazer.versao = 3, veio %v", d)
	}
	_, d = chamar(t, "own", "DELETE", "spaces/s1/tx/"+id, map[string]string{"de": dOutubro}, "")
	if d["desfazer"] != nil || d["ok"] != true {
		t.Errorf("excluir o excluido nao devolve desfazer: %v", d)
	}
	_, d = chamar(t, "own", "DELETE", "spaces/s1/tx/naoexiste", map[string]string{"de": dOutubro}, "")
	if d["desfazer"] != nil {
		t.Errorf("excluir o inexistente nao devolve desfazer: %v", d)
	}
}

// A-T9
func TestImportarEmMesFechado(t *testing.T) {
	agoraOutubro16(t)
	f := cenario(t)
	lote := `{"itens":[` + corpoImport("idagosto01", dAgosto, "agosto") + `,` + corpoImport("idoutub001", dOutubro, "outubro") + `]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "importar sem confirmar", c, 200)
	rej, _ := d["rejeitados"].([]any)
	if d["gravados"] != 1.0 || len(rej) != 1 || rej[0].(map[string]any)["codigo"] != "mes_fechado" {
		t.Fatalf("so o de outubro entra; o de agosto vai para rejeitados com o codigo: %v", d)
	}
	if len(versoesDe(t, f, "idagosto01")) != 0 {
		t.Error("o rejeitado nao tem versao")
	}

	c, d = chamar(t, "own", "POST", "spaces/s1/tx/batch", conf, lote)
	esperar(t, "importar confirmando", c, 200)
	if d["gravados"] != 2.0 || len(d["rejeitados"].([]any)) != 0 {
		t.Fatalf("com a confirmacao entram os dois: %v", d)
	}
	if v := versoesDe(t, f, "idagosto01"); len(v) != 1 || v[0]["confirmouMesFechado"] != true {
		t.Errorf("o de agosto precisou da confirmacao: %v", v)
	}
	if v := versoesDe(t, f, "idoutub001"); len(v) != 2 || v[1]["confirmouMesFechado"] != nil {
		t.Errorf("o de outubro nao precisou (e foi um update): %v", v)
	}
}

// A-T10 e A-T11
func TestRestaurarEmMesFechadoEExigeConfirmacaoNasVersoes(t *testing.T) {
	agoraOutubro16(t)
	f := cenario(t)
	_, d := chamar(t, "own", "POST", "spaces/s1/tx", conf, corpoTx(dAgosto, "ITAU", 5))
	id := d["id"].(string)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": dAgosto, "confirmaMesFechado": "1"}, `{"valor":9}`)
	esperar(t, "editar confirmando", c, 200)
	outro := criarTx(t, "own", dOutubro, "aberto", 5)
	editarTx(t, "own", outro, dOutubro, `{"valor":6}`)

	// as versoes avisam, antes de restaurar, que vai pedir confirmacao (e continuam restauraveis)
	_, v := versoesApi(t, "own", id, nil)
	v1 := v["versoes"].([]any)[1].(map[string]any)
	if v1["restauravel"] != true || v1["exigeConfirmacao"] != true || v1["confirmouMesFechado"] != true {
		t.Errorf("v1 em agosto: restauravel, exige confirmacao e foi criada com ela: %v", v1)
	}
	_, vo := versoesApi(t, "own", outro, nil)
	if vo1 := vo["versoes"].([]any)[1].(map[string]any); vo1["exigeConfirmacao"] != false || vo1["confirmouMesFechado"] != false {
		t.Errorf("em outubro nada exige confirmacao: %v", vo1)
	}

	c, d = restaurar(t, "own", id, 1)
	esperar(t, "restaurar sem confirmar", c, 409)
	confereMesFechado(t, d, "ITAU", "2026-08")
	if n := len(versoesDe(t, f, id)); n != 2 {
		t.Errorf("recusado: nada gravado (%d versoes)", n)
	}
	c, d = chamar(t, "own", "POST", fmt.Sprintf("spaces/s1/tx/%s/versoes/1/restaurar", id), conf, "")
	esperar(t, "restaurar confirmando", c, 200)
	vs := versoesDe(t, f, id)
	if len(vs) != 3 || vs[2]["acao"] != "restore" || vs[2]["confirmouMesFechado"] != true {
		t.Errorf("a versao restore deveria registrar a confirmacao: %v", vs)
	}
	_ = d
}

// A-T12: o conferido e absoluto, a confirmacao nao o destrava
func TestConfirmacaoNaoDestravaMesConferido(t *testing.T) {
	agoraOutubro16(t)
	cenario(t)
	c, _ := conferir(t, "own", "ITAU", "2026-03", 0)
	esperar(t, "conferir marco", c, 200)
	comConf := map[string]string{"de": dataDono, "confirmaMesFechado": "1"}
	c, d := chamar(t, "own", "PUT", "spaces/s1/tx/iddono", comConf, `{"valor":11}`)
	esperar(t, "editar em mes conferido, confirmando", c, 409)
	if d["codigo"] != "mes_conferido" {
		t.Errorf("continua mes_conferido: %v", d)
	}
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", conf, corpoTx("2026-02-10T12:00:00.000Z", "ITAU", 5))
	esperar(t, "criar em mes conferido, confirmando", c, 409)
	if d["codigo"] != "mes_conferido" || d["confirmavel"] != nil {
		t.Errorf("o conferido nao e confirmavel: %v", d)
	}
}

// A-T13: o que e anterior ao inicio do controle do banco e livre
func TestAnteriorAoInicioDoControleNaoExigeConfirmacao(t *testing.T) {
	agoraOutubro16(t)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 1.0,
		"porBanco": map[string]any{"ITAU": map[string]any{"saldoInicial": 100.0, "dataInicio": "2026-01-05"}}})
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2025-12-20T12:00:00.000Z", "ITAU", 9))
	esperar(t, "historico anterior ao inicio", c, 201)
	if vs := versoesDe(t, f, d["id"].(string)); len(vs) != 1 || vs[0]["confirmouMesFechado"] != nil {
		t.Errorf("livre: sem confirmacao e sem marca: %v", vs)
	}
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-01-10T12:00:00.000Z", "ITAU", 9))
	esperar(t, "do mes do inicio em diante", c, 409)
	confereMesFechado(t, d, "ITAU", "2026-01")
	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2025-12-20T12:00:00.000Z", "NUBANK", 9))
	esperar(t, "banco sem data de inicio: nada e isento", c, 409)
	confereMesFechado(t, d, "NUBANK", "2025-12")
}

// A-T14: a confirmacao nunca chega ao item vivo, e so a query vale
func TestConfirmacaoNaoVaiParaOItemVivoESoAQueryVale(t *testing.T) {
	agoraOutubro16(t)
	f := cenario(t)
	// o campo no corpo nao e confirmacao
	corpo := `{"dataEvento":"` + dAgosto + `","status":"PAGO","tipo":"saida","valor":5,"banco":"ITAU","confirmaMesFechado":true}`
	c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo)
	esperar(t, "confirmacao no corpo nao vale", c, 409)
	for _, valor := range []string{"0", "true", "sim", ""} {
		c, _ = chamar(t, "own", "POST", "spaces/s1/tx", map[string]string{"confirmaMesFechado": valor}, corpoTx(dAgosto, "ITAU", 5))
		esperar(t, "confirmaMesFechado="+valor, c, 409)
	}
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", conf, corpo)
	esperar(t, "confirmando pela query", c, 201)
	id := d["id"].(string)
	viv := f.ler(t, pkEspaco, chaveTx(dAgosto, id))
	for _, k := range []string{"confirmaMesFechado", "confirmouMesFechado"} {
		if viv[k] != nil {
			t.Errorf("o item vivo nao pode ter %s: %v", k, viv)
		}
	}
	if d["confirmaMesFechado"] != nil || d["confirmouMesFechado"] != nil {
		t.Errorf("a resposta tambem nao: %v", d)
	}
}

// A-T16: quem decide e o relogio do servidor: o mesmo pedido, no dia 7 e no dia 8 em Sao Paulo
func TestORelogioDoServidorDecide(t *testing.T) {
	setembro := corpoTx("2026-09-10T12:00:00.000Z", "ITAU", 5)
	t.Run("dia 7 as 23h59 em Sao Paulo", func(t *testing.T) {
		relogioEm(t, 2026, 10, 8, 2, 59) // 02h59 UTC de 08/10 = 23h59 de 07/10 em SP
		cenario(t)
		c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, setembro)
		esperar(t, "setembro ainda livre", c, 201)
	})
	t.Run("dia 8 as 0h em Sao Paulo", func(t *testing.T) {
		relogioEm(t, 2026, 10, 8, 3, 0) // 03h00 UTC = 00h00 de 08/10 em SP
		cenario(t)
		c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, setembro)
		esperar(t, "setembro fechou", c, 409)
		confereMesFechado(t, d, "ITAU", "2026-09")
	})
}

// A-T17: so o domain conhece o dia limite; o service, o controller e a raiz nao repetem o numero nem a regra. A decisao de
// "mes fechado" vive na trava (service/fechamento.go, desde a fatia 3 do item 32); os outros arquivos do service a recebem
// pela interface Travas.
func TestSoODomainConheceODiaLimite(t *testing.T) {
	filepath.Walk(".", func(caminho string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(caminho, ".go") || strings.HasSuffix(caminho, "_test.go") {
			return nil
		}
		p := filepath.ToSlash(caminho)
		if strings.HasPrefix(p, "domain/") {
			return nil
		}
		fonte, _ := os.ReadFile(caminho)
		if strings.Contains(string(fonte), "DiaLimiteMesAnterior") {
			t.Errorf("%s usa DiaLimiteMesAnterior: so o domain conhece o dia limite", p)
		}
		if strings.HasPrefix(p, "service/") && p != "service/fechamento.go" && strings.Contains(string(fonte), "MesFechadoPorData") {
			t.Errorf("%s decide o mes fechado: a regra vem da trava (Travas.Checar), nao do service", p)
		}
		return nil
	})
}
