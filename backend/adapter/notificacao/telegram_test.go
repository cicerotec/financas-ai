package notificacao

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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
		if err := tg.EnviarTexto(context.Background(), "111111", "olá"); err != nil {
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
		if err := tg.EnviarTexto(context.Background(), "1", "x"); !errors.Is(err, domain.ErrCanalNaoConfigurado) {
			t.Errorf("%s: erro = %v, esperado ErrCanalNaoConfigurado", nome, err)
		}
	}
	if len(*caminhos) != 0 {
		t.Errorf("nada deveria ter sido enviado: %v", *caminhos)
	}
}

func TestErroDoSSMNaoViraNaoConfigurado(t *testing.T) {
	tg := novo(&ssmFalso{err: errors.New("acesso negado")}, "/p", "http://invalido")
	err := tg.EnviarTexto(context.Background(), "1", "x")
	if err == nil || errors.Is(err, domain.ErrCanalNaoConfigurado) || !strings.Contains(err.Error(), "ler token no SSM") {
		t.Errorf("erro = %v", err)
	}
}

// V-T13: erro do Telegram. O corpo da resposta, a URL (com o token do bot) e o chat id nunca entram no erro.
func servidorTelegramComCorpo(t *testing.T, status int, corpo string) (*httptest.Server, *int) {
	chamadas := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamadas++
		w.WriteHeader(status)
		_, _ = io.WriteString(w, corpo)
	}))
	t.Cleanup(s.Close)
	return s, &chamadas
}

func TestErroDoTelegramNaoVazaCorpoTokenNemChatID(t *testing.T) {
	corpo := `{"ok":false,"error_code":500,"description":"falha 123:SEGREDO no chat 987654321 http://x/bot123:SEGREDO"}`
	for _, status := range []int{429, 500, 502} {
		srv, chamadas := servidorTelegramComCorpo(t, status, strings.Replace(corpo, "500", strconv.Itoa(status), 1))
		tg := novo(&ssmFalso{valor: texto("123:SEGREDO")}, "/p", srv.URL)
		err := tg.EnviarTexto(context.Background(), "987654321", "x")
		if err == nil || errors.Is(err, domain.ErrDestinoInvalido) || errors.Is(err, domain.ErrCredencialRecusada) {
			t.Fatalf("%d: erro = %v, esperado um erro comum", status, err)
		}
		for _, proibido := range []string{"SEGREDO", "987654321", "falha", "http", "bot123"} {
			if strings.Contains(err.Error(), proibido) {
				t.Errorf("%d: o erro leva %q: %v", status, proibido, err)
			}
		}
		if !strings.Contains(err.Error(), strconv.Itoa(status)) {
			t.Errorf("%d: o erro deveria dizer o codigo HTTP: %v", status, err)
		}
		if *chamadas != 1 {
			t.Errorf("%d: o erro foi repetido (%d chamadas)", status, *chamadas)
		}
	}
}

// 403 (bot bloqueado, conta apagada) e 400 "chat not found": o destino saiu, a rotina o apaga.
func TestTelegramDevolveDestinoInvalidoNo403E400ChatNotFound(t *testing.T) {
	casos := map[string]struct {
		status int
		corpo  string
	}{
		"403 bloqueado":      {403, `{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`},
		"403 conta apagada":  {403, `{"ok":false,"error_code":403,"description":"Forbidden: user is deactivated"}`},
		"400 chat not found": {400, `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`},
	}
	for nome, c := range casos {
		srv, _ := servidorTelegramComCorpo(t, c.status, c.corpo)
		err := novo(&ssmFalso{valor: texto("123:SEGREDO")}, "/p", srv.URL).EnviarTexto(context.Background(), "987654321", "x")
		if !errors.Is(err, domain.ErrDestinoInvalido) {
			t.Errorf("%s: erro = %v, esperado ErrDestinoInvalido", nome, err)
		}
		if err != nil && (strings.Contains(err.Error(), "SEGREDO") || strings.Contains(err.Error(), "987654321")) {
			t.Errorf("%s: o erro vaza token ou chat id: %v", nome, err)
		}
	}
}

