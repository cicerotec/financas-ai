# Spec: tela Repetir única

Roteiro verificável do [`intent.md`](intent.md) e do [`plan.md`](plan.md).

**Estado:** rascunho para aprovação do dono (2026-11-03). Demos aprovadas: [`demo-repetir-v2-seletor`](../../demos/demo-repetir-v2-seletor.html),
[`demo-lancar-v3-repetir`](../../demos/demo-lancar-v3-repetir.html), [`demo-historico-v2-repetir`](../../demos/demo-historico-v2-repetir.html).

## Regras

- **R1.** Existe **uma** tela Repetir, aberta pelo Lançar (linha Repetir em "Mais") e pelo Histórico (botão Repetir). O seletor tem as posições **Só este · N vezes · Recorrente** no Lançar e **N vezes ·
  Recorrente** no Histórico e em Futuros (o lançamento já existe).
- **R2.** A linha **Avisar no Telegram** aparece nas três posições. Em **Só este** ela só liga se a data do lançamento for futura; com data de hoje fica desligada, com a dica
  "Só com data futura, ou escolha Repetir ou Recorrente". Em N vezes e Recorrente o aviso vale para as cópias. O formulário não esconde mais nenhum controle por data.
- **R3.** Lançar, o que cada posição grava:
  - **Só este:** um lançamento, como hoje. `aviso` no lançamento só se a data for futura.
  - **N vezes:** o lançamento mais **N cópias além dele**, ligadas por uma série (`serie: {id, indice: 1, intervalo}`), com as mesmas validações de hoje (`Recorrencia.validar`) e o aviso de duplicados.
    O `aviso` vai **nas cópias** quando ligado e no lançamento só se a data dele for futura.
  - **Recorrente:** o lançamento com `recorrente: {intervalo}` (o servidor cria a primeira cópia). Onde o aviso vale: ver Pendência 1 do `plan.md`.
- **R4.** Histórico e Futuros: **N vezes** cria as cópias como o Prever de hoje (mesmos campos, mesmo `criarEventos`, mesmo tratamento de falha no meio) e **Recorrente** chama a mesma rota que o Tornar
  recorrente de hoje, com o aviso escolhido na primeira cópia.
- **R5.** O **Cartão** abre só a tela Avisar, com **Salvar** no rodapé (aviso padrão do cartão e aviso de cada fatura); a seta só desiste; sem Repetir. O corpo enviado ao servidor não muda.
- **R6.** O front **não decide** se um lançamento é previsto: só monta as datas das cópias e envia; o servidor marca o `previsto`.
- **R7.** Nenhuma rota, campo ou chave nova. O formato do `aviso` (`{dias, insistir}`) e o limite de prazos (6) continuam os de `recorrencia.js`.
- **R8.** A tela (casca, subtela Intervalo ou Avisar e janela de rodas) conta em `camadasAbertas()` e fecha em `fecharCamadaTopo()` na ordem de cima para baixo.
- **R9.** Sem estado global que decida por modo: a seção Avisar e a seção Repetir recebem o que mostram e devolvem o que foi escolhido. Nenhum `if (pv.fatura)` ou `if (pv.rec)` dentro delas.
- **R10.** Ícones: Recorrente é o 🔁; **N vezes é a pilha**. O chip 3× e o calendário não entram.
- **R11.** "N vezes" no Lançar é **"além deste"** (igual ao Prever); a nota mostra as datas e o total.
- **R12.** Cada tela lê só o que é dela: abrir a tela Repetir **não lê** lançamentos além do que o Prever já lê para os duplicados (`lerDadosParaDuplicados`), e a subida do app continua sem lê-los (`carga.test.js`).

## Testes

| # | Cenário | Esperado |
|---|---|---|
| R-T1 | `planoDeRepeticao`, Só este com data de hoje e aviso ligado | lançamento sem `aviso`; nenhuma cópia |
| R-T2 | Só este, data futura, aviso ligado | lançamento com `aviso`; nenhuma cópia |
| R-T3 | N vezes (3, mensal), data de hoje, aviso ligado | lançamento sem `aviso` e com `serie.indice = 1`; 3 cópias com `aviso` e `indice` 2 a 4; datas mês a mês |
| R-T4 | N vezes com data futura | lançamento e cópias com `aviso` |
| R-T5 | N vezes, dia 31, mensal | datas no último dia do mês quando o dia não existe (28/02, 31/03) |
| R-T6 | N vezes acima do limite ou intervalo inválido | erro de `Recorrencia.validar`, nada criado |
| R-T7 | Recorrente | lançamento com `recorrente.intervalo` e sem `serie`; sem cópias no navegador |
| R-T8 | Posições do seletor por contexto | Lançar: 3; Histórico e Futuros: 2 |
| R-T9 | Estado inicial do aviso (`avisoInicial`) com aviso salvo, sem aviso e aviso incompleto | `{on, dias, insistir}` iguais aos das quatro leituras de hoje; incompleto devolve o erro de hoje |
| R-T10 | Resumo da linha do Lançar | "Não", "3× mensal · 🔔", "🔁 Mensal, sem fim" |
| R-T11 | Aviso do Cartão (padrão e de fatura) | mesmo corpo que o de hoje; Salvar chama a mesma rota |
| R-T12 | `camadasAbertas()` com a casca, a subtela e a janela de rodas abertas | 1, 2 e 3 camadas; `fecharCamadaTopo()` fecha uma por vez |
| R-T13 | Nenhum resto do desenho antigo em `web/` | sem `htmlAviso`, `htmlRepeticao`, `formRec`, `fRepetirWrap`, `fRecWrap` e `fAvisoMount` depois da fase 4 |
| R-T14 | `sem-status.test.js`, `carga.test.js`, `recorrencia.test.js`, `voltar.test.js` | continuam verdes |
| R-T15 | Quebrar o código de propósito (por exemplo, esconder o Avisar por data) | R-T1 ou R-T8 falha |

## Telas

- **Demos aprovadas (2026-11-03):** as três acima, partindo de `demo-prever-v1` / `demo-repetir-v1` / `demo-avisar-v1`, `demo-lancar-mais-v1` e `demo-acoes-historico-v1`.
- Cada fase confere a tela **igual à demo** (celular ≈ 390 px e janela ≥ 1280 px, nos dois temas) e mede o pior caso do `guardrails.md`: descrição e banco compridos, valor de cinco dígitos, 6 prazos de
  aviso, intervalo personalizado e lista de datas longa.
- Desvio da demo se **avisa antes do PR**. O que a demo não cobre (lote, Estender, ações de Futuros) só entra com demo ou confirmação do dono (Pendências 2 a 4 do `plan.md`).
