# Plan: vínculo do Telegram por link, aviso só com a contagem e "ligar um canal" nas telas de aviso

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/adapter/notificacao/telegram.go` e `telegram_texto.go`, `service/avisos.go` e `perfil.go`, `repository/destinos.go`, `perfil.go` e `espacos.go`, `auth/authz.go`, `roteador.go`, `template.yaml`, `web/js/telegram.js`, `web/api.js` e `web/js/app.js`). O roteiro verificável é o [`spec.md`](spec.md); o **contrato das rotas** está na seção **"API"** dele (guardrails).

**Estado:** rascunho para aprovação (2026-11-20). Nada implementado. **Cria dado novo no DynamoDB (um item `TGLINK#<hash>`), ainda não aprovado pelo dono** (decisão D1 abaixo): nada é criado antes da conversa. A **revisão independente das premissas** (guardrails, "Roteiros") **não foi feita**: o dono dispensou-a em 2026-11-13 por não ter agente para isso, e ela é pedida quando existir um; a seção "Premissas" traz só o que **quem escreveu** levantou e **não a substitui**. O que o Telegram exige foi lido só em parte (links em "Fontes") e deve ser **conferido na documentação oficial e no `dev`** na hora de montar.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| O Telegram monta o resumo **inteiro** (descrição, valor, meio de pagamento, data) e o divide em partes de até 4000 letras | `adapter/notificacao/telegram.go` (`Enviar`), `telegram_texto.go` (`MontarResumo`, `PartesDoResumo`, `textoDaLinha`, `Moeda`) | Passa a ser uma linha de contagem. A divisão em partes, o `Moeda` e o `textoDaLinha` saem (nada que o canal mostre os usa). O push já tem a contagem em `fcm.go` (`TextoDoPush`): o texto da contagem vira uma função comum do pacote |
| Uma resposta não-2xx do Telegram vira erro **com os primeiros 300 caracteres do corpo** | `telegram.go` (`EnviarTexto`) | Passa a levar só o código HTTP e o `error_code`, como o FCM (R10 do push); 403 e "chat not found" viram `domain.ErrDestinoInvalido`; 401 do bot vira `ErrCredencialRecusada` |
| O Telegram **nunca** devolve `ErrDestinoInvalido`, embora `RemoverDestino` já saiba apagar o chat id (condição `telegramChatId = :c`) | `repository/destinos.go`, `service/avisos.go` (`Rodar`) | A rotina já apaga o destino recusado e conta o grupo como não entregue (R11 do push); o Telegram só precisa devolver o erro certo |
| O chat id é gravado por `PUT /spaces/{sid}/perfil` (`telegramChatId`, validado por `domain.ChatIDValido`) e **devolvido por `GET /me`** a cada espaço | `service/perfil.go`, `repository/perfil.go`, `controller/espacos.go`, `repository/espacos.go` | O front hoje **vê o chat id**. Passa a ver só `canais: {telegram, push}` (booleanos); o `PUT` deixa de aceitar Id digitado e o `GET /me` deixa de devolvê-lo |
| `GET /me` lê os vínculos da pessoa por **uma** `Query` (`EspacosDe`), que já traz cada item inteiro (`telegramChatId` e `pushTokens`) | `repository/espacos.go` | "Há canal ativo" sai **sem leitura nova**: os dois canais estão no mesmo item. Por isso **não há rota de "estado dos canais"** (a intent previa uma): o estado vai em `GET /me`, que o app já lê na subida |
| Há **uma** rota pública (`POST /mfa/recuperar`), reconhecida por `auth.RotaPublica` **antes** do token, com o rótulo gravado antes de entregar ao controller | `auth/authz.go`, `roteador.go` (`atender`, `publica`, `rotular`) | O webhook é a segunda: `RotaPublica` passa a reconhecer dois pares exatos (método e caminho); `publica` ganha um `case`; nenhuma outra rota muda |
| A Function URL é `AuthType: NONE` e o CORS só vale para navegador | `template.yaml` | A chamada do Telegram não tem CORS nem token do Cognito; **a barreira é o `secret_token`**. Os cabeçalhos chegam em minúsculas |
| O token do bot vem do SSM (`TELEGRAM_TOKEN_PARAM`, política `SSMParameterReadPolicy`); `dev` e `prod` usam o **mesmo bot** (`security.md`) | `template.yaml`, `adapter/notificacao/telegram.go` | Um bot só aceita **um** webhook: o `dev` precisa de bot próprio **antes** de qualquer `setWebhook`, senão o `dev` sequestra o webhook da produção |
| O aviso de segurança do MFA vai pelo Telegram como `ResumoAvisos.Mensagem` (texto fixo) | `domain/aviso_resumo.go`, `telegram_texto.go` (`PartesDoResumo`) | Continua igual; o teste do MFA (`mfa_aviso_test.go`) tem de seguir verde |
| O cartão "Avisos no Telegram" guarda o Id (`PUT /perfil`), lembra o último Id **em `localStorage`** (`financas.telegram`) e a dica do Lançar usa `telegramChatId()` | `web/js/telegram.js`, `web/api.js` (`definirTelegram`), `web/js/app.js:57-80` | O campo, o "lembrado" e a dica somem; um **chat id antigo fica em `localStorage`** nos aparelhos: o front o apaga na subida |
| Textos "Avisar no Telegram" e a prévia "Assim chega no Telegram" (com descrição, valor e banco) estão em várias telas | `web/js/app.js:62, 288, 410, 456, 498-499, 510, 738, 763, 2371, 2952, 313` | Rótulos neutros; a prévia deixa de mostrar dados (vira uma frase fixa) |
| O teste do arquitetura passa a barrar `telegram`, `fcm`, `firebase` e `push` em `domain/` e `service/`, com exceções nomeadas | `backend/arquitetura_test.go` | O serviço do vínculo **não nomeia canal**: recebe `canal` como valor opaco, e quem sabe o que é `/start`, `t.me` e `secret_token` é o `controller` e o adaptador |

