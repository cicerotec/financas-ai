# Intent: MFA opcional por aplicativo autenticador, com lista de códigos de recuperação

Este documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md) e as regras verificáveis em [`spec.md`](spec.md).

**Estado (2026-11-19):** Backend implementado e mergeado em 2026-11-19 (PRs 455, 456, 457 e 461); **o front (Fase 4) e a verificação no `dev` (Fase 5) ainda não foram feitos**, então hoje ninguém liga o MFA pelo app. É a Fase 1 do item 49 do BACKLOG ([`app-multiusuario`](../app-multiusuario/intent.md)).

## O problema

O login é só e-mail e senha (o `UserPool` do `template.yaml` não define `MfaConfiguration`). Uma senha vazada basta para ver todos os dados financeiros da família.
Está registrado em [`security.md`](../../security.md) como ponto em aberto de prioridade alta.
O único MFA que existe hoje é o da sessão AWS do dono (`scripts/aws-mfa.ps1`), que é outra coisa.

Ligar o MFA cria um segundo problema: **quem perde o acesso ao autenticador fica sem entrar.** O autenticador é onde o segredo do TOTP mora (o app no celular, o gerenciador de
senhas ou o programa no computador); perder o acesso é perder o celular, trocar de computador sem levar o segredo, apagar o app ou formatar a máquina. **Tem que haver uma saída
que a própria pessoa use, sem o dono.**

## O que queremos

1. **MFA por aplicativo autenticador (TOTP), sem SMS**, nos dois ambientes (`dev` e prod), configurado no `template.yaml`.
2. **Opcional para todos**, inclusive o dono: cada pessoa liga e desliga o seu. Não é obrigatório.
3. **Login na tela do Cognito, no Managed login, com o visual do app (logo, cores e fundo).** Depois que a pessoa liga o MFA, o Cognito pede o código de 6 dígitos no login. A senha continua nunca passando
   pelo nosso JavaScript, o que mantém a garantia do app Android (a senha só é digitada na aba do sistema).
4. **Uma tela nossa em Listas, Segurança da conta**, para ligar e desligar o MFA: mostra o QR code e a chave, confere o primeiro código e liga; ou desliga. Ela chama as APIs do Cognito
   (`AssociateSoftwareToken`, `VerifySoftwareToken`, `SetUserMFAPreference`) com o access token da própria pessoa. O Cognito guarda o segredo e confere o código; nós só desenhamos a tela.
5. **Lista de códigos de recuperação, a saída para desligar o MFA.** Ao ligar o MFA, a pessoa recebe uma **lista de códigos de uso único**, mostrada **uma única vez**. Se perder o acesso
   ao autenticador, ela digita **um código da lista** numa tela nossa, sem login, e o MFA dela **desliga**; depois entra só com e-mail e senha e pode ligar de novo, com uma lista nova.
   **O código sozinho não dá entrada**: só desliga o MFA, e a entrada continua exigindo a senha pelo login do Cognito.
6. **A frase que a pessoa vê, em destaque, junto da lista, e confirma que leu antes de o MFA ficar ligado:** **"SE PERDER A LISTA, PERDE O ACESSO."** Mais: a lista precisa ser guardada num
   lugar seguro, **separado do autenticador** (um gerenciador de senhas diferente, papel em local protegido, cofre). Perder o acesso ao autenticador **e** a lista significa, **para a pessoa e para o app, perder a conta**,
   mesmo sabendo a senha: o app não oferece nenhum outro caminho, e a tela **não promete nada além disso** (nem cita o dono nem um "último recurso"). O dono, por ser administrador da conta AWS, **pode** desligar o MFA de uma
   pessoa por fora do app, pelo console da AWS, como último recurso (decisão D1, abaixo); isso **não é uma promessa feita a quem usa o app**, e o passo a passo fica **fora do git** (o repositório é público).
7. O backend **não muda a validação do token**: continua validando só o ID token, como em `auth/token.go`. A geração da lista tem uma rota autenticada nova, e o uso de um código tem uma rota
   pública nova (a pessoa que perdeu o autenticador não tem sessão); as duas ganham ação própria em `auth/authz.go`.

## Decisões já tomadas (dono, 2026-11-18 e 2026-11-19)

