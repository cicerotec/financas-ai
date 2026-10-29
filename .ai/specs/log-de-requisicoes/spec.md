# Spec: log de cada requisição

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id
(`L-T<n>`), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale o plan.

**Estado:** aprovado e implementado em 2026-10-29 (`feature/log-de-requisicoes`). Uma fase só, só backend.

Desvio do texto abaixo: o `handler` ficou dividido em `handler` (mede e registra) e `atender` (autentica e roda), e o nome da ação sai de `rotular(m, p)`, sem mudar a assinatura de `processar`. Os testes cobrem `rotular`, `linhaDeLog` e `registrar` (em `backend/log_requisicao_test.go`); não há teste de ponta a ponta do `handler`, porque ele valida o token do Cognito. Os ids L-T7 e L-T9 valem por construção: o log só existe em `handler` e nada na resposta foi tocado.

## Regras

- **R1.** Toda requisição HTTP que chega ao `handler` deixa **exatamente uma** linha de log ao terminar, qualquer que seja o resultado
  (sucesso, 4xx, 5xx ou erro Go).
- **R2.** A linha tem o formato `req rid=<requestId> m=<método> acao=<ação> status=<código> ms=<duração>`, com campos fixos nessa
  ordem e valores sem espaço.
- **R3.** `acao` é o nome da ação de `auth.RotaParaAcao` já classificada. Nas saídas que não chegam à classificação vale: `sem-token`
  (401 por token ausente ou inválido), `me` (`GET /me`) e `desconhecida` (404 por rota inexistente). Nunca é o caminho.
- **R4.** A linha **não contém** corpo, cabeçalho, query string, id de espaço, id de lançamento, e-mail nem `sub`.
- **R5.** `status` é o `StatusCode` da resposta; se `processar` devolver erro Go, é `500`.
- **R6.** `ms` é a duração inteira em milissegundos entre o início do `handler` e a resposta pronta.
- **R7.** O agendamento diário de avisos (`{"acao":"avisos"}`) **não** gera esta linha.
- **R8.** Nenhuma resposta, status ou efeito da API muda. O log não pode falhar a requisição.

## Testes

Cada um falha antes da mudança.

| Id | O que prova |
|---|---|
| L-T1 | Uma requisição de leitura válida produz uma linha com `acao=` da rota, `status=200` e `ms=` numérico (R1, R2, R3) |
| L-T2 | Token ausente produz `acao=sem-token status=401` (R1, R3) |
| L-T3 | `GET /me` produz `acao=me` e rota inexistente produz `acao=desconhecida status=404` (R3) |
| L-T4 | Sem permissão produz a linha com `status=403` e a `acao` do que foi negado (R1, R3) |
| L-T5 | Com id de espaço, id de lançamento, corpo, query string e `Authorization` na requisição, **nenhum** deles aparece na linha (R4) |
| L-T6 | Erro Go de `processar` produz `status=500` e ainda uma linha (R5) |
| L-T7 | O evento `{"acao":"avisos"}` não produz a linha (R7) |
| L-T8 | `linhaDeLog` com a mesma entrada devolve sempre o mesmo texto, na ordem dos campos de R2 |
| L-T9 | O corpo e o status da resposta são idênticos aos de antes, com a linha ligada (R8) |

**Quebrar de propósito:** acrescentar o `r.RawPath` à linha tem de fazer L-T5 falhar.

## Verificação no dev

1. Fazer o deploy no `financas-dev`, abrir o app e trocar de aba algumas vezes.
2. No Logs Insights do grupo `/aws/lambda/financas-api-dev`:
   `fields @timestamp, @message | filter @message like /^req / | stats count() by acao`
3. Conferir que a contagem por ação bate com o que foi clicado e que nenhuma linha traz id ou conteúdo.

## Fora do escopo

Contagem de RCU por requisição, `sub` no log, painéis e alarmes (ver Limites do `intent.md`).
