package main

import (
	"context"
	"errors"
	"log"
	"time"
)

// O service de lancamentos e o unico caminho de escrita de lancamento: os controllers chegam ao dado so por ele.
// Ele nao importa o SDK da AWS (ha um teste que falha se importar). Os efeitos derivados (saldos, tags, avisos, seq)
// ainda usam as funcoes de pacote de fechamento.go, tags.go, avisos.go e seq.go; sai dai nos proximos itens do 32.

// RepositorioLancamentos e o que o service precisa do banco, na linguagem do dominio. Nao ha metodo para editar
// ou apagar historico: quando o historico nascer, a interface so ganha o anexar.
type RepositorioLancamentos interface {
	// Buscar devolve nil (sem erro) quando o lancamento nao existe.
	Buscar(ctx context.Context, sid, dataEvento, id string) (Lancamento, error)
	// Salvar grava o estado novo. Sem Antes, so cria (falha se ja existir). Se a data mudou, troca a chave
	// antiga pela nova numa transacao.
	Salvar(ctx context.Context, sid string, m Mudanca) error
	Excluir(ctx context.Context, sid, dataEvento, id string) error
	// GravarLote grava os itens (ja validados) e devolve os que o banco nao aceitou a tempo.
	GravarLote(ctx context.Context, sid string, itens []Lancamento) []Lancamento
}

// Erros do service. O controller traduz cada um na resposta HTTP de sempre.
var (
	errNaoEncontrado  = errors.New("lancamento nao encontrado")
	errSoDono         = errors.New("so e possivel editar os lancamentos criados por voce")
	errSalvar         = errors.New("erro ao salvar")
	errExcluir        = errors.New("erro ao excluir")
	errInterno        = errors.New("erro interno")
	errDadosInvalidos = errors.New("dados invalidos")
)

// erroValidacao e um 400 com a mensagem da regra que falhou.
type erroValidacao struct{ msg string }

func (e *erroValidacao) Error() string { return e.msg }

type ServicoLancamentos struct{ repo RepositorioLancamentos }

func novoServicoLancamentos(repo RepositorioLancamentos) *ServicoLancamentos {
	return &ServicoLancamentos{repo: repo}
}

const formatoTempo = "2006-01-02T15:04:05.000Z"

// escrita traduz o erro do repository: "dados invalidos" passa; o resto vira o erro de salvar ou excluir.
func escrita(op string, sentinela, err error) error {
	if errors.Is(err, errDadosInvalidos) {
		return err
	}
	log.Println(op+":", err)
	return sentinela
}

