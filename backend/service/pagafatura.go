package service

import (
	"context"
	"log"
	"sort"

	"financas/domain"
)

// Pagar a fatura do cartao (itens 33b e 35, .ai/specs/pagar-fatura): a marca pagaFatura, a trava da fatura conferida e as faturas em Futuros.

// FaturasDeFuturos e o que o service de lancamentos precisa do de cartoes para Futuros: as faturas a pagar (R15 do spec).
type FaturasDeFuturos interface {
	AFuturos(ctx context.Context, sid string, seq float64, seqOk bool) ([]domain.Doc, error)
}

// UsarCartoes liga o service de lancamentos a configuracao dos cartoes (para validar a marca e travar a fatura conferida) e as faturas de Futuros.
// E chamado uma vez, na montagem do servidor (os cartoes dependem do service de lancamentos para registrar a diferenca).
func (s *Lancamentos) UsarCartoes(cfg RepositorioConfig, faturas FaturasDeFuturos) {
	s.cfg, s.faturas = cfg, faturas
}

// travaFatura consulta, ao longo de uma chamada, se a escrita muda uma fatura conferida. Le CFG#CARTOES no maximo uma vez e so quando um dos lancamentos e de credito.
type travaFatura struct {
	s    *Lancamentos
	ctx  context.Context
	sid  string
	lido bool
	doc  domain.Doc
}

func (s *Lancamentos) travaDeFatura(ctx context.Context, sid string) *travaFatura {
	return &travaFatura{s: s, ctx: ctx, sid: sid}
}

func (t *travaFatura) cartoes() (domain.Doc, error) {
	if t.lido {
		return t.doc, nil
	}
	t.lido = true
	if t.s.cfg == nil {
		return nil, nil
	}
	d, err := t.s.cfg.Ler(t.ctx, t.sid, "cartoes")
	if err != nil {
		log.Println("trava de fatura: cartoes:", err)
		return nil, ErrInterno
	}
	t.doc = d
	return d, nil
}

// Checar recusa a escrita de antes para depois (qualquer um pode ser nil) quando ela muda o que uma fatura conferida soma (R17 do spec).
func (t *travaFatura) Checar(antes, depois domain.Lancamento) error {
	if !domain.EhCredito(domain.StatusDe(antes)) && !domain.EhCredito(domain.StatusDe(depois)) {
		return nil
	}
	cart, err := t.cartoes()
	if err != nil || cart == nil {
		return err
	}
	if !domain.MudaFatura(antes, depois, cart) {
		return nil
	}
	for _, e := range domain.FaturasAfetadas(antes, depois, cart) {
		if c, ok := domain.CartaoDe(cart, e.Meio); ok && domain.FaturaConferida(c, e.Ym) {
			return &domain.ErroFaturaConferida{Meio: e.Meio, Ym: e.Ym}
		}
	}
	return nil
}

// validarMarca confere a marca pagaFatura do lancamento, se ele a traz (R8 do spec). Le CFG#CARTOES so quando ha marca.
func (s *Lancamentos) validarMarca(ctx context.Context, sid string, d domain.Lancamento) error {
	if _, tem := d["pagaFatura"]; !tem {
		return nil
	}
	var cart domain.Doc
	if s.cfg != nil {
		c, err := s.cfg.Ler(ctx, sid, "cartoes")
		if err != nil {
			log.Println("marca: cartoes:", err)
			return ErrInterno
		}
		cart = c
	}
	if err := domain.ValidarPagaFatura(d, cart); err != nil {
		return &ErroValidacao{err.Error()}
	}
	return nil
}

// ---------- as faturas em Futuros ----------

type cacheFaturas struct {
	seq     float64
	versao  float64
	dia     string
	itens   []domain.Doc
}

