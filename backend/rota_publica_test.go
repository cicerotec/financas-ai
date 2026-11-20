package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"financas/controller"
)

// Spec mfa-opcional, AD-1 (a rota publica e uma etapa do roteador, antes do token): M-T27b a M-T27f e o rotulo do log.

const mfaRecuperarFalha = `{"erro":"recuperacao_invalida"}`

// recuperadorFalso grava o que o controller entregou ao service.
type recuperadorFalso struct {
	chamadas      int
	origem, email string
	codigo        string
	devolve       error
}

func (r *recuperadorFalso) Recuperar(_ context.Context, origem, email, codigo string) error {
	r.chamadas++
	r.origem, r.email, r.codigo = origem, email, codigo
	return r.devolve
}

func servidorComRecuperador(f *recuperadorFalso) *Servidor {
	s := novoServidor(novoBancoFake(), "", relogioTeste, notifTeste, "")
	s.recup = controller.NewRecuperacao(f)
	return s
}

func eventoPublico(corpo string, cab map[string]string, ip string) req {
	var r req
	r.RawPath = "/mfa/recuperar"
	r.Body = corpo
	r.Headers = cab
	r.RequestContext.HTTP.Method = "POST"
	r.RequestContext.HTTP.SourceIP = ip
	return r
}

// M-T27b: toda rota AUTENTICADA que o codigo reconhece, sem token (ou com token lixo), continua 401. So a publica muda.
func TestTodaRotaAutenticadaSemTokenSegue401(t *testing.T) {
	s := servidorComRecuperador(&recuperadorFalso{})
	reconhecidas := apiReconhecidas(t)
	if len(reconhecidas) < 40 {
		t.Fatalf("o colhedor achou so %d rotas: parou de entender o codigo", len(reconhecidas))
	}
	for chave := range reconhecidas {
		partes := strings.SplitN(chave, " /", 2)
		m, caminho := partes[0], "/"+partes[1]
		if apiEhPublica(m, strings.Split(partes[1], "/")) {
			continue
		}
		for _, cab := range []map[string]string{nil, {"authorization": "Bearer lixo"}, {"authorization": ""}} {
			var r req
			r.RawPath = caminho
			r.Headers = cab
			r.RequestContext.HTTP.Method = m
			var rt rotulo
			res, _ := s.atender(context.Background(), r, &rt)
			if res.StatusCode != http.StatusUnauthorized {
				t.Errorf("%s sem token (%v) respondeu %d, esperado 401", chave, cab, res.StatusCode)
			}
		}
	}
	// e as autenticadas fora do espaco, que o colhedor tambem ve
	for _, c := range []struct{ m, p string }{{"GET", "/me"}, {"POST", "/push/sair"}} {
		var r req
		r.RawPath = c.p
		r.RequestContext.HTTP.Method = c.m
		var rt rotulo
		if res, _ := s.atender(context.Background(), r, &rt); res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s sem token respondeu %d, esperado 401", c.m, c.p, res.StatusCode)
		}
	}
}

// M-T27c: POST /mfa/recuperar SEM authorization chega ao controller; com authorization lixo, vazio ou "expirado" a resposta e a MESMA;
// o cabecalho nem e lido (se VerificarToken rodasse, o lixo viraria 401).
func TestRotaPublicaChegaAoControllerSemLerOToken(t *testing.T) {
	f := &recuperadorFalso{devolve: errors.New("qualquer")}
	s := servidorComRecuperador(f)
	corpo := `{"email":"ana@exemplo.com","codigo":"K7QM-4TXD-9WPB-2HRC-6NVF"}`
	var base resp
	for i, cab := range []map[string]string{nil, {"authorization": ""}, {"authorization": "Bearer lixo"}, {"authorization": "Bearer eyJhbGciOiJSUzI1NiJ9.e30.expirado"}} {
		var rt rotulo
		res, err := s.atender(context.Background(), eventoPublico(corpo, cab, "203.0.113.9"), &rt)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode == http.StatusUnauthorized {
			t.Fatalf("a rota publica respondeu 401 com cabecalho %v: o token foi lido", cab)
		}
		if i == 0 {
			base = res
			continue
		}
		if res.StatusCode != base.StatusCode || res.Body != base.Body {
			t.Errorf("com cabecalho %v a resposta mudou: %d %s (sem cabecalho: %d %s)", cab, res.StatusCode, res.Body, base.StatusCode, base.Body)
		}
	}
	if f.chamadas != 4 {
		t.Errorf("o controller deveria ter chamado o service 4 vezes, chamou %d", f.chamadas)
	}
}

