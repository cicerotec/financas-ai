# Plan: MFA opcional por aplicativo autenticador

Como construir o que o [`intent.md`](intent.md) pede. Cada fase é um PR pequeno e vai primeiro ao `dev`. Tela nova pede **demo aprovada antes do código**.

**Estado:** rascunho para aprovação (2026-11-19). Nada implementado. O `spec.md` vem depois da aprovação deste plan. **A revisão independente das premissas ainda não foi feita** (ver "Premissas").

## Onde estamos hoje

| Fato | Onde | Consequência |
|---|---|---|
| O pool não define MFA; o plano do pool é Essentials (confirmado no console em 2026-11-18/19) | `template.yaml` (`UserPool`) | Basta uma propriedade no pool; o plano não muda |
| Cliente com `openid email`, fluxo `code` e `ALLOW_REFRESH_TOKEN_AUTH` | `template.yaml` (`UserPoolClient`) | O escopo `aws.cognito.signin.user.admin` precisa ser **permitido** no cliente |
| O front guarda só o **ID token** e o refresh token; o access token nunca é guardado | `web/js/sessao.js` (`salvarAuth`, `token`) | A tela do MFA precisa do **access token**, que hoje o app não tem |
| O backend valida só o ID token | `backend/auth/token.go` | **Não muda**: as APIs de MFA vão do front direto ao Cognito, sem passar pela Lambda |
| Login no Hosted UI clássico, na aba do sistema no Android e na mesma aba no navegador | `web/js/sessao.js` | O desafio do código já aparece ali (testado no `dev`, intent) |
| Listas tem o cartão Aparência com a subseção Segurança (trava e apps recentes, só no app, valem só no aparelho) | `.ai/padrao-de-telas.md` | A seção do MFA **não** é só do app e **não** vale só no aparelho: precisa ficar em lugar próprio |
| Provado no `dev` (2026-11-19): token do Hosted UI com o escopo novo aceito por `GetUser`, `Associate`, `Verify` e `SetUserMFAPreference` | `intent.md` | O desenho funciona; o que falta é construir |

## Decisão de desenho (a mais importante) — **confirmada pelo dono em 2026-11-19**: o escopo só no segundo login

O escopo `aws.cognito.signin.user.admin` dá ao **access token** o poder de todas as APIs de autoatendimento do Cognito (trocar a senha, apagar o próprio usuário, mexer no MFA), não só do MFA. Pedir esse escopo em **todo login** deixaria
esse token, mais poderoso, circulando no app o tempo todo, e um XSS (ponto em aberto do `security.md`) o alcançaria.

**Proposta do plan:** o login normal **continua pedindo só `openid email`**. O escopo novo é pedido **só quando a pessoa abre a seção do MFA**, num segundo login ("confirmar que é você"), e o token vive **só em memória**, até a pessoa sair da seção.
O cliente **permite** o escopo no `template.yaml`, mas o app só o **pede** nesse momento. Isso também ajuda o MFA: o segundo login passa pelo Cognito (senha e, se já houver MFA, o código), então **desligar o MFA exige provar de novo quem a pessoa é**.

- **Alternativa:** pedir o escopo em todo login. É mais simples (um login só, o token já está à mão), mas o token mais poderoso fica sempre disponível. **Não recomendada.**
- **Ponto a verificar no `dev`, não provado:** se o Hosted UI aceita forçar a senha de novo (o parâmetro `prompt=login` ou `max_age`), em vez de reaproveitar a sessão por cookie (a tela "continuar com este usuário ou trocar" apareceu nos testes e o login voltou sem senha). Sem isso, o "confirmar de novo" não confirma nada e a proposta perde o argumento do MFA (continua valendo o do escopo em memória).

## Fases (cada uma é um PR)

### Fase 1: demo da tela (sem código)
- Demo versionada, partindo da **última versão de Listas** (a tabela do início de `.ai/padrao-de-telas.md` diz qual é a padrão; as de Avisos, `demo-push-v2-telegram-chave`, e Bloqueio, `demo-bloqueio-v1`, são as mais recentes citadas no guardrails): `demo-listas-v<N>-mfa.html`, artefato privado com link que abre no celular.
- Mostra a tela inteira de Listas, com a nova seção **Segurança da conta** (separada da Segurança do aparelho do cartão Aparência, que fica como está) e estes estados: MFA desligado; confirmando a identidade (o segundo login); cadastrando (QR code e chave para digitar, campo do código); **aviso obrigatório** com a confirmação de leitura antes de ligar; ligado; desligar (com confirmação e o aviso de 6 s do padrão do app); erro de código; sem internet.
- Seletores nativos, janelas de baixo, linhas de pelo menos 56 px, tokens do app, tema claro e escuro, celular (≈ 390 px) e computador (≥ 1280 px). A demo diz o que a proposta faz perder.
- **A demo diz** (decidir com o dono): a seção aparece para dono e membro, no navegador, no PWA e no app.

