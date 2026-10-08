#!/usr/bin/env bash
# PR de release (develop -> main) e o comentario com a versao calculada. Roda no GitHub Actions (precisa de GH_TOKEN e
# do repositorio com historico completo e tags).
#   scripts/pr-release.sh criar          # abre o PR se nao houver um aberto e a develop estiver a frente da main; comenta
#   scripts/pr-release.sh comentar <pr>  # so atualiza o comentario (usado quando o rotulo ou os commits mudam)
# DRY_RUN=1 imprime o comentario em vez de falar com o GitHub.
set -euo pipefail
cd "$(dirname "$0")/.."
acao="${1:?use: criar | comentar <pr>}"
dir=$(dirname "$0")

git fetch -q origin main develop --tags

# "O que entra": os PRs mergeados na develop desde a main (titulo e numero); commits diretos entram pelo assunto.
lista() {
  git log --first-parent --reverse --format=%H origin/main..origin/develop | while read -r h; do
    assunto=$(git log -1 --format=%s "$h")
    if [[ "$assunto" =~ ^Merge\ pull\ request\ \#([0-9]+) ]]; then
      titulo=$(git log -1 --format=%b "$h" | head -1)
      echo "- ${titulo:-$assunto} (#${BASH_REMATCH[1]})"
    elif [[ ! "$assunto" =~ ^Merge\  ]]; then
      echo "- $assunto"
    fi
  done
}

comentar() {
  local pr="$1" rotulos="${2:-}"
  eval "$(ROTULOS="$rotulos" bash "$dir/proxima-versao.sh" origin/develop)"
  local corpo
  corpo=$(cat <<EOF
<!-- versao-release -->
## Próxima versão: \`$VERSAO\`
Calculada a partir de \`$BASE\`: $MOTIVO. A tag e a release são criadas **depois** do deploy da produção.

Para mudar a versão, ponha o rótulo \`release:minor\` ou \`release:patch\` neste PR (vale o que estiver nele quando a tag for criada). Sem rótulo, \`feat:\` sobe o número do meio e o resto sobe o último.

### O que entra
$(lista)
EOF
)
  if [ -n "${DRY_RUN:-}" ]; then printf '%s\n' "$corpo"; return; fi
  gh pr comment "$pr" --body "$corpo" --edit-last --create-if-none
}

case "$acao" in
  comentar)
    pr="${2:?informe o numero do PR}"
    rotulos=$(gh pr view "$pr" --json labels --jq '[.labels[].name] | join(",")')
    comentar "$pr" "$rotulos"
    ;;
  criar)
    if [ "$(git rev-list --count origin/main..origin/develop)" -eq 0 ]; then echo "develop nao esta a frente da main."; exit 0; fi
    pr=$(gh pr list --base main --head develop --state open --json number --jq '.[0].number // empty')
    if [ -z "$pr" ]; then
      corpo=$(cat <<'EOF'
PR de release aberto automaticamente depois de um deploy do dev bem-sucedido. A lista do que entra e a versão calculada estão no comentário abaixo.

## Produção
**O merge dispara o deploy de produção pelo Actions**, que espera a aprovação no environment `prod`. Antes de aprovar, abra o log do passo "Conferir o changeset": só mudanças esperadas.

Depois do deploy, a tag e a release são criadas sozinhas.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)
      url=$(gh pr create --base main --head develop --title "release: develop → main ($(date -u +%F))" --body "$corpo")
      pr="${url##*/}"
      echo "PR de release aberto: $url"
    else
      echo "Ja existe o PR de release #$pr."
    fi
    comentar "$pr" "$(gh pr view "$pr" --json labels --jq '[.labels[].name] | join(",")')"
    ;;
  *) echo "acao desconhecida: $acao" >&2; exit 1 ;;
esac
