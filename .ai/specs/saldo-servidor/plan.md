# Plan: saldo, fechamento e conferência calculados e validados no servidor

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/domain/saldo.go`,
`domain/fechamento.go`, `service/fechamento.go`, `service/lancamento.go`, `repository/saldos.go`, `web/js/app.js:2289-2725`). O
roteiro verificável é o [`spec.md`](spec.md).

**Estado:** rascunho para aprovação do dono (2026-10-24).

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| O navegador calcula a cascata: baixa os lançamentos do banco (páginas de 1000, até 500), filtra `oculto`, `afetaSaldo(status)` e a data de início, soma por mês e faz `acc = arred(acc + mês)` | `app.js:2519` (`calcularSaldoBanco`) | A regra vai para uma **função pura** em `domain/` (`CalcularSaldo`), testada em tabela, que o service chama com o que o repository leu |
| O cache (`caches`, `base`) é calculado no navegador e gravado por `PUT /fechamentos`, que só valida mês e número finito | `app.js:2572`, `service/fechamento.go:219` | A rota some; o service grava o cache depois de calcular |
| `Conferir` recebe `saldo` e `real` do cliente e só compara os dois | `service/fechamento.go:265` | Recebe só `real`; o `saldo` é o que o servidor calculou |
| O ajuste da conferência é montado pelo navegador (`criarAjusteConferencia`): status, `excluirDoTotal`, data do último dia do mês, texto | `app.js:2664` | O servidor cria o lançamento pelo mesmo caminho de escrita (`Lancamentos.Criar`), com as travas |
| O total dos bancos é a soma de `saldosAtuais` no navegador, respeitando `foraDoTotal` | `app.js:2323` (`totalSaldos`) | O servidor devolve `total` pronto |
| O servidor já tem o relógio, o fuso (`domain.FusoApp`, `MesDe`, `MesAtual`), as travas (`Travas.Checar`), a invalidação (`InvalidarSaldos`) e o item `SALDO#<banco>` | `domain/saldo.go`, `service/fechamento.go`, `repository/saldos.go` | Nada disso muda: o cálculo é acrescentado ao lado |
| O que conta no saldo vem de `CFG#LISTAS`: `afetaSaldoBanco` (padrão `PAGO` e `CONTAS`) e `statusFuturo` (padrão os três nomes) | `app.js:8-12`, `domain/aviso_envio.go:204` | O servidor lê **a mesma configuração** que o front lê hoje, para o resultado sair idêntico. O item 41 troca isso por `previsto` e pelo catálogo |
| A data de início do controle é interpretada no fuso do navegador (`new Date(dataInicio + "T00:00:00")`) | `app.js:2520` | O servidor usa São Paulo (como `MesDe`); para quem usa o app no Brasil dá o mesmo |
| `base` do cache é `saldoInicial\|dataInicio\|assinaturaStatus` montada no navegador | `app.js:2485` | O servidor monta a sua `base`, com prefixo de versão: os caches de hoje deixam de casar e são recalculados |

## Desenho

- **`domain.CalcularSaldos` (função pura):** entrada = os lançamentos (um iterador, para não carregar tudo na memória), a
  configuração de cada banco (`saldoInicial`, `dataInicio`, `foraDoTotal`), o ponto de partida de cada banco (mês e saldo do último
  conferido ou do cache válido), a regra "conta no saldo" (`ContaNoSaldo(l, listas)`: não oculto, `afetaSaldoBanco[status]`, não
  futuro) e o `agora`. Saída = por banco, o saldo atual e os fechamentos por mês. Um só passe pela partição do espaço agrupa por
  banco (em vez de uma leitura por banco).
- **Arredondamento e fuso iguais aos de hoje:** `arred(v) = round(v*100)/100` a cada mês; o mês de um lançamento é `MesDe` (São Paulo);
  `dataInicio` vale a partir de 00:00 em São Paulo; cada mês com movimento ou sem, do início até o mês atual, entra na cascata.
- **`service.Saldos`:** monta tudo e devolve o corpo de `GET /saldos`: para cada banco, o `saldo`, os **12 últimos meses** com `valor` e
  `estado` (`conferido`, `travado`, `calculado`, `aberto`, a mesma regra de `historicoMeses`) e a conferência do mês quando há; e o
  `total` (soma dos bancos que **não** estão `foraDoTotal`). Sem `dataInicio`, o banco calcula do primeiro movimento e **não grava cache**
  (como hoje).
- **O cache é do servidor:** depois de calcular, `Saldos` grava os `caches` dos meses encerrados (anteriores ao atual e à
  invalidação) em `SALDO#<banco>` com controle otimista, como `SalvarCaches` fazia, mas com o número que ele mesmo calculou. Falha ao
  gravar só vai para o log (o saldo devolvido está certo). `base` = `v2|saldoInicial|dataInicio|<afetaSaldo ligados>/<statusFuturo ligados>`.
- **`PUT /fechamentos` é removida** (e a ação `fechamento.escrever` com ela). `GET /fechamentos` continua até a tela nova entrar
  (Fase B) e some na C.
- **Conferir vira um comando que confere o servidor:** `POST /fechamentos/conferir` com `{ banco, mes, real, ajustar? }`. O servidor
  calcula o saldo do mês; se bate com `real` (diferença menor que meio centavo), grava `conferidos[mes] = { saldo, real, em, por }`
  com o `saldo` **dele**; se não bate e `ajustar` é verdadeiro, cria o lançamento de ajuste (mesma lógica de hoje: último dia do
  mês, 12:00 em São Paulo, entrada se `real` > calculado, descrição `Ajuste de conferência AAAA-MM`) **pelo `Lancamentos.Criar`**,
  recalcula e confere; se não bate e `ajustar` não vem, devolve `409 diferenca` com `calculado`, `real` e `diferenca`. A ordem e
  o "só do mais recente" (`ErroOrdemFechamento`) seguem como estão.
