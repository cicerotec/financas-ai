package main

import (
	"context"
	"strings"
	"testing"
)

// Item 46, fase A (.ai/specs/aviso-fatura/spec.md): aviso no Telegram para a fatura do cartao.

const avisoFatura = `{"vencimento":"2026-11-05","aviso":{"dias":[1,0],"insistir":true}}`

// cenarioAvisoFatura: o cenario do cartao com o dono vinculado ao Telegram.
func cenarioAvisoFatura(t *testing.T) *bancoFake {
	t.Helper()
	f := cenarioCartao(t)
	f.colocar(t, doc{"PK": "USER#own", "SK": "SPACE#s1", "role": "owner", "telegramChatId": "111111"})
	f.colocar(t, doc{"PK": pkEspaco, "SK": "META", "nome": "Familia", "ownerSub": "own"})
	return f
}

func auxFatura(t *testing.T, f *bancoFake, ym string) doc {
	t.Helper()
	return f.ler(t, pkAvisos, "s1#fatura#CARTAO-A#"+ym)
}

// G-T1 e G-T2: o aviso valido, o invalido e o aviso sem vencimento.
func TestAvisoDaFaturaValidacao(t *testing.T) {
	f := cenarioAvisoFatura(t)
	for nome, corpo := range map[string]string{
		"dia 31":           `{"vencimento":"2026-11-05","aviso":{"dias":[31]}}`,
		"sete itens":       `{"vencimento":"2026-11-05","aviso":{"dias":[0,1,2,3,4,5,6]}}`,
		"nao e objeto":     `{"vencimento":"2026-11-05","aviso":"1"}`,
		"vazio":            `{"vencimento":"2026-11-05","aviso":{}}`,
		"sem vencimento":   `{"aviso":{"dias":[1]}}`,
		"vencimento limpo": `{"vencimento":null,"aviso":{"dias":[1]}}`,
	} {
		c, _ := putFatura(t, "2026-10", corpo)
		esperar(t, nome, c, 400)
	}
	if faturaGravada(t, f, "2026-10") != nil {
		t.Fatal("nada pode ser gravado")
	}
	c, _ := putFatura(t, "2026-10", avisoFatura)
	esperar(t, "aviso com vencimento na mesma chamada", c, 200)
	g := faturaGravada(t, f, "2026-10")
	if g["vencimento"] != "2026-11-05" || g["aviso"] == nil {
		t.Errorf("gravado: %v", g)
	}
	c, _ = putFatura(t, "2026-10", `{"aviso":null}`)
	esperar(t, "desligar", c, 200)
	if _, tem := faturaGravada(t, f, "2026-10")["aviso"]; tem {
		t.Error("aviso:null remove o campo")
	}
	if faturaGravada(t, f, "2026-10")["vencimento"] != "2026-11-05" {
		t.Error("desligar o aviso nao mexe no vencimento")
	}
	// o membro nao grava
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/cartoes/CARTAO-A/faturas/2026-10", nil, avisoFatura)
	esperar(t, "member", c, 403)
}

// G-T3 a G-T5: o item auxiliar acompanha o aviso e o vencimento.
func TestAvisoDaFaturaMantemOItemAuxiliar(t *testing.T) {
	f := cenarioAvisoFatura(t)
	if auxFatura(t, f, "2026-10") != nil {
		t.Fatal("sem aviso nao ha item")
	}
	putFatura(t, "2026-10", avisoFatura)
	a := auxFatura(t, f, "2026-10")
	if a == nil || a["venc"] != "2026-11-05" || a["insistir"] != true || a["meio"] != "CARTAO-A" || a["ym"] != "2026-10" || a["tipo"] != "fatura" {
		t.Fatalf("item auxiliar: %v", a)
	}
	// mudar so o vencimento atualiza o item
	putFatura(t, "2026-10", `{"vencimento":"2026-11-09"}`)
	if a := auxFatura(t, f, "2026-10"); a == nil || a["venc"] != "2026-11-09" {
		t.Errorf("vencimento novo: %v", a)
	}
	// limpar o vencimento tira o aviso e o item
	putFatura(t, "2026-10", `{"vencimento":null}`)
	if auxFatura(t, f, "2026-10") != nil {
		t.Error("sem vencimento o item sai")
	}
	if _, tem := faturaGravada(t, f, "2026-10")["aviso"]; tem {
		t.Error("sem vencimento o aviso sai junto")
	}
	// ligar e depois desligar o aviso
	putFatura(t, "2026-10", avisoFatura)
	putFatura(t, "2026-10", `{"aviso":null}`)
	if auxFatura(t, f, "2026-10") != nil {
		t.Error("aviso desligado: o item sai")
	}
}

