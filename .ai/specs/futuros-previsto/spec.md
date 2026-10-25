# Spec: Futuros com o marcador `previsto` e status do backend

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id
(`P-T<n>` para o backend da fase A), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale o
plan.

**Estado:** aprovado pelo dono em 2026-10-24. **Fase A implementada** (`feature/futuros-previsto-fase-a`). Desvios da fase A em relação a este texto: (1) a trava de **criar** e de **excluir** também passou a seguir o efeito no saldo (um previsto ou um status que não conta, como `CREDITO`, não pede confirmação de mês fechado); (2) `GET /tx` calcula `filtros.status` lendo o período inteiro e só na primeira página (sem `cursor`); (3) `GET /status` e `GET /futuros` usam a ação `tx.ler`; (4) o antigo (só pelo nome) não ganha o marcador gravado ao ser editado; (5) `GET /cfg/listas` também devolve `faturaPadrao` (derivado do catálogo, item 33a); (6) os caches de saldo foram invalidados uma vez (`BaseCache` passou a `v3`). O nome do campo, **`previsto`** (booleano), foi aprovado pelo dono em 2026-10-24. Vem
**depois** dos itens 40 ([`saldo-servidor`](../saldo-servidor/spec.md)) e 33 fatiado ([`fatura-servidor`](../fatura-servidor/spec.md)).

## Regras

**O que é previsto**
- **R1.** Um lançamento é **previsto** (`domain.EhPrevisto`) quando tem `previsto: true` **ou** o seu `status` é `PREVISTO`,
  `TRANSFERINDO` ou `CREDITANDO` (os "antigos", numa constante única `domain.StatusAntigosPrevistos`). A configuração `statusFuturo`
  de `CFG#LISTAS` **deixa de ser lida**.
- **R2.** O dia de um lançamento é o da `dataEvento` no fuso de São Paulo (`domain.DiaBR`). Uma data é **futura** quando o dia é
  **posterior** ao de `agora`; hoje não é futuro (`domain.PrevistoPorData(dataEvento, agora)`, que recebe o instante como
  parâmetro). O instante é o relógio do servidor, nunca um valor do cliente.
- **R3.** **Atrasado** é um previsto cujo dia é **anterior** a hoje (`domain.Atrasado(l, agora)`).

**Quem liga e desliga o marcador**
- **R4.** O corpo de `POST /tx`, `PUT /tx/{id}` e de cada item de `POST /tx/batch` **não pode definir** `previsto`: o campo é
  descartado (entra em `domain.Protegidos`). Exceção (R8): a importação mantém `previsto: true`.
- **R5.** `POST /tx`: grava `previsto: true` quando a data é futura (R2); senão não grava o campo.
- **R6.** `PUT /tx/{id}`: se o estado resultante tem data futura, o marcador fica ligado; se o lançamento já era previsto,
  **continua** previsto (só o Confirmar o tira). Um antigo (R1, só pelo status) deixa de ser previsto quando o status editado sai
  dos três nomes.
- **R7.** Criar com um `status` **não oferecido** (R9: qualquer dos antigos, inclusive `CREDITO IN`, `CREDITO EX` e os três nomes de
  previsto) é recusado (`400`, `status descontinuado`). **Importar** aceita o status que for **válido para a data do evento** (R12), para
  um backup antigo voltar como era. Editar um lançamento que já tem um status antigo **sem mudar** o status ou os campos de saldo é aceito.
- **R8.** `POST /tx/batch`: o marcador é calculado como em R5 e **mantém** `previsto: true` que venha no item (o backup restaura
  fiel), mas nunca o desliga.

