# Modelo de dados do DynamoDB (exemplos fictícios)

Uma tabela só (`FinancasApp`, `FinancasApp-dev` no dev), com `PK` e `SK`. Estes arquivos mostram **um item de cada tipo
de chave que existe**, para saber quais chaves há e o formato de cada uma.

| Arquivo | Para quê |
|---|---|
| [`dynamodb-payload-exemplo.json`](dynamodb-payload-exemplo.json) | Os itens em JSON simples, legível |
| [`dynamodb-batch-write-request.json`](dynamodb-batch-write-request.json) | Os mesmos itens no formato do `BatchWriteItem` (`aws dynamodb batch-write-item`) |

**Só dados fictícios.** O repositório é público: nunca coloque aqui lançamento, saldo, e-mail ou `sub` reais. Exportações
de dados reais ficam em `/local/`, que o git ignora. A pasta tem uma exceção no `.gitignore` (`!.ai/specs/**/*.json`)
justamente porque o conteúdo é inventado.

## Chaves que existem

| `PK` | `SK` | O que é | Onde está no código |
|---|---|---|---|
| `USER#<sub>` | `SPACE#<sid>` | Vínculo da pessoa com o espaço: `role` (`owner` ou `member`), `apelido` e `telegramChatId` | `store.go` (`papel`), `perfil.go` |
| `SPACE#<sid>` | `META` | Dados do espaço: `nome` e `ownerSub` | `main.go`, `avisos.go` |
| `SPACE#<sid>` | `SEQ` | Contador de alterações (`seq`), somado a cada escrita de lançamento | `seq.go` |
| `SPACE#<sid>` | `CFG#LISTAS` | Listas de status e bancos, cores e as caixas (`statusReal`, `statusFuturo`, `afetaSaldoBanco`) | `main.go` (`cfgNomes`) |
| `SPACE#<sid>` | `CFG#TAGS` | Tags já usadas (`usadas`); o member só acrescenta | `tags.go` |
| `SPACE#<sid>` | `CFG#COMBOS` | Combos de tags | `main.go` |
| `SPACE#<sid>` | `CFG#SALDOS` | Início do controle por banco (`dataInicio`, `saldoInicial`) | `fechamento.go` |
| `SPACE#<sid>` | `CFG#CARTOES` | Por cartão: `diaPadrao`, `inicio` e `faturas["AAAA-MM"]` (`fechamento`, `valorBanco`, `conferida`) | front (`atualizarConfigCartoes`) |
| `SPACE#<sid>` | `SALDO#<banco>` | Fechamento mensal do banco: `caches`, `conferidos`, `invalidoDe`, `base` | `fechamento.go` |
| `SPACE#<sid>` | `USER#<sub>#CFG#APARENCIA` | Aparência da própria pessoa (cores, estilo) | `main.go` |
| `SPACE#<sid>` | `TX#<dataEvento>#<id>` | Um lançamento | `main.go` (`postTx`), `store.go` (`txSK`) |

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

## Regra para mexer neste modelo
Nenhuma chave ou campo novo no DynamoDB sem discutir antes com o dono (veja [`guardrails.md`](../../guardrails.md)).
Quando uma chave nascer, **entra aqui na mesma entrega**, com um exemplo fictício nos dois JSON.
