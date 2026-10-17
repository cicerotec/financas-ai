package main

import (
	"context"
	"errors"
	"log"
	"net/http"

)

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
