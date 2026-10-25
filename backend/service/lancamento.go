// Package service tem a regra de uso: orquestra e decide o que o front recebe. O service de lancamentos e o unico
// caminho de escrita de lancamento: os controllers chegam ao dado so por ele. Ele declara as interfaces de que precisa
// (repositorio e efeitos) e nao importa o SDK da AWS nem HTTP; ha um teste que falha se importar.
package service

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"financas/domain"
)

// Repositorio e o que o service precisa do banco, na linguagem do dominio. Nao ha metodo para editar nem apagar
// versao do historico: toda escrita grava o lancamento e as suas versoes juntos, e so acrescenta.
type Repositorio interface {
	// Buscar devolve o lancamento vivo, inclusive o excluido logicamente, ou nil (sem erro) se nao existe.
	Buscar(ctx context.Context, sid, dataEvento, id string) (domain.Lancamento, error)
	// UltimaVersao devolve o maior numero de versao do lancamento; 0 se ele nao tem historico.
	UltimaVersao(ctx context.Context, sid, id string) (int, error)
	// Salvar grava o lancamento e as suas versoes numa transacao. Sem Antes, so cria. Se a data mudou, troca a chave
	// antiga pela nova. Devolve domain.ErrConflitoVersao se outra escrita gravou a mesma versao antes.
	Salvar(ctx context.Context, sid string, m domain.Mudanca) error
	// SalvarLote e o Salvar de varias mudancas, tudo ou nada (importacao).
	SalvarLote(ctx context.Context, sid string, ms []domain.Mudanca) error
	// Versoes devolve o historico do lancamento, da primeira versao para a ultima (vazio se nao tem).
	Versoes(ctx context.Context, sid, id string) ([]domain.Versao, error)
	// Listar devolve os lancamentos vivos do espaco, em paginas; proximo e o cursor da seguinte ("" se acabou).
	Listar(ctx context.Context, sid string, f domain.FiltroTx) (itens []domain.Doc, proximo string, err error)
	// VersoesDoEspaco devolve as versoes de todos os lancamentos do espaco, ate o limite; truncado diz que havia mais.
	VersoesDoEspaco(ctx context.Context, sid string, limite int) ([]domain.Versao, bool, error)
}

// Travas consulta, ao longo de uma chamada, se o mes de um banco pode receber a escrita. Checar devolve:
//   - *domain.ErroMesConferido: o mes foi conferido com o extrato (absoluto: a confirmacao nao o destrava);
//   - *domain.ErroMesFechado: o mes fechou por data e falta a confirmacao explicita (confirmado = false);
//   - qualquer outro erro: falha ao consultar.
//
// exigiu diz que o mes esta fechado por data e a confirmacao dada era necessaria (vai para a versao do historico).
type Travas interface {
	Checar(a domain.Alvo, confirmado bool) (exigiu bool, err error)
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
	// Seq le o contador de alteracoes sem somar (a leitura barata que diz se alguem gravou); ok=false se nao der para ler.
	Seq(ctx context.Context, sid string) (n float64, ok bool)
}

// Erros do service. O controller traduz cada um na resposta HTTP de sempre.
var (
	ErrNaoEncontrado = errors.New("lancamento nao encontrado")
	ErrSoDono        = errors.New("so e possivel editar os lancamentos criados por voce")
	ErrSalvar        = errors.New("erro ao salvar")
	ErrExcluir       = errors.New("erro ao excluir")
	ErrInterno       = errors.New("erro interno")
	ErrListar        = errors.New("erro ao listar")
)

// ErroValidacao e um 400 com a mensagem da regra que falhou.
type ErroValidacao struct{ Msg string }

func (e *ErroValidacao) Error() string { return e.Msg }

type Lancamentos struct {
	cfg     RepositorioConfig // CFG#CARTOES: valida a marca pagaFatura e trava a fatura conferida (UsarCartoes)
	faturas FaturasDeFuturos  // as faturas de Futuros
	mu      sync.Mutex
	futuros map[string]*cacheFuturos // a leitura dos previstos de cada espaco, valida enquanto o contador de alteracoes nao mudar
	repo    Repositorio
	ef      Efeitos
	pessoas Pessoas
	rel     Relogio // o instante "agora" da busca por dias no historico
}

