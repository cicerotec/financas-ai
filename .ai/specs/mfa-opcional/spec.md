# Spec: MFA opcional por aplicativo autenticador

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id (`M-T<n>`), para o PR dizer exatamente o que cobriu.
Quando este texto e o do intent divergirem, vale o plan. A tabela "Premissas" do plan tem uma regra aqui para cada linha (P1 a P11), indicada ao lado da regra.

**Estado:** rascunho para aprovação (2026-11-19). Nada implementado. **A revisão independente das premissas ainda não foi feita** e é pré-requisito da aprovação (guardrails).
Itens marcados **[PENDENTE]** dependem de uma verificação no `dev` ou de uma decisão do dono; a fase que os usa não começa sem eles.

## Regras

### Infraestrutura (`template.yaml`, Fase 2)

- **R1.** O `UserPool` define `MfaConfiguration: OPTIONAL` e `EnabledMfas: [SOFTWARE_TOKEN_MFA]`. Não há SMS nem MFA por e-mail. Nenhuma outra propriedade do pool muda.
- **R2.** O `UserPoolClient` passa a **permitir** `aws.cognito.signin.user.admin` em `AllowedOAuthScopes`, junto de `openid` e `email`. `ExplicitAuthFlows` continua `[ALLOW_REFRESH_TOKEN_AUTH]`, sem nenhum fluxo de senha. `CallbackURLs`, `LogoutURLs` e o resto do cliente não mudam. (P1)
- **R3.** O changeset do deploy no `dev` mostra só **modificação de propriedade**, sem `Replacement: True` no pool, no cliente nem no domínio. Se mostrar substituição, **para-se** e a decisão volta ao dono; o deploy já barra substituição do login.
- **R4.** A **validação do token não muda**: `auth/token.go` segue validando só o ID token, e nenhuma rota existente muda de ação. O backend só ganha as duas rotas da lista de códigos (R25 a R32), cada uma com ação própria em `auth/authz.go`; qualquer rota sem ação continua `404` e o que não está liberado continua negado. (P7)
- **R5.** Antes do merge da Fase 2, confere-se que a role do Actions de cada ambiente (`infra/github-oidc.yaml`) tem as permissões que o deploy dessas propriedades exige; se faltar, atualiza-se **antes** do merge.
- **R6.** Nem o template nem o app definem `MfaConfiguration: ON`. (P5)

### Sessão (`web/js/sessao.js`, Fase 4)

- **R7.** O **login normal** continua pedindo exatamente `openid email`. Nenhum caminho do login normal pede `aws.cognito.signin.user.admin`. (P6)
- **R8.** A seção do MFA obtém o access token por um **segundo login**, que pede `openid email aws.cognito.signin.user.admin`, com PKCE e `state` novos a cada vez. O segundo login **exige que a pessoa prove de novo quem é** (senha e, se tiver MFA, o código). **[PENDENTE]** O parâmetro que força isso no Hosted UI (`prompt=login` ou `max_age`) não foi provado: se o Cognito reaproveitar a sessão por cookie, esta regra não pode ser cumprida e a decisão volta ao dono. (P2)
- **R9.** O access token do segundo login vive **só em memória**: não vai a `localStorage`, `sessionStorage`, ao `Cofre` do Android nem a nenhum outro módulo. É descartado ao fechar a seção, ao sair da conta e ao expirar. (P3)
- **R10.** O segundo login **não substitui a sessão do app**: o ID token e o refresh token do login normal continuam como estavam, e os tokens do segundo login (inclusive ID e refresh) são descartados, exceto o access token de R9.
- **R11.** Cancelar o segundo login (voltar, fechar a aba do sistema, erro do Cognito) deixa o app **logado, como estava**, e a seção volta ao estado de antes, sem MFA alterado.
- **R12.** No navegador e no PWA, o segundo login sai da página (como o login normal). Antes de sair, guarda-se **só a intenção** ("abrir a seção do MFA"), sem token; ao voltar, a seção abre. No app Android o segundo login abre na aba do sistema, como o login de hoje.
- **R13.** Nenhum log escreve token, `code`, `verifier` nem a URL de retorno (R18 do `app-android`).

### Chamadas ao Cognito (módulo novo do front, Fase 4)

