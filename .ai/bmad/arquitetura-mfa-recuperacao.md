---
name: 'MFA opcional: rota pública de recuperação e lista em duas fases'
type: architecture-spine
purpose: build-substrate
altitude: feature
paradigm: 'Camadas do financas-ai; a rota pública é uma etapa explícita e mínima do roteador, antes da validação do token'
scope: 'Duas decisões que a spec mfa-opcional precisa absorver antes da Fase 3: onde a rota pública de recuperação entra no roteador (A1) e como a lista de códigos é gerada e ativada sem deixar a pessoa sem saída (A5)'
status: handoff
created: '2026-11-19'
updated: '2026-11-19'
binds: ['R4, R16, R25, R27, R28, R29, R31, R33 e M-T13, M-T21, M-T27 de .ai/specs/mfa-opcional/spec.md']
sources: ['financas-ai/.ai/specs/mfa-opcional/{intent,plan,spec}.md', 'financas-ai/.ai/guardrails.md', 'financas-ai/.ai/security.md', 'financas-ai/backend/roteador.go, local.go, auth/authz.go, auth/token.go, repository/espacos.go (lidos em 2026-11-19)', 'revisão de arquitetura de 2026-11-19, achados A1 e A5 (E1 e E2 como decorrência de A5)']
---

# MFA opcional: rota pública de recuperação e lista em duas fases

Repassado pelo dono em 2026-11-19 ao agente do financas-ai, para atualizar `plan.md` e `spec.md` e implementar. Onde divergir de `guardrails.md`, `arquitetura.md` ou `decisoes.md`, a divergência vai ao dono antes de qualquer código. **Nada no DynamoDB é criado antes da conversa com o dono** (AD-2, "O que fica para o dono").

## O que a revisão achou, em duas linhas

- **A1:** `atender` (`backend/roteador.go:39`) valida o token antes de classificar qualquer rota. Uma rota sem token, como `POST /mfa/recuperar`, hoje responde `401` e nunca chega ao controller. A spec fala em "ação liberada sem token" sem dizer onde isso entra.
- **A5:** R25 invalida a lista anterior ao **gerar** a nova. Com o MFA já ligado, se a pessoa fechar a janela, perder a resposta ou abrir duas abas antes de guardar a lista nova, fica sem nenhuma lista válida e sem saída.

## Invariantes herdados

| Herdado | De | Vale aqui |
| --- | --- | --- |
| O que não está liberado explicitamente é negado; rota nova exige ação nova com teste | `guardrails.md`, `security.md` | AD-1: a rota pública é a única exceção, por método e caminho exatos |
| Não afrouxar a validação do token | `security.md`, "Regras ao mexer" | AD-1: `token.go` e `VerificarToken` não mudam; muda a ordem no roteador, para uma rota só |
| Nenhuma chave ou campo novo no DynamoDB sem o dono | `guardrails.md` | AD-2 descreve o modelo lógico; o formato físico é do dono |
| Nunca logar token, e-mail, corpo | `security.md` | AD-1: a linha de log da rota pública segue o padrão de `linhaDeLog` |
| Camadas `controller` → `service` → `domain`; adaptador atrás de interface do service; `arquitetura_test.go` passa | `guardrails.md` | AD-1 e AD-2 |

## AD-1: a rota pública é uma etapa do roteador, antes do token

- **Binds:** `backend/roteador.go` (`atender`, `rotular`), `backend/auth/authz.go`, `backend/local.go`, R4, R27, R28, M-T27.
- **Prevents:** rota pública inalcançável (`401`), ou, pior, a validação do token afrouxada para "deixar passar" quando o caminho parecer público.

**Regra**

1. **Classificação pura em `auth`, ao lado de `RotaSemEspaco`:**

   ```go
   // RotaPublica classifica as rotas que valem SEM token (hoje so a recuperacao do MFA). Metodo e caminho exatos; o resto e false.
   func RotaPublica(metodo string, p []string) (Acao, bool) {
       if metodo == "POST" && len(p) == 2 && p[0] == "mfa" && p[1] == "recuperar" {
           return AcaoRecuperarMFA, true // "mfa.recuperar"
       }
       return "", false
   }
   ```

   `AcaoRecuperarMFA` é uma `Acao` nova. `Permitido` **não** a lista para nenhum papel: ela não passa por papel. A exceção é só `RotaPublica`.
