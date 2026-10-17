package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Contador guarda o contador de alteracoes do espaco: PK = SPACE#<sid>, SK = SEQ, atributo seq. E um numero do DynamoDB
// (ate 38 digitos) e trafega como numero JSON (float64 exato ate 2^53); nunca vira inteiro de 32 bits.
type Contador struct {
	db    DynamoAPI
	table string
}

func NewContador(db DynamoAPI, table string) *Contador { return &Contador{db: db, table: table} }

const skSeq = "SEQ"

// Proximo soma 1 ao contador do espaco (criando-o se nao existir) e devolve o valor novo.
func (r *Contador) Proximo(ctx context.Context, sid string) (float64, error) {
	out, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        &r.table,
		Key:              Key(SpacePK(sid), skSeq),
		UpdateExpression: aws.String("ADD seq :um"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":um": &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return 0, err
	}
	var d domain.Doc
	if err := attributevalue.UnmarshalMap(out.Attributes, &d); err != nil {
		return 0, err
	}
	n, _ := d["seq"].(float64)
	return n, nil
}

// Atual devolve o contador sem mexer nele; 0 se ainda nao existe.
func (r *Contador) Atual(ctx context.Context, sid string) (float64, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &r.table, Key: Key(SpacePK(sid), skSeq)})
	if err != nil || out.Item == nil {
		return 0, err
	}
	var d domain.Doc
	if err := attributevalue.UnmarshalMap(out.Item, &d); err != nil {
		return 0, err
	}
	n, _ := d["seq"].(float64)
	return n, nil
}
