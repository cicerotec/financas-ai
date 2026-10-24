package main

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Item 40 (specs/saldo-servidor): o saldo, o cache e a conferencia sao calculados e validados no servidor.
// Hoje, nestes testes, e 05/10/2026 (relogioFixo), entao o mes atual e 2026-10.

func saldos(t *testing.T, quem string) (int, doc) {
	t.Helper()
	return chamar(t, quem, "GET", "spaces/s1/saldos", nil, "")
}

func itemDoMeio(t *testing.T, d doc, meio string) map[string]any {
	t.Helper()
	itens, _ := d["meios"].([]any)
	for _, x := range itens {
		m := x.(map[string]any)
		if m["meio"] == meio {
			return m
		}
	}
	t.Fatalf("o meio %s nao veio na resposta: %v", meio, d)
	return nil
}

func quase(a, b float64) bool { return math.Abs(a-b) < 0.0001 }

// configuraSaldos grava CFG#SALDOS: ITAU com saldo inicial 100 desde 01/01/2026 (os tres lancamentos do cenario, de 10 cada,
// ficam em janeiro: o saldo do ITAU e 70).
func configuraSaldos(t *testing.T, f *bancoFake) {
	t.Helper()
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 1.0,
		"porBanco": map[string]any{"ITAU": map[string]any{"saldoInicial": 100.0, "dataInicio": "2026-01-01"}}})
}

func conferirMeio(t *testing.T, quem, corpo string) (int, doc) {
	t.Helper()
	return chamar(t, quem, "POST", "spaces/s1/fechamentos/conferir", nil, corpo)
}

// contaTx conta os lancamentos vivos gravados no banco falso.
func contaTx(f *bancoFake) int {
	n := 0
	for k := range f.itens {
		if strings.HasPrefix(k, pkEspaco+"|"+dePrefixoTx) {
			n++
		}
	}
	return n
}

// S-T9 e S-T11: tres meios, um fora do total e um sem configuracao; o total e a ordem da lista.
func TestSaldosDevolveMeiosTotalEMeses(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#LISTAS", "banco": []any{"ITAU", "NUBANK", "CARTAO"}, "version": 1.0})
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 2.0, "porBanco": map[string]any{
		"ITAU":   map[string]any{"saldoInicial": 100.0, "dataInicio": "2026-01-01"},
		"NUBANK": map[string]any{"saldoInicial": 5.0, "dataInicio": "2026-01-01", "foraDoTotal": true},
	}})
	lancar := func(banco string, valor float64, tipo string) {
		c, _ := chamar(t, "own", "POST", "spaces/s1/tx", conf, fmt.Sprintf(
			`{"dataEvento":"2026-02-10T12:00:00.000Z","status":"PAGO","tipo":%q,"valor":%v,"banco":%q,"descricao":"t"}`, tipo, valor, banco))
		esperar(t, "lancar", c, 201)
	}
	lancar("NUBANK", 20, "entrada")
	lancar("CARTAO", 7, "saida")

	c, d := saldos(t, "mem") // o member tambem le
	esperar(t, "member le os saldos", c, 200)
	if d["mesAtual"] != "2026-10" {
		t.Errorf("mesAtual = %v", d["mesAtual"])
	}
	itens := d["meios"].([]any)
	if len(itens) != 3 || itens[0].(map[string]any)["meio"] != "ITAU" || itens[2].(map[string]any)["meio"] != "CARTAO" {
		t.Fatalf("os meios vem na ordem da lista: %v", itens)
	}
	itau, nu, cartao := itemDoMeio(t, d, "ITAU"), itemDoMeio(t, d, "NUBANK"), itemDoMeio(t, d, "CARTAO")
	if !quase(itau["saldo"].(float64), 70) || itau["semInicio"] != false || itau["foraDoTotal"] != false {
		t.Errorf("ITAU: %v", itau)
	}
	if !quase(nu["saldo"].(float64), 25) || nu["foraDoTotal"] != true {
		t.Errorf("NUBANK: %v", nu)
	}
	if cartao["semInicio"] != true || !quase(cartao["saldo"].(float64), -7) {
		t.Errorf("CARTAO sem configuracao: %v", cartao)
	}
	tot := d["total"].(map[string]any)
	if !quase(tot["valor"].(float64), 63) || tot["meios"] != 2.0 || tot["fora"] != 1.0 {
		t.Errorf("o total soma so os que entram (ITAU 70 e CARTAO -7): %v", tot)
	}
	// S-T11: os meses do ITAU, do atual para tras, de outubro a janeiro (o inicio do controle)
	ms := itau["meses"].([]any)
	m0, m1, ultimo := ms[0].(map[string]any), ms[1].(map[string]any), ms[len(ms)-1].(map[string]any)
	if len(ms) != 10 || m0["mes"] != "2026-10" || m0["estado"] != "aberto" || m1["estado"] != "calculado" || ultimo["mes"] != "2026-01" {
		t.Errorf("meses do ITAU: %v", ms)
	}
	c, _ = chamar(t, "estranho", "GET", "spaces/s1/saldos", nil, "")
	esperar(t, "estranho nao le", c, 403)
}

