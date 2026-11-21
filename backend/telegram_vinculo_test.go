package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Spec telegram-vinculo, fase 2: o link que gera o codigo (R1 a R3), o webhook publico do bot (R4 a R10) e o estado dos canais em GET /me (R14).
// Todos os valores sao ficticios.

const (
	subAna       = "3f2a9c41-7b1d-4e08-9a55-0c6d2e8b1f73"
	subBia       = "9e1b7d20-52c3-4f6a-8d14-a7b3c5e96f02"
	segredoTeste = "segredo_de_teste-1"
	chatNovo     = int64(555000111)
	chatAntigo   = "111000222"

	respostaNaoVale   = "Este link não vale mais. Volte ao app e toque em Ligar Telegram de novo."
	respostaNaoLigou  = "Não consegui ligar agora. Volte ao app e toque em Ligar Telegram de novo."
	respostaSoAvisos  = "Este bot só envia avisos. Para ligar, use o app; para parar, desligue em Listas, Avisos."
	respostaOutroChat = "Este espaço foi ligado a outro chat do Telegram. Se não foi você, desligue em Listas, Avisos."
)

// botFalso e o bot do Telegram dos testes: o que o servidor precisa dele (segredo, usuario e o aviso ao chat antigo).
type botFalso struct {
	usuario     string
	pronto      error
	segredo     string
	erroSegredo error
	enviados    []envio
	falhaEnvio  error
}

func (b *botFalso) Pronto(context.Context) (string, error) {
	if b.pronto != nil {
		return "", b.pronto
	}
	return b.usuario, nil
}
func (b *botFalso) SegredoDoWebhook(context.Context) (string, error) { return b.segredo, b.erroSegredo }
func (b *botFalso) EnviarTexto(_ context.Context, chat, texto string) error {
	b.enviados = append(b.enviados, envio{chat, texto})
	return b.falhaEnvio
}

// cenarioVinculo: o espaco s1 com a Ana (membro, apelido "Ana") e a Bia (membro, sem apelido), e o bot configurado.
func cenarioVinculo(t *testing.T) (*bancoFake, *botFalso) {
	t.Helper()
	f := cenario(t)
	f.colocar(t, doc{"PK": "USER#" + subAna, "SK": "SPACE#s1", "role": "member", "apelido": "Ana"})
	f.colocar(t, doc{"PK": "USER#" + subBia, "SK": "SPACE#s1", "role": "member"})
	bot := &botFalso{usuario: "financas_dev_bot", segredo: segredoTeste}
	srvTeste.usarBot(bot)
	return f, bot
}

// gerarLink pede o link e devolve o status, a resposta e o codigo (o que vem depois de ?start=).
func gerarLink(t *testing.T, sub string) (status int, d doc, codigo string) {
	t.Helper()
	status, d = chamar(t, sub, "POST", "spaces/s1/telegram/vinculo", nil, "")
	if url, ok := d["url"].(string); ok {
		_, codigo, _ = strings.Cut(url, "?start=")
	}
	return status, d, codigo
}

// updateTelegram monta o JSON de uma atualizacao de mensagem.
func updateTelegram(id int, chat int64, tipo, texto string) string {
	b, _ := json.Marshal(map[string]any{"update_id": id, "message": map[string]any{"text": texto, "chat": map[string]any{"id": chat, "type": tipo}}})
	return string(b)
}

func eventoWebhook(corpo string, cab map[string]string) req {
	var r req
	r.RawPath = "/telegram/webhook"
	r.Body = corpo
	r.Headers = cab
	r.RequestContext.HTTP.Method = "POST"
	r.RequestContext.HTTP.SourceIP = "203.0.113.9"
	return r
}

