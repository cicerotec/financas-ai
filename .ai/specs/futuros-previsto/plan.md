# Plan: Futuros com o marcador `previsto` e status do backend

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/domain/`, `service/lancamento.go`,
`service/config.go`, `service/avisos.go`, `repository/lancamento.go`, `web/js/app.js`). O roteiro verificável é o
[`spec.md`](spec.md).

**Estado:** rascunho para aprovação do dono.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| "Futuro" é o nome do status, lido de `statusFuturo` em `CFG#LISTAS` (ou do padrão `PREVISTO`, `TRANSFERINDO`, `CREDITANDO`) | `domain/aviso_envio.go` (`EhStatusFuturo`, `StatusFuturoPadrao`), `web/js/app.js:8-12` | O servidor só usa isso nos avisos; o resto é do navegador. A função única "isto é previsto?" nasce em `domain/` e troca as duas |
| O saldo, o resumo e os grupos de Futuros são somados no navegador a partir de `lancamentosCache` (1000 mais recentes) | `app.js` (`calcularSaldoBanco`, `resumoFuturos`, `gruposFuturos`, `renderHistorico`) | Se `GET /tx` deixar de devolver previstos, o navegador já não os soma e não os mostra no Histórico, sem regra nova no front. O resumo de Futuros precisa de rota própria (item 40 cobre o resto) |
| `GET /tx` lista por data e filtra banco, status e texto em Go, depois do `Query` | `repository/lancamento.go:239` | Excluir previstos é um `continue` a mais; achar **todos** os previstos é uma leitura da partição inteira filtrada em Go, sem expressão nova |
| `Editar` só chama a trava quando `MudaSaldo`, e confere o alvo de antes e o de depois | `service/lancamento.go:169`, `domain/saldo.go:41` | `previsto` entra em `CamposSaldo`, e **o alvo de um estado previsto não conta** (não mexe no saldo): é isso que deixa confirmar um previsto de mês fechado sem pedir confirmação quando a data nova é hoje |
| `Protegidos` descarta do corpo os campos que só o servidor define | `domain/lancamento.go:93` | `previsto` entra na lista; `Importar` não usa `Protegidos` e precisa do mesmo cuidado |
| O `DELETE` já devolve `desfazer: {versao}` e o Desfazer restaura essa versão | `service/lancamento.go:244` | Confirmar devolve o mesmo formato, e o Desfazer reaproveita `POST .../restaurar` |
| A rotina de avisos só avisa se `EhStatusFuturo(listas, status)` | `service/avisos.go:168` | Troca por `EhPrevisto(lançamento)`; o servidor deixa de ler `listas` ali |
| `Config.Gravar` grava o corpo todo (menos `Protegidos`) com controle otimista; `Config.Ler` devolve o item | `service/config.go` | O service ganha uma regra para `nome == "listas"`: manter os campos de status do que está gravado e esconder `statusFuturo` e os três nomes antigos na leitura |
| O front carrega só os 1000 lançamentos mais recentes, e Prever/Estender procuram duplicados e a série nesse cache | `app.js:731`, `web/recorrencia.js` | Sem previstos no cache, Prever e Estender passam a ler a lista de Futuros |
| O status é validado só como "não vazio" | `ValidarTx` | Criar e importar com um dos três nomes antigos passa a ser recusado; editar um antigo sem mudar o status continua aceito |

## Desenho

- **O campo:** `previsto` (booleano), **gravado só quando ligado**, no lançamento e, por consequência, nas versões do histórico
  (que guardam o lançamento inteiro). Sem chave nova, sem índice, sem item novo. **Nome e tipo a aprovar pelo dono.**