**A lista de status e o que deriva dela (o front só exibe)**
- **R9.** O catálogo de status é uma **constante do backend** (`domain.Catalogo`), **uma entrada por nome**: `nome`, `rotulo`, `corPadrao`, as
  propriedades que o servidor usa (`contaNoSaldo`, `foraDoTotal`, `credito`, `ajustePadrao`, `previsto`), `selecionavel` (pode ser escolhido ao
  gravar) e, opcionalmente, `desde` e `ate`, que **só orientam a sugestão** da lista de escolha (R10), nunca a validade. **Nenhum status sai do
  catálogo**: o que deixa de ser usado vira `selecionavel: false` e continua valendo onde já está gravado. Na tela o status se chama **forma de
  pagamento** (um rótulo; o campo continua `status`). Catálogo do corte (os valores são os padrões de hoje):

  | nome | selecionável | conta no saldo | fora do total | crédito | ajuste padrão | previsto |
  |---|---|---|---|---|---|---|
  | `PAGO` | sim | sim | não | não | 0 | não |
  | `CREDITO` | sim | não | não | sim | 0 | não |
  | `CONTAS` | não (é o comando de R14) | sim | sim | não | 0 | não |
  | `CREDITO IN` | não | não | não | sim | `−1` | não |
  | `CREDITO EX` | não | não | não | sim | `+1` | não |
  | `PREVISTO` | não | não | não | não | 0 | sim |
  | `TRANSFERINDO` | não | não | sim | não | 0 | sim |
  | `CREDITANDO` | não | não | não | não | 0 | sim |

- **R9b.** **Não há data de corte global** (decisão do dono, 2026-10-24: cada coisa tem a sua vigência, e a troca do passado é do usuário). A
  passagem dos cinco status antigos acontece no deploy do item 41: nesse dia eles viram `selecionavel: false`, sem data gravada em lugar nenhum.
  O dono confirmou que **não criou status além dos sete padrão**; antes do deploy, uma **leitura** confere, em cada espaço, que `afetaSaldoBanco`,
  `statusReal` e `statusFuturo` de `CFG#LISTAS` são iguais aos da tabela.
- **R10.** `GET /cfg/listas` devolve `status` = as entradas **selecionáveis e sugeridas hoje**, só os nomes, na ordem do catálogo (no corte:
  `["PAGO", "CREDITO"]`; o primeiro é o padrão). `GET /spaces/{sid}/status?data=<ISO>[&todos=1]` devolve a lista sugerida **para a data dada**
  (as selecionáveis cujo `desde` e `ate`, se houver, incluem a data); com `todos=1` devolve **todas as selecionáveis**, para quem vai reclassificar
  o passado. O front usa a lista em todo seletor, pede de novo quando a data do formulário muda, guarda o texto escolhido e o devolve como veio.
  `GET /cfg/listas` **não devolve** `statusFuturo`, `statusReal` nem `afetaSaldoBanco`.
- **R11.** `PUT /cfg/listas` **ignora** `status`, `statusReal`, `afetaSaldoBanco` e `statusFuturo` do corpo: o servidor mantém os do
  item gravado e grava o resto (bancos, `statusCor`, `estiloCor`, ordens). O item no Dynamo **não** perde os três nomes por isso.
- **R12.** **Escolher o status é livre; a data só sugere** (o passado é do usuário: um `PAGO` de 2018 pode virar `PIX` ou `DEBITO` porque descreve
  melhor). Ao criar, editar, importar ou confirmar, o servidor aceita qualquer status **selecionável** do catálogo, **em qualquer data**. Criar ou
  confirmar com um status **não selecionável** é recusado (`400`, `status descontinuado`). Editar um lançamento sem mudar o status, e importar um
  backup, aceitam o status que já está gravado. O que protege o passado são as **travas de data**, que dependem do **efeito no saldo**, não do nome
  (R15), e o **histórico de versões**, que torna toda troca auditável e reversível.
