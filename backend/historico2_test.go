package main

import (
	"fmt"
	"strings"
	"testing"
)

// Fase 2 do historico (.ai/specs/historico/spec.md): busca por mes, versoes com diff e restauracao. Os ids F2-T* sao
// os do spec. F2-T1 e F2-T2 (rotas e permissoes) estao em auth/authz_test.go.

// hist grava uma versao direto no banco falso, para controlar o instante (em) da alteracao.
func hist(t *testing.T, f *bancoFake, id string, n int, acao, porSub, em string, registro doc) {
	t.Helper()
	f.colocar(t, doc{"PK": pkHist, "SK": fmt.Sprintf("TX#%s#%08d", id, n), "versao": float64(n), "acao": acao,
		"porSub": porSub, "em": em, "registro": map[string]any(registro)})
}

func reg(id, descricao string, valor float64, criadoPor string) doc {
	return doc{"id": id, "dataEvento": "2026-09-10T12:00:00.000Z", "descricao": descricao, "valor": valor, "tipo": "saida",
		"status": "PAGO", "banco": "ITAU", "criadoPor": criadoPor, "criadoEm": "2026-09-10T12:05:00.000Z"}
}

func itensDe(d doc) []map[string]any {
	var r []map[string]any
	for _, x := range d["itens"].([]any) {
		r = append(r, x.(map[string]any))
	}
	return r
}

func idsDe(d doc) string {
	var ids []string
	for _, it := range itensDe(d) {
		ids = append(ids, it["id"].(string))
	}
	return strings.Join(ids, ",")
}

// criarTx cria um lancamento pela API e devolve o id.
func criarTx(t *testing.T, quem, data, descricao string, valor int) string {
	t.Helper()
	c, d := chamar(t, quem, "POST", "spaces/s1/tx", nil,
		fmt.Sprintf(`{"dataEvento":%q,"status":"PAGO","tipo":"saida","valor":%d,"banco":"ITAU","descricao":%q}`, data, valor, descricao))
	esperar(t, "criar", c, 201)
	return d["id"].(string)
}

func editarTx(t *testing.T, quem, id, de, corpo string) {
	t.Helper()
	c, _ := chamar(t, quem, "PUT", "spaces/s1/tx/"+id, map[string]string{"de": de}, corpo)
	esperar(t, "editar", c, 200)
}

func restaurar(t *testing.T, quem, id string, n int) (int, doc) {
	t.Helper()
	return chamar(t, quem, "POST", fmt.Sprintf("spaces/s1/tx/%s/versoes/%d/restaurar", id, n), nil, "")
}

func versoesApi(t *testing.T, quem, id string, qs map[string]string) (int, doc) {
	t.Helper()
	return chamar(t, quem, "GET", "spaces/s1/tx/"+id+"/versoes", qs, "")
}

const dataSet = "2026-09-10T12:00:00.000Z"

// F2-T3
func TestHistoricoExigeUmMesValido(t *testing.T) {
	cenario(t)
	for _, qs := range []map[string]string{nil, {"mes": ""}, {"mes": "2026-13"}, {"mes": "2026-9"}, {"mes": "setembro"}} {
		c, d := chamar(t, "own", "GET", "spaces/s1/historico", qs, "")
		esperar(t, fmt.Sprintf("mes %v", qs), c, 400)
		if d["erro"] == nil {
			t.Errorf("deveria explicar o erro: %v", d)
		}
	}
	c, _ := chamar(t, "own", "GET", "spaces/s1/historico", map[string]string{"mes": "2026-09"}, "")
	esperar(t, "mes valido sem nada", c, 200)
}