// G-T7 e G-T10: a rotina diaria avisa 1 dia antes, no dia, depois (insistir) e para depois de 60 dias.
func TestRotinaAvisaAFatura(t *testing.T) {
	f := cenarioAvisoFatura(t)
	putFatura(t, "2026-10", avisoFatura)
	env := trocaEnvio(t, nil)

	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-11-03")); n != 0 || len(*env) != 0 {
		t.Fatalf("dois dias antes nao avisa: %v", *env)
	}
	n, _ := rodarAvisos(context.Background(), agoraBR("2026-11-04"))
	if n != 1 || len(*env) != 1 || (*env)[0].chat != "111111" ||
		!strings.Contains((*env)[0].texto, "Vence em 1 dia (05/11/2026): Fatura out/26 · CARTAO-A — R$ 135,50 (CARTAO-A)") {
		t.Fatalf("1 dia antes: %d %v", n, *env)
	}
	*env = nil
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-11-05")); n != 1 || !strings.Contains((*env)[0].texto, "Vence hoje (05/11/2026): Fatura out/26") {
		t.Fatalf("no dia: %d %v", n, *env)
	}
	*env = nil
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-11-08")); n != 1 || !strings.Contains((*env)[0].texto, "Atrasado há 3 dias (05/11/2026): Fatura out/26") {
		t.Fatalf("insistindo: %d %v", n, *env)
	}
	// passados 60 dias de atraso o item sai e nada e enviado
	*env = nil
	if n, _ := rodarAvisos(context.Background(), agoraBR("2027-01-05")); n != 0 || len(*env) != 0 {
		t.Errorf("61 dias depois: %v", *env)
	}
	if auxFatura(t, f, "2026-10") != nil {
		t.Error("o item sai depois de 60 dias de atraso")
	}
}

// G-T8: a fatura paga nao avisa; desfeito o pagamento, volta.
func TestRotinaNaoAvisaFaturaPaga(t *testing.T) {
	f := cenarioAvisoFatura(t)
	putFatura(t, "2026-10", avisoFatura)
	env := trocaEnvio(t, nil)
	marca := `,"pagaFatura":{"meio":"CARTAO-A","ym":"2026-10"}`
	_, p := chamar(t, "own", "POST", "spaces/s1/tx", nil, pagamento("ITAU", 135.5, hoje24, marca))
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-11-04")); n != 0 || len(*env) != 0 {
		t.Fatalf("fatura paga nao avisa: %v", *env)
	}
	if auxFatura(t, f, "2026-10") == nil {
		t.Error("o item fica, para voltar a avisar se o pagamento for desfeito")
	}
	chamar(t, "own", "DELETE", "spaces/s1/tx/"+p["id"].(string), map[string]string{"de": hoje24}, "")
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-11-04")); n != 1 {
		t.Errorf("pagamento desfeito: a fatura volta a avisar (%d)", n)
	}
}

// Aviso sem o mesmo vencimento no item (item velho): a rotina confere a configuracao de verdade.
func TestRotinaConfereAFaturaDeVerdade(t *testing.T) {
	f := cenarioAvisoFatura(t)
	putFatura(t, "2026-10", avisoFatura)
	// o dono desliga o aviso direto na configuracao (sem passar pelo comando): o item sobra, mas nada e enviado e ele sai
	cfg := f.ler(t, pkEspaco, "CFG#CARTOES")
	delete(cfg["porBanco"].(map[string]any)["CARTAO-A"].(map[string]any)["faturas"].(map[string]any)["2026-10"].(map[string]any), "aviso")
	f.colocar(t, cfg)
	env := trocaEnvio(t, nil)
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-11-04")); n != 0 || len(*env) != 0 {
		t.Fatalf("sem aviso na configuracao: %v", *env)
	}
	if auxFatura(t, f, "2026-10") != nil {
		t.Error("o item velho sai")
	}
}

// G-T9: um dia sem aviso a disparar le so os itens de AVISOS#ATIVOS (nenhuma compra, nenhum lancamento).
func TestRotinaDeUmDiaSemAvisoNaoLeNada(t *testing.T) {
	f := cenarioAvisoFatura(t)
	putFatura(t, "2026-10", avisoFatura)
	trocaEnvio(t, nil)
	f.lidosNoQuery = 0
	rodarAvisos(context.Background(), agoraBR("2026-10-30"))
	if f.lidosNoQuery != 1 {
		t.Errorf("itens lidos = %d, esperado 1 (so o item do aviso)", f.lidosNoQuery)
	}
}