- **R12a.** **Rótulo e cor pelo catálogo.** `GET /tx` e `GET /futuros` devolvem cada lançamento com `rotulo` e `corPadrao` da entrada do catálogo do
  seu `status` (se o texto gravado não está no catálogo, `rotulo` é o texto gravado e a cor, neutra). O `status` gravado nunca muda. O que a
  listagem usa (o que é crédito, o que fica fora do total, o ajuste padrão) é o da entrada. O front exibe `rotulo`, usa a `statusCor` da pessoa (se
  houver) ou `corPadrao`, e devolve o `status` gravado.
- **R12b.** **Filtros do Histórico pelo uso no período.** `GET /tx` devolve `filtros: { "status": [...] }` = os status **distintos que aparecem nos
  lançamentos do período consultado** (`de` a `ate`; sem os dois, todos), considerando os **outros** filtros mas **não** o de status (senão a lista
  encolheria para o que já está escolhido), na ordem do catálogo, com os textos fora do catálogo no fim. Assim `CREDITO IN` aparece em 2019 e um `PAGO`
  de 2018 reclassificado para `PIX` aparece no filtro de 2018. O parâmetro `status` filtra pelo **texto gravado**. O front só exibe a lista e devolve
  a escolha.
- **R12c.** **O Histórico não mostra total até o item 42** (decisão do dono, 2026-10-24). O **total da lista exibida** e a **soma por mês** saem da tela, e `GET /tx` **não**
  devolve `resumo` nem `meses`: o front deixa de somar no Histórico (`somaLista` e as somas por mês). O resumo do filtro inteiro, com o filtro de tags no servidor, volta
  com o item 42.
- **R13.** `excluirDoTotal` **deixa de vir do cliente**: o servidor o calcula do catálogo (e do `transferParId`) em criar, editar e
  importar. O valor do corpo é descartado. (Depois que o front parar de enviá-lo, o corpo que o trouxer passa a ser `400`.)
- **R14.** Transferência entre bancos é **um comando do servidor**: `POST /spaces/{sid}/transferencias` com
  `{ origem, destino, valor, dataEvento? }` (os dois são **meios**, R9 do item 33a) cria as duas pontas (saída na origem, entrada no destino) com o mesmo `transferParId`,
  o status de transferência do catálogo e `excluirDoTotal` derivado, numa transação, com as travas de mês nos dois bancos. Ação
  `tx.criar`. O front deixa de montar as duas pontas e de escrever `CONTAS`.

**Efeito no saldo e nas listas**
- **R15.** Um previsto **não mexe no saldo** (item 40: o saldo do servidor ignora `EhPrevisto`). **A trava depende do efeito no saldo, não do nome
  dos campos:** `domain.EfeitoSaldo(l)` é o par (meio e mês, contribuição com sinal), com contribuição zero quando o lançamento **não conta no
  saldo** (`contaNoSaldo` do catálogo), está **oculto** ou é **previsto**. `MudaSaldo(antes, depois)` é verdadeiro quando o efeito **difere**, e
  `domain.AlvosDoSaldo(antes, depois)` devolve os alvos (meio e mês) de cada efeito **não nulo** que mudou. Consequências: trocar `PAGO` por `PIX` (os dois
  contam) **não** passa pela trava, em nenhum mês; trocar para um status que não conta, sim; confirmar um previsto de mês fechado com a data de hoje
  não pede confirmação (o efeito de antes era nulo); com a data do mês fechado, pede (e mês conferido recusa). Editar descrição, nota e tags nunca
  é travado.
- **R16.** `GET /tx` **não devolve previstos** (R1), em nenhuma página, e o parâmetro `status` continua filtrando o resto.
- **R17.** Toda resposta que devolve um lançamento (`GET /tx`, `GET /futuros`, `POST`, `PUT`, `confirmar`) inclui `previsto: true`
  quando R1 se aplica, **também para os antigos (derivado, não gravado)**; senão, o campo não aparece. O front nunca precisa
  olhar o nome do status.