// F2-T4, F2-T5 e F2-T6
func TestHistoricoFiltraPeloMesDaAlteracaoNoFusoDeSaoPaulo(t *testing.T) {
	f := cenario(t)
	hist(t, f, "idset00001", 1, "insert", "own", "2026-09-05T15:00:00.000Z", reg("idset00001", "so setembro", 1, "own"))
	hist(t, f, "idout00001", 1, "insert", "own", "2026-10-05T15:00:00.000Z", reg("idout00001", "so outubro", 2, "own"))
	// 22h30 de 30/09 em Sao Paulo ja e 01/10 em UTC: a alteracao e de setembro
	hist(t, f, "idvira0001", 1, "update", "own", "2026-10-01T01:30:00.000Z", reg("idvira0001", "virada do mes", 3, "own"))
	// tres versoes do mesmo lancamento no mes: uma linha, com a ultima
	for n, dia := range []string{"06", "07", "08"} {
		hist(t, f, "idtres0001", n+1, "update", "own", "2026-09-"+dia+"T12:00:00.000Z", reg("idtres0001", fmt.Sprintf("v%d", n+1), float64(n+1), "own"))
	}
	c, d := chamar(t, "own", "GET", "spaces/s1/historico", map[string]string{"mes": "2026-09"}, "")
	esperar(t, "setembro", c, 200)
	if d["mes"] != "2026-09" || d["truncado"] != false {
		t.Errorf("resposta = %v", d)
	}
	itens := itensDe(d)
	if len(itens) != 3 {
		t.Fatalf("setembro: esperava 3 lancamentos (virada, tres, set), vieram %s", idsDe(d))
	}
	// ordem: em decrescente (a virada do mes, 01/10 UTC, e a mais nova)
	if idsDe(d) != "idvira0001,idtres0001,idset00001" {
		t.Errorf("ordem = %s", idsDe(d))
	}
	if itens[1]["versao"] != 3.0 || itens[1]["descricao"] != "v3" || itens[1]["acao"] != "update" {
		t.Errorf("uma linha por lancamento, com a ultima versao do mes: %v", itens[1])
	}
	_, d = chamar(t, "own", "GET", "spaces/s1/historico", map[string]string{"mes": "2026-10"}, "")
	if idsDe(d) != "idout00001" {
		t.Errorf("outubro so tem o de outubro (a virada e de setembro): %s", idsDe(d))
	}
}

// F2-T7
func TestHistoricoBuscaPeloNomeAntigoEPorNome(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": "USER#own", "SK": "SPACE#s1", "role": "owner", "apelido": "Ana"})
	hist(t, f, "idrenom001", 1, "insert", "own", "2026-09-05T15:00:00.000Z", reg("idrenom001", "Mercado", 1, "own"))
	hist(t, f, "idrenom001", 2, "update", "own", "2026-09-06T15:00:00.000Z", reg("idrenom001", "Padaria", 1, "own"))
	hist(t, f, "idoutro001", 1, "insert", "own", "2026-09-05T15:00:00.000Z", reg("idoutro001", "Farmacia", 1, "own"))
	_, d := chamar(t, "own", "GET", "spaces/s1/historico", map[string]string{"mes": "2026-09", "q": "MERCADO"}, "")
	itens := itensDe(d)
	if len(itens) != 1 || itens[0]["id"] != "idrenom001" || itens[0]["descricao"] != "Padaria" {
		t.Fatalf("o nome antigo acha o lancamento, que aparece com o estado mais novo: %v", itens)
	}
	if itens[0]["porNome"] != "Ana" || itens[0]["porSub"] != "own" {
		t.Errorf("deveria trazer o apelido de quem agiu: %v", itens[0])
	}
	_, d = chamar(t, "own", "GET", "spaces/s1/historico", map[string]string{"mes": "2026-09", "q": "nada disso"}, "")
	if len(itensDe(d)) != 0 {
		t.Errorf("sem correspondencia: %v", d)
	}
}

// F2-T8
func TestHistoricoParaNoTetoEAvisa(t *testing.T) {
	f := cenario(t)
	for i := 0; i < 5001; i++ {
		id := fmt.Sprintf("idmuitos%05d", i)
		hist(t, f, id, 1, "insert", "own", "2026-09-05T15:00:00.000Z", reg(id, "x", 1, "own"))
	}
	c, d := chamar(t, "own", "GET", "spaces/s1/historico", map[string]string{"mes": "2026-09"}, "")
	esperar(t, "muitas versoes", c, 200)
	if d["truncado"] != true {
		t.Error("passou do teto de 5.000 versoes: deveria avisar com truncado")
	}
	if n := len(itensDe(d)); n != 5000 {
		t.Errorf("deveria devolver o que leu (5.000), veio %d", n)
	}
}