// Criar grava um lancamento novo. O id, criadoEm e criadoPor sao do servidor.
func (s *ServicoLancamentos) Criar(ctx context.Context, sid, sub string, body Lancamento) (Lancamento, error) {
	protegidos(body)
	if err := validarTx(body); err != nil {
		return nil, &erroValidacao{err.Error()}
	}
	body["id"] = novoID()
	body["criadoEm"] = time.Now().UTC().Format(formatoTempo)
	body["criadoPor"] = sub
	al, temAlvo := alvoDe(body)
	if temAlvo {
		if err := novasTravas(ctx, sid).checar(al); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Salvar(ctx, sid, Mudanca{Depois: body}); err != nil {
		return nil, escrita("post tx", errSalvar, err)
	}
	if temAlvo {
		invalidarSaldos(ctx, sid, []alvo{al})
	}
	garantirTagsServidor(ctx, sid, tagsDe(body))
	sincronizarAviso(ctx, sid, body)
	return comSeq(ctx, sid, publico(body)), nil
}

// Editar aplica o patch ao lancamento que esta sob a data "de". restrito: so quem criou pode editar (member).
func (s *ServicoLancamentos) Editar(ctx context.Context, sid, id, de string, patch Lancamento, sub string, restrito bool) (Lancamento, error) {
	de, err := normData(de)
	if err != nil {
		return nil, &erroValidacao{"parametro de (dataEvento atual) obrigatorio"}
	}
	antigo, err := s.repo.Buscar(ctx, sid, de, id)
	if err != nil {
		return nil, errInterno
	}
	if antigo == nil {
		return nil, errNaoEncontrado
	}
	if restrito && !ehDono(antigo, sub) {
		log.Printf("negado: sub=%s tentou editar lancamento de outra pessoa (%s)", sub, id)
		return nil, errSoDono
	}
	protegidos(patch)
	antes := copiaRasa(antigo)
	novo := antigo
	for k, v := range patch {
		novo[k] = v
	}
	if err := validarTx(novo); err != nil {
		return nil, &erroValidacao{err.Error()}
	}
	var alvos []alvo
	if mudaSaldo(antes, novo) { // editar descricao, nota ou tags nao mexe no saldo e nao e travado
		for _, d := range []Lancamento{antes, novo} {
			if al, ok := alvoDe(d); ok {
				alvos = append(alvos, al)
			}
		}
		tr := novasTravas(ctx, sid)
		for _, al := range alvos {
			if err := tr.checar(al); err != nil {
				return nil, err
			}
		}
	}
	if err := s.repo.Salvar(ctx, sid, Mudanca{Antes: antes, Depois: novo, DataAnterior: de}); err != nil {
		return nil, escrita("put tx", errSalvar, err)
	}
	if len(alvos) > 0 {
		invalidarSaldos(ctx, sid, alvos)
	}
	garantirTagsServidor(ctx, sid, tagsDe(novo))
	sincronizarAviso(ctx, sid, novo)
	return comSeq(ctx, sid, publico(novo)), nil
}

// Excluir apaga o lancamento que esta sob a data "de". Excluir o que nao existe responde ok.
func (s *ServicoLancamentos) Excluir(ctx context.Context, sid, id, de string) (doc, error) {
	de, err := normData(de)
	if err != nil {
		return nil, &erroValidacao{"parametro de (dataEvento atual) obrigatorio"}
	}
	antigo, err := s.repo.Buscar(ctx, sid, de, id)
	if err != nil {
		return nil, errInterno
	}
	al, temAlvo := alvoDe(antigo)
	if antigo != nil && temAlvo {
		if err := novasTravas(ctx, sid).checar(al); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Excluir(ctx, sid, de, id); err != nil {
		return nil, escrita("del tx", errExcluir, err)
	}
	removerAviso(ctx, sid, id)
	if antigo != nil && temAlvo {
		invalidarSaldos(ctx, sid, []alvo{al})
	}
	resposta := doc{"ok": true}
	if antigo != nil {
		comSeq(ctx, sid, resposta)
	}
	return resposta, nil
}

// Importar grava ate 25 lancamentos por chamada. Preserva id e criadoEm quando vierem no item, entao reimportar o
// mesmo backup sobrescreve em vez de duplicar. O que o banco nao aceitar a tempo volta em "pendentes".
func (s *ServicoLancamentos) Importar(ctx context.Context, sid, sub string, body doc) (doc, error) {
	bruto, _ := body["itens"].([]any)
	if len(bruto) == 0 || len(bruto) > 25 {
		return nil, &erroValidacao{"itens deve ter de 1 a 25 lancamentos"}
	}
	ts := time.Now().UTC().Format(formatoTempo)
	rejeitados := []doc{}
	vistas := map[string]bool{}
	tr := novasTravas(ctx, sid)
	var alvos []alvo
	var aceitos []Lancamento
	for i, x := range bruto {
		d, ok := x.(map[string]any)
		if !ok {
			rejeitados = append(rejeitados, doc{"indice": i, "motivo": "item invalido"})
			continue
		}
		if err := validarTx(d); err != nil {
			rejeitados = append(rejeitados, doc{"indice": i, "motivo": err.Error()})
			continue
		}
		if id, _ := d["id"].(string); !reID.MatchString(id) {
			d["id"] = novoID()
		}
		if c, _ := d["criadoEm"].(string); c == "" {
			d["criadoEm"] = ts
		}
		d["criadoPor"] = sub
		if vistas[identidade(d)] {
			rejeitados = append(rejeitados, doc{"indice": i, "motivo": "duplicado no mesmo lote"})
			continue
		}
		vistas[identidade(d)] = true
		al, temAlvo := alvoDe(d)
		if temAlvo {
			if err := tr.checar(al); err != nil {
				var te *travaErro
				if !errors.As(err, &te) {
					log.Println("trava:", err)
					return nil, errInterno
				}
				rejeitados = append(rejeitados, doc{"indice": i, "motivo": "mes conferido e travado (" + te.banco + ", " + te.mes + ")"})
				continue
			}
		}
		aceitos = append(aceitos, d)
		if temAlvo {
			alvos = append(alvos, al)
		}
	}
	enviados := len(aceitos)
	var pend []Lancamento
	if enviados > 0 {
		pend = s.repo.GravarLote(ctx, sid, aceitos)
		invalidarSaldos(ctx, sid, alvos)
	}
	pendentes := []doc{}
	naoGravados := map[string]bool{}
	for _, d := range pend {
		naoGravados[identidade(d)] = true
		pendentes = append(pendentes, publico(d))
	}
	for _, d := range aceitos {
		if !naoGravados[identidade(d)] {
			sincronizarAviso(ctx, sid, d)
		}
	}
	resposta := doc{}
	if enviados > len(pend) {
		comSeq(ctx, sid, resposta) // algo foi gravado: quem mais estiver com a tela aberta precisa saber
	}
	resposta["gravados"], resposta["rejeitados"], resposta["pendentes"] = enviados-len(pend), rejeitados, pendentes
	return resposta, nil
}