// S-T12: o servidor grava o cache dos meses encerrados e a base propria.
func TestSaldosGravaOCacheDosMesesEncerrados(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	c, _ := saldos(t, "own")
	esperar(t, "saldos", c, 200)
	st := f.ler(t, pkEspaco, "SALDO#ITAU")
	caches, _ := st["caches"].(map[string]any)
	if len(caches) != 9 || caches["2026-10"] != nil || !quase(caches["2026-01"].(float64), 70) || !quase(caches["2026-09"].(float64), 70) {
		t.Errorf("de janeiro a setembro (o mes atual nunca e cache): %v", caches)
	}
	if st["base"] != "v2|100|2026-01-01|CONTAS,PAGO/CREDITANDO,PREVISTO,TRANSFERINDO" || st["version"] != 1.0 {
		t.Errorf("base e version inesperados: %v / %v", st["base"], st["version"])
	}
	// a segunda chamada nao regrava (nada mudou)
	saldos(t, "own")
	if v := f.ler(t, pkEspaco, "SALDO#ITAU")["version"]; v != 1.0 {
		t.Errorf("sem mudanca nao grava: version %v", v)
	}
	// uma escrita em fevereiro invalida o cache; a chamada seguinte recalcula e regrava
	chamar(t, "own", "POST", "spaces/s1/tx", conf, corpoTx("2026-02-20T12:00:00.000Z", "ITAU", 5))
	_, d := saldos(t, "mem")
	if !quase(itemDoMeio(t, d, "ITAU")["saldo"].(float64), 65) {
		t.Errorf("saldo depois da escrita: %v", itemDoMeio(t, d, "ITAU"))
	}
	st = f.ler(t, pkEspaco, "SALDO#ITAU")
	if st["invalidoDe"] != nil || !quase(st["caches"].(map[string]any)["2026-09"].(float64), 65) {
		t.Errorf("o cache deveria ter sido regravado e o invalidoDe limpo: %v", st)
	}
}

// S-T12: se outra escrita muda o item no meio, o servidor nao grava o cache e devolve o que calculou.
func TestSaldosNaoGravaCacheSeOutraEscritaChegouAntes(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "SALDO#ITAU", "banco": "ITAU", "version": 1.0, "invalidoDe": "2026-01"})
	f.antesDoPut = func(in *dynamodb.PutItemInput) {
		f.antesDoPut = nil
		f.colocar(t, doc{"PK": pkEspaco, "SK": "SALDO#ITAU", "banco": "ITAU", "version": 7.0, "invalidoDe": "2026-01"})
	}
	c, d := saldos(t, "own")
	esperar(t, "saldos", c, 200)
	if !quase(itemDoMeio(t, d, "ITAU")["saldo"].(float64), 70) {
		t.Errorf("o saldo calculado vem mesmo sem gravar o cache: %v", itemDoMeio(t, d, "ITAU"))
	}
	if st := f.ler(t, pkEspaco, "SALDO#ITAU"); st["version"] != 7.0 || st["caches"] != nil {
		t.Errorf("a escrita concorrente vence: %v", st)
	}
}

