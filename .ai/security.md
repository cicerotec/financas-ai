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
- **Onde o token fica** (`web/js/sessao.js`, item 54 fase 2). **Navegador e PWA:** em `localStorage` (`financas.auth`), como sempre; é o ponto em aberto do item 18. **Sem sessão, o navegador e a PWA mostram a tela de entrada e o login só abre no toque em "Entrar"** (R33a do `mfa-opcional`, 2026-11-19; antes iam direto ao login). **O access token do segundo login do MFA fica só em memória** (`web/js/sessao.js`; nunca em `localStorage`, `sessionStorage` nem no `Cofre`), o ID token e o refresh desse login são descartados e o refresh é revogado na troca.
  **App Android:** o login abre na **aba do sistema** (Custom Tabs; a WebView nunca vê a senha) e só no toque em "Entrar"; o **refresh token** fica no plugin
  `Cofre` (`app-android/.../CofrePlugin.java`), cifrado com AES-GCM por uma chave do **Android Keystore** que nunca sai dele, num arquivo fora do backup
  (`allowBackup="false"` e regras de extração); o **ID token fica só em memória**; nada vai a `localStorage` nem a `sessionStorage`. O `verifier` e o `state` do PKCE
  ficam no mesmo `Cofre` durante o login e são apagados ao fim dele. O log do Capacitor fica desligado (`loggingBehavior: "none"`), e nem o módulo nem o plugin escrevem log.
  - **O que o Keystore não protege:** o plugin responde "ler" ao JavaScript da página, então um script injetado na WebView obtém o refresh token como obteria do
    `localStorage`. O ganho é o arquivo cifrado em disco e fora do backup. Fazer o refresh em Java, sem o token chegar ao JavaScript, é endurecimento futuro. Aparelho
    com root também não é coberto.
  - **Retorno do login por esquema próprio** (`<applicationId>://auth`, um por ambiente): outro app instalado pode registrar o mesmo esquema. O `code` sozinho não vale
    (falta o `code_verifier` do PKCE) e o `state` é conferido; sobra atrapalhar o login. O App Link por `https` fecha isso na fase 5.
  - **Sair sem internet:** o app apaga o que guarda, mas o Cognito não é avisado e o navegador do aparelho continua lembrando do login; o próximo "Entrar" pode voltar
    sem pedir a senha.
  - **No app, rede não é logout:** só a recusa do refresh pelo Cognito (`400`) encerra a sessão; falha de rede e `5xx` mantêm. Um `401` da API faz um refresh e uma
    repetição; o segundo `401` vira erro na tela. O mesmo `UserPoolClient` serve o navegador e o app: não dá para encerrar só as sessões do app.
- **Trava por biometria no app Android** (item 54, fase 3a; `web/js/bloqueio.js` e o plugin `Trava`): opcional, ligada por cada pessoa em Listas, guardada só no aparelho. Protege
  contra quem pega o **aparelho desbloqueado**: com ela, o app pede a digital, o rosto ou a senha do aparelho ao abrir e depois de 1 minuto fora. **É só da tela:** não cifra os
  dados nem o refresh token (que continua no `Cofre`), não vale em aparelho com root, e um script injetado na página poderia desligá-la, porque a escolha fica em `localStorage`.
  Não substitui a senha nem o MFA do Cognito. A chave "Esconder nos apps recentes" marca a janela como protegida (`FLAG_SECURE`): o app some da lista de apps abertos e não aceita
  captura nem gravação de tela. Sair da conta desliga as duas.
- **Validação no backend** (`backend/auth/token.go`), em toda requisição: assinatura RS256 com as chaves públicas do pool
  (JWKS), `iss`, `aud` (o `CLIENT_ID`), expiração obrigatória e `token_use == "id"`. Sem `sub`, recusa. Qualquer falha é
  `401`.
- A Function URL é `AuthType: NONE` **de propósito**: a barreira é o JWT validado dentro da Lambda. Por isso a
  validação do token é código crítico (veja "Regras ao mexer").
