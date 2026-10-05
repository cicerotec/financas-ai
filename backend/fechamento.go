package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"time"
	_ "time/tzdata" // fuso de Sao Paulo mesmo onde o sistema nao tem /usr/share/zoneinfo

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Fechamento mensal por banco.
//
// O saldo de um banco e uma cascata: o mes abre com o fechamento do anterior, desde a data inicial.
// Quem calcula e o navegador (so ele conhece quais status mexem no saldo); o servidor guarda e protege:
//
//   - caches: fechamento calculado de meses ja encerrados, so para nao reler o historico inteiro.
//     Qualquer escrita em lancamento marca "invalidoDe" e muda a version, entao o cache nunca sobrevive
//     a uma alteracao e um cache calculado antes dela nao consegue ser gravado depois.
//   - conferidos: o dono confirmou o mes com o extrato do banco. Mes conferido trava: nenhum lancamento
//     que mexa no saldo pode ser criado, alterado ou excluido nele nem nos meses anteriores, ate o dono
//     reabrir (do mais recente para o mais antigo).
//
// Um item por banco: PK=SPACE#<id>, SK=SALDO#<banco>. O mes e sempre o do fuso de Sao Paulo.

var fusoApp = func() *time.Location {
	l, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("BRT", -3*3600)
	}
	return l
}()

var (
	agora = time.Now // os testes trocam
	reMes = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)
)

func mesAtual() string { return agora().In(fusoApp).Format("2006-01") }

func mesDe(dataEvento string) string {
	t, err := time.Parse(time.RFC3339Nano, dataEvento)
	if err != nil {
		return ""
	}
	return t.In(fusoApp).Format("2006-01")
}

func skSaldo(banco string) string { return "SALDO#" + banco }

type alvo struct{ banco, mes string }

func alvoDe(d doc) (alvo, bool) {
	b, _ := d["banco"].(string)
	s, _ := d["dataEvento"].(string)
	m := mesDe(s)
	return alvo{b, m}, b != "" && m != ""
}

// campos que mudam o saldo de um banco; o resto (descricao, nota, tags...) pode ser editado em mes conferido
var camposSaldo = []string{"valor", "tipo", "status", "banco", "dataEvento"}

func mudaSaldo(antigo, novo doc) bool {
	for _, c := range camposSaldo {
		if !reflect.DeepEqual(antigo[c], novo[c]) {
			return true
		}
	}
	a, _ := antigo["oculto"].(bool)
	n, _ := novo["oculto"].(bool)
	return a != n
}

func copiaRasa(d doc) doc {
	c := make(doc, len(d))
	for k, v := range d {
		c[k] = v
	}
	return c
}

func lerSaldo(ctx context.Context, sid, banco string) (doc, error) {
	return getItem(ctx, spacePK(sid), skSaldo(banco))
}

func ultimoConferido(d doc) string {
	m, _ := d["conferidos"].(map[string]any)
	u := ""
	for k := range m {
		if k > u {
			u = k
		}
	}
	return u
}

func versaoDe(d doc) float64 {
	v, _ := d["version"].(float64)
	return v
}

// atualizarSaldo le, aplica fn e grava com controle otimista (tenta de novo se alguem gravou no meio).
func atualizarSaldo(ctx context.Context, sid, banco string, fn func(d doc) error) (doc, error) {
	for i := 0; i < 5; i++ {
		d, err := lerSaldo(ctx, sid, banco)
		if err != nil {
			return nil, err
		}
		if d == nil {
			d = doc{"banco": banco}
		}
		ver := versaoDe(d)
		if err := fn(d); err != nil {
			return nil, err
		}
		res, err := salvarCfg(ctx, spacePK(sid), skSaldo(banco), d, ver)
		if errors.Is(err, errConflito) {
			continue
		}
		return res, err
	}
	return nil, errConflito
}

// ---------- trava dos meses conferidos ----------

type travaErro struct{ banco, mes, ate string }

func (e *travaErro) Error() string {
	return fmt.Sprintf("o mes %s do banco %s esta conferido (ate %s) e travado: reabra o mes na aba Saldos para alterar", e.mes, e.banco, e.ate)
}

// travas consulta o ultimo mes conferido de cada banco, lendo cada banco uma vez so.
type travas struct {
	ctx     context.Context
	sid     string
	ate     map[string]string
	inicio  map[string]string // banco -> mes da data de inicio ("" = sem data de inicio)
	cfgLida bool
	cfg     doc
}

func novasTravas(ctx context.Context, sid string) *travas {
	return &travas{ctx: ctx, sid: sid, ate: map[string]string{}, inicio: map[string]string{}}
}

