# Plan: novas visões dos gastos na Tendências (item 55)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`web/js/tendencias.js`, `web/consulta-tags.js`, `web/api.js`, `backend/service/consulta.go`). O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** rascunho para aprovação (2026-11-01). Este plan **não cria chave nem campo no DynamoDB**; o único dado novo proposto (`limite`, na fase E) está em "Propostas que o dono pode contestar" e só
se discute depois de as fases A a C estarem aprovadas.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| A Tendências soma no navegador os lançamentos que o app já carregou (`lancamentosCache`, até 1000) e nunca grava | `tendencias.js`, `CT.base`, `CT.resumir` | As visões novas seguem o mesmo princípio: calculam para **exibir**, sem enviar nada |
| `CT.base` já deixa de fora oculto e `previsto` e devolve `{mes, valor, tags, entrada, ref}`, com `ref` o lançamento inteiro | `consulta-tags.js` | Cada visão parte de `base` e do predicado da consulta (`q.pred`), então o filtro é um só e o total bate com o das barras |
| O lançamento traz `recorrente: { intervalo }` quando marcado (item 51, R13), `status`, `banco`, `dataEvento` (com hora) | `GET /tx` | Fixos e Pagamento só leem campos que já chegam; nenhum campo novo no lançamento |
| `visao` ('gastos', 'receitas', 'saldo') já é guardada na consulta salva; o servidor valida só o formato | `service/consulta.go`, item 53 | A visão **nova** (Ver como) é só da tela, não é guardada na consulta (proposta 2); o "Comparar com" pode ser (proposta 3) |
| Cada tela lê só o que é dela; a Tendências lê `/consultas` só quando a janela abre, e `carga.test.js` prova isso | guardrails, `carga.test.js` | Metas precisa ler `/consultas` ao abrir a visão: o guardrail e o teste ganham esse caso |
| A Tendências tem o aviso de que o mês mais antigo pode estar incompleto (`CT.corte`) | `consulta-tags.js` | Vale para todas as visões; a comparação com mês cortado vira "sem base" |
| Janela ou tela cheia nova entra em `camadasAbertas()` e `fecharCamadaTopo()` | `app.js`, `voltar.js` | A janela do **Ver como** (celular) entra nos dois |
| O front não trata status por nome; o servidor manda lista, rótulo e cor | guardrails, item 41 | A visão Pagamento agrupa pelo texto devolvido e usa o que o servidor mandar; sem lista nem cor escrita no front |

## Desenho

- **Módulo novo `web/visoes.js`**, lógica pura e testada (como `consulta-tags.js`): recebe `r` (o resultado de `CT.resumir`, que já traz os itens da consulta) e o mês escolhido, e devolve os dados de cada
  visão (`calendario`, `tags`, `ritmo`, `fixos`, `pagamento`, `metas`). O `tendencias.js` só desenha. Nenhum número daqui vai para a API.
- **Seletor "Ver como"** no resultado da Tendências, abaixo do Gastos | Receitas | Resultado. No celular é **um botão com a visão atual** que abre uma **janela de baixo** com as sete opções (cada uma com um
  desenho pequeno e o nome); no computador são **fichas em linha**. Com Resultado escolhido, só Meses fica ativa e as outras mostram o motivo.
- **Mês de referência:** as visões novas olham **um mês** (padrão: o mês atual, pela data local), com `‹ ›` entre os meses que existem nos lançamentos carregados. Meses não muda: continua mostrando todos.
- **Comparar com** (Tags e Ritmo): fichas "Mês anterior | Média de 3 meses | Mesmo mês do ano passado" (três opções, então fichas e não janela). Padrão: Mês anterior. Se a base cai antes do corte dos 1000
  lançamentos ou não tem dado, mostra "sem base", nunca um percentual.
- **Calendário:** semanas nas colunas, dia da semana nas linhas; cinco faixas de cor **pelo máximo do próprio mês** (0, até 25%, 50%, 75%, 100% do maior dia), então vale para qualquer tamanho de gasto; a
  legenda mostra os valores de corte do mês. Cada célula mostra o dia e o valor; tocar abre o valor completo.
- **Tags:** as tags dos lançamentos da consulta no mês, ordenadas pelo valor, no máximo 8 (o resto soma em "outras"); um lançamento com várias tags conta em cada uma (a mesma regra de "Outras tags"). Cada linha
  mostra valor e variação contra a base escolhida.
