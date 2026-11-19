# Spec: MFA opcional por aplicativo autenticador

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id (`M-T<n>`), para o PR dizer exatamente o que cobriu.
Quando este texto e o do intent divergirem, vale o plan. A tabela "Premissas" do plan tem uma regra aqui para cada linha (P1 a P15), indicada ao lado da regra.

**Estado:** rascunho para aprovação (2026-11-19). Nada implementado. **A revisão independente das premissas foi feita em 2026-11-19** ([`../../bmad/arquitetura-mfa-recuperacao.md`](../../bmad/arquitetura-mfa-recuperacao.md)); este spec absorve os achados A1 e A5 (R4, R16, R25, R25a, R27 a R29, R34 e os testes M-T13, M-T21, M-T22 e M-T27). O resto da revisão, [`revisao-mfa-opcional-pendencias.md`](../../bmad/revisao-mfa-opcional-pendencias.md), está aplicado nas Partes 1 e 2 (R10, R12, R17, R18, R23, R25, R28, R29, R32, R33 e os testes e passos de verificação correspondentes); **a Parte 3 (D1 a D6) é decisão do dono** e está no `plan.md`, "Decisões do dono pendentes": nenhuma regra deste spec depende de ter sido decidida, exceto o que cada uma disser quando for. **Uma nova revisão é pedida depois desta atualização** (P12 a P15, R25a e R34 são novas). Onde o relatório divergir de `guardrails.md`, `arquitetura.md` ou `decisoes.md`, a divergência vai ao dono antes de qualquer código.
Itens marcados **[PENDENTE]** dependem de uma verificação no `dev` ou de uma decisão do dono; a fase que os usa não começa sem eles.

## Regras

### Infraestrutura (`template.yaml`, Fase 2)

- **R1.** O `UserPool` define `MfaConfiguration: OPTIONAL` e `EnabledMfas: [SOFTWARE_TOKEN_MFA]`. Não há SMS nem MFA por e-mail. Nenhuma outra propriedade do pool muda.
- **R2.** O `UserPoolClient` passa a **permitir** `aws.cognito.signin.user.admin` em `AllowedOAuthScopes`, junto de `openid` e `email`. `ExplicitAuthFlows` continua `[ALLOW_REFRESH_TOKEN_AUTH]`, sem nenhum fluxo de senha. `CallbackURLs`, `LogoutURLs` e o resto do cliente não mudam. (P1)
- **R2a.** O `UserPoolDomain` define `ManagedLoginVersion: 2` e existe um estilo de marca (`AWS::Cognito::ManagedLoginBranding`) ligado ao cliente, com o visual aprovado na demo (logo, cores, fundo, claro e escuro). O nome do domínio e o endereço de login **não mudam**; o app, o fluxo `code` com PKCE e as URLs de retorno continuam os mesmos.
- **R3.** O changeset do deploy no `dev` mostra só **modificação de propriedade**, sem `Replacement: True` no pool, no cliente nem no domínio. Se mostrar substituição, **para-se** e a decisão volta ao dono; o deploy já barra substituição do login.
- **R4.** **A validação do token não muda**: `auth/token.go` e `VerificarToken` ficam intactos e o backend segue validando só o ID token. O **roteador ganha uma etapa anterior** que só reconhece `POST /mfa/recuperar` (R27); toda outra rota continua passando por `VerificarToken` (`401` se falhar), e nenhuma rota existente muda de ação. O backend ganha as três rotas da lista de códigos (R25, R25a e R27), cada uma com ação própria em `auth/authz.go`; qualquer rota sem ação continua `404` e o que não está liberado continua negado. (P7, P12)
- **R5.** Antes do merge da Fase 2, confere-se que a role do Actions de cada ambiente (`infra/github-oidc.yaml`) tem as permissões que o deploy dessas propriedades exige; se faltar, atualiza-se **antes** do merge.
- **R6.** Nem o template nem o app definem `MfaConfiguration: ON`. (P5)

### Sessão (`web/js/sessao.js`, Fase 4)

