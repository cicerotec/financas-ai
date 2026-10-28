# Arquitetura: camadas, regras no servidor e histórico imutável

Registro das decisões tomadas em 2026-10-10, depois de o dono achar um erro grave: **uma edição sobrescreveu um
registro e o dado antigo se perdeu** (o `PUT` substitui o item; o PITR do DynamoDB está desligado). A conversa também
mostrou que as regras do sistema estão espalhadas no front e que o backend é pouco mais que um armazém com login.

Nada aqui foi implementado ainda. Cada decisão tem o estado **fechada** (o dono confirmou) ou **em aberto**.

## Princípios

1. **O front não decide (fechada).** Ele renderiza e consome API. Toda regra de negócio (futuro ou real, o que afeta o
   saldo, fatura, ajuste, o que é histórico) vive no backend e chega pronta pela API. Nenhuma decisão por **etiqueta**
   (o nome de um status ou de um banco).
2. **O domínio é claro e todos o usam, sem exceção (fechada).** Nada de regra inventada fora dele.
   **Corolário (fechada em 2026-10-18): o front não é fonte de número.** Ele exibe e, no máximo, calcula para mostrar
   (como a aba Tendências), mas nunca envia o resultado como verdade. O servidor calcula saldo, fechamento, totais e
   faturas por conta própria e **rejeita** qualquer campo derivado que venha do cliente. Hoje isso **não** é assim:
   veja "Saldo e totais calculados no servidor" abaixo.
3. **Nenhuma chave ou campo novo no DynamoDB sem discutir antes (fechada).** Cada campo carrega uma decisão de
   modelagem que não é óbvia lendo o código; `faturaAjuste` é o exemplo (veja "Domínio").
4. **O histórico nunca apaga (fechada).** Todo dado que entra vai também para o histórico, inteiro; o que muda vira
   uma versão nova.

## Camadas

```
main         só cria os clients, monta o repository e liga as peças (não faz mais nada)
  │
controller   recebe o request (rota, token, JSON) e chama o service. Recebe SÓ o service (e o client de auth)
  │
service      regra de uso: orquestra, decide o que o front recebe. Recebe o repository e os outros adaptadores por
  │          construtor, não chama ninguém
domain       tipos e regras puras (Lancamento, Status, Fatura, Saldo, Aviso). Não importa nada; todos o usam

adapters     repository (dynamodb), telegram, e-mail, whatsapp, auth, relógio: implementam as interfaces do service
```

**Dependência só para dentro (fechada).**
- O `service` **declara** as interfaces de que precisa (`Repositorio`, `Avisador`, relógio) e não importa AWS, HTTP nem
  Telegram. O adaptador implementa a interface e converte de/para tipos do domínio. O teste usa uma implementação falsa.
- **Sem variáveis globais**: `db`, `ssmCli`, `table` e `agora()` deixam de ser globais e passam a ser injetados.
- Dependências entram pelo **construtor**; os métodos recebem só os dados da requisição.
- O miolo das regras (fatura base, ajuste, saldo) são **funções puras**, sem I/O; o service lê pelo repositório, chama a
  função pura e devolve a visão pronta.

## Estrutura de pastas do backend (fechada em 2026-10-16)

Cada camada é uma pasta, que em Go é um pacote. O `main.go` fica na raiz de `backend/`.

