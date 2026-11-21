package notificacao

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"financas/domain"
)

// ssmPorNome e um SSM falso com um valor por parametro, que conta as leituras de cada um (o segredo e o usuario sao lidos de parametros diferentes).
type ssmPorNome struct {
	mu       sync.Mutex
	valores  map[string]string
	leituras map[string]int
}

func novoSSMPorNome(valores map[string]string) *ssmPorNome {
	return &ssmPorNome{valores: valores, leituras: map[string]int{}}
}

func (s *ssmPorNome) GetParameter(_ context.Context, in *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nome := aws.ToString(in.Name)
	s.leituras[nome]++
	v, ok := s.valores[nome]
	if !ok {
		return &ssm.GetParameterOutput{}, nil
	}
	return &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: &v}}, nil
}

func (s *ssmPorNome) trocar(nome, valor string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.valores[nome] = valor
}

const (
	paramToken   = "/financas/dev/telegram-token"
	paramSegredo = "/financas/dev/telegram-webhook-secret"
	paramUsuario = "/financas/dev/telegram-bot-usuario"
)

func botDeTeste(valores map[string]string) (*Telegram, *ssmPorNome, *time.Time) {
	fake := novoSSMPorNome(valores)
	tg := NovoTelegram(fake, paramToken)
	tg.UsarVinculo(paramSegredo, paramUsuario)
	agora := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	tg.agora = func() time.Time { return agora }
	return tg, fake, &agora
}

// V-T3a: o segredo e lido com cache de no maximo 5 minutos; trocado no SSM, o novo vale depois disso e o antigo deixa de valer.
func TestSegredoDoWebhookTemCacheDeCincoMinutos(t *testing.T) {
	tg, fake, agora := botDeTeste(map[string]string{paramSegredo: "segredo-antigo_1"})
	ctx := context.Background()
	if s, err := tg.SegredoDoWebhook(ctx); err != nil || s != "segredo-antigo_1" {
		t.Fatalf("segredo = (%q, %v)", s, err)
	}
	fake.trocar(paramSegredo, "segredo-novo_2")

	*agora = agora.Add(4*time.Minute + 59*time.Second) // antes de 5 minutos: o do cache
	if s, _ := tg.SegredoDoWebhook(ctx); s != "segredo-antigo_1" {
		t.Errorf("antes de 5 minutos o cache vale, veio %q", s)
	}
	if fake.leituras[paramSegredo] != 1 {
		t.Errorf("o SSM foi lido %d vezes, esperado 1 ate aqui", fake.leituras[paramSegredo])
	}
	*agora = agora.Add(2 * time.Second) // passou dos 5 minutos
	if s, _ := tg.SegredoDoWebhook(ctx); s != "segredo-novo_2" {
		t.Errorf("depois de 5 minutos o segredo novo vale, veio %q", s)
	}
	if fake.leituras[paramSegredo] != 2 {
		t.Errorf("o SSM foi lido %d vezes, esperado 2", fake.leituras[paramSegredo])
	}
}

// Sem parametro, sem valor ou fora do formato (1 a 256 caracteres de A-Z a-z 0-9 _ -): nao configurado. Falha fechada.
func TestSegredoDoWebhookForaDoFormatoEhNaoConfigurado(t *testing.T) {
	longo := make([]byte, 257)
	for i := range longo {
		longo[i] = 'a'
	}
	casos := map[string]map[string]string{
		"sem valor":            {},
		"vazio":                {paramSegredo: "   "},
		"com espaco":           {paramSegredo: "abc def"},
		"com ponto":            {paramSegredo: "abc.def"},
		"com acento":           {paramSegredo: "segredó"},
		"mais de 256 caracter": {paramSegredo: string(longo)},
	}
	for nome, v := range casos {
		tg, _, _ := botDeTeste(v)
		if s, err := tg.SegredoDoWebhook(context.Background()); !errors.Is(err, domain.ErrCanalNaoConfigurado) || s != "" {
			t.Errorf("%s: (%q, %v), esperado ErrCanalNaoConfigurado", nome, s, err)
		}
	}
	semParametro := NovoTelegram(novoSSMPorNome(map[string]string{paramSegredo: "ok"}), paramToken) // UsarVinculo nunca chamado
	if _, err := semParametro.SegredoDoWebhook(context.Background()); !errors.Is(err, domain.ErrCanalNaoConfigurado) {
		t.Errorf("sem o nome do parametro: erro = %v, esperado ErrCanalNaoConfigurado", err)
	}
}

// R3: o link so sai com o token, o usuario do bot E o segredo do webhook; falta qualquer um e nao esta configurado.
func TestProntoExigeTokenUsuarioESegredo(t *testing.T) {
	todos := map[string]string{paramToken: "123:ABC", paramSegredo: "segredo_1", paramUsuario: "financas_dev_bot"}
	tg, _, _ := botDeTeste(todos)
	if u, err := tg.Pronto(context.Background()); err != nil || u != "financas_dev_bot" {
		t.Fatalf("Pronto = (%q, %v)", u, err)
	}
	for _, falta := range []string{paramToken, paramSegredo, paramUsuario} {
		v := map[string]string{}
		for k, val := range todos {
			if k != falta {
				v[k] = val
			}
		}
		tg, _, _ := botDeTeste(v)
		if _, err := tg.Pronto(context.Background()); !errors.Is(err, domain.ErrCanalNaoConfigurado) {
			t.Errorf("sem %s: erro = %v, esperado ErrCanalNaoConfigurado", falta, err)
		}
	}
}

// O usuario entra na URL do link: so letras, numeros e _ (sem @, barra, ? nem espaco).
func TestUsuarioDoBotForaDoFormatoEhNaoConfigurado(t *testing.T) {
	for _, ruim := range []string{"@meu_bot", "meu bot", "meu/bot", "meu_bot?start=x", "meu.bot", "https://t.me/x"} {
		tg, _, _ := botDeTeste(map[string]string{paramToken: "1:A", paramSegredo: "s", paramUsuario: ruim})
		if _, err := tg.UsuarioDoBot(context.Background()); !errors.Is(err, domain.ErrCanalNaoConfigurado) {
			t.Errorf("usuario %q: erro = %v, esperado ErrCanalNaoConfigurado", ruim, err)
		}
	}
}

// Erro do SSM nao vira "nao configurado": e falha de infraestrutura.
func TestErroDoSSMNoSegredoNaoViraNaoConfigurado(t *testing.T) {
	tg := NovoTelegram(&ssmFalso{err: errors.New("acesso negado")}, paramToken)
	tg.UsarVinculo(paramSegredo, paramUsuario)
	if _, err := tg.SegredoDoWebhook(context.Background()); err == nil || errors.Is(err, domain.ErrCanalNaoConfigurado) {
		t.Errorf("erro = %v", err)
	}
}
