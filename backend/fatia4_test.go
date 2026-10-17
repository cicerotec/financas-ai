package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fatia 4 do item 32 (.ai/specs/arquitetura-camadas/spec-fatia-4.md): avisos e Telegram migraram para as camadas e a raiz
// ficou so com a ligacao. Os testes de avisos_test.go (decisao, texto, rotina, teste do aviso, marca [DEV]) seguem como a
// rede; aqui ficam as garantias novas.

// F4-T6: o "hoje" da rotina vem do Relogio injetado, no fuso de Sao Paulo. O cenario avisa 3 dias antes do vencimento (10/10).
func TestF4RotinaLeODiaPeloRelogio(t *testing.T) {
	cenarioAvisos(t)
	env := trocaEnvio(t, nil)
	// 07/10 02:00 UTC ainda e 06/10 em Sao Paulo (faltam 4 dias): nao e dia de avisar
	relogioEm(t, 2026, 10, 7, 2, 0)
	d, err := srvTeste.avisos.Rodar(context.Background())
	if err != nil || d["enviados"] != 0 {
		t.Fatalf("06/10 em SP: enviados=%v err=%v, esperado 0", d["enviados"], err)
	}
	// 07/10 04:00 UTC ja e 07/10 em Sao Paulo (faltam 3 dias): avisa, e o resumo traz o dia de Sao Paulo
	relogioEm(t, 2026, 10, 7, 4, 0)
	d, err = srvTeste.avisos.Rodar(context.Background())
	if err != nil || d["enviados"] != 2 {
		t.Fatalf("07/10 em SP: enviados=%v err=%v, esperado 2", d["enviados"], err)
	}
	for _, e := range *env {
		if !strings.Contains(e.texto, "Avisos de 07/10/2026") || !strings.Contains(e.texto, "Vence em 3 dias") {
			t.Errorf("resumo inesperado: %q", e.texto)
		}
	}
}

// Telegram nao configurado ou com falha de envio nao derruba a rotina (a regra continua a de antes).
func TestF4FalhaDeEnvioNaoDerrubaARotina(t *testing.T) {
	cenarioAvisos(t)
	trocaEnvio(t, errors.New("telegram respondeu 500"))
	if n, err := rodarAvisos(context.Background(), agoraBR("2026-10-10")); err != nil || n != 0 {
		t.Errorf("falha de envio: n=%d err=%v", n, err)
	}
}

// F4-T8: a raiz perdeu tudo: nenhum arquivo antigo existe e nao ha variavel global de dependencia.
func TestF4RaizSoLiga(t *testing.T) {
	for _, nome := range []string{"store.go", "avisos.go", "telegram.go", "perfil.go", "efeitos.go", "dominio_compat.go", "fechamento.go", "tags.go", "seq.go"} {
		if _, err := os.Stat(nome); err == nil {
			t.Errorf("%s ainda existe na raiz: migrou para as camadas", nome)
		}
	}
	proibidos := []string{"var db ", "var table", "ssmCli", "enviarMensagem", "ambienteApp", "tokenCache", "func initDB(", "func getItem(", "agora = time.Now"}
	filepath.Walk(".", func(caminho string, info os.FileInfo, err error) error {
		p := filepath.ToSlash(caminho)
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.Contains(p, "/") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, _ := os.ReadFile(caminho)
		for _, pr := range proibidos {
			if strings.Contains(string(b), pr) {
				t.Errorf("%s ainda tem %q: dependencia global ou ponte que a fatia 4 removeu", p, pr)
			}
		}
		return nil
	})
}

// F4-T9: o service nao conhece o SDK da AWS (nem o do SSM) e o de avisos le o dia so pelo Relogio.
func TestF4ServiceDeAvisosSemTimeNow(t *testing.T) {
	for _, nome := range []string{"service/avisos.go", "service/efeitos.go"} {
		b, err := os.ReadFile(nome)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "time.Now(") || strings.Contains(string(b), "aws-sdk") {
			t.Errorf("%s usa time.Now ou o SDK da AWS: o dia vem do Relogio e o envio do Notificador", nome)
		}
	}
}

// Quando o criador e o dono usam o mesmo chat, o chat recebe um resumo so, e a linha aparece uma vez.
func TestF4MesmoChatRecebeUmResumoSo(t *testing.T) {
	f, _ := cenarioAvisos(t)
	env := trocaEnvio(t, nil)
	f.colocar(t, doc{"PK": "USER#mem", "SK": "SPACE#s1", "role": "member", "telegramChatId": "111111"}) // o mesmo do dono
	n, err := rodarAvisos(context.Background(), agoraBR("2026-10-07"))
	if err != nil || n != 1 || len(*env) != 1 {
		t.Fatalf("enviados=%d mensagens=%d err=%v, esperado 1 e 1", n, len(*env), err)
	}
	if c := strings.Count((*env)[0].texto, "•"); c != 1 {
		t.Errorf("a linha deveria aparecer uma vez, apareceu %d:\n%s", c, (*env)[0].texto)
	}
}
