package repository

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Espacos le o vinculo da pessoa com os espacos e o nome de cada um.
//
//	vinculo   PK = USER#<sub>    SK = SPACE#<sid>   (role, apelido, telegramChatId)
//	espaco    PK = SPACE#<sid>   SK = META          (nome)
type Espacos struct {
	db    DynamoAPI
	table string
}

func NewEspacos(db DynamoAPI, table string) *Espacos { return &Espacos{db: db, table: table} }

func (r *Espacos) getItem(ctx context.Context, pk, sk string) (domain.Doc, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &r.table, Key: Key(pk, sk)})
	if err != nil || out.Item == nil {
		return nil, err
	}
	var d domain.Doc
	return d, attributevalue.UnmarshalMap(out.Item, &d)
}

// Papel devolve o papel da pessoa no espaco; "" = sem vinculo (sem acesso).
func (r *Espacos) Papel(ctx context.Context, sid, sub string) (string, error) {
	m, err := r.getItem(ctx, "USER#"+sub, "SPACE#"+sid)
	if err != nil || m == nil {
		return "", err
	}
	role, _ := m["role"].(string)
	return role, nil
}

// EspacosDe devolve os espacos a que a pessoa pertence. O nome vem do META; sem META (ou sem nome), o proprio id.
func (r *Espacos) EspacosDe(ctx context.Context, sub string) ([]domain.Espaco, error) {
	q, err := r.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :s)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "USER#" + sub},
			":s":  &types.AttributeValueMemberS{Value: "SPACE#"},
		},
	})
	if err != nil {
		return nil, err
	}
	res := []domain.Espaco{}
	for _, it := range q.Items {
		var d domain.Doc
		if attributevalue.UnmarshalMap(it, &d) != nil {
			continue
		}
		id := strings.TrimPrefix(d["SK"].(string), "SPACE#")
		nome := id
		if meta, _ := r.getItem(ctx, SpacePK(id), "META"); meta != nil {
			if n, ok := meta["nome"].(string); ok {
				nome = n
			}
		}
		res = append(res, domain.Espaco{ID: id, Nome: nome, Role: d["role"], Apelido: d["apelido"], TelegramChatID: d["telegramChatId"]})
	}
	return res, nil
}
