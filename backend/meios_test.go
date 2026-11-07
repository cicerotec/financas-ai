package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
)

// Item 43 (.ai/specs/meios-de-pagamento/spec.md): o dono desativa um meio a partir de uma data (CFG#LISTAS.ativoAte) e remove so o que nunca foi
// usado, conferido pelo servidor ao clicar. O passado nao muda.

// cenarioMeios monta Listas com seis meios: ITAU (usado por vivos), VR-NOVO (sem uso), CITIBANK (cartao, sem uso), SO-EXCLUIDO (so um lancamento
// excluido logicamente), SO-PREVISTO (so um previsto) e SODEXO (sem uso). Hoje e 2026-11-03.
func cenarioMeios(t *testing.T) *bancoFake {
	t.Helper()
	relogioEm(t, 2026, 11, 3, 15, 0)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#LISTAS", "version": 1.0,
		"banco": []any{"ITAU", "VR-NOVO", "CITIBANK", "SO-EXCLUIDO", "SO-PREVISTO", "SODEXO"}})
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#CARTOES", "version": 1.0, "porBanco": map[string]any{"CITIBANK": map[string]any{"diaPadrao": 5.0}}})
	mk := func(id, banco string, extra doc) {
		d := doc{"PK": pkEspaco, "SK": dePrefixoTx + "2026-09-10T12:00:00.000Z#" + id, "id": id, "dataEvento": "2026-09-10T12:00:00.000Z",
			"status": "PAGO", "tipo": "saida", "valor": 5.0, "banco": banco, "descricao": id, "tags": []any{}, "criadoPor": "own"}
		for k, v := range extra {
			d[k] = v
		}
		f.colocar(t, d)
	}
	f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2019-01-01T12:00:00.000Z#antigo", "id": "antigo", "dataEvento": "2019-01-01T12:00:00.000Z",
		"status": "PAGO", "tipo": "saida", "valor": 5.0, "banco": "ITAU", "descricao": "antigo", "tags": []any{}, "criadoPor": "own"}) // o primeiro da varredura
	mk("exc", "SO-EXCLUIDO", doc{"excluidoEm": "2026-09-11T12:00:00.000Z"})
	mk("prev", "SO-PREVISTO", doc{"previsto": true})
	return f
}

func listasDoMeio(t *testing.T, sub string) map[string]doc {
	t.Helper()
	c, d := chamar(t, sub, "GET", "spaces/s1/cfg/listas", nil, "")
	esperar(t, "GET listas", c, 200)
	res := map[string]doc{}
	for _, x := range d["meios"].([]any) {
		m := doc(x.(map[string]any))
		res[m["nome"].(string)] = m
	}
	if _, tem := d["ativoAte"]; tem {
		t.Error("o mapa cru ativoAte nao sai na leitura (R3)")
	}
	return res
}

// M-T1: PUT /meios/{meio}: data futura, passada, hoje, null e formato invalido.
func TestMeioDesativarEReativar(t *testing.T) {
	f := cenarioMeios(t)
	casos := []struct {
		corpo     string
		status    int
		ativo     bool
		ativoAte  string
		descricao string
	}{
		{`{"ativoAte":"2026-12-31"}`, 200, true, "2026-12-31", "data futura: ainda ativo"},
		{`{"ativoAte":"2026-10-31"}`, 200, false, "2026-10-31", "data passada: inativo"},
		{`{"ativoAte":"2026-11-03"}`, 200, false, "2026-11-03", "hoje: ja deixou de valer"},
	}
	for _, c := range casos {
		st, d := chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, c.corpo)
		esperar(t, c.descricao, st, c.status)
		if d["ativo"] != c.ativo || d["ativoAte"] != c.ativoAte || d["meio"] != "SODEXO" {
			t.Errorf("%s: %v", c.descricao, d)
		}
	}
	st, d := chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, `{"ativoAte":null}`)
	esperar(t, "reativar", st, 200)
	if d["ativo"] != true || d["ativoAte"] != nil {
		t.Errorf("reativar: %v", d)
	}
	if _, tem := f.ler(t, pkEspaco, "CFG#LISTAS")["ativoAte"]; tem {
		t.Error("reativar tira a chave: sem mapa vazio gravado")
	}
	for _, ruim := range []string{`{"ativoAte":"2026-13-01"}`, `{"ativoAte":"amanha"}`, `{"ativoAte":20261231}`, `{}`} {
		st, _ = chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, ruim)
		esperar(t, "corpo invalido "+ruim, st, 400)
	}
	st, _ = chamar(t, "own", "PUT", "spaces/s1/meios/NAO-EXISTE", nil, `{"ativoAte":"2026-12-31"}`)
	esperar(t, "meio fora da lista", st, 404)
}