// S-T13: sem data de inicio o saldo vai desde o primeiro movimento e nao grava cache.
func TestSaldosSemDataDeInicioNaoGravaCache(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	_, d := saldos(t, "own")
	itau := itemDoMeio(t, d, "ITAU")
	if !quase(itau["saldo"].(float64), -30) || itau["semInicio"] != true {
		t.Errorf("saldo desde o primeiro movimento: %v", itau)
	}
	if st := f.ler(t, pkEspaco, "SALDO#ITAU"); st != nil {
		t.Errorf("sem dataInicio nao ha cache: %v", st)
	}
}

// S-T10: muitos lancamentos (varias paginas do Query) contam todos, sem o limite dos 1000. O inicio do controle aqui e em 2016 so para
// ter volume a ler; no uso real ele e recente e o que vem antes e historico (veja o teste seguinte, S-T25).
func TestSaldosComVariasPaginasDoQuery(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 1.0,
		"porBanco": map[string]any{"ITAU": map[string]any{"saldoInicial": 0.0, "dataInicio": "2016-01-01"}}})
	esperado := -30.0 // os tres do cenario (janeiro de 2026, 10 de saida cada)
	n := 0
	for ano := 2016; ano <= 2026; ano++ {
		for mes := 1; mes <= 12; mes++ {
			if ano == 2026 && mes > 9 {
				break
			}
			for d := 1; d <= 28; d += 3 {
				for k := 0; k < 4; k++ {
					valor := float64(1+(n%97)) + 0.25
					tipo := "saida"
					if n%5 == 0 {
						tipo = "entrada"
					}
					data := time.Date(ano, time.Month(mes), d, 12+k%3, 0, 0, 0, time.UTC).Format("2006-01-02T15:04:05.000Z")
					id := fmt.Sprintf("v%019d", n)
					f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + data + "#" + id, "id": id, "dataEvento": data, "status": "PAGO",
						"tipo": tipo, "valor": valor, "banco": "ITAU", "criadoPor": "own"})
					if tipo == "entrada" {
						esperado += valor
					} else {
						esperado -= valor
					}
					n++
				}
			}
		}
	}
	if n < 4000 {
		t.Fatalf("o teste precisa de varias paginas: %d lancamentos", n)
	}
	inicio := time.Now()
	c, d := saldos(t, "own")
	esperar(t, "saldos com volume", c, 200)
	itau := itemDoMeio(t, d, "ITAU")
	if !quase(itau["saldo"].(float64), math.Round(esperado*100)/100) {
		t.Errorf("saldo com %d lancamentos: %v, esperado %v", n, itau["saldo"], esperado)
	}
	if _, tem := itau["incompleto"]; tem {
		t.Error("nao existe mais o campo incompleto")
	}
	t.Logf("%d lancamentos, primeira chamada (sem cache): %v", n, time.Since(inicio))
	// a segunda le so os meses abertos e da o mesmo saldo
	c, d2 := saldos(t, "own")
	esperar(t, "saldos com cache", c, 200)
	if itemDoMeio(t, d2, "ITAU")["saldo"] != itau["saldo"] {
		t.Errorf("com cache o saldo deve ser igual: %v x %v", itemDoMeio(t, d2, "ITAU")["saldo"], itau["saldo"])
	}
}

