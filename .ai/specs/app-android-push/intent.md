# Intent: push (FCM) como canal de aviso do app Android (fase 4 do item 54)

Este documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md) e o roteiro verificável, em [`spec.md`](spec.md). É a fase 4 de [`../app-android/`](../app-android/plan.md) e usa o despachante
de avisos que o item 34 deixou pronto ([`../avisos-despachante/`](../avisos-despachante/intent.md)).

**Estado:** rascunho para aprovação (2026-11-13). Nada implementado.

## O problema

Os avisos de contas a pagar só chegam pelo Telegram. O app Android já existe (fases 1 a 3), mas não avisa nada sozinho: a pessoa precisa abrir o Telegram, que pede um chat id e um bot. O
dono quer que o **próprio app** avise, como qualquer app de banco, e que a notificação **não revele o que a pessoa deve** na tela bloqueada.

## O que queremos

1. **Receber o aviso do dia como notificação do app**, às 8h, no mesmo horário e pela mesma rotina de hoje.
2. **A notificação mostra só a contagem** ("2 atrasados · 1 vence hoje"): sem descrição, valor, meio de pagamento nem nome do espaço. Os detalhes ficam dentro do app, atrás da trava.
3. **Ligar e desligar por aparelho e por espaço**, numa chave em Listas ("Receber avisos neste aparelho"), no app, como o Telegram é por espaço. Quem tem dois aparelhos liga nos dois.
4. **O Telegram continua como está.** O push é mais um canal do despachante; quem tem os dois recebe os dois.

## Decisões já tomadas (conversa de 2026-11-13)

- O item 34 vem antes (feito e conferido no `dev`).
- **O token do aparelho fica no campo `pushTokens` do vínculo** `USER#<sub>` / `SPACE#<id>`, junto do `telegramChatId` (opção "A"). Cada espaço é uma conta separada.
- A notificação mostra **só a contagem** e **não mostra o nome do espaço**. O mesmo aparelho em dois espaços recebe **uma** notificação, com a contagem somada.
- **Plugin nosso, em Java**, como o `Cofre` e a `Trava`.
- A revisão independente das premissas foi **dispensada** pelo dono (sem agente); a seção "Premissas" do `plan.md` é só de quem escreveu.

## Limites (o que fica de fora)

- **iOS**, **e-mail**, **WhatsApp**.
- **Avisar a esposa das contas que o dono cria.** Quem recebe continua sendo o criador do lançamento e o dono do espaço (decisão à parte).
- **Idempotência da entrega** (aviso em dobro se o agendamento disparar duas vezes) e **horário por pessoa**: continuam como no item 34.
- **Notificação com ações** (pagar pela notificação), imagens e qualquer detalhe do lançamento.
- **Notificação com o app aberto:** o app aberto já mostra o estado; o aviso só aparece em segundo plano ou com o app fechado.

## Custo

FCM não tem custo. A credencial do Firebase fica no SSM (SecureString padrão, sem custo adicional relevante). Uma Lambda a mais não existe: o envio roda dentro da rotina de hoje. O que o dono faz à mão:
criar o projeto no Firebase e baixar dois arquivos (ver o `plan.md`).

## Pronto quando

- No aparelho do dono, ligar a chave em Listas pede a permissão, registra o aparelho e, rodando a rotina no `dev`, chega **uma notificação** com a contagem certa; tocar nela abre o app (com a trava, se ligada).
- Desligar a chave, sair da conta e desinstalar fazem o aparelho parar de receber (os dois últimos, pela rotina que apaga o token recusado).
- Nenhum token, `google-services.json` ou credencial no git, no log ou no chat.
- `security.md`, `avisos.md`, `arquitetura.md`, `modelo-de-dados/README.md`, `padrao-de-telas.md`, o resumo do `guardrails.md` e o `BACKLOG.md` ficam em dia.

## Para o dono decidir

Nada que impeça começar. O que o dono **faz** (não decide) está no `plan.md`, "O que o dono prepara": criar o projeto no Firebase, registrar os dois apps Android (`dev` e `prod`) e guardar a credencial pelo script.
