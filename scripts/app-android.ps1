# Monta o pacote do app Android (Capacitor) e gera o APK de DEBUG.
#   .\scripts\app-android.ps1 -Ambiente dev              # monta www/, roda cap sync e gera o APK de debug do dev
#   .\scripts\app-android.ps1 -Ambiente prod             # idem para o prod (br.com.cicerotec.financas)
#   .\scripts\app-android.ps1 -Ambiente dev -SoMontar    # so monta www/ e sincroniza, sem o Gradle (abra app-android/android no Android Studio)
# O web/ e a unica fonte das telas; app-android/www e gerado aqui e nunca editado nem versionado.
# O config do ambiente (web/config.dev.js no dev, web/config.js no prod) fica fora do git e precisa existir nesta maquina.
# Precisa de Node 22 ou mais (Capacitor 8), JDK 21 ou mais e do Android SDK com a plataforma 36 (ANDROID_HOME).
# Se o Node e o JDK padrao da maquina forem mais antigos, aponte os certos so para esta execucao, sem mexer no sistema:
#   .\scripts\app-android.ps1 -Ambiente dev -NodeDir D:\Tools\node-22 -JavaHome D:\Java\jdk-21
# ou deixe as variaveis APP_ANDROID_NODE e APP_ANDROID_JAVA_HOME definidas. O PATH e o JAVA_HOME voltam ao que eram no fim.
# Fase 1: so APK de debug; assinatura e AAB sao da fase 5 e o keystore nunca vai para o git.
param(
    [Parameter(Mandatory)][ValidateSet('prod','dev')][string]$Ambiente,
    [switch]$SoMontar,
    [string]$NodeDir = $env:APP_ANDROID_NODE,
    [string]$JavaHome = $env:APP_ANDROID_JAVA_HOME
)
$ErrorActionPreference = 'Stop'
$raiz = Split-Path $PSScriptRoot -Parent
$pathAntes = $env:Path
$javaAntes = $env:JAVA_HOME
Push-Location $raiz
try {
    if ($NodeDir) {
        if (-not (Test-Path (Join-Path $NodeDir 'node.exe'))) { throw "Nao achei node.exe em $NodeDir." }
        $env:Path = "$NodeDir;$env:Path"
    }
    if ($JavaHome) { $env:JAVA_HOME = $JavaHome }

    $node = (node --version) -replace '^v', ''
    if ([int]($node -split '\.')[0] -lt 22) { throw "O Capacitor 8 pede Node 22 ou mais e o Node em uso e o $node. Passe -NodeDir <pasta do Node 22> ou defina APP_ANDROID_NODE." }

    if (-not $SoMontar) {
        $release = if ($env:JAVA_HOME) { Join-Path $env:JAVA_HOME 'release' } else { '' }
        if (-not $release -or -not (Test-Path $release)) { throw "JAVA_HOME nao aponta para um JDK. Passe -JavaHome <pasta do JDK 21> ou defina APP_ANDROID_JAVA_HOME." }
        $linha = Select-String -Path $release -Pattern '^JAVA_VERSION="?(\d+)' | Select-Object -First 1
        $java = if ($linha) { [int]$linha.Matches[0].Groups[1].Value } else { 0 }
        if ($java -lt 21) { throw "O projeto Android pede JDK 21 ou mais e o JAVA_HOME ($env:JAVA_HOME) e o JDK $java. Passe -JavaHome <pasta do JDK 21> ou defina APP_ANDROID_JAVA_HOME." }
        if (-not $env:ANDROID_HOME -or -not (Test-Path (Join-Path $env:ANDROID_HOME 'platforms'))) { throw "ANDROID_HOME nao aponta para o Android SDK (instale pelo Android Studio)." }
    }

    node app-android/montar-www.js $Ambiente
    if ($LASTEXITCODE -ne 0) { throw "Nao montei o www/ do $Ambiente (veja a mensagem acima). Nada foi gerado." }

    Push-Location app-android
    try {
        if (-not (Test-Path node_modules)) { npm ci; if ($LASTEXITCODE -ne 0) { throw "npm ci falhou." } }
        npx cap sync android
        if ($LASTEXITCODE -ne 0) { throw "cap sync falhou." }
        if ($SoMontar) { Write-Host "Pronto. Abra app-android/android no Android Studio e escolha o sabor $Ambiente."; return }
        Push-Location android
        try {
            $tarefa = if ($Ambiente -eq 'dev') { 'assembleDevDebug' } else { 'assembleProdDebug' }
            .\gradlew.bat $tarefa
            if ($LASTEXITCODE -ne 0) { throw "Gradle falhou (veja a mensagem acima)." }
        } finally { Pop-Location }
    } finally { Pop-Location }
    $apk = Join-Path $raiz "app-android/android/app/build/outputs/apk/$Ambiente/debug/app-$Ambiente-debug.apk"
    if (-not (Test-Path $apk)) { throw "O Gradle terminou, mas o APK nao esta em $apk." }
    Write-Host "APK de debug: $apk"
    Write-Host "Instale com: adb install -r `"$apk`""
} finally {
    Pop-Location
    $env:Path = $pathAntes
    $env:JAVA_HOME = $javaAntes
}