- **R7.** O **login normal** continua pedindo exatamente `openid email`. Nenhum caminho do login normal pede `aws.cognito.signin.user.admin`. (P6)
- **R8.** A seção do MFA obtém o access token por um **segundo login**, que pede `openid email aws.cognito.signin.user.admin`, com PKCE e `state` novos a cada vez. O segundo login **exige que a pessoa prove de novo quem é** (senha e, se tiver MFA, o código). **[PENDENTE]** O parâmetro que força isso no Managed login (`prompt=login` ou `max_age`) não foi provado: se o Cognito reaproveitar a sessão por cookie, esta regra não pode ser cumprida e a decisão volta ao dono. (P2)
- **R9.** O access token do segundo login vive **só em memória**: não vai a `localStorage`, `sessionStorage`, ao `Cofre` do Android nem a nenhum outro módulo. É descartado ao fechar a seção, ao sair da conta e ao expirar. (P3)
- **R10.** O segundo login **não substitui a sessão do app**: o ID token e o refresh token do login normal continuam como estavam, e os tokens do segundo login (inclusive ID e refresh) são descartados, exceto o access token de R9. Hoje o código de sessão não separa os dois (no navegador, `token()` trata todo `?code=` como login normal e `salvarAuth` grava o ID token em `localStorage`, com um só `financas.pkce`; no app, `tratarRetorno` grava o `refresh` no `Cofre` e chama `env.recarregar()`, o que apagaria o access token em memória). **Regra:** o PKCE pendente leva um **tipo** (`normal` ou `mfa`), e o retorno do tipo `mfa` segue um **ramo próprio**: troca o `code`, entrega **só o access token** à seção, **não** chama `salvarAuth`, **não** grava `refresh` e **não** recarrega a página; o retorno `normal` continua como hoje. **O refresh token do segundo login continua válido por 30 dias no Cognito** (descartar no cliente não o invalida): logo depois de trocar o `code`, o front chama `POST /oauth2/revoke` com esse refresh (cliente público, sem segredo); o access token já emitido deve continuar valendo até a seção fechar. **[PENDENTE]** Confirmar no `dev` (Verificação, passo 9).
- **R11.** Cancelar o segundo login (voltar, fechar a aba do sistema, erro do Cognito) deixa o app **logado, como estava**, e a seção volta ao estado de antes, sem MFA alterado.
- **R12.** No navegador e no PWA, o segundo login sai da página (como o login normal). Antes de sair, guarda-se **só a intenção** ("abrir a seção do MFA"), sem token; ao voltar, a seção abre. **A intenção tem TTL curto** (10 minutos) e é **apagada no retorno com erro ou cancelamento**: se o segundo login for cancelado no Cognito (voltar, erro) e a pessoa abrir o app depois, a seção **não** reabre sozinha. No app Android o segundo login abre na aba do sistema, como o login de hoje; o retorno do tipo `mfa` segue o ramo de R10.
- **R13.** Nenhum log escreve token, `code`, `verifier` nem a URL de retorno (R18 do `app-android`).

### Chamadas ao Cognito (módulo novo do front, Fase 4)

- **R14.** As APIs de MFA são chamadas **direto ao Cognito** (`https://cognito-idp.<região>.amazonaws.com/`, `POST`, `Content-Type: application/x-amz-json-1.1`, `X-Amz-Target`), **sem passar pela Lambda**. O access token vai **no corpo**, nunca na URL, em cabeçalho extra nem em log.
- **R15.** O front lê o estado do MFA por `GetUser` e **só exibe o que ele devolve** (`UserMFASettingList` e `PreferredMfaSetting`). Não guarda esse estado, não o infere e não decide por ele. Não exibe nem guarda e-mail nem outros atributos que o `GetUser` traga. (P8)
- **R16.** **Ligar** segue esta ordem, e cada passo só começa depois do anterior: (1) `AssociateSoftwareToken`; (2) a tela mostra o QR code e a chave para digitar; (3) a pessoa digita o código de 6 dígitos e `VerifySoftwareToken` devolve `SUCCESS`; (4) o front pede a lista a `POST /me/mfa/codigos` (R25), que a cria **pendente** e devolve `{listaId, codigos}`; (5) a tela mostra a **lista de códigos com a frase em destaque** (R20) e a pessoa toca em **"Guardei a lista"**; (6) **só então** o front chama `POST /me/mfa/codigos/confirmar` com o `listaId` (R25a); (7) **só com o `200` do confirmar**, `SetUserMFAPreference` com `Enabled: true` e `PreferredMfa: true`; (8) o estado é relido por `GetUser` e exibido. **Regerar com o MFA já ligado** usa só os passos 4 a 6. O confirmar **nunca** é chamado antes do toque, e `SetUserMFAPreference` **nunca** antes do `200`. Se a resposta do confirmar se perder, a tela mantém a lista aberta e repete o confirmar com o mesmo `listaId` (é idempotente). Se o passo 7 falhar, a lista ativa fica e o MFA continua desligado: a tela relê o estado e oferece repetir. O servidor **não** garante que MFA ligado implique lista ativa (ele não vê o Cognito); isso é imposto só por esta ordem. (P13)
- **R17.** **Desligar** pede confirmação e usa `SetUserMFAPreference` com `Enabled: false` e `PreferredMfa: false`; depois relê o estado por `GetUser`. Depois de desligar, mostra o aviso de 6 s do padrão do app (sem Desfazer: o Desfazer não existe aqui). **Desligar em Listas não toca a lista no servidor:** a lista ativa **fica** (inofensiva, porque a rota pública responde genérico a quem está sem MFA) e é substituída na próxima vez que a pessoa ligar o MFA (R16). Não há rota de limpeza.
- **R18.** Erros mostrados **na própria janela**: código errado (a janela continua aberta e deixa tentar de novo); bloqueio por excesso de tentativas (mensagem de aguardar quando o Cognito recusar por excesso de tentativas, sem contar tentativas no front; o número de erros que dispara o bloqueio **não** é afirmado aqui, porque não foi provado no `VerifySoftwareToken`: Verificação, passo 10); sem internet (nada muda); recusa do Cognito por falta de permissão no token (mensagem de que não foi possível, sem detalhe técnico); e `409 lista_substituida` do confirmar (outra aba gerou uma lista nova ou a pendente expirou: a mensagem diz para gerar de novo, e a lista ativa continua valendo). (P1)
- **R19.** A chave do TOTP e o QR code são gerados **no aparelho, em memória**: nunca em armazenamento, log, URL, serviço externo nem área de transferência automática. Somem ao fechar a janela, ao trocar de tela e ao sair da conta. A chave sempre aparece também em texto, para digitar. O QR vem de código embutido em `web/`, **sem CDN**; a biblioteca ou o código próprio é conferido antes de entrar.

