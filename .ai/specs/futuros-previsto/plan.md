# Plan: Futuros com o marcador `previsto` e status do backend

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/domain/`, `service/lancamento.go`,
`service/config.go`, `service/avisos.go`, `repository/lancamento.go`, `web/js/app.js`). O roteiro verificável é o
[`spec.md`](spec.md).

**Estado:** aprovado pelo dono em 2026-10-24. Fases A e B feitas; C pendente. **O release para a `main` não deve sair entre a A e a B:** o app de hoje ainda monta Futuros a partir de `GET /tx` e cria `PREVISTO` pelo status. **Vem depois** dos itens 40
([`saldo-servidor`](../saldo-servidor/plan.md)) e 33 fatiado ([`fatura-servidor`](../fatura-servidor/plan.md)).

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| "Futuro" é o nome do status, lido de `statusFuturo` em `CFG#LISTAS` (ou do padrão `PREVISTO`, `TRANSFERINDO`, `CREDITANDO`) | `domain/aviso_envio.go` (`EhStatusFuturo`, `StatusFuturoPadrao`), `web/js/app.js:8-12` | O servidor só usa isso nos avisos; o resto é do navegador. A função única "isto é previsto?" nasce em `domain/` e troca as duas |
| A lista de status é o padrão fixo do front (`app.js:5`) mais o que está em `CFG#LISTAS`; o app a edita (`app.js:895-935, 1008-1035, 1312-1317`) | `web/js/app.js`, `web/index.html:213-217` | O servidor passa a mandar a lista; o front só a exibe e devolve a escolha |
| O saldo, o resumo e os grupos de Futuros são somados no navegador a partir de `lancamentosCache` (1000 mais recentes) | `app.js` (`calcularSaldoBanco`, `resumoFuturos`, `gruposFuturos`, `renderHistorico`) | O saldo sai do navegador no item 40 e as faturas no item 33. Aqui falta o resumo e os grupos de Futuros (rota própria); o total do Histórico **sai da tela até o item 42** |
| `GET /tx` lista por data e filtra banco, status e texto em Go, depois do `Query` | `repository/lancamento.go:239` | Excluir previstos é um `continue` a mais; achar **todos** os previstos é uma leitura da partição inteira filtrada em Go, sem expressão nova |
| `Editar` só chama a trava quando `MudaSaldo`, e confere o alvo de antes e o de depois | `service/lancamento.go:169`, `domain/saldo.go:41` | `previsto` entra em `CamposSaldo`, e **o alvo de um estado previsto não conta** (não mexe no saldo): é isso que deixa confirmar um previsto de mês fechado sem pedir confirmação quando a data nova é hoje |
| `Protegidos` descarta do corpo os campos que só o servidor define | `domain/lancamento.go:93` | `previsto` entra na lista; `Importar` não usa `Protegidos` e precisa do mesmo cuidado |
| O `DELETE` já devolve `desfazer: {versao}` e o Desfazer restaura essa versão | `service/lancamento.go:244` | Confirmar devolve o mesmo formato, e o Desfazer reaproveita `POST .../restaurar` |
| A rotina de avisos só avisa se `EhStatusFuturo(listas, status)` | `service/avisos.go:168` | Troca por `EhPrevisto(lançamento)`; o servidor deixa de ler `listas` ali |
| `Config.Gravar` grava o corpo todo (menos `Protegidos`) com controle otimista; `Config.Ler` devolve o item | `service/config.go` | O service ganha uma regra para `nome == "listas"`: devolver a lista de status do servidor, esconder as caixas e manter no item gravado os campos de status |
| O front carrega só os 1000 lançamentos mais recentes, e Prever/Estender procuram duplicados e a série nesse cache | `app.js:731`, `web/recorrencia.js` | Sem previstos no cache, Prever e Estender passam a ler a lista de Futuros |
| O status é validado só como "não vazio" | `ValidarTx` | Criar e importar com um dos três nomes antigos passa a ser recusado; editar um antigo sem mudar o status continua aceito |

## Desenho

- **O campo:** `previsto` (booleano), **gravado só quando ligado**, no lançamento e, por consequência, nas versões do histórico
  (que guardam o lançamento inteiro). Sem chave nova, sem índice, sem item novo. Nome (`previsto`) e tipo (booleano) **aprovados pelo dono em 2026-10-24**.
