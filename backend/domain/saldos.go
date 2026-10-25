package domain

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Saldo calculado no servidor (item 40, .ai/specs/saldo-servidor). Tudo aqui e funcao pura: o instante, a configuracao e os
// lancamentos vem de quem chama, e o resultado e o mesmo que o navegador calculava (cascata mensal, arredondada a cada mes).

// ConfigBanco e o que CFG#SALDOS guarda de um banco (um meio de pagamento): o ponto de partida do controle.
type ConfigBanco struct {
	Existe       bool    // ha configuracao para o banco; sem ela o saldo comeca do primeiro movimento
	SaldoInicial float64 // o que a conta tinha em DataInicio
	DataInicio   string  // AAAA-MM-DD; vazio = sem inicio
	ForaDoTotal  bool    // o saldo aparece mas nao entra no total
}

// ConfigBancoDe le a configuracao de um banco de CFG#SALDOS.
func ConfigBancoDe(cfgSaldos Doc, banco string) ConfigBanco {
	por, _ := cfgSaldos["porBanco"].(map[string]any)
	b, ok := por[banco].(map[string]any)
	if !ok {
		return ConfigBanco{}
	}
	c := ConfigBanco{Existe: true}
	c.SaldoInicial, _ = b["saldoInicial"].(float64)
	c.DataInicio, _ = b["dataInicio"].(string)
	c.ForaDoTotal, _ = b["foraDoTotal"].(bool)
	return c
}

// Arred arredonda a 2 casas como o navegador fazia (Math.round: a metade vai para cima).
func Arred(v float64) float64 { return math.Floor(v*100+0.5) / 100 }

func ligados(m map[string]bool) string {
	var ks []string
	for k, v := range m {
		if v {
			ks = append(ks, k)
		}
	}
	sort.Strings(ks)
	return strings.Join(ks, ",")
}

// ContaNoSaldo diz se um lancamento entra no saldo do seu banco: nao esta excluido, oculto nem previsto, e o status (pelo catalogo do
// servidor) mexe no saldo. A configuracao de Listas (afetaSaldoBanco, statusFuturo) deixou de ser lida (item 41).
func ContaNoSaldo(l Lancamento) bool {
	if Excluido(l) || EhPrevisto(l) {
		return false
	}
	if o, _ := l["oculto"].(bool); o {
		return false
	}
	return ContaNoSaldoStatus(StatusDe(l))
}

// BaseCache e a assinatura da configuracao com a qual um cache foi calculado: saldo inicial, data de inicio e as regras de status do catalogo
// (quais contam no saldo). Um cache so vale para a base em que foi calculado. O prefixo v3 invalida os caches calculados com a regra
// de Listas, que o catalogo substituiu (item 41).
func BaseCache(cfg ConfigBanco) string {
	var contam []string
	for _, e := range Catalogo {
		if e.ContaNoSaldo && !e.Previsto {
			contam = append(contam, e.Nome)
		}
	}
	return "v3|" + strconv.FormatFloat(cfg.SaldoInicial, 'f', -1, 64) + "|" + cfg.DataInicio + "|" + strings.Join(contam, ",")
}

