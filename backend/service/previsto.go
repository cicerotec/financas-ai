package service

import (
	"context"
	"errors"
	"log"
	"sort"
	"time"

	"financas/domain"
)

// Previsto, catalogo de status e Futuros (item 41, .ai/specs/futuros-previsto). O servidor decide o marcador `previsto`, o excluirDoTotal e o que
// e um status valido; o cliente so manda o que a pessoa escolheu.

// ErrNaoPrevisto: confirmar um lancamento que nao e previsto (409 nao_previsto).
var ErrNaoPrevisto = errors.New("o lancamento nao e um previsto")

// validarStatus confere o status de um lancamento novo (ou de um status trocado): precisa estar no catalogo, com o texto exato, e ser selecionavel (R12).
func validarStatus(status string) error {
	e, ok := domain.EntradaDe(status)
	if !ok || e.Nome != status {
		return &ErroValidacao{"status invalido"}
	}
	if !e.Selecionavel {
		return &ErroValidacao{"status descontinuado"}
	}
	return nil
}

// derivar acerta no lancamento o que so o servidor decide: liga o marcador previsto quando a data e futura (R5, R6) e calcula o excluirDoTotal
// pelo catalogo e pela transferencia (R13). O previsto que ja estava ligado fica ligado; so o confirmar o desliga.
func (s *Lancamentos) derivar(l domain.Lancamento) {
	data, _ := l["dataEvento"].(string)
	if !domain.EhPrevisto(l) && domain.PrevistoPorData(data, s.rel.Agora()) { // o antigo (so pelo nome) ou o ja marcado nao precisam de marcador novo
		l["previsto"] = true
	}
	par, _ := l["transferParId"].(string)
	l["excluirDoTotal"] = domain.ForaDoTotalStatus(domain.StatusDe(l)) || par != ""
}

// comPrevisto acrescenta previsto: true na resposta quando o lancamento e previsto, tambem o antigo (derivado, nao gravado), para o front nao olhar o status.
func comPrevisto(d domain.Doc) domain.Doc {
	if domain.EhPrevisto(d) {
		d["previsto"] = true
	}
	return d
}

// alvoDoEfeito devolve o alvo (meio e mes) do efeito do lancamento no saldo; ok=false quando ele nao mexe no saldo (R15).
func alvoDoEfeito(l domain.Lancamento) (domain.Alvo, bool) {
	e := domain.EfeitoSaldo(l)
	return e.Alvo, e.Existe
}

// Confirmar: POST /spaces/{sid}/tx/{id}/confirmar?de=<dataEvento atual> com { valor, dataEvento, status } (R18 a R21). E a unica rota que
// tira o marcador previsto. restrito: so quem criou pode confirmar (member).
func (s *Lancamentos) Confirmar(ctx context.Context, sid, id, de string, body domain.Doc, sub string, restrito, confirma bool) (domain.Doc, error) {
	de, err := domain.NormData(de)
	if err != nil {
		return nil, &ErroValidacao{"parametro de (dataEvento atual) obrigatorio"}
	}
	valor, ok := body["valor"].(float64)
	if !ok || !(valor > 0) {
		return nil, &ErroValidacao{"o valor da confirmacao deve ser maior que zero"}
	}
	dataTxt, _ := body["dataEvento"].(string)
	data, err := domain.NormData(dataTxt)
	if err != nil {
		return nil, &ErroValidacao{"dataEvento da confirmacao invalida"}
	}
	if domain.PrevistoPorData(data, s.rel.Agora()) {
		return nil, &ErroValidacao{"a data da confirmacao nao pode ser depois de hoje"}
	}
	status, _ := body["status"].(string)
	if err := validarStatus(status); err != nil {
		return nil, err
	}
	antigo, err := s.repo.Buscar(ctx, sid, de, id)
	if err != nil {
		return nil, ErrInterno
	}
	if antigo == nil || domain.Excluido(antigo) {
		return nil, ErrNaoEncontrado
	}
	if restrito && !domain.EhDono(antigo, sub) {
		log.Printf("negado: sub=%s tentou confirmar lancamento de outra pessoa (%s)", sub, id)
		return nil, ErrSoDono
	}
	if !domain.EhPrevisto(antigo) {
		return nil, ErrNaoPrevisto
	}
	antes := domain.CopiaRasa(antigo)
	novo := antigo
	novo["valor"], novo["dataEvento"], novo["status"] = valor, data, status
	delete(novo, "previsto")
	s.derivar(novo)
	if err := domain.ValidarTx(novo); err != nil {
		return nil, &ErroValidacao{err.Error()}
	}
	exigiu := false
	var alvos []domain.Alvo
	if domain.MudaSaldo(antes, novo) {
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
		log.Println("confirmar: ultima versao:", err)
		return nil, ErrInterno
	}
	versoes := domain.VersoesDaEscrita(ultima, antes, novo, domain.AcaoUpdate, sub, time.Now().UTC().Format(formatoTempo))
	marcarConfirmacao(versoes, exigiu)
	if err := s.repo.Salvar(ctx, sid, domain.Mudanca{Antes: antes, Depois: novo, DataAnterior: de, Versoes: versoes}); err != nil {
		return nil, escrita("confirmar", ErrSalvar, err)
	}
	if len(alvos) > 0 {
		s.ef.InvalidarSaldos(ctx, sid, alvos)
	}
	s.ef.GarantirTags(ctx, sid, domain.TagsDe(novo))
	s.ef.SincronizarAviso(ctx, sid, novo)
	resposta := comPrevisto(domain.Publico(novo))
	resposta["desfazer"] = domain.Doc{"versao": versoes[len(versoes)-1].Numero - 1}
	return s.ef.ComSeq(ctx, sid, resposta), nil
}

