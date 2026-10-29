# Intent: log de cada requisição (item 50)

Item 50 do BACKLOG. Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`) vêm junto,
para o dono aprovar antes de qualquer código.

**Estado:** rascunho para aprovação (2026-10-29).

## Por quê

Hoje o log da Lambda só tem as linhas `START`, `END` e `REPORT` da AWS e alguns `log.Printf` soltos dos services. Não dá para
saber **qual rota foi chamada, quantas vezes seguidas e quanto cada chamada demorou**. Isso atrapalha o que está em aberto:

1. **Leituras em sequência sem sentido:** apareceram chamadas repetidas que não se explicam olhando o código. Sem um registro
   por requisição, não há como contar nem ver a ordem.
2. **O Histórico não tem índice** e o custo de leitura (RCU) já estourou uma vez (`futuros ano`, 2026-10-27). Para decidir o que
   indexar, é preciso saber quais rotas são chamadas e com que frequência.

## O que queremos

1. **Uma linha de log por requisição HTTP**, ao terminar: método, **ação** (a rota já classificada, sem ids), status e duração
   em milissegundos, mais o `requestId` da Lambda para ligar a linha ao `REPORT`.
2. **Só o que foi chamado, nunca o conteúdo:** sem corpo, sem cabeçalhos, sem query string, sem ids de espaço ou de lançamento e
   sem dados da pessoa.
3. **O agendamento diário de avisos não entra:** não é uma requisição HTTP e já tem os seus próprios logs.
4. **Nenhum comportamento da API muda:** mesmas respostas, mesmos status, mesma ordem de execução.

## Limites (o que fica de fora)

- Contar leituras ao DynamoDB por requisição (`ReturnConsumedCapacity`) é um segundo passo, **só se este log não bastar**: mexe
  nos repositories e merece o seu próprio intent.
- Painéis, alarmes e consultas prontas do Logs Insights.
- Logar quem chamou (`sub`): fica de fora por privacidade; entra só se o dono pedir.
- Trocar o `log` padrão por `slog` no código todo.

## Custo

- Uma linha de ~150 a 250 bytes por requisição. Com 100 mil requisições por mês são ~25 MB, menos de 1% dos 5 GB de ingestão
  gratuitos do CloudWatch Logs. A retenção do grupo já é de 14 dias (`template.yaml`).
- O custo de tempo é uma leitura de relógio e um `Printf` por requisição.

## Pronto quando

- Cada rota da API deixa exatamente uma linha de log ao terminar, com a ação, o status e a duração, e nenhuma linha traz id,
  corpo, query string ou cabeçalho (teste que tenta vazar cada um).
- Uma requisição recusada (sem token, rota inexistente, sem permissão) também deixa a sua linha.
- `go test ./...` passa, o `arquitetura_test.go` continua verde e o `guardrails.md` ou o `ambiente-de-teste.md` dizem onde ver o
  log e como contar chamadas no Logs Insights.