2. **Ordem em `atender`:** quebrar o caminho (`p`) e o método (`m`) **antes** de `VerificarToken`; chamar `RotaPublica`; se for pública, gravar o rótulo e entregar ao controller **sem ler o cabeçalho `authorization`**; senão, seguir como hoje (`VerificarToken`, `401` se falhar).
   - Um token inválido numa rota pública **não** muda a resposta (R28): o cabeçalho é ignorado.
   - Nenhuma outra rota muda de comportamento. `token.go` e `VerificarToken` ficam intactos. **Reescrever R4**: "a validação do token não muda; o roteador ganha uma etapa anterior que só reconhece `POST /mfa/recuperar`".
3. **Origem do limite (R29):** o controller recebe a origem de `r.RequestContext.HTTP.SourceIP` (campo do evento da Function URL), **nunca** de cabeçalho (`x-forwarded-for` é controlado por quem chama). O `eventoDe` de `local.go` hoje não preenche esse campo: preencher a partir de `r.RemoteAddr` (sem a porta), para o modo local e os testes.
4. **Entrada mínima, antes de qualquer trabalho:** corpo maior que 2 KiB, JSON inválido ou campo faltando devolvem **a mesma resposta genérica** das outras falhas (R28). Só depois lê limite e código.
5. **Resposta única de falha** (R28): um só status e um só corpo constantes para e-mail sem conta, pessoa sem MFA, código errado ou usado, limite estourado, corpo inválido e falha interna do Cognito ou do DynamoDB que não seja sucesso. Sucesso: `200` com corpo fixo. O texto na tela cobre todos os casos ("Confira o e-mail e o código. Depois de várias tentativas erradas, aguarde alguns minutos."), sem o servidor distinguir.
6. **Camadas:** `controller.Recuperacao` recebe e devolve; `service.Recuperacao` decide e declara as portas (`CodigosDeRecuperacao` para os hashes e o limite, `DesligadorDeMFA` para o Cognito); o adaptador do Cognito mora em `adapter/identidade/` (padrão de `adapter/notificacao/`); o `main` registra. O `service` não importa a AWS.
7. **Log:** a rota ganha `acao=mfa.recuperar rota=mfa` por `rotular`. A linha **não** leva e-mail, origem, código nem corpo (`linhaDeLog` já só aceita campos fixos). Falha interna loga só o nome do erro, sem o e-mail.
8. **Rótulo do log:** o `handler` inicia o rótulo com `acaoSemToken` e `atender` só o troca depois de validar o token. A rota pública grava o próprio rótulo (`*rt = ...`) antes de entregar ao controller, para a linha não dizer `sem-token` numa chamada atendida.

**Testes (todos falham antes da mudança)**

| Id | O que prova |
| --- | --- |
| M-T27a | `RotaPublica` devolve `true` **só** para `POST` `["mfa","recuperar"]`; `false` para `GET`, `PUT`, `["mfa","recuperar","x"]`, `["mfa"]`, `["me"]`, `["push","sair"]`, `["spaces",...]`, caixa diferente e `["mfa","","recuperar"]` |
| M-T27b | No roteador, toda rota **autenticada** conhecida (as de `RotaParaAcao`, `/me`, `/push/sair`, `/me/mfa/codigos` e `/me/mfa/codigos/confirmar`) sem token continua `401` |
| M-T27c | `POST /mfa/recuperar` **sem** `authorization` chega ao controller; com `authorization` lixo ou expirado dá a **mesma** resposta; `VerificarToken` não é chamado nessa rota |
| M-T27d | Corpo vazio, JSON inválido, corpo de 3 KiB e e-mail ausente respondem **igual** a código errado (R28) |
| M-T27e | A origem do limite vem de `SourceIP`; um `x-forwarded-for` falso não muda a chave do limite |

**Quebrar de propósito** (guardrails): (a) fazer `RotaPublica` devolver `true` para qualquer `POST` tem de falhar M-T27a e M-T27b; (b) mover `VerificarToken` para depois do `switch` inteiro tem de falhar M-T27b.

**Documentos a atualizar na mesma entrega:** `security.md` (nova seção "Rota pública", a regra "validação do token" com a ordem nova, e a linha de pontos em aberto sobre WAF e limite de taxa, agora com uma rota pública que escreve no DynamoDB); R4, R27 e M-T27 do `spec.md`.

## AD-2: a lista nasce pendente e só vale depois de confirmada

- **Binds:** R16, R25, R26, R27, R30, R32, M-T13, M-T21, M-T22; `backend/roteador.go:109` (`processar` só trata `AcaoSairPush` nas rotas sem espaço).
- **Prevents:** a pessoa ficar sem lista válida ao regerar (janela fechada, resposta perdida, duas abas), e o mesmo código valendo duas vezes sob concorrência.

**Modelo lógico** (o formato físico é do dono, ver "O que fica para o dono")