- **Funções puras em `domain/`** (recebem o instante como parâmetro, nada de relógio global):
  - `EhPrevisto(l)`: `l["previsto"] == true` **ou** o status é um dos três nomes antigos (constante `StatusAntigosPrevistos`,
    um lugar só). É o que os avisos, a listagem, o saldo (item 40) e a fatura (item 33) usam.
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
- **O catálogo de status é do servidor, uma entrada por nome** (`spec.md`, R9 a R12c): rótulo, cor, as regras de cada status e se é
  **selecionável**. Nenhum status sai do catálogo; o que deixa de ser usado vira não selecionável e continua valendo onde já está gravado. Na
  tela o status se chama **forma de pagamento**. **Não há data de corte global** (decisão do dono, 2026-10-24): as datas `desde` e `ate`, quando
  existem, só **sugerem** a lista de escolha, e o usuário pode reclassificar o passado (um `PAGO` de 2018 pode virar `PIX`). O dono confirmou que
  não criou status além dos sete padrão; as caixas de `CFG#LISTAS` deixam de ser lidas e uma conferência antes do deploy garante que coincidem.
- **A lista de escolha é do servidor:** `GET /cfg/listas` devolve `status` = os selecionáveis sugeridos hoje (no corte, `["PAGO", "CREDITO"]`, o
  primeiro é o padrão); `GET /status?data=` devolve a lista **para uma data**, e `todos=1` devolve todos os selecionáveis (reclassificar o passado). O
  front a usa em todo seletor, pede de novo quando a data do formulário muda, guarda o texto escolhido e o devolve **como veio** (o texto da opção,
  não um índice). Se o status gravado de um lançamento não está na lista (um antigo), o seletor o mostra como uma opção a mais e o devolve
  **inalterado**: editar só a descrição nunca troca o status sozinho. "Ver todas" no seletor é mudança de tela: **demo antes**.
- **A trava depende do efeito no saldo (R15):** `EfeitoSaldo` (meio e mês, contribuição) no lugar de `CamposSaldo` por nome. Trocar `PAGO` por `PIX`
  não passa pela trava; trocar para um status que não conta, ou mexer em valor, tipo, meio, data, oculto ou previsto, sim.
- **Rota nova de confirmação:** `POST /spaces/{sid}/tx/{id}/confirmar?de=<dataEvento atual>[&confirmaMesFechado=1]`, com
  `{ valor, dataEvento, status }`. O servidor valida (valor maior que zero, data que não seja futura, status que não seja um dos
  três antigos), tira o marcador, grava uma versão `update` (sem valor novo em `acao`) e devolve o lançamento, `desfazer` e
  `_seq`. As travas rodam com o **alvo novo**: mês conferido recusa (`409 mes_conferido`), mês fechado por data pede
  `confirmaMesFechado=1` (`409 mes_fechado`).
- **Rota nova de Futuros:** `GET /spaces/{sid}/futuros[?ocultos=1]`. Devolve os previstos e o resumo calculado
  (`pagar`, `receber`, `atrasados`), em dois grupos: `atrasados` e `meses` (cada um com `mes` AAAA-MM e os itens). Os itens
  levam `previsto: true`; o front só formata data e valor. Sem saldo por mês.
- **`GET /tx` passa a excluir previstos** (e perde o filtro por status futuro). Isso tira os previstos do Histórico e da
  Tendências sem regra nova no front.
- **Respostas de lançamento** levam `previsto: true` quando `EhPrevisto` (inclusive para os antigos, derivado), para o front
  nunca olhar o nome do status.
- **O Histórico sai do servidor pronto (R12a e R12b) e sem total (R12c):** cada linha traz `rotulo` e `corPadrao`; `GET /tx` traz a lista de filtros de status **que
  aparecem no período consultado** (inclusive os antigos, como `CREDITO IN` em 2019, e os reclassificados). **O total da lista e a soma por mês saem da tela até o item 42**
  (decisão do dono, 2026-10-24: o resumo do filtro inteiro precisa do filtro de tags no servidor). O front não soma, não decide o que existe em cada época e só exibe.
- **Avisos:** `Rodar` usa `EhPrevisto(tx)`; o resto (item auxiliar, `MaxAtrasoAviso`) não muda.
- **Listas:** `Config.Ler("listas")` devolve `status` (a lista do servidor) e **não** devolve `statusFuturo`, `statusReal` nem
  `afetaSaldoBanco` (os itens 40 e 33 já tiraram o último uso deles do front); `Config.Gravar("listas")` mantém esses campos do
  que está gravado e ignora os do corpo. O dado no Dynamo não é regravado por isso.
