# Spec: trilha A (tirar a duplicação sem mudar o que se vê)

Roteiro verificável do [`intent.md`](intent.md) e do [`plan.md`](plan.md), sobre o [`inventario.md`](inventario.md). **A trilha B (redesenho) não tem spec**: só será escrita depois das respostas do dono e de uma
demo que cubra o inventário.

**Estado:** rascunho, **sem aprovação do dono** (2026-11-03).

## Regras

- **A-R1.** **Nada visível muda.** Para cada tela e cada modo do inventário (seção 2), o HTML que a função devolve depois da fase é **idêntico** ao de antes (arquivos de referência da A0).
- **A-R2.** **Nenhuma chamada muda.** O corpo e a rota que cada gesto envia ao servidor (seção 3 do inventário) são idênticos aos de hoje: `POST /tx` de Prever, Estender e Lote; `POST /recorrencias`; `PUT` do aviso
  da fatura; `PUT` do aviso padrão do cartão.
- **A-R3.** Só **uma** função cria a janela cheia (`#tc`): `montarTc`. Prever, Lote, Recorrência, Fatura e Aviso padrão passam por ela.
- **A-R4.** Só **uma** função monta o estado inicial do aviso (`{on, dias, insistir}`) a partir de um aviso salvo, com os mesmos padrões de hoje (`[1,0]` e "insistir" ligado quando não há aviso salvo).
- **A-R5.** Não existe mais o ramo morto `avisoVisivel.pv` nem o prefixo `pv` em `htmlAviso`. O formulário do Lançar (`f`) continua como está.
- **A-R6.** A tela **Repetir** é uma função com parâmetros (tem quantidade, tem interruptor, texto da quantidade, texto da nota); a tela **Avisar** recebe entradas explícitas e **não lê** `pv` nem outro
  estado global, nem decide por modo.
- **A-R7.** O botão voltar do celular conta as mesmas camadas de hoje: casca, subtela (Repetir, Avisar, Mês de destino, Status) e janela de rodas.
- **A-R8.** O teste de unicidade (`web/telas-unicas.test.js`) falha se (a) outra função além de `montarTc` criar o elemento `#tc`, (b) o título "Avisar no Telegram" ou a linha "Repetir quantas vezes" for
  montado em mais de um lugar de `web/js/`, (c) as funções das telas Repetir e Avisar referirem `pv`.
- **A-R9.** Nenhuma rota, campo ou chave nova; nenhum arquivo do `backend/` muda. O Lançar e o Cartão não mudam de aparência nem de comportamento.
- **A-R10.** Cada tela lê só o que é dela: abrir Prever, Estender, Lote ou Recorrência lê o mesmo que hoje (`lerDadosParaDuplicados` onde já lê) e a subida do app continua sem ler lançamentos (`carga.test.js`).

## Testes

| # | Cenário | Esperado |
|---|---|---|
| A-T1 | HTML da tela principal, de Repetir e de Avisar em cada um dos 7 modos, com relógio fixo | igual ao arquivo de referência (A-R1) |
| A-T2 | Mês de destino e card de duplicados do Lote; numeração de parcelas do Prever e do Estender | igual ao de referência |
| A-T3 | Avisar com 0, 1, 6 prazos (limite) e com "Insistir" ligado e desligado; erro de "prazo já existe" | igual ao de referência |
| A-T4 | Corpo e rota das chamadas de Prever, Estender, Lote, Tornar recorrente, aviso da fatura e aviso padrão, com um `db` de teste | iguais aos de referência (A-R2) |
| A-T5 | `avisoInicial` com aviso salvo, sem aviso e com `dias` ausente | `{on, dias, insistir}` iguais aos das leituras de hoje |
| A-T6 | `camadasAbertas()` e `fecharCamadaTopo()` com a casca, a subtela e a janela de rodas abertas | 1, 2 e 3 camadas; fecha uma por vez |
| A-T7 | `web/telas-unicas.test.js` no código novo | passa |
| A-T8 | Reintroduzir de propósito um segundo `createElement` da janela ou um segundo título "Avisar no Telegram" | A-T7 **falha** (quebrar o código de propósito, como pede o `guardrails.md`) |
| A-T9 | `sem-status.test.js`, `carga.test.js`, `recorrencia.test.js`, `consultas-camadas.test.js`, `voltar.test.js` | continuam verdes |
| A-T10 | Conferência lado a lado no `dev`: cada modo, no celular (≈ 390 px) e na janela larga (≥ 1280 px), nos dois temas | sem diferença visível |

## Telas

Nenhuma demo nova na trilha A: a referência é o app de hoje e as demos `-v1` (`demo-prever-v1`, `demo-repetir-v1`, `demo-avisar-v1`). Se algum PR da trilha A precisar mudar um pixel, ele **para** e vira
pergunta ao dono (a mudança pertence à trilha B).
