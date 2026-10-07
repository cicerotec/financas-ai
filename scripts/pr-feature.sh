#!/usr/bin/env bash
# Abre o PR (rascunho) da branch atual para a develop, se ainda nao houver. Roda no GitHub Actions a cada push em
# feature/**, fix/**, bug/**, docs/** e chore/**. Precisa de GH_TOKEN e BRANCH, e do historico completo.
set -euo pipefail
branch="${BRANCH:?informe BRANCH}"

git fetch -q origin develop
novos=$(git rev-list --count origin/develop..HEAD)
if [ "$novos" -eq 0 ]; then echo "Sem commits novos em relacao a develop."; exit 0; fi

# um PR aberto basta; um PR fechado sem merge significa "nao quero": nao reabre. Um PR ja mergeado nao impede outro.
existe=$(gh pr list --head "$branch" --state all --json state --jq '[.[] | select(.state == "OPEN" or .state == "CLOSED")] | length')
if [ "$existe" -gt 0 ]; then echo "Ja existe PR aberto ou fechado para $branch."; exit 0; fi

titulo=$(git log --reverse --format=%s origin/develop..HEAD | head -1)
commits=$(git log --reverse --format='- %s' origin/develop..HEAD | head -30)
corpo=$(cat <<EOF
PR aberto automaticamente em rascunho (draft) no push da branch. Clique em **Ready for review** quando terminar.

## Commits
$commits

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)
gh pr create --draft --base develop --head "$branch" --title "$titulo" --body "$corpo"
