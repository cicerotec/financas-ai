# Spec: meios de pagamento (item 43)

Roteiro verificável do [`intent.md`](intent.md) e do [`plan.md`](plan.md).

**Estado:** aprovado pelo dono em 2026-11-03 (inclui o `ativoAte` em `CFG#LISTAS`); fases A (#347) e B (#348) concluídas em 2026-11-03 e conferidas em dev pelo dono.

## Regras

- **R1.** `CFG#LISTAS.ativoAte` é um mapa `{ "<meio>": "AAAA-MM-DD" }`. Só existem chaves de meios presentes em `banco`. Ausente = ativo.
- **R2.** Um meio é **ativo** quando não tem `ativoAte` ou `hoje < ativoAte` (data do servidor, fuso de São Paulo). Quem calcula é o servidor.
- **R3.** `GET /spaces/{sid}/cfg/listas` devolve `meios: [{ nome, cartao, ativo, ativoAte? }]` (o `ativoAte` só nos que têm). O mapa cru `ativoAte` **não** sai no corpo.
- **R4.** O `PUT` de Listas **ignora** `ativoAte` do corpo (o servidor mantém o gravado) e **recusa** (`400`) um `banco` que tire um nome presente na lista gravada. Acrescentar nomes continua valendo.
- **R5.** `PUT /spaces/{sid}/meios/{meio}` com `{ "ativoAte": "AAAA-MM-DD" }` grava a data; com `{ "ativoAte": null }` remove (reativa). Só o dono. `404` se o meio não está em `banco`;
  `400` se a data for inválida. Não toca lançamento, saldo nem fatura. Sobe o `version` de `CFG#LISTAS`.
- **R6.** `DELETE /spaces/{sid}/meios/{meio}` remove o meio de `banco` e o seu `ativoAte`, **se e só se** nenhum lançamento o usa. Só o dono.
  - Há lançamento (vivo **ou excluído logicamente**, previsto ou não): `409` com `codigo: "MEIO_EM_USO"` e mensagem sem contagem ("tem lançamentos; desative o meio").
  - O meio existe em `CFG#CARTOES.porBanco`: `409` com `codigo: "MEIO_E_CARTAO"`.
  - O meio não está em `banco`: `404`.
  - Sem uso e sem cartão: `204`; `banco` sem o nome; `ativoAte` sem a chave; versão de `CFG#LISTAS` sobe.
- **R7.** A conferência de uso lê só quando o `DELETE` é chamado (nenhuma rota de leitura nem a abertura de Listas varre lançamentos) e **para no primeiro lançamento achado**.
- **R8.** Cada `DELETE` escreve uma linha de log `meios: remover meio=<n> paginas=<n> itens=<n> ms=<n> resultado=<ok|em_uso|cartao>`, sem conteúdo de lançamento (gatilho para a flag `jaUsado`).
- ~~**R9.** O servidor **não** valida lançamento contra o `ativoAte`: lançar, editar, importar e restaurar num meio inativo continuam funcionando.~~
  **R9 revogado em 2026-11-08 (item 58, [`BACKLOG.md`](../../../BACKLOG.md#item-58)):** o servidor passa a recusar criar, Transferir, Confirmar previsto, recorrência, restaurar e trocar o banco para um meio desativado
  (`MEIO_DESATIVADO`); importar só aceita `dataEvento` anterior à `dataInicio` do meio; desativar só a partir de hoje e só com saldo zero e sem evento. O teste M-T14 abaixo vale só para a fase do item 43 e será substituído.
- **R10.** Saldo, fechamento, conferência, fatura e histórico não mudam. Nenhum lançamento é regravado. O passado não é tocado.
- **R11.** Tela (fase B): Lançar e Transferência listam só os meios com `ativo`; o meio de um lançamento em **edição** fica na lista mesmo inativo; as demais telas usam todos. O front não compara data nem nome.
  A etiqueta "crédito" aparece quando `cartao` é verdadeiro (vindo de `CFG#CARTOES`).

## API

```
GET    /spaces/{sid}/cfg/listas        meios: [{ nome, cartao, ativo, ativoAte? }]
PUT    /spaces/{sid}/meios/{meio}      { "ativoAte": "2026-12-31" }  -> 200 { meio, ativoAte, ativo }
PUT    /spaces/{sid}/meios/{meio}      { "ativoAte": null }          -> 200 reativado
DELETE /spaces/{sid}/meios/{meio}      -> 204 | 404 | 409 MEIO_EM_USO | 409 MEIO_E_CARTAO
PUT    /spaces/{sid}/cfg/listas        ignora ativoAte; 400 se tirar um nome de banco
```

## Testes

| # | Cenário | Esperado |
|---|---|---|
| M-T1 | `PUT /meios/{meio}` com data futura, passada, hoje, `null`, formato inválido | `200` (ativo); `200` (inativo); `200` (inativo, pois `hoje < ativoAte` é falso); `200` reativa; `400` |
| M-T2 | `GET /cfg/listas` com `ativoAte` | `meios[].ativo` e `ativoAte` certos; o mapa cru não aparece |
| M-T3 | `PUT` de Listas com `ativoAte` no corpo | ignorado; o gravado continua |
| M-T4 | `PUT` de Listas tirando um nome de `banco` | `400`; lista intacta |
| M-T5 | `PUT` de Listas acrescentando um meio | `200`; o meio aparece ativo |
| M-T6 | `DELETE` de meio com lançamento vivo | `409 MEIO_EM_USO`; nada muda |
| M-T7 | `DELETE` de meio só com lançamento excluído logicamente; só com previsto | `409 MEIO_EM_USO` nos dois |
| M-T8 | `DELETE` de meio existente em `CFG#CARTOES` | `409 MEIO_E_CARTAO` |
| M-T9 | `DELETE` de meio sem uso | `204`; sem o nome em `banco` e sem a chave em `ativoAte`; `GET` mostra a lista nova |
| M-T10 | `DELETE` de nome que não existe | `404` |
| M-T11 | Contagem de leituras: meio em uso nas primeiras páginas; meio sem uso | para na 1ª página com achado; o sem uso lê a partição inteira; o log traz `paginas` e `itens` |
| M-T12 | Abrir Listas e `GET /tx` | nenhuma leitura extra de lançamentos (R7) |
| M-T13 | `authz`: member chama `PUT` e `DELETE` de meio | `403`; o `GET` de Listas do member traz o `ativo` |
| M-T14 | Lançar, editar e restaurar num meio inativo | `200`; saldo igual ao de um meio ativo (R9, R10) |
| M-T15 | Exportar e importar configurações com `ativoAte` | o campo é preservado; a importação não o apaga nem o inventa (a conferir) |
| M-T16 | Remover um meio com data futura e sem uso | `204`; o `ativoAte` sai junto |

## Telas (fase B, demo aprovada em 2026-11-03; implementada)

**Desvios da demo, avisados antes do PR:** (1) **Remover** é um passo só: o dono confirma "Remover X?", o app mostra "Conferindo…" e o servidor remove ou recusa na mesma chamada (a demo
mostrava "nunca foi usado" e pedia outra confirmação, o que exigiria uma conferência separada da remoção); (2) **Fim do mês** grava o 1º dia do mês seguinte (o meio vale até o fim do mês) e
a linha mostra essa data em "Deixa de ser sugerido em…".

- Demo: [`demo-listas-v3-meios`](../../demos/demo-listas-v3-meios.html), partindo da v2 de Listas.
- Card **Meios de pagamento**: uma linha por meio, tocável; etiqueta "crédito" quando `cartao`; "Inativo desde…" ou "Deixa de ser sugerido em…" a partir de `ativoAte`;
  interruptor "Mostrar meios inativos"; campo e botão Adicionar como hoje.
- Janela de ações: **Desativar…** (A partir de hoje, Fim do mês, Outra data; botões que quebram linha) ou **Reativar**, e **Remover**. Remover mostra "Conferindo…" e responde com
  o que o servidor disse: `204` pede a confirmação final e remove; `409 MEIO_EM_USO` explica e oferece Desativar; `409 MEIO_E_CARTAO` manda para a aba Cartão.
- Sai o `removerBanco` do navegador (a leitura de até 1000 lançamentos). Nenhum nome de status entra em `web/` (`sem-status.test.js`).
- Testes do front: `web/*.test.js` sem chamada a função indefinida; o combo do Lançar com um meio inativo em edição.

## Fase C: encerrar o meio (item 58, decidido em 2026-11-08)

Substitui o R9 e muda o R5. O texto das decisões está no item 58 do [`BACKLOG.md`](../../../BACKLOG.md#item-58). **Desativar passa a ser encerrar a conta.**

- **R12.** `PUT /meios/{meio}` só desativa **a partir de hoje**: `ativoAte` precisa ser a data de hoje do servidor (fuso de São Paulo); outra data, `400`. `null` reativa como antes. Meios que já têm
  `ativoAte` futuro ou passado gravado (dados antigos) continuam valendo pela regra de R2 e podem ser reativados.
- **R13.** Antes de gravar a desativação o servidor confere, sem alterar nada, se o meio ainda participa de algum evento. Se sim: `409` com `codigo: "MEIO_COM_PENDENCIA"` e `pendencias`, a lista
  de frases do que falta. Pendências: (0) **meio sem nenhum ponto de partida do saldo não desativa**: sem mês conferido, sem cache do fechamento automático e sem `dataInicio` ("Não há de onde partir o saldo de X: confira o saldo de um mês ou defina o início do controle em Saldos antes de desativar."; a recusa vem sem ler lançamento algum); (a) saldo diferente de zero (calculado pelo servidor **a partir do último fechamento**, a `dataInicio` é só o piso; arredondado a centavos, também fora do total); (b) lançamento previsto ou com data
  de **hoje em diante** no meio (inclui ponta de transferência futura; previsto no passado não existe e a conferência **não lê o passado**: a tabela tem 5 RCU); (c) recorrência ativa cuja última cópia está no meio; (d) meio que é cartão com alguma das **três últimas faturas** (a aberta e as duas fechadas antes dela, as não históricas) com valor maior que zero e **sem pagamento registrado** (a marca `pagaFatura`),
  com ou sem vencimento, sem depender de Futuros (a frase diz a fatura e o valor); fatura mais antiga que essas três não conta; (e) meio que é pagador padrão de algum cartão ou pagador de fatura em aberto. Os avisos pendentes são deste mesmo conjunto (de lançamento: b e c; de fatura: d).
- **R14.** Meio desativado (`ativo = false`) não recebe nada novo: `409` com `codigo: "MEIO_DESATIVADO"` ao **criar**, **copiar pela recorrência**, **registrar o ajuste da conferência**, **Transferir** (origem ou
  destino), **Confirmar previsto**, **restaurar** uma versão e **trocar o `banco`** de um lançamento para ele; também ao **excluir** um lançamento dele.
- **R15.** Editar um lançamento de meio desativado só aceita mudar `descricao`, `nota` e `tags` (e tirar a marca `recorrente`). Qualquer outro campo diferente do gravado: `409 MEIO_DESATIVADO`.
  A comparação é por **equivalência**, porque o formulário manda o registro inteiro: ausente, `null`, vazio, `0` e `false` valem o mesmo, e `dataEvento` é comparada como instante (sem os milissegundos também vale).
- **R16.** **Importar** num meio desativado aceita o registro só se `dataEvento` for **anterior** à `dataInicio` do meio em `CFG#SALDOS`; sem `dataInicio`, ou data igual ou posterior, o registro volta em
  `rejeitados` com `codigo: "meio_desativado"` e os demais entram. Em meio ativo, nada muda. A trava de mês conferido continua valendo por cima.
  **Também o registro que o lote vai sobrescrever ou ressuscitar** (mesmo `id` já gravado, inclusive excluído) precisa estar num meio ativo ou ser anterior ao `dataInicio` do meio dele; senão o importar tiraria valor de um meio encerrado.
- **R17.** **Restaurar** uma versão confere o `banco` do registro que vai voltar: meio fora da lista (removido) ou desativado é recusado com `409 MEIO_DESATIVADO` (mensagem explica). O histórico `HIST#` não muda.
- **R18.** `DELETE /meios/{meio}` ignora os lançamentos excluídos logicamente (`excluidoEm`): um meio só com excluídos pode ser removido. Vivo ou previsto continua `MEIO_EM_USO`.
- **R14b.** A rotina da recorrência, ao receber `MEIO_DESATIVADO` da cópia, **encerra a recorrência** (remove o item auxiliar, com log) em vez de tentar todo dia.
- **R19.** O pagador de cartão e de fatura (`PUT /cartoes/...`) precisa ser um meio ativo; desativado: `400`.
- **R21.** `PUT /cfg/saldos` não muda o `saldoInicial` nem a `dataInicio` de um meio desativado (mudaria o saldo sem lançamento e o limite do Importar): `409 MEIO_DESATIVADO`. Reativa, edita e desativa de novo (a conferência roda outra vez).
- **R22.** `POST /cartoes` não transforma um meio desativado em cartão: `409 MEIO_DESATIVADO` (fatura, pagador e vencimento são eventos novos).
- **R20.** `GET /saldos` e `GET /cartoes` marcam `encerrado: true` no meio desativado cujo saldo (ou fatura atual) é zero, para a tela escondê-lo; o total não muda. Um meio desativado **com saldo continua visível**
  (dado antigo): o servidor nunca esconde dinheiro. Nenhum dado novo no DynamoDB.

| # | Cenário | Esperado |
|---|---|---|
| M-T17 | Desativar com data de amanhã, passada, `null` | `400`, `400`, reativa |
| M-T18 | Desativar com saldo diferente de zero | `409 MEIO_COM_PENDENCIA` com a frase do saldo; nada gravado |
| M-T19 | Desativar com lançamento de data futura (previsto) e com ponta de transferência futura; com previsto do passado | `409 MEIO_COM_PENDENCIA` nos dois primeiros; o do passado não impede |
| M-T20 | Desativar com recorrência ativa cuja ponta é do meio | `409 MEIO_COM_PENDENCIA` |
| M-T21 | Cartão pagador padrão de outro cartão | `409 MEIO_COM_PENDENCIA` |
| M-T22 | Desativar meio zerado e sem evento | `200`; `ativo` falso |
| M-T23 | Criar, Transferir, Confirmar, ajuste da conferência, recorrência e trocar banco para meio desativado | `409 MEIO_DESATIVADO`; nada gravado |
| M-T24 | Editar descrição, nota e tags de lançamento de meio desativado; editar valor, data, status, ocultar; excluir | `200`; `409`; `409` |
| M-T25 | Importar para meio desativado: antes da `dataInicio`, depois, sem `dataInicio`, em meio ativo | entra; rejeitado; rejeitado; entra |
| M-T26 | Restaurar versão para meio desativado e para meio removido | `409 MEIO_DESATIVADO` nos dois |
| M-T27 | Remover meio só com lançamento excluído | `204`; com um vivo ou previsto, `409 MEIO_EM_USO` |
| M-T28 | Pagador de cartão desativado | `400` |
| M-T29 | `GET /saldos` e `GET /cartoes` com meio desativado zerado e outro desativado com saldo | só o zerado leva `encerrado` |
| M-T30 | Exceção do M-T14 e do M-T1 (datas futuras) | substituídos por M-T17 e M-T23 |
| M-T31 | Desativar cartão sem vencimento com a fatura atual e uma fechada com valor e sem pagamento registrado; registrar o pagamento de cada uma | `409 MEIO_COM_PENDENCIA` citando as duas e o valor; some uma a uma; com as duas pagas, `200` |
| M-T32 | `PUT /cfg/saldos` mudando saldo inicial ou `dataInicio` de meio desativado; mudando só outro meio | `409 MEIO_DESATIVADO` e nada gravado; `200` |
| M-T35 | Desativar meio sem mês conferido, sem cache e sem `dataInicio` | `409 MEIO_COM_PENDENCIA` pedindo o saldo de um mês ou o início do controle, sem ler lançamento algum |
| M-T37 | Desativar meio sem `dataInicio`, mas com um mês conferido | `200`, sem ler o passado |
| M-T36 | Desativar com `dataInicio`, com a leitura a partir do início dos tempos proibida no fake | `200` (nada lê o passado) |
| M-T33 | Transformar em cartão (`POST /cartoes`) um meio desativado; um meio ativo | `409 MEIO_DESATIVADO`; `200` |
