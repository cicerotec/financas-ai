package notificacao

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/golang-jwt/jwt/v5"

	"financas/domain"
)

// Autenticacao no Firebase Cloud Messaging sem biblioteca nova (fluxo "service account, so HTTP" do Google): a credencial e o JSON da conta
// de servico, no SSM; o canal assina um JWT RS256 com a chave privada e o troca por um token de acesso de curta duracao no endpoint de token.
// O token de acesso fica em cache, um por processo, ate pouco antes de vencer.
const (
	audTokenGoogle  = "https://oauth2.googleapis.com/token"
	escopoMensagens = "https://www.googleapis.com/auth/firebase.messaging"
	folgaDoToken    = 5 * time.Minute
)

// contaDeServico e o que o servidor usa do JSON baixado do Firebase (as demais chaves do arquivo sao ignoradas e nunca guardadas).
type contaDeServico struct {
	email     string
	projectID string
	chave     *rsa.PrivateKey
}

// credencial le a conta de servico do SSM (uma vez por processo). Sem parametro, sem valor ou JSON incompleto: ErrCanalNaoConfigurado.
// Erro do SSM nao vira "nao configurado".
func (f *FCM) credencial(ctx context.Context) (*contaDeServico, error) {
	if f.conta != nil {
		return f.conta, nil
	}
	if f.param == "" || f.ssm == nil {
		return nil, domain.ErrCanalNaoConfigurado
	}
	out, err := f.ssm.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(f.param), WithDecryption: aws.Bool(true)})
	if err != nil {
		return nil, errors.New("ler a credencial do FCM no SSM falhou")
	}
	if out.Parameter == nil || out.Parameter.Value == nil || strings.TrimSpace(*out.Parameter.Value) == "" {
		return nil, domain.ErrCanalNaoConfigurado
	}
	var j struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
		ProjectID   string `json:"project_id"`
	}
	if json.Unmarshal([]byte(*out.Parameter.Value), &j) != nil || j.ClientEmail == "" || j.PrivateKey == "" || j.ProjectID == "" {
		return nil, domain.ErrCanalNaoConfigurado
	}
	chave, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(j.PrivateKey))
	if err != nil {
		return nil, domain.ErrCanalNaoConfigurado // chave que nao le: credencial invalida, tratada como nao configurada
	}
	f.conta = &contaDeServico{email: j.ClientEmail, projectID: j.ProjectID, chave: chave}
	return f.conta, nil
}

// tokenDeAcesso devolve um token de acesso valido, pedindo outro so quando o guardado esta perto de vencer.
func (f *FCM) tokenDeAcesso(ctx context.Context) (string, *contaDeServico, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	conta, err := f.credencial(ctx)
	if err != nil {
		return "", nil, err
	}
	agora := f.agora()
	if f.acesso != "" && agora.Before(f.expira.Add(-folgaDoToken)) {
		return f.acesso, conta, nil
	}
	assinado, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": conta.email, "scope": escopoMensagens, "aud": audTokenGoogle,
		"iat": agora.Unix(), "exp": agora.Add(time.Hour).Unix(),
	}).SignedString(conta.chave)
	if err != nil {
		return "", nil, errors.New("assinar o JWT do FCM falhou")
	}
	form := url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"}, "assertion": {assinado}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.AuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", nil, errors.New("montar o pedido do token de acesso do FCM falhou")
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	res, err := f.HTTP.Do(req)
	if err != nil {
		return "", nil, errors.New("falha de rede ao pedir o token de acesso do FCM")
	}
	defer res.Body.Close()
	corpo, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	switch {
	case res.StatusCode == http.StatusBadRequest || res.StatusCode == http.StatusUnauthorized:
		return "", nil, fmt.Errorf("o token de acesso do FCM foi recusado (%d): %w", res.StatusCode, domain.ErrCredencialRecusada)
	case res.StatusCode/100 != 2:
		return "", nil, fmt.Errorf("o endpoint de token do FCM respondeu %d", res.StatusCode)
	}
	var t struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if json.Unmarshal(corpo, &t) != nil || t.AccessToken == "" {
		return "", nil, errors.New("resposta inesperada do endpoint de token do FCM")
	}
	if t.ExpiresIn <= 0 {
		t.ExpiresIn = 3600
	}
	f.acesso, f.expira = t.AccessToken, agora.Add(time.Duration(t.ExpiresIn)*time.Second)
	return f.acesso, conta, nil
}
