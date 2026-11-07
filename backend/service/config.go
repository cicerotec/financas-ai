package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"financas/domain"
)

// RepositorioConfig e o que o service precisa do banco para as configuracoes do espaco.
type RepositorioConfig interface {
	// Ler devolve a configuracao compartilhada, ou nil se ainda nao existe.
	Ler(ctx context.Context, sid, nome string) (domain.Doc, error)
	// Salvar grava com controle otimista (esperada = version lida). domain.ErrConflitoConfig se alguem gravou antes.
	Salvar(ctx context.Context, sid, nome string, body domain.Doc, esperada float64) (domain.Doc, error)
	LerAparencia(ctx context.Context, sid, sub string) (domain.Doc, error)
	GravarAparencia(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error)
}

// ErroConflitoConfig e o 409 da configuracao: outra pessoa gravou antes; Atual e o estado que ficou no banco.
type ErroConflitoConfig struct{ Atual domain.Doc }

func (e *ErroConflitoConfig) Error() string { return domain.ErrConflitoConfig.Error() }

// Config le e grava as configuracoes compartilhadas (listas, tags, combos, saldos, cartoes) e a aparencia de cada pessoa.
type Config struct {
	repo   RepositorioConfig
	rel    Relogio
	rotina LeitorRotinaCombos
}

// LeitorRotinaCombos diz se a rotina semanal dos combos sugeridos esta ligada no espaco (item 57); quem implementa e o service de combos.
type LeitorRotinaCombos interface {
	RotinaAtiva(ctx context.Context, sid string) (bool, error)
}

// UsarRotinaCombos liga a leitura do interruptor da rotina (montada depois, porque o service de combos usa a configuracao).
func (s *Config) UsarRotinaCombos(r LeitorRotinaCombos) { s.rotina = r }

func NewConfig(repo RepositorioConfig, rel Relogio) *Config { return &Config{repo: repo, rel: rel} }

// semItem e o que se devolve quando a configuracao ainda nao existe.
func semItem(d domain.Doc) domain.Doc {
	if d == nil {
		return domain.Doc{"version": 0}
	}
	return domain.Publico(d)
}

// statusDeListas sao os campos de CFG#LISTAS que o catalogo de status do servidor substituiu (item 41).
var statusDeListas = []string{"status", "statusReal", "afetaSaldoBanco", "statusFuturo"}

