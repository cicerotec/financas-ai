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
  exatamente como hoje (Hosted UI na mesma aba, PKCE em `sessionStorage`, token em `localStorage`, **qualquer recusa do refresh e todo `401` levam ao login**). A escolha é feita
  por `window.Capacitor.isNativePlatform()`. **R15 e R16 valem só no app** (decisão do dono em 2026-11-11); estender ao navegador é outro PR.
- **R9a.** O armazenamento do app é um **plugin nosso, em Java** (decisão do dono em 2026-11-11), dentro de `app-android/android/`, exposto em `window.Capacitor.Plugins` sem
  bundler. Ele guarda pares nome e valor **cifrados com uma chave do Android Keystore** (AES-GCM; a chave nunca sai do Keystore) e só tem três operações: ler, gravar e apagar.
  Nenhum plugin de terceiros entra; o único plugin oficial novo é o `@capacitor/browser` (Custom Tabs). O `allowBackup` do manifest passa a `false` **(decidido)**, para o que o plugin
  guarda não ir para o backup do Android.
- **R10.** No app, o Hosted UI abre **no navegador do sistema (Custom Tabs)**. A WebView nunca renderiza a tela do Cognito nem vê a senha.
- **R11.** `redirect_uri` e `logout_uri` do app são `<applicationId>://auth`. O app só aceita o retorno quando o esquema e o caminho batem **e** o `state` devolvido é
  igual ao guardado; qualquer outro URL aberto é ignorado.
- **R12.** `code_verifier` e `state` são gerados com `crypto.getRandomValues`, guardados no armazenamento do app (o plugin do R9a; não em `sessionStorage` nem em variável) e **apagados**
  ao fim do login, com sucesso ou erro. Um retorno com `state` ausente ou diferente reinicia o login e não troca o `code`.
- **R13.** O refresh token fica no armazenamento seguro do Android (o plugin do R9a, cifrado por chave do Keystore). O ID token fica **só em memória**. `localStorage` e `sessionStorage` não recebem token no app.
- **R14.** `sair()` no app apaga o armazenamento seguro, os dados de preferência do aparelho (incluindo a biometria) e abre o logout do Cognito na aba do sistema.
- **R15.** **No app,** só a resposta `invalid_grant` do refresh leva ao login. Falha de rede, `5xx` ou resposta inesperada **mantêm a sessão**, não abrem o navegador e deixam o erro
  chegar à tela como hoje.
- **R16.** **No app,** `401` da API apaga o ID token em memória e tenta **um** refresh; se falhar por `invalid_grant`, vai ao login. Nunca entra em laço.
- **R17.** O `template.yaml` acrescenta a `CallbackURLs` e `LogoutURLs` do `UserPoolClient` de cada ambiente **só o URL do seu app** (`<applicationId>://auth` no `prod`, o com
  `.dev` no `dev`; ver R18f). `AllowedOAuthFlows`,
  escopos, validade dos tokens e `ExplicitAuthFlows` não mudam. O backend (`auth/token.go`) não muda.
- **R18.** Nenhum log do app imprime token, `code`, `verifier` ou URL de retorno completa.

### Regras que saíram da revisão de premissas (2026-11-11)

Uma regra para cada premissa da seção "Premissas" do [`plan.md`](plan.md) (P1 a P15). As cinco decisões que estavam abertas foram aceitas pelo dono em 2026-11-11 e estão marcadas **(decidido)**.

- **R18a (P1). No app a página não vai embora.** Depois de um login concluído pelo retorno e depois de `sair()`, o app **recarrega a página** (com o refresh token já gravado ou já
  apagado), para zerar o que fica em memória (`ESPACO`, papel, caches, ouvintes). Nenhuma chamada fica presa numa promessa que não termina: quando o app precisa de login e não o
  conclui (a pessoa fechou a aba do sistema, a troca do `code` falhou), ele chega a um estado **"sem sessão"** com saída. **(decidido)** Esse estado é mudança na Abertura e **pede demo
  aprovada antes do código** (botão "Entrar" e o que mais a demo disser). **Demo aprovada em 2026-11-12:**
  [`demo-abertura-v3-sem-sessao`](../../demos/demo-abertura-v3-sem-sessao.html). O que ela fixa: **o login abre só no toque em "Entrar"** (o app nunca abre a aba do sistema
  sozinho; a primeira abertura, o Sair e a sessão recusada param na tela "sem sessão"); fechar a aba sem entrar volta a essa tela sem mensagem de erro; a troca do `code` que
  falha mostra "Não consegui concluir o login." em `--danger`; a sessão recusada pelo servidor mostra "Sua sessão terminou.", sem vermelho; na volta com o `code` aparece
  "Entrando…"; e o erro do R18d (armazenamento seguro) é um estado de erro com "Tentar de novo" e **sem** "Entrar". No navegador nada disso aparece.