- **Permissão:** ação nova `tx.confirmar` em `auth/authz.go`: owner pode; member só nos lançamentos que criou (como `tx.editar`).
  `GET /futuros` usa `tx.ler`. Rota nova exige teste em `authz_test.go`.

## Fases (cada uma é um PR; o release para a `main` só depois da B)

Pré-requisitos: itens 40 e 33 fatiado concluídos (o navegador não soma mais nada com status).

O backend sozinho tiraria os previstos do `GET /tx` enquanto o app de hoje ainda monta Futuros a partir dele. Por isso o PR de
release `develop → main` **não deve ser mergeado entre a A e a B**. (Alternativa, se preferir: a A entrega a exclusão atrás de um
parâmetro `semPrevistos=1` que só a B liga. Não está no spec; é só dizer.)

- **Fase A, backend:** funções de domínio, o marcador em criar, editar e importar, `CamposSaldo` e `AlvosDoSaldo`, `GET /tx`
  sem previstos, `GET /futuros`, `POST .../confirmar`, `tx.confirmar`, a lista de status em `GET /cfg/listas`, avisos pelo
  marcador, regra de `listas`, o saldo e a fatura do servidor passando a tratar previsto como fora, testes (relógio fixo) e docs
  (`modelo-de-dados`, `security`, `avisos`, `eventos-futuros`).
- **Fase B, Futuros e confirmar (demo aprovada):** a aba Futuros passa a ler `GET /futuros` (resumo e Atrasados prontos, sem saldo
  por mês), ganha a ação **Confirmar** (janela de baixo e botão verde na tabela) com o aviso e o Desfazer, e o Histórico sai
  do `ehFuturo`. Sai do front o que olha status ou soma em Futuros.
- **Fase B feita** (`feature/futuros-tela-servidor`): a aba Futuros lê `GET /futuros`; a ação **Confirmar** ("Já paguei" / "Já recebi") abre a janela com valor, data e status e dá o aviso de 6 s com Desfazer; o Histórico perdeu o total e a soma por mês e não olha status; o cache do Histórico não tem previstos. Para o app continuar inteiro com o backend novo, a B também: manda a **transferência** para `POST /transferencias`; deixa o Lançar mostrar o aviso e a repetição pela **data futura** (não pelo status); mostra um status antigo (ex.: `PREVISTO`) como uma opção a mais ao editar; faz o Prever copiar com um status da lista do servidor; esconde "Incluir eventos futuros" da busca; e monta os filtros de status do Histórico com os status que aparecem nos dados. A data sugerida da janela de confirmar é **hoje** (a demo sugeria a data prevista; saber se o mês fechou é do servidor, que pergunta se for o caso). Ainda sem demo nova: nada visual além do que a demo aprovada mostra.
- **Fase C, o front sem status (demo antes do código, uma por tela):** a lista fixa de `app.js:5` e as caixas somem; Listas só
  exibe a lista de status (e as cores, como hoje); Lançar, Prever e o filtro do Histórico usam a lista do servidor; Prever e
  Estender perdem o "Status das cópias" e leem duplicados e série de `GET /futuros`; "Incluir eventos futuros" some da busca;
  `consulta-tags.js` deixa de receber `ehReal`. Entra **um teste de CI que falha se `web/` mencionar um nome de status ou
  `ehFuturo`, `ehStatusReal`, `afetaSaldo`, `ehCredito`, `statusFuturo`**.

## Riscos

