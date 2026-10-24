# Spec: saldo, fechamento e conferência calculados e validados no servidor

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id
(`S-T<n>` para o backend da fase A), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale o plan.

**Estado:** rascunho para aprovação (2026-10-24). Nada implementado.

## Regras

**O que conta no saldo**
- **R1.** Um lançamento **conta no saldo** do seu `banco` quando: não está excluído (`excluidoEm`), **não está oculto**, o seu `status`
  tem `afetaSaldoBanco[status] == true` em `CFG#LISTAS` (sem o campo, o padrão é `PAGO` e `CONTAS`) e **não é de evento futuro**
  (`EhStatusFuturo(listas, status)`, que o item 41 troca por `EhPrevisto`). Função pura `domain.ContaNoSaldo(l, listas)`.
- **R1a.** Com o item 41, `ContaNoSaldo` deixa de ler `CFG#LISTAS` e passa a usar a propriedade `contaNoSaldo` (e `previsto`) da entrada do catálogo de
  status do `status` do lançamento (uma entrada por nome); a conta continua a mesma para os dados de hoje. Até lá, vale R1.
- **R2.** O valor com sinal é `+valor` para `entrada` e `-valor` para `saida`. Valor zero e negativo contam como vierem.
- **R3.** O **mês** de um lançamento é o da `dataEvento` em São Paulo (`domain.MesDe`). Lançamento sem `banco` ou com data inválida não
  conta em nenhum banco.
- **R4.** Se o banco tem `dataInicio` em `CFG#SALDOS`, só contam lançamentos com `dataEvento` **a partir de 00:00 de `dataInicio` em
  São Paulo**; os anteriores são "histórico" e não entram. Sem `dataInicio`, entram todos.

**A cascata**
- **R5.** O saldo de um banco é uma cascata mensal: parte de `saldoInicial` (0 se não houver) e, **para cada mês** do início (o mês de
  `dataInicio`, ou do primeiro movimento, ou o mês atual) até o **maior entre o mês atual e o último mês com movimento**,
  `acc = arred(acc + soma do mês)`, com `arred(v) = round(v × 100) / 100`. O `saldo` do banco é o `acc` do último mês.
- **R6.** O **ponto de partida** pode ser o fechamento do último mês **conferido** ou o do último mês em **cache válido** (o maior dos
  dois, mas nunca antes do mês de `dataInicio`): a cascata recomeça no mês seguinte, com esse saldo. O resultado é o mesmo de
  calcular desde o início.
- **R7.** **Cache válido:** `base` do item igual à do servidor, mês anterior ao mês atual, anterior a `invalidoDe` (se houver) e não
  anterior ao mês de `dataInicio`. Só os meses **encerrados** (anteriores ao atual) são gravados, e só quando há `dataInicio`.
- **R8.** `base` = `v2|<saldoInicial>|<dataInicio>|<status de afetaSaldoBanco ligados, ordenados, separados por vírgula>/<status de
  statusFuturo ligados>`. Um item com outra `base` (inclusive as de antes do `v2`) não vale e é regravado.
- **R9.** O **estado** de cada mês devolvido: `conferido` (há `conferidos[mes]`), `travado` (é anterior ao último conferido, mas não
  conferido), `aberto` (o mês atual) ou `calculado` (os outros). São devolvidos os **12 últimos** meses, do mais novo para o mais antigo,
  sem os posteriores ao mês atual.

**Total e consulta**
- **R10.** O `total` é a soma dos `saldo` dos bancos **sem** `foraDoTotal` em `CFG#SALDOS`; devolve também `meios` (quantos entraram) e
  `fora` (quantos ficaram de fora).
- **R11.** `GET /spaces/{sid}/saldos` devolve um item para cada meio de `CFG#LISTAS` (`meio`, o `banco` gravado; R0 do item 33a), na ordem da lista, mais `total` e
  `mesAtual` (o do relógio do servidor). Banco sem configuração em `CFG#SALDOS` devolve `semInicio: true`. Não há limite de
  lançamentos e não há o campo `incompleto`.
- **R12.** Depois de calcular, o servidor grava o cache (R7) em `SALDO#<banco>` com controle otimista (`version`): se outra escrita
  mudou o item no meio, **não grava** e devolve o que calculou. Falha de gravação só vai ao log.
- **R13.** O cliente **não** define `saldo`, `caches`, `base` nem `invalidoDe`. `PUT /spaces/{sid}/fechamentos` **não existe** (Fase C;
  antes, o corpo é ignorado). `PUT /cfg/saldos` continua gravando só dados de entrada (`saldoInicial`, `dataInicio`, `foraDoTotal`).

**Conferir**
- **R14.** `POST /spaces/{sid}/fechamentos/conferir` com `{ "meio", "mes", "real", "ajustar"? }` (aceita `banco` no lugar de `meio`). Exige meio, mês `AAAA-MM`
  (o atual ou anterior) e `real` numérico. O servidor calcula o `saldo` do mês (R5) **sem ler o do corpo**.
