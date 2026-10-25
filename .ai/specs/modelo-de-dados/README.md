# Modelo de dados do DynamoDB (exemplos fictícios)

Uma tabela só (`FinancasApp`, `FinancasApp-dev` no dev), com `PK` e `SK`. Estes arquivos mostram **um item de cada tipo
de chave que existe**, para saber quais chaves há e o formato de cada uma.

| Arquivo | Para quê |
|---|---|
| [`dynamodb-payload-exemplo.json`](dynamodb-payload-exemplo.json) | Os itens em JSON simples, legível |
| [`dynamodb-batch-write-request.json`](dynamodb-batch-write-request.json) | Os mesmos itens no formato do `BatchWriteItem` (`aws dynamodb batch-write-item`) |
| [`dynamodb-single-table/`](dynamodb-single-table/) | Um arquivo por item, como ele fica na tabela (`PK`, `SK` e atributos), com identificadores truncados (`8c95812f-...`) |

**Só dados fictícios.** O repositório é público: nunca coloque aqui lançamento, saldo, e-mail ou `sub` reais. Exportações
de dados reais ficam em `/local/`, que o git ignora. A pasta tem uma exceção no `.gitignore` (`!.ai/specs/**/*.json`)
justamente porque o conteúdo é inventado.

## Chaves que existem

| `PK` | `SK` | O que é | Onde está no código |
|---|---|---|---|
| `USER#<sub>` | `SPACE#<sid>` | Vínculo da pessoa com o espaço: `role` (`owner` ou `member`), `apelido` e `telegramChatId` | `store.go` (`papel`), `perfil.go` |
| `SPACE#<sid>` | `META` | Dados do espaço: `nome` e `ownerSub` | `main.go`, `avisos.go` |
| `SPACE#<sid>` | `SEQ` | Contador de alterações (`seq`), somado a cada escrita de lançamento | `seq.go` |
| `SPACE#<sid>` | `CFG#LISTAS` | Listas de status e bancos, cores e as caixas (`statusReal`, `statusFuturo`, `afetaSaldoBanco`) | `auth/authz.go` (`cfgNomes`) |
| `SPACE#<sid>` | `CFG#TAGS` | Tags já usadas (`usadas`); o member só acrescenta | `tags.go` |
| `SPACE#<sid>` | `CFG#COMBOS` | Combos de tags | `main.go` |
| `SPACE#<sid>` | `CFG#SALDOS` | Início do controle por banco (`dataInicio`, `saldoInicial`) | `fechamento.go` |
| `SPACE#<sid>` | `CFG#CARTOES` | **Hoje por banco** (`porBanco[banco]`): `diaPadrao`, `inicio` e `faturas["AAAA-MM"]` (`fechamento`, `valorBanco`, `conferida`; **`conferida = {total, em, por}` é gravado só pelo servidor**, no `POST .../conferir`, com o total que ele calculou). **Dívida:** o cartão não é um banco (pode não ter nenhum, e quem paga a fatura é uma escolha); deve virar uma entidade própria, por cartão, no item 33 do BACKLOG | comandos do servidor (`/cartoes`, item 33a); o front ainda grava pelo `PUT /cfg/cartoes` até a fase B |
| `SPACE#<sid>` | `SALDO#<banco>` | Fechamento mensal do banco: `caches`, `conferidos`, `invalidoDe`, `base` (gravados **só pelo servidor**, ver abaixo) | `service/saldos.go`, `service/fechamento.go` |
| `SPACE#<sid>` | `USER#<sub>#CFG#APARENCIA` | Aparência da própria pessoa (cores, estilo) | `main.go` |
| `SPACE#<sid>` | `TX#<dataEvento>#<id>` | Um lançamento. Excluído logicamente, ganha `excluidoEm` (histórico, Fase 1) | `repository/lancamento.go` (`Salvar`), `repository/dynamo.go` (`TxSK`) |
| `HIST#<sid>` | `TX#<id>#<versão com 8 dígitos>` | Uma versão de um lançamento (histórico, Fase 1) | `repository/lancamento.go` (`Salvar`, `HistSK`); veja [`specs/historico/`](../historico/spec.md) |

Os itens `CFG#*` e `SALDO#*` têm `version`: é o controle otimista (o cliente manda a versão que leu; se mudou, a
gravação é recusada com conflito).

## Lançamento (`TX#...`)

- **A chave:** `dataEvento` em UTC, ISO 8601 com milissegundos (`2026-09-28T22:08:00.000Z`), e `id` de 20 caracteres
  (minúsculas e dígitos), gerado no servidor ou na tela.
- **Obrigatórios** (o backend recusa sem eles): `dataEvento`, `valor` (número; zero e negativo são aceitos), `tipo`
  (`entrada` ou `saida`), `status` e `banco`.
