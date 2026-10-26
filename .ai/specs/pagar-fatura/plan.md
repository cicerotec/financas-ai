# Plan: pagar a fatura do cartão (itens 33b e 35)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/domain/fatura.go`, `service/cartoes.go`,
`service/previsto.go`, `service/lancamento.go`, `service/fechamento.go`, `web/js/app.js`). O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** aprovado pelo dono em 2026-10-25 (spec e demo). **Fases A e B implementadas** (`feature/pagar-fatura-fase-a`, `feature/pagar-fatura-fase-b`).

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| A entrada do cartão em `CFG#CARTOES.porBanco[meio]` tem `diaPadrao`, `inicio` e `faturas[ym].{fechamento, valorBanco, conferida}` | `domain.CartaoDe`, `service/cartoes.go` | `pagador` e vencimento entram na mesma entrada, na mesma forma (`domain.Cartao` e `FaturaCfg` ganham campos) |
| A conferência grava `conferida = {total, em, por}` e o aviso "mudou desde a conferência" é só informação | `service.Cartoes.Conferir`, `mudou()` | A trava nova roda **antes** de a escrita do lançamento ser gravada, nas rotas de lançamento |
| A trava de mês de um banco é uma interface `Travas` que cada chamada de escrita consulta (`Checar(alvo, confirmado)`) e que devolve `ErroMesConferido` ou `ErroMesFechado` | `service/lancamento.go`, `service/fechamento.go` | A trava de fatura é **outra** consulta, na mesma chamada, com erro próprio (`ErroFaturaConferida`, `409 fatura_conferida`), sem confirmação que a destrave |
| O efeito de um lançamento no saldo é `EfeitoSaldo` (meio, mês, valor) e a trava compara o efeito de antes e de depois | `domain/previsto.go` | O **efeito na fatura** é o par irmão: `EfeitoFatura` (cartão, fatura, valor) e `MudaFatura(antes, depois)` |
| `GET /futuros` lê os previstos de um `Query` com `De` (mês passado em diante) e guarda o resultado pelo contador de alterações | `service/previsto.go` (`lerPrevistos`) | As faturas entram na mesma resposta; o cálculo delas entra no mesmo cache |
| `excluirDoTotal` é calculado pelo servidor (catálogo e `transferParId`) | `service.derivar` | O lançamento que leva a marca `pagaFatura` também sai do total |
| `faturaAjuste` é um campo que o dono escolhe e o servidor só valida (`-1`, `0`, `+1`, e só em crédito) | `domain.ValidarFaturaAjuste` | A marca `pagaFatura` é do mesmo tipo: dado escolhido pelo dono, validado pelo servidor |
| O front não lê `excluirDoTotal` em lugar nenhum hoje (Histórico sem total até o item 42; Tendências só conta lançamentos com tag) | `web/js` | "Não conta de novo como gasto" fica garantido pelo campo; o item 42 passa a respeitá-lo |

## Desenho

- **Dados novos, todos opcionais:**
  - Dentro de `CFG#CARTOES.porBanco[meio]` (sem item, chave ou índice novo): `pagador` (o meio com que a fatura costuma ser paga) e `diaVencimento` (1 a 31); em
    `faturas[ym]`: `pagador` e `vencimento` (`AAAA-MM-DD`).
  - No **lançamento**: `pagaFatura = { meio, ym }`, a marca. É **escolhida pelo dono** (vem do cliente, como o `faturaAjuste`) e o servidor só valida: o `meio`
    é um cartão, o `ym` é um mês válido, e o lançamento **não é uma compra de crédito** (o débito do mesmo banco do cartão pode levar a marca). `null` remove.
- **Funções puras em `domain/`:**
  - `VencimentoDe(cartao, ym)`: **só** `faturas[ym].vencimento`; sem ele não há vencimento e a fatura não entra em Futuros. O `diaVencimento` do cartão **não decide
    nenhuma fatura** (decisão do dono, 2026-10-25); `VencimentoSugerido` (o primeiro dia depois do fechamento cujo dia do mês é `diaVencimento`) só preenche a janelinha de editar.
  - `PagadorDe(cartao, ym)`: **só** `faturas[ym].pagador`. `PagadorSugerido` (da fatura, senão do cartão) só preenche a janelinha e pré-seleciona o meio no "Já paguei".
  - `ValidarPagaFatura(d, cartoes)`: confere a marca (cartão, mês, lançamento que não é crédito).
  - `EfeitoFatura(l, cartao)` e `MudaFatura(antes, depois)`: a fatura em que a compra cai (R7 do item 33a: `FaturaBase` + `AjusteEfetivo`), o valor com sinal e se
    conta (crédito, não oculto, não excluído, não previsto). Muda quando o par (fatura, valor) muda.
- **A marca não faz regra.** O único efeito dela no servidor é `excluirDoTotal` (para o pagamento não contar de novo como gasto) e a fatura sair de Futuros. Não há
  limite de pagamentos por fatura, nem de valor, nem de meio; excluir ou editar o lançamento é normal.
- **Não existe um comando "pagar".** "Já paguei" usa o `POST /tx` de sempre, com a marca no corpo. O que o servidor ajuda é a **sugestão** que vem no item de fatura
  de `GET /futuros` (`sugestao: { descricao, valor, pagador }`), para o app não montar texto nem valor.
