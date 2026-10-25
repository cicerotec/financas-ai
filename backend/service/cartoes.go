package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"sync"
	"time"

	"financas/domain"
)

// Faturas do cartao calculadas e validadas no servidor (item 33a, .ai/specs/fatura-servidor). O navegador so exibe: o periodo, a fatura de cada
// compra, o ajuste, o total, a conferencia e as sugestoes chegam prontos, e a configuracao do cartao so muda por comandos validados.

var (
	ErrCartaoNaoEncontrado = errors.New("esse meio nao e um cartao")
	ErrFaturaNaoConferida  = errors.New("essa fatura nao esta conferida")
)

// ErroDiferencaFatura: o valor do app do cartao nao bate com o total que o servidor calculou (ou nao foi informado). E um 409 com os numeros, para a
// pessoa decidir entre cancelar e conferir mesmo assim (aceitarDiferenca).
type ErroDiferencaFatura struct {
	Total      float64
	ValorBanco *float64
	Diferenca  *float64
}

func (e *ErroDiferencaFatura) Error() string {
	if e.ValorBanco == nil {
		return "a fatura nao tem o valor do app do cartao para comparar: confirme para conferir com o total calculado"
	}
	return fmt.Sprintf("o total calculado (%.2f) difere do valor informado do banco (%.2f): confirme para conferir mesmo assim", e.Total, *e.ValorBanco)
}

// Cartoes calcula as faturas e aplica os comandos de configuracao dos cartoes.
type Cartoes struct {
	mu      sync.Mutex
	faturas map[string]*cacheFaturas // as faturas de Futuros de cada espaco, validas enquanto o contador e a configuracao nao mudam
	lanc    LeitorDeLancamentos
	cfg     RepositorioConfig
	rel     Relogio
	criador CriadorDeLancamento
}

func NewCartoes(lanc LeitorDeLancamentos, cfg RepositorioConfig, rel Relogio, criador CriadorDeLancamento) *Cartoes {
	return &Cartoes{lanc: lanc, cfg: cfg, rel: rel, criador: criador, faturas: map[string]*cacheFaturas{}}
}

const formatoISO = "2006-01-02T15:04:05.000Z"

// hoje e o dia de hoje em Sao Paulo (AAAA-MM-DD).
func (s *Cartoes) hoje() string { return s.rel.Agora().In(domain.FusoApp).Format("2006-01-02") }

// instanteDoDia e a meia-noite de Sao Paulo de um dia (AAAA-MM-DD), deslocada em n dias, no formato de dataEvento.
func instanteDoDia(dia string, n int) string {
	t, err := time.ParseInLocation("2006-01-02", domain.SomaDias(dia, n), domain.FusoApp)
	if err != nil {
		return ""
	}
	return t.UTC().Format(formatoISO)
}

func (s *Cartoes) ler(ctx context.Context, sid string) (listas, cartoes domain.Doc, err error) {
	if listas, err = s.cfg.Ler(ctx, sid, "listas"); err == nil {
		cartoes, err = s.cfg.Ler(ctx, sid, "cartoes")
	}
	if err != nil {
		log.Println("cartoes: config:", err)
		return nil, nil, ErrInterno
	}
	return listas, cartoes, nil
}

// compras le os lancamentos do periodo [de, ate] (instantes), do banco pedido (ou de todos se banco == ""), da mais recente para a mais antiga,
// em quantas paginas forem precisas: sem limite de 1000 e sem "incompleto".
func (s *Cartoes) compras(ctx context.Context, sid, banco, de, ate string) ([]domain.Doc, error) {
	f := domain.FiltroTx{Banco: banco, De: de, Ate: ate, Asc: false, Limite: 1000}
	var todos []domain.Doc
	for {
		itens, proximo, err := s.lanc.Listar(ctx, sid, f)
		if err != nil {
			log.Println("cartoes: listar:", err)
			return nil, ErrListar
		}
		todos = append(todos, itens...)
		if proximo == "" {
			return todos, nil
		}
		f.Cursor = proximo
	}
}

