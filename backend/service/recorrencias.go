package service

import (
	"context"
	"errors"
	"log"

	"financas/domain"
)

// Recorrencia (item 51): o lancamento marcado com recorrente: { intervalo } e copiado sozinho. Quem comanda e um item auxiliar no
// AVISOS#ATIVOS (tipo "recorrencia"), que a rotina diaria dos avisos ja le: nenhuma chave nova, nenhuma varredura da tabela.
//
//	id        "recorrencia#<id do primeiro registro>" (estavel: nao muda a cada ciclo)
//	ultimoId  a ponta: a copia futura mais recente
//	dataUltimo  a dataEvento dela (parte da chave do lancamento)
//	intervalo meses entre uma copia e a seguinte
//	dia       o dia do mes original (para o 31 nao virar 28 para sempre)

// MaxCopiasPorRodada limita o que uma rodada da rotina grava (o banco tem 5 WCU); o que sobrar vai para o dia seguinte.
const MaxCopiasPorRodada = 50

const prefixoItemRecorrencia = "recorrencia#"

// RepositorioRecorrencias e o que a recorrencia precisa do banco: o item auxiliar no AVISOS#ATIVOS.
type RepositorioRecorrencias interface {
	Sincronizar(ctx context.Context, sid string, item domain.Doc) error
	Remover(ctx context.Context, sid, id string) error
	BuscarItem(ctx context.Context, sid, id string) (domain.Doc, error)
	ListarRecorrencias(ctx context.Context, sid string) ([]domain.Doc, error)
}

// LancamentosDaRecorrencia e o que a recorrencia pede ao service de lancamentos (o unico caminho de escrita).
type LancamentosDaRecorrencia interface {
	Ler(ctx context.Context, sid, dataEvento, id string) (domain.Lancamento, error)
	// CriarCopia grava a copia com o id dela; criada=false diz que ja existia (inclusive a apagada).
	CriarCopia(ctx context.Context, sid, criadoPor string, copia domain.Lancamento) (criada bool, err error)
	// AlterarMarca poe (marca != nil) ou tira (nil) o recorrente de um lancamento, como uma alteracao do dono (grava versao).
	AlterarMarca(ctx context.Context, sid, id, de string, marca any, sub string, restrito bool) error
}

type Recorrencias struct {
	repo RepositorioRecorrencias
	lanc LancamentosDaRecorrencia
	rel  Relogio
}

func NewRecorrencias(repo RepositorioRecorrencias, lanc LancamentosDaRecorrencia, rel Relogio) *Recorrencias {
	return &Recorrencias{repo: repo, lanc: lanc, rel: rel}
}

func (s *Recorrencias) hoje() string { return s.rel.Agora().In(domain.FusoApp).Format("2006-01-02") }

func idItemRecorrencia(primeiro string) string { return prefixoItemRecorrencia + primeiro }

// abrir cria a primeira copia (na primeira data depois de hoje) e o item. Idempotente: a copia tem id derivado e o item e regravado.
func (s *Recorrencias) abrir(ctx context.Context, sid string, origem domain.Lancamento, intervalo int) error {
	primeiro, _ := origem["id"].(string)
	data, _ := origem["dataEvento"].(string)
	dia, err := domain.DiaDoMes(data)
	if err != nil {
		return err
	}
	proxima, err := domain.ProximaCopiaRecorrencia(data, dia, intervalo, s.hoje())
	if err != nil {
		return err
	}
	criadoPor, _ := origem["criadoPor"].(string)
	copia := domain.CopiaDeRecorrencia(origem, proxima)
	copia["recorrente"] = domain.Doc{"intervalo": float64(intervalo)}
	copia["id"] = domain.IDCopiaRecorrencia(primeiro, proxima)
	if _, err := s.lanc.CriarCopia(ctx, sid, criadoPor, copia); err != nil {
		return err
	}
	return s.repo.Sincronizar(ctx, sid, domain.Doc{"id": idItemRecorrencia(primeiro), "tipo": "recorrencia",
		"ultimoId": copia["id"], "dataUltimo": proxima, "intervalo": float64(intervalo), "dia": float64(dia)})
}

// Iniciar abre a recorrencia de um lancamento recem-criado ja marcado (a marca ja foi gravada com ele, sem versao).
func (s *Recorrencias) Iniciar(ctx context.Context, sid string, origem domain.Lancamento, intervalo int) error {
	return s.abrir(ctx, sid, origem, intervalo)
}

