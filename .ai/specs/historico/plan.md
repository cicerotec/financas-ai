# Plan: histórico de versões e recuperação

Como construir o que o [`intent.md`](intent.md) pede. O intent diz **o quê e por quê**; este documento diz **como** e em
que ordem. O roteiro verificável (`spec.md`) vem depois de o dono aprovar este plan.

**Estado:** rascunho de 2026-10-15, escrito a partir do código de `backend/` (`main.go`, `store.go`, `fechamento.go`,
`authz.go`, `handler_test.go`) e do `template.yaml`. Nada foi implementado e nenhuma chave foi criada.

## Onde o código está hoje (o que o plan precisa respeitar)

| Fato | Onde | Consequência para o plan |
|---|---|---|
| Backend é um `package main` plano, com `db`, `table` e `agora()` globais | `store.go`, `main.go` | A refatoração em camadas inteira (item 32) é grande demais para entrar junto. O plan faz só o corte necessário (fase 0) |
| Quatro caminhos de escrita de lançamento, cada um com a sua lógica: `postTx`, `postTxBatch`, `putTx`, `delTx` | `main.go:199`, `:238`, `:356`, `:420` | O primeiro trabalho é juntá-los num só (`intent`, "Um único caminho de escrita") |
| `putTx` e `postTx` gravam com `PutItem`; só a mudança de data usa `TransactWriteItems` | `main.go:400` | Toda escrita passa a ser transação (registro + versão) |
| `postTxBatch` usa `BatchWriteItem`, sem condição, e **sobrescreve** quando o `id` vem no item | `main.go:238`, `:331` | Reimportar um backup é uma edição e precisa virar versão (decisão 5) |
| `delTx` apaga de verdade e é idempotente (item inexistente responde `ok`) | `main.go:420` | Passa a ser exclusão lógica (decisão 1) |
| `listarTx` filtra banco, status e texto **em Go**, depois do `Query` | `store.go:124` | Filtrar excluídos é um `continue` a mais, sem expressão nova no DynamoDB |
| A role da Lambda usa `DynamoDBCrudPolicy` na tabela inteira | `template.yaml:109` | O `Deny` precisa de uma `Statement` própria (fase 1) |
| O banco falso dos testes **não é atômico** e ignora condição em `TransactWriteItems` | `handler_test.go:67` | Os testes de "tudo ou nada" só valem depois de tornar o falso fiel (fase 0) |
| Tabela com `WriteCapacityUnits: 5` (provisionado) | `template.yaml:42` | Escrita transacional custa o dobro e a versão dobra o número de itens (decisão 3) |

## Decisões que este plan precisa que o dono confirme

Cada uma está marcada **[dono]** na fase onde bloqueia. Nenhum campo ou chave nova nasce sem este ok (princípio 3).

1. **Indicador de exclusão: `excluidoEm` (fechada pelo dono em 2026-10-15).** Data e hora (ISO) gravadas **só nos
   excluídos**. Sem campo = vivo: **sem backfill** e sem a dualidade de `ativo` (campo ausente e `true` querendo
   dizer a mesma coisa). Já informa a exclusão e quando ocorreu. O filtro pula quem tem `excluidoEm`
   (`attribute_not_exists(excluidoEm)` no DynamoDB, ou um `continue` em Go como em `listarTx`).
2. **A rota antiga `GET /tx` também esconde os excluídos.** O intent diz que só as rotas novas devolvem "só os
   ativos", mas o `DELETE` antigo passa a ser lógico: se o `GET` antigo não filtrar, o excluído **reaparece** nas
   telas atuais. Filtrar não muda o formato da resposta; devolve o mesmo que o `DELETE` sempre prometeu.
3. **Capacidade de escrita.** Uma escrita de 1 KB passa de 1 WCU para **4 WCU** (put + versão, ambos transacionais, a
   2 WCU cada). Uma importação de 25 itens passa de 25 para cerca de **100 WCU** contra 5 provisionados: o
   `gravarLote` repetiria com espera, e a tela de importação ficaria lenta. Opções: (a) subir a tabela para 25 WCU,
   que ainda cabe no always-free; (b) tornar a tabela `PAY_PER_REQUEST`; (c) manter 5 e aceitar a lentidão. **Proposta:
   (a)**, medindo em dev antes.
