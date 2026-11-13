# Spec: app Android, fases 1 a 3 (casca, login nativo e biometria)

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id (`A-T<n>`), para o PR dizer exatamente o
que cobriu. Quando este texto e o do intent divergirem, vale o plan. As fases 4 (push), 5 (release) e 6 (produção aberta) ganham spec própria.

**Estado:** rascunho (2026-11-01). **Fase 1 implementada em 2026-11-11** (branch `feature/app-android-casca`), com o APK do `dev` **compilado, instalado e abrindo no aparelho**; os testes que pedem sessão (A-T6 e A-T7) passaram para o fim da fase 2 (ver "Fase 1: o que foi verificado"). **Fase 2 implementada em 2026-11-12** (PR 419), com o login completo visto no aparelho; o roteiro de aparelho da fase 2 está fechado; ficam os herdados A-T5, (c) e o desinstalar (ver "Fase 2: o que foi verificado"). **Fase 3a (a trava por biometria) com código em 2026-11-12** (branch `feature/app-android-biometria`), conferida pelo dono no aparelho; a 3b (ícone, splash e barra de status) sem código. Onde o texto diz "confirmar", é uma verificação que a fase faz e registra aqui no PR.

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
| Compilar o APK (`assembleDevDebug`) | **Verde** no computador do dono (2026-11-11): `scripts/app-android.ps1 -Ambiente dev` gera `app-dev-debug.apk` (4,1 MB) com pacote `br.com.cicerotec.financas.dev`, nome "Finanças DEV", `minSdk` 24 e `targetSdk` 36 (lidos do APK com `aapt2 dump badging`). Pede Node 22, JDK 21 e a plataforma Android 36; o script recusa versões mais antigas com mensagem clara e aceita `-NodeDir` e `-JavaHome`. O `prod` não foi compilado |
| Instalar e abrir no aparelho | **Verde** (2026-11-11, Xiaomi 2311DRK48G): `adb install` do APK do `dev` deu `Success` e o pacote `br.com.cicerotec.financas.dev` aparece em `pm list packages`. Em Xiaomi é preciso ligar "Instalar via USB" nas Opções do desenvolvedor, senão o erro é `INSTALL_FAILED_USER_RESTRICTED` |
| R5 pelo lado do servidor | **Verde** depois do deploy do `dev`: o preflight `OPTIONS` com `Origin: https://localhost` volta `200` com `Access-Control-Allow-Origin: https://localhost`. Não foi visto a partir do app |
| A-T5 | **Parcial**: o `dev` está instalado; nome e ícone não foram conferidos por captura e o `prod` não foi compilado nem instalado |
| Verificações (a) e (b) | **Verde em 2026-11-12**, no começo da fase 2 (ver "Fase 2: o que foi verificado") |
| A-T6 e A-T7 | Transferidos para a fase 2 (decisão do dono em 2026-11-11). **A-T6 em parte verde e A-T7 verde** em 2026-11-12: ver "Fase 2: o que foi verificado" |
| Verificação (c) | Pendente (ver "Fase 2: o que foi verificado") |

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

**Herdados da fase 1 (2026-11-11):** A-T6, A-T7 e as verificações (a), (b) e (c) só podem ser vistos com sessão dentro do app, então entram no roteiro de aparelho desta fase e o
resultado é escrito na tabela "Fase 1: o que foi verificado".

**Quebrar de propósito:** gravar o ID token em `localStorage` tem de fazer o A-T12 falhar; aceitar um `state` vazio tem de fazer o A-T10 falhar.

### Fase 2: o que foi verificado (2026-11-12)

Código no PR 419 (mergeado em 2026-11-12). Depois do deploy do `dev` (changeset com `UserPoolClient` em `Modify`, substituição `False`), o roteiro de aparelho rodou no
Xiaomi 2311DRK48G com o dono fazendo o login; o estado do app foi lido pela inspeção da WebView e o que o sistema fez, pelo `adb logcat`.

