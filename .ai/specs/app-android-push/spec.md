# Spec: push (FCM) como canal de aviso do app Android

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id (`P-T<n>`), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent
divergirem, vale o plan.

**Estado:** rascunho para aprovação (2026-11-13). A revisão independente das premissas foi **dispensada pelo dono** (2026-11-13, sem agente para isso; pedir quando existir); as premissas levantadas por quem
escreveu estão no `plan.md` e cada uma aponta a sua regra aqui. Convenção dos testes: os de lógica são `go test ./...` em `backend/` e `node web/*.test.js`, e **falham antes da mudança**; o que depende do aparelho
ou do `dev` é marcado **[aparelho]** ou **[dev]** e entra como roteiro manual no PR, com o resultado escrito. Diga o que não deu para ver em vez de afirmar que funciona. **Token, credencial e
`google-services.json` nunca aparecem no chat, no log nem em captura de tela.**

## Regras

### Dado e rotas

- **R1. Campo `pushTokens`.** Conjunto de textos (`SS` do DynamoDB) no vínculo `USER#<sub>` / `SPACE#<sid>`, ao lado do `telegramChatId`. Nenhum item, chave ou índice novo. O atributo some quando o conjunto esvazia.
  **Decidido pelo dono em 2026-11-13** (opção "A"); nome e tipo confirmados neste roteiro.
- **R2. Registrar.** `PUT /spaces/{sid}/push` com `{"token": "<texto>"}` acrescenta o token ao conjunto do vínculo **de quem fez a chamada** (o `sub` vem do JWT, nunca do corpo). Idempotente: registrar o mesmo token de
  novo não muda nada. Valida: texto, sem espaços, só letras, números e `_ - : .`, entre 20 e 1024 caracteres; `400` se não bater. `404` se a pessoa não tem vínculo com o espaço (nada é criado). Resposta: `{"aparelhos": n}`;
  **nunca** devolve token. Vale para `owner` e `member`, cada um só no próprio vínculo.
- **R3. Remover do espaço.** `POST /spaces/{sid}/push/remover` com `{"token"}` tira o token do vínculo daquele espaço; token que não está lá não é erro. Resposta `{"aparelhos": n}`.
- **R4. Limite de 5 aparelhos** por pessoa e espaço, decidido **na própria gravação** (condição, sem leitura antes): um sexto token **diferente** é recusado com `409` e a frase "Você já tem 5 aparelhos neste espaço. Desligue um deles
  para ligar este."; o token que já está no conjunto continua aceito (idempotência).
- **R5. Sair.** `POST /push/sair` com `{"token"}` tira o token de **todos** os vínculos da pessoa (os espaços que `GET /me` lista), só os dela. Falha parcial não desfaz o resto e devolve `200` com os que conseguiu apagar;
  falha de leitura dos espaços devolve erro. Resposta `{"removidos": n}`.
- **R6. Destinos.** `LeitorContatos.Destinos(sid, sub)` devolve também `"push": [tokens]`, do `pushTokens` do mesmo `GetItem` (**nenhuma leitura nova**). Sem token, a chave `push` não aparece. Ordem estável (os tokens são
  ordenados).

### Canal

- **R7. Canal `push`.** `adapter/notificacao/fcm.go` implementa `service.Canal` com `Nome() == "push"`, sem importar o `service` (o `arquitetura_test.go` continua verde). É registrado no `main` junto do Telegram.
- **R8. Autenticação no Firebase.** A credencial é o JSON da conta de serviço, lido do SSM (SecureString) por `FCM_CREDENCIAL_PARAM`; o `project_id` vem dele. O canal assina um JWT RS256 (`iss` = e-mail da conta,
  `scope` = `https://www.googleapis.com/auth/firebase.messaging`, `aud` = `https://oauth2.googleapis.com/token`, `iat` e `exp` de no máximo 1 h) com `golang-jwt` (**sem biblioteca nova**), troca por um token de acesso e o guarda em cache
  até 5 minutos antes de vencer (um por processo, com trava). Sem parâmetro, sem valor ou JSON sem `private_key`, `client_email` ou `project_id`: `domain.ErrCanalNaoConfigurado`. Erro ao ler o SSM, ao assinar ou
  ao obter o token de acesso **não** vira "não configurado" (é falha do envio, com mensagem sem segredo).
