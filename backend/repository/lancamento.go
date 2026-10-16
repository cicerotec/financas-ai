package repository

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Lancamentos implementa a persistencia de lancamentos (service.RepositorioLancamentos). E o unico lugar que
// monta a chave de um lancamento e as expressoes de escrita.
type Lancamentos struct {
	db    DynamoAPI
	table string
}

func NewLancamentos(db DynamoAPI, table string) *Lancamentos {
	return &Lancamentos{db: db, table: table}
}

// Buscar devolve nil (sem erro) quando o lancamento nao existe.
func (r *Lancamentos) Buscar(ctx context.Context, sid, dataEvento, id string) (domain.Lancamento, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &r.table, Key: Key(SpacePK(sid), TxSK(dataEvento, id))})
	if err != nil || out.Item == nil {
		return nil, err
	}
	var d domain.Lancamento
	return d, attributevalue.UnmarshalMap(out.Item, &d)
}

// Salvar poe PK e SK no proprio Depois (o service ainda usa o mesmo mapa depois de gravar). Sem Antes, so cria
// (falha se ja existir). Se a data mudou, a chave muda: apaga a antiga e grava a nova na mesma transacao.
func (r *Lancamentos) Salvar(ctx context.Context, sid string, m domain.Mudanca) error {
	d := m.Depois
	data, _ := d["dataEvento"].(string)
	id, _ := d["id"].(string)
	d["PK"], d["SK"] = SpacePK(sid), TxSK(data, id)
	item, err := attributevalue.MarshalMap(d)
	if err != nil {
		return domain.ErrDadosInvalidos
	}
	switch {
	case m.Antes == nil:
		_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &r.table, Item: item,
			ConditionExpression: aws.String("attribute_not_exists(PK)")})
	case d["SK"] == TxSK(m.DataAnterior, id):
		_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &r.table, Item: item})
	default: // dataEvento mudou: a chave muda, entao apaga a antiga e grava a nova juntas
		_, err = r.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
			{Delete: &types.Delete{TableName: &r.table, Key: Key(SpacePK(sid), TxSK(m.DataAnterior, id))}},
			{Put: &types.Put{TableName: &r.table, Item: item}},
		}})
	}
	return err
}

func (r *Lancamentos) Excluir(ctx context.Context, sid, dataEvento, id string) error {
	_, err := r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: &r.table, Key: Key(SpacePK(sid), TxSK(dataEvento, id))})
	return err
}

// GravarLote usa BatchWriteItem com nova tentativa dos itens nao processados (limite de escrita). Devolve o que
// sobrou quando o tempo da Lambda esta acabando.
func (r *Lancamentos) GravarLote(ctx context.Context, sid string, itens []domain.Lancamento) []domain.Lancamento {
	var reqs []types.WriteRequest
	var sobra []domain.Lancamento
	for _, d := range itens {
		data, _ := d["dataEvento"].(string)
		id, _ := d["id"].(string)
		d["PK"], d["SK"] = SpacePK(sid), TxSK(data, id)
		item, err := attributevalue.MarshalMap(d)
		if err != nil { // os valores vem de JSON e sempre serializam; mesmo assim, nunca some em silencio
			log.Println("lote: marshal:", err)
			sobra = append(sobra, d)
			continue
		}
		reqs = append(reqs, types.WriteRequest{PutRequest: &types.PutRequest{Item: item}})
	}
	for _, w := range r.gravarLote(ctx, reqs) {
		var d domain.Lancamento
		if attributevalue.UnmarshalMap(w.PutRequest.Item, &d) == nil {
			sobra = append(sobra, d)
		}
	}
	return sobra
}

func (r *Lancamentos) gravarLote(ctx context.Context, reqs []types.WriteRequest) []types.WriteRequest {
	limite := time.Now().Add(10 * time.Second)
	if prazo, ok := ctx.Deadline(); ok {
		limite = prazo.Add(-4 * time.Second)
	}
	espera := 200 * time.Millisecond
	for len(reqs) > 0 {
		saida, err := r.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{r.table: reqs}})
		if err != nil {
			log.Println("batch write:", err)
		} else {
			reqs = saida.UnprocessedItems[r.table]
		}
		if len(reqs) == 0 || time.Now().Add(espera).After(limite) {
			break
		}
		time.Sleep(espera)
		if espera < 2*time.Second {
			espera *= 2
		}
	}
	return reqs
}