## Desenho

- **Vínculo por link.** `POST /spaces/{sid}/telegram/vinculo` gera um código aleatório (20 símbolos de um alfabeto de 32, 100 bits, o mesmo desenho dos códigos do MFA), guarda **só o hash** (`SHA-256` do código; a chave do item tem de sair do código sozinho, então não há `listaId` para usar de chave) com o `sub`, o `sid` e a validade (10 minutos), e devolve a **URL pronta** (`https://t.me/<bot>?start=<código>`). O nome do bot vem do servidor (SSM, tipo texto); **o front só abre a URL**.
- **Webhook.** `POST /telegram/webhook` (pública) confere o cabeçalho `x-telegram-bot-api-secret-token` em tempo constante contra o segredo do SSM; sem segredo configurado, **recusa tudo** (falha fechada). Com o segredo certo, só trata `message` de chat **privado**; o resto responde `200 {}` sem trabalho. `/start <código>`: formato, limite por chat (memória), **consome** o item (delete condicional que devolve o antigo) e só então grava `telegramChatId` no vínculo (`attribute_exists(PK)`). Consumir antes de gravar é de propósito: o código **nunca vale duas vezes**, e se a gravação falhar a pessoa toca em Ligar de novo.
- **Resposta no próprio webhook.** O texto que o bot manda de volta ("Ligado ✓", "Este link não vale mais") vai **no corpo da resposta** do webhook (`{"method":"sendMessage","chat_id":…,"text":…}`, forma que o Telegram aceita), sem uma segunda chamada HTTP e sem o token do bot nesse caminho. **A confirmar na documentação e no `dev`** (premissa P4).
- **Aviso só com a contagem.** `telegram.go` formata `🔔 [DEV] Avisos de dd/mm/aaaa: 2 atrasados · 1 vence hoje · 3 a vencer. Abra o app para ver o que é cada um.`, lendo **só** `domain.Contagens` (AD-1 e AD-3 da spine). A frase da contagem sai de uma função do pacote comum ao push e ao Telegram. O texto de teste e o `Mensagem` (MFA) não mudam.
- **Destino recusado é apagado.** `403` (bot bloqueado) e `400 chat not found` viram `ErrDestinoInvalido`; a rotina de 8h já apaga o chat id do vínculo. **`/parar` não apaga** (ver D2): responde um texto fixo que manda desligar em Listas.
- **Estado dos canais em `GET /me`.** Cada espaço ganha `canais: {"telegram": bool, "push": bool}` (há destino no vínculo), **somado** ao que existe hoje na fase 2 e **trocando** o `telegramChatId` na fase 3. O front relê `GET /me` ao voltar do Telegram e depois de ligar ou desligar qualquer canal.
- **Perfil.** `PUT /perfil` com `telegramChatId` **não vazio** é `400` (a fase 3 liga isso junto da tela); vazio continua apagando (é o "desligar" de hoje). Nenhum campo novo no vínculo.
- **Telas.** Uma **janela "Ligar um canal"** por cima da tela atual (componente só, entra em `camadasAbertas()` e `fecharCamadaTopo()`), usada pelas caixas "Avisar" do Lançar, do Repetir/Prever/Estender e da fatura do cartão quando **não há canal ativo no espaço**; "Continuar assim" guarda o aviso como hoje. O cartão **Avisos no Telegram** de Listas troca o campo do Id por **Ligar Telegram** e os estados. A prévia vira uma frase neutra. Tudo celular primeiro, claro e escuro, com demo antes.
- **Segredos e scripts.** Por ambiente: `/financas/<amb>/telegram-token` (já existe), `/financas/<amb>/telegram-webhook-secret` (SecureString, gerado pelo script) e `/financas/<amb>/telegram-bot-usuario` (texto). Dois scripts do dono, no molde de `guardar-fcm.ps1`: `guardar-telegram.ps1` (lê o token com `Read-Host -AsSecureString` e grava) e `registrar-webhook-telegram.ps1` (gera o segredo, grava, lê o usuário do bot por `getMe` e chama `setWebhook` com `allowed_updates: ["message"]`; **recusa** se o bot já tem outro webhook). **A IA nunca roda nenhum dos dois**: são do dono, ambiente por ambiente.

