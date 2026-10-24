// Autorizacao por papel. Tudo aqui e puro (sem AWS), para ser testado em authz_test.go.
// Regra de ouro: o que nao esta liberado explicitamente e negado.
package auth

// cfgNomes sao as configuracoes compartilhadas que a API aceita em /cfg/{nome}.
var cfgNomes = map[string]bool{"listas": true, "tags": true, "combos": true, "saldos": true, "cartoes": true}

// Acao e uma coisa que alguem pode tentar fazer; cada rota da API vira uma Acao.
type Acao string

const (
	AcaoLerTx           Acao = "tx.ler"
	AcaoCriarTx         Acao = "tx.criar"
	AcaoEditarTx        Acao = "tx.editar" // inclui ocultar e ajustar fatura (sao PUT no mesmo lancamento)
	AcaoExcluirTx       Acao = "tx.excluir"
	AcaoImportarTx      Acao = "tx.importar"
	AcaoLerCfg          Acao = "cfg.ler"
	AcaoEscreverCfg     Acao = "cfg.escrever"
	AcaoAdicionarTags   Acao = "tags.adicionar"
	AcaoLerAparencia    Acao = "aparencia.ler"
	AcaoEscreverAparenc Acao = "aparencia.escrever"
	AcaoEditarPerfil    Acao = "perfil.editar" // o proprio nome de exibicao
	AcaoLerSeq          Acao = "seq.ler" // contador de alteracoes do espaco (a tela checa se alguem gravou)
	AcaoLerFech         Acao = "fechamento.ler"
	AcaoConferirMes     Acao = "fechamento.conferir"
	AcaoReabrirMes      Acao = "fechamento.reabrir"
	AcaoTestarAviso     Acao = "avisos.testar" // manda uma mensagem de teste para o proprio Telegram
	AcaoLerHistorico    Acao = "historico.ler" // busca por mes e versoes de um lancamento
	AcaoRestaurar       Acao = "historico.restaurar"
)

// Decisao e o que um papel pode fazer com uma acao.
type Decisao int

const (
	Negado   Decisao = iota // 403
	Liberado                // pode
	SoDono                  // pode apenas no que ele mesmo criou (criadoPor == sub)
)

const (
	PapelOwner  = "owner"
	PapelMember = "member"
)

// Permitido devolve o que o papel pode fazer com a acao. Papel ou acao desconhecidos: Negado.
func Permitido(papel string, a Acao) Decisao {
	switch papel {
	case PapelOwner:
		switch a {
		case AcaoLerTx, AcaoCriarTx, AcaoEditarTx, AcaoExcluirTx, AcaoImportarTx,
			AcaoLerCfg, AcaoEscreverCfg, AcaoAdicionarTags, AcaoLerAparencia, AcaoEscreverAparenc, AcaoEditarPerfil,
			AcaoLerFech, AcaoConferirMes, AcaoReabrirMes, AcaoLerSeq, AcaoTestarAviso,
			AcaoLerHistorico, AcaoRestaurar:
			return Liberado
		}
	case PapelMember:
		switch a {
		case AcaoLerTx, AcaoCriarTx, AcaoLerCfg, AcaoAdicionarTags, AcaoLerAparencia, AcaoEscreverAparenc, AcaoEditarPerfil,
			AcaoLerFech, AcaoLerSeq, AcaoTestarAviso:
			return Liberado
		case AcaoEditarTx, AcaoLerHistorico, AcaoRestaurar: // so o que ela criou (restaurar excluido e so do owner: service)
			return SoDono
		}
	}
	return Negado
}

// RotaParaAcao traduz metodo + caminho (o que vem depois de /spaces/{sid}/) na Acao.
// ok=false significa rota inexistente.
func RotaParaAcao(metodo string, rest []string) (Acao, bool) {
	switch {
	case len(rest) == 1 && rest[0] == "tx":
		switch metodo {
		case "GET":
			return AcaoLerTx, true
		case "POST":
			return AcaoCriarTx, true
		}
	case len(rest) == 2 && rest[0] == "tx" && rest[1] == "batch" && metodo == "POST":
		return AcaoImportarTx, true
	case len(rest) == 2 && rest[0] == "tx":
		switch metodo {
		case "PUT":
			return AcaoEditarTx, true
		case "DELETE":
			return AcaoExcluirTx, true
		}
	case len(rest) == 1 && rest[0] == "historico" && metodo == "GET":
		return AcaoLerHistorico, true
	case len(rest) == 3 && rest[0] == "tx" && rest[2] == "versoes" && metodo == "GET":
		return AcaoLerHistorico, true
	case len(rest) == 5 && rest[0] == "tx" && rest[2] == "versoes" && rest[4] == "restaurar" && metodo == "POST":
		return AcaoRestaurar, true
	case len(rest) == 2 && rest[0] == "cfg" && cfgNomes[rest[1]]:
		switch metodo {
		case "GET":
			return AcaoLerCfg, true
		case "PUT":
			return AcaoEscreverCfg, true
		}
	case len(rest) == 1 && rest[0] == "seq" && metodo == "GET":
		return AcaoLerSeq, true
	case len(rest) == 1 && rest[0] == "saldos" && metodo == "GET": // o saldo de cada meio, calculado pelo servidor (item 40)
		return AcaoLerFech, true
	case len(rest) == 2 && rest[0] == "fechamentos" && rest[1] == "conferir" && metodo == "POST":
		return AcaoConferirMes, true
	case len(rest) == 2 && rest[0] == "fechamentos" && rest[1] == "reabrir" && metodo == "POST":
		return AcaoReabrirMes, true
	case len(rest) == 1 && rest[0] == "tags" && metodo == "POST":
		return AcaoAdicionarTags, true
	case len(rest) == 1 && rest[0] == "perfil" && metodo == "PUT":
		return AcaoEditarPerfil, true
	case len(rest) == 2 && rest[0] == "avisos" && rest[1] == "teste" && metodo == "POST":
		return AcaoTestarAviso, true
	case len(rest) == 1 && rest[0] == "aparencia":
		switch metodo {
		case "GET":
			return AcaoLerAparencia, true
		case "PUT":
			return AcaoEscreverAparenc, true
		}
	}
	return "", false
}
