package service

import (
	"context"
	"errors"
	"log"
	"sort"
	"strings"

	"financas/domain"
)

// LimiteVersoesLidas e o teto de versoes que a busca do historico le por chamada. Sem indice por data, a busca le a
// particao do espaco inteira; passou do teto, para e avisa (truncado) para a pessoa refinar. Um GSI por data entra
// depois, sem mudar a rota.
const LimiteVersoesLidas = 5000

// Pessoas resolve o nome de quem agiu (o apelido no vinculo com o espaco).
type Pessoas interface {
	Apelido(ctx context.Context, sid, sub string) string
}

// Motivos pelos quais uma versao nao pode ser restaurada. O texto aparece na tela, ao lado do botao desabilitado.
var (
	ErrVersaoDeExclusao = errors.New("esta versao e a exclusao: restaure uma versao anterior")
	ErrJaNessaVersao    = errors.New("o lancamento ja esta nessa versao")
	ErrSoDonoRestaura   = errors.New("so o dono pode restaurar um lancamento excluido")
	// ErrSemAcesso: o member so acessa o historico dos lancamentos que ele criou.
	ErrSemAcesso       = errors.New("so e possivel acessar os lancamentos criados por voce")
	ErrVersaoNaoAchada = errors.New("versao nao encontrada")
)

// apelidos guarda o apelido de cada pessoa durante uma chamada, para nao reler o mesmo vinculo.
type apelidos struct {
	ctx   context.Context
	sid   string
	p     Pessoas
	cache map[string]string
}

func (a *apelidos) de(sub string) string {
	if sub == "" || a.p == nil {
		return ""
	}
	if n, ok := a.cache[sub]; ok {
		return n
	}
	n := a.p.Apelido(a.ctx, a.sid, sub)
	a.cache[sub] = n
	return n
}

func texto(x any) string { s, _ := x.(string); return s }

func contem(registro domain.Lancamento, q string) bool {
	return strings.Contains(strings.ToLower(texto(registro["descricao"])+" "+texto(registro["nota"])), q)
}

// Historico lista os lancamentos que mudaram no mes (excluidos e alterados juntos): uma linha por lancamento, com a
// ultima versao do mes. O mes e o da alteracao (fuso de Sao Paulo). q procura em descricao e nota de qualquer versao do
// mes. restrito: so os lancamentos que a pessoa criou (member).
func (s *Lancamentos) Historico(ctx context.Context, sid, mes, q, sub string, restrito bool) (domain.Doc, error) {
	if !domain.MesValido(mes) {
		return nil, &ErroValidacao{"mes obrigatorio, no formato AAAA-MM"}
	}
	vs, truncado, err := s.repo.VersoesDoEspaco(ctx, sid, LimiteVersoesLidas)
	if err != nil {
		log.Println("historico: ler versoes:", err)
		return nil, ErrInterno
	}
	q = strings.ToLower(strings.TrimSpace(q))
	type grupo struct {
		ultima domain.Versao
		casa   bool
	}
	grupos := map[string]*grupo{}
	for _, v := range vs {
		if domain.MesDe(v.Em) != mes || (restrito && !domain.EhDono(v.Registro, sub)) {
			continue
		}
		g := grupos[v.ID]
		if g == nil {
			g = &grupo{}
			grupos[v.ID] = g
		}
		if v.Numero > g.ultima.Numero {
			g.ultima = v
		}
		if q == "" || contem(v.Registro, q) {
			g.casa = true
		}
	}
	nomes := &apelidos{ctx: ctx, sid: sid, p: s.pessoas, cache: map[string]string{}}
	itens := []domain.Doc{}
	for id, g := range grupos {
		if !g.casa {
			continue
		}
		v, r := g.ultima, g.ultima.Registro
		var excluidoEm any
		if domain.Excluido(r) {
			excluidoEm = r["excluidoEm"]
		}
		itens = append(itens, domain.Doc{"id": id, "descricao": r["descricao"], "valor": r["valor"], "tipo": r["tipo"],
			"dataEvento": r["dataEvento"], "banco": r["banco"], "status": r["status"],
			"excluido": excluidoEm != nil, "excluidoEm": excluidoEm, "acao": string(v.Acao), "versao": v.Numero,
			"em": v.Em, "porSub": v.PorSub, "porNome": nomes.de(v.PorSub)})
	}
	sort.Slice(itens, func(i, j int) bool {
		if itens[i]["em"] != itens[j]["em"] {
			return texto(itens[i]["em"]) > texto(itens[j]["em"])
		}
		return texto(itens[i]["id"]) < texto(itens[j]["id"])
	})
	return domain.Doc{"mes": mes, "truncado": truncado, "itens": itens}, nil
}

