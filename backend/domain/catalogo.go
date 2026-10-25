package domain

import (
	"strings"
	"time"
)

// Catalogo de status (item 41, .ai/specs/futuros-previsto, R9): uma constante do backend, uma entrada por nome, com as regras que o servidor
// usa. Nenhum status sai dele: o que deixa de ser usado vira Selecionavel=false e continua valendo onde ja esta gravado. Na tela o status se
// chama forma de pagamento; o campo continua status. O front so exibe a lista que o servidor manda e devolve o texto escolhido.

// EntradaStatus e um status do catalogo.
type EntradaStatus struct {
	Nome         string
	Rotulo       string
	CorPadrao    string
	ContaNoSaldo bool // o lancamento mexe no saldo do meio
	ForaDoTotal  bool // o valor fica fora do total (excluirDoTotal)
	Credito      bool // e compra de cartao (entra na fatura)
	AjustePadrao int  // deslocamento de fatura de um status de credito (-1, 0 ou +1)
	Previsto     bool // e um dos nomes antigos de evento futuro
	Selecionavel bool // pode ser escolhido ao gravar
	Desde, Ate   string // AAAA-MM-DD: so orientam a sugestao da lista de escolha, nunca a validade
}

// StatusAntigosPrevistos sao os nomes que, antes do marcador previsto, faziam um lancamento ser de evento futuro.
var StatusAntigosPrevistos = []string{"PREVISTO", "TRANSFERINDO", "CREDITANDO"}

// Catalogo e o catalogo do corte, na ordem em que as selecionaveis sao sugeridas (o primeiro e o padrao). Os valores sao os padroes de hoje.
// E uma variavel so para os testes trocarem por um catalogo com mais entradas; o codigo de producao nunca a altera.
var Catalogo = []EntradaStatus{
	{Nome: "PAGO", Rotulo: "PAGO", CorPadrao: "#639922", ContaNoSaldo: true, Selecionavel: true},
	{Nome: "CREDITO", Rotulo: "CREDITO", CorPadrao: "#888780", Credito: true, Selecionavel: true},
	{Nome: "CONTAS", Rotulo: "CONTAS", CorPadrao: "#888780", ContaNoSaldo: true, ForaDoTotal: true},
	{Nome: "CREDITO IN", Rotulo: "CREDITO IN", CorPadrao: "#378ADD", Credito: true, AjustePadrao: -1},
	{Nome: "CREDITO EX", Rotulo: "CREDITO EX", CorPadrao: "#7F77DD", Credito: true, AjustePadrao: 1},
	{Nome: "PREVISTO", Rotulo: "PREVISTO", CorPadrao: "#BA7517", Previsto: true},
	{Nome: "TRANSFERINDO", Rotulo: "TRANSFERINDO", CorPadrao: "#888780", ForaDoTotal: true, Previsto: true},
	{Nome: "CREDITANDO", Rotulo: "CREDITANDO", CorPadrao: "#D85A30", Previsto: true},
}

// CorNeutra e a cor de um status fora do catalogo.
const CorNeutra = "#888780"

// EntradaDe devolve a entrada do catalogo com o nome dado.
func EntradaDe(nome string) (EntradaStatus, bool) {
	for _, e := range Catalogo {
		if e.Nome == nome {
			return e, true
		}
	}
	// o texto gravado pode vir com outra caixa, acento ou espacos a mais ("Crédito  in"): compara normalizado
	n := normStatus(nome)
	for _, e := range Catalogo {
		if normStatus(e.Nome) == n {
			return e, true
		}
	}
	return EntradaStatus{}, false
}

func normStatus(s string) string { return strings.Join(strings.Fields(strings.ToUpper(Normalizar(s))), " ") }

// StatusDe devolve o status gravado em um lancamento.
func StatusDe(l Lancamento) string {
	s, _ := l["status"].(string)
	return s
}

// Selecionaveis devolve os nomes que podem ser escolhidos, na ordem do catalogo.
func Selecionaveis() []string {
	out := []string{}
	for _, e := range Catalogo {
		if e.Selecionavel {
			out = append(out, e.Nome)
		}
	}
	return out
}

// SugeridosPara devolve as selecionaveis cuja vigencia (Desde e Ate, se houver) inclui o dia da data dada (um instante ISO; vazio vale hoje).
// A lista e so uma sugestao: a escolha do status e livre em qualquer data.
func SugeridosPara(dataEvento string, agora time.Time) []string {
	dia := agora.In(FusoApp).Format("2006-01-02")
	if dataEvento != "" {
		if d, err := DiaBR(dataEvento); err == nil {
			dia = d
		}
	}
	out := []string{}
	for _, e := range Catalogo {
		if !e.Selecionavel || (e.Desde != "" && dia < e.Desde) || (e.Ate != "" && dia > e.Ate) {
			continue
		}
		out = append(out, e.Nome)
	}
	return out
}

// ContaNoSaldoStatus: o status mexe no saldo (um status fora do catalogo nao mexe).
func ContaNoSaldoStatus(nome string) bool { e, ok := EntradaDe(nome); return ok && e.ContaNoSaldo }

// ForaDoTotalStatus: o valor do status fica fora do total.
func ForaDoTotalStatus(nome string) bool { e, ok := EntradaDe(nome); return ok && e.ForaDoTotal }

// RotuloECor devolvem o rotulo e a cor padrao do status; fora do catalogo, o proprio texto e a cor neutra.
func RotuloECor(nome string) (rotulo, cor string) {
	if e, ok := EntradaDe(nome); ok {
		return e.Rotulo, e.CorPadrao
	}
	return nome, CorNeutra
}

// OrdemNoCatalogo devolve a posicao do status no catalogo; os de fora vem depois de todos.
func OrdemNoCatalogo(nome string) int {
	for i, e := range Catalogo {
		if e.Nome == nome {
			return i
		}
	}
	return len(Catalogo)
}