// M-T2: a leitura traz ativo e ativoAte por meio; o mapa cru nao aparece.
func TestMeiosNaLeituraDeListas(t *testing.T) {
	cenarioMeios(t)
	chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, `{"ativoAte":"2026-10-31"}`)
	chamar(t, "own", "PUT", "spaces/s1/meios/VR-NOVO", nil, `{"ativoAte":"2026-12-31"}`)
	m := listasDoMeio(t, "own")
	if m["SODEXO"]["ativo"] != false || m["SODEXO"]["ativoAte"] != "2026-10-31" {
		t.Errorf("SODEXO: %v", m["SODEXO"])
	}
	if m["VR-NOVO"]["ativo"] != true || m["VR-NOVO"]["ativoAte"] != "2026-12-31" {
		t.Errorf("VR-NOVO: %v", m["VR-NOVO"])
	}
	if m["ITAU"]["ativo"] != true || m["ITAU"]["ativoAte"] != nil || m["CITIBANK"]["cartao"] != true {
		t.Errorf("ITAU e CITIBANK: %v %v", m["ITAU"], m["CITIBANK"])
	}
}

func listasCorpo(t *testing.T, f *bancoFake, banco []string, extra string) string {
	t.Helper()
	v := f.ler(t, pkEspaco, "CFG#LISTAS")["version"].(float64)
	nomes, _ := json.Marshal(banco)
	return fmt.Sprintf(`{"version":%v,"banco":%s%s}`, v, nomes, extra)
}

var todosOsMeios = []string{"ITAU", "VR-NOVO", "CITIBANK", "SO-EXCLUIDO", "SO-PREVISTO", "SODEXO"}

// M-T3: o PUT de Listas ignora ativoAte do corpo e mantem o gravado.
func TestPutDeListasIgnoraAtivoAte(t *testing.T) {
	f := cenarioMeios(t)
	chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, `{"ativoAte":"2026-10-31"}`)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/cfg/listas", nil, listasCorpo(t, f, todosOsMeios, `,"ativoAte":{"SODEXO":"2099-01-01","ITAU":"2026-01-01"}`))
	esperar(t, "gravar listas", c, 200)
	ate, _ := f.ler(t, pkEspaco, "CFG#LISTAS")["ativoAte"].(map[string]any)
	if len(ate) != 1 || ate["SODEXO"] != "2026-10-31" {
		t.Errorf("o gravado continua: %v", ate)
	}
	c, _ = chamar(t, "own", "PUT", "spaces/s1/cfg/listas", nil, listasCorpo(t, f, todosOsMeios, ""))
	esperar(t, "gravar listas sem ativoAte no corpo", c, 200)
	if ate, _ := f.ler(t, pkEspaco, "CFG#LISTAS")["ativoAte"].(map[string]any); ate["SODEXO"] != "2026-10-31" {
		t.Errorf("o PUT sem ativoAte nao pode apagar o gravado: %v", ate)
	}
}

// M-T4: o PUT de Listas nao tira nome de banco.
func TestPutDeListasNaoTiraMeio(t *testing.T) {
	f := cenarioMeios(t)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/cfg/listas", nil, listasCorpo(t, f, []string{"ITAU", "VR-NOVO"}, ""))
	esperar(t, "tirar nomes pelo PUT", c, 400)
	if b := f.ler(t, pkEspaco, "CFG#LISTAS")["banco"].([]any); len(b) != len(todosOsMeios) {
		t.Errorf("a lista nao muda: %v", b)
	}
	// sem a chave banco no corpo, a lista gravada continua
	v := f.ler(t, pkEspaco, "CFG#LISTAS")["version"].(float64)
	c, _ = chamar(t, "own", "PUT", "spaces/s1/cfg/listas", nil, fmt.Sprintf(`{"version":%v}`, v))
	esperar(t, "corpo sem banco", c, 200)
	if b := f.ler(t, pkEspaco, "CFG#LISTAS")["banco"].([]any); len(b) != len(todosOsMeios) {
		t.Errorf("sem banco no corpo, a lista continua: %v", b)
	}
}