// AFuturos devolve as faturas que aparecem em Futuros (R15): dos cartoes com vencimento, as ultimas tres (a aberta e as duas fechadas antes dela) que ainda nao tem
// lancamento marcado, com vencimento do primeiro dia do mes passado em diante. O resultado fica em memoria enquanto o contador de alteracoes, a configuracao dos
// cartoes e o dia nao mudarem.
func (s *Cartoes) AFuturos(ctx context.Context, sid string, seq float64, seqOk bool) ([]domain.Doc, error) {
	cfgC, err := s.cfg.Ler(ctx, sid, "cartoes")
	if err != nil {
		log.Println("faturas de futuros: cartoes:", err)
		return nil, ErrInterno
	}
	versao, _ := cfgC["version"].(float64)
	agora := s.rel.Agora()
	hoje := s.hoje()
	if seqOk && seq > 0 {
		s.mu.Lock()
		c := s.faturas[sid]
		s.mu.Unlock()
		if c != nil && c.seq == seq && c.versao == versao && c.dia == hoje {
			return c.itens, nil
		}
	}
	por, _ := cfgC["porBanco"].(map[string]any)
	meios := make([]string, 0, len(por))
	for nome := range por {
		meios = append(meios, nome)
	}
	sort.Strings(meios)
	inicioJanela := inicioDeFuturos(agora)
	itens := []domain.Doc{}
	var marcados []domain.Doc
	leuMarcados := false
	for _, meio := range meios {
		c, ok := domain.CartaoDe(cfgC, meio)
		if !ok {
			continue
		}
		atual := domain.FaturaAtual(c, hoje)
		// as candidatas com vencimento e dentro da janela de Futuros
		var cands []string
		for _, ym := range []string{domain.SomaMes(atual, -2), domain.SomaMes(atual, -1), atual} {
			if domain.FaturaHistorica(c, ym) {
				continue
			}
			if v := domain.VencimentoDe(c, ym); v != "" && v >= inicioJanela {
				cands = append(cands, ym)
			}
		}
		if len(cands) == 0 {
			continue
		}
		if !leuMarcados {
			ini, _ := domain.PeriodoFatura(c, domain.SomaMes(atual, -2))
			marcados, err = s.marcados(ctx, sid, instanteDoDia(ini, -1), instanteDoDia(domain.SomaDias(hoje, 120), 2))
			if err != nil {
				return nil, err
			}
			leuMarcados = true
		}
		rs, err := s.registros(ctx, sid, c, meio, domain.SomaMes(atual, -1), nil)
		if err != nil {
			return nil, err
		}
		for _, ym := range cands {
			total := domain.TotalFatura(rs, ym)
			vb := c.Faturas[ym].ValorBanco
			if total == 0 && vb == nil {
				continue
			}
			if temMarca(marcados, meio, ym) {
				continue
			}
			estado, valor := "aberta", total
			if ym < atual {
				estado = "fechada"
				if vb != nil {
					valor = *vb
				}
			}
			if !(valor > 0) {
				continue
			}
			venc := domain.VencimentoDe(c, ym)
			pagador := domain.PagadorDe(c, ym)
			item := domain.Doc{
				"tipo": "fatura", "meio": meio, "ym": ym, "rotulo": domain.RotuloMes(ym), "estado": estado, "valor": valor, "total": total,
				"valorBanco": nil, "vencimento": venc, "atrasada": venc < hoje, "pagador": nil,
				"sugestao": domain.Doc{"descricao": "Pagamento da fatura " + domain.RotuloMes(ym) + " " + meio, "valor": valor, "pagador": nil},
			}
			if vb != nil {
				item["valorBanco"] = *vb
			}
			if pagador != "" {
				item["pagador"] = pagador
			}
			item["aviso"] = nil
			if dias, insistir, tem := domain.LerAviso(domain.Doc{"aviso": c.Faturas[ym].Aviso}); tem {
				lista := make([]any, len(dias))
				for i, n := range dias {
					lista[i] = float64(n)
				}
				item["aviso"] = domain.Doc{"dias": lista, "insistir": insistir}
			}
			if sug := domain.PagadorSugerido(c, ym); sug != "" {
				item["sugestao"].(domain.Doc)["pagador"] = sug
			}
			itens = append(itens, item)
		}
	}
	sort.SliceStable(itens, func(i, j int) bool { return itens[i]["vencimento"].(string) < itens[j]["vencimento"].(string) })
	if seqOk && seq > 0 {
		s.mu.Lock()
		s.faturas[sid] = &cacheFaturas{seq: seq, versao: versao, dia: hoje, itens: itens}
		s.mu.Unlock()
	}
	return itens, nil
}

