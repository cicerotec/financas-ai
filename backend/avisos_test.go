package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// Os testes nao dependem do ambiente em que rodam: o deploy do dev executa "go test" com AMBIENTE=dev, o que poria
// [DEV] nas mensagens esperadas sem marca. Quem testa a marca define ambienteApp por conta propria.
func init() { ambienteApp = "" }

// ---------- validacao ----------

func TestValidarAviso(t *testing.T) {
	d := doc{"aviso": map[string]any{"dias": []any{1.0, 3.0, 0.0, 3.0}, "insistir": true}}
	if err := validarAviso(d); err != nil {
		t.Fatal(err)
	}
	a := d["aviso"].(doc)
	dias := a["dias"].([]any)
	if len(dias) != 3 || dias[0] != 3.0 || dias[1] != 1.0 || dias[2] != 0.0 {
		t.Errorf("dias normalizados = %v, esperado [3 1 0] sem repeticao", dias)
	}
	if a["insistir"] != true {
		t.Error("insistir deveria continuar true")
	}

	// so insistir, sem dias, e valido
	if err := validarAviso(doc{"aviso": map[string]any{"insistir": true}}); err != nil {
		t.Errorf("so insistir deveria valer: %v", err)
	}
	// null remove o campo
	n := doc{"aviso": nil, "outro": 1}
	if err := validarAviso(n); err != nil {
		t.Fatal(err)
	}
	if _, tem := n["aviso"]; tem {
		t.Error("aviso:null deveria remover o campo")
	}
	// sem o campo, nada muda
	if err := validarAviso(doc{}); err != nil {
		t.Fatal(err)
	}

	ruins := []any{
		"texto",
		map[string]any{}, // nem dias nem insistir
		map[string]any{"dias": []any{}, "insistir": false}, // idem
		map[string]any{"dias": []any{-1.0}},
		map[string]any{"dias": []any{31.0}},
		map[string]any{"dias": []any{1.5}},
		map[string]any{"dias": []any{"1"}},
		map[string]any{"dias": "1"},
		map[string]any{"dias": []any{0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}}, // mais de 8
	}
	for i, r := range ruins {
		if err := validarAviso(doc{"aviso": r}); err == nil {
			t.Errorf("caso %d (%v) deveria ser recusado", i, r)
		}
	}
}

// ---------- decisao ----------

func TestDecidirAviso(t *testing.T) {
	casos := []struct {
		nome       string
		hoje, venc string
		dias       []int
		insistir   bool
		kind       kindAviso
		n          int
	}{
		{"3 dias antes", "2026-10-07", "2026-10-10", []int{3, 1, 0}, false, avisoAntes, 3},
		{"1 dia antes", "2026-10-09", "2026-10-10", []int{3, 1, 0}, false, avisoAntes, 1},
		{"no dia", "2026-10-10", "2026-10-10", []int{3, 1, 0}, false, avisoHoje, 0},
		{"dia fora da lista", "2026-10-08", "2026-10-10", []int{3, 1, 0}, false, semAviso, 0},
		{"antes demais", "2026-10-01", "2026-10-10", []int{3}, false, semAviso, 0},
		{"depois sem insistir", "2026-10-11", "2026-10-10", []int{3, 1, 0}, false, semAviso, 0},
		{"insistir avisa no dia mesmo sem 0", "2026-10-10", "2026-10-10", nil, true, avisoHoje, 0},
		{"insistir atrasado 1", "2026-10-11", "2026-10-10", []int{1}, true, avisoAtrasado, 1},
		{"insistir atrasado 30", "2026-11-09", "2026-10-10", nil, true, avisoAtrasado, 30},
		{"insistir passou do limite", "2026-12-20", "2026-10-10", nil, true, semAviso, 0},
		{"virada de mes", "2026-02-28", "2026-03-01", []int{1}, false, avisoAntes, 1},
		{"data invalida", "x", "2026-10-10", []int{0}, true, semAviso, 0},
	}
	for _, c := range casos {
		k, n := decidirAviso(c.hoje, c.venc, c.dias, c.insistir)
		if k != c.kind || n != c.n {
			t.Errorf("%s: (%v, %d), esperado (%v, %d)", c.nome, k, n, c.kind, c.n)
		}
	}
}

