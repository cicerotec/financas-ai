# Apoio dos scripts de publicacao (deploy.ps1 e publicar-front.ps1). Nao e para rodar sozinho.
# BLOQUEIA (sem opcao de seguir, nem com -Forcar) a publicacao fora da branch do ambiente: main para prod, develop para dev.
# Avisa, e pergunta, quando ha alteracoes sem commit nos arquivos versionados ou a branch esta atrasada em relacao ao GitHub.
# O deploy publica o que esta na sua pasta agora, nao o que esta no GitHub.

function Obter-BranchEsperada {
    param([string]$Ambiente = 'prod')
    if ($Ambiente -eq 'dev') { return 'develop' } else { return 'main' }
}

function Obter-AvisosDePublicacao {
    $ErrorActionPreference = 'Continue'   # git escreve em stderr quando nao ha upstream; nao e erro
    $avisos = @()

    $branch = git rev-parse --abbrev-ref HEAD
    if ($LASTEXITCODE -ne 0 -or -not $branch) { return @("Nao consegui ler a branch atual do Git.") }
    $branch = $branch.Trim()
    # so arquivos versionados (os ignorados, como web/config.js, e os novos nao contam)
    $sujos = @(git status --porcelain --untracked-files=no)
    if ($sujos.Count -gt 0) { $avisos += "Ha $($sujos.Count) arquivo(s) versionado(s) com alteracoes sem commit (eles seriam publicados)." }

    git fetch origin --quiet 2>$null
    $atras = git rev-list --count "HEAD..@{u}" 2>$null   # sem upstream nao ha o que comparar
    if ($LASTEXITCODE -eq 0 -and $atras -and [int]$atras -gt 0) {
        $avisos += "Sua branch esta $atras commit(s) atras do GitHub (rode git pull)."
    }
    return $avisos
}

function Confirmar-Publicacao {
    param([switch]$Forcar, [string]$Ambiente = 'prod')
    $ErrorActionPreference = 'Continue'
    $branch = (git rev-parse --abbrev-ref HEAD)
    $versao = (git describe --tags --always)
    Write-Host "Publicando o ambiente $Ambiente a partir de: $branch ($versao)" -ForegroundColor Cyan

    $esperada = Obter-BranchEsperada -Ambiente $Ambiente
    if ($branch.Trim() -ne $esperada) {
        Write-Host "BLOQUEADO: o ambiente $Ambiente so publica a partir da branch '$esperada' e voce esta em '$($branch.Trim())'. Nao ha como pular." -ForegroundColor Red
        return $false
    }

    $avisos = @(Obter-AvisosDePublicacao)
    if ($avisos.Count -eq 0) { return $true }

    foreach ($a in $avisos) { Write-Host "AVISO: $a" -ForegroundColor Yellow }
    if ($Forcar) { Write-Host "-Forcar: seguindo mesmo assim." -ForegroundColor Yellow; return $true }

    $resposta = Read-Host "Publicar mesmo assim? (s/N)"
    return ($resposta -match '^(s|sim|y|yes)$')
}