- **Um service que junta os dois:** `service.Conferencia` recebe `Saldos`, `Fechamento` e `Lancamentos` por construtor (o
  `Fechamento` e o `Lancamentos` já se conhecem por `Travas`; criar o ajuste de dentro de `Fechamento` seria um ciclo).
- **O ajuste conta como confirmação** do item 38: quem pede para registrar o ajuste de um mês fechado por data está confirmando
  mexer nele; a versão grava `confirmouMesFechado`. Mês conferido anterior continua recusando (`409 mes_conferido`).
- **Campos derivados do cliente:** `Conferir` e a nova `GET /saldos` não leem `saldo`, `caches`, `base` nem `invalidoDe` do corpo.
  `PUT /cfg/saldos` (início, saldo inicial e fora do total, que são dados de **entrada**) continua, só do dono.
- **Autorização:** `GET /saldos` usa `fechamento.ler` (owner e member, como hoje). `conferir` e `reabrir` seguem só do dono. A ação
  `fechamento.escrever` é removida de `auth/authz.go` e dos testes.

## Fases (cada uma é um PR)

- **Fase A, backend, só acrescenta:** `domain.CalcularSaldos` e `ContaNoSaldo`, `service.Saldos`, `GET /saldos`, `Conferir` que ignora
  o `saldo` do cliente (continua aceitando o corpo antigo, para o app de hoje não quebrar), `service.Conferencia` com `ajustar`, o
  cache gravado pelo servidor, testes e docs. **Sem remover nada**, então o release pode ir para a `main`.
- **Fase B, a aba Saldos (sem demo: o visual não muda):** a tela chama `GET /saldos`, mostra o que veio, e conferir manda só `real`
  (e `ajustar`). Saem do front as funções listadas no intent. **Antes de ligar**, a conferência lado a lado (abaixo).
- **Fase C, remover:** apaga `PUT /fechamentos`, `GET /fechamentos` e a ação `fechamento.escrever`; `Conferir` passa a recusar `saldo`
  no corpo (`400`); atualiza o `security.md` (sai a linha da brecha).

## A conferência lado a lado (obrigatória na Fase B)

1. No **dev**, com os dados fictícios (`scripts/dados-ficticios.py`): abrir a aba Saldos antiga e chamar `GET /saldos`; os saldos e
   os fechamentos dos 12 meses de cada banco são **iguais**. Anotar no PR.
2. Na **produção**, só leitura, pelo dono: a aba antiga e o `GET /saldos` (com o `curl` do spec) dão o mesmo número em cada banco.
   Se algum divergir, **a tela nova não entra** até entender (as suspeitas: fuso de `dataInicio`, arredondamento, lançamento
   sem `banco`, status que mexe no saldo e não está na configuração).

## Riscos

| Risco | Resposta |
|---|---|
| O saldo novo diferir do antigo e a pessoa ver o número mudar | A conferência lado a lado acima; função pura testada em tabela com as bordas (virada de mês em São Paulo, `dataInicio` no meio do mês, lançamento exatamente à meia-noite UTC) |
| Ler a partição inteira a cada chamada pesar (a base vai de 2016 em diante) | Um passe só para todos os bancos, a partir do último ponto verificado; **teste de volume com 10 anos de dados fictícios** (estender `scripts/dados-ficticios.py`), medindo a primeira abertura depois do deploy (sem cache) e as seguintes (com cache), dentro do tempo da Lambda; sem chave nova; um GSI só se o dono aprovar |
| Caches de hoje (escritos pelo navegador) conterem números errados ou forjados | O prefixo `v2` na `base` os invalida no primeiro cálculo; nada é apagado do Dynamo |
| O `conferidos` já gravado ter `saldo` vindo do navegador | Ficam como estão (são a conferência do dono com o extrato); só as novas gravam o `saldo` do servidor |
| Criar o ajuste dentro do comando falhar no meio | O ajuste é gravado primeiro (uma transação do lançamento); se o conferir falhar depois, o ajuste fica e a mensagem diz isso; a pessoa repete a conferência sem criar outro |
| O app de hoje chamar `PUT /fechamentos` depois da Fase C | A Fase C só entra depois da B no ar nos dois ambientes |

## Decisões que o plan fixou e o dono pode contestar

1. O servidor lê `afetaSaldoBanco` e `statusFuturo` de `CFG#LISTAS` **por enquanto** (para sair idêntico). O item 41 troca isso pelo **catálogo de status com vigência**, e a regra passa a usar as propriedades da entrada válida na **data do evento**; o dono confirmou que não criou status além dos sete padrão, então as duas fontes coincidem.
2. O **ajuste** da conferência é criado pelo servidor dentro do comando, e vale como a confirmação de mês fechado por data.
3. `GET /saldos` devolve os **12 últimos meses** (como a tela de hoje) e não tem paginação.
4. Sem `dataInicio`, o banco não grava cache e mostra o saldo desde o primeiro movimento, como hoje.
5. O `base` do cache ganha o prefixo `v2`, o que **invalida todos os caches de hoje** de uma vez.
6. `Conferir` na Fase A ainda aceita o corpo antigo (ignorando o `saldo`); a recusa com `400` só vem na Fase C.
7. A ação `fechamento.escrever` deixa de existir.
