package domain

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
)

const (
	maxDiasAviso  = 30
	maxItensAviso = 6 // os quatro atalhos da tela mais dois prazos proprios (.ai/padrao-de-telas.md)

	maxIndiceSerie    = 999
	maxIntervaloSerie = 120 // meses
)

// ReID e o formato aceito para o id de um lancamento ou de uma serie.
var ReID = regexp.MustCompile(`^[a-z0-9]{6,40}$`)

// ValidarAviso confere e normaliza d["aviso"]; null remove o campo.
func ValidarAviso(d Doc) error {
	bruto, existe := d["aviso"]
	if !existe {
		return nil
	}
	if bruto == nil {
		delete(d, "aviso")
		return nil
	}
	m, ok := bruto.(map[string]any)
	if !ok {
		return errors.New("aviso deve ser um objeto")
	}
	var dias []int
	vistos := map[int]bool{}
	if l, tem := m["dias"]; tem && l != nil {
		arr, ok := l.([]any)
		if !ok {
			return errors.New("aviso.dias deve ser uma lista")
		}
		if len(arr) > maxItensAviso {
			return fmt.Errorf("aviso.dias aceita no maximo %d itens", maxItensAviso)
		}
		for _, x := range arr {
			f, ok := x.(float64)
			if !ok || f != math.Trunc(f) || f < 0 || f > maxDiasAviso {
				return fmt.Errorf("aviso.dias: use inteiros de 0 a %d", maxDiasAviso)
			}
			if n := int(f); !vistos[n] {
				vistos[n] = true
				dias = append(dias, n)
			}
		}
	}
	insistir, _ := m["insistir"].(bool)
	if len(dias) == 0 && !insistir {
		return errors.New("aviso: informe os dias ou marque insistir")
	}
	sort.Sort(sort.Reverse(sort.IntSlice(dias)))
	lista := make([]any, len(dias))
	for i, n := range dias {
		lista[i] = float64(n)
	}
	d["aviso"] = Doc{"dias": lista, "insistir": insistir}
	return nil
}

// ValidarSerie confere e normaliza d["serie"]: { id, indice, intervalo }. Liga as copias criadas por uma repeticao
// ("Prever" ou "Repetir" ao lancar): indice = posicao na serie e intervalo = meses entre uma ocorrencia e a seguinte.
// null remove o campo.
func ValidarSerie(d Doc) error {
	bruto, existe := d["serie"]
	if !existe {
		return nil
	}
	if bruto == nil {
		delete(d, "serie")
		return nil
	}
	m, ok := bruto.(map[string]any)
	if !ok {
		return errors.New("serie deve ser um objeto")
	}
	id, _ := m["id"].(string)
	if !ReID.MatchString(id) {
		return errors.New("serie.id invalido (6 a 40 letras minusculas ou numeros)")
	}
	inteiro := func(campo string, max float64) (float64, error) {
		f, ok := m[campo].(float64)
		if !ok || f != math.Trunc(f) || f < 1 || f > max {
			return 0, fmt.Errorf("serie.%s: use um inteiro de 1 a %d", campo, int(max))
		}
		return f, nil
	}
	indice, err := inteiro("indice", maxIndiceSerie)
	if err != nil {
		return err
	}
	intervalo, err := inteiro("intervalo", maxIntervaloSerie)
	if err != nil {
		return err
	}
	d["serie"] = Doc{"id": id, "indice": indice, "intervalo": intervalo}
	return nil
}