- **Funções puras em `domain/`** (recebem o instante como parâmetro, nada de relógio global):
  - `EhPrevisto(l)`: `l["previsto"] == true` **ou** o status é um dos três nomes antigos (constante `StatusAntigosPrevistos`,
    um lugar só). É o que os avisos, a listagem e o saldo usam.
  - `PrevistoPorData(dataEvento, agora)`: o **dia** da `dataEvento` em São Paulo é posterior ao dia de `agora` (hoje não é
    previsto).
  - `AlvosDoSaldo(antes, depois)`: os alvos (banco e mês) que mudam o saldo, **sem** o de um estado `EhPrevisto`.
  - `Atrasado(l, agora)`: previsto com o dia da data **anterior** a hoje.
- **O servidor decide o marcador**, nunca o cliente:
  - `Criar` e `Importar`: `previsto` do corpo é descartado; se `PrevistoPorData`, grava `previsto: true`.
  - `Editar`: descarta o `previsto` do corpo. Se o estado resultante é futuro por data, liga o marcador; se já era previsto,
    **continua** (só o Confirmar o tira). Um antigo (status `PREVISTO`...) deixa de ser previsto quando o status editado sai
    dos três nomes, o que é o caminho que o app de hoje já usa.
  - `Confirmar` é a única rota que tira o marcador.
- **Rota nova de confirmação:** `POST /spaces/{sid}/tx/{id}/confirmar?de=<dataEvento atual>[&confirmaMesFechado=1]`, com
  `{ valor, dataEvento, status }`. O servidor valida (valor maior que zero, data que não seja futura, status que não seja um dos
  três antigos), tira o marcador, grava uma versão `update` (sem valor novo em `acao`) e devolve o lançamento, `desfazer` e
  `_seq`. As travas rodam com o **alvo novo**: mês conferido recusa (`409 mes_conferido`), mês fechado por data pede
  `confirmaMesFechado=1` (`409 mes_fechado`).
- **Rota nova de Futuros:** `GET /spaces/{sid}/futuros[?ocultos=1]`. Devolve os previstos e o resumo calculado
  (`pagar`, `receber`, `atrasados`), em dois grupos: `atrasados` e `meses` (cada um com `mes` AAAA-MM e os itens). Os itens
  levam `previsto: true`; o front só formata data e valor. Sem saldo por mês.
- **`GET /tx` passa a excluir previstos** (e perde o filtro por status futuro). Isso sozinho tira os previstos do Histórico,
  da conta de saldo do navegador e da Tendências, sem regra nova no front.
- **Respostas de lançamento** levam `previsto: true` quando `EhPrevisto` (inclusive para os antigos, derivado), para o front
  nunca olhar o nome do status.
- **Avisos:** `Rodar` usa `EhPrevisto(tx)`; o resto (item auxiliar, `MaxAtrasoAviso`) não muda.
- **Listas:** `Config.Ler("listas")` não devolve `statusFuturo` nem os três nomes antigos em `status`; `Config.Gravar("listas")`
  mantém `status`, `statusReal`, `afetaSaldoBanco` e `statusFuturo` do que está gravado e ignora os do corpo. O dado no Dynamo
  não é regravado por isso.
- **Permissão:** ação nova `tx.confirmar` em `auth/authz.go`: owner pode; member só nos lançamentos que criou (como `tx.editar`).
  `GET /futuros` usa `tx.ler`. Rota nova exige teste em `authz_test.go`.

## Fases (cada uma é um PR; o release para a `main` só depois da B)

O backend sozinho tiraria os previstos do `GET /tx` enquanto o app de hoje ainda monta Futuros a partir dele. Por isso o PR de
release `develop → main` **não deve ser mergeado entre a A e a B**. (Alternativa, se preferir: a A entrega a exclusão atrás de um
parâmetro `semPrevistos=1` que só a B liga. Não está no spec; é só dizer.)

- **Fase A, backend:** funções de domínio, o marcador em criar, editar e importar, `CamposSaldo` e `AlvosDoSaldo`, `GET /tx`
  sem previstos, `GET /futuros`, `POST .../confirmar`, `tx.confirmar`, avisos pelo marcador, regra de `listas`, testes (relógio
  fixo) e docs (`modelo-de-dados`, `security`, `avisos`, `eventos-futuros`).