- MFA **opcional para todos**, não obrigatório.
- **TOTP**, sem SMS (o SMS é cobrado à parte pelo SNS e tem troca de chip e interceptação como pontos fracos).
- **A recuperação que o app oferece é da própria pessoa, sem o dono**, e é a **lista de códigos**. Não é por código no e-mail (vale tanto quanto a conta de e-mail, e quem a invade e tem a senha entra), não é por
  perguntas de segurança (respostas adivinháveis; quase ninguém preenche) e **não é por prazo**.
- **Tela do Cognito, não tela de login própria**: uma tela própria faria a senha passar pelo nosso código e exigiria habilitar fluxos de senha no `UserPoolClient`.
- **O login passa para o Managed login nesta entrega** (dono, 2026-11-19, depois dos testes no `dev`). O pool `financas` já está no plano **Essentials** (visto no console em 2026-11-18); o
  `financas-dev` também. O domínio hoje está em "UI hospedada (clássica)" e passa à versão 2, com um estilo de marca (logo, cores e fundo) por cliente. O layout e o fluxo continuam sendo do Cognito.
- O login normal **não muda**; o escopo `aws.cognito.signin.user.admin` é pedido só num segundo login, ao abrir a seção do MFA (decisão confirmada em 2026-11-19, ver `plan.md`).
- **Decisões da revisão (dono, 2026-11-19; detalhe em `plan.md`, "Decisões do dono"):**
  - **D1:** o texto de impossibilidade sai. O app não oferece caminho para quem perdeu o autenticador e a lista; o dono pode, por fora do app, desligar o MFA pelo console como último recurso, e o passo a passo fica fora do git. A frase da tela é regra de produto.
  - **D2:** registrar o risco (um script injetado na seção do MFA pode ligar o MFA com o autenticador do atacante e deixar a vítima sem saída) e o último recurso do dono (D1). Fica em `security.md`.
  - **D3:** avisar a pessoa quando o MFA mudar, **pelo canal que ela já tem ligado** (Telegram ou push), sem prometer entrega a quem não tem.
  - **D4:** como a pessoa guarda a lista decide-se na demo da Fase 1; a inclinação é Copiar tudo (gesto explícito) e Imprimir.
  - **D5:** se a seção mostra que há lista válida (um `GET`) ainda não está decidido; a inclinação é incluir.
  - **D6:** decidido na conversa do DynamoDB (2026-11-19): HMAC-SHA256 com o `listaId` como chave, sem pepper (código de 100 bits).

## Dado novo no DynamoDB (conversa feita em 2026-11-19)

A lista de códigos precisa ser guardada, **só como hash** (nunca o código). Isso exige **item ou campo novo**, e o guardrails manda não criar, renomear nem remover chave ou campo sem discutir com o dono.
**Decidido na conversa (detalhe em `spec.md`, R36):** **dois** itens por pessoa em `USER#<sub>` (`MFACODIGOS#ATIVA` e `MFACODIGOS#PENDENTE`), só hashes (HMAC-SHA256 com o `listaId` como chave, sem pepper), 10 códigos de 20 caracteres por lista, pendente de 1 hora sem TTL, e o limite de tentativas **por par (pessoa, origem), em memória**, sem item no DynamoDB. **Nada é criado na tabela até a Fase 3.**


## Segurança da rota de recuperação (requisitos)

A pessoa que perdeu o autenticador **não tem token**, então a rota é pública e é o ponto mais exposto desta entrega:
- **Limite de tentativas por pessoa e por origem**, com teste; força bruta não pode compensar (códigos longos e aleatórios, hash adequado à entropia no servidor).
- **Resposta idêntica** para e-mail sem conta, sem MFA ou com código errado, para a rota não revelar quem tem conta.
- **Cada código vale uma vez**, com teste que prove isso; a lista nova nasce **pendente** e só substitui a anterior depois de a pessoa a confirmar (para nunca ficar sem saída); usar um código desliga o MFA e invalida o restante da lista.
- **Nunca logar** código, e-mail nem corpo (regra de `security.md`).
- A rota desliga o MFA **só da pessoa dona do código**, e a permissão de IAM da Lambda para isso é **mínima** (só `AdminSetUserMFAPreference` e `AdminGetUser`, para achar a pessoa pelo e-mail, sobre o próprio pool). Permissão nova na infra exige atualizar as roles do Actions antes do merge.
- Um código vazado só desliga o MFA; a senha continua sendo exigida no login. Quem tem a lista e não a senha **não entra**.

