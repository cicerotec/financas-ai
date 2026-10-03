# Finanças pessoais: código e modelo de dados

Este documento acompanha `financas-app.html` (o código) e o arquivo `.json` gerado pelo botão **Exportar dados** (os dados). Com os dois você consegue reproduzir o app em outro banco e outro lugar.

## O que é o app
Uma única página HTML, com CSS e JavaScript dentro do próprio arquivo. Não tem framework nem etapa de build. A única biblioteca externa é a SheetJS (xlsx 0.18.5, via CDN), usada só para importar planilhas.

## O que o código pede ao ambiente do Claude (trocar ao migrar)
1. `claude.use("db")`: banco de documentos no estilo Firestore. Chamadas usadas:
   - `db.collection("lancamentos")` com `.add(obj)`, `.doc(id).update(obj)`, `.doc(id).delete()`
   - `db.doc("config/x")` com `.get()`, `.set(obj)`, `.onSnapshot(cb)`
   - consultas: `.where(campo, op, valor)` com `==`, `<`, `<=`, `>=` e `array-contains`; `.orderBy(campo, "asc"|"desc")`; `.limit(n)`; `.get()`; `.onSnapshot(cb)`
   - Em outro banco, o caminho mais curto é escrever um adaptador com esses mesmos métodos.
2. `claude.use("sample")` → `sample.json(prompt, {modelTier})`: pede a um modelo de IA que transforme texto livre ("paguei 87,40 no posto") em campos do lançamento e devolva JSON. Troque por uma chamada à API de IA da sua escolha.
3. `claude.use("downloads")` → `save({filename, data})`: baixa um arquivo. Troque por um download normal com `Blob` e `<a download>`.

Atenção: no ambiente do Claude os dados entregues pelo banco chegam **congelados** (não podem ser alterados). Por isso o código faz uma cópia com `clonar()` antes de alterar. Em outro banco essa cópia é inofensiva.

## Modelo de dados

### Coleção `lancamentos` (um documento por lançamento)
| campo | tipo | observação |
|---|---|---|
| `status` | texto | um dos status de `config/listas` (PAGO, PREVISTO, CRÉDITO IN, etc.) |
| `tipo` | texto | `"saida"` (gasto) ou `"entrada"` (receita) |
| `valor` | número | sempre positivo; o sinal vem de `tipo` |
| `descricao` | texto | |
| `nota` | texto | opcional |
| `dataEvento` | texto ISO 8601 (UTC) | data e hora do evento; é o campo de ordenação |
| `banco` | texto | um dos bancos de `config/listas` |
| `tags` | lista de textos | sem hierarquia nem grupos |
| `excluirDoTotal` | booleano | verdadeiro para status CONTAS e TRANSFERINDO |
| `oculto` | booleano | quando verdadeiro, some do histórico e não entra em saldos |
| `criadoEm` | texto ISO | presente em importações e transferências |
| `transferParId` | texto | liga as duas pontas de uma transferência entre bancos |
| `faturaAjuste` | -1, 0 ou 1 | só em status de crédito: 1 = a compra vai para a fatura seguinte à da data; -1 = para a anterior; ausente = pela data (CRÉDITO EX antigo sem esse campo conta como 1) |
| `valorEstimado` | booleano | só em status de crédito: valor ainda estimado (ex.: compra em dólar); vira falso quando o valor real é corrigido |

No arquivo exportado cada lançamento traz também o campo `id` (o id do documento).

### Documentos de configuração
- `config/listas`:
  - `status`: lista de textos
  - `banco`: lista de textos
  - `statusReal`: mapa status → booleano ("conta como real"; ausente ou verdadeiro conta, só `false` exclui; padrão `PREVISTO`, `TRANSFERINDO` e `CREDITANDO` = false)
  - `afetaSaldoBanco`: mapa status → booleano (só `true` afeta o saldo do banco; ausente = não afeta)
  - `statusCor`: mapa status → cor em hexadecimal
  - `estiloCor`: `"linha"`, `"faixa"`, `"ambos"` (padrão) ou `"nenhum"`