- **R14.** As APIs de MFA são chamadas **direto ao Cognito** (`https://cognito-idp.<região>.amazonaws.com/`, `POST`, `Content-Type: application/x-amz-json-1.1`, `X-Amz-Target`), **sem passar pela Lambda**. O access token vai **no corpo**, nunca na URL, em cabeçalho extra nem em log.
- **R15.** O front lê o estado do MFA por `GetUser` e **só exibe o que ele devolve** (`UserMFASettingList` e `PreferredMfaSetting`). Não guarda esse estado, não o infere e não decide por ele. Não exibe nem guarda e-mail nem outros atributos que o `GetUser` traga. (P8)
- **R16.** **Ligar** segue esta ordem, e cada passo só começa depois do anterior: (1) `AssociateSoftwareToken`; (2) a tela mostra o QR code e a chave para digitar; (3) a pessoa digita o código de 6 dígitos e `VerifySoftwareToken` devolve `SUCCESS`; (4) o front pede a lista à rota `POST /me/mfa/codigos` (R25); (5) a tela mostra a **lista de códigos com a frase em destaque** (R20), e a pessoa confirma que leu e guardou; (6) só então `SetUserMFAPreference` com `Enabled: true` e `PreferredMfa: true`; (7) o estado é relido por `GetUser` e exibido. O MFA **nunca** é ligado sem `SUCCESS`, sem a lista gerada nem sem a confirmação de leitura.
- **R17.** **Desligar** pede confirmação e usa `SetUserMFAPreference` com `Enabled: false` e `PreferredMfa: false`; depois relê o estado por `GetUser`. Depois de desligar, mostra o aviso de 6 s do padrão do app (sem Desfazer: o Desfazer não existe aqui).
- **R18.** Erros mostrados **na própria janela**: código errado (a janela continua aberta e deixa tentar de novo); bloqueio por excesso de tentativas, que o Cognito aplica depois de 5 erros (mensagem de aguardar, sem contar tentativas no front); sem internet (nada muda); recusa do Cognito por falta de permissão no token (mensagem de que não foi possível, sem detalhe técnico). (P1)
- **R19.** A chave do TOTP e o QR code são gerados **no aparelho, em memória**: nunca em armazenamento, log, URL, serviço externo nem área de transferência automática. Somem ao fechar a janela, ao trocar de tela e ao sair da conta. A chave sempre aparece também em texto, para digitar. O QR vem de código embutido em `web/`, **sem CDN**; a biblioteca ou o código próprio é conferido antes de entrar.

### Tela (Listas, Fases 1 e 4)

- **R20.** A tela da lista mostra, em destaque e como texto fixo igual ao da demo aprovada, a frase **"SE PERDER A LISTA, PERDE O ACESSO."**, e diz que a lista deve ser guardada num lugar seguro, **separado do autenticador**, e que perder o autenticador **e** a lista é perder a conta, **sem volta**, mesmo sabendo a senha. A pessoa confirma que leu. A lista aparece **uma única vez** e **não é guardada no aparelho**: some ao fechar a janela, ao trocar de tela e ao sair da conta (R19). (P4: se outro caminho de recuperação for acrescentado, esta frase é reescrita junto com ele.)
- **R21.** A seção **Segurança da conta** aparece a dono e membro, no navegador, no PWA e no app, e é **separada** da Segurança do aparelho (trava e apps recentes, só no app, valem só no aparelho). Esconder para o membro, se a demo decidir, é só conforto. (P8)
- **R22.** Janela ou tela cheia nova entra em `camadasAbertas()` e `fecharCamadaTopo()` de `web/js/app.js`. O botão voltar fecha a camada de cima. O segundo login **não** é camada.
- **R23.** A seção **não lê nada do servidor nosso ao abrir**: a subida do app não muda, e `web/carga.test.js` continua passando. O servidor só é chamado quando a pessoa **gera a lista** (R25); a tela de recuperação chama só a rota pública (R27).
- **R24.** A tela implementada fica **igual à demo aprovada**; desvio se avisa antes do PR. Vale o padrão de telas: tokens do app (claro e escuro), linhas de pelo menos 56 px, datas com ano, janelas de baixo, seletores nativos, pior caso medido (celular ≈ 390 px e janela ≥ 1280 px). Sair da conta apaga o estado da seção (R9, R19).

### Lista de códigos de recuperação (backend e tela de recuperação, Fases 3 e 4)

**Pré-requisito:** o dono discute o dado novo no DynamoDB (hashes e controle de tentativas) **antes** de qualquer código; nada é criado antes disso (guardrails).

