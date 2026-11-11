# Spec: app Android, fases 1 a 3 (casca, login nativo e biometria)

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id (`A-T<n>`), para o PR dizer exatamente o
que cobriu. Quando este texto e o do intent divergirem, vale o plan. As fases 4 (push), 5 (release) e 6 (produção aberta) ganham spec própria.

**Estado:** rascunho (2026-11-01). **Fase 1 implementada em 2026-11-11** (branch `feature/app-android-casca`), com o APK ainda **não compilado nem testado em aparelho** (ver "Fase 1: o que foi verificado" no `spec.md`). Fases 2 e 3 sem código. Onde o texto diz "confirmar", é uma verificação que a fase faz e registra aqui no PR.

Convenção dos testes: os de lógica são `node web/*.test.js` (o CI roda todos sem lista) e **falham antes da mudança**. Os que dependem do aparelho são marcados
**[aparelho]** e entram como roteiro manual no PR, com o resultado escrito. Diga o que não deu para ver em vez de afirmar que funciona.

## Fase 1: a casca

### Regras

- **R1.** `app-android/` contém o projeto Capacitor. `webDir` é `www`, gerado por `scripts/app-android.ps1`; **nada** de `www/` vai para o git.
- **R2.** `scripts/app-android.ps1 -Ambiente dev|prod` falha com mensagem clara se o `web/config.dev.js` (dev) ou o `web/config.js` (prod) não existir, e nunca copia o
  de um ambiente para o pacote do outro.
- **R3.** O pacote do `dev` tem `applicationId` `<base>.dev`, nome "Finanças DEV" e ícone marcado; o do `prod` tem `<base>` e "Finanças". Os dois instalam juntos.
- **R4.** O app carrega `index.html` do pacote (sem URL remota) e a faixa "AMBIENTE DE TESTE" aparece só no `dev`, pelo `ambiente.js` de hoje.
- **R5.** A origem da WebView é `https://localhost` (`androidScheme: "https"`). Essa origem entra em `Cors.AllowOrigins` da Function URL por um parâmetro `AppOrigin`,
  nos dois ambientes. Nenhuma outra rota, política ou autenticação muda.
- **R6.** O botão voltar físico segue `voltar.js`: fecha a camada de cima, troca de aba como o histórico do navegador e, na raiz, **minimiza** o app sem perder o estado.
- **R7.** Nenhuma regra de negócio, rota, campo ou chave do DynamoDB muda. O front não ganha nome de status, banco ou predicado (`web/sem-status.test.js` continua verde).
- **R8.** O `.gitignore` ignora `app-android/www/`, `app-android/android/app/build/`, `app-android/android/.gradle/`, `local.properties`, `*.jks`, `*.keystore`, `*.apk` e
  `*.aab`. Nenhum arquivo de assinatura, `config.js` ou dado real entra no repositório.

### Testes

| Id | O que prova |
|---|---|
| A-T1 | `app-android.ps1` sem o config do ambiente termina com erro e não gera `www/` (R2) |
| A-T2 | O `www/` do `dev` contém o `apiUrl` e o `clientId` do `dev` e **nenhum** do `prod`, e vice-versa; teste lê os dois pacotes gerados (R2) |
| A-T3 | O template validado (`sam validate`) tem `AppOrigin` em `AllowOrigins` e as demais origens intactas (R5) |
| A-T4 | `git status` depois de um build não lista `www/`, `build/`, `*.apk` nem `local.properties` (R1, R8) |
| A-T5 **[aparelho]** | `dev` e `prod` instalados juntos, cada um com nome, ícone e sessão próprios (R3) |
| A-T6 **[aparelho]** | A tela abre sem rede e mostra o erro de conexão de sempre; com rede, a primeira chamada à API do `dev` volta sem erro de CORS (R4, R5) |
| A-T7 **[aparelho]** | Os cenários de `demo-voltar-navegador.html` funcionam: janela, aba, edição com texto digitado e, na raiz, minimizar (R6) |
| A-T8 | `web/*.test.js` e `go test ./...` seguem verdes (R7) |

**Verificações da fase 1, a registrar no PR:** (a) `window.Capacitor.Plugins` expõe `App`, `Browser` e o armazenamento seguro sem `import`; (b) a origem real da WebView é
a esperada (`location.origin` no console remoto); (c) o que, de `web/`, atrapalha dentro do app (por exemplo `manifest.webmanifest` e o registro de PWA).

### Fase 1: o que foi verificado (2026-11-11)

