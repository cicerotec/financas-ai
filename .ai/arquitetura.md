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

**Regra de domínio (fechada):** toda criação, edição, ocultação e exclusão gera uma **versão** com o registro inteiro,
quem fez e quando. Restaurar uma versão antiga cria uma versão nova com aquele conteúdo; o passado nunca é reescrito.

Esboço do mecanismo, **a discutir antes de qualquer chave** (princípio 3):
- registro atual e versão gravados **na mesma transação** (`TransactWriteItems`, que o backend já usa), com checagem da
  versão esperada contra edição concorrente (parte do item 12);
- a interface `Repositorio` só expõe anexar versão; não existe método de editar ou apagar histórico;
- o histórico numa partição própria, para a role da Lambda poder ter `Deny` de `UpdateItem` e `DeleteItem` por IAM;
- **exclusão lógica** (some das telas, fica no histórico);
- escopo: lançamentos primeiro, depois configurações (`listas`, `cartoes`, `saldos`, `combos`, hoje sobrescritas por
  `PUT` do blob inteiro) e fechamentos;
- migração: o estado atual de cada registro vira a versão 1.

O service tem **um único caminho de escrita**, que sempre grava registro e versão juntos. Como o controller só chega ao
dado pelo service, não existe atalho que pule o histórico.

Em aberto: ligar o **PITR** do DynamoDB já, como proteção até o histórico entrar (hoje `PointInTimeRecoveryEnabled:
false` e o item 11 do backlog está adiado); confirmar a exclusão lógica.

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
  vezes. Hoje isso é feito pelo status `CONTAS` (afeta saldo) com `excluirDoTotal`.
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
| Hoje | cascata, cache, conferência e **trava** no backend (`fechamento.go`) | conferência só no front, sem validação nem trava no backend |

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
| Calculado | a soma dos lançamentos do período (com `faturaAjuste`) | calculado no front |
| Da fatura | a operadora (app ou documento) | `faturas[ym].valorBanco`, digitado |
| Do boleto / pago | o que de fato sai da conta | lançamento do banco criado à mão, sem ligação com a fatura |

Caso real que originou isto: um estorno entrou no dia do corte; a fatura (extrato do cartão) trouxe um valor e o
pagamento efetivo foi outro, menor. Nada ligava a fatura ao pagamento, e a explicação virou um lançamento de valor zero
(nota solta, sem efeito em nenhum cálculo).

Direção acordada, **detalhes em aberto**: o pagamento nasce **da fatura**, não da soma dos lançamentos. Ao gerar o
pagamento, o sistema propõe o valor da fatura (o da operadora, se informado; senão o calculado), o dono **escolhe o
banco pagador** (pode ser outro que não o do cartão) e confirma ou corrige para o valor do boleto. O lançamento do
pagamento fica no banco pagador, ligado àquela fatura, com a diferença visível como fato da fatura. Esse lançamento mexe
no saldo do banco pagador e não conta de novo como gasto. Pagamento **mínimo ou parcial** não é usado pelo dono, mas é o comum entre os usuários:
**fica no backlog** (item 35); isso transforma o vínculo em um-para-muitos.

## Ordem de execução proposta
1. Histórico imutável de lançamentos, dentro da nova estrutura de camadas (item 31 sobre 32).
2. Fatura do cartão calculada no backend, com a trava de fatura conferida (item 33).
3. Regras de futuro, real e afeta saldo aplicadas pelo backend, usando as caixas que já existem em Listas (sem tipo
   novo de status) (item 33).
4. Avisos pelo despachante e canais (item 34).
5. Pagar fatura com valor do boleto e vínculo (item 35).
6. Remover `financas-app.html`, que só o README cita (item 36).

Cada passo é uma fatia vertical (front, controller, service, domain, adapter) de uma regra por vez, com o app seguindo
no ar. Nenhum passo cria chave no DynamoDB sem aprovação do dono.
