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
	faturas     LeitorFaturaAviso
	recorrencias RodadorRecorrencias
}

// RodadorRecorrencias cria a copia seguinte das recorrencias (item 51); quem implementa e o service de recorrencias.
type RodadorRecorrencias interface {
	Rodar(ctx context.Context, hoje string, itens []domain.Doc) int
}

// UsarRecorrencias liga a rotina das recorrencias (montada depois, porque ela usa o service de lancamentos).
func (s *Avisos) UsarRecorrencias(r RodadorRecorrencias) { s.recorrencias = r }

// LeitorFaturaAviso le uma fatura no dia de avisar (item 46); quem implementa e o service de cartoes.
type LeitorFaturaAviso interface {
	FaturaParaAviso(ctx context.Context, sid, meio, ym string) (FaturaAviso, error)
}

// UsarFaturas liga a leitura das faturas (montada depois, porque os cartoes tambem usam os avisos).
func (s *Avisos) UsarFaturas(l LeitorFaturaAviso) { s.faturas = l }

// idAvisoFatura e o id do item auxiliar de uma fatura, que nao colide com o de um lancamento.
func idAvisoFatura(meio, ym string) string { return "fatura#" + meio + "#" + ym }

// SincronizarFatura grava o item auxiliar do aviso de uma fatura. Falha so vai ao log.
func (s *Avisos) SincronizarFatura(ctx context.Context, sid, meio, ym, venc string, dias []int, insistir bool) {
	lista := make([]any, len(dias))
	for i, n := range dias {
		lista[i] = float64(n)
	}
	item := domain.Doc{"id": idAvisoFatura(meio, ym), "tipo": "fatura", "meio": meio, "ym": ym, "venc": venc, "dias": lista, "insistir": insistir}
	if err := s.repo.Sincronizar(ctx, sid, item); err != nil {
		log.Println("aviso: gravar item auxiliar da fatura:", err)
	}
}

// RemoverFatura apaga o item auxiliar do aviso de uma fatura.
func (s *Avisos) RemoverFatura(ctx context.Context, sid, meio, ym string) {
	s.Remover(ctx, sid, idAvisoFatura(meio, ym))
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

	var recorrencias []domain.Doc
	for _, aux := range itens {
		sid, _ := aux["sid"].(string)
		id, _ := aux["id"].(string)
		if tipo, _ := aux["tipo"].(string); tipo == "recorrencia" { // a recorrencia nao e aviso: a rotina dela vem depois da leitura
			recorrencias = append(recorrencias, aux)
			continue
		}
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
		if tipo, _ := aux["tipo"].(string); tipo == "fatura" {
			if linha, ok := s.linhaDaFatura(ctx, sid, aux, hoje); ok {
				esp := espacos[sid]
				if esp == nil {
					esp = &contextoEspaco{ownerSub: s.dono.Dono(ctx, sid)}
					espacos[sid] = esp
				}
				// a fatura nao tem criador: o aviso vai para o dono do espaco
				if esp.ownerSub != "" {
					if chat := chatDe(sid, esp.ownerSub); chat != "" {
						porChat[chat] = append(porChat[chat], linha)
					}
				}
			}
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
			esp.ownerSub = s.dono.Dono(ctx, sid)
			espacos[sid] = esp
		}
		if !domain.EhPrevisto(tx) {
			continue // ja resolvido (ou nunca foi previsto): nao avisa
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

	if s.recorrencias != nil && len(recorrencias) > 0 {
		if n := s.recorrencias.Rodar(ctx, hoje, recorrencias); n > 0 {
			log.Printf("recorrencia: %d copia(s) criada(s)", n)
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

// linhaDaFatura confere a fatura de verdade na hora de avisar (item 46): o aviso e o vencimento ainda existem, nao foi paga e tem valor. Devolve a linha do resumo.
func (s *Avisos) linhaDaFatura(ctx context.Context, sid string, aux domain.Doc, hoje string) (domain.LinhaAviso, bool) {
	meio, _ := aux["meio"].(string)
	ym, _ := aux["ym"].(string)
	if s.faturas == nil || meio == "" || ym == "" {
		return domain.LinhaAviso{}, false
	}
	fa, err := s.faturas.FaturaParaAviso(ctx, sid, meio, ym)
	if err != nil {
		log.Println("avisos: ler fatura:", err)
		return domain.LinhaAviso{}, false
	}
	if !fa.Existe {
		s.RemoverFatura(ctx, sid, meio, ym)
		return domain.LinhaAviso{}, false
	}
	if fa.Paga || !(fa.Valor > 0) {
		return domain.LinhaAviso{}, false // o item fica: se o pagamento for desfeito, volta a avisar
	}
	kind, n := domain.DecidirAviso(hoje, fa.Venc, fa.Dias, fa.Insistir)
	if kind == domain.SemAviso {
		return domain.LinhaAviso{}, false
	}
	return domain.LinhaAviso{Kind: kind, N: n, Venc: fa.Venc, Tipo: "saida", Descricao: "Fatura " + domain.RotuloMes(ym) + " · " + meio, Valor: fa.Valor, Banco: meio}, true
}