| Item | Resultado |
|---|---|
| A-T9 a A-T18 e A-T22a a A-T22j (lógica) | **Verde**: `node web/sessao.test.js` (15 testes) e `node web/abertura.test.js` (17 testes); os 27 arquivos `web/*.test.js` passam |
| Quebrar de propósito | Gravar o ID token em `localStorage` derruba o A-T12; aceitar `state` vazio derruba o A-T10 (feito e desfeito em 2026-11-12) |
| Verificações (a) e (b) da fase 1 (R18o) | **Verde no aparelho** (Xiaomi 2311DRK48G, pela inspeção da WebView): `location.origin` é `https://localhost`; `Capacitor.isNativePlatform()` é `true`; `Capacitor.Plugins` traz `App`, `Browser` e `Cofre` sem bundler; `App.getInfo().id` devolve `br.com.cicerotec.financas.dev` |
| Compilar e instalar | **Verde**: `scripts/app-android.ps1 -Ambiente dev` compila com o plugin em Java e o `@capacitor/browser` 8.0.5; `adb install -r` deu `Success` |
| Tela "sem sessão" no aparelho | **Verde, com captura**: ícone com a faixa DEV, "Finanças", a frase "Entre com a sua conta para continuar." e o botão "Entrar", como na demo v3; `Sessao.modo` é `app` |
| A-T16b (parte) | **Verde**: gravar, matar o app e ler devolve o mesmo valor; o arquivo `shared_prefs/cofre.xml` só tem Base64 cifrado e a busca pelo valor em claro nos arquivos do app (`run-as`) não acha nada; apagar devolve `null`. Não foi visto: sumir ao desinstalar |
| R18, R18e (log) | **Verde**: `adb logcat` durante dois logins, um refresh, o Sair e uma troca recusada não traz nenhum JWT, `code`, `code_verifier` nem `refresh_token`. O Android registra o retorno só como `dat=br.com.cicerotec.financas.dev://auth/...`, sem os parâmetros; do Capacitor só aparece "Starting BridgeActivity" |
| Navegador | A página servida localmente continua indo ao Hosted UI na mesma aba, como antes. Login completo no navegador não foi testado (pede o `dev` publicado) |
| A-T19 (login completo) | **Verde**: "Entrar" abre a aba do sistema (`CustomTabActivity` do Chrome), o login volta ao app sem toque extra, o refresh vai ao `Cofre`, o PKCE é apagado, a página recarrega e o Lançar aparece com os 11 meios de pagamento do `dev` e a faixa de ambiente de teste. Fechar (`force-stop`) e abrir mantém a sessão, sem aba de login |
| A-T22 e A-T12 no aparelho | **Verde**: com sessão, `localStorage` só tem `valoresEscondidos` e `financas.tema`, `sessionStorage` está vazio e nenhum dos dois contém JWT; `shared_prefs/cofre.xml` só tem o `refresh` cifrado |
| R16 (refresh) | **Verde**: `Sessao.recusado(token)` devolveu um ID token novo, diferente do anterior, sem abrir login |
| A-T21 (Sair) | **Verde**: o Sair apaga o `refresh` e `financas.space` (o tema fica), abre o logout na aba do sistema e o retorno traz o app de volta em 0,3 s, sem toque, na tela "sem sessão". Fechar a aba de login sem entrar volta à mesma tela, sem mensagem |
| A-T21 e R15 (sem rede) | **Verde** (2026-11-12): com o **modo avião** ligado pelo dono e o app aberto e logado, nenhuma tela foi para o "Entrar"; cada tela mostrou a falha do jeito dela (ver "Sem rede: o que cada tela mostra", abaixo). Antes, com a rede cortada só dentro da WebView, o refresh falhou com erro de rede, o `refresh` continuou no `Cofre` e o token voltou a sair com a rede de volta |
| A-T6 (herdado) | **Parte verde**: com rede, `/me` e as três leituras do Lançar voltaram de `https://localhost` sem erro de CORS, e a troca do `code` e o refresh no Cognito também (R18j, P14). Abrir o app sem rede no aparelho não foi visto |
| A-T20 (app morto com a aba aberta) | **Verde** (segunda rodada, 2026-11-12): com a aba de login na frente e o PKCE pendente no `Cofre`, o processo do app foi morto (`run-as ... kill -9`; `pidof` ficou vazio). O dono fez o login; o Android iniciou um processo novo pelo retorno ("Start proc ... for top-activity"), o app leu o URL de lançamento, trocou o `code`, gravou o `refresh`, apagou o PKCE e abriu o Lançar com sessão. Nada de token no `logcat` nem no `localStorage`. Antes disso, um retorno falso em partida a frio já tinha mostrado "Não consegui concluir o login." com o PKCE apagado |
| R18j (voltar depois do retorno) | **Verde no que foi visto**: depois do login o histórico da WebView tem 1 entrada (a página do Cognito nunca entra nela) e o voltar físico segue a pilha do app (A-T7) |
| A-T7 (herdado: botão voltar físico) | **Verde** (2026-11-12, feito pelo dono no aparelho): com a janela de Tags aberta o voltar fecha a janela; do Histórico volta ao Lançar; na edição com texto digitado pergunta "Descartar o que foi digitado?"; na raiz o app minimiza e, reaberto pelo ícone, mantém o valor digitado. O computador não consegue apertar o voltar neste aparelho (`adb shell input` é recusado sem a permissão de segurança da MIUI) |
| Verificação (c) (herdada), A-T5, A-T16b (sumir ao desinstalar) | **Pendentes**: o que de `web/` atrapalha dentro do app, `dev` e `prod` instalados juntos, e desinstalar |
| A-T18 (`sam validate --lint`) | Roda no CI; aqui só o texto do template foi conferido |