```
backend/
  main.go        só liga e parte: cria o relógio e o Servidor e entrega a entrada à Lambda (ou ao modo local)
  servidor.go    liga repository, service e controller dos assuntos já migrados (novoServidor, uma vez)
  roteador.go    recebe o evento (HTTP ou agendamento), autentica, classifica a rota, confere o papel e chama o controller
  local.go       modo HTTP local para depuração (só AMBIENTE=dev)
  domain/        tipos e regras puras (Doc, Lancamento, Mudanca, ValidarTx, AlvoDe, MudaSaldo, tags, aviso, série,
                 ErroMesConferido). Não importa nada do projeto nem a AWS
  auth/          token do Cognito (token.go) e permissões por papel (authz.go)
  repository/    adaptador DynamoDB: dynamo.go (a interface do cliente e as chaves), lancamento.go (inclui a listagem),
                 espacos.go (papel e espaços da pessoa), config.go (configurações e aparência), contador.go (o _seq),
                 perfil.go (apelido e chat id no vínculo), saldos.go (fechamento mensal por banco) e avisos.go (o item auxiliar
                 dos avisos). É o único lugar com PK, SK e expressões
  service/       regra de uso. Declara as interfaces de que precisa (Repositorio, Efeitos); não importa AWS nem HTTP
  controller/    um controller por assunto: chama o service e traduz o erro em resposta. resposta.go tem o tipo Resp,
                 Out, Erro e a tradução dos erros do service em status HTTP. Não toca o banco
  adapter/       adaptadores para o que não é banco, um pacote por assunto. Hoje: adapter/relogio (Real e Fixo; o service
                 declara a interface Relogio e nunca chama time.Now) e adapter/notificacao (um arquivo por canal de aviso:
                 telegram.go agora; e-mail e WhatsApp depois). O banco continua em repository/
  (raiz)         só main.go, servidor.go, roteador.go e local.go: não há regra, banco nem variável global
```

**Quem pode importar quem** (o teste `arquitetura_test.go` falha se alguém quebrar):

| Pacote | Não pode importar |
|---|---|
| `domain` | nada do projeto, nada da AWS, `net/http` |
| `service` | `controller`, `repository`, `auth`, AWS, `net/http` |
| `controller` | `repository`, `auth`, o SDK da AWS |
| `repository` | `service`, `controller`, `auth` |
| `auth` | `service`, `controller`, `repository`, o SDK da AWS |
| `adapter/*` | `service`, `controller`, `repository`, `auth` (o relógio não importa nada do projeto nem a AWS) |

- **A migração terminou** (item 32, em quatro fatias, 2026-10): lançamentos, espaços, configurações, tags, contador,
  perfil, fechamento e a trava de mês, avisos e Telegram estão nas pastas. Não há mais ponte na raiz nem variável global
  de dependência (`db`, `table`, `ssmCli`, `agora`...): o `main` cria os clients uma vez e o `Servidor` liga tudo.
  Histórico das fatias: [`specs/arquitetura-camadas/`](specs/arquitetura-camadas/plan.md).
- **Despachante de avisos (item 34):** vive no `service`, em cima da interface `Notificador` (o `service.Avisos` de hoje
  já a usa); cada canal novo é um arquivo em `adapter/notificacao/`. Não há pasta `dispatcher/`.
- Os testes ficam ao lado do código que testam. Os de comportamento da API ficam na raiz, porque passam por `processar`.

## Persistência: o repository (fechada)

O service **não recebe o client do DynamoDB**: recebe um **repository**, que esconde o banco. Quem conhece o DynamoDB é
só o adaptador.

```
main:  config → client DynamoDB → repository (usa o client) → service (recebe o repository) → controller (recebe o service)
```

- **O controller recebe só o service.** Se o controller tivesse o repository, poderia gravar sem passar pelo service e o
  caminho único de escrita do histórico (item 31) deixaria de ser garantido.
- **A interface do repository é declarada pelo service** (o que ele precisa), e o adaptador DynamoDB a implementa. O
  service nunca importa o SDK da AWS.
- **O repository fala a linguagem do domínio**, não "put/get" genérico: `SalvarLancamento(mudança)`, não `PutItem`. Assim
  as chaves (`PK`, `SK`, `TX#...`, `CFG#...`), o mapeamento entre `map[string]any` e os tipos do domínio e as expressões
  do DynamoDB **moram só nele**. É o único lugar onde uma chave pode aparecer, o que sustenta a regra de não criar chave
  sem discutir.
- **A transação é dele.** Gravar o registro e a versão do histórico juntos (`TransactWriteItems`) é detalhe de
  persistência: o service pede "salvar esta mudança com a sua versão" numa única operação. Se o service montasse duas
  chamadas, a atomicidade deixaria de existir.
- **Um repository por assunto** (lançamentos, configuração, fechamentos, histórico), em vez de um grande.
- Os exemplos das chaves existentes, com dados fictícios, estão em
  [`specs/modelo-de-dados/`](specs/modelo-de-dados/README.md).