// Marcar liga a recorrencia de um lancamento que ja existe (sob a data de). Cria a primeira copia e o item, e por fim poe a marca
// (uma alteracao: grava versao). Cada passo e repetivel: se cair no meio, marcar de novo termina o trabalho.
func (s *Recorrencias) Marcar(ctx context.Context, sid, sub, id, de string, intervalo int, restrito bool) (domain.Doc, error) {
	if _, _, err := domain.IntervaloRecorrente(map[string]any{"intervalo": float64(intervalo)}); err != nil {
		return nil, &ErroValidacao{err.Error()}
	}
	de, err := domain.NormData(de)
	if err != nil {
		return nil, &ErroValidacao{"parametro de (dataEvento atual) obrigatorio"}
	}
	origem, err := s.lanc.Ler(ctx, sid, de, id)
	if err != nil {
		return nil, ErrInterno
	}
	if origem == nil || domain.Excluido(origem) {
		return nil, &ErroValidacao{"lancamento nao encontrado"}
	}
	if restrito && !domain.EhDono(origem, sub) {
		return nil, ErrSoDono
	}
	if par, _ := origem["transferParId"].(string); par != "" {
		return nil, &ErroValidacao{"uma transferencia nao pode ser recorrente"}
	}
	// ja marcado (ele ou uma copia): desligue antes. O que ficou pela metade (item gravado e a marca ainda nao) se retoma marcando de novo.
	if origem["recorrente"] != nil {
		return nil, &ErroValidacao{"o lancamento ja e recorrente"}
	}
	origem["recorrente"] = domain.Doc{"intervalo": float64(intervalo)}
	if err := s.abrir(ctx, sid, origem, intervalo); err != nil {
		log.Println("recorrencia: abrir:", err)
		return nil, ErrSalvar
	}
	if err := s.lanc.AlterarMarca(ctx, sid, id, de, origem["recorrente"], sub, restrito); err != nil {
		return nil, err
	}
	return s.resumo(ctx, sid, id)
}

// resumo devolve { id, intervalo, dataUltimo } da recorrencia de um primeiro registro.
func (s *Recorrencias) resumo(ctx context.Context, sid, primeiro string) (domain.Doc, error) {
	item, err := s.repo.BuscarItem(ctx, sid, idItemRecorrencia(primeiro))
	if err != nil || item == nil {
		log.Println("recorrencia: ler item:", err)
		return nil, ErrInterno
	}
	return publicoDoItem(item), nil
}

func publicoDoItem(item domain.Doc) domain.Doc {
	id, _ := item["id"].(string)
	return domain.Doc{"id": id[len(prefixoItemRecorrencia):], "ultimoId": item["ultimoId"], "intervalo": item["intervalo"], "dataUltimo": item["dataUltimo"]}
}

// Desligar apaga o item e tira a marca da ponta. As copias futuras ja criadas ficam como estao. Desligar o que ja esta desligado
// nao faz nada.
func (s *Recorrencias) Desligar(ctx context.Context, sid, sub, primeiro string, restrito bool) error {
	item, err := s.repo.BuscarItem(ctx, sid, idItemRecorrencia(primeiro))
	if err != nil {
		log.Println("recorrencia: ler item:", err)
		return ErrInterno
	}
	if item == nil {
		return nil
	}
	ultimoID, _ := item["ultimoId"].(string)
	dataUltimo, _ := item["dataUltimo"].(string)
	ponta, err := s.lanc.Ler(ctx, sid, dataUltimo, ultimoID)
	if err != nil {
		return ErrInterno
	}
	vivo := ponta != nil && !domain.Excluido(ponta) && ponta["recorrente"] != nil
	if vivo && restrito && !domain.EhDono(ponta, sub) {
		return ErrSoDono
	}
	if err := s.repo.Remover(ctx, sid, idItemRecorrencia(primeiro)); err != nil {
		log.Println("recorrencia: apagar item:", err)
		return ErrSalvar
	}
	if vivo {
		return s.lanc.AlterarMarca(ctx, sid, ultimoID, dataUltimo, nil, sub, restrito)
	}
	return nil
}

// Listar devolve as recorrencias ativas do espaco.
func (s *Recorrencias) Listar(ctx context.Context, sid string) ([]domain.Doc, error) {
	itens, err := s.repo.ListarRecorrencias(ctx, sid)
	if err != nil {
		log.Println("recorrencia: listar:", err)
		return nil, ErrListar
	}
	lista := make([]domain.Doc, 0, len(itens))
	for _, it := range itens {
		if id, _ := it["id"].(string); len(id) > len(prefixoItemRecorrencia) {
			lista = append(lista, publicoDoItem(it))
		}
	}
	return lista, nil
}