## O que o dono prepara (uma vez, à mão)

1. Criar o **bot do `dev`** no `@BotFather` (nome e usuário próprios; o do `dev` mostra `[DEV]` nas mensagens, como hoje). O da produção continua o atual.
2. Guardar os tokens: `scripts/guardar-telegram.ps1 -Ambiente dev|prod` (o `dev` com o token **novo**; **nunca** o mesmo token nos dois).
3. Depois do deploy de cada ambiente, `scripts/registrar-webhook-telegram.ps1 -Ambiente dev|prod`. Sem isso o link gera, mas o `/start` não chega.
4. Refazer o **próprio vínculo** pelo link (e, se não bastar, o ajuste por script dele; ver "Fase 6").

## Fases (cada uma é um PR; um assunto por branch)

0. **Demos e decisões (sem código).** O dono responde D1 a D4 e aprova as demos (abaixo), que partem da **última versão** de cada tela. Nada de `web/` antes.
1. **Aviso do Telegram só com a contagem** (backend + a prévia da tela Avisar, que passa a ser neutra): `telegram.go`, a função comum da contagem, erros do Telegram sem corpo, `ErrDestinoInvalido`, testes, `avisos.md`, `security.md`. Sem dado novo, sem rota nova. **Sai antes do vínculo** porque tira a exceção de privacidade (o Telegram deixa de levar valor) e porque não depende do dado novo.
2. **Backend do vínculo:** o item `TGLINK#<hash>` (**só depois de D1**), a rota de gerar, o webhook, `RotaPublica`, ações em `auth/authz.go`, `GET /me` com `canais` (**somado**), `template.yaml` (variável do bot, política do segredo e do usuário), os dois scripts, linhas no `api.md`, `modelo-de-dados/README.md`. Conferido no `dev` com o bot do `dev`.
3. **Tela de Listas** (depois da demo `demo-listas-v11-telegram-vinculo`): o cartão Avisos no Telegram sem o Id, os estados, o "Já iniciei", o fim de `financas.telegram` e, **na mesma entrega**, `PUT /perfil` recusando Id digitado e `GET /me` deixando de devolvê-lo.
4. **Telas de aviso neutras e a janela "Ligar um canal"** (depois das demos `demo-avisar-v3-canal`, `demo-lancar-v9-aviso-canal` e `demo-cartao-v5-aviso-canal`): Lançar, Repetir/Prever/Estender, aviso da fatura, toast da recorrência.
5. **O 🔔 de Histórico e Futuros diz "sem canal ativo"** (depois das demos `demo-historico-v5-sem-canal` e `demo-futuros-v4-sem-canal`). Fase 2 da intent.
6. **Verificação no `dev` e na produção:** o roteiro do `spec.md`, o dono refazendo o vínculo, `padrao-de-telas.md`, `guardrails.md`, `BACKLOG.md` e as demos em dia.

## Decisões a confirmar com o dono

