# Ambiente de teste (`dev`)

Decidido: **opção A**, uma segunda stack inteira (`financas-dev`) com o mesmo template, em vez de só outra tabela ou de
prefixo na tabela real (isolamento fraco, risco de misturar com dados reais). **Dados fictícios**, ambiente **sempre
ligado**, **publicação manual** (automatizar a partir da `develop` fica para depois).
- Hoje há nomes fixos no `template.yaml`: tabela `FinancasApp`, função `financas-api`, pool `financas`; os scripts
  `seed.sh`, `definir-nome.sh` e `limpar-espaco.py` também fixam `FinancasApp`. Uma segunda stack colidiria.
- Parâmetro `Ambiente` (`prod` por padrão, `dev`). O padrão tem que gerar **exatamente os nomes atuais**, senão o
  CloudFormation tentaria substituir a tabela real (o `DeletionPolicy: Retain` já protege, mas conferir o changeset).
  Em dev: `FinancasApp-dev` etc.
- `samconfig.toml` com perfil `dev` (`sam deploy --config-env dev`), com `CognitoDomainPrefix` próprio.
- Scripts com `-Ambiente dev` (`deploy.ps1`, `publicar-front.ps1`, `seed.sh`, `definir-nome.sh`). A guarda de
  publicação passa a aceitar a `develop` quando o alvo for dev (hoje avisa para quem não está na `main`).
- Front: `web/config.dev.js` (URL da API e login do dev), enviado pelo `publicar-front.ps1 -Ambiente dev`; bucket e
  distribuição separados.
- Dados: script que cria um dono e um membro de teste (e-mail "+dev") e carrega dados fictícios com volume parecido
  (status, bancos, cartões, combos, centenas de tags) para testar as telas como a esposa usa.
- Custo: tabela de teste com 5/5 RCU/WCU; o always-free são 25 no total e a produção já usa 5/5, então cabe.
- **Concluído.** Parâmetro `Ambiente` no template, `-Ambiente dev` nos scripts e `--dev` no `limpar-espaco.py`. A
  publicação é **bloqueada** fora da branch do ambiente (`main` para prod, `develop` para dev), sem opção de pular. O
  perfil `dev` do `samconfig.toml` fica só na máquina (o arquivo é ignorado pelo git): stack `financas-dev`,
  `Ambiente="dev"`, `CognitoDomainPrefix="financas-cicero-dev"`. O `web/config.dev.js` (fora do git) leva
  `ambiente: "dev"`, que acende a faixa laranja "AMBIENTE DE TESTE" e o prefixo `[DEV]` no título da aba.
- Dados: `scripts/dados-ficticios.py` (grava só em `FinancasApp-dev`; ids fixos, rodar de novo sobrescreve; recomeçar do
  zero com `limpar-espaco.py <space-id> --dev --confirmar`). Dono e membro criados com `scripts/seed.sh financas-dev`.
- Fica de fora: configuração de **cartões** e **saldo inicial** por banco nos dados fictícios; as despesas superam as
  entradas, então os saldos ficam negativos (serve para testar esse caso). A URL é a do CloudFront; domínio próprio só
  se entrar domínio para a produção.

## Rodar a API na sua máquina (depurar com breakpoints)

Com a variável `LOCAL_PORT` definida, o `main` sobe um servidor HTTP em vez de `lambda.Start` (`backend/local.go`) e entrega cada
requisição ao **mesmo handler** da Lambda. Na AWS a variável não existe e nada muda. Chame pelo Postman (`baseUrl` =
`http://127.0.0.1:8090`) ou aponte o front para ela. **O token do Cognito continua sendo validado de verdade**, então use o
`id` de quem está logado no app de **dev**.

**Travas (porque isto usa as suas credenciais da AWS):** só roda com `AMBIENTE=dev` (sem isso o programa recusa) e escuta só em
`127.0.0.1`. A tabela é a de dev; nunca aponte `TABLE_NAME` para a de produção.

| Variável | Valor em dev | De onde vem |
|---|---|---|
| `LOCAL_PORT` | `8090` | a porta da API (a do front, de `servir.ps1`, é a 8080) |
| `AMBIENTE` | `dev` | fixo |
| `AWS_REGION` | `sa-east-1` | fixo (também monta o `iss` do token) |
| `TABLE_NAME` | `FinancasApp-dev` | fixo |
| `USER_POOL_ID` | id do pool de dev | Output `UserPoolId` da stack `financas-dev` |
| `CLIENT_ID` | id do client de dev | Output `ClientId` (é o `clientId` do `web/config.dev.js`) |
| `TELEGRAM_TOKEN_PARAM` | `/financas/dev/telegram-token` | só para avisos e a rota de teste; sem isso o resto funciona |
| credenciais da AWS | sessão com MFA | `. .\scripts\aws-mfa.ps1 123456` |

Os Outputs saem com (na sessão com MFA):

```powershell
aws cloudformation describe-stacks --stack-name financas-dev --region sa-east-1 --query "Stacks[0].Outputs[?OutputKey=='UserPoolId' || OutputKey=='ClientId'].[OutputKey,OutputValue]" --output text
```

**No VS Code**, o depurador herda o ambiente de quem abriu o editor. Por isso: abra o terminal, rode `. .\scripts\aws-mfa.ps1 <código>`
e, **nesse mesmo terminal**, `code .` (ou feche e reabra o VS Code por ele). Credenciais nunca vão para o `launch.json`. O arquivo
`.vscode/launch.json` fica fora do git (o `.gitignore` ignora `*.json`), mas mesmo assim só ponha ali o que não é segredo:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "API local (dev)",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/backend",
      "env": {
        "LOCAL_PORT": "8090",
        "AMBIENTE": "dev",
        "AWS_REGION": "sa-east-1",
        "TABLE_NAME": "FinancasApp-dev",
        "USER_POOL_ID": "<UserPoolId de dev>",
        "CLIENT_ID": "<ClientId de dev>",
        "TELEGRAM_TOKEN_PARAM": "/financas/dev/telegram-token"
      }
    },
    {
      "name": "Testes do backend",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}/backend"
    }
  ]
}
```

Para o front usar a API local, troque o `apiUrl` do `web/config.dev.js` (que também fica fora do git) por `http://127.0.0.1:8090` e suba o
front com `.\scripts\servir.ps1`. O modo local já responde o CORS de `localhost`.
