# Plan: aviso da fatura do cartão (item 46)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/service/avisos.go`, `repository/avisos.go`, `domain/aviso.go`, `domain/aviso_envio.go`,
`service/cartoes.go`, `service/pagafatura.go`, `web/js/app.js`). O roteiro verificável é o [`spec.md`](spec.md). Demo: `.ai/demos/demo-aviso-fatura.html`.

**Estado:** aprovado pelo dono em 2026-10-26. **Fase A implementada** (`feature/aviso-fatura-fase-a`).

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| Cada lançamento com `aviso` tem um **item auxiliar** em `PK = AVISOS#ATIVOS`, `SK = <sid>#<id>` (`id`, `dataEvento`, `venc`, `dias`, `insistir`) | `service.Avisos.Sincronizar`, `repository/avisos.go` | A fatura usa a **mesma partição**, com uma chave que não colide: `SK = <sid>#fatura#<meio>#<ym>` |
| A rotina diária faz **uma Query** na partição e decide quem avisa só pelas datas (`DecidirAviso`); só então lê o lançamento para confirmar e montar a linha | `Avisos.Rodar` | Dias sem aviso não leem nada; o item da fatura entra no mesmo laço |
| O texto do resumo é `LinhaAviso` + `MontarResumo`, com tipo, descrição, valor e banco | `domain/aviso_envio.go` | A fatura vira uma `LinhaAviso` ("Fatura out/26 · SANTANDER", saída, valor da fatura) |
| `ValidarAviso` valida `{dias, insistir}` (0 a 30 dias, até 6 itens) | `domain/aviso.go` | Reaproveitada para `faturas[ym].aviso` |
| `PUT /cartoes/{meio}/faturas/{ym}` grava `fechamento`, `valorBanco`, `pagador` e `vencimento` com controle otimista em `CFG#CARTOES` | `service.Cartoes.Fechamento` | Aceita também `aviso` (objeto ou `null`) |
| `AFuturos` calcula as faturas (valor, vencimento, estado) e sabe se há pagamento marcado | `service/pagafatura.go` | Quando o item dispara, a rotina reaproveita esse cálculo para o valor atual e para saber se foi paga |

## Desenho

- **Dado novo (um só):** `faturas[ym].aviso = { dias: [..], insistir }` na entrada do cartão em `CFG#CARTOES`. Sem item, chave ou índice novo na tabela principal. O item auxiliar
  em `AVISOS#ATIVOS` é o mesmo mecanismo dos lançamentos.
- **Gravar:** `Cartoes.Fechamento` valida o `aviso` (`ValidarAviso`), exige `vencimento` na fatura (R2 do spec) e, depois de gravar o `CFG#CARTOES`, **sincroniza o item auxiliar**
  (`Avisos.SincronizarFatura`). Mudar o vencimento de uma fatura com aviso atualiza o `venc` do item; limpar o aviso ou o vencimento apaga o item. Falha ao sincronizar vai ao log e
  não derruba o salvamento (como nos lançamentos).
- **Enviar:** em `Avisos.Rodar`, o item de fatura passa por `DecidirAviso(hoje, venc, dias, insistir)` como os outros. Só se decidir avisar, o servidor lê o `CFG#CARTOES`, confere
  que o `aviso` e o `vencimento` ainda existem, calcula a fatura (uma Query das compras da janela, a mesma de `AFuturos`) e confere que **não há pagamento marcado**. Fatura paga
  ou sem valor: nada é enviado e o item fica (volta a avisar se o pagamento for desfeito); com aviso desligado na configuração, o item é removido. Depois de 60 dias de atraso (`MaxAtrasoAviso`) o item sai, como nos lançamentos.
- **Destino:** o chat do dono do espaço (a fatura não tem "criador"); membros não recebem o aviso da fatura.
- **Tela (demo aprovada antes):** botão **Avisar** na linha da fatura em Futuros (e na janelinha de ações do celular), que abre a tela "Avisar no Telegram" já existente (`pvTelaAvisar`,
  interruptor, atalhos de dias, "insistir"). O 🔔 e o resumo ("1 dia antes, no dia e insiste depois") aparecem na linha quando ligado. O servidor manda `aviso` na linha da fatura em
  `GET /futuros`; o front só mostra. Sem `vencimento`, a fatura nem aparece em Futuros (item 35), então não há caso sem vencimento na linha; a tela da aba Cartão (fatura sem vencimento)
  não ganha botão.
- **Desempenho:** gravar = 1 `PutItem`/`DeleteItem` por mudança de aviso (mais a gravação do `CFG#CARTOES` que já existe). Ler = a Query diária que já existe + leituras só nos dias em que
  um aviso dispara. Futuros não lê nada a mais (o `aviso` vem do `CFG#CARTOES` que já é lido).

## Fases

- **A (backend):** campo, validação, sincronização do item auxiliar, rotina diária, `aviso` na linha de `GET /futuros`, testes. Só acrescenta.
- **B (tela):** botão Avisar e o 🔔, reaproveitando a tela "Avisar no Telegram" do Prever parametrizada (título, vencimento, balão da fatura, Salvar no rodapé), com demo aprovada. A reformulação da tela como
  componente reutilizável é o item 47, que depois troca os dois usos pelo mesmo componente.

## Riscos

- Item auxiliar fora de sincronia (vencimento mudado, fatura apagada): coberto por testes (G-T3 a G-T6) e pela conferência na hora de enviar.
- Fatura com aviso sem vencimento: impossível por R2.
