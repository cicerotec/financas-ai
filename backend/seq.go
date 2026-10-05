package main

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Contador de alteracoes do espaco (item SPACE#<id> / SEQ, atributo seq).
//
// Cada escrita de lancamento (criar, editar, excluir, importar) soma 1 e devolve o valor novo em "_seq".
// A tela guarda o ultimo seq que viu: para saber se outra pessoa gravou, le so este item (1 leitura) em vez de
// reler os 1000 lancamentos. Se o seq que volta de uma escrita propria e o anterior + 1, ninguem gravou no meio.
// O seq e um numero do DynamoDB (ate 38 digitos) e trafega como numero JSON (float64 exato ate 2^53, ~9 quadrilhoes
// de escritas); nunca e convertido para inteiro de 32 bits.
const skSeq = "SEQ"

// proximoSeq soma 1 ao contador do espaco (criando-o se nao existir) e devolve o valor novo.
func proximoSeq(ctx context.Context, sid string) (float64, error) {
	out, err := db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        &table,
		Key:              key(spacePK(sid), skSeq),
		UpdateExpression: aws.String("ADD seq :um"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":um": &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return 0, err
	}
	var d doc
	if err := attributevalue.UnmarshalMap(out.Attributes, &d); err != nil {
		return 0, err
	}
	n, _ := d["seq"].(float64)
	return n, nil
}

// comSeq soma 1 ao contador e anexa o valor novo na resposta. A escrita ja foi feita: se o contador falhar, so
// registra no log e a resposta sai sem "_seq" (a tela entende como "nao sei" e recarrega).
func comSeq(ctx context.Context, sid string, d doc) doc {
	n, err := proximoSeq(ctx, sid)
	if err != nil {
		log.Printf("ERRO seq sid=%s: %v", sid, err)
		return d
	}
	d["_seq"] = n
	return d
}

func getSeq(ctx context.Context, sid string) (resp, error) {
	d, err := getItem(ctx, spacePK(sid), skSeq)
	if err != nil {
		log.Println("seq:", err)
		return erro(http.StatusInternalServerError, "erro interno")
	}
	n, _ := d["seq"].(float64)
	return out(200, doc{"seq": n})
}
