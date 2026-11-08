package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"financas/domain"
)

// Encerrar um meio (item 58, regra R13): antes de desativar, o servidor confere, so lendo, se o meio ainda participa de algum evento. Cada pendencia e
// uma frase que a tela mostra; sem nenhuma, o meio pode ser encerrado.

// Pendencias devolve o que ainda impede de desativar o meio (vazio = pode).
type Pendencias interface {
	De(ctx context.Context, sid, meio string) ([]string, error)
}

// CalculadorDeSaldo e o que o encerramento precisa do service de saldos.
type CalculadorDeSaldo interface {
	Calcular(ctx context.Context, sid, banco string) (ResultadoBanco, error)
}

// LeitorDeCartoes e o que o encerramento precisa do service de cartoes: as tres ultimas faturas de cada cartao sem pagamento registrado (com ou sem vencimento).
type LeitorDeCartoes interface {
	EmAberto(ctx context.Context, sid string) ([]domain.Doc, error)
}

// LeitorDePonta le um lancamento pela data e id (a ponta de uma recorrencia).
type LeitorDePonta interface {
	Ler(ctx context.Context, sid, dataEvento, id string) (domain.Lancamento, error)
}

// ListadorDeRecorrencias lista as recorrencias ativas do espaco.
type ListadorDeRecorrencias interface {
	ListarRecorrencias(ctx context.Context, sid string) ([]domain.Doc, error)
}

type Encerramento struct {
	saldos  CalculadorDeSaldo
	lanc    LeitorDeLancamentos
	cfg     RepositorioConfig
	rec     ListadorDeRecorrencias
	pontas  LeitorDePonta
	cartoes LeitorDeCartoes
	rel     Relogio
}

func NewEncerramento(saldos CalculadorDeSaldo, lanc LeitorDeLancamentos, cfg RepositorioConfig, rec ListadorDeRecorrencias, pontas LeitorDePonta, cartoes LeitorDeCartoes, rel Relogio) *Encerramento {
	return &Encerramento{saldos: saldos, lanc: lanc, cfg: cfg, rec: rec, pontas: pontas, cartoes: cartoes, rel: rel}
}

func reais(v float64) string { return "R$ " + strings.Replace(fmt.Sprintf("%.2f", v), ".", ",", 1) }

func plural(n int, um, varios string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, um)
	}
	return fmt.Sprintf("%d %s", n, varios)
}

// De confere as cinco pendencias (saldo, lancamentos previstos ou futuros, recorrencia, fatura do cartao, pagador) e devolve as frases do que falta.
func (e *Encerramento) De(ctx context.Context, sid, meio string) ([]string, error) {
	var falta []string
	// (a) saldo diferente de zero, calculado pelo servidor (tambem sem inicio do controle e fora do total)
	calc, err := e.saldos.Calcular(ctx, sid, meio)
	if err != nil {
		return nil, err
	}
	if domain.Arred(calc.Saldo) != 0 {
		falta = append(falta, "O saldo é "+reais(calc.Saldo)+": só dá para desativar com o saldo zerado.")
	}
	// (b) lancamento previsto ou com data depois de agora (inclui a ponta futura de uma transferencia)
	agora := e.rel.Agora()
	prev, fut := 0, 0
	f := domain.FiltroTx{Banco: meio, Limite: 1000}
	for {
		itens, proximo, err := e.lanc.Listar(ctx, sid, f)
		if err != nil {
			log.Println("encerramento: listar:", err)
			return nil, ErrListar
		}
		for _, l := range itens {
			data, _ := l["dataEvento"].(string)
			switch {
			case domain.EhPrevisto(l):
				prev++
			case domain.PrevistoPorData(data, agora):
				fut++
			}
		}
		if proximo == "" {
			break
		}
		f.Cursor = proximo
	}
	if prev > 0 {
		falta = append(falta, plural(prev, "lançamento previsto", "lançamentos previstos")+" neste meio: confirme ou exclua em Futuros.")
	}
	if fut > 0 {
		falta = append(falta, plural(fut, "lançamento", "lançamentos")+" com data futura neste meio.")
	}
	// (c) recorrencia ativa cuja ultima copia (a ponta) e deste meio: a ponta pode ja ter vencido e a proxima copia ainda nao existir
	itens, err := e.rec.ListarRecorrencias(ctx, sid)
	if err != nil {
		log.Println("encerramento: recorrencias:", err)
		return nil, ErrListar
	}
	recs := 0
	for _, it := range itens {
		id, _ := it["id"].(string)
		ultimo, _ := it["ultimoId"].(string)
		data, _ := it["dataUltimo"].(string)
		if len(id) <= len(prefixoItemRecorrencia) || ultimo == "" {
			continue
		}
		ponta, err := e.pontas.Ler(ctx, sid, data, ultimo)
		if err != nil {
			log.Println("encerramento: ler a ponta:", err)
			return nil, ErrInterno
		}
		if ponta != nil && bancoDe(ponta) == meio { // a ponta excluida tambem conta: a rotina continua copiando dela
			recs++
		}
	}
	if recs > 0 {
		falta = append(falta, plural(recs, "recorrência ativa", "recorrências ativas")+" neste meio: pare a recorrência antes.")
	}
	// (d) o meio e um cartao: fatura atual com total maior que zero ou fatura a pagar em Futuros
	cfgC, err := e.cfg.Ler(ctx, sid, "cartoes")
	if err != nil {
		log.Println("encerramento: cartoes:", err)
		return nil, ErrInterno
	}
	// as tres ultimas faturas de cada cartao com valor e sem pagamento registrado: nao depende de vencimento nem de Futuros
	faturas, err := e.cartoes.EmAberto(ctx, sid)
	if err != nil {
		return nil, err
	}
	if _, ehCartao := domain.CartaoDe(cfgC, meio); ehCartao {
		for _, fa := range faturas {
			if fa["meio"] == meio {
				rot, _ := fa["rotulo"].(string)
				valor, _ := fa["valor"].(float64)
				falta = append(falta, "A fatura de "+rot+" ("+reais(valor)+") não tem pagamento registrado: registre que pagou para desativar o cartão.")
			}
		}
	}
	// (e) pagador: o padrao de algum cartao, ou o de uma fatura ainda a pagar
	por, _ := cfgC["porBanco"].(map[string]any)
	nomes := make([]string, 0, len(por))
	for nome := range por {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)
	for _, nome := range nomes {
		if c, ok := domain.CartaoDe(cfgC, nome); ok && c.Pagador == meio {
			falta = append(falta, "É o pagador padrão do cartão "+nome+": troque o pagador na aba Cartão.")
		}
	}
	for _, fa := range faturas {
		if fa["pagador"] == meio {
			c, _ := fa["meio"].(string)
			rot, _ := fa["rotulo"].(string)
			falta = append(falta, "É o pagador da fatura de "+rot+" do cartão "+c+": troque o pagador na aba Cartão.")
		}
	}
	return falta, nil
}
