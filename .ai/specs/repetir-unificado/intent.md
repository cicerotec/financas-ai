# Intent: uma tela Repetir só (Repetir, Recorrente e Avisar)

Itens 47 e 42 do BACKLOG. Este documento diz **o quê e por quê**; o como está no [`plan.md`](plan.md) e o roteiro verificável no [`spec.md`](spec.md).

**Estado:** rascunho para aprovação do dono (2026-11-03). As demos já foram **aprovadas pelo dono em 2026-11-03**:
[`demo-repetir-v2-seletor`](../../demos/demo-repetir-v2-seletor.html), [`demo-lancar-v3-repetir`](../../demos/demo-lancar-v3-repetir.html) e
[`demo-historico-v2-repetir`](../../demos/demo-historico-v2-repetir.html) (PR #350). Nada foi implementado.

## Por quê

O mesmo assunto (repetir um lançamento e avisar no Telegram) está desenhado várias vezes em `web/js/app.js`, e o dono, ao usar o app, achou o Lançar "horrível" para isso:

- O aviso tem **dois desenhos**: a tela cheia `pvTelaAvisar` (Prever, lote, Recorrência, fatura e aviso padrão do cartão, cinco modos no mesmo estado global `pv`) e o bloco do Lançar
  (`htmlAviso`, caixa mais campos soltos).
- O Repetir tem **três versões** (`pvTelaRepetir`, `pvTelaRepetirLote` e `pvTelaRepetirRec`) e um quarto desenho no Lançar (`htmlRepeticao`).
- Quatro funções repetem a mesma abertura da janela (`abrirPrever`, `abrirLote`, `abrirAvisoFatura`, `abrirAvisoPadrao`), embora `montarTc` já exista.
- No Lançar, Repetir e Avisar **só aparecem com data futura** (`dataFuturaForm()`), então quem combina "pago hoje e mais duas parcelas nos próximos meses" não os acha. A mesma ideia de
  "é futuro" também é decidida pelo servidor (o `previsto`): dois critérios para o mesmo fato, contra a regra de que o front não decide.
- Recorrente é "repetir sem o repetir", e hoje são dois botões e duas telas.

## O que queremos

1. **Uma tela Repetir** com o seletor **Só este · N vezes · Recorrente** (um toque), a linha **Avisar no Telegram** dentro dela e as subtelas Intervalo e Avisar (demo aprovada).
2. **Lançar:** uma linha **Repetir** em "Mais", sempre visível; some o formulário solto de Repetir, Recorrente e Avisar.
3. **Histórico:** o botão **Repetir** (versão A) no lugar de Prever e Tornar recorrente; a tela abre em N vezes. O ícone de "N vezes" é a **pilha**; o Recorrente mantém o 🔁.
4. **Avisar continua sendo uma tela reaproveitável** que o Cartão abre sozinha, com o botão **Salvar** no rodapé (aviso padrão do cartão e aviso de cada fatura). O Cartão não ganha Repetir.
5. **Código:** uma casca (`montarTc`) e seções com entradas explícitas, sem o estado global `pv` decidindo por modo.

## Limites (o que fica de fora)

- **Nenhum campo nem chave nova no DynamoDB e nenhuma rota nova.** O backend não muda: N vezes continua criando as cópias pelo `POST /tx`, Recorrente continua pela rota que já existe e o aviso continua no
  mesmo formato (`{dias, insistir}`).
- O **filtro de tags, combos e busca** (item 42) é outro componente; este roteiro só usa o método (matriz de telas × diferenças antes da demo).
- **Não coberto pelas demos aprovadas** (precisa de demo ou decisão antes do código, ver o `plan.md`): o lote ("Tornar previstos"), o Estender a série, a numeração de parcelas, o aviso de duplicados
  e o menu de ações de Futuros.
- Não muda o que o servidor faz com datas, `previsto`, séries e recorrência.
