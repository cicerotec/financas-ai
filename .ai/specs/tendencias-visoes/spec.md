# Spec: novas visões dos gastos na Tendências (item 55)

Roteiro verificável do [`intent.md`](intent.md) e do [`plan.md`](plan.md).

**Estado:** implementado em 2026-11-02 (fases A a F do item 55). O que mudou do rascunho: as visões são **seções que recolhem pelo título** (demos [`v2`](../../demos/demo-tendencias-v2.html) e [`v3`](../../demos/demo-tendencias-v3.html)), então **V1 e V12 (seletor e janela "Ver como") não existem**; o mês de referência (V3) e o "Comparar com" (V6) ficam no topo das visões, com ‹ ›, e **não são guardados**; o Pagamento (V15) usa o texto do status e a cor que o app recebe do servidor, sem a regra de esconder (V16: o item 41 já está pronto); V17 a V19 valem (o `limite`, aprovado em 2026-11-02, a janela do limite e a leitura de `/consultas`, que acontece uma vez, quando a Tendências mostra uma seção que precisa delas, nunca na subida do app). Pagamento de fatura **sem a marca** `pagaFatura` continua somando duas vezes com as compras no CREDITO: o servidor só tira do total o que está marcado.

## Regras

- **V1. Seletor "Ver como".** A Tendências ganha Meses (a de hoje), Ritmo, Calendário, Fixos, Tags, Pagamento e Metas. Abre em **Meses**, sempre; a escolha não é guardada na consulta nem ao recarregar.
  Com **Resultado** escolhido, só Meses fica ativa; as outras mostram o motivo ("Disponível em Gastos e Receitas").
- **V2. A consulta é a mesma.** Toda visão parte de `CT.base` filtrada por `q.pred` e pela visão de soma (Gastos ou Receitas), como `CT.resumir`. Oculto, `previsto` e o que o servidor marca com `excluirDoTotal` (pagamento de fatura, transferência, status fora do total) ficam de fora; o front só lê o marcador. **A soma de uma visão, para o mesmo
  período, é igual à da visão Meses** (conferência dos números). Nada é enviado ao servidor, exceto V17.
- **V3. Mês de referência.** As visões novas olham um mês `aaaa-mm` pela data **local** do lançamento (a regra de `CT.base`). Padrão: o mês atual; `‹ ›` navegam pelos meses que têm lançamento na base; o mês atual
  sempre existe, mesmo vazio. Mês sem lançamento da consulta mostra o estado vazio com o nome do mês.
- **V4. Calendário.** Para o mês de referência: uma célula por dia (1 até o último dia do mês), semanas de segunda a domingo nas colunas e os dias da semana nas linhas. Valor do dia = soma dos itens da consulta
  naquele dia. Dia sem gasto: célula vazia, sem número. Faixas: 0 = sem gasto; 1 a 5 = até 20%, 40%, 60%, 80% e 100% do **maior dia do mês** (como na demo aprovada). A legenda mostra os valores de corte desse mês. Cada célula mostra o dia e o
  valor arredondado; tocar ou passar o mouse mostra a data e o valor completos.
- **V5. Tags.** Para o mês de referência: as tags dos itens da consulta, valor = soma dos itens que a têm (item com várias tags conta em cada uma), ordem decrescente por valor (empate: ordem alfabética pt-BR),
  no máximo 8 linhas; as demais somam em **outras**. As tags que a própria consulta menciona entram como as outras (sem filtro especial).
- **V6. Comparar com.** Base = **mês anterior** (padrão), **média de 3 meses** (os três meses imediatamente anteriores ao de referência; mês sem gasto da consulta conta 0) ou **mesmo mês do ano passado**. Variação
  = `(atual − base) ÷ base`, em %, arredondada a inteiro. Base 0 e atual > 0: "novo". Base 0 e atual 0: "igual". Qualquer mês da base que seja o **mês cortado** pelos 1000 lançamentos (`CT.corte`: o mais antigo carregado, possivelmente incompleto), anterior a ele ou fora do que foi carregado: "sem
  base", sem percentual. Setas e palavras: ▲ gastou mais, ▼ gastou menos, ■ igual (em Receitas, a seta continua dizendo "mais" ou "menos", sem cor de bom ou ruim).
