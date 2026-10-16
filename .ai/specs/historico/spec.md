# Spec: histórico de versões e recuperação

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Cada regra tem um número (`R`) e
cada teste tem um id (`F<fase>-T<n>`), para o PR dizer exatamente o que cobriu. Quando um texto daqui e o do intent
divergirem, vale o plan, que registra as decisões do dono de 2026-10-15.

**Estado:** rascunho de 2026-10-16, aguardando o ok do dono. Nada foi implementado nem criado no DynamoDB.

**Como usar:** uma fase por PR, na ordem. Cada fase só termina quando todos os seus testes passam e o "Pronto quando"
está marcado. Teste de regra de negócio é teste de backend (`go test ./...` em `backend/`); o front não decide nada.

## Dados

### Item vivo (já existe, com um campo novo só nos excluídos)

`PK = SPACE#<sid>`, `SK = TX#<dataEvento>#<id>`. Ganha **`excluidoEm`** (string ISO `2006-01-02T15:04:05.000Z`), gravado
**só** na exclusão lógica e **removido** na restauração. Sem o campo, o lançamento está vivo. Nenhum outro campo novo.

### Versão (chave nova)

| Atributo | Tipo | Regra |
|---|---|---|
| `PK` | string | `HIST#<sid>` (o `sid` vem da rota, depois de `papel()`; nunca do corpo) |
| `SK` | string | `TX#<id>#<versão com 8 dígitos>`, ex.: `TX#k3x9a1m7q2w8e5r4t6y1#00000003` |
| `versao` | número | igual ao sufixo da `SK`; a primeira é `1` |
| `acao` | string | `insert`, `update`, `delete` ou `restore` |
| `registro` | mapa | o lançamento **como ficou depois** da ação (na exclusão, com `excluidoEm`); no v1 preguiçoso, **como era antes** |
| `porSub` | string | `sub` de quem agiu; **vazio** no v1 preguiçoso (autor desconhecido) |
| `em` | string | ISO em UTC, com milissegundos |
| `deVersao` | número | só no `restore`: a versão de origem |

`registro` é um **mapa aninhado**, não campos soltos, para nenhum campo do lançamento colidir com `acao`, `versao` ou
`em`. Exemplo fictício:

```json
{
  "PK": "HIST#esp-demo", "SK": "TX#k3x9a1m7q2w8e5r4t6y1#00000002",
  "versao": 2, "acao": "update", "porSub": "sub-demo-1", "em": "2026-09-14T13:02:11.482Z",
  "registro": {
    "id": "k3x9a1m7q2w8e5r4t6y1", "dataEvento": "2026-09-10T12:00:00.000Z", "descricao": "Mercado", "valor": 187.5,
    "tipo": "saida", "status": "PAGO", "banco": "BANCO-A", "criadoPor": "sub-demo-1", "criadoEm": "2026-09-10T12:05:00.000Z"
  }
}
```

## Regras

**Escrita**
- **R1.** Toda escrita de lançamento (`postTx`, `postTxBatch`, `putTx`, `delTx` e a restauração) passa pelo **service**.
  Nenhum controller chama `db` nem importa o SDK da AWS.
- **R2.** Toda escrita é **uma** `TransactWriteItems` com: o put do lançamento vivo, o put da versão (com
  `attribute_not_exists(SK)`) e, só se a `dataEvento` mudou, o delete da chave antiga. Falhou qualquer item: nada muda.
- **R3.** `versao` = (maior versão do `id`) + 1. Duas escritas concorrentes disputam a mesma `SK`: a segunda recebe
  `409` com `codigo: "conflito_versao"` e nada é gravado.
- **R4.** **Criar** grava a versão 1 com `acao = insert`. **Editar** (inclui ocultar, ajustar fatura e mudar a data)
  grava uma versão `update`; mudar a data gera **uma** versão, não duas.
