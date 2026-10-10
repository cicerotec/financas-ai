# Eventos futuros

Conceito: status marcados como "evento futuro" (PREVISTO, TRANSFERINDO, CREDITANDO) ainda não aconteceram: não contam
como reais, não mexem no saldo, saem do Histórico e aparecem na aba **Futuros**. O marcador manda por cima das caixas
"conta como real" e "afeta saldo" sem apagar os valores guardados. A Parte A está feita (PR #4).

**Parte B, feita.** Duas entradas, a mesma lógica (`web/recorrencia.js`, testada com `node web/recorrencia.test.js`):
- **Botão Prever** em cada registro do Histórico e de Futuros (não aparece nas pontas de transferência): abre um fluxo em
  **tela cheia** no padrão de telas (`.ai/padrao-de-telas.md`, padrão 12), com as linhas Status das cópias
  (PREVISTO, CREDITANDO ou TRANSFERINDO), Repetir (Mensal, Trimestral, Semestral, Anual, "Outro intervalo" em meses ou
  anos, e "Repetir quantas vezes" de 1 a 36, por rodas) e Avisar no Telegram (lista que acumula, até 6 prazos; herda o
  do original), a prévia das datas e do total, e o botão fixo "Criar N eventos".
- **Repetir ao lançar:** num evento futuro novo, "Total de ocorrências" cria este e as repetições já com o aviso.
- **Datas:** mesma data e hora do original; dia que não existe no mês vira o último, sempre a partir da data base
  (31/01 mensal: 28/02, 31/03, 30/04; 29/02 anual: 28/02/2029 e volta a 29/02 em 2032).
- **Parcelas:** `2/10` na descrição vira `3/10`, `4/10`... (opcional) e a quantidade para no total.
- **Duplicado:** avisa antes de criar (status, descrição, valor, banco e dia, sem a hora) e deixa pular os iguais ou
  criar mesmo assim. Se uma criação falhar no meio, a janela diz onde parou e o "tentar de novo" pula o que já existe.
- **Série:** as cópias levam `serie: { id, indice, intervalo }`; num registro de série o botão vira **Estender série**
  e parte da última ocorrência carregada, com o mesmo intervalo.
- Criação feita pelo front, uma chamada por cópia (o membro também pode). Fica de fora: editar ou apagar a série
  inteira de uma vez (o `serie.id` já permite) e regra recorrente sem fim.

Limite conhecido: Futuros e o aviso usam os 1000 lançamentos mais recentes carregados; um evento futuro muito antigo
pode ficar de fora.
