// Package service tem a regra de uso: orquestra e decide o que o front recebe. O service de lancamentos e o unico
// caminho de escrita de lancamento: os controllers chegam ao dado so por ele. Ele declara as interfaces de que precisa
// (repositorio e efeitos) e nao importa o SDK da AWS nem HTTP; ha um teste que falha se importar.
package service

import (
	"context"
	"errors"
	"log"
	"time"

	"financas/domain"
)

// Repositorio e o que o service precisa do banco, na linguagem do dominio. Nao ha metodo para editar ou apagar
// historico: quando o historico nascer, a interface so ganha o anexar.
type Repositorio interface {
	// Buscar devolve nil (sem erro) quando o lancamento nao existe.
	Buscar(ctx context.Context, sid, dataEvento, id string) (domain.Lancamento, error)
	// Salvar grava o estado novo. Sem Antes, so cria (falha se ja existir). Se a data mudou, troca a chave
	// antiga pela nova numa transacao.
	Salvar(ctx context.Context, sid string, m domain.Mudanca) error
	Excluir(ctx context.Context, sid, dataEvento, id string) error
	// GravarLote grava os itens (ja validados) e devolve os que o banco nao aceitou a tempo.
	GravarLote(ctx context.Context, sid string, itens []domain.Lancamento) []domain.Lancamento
}

// Travas consulta, ao longo de uma chamada, se o mes de um banco esta conferido (fechamento mensal).
// Checar devolve *domain.ErroMesConferido quando esta travado; qualquer outro erro e falha ao consultar.
type Travas interface {
	Checar(a domain.Alvo) error
}

// Efeitos sao os efeitos derivados de uma escrita de lancamento. Ficam fora da transacao do lancamento e uma
// falha neles nao desfaz a escrita (so vai para o log).
type Efeitos interface {
	Travas(ctx context.Context, sid string) Travas
	InvalidarSaldos(ctx context.Context, sid string, alvos []domain.Alvo)
	GarantirTags(ctx context.Context, sid string, tags []string)
	SincronizarAviso(ctx context.Context, sid string, l domain.Lancamento)
	RemoverAviso(ctx context.Context, sid, id string)
	// ComSeq soma 1 ao contador de alteracoes do espaco e anexa o valor novo na resposta (_seq).
	ComSeq(ctx context.Context, sid string, d domain.Doc) domain.Doc
}

// Erros do service. O controller traduz cada um na resposta HTTP de sempre.
var (
	ErrNaoEncontrado = errors.New("lancamento nao encontrado")
	ErrSoDono        = errors.New("so e possivel editar os lancamentos criados por voce")
	ErrSalvar        = errors.New("erro ao salvar")
	ErrExcluir       = errors.New("erro ao excluir")
	ErrInterno       = errors.New("erro interno")
)

// ErroValidacao e um 400 com a mensagem da regra que falhou.
type ErroValidacao struct{ Msg string }

func (e *ErroValidacao) Error() string { return e.Msg }

type Lancamentos struct {
	repo Repositorio
	ef   Efeitos
}

func NewLancamentos(repo Repositorio, ef Efeitos) *Lancamentos {
	return &Lancamentos{repo: repo, ef: ef}
}

const formatoTempo = "2006-01-02T15:04:05.000Z"

// escrita traduz o erro do repository: "dados invalidos" passa; o resto vira o erro de salvar ou excluir.
func escrita(op string, sentinela, err error) error {
	if errors.Is(err, domain.ErrDadosInvalidos) {
		return err
	}
	log.Println(op+":", err)
	return sentinela
}

