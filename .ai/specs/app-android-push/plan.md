# Plan: push (FCM) como canal de aviso do app Android

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/service/despachante.go`, `service/avisos.go`, `repository/pessoas.go`, `repository/perfil.go`,
`adapter/notificacao/telegram.go`, `auth/authz.go`, `roteador.go`, `web/js/sessao.js` e o projeto Android em `app-android/`). O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** rascunho para aprovação (2026-11-13). **Cria um campo no DynamoDB (`pushTokens`), já aprovado pelo dono** (opção "A", 2026-11-13); nenhuma chave nem item novo. A **revisão independente das
premissas** (guardrails, "Roteiros") **não foi feita**: o dono não tem agente para isso e a **dispensou em 2026-11-13**; ela é pedida quando existir um. A seção "Premissas" traz só o que **quem escreveu**
levantou e **não a substitui**. O que o Firebase exige foi lido na documentação oficial em 2026-11-13 (links em "Fontes") e deve ser **conferido de novo** no console na hora de montar, porque muda.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| O despachante entrega um `ResumoAvisos` a um `Destino{Canal, Endereco}`; canal novo é um arquivo em `adapter/notificacao/` que implementa `service.Canal` sem importar o `service` | `service/despachante.go`, `arquitetura.md` | O push é `fcm.go` e o `main` o registra ao lado do Telegram. O `service` não aprende o que é FCM |
| A rotina agrupa por `(canal, endereço)`, sem o espaço, em ordem estável, e `ErrCanalNaoConfigurado` pula o resto daquele canal | `service/avisos.go` | O mesmo token em dois espaços recebe **uma** notificação com a contagem somada; sem credencial, só o push é pulado |
| `LeitorContatos.Destinos(sid, sub)` lê o vínculo por `GetItem` e devolve `{"telegram": [chat id]}` | `repository/pessoas.go` | Passa a devolver também `"push": [tokens]`, do campo `pushTokens` do mesmo item (sem leitura nova) |
| O vínculo (`USER#<sub>` / `SPACE#<sid>`) guarda `role`, `apelido` e `telegramChatId`; `AtualizarPerfil` usa `UpdateItem` com `attribute_exists(PK)` sobre o item de quem chamou | `repository/perfil.go` | `pushTokens` é um **conjunto de textos** no mesmo item, gravado com `ADD`/`DELETE` e a mesma condição: só quem tem vínculo grava, e só no próprio |
| `GET /me` lista os espaços da pessoa (`Espacos.ListarEspacos`) | `repository/espacos.go` | O "Sair" acha todos os espaços do token por essa consulta; não há varredura |
| O token do Telegram vem do SSM por `TELEGRAM_TOKEN_PARAM`, fica em cache no processo, e sem ele o canal devolve `ErrCanalNaoConfigurado` | `adapter/notificacao/telegram.go`, `template.yaml` (`SSMParameterReadPolicy`) | A credencial do Firebase segue o mesmo molde: `FCM_CREDENCIAL_PARAM`, política de leitura no `template.yaml` |
| `golang-jwt/jwt/v5` já é dependência do backend | `go.mod` | Assinar o JWT da conta de serviço (RS256) e trocá-lo por um token de acesso **não pede biblioteca nova** |
| Cada rota nova pede uma ação em `auth/authz.go`, com teste; o que não está liberado é negado | `auth/authz.go`, `security.md` | Ações novas: `push.registrar`, `push.remover` e `push.sair` |
| O app tem os plugins `Cofre` e `Trava` em Java, registrados na `MainActivity`; `google-services` só é aplicado se existir `app/google-services.json` (bloco `try` já no `build.gradle`) | `app-android/android/app/build.gradle`, `MainActivity.java` | O plugin `Push` entra do mesmo jeito; sem o arquivo o app compila e o plugin diz `disponivel() = false` |
| `sessao.sair()` apaga o refresh, `financas.space`, `financas.trava` e `financas.recentes`, depois abre o logout | `web/js/sessao.js` | O Sair ganha dois passos antes de apagar: remover o aparelho do servidor e apagar o token no FCM |
| A tela Listas tem o cartão do Telegram (`#telegramCard`) e as chaves de Segurança só no app | `web/index.html`, `web/js/app.js` | A chave nova é tela nova: **demo aprovada antes do código** |

## Desenho