## Avisos: um despachante, vários canais (fechada)

O service só sabe **o que avisar, para quem e com qual conteúdo**. Ele depende de uma interface `Avisador` e não sabe
qual canal está por trás.

| Pergunta | Quem responde |
|---|---|
| O quê e para quem avisar (qual lançamento vence, quem criou, o dono, o conteúdo) | **service** (regra de domínio) |
| Quando o gatilho dispara (todo dia às 8h) | **entrada** (agendador → controller), com o relógio injetado |
| Por onde e como entregar (formato, divisão da mensagem, retentativa) | **despachante e adaptadores** |
| Quais canais a pessoa assinou | **despachante** (lê as preferências da pessoa) |

- Cada canal (`Telegram`, `E-mail`, `WhatsApp`) é uma **strategy**: a própria interface `Avisador`.
- O `Despachante` também é um `Avisador`: lê os canais ligados da pessoa e chama as strategies correspondentes. O `main`
  injeta só ele no service; canal novo é um adaptador novo registrado no `main`.
- O core manda **conteúdo estruturado**, não texto formatado. O limite de 4096 caracteres do Telegram (limitação
  conhecida do item 28) vira problema do adaptador do Telegram.
- Falha de um canal não derruba os outros. Agrupamento (um resumo por pessoa por dia) e idempotência de entrega ficam no
  despachante, não no service.
- Sem canal assinado, o despachante simplesmente não entrega; o service não tem caso especial.

## Histórico imutável e versionado

**Regra de domínio (fechada):** toda edição, ocultação, exclusão e restauração gera uma **versão** com o registro inteiro,
quem fez e quando; **criar (e importar `id` novo) não gera versão** (ajuste de 2026-10-23: a versão só nasce na alteração). Restaurar uma versão antiga cria uma versão nova com aquele conteúdo; o passado nunca é reescrito.

Mecanismo (decisões do dono de 2026-10-15; o roteiro está em [`specs/historico/`](specs/historico/spec.md)). A **Fase 1**
está implementada para lançamentos:
- registro atual e versão gravados **na mesma transação** (`TransactWriteItems`, que o backend já usa), com checagem da
  versão esperada contra edição concorrente (parte do item 12);
- a interface `Repositorio` não tem método de editar nem apagar versão (há teste de reflexão); toda versão é gravada
  com `attribute_not_exists(SK)`;
- o histórico numa partição própria, para a role da Lambda poder ter `Deny` de `UpdateItem` e `DeleteItem` por IAM;
- **exclusão lógica**: o lançamento ganha `excluidoEm` (só nos excluídos), some das telas e fica na tabela e no
  histórico;
- escopo: lançamentos primeiro, depois configurações (`listas`, `cartoes`, `saldos`, `combos`, hoje sobrescritas por
  `PUT` do blob inteiro) e fechamentos;
- **sem migração**: a versão 1 nasce na primeira edição ou exclusão de um lançamento que ainda não tem histórico
  (estado de antes, autor desconhecido), junto com a versão 2. Desde 2026-10-23 isso vale para **todo** lançamento: criar e
  importar `id` novo não gravam versão.

O service tem **um único caminho de escrita**, que sempre grava registro e versão juntos. Como o controller só chega ao
dado pelo service, não existe atalho que pule o histórico.

**PITR** do DynamoDB ligado só em prod, como proteção complementar (`template.yaml`, 35 dias de retenção).

## Domínio: o que o dono explicou

### Cartão é um contexto próprio (fechada)
O cartão tem **lógica, tela e conceitos próprios**; não é um banco com outro nome.

- **O registro de `CREDITO` é um gasto real que será pago depois.** Ele aparece no Histórico e **conta como real**
  (entra nos gastos e nas tendências), mas **não afeta o saldo do banco**. Isso já está resolvido pelas caixas de
  status em Listas (`statusReal`, `afetaSaldoBanco`) e **não pode ser perdido**: é o controle de gastos.
