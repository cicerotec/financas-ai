package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"financas/domain"
)

// ---------- falsos ----------

type repoAvisosFalso struct{ itens []domain.Doc }

func (r *repoAvisosFalso) Sincronizar(context.Context, string, domain.Doc) error { return nil }
func (r *repoAvisosFalso) Remover(context.Context, string, string) error         { return nil }
func (r *repoAvisosFalso) Ativos(context.Context) ([]domain.Doc, error)         { return r.itens, nil }

type lancFalso struct{ porId map[string]domain.Lancamento }

func (l *lancFalso) Buscar(_ context.Context, sid, _ string, id string) (domain.Lancamento, error) {
	return l.porId[sid+"|"+id], nil
}

type contatosFalso struct{ m map[string]map[string][]string } // sid|sub -> canal -> enderecos

func (c *contatosFalso) Destinos(_ context.Context, sid, sub string) (map[string][]string, error) {
	return c.m[sid+"|"+sub], nil
}

type donoFalso map[string]string

func (d donoFalso) Dono(_ context.Context, sid string) string { return d[sid] }

type relogioFalso struct{ t time.Time }

func (r relogioFalso) Agora() time.Time { return r.t }

type entregaFalsa struct {
	destino string
	resumo  domain.ResumoAvisos
}

// canalFalso guarda o resumo que recebeu (o conteudo, sem formato) ou falha com o erro dado.
type canalFalso struct {
	nome     string
	falhaCom error
	chamadas []string
	entregas []entregaFalsa
}

func (c *canalFalso) Nome() string { return c.nome }
func (c *canalFalso) Enviar(_ context.Context, destino string, r domain.ResumoAvisos) error {
	c.chamadas = append(c.chamadas, destino)
	if c.falhaCom != nil {
		return c.falhaCom
	}
	c.entregas = append(c.entregas, entregaFalsa{destino, r})
	return nil
}

// cenarioRotina monta uma rotina de avisos so com falsos. itens: cada um vira um aviso "vence hoje" (2026-10-10).
type itemRotina struct{ sid, id, criador, desc string }

func montarRotina(t *testing.T, itens []itemRotina, contatos map[string]map[string][]string, dono map[string]string, canais ...Canal) *Avisos {
	t.Helper()
	repo := &repoAvisosFalso{}
	lanc := &lancFalso{porId: map[string]domain.Lancamento{}}
	for _, it := range itens {
		repo.itens = append(repo.itens, domain.Doc{"sid": it.sid, "id": it.id, "dataEvento": "2026-10-10T12:00:00.000Z", "venc": "2026-10-10", "dias": []any{0.0}, "insistir": false})
		lanc.porId[it.sid+"|"+it.id] = domain.Lancamento{"previsto": true, "valor": 10.0, "tipo": "saida", "descricao": it.desc, "banco": "ITAU", "criadoPor": it.criador}
	}
	d, err := NovoDespachante(canais...)
	if err != nil {
		t.Fatal(err)
	}
	hoje, _ := time.ParseInLocation("2006-01-02 15:04", "2026-10-10 08:00", domain.FusoBR)
	return NewAvisos(repo, lanc, nil, &contatosFalso{contatos}, donoFalso(dono), d, relogioFalso{hoje}, "")
}

func capturarLog(t *testing.T) *bytes.Buffer {
	var b bytes.Buffer
	log.SetOutput(&b)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &b
}

// ---------- D-T4: um canal que falha nao derruba o outro ----------

func TestRotinaUmCanalFalhaEOOutroEntrega(t *testing.T) {
	a := &canalFalso{nome: "a", falhaCom: errors.New("recusou")}
	b := &canalFalso{nome: "b"}
	r := montarRotina(t, []itemRotina{{"s1", "x1", "", "Luz"}},
		map[string]map[string][]string{"s1|own": {"a": {"ea"}, "b": {"eb"}}}, map[string]string{"s1": "own"}, a, b)
	n, err := r.Rodar(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("Rodar = (%d, %v), esperado 1 entregue e sem erro", n, err)
	}
	if len(a.chamadas) != 1 || len(b.entregas) != 1 || b.entregas[0].destino != "eb" {
		t.Errorf("a=%v b=%v", a.chamadas, b.entregas)
	}
}

// ---------- D-T5: canal nao configurado pula o resto dele, com uma linha de log ----------

func TestRotinaCanalNaoConfiguradoPulaOResto(t *testing.T) {
	logs := capturarLog(t)
	a := &canalFalso{nome: "a", falhaCom: domain.ErrCanalNaoConfigurado}
	b := &canalFalso{nome: "b"}
	r := montarRotina(t, []itemRotina{{"s1", "x1", "mem", "Luz"}},
		map[string]map[string][]string{"s1|own": {"a": {"a1"}, "b": {"eb"}}, "s1|mem": {"a": {"a2"}}}, map[string]string{"s1": "own"}, a, b)
	n, err := r.Rodar(context.Background())
	if err != nil || n != 1 || len(b.entregas) != 1 {
		t.Fatalf("Rodar = (%d, %v); b=%v", n, err, b.entregas)
	}
	if len(a.chamadas) != 1 {
		t.Errorf("o canal a deveria ser tentado uma vez so, foi %v", a.chamadas)
	}
	if c := strings.Count(logs.String(), "canal a nao configurado"); c != 1 {
		t.Errorf("esperada 1 linha de log do canal a, houve %d:\n%s", c, logs)
	}
}