// M-T5: acrescentar um meio pelo PUT de Listas continua valendo, e ele nasce ativo.
func TestPutDeListasAcrescentaMeio(t *testing.T) {
	f := cenarioMeios(t)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/cfg/listas", nil, listasCorpo(t, f, append(append([]string{}, todosOsMeios...), "NOVO"), ""))
	esperar(t, "acrescentar", c, 200)
	if m := listasDoMeio(t, "own")["NOVO"]; m == nil || m["ativo"] != true {
		t.Errorf("o meio novo nasce ativo: %v", m)
	}
}

func capturaLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

// M-T6, M-T7, M-T8, M-T10: o DELETE recusa o meio com lancamento (vivo, excluido ou previsto) e o que e cartao, e responde 404 ao desconhecido.
func TestRemoverMeioRecusas(t *testing.T) {
	f := cenarioMeios(t)
	for _, c := range []struct{ meio, codigo string }{{"ITAU", "MEIO_EM_USO"}, {"SO-EXCLUIDO", "MEIO_EM_USO"}, {"SO-PREVISTO", "MEIO_EM_USO"}, {"CITIBANK", "MEIO_E_CARTAO"}} {
		st, d := chamar(t, "own", "DELETE", "spaces/s1/meios/"+c.meio, nil, "")
		esperar(t, "remover "+c.meio, st, 409)
		if d["codigo"] != c.codigo {
			t.Errorf("%s: %v", c.meio, d)
		}
		if strings.Contains(fmt.Sprint(d["erro"]), "lancamento(s)") {
			t.Errorf("a mensagem nao leva contagem: %v", d["erro"])
		}
	}
	if b := f.ler(t, pkEspaco, "CFG#LISTAS")["banco"].([]any); len(b) != len(todosOsMeios) {
		t.Errorf("nada muda nas recusas: %v", b)
	}
	st, _ := chamar(t, "own", "DELETE", "spaces/s1/meios/NAO-EXISTE", nil, "")
	esperar(t, "meio desconhecido", st, 404)
}

// M-T9 e M-T16: remover o meio sem uso apaga o nome e o ativoAte, e a leitura mostra a lista nova.
func TestRemoverMeioSemUso(t *testing.T) {
	f := cenarioMeios(t)
	chamar(t, "own", "PUT", "spaces/s1/meios/VR-NOVO", nil, `{"ativoAte":"2026-12-31"}`)
	chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, `{"ativoAte":"2026-10-31"}`)
	st, _ := chamar(t, "own", "DELETE", "spaces/s1/meios/VR-NOVO", nil, "")
	esperar(t, "remover sem uso", st, 204)
	listas := f.ler(t, pkEspaco, "CFG#LISTAS")
	if b := listas["banco"].([]any); len(b) != len(todosOsMeios)-1 {
		t.Errorf("banco: %v", b)
	}
	ate, _ := listas["ativoAte"].(map[string]any)
	if _, tem := ate["VR-NOVO"]; tem || ate["SODEXO"] != "2026-10-31" {
		t.Errorf("so a chave do meio removido sai do ativoAte: %v", ate)
	}
	if _, tem := listasDoMeio(t, "own")["VR-NOVO"]; tem {
		t.Error("a leitura nao traz o meio removido")
	}
	st, _ = chamar(t, "own", "DELETE", "spaces/s1/meios/SODEXO", nil, "")
	esperar(t, "remover o ultimo com ativoAte", st, 204)
	if _, tem := f.ler(t, pkEspaco, "CFG#LISTAS")["ativoAte"]; tem {
		t.Error("sem meios com data, o mapa some")
	}
}

