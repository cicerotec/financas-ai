package notificacao

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"financas/domain"
)

// Tudo aqui usa conta de servico, chave e token FICTICIOS, gerados no proprio teste.

const aparelhoFicticio = "tok-ficticio-do-aparelho-0123456789abcdef"

type fcmFalso struct {
	chave          *rsa.PrivateKey
	json           string
	tokenEndpoint  *httptest.Server
	fcm            *httptest.Server
	pedidosToken   int
	envios         []map[string]any
	cabecalhos     []string
	statusToken    int
	statusEnvio    int
	corpoEnvio     string
	assertions     []string
	formulariosTok []string
}

func novoFCMFalso(t *testing.T) *fcmFalso {
	t.Helper()
	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemChave := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustPKCS8(t, chave)}))
	cred, _ := json.Marshal(map[string]string{"client_email": "robo@projeto-ficticio.iam.gserviceaccount.com", "private_key": pemChave, "project_id": "projeto-ficticio"})
	f := &fcmFalso{chave: chave, json: string(cred), statusToken: 200, statusEnvio: 200}
	f.tokenEndpoint = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.pedidosToken++
		b, _ := io.ReadAll(r.Body)
		f.formulariosTok = append(f.formulariosTok, string(b))
		w.WriteHeader(f.statusToken)
		_, _ = io.WriteString(w, `{"access_token":"acesso-ficticio-`+string(rune('0'+f.pedidosToken))+`","expires_in":3600,"token_type":"Bearer"}`)
	}))
	f.fcm = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		f.envios = append(f.envios, m)
		f.cabecalhos = append(f.cabecalhos, r.Method+" "+r.URL.Path+" "+r.Header.Get("authorization"))
		w.WriteHeader(f.statusEnvio)
		_, _ = io.WriteString(w, f.corpoEnvio)
	}))
	t.Cleanup(func() { f.tokenEndpoint.Close(); f.fcm.Close() })
	return f
}

func mustPKCS8(t *testing.T, k *rsa.PrivateKey) []byte {
	b, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (f *fcmFalso) canal(agora func() time.Time) *FCM {
	c := NovoFCM(&ssmFalso{valor: &f.json}, "/financas/dev/fcm-credencial")
	c.Base, c.AuthURL = f.fcm.URL, f.tokenEndpoint.URL
	if agora != nil {
		c.agora = agora
	}
	return c
}

func resumoDe(linhas ...domain.LinhaAviso) domain.ResumoAvisos {
	return domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: linhas}
}

// P-T7: a notificacao so tem a contagem, mesmo que as linhas tenham descricao, valor, meio e data.
func TestPushSoTemAContagem(t *testing.T) {
	f := novoFCMFalso(t)
	linhas := []domain.LinhaAviso{
		{Kind: domain.AvisoAtrasado, N: 2, Venc: "2026-10-05", Tipo: "saida", Descricao: "Condominio do bloco secreto", Valor: 1234.56, Banco: "BANCO-SECRETO"},
		{Kind: domain.AvisoAtrasado, N: 1, Venc: "2026-10-06", Tipo: "entrada", Descricao: "Aluguel", Valor: 2500, Banco: "ITAU"},
		{Kind: domain.AvisoHoje, Venc: "2026-10-07", Tipo: "saida", Descricao: "Luz", Valor: 80, Banco: "ITAU"},
		{Kind: domain.AvisoAntes, N: 3, Venc: "2026-10-10", Tipo: "saida", Descricao: "Agua", Valor: 40, Banco: "ITAU"},
		{Kind: domain.AvisoAntes, N: 3, Venc: "2026-10-10", Tipo: "saida", Descricao: "Gas", Valor: 40, Banco: "ITAU"},
	}
	if err := f.canal(nil).Enviar(context.Background(), aparelhoFicticio, resumoDe(linhas...)); err != nil {
		t.Fatal(err)
	}
	bruto, _ := json.Marshal(f.envios[0])
	for _, proibido := range []string{"Condominio", "SECRETO", "234,56", "1.234", "Aluguel", "Luz", "Agua", "Gas", "ITAU", "2026", "10/10", "R$"} {
		if strings.Contains(string(bruto), proibido) {
			t.Errorf("a notificacao contem %q: %s", proibido, bruto)
		}
	}
	n := f.envios[0]["message"].(map[string]any)["notification"].(map[string]any)
	if n["title"] != "Finanças" || n["body"] != "2 atrasados · 1 vence hoje · 2 a vencer" {
		t.Errorf("notificacao = %v", n)
	}
}

