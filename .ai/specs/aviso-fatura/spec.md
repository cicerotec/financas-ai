# Spec: aviso da fatura do cartão (item 46)

Roteiro verificável do [`intent.md`](intent.md) e do [`plan.md`](plan.md).

**Estado:** rascunho para aprovação (2026-10-26). Nada implementado.

## Regras

- **R1.** `faturas[ym].aviso` (`CFG#CARTOES`) é `{ dias: [inteiros 0..30, até 6], insistir: bool }`, o mesmo formato e a mesma validação do aviso do lançamento. `null` remove.
- **R2.** `PUT /spaces/{sid}/cartoes/{meio}/faturas/{ym}` aceita `aviso`. Só o dono (`cartao.escrever`). Recusa (`400`) um `aviso` inválido e **um `aviso` quando a fatura não tem
  `vencimento`** (nem o que está sendo gravado na mesma chamada). Limpar o `vencimento` de uma fatura com aviso remove o aviso junto.
- **R3.** Ao gravar, o servidor mantém o item auxiliar `AVISOS#ATIVOS` / `<sid>#fatura#<meio>#<ym>` (`tipo: "fatura"`, `meio`, `ym`, `venc`, `dias`, `insistir`): grava com aviso,
  atualiza o `venc` quando o vencimento muda, apaga sem aviso. Falha ao sincronizar vai ao log e não derruba a gravação.
- **R4.** A rotina diária decide **só pelas datas** (`DecidirAviso`) e só lê o `CFG#CARTOES` e as compras do cartão quando for avisar. Aviso de fatura **paga** (pagamento marcado),
  sem valor (`valor ≤ 0`), sem `aviso` ou sem `vencimento`: nada é enviado e o item é removido. Mais de 60 dias de atraso: o item sai.
- **R5.** A linha do resumo é "Fatura <rótulo> · <meio>", saída, com o **valor da fatura no dia** (o do app do cartão se informado, senão o total, como em Futuros). Vai ao chat do dono do
  espaço, no mesmo resumo diário dos lançamentos.
- **R6.** `GET /futuros` devolve `aviso` (objeto ou `null`) em cada linha de fatura. Nada mais muda na leitura.
- **R7.** Desempenho: um dia sem nenhum aviso a disparar faz **zero leituras além da Query de `AVISOS#ATIVOS`**; gravar o aviso faz uma escrita no item auxiliar além da do `CFG#CARTOES`.
- **R8.** O passado não é tocado: nenhum lançamento nem fatura existente é alterado; faturas sem `aviso` seguem como estão.

## API

```
PUT /spaces/{sid}/cartoes/{meio}/faturas/{ym}   { "aviso": { "dias": [1, 0], "insistir": true } }   -> 200 a fatura
PUT ...                                           { "aviso": null }                                    -> 200, remove o aviso e o item auxiliar
GET /spaces/{sid}/futuros                        faturas[].aviso = { dias, insistir } | null
```

## Testes

| # | Cenário | Esperado |
|---|---|---|
| G-T1 | `aviso` válido e inválido (dia 31, 7 itens, não objeto), `null` | `200`; `400`; `400`; `400`; remove |
| G-T2 | `aviso` numa fatura sem vencimento; com `vencimento` na mesma chamada | `400`; `200` |
| G-T3 | Gravar o aviso | item auxiliar criado com `venc`, `dias`, `insistir` |
| G-T4 | Mudar o vencimento da fatura com aviso | item auxiliar com o `venc` novo |
| G-T5 | Limpar o aviso; limpar o vencimento | item removido; aviso removido junto |
| G-T6 | Falha ao sincronizar o item | a gravação do `CFG#CARTOES` segue `200` e o erro vai ao log |
| G-T7 | Rotina diária: 1 dia antes, no dia, depois (insistir), 61 dias depois | avisa nos três primeiros; no último remove o item |
| G-T8 | Rotina diária com a fatura paga (lançamento marcado) e com o pagamento desfeito | não avisa e remove o item; volta a avisar se o item for recriado |
| G-T9 | Dia sem aviso a disparar | só a Query de `AVISOS#ATIVOS` (contagem de leituras no fake) |
| G-T10 | Resumo com fatura e lançamento no mesmo chat | uma mensagem, fatura com o valor do dia |
| G-T11 | `GET /futuros` | `aviso` na linha da fatura; `null` sem aviso |
| G-T12 | `authz`: owner grava; member recebe `403` e não recebe o aviso da fatura | como R2 e R5 |

## Telas (fase B, depois da demo aprovada)

- Botão **Avisar** na linha da fatura (desktop) e na janelinha de ações (celular); abre a tela "Avisar no Telegram" já existente; salva com `PUT .../faturas/{ym}`; o 🔔 e o resumo
  vêm do `aviso` da linha. O front não calcula prazo nem decide por status.
- Testes do front: `web/*.test.js` sem chamada a função indefinida e sem nome de status.
