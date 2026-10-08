# Finanças da Família

App de finanças pessoais e da família: lançamentos, saldos por banco, faturas de cartão, eventos futuros e tendências.
Duas pessoas usam o mesmo espaço (dono e membro), pelo navegador do computador ou do celular (instalado na tela inicial).

- **Front:** uma página HTML com CSS e JavaScript dentro (`web/index.html`), sem framework e sem etapa de build. Única
  biblioteca externa: SheetJS (xlsx), via CDN, só para importar planilhas.
- **Backend:** AWS Lambda em Go com Function URL, DynamoDB e login com Cognito (Hosted UI + PKCE).
- **Hospedagem do front:** S3 privado + CloudFront (HTTPS).
- **Região:** `sa-east-1`. Infra descrita em `template.yaml` (AWS SAM).

Roadmap, decisões e o fluxo de versões estão em [`BACKLOG.md`](BACKLOG.md). A proposta de colaboração em tempo real está
em [`docs/colaboracao-tempo-real.md`](docs/colaboracao-tempo-real.md).

## Como as peças se encaixam
```
navegador ──(HTTPS)──> CloudFront ──> S3 privado           (web/: index.html, api.js, config.js)
    │
    ├──(login)──> Cognito Hosted UI (PKCE) ──> token JWT
    └──(token no cabeçalho)──> Lambda Function URL (Go) ──> DynamoDB (tabela FinancasApp)
```
A Lambda valida o token do Cognito, descobre o papel da pessoa no espaço (`owner` ou `member`), confere a permissão e só
então lê ou grava. `web/api.js` expõe ao front uma API no estilo `collection/doc/where/orderBy/limit/onSnapshot` e a
traduz para as rotas da Lambda.

## Estrutura do repositório
| Pasta ou arquivo | O que tem |
|---|---|
| `web/` | O app. `index.html` (telas e lógica), `api.js` (cliente da API e do login), `config.example.js` (modelo; `config.js` fica fora do git) |
| `backend/` | Lambda em Go: `main.go` (rotas), `authz.go` (permissões por papel), `store.go` (DynamoDB), `auth.go` (token), `tags.go`, `perfil.go` (apelido e chat id do Telegram), `avisos.go` e `telegram.go` (avisos diários de lançamentos futuros), `fechamento.go` (fechamento mensal dos saldos), `seq.go` (contador de alterações) e os testes `*_test.go` |
| `template.yaml` | Infra (SAM): tabela, Cognito, Lambda, bucket e distribuição do CloudFront |
| `scripts/` | Deploy, sessão com MFA, publicação do front, convite de usuários e manutenção (ver abaixo) |
| `docs/` | Documentos de projeto |
| `financas-app.html` | Versão original em arquivo único, de quando o app rodava no ambiente do Claude. Mantida como referência; o app atual é o de `web/` |
| `BACKLOG.md` | Roadmap por área, decisões, detalhes e fluxo de branches e versões |

## Rodando no seu computador
Precisa de um front servido em `http://localhost:8080` (o login e o CORS aceitam essa origem) e de uma stack já criada na AWS.
1. Copie `web/config.example.js` para `web/config.js` e preencha com os Outputs do deploy: `ApiUrl`, `ClientId` e `LoginDomain`.
2. Sirva a pasta `web/` na porta 8080, por exemplo: `python -m http.server 8080 --directory web`.
3. Abra `http://localhost:8080` e entre com o login do Cognito.

Testes do backend: `cd backend && go test ./...` (usam um DynamoDB em memória, não tocam a AWS).

## Publicando
Tudo roda no PowerShell, com a sessão AWS aberta por MFA. Os scripts de publicação avisam quando você não está na `main`,
tem alterações sem commit ou está atrás do GitHub, porque publicam **o que está na sua pasta**, não o que está no GitHub.

