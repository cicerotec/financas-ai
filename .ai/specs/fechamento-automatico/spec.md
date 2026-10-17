# Spec: fechamento automático do mês anterior

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id
(`A-T<n>` para o backend da fase A), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale
o plan.

**Estado:** aprovado pelo dono. **Fase A (backend) implementada**; as fases B e C (tela) esperam a demo.

## Regras

**Quando um mês é fechado por data**
- **R1.** O mês de um evento é o da `dataEvento` no fuso de São Paulo (`domain.MesDe`, o mesmo do fechamento).
- **R2.** Em um instante `agora` (fuso de São Paulo), o mês M é **aberto** se for o mês atual, um mês futuro, ou o mês anterior ao
  atual enquanto o **dia do mês for 7 ou menos**. Qualquer outro mês é **fechado por data**. Exemplos com `agora` = 2026-10-07
  23:59: outubro, setembro e novembro abertos; agosto fechado. Com `agora` = 2026-10-08 00:00: setembro **fechado**. Com `agora` =
  2027-01-05: dezembro de 2026 aberto; novembro fechado.
- **R3.** O instante é o relógio do servidor (`agora()` da raiz), nunca um valor enviado pelo cliente. A função de domínio recebe
  esse instante como parâmetro. O limite "dia 7" é a **constante** `domain.DiaLimiteMesAnterior = 7` (não é variável, nem global), e a
  mensagem do erro e os testes a usam em vez de repetir o número.
- **R4.** Não grava estado nem campo no lançamento: a regra é função do mês e do relógio.

**O que a regra exige**
- **R5.** Escrever num lançamento cujo **alvo** (banco e mês) cai em mês fechado, **e que muda o saldo**, exige confirmação:
  criar e excluir sempre mudam o saldo; editar só quando `MudaSaldo` (valor, tipo, status, banco, data, oculto); restaurar quando
  `MudaSaldo` ou quando reativa um excluído. Editar descrição, nota e tags não exige.
- **R6.** Mudar a data de um mês aberto para um fechado, ou o contrário, confere os **dois** alvos (o de antes e o de depois).
- **R7.** O que é **anterior ao início do controle** do banco (`dataInicio` em `CFG#SALDOS`) não exige confirmação, como na trava
  do conferido. Sem `dataInicio`, nada é isento.
- **R8.** A ordem é: **conferido primeiro** (`409 mes_conferido`, absoluto, a confirmação **não** o destrava), depois o fechado
  por data (`409 mes_fechado`, confirmável).
- **R9.** A confirmação é o parâmetro de query `confirmaMesFechado=1` em qualquer rota de escrita (`POST /tx`, `PUT`, `DELETE`,
  `POST /tx/batch`, `POST .../restaurar`). Outro valor ou ausência = sem confirmação. Ela nunca é gravada no lançamento.

**Registro**
- **R10.** Quando a confirmação foi **necessária** (a regra exigiria) **e dada**, a versão gravada leva o atributo
  `confirmouMesFechado: true` (só no item `HIST#`). Confirmação dada sem necessidade não grava nada. A versão 1 preguiçosa nunca
  leva o atributo (o estado de antes não foi uma ação de ninguém).
- **R11.** `GET /historico` e `GET .../versoes` devolvem `confirmouMesFechado` (boolean) em cada versão ou linha.

**Importação**
- **R12.** No lote, a confirmação vale para todos os itens. Sem ela, cada item em mês fechado vai para `rejeitados` com
  `codigo: "mes_fechado"`, e os demais gravam. As versões dos itens que precisaram da confirmação levam o atributo (R10).

**Restauração e exclusão**
- **R13.** `GET .../versoes` devolve em cada versão `exigeConfirmacao` (boolean): restaurá-la cairia em mês fechado e, portanto,
  pediria confirmação. `restauravel` **não** vira `false` por isso.
- **R14.** O `DELETE` bem-sucedido devolve, além de `ok` e `_seq`, `desfazer: {"versao": N}`, a versão de antes da exclusão (a
  que o Desfazer restaura). Excluir o que já está excluído, ou o que não existe, **não** devolve `desfazer`.

## Contratos de API

Erros seguem o formato de hoje.

### `409 mes_fechado`

