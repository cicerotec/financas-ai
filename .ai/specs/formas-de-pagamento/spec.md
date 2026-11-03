# Spec: PIX, DEBITO, DINHEIRO e TED

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`F`), testes com id (`F-T<n>`).

**Estado:** aprovado pelo dono em 2026-11-02. **Implementado** (`feature/formas-de-pagamento`): `backend/domain/catalogo.go` e os testes em `backend/formas_pagamento_test.go`.

## Regras

- **F1.** `domain.Catalogo` ganha `PIX`, `DEBITO`, `DINHEIRO` e `TED`, todos `Selecionavel` e `ContaNoSaldo`, na ordem de uso do plan (decisão 3):
  `PIX DEBITO CREDITO DINHEIRO TED PAGO`, seguidos dos não selecionáveis como estão. Nenhum outro campo do catálogo muda.
- **F2.** **Sem tipo no catálogo e sem validação por tipo:** as quatro valem para entrada e saída, como o `PAGO`. `validarStatus` não muda (o nome
  precisa estar no catálogo, com o texto exato, e ser selecionável).
- **F3.** O efeito no saldo de cada uma é o do `PAGO` (`ContaNoSaldo`, sem `ForaDoTotal`); trocar `PAGO` por qualquer uma, ou uma pela outra, **não**
  aciona a trava de data (`AlvosDoSaldo` não muda).
- **F4.** `GET /status` e `GET /config/listas` mandam a lista na ordem do catálogo (R9 do `futuros-previsto`), sem nenhuma rota ou parâmetro novo.
- **F5.** Nada no Dynamo muda: o `status` é o texto do nome do catálogo; não há campo, chave nem item novo. Nenhum lançamento é regravado.
- **F6.** O **ajuste da conferência** continua `PAGO`: `domain.StatusGenerico` (`PAGO`) é o status dos lançamentos que o servidor cria por conta própria, e `domain.StatusDeAjuste()` deixou de ser "o primeiro selecionável que conta no saldo" (que passaria a ser o `PIX`).
- **F7.** A **base do cache de saldos** (`domain.BaseCache`) lista os status que contam no saldo; com os novos, ela muda uma vez e os caches antigos deixam de valer. O servidor os recalcula e regrava na primeira leitura de cada meio (sem perda de dado).
- **F8.** O front não ganha nome de status (`web/sem-status.test.js` continua passando) e nenhuma tela muda de desenho.

## Testes (backend)

| Id | Cenário | Esperado |
|---|---|---|
| F-T1 | `GET /status` | `[PIX DEBITO CREDITO DINHEIRO TED PAGO]` (nessa ordem) |
| F-T2 | o padrão: primeiro da lista de `GET /status` e do Confirmar de um antigo | `PIX` (o `PAGO` segue selecionável, no fim) |
| F-T3 | criar saída com cada um de `PIX`, `DEBITO`, `DINHEIRO`, `TED`, `PAGO` | aceito |
| F-T4 | criar entrada com cada um de `PIX`, `DEBITO`, `DINHEIRO`, `TED`, `PAGO` | aceito (estorno no débito é entrada real) |
| F-T5 | entrada `CREDITO` em um cartão (estorno ou reembolso) | aceita; reduz o total da fatura (R6 do `fatura-servidor`) |
| F-T6 | saldo de um meio com um lançamento de cada: `PAGO`, `PIX`, `DEBITO`, `DINHEIRO` e `TED` | igual ao dos mesmos cinco como `PAGO` |
| F-T7 | `PAGO` de mês conferido para `PIX`, mesmo meio, valor, tipo e data | aceito (a trava não é tocada); a versão guarda quem e quando |
| F-T8 | `PIX` para `TED` numa saída de mês fechado | aceito sem a confirmação extra do item 38 (o efeito não muda) |
| F-T9 | `PIX` para um status que não conta no saldo (`CREDITO`) em mês conferido | recusado pela trava de data |
| F-T10 | editar lançamento antigo `PAGO` sem mudar o status | aceito (como hoje) |
| F-T11 | criar com `RECEBIMENTO` ou `BOLETO` | `400 status invalido` (não estão no catálogo) |
| F-T12 | importar backup com `PAGO` e com `PIX` | as duas linhas entram |
| F-T13 | Prever e Repetir de um `TED` | a cópia sai `TED` |
| F-T14 | `RotuloECor` de cada uma | rótulo igual ao nome e a cor do plan |
| F-T15 | `web/sem-status.test.js` | passa sem nenhum dos novos nomes em `web/` |

## Como os testes ficaram

Os ids F-T1 a F-T15 acima são o roteiro; no código eles estão em `backend/formas_pagamento_test.go` (lista e padrão, entrada e saída, saldo igual ao do `PAGO`, trava, `RECEBIMENTO` e `BOLETO` recusados, importar e Confirmar, rótulo, cor e ajuste) e nos testes do catálogo, da lista de status e da base do cache, que mudaram de `[PAGO CREDITO]` para a lista nova. A entrada de `CREDITO` (estorno no cartão) é criada no teste de entrada e saída; o efeito na fatura segue coberto pelo `fatura-servidor`. **Sem teste próprio:** F-T13 (Prever e Repetir copiam o status do original), porque não há código novo nessa parte, e F-T10 (editar um `PAGO` antigo sem mudar o status), que já tinha cobertura.

## Conferir à mão (não dá para provar por teste)

- O nome `DINHEIRO` no Histórico e em Futuros, no celular (≈ 390 px) e na janela larga, nos dois temas: não passa da própria coluna nem encosta no vizinho.
- Lançar abre com `PIX` selecionado.