func eDonoDe(papel string) bool { return papel == "owner" }

// ---------- GET /cartoes ----------

// Listar: GET /spaces/{sid}/cartoes. A fatura atual de cada cartao, o total geral, o proximo fechamento e os meios que ainda nao sao cartao.
// Le uma vez so os lancamentos do espaco dos ultimos 12 meses (a fatura atual e as vizinhas ficam dentro disso).
func (s *Cartoes) Listar(ctx context.Context, sid string) (domain.Doc, error) {
	listas, cfgC, err := s.ler(ctx, sid)
	if err != nil {
		return nil, err
	}
	hoje := s.hoje()
	desde := domain.SomaDias(hoje, -366)
	itens, err := s.compras(ctx, sid, "", instanteDoDia(desde, 0), "")
	if err != nil {
		return nil, err
	}
	total, minDias := 0.0, 0
	temMin := false
	cartoes := []domain.Doc{}
	nomes := nomesDeBancos(listas)
	for _, nome := range nomes {
		c, ok := domain.CartaoDe(cfgC, nome)
		if !ok {
			continue
		}
		rs := domain.RegistrosDoCartao(c, nome, itens)
		atual := domain.FaturaAtual(c, hoje)
		fech := domain.FechamentoDe(c, atual)
		dias := domain.NumeroDoDia(fech) - domain.NumeroDoDia(hoje)
		t := domain.TotalFatura(rs, atual)
		total += t
		if !temMin || dias < minDias {
			minDias, temMin = dias, true
		}
		var pagador, diaVenc any
		if c.Pagador != "" {
			pagador = c.Pagador
		}
		if c.DiaVencimento > 0 {
			diaVenc = c.DiaVencimento
		}
		cartoes = append(cartoes, domain.Doc{
			"pagador": pagador, "diaVencimento": diaVenc,
			"meio":                 nome,
			"fatura":               domain.Doc{"ym": atual, "fechamento": fech, "diasAteFechar": dias, "total": t},
			"conferidasQueMudaram": conferidasQueMudaram(c, rs, atual),
		})
	}
	comCredito := map[string]bool{}
	for _, l := range itens {
		if st, _ := l["status"].(string); st != "" && domain.EhCredito(st) && !domain.Excluido(l) {
			b, _ := l["banco"].(string)
			comCredito[b] = true
		}
	}
	var livres []string
	for _, nome := range nomes {
		if _, ok := domain.CartaoDe(cfgC, nome); !ok {
			livres = append(livres, nome)
		}
	}
	sort.SliceStable(livres, func(i, j int) bool { return comCredito[livres[i]] && !comCredito[livres[j]] })
	disponiveis := []domain.Doc{}
	for _, nome := range livres {
		disponiveis = append(disponiveis, domain.Doc{"meio": nome, "temComprasNoCredito": comCredito[nome]})
	}
	out := domain.Doc{"total": domain.Arred(total), "cartoes": cartoes, "disponiveis": disponiveis}
	if temMin {
		out["proximoFechamentoEmDias"] = minDias
	} else {
		out["proximoFechamentoEmDias"] = nil
	}
	return out, nil
}

// conferidasQueMudaram conta as faturas conferidas, de duas antes da atual ate a seguinte (a janela que a leitura cobre por inteiro), cujo total de
// agora difere do total conferido.
func conferidasQueMudaram(c domain.Cartao, rs []domain.RegistroFatura, atual string) int {
	n := 0
	for ym, f := range c.Faturas {
		if f.Conferida == nil || ym < domain.SomaMes(atual, -2) || ym > domain.SomaMes(atual, 1) {
			continue
		}
		if mudou(f, domain.TotalFatura(rs, ym)) {
			n++
		}
	}
	return n
}

func mudou(f domain.FaturaCfg, total float64) bool {
	conf, ok := f.Conferida["total"].(float64)
	return ok && math.Floor(conf*100+0.5) != math.Floor(total*100+0.5)
}

