package main

import (
	"context"
	"os"

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

func txSK(data, id string) string { return repository.TxSK(data, id) }
