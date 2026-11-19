---
name: 'MFA opcional: pendências da revisão de arquitetura'
type: review-handoff
status: handoff
created: '2026-11-19'
complements: 'arquitetura-mfa-recuperacao.md (A1 e A5, já absorvidos)'
sources: ['financas-ai/.ai/specs/mfa-opcional/{intent,plan,spec}.md (estado de 2026-11-19, depois da atualização)', 'financas-ai/.ai/security.md', 'financas-ai/backend/roteador.go, auth/authz.go, local.go', 'financas-ai/web/js/sessao.js']
---

# MFA opcional: pendências da revisão de arquitetura

Complementa [`arquitetura-mfa-recuperacao.md`](arquitetura-mfa-recuperacao.md). Aquele documento cobriu A1 e A5 e o agente do projeto os absorveu no `plan.md`, no `spec.md` e no `security.md`. Este cobre **o resto**, por escrito, para o agente poder agir. Os itens se dividem em três grupos:

- **Parte 1:** correções ao que o agente escreveu na última atualização.
- **Parte 2:** ajustes de spec que o agente faz sem discussão.
- **Parte 3:** decisões do dono (o agente não decide; leva ao dono com as opções).

Nada aqui cria dado no DynamoDB. Onde divergir de `guardrails.md`, `arquitetura.md` ou `decisoes.md`, a divergência vai ao dono antes de qualquer código.

## Parte 1: correções à última atualização

**C1. R28 perdeu a exigência de tempo.** O texto original dizia "mesmo status, corpo e tempo aproximado"; hoje diz só "status e corpo constantes". Restaurar. Como o tempo é o canal que sobra, a regra de implementação é: **e-mail sem conta faz o mesmo trabalho que e-mail com conta** (a mesma consulta ao Cognito e uma comparação de código contra um hash fictício do mesmo custo). Escrever também que o teste de unidade **não prova tempo**; o que se testa é que os dois caminhos chamam as mesmas portas na mesma ordem, e a prova de tempo é medida no `dev` (acrescentar à "Verificação no `dev`").

**C2. P13 e `security.md` dizem "o dano é da própria pessoa", e isso é falso no caso do XSS.** Quem liga o MFA à mão com o token de escopo amplo pode ser um script injetado, não a pessoa. Com o MFA ligado por um autenticador do atacante e sem lista, a vítima fica sem saída. Reescrever em P13, no risco da tabela do plan e na linha de pontos em aberto do `security.md`: "dano da própria pessoa **ou de quem executar código na página durante a seção do MFA**". O tratamento é da Parte 3 (D2 e D3).

**C3. Resíduos do desenho antigo:**
- M-T3: "as duas rotas novas" passa a "as três rotas novas".
- R23: "o servidor só é chamado quando a pessoa **gera a lista** (R25)" passa a "gera ou confirma a lista (R25 e R25a)".
- `intent.md`, seção "Dado novo no DynamoDB": "o tipo de hash (lento, por ser código curto)" passa a "o tipo de hash (HMAC com pepper se o código for longo; lento só se for curto)".

**C4. Roteador: as rotas autenticadas novas não cabem no ramo de hoje.** `processar` (`backend/roteador.go:109`) só trata `AcaoSairPush` entre as rotas sem espaço, e lê o corpo como JSON obrigatório (`json invalido` se vazio). O plano da Fase 3 deve dizer que: (a) `RotaSemEspaco` passa a reconhecer `POST /me/mfa/codigos` e `POST /me/mfa/codigos/confirmar` (ações próprias, com teste em `authz_test.go`); (b) o ramo de `processar` vira um `switch` por ação, não um `if` por `AcaoSairPush`; (c) **gerar aceita corpo vazio** (R25 já diz), **confirmar exige JSON** com `listaId`. Teste: gerar sem corpo é `200`, confirmar sem corpo é `400`.

**C5. `guardrails.md` ganhou a regra "Seção API no `spec.md`".** Foi pedida pelo dono (2026-11-19), então fica. Registrar em `decisoes.md` a decisão e o motivo (não há gateway nem documento central de rotas), para a regra não parecer iniciativa de agente.

## Parte 2: ajustes de spec (o agente faz)

