# Finanças da Família

App de finanças pessoais e da família: lançamentos, saldos por banco, faturas de cartão, eventos futuros e tendências.
Duas pessoas usam o mesmo espaço (dono e membro), pelo navegador do computador ou do celular (instalado na tela inicial).

- **Front:** uma página HTML (`web/index.html`) com o CSS em `web/css/app.css` e o JavaScript em `web/js/`, sem framework e sem etapa de build. Única
  biblioteca externa: SheetJS (xlsx), via CDN, só para importar planilhas.
- **Backend:** AWS Lambda em Go com Function URL, DynamoDB e login com Cognito (Hosted UI + PKCE).
- **Hospedagem do front:** S3 privado + CloudFront (HTTPS).
- **Região:** `sa-east-1`. Infra descrita em `template.yaml` (AWS SAM).

**Quem constrói com IA começa por [`AGENTS.md`](AGENTS.md)** (e pelo [`.ai/guardrails.md`](.ai/guardrails.md), lido inteiro).

Roadmap, decisões e o fluxo de versões estão em [`BACKLOG.md`](BACKLOG.md). A proposta de colaboração em tempo real está
em [`.ai/colaboracao-tempo-real.md`](.ai/colaboracao-tempo-real.md).

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
| `web/` | O app. `index.html` (telas), `css/app.css` (estilos), `js/app.js` (lógica), `js/ambiente.js` (faixa do ambiente de teste), `api.js` (cliente da API e do login), `config.example.js` (modelo; `config.js` fica fora do git) |
| `backend/` | Lambda em Go, em pacotes por responsabilidade: `domain/` (tipos e regras puras), `auth/` (token do Cognito e permissões por papel), `repository/` (DynamoDB), `service/` (regra de uso), `controller/` (HTTP). Na raiz ficam `main.go` (liga as peças e roteia) e o que ainda não migrou: `store.go`, `fechamento.go` (fechamento mensal dos saldos), `tags.go`, `perfil.go`, `avisos.go` e `telegram.go` (avisos diários), `seq.go` (contador de alterações) e `efeitos.go`. Os testes ficam ao lado do código (`*_test.go`). Detalhes em [`.ai/arquitetura.md`](.ai/arquitetura.md) |
| `template.yaml` | Infra (SAM): tabela, Cognito, Lambda, bucket e distribuição do CloudFront |
| `scripts/` | Deploy, sessão com MFA, publicação do front, convite de usuários e manutenção (ver abaixo) |
| `.ai/` | Documentos de projeto e contexto para quem constrói com IA: regras (`guardrails.md`), arquitetura, padrão de telas e demos, deploy, avisos |
| `AGENTS.md` e `CLAUDE.md` | Instruções de início de sessão para quem constrói com IA: leitura obrigatória do `.ai/guardrails.md`, o que ler por tipo de tarefa e o resumo das regras duras (git, tela, DynamoDB). O `CLAUDE.md` só importa o `AGENTS.md` |
| `docs/` | Só a coleção do Postman (`docs/postman/`) |
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
Detalhes e comandos em [`.ai/fluxo-de-versoes.md`](.ai/fluxo-de-versoes.md). As regras de branch para a IA (mantra, prefixos, quando commitar) estão na seção Git de [`.ai/guardrails.md`](.ai/guardrails.md).

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
| Confirmar um previsto, ver o histórico de versões e restaurar uma versão | todos | só os criados por ela (restaurar um excluído é só do owner) |
| Faturas do cartão: ver | sim | sim |
| Faturas do cartão: configurar, conferir e pagador | sim | não |
| Consultas salvas da Tendências: ver e criar | sim | sim |
| Consultas salvas: renomear, atualizar, excluir e restaurar | todas | só as dela |
| Desativar, reativar e remover um meio de pagamento; ligar a rotina dos combos sugeridos | sim | não |

A regra está em `backend/auth/authz.go` e tem testes; o que não está liberado explicitamente é negado. A tabela completa, com cada ação, está em [`.ai/security.md`](.ai/security.md).

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
| Aviso ativo, aviso de fatura, recorrência e rotina dos combos (auxiliar das rotinas, para não varrer a tabela) | `AVISOS#ATIVOS` | `<espaço>#<id>` |
| Aparência da pessoa | `SPACE#<id>` | `USER#<sub>#CFG#APARENCIA` |
| Consulta salva da Tendências | `SPACE#<id>` | `CONSULTA#<id>` |
| Versão do histórico de um lançamento (só se acrescenta) | `HIST#<id>` | `TX#<id do lançamento>#<versão com 8 dígitos>` |

Todos os exemplos, campo a campo e com dados fictícios, estão em [`.ai/specs/modelo-de-dados/`](.ai/specs/modelo-de-dados/README.md). **Não crie, renomeie nem remova chave ou campo sem discutir com o dono.**

