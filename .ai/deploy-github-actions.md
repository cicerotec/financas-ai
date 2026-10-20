# Deploy por GitHub Actions (OIDC)

| Quando | Workflow | O que faz |
|---|---|---|
| push em branch de trabalho (`feature/`, `fix/`, `bug/`, `docs/`, `chore/`, `hotfix/`) ou na `develop` | `ci.yml` | `go vet`, `go test`, os testes do front e `sam validate --lint`. Sem credenciais da AWS. Só `push`, sem `pull_request` (veja o fim deste documento): o check aparece no PR. |
| push na `develop` | `deploy-dev.yml` | publica a stack `financas-dev` (back e front). Sem aprovação. |
| push na `main` | `deploy-prod.yml` | publica a stack `financas`. **Espera a aprovação** do environment `prod`. |

Os dois deploys usam `_deploy.yml`: testes, `sam build`, `sam deploy --no-execute-changeset`, conferência do changeset
(`scripts/verificar-changeset.sh`, falha se a tabela do DynamoDB ou o pool do Cognito forem removidos/substituídos),
execução do changeset e publicação do front. O `config.js` do front é gerado dos Outputs da stack.

**Testes num lugar só:** a action `.github/actions/testes` roda `go vet`, `go test` e **todo `web/*.test.js`** (arquivo
novo entra sozinho). O `ci.yml` e o `_deploy.yml` a usam, então o deploy nunca roda menos que o CI. Ela roda os testes
sem `AMBIENTE`, mesmo no deploy que o define. Erro de AWS no `verificar-changeset.sh` e na limpeza de changesets derruba o
passo (só "stack ainda não existe" é tolerado na limpeza). As ações de terceiros estão **pinadas por SHA** (o comentário
`# vN` diz a versão; para atualizar, troque o SHA) e todo job tem `timeout-minutes`.

## Segurança
- Sem chave de acesso: o Action troca o token OIDC do GitHub por credenciais temporárias.
- Cada role só aceita `repo:cicerotec@24279229/financas-ai@1394222228:environment:<ambiente>` (outro repositório ou outro
  environment não assume). O repositório usa *subject imutável*, por isso o `sub` leva os IDs numéricos; para conferir:
  `gh api repos/cicerotec/financas-ai/actions/oidc/customization/sub`. As policies são limitadas aos nomes de cada ambiente e negam `dynamodb:DeleteTable`.
- Limite conhecido: Cognito e CloudFront não aceitam restrição por nome na criação, então a role de dev também alcança o
  pool e a distribuição de prod pelas ações listadas em `infra/github-oidc.yaml`. A proteção é o environment `dev`
  só aceitar a `develop`.

## Configuração única (cada passo é seu: mexe em IAM e nas configurações do repositório)

1. **Provedor OIDC do GitHub na AWS** (pule se `aws iam list-open-id-connect-providers` já listar `token.actions.githubusercontent.com`):
   ```bash
   aws iam create-open-id-connect-provider --url https://token.actions.githubusercontent.com --client-id-list sts.amazonaws.com
   ```
2. **Roles** (uma por ambiente; use sua sessão com MFA):
   ```bash
   aws cloudformation deploy --stack-name financas-gha-dev  --template-file infra/github-oidc.yaml --capabilities CAPABILITY_NAMED_IAM --parameter-overrides Ambiente=dev  --region sa-east-1
   aws cloudformation deploy --stack-name financas-gha-prod --template-file infra/github-oidc.yaml --capabilities CAPABILITY_NAMED_IAM --parameter-overrides Ambiente=prod --region sa-east-1
   aws cloudformation describe-stacks --stack-name financas-gha-dev --query "Stacks[0].Outputs" --region sa-east-1
   ```
3. **Environments no GitHub** (Settings > Environments):
   - `dev`: *Deployment branches* = só `develop`.
   - `prod`: *Deployment branches* = só `main` e *Required reviewers* = você.
   - Em cada um, *Variables*: `AWS_ROLE_ARN` (Output `RoleArn`) e `COGNITO_DOMAIN_PREFIX` (`financas-cicero-dev` no dev,
     `financas-cicero` no prod). `FRONT_ORIGIN` é opcional (padrão `http://localhost:8080`).
4. Faça um push na `develop` e confira a aba Actions. Depois, o PR `develop → main` publica a produção após a aprovação.

O deploy manual (`scripts/deploy.ps1`) continua funcionando. Se o Action e o deploy manual disputarem a stack, o
CloudFormation recusa o segundo (a stack fica em atualização).

## PRs e releases automáticos

| Quando | Workflow | O que faz |
|---|---|---|
| push em `feature/**`, `fix/**`, `bug/**`, `docs/**` ou `chore/**` | `pr-feature.yml` | abre o PR para a `develop`, **já pronto para mergear** (sem rascunho), se não houver um (PR fechado sem merge não é reaberto). O CI do push já aparece nos checks do PR. |
| `Deploy dev` bem-sucedido | job `pr-release` em `deploy-dev.yml` | abre o PR `develop → main` (se não houver) e comenta a **próxima versão** e a lista do que entra. |
| rótulo ou commits mudam no PR de release | `pr-release-versao.yml` | atualiza esse comentário. |
| `Deploy prod` bem-sucedido | job `lancar-versao` em `deploy-prod.yml` | cria a tag anotada `vX.Y.Z` no commit publicado e a release (`--generate-notes`). |

**Versão** (`scripts/proxima-versao.sh`, a partir da última tag `v*`): algum commit `feat:` sobe o número do meio; senão
sobe o último (`fix:`, `docs:`, `chore:`, `test:` etc.). O rótulo `release:minor` ou `release:patch` no PR de release
manda por cima; vale o que estiver nele quando a tag for criada (depois do deploy da produção). Se o commit já tiver
tag, nada é criado.

Configuração única (sua):
1. Settings > Actions > General > Workflow permissions: marcar **Allow GitHub Actions to create and approve pull requests**.
2. Criar os rótulos:
   ```bash
   gh label create "release:minor" --color 0e8a16 --description "Release: sobe o numero do meio"
   gh label create "release:patch" --color 1d76db --description "Release: sobe o ultimo numero"
   ```
3. Antes do primeiro release automático, a última versão lançada precisa ter tag (o cálculo parte dela).

PR criado com o `GITHUB_TOKEN` não dispara outros workflows de forma normal: o run de `pull_request` de um PR de robô
fica parado em *Approve and run* (aparece vermelho, mas não há nada a aprovar e não bloqueia o merge). Por isso o
`ci.yml` **não** usa `pull_request`: o CI roda só no `push`, que é seu, e o check dele aparece no PR (o da branch no PR
da branch; o da `develop` no PR de release). O primeiro comentário do PR de release é feito pelo próprio job que o cria.

**Antes de aprovar a produção:** o environment `prod` pede a aprovação antes de o job começar, então o changeset ainda
não existe nessa hora. A conferência vem depois do aval: `verificar-changeset.sh` bloqueia sozinho se a tabela do
DynamoDB ou o pool do Cognito forem removidos ou substituídos (os outros recursos não são bloqueados), e a lista de
mudanças fica no log do passo "Conferir o changeset". Para um deploy de mais risco, confira o que mudou em
`template.yaml` no PR de release antes de aprovar.

**Proteção de branch:** hoje não há. A única trava antes da produção é a aprovação do `prod`; o merge na `main` com o CI
vermelho ou um push direto não são impedidos. Se for ligar um ruleset na `main`, exija o check `validar`.
