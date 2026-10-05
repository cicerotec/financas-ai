package main

import "strings"

// Autorizacao por papel. Tudo aqui e puro (sem AWS), para ser testado em authz_test.go.
// Regra de ouro: o que nao esta liberado explicitamente e negado.

type acao string

const (
	acaoLerTx           acao = "tx.ler"
	acaoCriarTx         acao = "tx.criar"
	acaoEditarTx        acao = "tx.editar" // inclui ocultar e ajustar fatura (sao PUT no mesmo lancamento)
	acaoExcluirTx       acao = "tx.excluir"
	acaoImportarTx      acao = "tx.importar"
	acaoLerCfg          acao = "cfg.ler"
	acaoEscreverCfg     acao = "cfg.escrever"
	acaoAdicionarTags   acao = "tags.adicionar"
	acaoLerAparencia    acao = "aparencia.ler"
	acaoEscreverAparenc acao = "aparencia.escrever"
	acaoEditarPerfil    acao = "perfil.editar" // o proprio nome de exibicao
	acaoLerFech         acao = "fechamento.ler"
	acaoEscreverFech    acao = "fechamento.escrever" // cache dos meses fechados
	acaoConferirMes     acao = "fechamento.conferir"
	acaoReabrirMes      acao = "fechamento.reabrir"
)

type decisao int

const (
	negado   decisao = iota // 403
	liberado                // pode
	soDono                  // pode apenas no que ele mesmo criou (criadoPor == sub)
)

const (
	papelOwner  = "owner"
	papelMember = "member"
)

// permitido devolve o que o papel pode fazer com a acao. Papel ou acao desconhecidos: negado.
func permitido(papel string, a acao) decisao {
	switch papel {
	case papelOwner:
		switch a {
		case acaoLerTx, acaoCriarTx, acaoEditarTx, acaoExcluirTx, acaoImportarTx,
			acaoLerCfg, acaoEscreverCfg, acaoAdicionarTags, acaoLerAparencia, acaoEscreverAparenc, acaoEditarPerfil,
			acaoLerFech, acaoEscreverFech, acaoConferirMes, acaoReabrirMes:
			return liberado
		}
	case papelMember:
		switch a {
		case acaoLerTx, acaoCriarTx, acaoLerCfg, acaoAdicionarTags, acaoLerAparencia, acaoEscreverAparenc, acaoEditarPerfil,
			acaoLerFech:
			return liberado
		case acaoEditarTx:
			return soDono
		}
	}
	return negado
}

// ehDono: o lancamento foi criado por este usuario? Registro sem criadoPor nao e de ninguem
// (so o owner mexe nele).
func ehDono(reg doc, sub string) bool {
	c, _ := reg["criadoPor"].(string)
	return sub != "" && c == sub
}

// rotaParaAcao traduz metodo + caminho (o que vem depois de /spaces/{sid}/) na acao.
// ok=false significa rota inexistente.
func rotaParaAcao(metodo string, rest []string) (acao, bool) {
	switch {
	case len(rest) == 1 && rest[0] == "tx":
		switch metodo {
		case "GET":
			return acaoLerTx, true
		case "POST":
			return acaoCriarTx, true
		}
	case len(rest) == 2 && rest[0] == "tx" && rest[1] == "batch" && metodo == "POST":
		return acaoImportarTx, true
	case len(rest) == 2 && rest[0] == "tx":
		switch metodo {
		case "PUT":
			return acaoEditarTx, true
		case "DELETE":
			return acaoExcluirTx, true
		}
	case len(rest) == 2 && rest[0] == "cfg" && cfgNomes[rest[1]]:
		switch metodo {
		case "GET":
			return acaoLerCfg, true
		case "PUT":
			return acaoEscreverCfg, true
		}
	case len(rest) == 1 && rest[0] == "fechamentos":
		switch metodo {
		case "GET":
			return acaoLerFech, true
		case "PUT":
			return acaoEscreverFech, true
		}
	case len(rest) == 2 && rest[0] == "fechamentos" && rest[1] == "conferir" && metodo == "POST":
		return acaoConferirMes, true
	case len(rest) == 2 && rest[0] == "fechamentos" && rest[1] == "reabrir" && metodo == "POST":
		return acaoReabrirMes, true
	case len(rest) == 1 && rest[0] == "tags" && metodo == "POST":
		return acaoAdicionarTags, true
	case len(rest) == 1 && rest[0] == "perfil" && metodo == "PUT":
		return acaoEditarPerfil, true
	case len(rest) == 1 && rest[0] == "aparencia":
		switch metodo {
		case "GET":
			return acaoLerAparencia, true
		case "PUT":
			return acaoEscreverAparenc, true
		}
	}
	return "", false
}

// normalizarTags limpa a lista vinda do cliente: sem espacos nem "#", sem vazias, sem repetidas,
// no maximo 60 caracteres cada e 100 por chamada.
func normalizarTags(entrada []string) []string {
	vistas := map[string]bool{}
	var saida []string
	for _, t := range entrada {
		t = strings.TrimPrefix(strings.TrimSpace(t), "#")
		t = strings.TrimSpace(t)
		if t == "" || len([]rune(t)) > 60 || vistas[t] {
			continue
		}
		vistas[t] = true
		saida = append(saida, t)
		if len(saida) == 100 {
			break
		}
	}
	return saida
}

// faltantes devolve, na ordem, o que de "novas" ainda nao esta em "existentes".
func faltantes(novas, existentes []string) []string {
	tem := make(map[string]bool, len(existentes))
	for _, e := range existentes {
		tem[e] = true
	}
	var f []string
	for _, n := range novas {
		if !tem[n] {
			f = append(f, n)
		}
	}
	return f
}
