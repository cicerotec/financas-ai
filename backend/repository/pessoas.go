package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Pessoas le o vinculo de uma pessoa com o espaco (USER#<sub> / SPACE#<sid>), onde ficam o papel e o apelido.
type Pessoas struct {
	db    DynamoAPI
	table string
}

func NewPessoas(db DynamoAPI, table string) *Pessoas { return &Pessoas{db: db, table: table} }

// Apelido devolve o apelido da pessoa no espaco; vazio se nao houver (ou se nao for possivel ler).
func (p *Pessoas) Apelido(ctx context.Context, sid, sub string) string {
	if sub == "" {
		return ""
	}
	out, err := p.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &p.table, Key: Key("USER#"+sub, SpacePK(sid))})
	if err != nil || out.Item == nil {
		return ""
	}
	var d map[string]any
	if attributevalue.UnmarshalMap(out.Item, &d) != nil {
		return ""
	}
	s, _ := d["apelido"].(string)
	return s
}

// Destinos devolve para onde enviar os avisos da pessoa neste espaco, por canal: hoje so o "telegram", a partir do chat id do
// vinculo. Sem destino, o mapa vem vazio (estar assinado e ter destino).
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
	if chat, _ := d["telegramChatId"].(string); chat != "" {
		destinos["telegram"] = []string{chat}
	}
	return destinos, nil
}