- **R5. Versão 1 preguiçosa.** Se o lançamento não tem nenhuma versão (legado) e vai ser editado ou excluído, a mesma
  transação grava duas versões: **v1** = estado de antes (`insert`, `porSub` vazio) e **v2** = estado de depois.
- **R6. Excluir** é lógico: grava `excluidoEm` no item vivo e uma versão `delete`. O item continua na tabela. Excluir
  o que já está excluído, ou o que não existe, responde `200 {"ok": true}` como hoje, **sem versão nova**.
- **R7.** A trava do fechamento (`novasTravas().checar`) roda **antes** de qualquer gravação e responde `409`
  `codigo: "mes_conferido"`, igual a hoje. Sem escrita, não há versão.
- **R8.** Os efeitos derivados ficam **fora** da transação e na mesma ordem de hoje: `invalidarSaldos`,
  `garantirTagsServidor`, `sincronizarAviso` (e `removerAviso` na exclusão), `comSeq` (uma vez por escrita).
- **R9.** Nenhum código nem policy edita ou apaga versão: a interface do repository só tem `AnexarVersao` (via
  `SalvarLancamento`), e o `template.yaml` nega `UpdateItem` e `DeleteItem` em `HIST#*`.
- **R10.** **Importação:** cada lote é uma transação com até 25 lançamentos e 25 versões. `id` novo = `insert`;
  `id` existente = `update` (e R5 se não tinha versão). A trava rejeita item a item (`rejeitados`). A resposta mantém
  `gravados`, `rejeitados` e `pendentes`; lote que falha inteiro volta em `pendentes`.

**Leitura**
- **R11.** `GET /tx` e qualquer listagem de lançamentos **não devolvem** item com `excluidoEm`. `publico()` remove
  `excluidoEm` e `PK`/`SK`.
- **R12.** O mês de uma alteração é o de `em` **no fuso do app** (`fusoApp`, `America/Sao_Paulo`), o mesmo de
  `mesAtual()`, não o UTC.

**Restauração**
- **R13.** Restaurar a versão `n` grava uma versão **nova** (`acao = restore`, `deVersao = n`, `porSub` = quem restaurou)
  com o conteúdo de `registro` da versão `n`. `criadoPor` e `criadoEm` voltam **da versão**.
- **R14.** Lançamento excluído: restaurar **reativa** (remove `excluidoEm`) no mesmo `id`; se a `dataEvento` da versão
  difere da do item excluído, vale a mecânica de mudança de chave (R2).
- **R15.** Restaurar quando o conteúdo vivo já é igual ao da versão, e o item não está excluído, responde `409` com
  `codigo: "ja_nessa_versao"` e **não grava**.
- **R15b.** Uma versão cujo `registro` tem `excluidoEm` (a versão `delete`) **não é restaurável**: restaurar uma
  exclusão seria excluir de novo. `restauravel = false` com o motivo "Esta versão é a exclusão. Restaure uma versão
  anterior."; `POST .../restaurar` nela responde `409` com `codigo: "versao_de_exclusao"`.
- **R16.** A trava vale para a versão **e** para o estado atual: se qualquer um dos dois cai em mês conferido, a
  restauração é bloqueada (`mes_conferido`) e a versão aparece como não restaurável, com o motivo. **Fechamento
  conferido é absoluto.**
- **R17.** A trava de **fatura** do cartão não existe (item 33): restaurar lançamento de cartão em fatura conferida
  **não** é bloqueado, e isso está documentado no intent.

## Contratos de API

Erros seguem o formato de hoje (`{"erro": "...", "codigo": "..."}`).

### `GET /spaces/{sid}/historico?mes=AAAA-MM&q=` (ação `historico.ler`)

- `mes` é obrigatório e deve ser um mês válido (`400` se faltar ou for inválido). `q` é opcional.
- Lê as versões do espaço (`Query` em `HIST#<sid>`, `begins_with(SK, "TX#")`), filtra por `em` no mês (R12) e agrupa
  por `id`. Casa `q` (sem diferenciar maiúsculas) em `descricao` e `nota` de **qualquer versão do mês**.