**Confirmar**
- **R18.** `POST /spaces/{sid}/tx/{id}/confirmar?de=<dataEvento atual>` com `{ "valor", "dataEvento", "status" }`. Exige: `valor`
  numérico **maior que zero**; `dataEvento` ISO cujo dia **não seja futuro** (R2); `status` **do catálogo** (R9). Qualquer falha é `400`
  com a mensagem da regra. Se o lançamento não existe ou está excluído, `404`; se não é previsto, `409` (`nao_previsto`).
- **R19.** A confirmação tira o marcador (o atributo `previsto` sai do item), grava `valor`, `dataEvento` (a chave do item muda
  se a data mudou, como no `PUT`) e `status`, e grava **uma versão** com `acao: "update"`, na mesma transação. As travas rodam sobre
  o **alvo novo** (R15). O parâmetro `confirmaMesFechado=1` vale como nas outras escritas (item 38) e é gravado na versão quando
  era necessário; mês conferido recusa sempre (`409 mes_conferido`).
- **R20.** A resposta é o lançamento atualizado mais `desfazer: {"versao": N}` (a versão de antes da confirmação) e `_seq`. O Desfazer
  é `POST .../versoes/{N}/restaurar`, que devolve o lançamento previsto como estava (a restauração **mantém** o `previsto` que o registro
  da versão tem, pois quem o gravou foi o servidor).
- **R21.** Quem pode: owner sempre; member só os lançamentos que criou (`criadoPor == sub`), como `tx.editar`.

**Futuros**
- **R22.** `GET /spaces/{sid}/futuros[?ocultos=1]` devolve todos os previstos do espaço (R1), sem excluídos. Os **ocultos** ficam de
  fora, a não ser com `ocultos=1`; ocultos nunca entram no resumo.
- **R23.** O corpo traz `resumo` (`pagar`, `receber`, `atrasados: {n, valor}`, arredondados a 2 casas), `atrasados` (os itens com o
  dia anterior a hoje, mais antigos primeiro) e `meses` (um grupo por mês AAAA-MM, em ordem, com os itens por data). Cada item é o
  lançamento público com `previsto: true` (R17). **Não há saldo por mês.** O mês de um item é o de `MesDe` (São Paulo).
- **R24.** Mais de 1000 previstos: devolve os 1000 primeiros e `truncado: true`.
- **R25.** Permissão: `tx.ler` (owner e member).

**Avisos**
- **R26.** A rotina diária de avisos só avisa um lançamento que `EhPrevisto` e não esteja oculto (no lugar de
  `EhStatusFuturo(listas, status)`). Confirmar, ocultar ou excluir encerra o aviso. O item auxiliar `AVISOS#ATIVOS` não muda.

## Dados (aprovados pelo dono em 2026-10-24)

Um **atributo opcional** no lançamento, sem chave nova, sem índice e sem item novo:

| Atributo | Tipo | Quando existe | Quem escreve |
|---|---|---|---|
| `previsto` | booleano | só quando `true`; sai ao confirmar | só o servidor (R4 a R8) |

As versões do histórico guardam o lançamento inteiro e, portanto, o atributo quando ele existe; `acao` não ganha valor novo. O exemplo
fictício entra em `.ai/specs/modelo-de-dados/dynamodb-single-table/` na Fase A (`space-tx_previsto.json`). A lista de status **não é
gravada**: é uma constante do código.

## Contratos de API

Erros seguem o formato de hoje. Variáveis como no spec do histórico: `API`, `SID`, `TOKEN`, `ID`. Hoje é 24/10/2026; dados fictícios.

### `GET /spaces/{sid}/cfg/listas` (o trecho novo)

