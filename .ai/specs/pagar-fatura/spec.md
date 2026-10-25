# Spec: pagar a fatura do cartão (itens 33b e 35)

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id
(`G-T<n>` para o backend da fase A), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale o plan.

**Estado:** aprovado pelo dono em 2026-10-25. **Fase A implementada** (backend). Desvios: o `pagador` é um meio da lista e diferente do próprio cartão (R4), mas pode ser outro cartão (a decisão de proibir cartão pagando cartão saiu); `GET /futuros` traz as faturas inteiras (com `tipo: "fatura"`) também dentro de `atrasados` e `meses`, não só os três campos do exemplo. Nomes como
no spec do item 33a: o `banco` gravado chama-se **`meio`** nas rotas e respostas novas; o status, na tela, é a **forma de pagamento**.

## Regras

**Dados do cartão** (dentro de `CFG#CARTOES.porBanco[meio]`, opcionais, escritos só pelos comandos)
- **R1.** O cartão guarda `pagador` (nome de um meio da lista) e `diaVencimento` (inteiro de 1 a 31). Cada fatura (`faturas[ym]`) guarda `pagador` e `vencimento`
  (`AAAA-MM-DD`). Sem eles, valem os do cartão; sem os do cartão, não há.
- **R2.** `PUT /spaces/{sid}/cartoes/{meio}` aceita, além de `inicio`, `pagador` (texto ou `null`) e `diaVencimento` (inteiro de 1 a 31 ou `null`). `null` remove o campo.
- **R3.** `PUT /spaces/{sid}/cartoes/{meio}/faturas/{ym}` aceita, além de `fechamento` e `valorBanco`, `pagador` (texto ou `null`) e `vencimento` (data real ou `null`). O
  `vencimento` precisa ser **depois do fechamento** da fatura e no máximo 45 dias depois dele. Fatura de histórico (anterior ao `inicio`) aceita `valorBanco` e nada mais.
- **R4.** O `pagador` precisa ser um meio **da lista** e **diferente do próprio cartão** (`400` caso contrário).
- **R5.** Os comandos de R2 e R3 valem a regra de R25 do item 33a: controle otimista de `CFG#CARTOES`, tentam de novo, gravam só estes campos.

**Vencimento e pagador**
- **R6.** O **vencimento da fatura `ym`** (`domain.VencimentoDe`) é: `faturas[ym].vencimento`, se existir; senão o **primeiro dia depois do fechamento de `ym`** cujo dia
  do mês é `diaVencimento` (limitado ao último dia daquele mês); sem nenhum dos dois, **não há vencimento**.
- **R7.** O **pagador da fatura** (`domain.PagadorDe`) é `faturas[ym].pagador`, senão `pagador` do cartão, senão vazio. É só uma **sugestão**: o pagamento sai do meio que o dono escolher.

**A marca `pagaFatura`** (uma label no lançamento; não faz regra)
- **R8.** Um lançamento pode levar `pagaFatura: { meio, ym }`: "este lançamento pagou a fatura `ym` do cartão `meio`". O campo é **escolhido pelo dono** e vem no corpo de
  `POST /tx`, `PUT /tx/{id}` e de cada item de `POST /tx/batch`; `null` remove. O servidor **só valida**: `meio` é um cartão (chave de `CFG#CARTOES.porBanco`), `ym` é `AAAA-MM`
  válido e o lançamento **não é uma compra de crédito** (status de crédito do catálogo, ou o `banco` do próprio lançamento é um cartão): `400` caso contrário. Qualquer outro
  formato é `400`.
- **R9.** A marca **não faz regra**: não recusa pagamento, não limita quantos lançamentos têm a mesma marca, nem valor, nem meio, e não muda o saldo nem trava nada. O pagamento
  é um lançamento comum, no meio que o dono escolher (inclusive dois meios para a mesma fatura), e editar ou excluir é como em qualquer lançamento. **Único efeito:** `excluirDoTotal`
  verdadeiro (R10).
- **R10.** O `excluirDoTotal` de um lançamento com a marca é **verdadeiro**, calculado pelo servidor (junto das regras de R13 do item 41), para o pagamento não contar de novo como
  gasto (o gasto já contou nas compras de crédito).
- **R11.** A marca sobrevive ao backup: `POST /tx/batch` a mantém quando vem no item (validada como em R8).