- **V7. Ritmo.** Para o mês de referência: acumulado por dia, de 1 até hoje (mês atual) ou até o último dia (mês passado), e a curva do **mês-base inteiro** (V6, só mês anterior ou mesmo mês do ano passado; com "média de
  3 meses" a base é a curva média dia a dia dos três meses). **Projeção** (só mês atual, com **3 dias passados ou mais**): `acumulado até hoje ÷ dias passados × dias do mês`, desenhada tracejada de hoje até o último dia, com o valor
  final escrito. Menos de 3 dias: sem projeção, com o motivo. Rotulada **Simulação**; não soma previstos nem lê Futuros.
- **V8. Fixos e variáveis.** Para cada um dos **6 meses** terminando no de referência: fixos = soma dos itens da consulta com `recorrente`; variáveis = o resto; percentual de fixos = fixos ÷ total. Mês sem gasto: barra
  vazia, sem percentual. Nota fixa sobre a marcação (intent, "Limites").
- **V9. Corte dos 1000.** O aviso de mês incompleto (`CT.corte`) aparece em todas as visões que incluem o mês mais antigo carregado; nenhuma visão mostra variação contra mês cortado (V6).
- **V10. A visão Meses não muda** (barras, "Outras tags nesses lançamentos", lista da consulta). A faixa "Simulação" fica no topo em todas as visões.
- **V11. Cores e acessibilidade.** Só tokens do app, claro e escuro. Valor escrito no calendário; variação em seta **e** palavra; estado das metas em ícone **e** palavra; nenhuma informação só por cor. Cada gráfico tem
  um título de leitura (`aria-label`) e, para leitor de tela, uma lista com os mesmos números.
- **V12. Janela "Ver como" (celular).** Janela de baixo com as sete opções em linhas de 56 px, a atual marcada com "✓". Entra em `camadasAbertas()` e `fecharCamadaTopo()`; o voltar fecha só ela. No computador (≥ 1160 px)
  são fichas em linha, sem janela.
- **V13. Carga.** As visões A a D não leem nada novo: usam `lancamentosCache`. `carga.test.js` continua provando que a subida do app não lê lançamentos nem `/consultas`.
- **V14. Pior caso.** Medir por script, nos dois temas e em ≈ 390 px e ≥ 1280 px: valor de 7 dígitos, tag de 40 caracteres, mês de 31 dias, mês vazio, 1000 lançamentos, consulta com 60 fichas.
- **V15. Pagamento (depende do item 41).** Agrupa os itens da consulta pelo **texto** de `status`, sem interpretar. Nome e cor de cada grupo vêm do que o servidor mandar para esse status (a mesma fonte do Histórico).
  Sem nome de status no código de `web/` (`sem-status.test.js` continua passando). Mostra percentual e valor; ordem decrescente.
- **V16. Pagamento sem catálogo.** Enquanto o servidor não mandar rótulo e cor, a visão **não aparece** no seletor.
- **V17. Metas (depende da aprovação do `limite`).** O item da consulta (`CONSULTA#<id>`) ganha `limite`: número ≥ 0, em reais, até 2 casas, opcional; ausente = sem limite. O servidor valida **só o formato**, não
  recalcula nada. Quem altera é quem pode alterar a consulta (autor e dono, `consulta.editar`); ler é de quem lê consultas. O `limite` entra na versão do item como qualquer atributo (`version`).
- **V18. Metas, cálculo.** Para o mês de referência, gasto = soma da consulta no mês (V2). Estado: **dentro** abaixo de 85% do limite (✓), **perto** de 85% a 100% (!), **passou** acima de 100% (✕), com a palavra e o
  quanto passou. A barra mostra o limite como marca fixa. A visão lista a consulta aberta e as outras consultas salvas com `limite`, cada uma calculada no navegador.
- **V19. Metas e consultas, leitura.** (Implementado assim: as seções "Comparar" e "Ao longo do mês" leem `GET /consultas` **uma vez**, quando a Tendências é mostrada com uma delas aberta.) A visão lê `GET /consultas` **ao abrir** (e relê se a janela de consultas salvas alterar algo), nunca na aba nem na subida. `carga.test.js` ganha esse caso.

## Testes

| # | Cenário | Esperado |
|---|---|---|
| VS-T1 | Mesma consulta no mês, todas as visões | Soma do Calendário = soma das Tags (sem sobreposição) = barra do mês na visão Meses |
| VS-T2 | Calendário: mês de 28, 30 e 31 dias; mês começando em dia diferente da semana | Células nos lugares certos; dias fora do mês não existem |
| VS-T3 | Calendário: faixas de um mês de gasto baixo e de um alto | Mesmas 5 faixas pelo maior dia; legenda com os cortes de cada |
| VS-T4 | Lançamento à 23h50 e à 00:10 (fuso local) | Cai no dia local certo, como em `CT.base` |
| VS-T5 | Tags: item com 3 tags; mais de 8 tags | Conta nas 3; 8 linhas mais "outras" |
| VS-T6 | Variação: base 0, base e atual 0, base antes do corte, média de 3 meses com mês vazio | "novo", "igual", "sem base", média com 0 |
| VS-T7 | Comparar com: troca entre as três bases | Percentuais recalculados; a escolha não vai para a consulta (fases A a D) |
| VS-T8 | Ritmo: dia 1, 2 e 3 do mês; mês passado; mês sem gasto | Sem projeção nos dias 1 e 2; projeção a partir do 3; só curva no passado; vazio |
| VS-T9 | Ritmo: projeção em mês de 30 e de 31 dias | `acumulado ÷ dias × dias do mês` |
| VS-T10 | Fixos: item com e sem `recorrente`; mês só de variáveis; mês vazio | Soma e percentual; 0%; barra vazia |
| VS-T11 | Resultado escolhido | Só Meses ativa; as outras com motivo |
| VS-T12 | Oculto e `previsto` | Ficam de fora de todas as visões |
| VS-T13 | `carga.test.js` | A subida não lê lançamentos nem `/consultas`; as visões A a D não leem nada |
| VS-T14 | Janela "Ver como" | Entra em `camadasAbertas()`; voltar fecha só ela |
| VS-T15 | `sem-status.test.js` | Nenhum nome de status em `web/visoes.js` nem em `tendencias.js` |
| VS-T16 | Pior caso (V14) | Nada passa da coluna nem encosta no vizinho; captura nos dois temas |
| VS-T17 | Metas: formato do `limite` (negativo, texto, 3 casas) e permissão (membro na consulta de outro) | `400`; `403`; dono consegue |
| VS-T18 | Metas: gasto de 84,9%, 85%, 100% e 100,01% do limite | dentro, perto, perto, passou |
| VS-T19 | Metas: abrir a visão | Uma leitura de `/consultas`; trocar de aba e voltar não relê |

## Telas (depois da demo aprovada)

Demo v3 aprovada em 2026-11-02 (mês de referência, Comparar com e janela do limite): [`demo-tendencias-v3`](../../demos/demo-tendencias-v3.html). Demo anterior: [`demo-tendencias-v2`](../../demos/demo-tendencias-v2.html), com as visões em seções recolhíveis (no lugar do seletor "Ver como" da V1/V12). Texto antigo: com o resultado da Tendências inteiro em cada visão (celular e computador), a janela "Ver como", o "Comparar com", o seletor de
mês e os estados vazio e sem base. O padrão ganha uma linha em `padrao-de-telas.md` na mesma entrega.

## Fora desta entrega

Painel com todas as consultas lado a lado; somar previstos na projeção; inferir fixos pelo histórico; guardar o "Ver como" na consulta; comparar duas consultas entre si; exportar o gráfico; metas por tag ou por
combo; alerta no Telegram quando uma meta passa (seria uma entrega do mecanismo de avisos).
