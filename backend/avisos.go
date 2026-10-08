package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Avisos de lancamentos futuros. O lancamento leva um campo opcional
//
//	aviso: { dias: [3, 1, 0], insistir: true }
//
// "dias" = quantos dias antes do vencimento avisar (0 = no dia); "insistir" = avisa no dia e todo dia depois, ate o
// lancamento deixar de ser evento futuro. O aviso so e enviado se o status for de evento futuro (configuracao
// statusFuturo) e o lancamento nao estiver oculto.
//
// Para a rotina diaria nao varrer a tabela, cada lancamento com aviso tem um item auxiliar na particao AVISOS#ATIVOS
// (SK = <espaco>#<id>), mantido ao criar, editar e excluir. A rotina le so essa particao e confere o lancamento de
// verdade antes de enviar.

const (
	pkAvisos       = "AVISOS#ATIVOS"
	maxDiasAviso   = 30
	maxItensAviso  = 8
	maxAtrasoAviso = 60 // depois disso o item auxiliar sai e o aviso para de insistir
)

// O Brasil nao tem horario de verao desde 2019, entao um fuso fixo basta (e dispensa o banco de fusos na Lambda).
var fusoBR = time.FixedZone("BRT", -3*3600)

// validarAviso confere e normaliza d["aviso"]; null remove o campo.
func validarAviso(d doc) error {
	bruto, existe := d["aviso"]
	if !existe {
		return nil
	}
	if bruto == nil {
		delete(d, "aviso")
		return nil
	}
	m, ok := bruto.(map[string]any)
	if !ok {
		return errors.New("aviso deve ser um objeto")
	}
	var dias []int
	vistos := map[int]bool{}
	if l, tem := m["dias"]; tem && l != nil {
		arr, ok := l.([]any)
		if !ok {
			return errors.New("aviso.dias deve ser uma lista")
		}
		if len(arr) > maxItensAviso {
			return fmt.Errorf("aviso.dias aceita no maximo %d itens", maxItensAviso)
		}
		for _, x := range arr {
			f, ok := x.(float64)
			if !ok || f != math.Trunc(f) || f < 0 || f > maxDiasAviso {
				return fmt.Errorf("aviso.dias: use inteiros de 0 a %d", maxDiasAviso)
			}
			if n := int(f); !vistos[n] {
				vistos[n] = true
				dias = append(dias, n)
			}
		}
	}
	insistir, _ := m["insistir"].(bool)
	if len(dias) == 0 && !insistir {
		return errors.New("aviso: informe os dias ou marque insistir")
	}
	sort.Sort(sort.Reverse(sort.IntSlice(dias)))
	lista := make([]any, len(dias))
	for i, n := range dias {
		lista[i] = float64(n)
	}
	d["aviso"] = doc{"dias": lista, "insistir": insistir}
	return nil
}

// lerAviso le o aviso ja gravado (numeros voltam como float64 do DynamoDB).
func lerAviso(d doc) (dias []int, insistir bool, ok bool) {
	m, tem := d["aviso"].(map[string]any)
	if !tem {
		return nil, false, false
	}
	if arr, _ := m["dias"].([]any); arr != nil {
		for _, x := range arr {
			if f, isNum := x.(float64); isNum {
				dias = append(dias, int(f))
			}
		}
	}
	insistir, _ = m["insistir"].(bool)
	return dias, insistir, len(dias) > 0 || insistir
}

// diaBR devolve o dia (AAAA-MM-DD) no fuso de Sao Paulo de uma dataEvento (UTC, como guardada).
func diaBR(dataEvento string) (string, error) {
	t, err := time.Parse(time.RFC3339Nano, dataEvento)
	if err != nil {
		return "", err
	}
	return t.In(fusoBR).Format("2006-01-02"), nil
}

func skAviso(sid, id string) string { return sid + "#" + id }

// sincronizarAviso mantem o item auxiliar de um lancamento: grava se ele tem aviso, apaga se nao tem.
// Falha aqui nao derruba o salvamento do lancamento (o erro vai para o log).
func sincronizarAviso(ctx context.Context, sid string, tx doc) {
	id, _ := tx["id"].(string)
	if id == "" {
		return
	}
	dias, insistir, tem := lerAviso(tx)
	if !tem {
		removerAviso(ctx, sid, id)
		return
	}
	data, _ := tx["dataEvento"].(string)
	venc, err := diaBR(data)
	if err != nil {
		log.Println("aviso: dataEvento invalida:", err)
		return
	}
	lista := make([]any, len(dias))
	for i, n := range dias {
		lista[i] = float64(n)
	}
	item, err := marshal(doc{"PK": pkAvisos, "SK": skAviso(sid, id), "sid": sid, "id": id, "dataEvento": data,
		"venc": venc, "dias": lista, "insistir": insistir})
	if err != nil {
		log.Println("aviso: marshal:", err)
		return
	}
	if _, err := db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &table, Item: item}); err != nil {
		log.Println("aviso: gravar item auxiliar:", err)
	}
}