**Leitura**
- **R12.** `GET /spaces/{sid}/cartoes/{meio}/faturas/{ym}` passa a devolver também: `vencimento` (ou `null`), `pagador` (ou `null`), `pagamentos` (os lançamentos não excluídos
  marcados com essa fatura: `{ id, dia, meio, valor, status }`, por data), `totalPago` (soma dos valores) e `diferencaPaga` (`totalPago` menos `valorBanco` se informado, senão
  menos `total`; arredondado; **só informação**). A leitura dos marcados vai do início do período da fatura a 120 dias depois do fechamento; marca fora disso não aparece.
- **R13.** `GET /spaces/{sid}/cartoes` passa a devolver, em cada cartão, `pagador` e `diaVencimento` (ou `null`), para a seção do cartão mostrar o padrão.
- **R14.** Os itens de `GET /tx` e de `GET /futuros` levam `pagaFatura` quando o lançamento tem a marca (para o Histórico mostrar a etiqueta).
- **R15.** `GET /spaces/{sid}/futuros` ganha `faturas`: uma lista, dentro da janela de Futuros (vencimento do primeiro dia do mês passado em diante) e só de cartões com
  vencimento (R6), de itens `{ meio, ym, rotulo, estado, valor, total, valorBanco, vencimento, pagador, atrasada, sugestao }` onde:
  - `estado` é `aberta` ou `fechada` (R10 do item 33a);
  - `valor` é, na **fechada**, `valorBanco` se informado, senão o total calculado; na **aberta**, o total calculado até agora;
  - `atrasada` é verdadeiro quando o vencimento é anterior a hoje;
  - `sugestao` é `{ descricao, valor, pagador }`: `"Pagamento da fatura <mmm/aa> <meio>"`, o `valor` acima e o pagador (R7), para o app preencher a janela de pagar;
  - só entram faturas **sem lançamento marcado** (R8, não excluído), com `total` diferente de zero ou `valorBanco` informado.
  Os itens também entram em `meses` (pelo mês do vencimento) e em `atrasados` (vencimento anterior a hoje) com `tipo: "fatura"`, para a tela agrupar do mesmo jeito que os
  previstos. O `resumo` (`pagar`, `atrasados`) **soma as faturas** listadas.
- **R16.** Os previstos continuam como são (R22 a R24 do item 41). As faturas não são previstos gravados e não viram lançamento até o dono pagar.

**Trava da fatura conferida**
- **R17.** Uma fatura é **conferida** quando `faturas[ym].conferida` existe (R11 do item 33a). O **efeito de um lançamento na fatura** (`domain.EfeitoFatura`) é o par (cartão, fatura
  `ym`, valor com sinal) de uma compra que entra na fatura (R5 a R8 do item 33a: crédito, do banco do cartão, não excluída, não oculta, não prevista). `domain.MudaFatura(antes, depois)`
  é verdadeiro quando o efeito difere. As rotas `POST /tx`, `PUT /tx/{id}`, `DELETE /tx/{id}`, `POST /tx/batch`, `POST /tx/{id}/versoes/{n}/restaurar` e `POST /tx/{id}/confirmar`
  **recusam** a escrita (`409`, `codigo: "fatura_conferida"`, com `meio` e `ym`) quando `MudaFatura` e **qualquer** das faturas do efeito de antes ou de depois está conferida.
  Editar descrição, nota, tags e a marca `pagaFatura` nunca é travado. **Não há confirmação que a destrave:** só `DELETE .../faturas/{ym}/conferir`.
- **R18.** O lançamento de pagamento (que não é compra de crédito) **não** passa pela trava. Conferir uma fatura não a marca como paga e pagá-la não a confere.

**Permissão**
- **R19.** Ações: `cartao.escrever` (já existe, só o dono) para os comandos de R2 e R3; `cartao.ler` (dono e membro) para as leituras. A marca `pagaFatura` segue a permissão de
  `tx.criar` e `tx.editar` do lançamento. Nenhuma rota nova de escrita: **não há comando "pagar"**.

## Contratos de API

Variáveis como no spec do item 33a: `API`, `SID`, `TOKEN`. Hoje é 25/10/2026; dados fictícios.