```json
{ "erro": "o mes 2026-08 do banco Bradesco ja fechou (so o mes atual e o anterior, ate o dia 7, ficam livres): confirme para alterar",
  "codigo": "mes_fechado", "banco": "Bradesco", "mes": "2026-08", "confirmavel": true }
```

### Exemplos (curl)

Variáveis como no spec do histórico: `API`, `SID`, `TOKEN`, `ID`; hoje é 16/10/2026, então agosto está fechado.

Editar o valor de um lançamento de agosto, **sem** confirmação:
```bash
curl -s -X PUT "$API/spaces/$SID/tx/$ID?de=2026-08-10T12:00:00.000Z" "${H[@]}" -d '{ "valor": 99 }'
```
```json
{ "erro": "o mes 2026-08 do banco Bradesco ja fechou (so o mes atual e o anterior, ate o dia 7, ficam livres): confirme para alterar",
  "codigo": "mes_fechado", "banco": "Bradesco", "mes": "2026-08", "confirmavel": true }
```
Status `409`; nada foi gravado. O mesmo `PUT` **com** confirmação (`200`, a versão `update` leva `confirmouMesFechado: true`):
```bash
curl -s -X PUT "$API/spaces/$SID/tx/$ID?de=2026-08-10T12:00:00.000Z&confirmaMesFechado=1" "${H[@]}" -d '{ "valor": 99 }'
```

Só a descrição, sem confirmação (`200`, não exige nada: não muda o saldo):
```bash
curl -s -X PUT "$API/spaces/$SID/tx/$ID?de=2026-08-10T12:00:00.000Z" "${H[@]}" -d '{ "descricao": "Mercado do bairro" }'
```

Excluir, com a resposta nova (`desfazer`):
```bash
curl -s -X DELETE "$API/spaces/$SID/tx/$ID?de=2026-10-10T12:00:00.000Z" "${H[@]}"
```
```json
{ "ok": true, "_seq": 105, "desfazer": { "versao": 2 } }
```

Uma versão no histórico, com o registro da confirmação (`GET .../versoes`):
```json
{ "versao": 3, "acao": "update", "porSub": "sub-demo-1", "porNome": "Ana", "em": "2026-10-16T13:00:00.000Z",
  "confirmouMesFechado": true, "exigeConfirmacao": false, "restauravel": true, "motivo": "", "deVersao": null,
  "diff": [ { "campo": "valor", "antes": 180, "depois": 99 } ], "registro": { "id": "k3x9a1m7q2w8e5r4t6y1", "valor": 99 } }
```

Importar com um item em mês fechado, sem confirmação (`200`, o item vai para `rejeitados`):
```json
{ "gravados": 1, "pendentes": [], "_seq": 106,
  "rejeitados": [ { "indice": 1, "codigo": "mes_fechado", "motivo": "mes fechado (Bradesco, 2026-08): confirme para importar" } ] }
```

## Fases e testes

### Fase A: backend

Muda: `domain/` (regra, erro, `Versao.ConfirmouMesFechado`), `service/` (`Checar` com confirmação em criar, editar, excluir,
importar e restaurar), `controller/` (query e `409`), `fechamento.go`/`efeitos.go` (a trava), `repository/` (o atributo), `main.go`
(passa a query). Nenhuma rota nova.