// alvosDe junta, sem repetir, os alvos (banco e mes) de dois estados do lancamento.
func alvosDe(a, b domain.Lancamento) []domain.Alvo {
	var alvos []domain.Alvo
	for _, l := range []domain.Lancamento{a, b} {
		al, ok := domain.AlvoDe(l)
		if !ok {
			continue
		}
		repetido := false
		for _, x := range alvos {
			repetido = repetido || x == al
		}
		if !repetido {
			alvos = append(alvos, al)
		}
	}
	return alvos
}

// bloqueioDeRestauracao diz por que a versao nao pode ser restaurada (nil = pode). Os motivos sao, nesta ordem: a
// versao e a exclusao; so o dono restaura um excluido; o lancamento ja esta nessa versao; e a trava do fechamento, que
// vale para a versao e para o estado atual e so quando a restauracao mexe no saldo (como na edicao). O segundo valor
// e falha ao consultar a trava, nao um motivo.
func bloqueioDeRestauracao(tr Travas, atual domain.Lancamento, v domain.Versao, member bool) (motivo error, falha error) {
	if domain.Excluido(v.Registro) {
		return ErrVersaoDeExclusao, nil
	}
	excluido := domain.Excluido(atual)
	if member && excluido {
		return ErrSoDonoRestaura, nil
	}
	if !excluido && domain.MesmoConteudo(atual, v.Registro) {
		return ErrJaNessaVersao, nil
	}
	if excluido || domain.MudaSaldo(atual, v.Registro) {
		for _, al := range alvosDe(atual, v.Registro) {
			if err := tr.Checar(al); err != nil {
				var te *domain.ErroMesConferido
				if errors.As(err, &te) {
					return te, nil
				}
				return nil, err
			}
		}
	}
	return nil, nil
}

// Versoes devolve o historico de um lancamento, da versao mais nova para a mais antiga, com o diff de cada uma e se
// ela pode ser restaurada. Um lancamento antigo, que nunca foi alterado, nao tem historico: com "de" (a dataEvento do
// lancamento) devolve semHistorico e o estado atual; sem ela, 404. restrito: so o que a pessoa criou (member).
func (s *Lancamentos) Versoes(ctx context.Context, sid, id, de, sub string, restrito bool) (domain.Doc, error) {
	vs, err := s.repo.Versoes(ctx, sid, id)
	if err != nil {
		log.Println("versoes: ler:", err)
		return nil, ErrInterno
	}
	if len(vs) == 0 {
		data, err := domain.NormData(de)
		if err != nil {
			return nil, ErrNaoEncontrado
		}
		atual, err := s.repo.Buscar(ctx, sid, data, id)
		if err != nil {
			return nil, ErrInterno
		}
		if atual == nil {
			return nil, ErrNaoEncontrado
		}
		if restrito && !domain.EhDono(atual, sub) {
			return nil, ErrSemAcesso
		}
		return domain.Doc{"id": id, "excluido": domain.Excluido(atual), "semHistorico": true,
			"versoes": []domain.Doc{}, "atual": domain.Publico(domain.CopiaRasa(atual))}, nil
	}
	ultimo := vs[len(vs)-1]
	if restrito && !domain.EhDono(ultimo.Registro, sub) {
		return nil, ErrSemAcesso
	}
	tr := s.ef.Travas(ctx, sid)
	nomes := &apelidos{ctx: ctx, sid: sid, p: s.pessoas, cache: map[string]string{}}
	saida := []domain.Doc{}
	for i := len(vs) - 1; i >= 0; i-- {
		v := vs[i]
		diff := []domain.Doc{}
		if i > 0 {
			diff = domain.Diff(vs[i-1].Registro, v.Registro)
		}
		bloqueio, falha := bloqueioDeRestauracao(tr, ultimo.Registro, v, restrito)
		if falha != nil {
			log.Println("versoes: trava:", falha)
			return nil, ErrInterno
		}
		motivo := ""
		if bloqueio != nil {
			motivo = bloqueio.Error()
		}
		var deVersao any
		if v.DeVersao > 0 {
			deVersao = v.DeVersao
		}
		saida = append(saida, domain.Doc{"versao": v.Numero, "acao": string(v.Acao), "porSub": v.PorSub,
			"porNome": nomes.de(v.PorSub), "em": v.Em, "deVersao": deVersao, "diff": diff, "registro": v.Registro,
			"restauravel": bloqueio == nil, "motivo": motivo})
	}
	return domain.Doc{"id": id, "excluido": domain.Excluido(ultimo.Registro), "semHistorico": false, "versoes": saida}, nil
}