// M-T27d: corpo vazio, JSON invalido, corpo de 3 KiB e e-mail ausente respondem IGUAL a codigo errado (e nao chegam ao service).
func TestRotaPublicaEntradaMinimaRespondeIgualAFalha(t *testing.T) {
	f := &recuperadorFalso{devolve: errors.New("codigo errado")}
	s := servidorComRecuperador(f)
	chamar := func(corpo string) resp {
		var rt rotulo
		res, _ := s.atender(context.Background(), eventoPublico(corpo, nil, "203.0.113.9"), &rt)
		return res
	}
	errado := chamar(`{"email":"ana@exemplo.com","codigo":"ERRADO"}`)
	if errado.StatusCode != http.StatusBadRequest || errado.Body != mfaRecuperarFalha {
		t.Fatalf("falha esperada 400 %s, veio %d %s", mfaRecuperarFalha, errado.StatusCode, errado.Body)
	}
	antes := f.chamadas
	for nome, corpo := range map[string]string{
		"vazio":               "",
		"json invalido":       "{nao e json",
		"array":               `["a","b"]`,
		"3 KiB":               `{"email":"ana@exemplo.com","codigo":"` + strings.Repeat("A", 3072) + `"}`,
		"sem e-mail":          `{"codigo":"K7QM-4TXD-9WPB-2HRC-6NVF"}`,
		"e-mail vazio":        `{"email":"  ","codigo":"K7QM"}`,
		"sem codigo":          `{"email":"ana@exemplo.com"}`,
		"campo de outro tipo": `{"email":1,"codigo":2}`,
	} {
		if res := chamar(corpo); res.StatusCode != errado.StatusCode || res.Body != errado.Body || res.Headers["content-type"] != errado.Headers["content-type"] {
			t.Errorf("%s respondeu %d %s, esperado igual a codigo errado (%d %s)", nome, res.StatusCode, res.Body, errado.StatusCode, errado.Body)
		}
	}
	if f.chamadas != antes {
		t.Errorf("entrada invalida nao pode chegar ao service: %d chamadas a mais", f.chamadas-antes)
	}
}

// Sucesso e o unico caso diferente: 200 com corpo fixo.
func TestRotaPublicaSucessoEFixo(t *testing.T) {
	s := servidorComRecuperador(&recuperadorFalso{})
	var rt rotulo
	res, _ := s.atender(context.Background(), eventoPublico(`{"email":"ana@exemplo.com","codigo":"K7QM"}`, nil, "203.0.113.9"), &rt)
	if res.StatusCode != 200 || res.Body != `{"ok":true}` {
		t.Errorf("sucesso: %d %s", res.StatusCode, res.Body)
	}
}

// M-T27e: a origem entregue ao service vem de SourceIP; um x-forwarded-for (ou qualquer cabecalho) falso nao a muda.
func TestOrigemVemDoSourceIPNuncaDeCabecalho(t *testing.T) {
	f := &recuperadorFalso{}
	s := servidorComRecuperador(f)
	cab := map[string]string{"x-forwarded-for": "198.51.100.1", "forwarded": "for=198.51.100.2", "x-real-ip": "198.51.100.3", "client-ip": "198.51.100.4"}
	var rt rotulo
	if _, err := s.atender(context.Background(), eventoPublico(`{"email":"ana@exemplo.com","codigo":"K7QM"}`, cab, "203.0.113.9"), &rt); err != nil {
		t.Fatal(err)
	}
	if f.origem != "203.0.113.9" {
		t.Errorf("a origem foi %q, esperado o SourceIP 203.0.113.9", f.origem)
	}
}

// O modo local preenche SourceIP a partir de RemoteAddr, sem a porta (e sem olhar cabecalho).
func TestOrigemDeSemPorta(t *testing.T) {
	for in, quer := range map[string]string{"203.0.113.7:5555": "203.0.113.7", "[::1]:80": "::1", "127.0.0.1:1": "127.0.0.1", "semporta": "semporta"} {
		if got := origemDe(in); got != quer {
			t.Errorf("origemDe(%q) = %q, esperado %q", in, got, quer)
		}
	}
}

// M-T27f: o log da rota publica e acao=mfa.recuperar rota=mfa (nao sem-token) e nao leva e-mail, origem, codigo nem corpo.
func TestLogDaRotaPublica(t *testing.T) {
	buf := capturarLog(t)
	s := servidorComRecuperador(&recuperadorFalso{devolve: errors.New("falha interna")})
	r := eventoPublico(`{"email":"segredo@exemplo.com","codigo":"CODIGO-SECRETO"}`, map[string]string{"authorization": "Bearer lixo"}, "203.0.113.77")
	r.RequestContext.RequestID = "rid-1"
	if _, err := s.handler(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	log := buf.String()
	if !strings.Contains(log, "acao=mfa.recuperar rota=mfa") {
		t.Errorf("a linha de log nao diz acao=mfa.recuperar rota=mfa: %s", log)
	}
	if strings.Contains(log, "sem-token") {
		t.Errorf("a rota publica foi logada como sem-token: %s", log)
	}
	for _, proibido := range []string{"segredo@exemplo.com", "exemplo.com", "CODIGO-SECRETO", "203.0.113.77", "Bearer", "lixo", "email", "codigo"} {
		if strings.Contains(log, proibido) {
			t.Errorf("o log leva %q: %s", proibido, log)
		}
	}
}

// O rotulo da rota publica para o log e o indice de rotas.
func TestRotularRotaPublica(t *testing.T) {
	if got := rotular("POST", []string{"mfa", "recuperar"}); got != (rotulo{"mfa.recuperar", "mfa"}) {
		t.Errorf("rotular = %+v", got)
	}
	if got := rotular("GET", []string{"mfa", "recuperar"}); got != (rotulo{acaoDesconhecida, "-"}) {
		t.Errorf("GET nao e publica, rotular = %+v", got)
	}
}
