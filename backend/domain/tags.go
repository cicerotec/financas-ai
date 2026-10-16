package domain

import "strings"

// TagsDe le a lista de tags de um item (ignora o que nao for texto).
func TagsDe(d Doc) []string {
	var t []string
	if l, ok := d["tags"].([]any); ok {
		for _, x := range l {
			if s, ok := x.(string); ok {
				t = append(t, s)
			}
		}
	}
	return t
}

// NormalizarTags limpa a lista vinda do cliente: sem espacos nem "#", sem vazias, sem repetidas,
// no maximo 60 caracteres cada e 100 por chamada.
func NormalizarTags(entrada []string) []string {
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

// Faltantes devolve, na ordem, o que de "novas" ainda nao esta em "existentes".
func Faltantes(novas, existentes []string) []string {
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