### Tela (Listas, Fases 1 e 4)

- **R20.** A tela da lista mostra, em destaque e como texto fixo igual ao da demo aprovada, a frase **"SE PERDER A LISTA, PERDE O ACESSO."**, e diz que a lista deve ser guardada num lugar seguro, **separado do autenticador**, e que perder o autenticador **e** a lista é perder a conta, **sem volta**, mesmo sabendo a senha. A pessoa confirma que leu. A lista aparece **uma única vez** e **não é guardada no aparelho**: some ao fechar a janela, ao trocar de tela e ao sair da conta (R19). (P4: se outro caminho de recuperação for acrescentado, esta frase é reescrita junto com ele.)
- **R21.** A seção **Segurança da conta** aparece a dono e membro, no navegador, no PWA e no app, e é **separada** da Segurança do aparelho (trava e apps recentes, só no app, valem só no aparelho). Esconder para o membro, se a demo decidir, é só conforto. (P8)
- **R22.** Janela ou tela cheia nova entra em `camadasAbertas()` e `fecharCamadaTopo()` de `web/js/app.js`. O botão voltar fecha a camada de cima. O segundo login **não** é camada.
- **R23.** A seção **não lê nada do servidor nosso ao abrir**: a subida do app não muda, e `web/carga.test.js` continua passando. O servidor só é chamado quando a pessoa **gera ou confirma a lista** (R25 e R25a); a tela de recuperação chama só a rota pública (R27).
- **R24.** A tela implementada fica **igual à demo aprovada**; desvio se avisa antes do PR. Vale o padrão de telas: tokens do app (claro e escuro), linhas de pelo menos 56 px, datas com ano, janelas de baixo, seletores nativos, pior caso medido (celular ≈ 390 px e janela ≥ 1280 px). Sair da conta apaga o estado da seção (R9, R19).

### Lista de códigos de recuperação (backend e tela de recuperação, Fases 3 e 4)

**Pré-requisito:** o dono discute o dado novo no DynamoDB (hashes e controle de tentativas) **antes** de qualquer código; nada é criado antes disso (guardrails).

