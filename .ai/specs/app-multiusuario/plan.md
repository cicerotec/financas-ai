# Plan: app multiusuário

Como construir o que o [`intent.md`](intent.md) pede. Cada fase é um PR pequeno, vai primeiro ao `dev` e deixa o app seguro para a fase seguinte. **A fase 1
vem antes de qualquer abertura de cadastro.** Telas novas pedem demo aprovada antes do código.

**Estado:** rascunho para aprovação (2026-10-27). Nada implementado.

## Onde estamos hoje

| Fato | Onde | Consequência |
|---|---|---|
| Cognito só por convite, sem MFA | `template.yaml` (`UserPool`), `.ai/security.md` | Abrir exige trocar `AllowAdminCreateUserOnly` e ligar o MFA |
| Cliente do pool sem segredo, fluxo `code` + PKCE, login pelo Hosted UI | `UserPoolClient` | O cadastro com código precisa de tela própria; o Hosted UI não tem campo de código |
| O espaço nasce no `seed.sh` (itens `USER#<sub>/SPACE#<id>` e `SPACE#<id>/META`) | `scripts/seed.sh` | A criação passa para o backend, com os mesmos itens (sem chave nova) |
| `GET /me` lista `EspacosDe(sub)` e só lê | `controller/espacos.go`, `service/espacos.go` | Quem não tem espaço recebe lista vazia; leitura não deve gravar |
| Toda rota exige JWT validado e papel no espaço | `auth/token.go`, `auth/authz.go` | Criar espaço é rota **fora** de `/spaces/{sid}`, com ação própria |
| Lambda tem `Deny` de `UpdateItem`/`DeleteItem` em `HIST#*` | `template.yaml` (`HistoricoImutavel`) | Excluir conta não pode ser feito pela Lambda comum |
| Function URL `AuthType: NONE`, sem limite de taxa; tabela 5/5 | `template.yaml` | Abertura sem teto é risco de custo e de indisponibilidade |
| `ReservedConcurrency` já é parâmetro opcional | `template.yaml` (`TemTetoDeConcorrencia`) | Teto de uso grátis já existe; falta decidir o valor |

## Fases (cada uma é um PR)

### Fase 1: proteger antes de abrir (itens 37 e 11)
- **MFA TOTP** no pool, nos dois ambientes (item 37): sem SMS; confirmar no `dev` como a pessoa cadastra o aplicativo no primeiro acesso; **recuperação**
  se o celular se perder (reset por script do dono, para ninguém ficar trancada); decidir obrigatório x opcional.
- **AWS Budget de US$ 1** com alerta por e-mail (a parte de custo do item 11); o backup periódico segue no plano próprio.
- **Limite de taxa:** definir o `ReservedConcurrency` de prod (e, se o dono aceitar o custo, WAF na frente). Medir antes: quantos acessos a tela faz por abertura.
- **Capacidade da tabela:** decidir 5/5, autoscaling dentro de 25 unidades ou sob demanda; mudar de modo no `template.yaml` não pode substituir a tabela (o deploy já barra isso).
- Atualizar `.ai/security.md` (autenticação) e o item 37 do BACKLOG.

### Fase 2: cadastro com código de convite
O Hosted UI não pede código, e ligar o auto cadastro do pool deixa **qualquer um chamar `SignUp` direto no Cognito**. Por isso o código é conferido **no próprio
Cognito**, não numa rota pública nova:
- `AllowAdminCreateUserOnly: false` no pool.
- **Gatilho `PreSignUp`** (uma Lambda pequena, em Go como o resto): lê o código de `ClientMetadata`, compara em tempo constante com o valor guardado no **SSM
  Parameter Store** (SecureString, mesmo lugar do token do Telegram) e **recusa** sem código ou com código errado. Trocar ou apagar o parâmetro revoga o convite.
  O gatilho **não** confirma o usuário nem o e-mail: a confirmação continua sendo por código enviado pelo Cognito ao e-mail.
- **Tela de cadastro própria** no front (e-mail, senha, código de convite, depois o código de confirmação do e-mail), chamando `SignUp` e `ConfirmSignUp` do
  Cognito por `fetch`; o login segue pelo Hosted UI. O link "Sign up" do Hosted UI continua aparecendo, mas o gatilho o recusa sem código; esconder o link é
  ajuste de aparência a decidir.
- Códigos por pessoa ou com limite de usos pedem registro: **parâmetros no SSM** (sem chave nova no DynamoDB) ou **chave nova, só com aprovação do dono**.
- Testes: sem código, com código errado e com código revogado a conta **não** nasce (a Lambda do gatilho tem teste de unidade; o fluxo vai ao `dev`).
- **Demo da tela de cadastro antes do código.** Atualizar `.ai/security.md`.

### Fase 3: espaço próprio no primeiro acesso
- Nova rota **`POST /me/espaco`** (idempotente): se a pessoa não tem espaço, cria `USER#<sub>/SPACE#<id>` com `role: owner` e `SPACE#<id>/META` com o nome e o
  `ownerSub`, **os mesmos itens do `seed.sh`**, numa `TransactWriteItems`. O id é derivado do `sub` (ou o put é condicional), para duas chamadas ao mesmo
  tempo nunca criarem dois espaços.
