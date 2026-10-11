# Segurança e permissões

Mapa do que protege o app, de onde vem cada proteção no código e o que ainda está em aberto. Tudo abaixo foi conferido no
código e no `template.yaml` em 2026-10-10; onde é proposta, está dito. As regras de conduta da IA (segredos, conflito
com decisão registrada) ficam em [`guardrails.md`](guardrails.md).

## O que se protege
Dados financeiros de uma família (lançamentos, saldos, cartões, notas) e o acesso à conta AWS que os guarda. O repositório
é **público**: nenhum dado real, segredo ou configuração de conta pode ir para o git.

## Quem entra: autenticação
- **Cognito, só por convite.** `AllowAdminCreateUserOnly: true`: não há cadastro aberto; o dono convida
  (`scripts/seed.sh`). Login com e-mail, senha de no mínimo 10 caracteres com minúscula, maiúscula e número.
- **Fluxo `code` com PKCE** (Hosted UI), sem segredo no cliente. Escopos `openid` e `email`.
- **Tokens:** ID token e access token de 60 minutos; refresh token de 30 dias.
- **Validação no backend** (`backend/auth.go`), em toda requisição: assinatura RS256 com as chaves públicas do pool
  (JWKS), `iss`, `aud` (o `CLIENT_ID`), expiração obrigatória e `token_use == "id"`. Sem `sub`, recusa. Qualquer falha é
  `401`.
- A Function URL é `AuthType: NONE` **de propósito**: a barreira é o JWT validado dentro da Lambda. Por isso a
  validação do token é código crítico (veja "Regras ao mexer").
- **Não há MFA no login do app** (o template não configura `MfaConfiguration`); é o item 37 do backlog, de prioridade
  alta. O MFA que existe hoje é o da **sessão AWS do dono** para operar a conta (`scripts/aws-mfa.ps1`).

## O que cada pessoa pode: autorização
**Regra de ouro (`backend/authz.go`): o que não está liberado explicitamente é negado.** Papel ou ação desconhecidos
viram `403`. A decisão é função pura, testada em `authz_test.go`.

O papel vem do item `USER#<sub> / SPACE#<sid>` (campo `role`); sem esse vínculo, `403`. Cada rota é traduzida em uma
**ação** (`rotaParaAcao`); rota sem ação não existe (`404`).

| Ação | owner | member |
|---|---|---|
| `tx.ler` (lançamentos e notas) | sim | sim |
| `tx.criar` | sim | sim |
| `tx.editar` (inclui ocultar e ajustar fatura) | todos | **só os criados por ela** (`criadoPor == sub`) |
| `tx.excluir` | sim | **não** |
| `tx.importar` (lote) | sim | **não** |
| `cfg.ler` (listas, tags, combos, saldos, cartões) | sim | sim |
| `cfg.escrever` (renomear/excluir tag, combos, bancos, status, flags, saldo inicial, início do controle, conferir fatura) | sim | **não** |
| `tags.adicionar` | sim | sim, **só tags que ainda não existem** (o servidor só acrescenta) |
| `fechamento.ler` | sim | sim |
| `fechamento.escrever`, `.conferir`, `.reabrir` | sim | **não** (só vê) |
| `aparencia.ler` / `.escrever` | sim | sim, **só a dela** |
| `perfil.editar` (o próprio nome e chat id do Telegram) | sim | sim |
| `seq.ler` | sim | sim |
| `avisos.testar` (mensagem de teste para o próprio Telegram) | sim | sim |

Dois detalhes que contam:
- **Registro sem `criadoPor` não é de ninguém:** só o owner mexe nele (`ehDono`).
- **Campos que o cliente não define** (`protegidos`): `PK`, `SK`, `id`, `criadoEm` e `criadoPor` são descartados do corpo e
  preenchidos pelo servidor; ninguém grava um lançamento "em nome de" outra pessoa nem escolhe a própria chave.
- **O `sub` do perfil vem sempre do token**, nunca do corpo: `putPerfil` grava no item de quem fez a chamada.

**O front não é fronteira de segurança.** Esconder um botão para o member é conforto; a recusa é do backend. Regra que
só existe no front não protege nada (veja [`arquitetura.md`](arquitetura.md)).

## Dados e privacidade
- O member **lê todos** os lançamentos e notas e vê todos os bancos e cartões. A visibilidade por banco e cartão é o
  item 27 do backlog (decidido: aplicar no backend, não só esconder no front; banco novo nasce invisível para quem não
  é dono).
- **Dado real nunca no git.** O `.gitignore` ignora `*.json` (exceção só para o Postman e para
  `.ai/specs/**/*.json`, que são fictícios), `web/config.js`, `web/config.dev.js` e `samconfig.toml`. Exportações reais
  ficam em `/local/`, também ignorado.
