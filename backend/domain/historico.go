package domain

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

var reMes = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// MesValido confere o formato AAAA-MM de um mes.
func MesValido(m string) bool { return reMes.MatchString(m) }

// Diff compara dois registros e lista so os campos que mudaram, em ordem alfabetica, com o valor antes e depois
// (nil quando o campo nao existia). PK e SK nao entram. Sem mudanca devolve uma lista vazia, nunca nil.
func Diff(antes, depois Doc) []Doc {
	campos := map[string]bool{}
	for k := range antes {
		campos[k] = true
	}
	for k := range depois {
		campos[k] = true
	}
	delete(campos, "PK")
	delete(campos, "SK")
	nomes := make([]string, 0, len(campos))
	for k := range campos {
		nomes = append(nomes, k)
	}
	sort.Strings(nomes)
	mudancas := []Doc{}
	for _, k := range nomes {
		if !reflect.DeepEqual(antes[k], depois[k]) {
			mudancas = append(mudancas, Doc{"campo": k, "antes": antes[k], "depois": depois[k]})
		}
	}
	return mudancas
}

// MesmoConteudo diz se dois lancamentos tem o mesmo conteudo, sem olhar as chaves do banco nem a marca de exclusao.
func MesmoConteudo(a, b Lancamento) bool {
	ra, rb := Registro(a), Registro(b)
	delete(ra, "excluidoEm")
	delete(rb, "excluidoEm")
	return reflect.DeepEqual(ra, rb)
}

// acentos leva as letras acentuadas do portugues (e vizinhas) para a letra simples; roda depois de ToLower.
var acentos = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "å", "a", "ç", "c",
	"é", "e", "è", "e", "ê", "e", "ë", "e", "í", "i", "ì", "i", "î", "i", "ï", "i", "ñ", "n",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o", "ú", "u", "ù", "u", "û", "u", "ü", "u", "ý", "y", "ÿ", "y",
)

// Normalizar devolve o texto em minusculas e sem acento, para comparar sem diferenciar uma coisa da outra. O dado
// guardado nunca muda: so a comparacao.
func Normalizar(s string) string { return acentos.Replace(strings.ToLower(s)) }

// TermoBusca prepara o que a pessoa digitou na busca: normalizado, sem "#" no comeco (quem digita #mercado procura a tag)
// e sem espacos sobrando. E a mesma regra do front (termoBusca em app.js).
func TermoBusca(s string) string {
	return strings.Join(strings.Fields(strings.TrimLeft(Normalizar(s), "# \t\r\n")), " ")
}

// Periodo e o recorte do historico pela data da ALTERACAO (o "em" da versao): um mes (AAAA-MM), os ultimos N dias ou tudo.
// Exatamente um dos tres.
type Periodo struct {
	Mes  string
	Dias int
	Tudo bool
}

// MaxDiasHistorico e o maior periodo em dias que a consulta aceita (10 anos).
const MaxDiasHistorico = 3650

// EmDentroDeDias diz se o instante em (ISO) cai nos ultimos dias dias, contando a partir do INICIO do dia, em Sao Paulo,
// de "hoje menos dias", ate agora. Mesma janela do Historico de lancamentos no front.
func EmDentroDeDias(em string, agora time.Time, dias int) bool {
	t, err := time.Parse(time.RFC3339Nano, em)
	if err != nil {
		return false
	}
	hoje := agora.In(FusoApp)
	inicio := time.Date(hoje.Year(), hoje.Month(), hoje.Day()-dias, 0, 0, 0, 0, FusoApp)
	return !t.Before(inicio) && !t.After(agora)
}

// Contem diz se a versao, alterada em "em", cai no periodo.
func (p Periodo) Contem(em string, agora time.Time) bool {
	switch {
	case p.Tudo:
		return true
	case p.Dias > 0:
		return EmDentroDeDias(em, agora, p.Dias)
	default:
		return MesDe(em) == p.Mes
	}
}
