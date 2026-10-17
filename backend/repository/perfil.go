package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// AtualizarPerfil altera o apelido e/ou o chat id do Telegram no vinculo da pessoa (USER#<sub> / SPACE#<sid>). Texto
// vazio remove o campo. domain.ErrVinculoNaoEncontrado se o vinculo nao existe.
func (p *Pessoas) AtualizarPerfil(ctx context.Context, sid, sub string, m domain.MudancaPerfil) error {
	var sets, removes []string
	vals := map[string]types.AttributeValue{}
	campo := func(nome, ph string, v *string) {
		if v == nil {
			return
		}
		if *v == "" {
			removes = append(removes, nome)
			return
		}
		sets, vals[ph] = append(sets, nome+" = "+ph), &types.AttributeValueMemberS{Value: *v}
	}
	campo("apelido", ":a", m.Apelido)
	campo("telegramChatId", ":t", m.ChatID)
	var expr []string
	if len(sets) > 0 {
		expr = append(expr, "SET "+strings.Join(sets, ", "))
	}
	if len(removes) > 0 {
		expr = append(expr, "REMOVE "+strings.Join(removes, ", "))
	}
	in := &dynamodb.UpdateItemInput{
		TableName:           &p.table,
		Key:                 Key("USER#"+sub, SpacePK(sid)), // sempre o item de quem fez a chamada
		ConditionExpression: aws.String("attribute_exists(PK)"),
		UpdateExpression:    aws.String(strings.Join(expr, " ")),
	}
	if len(vals) > 0 {
		in.ExpressionAttributeValues = vals
	}
	if _, err := p.db.UpdateItem(ctx, in); err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			return domain.ErrVinculoNaoEncontrado
		}
		return err
	}
	return nil
}
