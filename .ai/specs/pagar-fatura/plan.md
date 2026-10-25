# Plan: pagar a fatura do cartão (itens 33b e 35)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/domain/fatura.go`, `service/cartoes.go`,
`service/previsto.go`, `service/lancamento.go`, `service/fechamento.go`, `web/js/app.js`). O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** rascunho para aprovação do dono (2026-10-25).

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| A entrada do cartão em `CFG#CARTOES.porBanco[meio]` tem `diaPadrao`, `inicio` e `faturas[ym].{fechamento, valorBanco, conferida}` | `domain.CartaoDe`, `service/cartoes.go` | `pagador` e vencimento entram na mesma entrada, na mesma forma (`domain.Cartao` e `FaturaCfg` ganham campos) |
| A conferência grava `conferida = {total, em, por}` e o aviso "mudou desde a conferência" é só informação | `service.Cartoes.Conferir`, `mudou()` | A trava nova roda **antes** de a escrita do lançamento ser gravada, nas rotas de lançamento |
| A trava de mês de um banco é uma interface `Travas` que cada chamada de escrita consulta (`Checar(alvo, confirmado)`) e que devolve `ErroMesConferido` ou `ErroMesFechado` | `service/lancamento.go`, `service/fechamento.go` | A trava de fatura é **outra** consulta, na mesma chamada, com erro próprio (`ErroFaturaConferida`, `409 fatura_conferida`), sem confirmação que a destrave |
| O efeito de um lançamento no saldo é `EfeitoSaldo` (meio, mês, valor) e a trava compara o efeito de antes e de depois | `domain/previsto.go` | O **efeito na fatura** é o par irmão: `EfeitoFatura` (cartão, fatura, valor) e `MudaFatura(antes, depois)` |
| `GET /futuros` lê os previstos de um `Query` com `De` (mês passado em diante) e guarda o resultado pelo contador de alterações | `service/previsto.go` (`lerPrevistos`) | As faturas entram na mesma resposta; o cálculo delas entra no mesmo cache |
| `POST /tx/{id}/confirmar` confirma um previsto com valor, data e status e devolve `desfazer` | `service/previsto.go` | Pagar a fatura é um **comando parecido**, na rota do cartão, que cria o lançamento de pagamento |
| `excluirDoTotal` é calculado pelo servidor (catálogo e `transferParId`) | `service.derivar` | O pagamento de fatura também sai do total (`pagaFatura`) |
| O front não lê `excluirDoTotal` em lugar nenhum hoje (Histórico sem total até o item 42; Tendências só conta lançamentos com tag) | `web/js` | "Não conta de novo como gasto" fica garantido pelo campo e por o pagamento não ter tag; o item 42 passa a respeitar o campo |

## Desenho

- **Dados novos, todos opcionais, dentro de `CFG#CARTOES.porBanco[meio]`** (sem item, chave ou índice novo):
  - `pagador` (texto): o meio com que a fatura costuma ser paga; `faturas[ym].pagador` troca só aquela fatura.
  - `diaVencimento` (1 a 31): o dia do vencimento; `faturas[ym].vencimento` (`AAAA-MM-DD`) acerta só aquela fatura.
  - Um atributo opcional no **lançamento de pagamento**: `pagaFatura = { meio, ym }`, gravado só pelo servidor (entra em `Protegidos`; o backup,
    como o `previsto`, o mantém). **Aguarda a aprovação do dono no spec.**
- **Funções puras em `domain/`:**
  - `VencimentoDe(cartao, ym)`: o vencimento da fatura `ym` é `faturas[ym].vencimento`, se existir; senão o **primeiro dia depois do fechamento** de
    `ym` cujo dia do mês é `diaVencimento` (limitado ao último dia do mês); sem `diaVencimento` nem `vencimento` da fatura, não há vencimento e a fatura não entra em Futuros.
  - `PagadorDe(cartao, ym)`: `faturas[ym].pagador`, senão `pagador` do cartão, senão vazio.
  - `EfeitoFatura(l, cartao)` e `MudaFatura(antes, depois)`: a fatura em que a compra cai (R7 do item 33a: `FaturaBase` + `AjusteEfetivo`), o valor com sinal e se
    conta (crédito, não oculto, não excluído, não previsto). Muda quando o par (fatura, valor) muda.
- **`service.Cartoes` ganha:**
  - os campos novos nos comandos de `PUT /cartoes/{meio}` e `PUT .../faturas/{ym}`;
  - `POST /cartoes/{meio}/faturas/{ym}/pagar` com `{ valor, dataEvento, status, pagador? }`;
  - as faturas pagas e a pagar, para Futuros e para o painel da aba Cartão.
- **A trava de fatura conferida** (`service/faturas_travadas.go`): um objeto por chamada, como as `Travas`, que lê `CFG#CARTOES` uma vez e responde
  `ChecarFatura(antes, depois)`. As rotas que escrevem lançamento (criar, editar, excluir, importar, restaurar, confirmar) chamam `ChecarFatura` junto
  com a trava de mês. Só roda quando o lançamento é de crédito de um meio que é cartão.
