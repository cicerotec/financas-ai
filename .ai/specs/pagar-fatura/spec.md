# Spec: pagar a fatura do cartão (itens 33b e 35)

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id
(`G-T<n>` para o backend da fase A), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale o plan.

**Estado:** rascunho para aprovação (2026-10-25). Nada implementado. Nomes como no spec do item 33a: o `banco` gravado chama-se **`meio`** nas rotas
e respostas novas (R0 daquele spec); o status, na tela, é a **forma de pagamento**.

## Regras

**Dados do cartão** (tudo dentro de `CFG#CARTOES.porBanco[meio]`, opcional, escrito só pelos comandos)
- **R1.** O cartão guarda `pagador` (nome de um meio da lista) e `diaVencimento` (inteiro de 1 a 31). Cada fatura (`faturas[ym]`) guarda `pagador` e
  `vencimento` (`AAAA-MM-DD`). Sem eles, valem os do cartão; sem os do cartão, não há.
- **R2.** `PUT /spaces/{sid}/cartoes/{meio}` aceita, além de `inicio`, `pagador` (texto ou `null`) e `diaVencimento` (inteiro de 1 a 31 ou `null`). `null`
  remove o campo.
- **R3.** `PUT /spaces/{sid}/cartoes/{meio}/faturas/{ym}` aceita, além de `fechamento` e `valorBanco`, `pagador` (texto ou `null`) e `vencimento`
  (data real ou `null`). O `vencimento` precisa ser **depois do fechamento** da fatura e no máximo 45 dias depois dele. Fatura de histórico (anterior
  ao `inicio`) aceita `valorBanco` e nada mais.
- **R4.** O `pagador` precisa ser um meio **da lista**, **diferente do cartão** e que **não seja um meio que é cartão**. Recusa com `400`.
- **R5.** Os comandos de R2 e R3 valem a regra de R25 do item 33a: controle otimista de `CFG#CARTOES`, tentam de novo, gravam só estes campos.

**Vencimento e linhas de fatura**
- **R6.** O **vencimento da fatura `ym`** (`domain.VencimentoDe`) é: `faturas[ym].vencimento`, se existir; senão o **primeiro dia depois do fechamento de `ym`**
  cujo dia do mês é `diaVencimento` (limitado ao último dia daquele mês); sem nenhum dos dois, **não há vencimento**.
- **R7.** O **pagador da fatura** (`domain.PagadorDe`) é `faturas[ym].pagador`, senão `pagador` do cartão, senão vazio.
- **R8.** Uma fatura é **aberta** quando o dia de hoje (São Paulo) é até o fechamento dela, e **fechada** depois (R10 do item 33a). Uma fatura é **paga**
  quando existe um lançamento de pagamento dela (R11) que não está excluído.

**Pagamento**
- **R9.** `POST /spaces/{sid}/cartoes/{meio}/faturas/{ym}/pagar` com `{ valor, dataEvento, status, pagador? }` (só o dono). Exige: o meio é cartão; a fatura não é
  de histórico; `valor` numérico maior que zero; `dataEvento` com dia **não futuro**; `status` **selecionável** do catálogo (R12 do item 41); pagador
  (o do corpo, senão o da fatura) definido e válido (R4); a fatura **ainda sem pagamento** (`409 fatura_paga` se já tem).
- **R10.** O pagamento é criado por `Lancamentos.Criar` (travas de mês, histórico, contador, `previsto`, `excluirDoTotal`), como **saída** no pagador, com
  `descricao: "Pagamento da fatura <mmm/aa> <meio>"`, `valor`, `dataEvento`, `status`, **sem tags**, `faturaAjuste` ausente e `excluirDoTotal: true`.
  O valor pago **pode ser diferente** do total calculado e do `valorBanco`.
- **R11.** O lançamento de pagamento leva `pagaFatura: { meio, ym }`, **gravado só pelo servidor**: o corpo de `POST/PUT /tx` e de cada item de `POST /tx/batch` **não
  define** `pagaFatura` (entra em `domain.Protegidos`; o lote o mantém quando vem de backup, como o `previsto`). O pagamento mexe no saldo do pagador
  como qualquer saída que conta no saldo e **não entra na fatura** (não é crédito).