func TestDiaBRUsaOFusoDeSaoPaulo(t *testing.T) {
	// 01h UTC ainda e o dia anterior em Sao Paulo (UTC-3)
	d, err := diaBR("2026-10-10T01:00:00.000Z")
	if err != nil || d != "2026-10-09" {
		t.Errorf("diaBR = (%q, %v), esperado 2026-10-09", d, err)
	}
	d, _ = diaBR("2026-10-10T12:00:00.000Z")
	if d != "2026-10-10" {
		t.Errorf("diaBR = %q, esperado 2026-10-10", d)
	}
}

// ---------- texto ----------

func TestMoeda(t *testing.T) {
	for in, quer := range map[float64]string{
		0: "R$ 0,00", 5: "R$ 5,00", 1234.5: "R$ 1.234,50", 1234567.891: "R$ 1.234.567,89", -42.1: "-R$ 42,10", 999.999: "R$ 1.000,00",
	} {
		if got := moeda(in); got != quer {
			t.Errorf("moeda(%v) = %q, esperado %q", in, got, quer)
		}
	}
}

func TestMontarResumoOrdenaAtrasadosPrimeiro(t *testing.T) {
	linhas := []linhaAviso{
		{kind: avisoAntes, n: 3, venc: "2026-10-10", tipo: "saida", descricao: "Condomínio", valor: 800, banco: "ITAU"},
		{kind: avisoAtrasado, n: 2, venc: "2026-10-05", tipo: "saida", descricao: "Internet", valor: 120, banco: "NUBANK"},
		{kind: avisoHoje, venc: "2026-10-07", tipo: "entrada", descricao: "Aluguel", valor: 2500, banco: "ITAU"},
	}
	r := montarResumo("2026-10-07", linhas)
	ordem := []string{"Atrasado há 2 dias (05/10/2026): Internet — R$ 120,00 (NUBANK)", "A receber hoje (07/10/2026): Aluguel", "Vence em 3 dias (10/10/2026): Condomínio — R$ 800,00 (ITAU)"}
	ult := -1
	for _, trecho := range ordem {
		i := strings.Index(r, trecho)
		if i < 0 {
			t.Fatalf("resumo sem %q:\n%s", trecho, r)
		}
		if i < ult {
			t.Errorf("fora de ordem em %q:\n%s", trecho, r)
		}
		ult = i
	}
	if !strings.HasPrefix(r, "🔔 Avisos de 07/10/2026\n") {
		t.Errorf("cabecalho inesperado:\n%s", r)
	}
	if !strings.Contains(montarResumo("2026-10-07", []linhaAviso{{kind: avisoAntes, n: 1, venc: "2026-10-08", descricao: " "}}), "Vence em 1 dia (08/10/2026): (sem descrição)") {
		t.Error("singular e descricao vazia")
	}
}

// ---------- item auxiliar mantido pelas rotas de escrita ----------

func auxDe(t *testing.T, f *bancoFake, id string) doc { return f.ler(t, pkAvisos, "s1#"+id) }