// ---------- D-T6: agrupamento por destino ----------

func TestRotinaAgrupaPorDestinoSemRepetirLinha(t *testing.T) {
	tg := &canalFalso{nome: "telegram"}
	contatos := map[string]map[string][]string{
		"s1|own": {"telegram": {"9"}}, "s1|mem": {"telegram": {"9"}}, // duas pessoas, o mesmo chat
		"s2|own": {"telegram": {"9"}}, // o mesmo chat em outro espaco
	}
	r := montarRotina(t, []itemRotina{{"s1", "x1", "mem", "Luz"}, {"s1", "x2", "own", "Agua"}, {"s2", "x3", "own", "Gas"}},
		contatos, map[string]string{"s1": "own", "s2": "own"}, tg)
	n, err := r.Rodar(context.Background())
	if err != nil || n != 1 || len(tg.entregas) != 1 {
		t.Fatalf("Rodar = (%d, %v); entregas=%v (esperado um resumo so, com o mesmo destino juntando os espacos)", n, err, tg.entregas)
	}
	contagem := map[string]int{}
	for _, l := range tg.entregas[0].resumo.Linhas {
		contagem[l.Descricao]++
	}
	for _, d := range []string{"Luz", "Agua", "Gas"} {
		if contagem[d] != 1 {
			t.Errorf("%s aparece %d vezes (esperado 1): %v", d, contagem[d], contagem)
		}
	}
}

func TestRotinaOrdemEstavelDosGrupos(t *testing.T) {
	var ordens [][]string
	for i := 0; i < 8; i++ {
		tg := &canalFalso{nome: "telegram"}
		contatos := map[string]map[string][]string{"s1|own": {"telegram": {"3", "1", "2"}}}
		r := montarRotina(t, []itemRotina{{"s1", "x1", "", "Luz"}}, contatos, map[string]string{"s1": "own"}, tg)
		if _, err := r.Rodar(context.Background()); err != nil {
			t.Fatal(err)
		}
		ordens = append(ordens, tg.chamadas)
	}
	for _, o := range ordens {
		if !reflect.DeepEqual(o, []string{"1", "2", "3"}) {
			t.Fatalf("ordem dos envios = %v, esperado 1,2,3 sempre", o)
		}
	}
}

// ---------- D-T12: configuracao do despachante ----------

func TestDespachanteRecusaNomeRepetidoECanalDesconhecidoNaoDerruba(t *testing.T) {
	if _, err := NovoDespachante(&canalFalso{nome: "a"}, &canalFalso{nome: "a"}); err == nil {
		t.Error("nome de canal repetido deveria ser recusado")
	}
	d, _ := NovoDespachante(&canalFalso{nome: "a"})
	if err := d.Entregar(context.Background(), Destino{"zzz", "1"}, domain.ResumoAvisos{}); !errors.Is(err, ErrCanalDesconhecido) {
		t.Errorf("canal desconhecido = %v", err)
	}
	b := &canalFalso{nome: "b"}
	r := montarRotina(t, []itemRotina{{"s1", "x1", "", "Luz"}},
		map[string]map[string][]string{"s1|own": {"zzz": {"1"}, "b": {"eb"}}}, map[string]string{"s1": "own"}, b)
	if n, err := r.Rodar(context.Background()); err != nil || n != 1 {
		t.Errorf("Rodar com destino de canal desconhecido = (%d, %v)", n, err)
	}
}

// ---------- D-T13: o log nao leva destino, texto nem valor ----------

func TestLogDaRotinaNaoVazaDestinoTextoNemValor(t *testing.T) {
	logs := capturarLog(t)
	a := &canalFalso{nome: "a", falhaCom: errors.New("recusou")}
	b := &canalFalso{nome: "b", falhaCom: domain.ErrCanalNaoConfigurado}
	c := &canalFalso{nome: "c"}
	r := montarRotina(t, []itemRotina{{"s1", "x1", "", "Descricao secreta"}},
		map[string]map[string][]string{"s1|own": {"a": {"CHAT-SECRETO"}, "b": {"CHAT-SECRETO"}, "c": {"CHAT-SECRETO"}, "zzz": {"CHAT-SECRETO"}}}, map[string]string{"s1": "own"}, a, b, c)
	if _, err := r.Rodar(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, proibido := range []string{"CHAT-SECRETO", "Descricao secreta", "10,00"} {
		if strings.Contains(logs.String(), proibido) {
			t.Errorf("o log contem %q:\n%s", proibido, logs)
		}
	}
	if !strings.Contains(fmt.Sprint(logs), "canal a") {
		t.Errorf("o log deveria nomear o canal:\n%s", logs)
	}
}
