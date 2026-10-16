package main

import (
	"context"
	"errors"
	"net/http"
)

// Controllers de lancamento: recebem o request, chamam o service e traduzem o resultado em resposta HTTP.
// Nao tocam o banco; so o service (e, por baixo, o repository) chega ao dado.

func postTx(ctx context.Context, svc *ServicoLancamentos, sid string, u *user, body doc) (resp, error) {
	d, err := svc.Criar(ctx, sid, u.Sub, body)
	if err != nil {
		return respostaErroLancamento(err)
	}
	return out(201, d)
}

func postTxBatch(ctx context.Context, svc *ServicoLancamentos, sid string, u *user, body doc) (resp, error) {
	d, err := svc.Importar(ctx, sid, u.Sub, body)
	if err != nil {
		return respostaErroLancamento(err)
	}
	return out(200, d)
}

func putTx(ctx context.Context, svc *ServicoLancamentos, sid, id, de string, body doc, sub string, restrito bool) (resp, error) {
	d, err := svc.Editar(ctx, sid, id, de, body, sub, restrito)
	if err != nil {
		return respostaErroLancamento(err)
	}
	return out(200, d)
}

func delTx(ctx context.Context, svc *ServicoLancamentos, sid, id, de string) (resp, error) {
	d, err := svc.Excluir(ctx, sid, id, de)
	if err != nil {
		return respostaErroLancamento(err)
	}
	return out(200, d)
}

// respostaErroLancamento traduz os erros do service nas respostas HTTP de sempre.
func respostaErroLancamento(err error) (resp, error) {
	var ev *erroValidacao
	switch {
	case errors.As(err, &ev):
		return erro(http.StatusBadRequest, ev.Error())
	case errors.Is(err, errDadosInvalidos):
		return erro(http.StatusBadRequest, err.Error())
	case errors.Is(err, errNaoEncontrado):
		return erro(http.StatusNotFound, err.Error())
	case errors.Is(err, errSoDono):
		return erro(http.StatusForbidden, err.Error())
	case errors.Is(err, errSalvar), errors.Is(err, errExcluir), errors.Is(err, errInterno):
		return erro(http.StatusInternalServerError, err.Error())
	}
	if r, ok := respTrava(err); ok { // mes conferido (409) ou falha ao consultar a trava (500, com log)
		return r, nil
	}
	return erro(http.StatusInternalServerError, "erro interno")
}