`meios` vem do item 33a e `status` deste item; a resposta final traz os dois.
```bash
curl -s "$API/spaces/$SID/cfg/listas" "${H[@]}"
```
```json
{ "banco": ["BANCO-A", "SODEXO-REF", "CARTAO-A"],
  "meios": [ { "nome": "BANCO-A", "cartao": false, "ativo": true }, { "nome": "SODEXO-REF", "cartao": false, "ativo": false },
             { "nome": "CARTAO-A", "cartao": true, "ativo": true } ],
  "status": ["PAGO", "CREDITO"], "statusCor": { "PAGO": "#639922" }, "version": 7 }
```
O Lançar mostra a linha "Fatura" quando o meio escolhido tem `"cartao": true` (`CARTAO-A`), e não mostra com `BANCO-A`.

### `GET /spaces/{sid}/status?data=2018-03-10T15:00:00.000Z&todos=1`

```json
{ "data": "2018-03-10T15:00:00.000Z", "todos": true, "status": ["PAGO", "CREDITO"] }
```

### `GET /spaces/{sid}/tx?de=2019-03-01&ate=2019-03-31` (os trechos novos)

```json
{ "itens": [ { "id": "c5m1x7q3w9e2r4t6y8u0", "descricao": "Notebook 1/6", "valor": 289.9, "tipo": "saida", "status": "CREDITO IN",
               "rotulo": "CREDITO IN", "corPadrao": "#378ADD", "banco": "CARTAO-A", "dataEvento": "2019-03-10T15:00:00.000Z" } ],
  "filtros": { "status": ["PAGO", "CREDITO IN", "CREDITO EX"] },
  "proximo": "" }
```

### `GET /spaces/{sid}/futuros`

```bash
curl -s "$API/spaces/$SID/futuros" "${H[@]}"
```
```json
{
  "resumo": { "pagar": 3610.3, "receber": 7200, "atrasados": { "n": 2, "valor": 659.9 } },
  "atrasados": [
    { "id": "k3x9a1m7q2w8e5r4t6y1", "descricao": "Condomínio", "valor": 540, "tipo": "saida", "status": "PAGO",
      "banco": "BANCO-A", "dataEvento": "2026-09-10T15:00:00.000Z", "previsto": true }
  ],
  "meses": [
    { "mes": "2026-11", "itens": [
      { "id": "p9w2e5r1t8y3u6i4o7a0", "descricao": "Salário", "valor": 7200, "tipo": "entrada", "status": "PAGO",
        "banco": "BANCO-A", "dataEvento": "2026-11-05T15:00:00.000Z", "previsto": true } ] }
  ],
  "truncado": false
}
```

### `POST /spaces/{sid}/tx/{id}/confirmar`

Confirmar o condomínio de setembro (mês fechado) com a data de hoje: o alvo novo é outubro, aberto, então não pede confirmação.
```bash
curl -s -X POST "$API/spaces/$SID/tx/$ID/confirmar?de=2026-09-10T15:00:00.000Z" "${H[@]}" \
  -d '{ "valor": 548.9, "dataEvento": "2026-10-24T15:00:00.000Z", "status": "PAGO" }'
```
```json
{ "id": "k3x9a1m7q2w8e5r4t6y1", "descricao": "Condomínio", "valor": 548.9, "tipo": "saida", "status": "PAGO",
  "banco": "BANCO-A", "dataEvento": "2026-10-24T15:00:00.000Z",
  "desfazer": { "versao": 1 }, "_seq": 212 }
```

Manter a data de setembro **sem** `confirmaMesFechado=1`:
```json
{ "erro": "o mes 2026-09 do banco BANCO-A ja fechou (so o mes atual e o anterior, ate o dia 7, ficam livres): confirme para alterar",
  "codigo": "mes_fechado", "banco": "BANCO-A", "mes": "2026-09", "confirmavel": true }
```

Data futura, valor zero ou status fora do catálogo:
```json
{ "erro": "a data da confirmacao nao pode ser depois de hoje" }
```

### `POST /spaces/{sid}/transferencias`

