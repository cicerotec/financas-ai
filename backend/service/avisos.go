package service

import (
	"context"
	"errors"
	"log"

	"financas/domain"
)

// RepositorioAvisos e o que o service precisa do banco para o item auxiliar de cada lancamento com aviso.
type RepositorioAvisos interface {
	Sincronizar(ctx context.Context, sid string, item domain.Doc) error
	Remover(ctx context.Context, sid, id string) error
	// Ativos devolve os itens auxiliares de todos os espacos.
	Ativos(ctx context.Context) ([]domain.Doc, error)
}

// Notificador envia um texto a um destino (hoje, o chat id do Telegram). Cada canal e um adaptador que o implementa;
// domain.ErrCanalNaoConfigurado diz que o canal nao tem o que precisa para enviar.
type Notificador interface {
	Enviar(ctx context.Context, destino, texto string) error
}

// LeitorLancamento le um lancamento vivo (inclusive o excluido logicamente), ou nil se nao existe.
type LeitorLancamento interface {
	Buscar(ctx context.Context, sid, dataEvento, id string) (domain.Lancamento, error)
}

// LeitorContatos le para onde enviar: o chat id de uma pessoa e o dono do espaco.
type LeitorContatos interface {
	ChatID(ctx context.Context, sid, sub string) (string, error)
}

type LeitorDono interface {
	Dono(ctx context.Context, sid string) string
}

// Erros do teste de aviso. O controller traduz cada um na resposta HTTP de sempre.
var (
	ErrTelegramNaoConfigurado = errors.New("o envio pelo Telegram ainda nao foi configurado no servidor")
	ErrTelegramRecusou        = errors.New("o Telegram recusou a mensagem: confira o chat id e se voce ja iniciou a conversa com o bot")
)

// Avisos mantem o item auxiliar de cada lancamento com aviso e roda a rotina diaria que manda o resumo do dia. O "hoje" vem
// do Relogio; a marca [DEV] vem do ambiente.
type Avisos struct {
	repo        RepositorioAvisos
	lancamentos LeitorLancamento
	cfg         RepositorioConfig
	contatos    LeitorContatos
	dono        LeitorDono
	notificador Notificador
	rel         Relogio
	ambiente    string
}

func NewAvisos(repo RepositorioAvisos, lancamentos LeitorLancamento, cfg RepositorioConfig, contatos LeitorContatos,
	dono LeitorDono, notificador Notificador, rel Relogio, ambiente string) *Avisos {
	return &Avisos{repo: repo, lancamentos: lancamentos, cfg: cfg, contatos: contatos, dono: dono, notificador: notificador, rel: rel, ambiente: ambiente}
}

// Sincronizar mantem o item auxiliar de um lancamento: grava se ele tem aviso, apaga se nao tem. Falha aqui nao derruba o
// salvamento do lancamento (o erro vai para o log).
func (s *Avisos) Sincronizar(ctx context.Context, sid string, tx domain.Lancamento) {
	id, _ := tx["id"].(string)
	if id == "" {
		return
	}
	dias, insistir, tem := domain.LerAviso(tx)
	if !tem {
		s.Remover(ctx, sid, id)
		return
	}
	data, _ := tx["dataEvento"].(string)
	venc, err := domain.DiaBR(data)
	if err != nil {
		log.Println("aviso: dataEvento invalida:", err)
		return
	}
	lista := make([]any, len(dias))
	for i, n := range dias {
		lista[i] = float64(n)
	}
	item := domain.Doc{"id": id, "dataEvento": data, "venc": venc, "dias": lista, "insistir": insistir}
	if err := s.repo.Sincronizar(ctx, sid, item); err != nil {
		log.Println("aviso: gravar item auxiliar:", err)
	}
}

// Remover apaga o item auxiliar; falha so vai ao log.
func (s *Avisos) Remover(ctx context.Context, sid, id string) {
	if err := s.repo.Remover(ctx, sid, id); err != nil {
		log.Println("aviso: apagar item auxiliar:", err)
	}
}

type contextoEspaco struct {
	listas   domain.Doc
	ownerSub string
}

