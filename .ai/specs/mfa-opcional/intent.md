# Intent: MFA opcional por aplicativo autenticador

Este documento diz **o quê e por quê**; o como virá em `plan.md` e `spec.md`, depois da aprovação deste.

**Estado:** rascunho para aprovação (2026-11-18). Nada foi implementado. É a Fase 1 do item 49 do BACKLOG ([`app-multiusuario`](../app-multiusuario/intent.md)).

## O problema

O login é só e-mail e senha (o `UserPool` do `template.yaml` não define `MfaConfiguration`). Uma senha vazada basta para ver todos os dados financeiros da família.
Está registrado em [`security.md`](../../security.md) como ponto em aberto de prioridade alta.
O único MFA que existe hoje é o da sessão AWS do dono (`scripts/aws-mfa.ps1`), que é outra coisa.

## O que queremos

1. **MFA por aplicativo autenticador (TOTP), sem SMS**, nos dois ambientes (`dev` e prod), configurado no `template.yaml`.
2. **Opcional para todos**, inclusive o dono: cada pessoa liga e desliga o seu. Não é obrigatório.
3. **Login na tela do Cognito (UI hospedada clássica, como hoje).** Depois que a pessoa liga o MFA, o Cognito pede o código de 6 dígitos no login. A senha continua nunca passando
   pelo nosso JavaScript, o que mantém a garantia do app Android (a senha só é digitada na aba do sistema).
4. **Uma tela nossa em Listas, Segurança**, para ligar e desligar o MFA: mostra o QR code e a chave, confere o primeiro código e liga; ou desliga. Ela chama as APIs do Cognito
   (`AssociateSoftwareToken`, `VerifySoftwareToken`, `SetUserMFAPreference`) com o access token da própria pessoa. O Cognito guarda o segredo e confere o código; nós só desenhamos a tela.
5. **Aviso claro antes de ligar.** A tela diz, em texto fixo, que o MFA é uma camada a mais de proteção e que **perder o acesso ao autenticador pode significar perder o acesso à conta**, sem
   caminho de volta pelo app. O autenticador é onde o segredo do TOTP mora (o app no celular, o gerenciador de senhas ou o programa no computador); perder o acesso a ele é perder o celular, trocar
   de computador sem levar o segredo, apagar o app ou formatar a máquina. A pessoa confirma que leu **antes de o MFA ficar ligado**. Por ser opcional, ela decide sabendo.
6. O backend **não muda**: continua validando só o ID token, como em `auth/token.go`.

## Decisões já tomadas (dono, 2026-11-18)

- MFA **opcional para todos**, não obrigatório.
- **TOTP**, sem SMS (o SMS é cobrado à parte pelo SNS e não será usado).
- **Tela do Cognito, não tela de login própria**: uma tela própria faria a senha passar pelo nosso código e exigiria habilitar fluxos de senha no `UserPoolClient`.
- **Fica no UI clássico agora.** O pool `financas` já está no plano **Essentials** (visto no console em 2026-11-18); o `financas-dev` foi dado como igual pelo dono, e o domínio dele está
  em "UI hospedada (clássica)". O Managed login, com editor visual, é possível sem mudar de plano, mas fica como decisão posterior, com demo própria.
- **Sem recuperação nesta entrega** (dono, 2026-11-18: "nenhum por enquanto"). Nem chaves de recuperação, nem e-mail, nem prazo. O dono também não quer ser o caminho de volta.

## Limites (o que fica de fora)

- **Recuperação de quem perde o acesso ao autenticador: fica de fora.** Consequência dita sem suavizar: essa pessoa, com o MFA ligado, **não entra mais**, mesmo sabendo a senha (o MFA é pedido depois dela;
  redefinir a senha pelo e-mail não ajuda). Por isso o aviso do item 5 é parte da entrega. Uma fase seguinte pode tratar disso (chaves de recuperação, e-mail, prazo); as opções
  foram discutidas em 2026-11-18 e nenhuma foi escolhida. Qualquer uma que guarde dado novo no DynamoDB exige discutir com o dono antes (guardrails).
- Visual do login (Managed login, branding): outra entrega.
- Login social, SMS, MFA por e-mail do Cognito e obrigatoriedade do MFA para todos.
- Quem **não liga** o MFA continua com só senha: opcional significa que uma senha vazada ainda basta para ele.

## Custo

Zero. O TOTP não tem cobrança por uso em nenhum plano; o Cognito cobra por usuário ativo no mês (MAU), com 10.000 grátis no Essentials. As chamadas das telas ao Cognito não passam pela Lambda.
Não há dado novo no DynamoDB.

## O que foi testado no `dev` (2026-11-19) e o que falta