// Rodar cria a seguinte de cada recorrencia cuja ponta venceu (dataUltimo <= hoje). itens sao os itens auxiliares do tipo recorrencia
// que a Query dos avisos ja trouxe: dia sem ponta vencida nao le mais nada. Devolve quantas copias criou.
func (s *Recorrencias) Rodar(ctx context.Context, hoje string, itens []domain.Doc) int {
	criadas := 0
	for _, it := range itens {
		if criadas >= MaxCopiasPorRodada {
			break
		}
		sid, _ := it["sid"].(string)
		id, _ := it["id"].(string)
		ultimoID, _ := it["ultimoId"].(string)
		dataUltimo, _ := it["dataUltimo"].(string)
		intervalo, _ := it["intervalo"].(float64)
		dia, _ := it["dia"].(float64)
		if sid == "" || len(id) <= len(prefixoItemRecorrencia) || ultimoID == "" || intervalo < 1 {
			continue
		}
		if d, err := domain.DiaLocal(dataUltimo); err != nil || d > hoje {
			continue // a ponta ainda nao venceu
		}
		primeiro := id[len(prefixoItemRecorrencia):]
		ponta, err := s.lanc.Ler(ctx, sid, dataUltimo, ultimoID)
		if err != nil {
			log.Println("recorrencia: ler a ponta:", err)
			continue
		}
		if ponta == nil { // apagada direto no banco: a recorrencia para
			log.Println("recorrencia: a ponta nao existe mais; recorrencia removida")
			if err := s.repo.Remover(ctx, sid, id); err != nil {
				log.Println("recorrencia: apagar item:", err)
			}
			continue
		}
		proxima, err := domain.ProximaCopiaRecorrencia(dataUltimo, int(dia), int(intervalo), hoje)
		if err != nil {
			log.Println("recorrencia: calcular a data:", err)
			continue
		}
		copia := domain.CopiaDeRecorrencia(ponta, proxima)
		copia["recorrente"] = domain.Doc{"intervalo": intervalo}
		copia["id"] = domain.IDCopiaRecorrencia(primeiro, proxima)
		criadoPor, _ := ponta["criadoPor"].(string)
		nova, err := s.lanc.CriarCopia(ctx, sid, criadoPor, copia)
		if err != nil {
			log.Println("recorrencia: criar a copia:", err)
			continue
		}
		if nova {
			criadas++
		}
		avancado := domain.CopiaRasa(it)
		avancado["ultimoId"], avancado["dataUltimo"] = copia["id"], proxima
		delete(avancado, "PK")
		delete(avancado, "SK")
		if err := s.repo.Sincronizar(ctx, sid, avancado); err != nil {
			log.Println("recorrencia: avancar o item:", err) // a proxima rodada cai no mesmo id e so avanca
		}
	}
	return criadas
}

// ---------- o lado do service de lancamentos ----------

// Ler le um lancamento vivo (inclusive o excluido logicamente), ou nil se nao existe.
func (s *Lancamentos) Ler(ctx context.Context, sid, dataEvento, id string) (domain.Lancamento, error) {
	return s.repo.Buscar(ctx, sid, dataEvento, id)
}

// CriarCopia grava uma copia de recorrencia, com o id e o criadoPor dados. Copia que ja existe (inclusive apagada pelo dono, que e o
// jeito de pular o ciclo) nao e recriada: devolve criada=false.
func (s *Lancamentos) CriarCopia(ctx context.Context, sid, criadoPor string, copia domain.Lancamento) (bool, error) {
	id, _ := copia["id"].(string)
	if !domain.ReID.MatchString(id) {
		return false, errors.New("id da copia invalido")
	}
	if _, err := s.criar(ctx, sid, criadoPor, copia, false, id); err != nil {
		if errors.Is(err, domain.ErrConflitoVersao) {
			return false, nil
		}
		return false, err
	}
	s.ef.ComSeq(ctx, sid, domain.Doc{}) // quem estiver com a tela aberta precisa saber que gravou
	return true, nil
}

// AlterarMarca poe ou tira o recorrente de um lancamento, como uma alteracao do dono (grava versao).
func (s *Lancamentos) AlterarMarca(ctx context.Context, sid, id, de string, marca any, sub string, restrito bool) error {
	_, err := s.editar(ctx, sid, id, de, domain.Lancamento{"recorrente": marca}, sub, restrito, false)
	return err
}
