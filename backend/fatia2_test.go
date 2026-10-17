package main

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Fatia 2 do item 32 (.ai/specs/arquitetura-camadas/spec-fatia-2.md): configuracoes, aparencia, tags, contador e perfil
// migraram para as camadas sem mudar o contrato HTTP.

// F2-T1: GET /cfg/{nome}.
func TestF2LerConfiguracao(t *testing.T) {
	f := cenario(t)
	st, d := chamar(t, "mem", "GET", "spaces/s1/cfg/listas", nil, "")
	esperar(t, "item existente", st, 200)
	if d["version"] != 1.0 || d["banco"] == nil {
		t.Errorf("listas = %v", d)
	}
	if d["PK"] != nil || d["SK"] != nil {
		t.Errorf("chaves internas vazaram: %v", d)
	}
	st, d = chamar(t, "mem", "GET", "spaces/s1/cfg/combos", nil, "")
	esperar(t, "item ausente", st, 200)
	if d["version"] != 0.0 || len(d) != 1 {
		t.Errorf("sem item deve ser so {version:0}: %v", d)
	}
	st, _ = chamar(t, "mem", "GET", "spaces/s1/cfg/inexistente", nil, "")
	esperar(t, "nome fora da lista", st, 404)

	f.falharLeitura = errors.New("fora do ar")
	// a leitura do papel tambem falha, entao o erro vem como 500 "erro interno" antes de chegar a configuracao
	st, d = chamar(t, "own", "GET", "spaces/s1/cfg/listas", nil, "")
	esperar(t, "erro de leitura", st, 500)
	if d["erro"] != "erro interno" {
		t.Errorf("erro = %v", d["erro"])
	}
}

// F2-T2: PUT /cfg/{nome} com controle otimista.
func TestF2GravarConfiguracao(t *testing.T) {
	f := cenario(t)
	st, d := chamar(t, "own", "PUT", "spaces/s1/cfg/combos", nil, `{"itens":[["a","b"]],"version":0}`)
	esperar(t, "primeira gravacao", st, 200)
	if d["version"] != 1.0 || d["PK"] != nil {
		t.Errorf("resposta = %v", d)
	}
	st, d = chamar(t, "own", "PUT", "spaces/s1/cfg/combos", nil, `{"itens":[["a","b"],["c"]],"version":1,"PK":"x","criadoPor":"forjado"}`)
	esperar(t, "versao certa", st, 200)
	if d["version"] != 2.0 {
		t.Errorf("version = %v", d["version"])
	}
	if got := f.ler(t, pkEspaco, "CFG#COMBOS"); got["PK"] != pkEspaco || got["criadoPor"] != nil {
		t.Errorf("campos protegidos entraram: %v", got)
	}

	// conflito: o cliente ainda esta na versao 1
	st, d = chamar(t, "own", "PUT", "spaces/s1/cfg/combos", nil, `{"itens":[],"version":1}`)
	esperar(t, "conflito", st, 409)
	atual, _ := d["atual"].(map[string]any)
	if d["erro"] != "alterado por outra pessoa" || atual["version"] != 2.0 || atual["PK"] != nil {
		t.Errorf("409 = %v", d)
	}

	f.falharPut = errors.New("fora do ar")
	st, d = chamar(t, "own", "PUT", "spaces/s1/cfg/combos", nil, `{"itens":[],"version":2}`)
	esperar(t, "erro do banco", st, 500)
	if d["erro"] != "erro ao salvar" {
		t.Errorf("erro = %v", d["erro"])
	}
}

// F2-T3: aparencia por pessoa, sem controle de versao.
func TestF2Aparencia(t *testing.T) {
	f := cenario(t)
	st, d := chamar(t, "mem", "GET", "spaces/s1/aparencia", nil, "")
	esperar(t, "sem item", st, 200)
	if d["version"] != 0.0 {
		t.Errorf("sem item: %v", d)
	}
	st, d = chamar(t, "mem", "PUT", "spaces/s1/aparencia", nil, `{"tema":"escuro","PK":"x"}`)
	esperar(t, "grava", st, 200)
	if d["tema"] != "escuro" || d["PK"] != nil || d["SK"] != nil {
		t.Errorf("resposta = %v", d)
	}
	if got := f.ler(t, pkEspaco, "USER#mem#CFG#APARENCIA"); got == nil || got["tema"] != "escuro" || got["PK"] != pkEspaco {
		t.Errorf("item da aparencia = %v", got)
	}
	_, d = chamar(t, "own", "GET", "spaces/s1/aparencia", nil, "")
	if d["tema"] != nil {
		t.Error("a aparencia de um nao pode aparecer para o outro")
	}
	f.falharPut = errors.New("fora do ar")
	st, d = chamar(t, "mem", "PUT", "spaces/s1/aparencia", nil, `{"tema":"claro"}`)
	esperar(t, "erro do banco", st, 500)
	if d["erro"] != "erro ao salvar" {
		t.Errorf("erro = %v", d["erro"])
	}
}

