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

// CodigosDeRecuperacao guarda as listas de codigos de recuperacao do MFA (spec mfa-opcional, R36). E o UNICO lugar que conhece as chaves:
//
//	lista ativa     PK = USER#<sub>   SK = MFACODIGOS#ATIVA
//	lista pendente  PK = USER#<sub>   SK = MFACODIGOS#PENDENTE   (tem expiraEm, em segundos desde 1970)
//
// Cada item tem listaId, geradaEm e codigos (lista de {hash, usado}); nunca o codigo em texto. A unica consulta que le USER#<sub>
// (espacos.go) filtra begins_with(SK, "SPACE#"), entao estes itens nao aparecem nas listas de espacos. Sem TTL do DynamoDB, sem indice.
type CodigosDeRecuperacao struct {
	db    DynamoAPI
	table string
}

func NewCodigosDeRecuperacao(db DynamoAPI, table string) *CodigosDeRecuperacao {
	return &CodigosDeRecuperacao{db: db, table: table}
}

const (
	skCodigosAtiva    = "MFACODIGOS#ATIVA"
	skCodigosPendente = "MFACODIGOS#PENDENTE"
)

func pkUsuarioMFA(sub string) string { return "USER#" + sub }

type itemCodigo struct {
	Hash  string `dynamodbav:"hash"`
	Usado bool   `dynamodbav:"usado"`
}

type itemListaCodigos struct {
	PK       string       `dynamodbav:"PK"`
	SK       string       `dynamodbav:"SK"`
	ListaID  string       `dynamodbav:"listaId"`
	GeradaEm string       `dynamodbav:"geradaEm"`
	ExpiraEm int64        `dynamodbav:"expiraEm,omitempty"`
	Codigos  []itemCodigo `dynamodbav:"codigos"`
}

func paraItem(sub, sk string, l domain.ListaCodigos) itemListaCodigos {
	it := itemListaCodigos{PK: pkUsuarioMFA(sub), SK: sk, ListaID: l.ListaID, GeradaEm: l.GeradaEm.UTC().Format(time.RFC3339)}
	if !l.ExpiraEm.IsZero() {
		it.ExpiraEm = l.ExpiraEm.Unix()
	}
	for _, c := range l.Codigos {
		it.Codigos = append(it.Codigos, itemCodigo{Hash: c.Hash, Usado: c.Usado})
	}
	return it
}

// SalvarPendente grava a lista pendente da pessoa, substituindo a pendente anterior se houver. Nao toca na ativa.
func (r *CodigosDeRecuperacao) SalvarPendente(ctx context.Context, sub string, l domain.ListaCodigos) error {
	item, err := attributevalue.MarshalMap(paraItem(sub, skCodigosPendente, l))
	if err != nil {
		return err
	}
	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &r.table, Item: item})
	return err
}

// lerItem le um dos itens da pessoa com leitura forte (a confirmacao decide em cima dele). nil se nao existe.
func (r *CodigosDeRecuperacao) lerItem(ctx context.Context, sub, sk string) (*itemListaCodigos, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &r.table, Key: Key(pkUsuarioMFA(sub), sk), ConsistentRead: aws.Bool(true)})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}
	var it itemListaCodigos
	if err := attributevalue.UnmarshalMap(out.Item, &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// Confirmar promove a pendente (a de listaID, nao vencida) a ativa e apaga a ativa anterior, numa escrita condicional (spec R25a).
// Idempotente: se a ativa ja e a de listaID, nao faz nada e devolve nil. domain.ErrListaSubstituida se a pendente nao existe, tem outro
// listaID ou venceu, e a ativa nao e a pedida (a ativa continua como estava).
func (r *CodigosDeRecuperacao) Confirmar(ctx context.Context, sub, listaID string, agora time.Time) error {
	pend, err := r.lerItem(ctx, sub, skCodigosPendente)
	if err != nil {
		return err
	}
	if pend != nil && pend.ListaID == listaID && pend.ExpiraEm > agora.Unix() {
		ativa := *pend
		ativa.SK, ativa.ExpiraEm = skCodigosAtiva, 0
		item, err := attributevalue.MarshalMap(ativa)
		if err != nil {
			return err
		}
		_, err = r.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{TableName: &r.table, Item: item}}, // a ativa anterior e sobrescrita: a nova nasce no lugar dela
			{Delete: &types.Delete{TableName: &r.table, Key: Key(pkUsuarioMFA(sub), skCodigosPendente),
				ConditionExpression: aws.String("listaId = :id AND expiraEm > :agora"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":id":    &types.AttributeValueMemberS{Value: listaID},
					":agora": &types.AttributeValueMemberN{Value: strconv.FormatInt(agora.Unix(), 10)},
				}}},
		}})
		var cancelada *types.TransactionCanceledException
		if err == nil {
			return nil
		}
		if !errors.As(err, &cancelada) {
			return err
		}
		// a pendente mudou entre a leitura e a escrita: cai na checagem de idempotencia abaixo
	}
	ativa, err := r.lerItem(ctx, sub, skCodigosAtiva)
	if err != nil {
		return err
	}
	if ativa != nil && ativa.ListaID == listaID {
		return nil // confirmar de novo a lista que ja e a ativa
	}
	return domain.ErrListaSubstituida
}