## Limites (o que fica de fora)

- **Perder o autenticador e a lista ao mesmo tempo:** **o app não oferece caminho** (item 6): para quem usa, a conta se perde, mesmo sabendo a senha. O dono pode, por fora do app, desligar o MFA pelo console da AWS como último recurso (D1); **não é suporte, não é promessa e não aparece na tela**, e o passo a passo fica fora do git. Por isso a frase em destaque é parte da entrega.
- Um login com **layout ou textos próprios**: o Managed login só aceita marca (logo, cores, fundo, modo escuro); o layout e o fluxo são do Cognito. **Por isso a recuperação não está no login:** a entrada é a tela de entrada do app ("Entrar" e "Perdi o autenticador"), que o **navegador e o PWA passam a mostrar** antes do login, como o app Android já faz (decisão do dono, 2026-11-19; custo: um toque a mais para entrar pelo navegador).
- Login social, SMS, MFA por e-mail do Cognito, perguntas de segurança, recuperação por e-mail e MFA obrigatório para todos.
- Quem **não liga** o MFA continua com só senha: opcional significa que uma senha vazada ainda basta para ele.
- **A sessão do login do Cognito no navegador do aparelho** (cookie, cerca de 1 hora, não confirmado no pool) pode devolver o login sem senha e sem código; sair do app não a encerra. O MFA protege o login novo, não o
  reaproveitamento dessa sessão. É a mesma limitação que o `security.md` já registra para "sair do app".

## Custo

Zero no Cognito. O TOTP não tem cobrança por uso em nenhum plano; o Cognito cobra por usuário ativo no mês (MAU), com 10.000 grátis no Essentials. As rotas novas rodam na Lambda que já existe, e o item
novo no DynamoDB ocupa a tabela de 5/5 RCU/WCU: poucas leituras e gravações por pessoa, só ao gerar a lista e ao usar um código.

## O que foi testado no `dev` (2026-11-19)

**Primeira rodada, no UI clássico** (antes de o dono decidir pelo Managed login; o Managed login foi testado depois, na seção seguinte, com o mesmo resultado). Testado à mão no pool do `dev` (`financas-dev`, plano **Essentials**), com dados fictícios e um usuário de teste. Onde esta seção diz "Hosted UI", leia "o login do Cognito no UI clássico". Nada disto está no `template.yaml` ainda, e **o `dev` foi
devolvido ao estado original ao fim dos testes** (ver "Estado do `dev` depois dos testes").

**Confirmado**
- O pool aceita `MfaConfiguration: OPTIONAL` com o TOTP habilitado (`set-user-pool-mfa-config`).
- Com um token de login direto, `AssociateSoftwareToken`, `VerifySoftwareToken` (com o autenticador do celular) e `SetUserMFAPreference` funcionam e ligam o MFA do usuário.
- Com o MFA ligado, o Cognito devolve o desafio `SOFTWARE_TOKEN_MFA` pela API, e **o login do Hosted UI clássico pede o código de 6 dígitos** depois da senha, no navegador. Um usuário sem MFA continua entrando só com e-mail e senha.
- **No app Android do `dev`** (aba do sistema): o login pede senha e código e volta ao app logado. Voltar na aba **antes da senha** deixa o app sem sessão; voltar **depois da senha e antes do código** também deixa o app sem sessão, e um novo "Entrar" recomeça. Nenhum dos dois casos abriu o app logado sem o código.
- O pool do `dev` não tem dispositivo lembrado (`DeviceConfiguration` nulo) nem `UserPoolAddOns`; o único primeiro fator é a senha. A recuperação de senha é por e-mail e depois telefone; sem telefone, na prática só por e-mail.
- **Com o token do login do Hosted UI** (não o do login direto): um script de teste fez o login pedindo o escopo `aws.cognito.signin.user.admin`, capturou o código num ouvinte em `http://localhost:8080/` (URL de retorno que o cliente do `dev` já tem) e usou o access token nas APIs: `GetUser` devolveu o estado do MFA, `SetUserMFAPreference` **desligou** o MFA, e depois `AssociateSoftwareToken`, `VerifySoftwareToken` e `SetUserMFAPreference` **cadastraram e religaram** o TOTP, com o autenticador do celular. A própria pessoa desliga e liga o seu MFA.
- A resposta do endpoint de token **não traz o campo `scope`**; o escopo vale porque as APIs aceitaram o token. O front só precisa pedir o escopo novo (hoje `scope: 'openid email'`, `web/js/sessao.js:27`).
- Para esses testes o cliente do `dev` teve, à mão, o escopo marcado no console e o `ALLOW_USER_PASSWORD_AUTH` habilitado para o login direto. Ambos já foram removidos; o escopo entra pelo `template.yaml`, na Fase 2 do plan, como "permitido" no cliente.

