package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"

	"financas/domain"
)

// Consultas salvas da Tendencias (item 53, .ai/specs/tendencias-consultas). Compartilhadas no espaco: todos leem, criam e usam; cada
// member altera so as suas (criadoPor == sub) e o owner altera todas. O criadoPor vem sempre do token. A exclusao e logica.

// RepositorioConsultas e o que as consultas precisam do banco: um item por consulta, com controle otimista por version.
type RepositorioConsultas interface {
	Listar(ctx context.Context, sid string) ([]domain.Doc, error) // todas, inclusive as excluidas
	Ler(ctx context.Context, sid, id string) (domain.Doc, error)  // nil se nao existe
	Salvar(ctx context.Context, sid, id string, body domain.Doc, esperada float64) (domain.Doc, error)
}

var (
	ErrConsultaNaoEncontrada = errors.New("consulta nao encontrada")
	ErrSoDonoConsulta        = errors.New("so e possivel alterar as consultas criadas por voce")
)

const autorDesconhecido = "alguem do espaco"

type Consultas struct {
	repo    RepositorioConsultas
	pessoas Pessoas
	rel     Relogio
}

func NewConsultas(repo RepositorioConsultas, pessoas Pessoas, rel Relogio) *Consultas {
	return &Consultas{repo: repo, pessoas: pessoas, rel: rel}
}

func (s *Consultas) agora() string { return s.rel.Agora().UTC().Format("2006-01-02T15:04:05.000Z") }

func excluida(d domain.Doc) bool { return d["excluidoEm"] != nil }

// vivas separa as consultas que nao foram excluidas.
func vivas(todas []domain.Doc) []domain.Doc {
	v := make([]domain.Doc, 0, len(todas))
	for _, d := range todas {
		if !excluida(d) {
			v = append(v, d)
		}
	}
	return v
}

// nomeEmUso diz se outra consulta viva (que nao seja a de id "exceto") ja tem esse nome.
func nomeEmUso(vivas []domain.Doc, nome, exceto string) bool {
	for _, d := range vivas {
		if id, _ := d["id"].(string); id == exceto {
			continue
		}
		if n, _ := d["nome"].(string); domain.MesmoNomeConsulta(n, nome) {
			return true
		}
	}
	return false
}

func recusarLimite(vivas []domain.Doc) error {
	if len(vivas) >= domain.MaxConsultas {
		return &ErroValidacao{Msg: fmt.Sprintf("limite de %d consultas: exclua uma antes de salvar outra", domain.MaxConsultas)}
	}
	return nil
}

// visao monta a consulta como o cliente a ve: sem chaves internas, sem o sub (so "minha" e "podeAlterar", ja decididos aqui) e com o
// apelido atual do autor. O front so exibe.
func (s *Consultas) visao(ctx context.Context, sid string, d domain.Doc, sub string, owner bool, apelidos map[string]string) domain.Doc {
	criador, _ := d["criadoPor"].(string)
	autor, ok := apelidos[criador]
	if !ok {
		autor = s.pessoas.Apelido(ctx, sid, criador)
		if autor == "" {
			autor = autorDesconhecido
		}
		apelidos[criador] = autor
	}
	minha := domain.EhDono(d, sub)
	v := domain.Doc{"autor": autor, "minha": minha, "podeAlterar": owner || minha}
	for _, k := range []string{"id", "nome", "modo", "grupos", "expressao", "visao", "criadoEm", "atualizadoEm", "version"} {
		if x, tem := d[k]; tem {
			v[k] = x
		}
	}
	return v
}

// Listar devolve as consultas vivas do espaco, da mais nova para a mais antiga.
func (s *Consultas) Listar(ctx context.Context, sid, sub string, owner bool) ([]domain.Doc, error) {
	todas, err := s.repo.Listar(ctx, sid)
	if err != nil {
		log.Println("consultas: listar:", err)
		return nil, ErrListar
	}
	v := vivas(todas)
	sort.Slice(v, func(i, j int) bool {
		a, _ := v[i]["criadoEm"].(string)
		b, _ := v[j]["criadoEm"].(string)
		if a != b {
			return a > b
		}
		x, _ := v[i]["id"].(string)
		y, _ := v[j]["id"].(string)
		return x > y
	})
	apelidos := map[string]string{}
	out := make([]domain.Doc, 0, len(v))
	for _, d := range v {
		out = append(out, s.visao(ctx, sid, d, sub, owner, apelidos))
	}
	return out, nil
}

// Criar grava uma consulta nova. O sub do token e o criadoPor; o corpo nunca o define.
func (s *Consultas) Criar(ctx context.Context, sid, sub string, owner bool, body domain.Doc) (domain.Doc, error) {
	c, err := domain.ValidarCorpoConsulta(body, true)
	if err != nil {
		return nil, &ErroValidacao{Msg: err.Error()}
	}
	todas, err := s.repo.Listar(ctx, sid)
	if err != nil {
		log.Println("consultas: criar: listar:", err)
		return nil, ErrListar
	}
	v := vivas(todas)
	if err := recusarLimite(v); err != nil {
		return nil, err
	}
	if nomeEmUso(v, c.Nome, "") {
		return nil, &ErroValidacao{Msg: "ja existe uma consulta com esse nome"}
	}
	id, agora := domain.NovoID(), s.agora()
	item := domain.Doc{"id": id, "nome": c.Nome, "criadoPor": sub, "criadoEm": agora, "atualizadoEm": agora, "atualizadoPor": sub}
	for k, x := range c.Consulta {
		item[k] = x
	}
	gravado, err := s.repo.Salvar(ctx, sid, id, item, 0)
	if err != nil {
		return nil, s.erroDeGravar(err)
	}
	return s.visao(ctx, sid, gravado, sub, owner, map[string]string{}), nil
}

