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
- **R9.** O servidor **não** valida lançamento contra o `ativoAte`: lançar, editar, importar e restaurar num meio inativo continuam funcionando.
- **R12.** `GET /spaces/{sid}/saldos` e `GET /spaces/{sid}/cartoes` trazem só os meios **ativos** (R2), na lista e no total (em Cartão, também entre os meios livres para virar cartão); o meio desativado volta, com o saldo do passado, ao ser reativado. A conferência de um meio
  (`/fechamentos/conferir`) e o cálculo dele não mudam. A aba Saldos do front só desenha o que o servidor mandou (Teste M-T17).
- **R10.** Fechamento, conferência, fatura e histórico não mudam. Nenhum lançamento é regravado. O passado não é tocado.
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
