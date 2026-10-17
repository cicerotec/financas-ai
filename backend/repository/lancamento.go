package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Lancamentos implementa a persistencia de lancamentos (service.Repositorio). E o unico lugar que monta a chave de
// um lancamento e de uma versao do historico, e as expressoes de escrita.
//
//	lancamento vivo   PK = SPACE#<sid>   SK = TX#<dataEvento>#<id>
//	versao            PK = HIST#<sid>    SK = TX#<id>#<versao com 8 digitos>
//
// O historico so recebe versoes novas: toda versao e gravada com attribute_not_exists(SK) e nao ha metodo para
// editar nem apagar uma versao (o template.yaml ainda nega UpdateItem e DeleteItem em HIST#*).
type Lancamentos struct {
	db    DynamoAPI
	table string
}

func NewLancamentos(db DynamoAPI, table string) *Lancamentos {
	return &Lancamentos{db: db, table: table}
}

// HistPK e a PK do historico de um espaco. Separada da do espaco para o IAM poder negar edicao e exclusao nela.
func HistPK(sid string) string { return "HIST#" + sid }

// HistSK e a SK de uma versao: o id do lancamento (nao a chave viva, que muda com a data) e a versao com 8 digitos.
func HistSK(id string, versao int) string { return fmt.Sprintf("TX#%s#%08d", id, versao) }

// Buscar devolve o lancamento vivo, incluindo o excluido logicamente, ou nil (sem erro) se nao existe. A leitura e
// consistente: quem le para depois escrever (editar, excluir) nao pode ver o estado de antes da escrita anterior.
func (r *Lancamentos) Buscar(ctx context.Context, sid, dataEvento, id string) (domain.Lancamento, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &r.table, Key: Key(SpacePK(sid), TxSK(dataEvento, id)),
		ConsistentRead: aws.Bool(true)})
	if err != nil || out.Item == nil {
		return nil, err
	}
	var d domain.Lancamento
	return d, attributevalue.UnmarshalMap(out.Item, &d)
}

// UltimaVersao devolve o maior numero de versao gravado para o id (0 se o lancamento nao tem historico). A leitura e
// consistente: com uma leitura atrasada, duas edicoes seguidas (salvar e desfazer) dariam um falso conflito de versao.
func (r *Lancamentos) UltimaVersao(ctx context.Context, sid, id string) (int, error) {
	out, err := r.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &r.table,
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :s)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: HistPK(sid)},
			":s":  &types.AttributeValueMemberS{Value: "TX#" + id + "#"},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(1),
		ConsistentRead:   aws.Bool(true),
	})
	if err != nil || len(out.Items) == 0 {
		return 0, err
	}
	n, ok := out.Items[0]["versao"].(*types.AttributeValueMemberN)
	if !ok {
		return 0, errors.New("versao sem numero em " + HistSK(id, 0))
	}
	v, err := strconv.Atoi(n.Value)
	return v, err
}

// Salvar grava o lancamento e as suas versoes numa unica transacao: tudo ou nada. Poe PK e SK no proprio Depois (o
// service ainda usa o mesmo mapa depois de gravar). Sem Antes, so cria. Se a data mudou, a chave muda: a antiga sai
// na mesma transacao. Devolve domain.ErrConflitoVersao se outra escrita gravou a mesma versao antes.
func (r *Lancamentos) Salvar(ctx context.Context, sid string, m domain.Mudanca) error {
	itens, err := r.itens(sid, m)
	if err != nil {
		return err
	}
	return r.transacionar(ctx, itens)
}

// SalvarLote grava varias mudancas numa unica transacao (a importacao): se uma falhar, nenhuma entra.
func (r *Lancamentos) SalvarLote(ctx context.Context, sid string, ms []domain.Mudanca) error {
	var todos []types.TransactWriteItem
	for _, m := range ms {
		itens, err := r.itens(sid, m)
		if err != nil {
			return err
		}
		todos = append(todos, itens...)
	}
	return r.transacionar(ctx, todos)
}