- **R25.** `POST /me/mfa/codigos` é uma rota **autenticada**, fora de `/spaces/{sid}`, com ação própria em `auth/authz.go` liberada a qualquer pessoa autenticada, sempre com o `sub` do token (nunca do corpo; corpo vazio aceito). **Gera uma lista pendente**, guarda **só os hashes** e devolve `{listaId, codigos}` **uma única vez**, substituindo a pendente anterior se houver. **Gerar não toca na lista ativa e não a invalida.** **No roteador:** `RotaSemEspaco` passa a reconhecer `POST /me/mfa/codigos` e `POST /me/mfa/codigos/confirmar` (ações próprias, com teste em `authz_test.go`); o ramo de `processar` (`backend/roteador.go`, hoje um `if` só para `AcaoSairPush`, que lê o corpo como JSON obrigatório e devolve `json invalido` se vazio) vira um `switch` por ação; **gerar aceita corpo vazio** e **confirmar exige JSON** com `listaId` (`400` sem corpo). Por pessoa há no máximo uma lista ativa e uma pendente; a pendente expira sozinha (prazo definido com o dono). (P13, P15)
- **R25a.** `POST /me/mfa/codigos/confirmar` é uma rota **autenticada**, fora de `/spaces/{sid}`, com ação própria `mfa.confirmar` liberada a qualquer pessoa autenticada, sempre com o `sub` do token. Corpo `{"listaId": "..."}`. Promove a pendente a ativa **e** apaga a ativa anterior numa **escrita condicional** (a pendente existe, tem aquele `listaId` e não expirou). É **idempotente**: confirmar de novo o `listaId` que já é o da ativa devolve `200`. Pendente substituída por outra geração ou expirada devolve `409` com `lista_substituida`, e a ativa continua como estava. (P13, P15)
- **R26.** Cada código é aleatório (gerador criptográfico), tem comprimento fixo definido na conversa com o dono (com 100 bits ou mais, **HMAC-SHA256 com pepper no SSM** `/financas/<ambiente>/mfa-pepper`; com código curto, hash lento), usa um alfabeto sem caracteres ambíguos e vale **uma vez**. O servidor nunca grava nem loga o código em texto. Só a lista **ativa** vale na rota pública: um código da pendente não recupera nada. (P15)
- **R27.** `POST /mfa/recuperar` é uma rota **pública**, reconhecida por **`auth.RotaPublica(metodo, p)`**, que devolve `true` **só** para `POST` e o caminho exato `["mfa","recuperar"]` (a ação `mfa.recuperar` **não** é listada em `Permitido` para nenhum papel). Em `atender`, o caminho e o método são quebrados **antes** de `VerificarToken`: se for a rota pública, o roteador grava o rótulo e a entrega ao controller **sem ler o cabeçalho `authorization`**; um token inválido nela não muda a resposta. Recebe o e-mail e **um** código. Se o código vale para aquela pessoa e é da lista ativa, o marca como usado numa **escrita condicional** (`usado` ainda falso, lista ainda a ativa), **desliga o MFA dela** no Cognito (`AdminSetUserMFAPreference`) e, no sucesso, apaga a lista ativa inteira. **Se o Cognito falhar, a marca de usado é desfeita** (condicional ao mesmo `listaId`) e a resposta é a genérica: um código não é queimado por falha do Cognito. Dois usos simultâneos do mesmo código: só um passa. (P10, P12)
- **R28.** A rota pública responde **exatamente igual** (mesmo status, corpo e **tempo aproximado**) para e-mail sem conta, pessoa sem MFA, código errado, código já usado, limite estourado, corpo inválido e falha interna do Cognito ou do DynamoDB que não seja sucesso. O sucesso é `200` com corpo fixo e é o único caso diferente. **Entrada mínima antes de qualquer trabalho:** corpo acima de 2 KiB, JSON inválido ou campo faltando respondem a mesma resposta genérica. O texto da tela cobre todos os casos ("Confira o e-mail e o código. Depois de várias tentativas erradas, aguarde alguns minutos."), sem o servidor distinguir. **O tempo é o canal que sobra:** e-mail sem conta faz o **mesmo trabalho** que e-mail com conta (a mesma consulta ao Cognito e uma comparação de código contra um hash fictício do mesmo custo). **O teste de unidade não prova tempo**: o que se testa é que os dois caminhos chamam as **mesmas portas na mesma ordem**; a prova de tempo é **medida no `dev`** (Verificação, passo 12). (P11)
- **R29.** A rota pública tem **limite de tentativas por pessoa e por origem**; o e-mail é **normalizado (minúsculas, sem espaços nas pontas) antes de contar e de buscar**, para `A@x.com` e `a@x.com` não furarem o limite (o alias não entra: R30 protege pelo `sub`); ao estourar, responde o mesmo erro genérico e não confere mais códigos até o limite expirar. A origem vem de `r.RequestContext.HTTP.SourceIP`, **nunca** de cabeçalho (`x-forwarded-for` é de quem chama); `eventoDe` de `local.go` passa a preencher esse campo a partir de `r.RemoteAddr` (sem a porta). O limite é do servidor (a Function URL não tem WAF) e, por gravar no DynamoDB sem autenticação, **lê o contador por origem antes de qualquer outra coisa** e tem um **teto de gravações por origem**; os números (tentativas, janela, TTL) são do dono. (P11, P14)
- **R30.** Um código só vale para o usuário (`sub`) a que foi gerado; o código de uma pessoa **nunca** desliga o MFA de outra, mesmo com e-mail parecido ou alias. (P10)
- **R31.** A rota pública **só desliga o MFA**: nunca cria sessão, nunca devolve token e nunca altera a senha. A entrada continua exigindo a senha pelo login do Cognito. (P9)
- **R32.** A Lambda ganha, no Cognito, **só** as ações `AdminSetUserMFAPreference` e **`AdminGetUser`** (esta devolve o `sub` a partir do e-mail, se o pool aceitar o e-mail como `Username`, já que usa `UsernameAttributes: [email]`; **[PENDENTE]** confirmar no `dev`, Verificação, passo 11; se não aceitar, o fallback é `ListUsers` com filtro por `email`, que precisa de teto de uso porque a cota é compartilhada com outras chamadas administrativas do pool), **sobre o próprio pool**; as roles do Actions são atualizadas **antes** do merge. Nenhum log escreve código, e-mail nem corpo.
- **R33.** A **tela de recuperação** (sem login) pede e-mail e um código, chama `POST /mfa/recuperar` e mostra só o que o servidor devolve: no sucesso, "MFA desligado; entre com e-mail e senha"; no erro, uma mensagem genérica na própria tela. Ela fica **fora do login do Cognito**: na abertura do app Android, como um **segundo botão** ao lado do "Entrar" (a Abertura sem sessão já mostra "Entrar"; a recuperação **não dispara o login**), e por um endereço próprio no navegador e no PWA. **Hoje `Sessao.token()` do navegador sempre termina em `irParaLogin`:** o endereço da recuperação carrega **antes** de `Sessao.token()` e **nunca inicia o login**; depois do sucesso, o botão "Entrar" leva ao login. O front **não decide** nada sobre o código (não confere, não conta tentativas). Segue a demo aprovada (R24) e o botão voltar (R22).
- **R35.** **Aviso quando o MFA muda** (decisão D3, dono, 2026-11-19): a pessoa é avisada **pelo canal que já tem ligado** (Telegram ou push, pelo despachante de avisos), **sem prometer entrega a quem não tem nenhum**; sem canal ligado, nada é enviado e isso não falha a operação. O aviso diz **só o fato** (o MFA foi desligado ou ligado), nunca código, e-mail nem lista, e segue as regras de `security.md` e do despachante (o `service` trata o canal como valor opaco). **Quando o servidor vê a mudança:** no sucesso da rota pública (MFA desligado pela recuperação) e no `200` do confirmar (R25a: MFA ligado ou lista nova). **[PENDENTE]** Desligar e ligar em Listas falam direto com o Cognito (R14, R17) e o servidor não os vê; para avisar esse caso o front precisaria chamar uma rota, o que ainda não foi decidido. Até lá, o spec **não** promete aviso nesse caso. (D3)
- **R34.** O log da rota pública é `acao=mfa.recuperar rota=mfa` (por `rotular`), sem e-mail, origem, código nem corpo (`linhaDeLog` só aceita campos fixos); falha interna loga só o nome do erro, sem o e-mail. O `handler` inicia o rótulo com `acaoSemToken`, e a rota pública grava o próprio rótulo **antes** de entregar ao controller, para a linha não dizer `sem-token` numa chamada atendida. (P12)

