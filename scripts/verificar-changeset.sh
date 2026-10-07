#!/usr/bin/env bash
# Confere o changeset pendente da stack e FALHA se ele remover ou substituir a tabela do DynamoDB ou o pool de login
# (Cognito): os dois guardam dados que nao se recriam. Imprime changeset=<nome> (vai para $GITHUB_OUTPUT no Actions);
# vazio quando nao ha mudanca. Uso: scripts/verificar-changeset.sh <stack>
set -euo pipefail
stack="${1:?informe a stack}"

nome=$(aws cloudformation list-change-sets --stack-name "$stack" \
  --query "Summaries[?ExecutionStatus=='AVAILABLE'] | [0].ChangeSetName" --output text 2>/dev/null || true)
if [ -z "$nome" ] || [ "$nome" = "None" ]; then
  echo "Sem mudancas para publicar."
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "changeset=" >> "$GITHUB_OUTPUT"
  exit 0
fi

json=$(aws cloudformation describe-change-set --stack-name "$stack" --change-set-name "$nome" --output json)
echo "$json" | jq -r '.Changes[].ResourceChange | "\(.Action)\t\(.Replacement // "-")\t\(.ResourceType)\t\(.LogicalResourceId)"' | column -t -s $'\t'

perigosas=$(echo "$json" | jq -r '
  .Changes[].ResourceChange
  | select(.ResourceType | IN("AWS::DynamoDB::Table", "AWS::Cognito::UserPool"))
  | select(.Action == "Remove" or .Replacement == "True" or .Replacement == "Conditional")
  | "\(.Action) \(.Replacement // "-") \(.ResourceType) \(.LogicalResourceId)"')
if [ -n "$perigosas" ]; then
  echo "BLOQUEADO: o changeset remove ou substitui um recurso com dados:" >&2
  echo "$perigosas" >&2
  aws cloudformation delete-change-set --stack-name "$stack" --change-set-name "$nome" || true
  exit 1
fi

[ -n "${GITHUB_OUTPUT:-}" ] && echo "changeset=$nome" >> "$GITHUB_OUTPUT"
exit 0