func TestAvisoCriaAtualizaERemoveOItemAuxiliar(t *testing.T) {
	f := cenario(t)
	corpo := `{"dataEvento":"2026-10-10T12:00:00.000Z","valor":100,"tipo":"saida","status":"PREVISTO","banco":"ITAU",
		"descricao":"Luz","aviso":{"dias":[1,3],"insistir":true}}`
	st, d := chamar(t, "mem", "POST", "spaces/s1/tx", nil, corpo)
	esperar(t, "criar com aviso", st, 201)
	id := d["id"].(string)

	aux := auxDe(t, f, id)
	if aux == nil {
		t.Fatal("item auxiliar nao foi criado")
	}
	if aux["venc"] != "2026-10-10" || aux["insistir"] != true || aux["sid"] != "s1" {
		t.Errorf("item auxiliar = %v", aux)
	}
	if got := aux["dias"].([]any); len(got) != 2 || got[0] != 3.0 {
		t.Errorf("dias do auxiliar = %v, esperado [3 1]", got)
	}

	// mudar a data atualiza o vencimento (a chave do lancamento muda, o auxiliar acompanha)
	st, _ = chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-10T12:00:00.000Z"},
		`{"dataEvento":"2026-10-20T12:00:00.000Z"}`)
	esperar(t, "mudar data", st, 200)
	if aux = auxDe(t, f, id); aux == nil || aux["venc"] != "2026-10-20" || aux["dataEvento"] != "2026-10-20T12:00:00.000Z" {
		t.Errorf("auxiliar depois de mudar a data = %v", aux)
	}

	// aviso:null desliga
	st, _ = chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-20T12:00:00.000Z"}, `{"aviso":null}`)
	esperar(t, "desligar aviso", st, 200)
	if auxDe(t, f, id) != nil {
		t.Error("auxiliar deveria sumir quando o aviso e desligado")
	}
	tx := f.ler(t, pkEspaco, "TX#2026-10-20T12:00:00.000Z#"+id)
	if _, tem := tx["aviso"]; tem {
		t.Error("lancamento ainda guarda o campo aviso")
	}

	// ligar de novo e excluir
	chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-20T12:00:00.000Z"}, `{"aviso":{"dias":[0]}}`)
	if auxDe(t, f, id) == nil {
		t.Fatal("auxiliar deveria voltar")
	}
	st, _ = chamar(t, "own", "DELETE", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-20T12:00:00.000Z"}, "")
	esperar(t, "excluir", st, 200)
	if auxDe(t, f, id) != nil {
		t.Error("auxiliar deveria sumir quando o lancamento e excluido")
	}
}

func TestAvisoInvalidoERecusado(t *testing.T) {
	cenario(t)
	corpo := `{"dataEvento":"2026-10-10T12:00:00.000Z","valor":1,"tipo":"saida","status":"PREVISTO","banco":"ITAU","aviso":{"dias":[99]}}`
	st, _ := chamar(t, "mem", "POST", "spaces/s1/tx", nil, corpo)
	esperar(t, "aviso invalido", st, 400)
}

func TestImportacaoEmLoteSincronizaOAviso(t *testing.T) {
	f := cenario(t)
	corpo := `{"itens":[
		{"id":"loteaaaaaa","dataEvento":"2026-10-10T12:00:00.000Z","valor":1,"tipo":"saida","status":"PREVISTO","banco":"ITAU","aviso":{"dias":[0]}},
		{"id":"lotebbbbbb","dataEvento":"2026-10-11T12:00:00.000Z","valor":1,"tipo":"saida","status":"PREVISTO","banco":"ITAU"}]}`
	st, _ := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, corpo)
	esperar(t, "lote", st, 200)
	if auxDe(t, f, "loteaaaaaa") == nil {
		t.Error("item do lote com aviso deveria ter auxiliar")
	}
	if auxDe(t, f, "lotebbbbbb") != nil {
		t.Error("item do lote sem aviso nao deveria ter auxiliar")
	}
}

// ---------- rotina diaria ----------

type envio struct{ chat, texto string }

// trocaEnvio troca o envio real por um que guarda as mensagens.
func trocaEnvio(t *testing.T, falhaCom error) *[]envio {
	var enviados []envio
	antes := enviarMensagem
	enviarMensagem = func(_ context.Context, chat, texto string) error {
		if falhaCom != nil {
			return falhaCom
		}
		enviados = append(enviados, envio{chat, texto})
		return nil
	}
	t.Cleanup(func() { enviarMensagem = antes })
	return &enviados
}

func agoraBR(dia string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", dia+" 08:00", fusoBR)
	return t
}

// cenarioAvisos: dono "own" e membro "mem" com Telegram; um PREVISTO do membro vencendo em 2026-10-10.
func cenarioAvisos(t *testing.T) (*bancoFake, string) {
	f := cenario(t)
	f.colocar(t, doc{"PK": "USER#own", "SK": "SPACE#s1", "role": "owner", "telegramChatId": "111111"})
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member", "telegramChatId": "222222"})
	f.colocar(t, doc{"PK": pkEspaco, "SK": "META", "nome": "Familia", "ownerSub": "own"})
	f.colocar(t, doc{"PK": pkEspaco, "SK": "CFG#LISTAS", "banco": []any{"ITAU"}, "statusFuturo": map[string]any{"PREVISTO": true}, "version": 1.0})
	corpo := `{"dataEvento":"2026-10-10T12:00:00.000Z","valor":230.5,"tipo":"saida","status":"PREVISTO","banco":"ITAU",
		"descricao":"Conta de luz","aviso":{"dias":[3,0],"insistir":false}}`
	st, d := chamar(t, "mem", "POST", "spaces/s1/tx", nil, corpo)
	esperar(t, "criar", st, 201)
	return f, d["id"].(string)
}