| Item | Resultado |
|---|---|
| A-T1, A-T2, A-T4 (parte do `.gitignore`), R3 (sabores), R4 (sem `server.url`) | **Verde**: `node web/app-android.test.js` |
| A-T3 | Parte local verde (`AppOrigin` em `AllowOrigins`, demais origens intactas); o `sam validate --lint` roda no CI |
| R6 (lógica do voltar físico) | **Verde** com plugin de mentira: `node web/nativo.test.js`. Quebrar o código faz o teste falhar |
| A-T8 | `node web/*.test.js` e `go vet`/`go test` verdes |
| `cap sync` | Rodou e copiou `www/` e o `capacitor.config.json` (appId e `https`) para o projeto Android |
| Compilar o APK (`assembleDevDebug`) | **Não rodou**: a sessão onde isto foi feito não tem Android SDK nem PowerShell. Primeira coisa a fazer no computador do dono |
| A-T5, A-T6, A-T7 e as verificações (a), (b) e (c) | **[aparelho]**: pendentes. Nada disso foi visto funcionando |

### Decisões da fase 1

- `applicationId` base: `br.com.cicerotec.financas` (decidido em 2026-11-11).
- O botão voltar é `web/js/nativo.js`, carregado depois de `voltar.js`; no navegador e na PWA ele não faz nada.
- `AppOrigin` tem padrão `https://localhost`, então os dois ambientes o recebem sem mudar os workflows de deploy.
- Ícone legado em PNG nos dois sabores; o adaptativo fica na fase 3 (R24).

## Fase 2: login nativo e token seguro

### Regras

- **R9.** O login do navegador e o do app ficam atrás da mesma interface em `web/js/sessao.js`: `token()`, `sair()` e `limpar()`. O navegador e a PWA continuam
  exatamente como hoje (Hosted UI na mesma aba, PKCE em `sessionStorage`, token em `localStorage`). A escolha é feita por `window.Capacitor.isNativePlatform()`.
- **R10.** No app, o Hosted UI abre **no navegador do sistema (Custom Tabs)**. A WebView nunca renderiza a tela do Cognito nem vê a senha.
- **R11.** `redirect_uri` e `logout_uri` do app são `<applicationId>://auth`. O app só aceita o retorno quando o esquema e o caminho batem **e** o `state` devolvido é
  igual ao guardado; qualquer outro URL aberto é ignorado.
- **R12.** `code_verifier` e `state` são gerados com `crypto.getRandomValues`, guardados no armazenamento do app (não em `sessionStorage` nem em variável) e **apagados**
  ao fim do login, com sucesso ou erro. Um retorno com `state` ausente ou diferente reinicia o login e não troca o `code`.
- **R13.** O refresh token fica no armazenamento seguro do Android (Keystore). O ID token fica **só em memória**. `localStorage` e `sessionStorage` não recebem token no app.
- **R14.** `sair()` no app apaga o armazenamento seguro, os dados de preferência do aparelho (incluindo a biometria) e abre o logout do Cognito na aba do sistema.
- **R15.** Só a resposta `invalid_grant` do refresh leva ao login. Falha de rede, `5xx` ou resposta inesperada **mantêm a sessão**, não abrem o navegador e deixam o erro
  chegar à tela como hoje.
- **R16.** `401` da API apaga o ID token em memória e tenta **um** refresh; se falhar por `invalid_grant`, vai ao login. Nunca entra em laço.
- **R17.** O `template.yaml` acrescenta `<applicationId>://auth` (e a versão `.dev`) a `CallbackURLs` e `LogoutURLs` do `UserPoolClient` de cada ambiente. `AllowedOAuthFlows`,
  escopos, validade dos tokens e `ExplicitAuthFlows` não mudam. O backend (`auth/token.go`) não muda.
- **R18.** Nenhum log do app imprime token, `code`, `verifier` ou URL de retorno completa.

### Testes

`web/sessao.test.js`, com fakes para armazenamento, `fetch`, navegador do sistema e relógio.

| Id | O que prova |
|---|---|
| A-T9 | Retorno válido (esquema, caminho e `state` iguais) troca o `code` com o `verifier` guardado e devolve o ID token (R11, R12) |
| A-T10 | Retorno com `state` diferente, ausente ou de outro esquema **não** chama `/oauth2/token` e reinicia o login (R11, R12) |
| A-T11 | Depois do login, o `verifier` e o `state` não estão mais no armazenamento, nos dois resultados (R12) |
| A-T12 | Em modo app, nenhuma escrita em `localStorage` ou `sessionStorage` contém o token; o refresh vai ao armazenamento seguro e o ID token fica só na memória (R13) |
| A-T13 | `sair()` no app apaga o armazenamento seguro e as preferências e chama o logout do Cognito com `logout_uri` do app (R14) |
| A-T14 | Refresh com `invalid_grant` abre o login; refresh com `fetch` rejeitado (sem rede) e com `503` **não** abre o login nem apaga o token (R15) |
| A-T15 | `401` seguido de refresh falho por `invalid_grant` abre o login uma única vez; `401` seguido de refresh bom repete a chamada uma única vez (R16) |
| A-T16 | Em modo navegador, o fluxo é o de antes: `redirect_uri` é `location.origin + '/'`, PKCE em `sessionStorage` e token em `localStorage` (R9) |
| A-T17 | Nenhuma chamada de log do módulo recebe token, `code` ou `verifier` (R18) |
| A-T18 | O template (`sam validate`) tem os dois URLs de retorno nos dois ambientes e as demais configurações do cliente intactas (R17) |
| A-T19 **[aparelho]** | Login completo no `dev`: abre o navegador do sistema, volta ao app, mostra as telas; fechar e abrir o app mantém a sessão (R10, R13) |
| A-T20 **[aparelho]** | Matar o app (deslizar para fechar) com a aba de login aberta e voltar pelo retorno ainda conclui o login (R12) |
| A-T21 **[aparelho]** | Sair leva ao logout; abrir de novo pede o login; com o modo avião, o app aberto não é jogado para o login (R14, R15) |
| A-T22 **[aparelho]** | O token não aparece em `localStorage` ao inspecionar a WebView (R13) |