## API

As três rotas novas do backend (Fase 3). `{id}` e valores entre `<>` são do cliente; o `sub` vem **sempre** do token, nunca do corpo. Todas as outras rotas ficam como estão.

```
POST   /me/mfa/codigos                  (sem corpo; autenticada)
         -> 200 { "listaId": "<id>", "codigos": ["<c1>", ..., "<cN>"] }   (a lista PENDENTE; devolvida uma única vez; a ativa não muda)
         | 401 (sem token ou token inválido) | 500

POST   /me/mfa/codigos/confirmar        { "listaId": "<id>" }             (autenticada)
         -> 200 { "ok": true }                                            (promove a pendente a ativa e apaga a ativa anterior; idempotente: o mesmo listaId já ativo também dá 200)
         | 401 | 409 { "erro": "lista_substituida" }                      (a pendente foi substituída por outra geração ou expirou; a ativa continua)
         | 400 (corpo inválido) | 500

POST   /mfa/recuperar                   { "email": "<e-mail>", "codigo": "<código>" }   (PÚBLICA: sem token; o cabeçalho authorization é ignorado)
         -> 200 { "ok": true }                                            (o MFA da pessoa foi desligado; a lista ativa inteira foi apagada)
         | 400 { "erro": "recuperacao_invalida" }                         (a ÚNICA resposta de falha: e-mail sem conta, pessoa sem MFA, código errado ou usado,
                                                                           código da lista pendente, limite de tentativas estourado, corpo inválido ou acima de 2 KiB,
                                                                           falha interna do Cognito ou do DynamoDB)
```

- **Ações em `auth/authz.go`:** `mfa.confirmar` para a segunda rota (qualquer pessoa autenticada); a de gerar, para a primeira (nome fixado no código, qualquer pessoa autenticada); `mfa.recuperar` para a terceira, que **não** é listada em `Permitido` para nenhum papel e só existe por `auth.RotaPublica` (R27). Nenhuma das três passa por papel nem por `/spaces/{sid}`.
- **`400 recuperacao_invalida` é o valor fixado por este spec** para a resposta única de R28 (mesmo status e mesmo corpo em todo caso de falha, sem `429` nem distinção de motivo); mudar o status ou o texto é decisão do dono, mas ele continua **um só** para todos os casos.
- **Rótulo no log:** `acao=mfa.recuperar rota=mfa` para a terceira (R34); as duas autenticadas seguem o padrão de `rotular` com a ação própria. Nenhuma linha leva e-mail, origem, código, `listaId` nem corpo.
- **Limite de tentativas:** por pessoa e por origem (`SourceIP`), conferido **antes** de ler qualquer código (R29); estourar devolve a mesma resposta única.

