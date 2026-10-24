# Spec: Futuros com o marcador `previsto` e status do backend

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id
(`P-T<n>` para o backend da fase A), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale o
plan.

**Estado:** rascunho para aprovação. Nada implementado. O nome do campo (`previsto`) depende do dono.

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
- **R6.** `PUT /tx/{id}`: se o estado resultante tem data futura, o marcador fica ligado; se o lançamento já era previsto, **continua**
  previsto (só o Confirmar o tira). Um antigo (R1, só pelo status) deixa de ser previsto quando o status editado sai dos três
  nomes.
- **R7.** Criar ou importar com `status` igual a um dos três nomes antigos é recusado (`400`, `status descontinuado`). Editar um
  lançamento que já tem esse status **sem mudá-lo** é aceito.
- **R8.** `POST /tx/batch`: o marcador é calculado como em R5 e **mantém** `previsto: true` que venha no item (o backup restaura
  fiel), mas nunca o desliga.

**Efeito no saldo e nas listas**
- **R9.** Um previsto **não mexe no saldo**. Para as travas (conferido e mês fechado por data), o alvo (banco e mês) de um estado
  previsto não conta: `domain.AlvosDoSaldo(antes, depois)` devolve só os alvos de estados que não são previstos. `previsto` entra
  em `domain.CamposSaldo` (mudar o marcador muda o saldo), então `MudaSaldo` é verdadeiro ao confirmar.
- **R10.** `GET /tx` **não devolve previstos** (R1), em nenhuma página, e o parâmetro `status` continua filtrando o resto.
- **R11.** Toda resposta que devolve um lançamento (`GET /tx`, `GET /futuros`, `POST`, `PUT`, `confirmar`) inclui `previsto: true`
  quando R1 se aplica, **também para os antigos (derivado, não gravado)**; senão, o campo não aparece. O front nunca precisa
  olhar o nome do status.

**Confirmar**
- **R12.** `POST /spaces/{sid}/tx/{id}/confirmar?de=<dataEvento atual>` com `{ "valor", "dataEvento", "status" }`. Exige: `valor`
  numérico **maior que zero**; `dataEvento` ISO cujo dia **não seja futuro** (R2); `status` não vazio e **fora** dos três nomes
  antigos. Qualquer falha é `400` com a mensagem da regra. Se o lançamento não existe ou está excluído, `404`; se não é previsto,
  `409` (`nao_previsto`).
- **R13.** A confirmação tira o marcador (o atributo `previsto` sai do item), grava `valor`, `dataEvento` (a chave do item muda
  se a data mudou, como no `PUT`) e `status`, e grava **uma versão** com `acao: "update"`, na mesma transação. As travas rodam sobre
  o **alvo novo** (R9). O parâmetro `confirmaMesFechado=1` vale como nas outras escritas (item 38) e é gravado na versão quando
  era necessário; mês conferido recusa sempre (`409 mes_conferido`).
- **R14.** A resposta é o lançamento atualizado mais `desfazer: {"versao": N}` (a versão de antes da confirmação) e `_seq`. O Desfazer
  é `POST .../versoes/{N}/restaurar`, que devolve o lançamento previsto como estava (a restauração **mantém** o `previsto` que o registro
  da versão tem, pois quem o gravou foi o servidor).
- **R15.** Quem pode: owner sempre; member só os lançamentos que criou (`criadoPor == sub`), como `tx.editar`.

**Futuros**
- **R16.** `GET /spaces/{sid}/futuros[?ocultos=1]` devolve todos os previstos do espaço (R1), sem excluídos. Os **ocultos** ficam de
  fora, a não ser com `ocultos=1`; ocultos nunca entram no resumo.
- **R17.** O corpo traz `resumo` (`pagar`, `receber`, `atrasados: {n, valor}`, arredondados a 2 casas), `atrasados` (os itens com o
  dia anterior a hoje, mais antigos primeiro) e `meses` (um grupo por mês AAAA-MM, em ordem, com os itens por data). Cada item é o
  lançamento público com `previsto: true` (R11). **Não há saldo por mês.** O mês de um item é o de `MesDe` (São Paulo).
- **R18.** Mais de 1000 previstos: devolve os 1000 primeiros e `truncado: true`.
- **R19.** Permissão: `tx.ler` (owner e member).

**Avisos**
- **R20.** A rotina diária de avisos só avisa um lançamento que `EhPrevisto` e não esteja oculto (no lugar de
  `EhStatusFuturo(listas, status)`). Confirmar, ocultar ou excluir encerra o aviso. O item auxiliar `AVISOS#ATIVOS` não muda.

**Listas**
- **R21.** `GET /cfg/listas` não devolve `statusFuturo` nem os três nomes antigos em `status`.
- **R22.** `PUT /cfg/listas` **ignora** `status`, `statusReal`, `afetaSaldoBanco` e `statusFuturo` do corpo: o servidor mantém os do
  item gravado e grava o resto (bancos, `statusCor`, `estiloCor`, ordens). O item no Dynamo **não** perde os três nomes por isso.
  Sem item gravado, `status` nasce do catálogo padrão no código, sem os três nomes.