- **R9. Mensagem e entrega.** `POST https://fcm.googleapis.com/v1/projects/<project_id>/messages:send` com `Authorization: Bearer`, tempo limite de 10 s, e o corpo `message.token`, `message.notification{title, body}` e
  `message.android{ttl: "86400s", notification{channel_id: "avisos"}}`. **Mensagem de notificação**, não de dados (decisão do plan). Cada token é uma chamada; uma falha não impede o token seguinte do mesmo destino
  (cada token é um destino, R12) e o erro devolvido é o primeiro.
- **R10. Erros do FCM.** `404` com `errorCode` `UNREGISTERED` devolve `domain.ErrDestinoInvalido`. `400`, `401`, `403`, `429`, `5xx` e erro de rede devolvem um erro com **só o código HTTP e o `errorCode`**; nunca a URL, o corpo
  inteiro, o token do aparelho nem o de acesso. Nenhum erro é repetido na mesma rodada.
- **R11. Token recusado é apagado.** Quando a entrega devolve `ErrDestinoInvalido`, a rotina chama `RemovedorDeDestinos.RemoverDestino(ctx, sid, sub, "push", token)` **para cada vínculo de onde o token saiu nesta rodada**, e conta
  o grupo como não entregue. Falha ao apagar só vai ao log (nome do canal e contagem). A rotina não devolve erro por isso.
- **R12. Agrupamento (do item 34, sem mudança).** O destino é o `(push, token)`: o mesmo aparelho em dois espaços recebe **uma** notificação com a contagem das linhas dos dois, sem nome de espaço. Dois aparelhos da mesma
  pessoa recebem uma cada. A mesma linha não conta duas vezes para o mesmo token (criador igual ao dono).
- **R13. Teste de aviso.** `POST /spaces/{sid}/avisos/teste` aceita `{"canal": "push"}` e envia o texto de teste a **todos os aparelhos da pessoa naquele espaço**, pelo despachante. Sem o campo, é o Telegram de hoje,
  idêntico. Respostas: `200`; `400` sem aparelho registrado; `503` canal não configurado; `502` o FCM recusou; `403` fora do espaço. `canal` com outro valor: `400`.

### Conteúdo

- **R18. Só a contagem.** `domain.TextoDoPush(r)` devolve título e corpo **só com contagens** por situação: `N atrasado(s)`, `N vence(m) hoje`, `N a vencer`, na ordem atrasados, hoje, a vencer, separados por ` · `, omitindo o que for
  zero. O título é `Finanças` (`[DEV] Finanças` no `dev`, pela marca do ambiente). **Nenhuma descrição, valor, meio de pagamento, data, id, nome de espaço ou nome de pessoa** entra no texto, **mesmo que as linhas os tenham**.
  Entradas (a receber) contam na mesma situação do vencimento. O teste de aviso vira "Teste: os avisos vão chegar aqui.".

### App

- **R14. Plugin `Push` (Java, nosso).** Em `app-android/android/`, registrado na `MainActivity`, exposto em `window.Capacitor.Plugins` sem bundler: `disponivel()`, `permissao()`, `pedirPermissao()`, `token()` e `apagarToken()`.
  Cria o canal de notificação `avisos` (importância padrão) ao carregar. O `FirebaseMessagingService` do app **ignora** mensagem recebida com o app aberto e, em `onNewToken`, só marca que o token mudou; **nada escreve o
  token em log**. O SDK do Firebase Messaging é a única dependência de terceiros nova. O manifest declara a permissão `POST_NOTIFICATIONS` e o serviço.
- **R15. Registrar de novo a cada abertura.** Com sessão, o app lê o token atual (`token()`) e chama `PUT /spaces/{sid}/push` para **cada espaço ligado** em `financas.push`, no máximo uma vez por abertura; a falha é silenciosa
  e tenta de novo na abertura seguinte. Isto cobre a troca de token do Firebase sem rotina nova.
