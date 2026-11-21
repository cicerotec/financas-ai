// Package notificacao tem os canais de envio de aviso: um arquivo por canal (hoje o Telegram; e-mail e WhatsApp quando
// existirem). Cada um implementa service.Canal (sem importar o service: so tipos do domain) e devolve domain.ErrCanalNaoConfigurado quando falta o que precisa.
package notificacao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"financas/domain"
)

// SSM e o que o canal usa do Parameter Store; os testes trocam por um valor fixo.
type SSM interface {
	GetParameter(context.Context, *ssm.GetParameterInput, ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// Telegram envia mensagens pelo bot. O token fica no SSM Parameter Store (SecureString); aqui so entra o nome do parametro
// (TELEGRAM_TOKEN_PARAM). Sem o parametro, ou sem valor nele, os avisos simplesmente nao saem. O token lido fica em cache
// no proprio struct (um por processo da Lambda).
type Telegram struct {
	ssm   SSM
	param string
	// Base e HTTP existem para o teste apontar para um servidor falso.
	Base string
	HTTP *http.Client

	mu    sync.Mutex
	token string
}

func NovoTelegram(ssm SSM, param string) *Telegram {
	return &Telegram{ssm: ssm, param: param, Base: "https://api.telegram.org", HTTP: &http.Client{Timeout: 10 * time.Second}}
}

func (t *Telegram) tokenDoBot(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token != "" {
		return t.token, nil
	}
	if t.param == "" || t.ssm == nil {
		return "", domain.ErrCanalNaoConfigurado
	}
	out, err := t.ssm.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(t.param), WithDecryption: aws.Bool(true)})
	if err != nil {
		return "", fmt.Errorf("ler token no SSM: %w", err)
	}
	if out.Parameter == nil || out.Parameter.Value == nil || strings.TrimSpace(*out.Parameter.Value) == "" {
		return "", domain.ErrCanalNaoConfigurado
	}
	t.token = strings.TrimSpace(*out.Parameter.Value)
	return t.token, nil
}

// Nome e o nome do canal no despachante.
func (t *Telegram) Nome() string { return "telegram" }

// Enviar entrega o resumo ao chat id, numa mensagem so (TextoDoTelegram). Sem token, devolve ErrCanalNaoConfigurado e nada e enviado.
func (t *Telegram) Enviar(ctx context.Context, chatID string, r domain.ResumoAvisos) error {
	return t.EnviarTexto(ctx, chatID, TextoDoTelegram(r))
}

// respostaDeErro e o que se le do corpo de uma resposta de erro do Telegram. So o codigo e o motivo do 400 servem para decidir; nada
// disso vai para um erro ou para o log (o texto livre pode trazer o chat id ou o token).
type respostaDeErro struct {
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

// EnviarTexto manda um texto para o chat id, numa mensagem so. Erros: 403 (bot bloqueado, conta apagada) e 400 "chat not found" sao
// domain.ErrDestinoInvalido; 401 (token do bot recusado) e domain.ErrCredencialRecusada; o resto leva so o codigo HTTP e o error_code,
// nunca a URL, o token, o corpo da resposta nem o chat id. Nenhum erro e repetido.
func (t *Telegram) EnviarTexto(ctx context.Context, chatID, texto string) error {
	token, err := t.tokenDoBot(ctx)
	if err != nil {
		return err
	}
	corpo, _ := json.Marshal(map[string]any{"chat_id": chatID, "text": texto, "disable_web_page_preview": true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Base+"/bot"+token+"/sendMessage", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	res, err := t.HTTP.Do(req)
	if err != nil {
		// o erro do http inclui a URL (com o token): nao repassa o texto original
		return errors.New("falha de rede ao chamar o Telegram")
	}
	defer res.Body.Close()
	if res.StatusCode/100 == 2 {
		return nil
	}
	var e respostaDeErro
	_ = json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&e)
	switch {
	case res.StatusCode == http.StatusForbidden,
		res.StatusCode == http.StatusBadRequest && strings.Contains(strings.ToLower(e.Description), "chat not found"):
		return domain.ErrDestinoInvalido
	case res.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("o Telegram recusou a credencial do bot (%d): %w", res.StatusCode, domain.ErrCredencialRecusada)
	}
	if e.ErrorCode != 0 && e.ErrorCode != res.StatusCode {
		return fmt.Errorf("telegram respondeu %d (error_code %d)", res.StatusCode, e.ErrorCode)
	}
	return fmt.Errorf("telegram respondeu %d", res.StatusCode)
}
