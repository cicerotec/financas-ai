package service

import (
	"context"
	"log"

	"financas/domain"
)

// Meio desativado (item 58, .ai/specs/meios-de-pagamento, regras R14 a R17): desativar e encerrar a conta, e depois disso nada novo entra no meio.
// A guarda le CFG#LISTAS uma vez por chamada e e usada por todo caminho que grava um lancamento.

// guardaDeMeio consulta, ao longo de uma chamada, se um meio pode receber lancamentos. Sem a configuracao ligada (testes de unidade), tudo passa.
type guardaDeMeio struct {
	s      *Lancamentos
	ctx    context.Context
	sid    string
	lido   bool
	listas domain.Doc
}

func (s *Lancamentos) guardaDeMeio(ctx context.Context, sid string) *guardaDeMeio {
	return &guardaDeMeio{s: s, ctx: ctx, sid: sid}
}

func (g *guardaDeMeio) carregar() (domain.Doc, error) {
	if g.lido {
		return g.listas, nil
	}
	g.lido = true
	if g.s.cfg == nil {
		return nil, nil
	}
	l, err := g.s.cfg.Ler(g.ctx, g.sid, "listas")
	if err != nil {
		log.Println("guarda de meio: listas:", err)
		return nil, ErrInterno
	}
	g.listas = l
	return l, nil
}

// Desativado diz se o meio ja deixou de valer hoje.
func (g *guardaDeMeio) Desativado(meio string) (bool, error) {
	l, err := g.carregar()
	if err != nil || l == nil || meio == "" {
		return false, err
	}
	return domain.MeioDesativado(l, meio, domain.HojeDe(g.s.rel.Agora())), nil
}

// ExigirAtivo recusa (409 MEIO_DESATIVADO) o lancamento num meio desativado.
func (g *guardaDeMeio) ExigirAtivo(meio string) error {
	off, err := g.Desativado(meio)
	if err != nil {
		return err
	}
	if off {
		return &domain.ErroMeioDesativado{Meio: meio}
	}
	return nil
}

// ExigirNaListaEAtivo e o ExigirAtivo de quem traz um registro de volta (restaurar): o meio tambem precisa ainda estar em Listas.
func (g *guardaDeMeio) ExigirNaListaEAtivo(meio string) error {
	l, err := g.carregar()
	if err != nil || l == nil || meio == "" {
		return err
	}
	if !domain.MeioNaLista(l, meio) {
		return &domain.ErroMeioDesativado{Meio: meio, Removido: true}
	}
	return g.ExigirAtivo(meio)
}

func bancoDe(l domain.Lancamento) string {
	b, _ := l["banco"].(string)
	return b
}

// dataInicioDoMeio le o inicio do controle do meio em CFG#SALDOS ("" se nao tem).
func (s *Lancamentos) dataInicioDoMeio(ctx context.Context, sid, meio string) (string, error) {
	if s.cfg == nil {
		return "", nil
	}
	d, err := s.cfg.Ler(ctx, sid, "saldos")
	if err != nil {
		log.Println("guarda de meio: saldos:", err)
		return "", ErrInterno
	}
	return domain.ConfigBancoDe(d, meio).DataInicio, nil
}

// PodeEditar confere uma edicao: num lancamento de meio desativado so descricao, nota e tags mudam; trocar o banco para um meio desativado e recusado.
func (g *guardaDeMeio) PodeEditar(antes, depois domain.Lancamento) error {
	off, err := g.Desativado(bancoDe(antes))
	if err != nil {
		return err
	}
	if off && !domain.SoMudaCamposLivres(antes, depois) {
		return &domain.ErroMeioDesativado{Meio: bancoDe(antes), Motivo: "o meio esta desativado: so a descricao, a nota e as tags do lancamento podem mudar"}
	}
	if bancoDe(depois) != bancoDe(antes) {
		return g.ExigirAtivo(bancoDe(depois))
	}
	return nil
}

// PodeExcluir recusa excluir um lancamento de meio desativado (mudaria um saldo encerrado).
func (g *guardaDeMeio) PodeExcluir(l domain.Lancamento) error {
	off, err := g.Desativado(bancoDe(l))
	if err != nil {
		return err
	}
	if off {
		return &domain.ErroMeioDesativado{Meio: bancoDe(l), Motivo: "o meio esta desativado: reative-o em Listas para excluir lancamentos dele"}
	}
	return nil
}

// importavelNoMeio confere um registro de importacao (item 58, R16): num meio desativado so entra o anterior ao inicio do controle do meio. Meio ativo passa direto.
func (s *Lancamentos) importavelNoMeio(ctx context.Context, sid string, g *guardaDeMeio, d domain.Lancamento) error {
	off, err := g.Desativado(bancoDe(d))
	if err != nil || !off {
		return err
	}
	return s.anteriorAoInicio(ctx, sid, bancoDe(d), d)
}

// anteriorAoInicio recusa o registro de um meio desativado cuja data nao e anterior ao inicio do controle do meio (ou que o meio nao tem).
func (s *Lancamentos) anteriorAoInicio(ctx context.Context, sid, meio string, d domain.Lancamento) error {
	ini, err := s.dataInicioDoMeio(ctx, sid, meio)
	if err != nil {
		return err
	}
	data, _ := d["dataEvento"].(string)
	if !domain.ImportavelEmMeioDesativado(data, ini) {
		return &domain.ErroMeioDesativado{Meio: meio, Motivo: "o meio esta desativado: so entram registros anteriores ao inicio do controle dele"}
	}
	return nil
}

// importavelSobre confere o registro que o lote vai sobrescrever (ou ressuscitar): se o gravado esta num meio desativado, ele tambem precisa ser anterior ao inicio do controle.
func (s *Lancamentos) importavelSobre(ctx context.Context, sid string, g *guardaDeMeio, antigo domain.Lancamento) error {
	if antigo == nil {
		return nil
	}
	off, err := g.Desativado(bancoDe(antigo))
	if err != nil || !off {
		return err
	}
	return s.anteriorAoInicio(ctx, sid, bancoDe(antigo), antigo)
}
