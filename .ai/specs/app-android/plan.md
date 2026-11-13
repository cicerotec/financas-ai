# Plan: app Android com Capacitor

Como construir o que o [`intent.md`](intent.md) pede. Cada fase é um PR pequeno, vai primeiro ao `dev` e deixa o app utilizável. Telas novas pedem demo aprovada
antes do código. O roteiro verificável das fases 1 a 3 está em [`spec.md`](spec.md); as demais ganham spec própria quando chegar a vez.

**Estado:** rascunho (2026-11-01). **Fase 1 implementada em 2026-11-11** (branch `feature/app-android-casca`), com o APK do `dev` **compilado, instalado e abrindo no aparelho**; os testes que pedem sessão (A-T6 e A-T7) passaram para o fim da fase 2 (ver "Fase 1: o que foi verificado" no `spec.md`). **Fase 2 implementada em 2026-11-12** (PR 419), com o login completo visto no aparelho; pendências de aparelho no `spec.md`. **Fase 3a (a trava por biometria) com código em 2026-11-12**; a 3b (splash nativo e ícone adaptativo) com código em 2026-11-12.

## Onde estamos hoje

| Fato | Onde | Consequência |
|---|---|---|
| Front estático, JS puro, sem bundler; `index.html` e `js/*.js` carregados por `<script>` | `web/` | O Capacitor empacota a pasta como está; plugins devem ser acessíveis sem `import` (a verificar na fase 1) |
| `config.js` e `config.dev.js` fora do git, com `apiUrl`, `clientId`, `loginDomain` (e `ambiente: "dev"`) | `web/config.example.js`, `.gitignore` | Cada ambiente do app precisa do seu config no pacote; o build escolhe qual |
| Login: Hosted UI + PKCE, `redirect_uri = location.origin + '/'`, PKCE em `sessionStorage`, token em `localStorage` (`financas.auth`) | `web/api.js` (`irParaLogin`, `token`, `sair`) | No app não existe `location.origin` útil, a página não "vai embora" e o processo pode ser morto enquanto o navegador está aberto |
| `UserPoolClient` aceita só as URLs do front em `CallbackURLs` e `LogoutURLs` | `template.yaml` | Precisa da URL de retorno do app, por ambiente |
| CORS da Function URL aceita `FrontOrigin` e o domínio do CloudFront | `template.yaml` (`Cors.AllowOrigins`) | A WebView do Capacitor tem origem própria (`https://localhost` por padrão): precisa entrar |
| Botão voltar é uma pilha no histórico do navegador | `web/js/voltar.js`, `padrao-de-telas.md` | No Android o voltar físico tem de cair nessa pilha e, na raiz, sair do app sem perder o estado |
| Avisos só pelo Telegram, atrás da interface `Avisador` | `backend/adapter/notificacao/`, `arquitetura.md` | O push entra como mais um canal, sem tocar o `service` |
| Nenhum dado de aparelho no DynamoDB | `specs/modelo-de-dados/` | O token de push do aparelho é **dado novo**: decisão do dono antes (fase 4) |
| Deploy por GitHub Actions e roles OIDC por ambiente | `.ai/deploy-github-actions.md`, `infra/github-oidc.yaml` | Permissão nova na infra exige atualizar as roles antes do merge |

## Estrutura proposta

```
app-android/                 projeto do Capacitor (código nosso, sem dado real)
  capacitor.config.json      appId, appName, webDir "www", androidScheme "https" (JSON, para nao precisar de TypeScript)
  package.json               só Capacitor e plugins (o front continua sem build)
  android/                   projeto gerado pelo Capacitor (vai para o git, menos build/ e local.properties)
  montar-www.js              copia web/ para www/ com o config do ambiente (chamado pelo script e testado em web/app-android.test.js)
  www/                       cópia gerada de web/ + config do ambiente (ignorado pelo git)
scripts/app-android.ps1      chama montar-www.js para -Ambiente dev|prod, roda cap sync e o Gradle (assembleDevDebug ou assembleProdDebug)
```