// InicioDoControle devolve o instante em que a data de inicio (AAAA-MM-DD) comeca a valer: 00:00 em Sao Paulo.
func InicioDoControle(dataInicio string) (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02", dataInicio, FusoApp)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// Partida e o ponto de onde a cascata recomeca: o fechamento verificado de um mes.
type Partida struct {
	Mes   string
	Saldo float64
}

// somaMes soma n meses a um mes AAAA-MM.
func somaMes(ym string, n int) string {
	t, err := time.ParseInLocation("2006-01", ym, FusoApp)
	if err != nil {
		return ""
	}
	return time.Date(t.Year(), t.Month()+time.Month(n), 1, 0, 0, 0, 0, FusoApp).Format("2006-01")
}

// SomaMes e somaMes exportada para quem precisa andar de mes em mes.
func SomaMes(ym string, n int) string { return somaMes(ym, n) }

// LimiteAPartirDe devolve o instante a partir do qual os lancamentos contam para um banco: a data de inicio e, se a cascata
// parte de um mes verificado, o inicio do mes seguinte a ele (o que vale mais tarde). ok=false quando nao ha limite (conta tudo).
func LimiteAPartirDe(cfg ConfigBanco, p *Partida) (time.Time, bool) {
	var limite time.Time
	tem := false
	if ini, ok := InicioDoControle(cfg.DataInicio); ok {
		limite, tem = ini, true
	}
	if p != nil {
		prox, err := time.ParseInLocation("2006-01", somaMes(p.Mes, 1), FusoApp)
		if err == nil && (!tem || prox.After(limite)) {
			limite, tem = prox, true
		}
	}
	return limite, tem
}

// Acumulador soma, por mes, o que os lancamentos de um banco contam no saldo.
type Acumulador struct {
	de     *time.Time
	porMes map[string]float64
}

// NovoAcumulador cria um acumulador; de, se houver, e o instante a partir do qual os lancamentos contam.
func NovoAcumulador(de *time.Time) *Acumulador {
	return &Acumulador{de: de, porMes: map[string]float64{}}
}

// Somar acrescenta um lancamento (ja do banco certo) se ele conta no saldo e e do periodo.
func (a *Acumulador) Somar(l Lancamento) {
	if !ContaNoSaldo(l) {
		return
	}
	data, _ := l["dataEvento"].(string)
	t, err := time.Parse(time.RFC3339Nano, data)
	if err != nil {
		return
	}
	if a.de != nil && t.Before(*a.de) {
		return
	}
	valor, _ := l["valor"].(float64)
	if l["tipo"] != "entrada" {
		valor = -valor
	}
	a.porMes[t.In(FusoApp).Format("2006-01")] += valor
}

// PorMes devolve o total de cada mes (a copia pertence ao acumulador).
func (a *Acumulador) PorMes() map[string]float64 { return a.porMes }

// MesFechamento e o saldo de um banco ao fim de um mes.
type MesFechamento struct {
	Mes        string
	Fechamento float64
}

// Calculo e o resultado da cascata de um banco.
type Calculo struct {
	Saldo       float64            // saldo atual (o ultimo mes da cascata)
	Fechamentos map[string]float64 // fechamento de cada mes encerrado (anterior ao atual), so com data de inicio: candidatos a cache
	Meses       []MesFechamento    // todos os meses calculados, do primeiro ao ultimo
}

// Cascata calcula o saldo: parte do saldo inicial (ou do fechamento da Partida) e, mes a mes, ate o maior entre o mes atual e o
// ultimo mes com movimento, faz acc = arred(acc + soma do mes).
func Cascata(porMes map[string]float64, cfg ConfigBanco, p *Partida, atual string) Calculo {
	var movs []string
	for m := range porMes {
		movs = append(movs, m)
	}
	sort.Strings(movs)
	acc := cfg.SaldoInicial
	var m string
	switch {
	case p != nil:
		acc = p.Saldo
		m = somaMes(p.Mes, 1)
	case len(cfg.DataInicio) >= 7:
		m = cfg.DataInicio[:7]
	case len(movs) > 0:
		m = movs[0]
	default:
		m = atual
	}
	fim := atual
	if len(movs) > 0 && movs[len(movs)-1] > fim {
		fim = movs[len(movs)-1]
	}
	c := Calculo{Fechamentos: map[string]float64{}}
	for ; m != "" && m <= fim; m = somaMes(m, 1) {
		acc = Arred(acc + porMes[m])
		if m < atual && cfg.DataInicio != "" {
			c.Fechamentos[m] = acc
		}
		c.Meses = append(c.Meses, MesFechamento{Mes: m, Fechamento: acc})
	}
	c.Saldo = acc
	return c
}

// CachesValidos devolve os fechamentos em cache de um item SALDO#<banco> que ainda valem: a base e a do servidor, o mes e anterior ao
// atual e ao invalidoDe, e nao e anterior ao inicio do controle. Sem data de inicio nao ha cache.
func CachesValidos(item Doc, cfg ConfigBanco, base, atual string) map[string]float64 {
	out := map[string]float64{}
	if item == nil || cfg.DataInicio == "" {
		return out
	}
	if b, _ := item["base"].(string); b != base {
		return out
	}
	ate, _ := item["invalidoDe"].(string)
	if ate == "" {
		ate = "9999-99"
	}
	ini := ""
	if len(cfg.DataInicio) >= 7 {
		ini = cfg.DataInicio[:7]
	}
	caches, _ := item["caches"].(map[string]any)
	for m, v := range caches {
		n, ok := v.(float64)
		if ok && m < ate && m < atual && m >= ini {
			out[m] = n
		}
	}
	return out
}

// PontoDePartida escolhe de onde a cascata recomeca: o fechamento mais recente entre os meses conferidos e os caches validos (o
// conferido vence o cache do mesmo mes). Meses anteriores ao inicio do controle nao servem.
func PontoDePartida(item Doc, cfg ConfigBanco, base, atual string) (Partida, bool) {
	var melhor Partida
	tem := false
	conf, _ := item["conferidos"].(map[string]any)
	for m, v := range conf {
		if len(cfg.DataInicio) >= 7 && m < cfg.DataInicio[:7] {
			continue
		}
		reg, _ := v.(map[string]any)
		saldo, ok := reg["saldo"].(float64)
		if ok && (!tem || m > melhor.Mes) {
			melhor, tem = Partida{Mes: m, Saldo: saldo}, true
		}
	}
	for m, saldo := range CachesValidos(item, cfg, base, atual) {
		if !tem || m > melhor.Mes {
			melhor, tem = Partida{Mes: m, Saldo: saldo}, true
		}
	}
	return melhor, tem
}

// MesPainel e um mes na lista que a tela de Saldos mostra.
type MesPainel struct {
	Mes       string
	Valor     float64
	Estado    string // conferido, travado, calculado ou aberto
	Conferido Doc    // o registro da conferencia, so quando conferido
}

// MesesDoPainel monta os 12 ultimos meses (do atual para tras, sem os posteriores ao atual) com o estado de cada um: conferido
// (ha conferencia, e o valor mostrado e o conferido), travado (anterior ao ultimo conferido), aberto (o mes atual) ou calculado.
// mapa traz o fechamento de cada mes conhecido (cache e cascata); conf, o registro de cada mes conferido.
func MesesDoPainel(mapa map[string]float64, conf map[string]Doc, atual string) []MesPainel {
	todos := map[string]float64{}
	for m, v := range mapa {
		todos[m] = v
	}
	ult := ""
	for m, reg := range conf {
		if s, ok := reg["saldo"].(float64); ok {
			todos[m] = s
		}
		if m > ult {
			ult = m
		}
	}
	var meses []string
	for m := range todos {
		if m <= atual {
			meses = append(meses, m)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(meses)))
	if len(meses) > 12 {
		meses = meses[:12]
	}
	out := make([]MesPainel, 0, len(meses))
	for _, m := range meses {
		p := MesPainel{Mes: m, Valor: todos[m]}
		switch {
		case conf[m] != nil:
			p.Estado, p.Conferido = "conferido", conf[m]
		case m <= ult:
			p.Estado = "travado"
		case m == atual:
			p.Estado = "aberto"
		default:
			p.Estado = "calculado"
		}
		out = append(out, p)
	}
	return out
}

// StatusDeAjuste devolve o status de um lancamento de ajuste da conferencia: o primeiro selecionavel do catalogo que conta no saldo
// (PAGO). Vazio se nenhum conta.
func StatusDeAjuste() string {
	for _, e := range Catalogo {
		if e.Selecionavel && e.ContaNoSaldo && !e.Previsto {
			return e.Nome
		}
	}
	return ""
}

// UltimoDiaDoMes devolve o instante (ISO, UTC) do ultimo dia do mes AAAA-MM as 12:00 em Sao Paulo.
func UltimoDiaDoMes(ym string) (string, bool) {
	t, err := time.ParseInLocation("2006-01", ym, FusoApp)
	if err != nil {
		return "", false
	}
	d := time.Date(t.Year(), t.Month()+1, 0, 12, 0, 0, 0, FusoApp)
	return d.UTC().Format("2006-01-02T15:04:05.000Z"), true
}