func TestTextoDoPushSingularPluralMarcaEZero(t *testing.T) {
	for _, c := range []struct {
		in           domain.ContagemAvisos
		titulo, body string
	}{
		{domain.ContagemAvisos{Atrasados: 1}, "Finanças", "1 atrasado"},
		{domain.ContagemAvisos{Hoje: 1}, "Finanças", "1 vence hoje"},
		{domain.ContagemAvisos{Hoje: 3}, "Finanças", "3 vencem hoje"},
		{domain.ContagemAvisos{AVencer: 4, Marca: "[DEV] "}, "[DEV] Finanças", "4 a vencer"},
		{domain.ContagemAvisos{Teste: true, Marca: "[DEV] "}, "[DEV] Finanças", "Teste: os avisos vão chegar aqui."},
	} {
		if titulo, corpo := TextoDoPush(c.in); titulo != c.titulo || corpo != c.body {
			t.Errorf("%+v = (%q, %q), esperado (%q, %q)", c.in, titulo, corpo, c.titulo, c.body)
		}
	}
}

// P-T8: o JWT e o pedido de token de acesso; o token e reaproveitado e renovado quando vence; o corpo do envio e o do spec.
func TestFCMAutenticaComJWTECacheiaOToken(t *testing.T) {
	f := novoFCMFalso(t)
	agora := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)
	c := f.canal(func() time.Time { return agora })
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := c.Enviar(ctx, aparelhoFicticio, resumoDe(domain.LinhaAviso{Kind: domain.AvisoHoje})); err != nil {
			t.Fatal(err)
		}
	}
	if f.pedidosToken != 1 {
		t.Fatalf("o token de acesso deveria ser pedido uma vez para dois envios, foi %d", f.pedidosToken)
	}
	form := f.formulariosTok[0]
	if !strings.Contains(form, "grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Ajwt-bearer") {
		t.Errorf("grant_type = %s", form)
	}
	assertion := strings.Split(strings.Split(form, "assertion=")[1], "&")[0]
	var claims jwt.MapClaims
	if _, err := jwt.ParseWithClaims(assertion, &claims, func(*jwt.Token) (any, error) { return &f.chave.PublicKey, nil },
		jwt.WithValidMethods([]string{"RS256"}), jwt.WithTimeFunc(func() time.Time { return agora })); err != nil {
		t.Fatalf("o JWT nao confere com a chave: %v", err)
	}
	iat, _ := claims["iat"].(float64)
	exp, _ := claims["exp"].(float64)
	if claims["iss"] != "robo@projeto-ficticio.iam.gserviceaccount.com" || claims["scope"] != "https://www.googleapis.com/auth/firebase.messaging" ||
		claims["aud"] != "https://oauth2.googleapis.com/token" || exp-iat > 3600 || iat != float64(agora.Unix()) {
		t.Errorf("claims = %v", claims)
	}
	if f.cabecalhos[0] != "POST /v1/projects/projeto-ficticio/messages:send Bearer acesso-ficticio-1" {
		t.Errorf("pedido de envio = %q", f.cabecalhos[0])
	}
	msg := f.envios[0]["message"].(map[string]any)
	android := msg["android"].(map[string]any)
	if msg["token"] != aparelhoFicticio || android["ttl"] != "86400s" || android["notification"].(map[string]any)["channel_id"] != "avisos" {
		t.Errorf("mensagem = %v", msg)
	}
	agora = agora.Add(56 * time.Minute) // falta menos que a folga de 5 min: pede outro
	if err := c.Enviar(ctx, aparelhoFicticio, resumoDe()); err != nil {
		t.Fatal(err)
	}
	if f.pedidosToken != 2 {
		t.Errorf("perto de vencer deveria pedir outro token de acesso, pedidos = %d", f.pedidosToken)
	}
}

// P-T9: sem credencial o canal nao esta configurado e nada sai; erro do SSM nao vira "nao configurado".
func TestFCMSemCredencialNaoEstaConfigurado(t *testing.T) {
	f := novoFCMFalso(t)
	incompleto := `{"client_email":"x@y.iam.gserviceaccount.com","project_id":"p"}`
	casos := map[string]*FCM{
		"sem parametro":   NovoFCM(&ssmFalso{valor: &f.json}, ""),
		"sem client SSM":  NovoFCM(nil, "/p"),
		"valor vazio":     NovoFCM(&ssmFalso{valor: texto("  ")}, "/p"),
		"parametro vazio": NovoFCM(&ssmFalso{}, "/p"),
		"json incompleto": NovoFCM(&ssmFalso{valor: &incompleto}, "/p"),
		"nao e json":      NovoFCM(&ssmFalso{valor: texto("lixo")}, "/p"),
	}
	for nome, c := range casos {
		c.Base, c.AuthURL = f.fcm.URL, f.tokenEndpoint.URL
		if err := c.Enviar(context.Background(), aparelhoFicticio, resumoDe()); !errors.Is(err, domain.ErrCanalNaoConfigurado) {
			t.Errorf("%s: erro = %v, esperado ErrCanalNaoConfigurado", nome, err)
		}
	}
	if f.pedidosToken != 0 || len(f.envios) != 0 {
		t.Errorf("nenhuma chamada de rede era esperada: token=%d envios=%d", f.pedidosToken, len(f.envios))
	}
	c := NovoFCM(&ssmFalso{err: errors.New("acesso negado")}, "/p")
	if err := c.Enviar(context.Background(), aparelhoFicticio, resumoDe()); err == nil || errors.Is(err, domain.ErrCanalNaoConfigurado) {
		t.Errorf("erro do SSM = %v: nao pode virar nao configurado", err)
	}
	f.statusToken = 503
	if err := f.canal(nil).Enviar(context.Background(), aparelhoFicticio, resumoDe()); err == nil || errors.Is(err, domain.ErrCanalNaoConfigurado) || errors.Is(err, domain.ErrCredencialRecusada) {
		t.Errorf("falha do endpoint de token = %v: nem nao configurado nem credencial recusada", err)
	}
}

