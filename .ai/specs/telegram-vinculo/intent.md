# Intent: vínculo do Telegram por link e aviso só com a contagem

Este documento diz **o quê e por quê**. O `plan.md` e o `spec.md` só nascem depois que o dono aprovar este intent e responder "Para o dono decidir".

**Estado:** rascunho para aprovação (2026-11-20). Nada implementado. É o **item 66** do `BACKLOG.md` e pré-requisito do item 65 (abrir o app ao público). Toca a tela de aviso do **item 47** (componente reutilizável): o `plan.md` diz qual dos dois vem primeiro.

## O problema

1. **Ligar o Telegram é trabalho de técnico.** Hoje a pessoa abre o bot, toca em Iniciar, conversa com o `@userinfobot` para descobrir o chat id e o **digita** no app ([`avisos.md`](../../avisos.md), "Configuração única"). Um usuário novo desiste no meio.
2. **O app não prova que o chat id é de quem o digitou.** Um número errado, ou de outra pessoa, faz o resumo financeiro ir para o chat de um desconhecido. O backend só valida que é texto (`service/perfil.go`).
3. **O Telegram leva o dado financeiro a um terceiro.** O resumo de hoje traz descrição, valor e meio de pagamento ([`security.md`](../../security.md), "Telegram é um terceiro"). O app vai ser aberto ao público ([`AGENTS.md`](../../../AGENTS.md)); o push já decidiu mostrar só a contagem (`app-android-push`), e o Telegram é a exceção que sobrou.

4. **As telas de aviso falam como se o aviso fosse só do Telegram**, mas ele já não é: o lançamento guarda só `aviso: { dias, insistir }` e a rotina entrega a quem tiver destino (Telegram, push e, no futuro, WhatsApp). Em `web/js/app.js` o texto "Avisar no Telegram" está no formulário do Lançar e na janela de Repetir (`:62`), nas telas Prever, Estender e Repetir (`:288, 410, 456, 510`), no aviso da fatura do cartão (`:738, 763, 2952`), no 🔔 de Histórico e Futuros (`:2371`) e no toast da recorrência (`:313`). A dica "Você ainda não vinculou o Telegram" (`:79-80`) usa `telegramVinculado()` (`:57`), que só olha o chat id: **quem tem só o push vê uma dica falsa**. E a prévia **"Assim chega no Telegram"** (`:498-499`) mostra descrição, valor e meio de pagamento, o que deixa de ser verdade com a contagem.

## O que queremos

1. **Ligar por link, sem digitar nada.** Em Listas, Avisos, a pessoa toca em **Ligar Telegram**. O servidor gera um **código de uso único e de vida curta**, amarrado a ela e ao espaço; o app abre `t.me/<bot>?start=<código>`; ela toca em **Iniciar**; o servidor recebe o `/start <código>`, confere, lê o chat id **que o próprio Telegram entrega** e o grava no vínculo. O app passa a mostrar "Ligado" (lendo o estado do servidor; o front não decide).
2. **O chat id nunca é digitado nem exibido.** Vir do Telegram é a prova de que o chat é de quem iniciou a conversa. Só chat **privado** vale: um `/start` vindo de grupo ou canal é recusado.
3. **O aviso do Telegram mostra só a contagem**, como o push: "2 atrasados · 1 vence hoje". Sem descrição, valor, meio de pagamento nem nome do espaço. Os detalhes ficam dentro do app, atrás da trava. A contagem sai de `domain.Contagens` (a mesma do push, AD-3 de [`bmad/arquitetura-avisos-canais.md`](../../bmad/arquitetura-avisos-canais.md)) e o texto é montado em `adapter/notificacao/telegram.go` (AD-1). O que o resumo diário **deixa de mostrar** é perda real para quem lê só pelo Telegram; a demo diz isso.
4. **Desligar sem o app.** O que o Telegram avisa vira destino inválido: se a pessoa **bloqueia o bot** (ou apaga o chat), o envio recusado remove o destino pelo `ErrDestinoInvalido` (AD-4), como o push faz com `UNREGISTERED`; um `/parar` no bot faz o mesmo. A chave "Receber avisos no Telegram" continua apagando o destino pelo app.
5. **O bot responde a quem escreve sem código** com um texto fixo ("Este bot só envia avisos. Para ligar, use o app."), sem dado nenhum.
6. **O campo do Id some da tela para todos**, inclusive para quem já tem um chat id gravado, e o **servidor deixa de aceitar um chat id digitado** (o front esconder o campo não protege nada: a recusa é do backend, [`security.md`](../../security.md)). Só o vínculo por link grava o chat id; apagar (a chave "Receber avisos no Telegram") continua valendo. Quem já tem o Id gravado **continua recebendo, agora só a contagem**, até refazer o vínculo pelo link; **o dono refaz o dele** (um novo cadastro, que sobrescreve o antigo). Se refazer não for possível, ajusta-se por **script rodado pelo dono**, nunca pela IA (nada em produção).