// G-T10: a fatura e um lancamento no mesmo resumo, uma mensagem so.
func TestResumoJuntaFaturaELancamento(t *testing.T) {
	cenarioAvisoFatura(t)
	putFatura(t, "2026-10", avisoFatura)
	corpo := `{"dataEvento":"2026-11-05T12:00:00.000Z","valor":480,"tipo":"saida","status":"PAGO","banco":"ITAU","descricao":"Condominio","aviso":{"dias":[1],"insistir":false}}`
	c, _ := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpo)
	esperar(t, "criar", c, 201)
	env := trocaEnvio(t, nil)
	n, _ := rodarAvisos(context.Background(), agoraBR("2026-11-04"))
	if n != 1 || len(*env) != 1 || !strings.Contains((*env)[0].texto, "Fatura out/26") || !strings.Contains((*env)[0].texto, "Condominio") {
		t.Errorf("uma mensagem com os dois: %d %v", n, *env)
	}
}

// G-T11: GET /futuros traz o aviso na linha da fatura.
func TestFuturosTrazOAvisoDaFatura(t *testing.T) {
	cenarioAvisoFatura(t)
	putFatura(t, "2026-10", `{"vencimento":"2026-11-05"}`)
	if umaFatura(t, faturasDeFuturos(t), "CARTAO-A", "2026-10")["aviso"] != nil {
		t.Error("sem aviso: null")
	}
	putFatura(t, "2026-10", `{"aviso":{"dias":[1,0],"insistir":true}}`)
	a, _ := umaFatura(t, faturasDeFuturos(t), "CARTAO-A", "2026-10")["aviso"].(map[string]any)
	if a == nil || a["insistir"] != true || len(a["dias"].([]any)) != 2 {
		t.Errorf("aviso na linha: %v", a)
	}
}

// ---------- item 48: aviso padrao do cartao, copiado para a fatura ao definir o primeiro vencimento ----------

const avisoPadrao = `{"dias":[1,0],"insistir":true}`

// G-T13 e G-T14: o padrao do cartao (valido, invalido, null) e a leitura.
func TestAvisoPadraoDoCartao(t *testing.T) {
	f := cenarioAvisoFatura(t)
	for nome, corpo := range map[string]string{
		"dia 31":       `{"aviso":{"dias":[31]}}`,
		"nao e objeto": `{"aviso":"1"}`,
		"vazio":        `{"aviso":{}}`,
	} {
		c, _ := putCartao(t, "CARTAO-A", corpo)
		esperar(t, nome, c, 400)
	}
	c, _ := putCartao(t, "CARTAO-A", `{"aviso":`+avisoPadrao+`}`)
	esperar(t, "padrao valido", c, 200)
	if cartaoGravado(t, f, "CARTAO-A")["aviso"] == nil {
		t.Fatal("padrao gravado")
	}
	_, l := chamar(t, "own", "GET", "spaces/s1/cartoes", nil, "")
	var a1 any
	for _, x := range l["cartoes"].([]any) {
		m := x.(map[string]any)
		if m["meio"] == "CARTAO-A" {
			a1 = m["aviso"]
		}
		if m["meio"] == "CARTAO-B" && m["aviso"] != nil {
			t.Errorf("CARTAO-B sem padrao: %v", m["aviso"])
		}
	}
	if a, _ := a1.(map[string]any); a == nil || a["insistir"] != true || len(a["dias"].([]any)) != 2 {
		t.Errorf("GET /cartoes traz o padrao: %v", a1)
	}
	c, _ = putCartao(t, "CARTAO-A", `{"aviso":null}`)
	esperar(t, "limpar o padrao", c, 200)
	if _, tem := cartaoGravado(t, f, "CARTAO-A")["aviso"]; tem {
		t.Error("aviso:null remove o padrao")
	}
	c, _ = chamar(t, "mem", "PUT", "spaces/s1/cartoes/CARTAO-A", nil, `{"aviso":`+avisoPadrao+`}`)
	esperar(t, "member", c, 403)
}

// G-T15 e G-T16: o padrao e copiado so ao definir o primeiro vencimento; depois nao volta.
func TestAvisoPadraoECopiadoNoPrimeiroVencimento(t *testing.T) {
	f := cenarioAvisoFatura(t)
	putCartao(t, "CARTAO-A", `{"aviso":`+avisoPadrao+`}`)
	if _, tem := faturaGravada(t, f, "2026-10")["aviso"]; faturaGravada(t, f, "2026-10") != nil && tem {
		t.Fatal("definir o padrao nao mexe em fatura nenhuma")
	}
	c, _ := putFatura(t, "2026-10", `{"vencimento":"2026-11-05"}`)
	esperar(t, "primeiro vencimento", c, 200)
	g := faturaGravada(t, f, "2026-10")
	a, _ := g["aviso"].(map[string]any)
	if a == nil || a["insistir"] != true || len(a["dias"].([]any)) != 2 {
		t.Fatalf("aviso copiado: %v", g)
	}
	if x := auxFatura(t, f, "2026-10"); x == nil || x["venc"] != "2026-11-05" {
		t.Errorf("item auxiliar criado junto: %v", x)
	}
	if a, _ := umaFatura(t, faturasDeFuturos(t), "CARTAO-A", "2026-10")["aviso"].(map[string]any); a == nil {
		t.Error("Futuros ja traz o aviso")
	}
	// trocar so a data nao copia de novo; desligar e trocar a data tambem nao
	putFatura(t, "2026-10", `{"aviso":null}`)
	putFatura(t, "2026-10", `{"vencimento":"2026-11-09"}`)
	if _, tem := faturaGravada(t, f, "2026-10")["aviso"]; tem {
		t.Error("desligado e data trocada: o aviso nao volta")
	}
	// apagar o vencimento e definir de novo e um primeiro vencimento outra vez: copia de novo
	putFatura(t, "2026-10", `{"vencimento":null}`)
	putFatura(t, "2026-10", `{"vencimento":"2026-11-05"}`)
	if faturaGravada(t, f, "2026-10")["aviso"] == nil {
		t.Error("sem vencimento e sem aviso, o proximo vencimento copia o padrao")
	}
}