- **R12.** A resposta é o lançamento criado (com `previsto` quando for o caso), `_seq` e `desfazer: { id }`, o id do pagamento. **Desfazer** é excluir esse
  lançamento (`DELETE /tx/{id}`, que já devolve `desfazer` para restaurar); sem o pagamento, a fatura volta a Futuros.
- **R13.** Excluir ou editar o lançamento de pagamento (valor, data, status) não muda a fatura; excluí-lo faz a fatura deixar de ser paga (R8). Ninguém pode
  mudar o `pagaFatura` de um lançamento pela API.

**Leitura**
- **R14.** `GET /spaces/{sid}/cartoes/{meio}/faturas/{ym}` passa a devolver também: `vencimento` (ou `null`), `pagador` (ou `null`), `paga` (booleano) e
  `pagamento: { id, valor, dia, pagador, status }` (ou `null`), e `valorPago` e `diferencaPaga` (`valorPago - total`, arredondado) quando há pagamento.
- **R15.** `GET /spaces/{sid}/cartoes` passa a devolver, em cada cartão, `pagador` e `diaVencimento` (ou `null`), para a seção do cartão mostrar o padrão.
- **R16.** `GET /spaces/{sid}/futuros` ganha `faturas`: uma lista, dentro da janela de Futuros (vencimento do primeiro dia do mês passado em diante) e
  só de cartões com vencimento (R6), de itens `{ meio, ym, rotulo, estado, valor, total, valorBanco, vencimento, pagador, atrasada, podePagar }` onde:
  - `estado` é `aberta` ou `fechada`;
  - `valor` é, na **fechada**, `valorBanco` se informado, senão o total calculado; na **aberta**, o total calculado até agora;
  - `atrasada` é verdadeiro quando o vencimento é anterior a hoje e a fatura não está paga;
  - só entram faturas **não pagas** com `total` diferente de zero ou `valorBanco` informado;
  - `podePagar` é o papel (só o dono).
  Os itens também entram em `meses` (pelo mês do vencimento) e em `atrasados` (vencimento anterior a hoje) com `tipo: "fatura"`, para a tela agrupar do
  mesmo jeito que os previstos. O `resumo` (`pagar`, `atrasados`) **soma as faturas** que estão em `faturas` (R19).
- **R17.** Os previstos continuam como são (R22 a R24 do item 41). As faturas não são previstos gravados e não viram lançamento até serem pagas.

**Trava da fatura conferida**
- **R18.** Uma fatura é **conferida** quando `faturas[ym].conferida` existe (R11 do item 33a). O **efeito de um lançamento na fatura** (`domain.EfeitoFatura`) é
  o par (cartão, fatura `ym`, valor com sinal) de uma compra que entra na fatura (R5 a R8 do item 33a: crédito, do banco do cartão, não excluída, não oculta, não
  prevista). `domain.MudaFatura(antes, depois)` é verdadeiro quando o efeito difere. As rotas `POST /tx`, `PUT /tx/{id}`, `DELETE /tx/{id}`,
  `POST /tx/batch`, `POST /tx/{id}/versoes/{n}/restaurar` e `POST /tx/{id}/confirmar` **recusam** a escrita (`409`, `codigo: "fatura_conferida"`, com `meio` e `ym`) quando
  `MudaFatura` e **qualquer** das faturas do efeito de antes ou de depois está conferida. Editar descrição, nota ou tags nunca é travado. **Não há confirmação que a
  destrave:** só `DELETE .../faturas/{ym}/conferir`.
- **R19.** O pagamento de fatura (R9) **não** passa pela trava de R18 (ele não é crédito). Conferir uma fatura **não** a torna paga.

**Permissão**
- **R20.** Ações: `cartao.escrever` (já existe, só o dono) vale para `pagar`; `cartao.ler` (dono e membro) para as leituras. Rota nova exige a ação em
  `auth/authz.go` com teste.

## Contratos de API