```bash
curl -s -X POST "$API/spaces/$SID/transferencias" "${H[@]}" \
  -d '{ "origem": "BANCO-A", "destino": "BANCO-B", "valor": 300, "dataEvento": "2026-10-24T15:00:00.000Z" }'
```
```json
{ "ok": true, "transferParId": "t7m2q9x4w1e8r5y3u6i0", "ids": ["a1...", "b2..."], "_seq": 213 }
```

### Efeito do marcador em `POST /tx`

`POST /tx` com `dataEvento` em 05/11/2026 (corpo sem `previsto`, ou com `previsto: false`, que é descartado) grava e devolve
`"previsto": true`. Com `dataEvento` de hoje, a resposta não traz o campo e o lançamento conta no saldo.

## Testes do backend (fase A)

Relógio fixo em 2026-10-24 (como o `cenario` já faz); dados fictícios.

| Id | Cenário | Resultado esperado |
|---|---|---|
| P-T1 | `EhPrevisto` com `previsto: true`; com cada um dos três status antigos; com `PAGO` sem o campo | verdadeiro, verdadeiro, falso |
| P-T2 | `PrevistoPorData`: hoje às 00:01 e às 23:59 em São Paulo, amanhã às 00:00, ontem | falso, falso, verdadeiro, falso; virada de ano e dia 31/12 → 01/01 |
| P-T3 | `Atrasado`: previsto de ontem, de hoje e de amanhã | verdadeiro, falso, falso |
| P-T4 | `POST /tx` com data futura, sem `previsto` | grava `previsto: true`; resposta traz o campo |
| P-T5 | `POST /tx` com data de hoje e com `previsto: true` no corpo | **não** grava o campo (descartado) |
| P-T6 | `POST /tx` com `status: "PREVISTO"` (e os outros dois) | `400`, nada gravado |
| P-T7 | `PUT` de um antigo trocando só a descrição | aceito; segue previsto (derivado); status intacto |
| P-T8 | `PUT` de um antigo trocando o status para `PAGO` | deixa de ser previsto; sem marcador gravado |
| P-T9 | `PUT` de um previsto trocando a data para ontem | continua previsto (agora atrasado) |
| P-T10 | `PUT` tentando mandar `previsto: false` num previsto | descartado; continua previsto |
| P-T11 | `PUT` de um lançamento real passando a data para o mês seguinte | vira previsto |
| P-T12 | `POST /tx/batch` com item futuro, item antigo com `previsto: true` no passado e item comum | futuro liga; o do passado **mantém**; o comum não |
| P-T13 | `GET /tx` com previstos (marcados e antigos) e comuns | só os comuns; paginação sem lacuna; `status=PAGO` continua filtrando |
| P-T14 | `GET /futuros` com atrasado, hoje, mês futuro, oculto e excluído | grupos certos; oculto e excluído fora; resumo bate com a soma dos itens exibidos |
| P-T15 | `GET /futuros?ocultos=1` | o oculto entra na lista e **não** no resumo |
| P-T16 | Mais de 1000 previstos | 1000 itens e `truncado: true` |
| P-T17 | `confirmar` com valor, data de hoje e status válidos | marcador sai; versão `update`; `desfazer.versao` correto; `_seq` |
| P-T18 | `confirmar` com valor 0, data futura, status fora do catálogo, corpo sem campo | `400` em cada um; nada gravado |
| P-T19 | `confirmar` em lançamento que não é previsto; excluído; inexistente | `409 nao_previsto`; `404`; `404` |
| P-T20 | `confirmar` previsto de setembro mantendo a data (mês fechado) sem e com `confirmaMesFechado=1` | `409 mes_fechado`; depois grava e a versão leva `confirmouMesFechado` |
| P-T21 | `confirmar` previsto de setembro com a data de hoje | **não** pede confirmação (o alvo de antes é previsto) |
| P-T22 | `confirmar` com data em mês conferido | `409 mes_conferido`, mesmo com `confirmaMesFechado=1` |
| P-T23 | Desfazer: `restaurar` a versão devolvida por `confirmar` | volta previsto, com valor, data e status de antes |
| P-T24 | `MudaSaldo` ao ligar ou tirar `previsto` | verdadeiro; editar descrição de um previsto continua falso |
| P-T25 | Permissão: member confirma o que criou; member tenta o de outra pessoa | `200`; `403` |
| P-T26 | `authz`: ação `tx.confirmar` por papel e a rota `POST /tx/{id}/confirmar`; rota desconhecida | liberado, só dono do registro, `404` |
| P-T27 | Avisos: previsto atrasado, previsto futuro, confirmado, antigo com status `PREVISTO` | avisa; avisa; não avisa; avisa |
| P-T28 | `GET /cfg/listas` com item que tem `statusFuturo`, `statusReal`, `afetaSaldoBanco` e os três nomes em `status` | resposta com `status` = `["PAGO", "CREDITO"]` e sem as três caixas |
| P-T29 | `PUT /cfg/listas` com `status` e `statusFuturo` alterados | ignorados; o item gravado mantém os valores e leva o resto |
| P-T30 | Forjar: `POST/PUT/batch` com `previsto`, `excluirDoTotal` e `desfazer` no corpo | `previsto` descartado como em R4; `excluirDoTotal` recalculado pelo servidor; nada derivado é aceito |
| P-T31 | O teste de arquitetura (`arquitetura_test.go`) e o de reflexão do histórico | continuam verdes |
| P-T32 | `POST /tx` com status fora do catálogo (`FOO`); `PUT` de um antigo `CREDITO EX` sem mudar o status | `400 status invalido`; aceito |
| P-T33 | `POST /transferencias` | duas pontas com o mesmo `transferParId`, `excluirDoTotal` derivado, uma transação; trava de mês nos dois bancos |
| P-T34 | `POST /transferencias` com origem igual ao destino, valor zero ou banco vazio | `400`, nada gravado |
| P-T35 | O catálogo: `selecionavel` só em `PAGO` e `CREDITO`; as propriedades de cada um como na tabela de R9 | iguais |
| P-T36 | `POST /tx` com `CREDITO IN` (qualquer data); `POST /tx/batch` com o mesmo item | `400 status descontinuado`; o lote aceita (backup) |
| P-T37 | `PUT` mudando o valor de um `CREDITO IN` de 2019 | aceito; passa pelas travas conforme o efeito (R15) |
| P-T38 | `PUT` trocando o status de um `PAGO` de 2018 para `CREDITO` (muda o efeito) em mês conferido, em mês fechado, antes do início do controle | `409 mes_conferido`; `409 mes_fechado` confirmável; livre |
| P-T39 | `PUT` só da descrição de um lançamento cujo status não é selecionável | aceito, status intacto, sem trava |
| P-T40 | `confirmar` com `CREDITO IN` ou com outro status não selecionável | `400` |
| P-T41 | `GET /tx`: `rotulo` e `corPadrao` de um `CREDITO IN` e de um texto fora do catálogo | os do catálogo; o gravado e cor neutra |
| P-T42 | `GET /tx` com `filtros.status`: período de 2019 (com `CREDITO IN` e `CREDITO EX` gravados), mês recente, com `status=PAGO` ativo, e um texto fora do catálogo | só o que aparece no período; a lista **não** encolhe com o filtro de status; o texto de fora no fim |
| P-T43 | `GET /tx` com 5 000 lançamentos, várias páginas | devolve `itens`, `filtros` e `proximo`; **não** devolve `resumo` nem `meses` |
| P-T44 | `GET /spaces/{sid}/status?data=` para 2019 e para hoje, e com `todos=1`; uma entrada com `desde` no futuro | a sugerida para a data; todas as selecionáveis; a de `desde` futuro some da sugestão e aparece em `todos=1` |
| P-T45 | O catálogo contra os padrões de hoje: `afetaSaldoBanco` (`PAGO`, `CONTAS`), `statusReal` falso e `statusFuturo` verdadeiro nos três previstos, `excluirDoTotal` (`CONTAS`, `TRANSFERINDO`) | iguais |
| P-T46 | **Reclassificar o passado** com um catálogo de teste que tem `PIX` e `DEBITO` contando no saldo: `PAGO` de 2018 para `PIX`, em mês conferido, em mês fechado e antes do início | aceito nos três (o efeito não muda; a trava não é tocada); a versão guarda quem e quando |
| P-T47 | `EfeitoSaldo` e `MudaSaldo`: oculto ligado e desligado; previsto ligado e desligado; troca de valor, tipo, meio e data; troca entre dois status que contam; troca entre um que conta e um que não | verdadeiro nos que mudam o efeito; falso na troca entre dois que contam |

