package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
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
	// um lancamento com a marca pagaFatura (pagamento de fatura) tambem sai do total: o gasto ja contou nas compras de credito (R10 do pagar-fatura)
	l["excluirDoTotal"] = domain.ForaDoTotalStatus(domain.StatusDe(l)) || par != "" || domain.TemPagaFatura(l)
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
	if err := s.travaDeFatura(ctx, sid).Checar(antes, novo); err != nil {
		return nil, err
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
	// as faturas a pagar (itens 33b e 35): entram na lista propria, nos meses (pelo vencimento) e nos atrasados, e somam no resumo
	faturas := []domain.Doc{}
	if s.faturas != nil {
		seq, okSeq := s.ef.Seq(ctx, sid)
		fs, err := s.faturas.AFuturos(ctx, sid, seq, okSeq)
		if err != nil {
			return nil, err
		}
		for _, f := range fs {
			faturas = append(faturas, f)
			valor, _ := f["valor"].(float64)
			venc, _ := f["vencimento"].(string)
			pagar += valor
			if venc < hoje {
				nAtr++
				valorAtr += valor
				atrasados = append(atrasados, f)
			} else {
				porMes[venc[:7]] = append(porMes[venc[:7]], f)
			}
		}
		sort.SliceStable(atrasados, func(i, j int) bool { return diaDoItem(atrasados[i]) < diaDoItem(atrasados[j]) })
		for _, g := range porMes {
			g := g
			sort.SliceStable(g, func(i, j int) bool { return diaDoItem(g[i]) < diaDoItem(g[j]) })
		}
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
		"atrasados": agruparAtrasados(atrasados), "anosAnteriores": anosAnteriores(agora), "meses": meses, "faturas": faturas, "truncado": truncado,
	}, nil
}

// primeiroAnoDeFuturos e o primeiro ano que o seletor de atrasados antigos oferece (a base do dono comeca em 2016).
const primeiroAnoDeFuturos = 2016

// anosAnteriores lista, do mais novo para o mais velho, os anos que ainda tem dias fora da janela de Futuros e podem ser buscados com ?ano= (item 45).
func anosAnteriores(agora time.Time) []int {
	corte := inicioDeFuturos(agora)
	topo, _ := strconv.Atoi(corte[:4])
	if corte[5:] == "01-01" { // a janela comeca em janeiro: o ano inteiro dela ja vem em GET /futuros
		topo--
	}
	anos := []int{}
	for a := topo; a >= primeiroAnoDeFuturos; a-- {
		anos = append(anos, a)
	}
	return anos
}

// agruparAtrasados organiza os atrasados (ja do mais antigo para o mais novo) em anos e, dentro de cada ano, em meses, com a quantidade e a soma de cada ano
// (item 45). O front so exibe e recolhe; nao agrupa nem soma.
func agruparAtrasados(itens []domain.Doc) []domain.Doc {
	anos := []domain.Doc{}
	for _, it := range itens {
		dia := diaDoItem(it)
		if len(dia) < 7 {
			continue
		}
		ano, mes := dia[:4], dia[:7]
		if len(anos) == 0 || anos[len(anos)-1]["ano"] != ano {
			anos = append(anos, domain.Doc{"ano": ano, "n": 0, "valor": 0.0, "meses": []domain.Doc{}})
		}
		a := anos[len(anos)-1]
		ms := a["meses"].([]domain.Doc)
		if len(ms) == 0 || ms[len(ms)-1]["mes"] != mes {
			ms = append(ms, domain.Doc{"mes": mes, "n": 0, "itens": []domain.Doc{}})
		}
		m := ms[len(ms)-1]
		m["itens"] = append(m["itens"].([]domain.Doc), it)
		if oculto, _ := it["oculto"].(bool); !oculto { // o oculto aparece na lista (a pedido) mas nunca entra na conta
			valor, _ := it["valor"].(float64)
			a["n"] = a["n"].(int) + 1
			a["valor"] = domain.Arred(a["valor"].(float64) + valor)
			m["n"] = m["n"].(int) + 1
		}
		a["meses"] = ms
	}
	return anos
}

// FuturosAno: GET /spaces/{sid}/futuros?ano=AAAA. Os previstos atrasados de UM ano anterior ao que Futuros ja traz (item 45), numa consulta so pelo intervalo
// da data na chave (sem ler a particao inteira e sem cache: so a pedido). Devolve o grupo do ano pronto: {ano, n, valor, meses}. Ano que ja esta na
// janela de Futuros (do mes passado em diante) devolve so o que for anterior a ela, e ano sem atrasados devolve n = 0.
func (s *Lancamentos) FuturosAno(ctx context.Context, sid, ano string, ocultos bool) (domain.Doc, error) {
	a, err := strconv.Atoi(ano)
	if len(ano) != 4 || err != nil || a < 2000 || a > 2100 {
		return nil, &ErroValidacao{"ano invalido"}
	}
	corte := inicioDeFuturos(s.rel.Agora()) // os previstos a partir daqui ja vem em GET /futuros
	// a chave guarda a data em UTC e o dia de Futuros e o de Sao Paulo: pega um dia a mais de cada lado e confere o dia na volta
	f := domain.FiltroTx{SoPrevistos: true, Asc: true, Limite: 500,
		De: fmt.Sprintf("%04d-12-31", a-1), Ate: fmt.Sprintf("%04d-01-02", a+1)}
	var itens []domain.Doc
	for {
		pag, proximo, err := s.repo.Listar(ctx, sid, f)
		if err != nil {
			log.Println("futuros ano:", err)
			return nil, ErrListar
		}
		for _, l := range pag {
			if o, _ := l["oculto"].(bool); o && !ocultos {
				continue
			}
			data, _ := l["dataEvento"].(string)
			dia, _ := domain.DiaBR(data)
			if dia[:4] != ano || dia >= corte {
				continue
			}
			itens = append(itens, decorar(domain.CopiaRasa(l)))
		}
		if proximo == "" {
			break
		}
		f.Cursor = proximo
	}
	g := agruparAtrasados(itens)
	if len(g) == 0 {
		return domain.Doc{"ano": ano, "n": 0, "valor": 0.0, "meses": []domain.Doc{}}, nil
	}
	return g[0], nil
}

// inicioDeFuturos e o primeiro dia (AAAA-MM-DD, Sao Paulo) do mes anterior ao de agora: os previstos mais antigos que isso nao entram em Futuros.
func inicioDeFuturos(agora time.Time) string {
	a := agora.In(domain.FusoApp)
	return time.Date(a.Year(), a.Month()-1, 1, 0, 0, 0, 0, domain.FusoApp).Format("2006-01-02")
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
	// A chave de ordenacao tem a data (TX#<dataEvento>#<id>), entao o Query le so de um dia em diante: Futuros olha os atrasados do mes passado
	// em diante e ignora os mais antigos (decisao do dono, 2026-10-25), em vez de ler a particao inteira.
	f := domain.FiltroTx{SoPrevistos: true, Asc: true, Limite: 500, De: inicioDeFuturos(s.rel.Agora())}
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

// diaDoItem e o dia (AAAA-MM-DD) de um item de Futuros: o vencimento de uma fatura ou o dia da data de um previsto.
func diaDoItem(d domain.Doc) string {
	if v, ok := d["vencimento"].(string); ok && d["tipo"] == "fatura" {
		return v
	}
	data, _ := d["dataEvento"].(string)
	dia, _ := domain.DiaBR(data)
	return dia
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
