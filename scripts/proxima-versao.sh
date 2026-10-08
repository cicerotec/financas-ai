#!/usr/bin/env bash
# Calcula a proxima versao (vX.Y.Z) a partir da ultima tag e dos commits ate <ref>.
#   Regra (0.x.y): algum commit `feat:` => sobe o do meio; senao => sobe o ultimo.
#   Um rotulo do PR de release manda por cima: `release:minor` ou `release:patch`.
# Uso: ROTULOS="release:minor,outro" scripts/proxima-versao.sh <ref>
# Saida (stdout, para usar com eval): VERSAO, BASE, FEATS, FIXES, OUTROS, ROTULO, MOTIVO
set -euo pipefail
ref="${1:?informe o ref (ex.: origin/develop ou um sha)}"
rotulos="${ROTULOS:-}"

base=$(git tag --list 'v[0-9]*' --sort=-v:refname | head -1)
if [ -z "$base" ]; then base="v0.0.0"; faixa="$ref"; else faixa="$base..$ref"; fi
if ! [[ "$base" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then echo "tag invalida: $base" >&2; exit 1; fi
maior=${BASH_REMATCH[1]}; meio=${BASH_REMATCH[2]}; ultimo=${BASH_REMATCH[3]}

assuntos=$(git log --no-merges --format=%s "$faixa" || true)
conta() { printf '%s\n' "$assuntos" | grep -Ec "$1" || true; }
feats=$(conta '^feat(\(.+\))?!?:')
fixes=$(conta '^fix(\(.+\))?!?:')
total=$(printf '%s' "$assuntos" | grep -c . || true)
outros=$((total - feats - fixes))

rotulo=""
case ",$rotulos," in
  *,release:minor,*) rotulo="minor" ;;
  *,release:patch,*) rotulo="patch" ;;
esac

if [ -n "$rotulo" ]; then nivel="$rotulo"; motivo="rotulo release:$rotulo no PR"
elif [ "$feats" -gt 0 ]; then nivel="minor"; motivo="$feats commit(s) feat:"
else nivel="patch"; motivo="nenhum commit feat: ($fixes fix:, $outros outros)"; fi

if [ "$nivel" = "minor" ]; then meio=$((meio + 1)); ultimo=0; else ultimo=$((ultimo + 1)); fi
printf 'VERSAO=%q\nBASE=%q\nFEATS=%q\nFIXES=%q\nOUTROS=%q\nROTULO=%q\nMOTIVO=%q\n' \
  "v$maior.$meio.$ultimo" "$base" "$feats" "$fixes" "$outros" "$rotulo" "$motivo"