**Quebrar de propósito:** gravar o ID token em `localStorage` tem de fazer o A-T12 falhar; aceitar um `state` vazio tem de fazer o A-T10 falhar.

## Fase 3: biometria, ícone e acabamento

**Tela de bloqueio e interruptor são telas novas: nada disto vai ao código antes de uma demo aprovada** (`guardrails.md`). A demo mostra o bloqueio, o interruptor,
o que acontece quando a biometria falha ou não está cadastrada, os dois temas e o que a proposta faz perder (a trava não cifra o token e não vale em aparelho com root).

### Regras

- **R19.** A biometria é **opcional e desligada por padrão**. O interruptor fica em Configurações (ou onde a demo aprovada disser), existe só no app, e a escolha fica **só no
  aparelho** (armazenamento de preferências). Nenhum campo novo no DynamoDB, nenhuma rota nova.
- **R20.** Com a trava ligada, o app mostra o bloqueio ao abrir e ao voltar do segundo plano depois de **N minutos** (valor na demo; proposta: 1 minuto). O conteúdo das abas
  **não é desenhado** enquanto o bloqueio está ativo, nem aparece na visão de apps recentes do Android (janela marcada como protegida).
- **R21.** A tela libera só com sucesso da biometria ou da credencial do aparelho. Cancelar, falhar ou não ter biometria cadastrada **não libera**. Se o aparelho perdeu a
  biometria, a pessoa entra por "Sair" (login de novo), nunca por um atalho que pule a trava.
- **R22.** Ligar a trava exige uma verificação bem-sucedida na hora; desligar também.
- **R23.** O bloqueio é só interface: o token continua no Keystore (R13). `security.md` diz isso.
- **R24.** Ícone adaptativo a partir das imagens maskable de `web/icons/`, splash e cor da barra de status seguindo o tema claro e escuro do app. Mudou padrão: a demo, o
  `padrao-de-telas.md` e o resumo do `guardrails.md` mudam na mesma entrega.

### Testes

`web/bloqueio.test.js`, com fakes para relógio, armazenamento e biometria.

| Id | O que prova |
|---|---|
| A-T23 | Sem a trava ligada, nunca há bloqueio, em qualquer tempo de segundo plano (R19) |
| A-T24 | Com a trava ligada, voltar antes de N minutos não bloqueia e depois de N minutos bloqueia (R20) |
| A-T25 | Falha, cancelamento e "sem biometria cadastrada" mantêm o bloqueio e nenhum conteúdo é desenhado (R21) |
| A-T26 | Ligar e desligar chamam a verificação; se ela falhar, o estado não muda (R22) |
| A-T27 | `sair()` apaga a preferência da trava (R14) |
| A-T28 | A preferência é lida e gravada só no armazenamento do aparelho: nenhum `fetch` acontece ao ligar ou desligar (R19) |
| A-T29 **[aparelho]** | Ligar, bloquear, desbloquear com digital ou rosto e com o PIN do aparelho; a visão de apps recentes mostra a janela protegida (R20, R21) |
| A-T30 **[aparelho]** | Remover a biometria do aparelho com a trava ligada: o app não abre as abas e oferece "Sair" (R21) |
| A-T31 **[aparelho]** | Pior caso de layout (nome de banco comprido, tema escuro, 360 px e 412 px de largura) no bloqueio e no interruptor, medido por script e com captura (guardrails) |

## Verificação no `dev`

1. Fazer o deploy do `template.yaml` no `financas-dev` e conferir o changeset e o status final da stack (guardrails, "Deploy"); a tabela e o login **não** podem ser substituídos.
2. Instalar o **APK de debug** do `dev` e o do `prod` (quando houver) no aparelho do dono e rodar os testes **[aparelho]**. Nas fases 1 a 3 só se gera APK; o AAB é da fase 5.
3. Com o Logs Insights do grupo `/aws/lambda/financas-api-dev`, conferir que as chamadas do app aparecem como as do navegador e que nenhuma linha traz token ou conteúdo.

## Fora do escopo

Push, assinatura de release e Play Store, build no CI, App Link por `https`, iOS, uso sem internet, tela de versão do app, versão mínima exigida pelo servidor e qualquer mudança
de regra de negócio. O token de push e o canal FCM estão na fase 4 do plan.
