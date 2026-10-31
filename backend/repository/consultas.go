package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Consultas guarda as consultas salvas da Tendencias (item 53): um item por consulta, PK = SPACE#<sid>, SK = CONSULTA#<id>, com o
// controle otimista "version" (a mesma gravacao condicional de Config.SalvarItem). A exclusao e logica (excluidoEm): o item fica.
type Consultas struct {
	db    DynamoAPI
	table string
	cfg   *Config
}

func NewConsultas(db DynamoAPI, table string) *Consultas {
	return &Consultas{db: db, table: table, cfg: NewConfig(db, table)}
}

const prefixoConsulta = "CONSULTA#"

func skConsulta(id string) string { return prefixoConsulta + id }

// Listar devolve todas as consultas do espaco, inclusive as excluidas, com as chaves internas. Uma Query por prefixo.
func (r *Consultas) Listar(ctx context.Context, sid string) ([]domain.Doc, error) {
	itens := []domain.Doc{}
	var inicio map[string]types.AttributeValue
	for {
		q, err := r.db.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(r.table),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :s)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: SpacePK(sid)},
				":s":  &types.AttributeValueMemberS{Value: prefixoConsulta},
			},
			ExclusiveStartKey: inicio,
		})
		if err != nil {
			return nil, err
		}
		for _, it := range q.Items {
			var d domain.Doc
			if attributevalue.UnmarshalMap(it, &d) == nil {
				itens = append(itens, d)
			}
		}
		if len(q.LastEvaluatedKey) == 0 {
			return itens, nil
		}
		inicio = q.LastEvaluatedKey
	}
}

// Ler devolve a consulta (com chaves e version), ou nil se nao existe neste espaco.
func (r *Consultas) Ler(ctx context.Context, sid, id string) (domain.Doc, error) {
	return r.cfg.LerItem(ctx, SpacePK(sid), skConsulta(id))
}

// Salvar grava com controle otimista (esperada = version lida; 0 = item novo). domain.ErrConflitoConfig se alguem gravou antes.
func (r *Consultas) Salvar(ctx context.Context, sid, id string, body domain.Doc, esperada float64) (domain.Doc, error) {
	return r.cfg.SalvarItem(ctx, SpacePK(sid), skConsulta(id), body, esperada)
}