func removerAviso(ctx context.Context, sid, id string) {
	if _, err := db.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: &table, Key: key(pkAvisos, skAviso(sid, id))}); err != nil {
		log.Println("aviso: apagar item auxiliar:", err)
	}
}

// ---------- decisao (pura) ----------

type kindAviso int

const (
	semAviso kindAviso = iota
	avisoAntes
	avisoHoje
	avisoAtrasado
)

func diaParaTempo(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.UTC)
}

// decidirAviso diz se hoje (AAAA-MM-DD) e dia de avisar um vencimento, e quantos dias faltam ou passaram.
func decidirAviso(hoje, venc string, dias []int, insistir bool) (kindAviso, int) {
	h, err1 := diaParaTempo(hoje)
	v, err2 := diaParaTempo(venc)
	if err1 != nil || err2 != nil {
		return semAviso, 0
	}
	diff := int(math.Round(v.Sub(h).Hours() / 24)) // >0 falta, 0 hoje, <0 atrasado
	switch {
	case diff == 0 && (insistir || contem(dias, 0)):
		return avisoHoje, 0
	case diff > 0 && contem(dias, diff):
		return avisoAntes, diff
	case diff < 0 && insistir && -diff <= maxAtrasoAviso:
		return avisoAtrasado, -diff
	}
	return semAviso, 0
}

func contem(l []int, n int) bool {
	for _, x := range l {
		if x == n {
			return true
		}
	}
	return false
}

// ---------- texto ----------

func moeda(v float64) string {
	s := strconv.FormatFloat(math.Abs(v), 'f', 2, 64)
	inteiro, frac, _ := strings.Cut(s, ".")
	var partes []string
	for len(inteiro) > 3 {
		partes = append([]string{inteiro[len(inteiro)-3:]}, partes...)
		inteiro = inteiro[:len(inteiro)-3]
	}
	partes = append([]string{inteiro}, partes...)
	r := "R$ " + strings.Join(partes, ".") + "," + frac
	if v < 0 {
		r = "-" + r
	}
	return r
}

func plural(n int, um, varios string) string {
	if n == 1 {
		return "1 " + um
	}
	return fmt.Sprintf("%d %s", n, varios)
}

func ddmm(dia string) string {
	t, err := diaParaTempo(dia)
	if err != nil {
		return dia
	}
	return t.Format("02/01")
}

type linhaAviso struct {
	kind      kindAviso
	n         int
	venc      string
	tipo      string
	descricao string
	valor     float64
	banco     string
}

func (l linhaAviso) texto() string {
	saida := l.tipo != "entrada"
	var quando string
	switch l.kind {
	case avisoAtrasado:
		if saida {
			quando = "Atrasado há " + plural(l.n, "dia", "dias")
		} else {
			quando = "Recebimento atrasado há " + plural(l.n, "dia", "dias")
		}
	case avisoHoje:
		if saida {
			quando = "Vence hoje"
		} else {
			quando = "A receber hoje"
		}
	default:
		if saida {
			quando = "Vence em " + plural(l.n, "dia", "dias")
		} else {
			quando = "A receber em " + plural(l.n, "dia", "dias")
		}
	}
	desc := strings.TrimSpace(l.descricao)
	if desc == "" {
		desc = "(sem descrição)"
	}
	return fmt.Sprintf("• %s (%s): %s — %s (%s)", quando, ddmm(l.venc), desc, moeda(l.valor), l.banco)
}

// montarResumo junta as linhas de uma pessoa: atrasados primeiro, depois hoje, depois os que ainda vao vencer.
func montarResumo(hoje string, linhas []linhaAviso) string {
	sort.SliceStable(linhas, func(i, j int) bool {
		a, b := linhas[i], linhas[j]
		if a.kind != b.kind {
			return a.kind > b.kind // atrasado > hoje > antes
		}
		return a.venc < b.venc
	})
	partes := []string{"🔔 Avisos de " + ddmm(hoje)}
	for _, l := range linhas {
		partes = append(partes, l.texto())
	}
	return strings.Join(partes, "\n")
}

// ---------- rotina diaria ----------

var statusFuturoPadrao = map[string]bool{"PREVISTO": true, "TRANSFERINDO": true, "CREDITANDO": true}