| # | Decisão | Recomendação |
|---|---|---|
| **D1** | **Dado novo no DynamoDB:** um item `PK = TGLINK#<hash>`, `SK = TGLINK`, com `sub`, `sid`, `expiraEm` (segundos desde 1970) e `criadoEm`; **sem TTL** do DynamoDB (a tabela não tem e o MFA decidiu não ter); validade conferida no código; apagado ao ser usado ou ao ser achado vencido. Um código gerado e **nunca usado fica** na tabela (cerca de 200 bytes); o limite de 5 gerações por hora por pessoa (em memória) segura o ritmo. **Alternativa:** ligar o TTL do DynamoDB num atributo `ttl` (muda `template.yaml` e cria um campo) para limpar sozinho | Item sem TTL, como no MFA. Se os órfãos incomodarem antes de abrir ao público, uma rotina de limpeza ou o TTL entram no item 65 |
| **D2** | **`/parar` e o bloqueio na hora.** Achar o vínculo a partir do chat id (que é o que o Telegram manda) exige um **índice reverso** (item `TGCHAT#<chat>` ou um GSI), mais dado novo e mais escrita. **Sem ele**, `/parar` responde "para parar, desligue em Listas, Avisos" e o bloqueio do bot só é percebido na rotina das 8h seguinte (o `403` apaga o destino). **Isto contraria o item 4 da intent**, que prometia `/parar` apagando | Não fazer o índice: o `403` na rotina cobre o bloqueio e o desligar em Listas cobre o resto. Se o dono quiser `/parar` de verdade, vira feature à parte |
| **D3** | **Ordem frente ao item 47** (tela de aviso como componente). Os dois mexem nos rótulos e no balão da tela Avisar | O 66 vem **primeiro** (pré-requisito de abrir ao público), mexendo só em rótulos, balão e na janela; o 47 depois reaproveita. O `plan.md` do 47 anota a dependência |
| **D4** | **`GET /me` leva `canais` e perde `telegramChatId`** (em vez de uma rota de estado dos canais, como a intent previa). Muda o contrato de uma rota que o app lê na subida | Aprovar: não custa leitura nova e tira o chat id do navegador |
| D5 | Limites: código vale **10 min**; **5 gerações por hora** por pessoa; **10 tentativas por hora** por chat do Telegram; texto da contagem do Telegram (frase em "Desenho") | Os da tabela; o texto sai da demo |

## Premissas

Levantadas por quem escreveu o roteiro, lendo o código. **Não são a revisão independente** (dispensada pelo dono em 2026-11-13). Cada uma aponta a regra do `spec.md` que a cobre.

| # | A regra só funciona porque... | Se deixar de valer | Regra |
|---|---|---|---|
| P1 | **O destino mora no vínculo (por espaço):** o canal está "ativo" por espaço, e quem tem dois espaços liga o Telegram **em cada um** (um toque a mais) | Se o canal passasse a ser da pessoa, a dica "sem canal" de um espaço seria falsa no outro; hoje o push e o Telegram já são por espaço | R14, R19 |
| P2 | **Um bot aceita um webhook só, e `dev` e `prod` não podem dividir bot** | Com o mesmo token nos dois, registrar o webhook do `dev` tira o da produção e os `/start` da produção somem. Por isso o script recusa quando já há outro webhook | R23, R24 |
| P3 | **O cabeçalho do segredo chega em minúsculas** na Function URL e sem proxy no meio (o `/api/*` é chamada direta) | Se um dia o CloudFront ou outro proxy passar a ficar na frente, o nome ou o valor podem mudar e **toda** chamada legítima seria recusada (falha fechada, sem vazamento) | R4 |
| P4 | **O Telegram aceita a resposta no corpo do webhook** (`method: sendMessage`) | Se não aceitar o `chat_id` ou o texto, a pessoa liga mas não vê a confirmação; o app mostra "Ligado" mesmo assim (lido de `GET /me`). Confirmar no `dev` | R8 |
| P5 | **Tocar no link em um chat que já existe manda `/start <código>` na hora** (sem o botão Iniciar) | Se o cliente só abrisse o chat, o código não chegaria e a pessoa teria de enviar o `/start` à mão: o texto da tela diz o que fazer quando passam 10 minutos. **Conferir no aparelho** (inclui quem já tinha o bot) | R6, R17 |
| P6 | **O payload do `?start=` aceita só letras, números, `_` e `-`, até 64 caracteres** | Os 20 símbolos do alfabeto cabem; se o limite for menor, o código encurta (e perde bits): conferir antes de fechar o tamanho | R2 |
| P7 | **`403` do Telegram significa que a pessoa saiu** (bot bloqueado, conta apagada) e `400 chat not found` que o chat não existe mais | Se o Telegram usar `403` para outra coisa (por exemplo, o bot sem permissão num grupo), um destino válido seria apagado e a pessoa ficaria sem aviso até religar. Só chats privados são ligados, então esse caso não ocorre; **conferir os textos no `dev`** | R12 |
| P8 | **`GET /me` já lê o item inteiro do vínculo** (`Query` de `EspacosDe`) | Se um dia ela projetasse só alguns campos, `canais` ficaria sempre falso: o teste conta as leituras e confere `canais` | R14 |
| P9 | **O front antigo em cache** (PWA) ainda manda o Id e espera `telegramChatId` | Depois da fase 3, ele recebe `400` ao salvar e vê o Telegram como desligado até recarregar; front e backend vão no **mesmo release** | R15 |
| P10 | **O aviso do MFA continua pelo Telegram** como texto fixo (`Mensagem`) | Se a contagem trocasse o texto de todo `ResumoAvisos`, a frase do MFA sumiria: `mfa_aviso_test.go` segue verde sem mudar asserção | R11 |
| P11 | **A rotina de 8h agrupa por destino e conta o que o canal diz ter entregue** | Sem o `ErrDestinoInvalido` do Telegram, o destino bloqueado ficaria para sempre e a rotina tentaria todo dia | R12 |
| P12 | **O limite de 30 mensagens por segundo do Telegram e a Lambda de 30 s em série** (item 34) | Com centenas de usuários o envio das 8h não cabe numa Lambda; **com a mensagem curta** o problema encolhe, mas não some. Fica no item 65, medido antes de abrir | (Fora) |