- Ação própria em `authz.go` (`espaco.criar`), liberada a qualquer pessoa autenticada **sem** vínculo; continua valendo "o que não está liberado é negado".
  Só cria se `EspacosDe(sub)` vier vazio; falha de leitura é `500` e nunca cria às cegas.
- O front chama a rota quando `GET /me` volta sem espaços (leitura não grava). Camadas: `controller` recebe, `service` decide, `repository` grava; teste de
  arquitetura (`arquitetura_test.go`) deve passar.
- Testes: primeira chamada cria; segunda não duplica; duas simultâneas criam uma; quem já tem espaço não ganha outro; `sub` vazio é recusado.
- O `seed.sh` continua para convidar `member` e para o `dev`.

### Fase 4: o app vazio e a primeira tela
- Conferir no backend, **com teste**, que um espaço sem nenhum `CFG#...` funciona: listas, status, saldos, Futuros, Histórico e Tendências respondem vazio, sem
  `500`, e nenhum nome do dono (bancos, tags, status) vaza como padrão. O catálogo de status vem do servidor e é regra do produto, não dado do dono.
- Tela de **boas-vindas** (demo antes): dizer o que fazer primeiro (criar o primeiro meio de pagamento e o início do controle), com o padrão de janelas do app e
  seletores nativos do celular. O front só exibe; qualquer regra vem da API.
- O Telegram continua opcional por pessoa (`telegramChatId` já é por usuário); revisar o texto da tela de aviso para quem nunca configurou um bot.

### Fase 5: isolamento provado, exportar e excluir
- **Teste de isolamento:** dois espaços no mesmo banco de teste; para cada rota de `RotaParaAcao`, uma pessoa de um espaço pedindo o `sid` do outro recebe
  `403`; `GET /me` lista só os próprios. Falha de qualquer rota quebra o CI.
- **Sem acesso do dono pelo app:** conferir que nenhuma rota, tela ou script do repositório (fora do `seed.sh` e dos scripts operacionais já existentes) lê o
  espaço de outra pessoa; registrar em `security.md` o que o dono alcança pelo console e o limite honesto (intent).
- **Exportar a conta:** o Exportar de hoje não leva `HIST#`, `SALDO#*` nem `SEQ`; decidir o que a pessoa precisa receber (o JSON atual basta como "meus dados" ou é
  preciso um export completo).
- **Excluir a conta:** apaga o espaço (`SPACE#<id>` inteiro, incluindo `HIST#`, que a Lambda comum **não pode** apagar) e o usuário do Cognito. Pede um caminho
  administrativo separado (role própria, acionada só pela própria pessoa, confirmada por senha ou MFA); decisão de desenho antes do código, com o dono.
  Espaço compartilhado: sair do espaço não é o mesmo que excluí-lo.
- Política de privacidade e termos de uso publicados antes da abertura.

### Fase 6: abrir (no `dev` primeiro, depois prod)
- Rodar o roteiro "pronto quando" do intent no `dev` com pessoas de teste; só então liberar o código em prod, para poucas pessoas, com o Budget ativo e o alarme de
  custo conferido.
- Plano de revogação: trocar o código no SSM; o que fazer com contas que já entraram (nada, a menos que haja abuso).

## Riscos

| Risco | Resposta |
|---|---|
| Auto cadastro habilitado e `SignUp` chamado direto, sem passar pela tela | O gatilho `PreSignUp` é a barreira; sem código válido a conta não nasce |
| Força bruta no código de convite | Código longo e aleatório; o Cognito limita tentativas de `SignUp`; rotacionar no SSM se houver sinal de ataque |
| Custo ou indisponibilidade por uso alto | Fase 1 antes: Budget, teto de concorrência e decisão sobre a tabela |
| Pessoa vê dado de outra por erro de código | Teste de isolamento por rota no CI (fase 5); `403` por padrão |
| Criar dois espaços numa corrida | Id derivado do `sub` ou put condicional, com teste de duas chamadas simultâneas |
| Excluir conta deixa o histórico para trás | Caminho administrativo próprio, testado, que cobre `HIST#` e o Cognito |
| Promessa de "nem o admin" maior que a realidade | Escrita explícita no intent, na política de privacidade e em `security.md`; cifragem por campo é outra entrega |
| Dado financeiro de terceiros traz obrigação legal (LGPD) | Política, exportar e excluir conta antes de abrir; revisar com alguém da área se o uso crescer |

## Ordem recomendada

Fases 1, 2, 3 e 4 entregam o fluxo de ponta a ponta no `dev`; a 5 fecha o que impede abrir em prod; a 6 é a abertura. A fase 1 é a única que **não** depende
de decisão nova de produto e pode começar já.

## Decisões que o plan fixou e o dono pode contestar

1. O convite é conferido por um **gatilho `PreSignUp` do Cognito**, com o código no SSM, e não por rota pública da API.
2. A criação do espaço é uma **rota explícita** (`POST /me/espaco`), não um efeito colateral de `GET /me`.
3. O espaço novo reaproveita os itens do `seed.sh`: **nenhuma chave nova** no DynamoDB nesta entrega (códigos com limite de usos ficam como decisão).
4. A exclusão de conta usa **caminho administrativo próprio**, sem enfraquecer o `Deny` de `HIST#*` da Lambda comum.
5. "Nem o admin" vale como **isolamento lógico** agora; a garantia criptográfica fica com a cifragem por campo.