```powershell
. .\scripts\aws-mfa.ps1 <codigo>        # sessão com MFA (vale 12 h, só neste terminal)
.\scripts\deploy.ps1                    # sam build + sam deploy (backend e infra); -Sim não pergunta o changeset
.\scripts\publicar-front.ps1            # envia web/ ao S3 e invalida o CloudFront; -Simular só mostra
```
Outros scripts: `seed.sh` (cria usuários no Cognito e o espaço), `definir-nome.sh` (nome de exibição),
`limpar-espaco.py` (apaga lançamentos e configurações de um espaço; simula por padrão).

## Branches e versões
Trabalho em branches a partir da `develop` (`feature/...`, `docs/...`) com PR para a `develop`. Uma versão sai com PR
`develop → main`, tag anotada `vX.Y.Z` e release no GitHub. Publicar na AWS é um passo separado, feito a partir da `main`.
Detalhes e comandos em "Fluxo de branches e versões" no [`BACKLOG.md`](BACKLOG.md).

## Papéis e permissões
| Ação | owner | member |
|---|---|---|
| Criar e copiar lançamento | sim | sim |
| Editar, ocultar e ajustar fatura | todos | só os criados por ela (`criadoPor`) |
| Excluir lançamento | sim | não |
| Criar tag | sim | só tags novas |
| Renomear ou excluir tag, combos, bancos, status, flags | sim | não |
| Ver saldos, cartão, tendências e futuros | sim | sim (só leitura) |
| Saldo inicial, fechamento, conferir fatura, início do controle | sim | não |
| Conferir e reabrir mês de um banco (fechamento mensal) | sim | não (só vê) |
| Cores, layout e o próprio nome | sim | sim, só os dela |
| Importar, exportar e limpeza | sim | não |

A regra está em `backend/authz.go` e tem testes; o que não está liberado explicitamente é negado.

## Modelo de dados

### No DynamoDB (tabela `FinancasApp`, chaves `PK` e `SK`)
| Item | PK | SK |
|---|---|---|
| Lançamento | `SPACE#<id>` | `TX#<dataEvento>#<id>` |
| Configuração compartilhada | `SPACE#<id>` | `CFG#LISTAS`, `CFG#TAGS`, `CFG#COMBOS`, `CFG#CARTOES`, `CFG#SALDOS` |
| Fechamento mensal de um banco | `SPACE#<id>` | `SALDO#<banco>` |
| Contador de alterações do espaço | `SPACE#<id>` | `SEQ` |
| Dados do espaço | `SPACE#<id>` | `META` |
| Vínculo da pessoa com o espaço (papel, apelido, `telegramChatId`) | `USER#<sub>` | `SPACE#<id>` |
| Aviso ativo de um lançamento (auxiliar da rotina diária) | `AVISOS#ATIVOS` | `<espaço>#<id do lançamento>` |
| Aparência da pessoa | `SPACE#<id>` | `USER#<sub>#CFG#APARENCIA` |

### Lançamento
| campo | tipo | observação |
|---|---|---|
| `status` | texto | um dos status de `config/listas` (PAGO, PREVISTO, CRÉDITO IN, etc.) |
| `tipo` | texto | `"saida"` (gasto) ou `"entrada"` (receita) |
| `valor` | número | sempre positivo; o sinal vem de `tipo` (zero e negativo são aceitos para nota e estorno) |
| `descricao` | texto | |
| `nota` | texto | opcional |
| `dataEvento` | texto ISO 8601 (UTC) | data e hora do evento; ordena a lista |
| `banco` | texto | um dos bancos de `config/listas` |
| `tags` | lista de textos | sem hierarquia |
| `excluirDoTotal` | booleano | verdadeiro para status CONTAS e TRANSFERINDO |
| `oculto` | booleano | some do histórico e não entra em saldos |
| `criadoEm`, `criadoPor` | texto | quando e por quem (`sub` do Cognito); o servidor preenche |
| `transferParId` | texto | liga as duas pontas de uma transferência entre bancos |
| `faturaAjuste` | -1, 0 ou 1 | só em crédito: 1 = fatura seguinte à da data; -1 = anterior; ausente = pela data |
| `valorEstimado` | booleano | só em crédito: valor ainda estimado (ex.: compra em dólar) |
| `aviso` | objeto | opcional, em evento futuro: `{ dias: [3,1,0], insistir: true }` avisa no Telegram nos dias combinados antes do vencimento e, com `insistir`, todo dia depois dele (ver [`docs/avisos.md`](docs/avisos.md)) |