### Lançamento
| campo | tipo | observação |
|---|---|---|
| `status` | texto | na tela, **forma de pagamento**. Um da lista que o servidor manda (PIX, DEBITO, CREDITO, DINHEIRO, TED e PAGO); `CONTAS` (transferência) só o servidor grava; `CREDITO IN`, `CREDITO EX` e os antigos `PREVISTO`, `TRANSFERINDO` e `CREDITANDO` só existem em registros antigos |
| `tipo` | texto | `"saida"` (gasto) ou `"entrada"` (receita) |
| `valor` | número | sempre positivo; o sinal vem de `tipo` (zero e negativo são aceitos para nota e estorno) |
| `descricao` | texto | |
| `nota` | texto | opcional |
| `dataEvento` | texto ISO 8601 (UTC) | data e hora do evento; ordena a lista |
| `banco` | texto | na tela, **meio de pagamento**. Um dos meios de `config/listas` |
| `tags` | lista de textos | sem hierarquia |
| `excluirDoTotal` | booleano | calculado pelo servidor a partir do catálogo de status (CONTAS, transferência, e o pagamento de fatura marcado com `pagaFatura`); o valor que vier do cliente é descartado |
| `previsto` | booleano | só quando verdadeiro, gravado só pelo servidor: lançamento com data futura que ainda não foi confirmado; fica fora do saldo e do Histórico e aparece em Futuros |
| `oculto` | booleano | some do histórico e não entra em saldos |
| `criadoEm`, `criadoPor` | texto | quando e por quem (`sub` do Cognito); o servidor preenche |
| `transferParId` | texto | liga as duas pontas de uma transferência entre bancos |
| `faturaAjuste` | -1, 0 ou 1 | só em crédito: 1 = fatura seguinte à da data; -1 = anterior; ausente = pela data |
| `valorEstimado` | booleano | só em crédito: valor ainda estimado (ex.: compra em dólar) |
| `pagaFatura` | objeto | opcional: `{ meio, ym }`, marca o lançamento como pagamento da fatura `ym` do cartão `meio` |
| `recorrente` | objeto | opcional: `{ intervalo }` em meses; só o servidor o grava, ao marcar a recorrência, e as cópias o herdam |
| `excluidoEm` | texto | só nos excluídos (exclusão lógica); a API nunca o devolve nas listas |
| `serie` | objeto | opcional: `{ id, indice, intervalo }` liga as cópias criadas por Prever ou Repetir ao lançar (`indice` = posição na série, `intervalo` = meses entre ocorrências) |
| `aviso` | objeto | opcional, em evento futuro: `{ dias: [3,1,0], insistir: true }` avisa no Telegram nos dias combinados antes do vencimento e, com `insistir`, todo dia depois dele (ver [`.ai/avisos.md`](.ai/avisos.md)) |

### Configurações
- `listas`: `banco` (os meios de pagamento, com `ativoAte` quando o dono desativa um), `statusCor`, `estiloCor`, `layoutSaldos`, `ordemSaldos`, `ordemManual`
  e `tema` (aparência). **Os status não são mais do app:** a lista, as cores padrão e as regras (conta no saldo, fora do total, crédito) são o catálogo do
  backend (`backend/domain/catalogo.go`) e o `PUT /cfg/listas` ignora qualquer mudança neles. `meios`, `faturaPadrao` e `statusCorPadrao` são derivados pelo
  servidor na leitura e nunca gravados.
- `tags`: `{ usadas: [textos] }`.
- `combos`: `{ lista: [ { tags: [textos] } ], sugeridos: [ { tags: [textos], contagem } ], geradoEm: "AAAA-MM-DD" }`. `lista` são os combos
  salvos pelo dono; `sugeridos` e `geradoEm` são do servidor (item 57): uma rotina semanal (`{"acao":"combos"}`) conta os conjuntos de tags dos
  1000 lançamentos mais recentes (2 tags ou mais, 3 usos ou mais, até 10) e os guarda sem subir o `version`; o PUT manual os preserva. A rotina
  roda nos espaços que têm o item auxiliar `tipo: "combos"` no `AVISOS#ATIVOS`, que o dono liga e desliga em Listas (`PUT /combos/rotina`).
- `cartoes`: `{ porBanco: { "<banco>": { diaPadrao, inicio, faturas: { "AAAA-MM": { fechamento, valorBanco, conferida } } } } }`.
  Estar aqui marca o banco como cartão. `AAAA-MM` é o mês em que a fatura fecha.
- `saldos`: `{ porBanco: { "<banco>": { saldoInicial, dataInicio } } }`.

Escritas de configuração usam um campo `version` para detectar edição concorrente (conflito devolve erro em vez de
sobrescrever em silêncio).

## Regras de negócio
- **Saldo de um meio** = `saldoInicial` + soma dos lançamentos do meio que não estão ocultos, não são previstos, têm status que conta no saldo (pelo catálogo)
  e `dataEvento` a partir de `dataInicio`. Entrada soma e saída subtrai. **Quem calcula é o servidor** (`GET /saldos`); o navegador só exibe.
