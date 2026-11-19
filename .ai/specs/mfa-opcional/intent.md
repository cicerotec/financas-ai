# Intent: MFA opcional por aplicativo autenticador, com lista de códigos de recuperação

Este documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md) e as regras verificáveis em [`spec.md`](spec.md).

**Estado:** rascunho para aprovação (2026-11-19). Nada foi implementado. É a Fase 1 do item 49 do BACKLOG ([`app-multiusuario`](../app-multiusuario/intent.md)).

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
3. **Login na tela do Cognito (UI hospedada clássica, como hoje).** Depois que a pessoa liga o MFA, o Cognito pede o código de 6 dígitos no login. A senha continua nunca passando
   pelo nosso JavaScript, o que mantém a garantia do app Android (a senha só é digitada na aba do sistema).
4. **Uma tela nossa em Listas, Segurança da conta**, para ligar e desligar o MFA: mostra o QR code e a chave, confere o primeiro código e liga; ou desliga. Ela chama as APIs do Cognito
   (`AssociateSoftwareToken`, `VerifySoftwareToken`, `SetUserMFAPreference`) com o access token da própria pessoa. O Cognito guarda o segredo e confere o código; nós só desenhamos a tela.
5. **Lista de códigos de recuperação, a saída para desligar o MFA.** Ao ligar o MFA, a pessoa recebe uma **lista de códigos de uso único**, mostrada **uma única vez**. Se perder o acesso
   ao autenticador, ela digita **um código da lista** numa tela nossa, sem login, e o MFA dela **desliga**; depois entra só com e-mail e senha e pode ligar de novo, com uma lista nova.
   **O código sozinho não dá entrada**: só desliga o MFA, e a entrada continua exigindo a senha pelo login do Cognito.
6. **A frase que a pessoa vê, em destaque, junto da lista, e confirma que leu antes de o MFA ficar ligado:** **"SE PERDER A LISTA, PERDE O ACESSO."** Mais: a lista precisa ser guardada num
   lugar seguro, **separado do autenticador** (um gerenciador de senhas diferente, papel em local protegido, cofre). Perder o acesso ao autenticador **e** a lista significa perder a conta, **sem volta**,
   mesmo sabendo a senha: nem o dono nem o app conseguem devolvê-la.
7. O backend **não muda a validação do token**: continua validando só o ID token, como em `auth/token.go`. A geração da lista tem uma rota autenticada nova, e o uso de um código tem uma rota
   pública nova (a pessoa que perdeu o autenticador não tem sessão); as duas ganham ação própria em `auth/authz.go`.

## Decisões já tomadas (dono, 2026-11-18 e 2026-11-19)

- MFA **opcional para todos**, não obrigatório.
- **TOTP**, sem SMS (o SMS é cobrado à parte pelo SNS e tem troca de chip e interceptação como pontos fracos).
- **A recuperação é da própria pessoa, sem o dono**, e é a **lista de códigos**. Não é por código no e-mail (vale tanto quanto a conta de e-mail, e quem a invade e tem a senha entra), não é por
  perguntas de segurança (respostas adivinháveis; quase ninguém preenche) e **não é por prazo**.
- **Tela do Cognito, não tela de login própria**: uma tela própria faria a senha passar pelo nosso código e exigiria habilitar fluxos de senha no `UserPoolClient`.
- **Fica no UI clássico agora.** O pool `financas` já está no plano **Essentials** (visto no console em 2026-11-18); o `financas-dev` foi dado como igual pelo dono, e o domínio
  dele está em "UI hospedada (clássica)". O Managed login, com editor visual, é possível sem mudar de plano, mas fica como decisão posterior, com demo própria.
- O login normal **não muda**; o escopo `aws.cognito.signin.user.admin` é pedido só num segundo login, ao abrir a seção do MFA (decisão confirmada em 2026-11-19, ver `plan.md`).

## Dado novo no DynamoDB (o dono discute antes de qualquer código)