- **Colocados pelo servidor:** `id`, `criadoEm`, `criadoPor` (o `sub` de quem criou).
- **Opcionais:** `descricao`, `nota`, `tags`, `excluirDoTotal`, `oculto`, `faturaAjuste` (`-1`, `0` ou `+1`),
  `valorEstimado`, `transferParId` (liga as duas pontas de uma transferência), `aviso` (`{ dias: [3,1,0], insistir }`) e
  `serie` (`{ id, indice, intervalo }`). O backend valida `aviso` e `serie`; os demais ele grava como vieram.

### Campo novo no lançamento: `excluidoEm`
Texto ISO com milissegundos (`2026-09-22T10:15:30.120Z`), gravado **só** quando o lançamento é excluído, e removido se
ele for restaurado. **Sem o campo, o lançamento está vivo**: nada foi migrado. O servidor nunca devolve `excluidoEm` em
`GET /tx`; os excluídos somem das listas. Exemplo: [`space-tx_excluida.json`](dynamodb-single-table/space-tx_excluida.json).

## Versão do histórico (`HIST#...`)
Uma partição própria por espaço (`HIST#<sid>`), para o `Deny` de `UpdateItem` e `DeleteItem` por IAM
(`dynamodb:LeadingKeys` só enxerga a `PK`). A `SK` é `TX#<id>#<versão com 8 dígitos>` e usa o **`id`**, não a chave viva,
porque a chave do lançamento muda quando a data é editada.

| Atributo | O que é |
|---|---|
| `versao` | Número, igual ao sufixo da `SK`; a primeira é `1` |
| `acao` | `insert`, `update`, `delete` ou `restore`; `insert` só existe como a versão 1 (estado de antes da primeira alteração; "Original" na tela) |
| `registro` | O lançamento **como ficou depois** da ação (na exclusão, com `excluidoEm`); na versão 1 preguiçosa, **como era antes** |
| `porSub` | `sub` de quem agiu; vazio na versão 1 preguiçosa (autor desconhecido) |
| `em` | Texto ISO em UTC, com milissegundos |
| `deVersao` | Só no `restore`: a versão de origem |
| `confirmouMesFechado` | `true` só na versão em que a pessoa **confirmou** mexer em mês fechado por data (item 38) e a confirmação era necessária; nunca na versão 1 preguiçosa nem no lançamento vivo. Exemplo: [`hist-versao_update_confirmada.json`](dynamodb-single-table/hist-versao_update_confirmada.json) |

O histórico só recebe versões novas, gravadas junto com o lançamento na mesma `TransactWriteItems`, com `attribute_not_exists(SK)` (a segunda de duas edições concorrentes falha com `409`). A role da Lambda tem `Deny` de `UpdateItem` e `DeleteItem` em `HIST#*` (`template.yaml`, `HistoricoImutavel`).
Sem migração: a versão 1 nasce na primeira edição ou exclusão de **qualquer** lançamento que ainda não tem versão (desde 2026-10-23, criar e importar `id` novo não gravam versão, exceto quando exigiram confirmação de mês fechado). A vida de um lançamento (versão
1 preguiçosa, edição, exclusão) está em [`hist-versao_insert.json`](dynamodb-single-table/hist-versao_insert.json),
[`_update`](dynamodb-single-table/hist-versao_update.json) e [`_delete`](dynamodb-single-table/hist-versao_delete.json).

## Fechamento do banco (`SALDO#<banco>`)
Desde o item 40 (fase A) o **servidor** calcula o saldo e grava este item; o navegador não manda mais nenhum número.
- `caches`: o fechamento (saldo ao fim do mês) de cada mês **encerrado** (anterior ao atual), só para não reler o histórico inteiro. Gravado por `GET /saldos`
  com o controle otimista da `version` lida **antes** dos lançamentos; só existe com `dataInicio` configurada.
- `base`: a assinatura da configuração com a qual os caches foram calculados: `v2|<saldoInicial>|<dataInicio>|<status que afetam o saldo>/<status de evento futuro>`
  (os nomes ligados, em ordem alfabética). Um cache com outra `base` não vale. O prefixo `v2` invalidou, de uma vez, os caches que o navegador gravava antes.
- `invalidoDe`: o mês mais antigo alterado desde o último cache; a partir dele o cache não vale. Qualquer escrita que mexe no saldo o marca e muda a `version`.
- `conferidos[AAAA-MM]`: `{ saldo, real, em, por, ajuste? }`. `saldo` é o que o **servidor** calculou na conferência (não o do cliente); `real` é o extrato; `ajuste`, o lançamento
  de ajuste que a própria conferência criou (com sinal). Exemplo: [`space-saldo.json`](dynamodb-single-table/space-saldo.json).

## Regra para mexer neste modelo
Nenhuma chave ou campo novo no DynamoDB sem discutir antes com o dono (veja [`guardrails.md`](../../guardrails.md)).
Quando uma chave nascer, **entra aqui na mesma entrega**, com um exemplo fictício nos dois JSON.