- `config/tags`: `{ usadas: [lista de textos] }`, todas as tags conhecidas.
- `config/combos`: `{ lista: [ { tags: [textos] } ] }`, combos de tags salvos pelo usuário. Combos "sugeridos" não são guardados: o app os recalcula contando conjuntos de tags repetidos nos lançamentos.
- `config/cartoes`: `{ porBanco: { "<banco>": { diaPadrao: número, faturas: { "AAAA-MM": { fechamento: "AAAA-MM-DD", valorBanco: número } } } } }`. A presença do banco aqui o marca como cartão. `AAAA-MM` é o mês em que a fatura fecha. `fechamento` é opcional e, se faltar, vale o dia da fatura anterior que tiver um (ou `diaPadrao`). `valorBanco` é o total que o app do cartão mostra, usado na conferência.
- `config/saldos`: `{ porBanco: { "<banco>": { saldoInicial: número, dataInicio: "AAAA-MM-DD" } } }`.

## Regras de negócio que o código aplica
- **Saldo de um banco** = `saldoInicial` + soma dos lançamentos daquele banco que: não estão ocultos, têm status com `afetaSaldoBanco = true` e têm `dataEvento` a partir de `dataInicio`. Entrada soma e saída subtrai.
- **Transferência entre bancos** cria dois lançamentos com status `CONTAS`: uma saída no banco de origem e uma entrada no de destino, com o mesmo `transferParId`.
- **Faturas do cartão:** a fatura que fecha no mês M cobre as compras de crédito do dia seguinte ao fechamento da fatura anterior até o dia de fechamento de M, inclusive. Cada fatura tem a própria data de fechamento. O ajuste `faturaAjuste` empurra a compra para a fatura vizinha. Compras de crédito = status que começam com CRÉDITO (sem acento na comparação), não ocultas e com "conta como real"; estorno (`tipo = entrada`) subtrai.
- **Conferência da fatura fechada:** diferença = total calculado − `valorBanco`. Candidatos: registros (1 a 3) até 5 dias do corte cuja soma é igual à diferença; se o calculado for maior, tiram-se da fatura; se for menor, buscam-se nas faturas vizinhas.
- **Detecção de duplicado** ao salvar: mesma combinação de status, descrição (sem maiúsculas e sem espaços nas pontas), valor com 2 casas, banco e `dataEvento`.
- **Cor de um status**: `statusCor[status]`; se não houver, uma cor padrão por nome do status; se também não houver, cinza.

## Formato do arquivo exportado (versão 2)
```
{
  "app": "Finanças pessoais",
  "versaoExportacao": 2,
  "exportadoEm": "2026-09-30T12:00:00.000Z",
  "contagem": { "lancamentos": 1234 },
  "colecoes": { "lancamentos": [ { "id": "...", "status": "...", ... } ] },
  "documentos": {
    "config/listas": { ... },
    "config/tags": { ... },
    "config/combos": { ... } ,
    "config/saldos": { ... },
    "config/cartoes": { ... }
  }
}
```
Um documento que não existia no banco aparece como `null`.

## Como reimportar em outro banco
1. Para cada item de `colecoes.lancamentos`: criar um registro com `id` igual a `item.id` e os demais campos como estão.
2. Para cada chave de `documentos` que não seja `null`: gravar o conteúdo no caminho indicado (ou numa tabela de configuração equivalente).
3. Conferir: `contagem.lancamentos` deve bater com o número de registros criados.

## Limitações conhecidas do código atual
- O banco do ambiente Claude devolve no máximo 1000 documentos por consulta. O histórico normal carrega os 1000 mais recentes, e o cálculo de saldo de cada banco lê no máximo 1000 lançamentos daquele banco. Em um banco próprio, remova esses limites.
- O Exportar lê tudo em páginas de 1000 e para com um aviso se mais de 1000 lançamentos tiverem exatamente a mesma data e hora, em vez de gerar um arquivo incompleto.
