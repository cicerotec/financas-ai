# Intent: combos sugeridos calculados pelo servidor (item 57)

Este documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md) e o roteiro verificável em [`spec.md`](spec.md). Nada de código nem de DynamoDB antes da aprovação do dono.

**Estado:** rascunho para aprovação (2026-11-06). Nasce do pedido do dono: "os combos sugeridos não podem ler 1000 registros".

## Por quê

Hoje o app **calcula os combos sugeridos no navegador**, contando as tags dos `lancamentosCache` (os 1000 mais recentes). Esse cache só é lido quando o Histórico ou a Tendências abre. Resultado:

- ao abrir o Lançar, **não há sugerido nenhum** até o dono visitar uma dessas telas;
- se o Lançar passasse a ler os 1000 lançamentos para sugerir, seriam **mil leituras a cada abertura** (a tabela tem 5 unidades de leitura e já estourou em consultas grandes);
- a mesma conta é repetida em cada aparelho.

Combos **gravados à mão não existem em produção** (o dono não os cria), então o que vale é o sugerido.

## O que queremos

1. **O servidor calcula os sugeridos uma vez por semana** e guarda o resultado. O app só **lê** o que está guardado, na mesma leitura de `config/combos` que o Lançar já faz ao subir.
2. **Sem ler nenhum lançamento ao abrir o Lançar.** O custo de abrir não muda.
3. **Mesma regra de hoje:** conjuntos de **2 tags ou mais**, usados **3 vezes ou mais**, ignorando a ordem das tags e os lançamentos excluídos e os previstos. O servidor guarda até **10** e o app mostra **5** com o "Mostrar mais" que já existe.
4. **Só os 1000 lançamentos mais recentes** (cerca de 6 meses, o que o dono aceita), como o app já faz hoje, mas feito pelo servidor. Previstos e excluídos não entram na conta nem nos 1000. Se o dono lançar mais, os 1000 cobrem menos tempo.
5. **O app para de contar, em todas as telas.** Some `contagemConjuntos`: Lançar, Histórico (filtro por combos), Tendências e o texto da IA leem os mesmos sugeridos que o servidor guardou. O Histórico e a Tendências **continuam lendo os 1000 lançamentos para a lista e os gráficos**; o que muda é que deixam de usá-los para calcular combos. O Futuros não conta combos hoje.
6. **O dono comanda a rotina na aba Listas, seção Combos:** um interruptor "Gerar combos sugeridos toda semana", já ligado. Desligar para a rotina e mantém o último resultado.
7. **Combo manual continua igual** (salvar, apagar em Listas, só o dono). Se um manual tem o mesmo conjunto de um sugerido, o manual aparece com a contagem e o sugerido some.

## Limites (o que fica de fora)

- **Só `sugeridos`.** Contagem de uso por tag ("Mais usadas") e sugestão pela descrição (`porDescricao`) **não entram**: o dono dispensou em 2026-11-06.
- **Uma tela só mexe:** o interruptor em Listas > Combos (com demo antes). A janela de Tags da v5 do Lançar (`demo-lancar-v5-tags`) já mostra os combos; só muda de onde vêm os sugeridos.
- **O passado não é tocado.** Nenhum lançamento é alterado.
- **A sugestão pode ficar até uma semana velha.** É aceitável para sugerir.
- **Dependência da rotina:** com o interruptor desligado e sem nunca ter rodado, só aparecem os combos manuais, também no filtro do Histórico.
- **Sem botão "recalcular agora"** na tela. A primeira carga é a rotina disparada uma vez à mão, no dev e depois na produção.

## Custo e riscos (para o dono decidir)

- **Campos e item novos no DynamoDB:** `sugeridos` e `geradoEm` dentro de `CFG#COMBOS` (item que já existe) e um item auxiliar `tipo: "combos"` no `AVISOS#ATIVOS`, o interruptor da rotina. Nenhuma chave nova. **Precisa da sua aprovação.**
- **Uma regra nova no EventBridge** (semanal, madrugada de domingo), separada da dos avisos: se falhar, os avisos não são afetados.
- **Leitura da tabela:** os 1000 lançamentos mais recentes com 5 unidades de leitura (duas páginas de 500). A rotina lê em páginas pequenas e espera entre elas; uma vez por semana, de madrugada, é aceitável.
- **Como a rotina descobre os espaços:** pelo interruptor que o dono liga na aba Listas, seção Combos (já ligado), igual aos avisos e às recorrências (decisão D1 do [`plan.md`](plan.md)). É a única tela nova: **demo antes**.