- O `web/` continua sendo a **única fonte das telas**. `www/` nunca é editado à mão.
- **`applicationId` decidido em 2026-11-11: `br.com.cicerotec.financas`** (o `dev` leva `.dev`). Os dois ambientes são **sabores do Gradle** (`prod` e `dev`, em `android/app/build.gradle`); o nome e o ícone marcado do `dev` ficam em `android/app/src/dev/res/`. Na fase 1 o ícone é o legado em PNG (de `web/icons/` e `web/icons/dev/`); o adaptativo é da fase 3.
- Dois ambientes = dois `applicationId` (`...financas` e `...financas.dev`, o `dev` com nome "Finanças DEV" e ícone marcado), para os dois ficarem instalados juntos,
  cada um com seus dados e sua sessão. O `config.js` certo entra em `www/` pelo script, não por sabor do Gradle, para o pacote ter um config só.
- **Assinatura:** o keystore de upload **nunca** vai para o git (o repositório é público). Fica no computador do dono, fora do repositório, com senha lida por
  `Read-Host -AsSecureString` (guardrails, "Segurança"), e o App Signing da Play guarda a chave de assinatura final.

## Fases (cada uma é um PR)

### Fase 1: a casca (o app abre o `web/` atual, no `dev`)
- Projeto Capacitor em `app-android/`, `webDir` apontando para `www/`, `androidScheme: "https"` (a origem da WebView passa a ser `https://localhost`).
- `scripts/app-android.ps1 -Ambiente dev` monta `www/` (copia `web/`, troca o `config.js` pelo do ambiente, tira `manifest` e o que só serve ao navegador, se atrapalhar),
  roda `npx cap sync android` e gera o **APK de debug** (formato para instalar direto no aparelho com `adb install` ou o Run do Android Studio; o AAB não instala sozinho).
- **CORS:** parâmetro novo `AppOrigin` (`https://localhost`) em `AllowOrigins` da Function URL, nos dois ambientes.
- Verificar três coisas e registrar o resultado: (a) `window.Capacitor.Plugins` expõe os plugins nativos sem bundler; (b) a WebView chama a API no `dev` sem erro de CORS;
  (c) o botão voltar físico cai na pilha de `voltar.js` e, na raiz, minimiza o app.
- Sem login nativo ainda: nesta fase o login pode continuar quebrado no app; o foco é provar a casca, o CORS e o build. Entrega: APK no aparelho abrindo as telas com a
  faixa "AMBIENTE DE TESTE".
- **O que o aparelho mostrou (2026-11-11):** sem login o app não exibe tela nenhuma, porque a primeira coisa que faz é ir ao Hosted UI, que recusa o retorno `https://localhost/`.
  A fase 1 fecha com o APK compilando, instalando e abrindo; ver as telas, a faixa, o CORS a partir do app e o botão voltar fica para o fim da fase 2.
- **Para compilar:** Node 22 ou mais, JDK 21 ou mais e a plataforma Android 36. `scripts/app-android.ps1` confere as versões e aceita `-NodeDir` e `-JavaHome` (ou as variáveis
  `APP_ANDROID_NODE` e `APP_ANDROID_JAVA_HOME`) quando o padrão da máquina é mais antigo.

### Fase 2: login nativo e token no armazenamento seguro
- **Hosted UI no navegador do sistema (Custom Tabs), nunca na WebView.** O app abre a URL de autorização; o Cognito devolve o `code` por um esquema próprio
  (`<applicationId>://auth`); o app recebe pelo evento de URL aberta, fecha a aba e troca o `code` pelo token com o `code_verifier`.
- **O PKCE sobrevive à morte do processo:** `verifier` e `state` vão para o armazenamento do app (não `sessionStorage`), porque o Android pode matar o app enquanto a
  aba de login está aberta.
- **Armazenamento do app: plugin nosso, em Java** (decisão do dono em 2026-11-11), sem dependência de terceiros: ler, gravar e apagar pares cifrados com uma chave do Android
  Keystore. Guarda o refresh token e o `verifier` e o `state` do PKCE. O único plugin oficial novo é o `@capacitor/browser`.
- **O token sai do `localStorage`:** o refresh token vai para o armazenamento seguro (Keystore) e o ID token fica em memória. Sair limpa os dois e abre o logout do
  Cognito na aba do sistema.
- **Extrair o login de `api.js`** para um módulo próprio e testável (`web/js/sessao.js`, no molde de `voltar.js`), com duas implementações atrás da mesma interface:
  navegador (como hoje) e app. `api.js` só chama `token()`, `sair()` e trata `401`. A PWA e o navegador não mudam de comportamento.