- **R16. Indisponível não liga.** Sem Google Play Services, sem `google-services.json` no build (o plugin `google-services` não foi aplicado), ou com `disponivel()` falso, a chave aparece **desligada, desabilitada**, com a
  frase da demo ("Este aparelho não recebe avisos do app."); nada é gravado no aparelho nem chamado na API.
- **R17. Permissão negada.** Android 13 ou superior: a chave **explica o motivo antes** de pedir; se a pessoa negar (ou o sistema já tiver bloqueado), a chave volta a desligada com a frase da demo e o caminho para a
  configuração do app; **o app não pede de novo sozinho** nem insiste. Antes do Android 13 a permissão já vale e nada é pedido.
- **R19. Sair.** `sessao.sair()` no app, **antes** de apagar a sessão: `POST /push/sair` com o token atual (falha de rede **não impede** o Sair), depois `apagarToken()` e a remoção de `financas.push`. Se `apagarToken()` falhar, o
  Sair continua. A ordem existente (apagar a sessão e esperar, depois abrir o logout) não muda.
- **R20. Sem promessa de horário.** O texto da tela e o `avisos.md` não prometem que a notificação chega às 8h: ela sai às 8h do servidor e chega quando o aparelho a receber (economia de bateria e rede atrasam; sem Google
  Play Services não chega). O Telegram continua como reserva.

### Segredos, API e log

- **R21. Segredos fora do git.** `app-android/android/app/google-services.json` e qualquer `*-adminsdk-*.json` estão no `.gitignore`; um teste falha se algum estiver rastreado. A credencial entra no SSM **só** por
  `scripts/guardar-fcm.ps1 -Ambiente dev|prod -Arquivo <caminho>`: lê o arquivo, grava SecureString em `/financas/<ambiente>/fcm-credencial` (parâmetro **por ambiente**, P8), **não imprime o conteúdo**, não aceita o JSON
  na linha de comando e lembra de apagar o arquivo baixado. O `template.yaml` ganha `FCM_CREDENCIAL_PARAM` e a política de leitura **só desse parâmetro**; nenhuma outra mudança de infraestrutura.
- **R22. Autorização.** Três ações novas em `auth/authz.go`, com teste de negação para quem não tem vínculo: `push.registrar`, `push.remover` e `push.sair`; vale para `owner` e `member`, cada um só no próprio vínculo. O que não
  está liberado continua negado.
- **R23. Log.** Nada do que o canal, as rotas e a rotina registram contém token do aparelho, token de acesso, credencial, `client_email`, texto da notificação ou corpo da resposta do FCM. Só o nome do canal, contagens e o
  código HTTP.
- **R24. O que não muda.** Quem recebe o aviso (criador e dono; a fatura, só o dono), o horário (8h), os filtros, o agrupamento por destino, o Telegram, o evento `{"acao":"avisos"}` e a resposta `{"enviados": n}`, o login e a
  trava. Nenhuma chave nem item novo no DynamoDB; o front não decide quem recebe nem o que a notificação diz (`web/sem-status.test.js` continua verde).

## Testes

