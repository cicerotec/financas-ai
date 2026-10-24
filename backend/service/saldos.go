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
	erro    bool               // a leitura dos lancamentos deste meio falhou (so acontece para meio sem inicio do controle)
	semDe   bool               // sem inicio do controle nem ponto de partida: precisa ler desde o comeco
}

// leitura conta o que um calculo leu da tabela e quanto tempo isso levou. So vai para o log (numeros, nunca dados): serve para saber onde o tempo
// de um GET /saldos e gasto (a leitura da tabela, o calculo ou a gravacao do cache) sem adivinhar.
type leitura struct {
	paginas, itens int
	dur            time.Duration
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
func (s *Saldos) calcular(ctx context.Context, sid string, bancos []string) (map[string]*calculoBanco, domain.Doc, string, *leitura, error) {
	listas, err := s.cfg.Ler(ctx, sid, "listas")
	if err != nil {
		log.Println("saldos: listas:", err)
		return nil, nil, "", nil, ErrInterno
	}
	cfgSaldos, err := s.cfg.Ler(ctx, sid, "saldos")
	if err != nil {
		log.Println("saldos: cfg saldos:", err)
		return nil, nil, "", nil, ErrInterno
	}
	atual := domain.MesAtual(s.rel.Agora())
	por := map[string]*calculoBanco{}
	leit := &leitura{}
	var menor *time.Time
	for _, nome := range bancos {
		c := &calculoBanco{nome: nome, cfg: domain.ConfigBancoDe(cfgSaldos, nome)}
		if c.item, err = s.saldos.Ler(ctx, sid, nome); err != nil {
			log.Println("saldos: item:", err)
			return nil, nil, "", nil, ErrInterno
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
			c.semDe = true
		}
		c.acc = domain.NovoAcumulador(listas, de)
		por[nome] = c
	}
	if len(bancos) == 0 {
		return por, listas, atual, leit, nil
	}
	// Duas passadas: os meios com inicio do controle (ou ponto de partida) leem so de la em diante; os sem inicio leem desde o comeco, mas
	// numa passada a parte, para um meio sem configuracao nao obrigar os outros a ler tudo.
	base := domain.FiltroTx{Asc: true, Limite: 1000}
	if len(bancos) == 1 {
		base.Banco = bancos[0]
	}
	if menor != nil {
		f := base
		f.De = menor.UTC().Format("2006-01-02T15:04:05.000Z")
		if err := s.lerEAcumular(ctx, sid, f, por, false, leit); err != nil {
			log.Println("saldos: listar:", err)
			return nil, nil, "", nil, ErrListar
		}
	}
	for _, c := range por {
		if c.semDe {
			// uma falha aqui (a tabela limitando a leitura, por exemplo) marca so estes meios com erro; os outros seguem
			if err := s.lerEAcumular(ctx, sid, base, por, true, leit); err != nil {
				log.Println("saldos: listar (meios sem inicio):", err)
				for _, x := range por {
					if x.semDe {
						x.erro = true
					}
				}
			}
			break
		}
	}
	for _, c := range por {
		if c.erro {
			continue
		}
		c.calculo = domain.Cascata(c.acc.PorMes(), c.cfg, c.partida, atual)
		c.mapa = map[string]float64{}
		for m, v := range domain.CachesValidos(c.item, c.cfg, c.base, atual) {
			c.mapa[m] = v
		}
		for _, m := range c.calculo.Meses {
			c.mapa[m.Mes] = m.Fechamento
		}
	}
	return por, listas, atual, leit, nil
}

// lerEAcumular le, em paginas, os lancamentos do filtro e soma cada um no acumulador do seu banco. soDeSemInicio: so soma nos meios que
// precisam ler desde o comeco (a passada deles); nas outras passadas esses meios sao ignorados.
func (s *Saldos) lerEAcumular(ctx context.Context, sid string, filtro domain.FiltroTx, por map[string]*calculoBanco, soDeSemInicio bool, leit *leitura) error {
	for {
		t0 := time.Now() // so para medir a leitura (vai para o log); a regra do saldo usa o Relogio
		itens, proximo, err := s.lanc.Listar(ctx, sid, filtro)
		leit.dur += time.Since(t0)
		if err != nil {
			return err
		}
		leit.paginas++
		leit.itens += len(itens)
		for _, l := range itens {
			b, _ := l["banco"].(string)
			if c := por[b]; c != nil && c.semDe == soDeSemInicio {
				c.acc.Somar(l)
			}
		}
		if proximo == "" {
			return nil
		}
		filtro.Cursor = proximo
	}
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
func (s *Saldos) gravarCache(ctx context.Context, sid string, c *calculoBanco, atual string) bool {
	if c.cfg.DataInicio == "" {
		return false
	}
	novo := map[string]any{}
	for m, v := range domain.CachesValidos(c.item, c.cfg, c.base, atual) {
		novo[m] = v
	}
	for m, v := range c.calculo.Fechamentos {
		novo[m] = v
	}
	if len(novo) == 0 && c.item == nil {
		return false
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
			return false
		}
	}
	d := domain.Doc{"banco": c.nome}
	for k, v := range c.item {
		d[k] = v
	}
	d["caches"], d["base"] = novo, c.base
	delete(d, "invalidoDe")
	if _, err := s.saldos.Salvar(ctx, sid, c.nome, d, c.versao); err != nil {
		if !errors.Is(err, domain.ErrConflitoConfig) {
			log.Printf("saldos: gravar cache sid=%s banco=%s: %v", sid, c.nome, err)
		}
		return false
	}
	return true
}

// Listar: GET /spaces/{sid}/saldos. O saldo de cada meio da lista, os 12 ultimos meses e o total.
func (s *Saldos) Listar(ctx context.Context, sid string) (domain.Doc, error) {
	listas, err := s.cfg.Ler(ctx, sid, "listas")
	if err != nil {
		log.Println("saldos: listas:", err)
		return nil, ErrInterno
	}
	nomes := nomesDeBancos(listas)
	inicio := time.Now()
	por, _, atual, leit, err := s.calcular(ctx, sid, nomes)
	if err != nil {
		return nil, err
	}
	tCalculo := time.Since(inicio)
	cachesGravados := 0
	itens := []domain.Doc{}
	total, dentro, fora, erros := 0.0, 0, 0, 0
	for _, nome := range nomes {
		c := por[nome]
		if c.erro {
			itens = append(itens, domain.Doc{"meio": nome, "erro": true, "saldo": 0.0, "foraDoTotal": c.cfg.ForaDoTotal, "semInicio": !c.cfg.Existe, "meses": []domain.Doc{}})
			erros++
			continue
		}
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
		if s.gravarCache(ctx, sid, c, atual) {
			cachesGravados++
		}
	}
	log.Printf("saldos: meios=%d paginas=%d itens=%d leitura=%dms calculo=%dms total=%dms caches_gravados=%d",
		len(nomes), leit.paginas, leit.itens, leit.dur.Milliseconds(), tCalculo.Milliseconds(), time.Since(inicio).Milliseconds(), cachesGravados)
	return domain.Doc{
		"mesAtual": atual,
		"total":    domain.Doc{"valor": domain.Arred(total), "meios": dentro, "fora": fora, "erros": erros},
		"meios":    itens,
	}, nil
}

// Calcular devolve o saldo de um so meio, sem gravar cache (usado pela conferencia).
func (s *Saldos) Calcular(ctx context.Context, sid, banco string) (ResultadoBanco, error) {
	por, _, _, _, err := s.calcular(ctx, sid, []string{banco})
	if err != nil {
		return ResultadoBanco{}, err
	}
	c := por[banco]
	if c.erro {
		return ResultadoBanco{}, ErrListar
	}
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