### Fase 2: `template.yaml` e deploy no `dev`
- `UserPool`: `MfaConfiguration: OPTIONAL` e `EnabledMfas: [SOFTWARE_TOKEN_MFA]` (só TOTP; o nome exato das propriedades se confere em `sam validate --lint`).
- `UserPoolClient`: `AllowedOAuthScopes: [openid, email, aws.cognito.signin.user.admin]` (permitir, não pedir). **Não** acrescentar `ALLOW_USER_PASSWORD_AUTH` nem nenhum fluxo de senha.
- **Verificar no changeset** (o deploy já falha se substituir pool ou cliente; conferir também a olho): só **modificação de propriedade**, sem `Replacement: True` no pool nem no cliente. Se o changeset mostrar substituição, **parar** e voltar ao dono.
- O `dev` já está com o MFA opcional e o escopo ligados à mão: o primeiro deploy deve mostrar essas propriedades como já conformes ou como ajuste mínimo. Conferir que ele **não desfaz** o que está funcionando.
- A role de deploy precisa de permissão para as ações novas do Cognito? Conferir `infra/github-oidc.yaml` **antes do merge** (guardrails).
- Atualizar `.ai/security.md` (autenticação e a tabela de pontos em aberto) e o item 49 do BACKLOG (Fase 1: tirar o "reset do dono").

### Fase 3: front (sessão e tela)
- **`web/js/sessao.js`:** uma função nova que faz o segundo login com o escopo novo e devolve o access token **só em memória** (navegador e app), sem passar por `localStorage`, `sessionStorage` nem pelo Cofre; descartado ao sair da seção. No app Android abre na aba do sistema, como o login de hoje. Nenhum log de token, `code` ou `verifier` (R18 do `app-android`).
- **Chamadas ao Cognito** (`https://cognito-idp.<região>.amazonaws.com/`, por `fetch`, sem SDK): `GetUser`, `AssociateSoftwareToken`, `VerifySoftwareToken`, `SetUserMFAPreference`, com o access token no corpo, nunca na URL nem em log. Módulo novo, sem regra de negócio; o front **só exibe** e devolve o que o Cognito responder.
- **QR code:** decidir entre uma biblioteca pequena embutida em `web/` (sem CDN) e código nosso; sem serviço externo (a chave do TOTP **nunca** sai do aparelho). Além do QR, sempre a chave para digitar.
- **Aviso obrigatório** (texto fixo, da demo): perder o acesso ao autenticador pode significar perder a conta, sem volta, mesmo sabendo a senha; a pessoa confirma que leu antes de o MFA ficar ligado.
- **Janela ou tela cheia nova entra em `camadasAbertas()` e `fecharCamadaTopo()` de `web/js/app.js`** (botão voltar; guardrails).
- **Carga:** a seção **não lê nada do nosso servidor** e só toca o Cognito quando a pessoa a abre; `web/carga.test.js` continua passando (a subida não muda).
- Testes (`node web/<nome>.test.js`, o CI roda todos): a sessão de curta duração não grava token em nenhum armazenamento; o pedido do segundo login contém o escopo novo e o login normal **não**; código errado mostra o erro e não liga; só liga depois da confirmação do aviso; desligar pede o segundo login; nenhuma chamada leva o access token em URL. **Quebrar o código de propósito uma vez** para ver cada teste falhar.
- Backend: **nenhuma mudança**. Confirmar que `arquitetura_test.go` e `go test ./...` seguem verdes.

### Fase 4: verificação no `dev` e fechamento
- App do `dev` instalado com o front novo (`scripts/instalar-app-android.ps1 -Ambiente dev`) e navegador: ligar, sair, entrar com código, desligar; no Android, **o segundo login com o escopo novo na aba do sistema** (único ponto ainda não testado) e os dois casos de voltar.
- "Esqueci a senha" com o MFA ligado (ainda não provado, o intent registra).
- PWA instalado.
- Atualizar `.ai/security.md`, `BACKLOG.md` (item 49, Fase 1), `.ai/padrao-de-telas.md` (a tela e a linha em "Armadilhas já encontradas", se aparecer alguma) e este resumo de `guardrails.md`; a demo aprovada vira a padrão de Listas.
- Limpeza do `dev`: nada fora do `template.yaml` pode sobrar (o `ALLOW_USER_PASSWORD_AUTH` dos testes já saiu).

## Premissas (obrigatório antes da aprovação do roteiro)

