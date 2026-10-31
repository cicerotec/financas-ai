package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Consulta salva da aba Tendencias (item 53, .ai/specs/tendencias-consultas): uma consulta de tags com nome, compartilhada no espaco.
// O servidor valida so o FORMATO (R3): nao le a gramatica da expressao nem confere se a tag existe. Nenhum numero entra no item.
const (
	MaxConsultas       = 50 // consultas vivas por espaco
	MaxNomeConsulta    = 40
	MaxGruposConsulta  = 4  // grupos ligados por OU
	MaxTagsConsulta    = 8  // tags em cada lista ("tem" e "nao") de um grupo
	MaxFichasConsulta  = 60 // fichas da expressao livre
	MaxTagConsulta     = 60 // caracteres de uma tag
	ModoGrupos         = "grupos"
	ModoAvancado       = "avancado"
	campoExpressaoSalv = "expressao"
)

var (
	visoesConsulta = map[string]bool{"gastos": true, "receitas": true, "saldo": true}
	opsConsulta    = map[string]bool{"E": true, "OU": true, "NAO": true}
)

// NomeConsulta confere o nome (1 a 40 caracteres, sem espaco nas pontas) e o devolve limpo.
func NomeConsulta(v any) (string, error) {
	s, ok := v.(string)
	s = strings.TrimSpace(s)
	if !ok || s == "" {
		return "", errors.New("de um nome para a consulta")
	}
	if utf8.RuneCountInString(s) > MaxNomeConsulta {
		return "", fmt.Errorf("o nome tem no maximo %d caracteres", MaxNomeConsulta)
	}
	return s, nil
}