- **Ritmo:** acumulado por dia do mês atual até hoje, a projeção tracejada (`total até hoje ÷ dias passados × dias do mês`) e o mês-base inteiro por baixo. Mês já terminado: só a curva completa contra a base,
  sem projeção. Menos de 3 dias passados: sem projeção, com o motivo.
- **Fixos:** por mês (6 meses), o gasto da consulta com `recorrente` contra o resto, e o percentual de fixos. Nota fixa: "Conta como fixo o que foi marcado como recorrente; antes da marcação, o gasto aparece como variável."
- **Pagamento:** agrupa os itens da consulta pelo `status` **como texto opaco**, soma e ordena; nome e cor vêm do que o servidor mandar para o status (item 41). Até lá a fase não começa.
- **Metas:** cada consulta salva pode ter `limite` (valor mensal em reais). A visão mostra a consulta aberta contra o limite dela e, abaixo, as outras consultas salvas que têm limite (cada uma calculada no
  navegador com os lançamentos da tela). Estado em palavras e ícone: dentro (✓), perto a partir de 85% (!), passou (✕). Quem pode definir o limite é quem pode alterar a consulta (autor e dono, `consulta.editar`).

## Fases (cada uma é um PR; a demo cobre todas e é aprovada visão por visão)

- **A (tela, só front):** `web/visoes.js` com Calendário e Tags (com Comparar com), o seletor Ver como e a janela dele, o mês de referência. Sem backend.
- **B (tela, só front):** Ritmo.
- **C (tela, só front):** Fixos e variáveis.
- **D (tela e backend do item 41):** Pagamento. Só depois de o item 41 entregar status com rótulo e cor no servidor.
- **E (dado novo):** Metas. Antes: o dono aprova o campo `limite` (e se o "Comparar com" também é guardado). Backend (validação e `authz`), depois a tela.

O release segue por fase: cada uma é útil sozinha e não depende da próxima.

## Propostas que o dono pode contestar

1. **"Ver como" separado de Gastos | Receitas | Resultado:** um escolhe **o que somar**, o outro **como desenhar**. Resultado só em Meses.
2. **"Ver como" não é guardado na consulta:** a consulta guarda o critério (tags e visão de soma); o desenho é escolha de quem está olhando. Reabrir uma consulta salva abre em Meses.
3. **"Comparar com" é guardado na consulta?** Proposta: **não** nas fases A a D (volta a Mês anterior). Guardar mudaria o formato do item da consulta e é decisão de dado, junto com o `limite`, na fase E.
4. **Calendário pelas faixas do próprio mês** (e não por valores fixos em reais), para o mapa de calor valer para quem gasta R$ 500 ou R$ 50.000.
5. **Ritmo sem Futuros:** a projeção usa só o ritmo dos dias passados. Somar os previstos exige ler `/futuros`, e a Tendências não lê Futuros hoje.
6. **Fixos = marcado como recorrente**, sem inferir ("parece fixo porque se repete todo mês"). Inferir seria o front decidindo.
7. **Pagamento espera o item 41** em vez de agrupar por `PAGO` e `CREDITO` hoje.
8. **`limite` dentro do item da consulta** (`CONSULTA#<id>`), em reais, opcional; sem item novo e sem chave nova. Autor e dono alteram, como o resto da consulta. **Só depois de aprovado.**
9. **Metas lê `/consultas` quando a visão abre** (e não na aba), com o ajuste no guardrail e no `carga.test.js`.

## Riscos

| Risco | Resposta |
|---|---|
| Mês cortado pelos 1000 lançamentos gera percentual falso | A base antes do corte vira "sem base"; o aviso de mês incompleto aparece em todas as visões |
| Projeção do Ritmo enganar nos primeiros dias | Menos de 3 dias passados: sem projeção; o rótulo diz "Simulação" e o método (ritmo médio dos dias passados) |
| Cores do calendário ou do pagamento ilegíveis no tema escuro ou para daltônicos | Escala de uma cor só no calendário, com o valor escrito na célula; Pagamento com nome e percentual em texto, a cor só reforça. Validar nos dois temas |
| Fixos parecer baixo no passado | Nota fixa na visão explicando a marcação; nada é migrado |
| Visão virar tela pesada no celular | O desenho é SVG com poucos elementos (31 células, 8 linhas, 31 pontos); medir o pior caso antes do PR |
| Metas virar fonte de verdade | O limite é digitado pelo dono; o gasto é sempre recalculado na tela, nunca gravado |
