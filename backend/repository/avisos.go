package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Avisos guarda o item auxiliar de cada lancamento com aviso, para a rotina diaria nao varrer a tabela:
//
//	PK = AVISOS#ATIVOS   SK = <sid>#<id>   (sid, id, dataEvento, venc, dias, insistir)
//
// E mantido ao criar, editar e excluir o lancamento. A rotina le so esta particao e confere o lancamento de verdade antes
// de enviar.
type Avisos struct {
	db    DynamoAPI
	table string
}

func NewAvisos(db DynamoAPI, table string) *Avisos { return &Avisos{db: db, table: table} }

const pkAvisos = "AVISOS#ATIVOS"

func skAviso(sid, id string) string { return sid + "#" + id }

// Sincronizar grava o item auxiliar de um lancamento (sobrescreve o que houver). item leva id, dataEvento, venc, dias e
// insistir; o sid e as chaves entram aqui.
func (r *Avisos) Sincronizar(ctx context.Context, sid string, item domain.Doc) error {
	id, _ := item["id"].(string)
	gravar := domain.CopiaRasa(item)
	gravar["PK"], gravar["SK"], gravar["sid"] = pkAvisos, skAviso(sid, id), sid
	m, err := attributevalue.MarshalMap(gravar)
	if err != nil {
		return err
	}
	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &r.table, Item: m})
	return err
}

// Remover apaga o item auxiliar (nao falha se nao existe).
func (r *Avisos) Remover(ctx context.Context, sid, id string) error {
	_, err := r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: &r.table, Key: Key(pkAvisos, skAviso(sid, id))})
	return err
}

// Ativos devolve todos os itens auxiliares, de todos os espacos.
func (r *Avisos) Ativos(ctx context.Context) ([]domain.Doc, error) {
	in := &dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pkAvisos},
		},
	}
	var itens []domain.Doc
	for {
		out, err := r.db.Query(ctx, in)
		if err != nil {
			return nil, err
		}
		for _, it := range out.Items {
			var d domain.Doc
			if attributevalue.UnmarshalMap(it, &d) == nil {
				itens = append(itens, d)
			}
		}
		if out.LastEvaluatedKey == nil {
			return itens, nil
		}
		in.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

// BuscarItem le um item auxiliar pelo id (o mesmo usado em Sincronizar); nil se nao existe.
func (r *Avisos) BuscarItem(ctx context.Context, sid, id string) (domain.Doc, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &r.table, Key: Key(pkAvisos, skAviso(sid, id))})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}
	var d domain.Doc
	if err := attributevalue.UnmarshalMap(out.Item, &d); err != nil {
		return nil, err
	}
	return d, nil
}

// ListarRecorrencias devolve os itens auxiliares de recorrencia de um espaco (so a faixa do espaco na particao dos avisos).
func (r *Avisos) ListarRecorrencias(ctx context.Context, sid string) ([]domain.Doc, error) {
	in := &dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :s)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pkAvisos},
			":s":  &types.AttributeValueMemberS{Value: skAviso(sid, "recorrencia#")},
		},
	}
	var itens []domain.Doc
	for {
		out, err := r.db.Query(ctx, in)
		if err != nil {
			return nil, err
		}
		for _, it := range out.Items {
			var d domain.Doc
			if attributevalue.UnmarshalMap(it, &d) == nil {
				itens = append(itens, d)
			}
		}
		if out.LastEvaluatedKey == nil {
			return itens, nil
		}
		in.ExclusiveStartKey = out.LastEvaluatedKey
	}
}
