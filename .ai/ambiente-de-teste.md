# Ambiente de teste (`dev`)

Decidido: **opção A**, uma segunda stack inteira (`financas-dev`) com o mesmo template, em vez de só outra tabela ou de
prefixo na tabela real (isolamento fraco, risco de misturar com dados reais). **Dados fictícios**, ambiente **sempre
ligado**, **publicação automática** a cada push na `develop` pelo GitHub Actions (`deploy-dev.yml`, veja
[`deploy-github-actions.md`](deploy-github-actions.md)); os scripts `deploy.ps1` e `publicar-front.ps1` seguem como
alternativa manual.
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

## Comparar e medir o saldo do servidor (item 40, fase B)

Antes de a aba Saldos passar a usar `GET /saldos`, o saldo do servidor tem de **bater com o da tela antiga** (a que calcula no navegador) e a
**primeira abertura sem cache** tem de caber na Lambda e na leitura da tabela **como ela é hoje** (`specs/saldo-servidor/plan.md`, "A conferência lado a lado").
Primeiro no `dev`, depois na produção.

**Pré-requisito:** a fase A mergeada na `develop` e publicada no `dev` (o deploy é automático; confira o changeset e o status final da stack).

### Por que são dois testes, e não um
- **A tela antiga lê o histórico inteiro uma vez por banco.** Cada banco chama `GET /tx?banco=X`, e o servidor filtra o banco **depois** de ler a partição.
  Com anos de dados e sem cache, são dezenas de MB lidos em paralelo. A tabela é **provisionada com 5 RCU**, então a leitura estoura e a API responde
  `500` (`erro ao listar` ou `erro interno`). Isso acontece na tela antiga, com ou sem a fase A, e é o que aparece no console ao rodar `carregarSaldos`.
- **Por isso a comparação de igualdade usa poucos dados** (a carga padrão, de 2025 em diante, uns 2,6 mil lançamentos, que a tela antiga aguenta) e o
  **teste de volume usa dez anos, só no servidor** (`window.COMPARAR_SO_SERVIDOR = true`, que não roda a tela antiga).

### A capacidade da AWS não é parte da solução
O app tem de funcionar com a tabela **como ela está** (5 RCU), sem ninguém mexer na AWS: nem no dev, nem na produção, nem para quem usar o app amanhã. Subir a leitura só esconderia o
problema. A causa é do desenho: sem cache, o `GET /saldos` lê o histórico inteiro de uma vez. **A correção proposta é ler em partes** (um orçamento de leitura por chamada, gravar o
que já foi calculado e, se a tabela limitar a leitura, parar com calma em vez de dar `500`), descrita em `specs/saldo-servidor/` quando for aprovada. **Enquanto isso não existir, não
faça o teste de volume (etapa 2)**: com dez anos de dados ele estoura a leitura.

### Etapa 1: igualdade (poucos dados)
1. Dados: `python scripts/dados-ficticios.py <space-id> --confirmar` (a carga padrão). Se o dev já tem dez anos, limpe antes: `python scripts/limpar-espaco.py <space-id> --dev --confirmar`.
2. Abra o app do dev, entre como **dono** e espere a aba Saldos carregar.
3. F12 → Console, cole o conteúdo de `scripts/comparar-saldos-console.js` e aperte Enter. O token fica no navegador; o script só faz `GET`.
4. Ele calcula pela tela antiga, chama `GET /saldos` três vezes e mostra o **saldo e os meses de cada meio, antigo contra servidor**, e os **tempos**. Termina em `✓ IGUAL` ou `✗ DIFERENTE`.
   A primeira chamada é a sem cache; para repetir uma sem cache, mude e volte o saldo inicial de um banco na aba Saldos (isso muda a base do cache).
   Se a tela antiga falhar com `500`, o script avisa e manda usar o modo só servidor.

### Etapa 2: volume (dez anos, só o servidor) — só depois da leitura em partes
1. Dados: `python scripts/dados-ficticios.py <space-id> --desde 2016 --confirmar` (uns 14 mil lançamentos; a tabela tem 5 WCU, então leva mais de uma hora, ou bem menos com 20 WCU).
2. No console do app: `window.COMPARAR_SO_SERVIDOR = true` e Enter; depois cole o script. Ele chama `GET /saldos` três vezes e mostra o saldo de cada meio e os tempos.
3. Meça o que o navegador não vê (próxima seção).

**Critérios para ligar a tela nova**
- Etapa 1: `✓ IGUAL` em todos os meios e no total (a diferença aceita é menos de meio centavo).
- Etapa 2: primeira chamada dentro do tempo da Lambda (o limite é 30 s; mire em menos de 10 s), as seguintes em menos de 1 s, nenhum `500` e nenhum `ReadThrottleEvents`.

### Medir o que o navegador não vê (CloudWatch)
Precisa da sessão com MFA (`scripts/aws-mfa.ps1`); sem ela, a política `PermitirMFA` nega a leitura dos logs e das métricas.

Erros e duração da Lambda (a função tem **128 MB** e 30 s) na última hora. Procure `listar:` (a mensagem do erro, por exemplo `ProvisionedThroughputExceededException`) e, nas linhas `REPORT`, `Duration` e `Max Memory Used`:
```powershell
$ini = [DateTimeOffset]::UtcNow.AddMinutes(-60).ToUnixTimeMilliseconds()
aws logs filter-log-events --log-group-name /aws/lambda/financas-api-dev --region sa-east-1 --filter-pattern '"listar:"' --start-time $ini --query "events[].message" --output text
aws logs filter-log-events --log-group-name /aws/lambda/financas-api-dev --region sa-east-1 --filter-pattern '"REPORT"' --start-time $ini --query "events[].message" --output text
```
Leitura gasta e **throttling** da tabela:
```powershell
$ini = (Get-Date).AddMinutes(-60).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ"); $fim = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
foreach ($m in "ConsumedReadCapacityUnits","ReadThrottleEvents") {
  aws cloudwatch get-metric-statistics --namespace AWS/DynamoDB --metric-name $m --dimensions Name=TableName,Value=FinancasApp-dev --region sa-east-1 --start-time $ini --end-time $fim --period 60 --statistics Sum --query "Datapoints[].[Timestamp,Sum]" --output text
}
```
`ReadThrottleEvents` maior que zero explica o `500`. Registre os números no PR.

### Na produção (só leitura, pelo dono)
Só depois de a etapa 1 dar `✓ IGUAL` no dev e de a leitura em partes existir (acima). Repita a etapa 1 no app de produção, num horário calmo: **o servidor grava o cache de fechamento** (os itens `SALDO#<banco>`) por conta própria
na primeira chamada, e isso é esperado. Se algum meio divergir ou a primeira chamada falhar, **a tela nova não entra**: a tela antiga continua funcionando.