- **Teto de 5.000 versões lidas.** Passou: para de ler e devolve `"truncado": true`.
- Member: só os `id` cujo `registro.criadoPor` é o próprio `sub`.

```json
{
  "mes": "2026-09", "truncado": false,
  "itens": [
    { "id": "k3x9a1m7q2w8e5r4t6y1", "descricao": "Mercado", "valor": 187.5, "tipo": "saida", "dataEvento": "2026-09-10T12:00:00.000Z",
      "banco": "BANCO-A", "status": "PAGO", "excluido": false, "excluidoEm": null, "acao": "update", "versao": 2,
      "em": "2026-09-14T13:02:11.482Z", "porSub": "sub-demo-1", "porNome": "Ana" }
  ]
}
```

Uma linha por lançamento, com a **última** versão do mês, ordenada por `em` decrescente. `excluidoEm` vem do `registro`
da última versão do mês (`null` se vivo) e `excluido` é `excluidoEm != null`. `porNome` é o `apelido` do vínculo
`USER#<sub>` / `SPACE#<sid>` (vazio se não houver).

### `GET /spaces/{sid}/tx/{id}/versoes` (ação `historico.ler`)

Todas as versões do `id`, da mais nova para a mais antiga. Sem nenhuma versão mas com o lançamento vivo:
`200` com `"semHistorico": true`, `"versoes": []` e `"atual"`. Sem versão e sem lançamento: `404`.

```json
{
  "id": "k3x9a1m7q2w8e5r4t6y1", "excluido": false, "semHistorico": false,
  "versoes": [
    { "versao": 2, "acao": "update", "porSub": "sub-demo-1", "porNome": "Ana", "em": "2026-09-14T13:02:11.482Z", "deVersao": null,
      "diff": [ { "campo": "valor", "antes": 180, "depois": 187.5 } ],
      "registro": { "id": "k3x9a1m7q2w8e5r4t6y1", "valor": 187.5 },
      "restauravel": true, "motivo": "" },
    { "versao": 1, "acao": "insert", "porSub": "", "porNome": "", "em": "2026-09-14T13:02:11.482Z", "deVersao": null,
      "diff": [], "registro": { "id": "k3x9a1m7q2w8e5r4t6y1", "valor": 180 },
      "restauravel": false, "motivo": "O lançamento já está nessa versão." }
  ]
}
```

- `diff` compara `registro` com o da versão anterior (a v1 tem `[]`), ignora `PK`/`SK` e lista só campos que mudaram.
- `restauravel` e `motivo` vêm do backend (R15, R16, permissão). O front só desenha.

### `POST /spaces/{sid}/tx/{id}/versoes/{n}/restaurar` (ação `historico.restaurar`)

Corpo vazio. `200` devolve o lançamento (como o `putTx`, com `_seq`). Erros: `400` (`n` inválido), `403` (papel), `404`
(versão ou lançamento inexistente), `409` com `mes_conferido`, `ja_nessa_versao` ou `conflito_versao`.

### Rotas atuais (contrato mantido)

`POST /tx`, `PUT /tx/{id}`, `DELETE /tx/{id}`, `POST /tx/batch` e `GET /tx` **não mudam de formato**. Mudam por dentro:
passam pelo service (R1), gravam versão (R2) e `DELETE` é lógico (R6); `GET /tx` esconde os excluídos (R11).

## Exemplos (curl)

Um lançamento ao longo da vida: criar, editar, excluir, achar no histórico, ver as versões e restaurar. Só dados
fictícios. Os campos novos são `excluidoEm` (no item vivo e em `registro`), `registro`, `acao`, `versao` e `deVersao`.

