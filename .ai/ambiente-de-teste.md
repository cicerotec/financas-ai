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
**primeira abertura sem cache** tem de caber na Lambda (`specs/saldo-servidor/plan.md`, "A conferência lado a lado"). Primeiro no `dev`, depois na produção.

**Pré-requisito:** a fase A (PR do item 40) mergeada na `develop` e publicada no `dev` (o deploy é automático; confira o changeset e o status final da stack, como no guardrails).

### 1. Dados com volume (só no dev)
O dono tem dados desde 2016. Carregue dez anos de dados fictícios (uns 14 mil lançamentos, com `CFG#SALDOS` de cada banco):
```powershell
python scripts/dados-ficticios.py <space-id> --desde 2016              # simulação: só conta
python scripts/dados-ficticios.py <space-id> --desde 2016 --confirmar  # grava (a tabela tem 5 WCU: leva mais de uma hora)
```
Para acelerar, suba a escrita do dev por um tempo (o always-free são 25 RCU/WCU no total e a produção usa 5/5, então o dev pode ir a 20) e **volte depois**:
```powershell
aws dynamodb update-table --table-name FinancasApp-dev --region sa-east-1 --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=20
```
(O próximo deploy da stack volta a tabela para 5/5, porque o template manda.)

### 2. Comparar e medir pelo navegador
1. Abra o app do dev, entre como **dono** e espere a aba Saldos carregar.
2. F12 → Console, cole o conteúdo de `scripts/comparar-saldos-console.js` e aperte Enter. O token fica no navegador; o script só faz `GET`.
3. Ele calcula pela tela antiga, chama `GET /saldos` três vezes e mostra duas tabelas: **saldo e meses de cada meio, antigo contra servidor**, e os **tempos** de cada chamada. Termina com `✓ IGUAL` ou `✗ DIFERENTE`.
4. A **primeira chamada** é a sem cache (o servidor ainda não gravou nada com a base `v2`). Para repetir uma chamada sem cache, mude e volte o saldo inicial de um banco na própria aba Saldos (muda a base do cache). As outras duas já usam o cache.

**Critérios para ligar a tela nova**
- `✓ IGUAL` em todos os meios e no total (a diferença aceita é menos de meio centavo).
- Primeira chamada dentro do tempo da Lambda (o limite é 30 s; mire em menos de 10 s) e as seguintes em menos de 1 s.
- Nenhum erro `500` nem `ReadThrottleEvents` (veja o item 3).

### 3. Medir o que o navegador não vê (CloudWatch)
Duração e memória da Lambda (a função tem **128 MB** e 30 s) nos últimos 15 minutos; procure `Duration` e `Max Memory Used` nas linhas `REPORT`:
```powershell
$ini = [DateTimeOffset]::UtcNow.AddMinutes(-15).ToUnixTimeMilliseconds()
aws logs filter-log-events --log-group-name /aws/lambda/financas-api-dev --region sa-east-1 --filter-pattern '"REPORT"' --start-time $ini --query "events[].message" --output text
```
Leitura gasta e **throttling** da tabela (a tabela é provisionada com **5 RCU**: a primeira chamada sem cache lê o histórico inteiro e pode estourar a capacidade):
```powershell
$ini = (Get-Date).AddMinutes(-30).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ"); $fim = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
foreach ($m in "ConsumedReadCapacityUnits","ReadThrottleEvents") {
  aws cloudwatch get-metric-statistics --namespace AWS/DynamoDB --metric-name $m --dimensions Name=TableName,Value=FinancasApp-dev --region sa-east-1 --start-time $ini --end-time $fim --period 60 --statistics Sum --query "Datapoints[].[Timestamp,Sum]" --output text
}
```
Se `ReadThrottleEvents` aparecer, o `GET /saldos` sem cache pode falhar com `500`. Os caminhos são: subir a leitura do dev (e, depois, decidir sobre a produção) ou fazer o servidor ler o histórico em partes. **É decisão do dono**; registre os números no PR.

### 4. Na produção (só leitura, pelo dono)
Repita o passo 2 no app de produção, num horário calmo. O `GET /saldos` **grava o cache de fechamento** (os itens `SALDO#<banco>`) por conta própria na primeira chamada, e isso é esperado. Se algum meio divergir ou a primeira chamada falhar, **a tela nova não entra**: a tela antiga continua funcionando.
