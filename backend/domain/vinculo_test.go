package domain

import (
	"bytes"
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

const subDeTeste = "3f2a9c41-7b1d-4e08-9a55-0c6d2e8b1f73" // fictício

// V-T2: o codigo tem 43 caracteres de base64 de URL; os 16 primeiros bytes sao o sub empacotado e os 16 seguintes vem da fonte aleatoria.
func TestCodigoDeVinculoEOSubMaisAleatorio(t *testing.T) {
	fonte := bytes.NewReader(bytes.Repeat([]byte{0xAB}, 16))
	codigo, hash, err := NovoCodigoVinculo(subDeTeste, fonte)
	if err != nil {
		t.Fatal(err)
	}
	if len(codigo) != TamanhoCodigoVinculo || !regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`).MatchString(codigo) {
		t.Fatalf("codigo = %q, esperado 43 caracteres de base64 de URL", codigo)
	}
	b, err := base64.RawURLEncoding.DecodeString(codigo)
	if err != nil || len(b) != 32 {
		t.Fatalf("o codigo decodifica em %d bytes (%v), esperado 32", len(b), err)
	}
	if !bytes.Equal(b[16:], bytes.Repeat([]byte{0xAB}, 16)) {
		t.Errorf("os 16 bytes finais deveriam ser os da fonte aleatoria: %x", b[16:])
	}
	sub, hashLido, ok := LerCodigoVinculo(codigo)
	if !ok || sub != subDeTeste {
		t.Errorf("LerCodigoVinculo = (%q, %v), esperado o sub de volta", sub, ok)
	}
	if hash != hashLido || hash != HashDoAleatorio(bytes.Repeat([]byte{0xAB}, 16)) {
		t.Errorf("o hash e o SHA-256 do aleatorio: gerar=%q ler=%q", hash, hashLido)
	}
	if strings.Contains(hash, codigo) || len(hash) != 64 {
		t.Errorf("o hash nao e o codigo e tem 64 hexadecimais: %q", hash)
	}
}

// Dois codigos da mesma pessoa diferem (a fonte e o gerador criptografico em producao).
func TestDoisCodigosDaMesmaPessoaDiferem(t *testing.T) {
	a, ha, _ := NovoCodigoVinculo(subDeTeste, bytes.NewReader(bytes.Repeat([]byte{1}, 16)))
	b, hb, _ := NovoCodigoVinculo(subDeTeste, bytes.NewReader(bytes.Repeat([]byte{2}, 16)))
	if a == b || ha == hb {
		t.Error("codigos diferentes deveriam ter hashes diferentes")
	}
}

// Um sub que nao e UUID recusa gerar e nao consome a fonte.
func TestSubQueNaoEUUIDRecusaGerar(t *testing.T) {
	for _, sub := range []string{"", "mem", "3F2A9C41-7B1D-4E08-9A55-0C6D2E8B1F73", "3f2a9c41-7b1d-4e08-9a55", "3f2a9c41-7b1d-4e08-9a55-0c6d2e8b1f7z", subDeTeste + "0"} {
		fonte := bytes.NewReader(make([]byte, 16))
		if _, _, err := NovoCodigoVinculo(sub, fonte); err != ErrSubNaoUUID {
			t.Errorf("sub %q: erro = %v, esperado ErrSubNaoUUID", sub, err)
		}
		if fonte.Len() != 16 {
			t.Errorf("sub %q: a fonte foi lida antes de recusar", sub)
		}
	}
}

// V-T7: tudo que nao e exatamente um codigo (tamanho, alfabeto, bits sobrando) e recusado antes de qualquer leitura.
func TestCodigoForaDoFormatoEhRecusado(t *testing.T) {
	bom, _, _ := NovoCodigoVinculo(subDeTeste, bytes.NewReader(bytes.Repeat([]byte{7}, 16)))
	if _, _, ok := LerCodigoVinculo(bom); !ok {
		t.Fatal("o codigo gerado precisa ser lido")
	}
	ruins := []string{"", "abc", bom[:42], bom + "A", bom[:42] + "!", bom[:42] + "=", bom[:42] + "+", bom[:42] + "/", " " + bom[:42], bom[:42] + " ",
		bom[:42] + "\n", strings.Repeat("A", 44), strings.Repeat("-", 42)}
	for _, c := range ruins {
		if _, _, ok := LerCodigoVinculo(c); ok {
			t.Errorf("%q deveria ser recusado", c)
		}
	}
	// o ultimo caractere de 43 carrega 2 bits de preenchimento, que tem de ser zero: com "B" (01) nao e um codigo
	if _, _, ok := LerCodigoVinculo(bom[:42] + "B"); ok {
		t.Error("bits de preenchimento nao nulos deveriam ser recusados")
	}
}

// A caixa conta: o codigo em maiusculas, quando ainda tem o formato, e OUTRO codigo (outro sub), nunca o original.
func TestCaixaDoCodigoConta(t *testing.T) {
	bom, _, _ := NovoCodigoVinculo(subDeTeste, bytes.NewReader(bytes.Repeat([]byte{7}, 16)))
	if sub, _, ok := LerCodigoVinculo(strings.ToUpper(bom)); ok && sub == subDeTeste {
		t.Error("o codigo em maiusculas nao pode valer como o original")
	}
}
