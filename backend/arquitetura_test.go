package main

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// F0-T4: a regra de dependencia das camadas (.ai/arquitetura.md). Cada pacote so pode importar o que esta a
// "baixo" dele; o controller nunca chega ao banco e o service nao conhece o SDK da AWS nem HTTP.
func TestCamadasRespeitamAsDependencias(t *testing.T) {
	const sdk = "github.com/aws/aws-sdk-go-v2"
	proibidos := map[string][]string{
		"domain":     {"financas/", "github.com/aws/", "net/http"},
		"service":    {"financas/controller", "financas/repository", "financas/auth", "github.com/aws/", "net/http"},
		"controller": {"financas/repository", "financas/auth", sdk},
		"repository": {"financas/service", "financas/controller", "financas/auth"},
		"adapter/relogio": {"financas/", "github.com/aws/", "net/http"},
		"auth":       {"financas/service", "financas/controller", "financas/repository", sdk},
	}
	for pacote, lista := range proibidos {
		arquivos, err := filepath.Glob(filepath.Join(pacote, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		fonte := 0
		for _, arq := range arquivos {
			if strings.HasSuffix(arq, "_test.go") {
				continue
			}
			fonte++
			f, err := parser.ParseFile(token.NewFileSet(), arq, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, im := range f.Imports {
				caminho, _ := strconv.Unquote(im.Path.Value)
				for _, p := range lista {
					if strings.HasPrefix(caminho, p) {
						t.Errorf("%s importa %s: o pacote %s nao pode depender disso", arq, caminho, pacote)
					}
				}
			}
		}
		if fonte == 0 {
			t.Errorf("pacote %s sem arquivos: o teste nao pode passar vazio", pacote)
		}
	}
}
