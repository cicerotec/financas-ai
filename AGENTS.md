# Finanças da Família: início de sessão (leia isto primeiro)

App de finanças da família: `web/` (HTML, CSS e JS sem framework), `backend/` (Go, Lambda, DynamoDB), `.ai/` (contexto para quem constrói com IA) e `BACKLOG.md` (roadmap).
Respostas e documentos em português.

## 1. Comece assim (obrigatório)
1. Leia **inteiro** `.ai/guardrails.md`. É a fonte das regras. O resumo da seção 3 existe para não esquecer; **onde divergir, vale o guardrails**.
2. Leia o que a tarefa pede, **inteiro** (tabela abaixo). Não use `head`, `limit` nem trechos; se o arquivo for grande, leia em partes até o fim.
3. Antes de agir, responda: arquivos lidos (com o número de linhas), o que entendeu do pedido e o que vai fazer. Se o pedido for só ler ou analisar, pare aí: **nenhum arquivo é alterado**.

| Tarefa | Leia também |
|---|---|
| Pedido "leia X" | X inteiro e o que ele indexa (`.ai/README.md` é o índice do `.ai/`) |
| Bug ou feature no front, tela | `.ai/padrao-de-telas.md` e a demo-padrão mais recente da tela em `.ai/demos/` (a tabela no início do padrao-de-telas aponta qual é) |
| Backend, rota, permissão | `.ai/arquitetura.md` e `.ai/security.md` |
| Item do roadmap | o item em `BACKLOG.md` e `.ai/specs/<feature>/{intent,plan,spec}.md` |
| Decisão de produto, nomes, status | `.ai/decisoes.md` |
| Release, hotfix, bug em produção | `.ai/fluxo-de-versoes.md` |

O `BACKLOG.md` só é lido quando a tarefa pedir (item do roadmap ou pedido explícito).

## 2. Quando pedirem "leia tudo do .ai"
Leia `.ai/README.md` e cada documento de primeiro nível que ele lista (`specs/` e `demos/` só os do assunto do pedido). Termine dizendo o que leu e o que não leu.

## 3. Resumo das regras duras (texto completo: `.ai/guardrails.md`)
- **Git:** antes da aprovação do dono não se commita nem se envia nada. Aprovado: teste, commite e `git push origin <branch>` (o push abre o PR). Mantra, sempre nesta ordem: `git fetch`, `git checkout develop`, `git pull`, `git checkout -b <prefixo>/<assunto>`. Prefixos: `feature/`, `fix/`, `bug/`, `docs/`, `chore/`. Nunca commitar em `develop` ou `main`. PR mergeado: não envie mais nada para a branch. Um assunto por branch e por PR. **Se o ambiente da sessão impuser outro nome de branch, avise antes de usá-la.**
- **Tela:** nenhuma alteração de tela entra no código sem demo aprovada; a demo parte da última versão da tela.
- **O front não decide:** regra de negócio, saldo, totais e lista de status vêm do backend; o front só exibe.
- **DynamoDB:** não crie, renomeie nem remova chave ou campo sem discutir com o dono.
- **Cada tela lê só o que é dela** (`web/carga.test.js` garante).
- **Nomes na tela:** `banco` aparece como "Meio de pagamento" e `status` como "Forma de pagamento", por extenso.
- **Provas:** não escreva "provavelmente" sobre o que dá para ler no código ou no log; leia e mostre a evidência. Bug: reproduza antes de corrigir.
- Pedido que contrarie o guardrails: aponte o conflito antes de fazer. Especificação que não cobre o caso: pergunte.

## 4. Validar
Backend: `cd backend && go vet ./... && go test ./...`. Front: `node web/<nome>.test.js` (o CI roda todos os `web/*.test.js`). Correção de bug leva um teste que falha sem ela.
