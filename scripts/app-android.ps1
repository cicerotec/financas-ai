# Monta o pacote do app Android (Capacitor) e gera o APK de DEBUG.
#   .\scripts\app-android.ps1 -Ambiente dev              # monta www/, roda cap sync e gera o APK de debug do dev
#   .\scripts\app-android.ps1 -Ambiente prod             # idem para o prod (br.com.cicerotec.financas)
#   .\scripts\app-android.ps1 -Ambiente dev -SoMontar    # so monta www/ e sincroniza, sem o Gradle (abra app-android/android no Android Studio)
# O web/ e a unica fonte das telas; app-android/www e gerado aqui e nunca editado nem versionado.
# O config do ambiente (web/config.dev.js no dev, web/config.js no prod) fica fora do git e precisa existir nesta maquina.
# Precisa de Node e do Android Studio (SDK e JDK). Fase 1: so APK de debug; assinatura e AAB sao da fase 5 e o keystore nunca vai para o git.
param(
    [Parameter(Mandatory)][ValidateSet('prod','dev')][string]$Ambiente,
    [switch]$SoMontar
)
$ErrorActionPreference = 'Stop'
$raiz = Split-Path $PSScriptRoot -Parent
Set-Location $raiz

node app-android/montar-www.js $Ambiente
if ($LASTEXITCODE -ne 0) { throw "Nao montei o www/ do $Ambiente (veja a mensagem acima). Nada foi gerado." }

Push-Location app-android
try {
    if (-not (Test-Path node_modules)) { npm install; if ($LASTEXITCODE -ne 0) { throw "npm install falhou." } }
    npx cap sync android
    if ($LASTEXITCODE -ne 0) { throw "cap sync falhou." }
    if ($SoMontar) { Write-Host "Pronto. Abra app-android/android no Android Studio e escolha o sabor $Ambiente."; return }
    Push-Location android
    try {
        $tarefa = if ($Ambiente -eq 'dev') { 'assembleDevDebug' } else { 'assembleProdDebug' }
        .\gradlew.bat $tarefa
        if ($LASTEXITCODE -ne 0) { throw "Gradle falhou. O Android SDK esta instalado (Android Studio)?" }
    } finally { Pop-Location }
} finally { Pop-Location }
$apk = Get-ChildItem "app-android/android/app/build/outputs/apk" -Recurse -Filter "*-debug.apk" | Where-Object FullName -match $Ambiente | Select-Object -First 1
Write-Host "APK de debug: $($apk.FullName)"
Write-Host "Instale com: adb install -r `"$($apk.FullName)`""