```bash
API="https://<id>.lambda-url.<regiao>.on.aws"     # Function URL (web/config.js)
SID="esp-demo"                                    # id do espaço
TOKEN="<id token do Cognito>"                     # o front manda como "Authorization: Bearer <token>"
ID="k3x9a1m7q2w8e5r4t6y1"
DE="2026-09-10T12:00:00.000Z"                     # dataEvento atual (as rotas PUT e DELETE pedem ?de=)
H=(-H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json")
```

### 1. Criar (`POST /tx`) grava o lançamento e a versão 1 (`insert`)

```bash
curl -s -X POST "$API/spaces/$SID/tx" "${H[@]}" -d '{
  "dataEvento": "2026-09-10T12:00:00.000Z", "descricao": "Mercado", "valor": 180,
  "tipo": "saida", "status": "PAGO", "banco": "Nubank", "tags": ["mercado-mensal"]
}'
```
```json
{ "id": "k3x9a1m7q2w8e5r4t6y1", "dataEvento": "2026-09-10T12:00:00.000Z", "descricao": "Mercado", "valor": 180,
  "tipo": "saida", "status": "PAGO", "banco": "Nubank", "tags": ["mercado-mensal"],
  "criadoEm": "2026-09-10T12:05:00.000Z", "criadoPor": "sub-demo-1", "_seq": 101 }
```
Status `201`. A resposta é igual à de hoje: **não traz** `excluidoEm`, `registro` nem versão. No DynamoDB, além do
item vivo, nasce `HIST#esp-demo` / `TX#k3x9a1m7q2w8e5r4t6y1#00000001` com `acao = insert` e `registro` = o lançamento.

### 2. Editar (`PUT /tx/{id}?de=`) grava a versão 2 (`update`)

```bash
curl -s -X PUT "$API/spaces/$SID/tx/$ID?de=$DE" "${H[@]}" -d '{ "valor": 187.5 }'
```
```json
{ "id": "k3x9a1m7q2w8e5r4t6y1", "dataEvento": "2026-09-10T12:00:00.000Z", "descricao": "Mercado", "valor": 187.5,
  "tipo": "saida", "status": "PAGO", "banco": "Nubank", "tags": ["mercado-mensal"],
  "criadoEm": "2026-09-10T12:05:00.000Z", "criadoPor": "sub-demo-1", "_seq": 102 }
```
Status `200`. Em um lançamento **legado** (sem versões), a mesma chamada grava duas versões na mesma transação: a **v1**
(estado de antes, `acao = insert`, `porSub` vazio) e a **v2** (`update`).

### 3. Excluir (`DELETE /tx/{id}?de=`) é lógico: grava `excluidoEm` e a versão 3 (`delete`)

```bash
curl -s -X DELETE "$API/spaces/$SID/tx/$ID?de=$DE" "${H[@]}"
```
```json
{ "ok": true, "_seq": 103 }
```
Status `200`, a mesma resposta de hoje. **O que muda é o que fica gravado.** Item vivo no DynamoDB, agora com o campo novo:
```json
{ "PK": "SPACE#esp-demo", "SK": "TX#2026-09-10T12:00:00.000Z#k3x9a1m7q2w8e5r4t6y1",
  "id": "k3x9a1m7q2w8e5r4t6y1", "descricao": "Mercado", "valor": 187.5, "banco": "Nubank", "status": "PAGO", "tipo": "saida",
  "dataEvento": "2026-09-10T12:00:00.000Z", "criadoPor": "sub-demo-1", "criadoEm": "2026-09-10T12:05:00.000Z",
  "excluidoEm": "2026-09-14T15:40:05.221Z" }
```
Versão 3 (`HIST#esp-demo` / `TX#k3x9a1m7q2w8e5r4t6y1#00000003`): `acao = delete`, `porSub = sub-demo-1`,
`em = 2026-09-14T15:40:05.221Z` e `registro` **com** `excluidoEm`. Repetir o `DELETE` responde o mesmo `200` e **não**
grava versão (R6).

