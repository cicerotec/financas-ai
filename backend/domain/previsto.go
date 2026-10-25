package domain

import "time"

// Previsto (item 41, .ai/specs/futuros-previsto): um lancamento que ainda nao aconteceu. E o marcador previsto (booleano, gravado so quando ligado,
// so pelo servidor) ou, nos lancamentos antigos, um dos nomes de StatusAntigosPrevistos. Tudo aqui e puro: o instante vem de quem chama.

// EhPrevisto diz se o lancamento e previsto (R1).
func EhPrevisto(l Lancamento) bool {
	if p, _ := l["previsto"].(bool); p {
		return true
	}
	s := StatusDe(l)
	for _, antigo := range StatusAntigosPrevistos {
		if s == antigo {
			return true
		}
	}
	return false
}

// PrevistoPorData: o dia da dataEvento (Sao Paulo) e posterior ao dia de agora. Hoje nao e futuro (R2). Data invalida nao e futura.
func PrevistoPorData(dataEvento string, agora time.Time) bool {
	dia, err := DiaBR(dataEvento)
	if err != nil {
		return false
	}
	return dia > agora.In(FusoBR).Format("2006-01-02")
}

// Atrasado: previsto cujo dia e anterior a hoje (R3).
func Atrasado(l Lancamento, agora time.Time) bool {
	data, _ := l["dataEvento"].(string)
	dia, err := DiaBR(data)
	return err == nil && EhPrevisto(l) && dia < agora.In(FusoBR).Format("2006-01-02")
}

// Efeito e o que um lancamento faz no saldo: o meio e o mes e a contribuicao com sinal. Existe=false quando nao faz nada.
type Efeito struct {
	Alvo   Alvo
	Valor  float64
	Existe bool
}

// EfeitoSaldo calcula o efeito de um lancamento no saldo (R15): nenhum se o status nao conta no saldo, se esta oculto, excluido ou previsto.
func EfeitoSaldo(l Lancamento) Efeito {
	if Excluido(l) || EhPrevisto(l) || !ContaNoSaldoStatus(StatusDe(l)) {
		return Efeito{}
	}
	if o, _ := l["oculto"].(bool); o {
		return Efeito{}
	}
	al, ok := AlvoDe(l)
	if !ok {
		return Efeito{}
	}
	v, _ := l["valor"].(float64)
	if t, _ := l["tipo"].(string); t != "entrada" {
		v = -v
	}
	return Efeito{Alvo: al, Valor: v, Existe: true}
}

// MudaSaldo diz se a edicao de antigo para novo muda o efeito no saldo. Trocar PAGO por um status que tambem conta nao muda.
func MudaSaldo(antigo, novo Doc) bool { return EfeitoSaldo(antigo) != EfeitoSaldo(novo) }

// AlvosDoSaldo devolve os alvos (meio e mes), sem repetir, dos efeitos nao nulos de dois estados do lancamento: os que as travas precisam conferir.
func AlvosDoSaldo(a, b Lancamento) []Alvo {
	var alvos []Alvo
	for _, l := range []Lancamento{a, b} {
		e := EfeitoSaldo(l)
		if !e.Existe {
			continue
		}
		repetido := false
		for _, x := range alvos {
			repetido = repetido || x == e.Alvo
		}
		if !repetido {
			alvos = append(alvos, e.Alvo)
		}
	}
	return alvos
}
