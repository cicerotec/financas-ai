# Deploy por GitHub Actions (OIDC)

| Quando | Workflow | O que faz |
|---|---|---|
| PR para `develop` ou `main` | `ci.yml` | `go vet`, `go test`, `sam validate --lint`. Sem credenciais da AWS. |
| push na `develop` | `deploy-dev.yml` | publica a stack `financas-dev` (back e front). Sem aprovação. |
| push na `main` | `deploy-prod.yml` | publica a stack `financas`. **Espera a aprovação** do environment `prod`. |

Os dois deploys usam `_deploy.yml`: testes, `sam build`, `sam deploy --no-execute-changeset`, conferência do changeset
(`scripts/verificar-changeset.sh`, falha se a tabela do DynamoDB ou o pool do Cognito forem removidos/substituídos),
execução do changeset e publicação do front. O `config.js` do front é gerado dos Outputs da stack.

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
