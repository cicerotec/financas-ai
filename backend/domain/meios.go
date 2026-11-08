package domain

import (
	"reflect"
	"time"
)

// Meios de pagamento (item 43): o `banco` gravado em cada lancamento. O dono desativa um meio a partir de uma data (CFG#LISTAS.ativoAte, um mapa
// nome -> AAAA-MM-DD, ausente = ativo) sem tocar no passado.

// DataValida diz se s e uma data AAAA-MM-DD (so o formato de 10 caracteres, sem hora).
func DataValida(s string) bool {
	if len(s) != len("2006-01-02") {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// HojeDe devolve o dia (AAAA-MM-DD) do instante dado, no fuso do app.
func HojeDe(t time.Time) string { return t.In(FusoApp).Format("2006-01-02") }

// MeioAtivo diz se o meio ainda e sugerido no dia hoje: sem ativoAte ou antes da data em que ele deixa de valer.
func MeioAtivo(ativoAte, hoje string) bool { return ativoAte == "" || hoje < ativoAte }

// ---------- Encerrar o meio (item 58) ----------
// Desativar e encerrar a conta: so a partir de hoje, com saldo zero e sem evento. Depois disso nada novo entra no meio; o passado fica como esta.

// ErroMeioDesativado: o meio nao recebe o lancamento. Meio e o nome; Removido diz que ele nem esta mais na lista; Motivo e a frase que a tela mostra.
type ErroMeioDesativado struct {
	Meio     string
	Removido bool
	Motivo   string
}

func (e *ErroMeioDesativado) Error() string {
	if e.Motivo != "" {
		return e.Motivo
	}
	if e.Removido {
		return "o meio foi removido: cadastre-o de novo em Listas para usar esse lancamento"
	}
	return "o meio esta desativado e nao recebe lancamentos: reative-o em Listas"
}

// AtivoAteDe le o mapa nome -> data de CFG#LISTAS.ativoAte.
func AtivoAteDe(listas Doc) map[string]string {
	out := map[string]string{}
	m, _ := listas["ativoAte"].(map[string]any)
	for k, v := range m {
		if s, ok := v.(string); ok && s != "" {
			out[k] = s
		}
	}
	return out
}

// MeioDesativado diz se o meio ja deixou de valer no dia hoje (tem ativoAte e hoje nao e antes dele). Meio fora do mapa e ativo.
func MeioDesativado(listas Doc, meio, hoje string) bool {
	ate := AtivoAteDe(listas)[meio]
	return ate != "" && !MeioAtivo(ate, hoje)
}

// MudaInicioDoMeio diz se a configuracao de saldos nova muda o saldo inicial ou a data de inicio do meio em relacao a gravada.
func MudaInicioDoMeio(antes, depois Doc, meio string) bool {
	a, d := ConfigBancoDe(antes, meio), ConfigBancoDe(depois, meio)
	return Arred(a.SaldoInicial) != Arred(d.SaldoInicial) || a.DataInicio != d.DataInicio
}

// MeioNaLista diz se o nome esta em CFG#LISTAS.banco.
func MeioNaLista(listas Doc, meio string) bool {
	l, _ := listas["banco"].([]any)
	for _, x := range l {
		if s, ok := x.(string); ok && s == meio {
			return true
		}
	}
	return false
}

// camposLivresEmMeioDesativado sao os unicos que um lancamento de meio desativado aceita mudar: nao mexem no saldo nem na fatura. excluirDoTotal e
// derivado pelo servidor; version e controle.
var camposLivresEmMeioDesativado = map[string]bool{"descricao": true, "nota": true, "tags": true, "excluirDoTotal": true, "version": true}

// SoMudaCamposLivres diz se antes -> depois so muda descricao, nota e tags (e pode tirar a marca recorrente). Compara o conteudo, nao a presenca:
// o front manda o registro inteiro e o que nao mudou nao conta.
func SoMudaCamposLivres(antes, depois Lancamento) bool {
	chaves := map[string]bool{}
	for k := range antes {
		chaves[k] = true
	}
	for k := range depois {
		chaves[k] = true
	}
	for k := range chaves {
		if camposLivresEmMeioDesativado[k] {
			continue
		}
		if k == "recorrente" && depois[k] == nil {
			continue
		}
		if !mesmoValor(k, antes[k], depois[k]) {
			return false
		}
	}
	return true
}

// vazioDeCampo diz se o valor equivale a "nao tem": o formulario manda null, "" ou 0 onde o gravado nao tem o campo.
func vazioDeCampo(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case bool:
		return !x
	case float64:
		return x == 0
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

// mesmoValor compara o gravado com o enviado: ausente, null e vazio valem o mesmo, e a data e comparada como instante (o formulario manda sem segundos).
func mesmoValor(chave string, a, b any) bool {
	if vazioDeCampo(a) && vazioDeCampo(b) {
		return true
	}
	if chave == "dataEvento" {
		sa, _ := a.(string)
		sb, _ := b.(string)
		na, ea := NormData(sa)
		nb, eb := NormData(sb)
		if ea == nil && eb == nil {
			return na == nb
		}
	}
	return reflect.DeepEqual(a, b)
}

// ImportavelEmMeioDesativado diz se um registro importado pode entrar num meio desativado: so o anterior ao inicio do controle do meio, que o
// saldo nao conta. Sem dataInicio, ou com data igual ou posterior, nao.
func ImportavelEmMeioDesativado(dataEvento, dataInicio string) bool {
	ini, ok := InicioDoControle(dataInicio)
	if !ok {
		return false
	}
	t, err := time.Parse(time.RFC3339Nano, dataEvento)
	if err != nil {
		return false
	}
	return t.Before(ini)
}
