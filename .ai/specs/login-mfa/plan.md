# Plan: tela de login própria do app e verificação em duas etapas (opcional)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`template.yaml`, `web/api.js`, `backend/`). Cada
fase é uma entrega própria (branch, spec curto, demo quando há tela, PR, release).

**Estado:** rascunho para o dono aprovar.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| Login é a tela hospedada do Cognito (fluxo `code` + PKCE); o pool tem domínio, client e `AllowAdminCreateUserOnly` | `template.yaml` (`UserPool`, `UserPoolClient`, `UserPoolDomain`), `web/api.js` | A personalização é do Cognito; o app não desenha o formulário |
| O client pede só `openid` e `email`; o app guarda o **ID token** e o refresh em `localStorage` | `UserPoolClient.AllowedOAuthScopes`, `web/api.js` | Para o cadastro do MFA pela API é preciso o **access token** e o escopo `aws.cognito.signin.user.admin` |
| O pool não tem `MfaConfiguration` | `template.yaml` | A fase B acrescenta `OPTIONAL` com TOTP |
| O `username` do pool é o `sub` (o pool usa e-mail como atributo de login) | `UsernameAttributes: [email]` | Um endpoint do dono pode pedir ao Cognito "desligue o MFA do `sub`" sem guardar o e-mail |
| O backend valida o ID token e conhece `sub`, e-mail e papel | `auth/token.go`, `GET /me` | Nenhuma rota de lançamento muda |
| Telegram e `Notificador` prontos | `adapter/notificacao`, `service/avisos.go` | Servem à recuperação futura (fora destas fases) |

## Fase 0: verificar no `dev` (sem código no app)

Antes de desenhar, confirmar três fatos no ambiente de teste (nada vai a produção):
1. **Qual versão da tela hospedada** o pool usa (clássica ou *managed login*) e como se aplica a personalização em cada uma
   (`AWS::Cognito::UserPoolUICustomizationAttachment` na clássica; `AWS::Cognito::ManagedLoginBranding` na nova).
2. **Como a tela se comporta com MFA opcional**: confirmar que ela só pede o código de quem já cadastrou, e que não há "cadastrar
   aplicativo" por conta própria.
3. **O que o fator por e-mail exige** (plano, SES e custo), só para registrar a decisão.

Resultado: um parágrafo no spec da fase A e outro no da fase B. Sem PR de código.

## Fase A: tela de login personalizada

- **Demo primeiro** (Artifact): como o login ficaria com a identidade do app, claro e escuro, no celular e no computador, e o que fica
  fora do alcance do Cognito (a demo diz).
- Personalização aplicada no `template.yaml` para os dois ambientes: logo do app (arquivo no repositório, pequeno), cores do tema
  (os tokens do `padrao-de-telas.md`), título e campos. Recurso do tipo que a fase 0 indicar.
- Documentar em `padrao-de-telas.md` (seção nova "Tela de login") e em `hospedagem.md`/`security.md` o que é do Cognito e o que é do app.
- **Pronto quando:** demo aprovada; o `dev` mostra o login novo; o `prod` também depois do ok; `sam validate --lint` passa.

## Fase B: MFA opcional (TOTP) com tela no Perfil

- **Backend/infra:** `UserPool.MfaConfiguration: OPTIONAL`, `EnabledMfas: [SOFTWARE_TOKEN_MFA]`; o client passa a pedir o escopo
  `aws.cognito.signin.user.admin` (necessário para as chamadas de MFA com o access token); sem mudança em IAM da Lambda.
- **Front:** o `web/api.js` guarda também o *access token*; uma tela **Perfil → Verificação em duas etapas** (demo primeiro) com:
  estado (ligada ou desligada), "Ligar" (mostra o QR code e a chave, pede o primeiro código, confirma), "Desligar". As chamadas são
  `AssociateSoftwareToken`, `VerifySoftwareToken` e `SetUserMFAPreference` do Cognito, feitas pelo navegador com o access token.
- Biblioteca para o QR code só se for realmente necessária (o pedido é uma URL `otpauth://`); decidir no spec.
- **Segurança:** guardar o access token no mesmo `localStorage` do ID token amplia o que um XSS rouba; registrar no `security.md`
  (item 18 do BACKLOG já trata o XSS) e manter a validade curta.
- **Pronto quando:** demo da tela aprovada, MFA ligado e desligado por uma pessoa de teste no `dev`, login com e sem MFA funcionando,
  `security.md` com a seção de autenticação atualizada, testes em Node da tela.

## Fase C: o dono desliga o MFA de outra pessoa (botão no app)

- **Backend:** rota nova só do dono (`POST /spaces/{sid}/membros/{sub}/mfa/desligar`), no `controller`/`service` de sempre, que chama
  o Cognito (`AdminSetUserMFAPreference` com TOTP desabilitado) para o `sub` do vínculo. Permissão nova `cognito-idp:AdminSetUserMFAPreference`
  restrita ao pool, no `template.yaml`. Só atua em quem tem vínculo com o espaço do dono; registra no log quem pediu.
- **Front:** na área do dono (lista de pessoas do espaço), o botão "Desligar verificação em duas etapas" com janela de confirmação
  (demo primeiro, padrão de janela que já existe).
- **Pronto quando:** teste de API com o Cognito falso (não pode desligar fora do espaço, membro não consegue), demo aprovada,
  conferido no `dev`, `security.md` e `guardrails.md` atualizados.

## Depois (itens próprios, fora deste plano)

- **Recuperação pelo Telegram** (a pessoa pede o código no próprio Telegram, sem o dono): rota sem login, limite de tentativas,
  resposta igual para e-mail existente ou não, e guarda do código por poucos minutos (chave nova no DynamoDB, a discutir antes).
- **Fator por e-mail do Cognito** como segunda opção de MFA, se o custo e o SES valerem a pena.
- **Login 100% do app**, se a personalização do Cognito não bastar.

## Decisões para o dono (recomendação em negrito)

1. **Ordem das fases:** **0 → A → B → C** (login bonito primeiro; MFA e botão do dono depois). Alternativa: B antes de A, se a
   segurança vier na frente do visual.
2. **Mesma entrega de A para `dev` e `prod`**, com o `dev` conferido antes do release (**recomendado**).
3. **Quem é avisado a ligar o MFA:** só um lembrete discreto na tela de Perfil, **nunca bloqueante**; para o dono, um aviso a mais na
   abertura do app (a decidir na demo da fase B).
4. **A recuperação pelo Telegram fica fora** e vira o próximo item depois da fase C (**recomendado**).
