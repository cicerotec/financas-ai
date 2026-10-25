# Intent: pagar a fatura do cartão (itens 33b e 35)

Itens 33 (fatia b) e 35 do BACKLOG, juntos. Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`)
vêm junto, para o dono aprovar antes de qualquer código.

**Estado:** rascunho para aprovação (2026-10-25). Escopo fechado com o dono em 2026-10-25 (veja "O que foi decidido"). Vem depois dos itens
40, 33a e 41, que estão concluídos.

## Por quê

Hoje o sistema sabe o **total da fatura** (item 33a), mas não sabe **como ela será paga**:

1. **Quem manda é a operadora.** O extrato dela pode somar 3.000 e o boleto vir de 2.500. O pagamento nasce **da fatura**, não da soma dos
   lançamentos do sistema, e o valor pago precisa poder ser diferente do calculado.
2. **Não há como dizer por onde a fatura será paga.** O cartão Santander pode ser pago pelo Nubank, e ninguém sabe de antemão onde haverá
   saldo; a escolha precisa poder mudar a cada fatura.
3. **A fatura não aparece em Futuros.** Para saber quanto vai sair da conta (e quando), hoje só olhando a aba Cartão. A fatura aberta
   (que ainda cresce a cada compra) e a fechada (que a operadora já cobrou) são dinheiro a pagar, com data.
4. **O pagamento não fica ligado à fatura.** Quem paga hoje cria um lançamento à mão no banco; o sistema não sabe que aquela saída é o
   pagamento daquela fatura, e o gasto parece contar duas vezes (nas compras de crédito e no pagamento).
5. **A fatura conferida só avisa que mudou.** A decisão de 2026-10-10 é que ela seja **travada pelo servidor**, como o mês conferido de um banco,
   com forma de reabrir. Hoje o aviso "mudou desde a conferência" aparece depois do estrago.

## O que foi decidido (dono, 2026-10-25)

- **O cartão continua sendo uma entrada de `CFG#CARTOES`**, identificado pelo nome do meio (`meios[].cartao`). **Não vira entidade própria, não ganha
  id, e os lançamentos continuam apontando para ele pelo `banco`.** O que o dono quer cabe nessa entrada.
- **Não há migração do passado.** O passado está correto. Os lançamentos antigos ficam como estão; quem quiser ajustar um lança o `faturaAjuste`
  nele. O servidor continua conhecendo o padrão de `CREDITO IN` e `CREDITO EX` (só no código dele).
- **Dados novos aprovados:** `pagador` (padrão no cartão e troca por fatura) e vencimento (dia padrão no cartão e data por fatura), todos dentro da
  entrada do cartão em `CFG#CARTOES`, sem item nem chave nova. Um campo `pagaFatura` nos lançamentos de pagamento (veja o spec) aguarda a aprovação
  no spec.
- **A fatura conferida passa a ser travada** (com "Desfazer conferência" para reabrir).
- **Demo antes** da linha de fatura em Futuros e da seção do cartão.

## O que queremos

1. **Meio pagador:** na aba Cartão, na seção do cartão, o dono escolhe o meio com que paga (padrão do cartão) e pode trocar a cada fatura.
2. **Vencimento:** o dono informa o dia do vencimento do cartão e pode acertar a data de uma fatura.
3. **Futuros mostra as faturas**, calculadas pelo servidor (não são lançamentos gravados): a **fechada**, esperando pagamento, com o valor que a
   operadora cobra, e a **aberta**, com o total até agora. Ficam no meio pagador, na data do vencimento; passado o vencimento sem pagar, ficam
   como **atrasadas**.
4. **Pagar a fatura** pelo mesmo gesto de confirmar de Futuros ("Já paguei"): o dono confirma ou corrige o valor (o do boleto), a data e o status.
   O servidor cria o lançamento de pagamento no meio pagador, **ligado à fatura**, e a linha some de Futuros.
5. **O pagamento mexe no saldo do meio pagador e não conta de novo como gasto**: o gasto já contou nos lançamentos de crédito.
6. **A diferença entre o valor pago e o calculado fica à vista** na fatura (hoje isso virava um lançamento de valor zero, uma nota solta).
7. **A fatura conferida é travada pelo servidor.** Mudar valor, data, meio ou ajuste de uma compra dela, criar ou excluir uma compra no período dela,
   é recusado até a conferência ser desfeita.

## Limites (o que fica de fora)

- **Cartão como entidade própria, id do cartão e migração do `faturaAjuste`:** fora (decisão do dono).
- **Pagamento mínimo ou parcial** (vários pagamentos numa fatura): fora. Uma fatura tem **um** pagamento; um segundo é recusado até o primeiro ser excluído.
- **Membro (não dono):** só o dono configura o cartão e paga a fatura, como hoje.
- **Juros do rotativo e fatura parcelada:** fora.
- **Item 45** (atrasados antigos de Futuros por ano): continua separado; as faturas respeitam a mesma janela de Futuros (mês passado em diante).

## Custo

- Cada abertura de Futuros passa a ler também a janela de compras de cada cartão (o mesmo que a aba Cartão já lê) e os pagamentos recentes.
  O resultado entra no mesmo cache do contador de alterações que Futuros já usa.
- Toda escrita de lançamento de crédito num cartão com fatura conferida lê `CFG#CARTOES` (uma leitura de 1 item por chamada).
- Dois ou três campos novos em `CFG#CARTOES` e um atributo opcional nos lançamentos de pagamento. Nenhum item, chave ou índice novo.

## Pronto quando

- A fatura aparece em Futuros no meio pagador, na data certa, com o valor certo, e some ao pagar.
- O pagamento fica ligado à fatura, mexe no saldo e não entra de novo nos gastos.
- A fatura conferida recusa as escritas que a mudariam, com testes que tentam cada uma e que falham antes da mudança.
- `security.md`, `modelo-de-dados`, `arquitetura.md` e o BACKLOG ficam em dia na mesma entrega.