- **A tela do cartão gera uma fatura a pagar**, e quem paga **não é necessariamente o banco da operadora**: o cartão pode
  ser do SANTANDER e a fatura sair pelo NUBANK. O banco pagador é uma escolha, não algo presumido a partir do cartão.
- **Cada valor conta uma vez.** O gasto conta quando nasce o `CREDITO`; o saldo muda quando a fatura é paga. O
  pagamento mexe no saldo do banco pagador e **fica fora do total de gastos**, para o mesmo gasto não ser contado duas
  vezes. Hoje isso é feito pelo status `CONTAS` (afeta saldo) com `excluirDoTotal`. O `CONTAS` também carrega a
  **transferência entre bancos** (dois lançamentos com o mesmo `transferParId`). Seu destino fica para o desmembramento
  do `PAGO` (ver "Futuros, status e saldo no servidor"); não entra na mudança de Futuros.
- Cartão e banco compartilham **só a ideia** de ponto de partida verificado, descrita abaixo; o resto (período, total,
  conferência, pagamento) é de cada um.

### Início do controle e fechamento verificado
Banco e cartão compartilham o princípio: **há um início, e o cálculo parte do último ponto verificado**, sem voltar ao
início sempre.

| | Banco | Cartão |
|---|---|---|
| Início do controle | `dataInicio` + `saldoInicial` (`CFG#SALDOS`): quanto havia naquela data | `inicio` AAAA-MM (`CFG#CARTOES`): primeira fatura controlada |
| Período | mês do calendário, fixo | **ciclo da operadora**; o fechamento varia por mês e é dado de entrada |
| Ponto verificado | mês fechado e conferido com o extrato | fatura conferida |
| Natureza | **acumulativa**: o saldo de um mês depende do anterior | **independente**: a fatura é a soma do seu período |
| Hoje | a **trava** é do backend (`fechamento.go`), mas a **cascata e o cache são calculados no navegador** e o servidor grava o que ele manda (`SalvarCaches`; no `Conferir` o servidor só compara dois números do cliente entre si) | tudo no front, sem validação nem trava no backend |

- O corte do início **não apaga nem altera dado**: o passado continua gravado e visível, só deixa de ser calculado ou
  conferido ("histórico"). A regra deve vir pronta pela API (`historico: true`), não de `ym < inicio` no front.
- **Nomes:** no banco é **fechamento**, no sentido da palavra. No cartão o termo é **fatura** (calculada, emitida,
  paga), porque a operadora emite um documento e o pagamento pode ser outro valor.
- **Decisão (fechada):** fatura conferida deve ser **travada pelo backend**, como o mês do banco. A trava precisa de uma
  forma de reabrir, porque a operadora pode lançar um estorno depois da conferência.

### Ajuste de fatura (`faturaAjuste`)
- `faturaAjuste` (`-1`, `0`, `+1`) é gravado **no lançamento** e vence qualquer padrão: **congela a decisão de fatura
  daquele registro, inclusive no passado**, pela data que o dono escolhe. É uma decisão deliberada, não um campo sem uso.
- Os status `CREDITO IN` e `CREDITO EX` vieram da planilha para tratar o que a operadora escolhia pôr ou não na fatura.
  Com `faturaAjuste` eles **deixam de ser necessários**. O nome do status só importa hoje como valor padrão para
  registros **sem** o campo.
- Em aberto: migração única que grave o ajuste explícito nos lançamentos antigos sem `faturaAjuste`, para o nome do
  status deixar de ser consultado. Mexe em dado: só depois de discutir.

### Os três valores da fatura
| Valor | Quem produz | Hoje |
|---|---|---|
| Calculado | a soma dos lançamentos do período (com `faturaAjuste`) | **calculado no servidor** (`GET /cartoes/{meio}/faturas/{ym}`, item 33a fase A; a aba Cartão ainda calcula no navegador até a fase B) |
| Da fatura | a operadora (app ou documento) | `faturas[ym].valorBanco`, digitado |
| Do boleto / pago | o que de fato sai da conta | um lançamento normal, de qualquer valor e em qualquer meio, que o dono pode **marcar** (`pagaFatura`) como pagamento da fatura; a marca é só uma label para achar depois (itens 33b e 35) |

