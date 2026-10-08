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

// putPerfil altera o apelido e/ou o chat id do Telegram (para onde vao os avisos) de quem fez a chamada.
// Texto vazio remove o campo.
func putPerfil(ctx context.Context, sid string, u *user, body doc) (resp, error) {
	apelidoBruto, temApelido := body["apelido"]
	chatBruto, temChat := body["telegramChatId"]
	if !temApelido && !temChat {
		return erro(http.StatusBadRequest, "informe o apelido ou o telegramChatId")
	}
	var sets, removes []string
	vals := map[string]types.AttributeValue{}
	resposta := doc{}
	if temApelido {
		bruto, ok := apelidoBruto.(string)
		if !ok {
			return erro(http.StatusBadRequest, "informe o apelido")
		}
		nome, err := normalizarApelido(bruto)
		if err != nil {
			return erro(http.StatusBadRequest, err.Error())
		}
		if nome == "" {
			removes = append(removes, "apelido")
		} else {
			sets, vals[":a"] = append(sets, "apelido = :a"), &types.AttributeValueMemberS{Value: nome}
		}
		resposta["apelido"] = nome
	}
	if temChat {
		bruto, ok := chatBruto.(string)
		if !ok {
			return erro(http.StatusBadRequest, "informe o telegramChatId como texto")
		}
		chat := strings.TrimSpace(bruto)
		if chat == "" {
			removes = append(removes, "telegramChatId")
		} else if !reChatID.MatchString(chat) {
			return erro(http.StatusBadRequest, "o chat id do Telegram tem so numeros (ex.: 123456789)")
		} else {
			sets, vals[":t"] = append(sets, "telegramChatId = :t"), &types.AttributeValueMemberS{Value: chat}
		}
		resposta["telegramChatId"] = chat
	}
	var expr []string
	if len(sets) > 0 {
		expr = append(expr, "SET "+strings.Join(sets, ", "))
	}
	if len(removes) > 0 {
		expr = append(expr, "REMOVE "+strings.Join(removes, ", "))
	}
	in := &dynamodb.UpdateItemInput{
		TableName:           &table,
		Key:                 key("USER#"+u.Sub, "SPACE#"+sid), // sempre o item de quem fez a chamada
		ConditionExpression: aws.String("attribute_exists(PK)"),
		UpdateExpression:    aws.String(strings.Join(expr, " ")),
	}
	if len(vals) > 0 {
		in.ExpressionAttributeValues = vals
	}
	if _, err := db.UpdateItem(ctx, in); err != nil {
		var cf *types.ConditionalCheckFailedException
		if errors.As(err, &cf) {
			return erro(http.StatusNotFound, "vinculo com o espaco nao encontrado")
		}
		log.Println("put perfil:", err)
		return erro(http.StatusInternalServerError, "erro ao salvar")
	}
	return out(200, resposta)
}

// postAvisoTeste manda uma mensagem de teste para o Telegram de quem fez a chamada.
func postAvisoTeste(ctx context.Context, sid string, u *user) (resp, error) {
	v, err := getItem(ctx, "USER#"+u.Sub, "SPACE#"+sid)
	if err != nil {
		log.Println("aviso teste:", err)
		return erro(http.StatusInternalServerError, "erro interno")
	}
	chat, _ := v["telegramChatId"].(string)
	if chat == "" {
		return erro(http.StatusBadRequest, "cadastre o seu chat id do Telegram primeiro")
	}
	err = enviarMensagem(ctx, chat, "🔔 "+marcaAmbiente()+"Teste do financas.ai: se você recebeu esta mensagem, os avisos vão chegar aqui.")
	switch {
	case errors.Is(err, errTelegramNaoConfigurado):
		return erro(http.StatusServiceUnavailable, "o envio pelo Telegram ainda nao foi configurado no servidor")
	case err != nil:
		log.Println("aviso teste:", err)
		return erro(http.StatusBadGateway, "o Telegram recusou a mensagem: confira o chat id e se voce ja iniciou a conversa com o bot")
	}
	return out(200, doc{"ok": true})
}