// P-T10: UNREGISTERED apaga; credencial recusada e outra coisa; nenhum erro leva URL, token ou corpo; nada e repetido.
func TestFCMErrosNaoVazamENaoSaoRepetidos(t *testing.T) {
	unreg := `{"error":{"code":404,"status":"NOT_FOUND","message":"Requested entity was not found. SEGREDO-NO-CORPO","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`
	sender := `{"error":{"code":403,"status":"PERMISSION_DENIED","message":"x SEGREDO-NO-CORPO","details":[{"errorCode":"SENDER_ID_MISMATCH"}]}}`
	casos := []struct {
		nome   string
		status int
		corpo  string
		quer   func(error) bool
	}{
		{"UNREGISTERED", 404, unreg, func(e error) bool { return errors.Is(e, domain.ErrDestinoInvalido) }},
		{"404 sem codigo", 404, `{"error":{"message":"SEGREDO-NO-CORPO"}}`, func(e error) bool { return e != nil && !errors.Is(e, domain.ErrDestinoInvalido) }},
		{"400", 400, `{"error":{"details":[{"errorCode":"INVALID_ARGUMENT"}],"message":"SEGREDO-NO-CORPO"}}`, func(e error) bool {
			return e != nil && strings.Contains(e.Error(), "400 INVALID_ARGUMENT") && !errors.Is(e, domain.ErrDestinoInvalido)
		}},
		{"401", 401, `SEGREDO-NO-CORPO`, func(e error) bool { return errors.Is(e, domain.ErrCredencialRecusada) }},
		{"403 de credencial", 403, `{"error":{"message":"SEGREDO-NO-CORPO"}}`, func(e error) bool { return errors.Is(e, domain.ErrCredencialRecusada) }},
		{"403 sender", 403, sender, func(e error) bool { return e != nil && !errors.Is(e, domain.ErrCredencialRecusada) }},
		{"429", 429, `SEGREDO-NO-CORPO`, func(e error) bool { return e != nil && strings.Contains(e.Error(), "429") }},
		{"503", 503, `SEGREDO-NO-CORPO`, func(e error) bool { return e != nil && strings.Contains(e.Error(), "503") }},
	}
	for _, c := range casos {
		f := novoFCMFalso(t)
		f.statusEnvio, f.corpoEnvio = c.status, c.corpo
		err := f.canal(nil).Enviar(context.Background(), aparelhoFicticio, resumoDe())
		if !c.quer(err) {
			t.Errorf("%s: erro = %v", c.nome, err)
			continue
		}
		for _, proibido := range []string{"SEGREDO", aparelhoFicticio, "acesso-ficticio", f.fcm.URL, "Bearer", "projeto-ficticio", "http"} {
			if err != nil && strings.Contains(err.Error(), proibido) && !errors.Is(err, domain.ErrDestinoInvalido) {
				t.Errorf("%s: o erro contem %q: %v", c.nome, proibido, err)
			}
		}
		if len(f.envios) != 1 {
			t.Errorf("%s: o envio nao pode ser repetido (%d chamadas)", c.nome, len(f.envios))
		}
	}
	// o endpoint de token recusa (JWT ruim ou chave revogada): credencial recusada, e nenhum envio
	f := novoFCMFalso(t)
	f.statusToken = 400
	if err := f.canal(nil).Enviar(context.Background(), aparelhoFicticio, resumoDe()); !errors.Is(err, domain.ErrCredencialRecusada) || len(f.envios) != 0 {
		t.Errorf("token recusado: erro = %v, envios = %d", err, len(f.envios))
	}
	// erro de rede: texto proprio, sem a URL
	g := novoFCMFalso(t)
	c := g.canal(nil)
	g.fcm.Close()
	if err := c.Enviar(context.Background(), aparelhoFicticio, resumoDe()); err == nil || strings.Contains(err.Error(), "127.0.0.1") || err.Error() != "falha de rede ao chamar o FCM" {
		t.Errorf("erro de rede = %v", err)
	}
}