// F2-T9
func TestMemberVeSoOHistoricoDoQueCriou(t *testing.T) {
	f := cenario(t)
	hist(t, f, "iddela0001", 1, "insert", "mem", "2026-09-05T15:00:00.000Z", reg("iddela0001", "dela", 1, "mem"))
	hist(t, f, "iddono0001", 1, "insert", "own", "2026-09-05T15:00:00.000Z", reg("iddono0001", "do dono", 1, "own"))
	_, d := chamar(t, "mem", "GET", "spaces/s1/historico", map[string]string{"mes": "2026-09"}, "")
	if idsDe(d) != "iddela0001" {
		t.Errorf("member so ve o que criou: %s", idsDe(d))
	}
	_, d = chamar(t, "own", "GET", "spaces/s1/historico", map[string]string{"mes": "2026-09"}, "")
	if len(itensDe(d)) != 2 {
		t.Errorf("owner ve tudo: %s", idsDe(d))
	}
	c, _ := versoesApi(t, "mem", "iddono0001", nil)
	esperar(t, "member nas versoes do dono", c, 403)
	c, _ = versoesApi(t, "mem", "iddela0001", nil)
	esperar(t, "member nas proprias versoes", c, 200)
	c, _ = chamar(t, "estranho", "GET", "spaces/s1/historico", map[string]string{"mes": "2026-09"}, "")
	esperar(t, "quem nao e do espaco", c, 403)
}

// F2-T10 e F2-T11
func TestVersoesDeLegadoEDeIdInexistente(t *testing.T) {
	cenario(t)
	c, d := versoesApi(t, "own", "iddono", map[string]string{"de": dataDono})
	esperar(t, "legado com a data", c, 200)
	atual, _ := d["atual"].(map[string]any)
	if d["semHistorico"] != true || len(d["versoes"].([]any)) != 0 || atual["id"] != "iddono" || atual["PK"] != nil {
		t.Errorf("legado nunca alterado: semHistorico, sem versoes e o estado atual: %v", d)
	}
	c, _ = versoesApi(t, "own", "iddono", nil)
	esperar(t, "legado sem a data nao da para achar", c, 404)
	c, _ = versoesApi(t, "own", "naoexiste", map[string]string{"de": dataDono})
	esperar(t, "id inexistente", c, 404)
	c, _ = versoesApi(t, "own", "naoexiste", nil)
	esperar(t, "id inexistente sem data", c, 404)
}

// F2-T12
func TestVersoesTrazODiffSoDoQueMudou(t *testing.T) {
	cenario(t)
	id := criarTx(t, "own", dataSet, "Mercado", 180)
	editarTx(t, "own", id, dataSet, `{"valor":187.5}`)
	c, d := versoesApi(t, "own", id, nil)
	esperar(t, "versoes", c, 200)
	vs := d["versoes"].([]any)
	if len(vs) != 2 {
		t.Fatalf("esperava 2 versoes, vieram %d", len(vs))
	}
	v2, v1 := vs[0].(map[string]any), vs[1].(map[string]any) // da mais nova para a mais antiga
	if v2["versao"] != 2.0 || v1["versao"] != 1.0 {
		t.Errorf("ordem: %v / %v", v2["versao"], v1["versao"])
	}
	diff := v2["diff"].([]any)
	if len(diff) != 1 || diff[0].(map[string]any)["campo"] != "valor" || diff[0].(map[string]any)["antes"] != 180.0 || diff[0].(map[string]any)["depois"] != 187.5 {
		t.Errorf("diff da v2 = %v", diff)
	}
	if got := v1["diff"].([]any); len(got) != 0 {
		t.Errorf("a v1 nao tem com o que comparar: %v", got)
	}
	r := v2["registro"].(map[string]any)
	if r["PK"] != nil || r["SK"] != nil || r["valor"] != 187.5 {
		t.Errorf("registro sem chaves do banco: %v", r)
	}
	if v2["porSub"] != "own" || v2["deVersao"] != nil || v2["porNome"] != "" {
		t.Errorf("campos da v2 = %v", v2)
	}
	if v1["restauravel"] != true || v1["motivo"] != "" { // a v1 tem valor diferente do atual
		t.Errorf("a v1 difere do estado atual: pode restaurar: %v", v1)
	}
	if v2["restauravel"] != false || !strings.Contains(v2["motivo"].(string), "ja esta nessa versao") { // a ultima e o estado atual
		t.Errorf("a v2 e o estado atual: nao ha o que restaurar: %v", v2)
	}
}