func comSegredo(extra map[string]string) map[string]string {
	m := map[string]string{"x-telegram-bot-api-secret-token": segredoTeste}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func webhook(t *testing.T, corpo string) resp {
	t.Helper()
	var rt rotulo
	res, err := srvTeste.atender(context.Background(), eventoWebhook(corpo, comSegredo(nil)), &rt)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// respostaDoBot le o corpo de um 200 do webhook: {"method":"sendMessage","chat_id":n,"text":"..."}; vazio se for {}.
func respostaDoBot(t *testing.T, res resp) (chat int64, texto string, responde bool) {
	t.Helper()
	if res.StatusCode != 200 {
		t.Fatalf("status %d (%s), esperado 200", res.StatusCode, res.Body)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(res.Body), &m); err != nil {
		t.Fatalf("corpo %q: %v", res.Body, err)
	}
	if len(m) == 0 {
		return 0, "", false
	}
	if m["method"] != "sendMessage" {
		t.Fatalf("corpo %s: esperado method sendMessage", res.Body)
	}
	c, _ := m["chat_id"].(float64)
	txt, _ := m["text"].(string)
	return int64(c), txt, true
}

func semAcessoAoBanco(t *testing.T, f *bancoFake, nome string, antesL, antesE int) {
	t.Helper()
	if f.leituras != antesL || f.escritas != antesE {
		t.Errorf("%s: o banco foi acessado (%d leituras, %d escritas a mais), esperado nenhum acesso", nome, f.leituras-antesL, f.escritas-antesE)
	}
}

func itensDeVinculo(f *bancoFake, sub string) int {
	n := 0
	for k := range f.itens {
		if strings.HasPrefix(k, "USER#"+sub+"|LINK#") {
			n++
		}
	}
	return n
}

// ---------- V-T1: gerar o link ----------

func TestGerarLinkGravaUmItemSoComOHash(t *testing.T) {
	f, bot := cenarioVinculo(t)
	antes := f.escritas
	st, d, codigo := gerarLink(t, subAna)
	esperar(t, "gerar", st, 200)
	if want := "https://t.me/" + bot.usuario + "?start=" + codigo; d["url"] != want || len(codigo) != 43 {
		t.Errorf("url = %v, esperado %s com um codigo de 43 caracteres", d["url"], want)
	}
	if d["expiraEm"] != "2026-02-05T15:10:00Z" { // o relogio dos testes: 15:00 + 10 minutos
		t.Errorf("expiraEm = %v, esperado 10 minutos depois de agora", d["expiraEm"])
	}
	if f.escritas-antes != 1 {
		t.Errorf("gerar custou %d gravacoes, esperada 1", f.escritas-antes)
	}
	if itensDeVinculo(f, subAna) != 1 {
		t.Fatalf("esperado um item LINK# da Ana, ha %d", itensDeVinculo(f, subAna))
	}
	item := f.ler(t, "USER#"+subAna, "LINK#telegram")
	if item["sid"] != "s1" || item["canal"] != "telegram" {
		t.Errorf("item = %v", item)
	}
	if h, _ := item["hash"].(string); len(h) != 64 || strings.Contains(h, codigo) {
		t.Errorf("hash = %q: devia ser o SHA-256 em hexadecimal, nunca o codigo", h)
	}
	if exp, _ := item["expiraEm"].(float64); int64(exp) != relogioTeste.Agora().Add(domain.ValidadeVinculo).Unix() {
		t.Errorf("expiraEm do item = %v", item["expiraEm"])
	}
	bruto, _ := json.Marshal(item)
	if strings.Contains(string(bruto), codigo) {
		t.Error("o codigo em texto esta no item do banco")
	}
}

// Gerar de novo substitui o item (um so por pessoa) e o codigo antigo deixa de valer.
func TestGerarDeNovoSubstituiOItemEInvalidaOAnterior(t *testing.T) {
	f, _ := cenarioVinculo(t)
	_, _, velho := gerarLink(t, subAna)
	_, _, novo := gerarLink(t, subAna)
	if velho == novo {
		t.Fatal("dois codigos iguais")
	}
	if itensDeVinculo(f, subAna) != 1 {
		t.Errorf("ha %d itens LINK# da Ana, esperado 1", itensDeVinculo(f, subAna))
	}
	_, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+velho)))
	if texto != respostaNaoVale {
		t.Errorf("o codigo antigo respondeu %q, esperado que nao vale mais", texto)
	}
	if _, ok := f.ler(t, "USER#"+subAna, "SPACE#s1")["telegramChatId"]; ok {
		t.Error("o codigo antigo ligou o Telegram")
	}
	_, texto, _ = respostaDoBot(t, webhook(t, updateTelegram(2, chatNovo, "private", "/start "+novo)))
	if !strings.HasPrefix(texto, "Ligado ✓") {
		t.Errorf("o codigo novo respondeu %q", texto)
	}
}

// O sub e o do token: um sub ou sid no corpo nao muda de quem e o item.
func TestGerarUsaOSubDoTokenENaoDoCorpo(t *testing.T) {
	f, _ := cenarioVinculo(t)
	st, _ := chamar(t, subAna, "POST", "spaces/s1/telegram/vinculo", nil, `{"sub":"`+subBia+`","sid":"s9","canal":"outro"}`)
	esperar(t, "gerar com corpo", st, 200)
	if itensDeVinculo(f, subBia) != 0 || itensDeVinculo(f, subAna) != 1 {
		t.Errorf("itens: Ana=%d Bia=%d, esperado 1 e 0", itensDeVinculo(f, subAna), itensDeVinculo(f, subBia))
	}
	if it := f.ler(t, "USER#"+subAna, "LINK#telegram"); it["sid"] != "s1" {
		t.Errorf("sid = %v: vem da rota, nao do corpo", it["sid"])
	}
}