func (r *Lancamentos) itens(sid string, m domain.Mudanca) ([]types.TransactWriteItem, error) {
	d := m.Depois
	data, _ := d["dataEvento"].(string)
	id, _ := d["id"].(string)
	d["PK"], d["SK"] = SpacePK(sid), TxSK(data, id)
	vivo, err := attributevalue.MarshalMap(d)
	if err != nil {
		return nil, domain.ErrDadosInvalidos
	}
	put := &types.Put{TableName: &r.table, Item: vivo}
	if m.Antes == nil {
		put.ConditionExpression = aws.String("attribute_not_exists(PK)")
	}
	itens := []types.TransactWriteItem{{Put: put}}
	if m.Antes != nil && d["SK"] != TxSK(m.DataAnterior, id) { // a data mudou: a chave antiga sai junto
		itens = append(itens, types.TransactWriteItem{Delete: &types.Delete{TableName: &r.table, Key: Key(SpacePK(sid), TxSK(m.DataAnterior, id))}})
	}
	for _, v := range m.Versoes {
		item := domain.Doc{"PK": HistPK(sid), "SK": HistSK(id, v.Numero), "versao": float64(v.Numero),
			"acao": string(v.Acao), "porSub": v.PorSub, "em": v.Em, "registro": v.Registro}
		if v.DeVersao > 0 {
			item["deVersao"] = float64(v.DeVersao)
		}
		if v.ConfirmouMesFechado { // so a versao em que a confirmacao era necessaria e foi dada; nunca o item vivo
			item["confirmouMesFechado"] = true
		}
		av, err := attributevalue.MarshalMap(item)
		if err != nil {
			return nil, domain.ErrDadosInvalidos
		}
		// nunca sobrescreve uma versao: se a SK ja existe, outra escrita chegou antes
		itens = append(itens, types.TransactWriteItem{Put: &types.Put{TableName: &r.table, Item: av,
			ConditionExpression: aws.String("attribute_not_exists(SK)")}})
	}
	return itens, nil
}

func (r *Lancamentos) transacionar(ctx context.Context, itens []types.TransactWriteItem) error {
	_, err := r.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: itens})
	var cancelada *types.TransactionCanceledException
	if errors.As(err, &cancelada) {
		for _, motivo := range cancelada.CancellationReasons {
			if aws.ToString(motivo.Code) == "ConditionalCheckFailed" {
				return domain.ErrConflitoVersao
			}
		}
	}
	return err
}

// versaoDe le uma versao gravada no historico.
func versaoDe(item map[string]types.AttributeValue) (domain.Versao, error) {
	var d domain.Doc
	if err := attributevalue.UnmarshalMap(item, &d); err != nil {
		return domain.Versao{}, err
	}
	n, ok := d["versao"].(float64)
	if !ok {
		return domain.Versao{}, errors.New("versao sem numero em " + fmt.Sprint(d["SK"]))
	}
	v := domain.Versao{Numero: int(n)}
	v.Acao = domain.Acao(strOf(d["acao"]))
	v.PorSub, v.Em = strOf(d["porSub"]), strOf(d["em"])
	if dv, ok := d["deVersao"].(float64); ok {
		v.DeVersao = int(dv)
	}
	v.ConfirmouMesFechado, _ = d["confirmouMesFechado"].(bool)
	v.Registro, _ = d["registro"].(map[string]any)
	v.ID = strOf(v.Registro["id"])
	return v, nil
}

func strOf(x any) string { s, _ := x.(string); return s }