// F2-T13
func TestRestaurarVersaoAntigaGravaVersaoNovaEPreservaOCriador(t *testing.T) {
	f := cenario(t)
	id := criarTx(t, "mem", dataSet, "original", 10) // criado pela member
	editarTx(t, "own", id, dataSet, `{"valor":99}`)
	editarTx(t, "own", id, dataSet, `{"descricao":"mudou"}`)
	c, d := restaurar(t, "own", id, 1)
	esperar(t, "restaurar a v1", c, 200)
	if d["valor"] != 10.0 || d["descricao"] != "original" || d["criadoPor"] != "mem" || d["_seq"] == nil || d["excluidoEm"] != nil {
		t.Errorf("deveria voltar ao conteudo da v1, com criadoPor da versao: %v", d)
	}
	viv := f.ler(t, pkEspaco, chaveTx(dataSet, id))
	if viv["valor"] != 10.0 || viv["criadoPor"] != "mem" || viv["excluidoEm"] != nil {
		t.Errorf("lancamento vivo = %v", viv)
	}
	vs := versoesDe(t, f, id)
	if len(vs) != 4 {
		t.Fatalf("1 insert, 2 updates e o restore: esperava 4, vieram %d", len(vs))
	}
	confereVersao(t, vs[3], 4, "restore", "own")
	if vs[3]["deVersao"] != 1.0 || registroDe(vs[3])["valor"] != 10.0 {
		t.Errorf("o restore guarda a versao de origem e o conteudo: %v", vs[3])
	}
	for i := 0; i < 3; i++ { // o passado nao foi reescrito
		if vs[i]["versao"] != float64(i+1) {
			t.Errorf("versao %d alterada", i+1)
		}
	}
	if registroDe(vs[1])["valor"] != 99.0 {
		t.Error("a v2 continua com o valor 99")
	}
}

// F2-T14
func TestRestaurarExcluidoReativaNoMesmoId(t *testing.T) {
	f := cenario(t)
	id := criarTx(t, "own", dataSet, "apagado", 10)
	editarTx(t, "own", id, dataSet, `{"valor":20}`)
	c, _ := chamar(t, "own", "DELETE", "spaces/s1/tx/"+id, map[string]string{"de": dataSet}, "")
	esperar(t, "excluir", c, 200)
	_, d := chamar(t, "own", "GET", "spaces/s1/tx", nil, "")
	if strings.Contains(fmt.Sprint(d), id) {
		t.Fatal("excluido nao pode estar na lista")
	}
	c, d = restaurar(t, "own", id, 2) // a ultima antes da exclusao
	esperar(t, "restaurar o excluido", c, 200)
	if d["id"] != id || d["valor"] != 20.0 || d["excluidoEm"] != nil {
		t.Errorf("resposta = %v", d)
	}
	if viv := f.ler(t, pkEspaco, chaveTx(dataSet, id)); viv == nil || viv["excluidoEm"] != nil {
		t.Errorf("o lancamento deveria estar vivo de novo: %v", viv)
	}
	_, d = chamar(t, "own", "GET", "spaces/s1/tx", nil, "")
	if !strings.Contains(fmt.Sprint(d), id) {
		t.Error("o restaurado reaparece em GET /tx")
	}
	vs := versoesDe(t, f, id)
	confereVersao(t, vs[len(vs)-1], 4, "restore", "own")
}