func TestGerarSemVinculoOuSemTokenEhRecusado(t *testing.T) {
	f, _ := cenarioVinculo(t)
	antes := f.escritas
	st, _ := chamar(t, "estranho", "POST", "spaces/s1/telegram/vinculo", nil, "")
	esperar(t, "sem vinculo", st, 403)
	var rt rotulo
	res, _ := srvTeste.atender(context.Background(), func() req {
		var r req
		r.RawPath, r.RequestContext.HTTP.Method = "/spaces/s1/telegram/vinculo", "POST"
		return r
	}(), &rt)
	esperar(t, "sem token", res.StatusCode, 401)
	if f.escritas != antes {
		t.Error("uma chamada recusada gravou algo")
	}
}

// Sem bot configurado (sem usuario, token ou segredo) o link nao e gerado: nunca um link que nao liga.
func TestGerarSemBotConfiguradoEh503SemGravar(t *testing.T) {
	f, bot := cenarioVinculo(t)
	bot.pronto = domain.ErrCanalNaoConfigurado
	antes := f.escritas
	st, d := chamar(t, subAna, "POST", "spaces/s1/telegram/vinculo", nil, "")
	esperar(t, "nao configurado", st, 503)
	if d["erro"] != "o Telegram ainda nao foi configurado no servidor" {
		t.Errorf("erro = %v", d["erro"])
	}
	if f.escritas != antes || itensDeVinculo(f, subAna) != 0 {
		t.Error("gerou um item mesmo sem bot configurado")
	}
	// o limite de geracoes nao e gasto por quem bate numa configuracao que falta
	bot.pronto = nil
	for i := 0; i < 5; i++ {
		if st, _ := chamar(t, subAna, "POST", "spaces/s1/telegram/vinculo", nil, ""); st != 200 {
			t.Fatalf("geracao %d: status %d", i+1, st)
		}
	}
}

// A 6a geracao da hora e 429; passada a hora, volta.
func TestGerarTemLimiteDeCincoPorHora(t *testing.T) {
	f, _ := cenarioVinculo(t)
	for i := 1; i <= 5; i++ {
		if st, _, _ := gerarLink(t, subAna); st != 200 {
			t.Fatalf("geracao %d: status %d", i, st)
		}
	}
	antes := f.escritas
	if st, d, _ := gerarLink(t, subAna); st != 429 || d["erro"] == nil {
		t.Errorf("6a geracao: status %d %v, esperado 429 com erro", st, d)
	}
	if f.escritas != antes {
		t.Error("a geracao recusada gravou")
	}
	if st, _, _ := gerarLink(t, subBia); st != 200 { // o limite e por pessoa
		t.Errorf("a Bia nao pode ser afetada pelo limite da Ana: %d", st)
	}
	relogioEm(t, 2026, 2, 5, 16, 1) // uma hora e um minuto depois
	if st, _, _ := gerarLink(t, subAna); st != 200 {
		t.Errorf("passada a hora o limite recomeca: %d", st)
	}
}

// V-T2: um sub que nao e UUID recusa gerar (500, sem gravar) e nenhuma linha de log leva o sub.
func TestGerarComSubQueNaoEUUIDRecusaSemGravar(t *testing.T) {
	f, _ := cenarioVinculo(t)
	const estranho = "sub-nao-uuid-xyz"
	f.colocar(t, doc{"PK": "USER#" + estranho, "SK": "SPACE#s1", "role": "member"})
	buf := capturarLog(t)
	antes := f.escritas
	st, _ := chamar(t, estranho, "POST", "spaces/s1/telegram/vinculo", nil, "")
	esperar(t, "sub nao UUID", st, 500)
	if f.escritas != antes || itensDeVinculo(f, estranho) != 0 {
		t.Error("gravou um item para um sub que nao e UUID")
	}
	if strings.Contains(buf.String(), estranho) {
		t.Errorf("o log leva o sub: %s", buf.String())
	}
}

// V-T2a: o item do codigo so e acessado pela chave exata (nenhuma Query), e a Query de espacos nao o ve.
func TestItemDoVinculoNuncaEListadoNemAparecemNosEspacos(t *testing.T) {
	f, _ := cenarioVinculo(t)
	_, _, codigo := gerarLink(t, subAna)
	respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+codigo)))
	if f.queries != 0 {
		t.Errorf("gerar e usar o codigo fizeram %d Query: so GetItem, PutItem e DeleteItem pela chave exata", f.queries)
	}
	gerarLink(t, subAna) // com o item LINK# presente, GET /me nao pode devolve-lo como espaco
	if itensDeVinculo(f, subAna) != 1 {
		t.Fatal("o item do codigo devia estar na tabela")
	}
	st, d := chamar(t, subAna, "GET", "me", nil, "")
	esperar(t, "me", st, 200)
	if esp := d["espacos"].([]any); len(esp) != 1 || esp[0].(map[string]any)["id"] != "s1" {
		t.Errorf("espacos = %v: o item LINK# nao e um espaco", d["espacos"])
	}
}

// ---------- V-T3 a V-T5: o webhook e o segredo ----------

