package controller

import (
	"context"
	"net/http"
	"strconv"

	"financas/domain"
	"financas/service"
)

// Lancamentos liga as rotas de lancamento ao service.
type Lancamentos struct{ svc *service.Lancamentos }

func NewLancamentos(svc *service.Lancamentos) *Lancamentos { return &Lancamentos{svc: svc} }

// PostTx: POST /spaces/{sid}/tx
func (c *Lancamentos) PostTx(ctx context.Context, sid, sub string, body domain.Doc, confirma bool) (Resp, error) {
	d, err := c.svc.Criar(ctx, sid, sub, body, confirma)
	if err != nil {
		return respostaErro(err)
	}
	return Out(201, d)
}

// PostTxBatch: POST /spaces/{sid}/tx/batch
func (c *Lancamentos) PostTxBatch(ctx context.Context, sid, sub string, body domain.Doc, confirma bool) (Resp, error) {
	d, err := c.svc.Importar(ctx, sid, sub, body, confirma)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PutTx: PUT /spaces/{sid}/tx/{id}?de=<dataEvento atual>. restrito: so quem criou pode editar (member).
func (c *Lancamentos) PutTx(ctx context.Context, sid, id, de string, body domain.Doc, sub string, restrito, confirma bool) (Resp, error) {
	d, err := c.svc.Editar(ctx, sid, id, de, body, sub, restrito, confirma)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// DelTx: DELETE /spaces/{sid}/tx/{id}?de=<dataEvento atual>. A exclusao e logica.
func (c *Lancamentos) DelTx(ctx context.Context, sid, id, de, sub string, confirma bool) (Resp, error) {
	d, err := c.svc.Excluir(ctx, sid, id, de, sub, confirma)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// GetHistorico: GET /spaces/{sid}/historico?mes=AAAA-MM|dias=N|tudo=1&q=. Exatamente um dos tres periodos, pela data da
// alteracao. restrito: so o que a pessoa criou (member).
func (c *Lancamentos) GetHistorico(ctx context.Context, sid string, qs map[string]string, sub string, restrito bool) (Resp, error) {
	per, err := service.PeriodoDaConsulta(qs["mes"], qs["dias"], qs["tudo"])
	if err != nil {
		return respostaErro(err)
	}
	d, err := c.svc.Historico(ctx, sid, per, qs["q"], sub, restrito)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// GetVersoes: GET /spaces/{sid}/tx/{id}/versoes?de=<dataEvento, so para lancamento sem historico>
func (c *Lancamentos) GetVersoes(ctx context.Context, sid, id, de, sub string, restrito bool) (Resp, error) {
	d, err := c.svc.Versoes(ctx, sid, id, de, sub, restrito)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PostRestaurar: POST /spaces/{sid}/tx/{id}/versoes/{n}/restaurar (corpo vazio).
func (c *Lancamentos) PostRestaurar(ctx context.Context, sid, id, versao, sub string, restrito, confirma bool) (Resp, error) {
	n, err := strconv.Atoi(versao)
	if err != nil || n < 1 {
		return Erro(http.StatusBadRequest, "versao invalida")
	}
	d, err := c.svc.Restaurar(ctx, sid, id, n, sub, restrito, confirma)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// GetTxs: GET /spaces/{sid}/tx?de=&ate=&banco=&status=&q=&ordem=asc&limite=&cursor=. limite: 1 a 1000, senao 200.
func (c *Lancamentos) GetTxs(ctx context.Context, sid string, qs map[string]string) (Resp, error) {
	f := domain.FiltroTx{De: qs["de"], Ate: qs["ate"], Banco: qs["banco"], Q: qs["q"], Status: qs["status"],
		Asc: qs["ordem"] == "asc", Limite: 200, Cursor: qs["cursor"]}
	if n, err := strconv.Atoi(qs["limite"]); err == nil && n > 0 && n <= 1000 {
		f.Limite = n
	}
	d, err := c.svc.ListarTx(ctx, sid, f)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PostConfirmar: POST /spaces/{sid}/tx/{id}/confirmar?de=<dataEvento atual> com { valor, dataEvento, status }. restrito: so quem criou (member).
func (c *Lancamentos) PostConfirmar(ctx context.Context, sid, id, de string, body domain.Doc, sub string, restrito, confirma bool) (Resp, error) {
	d, err := c.svc.Confirmar(ctx, sid, id, de, body, sub, restrito, confirma)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// GetFuturos: GET /spaces/{sid}/futuros[?ocultos=1]
func (c *Lancamentos) GetFuturos(ctx context.Context, sid string, qs map[string]string) (Resp, error) {
	d, err := c.svc.Futuros(ctx, sid, qs["ocultos"] == "1")
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// GetStatus: GET /spaces/{sid}/status?data=<ISO>[&todos=1]
func (c *Lancamentos) GetStatus(qs map[string]string) (Resp, error) {
	return Out(200, c.svc.StatusSugeridos(qs["data"], qs["todos"] == "1"))
}

// PostTransferencia: POST /spaces/{sid}/transferencias { origem, destino, valor, dataEvento? }
func (c *Lancamentos) PostTransferencia(ctx context.Context, sid, sub string, body domain.Doc, confirma bool) (Resp, error) {
	d, err := c.svc.Transferir(ctx, sid, sub, body, confirma)
	if err != nil {
		return respostaErro(err)
	}
	return Out(201, d)
}
