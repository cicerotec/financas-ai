package service

import (
	"context"
	"log"
	"time"

	"financas/domain"
)

// Restaurar recupera a versao n de um lancamento: grava uma versao nova (restore) com o conteudo daquela, sem reescrever
// o passado. Se o lancamento esta excluido, reativa no mesmo id (e, se a data da versao e outra, troca a chave). Passa
// pelos mesmos efeitos de uma edicao e pela trava do fechamento, para a versao e para o estado atual. restrito: o
// member so restaura o que criou e que ainda existe.
func (s *Lancamentos) Restaurar(ctx context.Context, sid, id string, n int, sub string, restrito, confirma bool) (domain.Lancamento, error) {
	vs, err := s.repo.Versoes(ctx, sid, id)
	if err != nil {
		log.Println("restaurar: ler versoes:", err)
		return nil, ErrInterno
	}
	if len(vs) == 0 {
		return nil, ErrNaoEncontrado
	}
	var escolhida *domain.Versao
	for i := range vs {
		if vs[i].Numero == n {
			escolhida = &vs[i]
		}
	}
	if escolhida == nil {
		return nil, ErrVersaoNaoAchada
	}
	ultimo := vs[len(vs)-1]
	if restrito && !domain.EhDono(ultimo.Registro, sub) {
		return nil, ErrSemAcesso
	}
	bloqueio, exige, falha := bloqueioDeRestauracao(s.ef.Travas(ctx, sid), ultimo.Registro, *escolhida, restrito)
	if falha != nil {
		log.Println("restaurar: trava:", falha)
		return nil, ErrInterno
	}
	if bloqueio != nil {
		return nil, bloqueio
	}
	if exige != nil && !confirma { // mes fechado por data: so com a confirmacao explicita
		return nil, exige
	}
	dataVivo, _ := ultimo.Registro["dataEvento"].(string)
	vivo, err := s.repo.Buscar(ctx, sid, dataVivo, id)
	if err != nil {
		return nil, ErrInterno
	}
	if vivo == nil { // o historico diz que existe e a tabela nao: nao inventa um lancamento
		log.Printf("restaurar: lancamento %s sem item vivo (versao %d)", id, ultimo.Numero)
		return nil, ErrInterno
	}
	depois := domain.Registro(escolhida.Registro)
	delete(depois, "excluidoEm")
	if err := domain.ValidarTx(depois); err != nil {
		return nil, &ErroValidacao{err.Error()}
	}
	if err := s.travaDeFatura(ctx, sid).Checar(vivo, depois); err != nil {
		return nil, err
	}
	versoes := domain.VersoesDaEscrita(ultimo.Numero, vivo, depois, domain.AcaoRestore, sub, time.Now().UTC().Format(formatoTempo))
	versoes[len(versoes)-1].DeVersao = n
	marcarConfirmacao(versoes, exige != nil)
	if err := s.repo.Salvar(ctx, sid, domain.Mudanca{Antes: vivo, Depois: depois, DataAnterior: dataVivo, Versoes: versoes}); err != nil {
		return nil, escrita("restaurar", ErrSalvar, err)
	}
	if alvos := domain.AlvosDoSaldo(vivo, depois); len(alvos) > 0 {
		s.ef.InvalidarSaldos(ctx, sid, alvos)
	}
	s.ef.GarantirTags(ctx, sid, domain.TagsDe(depois))
	s.ef.SincronizarAviso(ctx, sid, depois)
	return s.ef.ComSeq(ctx, sid, domain.Publico(depois)), nil
}