func TestWebhookSemSegredoCertoEh401SemTocarNoBanco(t *testing.T) {
	f, bot := cenarioVinculo(t)
	_, _, codigo := gerarLink(t, subAna)
	corpo := updateTelegram(1, chatNovo, "private", "/start "+codigo)
	casos := map[string]map[string]string{
		"ausente":   nil,
		"vazio":     {"x-telegram-bot-api-secret-token": ""},
		"diferente": {"x-telegram-bot-api-secret-token": "outro_segredo"},
		"prefixo":   {"x-telegram-bot-api-secret-token": segredoTeste[:5]},
		"sufixo":    {"x-telegram-bot-api-secret-token": segredoTeste + "x"},
		"so authorization": {"authorization": "Bearer " + segredoTeste},
	}
	for nome, cab := range casos {
		l, e := f.leituras, f.escritas
		var rt rotulo
		res, _ := srvTeste.atender(context.Background(), eventoWebhook(corpo, cab), &rt)
		if res.StatusCode != 401 || res.Body != `{"erro":"nao autenticado"}` {
			t.Errorf("%s: %d %s, esperado 401 nao autenticado", nome, res.StatusCode, res.Body)
		}
		semAcessoAoBanco(t, f, nome, l, e)
	}
	// o segredo nao configurado no servidor recusa ate o cabecalho "certo" (falha fechada)
	for nome, ajusta := range map[string]func(){
		"segredo vazio no servidor": func() { bot.segredo = "" },
		"erro ao ler o segredo":     func() { bot.segredo, bot.erroSegredo = segredoTeste, domain.ErrCanalNaoConfigurado },
	} {
		ajusta()
		l, e := f.leituras, f.escritas
		var rt rotulo
		res, _ := srvTeste.atender(context.Background(), eventoWebhook(corpo, comSegredo(nil)), &rt)
		if res.StatusCode != 401 {
			t.Errorf("%s: %d, esperado 401", nome, res.StatusCode)
		}
		semAcessoAoBanco(t, f, nome, l, e)
		bot.segredo, bot.erroSegredo = segredoTeste, nil
	}
	// nada disso ligou o Telegram
	if _, ok := f.ler(t, "USER#"+subAna, "SPACE#s1")["telegramChatId"]; ok {
		t.Error("um pedido sem o segredo certo ligou o Telegram")
	}
}

// O nome do cabecalho em qualquer caixa vale, e o cabecalho authorization (lixo ou nao) nao muda nada: a rota publica nem o le (V-T4).
func TestWebhookAceitaCabecalhoEmCaixaMistaEIgnoraAuthorization(t *testing.T) {
	cenarioVinculo(t)
	for i, cab := range []map[string]string{
		{"X-Telegram-Bot-Api-Secret-Token": segredoTeste},
		{"x-telegram-bot-api-secret-token": segredoTeste, "authorization": "Bearer lixo"},
		{"x-telegram-bot-api-secret-token": segredoTeste, "authorization": ""},
		{"x-telegram-bot-api-secret-token": segredoTeste, "authorization": "Bearer eyJhbGciOiJSUzI1NiJ9.e30.expirado"},
	} {
		var rt rotulo
		res, _ := srvTeste.atender(context.Background(), eventoWebhook(updateTelegram(700+i, chatNovo, "private", "oi"), cab), &rt)
		if _, texto, ok := respostaDoBot(t, res); !ok || texto != respostaSoAvisos {
			t.Errorf("cabecalhos %v: %d %s", cab, res.StatusCode, res.Body)
		}
	}
}

// V-T5: entrada minima. Corpo acima de 16 KiB, JSON invalido, sem message, sem text e chat que nao e privado: 200 {} sem tocar no banco nem no bot.
func TestWebhookEntradaMinimaRespondeVazioSemTrabalho(t *testing.T) {
	f, bot := cenarioVinculo(t)
	_, _, codigo := gerarLink(t, subAna)
	grande := `{"update_id":9,"message":{"text":"` + strings.Repeat("x", 17<<10) + `","chat":{"id":1,"type":"private"}}}`
	casos := map[string]string{
		"17 KiB":         grande,
		"json invalido":  "{nao e json",
		"vazio":          "",
		"array":          `[1,2]`,
		"sem message":    `{"update_id":2,"edited_message":{"text":"x"}}`,
		"sem text":       `{"update_id":3,"message":{"chat":{"id":1,"type":"private"},"photo":[]}}`,
		"texto vazio":    updateTelegram(4, chatNovo, "private", ""),
		"grupo":          updateTelegram(5, -100123, "group", "/start "+codigo),
		"supergrupo":     updateTelegram(6, -100124, "supergroup", "/start "+codigo),
		"canal":          updateTelegram(7, -100125, "channel", "/start "+codigo),
		"tipo ausente":   `{"update_id":8,"message":{"text":"/start x","chat":{"id":1}}}`,
	}
	for nome, corpo := range casos {
		l, e, n := f.leituras, f.escritas, len(bot.enviados)
		res := webhook(t, corpo)
		if res.StatusCode != 200 || res.Body != "{}" {
			t.Errorf("%s: %d %s, esperado 200 {}", nome, res.StatusCode, res.Body)
		}
		semAcessoAoBanco(t, f, nome, l, e)
		if len(bot.enviados) != n {
			t.Errorf("%s: chamou o bot", nome)
		}
	}
}