### 4. Listar (`GET /tx`) já não mostra o excluído (R11)

```bash
curl -s "$API/spaces/$SID/tx?de=2026-09-01&ate=2026-09-30" "${H[@]}"
```
```json
{ "itens": [], "proximo": "" }
```
`excluidoEm` **nunca** sai em `GET /tx`.

### 5. Achar no histórico (`GET /historico`), por mês de alteração e texto

```bash
curl -s "$API/spaces/$SID/historico?mes=2026-09&q=mercado" "${H[@]}"
```
```json
{ "mes": "2026-09", "truncado": false,
  "itens": [
    { "id": "k3x9a1m7q2w8e5r4t6y1", "descricao": "Mercado", "valor": 187.5, "tipo": "saida",
      "dataEvento": "2026-09-10T12:00:00.000Z", "banco": "Nubank", "status": "PAGO",
      "excluido": true, "excluidoEm": "2026-09-14T15:40:05.221Z", "acao": "delete", "versao": 3,
      "em": "2026-09-14T15:40:05.221Z", "porSub": "sub-demo-1", "porNome": "Ana" } ] }
```

### 6. Ver as versões (`GET /tx/{id}/versoes`)

```bash
curl -s "$API/spaces/$SID/tx/$ID/versoes" "${H[@]}"
```
```json
{ "id": "k3x9a1m7q2w8e5r4t6y1", "excluido": true, "semHistorico": false,
  "versoes": [
    { "versao": 3, "acao": "delete", "porSub": "sub-demo-1", "porNome": "Ana", "em": "2026-09-14T15:40:05.221Z", "deVersao": null,
      "diff": [ { "campo": "excluidoEm", "antes": null, "depois": "2026-09-14T15:40:05.221Z" } ],
      "registro": { "id": "k3x9a1m7q2w8e5r4t6y1", "valor": 187.5, "excluidoEm": "2026-09-14T15:40:05.221Z" },
      "restauravel": false, "motivo": "Esta versão é a exclusão. Restaure uma versão anterior." },
    { "versao": 2, "acao": "update", "porSub": "sub-demo-1", "porNome": "Ana", "em": "2026-09-14T13:02:11.482Z", "deVersao": null,
      "diff": [ { "campo": "valor", "antes": 180, "depois": 187.5 } ],
      "registro": { "id": "k3x9a1m7q2w8e5r4t6y1", "valor": 187.5 },
      "restauravel": true, "motivo": "" },
    { "versao": 1, "acao": "insert", "porSub": "sub-demo-1", "porNome": "Ana", "em": "2026-09-10T12:05:00.000Z", "deVersao": null,
      "diff": [], "registro": { "id": "k3x9a1m7q2w8e5r4t6y1", "valor": 180 },
      "restauravel": true, "motivo": "" } ] }
```
(`registro` abreviado aqui; na resposta real vem o lançamento inteiro.)

### 7. Restaurar a versão 2 (`POST .../versoes/2/restaurar`) reativa e grava a versão 4 (`restore`)

```bash
curl -s -X POST "$API/spaces/$SID/tx/$ID/versoes/2/restaurar" "${H[@]}"
```
```json
{ "id": "k3x9a1m7q2w8e5r4t6y1", "dataEvento": "2026-09-10T12:00:00.000Z", "descricao": "Mercado", "valor": 187.5,
  "tipo": "saida", "status": "PAGO", "banco": "Nubank", "tags": ["mercado-mensal"],
  "criadoEm": "2026-09-10T12:05:00.000Z", "criadoPor": "sub-demo-1", "_seq": 104 }
```
Status `200`. O `excluidoEm` **sai** do item vivo, e nasce a versão 4: `acao = restore`, `deVersao = 2`, `porSub` = quem
restaurou, `registro` = o conteúdo da versão 2. As versões 1 a 3 continuam intactas (R13, R14).