- **MFA no login do app: opcional, por aplicativo autenticador (TOTP), sem SMS** (`MfaConfiguration: OPTIONAL` e `EnabledMfas: [SOFTWARE_TOKEN_MFA]` no `template.yaml`; fase 1 do item 49 do backlog). **Falta o front (a tela em Listas, Fase 4):** até ela existir, ninguém consegue ligar o MFA pelo app, e quem não liga continua só com a senha. O MFA da **sessão AWS do dono** para operar a conta (`scripts/aws-mfa.ps1`) é outra coisa.
- **MFA opcional por aplicativo autenticador (TOTP) com lista de códigos de recuperação. Backend implementado e mergeado em 2026-11-19 (PRs 455, 456, 457 e 461); o front (Fase 4) foi mergeado em 2026-11-19 (PRs 467 a 471) e a verificação no `dev` (Fase 5) foi feita em 2026-11-20, no navegador, no PWA e no app Android; quem quiser liga o MFA pelo cartão Segurança da conta, em Listas;** o roteiro
  está em [`specs/mfa-opcional/`](specs/mfa-opcional/spec.md) e a revisão de arquitetura em [`bmad/arquitetura-mfa-recuperacao.md`](bmad/arquitetura-mfa-recuperacao.md). O que o desenho assume, e o que ele **não** protege:
  - **Opcional para todos, sem SMS.** Cada pessoa liga e desliga o seu; quem não liga continua só com a senha. A tela de login passa a ser o Managed login do Cognito (plano Essentials), com marca do app; a senha continua nunca passando pelo nosso JavaScript. Testado à mão no `dev` (2026-11-19): o login pede o código depois da senha, no navegador e no app Android.
  - **O token com o escopo de autoatendimento** (`aws.cognito.signin.user.admin`, que permite ligar e desligar o MFA, mas também trocar a senha e apagar o usuário) **não é pedido no login normal**: só num segundo login, ao abrir a seção do MFA, e vive **só em memória** (nem `localStorage`, nem `sessionStorage`, nem o `Cofre`). O backend continua validando **só o ID token**. Ponto não provado: se o Managed login força a senha de novo nesse segundo login (`prompt=login` ou `max_age`); se reaproveitar a sessão por cookie, desligar o MFA não pede prova nova.
  - **A lista de códigos é a única saída que o app oferece** a quem perde o autenticador: sem e-mail, sem prazo, sem pedir ao dono. Perder o autenticador **e** a lista é, para a pessoa, perder a conta, mesmo sabendo a senha; a tela diz isso ("SE PERDER A LISTA, PERDE O ACESSO.") antes de ligar o MFA e **não promete nada além**. Um código só desliga o MFA: **não dá entrada** nem devolve token, e o login continua exigindo a senha.
  - **Último recurso do dono, por fora do app** (decisão do dono, 2026-11-19): o dono é administrador da conta AWS e **pode** desligar o MFA de uma pessoa pelo console do Cognito. Isso **não é oferecido pelo app, não aparece na tela e não é uma promessa a quem usa o app**; o passo a passo fica **fora do git** (o repositório é público). Por isso o texto "nem o dono devolve a conta" foi retirado: ele seria falso.
  - **Aviso quando o MFA muda** (decisão do dono, 2026-11-19; **implementado no backend**, R35 do spec): a pessoa é avisada **pelo canal que já tem ligado** (Telegram ou aparelho, pelo despachante de avisos), sem prometer entrega a quem não tem nenhum. Duas frases fixas, só o fato: "O MFA da sua conta foi desligado com um código de recuperação." (a recuperação com sucesso) e "Uma nova lista de códigos de recuperação foi gerada." (o `200` do confirmar; **não diz que o MFA foi ligado**). **Ligar e desligar em Listas não avisam:** falam direto com o Cognito, o servidor não os vê, e fechar isso exigiria uma rota nova que o dono ainda não decidiu. O aviso é de melhor esforço: a operação espera no máximo 1,5 s pelo envio e responde mesmo assim, e na Lambda o que não terminou pode se perder. **A notificação no aparelho passa a mostrar a frase na tela bloqueada** (exceção à regra "só a contagem" de 2026-11-13; a frase não leva valor, descrição nem meio de pagamento).
  - **A lista nasce pendente e só vale depois de confirmada** (gerar não invalida a anterior; confirmar promove a pendente e apaga a ativa), para a pessoa nunca ficar sem saída por janela fechada ou duas abas. Os hashes (nunca o código) ficam no DynamoDB, em **dois itens por pessoa**, `USER#<sub>` / `MFACODIGOS#ATIVA` e `MFACODIGOS#PENDENTE` (decididos em 2026-11-19, R36 do spec): HMAC-SHA256 com o `listaId` como chave, **sem pepper e sem parâmetro no SSM**; código de 20 símbolos (100 bits), lista de 10, pendente de 1 hora conferida no código (sem TTL do DynamoDB). Sair da conta e sair de um espaço **nunca** apagam a lista.
  - **"MFA ligado implica lista ativa" vale só pela ordem da tela**, não pelo servidor, que não vê o MFA no Cognito (o front fala direto com ele). Quem chamar `SetUserMFAPreference` à mão com o token de escopo amplo pode ligar o MFA sem lista; o dano é da própria pessoa **ou de quem executar código na página durante a seção do MFA** (um XSS pode ligar o MFA com o autenticador do atacante e deixar a vítima sem saída).
  - **A sessão por cookie do login do Cognito no navegador do aparelho** pode devolver o login sem senha e sem código por cerca de 1 hora (não confirmado no pool), e sair do app não a encerra: o MFA protege o login novo, não esse reaproveitamento.
  - **O que a verificação no `dev` provou (2026-11-20, Fase 5):** (1) **revogar o refresh do segundo login revoga também o access token** emitido com ele (`NotAuthorizedException: Access Token has been revoked` no `AssociateSoftwareToken`); por isso o refresh fica só em memória, junto do access token, e é revogado ao fim da seção, ao vencer o token ou ao fechar a página (`web/js/sessao.js`, R10); (2) o segundo login **pede a senha de novo** (`prompt=login`), e as chamadas do navegador ao Cognito a partir do CloudFront **funcionam** (CORS); (3) ligar, desligar e a **recuperação por código** funcionam no navegador, no PWA e no Android (o código gasto não vale de novo, e o login seguinte volta a pedir só e-mail e senha); (4) o `VerifySoftwareToken` tolerou **pelo menos 12 erros seguidos** sem recusar (o limite exato não foi provado), e a **página de login do Cognito** bloqueia a conta por um tempo depois de várias tentativas erradas ("Too many invalid credentials attempts. User temporarily locked."), o que é do Cognito e não nosso.
  - **Tempo da rota pública de recuperação (medido no `dev` em 2026-11-20, 5 chamadas por caso, `POST /mfa/recuperar` com código errado e o e-mail do usuário de teste do `dev`, que tem conta):** sem conta, mediana 107 ms (82 a 114); com conta, mediana 103 ms (101 a 113). **Nenhuma diferença mensurável** nessa amostra (a variação entre chamadas, cerca de 30 ms, é maior que a diferença das medianas). O canal de tempo continua **conhecido e aceito** (R28) e **deve ser medido de novo, com mais chamadas, antes de abrir o app a qualquer pessoa** (item 65).
  - **A lista de códigos copiada ou impressa não leva endereço, nome nem e-mail** (decisão do dono, 2026-11-20): uma lista achada não pode dizer onde usar os códigos. Só os códigos, a frase de aviso e a data.