// M-T11 e M-T12: a varredura para no primeiro achado e o log traz o custo, sem o nome; abrir Listas nao le lancamentos.
func TestRemoverMeioParaNoPrimeiroAchadoELogaOCusto(t *testing.T) {
	f := cenarioMeios(t)
	for i := 0; i < 600; i++ { // mais de uma pagina de 500 itens, de outro meio, depois do primeiro lancamento do ITAU
		id := fmt.Sprintf("f%03d", i)
		f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + "2020-01-01T12:00:00.000Z#" + id, "id": id, "dataEvento": "2020-01-01T12:00:00.000Z",
			"status": "PAGO", "tipo": "saida", "valor": 1.0, "banco": "OUTRO", "descricao": id, "tags": []any{}, "criadoPor": "own"})
	}
	buf := capturaLog(t)
	f.lidosNoQuery = 0
	st, _ := chamar(t, "own", "DELETE", "spaces/s1/meios/ITAU", nil, "")
	esperar(t, "em uso", st, 409)
	if f.lidosNoQuery != 500 {
		t.Errorf("itens lidos = %d, esperado 500 (uma pagina: o primeiro lancamento ja e do ITAU)", f.lidosNoQuery)
	}
	f.lidosNoQuery = 0
	st, _ = chamar(t, "own", "DELETE", "spaces/s1/meios/VR-NOVO", nil, "")
	esperar(t, "sem uso", st, 204)
	if f.lidosNoQuery < 600 {
		t.Errorf("o meio sem uso le a particao inteira: %d itens", f.lidosNoQuery)
	}
	saida := buf.String()
	for _, quer := range []string{"meios: remover paginas=1 itens=1 ", "resultado=em_uso", "paginas=2 ", "resultado=ok"} {
		if !strings.Contains(saida, quer) {
			t.Errorf("o log deveria ter %q:\n%s", quer, saida)
		}
	}
	if strings.Contains(saida, "VR-NOVO") || strings.Contains(saida, "ITAU") {
		t.Errorf("o nome do meio nao vai ao log:\n%s", saida)
	}
	f.lidosNoQuery = 0
	chamar(t, "own", "GET", "spaces/s1/cfg/listas", nil, "")
	if f.lidosNoQuery != 0 {
		t.Errorf("abrir Listas nao le lancamentos: %d", f.lidosNoQuery)
	}
}

// M-T13: so o dono desativa e remove; o member le o ativo.
func TestMeiosSoDono(t *testing.T) {
	f := cenarioMeios(t)
	for _, r := range []struct{ m, corpo string }{{"PUT", `{"ativoAte":"2026-12-31"}`}, {"DELETE", ""}} {
		for _, sub := range []string{"mem", "estranho"} {
			st, _ := chamar(t, sub, r.m, "spaces/s1/meios/VR-NOVO", nil, r.corpo)
			esperar(t, sub+" "+r.m, st, 403)
		}
	}
	if b := f.ler(t, pkEspaco, "CFG#LISTAS")["banco"].([]any); len(b) != len(todosOsMeios) {
		t.Errorf("nada muda: %v", b)
	}
	chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, `{"ativoAte":"2026-10-31"}`)
	if m := listasDoMeio(t, "mem")["SODEXO"]; m["ativo"] != false {
		t.Errorf("o member ve o ativo: %v", m)
	}
}

// M-T14: o servidor nao valida lancamento contra o ativoAte: lancar num meio inativo funciona.
func TestLancarNumMeioInativoContinuaValendo(t *testing.T) {
	f := cenarioMeios(t)
	chamar(t, "own", "PUT", "spaces/s1/meios/SODEXO", nil, `{"ativoAte":"2026-10-31"}`)
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", conf, corpoTx("2026-11-02T12:00:00.000Z", "SODEXO", 7))
	esperar(t, "lancar no inativo", c, 201)
	if d["banco"] != "SODEXO" {
		t.Errorf("lancamento: %v", d)
	}
	// agora ele tem uso: so desativar
	st, _ := chamar(t, "own", "DELETE", "spaces/s1/meios/SODEXO", nil, "")
	esperar(t, "remover depois de usado", st, 409)
	if ate, _ := f.ler(t, pkEspaco, "CFG#LISTAS")["ativoAte"].(map[string]any); ate["SODEXO"] != "2026-10-31" {
		t.Errorf("o ativoAte continua: %v", ate)
	}
}