- O ambiente `dev` usa **dados fictícios** (`scripts/dados-ficticios.py`), em stack, tabela e login próprios.
- **Telegram é um terceiro:** o resumo diário leva a **descrição, o valor e o banco** dos lançamentos futuros para o chat
  de cada pessoa. É uma escolha de produto (os avisos), mas significa que esses dados saem da AWS.
- **Logs** (CloudWatch, retenção de 14 dias): pelo que o código faz hoje, registram `sub`, papel e ação nas negações,
  contagens da rotina de avisos e mensagens de erro. Nenhum `log` imprime token, e-mail ou corpo da requisição; a regra é
  manter assim (veja "Regras ao mexer").

## Segredos
- O **token do bot do Telegram** fica no SSM Parameter Store como `SecureString`, em `/financas/<ambiente>/telegram-token`.
  A Lambda só lê esse parâmetro (`SSMParameterReadPolicy` do nome exato) e só decifra pela chave via SSM (`kms:ViaService`).
- **Nenhuma chave de acesso AWS no CI:** o GitHub Actions troca o token OIDC por credenciais temporárias.
- Segredo **nunca** em prompt, chat, commit, linha de comando ou log. Se aparecer, revogar (para o bot: `/revoke` no
  BotFather e gravar o novo no SSM).
- Os dois ambientes usam o **mesmo bot**; as mensagens do dev começam com `[DEV]`.

## Infraestrutura
- **Front:** S3 privado (bloqueio total de acesso público) servido só pelo CloudFront via OAC; a política do bucket aceita
  apenas a distribuição. HTTPS obrigatório (`redirect-to-https`) e a política de cabeçalhos de segurança gerenciada da
  AWS. O template **não define CSP própria**.
- **API:** Function URL com CORS restrito às origens do front (CloudFront e `localhost`). CORS **não é controle de
  acesso**: quem chama sem navegador ignora; quem protege é o JWT.
- **Lambda:** CRUD na própria tabela (inclui `DeleteItem`), leitura do parâmetro do Telegram e nada mais.
- **Tabela:** `DeletionPolicy: Retain` e `UpdateReplacePolicy: Retain` (o deploy não apaga a tabela). **PITR ligado só em
  prod** (`!If [EhDev, false, true]` no `template.yaml`; retenção de 35 dias; o `dev` fica desligado). Além dele, só o
  botão Exportar. O backup periódico do item 11 segue adiado.
- **Rate limit:** não há WAF nem teto por IP; o teto de concorrência está desligado porque a conta permite 10 execuções
  no total, o que já funciona como limite natural.

## CI/CD
Detalhes em [`deploy-github-actions.md`](deploy-github-actions.md). Em resumo: cada role OIDC só aceita o repositório
(com ID numérico imutável) e o `environment` do ambiente; as policies são limitadas aos nomes de cada ambiente e negam
`dynamodb:DeleteTable`; o deploy falha se o changeset remover ou substituir a tabela ou o login; produção exige aprovação
manual.

## Pontos em aberto
| Ponto | Risco | Onde acompanhar |
|---|---|---|
| Token do Cognito em `localStorage` | Um XSS no front rouba a sessão | item 18 do backlog |
| Sem CSP própria | Reduz a defesa em profundidade contra XSS | item 18 |
| Sem WAF nem limite de taxa na Function URL | Enxurrada de requisições (custo e disponibilidade) | item 18 |
| Member lê tudo | Dados de bancos e cartões que não são dela | item 27 |
| Sem MFA no login do app | Senha vazada basta para entrar | item 37 (prioridade alta) |
| Sem histórico de versões (PITR só em prod, janela de 35 dias) | Uma edição errada perde o dado depois da janela (aconteceu em 2026-10-10) | item 31 |
| Role de dev alcança pool e CloudFront de prod pelas ações listadas | Proteção é o `environment` dev só aceitar a `develop` | `infra/github-oidc.yaml` |
| Campos opcionais do lançamento gravados como vêm (ex. `faturaAjuste` sem validação) | Valor inesperado persiste | item 33 (regras no backend) |
| Resumo do Telegram acima de 4096 caracteres | Nenhum aviso chega naquele dia (disponibilidade, não vazamento) | item 28, item 34 |

## Regras ao mexer (propostas, a confirmar com o dono)
- **Rota nova exige ação nova** em `authz.go`, com o que owner e member podem, **e teste** em `authz_test.go`. Nunca
  liberar por padrão.
- **Não afrouxar a validação do token** (`algoritmo`, `iss`, `aud`, `exp`, `token_use`) nem passar a aceitar o access token
  sem decisão do dono.
- **Toda escrita confere o papel no servidor**, mesmo que o front já esconda o botão.
- **Nunca logar** token, e-mail, corpo da requisição, descrição ou valor de lançamento.
- **Dado novo que o member não deve ver** nasce filtrado na origem (backend), não escondido na tela.
- **Permissão nova na infra** exige atualizar as roles do Actions antes do merge (veja o guardrails).
- Mudou uma regra de permissão: atualize **esta tabela e o BACKLOG** na mesma entrega.
