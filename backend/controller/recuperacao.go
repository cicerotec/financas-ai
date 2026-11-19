package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"financas/domain"
	"financas/service"
)

// limiteCorpoRecuperacao e o maior corpo que a rota publica le (spec mfa-opcional, R28): acima disso a resposta e a de sempre, sem trabalho.
const limiteCorpoRecuperacao = 2048

// Recuperador e o que o controller precisa do service da recuperacao (declarado aqui para os testes trocarem por um falso).
type Recuperador interface {
	Recuperar(ctx context.Context, origem, email, codigo string) error
}

// Recuperacao liga a rota publica da recuperacao do MFA ao service. Ela chega aqui SEM token (auth.RotaPublica) e responde
// uma unica falha: o mesmo status e o mesmo corpo para qualquer motivo (spec mfa-opcional, R28).
type Recuperacao struct{ svc Recuperador }

func NewRecuperacao(svc Recuperador) *Recuperacao { return &Recuperacao{svc: svc} }

// Post: POST /mfa/recuperar com {"email","codigo"}. origem e o SourceIP do evento. Sucesso: 200 {"ok":true}; qualquer falha: 400 {"erro":"recuperacao_invalida"}.
func (c *Recuperacao) Post(ctx context.Context, origem, corpo string) (Resp, error) {
	falha := func() (Resp, error) { return Erro(400, "recuperacao_invalida") }
	// entrada minima, antes de qualquer trabalho: corpo vazio ou acima de 2 KiB, JSON invalido ou campo faltando
	if corpo == "" || len(corpo) > limiteCorpoRecuperacao {
		return falha()
	}
	var in struct {
		Email  string `json:"email"`
		Codigo string `json:"codigo"`
	}
	if json.Unmarshal([]byte(corpo), &in) != nil || strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Codigo) == "" {
		return falha()
	}
	if err := c.svc.Recuperar(ctx, origem, in.Email, in.Codigo); err != nil {
		if !errors.Is(err, service.ErrRecuperacaoInvalida) {
			log.Println("recuperacao: falha interna") // so o fato: nunca o e-mail, o codigo nem o corpo
		}
		return falha()
	}
	return Out(200, domain.Doc{"ok": true})
}