Testado à mão no `financas-dev` (pool `sa-east-1_GYBppGGE1`, plano **Essentials**, UI clássico), com dados fictícios e um usuário de teste. Nada disto está no `template.yaml` ainda.

**Confirmado**
- O pool aceita `MfaConfiguration: OPTIONAL` com o TOTP habilitado (`set-user-pool-mfa-config`).
- Com um token de login direto, `AssociateSoftwareToken`, `VerifySoftwareToken` (com o autenticador do celular) e `SetUserMFAPreference` funcionam e ligam o MFA do usuário.
- Com o MFA ligado, o Cognito devolve o desafio `SOFTWARE_TOKEN_MFA` pela API, e **o login do Hosted UI clássico pede o código de 6 dígitos** depois da senha, no navegador. Um usuário sem MFA continua entrando só com e-mail e senha.
- **No app Android do `dev`** (aba do sistema): o login pede senha e código e volta ao app logado. Voltar na aba **antes da senha** deixa o app sem sessão; voltar **depois da senha e antes do código** também deixa o app sem sessão, e um novo "Entrar" recomeça. Nenhum dos dois casos abriu o app logado sem o código.
- O pool do `dev` não tem dispositivo lembrado (`DeviceConfiguration` nulo) nem `UserPoolAddOns`; o único primeiro fator é a senha. A recuperação de senha é por e-mail e depois telefone; sem telefone, na prática só por e-mail.

**Descoberto no teste (vale para o plano)**
- O Hosted UI guarda uma **sessão própria por cookie no navegador do aparelho**. Com ela ativa, o Cognito mostra "continuar com este usuário ou trocar" e pode devolver o login **sem pedir senha nem código**. Pelo que sei, ela dura cerca de 1 hora (não confirmado no pool). Sair do app **não encerra essa sessão**: o MFA protege o login novo, não o reaproveitamento da sessão do navegador. É a mesma limitação que `security.md` já registra para "sair do app", e deve ser registrada ali também com o MFA.
- Um teste com o login do navegador pareceu "não pedir o código" porque foi feito com o usuário **sem** MFA. Em teste de MFA, confirme sempre qual usuário foi ligado (console do Cognito, Usuários, nome de usuário = `sub`).

**Ainda não provado**
- Se o **access token do login do Cognito** (e não o do login direto) aceita as APIs de MFA com o escopo `aws.cognito.signin.user.admin`. O escopo já foi marcado no cliente do `dev` pelo console, mas o app hoje só pede `openid email`; falta o front pedir o escopo novo. É o ponto mais importante antes da tela de Listas.
- Que **a própria pessoa desliga o MFA** (`SetUserMFAPreference` com o token dela).
- Que a mudança no `template.yaml` (`MfaConfiguration` e escopo do cliente) **não substitui o pool nem o cliente**: só o changeset do deploy mostra.
- Como a recuperação de senha do Cognito ("esqueci a senha", por e-mail) convive com o MFA ligado.
- Como gerar o QR code no front sem biblioteca pesada (dependência nova ou código nosso).
- No PWA e no navegador de computador do dev o desafio foi visto só no navegador; o PWA instalado não foi testado.

**Limpeza do `dev` ainda pendente**
- O cliente do `dev` está com `ALLOW_USER_PASSWORD_AUTH` habilitado **só para os testes**. Precisa ser desmarcado no console; o `template.yaml` não o define (só `ALLOW_REFRESH_TOKEN_AUTH`) e **não deve passar a defini-lo**.
- O pool do `dev` está com `MfaConfiguration: OPTIONAL` e o primeiro usuário de teste com o TOTP ligado, mudanças feitas à mão e fora do template.

## Pronto quando

- No `dev`, uma pessoa liga o MFA em Listas, sai, entra de novo e o Cognito pede o código do aplicativo; sem o código não entra.
- A tela mostra o aviso do item 5 e só liga o MFA depois de a pessoa confirmar que leu.
- Desligar o MFA em Listas (já logada) funciona e o login volta a ser só e-mail e senha.
- Funciona no navegador, no PWA e no app Android.
- O changeset do deploy mostra só mudança de propriedade, sem substituir pool nem cliente.
- `security.md` (autenticação e pontos em aberto), `BACKLOG.md` (item 49, Fase 1: tirar o "reset do dono") e o `plan.md` ficam em dia; a tela segue uma demo aprovada.

## Para o dono decidir

- Se a tela de Listas fica numa seção "Segurança" (já existe "Aparência, Segurança" com a trava e a chave de apps recentes, só no app) ou num cartão próprio; vale também no navegador, ao contrário da trava.
- Quando tratar da recuperação, e qual caminho.
