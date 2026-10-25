// Package controller recebe o request (rota, token, JSON), chama o service e traduz o resultado em resposta HTTP.
// Nao toca o banco: so o service (e, por baixo, o repository) chega ao dado.
package controller

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"financas/domain"
	"financas/service"
)

type Resp = events.LambdaFunctionURLResponse

// Out serializa v como JSON com o status dado.
func Out(status int, v any) (Resp, error) {
	b, _ := json.Marshal(v)
	return Resp{StatusCode: status, Headers: map[string]string{"content-type": "application/json"}, Body: string(b)}, nil
}

// Erro responde {"erro": msg} com o status dado.
func Erro(status int, msg string) (Resp, error) { return Out(status, domain.Doc{"erro": msg}) }

// respostaErro traduz os erros do service nas respostas HTTP de sempre.
func respostaErro(err error) (Resp, error) {
	var ev *service.ErroValidacao
	var mes *domain.ErroMesConferido
	var fechado *domain.ErroMesFechado
	var confl *service.ErroConflitoConfig
	var ordem *service.ErroOrdemFechamento
	var dif *service.ErroDiferenca
	var difFat *service.ErroDiferencaFatura
	var fatConf *domain.ErroFaturaConferida
	switch {
	case errors.As(err, &ev):
		return Erro(http.StatusBadRequest, ev.Error())
	case errors.Is(err, domain.ErrDadosInvalidos):
		return Erro(http.StatusBadRequest, err.Error())
	case errors.As(err, &confl): // configuracao gravada por outra pessoa: o cliente mescla com o estado atual
		return Out(http.StatusConflict, domain.Doc{"erro": confl.Error(), "atual": confl.Atual})
	case errors.As(err, &ordem): // conferir ou reabrir fora da sequencia
		return Out(http.StatusConflict, domain.Doc{"erro": ordem.Error(), "codigo": "ordem"})
	case errors.As(err, &dif): // o extrato nao bate com o saldo que o servidor calculou
		return Out(http.StatusConflict, domain.Doc{"erro": dif.Error(), "codigo": "diferenca", "calculado": dif.Calculado, "real": dif.Real, "diferenca": dif.Diferenca})
	case errors.As(err, &fatConf): // fatura conferida: so desfazendo a conferencia
		return Out(http.StatusConflict, domain.Doc{"erro": fatConf.Error(), "codigo": "fatura_conferida", "meio": fatConf.Meio, "ym": fatConf.Ym})
	case errors.As(err, &difFat): // o valor do app do cartao nao bate com o total da fatura (ou nao foi informado)
		return Out(http.StatusConflict, domain.Doc{"erro": difFat.Error(), "codigo": "diferenca", "total": difFat.Total, "valorBanco": difFat.ValorBanco, "diferenca": difFat.Diferenca})
	case errors.Is(err, service.ErrNaoPrevisto):
		return Out(http.StatusConflict, domain.Doc{"erro": err.Error(), "codigo": "nao_previsto"})
	case errors.Is(err, service.ErrCartaoNaoEncontrado), errors.Is(err, service.ErrFaturaNaoConferida):
		return Erro(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrTelegramNaoConfigurado):
		return Erro(http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, service.ErrTelegramRecusou):
		return Erro(http.StatusBadGateway, err.Error())
	case errors.Is(err, service.ErrMesNaoConferido):
		return Erro(http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrVinculoNaoEncontrado):
		return Erro(http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrConflitoVersao): // outra escrita chegou antes: nada foi gravado
		return Out(http.StatusConflict, domain.Doc{"erro": err.Error(), "codigo": "conflito_versao"})
	case errors.Is(err, service.ErrNaoEncontrado), errors.Is(err, service.ErrVersaoNaoAchada):
		return Erro(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrJaNessaVersao): // restaurar o que ja esta assim: nada a gravar
		return Out(http.StatusConflict, domain.Doc{"erro": err.Error(), "codigo": "ja_nessa_versao"})
	case errors.Is(err, service.ErrVersaoDeExclusao):
		return Out(http.StatusConflict, domain.Doc{"erro": err.Error(), "codigo": "versao_de_exclusao"})
	case errors.Is(err, service.ErrSoDono), errors.Is(err, service.ErrSemAcesso), errors.Is(err, service.ErrSoDonoRestaura):
		return Erro(http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrSalvar), errors.Is(err, service.ErrExcluir), errors.Is(err, service.ErrInterno), errors.Is(err, service.ErrListar), errors.Is(err, service.ErrSalvarTags):
		return Erro(http.StatusInternalServerError, err.Error())
	case errors.As(err, &fechado): // mes fechado por data: 409 confirmavel (reenviar com confirmaMesFechado=1)
		return Out(http.StatusConflict, domain.Doc{"erro": fechado.Error(), "codigo": "mes_fechado",
			"banco": fechado.Banco, "mes": fechado.Mes, "confirmavel": true})
	case errors.As(err, &mes): // mes conferido: 409 com o que o front precisa para explicar
		return Out(http.StatusConflict, domain.Doc{"erro": mes.Error(), "codigo": "mes_conferido",
			"banco": mes.Banco, "mes": mes.Mes, "conferidoAte": mes.Ate})
	}
	log.Println("trava:", err) // falha ao consultar a trava do fechamento
	return Erro(http.StatusInternalServerError, "erro interno")
}
