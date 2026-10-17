package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"financas/adapter/relogio"
)

func main() {
	if err := initDB(context.Background()); err != nil {
		log.Fatal(err)
	}
	s := novoServidor(db, table, relogio.Real{})
	if porta := os.Getenv("LOCAL_PORT"); porta != "" { // so no computador do desenvolvedor (local.go)
		rodarLocal(porta, s)
		return
	}
	lambda.Start(s.entrada)
}