- **R15.** Se `|saldo − real| < 0,005`: grava `conferidos[mes] = { saldo, real, em, por }` (o `saldo` é o do servidor) e responde `200` com
  o item do banco. A ordem (`ErroOrdemFechamento`), o "só do mais recente" e a trava seguem como hoje.
- **R16.** Se difere e `ajustar` **não** é verdadeiro: `409` com `codigo: "diferenca"`, `calculado`, `real` e `diferenca` (`real − calculado`,
  arredondada). Nada é gravado.
- **R17.** Se difere e `ajustar` é verdadeiro: cria o **lançamento de ajuste** pelo caminho de escrita de lançamentos (`Lancamentos.Criar`):
  `tipo` entrada se `real > calculado`, senão saída; `valor = |diferença|` arredondado; `dataEvento` = último dia do mês, 12:00 em
  São Paulo; `banco`; `descricao: "Ajuste de conferência AAAA-MM"`; `nota: "Criado ao conferir o mês com o extrato do banco."`; `status`
  = o primeiro status do catálogo que conta no saldo (`PAGO`); `criadoPor` = quem conferiu; `excluirDoTotal` calculado pelo servidor. O
  pedido de ajustar vale como a **confirmação** do item 38 para esse lançamento (a versão grava `confirmouMesFechado` quando era
  necessária); mês **conferido** anterior recusa (`409 mes_conferido`). Depois recalcula e confere como em R15.
- **R18.** Se o ajuste foi criado e o conferir falhar em seguida, a resposta diz que o ajuste ficou gravado; repetir a conferência
  (com `ajustar` ou sem) não cria outro, porque a diferença passa a ser zero.
- **R19.** `reabrir` continua como está. `conferir` e `reabrir` são só do dono.

**Permissão**
- **R20.** `GET /saldos` usa a ação `fechamento.ler` (owner e member). `conferir` e `reabrir` seguem `fechamento.conferir` e
  `fechamento.reabrir`. A ação `fechamento.escrever` e a rota `PUT /fechamentos` são removidas na Fase C.

## Contratos de API

Erros seguem o formato de hoje. Variáveis como no spec do histórico: `API`, `SID`, `TOKEN`. Hoje é 24/10/2026; dados fictícios.

### `GET /spaces/{sid}/saldos`

```bash
curl -s "$API/spaces/$SID/saldos" "${H[@]}"
```
```json
{
  "mesAtual": "2026-10",
  "total": { "valor": 12480.55, "meios": 2, "fora": 1 },
  "meios": [
    { "meio": "BANCO-A", "saldo": 5000.25, "foraDoTotal": false, "semInicio": false,
      "inicio": { "dataInicio": "2026-01-01", "saldoInicial": 1000 },
      "meses": [
        { "mes": "2026-10", "valor": 5000.25, "estado": "aberto" },
        { "mes": "2026-09", "valor": 4800.0, "estado": "calculado" },
        { "mes": "2026-08", "valor": 4500.0, "estado": "conferido",
          "conferido": { "real": 4500.0, "ajuste": 0, "em": "2026-09-03T14:20:00.000Z", "por": "sub-demo-1" } } ] },
    { "meio": "BANCO-B", "saldo": 7480.3, "foraDoTotal": false, "semInicio": false, "inicio": { "dataInicio": "2026-03-01", "saldoInicial": 0 }, "meses": [] },
    { "meio": "CARTAO-A", "saldo": -312.4, "foraDoTotal": true, "semInicio": true, "meses": [] }
  ]
}
```

### `POST /spaces/{sid}/fechamentos/conferir`

Conferir agosto com o extrato, e o servidor calculando o saldo:
```bash
curl -s -X POST "$API/spaces/$SID/fechamentos/conferir" "${H[@]}" -d '{ "meio": "BANCO-A", "mes": "2026-08", "real": 4500 }'
```
Quando o extrato difere do que o servidor calculou (4 510,00 contra 4 500,00), sem `ajustar`:
```json
{ "erro": "o saldo calculado (4500.00) nao bate com o extrato (4510.00): confirme o ajuste de 10.00",
  "codigo": "diferenca", "calculado": 4500.0, "real": 4510.0, "diferenca": 10.0 }
```
Com `"ajustar": true`, o servidor cria o lançamento `Ajuste de conferência 2026-08` de 10,00 (entrada), recalcula e confere.

## Testes do backend (fase A)

Relógio fixo em 2026-10-24; dados fictícios; a função pura é testada em tabela, e o resto por `processar`, como os testes de hoje.

