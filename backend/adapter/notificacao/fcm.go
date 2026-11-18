package notificacao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"financas/domain"
)

// FCM entrega o aviso como notificacao do app Android pelo Firebase Cloud Messaging (API HTTP v1). O destino e o token do aparelho. A
// notificacao so tem a CONTAGEM ("2 atrasados · 1 vence hoje"): ela aparece na tela bloqueada, entao descricao, valor e meio de pagamento
// nunca entram (domain.Contagens e a unica coisa do resumo que este arquivo le). A credencial fica no SSM (FCM_CREDENCIAL_PARAM); sem
// ela o canal devolve ErrCanalNaoConfigurado. Nada aqui vai ao log nem a um erro: token do aparelho, token de acesso, credencial ou corpo.
type FCM struct {
	ssm   SSM
	param string
	// Base, AuthURL, HTTP e agora existem para o teste apontar para servidores falsos e controlar o relogio.
	Base    string
	AuthURL string
	HTTP    *http.Client
	agora   func() time.Time

	mu     sync.Mutex
	conta  *contaDeServico
	acesso string
	expira time.Time
}

func NovoFCM(ssm SSM, param string) *FCM {
	return &FCM{ssm: ssm, param: param, Base: "https://fcm.googleapis.com", AuthURL: audTokenGoogle,
		HTTP: &http.Client{Timeout: 10 * time.Second}, agora: time.Now}
}

// Nome e o nome do canal no despachante.
func (f *FCM) Nome() string { return "push" }

// canalDeNotificacao e o canal de notificacao que o app cria ao abrir (PushPlugin); a mensagem pede esse.
const canalDeNotificacao = "avisos"

// ttlDoAviso: o aviso de ontem nao vale amanha, e a rotina manda o de amanha.
const ttlDoAviso = "86400s"

// TextoDoPush monta o titulo e o corpo da notificacao a partir so das contagens.
func TextoDoPush(c domain.ContagemAvisos) (titulo, corpo string) {
	titulo = c.Marca + "Finanças"
	if c.Teste {
		return titulo, "Teste: os avisos vão chegar aqui."
	}
	var partes []string
	if c.Atrasados > 0 {
		partes = append(partes, plural(c.Atrasados, "atrasado", "atrasados"))
	}
	if c.Hoje > 0 {
		if c.Hoje == 1 {
			partes = append(partes, "1 vence hoje")
		} else {
			partes = append(partes, fmt.Sprintf("%d vencem hoje", c.Hoje))
		}
	}
	if c.AVencer > 0 {
		partes = append(partes, fmt.Sprintf("%d a vencer", c.AVencer))
	}
	if len(partes) == 0 {
		return titulo, "Você tem avisos de hoje."
	}
	return titulo, strings.Join(partes, " · ")
}

var codigoDoFCM = regexp.MustCompile(`^[A-Z_]{1,40}$`)

// Enviar manda a notificacao ao aparelho (token). 404 UNREGISTERED e o aparelho que saiu: ErrDestinoInvalido.
func (f *FCM) Enviar(ctx context.Context, token string, r domain.ResumoAvisos) error {
	acesso, conta, err := f.tokenDeAcesso(ctx)
	if err != nil {
		return err
	}
	titulo, corpoTxt := TextoDoPush(domain.Contagens(r))
	corpo, _ := json.Marshal(map[string]any{"message": map[string]any{
		"token":        token,
		"notification": map[string]any{"title": titulo, "body": corpoTxt},
		"android":      map[string]any{"ttl": ttlDoAviso, "notification": map[string]any{"channel_id": canalDeNotificacao}},
	}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.Base+"/v1/projects/"+conta.projectID+"/messages:send", bytes.NewReader(corpo))
	if err != nil {
		return errors.New("montar o pedido ao FCM falhou")
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+acesso)
	res, err := f.HTTP.Do(req)
	if err != nil {
		return errors.New("falha de rede ao chamar o FCM") // o erro do http leva a URL: nao repassa o texto original
	}
	defer res.Body.Close()
	if res.StatusCode/100 == 2 {
		return nil
	}
	cod := codigoDeErro(res.Body)
	switch {
	case res.StatusCode == http.StatusNotFound && cod == "UNREGISTERED":
		return domain.ErrDestinoInvalido
	case res.StatusCode == http.StatusUnauthorized, res.StatusCode == http.StatusForbidden && cod != "SENDER_ID_MISMATCH":
		return fmt.Errorf("o FCM recusou a credencial (%d): %w", res.StatusCode, domain.ErrCredencialRecusada)
	}
	if cod == "" {
		return fmt.Errorf("fcm respondeu %d", res.StatusCode)
	}
	return fmt.Errorf("fcm respondeu %d %s", res.StatusCode, cod)
}

// codigoDeErro le o errorCode do FcmError da resposta; so devolve o que parece um codigo (letras maiusculas e _), nunca texto livre.
func codigoDeErro(corpo io.Reader) string {
	var e struct {
		Error struct {
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	if json.NewDecoder(io.LimitReader(corpo, 64<<10)).Decode(&e) != nil {
		return ""
	}
	for _, d := range e.Error.Details {
		if codigoDoFCM.MatchString(d.ErrorCode) {
			return d.ErrorCode
		}
	}
	return ""
}
