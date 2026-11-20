package domain

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// R26 (spec mfa-opcional): codigo de 20 simbolos de um alfabeto de 32 sem ambiguos (100 bits), em grupos de 4; HMAC-SHA256 com o listaId como chave.

func TestAlfabetoTem32SimbolosSemAmbiguos(t *testing.T) {
	if len(AlfabetoCodigo) != 32 {
		t.Fatalf("o alfabeto tem %d simbolos, precisa de 32 (5 bits cada)", len(AlfabetoCodigo))
	}
	for _, ruim := range "IO01ilo" {
		if strings.ContainsRune(AlfabetoCodigo, ruim) {
			t.Errorf("o alfabeto tem o simbolo ambiguo %q", ruim)
		}
	}
	seen := map[rune]bool{}
	for _, r := range AlfabetoCodigo {
		if seen[r] {
			t.Errorf("simbolo repetido %q", r)
		}
		seen[r] = true
	}
	if TamanhoCodigo*5 < 100 {
		t.Errorf("%d simbolos dao %d bits, o minimo e 100", TamanhoCodigo, TamanhoCodigo*5)
	}
}

// Sem vies: todos os 256 valores de byte caem no alfabeto, 8 vezes cada simbolo.
func TestNovoCodigoUsaTodoOByteSemVies(t *testing.T) {
	todos := make([]byte, 256)
	for i := range todos {
		todos[i] = byte(i)
	}
	contagem := map[byte]int{}
	for off := 0; off+TamanhoCodigo <= 256; off += TamanhoCodigo {
		c, err := NovoCodigo(bytes.NewReader(todos[off : off+TamanhoCodigo]))
		if err != nil || len(c) != TamanhoCodigo {
			t.Fatalf("NovoCodigo: %q %v", c, err)
		}
		for i := 0; i < len(c); i++ {
			if strings.IndexByte(AlfabetoCodigo, c[i]) < 0 {
				t.Fatalf("simbolo %q fora do alfabeto", c[i])
			}
			contagem[c[i]]++
		}
	}
	var min, max int = 1 << 30, 0
	for _, n := range contagem {
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	if max-min > 1 || len(contagem) != 32 {
		t.Errorf("distribuicao viesada: %d simbolos, min %d, max %d", len(contagem), min, max)
	}
	if _, err := NovoCodigo(bytes.NewReader([]byte("curto"))); err == nil {
		t.Error("fonte curta deveria dar erro")
	}
}

func TestFormatarENormalizarCodigo(t *testing.T) {
	c := "K7QM4TXD9WPB2HRC6NVF"
	f := FormatarCodigo(c)
	if f != "K7QM-4TXD-9WPB-2HRC-6NVF" {
		t.Fatalf("FormatarCodigo = %q", f)
	}
	for _, entrada := range []string{f, strings.ToLower(f), " k7qm 4txd 9wpb 2hrc 6nvf ", "K7QM4TXD9WPB2HRC6NVF", "k7qm-4txd-9wpb-2hrc-6nvf\t"} {
		if got, ok := NormalizarCodigo(entrada); !ok || got != c {
			t.Errorf("NormalizarCodigo(%q) = %q, %v", entrada, got, ok)
		}
	}
	for _, ruim := range []string{"", "K7QM-4TXD", f + "A", "K7QM-4TXD-9WPB-2HRC-6NV0", "K7QM-4TXD-9WPB-2HRC-6NVI", "K7QM-4TXD-9WPB-2HRC-6NV!"} {
		if got, ok := NormalizarCodigo(ruim); ok {
			t.Errorf("NormalizarCodigo(%q) = %q aceitou um formato invalido", ruim, got)
		}
	}
}

func TestHashUsaOListaIDComoChave(t *testing.T) {
	c := "K7QM4TXD9WPB2HRC6NVF"
	m := hmac.New(sha256.New, []byte("lista-a"))
	m.Write([]byte(c))
	if got, quer := HashCodigo("lista-a", c), hex.EncodeToString(m.Sum(nil)); got != quer {
		t.Errorf("HashCodigo = %s, esperado o HMAC-SHA256 com a chave listaId: %s", got, quer)
	}
	if HashCodigo("lista-a", c) != HashCodigo("lista-a", c) {
		t.Error("o hash precisa ser estavel")
	}
	if HashCodigo("lista-a", c) == HashCodigo("lista-b", c) {
		t.Error("o mesmo codigo em outra lista nao pode ter o mesmo hash")
	}
	if strings.Contains(HashCodigo("lista-a", c), c) {
		t.Error("o hash nao pode conter o codigo")
	}
}

func TestListaIDValido(t *testing.T) {
	for _, ok := range []string{"0123456789abcdef", "a1b2c3d4"} {
		if !ListaIDValido(ok) {
			t.Errorf("%q deveria ser valido", ok)
		}
	}
	for _, ruim := range []string{"", "curto", "ABCDEFGH12345678", "tem espaco 12345", "tem-hifen-12345", strings.Repeat("a", 65), "../../etc/passwd"} {
		if ListaIDValido(ruim) {
			t.Errorf("%q nao deveria ser valido", ruim)
		}
	}
}