4. **Corte da refatoração.** Proposta: um **repository de lançamentos** com a interface do `intent` e o service de
   lançamento, injetados; o resto do backend (`fechamento`, `tags`, `avisos`, `cfg`) segue como está e é migrado nos
   itens seguintes do 32. Não se cria "camada pela metade": o que entra é completo para lançamentos.
5. **Importação em lote.** Proposta: o lote vira uma sequência de `TransactWriteItems` de até 25 lançamentos + 25
   versões (50 itens, abaixo do limite de 100). `id` que já existe é uma **edição** (`acao = update`); `id` novo é
   `insert`. A resposta mantém `gravados`, `rejeitados` e `pendentes`.
6. **Como o v1 nasce.** O intent manda migrar o estado atual como versão 1 antes da escrita nova. Proposta de
   reforço: o caminho de escrita **também cria a versão 1 sozinho** (com `acao = insert`, `porSub` vazio) quando o
   lançamento ainda não tem nenhuma versão, na **mesma transação** da edição. Assim a ordem entre deploy e migração
   deixa de ser frágil, e um lançamento que escapar da migração se corrige na primeira edição.
7. **Lista de excluídos (lacuna do intent).** O intent restringe a consulta às versões de **um** lançamento, mas
   um lançamento excluído **some das telas** e a pessoa não tem como achar o `id`. Proposta: `GET /tx?excluidos=1&de=&ate=`,
   lendo a partição viva e filtrando em Go, como `listarTx` já faz. **Sem chave nova** e sem a visão cronológica do
   espaço inteiro, que continua fora.

## Desenho

### Tipos e interfaces (package `main`, em arquivos novos)

```
lancamento.go      tipo Lancamento (campos conhecidos + extras), Mudanca, Versao, Acao
repo_lancamento.go interface RepositorioLancamentos (declarada aqui, usada pelo service) + implementação DynamoDB
servico_lancamento.go  Criar, Editar, Excluir, Importar, Restaurar, Versoes  (o único caminho de escrita)
```

- `Mudanca` = `{antes *Lancamento, depois *Lancamento, acao, por, em, deVersao}`. O repository recebe uma `Mudanca` e
  devolve o que gravou: **a interface não tem método de editar nem apagar versão**.
- O service **não importa** o SDK da AWS; o repository é o único lugar com `HIST#`, `TX#`, `excluidoEm` e expressões.
- Os controllers atuais (`postTx`, `putTx`, `delTx`, `postTxBatch`) viram finos: validam o corpo, chamam o service e
  montam a resposta de sempre.
- Efeitos derivados continuam **fora da transação** e chamados pelo service, na mesma ordem de hoje:
  `invalidarSaldos`, `garantirTagsServidor`, `sincronizarAviso`/`removerAviso`, `comSeq`.

### Chaves (aprovadas no intent; a única novidade é a decisão 1)

| Item | PK | SK | Atributos |
|---|---|---|---|
| Lançamento vivo (existe) | `SPACE#<sid>` | `TX#<dataEvento>#<id>` | campos de hoje + `excluidoEm` (só excluído) |
| Versão (novo) | `HIST#<sid>` | `TX#<id>#<versão com 8 dígitos>` | registro inteiro, `acao`, `versao`, `porSub`, `em`, `deVersao` |

`HIST#` e `SPACE#` nunca se misturam: o `sid` vem da **rota**, depois de `papel()` confirmar o acesso, e **nunca do
corpo**. Um teste garante que nenhuma rota monta `HIST#` com valor do cliente.

### Fluxo de escrita (um só)

1. Ler o lançamento atual (`GetItem`) e, para versão, a **última** do `id` (`Query` em `HIST#<sid>`,
   `begins_with(SK, "TX#<id>#")`, `ScanIndexForward=false`, `Limit=1`).
