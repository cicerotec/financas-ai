# Intent: novas formas de pagamento (PIX, DEBITO, DINHEIRO e TED)

Item 56 do BACKLOG. É o "desmembramento do `PAGO`" que o item 41 deixou para depois (`arquitetura.md`, "Status, meios de pagamento e o
passado"). Este documento diz **o quê e por quê**; o como está no [`plan.md`](plan.md) e o roteiro verificável no [`spec.md`](spec.md).

**Estado:** aprovado em 2026-11-02; fase A (backend) implementada. Decisões do dono de 2026-11-02: entram **PIX, DEBITO,
DINHEIRO e TED**, em **ordem de uso**, com o `PAGO` no fim; todas valem para **entrada e saída**, sem validação de tipo; **não falta mais nada**
nesta frente, e qualquer outra forma (DREX, CRIPTO e o que vier) é uma entrada nova do catálogo, em outra versão, quando o app crescer.

## Por quê

Hoje toda movimentação que mexe no saldo é `PAGO`: não diz se foi um PIX, um débito ou dinheiro. O dono quer registrar **como** o dinheiro se
moveu, sem mexer nos saldos nem no histórico já gravado.

## O que queremos

1. Quatro entradas novas no `Catalogo` do backend, selecionáveis: **PIX**, **DEBITO**, **DINHEIRO** e **TED**. Contam no saldo **exatamente como
   o `PAGO`** e entram no total do Histórico como ele.
2. A lista segue uma **ordem de uso** (a mais usada primeiro; o primeiro é o padrão): PIX, DEBITO, CREDITO, DINHEIRO, TED e, por último, o
   **`PAGO`**, que **continua selecionável** como forma genérica, mas deixa de ser o padrão.
3. **Todas valem para entrada e saída.** O servidor não valida tipo: um estorno no débito é uma entrada real, e o estorno no cartão é uma entrada de
   `CREDITO`. O que torna uma entrada "salário" é a descrição ou a tag, não a forma; o salário do dono chega por **TED**.
4. O passado continua do usuário: um `PAGO` de 2018 pode virar `PIX`, um registro por vez, sem a trava de data, porque o efeito no saldo é o mesmo.
5. **O catálogo cresce sem refazer nada:** o que for entrando depois é uma linha em `domain/catalogo.go`, sem campo novo, sem migração e sem tocar no front.

## Decisões que ficaram de fora (e por quê)

- **BOLETO:** é o documento da cobrança, não o meio; o boleto se paga com PIX, DINHEIRO ou CREDITO. Entrar na lista daria dois nomes para o mesmo
  pagamento e deixaria o gráfico de forma de pagamento (Tendências) com fatias que se sobrepõem. "Quanto paguei de boleto" sai de uma tag.
- **RECEBIMENTO:** diz que entrou dinheiro, e isso o tipo `entrada` já diz; é o mesmo problema do boleto. O salário usa TED, com a tag `salário`.
- **Validação por tipo, `GET /status?tipo=` e a lista que muda ao trocar entre entrada e saída:** existiam só por causa do RECEBIMENTO e do DEBITO
  "só saída". Sem eles, a entrega não mexe em nenhuma tela.

## Limites (o que fica de fora)

- **Nada no DynamoDB muda:** o `status` já é um texto gravado no lançamento. Nenhum campo, chave ou item novo.
- **Nenhuma tela muda** (a lista de forma de pagamento já vem do servidor): sem demo. Só passam a aparecer os nomes novos e o PIX como padrão.
- Nenhum lançamento antigo é regravado; o `PAGO` de hoje fica `PAGO`. Tirar o `PAGO` da lista (`Selecionavel: false`) fica para quando o dono
  quiser: é uma linha, mais trocar o status de exemplo nos testes (21 arquivos usam `"PAGO"`).
- `CONTAS` (transferência entre bancos) **fica como está, por decisão do dono (2026-11-02):** não é selecionável (só o servidor o grava, nas duas pontas da transferência), conta no saldo e fica fora do total. `CREDITO` também segue como está.
- Saldo, fechamento, fatura e conferência não mudam: o servidor já lê as regras do catálogo (`ContaNoSaldo`), não o nome.
- Sem troca em massa do passado (item 44 removido).