// mesDeInicio devolve o mes da data de inicio do banco (config de saldos), "" se nao houver.
func (t *travas) mesDeInicio(banco string) (string, error) {
	if m, ok := t.inicio[banco]; ok {
		return m, nil
	}
	if !t.cfgLida {
		d, err := getItem(t.ctx, spacePK(t.sid), "CFG#SALDOS")
		if err != nil {
			return "", err
		}
		t.cfg, t.cfgLida = d, true
	}
	m := ""
	por, _ := t.cfg["porBanco"].(map[string]any)
	if b, ok := por[banco].(map[string]any); ok {
		if di, _ := b["dataInicio"].(string); len(di) >= 7 && reMes.MatchString(di[:7]) {
			m = di[:7]
		}
	}
	t.inicio[banco] = m
	return m, nil
}

// anteriorAoInicio: lancamento de mes anterior ao inicio do controle do banco nao entra no saldo
// (e historico), entao nao precisa de trava nem invalida cache.
func (t *travas) anteriorAoInicio(a alvo) (bool, error) {
	ini, err := t.mesDeInicio(a.banco)
	return ini != "" && a.mes < ini, err
}

// checar devolve *travaErro se o mes do alvo esta travado.
func (t *travas) checar(a alvo) error {
	ate, ok := t.ate[a.banco]
	if !ok {
		d, err := lerSaldo(t.ctx, t.sid, a.banco)
		if err != nil {
			return err
		}
		ate = ultimoConferido(d)
		t.ate[a.banco] = ate
	}
	if ate != "" && a.mes <= ate {
		antes, err := t.anteriorAoInicio(a)
		if err != nil {
			return err
		}
		if !antes {
			return &travaErro{a.banco, a.mes, ate}
		}
	}
	return nil
}

// respTrava traduz o erro da trava em resposta HTTP; ok=false quando nao ha erro.
func respTrava(err error) (resp, bool) {
	if err == nil {
		return resp{}, false
	}
	var te *travaErro
	if errors.As(err, &te) {
		r, _ := out(http.StatusConflict, doc{"erro": te.Error(), "codigo": "mes_conferido",
			"banco": te.banco, "mes": te.mes, "conferidoAte": te.ate})
		return r, true
	}
	log.Println("trava:", err)
	r, _ := erro(500, "erro interno")
	return r, true
}

// invalidarSaldos marca que os caches do banco deixaram de valer a partir do mes do alvo
// e muda a version (barra caches calculados antes desta escrita).
func invalidarSaldos(ctx context.Context, sid string, alvos []alvo) {
	minimo := map[string]string{}
	tr := novasTravas(ctx, sid)
	for _, a := range alvos {
		if antes, err := tr.anteriorAoInicio(a); err == nil && antes {
			continue // historico anterior ao inicio nao muda nenhum saldo
		}
		if cur, ok := minimo[a.banco]; !ok || a.mes < cur {
			minimo[a.banco] = a.mes
		}
	}
	for banco, mes := range minimo {
		_, err := atualizarSaldo(ctx, sid, banco, func(d doc) error {
			if cur, _ := d["invalidoDe"].(string); cur == "" || mes < cur {
				d["invalidoDe"] = mes
			}
			return nil
		})
		if err != nil {
			// o lancamento ja foi gravado; sem isto o cache do banco pode ficar velho ate o proximo ajuste
			log.Printf("ERRO invalidar saldo sid=%s banco=%s mes=%s: %v", sid, banco, mes, err)
		}
	}
}

// ---------- rotas ----------

func getFechamentos(ctx context.Context, sid string) (resp, error) {
	q, err := db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :s)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: spacePK(sid)},
			":s":  &types.AttributeValueMemberS{Value: "SALDO#"},
		},
	})
	if err != nil {
		log.Println("fechamentos:", err)
		return erro(500, "erro interno")
	}
	itens := []doc{}
	for _, it := range q.Items {
		var d doc
		if attributevalue.UnmarshalMap(it, &d) == nil {
			itens = append(itens, publico(d))
		}
	}
	return out(200, doc{"itens": itens, "mesAtual": mesAtual()})
}

func conflitoVersao(atual doc) (resp, error) {
	if atual == nil {
		atual = doc{"version": 0.0}
	}
	return out(http.StatusConflict, doc{"erro": "alterado por outra pessoa", "atual": publico(atual)})
}

