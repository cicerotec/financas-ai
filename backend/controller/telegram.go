package controller

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"financas/domain"
	"financas/service"
)

// O vinculo do Telegram por link (spec telegram-vinculo, R3 a R10 e R28). O controller sabe o que e Telegram: o cabecalho do segredo, o parse do
// update, o /start e os textos fixos. O service.Vinculos so recebe (canal, codigo, endereco) e trata o canal como valor opaco (AD-11).

const (
	canalDoVinculo = "telegram"

	cabecalhoDoSegredo = "x-telegram-bot-api-secret-token"
	limiteDoCorpo      = 16 << 10 // 16 KiB (R5)
	ultimosUpdates     = 100      // update_id ja tratados nesta instancia (R5)
)

// Textos fixos do bot (R6, R6a, R7). Nenhum leva dado da conta, exceto o apelido que a propria pessoa deu ao vinculo (D6).
const (
	textoNaoVale   = "Este link não vale mais. Volte ao app e toque em Ligar Telegram de novo."
	textoNaoLigou  = "Não consegui ligar agora. Volte ao app e toque em Ligar Telegram de novo."
	textoSoAvisos  = "Este bot só envia avisos. Para ligar, use o app; para parar, desligue em Listas, Avisos."
	textoOutroChat = "Este espaço foi ligado a outro chat do Telegram. Se não foi você, desligue em Listas, Avisos."
)

// BotDoTelegram e o que o controller precisa do adaptador do bot (declarado aqui para os testes trocarem por um falso).
type BotDoTelegram interface {
	// Pronto devolve o usuario do bot se o token, o usuario e o segredo do webhook estao configurados; domain.ErrCanalNaoConfigurado se falta algum.
	Pronto(ctx context.Context) (usuario string, err error)
	// SegredoDoWebhook devolve o segredo esperado no cabecalho do webhook; erro (ou vazio) = nao configurado.
	SegredoDoWebhook(ctx context.Context) (string, error)
	// EnviarTexto manda um texto a um chat (so o aviso ao chat antigo, R6a).
	EnviarTexto(ctx context.Context, chatID, texto string) error
}

// Telegram liga a rota que gera o link e o webhook publico do bot ao service do vinculo.
type Telegram struct {
	svc *service.Vinculos
	bot BotDoTelegram

	mu      sync.Mutex
	vistos  map[int64]bool
	ordemDe []int64 // os update_id na ordem de chegada, para esquecer o mais antigo
}

func NewTelegram(svc *service.Vinculos, bot BotDoTelegram) *Telegram {
	return &Telegram{svc: svc, bot: bot, vistos: map[int64]bool{}}
}

// PostVinculo: POST /spaces/{sid}/telegram/vinculo (sem corpo). O sub vem do token. 200 {url, expiraEm}; 429 na 6a geracao da hora; 503 sem bot configurado.
// So o servidor monta a URL (https://t.me/<usuario>?start=<codigo>); o front so a abre.
func (c *Telegram) PostVinculo(ctx context.Context, sid, sub string) (Resp, error) {
	usuario, err := c.bot.Pronto(ctx)
	if errors.Is(err, domain.ErrCanalNaoConfigurado) {
		return Erro(503, "o Telegram ainda nao foi configurado no servidor")
	}
	if err != nil {
		log.Println("vinculo: conferir o bot:", err)
		return respostaErro(service.ErrInterno)
	}
	codigo, expira, err := c.svc.Gerar(ctx, sid, sub, canalDoVinculo)
	switch {
	case errors.Is(err, service.ErrLimiteDeLinks):
		return Erro(429, "muitos links pedidos: tente de novo mais tarde")
	case err != nil:
		return respostaErro(err)
	}
	return Out(200, domain.Doc{"url": "https://t.me/" + usuario + "?start=" + codigo, "expiraEm": expira.UTC().Format(time.RFC3339)})
}

// updateDoTelegram e so o que o webhook le de uma atualizacao do Telegram.
type updateDoTelegram struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		Chat struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		} `json:"chat"`
	} `json:"message"`
}

