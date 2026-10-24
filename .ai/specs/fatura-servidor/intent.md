# Intent: faturas do cartão calculadas e validadas no servidor (item 33, fatia a)

Item 33 do BACKLOG, **fatiado**: esta é a fatia **a** (fatura, total e conferência no servidor). A fatia **b** (o cartão como entidade
própria, a trava da fatura conferida e a migração do `faturaAjuste`) fica para depois. Este documento diz **o quê e por quê**; o
como (`plan.md`) e o roteiro verificável (`spec.md`) vêm junto, para o dono aprovar antes de qualquer código.

**Estado:** rascunho para aprovação (2026-10-24). Vem **depois** do item 40 ([`saldo-servidor`](../saldo-servidor/intent.md)), embora
não dependa dele, e **antes** do 41 ([`futuros-previsto`](../futuros-previsto/intent.md)).

## Por quê

A aba Cartão é calculada inteira no navegador (`web/js/app.js:2725-3240`): o período de cada fatura, a que fatura cada compra
pertence, o ajuste de fatura, o total, a conferência com o valor do app do cartão e até a busca das compras que explicam uma
diferença. O servidor só guarda `CFG#CARTOES` como o front manda. Isso tem os mesmos problemas do saldo:

1. **O navegador manda o número como verdade.** `conferida.total` é calculado no front e gravado em `CFG#CARTOES` por
   `PUT /cfg/cartoes`, sem conferência: qualquer valor serve.
2. **A regra é por nome de status.** `ehCredito` (`startsWith('CREDITO')`), `ehCreditoEX`/`IN`, `ajustePadrao` e `ehStatusReal` decidem
   o que é compra de cartão e para que fatura ela vai. Enquanto isso estiver no front, o status não pode sair dele (item 41).
3. **O front descobre sozinho quais bancos são cartão**, olhando se há lançamento de crédito (`app.js:3065`), e carrega a configuração
   de cartões só quando se abre a aba, então a linha de fatura do Lançar não sabe se o banco tem cartão.

## O que queremos

1. **O servidor calcula as faturas:** o período (fechamento escolhido por mês ou o padrão do cartão), a fatura de cada compra, o ajuste
   efetivo (`faturaAjuste`, com o padrão dos registros antigos `CREDITO EX`/`IN`), o total, o estado (aberta ou fechada, histórico) e a
   lista de compras de cada fatura, tudo pronto.
2. **O servidor valida a conferência:** o total conferido é o que **ele** calculou; "mudou desde a conferência" também vem pronto.
3. **As compras que explicam uma diferença** (a busca perto do corte) passam a ser calculadas no servidor e chegam com a ação a
   aplicar. O front só mostra e, ao tocar, manda o `faturaAjuste` escolhido.
4. **A configuração do cartão vira comandos do servidor** (adicionar, remover, início do controle, fechamento e valor do banco de
   cada fatura, conferir e desfazer), validados, no lugar de o front regravar `CFG#CARTOES` inteiro.
5. **O servidor manda a lista de meios pronta com a marca de cartão:** `meios: [{ nome, cartao, ativo }]` em `GET /cfg/listas`. A linha "Fatura" do
   Lançar aparece pelo meio escolhido (`cartao: true`), sem olhar status e sem chamada extra. **Meio de pagamento** é o novo nome, só de exibição, do
   `banco` gravado (um de-para em `domain/`; o Dynamo não muda).
6. **"Registrar a diferença como lançamento"** vira um comando do servidor (ele escolhe o status e monta o lançamento).
7. **O servidor rejeita número derivado vindo do cliente** (`conferida`, total, `valorBanco` negativo, `faturaAjuste` fora de
   `-1`, `0`, `+1`). Um teste tenta forjar cada um.
8. **O resultado é o mesmo de hoje.** Mesmas regras, mesmo arredondamento, mesmos totais. Antes de ligar a tela nova, a aba Cartão
   antiga e a nova são comparadas lado a lado no dev e na produção.

## Limites (o que fica de fora, fatia b e outros itens)

- **O cartão como entidade própria** (hoje `porBanco[banco]`: o cartão não é um banco, pode não ter nenhum, e quem paga a fatura é
  uma escolha). Mexe em modelo de dados e na migração, então só entra com a aprovação do dono, em outra rodada.
- **A trava da fatura conferida pelo backend, com forma de reabrir** (decisão de 2026-10-10): muda o comportamento (hoje a fatura
  conferida só avisa que mudou) e precisa de demo. Fica na fatia b.
- **A migração que grava `faturaAjuste` explícito** nos lançamentos antigos sem o campo (para o nome do status deixar de ser
  consultado). Mexe em dado: só depois de discutir.
- **Pagar a fatura** (valor do boleto, banco pagador, vínculo) é o item 35.
- A visibilidade por cartão e por pessoa é o item 27. O status continua sendo tratado como o item 41 definir.

## Custo

- Cada tela da fatura é uma chamada ao servidor (hoje o navegador já tem os lançamentos carregados, e navegar entre faturas é
  instantâneo). O servidor lê os lançamentos do banco no período da fatura (de duas faturas antes a uma depois) a cada abertura.
- O Lançar passa a depender de `meios[].cartao` vindo de `GET /cfg/listas`, que já é lido na partida.
- `PUT /cfg/cartoes` some: quem usava o app antigo precisa do front novo (o release só vai depois da fase B).

## Pronto quando

- O servidor devolve as faturas, os totais, as compras de cada fatura e as sugestões de ajuste corretos (conferidos contra a tela
  antiga), valida a conferência e recusa o que o cliente tentar impor, com testes que forjam cada campo e falham antes da mudança.
- A aba Cartão usa só a API nova; saem do front `faturaBase`, `fechamentoDe`, `periodoFatura`, `faturaAtual`, `registrosCartao`,
  `totalFatura`, `ajusteEfetivo`, `ajustePadrao`, `candidatosFatura`, `ehCredito`, `ehCreditoEX`, `ehCreditoIN` e
  `atualizarConfigCartoes`.
- `PUT /cfg/cartoes` não existe mais; `security.md`, `arquitetura.md`, `modelo-de-dados` e o BACKLOG ficam em dia na mesma entrega.