Cada teste que rejeita ou trava (P-T5, P-T6, P-T10, P-T18, P-T29, P-T30, P-T32, P-T34, P-T36, P-T38, P-T40) é escrito primeiro e deve **falhar antes da mudança**;
quebre o código de propósito uma vez para vê-los falhar.

## O que muda no front (fases B e C; o front só exibe)

- **Fase B:** **o total da lista e a soma por mês saem do Histórico** (R12c, pedido do dono; sem demo); a aba Futuros lê `GET /futuros` (resumo e Atrasados prontos, sem saldo por mês); a ação **Confirmar**
  (janela de baixo com Valor, Data e Status e o botão verde da tabela), o aviso de 6 s com Desfazer (chama a restauração) e os
  textos "Já paguei" e "Já recebi" da demo aprovada; o Histórico passa a receber só o que `GET /tx` devolve. O seletor de Status da
  janela usa a lista do servidor: vem com o status atual escolhido e, se ele não está na lista (um antigo), com o **primeiro** da
  lista. Saem `eventosFuturos`, `resumoFuturos`, `gruposFuturos` e os usos de `ehFuturo` nessas telas. Atualizar
  `padrao-de-telas.md` (marcar implementada) e a demo na mesma entrega.
- **Fase C (demo antes do código, uma por tela):** `listas.status` e as caixas de `app.js:5-17` somem (a lista vem de
  `GET /cfg/listas`); Listas só exibe a lista de status e as cores; Lançar sem o "evento futuro" e com o seletor da lista; a
  Transferência chama `POST /transferencias`; Prever e Estender sem "Status das cópias" e com duplicados e série lidos de
  `GET /futuros`; fim de "Incluir eventos futuros" na busca; `consulta-tags.js` sem `ehReal`; `excluirDoTotal` deixa de ser
  enviado. Saem `statusFuturo`, `ehFuturo`, `ehStatusReal`, `afetaSaldo`, `ehCredito` e `COR_PADRAO`. **Teste de CI**
  (`web/sem-status.test.js`, no padrão dos outros `*.test.js`): lê `web/index.html`, `web/js/*.js`, `web/*.js` e falha se achar
  um nome de status (`PAGO`, `CREDITO`, `CONTAS`, `PREVISTO`, `TRANSFERINDO`, `CREDITANDO`) ou um dos predicados acima, com as
  exceções explícitas e comentadas (por exemplo, os textos de ajuda de Listas).

## Docs que ficam em dia na mesma entrega

- `.ai/specs/modelo-de-dados/` (o atributo `previsto`, com exemplo fictício) e `.ai/security.md` (tabela de ações com `tx.confirmar`
  e a regra de status ignorado em `cfg.escrever`).
- `.ai/avisos.md` e `.ai/eventos-futuros.md` (tirar o aviso de "mudança decidida" e descrever o comportamento novo).
- `.ai/padrao-de-telas.md`, `.ai/decisoes.md`, `.ai/arquitetura.md` e o BACKLOG (itens 40, 33 e 41).
