package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"financas/adapter/notificacao"
	"financas/adapter/relogio"
)

// main so liga e parte: le o ambiente, cria os clients da AWS uma vez, monta o Servidor e o entrega a Lambda (ou ao modo
// local, so para depuracao).
func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	table := os.Getenv("TABLE_NAME")
	telegram := notificacao.NovoTelegram(ssm.NewFromConfig(cfg), os.Getenv("TELEGRAM_TOKEN_PARAM"))
	s := novoServidor(dynamodb.NewFromConfig(cfg), table, relogio.Real{}, telegram, os.Getenv("AMBIENTE"))
	if porta := os.Getenv("LOCAL_PORT"); porta != "" { // so no computador do desenvolvedor (local.go)
		rodarLocal(porta, s, table)
		return
	}
	lambda.Start(s.entrada)
}