- `template.yaml`: `CallbackURLs` e `LogoutURLs` ganham `<applicationId>://auth` (e a versão `.dev`) por ambiente. Atualizar as roles do Actions se faltar permissão.
- **Retorno por esquema próprio nesta fase** (decisão do dono em 2026-11-11), protegido por PKCE e `state`. O App Link por `https` volta na fase 5.
- **Rede não é logout (só no app;** o navegador e a PWA seguem como hoje, decisão do dono em 2026-11-11**):** só `invalid_grant` (refresh token revogado ou vencido) leva ao login; falha de rede mantém a sessão e mostra o erro de sempre.
- Atualizar `security.md` (token fora do `localStorage` no app; esquema de retorno e o risco dele).

### Fase 3: biometria, ícone e acabamento
- **Tela de bloqueio é tela nova: demo aprovada antes do código** (celular primeiro, tema claro e escuro, o que a proposta faz perder). Padrão: ao abrir ou voltar de segundo
  plano depois de um tempo, mostra o bloqueio e pede a biometria do aparelho (com PIN do aparelho como alternativa); falhou, não entra.
- Opcional, ligado pela pessoa em um interruptor; a escolha fica **só no aparelho** (nenhum campo novo no DynamoDB) e some no logout.
- A trava cobre a interface; não cifra o token. Dizer isso na demo e em `security.md`.
- Ícone adaptativo (as imagens maskable já existem em `web/icons/`), splash e cor da barra de status seguindo o tema do app.
- Cada ajuste visual de padrão atualiza `padrao-de-telas.md` e as demos na mesma entrega (guardrails).

### Fase 4: push (FCM) como canal de aviso (spec própria)
- **Depende do item 34 (despachante de avisos), que vem inteiro antes** (decisão do dono em 2026-11-13). O plano original supunha o despachante pronto; ele não existe. Roteiro do item 34: [`../avisos-despachante/`](../avisos-despachante/intent.md).
- Novo adaptador em `backend/adapter/notificacao/` (`fcm.go`), registrado no despachante ao lado do Telegram. O `service` continua mandando conteúdo estruturado.
- **Onde guardar o token (decidido pelo dono em 2026-11-13, opção "A"):** um campo novo `pushTokens` (lista de textos, um token por aparelho) no **vínculo** `USER#<sub>` / `SPACE#<id>`, o mesmo item onde já está o `telegramChatId`.
  Vale por pessoa **dentro de cada espaço**, como o Telegram: quem quer o aviso dos dois espaços liga a chave nos dois; cada espaço é uma conta separada. Sair da conta remove o aparelho de **todos** os espaços onde ele está
  registrado (o servidor acha os espaços da pessoa pela consulta que já lista os espaços). Limite de 5 aparelhos por pessoa e espaço; só a própria pessoa registra (o `sub` vem do token); o token nunca vai ao log; o token que o
  Firebase recusar é apagado pela rotina. **O nome e o tipo do campo (`pushTokens`, conjunto de textos) são confirmados na spec da fase.** Foram descartados o item por aparelho e o item por pessoa (este por juntar os espaços).
- **Conteúdo da notificação (decidido em 2026-11-13): só a contagem**, por exemplo "2 atrasados, 1 vence hoje", sem descrição, valor nem meio de pagamento. Os detalhes ficam dentro do app, atrás da trava. **Decidido: não mostra o nome do espaço** (apareceria na tela bloqueada). O mesmo aparelho em dois espaços recebe **uma** notificação, com a contagem somada.
- **Plugin: nosso, em Java** (decidido em 2026-11-13), como o `Cofre` e a `Trava`; sem plugin de terceiros. Android 13 ou superior pede permissão de notificação em tempo de execução; explicar o porquê antes de pedir.
- **A chave "Receber avisos neste aparelho"** é uma tela nova (demo antes): fica em Listas, junto do cartão do Telegram, dentro de cada espaço, só no app. Sem token, o despachante simplesmente não entrega (já é a regra).
- **Credencial do Firebase** fica no SSM, nunca no git, como o token do Telegram, guardada por script com `Read-Host -AsSecureString` (nunca no chat nem na linha de comando). O arquivo de configuração do app
  (`google-services.json`) tem dado do projeto e **fica fora do git**, como o `config.js`. Como a Lambda se autentica no Firebase e o que o app precisa **serão conferidos na documentação** antes da spec da fase.