- Por pessoa (`sub`): no máximo **uma lista ativa** e **uma pendente**. Cada lista tem um `listaId` aleatório e os hashes dos códigos, cada um com `usado`.
- Só a lista **ativa** vale na rota pública. A pendente nunca recupera nada.
- Uma pendente expira sozinha (prazo a definir com o dono), para não acumular hashes de listas abandonadas.

**Regras**

1. **Gerar** (`POST /me/mfa/codigos`, autenticada, corpo vazio aceito): cria uma lista **pendente**, substituindo a pendente anterior se houver, e devolve `{listaId, codigos}` **uma única vez**. **Não toca na lista ativa.** Reescrever R25: "gerar não invalida a anterior".
2. **Confirmar** (`POST /me/mfa/codigos/confirmar`, autenticada, corpo `{"listaId": "..."}`): promove a pendente a ativa **e** apaga a ativa anterior, em **uma escrita condicional** (a pendente existe, tem aquele `listaId` e não expirou). Idempotente: confirmar de novo um `listaId` que já é o da ativa devolve `200`. Pendente substituída por outra geração ou expirada: `409` com `lista_substituida`; a ativa continua como estava. Rota nova, ação própria `mfa.confirmar` em `authz.go`, liberada a qualquer pessoa autenticada, sempre com o `sub` do token (nunca do corpo).
3. **Ordem da tela (reescrever R16):** (1) `AssociateSoftwareToken`; (2) QR e chave; (3) `VerifySoftwareToken` = `SUCCESS`; (4) `POST /me/mfa/codigos` (pendente); (5) a tela mostra a lista com a frase em destaque, e a pessoa toca em "Guardei a lista"; (6) **só então** `POST /me/mfa/codigos/confirmar`; (7) se o passo 6 deu `200`, `SetUserMFAPreference`; (8) `GetUser`. Regerar com o MFA ligado usa só os passos 4 a 6. O botão de "Guardei a lista" só vale depois de a lista aparecer; o confirmar **nunca** é chamado antes do toque.
4. **Falhas e o que sobra:**

   | Onde falha | Estado depois | A pessoa |
   | --- | --- | --- |
   | Passo 4 (resposta perdida) | Pendente criada, ativa intacta | Tenta de novo; a nova pendente a substitui |
   | Passo 5 (janela fechada) | Pendente órfã, ativa intacta | Nada se perde; a pendente expira |
   | Passo 6 (resposta perdida, mas o servidor confirmou) | A nova é a ativa | A tela mantém a lista aberta e repete o confirmar com o mesmo `listaId`; é idempotente |
   | Passo 7 | Lista ativa, MFA desligado | Inofensivo; a tela relê o estado e oferece repetir do início |
   | Duas abas | A última pendente vale; o confirmar da outra dá `409` | A janela da outra aba diz para gerar de novo |

5. **Uso de um código (rota pública, R27), com concorrência:**
   - Marcar o código como usado é uma escrita **condicional** (`usado` ainda falso, lista ainda a ativa). Duas chamadas com o mesmo código: só uma passa (R26 fecha sob concorrência).
   - Ordem: marcar → `AdminSetUserMFAPreference` → se o Cognito falhar, **desfazer a marca** (condicional ao mesmo `listaId`) e devolver a resposta genérica; o código não é queimado por uma falha do Cognito.
   - Sucesso: a lista ativa inteira é apagada ("usar um código invalida o resto"). Depois de recuperar, ligar de novo passa pelo passo 4 em diante e cria uma lista nova.
6. **O que o servidor não garante, dito com franqueza:** o servidor não vê o estado do MFA no Cognito (o front fala direto com ele). "MFA ligado implica lista ativa" é imposto pela **ordem da tela** (passo 7 depende do 6), não pelo servidor. Quem chamar `SetUserMFAPreference` à mão com o token de escopo amplo pode ligar o MFA sem lista; o dano é da própria pessoa e vem do mesmo escopo que o `plan.md` já trata como sensível. Registrar isto em `security.md` e em P4/P6.
7. **Hash e entropia (decidir antes de implementar, mas dentro desta decisão):** com código de 100 bits ou mais, **HMAC-SHA256 com pepper no SSM** (`SecureString`, um por ambiente) e busca por id do código resolvem sem hash lento; bcrypt ou argon2 só se o código for curto. Escolher uma das duas e fixar o comprimento no spec da Fase 3; a escolha muda o custo de CPU da Lambda pública (a conta tem 10 execuções).

**Testes (todos falham antes da mudança)**