2. Checar a regra: trava do fechamento (`novasTravas().checar`) e permissão (`ehDono` para member).
3. `TransactWriteItems`, tudo ou nada:
   - **Put** do lançamento vivo (ou, na exclusão, o mesmo item com `excluidoEm`);
   - **Delete** da chave antiga, só quando a `dataEvento` mudou (hoje já é assim);
   - **Put** da versão com `attribute_not_exists(SK)`: se outra edição gravou a mesma versão, a transação cancela e o
     service responde `409`.
4. Efeitos derivados (acima), fora da transação.

Falha em qualquer item cancela a edição inteira. O caso de **disputa** (`TransactionCanceledException` com
`ConditionalCheckFailed` na versão) vira `409 conflito`, com a mesma mensagem que o `conflitoVersao` do fechamento usa.

### Rotas novas (cada uma com ação em `authz.go` e teste)

| Método e caminho | Ação | Quem |
|---|---|---|
| `GET /spaces/{sid}/tx/{id}/versoes` | `historico.ler` | owner: todas; member: só dos que criou |
| `POST /spaces/{sid}/tx/{id}/versoes/{n}/restaurar` | `historico.restaurar` | owner: qualquer; member: só dos que criou e que ainda existem |
| `GET /spaces/{sid}/tx?excluidos=1` (decisão 7) | `tx.ler` com filtro | owner; member vê só os que criou |

- A resposta de `versoes` já traz, por versão: `versao`, `acao`, `porSub` (e o apelido), `em`, o **diff** em relação à
  anterior (campos que mudaram, valor antes e depois) e `restauravel` + `motivo`. O front só desenha.
- `restauravel` junta três regras do backend: a trava do fechamento sobre o `alvo` da versão **e** sobre o estado
  atual; "já está ativo nessa versão" (`409`); permissão do papel. A trava de **fatura** do cartão não existe ainda
  (item 33): a lacuna está no intent e não é resolvida aqui.
- Restaurar um excluído é só do owner, no mesmo `authz` de `tx.excluir`.

### Segurança (IAM)

`DynamoDBCrudPolicy` fica, e entra uma `Statement` de `Deny` para `dynamodb:UpdateItem` e `dynamodb:DeleteItem` com
`ForAllValues:StringLike` em `dynamodb:LeadingKeys` = `HIST#*`. Dois pontos que o plan registra para não virar
surpresa:

- **O `Deny` não impede `PutItem` sobre uma versão existente**, porque o IAM não enxerga o conteúdo da tabela. A
  proteção é a **condição `attribute_not_exists(SK)` no código**, com teste no repository que falha se alguma
  expressão de versão não a carregar. O intent diz que o `Deny` cobre "editar nem apagar"; o `Put` com condição fecha a
  parte do "editar".
- `BatchWriteItem` e `TransactWriteItems` com `Delete` ou `Update` passam pelas mesmas ações, então o `Deny` vale
  para eles. **Verificar em dev com uma chamada real** (item de teste em `HIST#`), não só no papel.

## Fases (cada uma é um PR, cada uma deixa o app funcionando)

### Fase 0: um caminho de escrita, sem mudar comportamento
- Criar `Lancamento`, `Mudanca`, o repository e o service; mover `postTx`, `putTx`, `delTx` e `postTxBatch` para
  chamá-lo. **Nenhuma resposta, rota ou chave muda.** Ainda sem versão.
- Tornar o `bancoFake` fiel: `TransactWriteItems` atômico (aplica tudo ou nada), honra `attribute_not_exists`, devolve
  `TransactionCanceledException`; `Query` com `ScanIndexForward` e `Limit`.
- **Pronto quando:** todos os testes atuais passam sem edição de expectativa, e um teste novo prova que o controller
  não alcança o `db` sem passar pelo service.

