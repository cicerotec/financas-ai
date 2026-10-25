package domain

import "errors"

// Pagar a fatura do cartao (itens 33b e 35, .ai/specs/pagar-fatura). O pagamento e um lancamento normal; a marca pagaFatura e so uma label (que cartao e que
// mes ele pagou), escolhida pelo dono. Funcoes puras: a configuracao do cartao vem de quem chama.

// VencimentoDe e o vencimento da fatura ym: o escolhido para a fatura; senao o primeiro dia depois do fechamento cujo dia do mes e o diaVencimento do cartao
// (limitado ao ultimo dia do mes); sem nenhum dos dois, "" (nao ha vencimento).
func VencimentoDe(c Cartao, ym string) string {
	if v := c.Faturas[ym].Vencimento; v != "" {
		return v
	}
	if c.DiaVencimento < 1 {
		return ""
	}
	fech := FechamentoDe(c, ym)
	mes := fech[:7]
	for i := 0; i < 2; i++ {
		dia := c.DiaVencimento
		if max := DiasNoMes(mes); dia > max {
			dia = max
		}
		cand := mes + "-" + pad2(dia)
		if cand > fech {
			return cand
		}
		mes = SomaMes(mes, 1)
	}
	return ""
}

// PagadorDe e o pagador sugerido da fatura ym: o da fatura, senao o do cartao, senao "".
func PagadorDe(c Cartao, ym string) string {
	if p := c.Faturas[ym].Pagador; p != "" {
		return p
	}
	return c.Pagador
}

// ValidarPagaFatura confere a marca pagaFatura de um lancamento: { meio, ym } com um cartao de verdade e um mes valido, e o lancamento nao pode ser compra de
// credito nem estar no banco de um cartao. null vale como ausente (a marca sai). cartoes e o documento CFG#CARTOES.
func ValidarPagaFatura(d Doc, cartoes Doc) error {
	v, tem := d["pagaFatura"]
	if !tem {
		return nil
	}
	if v == nil {
		delete(d, "pagaFatura")
		return nil
	}
	m, ok := v.(map[string]any)
	meio, _ := m["meio"].(string)
	ym, _ := m["ym"].(string)
	if !ok || len(m) != 2 || meio == "" || !MesValido(ym) {
		return errors.New("pagaFatura invalido: informe { meio, ym } com o cartao e o mes AAAA-MM")
	}
	if _, cartao := CartaoDe(cartoes, meio); !cartao {
		return errors.New("pagaFatura invalido: o meio precisa ser um cartao")
	}
	if s, _ := d["status"].(string); EhCredito(s) {
		return errors.New("pagaFatura invalido: uma compra de credito nao paga fatura")
	}
	if b, _ := d["banco"].(string); b != "" {
		if _, cartao := CartaoDe(cartoes, b); cartao {
			return errors.New("pagaFatura invalido: um lancamento do proprio cartao nao paga fatura")
		}
	}
	return nil
}

// TemPagaFatura diz se o lancamento leva a marca (bem formada ou nao).
func TemPagaFatura(l Lancamento) bool { return l["pagaFatura"] != nil }

// MarcaDe devolve o cartao e o mes da marca pagaFatura; ok=false se nao ha marca bem formada.
func MarcaDe(l Lancamento) (meio, ym string, ok bool) {
	m, _ := l["pagaFatura"].(map[string]any)
	meio, _ = m["meio"].(string)
	ym, _ = m["ym"].(string)
	return meio, ym, meio != "" && ym != ""
}

// EfeitoDeFatura e o que um lancamento faz numa fatura: o cartao, a fatura (AAAA-MM), o valor com sinal. Existe=false quando nao entra em fatura nenhuma.
type EfeitoDeFatura struct {
	Meio, Ym string
	Valor    float64
	Existe   bool
}

// EfeitoFatura calcula o efeito do lancamento na fatura do seu cartao (R5 a R8 do item 33a): so uma compra de credito do banco de um cartao, nao excluida, nao oculta
// e nao prevista, na fatura em que cai (a base pelo dia, deslocada pelo ajuste).
func EfeitoFatura(l Lancamento, cartoes Doc) EfeitoDeFatura {
	banco, _ := l["banco"].(string)
	c, ok := CartaoDe(cartoes, banco)
	if !ok || !ContaNaFatura(l, banco) {
		return EfeitoDeFatura{}
	}
	data, _ := l["dataEvento"].(string)
	dia, err := DiaBR(data)
	if err != nil {
		return EfeitoDeFatura{}
	}
	valor, _ := l["valor"].(float64)
	if t, _ := l["tipo"].(string); t == "entrada" {
		valor = -valor
	}
	return EfeitoDeFatura{Meio: banco, Ym: SomaMes(FaturaBase(c, dia), AjusteEfetivo(l)), Valor: valor, Existe: true}
}

// MudaFatura diz se a escrita de antes para depois muda o que a fatura de algum cartao soma.
func MudaFatura(antes, depois Lancamento, cartoes Doc) bool {
	return EfeitoFatura(antes, cartoes) != EfeitoFatura(depois, cartoes)
}

// FaturasAfetadas devolve as faturas (cartao e mes), sem repetir, dos efeitos nao nulos de antes e de depois: as que a trava de fatura conferida precisa olhar.
func FaturasAfetadas(antes, depois Lancamento, cartoes Doc) []EfeitoDeFatura {
	var out []EfeitoDeFatura
	for _, l := range []Lancamento{antes, depois} {
		if l == nil {
			continue
		}
		e := EfeitoFatura(l, cartoes)
		if !e.Existe {
			continue
		}
		repetida := false
		for _, x := range out {
			repetida = repetida || (x.Meio == e.Meio && x.Ym == e.Ym)
		}
		if !repetida {
			out = append(out, e)
		}
	}
	return out
}

// FaturaConferida diz se a fatura ym do cartao esta conferida.
func FaturaConferida(c Cartao, ym string) bool { return c.Faturas[ym].Conferida != nil }

// ErroFaturaConferida: a fatura do cartao foi conferida e esta travada. Nao ha confirmacao que a destrave: so desfazer a conferencia.
type ErroFaturaConferida struct{ Meio, Ym string }

func (e *ErroFaturaConferida) Error() string {
	return "a fatura " + RotuloMes(e.Ym) + " do cartao " + e.Meio + " esta conferida e travada: desfaca a conferencia na aba Cartao para alterar"
}
