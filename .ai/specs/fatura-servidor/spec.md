# Spec: faturas do cartão calculadas e validadas no servidor (item 33, fatia a)

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id
(`F-T<n>` para o backend da fase A), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale o plan.

**Estado:** rascunho para aprovação (2026-10-24). Nada implementado.

## Regras

**Fechamento e período**
- **R1.** O **fechamento da fatura `ym`** (`AAAA-MM`) de um cartão é: `faturas[ym].fechamento`, se existir; senão o **dia** do fechamento do
  mês mais recente anterior a `ym` (até 60 meses atrás) que tenha `fechamento` escolhido; senão `diaPadrao` (1 se não houver). O dia é
  limitado ao último dia de `ym` (31 em fevereiro vira 28 ou 29). `domain.FechamentoDe(cartao, ym)`.
- **R2.** O **período** da fatura `ym` vai do dia **seguinte** ao fechamento de `ym−1` até o fechamento de `ym`, inclusive.
  `domain.PeriodoFatura`.
- **R3.** A **fatura base** de uma compra, dado o seu dia (em São Paulo, `DiaBR`), é o mês `ym` desse dia se o dia é **até** o fechamento de
  `ym`, senão `ym+1`. `domain.FaturaBase`.
- **R4.** A **fatura atual** é a base do dia de hoje (relógio do servidor, São Paulo). `domain.FaturaAtual`.

**O que entra na fatura**
- **R5.** Uma compra entra na fatura de um cartão (`domain.ContaNaFatura(l, banco)`) quando: o `banco` é o do cartão; o `status` é de **crédito**
  (`domain.EhCredito`: o nome normalizado, sem acento e em maiúsculas, começa com `CREDITO`); **não** está excluída nem oculta; e **não é de
  evento futuro** (`EhStatusFuturo` agora, `EhPrevisto` com o item 41).
- **R6.** O valor da compra na fatura é `+valor` para `saida` e `−valor` para `entrada` (estorno).
- **R7.** O **ajuste efetivo** é `faturaAjuste` quando é número; senão o padrão do status (`CREDITO EX` → `+1`, `CREDITO IN` → `−1`, senão `0`).
  A **fatura efetiva** de uma compra é `FaturaBase` deslocada pelo ajuste, em meses. `domain.AjusteEfetivo`.
- **R8.** O **total** da fatura `ym` é a soma dos valores (R6) das compras cuja fatura efetiva é `ym`, arredondada a 2 casas.
  `domain.TotalFatura`.
- **R9.** `faturaAjuste`, quando enviado, só pode ser `-1`, `0` ou `+1` (`400` caso contrário), em criar, editar e importar. O servidor o
  ignora (não o grava) num lançamento que não é de crédito.

**Estado da fatura**
- **R10.** Uma fatura está **fechada** quando o dia de hoje (São Paulo) é **posterior** ao seu fechamento. É **histórico** quando o cartão tem
  `inicio` e `ym < inicio`: aparece, mas não se confere e não se edita o fechamento.
- **R11.** `valorBanco` é o valor que a pessoa digitou do app do cartão. A **diferença** é `total − valorBanco` (arredondada), quando há
  `valorBanco`. `conferida` é `{ total, em, por }`, gravado **só pelo servidor** (R16). `mudouDesdeConferencia` é verdadeiro quando existe
  `conferida` e o total de agora difere de `conferida.total`.
- **R12.** `conferidasQueMudaram` de um cartão conta as faturas com `conferida` cujo período começa a partir do início da janela lida e cujo
  total atual difere do conferido.

**Leitura**
- **R13.** `GET /spaces/{sid}/cartoes` devolve, para cada cartão (cada chave de `porBanco`), a fatura atual (`ym`, `fechamento`, `diasAteFechar`,
  `total`) e `conferidasQueMudaram`; o `total` geral (soma das faturas atuais); `proximoFechamentoEmDias` (o menor `diasAteFechar`); e
  `disponiveis`: os bancos **sem** cartão, cada um com `temComprasNoCredito` (há compra de crédito nele), os que têm compras primeiro.
