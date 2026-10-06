# Sessao AWS com MFA + sam build + sam deploy, num comando so.
#   .\scripts\deploy.ps1 324941            # pede o codigo do MFA, builda e publica (confirma o changeset)
#   .\scripts\deploy.ps1 324941 -Sim       # publica sem perguntar o changeset
#   .\scripts\deploy.ps1                   # reaproveita a sessao MFA ja aberta neste terminal
#   .\scripts\deploy.ps1 324941 -Forcar    # nao pergunta se estiver fora da main / com alteracoes sem commit
#   .\scripts\deploy.ps1 324941 -Ambiente dev   # publica a stack de teste (financas-dev); o esperado e estar na develop
# Antes de publicar, avisa se voce nao esta na main, tem alteracoes sem commit ou esta atras do GitHub.
param(
    [string]$Codigo,
    [switch]$Sim,
    [switch]$Forcar,
    [ValidateSet('prod','dev')][string]$Ambiente = 'prod'
)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)   # raiz do projeto (onde esta o template.yaml)

. "$PSScriptRoot\_guarda.ps1"
if (-not (Confirmar-Publicacao -Forcar:$Forcar -Ambiente $Ambiente)) { throw "Publicacao cancelada." }

if ($Codigo) {
    # as variaveis AWS_* ficam no processo, entao valem para os comandos abaixo
    & "$PSScriptRoot\aws-mfa.ps1" $Codigo
} else {
    aws sts get-caller-identity --query Arn --output text | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Sem sessao AWS neste terminal. Rode: .\scripts\deploy.ps1 <codigo-do-MFA>" }
}
if (-not $env:AWS_SESSION_TOKEN -and $Codigo) { throw "A sessao MFA nao foi aberta (confira o codigo)." }

sam build
if ($LASTEXITCODE -ne 0) { throw "sam build falhou." }

$perfil = if ($Ambiente -eq 'dev') { 'dev' } else { 'default' }
if ($Sim) { sam deploy --config-env $perfil --no-confirm-changeset --no-fail-on-empty-changeset }
else      { sam deploy --config-env $perfil --no-fail-on-empty-changeset }
if ($LASTEXITCODE -ne 0) { throw "sam deploy falhou." }
Write-Host "Publicado." -ForegroundColor Green
