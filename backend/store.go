package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type doc = map[string]any

// dynamoAPI e o que o codigo usa do DynamoDB; os testes trocam por um banco em memoria.
type dynamoAPI interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
	BatchWriteItem(context.Context, *dynamodb.BatchWriteItemInput, ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
}

var (
	db    dynamoAPI
	table = os.Getenv("TABLE_NAME")
)

func initDB(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	db = dynamodb.NewFromConfig(cfg)
	return nil
}

func key(pk, sk string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: pk},
		"SK": &types.AttributeValueMemberS{Value: sk},
	}
}

func getItem(ctx context.Context, pk, sk string) (doc, error) {
	out, err := db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &table, Key: key(pk, sk)})
	if err != nil || out.Item == nil {
		return nil, err
	}
	var d doc
	return d, attributevalue.UnmarshalMap(out.Item, &d)
}

func marshal(d doc) (map[string]types.AttributeValue, error) { return attributevalue.MarshalMap(d) }

// limpa chaves internas antes de devolver ao cliente
func publico(d doc) doc {
	delete(d, "PK")
	delete(d, "SK")
	return d
}

func spacePK(sid string) string { return "SPACE#" + sid }

// papel do usuario no espaco; "" = sem acesso
func papel(ctx context.Context, u *user, sid string) (string, error) {
	m, err := getItem(ctx, "USER#"+u.Sub, "SPACE#"+sid)
	if err != nil || m == nil {
		return "", err
	}
	r, _ := m["role"].(string)
	return r, nil
}

const alfa = "abcdefghijklmnopqrstuvwxyz0123456789"

func novoID() string {
	b := make([]byte, 20)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alfa))))
		b[i] = alfa[n.Int64()]
	}
	return string(b)
}

// normaliza para o formato usado nas chaves: 2026-09-28T22:08:00.000Z (UTC)
func normData(s string) (string, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return "", errors.New("dataEvento invalida (use ISO 8601)")
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z"), nil
}

func txSK(data, id string) string { return "TX#" + data + "#" + id }

func codCursor(sk string) string { return base64.RawURLEncoding.EncodeToString([]byte(sk)) }
func decCursor(c string) (string, bool) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil || !strings.HasPrefix(string(b), "TX#") {
		return "", false
	}
	return string(b), true
}

type filtroTx struct {
	De, Ate, Banco, Q, Status string
	Asc                       bool
	Limite                    int
	Cursor                    string
}

func listarTx(ctx context.Context, sid string, f filtroTx) ([]doc, string, error) {
	a, b := "TX#", "TX#~"
	if f.De != "" {
		a = "TX#" + f.De
	}
	if f.Ate != "" {
		b = "TX#" + f.Ate
	}
	in := &dynamodb.QueryInput{
		TableName:              aws.String(table),
		KeyConditionExpression: aws.String("PK = :pk AND SK BETWEEN :a AND :b"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: spacePK(sid)},
			":a":  &types.AttributeValueMemberS{Value: a},
			":b":  &types.AttributeValueMemberS{Value: b},
		},
		ScanIndexForward: aws.Bool(f.Asc),
		Limit:            aws.Int32(500),
	}
	if f.Cursor != "" {
		sk, ok := decCursor(f.Cursor)
		if !ok {
			return nil, "", errors.New("cursor invalido")
		}
		in.ExclusiveStartKey = key(spacePK(sid), sk)
	}
	q := strings.ToLower(f.Q)
	var res []doc
	for {
		out, err := db.Query(ctx, in)
		if err != nil {
			return nil, "", err
		}
		for _, it := range out.Items {
			var d doc
			if err := attributevalue.UnmarshalMap(it, &d); err != nil {
				return nil, "", err
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
			res = append(res, publico(d))
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