**A7. O segundo login colide com o código de sessão atual.** Em `web/js/sessao.js`:
- **Navegador:** `token()` (linhas 49 a 58) trata todo `?code=` como login normal e chama `salvarAuth`, que grava o ID token em `localStorage`. O `redirect_uri` é o mesmo e o `financas.pkce` é um slot só.
- **Android:** `tratarRetorno` (linhas 103 a 121) grava o refresh no `Cofre` e chama `env.recarregar()`, o que apaga o access token que vive em memória.

Regra nova (acrescentar a R10 e R12): o PKCE pendente leva um **tipo** (`normal` ou `mfa`) e o retorno do tipo `mfa` segue um ramo próprio: troca o `code`, entrega **só o access token** à seção, **não** chama `salvarAuth`, **não** grava `refresh`, **não** recarrega a página. Testes: M-T7 estendido (retorno `mfa` no navegador não toca `financas.auth`; no app não grava `refresh` nem recarrega) e um teste de que um retorno `normal` continua como hoje.

**A8. O refresh token do segundo login continua válido por 30 dias no Cognito.** "Descartado" (R10) só vale no cliente. Acrescentar a R10: logo depois de trocar o `code`, o front chama `POST /oauth2/revoke` com o refresh do segundo login (cliente público, sem segredo). **Confirmar no `dev`** que o revoke aceita o cliente e que o access token já emitido continua funcionando até a seção fechar; acrescentar à "Verificação no `dev`".

**A10. Duas premissas não provadas** (acrescentar a "Verificação no `dev`" e ao intent, em "Ainda não provado"):
1. O navegador e o WebView, a partir do CloudFront, chamam `cognito-idp.<região>.amazonaws.com` por `fetch` com `Content-Type: application/x-amz-json-1.1` (CORS). Os testes do intent foram feitos por script.
2. R18 afirma que o Cognito "aplica o bloqueio depois de 5 erros" no `VerifySoftwareToken`. Isso não está na lista de provados. Provar, ou reescrever R18 sem o número ("mensagem de aguardar quando o Cognito recusar por excesso de tentativas").

**A11. Entrada da tela de recuperação no navegador.** `Sessao.token()` do navegador sempre termina em `irParaLogin` (linhas 49 a 66). R33 diz "endereço próprio" mas não diz como a página carrega sem passar por ali. Regra nova em R33 e teste: o endereço da recuperação carrega **antes** de `Sessao.token()` e nunca inicia o login; depois de sucesso, o botão "Entrar" leva ao login. Mesma regra no app Android (a Abertura sem sessão já mostra "Entrar"; a recuperação é um segundo botão, e não dispara o login).

**A15. Ação de leitura do Cognito indefinida.** R32 e M-T30 dizem "mais a leitura mínima". Definir: **`AdminGetUser`** (devolve o `sub` a partir do e-mail, se o pool aceitar o e-mail como `Username`, já que usa `UsernameAttributes: [email]`) em vez de `ListUsers`. **Confirmar no `dev`** antes de fixar. Se `AdminGetUser` não aceitar o e-mail, o fallback é `ListUsers` com filtro por `email`, que precisa de teto de uso porque a cota é compartilhada com outras chamadas administrativas do pool. A ação escolhida entra nominalmente em R32 e M-T30.

**A16. Configuração de conta no repositório público.** O `intent.md` registra o id do pool do `dev`. O `security.md` diz que configuração de conta não vai ao git. Trocar por "pool do `dev`".

**E5. Normalização do e-mail.** O limite por pessoa e a busca partem do e-mail; `A@x.com` e `a@x.com` furam o limite. Acrescentar a R29: normalizar (minúsculas, sem espaços nas pontas) **antes** de contar e de buscar; o `plan.md` já fala em "hash do e-mail normalizado", o spec não. Teste: variações de caixa e espaço contam no mesmo contador. O alias não entra (R30 protege pelo `sub`).

**E6. R17: desligar em Listas não toca a lista no servidor.** Declarar em R17: a lista ativa **fica** (inofensiva, porque a rota pública responde genérico para quem está sem MFA) e é substituída na próxima vez que a pessoa ligar o MFA (R16). Sem rota de limpeza.

**E7. R12: a intenção "abrir a seção do MFA" não expira.** Se o segundo login for cancelado no Cognito (voltar, erro) e a pessoa abrir o app depois, a seção reabriria sozinha. Acrescentar a R12: a intenção tem **TTL curto** (por exemplo, 10 minutos) e é apagada no retorno com erro ou cancelamento. Teste.

## Parte 3: decisões do dono

Cada item vai ao dono com as opções; o agente não decide.

