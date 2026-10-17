package notificacao

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"financas/domain"
)

// ssmFalso devolve um valor fixo e conta quantas vezes foi chamado.
type ssmFalso struct {
	valor    *string
	err      error
	chamadas int
	nome     string
	decifrou bool
}

func (s *ssmFalso) GetParameter(_ context.Context, in *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	s.chamadas++
	s.nome, s.decifrou = aws.ToString(in.Name), aws.ToBool(in.WithDecryption)
	if s.err != nil {
		return nil, s.err
	}
	if s.valor == nil {
		return &ssm.GetParameterOutput{}, nil
	}
	return &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: s.valor}}, nil
}

func texto(s string) *string { return &s }

// servidorTelegram sobe um Telegram falso que guarda o que recebeu e responde com o status dado.
func servidorTelegram(t *testing.T, status int) (*httptest.Server, *[]string, *[]map[string]any) {
	var caminhos []string
	var corpos []map[string]any
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caminhos = append(caminhos, r.Method+" "+r.URL.Path)
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		corpos = append(corpos, m)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"ok":false,"description":"chat not found"}`)
	}))
	t.Cleanup(s.Close)
	return s, &caminhos, &corpos
}

func novo(s SSM, param, base string) *Telegram {
	t := NovoTelegram(s, param)
	t.Base = base
	return t
}

// F4-T7: o corpo e a URL certos, o token lido uma vez (decifrado) e guardado.
func TestEnviaPelaAPIDoBotELeOTokenUmaVez(t *testing.T) {
	srv, caminhos, corpos := servidorTelegram(t, 200)
	fake := &ssmFalso{valor: texto("  123:ABC  ")}
	tg := novo(fake, "/financas/dev/telegram-token", srv.URL)
	for i := 0; i < 2; i++ {
		if err := tg.Enviar(context.Background(), "111111", "olá"); err != nil {
			t.Fatal(err)
		}
	}
	if (*caminhos)[0] != "POST /bot123:ABC/sendMessage" {
		t.Errorf("caminho = %q (o token e aparado e vai na URL)", (*caminhos)[0])
	}
	c := (*corpos)[0]
	if c["chat_id"] != "111111" || c["text"] != "olá" || c["disable_web_page_preview"] != true {
		t.Errorf("corpo = %v", c)
	}
	if fake.chamadas != 1 || fake.nome != "/financas/dev/telegram-token" || !fake.decifrou {
		t.Errorf("SSM: %d chamada(s), nome=%q, decifrou=%v; esperado 1, o parametro e WithDecryption", fake.chamadas, fake.nome, fake.decifrou)
	}
}

// Sem parametro, sem valor ou sem client o canal nao esta configurado (o service trata como "nada a enviar", nao como erro).
func TestSemTokenOCanalNaoEstaConfigurado(t *testing.T) {
	srv, caminhos, _ := servidorTelegram(t, 200)
	casos := map[string]*Telegram{
		"sem parametro":   novo(&ssmFalso{valor: texto("x")}, "", srv.URL),
		"valor vazio":     novo(&ssmFalso{valor: texto("   ")}, "/p", srv.URL),
		"parametro vazio": novo(&ssmFalso{}, "/p", srv.URL),
		"sem client SSM":  novo(nil, "/p", srv.URL),
	}
	for nome, tg := range casos {
		if err := tg.Enviar(context.Background(), "1", "x"); !errors.Is(err, domain.ErrCanalNaoConfigurado) {
			t.Errorf("%s: erro = %v, esperado ErrCanalNaoConfigurado", nome, err)
		}
	}
	if len(*caminhos) != 0 {
		t.Errorf("nada deveria ter sido enviado: %v", *caminhos)
	}
}

func TestErroDoSSMNaoViraNaoConfigurado(t *testing.T) {
	tg := novo(&ssmFalso{err: errors.New("acesso negado")}, "/p", "http://invalido")
	err := tg.Enviar(context.Background(), "1", "x")
	if err == nil || errors.Is(err, domain.ErrCanalNaoConfigurado) || !strings.Contains(err.Error(), "ler token no SSM") {
		t.Errorf("erro = %v", err)
	}
}

// Resposta que nao e 2xx: o erro leva o status e o corpo, nunca o token.
func TestRespostaRuimDoTelegramNaoVazaOToken(t *testing.T) {
	srv, _, _ := servidorTelegram(t, 400)
	tg := novo(&ssmFalso{valor: texto("123:SEGREDO")}, "/p", srv.URL)
	err := tg.Enviar(context.Background(), "1", "x")
	if err == nil || !strings.Contains(err.Error(), "telegram respondeu 400") || strings.Contains(err.Error(), "SEGREDO") {
		t.Errorf("erro = %v", err)
	}
}

// Erro de rede: a mensagem do http inclui a URL (e o token), entao o adaptador devolve um texto proprio.
func TestErroDeRedeNaoVazaOToken(t *testing.T) {
	tg := novo(&ssmFalso{valor: texto("123:SEGREDO")}, "/p", "http://127.0.0.1:1")
	err := tg.Enviar(context.Background(), "1", "x")
	if err == nil || strings.Contains(err.Error(), "SEGREDO") || err.Error() != "falha de rede ao chamar o Telegram" {
		t.Errorf("erro = %v", err)
	}
}
