package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"
)

// Lista de codigos de recuperacao do MFA (spec mfa-opcional, R25, R26 e R36). So regras puras: o banco e do repository e o sorteio
// e do service (que injeta a fonte de bytes aleatorios).

const (
	// AlfabetoCodigo tem 32 simbolos (5 bits cada), sem I, O, 0 e 1 (ambiguos). 20 simbolos dao 100 bits.
	AlfabetoCodigo = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	// TamanhoCodigo e quantos simbolos tem um codigo (sem os hifens da exibicao).
	TamanhoCodigo = 20
	// QuantidadeCodigos e quantos codigos tem uma lista.
	QuantidadeCodigos = 10
	// ValidadePendente e quanto tempo uma lista gerada e ainda nao confirmada pode ser confirmada.
	ValidadePendente = time.Hour
)

// ErrListaSubstituida: a lista pendente nao existe mais (outra geracao a substituiu, ou venceu) e a ativa nao e a pedida.
var ErrListaSubstituida = errors.New("lista substituida")

// CodigoGuardado e o que fica de um codigo: so o hash e se ja foi usado. O codigo em texto nunca e gravado.
type CodigoGuardado struct {
	Hash  string
	Usado bool
}

// ListaCodigos e uma lista de codigos de uma pessoa, ativa ou pendente. A pendente tem ExpiraEm; a ativa nao.
type ListaCodigos struct {
	ListaID  string
	GeradaEm time.Time
	ExpiraEm time.Time
	Codigos  []CodigoGuardado
}

// NovoCodigo sorteia um codigo de TamanhoCodigo simbolos a partir da fonte (crypto/rand em producao). Como o alfabeto tem 32 simbolos e um
// byte tem 256 valores, cada simbolo e o byte com os 5 bits de baixo: sem vies.
func NovoCodigo(fonte io.Reader) (string, error) {
	b := make([]byte, TamanhoCodigo)
	if _, err := io.ReadFull(fonte, b); err != nil {
		return "", err
	}
	out := make([]byte, TamanhoCodigo)
	for i, x := range b {
		out[i] = AlfabetoCodigo[x&31]
	}
	return string(out), nil
}

// NovoListaID sorteia o id de uma lista (16 bytes em hexadecimal); ele tambem e a chave do hash dos codigos dela.
func NovoListaID(fonte io.Reader) (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(fonte, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ListaIDValido diz se o texto tem o formato de um listaId (so letras minusculas e digitos, de 8 a 64): confere o que o cliente devolve.
func ListaIDValido(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// FormatarCodigo mostra o codigo em grupos de 4 separados por hifen (K7QM-4TXD-9WPB-2HRC-6NVF).
func FormatarCodigo(c string) string {
	var b strings.Builder
	for i, r := range c {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// NormalizarCodigo aceita o que a pessoa digita (maiusculas ou minusculas, com espacos e hifens) e devolve os 20 simbolos. ok=false se
// o formato nao e o de um codigo: tamanho ou simbolo fora do alfabeto.
func NormalizarCodigo(s string) (string, bool) {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if r == ' ' || r == '-' || r == '\t' {
			continue
		}
		b.WriteRune(r)
	}
	c := b.String()
	if len(c) != TamanhoCodigo {
		return "", false
	}
	for i := 0; i < len(c); i++ {
		if strings.IndexByte(AlfabetoCodigo, c[i]) < 0 {
			return "", false
		}
	}
	return c, true
}

// HashCodigo e o HMAC-SHA256 do codigo (ja normalizado) com o listaId como chave, em hexadecimal. Com 100 bits aleatorios nem hash lento
// nem pepper acrescentam seguranca real (spec mfa-opcional, R26); a chave por lista impede que um hash valha em outra lista.
func HashCodigo(listaID, codigoNormalizado string) string {
	m := hmac.New(sha256.New, []byte(listaID))
	m.Write([]byte(codigoNormalizado))
	return hex.EncodeToString(m.Sum(nil))
}
