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
	switch {
	case errors.As(err, &ev):
		return Erro(http.StatusBadRequest, ev.Error())
	case errors.Is(err, domain.ErrDadosInvalidos):
		return Erro(http.StatusBadRequest, err.Error())
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
	case errors.Is(err, service.ErrSalvar), errors.Is(err, service.ErrExcluir), errors.Is(err, service.ErrInterno):
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
