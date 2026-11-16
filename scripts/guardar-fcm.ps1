# Guarda a credencial do Firebase (o JSON da conta de servico) no SSM, como SecureString, para o canal de push (spec app-android-push, R21).
#
# Uso, com a sessao do MFA ativa no terminal (. .\scripts\aws-mfa.ps1 <codigo>):
#   .\scripts\guardar-fcm.ps1 -Ambiente dev  -Arquivo C:\caminho\projeto-firebase-adminsdk-xxxxx.json
#   .\scripts\guardar-fcm.ps1 -Ambiente prod -Arquivo C:\caminho\projeto-firebase-adminsdk-xxxxx.json -ApagarOriginal
#
# - Um parametro POR AMBIENTE: /financas/dev/fcm-credencial e /financas/prod/fcm-credencial. Trocar a credencial e rodar este script de novo.
# - O conteudo do arquivo NUNCA e impresso, nem passado na linha de comando: o script so aponta o arquivo para o aws cli (file://).
# - Nunca cole o conteudo do arquivo no chat nem no git. Se ele vazar, gere outra chave no console do Firebase e revogue a antiga.
# - O arquivo baixado e uma chave privada: apague-o depois (-ApagarOriginal faz isso depois de gravar com sucesso).
param(
    [Parameter(Mandatory = $true)][ValidateSet('dev', 'prod')][string]$Ambiente,
    [Parameter(Mandatory = $true)][string]$Arquivo,
    [switch]$ApagarOriginal,
    [string]$Regiao = 'sa-east-1'
)
$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $Arquivo -PathType Leaf)) { Write-Error "Arquivo nao encontrado: $Arquivo"; return }
$caminho = (Resolve-Path -LiteralPath $Arquivo).Path

# confere o formato sem mostrar nada: precisa ser o JSON de uma conta de servico
try { $j = Get-Content -LiteralPath $caminho -Raw | ConvertFrom-Json } catch { Write-Error "O arquivo nao e um JSON valido."; return }
foreach ($campo in 'client_email', 'private_key', 'project_id') {
    if (-not $j.$campo) { Write-Error "O arquivo nao parece a chave de uma conta de servico (falta '$campo')."; return }
}
# limite do parametro padrao do SSM
if ((Get-Item -LiteralPath $caminho).Length -gt 4096) { Write-Error "O arquivo passa de 4096 bytes, o limite do parametro padrao do SSM."; return }
$j = $null

$nome = "/financas/$Ambiente/fcm-credencial"
aws ssm put-parameter --name $nome --type SecureString --overwrite --value "file://$caminho" --region $Regiao --output text --query Version | Out-Null
if ($LASTEXITCODE -ne 0) { Write-Error "Nao consegui gravar $nome (a sessao do MFA esta ativa?)."; return }
Write-Host "Credencial gravada em $nome (SecureString)."

if ($ApagarOriginal) {
    Remove-Item -LiteralPath $caminho -Force
    Write-Host "Arquivo original apagado."
} else {
    Write-Host "Lembrete: o arquivo baixado e uma chave privada. Apague-o: Remove-Item -LiteralPath '$caminho'"
}
