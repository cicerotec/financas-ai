package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"financas/auth"
	"financas/domain"
)

// Spec mfa-opcional, R37 (GET /me/mfa): o estado do MFA e a lista de codigos, lidos pelo servidor. M-T33.

func (c *cenarioRec) estado(sub string, qs map[string]string) (int, string) {
	r, _ := c.srv.processar(context.Background(), &user{Sub: sub}, "GET", []string{"me", "mfa"}, qs, "")
	return r.StatusCode, r.Body
}

func estadoJSON(t *testing.T, corpo string) map[string]any {
	t.Helper()
	var d map[string]any
	if err := json.Unmarshal([]byte(corpo), &d); err != nil {
		t.Fatalf("corpo nao e JSON: %s", corpo)
	}
	return d
}

// M-T33: o estado vem do login (ligado) e do banco (lista ativa e a data de geracao), e so isso.
func TestEstadoMFALigadoComLista(t *testing.T) {
	c := novoRec(t)
	c.id.pessoas["ana"] = domain.PessoaMFA{Sub: "ana", Habilitada: true, TemMFA: true}
	codigos := c.lista(t, "ana")
	status, corpo := c.estado("ana", nil)
	esperar(t, "GET /me/mfa", status, 200)
	d := estadoJSON(t, corpo)
	lista, _ := d["lista"].(map[string]any)
	quer := relogioTeste.Agora().UTC().Format("2006-01-02T15:04:05Z07:00")
	if d["ligado"] != true || lista["ativa"] != true || lista["geradaEm"] != quer {
		t.Fatalf("esperava ligado, lista ativa gerada em %s: %s", quer, corpo)
	}
	if len(d) != 2 || len(lista) != 2 {
		t.Errorf("a resposta so leva ligado e lista{ativa, geradaEm}: %s", corpo)
	}
	// nada que identifique a pessoa ou que valha como segredo
	n, _ := domain.NormalizarCodigo(codigos[0])
	for _, proibido := range []string{"@", "hash", "listaId", "codigos", "email", n, domain.FormatarCodigo(n), mfaListaIDDe(c.f, t)} {
		if strings.Contains(corpo, proibido) {
			t.Errorf("a resposta leva %q: %s", proibido, corpo)
		}
	}
}

func TestEstadoMFADesligadoSemListaESoPendente(t *testing.T) {
	c := novoRec(t)
	c.id.pessoas["ana"] = domain.PessoaMFA{Sub: "ana", Habilitada: true, TemMFA: false}
	status, corpo := c.estado("ana", nil)
	esperar(t, "sem MFA e sem lista", status, 200)
	if corpo != `{"ligado":false,"lista":{"ativa":false}}` {
		t.Errorf("sem MFA e sem lista: %s", corpo)
	}
	c.pendente(t, "ana") // uma lista pendente nao e lista ativa
	_, corpo = c.estado("ana", nil)
	if corpo != `{"ligado":false,"lista":{"ativa":false}}` {
		t.Errorf("a pendente nao conta como ativa: %s", corpo)
	}
	c.id.pessoas["ana"] = domain.PessoaMFA{Sub: "ana", Habilitada: true, TemMFA: true}
	_, corpo = c.estado("ana", nil)
	if d := estadoJSON(t, corpo); d["ligado"] != true || d["lista"].(map[string]any)["ativa"] != false {
		t.Errorf("MFA ligado sem lista ativa: %s", corpo)
	}
}

// R37: a pessoa e a do token; sub, e-mail ou qualquer outra coisa na URL nao muda quem e lido.
func TestEstadoMFASoDaPessoaDoToken(t *testing.T) {
	c := novoRec(t)
	c.id.pessoas["ana"] = domain.PessoaMFA{Sub: "ana", Habilitada: true, TemMFA: true}
	c.id.pessoas["bia"] = domain.PessoaMFA{Sub: "bia", Habilitada: true, TemMFA: false}
	c.lista(t, "ana")
	_, corpo := c.estado("bia", map[string]string{"sub": "ana", "email": "ana@exemplo.com", "pessoa": "ana"})
	if d := estadoJSON(t, corpo); d["ligado"] != false || d["lista"].(map[string]any)["ativa"] != false {
		t.Errorf("a Bia nao pode ver o estado da Ana: %s", corpo)
	}
	if len(c.id.buscas) != 1 || c.id.buscas[0] != "bia" {
		t.Errorf("o login so deve ser consultado pelo sub do token: %v", c.id.buscas)
	}
}

