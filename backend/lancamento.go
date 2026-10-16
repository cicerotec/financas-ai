package main

// Lancamento e um lancamento financeiro. Os campos sao dinamicos (aviso, serie e extras do front), por isso
// segue sendo um mapa; as regras ficam em funcoes puras (validarTx, alvoDe, mudaSaldo, tagsDe...).
type Lancamento = doc

// Mudanca e uma escrita de lancamento entregue ao repository. E o que o service pede; as chaves do banco
// (PK, SK) e as expressoes do DynamoDB sao assunto so do repository.
type Mudanca struct {
	Antes        Lancamento // estado anterior; nil na criacao
	Depois       Lancamento // estado novo, ja validado
	DataAnterior string     // dataEvento sob a qual o lancamento estava gravado (chave antiga); vazio na criacao
}

// identidade identifica um lancamento dentro de um espaco sem expor a chave do banco.
func identidade(l Lancamento) string {
	data, _ := l["dataEvento"].(string)
	id, _ := l["id"].(string)
	return data + "#" + id
}
