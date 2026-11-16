package repository

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// O UNICO lugar que liga um canal de aviso ao campo do vinculo (USER#<sub> / SPACE#<sid>) onde fica o destino da pessoa
// (.ai/bmad/arquitetura-avisos-canais.md, AD-4): o Telegram guarda um texto (o chat id), o push guarda um conjunto de textos (um token por
// aparelho). Leitura (Destinos), remocao (RemoverDestino) e gravacao dos aparelhos usam estas constantes, e nenhum outro arquivo as repete.
const (
	canalTelegram = "telegram"
	canalPush     = "push"

	campoTelegram = "telegramChatId" // texto
	campoPush     = "pushTokens"     // conjunto de textos (SS)
)

// Destinos devolve para onde enviar os avisos da pessoa neste espaco, por canal, com os enderecos em ordem. Sem destino, o mapa vem
// vazio (estar assinado e ter destino). E um GetItem so.
func (p *Pessoas) Destinos(ctx context.Context, sid, sub string) (map[string][]string, error) {
	out, err := p.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &p.table, Key: Key("USER#"+sub, SpacePK(sid))})
	if err != nil || out.Item == nil {
		return nil, err
	}
	var d map[string]any
	if err := attributevalue.UnmarshalMap(out.Item, &d); err != nil {
		return nil, err
	}
	destinos := map[string][]string{}
	if chat, _ := d[campoTelegram].(string); chat != "" {
		destinos[canalTelegram] = []string{chat}
	}
	if tokens := textosDe(d[campoPush]); len(tokens) > 0 {
		destinos[canalPush] = tokens
	}
	return destinos, nil
}

// textosDe le um conjunto de textos (SS) ja desserializado.
func textosDe(v any) []string {
	l, _ := v.([]string)
	r := append([]string(nil), l...)
	sort.Strings(r)
	return r
}

// RemoverDestino apaga um destino do vinculo: o chat id (se for este) ou o token do conjunto. Destino que nao esta la nao e erro;
// vinculo que nao existe tambem nao (nada a apagar). So a rotina chama, quando o canal diz domain.ErrDestinoInvalido.
func (p *Pessoas) RemoverDestino(ctx context.Context, sid, sub, canal, endereco string) error {
	key := Key("USER#"+sub, SpacePK(sid))
	var in *dynamodb.UpdateItemInput
	switch canal {
	case canalPush:
		in = &dynamodb.UpdateItemInput{TableName: &p.table, Key: key, UpdateExpression: aws.String("DELETE " + campoPush + " :t"),
			ConditionExpression:       aws.String("attribute_exists(PK)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{":t": &types.AttributeValueMemberSS{Value: []string{endereco}}}}
	case canalTelegram:
		in = &dynamodb.UpdateItemInput{TableName: &p.table, Key: key, UpdateExpression: aws.String("REMOVE " + campoTelegram),
			ConditionExpression:       aws.String(campoTelegram + " = :c"),
			ExpressionAttributeValues: map[string]types.AttributeValue{":c": &types.AttributeValueMemberS{Value: endereco}}}
	default:
		return nil
	}
	if _, err := p.db.UpdateItem(ctx, in); err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			return nil
		}
		return err
	}
	return nil
}

// RegistrarAparelho acrescenta o token ao conjunto do vinculo (idempotente) e devolve quantos aparelhos a pessoa tem agora. O limite de
// MaxAparelhos e a condicao da propria gravacao, sem leitura antes: o token que ja esta no conjunto continua aceito, outro acima do
// limite e domain.ErrLimiteAparelhos. domain.ErrVinculoNaoEncontrado se a pessoa nao tem vinculo (nada e criado).
func (p *Pessoas) RegistrarAparelho(ctx context.Context, sid, sub, token string) (int, error) {
	out, err := p.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        &p.table,
		Key:              Key("USER#"+sub, SpacePK(sid)),
		UpdateExpression: aws.String("ADD " + campoPush + " :t"),
		ConditionExpression: aws.String("attribute_exists(PK) AND (attribute_not_exists(" + campoPush + ") OR contains(" + campoPush + ", :tk) OR size(" +
			campoPush + ") < :max)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":t":   &types.AttributeValueMemberSS{Value: []string{token}},
			":tk":  &types.AttributeValueMemberS{Value: token},
			":max": &types.AttributeValueMemberN{Value: strconv.Itoa(domain.MaxAparelhos)},
		},
		ReturnValues:                        types.ReturnValueUpdatedNew,
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
	})
	if err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			if cf.Item == nil { // a condicao falhou e nao ha item: a pessoa nao tem vinculo
				return 0, domain.ErrVinculoNaoEncontrado
			}
			return 0, domain.ErrLimiteAparelhos
		}
		return 0, err
	}
	return contarTextos(out.Attributes[campoPush]), nil
}

// RemoverAparelho tira o token do conjunto do vinculo. Devolve se ele estava la e quantos aparelhos sobraram. Vinculo que nao existe e
// domain.ErrVinculoNaoEncontrado.
func (p *Pessoas) RemoverAparelho(ctx context.Context, sid, sub, token string) (removido bool, restantes int, err error) {
	out, err := p.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 &p.table,
		Key:                       Key("USER#"+sub, SpacePK(sid)),
		UpdateExpression:          aws.String("DELETE " + campoPush + " :t"),
		ConditionExpression:       aws.String("attribute_exists(PK)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":t": &types.AttributeValueMemberSS{Value: []string{token}}},
		ReturnValues:              types.ReturnValueAllOld,
	})
	if err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			return false, 0, domain.ErrVinculoNaoEncontrado
		}
		return false, 0, err
	}
	if ss, ok := out.Attributes[campoPush].(*types.AttributeValueMemberSS); ok {
		for _, t := range ss.Value {
			if t == token {
				return true, len(ss.Value) - 1, nil
			}
		}
		return false, len(ss.Value), nil
	}
	return false, 0, nil
}

func contarTextos(v types.AttributeValue) int {
	if ss, ok := v.(*types.AttributeValueMemberSS); ok {
		return len(ss.Value)
	}
	return 0
}
