package main

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func seqDoBanco(t *testing.T, f *bancoFake) float64 {
	t.Helper()
	d := f.ler(t, pkEspaco, "SEQ")
	if d == nil {
		return 0
	}
	n, _ := d["seq"].(float64)
	return n
}

func TestCadaEscritaSomaUmEDevolveOSeq(t *testing.T) {
	f := cenario(t)
	c, d := chamar(t, "own", "GET", "spaces/s1/seq", nil, "")
	esperar(t, "ler seq sem escritas", c, 200)
	if d["seq"] != 0.0 {
		t.Errorf("sem escritas o seq e 0, veio %v", d["seq"])
	}

	c, d = chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-02-10T12:00:00.000Z", "ITAU", 5))
	esperar(t, "criar", c, 201)
	if d["_seq"] != 1.0 {
		t.Errorf("criar deveria devolver _seq=1: %v", d["_seq"])
	}
	id := d["id"].(string)
	c, d = chamar(t, "own", "PUT", "spaces/s1/tx/"+id, map[string]string{"de": "2026-02-10T12:00:00.000Z"}, `{"descricao":"x"}`)
	esperar(t, "editar", c, 200)
	if d["_seq"] != 2.0 {
		t.Errorf("editar deveria devolver _seq=2: %v", d["_seq"])
	}
	c, d = chamar(t, "own", "DELETE", "spaces/s1/tx/"+id, map[string]string{"de": "2026-02-10T12:00:00.000Z"}, "")
	esperar(t, "excluir", c, 200)
	if d["_seq"] != 3.0 || d["ok"] != true {
		t.Errorf("excluir deveria devolver ok e _seq=3: %v", d)
	}
	if seqDoBanco(t, f) != 3 {
		t.Errorf("o contador no banco deveria estar em 3, esta em %v", seqDoBanco(t, f))
	}
	_, d = chamar(t, "mem", "GET", "spaces/s1/seq", nil, "")
	if d["seq"] != 3.0 {
		t.Errorf("o membro tambem le o seq: %v", d["seq"])
	}
}

func TestSeqNaoSobeQuandoNadaFoiGravado(t *testing.T) {
	f := cenario(t)
	// excluir o que nao existe, editar o que nao existe e escrita barrada pela trava nao mexem no contador
	chamar(t, "own", "DELETE", "spaces/s1/tx/naoexiste", map[string]string{"de": "2026-02-10T12:00:00.000Z"}, "")
	chamar(t, "own", "PUT", "spaces/s1/tx/naoexiste", map[string]string{"de": "2026-02-10T12:00:00.000Z"}, `{"descricao":"x"}`)
	chamar(t, "mem", "PUT", "spaces/s1/tx/iddono", map[string]string{"de": dataDono}, `{"descricao":"invasao"}`) // 403
	chamar(t, "own", "POST", "spaces/s1/tx", nil, `{"dataEvento":"lixo"}`)                                       // 400
	if got := seqDoBanco(t, f); got != 0 {
		t.Errorf("nenhuma dessas escritas vale, seq deveria ser 0, veio %v", got)
	}

	relogioFixo(t)
	conferir(t, "own", "ITAU", "2026-03", 0)
	chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-02-10T12:00:00.000Z", "ITAU", 5)) // 409: mes conferido
	if got := seqDoBanco(t, f); got != 0 {
		t.Errorf("escrita barrada pela trava nao conta, seq deveria ser 0, veio %v", got)
	}
}

func TestSeqNaImportacaoEmLote(t *testing.T) {
	f := cenario(t)
	lote := `{"itens":[` + corpoTx("2026-02-01T10:00:00.000Z", "ITAU", 1) + `,` + corpoTx("2026-02-02T10:00:00.000Z", "ITAU", 1) + `]}`
	c, d := chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, lote)
	esperar(t, "importar", c, 200)
	if d["gravados"] != 2.0 || d["_seq"] != 1.0 {
		t.Errorf("o lote inteiro soma 1 so: %v", d)
	}
	if seqDoBanco(t, f) != 1 {
		t.Errorf("seq no banco = %v", seqDoBanco(t, f))
	}
	// lote em que nada e gravado nao soma
	c, d = chamar(t, "own", "POST", "spaces/s1/tx/batch", nil, `{"itens":[{"dataEvento":"lixo"}]}`)
	esperar(t, "lote invalido", c, 200)
	if d["_seq"] != nil || seqDoBanco(t, f) != 1 {
		t.Errorf("lote sem nada gravado nao pode somar: %v", d)
	}
}

func TestSeqNaoSePerdeEmNumerosGrandes(t *testing.T) {
	f := cenario(t)
	// acima de 2^31 (o limite do inteiro de 32 bits do bug das 248 horas do Boeing 787) e perto de 2^53
	for _, inicio := range []float64{2147483647, 4294967295, 9007199254740990} {
		f.itens[chave(pkEspaco, "SEQ")] = map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pkEspaco}, "SK": &types.AttributeValueMemberS{Value: "SEQ"},
			"seq": &types.AttributeValueMemberN{Value: fmt.Sprintf("%.0f", inicio)},
		}
		c, d := chamar(t, "own", "POST", "spaces/s1/tx", nil, corpoTx("2026-02-10T12:00:00.000Z", "ITAU", 5))
		esperar(t, "criar", c, 201)
		if d["_seq"] != inicio+1 {
			t.Errorf("a partir de %.0f deveria devolver %.0f, veio %v", inicio, inicio+1, d["_seq"])
		}
		_, lido := chamar(t, "own", "GET", "spaces/s1/seq", nil, "")
		if lido["seq"] != inicio+1 {
			t.Errorf("a leitura deveria devolver %.0f, veio %v", inicio+1, lido["seq"])
		}
	}
}
