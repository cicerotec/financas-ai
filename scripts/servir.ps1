# Sobe o front localmente (http://localhost:8080). Por padrao usa o ambiente de TESTE (dev): nunca toca nos dados reais.
#   .\scripts\servir.ps1                # dev: web/config.dev.js vira o config.js de uma copia temporaria (faixa laranja)
#   .\scripts\servir.ps1 -Porta 8081    # outra porta (a origem precisa estar no FrontOrigin da stack)
#   .\scripts\servir.ps1 -Producao      # PRODUCAO: serve web/ como esta, com o web/config.js real (pede confirmacao)
# A pasta web/ nao e alterada: no dev o servidor usa uma copia temporaria, apagada ao encerrar (Ctrl+C).
param(
    [int]$Porta = 8080,
    [switch]$Producao
)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)

if ($Producao) {
    if (-not (Test-Path 'web/config.js')) { throw "web/config.js nao existe. Copie web/config.example.js e preencha." }
    Write-Host "ATENCAO: este servidor local vai usar a API e o login de PRODUCAO (dados reais)." -ForegroundColor Red
    if ((Read-Host "Digite 'producao' para continuar") -ne 'producao') { throw "Cancelado." }
    python -m http.server $Porta --directory web
    return
}

if (-not (Test-Path 'web/config.dev.js')) {
    throw "web/config.dev.js nao existe. Copie web/config.example.js para web/config.dev.js e preencha com os Outputs da stack financas-dev (com ambiente: ""dev"")."
}

$temp = Join-Path ([System.IO.Path]::GetTempPath()) "financas-web-dev-$PID"
try {
    New-Item -ItemType Directory -Path $temp | Out-Null
    Copy-Item -Path 'web/*' -Destination $temp -Recurse
    Copy-Item -Path 'web/config.dev.js' -Destination (Join-Path $temp 'config.js') -Force
    Write-Host "Ambiente de TESTE (dev) em http://localhost:$Porta  (Ctrl+C para encerrar)" -ForegroundColor Yellow
    python -m http.server $Porta --directory $temp
} finally {
    if (Test-Path $temp) { Remove-Item -Recurse -Force $temp }
}