// F2-T15
func TestRestaurarExcluidoComOutraDataTrocaAChave(t *testing.T) {
	f := cenario(t)
	outra := "2026-09-20T12:00:00.000Z"
	id := criarTx(t, "own", dataSet, "mudou de data", 10)
	editarTx(t, "own", id, dataSet, `{"dataEvento":"`+outra+`"}`) // v2: a chave vai para a data nova
	chamar(t, "own", "DELETE", "spaces/s1/tx/"+id, map[string]string{"de": outra}, "")
	if f.ler(t, pkEspaco, chaveTx(outra, id))["excluidoEm"] == nil {
		t.Fatal("o excluido esta na chave da data nova")
	}
	c, d := restaurar(t, "own", id, 1) // a v1 tem a data antiga
	esperar(t, "restaurar com outra data", c, 200)
	if d["dataEvento"] != dataSet {
		t.Errorf("dataEvento = %v", d["dataEvento"])
	}
	if f.ler(t, pkEspaco, chaveTx(outra, id)) != nil {
		t.Error("a chave da data nova (onde estava o excluido) deveria sair")
	}
	if viv := f.ler(t, pkEspaco, chaveTx(dataSet, id)); viv == nil || viv["excluidoEm"] != nil {
		t.Errorf("a chave da data da versao deveria existir, viva: %v", viv)
	}
}

// F2-T16
func TestRestaurarOQueJaEstaIgualNaoGrava(t *testing.T) {
	f := cenario(t)
	id := criarTx(t, "own", dataSet, "igual", 10)
	c, d := restaurar(t, "own", id, 1)
	esperar(t, "restaurar o que ja esta assim", c, 409)
	if d["codigo"] != "ja_nessa_versao" {
		t.Errorf("resposta = %v", d)
	}
	if n := len(versoesDe(t, f, id)); n != 1 {
		t.Errorf("nada deveria ter sido gravado: %d versoes", n)
	}
	_, v := versoesApi(t, "own", id, nil)
	v1 := v["versoes"].([]any)[0].(map[string]any)
	if v1["restauravel"] != false || !strings.Contains(v1["motivo"].(string), "ja esta nessa versao") {
		t.Errorf("a tela precisa saber que nao pode: %v", v1)
	}
}

// preparaMesConferido deixa o ITAU conferido ate marco de 2026. As escritas anteriores ja criaram o SALDO#ITAU (com o
// cache invalidado), entao a conferencia usa a version que esta no banco.
func preparaMesConferido(t *testing.T, f *bancoFake) {
	t.Helper()
	relogioFixo(t)
	ver, _ := f.ler(t, pkEspaco, "SALDO#ITAU")["version"].(float64)
	c, _ := conferir(t, "own", "ITAU", "2026-03", ver)
	esperar(t, "conferir marco", c, 200)
}

// F2-T17
func TestRestaurarVersaoEmMesConferidoBloqueiaEMostraOMotivo(t *testing.T) {
	f := cenario(t)
	jan := "2026-01-10T12:00:00.000Z"
	id := criarTx(t, "own", jan, "de janeiro", 10)
	editarTx(t, "own", id, jan, `{"valor":55}`) // v2; ainda nao conferido
	preparaMesConferido(t, f)
	c, d := restaurar(t, "own", id, 1) // a v1 (valor 10) cai em janeiro, conferido
	esperar(t, "restaurar em mes conferido", c, 409)
	if d["codigo"] != "mes_conferido" {
		t.Errorf("resposta = %v", d)
	}
	if viv := f.ler(t, pkEspaco, chaveTx(jan, id)); viv["valor"] != 55.0 || len(versoesDe(t, f, id)) != 2 {
		t.Errorf("nada deveria ter sido gravado: %v", viv)
	}
	_, v := versoesApi(t, "own", id, nil)
	v1 := v["versoes"].([]any)[1].(map[string]any)
	if v1["restauravel"] != false || !strings.Contains(v1["motivo"].(string), "conferido") {
		t.Errorf("a versao deveria vir bloqueada, com o motivo da trava: %v", v1)
	}
}