Variáveis como no spec do item 33a: `API`, `SID`, `TOKEN`. Hoje é 25/10/2026; dados fictícios.

### `PUT /spaces/{sid}/cartoes/CARTAO-A`
```json
{ "pagador": "BANCO-A", "diaVencimento": 5 }
```
### `PUT /spaces/{sid}/cartoes/CARTAO-A/faturas/2026-10`
```json
{ "pagador": "BANCO-B", "vencimento": "2026-11-05" }
```
### `POST /spaces/{sid}/cartoes/CARTAO-A/faturas/2026-09/pagar`
```json
{ "valor": 2500, "dataEvento": "2026-10-05T15:00:00.000Z", "status": "PAGO" }
```
Resposta:
```json
{ "id": "n4k8x2q9w1e7r5t3y6u0", "descricao": "Pagamento da fatura set/26 CARTAO-A", "valor": 2500, "tipo": "saida", "banco": "BANCO-A", "status": "PAGO",
  "dataEvento": "2026-10-05T15:00:00.000Z", "pagaFatura": { "meio": "CARTAO-A", "ym": "2026-09" }, "excluirDoTotal": true,
  "desfazer": { "id": "n4k8x2q9w1e7r5t3y6u0" }, "_seq": 231 }
```
Fatura já paga:
```json
{ "erro": "essa fatura ja tem um pagamento: exclua-o para pagar de novo", "codigo": "fatura_paga" }
```

### `GET /spaces/{sid}/futuros` (o trecho novo)
```json
{ "resumo": { "pagar": 5310.3, "receber": 7200, "atrasados": { "n": 1, "valor": 2500 } },
  "faturas": [
    { "meio": "CARTAO-A", "ym": "2026-09", "rotulo": "set/26", "estado": "fechada", "valor": 2500, "total": 3000, "valorBanco": 2500,
      "vencimento": "2026-10-05", "pagador": "BANCO-A", "atrasada": true, "podePagar": true },
    { "meio": "CARTAO-A", "ym": "2026-10", "rotulo": "out/26", "estado": "aberta", "valor": 1842.35, "total": 1842.35, "valorBanco": null,
      "vencimento": "2026-11-05", "pagador": "BANCO-A", "atrasada": false, "podePagar": true } ],
  "atrasados": [ { "tipo": "fatura", "meio": "CARTAO-A", "ym": "2026-09" } ],
  "meses": [ { "mes": "2026-11", "itens": [ { "tipo": "fatura", "meio": "CARTAO-A", "ym": "2026-10" } ] } ], "truncado": false }
```

### Escrita recusada pela trava
```json
{ "erro": "a fatura set/26 do cartao CARTAO-A esta conferida e travada: desfaça a conferencia na aba Cartao para alterar", "codigo": "fatura_conferida",
  "meio": "CARTAO-A", "ym": "2026-09" }
```

## Testes do backend (fase A)

Relógio fixo em 2026-10-25; dados fictícios; as funções puras são testadas em tabela e o resto por `processar`.

