package domain

import (
	"math"
	"sort"
	"strconv"
	"time"
)

// Fatura do cartao calculada no servidor (item 33a, .ai/specs/fatura-servidor). Tudo aqui e funcao pura: o instante, a configuracao
// do cartao e os lancamentos vem de quem chama, e o resultado e o mesmo que o navegador calculava (web/js/app.js, aba Cartao).
// Os dias sao AAAA-MM-DD no fuso de Sao Paulo e os meses AAAA-MM.

// Cartao e o que CFG#CARTOES.porBanco[banco] guarda de um cartao: o dia padrao do fechamento, o inicio do controle e, por fatura,
// o fechamento escolhido, o valor do app do cartao e a conferencia.
type Cartao struct {
	DiaPadrao     int
	Inicio        string // AAAA-MM; vazio = sem inicio
	Pagador       string // o meio com que a fatura costuma ser paga (so uma sugestao); vazio = nenhum
	DiaVencimento int    // 1 a 31; 0 = sem vencimento padrao
	Aviso         Doc    // { dias, insistir }: so o modelo copiado para a fatura quando ela ganha o primeiro vencimento (item 48); nunca vale por si
	Faturas       map[string]FaturaCfg
}

// FaturaCfg e a configuracao de uma fatura (CFG#CARTOES.porBanco[banco].faturas[ym]).
type FaturaCfg struct {
	Fechamento string   // AAAA-MM-DD escolhido para o mes; vazio = vale o dia herdado
	ValorBanco *float64 // o valor que a pessoa digitou do app do cartao
	Conferida  Doc      // { total, em, por }, gravado so pelo servidor
	Pagador    string   // troca o pagador so desta fatura
	Vencimento string   // AAAA-MM-DD escolhido para esta fatura
	Aviso      Doc      // { dias, insistir }: aviso no Telegram desta fatura (item 46); so existe com Vencimento
}

// CartaoDe le a configuracao de um cartao de CFG#CARTOES; false se o banco nao e cartao.
func CartaoDe(cfgCartoes Doc, banco string) (Cartao, bool) {
	por, _ := cfgCartoes["porBanco"].(map[string]any)
	b, ok := por[banco].(map[string]any)
	if !ok {
		return Cartao{}, false
	}
	c := Cartao{Faturas: map[string]FaturaCfg{}}
	if n, ok := b["diaPadrao"].(float64); ok {
		c.DiaPadrao = int(n)
	}
	c.Inicio, _ = b["inicio"].(string)
	c.Pagador, _ = b["pagador"].(string)
	if n, ok := b["diaVencimento"].(float64); ok {
		c.DiaVencimento = int(n)
	}
	c.Aviso, _ = b["aviso"].(map[string]any)
	fs, _ := b["faturas"].(map[string]any)
	for ym, v := range fs {
		m, _ := v.(map[string]any)
		f := FaturaCfg{}
		f.Fechamento, _ = m["fechamento"].(string)
		f.Pagador, _ = m["pagador"].(string)
		f.Vencimento, _ = m["vencimento"].(string)
		f.Aviso, _ = m["aviso"].(map[string]any)
		if n, ok := m["valorBanco"].(float64); ok {
			f.ValorBanco = &n
		}
		if cf, ok := m["conferida"].(map[string]any); ok {
			f.Conferida = cf
		}
		c.Faturas[ym] = f
	}
	return c, true
}

