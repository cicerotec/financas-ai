# Abre uma sessao AWS com MFA neste terminal (credenciais temporarias de 12 h).
# Use com "ponto + espaco" na frente, para as variaveis ficarem no seu terminal:
#   . .\scripts\aws-mfa.ps1 123456
# O numero de serie do MFA e descoberto sozinho; se preferir, defina antes
#   $env:AWS_MFA_SERIAL = "arn:aws:iam::<conta>:mfa/<nome>"
param([Parameter(Mandatory = $true)][string]$Codigo, [string]$Regiao = "sa-east-1")

# remove sessao anterior: o STS nao aceita get-session-token com credenciais de sessao
Remove-Item Env:AWS_ACCESS_KEY_ID, Env:AWS_SECRET_ACCESS_KEY, Env:AWS_SESSION_TOKEN -ErrorAction SilentlyContinue

$serial = $env:AWS_MFA_SERIAL
if (-not $serial) {
    $serial = aws iam list-mfa-devices --query "MFADevices[0].SerialNumber" --output text
    if (-not $serial -or $serial -eq "None") { Write-Error "Nao achei o MFA. Defina `$env:AWS_MFA_SERIAL."; return }
}

$r = aws sts get-session-token --serial-number $serial --token-code $Codigo --duration-seconds 43200 | ConvertFrom-Json
if (-not $r) { Write-Error "Falhou: confira o codigo do app autenticador."; return }

$env:AWS_ACCESS_KEY_ID     = $r.Credentials.AccessKeyId
$env:AWS_SECRET_ACCESS_KEY = $r.Credentials.SecretAccessKey
$env:AWS_SESSION_TOKEN     = $r.Credentials.SessionToken
$env:AWS_DEFAULT_REGION    = $Regiao
Write-Host "Sessao ativa ate $($r.Credentials.Expiration)"