**Chamadas do front ao Cognito** (não passam pela Lambda; R14). `POST https://cognito-idp.<região>.amazonaws.com/`, `Content-Type: application/x-amz-json-1.1`, o access token **no corpo**, nunca na URL nem em cabeçalho extra:

```
X-Amz-Target: AWSCognitoIdentityProviderService.GetUser                  { "AccessToken": "<t>" }
         -> 200 { "UserMFASettingList": [...], "PreferredMfaSetting": "..." }        (o front só exibe estes dois campos; não guarda o resto)
X-Amz-Target: AWSCognitoIdentityProviderService.AssociateSoftwareToken   { "AccessToken": "<t>" }
         -> 200 { "SecretCode": "<chave>" }                                          (vira o QR e a chave em texto; só em memória)
X-Amz-Target: AWSCognitoIdentityProviderService.VerifySoftwareToken      { "AccessToken": "<t>", "UserCode": "<6 dígitos>", "FriendlyDeviceName": "..." }
         -> 200 { "Status": "SUCCESS" }                                              (qualquer outro Status ou erro: não liga)
X-Amz-Target: AWSCognitoIdentityProviderService.SetUserMFAPreference     { "AccessToken": "<t>", "SoftwareTokenMfaSettings": { "Enabled": true|false, "PreferredMfa": true|false } }
         -> 200 {}                                                                   (ligar: só depois do 200 do confirmar, R16; desligar: R17)
```

**Login** (`GET https://<domínio do pool>/oauth2/authorize`, fluxo `code` com PKCE, o mesmo de hoje): o **login normal** pede `scope=openid email` (R7); o **segundo login**, ao abrir a seção do MFA, pede `scope=openid email aws.cognito.signin.user.admin` com `code_challenge` e `state` novos e o parâmetro de reautenticação a confirmar no `dev` (`prompt=login` ou `max_age`, R8 **[PENDENTE]**); em ambos o retorno é `?code=...&state=...` na URL de retorno do cliente.

## Testes

Cada um falha antes da mudança. Os de front são `web/<nome>.test.js` (o CI roda todos); os de infra são conferidos por `sam validate --lint` e pelo changeset.