// F2-T18
func TestRestaurarComOEstadoAtualEmMesConferidoBloqueia(t *testing.T) {
	f := cenario(t)
	fev, jun := "2026-02-10T12:00:00.000Z", "2026-06-10T12:00:00.000Z"
	id := criarTx(t, "own", jun, "andou de mes", 10) // v1 em junho (aberto)
	editarTx(t, "own", id, jun, `{"dataEvento":"`+fev+`"}`)
	preparaMesConferido(t, f)
	c, d := restaurar(t, "own", id, 1) // a versao e de junho, mas o lancamento esta hoje em fevereiro (conferido)
	esperar(t, "restaurar com o estado atual conferido", c, 409)
	if d["codigo"] != "mes_conferido" {
		t.Errorf("resposta = %v", d)
	}
	if f.ler(t, pkEspaco, chaveTx(fev, id)) == nil || len(versoesDe(t, f, id)) != 2 {
		t.Error("nada deveria ter mudado")
	}
}

// Restaurar so o texto, em mes conferido, nao mexe no saldo: como na edicao, passa.
func TestRestaurarSoOTextoEmMesConferidoPassa(t *testing.T) {
	f := cenario(t)
	jan := "2026-01-10T12:00:00.000Z"
	id := criarTx(t, "own", jan, "texto antigo", 10)
	editarTx(t, "own", id, jan, `{"descricao":"texto novo"}`)
	preparaMesConferido(t, f)
	c, d := restaurar(t, "own", id, 1)
	esperar(t, "restaurar so o texto", c, 200)
	if d["descricao"] != "texto antigo" {
		t.Errorf("resposta = %v", d)
	}
}

// F2-T19 e F2-T20
func TestMemberRestauraSoOQueCriouEQueAindaExiste(t *testing.T) {
	f := cenario(t)
	meu := criarTx(t, "mem", dataSet, "meu", 10)
	editarTx(t, "mem", meu, dataSet, `{"valor":11}`)
	c, d := restaurar(t, "mem", meu, 1)
	esperar(t, "member restaura o proprio", c, 200)
	if d["valor"] != 10.0 {
		t.Errorf("resposta = %v", d)
	}

	dono := criarTx(t, "own", dataSet, "do dono", 10)
	editarTx(t, "own", dono, dataSet, `{"valor":12}`)
	c, _ = restaurar(t, "mem", dono, 1)
	esperar(t, "member restaura o do dono", c, 403)

	chamar(t, "own", "DELETE", "spaces/s1/tx/"+meu, map[string]string{"de": dataSet}, "")
	c, _ = restaurar(t, "mem", meu, 1)
	esperar(t, "member restaura o proprio, mas excluido", c, 403)
	if viv := f.ler(t, pkEspaco, chaveTx(dataSet, meu)); viv["excluidoEm"] == nil {
		t.Error("o excluido deve continuar excluido")
	}
	_, v := versoesApi(t, "mem", meu, nil)
	for _, x := range v["versoes"].([]any) {
		vv := x.(map[string]any)
		if vv["restauravel"] != false || !strings.Contains(vv["motivo"].(string), "dono") && vv["acao"] != "delete" {
			t.Errorf("para o member, tudo de um excluido vem bloqueado: %v", vv)
		}
	}
	c, _ = restaurar(t, "own", meu, 1)
	esperar(t, "o owner restaura o excluido", c, 200)
	c, _ = restaurar(t, "estranho", meu, 1)
	esperar(t, "quem nao e do espaco", c, 403)
}