// ---------- datas ----------

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// DiasNoMes devolve quantos dias tem o mes AAAA-MM.
func DiasNoMes(ym string) int {
	a, _ := strconv.Atoi(ym[:4])
	m, _ := strconv.Atoi(ym[5:7])
	return time.Date(a, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// SomaDias soma n dias a um dia AAAA-MM-DD.
func SomaDias(dia string, n int) string {
	t, err := DiaParaTempo(dia)
	if err != nil {
		return dia
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}

// NumeroDoDia conta os dias desde 1970-01-01 (so para comparar distancias).
func NumeroDoDia(dia string) int {
	t, err := DiaParaTempo(dia)
	if err != nil {
		return 0
	}
	return int(t.Unix() / 86400)
}

// DiferencaDeMeses devolve a - b em meses (a e b em AAAA-MM).
func DiferencaDeMeses(a, b string) int {
	aa, _ := strconv.Atoi(a[:4])
	am, _ := strconv.Atoi(a[5:7])
	ba, _ := strconv.Atoi(b[:4])
	bm, _ := strconv.Atoi(b[5:7])
	return (aa-ba)*12 + (am - bm)
}

// ---------- fechamento e periodo (R1 a R4) ----------

// FechamentoDe e o dia de fechamento da fatura ym: o escolhido para o mes; senao o dia do mes mais recente anterior (ate 60 meses) que
// tenha fechamento escolhido; senao o dia padrao (1 se nao houver). O dia e limitado ao ultimo dia de ym.
func FechamentoDe(c Cartao, ym string) string {
	if f := c.Faturas[ym].Fechamento; f != "" {
		return f
	}
	dia := c.DiaPadrao
	if dia == 0 {
		dia = 1
	}
	for i := 1; i <= 60; i++ {
		if g := c.Faturas[SomaMes(ym, -i)].Fechamento; g != "" {
			if n, err := strconv.Atoi(g[8:10]); err == nil {
				dia = n
			}
			break
		}
	}
	if max := DiasNoMes(ym); dia > max {
		dia = max
	}
	return ym + "-" + pad2(dia)
}

// PeriodoFatura vai do dia seguinte ao fechamento de ym-1 ate o fechamento de ym, inclusive.
func PeriodoFatura(c Cartao, ym string) (ini, fim string) {
	return SomaDias(FechamentoDe(c, SomaMes(ym, -1)), 1), FechamentoDe(c, ym)
}

// FaturaBase e a fatura de uma compra pelo seu dia: o mes do dia se ele e ate o fechamento, senao o seguinte.
func FaturaBase(c Cartao, dia string) string {
	ym := dia[:7]
	if dia <= FechamentoDe(c, ym) {
		return ym
	}
	return SomaMes(ym, 1)
}

// FaturaAtual e a fatura base do dia de hoje.
func FaturaAtual(c Cartao, hoje string) string { return FaturaBase(c, hoje) }

// RotuloFatura e o nome da fatura pela distancia da atual: Anterior (-1), Atual (0), Proxima (+1) e mmm/aa nos outros.
func RotuloFatura(off int, ym string) string {
	switch off {
	case -1:
		return "Anterior"
	case 0:
		return "Atual"
	case 1:
		return "Próxima"
	}
	return RotuloMes(ym)
}

var mesesAbrev = []string{"jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"}

// RotuloMes escreve o mes como mmm/aa.
func RotuloMes(ym string) string {
	m, _ := strconv.Atoi(ym[5:7])
	return mesesAbrev[m-1] + "/" + ym[2:4]
}

// ---------- o que entra na fatura (R5 a R8) ----------

// EhCredito: o status e de credito pelo catalogo (compra de cartao).
func EhCredito(status string) bool { e, ok := EntradaDe(status); return ok && e.Credito }

// AjustePadrao e o deslocamento de fatura de um status: CREDITO EX empurra a compra para a fatura seguinte (+1) e CREDITO IN puxa uma
// compra posterior ao fechamento para a fatura que fecha neste mes (-1). Vem da entrada do catalogo.
func AjustePadrao(status string) int {
	if e, ok := EntradaDe(status); ok {
		return e.AjustePadrao
	}
	return 0
}

// AjusteEfetivo e o faturaAjuste gravado no lancamento, ou o padrao do status.
func AjusteEfetivo(l Lancamento) int {
	if n, ok := l["faturaAjuste"].(float64); ok {
		return int(n)
	}
	return AjustePadrao(StatusDe(l))
}

// ContaNaFatura diz se um lancamento entra na fatura do cartao: e do banco, de credito, nao esta excluido nem oculto e nao e de evento futuro.
func ContaNaFatura(l Lancamento, banco string) bool {
	if b, _ := l["banco"].(string); b != banco {
		return false
	}
	if Excluido(l) {
		return false
	}
	if o, _ := l["oculto"].(bool); o {
		return false
	}
	status, _ := l["status"].(string)
	return status != "" && EhCredito(status) && !EhPrevisto(l)
}

// RegistroFatura e uma compra do cartao pronta para a fatura.
type RegistroFatura struct {
	ID        string
	Descricao string
	Dia       string  // AAAA-MM-DD em Sao Paulo
	Valor     float64 // com sinal: saida +, entrada (estorno) -
	Base      string  // fatura pelo dia
	Ajuste    int
	Fatura    string // fatura efetiva (base deslocada pelo ajuste)
	Estimado  bool
	CriadoPor string
}

// RegistrosDoCartao monta as compras do cartao, na ordem de entrada (o chamador manda a mais recente primeiro, como a consulta antiga).
func RegistrosDoCartao(c Cartao, banco string, itens []Doc) []RegistroFatura {
	var rs []RegistroFatura
	for _, l := range itens {
		if !ContaNaFatura(l, banco) {
			continue
		}
		data, _ := l["dataEvento"].(string)
		dia, err := DiaBR(data)
		if err != nil {
			continue
		}
		valor, _ := l["valor"].(float64)
		if t, _ := l["tipo"].(string); t == "entrada" {
			valor = -valor
		}
		base := FaturaBase(c, dia)
		aj := AjusteEfetivo(l)
		r := RegistroFatura{Dia: dia, Valor: valor, Base: base, Ajuste: aj, Fatura: SomaMes(base, aj)}
		r.ID, _ = l["id"].(string)
		r.Descricao, _ = l["descricao"].(string)
		r.Estimado, _ = l["valorEstimado"].(bool)
		r.CriadoPor, _ = l["criadoPor"].(string)
		rs = append(rs, r)
	}
	return rs
}

// TotalFatura soma as compras cuja fatura efetiva e ym, arredondada a 2 casas.
func TotalFatura(rs []RegistroFatura, ym string) float64 {
	t := 0.0
	for _, r := range rs {
		if r.Fatura == ym {
			t += r.Valor
		}
	}
	return Arred(t)
}

// ---------- estado (R10) ----------

// FaturaFechada: o dia de hoje e posterior ao fechamento da fatura.
func FaturaFechada(c Cartao, ym, hoje string) bool { return hoje > FechamentoDe(c, ym) }

// FaturaHistorica: a fatura e anterior ao inicio do controle do cartao.
func FaturaHistorica(c Cartao, ym string) bool { return c.Inicio != "" && ym < c.Inicio }

// ---------- sugestoes (R16) ----------

// AcaoFatura e o faturaAjuste a gravar numa compra para a sugestao valer.
type AcaoFatura struct {
	ID     string
	Ajuste int
}

// SugestaoFatura e uma combinacao de 1 a 3 compras perto do corte que explica a diferenca, com a acao a aplicar em cada uma.
type SugestaoFatura struct {
	Registros []RegistroFatura
	Acoes     []AcaoFatura
	Rotulo    string
}

// Candidatos acha as compras perto do corte que explicam a diferenca d (total - valorBanco): ate 5 combinacoes de 1 a 3 compras, entre as
// 30 mais perto do corte (margem de 5 dias do inicio ou do fim do periodo), cuja soma em centavos e |d|. E o candidatosFatura do navegador,
// na mesma ordem.
func Candidatos(c Cartao, rs []RegistroFatura, ym string, d float64) []SugestaoFatura {
	const margem = 5
	ini, fim := PeriodoFatura(c, ym)
	dist := func(r RegistroFatura) int {
		a := abs(NumeroDoDia(r.Dia) - NumeroDoDia(ini))
		b := abs(NumeroDoDia(r.Dia) - NumeroDoDia(fim))
		if a < b {
			return a
		}
		return b
	}
	var pool []RegistroFatura
	for _, r := range rs {
		if d > 0 && r.Fatura == ym || d <= 0 && abs(DiferencaDeMeses(r.Fatura, ym)) == 1 {
			if r.Valor > 0 && dist(r) <= margem {
				pool = append(pool, r)
			}
		}
	}
	sort.SliceStable(pool, func(i, j int) bool { return dist(pool[i]) < dist(pool[j]) })
	if len(pool) > 30 {
		pool = pool[:30]
	}
	alvo := centavos(abs64(d))
	cents := func(r RegistroFatura) int64 { return centavos(r.Valor) }
	var achados [][]RegistroFatura
	n := len(pool)
	for i := 0; i < n; i++ {
		if cents(pool[i]) == alvo {
			achados = append(achados, []RegistroFatura{pool[i]})
		}
		for j := i + 1; j < n; j++ {
			if cents(pool[i])+cents(pool[j]) == alvo {
				achados = append(achados, []RegistroFatura{pool[i], pool[j]})
			}
			for k := j + 1; k < n; k++ {
				if cents(pool[i])+cents(pool[j])+cents(pool[k]) == alvo {
					achados = append(achados, []RegistroFatura{pool[i], pool[j], pool[k]})
				}
			}
		}
	}
	soma := func(a []RegistroFatura) int {
		s := 0
		for _, r := range a {
			s += dist(r)
		}
		return s
	}
	sort.SliceStable(achados, func(i, j int) bool {
		if len(achados[i]) != len(achados[j]) {
			return len(achados[i]) < len(achados[j])
		}
		return soma(achados[i]) < soma(achados[j])
	})
	var res []SugestaoFatura
	for _, arr := range achados {
		var acoes []AcaoFatura
		var rotulo string
		if d > 0 {
			todosProx, todosAnt := true, true
			for _, r := range arr {
				dir := -1
				if abs(NumeroDoDia(r.Dia)-NumeroDoDia(fim)) <= abs(NumeroDoDia(r.Dia)-NumeroDoDia(ini)) {
					dir = 1
				}
				if dir == 1 {
					todosAnt = false
				} else {
					todosProx = false
				}
				acoes = append(acoes, AcaoFatura{r.ID, DiferencaDeMeses(SomaMes(ym, dir), r.Base)})
			}
			switch {
			case todosProx:
				rotulo = "Passar para a próxima fatura"
			case todosAnt:
				rotulo = "Trazer para a anterior"
			default:
				rotulo = "Tirar desta fatura"
			}
		} else {
			for _, r := range arr {
				acoes = append(acoes, AcaoFatura{r.ID, DiferencaDeMeses(ym, r.Base)})
			}
			rotulo = "Trazer para esta fatura"
		}
		if foraDeUm(acoes) {
			continue
		}
		res = append(res, SugestaoFatura{Registros: arr, Acoes: acoes, Rotulo: rotulo})
		if len(res) >= 5 {
			break
		}
	}
	return res
}

func foraDeUm(acoes []AcaoFatura) bool {
	for _, a := range acoes {
		if abs(a.Ajuste) > 1 {
			return true
		}
	}
	return false
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// centavos arredonda como Math.round do navegador (a metade vai para cima).
func centavos(v float64) int64 { return int64(math.Floor(v*100 + 0.5)) }
