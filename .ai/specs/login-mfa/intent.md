# Intent: tela de login própria do app e verificação em duas etapas (opcional)

Item 37 do BACKLOG (MFA no login), ampliado pelo dono em 2026-10-17 para incluir a **personalização da tela de login**. Este
documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md). O `spec.md` de cada fase é escrito quando a fase começa,
depois do ok do dono, como nos itens 31, 32 e 38.

**Estado:** rascunho para o dono aprovar. Nenhum código foi alterado.

## Por quê

Hoje a pessoa entra só com e-mail e senha, numa página que não é do app: é a tela hospedada do Cognito
(`<prefixo>.auth.<região>.amazoncognito.com`), com o visual padrão da AWS. Duas coisas incomodam:

1. **A tela de login não parece o app.** O dono quer poder personalizá-la, mesmo que minimamente (logo, cores, textos).
2. **Uma senha vazada basta para ver tudo** (um membro lê todos os lançamentos do espaço). O dono quer um segundo fator, mas
   **nunca obrigatório**: cada pessoa decide se liga.

## O que queremos

1. **A tela de login pode ser customizada**, no mínimo com a identidade do app (logo e cores do tema, claro e escuro). Quanto mais
   além disso (textos, fundo) depende do que o Cognito permite sem o app assumir o tratamento de senha.
2. **Verificação em duas etapas (MFA) opcional e por pessoa.** Quem não ligar entra como hoje, sem nenhuma mudança. Quem ligar passa a
   digitar, no login, o código de 6 dígitos do aplicativo autenticador (TOTP, sem SMS e sem custo).
3. **Cada pessoa liga e desliga o MFA sozinha**, numa tela do app (Perfil), com QR code para cadastrar o aplicativo. Não há script
   nem pedido ao dono para isso.
4. **Perdeu o celular? Sem script.** O **dono desliga o MFA de outra pessoa por um botão no app**, e a pessoa volta a entrar só com a
   senha. A recuperação pelo próprio Telegram da pessoa (sem depender do dono) é um item seguinte, com spec e revisão de segurança
   próprios, porque abre a primeira rota sem login.
5. **O backend continua validando o ID token como hoje**; MFA é problema do Cognito, não das rotas da API.
6. **Os dois ambientes** (`dev` e `prod`) recebem a mudança pelo `template.yaml`, o `dev` primeiro.

## Limites (o que fica de fora)

- **MFA obrigatório**: não é objetivo, nem para o dono.
- **SMS e WhatsApp como fator**: SMS tem custo; WhatsApp exige a API de empresas. O fator de **e-mail** do Cognito exige um plano pago
  mais alto e o SES, e só entra se o dono quiser depois de ver o custo.
- **Tela de login escrita pelo app** (formulário próprio falando com a API do Cognito): fora do escopo. Traria para o código do app o
  tratamento de senha, primeira senha, "esqueci a senha" e dos desafios de MFA, e aumentaria a superfície de segurança. Fica como
  evolução futura, se a personalização do Cognito não bastar.
- **Recuperação pelo Telegram** (rota sem login): item seguinte, não faz parte das fases abaixo.
- **Passkeys / login social**: fora.

## Riscos

| Risco | Resposta |
|---|---|
| Com MFA opcional a tela hospedada **não oferece sozinha o cadastro** do aplicativo (só pede o código de quem já cadastrou) | A fase B entrega a tela de Perfil que faz o cadastro pela API do Cognito |
| Personalização da tela hospedada limitada ao que o Cognito deixa (a versão clássica e a nova "managed login" diferem) | A fase 0 confirma no `dev` qual versão o pool usa e o que dá para mudar, antes da demo |
| Alguém ligar o MFA e perder o celular | Botão do dono (fase C); depois a recuperação pelo Telegram |
| O dono perder o celular do próprio MFA | O mesmo botão em outro dono, ou o script/console como último recurso (documentado em `security.md`) |
| A tela de Perfil pedir o *access token* e um escopo a mais | Fase B decide como o app o guarda (hoje só o ID token e o refresh) e atualiza `security.md` |
