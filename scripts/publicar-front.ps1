# Envia a pasta web/ para o bucket do front (S3 + CloudFront) e limpa o cache da distribuicao.
#   .\scripts\publicar-front.ps1            # publica e invalida o cache
#   .\scripts\publicar-front.ps1 -Simular   # mostra o que mudaria, sem enviar nada
#   .\scripts\publicar-front.ps1 -Forcar    # nao pergunta se estiver fora da main / com alteracoes sem commit
#   .\scripts\publicar-front.ps1 -Ambiente dev   # publica na stack financas-dev, com web/config.dev.js
# Antes de publicar, avisa se voce nao esta na main, tem alteracoes sem commit ou esta atras do GitHub.
# Precisa da sessao AWS aberta (MFA): . .\scripts\aws-mfa.ps1 <codigo>
# O web/config.js (web/config.dev.js no dev) fica fora do git (tem a URL da API e os dados do login); precisa existir aqui na maquina.
param(
    [string]$Stack,
    [switch]$Simular,
    [switch]$Forcar,
    [ValidateSet('prod','dev')][string]$Ambiente = 'prod'
)
$ErrorActionPreference = 'Stop'
if (-not $Stack) { $Stack = if ($Ambiente -eq 'dev') { 'financas-dev' } else { 'financas' } }
Set-Location (Split-Path $PSScriptRoot -Parent)

. "$PSScriptRoot\_guarda.ps1"
if (-not $Simular) {   # a simulacao nao publica nada, entao nao precisa perguntar
    if (-not (Confirmar-Publicacao -Forcar:$Forcar -Ambiente $Ambiente)) { throw "Publicacao cancelada." }
}

$arquivoConfig = if ($Ambiente -eq 'dev') { 'web/config.dev.js' } else { 'web/config.js' }
if (-not (Test-Path $arquivoConfig)) {
    throw "$arquivoConfig nao existe. Copie web/config.example.js para $arquivoConfig e preencha com os Outputs do deploy."
}

$saidas = aws cloudformation describe-stacks --stack-name $Stack --query "Stacks[0].Outputs" --output json | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or -not $saidas) { throw "Nao consegui ler os Outputs da stack '$Stack'. A sessao AWS esta aberta?" }
$bucket = ($saidas | Where-Object OutputKey -eq 'FrontBucket').OutputValue
$dist   = ($saidas | Where-Object OutputKey -eq 'DistributionId').OutputValue
$url    = ($saidas | Where-Object OutputKey -eq 'FrontUrl').OutputValue
if (-not $bucket -or -not $dist) { throw "A stack ainda nao tem FrontBucket/DistributionId. Rode primeiro: .\scripts\deploy.ps1 <codigo>" }

# --delete remove do bucket o que nao existe mais em web/. O config.example.js nao precisa ir.
# No dev, o config.dev.js sobe como config.js (o front sempre le config.js) e o config.js de producao nao vai.
$parametrosS3 = @("s3", "sync", "web", "s3://$bucket", "--delete", "--exclude", "config.example.js", "--exclude", "config.dev.js", "--cache-control", "no-cache")
if ($Ambiente -eq 'dev') { $parametrosS3 += @("--exclude", "config.js") }
if ($Simular) { $parametrosS3 += "--dryrun" }
aws @parametrosS3
if ($LASTEXITCODE -ne 0) { throw "aws s3 sync falhou." }

if ($Ambiente -eq 'dev') {
    $copiar = @("s3", "cp", "web/config.dev.js", "s3://$bucket/config.js", "--cache-control", "no-cache")
    if ($Simular) { $copiar += "--dryrun" }
    aws @copiar
    if ($LASTEXITCODE -ne 0) { throw "aws s3 cp do config.dev.js falhou." }
}

if ($Simular) { Write-Host "Simulacao: nada foi enviado." -ForegroundColor Yellow; return }

aws cloudfront create-invalidation --distribution-id $dist --paths "/*" | Out-Null
if ($LASTEXITCODE -ne 0) { throw "A invalidacao do cache falhou (os arquivos ja foram enviados)." }
Write-Host "Publicado: $url" -ForegroundColor Green
