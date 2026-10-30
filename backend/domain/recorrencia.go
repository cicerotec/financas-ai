package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"
)

// Recorrencia (item 51): um lancamento marcado com recorrente: { intervalo } e copiado sozinho pelo servidor, a cada intervalo de
// meses, ate o dono desligar. A conta das datas e feita aqui (pura), no fuso do app, como o front ja faz no Repetir.

const (
	MaxIntervaloRecorrencia = 120 // meses, como serie.intervalo

	tetoPassosRecorrencia = 2000 // trava de seguranca do laco que avanca ate depois de hoje
)

// IntervaloRecorrente le o recorrente de um lancamento ({ intervalo }). tem=false quando o campo nao existe ou e null.
func IntervaloRecorrente(bruto any) (intervalo int, tem bool, err error) {
	if bruto == nil {
		return 0, false, nil
	}
	m, ok := bruto.(map[string]any)
	if !ok {
		return 0, false, errors.New("recorrente deve ser um objeto")
	}
	f, ok := m["intervalo"].(float64)
	if !ok || f != math.Trunc(f) || f < 1 || f > MaxIntervaloRecorrencia {
		return 0, false, fmt.Errorf("recorrente.intervalo: use um inteiro de 1 a %d (meses)", MaxIntervaloRecorrencia)
	}
	return int(f), true, nil
}

// ValidarRecorrente confere e normaliza d["recorrente"]; null remove o campo. Transferencia nao e recorrente.
func ValidarRecorrente(d Doc) error {
	bruto, existe := d["recorrente"]
	if !existe {
		return nil
	}
	intervalo, tem, err := IntervaloRecorrente(bruto)
	if err != nil {
		return err
	}
	if !tem {
		delete(d, "recorrente")
		return nil
	}
	if par, _ := d["transferParId"].(string); par != "" {
		return errors.New("uma transferencia nao pode ser recorrente")
	}
	d["recorrente"] = Doc{"intervalo": float64(intervalo)}
	return nil
}

// DiaLocal devolve o dia (AAAA-MM-DD) de uma dataEvento no fuso do app.
func DiaLocal(dataEvento string) (string, error) {
	t, err := time.Parse(time.RFC3339Nano, dataEvento)
	if err != nil {
		return "", err
	}
	return t.In(FusoApp).Format("2006-01-02"), nil
}

// DiaDoMes devolve o dia do mes (1 a 31) de uma dataEvento no fuso do app.
func DiaDoMes(dataEvento string) (int, error) {
	t, err := time.Parse(time.RFC3339Nano, dataEvento)
	if err != nil {
		return 0, err
	}
	return t.In(FusoApp).Day(), nil
}

// SomaMesesRecorrencia devolve a dataEvento `meses` meses depois, no dia `dia` do mes (ou no ultimo, se o mes e mais curto) e com a
// mesma hora local. Parte sempre do dia original, para o 31 nao virar 28 para sempre (31/01 mensal: 28/02, 31/03, 30/04).
func SomaMesesRecorrencia(dataEvento string, dia, meses int) (string, error) {
	t, err := time.Parse(time.RFC3339Nano, dataEvento)
	if err != nil {
		return "", err
	}
	l := t.In(FusoApp)
	primeiro := time.Date(l.Year(), l.Month(), 1, l.Hour(), l.Minute(), l.Second(), l.Nanosecond(), FusoApp).AddDate(0, meses, 0)
	ultimo := time.Date(primeiro.Year(), primeiro.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if dia > ultimo {
		dia = ultimo
	}
	r := time.Date(primeiro.Year(), primeiro.Month(), dia, l.Hour(), l.Minute(), l.Second(), l.Nanosecond(), FusoApp)
	return r.UTC().Format("2006-01-02T15:04:05.000Z"), nil
}

// ProximaCopiaRecorrencia devolve a dataEvento da proxima copia: a primeira data, de `intervalo` em `intervalo` meses a partir de
// `base`, que fica **depois de hoje** (hoje e um dia AAAA-MM-DD). Nunca devolve data passada: se a base e antiga, ou a rotina
// ficou parada, os ciclos perdidos nao sao recuperados.
func ProximaCopiaRecorrencia(base string, dia, intervalo int, hoje string) (string, error) {
	atual := base
	for i := 1; i <= tetoPassosRecorrencia; i++ {
		prox, err := SomaMesesRecorrencia(base, dia, i*intervalo)
		if err != nil {
			return "", err
		}
		atual = prox
		d, err := DiaLocal(atual)
		if err != nil {
			return "", err
		}
		if d > hoje {
			return atual, nil
		}
	}
	return "", errors.New("recorrencia: data muito distante de hoje")
}

// IDCopiaRecorrencia e o id de uma copia, derivado da recorrencia e da data: a criacao e idempotente (rodar duas vezes, ou depois de
// uma falha no meio, cai no mesmo registro). Cumpre ReID.
func IDCopiaRecorrencia(idRecorrencia, dataEvento string) string {
	h := sha256.Sum256([]byte(idRecorrencia + "|" + dataEvento))
	return hex.EncodeToString(h[:])[:24]
}

// CopiaDeRecorrencia monta a copia de um registro: leva status, tipo, valor, descricao, nota, banco, tags e aviso, e o mesmo
// recorrente; nao leva serie. O id, o criadoPor e a data entram em quem cria.
func CopiaDeRecorrencia(origem Doc, dataEvento string) Doc {
	c := Doc{"dataEvento": dataEvento}
	for _, k := range []string{"status", "tipo", "valor", "descricao", "nota", "banco", "tags", "aviso", "recorrente"} {
		if v, ok := origem[k]; ok && v != nil {
			c[k] = v
		}
	}
	return c
}