// F2-T21
func TestRestaurarFazOsMesmosEfeitosDeUmaEdicao(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "SALDO#ITAU", "banco": "ITAU", "version": 1.0, "caches": map[string]any{"2026-08": 5.0}})
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil,
		fmt.Sprintf(`{"dataEvento":%q,"status":"PAGO","tipo":"saida","valor":10,"banco":"ITAU","descricao":"x","tags":["Praia"]}`, dataSet))
	esperar(t, "criar com tag", c, 201)
	id := d["id"].(string)
	editarTx(t, "own", id, dataSet, `{"valor":20,"tags":["Lazer"]}`)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#TAGS", "usadas": []any{"Lazer"}, "version": 5.0}) // a tag Praia saiu da lista
	f.colocar(t, doc{"PK": pkEspaco, "SK": "SALDO#ITAU", "banco": "ITAU", "version": 9.0})        // sem invalidoDe
	seqAntes, _ := f.ler(t, pkEspaco, "SEQ")["seq"].(float64)
	c, _ = restaurar(t, "own", id, 1)
	esperar(t, "restaurar", c, 200)
	if got := tagsDoBanco(t, f); strings.Join(got, ",") != "Lazer,Praia" {
		t.Errorf("a tag restaurada deveria voltar para a lista: %v", got)
	}
	if st := f.ler(t, pkEspaco, "SALDO#ITAU"); st["invalidoDe"] != "2026-09" {
		t.Errorf("restaurar mexeu no saldo de setembro: invalidoDe = %v", st["invalidoDe"])
	}
	if seq, _ := f.ler(t, pkEspaco, "SEQ")["seq"].(float64); seq != seqAntes+1 {
		t.Errorf("o contador de alteracoes deveria andar 1: %v -> %v", seqAntes, seq)
	}
}

// F2-T22: a trava de fatura do cartao nao existe no backend (item 33); restaurar nao e bloqueado. O teste documenta a lacuna.
func TestRestaurarEmFaturaDeCartaoConferidaNaoBloqueia(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#CARTOES", "version": 1.0, "porBanco": map[string]any{"ITAU": map[string]any{
		"diaPadrao": 25.0, "inicio": "2026-01", "faturas": map[string]any{"2026-09": map[string]any{"conferida": true, "valorBanco": 100.0}}}}})
	c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil,
		fmt.Sprintf(`{"dataEvento":%q,"status":"CREDITO","tipo":"saida","valor":10,"banco":"ITAU","descricao":"cartao"}`, dataSet))
	esperar(t, "criar no credito", c, 201)
	id := d["id"].(string)
	editarTx(t, "own", id, dataSet, `{"valor":999}`)
	c, d = restaurar(t, "own", id, 1)
	esperar(t, "restaurar numa fatura conferida (lacuna do item 33)", c, 200)
	if d["valor"] != 10.0 {
		t.Errorf("resposta = %v", d)
	}
}

// F2-T23
func TestRestaurarAVersaoDeExclusaoNaoPode(t *testing.T) {
	f := cenario(t)
	id := criarTx(t, "own", dataSet, "excluir", 10)
	chamar(t, "own", "DELETE", "spaces/s1/tx/"+id, map[string]string{"de": dataSet}, "")
	c, d := restaurar(t, "own", id, 2) // a versao delete
	esperar(t, "restaurar a exclusao", c, 409)
	if d["codigo"] != "versao_de_exclusao" {
		t.Errorf("resposta = %v", d)
	}
	_, v := versoesApi(t, "own", id, nil)
	delv := v["versoes"].([]any)[0].(map[string]any)
	if delv["acao"] != "delete" || delv["restauravel"] != false || delv["motivo"] == "" {
		t.Errorf("a versao delete vem bloqueada com o motivo: %v", delv)
	}
	if viv := f.ler(t, pkEspaco, chaveTx(dataSet, id)); viv["excluidoEm"] == nil {
		t.Error("continua excluido")
	}
}

func TestRestaurarVersaoOuLancamentoInexistente(t *testing.T) {
	cenario(t)
	id := criarTx(t, "own", dataSet, "a", 10)
	editarTx(t, "own", id, dataSet, `{"valor":11}`)
	c, d := restaurar(t, "own", id, 9)
	esperar(t, "versao que nao existe", c, 404)
	if d["erro"] != "versao nao encontrada" {
		t.Errorf("mensagem = %v", d["erro"])
	}
	c, _ = restaurar(t, "own", "naoexiste", 1)
	esperar(t, "lancamento sem historico", c, 404)
	for _, n := range []string{"0", "-1", "x", "1.5"} {
		c, _ = chamar(t, "own", "POST", "spaces/s1/tx/"+id+"/versoes/"+n+"/restaurar", nil, "")
		esperar(t, "numero de versao "+n, c, 400)
	}
}