// MesmoNomeConsulta diz se dois nomes sao o mesmo (sem diferenciar maiusculas).
func MesmoNomeConsulta(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// CorpoConsulta e o que o cliente pode mandar, ja conferido: so os campos aceitos, normalizados.
type CorpoConsulta struct {
	Nome      string // vazio = nao veio
	Consulta  Doc    // modo + (grupos | expressao) + visao, em bloco; nil = nao veio
	Versao    float64
	TemVersao bool
}

// ValidarCorpoConsulta confere o corpo de um POST (criar=true: nome e consulta obrigatorios, sem version) ou de um PUT (criar=false:
// version obrigatoria, nome e/ou consulta). Campo desconhecido, inclusive criadoPor, e erro.
func ValidarCorpoConsulta(corpo Doc, criar bool) (CorpoConsulta, error) {
	var c CorpoConsulta
	for k := range corpo {
		switch k {
		case "nome", "modo", "grupos", campoExpressaoSalv, "visao":
		case "version":
			if criar {
				return c, fmt.Errorf("campo desconhecido: %s", k)
			}
		default:
			return c, fmt.Errorf("campo desconhecido: %s", k)
		}
	}
	if !criar {
		v, ok := corpo["version"].(float64)
		if !ok || v < 1 {
			return c, errors.New("informe a version lida")
		}
		c.Versao, c.TemVersao = v, true
	}
	if v, tem := corpo["nome"]; tem {
		n, err := NomeConsulta(v)
		if err != nil {
			return c, err
		}
		c.Nome = n
	} else if criar {
		return c, errors.New("de um nome para a consulta")
	}
	_, temModo := corpo["modo"]
	_, temGrupos := corpo["grupos"]
	_, temExpr := corpo[campoExpressaoSalv]
	_, temVisao := corpo["visao"]
	if temModo || temGrupos || temExpr || temVisao || criar {
		q, err := validarBlocoConsulta(corpo)
		if err != nil {
			return c, err
		}
		c.Consulta = q
	}
	if !criar && c.Nome == "" && c.Consulta == nil {
		return c, errors.New("nada para alterar: informe o nome ou a consulta")
	}
	return c, nil
}

func validarBlocoConsulta(corpo Doc) (Doc, error) {
	modo, _ := corpo["modo"].(string)
	visao, _ := corpo["visao"].(string)
	if modo != ModoGrupos && modo != ModoAvancado {
		return nil, errors.New("modo invalido (grupos ou avancado)")
	}
	if !visoesConsulta[visao] {
		return nil, errors.New("visao invalida (gastos, receitas ou saldo)")
	}
	q := Doc{"modo": modo, "visao": visao}
	if modo == ModoGrupos {
		if _, tem := corpo[campoExpressaoSalv]; tem {
			return nil, errors.New("no modo grupos nao ha expressao")
		}
		g, err := validarGruposConsulta(corpo["grupos"])
		if err != nil {
			return nil, err
		}
		q["grupos"] = g
		return q, nil
	}
	if _, tem := corpo["grupos"]; tem {
		return nil, errors.New("no modo avancado nao ha grupos")
	}
	e, err := validarExpressaoConsulta(corpo[campoExpressaoSalv])
	if err != nil {
		return nil, err
	}
	q[campoExpressaoSalv] = e
	return q, nil
}

func tagConsulta(v any) (string, error) {
	s, ok := v.(string)
	s = strings.TrimSpace(s)
	if !ok || s == "" {
		return "", errors.New("tag vazia")
	}
	if utf8.RuneCountInString(s) > MaxTagConsulta {
		return "", fmt.Errorf("tag com mais de %d caracteres", MaxTagConsulta)
	}
	return s, nil
}

func validarGruposConsulta(v any) ([]any, error) {
	lista, ok := v.([]any)
	if !ok || len(lista) == 0 {
		return nil, errors.New("informe os grupos")
	}
	if len(lista) > MaxGruposConsulta {
		return nil, fmt.Errorf("no maximo %d grupos", MaxGruposConsulta)
	}
	out := make([]any, 0, len(lista))
	tags := 0
	for _, x := range lista {
		g, ok := x.(map[string]any)
		if !ok {
			return nil, errors.New("grupo invalido")
		}
		for k := range g {
			if k != "tem" && k != "nao" {
				return nil, fmt.Errorf("campo desconhecido no grupo: %s", k)
			}
		}
		ng := Doc{}
		for _, campo := range []string{"tem", "nao"} {
			bruto, tem := g[campo]
			arr := []any{}
			if tem {
				l, ok := bruto.([]any)
				if !ok {
					return nil, fmt.Errorf("%s deve ser uma lista de tags", campo)
				}
				if len(l) > MaxTagsConsulta {
					return nil, fmt.Errorf("no maximo %d tags em cada lista do grupo", MaxTagsConsulta)
				}
				for _, t := range l {
					s, err := tagConsulta(t)
					if err != nil {
						return nil, err
					}
					arr = append(arr, s)
				}
			}
			tags += len(arr)
			ng[campo] = arr
		}
		out = append(out, ng)
	}
	if tags == 0 {
		return nil, errors.New("escolha ao menos uma tag")
	}
	return out, nil
}

func validarExpressaoConsulta(v any) ([]any, error) {
	lista, ok := v.([]any)
	if !ok || len(lista) == 0 {
		return nil, errors.New("informe a expressao")
	}
	if len(lista) > MaxFichasConsulta {
		return nil, fmt.Errorf("a expressao passa de %d fichas", MaxFichasConsulta)
	}
	out := make([]any, 0, len(lista))
	tags := 0
	for _, x := range lista {
		f, ok := x.(map[string]any)
		if !ok {
			return nil, errors.New("ficha invalida")
		}
		for k := range f {
			if k != "k" && k != "v" {
				return nil, fmt.Errorf("campo desconhecido na ficha: %s", k)
			}
		}
		k, _ := f["k"].(string)
		switch k {
		case "tag":
			s, err := tagConsulta(f["v"])
			if err != nil {
				return nil, err
			}
			out = append(out, Doc{"k": "tag", "v": s})
			tags++
		case "op":
			op, _ := f["v"].(string)
			if !opsConsulta[op] {
				return nil, errors.New("operador invalido (E, OU ou NAO)")
			}
			out = append(out, Doc{"k": "op", "v": op})
		case "(", ")":
			if _, tem := f["v"]; tem {
				return nil, errors.New("parentese nao leva valor")
			}
			out = append(out, Doc{"k": k})
		default:
			return nil, errors.New("ficha invalida")
		}
	}
	if tags == 0 {
		return nil, errors.New("escolha ao menos uma tag")
	}
	return out, nil
}