// ---------- GET /futuros ----------

const limiteFuturos = 1000

// Futuros: GET /spaces/{sid}/futuros[?ocultos=1]. Todos os previstos do espaco, com o resumo, os atrasados e um grupo por mes (R22 a R24). Os ocultos
// ficam de fora, e nunca entram no resumo.
func (s *Lancamentos) Futuros(ctx context.Context, sid string, ocultos bool) (domain.Doc, error) {
	agora := s.rel.Agora()
	todos, err := s.lerPrevistos(ctx, sid)
	if err != nil {
		return nil, err
	}
	var itens []domain.Doc
	truncado := false
	for _, l := range todos {
		if o, _ := l["oculto"].(bool); o && !ocultos {
			continue
		}
		if len(itens) >= limiteFuturos {
			truncado = true
			break
		}
		itens = append(itens, domain.CopiaRasa(l)) // copia: decorar mexe no mapa e o original fica no cache
	}
	hoje := agora.In(domain.FusoBR).Format("2006-01-02")
	pagar, receber, valorAtr, nAtr := 0.0, 0.0, 0.0, 0
	atrasados := []domain.Doc{}
	porMes := map[string][]domain.Doc{}
	for _, l := range itens {
		oculto, _ := l["oculto"].(bool)
		data, _ := l["dataEvento"].(string)
		dia, _ := domain.DiaBR(data)
		item := decorar(l)
		valor, _ := l["valor"].(float64)
		if !oculto {
			if t, _ := l["tipo"].(string); t == "entrada" {
				receber += valor
			} else {
				pagar += valor
			}
			if dia < hoje {
				nAtr++
				valorAtr += valor
			}
		}
		if dia < hoje && !oculto {
			atrasados = append(atrasados, item)
			continue
		}
		mes := domain.MesDe(data)
		porMes[mes] = append(porMes[mes], item)
	}
	meses := []domain.Doc{}
	chaves := make([]string, 0, len(porMes))
	for m := range porMes {
		chaves = append(chaves, m)
	}
	sort.Strings(chaves)
	for _, m := range chaves {
		meses = append(meses, domain.Doc{"mes": m, "itens": porMes[m]})
	}
	return domain.Doc{
		"resumo":    domain.Doc{"pagar": domain.Arred(pagar), "receber": domain.Arred(receber), "atrasados": domain.Doc{"n": nAtr, "valor": domain.Arred(valorAtr)}},
		"atrasados": atrasados, "meses": meses, "truncado": truncado,
	}, nil
}

// cacheFuturos guarda o que a ultima leitura achou, com o valor do contador de alteracoes lido ANTES dela.
type cacheFuturos struct {
	seq    float64
	itens  []domain.Doc
	quando time.Time
}

const validadeDoCacheDeFuturos = 5 * time.Minute

