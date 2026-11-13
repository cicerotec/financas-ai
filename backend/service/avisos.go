package service

import (
	"context"
	"errors"
	"log"
	"sort"

	"financas/domain"
)

// RepositorioAvisos e o que o service precisa do banco para o item auxiliar de cada lancamento com aviso.
type RepositorioAvisos interface {
	Sincronizar(ctx context.Context, sid string, item domain.Doc) error
	Remover(ctx context.Context, sid, id string) error
	// Ativos devolve os itens auxiliares de todos os espacos.
	Ativos(ctx context.Context) ([]domain.Doc, error)
}

// LeitorLancamento le um lancamento vivo (inclusive o excluido logicamente), ou nil se nao existe.
type LeitorLancamento interface {
	Buscar(ctx context.Context, sid, dataEvento, id string) (domain.Lancamento, error)
}

// LeitorContatos le para onde enviar: os destinos de uma pessoa no espaco, por canal ({"telegram": [chat id]}). Ter destino e
// estar assinado; nao ha campo de "canais ligados".
type LeitorContatos interface {
	Destinos(ctx context.Context, sid, sub string) (map[string][]string, error)
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
	despachante *Despachante
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
	dono LeitorDono, despachante *Despachante, rel Relogio, ambiente string) *Avisos {
	return &Avisos{repo: repo, lancamentos: lancamentos, cfg: cfg, contatos: contatos, dono: dono, despachante: despachante, rel: rel, ambiente: ambiente}
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
	cache := map[string][]Destino{} // sid|sub -> destinos (vazio = sem vinculo ou sem destino)
	porDestino := map[Destino][]domain.LinhaAviso{}

	destinosDe := func(sid, sub string) []Destino {
		k := sid + "|" + sub
		if d, ok := cache[k]; ok {
			return d
		}
		var lista []Destino
		if m, err := s.contatos.Destinos(ctx, sid, sub); err == nil {
			for canal, enderecos := range m {
				for _, e := range enderecos {
					if e != "" {
						lista = append(lista, Destino{canal, e})
					}
				}
			}
		}
		cache[k] = lista
		return lista
	}

	var recorrencias []domain.Doc
	for _, aux := range itens {
		sid, _ := aux["sid"].(string)
		id, _ := aux["id"].(string)
		if tipo, _ := aux["tipo"].(string); tipo == "recorrencia" { // a recorrencia nao e aviso: a rotina dela vem depois da leitura
			recorrencias = append(recorrencias, aux)
			continue
		}
		if tipo, _ := aux["tipo"].(string); tipo == domain.TipoItemCombos { // o interruptor da rotina dos combos (item 57) nao e aviso
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
					for _, d := range destinosDe(sid, esp.ownerSub) {
						porDestino[d] = append(porDestino[d], linha)
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
		vistos := map[Destino]bool{} // a mesma linha nao vai duas vezes ao mesmo destino (criador igual ao dono, ou mesmo chat)
		for _, sub := range []string{criador, esp.ownerSub} {
			if sub == "" {
				continue
			}
			for _, d := range destinosDe(sid, sub) {
				if vistos[d] {
					continue
				}
				vistos[d] = true
				porDestino[d] = append(porDestino[d], linha)
			}
		}
	}

	if s.recorrencias != nil && len(recorrencias) > 0 {
		if n := s.recorrencias.Rodar(ctx, hoje, recorrencias); n > 0 {
			log.Printf("recorrencia: %d copia(s) criada(s)", n)
		}
	}

	// os grupos saem em ordem estavel (canal e endereco): a iteracao de mapa e aleatoria
	destinos := make([]Destino, 0, len(porDestino))
	for d := range porDestino {
		destinos = append(destinos, d)
	}
	sort.Slice(destinos, func(i, j int) bool {
		if destinos[i].Canal != destinos[j].Canal {
			return destinos[i].Canal < destinos[j].Canal
		}
		return destinos[i].Endereco < destinos[j].Endereco
	})

	enviados := 0
	parados := map[string]bool{} // canais sem configuracao nesta rodada: o resto dos grupos deles e pulado
	marca := domain.MarcaAmbiente(s.ambiente)
	for _, d := range destinos {
		if parados[d.Canal] {
			continue
		}
		// nenhuma linha de log leva o destino, o texto ou o valor: so o nome do canal
		err := s.despachante.Entregar(ctx, d, domain.ResumoAvisos{Hoje: hoje, Marca: marca, Linhas: porDestino[d]})
		switch {
		case errors.Is(err, domain.ErrCanalNaoConfigurado):
			parados[d.Canal] = true
			log.Printf("avisos: canal %s nao configurado; nada enviado por ele", d.Canal)
		case err != nil:
			log.Printf("avisos: enviar pelo canal %s: %v", d.Canal, err)
		default:
			enviados++
		}
	}
	log.Printf("avisos: %d item(ns) ativos, %d resumo(s) enviado(s)", len(itens), enviados)
	return enviados, nil
}

// Testar manda uma mensagem de teste para o Telegram de quem fez a chamada.
func (s *Avisos) Testar(ctx context.Context, sid, sub string) error {
	todos, err := s.contatos.Destinos(ctx, sid, sub)
	if err != nil {
		log.Println("aviso teste:", err)
		return ErrInterno
	}
	var chat string
	if l := todos["telegram"]; len(l) > 0 {
		chat = l[0]
	}
	if chat == "" {
		return &ErroValidacao{"cadastre o seu chat id do Telegram primeiro"}
	}
	err = s.despachante.Entregar(ctx, Destino{"telegram", chat}, domain.ResumoAvisos{Marca: domain.MarcaAmbiente(s.ambiente), Teste: true})
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