- **R25.** `POST /me/mfa/codigos` é uma rota **autenticada**, fora de `/spaces/{sid}`, com ação própria em `auth/authz.go` liberada a qualquer pessoa autenticada. Gera a lista, guarda **só os hashes** (hash lento) e devolve os códigos em texto **uma única vez**. Nunca devolve de novo uma lista já gerada. Gerar de novo **invalida** a anterior.
- **R26.** Cada código é aleatório (gerador criptográfico), tem comprimento fixo definido no spec da Fase 3, usa um alfabeto sem caracteres ambíguos e vale **uma vez**. O servidor nunca grava nem loga o código em texto.
- **R27.** `POST /mfa/recuperar` é uma rota **pública** (sem token), com ação própria em `authz.go` liberada **sem token**; nenhuma outra rota ganha essa liberação. Recebe o e-mail e **um** código. Se o código vale para aquela pessoa, o marca como usado, **desliga o MFA dela** no Cognito (`AdminSetUserMFAPreference`) e invalida o resto da lista. (P10)
- **R28.** A rota pública responde **exatamente igual** (mesmo status, corpo e tempo aproximado) para e-mail sem conta, pessoa sem MFA, código errado e código já usado. O sucesso é o único caso diferente. (P11)
- **R29.** A rota pública tem **limite de tentativas por pessoa e por origem**; ao estourar, responde o mesmo erro genérico e não confere mais códigos até o limite expirar. O limite é do servidor (a Function URL não tem WAF). (P11)
- **R30.** Um código só vale para o usuário (`sub`) a que foi gerado; o código de uma pessoa **nunca** desliga o MFA de outra, mesmo com e-mail parecido ou alias. (P10)
- **R31.** A rota pública **só desliga o MFA**: nunca cria sessão, nunca devolve token e nunca altera a senha. A entrada continua exigindo a senha pelo login do Cognito. (P9)
- **R32.** A Lambda ganha **uma** permissão nova no Cognito, `AdminSetUserMFAPreference` (mais a leitura mínima para achar a pessoa pelo e-mail), **sobre o próprio pool**; as roles do Actions são atualizadas **antes** do merge. Nenhum log escreve código, e-mail nem corpo.
- **R33.** A **tela de recuperação** (sem login) pede e-mail e um código, chama `POST /mfa/recuperar` e mostra só o que o servidor devolve: no sucesso, "MFA desligado; entre com e-mail e senha"; no erro, uma mensagem genérica na própria tela. Ela fica **fora do Hosted UI**: na abertura do app Android, ao lado do "Entrar", e por um endereço próprio no navegador e no PWA. O front **não decide** nada sobre o código (não confere, não conta tentativas). Segue a demo aprovada (R24) e o botão voltar (R22).
## Testes

Cada um falha antes da mudança. Os de front são `web/<nome>.test.js` (o CI roda todos); os de infra são conferidos por `sam validate --lint` e pelo changeset.

