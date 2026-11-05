# Intent: parar a duplicação das telas de Repetir, Recorrente e Avisar

Itens 47 e 42 do BACKLOG. Este documento diz **o quê e por quê**; o levantamento de tudo o que existe está no [`inventario.md`](inventario.md), o como no [`plan.md`](plan.md) e o roteiro
verificável no [`spec.md`](spec.md).

**Estado:** refeito em 2026-11-03, **ainda sem aprovação do dono**. Substitui o primeiro roteiro (PR #353, fechado), que decidia o redesenho antes de levantar as telas. Nenhuma decisão de tela
está tomada aqui.

## Por quê

O mesmo assunto está desenhado várias vezes e cada cópia foi crescendo sozinha (detalhes e números de linha no [`inventario.md`](inventario.md)):

- **Treze pontos de entrada** (E1 a E13) abrem Repetir, Recorrente ou Avisar, em Lançar, Histórico, Futuros e Cartão.
- A janela cheia e o estado global `pv` servem a **sete modos** (Prever, Estender, Lote, Recorrência pelo histórico, Recorrência pelo Lançar, Fatura e Aviso padrão); o **Repetir tem três versões**
  e o **Avisar** é uma função com um ramo de texto por modo.
- O **Lançar tem um segundo desenho** do aviso e do repetir (`htmlAviso`, `htmlRepeticao`), escondido por uma regra de data futura que o servidor decide de outro jeito (o `previsto`).
- **Nada impede a próxima cópia:** não há teste que falhe quando uma tela é copiada, e nenhum teste cobre o HTML dessas telas.
- Existem **dez premissas escondidas** (P1 a P10 do inventário), regras que só funcionam por causa de outra. A do aviso do Recorrente criado pelo Lançar foi a que apareceu primeiro.

## O que queremos

Dois objetivos **separados**, que não devem andar juntos:

1. **Trilha A, tirar a duplicação sem mudar nada que o dono vê ou usa.** Mesmo visual, mesmas regras, mesmas chamadas ao servidor. Só estrutura: uma casca, uma tela Repetir parametrizada, uma tela Avisar
   com entradas explícitas, um estado inicial do aviso, e um **teste que impede** a duplicação de voltar. Prova: o HTML de cada tela e de cada modo sai **idêntico** ao de hoje.
2. **Trilha B, redesenhar o que o dono vê** (por exemplo, um seletor único no lugar dos botões Prever e Recorrente, e o Lançar sem a regra de data futura). É mudança de produto: só começa depois que o dono
   decidir as perguntas do `plan.md`, com **uma demo que cobre todas as linhas do inventário**.

A trilha A pode ser feita sem a B, e deixa a B mais barata e mais segura. A B não deve começar antes da A.

## Limites (o que fica de fora)

- **Nenhum campo, chave ou rota nova** no DynamoDB ou na API, em nenhuma das trilhas. O servidor não muda.
- As três demos de 2026-11-03 (`demo-repetir-v2-seletor`, `demo-lancar-v3-repetir`, `demo-historico-v2-repetir`) são **exploração da trilha B**. Elas **não cobrem** Estender, Parar recorrência, lote,
  edição, cópia nem o Cartão, e **não valem como padrão aprovado**.
- O **filtro de tags, combos e busca** (item 42) é outro componente: este roteiro só aproveita o método (inventário antes da demo).
- Não muda o que o servidor faz com datas, `previsto`, séries e recorrência.
