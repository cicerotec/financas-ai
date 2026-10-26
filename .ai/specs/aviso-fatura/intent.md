# Intent: aviso no Telegram para a fatura do cartão (item 46)

Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`) vêm junto, para o dono aprovar antes de qualquer código.

**Estado:** rascunho para aprovação (2026-10-26). Depende dos itens 33b e 35 (pagar a fatura), concluídos.

## A ideia

Hoje só um **lançamento** pode ter aviso no Telegram (o campo `aviso`, ligado no Prever ou no formulário). A fatura do cartão aparece em Futuros, mas é uma linha **calculada pelo
servidor**, não um lançamento, e só tem o botão **Já paguei**: não há como pedir para ser avisado de que ela vai vencer.

O dono quer ligar o aviso **na própria fatura**, com a mesma tela e os mesmos prazos do Prever (dias antes, no dia, insistir depois do vencimento).

## Decisões do dono (2026-10-26)

1. O aviso vale **só para aquela fatura**, como o vencimento: o cartão não tem aviso padrão (o padrão do cartão não vale para nenhuma fatura).
2. O aviso depende do **vencimento da fatura**: sem vencimento próprio não há aviso (a tela manda definir o vencimento).
3. A fatura **paga** (com lançamento marcado) sai de Futuros e o aviso para sozinho; se o pagamento for desfeito, volta.
4. **Desempenho (requisito):** nada de varrer lançamentos todo dia. O aviso usa o mesmo item auxiliar pequeno que os lançamentos já usam, e dias sem aviso não leem nada
   além da Query dos avisos.
5. Dado novo aprovado: `aviso` dentro de `faturas[ym]` do cartão (`CFG#CARTOES`), com o mesmo formato do aviso do lançamento. Nada além disso.

## Extensão: aviso padrão do cartão (item 48, 2026-10-26)

O dono pede um **aviso padrão no cartão** (`porBanco.<Banco>.aviso`, mesmo formato do aviso da fatura). Ele **não vale para nenhuma fatura existente**: é só o modelo que é **copiado** para
`faturas[ym].aviso` **no momento em que a fatura ganha o primeiro vencimento** (o momento em que ela vira registro em Futuros). Daí em diante a fatura tem o aviso dela e o dono muda
pelo botão **Avisar** de Futuros. Mudar o padrão depois **não altera** nenhuma fatura já copiada, e nada grava ao *mostrar* a fatura (nenhuma leitura grava).

## Fora

- Aviso padrão por cartão; avisos por e-mail ou WhatsApp (item 34); mexer em qualquer fatura ou lançamento do passado.
