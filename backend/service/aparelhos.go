package service

import (
	"context"
	"errors"
	"log"

	"financas/domain"
)

// RepositorioAparelhos e o que o service precisa do banco para os aparelhos que recebem aviso (o push): um token por aparelho, no
// vinculo da pessoa com o espaco. So o repository sabe em que campo isso fica.
type RepositorioAparelhos interface {
	// RegistrarAparelho acrescenta o token (idempotente) e devolve quantos aparelhos a pessoa tem agora naquele espaco.
	// domain.ErrVinculoNaoEncontrado sem vinculo; domain.ErrLimiteAparelhos acima de domain.MaxAparelhos.
	RegistrarAparelho(ctx context.Context, sid, sub, token string) (int, error)
	// RemoverAparelho tira o token; devolve se estava la e quantos sobraram.
	RemoverAparelho(ctx context.Context, sid, sub, token string) (removido bool, restantes int, err error)
}

// Aparelhos registra e tira os aparelhos que recebem aviso pelo app. O sub sempre vem do token de quem chama, nunca do corpo: cada
// pessoa so mexe nos proprios aparelhos. Nenhum token vai ao log nem a resposta.
type Aparelhos struct {
	repo    RepositorioAparelhos
	espacos RepositorioEspacos
}

func NewAparelhos(repo RepositorioAparelhos, espacos RepositorioEspacos) *Aparelhos {
	return &Aparelhos{repo: repo, espacos: espacos}
}

func tokenDoCorpo(body domain.Doc) (string, error) {
	bruto, ok := body["token"].(string)
	if !ok || !domain.TokenAparelhoValido(bruto) {
		return "", &ErroValidacao{"informe o token do aparelho"}
	}
	return bruto, nil
}

// Registrar: PUT /spaces/{sid}/push. Devolve {"aparelhos": n}.
func (s *Aparelhos) Registrar(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error) {
	token, err := tokenDoCorpo(body)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.RegistrarAparelho(ctx, sid, sub, token)
	switch {
	case errors.Is(err, domain.ErrVinculoNaoEncontrado), errors.Is(err, domain.ErrLimiteAparelhos):
		return nil, err
	case err != nil:
		log.Println("aparelho: registrar:", err)
		return nil, ErrSalvar
	}
	return domain.Doc{"aparelhos": n}, nil
}

// Remover: POST /spaces/{sid}/push/remover. Tira o token deste espaco. Devolve {"aparelhos": n}.
func (s *Aparelhos) Remover(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error) {
	token, err := tokenDoCorpo(body)
	if err != nil {
		return nil, err
	}
	_, n, err := s.repo.RemoverAparelho(ctx, sid, sub, token)
	switch {
	case errors.Is(err, domain.ErrVinculoNaoEncontrado):
		return nil, err
	case err != nil:
		log.Println("aparelho: remover:", err)
		return nil, ErrSalvar
	}
	return domain.Doc{"aparelhos": n}, nil
}

// Sair: POST /push/sair. Tira o token de TODOS os espacos da pessoa (so dela). Falha parcial nao desfaz o resto: devolve em quantos
// espacos o token foi tirado, {"removidos": n}. So a falha de ler os espacos devolve erro.
func (s *Aparelhos) Sair(ctx context.Context, sub string, body domain.Doc) (domain.Doc, error) {
	token, err := tokenDoCorpo(body)
	if err != nil {
		return nil, err
	}
	espacos, err := s.espacos.EspacosDe(ctx, sub)
	if err != nil {
		log.Println("aparelho: sair: ler espacos:", err)
		return nil, ErrInterno
	}
	removidos, falhas := 0, 0
	for _, e := range espacos {
		removido, _, err := s.repo.RemoverAparelho(ctx, e.ID, sub, token)
		switch {
		case err != nil && !errors.Is(err, domain.ErrVinculoNaoEncontrado):
			falhas++
		case removido:
			removidos++
		}
	}
	if falhas > 0 {
		log.Printf("aparelho: sair: %d espaco(s) com falha", falhas)
	}
	return domain.Doc{"removidos": removidos}, nil
}