// lerPrevistos devolve todos os previstos do espaco (sem os excluidos), do mais antigo para o mais novo. Achar os previstos de qualquer ano
// exige ler a particao inteira, que e cara na tabela de 5 RCU; por isso o resultado fica em memoria enquanto o contador de alteracoes do
// espaco (que toda escrita de lancamento soma) nao mudar, o que custa uma leitura de 1 item. O contador e lido ANTES da leitura grande: uma
// escrita que chegue durante ela muda o contador e a proxima chamada le de novo. Sem contador (0) ou depois de 5 minutos, nao reaproveita.
func (s *Lancamentos) lerPrevistos(ctx context.Context, sid string) ([]domain.Doc, error) {
	seq, ok := s.ef.Seq(ctx, sid)
	if ok && seq > 0 {
		s.mu.Lock()
		c := s.futuros[sid]
		s.mu.Unlock()
		if c != nil && c.seq == seq && time.Since(c.quando) < validadeDoCacheDeFuturos {
			return c.itens, nil
		}
	}
	f := domain.FiltroTx{SoPrevistos: true, Asc: true, Limite: 500}
	var itens []domain.Doc
	for {
		pag, proximo, err := s.repo.Listar(ctx, sid, f)
		if err != nil {
			log.Println("futuros:", err)
			return nil, ErrListar
		}
		itens = append(itens, pag...)
		if proximo == "" {
			break
		}
		f.Cursor = proximo
	}
	if ok && seq > 0 {
		s.mu.Lock()
		s.futuros[sid] = &cacheFuturos{seq: seq, itens: itens, quando: time.Now()}
		s.mu.Unlock()
	}
	return itens, nil
}

// decorar acrescenta ao lancamento o que a listagem mostra pronto: previsto (derivado), rotulo e corPadrao do catalogo (R12a, R17).
func decorar(l domain.Doc) domain.Doc {
	comPrevisto(l)
	l["rotulo"], l["corPadrao"] = domain.RotuloECor(domain.StatusDe(l))
	return l
}

// ---------- GET /tx pronto (R12a, R12b, R16) ----------

// ListarTx: GET /spaces/{sid}/tx. Nao devolve previstos; cada item leva rotulo e corPadrao; com filtros=1, a primeira pagina leva `filtros.status`, os status que
// aparecem no periodo (considerando os outros filtros, mas nao o de status). So a pedido: calcula-los le o periodo inteiro de novo. Nao ha total nem soma por mes (R12c).
func (s *Lancamentos) ListarTx(ctx context.Context, sid string, f domain.FiltroTx, comFiltros bool) (domain.Doc, error) {
	f.SemPrevistos = true
	itens, proximo, err := s.Listar(ctx, sid, f)
	if err != nil {
		return nil, err
	}
	for _, l := range itens {
		decorar(l)
	}
	out := domain.Doc{"itens": itens, "proximo": proximo}
	if comFiltros && f.Cursor == "" {
		filtros, err := s.statusDoPeriodo(ctx, sid, f)
		if err != nil {
			return nil, err
		}
		out["filtros"] = domain.Doc{"status": filtros}
	}
	return out, nil
}

// statusDoPeriodo le o periodo inteiro (sem o filtro de status) e junta os status distintos: na ordem do catalogo, os de fora no fim.
func (s *Lancamentos) statusDoPeriodo(ctx context.Context, sid string, f domain.FiltroTx) ([]string, error) {
	g := f
	g.Status, g.Cursor, g.Limite, g.Asc = "", "", 1000, true
	vistos := map[string]bool{}
	for {
		pag, proximo, err := s.repo.Listar(ctx, sid, g)
		if err != nil {
			log.Println("filtros de status:", err)
			return nil, ErrListar
		}
		for _, l := range pag {
			vistos[domain.StatusDe(l)] = true
		}
		if proximo == "" {
			break
		}
		g.Cursor = proximo
	}
	out := make([]string, 0, len(vistos))
	for st := range vistos {
		if st != "" {
			out = append(out, st)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := domain.OrdemNoCatalogo(out[i]), domain.OrdemNoCatalogo(out[j])
		if a != b {
			return a < b
		}
		return out[i] < out[j]
	})
	return out, nil
}

// StatusSugeridos: GET /spaces/{sid}/status?data=<ISO>[&todos=1] (R10). A lista sugerida para a data, ou todas as selecionaveis.
func (s *Lancamentos) StatusSugeridos(data string, todos bool) domain.Doc {
	out := domain.Doc{"todos": todos}
	if data != "" {
		out["data"] = data
	}
	if todos {
		out["status"] = domain.Selecionaveis()
	} else {
		out["status"] = domain.SugeridosPara(data, s.rel.Agora())
	}
	return out
}