### Configurações
- `listas`: `status` e `banco` (listas de textos); `statusReal` (status → conta como real; só `false` exclui);
  `afetaSaldoBanco` (só `true` afeta o saldo do banco); `statusFuturo` (status → evento futuro: não conta como real,
  não mexe no saldo e aparece em Futuros); `statusCor`; `estiloCor`, `layoutSaldos`, `ordemSaldos`, `ordemManual`
  (aparência).
- `tags`: `{ usadas: [textos] }`.
- `combos`: `{ lista: [ { tags: [textos] } ] }`. Combos sugeridos não são guardados: o app os recalcula contando
  conjuntos de tags repetidos nos lançamentos.
- `cartoes`: `{ porBanco: { "<banco>": { diaPadrao, inicio, faturas: { "AAAA-MM": { fechamento, valorBanco, conferida } } } } }`.
  Estar aqui marca o banco como cartão. `AAAA-MM` é o mês em que a fatura fecha.
- `saldos`: `{ porBanco: { "<banco>": { saldoInicial, dataInicio } } }`.

Escritas de configuração usam um campo `version` para detectar edição concorrente (conflito devolve erro em vez de
sobrescrever em silêncio).

## Regras de negócio
- **Saldo de um banco** = `saldoInicial` + soma dos lançamentos do banco que não estão ocultos, têm status com
  `afetaSaldoBanco` e `dataEvento` a partir de `dataInicio`. Entrada soma e saída subtrai.
- **Fechamento mensal.** O saldo é uma cascata: cada mês abre com o fechamento do anterior, desde `dataInicio`. O
  servidor guarda por banco (`SALDO#<banco>`) o fechamento calculado dos meses encerrados (`caches`, só para não reler
  o histórico inteiro) e os meses conferidos com o extrato (`conferidos`). Quem calcula é o navegador, porque só ele
  sabe quais status mexem no saldo; o saldo atual parte do último fechamento válido e lê só os meses seguintes.
  - Qualquer escrita em lançamento que mexa no saldo (criar, excluir, ou mudar valor, tipo, status, banco, data ou
    ocultar) marca `invalidoDe` e muda a `version` do banco; um cache calculado antes da escrita é recusado (409).
    Editar descrição, nota ou tags não invalida nada.
  - **Conferir** um mês (só o dono) exige que o saldo calculado bata com o extrato; a diferença se resolve com um
    lançamento de ajuste criado pela própria tela. O mês conferido e todos os anteriores ficam **travados**: o servidor
    recusa (409, `mes_conferido`) criar, alterar ou excluir lançamento que mexa no saldo desse banco nesses meses,
    inclusive na importação. Meses anteriores ao da `dataInicio` do banco são histórico (não entram no saldo): nunca
    travam nem invalidam o cache, então dá para importar anos anteriores com meses já conferidos. **Reabrir** só vale
    para o mês conferido mais recente.
  - Mês é sempre o do fuso de Brasília (`America/Sao_Paulo`). O cache também vale só para o mesmo saldo inicial, data
    de início e conjunto de status que mexem no saldo (muda um deles, o cache é descartado e refeito).
- **Transferência entre bancos** cria dois lançamentos `CONTAS` (saída na origem, entrada no destino) com o mesmo
  `transferParId` e a data escolhida.
- **Faturas do cartão:** a fatura que fecha no mês M cobre as compras de crédito do dia seguinte ao fechamento da anterior
  até o fechamento de M, inclusive. `faturaAjuste` empurra a compra para a fatura vizinha. Crédito = status que começam
  com CRÉDITO, não ocultos e que contam como reais; estorno (`entrada`) subtrai.
- **Conferência da fatura fechada:** diferença = total calculado − `valorBanco`. O app sugere registros (1 a 3) até 5 dias do
  corte cuja soma é igual à diferença. A fatura conferida avisa se mudar depois.