func ehStatusFuturo(listas doc, status string) bool {
	if m, ok := listas["statusFuturo"].(map[string]any); ok {
		v, _ := m[status].(bool)
		return v
	}
	return statusFuturoPadrao[status]
}

type contextoEspaco struct {
	listas   doc
	ownerSub string
}

// rodarAvisos e chamada uma vez por dia pelo agendamento. Devolve quantos resumos foram enviados.
func rodarAvisos(ctx context.Context, agora time.Time) (int, error) {
	hoje := agora.In(fusoBR).Format("2006-01-02")
	in := &dynamodb.QueryInput{
		TableName:              aws.String(table),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pkAvisos},
		},
	}
	var itens []doc
	for {
		out, err := db.Query(ctx, in)
		if err != nil {
			return 0, err
		}
		for _, it := range out.Items {
			var d doc
			if attributevalue.UnmarshalMap(it, &d) == nil {
				itens = append(itens, d)
			}
		}
		if out.LastEvaluatedKey == nil {
			break
		}
		in.ExclusiveStartKey = out.LastEvaluatedKey
	}

	espacos := map[string]*contextoEspaco{}
	chats := map[string]string{} // sub -> chat id ("" = sem vinculo)
	porChat := map[string][]linhaAviso{}

	chatDe := func(sid, sub string) string {
		k := sid + "|" + sub
		if c, ok := chats[k]; ok {
			return c
		}
		v, err := getItem(ctx, "USER#"+sub, "SPACE#"+sid)
		c := ""
		if err == nil && v != nil {
			c, _ = v["telegramChatId"].(string)
		}
		chats[k] = c
		return c
	}

	for _, aux := range itens {
		sid, _ := aux["sid"].(string)
		id, _ := aux["id"].(string)
		data, _ := aux["dataEvento"].(string)
		venc, _ := aux["venc"].(string)
		dias, insistir, _ := lerAviso(doc{"aviso": doc{"dias": aux["dias"], "insistir": aux["insistir"]}})
		kind, n := decidirAviso(hoje, venc, dias, insistir)
		if sid == "" || id == "" {
			continue
		}
		// vencido ha muito tempo: para de insistir e limpa
		if d, err := diaParaTempo(venc); err == nil {
			if h, _ := diaParaTempo(hoje); int(h.Sub(d).Hours()/24) > maxAtrasoAviso {
				removerAviso(ctx, sid, id)
				continue
			}
		}
		if kind == semAviso {
			continue
		}
		tx, err := getItem(ctx, spacePK(sid), txSK(data, id))
		if err != nil {
			log.Println("avisos: ler lancamento:", err)
			continue
		}
		if tx == nil { // excluido
			removerAviso(ctx, sid, id)
			continue
		}
		if oculto, _ := tx["oculto"].(bool); oculto {
			continue
		}
		esp := espacos[sid]
		if esp == nil {
			esp = &contextoEspaco{}
			esp.listas, _ = getItem(ctx, spacePK(sid), "CFG#LISTAS")
			if meta, _ := getItem(ctx, spacePK(sid), "META"); meta != nil {
				esp.ownerSub, _ = meta["ownerSub"].(string)
			}
			espacos[sid] = esp
		}
		status, _ := tx["status"].(string)
		if !ehStatusFuturo(esp.listas, status) {
			continue // ja resolvido (ou nunca foi futuro): nao avisa
		}
		valor, _ := tx["valor"].(float64)
		tipo, _ := tx["tipo"].(string)
		desc, _ := tx["descricao"].(string)
		banco, _ := tx["banco"].(string)
		linha := linhaAviso{kind: kind, n: n, venc: venc, tipo: tipo, descricao: desc, valor: valor, banco: banco}

		criador, _ := tx["criadoPor"].(string)
		vistos := map[string]bool{}
		for _, sub := range []string{criador, esp.ownerSub} {
			if sub == "" {
				continue
			}
			chat := chatDe(sid, sub)
			if chat == "" || vistos[chat] {
				continue
			}
			vistos[chat] = true
			porChat[chat] = append(porChat[chat], linha)
		}
	}

	enviados := 0
	for chat, linhas := range porChat {
		if err := enviarMensagem(ctx, chat, montarResumo(hoje, linhas)); err != nil {
			if errors.Is(err, errTelegramNaoConfigurado) {
				log.Println("avisos: Telegram nao configurado; nada enviado")
				return enviados, nil
			}
			log.Println("avisos: enviar:", err)
			continue
		}
		enviados++
	}
	log.Printf("avisos: %d item(ns) ativos, %d resumo(s) enviado(s)", len(itens), enviados)
	return enviados, nil
}