| Id | O que prova |
|---|---|
| M-T1 | O `template.yaml` tem `MfaConfiguration: OPTIONAL` e `EnabledMfas: [SOFTWARE_TOKEN_MFA]`, sem SMS (R1) |
| M-T1a | O domínio tem `ManagedLoginVersion: 2` e o estilo de marca está ligado ao cliente; o nome do domínio e as URLs de retorno não mudaram (R2a) |
| M-T2 | O cliente **permite** o escopo novo, **não** lista fluxo de senha, e não perde as URLs de retorno (R2) |
| M-T3 | A validação de `token.go` não mudou e nenhuma rota existente trocou de ação; as **três** rotas novas têm ação própria em `authz.go`; `go vet ./...`, `go test ./...` e `arquitetura_test.go` passam (R4) |
| M-T4 | O pedido de login normal tem o escopo `openid email` e **não** tem `aws.cognito.signin.user.admin` (R7) |
| M-T5 | O pedido do segundo login tem o escopo novo, PKCE e `state` novos e o parâmetro de reautenticação confirmado no `dev` (R8) |
| M-T6 | O access token do segundo login não é gravado em `localStorage`, `sessionStorage` nem no `Cofre`, e some ao fechar a seção e ao sair da conta (R9) |
| M-T7 | Depois do segundo login, o ID token e o refresh do login normal continuam os mesmos; os tokens do segundo são descartados; **no navegador o retorno `mfa` não toca `financas.auth`; no app não grava `refresh` no `Cofre` nem recarrega a página**; depois de trocar o `code`, o front chama `POST /oauth2/revoke` com o refresh do segundo login (R10) |
| M-T7a | Um retorno do tipo `normal` continua exatamente como hoje (grava `financas.auth` no navegador; grava `refresh` e recarrega no app); o tipo do PKCE pendente decide o ramo (R10) |
| M-T8 | Cancelar o segundo login deixa o app logado e o MFA sem mudança (R11) |
| M-T9 | No navegador, sair para o segundo login guarda só a intenção, sem token, e a seção reabre ao voltar; a intenção expira em 10 minutos e é apagada no retorno com erro ou cancelamento, então a seção **não** reabre sozinha depois (R12) |
| M-T10 | Nenhuma linha de log contém token, `code`, `verifier` nem URL de retorno (R13) |
| M-T11 | Toda chamada ao Cognito leva o access token no corpo e nunca na URL nem em cabeçalho extra (R14) |
| M-T12 | O front exibe o estado devolvido por `GetUser` e não o guarda; atributos como e-mail não aparecem nem são guardados (R15) |
| M-T13 | Ligar segue a ordem de R16; sem `Verify` igual a `SUCCESS`, sem a lista gerada, **sem o toque em "Guardei a lista"** ou **sem o `200` do confirmar**, `SetUserMFAPreference` **não** é chamado; o confirmar **nunca** é chamado antes do toque; regerar com o MFA ligado usa só gerar, mostrar e confirmar; `409 lista_substituida` mostra a mensagem de gerar de novo (R16, R18) |
| M-T14 | Desligar chama `SetUserMFAPreference` com `Enabled: false` e relê por `GetUser`, depois de confirmar; **nenhuma rota nossa é chamada** ao desligar (a lista ativa fica) (R17) |
| M-T15 | Código errado, bloqueio por tentativas, falta de internet e recusa por falta de permissão mostram a mensagem **na janela** e não alteram o estado (R18) |
| M-T16 | A chave e o QR não vão a armazenamento, log, URL, serviço externo nem área de transferência, e são limpos ao fechar (R19) |
| M-T17 | A tela da lista mostra a frase "SE PERDER A LISTA, PERDE O ACESSO." e o texto da demo aprovada; a lista não é gravada no aparelho e some ao fechar (R20) |
| M-T18 | A janela do MFA entra em `camadasAbertas()` e `fecharCamadaTopo()`; o voltar a fecha; o segundo login não é camada (R22) |
| M-T19 | A subida e a abertura da seção não leem nada do servidor nosso (`carga.test.js`) (R23) |
| M-T20 | Nem o template nem o front definem `MfaConfiguration: ON` (R6) |
| M-T21 | `POST /me/mfa/codigos` sem token é `401`; com token devolve a lista pendente uma vez e grava só hashes (R25, R26) |
| M-T21f | Gerar **sem corpo** é `200`; confirmar **sem corpo** é `400`; `RotaSemEspaco` reconhece as duas rotas e o ramo de `processar` é um `switch` por ação (as rotas de hoje, como `push/sair`, seguem iguais) (R25, R25a) |
| M-T21a | Gerar com uma lista ativa existente **não** invalida a ativa: um código dela ainda recupera (R25) |
| M-T21b | Um código da lista **pendente** não recupera nada na rota pública (resposta genérica) (R26) |
| M-T21c | Confirmar com o `listaId` certo promove a pendente e apaga a ativa anterior; o código antigo deixa de valer e o novo vale (R25a) |
| M-T21d | Confirmar é idempotente; confirmar o `listaId` de uma pendente substituída ou expirada dá `409` e não muda a ativa (R25a) |
| M-T21e | `confirmar` sem token é `401`; com o token de outra pessoa não mexe na lista desta (o `sub` vem só do token) (R25a) |
| M-T22 | Um código funciona uma vez: o segundo uso do mesmo código responde o erro genérico (R26, R27) |
| M-T22a | Dois usos simultâneos do mesmo código: só um sucesso; o outro, resposta genérica (R27) |
| M-T22b | `AdminSetUserMFAPreference` falha: a marca de usado é desfeita e o código ainda vale (R27) |
| M-T23 | E-mail sem conta, pessoa sem MFA, código errado e código já usado produzem a **mesma** resposta; e-mail sem conta e e-mail com conta chamam as **mesmas portas na mesma ordem** (o teste de unidade **não prova tempo**; o tempo se mede no `dev`) (R28) |
| M-T24 | Depois do limite de tentativas por pessoa e por origem, nem o código certo é conferido até o limite expirar (R29) |
| M-T24a | Variações de caixa e de espaço nas pontas do e-mail (`A@x.com`, `a@x.com`, ` a@x.com `) contam no **mesmo** contador de tentativas (R29) |
| M-T25 | O código de uma pessoa não desliga o MFA de outra, nem com e-mail parecido ou alias (R30) |
| M-T26 | A rota pública, no sucesso, só chama `AdminSetUserMFAPreference` e não devolve token nem cria sessão nem altera a senha (R31) |
| M-T27 | `POST /mfa/recuperar` é a **única** rota liberada sem token; qualquer outra sem token segue `401` (R27) |
| M-T27a | `RotaPublica` devolve `true` **só** para `POST` `["mfa","recuperar"]`; `false` para `GET`, `PUT`, `["mfa","recuperar","x"]`, `["mfa"]`, `["me"]`, `["push","sair"]`, `["spaces",...]`, caixa diferente e `["mfa","","recuperar"]` (R27) |
| M-T27b | No roteador, toda rota **autenticada** conhecida (as de `RotaParaAcao`, `/me`, `/push/sair`, `/me/mfa/codigos` e `/me/mfa/codigos/confirmar`) sem token continua `401` (R4, R27) |
| M-T27c | `POST /mfa/recuperar` **sem** `authorization` chega ao controller; com `authorization` lixo ou expirado dá a **mesma** resposta; `VerificarToken` não é chamado nessa rota (R27) |
| M-T27d | Corpo vazio, JSON inválido, corpo de 3 KiB e e-mail ausente respondem **igual** a código errado (R28) |
| M-T27e | A origem do limite vem de `SourceIP`; um `x-forwarded-for` falso não muda a chave do limite (R29) |
| M-T27f | O log da rota pública é `acao=mfa.recuperar rota=mfa`, não `sem-token`, e não contém e-mail, origem, código nem corpo (R34) |
| M-T28 | Nenhum log, resposta de erro ou teste contém código, e-mail ou corpo (R26, R32) |
| M-T29 | A tela de recuperação mostra só o que o servidor devolve, sem conferir código nem contar tentativas, e leva a pessoa a entrar com e-mail e senha (R33) |
| M-T29a | O endereço da recuperação carrega **antes** de `Sessao.token()` e **nunca inicia o login** (nenhum `irParaLogin`); depois do sucesso, "Entrar" leva ao login; no app, a Abertura sem sessão tem o segundo botão e ele não dispara o login (R33) |
| M-T31 | O sucesso da recuperação e o `200` do confirmar pedem um aviso ao despachante só com o fato (sem código, e-mail nem lista); sem canal ligado nada é enviado e a operação **não falha**; a falha do canal não derruba a resposta (R35) |
| M-T30 | A permissão nova da Lambda é só `AdminSetUserMFAPreference` e `AdminGetUser` (nominais) sobre o próprio pool, e as roles do Actions foram atualizadas (R32) |

