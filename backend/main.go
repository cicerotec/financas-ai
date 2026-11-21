package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"financas/adapter/identidade"
	"financas/adapter/notificacao"
	"financas/adapter/relogio"
	"financas/service"
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
	telegram.UsarVinculo(os.Getenv("TELEGRAM_WEBHOOK_SECRET_PARAM"), os.Getenv("TELEGRAM_BOT_USUARIO_PARAM")) // o segredo do webhook e o usuario do bot (spec telegram-vinculo, R23)
	fcm := notificacao.NovoFCM(ssm.NewFromConfig(cfg), os.Getenv("FCM_CREDENCIAL_PARAM"))
	despachante, err := service.NovoDespachante(telegram, fcm) // canal novo (e-mail...) e registrado aqui
	if err != nil {
		log.Fatal(err)
	}
	s := novoServidor(dynamodb.NewFromConfig(cfg), table, relogio.Real{}, despachante, os.Getenv("AMBIENTE"))
	s.usarBot(telegram) // o webhook e o link do vinculo falam com o mesmo bot dos avisos
	// a recuperacao do MFA fala com o login (o Cognito) so para achar a pessoa pelo e-mail e desligar o MFA dela (spec mfa-opcional, R32)
	s.usarIdentidade(identidade.NovoCognito(cognitoidentityprovider.NewFromConfig(cfg), os.Getenv("USER_POOL_ID")))
	if porta := os.Getenv("LOCAL_PORT"); porta != "" { // so no computador do desenvolvedor (local.go)
		rodarLocal(porta, s, table)
		return
	}
	lambda.Start(s.entrada)
}