### `PUT /spaces/{sid}/cartoes/CARTAO-A`
```json
{ "pagador": "BANCO-A", "diaVencimento": 5 }
```
### `PUT /spaces/{sid}/cartoes/CARTAO-A/faturas/2026-10`
```json
{ "pagador": "BANCO-B", "vencimento": "2026-11-05" }
```
### `POST /spaces/{sid}/tx` (um pagamento marcado)
```json
{ "dataEvento": "2026-10-05T15:00:00.000Z", "status": "PAGO", "tipo": "saida", "valor": 600, "banco": "BANCO-A",
  "descricao": "Pagamento da fatura set/26 CARTAO-A", "pagaFatura": { "meio": "CARTAO-A", "ym": "2026-09" } }
```
O segundo meio, no mesmo mês: outro `POST /tx` com `"banco": "BANCO-B"`, `"valor": 1900` e a mesma marca.

Marca inválida (mês errado, ou lançamento de crédito):
```json
{ "erro": "pagaFatura invalido: o meio precisa ser um cartao, o mes AAAA-MM, e o lancamento nao pode ser compra de credito" }
```

### `GET /spaces/{sid}/cartoes/CARTAO-A/faturas/2026-09` (o trecho novo)
```json
{ "vencimento": "2026-10-05", "pagador": "BANCO-A", "totalPago": 2500, "diferencaPaga": -500,
  "pagamentos": [ { "id": "n4k8x2q9w1e7r5t3y6u0", "dia": "2026-10-05", "meio": "BANCO-A", "valor": 600, "status": "PAGO" },
                  { "id": "q7z1c5v8b2n6m4k9j3h0", "dia": "2026-10-05", "meio": "BANCO-B", "valor": 1900, "status": "PAGO" } ] }
```

### `GET /spaces/{sid}/futuros` (o trecho novo)
```json
{ "resumo": { "pagar": 5310.3, "receber": 7200, "atrasados": { "n": 1, "valor": 2500 } },
  "faturas": [
    { "meio": "CARTAO-A", "ym": "2026-09", "rotulo": "set/26", "estado": "fechada", "valor": 2500, "total": 3000, "valorBanco": 2500,
      "vencimento": "2026-10-05", "pagador": "BANCO-A", "atrasada": true,
      "sugestao": { "descricao": "Pagamento da fatura set/26 CARTAO-A", "valor": 2500, "pagador": "BANCO-A" } },
    { "meio": "CARTAO-A", "ym": "2026-10", "rotulo": "out/26", "estado": "aberta", "valor": 1842.35, "total": 1842.35, "valorBanco": null,
      "vencimento": "2026-11-05", "pagador": "BANCO-A", "atrasada": false,
      "sugestao": { "descricao": "Pagamento da fatura out/26 CARTAO-A", "valor": 1842.35, "pagador": "BANCO-A" } } ],
  "atrasados": [ { "tipo": "fatura", "meio": "CARTAO-A", "ym": "2026-09" } ],
  "meses": [ { "mes": "2026-11", "itens": [ { "tipo": "fatura", "meio": "CARTAO-A", "ym": "2026-10" } ] } ], "truncado": false }
```

### Escrita recusada pela trava
```json
{ "erro": "a fatura set/26 do cartao CARTAO-A esta conferida e travada: desfaça a conferencia na aba Cartao para alterar", "codigo": "fatura_conferida",
  "meio": "CARTAO-A", "ym": "2026-09" }
```

## Testes do backend (fase A)

Relógio fixo em 2026-10-25; dados fictícios; as funções puras são testadas em tabela e o resto por `processar`.