// S-T14: conferir com o real igual ao que o servidor calcula grava o saldo do servidor.
func TestConferirGravaOSaldoDoServidor(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	c, d := conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-03","real":70}`)
	esperar(t, "conferir", c, 200)
	reg := d["conferidos"].(map[string]any)["2026-03"].(map[string]any)
	if !quase(reg["saldo"].(float64), 70) || !quase(reg["real"].(float64), 70) || reg["por"] != "own" {
		t.Errorf("registro da conferencia: %v", reg)
	}
	// o corpo antigo (banco, version, saldo) ainda e aceito
	c, _ = conferirMeio(t, "own", `{"banco":"ITAU","mes":"2026-04","version":1,"saldo":70,"real":70}`)
	esperar(t, "corpo antigo", c, 200)
}

// S-T15: o saldo do corpo e ignorado: quem forja saldo e real iguais nao trava o mes com o valor que quiser.
func TestConferirIgnoraOSaldoDoCliente(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	c, d := conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-03","version":0,"saldo":5,"real":5}`)
	esperar(t, "saldo forjado", c, 409)
	if d["codigo"] != "diferenca" || !quase(d["calculado"].(float64), 70) || !quase(d["real"].(float64), 5) || !quase(d["diferenca"].(float64), -65) {
		t.Errorf("a resposta traz o que o servidor calculou: %v", d)
	}
	if st := f.ler(t, pkEspaco, "SALDO#ITAU"); st != nil && st["conferidos"] != nil {
		t.Errorf("nada pode ficar conferido: %v", st)
	}
}

// S-T16: com diferenca e sem ajustar, nada e gravado.
func TestConferirComDiferencaSemAjustarNaoGrava(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	antes := len(f.itens)
	c, d := conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-03","real":80}`)
	esperar(t, "diferenca", c, 409)
	if d["codigo"] != "diferenca" || !quase(d["diferenca"].(float64), 10) {
		t.Errorf("diferenca esperada 10: %v", d)
	}
	if len(f.itens) != antes {
		t.Error("nada deveria ter sido gravado")
	}
}

// S-T17: ajustar cria o lancamento de ajuste pelo servidor, confirma o mes fechado por data e confere.
func TestConferirComAjusteCriaOLancamento(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	c, d := conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-03","real":80,"ajustar":true}`)
	esperar(t, "conferir com ajuste", c, 200)
	reg := d["conferidos"].(map[string]any)["2026-03"].(map[string]any)
	if !quase(reg["saldo"].(float64), 80) || !quase(reg["ajuste"].(float64), 10) {
		t.Errorf("conferido com o saldo ajustado: %v", reg)
	}
	prefixo := pkEspaco + "|" + dePrefixoTx + "2026-03-31T15:00:00.000Z#" // 12:00 em Sao Paulo, ultimo dia do mes
	var achado doc
	for k, it := range f.itens {
		if strings.HasPrefix(k, prefixo) {
			achado = f.ler(t, pkEspaco, sv(it["SK"]))
		}
	}
	if achado == nil {
		t.Fatal("o lancamento de ajuste deveria existir no ultimo dia de marco, 12:00 em Sao Paulo")
	}
	if achado["tipo"] != "entrada" || !quase(achado["valor"].(float64), 10) || achado["banco"] != "ITAU" ||
		achado["descricao"] != "Ajuste de conferência 2026-03" || achado["criadoPor"] != "own" || achado["status"] != "PAGO" {
		t.Errorf("ajuste inesperado: %v", achado)
	}
	vs := versoesDe(t, f, achado["id"].(string))
	if len(vs) != 1 || vs[0]["confirmouMesFechado"] != true {
		t.Errorf("o pedido de ajustar vale como a confirmacao de mes fechado, e fica na versao: %v", vs)
	}
	// uma saida quando o extrato e menor (abril parte de 80, ja com o ajuste de marco)
	c, d = conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-04","real":75.5,"ajustar":true}`)
	esperar(t, "conferir abril", c, 200)
	if reg := d["conferidos"].(map[string]any)["2026-04"].(map[string]any); !quase(reg["ajuste"].(float64), -4.5) {
		t.Errorf("ajuste negativo: %v", reg)
	}
}