- **O pagamento:** `Cartoes.Pagar` valida (fatura de cartão, `valor > 0`, data não futura, status selecionável, pagador na lista de meios e diferente do cartão,
  fatura ainda sem pagamento), monta o lançamento (saída no pagador, descrição "Pagamento da fatura <mmm/aa> <cartão>", sem tags, `pagaFatura`,
  `excluirDoTotal` verdadeiro) e o grava por `Lancamentos.Criar`, o mesmo caminho de escrita de qualquer lançamento (travas de mês, histórico, contador).
  Quem acha o pagamento de uma fatura é uma leitura do período dela em diante (do fechamento até hoje), filtrando `pagaFatura`.
- **Futuros ganha o grupo `faturas`** na resposta de `GET /futuros`: para cada cartão com vencimento, a fatura **fechada e não paga**
  (valor = `valorBanco` se informado, senão o total calculado) e a **aberta** (total até agora), cada uma com `vencimento`, `pagador`, `atrasada` e
  o botão de pagar. Entram nos `meses` pelo mês do vencimento e em `atrasados` quando o vencimento já passou, sempre dentro da janela de Futuros.
- **O front não decide nada:** mostra o que `GET /futuros` e `GET /cartoes/{meio}/faturas/{ym}` mandam (vencimento, pagador, pagamento, diferença), e as
  linhas de fatura têm o mesmo gesto de confirmar. Os campos de pagador e vencimento são entradas do dono, validadas pelo servidor.

## Fases (cada uma é um PR)

- **Fase A, backend, só acrescenta:** os campos novos nos comandos do cartão, as funções puras, a trava de fatura conferida, `POST .../pagar`, as
  extensões de `GET /cartoes/{meio}/faturas/{ym}` e de `GET /futuros`, a ação de permissão, testes que falham antes e docs (`modelo-de-dados`,
  `security`, `arquitetura`). **Antes de ligar a trava no dev**, uma leitura confere, em cada espaço, quais faturas já estão conferidas.
- **Fase B, as telas (demo antes, aprovada):** a seção do cartão ganha "Paga com", "Vence dia" e, por fatura, "Vencimento" e "Paga com"; o painel da
  fatura mostra o pagamento e a diferença; Futuros ganha as linhas de fatura com "Já paguei" (janela de confirmar com valor, data, status e "Paga com");
  a mensagem de fatura conferida aparece quando uma escrita é recusada. O release só depois da B (a trava recusa escritas que o app de hoje faz).

## Riscos

| Risco | Resposta |
|---|---|
| A trava de fatura conferida recusa uma escrita que o app de hoje faz (o app antigo não conhece o erro) | O release só depois da fase B; o erro vem com mensagem clara (como o de mês conferido) e o app mostra a mensagem do servidor |
| Uma fatura conferida por engano trava as compras dela | O "Desfazer conferência" reabre; a trava vale só para a fatura conferida, não para as vizinhas |
| Compra lançada depois da conferência com data de fatura conferida | Recusada (R7); o dono desfaz a conferência, lança e confere de novo |
| O pagamento criado no mês fechado de um banco | Passa pela trava de mês normal (conferido recusa; fechado por data pede confirmação), como qualquer lançamento |
| Pagar a fatura duas vezes | Uma fatura tem um pagamento; o segundo é recusado (`409 fatura_paga`) até o primeiro ser excluído; excluir o pagamento faz a fatura voltar para Futuros |
| O valor da fatura fechada muda com compra antiga lançada depois | O dono digita o valor da operadora (`valorBanco`); conferir trava. Sem isso a fatura mostra o calculado, como a aba Cartão já faz |
| Ler os pagamentos a cada abertura de Futuros pesa | A leitura vai do fechamento da fatura em diante (dias), cabe no cache pelo contador de alterações |
| Pagador igual ao cartão, ou um cartão pagando outro cartão | O servidor recusa pagador que seja um meio que é cartão (decisão 4 abaixo) |

## Decisões que o plan fixou e o dono pode contestar

1. **O cartão não vira entidade, não ganha id e não há migração** (decisão do dono, 2026-10-25).
2. **Uma fatura tem um pagamento** (sem parcial nem mínimo).
3. **O valor mostrado da fatura fechada** é `valorBanco` (o que a operadora cobra, digitado pelo dono) e, sem ele, o total calculado.
4. **O pagador não pode ser um meio que é cartão** (cartão paga fatura pelo banco, não por outro cartão).
5. **O pagamento não leva tags** e sai do total (`excluirDoTotal`), para não contar de novo como gasto.
6. **Só o dono** configura o pagador, o vencimento e paga a fatura.
7. **Fatura sem vencimento** (nem `diaVencimento` no cartão nem `vencimento` na fatura) não aparece em Futuros: o dono só ganha a linha depois de dizer quando vence. **Sem `pagador`** a linha aparece e o pagador é escolhido na janela de pagar.
8. **A trava de fatura conferida não tem confirmação que a destrave:** só "Desfazer conferência".
