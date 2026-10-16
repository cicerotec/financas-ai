// Package domain tem os tipos e as regras puras que todas as camadas usam. Nao importa nada do projeto nem a AWS.
package domain

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"
)

// Doc e um item como mapa: lancamentos e configuracoes tem campos dinamicos (aviso, serie, extras do front).
type Doc = map[string]any

// Lancamento e um lancamento financeiro. As regras ficam em funcoes puras (ValidarTx, AlvoDe, MudaSaldo, TagsDe...).
type Lancamento = Doc

// Mudanca e uma escrita de lancamento entregue ao repository. E o que o service pede; as chaves do banco
// (PK, SK) e as expressoes do DynamoDB sao assunto so do repository.
type Mudanca struct {
	Antes        Lancamento // estado anterior; nil na criacao
	Depois       Lancamento // estado novo, ja validado
	DataAnterior string     // dataEvento sob a qual o lancamento estava gravado (chave antiga); vazio na criacao
}

// ErrDadosInvalidos: o lancamento nao pode ser serializado para gravar (nao acontece com valores vindos de JSON).
var ErrDadosInvalidos = errors.New("dados invalidos")

// Identidade identifica um lancamento dentro de um espaco sem expor a chave do banco.
func Identidade(l Lancamento) string {
	data, _ := l["dataEvento"].(string)
	id, _ := l["id"].(string)
	return data + "#" + id
}

// Protegidos tira do corpo os campos que so o servidor define.
func Protegidos(d Doc) {
	for _, k := range []string{"PK", "SK", "id", "criadoEm", "criadoPor"} {
		delete(d, k)
	}
}

// Publico limpa as chaves internas antes de devolver ao cliente.
func Publico(d Doc) Doc {
	delete(d, "PK")
	delete(d, "SK")
	return d
}

// CopiaRasa copia o mapa de primeiro nivel.
func CopiaRasa(d Doc) Doc {
	c := make(Doc, len(d))
	for k, v := range d {
		c[k] = v
	}
	return c
}

// EhDono: o registro foi criado por este usuario? Registro sem criadoPor nao e de ninguem (so o owner mexe nele).
func EhDono(reg Doc, sub string) bool {
	c, _ := reg["criadoPor"].(string)
	return sub != "" && c == sub
}

const alfa = "abcdefghijklmnopqrstuvwxyz0123456789"

// NovoID gera o id de 20 caracteres (minusculas e digitos) de um lancamento.
func NovoID() string {
	b := make([]byte, 20)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alfa))))
		b[i] = alfa[n.Int64()]
	}
	return string(b)
}

// NormData normaliza para o formato usado nas chaves: 2026-09-28T22:08:00.000Z (UTC).
func NormData(s string) (string, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return "", errors.New("dataEvento invalida (use ISO 8601)")
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z"), nil
}

// ValidarTx valida e normaliza os campos obrigatorios de um lancamento.
func ValidarTx(d Doc) error {
	s, _ := d["dataEvento"].(string)
	if s == "" {
		return errors.New("dataEvento obrigatoria")
	}
	n, err := NormData(s)
	if err != nil {
		return err
	}
	d["dataEvento"] = n
	// zero e negativo sao aceitos de proposito (lancamentos so de nota, estornos)
	if _, ok := d["valor"].(float64); !ok {
		return errors.New("valor deve ser numerico")
	}
	if t := d["tipo"]; t != "entrada" && t != "saida" {
		return errors.New("tipo deve ser entrada ou saida")
	}
	for _, c := range []string{"status", "banco"} {
		if s, _ := d[c].(string); s == "" {
			return errors.New(c + " obrigatorio")
		}
	}
	if err := ValidarAviso(d); err != nil {
		return err
	}
	return ValidarSerie(d)
}
