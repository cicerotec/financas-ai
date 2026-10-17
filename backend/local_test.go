package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEventoDeImitaAFunctionURL(t *testing.T) {
	r := httptest.NewRequest("PUT", "/spaces/s1/tx/abc?de=2026-01-01T10:00:00.000Z&q=a&q=b", strings.NewReader(`{"x":1}`))
	r.Header.Set("Authorization", "Bearer token")
	r.Header.Set("Content-Type", "application/json")
	ev := eventoDe(r, `{"x":1}`)
	if ev.RequestContext.HTTP.Method != "PUT" || ev.RawPath != "/spaces/s1/tx/abc" || ev.Body != `{"x":1}` {
		t.Errorf("metodo, caminho ou corpo errados: %+v", ev)
	}
	if ev.Headers["authorization"] != "Bearer token" || ev.Headers["content-type"] != "application/json" {
		t.Errorf("os cabecalhos tem de vir em minusculas, como na Function URL: %v", ev.Headers)
	}
	if ev.QueryStringParameters["de"] != "2026-01-01T10:00:00.000Z" || ev.QueryStringParameters["q"] != "a" {
		t.Errorf("query = %v", ev.QueryStringParameters)
	}
}

// Sem token a API local responde 401, como a da AWS (o handler e o mesmo).
func TestServidorLocalSemTokenDa401(t *testing.T) {
	s := httptest.NewServer(servidorLocal(novoServidor(novoBancoFake(), "", relogioTeste)))
	defer s.Close()
	res, err := http.Get(s.URL + "/me")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, esperado 401", res.StatusCode)
	}
	if res.Header.Get("content-type") != "application/json" {
		t.Errorf("content-type = %q", res.Header.Get("content-type"))
	}
}

func TestServidorLocalResponderPreflightDoCORS(t *testing.T) {
	s := httptest.NewServer(servidorLocal(novoServidor(novoBancoFake(), "", relogioTeste)))
	defer s.Close()
	r, _ := http.NewRequest(http.MethodOptions, s.URL+"/spaces/s1/tx", nil)
	r.Header.Set("Origin", "http://localhost:8080")
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent || res.Header.Get("Access-Control-Allow-Origin") != "http://localhost:8080" ||
		!strings.Contains(res.Header.Get("Access-Control-Allow-Headers"), "authorization") {
		t.Errorf("preflight: %d %v", res.StatusCode, res.Header)
	}
}
