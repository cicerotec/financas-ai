package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Envio de mensagens pelo Telegram. O token do bot fica no SSM Parameter Store (SecureString); a Lambda so le o nome do
// parametro em TELEGRAM_TOKEN_PARAM. Sem o parametro configurado, os avisos simplesmente nao saem.

var errTelegramNaoConfigurado = errors.New("telegram nao configurado")

// ssmAPI e o que o codigo usa do SSM; os testes trocam por um valor fixo.
type ssmAPI interface {
	GetParameter(context.Context, *ssm.GetParameterInput, ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

var (
	ssmCli       ssmAPI
	tokenParam   = os.Getenv("TELEGRAM_TOKEN_PARAM")
	tokenMu      sync.Mutex
	tokenCache   string
	telegramBase = "https://api.telegram.org"
	httpTelegram = &http.Client{Timeout: 10 * time.Second}
)

// enviarMensagem e trocada nos testes.
var enviarMensagem = enviarTelegram

func tokenTelegram(ctx context.Context) (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if tokenCache != "" {
		return tokenCache, nil
	}
	if tokenParam == "" || ssmCli == nil {
		return "", errTelegramNaoConfigurado
	}
	out, err := ssmCli.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(tokenParam), WithDecryption: aws.Bool(true)})
	if err != nil {
		return "", fmt.Errorf("ler token no SSM: %w", err)
	}
	if out.Parameter == nil || out.Parameter.Value == nil || strings.TrimSpace(*out.Parameter.Value) == "" {
		return "", errTelegramNaoConfigurado
	}
	tokenCache = strings.TrimSpace(*out.Parameter.Value)
	return tokenCache, nil
}

func enviarTelegram(ctx context.Context, chatID, texto string) error {
	token, err := tokenTelegram(ctx)
	if err != nil {
		return err
	}
	corpo, _ := json.Marshal(map[string]any{"chat_id": chatID, "text": texto, "disable_web_page_preview": true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telegramBase+"/bot"+token+"/sendMessage", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	res, err := httpTelegram.Do(req)
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

// chat id do Telegram: numero (negativo em grupos). Vazio remove o vinculo.
var reChatID = regexp.MustCompile(`^-?[0-9]{5,20}$`)