- **R18b (P2). O retorno é consultado antes de decidir logar, e só vale com login pendente.** Ao subir, o app lê a URL de lançamento (`App.getLaunchUrl()`, partida a frio) e escuta
  `appUrlOpen` (app vivo) **antes** de abrir um login. "Login pendente" é haver `verifier` e `state` guardados. Com login pendente, retorno com `state` diferente ou ausente
  reinicia o login (R12); **sem login pendente, qualquer retorno é ignorado** (R11), inclusive a mesma URL de lançamento entregue de novo a cada recarga e o retorno do logout.
  Isto resolve a diferença entre R11 e R12.
- **R18c (P3). Um refresh e um login por vez.** No app existe no máximo um refresh e um login em curso; as chamadas simultâneas a `token()` esperam o mesmo resultado.
- **R18d (P4). A escolha falha fechada.** No pacote do app (`isNativePlatform()` verdadeiro), faltar o plugin de armazenamento, o `App` ou o `Browser` é **erro na tela**, nunca o
  fluxo do navegador: nada vai a `localStorage` e `/oauth2/token` não é chamado. O plugin é registrado na `MainActivity`.
- **R18e (P5). O log do Capacitor fica desligado.** `capacitor.config.json` leva `loggingBehavior: "none"`: no padrão (`debug`) o APK de debug imprime no logcat os argumentos e
  os resultados de cada chamada de plugin, ou seja, o refresh token e a URL de retorno com o `code`.
- **R18f (P6). O esquema de retorno é o `applicationId` de cada sabor.** O `intent-filter` usa `${applicationId}` (nunca `@string/custom_url_scheme`, que no `dev` vale o do
  `prod`), e o `redirect_uri` do app vem de `App.getInfo().id`, não do `ambiente` do config. **Cada `UserPoolClient` aceita só o URL do seu ambiente** (`!If [EhDev, ...]`); isto
  corrige o R17 e o A-T18, que diziam "os dois URLs nos dois ambientes", e segue o plan.
- **R18g (P7, P11). Sem sessão não é beco.** Não haver refresh token guardado, ou o plugin falhar ao ler ou decifrar (chave do Keystore invalidada, dado restaurado em outro
  aparelho), é "sem sessão": apaga o que houver e vai ao estado do R18a. **(decidido)** Resposta `400` do refresh com qualquer `error` do OAuth (`invalid_grant`,
  `invalid_client`, `invalid_request`) leva ao login; rede e `5xx` mantêm a sessão (R15). Além do `allowBackup="false"`, o manifest ganha `dataExtractionRules` e
  `fullBackupContent` que excluem o arquivo do plugin.
- **R18h (P8). `sair()` apaga antes de abrir o logout.** A ordem é: apagar o armazenamento e **esperar a confirmação**, depois abrir o logout na aba do sistema. Se apagar falhar,
  o logout não abre em silêncio: erro na tela. O retorno do logout não tem `state` e cai no R18b (ignorado); a tela vai para "sem sessão". Sem rede o logout do Cognito não
  acontece e o cookie do Hosted UI continua no navegador do aparelho: o próximo "Entrar" pode voltar sem pedir senha. `security.md` registra isso.
- **R18i (P9). O que o Keystore protege e o que não.** O plugin expõe "ler" ao JavaScript, então um script injetado na WebView obtém o refresh token como obteria do
  `localStorage`; o ganho é o arquivo em disco cifrado e fora do backup. `security.md` diz isso. **(decidido)** A fase 2 fica com o plugin simples; o refresh feito em Java, sem o
  refresh token chegar ao JavaScript, é endurecimento para depois.
- **R18j (P10, P14). Verificações de aparelho a mais.** Depois do retorno, o voltar físico não mostra a página do Cognito; login com sessão ainda ativa no navegador e o retorno
  do logout voltam ao app sem toque extra; a troca do `code` e o refresh, inclusive a resposta de erro, são legíveis a partir de `https://localhost`. (Pelo servidor já foi visto
  em 2026-11-11: o `400` de `/oauth2/token` do `dev` volta com `Access-Control-Allow-Origin: https://localhost`.)
- **R18k (P12). O refresh não devolve refresh token novo.** Refresh bom sem `refresh_token` na resposta mantém o guardado; com um novo, grava o novo. Nunca grava vazio.
- **R18l (P13). `401` duas vezes não é login.** No app, `401` na chamada repetida depois de um refresh bom vira erro na tela: sem login e sem terceira tentativa (o `401` também
  sai quando o servidor não consegue validar o token por falha dele).
- **R18m (P15). O mesmo `UserPoolClient` para web e app, de propósito.** O changeset do deploy tem de mostrar `Modify` no cliente, sem substituição; não dá para encerrar só as
  sessões do app.