**D1. O texto "nem o dono devolve a conta" é falso (A2).** O dono tem `admin-set-user-mfa-preference` no console ou na CLI, e foi o que se usou nos testes do `dev`. O que existe é a decisão de produto de **o app não oferecer** esse caminho. Opções:
- (a) **Recomendada:** reescrever `intent.md` (item 6 e "Limites") e `security.md` como "o app não oferece caminho; o dono pode, por fora do app, desligar o MFA pelo console da AWS como último recurso" e **manter a frase da tela** ("SE PERDER A LISTA, PERDE O ACESSO.") como regra de produto, sem prometer nada à pessoa. O passo a passo do comando fica **fora do git** (o repositório é público).
- (b) Manter o texto de impossibilidade, o que obriga a retirar a capacidade do dono (não é possível: ele é administrador da conta) e deixa a promessa falsa.
- (c) Documentar o último recurso na própria tela. Não recomendado: vira vetor de engenharia social.

**D2. Tomada do MFA por script injetado (A3).** Um XSS durante a seção tem o access token de escopo amplo e o ID token. Liga o MFA com o autenticador do atacante, gera uma lista que só ele vê, e a vítima fica sem saída. Mitigações, em ordem de custo: (1) registrar o risco (C2); (2) D1 (o último recurso do dono); (3) D3 (aviso); (4) CSP própria (item 18 do backlog). Decidir quais entram nesta entrega.

**D3. Aviso quando o MFA muda (A9).** Quem tem a senha e um código da lista desliga o MFA pela rota pública, e nada avisa a pessoa. O despachante de avisos (Telegram, push) já existe. Opções:
- (a) Avisar pelo canal que a pessoa já tem ligado, quando houver um, sem prometer entrega a quem não tem. Custo: um tipo novo de aviso no despachante.
- (b) Não avisar, e registrar a limitação em `security.md`.
- Vale para o desligar pela recuperação e para o ligar e desligar em Listas.

**D4. Como a pessoa guarda a lista (A12).** O spec não define o gesto. R19 proíbe a área de transferência **automática** só para a chave do TOTP. Opções (a decidir na demo da Fase 1): (a) botão **Copiar tudo**, por gesto explícito; (b) **Imprimir**; (c) **Baixar .txt**, que deixa um arquivo no aparelho e vai a backup; (d) só ler e anotar, que é o que o texto atual dá a entender. Recomendação: (a) e (b). No app Android, ligar `FLAG_SECURE` enquanto a lista estiver na tela, para a captura de tela não a levar ao backup de fotos; a demo diz o que isso faz perder (a pessoa não consegue tirar print).

**D5. A pessoa não vê se tem lista válida (A13).** R23 impede ler o servidor ao abrir a seção. Sem um `GET /me/mfa/codigos` (só `{ativa, restantes, geradaEm}`, chamado quando a janela abre, nunca na subida), nada diz que a lista existe. Opções: (a) incluir o `GET` na Fase 3; (b) não incluir, aceitando que a seção mostra só o estado do MFA (R15). É uma quarta rota; entra na seção "API" se aprovada.

**D6. Entropia e hash (A6).** O plan já descreve as duas saídas (HMAC com pepper para código de 100 bits ou mais; hash lento para código curto) e deixa a escolha para a conversa do DynamoDB. Registro aqui só para a conversa levar o número: a conta permite 10 execuções simultâneas, então hash lento em rota pública é custo de CPU e vetor de negação de serviço.

## Ordem sugerida para o agente

1. Parte 1 inteira e Parte 2 inteira, no `plan.md`, `spec.md`, `intent.md` e `security.md`, sem implementar.
2. Levar a Parte 3 ao dono junto com a conversa do DynamoDB (já pendente). Cada decisão que mudar o desenho entra no spec antes da Fase 3.
3. Pedir de novo a revisão independente do `spec.md` depois disso (o guardrails manda, e entraram regras novas).

## Teste de aceitação desta pendência

Um revisor que leia só o `spec.md` deve achar, por regra e por teste: R28 com o tempo; R10 com o revoke e o ramo `mfa` do retorno; R12 com TTL; R17 com a lista ativa que fica; R29 com a normalização; R32 com a ação de leitura nominal; R33 com a entrada antes de `Sessao.token()`; M-T7 estendido; e a "Verificação no `dev`" com CORS, o bloqueio de 5 erros, o revoke, `AdminGetUser` e o tempo da rota pública.
