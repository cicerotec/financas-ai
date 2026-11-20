package service

import (
	"sync"
	"time"
)

// Limites da recuperacao do MFA (spec mfa-opcional, R29). Sao EM MEMORIA da Lambda, sem DynamoDB: a rota publica nao pode gastar a capacidade
// da tabela (5 RCU/WCU) com chamadas sem token. Sao de melhor esforco: zeram a cada partida a frio de cada instancia; o teto real de
// concorrencia e o da Lambda.
const (
	MaxFalhasPar       = 10        // falhas do mesmo par (pessoa, origem) na janela
	JanelaPar          = time.Hour // janela de contagem das falhas
	BloqueioPar        = time.Hour // quanto o par fica bloqueado depois de estourar
	MaxPorMinutoOrigem = 30        // chamadas por minuto de uma mesma origem
	tetoEntradas       = 10000     // maximo de chaves guardadas de cada tipo (evita crescer sem limite)
)

type falhasDoPar struct {
	n            int
	desde        time.Time
	bloqueadoAte time.Time
}

type janelaDaOrigem struct {
	minuto time.Time
	n      int
}

// LimitadorRecuperacao conta falhas por par (pessoa, origem) e chamadas por origem. Seguro para uso concorrente.
type LimitadorRecuperacao struct {
	mu      sync.Mutex
	pares   map[string]*falhasDoPar
	origens map[string]*janelaDaOrigem
}

func NewLimitadorRecuperacao() *LimitadorRecuperacao {
	return &LimitadorRecuperacao{pares: map[string]*falhasDoPar{}, origens: map[string]*janelaDaOrigem{}}
}

// PermiteOrigem conta uma chamada da origem e diz se ainda cabe no minuto (30 por minuto).
func (l *LimitadorRecuperacao) PermiteOrigem(origem string, agora time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	min := agora.Truncate(time.Minute)
	j := l.origens[origem]
	if j == nil {
		if len(l.origens) >= tetoEntradas {
			l.limparOrigens(min)
		}
		j = &janelaDaOrigem{minuto: min}
		l.origens[origem] = j
	}
	if !j.minuto.Equal(min) {
		j.minuto, j.n = min, 0
	}
	j.n++
	return j.n <= MaxPorMinutoOrigem
}

func (l *LimitadorRecuperacao) limparOrigens(min time.Time) {
	for k, j := range l.origens {
		if j.minuto.Before(min) {
			delete(l.origens, k)
		}
	}
	if len(l.origens) >= tetoEntradas { // ainda cheio de minutos correntes: recomeca (melhor esforco)
		l.origens = map[string]*janelaDaOrigem{}
	}
}

// ParBloqueado diz se o par esta bloqueado agora.
func (l *LimitadorRecuperacao) ParBloqueado(par string, agora time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	p := l.pares[par]
	return p != nil && agora.Before(p.bloqueadoAte)
}

// RegistrarFalhaPar conta uma falha do par; a 10a na janela de 1 hora bloqueia o par por 1 hora.
func (l *LimitadorRecuperacao) RegistrarFalhaPar(par string, agora time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	p := l.pares[par]
	if p == nil {
		if len(l.pares) >= tetoEntradas {
			l.limparPares(agora)
		}
		p = &falhasDoPar{desde: agora}
		l.pares[par] = p
	}
	if agora.Sub(p.desde) > JanelaPar && !agora.Before(p.bloqueadoAte) {
		p.n, p.desde = 0, agora
	}
	p.n++
	if p.n >= MaxFalhasPar {
		p.bloqueadoAte = agora.Add(BloqueioPar)
		p.n, p.desde = 0, agora
	}
}

func (l *LimitadorRecuperacao) limparPares(agora time.Time) {
	for k, p := range l.pares {
		if agora.Sub(p.desde) > JanelaPar && !agora.Before(p.bloqueadoAte) {
			delete(l.pares, k)
		}
	}
	if len(l.pares) >= tetoEntradas {
		l.pares = map[string]*falhasDoPar{}
	}
}

// ZerarPar apaga o contador do par (depois de um sucesso).
func (l *LimitadorRecuperacao) ZerarPar(par string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.pares, par)
}
