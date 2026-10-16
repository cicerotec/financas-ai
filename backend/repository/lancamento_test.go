package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"financas/domain"
)

// espiao guarda as transacoes que o repository manda ao DynamoDB, sem executar nada.
type espiao struct {
	DynamoAPI
	enviadas []*dynamodb.TransactWriteItemsInput
}

func (e *espiao) TransactWriteItems(_ context.Context, in *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	e.enviadas = append(e.enviadas, in)
	return &dynamodb.TransactWriteItemsOutput{}, nil
}

func mudancaDeExemplo(id string, versoes ...domain.Versao) domain.Mudanca {
	l := domain.Lancamento{"id": id, "dataEvento": "2026-01-01T10:00:00.000Z", "valor": 1.0}
	return domain.Mudanca{Antes: l, Depois: domain.CopiaRasa(l), DataAnterior: "2026-01-01T10:00:00.000Z", Versoes: versoes}
}

func versaoDeExemplo(n int, acao domain.Acao, l domain.Lancamento) domain.Versao {
	return domain.Versao{Numero: n, Acao: acao, PorSub: "sub", Em: "2026-01-01T10:00:00.000Z", Registro: l}
}

// F1-T15: nenhuma escrita de versao sai sem attribute_not_exists(SK), e o historico nunca recebe Delete.
func TestTodaVersaoVaiComAttributeNotExistsNoSK(t *testing.T) {
	e := &espiao{}
	r := NewLancamentos(e, "T")
	l := domain.Lancamento{"id": "abc", "dataEvento": "2026-01-01T10:00:00.000Z"}
	m := mudancaDeExemplo("abc", versaoDeExemplo(1, domain.AcaoInsert, l), versaoDeExemplo(2, domain.AcaoUpdate, l))
	if err := r.Salvar(context.Background(), "s1", m); err != nil {
		t.Fatal(err)
	}
	if err := r.SalvarLote(context.Background(), "s1", []domain.Mudanca{m, mudancaDeExemplo("def", versaoDeExemplo(1, domain.AcaoInsert, l))}); err != nil {
		t.Fatal(err)
	}
	if len(e.enviadas) != 2 {
		t.Fatalf("esperava 2 transacoes, vieram %d", len(e.enviadas))
	}
	versoes := 0
	for _, in := range e.enviadas {
		for _, it := range in.TransactItems {
			if it.Delete != nil && strings.HasPrefix(chaveS(it.Delete.Key["PK"]), "HIST#") {
				t.Error("o historico nunca recebe Delete")
			}
			if it.Update != nil {
				t.Error("o historico nunca recebe Update")
			}
			if it.Put == nil || !strings.HasPrefix(chaveS(it.Put.Item["PK"]), "HIST#") {
				continue
			}
			versoes++
			if aws.ToString(it.Put.ConditionExpression) != "attribute_not_exists(SK)" {
				t.Errorf("a versao %s foi gravada sem attribute_not_exists(SK): %q", chaveS(it.Put.Item["SK"]), aws.ToString(it.Put.ConditionExpression))
			}
		}
	}
	if versoes != 5 {
		t.Errorf("esperava 5 versoes nas transacoes, vieram %d", versoes)
	}
}

func chaveS(a types.AttributeValue) string {
	if s, ok := a.(*types.AttributeValueMemberS); ok {
		return s.Value
	}
	return ""
}

func TestChavesDoHistorico(t *testing.T) {
	if got := HistPK("s1"); got != "HIST#s1" {
		t.Errorf("HistPK = %q", got)
	}
	if got := HistSK("abc", 7); got != "TX#abc#00000007" {
		t.Errorf("HistSK = %q", got)
	}
	if HistSK("abc", 2) >= HistSK("abc", 10) {
		t.Error("as versoes precisam ordenar em ordem numerica (8 digitos)")
	}
}

// leitor responde vazio e guarda se as leituras pediram consistencia forte.
type leitor struct {
	DynamoAPI
	getConsistente, queryConsistente bool
}

func (l *leitor) GetItem(_ context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	l.getConsistente = aws.ToBool(in.ConsistentRead)
	return &dynamodb.GetItemOutput{}, nil
}

func (l *leitor) Query(_ context.Context, in *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	l.queryConsistente = aws.ToBool(in.ConsistentRead)
	return &dynamodb.QueryOutput{}, nil
}

// Quem le para depois escrever usa leitura consistente; senao salvar e desfazer em seguida dariam falso conflito.
func TestLeiturasParaEscreverSaoConsistentes(t *testing.T) {
	l := &leitor{}
	r := NewLancamentos(l, "T")
	if _, err := r.Buscar(context.Background(), "s1", "2026-01-01T10:00:00.000Z", "abc"); err != nil {
		t.Fatal(err)
	}
	if n, err := r.UltimaVersao(context.Background(), "s1", "abc"); err != nil || n != 0 {
		t.Fatalf("UltimaVersao sem historico = (%d, %v)", n, err)
	}
	if !l.getConsistente || !l.queryConsistente {
		t.Errorf("Buscar e UltimaVersao devem usar ConsistentRead (get=%v, query=%v)", l.getConsistente, l.queryConsistente)
	}
}