func NewLancamentos(repo Repositorio, ef Efeitos, pessoas Pessoas, rel Relogio) *Lancamentos {
	return &Lancamentos{repo: repo, ef: ef, pessoas: pessoas, rel: rel, futuros: map[string]*cacheFuturos{}}
}

const formatoTempo = "2006-01-02T15:04:05.000Z"

// escrita traduz o erro do repository: "dados invalidos" e o conflito de versao passam; o resto vira o erro de
// salvar ou excluir.
func escrita(op string, sentinela, err error) error {
	if errors.Is(err, domain.ErrDadosInvalidos) || errors.Is(err, domain.ErrConflitoVersao) {
		return err
	}
	log.Println(op+":", err)
	return sentinela
}

// Criar grava um lancamento novo. O id, criadoEm e criadoPor sao do servidor.
// marcarConfirmacao poe na versao da acao (a ultima; a versao 1 preguicosa nao e acao de ninguem) que a confirmacao de mes
// fechado por data era necessaria e foi dada.
func marcarConfirmacao(vs []domain.Versao, exigiu bool) {
	if exigiu && len(vs) > 0 {
		vs[len(vs)-1].ConfirmouMesFechado = true
	}
}

// Criar grava um lancamento novo. confirma: a pessoa confirmou mexer em mes fechado por data.
func (s *Lancamentos) Criar(ctx context.Context, sid, sub string, body domain.Lancamento, confirma bool) (domain.Lancamento, error) {
	domain.Protegidos(body)
	if err := domain.ValidarTx(body); err != nil {
		return nil, &ErroValidacao{err.Error()}
	}
	if err := validarStatus(domain.StatusDe(body)); err != nil {
		return nil, err
	}
	if err := s.validarMarca(ctx, sid, body); err != nil {
		return nil, err
	}
	if err := s.travaDeFatura(ctx, sid).Checar(nil, body); err != nil {
		return nil, err
	}
	s.derivar(body)
	agora := time.Now().UTC().Format(formatoTempo)
	body["id"] = domain.NovoID()
	body["criadoEm"] = agora
	body["criadoPor"] = sub
	al, temAlvo := alvoDoEfeito(body)
	exigiu := false
	if temAlvo {
		ex, err := s.ef.Travas(ctx, sid).Checar(al, confirma)
		if err != nil {
			return nil, err
		}
		exigiu = ex
	}
	// Criar nao grava versao (decisao 8 do plan): o lancamento ja guarda criadoPor e criadoEm, e a versao 1 nasce na primeira
	// alteracao. Unica excecao: a criacao que exigiu confirmacao de mes fechado, para a confirmacao ficar gravada no historico.
	var versoes []domain.Versao
	if exigiu {
		versoes = domain.VersoesDaEscrita(0, nil, body, domain.AcaoInsert, sub, agora)
		marcarConfirmacao(versoes, exigiu)
	}
	if err := s.repo.Salvar(ctx, sid, domain.Mudanca{Depois: body, Versoes: versoes}); err != nil {
		return nil, escrita("post tx", ErrSalvar, err)
	}
	if temAlvo {
		s.ef.InvalidarSaldos(ctx, sid, []domain.Alvo{al})
	}
	s.ef.GarantirTags(ctx, sid, domain.TagsDe(body))
	s.ef.SincronizarAviso(ctx, sid, body)
	return s.ef.ComSeq(ctx, sid, comPrevisto(domain.Publico(body))), nil
}