Caso real que originou isto: um estorno entrou no dia do corte; a fatura (extrato do cartão) trouxe um valor e o
pagamento efetivo foi outro, menor. Nada ligava a fatura ao pagamento, e a explicação virou um lançamento de valor zero
(nota solta, sem efeito em nenhum cálculo).

Direção acordada, **detalhes em aberto**: o pagamento nasce **da fatura**, não da soma dos lançamentos. Ao gerar o
pagamento, o sistema propõe o valor da fatura (o da operadora, se informado; senão o calculado), o dono **escolhe o
banco pagador** (pode ser outro que não o do cartão) e confirma ou corrige para o valor do boleto. O lançamento do
pagamento fica no banco pagador, ligado àquela fatura, com a diferença visível como fato da fatura. Esse lançamento mexe
no saldo do banco pagador e não conta de novo como gasto. Pagamento **mínimo ou parcial** não é usado pelo dono, mas é o comum entre os usuários:
**fica no backlog** (item 35); isso transforma o vínculo em um-para-muitos.

## Saldo e totais calculados no servidor (fechada em 2026-10-18; não implementada)

**Achado.** O navegador calcula o saldo de cada banco (`calcularSaldoBanco`, cascata mensal), o total dos bancos, as
faturas do cartão, o resumo e o saldo por mês de Futuros e a soma do Histórico. O servidor grava o que recebe:
`SalvarCaches` aceita qualquer número finito para um mês passado e `Conferir` só confere se `saldo` e `real`, ambos do
cliente, são iguais entre si. Quem forja a requisição escolhe o saldo, e esse número vira o ponto de partida da cascata.
É uma brecha de segurança, não só de organização.

**Decisão.**
- O servidor calcula saldo, cascata, fechamento, total dos bancos, totais e saldo por mês de Futuros e do Histórico,
  fatura e o que é "fora do total". Usa o relógio e o fuso do servidor, nunca o do computador de quem usa.
- O cache dos meses encerrados passa a ser gravado só pelo servidor, a cada recálculo. `SalvarCaches` deixa de aceitar
  valor do cliente.
- `Conferir` recebe só o saldo do extrato (`real`). O servidor compara com o saldo que ele mesmo calculou; o cliente não
  manda mais o `saldo`.
- O servidor **rejeita** campo derivado vindo do cliente (saldo, cache, total, `excluirDoTotal`, o `previsto` calculado,
  qualquer resumo). O código de erro e a lista exata saem do spec.
- Um teste tenta forjar cada um desses campos e precisa falhar antes da mudança e passar depois.
- Calcular no front só vale para **exibir** (Tendências, projeções) e nunca volta para a API.

## Futuros, status e saldo no servidor (fechada em 2026-10-18; não implementada)

Só a aba **Futuros** entra nesta rodada. Decisões:
- **Marcador `previsto` no lançamento** substitui os status `PREVISTO`, `TRANSFERINDO` e `CREDITANDO`. O lançamento
  futuro nasce com o **status real** (`PAGO`, `CREDITO`...) e o marcador ligado. O marcador é um **campo novo no DynamoDB**:
  nome (`previsto`) e tipo (booleano) foram aprovados pelo dono em 2026-10-24 (princípio 3).
- **Enquanto o marcador existir, o lançamento fica fora do saldo e fora do total do Histórico**, mesmo já estando no
  Histórico (data chegada). Por isso o saldo só muda por escrita (confirmar, criar, editar), e o cache e a cascata do
  fechamento continuam válidos; nada muda à meia-noite sem uma gravação.
- **Cada aba recebe do servidor o que deve mostrar.** O front não olha status nem `statusFuturo`. **Futuros** traz todos
  os previstos, inclusive os vencidos, que ficam no grupo **Atrasados** como hoje (decisão de 2026-10-24). O **Histórico**
  só traz o que já aconteceu, nunca uma linha de previsto, e cada linha sai **resolvida pelas regras da data do evento**
  (ver "Status, meios de pagamento e o passado" abaixo).
- **Quem liga o marcador:** o servidor, em todo lançamento criado com data **depois de hoje**; o Prever e o "Repetir ao
  lançar" também criam previstos (decisão de 2026-10-24). Um lançamento com data de hoje ou do passado não é previsto.