// V-T5a: a mesma atualizacao duas vezes na mesma instancia: a primeira liga e responde "Ligado", a segunda e 200 {} sem acesso ao banco.
func TestWebhookReentregaDoMesmoUpdateIdEhIgnorada(t *testing.T) {
	f, _ := cenarioVinculo(t)
	_, _, codigo := gerarLink(t, subAna)
	corpo := updateTelegram(42, chatNovo, "private", "/start "+codigo)
	if _, texto, _ := respostaDoBot(t, webhook(t, corpo)); !strings.HasPrefix(texto, "Ligado ✓") {
		t.Fatalf("a primeira respondeu %q", texto)
	}
	l, e := f.leituras, f.escritas
	if res := webhook(t, corpo); res.Body != "{}" {
		t.Errorf("a segunda respondeu %s, esperado {}", res.Body)
	}
	semAcessoAoBanco(t, f, "reentrega", l, e)
}

// ---------- V-T6 a V-T9: usar o codigo ----------

func TestStartComCodigoValidoLigaOChatDoTelegram(t *testing.T) {
	f, _ := cenarioVinculo(t)
	_, _, codigo := gerarLink(t, subAna)
	res := webhook(t, updateTelegram(1, chatNovo, "private", "/start "+codigo))
	chat, texto, ok := respostaDoBot(t, res)
	if !ok || chat != chatNovo || texto != "Ligado ✓ Os avisos de Ana chegam aqui, só com a contagem." {
		t.Errorf("resposta = (%d, %q, %v)", chat, texto, ok)
	}
	if got := f.ler(t, "USER#"+subAna, "SPACE#s1")["telegramChatId"]; got != strconv.FormatInt(chatNovo, 10) {
		t.Errorf("telegramChatId = %v, esperado o chat.id do Telegram", got)
	}
	if itensDeVinculo(f, subAna) != 0 {
		t.Error("o codigo continua na tabela depois de usado")
	}
	if got := f.ler(t, "USER#"+subBia, "SPACE#s1")["telegramChatId"]; got != nil {
		t.Errorf("o codigo da Ana ligou o vinculo da Bia: %v", got)
	}
	// o mesmo codigo de novo nao vale e nao grava
	e := f.escritas
	if _, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(2, chatNovo, "private", "/start "+codigo))); texto != respostaNaoVale {
		t.Errorf("segundo uso: %q", texto)
	}
	if f.escritas != e {
		t.Error("o segundo uso gravou")
	}
}

func TestConfirmacaoSemApelidoDizDesteEspaco(t *testing.T) {
	cenarioVinculo(t)
	_, _, codigo := gerarLink(t, subBia)
	_, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+codigo)))
	if texto != "Ligado ✓ Os avisos de deste espaço chegam aqui, só com a contagem." {
		t.Errorf("texto = %q", texto)
	}
}

// Codigo vencido e codigo de formato certo mas que nunca existiu: so a leitura, nenhuma escrita (uma escrita condicional que falha ainda gasta WCU).
func TestCodigoInexistenteOuVencidoSoLeNuncaEscreve(t *testing.T) {
	f, _ := cenarioVinculo(t)
	_, _, codigo := gerarLink(t, subAna)
	inexistente, _, _ := domain.NovoCodigoVinculo(subBia, bytes.NewReader(bytes.Repeat([]byte{9}, 16)))

	e := f.escritas
	if _, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+inexistente))); texto != respostaNaoVale {
		t.Errorf("inexistente: %q", texto)
	}
	relogioEm(t, 2026, 2, 5, 15, 10) // exatamente 10 minutos: venceu
	if _, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(2, chatNovo, "private", "/start "+codigo))); texto != respostaNaoVale {
		t.Errorf("vencido: %q", texto)
	}
	if f.escritas != e {
		t.Errorf("codigo inexistente ou vencido fez %d escrita(s), esperado so leitura", f.escritas-e)
	}
	if _, ok := f.ler(t, "USER#"+subAna, "SPACE#s1")["telegramChatId"]; ok {
		t.Error("um codigo vencido ligou o Telegram")
	}
}

// Um codigo com o sub certo mas outro aleatorio (hash diferente) nao liga.
func TestCodigoComHashDiferenteNaoLiga(t *testing.T) {
	f, _ := cenarioVinculo(t)
	gerarLink(t, subAna)
	forjado, _, _ := domain.NovoCodigoVinculo(subAna, bytes.NewReader(bytes.Repeat([]byte{3}, 16)))
	e := f.escritas
	if _, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+forjado))); texto != respostaNaoVale {
		t.Errorf("texto = %q", texto)
	}
	if f.escritas != e || itensDeVinculo(f, subAna) != 1 {
		t.Error("um codigo forjado gravou ou apagou o item verdadeiro")
	}
}