| Id | O que prova |
|---|---|
| M-T1 | O `template.yaml` tem `MfaConfiguration: OPTIONAL` e `EnabledMfas: [SOFTWARE_TOKEN_MFA]`, sem SMS (R1) |
| M-T2 | O cliente **permite** o escopo novo, **não** lista fluxo de senha, e não perde as URLs de retorno (R2) |
| M-T3 | A validação de `token.go` não mudou e nenhuma rota existente trocou de ação; as duas rotas novas têm ação própria em `authz.go`; `go vet ./...`, `go test ./...` e `arquitetura_test.go` passam (R4) |
| M-T4 | O pedido de login normal tem o escopo `openid email` e **não** tem `aws.cognito.signin.user.admin` (R7) |
| M-T5 | O pedido do segundo login tem o escopo novo, PKCE e `state` novos e o parâmetro de reautenticação confirmado no `dev` (R8) |
| M-T6 | O access token do segundo login não é gravado em `localStorage`, `sessionStorage` nem no `Cofre`, e some ao fechar a seção e ao sair da conta (R9) |
| M-T7 | Depois do segundo login, o ID token e o refresh do login normal continuam os mesmos; os tokens do segundo são descartados (R10) |
| M-T8 | Cancelar o segundo login deixa o app logado e o MFA sem mudança (R11) |
| M-T9 | No navegador, sair para o segundo login guarda só a intenção, sem token, e a seção reabre ao voltar (R12) |
| M-T10 | Nenhuma linha de log contém token, `code`, `verifier` nem URL de retorno (R13) |
| M-T11 | Toda chamada ao Cognito leva o access token no corpo e nunca na URL nem em cabeçalho extra (R14) |
| M-T12 | O front exibe o estado devolvido por `GetUser` e não o guarda; atributos como e-mail não aparecem nem são guardados (R15) |
| M-T13 | Ligar segue a ordem de R16; sem `Verify` igual a `SUCCESS`, sem a lista gerada ou sem a confirmação de leitura, `SetUserMFAPreference` **não** é chamado |
| M-T14 | Desligar chama `SetUserMFAPreference` com `Enabled: false` e relê por `GetUser`, depois de confirmar (R17) |
| M-T15 | Código errado, bloqueio por tentativas, falta de internet e recusa por falta de permissão mostram a mensagem **na janela** e não alteram o estado (R18) |
| M-T16 | A chave e o QR não vão a armazenamento, log, URL, serviço externo nem área de transferência, e são limpos ao fechar (R19) |
| M-T17 | A tela da lista mostra a frase "SE PERDER A LISTA, PERDE O ACESSO." e o texto da demo aprovada; a lista não é gravada no aparelho e some ao fechar (R20) |
| M-T18 | A janela do MFA entra em `camadasAbertas()` e `fecharCamadaTopo()`; o voltar a fecha; o segundo login não é camada (R22) |
| M-T19 | A subida e a abertura da seção não leem nada do servidor nosso (`carga.test.js`) (R23) |
| M-T20 | Nem o template nem o front definem `MfaConfiguration: ON` (R6) |
| M-T21 | `POST /me/mfa/codigos` sem token é `401`; com token devolve a lista uma vez e grava só hashes; gerar de novo invalida a anterior (R25, R26) |
| M-T22 | Um código funciona uma vez: o segundo uso do mesmo código responde o erro genérico (R26, R27) |
| M-T23 | E-mail sem conta, pessoa sem MFA, código errado e código já usado produzem a **mesma** resposta (R28) |
| M-T24 | Depois do limite de tentativas por pessoa e por origem, nem o código certo é conferido até o limite expirar (R29) |
| M-T25 | O código de uma pessoa não desliga o MFA de outra, nem com e-mail parecido ou alias (R30) |
| M-T26 | A rota pública, no sucesso, só chama `AdminSetUserMFAPreference` e não devolve token nem cria sessão nem altera a senha (R31) |
| M-T27 | `POST /mfa/recuperar` é a **única** rota liberada sem token; qualquer outra sem token segue `401` (R27) |
| M-T28 | Nenhum log, resposta de erro ou teste contém código, e-mail ou corpo (R26, R32) |
| M-T29 | A tela de recuperação mostra só o que o servidor devolve, sem conferir código nem contar tentativas, e leva a pessoa a entrar com e-mail e senha (R33) |
| M-T30 | A permissão nova da Lambda é só `AdminSetUserMFAPreference` (mais a leitura mínima) sobre o próprio pool, e as roles do Actions foram atualizadas (R32) |

**Quebrar de propósito:** (a) fazer o login normal pedir o escopo novo tem de fazer M-T4 falhar; (b) gravar o access token em `localStorage` tem de fazer M-T6 falhar; (c) chamar `SetUserMFAPreference` antes do `Verify` tem de fazer M-T13 falhar.

## Verificação no `dev`

1. Deploy da Fase 2 no `financas-dev` e **changeset conferido a olho**: só modificação de propriedade (R3). O pool e o cliente do `dev` foram **devolvidos ao original** em 2026-11-19; o primeiro deploy deve mostrar `MfaConfiguration` e o escopo como mudança.
2. **P2:** com uma sessão do Hosted UI ativa, o segundo login pede senha de novo? Testar `prompt=login` e `max_age`. Sem isso, decidir com o dono antes da Fase 3 (R8).
3. App do `dev` instalado com o front novo (`scripts/instalar-app-android.ps1 -Ambiente dev`) e o navegador: ligar com o autenticador do celular, sair, entrar pedindo o código, desligar.
4. **Android:** o segundo login com o escopo novo na aba do sistema, e os dois casos de voltar (antes da senha e antes do código).
5. **Recuperação:** com o MFA ligado e sem o autenticador, um código da lista, na tela de recuperação, desliga o MFA; o mesmo código não funciona de novo; a pessoa entra com e-mail e senha e liga de novo com uma lista nova. No navegador, no PWA e no app Android.
6. "Esqueci a senha" com o MFA ligado; PWA instalado.
7. O usuário de teste termina sem MFA e nada do `dev` fica fora do `template.yaml`.

## Fora do escopo

Recuperação por e-mail, por prazo ou por perguntas de segurança, Managed login e branding, passkey, SMS, MFA por e-mail, MFA obrigatório, login social e qualquer caminho para quem perdeu o autenticador **e** a lista (ver `intent.md`).