**Descoberto no teste (vale para o plano)**
- O login do Cognito guarda uma **sessão própria por cookie no navegador do aparelho** (no UI clássico e também no Managed login). Com ela ativa, o Cognito mostra "continuar com este usuário ou trocar" e pode devolver o login **sem pedir senha nem código**.
- Um teste com o login do navegador pareceu "não pedir o código" porque foi feito com o usuário **sem** MFA. Em teste de MFA, confirme sempre qual usuário foi ligado (console do Cognito, Usuários, nome de usuário = `sub`).

**Ainda não provado**
- Que o segundo login **força a senha de novo** no Managed login (`prompt=login` ou `max_age`): é a premissa P2 do plan.
- O login do **app Android** com o escopo novo no pedido. A diferença é só uma palavra a mais no endereço, sem efeito esperado; **não se faz um teste à parte**, acontece na verificação da implementação.
- Que a mudança no `template.yaml` (`MfaConfiguration` e escopo do cliente) **não substitui o pool nem o cliente**: só o changeset do deploy mostra.
- Que a Lambda comum consegue desligar o MFA de uma pessoa (`AdminSetUserMFAPreference`) com permissão mínima.
- **Confirmado no `dev` em 2026-11-19 (lido do terminal do dono, sessão com MFA, só leitura, sem imprimir e-mail):** `AdminGetUser` aceita o **e-mail como `Username`** e devolve a pessoa certa (o `ListUsers` filtrado não é necessário). **A busca diferencia maiúsculas:** o mesmo e-mail em maiúsculas não achou a conta, e o pool não define `UsernameConfiguration.CaseSensitive` (o padrão do CloudFormation é diferenciar). O `dev` tem 2 usuários e **nenhum** e-mail com maiúscula. **Produção: conferido pelo dono em 2026-11-19: todos os e-mails de produção estão em minúsculas (pendência fechada).** Consequência: normalizar o e-mail para minúsculas (R29) só acha quem foi cadastrado em minúsculas; convite novo precisa gravar o e-mail em minúsculas, e o `CaseSensitive` do pool não se muda sem substituí-lo.
- Que o navegador e o WebView, a partir do CloudFront, chamam o Cognito (`cognito-idp.<região>.amazonaws.com`) por `fetch` com `Content-Type: application/x-amz-json-1.1` (CORS): os testes acima foram feitos por script.
- Que o `POST /oauth2/revoke` aceita o cliente público com o refresh do segundo login, e que o access token já emitido continua valendo.
- Quantos erros de `VerifySoftwareToken` o Cognito tolera antes de recusar por excesso de tentativas (não está provado; o spec não afirma o número).
- Que o tempo de resposta da rota pública de recuperação é próximo para e-mail com e sem conta (medido no `dev`).
- Como o "esqueci a senha" do Cognito convive com o MFA ligado; como gerar o QR code no front sem biblioteca pesada; o PWA instalado.