// ---------- GET /cartoes/{banco}/faturas/{ym} ----------

var reMesFatura = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// Fatura: GET /spaces/{sid}/cartoes/{banco}/faturas/{ym}. ym vazio vale a fatura atual.
func (s *Cartoes) Fatura(ctx context.Context, sid, sub string, dono bool, banco, ym string) (domain.Doc, error) {
	listas, cfgC, err := s.ler(ctx, sid)
	if err != nil {
		return nil, err
	}
	c, ok := domain.CartaoDe(cfgC, banco)
	if !ok {
		return nil, ErrCartaoNaoEncontrado
	}
	hoje := s.hoje()
	atual := domain.FaturaAtual(c, hoje)
	if ym == "" {
		ym = atual
	}
	if !reMesFatura.MatchString(ym) {
		return nil, &ErroValidacao{"fatura invalida (use AAAA-MM)"}
	}
	rs, err := s.registros(ctx, sid, c, banco, ym, listas)
	if err != nil {
		return nil, err
	}
	out := montarFatura(c, banco, ym, atual, hoje, sub, dono, rs)
	out["vencimento"], out["pagador"] = nil, nil
	if v := domain.VencimentoDe(c, ym); v != "" {
		out["vencimento"] = v
	}
	if p := domain.PagadorDe(c, ym); p != "" {
		out["pagador"] = p
	}
	pgs, totalPago, err := s.pagamentosDa(ctx, sid, c, banco, ym)
	if err != nil {
		return nil, err
	}
	out["pagamentos"], out["totalPago"] = pgs, totalPago
	if len(pgs) > 0 {
		ref, _ := out["total"].(float64)
		if vb, ok := out["valorBanco"].(float64); ok {
			ref = vb
		}
		out["diferencaPaga"] = domain.Arred(totalPago - ref)
	} else {
		out["diferencaPaga"] = nil
	}
	return out, nil
}

// registros le os lancamentos de uma janela em torno de ym (do inicio de ym-2 ao fim de ym+1) e monta as compras do cartao.
func (s *Cartoes) registros(ctx context.Context, sid string, c domain.Cartao, banco, ym string, listas domain.Doc) ([]domain.RegistroFatura, error) {
	ini, _ := domain.PeriodoFatura(c, domain.SomaMes(ym, -2))
	_, fim := domain.PeriodoFatura(c, domain.SomaMes(ym, 1))
	itens, err := s.compras(ctx, sid, banco, instanteDoDia(ini, -1), instanteDoDia(fim, 2))
	if err != nil {
		return nil, err
	}
	return domain.RegistrosDoCartao(c, banco, itens), nil
}

