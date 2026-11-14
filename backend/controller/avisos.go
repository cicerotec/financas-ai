package controller

import (
	"context"
	"errors"
	"net/http"

	"financas/domain"
	"financas/service"
)

// Avisos liga o teste do aviso e a rotina diaria ao service.
type Avisos struct{ svc *service.Avisos }

func NewAvisos(svc *service.Avisos) *Avisos { return &Avisos{svc: svc} }

// frasesDoCanal e a unica tabela que liga um canal de aviso as frases da API (o service nao conhece canal: ele so devolve erros
// genericos). Canal sem entrada cai nas frases do canal desconhecido.
var frasesDoCanal = map[string]struct{ semDestino, naoConfigurado, recusou string }{
	"telegram": {
		semDestino:     "cadastre o seu chat id do Telegram primeiro",
		naoConfigurado: "o envio pelo Telegram ainda nao foi configurado no servidor",
		recusou:        "o Telegram recusou a mensagem: confira o chat id e se voce ja iniciou a conversa com o bot",
	},
}

// canalPadraoDoTeste e o canal do teste de aviso quando o corpo nao diz qual (compativel com o que a rota sempre fez).
const canalPadraoDoTeste = "telegram"

// PostAvisoTeste: POST /spaces/{sid}/avisos/teste. Manda uma mensagem de teste, pelo canal do corpo ({"canal": "..."}; sem ele,
// o Telegram), aos destinos de quem fez a chamada.
func (c *Avisos) PostAvisoTeste(ctx context.Context, sid, sub string, body domain.Doc) (Resp, error) {
	canal := canalPadraoDoTeste
	if v, tem := body["canal"]; tem {
		bruto, ok := v.(string)
		if !ok {
			return Erro(http.StatusBadRequest, "informe o canal como texto")
		}
		canal = bruto
	}
	err := c.svc.Testar(ctx, sid, sub, canal)
	frases := frasesDoCanal[canal]
	switch {
	case err == nil:
		return Out(200, domain.Doc{"ok": true})
	case errors.Is(err, service.ErrCanalDesconhecido):
		return Erro(http.StatusBadRequest, "canal de aviso desconhecido")
	case errors.Is(err, service.ErrSemDestino):
		return Erro(http.StatusBadRequest, orDefault(frases.semDestino, "cadastre o destino deste canal primeiro"))
	case errors.Is(err, domain.ErrCanalNaoConfigurado):
		return Erro(http.StatusServiceUnavailable, orDefault(frases.naoConfigurado, "o envio por este canal ainda nao foi configurado no servidor"))
	case errors.Is(err, domain.ErrEntregaFalhou):
		return Erro(http.StatusBadGateway, orDefault(frases.recusou, "o canal recusou a mensagem"))
	}
	return respostaErro(err)
}

func orDefault(v, padrao string) string {
	if v == "" {
		return padrao
	}
	return v
}

// Rodar e o evento diario do agendamento ({"acao":"avisos"}). Devolve {"enviados": n}.
func (c *Avisos) Rodar(ctx context.Context) (domain.Doc, error) {
	n, err := c.svc.Rodar(ctx)
	return domain.Doc{"enviados": n}, err
}
