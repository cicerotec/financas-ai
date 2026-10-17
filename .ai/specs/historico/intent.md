# Intent: histórico de versões e recuperação

Primeira feature da nova arquitetura ([`arquitetura.md`](../../arquitetura.md), itens 31 e 32 do backlog). Este
documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro (`spec.md`) vêm depois, depois de o dono aprovar o
intent.

**Estado:** aprovado pelo dono (decisões fechadas em 2026-10-13 e ajustadas em 2026-10-15; veja `plan.md` e `spec.md`).
As fases 0 a 2 estão implementadas e verificadas em dev; a 3 (tela) está implementada e falta conferir no app de dev.

## Por quê

Em 2026-10-10 uma edição sobrescreveu um registro e o dado antigo se perdeu: o `PUT` substitui o item inteiro e o PITR
do DynamoDB (só em prod, janela de 35 dias) não mostra "o que o registro foi". Hoje não há como saber o que um registro
foi, quem o mudou nem voltar atrás.

## O que queremos

1. **Todo evento de escrita de lançamento vira uma versão no histórico.** Criar, editar, ocultar, excluir e restaurar
   gravam, junto com o registro atual, uma versão com o **registro inteiro**, quem fez e quando.
2. **Cada versão diz o que ocorreu**, num campo de evento (`acao`):

   | `acao` | Quando |
   |---|---|
   | `insert` | o registro nasceu (versão 1) |
   | `update` | o registro mudou (inclui ocultar, ajustar fatura e mudar a data, que são edições) |
   | `delete` | o registro foi excluído (exclusão lógica: some das telas, fica na tabela e no histórico) |
   | `restore` | uma versão antiga foi recuperada (aponta qual versão originou) |

3. **O histórico nunca perde registro.** Só se acrescenta: não há método para editar nem apagar versão, e a role da
   Lambda tem `Deny` de `UpdateItem` e `DeleteItem` sobre a partição do histórico.
4. **Uma tela permite recuperar um registro.** Ela lista as versões de um registro, mostra o que mudou entre versões e
   deixa **restaurar** uma versão. Restaurar **grava uma versão nova** com aquele conteúdo (`acao = restore`); o passado
   não é reescrito e a versão restaurada continua lá.

## Decisões

### Modelo (chave nova, aprovada pelo dono)
- **Partição própria `HIST#<sid>`**, para o `Deny` por IAM (a condição `dynamodb:LeadingKeys` só enxerga a `PK`).
- **`SK = TX#<id>#<versão com zeros à esquerda>`** (ex.: `TX#abc...#00000007`). A versão é indexada pelo **`id`**, não
  pela chave viva, porque a chave do lançamento (`TX#<dataEvento>#<id>`) muda quando a data é editada. A linha do tempo
  do `id` não se parte.
- **Atributos:** registro inteiro, `acao`, `versao`, `porSub`, `em` e `deVersao` (só no `restore`).
- **Número da versão** = maior existente + 1, gravado com `attribute_not_exists(SK)`: duas edições concorrentes disputam
  a mesma versão e a segunda falha. É a checagem de concorrência, **sem campo novo no item vivo**.
- **Sem** índice (GSI), sem campo de versão no item vivo e sem TTL.
- **Consulta:** as versões de um lançamento (`Query` por `PK` com `begins_with(SK, "TX#<id>#")`) e, na tela, a
  **busca por mês de alteração** (decisão do dono, 2026-10-15): `GET .../historico?mes=AAAA-MM&q=`, que lê a partição
  `HIST#` e filtra por `em` em Go, com teto de 5.000 versões. O **GSI por data** entra depois, sem mudar a rota.
- **Retenção:** versão guarda o **registro inteiro**. Analisar mais tarde a saída do histórico antigo para um S3.

### Exclusão
- **Exclusão lógica**, como em `arquitetura.md` e no item 31 do backlog. Excluir não apaga o item vivo: marca-o como
  inativo e grava a versão `delete`. O excluído some das telas e continua na tabela e no histórico.
- Isso exige um **indicador novo no item vivo**: **`excluidoEm`** (data e hora ISO), gravado **só nos excluídos**; sem
  o campo, o lançamento está vivo e não há migração. Nome fechado pelo dono em 2026-10-15 (substitui o `ativo`
  provisório, que exigiria valor nos vivos). É campo novo no DynamoDB e entra no modelo de dados na mesma entrega.
- As consultas das **rotas novas** devolvem só os ativos. As rotas atuais não mudam de contrato (veja "Rotas").
- Continua só do owner, como hoje.

### Rotas
- **Decisão do dono (2026-10-15): sem rotas paralelas de CRUD.** `/tx`, `PUT` e `DELETE` seguem como estão, com o
  mesmo contrato, e **sempre passam pelo service**. Só as rotas de histórico são novas (versões, restaurar, lista de
  excluídos), e o `GET /tx` esconde os excluídos. (Texto anterior: o front migra tela por
  tela para as novas; descartado.)
- Cada rota nova exige **ação nova em `auth/authz.go`, com teste** (o que não está liberado é negado).
- **Sem buraco no histórico:** toda rota de escrita passa pelo mesmo caminho de escrita do
  service e grava a versão. O `DELETE` passa a fazer exclusão lógica por dentro, mantendo a resposta que o front
  já espera.

### Restauração e a regra de negócio
- **O histórico nunca briga com a regra de negócio.** Se a trava do fechamento não permite a escrita, a restauração
  fica bloqueada até a regra permitir (reabertura do mês, do mais recente para o mais antigo). A versão fica gravada e
  visível, sem possibilidade de retorno enquanto a trava valer. Fechamento conferido é absoluto: sem atalho nem exceção.