A lista de códigos precisa ser guardada, **só como hash** (nunca o código). Isso exige **item ou campo novo**, e o guardrails manda não criar, renomear nem remover chave ou campo sem discutir com o dono.
**Nada será criado até essa conversa.** O que ela precisa definir: onde o hash mora, o tipo de hash (lento, por ser código curto), o formato do código e a quantidade, e o que fica gravado de cada uso
(se foi gasto e quando, sem gravar o código) e do controle de tentativas.

## Segurança da rota de recuperação (requisitos)

A pessoa que perdeu o autenticador **não tem token**, então a rota é pública e é o ponto mais exposto desta entrega:
- **Limite de tentativas por pessoa e por origem**, com teste; força bruta não pode compensar (códigos longos e aleatórios, hash lento no servidor).
- **Resposta idêntica** para e-mail sem conta, sem MFA ou com código errado, para a rota não revelar quem tem conta.
- **Cada código vale uma vez**, com teste que prove isso; gerar uma lista nova invalida a anterior; usar um código desliga o MFA e invalida o restante da lista.
- **Nunca logar** código, e-mail nem corpo (regra de `security.md`).
- A rota desliga o MFA **só da pessoa dona do código**, e a permissão de IAM da Lambda para isso é **mínima** (uma ação só, sobre o próprio pool). Permissão nova na infra exige atualizar as roles do Actions antes do merge.
- Um código vazado só desliga o MFA; a senha continua sendo exigida no login. Quem tem a lista e não a senha **não entra**.

## Limites (o que fica de fora)

- **Perder o autenticador e a lista ao mesmo tempo:** a conta se perde, sem volta (item 6). Sem o dono e sem suporte, não há outro caminho; por isso a frase em destaque é parte da entrega.
- Visual do login (Managed login, branding): outra entrega.
- Login social, SMS, MFA por e-mail do Cognito, perguntas de segurança, recuperação por e-mail e MFA obrigatório para todos.
- Quem **não liga** o MFA continua com só senha: opcional significa que uma senha vazada ainda basta para ele.
- **A sessão do Hosted UI no navegador do aparelho** (cookie, cerca de 1 hora, não confirmado no pool) pode devolver o login sem senha e sem código; sair do app não a encerra. O MFA protege o login novo, não o
  reaproveitamento dessa sessão. É a mesma limitação que o `security.md` já registra para "sair do app".

## Custo

Zero no Cognito. O TOTP não tem cobrança por uso em nenhum plano; o Cognito cobra por usuário ativo no mês (MAU), com 10.000 grátis no Essentials. As rotas novas rodam na Lambda que já existe, e o item
novo no DynamoDB ocupa a tabela de 5/5 RCU/WCU: poucas leituras e gravações por pessoa, só ao gerar a lista e ao usar um código.

## O que foi testado no `dev` (2026-11-19)

Testado à mão no `financas-dev` (pool `sa-east-1_GYBppGGE1`, plano **Essentials**, UI clássico), com dados fictícios e um usuário de teste. Nada disto está no `template.yaml` ainda, e **o `dev` foi
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
- O Hosted UI guarda uma **sessão própria por cookie no navegador do aparelho**. Com ela ativa, o Cognito mostra "continuar com este usuário ou trocar" e pode devolver o login **sem pedir senha nem código**.
- Um teste com o login do navegador pareceu "não pedir o código" porque foi feito com o usuário **sem** MFA. Em teste de MFA, confirme sempre qual usuário foi ligado (console do Cognito, Usuários, nome de usuário = `sub`).

**Ainda não provado**
- Que o segundo login **força a senha de novo** no Hosted UI (`prompt=login` ou `max_age`): é a premissa P2 do plan.
- O login do **app Android** com o escopo novo no pedido. A diferença é só uma palavra a mais no endereço, sem efeito esperado; **não se faz um teste à parte**, acontece na verificação da implementação.
- Que a mudança no `template.yaml` (`MfaConfiguration` e escopo do cliente) **não substitui o pool nem o cliente**: só o changeset do deploy mostra.
- Que a Lambda comum consegue desligar o MFA de uma pessoa (`AdminSetUserMFAPreference`) com permissão mínima.
- Como o "esqueci a senha" do Cognito convive com o MFA ligado; como gerar o QR code no front sem biblioteca pesada; o PWA instalado.

