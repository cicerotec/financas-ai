package domain

import "sort"

// Combos sugeridos (item 57): os conjuntos de tags que mais se repetem nos lancamentos recentes. O servidor conta uma vez por
// semana e guarda o resultado em CFG#COMBOS (sugeridos); nenhuma tela conta.
const (
	CombosMinTags = 2    // um conjunto tem 2 tags ou mais
	CombosMinUsos = 3    // e aparece em 3 lancamentos ou mais
	CombosMax     = 10   // o servidor guarda ate 10 (o app mostra 5 e "Mostrar mais")
	CombosAmostra = 1000 // os 1000 lancamentos mais recentes (cerca de 6 meses)
)

// TipoItemCombos e o tipo do item auxiliar no AVISOS#ATIVOS que liga a rotina semanal de um espaco (o interruptor em Listas).
const TipoItemCombos = "combos"

// CombosSugeridos conta os conjuntos de tags dos lancamentos, na ordem em que chegam (do mais novo para tras). A chave do
// conjunto e a lista de tags ordenada, entao a ordem das tags nao importa; as tags saem na ordem do primeiro lancamento que
// formou o conjunto. Excluidos e previstos nao contam. Devolve [{tags, contagem}] do mais usado ao menos (empate: pela
// chave), no maximo CombosMax.
func CombosSugeridos(lancs []Lancamento) []Doc {
	type conjunto struct {
		tags  []string
		chave string
		n     int
	}
	por := map[string]*conjunto{}
	for _, l := range lancs {
		if Excluido(l) || EhPrevisto(l) {
			continue
		}
		tags := tagsDoLancamento(l)
		if len(tags) < CombosMinTags {
			continue
		}
		ord := append([]string(nil), tags...)
		sort.Strings(ord)
		k := ""
		for i, t := range ord {
			if i > 0 {
				k += "|"
			}
			k += t
		}
		if c := por[k]; c != nil {
			c.n++
		} else {
			por[k] = &conjunto{tags: tags, chave: k, n: 1}
		}
	}
	var lista []*conjunto
	for _, c := range por {
		if c.n >= CombosMinUsos {
			lista = append(lista, c)
		}
	}
	sort.Slice(lista, func(i, j int) bool {
		if lista[i].n != lista[j].n {
			return lista[i].n > lista[j].n
		}
		return lista[i].chave < lista[j].chave
	})
	if len(lista) > CombosMax {
		lista = lista[:CombosMax]
	}
	res := make([]Doc, 0, len(lista))
	for _, c := range lista {
		tags := make([]any, len(c.tags))
		for i, t := range c.tags {
			tags[i] = t
		}
		res = append(res, Doc{"tags": tags, "contagem": c.n})
	}
	return res
}

// tagsDoLancamento le as tags (texto) de um lancamento, sem repetir.
func tagsDoLancamento(l Lancamento) []string {
	lista, _ := l["tags"].([]any)
	vistas := map[string]bool{}
	var tags []string
	for _, x := range lista {
		if t, ok := x.(string); ok && t != "" && !vistas[t] {
			vistas[t] = true
			tags = append(tags, t)
		}
	}
	return tags
}