// putFechamentos grava o cache dos meses encerrados. Exige a version que o cliente leu ANTES de
// ler os lancamentos: se qualquer escrita aconteceu desde entao, 409 e o cliente recalcula.
func putFechamentos(ctx context.Context, sid string, body doc) (resp, error) {
	banco, _ := body["banco"].(string)
	if banco == "" {
		return erro(400, "banco obrigatorio")
	}
	ver, _ := body["version"].(float64)
	base, _ := body["base"].(string)
	bruto, _ := body["caches"].(map[string]any)
	caches := map[string]any{}
	for m, v := range bruto {
		n, ok := v.(float64)
		if !reMes.MatchString(m) || !ok || math.IsNaN(n) || math.IsInf(n, 0) {
			return erro(400, "caches invalidos")
		}
		if m < mesAtual() { // o mes em andamento nunca e cache
			caches[m] = n
		}
	}
	d, err := lerSaldo(ctx, sid, banco)
	if err != nil {
		return erro(500, "erro interno")
	}
	if versaoDe(d) != ver {
		return conflitoVersao(d)
	}
	if d == nil {
		d = doc{"banco": banco}
	}
	d["caches"], d["base"] = caches, base
	delete(d, "invalidoDe")
	res, err := salvarCfg(ctx, spacePK(sid), skSaldo(banco), d, ver)
	switch {
	case errors.Is(err, errConflito):
		atual, _ := lerSaldo(ctx, sid, banco)
		return conflitoVersao(atual)
	case err != nil:
		log.Println("put fechamentos:", err)
		return erro(500, "erro ao salvar")
	}
	return out(200, res)
}

// conferirMes registra a conferencia com o extrato e trava o mes (e os anteriores).
// O saldo calculado precisa bater com o extrato: a diferenca se resolve lancando o ajuste antes.
func conferirMes(ctx context.Context, sid, sub string, body doc) (resp, error) {
	banco, _ := body["banco"].(string)
	mes, _ := body["mes"].(string)
	saldo, ok1 := body["saldo"].(float64)
	real, ok2 := body["real"].(float64)
	ver, _ := body["version"].(float64)
	if banco == "" || !reMes.MatchString(mes) || !ok1 || !ok2 {
		return erro(400, "informe banco, mes (AAAA-MM), saldo calculado e saldo do extrato")
	}
	if mes > mesAtual() {
		return erro(400, "so da para conferir o mes atual ou meses anteriores")
	}
	if math.Abs(saldo-real) >= 0.005 {
		return erro(400, "o saldo calculado nao bate com o extrato: lance o ajuste antes de conferir")
	}
	d, err := lerSaldo(ctx, sid, banco)
	if err != nil {
		return erro(500, "erro interno")
	}
	if versaoDe(d) != ver {
		return conflitoVersao(d)
	}
	if u := ultimoConferido(d); u != "" && mes <= u {
		return out(http.StatusConflict, doc{"erro": "o mes " + mes + " nao pode ser conferido: o ultimo conferido e " + u, "codigo": "ordem"})
	}
	if d == nil {
		d = doc{"banco": banco}
	}
	conf, _ := d["conferidos"].(map[string]any)
	if conf == nil {
		conf = map[string]any{}
	}
	reg := map[string]any{"saldo": saldo, "real": real, "em": agora().UTC().Format("2006-01-02T15:04:05.000Z"), "por": sub}
	if aj, ok := body["ajuste"].(float64); ok && aj != 0 {
		reg["ajuste"] = aj
	}
	conf[mes] = reg
	d["conferidos"] = conf
	res, err := salvarCfg(ctx, spacePK(sid), skSaldo(banco), d, ver)
	switch {
	case errors.Is(err, errConflito):
		atual, _ := lerSaldo(ctx, sid, banco)
		return conflitoVersao(atual)
	case err != nil:
		log.Println("conferir:", err)
		return erro(500, "erro ao salvar")
	}
	return out(200, res)
}

var (
	errNaoConferido = errors.New("mes nao conferido")
	errNaoUltimo    = errors.New("nao e o ultimo conferido")
)

// reabrirMes destrava: so o ultimo mes conferido, para nunca sobrar conferencia por cima de mes aberto.
func reabrirMes(ctx context.Context, sid string, body doc) (resp, error) {
	banco, _ := body["banco"].(string)
	mes, _ := body["mes"].(string)
	if banco == "" || !reMes.MatchString(mes) {
		return erro(400, "informe banco e mes (AAAA-MM)")
	}
	var naoUltimo string
	res, err := atualizarSaldo(ctx, sid, banco, func(d doc) error {
		u := ultimoConferido(d)
		if u == "" {
			return errNaoConferido
		}
		if mes != u {
			naoUltimo = u
			return errNaoUltimo
		}
		conf, _ := d["conferidos"].(map[string]any)
		delete(conf, mes)
		d["conferidos"] = conf
		return nil
	})
	switch {
	case naoUltimo != "":
		return out(http.StatusConflict, doc{"erro": "reabra primeiro o mes " + naoUltimo + " (o mais recente conferido)", "codigo": "ordem"})
	case errors.Is(err, errNaoConferido):
		return erro(404, "esse mes nao esta conferido")
	case err != nil:
		log.Println("reabrir:", err)
		return erro(500, "erro ao salvar")
	}
	return out(200, res)
}
