package notificacao

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"financas/domain"
)

func linhasFicticias(n int, desc string) []domain.LinhaAviso {
	var l []domain.LinhaAviso
	for i := 0; i < n; i++ {
		l = append(l, domain.LinhaAviso{Kind: domain.AvisoAntes, N: 3, Venc: fmt.Sprintf("2026-10-%02d", 10+i%15), Tipo: "saida", Descricao: fmt.Sprintf("%s %03d", desc, i), Valor: 100 + float64(i), Banco: "ITAU"})
	}
	return l
}

func letras(s string) int { return len([]rune(s)) }

// D-T1: cabendo, e uma mensagem identica a MontarResumo; no limite exato, uma; um a mais, duas.
func TestPartesDoResumoCabendoSaiIgualAoDeHoje(t *testing.T) {
	r := domain.ResumoAvisos{Hoje: "2026-10-07", Marca: "[DEV] ", Linhas: linhasFicticias(5, "Conta")}
	inteiro := MontarResumo(r.Marca, r.Hoje, append([]domain.LinhaAviso(nil), r.Linhas...))
	if p := PartesDoResumo(r, LimiteTelegram); len(p) != 1 || p[0] != inteiro {
		t.Errorf("partes = %q, esperado o MontarResumo de hoje", p)
	}
	n := letras(inteiro)
	if p := PartesDoResumo(r, n); len(p) != 1 || p[0] != inteiro {
		t.Errorf("no limite exato deveria ser uma parte, foram %d", len(p))
	}
	if p := PartesDoResumo(r, n-1); len(p) < 2 {
		t.Errorf("um caractere acima do limite deveria dividir, foi %d parte(s)", len(p))
	}
}

// D-T2: varias partes, cabecalho com (i/n) e a marca em todas, ordem preservada, nada perdido nem repetido, lista de fora intacta.
func TestPartesDoResumoEmTabela(t *testing.T) {
	for _, n := range []int{40, 80, 200} {
		linhas := linhasFicticias(n, "Conta de teste")
		linhas = append(linhas, domain.LinhaAviso{Kind: domain.AvisoAtrasado, N: 2, Venc: "2026-10-05", Tipo: "saida", Descricao: "Atrasada", Valor: 1, Banco: "ITAU"})
		copia := append([]domain.LinhaAviso(nil), linhas...)
		r := domain.ResumoAvisos{Hoje: "2026-10-07", Marca: "[DEV] ", Linhas: linhas}
		partes := PartesDoResumo(r, 1000)
		if len(partes) < 2 {
			t.Fatalf("%d linhas: esperado mais de uma parte", n)
		}
		if !reflect.DeepEqual(linhas, copia) {
			t.Fatal("a lista de quem chamou foi reordenada")
		}
		esperado := strings.Split(MontarResumo(r.Marca, r.Hoje, append([]domain.LinhaAviso(nil), linhas...)), "\n")[1:]
		var juntas []string
		for i, p := range partes {
			cab, resto, _ := strings.Cut(p, "\n")
			if want := fmt.Sprintf("🔔 [DEV] Avisos de 07/10/2026 (%d/%d)", i+1, len(partes)); cab != want {
				t.Errorf("cabecalho = %q, esperado %q", cab, want)
			}
			if letras(p) > 1000 {
				t.Errorf("a parte %d tem %d letras", i+1, letras(p))
			}
			juntas = append(juntas, strings.Split(resto, "\n")...)
		}
		if !reflect.DeepEqual(juntas, esperado) {
			t.Errorf("%d linhas: a uniao das partes nao e a lista original, na mesma ordem", n)
		}
	}
}

// D-T3: linha sozinha acima do limite e truncada; o texto e contado em letras (🔔, acentos e —).
func TestPartesDoResumoTruncaLinhaGigante(t *testing.T) {
	longa := domain.LinhaAviso{Kind: domain.AvisoHoje, Venc: "2026-10-07", Tipo: "saida", Descricao: strings.Repeat("ç—", 3000), Valor: 1, Banco: "ITAU"}
	r := domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: []domain.LinhaAviso{longa, {Kind: domain.AvisoHoje, Venc: "2026-10-07", Tipo: "saida", Descricao: "curta", Valor: 1, Banco: "ITAU"}}}
	partes := PartesDoResumo(r, LimiteTelegram)
	if len(partes) < 2 {
		t.Fatalf("esperadas 2 partes ou mais, foram %d", len(partes))
	}
	for i, p := range partes {
		if letras(p) > LimiteTelegram {
			t.Errorf("a parte %d tem %d letras (limite %d)", i+1, letras(p), LimiteTelegram)
		}
	}
	if !strings.Contains(strings.Join(partes, "\n"), "…") || !strings.Contains(strings.Join(partes, "\n"), "curta") {
		t.Error("a linha gigante deveria ser truncada com … e a curta continuar")
	}
}

func TestTextoDeTesteTemAMarca(t *testing.T) {
	if p := PartesDoResumo(domain.ResumoAvisos{Marca: "[DEV] ", Teste: true}, LimiteTelegram); len(p) != 1 || !strings.HasPrefix(p[0], "🔔 [DEV] Teste do financas.ai") {
		t.Errorf("texto de teste = %q", p)
	}
}

// D-T1 (limite de producao): um resumo de ~4100 letras divide; com o limite subido para 10.000 isto falharia.
func TestResumoAcimaDoLimiteDoTelegramDivide(t *testing.T) {
	r := domain.ResumoAvisos{Hoje: "2026-10-07", Linhas: linhasFicticias(45, "Conta de teste com descricao um pouco longa")}
	if inteiro := MontarResumo("", r.Hoje, append([]domain.LinhaAviso(nil), r.Linhas...)); letras(inteiro) <= LimiteTelegram {
		t.Fatalf("o cenario deveria passar de %d letras, tem %d", LimiteTelegram, letras(inteiro))
	}
	if p := PartesDoResumo(r, LimiteTelegram); len(p) < 2 {
		t.Errorf("deveria dividir, foi %d parte(s)", len(p))
	}
}
