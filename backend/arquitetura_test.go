package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
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
		"adapter/notificacao": {"financas/service", "financas/controller", "financas/repository", "financas/auth"},
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

// AD-8 (.ai/bmad/arquitetura-avisos-canais.md): canal de aviso e vocabulario da API e da infraestrutura, nao da regra de negocio.
// O teste acima so olha imports e passou com ErrTelegram* no service; este olha NOMES: nenhum identificador ou texto (comentario
// nao conta) de domain/ e service/ pode ser, como palavra inteira, o nome de um canal. Identificador em camelCase e quebrado em
// palavras (ErrTelegramNaoConfigurado -> Err, Telegram, Nao, Configurado). As excecoes sao fixas e nomeadas.
func TestDomainEServiceNaoNomeiamCanaisDeAviso(t *testing.T) {
	proibidas := map[string]bool{"telegram": true, "fcm": true, "firebase": true, "push": true}
	excecoes := map[string]string{
		"domain/espaco.go":     "Espaco.TelegramChatID: o contrato de GET /me (AD-9)",
		"domain/perfil.go":     "MudancaPerfil.ChatID e o cadastro do chat id (AD-9)",
		"service/perfil.go":    "PUT /perfil recebe telegramChatId (AD-9)",
		"service/aparelhos.go": "registrar aparelho de push e do push por natureza (AD-5)",
	}
	for _, pacote := range []string{"domain", "service"} {
		arquivos, err := filepath.Glob(filepath.Join(pacote, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, arq := range arquivos {
			if strings.HasSuffix(arq, "_test.go") || excecoes[filepath.ToSlash(arq)] != "" {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), arq, nil, 0) // 0: sem comentarios na arvore
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				var texto string
				switch v := n.(type) {
				case *ast.Ident:
					texto = v.Name
				case *ast.BasicLit:
					if v.Kind == token.STRING {
						texto = v.Value
					}
				default:
					return true
				}
				for _, palavra := range palavrasDe(texto) {
					if proibidas[strings.ToLower(palavra)] {
						t.Errorf("%s nomeia o canal %q em %q: canal e coisa do controller, do adaptador e do repository (AD-8)", arq, palavra, texto)
					}
				}
				return true
			})
		}
	}
}

// palavrasDe quebra um identificador ou texto em palavras: separa por tudo que nao e letra ou numero e pela troca de minuscula para
// maiuscula (camelCase).
func palavrasDe(s string) []string {
	var palavras []string
	var atual []rune
	fecha := func() {
		if len(atual) > 0 {
			palavras = append(palavras, string(atual))
			atual = nil
		}
	}
	runas := []rune(s)
	for i, r := range runas {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			fecha()
			continue
		}
		if unicode.IsUpper(r) && i > 0 && unicode.IsLower(runas[i-1]) {
			fecha()
		}
		atual = append(atual, r)
	}
	fecha()
	return palavras
}
