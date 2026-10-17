package repository

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Config guarda as configuracoes do espaco: documentos pequenos com controle otimista por "version".
//
//	configuracao compartilhada   PK = SPACE#<sid>   SK = CFG#<NOME>                    (listas, tags, combos, saldos, cartoes)
//	aparencia de uma pessoa      PK = SPACE#<sid>   SK = USER#<sub>#CFG#APARENCIA      (sem controle de versao)
type Config struct {
	db    DynamoAPI
	table string
}

func NewConfig(db DynamoAPI, table string) *Config { return &Config{db: db, table: table} }

// LerItem devolve o item inteiro (com PK, SK e version), ou nil se nao existe.
func (r *Config) LerItem(ctx context.Context, pk, sk string) (domain.Doc, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &r.table, Key: Key(pk, sk)})
	if err != nil || out.Item == nil {
		return nil, err
	}
	var d domain.Doc
	return d, attributevalue.UnmarshalMap(out.Item, &d)
}

// SalvarItem grava com controle otimista: so grava se a versao no banco ainda e a que o cliente leu (esperada; 0 = o
// item ainda nao existe ou nunca teve versao). Devolve o item gravado, sem chaves internas, com version = esperada + 1.
// domain.ErrConflitoConfig se alguem gravou antes; domain.ErrDadosInvalidos se o corpo nao serializa.
func (r *Config) SalvarItem(ctx context.Context, pk, sk string, body domain.Doc, esperada float64) (domain.Doc, error) {
	body["PK"], body["SK"], body["version"] = pk, sk, esperada+1
	item, err := attributevalue.MarshalMap(body)
	if err != nil {
		return nil, domain.ErrDadosInvalidos
	}
	in := &dynamodb.PutItemInput{TableName: &r.table, Item: item}
	if esperada == 0 {
		in.ConditionExpression = aws.String("attribute_not_exists(PK) OR attribute_not_exists(version)")
	} else {
		in.ConditionExpression = aws.String("version = :v")
		in.ExpressionAttributeValues = map[string]types.AttributeValue{":v": &types.AttributeValueMemberN{Value: strconv.FormatFloat(esperada, 'f', -1, 64)}}
	}
	if _, err := r.db.PutItem(ctx, in); err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			return nil, domain.ErrConflitoConfig
		}
		return nil, err
	}
	return domain.Publico(body), nil
}

func skConfig(nome string) string { return "CFG#" + strings.ToUpper(nome) }

func skAparencia(sub string) string { return "USER#" + sub + "#CFG#APARENCIA" }

// Ler devolve a configuracao compartilhada (nome em qualquer caixa), ou nil se ainda nao existe.
func (r *Config) Ler(ctx context.Context, sid, nome string) (domain.Doc, error) {
	return r.LerItem(ctx, SpacePK(sid), skConfig(nome))
}

// Salvar grava a configuracao compartilhada com controle otimista (veja SalvarItem).
func (r *Config) Salvar(ctx context.Context, sid, nome string, body domain.Doc, esperada float64) (domain.Doc, error) {
	return r.SalvarItem(ctx, SpacePK(sid), skConfig(nome), body, esperada)
}

// LerAparencia devolve a aparencia da pessoa no espaco, ou nil se ainda nao definiu.
func (r *Config) LerAparencia(ctx context.Context, sid, sub string) (domain.Doc, error) {
	return r.LerItem(ctx, SpacePK(sid), skAparencia(sub))
}

// GravarAparencia substitui a aparencia da pessoa (sem controle de versao: so ela escreve nela).
func (r *Config) GravarAparencia(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error) {
	body["PK"], body["SK"] = SpacePK(sid), skAparencia(sub)
	item, err := attributevalue.MarshalMap(body)
	if err != nil {
		return nil, domain.ErrDadosInvalidos
	}
	if _, err := r.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &r.table, Item: item}); err != nil {
		return nil, err
	}
	return domain.Publico(body), nil
}
