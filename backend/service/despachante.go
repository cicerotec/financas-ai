package service

import (
	"context"
	"errors"
	"fmt"

	"financas/domain"
)

// Canal entrega um resumo de avisos a um destino do seu tipo (o chat id do Telegram, hoje). Cada canal e um adaptador em
// adapter/notificacao/ que escolhe o formato do texto; domain.ErrCanalNaoConfigurado diz que ele nao tem o que precisa
// para enviar. Nao registra no log nem devolve em erro o destino, o texto ou token.
type Canal interface {
	Nome() string
	Enviar(ctx context.Context, destino string, r domain.ResumoAvisos) error
}

// Destino e para onde um resumo vai: o canal e o endereco dentro dele.
type Destino struct{ Canal, Endereco string }

// ErrCanalDesconhecido: o destino pede um canal que nao foi registrado (erro de configuracao, nunca panico).
var ErrCanalDesconhecido = errors.New("canal de aviso desconhecido")

// Despachante guarda os canais por nome e entrega um resumo ao destino certo.
type Despachante struct{ canais map[string]Canal }

// NovoDespachante registra os canais; nome repetido (ou vazio) e erro de montagem.
func NovoDespachante(canais ...Canal) (*Despachante, error) {
	d := &Despachante{canais: map[string]Canal{}}
	for _, c := range canais {
		n := c.Nome()
		if n == "" {
			return nil, errors.New("canal de aviso sem nome")
		}
		if _, dup := d.canais[n]; dup {
			return nil, fmt.Errorf("canal de aviso repetido: %s", n)
		}
		d.canais[n] = c
	}
	return d, nil
}

// Entregar manda o resumo ao destino pelo canal dele.
func (d *Despachante) Entregar(ctx context.Context, dest Destino, r domain.ResumoAvisos) error {
	c, ok := d.canais[dest.Canal]
	if !ok {
		return ErrCanalDesconhecido
	}
	return c.Enviar(ctx, dest.Endereco, r)
}