// Editar aplica o patch ao lancamento que esta sob a data "de". restrito: so quem criou pode editar (member).
func (s *Lancamentos) Editar(ctx context.Context, sid, id, de string, patch domain.Lancamento, sub string, restrito, confirma bool) (domain.Lancamento, error) {
	de, err := domain.NormData(de)
	if err != nil {
		return nil, &ErroValidacao{"parametro de (dataEvento atual) obrigatorio"}
	}
	antigo, err := s.repo.Buscar(ctx, sid, de, id)
	if err != nil {
		return nil, ErrInterno
	}
	if antigo == nil || domain.Excluido(antigo) {
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
	if domain.StatusDe(novo) != domain.StatusDe(antes) { // sem mudar o status, o gravado vale (um antigo continua editavel)
		if err := validarStatus(domain.StatusDe(novo)); err != nil {
			return nil, err
		}
	}
	if _, mexeu := patch["pagaFatura"]; mexeu {
		if err := s.validarMarca(ctx, sid, novo); err != nil {
			return nil, err
		}
	}
	if err := s.travaDeFatura(ctx, sid).Checar(antes, novo); err != nil {
		return nil, err
	}
	s.derivar(novo)
	var alvos []domain.Alvo
	exigiu := false
	if domain.MudaSaldo(antes, novo) { // so o que muda o efeito no saldo e travado: descricao, nota, tags e trocar entre status que contam, nao
		alvos = domain.AlvosDoSaldo(antes, novo)
		tr := s.ef.Travas(ctx, sid)
		for _, al := range alvos {
			ex, err := tr.Checar(al, confirma)
			if err != nil {
				return nil, err
			}
			exigiu = exigiu || ex
		}
	}
	ultima, err := s.repo.UltimaVersao(ctx, sid, id)
	if err != nil {
		log.Println("put tx: ultima versao:", err)
		return nil, ErrInterno
	}
	versoes := domain.VersoesDaEscrita(ultima, antes, novo, domain.AcaoUpdate, sub, time.Now().UTC().Format(formatoTempo))
	marcarConfirmacao(versoes, exigiu)
	if err := s.repo.Salvar(ctx, sid, domain.Mudanca{Antes: antes, Depois: novo, DataAnterior: de, Versoes: versoes}); err != nil {
		return nil, escrita("put tx", ErrSalvar, err)
	}
	if len(alvos) > 0 {
		s.ef.InvalidarSaldos(ctx, sid, alvos)
	}
	s.ef.GarantirTags(ctx, sid, domain.TagsDe(novo))
	s.ef.SincronizarAviso(ctx, sid, novo)
	return s.ef.ComSeq(ctx, sid, comPrevisto(domain.Publico(novo))), nil
}

// Excluir exclui o lancamento que esta sob a data "de". A exclusao e logica: o lancamento fica na tabela, marcado
// com excluidoEm, e some das telas; a versao delete guarda como ficou. Excluir o que ja esta excluido, ou o que nao
// existe, responde ok sem gravar nada.
func (s *Lancamentos) Excluir(ctx context.Context, sid, id, de, sub string, confirma bool) (domain.Doc, error) {
	de, err := domain.NormData(de)
	if err != nil {
		return nil, &ErroValidacao{"parametro de (dataEvento atual) obrigatorio"}
	}
	antigo, err := s.repo.Buscar(ctx, sid, de, id)
	if err != nil {
		return nil, ErrInterno
	}
	if antigo == nil || domain.Excluido(antigo) {
		return domain.Doc{"ok": true}, nil
	}
	if err := s.travaDeFatura(ctx, sid).Checar(antigo, nil); err != nil {
		return nil, err
	}
	al, temAlvo := alvoDoEfeito(antigo)
	exigiu := false
	if temAlvo {
		ex, err := s.ef.Travas(ctx, sid).Checar(al, confirma)
		if err != nil {
			return nil, err
		}
		exigiu = ex
	}
	ultima, err := s.repo.UltimaVersao(ctx, sid, id)
	if err != nil {
		log.Println("del tx: ultima versao:", err)
		return nil, ErrInterno
	}
	agora := time.Now().UTC().Format(formatoTempo)
	depois := domain.CopiaRasa(antigo)
	depois["excluidoEm"] = agora
	versoes := domain.VersoesDaEscrita(ultima, antigo, depois, domain.AcaoDelete, sub, agora)
	marcarConfirmacao(versoes, exigiu)
	if err := s.repo.Salvar(ctx, sid, domain.Mudanca{Antes: antigo, Depois: depois, DataAnterior: de, Versoes: versoes}); err != nil {
		return nil, escrita("del tx", ErrExcluir, err)
	}
	s.ef.RemoverAviso(ctx, sid, id)
	if temAlvo {
		s.ef.InvalidarSaldos(ctx, sid, []domain.Alvo{al})
	}
	// desfazer: a versao de antes da exclusao (a que o Desfazer restaura); a exclusao e sempre a ultima versao gravada
	resposta := domain.Doc{"ok": true, "desfazer": domain.Doc{"versao": versoes[len(versoes)-1].Numero - 1}}
	s.ef.ComSeq(ctx, sid, resposta)
	return resposta, nil
}

// Importar grava ate 25 lancamentos por chamada, numa unica transacao: ou entram todos os aceitos, ou nenhum (e todos
// voltam em "pendentes" para reenviar). Preserva id e criadoEm quando vierem no item. Um id que ja existe e uma edicao
// (a versao guarda o estado anterior); um id novo e uma criacao.
func (s *Lancamentos) Importar(ctx context.Context, sid, sub string, body domain.Doc, confirma bool) (domain.Doc, error) {
	bruto, _ := body["itens"].([]any)
	if len(bruto) == 0 || len(bruto) > 25 {
		return nil, &ErroValidacao{"itens deve ter de 1 a 25 lancamentos"}
	}
	agora := time.Now().UTC().Format(formatoTempo)
	rejeitados := []domain.Doc{}
	vistas := map[string]bool{}
	ultimas := map[string]int{} // maior versao por id, contando o que este lote ja vai gravar
	tr := s.ef.Travas(ctx, sid)
	trF := s.travaDeFatura(ctx, sid)
	var alvos []domain.Alvo
	var aceitos []domain.Lancamento
	var mudancas []domain.Mudanca
	for i, x := range bruto {
		d, ok := x.(map[string]any)
		if !ok {
			rejeitados = append(rejeitados, domain.Doc{"indice": i, "motivo": "item invalido"})
			continue
		}
		delete(d, "excluidoEm") // a exclusao so se faz pela rota de exclusao
		delete(d, "confirmaMesFechado")  // a confirmacao so vale pela query, e nunca vai para o lancamento
		delete(d, "confirmouMesFechado") // a marca e so da versao do historico
		marcado, _ := d["previsto"].(bool) // o backup restaura o marcador fiel; nunca o desliga (R8)
		delete(d, "previsto")
		if err := domain.ValidarTx(d); err != nil {
			rejeitados = append(rejeitados, domain.Doc{"indice": i, "motivo": err.Error()})
			continue
		}
		if e, ok := domain.EntradaDe(domain.StatusDe(d)); !ok || e.Nome != domain.StatusDe(d) { // o backup aceita qualquer status do catalogo, inclusive os antigos
			rejeitados = append(rejeitados, domain.Doc{"indice": i, "motivo": "status invalido"})
			continue
		}
		if err := s.validarMarca(ctx, sid, d); err != nil {
			var ev *ErroValidacao
			if errors.As(err, &ev) {
				rejeitados = append(rejeitados, domain.Doc{"indice": i, "motivo": ev.Msg})
				continue
			}
			return nil, err
		}
		s.derivar(d)
		if marcado {
			d["previsto"] = true
		}
		if id, _ := d["id"].(string); !domain.ReID.MatchString(id) {
			d["id"] = domain.NovoID()
		}
		if c, _ := d["criadoEm"].(string); c == "" {
			d["criadoEm"] = agora
		}
		d["criadoPor"] = sub
		if vistas[domain.Identidade(d)] {
			rejeitados = append(rejeitados, domain.Doc{"indice": i, "motivo": "duplicado no mesmo lote"})
			continue
		}
		vistas[domain.Identidade(d)] = true
		al, temAlvo := alvoDoEfeito(d)
		exigiu := false
		if temAlvo {
			ex, err := tr.Checar(al, confirma)
			if err != nil {
				var conferido *domain.ErroMesConferido
				var fechado *domain.ErroMesFechado
				switch {
				case errors.As(err, &conferido):
					rejeitados = append(rejeitados, domain.Doc{"indice": i, "codigo": "mes_conferido", "motivo": "mes conferido e travado (" + conferido.Banco + ", " + conferido.Mes + ")"})
				case errors.As(err, &fechado):
					rejeitados = append(rejeitados, domain.Doc{"indice": i, "codigo": "mes_fechado", "motivo": "mes fechado (" + fechado.Banco + ", " + fechado.Mes + "): confirme para importar"})
				default:
					log.Println("trava:", err)
					return nil, ErrInterno
				}
				continue
			}
			exigiu = ex
		}
		id, _ := d["id"].(string)
		data, _ := d["dataEvento"].(string)
		antigo, err := s.repo.Buscar(ctx, sid, data, id)
		if err != nil {
			log.Println("importar: buscar:", err)
			return nil, ErrInterno
		}
		if err := trF.Checar(antigo, d); err != nil {
			var fc *domain.ErroFaturaConferida
			if errors.As(err, &fc) {
				rejeitados = append(rejeitados, domain.Doc{"indice": i, "codigo": "fatura_conferida", "motivo": err.Error()})
				continue
			}
			return nil, err
		}
		// id novo nao grava versao (decisao 8 do plan), a menos que a importacao tenha exigido confirmacao de mes fechado.
		// id que ja existe e sobrescrita: grava versao (e, se ainda nao tinha, a v1 com o estado de antes).
		var versoes []domain.Versao
		if antigo != nil || exigiu {
			ultima, visto := ultimas[id]
			if !visto {
				if ultima, err = s.repo.UltimaVersao(ctx, sid, id); err != nil {
					log.Println("importar: ultima versao:", err)
					return nil, ErrInterno
				}
			}
			acao := domain.AcaoInsert
			if antigo != nil {
				acao = domain.AcaoUpdate
			}
			versoes = domain.VersoesDaEscrita(ultima, antigo, d, acao, sub, agora)
			marcarConfirmacao(versoes, exigiu)
			ultimas[id] = versoes[len(versoes)-1].Numero
		}
		mudancas = append(mudancas, domain.Mudanca{Antes: antigo, Depois: d, DataAnterior: data, Versoes: versoes})
		aceitos = append(aceitos, d)
		if temAlvo {
			alvos = append(alvos, al)
		}
	}
	gravou := false
	if len(mudancas) > 0 {
		if err := s.repo.SalvarLote(ctx, sid, mudancas); err != nil {
			log.Println("importar: salvar lote:", err)
		} else {
			gravou = true
			s.ef.InvalidarSaldos(ctx, sid, alvos)
		}
	}
	pendentes := []domain.Doc{}
	gravados := 0
	if gravou {
		gravados = len(aceitos)
		for _, d := range aceitos {
			s.ef.SincronizarAviso(ctx, sid, d)
		}
	} else {
		for _, d := range aceitos {
			pendentes = append(pendentes, domain.Publico(d))
		}
	}
	resposta := domain.Doc{}
	if gravou {
		s.ef.ComSeq(ctx, sid, resposta) // algo foi gravado: quem mais estiver com a tela aberta precisa saber
	}
	resposta["gravados"], resposta["rejeitados"], resposta["pendentes"] = gravados, rejeitados, pendentes
	return resposta, nil
}

// Listar: GET /spaces/{sid}/tx. Nunca devolve nil nos itens; qualquer falha de leitura vira ErrListar.
func (s *Lancamentos) Listar(ctx context.Context, sid string, f domain.FiltroTx) ([]domain.Doc, string, error) {
	itens, proximo, err := s.repo.Listar(ctx, sid, f)
	if err != nil {
		log.Println("listar:", err)
		return nil, "", ErrListar
	}
	if itens == nil {
		itens = []domain.Doc{}
	}
	return itens, proximo, nil
}