// consultaHistorico le as versoes de um prefixo de SK da particao do espaco, pagina por pagina, ate acabar ou
// juntar mais de "limite" (0 = sem teto). Devolve ate "limite" e se havia mais.
func (r *Lancamentos) consultaHistorico(ctx context.Context, sid, prefixoSK string, limite int) ([]domain.Versao, bool, error) {
	in := &dynamodb.QueryInput{
		TableName:              &r.table,
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :s)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: HistPK(sid)},
			":s":  &types.AttributeValueMemberS{Value: prefixoSK},
		},
		ConsistentRead: aws.Bool(true),
	}
	var vs []domain.Versao
	for {
		out, err := r.db.Query(ctx, in)
		if err != nil {
			return nil, false, err
		}
		for _, it := range out.Items {
			v, err := versaoDe(it)
			if err != nil {
				return nil, false, err
			}
			vs = append(vs, v)
		}
		if limite > 0 && len(vs) > limite {
			return vs[:limite], true, nil
		}
		if out.LastEvaluatedKey == nil {
			return vs, false, nil
		}
		in.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

// Versoes devolve todas as versoes do lancamento, da primeira para a ultima (vazio se nao tem historico).
func (r *Lancamentos) Versoes(ctx context.Context, sid, id string) ([]domain.Versao, error) {
	vs, _, err := r.consultaHistorico(ctx, sid, "TX#"+id+"#", 0)
	return vs, err
}

// VersoesDoEspaco devolve as versoes de todos os lancamentos do espaco, ate o limite; truncado diz que havia mais.
// Nao tem indice por data: le a particao inteira (um GSI por data entra depois, sem mudar o contrato).
func (r *Lancamentos) VersoesDoEspaco(ctx context.Context, sid string, limite int) ([]domain.Versao, bool, error) {
	return r.consultaHistorico(ctx, sid, "TX#", limite)
}

// ErrCursorInvalido: o cursor da paginacao nao e uma SK de lancamento.
var ErrCursorInvalido = errors.New("cursor invalido")

func codCursor(sk string) string { return base64.RawURLEncoding.EncodeToString([]byte(sk)) }

func decCursor(c string) (string, bool) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil || !strings.HasPrefix(string(b), "TX#") {
		return "", false
	}
	return string(b), true
}

// Listar devolve os lancamentos vivos do espaco, em paginas por SK (a data). Os excluidos logicamente nao saem. banco,
// status e texto filtram em memoria; proximo e o cursor da pagina seguinte ("" se acabou).
func (r *Lancamentos) Listar(ctx context.Context, sid string, f domain.FiltroTx) ([]domain.Doc, string, error) {
	a, b := "TX#", "TX#~"
	if f.De != "" {
		a = "TX#" + f.De
	}
	if f.Ate != "" {
		b = "TX#" + f.Ate
	}
	in := &dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("PK = :pk AND SK BETWEEN :a AND :b"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: SpacePK(sid)},
			":a":  &types.AttributeValueMemberS{Value: a},
			":b":  &types.AttributeValueMemberS{Value: b},
		},
		ScanIndexForward: aws.Bool(f.Asc),
		Limit:            aws.Int32(500),
	}
	if f.Cursor != "" {
		sk, ok := decCursor(f.Cursor)
		if !ok {
			return nil, "", ErrCursorInvalido
		}
		in.ExclusiveStartKey = Key(SpacePK(sid), sk)
	}
	q := strings.ToLower(f.Q)
	var res []domain.Doc
	for {
		out, err := r.db.Query(ctx, in)
		if err != nil {
			return nil, "", err
		}
		for _, it := range out.Items {
			var d domain.Doc
			if err := attributevalue.UnmarshalMap(it, &d); err != nil {
				return nil, "", err
			}
			if d["excluidoEm"] != nil { // excluido logicamente: some das listas, mas continua na tabela e no historico
				continue
			}
			if f.Banco != "" && d["banco"] != f.Banco {
				continue
			}
			if f.Status != "" && d["status"] != f.Status {
				continue
			}
			if q != "" {
				t, _ := d["descricao"].(string)
				n, _ := d["nota"].(string)
				if !strings.Contains(strings.ToLower(t+" "+n), q) {
					continue
				}
			}
			sk, _ := d["SK"].(string)
			res = append(res, domain.Publico(d))
			if len(res) >= f.Limite {
				return res, codCursor(sk), nil
			}
		}
		if out.LastEvaluatedKey == nil {
			return res, "", nil
		}
		in.ExclusiveStartKey = out.LastEvaluatedKey
	}
}