| Id | Cenário | Resultado esperado |
|---|---|---|
| G-T1 | `VencimentoDe`: fatura com `vencimento`; só `diaVencimento` maior e menor que o dia do fechamento; dia 31 em abril; sem nenhum dos dois | o do mês certo, limitado ao mês; vazio sem nenhum |
| G-T2 | `PagadorDe`: da fatura, do cartão, nenhum | a precedência de R7 |
| G-T3 | `EfeitoFatura` e `MudaFatura`: valor, data para outra fatura, `faturaAjuste`, oculto, excluído, previsto, status que deixa de ser crédito, descrição/tags/marca | muda nos que mudam o par (fatura, valor); não muda em descrição, tags e marca |
| G-T4 | `PUT /cartoes/{meio}` com `pagador` válido, fora da lista, o próprio cartão, `diaVencimento` 0 e 32, `null` | `200`; `400`; `400`; `400`; remove o campo |
| G-T5 | `PUT .../faturas/{ym}` com `vencimento` antes do fechamento, 46 dias depois, data inexistente, válido; fatura de histórico com `vencimento` | `400`; `400`; `400`; aceito; `400` |
| G-T6 | **Marca inválida:** `pagaFatura` com meio que não é cartão, mês `2026-13`, campo que não é objeto, em compra de crédito, em lançamento do banco de um cartão; em `POST`, `PUT` e `batch` | `400` em cada um; nada gravado |
| G-T7 | **Marca válida:** `POST /tx` com a marca; o segundo lançamento em outro meio com a mesma marca; mesmo meio duas vezes; valor diferente do total; `PUT` que troca e que remove (`null`) | `201`/`200` sempre; nada é recusado; a marca muda e sai |
| G-T8 | O lançamento com a marca: `excluirDoTotal` verdadeiro, mesmo que o corpo mande `false`; saldo do meio de saída diminui como qualquer saída | derivado pelo servidor; saldo normal |
| G-T9 | `GET .../faturas/{ym}` sem e com pagamentos marcados (dois meios), um excluído, um fora da janela | `pagamentos`, `totalPago`, `diferencaPaga`; o excluído e o de fora não entram |
| G-T10 | `GET /futuros` com fatura fechada sem marca, aberta, com marca, sem vencimento, de cartão sem `diaVencimento`, de mês anterior à janela | só as sem marca e com vencimento, na janela; `valor` da fechada = `valorBanco` ou total; `atrasada`; `sugestao` |
| G-T11 | `GET /futuros`: o `resumo` soma as faturas; excluir o lançamento marcado faz a fatura voltar | somas certas; volta |
| G-T12 | **Trava:** com a fatura conferida, `POST /tx` de crédito no período dela, `PUT` do valor, da data (para outra fatura), do `faturaAjuste`, `DELETE`, restaurar, importar | `409 fatura_conferida` em cada um; nada gravado |
| G-T13 | **Trava:** editar descrição/nota/tags/marca, ocultar uma compra de **outra** fatura, lançar em fatura **não** conferida, lançar o pagamento (não é crédito) | passam |
| G-T14 | **Trava:** depois de `DELETE .../conferir`, as mesmas escritas | passam |
| G-T15 | **Trava:** mover uma compra de uma fatura não conferida **para** uma conferida; e **de** uma conferida para fora | `409` nos dois |
| G-T16 | `authz`: as rotas de leitura e os comandos de R2 e R3 por owner e membro; a marca por quem pode editar o lançamento | como R19 |
| G-T17 | O teste de arquitetura e os de reflexão continuam verdes; o cache de Futuros muda quando alguém grava uma marca | verdes; relê |

Os testes que rejeitam ou travam (G-T4, G-T5, G-T6, G-T12, G-T15) são escritos primeiro e devem **falhar antes da mudança**.

## O que muda no front (fase B; demo antes)

- **Aba Cartão, seção de cada cartão:** "Paga com" (lista de meios) e "Vence dia" (1 a 31). Em cada fatura: "Vencimento" (data) e "Paga com" (só daquela fatura). O painel da
  fatura ganha **"Pagamentos desta fatura"**: a lista dos lançamentos marcados (data, meio, valor), a soma e a diferença para o valor da fatura, só como informação.
  "Desfazer conferência" avisa que a fatura está travada.
- **Futuros:** as faturas aparecem no meio pagador sugerido, na data do vencimento, com o valor, "fechada" ou "aberta" e "atrasada". **"Já paguei"** abre a janela de confirmar
  (valor, data, status e o meio de saída, tudo já sugerido) e cria o lançamento marcado, com aviso e Desfazer.
- **Lançar e edição:** uma linha opcional **"Paga a fatura"** (cartão e mês) para marcar qualquer lançamento.
- **Histórico:** o lançamento marcado mostra a etiqueta "Fatura set/26 · CARTAO-A".
- **Mensagem de fatura conferida:** a escrita recusada mostra a mensagem do servidor com o caminho para desfazer a conferência.
- Sai do front tudo que o servidor passa a mandar pronto; o front não calcula vencimento, valor ou atraso.

## Docs que ficam em dia na mesma entrega

- `.ai/specs/modelo-de-dados/` (`CFG#CARTOES`: `pagador`, `diaVencimento`, `faturas[ym].pagador` e `vencimento`; o atributo `pagaFatura` do lançamento, com exemplo fictício),
  `.ai/security.md` (a trava de fatura conferida e a validação de `pagaFatura`), `.ai/arquitetura.md` (a seção "Os três valores da fatura") e o BACKLOG (itens 33, 35 e 17).