| Id | Cenário | Resultado esperado |
|---|---|---|
| G-T1 | `VencimentoDe`: fatura com `vencimento`; só `diaVencimento` maior e menor que o dia do fechamento; dia 31 em abril; sem nenhum dos dois | o do mês certo, limitado ao mês; vazio sem nenhum |
| G-T2 | `PagadorDe`: da fatura, do cartão, nenhum | a precedência de R7 |
| G-T3 | `EfeitoFatura` e `MudaFatura`: valor, data para outra fatura, `faturaAjuste`, oculto, excluído, previsto, status que deixa de ser crédito, descrição/tags | muda nos que mudam o par (fatura, valor); não muda em descrição/tags |
| G-T4 | `PUT /cartoes/{meio}` com `pagador` válido, fora da lista, o próprio cartão, um meio que é cartão, `diaVencimento` 0 e 32, `null` | `200`; `400`; `400`; `400`; `400`; remove o campo |
| G-T5 | `PUT .../faturas/{ym}` com `vencimento` antes do fechamento, 46 dias depois, data inexistente, de outro mês válido; fatura de histórico com `vencimento` | `400`; `400`; `400`; aceito; `400` |
| G-T6 | `pagar` com valor 0, data futura, status não selecionável, pagador ausente, pagador cartão, meio que não é cartão, fatura de histórico | `400` em cada um; nada gravado |
| G-T7 | `pagar` feliz, valor diferente do calculado | lançamento no pagador com `pagaFatura`, `excluirDoTotal`, sem tags; saldo do pagador diminui; `desfazer.id` |
| G-T8 | `pagar` duas vezes; excluir o pagamento e pagar de novo | `409 fatura_paga`; depois do excluir, `200` |
| G-T9 | **Forjar:** `pagaFatura`, `excluirDoTotal:false` e `tags` em `POST/PUT /tx` e `batch`; `PUT` que tenta trocar o `pagaFatura` | descartados / `pagaFatura` não muda; o lote de backup mantém |
| G-T10 | `GET .../faturas/{ym}` com e sem pagamento | `vencimento`, `pagador`, `paga`, `pagamento`, `valorPago`, `diferencaPaga` |
| G-T11 | `GET /futuros` com fatura fechada não paga, aberta, paga, sem vencimento, de cartão sem `diaVencimento`, de mês anterior à janela | só as não pagas e com vencimento, na janela; `valor` da fechada = `valorBanco` ou total; ordem; `atrasada` |
| G-T12 | `GET /futuros`: o `resumo` soma as faturas; o membro vê `podePagar: false` | somas certas; `false` |
| G-T13 | **Trava:** com a fatura conferida, `POST /tx` de crédito no período dela, `PUT` do valor, da data (para outra fatura), do `faturaAjuste`, `DELETE`, restaurar, importar | `409 fatura_conferida` em cada um; nada gravado |
| G-T14 | **Trava:** editar descrição/nota/tags, ocultar uma compra de **outra** fatura, lançar em fatura **não** conferida, pagar a fatura conferida | passam |
| G-T15 | **Trava:** depois de `DELETE .../conferir`, as mesmas escritas | passam |
| G-T16 | **Trava:** mover uma compra de uma fatura não conferida **para** uma conferida; e **de** uma conferida para fora | `409` nos dois (antes e depois são conferidos) |
| G-T17 | `authz`: `pagar` por owner e membro; rota desconhecida | liberado e `403`; `404` |
| G-T18 | O teste de arquitetura e os de reflexão continuam verdes; o cache de Futuros muda quando alguém grava um pagamento | verdes; relê |

Os testes que rejeitam ou travam (G-T4, G-T5, G-T6, G-T8, G-T9, G-T13, G-T16) são escritos primeiro e devem **falhar antes da mudança**.

## O que muda no front (fase B; demo antes)

- **Aba Cartão, seção de cada cartão:** "Paga com" (lista de meios que não são cartão) e "Vence dia" (1 a 31). Em cada fatura: "Vencimento" (data) e "Paga com"
  (só daquela fatura). O painel da fatura mostra o pagamento (valor, data, meio) e a diferença para o calculado, e "Desfazer conferência" avisa que a fatura está travada.
- **Futuros:** as faturas aparecem no meio pagador, na data do vencimento, com o valor, "fechada" ou "aberta" e "atrasada"; a ação "Já paguei" abre a janela de
  confirmar (valor, data, status e "Paga com"), cria o pagamento e dá o aviso com Desfazer.
- **Mensagem de fatura conferida:** a escrita recusada mostra a mensagem do servidor com o caminho para desfazer a conferência.
- Sai do front tudo que o servidor passa a mandar pronto; o front não calcula vencimento, valor ou atraso.

## Docs que ficam em dia na mesma entrega

- `.ai/specs/modelo-de-dados/` (`CFG#CARTOES`: `pagador`, `diaVencimento`, `faturas[ym].pagador` e `vencimento`; o atributo `pagaFatura`, com exemplo fictício),
  `.ai/security.md` (a trava de fatura conferida e `pagaFatura` protegido), `.ai/arquitetura.md` (a seção "Os três valores da fatura" e o pagamento) e o BACKLOG
  (itens 33, 35 e 17).