Regras que só funcionam por causa de outra coisa, e o que acontece se ela deixar de valer:

| # | Regra | Depende de | Se a outra deixar de valer |
|---|---|---|---|
| P1 | A tela chama as APIs de MFA com o access token | O cliente **permitir** o escopo `aws.cognito.signin.user.admin` (Fase 2) | Sem o escopo no cliente, o Cognito recusa o segundo login; a tela mostra erro e não liga nada. Teste: pedir o escopo num cliente sem ele |
| P2 | "Desligar exige confirmar quem você é de novo" | O Hosted UI **forçar a senha** no segundo login (`prompt=login` ou `max_age`, **não provado**) | Se ele reaproveitar a sessão por cookie, o segundo login passa sem senha nem código e desligar o MFA fica sem a garantia; precisa de outra proteção ou da decisão do dono |
| P3 | O token com o escopo vive só em memória | O front **não** o repassar a nenhum outro módulo, log ou armazenamento | Se algum módulo o guardar (por exemplo, o `Cofre` do Android), o ganho de pedir o escopo só no momento some. Teste de que nenhum armazenamento o recebe |
| P4 | O aviso diz "sem volta" | **Não haver recuperação** nesta entrega (intent) | Quando a fase das chaves de recuperação chegar, o aviso fica **errado** e deve ser reescrito junto com ela; registrado aqui para não esquecer |
| P5 | "Opcional" faz o login de quem não ligou ficar só com a senha | O pool em `OPTIONAL` | Se alguém passar o pool a `ON` (obrigatório), quem não cadastrou autenticador **não entra** e a tela do app não é alcançada; o app não deve pedir `ON` sem outra entrega de cadastro obrigatório |
| P6 | O front não guarda o access token hoje (`sessao.js`) | A sessão normal seguir pedindo só `openid email` | Se alguém passar a pedir o escopo no login normal, o token mais poderoso passa a ficar à mão do XSS; teste de que o login normal não o pede |
| P7 | O backend não muda | O backend validar só o ID token (`auth/token.go`) | Se alguém passar a aceitar o access token (o `security.md` proíbe sem decisão do dono), o escopo novo estaria na API |
| P8 | A seção do MFA fica visível a dono e membro | O front não decidir por papel; o Cognito não tem papel | Nada no servidor depende disso; se a demo escolher esconder do membro, isso é só conforto e não é segurança |

**Revisão independente: pendente.** O guardrails manda um agente sem o contexto de quem escreveu ler o `spec.md` **e o código** e responder por escrito: *"que regra desta feature só funciona por causa de outra, e o que acontece se essa outra deixar de valer?"*. A resposta entra aqui, mesmo que seja "nenhuma", e o dono a vê com o roteiro.
O `spec.md` ainda não existe; a revisão vem depois dele. **Quem faz** (subagente ou outra sessão) é decisão do dono, que registrou em 2026-11-13 que dispensava subagente enquanto não existisse o agente, e em 2026-11-18 disse que ele existe agora.

## Riscos

| Risco | Resposta |
|---|---|
| O changeset mostrar substituição do pool ou do cliente | Parar; o deploy já barra; voltar ao dono antes de seguir |
| O segundo login reaproveitar a sessão do Hosted UI e não pedir nada | Verificar `prompt=login` no `dev` na Fase 2; sem ele, decidir com o dono |
| Token com escopo amplo vazar | Só em memória, só durante a seção, nunca em log nem armazenamento (P3, P6) |
| Pessoa liga o MFA sem entender o risco e perde o autenticador | Aviso obrigatório com confirmação; sem recuperação nesta entrega (intent) |
| QR code vaza a chave | Gerado no aparelho, sem serviço externo, com a chave também em texto para digitar |
| Dependência nova no front (QR) | Pequena, embutida, sem CDN; conferida antes de entrar |

## Ordem recomendada

Fase 1 (demo, sem código) → aprovação do dono → Fase 2 (infra, vai ao `dev`) → Fase 3 (front) → Fase 4 (verificação e documentos). A Fase 2 pode andar em paralelo à 1, porque o `template.yaml` não depende da aparência da tela.

## Decisões que o plan fixou e o dono pode contestar

1. (**Confirmada pelo dono em 2026-11-19.**) O login normal **não** muda; o escopo novo é pedido só num segundo login ao abrir a seção do MFA.
2. As APIs de MFA vão do front ao Cognito, **sem passar pela Lambda**; o backend e o `authz.go` não mudam.
3. Nenhum fluxo de senha no cliente; nenhum dado novo no DynamoDB.
4. O QR code é gerado no aparelho, sem serviço externo.
5. A tela é uma seção própria de Listas (a "Segurança da conta"), diferente da Segurança do aparelho.