// F2-T4: tags, com conflito persistente.
func TestF2TagsConflitoPersistente(t *testing.T) {
	f := cenario(t)
	// a cada gravacao de CFG#TAGS, "outra pessoa" grava antes e sobe a versao: as 3 tentativas falham
	f.antesDoPut = func(in *dynamodb.PutItemInput) {
		if sv(in.Item["SK"]) != "CFG#TAGS" {
			return
		}
		atual := f.itens[chave(sv(in.Item["PK"]), "CFG#TAGS")]
		v, _ := strconv.ParseFloat(atual["version"].(*types.AttributeValueMemberN).Value, 64)
		atual["version"] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(v+1, 'f', -1, 64)}
	}
	st, d := chamar(t, "mem", "POST", "spaces/s1/tags", nil, `{"tags":["Nova"]}`)
	esperar(t, "conflito persistente", st, 500)
	if d["erro"] != "erro ao salvar tags" {
		t.Errorf("erro = %v", d["erro"])
	}
	// salvar um lancamento com tag nova nao e derrubado por isso: so vai ao log
	st, _ = chamar(t, "mem", "POST", "spaces/s1/tx", nil,
		`{"dataEvento":"2026-02-01T10:00:00.000Z","status":"PAGO","tipo":"saida","valor":5,"banco":"ITAU","descricao":"x","tags":["Praia"]}`)
	esperar(t, "lancamento salva mesmo assim", st, 201)
}

// F2-T5: o contador nao derruba a escrita quando falha: a resposta sai sem _seq.
func TestF2ContadorQuebradoNaoDerrubaAEscrita(t *testing.T) {
	f := cenario(t)
	f.falharContador = errors.New("fora do ar")
	st, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-02-10T12:00:00.000Z", "ITAU", 5))
	esperar(t, "criar", st, 201)
	if _, tem := d["_seq"]; tem {
		t.Errorf("sem contador a resposta nao leva _seq: %v", d["_seq"])
	}
	f.falharLeitura = errors.New("fora do ar")
	st, d = chamar(t, "own", "GET", "spaces/s1/seq", nil, "")
	esperar(t, "GET seq com banco fora", st, 500)
	if d["erro"] != "erro interno" {
		t.Errorf("erro = %v", d["erro"])
	}
}

// F2-T6: perfil (a rota exige o vinculo para ter papel; o 404 do UPDATE condicional vem chamando o controller direto).
func TestF2Perfil(t *testing.T) {
	f := cenario(t)
	st, d := chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"apelido":"Maria","telegramChatId":"123456789"}`)
	esperar(t, "grava os dois", st, 200)
	if d["apelido"] != "Maria" || d["telegramChatId"] != "123456789" {
		t.Errorf("resposta = %v", d)
	}
	vinc := f.ler(t, "USER#mem", "SPACE#s1")
	if vinc["apelido"] != "Maria" || vinc["telegramChatId"] != "123456789" || vinc["role"] != "member" {
		t.Errorf("vinculo = %v", vinc)
	}
	st, d = chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"telegramChatId":"abc"}`)
	esperar(t, "chat id invalido", st, 400)
	if d["erro"] != "o chat id do Telegram tem so numeros (ex.: 123456789)" {
		t.Errorf("erro = %v", d["erro"])
	}
	st, _ = chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"telegramChatId":123}`)
	esperar(t, "chat id nao e texto", st, 400)
	st, _ = chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{}`)
	esperar(t, "sem campos", st, 400)
	st, _ = chamar(t, "mem", "PUT", "spaces/s1/perfil", nil, `{"telegramChatId":""}`)
	esperar(t, "vazio remove", st, 200)
	if f.ler(t, "USER#mem", "SPACE#s1")["telegramChatId"] != nil {
		t.Error("o chat id deveria ter sido removido")
	}
	// sem vinculo: o UPDATE condicional (attribute_exists) recusa e o controller responde 404
	r, _ := srvTeste.perfil.PutPerfil(context.Background(), "s1", "sem-vinculo", doc{"apelido": "Fantasma"})
	if r.StatusCode != 404 || !strings.Contains(r.Body, "vinculo com o espaco nao encontrado") {
		t.Errorf("sem vinculo: %d %s", r.StatusCode, r.Body)
	}
	f.falharLeitura = nil
}

// F2-T7: a raiz perdeu os assuntos migrados.
func TestF2RaizSemOsAssuntosMigrados(t *testing.T) {
	for _, nome := range []string{"tags.go", "seq.go"} {
		if _, err := os.Stat(nome); err == nil {
			t.Errorf("%s ainda existe na raiz: migrou para service/ e repository/", nome)
		}
	}
	ler := func(nome string) string {
		b, err := os.ReadFile(nome)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	proibidos := []string{"func getCfg(", "func putCfg(", "func putAparencia(", "func putPerfil(", "func normalizarApelido(",
		"UpdateItemInput", "dynamodb.PutItemInput"}
	for _, nome := range []string{"main.go", "roteador.go"} {
		fonte := ler(nome)
		for _, p := range proibidos {
			if strings.Contains(fonte, p) {
				t.Errorf("%s ainda tem %q: migrou para as camadas", nome, p)
			}
		}
	}
	// a ponte salvarCfg do fechamento sumiu na fatia 3
	if strings.Contains(ler("main.go")+ler("roteador.go"), "func salvarCfg(") {
		t.Error("salvarCfg era a ponte do fechamento e saiu na fatia 3")
	}
}
