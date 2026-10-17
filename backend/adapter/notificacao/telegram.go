// Package notificacao tem os canais de envio de aviso: um arquivo por canal (hoje o Telegram; e-mail e WhatsApp quando
// existirem). Cada um implementa service.Notificador e devolve domain.ErrCanalNaoConfigurado quando falta o que precisa.
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

// Enviar manda o texto para o chat id.
func (t *Telegram) Enviar(ctx context.Context, chatID, texto string) error {
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
	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 300))
		return fmt.Errorf("telegram respondeu %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}
