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

// Acao diz o que uma versao do historico registra.
type Acao string

const (
	AcaoInsert  Acao = "insert"  // o lancamento nasceu
	AcaoUpdate  Acao = "update"  // mudou (inclui ocultar, ajustar fatura e mudar a data)
	AcaoDelete  Acao = "delete"  // exclusao logica
	AcaoRestore Acao = "restore" // uma versao antiga foi recuperada
)

// Versao e um ponto do historico de um lancamento. O Registro e o lancamento inteiro (sem chaves do banco).
type Versao struct {
	ID       string // id do lancamento a que a versao pertence
	Numero   int
	Acao     Acao
	PorSub   string // sub de quem agiu; vazio na versao 1 preguicosa (autor desconhecido)
	Em       string // ISO, UTC, com milissegundos
	DeVersao int    // so no restore: a versao de origem
	ConfirmouMesFechado bool // a confirmacao explicita de mes fechado por data era necessaria e foi dada (so nas versoes de acao)
	Registro Lancamento
}

// Mudanca e uma escrita de lancamento entregue ao repository. E o que o service pede; as chaves do banco
// (PK, SK) e as expressoes do DynamoDB sao assunto so do repository. O lancamento e as Versoes entram na mesma
// transacao: ou gravam os dois, ou nenhum.
type Mudanca struct {
	Antes        Lancamento // estado anterior; nil na criacao
	Depois       Lancamento // estado novo, ja validado
	DataAnterior string     // dataEvento sob a qual o lancamento estava gravado (chave antiga); vazio na criacao
	Versoes      []Versao   // o que vai para o historico junto com a escrita
}

var (
	// ErrDadosInvalidos: o lancamento nao pode ser serializado para gravar (nao acontece com valores vindos de JSON).
	ErrDadosInvalidos = errors.New("dados invalidos")
	// ErrConflitoVersao: outra escrita gravou a mesma versao antes (ou o lancamento ja existe). Nada foi gravado.
	ErrConflitoVersao = errors.New("alterado por outra pessoa")
)

// Registro devolve o lancamento como o historico o guarda: uma copia, sem as chaves do banco.
func Registro(l Lancamento) Lancamento {
	c := CopiaRasa(l)
	delete(c, "PK")
	delete(c, "SK")
	return c
}

// Excluido diz se o lancamento foi excluido logicamente (so os excluidos tem o campo excluidoEm).
func Excluido(l Lancamento) bool {
	_, tem := l["excluidoEm"]
	return tem && l["excluidoEm"] != nil
}

// VersoesDaEscrita decide o que vai para o historico numa escrita. ultima e a maior versao ja gravada do lancamento.
//   - lancamento novo (antes == nil): uma versao, com a acao dada;
//   - lancamento antigo, sem nenhuma versao: a versao 1 (estado de antes, insert, autor desconhecido) e a 2 (estado
//     de depois), na mesma transacao;
//   - lancamento com historico: so a proxima.
func VersoesDaEscrita(ultima int, antes, depois Lancamento, acao Acao, porSub, em string) []Versao {
	nova := func(n int) Versao {
		return Versao{Numero: n, Acao: acao, PorSub: porSub, Em: em, Registro: Registro(depois)}
	}
	if antes != nil && ultima == 0 {
		return []Versao{{Numero: 1, Acao: AcaoInsert, Em: em, Registro: Registro(antes)}, nova(2)}
	}
	return []Versao{nova(ultima + 1)}
}

// Identidade identifica um lancamento dentro de um espaco sem expor a chave do banco.
func Identidade(l Lancamento) string {
	data, _ := l["dataEvento"].(string)
	id, _ := l["id"].(string)
	return data + "#" + id
}

// Protegidos tira do corpo os campos que so o servidor define.
func Protegidos(d Doc) {
	for _, k := range []string{"PK", "SK", "id", "criadoEm", "criadoPor", "excluidoEm", "confirmaMesFechado", "confirmouMesFechado"} {
		delete(d, k)
	}
}

// Publico limpa as chaves internas (e o excluidoEm) antes de devolver ao cliente.
func Publico(d Doc) Doc {
	delete(d, "PK")
	delete(d, "SK")
	delete(d, "excluidoEm")
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