| Risco | Resposta |
|---|---|
| Release só com a A esvazia Futuros no app de hoje | A B vem antes do release; a descrição do PR repete (como no item 38) |
| Algum espaço marcou **outro** status como "evento futuro" em `CFG#LISTAS` e ele deixa de ser futuro | Antes do deploy, ler o `statusFuturo` real de cada espaço (dev e prod) e comparar com os três nomes; o que sobrar vira decisão do dono |
| Um lançamento antigo tem um status que não está mais na lista (`CREDITO IN`, `CREDITO EX`) | Ele continua com o texto gravado e aparece como opção extra ao editar; a migração do `faturaAjuste` explícito (item 33b) é separada |
| Ler a partição inteira a cada `GET /futuros` pesa com o tempo | Medir no dev com os dados fictícios; sem chave nova nesta entrega; um GSI só se o dono aprovar |
| Prever e Estender perdem a detecção de duplicados quando o cache não tem mais previstos | A Fase C troca a fonte pela lista de Futuros; até lá, Prever continua com o comportamento de hoje só nos previstos antigos |
| Confirmar vira hábito e perde o sentido | A versão guarda quem e quando; a trava de mês fechado continua valendo |
| O cliente tenta ligar ou desligar o marcador | `previsto` vem descartado do corpo em criar, editar e importar; só `POST .../confirmar` o tira; testes que forjam |
| A cópia de um Prever com data passada (série antiga) deixa de ser previsto | É a regra da data; a cópia vira lançamento comum e conta no saldo. Vai nas decisões abaixo |
| `GET /futuros` lê a partição inteira (os `PREVISTO` antigos, de qualquer ano, continuam em Atrasados), e a base vai de 2016 em diante | Medir no dev com **10 anos de dados fictícios** (gerar mais lançamentos em `scripts/dados-ficticios.py`); sem chave nova; se pesar, um GSI só com a aprovação do dono |
| As caixas de `CFG#LISTAS` de algum espaço diferirem do catálogo (alguém mexeu num padrão) | A conferência de leitura antes do deploy (R9b); a divergência vira decisão do dono antes de ligar |
| Reclassificar o passado por engano | Cada troca é uma versão do histórico (quem, quando), reversível pela restauração; a trava de data segue o efeito no saldo; a troca em massa é só do admin (item 44) |
| Um status criado em Listas que o dono esqueceu | O dono confirmou que não há; a conferência acima também lista qualquer status de `CFG#LISTAS` ou de lançamento fora do catálogo |

## Decisões que o plan fixou e o dono pode contestar

1. O campo se chama **`previsto`**, booleano, gravado só quando ligado (**aprovado pelo dono em 2026-10-24**).
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
10. **O catálogo é uma constante do backend, uma entrada por nome**, com `selecionavel`; `["PAGO", "CREDITO"]` é o sugerido no corte, com o primeiro como
    padrão, e o front devolve o **texto** da opção. `CREDITO IN` e `CREDITO EX` (substituídos pelo `faturaAjuste`), `CONTAS` (a transferência, um comando
    do servidor) e os três nomes de previsto **não são selecionáveis**, mas **continuam no catálogo** para ler o histórico.
11. **Não há data de corte global:** as datas `desde` e `ate`, quando existem, só orientam a sugestão. Cada coisa tem a sua vigência (status, meios).
12. **A escolha do status é livre em qualquer data** (o passado é do usuário), desde que o status seja selecionável. Criar ou confirmar com um não selecionável
    é recusado; editar sem mudar o status e importar backup aceitam o gravado.
13. **A trava de data segue o efeito no saldo**, não o nome dos campos: `PAGO` para `PIX` (os dois contam) não é travado.
14. **O Histórico sai do servidor pronto, sem total:** `rotulo`, `corPadrao` e os filtros de status **do uso no período**. O total da lista e a soma por mês **saem da tela até o
    item 42** (pedido do dono, sem demo). Os filtros que o servidor entende são período, meio, status, texto e ocultos.
15. ~~O `PREVISTO` e os outros dois continuam em Atrasados, mesmo com histórico desde 2016.~~ **Mudou em 2026-10-25 (decisão do dono):** Futuros e Atrasados olham só **do mês passado em diante**; um previsto mais antigo (por exemplo, um `PREVISTO` esquecido de anos atrás) fica de fora de Futuros. Como a chave tem a data (`TX#<dataEvento>#<id>`), o `Query` lê só dessa data em diante, sem ler a partição inteira e **sem índice novo**. Esses lançamentos antigos continuam gravados e não contam no saldo; também não aparecem no Histórico. **A forma de buscar os atrasados antigos vai mudar na tela:** por ano, sob demanda (item 45 do BACKLOG).
16. **A troca em massa do passado** (por exemplo, `PAGO` de 2018 para `PIX`, ou o pedido de um dono novo) é uma ferramenta **só do admin**, em outro item (44);
    aqui o usuário reclassifica um registro por vez, pelo app, com as travas.
17. O status é mostrado como **forma de pagamento**; o campo e a API continuam `status`. `CRIPTO` e `DREX` são formas (o saldo mora no meio onde o dinheiro está).

## Resolvido pelo dono (2026-10-24)

- **O total do Histórico** sai da tela até o item 42 (o total e a soma por mês); o item 42 o traz de volta calculado no servidor junto com o filtro de tags.
- **O nome do campo** é `previsto` (booleano, gravado só quando ligado).
