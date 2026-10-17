package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"financas/domain"
)

// RepositorioPerfil e o que o service precisa do banco para o vinculo da pessoa com o espaco.
type RepositorioPerfil interface {
	// AtualizarPerfil altera o apelido e/ou o chat id; texto vazio remove. domain.ErrVinculoNaoEncontrado se nao ha vinculo.
	AtualizarPerfil(ctx context.Context, sid, sub string, m domain.MudancaPerfil) error
}

// Perfil e o nome de exibicao (apelido) e o chat id do Telegram de cada pessoa. Ficam no vinculo dela com o espaco, ao lado
// do papel, e cada pessoa so altera o proprio.
type Perfil struct{ repo RepositorioPerfil }

func NewPerfil(repo RepositorioPerfil) *Perfil { return &Perfil{repo: repo} }

// Editar: PUT /spaces/{sid}/perfil. Altera o apelido e/ou o telegramChatId de quem fez a chamada. Devolve os campos
// recebidos, ja normalizados.
func (s *Perfil) Editar(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error) {
	apelidoBruto, temApelido := body["apelido"]
	chatBruto, temChat := body["telegramChatId"]
	if !temApelido && !temChat {
		return nil, &ErroValidacao{"informe o apelido ou o telegramChatId"}
	}
	var m domain.MudancaPerfil
	resposta := domain.Doc{}
	if temApelido {
		bruto, ok := apelidoBruto.(string)
		if !ok {
			return nil, &ErroValidacao{"informe o apelido"}
		}
		nome, err := domain.NormalizarApelido(bruto)
		if err != nil {
			return nil, &ErroValidacao{err.Error()}
		}
		m.Apelido = &nome
		resposta["apelido"] = nome
	}
	if temChat {
		bruto, ok := chatBruto.(string)
		if !ok {
			return nil, &ErroValidacao{"informe o telegramChatId como texto"}
		}
		chat := strings.TrimSpace(bruto)
		if chat != "" && !domain.ChatIDValido(chat) {
			return nil, &ErroValidacao{"o chat id do Telegram tem so numeros (ex.: 123456789)"}
		}
		m.ChatID = &chat
		resposta["telegramChatId"] = chat
	}
	if err := s.repo.AtualizarPerfil(ctx, sid, sub, m); err != nil {
		if errors.Is(err, domain.ErrVinculoNaoEncontrado) {
			return nil, err
		}
		log.Println("put perfil:", err)
		return nil, ErrSalvar
	}
	return resposta, nil
}