#### Sem rede: o que cada tela mostra (modo avião, 2026-11-12)

Visto pelo dono no aparelho. **Nada disto é da fase 2**: o `web/js/app.js` não mudou nela e o navegador se comporta igual; fica registrado porque o app Android torna o uso sem
rede mais comum. Corrigir é mudança de tela (pede demo) e não está no escopo do item 54: é o item 63 do `BACKLOG.md`.

| Tela | O que aparece | De onde vem |
|---|---|---|
| Histórico e Histórico de versões | "Failed to fetch" | O texto cru do erro do `fetch` (`err.message`) vai para a tela |
| Futuros | Não carrega e não diz nada (esperado 1 minuto) | A falha só vai ao console (`carregarFuturos` faz `console.error` e marca `futurosSujo`) |
| Saldos | Não carrega e não diz nada (esperado 1 minuto) | `carregarSaldos` marca a falha (`fechFalhou`), mas a frase só aparece dentro do bloco "Fechamento mensal" de um meio aberto |
| Cartão de crédito | Não carrega e não diz nada (esperado 1 minuto) | `carregarCartoes` guarda "Não consegui ler os cartões…" (`cartaoErro`), mas `renderCartao` só a mostra quando ainda não há resultado anterior |
| Listas, ao excluir uma tag | "Não consegui conferir a tag. Tente de novo." | Mensagem própria da tela |
| Lançar, ao salvar | "Não foi possível salvar. Confira a conexão e tente de novo." | Mensagem própria da tela |

### Decisões da fase 2

- **Login só no toque** (demo aprovada em 2026-11-12). Onde R12 e R18b dizem "reinicia o login", o app **não abre a aba sozinho**: apaga o `verifier` e o `state` e mostra a tela
  "sem sessão" com "Não consegui concluir o login."; quem reinicia é a pessoa, no "Entrar".
- O plugin se chama `Cofre` (`CofrePlugin.java`): `ler({nome})`, `gravar({nome, valor})` e `apagar({nome})`. O nome entra como dado autenticado da cifra, então um valor
  copiado para outro nome não decifra. O arquivo é `shared_prefs/cofre.xml`.
- A interface do módulo ganhou `recusado(tokenUsado)` (o que o `api.js` chama no `401`) e `entrar()` (o toque no botão), além de `token()`, `sair()` e `limpar()`.
- Só o status `400` do refresh encerra a sessão; qualquer outra resposta que não seja um token (rede, `5xx`, `200` sem `id_token`) mantém a sessão e vira erro na tela.
- Quando a sessão termina com o app aberto, a Abertura **volta a cobrir** a tela com o "Entrar"; depois do login a página recarrega.
- **Achado no aparelho (2026-11-12), sem correção:** um retorno com `state` certo e `code` recusado, chegando com sessão válida e PKCE pendente, cobre o app com "Não consegui
  concluir o login." mesmo havendo `refresh` guardado; reabrir o app volta ao normal. Só acontece se houver PKCE pendente com sessão ativa, o que o uso normal não produz (o "Entrar" só
  existe sem sessão).

## Fase 3: biometria, ícone e acabamento

**Tela de bloqueio e interruptor são telas novas: nada disto vai ao código antes de uma demo aprovada** (`guardrails.md`). A demo mostra o bloqueio, o interruptor,
o que acontece quando a biometria falha ou não está cadastrada, os dois temas e o que a proposta faz perder (a trava não cifra o token e não vale em aparelho com root).

### Regras

- **R19.** A biometria é **opcional e desligada por padrão**. As chaves ficam em **Listas, no cartão Aparência, subseção "Segurança"** (decisão do dono em 2026-11-12: a aba
  Configurações é só do dono, e a trava é de cada pessoa, inclusive do membro), existem só no app, e a escolha fica **só no aparelho** (`localStorage`: `financas.trava` e
  `financas.recentes`). Nenhum campo novo no DynamoDB, nenhuma rota nova.
