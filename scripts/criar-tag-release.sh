#!/usr/bin/env bash
# Cria a tag anotada e a release no GitHub para o commit publicado em producao (GITHUB_SHA). Roda no Actions depois do
# deploy da main bem-sucedido, entao a tag marca o que realmente foi para producao. Precisa de GH_TOKEN, contents: write,
# historico completo e tags. A versao vem de scripts/proxima-versao.sh (commits feat: e rotulo do PR de release).
set -euo pipefail
cd "$(dirname "$0")/.."
sha="${GITHUB_SHA:?sem GITHUB_SHA}"

if git tag --points-at "$sha" | grep -q '^v[0-9]'; then
  echo "O commit $sha ja tem tag ($(git tag --points-at "$sha" | tr '\n' ' ')). Nada a fazer."; exit 0
fi

pr=$(gh api "repos/${GITHUB_REPOSITORY}/commits/$sha/pulls" --jq '.[0].number // empty')
rotulos=""; titulo="release"
if [ -n "$pr" ]; then
  rotulos=$(gh pr view "$pr" --json labels --jq '[.labels[].name] | join(",")')
  titulo=$(gh pr view "$pr" --json title --jq .title)
fi

eval "$(ROTULOS="$rotulos" bash scripts/proxima-versao.sh "$sha")"
echo "Versao calculada: $VERSAO (a partir de $BASE: $MOTIVO)"
if git rev-parse -q --verify "refs/tags/$VERSAO" > /dev/null; then echo "A tag $VERSAO ja existe." >&2; exit 1; fi

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git tag -a "$VERSAO" -m "$VERSAO: $titulo" "$sha"
git push origin "$VERSAO"
gh release create "$VERSAO" --verify-tag --title "$VERSAO" --generate-notes --latest
echo "Release $VERSAO criada."