// Dois /start do mesmo codigo ao mesmo tempo: so um liga. Aqui a outra execucao consome o codigo entre a leitura e a exclusao.
func TestDoisStartSimultaneosSoUmLiga(t *testing.T) {
	f, _ := cenarioVinculo(t)
	_, _, codigo := gerarLink(t, subAna)
	f.depoisDoGet = func(pk, sk string) {
		if strings.HasPrefix(sk, "LINK#") {
			delete(f.itens, chave(pk, sk)) // a outra execucao consumiu
			f.depoisDoGet = nil
		}
	}
	if _, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+codigo))); texto != respostaNaoVale {
		t.Errorf("a perdedora respondeu %q", texto)
	}
	if _, ok := f.ler(t, "USER#"+subAna, "SPACE#s1")["telegramChatId"]; ok {
		t.Error("a execucao que perdeu a corrida gravou o chat id")
	}
}

// O vinculo foi apagado entre gerar e usar: "Nao consegui ligar", o codigo foi consumido (nao vale duas vezes) e nada e criado.
func TestVinculoApagadoNoMeioDoCaminhoNaoLigaMasConsomeOCodigo(t *testing.T) {
	f, _ := cenarioVinculo(t)
	_, _, codigo := gerarLink(t, subAna)
	delete(f.itens, chave("USER#"+subAna, "SPACE#s1"))
	if _, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+codigo))); texto != respostaNaoLigou {
		t.Errorf("texto = %q", texto)
	}
	if itensDeVinculo(f, subAna) != 0 {
		t.Error("o codigo devia ter sido consumido")
	}
	if f.ler(t, "USER#"+subAna, "SPACE#s1") != nil {
		t.Error("a gravacao criou um vinculo que nao existia")
	}
}

// V-T6a: religar nao troca o destino em silencio. O chat antigo recebe um texto fixo, uma vez, com o chat id lido do vinculo.
func TestReligarAvisaOChatAntigo(t *testing.T) {
	f, bot := cenarioVinculo(t)
	f.colocar(t, doc{"PK": "USER#" + subAna, "SK": "SPACE#s1", "role": "member", "apelido": "Ana", "telegramChatId": chatAntigo})
	_, _, codigo := gerarLink(t, subAna)
	if _, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+codigo))); !strings.HasPrefix(texto, "Ligado ✓") {
		t.Fatalf("texto = %q", texto)
	}
	if len(bot.enviados) != 1 || bot.enviados[0].chat != chatAntigo || bot.enviados[0].texto != respostaOutroChat {
		t.Errorf("o aviso ao chat antigo = %v", bot.enviados)
	}
	if got := f.ler(t, "USER#"+subAna, "SPACE#s1")["telegramChatId"]; got != strconv.FormatInt(chatNovo, 10) {
		t.Errorf("telegramChatId = %v", got)
	}
}

func TestReligarNoMesmoChatOuSemChatAntesNaoAvisaNinguem(t *testing.T) {
	f, bot := cenarioVinculo(t)
	f.colocar(t, doc{"PK": "USER#" + subAna, "SK": "SPACE#s1", "role": "member", "apelido": "Ana", "telegramChatId": strconv.FormatInt(chatNovo, 10)})
	_, _, c1 := gerarLink(t, subAna)
	respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+c1))) // o mesmo chat
	_, _, c2 := gerarLink(t, subBia)
	respostaDoBot(t, webhook(t, updateTelegram(2, chatNovo+1, "private", "/start "+c2))) // a Bia nao tinha chat
	if len(bot.enviados) != 0 {
		t.Errorf("nao devia avisar ninguem: %v", bot.enviados)
	}
}

// A falha no aviso ao chat antigo nao derruba o "Ligado" e o log nao leva chat id.
func TestFalhaNoAvisoAoChatAntigoNaoDerrubaOLigado(t *testing.T) {
	f, bot := cenarioVinculo(t)
	f.colocar(t, doc{"PK": "USER#" + subAna, "SK": "SPACE#s1", "role": "member", "apelido": "Ana", "telegramChatId": chatAntigo})
	bot.falhaEnvio = errors.New("telegram respondeu 403")
	buf := capturarLog(t)
	_, _, codigo := gerarLink(t, subAna)
	if _, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+codigo))); !strings.HasPrefix(texto, "Ligado ✓") {
		t.Errorf("texto = %q", texto)
	}
	if strings.Contains(buf.String(), chatAntigo) {
		t.Errorf("o log leva o chat id: %s", buf.String())
	}
}