// Ler devolve a configuracao compartilhada; sem item, so {"version": 0}.
func (s *Config) Ler(ctx context.Context, sid, nome string) (domain.Doc, error) {
	d, err := s.repo.Ler(ctx, sid, nome)
	if err != nil {
		log.Println("cfg:", err)
		return nil, ErrInterno
	}
	out := semItem(d)
	if strings.EqualFold(nome, "combos") && s.rotina != nil {
		// o interruptor da rotina vem do item auxiliar dos avisos, nunca do CFG#COMBOS
		ativa, err := s.rotina.RotinaAtiva(ctx, sid)
		if err != nil {
			return nil, err
		}
		out["rotinaAtiva"] = ativa
	}
	if strings.EqualFold(nome, "listas") {
		// a lista de status e do servidor (item 41): so as selecionaveis sugeridas hoje; as caixas de status deixam de ser lidas e de sair
		for _, k := range statusDeListas {
			delete(out, k)
		}
		out["status"] = domain.SugeridosPara("", s.rel.Agora())
		ate, _ := out["ativoAte"].(map[string]any) // o mapa cru nao sai: o front recebe so o ativo de cada meio (item 43)
		delete(out, "ativoAte")
		if err := s.juntarMeios(ctx, sid, out, ate); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// juntarMeios acrescenta a lista de meios (item 33a): cada nome de banco com a marca de cartao (e chave de CFG#CARTOES.porBanco) e
// ativo (hoje antes do ativoAte do meio, item 43) e, nos que tem, o ativoAte. O servidor junta os itens; o front nao faz chamada extra nem olha status.
func (s *Config) juntarMeios(ctx context.Context, sid string, listas domain.Doc, ativoAte map[string]any) error {
	nomes, ok := listas["banco"].([]any)
	if !ok {
		return nil
	}
	cartoes, err := s.repo.Ler(ctx, sid, "cartoes")
	if err != nil {
		log.Println("cfg: cartoes:", err)
		return ErrInterno
	}
	por, _ := cartoes["porBanco"].(map[string]any)
	hoje := domain.HojeDe(s.rel.Agora())
	meios := make([]domain.Doc, 0, len(nomes))
	for _, x := range nomes {
		nome, ok := x.(string)
		if !ok {
			continue
		}
		_, cartao := por[nome].(map[string]any)
		ate, _ := ativoAte[nome].(string)
		meio := domain.Doc{"nome": nome, "cartao": cartao, "ativo": domain.MeioAtivo(ate, hoje)}
		if ate != "" {
			meio["ativoAte"] = ate
		}
		meios = append(meios, meio)
	}
	listas["meios"] = meios
	// o ajuste de fatura que cada status de credito do catalogo traz por padrao (CREDITO EX +1, CREDITO IN -1, CREDITO 0): o front so consulta, sem olhar o nome do status
	padrao := domain.Doc{}
	for _, e := range domain.Catalogo {
		if e.Credito {
			padrao[e.Nome] = e.AjustePadrao
		}
	}
	listas["faturaPadrao"] = padrao
	// a cor padrao de cada status do catalogo, para o front colorir sem conhecer nome nenhum (a cor que a pessoa escolheu, statusCor, vale por cima)
	cores := domain.Doc{}
	for _, e := range domain.Catalogo {
		cores[e.Nome] = e.CorPadrao
	}
	listas["statusCorPadrao"] = cores
	return nil
}

// Gravar grava a configuracao. O "version" do corpo e o que o cliente leu; os campos protegidos nao entram. No conflito,
// devolve *ErroConflitoConfig com o estado atual para o cliente mesclar.
func (s *Config) Gravar(ctx context.Context, sid, nome string, body domain.Doc) (domain.Doc, error) {
	esperada, _ := body["version"].(float64)
	delete(body, "version")
	domain.Protegidos(body)
	switch strings.ToLower(nome) {
	case "listas":
		delete(body, "meios") // derivados pelo servidor na leitura, nunca gravados
		delete(body, "faturaPadrao")
		delete(body, "statusCorPadrao")
		// o status e as caixas de status sao do servidor (item 41): o corpo e ignorado e o que esta gravado continua como esta
		atual, _ := s.repo.Ler(ctx, sid, "listas")
		for _, k := range statusDeListas {
			delete(body, k)
			if v, ok := atual[k]; ok {
				body[k] = v
			}
		}
		// o ativoAte e do servidor (item 43): so as rotas de /meios o mudam; o corpo e ignorado e o gravado continua
		delete(body, "ativoAte")
		if v, ok := atual["ativoAte"]; ok {
			body["ativoAte"] = v
		}
		// um meio so sai da lista por DELETE /meios/{meio}, que confere o uso; o PUT acrescenta nomes, nunca tira
		if novos, ok := body["banco"].([]any); ok {
			presentes := map[string]bool{}
			for _, x := range novos {
				if n, ok := x.(string); ok {
					presentes[n] = true
				}
			}
			for _, n := range nomesDeBanco(atual) {
				if !presentes[n] {
					return nil, &ErroValidacao{"para tirar um meio de pagamento da lista use Remover: o servidor confere se ele ja foi usado"}
				}
			}
		} else if antigos, ok := atual["banco"]; ok {
			body["banco"] = antigos
		}
	case "combos":
		// sugeridos e geradoEm sao do servidor (item 57): so a rotina semanal os muda; o corpo e ignorado e o gravado continua
		delete(body, "rotinaAtiva")
		delete(body, "sugeridos")
		delete(body, "geradoEm")
		atual, _ := s.repo.Ler(ctx, sid, "combos")
		for _, k := range []string{"sugeridos", "geradoEm"} {
			if v, ok := atual[k]; ok {
				body[k] = v
			}
		}
	}
	res, err := s.repo.Salvar(ctx, sid, nome, body, esperada)
	switch {
	case errors.Is(err, domain.ErrDadosInvalidos):
		return nil, err
	case errors.Is(err, domain.ErrConflitoConfig):
		atual, _ := s.repo.Ler(ctx, sid, nome)
		return nil, &ErroConflitoConfig{Atual: domain.Publico(atual)}
	case err != nil:
		log.Println("put cfg:", err)
		return nil, ErrSalvar
	}
	return res, nil
}

// LerAparencia devolve a aparencia da pessoa; sem item, so {"version": 0}.
func (s *Config) LerAparencia(ctx context.Context, sid, sub string) (domain.Doc, error) {
	d, err := s.repo.LerAparencia(ctx, sid, sub)
	if err != nil {
		log.Println("aparencia:", err)
		return nil, ErrInterno
	}
	return semItem(d), nil
}

// GravarAparencia substitui a aparencia da pessoa (so a dela).
func (s *Config) GravarAparencia(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error) {
	domain.Protegidos(body)
	// o tema e escolha da pessoa, mas so existem tres: automatico (segue o aparelho), claro e escuro
	if t, ok := body["tema"]; ok {
		if s, _ := t.(string); s != "auto" && s != "claro" && s != "escuro" {
			return nil, &ErroValidacao{"tema invalido"}
		}
	}
	res, err := s.repo.GravarAparencia(ctx, sid, sub, body)
	switch {
	case errors.Is(err, domain.ErrDadosInvalidos):
		return nil, err
	case err != nil:
		log.Println("put aparencia:", err)
		return nil, ErrSalvar
	}
	return res, nil
}