- **Quem acha os lançamentos marcados:** uma leitura da janela da fatura (do início do período dela a 120 dias depois do fechamento), filtrando `pagaFatura`. É a
  mesma leitura que a aba Cartão já faz das compras, estendida; sem índice novo. Marca fora da janela não aparece na fatura (limite anotado no spec).
- **Futuros ganha o grupo `faturas`** na resposta de `GET /futuros`: para cada cartão com vencimento, a fatura **fechada sem pagamento marcado** (valor = `valorBanco`
  se informado, senão o total calculado) e a **aberta** (total até agora), cada uma com `vencimento`, `pagador`, `atrasada` e a `sugestao`. Entram nos `meses` pelo mês
  do vencimento e em `atrasados` quando o vencimento já passou, sempre dentro da janela de Futuros.
- **`GET /cartoes/{meio}/faturas/{ym}`** devolve também `pagamentos` (os lançamentos marcados: id, dia, meio, valor, status), `totalPago` e `diferencaPaga`
  (só informação).
- **A trava da fatura conferida** (`service/faturas_travadas.go`): um objeto por chamada, como as `Travas`, que lê `CFG#CARTOES` uma vez e responde
  `ChecarFatura(antes, depois)`. As rotas que escrevem lançamento (criar, editar, excluir, importar, restaurar, confirmar) a chamam junto com a trava de mês. Só roda
  quando o lançamento é uma compra de crédito de um meio que é cartão.
- **O front não decide nada:** mostra o que o servidor manda. Os campos de pagador, vencimento e a marca são entradas do dono, validadas pelo servidor.

## Fases (cada uma é um PR)

- **Fase A, backend, só acrescenta:** os campos novos nos comandos do cartão, as funções puras, a validação da marca, a trava de fatura conferida, as extensões de
  `GET /cartoes/{meio}/faturas/{ym}` e de `GET /futuros`, testes que falham antes e docs (`modelo-de-dados`, `security`, `arquitetura`). **Antes de ligar a trava no dev**,
  uma leitura confere, em cada espaço, quais faturas já estão conferidas.
- **Fase B, as telas (demo antes, aprovada):** a seção do cartão ganha "Paga com" e "Vence dia" e, por fatura, "Vencimento" e "Paga com"; a fatura mostra os pagamentos
  marcados; Futuros ganha as linhas de fatura com "Já paguei"; o Lançar ganha a linha "Paga a fatura"; o Histórico mostra a etiqueta; a mensagem de fatura conferida
  aparece quando uma escrita é recusada. O release só depois da B (a trava recusa escritas que o app de hoje faz).

## Riscos

| Risco | Resposta |
|---|---|
| A trava de fatura conferida recusa uma escrita que o app de hoje faz (o app antigo não conhece o erro) | O release só depois da fase B; o erro vem com mensagem clara (como o de mês conferido) e o app mostra a mensagem do servidor |
| Uma fatura conferida por engano trava as compras dela | O "Desfazer conferência" reabre; a trava vale só para a fatura conferida, não para as vizinhas |
| Compra lançada depois da conferência com data de fatura conferida | Recusada; o dono desfaz a conferência, lança e confere de novo |
| A marca pode ficar errada (cartão ou mês trocado) | É só uma label: edita-se o lançamento e troca-se a marca; nada mais depende dela além da linha de Futuros |
| O pagamento marcado de uma fatura fica fora da janela de leitura (pago muito tempo depois) | A fatura continua em Futuros como não paga até o dono marcar um pagamento dentro da janela; limite anotado, sem custo de índice |
| Ler os lançamentos marcados a cada abertura de Futuros pesa | A leitura vai do início do período da fatura a 120 dias depois do fechamento, e o resultado fica no cache pelo contador de alterações |
| Marca em lançamento de crédito do próprio cartão | O servidor recusa (a marca é de pagamento, não de compra) |

## Decisões que o plan fixou e o dono pode contestar

1. **O cartão não vira entidade, não ganha id e não há migração** (decisão do dono, 2026-10-25).
2. **O pagamento é um lançamento normal e a marca é uma label:** sem limite de pagamentos, de valor ou de meio; sem estado "paga" com consequência.
3. **A marca é escolhida pelo dono** (no Lançar e na edição; "Já paguei" já a leva preenchida) e validada pelo servidor.
4. **O valor mostrado da fatura fechada** é `valorBanco` (o que a operadora cobra, digitado pelo dono) e, sem ele, o total calculado.
5. **Um lançamento com a marca sai do total** (`excluirDoTotal`), para não contar de novo como gasto. É o único efeito da marca.
6. **A linha de Futuros some** quando existe pelo menos um lançamento marcado com a fatura (e não excluído).
7. **Só o dono** configura o pagador e o vencimento. A marca vale para quem pode editar o lançamento.
8. **Fatura sem `vencimento` próprio** (o `diaVencimento` do cartão não conta) não aparece em Futuros. **Sem `pagador`** a linha aparece e o meio é escolhido na janela de pagar.
9. **A trava de fatura conferida não tem confirmação que a destrave:** só "Desfazer conferência".
10. **O pagador sugerido** precisa ser um meio da lista (pode ser o próprio nome do cartão: o débito do mesmo banco); fora isso é livre (o pagamento em si é um lançamento comum, no meio que o dono quiser).