// Um 400 que nao e "chat not found" (texto grande demais, por exemplo) nao apaga o destino de ninguem.
func TestTelegram400QueNaoEChatNotFoundNaoApagaODestino(t *testing.T) {
	srv, _ := servidorTelegramComCorpo(t, 400, `{"ok":false,"error_code":400,"description":"Bad Request: message is too long"}`)
	err := novo(&ssmFalso{valor: texto("123:SEGREDO")}, "/p", srv.URL).EnviarTexto(context.Background(), "1", "x")
	if err == nil || errors.Is(err, domain.ErrDestinoInvalido) || !strings.Contains(err.Error(), "400") {
		t.Errorf("erro = %v, esperado um erro comum com o 400", err)
	}
}

// 401 e a credencial do bot (token revogado): a rotina pula o canal e nao apaga ninguem.
func TestTelegram401DevolveCredencialRecusada(t *testing.T) {
	srv, _ := servidorTelegramComCorpo(t, 401, `{"ok":false,"error_code":401,"description":"Unauthorized"}`)
	err := novo(&ssmFalso{valor: texto("123:SEGREDO")}, "/p", srv.URL).EnviarTexto(context.Background(), "1", "x")
	if !errors.Is(err, domain.ErrCredencialRecusada) || errors.Is(err, domain.ErrDestinoInvalido) || strings.Contains(err.Error(), "SEGREDO") {
		t.Errorf("erro = %v, esperado ErrCredencialRecusada sem o token", err)
	}
}

// Erro de rede: a mensagem do http inclui a URL (e o token), entao o adaptador devolve um texto proprio.
func TestErroDeRedeNaoVazaOToken(t *testing.T) {
	tg := novo(&ssmFalso{valor: texto("123:SEGREDO")}, "/p", "http://127.0.0.1:1")
	err := tg.EnviarTexto(context.Background(), "1", "x")
	if err == nil || strings.Contains(err.Error(), "SEGREDO") || err.Error() != "falha de rede ao chamar o Telegram" {
		t.Errorf("erro = %v", err)
	}
}

// R11: o resumo sai numa mensagem so, com a contagem, pela API do bot.
func TestEnviarMandaUmaMensagemSoComAContagem(t *testing.T) {
	srv, caminhos, corpos := servidorTelegram(t, 200)
	tg := novo(&ssmFalso{valor: texto("123:ABC")}, "/p", srv.URL)
	var linhas []domain.LinhaAviso
	for i := 0; i < 100; i++ {
		linhas = append(linhas, domain.LinhaAviso{Kind: domain.AvisoAntes, N: 3, Venc: "2026-10-10", Tipo: "saida", Descricao: "Conta de teste numero " + strings.Repeat("x", 40), Valor: 10, Banco: "ITAU"})
	}
	if err := tg.Enviar(context.Background(), "111111", domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: linhas}); err != nil {
		t.Fatal(err)
	}
	if len(*caminhos) != 1 {
		t.Fatalf("esperada 1 mensagem, saiu %d", len(*caminhos))
	}
	c := (*corpos)[0]
	if c["chat_id"] != "111111" || c["text"] != "🔔 Avisos de 07/10/2026: 100 a vencer. Abra o app para ver o que é cada um." {
		t.Errorf("corpo = %v", c)
	}
}

// Sem token, o canal devolve ErrCanalNaoConfigurado e nada e enviado.
func TestSemTokenNadaEEnviado(t *testing.T) {
	srv, caminhos, _ := servidorTelegram(t, 200)
	tg := novo(&ssmFalso{}, "/p", srv.URL)
	err := tg.Enviar(context.Background(), "1", domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: []domain.LinhaAviso{{Kind: domain.AvisoHoje}}})
	if !errors.Is(err, domain.ErrCanalNaoConfigurado) || len(*caminhos) != 0 {
		t.Errorf("erro = %v, chamadas = %v", err, *caminhos)
	}
}