- **Fechamento mensal.** O saldo é uma cascata: cada mês abre com o fechamento do anterior, desde `dataInicio`. O
  servidor guarda por banco (`SALDO#<banco>`) o fechamento calculado dos meses encerrados (`caches`, só para não reler
  o histórico inteiro, gravado só por ele) e os meses conferidos com o extrato (`conferidos`). O saldo atual parte do último
  fechamento válido e lê só os meses seguintes. O cliente nunca envia saldo nem cache: a conferência recebe só o saldo do extrato.
  - Qualquer escrita em lançamento que mexa no saldo (criar, excluir, ou mudar valor, tipo, status, banco, data ou
    ocultar) marca `invalidoDe` e muda a `version` do banco; um cache calculado antes da escrita é recusado (409).
    Editar descrição, nota ou tags não invalida nada.
  - **Conferir** um mês (só o dono) exige que o saldo calculado bata com o extrato; a diferença se resolve com um
    lançamento de ajuste criado pela própria tela. O mês conferido e todos os anteriores ficam **travados**: o servidor
    recusa (409, `mes_conferido`) criar, alterar ou excluir lançamento que mexa no saldo desse banco nesses meses,
    inclusive na importação. Meses anteriores ao da `dataInicio` do banco são histórico (não entram no saldo): nunca
    travam nem invalidam o cache, então dá para importar anos anteriores com meses já conferidos. **Reabrir** só vale
    para o mês conferido mais recente.
  - **Mês anterior fecha sozinho:** a partir do dia 8, mexer no mês anterior (e nos mais antigos) em algo que muda o saldo pede uma confirmação explícita
    (`409 mes_fechado`, reenvio com `confirmaMesFechado=1`); a conferência com o extrato é opcional e continua travando de forma absoluta.
  - Mês é sempre o do fuso de Brasília (`America/Sao_Paulo`). O cache também vale só para o mesmo saldo inicial, data
    de início e catálogo de status (muda um deles, o cache é descartado e refeito).
- **Transferência entre meios** (`POST /transferencias`) cria dois lançamentos `CONTAS` (saída na origem, entrada no destino) com o mesmo
  `transferParId` e a data escolhida.
- **Faturas do cartão** (calculadas pelo servidor: `GET /cartoes` e `GET /cartoes/{meio}/faturas/{ym}`): a fatura que fecha no mês M cobre as compras de
  crédito do dia seguinte ao fechamento da anterior até o fechamento de M, inclusive. `faturaAjuste` empurra a compra para a fatura vizinha. Crédito = status
  marcado como crédito no catálogo, não oculto; estorno (`entrada`) subtrai. O meio pagador e o vencimento são escolhidos por cartão e por fatura, e a fatura aparece
  em Futuros no meio pagador a partir do vencimento.
- **Conferência da fatura:** diferença = total calculado − `valorBanco`; o servidor sugere registros (1 a 3) até 5 dias do corte cuja soma é igual à diferença.
  Uma fatura **conferida é travada** pelo servidor (`409 fatura_conferida`) até ser desconferida.
- **Eventos futuros:** o lançamento com data depois de hoje nasce com o marcador `previsto` (o servidor liga). Enquanto o marcador existir, ele fica fora do
  Histórico e do saldo e aparece na aba Futuros; os vencidos ficam em Atrasados. **Confirmar** tira o marcador e o lançamento passa a contar no saldo.
- **Duplicado ao salvar:** mesma combinação de status, descrição, valor, banco e `dataEvento`.
- **Meios de pagamento:** o dono cria, desativa (a partir de uma data, o meio deixa de ser sugerido e o passado continua) e remove; só se remove um meio que
  nunca foi usado, conferido pelo servidor na hora. Os status não são editáveis pelo app.

## Backup e importação
**Exportar dados** (aba Configurações, só dono) gera um `.json` versão 2 com `colecoes.lancamentos` e `documentos` das
configurações; **Importar backup** o lê de volta em lotes. Também importa `.xlsx` (lançamentos, notas e fechamentos de
fatura). Há ainda um backup do próprio DynamoDB previsto no [`BACKLOG.md`](BACKLOG.md).

**Importação da planilha** (aba Configurações): depois da prévia, **Confirmar importação** envia as linhas em lotes de 25 pelo
mesmo endpoint de lote do backup (`POST /tx/batch`). Cada linha recebe um id gerado na tela, então reenviar um lote nunca
duplica. A tela mostra `N/total (%)` e o tempo decorrido, que continuam andando enquanto espera o banco liberar
capacidade; falha de rede ou erro 5xx é tentada de novo automaticamente. Listas (bancos, status) e tags novas são gravadas
**antes** das linhas. Se a importação parar no meio, a prévia é descartada e a mensagem diz onde parou: processar o mesmo
arquivo de novo importa só o que faltou (o que já está no banco é ignorado).

**Verificação arquivo x banco de dados** (aba Configurações): depois de importar, e a qualquer momento com o arquivo já
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
  Histórico. O saldo de cada meio e as faturas do cartão são calculados pelo servidor; só as tendências são calculadas no
  navegador, de propósito: são uma simulação que apenas exibe.
- A tela só vê o que outra pessoa gravou na checagem periódica (a cada 3 minutos, com a aba visível, e ao voltar para a
  aba); colaboração em tempo real é uma proposta em `.ai/`.
