# Plan: app Android com Capacitor

Como construir o que o [`intent.md`](intent.md) pede. Cada fase é um PR pequeno, vai primeiro ao `dev` e deixa o app utilizável. Telas novas pedem demo aprovada
antes do código. O roteiro verificável das fases 1 a 3 está em [`spec.md`](spec.md); as demais ganham spec própria quando chegar a vez.

**Estado:** rascunho para aprovação (2026-11-01). Nada implementado.

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
  capacitor.config.ts        appId, appName, webDir "www", androidScheme "https"
  package.json               só Capacitor e plugins (o front continua sem build)
  android/                   projeto gerado pelo Capacitor (vai para o git, menos build/ e local.properties)
  www/                       cópia gerada de web/ + config do ambiente (ignorado pelo git)
scripts/app-android.ps1      monta www/ para -Ambiente dev|prod, roda cap sync e o Gradle
```

- O `web/` continua sendo a **única fonte das telas**. `www/` nunca é editado à mão.
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

### Fase 2: login nativo e token no armazenamento seguro
- **Hosted UI no navegador do sistema (Custom Tabs), nunca na WebView.** O app abre a URL de autorização; o Cognito devolve o `code` por um esquema próprio
  (`<applicationId>://auth`); o app recebe pelo evento de URL aberta, fecha a aba e troca o `code` pelo token com o `code_verifier`.
- **O PKCE sobrevive à morte do processo:** `verifier` e `state` vão para o armazenamento do app (não `sessionStorage`), porque o Android pode matar o app enquanto a
  aba de login está aberta.
- **O token sai do `localStorage`:** o refresh token vai para o armazenamento seguro (Keystore) e o ID token fica em memória. Sair limpa os dois e abre o logout do
  Cognito na aba do sistema.
- **Extrair o login de `api.js`** para um módulo próprio e testável (`web/js/sessao.js`, no molde de `voltar.js`), com duas implementações atrás da mesma interface:
  navegador (como hoje) e app. `api.js` só chama `token()`, `sair()` e trata `401`. A PWA e o navegador não mudam de comportamento.
- `template.yaml`: `CallbackURLs` e `LogoutURLs` ganham `<applicationId>://auth` (e a versão `.dev`) por ambiente. Atualizar as roles do Actions se faltar permissão.
- **Rede não é logout:** só `invalid_grant` (refresh token revogado ou vencido) leva ao login; falha de rede mantém a sessão e mostra o erro de sempre.
- Atualizar `security.md` (token fora do `localStorage` no app; esquema de retorno e o risco dele).

### Fase 3: biometria, ícone e acabamento
- **Tela de bloqueio é tela nova: demo aprovada antes do código** (celular primeiro, tema claro e escuro, o que a proposta faz perder). Padrão: ao abrir ou voltar de segundo
  plano depois de um tempo, mostra o bloqueio e pede a biometria do aparelho (com PIN do aparelho como alternativa); falhou, não entra.
- Opcional, ligado pela pessoa em um interruptor; a escolha fica **só no aparelho** (nenhum campo novo no DynamoDB) e some no logout.
- A trava cobre a interface; não cifra o token. Dizer isso na demo e em `security.md`.
- Ícone adaptativo (as imagens maskable já existem em `web/icons/`), splash e cor da barra de status seguindo o tema do app.
- Cada ajuste visual de padrão atualiza `padrao-de-telas.md` e as demos na mesma entrega (guardrails).

### Fase 4: push (FCM) como canal de aviso (spec própria)
- Novo adaptador em `backend/adapter/notificacao/` (`fcm.go`), registrado no despachante ao lado do Telegram. O `service` continua mandando conteúdo estruturado.
- **O token do aparelho precisa ser guardado** (por pessoa e por aparelho): isso é **item novo no DynamoDB**, então a forma (campo no vínculo `USER#<sub>/SPACE#<id>`, item novo
  ou parâmetro) é **decisão do dono antes de qualquer código**. A spec desta fase abre com essa conversa.
- Android 13 ou superior pede permissão de notificação em tempo de execução; explicar o porquê antes de pedir. Cada pessoa liga ou desliga o canal; sem token, o despachante
  simplesmente não entrega (já é a regra).
- Conteúdo da notificação: **sem valor na tela bloqueada** como padrão (decisão a confirmar; ver `privacidade-e-cifragem.md`, "Avisos no Telegram").
- Credencial do Firebase fica no SSM, nunca no git, como o token do Telegram.

### Fase 5: release e Play Store (spec própria, em parte processo)
- **Formatos:** o artefato de entrega à Play é o **AAB assinado** (a Play exige AAB para apps novos e gera os APKs de cada aparelho). O APK de debug fica para o ciclo
  de desenvolvimento (fases 1 a 3) e o APK de release assinado, só como opção de distribuição direta enquanto a conta da Play não estiver pronta.
- Gerar o **AAB assinado** (keystore de upload local; Play App Signing). Versão do app (`versionCode` e `versionName`) subindo a cada envio, ligada às tags de
  [`fluxo-de-versoes.md`](../../fluxo-de-versoes.md).
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

## Riscos

| Risco | Resposta |
|---|---|
| Esquema próprio de retorno pode ser registrado por outro app instalado | O `code` sozinho não vale: o atacante não tem o `code_verifier` (PKCE) e o `state` é conferido. App Link por `https` fica como endurecimento, decisão do dono |
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