7. **O aviso deixa de ser "do Telegram" em todas as telas.** Os rótulos passam a ser neutros ("Avisar", "Avisos") e a prévia "Assim chega no Telegram" vira uma frase da contagem, sem descrição, valor nem meio de pagamento (o texto vem do servidor; o front não monta aviso). Um canal novo (WhatsApp) entra sem mexer de novo nessas telas.
8. **Ao ligar um aviso sem canal ativo, a pessoa é informada e pode ligar um canal ali mesmo.** A caixa "Avisar" aparece como hoje; quando a pessoa a marca (ou abre o aviso da fatura) e **não há canal ativo neste espaço**, o app diz "Você ainda não tem um canal ativo para receber este aviso" com **Ligar um canal** e **Continuar assim**.
   - **Não bloqueia o salvar:** o aviso é guardado e só sai quando houver canal, como hoje.
   - **A escolha abre numa janela de baixo, por cima da tela atual**, sem levar a pessoa a Listas, para não perder o que ela digitou (o botão voltar já pergunta antes de descartar texto). Os botões quebram linha: **Telegram** (o vínculo por link) e **Neste aparelho** (push, só no app Android); WhatsApp, quando existir, entra como mais um botão.
   - **Quem diz se há canal ativo é o servidor** (o front só mostra): uma rota autenticada devolve, por espaço, se há destino ligado e quais canais existem para ligar. Isso também corrige o erro de quem só tem push.
   - O canal é **por espaço** (o vínculo é por espaço): quem liga o Telegram no espaço A continua sem canal no B.
9. **O 🔔 de Histórico e Futuros diz "sem canal ativo"** quando o lançamento tem aviso e a pessoa não tem canal (**fase 2**, só depois das telas de cima).

## O que isto toca (e o que o dono precisa saber)

- **Rota pública nova.** O Telegram chama o servidor (webhook) sem token do Cognito. É a **segunda exceção** ao "sem token é `401`" (a primeira é `POST /mfa/recuperar`) e [`security.md`](../../security.md) exige decisão do dono, ação própria em `auth/authz.go` e teste de que nenhuma outra rota ficou pública. A barreira é o **`secret_token`** que o Telegram devolve num cabeçalho, comparado em tempo constante, mais o limite de tentativas do estilo do MFA.
- **Dado novo no DynamoDB** (guardrails: discutir antes): o **código de vínculo** (guardado só como hash, com validade e uso único). O `chat id` já tem lugar no vínculo (`telegramChatId`). Nada é criado antes da conversa.
- **O bot do `dev` e o da produção passam a ser dois.** Hoje os dois ambientes usam o mesmo bot ([`security.md`](../../security.md), "Segredos"); um bot só aceita **um** endereço de webhook. Precisa de um segundo bot (BotFather), com token e `secret_token` por ambiente no SSM e um script como `guardar-fcm.ps1` para registrar o webhook sem imprimir segredo. Permissão nova de SSM exige atualizar as roles do Actions **antes** do merge.
- **Tela:** o cartão **Avisos no Telegram** de Listas muda (o campo do Id some ou fica só para quem já tem; entram o botão e os estados "Aguardando você tocar em Iniciar", "Ligado", "Link expirou"). Mudança de tela: **demo aprovada antes do código** (parte de `demo-push-v2-telegram-chave`, a padrão de Avisos). O front só exibe; **não monta texto de aviso nem decide quem recebe** (AD-7).
- **Rotas autenticadas novas** (gerar o código, ler o estado do vínculo e ler "há canal ativo e quais posso ligar"): ação própria, linha no [`api.md`](../../api.md) e seção "API" no `spec.md`. A leitura do estado dos canais não pode devolver token, chat id nem dado de aparelho.
- **Telas a mudar, cada uma com demo** (guardrails): Listas (cartão Avisos no Telegram), Lançar, Repetir/Prever/Estender, o aviso da fatura do cartão e, na fase 2, o 🔔 de Histórico e Futuros. A janela "Ligar um canal" é um componente só, reaproveitado por todas.

## Decisões já tomadas (dono, 2026-11-20)

- O aviso do Telegram **só com a contagem, "como no push do Android"**.
- O vínculo é **por link**, sem digitar chat id (conversa de 2026-11-20).
- **Webhook** (rota pública, com `secret_token`), não a consulta sob demanda.
- **A contagem vale para todos**, inclusive o Telegram do próprio dono.
- **O campo do Id some para todos.** O dono refaz o cadastro pelo link; se não der, ajusta por script (rodado por ele).
- **Um bot por ambiente** (o do `dev` é novo, criado pelo dono no BotFather).
- **Ligar um aviso sem canal ativo informa e oferece ligar um canal numa janela por cima da tela**, sem bloquear o salvar e sem ir a Listas (conversa de 2026-11-20). O 🔔 "sem canal ativo" é a **fase 2**.

## Limites (o que fica de fora)

