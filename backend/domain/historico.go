package domain

import (
	"reflect"
	"regexp"
	"sort"
)

var reMes = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// MesValido confere o formato AAAA-MM de um mes.
func MesValido(m string) bool { return reMes.MatchString(m) }

// Diff compara dois registros e lista so os campos que mudaram, em ordem alfabetica, com o valor antes e depois
// (nil quando o campo nao existia). PK e SK nao entram. Sem mudanca devolve uma lista vazia, nunca nil.
func Diff(antes, depois Doc) []Doc {
	campos := map[string]bool{}
	for k := range antes {
		campos[k] = true
	}
	for k := range depois {
		campos[k] = true
	}
	delete(campos, "PK")
	delete(campos, "SK")
	nomes := make([]string, 0, len(campos))
	for k := range campos {
		nomes = append(nomes, k)
	}
	sort.Strings(nomes)
	mudancas := []Doc{}
	for _, k := range nomes {
		if !reflect.DeepEqual(antes[k], depois[k]) {
			mudancas = append(mudancas, Doc{"campo": k, "antes": antes[k], "depois": depois[k]})
		}
	}
	return mudancas
}

// MesmoConteudo diz se dois lancamentos tem o mesmo conteudo, sem olhar as chaves do banco nem a marca de exclusao.
func MesmoConteudo(a, b Lancamento) bool {
	ra, rb := Registro(a), Registro(b)
	delete(ra, "excluidoEm")
	delete(rb, "excluidoEm")
	return reflect.DeepEqual(ra, rb)
}