- **Atraso** é o marcador ainda ligado antes de hoje. O servidor entrega o grupo Atrasados pronto (o front não calcula) e
  o mesmo critério alimenta o aviso do Telegram.
- **Confirmar** (demo aprovada em 2026-10-24, `demos/demo-confirmar-previsto.html`): ação da linha que abre uma janela
  com **Valor** (já vem o previsto, dá para trocar: multa e juros), **Data** e **Status** (já vem o do lançamento, dá para
  trocar). Confirmar tira o marcador, a linha sai de Futuros, entra no Histórico e passa a contar no saldo; o aviso de 6 s
  tem **Desfazer**, que no servidor é uma restauração. A data sugerida é a prevista; se ela cai em **mês já fechado**, a
  sugestão é hoje, voltar a uma data de mês fechado pede a confirmação extra do item 38, e mês conferido recusa. O valor,
  a data e o status chegam como **dado de entrada** da pessoa: o servidor valida e calcula o efeito no saldo.
- **Status são do backend.** `status`, `statusReal`, `afetaSaldoBanco` e `statusFuturo` deixam de ser editáveis pelo app
  e `statusFuturo` não vai mais ao front. `salvarListas` continua gravando o resto de `config/listas` (bancos, cores
  e `statusCor`, `estiloCor`, aparência e ordem dos saldos) e o servidor ignora qualquer mudança nos campos de status.
  **Regra do dono (2026-10-24): o front não trata status de forma nenhuma.** O backend **manda a lista de status** (no corte,
  `["PAGO", "CREDITO"]`; no futuro `PIX`, `DREX`, `DEBITO`...), e o front só a exibe, guarda a escolha da pessoa e a
  **devolve como veio** (o texto da opção, nunca um índice). Não há lista fixa no front (a de `app.js:5` deixa de existir), nem
  predicado por nome (`ehFuturo`, `afetaSaldo`, `ehCredito`...), nem cor ou regra escondida em nome. O texto do status de um
  lançamento antigo continua aparecendo como está gravado. Um teste de CI falha se um nome de status voltar a `web/`.
- **O backend manda pronto o que antes o front deduzia do status:** `previsto`, o que é cartão (`meios: [{nome, cartao, ativo}]` na
  lista de meios, sem chamada extra), `excluirDoTotal` (calculado pelo servidor), a fatura efetiva e o padrão do ajuste, e
  a transferência entre bancos (um comando que cria as duas pontas).
- **Lançamentos antigos** com `PREVISTO`, `TRANSFERINDO` ou `CREDITANDO` **ficam como estão** (eram só previsões, nada é
  regravado) e continuam em Futuros, os do passado em Atrasados, até alguém tratar. Ao confirmar um deles, a janela abre
  com o **primeiro status da lista do servidor** escolhido (hoje `PAGO`) e a pessoa troca se for o caso; os três nomes antigos não aparecem na lista de
  status da janela. Editar e excluir seguem como hoje. Não há migração em massa; o backend tem uma única função que
  responde "isto é futuro?".
- **Fora desta rodada:** o desmembramento do `PAGO` por tipo e meio (recebimento, débito, pix, drex, cripto), o destino
  do `CONTAS` e do `excluirDoTotal` como regra do servidor, e um papel de Admin (complemento ao dono).
- **Aberto para o spec:** como o servidor encontra todos os previstos sem depender dos 1000 lançamentos mais
  recentes (só se exigir chave nova no Dynamo volta ao dono); e o que o Prever faz com o "Status das cópias".

## Status, meios de pagamento e o passado (fechada em 2026-10-24; não implementada)

Três ideias separadas que estavam misturadas, e o que o dono decidiu:

| O quê | Hoje | Passa a ser | Onde mora |
|---|---|---|---|
| **Onde o dinheiro está** (conta, carteira, vale, cartão): `SODEXO-REF-IBM`, `DINHEIRO`, `CARTEIRA`... | o `banco` de cada lançamento (um vetor de nomes em `CFG#LISTAS`) | **meio de pagamento**, só um nome de exibição: um **de-para em `domain/`** (`MeioDe`/`BancoDe`); o Dynamo **não muda** | dado do espaço; o **dono** cria e desativa pelo app (item 43) |
| **Como a movimentação foi feita** (`PIX`, `DEBITO`, `DREX`, `CRIPTO`, `CREDITO`) | o `status` (`PAGO`, `CREDITO IN`...) | **forma de pagamento**, só um rótulo; o campo e a API continuam `status` | catálogo do **produto**, em código (muda por release) |
| **Já aconteceu ou é previsto** | `PREVISTO`, `TRANSFERINDO`, `CREDITANDO` | o marcador `previsto` | atributo do lançamento (item 41) |

`CRIPTO` e `DREX` são **formas** (são amplos: qual cripto, qual carteira); o saldo mora no meio onde o dinheiro está.

Regras:
- **O catálogo de status tem uma entrada por nome** (rótulo, cor, regras: conta no saldo, fora do total, crédito, ajuste padrão, previsto) e a marca
  **selecionável**. **Nenhum status sai dele**: o que deixa de ser usado vira não selecionável e continua valendo onde já está gravado, porque o
  histórico (desde 2016 ou antes) é lido pelas regras do status.
- **Não há data de corte global.** Cada coisa tem a sua vigência: um meio (`SODEXO` acaba quando o dono sair da IBM) tem o seu fim, e um status pode ter
  `desde` e `ate` **só para sugerir** a lista de escolha de uma data. A lista de escolha é do servidor; o front exibe, guarda a escolha e a devolve como veio.
- **O passado é do usuário.** Ele pode reclassificar um registro antigo (um `PAGO` de 2018 pode virar `PIX` ou `DEBITO`): o servidor aceita qualquer
  status selecionável em qualquer data. O que protege é a **trava de data pelo efeito no saldo** (trocar `PAGO` por `PIX`, que contam igual, não é
  travado; trocar para um que não conta, ou mexer em valor, tipo, meio, data, oculto ou previsto, é) e o **histórico de versões** (toda troca é auditável e
  reversível).
- **Troca em massa:** não há ferramenta de admin (item 44 removido em 2026-10-27); o passado se reclassifica um registro por vez pelo app.
- **Ao ler**, o Histórico sai do servidor pronto: cada linha com rótulo e cor, a **lista de filtros de status que aparecem no período consultado** (um 2019
  mostra `CREDITO IN`; um 2018 reclassificado mostra `PIX`) O **total da lista e a soma por mês saem da tela até o item 42** (decisão do dono, 2026-10-24).

Spec: [`specs/futuros-previsto/`](specs/futuros-previsto/spec.md), regras R9 a R12c e R15.

## Ordem de execução proposta
1. Histórico imutável de lançamentos, dentro da nova estrutura de camadas (item 31 sobre 32).
2. **Saldo, fechamento e conferência calculados e validados no servidor (item 40)**, que fecha a brecha descrita acima. **Revisto em
   2026-10-24:** é o primeiro da fila, porque o front só deixa de tratar status quando para de somar. Spec em
   [`specs/saldo-servidor/`](specs/saldo-servidor/spec.md).
3. **Fatura do cartão calculada no backend (item 33, fatia a)**; a trava de fatura conferida e o cartão como entidade própria
   ficam na fatia b. Spec em [`specs/fatura-servidor/`](specs/fatura-servidor/spec.md).
4. **Futuros, marcador `previsto` e status do backend (item 41)**, com a lista de status mandada pelo servidor e o front sem
   status. Spec em [`specs/futuros-previsto/`](specs/futuros-previsto/spec.md). (Em 2026-10-18 isto estava no item 33 com as caixas
   de Listas; mudou.)
5. Avisos pelo despachante e canais (item 34).
6. Pagar fatura com valor do boleto e vínculo (item 35).
7. Remover `financas-app.html` (item 36; feito em 2026-10-26).

Cada passo é uma fatia vertical (front, controller, service, domain, adapter) de uma regra por vez, com o app seguindo
no ar. Nenhum passo cria chave no DynamoDB sem aprovação do dono.