- **Restaurar é uma escrita normal**, pelo mesmo caminho do `putTx` (versão de um lançamento vivo) ou do `postTx`
  (reativar um excluído, mesmo `id` e mesma chave), com as mesmas checagens (trava, `invalidarSaldos`, tags, aviso, `seq`).
- **A tela mostra "restaurar ou não" por versão**: a API devolve `restauravel` e o motivo (ex.: "mês 2026-08 do NUBANK
  está conferido; reabra na aba Saldos"). O front não decide nada.
- **`criadoPor` e `criadoEm` voltam da versão**, não de quem restaurou; quem restaurou fica em `porSub`.
- Restaurar um lançamento que já está ativo na versão pedida não faz nada e responde `409`.

### Permissões (ação nova em `auth/authz.go`, com teste)
- **Owner:** vê todas as versões e restaura qualquer uma.
- **Member:** vê o histórico dos lançamentos que **criou** (incluindo as edições do owner neles) e restaura só versões de
  lançamentos que criou **e que ainda existem**, a mesma regra do `tx.editar`.
- **Restaurar um excluído** é só do owner, porque excluir é só do owner.

### Transação
- **Na mesma `TransactWriteItems`:** o lançamento vivo (put, com o indicador de ativo na exclusão lógica), a versão em
  `HIST#` e, quando a data muda, o delete da chave antiga (mecânica da mudança de chave, não uma exclusão do registro). Mudar a data gera **uma** versão (`update`), não duas.
- **Falha ao gravar a versão aborta a edição inteira** (tudo ou nada).
- **`SALDO#<banco>` fica fora da transação.** Ele usa controle otimista com retentativa; dentro da transação, uma disputa
  derrubaria o lançamento e a versão. O cache é derivado e já se protege pela `version`. Hoje, se a invalidação falha, só
  há log: é **dívida do fechamento mensal**, anotada aqui, fora do escopo do histórico.
- `seq`, tags e aviso também ficam fora (efeitos derivados).

### Tela
- **Aba Configurações**, com itens como Listas, Importações, Exportações e **Histórico de versões**. O nome "Histórico"
  continua sendo a lista de lançamentos reais; a nova entrada fica dentro de Configurações, para não duplicar o nome.
- Demo aprovada antes do código (guardrails), celular primeiro, seletores nativos.

### PITR
- Já está ligado em prod (35 dias). Em dev segue desligado, de propósito.

## Limites (o que fica de fora)

- Não é backup nem substitui o PITR; é a trilha de versões do app.
- Não edita nem apaga versões, nem por admin.
- **Só lançamentos** nesta entrega. Configurações (`CFG#*`), fechamentos (`SALDO#*`), perfil, aparência e avisos ficam
  para fatias seguintes, cada uma discutida antes.
- Não reconstrói saldo ou fatura: restaurar um lançamento só devolve o lançamento; os cálculos derivados seguem as
  regras do backend a partir do registro novo.
- Não guarda segredo: o token do Telegram está no SSM, fora da tabela.

## Lacuna conhecida: a trava de fatura do cartão

Hoje a trava do backend protege só o **saldo do banco** (`mudaSaldo` + `conferidos` em `SALDO#`). A conferência de
**fatura** do cartão existe só no front, sem trava no backend (item 33). Foi assim que uma edição de lançamento de
crédito, numa fatura já conferida, alterou um valor fechado e bagunçou os eventos futuros.

Como o histórico herda a regra de negócio, **enquanto o item 33 não existir, restaurar um lançamento de cartão numa
fatura conferida não é bloqueado**. O histórico ajuda a desfazer o estrago (a versão anterior está gravada), mas não
impede. Quando a trava de fatura entrar no backend, a restauração passa a respeitá-la sem mudança neste recurso.
Dependência: item 33.

## Regras gerais que valem

- **Um único caminho de escrita:** o service grava registro e versão juntos; o controller não chega ao banco sem ele.
- **O repository fala o domínio** (`SalvarLancamento(mudança)`), e só ele conhece chaves e expressões do DynamoDB. A
  interface não tem método de editar nem apagar versão.
- **O front não decide:** a lista de versões, o diff e se uma versão pode ser restaurada chegam prontos pela API.
- **Dado real nunca no git:** exemplos e fixtures só fictícios.

## Sem migração

**Decisão do dono (2026-10-15): não há migração.** Os lançamentos já gravados são do passado e continuam como estão. O
histórico começa quando a escrita nova entra no ar. Na **primeira edição ou exclusão** de um lançamento sem versões, o
service grava, na mesma transação, a **versão 1** com o estado de antes (`acao = insert`, autor desconhecido) e a
versão 2 com o estado de depois. Nenhuma gravação em massa.

## Pronto quando

- Toda escrita de lançamento passa por um único caminho que grava registro e versão juntos, com teste que falha se a
  versão não for gravada.
- Editar ou excluir um lançamento sem versões grava a versão 1 (estado de antes) e a versão 2 na mesma transação.
- Nenhum código nem policy permite editar ou apagar uma versão (teste no repository e `Deny` no `template.yaml`).
- A tela lista as versões, mostra o que mudou e restaura criando versão nova, em celular e computador, nos dois temas;
  uma versão bloqueada pela trava aparece desabilitada, com o motivo.
- `security.md` (ações novas), o modelo de dados (`specs/modelo-de-dados/`, com a chave `HIST#`
  e um exemplo fictício) e o BACKLOG ficam em dia na mesma entrega.