func montarFatura(c domain.Cartao, banco, ym, atual, hoje, sub string, dono bool, rs []domain.RegistroFatura) domain.Doc {
	ini, fim := domain.PeriodoFatura(c, ym)
	total := domain.TotalFatura(rs, ym)
	fechada := domain.FaturaFechada(c, ym, hoje)
	historico := domain.FaturaHistorica(c, ym)
	cfgF := c.Faturas[ym]
	off := domain.DiferencaDeMeses(ym, atual)
	chips := []domain.Doc{}
	for _, o := range []int{-1, 0, 1} {
		m := domain.SomaMes(ym, o)
		chips = append(chips, domain.Doc{"ym": m, "rotulo": domain.RotuloFatura(domain.DiferencaDeMeses(m, atual), m)})
	}
	out := domain.Doc{
		"meio": banco, "ym": ym, "rotulo": domain.RotuloFatura(off, ym), "chips": chips,
		"anterior": domain.SomaMes(ym, -1), "proxima": domain.SomaMes(ym, 1),
		"periodo": domain.Doc{"ini": ini, "fim": fim}, "fechada": fechada, "historico": historico,
		"total": total, "incompleto": false, "mudouDesdeConferencia": false,
		"valorBanco": nil, "diferenca": nil, "conferida": nil, "inicio": nil,
	}
	if c.Inicio != "" {
		out["inicio"] = c.Inicio
	}
	var diferenca *float64
	if cfgF.ValorBanco != nil {
		out["valorBanco"] = *cfgF.ValorBanco
		d := domain.Arred(total - *cfgF.ValorBanco)
		diferenca = &d
		out["diferenca"] = d
	}
	if cfgF.Conferida != nil {
		out["conferida"] = cfgF.Conferida
		out["mudouDesdeConferencia"] = mudou(cfgF, total)
	}
	// as compras da fatura: as de base ym ou efetivas em ym, por dia
	lanc := []domain.Doc{}
	var doMes []domain.RegistroFatura
	for _, r := range rs {
		if r.Base == ym || r.Fatura == ym {
			doMes = append(doMes, r)
		}
	}
	sort.SliceStable(doMes, func(i, j int) bool { return doMes[i].Dia < doMes[j].Dia })
	for _, r := range doMes {
		item := domain.Doc{
			"id": r.ID, "descricao": r.Descricao, "dia": r.Dia, "valor": r.Valor, "base": r.Base, "fatura": r.Fatura,
			"ajuste": r.Ajuste, "origem": nil, "estimado": r.Estimado, "podeMexer": dono || (sub != "" && r.CriadoPor == sub),
			"ocupaOutraFatura": nil,
		}
		switch {
		case r.Base == ym && r.Fatura != ym:
			item["origem"], item["ocupaOutraFatura"] = "foi", r.Fatura
		case r.Fatura == ym && r.Base != ym:
			item["origem"], item["ocupaOutraFatura"] = "veio", r.Base
		}
		lanc = append(lanc, item)
	}
	out["lancamentos"] = lanc
	// as sugestoes (R16): so com a fatura fechada, diferenca e fora do historico
	out["sugestoes"], out["semSugestao"], out["podeRegistrarDiferenca"] = []domain.Doc{}, nil, false
	if fechada && !historico && diferenca != nil && *diferenca != 0 {
		sugs := []domain.Doc{}
		for _, sg := range domain.Candidatos(c, rs, ym, *diferenca) {
			regs := []domain.Doc{}
			for _, r := range sg.Registros {
				regs = append(regs, domain.Doc{"id": r.ID, "descricao": r.Descricao, "dia": r.Dia, "valor": r.Valor})
			}
			acoes := []domain.Doc{}
			for _, a := range sg.Acoes {
				acoes = append(acoes, domain.Doc{"id": a.ID, "faturaAjuste": a.Ajuste})
			}
			sugs = append(sugs, domain.Doc{"lancamentos": regs, "acoes": acoes, "rotulo": sg.Rotulo})
		}
		out["sugestoes"] = sugs
		if len(sugs) == 0 {
			if *diferenca < 0 {
				out["semSugestao"] = "Pode ser anuidade, juros, IOF ou uma compra não registrada."
			} else {
				out["semSugestao"] = "Pode ser um valor registrado a mais ou um registro duplicado."
			}
		}
		out["podeRegistrarDiferenca"] = *diferenca < 0
	}
	return out
}

// ---------- comandos (so o dono) ----------

// atualizar le CFG#CARTOES, aplica a mudanca na copia de porBanco e grava com o controle otimista (version); se outra pessoa gravou no meio, le de
// novo e repete. So o campo porBanco muda e a forma do item e a de sempre.
func (s *Cartoes) atualizar(ctx context.Context, sid string, mudar func(porBanco map[string]any) error) error {
	for tentativa := 0; tentativa < 5; tentativa++ {
		atual, err := s.cfg.Ler(ctx, sid, "cartoes")
		if err != nil {
			log.Println("cartoes: ler:", err)
			return ErrInterno
		}
		corpo := domain.Publico(copiaProfunda(atual))
		versao, _ := atual["version"].(float64)
		delete(corpo, "version")
		por, _ := corpo["porBanco"].(map[string]any)
		if por == nil {
			por = map[string]any{}
		}
		if err := mudar(por); err != nil {
			return err
		}
		corpo["porBanco"] = por
		_, err = s.cfg.Salvar(ctx, sid, "cartoes", corpo, versao)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, domain.ErrConflitoConfig):
			continue
		case errors.Is(err, domain.ErrDadosInvalidos):
			return err
		}
		log.Println("cartoes: salvar:", err)
		return ErrSalvar
	}
	return ErrSalvar
}

