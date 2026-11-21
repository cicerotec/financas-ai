# Registra o webhook do bot do Telegram do ambiente e grava o segredo e o usuario do bot no SSM (spec telegram-vinculo, R24).
# SO O DONO RODA este script; a IA nunca o roda. Rode DEPOIS do deploy do ambiente e DEPOIS de guardar-telegram.ps1.
#
# Uso, com a sessao do MFA ativa no terminal (. .\scripts\aws-mfa.ps1 <codigo>):
#   .\scripts\registrar-webhook-telegram.ps1 -Ambiente dev
#   .\scripts\registrar-webhook-telegram.ps1 -Ambiente prod
#   .\scripts\registrar-webhook-telegram.ps1 -Ambiente dev -Forcar   # so se voce sabe que o bot pode trocar de webhook
#
# O que faz, nesta ordem:
#   1. le o token do ambiente no SSM e o usuario do bot (getMe);
#   2. confere o getWebhookInfo e RECUSA se o bot ja tem um webhook de OUTRA URL (sem -Forcar): um bot aceita um webhook so, e trocar o de producao por engano
#      tiraria os /start dela do ar;
#   3. gera o segredo com o gerador criptografico (A-Z a-z 0-9 _ -, 43 caracteres);
#   4. chama setWebhook (URL da API do ambiente, o segredo e allowed_updates ["message"]);
#   5. SO DEPOIS de o setWebhook responder ok, grava o segredo e, por ULTIMO, o usuario do bot. O usuario por ultimo impede que o app gere um link para um webhook
#      que nao existe. Se uma gravacao falhar, o script diz para rodar de novo. Entre o setWebhook e a gravacao o Telegram pode receber 401 por alguns segundos e reenvia.
# Rodar de novo gera OUTRO segredo (o servidor aceita o novo em ate 5 minutos). Token, segredo e codigos nunca sao impressos nem vao a linha de comando.
param(
    [Parameter(Mandatory = $true)][ValidateSet('dev', 'prod')][string]$Ambiente,
    [switch]$Forcar,
    [string]$Regiao = 'sa-east-1'
)
$ErrorActionPreference = 'Stop'

function Gravar-Parametro($nome, $tipo, $valor) {
    $arq = [IO.Path]::GetTempFileName()
    try {
        [IO.File]::WriteAllText($arq, $valor, (New-Object Text.UTF8Encoding($false)))
        aws ssm put-parameter --name $nome --type $tipo --overwrite --value "file://$arq" --region $Regiao --output text --query Version | Out-Null
        return ($LASTEXITCODE -eq 0)
    } finally {
        [IO.File]::WriteAllText($arq, (' ' * 128))
        Remove-Item -LiteralPath $arq -Force -ErrorAction SilentlyContinue
    }
}

# chama a API do Telegram sem deixar o token aparecer: o erro mostra so o codigo HTTP
function Chamar-Telegram($token, $metodo, $corpo) {
    $uri = "https://api.telegram.org/bot$token/$metodo"
    try {
        if ($corpo) { return Invoke-RestMethod -Uri $uri -Method Post -Body ($corpo | ConvertTo-Json -Depth 5 -Compress) -ContentType 'application/json' }
        return Invoke-RestMethod -Uri $uri -Method Get
    } catch {
        $codigo = if ($_.Exception.Response) { [int]$_.Exception.Response.StatusCode } else { 0 }
        throw "O Telegram recusou $metodo (HTTP $codigo). O token do ambiente $Ambiente esta certo e o bot ainda existe?"
    }
}

$stack = if ($Ambiente -eq 'dev') { 'financas-dev' } else { 'financas' }

$token = aws ssm get-parameter --name "/financas/$Ambiente/telegram-token" --with-decryption --query Parameter.Value --output text --region $Regiao
if ($LASTEXITCODE -ne 0 -or -not $token) { Write-Error "Nao achei /financas/$Ambiente/telegram-token. Rode antes: .\scripts\guardar-telegram.ps1 -Ambiente $Ambiente"; return }
$token = $token.Trim()

$api = aws cloudformation describe-stacks --stack-name $stack --query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" --output text --region $Regiao
if ($LASTEXITCODE -ne 0 -or -not $api -or $api -eq 'None') { Write-Error "Nao consegui ler o ApiUrl da stack $stack. O deploy do ambiente ja foi feito?"; return }
$urlDoWebhook = $api.Trim().TrimEnd('/') + '/telegram/webhook'

$eu = Chamar-Telegram $token 'getMe' $null
$usuario = $eu.result.username
if (-not $usuario -or $usuario -notmatch '^[A-Za-z0-9_]{1,64}$') { Write-Error "O getMe nao devolveu um usuario de bot valido."; return }

$atual = (Chamar-Telegram $token 'getWebhookInfo' $null).result.url
if ($atual -and $atual -ne $urlDoWebhook -and -not $Forcar) {
    Write-Error "Este bot ja tem um webhook de OUTRA URL. Um bot aceita um webhook so: registrar este tiraria o outro do ar. Se o bot e mesmo do ambiente $Ambiente, rode de novo com -Forcar. Nada foi alterado."
    return
}

# segredo: 32 bytes do gerador criptografico em base64 de URL (so A-Z a-z 0-9 _ -, sem preenchimento): 43 caracteres, dentro dos 1 a 256 do Telegram
$bytes = New-Object byte[] 32
$rng = [Security.Cryptography.RandomNumberGenerator]::Create()
try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
$segredo = [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_')

$r = Chamar-Telegram $token 'setWebhook' @{ url = $urlDoWebhook; secret_token = $segredo; allowed_updates = @('message') }
if (-not $r.ok) { Write-Error "O setWebhook nao respondeu ok. Nada foi gravado no SSM."; return }

if (-not (Gravar-Parametro "/financas/$Ambiente/telegram-webhook-secret" 'SecureString' $segredo)) {
    Write-Error "O webhook foi registrado, mas nao consegui gravar o segredo no SSM. Rode este script de novo."; return
}
$segredo = $null; $token = $null
if (-not (Gravar-Parametro "/financas/$Ambiente/telegram-bot-usuario" 'String' $usuario)) {
    Write-Error "O webhook e o segredo estao gravados, mas nao consegui gravar o usuario do bot. Rode este script de novo."; return
}
Write-Host "Webhook do ambiente $Ambiente registrado; segredo e usuario do bot gravados no SSM. O app ja pode gerar o link (em ate 5 minutos o servidor le o segredo novo)."