### 8. Restaurar bloqueado pelo fechamento (R16)

Se o mês 2026-09 do banco `Nubank` estiver conferido, a mesma chamada responde `409` e **não grava nada**:
```json
{ "erro": "o mes 2026-09 do banco Nubank esta conferido (ate 2026-09) e travado: reabra o mes na aba Saldos para alterar",
  "codigo": "mes_conferido", "banco": "Nubank", "mes": "2026-09", "conferidoAte": "2026-09" }
```
Em `GET /versoes`, a mesma versão vem com `"restauravel": false` e `motivo` com esse texto, e a tela mostra o botão
desabilitado. Outros `409` da restauração: `{"erro": "...", "codigo": "ja_nessa_versao"}` (R15) e
`{"erro": "alterado por outra pessoa", "codigo": "conflito_versao"}` (R3).

## Permissões (`authz.go`, com teste)

| Ação | owner | member |
|---|---|---|
| `historico.ler` | liberado | `soDono` (só o que criou) |
| `historico.restaurar` | liberado | `soDono`, e só se o lançamento **não** está excluído |
| restaurar um excluído | liberado | negado (excluir é só do owner) |

`rotaParaAcao` reconhece as três rotas novas e rejeita métodos errados; o que não está listado é negado.

## Fases e testes

### Fase 0: um caminho de escrita, sem mudar comportamento

