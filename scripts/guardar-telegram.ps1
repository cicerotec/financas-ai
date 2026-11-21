# Guarda o token do bot do Telegram no SSM, como SecureString (spec telegram-vinculo, R24). SO O DONO RODA este script; a IA nunca o roda.
#
# Uso, com a sessao do MFA ativa no terminal (. .\scripts\aws-mfa.ps1 <codigo>):
#   .\scripts\guardar-telegram.ps1 -Ambiente dev
#   .\scripts\guardar-telegram.ps1 -Ambiente prod
#
# - Um token POR AMBIENTE (/financas/dev/telegram-token e /financas/prod/telegram-token) e UM BOT POR AMBIENTE: o Telegram aceita um webhook so por bot,
#   entao o dev nunca pode usar o token da producao (registrar o webhook do dev tiraria o da producao do ar). O script recusa o mesmo token nos dois.
# - O token e lido com Read-Host -AsSecureString: nao aparece na tela, NAO e aceito na linha de comando e nunca e impresso.
# - Para o aws cli o valor vai por um arquivo temporario (file://), apagado no fim; ele nao passa pela linha de comando.
# - Se o token ja apareceu numa conversa ou num log, revogue-o no @BotFather (/revoke) e rode este script de novo.
param(
    [Parameter(Mandatory = $true)][ValidateSet('dev', 'prod')][string]$Ambiente,
    [string]$Regiao = 'sa-east-1'
)
$ErrorActionPreference = 'Stop'

$seguro = Read-Host -Prompt "Token do bot do ambiente $Ambiente (o texto nao aparece)" -AsSecureString
$bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($seguro)
try { $token = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr) } finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr) }
$token = $token.Trim()
if ($token -notmatch '^\d{5,}:[A-Za-z0-9_-]{30,}$') { Write-Error "Isso nao parece o token de um bot (numero:letras). Nada foi gravado."; return }

# um bot por ambiente: o token do outro ambiente nao pode ser este
$outro = if ($Ambiente -eq 'dev') { 'prod' } else { 'dev' }
$doOutro = aws ssm get-parameter --name "/financas/$outro/telegram-token" --with-decryption --query Parameter.Value --output text --region $Regiao 2>$null
if ($LASTEXITCODE -eq 0 -and $doOutro -and $doOutro.Trim() -eq $token) {
    $token = $null; $doOutro = $null
    Write-Error "Este e o mesmo token do ambiente $outro. Cada ambiente tem o seu bot (crie outro no @BotFather). Nada foi gravado."; return
}
$doOutro = $null

$nome = "/financas/$Ambiente/telegram-token"
$arquivo = [IO.Path]::GetTempFileName()
try {
    [IO.File]::WriteAllText($arquivo, $token, (New-Object Text.UTF8Encoding($false)))
    aws ssm put-parameter --name $nome --type SecureString --overwrite --value "file://$arquivo" --region $Regiao --output text --query Version | Out-Null
    if ($LASTEXITCODE -ne 0) { Write-Error "Nao consegui gravar $nome (a sessao do MFA esta ativa?)."; return }
} finally {
    [IO.File]::WriteAllText($arquivo, (' ' * 128))   # sobrescreve antes de apagar
    Remove-Item -LiteralPath $arquivo -Force -ErrorAction SilentlyContinue
    $token = $null
}
Write-Host "Token gravado em $nome (SecureString). Proximo passo, depois do deploy: .\scripts\registrar-webhook-telegram.ps1 -Ambiente $Ambiente"