- A fase 4 **não** inclui avisar a esposa das contas que o dono cria: quem recebe o aviso continua sendo o criador do lançamento e o dono do espaço.

> **No backlog (decisão do dono em 2026-11-13):** as fases 1 a 4 são o item 54, que se conclui com o push; as fases 5, 6 e 7, abaixo, são o item 64. O roteiro continua sendo este arquivo.

### Fase 5: release e Play Store (spec própria, em parte processo)
- **Formatos:** o artefato de entrega à Play é o **AAB assinado** (a Play exige AAB para apps novos e gera os APKs de cada aparelho). O APK de debug fica para o ciclo
  de desenvolvimento (fases 1 a 3) e o APK de release assinado, só como opção de distribuição direta enquanto a conta da Play não estiver pronta.
- Gerar o **AAB assinado** (keystore de upload local; Play App Signing). Versão do app (`versionCode` e `versionName`) subindo a cada envio, ligada às tags de
  [`fluxo-de-versoes.md`](../../fluxo-de-versoes.md).
- **App Link por `https` no retorno do login** (decisão do dono em 2026-11-11: volta nesta fase). Com a chave de release definida, o retorno deixa de ser o esquema próprio e
  passa a ser um endereço `https` do CloudFront de cada ambiente, que só o app assinado por essa chave recebe: publicar `/.well-known/assetlinks.json` com a impressão digital
  da chave (a de upload e a do Play App Signing), reservar o caminho de retorno no front, trocar `CallbackURLs` e `LogoutURLs` e o `intent-filter` (`autoVerify`). Verificar no
  aparelho que o retorno pelo Custom Tabs abre o app e, se a verificação do domínio falhar, o que a pessoa vê. Entra na spec desta fase.
- **Teste interno** no Play Console para as duas pessoas. Se a conta for pessoal e a regra de teste fechado valer, abrir o teste fechado cedo, porque o relógio corre.
- Exigências da loja: política de privacidade pública (a mesma da abertura multiusuário), formulário de Segurança dos Dados, declaração de app financeiro, classificação
  de conteúdo, ícone e capturas de tela, versão mínima e alvo de API vigentes. **Confirmar tudo no Play Console na hora**, porque as regras mudam.
- Build no GitHub Actions (JDK, Android SDK, secrets do keystore) é opcional e só se justifica quando o processo manual incomodar; permissão nova exige atualizar as roles.
- Cada mudança de tela é uma nova versão na Play; combinar com o dono quando vale publicar (por tag, não por commit).

### Fase 6: produção aberta
- Só depois das fases 1 a 5 de [`../app-multiusuario/plan.md`](../app-multiusuario/plan.md) (MFA, limite de taxa, cadastro, espaço próprio, exportar e excluir conta), da
  política de privacidade e, de preferência, da cifragem dos dados. A Play exige que quem cria conta dentro do app consiga **pedir a exclusão** dela.
- Promover do teste interno ao fechado e à produção, com poucas pessoas primeiro.

### Fase 7: uso sem internet (futuro, fora deste plano)
- Segue [`offline-e-sincronizacao.md`](../../offline-e-sincronizacao.md): IndexedDB (agora confiável), projeção do saldo no aparelho, fila e tela de divergências. Tem demo
  e spec próprias e depende do gatilho "muitos usuários querendo offline".

## Premissas (fase 2)

Exigência do `guardrails.md` ("Roteiros"). Em 2026-11-11 um agente independente, sem o contexto de quem escreveu o roteiro, leu o `spec.md`, o `plan.md`, o `intent.md`, o
`guardrails.md`, o `security.md` e o código (`web/api.js`, `web/index.html`, `abertura.js`, `usuario.js`, `voltar.js`, `nativo.js`, `backend/auth/token.go`, o manifest, o
`build.gradle`, o Capacitor 8.6.0 instalado, entre outros) e respondeu à pergunta: *"que regra desta feature só funciona por causa de outra, e o que acontece se essa outra
deixar de valer?"*. A resposta, resumida; cada premissa tem a sua regra no `spec.md` (R18a a R18o).

