#!/usr/bin/env bash
# Define (ou remove) o nome de exibicao de uma pessoa nos espacos dela.
# Uso: scripts/definir-nome.sh <stack> <email> "<nome>" [<space-id>]
#   - sem <space-id>: aplica em todos os espacos em que a pessoa participa
#   - nome vazio ("")  : remove o nome
# Precisa da sessao AWS aberta (MFA). Cada pessoa tambem pode trocar o proprio nome no app (Listas > Seu nome).
set -euo pipefail
STACK=${1:?uso: definir-nome.sh <stack> <email> "<nome>" [space-id]}
EMAIL=${2:?email}
NOME=${3-}
SPACE=${4:-}
TABLE=FinancasApp

# normaliza como o servidor: espacos sobrando, no maximo 30 caracteres
NOME=$(python -c 'import sys;print(" ".join(sys.argv[1].split()))' "$NOME")
if [ "$(python -c 'import sys;print(len(sys.argv[1]))' "$NOME")" -gt 30 ]; then
  echo "O nome pode ter no maximo 30 caracteres." >&2; exit 1
fi

POOL=$(aws cloudformation describe-stacks --stack-name "$STACK" \
  --query "Stacks[0].Outputs[?OutputKey=='UserPoolId'].OutputValue" --output text)
SUB=$(aws cognito-idp admin-get-user --user-pool-id "$POOL" --username "$EMAIL" \
  --query "UserAttributes[?Name=='sub'].Value" --output text)
[ -n "$SUB" ] && [ "$SUB" != "None" ] || { echo "Usuario $EMAIL nao encontrado no Cognito." >&2; exit 1; }

if [ -n "$SPACE" ]; then
  SKS="SPACE#${SPACE#SPACE#}"
else
  SKS=$(aws dynamodb query --table-name "$TABLE" \
    --key-condition-expression "PK = :p AND begins_with(SK, :s)" \
    --expression-attribute-values "{\":p\":{\"S\":\"USER#$SUB\"},\":s\":{\"S\":\"SPACE#\"}}" \
    --query "Items[].SK.S" --output text)
fi
[ -n "$SKS" ] && [ "$SKS" != "None" ] || { echo "$EMAIL nao participa de nenhum espaco." >&2; exit 1; }

if [ -n "$NOME" ]; then
  VALORES=$(python -c 'import json,sys;print(json.dumps({":a":{"S":sys.argv[1]}}))' "$NOME")
fi
for SK in $SKS; do
  KEY="{\"PK\":{\"S\":\"USER#$SUB\"},\"SK\":{\"S\":\"$SK\"}}"
  if [ -n "$NOME" ]; then
    aws dynamodb update-item --table-name "$TABLE" --key "$KEY" \
      --condition-expression "attribute_exists(PK)" \
      --update-expression "SET apelido = :a" --expression-attribute-values "$VALORES" >/dev/null
    echo "$EMAIL -> \"$NOME\" em $SK"
  else
    aws dynamodb update-item --table-name "$TABLE" --key "$KEY" \
      --condition-expression "attribute_exists(PK)" \
      --update-expression "REMOVE apelido" >/dev/null
    echo "$EMAIL -> nome removido em $SK"
  fi
done