func TestRotinaAvisaCriadorEDonoUmaVezCada(t *testing.T) {
	cenarioAvisos(t)
	env := trocaEnvio(t, nil)

	n, err := rodarAvisos(context.Background(), agoraBR("2026-10-07")) // 3 dias antes
	if err != nil || n != 2 {
		t.Fatalf("rodarAvisos = (%d, %v), esperado 2 resumos", n, err)
	}
	chats := map[string]string{}
	for _, e := range *env {
		chats[e.chat] = e.texto
	}
	for _, chat := range []string{"111111", "222222"} {
		if !strings.Contains(chats[chat], "Vence em 3 dias (10/10/2026): Conta de luz — R$ 230,50 (ITAU)") {
			t.Errorf("chat %s recebeu %q", chat, chats[chat])
		}
	}

	// fora dos dias configurados nao manda nada
	*env = nil
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-08")); n != 0 || len(*env) != 0 {
		t.Errorf("2026-10-08 nao e dia de aviso, mas enviou %d", n)
	}
	// no dia
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-10")); n != 2 {
		t.Errorf("no dia deveria enviar 2, enviou %d", n)
	}
	// depois do vencimento, sem insistir, para
	*env = nil
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-11")); n != 0 {
		t.Errorf("sem insistir nao avisa atrasado, enviou %d", n)
	}
}

func TestRotinaCriadorIgualAoDonoRecebeUmaSo(t *testing.T) {
	f, _ := cenarioAvisos(t)
	f.colocar(t, doc{"PK": pkEspaco, "SK": "META", "nome": "Familia", "ownerSub": "mem"}) // o membro vira dono: mesma pessoa
	env := trocaEnvio(t, nil)
	n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-10"))
	if n != 1 || len(*env) != 1 || (*env)[0].chat != "222222" {
		t.Errorf("deveria mandar 1 resumo para 222222, mandou %v", *env)
	}
}

func TestRotinaInsistirAvisaAtrasadoAteResolver(t *testing.T) {
	f, id := cenarioAvisos(t)
	chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-10T12:00:00.000Z"}, `{"aviso":{"dias":[],"insistir":true}}`)
	env := trocaEnvio(t, nil)
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-13")); n != 2 || !strings.Contains((*env)[0].texto, "Atrasado há 3 dias (10/10/2026)") {
		t.Fatalf("atrasado: %d, %v", n, *env)
	}
	// pagou: status deixa de ser evento futuro, para de avisar
	chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-10T12:00:00.000Z"}, `{"status":"PAGO"}`)
	*env = nil
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-14")); n != 0 {
		t.Errorf("depois de pago nao deve avisar, enviou %d", n)
	}
	// o auxiliar de um pago ja vencido e limpo; um pago ainda por vencer fica (se voltar a PREVISTO, volta a avisar)
	_ = f
}

func TestRotinaIgnoraOcultoExcluidoENaoFuturo(t *testing.T) {
	f, id := cenarioAvisos(t)
	env := trocaEnvio(t, nil)

	chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-10T12:00:00.000Z"}, `{"oculto":true}`)
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-10")); n != 0 {
		t.Errorf("oculto nao deve avisar, enviou %d", n)
	}
	chamar(t, "mem", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-10-10T12:00:00.000Z"}, `{"oculto":false}`)
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-10")); n != 2 {
		t.Errorf("desocultado deve voltar a avisar, enviou %d", n)
	}

	// apagado por fora (sem passar pela rota): o auxiliar orfao e limpo pela rotina
	delete(f.itens, chave(pkEspaco, "TX#2026-10-10T12:00:00.000Z#"+id))
	*env = nil
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-10")); n != 0 {
		t.Errorf("excluido nao deve avisar, enviou %d", n)
	}
	if auxDe(t, f, id) != nil {
		t.Error("auxiliar orfao deveria ter sido limpo")
	}
}

