# Plan: faturas do cartão calculadas e validadas no servidor (item 33, fatia a)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`web/js/app.js:2725-3240`, `backend/domain/`,
`service/config.go`, `repository/config.go`). O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** aprovado pelo dono em 2026-10-24. Fase A feita; B e C pendentes.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| O fechamento de uma fatura é o escolhido para o mês (`faturas[ym].fechamento`) ou, sem ele, o dia do **último mês com fechamento escolhido** (até 60 meses atrás) ou `diaPadrao`, limitado aos dias do mês | `app.js:2751` (`fechamentoDe`) | Vira função pura em `domain/` (`FechamentoDe`), com as bordas (dia 31, fevereiro, 60 meses) |
| O período da fatura `ym` vai do dia seguinte ao fechamento de `ym−1` até o fechamento de `ym`; a fatura base de uma compra é `ym` do dia dela se ela é até o fechamento, senão o mês seguinte | `app.js:2761-2762` | `PeriodoFatura` e `FaturaBase`, puras |
| A fatura efetiva de uma compra é `base + ajuste`, com `ajuste = faturaAjuste` (número) ou o padrão do status (`CREDITO EX` +1, `CREDITO IN` −1, senão 0) | `app.js:51-52, 2770` | `AjusteEfetivo(l)` em `domain/`; o padrão por status é uma constante do backend |
| Entra na fatura: status de crédito (`startsWith('CREDITO')`), não oculto e "real" (não futuro), do banco do cartão; o valor é `+valor` para saída e `−valor` para entrada (estorno) | `app.js:2770-2778`, `46` | `ContaNaFatura(l)` pura; "real" passa a ser `!EhStatusFuturo` agora e `!EhPrevisto` com o item 41 |
| O total da fatura soma os valores das compras com `eff == ym`, arredondado a centavos | `app.js:2780` | `TotalFatura` pura |
| A conferência compara o total com `valorBanco` (digitado); o total conferido é gravado em `faturas[ym].conferida = {total, em}` **pelo navegador** | `app.js:3075-3100` | O servidor calcula o total e grava `conferida` sozinho |
| "Mudou desde a conferência" compara o total de agora com `conferida.total` | `app.js:2977-2992, 3053` | Chega pronto (`mudouDesdeConferencia`, e a contagem por cartão) |
| As compras que explicam a diferença são buscadas no navegador (até 30 compras perto do corte, combinações de 1 a 3, margem de 5 dias) | `app.js:2884-2922` (`candidatosFatura`) | O algoritmo é portado para Go, com os mesmos vetores de teste; a resposta traz a ação (`ajuste` por compra) e o rótulo |
| A configuração de cartões é gravada pelo front inteira em `config/cartoes` (`PUT /cfg/cartoes`) | `app.js:2855` (`atualizarConfigCartoes`) | Viram comandos validados; a rota genérica deixa de aceitar `conferida` e depois deixa de existir |
| A lista de bancos é um vetor de nomes; o front só descobre os cartões ao abrir a aba Cartão (`app.js:825`) e usa o status para saber quais bancos têm compras de crédito | `app.js:2769, 3065` | `GET /cfg/listas` devolve também `meios: [{nome, cartao, ativo}]`, pronto |
| `faturaAjuste` é gravado como vem (sem validação) | `security.md` (pontos em aberto) | A validação (`-1`, `0`, `+1`) entra nesta fatia |
| Os dias de uma compra e de "hoje" são calculados no fuso do **navegador** | `app.js:2739` (`chaveDia`) | O servidor usa São Paulo (`DiaBR`); para quem usa o app no Brasil dá o mesmo |

## Desenho

- **Funções puras em `domain/`** (recebem o instante e os dados, sem I/O): `FechamentoDe(cartao, ym)`, `PeriodoFatura(cartao, ym)`,
  `FaturaBase(cartao, dia)`, `FaturaAtual(cartao, hoje)`, `AjusteEfetivo(l)`, `ContaNaFatura(l, banco)`, `TotalFatura(registros, ym)`,
  `Candidatos(...)`. O que decide por nome de status (`EhCredito`, o padrão do ajuste) fica no **catálogo de status do backend** (item 41), uma entrada por nome: `CREDITO IN` continua sendo crédito com ajuste `−1`, em qualquer data, enquanto estiver gravado. Até o 41 chegar, é uma constante com os sete padrão (o dono não criou outros).
- **`service.Cartoes`:** lê `CFG#CARTOES` e os lançamentos do banco (um `Query` por intervalo de datas do espaço, filtrando o banco em
  Go, como o `Listar` de hoje faz) e devolve os corpos de `GET /cartoes` e `GET /cartoes/{banco}/faturas/{ym}`.
- **Janela de leitura:** para a fatura `ym`, os lançamentos do banco entre o início do período de `ym−2` e o fim do período de `ym+1`
  (uma compra com ajuste `±1` pode pertencer a uma fatura vizinha). Sem limite de 1000 e sem `incompleto`.
- **Comandos de configuração** (só do dono), cada um com validação e controle otimista em `CFG#CARTOES` (`version`), no lugar de
  `PUT /cfg/cartoes`: adicionar, remover, início do controle, fechamento e valor do banco por fatura, conferir e desfazer. A forma
  dos dados em `CFG#CARTOES` **não muda** (`porBanco[banco].{diaPadrao, inicio, faturas[ym].{fechamento, valorBanco, conferida}}`).
