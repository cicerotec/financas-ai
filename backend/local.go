package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Modo local, so para desenvolvimento: com LOCAL_PORT definida, o main sobe um servidor HTTP em vez de chamar
// lambda.Start e entrega cada requisicao ao mesmo handler da Lambda. Na AWS a variavel nao existe e nada muda.
// Veja .ai/ambiente-de-teste.md ("Rodar a API na sua maquina").
//
// Travas, porque este modo usa as suas credenciais da AWS:
//   - so roda com AMBIENTE=dev (nunca contra a tabela de producao);
//   - escuta so em 127.0.0.1, nunca na rede.
//
// O token do Cognito continua sendo validado de verdade (pool e client de dev).

const limiteCorpoLocal = 6 << 20 // 6 MB, o teto de uma Function URL

// servidorLocal devolve o handler HTTP do modo local. Separado do rodarLocal para ser testado sem abrir porta.
func servidorLocal(s *Servidor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// o front em outra porta (localhost:8080) chama esta API: na AWS isso e o CORS da Function URL
		if origem := r.Header.Get("Origin"); origem != "" {
			w.Header().Set("Access-Control-Allow-Origin", origem)
			w.Header().Set("Access-Control-Allow-Headers", "authorization, content-type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		corpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limiteCorpoLocal))
		if err != nil {
			http.Error(w, "corpo grande demais", http.StatusRequestEntityTooLarge)
			return
		}
		res, err := s.handler(r.Context(), eventoDe(r, string(corpo)))
		if err != nil {
			log.Println("local: handler:", err)
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}
		for k, v := range res.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(res.StatusCode)
		_, _ = io.WriteString(w, res.Body)
	})
}

// eventoDe monta o evento que a Function URL entregaria: cabecalhos em minusculas, uma query por chave.
func eventoDe(r *http.Request, corpo string) req {
	cab := make(map[string]string, len(r.Header))
	for k, v := range r.Header {
		cab[strings.ToLower(k)] = strings.Join(v, ",")
	}
	qs := map[string]string{}
	for k, v := range r.URL.Query() {
		qs[k] = v[0]
	}
	var ev req
	ev.RawPath = r.URL.Path
	ev.RawQueryString = r.URL.RawQuery
	ev.Headers = cab
	ev.QueryStringParameters = qs
	ev.Body = corpo
	ev.RequestContext = events.LambdaFunctionURLRequestContext{HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{
		Method: r.Method, Path: r.URL.Path}}
	return ev
}

// rodarLocal abre a porta e nao volta (so se der erro).
func rodarLocal(porta string, s *Servidor, table string) {
	if os.Getenv("AMBIENTE") != "dev" {
		log.Fatal("modo local so roda com AMBIENTE=dev (ele usa as suas credenciais da AWS e nunca deve tocar a tabela de producao)")
	}
	endereco := "127.0.0.1:" + porta
	log.Printf("API local em http://%s (tabela %s). Ctrl+C para parar.", endereco, table)
	log.Fatal(http.ListenAndServe(endereco, servidorLocal(s)))
}