| Id | O que prova |
|---|---|
| P-T1 | `PUT /spaces/{sid}/push` grava o token no vínculo do chamador, é idempotente e devolve só a contagem; outro `sub` não consegue gravar no vínculo alheio (o `sub` é o do JWT); sem vínculo, `404` e nada criado (R2) |
| P-T2 | Token inválido (vazio, com espaço, curto demais, longo demais, caractere fora da lista) é `400` (R2) |
| P-T3 | O sexto token diferente é `409` com a frase; o quinto entra; o mesmo token de novo, com o conjunto cheio, é aceito (R4) |
| P-T4 | `POST /spaces/{sid}/push/remover` tira o token; token ausente não é erro; o conjunto vazio faz o atributo sumir do item (R1, R3) |
| P-T5 | `POST /push/sair` tira o token de **todos** os espaços da pessoa e de nenhum de outra pessoa; falha parcial devolve `200` com a contagem (R5) |
| P-T6 | `Destinos` devolve `"push"` ordenado e o `GetItem` continua sendo **um só** por pessoa e espaço (contador do banco falso) (R6) |
| P-T7 | `TextoDoPush` com linhas cheias de descrição, valor, banco e data não contém nenhum deles; zero some; `[DEV]` só no `dev`; singular e plural; entradas contam (R18) |
| P-T8 | Canal `push` com o servidor falso do FCM e do endpoint de token: o JWT tem `iss`, `scope`, `aud` e `exp` certos e é assinado com a chave do teste; o token de acesso é pedido **uma vez** para dois envios e de novo depois de vencer; o corpo do envio é o do R9 (R8, R9) |
| P-T9 | Sem parâmetro, sem valor ou JSON incompleto: `ErrCanalNaoConfigurado` e nenhuma chamada de rede; erro do SSM e do endpoint de token **não** viram "não configurado" (R8) |
| P-T10 | `404 UNREGISTERED` devolve `ErrDestinoInvalido`; `400`, `401`, `429`, `503` e erro de rede devolvem erro **sem** a URL, o token nem o corpo; nenhum é repetido (R10) |
| P-T11 | Rotina com canal falso que devolve `ErrDestinoInvalido`: o token é apagado de **cada** vínculo de onde saiu, o grupo não conta como entregue, a rotina não devolve erro; os outros destinos entregam (R11) |
| P-T12 | O mesmo token em dois espaços: **uma** notificação com a contagem somada; dois tokens da mesma pessoa: duas; a mesma linha não conta duas vezes para o mesmo token (R12) |
| P-T13 | Teste de aviso com `{"canal":"push"}`: `200` e uma notificação por aparelho; `400` sem aparelho; `503`; `502`; `403`; canal inválido `400`; sem o campo, igual ao de hoje (`TestMensagemDeTeste` passa **sem mudar asserção**) (R13) |
| P-T14 | `authz`: as três ações liberadas para `owner` e `member`, negadas a quem não tem vínculo e a rota desconhecida segue negada (R22) |
| P-T15 | Nenhuma linha de log da rotina, das rotas e do canal contém token, texto da notificação, credencial nem corpo da resposta do FCM (R23) |
| P-T16 | `arquitetura_test.go` verde; `fcm.go` não importa o `service`; `golang.org/x/oauth2` e `firebase.google.com/go` **não** entram no `go.mod` (R7, R8) |
| P-T17 | `.gitignore` cobre `google-services.json` e `*-adminsdk-*.json`, e nenhum está rastreado; `scripts/guardar-fcm.ps1` não imprime o conteúdo, não aceita o JSON como parâmetro e recusa ambiente que não seja `dev` ou `prod` (R21) |
| P-T18 | `template.yaml`: `FCM_CREDENCIAL_PARAM` por ambiente e a política de leitura só desse parâmetro; o resto do template idêntico (R21) |
| P-T19 | `web/push.test.js`, com plugin e `fetch` falsos: ligar chama a permissão, o token e o `PUT`, e só então grava `financas.push`; permissão negada deixa tudo desligado e **não** pede de novo; plugin indisponível não grava nem chama a API; a cada abertura com sessão registra de novo os espaços ligados, uma vez; falha de rede na abertura não apaga a chave (R14 a R17) |
| P-T20 | `web/sessao.test.js`: o Sair chama `/push/sair` e `apagarToken()` **antes** de apagar a sessão; falha de rede no `/push/sair` e no `apagarToken()` não impede o Sair; `financas.push` some (R19) |
| P-T21 | O front não tem texto de aviso nem regra de quem recebe: `web/sem-status.test.js` e `web/carga.test.js` continuam verdes, e a chave só existe no app (R24) |
| P-T22 **[aparelho]** | Ligar a chave no `dev`: a explicação, o pedido de permissão do Android e o aparelho registrado (a contagem do servidor sobe); recusar a permissão deixa a chave desligada sem pedir de novo (R15, R17) |
| P-T23 **[dev]** | Rodar a rotina no `dev` com avisos fictícios: **uma** notificação com a contagem certa chega ao aparelho em segundo plano e com o app fechado; **nada** além da contagem aparece, nem na tela bloqueada; tocar abre o app e, com a trava ligada, o bloqueio aparece (R9, R18) |
| P-T24 **[dev]** | Mesmo token em dois espaços: **uma** notificação com a soma; com o app aberto, nenhuma notificação do sistema aparece (R12, R14) |
| P-T25 **[dev]** | Enviar a um token fictício na rotina real do `dev` o apaga do vínculo (conferir o campo antes e depois); o log só traz contagens e o nome do canal (R11, R23) |
| P-T26 **[aparelho]** | Sair da conta, com rede: o aparelho some do servidor e a rotina seguinte não manda nada; sair no modo avião: o token fica no servidor, o token no FCM é apagado e a rotina seguinte o remove (R5, R19) |
| P-T27 **[aparelho]** | Desinstalar o app e rodar a rotina: o token é apagado pela rotina (R11). Reinstalar e ligar de novo funciona (R2, R15) |
| P-T28 **[aparelho]** | Medir o atraso real da notificação com a economia de bateria padrão da MIUI e registrar o resultado, sem prometê-lo (R20) |
| P-T29 **[aparelho]** | Pior caso de layout da chave (frase de "indisponível", tema claro e escuro, 360 e 412 px) medido por script e com captura (guardrails, "Telas") (R16, R17) |

