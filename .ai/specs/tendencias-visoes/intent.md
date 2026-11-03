# Intent: novas visões dos gastos na Tendências (item 55)

Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`) vêm junto, para o dono aprovar antes de qualquer código.

**Estado:** implementado em 2026-11-02 (texto original de 2026-11-01; as diferenças estão no [`spec.md`](spec.md)). Saiu da conversa com o dono em 2026-11-01, depois de ele ver onze gráficos de exemplo com dados inventados (a página de exploração não vai para o repositório). Ele
escolheu seis: **Calendário de calor, Metas por consulta, Forma de pagamento, Maiores tags do mês, Fixos e variáveis e Ritmo do mês**. O que ainda é proposta está marcado em "Propostas que o dono pode contestar" no [`plan.md`](plan.md).

## Por quê

A Tendências mostra hoje **uma coisa só**: o gasto (ou a receita, ou o resultado) por mês das tags da consulta, em barras. Com as consultas salvas (item 53), o dono monta a consulta uma vez e a reutiliza,
mas continua olhando sempre o mesmo desenho. Ele quer enxergar os gastos de outros ângulos: **quando** o dinheiro sai (dias do mês), **quanto já nasce comprometido** (recorrentes), **onde o mês vai
terminar** no ritmo de hoje, **quem pesa e quem mudou** (tags), **por onde pagou** (forma de pagamento) e **se está dentro do que combinou consigo** (metas).

## O que queremos

1. **Uma consulta, várias visões.** A consulta continua sendo o filtro (tags, operadores, Gastos | Receitas | Resultado). As visões desenham essa consulta de outros ângulos, em **seções que recolhem pelo título** (Comparar, Ao longo do mês, Para onde foi), sem seletor "Ver como" (a demo com o seletor foi reprovada): Ritmo,
   Calendário, Fixos, Tags, Pagamento e Metas, com o mês de referência e o "Comparar com" no topo.
2. **Calendário de calor.** O mês escolhido em dias (semanas nas colunas, dias da semana nas linhas), cada dia pintado pelo quanto saiu.
3. **Maiores tags do mês** com a **variação contra uma base à escolha**: mês anterior (padrão), média dos 3 meses anteriores ou o mesmo mês do ano passado.
4. **Ritmo do mês.** A curva acumulada do mês contra a do mês anterior e uma **projeção** do fechamento pelo ritmo dos dias que já passaram.
5. **Fixos e variáveis.** Por mês, quanto do gasto da consulta é de lançamentos **marcados como recorrentes** e quanto é o resto, com o percentual de comprometido.
6. **Forma de pagamento.** Quanto do gasto da consulta passou por cada forma (crédito, Pix, débito...), com a cor e o nome que o app já usa.
7. **Metas por consulta.** Um **limite mensal** guardado na consulta salva, com a barra do quanto já foi gasto e o estado em palavras (dentro, perto, passou).
8. **Continua simulação.** Tudo é calculado no navegador com o que já está na tela, nada vira saldo e **nada é enviado ao servidor**, com **uma exceção**: o limite da meta (item 7), que é um valor que o dono
   digita, não um número derivado.

## Limites (o que fica de fora)

- **Nada de gráfico de pizza ou de dois eixos**, e nada que misture medidas de escalas diferentes no mesmo desenho.
- **Sem painel com todas as consultas lado a lado** (a ideia do "painel das consultas gravadas" ficou para depois): cada visão mostra a consulta aberta.
- **Sem leitura nova do servidor** em seis das sete visões: usam os 1000 lançamentos que a Tendências já carrega. A exceção é Metas, que lê `/consultas` quando a visão abre (ajuste no guardrail).
- **Ritmo não lê Futuros:** a projeção é só o ritmo dos dias que já passaram. Somar os previstos do mês fica para uma entrega própria (outra leitura, outra tela).
- **Sem migração:** o passado não é tocado. Só conta como fixo o lançamento marcado como recorrente (a marca existe desde o item 51); os meses anteriores à marcação aparecem como variáveis, e a tela diz isso.
- **Não decide status:** a visão Pagamento agrupa pelo texto que o servidor mandou e usa nome e cor que o servidor mandou; o front não conhece nome de status.
- **Sem a visão Resultado fora dos Meses:** com Resultado escolhido, só a visão Meses fica ativa (um calendário ou um acumulado de "renda menos gasto" não diz nada claro).

## Custo (vai na demo)

- **O limite de 1000 lançamentos continua valendo** e vale para todas as visões: o mês mais antigo pode estar incompleto, e a comparação com um mês cortado mostra "sem base" em vez de um percentual falso.
- **A Metas é a única que grava:** exige um campo novo no item da consulta (`limite`). Por isso é a **última fase** e só começa depois de o dono aprovar o campo.
- **A Pagamento depende do item 41** (catálogo de status e formas de pagamento no servidor). Antes dele existem só `PAGO` e `CREDITO`, e o gráfico seria pobre.

## Pronto quando

- A demo é aprovada pelo dono (visão por visão, no celular e no computador) antes de qualquer código.
- Cada visão mostra o mesmo total que a visão Meses para a mesma consulta e o mesmo período (conferência dos números), com testes em `web/visoes.test.js`.
- Os pontos de dado novo (`limite` na consulta, e o que a visão Pagamento pede do servidor) foram aprovados antes de existirem, e `modelo-de-dados`, `padrao-de-telas.md`, o guardrails e o BACKLOG ficam em dia na mesma entrega.