// V-T7: codigo fora do formato nao le o banco; a 11a tentativa do mesmo chat na hora e ignorada; outro chat nao e afetado.
func TestCodigoForaDoFormatoNaoLeOBancoEChatTemLimite(t *testing.T) {
	f, _ := cenarioVinculo(t)
	for i, c := range []string{"abc", strings.Repeat("A", 42), strings.Repeat("A", 44), strings.Repeat("!", 43)} {
		l, e := f.leituras, f.escritas
		_, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(i+1, chatNovo, "private", "/start "+c)))
		if texto != respostaNaoVale {
			t.Errorf("%q: %q", c, texto)
		}
		semAcessoAoBanco(t, f, "formato "+c, l, e)
	}
	// 10 tentativas de formato certo (que nunca existiram) por hora; a 11a e ignorada
	for i := 0; i < 10; i++ {
		c, _, _ := domain.NovoCodigoVinculo(subBia, bytes.NewReader(bytes.Repeat([]byte{byte(10 + i)}, 16)))
		if _, texto, _ := respostaDoBot(t, webhook(t, updateTelegram(100+i, chatNovo, "private", "/start "+c))); texto != respostaNaoVale {
			t.Fatalf("tentativa %d: %q", i+1, texto)
		}
	}
	onze, _, _ := domain.NovoCodigoVinculo(subBia, bytes.NewReader(bytes.Repeat([]byte{99}, 16)))
	l, e := f.leituras, f.escritas
	if res := webhook(t, updateTelegram(200, chatNovo, "private", "/start "+onze)); res.Body != "{}" {
		t.Errorf("a 11a tentativa respondeu %s, esperado {} (ignorada)", res.Body)
	}
	semAcessoAoBanco(t, f, "11a tentativa", l, e)
	if _, texto, ok := respostaDoBot(t, webhook(t, updateTelegram(201, chatNovo+7, "private", "/start "+onze))); !ok || texto != respostaNaoVale {
		t.Errorf("outro chat nao pode ser afetado: %q", texto)
	}
}

// V-T8: /start sem codigo, texto livre e /parar: o texto fixo, sem gravar nem apagar nada e sem dado da conta.
func TestOutrosTextosRespondemOTextoFixoSemTocarNoBanco(t *testing.T) {
	f, _ := cenarioVinculo(t)
	f.colocar(t, doc{"PK": "USER#" + subAna, "SK": "SPACE#s1", "role": "member", "apelido": "Ana", "telegramChatId": strconv.FormatInt(chatNovo, 10)})
	for i, txt := range []string{"/start", "/parar", "oi", "/start  ", "/start a b", "/outra coisa", "/stop"} {
		l, e := f.leituras, f.escritas
		_, texto, ok := respostaDoBot(t, webhook(t, updateTelegram(i+1, chatNovo, "private", txt)))
		if !ok || texto != respostaSoAvisos {
			t.Errorf("%q: %q", txt, texto)
		}
		semAcessoAoBanco(t, f, txt, l, e)
		for _, proibido := range []string{"Ana", "s1", "Familia", subAna} {
			if strings.Contains(texto, proibido) {
				t.Errorf("%q: a resposta leva dado da conta (%q)", txt, proibido)
			}
		}
	}
	if got := f.ler(t, "USER#"+subAna, "SPACE#s1")["telegramChatId"]; got != strconv.FormatInt(chatNovo, 10) {
		t.Errorf("/parar nao apaga o destino, mas ficou %v", got)
	}
}

// V-T9: codigo inexistente, vencido, ja usado e de formato errado dao a MESMA resposta; o webhook nao revela se um codigo existiu.
func TestRespostaDeCodigoQueNaoValeEIgualParaTodosOsMotivos(t *testing.T) {
	cenarioVinculo(t)
	_, _, usado := gerarLink(t, subAna)
	respostaDoBot(t, webhook(t, updateTelegram(1, chatNovo, "private", "/start "+usado)))
	_, _, vencido := gerarLink(t, subBia)
	inexistente, _, _ := domain.NovoCodigoVinculo(subBia, bytes.NewReader(bytes.Repeat([]byte{5}, 16)))
	relogioEm(t, 2026, 2, 5, 15, 30)
	var corpos []string
	for i, c := range []string{usado, vencido, inexistente, "formato-errado"} {
		res := webhook(t, updateTelegram(10+i, chatNovo+int64(i), "private", "/start "+c))
		corpos = append(corpos, strings.Replace(res.Body, strconv.FormatInt(chatNovo+int64(i), 10), "CHAT", 1))
		if res.StatusCode != 200 {
			t.Errorf("%q: status %d", c, res.StatusCode)
		}
	}
	for _, c := range corpos[1:] {
		if c != corpos[0] {
			t.Errorf("respostas diferentes revelam o motivo: %v", corpos)
			break
		}
	}
}