- **R14.** `GET /spaces/{sid}/cartoes/{banco}/faturas/{ym}` (o `ym` é opcional e vale a fatura atual) devolve: `ym`, `rotulo`, `chips` (a fatura
  anterior, esta e a próxima, cada uma com `ym` e `rotulo`) e `anterior`/`proxima` (os `ym` dos botões ‹ ›), `periodo` (`ini`, `fim`), `fechada`,
  `historico`, `inicio`, `total`, `valorBanco`, `diferenca`, `conferida`, `mudouDesdeConferencia`, `incompleto: false`, os `lancamentos` da fatura e
  as `sugestoes`. O **rótulo** é "Anterior" (−1), "Atual" (0), "Próxima" (+1), e `mmm/aa` nos outros.
- **R15.** Cada item de `lancamentos` traz `id`, `descricao`, `dia` (`AAAA-MM-DD`), `valor` (com o sinal de R6), `base`, `fatura` (a efetiva),
  `ajuste`, `origem` (`"foi"` se a compra foi para outra fatura, `"veio"` se veio de outra, ou nulo), `estimado` (`valorEstimado`),
  `podeMexer` (o dono, ou quem criou) e `ocupaOutraFatura` (a fatura para onde foi ou de onde veio). A lista tem as compras de base `ym` ou
  efetivas em `ym`, por dia.
