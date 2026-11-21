package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// Vinculos guarda o codigo de vinculo de um canal (spec telegram-vinculo, R1; AD-11 da spine de canais). E o UNICO lugar que conhece a chave:
//
//	codigo pendente   PK = USER#<sub>   SK = LINK#<canal>   (sid, hash, expiraEm em segundos desde 1970, criadoEm, canal)
//
// Um item por pessoa e por canal: gerar substitui, usar apaga, e o teto e o numero de pessoas. So GetItem, PutItem, DeleteItem e UpdateItem pela
// chave exata; ninguem lista, ordena nem varre esta chave. A Query de espacos filtra begins_with(SK, "SPACE#") e nao o ve. Sem TTL, sem indice.
// O endereco que o canal prova (o chat id) vai para o campo do vinculo, que o mapa de destinos.go conhece.
type Vinculos struct {
	db    DynamoAPI
	table string
}

func NewVinculos(db DynamoAPI, table string) *Vinculos { return &Vinculos{db: db, table: table} }

func skVinculo(canal string) string { return "LINK#" + canal }

type itemVinculo struct {
	PK       string `dynamodbav:"PK"`
	SK       string `dynamodbav:"SK"`
	Canal    string `dynamodbav:"canal"`
	Sid      string `dynamodbav:"sid"`
	Hash     string `dynamodbav:"hash"`
	ExpiraEm int64  `dynamodbav:"expiraEm"`
	CriadoEm string `dynamodbav:"criadoEm"`
}

func (it itemVinculo) paraDominio() *domain.VinculoPendente {
	p := &domain.VinculoPendente{Sid: it.Sid, Hash: it.Hash, ExpiraEm: time.Unix(it.ExpiraEm, 0)}
	if t, err := time.Parse(time.RFC3339, it.CriadoEm); err == nil {
		p.CriadoEm = t
	}
	return p
}

// SalvarPendente grava o codigo pendente da pessoa neste canal, substituindo o anterior (uma gravacao so).
func (r *Vinculos) SalvarPendente(ctx context.Context, sub, canal string, p domain.VinculoPendente) error {
	item, err := attributevalue.MarshalMap(itemVinculo{PK: "USER#" + sub, SK: skVinculo(canal), Canal: canal, Sid: p.Sid, Hash: p.Hash,
		ExpiraEm: p.ExpiraEm.Unix(), CriadoEm: p.CriadoEm.UTC().Format(time.RFC3339)})
	if err != nil {
		return err
	}
	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &r.table, Item: item})
	return err
}

// LerPendente le o codigo pendente com leitura forte (1 RCU), ou nil se nao ha. Nao escreve nada.
func (r *Vinculos) LerPendente(ctx context.Context, sub, canal string) (*domain.VinculoPendente, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &r.table, Key: Key("USER#"+sub, skVinculo(canal)), ConsistentRead: aws.Bool(true)})
	if err != nil || out.Item == nil {
		return nil, err
	}
	var it itemVinculo
	if err := attributevalue.UnmarshalMap(out.Item, &it); err != nil {
		return nil, err
	}
	return it.paraDominio(), nil
}

// ConsumirPendente apaga o codigo pendente, de forma condicional (so se o hash ainda e o lido e ele nao venceu), e devolve o item que apagou.
// nil se outra execucao o consumiu antes, a pessoa gerou outro codigo ou ele venceu: o codigo nunca vale duas vezes, nem sob concorrencia.
func (r *Vinculos) ConsumirPendente(ctx context.Context, sub, canal, hash string, agora time.Time) (*domain.VinculoPendente, error) {
	out, err := r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:           &r.table,
		Key:                 Key("USER#"+sub, skVinculo(canal)),
		ConditionExpression: aws.String("#h = :h AND expiraEm > :agora"),
		ExpressionAttributeNames: map[string]string{"#h": "hash"}, // HASH e palavra reservada do DynamoDB: sem o alias a condicao e recusada
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":h":     &types.AttributeValueMemberS{Value: hash},
			":agora": &types.AttributeValueMemberN{Value: strconv.FormatInt(agora.Unix(), 10)},
		},
		ReturnValues: types.ReturnValueAllOld,
	})
	if err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			return nil, nil
		}
		return nil, err
	}
	var it itemVinculo
	if err := attributevalue.UnmarshalMap(out.Attributes, &it); err != nil || out.Attributes == nil {
		return nil, err
	}
	return it.paraDominio(), nil
}

// LigarEndereco grava o endereco que o canal provou (o chat id) no vinculo da pessoa no espaco, so se o vinculo existe, e devolve o apelido e o
// endereco que ele tinha antes. domain.ErrVinculoNaoEncontrado se o vinculo nao existe (nada e criado). Canal sem um endereco unico no vinculo
// (o push guarda varios) nao passa por aqui.
func (r *Vinculos) LigarEndereco(ctx context.Context, sid, sub, canal, endereco string) (domain.EnderecoLigado, error) {
	if canal != canalTelegram {
		return domain.EnderecoLigado{}, errors.New("canal sem endereco unico no vinculo")
	}
	out, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 &r.table,
		Key:                       Key("USER#"+sub, SpacePK(sid)), // sempre o vinculo de quem gerou o link
		UpdateExpression:          aws.String("SET " + campoTelegram + " = :c"),
		ConditionExpression:       aws.String("attribute_exists(PK)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":c": &types.AttributeValueMemberS{Value: endereco}},
		ReturnValues:              types.ReturnValueAllOld,
	})
	if err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			return domain.EnderecoLigado{}, domain.ErrVinculoNaoEncontrado
		}
		return domain.EnderecoLigado{}, err
	}
	var antigo map[string]any
	if err := attributevalue.UnmarshalMap(out.Attributes, &antigo); err != nil {
		return domain.EnderecoLigado{}, err
	}
	apelido, _ := antigo["apelido"].(string)
	anterior, _ := antigo[campoTelegram].(string)
	return domain.EnderecoLigado{Apelido: apelido, Anterior: anterior}, nil
}