Entrega: `Lancamento`, `Mudanca`, repository e service de lançamentos, cada um na sua pasta (`domain/`, `repository/`,
`service/`, `controller/`, mais `auth/`); os controllers de lançamento só chamam o service. Sem versão, sem rota nova,
sem chave nova. `bancoFake` fiel. Estado: **implementada** (PR #100).

| Id | Teste | Esperado |
|---|---|---|
| F0-T1 | Toda a suíte atual (`go test ./...`) | Passa **sem alterar nenhuma expectativa** |
| F0-T2 | `bancoFake.TransactWriteItems` com um item que viola `attribute_not_exists` | Devolve `TransactionCanceledException` e **nenhum** item muda |
| F0-T3 | `bancoFake.Query` com `ScanIndexForward=false` e `Limit=1` | Devolve o último por `SK` |
| F0-T4 | Teste de arquitetura (`arquitetura_test.go`, análise dos imports com `go/parser`) | `domain` não importa o projeto nem a AWS; `service` não importa `controller`, `repository`, AWS nem `net/http`; `controller` não importa `repository` nem o SDK da AWS; `repository` e `auth` não importam as camadas de cima |

**Pronto quando:** F0-T1 a F0-T4 passam e o diff de `*_test.go` existentes só acrescenta testes.

### Fase 1: gravar versões e excluir logicamente

Estado: **implementada e verificada em dev** (PR #102, 2026-10-16). O `GET /tx` e a rotina de avisos tratam `excluidoEm`
como excluído. A simulação do IAM (abaixo) deu `explicitDeny` em `HIST#` e `allowed` em `SPACE#`, e os testes pela tela de
dev passaram.

Entrega: `HIST#`, R2 a R11, `excluidoEm`, `Deny` no `template.yaml`. Capacidade da tabela inalterada.

| Id | Cenário | Esperado |
|---|---|---|
| F1-T1 | Criar lançamento | Item vivo + versão 1 `insert`, `porSub` = quem criou |
| F1-T2 | Editar lançamento **com** versões | Versão `n+1` `update`; `registro` = depois |
| F1-T3 | Editar lançamento legado (sem versões) | v1 `insert` (antes, `porSub` vazio) e v2 `update`, na mesma transação |
| F1-T4 | Ocultar e ajustar fatura | Cada um é uma versão `update` |
| F1-T5 | Mudar a `dataEvento` | **Uma** versão; chave antiga some, nova existe, na mesma transação |
| F1-T6 | Excluir lançamento vivo | `excluidoEm` no item vivo, versão `delete`, aviso removido, `seq` +1 |
| F1-T7 | Excluir de novo o excluído e excluir `id` inexistente | `200 {"ok": true}`, **sem versão nova** |
| F1-T8 | Excluir legado | v1 `insert` (antes) e v2 `delete` |
| F1-T9 | Falha injetada ao gravar a versão | A edição **inteira** aborta: item vivo igual ao de antes, `500` |
| F1-T10 | Duas edições concorrentes do mesmo `id` | Uma grava; a outra `409` `conflito_versao`; só uma versão nova existe |
| F1-T11 | Edição em mês conferido | `409` `mes_conferido` e **nenhum** item gravado (nem versão) |
| F1-T12 | `GET /tx` depois de `DELETE` | O excluído **não** aparece; `excluidoEm` nunca sai em resposta |
| F1-T13 | Importar lote com `id` novo e `id` existente | `insert` e `update`; resposta com `gravados`/`rejeitados`/`pendentes` |
| F1-T14 | Importar com um item em mês conferido | Esse item em `rejeitados`; os demais gravados com versão |
| F1-T15 | Repository: nenhuma expressão de escrita de versão sem `attribute_not_exists(SK)` | O teste falha se alguma faltar |
| F1-T16 | Repository e service | Não existe método de editar ou apagar versão (teste de reflexão sobre a interface) |
| F1-T17 | Nenhuma rota monta `HIST#` a partir do corpo ou da query | Teste varre os controllers |

**Verificação em dev (manual, registrada no PR):**
1. `sam deploy` na stack `financas-dev`.
2. `aws iam simulate-principal-policy` com a role da Lambda, ação `dynamodb:DeleteItem` e a chave de contexto
   `dynamodb:LeadingKeys = HIST#exemplo`: resultado **negado**. Com `SPACE#exemplo`: **permitido**.
3. Repetir para `dynamodb:UpdateItem`. Fazer uma edição com mudança de data (a transação inclui `Delete` em `SPACE#`) e
   confirmar que **continua funcionando**.

`Deny` esperado (condição `ForAnyValue`, para não negar ações sem chave):

```yaml
- Statement:
    - Sid: HistoricoImutavel
      Effect: Deny
      Action: [dynamodb:UpdateItem, dynamodb:DeleteItem]
      Resource: !GetAtt Table.Arn
      Condition:
        ForAnyValue:StringLike:
          dynamodb:LeadingKeys: ['HIST#*']
```

**Pronto quando:** F1-T1 a F1-T17 passam; a verificação em dev está colada no PR; `security.md` e
`specs/modelo-de-dados/` (chave `HIST#`, `excluidoEm`, exemplo fictício) atualizados.

### Fase 2: busca por mês, diff e restauração (API)

| Id | Cenário | Esperado |
|---|---|---|
| F2-T1 | `rotaParaAcao` | Reconhece as 3 rotas; método errado ou caminho extra: `ok=false` |
| F2-T2 | `permitido` (tabela de permissões) | Exatamente a matriz acima; papel ou ação desconhecidos: `negado` |
| F2-T3 | `GET /historico` sem `mes` ou `mes=2026-13` | `400` |
| F2-T4 | Versões em setembro e outubro | `mes=2026-09` só devolve as de setembro |
| F2-T5 | Alteração às 22h30 de 30/09 (SP), `em` já é 01/10 em UTC | Entra em **setembro** (R12) |
| F2-T6 | Um lançamento com 3 versões no mês | Uma linha, com a última |
| F2-T7 | `q` casa o nome **antigo** de um item renomeado no mês | O item aparece |
| F2-T8 | Mais de 5.000 versões | Para a leitura e devolve `truncado: true` |
| F2-T9 | Member em `/historico` e `/versoes` | Só o que criou; o de outro: vazio e `403` |
| F2-T10 | `GET /versoes` de legado nunca editado | `semHistorico: true`, `atual` preenchido |
| F2-T11 | `GET /versoes` de `id` inexistente | `404` |
| F2-T12 | `diff` | Só campos que mudaram; v1 com `[]`; sem `PK`/`SK` |
| F2-T13 | Restaurar versão antiga de lançamento vivo | Versão `restore` com `deVersao`; `criadoPor`/`criadoEm` da versão (R13) |
| F2-T14 | Restaurar excluído (owner) | `excluidoEm` removido, mesmo `id`; reaparece em `GET /tx` |
| F2-T15 | Restaurar excluído com `dataEvento` diferente | Chave antiga some, nova existe (R14) |
| F2-T16 | Restaurar o que já está igual à versão | `409` `ja_nessa_versao`, nada gravado |
| F2-T17 | Restaurar com a versão em mês conferido | `409` `mes_conferido`; em `/versoes` a versão vem com `restauravel: false` e o `motivo` |
| F2-T18 | Restaurar com o **estado atual** em mês conferido | Igual ao anterior (R16) |
| F2-T19 | Member restaura o que criou e que está vivo | Permitido |
| F2-T20 | Member restaura excluído ou de outro | `403` |
| F2-T21 | Restaurar dispara `invalidarSaldos`, tags, aviso e `seq` | Como uma edição normal |
| F2-T23 | Restaurar a versão `delete` | `409` `versao_de_exclusao`, e em `/versoes` ela vem com `restauravel: false` e o motivo (R15b) |
| F2-T22 | Restaurar lançamento de cartão em fatura conferida | **Não** bloqueia (R17); teste documenta a lacuna do item 33 |

**Pronto quando:** F2-T1 a F2-T22 passam; `security.md` com `historico.ler` e `historico.restaurar`.

### Fase 3: tela

Sem código antes da **demo aprovada** (guardrails, "Telas"). A demo é um artefato privado que abre no celular.

- **A demo mostra:** Configurações → Histórico de versões; seletor de mês (padrão: o atual) e busca de texto;
  excluídos e alterados na mesma lista; versões do lançamento na própria tela, com diff; Restaurar **desabilitado com o
  motivo**; aviso "refine a busca" (`truncado`); estado vazio; visão de computador. E o **custo**: só aparece o mês
  escolhido, e a restauração esbarra na trava do fechamento.
- **Depois do ok:** implementar com os tokens do app; testar em ≈ 390 px e ≥ 1280 px, nos dois temas, com o pior caso
  (muitas versões, valor de sete dígitos, descrição longa, mês sem alteração, `truncado`).
- O front **não decide** nada: lista, diff, `restauravel`, `motivo` e `truncado` vêm da API. Nenhuma lógica por nome
  de status.

**Pronto quando:** o critério de tela do intent; `padrao-de-telas.md` e a demo atualizados; `BACKLOG.md` em dia.

## Definição de pronto da feature

- Todos os testes F0 a F2 passam em `go test ./...` e a suíte antiga segue verde.
- Nenhuma rota de escrita chega ao banco sem passar pelo service e sem gravar versão.
- Nenhum código nem policy permite editar ou apagar versão (F1-T15, F1-T16 e a simulação de IAM).
- Documentação em dia na mesma entrega: `security.md`, modelo de dados, `padrao-de-telas.md`, demo e `BACKLOG.md`.

## Detalhes que o spec fixou e o dono pode contestar

1. `registro` como **mapa aninhado** (e não campos soltos), para evitar colisão de nomes.
2. O mês de `/historico` usa o **fuso do app** (São Paulo), não UTC.
3. Uma linha por lançamento, com a **última** versão do mês, e `q` olhando todas as versões do mês.
4. Teto de **5.000** versões e `Deny` com `ForAnyValue` (verificado em dev antes de confiar).
5. Excluir o que já está excluído, ou que não existe, **não** gera versão.