| Id | O que prova |
| --- | --- |
| M-T21a | Gerar com lista ativa existente **não** invalida a ativa: um código dela ainda recupera |
| M-T21b | Um código da lista **pendente** não recupera nada na rota pública (resposta genérica) |
| M-T21c | Confirmar com o `listaId` certo promove a pendente e apaga a ativa anterior; o código antigo deixa de valer e o novo vale |
| M-T21d | Confirmar é idempotente; confirmar `listaId` de pendente substituída ou expirada dá `409` e não muda a ativa |
| M-T21e | `confirmar` sem token é `401`; com token de outra pessoa não mexe na lista desta (o `sub` vem só do token) |
| M-T22a | Dois usos simultâneos do mesmo código: só um sucesso; o outro, resposta genérica |
| M-T22b | `AdminSetUserMFAPreference` falha: a marca de usado é desfeita e o código ainda vale |
| M-T13 (emenda) | `SetUserMFAPreference` **não** é chamado sem o `200` do confirmar, nem sem o toque em "Guardei a lista" |

**Quebrar de propósito:** (a) fazer o gerar apagar a ativa tem de falhar M-T21a; (b) tirar a condição `usado` do uso do código tem de falhar M-T22a.

## As rotas, em resumo (o contrato completo está em `.ai/specs/mfa-opcional/spec.md`, seção "API")

```
POST /me/mfa/codigos            (autenticada)               -> 200 { listaId, codigos } | 401 | 500          gera a lista PENDENTE
POST /me/mfa/codigos/confirmar  { listaId } (autenticada)   -> 200 { ok } | 401 | 409 lista_substituida | 400  promove a pendente
POST /mfa/recuperar             { email, codigo } (PÚBLICA) -> 200 { ok } | 400 recuperacao_invalida         única falha (R28)
```

O status e o texto da resposta única de falha (`400 recuperacao_invalida`) foram fixados no spec; esta spine só exige que seja **um só** para todos os casos (AD-1, regra 5).

## O que fica para o dono

Nada abaixo é criado antes da conversa (guardrails). O que a conversa precisa fechar, com uma proposta para discutir:

1. **Onde moram os hashes.** Proposta: um item por pessoa e por estado na partição que já existe, `PK = USER#<sub>` com `SK = MFACODIGOS#ATIVA` e `SK = MFACODIGOS#PENDENTE`, cada item com `listaId`, a lista de `{id, hash, usado}` e, na pendente, um atributo de expiração (TTL do DynamoDB). Conferido no código: a única consulta que lê `USER#<sub>` (`repository/espacos.go`) filtra `begins_with(SK, "SPACE#")`, então esses itens novos não aparecem nas listas de espaços.
2. **Controle de tentativas da rota pública.** Itens com chave por **hash do e-mail normalizado** e por origem, **com TTL curto**, para que e-mails inexistentes não acumulem. Escritas não autenticadas em tabela 5/5 RCU/WCU (que já estourou em 2026-10-27): ler o contador por origem antes de qualquer outra coisa e fixar um teto de gravações por origem. O dono decide números (tentativas, janela, TTL); sem eles M-T24 não é escrevível.
3. **Pepper no SSM** (`/financas/<ambiente>/mfa-pepper`), se o HMAC for a escolha.
4. **Comprimento do código, quantidade por lista e prazo da pendente.**

## O que o agente do financas-ai faz, na ordem

1. Atualiza `plan.md` e `spec.md` (sem implementar): R4, R16, R25 (e nova R25a para o confirmar), R27, M-T13, M-T21 (a a e), M-T22 (a e b), M-T27 (a a e); corrige as contradições já apontadas (plan linhas 5 e 97, decisões 2 e 3 do plan, "uma ação só" do intent contra R32). Mantém a regra de premissas: acrescenta uma premissa para cada dependência desta spine (a ordem do roteador, o passo 7 depender do passo 6) e pede a revisão independente de novo.
2. Atualiza `security.md` (rota pública, ordem do roteador, MFA↔lista só pela ordem da tela, limite que grava no DynamoDB).
3. Leva ao dono a conversa do DynamoDB (seção acima) com as perguntas já formuladas.
4. Só depois, Fase 3: primeiro `RotaPublica` e a ordem do `atender` (AD-1) com os testes M-T27; depois o repositório e o service da lista (AD-2).
5. As fases 1 (demo), 2 (infra) e 4 (front) seguem o `plan.md`; o front muda só pelo passo 6 e pela mensagem de `409`.

## O que esta spine não resolve

Os achados A2, A3, A6, A9 e A12 da revisão (texto "nem o dono devolve", tomada do MFA por XSS, hash lento contra entropia, aviso ao desligar, como guardar a lista) são decisões do dono ou da demo; esta spine só reserva o lugar do hash (AD-2, regra 7). Os achados A7, A8, A10, A11, A13 a A16 e E3 a E7 seguem como ajustes de spec a fazer pelo agente de lá.