- **Fase B, Futuros e confirmar (demo aprovada):** a aba Futuros passa a ler `GET /futuros` (resumo e Atrasados prontos, sem saldo
  por mês), ganha a ação **Confirmar** (janela de baixo e botão verde na tabela) com o aviso e o Desfazer, e o Histórico sai
  do `ehFuturo`. Sai do front o que olha status ou soma em Futuros.
- **Fase C, limpeza de status (precisa de demos):** Listas perde a edição de status e as três caixas; Lançar perde o "evento
  futuro" e os status antigos do combo; Prever e Estender perdem o "Status das cópias" e passam a achar duplicados pela lista de
  Futuros; o filtro "Incluir eventos futuros" da busca some. **Cada tela tem a sua demo antes do código**, como as outras.

## Riscos

| Risco | Resposta |
|---|---|
| Release só com a A esvazia Futuros no app de hoje | A B vem antes do release; a descrição do PR repete (como no item 38) |
| Algum espaço marcou **outro** status como "evento futuro" em `CFG#LISTAS` e ele deixa de ser futuro | Antes do deploy, ler o `statusFuturo` real de cada espaço (dev e prod) e comparar com os três nomes; o que sobrar vira decisão do dono |
| Ler a partição inteira a cada `GET /futuros` pesa com o tempo | Medir no dev com os dados fictícios; sem chave nova nesta entrega; um GSI só se o dono aprovar |
| Prever e Estender perdem a detecção de duplicados quando o cache não tem mais previstos | A Fase C troca a fonte pela lista de Futuros; até lá, Prever continua com o comportamento de hoje só nos previstos antigos |
| Confirmar vira hábito e perde o sentido | A versão guarda quem e quando; a trava de mês fechado continua valendo |
| O cliente tenta ligar ou desligar o marcador | `previsto` vem descartado do corpo em criar, editar e importar; só `POST .../confirmar` o tira; testes que forjam |
| A cópia de um Prever com data passada (série antiga) deixa de ser previsto | É a regra da data; a cópia vira lançamento comum e conta no saldo. Vai nas decisões abaixo |

## Decisões que o plan fixou e o dono pode contestar

1. O campo se chama **`previsto`**, booleano, gravado só quando ligado (alternativa: `aConfirmar`).
2. **Hoje não é previsto:** o marcador liga quando o **dia** da data (em São Paulo) é posterior a hoje.
3. **Confirmar** exige valor maior que zero, data **não futura** e status que não seja um dos três antigos; a versão gravada é um
   `update` comum.
4. **O alvo de um previsto não conta** para as travas: confirmar um previsto de mês fechado com a data de hoje não pede confirmação;
   com a data do mês fechado, pede (e mês conferido recusa).
5. **`GET /tx` deixa de devolver previstos** (o de hoje devolve tudo e o front filtra).
6. **Criar e importar com um dos três nomes antigos é recusado**; editar um antigo sem mudar o status continua aceito.
7. **Importar** recalcula o marcador pela data e **mantém** o `previsto: true` que vier num item (backup fiel), mas nunca o tira.
8. A cópia de um Prever cuja data já passou **não é** previsto.
9. `salvarListas` ignora os campos de status em vez de recusar (o front antigo continua gravando sem erro).
10. O **catálogo padrão de status passa a existir no código** (hoje o app grava os padrões em `CFG#LISTAS` na primeira vez, `app.js:704`):
    um espaço sem item de listas nasce com `PAGO`, `CREDITO IN`, `CREDITO EX` e `CONTAS` (a lista de hoje, `app.js:5`, sem os três nomes antigos); o dono
    confirma ao aprovar. A lista é herança da planilha e muda com o desmembramento do `PAGO`. **A demo de confirmar mostra `PAGO` e
    `CREDITO` por simplicidade; no app o seletor de Status lista o catálogo real.**