## O que cada pessoa pode: autorização
**Regra de ouro (`backend/auth/authz.go`): o que não está liberado explicitamente é negado.** Papel ou ação desconhecidos
viram `403`. A decisão é função pura, testada em `authz_test.go`.

O papel vem do item `USER#<sub> / SPACE#<sid>` (campo `role`); sem esse vínculo, `403`. Cada rota é traduzida em uma
**ação** (`rotaParaAcao`); rota sem ação não existe (`404`).

| Ação | owner | member |
|---|---|---|
| `tx.ler` (lançamentos e notas) | sim | sim |
| `tx.criar` | sim | sim |
| `tx.editar` (inclui ocultar e ajustar fatura) | todos | **só os criados por ela** (`criadoPor == sub`) |
| `tx.confirmar` (`POST /tx/{id}/confirmar`: tira o marcador `previsto`) | todos | **só os criados por ela** |
| `tx.excluir` | sim | **não** |
| `tx.importar` (lote) | sim | **não** |
| `historico.ler` (busca por mês e versões de um lançamento) | todos | **só os criados por ela** |
| `historico.restaurar` | todos | **só os criados por ela e que ainda existem**; restaurar um excluído é só do owner |
| `cfg.ler` (listas, tags, combos, saldos, cartões) | sim | sim |
| `cfg.escrever` (renomear/excluir tag, combos, bancos, status, flags, saldo inicial, início do controle (**`cartoes` não é mais gravável por aqui**: o `PUT /cfg/cartoes` saiu na fase C do item 33a; só os comandos de `/cartoes` o alteram) | sim | **não** |
| `cartao.ler` (`GET /cartoes` e `GET /cartoes/{meio}/faturas/{ym}`: as faturas calculadas pelo servidor) | sim | sim |
| `cartao.escrever` (adicionar/remover cartão, início do controle, pagador e vencimento do cartão e da fatura, fechamento e valor do banco por fatura, conferir, desconferir, registrar a diferença) | sim | **não** |
| Trava da **fatura conferida** (`409 fatura_conferida`): criar, editar valor/data/meio/ajuste, excluir, restaurar ou importar uma compra de crédito que muda uma fatura conferida é recusado, sem confirmação que destrave; só `DELETE .../conferir` | vale para todos | vale para todos |
| `tags.adicionar` | sim | sim, **só tags que ainda não existem** (o servidor só acrescenta) |
| `fechamento.ler` (inclui `GET /saldos`, o saldo de cada meio calculado pelo servidor) | sim | sim |
| `fechamento.conferir`, `.reabrir` (o `PUT /fechamentos` e a ação `fechamento.escrever` saíram na fase C do item 40) | sim | **não** (só vê) |
| `aparencia.ler` / `.escrever` | sim | sim, **só a dela** |
| `perfil.editar` (o próprio nome e o **desligar** do Telegram: `telegramChatId` vazio apaga o destino; um Id não vazio é `400`, o chat id só nasce pelo link) | sim | sim |
| `seq.ler` | sim | sim |
| `avisos.testar` (mensagem de teste para o próprio Telegram ou, com `{"canal":"push"}`, para os próprios aparelhos) | sim | sim |
| `push.registrar`, `push.remover` (o aparelho do app no próprio vínculo) e `push.sair` (tira o aparelho de todos os espaços da pessoa) | sim | sim |
| `telegram.vincular` (`POST /spaces/{sid}/telegram/vinculo`: gera o link que liga o Telegram, **só no próprio vínculo**; o `sub` vem do token) | sim | sim |

**Rotas fora de `/spaces/{sid}` e a rota pública do MFA. Implementadas em 2026-11-19** ([`specs/mfa-opcional/spec.md`](specs/mfa-opcional/spec.md), R4 e R25 a R36).**
Hoje `atender` valida o token **antes** de classificar qualquer rota, então uma rota sem token responde `401`. Há **quatro rotas** do MFA, cada uma com ação própria em `auth/authz.go` e teste (e uma linha em [`api.md`](api.md)):

| Rota | Ação | Quem | Token |
|---|---|---|---|
| `GET /me/mfa` (o estado do MFA da própria pessoa e se há lista de códigos ativa; **só lê**, só pelo `sub` do token; nunca hash, `listaId`, código nem e-mail; R37) | `mfa.ler` | qualquer pessoa autenticada | ID token, como hoje |
| `POST /me/mfa/codigos` (gera a lista **pendente**; não toca na ativa) | de gerar (nome fixado no código) | qualquer pessoa autenticada, sempre pelo `sub` do token | ID token, como hoje |
| `POST /me/mfa/codigos/confirmar` (promove a pendente a ativa e apaga a anterior; idempotente; `409 lista_substituida`) | `mfa.confirmar` | qualquer pessoa autenticada, sempre pelo `sub` do token | ID token, como hoje |
| `POST /mfa/recuperar` (um código da lista ativa desliga o MFA da pessoa) | `mfa.recuperar` | **qualquer um**, sem token | **nenhum** |

- **Há duas exceções ao "sem token é `401`", por método e caminho exatos** (a do MFA e a do webhook do Telegram, abaixo). `auth.RotaPublica(metodo, p)` devolve `true` **só** para `POST` `["mfa","recuperar"]` e `POST` `["telegram","webhook"]`; as ações `mfa.recuperar` e `telegram.webhook` **não** são listadas em `Permitido` para nenhum papel. Em `atender`, o método e o caminho são quebrados **antes** de `VerificarToken`; se a rota for uma dessas, o roteador a entrega ao controller **sem ler o cabeçalho `authorization`** (um token inválido nela não muda a resposta); senão tudo segue como hoje. **`token.go` e `VerificarToken` não mudam**, e nenhuma outra rota muda de comportamento. Testes: só aquele método e caminho é público; toda rota autenticada sem token segue `401`; o log não diz `sem-token` numa chamada atendida.
- **A rota pública só desliga o MFA** (`AdminSetUserMFAPreference`): nunca cria sessão, nunca devolve token, nunca altera a senha. Responde **uma só resposta de falha** (mesmo status e corpo) para e-mail sem conta, pessoa sem MFA, código errado ou usado, limite estourado, corpo inválido e falha interna; entrada mínima antes de qualquer trabalho (corpo acima de 2 KiB, JSON inválido ou campo faltando respondem igual).
- **Limite de tentativas, em memória, sem DynamoDB** (R29): por par (pessoa, origem), 10 falhas por hora bloqueiam o par por 1 hora, e por origem, 30 chamadas por minuto. A origem é `r.RequestContext.HTTP.SourceIP` do evento da Function URL, **nunca** cabeçalho (`x-forwarded-for` é de quem chama). A rota **só grava no DynamoDB ao usar um código válido**: e-mail sem conta, formato de código inválido, par bloqueado e código errado não gravam nada, e e-mail sem conta nem lê. Um código vale uma vez, mesmo sob concorrência (escrita condicional), e a marca de usado é desfeita se o Cognito falhar, para um código não ser queimado por falha do Cognito. A comparação do hash é em tempo constante.
- O código só vale para o **usuário (`sub`) a que foi gerado**; o código de uma pessoa nunca desliga o MFA de outra.

**O webhook do Telegram (a segunda rota pública; item 66, fase 2, [`specs/telegram-vinculo/spec.md`](specs/telegram-vinculo/spec.md), R3 a R10).**
- `POST /telegram/webhook` chega sem token do Cognito. **A barreira é o segredo** que o Telegram manda no cabeçalho `x-telegram-bot-api-secret-token`, comparado **em tempo constante** (pelos SHA-256) com o parâmetro `/financas/<amb>/telegram-webhook-secret` do SSM, lido com **cache de no máximo 5 minutos** (rodar o script de registro de novo gera outro segredo). **Falha fechada:** sem segredo configurado, ou cabeçalho ausente, vazio ou diferente, a resposta é `401 {"erro":"nao autenticado"}` **antes de ler o corpo e de qualquer acesso ao DynamoDB**. Ao que o segredo autenticou o webhook responde sempre `200`, e o corpo nunca diz se um código existiu.
- **Entrada mínima:** corpo acima de 16 KiB, JSON inválido, atualização sem `message` ou `text` e chat que não é privado respondem `200 {}` sem trabalho; o mesmo `update_id` na mesma instância é ignorado (o Telegram reenvia se a Lambda demora). O código de vínculo é conferido do mais barato ao mais caro: **formato** (sem tocar o banco), **limite de 10 tentativas por hora por chat** (em memória), **leitura** (1 RCU, **sem escrita** para um código que não existe: a tabela tem 5 WCU) e **exclusão condicional** (o código vale uma vez, mesmo sob concorrência) antes de gravar o chat id.
- **O código** tem 43 caracteres: o `sub` empacotado em 16 bytes (**não é segredo**) e 16 bytes aleatórios (128 bits); só o SHA-256 do aleatório fica no banco, em **um item por pessoa** (`USER#<sub>` / `LINK#telegram`, que o próximo código substitui), válido por 10 minutos. Gerar o link é limitado a 5 por hora por pessoa (em memória, melhor esforço).
- **O chat id vem só do Telegram**, nunca do corpo de uma rota autenticada. **Religar avisa o chat antigo** com um texto fixo, para um link vazado não trocar o destino em silêncio. A confirmação "Ligado ✓" leva o apelido que a própria pessoa deu ao vínculo (exceção pequena à privacidade por padrão, decisão D6). **Nenhuma linha de log leva chat id, texto da mensagem, código, `sub`, `sid`, segredo nem corpo**; o rótulo é `acao=telegram.webhook rota=telegram`.
- **Quem toca num link alheio** passa a receber a contagem do espaço de quem o gerou; só o bloqueio do bot para na hora, e as mensagens cessam na rotina das 8h seguinte (`403` apaga o destino). `/parar` não apaga o destino (D2: sem índice reverso).

Dois detalhes que contam:
- **Registro sem `criadoPor` não é de ninguém:** só o owner mexe nele (`ehDono`).
- **Campos que o cliente não define** (`protegidos`): `PK`, `SK`, `id`, `criadoEm`, `criadoPor`, `excluidoEm` e as marcas de confirmação
  (`confirmaMesFechado`, `confirmouMesFechado`) são descartados do corpo e preenchidos pelo servidor; ninguém grava um lançamento
  "em nome de" outra pessoa, escolhe a própria chave nem se declara excluído ou confirmado.
- **Mês fechado por data (item 38):** escrever em mês anterior ao atual (depois do dia 7, em São Paulo) que muda o saldo exige a
  **confirmação explícita** na query (`confirmaMesFechado=1`); sem ela, `409 mes_fechado`. A decisão é do servidor e do relógio dele,
  nunca do cliente. O **conferido** continua absoluto: a confirmação não o destrava. A confirmação fica gravada na versão do histórico.
- **O `sub` do perfil vem sempre do token**, nunca do corpo: `putPerfil` grava no item de quem fez a chamada.

**O front não é fronteira de segurança.** Esconder um botão para o member é conforto; a recusa é do backend. Regra que
só existe no front não protege nada (veja [`arquitetura.md`](arquitetura.md)).

## Dados e privacidade
- O member **lê todos** os lançamentos e notas e vê todos os bancos e cartões. A visibilidade por banco e cartão é o
  item 27 do backlog (decidido: aplicar no backend, não só esconder no front; banco novo nasce invisível para quem não
  é dono). **Em desenho (2026-10-31, nada implementado):** três níveis por banco ou cartão — Visível, Resumido (some de
  Saldos e Cartão, os lançamentos e os totais continuam, com aviso) e Oculto (some de tudo e fora das somas). Só o Oculto
  esconde o valor de verdade: no Resumido o total completo permite deduzir o saldo do banco escondido por subtração.
  Detalhes e pontos em aberto no item 27 do backlog.
- **Dado real nunca no git.** O `.gitignore` ignora `*.json` (exceção só para o Postman e para
  `.ai/specs/**/*.json`, que são fictícios), `web/config.js`, `web/config.dev.js` e `samconfig.toml`. Exportações reais
  ficam em `/local/`, também ignorado.
- O ambiente `dev` usa **dados fictícios** (`scripts/dados-ficticios.py`), em stack, tabela e login próprios.
- **Telegram é um terceiro, e só recebe a contagem (item 66, fase 1; decisão do dono, 2026-11-20):** o aviso diário leva uma linha
  (`2 atrasados · 1 vence hoje · 1 vence amanhã · 2 vencem nos próximos dias`), **sem descrição, valor, meio de pagamento, data de vencimento nem nome de espaço ou de
  pessoa**, como o push. Antes ele levava a descrição, o valor e o banco de cada lançamento futuro, e isso saía da AWS; deixou de sair.
  A rotina agrupa por **pessoa e destino**: duas pessoas que dividem um chat ou um token recebem mensagens separadas, e uma nunca vê a
  contagem da outra. **O chat id deixou de ser digitado (item 66, fase 3):** `PUT /perfil` recusa um `telegramChatId` não vazio (`400`) e só o vínculo por link o grava, com o `chat.id` que o próprio Telegram mandou; `GET /me` **não devolve mais o chat id** (só `canais: {telegram, push}`), então ele nunca chega ao navegador nem fica no aparelho (o `financas.telegram` de antes é apagado ao abrir o app). Um front antigo, em cache ou instalado, que ainda mande o Id recebe `400` até atualizar. Um erro do Telegram nunca leva ao log o token, a URL, o corpo da resposta nem o chat id.
- **Logs** (CloudWatch, retenção de 14 dias): pelo que o código faz hoje, registram `sub`, papel e ação nas negações,
  contagens da rotina de avisos e mensagens de erro. Nenhum `log` imprime token, e-mail ou corpo da requisição; a regra é
  manter assim (veja "Regras ao mexer").

## Segredos
- A **credencial do Firebase** (JSON da conta de serviço do push) fica no SSM como `SecureString`, em `/financas/<ambiente>/fcm-credencial` (um por ambiente), gravada por `scripts/guardar-fcm.ps1`, que não imprime o conteúdo. A Lambda só lê esse parâmetro. O `google-services.json` do app e qualquer `*-adminsdk-*.json` estão no `.gitignore`. O **token do aparelho** (`pushTokens`) nunca vai ao log nem a uma resposta da API; só a própria pessoa o grava, no próprio vínculo.
- O **token do bot do Telegram** fica no SSM Parameter Store como `SecureString`, em `/financas/<ambiente>/telegram-token`.
  A Lambda só lê esse parâmetro (`SSMParameterReadPolicy` do nome exato) e só decifra pela chave via SSM (`kms:ViaService`).
- **Nenhuma chave de acesso AWS no CI:** o GitHub Actions troca o token OIDC por credenciais temporárias.
- Segredo **nunca** em prompt, chat, commit, linha de comando ou log. Se aparecer, revogar (para o bot: `/revoke` no
  BotFather e gravar o novo no SSM).
- **Um bot por ambiente (item 66, R27):** um bot do Telegram aceita **um webhook só**, então o `dev` e a produção **não podem dividir o bot** (registrar o webhook do `dev` tiraria o da produção do ar). Cada ambiente tem o token em `/financas/<ambiente>/telegram-token`, o segredo do webhook em `/financas/<ambiente>/telegram-webhook-secret` (SecureString) e o usuário do bot em `/financas/<ambiente>/telegram-bot-usuario` (texto), gravados por `scripts/guardar-telegram.ps1` e `scripts/registrar-webhook-telegram.ps1`, **que só o dono roda** (o segundo recusa um bot que já tem outro webhook). **Enquanto o dono não criar o bot do `dev` e rodar os scripts, o link responde `503` e o webhook recusa tudo** (falha fechada). As mensagens do dev começam com `[DEV]`.
- A Lambda só lê esses três parâmetros (uma `SSMParameterReadPolicy` do nome exato para cada um), como já fazia com o token.

## Infraestrutura
- **Front:** S3 privado (bloqueio total de acesso público) servido só pelo CloudFront via OAC; a política do bucket aceita
  apenas a distribuição. HTTPS obrigatório (`redirect-to-https`) e a política de cabeçalhos de segurança gerenciada da
  AWS. O template **não define CSP própria**.
- **API:** Function URL com CORS restrito às origens do front (CloudFront e `localhost`). CORS **não é controle de
  acesso**: quem chama sem navegador ignora; quem protege é o JWT.
- **Lambda:** CRUD na própria tabela (inclui `DeleteItem`), leitura do parâmetro do Telegram e nada mais. **Exceto no
  histórico:** um `Deny` (`HistoricoImutavel`, no `template.yaml`) nega `UpdateItem` e `DeleteItem` em itens cuja `PK` começa
  com `HIST#`, então a Lambda só acrescenta versões. O IAM não impede um `PutItem` sobre uma versão que já existe (ele
  não vê o conteúdo da tabela); isso é coberto pela condição `attribute_not_exists(SK)` em toda escrita de versão, com
  teste. Verificar a negação em dev com `aws iam simulate-principal-policy` (ver `specs/historico/spec.md`, Fase 1).
- **Tabela:** `DeletionPolicy: Retain` e `UpdateReplacePolicy: Retain` (o deploy não apaga a tabela). **PITR ligado só em
  prod** (`!If [EhDev, false, true]` no `template.yaml`; retenção de 35 dias; o `dev` fica desligado). Além dele, só o
  botão Exportar. O backup periódico do item 11 segue adiado.
- **Rate limit:** não há WAF nem teto por IP; o teto de concorrência está desligado porque a conta permite 10 execuções
  no total, o que já funciona como limite natural. **MFA (2026-11-19):** a rota pública de recuperação tem limite de tentativas **no servidor, em memória**, por par (pessoa, origem) e por origem.
- **MFA (2026-11-19, implementado no `template.yaml`; o primeiro deploy no `dev` travou num estilo de marca de teste que sobrou lá e foi apagado à mão):** a role da Lambda ganha, no Cognito, **só** as ações `AdminSetUserMFAPreference` (desligar o MFA) e `AdminGetUser` (achar o `sub` pelo e-mail; **confirmado no `dev` em 2026-11-19**: aceita o e-mail como `Username`, e a busca **diferencia maiúsculas**; conferido pelo dono em 2026-11-19: todos os e-mails de produção estão em minúsculas) **sobre o próprio pool**; as roles do Actions (`infra/github-oidc.yaml`) são atualizadas **antes** do merge. O hash dos códigos é um HMAC-SHA256 com o `listaId` como chave, **sem pepper e sem parâmetro novo no SSM** (decidido em 2026-11-19).

## CI/CD
Detalhes em [`deploy-github-actions.md`](deploy-github-actions.md). Em resumo: cada role OIDC só aceita o repositório
(com ID numérico imutável) e o `environment` do ambiente; as policies são limitadas aos nomes de cada ambiente e negam
`dynamodb:DeleteTable`; o deploy falha se o changeset remover ou substituir a tabela ou o login; produção exige aprovação
manual.

## Pontos em aberto
| Ponto | Risco | Onde acompanhar |
|---|---|---|
| Token do Cognito em `localStorage` (navegador e PWA; no app Android fica no `Cofre`, mas um script injetado ainda consegue lê-lo) | Um XSS no front rouba a sessão | item 18 do backlog; item 54 (refresh em Java) |
| Sem CSP própria | Reduz a defesa em profundidade contra XSS | item 18 |
| Sem WAF nem limite de taxa na Function URL | Enxurrada de requisições (custo e disponibilidade). **Com a rota pública do MFA (em desenho), a rota só grava no DynamoDB ao usar um código válido** (a tabela tem 5 RCU/WCU e já estourou em 2026-10-27); o limite de tentativas é **por par (pessoa, origem), em memória** da Lambda, de melhor esforço, e a rejeição segue da mais barata à mais cara. **Canal conhecido e aceito (decisão de 2026-11-19, reavaliar antes de abrir o app ao público):** o tempo de resposta difere entre e-mail com conta e sem conta (não é igualado, para não gastar capacidade). **Limite conhecido:** um robô ainda pode saturar as 10 execuções da conta; as saídas estruturais, ambas decisão do dono, são pedir aumento da cota de concorrência (teto próprio para a rota pública) ou CloudFront com regra de taxa no WAF | item 18; item 49, fase 1 (`specs/mfa-opcional`, P14, R28, R29) |
| Member lê tudo | Dados de bancos e cartões que não são dela | item 27 |
| MFA é opcional: quem não ligar continua só com a senha | O MFA existe (opcional, TOTP), o front e o backend estão prontos e verificados no `dev`, mas **a pessoa precisa ligá-lo** no cartão Segurança da conta. Quem não ligar continua só com a senha, e uma senha vazada basta para entrar | item 49, fase 1; `specs/mfa-opcional` |
| (MFA) O access token com o escopo `aws.cognito.signin.user.admin` alcança as APIs de autoatendimento do Cognito (trocar a senha, apagar o usuário, mexer no MFA) | Um XSS que o alcance toma a conta e pode ligar ou desligar o MFA; por isso o escopo só é pedido num segundo login e o token vive só em memória. **Não provado:** que o segundo login obriga a senha de novo | `specs/mfa-opcional` (P2, P3, P6) |
| (MFA) "MFA ligado implica lista ativa" só vale pela ordem da tela, não pelo servidor | Quem chamar `SetUserMFAPreference` à mão com o token de escopo amplo liga o MFA sem lista; o dano é da própria pessoa **ou de quem executar código na página durante a seção do MFA** (um XSS pode ligar o MFA com o autenticador do atacante e deixar a vítima sem saída) | `specs/mfa-opcional` (P13; decisões D2 e D3 do dono) |
| (MFA) Perder o autenticador **e** a lista | Para a pessoa a conta se perde, mesmo sabendo a senha: o app não oferece outro caminho. O dono, por ser administrador da conta AWS, pode desligar o MFA pelo console como último recurso, **por fora do app e sem promessa a quem usa**; o passo a passo fica fora do git (decisão D1) | `specs/mfa-opcional/intent.md` |
| (MFA) Quem tem a senha e um código da lista desliga o MFA pela rota pública | Só quem tem um canal de aviso ligado é avisado (decisão D3, implementada: avisar pelo canal ligado, sem prometer entrega a quem não tem); sem canal, nada avisa a pessoa | `specs/mfa-opcional` (D3) |
| Histórico de versões só dos lançamentos, e sem tela de recuperação ainda (PITR só em prod, janela de 35 dias) | Configurações e fechamentos continuam sem versão; recuperar um lançamento ainda é pelo banco | item 31 (fases 2 e 3) |
| Role de dev alcança pool e CloudFront de prod pelas ações listadas | Proteção é o `environment` dev só aceitar a `develop` | `infra/github-oidc.yaml` |
| `excluirDoTotal` montado pelo front a partir do nome do status | **Fechada na fase A do item 41 (2026-10-24):** o servidor calcula `excluirDoTotal` pelo catálogo e o valor do corpo é descartado; o front ainda o envia, ignorado | item 41 (fase C tira o envio) |
| Campos opcionais do lançamento gravados como vêm | Valor inesperado persiste. **`faturaAjuste` já é validado** (só `-1`, `0` ou `+1`, e só em crédito; item 33a, fase A, 2026-10-24); os outros campos opcionais seguem como vêm | itens 33 e 41 |
| Resumo do Telegram acima de 4096 caracteres | **Fechada no item 34 e superada no item 66:** o resumo é uma linha de contagem, sem divisão em partes | item 28, item 34, item 66 |

## Regras ao mexer (propostas, a confirmar com o dono)
- **Rota nova exige ação nova** em `auth/authz.go`, com o que owner e member podem, **e teste** em `auth/authz_test.go`, **e uma linha em [`api.md`](api.md)**
  (o índice das rotas; `backend/api_doc_test.go` falha o `go test ./...` se a rota existir sem linha, se a linha não existir no código ou se a ação ou o papel divergirem de `auth.Permitido`). Nunca
  liberar por padrão.
- **Não afrouxar a validação do token** (`algoritmo`, `iss`, `aud`, `exp`, `token_use`) nem passar a aceitar o access token
  sem decisão do dono. As únicas exceções ao "sem token é `401`" são as de `auth.RotaPublica` (hoje `POST /mfa/recuperar` e `POST /telegram/webhook`), por método e caminho exatos, **antes** do `VerificarToken`, sem mexer nele.
  Qualquer rota nova sem token exige decisão do dono, ação própria, e teste de que nenhuma outra rota ganhou a liberação.
- **Toda escrita confere o papel no servidor**, mesmo que o front já esconda o botão.
- **O cliente nunca manda número derivado como verdade** (decisão de 2026-10-18). Saldo, cache de fechamento, totais,
  faturas, `excluirDoTotal` e qualquer resumo são calculados pelo servidor, que **rejeita** o campo se ele vier no corpo.
  Todo endpoint novo que receba um valor calculável nasce com um teste que tenta forjá-lo. O que o cliente pode mandar é
  dado de entrada (o saldo do extrato, a data escolhida, o valor digitado).
- **O status é do backend:** mudar `status`, `statusReal`, `afetaSaldoBanco` ou `statusFuturo` pela API de configuração
  é **ignorado** (`PUT /cfg/listas` mantém o que está gravado; item 41, fase A). O catálogo de status e o marcador `previsto` são do servidor.
- **Nunca logar** token, e-mail, corpo da requisição, descrição ou valor de lançamento.
- **Dado novo que o member não deve ver** nasce filtrado na origem (backend), não escondido na tela.
- **Permissão nova na infra** exige atualizar as roles do Actions antes do merge (veja o guardrails).
- Mudou uma regra de permissão: atualize **esta tabela e o BACKLOG** na mesma entrega.
