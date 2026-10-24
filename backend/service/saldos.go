package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"financas/domain"
)

// LeitorDeLancamentos e o que o calculo do saldo precisa do banco: ler os lancamentos do espaco, em ordem de data, em paginas.
// Quem implementa e o repository de lancamentos (o mesmo Listar do GET /tx).
type LeitorDeLancamentos interface {
	Listar(ctx context.Context, sid string, f domain.FiltroTx) (itens []domain.Doc, proximo string, err error)
}

// ErroDiferenca: o extrato (real) nao bate com o saldo que o servidor calculou. E um 409 com o que ele calculou, para a pessoa
// decidir entre cancelar e pedir o ajuste.
type ErroDiferenca struct{ Calculado, Real, Diferenca float64 }

func (e *ErroDiferenca) Error() string {
	return fmt.Sprintf("o saldo calculado (%.2f) nao bate com o extrato (%.2f): confirme o ajuste de %.2f", e.Calculado, e.Real, e.Diferenca)
}

// Saldos calcula, no servidor, o saldo de cada meio (banco), o total e os meses (item 40). Le os lancamentos do espaco uma vez
// so para todos os bancos, parte do ultimo ponto verificado (mes conferido ou cache valido) e grava o cache dos meses encerrados.
// O navegador so exibe: nenhum numero dele vira verdade.
type Saldos struct {
	lanc   LeitorDeLancamentos
	cfg    RepositorioConfig
	saldos RepositorioSaldos
	rel    Relogio
}

func NewSaldos(lanc LeitorDeLancamentos, cfg RepositorioConfig, saldos RepositorioSaldos, rel Relogio) *Saldos {
	return &Saldos{lanc: lanc, cfg: cfg, saldos: saldos, rel: rel}
}

// calculoBanco e o estado de um banco durante uma chamada.
type calculoBanco struct {
	nome    string
	cfg     domain.ConfigBanco
	item    domain.Doc // o item SALDO#<banco> como estava ANTES de ler os lancamentos (nil se nao existe)
	versao  float64
	base    string
	partida *domain.Partida
	acc     *domain.Acumulador
	calculo domain.Calculo
	mapa    map[string]float64 // fechamento de cada mes conhecido (cache e cascata)
}

// ResultadoBanco e o que um calculo devolve de um banco.
type ResultadoBanco struct {
	Saldo float64
	Mapa  map[string]float64
	Item  domain.Doc
	Cfg   domain.ConfigBanco
}

func nomesDeBancos(listas domain.Doc) []string {
	var nomes []string
	vistos := map[string]bool{}
	arr, _ := listas["banco"].([]any)
	for _, x := range arr {
		if n, ok := x.(string); ok && n != "" && !vistos[n] {
			vistos[n] = true
			nomes = append(nomes, n)
		}
	}
	return nomes
}

// calcular le a configuracao, o item de cada banco e os lancamentos (uma passada, filtrando por banco se so um foi pedido) e faz a
// cascata de cada um. A version de cada item e lida ANTES dos lancamentos: se alguem gravar no meio, o cache nao e gravado.
func (s *Saldos) calcular(ctx context.Context, sid string, bancos []string) (map[string]*calculoBanco, domain.Doc, string, error) {
	listas, err := s.cfg.Ler(ctx, sid, "listas")
	if err != nil {
		log.Println("saldos: listas:", err)
		return nil, nil, "", ErrInterno
	}
	cfgSaldos, err := s.cfg.Ler(ctx, sid, "saldos")
	if err != nil {
		log.Println("saldos: cfg saldos:", err)
		return nil, nil, "", ErrInterno
	}
	atual := domain.MesAtual(s.rel.Agora())
	por := map[string]*calculoBanco{}
	var menor *time.Time
	semLimite := false
	for _, nome := range bancos {
		c := &calculoBanco{nome: nome, cfg: domain.ConfigBancoDe(cfgSaldos, nome)}
		if c.item, err = s.saldos.Ler(ctx, sid, nome); err != nil {
			log.Println("saldos: item:", err)
			return nil, nil, "", ErrInterno
		}
		c.versao = versaoDe(c.item)
		c.base = domain.BaseCache(c.cfg, listas)
		if p, ok := domain.PontoDePartida(c.item, c.cfg, c.base, atual); ok {
			c.partida = &p
		}
		var de *time.Time
		if limite, ok := domain.LimiteAPartirDe(c.cfg, c.partida); ok {
			de = &limite
			if menor == nil || limite.Before(*menor) {
				l := limite
				menor = &l
			}
		} else {
			semLimite = true
		}
		c.acc = domain.NovoAcumulador(listas, de)
		por[nome] = c
	}
	if len(bancos) == 0 {
		return por, listas, atual, nil
	}
	filtro := domain.FiltroTx{Asc: true, Limite: 1000}
	if !semLimite && menor != nil {
		filtro.De = menor.UTC().Format("2006-01-02T15:04:05.000Z")
	}
	if len(bancos) == 1 {
		filtro.Banco = bancos[0]
	}
	for {
		itens, proximo, err := s.lanc.Listar(ctx, sid, filtro)
		if err != nil {
			log.Println("saldos: listar:", err)
			return nil, nil, "", ErrListar
		}
		for _, l := range itens {
			b, _ := l["banco"].(string)
			if c := por[b]; c != nil {
				c.acc.Somar(l)
			}
		}
		if proximo == "" {
			break
		}
		filtro.Cursor = proximo
	}
	for _, c := range por {
		c.calculo = domain.Cascata(c.acc.PorMes(), c.cfg, c.partida, atual)
		c.mapa = map[string]float64{}
		for m, v := range domain.CachesValidos(c.item, c.cfg, c.base, atual) {
			c.mapa[m] = v
		}
		for _, m := range c.calculo.Meses {
			c.mapa[m.Mes] = m.Fechamento
		}
	}
	return por, listas, atual, nil
}