**Quebrar de propósito:** pôr a descrição no corpo da notificação tem de derrubar P-T7; apagar o token em qualquer erro (e não só `UNREGISTERED`) tem de derrubar P-T10; gravar o `sub` do corpo em vez do JWT tem de
derrubar P-T1; esquecer a condição de tamanho do conjunto tem de derrubar P-T3; pedir um token de acesso a cada envio tem de derrubar P-T8; o Sair apagar a sessão antes do `/push/sair` tem de derrubar P-T20; gravar
`financas.push` antes do `PUT` voltar tem de derrubar P-T19.

## Telas (demo antes do código)

**A chave "Receber avisos neste aparelho" é tela nova: nada dela vai ao código antes de uma demo aprovada** (`guardrails.md`). A demo `demo-push-v1` parte da **última versão de Listas** (a tabela do
`padrao-de-telas.md` diz qual é) e mostra, celular primeiro, claro e escuro: a chave desligada, ligada e com "N aparelhos"; a explicação antes do pedido de permissão; a permissão negada; o aparelho indisponível; o
limite de 5; o erro de rede; o botão de teste; e **o que a proposta faz perder** (a notificação não diz o que vence; tocar abre o app; não há horário garantido; a chave vale por espaço). Termina com o aviso na tela
bloqueada, só com a contagem.

## Verificação no `dev`

1. Fazer o deploy e conferir o changeset e o status final da stack (guardrails): a tabela e o login **não** podem ser substituídos; o changeset da fase 1 deve mostrar **só** a Lambda e a política de leitura do parâmetro novo.
2. O dono cria o projeto no Firebase e guarda a credencial do `dev` por `scripts/guardar-fcm.ps1` (passos do `plan.md`). Sem isto, o canal fica "não configurado" e o resto segue funcionando.
3. Registrar um token **fictício** pelo `PUT` (sessão do `dev`), rodar `{"acao":"avisos"}` com avisos fictícios e conferir que o token é apagado (P-T25).
4. Com o app do `dev` e o token real: P-T22 a P-T28.
5. No Logs Insights de `/aws/lambda/financas-api-dev`, conferir que as linhas da rotina e das rotas novas trazem só contagens, códigos e o nome do canal (R23).

## Fora do escopo

iOS, e-mail, WhatsApp, avisar a esposa das contas do dono, idempotência da entrega, horário por pessoa, ações na notificação, mensagem de dados, tela de "meus aparelhos" com nomes, App Link e Play Store (fase 5 do
item 54, que passa a ser o item 64) e qualquer campo além de `pushTokens`.