// S-T18: ajustar num mes anterior a um conferido e recusado e nada e criado.
func TestConferirComAjusteAntesDeMesConferidoRecusa(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	c, _ := conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-03","real":70}`)
	esperar(t, "conferir marco", c, 200)
	antes := len(f.itens)
	c, d := conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-02","real":90,"ajustar":true}`)
	esperar(t, "fevereiro com marco conferido", c, 409)
	if d["codigo"] != "mes_conferido" {
		t.Errorf("deveria ser mes_conferido: %v", d)
	}
	if len(f.itens) != antes {
		t.Error("nada pode ter sido criado")
	}
}

// S-T19: os erros de hoje continuam.
func TestConferirValidacoes(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	pergunta := func(quem, corpo string) int {
		c, _ := conferirMeio(t, quem, corpo)
		return c
	}
	esperar(t, "member nao confere", pergunta("mem", `{"meio":"ITAU","mes":"2026-03","real":70}`), 403)
	esperar(t, "mes futuro", pergunta("own", `{"meio":"ITAU","mes":"2026-11","real":70}`), 400)
	esperar(t, "mes mal formado", pergunta("own", `{"meio":"ITAU","mes":"03/2026","real":70}`), 400)
	esperar(t, "sem meio", pergunta("own", `{"mes":"2026-03","real":70}`), 400)
	esperar(t, "real nao numerico", pergunta("own", `{"meio":"ITAU","mes":"2026-03","real":"x"}`), 400)
	esperar(t, "sem real", pergunta("own", `{"meio":"ITAU","mes":"2026-03"}`), 400)
	esperar(t, "marco", pergunta("own", `{"meio":"ITAU","mes":"2026-03","real":70}`), 200)
	c, d := conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-02","real":70}`)
	esperar(t, "fora de ordem", c, 409)
	if d["codigo"] != "ordem" {
		t.Errorf("deveria ser ordem: %v", d)
	}
}

// S-T20: se a diferenca ja foi resolvida (um ajuste que ficou gravado), repetir nao cria um segundo ajuste.
func TestRepetirConferenciaNaoCriaSegundoAjuste(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	c, _ := chamar(t, "own", "POST", "spaces/s1/tx", conf, `{"dataEvento":"2026-03-31T15:00:00.000Z","status":"PAGO","tipo":"entrada","valor":10,"banco":"ITAU","descricao":"Ajuste de conferência 2026-03"}`)
	esperar(t, "o ajuste que ficou gravado", c, 201)
	antes := contaTx(f)
	c, _ = conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-03","real":80,"ajustar":true}`)
	esperar(t, "repetir", c, 200)
	if contaTx(f) != antes {
		t.Errorf("nao deveria criar lancamento novo: %d -> %d", antes, contaTx(f))
	}
}

// S-T21: o cache vindo do cliente e ignorado (a rota continua, para o app de hoje nao quebrar).
func TestCacheVindoDoClienteEIgnorado(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	c, _ := chamar(t, "own", "PUT", "spaces/s1/fechamentos", nil,
		`{"banco":"ITAU","version":0,"base":"v2|100|2026-01-01|CONTAS,PAGO/CREDITANDO,PREVISTO,TRANSFERINDO","caches":{"2026-01":99999,"2026-02":99999}}`)
	esperar(t, "a rota responde como antes", c, 200)
	if st := f.ler(t, pkEspaco, "SALDO#ITAU"); st != nil && st["caches"] != nil {
		t.Errorf("o cache do cliente nao pode ser gravado: %v", st)
	}
	_, d := saldos(t, "own")
	if !quase(itemDoMeio(t, d, "ITAU")["saldo"].(float64), 70) {
		t.Errorf("o saldo nao pode sofrer o cache forjado: %v", itemDoMeio(t, d, "ITAU"))
	}
	if st := f.ler(t, pkEspaco, "SALDO#ITAU"); !quase(st["caches"].(map[string]any)["2026-01"].(float64), 70) {
		t.Errorf("o cache que fica e o do servidor: %v", st["caches"])
	}
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/fechamentos", nil, `{"banco":"ITAU","version":1,"caches":{}}`)
	esperar(t, "member continua sem permissao", c, 403)
}