| Id | Cenário | Esperado |
|---|---|---|
| A-T1 | `MesFechadoPorData` em tabela: dia 7 às 23h59 e dia 8 às 0h em SP; o mesmo instante em UTC; atual, futuro, anterior e antigo; virada de ano | R2, inclusive que o instante em UTC não decide |
| A-T2 | Criar em mês fechado sem confirmação | `409 mes_fechado` com `banco`, `mes`, `confirmavel`; **nada gravado** (nem versão) |
| A-T3 | Criar em mês fechado com `confirmaMesFechado=1` | `201`; versão 1 `insert` com `confirmouMesFechado: true` |
| A-T4 | Criar em mês aberto, com a confirmação enviada sem necessidade | `201`; a versão **não** leva `confirmouMesFechado` |
| A-T5 | Editar valor em mês fechado: sem confirmação / com confirmação | `409` sem nada gravado / `200` com a versão `update` marcada |
| A-T6 | Editar só descrição, nota ou tags em mês fechado | `200` sem confirmação, sem marca na versão |
| A-T7 | Mudar a data de um mês aberto para um fechado (e o contrário) | `409` nos dois sentidos sem confirmação (R6) |
| A-T8 | Excluir em mês fechado: sem / com confirmação | `409` sem nada gravado / `200` com `desfazer`, versão `delete` marcada |
| A-T9 | Importar com itens em meses aberto e fechado: sem / com confirmação | sem: o fechado em `rejeitados` com `codigo: mes_fechado`, o aberto grava; com: todos gravam e os necessários levam a marca |
| A-T10 | Restaurar versão cujo alvo (ou o estado atual) está em mês fechado: sem / com confirmação | `409 mes_fechado` / `200` com a versão `restore` marcada |
| A-T11 | `GET .../versoes` | `exigeConfirmacao` por versão (R13); `restauravel` continua verdadeiro; `confirmouMesFechado` por versão |
| A-T12 | Mês **conferido** com `confirmaMesFechado=1` | continua `409 mes_conferido` (R8) |
| A-T13 | Anterior ao início do controle do banco | sem confirmação, mesmo em mês antigo (R7); sem `dataInicio`, nada é isento |
| A-T14 | O item vivo nunca ganha `confirmaMesFechado` nem `confirmouMesFechado` | confere o item gravado em cada caso |
| A-T15 | `DELETE` devolve `desfazer.versao` = versão de antes da exclusão; excluir o excluído ou o inexistente não devolve | R14 |
| A-T16 | O relógio do servidor decide: o mesmo pedido, com `agora` no dia 7 e no dia 8 | passa no dia 7, `409` no dia 8 |
| A-T17 | Teste de arquitetura | o service continua sem importar `time` para decidir a regra (ela vem da trava) |

`cenario()` dos testes passa a fixar o relógio em uma data de fevereiro de 2026 para que os testes atuais (que usam janeiro e
fevereiro de 2026) sigam sem confirmação; é ajuste do ambiente de teste, não de expectativa.

**Pronto quando:** A-T1 a A-T17 passam; o modelo de dados (`HIST#`, atributo novo, exemplo fictício), o `security.md` e o spec do
histórico (R16) estão em dia; a descrição do PR avisa que o release só sai depois da fase B.

### Fase B: tela da confirmação (demo antes do código)

A demo mostra: o `409 mes_fechado` ao salvar uma edição, ao criar e ao restaurar, com a janela "Este lançamento é de agosto, que
já fechou. Alterar mesmo assim?" (Cancelar, Alterar mesmo assim); o aviso antes de restaurar quando `exigeConfirmacao`; a
importação com a pergunta única do lote; e o que se perde (um toque a mais). O front **só reage ao `409`** e reenvia com
`confirmaMesFechado=1`; não calcula o mês fechado.

**Pronto quando:** demo aprovada, tela implementada com teste em Node, `padrao-de-telas.md` em dia.

### Fase C: excluir imediato (demo antes do código)

A demo mostra: confirmação, `DELETE` na hora, recusa antes de o item sumir (conferido ou fechado, este com a janela de confirmação),
aviso "Excluído" com **Desfazer** (que restaura a versão de `desfazer.versao`), e o erro do Desfazer. Muda o padrão 11b: some o
"pendente de 6 segundos" no servidor, e trocar de aba não efetiva nada (já foi efetivado).

**Pronto quando:** demo aprovada, tela implementada com teste, padrão 11b reescrito e o custo antigo ("trocar de aba efetiva a
exclusão") removido.

## Definição de pronto da feature

- A, B e C implementadas, com os testes passando e as demos aprovadas.
- O release para a `main` só depois de A e B juntas (C pode ir depois).
- Documentação em dia: `padrao-de-telas.md`, modelo de dados, `security.md`, BACKLOG (item 38) e o spec do histórico.

## Detalhes que o spec fixou e o dono pode contestar

1. A confirmação é **query** (`confirmaMesFechado=1`), não corpo.
2. O código é **`mes_fechado`**, com `confirmavel: true`; o texto do erro está em português sem acento, como o resto da API.
3. O **dia 7 é inclusivo** (no dia 8, às 0h de São Paulo, o mês anterior fecha).
4. **Conferido primeiro**; a confirmação nunca destrava o conferido.
5. A marca `confirmouMesFechado` só existe quando a confirmação foi necessária.
6. Na importação, a confirmação vale para o **lote inteiro**.
7. `restauravel` continua verdadeiro em mês fechado; quem avisa é `exigeConfirmacao`.
