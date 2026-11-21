package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
)

// Vinculo de um canal por codigo (spec telegram-vinculo, R1 e R2). A pessoa pede um link no app; o canal devolve o codigo quando ela o
// toca. Regras puras: gerar, ler e conferir o codigo. Onde ele fica guardado e do repository; como o canal o entrega e do controller.

// ValidadeVinculo e quanto o codigo vale: 10 minutos e uma vez so (D5).
const ValidadeVinculo = 10 * time.Minute

// TamanhoCodigoVinculo: o sub empacotado (16 bytes) mais 16 bytes aleatorios, em base64 de URL sem preenchimento: 43 caracteres.
const TamanhoCodigoVinculo = 43

// ErrSubNaoUUID: o identificador da pessoa nao cabe empacotado em 16 bytes. Gerar recusa, sem gravar (P13).
var ErrSubNaoUUID = errors.New("o identificador da pessoa nao e um UUID")

var reUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// VinculoPendente e o que fica guardado de um codigo ainda nao usado: nunca o codigo, so o hash do aleatorio (R1).
type VinculoPendente struct {
	Sid      string // o espaco em que a pessoa pediu o link
	Hash     string // SHA-256 dos 16 bytes aleatorios, em hexadecimal
	ExpiraEm time.Time
	CriadoEm time.Time
}

// EnderecoLigado e o que a gravacao do endereco devolve: o apelido que a pessoa deu ao vinculo e o endereco que ele tinha antes ("" se nao tinha).
type EnderecoLigado struct {
	Apelido  string
	Anterior string
}

// HashDoAleatorio e o SHA-256, em hexadecimal, dos bytes aleatorios do codigo. Com 128 bits aleatorios um hash lento nao acrescenta seguranca (R2).
func HashDoAleatorio(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// NovoCodigoVinculo sorteia um codigo para a pessoa: o sub empacotado (nao e segredo) seguido de 16 bytes aleatorios (a barreira). Devolve o
// codigo, que so existe na resposta de quem o pediu, e o hash a guardar. ErrSubNaoUUID se o sub nao cabe em 16 bytes; nada e sorteado.
func NovoCodigoVinculo(sub string, fonte io.Reader) (codigo, hash string, err error) {
	if !reUUID.MatchString(sub) {
		return "", "", ErrSubNaoUUID
	}
	bytesDoSub, err := hex.DecodeString(strings.ReplaceAll(sub, "-", ""))
	if err != nil {
		return "", "", ErrSubNaoUUID
	}
	aleatorio := make([]byte, 16)
	if _, err := io.ReadFull(fonte, aleatorio); err != nil {
		return "", "", err
	}
	return base64.RawURLEncoding.EncodeToString(append(bytesDoSub, aleatorio...)), HashDoAleatorio(aleatorio), nil
}

// LerCodigoVinculo confere o formato do codigo (43 caracteres do alfabeto base64 de URL que decodificam em 32 bytes, sem normalizar a caixa) e
// devolve o sub e o hash do aleatorio. ok=false para qualquer coisa fora do formato: nada e lido do banco por causa dele.
func LerCodigoVinculo(codigo string) (sub, hash string, ok bool) {
	if len(codigo) != TamanhoCodigoVinculo {
		return "", "", false
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(codigo)
	if err != nil || len(b) != 32 {
		return "", "", false
	}
	h := hex.EncodeToString(b[:16])
	sub = h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
	return sub, HashDoAleatorio(b[16:]), true
}