func TestRotinaSemVinculoOuSemTelegramNaoQuebra(t *testing.T) {
	f, _ := cenarioAvisos(t)
	f.colocar(t, doc{"PK": "USER#own", "SK": "SPACE#s1", "role": "owner"}) // dono sem chat id
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member"})
	env := trocaEnvio(t, nil)
	if n, err := rodarAvisos(context.Background(), agoraBR("2026-10-10")); err != nil || n != 0 || len(*env) != 0 {
		t.Errorf("sem chat id nao envia: (%d, %v, %v)", n, err, *env)
	}

	// Telegram nao configurado no servidor: encerra sem erro
	f2, _ := cenarioAvisos(t)
	_ = f2
	trocaEnvio(t, errTelegramNaoConfigurado)
	if n, err := rodarAvisos(context.Background(), agoraBR("2026-10-10")); err != nil || n != 0 {
		t.Errorf("nao configurado: (%d, %v)", n, err)
	}
	// uma falha de envio nao derruba a rotina
	trocaEnvio(t, errors.New("chat not found"))
	if n, err := rodarAvisos(context.Background(), agoraBR("2026-10-10")); err != nil || n != 0 {
		t.Errorf("falha de envio: (%d, %v)", n, err)
	}
}

func TestRotinaStatusFuturoPadraoSemConfiguracao(t *testing.T) {
	f, _ := cenarioAvisos(t)
	delete(f.itens, chave(pkEspaco, "CFG#LISTAS")) // sem config: vale o padrao PREVISTO/TRANSFERINDO/CREDITANDO
	trocaEnvio(t, nil)
	if n, _ := rodarAvisos(context.Background(), agoraBR("2026-10-10")); n != 2 {
		t.Errorf("com o padrao PREVISTO e futuro; enviou %d", n)
	}
}

// ---------- evento agendado x HTTP ----------