- **R18n (lacunas). `limpar()`, o que o Sair apaga e onde o módulo entra.** `limpar()` apaga a sessão local sem abrir o logout (é o que o `401` e o R18g usam). **(decidido)**
  Para o R14: o Sair apaga a sessão e `financas.space`; tema, "esconder valores" e as seções fechadas da Tendências ficam, como hoje no navegador. `sessao.js` entra no
  `index.html` antes do `api.js`, e os testes que carregam só o `api.js` (`web/carga.test.js`, `web/confirmacao.test.js`) passam a carregar os dois.
- **R18o (precondição).** A fase 2 começa pelas verificações (a) e (b) da fase 1 no aparelho (`Capacitor.Plugins` sem bundler e `location.origin`), porque a escolha do R9 e o
  plugin do R9a dependem delas.

Testes de lógica novos, em `web/sessao.test.js`: **A-T22a** segundo login não reaproveita o estado do primeiro (a recarga é pedida depois do login e do Sair) (R18a); **A-T22b**
instância nova do módulo com o mesmo armazenamento e URL de lançamento com `code` conclui o login, e a mesma URL entregue de novo não faz nada (R18b); **A-T22c** retorno sem
login pendente não chama `/oauth2/token` nem abre o navegador (R18b); **A-T22d** cinco chamadas simultâneas com `401` resultam em um refresh e um login (R18c); **A-T22e** modo
app sem o plugin não escreve em `localStorage` nem chama `/oauth2/token` (R18d); **A-T22f** `capacitor.config.json` tem `loggingBehavior: "none"`, o manifest usa
`${applicationId}` no esquema e exclui o arquivo do plugin do backup, e o template tem um URL de retorno por ambiente (R18e, R18f, R18g); **A-T22g** leitura do plugin que lança
erro apaga e vai a "sem sessão" (R18g); **A-T22h** a ordem do `sair()` e o caso em que apagar falha (R18h); **A-T22i** refresh sem `refresh_token` mantém o guardado (R18k);
**A-T22j** segundo `401` depois de refresh bom vira erro, sem login (R18l). O A-T12 passa a conferir também que `verifier` e `state` não vão a `sessionStorage`, e o roteiro de
aparelho ganha `adb logcat` durante login, refresh e sair sem token, `code` nem URL (R18, R18e).

### Testes

`web/sessao.test.js`, com fakes para armazenamento, `fetch`, navegador do sistema e relógio.

| Id | O que prova |
|---|---|
| A-T9 | Retorno válido (esquema, caminho e `state` iguais) troca o `code` com o `verifier` guardado e devolve o ID token (R11, R12) |
| A-T10 | Retorno com `state` diferente, ausente ou de outro esquema **não** chama `/oauth2/token` e reinicia o login (R11, R12) |
| A-T11 | Depois do login, o `verifier` e o `state` não estão mais no armazenamento, nos dois resultados (R12) |
| A-T12 | Em modo app, nenhuma escrita em `localStorage` ou `sessionStorage` contém o token; o refresh vai ao armazenamento seguro e o ID token fica só na memória (R13) |
| A-T13 | `sair()` no app apaga o armazenamento seguro e as preferências e chama o logout do Cognito com `logout_uri` do app (R14) |
| A-T14 | Em modo app, refresh com `invalid_grant` abre o login; refresh com `fetch` rejeitado (sem rede) e com `503` **não** abre o login nem apaga o token (R15) |
| A-T15 | Em modo app, `401` seguido de refresh falho por `invalid_grant` abre o login uma única vez; `401` seguido de refresh bom repete a chamada uma única vez (R16) |
| A-T16 | Em modo navegador, o fluxo é o de antes: `redirect_uri` é `location.origin + '/'`, PKCE em `sessionStorage` e token em `localStorage`; refresh recusado (`400` ou `503`) e `401` da API levam ao login, como hoje (R9) |
| A-T16a | O plugin de armazenamento só é chamado em modo app; em modo navegador nenhuma chamada a `Capacitor.Plugins` acontece (R9, R9a) |
| A-T16b **[aparelho]** | O que o plugin grava não aparece em claro nos arquivos do app (`run-as` no APK de debug) e some ao desinstalar; gravar, fechar o app e ler devolve o mesmo valor (R9a) |
| A-T17 | Nenhuma chamada de log do módulo recebe token, `code` ou `verifier` (R18) |
| A-T18 | O template (`sam validate`) tem um URL de retorno do app por ambiente (o do `dev` só no `dev`, o do `prod` só no `prod`) e as demais configurações do cliente intactas (R17) |
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

Push, assinatura de release e Play Store, build no CI, App Link por `https` (**volta na fase 5**, quando existir a chave de release; decisão do dono em 2026-11-11), iOS, uso sem internet, tela de versão do app, versão mínima exigida pelo servidor e qualquer mudança
de regra de negócio. O token de push e o canal FCM estão na fase 4 do plan.