- **Conferir:** o servidor calcula o total da fatura; se há `valorBanco` e ele difere (ou não há `valorBanco`), `409 diferenca` com o
  que calculou, a menos que `aceitarDiferenca` seja verdadeiro; grava `conferida = {total, em, por}` com o total **dele**. Fatura de
  histórico (anterior ao `inicio` do cartão) não confere.
- **Registrar a diferença:** o servidor calcula `falta = valorBanco − total`; se for maior que zero, cria o lançamento pelo
  `Lancamentos.Criar` (saída, `falta`, descrição `Ajuste da fatura <rótulo> (juros, anuidade ou IOF)`, no dia do fechamento às 12:00
  em São Paulo, `faturaAjuste` 0, `valorEstimado` falso, status de crédito do catálogo).
- **`meios[].cartao`:** `Config.Ler("listas")` junta `CFG#LISTAS` e `CFG#CARTOES` e devolve `bancos` além do `banco` de hoje (que
  continua, para o front antigo, até a Fase C).
- **Autorização:** `cartao.ler` (owner e member, como a leitura de configuração hoje) e `cartao.escrever` (só do dono). A ação genérica
  `cfg.escrever` deixa de valer para `cartoes`.
- **Validações novas em `ValidarTx`:** `faturaAjuste`, quando presente, só `-1`, `0` ou `+1`.

## Fases (cada uma é um PR)

- **Fase A, backend, só acrescenta:** as funções puras e os testes, `GET /cartoes`, `GET /cartoes/{banco}/faturas/{ym}`, os comandos,
  `meios[]` em `GET /cfg/listas`, a validação de `faturaAjuste`, e a trava para `PUT /cfg/cartoes` **ignorar** `faturas[*].conferida`
  do corpo (a rota continua, para o app de hoje não quebrar). O release pode ir para a `main`.
- **Fase B, a aba Cartão e a linha Fatura do Lançar (sem demo: o visual não muda):** a aba lê as duas rotas novas e usa os comandos; o
  Lançar mostra a linha "Fatura" quando `meios[].cartao`. **Antes de ligar**, a conferência lado a lado (abaixo). Saem do front as
  funções listadas no intent.
- **Fase C, remover:** apaga `PUT /cfg/cartoes` e a ação `cfg.escrever` para `cartoes`; `banco[]` sem marca some da resposta quando o
  front só lê `bancos`; atualiza o `security.md` (sai a linha de `faturaAjuste` e a da conferência de fatura).

## A conferência lado a lado (obrigatória na Fase B)

1. No **dev**, com os dados fictícios: a aba Cartão antiga e `GET /cartoes` e `GET .../faturas/{ym}` dão o **mesmo** total, período,
   compras (com os mesmos ajustes) e candidatos, nas faturas anterior, atual e próxima de cada cartão. Anotar no PR.
2. Na **produção**, só leitura, pelo dono: o mesmo. Se algum total divergir, **a tela nova não entra** até entender (as suspeitas:
   fuso do dia da compra, `diaPadrao` com dia 31, fechamento escolhido em mês anterior ao primeiro, estorno com sinal, valor estimado).

## Riscos

| Risco | Resposta |
|---|---|
| Total diferente do de hoje | A conferência lado a lado; funções puras em tabela com as bordas (dia 31, fevereiro, compra no dia do fechamento, 00:00 em São Paulo) |
| O algoritmo dos candidatos mudar de resultado ao portar | Os mesmos vetores de teste do JS rodam no Go (compras, diferença, resultado esperado); a ordem do resultado é testada |
| Cada abertura de fatura lê muitos lançamentos | Janela de 4 faturas; sem chave nova; medir no dev; um GSI só se o dono aprovar |
| O front antigo ainda gravar `conferida` forjado depois da A | A Fase A faz `PUT /cfg/cartoes` ignorar `conferida`; a C remove a rota |
| O app antigo parar de funcionar quando `PUT /cfg/cartoes` sumir | A Fase C só entra depois da B no ar nos dois ambientes |
| Lançamento de crédito num banco sem cartão configurado | Não entra em fatura nenhuma (como hoje); `meios[].cartao` é falso e a linha Fatura não aparece |

## Decisões que o plan fixou e o dono pode contestar

1. O servidor **calcula e manda** os candidatos de ajuste (o front só mostra e aplica).
2. Os comandos de cartão são **rotas próprias**, e `PUT /cfg/cartoes` é removida (na Fase C).
3. `meios: [{nome, cartao, ativo}]` em `GET /cfg/listas`, **sem chamada extra**; a linha Fatura do Lançar aparece pelo meio, não pelo status.
   O servidor ignora `faturaAjuste` em lançamento que não é de crédito.
4. **Conferir fatura com diferença** pede `aceitarDiferenca` (hoje é uma pergunta do front); sem `valorBanco`, também.
5. A janela de leitura é de `ym−2` a `ym+1` faturas, sem limite de lançamentos.
6. **A trava da fatura conferida fica na fatia b** (decisão anterior mantida; muda comportamento e precisa de demo).
7. **Nomes:** nas rotas e respostas novas o `banco` gravado se chama **`meio`** (meio de pagamento), por um **de-para em `domain/`**; o Dynamo não muda e `GET /tx`
   mantém `banco` até o item que renomeia a tela (R0 do spec). O status se chama **forma de pagamento** na tela.