| # | A regra só funciona porque... | Se deixar de valer | Regra |
|---|---|---|---|
| P1 | **A página vai embora no login e no logout** (`web/api.js:35`, "a pagina vai embora"; `web/js/usuario.js`, "A pagina vai embora sozinha"). É isso que zera `ESPACO`, papel, caches e ouvintes | No app a WebView continua viva: chamadas presas para sempre ("Salvando…"), a abertura parada em "Está demorando mais que o normal" sem botão se a pessoa fecha a aba, dados da pessoa anterior na tela depois do Sair | R18a |
| P2 | **`token()` olha o retorno antes de decidir logar** (`web/api.js:38-46`), e o retorno some da URL. No Capacitor, `appUrlOpen` só dispara com o app vivo; na partida a frio a URL fica em `getLaunchUrl()` e **nunca é limpa** | Partida a frio abre outro login e perde o `code` (A-T20 falha); ou toda recarga reapresenta o retorno já usado e vira laço de login | R18b |
| P3 | **Chamadas simultâneas a `token()` são inofensivas** porque o `localStorage` é síncrono e o último `location.href` vence. A subida faz três leituras em paralelo, e o timer e o `visibilitychange` rodam justamente quando a pessoa volta da aba de login | No app: vários refreshes, várias abas do sistema e o `state` guardado deixa de bater com a aba aberta | R18c |
| P4 | **A escolha por `isNativePlatform()` e o plugin registrado.** O molde (`nativo.js`) devolve `false` em silêncio quando falta o plugin, e a `MainActivity` está vazia | O app cai no fluxo do navegador sem erro: token em `localStorage` e retorno `https://localhost/` | R18d |
| P5 | **O R18 conta com ninguém mais logando.** O padrão do Capacitor (`loggingBehavior` `debug`) liga o log no APK de debug e imprime argumentos e resultados de cada plugin | O refresh token e a URL de retorno com o `code` vão ao logcat, e o teste A-T17 passa verde | R18e |
| P6 | **O esquema de retorno por sabor.** O manifest só tem o filtro de abertura; `custom_url_scheme` no `dev` vale o do `prod`; o front não sabe o `applicationId` | O `dev` registra o esquema do `prod`; com os dois instalados o retorno cai no app errado. Pôr os dois URLs nos dois clientes tira a barreira que denunciaria a troca | R18f |
| P7 | **"Só `invalid_grant` leva ao login" conta com sempre haver uma saída.** A tela de erro da abertura só mostra "Sair da conta" para conta sem espaço; para rede só há "Tentar de novo" | Refresh com outro `400`, falha do plugin ao ler, ou troca do `code` que falha por rede (o R12 apaga o `verifier` e o R15 proíbe reabrir): sem sessão e sem login | R18g, R18a |
| P8 | **`logout_uri` e `redirect_uri` são o mesmo URL** e hoje o retorno do logout reabre o login; apagar e navegar são síncronos | O retorno do logout chega sem `state` (R11 manda ignorar, R12 manda reiniciar). Sem rede o cookie do Hosted UI fica e o próximo "Entrar" volta sem senha | R18h |
| P9 | **O ganho do Keystore conta com não haver script injetado.** O plugin expõe "ler" ao JavaScript, o app carrega script de terceiro sem `integrity` e não há CSP | Um script na WebView lê o refresh token como leria o `localStorage`. O que muda de fato é o arquivo em disco e o backup | R18i |
| P10 | **`launchMode="singleTask"` traz o app por cima da aba**, e o navegador aceita redirecionar ao esquema próprio sem toque | O voltar mostra a página do Cognito; login com sessão ativa ou o retorno do logout não voltam ao app. Só se vê no aparelho | R18j |
| P11 | **`allowBackup="false"` basta e a chave do Keystore não some** | Dado cifrado restaurado em outro aparelho ou chave invalidada: a leitura falha e cai no P7 | R18g |
| P12 | **O refresh não devolve refresh token novo** (`web/api.js:15` mantém o antigo; o template não liga rotação) | Gravar `undefined` depois de um refresh bom apaga a sessão | R18k |
| P13 | **`401` é só token vencido.** Qualquer falha de validação vira `401`, inclusive o servidor não conseguir buscar as chaves do Cognito | Refresh bom e `401` de novo: ir ao login desloga a pessoa por falha do servidor | R18l |
| P14 | **A troca do `code` sai de `https://localhost`.** O R5 só trata do CORS da Function URL | Se o Cognito não devolver o cabeçalho na resposta de erro, um refresh vencido parece falha de rede. Pelo servidor já foi visto que devolve; falta ver do app | R18j |
| P15 | **O mesmo `UserPoolClient` serve web e app** | Não dá para encerrar só as sessões do app; mudar `CallbackURLs` mexe no login da web e a guarda do deploy é o que impede a substituição | R18m |