// carregar le a consulta e confere a permissao: 404 se nao existe, 403 se o member nao e o autor.
func (s *Consultas) carregar(ctx context.Context, sid, id, sub string, restrito bool) (domain.Doc, error) {
	d, err := s.repo.Ler(ctx, sid, id)
	if err != nil {
		log.Println("consultas: ler:", err)
		return nil, ErrInterno
	}
	if d == nil {
		return nil, ErrConsultaNaoEncontrada
	}
	if restrito && !domain.EhDono(d, sub) {
		return nil, ErrSoDonoConsulta
	}
	return d, nil
}

func (s *Consultas) erroDeGravar(err error) error {
	if errors.Is(err, domain.ErrConflitoConfig) {
		return domain.ErrConflitoVersao // 409: alguem gravou antes, releia
	}
	log.Println("consultas: gravar:", err)
	return ErrSalvar
}

// Atualizar renomeia e/ou troca a consulta, com a version lida. restrito = member: so a propria. O criadoPor nao muda.
func (s *Consultas) Atualizar(ctx context.Context, sid, id, sub string, restrito bool, body domain.Doc) (domain.Doc, error) {
	c, err := domain.ValidarCorpoConsulta(body, false)
	if err != nil {
		return nil, &ErroValidacao{Msg: err.Error()}
	}
	d, err := s.carregar(ctx, sid, id, sub, restrito)
	if err != nil {
		return nil, err
	}
	if excluida(d) {
		return nil, ErrConsultaNaoEncontrada
	}
	if c.Nome != "" {
		todas, err := s.repo.Listar(ctx, sid)
		if err != nil {
			log.Println("consultas: atualizar: listar:", err)
			return nil, ErrListar
		}
		if nomeEmUso(vivas(todas), c.Nome, id) {
			return nil, &ErroValidacao{Msg: "ja existe uma consulta com esse nome"}
		}
		d["nome"] = c.Nome
	}
	if c.Consulta != nil {
		for _, k := range []string{"modo", "grupos", "expressao", "visao"} {
			delete(d, k)
		}
		for k, x := range c.Consulta {
			d[k] = x
		}
	}
	d["atualizadoEm"], d["atualizadoPor"] = s.agora(), sub
	gravado, err := s.repo.Salvar(ctx, sid, id, d, c.Versao)
	if err != nil {
		return nil, s.erroDeGravar(err)
	}
	return s.visao(ctx, sid, gravado, sub, !restrito, map[string]string{}), nil
}

// Excluir marca excluidoEm (exclusao logica). Excluir o que ja esta excluido nao faz nada.
func (s *Consultas) Excluir(ctx context.Context, sid, id, sub string, restrito bool) error {
	d, err := s.carregar(ctx, sid, id, sub, restrito)
	if err != nil {
		return err
	}
	if excluida(d) {
		return nil
	}
	versao, _ := d["version"].(float64)
	agora := s.agora()
	d["excluidoEm"], d["atualizadoEm"], d["atualizadoPor"] = agora, agora, sub
	if _, err := s.repo.Salvar(ctx, sid, id, d, versao); err != nil {
		return s.erroDeGravar(err)
	}
	return nil
}

// Restaurar desfaz a exclusao do mesmo item, com o mesmo autor. Recusa se o nome foi tomado ou o limite foi atingido nesse meio tempo.
func (s *Consultas) Restaurar(ctx context.Context, sid, id, sub string, restrito bool) (domain.Doc, error) {
	d, err := s.carregar(ctx, sid, id, sub, restrito)
	if err != nil {
		return nil, err
	}
	if excluida(d) {
		todas, err := s.repo.Listar(ctx, sid)
		if err != nil {
			log.Println("consultas: restaurar: listar:", err)
			return nil, ErrListar
		}
		v := vivas(todas)
		if err := recusarLimite(v); err != nil {
			return nil, err
		}
		nome, _ := d["nome"].(string)
		if nomeEmUso(v, nome, id) {
			return nil, &ErroValidacao{Msg: "ja existe uma consulta com esse nome: renomeie a outra antes de restaurar"}
		}
		versao, _ := d["version"].(float64)
		delete(d, "excluidoEm")
		d["atualizadoEm"], d["atualizadoPor"] = s.agora(), sub
		if d, err = s.repo.Salvar(ctx, sid, id, d, versao); err != nil {
			return nil, s.erroDeGravar(err)
		}
	}
	return s.visao(ctx, sid, d, sub, !restrito, map[string]string{}), nil
}
