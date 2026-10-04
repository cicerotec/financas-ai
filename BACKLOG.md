# Backlog

Atualizado em 2026-10-04. Ordem = prioridade. Marque `[x]` ao concluir.

## Próximos passos (liberar o app para a esposa)

1. [x] **Autorização por papel no backend (Go) + testes automatizados da matriz.**
   Tabela de permissões abaixo. A trava fica na Lambda, não no front. Feito: `authz.go`, `tags.go`,
   `authz_test.go`, `handler_test.go` (banco em memória). **Falta publicar** (`.\scripts\deploy.ps1 <codigo>`);
   a API nova `POST /spaces/{sid}/tags` precisa ser usada pelo front no item 2.
2. [x] **Front por papel.** `/me` devolve o papel; a tela esconde o que o `member` não pode usar
   (aba Importar, edição de saldos/cartão/listas, botões de editar nos registros alheios).
   Feito e verificado na tela com dados inventados (owner e member). Falta publicar o backend (item 1) e
   exercitar com um usuário member de verdade (item 3).
3. [x] **Teste local com um usuário `member`** (janela anônima; `seed.sh` com e-mail `+alias`).
4. [ ] **Hospedagem do front em S3 + CloudFront** (bucket privado, HTTPS, domínio `*.cloudfront.net`):
   - `template.yaml`: bucket, acesso do CloudFront, distribuição; CORS da Lambda e URLs de retorno do
     Cognito aceitando CloudFront **e** `localhost`.
   - `scripts/publicar-front.ps1`: envia `web/` (inclui `config.js`, que fica fora do git) e limpa o cache.
   - Antes de começar: resolver a **decisão pendente** sobre rotear `/api/*` pelo CloudFront (seção "Colaboração em
     tempo real"), porque ela muda o cabeçalho de autenticação e o CORS.
5. [ ] **Convidar a esposa** (`scripts/seed.sh financas <dono> <email-dela>`), instalar na tela inicial do celular.

### Matriz de permissões (decidida)

| Ação | owner | member |
|---|---|---|
| Criar lançamento | sim | sim |
| Copiar lançamento | sim | sim |
| Editar / ajustar fatura | todos | **só os criados por ela** (`criadoPor == sub`) |
| Ocultar lançamento | todos | **só os dela** |
| Excluir lançamento | todos | **não** |
| Criar tag | sim | **só tags que não existem** (servidor só acrescenta) |
| Renomear/excluir tag, combos, bancos, status, flags | sim | não |
| Ver saldos, cartão, tendências | sim | sim (só leitura) |
| Alterar saldo inicial, fechamento, valor do banco, conferir fatura, início do controle | sim | não |
| Aparência (cores, layout, ordem) | sim | sim, **só a dela** |
| Importar / exportar / limpeza | sim | não |

Decisões: ela pode ler **todos** os lançamentos e notas (saldos/cartão/tendências são calculados no
navegador). Hospedagem: **S3 + CloudFront** (GitHub Pages descartado; localhost não serve para ela:
sem HTTPS o PKCE e o Cognito não funcionam).

## Eventos futuros (aba Futuros)

Conceito: status marcados como "evento futuro" (PREVISTO, TRANSFERINDO, CREDITANDO) ainda não aconteceram: não contam
como reais, não mexem no saldo, saem do Histórico e aparecem na aba **Futuros**. O marcador manda por cima das caixas
"conta como real" e "afeta saldo" sem apagar os valores guardados.

- [x] **Parte A (só no front, sem publicar nada na AWS):** marcador "evento futuro" em Listas > Status (e ao criar um
      status); aviso no topo do Histórico (quantos, a pagar, a receber, atrasados); filtro de status sem os futuros;
      busca com a caixa "Incluir eventos futuros" (eventos aparecem marcados); aba Futuros (totais, atrasados no topo,
      por mês, ocultos opcionais, botões Editar/Copiar/Ocultar/Excluir com as regras de permissão); a edição volta para
      a aba de onde começou.
- [ ] **Parte B:** botão **Prever** em cada registro do Histórico: cria cópias como PREVISTO, CREDITANDO ou TRANSFERINDO
      com a mesma data e hora nos meses seguintes ("repetir N meses" além do original; dia que não existe no mês vira o
      último dia, calculado a partir da data original; numeração `1/3` vira `2/3`, `3/3` na descrição). Aviso antes de
      criar se já existe evento igual (status, descrição, valor, banco e data, sem a hora): pular, criar mesmo assim ou
      cancelar.
- [ ] Limpar os PREVISTO e TRANSFERINDO antigos que vieram na importação: com a Parte A eles aparecem como
      **atrasados** em Futuros (use Ocultar ou Editar para o status que de fato aconteceu).
- Limite conhecido: Futuros e o aviso usam os 1000 lançamentos mais recentes carregados; um evento futuro muito antigo
  pode ficar de fora.

## Visibilidade entre usuários

- [x] **Nome de exibição (apelido):** cada pessoa define o seu em Listas > Seu nome; `scripts/definir-nome.sh`
      define pelo e-mail. Guardado no vínculo `USER#sub / SPACE#id`. Publicado a partir da PR #3 (sem merge ainda).
- [ ] O resto (atualização da tela, quem criou/alterou/ocultou, "novo desde a última visita", atividade recente,
      idempotência, edição concorrente) vem da seção **Colaboração em tempo real**, logo abaixo.
- Descartado: bloquear duplicidade por conteúdo (a mesma compra pode ter outro horário e outra descrição).

## Colaboração em tempo real (decidido; **ainda não iniciado**)

Referência: `colaboracao-tempo-real.md` (gerado no chat do Claude; arquivo do usuário, não versionado).
O problema de origem: a tela de uma pessoa não percebe o que a outra gravou, alterou ou ocultou (cópia em
memória só renovada por gravação própria ou a cada 2 min). Sem temporizador curto (15 s foi descartado).

### Fase 1 — log de operações + sincronização incremental
- [ ] **Log de operações no servidor:** cada criação, edição, ocultar/mostrar, exclusão e mudança de configuração
      vira uma operação com número de sequência (`seq`) por espaço, gravada na mesma transação da alteração
      (`TransactWriteItems`: contador no item `META`, item `OP#<seq>` e o registro). Validade de ~180 dias (TTL).
- [ ] `GET /spaces/{sid}/ops?after=N`: devolve só o que mudou desde N (inclui exclusões e ocultações), mais o `seq`
      atual. Sem `after`, devolve só o `seq`. Leitura liberada para owner e member (acrescentar à matriz de permissões).
- [ ] **Atualização da tela por gatilho:** foco da aba, troca de aba, antes de salvar, botão "Atualizar" e uma
      checagem lenta de segurança (3 a 5 min, só com a aba visível). A tela aplica as operações recebidas em vez de
      recarregar os 1000 lançamentos.
- [ ] Registrar em cada lançamento quem criou/alterou/ocultou e quando (`atualizadoPor/Em`, `ocultoPor/Em`) e
      mostrar na linha do registro; lista de membros do espaço (item espelho `SPACE#id / MEMBER#sub`) para
      converter `sub` em nome.
- [ ] "Novo desde a sua última visita" (etiquetas automáticas novo/alterado/oculto + contagem) e "Atividade
      recente" (lista de eventos), ambos a partir do log, sem nada para o usuário marcar ou revisar.
- [ ] **Idempotência e edição concorrente:** id do lançamento gerado na tela (criar duas vezes devolve o mesmo
      registro), travar o botão Salvar, e 409 "alterado por outra pessoa" se o registro mudou desde que foi aberto.
- Cuidados: a importação em lote não deve gerar uma operação por item (emitir uma operação-resumo); a transação
  consome ~2× de escrita (capacidade atual 5 por segundo); registros antigos não têm `atualizadoEm`.

### Fase 2 — presença ("fulano está mexendo neste registro agora")
- [ ] **Presença em tempo real:** saber, enquanto você usa o app, que a outra pessoa está criando ou editando um
      registro naquele momento (aviso global "esposa está criando um registro"; no registro aberto, "fulano está
      aqui"; rascunho com auto-save visível no Histórico). Presença é efêmera e separada dos dados e expira sozinha
      (~60 a 90 s sem sinal de vida).
- [ ] Caminho A (sem serviço novo): polling de um `head.json` no CloudFront com cache de 1 s (atraso de 2 a 3 s;
      presença passa por Lambda + DynamoDB + S3). Caminho B (depois): AWS AppSync Events (WebSocket gerenciado, atraso
      de centenas de ms, centavos por mês). O modelo de operações é o mesmo, então dá para começar em A e migrar.
- Cuidados levantados na leitura do documento: o `head.json` público deve ter só o `seq` (presença e nomes vêm da
  API autenticada); a presença não pode incrementar o `seq` dos dados; no AppSync, canais precisam de autorização
  por espaço, o exemplo usa `aws-amplify` (exige bundler; o front não tem build) e é preciso confirmar a
  disponibilidade em `sa-east-1`.

### Decisão pendente antes do item 4
- [ ] **Rotear `/api/*` pelo CloudFront (Function URL privada com OAC) ou manter a chamada direta à Function URL?**
      Com OAC o token do Cognito não pode ir em `Authorization` (o OAC usa esse cabeçalho; mandar em outro) e
      `POST`/`PUT` exigem o hash do corpo em `x-amz-content-sha256`. Conferir na documentação da AWS antes de decidir.

## Confiabilidade e dados

- [ ] Conferir no navegador a aba Cartão: início do controle, fatura conferida e "mudou desde a conferência".
- [ ] Carregar anos anteriores (do mais novo para o mais antigo); definir o **início do controle** de cada cartão.
- [ ] Conferir totais de fatura de cada cartão contra o banco (MERCADO PAGO, NUBANK, INTER, SANTANDER).
- [ ] Importação do `.xlsx` usar o endpoint de lote (hoje grava um lançamento por requisição).
- [ ] Aviso "já existe um igual" ao salvar manualmente usa só os 1000 mais recentes carregados.
- [ ] Cartão: campos **valor pago** e **observação** por fatura (exceções como extrato ≠ pago).

## Arquitetura

- [ ] Mover **saldos e faturas do cartão** para a Lambda (hoje o navegador lê todo o histórico).
      Isso também permitiria esconder notas/lançamentos alheios do `member`, se um dia for preciso.
- [ ] **IA** ("Preencher com IA") via Lambda, com a chave da Anthropic no SSM Parameter Store.
- [ ] Testes automatizados (backend e a lógica de faturas do front).
- [ ] Salvamento simultâneo de configurações: hoje o front repete por cima no conflito (409); mesclar.
- [ ] Avaliar índice (GSI) por banco se o histórico crescer.

## Segurança e operação

- [ ] Revisar o armazenamento do token (`localStorage`) e adicionar limite de taxa na Function URL.
- [ ] AWS Budget de US$ 1 com alerta por e-mail (conferir se existe).
- [ ] Backup periódico (exportar JSON) e/ou point-in-time recovery do DynamoDB.

## Git e releases

- [ ] Fazer o merge da PR #3 (nome de exibição) em `develop`. (As PRs #1 e #2 já foram mergeadas.)
- [ ] Primeira release: PR `develop → main` e tag `v0.1.0`.

## Ideias

- `#Garantia` com filtro de garantias vigentes (data de vencimento na nota).
- Anexar foto/PDF da nota fiscal ao lançamento (bucket S3).
- Fatura: aviso quando uma compra `CREDITO EX`/`CREDITO IN` cair fora do esperado (a planilha as ignora).
