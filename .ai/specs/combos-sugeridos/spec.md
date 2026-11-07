# Spec: combos sugeridos calculados pelo servidor (item 57)

Roteiro verificável do [`intent.md`](intent.md) e do [`plan.md`](plan.md).

**Estado:** rascunho para aprovação (2026-11-06). Nada implementado.

## Regras

- **R1.** `CFG#COMBOS` ganha `sugeridos: [{ tags: [texto, ...], contagem: n }]` e `geradoEm: "AAAA-MM-DD"`. Nenhuma chave nova.
- **R2.** Um conjunto entra em `sugeridos` se tem **2 tags ou mais** e aparece em **3 lançamentos ou mais**, ignorando a ordem das tags (a chave é a lista ordenada). Lançamentos **excluídos** e **previstos** não contam.
- **R3.** Só contam os **1000 lançamentos mais recentes** (por data, do mais novo para trás); excluídos e previstos não entram nem nos 1000.
- **R4.** Ordem: maior `contagem` primeiro; empate pela chave em ordem alfabética. Guarda no máximo **10**.
- **R5.** A rotina é o evento `{"acao":"combos"}` (regra semanal do EventBridge, separada da dos avisos). Age só nos itens `tipo: "combos"` do `AVISOS#ATIVOS` e devolve `{"espacos": n, "sugeridos": n}`. Sem item ativo, não faz nada e devolve zeros.
- **R5a.** **Interruptor (só o dono):** `PUT /spaces/{sid}/combos/rotina { "ativa": true|false }` cria ou apaga o item `tipo: "combos"` (com o `sid`) e `GET` de `combos` devolve `rotinaAtiva`. Ligar não roda a rotina na hora. Desligar mantém `sugeridos`. Membro que não é dono recebe `403`. A aba Listas mostra o interruptor já ligado quando o item existe.
- **R5b.** A rotina diária dos avisos **ignora** o item `tipo: "combos"` (não é aviso), como já faz com `recorrencia`.
- **R6.** A rotina grava só `sugeridos` e `geradoEm`, por atualização parcial: **não altera `version` nem `lista`**. Item inexistente é criado com `lista: []` e `version` 0.
- **R7.** Falha em um espaço não impede os outros; o erro vai ao log com o `sid`.
- **R8.** Leitura em páginas pequenas com espera entre elas; nenhuma varredura da tabela inteira; só os `TX#` do espaço, do mais novo para trás, parando em 1000.
- **R9.** O PUT de `combos` (gravação manual) **preserva** `sugeridos` e `geradoEm` do item guardado; o corpo do front não precisa enviá-los, e se enviar são ignorados. O `version` sobe como hoje.
- **R10.** `GET` de `combos` devolve `lista`, `sugeridos` e `geradoEm`.
- **R11.** O Lançar mostra os combos manuais primeiro e depois os sugeridos (5, com "Mostrar mais"); um sugerido igual a um manual não aparece e o manual mostra a contagem.
- **R12.** Nenhuma tela conta combos: Lançar, filtro do Histórico, Tendências e texto da IA usam os sugeridos do servidor. Abrir o Lançar **não lê nenhum lançamento** (nenhuma chamada a `/tx`).
- **R13.** O passado não é tocado: nenhum lançamento é alterado.
- **R14.** Isolamento: a rotina lê e grava só no espaço do `sid` de cada item ativo.

## Testes (backend)

1. Conta: `[mercado, casa]` e `[casa, mercado]` contam como o mesmo conjunto (R2).
2. 2 usos não entram; 3 entram (R2).
3. Conjunto de 1 tag não entra (R2).
4. Excluído e previsto não contam (R2).
5. Com 1001 lançamentos, o mais antigo não conta; previstos e excluídos não entram nos 1000 (R3).
6. Empate ordena pela chave; mais de 10 conjuntos guardam 10 (R4).
7. Rotina grava `sugeridos` e `geradoEm` sem mudar `version` nem `lista` (R6).
8. Rotina cria o item quando não existe (R6).
9. Um espaço com erro não impede o seguinte (R7).
9a. Ligar cria o item; desligar apaga e mantém `sugeridos`; membro não dono recebe 403 (R5a).
9b. A rotina dos avisos não trata o item `combos` como aviso (R5b).
10. PUT manual preserva `sugeridos` e `geradoEm`, mesmo se o corpo trouxer outros (R9).
11. `GET` devolve `sugeridos` (R10).
12. Sem item ativo a rotina devolve zeros (R5).

## Testes (front, `web/*.test.js`)

1. Manual + sugerido: o manual vem primeiro; sugerido igual ao manual some e o manual leva a contagem (R11).
2. Sem `sugeridos`, só aparecem os manuais (item ausente = lista vazia).
3. `combosAtuaisLista` não usa `lancamentosCache`; o filtro do Histórico e o texto da IA usam a mesma função (R12).

## Conferência manual

Disparar a rotina uma vez no dev (`aws lambda invoke` com a entrada da regra), abrir o Lançar sem passar pelo Histórico e ver os combos sugeridos; depois repetir na produção.
