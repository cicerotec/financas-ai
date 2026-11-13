# Intent: avisos por despachante e vários canais (item 34)

Este documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md) e o roteiro verificável, em [`spec.md`](spec.md).

**Estado:** rascunho para aprovação (2026-11-13). Nada implementado.

## O problema

Hoje a rotina das 8h (`service.Avisos.Rodar`) faz três coisas ao mesmo tempo: decide **quem** deve ser avisado de **quê**, **monta o texto** do resumo (com descrição, valor e meio de pagamento) e o entrega ao **Telegram**,
que é o único canal. Isso trava o que vem pela frente:

- **O push do app Android** (item 54, fase 4) precisa de um texto diferente. O dono decidiu em 2026-11-13 que a notificação mostra **só a contagem** ("2 atrasados, 1 vence hoje"), sem descrição nem valor, porque ela
  aparece na tela bloqueada. Com o texto pronto dentro do `service`, não há como formatar de dois jeitos.
- **Uma falha derruba a rotina inteira.** Quando o Telegram não está configurado, a rotina para de vez (`ErrCanalNaoConfigurado` faz `return`). Com dois canais, um canal com problema deixaria o outro sem aviso.
- **O resumo do dia vai numa mensagem só**, e o Telegram recusa mensagens acima de **4096 caracteres**: a partir de uns 40 avisos para a mesma pessoa no mesmo dia, nenhum aviso chega naquele dia
  (limite conhecido, registrado em [`../../avisos.md`](../../avisos.md) e no item 28).

## O que queremos

1. **O `service` manda conteúdo estruturado** (o dia, a marca do ambiente e as linhas de aviso), nunca texto formatado. Cada canal formata do seu jeito (decisão de [`arquitetura.md`](../../arquitetura.md), "Avisos: um despachante, vários canais").
2. **Um despachante** entrega esse conteúdo aos canais que a pessoa tem: o Telegram agora, o push na fase 4 do item 54, e o que vier depois. Canal novo é um adaptador novo em `adapter/notificacao/`, sem mexer no `service`.
3. **Uma falha de canal não derruba os outros** nem a rotina. A rotina continua sem devolver erro por causa de envio.
4. **O resumo do Telegram é dividido** em várias mensagens quando passa do limite, sem cortar uma linha e sem perder nada.
5. **Para quem já recebe, nada muda:** as mesmas pessoas, no mesmo horário, com o mesmo texto quando ele cabe numa mensagem.

## Decisões já tomadas (conversa de 2026-11-13)

- O item 34 **vem inteiro antes** do push (fase 4 do item 54). O plano da fase 4 supunha que o despachante já existisse; ele não existe.
- **"Inteiro" quer dizer:** o despachante, a divisão da mensagem longa do Telegram e o isolamento de falha, com o **Telegram como único canal real**. E-mail e WhatsApp ficam como adaptadores futuros, **sem código agora**
  (cada um pede um serviço externo e um custo próprios). *(Interpretação registrada pelo roteiro; o dono não a contestou ao dizer "pode seguir".)*
- **Quem assina o quê é deduzido de quem tem destino cadastrado** (o chat id do Telegram hoje; o token do aparelho na fase 4): não há campo "canais ligados". Nenhum campo novo no DynamoDB neste item.
- A notificação do push mostra só a contagem (isso é da fase 4; está aqui porque explica por que o conteúdo é estruturado).

## Limites (o que fica de fora)

- **E-mail e WhatsApp.** Adaptadores futuros.
- **Idempotência da entrega** (evitar aviso em dobro se o agendamento disparar duas vezes). Pede um registro de "já enviei hoje", que é dado novo no DynamoDB; fica para discutir à parte (ver "Para o dono decidir").
- **Quem recebe e quando.** Continua como hoje: o criador do lançamento e o dono do espaço (a fatura, só o dono), às 8h, só de previsto não oculto. Mudar isso (por exemplo, avisar a esposa das contas que o dono criou) é outra decisão.
- **Visibilidade por pessoa** (item 27) e **tela nova**: nenhuma. O front não muda.
- **O push em si** (plugin, Firebase, `pushTokens`): fase 4 do item 54.

## Custo

Nenhum custo de infraestrutura: é reorganização de código do backend, sem serviço novo. O risco é de comportamento: a rotina diária é o que avisa de contas a pagar, então os testes de hoje têm de continuar
dizendo a mesma coisa (ver o `plan.md`).

## Pronto quando

- Os testes de aviso de hoje passam com o mesmo significado, o resumo curto sai **idêntico** ao de hoje e o `arquitetura_test.go` continua verde.
- Um resumo acima de 4000 caracteres chega ao Telegram em várias mensagens, cada uma com o cabeçalho e a marca do ambiente, na ordem de sempre (atrasados primeiro).
- Um canal que falha, ou que não está configurado, não impede os outros de entregar (teste com dois canais falsos).
- Conferido no `dev` com um resumo real e com um resumo grande de dados fictícios.
- `arquitetura.md`, `avisos.md`, `security.md` e o `BACKLOG.md` ficam em dia.

## Para o dono decidir

Nada em aberto. Decidido em 2026-11-13: o resumo é um **por destino**, como hoje (o mesmo destino em dois espaços recebe um só, com as linhas dos dois), **sem o nome do espaço** (apareceria na tela bloqueada). A
idempotência da entrega fica fora do item e só volta a ser discutida se o aviso em dobro acontecer de verdade.