**Quebrar de propósito:** (a) fazer o login normal pedir o escopo novo tem de fazer M-T4 falhar; (b) gravar o access token em `localStorage` tem de fazer M-T6 falhar; (c) chamar `SetUserMFAPreference` antes do `Verify` tem de fazer M-T13 falhar; (d) fazer o gerar apagar a lista ativa tem de fazer M-T21a falhar; (e) tirar a condição `usado` do uso do código tem de fazer M-T22a falhar; (f) fazer `RotaPublica` devolver `true` para qualquer `POST` tem de fazer M-T27a e M-T27b falharem; (g) mover `VerificarToken` para depois do `switch` inteiro tem de fazer M-T27b falhar.

## Verificação no `dev`

1. Deploy da Fase 2 no `financas-dev` e **changeset conferido a olho**: só modificação de propriedade (R3). O pool e o cliente do `dev` foram **devolvidos ao original** em 2026-11-19; o primeiro deploy deve mostrar `MfaConfiguration` e o escopo como mudança.
2. **P2:** com uma sessão do login do Cognito ativa, o segundo login pede senha de novo? Testar `prompt=login` e `max_age`. Sem isso, decidir com o dono antes da Fase 3 (R8).
3. App do `dev` instalado com o front novo (`scripts/instalar-app-android.ps1 -Ambiente dev`) e o navegador: ligar com o autenticador do celular, sair, entrar pedindo o código, desligar.
4. **Android:** o segundo login com o escopo novo na aba do sistema, e os dois casos de voltar (antes da senha e antes do código).
5. **Recuperação:** com o MFA ligado e sem o autenticador, um código da lista, na tela de recuperação, desliga o MFA; o mesmo código não funciona de novo; a pessoa entra com e-mail e senha e liga de novo com uma lista nova. No navegador, no PWA e no app Android.
6. "Esqueci a senha" com o MFA ligado; PWA instalado.
7. O usuário de teste termina sem MFA e nada do `dev` fica fora do `template.yaml`.
8. **CORS das chamadas ao Cognito:** o navegador e o WebView, a partir do CloudFront, chamam `cognito-idp.<região>.amazonaws.com` por `fetch` com `Content-Type: application/x-amz-json-1.1`. Os testes do intent foram feitos por script, então isto **não** está provado (R14).
9. **Revoke:** `POST /oauth2/revoke` aceita o cliente público com o refresh do segundo login, e o access token já emitido continua valendo até a seção fechar (R10).
10. **Bloqueio por tentativas:** quantos erros de `VerifySoftwareToken` o Cognito tolera antes de recusar por excesso de tentativas, e a mensagem que ele devolve (R18); até provar, o spec não afirma o número.
11. **`AdminGetUser`** aceita o e-mail como `Username` e devolve o `sub`; se não aceitar, o fallback é `ListUsers` com filtro por `email` e teto de uso (R32).
12. **Tempo da rota pública:** e-mail sem conta e e-mail com conta respondem em tempos próximos, medidos no `dev` com a mesma carga (R28).

## Fora do escopo

Recuperação por e-mail, por prazo ou por perguntas de segurança, layout ou textos próprios no login (o Managed login só aceita marca), passkey, SMS, MFA por e-mail, MFA obrigatório, login social e qualquer caminho para quem perdeu o autenticador **e** a lista (ver `intent.md`).
