#!/usr/bin/env bash
# Cria usuarios no Cognito (convite por e-mail) e o espaco compartilhado no DynamoDB.
# Uso: scripts/seed.sh <stack> <email-dono> [<email-membro>] ["Nome do espaco"]
# - Sem membro: cria so o dono e o espaco.
# - Rodar de novo com o membro: reaproveita o espaco que o dono ja tem e adiciona o membro.
set -euo pipefail
STACK=${1:?uso: seed.sh <stack> <email-dono> [<email-membro>] [nome]}
OWNER=${2:?email do dono}
MEMBER=${3:-}
NOME=${4:-Financas da Familia}
# a tabela do dev e a da stack financas-dev; em qualquer outra, a de producao
if [ "$STACK" = "financas-dev" ]; then TABLE=FinancasApp-dev; else TABLE=FinancasApp; fi

POOL=$(aws cloudformation describe-stacks --stack-name "$STACK" \
  --query "Stacks[0].Outputs[?OutputKey=='UserPoolId'].OutputValue" --output text)

criar() { # cria (se nao existir) e imprime o sub
  aws cognito-idp admin-create-user --user-pool-id "$POOL" --username "$1" \
    --user-attributes Name=email,Value="$1" Name=email_verified,Value=true \
    --desired-delivery-mediums EMAIL >/dev/null 2>&1 || echo "(usuario $1 ja existe)" >&2
  aws cognito-idp admin-get-user --user-pool-id "$POOL" --username "$1" \
    --query "UserAttributes[?Name=='sub'].Value" --output text
}
put() { aws dynamodb put-item --table-name "$TABLE" --item "$1" >/dev/null; }

SUB_DONO=$(criar "$OWNER")

# espaco do dono: reaproveita se ja existir
SPACE=$(aws dynamodb query --table-name "$TABLE" \
  --key-condition-expression "PK = :p AND begins_with(SK, :s)" \
  --expression-attribute-values "{\":p\":{\"S\":\"USER#$SUB_DONO\"},\":s\":{\"S\":\"SPACE#\"}}" \
  --query "Items[?role.S=='owner'].SK.S | [0]" --output text)
if [ "$SPACE" = "None" ] || [ -z "$SPACE" ]; then
  ID=$(python -c 'import uuid;print(uuid.uuid4())' 2>/dev/null || uuidgen)
  put "{\"PK\":{\"S\":\"USER#$SUB_DONO\"},\"SK\":{\"S\":\"SPACE#$ID\"},\"role\":{\"S\":\"owner\"}}"
  put "{\"PK\":{\"S\":\"SPACE#$ID\"},\"SK\":{\"S\":\"META\"},\"nome\":{\"S\":\"$NOME\"},\"ownerSub\":{\"S\":\"$SUB_DONO\"}}"
else
  ID=${SPACE#SPACE#}
fi
echo "Espaco: $ID"
echo "dono:   $OWNER ($SUB_DONO)"

if [ -n "$MEMBER" ]; then
  SUB_MEMBRO=$(criar "$MEMBER")
  if [ "$SUB_MEMBRO" = "$SUB_DONO" ]; then echo "membro igual ao dono: ignorado" >&2; exit 1; fi
  put "{\"PK\":{\"S\":\"USER#$SUB_MEMBRO\"},\"SK\":{\"S\":\"SPACE#$ID\"},\"role\":{\"S\":\"member\"}}"
  echo "membro: $MEMBER ($SUB_MEMBRO)"
fi
echo "Cada pessoa nova recebe a senha temporaria por e-mail."
