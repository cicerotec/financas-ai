# Sessao AWS com MFA + sam build + sam deploy, num comando so.
#   .\scripts\deploy.ps1 324941            # pede o codigo do MFA, builda e publica (confirma o changeset)
#   .\scripts\deploy.ps1 324941 -Sim       # publica sem perguntar o changeset
#   .\scripts\deploy.ps1                   # reaproveita a sessao MFA ja aberta neste terminal
param(
    [string]$Codigo,
    [switch]$Sim
)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)   # raiz do projeto (onde esta o template.yaml)

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

if ($Sim) { sam deploy --no-confirm-changeset --no-fail-on-empty-changeset }
else      { sam deploy --no-fail-on-empty-changeset }
if ($LASTEXITCODE -ne 0) { throw "sam deploy falhou." }
Write-Host "Publicado." -ForegroundColor Green
