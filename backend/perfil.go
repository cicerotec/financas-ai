package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Nome de exibicao ("apelido") de cada pessoa. Fica no item de vinculo dela com o espaco
// (USER#sub / SPACE#id), ao lado do papel, e cada pessoa so altera o proprio.

const maxApelido = 30

// normalizarApelido tira espacos sobrando e recusa nomes longos ou com caracteres de controle.
// Vazio e valido: significa "sem apelido".
func normalizarApelido(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) > maxApelido {
		return "", errors.New("o nome pode ter no maximo 30 caracteres")
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return "", errors.New("o nome tem caracteres invalidos")
		}
	}
	return s, nil
}

func putPerfil(ctx context.Context, sid string, u *user, body doc) (resp, error) {
	bruto, ok := body["apelido"].(string)
	if !ok {
		return erro(http.StatusBadRequest, "informe o apelido")
	}
	nome, err := normalizarApelido(bruto)
	if err != nil {
		return erro(http.StatusBadRequest, err.Error())
	}
	in := &dynamodb.UpdateItemInput{
		TableName:           &table,
		Key:                 key("USER#"+u.Sub, "SPACE#"+sid), // sempre o item de quem fez a chamada
		ConditionExpression: aws.String("attribute_exists(PK)"),
	}
	if nome == "" {
		in.UpdateExpression = aws.String("REMOVE apelido")
	} else {
		in.UpdateExpression = aws.String("SET apelido = :a")
		in.ExpressionAttributeValues = map[string]types.AttributeValue{":a": &types.AttributeValueMemberS{Value: nome}}
	}
	if _, err := db.UpdateItem(ctx, in); err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			return erro(http.StatusNotFound, "vinculo com o espaco nao encontrado")
		}
		log.Println("put perfil:", err)
		return erro(http.StatusInternalServerError, "erro ao salvar")
	}
	return out(200, doc{"apelido": nome})
}