- **R19a.** São **duas chaves independentes** (decisão do dono em 2026-11-12): "Bloquear com biometria" (a trava) e "Esconder nos apps recentes" (a janela protegida). A segunda
  liga e desliga na hora, sem confirmação, e **também impede capturas e gravação de tela do app** (o Android só oferece as duas coisas juntas, por `FLAG_SECURE`).
- **R20.** Com a trava ligada, o app mostra o bloqueio ao abrir e ao voltar do segundo plano depois de **1 minuto** (decisão do dono em 2026-11-12). O conteúdo das abas
  **não é mostrado** enquanto o bloqueio está ativo. A janela protegida (o app em branco nos apps recentes) é a chave do R19a, não vem junto com a trava.
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

### Fase 3a (a trava): o que foi verificado (2026-11-12)

A fase 3 foi dividida (decisão do dono em 2026-11-12): **3a, a trava por biometria** (esta) e **3b, ícone adaptativo, splash nativo e barra de status** (R24, com demo própria,
ainda sem código). Demo aprovada em 2026-11-12: [`demo-bloqueio-v1`](../../demos/demo-bloqueio-v1.html). Código na branch `feature/app-android-biometria`.

| Item | Resultado |
|---|---|
| A-T23 a A-T28 (lógica) | **Verde**: `node web/bloqueio.test.js` (10 testes); os 28 arquivos `web/*.test.js` passam |
| Quebrar de propósito | Liberar com qualquer resposta derruba A-T24 e A-T25; esticar o tempo derruba A-T24; gravar a chave sem confirmação derruba A-T26 |
| Compilar e instalar | **Verde**: o APK do `dev` compila com `androidx.biometric:biometric:1.1.0` e o `TravaPlugin`; `adb install -r` deu `Success` |
| Plugin no aparelho | **Verde** (Xiaomi 2311DRK48G, pela inspeção da WebView): `Capacitor.Plugins.Trava` existe, `disponivel()` devolve `true`, a subseção "Segurança" aparece em Listas com as duas chaves desligadas e a tela de bloqueio fica escondida com a trava desligada |
| A-T29 **[aparelho]** | **Verde, pelo relato do dono** (2026-11-12, Xiaomi 2311DRK48G): seguiu o roteiro de seis passos (ligar a trava com a digital, fechar e abrir, cancelar a janela, voltar depois de 1 minuto e antes, esconder nos apps recentes com tentativa de print, desligar a trava) e respondeu "funcionou". Não houve captura nem leitura pela inspeção nesse roteiro, e ele não detalhou passo a passo |
| A-T30 e A-T31 **[aparelho]** | **Pendentes**: remover a biometria do aparelho com a trava ligada, e a medição de layout (360 e 412 px, tema claro e escuro) |

### Decisões da fase 3a

- O plugin se chama `Trava` (`TravaPlugin.java`): `disponivel()`, `confirmar({titulo})` e `protegerJanela({ligar})`. Usa a biblioteca `androidx.biometric` (do próprio Android),
  aceita digital ou rosto de qualquer classe **ou a credencial do aparelho** (PIN, padrão ou senha) e não escreve log. Nenhum plugin de terceiros.
- "Sair da conta" desliga as duas chaves (R14, A-T27): `sessao.js` apaga `financas.trava` e `financas.recentes` junto com `financas.space`.
- Com a tela de bloqueio na frente, o voltar físico só minimiza o app (`nativo.js`); não mexe no que está por baixo.
- A janela do Android tira o app da frente; isso **não conta** como saída para o relógio de 1 minuto, e só uma pergunta fica aberta por vez.
- A escolha fica em `localStorage`, como o tema e o "esconder valores". Se o Android apagar os dados da WebView, a trava volta desligada. Aceito: a trava é só da tela (R23).

## Verificação no `dev`

1. Fazer o deploy do `template.yaml` no `financas-dev` e conferir o changeset e o status final da stack (guardrails, "Deploy"); a tabela e o login **não** podem ser substituídos.
2. Instalar o **APK de debug** do `dev` e o do `prod` (quando houver) no aparelho do dono e rodar os testes **[aparelho]**. Nas fases 1 a 3 só se gera APK; o AAB é da fase 5.
3. Com o Logs Insights do grupo `/aws/lambda/financas-api-dev`, conferir que as chamadas do app aparecem como as do navegador e que nenhuma linha traz token ou conteúdo.

## Fora do escopo

Push, assinatura de release e Play Store, build no CI, App Link por `https` (**volta na fase 5**, quando existir a chave de release; decisão do dono em 2026-11-11), iOS, uso sem internet, tela de versão do app, versão mínima exigida pelo servidor e qualquer mudança
de regra de negócio. O token de push e o canal FCM estão na fase 4 do plan.