**A pergunta ao contrário** (guardrails: "que outra feature contava com isso?") para o que esta feature esconde, limita ou preenche:
- **O campo do Id sai da tela:** contavam com ele a dica do Lançar (`telegramVinculado`) e o `telegram.test.js`; os dois mudam nesta feature.
- **O Telegram deixa de mostrar valor:** contava com isso quem lê o resumo para saber **quanto** pagar; é a perda dita na demo. O aviso da fatura do cartão (item 46) manda "Fatura <mês> · <cartão>" e o valor no corpo hoje: passa a contagem como os demais.
- **`GET /me` deixa de devolver `telegramChatId`:** o front é o único que o lê (`web/api.js`); o `api.md` e o spec do app multiusuário (`GET /me`) ganham a nota.
- **`PUT /perfil` recusa Id digitado:** contava com ele a importação de dados do dono (nenhuma) e o botão de teste, que usa o destino já gravado.

## Riscos

| Risco | Resposta |
|---|---|
| O webhook ser o alvo de enxurrada (é público) | Segredo conferido **antes** de qualquer trabalho; só chat privado; formato do código antes de qualquer leitura; limite por chat em memória; **nenhuma escrita** sem um código válido (a tabela tem 5 WCU) |
| Um código de vínculo vazar (link repassado) | Vale uma vez e 10 min; só liga o vínculo de quem o gerou; quem toca no link alheio passa a receber **a contagem** do outro (dano pequeno, aceito na intent) |
| O `dev` tomar o webhook da produção | Bots separados e o script recusa se o bot já tem outro webhook (P2); o token do `dev` nunca é o da produção |
| Sem o webhook registrado, o link parece não funcionar | A tela diz "Aguardando você tocar em Iniciar" por 10 minutos e depois "O link venceu"; o dono registra o webhook antes de abrir a tela a alguém; o `dev` prova o fluxo antes da produção |
| Resposta do webhook lenta (partida a frio) | O Telegram tolera; a Lambda do `dev` e da produção têm partida a frio de centenas de milissegundos; medir no `dev` |
| Quem já recebe o resumo com valor estranhar a mudança | É a decisão do dono; a demo e o texto da mensagem dizem "abra o app para ver o que é cada um" |
| Chat id antigo continuar em `localStorage` | O front o apaga na subida (R17) e há teste |

## Fontes (a conferir na hora de montar)

- Bot API, `setWebhook` (campo `secret_token` e o cabeçalho `X-Telegram-Bot-Api-Secret-Token`, `allowed_updates`) e a forma de responder no corpo do webhook: <https://core.telegram.org/bots/api#setwebhook>
- Links profundos (`?start=`): <https://core.telegram.org/bots/features#deep-linking>
- Erros do envio (`403`, `400`) e `my_chat_member`: <https://core.telegram.org/bots/api>
- Guia de webhooks do Telegram (segredo de 1 a 256 caracteres): <https://hookdeck.com/webhooks/platforms/guide-to-telegram-webhooks-features-and-best-practices>
