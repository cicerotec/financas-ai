# Intent: saldo, fechamento e conferência calculados e validados no servidor

Item 40 do BACKLOG. Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`) vêm junto,
para o dono aprovar antes de qualquer código.

**Estado:** rascunho para aprovação (2026-10-24). Decisão do dono de 2026-10-18: o front não é fonte de número. **É o primeiro da
fila:** vem antes do item 33 fatiado ([`fatura-servidor`](../fatura-servidor/intent.md)) e do 41
([`futuros-previsto`](../futuros-previsto/intent.md)).

## Por quê

O navegador calcula o saldo de cada banco (a cascata mensal, `calcularSaldoBanco` em `web/js/app.js`), soma o total dos bancos e
**manda o resultado ao servidor**, que o guarda sem conferir:

- `PUT /spaces/{sid}/fechamentos` (`SalvarCaches`) aceita qualquer número finito para um mês passado, e esse número vira o ponto de
  partida da cascata de todo mundo.
- `POST /spaces/{sid}/fechamentos/conferir` recebe `saldo` e `real`, **os dois do cliente**, e só confere se são iguais entre si.
  Mandando o mesmo valor nos dois, o mês fica "conferido" e travado com o saldo que a pessoa quiser.

É uma brecha de segurança (só o dono chama essas rotas, mas o número fica gravado como verdade) e também desorganização: a regra
"o que mexe no saldo" mora no navegador, que a lê por **nome de status** (`afetaSaldo`, `ehFuturo`). Enquanto ele somar, não dá para
esconder o status do front (regra do item 41).

## O que queremos

1. **O servidor calcula o saldo de cada banco**, mês a mês (a cascata de hoje), e o **total dos bancos**, com o relógio e o fuso dele
   (São Paulo). O navegador só exibe.
2. **O cache dos meses encerrados é gravado só pelo servidor**, a cada cálculo. A rota que aceitava o cache do cliente
   (`PUT /fechamentos`) deixa de existir.
3. **A conferência recebe só o saldo do extrato (`real`).** O servidor compara com o saldo que **ele** calculou. Se houver diferença,
   a pessoa escolhe entre cancelar e **registrar o ajuste**, que o servidor cria (no lugar de o navegador montar o lançamento).
4. **O servidor rejeita número derivado vindo do cliente** (`saldo`, `caches`, `base`, `invalidoDe`, `conferidos`), em vez de
   confiar nele. Um teste tenta forjar cada um.
5. **O resultado é o mesmo de hoje.** Mesmas regras de quais lançamentos contam (hoje lidas de `afetaSaldoBanco` e `statusFuturo` de
   `CFG#LISTAS`, o que o item 41 troca depois), mesmo arredondamento, mesma cascata. Antes de ligar a tela nova, a aba Saldos antiga e
   a nova são comparadas lado a lado no dev e na produção.
6. **A aba Saldos fica com uma chamada** que devolve tudo pronto (saldo atual, os últimos 12 meses com o estado de cada um e o
   total), sem o limite dos 1000 lançamentos e sem o aviso "⚠ confira".

## Limites (o que fica de fora)

- Não muda a regra de **o que conta** no saldo (futuro, real, afeta saldo): continua lida da configuração de hoje. O item 41
  troca a leitura por `previsto` e pelo catálogo do backend.
- **Não faz fatura nem cartão** (item 33 fatiado) e **não decide status** (item 41).
- Não muda a trava de mês conferido nem a de mês fechado por data (item 38), nem o formato de `SALDO#<banco>` e de `CFG#SALDOS`.
- Não valida o saldo contra o banco (Open Finance, item 39).
- Não muda quem pode ver os saldos (hoje owner e member; a visibilidade por banco é o item 27).

## Custo

- Cada `GET /saldos` lê o histórico do espaço desde o último ponto verificado (o cache evita reler tudo). No servidor isso custa
  leitura de tabela; no navegador deixa de custar tempo e rede (hoje cada banco baixa até 500 páginas).
- Uma **conferência com diferença** passa a ser um comando só ("conferir e registrar o ajuste"), em vez de três passos do front. O
  ajuste é criado no mês conferido, que normalmente já fechou por data; o comando vale como a confirmação explícita do item 38 e a
  versão do histórico a registra.
- Os caches de hoje (escritos pelo navegador) deixam de valer no primeiro deploy e são recalculados pelo servidor: a primeira
  abertura de Saldos depois do deploy é mais lenta.

## Pronto quando

- O servidor devolve os saldos, os meses e o total corretos (conferidos contra a tela antiga), grava o cache sozinho e recusa o que
  o cliente tentar impor, com testes que forjam cada campo e falham antes da mudança.
- A aba Saldos usa só a API nova; saem do front `calcularSaldoBanco`, `persistirCaches`, `historicoMeses`, `totalSaldos`,
  `cachesValidos`, `pontoDePartida`, `baseCfg`, `assinaturaStatus` e `criarAjusteConferencia`.
- `PUT /fechamentos` não existe mais e `Conferir` não lê `saldo` do corpo.
- `security.md` (a linha de pontos em aberto sai), `arquitetura.md`, `modelo-de-dados` e o BACKLOG ficam em dia na mesma entrega.