- **Dado (`pushTokens`).** Conjunto de textos (tipo `SS` do DynamoDB) no vínculo da pessoa no espaço. `ADD` para registrar (idempotente: o mesmo token duas vezes é um só), `DELETE` para tirar; o DynamoDB apaga o
  atributo quando o conjunto esvazia. Limite de 5 por pessoa e espaço, por condição na própria gravação (`attribute_not_exists(pushTokens) OR contains(pushTokens, :t) OR size(pushTokens) < :max`), sem leitura antes.
- **Rotas** (todas do próprio chamador; o `sub` vem do token, nunca do corpo):
  `PUT /spaces/{sid}/push` registra `{"token"}`; `POST /spaces/{sid}/push/remover` tira `{"token"}` daquele espaço; `POST /push/sair` tira `{"token"}` de **todos** os espaços da pessoa. Nenhuma devolve token.
- **Canal `push` (`adapter/notificacao/fcm.go`).** `Nome()` é `"push"`. Formata **só a contagem** por função pura de `domain/` (`TextoDoPush`): título `Finanças` (com `[DEV] ` no `dev`), corpo
  `2 atrasados · 1 vence hoje · 3 a vencer` (some o que for zero). `ResumoAvisos{Teste: true}` vira "Teste: os avisos vão chegar aqui.".
- **Autenticação no Firebase, sem biblioteca nova.** A credencial é o JSON da conta de serviço, no SSM (`FCM_CREDENCIAL_PARAM`, SecureString). O canal monta o JWT (RS256, `iss` = e-mail da conta, escopo
  `https://www.googleapis.com/auth/firebase.messaging`, `aud` = `https://oauth2.googleapis.com/token`, validade de até 1 h), troca por um token de acesso no endpoint de token (`grant_type` do tipo `jwt-bearer`) e o
  guarda em cache até pouco antes de vencer. O `project_id` vem do mesmo JSON. Depois envia `POST https://fcm.googleapis.com/v1/projects/<project_id>/messages:send` com `Authorization: Bearer`.
- **Mensagem.** `message.token`, `message.notification{title, body}` e `message.android{ttl, notification{channel_id}}`. É mensagem de **notificação**: com o app em segundo plano ou fechado o próprio Android a
  mostra, **sem código nosso rodando**. Com o app aberto ela chega ao serviço e é ignorada (o app já está na tela). TTL de 1 dia (o aviso de ontem não vale amanhã, e a rotina manda o de amanhã).
- **Token recusado.** `UNREGISTERED` (HTTP 404) vira `domain.ErrDestinoInvalido`; a rotina, que sabe de quais vínculos o destino saiu, chama `RemovedorDeDestinos` e apaga o token de cada um. Outros erros
  (`429`, `5xx`, `401`) não apagam nada, não se repetem e vão ao log só com o nome do canal e o código (mesma regra do item 34).
- **Plugin `Push` (Java, nosso).** `disponivel()`, `permissao()`, `pedirPermissao()` (Android 13 ou superior; antes dele a permissão já vale), `token()` e `apagarToken()`; cria o canal de notificação `avisos` ao
  carregar. O SDK do Firebase Messaging (do Google) entra como dependência do app: **é o que obtém o token e é obrigatório**, e é o único código de terceiros novo.
  Um `FirebaseMessagingService` mínimo existe porque o SDK pede; `onNewToken` só marca "o token mudou" no armazenamento do app, e **não escreve o token em log**.
- **Front (`web/`).** A chave "Receber avisos neste aparelho" fica em Listas, ao lado do cartão do Telegram, **só no app**. Ligar: explica o porquê, pede a permissão, pega o token, chama `PUT /spaces/{sid}/push`.
  O estado por espaço fica no aparelho (`financas.push`, lista de espaços, como a trava), é apagado no Sair, e a cada abertura com sessão o app **registra de novo** o token atual nos espaços ligados (cobre a troca de token).
  O front **não decide quem recebe nem o que dizer**: só guarda a escolha e chama a API.
- **Sair.** Antes de apagar a sessão: `POST /push/sair` (se falhar por rede, o Sair segue) e `apagarToken()`, que invalida o token no FCM; assim, mesmo sem rede, o servidor apaga o token na próxima rotina (R11 e R19 do `spec.md`).
- **Teste de aviso.** `POST /spaces/{sid}/avisos/teste` aceita `{"canal": "push"}` e manda o teste aos aparelhos da pessoa naquele espaço; sem o campo, é o Telegram de hoje. É também como se confere no `dev`.