| Id | Cenário | Resultado esperado |
|---|---|---|
| S-T1 | `ContaNoSaldo`: `PAGO` normal; oculto; excluído; status com `afetaSaldoBanco` falso; status de evento futuro; sem `afetaSaldoBanco` na configuração | conta; não; não; não; não; padrão `PAGO` e `CONTAS` contam |
| S-T2 | Cascata de 3 meses com entradas e saídas e `saldoInicial` | cada fechamento e o saldo final batem com a conta feita à mão, com arredondamento a cada mês |
| S-T3 | `dataInicio` no meio do mês; lançamento um segundo antes e um depois (00:00 em São Paulo) | o anterior não conta; o posterior conta |
| S-T4 | Lançamento às 23:30 de 31/10 em São Paulo (02:30 UTC de 01/11) | conta em **outubro** |
| S-T5 | Mês sem movimento no meio | o fechamento repete o do mês anterior |
| S-T6 | Movimento em mês **futuro** | a cascata vai até esse mês; o saldo atual inclui |
| S-T7 | Ponto de partida por mês **conferido** e por **cache válido**; cache com `base` diferente; cache de mês atual; cache anterior a `dataInicio` | resultado igual ao de calcular desde o início; o inválido é ignorado |
| S-T8 | `base` de uma versão antiga (sem `v2`) | descartada; o servidor recalcula e regrava |
| S-T9 | `GET /saldos` com 2 bancos, um `foraDoTotal` e um sem configuração | `total` só do primeiro; `semInicio` no outro; ordem da lista |
| S-T10 | `GET /saldos` com 5 000 lançamentos (várias páginas do `Query`) | todos contam; sem `incompleto` |
| S-T11 | O estado dos 12 meses: conferido, travado, calculado, aberto | `estado` certo em cada um; sem meses depois do atual |
| S-T12 | Cache: o `GET /saldos` grava os meses encerrados; uma escrita concorrente muda a `version` no meio | grava na primeira; na segunda não grava e devolve o calculado |
| S-T13 | Sem `dataInicio` | não grava cache; saldo desde o primeiro movimento |
| S-T14 | `conferir` com `real` igual ao calculado | `200`; `conferidos[mes].saldo` é o do **servidor** |
| S-T15 | `conferir` com `saldo` forjado no corpo (igual ao `real` e diferente do calculado) | **ignora o `saldo`**; `409 diferenca` com o valor do servidor |
| S-T16 | `conferir` com `real` diferente e sem `ajustar` | `409 diferenca` com `calculado`, `real`, `diferenca`; nada gravado |
| S-T17 | `conferir` com `ajustar: true` num mês fechado por data | cria o ajuste (entrada ou saída certa, último dia do mês, 12:00 SP), confere; a versão do ajuste leva `confirmouMesFechado` |
| S-T18 | `conferir` com `ajustar: true` em mês anterior a um conferido | `409 mes_conferido`; nada criado |
| S-T19 | `conferir` fora de ordem; mês futuro; sem banco; `real` não numérico | os erros de hoje (`ordem`, `400`) |
| S-T20 | Repetir a conferência depois de um ajuste criado e de uma falha | não cria segundo ajuste |
| S-T21 | `PUT /fechamentos` com `caches` forjados | Fase A: ignorado e nada gravado; Fase C: `404` |
| S-T22 | `authz`: `GET /saldos` para owner e member; `conferir` e `reabrir` só do dono; rota desconhecida | liberado; liberado e negado; `404` |
| S-T23 | O teste de arquitetura e os de reflexão do histórico | continuam verdes |
| S-T25 | **Volume:** 10 anos de dados fictícios (centenas de milhares de lançamentos no teste de carga, ou o que a tabela de dev comportar), 3 bancos: a primeira chamada sem cache e a seguinte com cache | a primeira termina dentro do tempo da Lambda (o limite medido vai no PR); a segunda lê só os meses abertos; os saldos batem |
| S-T24 | **Equivalência:** o mesmo conjunto de lançamentos e configuração calculado pela função pura e por uma implementação de referência escrita no teste, sem cache e com cache | iguais em todos os meses |

Os testes que rejeitam (S-T15, S-T21) são escritos primeiro e devem **falhar antes da mudança**.

## O que muda no front (Fase B; o visual não muda)

- A aba Saldos chama `GET /saldos` (uma vez, e de novo ao salvar a configuração, conferir ou reabrir), mostra `saldo`, `meses`, `total`
  e `fora`, e guarda só o rascunho dos campos.
- Conferir manda `{ banco, mes, real, ajustar }`; a pergunta "criar o ajuste?" passa a ser a resposta `409 diferenca`.
- Saem `calcularSaldoBanco`, `persistirCaches`, `historicoMeses`, `calcularSaldoCompleto`, `carregarSaldos` (cálculo), `totalSaldos`,
  `cachesValidos`, `pontoDePartida`, `baseCfg`, `assinaturaStatus`, `lerFechamentos` e `criarAjusteConferencia`, e as constantes de
  fuso e de mês que só eles usam.
- Atualizar `padrao-de-telas.md` se algum texto mudar (o aviso "⚠ confira" some).

## Docs que ficam em dia na mesma entrega

- `.ai/security.md` (a linha da brecha sai; a tabela perde `fechamento.escrever`), `.ai/arquitetura.md` (a tabela "Hoje" do banco),
  `.ai/specs/modelo-de-dados/README.md` (`base` com `v2`, `conferidos[mes].saldo` do servidor) e o BACKLOG (item 40; o item 14 já
  está absorvido pelo 33).
