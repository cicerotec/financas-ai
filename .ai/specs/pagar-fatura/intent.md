# Intent: pagar a fatura do cartão (itens 33b e 35)

Itens 33 (fatia b) e 35 do BACKLOG, juntos. Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`)
vêm junto, para o dono aprovar antes de qualquer código.

**Estado:** rascunho para aprovação (2026-10-25), revisto no mesmo dia depois da conversa com o dono sobre o que é o pagamento. Vem depois dos itens 40,
33a e 41, que estão concluídos.

## A ideia central

**O pagamento da fatura é um lançamento normal**, um evento como pagar a conta de água ou de luz. Ele pode ter qualquer valor (o total, o mínimo, o que a
operadora permitir), sair de qualquer meio, ser feito em mais de um lançamento e em mais de um meio. **Quem controla a fatura é a operadora**: se ela
cobrar juros, restante ou qualquer coisa no extrato, o dono lança como um registro qualquer. O sistema não carrega saldo nem restante de uma fatura
para a outra.

O que se acrescenta é uma **marca** (uma *label*, como as do GitHub): o dono marca um lançamento dizendo "este lançamento pagou a fatura de set/26 do cartão X".
A marca **não decide nada**: não recusa, não trava, não muda o saldo. Ela serve para **achar depois** (na aba Cartão, na fatura; e no Histórico, no
lançamento) e para a fatura **sair de Futuros** quando já foi paga.

## Por quê

1. **Não há como dizer por onde a fatura será paga.** O cartão Santander pode ser pago pelo Nubank, e ninguém sabe de antemão onde haverá saldo; a
   escolha muda a cada fatura.
2. **A fatura não aparece em Futuros.** Para saber quanto vai sair da conta (e quando), hoje só olhando a aba Cartão. A fatura aberta (que ainda cresce a
   cada compra) e a fechada (que a operadora já cobrou) são dinheiro a pagar, com data.
3. **O pagamento não fica à vista na fatura.** Quem paga cria um lançamento à mão e depois não acha qual saída pagou qual fatura.
4. **A fatura conferida só avisa que mudou.** A decisão de 2026-10-10 é que ela seja **travada pelo servidor**, como o mês conferido de um banco, com
   forma de reabrir.

## O que foi decidido (dono, 2026-10-25)

- **O cartão continua sendo uma entrada de `CFG#CARTOES`**, identificado pelo nome do meio (`meios[].cartao`). Não vira entidade própria, não ganha id, e os
  lançamentos continuam apontando para ele pelo `banco`.
- **Não há migração do passado.** O passado está correto.
- **Dados novos aprovados:** `pagador` (padrão no cartão e troca por fatura) e vencimento (dia padrão no cartão e data por fatura), dentro da entrada do
  cartão em `CFG#CARTOES`. **A marca** é um atributo opcional do lançamento, `pagaFatura` (o cartão e o mês), que o dono escolhe.
- **O pagamento é um evento normal, a marca é só uma label** (a ideia central acima).
- **A fatura conferida passa a ser travada**, com "Desfazer conferência" para reabrir.
- **Demo antes** da linha de fatura em Futuros, da seção do cartão e da marca no lançamento.

## O que queremos

1. **Meio pagador:** na aba Cartão, na seção do cartão, o dono escolhe o meio com que costuma pagar (padrão do cartão) e pode trocar a cada fatura. É a
   **sugestão** da linha de Futuros e da janela de pagar; o dono paga com o meio que quiser.
2. **Vencimento:** o dono informa o dia do vencimento do cartão e pode acertar a data de uma fatura.
3. **Futuros mostra as faturas**, calculadas pelo servidor (não são lançamentos gravados): a **fechada**, com o valor que a operadora cobra, e a **aberta**, com o
   total até agora. Ficam no meio pagador sugerido, na data do vencimento; passado o vencimento sem pagar, ficam como **atrasadas**.
4. **"Já paguei" na linha da fatura** abre a janela de confirmar (valor, data, status e o meio); o lançamento de pagamento sai **já marcado** com a fatura.
5. **A marca no Lançar:** uma linha opcional, "Paga a fatura", para marcar qualquer lançamento (inclusive o segundo meio lançado à mão), na criação e na edição.
6. **A marca aparece onde ajuda a achar:** uma etiqueta no lançamento (Histórico) e, na aba Cartão, a seção "Pagamentos desta fatura" com a lista e a soma.
7. **A fatura sai de Futuros** quando existe pelo menos um lançamento marcado com ela (só para a lista não repetir o que já foi pago).
8. **A fatura conferida é travada pelo servidor.** Mudar valor, data, meio ou ajuste de uma compra dela, criar ou excluir uma compra no período dela, é
   recusado até a conferência ser desfeita.

## Limites (o que fica de fora)

- **Cartão como entidade própria, id do cartão e migração do `faturaAjuste`:** fora (decisão do dono).
- **Saldo anterior, restante, rotativo, juros e pagamento mínimo como regra:** fora. O pagamento mínimo é uma oferta da operadora: o dono paga o valor que
  quiser e lança o que ela cobrar depois.
- **Qualquer regra que a marca imponha** (recusar segundo pagamento, exigir valor, travar): fora. A marca só etiqueta.
- **Membro (não dono):** só o dono configura o cartão. A marca num lançamento vale para quem pode editar o lançamento.
- **Item 45** (atrasados antigos de Futuros por ano): continua separado; as faturas respeitam a mesma janela de Futuros (mês passado em diante).

## Custo

- Cada abertura de Futuros passa a ler também a janela de compras de cada cartão (o mesmo que a aba Cartão já lê) e os lançamentos marcados das faturas em
  aberto. O resultado entra no mesmo cache do contador de alterações que Futuros já usa.
- Toda escrita de lançamento de crédito num cartão com fatura conferida lê `CFG#CARTOES` (uma leitura de 1 item por chamada).
- Alguns campos novos em `CFG#CARTOES` e um atributo opcional nos lançamentos. Nenhum item, chave ou índice novo.

## Pronto quando

- A fatura aparece em Futuros no meio pagador sugerido, na data certa, com o valor certo, e some quando há um pagamento marcado.
- O dono marca um lançamento como pagamento de uma fatura e o encontra no Histórico e na aba Cartão.
- A fatura conferida recusa as escritas que a mudariam, com testes que tentam cada uma e que falham antes da mudança.
- `security.md`, `modelo-de-dados`, `arquitetura.md` e o BACKLOG ficam em dia na mesma entrega.

## Para o dono confirmar

- **Pagamento marcado sai do total:** o servidor marca `excluirDoTotal` num lançamento que leva a marca, para o pagamento não contar de novo como gasto
  (o gasto já contou nas compras de crédito). É o único efeito da marca; se preferir que a marca não faça nem isso, é só dizer.