// Falha do login ou do banco: 500 com a frase de sempre, sem estado pela metade e sem vazar o erro; a rota so le (nenhuma escrita).
func TestEstadoMFAFalhaViraErro500SemVazar(t *testing.T) {
	c := novoRec(t)
	c.id.pessoas["ana"] = domain.PessoaMFA{Sub: "ana", Habilitada: true, TemMFA: true}
	c.lista(t, "ana")
	buf := capturarLog(t)
	c.id.falharBusca = errors.New("cognito caiu com ana e segredo@exemplo.com")
	status, corpo := c.estado("ana", nil)
	if status != http.StatusInternalServerError || corpo != `{"erro":"erro interno"}` {
		t.Errorf("login fora do ar: %d %s", status, corpo)
	}
	c.id.falharBusca = nil
	c.db.falharLeitura2 = errors.New("dynamo caiu com ana")
	status, corpo = c.estado("ana", nil)
	if status != http.StatusInternalServerError || corpo != `{"erro":"erro interno"}` {
		t.Errorf("banco fora do ar: %d %s", status, corpo)
	}
	c.db.falharLeitura2 = nil
	delete(c.id.pessoas, "ana") // conta que o login nao acha: tambem e erro, nunca um "desligado" inventado
	if status, corpo = c.estado("ana", nil); status != http.StatusInternalServerError {
		t.Errorf("conta nao achada no login: %d %s", status, corpo)
	}
	if log := buf.String(); strings.Contains(log, "segredo@exemplo.com") || strings.Contains(log, "ana") {
		t.Errorf("o log leva dado da pessoa:\n%s", log)
	}
	c.id.pessoas["ana"] = domain.PessoaMFA{Sub: "ana", Habilitada: true, TemMFA: true}
	c.db.zerar()
	c.estado("ana", nil)
	if c.db.escritas != 0 || c.db.leituras != 1 {
		t.Errorf("a rota so le: esperava 1 leitura e nenhuma escrita, veio %d e %d", c.db.leituras, c.db.escritas)
	}
}

// Sem token e 401; so GET e rota; o rotulo do log e acao=mfa.ler rota=me; antes de o main ligar o login, a rota responde 500.
func TestEstadoMFARotaERotulo(t *testing.T) {
	c := novoRec(t)
	var r req
	r.RawPath = "/me/mfa"
	r.RequestContext.HTTP.Method = "GET"
	var rt rotulo
	if res, _ := c.srv.atender(context.Background(), r, &rt); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /me/mfa sem token respondeu %d, esperado 401", res.StatusCode)
	}
	for _, m := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		if a, ok := auth.RotaSemEspaco(m, []string{"me", "mfa"}); ok {
			t.Errorf("%s /me/mfa nao e rota (acao %q)", m, a)
		}
		if res, _ := c.srv.processar(context.Background(), &user{Sub: "ana"}, m, []string{"me", "mfa"}, nil, "{}"); res.StatusCode != http.StatusNotFound {
			t.Errorf("%s /me/mfa respondeu %d, esperado 404", m, res.StatusCode)
		}
	}
	if got := rotular("GET", []string{"me", "mfa"}); got != (rotulo{"mfa.ler", "me"}) {
		t.Errorf("rotular = %+v", got)
	}
	f := cenario(t)
	sem := novoServidor(f, "", relogioTeste, notifTeste, "")
	if res, _ := sem.processar(context.Background(), &user{Sub: "ana"}, "GET", []string{"me", "mfa"}, nil, ""); res.StatusCode != http.StatusInternalServerError {
		t.Errorf("sem identidade ligada: %d, esperado 500", res.StatusCode)
	}
}
