package service

import (
	"context"
	"log"
	"math"
	"time"

	"financas/domain"
)

// statusTransferencia e o status das duas pontas de uma transferencia entre meios (a entrada CONTAS do catalogo).
const statusTransferencia = "CONTAS"

// Transferir: POST /spaces/{sid}/transferencias { origem, destino, valor, dataEvento? } (R14). Cria as duas pontas (saida na origem, entrada no destino)
// com o mesmo transferParId, o status de transferencia do catalogo e o excluirDoTotal derivado, numa transacao so, com as travas de mes nos dois meios.
func (s *Lancamentos) Transferir(ctx context.Context, sid, sub string, body domain.Doc, confirma bool) (domain.Doc, error) {
	origem, _ := body["origem"].(string)
	destino, _ := body["destino"].(string)
	valor, ok := body["valor"].(float64)
	if origem == "" || destino == "" || origem == destino || !ok || !(valor > 0) || math.IsInf(valor, 0) {
		return nil, &ErroValidacao{"confira os meios (origem e destino diferentes) e o valor (maior que zero)"}
	}
	agora := time.Now().UTC().Format(formatoTempo)
	data := agora
	if txt, _ := body["dataEvento"].(string); txt != "" {
		d, err := domain.NormData(txt)
		if err != nil {
			return nil, &ErroValidacao{"dataEvento invalida"}
		}
		data = d
	}
	valor = domain.Arred(valor)
	parID := domain.NovoID()
	ponta := func(tipo, banco, desc string) domain.Lancamento {
		l := domain.Lancamento{
			"id": domain.NovoID(), "status": statusTransferencia, "tipo": tipo, "valor": valor, "descricao": desc, "nota": "",
			"dataEvento": data, "banco": banco, "tags": []any{}, "criadoEm": agora, "criadoPor": sub, "transferParId": parID,
		}
		s.derivar(l)
		return l
	}
	pontas := []domain.Lancamento{
		ponta("saida", origem, "Transferência para "+destino),
		ponta("entrada", destino, "Transferência de "+origem),
	}
	tr := s.ef.Travas(ctx, sid)
	var mudancas []domain.Mudanca
	var alvos []domain.Alvo
	for _, l := range pontas {
		exigiu := false
		if al, tem := alvoDoEfeito(l); tem {
			ex, err := tr.Checar(al, confirma)
			if err != nil {
				return nil, err
			}
			exigiu = ex
			alvos = append(alvos, al)
		}
		var versoes []domain.Versao
		if exigiu { // a criacao que exigiu confirmacao de mes fechado fica no historico (como em Criar)
			versoes = domain.VersoesDaEscrita(0, nil, l, domain.AcaoInsert, sub, agora)
			marcarConfirmacao(versoes, exigiu)
		}
		mudancas = append(mudancas, domain.Mudanca{Depois: l, Versoes: versoes})
	}
	if err := s.repo.SalvarLote(ctx, sid, mudancas); err != nil {
		log.Println("transferencia: salvar lote:", err)
		return nil, ErrSalvar
	}
	if len(alvos) > 0 {
		s.ef.InvalidarSaldos(ctx, sid, alvos)
	}
	resposta := domain.Doc{"ok": true, "transferParId": parID, "ids": []string{pontas[0]["id"].(string), pontas[1]["id"].(string)}}
	return s.ef.ComSeq(ctx, sid, resposta), nil
}