func copiaProfunda(d domain.Doc) domain.Doc {
	return copiaAny(d).(map[string]any)
}

func copiaAny(v any) any {
	switch x := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, e := range x {
			m[k] = copiaAny(e)
		}
		return m
	case []any:
		a := make([]any, len(x))
		for i, e := range x {
			a[i] = copiaAny(e)
		}
		return a
	}
	return v
}

func cartaoDe(por map[string]any, banco string) (map[string]any, error) {
	c, ok := por[banco].(map[string]any)
	if !ok {
		return nil, ErrCartaoNaoEncontrado
	}
	return c, nil
}

func nomeDoMeio(body domain.Doc) string {
	b, _ := body["meio"].(string)
	if b == "" {
		b, _ = body["banco"].(string)
	}
	return b
}

// Adicionar: POST /spaces/{sid}/cartoes { meio, diaPadrao } (R17).
func (s *Cartoes) Adicionar(ctx context.Context, sid string, body domain.Doc) (domain.Doc, error) {
	banco := nomeDoMeio(body)
	dia, ok := body["diaPadrao"].(float64)
	if banco == "" || !ok || dia != float64(int(dia)) || dia < 1 || dia > 31 {
		return nil, &ErroValidacao{"informe o meio e o dia do fechamento (1 a 31)"}
	}
	listas, _, err := s.ler(ctx, sid)
	if err != nil {
		return nil, err
	}
	na := false
	for _, n := range nomesDeBancos(listas) {
		na = na || n == banco
	}
	if !na {
		return nil, &ErroValidacao{"esse meio nao esta na lista"}
	}
	err = s.atualizar(ctx, sid, func(por map[string]any) error {
		if _, ja := por[banco]; ja {
			return &ErroValidacao{"esse meio ja e um cartao"}
		}
		por[banco] = map[string]any{"diaPadrao": dia, "faturas": map[string]any{}}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return domain.Doc{"meio": banco, "diaPadrao": dia}, nil
}

// Remover: DELETE /spaces/{sid}/cartoes/{banco} (R18). Os lancamentos nao mudam.
func (s *Cartoes) Remover(ctx context.Context, sid, banco string) (domain.Doc, error) {
	err := s.atualizar(ctx, sid, func(por map[string]any) error {
		if _, err := cartaoDe(por, banco); err != nil {
			return err
		}
		delete(por, banco)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return domain.Doc{"meio": banco, "removido": true}, nil
}

// Inicio: PUT /spaces/{sid}/cartoes/{banco} { inicio?, pagador?, diaVencimento? } (R19 do item 33a e R2 do pagar-fatura). null remove o campo.
func (s *Cartoes) Inicio(ctx context.Context, sid, banco string, body domain.Doc) (domain.Doc, error) {
	ini, temIni := body["inicio"]
	pag, temPag := body["pagador"]
	dv, temDV := body["diaVencimento"]
	if !temIni && !temPag && !temDV {
		return nil, &ErroValidacao{"informe inicio (AAAA-MM), pagador e/ou diaVencimento (ou null para limpar)"}
	}
	var iniTxt string
	if temIni && ini != nil {
		t, ok := ini.(string)
		if !ok || !reMesFatura.MatchString(t) {
			return nil, &ErroValidacao{"inicio invalido (use AAAA-MM)"}
		}
		iniTxt = t
	}
	var pagTxt string
	if temPag && pag != nil {
		t, ok := pag.(string)
		if !ok || t == "" {
			return nil, &ErroValidacao{"pagador invalido (o nome de um meio)"}
		}
		if err := s.validarPagador(ctx, sid, banco, t); err != nil {
			return nil, err
		}
		pagTxt = t
	}
	var dia float64
	if temDV && dv != nil {
		n, ok := dv.(float64)
		if !ok || n != float64(int(n)) || n < 1 || n > 31 {
			return nil, &ErroValidacao{"diaVencimento invalido (inteiro de 1 a 31)"}
		}
		dia = n
	}
	err := s.atualizar(ctx, sid, func(por map[string]any) error {
		c, err := cartaoDe(por, banco)
		if err != nil {
			return err
		}
		if temIni {
			if ini == nil {
				delete(c, "inicio")
			} else {
				c["inicio"] = iniTxt
			}
		}
		if temPag {
			if pag == nil {
				delete(c, "pagador")
			} else {
				c["pagador"] = pagTxt
			}
		}
		if temDV {
			if dv == nil {
				delete(c, "diaVencimento")
			} else {
				c["diaVencimento"] = dia
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return domain.Doc{"meio": banco}, nil
}

// validarPagador confere o pagador sugerido (R4): um meio da lista e diferente do proprio cartao.
func (s *Cartoes) validarPagador(ctx context.Context, sid, cartao, pagador string) error {
	if pagador == cartao {
		return &ErroValidacao{"o pagador nao pode ser o proprio cartao"}
	}
	listas, _, err := s.ler(ctx, sid)
	if err != nil {
		return err
	}
	for _, n := range nomesDeBancos(listas) {
		if n == pagador {
			return nil
		}
	}
	return &ErroValidacao{"o pagador precisa ser um meio da lista"}
}

// Fechamento: PUT /spaces/{sid}/cartoes/{banco}/faturas/{ym} { fechamento?, valorBanco? } (R20).
func (s *Cartoes) Fechamento(ctx context.Context, sid, banco, ym string, body domain.Doc) (domain.Doc, error) {
	if !reMesFatura.MatchString(ym) {
		return nil, &ErroValidacao{"fatura invalida (use AAAA-MM)"}
	}
	fech, temFech := body["fechamento"]
	vb, temVB := body["valorBanco"]
	pag, temPag := body["pagador"]
	venc, temVenc := body["vencimento"]
	if !temFech && !temVB && !temPag && !temVenc {
		return nil, &ErroValidacao{"informe fechamento, valorBanco, pagador e/ou vencimento"}
	}
	var pagTxt, vencTxt string
	if temPag && pag != nil {
		t, ok := pag.(string)
		if !ok || t == "" {
			return nil, &ErroValidacao{"pagador invalido (o nome de um meio)"}
		}
		pagTxt = t
	}
	if temVenc && venc != nil {
		t, ok := venc.(string)
		if !ok {
			return nil, &ErroValidacao{"vencimento invalido (use AAAA-MM-DD)"}
		}
		d, err := time.Parse("2006-01-02", t)
		if err != nil || d.Format("2006-01-02") != t {
			return nil, &ErroValidacao{"essa data de vencimento nao existe (use AAAA-MM-DD)"}
		}
		vencTxt = t
	}
	var fechTexto string
	if temFech && fech != nil {
		t, ok := fech.(string)
		if !ok {
			return nil, &ErroValidacao{"fechamento invalido (use AAAA-MM-DD)"}
		}
		d, err := time.Parse("2006-01-02", t)
		if err != nil || d.Format("2006-01-02") != t {
			return nil, &ErroValidacao{"essa data nao existe (use AAAA-MM-DD)"}
		}
		if t[:7] != ym {
			return nil, &ErroValidacao{"o fechamento desta fatura precisa ser em " + domain.RotuloMes(ym)}
		}
		fechTexto = t
	}
	var vbNum float64
	if temVB && vb != nil {
		n, ok := vb.(float64)
		if !ok || n < 0 {
			return nil, &ErroValidacao{"valorBanco invalido (numero, nao negativo)"}
		}
		vbNum = n
	}
	err := s.atualizar(ctx, sid, func(por map[string]any) error {
		c, err := cartaoDe(por, banco)
		if err != nil {
			return err
		}
		if fechTexto != "" || pagTxt != "" || vencTxt != "" {
			if ini, _ := c["inicio"].(string); ini != "" && ym < ini {
				return &ErroValidacao{"fatura de historico (anterior ao inicio do controle) so aceita o valor do banco"}
			}
		}
		if pagTxt != "" {
			if err := s.validarPagador(ctx, sid, banco, pagTxt); err != nil {
				return err
			}
		}
		fs, _ := c["faturas"].(map[string]any)
		if fs == nil {
			fs = map[string]any{}
		}
		cur, _ := fs[ym].(map[string]any)
		if cur == nil {
			cur = map[string]any{}
		}
		if temFech {
			if fech == nil {
				delete(cur, "fechamento")
			} else {
				cur["fechamento"] = fechTexto
			}
		}
		if temVB {
			if vb == nil {
				delete(cur, "valorBanco")
			} else {
				cur["valorBanco"] = vbNum
			}
		}
		if temPag {
			if pag == nil {
				delete(cur, "pagador")
			} else {
				cur["pagador"] = pagTxt
			}
		}
		if temVenc {
			if venc == nil {
				delete(cur, "vencimento")
			} else {
				// o vencimento e depois do fechamento da fatura (o que vai valer depois deste comando) e no maximo 45 dias depois
				fs2 := map[string]any{}
				for k, v := range fs {
					fs2[k] = v
				}
				tmp := map[string]any{}
				for k, v := range cur {
					tmp[k] = v
				}
				fs2[ym] = tmp
				copia := map[string]any{}
				for k, v := range c {
					copia[k] = v
				}
				copia["faturas"] = fs2
				cc, _ := domain.CartaoDe(domain.Doc{"porBanco": map[string]any{banco: copia}}, banco)
				fech := domain.FechamentoDe(cc, ym)
				if vencTxt <= fech || vencTxt > domain.SomaDias(fech, 45) {
					return &ErroValidacao{"o vencimento precisa ser depois do fechamento (" + fech + ") e no maximo 45 dias depois dele"}
				}
				cur["vencimento"] = vencTxt
			}
		}
		if len(cur) == 0 {
			delete(fs, ym)
		} else {
			fs[ym] = cur
		}
		c["faturas"] = fs
		return nil
	})
	if err != nil {
		return nil, err
	}
	return domain.Doc{"meio": banco, "ym": ym}, nil
}

// totalAgora calcula o total que o servidor ve agora para a fatura ym.
func (s *Cartoes) totalAgora(ctx context.Context, sid, banco, ym string) (c domain.Cartao, total float64, rs []domain.RegistroFatura, listas domain.Doc, err error) {
	var cfgC domain.Doc
	if listas, cfgC, err = s.ler(ctx, sid); err != nil {
		return
	}
	c, ok := domain.CartaoDe(cfgC, banco)
	if !ok {
		err = ErrCartaoNaoEncontrado
		return
	}
	if !reMesFatura.MatchString(ym) {
		err = &ErroValidacao{"fatura invalida (use AAAA-MM)"}
		return
	}
	if rs, err = s.registros(ctx, sid, c, banco, ym, listas); err != nil {
		return
	}
	return c, domain.TotalFatura(rs, ym), rs, listas, nil
}

// Conferir: POST /spaces/{sid}/cartoes/{banco}/faturas/{ym}/conferir { aceitarDiferenca? } (R21). O total gravado e o que o SERVIDOR calculou.
func (s *Cartoes) Conferir(ctx context.Context, sid, sub, banco, ym string, body domain.Doc) (domain.Doc, error) {
	c, total, _, _, err := s.totalAgora(ctx, sid, banco, ym)
	if err != nil {
		return nil, err
	}
	if domain.FaturaHistorica(c, ym) {
		return nil, &ErroValidacao{"fatura de historico (anterior ao inicio do controle) nao se confere"}
	}
	aceitar, _ := body["aceitarDiferenca"].(bool)
	vb := c.Faturas[ym].ValorBanco
	if !aceitar {
		if vb == nil {
			return nil, &ErroDiferencaFatura{Total: total}
		}
		if d := domain.Arred(total - *vb); d != 0 {
			return nil, &ErroDiferencaFatura{Total: total, ValorBanco: vb, Diferenca: &d}
		}
	}
	em := s.rel.Agora().UTC().Format(formatoISO)
	err = s.atualizar(ctx, sid, func(por map[string]any) error {
		card, err := cartaoDe(por, banco)
		if err != nil {
			return err
		}
		fs, _ := card["faturas"].(map[string]any)
		if fs == nil {
			fs = map[string]any{}
		}
		cur, _ := fs[ym].(map[string]any)
		if cur == nil {
			cur = map[string]any{}
		}
		cur["conferida"] = map[string]any{"total": total, "em": em, "por": sub}
		fs[ym] = cur
		card["faturas"] = fs
		return nil
	})
	if err != nil {
		return nil, err
	}
	return domain.Doc{"meio": banco, "ym": ym, "conferida": domain.Doc{"total": total, "em": em, "por": sub}}, nil
}

// Desconferir: DELETE /spaces/{sid}/cartoes/{banco}/faturas/{ym}/conferir (R22).
func (s *Cartoes) Desconferir(ctx context.Context, sid, banco, ym string) (domain.Doc, error) {
	if !reMesFatura.MatchString(ym) {
		return nil, &ErroValidacao{"fatura invalida (use AAAA-MM)"}
	}
	err := s.atualizar(ctx, sid, func(por map[string]any) error {
		card, err := cartaoDe(por, banco)
		if err != nil {
			return err
		}
		fs, _ := card["faturas"].(map[string]any)
		cur, _ := fs[ym].(map[string]any)
		if cur == nil || cur["conferida"] == nil {
			return ErrFaturaNaoConferida
		}
		delete(cur, "conferida")
		if len(cur) == 0 {
			delete(fs, ym)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return domain.Doc{"meio": banco, "ym": ym}, nil
}

// RegistrarDiferenca: POST /spaces/{sid}/cartoes/{banco}/faturas/{ym}/diferenca (R23 e R24). Cria o lancamento do que falta (valorBanco - total).
func (s *Cartoes) RegistrarDiferenca(ctx context.Context, sid, sub, banco, ym string) (domain.Doc, error) {
	c, total, _, _, err := s.totalAgora(ctx, sid, banco, ym)
	if err != nil {
		return nil, err
	}
	vb := c.Faturas[ym].ValorBanco
	if vb == nil {
		return nil, &ErroValidacao{"a fatura nao tem o valor do app do cartao: informe-o antes de registrar a diferenca"}
	}
	falta := domain.Arred(*vb - total)
	if !(falta > 0) {
		return nil, &ErroValidacao{"nao falta valor na fatura: o valor do app do cartao nao e maior que o total"}
	}
	status := ""
	for _, e := range domain.Catalogo { // o status de credito que o servidor sugere (CREDITO)
		if e.Selecionavel && e.Credito {
			status = e.Nome
			break
		}
	}
	if status == "" {
		return nil, &ErroValidacao{"nao ha status de credito para registrar a diferenca"}
	}
	_, fim := domain.PeriodoFatura(c, ym)
	data, err := time.ParseInLocation("2006-01-02T15:04:05", fim+"T12:00:00", domain.FusoApp)
	if err != nil {
		return nil, ErrInterno
	}
	_, err = s.criador.Criar(ctx, sid, sub, domain.Lancamento{
		"dataEvento": data.UTC().Format(formatoISO), "status": status, "tipo": "saida", "valor": falta, "banco": banco,
		"descricao": "Ajuste da fatura " + domain.RotuloMes(ym) + " (juros, anuidade ou IOF)", "nota": "",
		"tags": []any{}, "excluirDoTotal": false, "faturaAjuste": 0.0, "valorEstimado": false,
	}, true)
	if err != nil {
		return nil, err
	}
	return s.Fatura(ctx, sid, sub, true, banco, ym)
}
