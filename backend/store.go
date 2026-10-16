package main

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"financas/repository"
)

// dynamoAPI e o que o codigo usa do DynamoDB; os testes trocam por um banco em memoria.
type dynamoAPI = repository.DynamoAPI

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
	ssmCli = ssm.NewFromConfig(cfg)
	return nil
}

func key(pk, sk string) map[string]types.AttributeValue { return repository.Key(pk, sk) }

func getItem(ctx context.Context, pk, sk string) (doc, error) {
	out, err := db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &table, Key: key(pk, sk)})
	if err != nil || out.Item == nil {
		return nil, err
	}
	var d doc
	return d, attributevalue.UnmarshalMap(out.Item, &d)
}

func marshal(d doc) (map[string]types.AttributeValue, error) { return attributevalue.MarshalMap(d) }

func spacePK(sid string) string { return repository.SpacePK(sid) }

// papel do usuario no espaco; "" = sem acesso
func papel(ctx context.Context, u *user, sid string) (string, error) {
	m, err := getItem(ctx, "USER#"+u.Sub, "SPACE#"+sid)
	if err != nil || m == nil {
		return "", err
	}
	r, _ := m["role"].(string)
	return r, nil
}

func txSK(data, id string) string { return repository.TxSK(data, id) }

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
