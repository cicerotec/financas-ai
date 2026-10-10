# Segurança e permissões

O backend aplica as permissões por papel (`backend/authz.go`, com testes); o front só reflete. Regras de segredo e de
conduta da IA estão em [`guardrails.md`](guardrails.md). A tela de permissões por usuário é o item 27 do backlog.

## Papéis: owner e member

| Ação | owner | member |
|---|---|---|
| Criar lançamento | sim | sim |
| Copiar lançamento | sim | sim |
| Editar / ajustar fatura | todos | **só os criados por ela** (`criadoPor == sub`) |
| Ocultar lançamento | todos | **só os dela** |
| Excluir lançamento | todos | **não** |
| Criar tag | sim | **só tags que não existem** (o servidor só acrescenta) |
| Renomear/excluir tag, combos, bancos, status, flags | sim | não |
| Ver saldos, cartão, tendências, futuros | sim | sim (só leitura) |
| Alterar saldo inicial, fechamento, valor do banco, conferir fatura, início do controle | sim | não |
| Conferir e reabrir mês de um banco (fechamento mensal) | sim | não (só vê) |
| Aparência (cores, layout, ordem) e o próprio nome | sim | sim, **só a dela** |
| Importar / exportar / limpeza | sim | não |

- Ela lê **todos** os lançamentos e notas (saldos, cartão e tendências são calculados no navegador).