## O que o dono prepara (uma vez, à mão)

1. Criar um projeto no [Firebase Console](https://console.firebase.google.com/) (o Analytics pode ficar desligado) e registrar **dois apps Android**: `br.com.cicerotec.financas` e `br.com.cicerotec.financas.dev`.
2. Baixar o `google-services.json` (um só, com os dois apps) e guardar em `app-android/android/app/google-services.json`: o `.gitignore` o ignora.
3. Em Configurações do projeto, Contas de serviço, gerar uma chave privada (JSON) e guardá-la com `scripts/guardar-fcm.ps1` (lê o arquivo, grava no SSM como SecureString e não imprime nada; **nunca** colar o conteúdo no chat). Apagar o arquivo baixado depois.
4. Conferir no Google Cloud que a API "Firebase Cloud Messaging API" está habilitada no projeto (o console do Firebase normalmente já a habilita).

## Fases (cada uma é um PR)

1. **Backend** (sem tela): `domain` (`TextoDoPush`, `ErrDestinoInvalido`), `fcm.go`, `Destinos` com `push`, rotas e ações, remoção do token recusado, `template.yaml` (parâmetro e política), `scripts/guardar-fcm.ps1`, testes. Conferido no `dev` com um
   token **fictício** (a rotina deve apagá-lo) e, com o app da fase 2, com o token real.
2. **Plugin e app** (sem tela nova): `PushPlugin.java`, o serviço, o `build.gradle` (BoM do Firebase e o plugin `google-services`), o canal, o manifest (permissão e serviço), testes de lógica do que couber no JS, e o teste no aparelho
   com o token impresso **só no aparelho** (nunca no chat).
3. **Tela** (depois da demo aprovada): a chave em Listas, o registro, o Sair e as mensagens de erro; atualiza `padrao-de-telas.md`, as demos e o resumo do `guardrails.md` na mesma entrega.

A demo (`demo-push-v1`, parte da última versão de Listas) pode ser feita **já**, em paralelo à fase 1, porque só exige decisão do dono.

## Premissas

Levantadas por quem escreveu o roteiro, lendo o código e a documentação. **Não são a revisão independente** (dispensada pelo dono em 2026-11-13; pedir quando houver agente). Cada uma aponta a regra do `spec.md` que a cobre.

| # | A regra só funciona porque... | Se deixar de valer | Regra |
|---|---|---|---|
| P1 | **O token é do aparelho, não do espaço**: o mesmo token entra nos vínculos dos espaços em que a pessoa ligou a chave, e o despachante agrupa por token | Se o agrupamento por destino mudasse (item 34), o mesmo aparelho receberia uma notificação por espaço, e a de um espaço mostraria só a contagem dele | R6, R12 |
| P2 | **O SDK do Firebase só dá o token se o app inicializou o Firebase**, e isso depende do `google-services.json` no build | Sem o arquivo o app compila, mas `token()` falha: a chave tem de dizer "indisponível", nunca ficar ligada sem aparelho registrado | R16, R17 |
| P3 | **Mensagem de notificação é exibida pelo Android sozinha só com o app em segundo plano ou fechado**; com o app aberto chega ao código | Se o serviço mostrasse algo em primeiro plano, a notificação apareceria por cima do app que a pessoa está usando, e um código nosso passaria a tratar conteúdo | R9 |
| P4 | **A entrega às 8h depende do aparelho e do FCM**: a mensagem espera o aparelho voltar (até o TTL), o aparelho em economia de bateria pode atrasar, e aparelho sem Google Play Services não recebe | O aviso pode chegar tarde ou não chegar; o Telegram continua de reserva. O app **não promete horário** | R9, R20 |
| P5 | **O servidor só apaga um token quando o FCM diz `UNREGISTERED`**, e só a rotina diária o descobre | Entre o desinstalar e a próxima rotina o token continua guardado (inofensivo: o FCM descarta); se o FCM mudar o código do erro, tokens mortos se acumulam até o limite de 5 e a pessoa não consegue ligar o aparelho novo | R11, R4 |
| P6 | **O Sair depende da rede para tirar o aparelho do servidor** | Sem rede o token fica gravado; o `apagarToken()` do FCM o invalida e a rotina o apaga; antes disso uma notificação (só com contagem) pode chegar a um aparelho que saiu da conta | R19 |
| P7 | **Um `ADD` num conjunto só funciona para o tipo `SS`**: se o atributo existisse com outro tipo, a gravação falharia | Hoje o campo não existe em nenhum item; o teste cria o primeiro, e o repository só fala em `SS` | R1 |
| P8 | **A credencial de serviço vale para `dev` e `prod` se for a mesma conta**; cada ambiente lê o seu parâmetro | Com um parâmetro só, o `dev` mandaria avisos com a credencial de produção. Por isso há **dois parâmetros** e o script grava o do ambiente escolhido | R8, R21 |
| P9 | **O app `dev` e o `prod` têm tokens diferentes** (pacotes diferentes) mesmo no mesmo aparelho | Se os dois fossem o mesmo app, os dois ambientes dividiriam token e a rotina de um apagaria o do outro | R2 |
| P10 | **O limite de 4096 bytes do payload e o do TTL** valem como a documentação diz | Contagem curta nunca chega perto; se o limite caísse, o teste de tamanho do texto falharia | R18 |

## Propostas que o dono pode contestar

1. Notificação por **mensagem de notificação** (o Android mostra) em vez de mensagem de **dados** (o app monta): é mais simples e menos código nosso rodando com o app fechado. A troca ao contrário só vale se um dia a notificação precisar de ação.
2. **Limite de 5 aparelhos** por pessoa e espaço.
3. **TTL de 1 dia.**
4. **A chave por espaço**, com o estado no aparelho (`financas.push`), em vez de o servidor dizer qual aparelho está ligado (o servidor guarda tokens, não sabe qual é o deste aparelho sem receber o token).
5. **Só `UNREGISTERED` apaga o token** (`INVALID_ARGUMENT` e o resto vão ao log).
6. `POST` para remover (e não `DELETE` com corpo), para não depender de cliente que descarta o corpo.
7. Revisão independente das premissas **dispensada** por falta de agente.

## Riscos

| Risco | Resposta |
|---|---|
| Notificação revelar o que a pessoa deve | O texto sai de `TextoDoPush`, que só conta; teste com linhas cheias de descrição, valor e meio de pagamento prova que nada disso entra (R18) |
| Credencial do Firebase vazar | Só no SSM; o script lê de arquivo e não imprime; nenhum log leva o JSON, o token de acesso nem o token do aparelho; o arquivo baixado é apagado; se vazar, gerar outra chave e revogar a antiga |
| `google-services.json` ir para o git | `.gitignore` e um teste que falha se o arquivo estiver rastreado (R21) |
| Rotina de avisos demorar mais com dois canais | O envio é em série dentro dos 30 s da Lambda (risco aceito no item 34); cada chamada ao FCM tem tempo limite de 10 s, como o Telegram. O token de acesso é reaproveitado entre os envios |
| Token novo do Firebase e servidor com o antigo | A cada abertura o app registra o token atual nos espaços ligados (R15); o antigo vira `UNREGISTERED` e é apagado |
| Aparelho Xiaomi/MIUI matar o app e atrasar a notificação | Notificação de mensagem é entregue pelo sistema, mas a economia de bateria da MIUI pode atrasar; **conferir no aparelho do dono** e registrar o que for visto (o spec não promete horário) |
| Mudar a rotina de avisos e deixar de avisar pelo Telegram | Os testes do item 34 continuam passando sem mudar asserção; o push é só mais um canal |
| Mudar regra do Firebase (campo, escopo, endpoint) | Conferir a documentação e o console na hora de montar (fontes abaixo) |

## Fontes (lidas em 2026-11-13)

- Enviar mensagens pela API HTTP v1 do FCM: <https://firebase.google.com/docs/cloud-messaging/send/v1-api>
- Receber mensagens no app Android (serviço, permissão `POST_NOTIFICATIONS`, canal, `onNewToken`): <https://firebase.google.com/docs/cloud-messaging/android/client>
- Códigos de erro do FCM (`UNREGISTERED` e demais): <https://firebase.google.com/docs/cloud-messaging/error-codes>
- Duração e entrega de mensagens (`android.ttl`): <https://firebase.google.com/docs/cloud-messaging/customize-messages/setting-message-lifespan>
- OAuth 2.0 com conta de serviço, fluxo só por HTTP: <https://developers.google.com/identity/protocols/oauth2/service-account>
