// Package relogio tem os adaptadores do relogio: o service declara o que precisa (service.Relogio) e quem monta o sistema
// escolhe qual usar. Nao importa nada do projeto.
package relogio

import (
	"sync"
	"time"
)

// Real e o relogio do sistema.
type Real struct{}

func (Real) Agora() time.Time { return time.Now() }

// Fixo e o relogio dos testes: devolve o instante que lhe deram, ate alguem trocar.
type Fixo struct {
	mu sync.Mutex
	em time.Time
}

func NewFixo(em time.Time) *Fixo { return &Fixo{em: em} }

func (f *Fixo) Agora() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.em
}

// Definir troca o instante e devolve o anterior (para o teste restaurar no fim).
func (f *Fixo) Definir(em time.Time) time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	antes := f.em
	f.em = em
	return antes
}