- **Eventos futuros:** status marcados como futuros ficam fora do Histórico e do saldo e aparecem na aba Futuros, com
  aviso de atrasados.
- **Duplicado ao salvar:** mesma combinação de status, descrição, valor, banco e `dataEvento`.
- **Remover status ou banco:** status em uso migra os lançamentos para outro status antes de sair; banco em uso não pode
  ser removido.

## Backup e importação
**Exportar dados** (aba Importar, só dono) gera um `.json` versão 2 com `colecoes.lancamentos` e `documentos` das
configurações; **Importar backup** o lê de volta em lotes. Também importa `.xlsx` (lançamentos, notas e fechamentos de
fatura). Há ainda um backup do próprio DynamoDB previsto no [`BACKLOG.md`](BACKLOG.md).

**Importação da planilha** (aba Importar): depois da prévia, **Confirmar importação** envia as linhas em lotes de 25 pelo
mesmo endpoint de lote do backup (`POST /tx/batch`). Cada linha recebe um id gerado na tela, então reenviar um lote nunca
duplica. A tela mostra `N/total (%)` e o tempo decorrido, que continuam andando enquanto espera o banco liberar
capacidade; falha de rede ou erro 5xx é tentada de novo automaticamente. Listas (bancos, status) e tags novas são gravadas
**antes** das linhas. Se a importação parar no meio, a prévia é descartada e a mensagem diz onde parou: processar o mesmo
arquivo de novo importa só o que faltou (o que já está no banco é ignorado).

**Verificação arquivo x banco de dados** (aba Importar): depois de importar, e a qualquer momento com o arquivo já
processado (botão *Conferir arquivo com o banco de dados*), o app lê de volta o que está gravado no mesmo período e
bancos e compara com o arquivo: quantidade, entradas e saídas. A tabela tem uma linha por **ano e banco**, recolhida;
tocar nela abre os **meses** (mês pela data do lançamento, no fuso de Brasília), e quem tem divergência já vem aberta. O
que está no arquivo e não está no banco aparece linha a linha; o que já existia no banco fora do arquivo só é contado. Na planilha a comparação é por
status, descrição, valor, banco, data e tipo; no backup `.json`, pelo id (e acusa dados diferentes). Linhas idênticas
repetidas no arquivo contam como o mesmo lançamento: o app não grava duplicados (chave igual entra uma vez só, tanto
na importação quanto na verificação, que avisa quantas linhas repetidas havia). Linhas sem data ou valor, ignoradas na
leitura, são contadas à parte.

## Sincronização da tela (contador `seq`)
O front mantém um cache dos 1000 lançamentos mais recentes. Cada escrita de lançamento (criar, editar, excluir,
importar) soma 1 no item `SEQ` do espaço e devolve o valor novo em `_seq`.
- **Escrita própria:** a tela ajusta só aquele registro no cache e o reentrega, sem reler o banco. Se o `_seq` que volta
  é o anterior mais 1, ninguém gravou no meio; se pulou, outra pessoa gravou e o cache é recarregado.
- **Checagem periódica** (a cada 3 minutos com a aba visível, e ao voltar para a aba): lê só `GET /spaces/{sid}/seq`
  (1 leitura). Só recarrega os 1000 lançamentos se o número mudou ou não deu para lê-lo.
- O `seq` é um número do DynamoDB (até 38 dígitos) e trafega como número JSON, sem conversão para inteiro de 32 bits.
  Não há log de "o que mudou": quando o número muda, o cache é recarregado inteiro (o log está no BACKLOG, item 12).

## Limitações conhecidas
- O front carrega os 1000 lançamentos mais recentes; meses mais antigos são buscados sob demanda pela navegação do
  Histórico. O saldo de cada banco parte do último fechamento mensal e lê só os meses seguintes; as faturas do cartão
  e as tendências ainda são calculadas no navegador, lendo o histórico (mover isso para a Lambda está no backlog).
- A tela só vê o que outra pessoa gravou na checagem periódica (a cada 3 minutos, com a aba visível, e ao voltar para a
  aba); colaboração em tempo real é uma proposta em `docs/`.