### Fase 1: gravar versões e excluir logicamente  **[dono: decisões 1, 2, 3, 5, 6]**
- Chave `HIST#`, fluxo de escrita completo, `excluidoEm`, filtro em `listarTx` e no `GET /tx` antigo, `v1` preguiçoso.
- `template.yaml`: `Statement` de `Deny` e, se aprovado, a capacidade.
- `publico()` passa a remover `excluidoEm`. Aviso: `removerAviso` continua na exclusão e `sincronizarAviso` na restauração.
- **Pronto quando:** teste que falha se a versão não for gravada; teste de edição concorrente (segunda recebe `409`);
  teste de falha da versão abortando a edição; teste de importação que vira `update`; `security.md` e o modelo de dados
  (`HIST#`, exemplo fictício e `excluidoEm`) atualizados; deploy em `financas-dev` e uma tentativa real de
  `DeleteItem` em `HIST#` negada.

### Fase 2: migração do v1
- Script de uma vez (`scripts/`), com **modo simulação** que só conta e lista, e modo de gravação em lote com
  `attribute_not_exists(SK)` (reexecutável sem duplicar). Autor desconhecido: `porSub` vazio.
- Roda **primeiro em dev**, depois em prod, **com o dono acompanhando** (mexe em dado). Como o v1 também nasce
  preguiçoso (decisão 6), a ordem entre migração e deploy deixa de ser crítica.
- **Pronto quando:** contagem de `HIST#` bate com a de `TX#` no espaço, e uma segunda execução grava zero.

### Fase 3: leitura, diff e restauração (API)  **[dono: decisão 7]**
- Rotas novas, `authz.go` com teste (o que não está liberado é negado), diff e `restauravel` calculados no backend.
- **Pronto quando:** testes de permissão (owner, member dono, member de outro, restaurar excluído), trava do
  fechamento bloqueando com motivo, `409` ao restaurar o que já está ativo, `criadoPor` e `criadoEm` vindos da versão.

### Fase 4: tela
- **Demo antes do código** (guardrails, "Telas"), em artefato privado que abra no celular: Configurações →
  Histórico de versões, lista de excluídos, versões de um lançamento, diff, botão Restaurar desabilitado com o
  motivo, e a visão de computador. Espera o ok do dono.
- Depois da demo aprovada: implementar e testar em celular e janela larga, nos dois temas, com o pior caso (muitas
  versões, valor de sete dígitos, descrição longa).
- **Pronto quando:** o critério de tela do intent, e `padrao-de-telas.md` e a demo em dia na mesma entrega.

### Fase 5: aposentar as rotas antigas
- Item próprio, depois de nenhuma tela usar `PUT` e `DELETE` de `/tx`. Fora deste plan.

## Riscos e como o plan responde

| Risco | Resposta |
|---|---|
| Duas edições concorrentes perdem uma | A versão com `attribute_not_exists(SK)` faz a segunda falhar com `409` |
| Item vivo e versão divergem | Os dois na mesma transação; falha da versão aborta |
| `SALDO#` fica velho se `invalidarSaldos` falhar | Já é dívida do fechamento (intent); não entra aqui |
| Excluído reaparece em alguma tela | Decisão 2 + teste de `GET /tx` depois de `DELETE`; conferir os leitores de `TX#` além de `listarTx` (`avisos.go`) |
| Importação lenta com 5 WCU | Decisão 3, medida em dev na fase 1 |
| Registro de 400 KB | Versão guarda o registro inteiro; hoje um lançamento tem poucos KB, e o limite do item aborta a edição com erro claro |
| Restaurar bagunçar fatura do cartão | Lacuna do item 33, documentada no intent; a restauração herda a regra quando existir |

## Fora de escopo

Configurações (`CFG#*`), fechamentos (`SALDO#*`), perfil, aparência e avisos; visão cronológica do espaço; S3 para
histórico antigo; trava de fatura (item 33); refatoração do restante do backend em camadas (item 32).

## Entregas de documentação (em cada fase, não no fim)

- `.ai/security.md`: ações `historico.ler` e `historico.restaurar`, e o `Deny` (fase 1 e 3).
- `.ai/specs/modelo-de-dados/README.md` e o JSON de exemplo: `HIST#`, `excluidoEm` (fase 1).
- `.ai/README.md`: linha do `plan.md`. `BACKLOG.md`: itens 31 e 32 com o estado de cada fase.
- `.ai/padrao-de-telas.md` e a demo (fase 4).