// PostWebhook: POST /telegram/webhook. PUBLICA: chega sem token do Cognito e o cabecalho authorization nem e lido; a barreira e o segredo no
// cabecalho x-telegram-bot-api-secret-token, conferido em tempo constante ANTES de ler o corpo e de qualquer acesso ao banco, com falha fechada
// (R4). Ao que o segredo autenticou responde sempre 200, para o Telegram nao repetir a entrega; so o segredo errado e 401.
func (c *Telegram) PostWebhook(ctx context.Context, cabecalhos map[string]string, corpo string) (Resp, error) {
	if !c.segredoConfere(ctx, cabecalhos) {
		return Erro(401, "nao autenticado")
	}
	nada := func() (Resp, error) { return Out(200, domain.Doc{}) }
	if len(corpo) > limiteDoCorpo {
		return nada()
	}
	var up updateDoTelegram
	if json.Unmarshal([]byte(corpo), &up) != nil || up.Message == nil || up.Message.Text == "" || up.Message.Chat.Type != "private" {
		return nada()
	}
	if c.jaTratado(up.UpdateID) { // o Telegram reenvia a mesma atualizacao se a Lambda demora (R5)
		return nada()
	}
	chat := up.Message.Chat.ID
	responder := func(texto string) (Resp, error) {
		return Out(200, domain.Doc{"method": "sendMessage", "chat_id": chat, "text": texto})
	}

	codigo, ehLink := codigoDoStart(up.Message.Text)
	if !ehLink { // /start sem codigo, qualquer outro texto e /parar: o texto fixo, sem gravar nem apagar nada (R7)
		return responder(textoSoAvisos)
	}
	endereco := strconv.FormatInt(chat, 10)
	ligado, err := c.svc.Ligar(ctx, canalDoVinculo, codigo, endereco)
	switch {
	case errors.Is(err, service.ErrTentativasDemais):
		return nada()
	case errors.Is(err, service.ErrVinculoInvalido): // a mesma resposta para formato errado, inexistente, vencido e ja usado (R9)
		return responder(textoNaoVale)
	case err != nil:
		return responder(textoNaoLigou)
	}
	if ligado.Anterior != "" && ligado.Anterior != endereco { // religar nao troca o destino em silencio (R6a)
		if err := c.bot.EnviarTexto(ctx, ligado.Anterior, textoOutroChat); err != nil {
			log.Println("vinculo: avisar o chat antigo:", err) // melhor esforco: o erro so leva o codigo HTTP, nunca o chat id
		}
	}
	return responder(textoLigado(ligado.Apelido))
}

// textoLigado e a confirmacao: leva o apelido que a pessoa deu ao vinculo (D6); sem apelido, "deste espaco".
func textoLigado(apelido string) string {
	de := "deste espaço"
	if a := strings.TrimSpace(apelido); a != "" {
		de = a
	}
	return "Ligado ✓ Os avisos de " + de + " chegam aqui, só com a contagem."
}

// codigoDoStart devolve o codigo de "/start <codigo>". Sem codigo, ou outro texto, ehLink=false.
func codigoDoStart(texto string) (codigo string, ehLink bool) {
	f := strings.Fields(texto)
	if len(f) != 2 || f[0] != "/start" {
		return "", false
	}
	return f[1], true
}

// segredoConfere compara o cabecalho (qualquer caixa no nome) com o segredo do SSM, em tempo constante. Sem segredo configurado, ou cabecalho
// ausente, vazio ou diferente: false. Compara os SHA-256 para o tempo nao depender do tamanho.
func (c *Telegram) segredoConfere(ctx context.Context, cabecalhos map[string]string) bool {
	esperado, err := c.bot.SegredoDoWebhook(ctx)
	if err != nil || esperado == "" {
		return false
	}
	recebido := ""
	for k, v := range cabecalhos {
		if strings.EqualFold(k, cabecalhoDoSegredo) {
			recebido = v
		}
	}
	if recebido == "" {
		return false
	}
	a, b := sha256.Sum256([]byte(esperado)), sha256.Sum256([]byte(recebido))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

// jaTratado diz se este update_id ja passou por esta instancia e, se nao, o lembra (os ultimos 100). Melhor esforco: outra instancia pode nao o ter visto.
func (c *Telegram) jaTratado(id int64) bool {
	if id == 0 {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.vistos[id] {
		return true
	}
	c.vistos[id] = true
	c.ordemDe = append(c.ordemDe, id)
	if len(c.ordemDe) > ultimosUpdates {
		delete(c.vistos, c.ordemDe[0])
		c.ordemDe = c.ordemDe[1:]
	}
	return false
}