Lacunas do próprio spec que a revisão apontou e já foram tratadas: a diferença entre R11 e R12 (R18b), o retorno do logout sem regra (R18h), o R17 contra este plan (R18f), os casos
que o R15 não cobria (R18g), o `401` repetido (R18l), `limpar()` sem regra e o que o Sair apaga (R18n), o A-T12 e o A-T17 provando menos que a regra, e os testes que carregam só o
`api.js` (R18n). O revisor leu a `develop` de antes do fecho da fase 1 e por isso registrou o APK como não compilado; ele já compila e instala (ver o `spec.md`).

**Decisões do dono em 2026-11-11** (marcadas **(decidido)** no spec): a tela "sem sessão" na Abertura entra, com demo antes do código (R18a); qualquer `400` do refresh leva ao
login (R18g); o plugin fica simples e o refresh em Java é endurecimento para depois (R18i); o Sair apaga a sessão e o espaço escolhido (R18n); e `allowBackup="false"` (R9a).
As regras R18a a R18o foram redigidas por quem escreveu o roteiro a partir das sugestões do revisor; o texto delas não passou por revisão independente.

## Riscos

| Risco | Resposta |
|---|---|
| Esquema próprio de retorno pode ser registrado por outro app instalado | O `code` sozinho não vale: o atacante não tem o `code_verifier` (PKCE) e o `state` é conferido. O que sobra é atrapalhar o login ou mostrar uma tela falsa. O App Link por `https` fecha isso e entra na fase 5 (decidido em 2026-11-11) |
| Android mata o processo durante o login e o PKCE se perde | `verifier` e `state` no armazenamento do app, não em memória nem em `sessionStorage` |
| CORS aceita `https://localhost` e qualquer página com essa origem poderia chamar a API | CORS não é controle de acesso (`security.md`): a barreira continua sendo o JWT validado na Lambda. A origem só existe dentro do app ou na máquina de quem desenvolve |
| Plugins não aparecem sem bundler | Verificado na fase 1; se falhar, o menor passo de build possível e a decisão volta ao dono |
| Token roubado de aparelho com root | O Keystore reduz o risco, não o elimina. O token tem 60 min (refresh de 30 dias); MFA do item 49 limita o estrago de uma senha vazada |
| Keystore de upload perdido ou vazado | Fora do git, com cópia segura; o App Signing da Play permite redefinir a chave de upload |
| `applicationId` errado no primeiro envio | É permanente: decidir antes da fase 1 (o `dev` usa sufixo) |
| App dentro do pacote fica desatualizado em relação ao backend | A API é compatível para trás por regra; versões muito antigas do app são tratadas por uma versão mínima no servidor (decisão futura, fase 5) |
| Regras da Play mudam | Conferir o Play Console na fase 5; este plan não fixa números de exigência |
| Dois aparelhos, dois `applicationId`: confusão entre `dev` e `prod` | Nome "Finanças DEV", ícone marcado e a faixa laranja que o app já mostra |

## Ordem recomendada

Decidir o `applicationId` e a conta da Play, depois as fases 1, 2 e 3 em sequência (cada uma usável no `dev`). A fase 4 espera a conversa sobre onde guardar o token de
push. A fase 5 pode começar em paralelo à 4, porque o teste interno só pede um build assinado. A 6 espera o multiusuário.

## Decisões que o plan fixou e o dono pode contestar

1. **Capacitor com o `web/` empacotado**, sem mudar telas nem regras.
2. **Login pelo navegador do sistema**, com PKCE e retorno por esquema próprio; o `code` nunca passa pela WebView.
3. **O token sai do `localStorage` no app** e vai para o Keystore; o login vira um módulo testável com duas implementações.
4. **`dev` e `prod` coexistem** por `applicationId` distinto; o config entra no pacote pelo script de build.
5. **A biometria é local e opcional**; nenhuma preferência dela vai para o DynamoDB.
6. **Nenhuma chave nova no DynamoDB** nas fases 1 a 3; o token de push (fase 4) é decisão à parte.
7. **Push como canal do despachante**, sem o `service` saber que ele existe.
8. **Keystore e credenciais fora do git**, sempre.