// marcados le os lancamentos de todos os meios entre dois instantes e devolve os que levam a marca pagaFatura e nao estao excluidos.
func (s *Cartoes) marcados(ctx context.Context, sid, de, ate string) ([]domain.Doc, error) {
	todos, err := s.compras(ctx, sid, "", de, ate)
	if err != nil {
		return nil, err
	}
	var out []domain.Doc
	for _, l := range todos {
		if domain.TemPagaFatura(l) && !domain.Excluido(l) {
			out = append(out, l)
		}
	}
	return out, nil
}

func temMarca(marcados []domain.Doc, meio, ym string) bool {
	for _, l := range marcados {
		if m, y, ok := domain.MarcaDe(l); ok && m == meio && y == ym {
			return true
		}
	}
	return false
}

// pagamentosDa devolve, por data, os lancamentos marcados com a fatura (R12): do inicio do periodo dela a 120 dias depois do fechamento.
func (s *Cartoes) pagamentosDa(ctx context.Context, sid string, c domain.Cartao, meio, ym string) ([]domain.Doc, float64, error) {
	ini, fim := domain.PeriodoFatura(c, ym)
	marcados, err := s.marcados(ctx, sid, instanteDoDia(ini, -1), instanteDoDia(domain.SomaDias(fim, 120), 2))
	if err != nil {
		return nil, 0, err
	}
	var sel []domain.Doc
	for _, l := range marcados {
		if m, y, ok := domain.MarcaDe(l); ok && m == meio && y == ym {
			sel = append(sel, l)
		}
	}
	sort.SliceStable(sel, func(i, j int) bool {
		a, _ := sel[i]["dataEvento"].(string)
		b, _ := sel[j]["dataEvento"].(string)
		return a < b
	})
	out := make([]domain.Doc, 0, len(sel))
	soma := 0.0
	for _, l := range sel {
		data, _ := l["dataEvento"].(string)
		dia, _ := domain.DiaBR(data)
		valor, _ := l["valor"].(float64)
		banco, _ := l["banco"].(string)
		out = append(out, domain.Doc{"id": l["id"], "dia": dia, "meio": banco, "valor": valor, "status": domain.StatusDe(l)})
		soma += valor
	}
	return out, domain.Arred(soma), nil
}

// FaturaAviso e o que a rotina de avisos precisa saber de uma fatura no dia de avisar.
type FaturaAviso struct {
	Existe   bool // a fatura ainda tem aviso e vencimento
	Venc     string
	Dias     []int
	Insistir bool
	Valor    float64
	Paga     bool // ha lancamento marcado com a fatura
}

// FaturaParaAviso le a fatura so quando a rotina decidiu avisar (item 46): a configuracao do cartao e, se ainda vale, o valor do dia e se ja foi paga.
func (s *Cartoes) FaturaParaAviso(ctx context.Context, sid, meio, ym string) (FaturaAviso, error) {
	cfgC, err := s.cfg.Ler(ctx, sid, "cartoes")
	if err != nil {
		return FaturaAviso{}, err
	}
	c, ok := domain.CartaoDe(cfgC, meio)
	if !ok {
		return FaturaAviso{}, nil
	}
	fc := c.Faturas[ym]
	dias, insistir, tem := domain.LerAviso(domain.Doc{"aviso": fc.Aviso})
	if !tem || fc.Vencimento == "" {
		return FaturaAviso{}, nil
	}
	out := FaturaAviso{Existe: true, Venc: fc.Vencimento, Dias: dias, Insistir: insistir}
	ini, fim := domain.PeriodoFatura(c, ym)
	marcados, err := s.marcados(ctx, sid, instanteDoDia(ini, -1), instanteDoDia(domain.SomaDias(fim, 120), 2))
	if err != nil {
		return FaturaAviso{}, err
	}
	if out.Paga = temMarca(marcados, meio, ym); out.Paga {
		return out, nil
	}
	rs, err := s.registros(ctx, sid, c, meio, ym, nil)
	if err != nil {
		return FaturaAviso{}, err
	}
	out.Valor = domain.TotalFatura(rs, ym)
	if fc.ValorBanco != nil && ym < domain.FaturaAtual(c, s.hoje()) {
		out.Valor = *fc.ValorBanco
	}
	return out, nil
}