func TestEntradaDistingueAgendamentoDeHTTP(t *testing.T) {
	cenarioAvisos(t)
	env := trocaEnvio(t, nil)
	// o agendamento so manda {"acao":"avisos"}; a data de hoje real nao bate com o cenario, entao nada e enviado
	r, err := entrada(context.Background(), json.RawMessage(`{"acao":"avisos"}`))
	if err != nil {
		t.Fatal(err)
	}
	if m, ok := r.(doc); !ok || m["enviados"] == nil {
		t.Errorf("resposta do agendamento = %v", r)
	}
	_ = env

	// uma chamada HTTP sem token cai no handler e volta 401, mesmo que o corpo fale em "avisos"
	r, err = entrada(context.Background(), json.RawMessage(`{"requestContext":{"http":{"method":"GET"}},"rawPath":"/me","headers":{},"body":"{\"acao\":\"avisos\"}"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp, ok := r.(resp); !ok || resp.StatusCode != 401 {
		t.Errorf("HTTP sem token = %v", r)
	}
}

// ---------- vinculo do Telegram e mensagem de teste ----------

func TestPerfilGuardaOChatIDDoTelegram(t *testing.T) {
	f := cenario(t)
	st, d := chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"telegramChatId":" 123456789 "}`)
	esperar(t, "salvar chat id", st, 200)
	if d["telegramChatId"] != "123456789" {
		t.Errorf("resposta = %v", d)
	}
	if got := f.ler(t, "USER#mem", "SPACE#s1")["telegramChatId"]; got != "123456789" {
		t.Errorf("gravado = %v", got)
	}
	// junto com o apelido, na mesma chamada
	st, _ = chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"apelido":"Ana","telegramChatId":"-1001234567890"}`)
	esperar(t, "apelido e chat id", st, 200)
	v := f.ler(t, "USER#mem", "SPACE#s1")
	if v["apelido"] != "Ana" || v["telegramChatId"] != "-1001234567890" {
		t.Errorf("vinculo = %v", v)
	}
	// so o apelido nao mexe no chat id
	chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"apelido":"Bia"}`)
	if got := f.ler(t, "USER#mem", "SPACE#s1")["telegramChatId"]; got != "-1001234567890" {
		t.Errorf("chat id mudou sem querer: %v", got)
	}
	// vazio remove
	chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"telegramChatId":""}`)
	if _, tem := f.ler(t, "USER#mem", "SPACE#s1")["telegramChatId"]; tem {
		t.Error("chat id deveria ter sido removido")
	}
	// invalidos
	for _, ruim := range []string{`{"telegramChatId":"abc"}`, `{"telegramChatId":"12"}`, `{"telegramChatId":123456}`, `{"telegramChatId":"1 2 3 4 5 6"}`, `{}`} {
		if st, _ := chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, ruim); st != 400 {
			t.Errorf("%s deveria ser 400, foi %d", ruim, st)
		}
	}
	// cada pessoa so mexe no proprio
	chamar(t, "own", "PUT", "spaces/s1/perfil", nil, `{"telegramChatId":"999999"}`)
	if _, tem := f.ler(t, "USER#mem", "SPACE#s1")["telegramChatId"]; tem {
		t.Error("o dono nao pode alterar o chat id do membro")
	}
}

func TestMensagemDeTeste(t *testing.T) {
	f := cenario(t)
	env := trocaEnvio(t, nil)
	st, _ := chamar(t, "mem", "POST", "spaces/s1/avisos/teste", nil, "{}")
	esperar(t, "sem chat id", st, 400)

	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member", "telegramChatId": "222222"})
	st, _ = chamar(t, "mem", "POST", "spaces/s1/avisos/teste", nil, "{}")
	esperar(t, "com chat id", st, 200)
	if len(*env) != 1 || (*env)[0].chat != "222222" || !strings.Contains((*env)[0].texto, "Teste") {
		t.Errorf("enviado = %v", *env)
	}

	trocaEnvio(t, errTelegramNaoConfigurado)
	st, _ = chamar(t, "mem", "POST", "spaces/s1/avisos/teste", nil, "{}")
	esperar(t, "nao configurado", st, 503)
	trocaEnvio(t, errors.New("chat not found"))
	st, _ = chamar(t, "mem", "POST", "spaces/s1/avisos/teste", nil, "{}")
	esperar(t, "telegram recusou", st, 502)

	st, _ = chamar(t, "estranho", "POST", "spaces/s1/avisos/teste", nil, "{}")
	esperar(t, "fora do espaco", st, 403)
}

func TestGetMeDevolveOChatID(t *testing.T) {
	f := cenario(t)
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member", "telegramChatId": "222222"})
	st, d := chamar(t, "mem", "GET", "me", nil, "")
	esperar(t, "me", st, 200)
	esp := d["espacos"].([]any)[0].(map[string]any)
	if esp["telegramChatId"] != "222222" {
		t.Errorf("espaco = %v", esp)
	}
}

func TestMensagensDoDevTemMarcaDev(t *testing.T) {
	linhas := []linhaAviso{{kind: avisoHoje, venc: "2026-10-07", tipo: "saida", descricao: "Luz", valor: 10, banco: "ITAU"}}
	original := ambienteApp
	t.Cleanup(func() { ambienteApp = original })

	ambienteApp = "dev"
	if r := montarResumo("2026-10-07", linhas); !strings.HasPrefix(r, "🔔 [DEV] Avisos de 07/10/2026\n") {
		t.Errorf("resumo do dev sem [DEV]:\n%s", r)
	}
	f := cenario(t)
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member", "telegramChatId": "222222"})
	env := trocaEnvio(t, nil)
	chamar(t, "mem", "POST", "spaces/s1/avisos/teste", nil, "{}")
	if len(*env) != 1 || !strings.HasPrefix((*env)[0].texto, "🔔 [DEV] Teste") {
		t.Errorf("teste do dev sem [DEV]: %v", *env)
	}

	for _, amb := range []string{"prod", ""} {
		ambienteApp = amb
		if r := montarResumo("2026-10-07", linhas); strings.Contains(r, "[DEV]") || !strings.HasPrefix(r, "🔔 Avisos de 07/10/2026\n") {
			t.Errorf("ambiente %q nao deve ter [DEV]:\n%s", amb, r)
		}
	}
}
