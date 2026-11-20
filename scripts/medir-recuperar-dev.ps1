# Mede o tempo da rota publica POST /mfa/recuperar no DEV (spec mfa-opcional, R28 e Verificacao passo 12).
# Compara e-mail SEM conta com e-mail COM conta, os dois com codigo no formato certo e errado (nenhum codigo vale; nada e desligado).
# Seguro: le o endereco so de web/config.dev.js (nunca producao), imprime so tempos em milissegundos (nunca e-mail, codigo ou corpo)
# e faz 5 chamadas erradas para a conta (o limite e 10 falhas por hora do par pessoa+origem; sobra folga).
# Uso:  .\scripts\medir-recuperar-dev.ps1 -EmailComConta "<e-mail do usuario de teste do dev>"
param([Parameter(Mandatory = $true)][string]$EmailComConta, [int]$Rodadas = 5)
$ErrorActionPreference = 'Stop'
$cfg = Join-Path $PSScriptRoot '..\web\config.dev.js'
if (-not (Test-Path $cfg)) { throw 'web/config.dev.js nao encontrado.' }
$m = [regex]::Match((Get-Content $cfg -Raw), 'apiUrl:\s*"([^"]+)"')
if (-not $m.Success) { throw 'apiUrl nao encontrado em config.dev.js.' }
$url = $m.Groups[1].Value.TrimEnd('/') + '/mfa/recuperar'
if ($Rodadas -gt 5) { throw 'No maximo 5 rodadas (o limite por par e 10 falhas por hora).' }
$semConta = 'ninguem-' + [guid]::NewGuid().ToString('N').Substring(0, 8) + '@exemplo.invalid'
$codigo = 'ABCD-EFGH-JKLM-NPQR-STUV'
function Medir($email) {
  $corpo = @{ email = $email; codigo = $codigo } | ConvertTo-Json -Compress
  $t = [Diagnostics.Stopwatch]::StartNew()
  try { Invoke-WebRequest -Uri $url -Method Post -ContentType 'application/json' -Body $corpo -UseBasicParsing | Out-Null; $st = 200 }
  catch { $st = [int]$_.Exception.Response.StatusCode }
  $t.Stop()
  [pscustomobject]@{ ms = [math]::Round($t.Elapsed.TotalMilliseconds); status = $st }
}
Medir $semConta | Out-Null # aquecimento da Lambda (nao conta)
$a = @(); $b = @()
for ($i = 0; $i -lt $Rodadas; $i++) { $a += Medir $semConta; $b += Medir $EmailComConta }
function Resumo($nome, $l) {
  $ms = ($l | Sort-Object ms).ms
  '{0,-10} status={1}  min={2}  mediana={3}  max={4} (ms, {5} chamadas)' -f $nome, (($l.status | Sort-Object -Unique) -join ','), $ms[0], $ms[[int][math]::Floor($ms.Count / 2)], $ms[-1], $ms.Count
}
Resumo 'sem conta' $a
Resumo 'com conta' $b