**Estado do `dev` depois dos testes (devolvido ao original em 2026-11-19, conferido por leitura)**
- Pool: `MfaConfiguration: OFF`.
- Cliente: `AllowedOAuthScopes` só `email` e `openid`; `ExplicitAuthFlows` só `ALLOW_REFRESH_TOKEN_AUTH`. O template **não define** `ALLOW_USER_PASSWORD_AUTH` e **não deve passar a defini-lo**.
- Usuário de teste: sem MFA preferido e sem MFA ligado. O segredo do TOTP dele continua registrado e **desativado** no Cognito (sem efeito com o pool em `OFF`); não foi removido.
- O pool de produção (`financas`) nunca foi tocado.
- O primeiro deploy da Fase 2 deve mostrar `MfaConfiguration` e o escopo do cliente como mudança de propriedade.
- **Lição do primeiro deploy (2026-11-19):** o **estilo de marca de teste** (`create-managed-login-branding`) **ficou no `dev`** depois dos testes, e o CloudFormation recusou criar o dele (`AlreadyExists` para o cliente), voltando o stack (`UPDATE_ROLLBACK_COMPLETE`). O dono o apagou à mão (`delete-managed-login-branding`) e o deploy passou. Quem repetir os testes à mão num ambiente novo deve apagar o estilo no fim.

## O Managed login também foi testado no `dev` (2026-11-19)

Testado à mão, sem mudar código: estilo padrão do Cognito para o cliente (`create-managed-login-branding --use-cognito-provided-values`) e o domínio passado para a versão 2 (`update-user-pool-domain --managed-login-version 2`). Segundo a documentação da AWS, essa troca é uma **atualização no lugar** (não recria o domínio) e leva até 1 minuto; no `template.yaml` ainda se confere no changeset. Depois dos testes o `dev` é devolvido ao original (domínio na versão 1, escopo removido, MFA do usuário desligado, pool em `OFF`).

**Confirmado no navegador e no app Android do `dev`**
- Usuário **sem MFA** entra só com e-mail e senha; usuário **com MFA** recebe o pedido do código depois da senha e, com o código certo, entra.
- Voltar **antes da senha** e **depois da senha** leva de volta à tela de senha; no app, a sessão não se perde indevidamente.
- O cadastro e a ligação do TOTP com o token do login do Managed login (`AssociateSoftwareToken`, `VerifySoftwareToken`, `SetUserMFAPreference`) funcionam com o escopo `aws.cognito.signin.user.admin` e o retorno em `http://localhost:8080/`.
- Alterar a senha funciona com e sem MFA, e o "esqueci a senha" existe (o e-mail de recuperação **não** foi enviado no teste).
- No navegador, voltar depois de logado mostra a tela de "continuar com este usuário ou trocar" (a sessão por cookie do Cognito, já registrada acima); no app a sessão não se perde.
- Com o Managed login, a tela de login demorou a responder em um dos testes (mais de 4 minutos até o retorno ao ouvinte local); não ficou claro se foi o Cognito ou o tempo da pessoa. Vale medir de novo na implementação.

**Conclusão:** nos testes, o Managed login faz o mesmo que o UI clássico para o MFA. **O dono decidiu adotá-lo nesta entrega (2026-11-19).** O visual novo do login é mudança de tela e precisa de demo aprovada antes do código (guardrails).

## Pronto quando

- No `dev`, uma pessoa liga o MFA em Listas, recebe a lista de códigos com a frase em destaque, confirma que leu, sai, entra de novo e o Cognito pede o código do aplicativo; sem o código não entra.
- Sem acesso ao autenticador, ela digita **um código da lista** na tela de recuperação, o MFA desliga, ela entra com e-mail e senha e liga de novo com uma lista nova; o mesmo código não funciona uma segunda vez.
- Teste de unidade prova: código usado não vale de novo, código errado e e-mail sem conta respondem igual, o limite de tentativas bloqueia, o código de uma pessoa nunca desliga o MFA de outra, e a lista só é mostrada uma vez.
- Desligar o MFA em Listas (já logada, com o segundo login) funciona e o login volta a ser só e-mail e senha.
- Funciona no navegador, no PWA e no app Android.
- O changeset do deploy mostra só mudança de propriedade e a permissão mínima da Lambda, sem substituir pool nem cliente.
- `security.md` (autenticação e pontos em aberto), `BACKLOG.md` (item 49, Fase 1: o "reset do dono" vira a lista de códigos) e o `plan.md` ficam em dia; as telas seguem demo aprovada.