// Criar grava um lancamento novo. O id, criadoEm e criadoPor sao do servidor.
func (s *Lancamentos) Criar(ctx context.Context, sid, sub string, body domain.Lancamento) (domain.Lancamento, error) {
	domain.Protegidos(body)
	if err := domain.ValidarTx(body); err != nil {
		return nil, &ErroValidacao{err.Error()}
	}
	body["id"] = domain.NovoID()
	body["criadoEm"] = time.Now().UTC().Format(formatoTempo)
	body["criadoPor"] = sub
	al, temAlvo := domain.AlvoDe(body)
	if temAlvo {
		if err := s.ef.Travas(ctx, sid).Checar(al); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Salvar(ctx, sid, domain.Mudanca{Depois: body}); err != nil {
		return nil, escrita("post tx", ErrSalvar, err)
	}
	if temAlvo {
		s.ef.InvalidarSaldos(ctx, sid, []domain.Alvo{al})
	}
	s.ef.GarantirTags(ctx, sid, domain.TagsDe(body))
	s.ef.SincronizarAviso(ctx, sid, body)
	return s.ef.ComSeq(ctx, sid, domain.Publico(body)), nil
}

// Editar aplica o patch ao lancamento que esta sob a data "de". restrito: so quem criou pode editar (member).
func (s *Lancamentos) Editar(ctx context.Context, sid, id, de string, patch domain.Lancamento, sub string, restrito bool) (domain.Lancamento, error) {
	de, err := domain.NormData(de)
	if err != nil {
		return nil, &ErroValidacao{"parametro de (dataEvento atual) obrigatorio"}
	}
	antigo, err := s.repo.Buscar(ctx, sid, de, id)
	if err != nil {
		return nil, ErrInterno
	}
	if antigo == nil {
		return nil, ErrNaoEncontrado
	}
	if restrito && !domain.EhDono(antigo, sub) {
		log.Printf("negado: sub=%s tentou editar lancamento de outra pessoa (%s)", sub, id)
		return nil, ErrSoDono
	}
	domain.Protegidos(patch)
	antes := domain.CopiaRasa(antigo)
	novo := antigo
	for k, v := range patch {
		novo[k] = v
	}
	if err := domain.ValidarTx(novo); err != nil {
		return nil, &ErroValidacao{err.Error()}
	}
	var alvos []domain.Alvo
	if domain.MudaSaldo(antes, novo) { // editar descricao, nota ou tags nao mexe no saldo e nao e travado
		for _, d := range []domain.Lancamento{antes, novo} {
			if al, ok := domain.AlvoDe(d); ok {
				alvos = append(alvos, al)
			}
		}
		tr := s.ef.Travas(ctx, sid)
		for _, al := range alvos {
			if err := tr.Checar(al); err != nil {
				return nil, err
			}
		}
	}
	if err := s.repo.Salvar(ctx, sid, domain.Mudanca{Antes: antes, Depois: novo, DataAnterior: de}); err != nil {
		return nil, escrita("put tx", ErrSalvar, err)
	}
	if len(alvos) > 0 {
		s.ef.InvalidarSaldos(ctx, sid, alvos)
	}
	s.ef.GarantirTags(ctx, sid, domain.TagsDe(novo))
	s.ef.SincronizarAviso(ctx, sid, novo)
	return s.ef.ComSeq(ctx, sid, domain.Publico(novo)), nil
}

// Excluir apaga o lancamento que esta sob a data "de". Excluir o que nao existe responde ok.
func (s *Lancamentos) Excluir(ctx context.Context, sid, id, de string) (domain.Doc, error) {
	de, err := domain.NormData(de)
	if err != nil {
		return nil, &ErroValidacao{"parametro de (dataEvento atual) obrigatorio"}
	}
	antigo, err := s.repo.Buscar(ctx, sid, de, id)
	if err != nil {
		return nil, ErrInterno
	}
	al, temAlvo := domain.AlvoDe(antigo)
	if antigo != nil && temAlvo {
		if err := s.ef.Travas(ctx, sid).Checar(al); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Excluir(ctx, sid, de, id); err != nil {
		return nil, escrita("del tx", ErrExcluir, err)
	}
	s.ef.RemoverAviso(ctx, sid, id)
	if antigo != nil && temAlvo {
		s.ef.InvalidarSaldos(ctx, sid, []domain.Alvo{al})
	}
	resposta := domain.Doc{"ok": true}
	if antigo != nil {
		s.ef.ComSeq(ctx, sid, resposta)
	}
	return resposta, nil
}

// Importar grava ate 25 lancamentos por chamada. Preserva id e criadoEm quando vierem no item, entao reimportar o
// mesmo backup sobrescreve em vez de duplicar. O que o banco nao aceitar a tempo volta em "pendentes".
func (s *Lancamentos) Importar(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error) {
	bruto, _ := body["itens"].([]any)
	if len(bruto) == 0 || len(bruto) > 25 {
		return nil, &ErroValidacao{"itens deve ter de 1 a 25 lancamentos"}
	}
	ts := time.Now().UTC().Format(formatoTempo)
	rejeitados := []domain.Doc{}
	vistas := map[string]bool{}
	tr := s.ef.Travas(ctx, sid)
	var alvos []domain.Alvo
	var aceitos []domain.Lancamento
	for i, x := range bruto {
		d, ok := x.(map[string]any)
		if !ok {
			rejeitados = append(rejeitados, domain.Doc{"indice": i, "motivo": "item invalido"})
			continue
		}
		if err := domain.ValidarTx(d); err != nil {
			rejeitados = append(rejeitados, domain.Doc{"indice": i, "motivo": err.Error()})
			continue
		}
		if id, _ := d["id"].(string); !domain.ReID.MatchString(id) {
			d["id"] = domain.NovoID()
		}
		if c, _ := d["criadoEm"].(string); c == "" {
			d["criadoEm"] = ts
		}
		d["criadoPor"] = sub
		if vistas[domain.Identidade(d)] {
			rejeitados = append(rejeitados, domain.Doc{"indice": i, "motivo": "duplicado no mesmo lote"})
			continue
		}
		vistas[domain.Identidade(d)] = true
		al, temAlvo := domain.AlvoDe(d)
		if temAlvo {
			if err := tr.Checar(al); err != nil {
				var te *domain.ErroMesConferido
				if !errors.As(err, &te) {
					log.Println("trava:", err)
					return nil, ErrInterno
				}
				rejeitados = append(rejeitados, domain.Doc{"indice": i, "motivo": "mes conferido e travado (" + te.Banco + ", " + te.Mes + ")"})
				continue
			}
		}
		aceitos = append(aceitos, d)
		if temAlvo {
			alvos = append(alvos, al)
		}
	}
	enviados := len(aceitos)
	var pend []domain.Lancamento
	if enviados > 0 {
		pend = s.repo.GravarLote(ctx, sid, aceitos)
		s.ef.InvalidarSaldos(ctx, sid, alvos)
	}
	pendentes := []domain.Doc{}
	naoGravados := map[string]bool{}
	for _, d := range pend {
		naoGravados[domain.Identidade(d)] = true
		pendentes = append(pendentes, domain.Publico(d))
	}
	for _, d := range aceitos {
		if !naoGravados[domain.Identidade(d)] {
			s.ef.SincronizarAviso(ctx, sid, d)
		}
	}
	resposta := domain.Doc{}
	if enviados > len(pend) {
		s.ef.ComSeq(ctx, sid, resposta) // algo foi gravado: quem mais estiver com a tela aberta precisa saber
	}
	resposta["gravados"], resposta["rejeitados"], resposta["pendentes"] = enviados-len(pend), rejeitados, pendentes
	return resposta, nil
}