- **WhatsApp e e-mail.** Outros canais, outras features (o WhatsApp também exigirá provar a posse do número; este desenho serve de modelo).
- **Quem recebe e quando** (o criador do lançamento e o dono do espaço, às 8h) e o **horário por pessoa**: como hoje.
- **Idempotência da entrega** e a **Lambda de 30 s em série**: seguem aceitas do item 34. **Agravante a medir antes de abrir ao público:** o Telegram limita a cerca de 30 mensagens por segundo no total e 1 por segundo por chat; com centenas de usuários o envio das 8h deixa de caber numa Lambda (item 65).
- **Comandos além de `/start` e `/parar`**, botões ou menus no bot, e resposta a texto livre.
- **Várias contas Telegram por pessoa** e vincular grupo ou canal.

## Premissas já conhecidas (entram no `plan.md`, seção Premissas)

- "Há canal ativo" **só vale por espaço** (o destino mora no vínculo). Se o canal mudar de espaço, a dica muda de espaço.
- A janela oferece o push **só no app Android**, e o front sabe disso pelo ambiente (é exibição, não regra de negócio). O servidor não deve devolver "ligue o push" a quem não está no app.
- A dica de hoje ("Você ainda não vinculou o Telegram") existia por causa do campo do Id em Listas; **sai junto com ele**, e o Lançar não pode ficar sem dica nenhuma: quem pergunta é o servidor.
- O aviso guardado sem canal **fica mudo** até haver um canal; a rotina já se comporta assim (sem destino, não entrega).

## Riscos

- **Quem repassa o link:** o código só vale para o espaço de quem o gerou; se alguém tocar em Iniciar num link alheio, o chat dele passa a receber a **contagem** do outro. Dano pequeno (só contagem) e aceito; o código vive poucos minutos e vale uma vez.
- **O webhook é público**, então é alvo de enxurrada. Mitigação: `secret_token` conferido antes de qualquer trabalho, resposta barata e **nenhuma escrita no DynamoDB** sem um código válido (a tabela tem 5 WCU).
- **Perda para quem lê só pelo Telegram:** sem descrição e valor, a mensagem manda abrir o app. É a troca pedida pelo dono; o texto da contagem precisa ser claro.

## Exceções às regras (registradas como pede o `AGENTS.md`)

| Regra | Exceção | Para fechar |
|---|---|---|
| Rota autenticada por padrão | `POST` do webhook é público | Decisão do dono, ação própria, teste de que nenhuma outra rota ficou pública |
| Privacidade por padrão | O Telegram continua um terceiro (agora só vê a contagem) | Nada: a contagem é o padrão decidido |
| Isolamento por pessoa | Chat ids digitados antes da feature não têm prova de posse e seguem recebendo (só a contagem) até serem refeitos | O dono refaz o vínculo pelo link (ou, se não for possível, por script dele); fecha quando não restar chat id sem prova |

## Custo

Telegram e FCM não cobram. O webhook roda na Lambda de hoje (algumas invocações por vínculo). Escritas no DynamoDB: o código (uma), o uso (uma) e o chat id (uma), por pessoa que liga.

## Pronto quando

- No `dev`, a pessoa toca em Ligar Telegram, o bot abre, ela toca em Iniciar, o app mostra "Ligado" e o teste chega com a contagem. Nenhum chat id passou pela tela.
- Um código usado, expirado, de outro espaço, ou um `/start` vindo de grupo **não** liga nada; o webhook sem o `secret_token` certo é recusado e não grava nada.
- Rodando a rotina das 8h no `dev`, o Telegram recebe só a contagem, igual ao push, e **nenhum teste de aviso** acha descrição, valor ou meio de pagamento na mensagem do Telegram.
- Bloquear o bot e rodar a rotina apaga o destino do vínculo.
- No `dev`, marcar "Avisar" no Lançar **sem nenhum canal** mostra o aviso com **Ligar um canal**; ligar o Telegram pela janela volta ao formulário **com o que foi digitado intacto**. Quem tem **só o push** não vê a dica de "sem canal". Nenhuma tela de aviso diz "Telegram" fora do cartão de Listas e da janela de escolha.
- Um `PUT` do perfil com `telegramChatId` digitado é **recusado** pelo servidor (teste que o tenta forjar); só apagar (valor vazio) passa.
- No `dev`, refazer o vínculo de um chat id antigo pelo link **sobrescreve** o valor, o que confirma se o "refazer o cadastro" basta; se não bastar, o `plan.md` traz o script.
- `avisos.md`, `security.md`, `api.md`, `arquitetura.md`, `padrao-de-telas.md`, `modelo-de-dados/README.md`, o resumo do `guardrails.md` e o `BACKLOG.md` ficam em dia, e o `plan.md` traz a seção **Premissas** (guardrails).

## Para o dono decidir

Nada em aberto para começar o `plan.md`. As perguntas foram respondidas em 2026-11-20 (ver "Decisões já tomadas") e o item é o **66** do `BACKLOG.md`. O que o `plan.md` ainda precisa decidir com o dono: a ordem frente ao item 47 e o dado novo do código de vínculo no DynamoDB.