// V-T10: o rotulo e acao=telegram.webhook rota=telegram (nao sem-token) e nenhuma linha leva chat id, texto, codigo, segredo, sub nem sid.
func TestLogDoWebhookNaoLevaDadoNenhum(t *testing.T) {
	cenarioVinculo(t)
	buf := capturarLog(t)
	_, _, codigo := gerarLink(t, subAna)
	for _, corpo := range []string{updateTelegram(1, chatNovo, "private", "/start "+codigo), updateTelegram(2, chatNovo, "private", "texto livre secreto")} {
		r := eventoWebhook(corpo, comSegredo(map[string]string{"authorization": "Bearer lixo"}))
		r.RequestContext.RequestID = "rid-1"
		if _, err := srvTeste.handler(context.Background(), r); err != nil {
			t.Fatal(err)
		}
	}
	r := eventoWebhook("{}", map[string]string{"x-telegram-bot-api-secret-token": "errado"}) // 401 tambem e logado com o rotulo certo
	if _, err := srvTeste.handler(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	log := buf.String()
	if !strings.Contains(log, "acao=telegram.webhook rota=telegram") || strings.Contains(log, "sem-token") {
		t.Errorf("o rotulo do log esta errado: %s", log)
	}
	for _, proibido := range []string{strconv.FormatInt(chatNovo, 10), codigo, segredoTeste, "errado", "texto livre", "secreto", subAna, "s1", "Bearer", "lixo", "203.0.113.9"} {
		if strings.Contains(log, proibido) {
			t.Errorf("o log leva %q: %s", proibido, log)
		}
	}
}

func TestRotularOWebhookEOLink(t *testing.T) {
	if got := rotular("POST", []string{"telegram", "webhook"}); got != (rotulo{"telegram.webhook", "telegram"}) {
		t.Errorf("rotular webhook = %+v", got)
	}
	if got := rotular("POST", []string{"spaces", "s1", "telegram", "vinculo"}); got != (rotulo{"telegram.vincular", "telegram"}) {
		t.Errorf("rotular vinculo = %+v", got)
	}
	if got := rotular("GET", []string{"telegram", "webhook"}); got != (rotulo{acaoDesconhecida, "-"}) {
		t.Errorf("GET nao e a rota publica: %+v", got)
	}
}

// ---------- V-T15: GET /me ganha canais ----------

func TestGetMeDevolveOEstadoDosCanais(t *testing.T) {
	f := cenario(t)
	vinc := func(sub string, extra doc) {
		d := doc{"PK": "USER#" + sub, "SK": "SPACE#s1", "role": "member"}
		for k, v := range extra {
			d[k] = v
		}
		f.colocar(t, d)
	}
	comTokens := func(sub string, tokens ...string) {
		f.itens[chave("USER#"+sub, "SPACE#s1")]["pushTokens"] = &types.AttributeValueMemberSS{Value: tokens}
	}
	vinc("so-chat", doc{"telegramChatId": "222222"})
	vinc("so-aparelho", nil)
	comTokens("so-aparelho", "tok-1")
	vinc("os-dois", doc{"telegramChatId": "333333"})
	comTokens("os-dois", "tok-1", "tok-2")
	vinc("nenhum", doc{"telegramChatId": ""})
	casos := map[string][2]bool{"so-chat": {true, false}, "so-aparelho": {false, true}, "os-dois": {true, true}, "nenhum": {false, false}}
	for sub, quer := range casos {
		f.queries, f.leituras = 0, 0
		st, d := chamar(t, sub, "GET", "me", nil, "")
		esperar(t, sub, st, 200)
		esp := d["espacos"].([]any)[0].(map[string]any)
		canais, _ := esp["canais"].(map[string]any)
		if canais["telegram"] != quer[0] || canais["push"] != quer[1] {
			t.Errorf("%s: canais = %v, esperado telegram=%v push=%v", sub, canais, quer[0], quer[1])
		}
		if _, tem := esp["telegramChatId"]; !tem {
			t.Errorf("%s: na fase 2 o telegramChatId ainda vem", sub)
		}
		if f.queries != 1 || f.leituras != 2 { // a Query dos espacos e o META do espaco, como antes: nenhuma leitura nova
			t.Errorf("%s: %d Query e %d leituras no total, esperado 1 e 2", sub, f.queries, f.leituras)
		}
	}
}

// ---------- V-T19: arquitetura ----------

// service/vinculos.go nao nomeia o canal: nem o nome, nem a URL do link, nem o comando de entrada (AD-11 e R28), sem excecao nova no teste de nomes.
func TestServicoDoVinculoNaoNomeiaOCanal(t *testing.T) {
	b, err := os.ReadFile("service/vinculos.go")
	if err != nil {
		t.Fatal(err)
	}
	texto := strings.ToLower(string(b))
	for _, proibido := range []string{"telegram", "t.me", "/start", "secret_token", "fcm", "push"} {
		if strings.Contains(texto, proibido) {
			t.Errorf("service/vinculos.go contem %q: o service trata o canal como valor opaco", proibido)
		}
	}
}

