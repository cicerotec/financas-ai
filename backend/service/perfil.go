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

// Perfil e o nome de exibicao (apelido) de cada pessoa e o "desligar" do Telegram (o chat id vazio). Ficam no vinculo dela com o espaco, ao lado
// do papel, e cada pessoa so altera o proprio. O chat id so nasce pelo vinculo por link (service.Vinculos), nunca digitado.
type Perfil struct{ repo RepositorioPerfil }

func NewPerfil(repo RepositorioPerfil) *Perfil { return &Perfil{repo: repo} }

// Editar: PUT /spaces/{sid}/perfil. Altera o apelido e/ou apaga o telegramChatId (so vazio) de quem fez a chamada. Devolve os campos
// recebidos, ja normalizados. telegramChatId nao vazio e recusado.
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
		if chat != "" { // o Id nao se digita mais: o chat id e o do proprio Telegram, gravado so pelo vinculo por link (spec telegram-vinculo, R15)
			return nil, &ErroValidacao{"O Telegram agora é ligado pelo link, em Listas."}
		}
		m.ChatID = &chat // vazio e o desligar: apaga o destino
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
