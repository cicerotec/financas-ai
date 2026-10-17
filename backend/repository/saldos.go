package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Saldos guarda o fechamento mensal de cada banco: um item por banco, PK = SPACE#<sid>, SK = SALDO#<banco>, com os
// campos caches, conferidos, invalidoDe, base e o controle otimista "version". Usa a mesma gravacao condicional das
// configuracoes (Config.SalvarItem).
type Saldos struct {
	db    DynamoAPI
	table string
	cfg   *Config
}

func NewSaldos(db DynamoAPI, table string) *Saldos {
	return &Saldos{db: db, table: table, cfg: NewConfig(db, table)}
}

func skSaldo(banco string) string { return "SALDO#" + banco }

// Ler devolve o item do banco (com chaves e version), ou nil se ainda nao existe.
func (r *Saldos) Ler(ctx context.Context, sid, banco string) (domain.Doc, error) {
	return r.cfg.LerItem(ctx, SpacePK(sid), skSaldo(banco))
}

// Salvar grava com controle otimista (esperada = version lida). domain.ErrConflitoConfig se alguem gravou antes.
func (r *Saldos) Salvar(ctx context.Context, sid, banco string, body domain.Doc, esperada float64) (domain.Doc, error) {
	return r.cfg.SalvarItem(ctx, SpacePK(sid), skSaldo(banco), body, esperada)
}

// Todos devolve o item de cada banco do espaco, sem as chaves internas.
func (r *Saldos) Todos(ctx context.Context, sid string) ([]domain.Doc, error) {
	q, err := r.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :s)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: SpacePK(sid)},
			":s":  &types.AttributeValueMemberS{Value: "SALDO#"},
		},
	})
	if err != nil {
		return nil, err
	}
	itens := []domain.Doc{}
	for _, it := range q.Items {
		var d domain.Doc
		if attributevalue.UnmarshalMap(it, &d) == nil {
			itens = append(itens, domain.Publico(d))
		}
	}
	return itens, nil
}