// G-T17: o aviso do corpo vale e nao ha copia; sem padrao no cartao nao copia nada.
func TestAvisoPadraoNaoSobrepoeOQueVemNoCorpo(t *testing.T) {
	f := cenarioAvisoFatura(t)
	putCartao(t, "CARTAO-A", `{"aviso":`+avisoPadrao+`}`)
	putFatura(t, "2026-10", `{"vencimento":"2026-11-05","aviso":{"dias":[7],"insistir":false}}`)
	a, _ := faturaGravada(t, f, "2026-10")["aviso"].(map[string]any)
	if a == nil || a["insistir"] != false || len(a["dias"].([]any)) != 1 {
		t.Errorf("vale o do corpo: %v", a)
	}
	putFatura(t, "2026-10", `{"vencimento":null}`)
	putFatura(t, "2026-10", `{"vencimento":"2026-11-05","aviso":null}`)
	if _, tem := faturaGravada(t, f, "2026-10")["aviso"]; tem {
		t.Error("aviso:null no corpo: sem aviso e sem copia")
	}
	// cartao sem padrao: nada e copiado
	c, _ := chamar(t, "own", "PUT", "spaces/s1/cartoes/CARTAO-A/faturas/2026-09", nil, `{"vencimento":"2026-10-05"}`)
	esperar(t, "outra fatura", c, 200)
	putCartao(t, "CARTAO-A", `{"aviso":null}`)
	putFatura(t, "2026-10", `{"vencimento":null}`)
	putFatura(t, "2026-10", `{"vencimento":"2026-11-05"}`)
	if _, tem := faturaGravada(t, f, "2026-10")["aviso"]; tem {
		t.Error("sem padrao no cartao, nada e copiado")
	}
}

// G-T18: mudar o padrao depois nao altera as faturas ja copiadas.
func TestMudarOPadraoNaoAlteraFaturasCopiadas(t *testing.T) {
	f := cenarioAvisoFatura(t)
	putCartao(t, "CARTAO-A", `{"aviso":`+avisoPadrao+`}`)
	putFatura(t, "2026-10", `{"vencimento":"2026-11-05"}`)
	putCartao(t, "CARTAO-A", `{"aviso":{"dias":[7],"insistir":false}}`)
	a, _ := faturaGravada(t, f, "2026-10")["aviso"].(map[string]any)
	if a == nil || a["insistir"] != true || len(a["dias"].([]any)) != 2 {
		t.Errorf("a fatura ja copiada fica como estava: %v", a)
	}
	putCartao(t, "CARTAO-A", `{"aviso":null}`)
	if faturaGravada(t, f, "2026-10")["aviso"] == nil {
		t.Error("apagar o padrao nao tira o aviso da fatura")
	}
}

// G-T19: ler a fatura (inclusive as antigas) nao grava nada.
func TestLerAFaturaNaoGrava(t *testing.T) {
	f := cenarioAvisoFatura(t)
	putCartao(t, "CARTAO-A", `{"aviso":`+avisoPadrao+`}`)
	antes := f.ler(t, pkEspaco, "CFG#CARTOES")["version"]
	for _, ym := range []string{"2026-07", "2026-08", "2026-09", "2026-10", "2026-11"} {
		fatura(t, "own", "CARTAO-A", ym)
	}
	chamar(t, "own", "GET", "spaces/s1/cartoes", nil, "")
	faturasDeFuturos(t)
	if f.ler(t, pkEspaco, "CFG#CARTOES")["version"] != antes {
		t.Error("nenhuma leitura grava a configuracao")
	}
	if faturaGravada(t, f, "2026-09") != nil {
		t.Error("as faturas antigas continuam sem nada")
	}
}