func conferidosDe(item domain.Doc) map[string]domain.Doc {
	out := map[string]domain.Doc{}
	conf, _ := item["conferidos"].(map[string]any)
	for m, v := range conf {
		if r, ok := v.(map[string]any); ok {
			out[m] = r
		}
	}
	return out
}

func mesesParaJSON(ms []domain.MesPainel) []domain.Doc {
	out := make([]domain.Doc, 0, len(ms))
	for _, m := range ms {
		d := domain.Doc{"mes": m.Mes, "valor": m.Valor, "estado": m.Estado}
		if m.Conferido != nil {
			d["conferido"] = domain.Doc{"real": m.Conferido["real"], "ajuste": m.Conferido["ajuste"], "em": m.Conferido["em"], "por": m.Conferido["por"]}
		}
		out = append(out, d)
	}
	return out
}

// gravarCache guarda no servidor os fechamentos dos meses encerrados, calculados por ele, com o controle otimista da version lida
// antes dos lancamentos. Se nada mudou, ou se outra escrita chegou antes, nao grava (o saldo devolvido continua certo).
func (s *Saldos) gravarCache(ctx context.Context, sid string, c *calculoBanco, atual string) {
	if c.cfg.DataInicio == "" {
		return
	}
	novo := map[string]any{}
	for m, v := range domain.CachesValidos(c.item, c.cfg, c.base, atual) {
		novo[m] = v
	}
	for m, v := range c.calculo.Fechamentos {
		novo[m] = v
	}
	if len(novo) == 0 && c.item == nil {
		return
	}
	if c.item != nil {
		antigo, _ := c.item["caches"].(map[string]any)
		inv, _ := c.item["invalidoDe"].(string)
		b, _ := c.item["base"].(string)
		igual := b == c.base && inv == "" && len(antigo) == len(novo)
		for m, v := range novo {
			if igual && antigo[m] != v {
				igual = false
			}
		}
		if igual {
			return
		}
	}
	d := domain.Doc{"banco": c.nome}
	for k, v := range c.item {
		d[k] = v
	}
	d["caches"], d["base"] = novo, c.base
	delete(d, "invalidoDe")
	if _, err := s.saldos.Salvar(ctx, sid, c.nome, d, c.versao); err != nil && !errors.Is(err, domain.ErrConflitoConfig) {
		log.Printf("saldos: gravar cache sid=%s banco=%s: %v", sid, c.nome, err)
	}
}

// Listar: GET /spaces/{sid}/saldos. O saldo de cada meio da lista, os 12 ultimos meses e o total.
func (s *Saldos) Listar(ctx context.Context, sid string) (domain.Doc, error) {
	listas, err := s.cfg.Ler(ctx, sid, "listas")
	if err != nil {
		log.Println("saldos: listas:", err)
		return nil, ErrInterno
	}
	nomes := nomesDeBancos(listas)
	por, _, atual, err := s.calcular(ctx, sid, nomes)
	if err != nil {
		return nil, err
	}
	itens := []domain.Doc{}
	total, dentro, fora := 0.0, 0, 0
	for _, nome := range nomes {
		c := por[nome]
		item := domain.Doc{
			"meio": nome, "saldo": c.calculo.Saldo, "foraDoTotal": c.cfg.ForaDoTotal, "semInicio": !c.cfg.Existe,
			"meses": mesesParaJSON(domain.MesesDoPainel(c.mapa, conferidosDe(c.item), atual)),
		}
		if c.cfg.Existe {
			item["inicio"] = domain.Doc{"dataInicio": c.cfg.DataInicio, "saldoInicial": c.cfg.SaldoInicial}
		}
		itens = append(itens, item)
		if c.cfg.ForaDoTotal {
			fora++
		} else {
			total += c.calculo.Saldo
			dentro++
		}
		s.gravarCache(ctx, sid, c, atual)
	}
	return domain.Doc{
		"mesAtual": atual,
		"total":    domain.Doc{"valor": domain.Arred(total), "meios": dentro, "fora": fora},
		"meios":    itens,
	}, nil
}

// Calcular devolve o saldo de um so meio, sem gravar cache (usado pela conferencia).
func (s *Saldos) Calcular(ctx context.Context, sid, banco string) (ResultadoBanco, error) {
	por, _, _, err := s.calcular(ctx, sid, []string{banco})
	if err != nil {
		return ResultadoBanco{}, err
	}
	c := por[banco]
	return ResultadoBanco{Saldo: c.calculo.Saldo, Mapa: c.mapa, Item: c.item, Cfg: c.cfg}, nil
}

// Listas devolve CFG#LISTAS (para o status do ajuste).
func (s *Saldos) listas(ctx context.Context, sid string) (domain.Doc, error) {
	l, err := s.cfg.Ler(ctx, sid, "listas")
	if err != nil {
		log.Println("saldos: listas:", err)
		return nil, ErrInterno
	}
	return l, nil
}