**Estado do `dev` depois dos testes (devolvido ao original em 2026-11-19, conferido por leitura)**
- Pool: `MfaConfiguration: OFF`.
- Cliente: `AllowedOAuthScopes` só `email` e `openid`; `ExplicitAuthFlows` só `ALLOW_REFRESH_TOKEN_AUTH`. O template **não define** `ALLOW_USER_PASSWORD_AUTH` e **não deve passar a defini-lo**.
- Usuário de teste: sem MFA preferido e sem MFA ligado. O segredo do TOTP dele continua registrado e **desativado** no Cognito (sem efeito com o pool em `OFF`); não foi removido.
- O pool de produção (`financas`) nunca foi tocado.
- O primeiro deploy da Fase 2 deve mostrar `MfaConfiguration` e o escopo do cliente como mudança de propriedade.

## O Managed login também foi testado no `dev` (2026-11-19)

Testado à mão, sem mudar código: estilo padrão do Cognito para o cliente (`create-managed-login-branding --use-cognito-provided-values`) e o domínio passado para a versão 2 (`update-user-pool-domain --managed-login-version 2`). Segundo a documentação da AWS, essa troca é uma **atualização no lugar** (não recria o domínio) e leva até 1 minuto; no `template.yaml` ainda se confere no changeset. Depois dos testes o `dev` é devolvido ao original (domínio na versão 1, escopo removido, MFA do usuário desligado, pool em `OFF`).

**Confirmado no navegador e no app Android do `dev`**
- Usuário **sem MFA** entra só com e-mail e senha; usuário **com MFA** recebe o pedido do código depois da senha e, com o código certo, entra.
- Voltar **antes da senha** e **depois da senha** leva de volta à tela de senha; no app, a sessão não se perde indevidamente.
- O cadastro e a ligação do TOTP com o token do login do Managed login (`AssociateSoftwareToken`, `VerifySoftwareToken`, `SetUserMFAPreference`) funcionam com o escopo `aws.cognito.signin.user.admin` e o retorno em `http://localhost:8080/`.
- Alterar a senha funciona com e sem MFA, e o "esqueci a senha" existe (o e-mail de recuperação **não** foi enviado no teste).
- No navegador, voltar depois de logado mostra a tela de "continuar com este usuário ou trocar" (a sessão por cookie do Cognito, já registrada acima); no app a sessão não se perde.
- Com o Managed login, a tela de login demorou a responder em um dos testes (mais de 4 minutos até o retorno ao ouvinte local); não ficou claro se foi o Cognito ou o tempo da pessoa. Vale medir de novo na implementação.

**Conclusão:** nos testes, o Managed login faz o mesmo que o UI clássico para o MFA, então ele **pode** ser a tela de login quando o dono decidir mudar o visual; a escolha continua sendo outra entrega, com demo própria (guardrails), e esta entrega não depende dela.

## Pronto quando

- No `dev`, uma pessoa liga o MFA em Listas, recebe a lista de códigos com a frase em destaque, confirma que leu, sai, entra de novo e o Cognito pede o código do aplicativo; sem o código não entra.
- Sem acesso ao autenticador, ela digita **um código da lista** na tela de recuperação, o MFA desliga, ela entra com e-mail e senha e liga de novo com uma lista nova; o mesmo código não funciona uma segunda vez.
- Teste de unidade prova: código usado não vale de novo, código errado e e-mail sem conta respondem igual, o limite de tentativas bloqueia, o código de uma pessoa nunca desliga o MFA de outra, e a lista só é mostrada uma vez.
- Desligar o MFA em Listas (já logada, com o segundo login) funciona e o login volta a ser só e-mail e senha.
- Funciona no navegador, no PWA e no app Android.
- O changeset do deploy mostra só mudança de propriedade e a permissão mínima da Lambda, sem substituir pool nem cliente.
- `security.md` (autenticação e pontos em aberto), `BACKLOG.md` (item 49, Fase 1: o "reset do dono" vira a lista de códigos) e o `plan.md` ficam em dia; as telas seguem demo aprovada.