## Dados (a aprovar pelo dono)

Um **atributo opcional** no lançamento, sem chave nova, sem índice e sem item novo:

| Atributo | Tipo | Quando existe | Quem escreve |
|---|---|---|---|
| `previsto` | booleano | só quando `true`; sai ao confirmar | só o servidor (R4 a R8) |

As versões do histórico guardam o lançamento inteiro e, portanto, o atributo quando ele existe; `acao` não ganha valor novo. O exemplo
fictício entra em `.ai/specs/modelo-de-dados/dynamodb-single-table/` na Fase A (`space-tx_previsto.json`).

## Contratos de API

Erros seguem o formato de hoje. Variáveis como no spec do histórico: `API`, `SID`, `TOKEN`, `ID`. Hoje é 24/10/2026; dados fictícios.

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

Data futura, valor zero ou status antigo:
```json
{ "erro": "a data da confirmacao nao pode ser depois de hoje" }
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
| P-T18 | `confirmar` com valor 0, data futura, status antigo, corpo sem campo | `400` em cada um; nada gravado |
| P-T19 | `confirmar` em lançamento que não é previsto; excluído; inexistente | `409 nao_previsto`; `404`; `404` |
| P-T20 | `confirmar` previsto de setembro mantendo a data (mês fechado) sem e com `confirmaMesFechado=1` | `409 mes_fechado`; depois grava e a versão leva `confirmouMesFechado` |
| P-T21 | `confirmar` previsto de setembro com a data de hoje | **não** pede confirmação (o alvo de antes é previsto) |
| P-T22 | `confirmar` com data em mês conferido | `409 mes_conferido`, mesmo com `confirmaMesFechado=1` |
| P-T23 | Desfazer: `restaurar` a versão devolvida por `confirmar` | volta previsto, com valor, data e status de antes |
| P-T24 | `MudaSaldo` ao ligar ou tirar `previsto` | verdadeiro; editar descrição de um previsto continua falso |
| P-T25 | Permissão: member confirma o que criou; member tenta o de outra pessoa | `200`; `403` |
| P-T26 | `authz`: ação `tx.confirmar` por papel e a rota `POST /tx/{id}/confirmar`; rota desconhecida | liberado, só dono do registro, `404` |
| P-T27 | Avisos: previsto atrasado, previsto futuro, confirmado, antigo com status `PREVISTO` | avisa; avisa; não avisa; avisa |
| P-T28 | `GET /cfg/listas` com item que tem `statusFuturo` e os três nomes | resposta sem eles |
| P-T29 | `PUT /cfg/listas` com `status` e `statusFuturo` alterados | ignorados; o item gravado mantém os valores e leva o resto |
| P-T30 | Forjar: `POST/PUT/batch` com `previsto`, `excluirDoTotal` e `desfazer` no corpo | `previsto` descartado como em R4; nada derivado é aceito |
| P-T31 | O teste de arquitetura (`arquitetura_test.go`) e o de reflexão do histórico | continuam verdes |

Cada teste que rejeita (P-T5, P-T6, P-T10, P-T18, P-T29, P-T30) é escrito primeiro e deve **falhar antes da mudança**; quebre o código de propósito uma vez
para vê-los falhar.

## O que muda no front (fases B e C; o front só exibe)

- **Fase B:** a aba Futuros lê `GET /futuros` (resumo e Atrasados prontos, sem saldo por mês); a ação **Confirmar**
  (janela de baixo com Valor, Data e Status e o botão verde da tabela), o aviso de 6 s com Desfazer (chama a restauração) e os
  textos "Já paguei" e "Já recebi" da demo aprovada; o Histórico e a conta de saldo passam a receber só o que `GET /tx` devolve. Saem
  `eventosFuturos`, `resumoFuturos`, `gruposFuturos` e os usos de `ehFuturo` nessas telas. Atualizar `padrao-de-telas.md` (marcar
  implementada) e a demo na mesma entrega.
- **Fase C (demo antes do código, uma por tela):** Listas sem edição de status e sem as caixas (continua com bancos, cores,
  aparência e ordens); Lançar sem o "evento futuro" e sem os nomes antigos no combo; Prever e Estender sem "Status das cópias" e
  com duplicados e série lidos de `GET /futuros`; fim de "Incluir eventos futuros" na busca. Saem `statusFuturo`, `ehFuturo`,
  `statusReal`/`afetaSaldoBanco` editáveis e o `novoStatusFuturo`; `assinaturaStatus` (o cache do fechamento) é assunto do item 40.

## Docs que ficam em dia na mesma entrega

- `.ai/specs/modelo-de-dados/` (o atributo `previsto`, com exemplo fictício) e `.ai/security.md` (tabela de ações com `tx.confirmar`
  e a regra de status ignorado em `cfg.escrever`).
- `.ai/avisos.md` e `.ai/eventos-futuros.md` (tirar o aviso de "mudança decidida" e descrever o comportamento novo).
- `.ai/padrao-de-telas.md`, `.ai/decisoes.md`, `.ai/arquitetura.md` e o BACKLOG (itens 40 e 41).
