package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// repoLancamentosDynamo e o adaptador DynamoDB de RepositorioLancamentos. E o unico lugar que monta a chave de um
// lancamento (PK e SK) e as expressoes de escrita. Ele ainda le o cliente e a tabela globais (db e table); isso sai
// com o restante do item 32.
type repoLancamentosDynamo struct{}

func novoRepoLancamentos() RepositorioLancamentos { return repoLancamentosDynamo{} }

func (repoLancamentosDynamo) Buscar(ctx context.Context, sid, dataEvento, id string) (Lancamento, error) {
	return getItem(ctx, spacePK(sid), txSK(dataEvento, id))
}

// Salvar poe PK e SK no proprio Depois (o service ainda usa o mesmo mapa depois de gravar).
func (repoLancamentosDynamo) Salvar(ctx context.Context, sid string, m Mudanca) error {
	d := m.Depois
	data, _ := d["dataEvento"].(string)
	id, _ := d["id"].(string)
	d["PK"], d["SK"] = spacePK(sid), txSK(data, id)
	item, err := marshal(d)
	if err != nil {
		return errDadosInvalidos
	}
	switch {
	case m.Antes == nil:
		_, err = db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &table, Item: item,
			ConditionExpression: aws.String("attribute_not_exists(PK)")})
	case d["SK"] == txSK(m.DataAnterior, id):
		_, err = db.PutItem(ctx, &dynamodb.PutItemInput{TableName: &table, Item: item})
	default: // dataEvento mudou: a chave muda, entao apaga a antiga e grava a nova juntas
		_, err = db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
			{Delete: &types.Delete{TableName: &table, Key: key(spacePK(sid), txSK(m.DataAnterior, id))}},
			{Put: &types.Put{TableName: &table, Item: item}},
		}})
	}
	return err
}

func (repoLancamentosDynamo) Excluir(ctx context.Context, sid, dataEvento, id string) error {
	_, err := db.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: &table, Key: key(spacePK(sid), txSK(dataEvento, id))})
	return err
}

// GravarLote usa BatchWriteItem com nova tentativa dos itens nao processados (limite de escrita). Devolve o que
// sobrou quando o tempo da Lambda esta acabando.
func (repoLancamentosDynamo) GravarLote(ctx context.Context, sid string, itens []Lancamento) []Lancamento {
	var reqs []types.WriteRequest
	var naoMontados []Lancamento
	for _, d := range itens {
		data, _ := d["dataEvento"].(string)
		id, _ := d["id"].(string)
		d["PK"], d["SK"] = spacePK(sid), txSK(data, id)
		item, err := marshal(d)
		if err != nil { // os valores vem de JSON e sempre serializam; mesmo assim, nunca some em silencio
			log.Println("lote: marshal:", err)
			naoMontados = append(naoMontados, d)
			continue
		}
		reqs = append(reqs, types.WriteRequest{PutRequest: &types.PutRequest{Item: item}})
	}
	pend := gravarLote(ctx, reqs)
	sobra := naoMontados
	for _, w := range pend {
		var d Lancamento
		if attributevalue.UnmarshalMap(w.PutRequest.Item, &d) == nil {
			sobra = append(sobra, d)
		}
	}
	return sobra
}

// gravarLote: BatchWriteItem com nova tentativa dos itens nao processados (limite de escrita).
func gravarLote(ctx context.Context, reqs []types.WriteRequest) []types.WriteRequest {
	limite := time.Now().Add(10 * time.Second)
	if prazo, ok := ctx.Deadline(); ok {
		limite = prazo.Add(-4 * time.Second)
	}
	espera := 200 * time.Millisecond
	for len(reqs) > 0 {
		saida, err := db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{table: reqs}})
		if err != nil {
			log.Println("batch write:", err)
		} else {
			reqs = saida.UnprocessedItems[table]
		}
		if len(reqs) == 0 || time.Now().Add(espera).After(limite) {
			break
		}
		time.Sleep(espera)
		if espera < 2*time.Second {
			espera *= 2
		}
	}
	return reqs
}