// Um corpo com `banco` e outro com `meio` dao o mesmo (de-para na entrada, R0 do spec da fatura).
func TestConferirAceitaBancoOuMeio(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	c, _ := conferirMeio(t, "own", `{"banco":"ITAU","mes":"2026-03","real":70}`)
	esperar(t, "com banco", c, 200)
}

// O ponto de partida pelo mes conferido nao muda o resultado de hoje.
func TestSaldoPartindoDoMesConferidoEIgualAoCompleto(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	configuraSaldos(t, f)
	_, antes := saldos(t, "own")
	conferirMeio(t, "own", `{"meio":"ITAU","mes":"2026-03","real":70}`)
	chamar(t, "own", "POST", "spaces/s1/tx", conf, corpoTx("2026-05-10T12:00:00.000Z", "ITAU", 12))
	_, depois := saldos(t, "own")
	a, b := itemDoMeio(t, antes, "ITAU")["saldo"].(float64), itemDoMeio(t, depois, "ITAU")["saldo"].(float64)
	if !quase(a, 70) || !quase(b, 58) {
		t.Errorf("saldos %v -> %v", a, b)
	}
	ms := itemDoMeio(t, depois, "ITAU")["meses"].([]any)
	if estado := ms[7].(map[string]any)["estado"]; estado != "conferido" {
		t.Errorf("marco deveria estar conferido: %v", ms)
	}
}

// S-T25: o inicio do controle existe para NAO calcular desde o comeco: com dez anos de historico e o controle comecando em 01/01/2026, o servidor le
// so a partir dessa data (a chave de data limita o Query) e o que e anterior nao entra na conta nem na leitura da tabela.
func TestSaldosLeSoDoInicioDoControleEmDiante(t *testing.T) {
	relogioFixo(t)
	f := cenario(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#SALDOS", "version": 1.0,
		"porBanco": map[string]any{"ITAU": map[string]any{"saldoInicial": 100.0, "dataInicio": "2026-01-01"}}})
	antigos, recentes := 0, 0
	for ano := 2016; ano <= 2026; ano++ {
		for mes := 1; mes <= 9; mes++ {
			for d := 1; d <= 28; d += 3 {
				data := time.Date(ano, time.Month(mes), d, 12, 0, 0, 0, time.UTC).Format("2006-01-02T15:04:05.000Z")
				id := fmt.Sprintf("h%019d", antigos+recentes)
				f.colocar(t, doc{"PK": pkEspaco, "SK": dePrefixoTx + data + "#" + id, "id": id, "dataEvento": data, "status": "PAGO",
					"tipo": "saida", "valor": 1.0, "banco": "ITAU", "criadoPor": "own"})
				if ano < 2026 {
					antigos++
				} else {
					recentes++
				}
			}
		}
	}
	f.lidosNoQuery = 0
	c, d := saldos(t, "own")
	esperar(t, "saldos", c, 200)
	// 100 (inicial) - 30 (os tres do cenario, em janeiro) - os lancamentos de 2026 (1 cada); os de 2016 a 2025 nao contam
	if want := 100.0 - 30 - float64(recentes); !quase(itemDoMeio(t, d, "ITAU")["saldo"].(float64), want) {
		t.Errorf("saldo = %v, esperado %v", itemDoMeio(t, d, "ITAU")["saldo"], want)
	}
	if f.lidosNoQuery > recentes+3+10 {
		t.Errorf("o servidor leu %d itens da tabela; so os %d de 2026 em diante (mais os do cenario) deveriam ser lidos, e nao os %d anteriores", f.lidosNoQuery, recentes+3, antigos)
	}
	t.Logf("%d itens no historico antigo, %d de 2026: o Query leu %d", antigos, recentes, f.lidosNoQuery)
}