- **R16.** As **sugestões** vêm quando a fatura está fechada, há diferença e não é histórico: até 5 combinações de 1 a 3 compras, entre as
  30 mais perto do corte (margem de 5 dias do início ou do fim do período), cuja soma em centavos é a diferença (em módulo), com o ajuste
  a aplicar em cada uma e o **rótulo** ("Passar para a próxima fatura", "Trazer para a anterior", "Tirar desta fatura" ou "Trazer para esta
  fatura"). É o mesmo algoritmo de `candidatosFatura`, com a mesma ordem. Quando não há sugestão, vem `semSugestao` com a explicação
  (anuidade, juros, IOF, compra não registrada, valor a mais ou duplicado) e `podeRegistrarDiferenca` (a diferença é negativa: falta valor).

**Comandos (só o dono)**
- **R17.** `POST /spaces/{sid}/cartoes` `{ banco, diaPadrao }`: o banco é da lista e ainda não é cartão; `diaPadrao` de 1 a 31. Grava
  `porBanco[banco] = { diaPadrao, faturas: {} }`.
- **R18.** `DELETE /spaces/{sid}/cartoes/{banco}` remove a configuração do cartão. Os lançamentos **não** mudam.
- **R19.** `PUT /spaces/{sid}/cartoes/{banco}` `{ "inicio": "AAAA-MM" | null }`: define ou limpa o início do controle (formato `AAAA-MM`).
- **R20.** `PUT /spaces/{sid}/cartoes/{banco}/faturas/{ym}` `{ "fechamento"?: "AAAA-MM-DD" | null, "valorBanco"?: número | null }`: `fechamento` precisa ser
  uma data real **do mês `ym`**; `valorBanco` não pode ser negativo; `null` remove o campo. Fatura de histórico não aceita `fechamento`.
- **R21.** `POST .../faturas/{ym}/conferir` `{ "aceitarDiferenca"?: bool }`: o servidor calcula o `total`. Sem `valorBanco`, ou com diferença e sem
  `aceitarDiferenca`, devolve `409` com `codigo: "diferenca"`, `total`, `valorBanco` e `diferenca`. Senão grava `conferida = { total, em, por }`
  com o total **dele**. Fatura de histórico não confere (`400`).
- **R22.** `DELETE .../faturas/{ym}/conferir` remove `conferida`. Sem `conferida`, `404`.
- **R23.** `POST .../faturas/{ym}/diferenca`: exige `valorBanco` e `falta = valorBanco − total > 0` (senão `400`); cria o lançamento da diferença
  pelo caminho de escrita de lançamentos (R24) e devolve a fatura de novo.
- **R24.** O lançamento da diferença é: `tipo: saida`, `valor: falta`, `descricao: "Ajuste da fatura <rótulo> (juros, anuidade ou IOF)"`
  (o rótulo é `mmm/aa`), `dataEvento` no dia do fechamento às 12:00 em São Paulo, `banco`, `faturaAjuste: 0`, `valorEstimado: false`, status de
  crédito do catálogo e `excluirDoTotal` calculado pelo servidor.
- **R25.** Cada comando usa o controle otimista de `CFG#CARTOES` (`version`), tenta de novo se outra pessoa gravou no meio e **grava só os
  campos de R17 a R23**. A forma do item não muda. O cliente não define `conferida`, `total` nem `por`.
- **R26.** Editar o `faturaAjuste` e corrigir o valor de uma compra estimada continuam sendo `PUT /spaces/{sid}/tx/{id}` (dados de entrada da pessoa).

**Lista de bancos**
- **R27.** `GET /spaces/{sid}/cfg/listas` devolve, além de `banco` (nomes, como hoje), `bancos: [{ "nome", "cartao": bool }]` na mesma ordem:
  `cartao` é verdadeiro quando o banco é uma chave de `CFG#CARTOES.porBanco`. O servidor junta os dois itens; o front não faz chamada extra.

**Legado (Fase A) e remoção (Fase C)**
- **R28.** Na Fase A, `PUT /cfg/cartoes` continua funcionando, mas o servidor **mantém o `conferida` já gravado** e descarta o do corpo. Na
  Fase C a rota e a ação `cfg.escrever` para `cartoes` deixam de existir.

**Permissão**
- **R29.** Ações novas: `cartao.ler` (owner e member) para as duas leituras, e `cartao.escrever` (só do dono) para todos os comandos. Rota nova
  exige a ação em `auth/authz.go` com teste.

## Contratos de API

Erros seguem o formato de hoje. Variáveis como no spec do histórico: `API`, `SID`, `TOKEN`. Hoje é 24/10/2026; dados fictícios.

### `GET /spaces/{sid}/cfg/listas` (a lista de bancos pronta para o Lançar)

O servidor junta `CFG#LISTAS` e `CFG#CARTOES` e manda a marca de cartão em cada banco; o front não faz chamada extra nem olha status.
```bash
curl -s "$API/spaces/$SID/cfg/listas" "${H[@]}"
```
```json
{ "banco": ["BANCO-A", "BANCO-B", "CARTAO-A"],
  "bancos": [ { "nome": "BANCO-A", "cartao": false }, { "nome": "BANCO-B", "cartao": false }, { "nome": "CARTAO-A", "cartao": true } ],
  "statusCor": { "PAGO": "#639922" }, "version": 7 }
```
No Lançar, escolher `CARTAO-A` mostra a linha "Fatura" (`"cartao": true`); escolher `BANCO-A` não mostra. Com o item 41, a mesma resposta também traz `status`.

### `GET /spaces/{sid}/cartoes`

```json
{ "total": 1842.35, "proximoFechamentoEmDias": 4,
  "cartoes": [ { "banco": "CARTAO-A", "fatura": { "ym": "2026-10", "fechamento": "2026-10-28", "diasAteFechar": 4, "total": 1842.35 },
                 "conferidasQueMudaram": 0 } ],
  "disponiveis": [ { "banco": "BANCO-B", "temComprasNoCredito": false } ] }
```

### `GET /spaces/{sid}/cartoes/CARTAO-A/faturas/2026-10`

```json
{ "banco": "CARTAO-A", "ym": "2026-10", "rotulo": "Atual",
  "chips": [ { "ym": "2026-09", "rotulo": "Anterior" }, { "ym": "2026-10", "rotulo": "Atual" }, { "ym": "2026-11", "rotulo": "Próxima" } ],
  "anterior": "2026-09", "proxima": "2026-11",
  "periodo": { "ini": "2026-09-29", "fim": "2026-10-28" }, "fechada": false, "historico": false, "inicio": "2026-01",
  "total": 1842.35, "valorBanco": 1842.35, "diferenca": 0,
  "conferida": { "total": 1842.35, "em": "2026-10-20T14:00:00.000Z", "por": "sub-demo-1" }, "mudouDesdeConferencia": false,
  "lancamentos": [
    { "id": "m4k8x2q9w1e7r5t3y6u0", "descricao": "Notebook 3/6", "dia": "2026-10-03", "valor": 289.9, "base": "2026-10", "fatura": "2026-10",
      "ajuste": 0, "origem": null, "estimado": false, "podeMexer": true } ],
  "sugestoes": [], "semSugestao": null, "podeRegistrarDiferenca": false }
```

### `POST .../faturas/2026-10/conferir`

Sem `valorBanco` informado:
```json
{ "erro": "a fatura nao tem o valor do app do cartao para comparar: confirme para conferir com o total calculado",
  "codigo": "diferenca", "total": 1842.35, "valorBanco": null, "diferenca": null }
```

## Testes do backend (fase A)

Relógio fixo em 2026-10-24; dados fictícios; as funções puras são testadas em tabela e o resto por `processar`.

| Id | Cenário | Resultado esperado |
|---|---|---|
| F-T1 | `FechamentoDe`: mês com fechamento escolhido; sem escolhido herdando o dia do último que tem; sem nenhum (usa `diaPadrao`); dia 31 em fevereiro, abril e ano bissexto; mais de 60 meses | o dia certo em cada caso, limitado ao mês |
| F-T2 | `PeriodoFatura` e `FaturaBase`: compra no dia do fechamento, no dia seguinte e no primeiro dia do período; virada de ano | base certa nas bordas |
| F-T3 | `FaturaAtual` antes, no dia e depois do fechamento | a fatura certa |
| F-T4 | `AjusteEfetivo`: `faturaAjuste` explícito (`0`, `+1`, `-1`), ausente em `CREDITO EX`, em `CREDITO IN` e em `CREDITO` | explícito vence; `+1`, `-1`, `0` |
| F-T5 | `ContaNaFatura`: crédito normal; oculto; excluído; evento futuro; outro banco; status `PAGO`; `entrada` (estorno) | entra; não; não; não; não; não; entra com sinal negativo |
| F-T6 | `TotalFatura` com compras de três faturas, um estorno e um ajuste `+1` | a soma a centavos de cada fatura; a compra com ajuste conta na vizinha |
| F-T7 | Compra às 23:30 de 31/10 em São Paulo (02:30 UTC de 01/11) | o dia é 31/10 |
| F-T8 | Fatura de histórico (`ym < inicio`); fatura aberta e fechada | `historico`, `fechada` corretos; sem sugestões no histórico |
| F-T9 | `GET /cartoes` com 2 cartões e 1 banco sem cartão com e sem compras de crédito | totais, `diasAteFechar`, `proximoFechamentoEmDias`, `disponiveis` ordenados |
| F-T10 | `GET .../faturas/{ym}` sem `ym`, com `ym` futuro e com `ym` de banco sem cartão | atual; válida; `404` |
| F-T11 | `chips`, `anterior`, `proxima` e rótulos (Anterior, Atual, Próxima, `mmm/aa`) | corretos, incluindo a virada de ano |
| F-T12 | `lancamentos` com compra que foi para a próxima e outra que veio da anterior | `origem` `foi` e `veio`, `fatura` e `base` certas |
| F-T13 | `mudouDesdeConferencia` e `conferidasQueMudaram` depois de editar o valor de uma compra conferida | verdadeiro; contagem certa |
| F-T14 | Sugestões: os vetores do `candidatosFatura` atual (1, 2 e 3 compras; diferença positiva e negativa; margem de 5 dias; limite de 5; ajuste fora de ±1 descartado) | o mesmo resultado, na mesma ordem, com rótulo e ação |
| F-T15 | Sem sugestão, com diferença negativa | `semSugestao` com o texto e `podeRegistrarDiferenca` verdadeiro |
| F-T16 | `POST /cartoes` com banco que já é cartão, que não está na lista, `diaPadrao` 0 e 32 | `400`, nada gravado |
| F-T17 | `PUT .../faturas/{ym}` com `fechamento` de outro mês, data inexistente, `valorBanco` negativo, `null` | `400`, `400`, `400`; `null` remove |
| F-T18 | `conferir` sem `valorBanco`, com diferença, com `aceitarDiferenca`, batendo | `409`; `409`; grava; grava |
| F-T19 | **Forjar:** `conferir` com `total` e `conferida` no corpo; `PUT /cfg/cartoes` com `conferida` forjada | o total gravado é o do servidor; o `conferida` do corpo é descartado |
| F-T20 | `desconferir` sem conferida | `404` |
| F-T21 | `POST .../diferenca` com falta positiva, zero e sem `valorBanco` | cria o lançamento certo (data do fechamento 12:00 SP, `faturaAjuste` 0); `400`; `400` |
| F-T22 | `faturaAjuste` `2`, `"1"`, `null` em `POST /tx`, `PUT` e batch; em lançamento que não é de crédito | `400`; `400`; aceito como ausente; ignorado (não gravado) |
| F-T23 | `GET /cfg/listas` com bancos, alguns em `CFG#CARTOES` | `bancos[].cartao` verdadeiro só nesses, na ordem da lista; `banco` de nomes continua |
| F-T24 | `authz`: as duas leituras para owner e member; os comandos só do dono; rota desconhecida | liberado; liberado e negado; `404` |
| F-T25 | Comandos concorrentes em `CFG#CARTOES` (o controle otimista) | tenta de novo e grava; sem perda |
| F-T26 | O teste de arquitetura e os de reflexão do histórico | continuam verdes |
| F-T27 | **Equivalência:** a mesma base fictícia calculada pelas funções puras e por uma implementação de referência escrita no teste | iguais em cada fatura |

Os testes que rejeitam (F-T16, F-T17, F-T19, F-T22) são escritos primeiro e devem **falhar antes da mudança**.

## O que muda no front (Fase B; o visual não muda)

- A aba Cartão lê `GET /cartoes` (a lista e o total do topo) e `GET .../faturas/{ym}` (o painel), e usa os comandos para cada ação. Os
  botões ‹ › e os chips pedem o `ym` que o servidor mandou, **sem aritmética de mês no front**.
- O Lançar mostra a linha "Fatura" quando `bancos[nome].cartao` é verdadeiro, em vez de olhar o status.
- Saem `fechamentoDe`, `periodoFatura`, `faturaBase`, `faturaAtual`, `registrosCartao`, `totalFatura`, `ajusteEfetivo`, `ajustePadrao`,
  `candidatosFatura`, `conferenciaHtml` (o cálculo), `ehCredito`, `ehCreditoEX`, `ehCreditoIN`, `atualizarConfigCartoes`,
  `lerLancamentosBanco`, `carregarBancoCartao`, `desdeNecessario` e o status `CREDITO` procurado por nome na diferença.
- Atualizar `padrao-de-telas.md` se algum texto mudar.

## Docs que ficam em dia na mesma entrega

- `.ai/specs/modelo-de-dados/README.md` (`CFG#CARTOES`: `conferida` gravado só pelo servidor), `.ai/security.md` (a tabela ganha `cartao.ler` e
  `cartao.escrever`; saem as linhas de `faturaAjuste` sem validação e de `conferida`), `.ai/arquitetura.md` (a tabela "Hoje" do cartão) e o
  BACKLOG (item 33 fatiado; o item 14 segue absorvido).
