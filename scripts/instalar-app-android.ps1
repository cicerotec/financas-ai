# Gera o APK de debug (dev e/ou prod) e instala no aparelho conectado por USB.
#   .\scripts\instalar-app-android.ps1 -Ambiente prod            # compila o prod e instala
#   .\scripts\instalar-app-android.ps1 -Ambiente dev             # compila o dev e instala
#   .\scripts\instalar-app-android.ps1 -Ambiente ambos           # dev e prod, um depois do outro (coexistem: applicationId diferente)
#   .\scripts\instalar-app-android.ps1 -Ambiente prod -SemCompilar   # so instala o APK que ja foi gerado
#   .\scripts\instalar-app-android.ps1 -Ambiente dev -Serial XOCYTOEAIJSCMZX4   # com mais de um aparelho conectado
# Quem compila e scripts\app-android.ps1 (Node 22 e JDK 21: -NodeDir e -JavaHome, ou APP_ANDROID_NODE e APP_ANDROID_JAVA_HOME).
# A instalacao e "adb install -r": mantem os dados do app que ja esta no aparelho. O aparelho precisa estar com a depuracao USB ligada e autorizada.
param(
    [Parameter(Mandatory)][ValidateSet('dev','prod','ambos')][string]$Ambiente,
    [switch]$SemCompilar,
    [string]$Serial,
    [string]$NodeDir = $env:APP_ANDROID_NODE,
    [string]$JavaHome = $env:APP_ANDROID_JAVA_HOME
)
$ErrorActionPreference = 'Stop'
$raiz = Split-Path $PSScriptRoot -Parent

# 1. o adb: no PATH ou em ANDROID_HOME\platform-tools
$adb = (Get-Command adb -ErrorAction SilentlyContinue).Source
if (-not $adb -and $env:ANDROID_HOME) {
    $candidato = Join-Path $env:ANDROID_HOME 'platform-tools\adb.exe'
    if (Test-Path $candidato) { $adb = $candidato }
}
if (-not $adb) { throw "Nao achei o adb. Instale o Android SDK (platform-tools) e defina ANDROID_HOME ou ponha o adb no PATH." }

# 2. o aparelho: um so, em estado "device" (os "unauthorized" e "offline" nao servem)
$linhas = & $adb devices | Select-Object -Skip 1 | Where-Object { $_.Trim() }
$prontos = @($linhas | Where-Object { $_ -match '\sdevice$' } | ForEach-Object { ($_ -split '\s+')[0] })
$outros = @($linhas | Where-Object { $_ -notmatch '\sdevice$' })
if ($Serial) {
    if ($prontos -notcontains $Serial) { throw "O aparelho $Serial nao esta pronto. Conectados e prontos: $($prontos -join ', ')." }
} elseif ($prontos.Count -eq 1) {
    $Serial = $prontos[0]
} elseif ($prontos.Count -eq 0) {
    $msg = "Nenhum aparelho pronto. Conecte o cabo, ligue a depuracao USB e autorize este computador na tela do celular."
    if ($outros.Count) { $msg += " O adb viu: $($outros -join '; ')." }
    throw $msg
} else {
    throw "Ha mais de um aparelho conectado ($($prontos -join ', ')). Escolha um com -Serial."
}
Write-Host "Aparelho: $Serial"

# 3. para cada ambiente: compila (se nao for -SemCompilar) e instala
$alvos = if ($Ambiente -eq 'ambos') { @('dev', 'prod') } else { @($Ambiente) }
foreach ($a in $alvos) {
    if (-not $SemCompilar) {
        $parametros = @{ Ambiente = $a }
        if ($NodeDir) { $parametros.NodeDir = $NodeDir }
        if ($JavaHome) { $parametros.JavaHome = $JavaHome }
        & (Join-Path $PSScriptRoot 'app-android.ps1') @parametros
    }
    $apk = Join-Path $raiz "app-android\android\app\build\outputs\apk\$a\debug\app-$a-debug.apk"
    if (-not (Test-Path $apk)) { throw "Nao achei o APK do $a em $apk. Rode sem -SemCompilar para gerar." }
    Write-Host "Instalando o $a ($([math]::Round((Get-Item $apk).Length / 1MB, 1)) MB)..."
    $antes = $ErrorActionPreference; $ErrorActionPreference = 'Continue' # o adb escreve a falha em stderr; aqui ela e lida, nao e erro do PowerShell
    $saida = (& $adb -s $Serial install -r $apk 2>&1 | ForEach-Object { "$_" }) -join ' '
    $ErrorActionPreference = $antes
    if ($saida -notmatch 'Success') {
        $dica = ''
        if ($saida -match 'USER_RESTRICTED') { $dica = " No Xiaomi: ligue 'Instalar via USB' (e 'Depuracao USB (configuracoes de seguranca)') em Opcoes do desenvolvedor e aceite a janela que aparece no celular durante a instalacao."
        } elseif ($saida -match 'INSTALL_FAILED_UPDATE_INCOMPATIBLE|signatures do not match') { $dica = " O app instalado foi assinado com outra chave: desinstale-o no celular e instale de novo (os dados do app local se perdem)." }
        throw "A instalacao do $a falhou: $($saida.Trim()).$dica"
    }
    Write-Host "OK: $a instalado."
}