// Rodar e chamada uma vez por dia pelo agendamento. Devolve quantos resumos foram enviados.
func (s *Avisos) Rodar(ctx context.Context) (int, error) {
	hoje := s.rel.Agora().In(domain.FusoBR).Format("2006-01-02")
	itens, err := s.repo.Ativos(ctx)
	if err != nil {
		return 0, err
	}

	espacos := map[string]*contextoEspaco{}
	chats := map[string]string{} // sid|sub -> chat id ("" = sem vinculo)
	porChat := map[string][]domain.LinhaAviso{}

	chatDe := func(sid, sub string) string {
		k := sid + "|" + sub
		if c, ok := chats[k]; ok {
			return c
		}
		c, err := s.contatos.ChatID(ctx, sid, sub)
		if err != nil {
			c = ""
		}
		chats[k] = c
		return c
	}

	for _, aux := range itens {
		sid, _ := aux["sid"].(string)
		id, _ := aux["id"].(string)
		data, _ := aux["dataEvento"].(string)
		venc, _ := aux["venc"].(string)
		dias, insistir, _ := domain.LerAviso(domain.Doc{"aviso": domain.Doc{"dias": aux["dias"], "insistir": aux["insistir"]}})
		kind, n := domain.DecidirAviso(hoje, venc, dias, insistir)
		if sid == "" || id == "" {
			continue
		}
		// vencido ha muito tempo: para de insistir e limpa
		if d, err := domain.DiaParaTempo(venc); err == nil {
			if h, _ := domain.DiaParaTempo(hoje); int(h.Sub(d).Hours()/24) > domain.MaxAtrasoAviso {
				s.Remover(ctx, sid, id)
				continue
			}
		}
		if kind == domain.SemAviso {
			continue
		}
		tx, err := s.lancamentos.Buscar(ctx, sid, data, id)
		if err != nil {
			log.Println("avisos: ler lancamento:", err)
			continue
		}
		if tx == nil || tx["excluidoEm"] != nil { // excluido
			s.Remover(ctx, sid, id)
			continue
		}
		if oculto, _ := tx["oculto"].(bool); oculto {
			continue
		}
		esp := espacos[sid]
		if esp == nil {
			esp = &contextoEspaco{}
			esp.listas, _ = s.cfg.Ler(ctx, sid, "listas")
			esp.ownerSub = s.dono.Dono(ctx, sid)
			espacos[sid] = esp
		}
		status, _ := tx["status"].(string)
		if !domain.EhStatusFuturo(esp.listas, status) {
			continue // ja resolvido (ou nunca foi futuro): nao avisa
		}
		valor, _ := tx["valor"].(float64)
		tipo, _ := tx["tipo"].(string)
		desc, _ := tx["descricao"].(string)
		banco, _ := tx["banco"].(string)
		linha := domain.LinhaAviso{Kind: kind, N: n, Venc: venc, Tipo: tipo, Descricao: desc, Valor: valor, Banco: banco}

		criador, _ := tx["criadoPor"].(string)
		vistos := map[string]bool{}
		for _, sub := range []string{criador, esp.ownerSub} {
			if sub == "" {
				continue
			}
			chat := chatDe(sid, sub)
			if chat == "" || vistos[chat] {
				continue
			}
			vistos[chat] = true
			porChat[chat] = append(porChat[chat], linha)
		}
	}

	enviados := 0
	marca := domain.MarcaAmbiente(s.ambiente)
	for chat, linhas := range porChat {
		if err := s.notificador.Enviar(ctx, chat, domain.MontarResumo(marca, hoje, linhas)); err != nil {
			if errors.Is(err, domain.ErrCanalNaoConfigurado) {
				log.Println("avisos: Telegram nao configurado; nada enviado")
				return enviados, nil
			}
			log.Println("avisos: enviar:", err)
			continue
		}
		enviados++
	}
	log.Printf("avisos: %d item(ns) ativos, %d resumo(s) enviado(s)", len(itens), enviados)
	return enviados, nil
}

// Testar manda uma mensagem de teste para o Telegram de quem fez a chamada.
func (s *Avisos) Testar(ctx context.Context, sid, sub string) error {
	chat, err := s.contatos.ChatID(ctx, sid, sub)
	if err != nil {
		log.Println("aviso teste:", err)
		return ErrInterno
	}
	if chat == "" {
		return &ErroValidacao{"cadastre o seu chat id do Telegram primeiro"}
	}
	err = s.notificador.Enviar(ctx, chat, "🔔 "+domain.MarcaAmbiente(s.ambiente)+"Teste do financas.ai: se você recebeu esta mensagem, os avisos vão chegar aqui.")
	switch {
	case errors.Is(err, domain.ErrCanalNaoConfigurado):
		return ErrTelegramNaoConfigurado
	case err != nil:
		log.Println("aviso teste:", err)
		return ErrTelegramRecusou
	}
	return nil
}
