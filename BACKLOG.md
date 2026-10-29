# Backlog e roadmap

Atualizado em 2026-10-27. O roadmap é separado por **área de responsabilidade**; dentro de cada área, os itens estão em
ordem de prioridade e levam a etiqueta **[Agora]**, **[Próximo]** ou **[Depois]**. A numeração é fixa (não renumerar:
outros trechos referenciam os números). Os detalhes de cada assunto ficam em `.ai/`. Marque `[x]` ao concluir e
retire o item do índice "Seções".

## Roadmap

### Bugs (têm prioridade sobre o resto)
Cada item começa com a área correspondente entre colchetes. A numeração segue a mesma sequência do roadmap.
- 25. [x] **[UX/UI] Combo de bancos volta sozinho ao primeiro item ao lançar após copiar pelo Histórico**, o que pode
  gerar erro no cadastro. Detalhe em [`.ai/ux.md`](.ai/ux.md).
- 29. [x] **[UX/UI] O botão do formulário de Lançar volta a dizer "Salvar edição" depois de salvar uma edição**, já
  num formulário vazio para um lançamento novo (o `editandoId` fica `null`, então salvar cria um lançamento novo, mas o
  rótulo engana). Causa: `comBotaoOcupado` (`web/index.html`) guarda o texto do botão antes da ação e o restaura no
  `finally`, desfazendo o "Salvar lançamento" que o `cancelarEdicao` acabou de pôr. Já existia antes dos avisos; foi
  achado ao testá-los. Correção provável: restaurar o texto só se ele não mudou durante a ação.

### Seções (índice dos itens pendentes)
Só o que ainda não foi feito, na ordem de prioridade (**[Agora]**, depois **[Próximo]**, depois **[Depois]**), cada um
com o salto para o registro real. **Ao concluir um item, marque `[x]` no registro e tire a linha daqui.** Bugs
pendentes: nenhum.

- [\[Infra\] - 49 - **\[Próximo\] App multiusuário (MFA, cadastro com convite, espaço próprio, isolamento):**](#item-49)
- [\[Backend\] - 34 - **\[Próximo\] Avisos por despachante e vários canais:**](#item-34)
- [\[UX/UI\] - 42 - **\[Próximo\] Filtro único (tags, bancos, status e combos) e lista da consulta da Tendências:**](#item-42)
- [\[UX/UI\] - 47 - **\[Próximo\] Tela de aviso como componente reutilizável:**](#item-47)
- [\[Infra\] - 11 - **\[Depois\] Backup periódico:**](#item-11)
- [\[Backend\] - 39 - **\[Depois\] Open Finance (visão):**](#item-39)
- [\[Backend\] - 43 - **\[Depois\] Meios de pagamento: o dono cria e desativa:**](#item-43)
- [\[Backend\] - 12 - **\[Depois\] Colaboração, Fase 1:**](#item-12)
- [\[Backend\] - 13 - **\[Depois\] Colaboração, Fase 2:**](#item-13)
- [\[Backend\] - 15 - **\[Depois\] IA ("Preencher com IA"):**](#item-15)
- [\[Backend\] - 18 - **\[Depois\] Segurança e escala:**](#item-18)
- [\[Backend\] - 27 - **\[Depois\] Tela de permissões por usuário:**](#item-27)
- [\[Front\] - 20 - **\[Depois\] Ideias:**](#item-20)

### Infra (AWS, SAM, ambientes, custo, backup)
- 50. [x] <a id="item-50"></a>**Log de cada requisição** (concluído em 2026-10-29; roteiro em [`.ai/specs/log-de-requisicoes/`](.ai/specs/log-de-requisicoes/intent.md)): o `handler` escreve uma linha `req rid= m= acao= status= ms=` ao terminar, sem corpo, query string nem ids; serve para achar as leituras repetidas e decidir o índice do Histórico.
- 49. [ ] <a id="item-49"></a>**[Próximo] App multiusuário (2026-10-27; absorve o antigo item 37, MFA):** abrir o app a qualquer pessoa, cada uma no seu espaço, em fases
  (plano em [`.ai/specs/app-multiusuario/`](.ai/specs/app-multiusuario/intent.md)). Decidido com o dono: **ninguém lê dado de outro, nem o dono** (isolamento lógico; a garantia
  criptográfica seria a cifragem por campo de `.ai/privacidade-e-cifragem.md`); **o status é do app, global e nunca do usuário** (cada status novo, como `PIX`, `DEBITO` e `CRIPTO`, é
  uma entrada do `Catalogo` por spec).
  - [ ] **Fase 1, proteger antes de abrir:** **MFA** no login (Cognito, TOTP por aplicativo autenticador, sem o custo do SMS, `template.yaml`, os dois ambientes; hoje o login é só e-mail e
    senha e uma senha vazada basta para ver tudo; **recuperação** se o celular se perder, por reset do dono via script; ajuste no convite de `scripts/seed.sh`); **AWS Budget de US$ 1**
    com alerta por e-mail (conferir se existe; controla o custo da conta AWS, os 2 primeiros são grátis: Billing > Budgets); **limite de taxa** na Function URL (teto de concorrência,
    WAF só se o dono aceitar o custo); **capacidade da tabela** (5/5, autoscaling dentro das 25 unidades gratuitas ou sob demanda). Ao concluir, atualizar a autenticação de `.ai/security.md`.
  - [ ] **Fase 2, cadastro só com código de convite** (gatilho `PreSignUp` do Cognito, código no SSM, tela de cadastro própria; demo antes).
  - [ ] **Fase 3, espaço próprio no primeiro acesso** (`POST /me/espaco`, mesmos itens do `seed.sh`, sem chave nova).
  - [ ] **Fase 4, app vazio e boas-vindas** (espaço sem configuração funciona, só com status do catálogo; demo antes).
  - [ ] **Fase 5, isolamento provado, exportar e excluir a conta** (teste por rota; exclusão cobre `HIST#` e o Cognito; política de privacidade).
  - [ ] **Fase 6, abrir** no `dev` e depois na produção, para poucas pessoas.
  **A decidir (no `intent.md`):** como limitar a taxa, capacidade da tabela, um código ou vários, MFA obrigatório para todos ou só para o dono, como a pessoa cadastra o aplicativo no
  primeiro acesso (confirmar no `dev`), política de privacidade. **Fora:** a tela de permissões (item 27), login social e planos pagos.
- 2. [x] **Decidir o `/api/*`: chamada direta à Function URL (opção B)**, front estático no CloudFront. Reavaliar
  rotear a API pelo CloudFront (OAC) quando entrar domínio próprio, WAF ou mais usuários. Detalhe em [`.ai/hospedagem.md`](.ai/hospedagem.md).
- 3. [x] **Hospedar o front em S3 + CloudFront** (HTTPS, `*.cloudfront.net`). No ar (PR #5). Detalhes em [`.ai/hospedagem.md`](.ai/hospedagem.md).
- 22. [x] **Ambiente de teste (`dev`)**: segunda stack `financas-dev` com tabela, login, API e front próprios,
  dados fictícios, sempre ligada, publicada à mão a partir da `develop`. No ar (PRs #33, #34 e #35). Detalhes em
  [`.ai/ambiente-de-teste.md`](.ai/ambiente-de-teste.md).
- 24. [x] **Deploy por GitHub Actions (OIDC)**: `develop` publica o dev, `main` publica a prod com aprovação manual,
  PR só valida; roles IAM por OIDC separadas por ambiente, sem chave nos secrets; o deploy falha se o changeset remover
  ou substituir a tabela ou o login. No ar (PRs #38, #39 e #40, release v0.6.0). Guia em
  `.ai/deploy-github-actions.md`.
- 11. [ ] <a id="item-11"></a>**[Depois] Backup periódico** (exportar JSON e/ou point-in-time recovery do DynamoDB). Adiado por decisão: não será feito agora. O **AWS Budget** que
  este item trazia passou para a fase 1 do item 49.
  - **Backup:** o PITR do DynamoDB está ligado **só em prod** pelo `template.yaml` (`!If [EhDev, false, true]`; ligar
    no console sem mudar o template seria desfeito no próximo deploy). Falta o backup periódico; enquanto isso, o
    botão Exportar (`.json`) é a cópia manual. Plano em fases (dump cifrado fora da AWS, restauração testada, espelho do
    código, automação e vigília) em [`.ai/specs/backup/`](.ai/specs/backup/plan.md); rascunho para aprovação (2026-10-26).

### Backend (Go, DynamoDB, Cognito, API)
Os itens 31 a 34 vêm das decisões de 2026-10-10 (erro grave: uma edição sobrescreveu um registro e o dado se perdeu).
O documento de referência é [`.ai/arquitetura.md`](.ai/arquitetura.md); leia antes de mexer. **Nenhuma chave nova no
DynamoDB sem discutir com o dono.**
- 31. [x] <a id="item-31"></a>**[Agora] Histórico imutável e versionado:** todo dado que entra vai também para o histórico, inteiro; toda
  edição, ocultação, exclusão e restauração gera uma versão (quem, quando, o registro completo; **criar e importar `id` novo não geram**, ajuste de 2026-10-23), gravada na mesma transação
  do registro atual (`TransactWriteItems`) e com checagem de versão contra edição concorrente. Excluir passa a ser
  lógico, com o campo `excluidoEm` só nos excluídos; restaurar cria versão nova; o histórico nunca é editado nem
  apagado (sem método no repositório e `Deny` por IAM em `HIST#*`). **Só lançamentos** nesta entrega; configurações e
  fechamentos ficam para depois. **Sem migração:** a versão 1 nasce na primeira edição ou exclusão do lançamento. **Fase 3b (2026-10-23, período e busca por tag no Histórico de versões): implementada.** **Ajuste de 2026-10-23 (Fase 1b, implementada):** criar e importar `id` novo deixam de gravar versão e a v1 passa a se chamar "Original" na tela; roteiro em [`.ai/specs/historico/`](.ai/specs/historico/).
  Decisões fechadas com o dono em 2026-10-15; PITR já está ligado em prod. Pontos que valem: as rotas atuais de `/tx`
  mantêm o contrato e passam sempre pelo service (só as rotas de histórico são novas); a tela "Histórico de versões"
  busca por **mês de alteração**; o GSI por data entra depois, quando a leitura pesar; a capacidade da tabela (5 WCU)
  fica para depois da refatoração. Fases: 0 (um caminho de escrita), 1 (versões e exclusão lógica), 2 (busca, diff e
  restauração na API), 3 (tela, com demo antes). **Fases 0 e 1 implementadas e verificadas em dev** (a 1 grava as
  versões e exclui logicamente, com o `Deny` de IAM conferido); **a 2 (busca por mês, diff e restauração na API) está
  implementada e verificada em dev**; **a 3 (tela Histórico de versões) está implementada e verificada em dev**. O item 38 (fechamento automático do mês anterior) fica para
  depois do histórico: a restauração da Fase 2 respeita a trava do mês conferido, como as outras escritas. Detalhes em
  [`.ai/specs/historico/`](.ai/specs/historico/) (`intent.md`, `plan.md` e `spec.md`).
- 32. [x] <a id="item-32"></a>**[Agora] Arquitetura em camadas:** `main` só liga as peças; `controller` recebe o request; `service` decide o
  que o front recebe e recebe tudo por construtor; `domain` com tipos e regras puras que todos usam; adaptadores
  (DynamoDB, Telegram, auth, relógio) atrás de interfaces declaradas pelo service; sem variáveis globais (`db`,
  `ssmCli`, `table`, `agora()`). Entra por fatias verticais, uma regra por vez, com o app no ar. **Só a fatia dos
  lançamentos** (repository e service de lançamentos, injetados) entra com o 31, na Fase 0 dele (decisão de
  2026-10-15); `fechamento`, `tags`, `avisos`, `cfg` e `perfil` seguem como estão e migram nas fatias seguintes.
  **Feito (PR #100):** cada camada virou uma pasta em `backend/` (`domain/`, `auth/`, `repository/`, `service/`,
  `controller/`), com a regra de dependência testada em `arquitetura_test.go`; lançamentos e `auth` migraram primeiro. O
  resto da raiz (`fechamento`, `tags`, `perfil`, `seq`, `cfg`, avisos e Telegram) migrou depois, em quatro fatias, **todas no ar e conferidas em dev (concluído em 2026-10-17)** (plano em
  [`.ai/specs/arquitetura-camadas/`](.ai/specs/arquitetura-camadas/plan.md)). **Fatias 1 a 4 (espaços e leitura de lançamentos;
  configurações, tags, contador e perfil; fechamento, saldos e a trava de mês, com o `Relogio` injetado; avisos e Telegram,
  `adapter/notificacao`, sem variáveis globais e sem as pontes da raiz).** A raiz de `backend/` só liga (`main.go`,
  `servidor.go`, `roteador.go`, `local.go`). O despachante de avisos (item 34) vive no `service`, em cima do `Notificador`, e os canais ficam em
  `adapter/notificacao/`.
  Achado para outra entrega: cursor inválido em `GET /tx` devolve `500`; o certo seria `400`.
- 38. [x] <a id="item-38"></a>**[Próximo] Fechamento automático do mês anterior (confirmação explícita para mexer no passado):**
  decisões do dono (2026-10-16). O **conferido manual** da aba Saldos é herança da planilha e fica **opcional** (trava
  absoluta, como hoje, sem ser removido); o sistema deve fluir sem ele. O mês **fecha sozinho**: um evento de mês anterior
  só pode ser alterado com **confirmação explícita**. O mês anterior fica livre até o dia 7 do mês seguinte; **a partir do
  dia 8** (fuso de São Paulo) exige confirmação; meses mais antigos sempre. A regra é **calculada pela data**, sem gravar
  estado nem campo novo no lançamento, e vale só para o que muda o saldo (valor, tipo, status, banco, data, oculto):
  descrição, nota e tags seguem livres, e o que é anterior ao início do controle também. O backend recusa com `409` e um
  código próprio (nome a definir no spec); a tela mostra a janela de confirmação e reenvia com o sinal de confirmação;
  o front só exibe. Vale para criar, editar, excluir, importar (item a item, como no mês conferido) e restaurar (Fase 2
  do item 31). A confirmação fica gravada **na versão do histórico** (campo novo só no item `HIST#`, nunca no lançamento
  vivo; o nome exato entra no spec). **Excluir passa a ser imediato:** o `DELETE` vai na hora, a recusa aparece antes de
  o item sumir, e o **Desfazer vira uma restauração** (a resposta do `DELETE` diz qual versão restaurar); depois dos 6 s
  ainda dá para recuperar pela tela de histórico. Isso muda o padrão 11b de `padrao-de-telas.md` (demo antes do código).
  **Ordem (decidida em 2026-10-16):** vem **depois** do histórico (item 31, fases 2 e 3). A
  restauração da Fase 2 passa pelo mesmo ponto de trava das outras escritas, então este item entra num lugar só e vale
  para criar, editar, excluir, importar e restaurar. A mudança de tela (exclusão imediata, Desfazer por restauração) é
  deste item, não da Fase 3 do histórico. **Custo:** um toque a mais ao editar o passado. Depende do item 31. **Concluído em 2026-10-17:** as fases A (backend), B (tela da
  confirmação) e C (excluir imediato, Desfazer por restauração) estão implementadas e verificadas em dev.
- 40. [x] <a id="item-40"></a>**[Próximo] Saldo, fechamento e conferência calculados e validados no servidor (segurança):** decisão do
  dono de 2026-10-18, depois de ver que o navegador calcula o saldo e o servidor grava o que ele manda. Hoje
  `SalvarCaches` aceita qualquer número finito de um mês passado e `Conferir` só compara `saldo` e `real`, os dois vindos
  do cliente, então quem forja a requisição trava um mês com o saldo que quiser (só o owner chama essas rotas, mas o
  número vira o ponto de partida da cascata). Passa a valer: o servidor calcula a cascata, o fechamento, o total dos
  bancos e os totais de Futuros e do Histórico (relógio e fuso dele); o cache é gravado só por ele; `Conferir` recebe só o
  saldo do extrato e compara com o que o servidor calculou; campo derivado vindo do cliente é **rejeitado**. Cada
  rejeição ganha um teste que tenta forjar o valor e que falha antes da mudança. Calcular no front só vale para exibir
  (Tendências). **Primeiro da fila (2026-10-24):** vem antes do 33a e do 41, porque o front só deixa de tratar status quando para
  de somar. Escopo: saldo por banco e total de **Saldos** e conferência com o extrato (o `GET /saldos`, o cache só do
  servidor, `Conferir` só com o `real` e o ajuste criado pelo servidor); o resumo de Futuros fica com o item 41 e o total do Histórico sai da tela até o item 42 e as
  faturas do cartão com o 33a. Fases: A (backend, só acrescenta; **com teste de que o servidor lê só do início do controle em diante**: o histórico anterior à `dataInicio` de cada banco não entra na conta nem na leitura da tabela), B (a aba Saldos, visual igual, **conferida lado a lado** com a
  antiga no dev e na produção), C (remove `PUT /fechamentos`). Intent, plan e spec (rascunho) em
  [`.ai/specs/saldo-servidor/`](.ai/specs/saldo-servidor/intent.md). Detalhes em [`.ai/arquitetura.md`](.ai/arquitetura.md) e
  [`.ai/security.md`](.ai/security.md). Não cria campo no Dynamo por si; qualquer campo novo passa pelo dono. Absorve o
  trecho de saldos do item 14.
  **Concluído em 2026-10-24:** fase A (backend: `GET /saldos`, cache só do servidor, conferência com o `real`), B (a aba Saldos lê o servidor, conferida pelo dono em dev e produção) e C (saíram `GET`/`PUT /fechamentos` e a ação `fechamento.escrever`; a conferência recusa `saldo` no corpo).
- 41. [x] <a id="item-41"></a>**[Próximo] Futuros: marcador `previsto` e status do backend:** decisões do dono de 2026-10-18, só a
  aba Futuros, **depois do 40 e do 33a** (2026-10-24). **Regra do dono: o front não trata status de forma nenhuma.** O
  backend manda a lista de status (`["PAGO", "CREDITO"]` no corte; no futuro `PIX`, `DREX`, `DEBITO`...), e o front só a
  exibe, guarda a escolha e a devolve como veio; a lista fixa de `app.js:5` deixa de existir e um teste de CI falha se um nome de
  status voltar a `web/`. **Sem data de corte global** (2026-10-24): o catálogo tem uma entrada por nome e a marca `selecionavel` (nenhum status sai dele); o
  usuário pode reclassificar o passado (um `PAGO` de 2018 pode virar `PIX`) e a trava de data segue o efeito no saldo; o Histórico sai do servidor pronto
  (rótulo e cor, filtros de status do uso no período; **sem total**: o total da lista e a soma por mês saem da tela até o item 42, pedido do dono). O dono não criou status além dos sete padrão. Os status `PREVISTO`, `TRANSFERINDO` e `CREDITANDO` saem; o lançamento futuro nasce com o status real e um
  marcador `previsto` (campo novo no Dynamo, booleano gravado só quando ligado, **aprovado em 2026-10-24**) que o tira do saldo e do total do Histórico até ser
  confirmado. **Quem liga o marcador:** o servidor, em todo lançamento com data depois de hoje; o Prever e o "Repetir ao
  lançar" também (2026-10-24). O servidor entrega Futuros e Histórico prontos: os vencidos **continuam em Atrasados,
  dentro de Futuros**, e o Histórico só traz o que já aconteceu (nunca um previsto); o mesmo critério alimenta o aviso do Telegram; os
  status viram do backend (`salvarListas` segue gravando bancos, cores, aparência e ordens e ignora mudança em status;
  `statusFuturo` deixa de ir ao front). **Lançamentos antigos** com os três status **ficam como estão** (eram só
  previsões): continuam em Futuros, os do passado em Atrasados, até alguém tratar; sem migração em massa e nada
  regravado. **Confirmar** (demo aprovada em 2026-10-24, `.ai/demos/demo-confirmar-previsto.html`): janela com Valor,
  Data e Status, cada um já preenchido e alterável, textos "Confirmar", "Já paguei" e "Já recebi", botão verde na
  tabela, aviso de 6 s com Desfazer (restauração no servidor), data sugerida hoje em mês fechado; ao confirmar um
  antigo, a janela abre com o primeiro status da lista do servidor (hoje `PAGO`). Sai do front o que olha status ou soma: `ehFuturo`, `afetaSaldo`, `resumoFuturos`,
  `gruposFuturos`, a soma do Histórico, a inclusão automática de `CREDITANDO` em Listas e o "Status das cópias" do Prever
  (a decidir no spec). Fora daqui: desmembrar o `PAGO` (tipo e meio), o `CONTAS` e o `excluirDoTotal`, e o papel de
  Admin. **Em aberto para o spec:** como o servidor acha todos os previstos sem o limite dos 1000 lançamentos
  mais recentes (só volta ao dono se pedir chave nova) e o que o Prever faz com o "Status das cópias". Demo antes do
  código para qualquer tela além da de confirmar. Depende dos itens 40 e 33a (o navegador já não soma com status). Intent, plan e spec (aprovados em 2026-10-24; **fases A e B feitas** (backend e tela de Futuros), C pendente) em
  [`.ai/specs/futuros-previsto/`](.ai/specs/futuros-previsto/intent.md): fases A (backend), B (Futuros e Confirmar) e C (Listas, Lançar e Prever).
  - [x] **Fase A, backend** (concluída em 2026-10-25): catálogo de status, marcador `previsto`, `GET /futuros`, `POST /tx/{id}/confirmar`, `GET /status`, `POST /transferencias`, `GET /tx` sem previstos.
  - [x] **Fase B, a tela de Futuros e Confirmar** (concluída em 2026-10-25): Futuros lê o servidor, Confirmar com Desfazer, Histórico sem total e sem o resumo de futuros. Futuros olha só do mês passado em diante (os atrasados antigos: item 45).
  - [x] **Fase C, o front sem status** (concluída em 2026-10-25, conferida pelo dono no dev): Listas só com a lista do servidor e a cor, Prever e Estender sem "Status das cópias", cores e ajuste de fatura vindos do servidor, e `web/sem-status.test.js` (roda no CI) que proíbe nome de status em `web/`.
  **Concluído em 2026-10-25:** fases A, B e C. Sobra o item 45 (atrasados antigos de Futuros buscados por ano).
- 33. [x] <a id="item-33"></a>**[Próximo] Regras no backend, front só exibe:** fatura do cartão (fechamento, base, ajuste e total prontos na
  API, preservando `faturaAjuste` e o fechamento escolhido por mês), início do controle (`historico: true` vindo da API),
  conferência de fatura **travada pelo backend** como o mês do banco (com forma de reabrir), e futuro/real/afeta saldo
  aplicados pelo servidor com as caixas que já existem em Listas (o saldo do mês da aba Futuros, hoje somado no front em
`gruposFuturos`, e o total da lista do Histórico, somado em `renderHistorico`, também passam a vir prontos; o
`excluirDoTotal`, hoje gravado pelo front a partir
  do nome `CONTAS`/`TRANSFERINDO`, é outra decisão por etiqueta: a regra passa a ser do servidor, e o que fazer com o
  campo gravado é discussão). **Revisto em 2026-10-18:** a parte de futuro/real/afeta saldo passa pelos itens 40 e 41
  (marcador `previsto`, catálogo de status do backend), e não mais pelas caixas de Listas. Saem do front `ehCredito*`, `ajustePadrao`,
  `faturaBase`, `ehFuturo`, `afetaSaldo`. Os status `CREDITO IN`/`EX` deixam de ser necessários; migração dos
  lançamentos antigos sem `faturaAjuste` fica a discutir. **O cartão vira entidade própria:** hoje `CFG#CARTOES` é
  `porBanco` e o lançamento de crédito liga ao cartão pelo campo `banco`, mas o cartão não é um banco (pode não ter
  nenhum, e o banco que paga a fatura é uma escolha). Passa a ser por cartão, com o vínculo ao lançamento e a migração
  a discutir antes de criar qualquer campo. Absorve o item 14.
  **Fatiado em 2026-10-24 (vem depois do 40 e antes do 41):** a **fatia a** é a fatura, o total e a conferência no servidor
  (`GET /cartoes`, `GET .../faturas/{ym}`, comandos de cartão, as sugestões de ajuste portadas para Go, `bancos[].cartao` em
  `GET /cfg/listas`, validação de `faturaAjuste`, `PUT /cfg/cartoes` removida), com a aba Cartão **conferida lado a lado**; a
  **fatia b** (o cartão como entidade própria, a trava da fatura conferida e a migração do `faturaAjuste` explícito) segue a discutir.
  Intent, plan e spec da fatia a (aprovados em 2026-10-24) em [`.ai/specs/fatura-servidor/`](.ai/specs/fatura-servidor/intent.md).
  **Fatia a concluída em 2026-10-24** (fases A, B e C: a aba Cartão e a linha Fatura do Lançar leem o servidor, conferidas pelo dono em dev e produção; o `PUT /cfg/cartoes` saiu). Sobra a fatia b.
  - [x] **Fatia a: faturas do cartão no servidor** (concluída em 2026-10-24: fases A, B e C).
  - [x] **Fatia b, com o escopo reduzido (2026-10-25; concluída em 2026-10-25, junto com o item 35):** só a **trava da fatura conferida pelo servidor**, feita junto com o item 35. **Fora, por decisão do dono:** o cartão como entidade própria (ele continua sendo uma entrada de `CFG#CARTOES`, `meios[].cartao`) e a migração do `faturaAjuste` explícito (o passado está correto). Intent, plan e spec (rascunho) em [`.ai/specs/pagar-fatura/`](.ai/specs/pagar-fatura/intent.md).
- 34. [ ] <a id="item-34"></a>**[Próximo] Avisos por despachante e vários canais:** o service só sabe o que avisar e usa um `Avisador`; um
  despachante lê os canais que a pessoa assinou (Telegram, e-mail, WhatsApp), formata, divide e entrega, e uma falha de
  canal não derruba as outras. Cada canal é uma strategy; canal novo é um adaptador novo. Resolve na origem o limite de
  4096 caracteres do item 28. **Ponto de partida (item 32, fatia 4):** o `Avisador` já existe com o nome `service.Notificador`
  (`Enviar(destino, texto)`), o canal Telegram é o primeiro arquivo de `adapter/notificacao/`, e a rotina diária está em
  `service.Avisos`; falta o despachante por assinatura e os outros canais.
- 43. [ ] <a id="item-43"></a>**[Depois] Meios de pagamento: o dono cria e desativa (rascunho, 2026-10-24):** o que o dono chama de **meio de pagamento** (conta, carteira, vale, cartão:
  `SODEXO-REF-IBM`, `DINHEIRO`, `CARTEIRA`...) é o `banco` gravado, com um **de-para só de exibição em `domain/`** (o Dynamo não muda, o campo continua `banco`).
  **O dono cria e desativa meios pelo app** (dado do espaço, não do produto); o front nunca conhece os nomes. Um meio desativado (por exemplo, o `SODEXO` quando o dono sair da
  IBM) **continua nos lançamentos e no saldo do passado** e deixa de ser sugerido a partir de uma data; "ver todos" mostra também os inativos, porque o passado é do
  usuário. `CRIPTO` e `DREX` são **formas** (status), não meios. Pede **um atributo novo no Dynamo** (a data em que o meio deixa de valer, `ativoAte` (AAAA-MM-DD) em
  `CFG#SALDOS.porBanco[meio]`, ausente = ativo; **aprovado pelo dono em 2026-10-24**) e a lista `meios: [{nome, cartao, ativo}]` que o item 33a já entrega com `ativo` sempre verdadeiro.
  Intent, plan e spec depois da conversa sobre o 40, o 33a e o 41. Demo antes de qualquer tela.
- 46. [x] <a id="item-46"></a>**Aviso no Telegram para a fatura do cartão** (concluído em 2026-10-26: fase A do backend no #244 e fase B da tela no #246; conferido em dev pelo dono): hoje só um lançamento tem aviso; a fatura, que é calculada pelo servidor, só tem o botão Já paguei. O aviso passa a ser ligado **na própria fatura** em Futuros (botão Avisar, mesma tela do Prever), só para aquela fatura (o cartão não tem aviso padrão) e só se ela tiver vencimento. Dado novo: `faturas[ym].aviso` (mesmo formato do lançamento) mais o item auxiliar já existente em `AVISOS#ATIVOS`; **desempenho: dias sem aviso não leem nada além da Query dos avisos**. A fatura paga some do resumo sozinha. Intent, plan e spec (rascunho) em [`.ai/specs/aviso-fatura/`](.ai/specs/aviso-fatura/intent.md); demo `.ai/demos/demo-aviso-fatura.html`.
- 45. [x] <a id="item-45"></a>**Futuros: os atrasados antigos são buscados por ano, sob demanda** (concluído em 2026-10-26, em produção; demo `.ai/demos/demo-atrasados-por-ano.html`; `GET /futuros` agrupa em `atrasados[{ano,n,valor,meses}]` e devolve `anosAnteriores`; `?ano=AAAA` busca um ano; anotado em 2026-10-25): hoje Futuros olha só **do mês passado em diante**
  (decisão do dono: ler todos os atrasados de todos os anos de uma vez sobrecarrega a tabela e não faz sentido). Quem tem um previsto mais antigo (por exemplo, um `PREVISTO` esquecido de 2019)
  não o vê em lugar nenhum, porque o Histórico nunca mostra previstos. **A forma de buscar os atrasados vai mudar na tela:** em vez de uma lista única, os atrasados mais antigos passam a ser
  buscados **por ano, quando a pessoa pedir** (por exemplo, "Ver atrasados de 2019"), cada ano um `Query` pelo intervalo da data na chave (`TX#<ano>...`), sem ler a partição inteira e sem índice novo.
  A rota `GET /futuros` ganha o parâmetro do ano (ou período); a tela decide como oferecer (seletor de ano, aviso "há N atrasados de anos anteriores" só se houver). **Demo antes de qualquer tela.**
  Hoje o corte está em `inicioDeFuturos` (`backend/service/previsto.go`) e na decisão 15 do `plan.md` do item 41.
  **Agrupar e recolher (pedido do dono, 2026-10-25):** os atrasados passam a vir **agrupados** (por ano, e dentro do ano por mês) em blocos **recolhíveis**, com o mesmo gesto do painel do cartão (título com ▾/▴, como em "Configurar cartão ▾"): o grupo mostra o título e a quantidade, e abre ou fecha ao tocar. O agrupamento e as contagens vêm do servidor (o front só exibe e guarda se o bloco está aberto). Os atrasados do mês passado em diante continuam abertos por padrão; os dos anos anteriores começam recolhidos. Entra na mesma demo.
- 39. [ ] <a id="item-39"></a>**[Depois] Open Finance: lançamentos e saldos vindos do banco (visão):** visão do dono
  (2026-10-16): o app deixa de depender de lançar e conferir à mão; os lançamentos e o saldo de cada conta vêm do banco.
  (a) O lançamento vindo do banco é a **fonte da verdade**: valor, tipo, data e banco **não editáveis**; a pessoa edita a
  classificação (tags, descrição, nota). Os campos que mudam o saldo são os mesmos de `mudaSaldo`. (b) **Conferência
  automática:** o sistema compara o saldo calculado com o que o banco informa e marca o mês como conferido pelo sistema,
  ou aponta a divergência; o botão manual some. (c) **Pré-requisitos:** saldo calculado no servidor (itens 14 e 33); id
  estável do banco em cada lançamento para não duplicar (a importação já preserva o `id` do item); e a origem do
  lançamento (manual ou banco), uma decisão de modelo que só entra com aprovação (princípio 3). (d) Provedor, custo e
  consentimento a pesquisar. Depende dos itens 33 e 38.
- 14. [x] <a id="item-14"></a>**[Depois] Saldos e faturas do cartão calculados na Lambda** (o saldo do banco já lê só o mês aberto, graças
  ao fechamento mensal; faltam as faturas do cartão e as tendências). Absorvido pelo item 33.
  **Concluído em 2026-10-24:** o saldo saiu pelo item 40 e as faturas do cartão pelo item 33a (fatia a). As tendências seguem calculadas no navegador, de propósito: são uma simulação que só exibe.
- 12. [ ] <a id="item-12"></a>**[Depois] Colaboração, Fase 1:** log de operações + atualização incremental da tela por gatilho (o contador
  `seq` e a checagem barata já existem; falta o log do que mudou, para atualizar só os registros afetados); quem
  criou/alterou/ocultou; "novo desde a última visita"; atividade recente; idempotência e edição concorrente
  (exige trabalho no front também).
- 13. [ ] <a id="item-13"></a>**[Depois] Colaboração, Fase 2:** presença em tempo real ("fulano está mexendo neste registro agora").
- 15. [ ] <a id="item-15"></a>**[Depois] IA** ("Preencher com IA") via Lambda, com a chave da Anthropic no SSM Parameter Store.
- 18. [ ] <a id="item-18"></a>**[Depois] Segurança e escala:** revisar o `localStorage` do token (front), mesclar conflito de configuração (hoje o front repete por cima), avaliar índice (GSI) por banco.
- 28. [x] **Avisos de lançamentos futuros no Telegram**: campo `aviso` no lançamento (dias antes do vencimento e
  "insistir todo dia depois"), desligado por padrão e independente do status (o PREVISTO continua sendo só a previsão
  que soma em Futuros); rotina diária às 8h pela mesma Lambda (EventBridge), um resumo por pessoa (quem criou e o
  dono); token do bot no SSM; chat id colado em Listas, com botão de teste; mensagens do dev começam com `[DEV]`
  (os dois ambientes usam o mesmo bot). No ar e validado no dev e na produção (v0.8.0; PRs #49, #52, #54 e #55). A
  configuração única está em `.ai/avisos.md`. Depois: vínculo automático por link (webhook), outros canais (Web Push,
  e-mail) e respeitar a visibilidade por pessoa do item 27.
  Limite conhecido, decidido não tratar agora: o resumo do dia vai numa única mensagem e o Telegram recusa acima de
  4096 caracteres (uns 40 avisos no mesmo dia para a mesma pessoa); nesse caso nenhum aviso chega naquele dia. A
  correção seria dividir o resumo em mensagens menores (detalhe em `.ai/avisos.md`).
- 27. [ ] <a id="item-27"></a>**[Depois] Tela de permissões por usuário** (hoje só há dois papéis fixos, `owner` e `member`, na matriz de
  `auth/authz.go`): centralizar numa tela do dono o que cada pessoa pode ver e fazer. Motivos vindos do uso real: a esposa
  clica em "valor estimado" ao lançar (campo que hoje não precisa) e vê cartões e saldos de bancos que não são dela.
  Decidido:
  - **Valor estimado** é permissão (liga/desliga por pessoa), não preferência de tela: ela pode passar a fazer
    compras em dólar e então precisar informar o valor estimado.
  - **Cartões e saldos** são por **banco e cartão individual** (não por aba inteira).
  - **Banco ou cartão novo nasce invisível** para quem não é dono, até o dono liberar.
  - **Coerência dos números:** esconder um banco ou cartão para uma pessoa tem que se comportar como o "ocultar"
    do Histórico: nada dele entra em soma nenhuma na tela dela (saldos, faturas, Histórico, Futuros, Tendências,
    totais, filtros por tag). Diferença importante: o `oculto` do lançamento vale para todo mundo e tira o valor do
    saldo; a visibilidade por pessoa **não muda o saldo real**, só o que cada uma enxerga (o dono continua vendo tudo).
  A decidir: modelo (permissão por usuário ou perfis nomeados) e onde guardar (item do usuário no espaço, perto do
  `MEMBER#sub`). Aplicar no **backend**, não só esconder no front: a API não devolve lançamentos, saldos (itens
  `SALDO#<banco>`) nem faturas do que a pessoa não pode ver. Como o navegador é quem calcula o saldo a partir do que
  recebe, filtrar na origem já mantém as contas dela coerentes; conferir os agregados que misturam bancos (totais,
  Tendências) e as telas de conferência e fechamento, que continuam só do dono.

### Front (funcionalidades novas)
- 7. [x] **Parte B dos eventos futuros: botão Prever e repetição ao lançar**: cópias nos meses seguintes (mensal,
  trimestral, semestral, anual ou a cada N meses), parcelas numeradas, aviso de duplicado, aviso herdado das cópias,
  "Estender série" nas linhas de uma série. Detalhes em [`.ai/eventos-futuros.md`](.ai/eventos-futuros.md).
- 8. [x] **Carregar os anos anteriores** (2025 carregado e conferido com a verificação arquivo x banco de dados). Novas
  cargas virão: meses anteriores à data de início do banco são histórico e não são travados pelo fechamento mensal.
  Antes de importar, conferir na planilha as datas fora do mês da aba (no app o mês é o da data, não o da aba).
- 9. [x] **Importação do `.xlsx` usando o endpoint de lote** (25 por chamada, com contador de progresso e erro tratado).
- 23. [x] <a id="item-23"></a>**Fechamento mensal, melhorias** (descartado em 2026-10-26, por decisão do dono): os meses fecham sozinhos
  (item 38) e a conferência com o extrato é opcional, então "ver mais" na lista, aviso de mês conferido que deixou de
  bater, "reabrir tudo" e conferir meses anteriores sem reabrir os mais novos não são mais necessários. Nada foi feito.
- 17. [x] <a id="item-17"></a>**[Depois] Cartão: valor pago e observação** por fatura (exceções como extrato diferente do pago).
  Evoluído e absorvido pelo item 35 (concluído em 2026-10-25).
- 35. [x] <a id="item-35"></a>**[Depois] Pagar fatura: meio pagador, vencimento e vínculo com o pagamento.** A **operadora manda**: o extrato dela
  pode somar 3.000 e o boleto vir de 2.500, então o pagamento nasce **da fatura**, não da soma dos lançamentos do sistema
  (decisões do dono, 2026-10-24).
  - **Três valores por fatura:** calculado (a soma dos nossos registros de `CREDITO`), o da operadora (`valorBanco`) e o do boleto, que é
    o que se paga. O sistema propõe, o dono confirma ou corrige, e a diferença fica à vista (hoje vira um lançamento de valor zero como nota solta).
  - **Meio pagador:** escolhido na seção do cartão, na aba Cartão. Vale para o cartão inteiro como padrão e **pode ser trocado em cada fatura**,
    porque ninguém sabe onde haverá saldo (cartão do SANTANDER pago pelo NUBANK, em débito).
  - **Vencimento:** a data **de cada fatura** (o dia do cartão só sugere, não vale para nenhuma fatura; decisão do dono, 2026-10-25). É a data em que a fatura aparece em Futuros; sem ela, a fatura não aparece.
  - **Em Futuros**, no meio pagador, até duas linhas por cartão, **calculadas pelo servidor** (não são previstos gravados):
    **fatura fechada** (valor **congelado** no fechamento: só mudam o valor da operadora, o do boleto e ajuste ou estorno explícito;
    passado o vencimento sem pagar, fica como **atrasada**) e **fatura aberta** (total vivo, muda a cada compra no crédito). Ao fechar,
    no dia seguinte ao fechamento, a aberta vira fechada e a seguinte ganha a sua linha. Compra com data anterior lançada depois do
    fechamento segue a regra de hoje: o `faturaAjuste` de ±1 a leva para a fatura vizinha; o valor congelado só muda por ajuste explícito.
  - **Confirmar o pagamento** reaproveita o gesto de confirmar do Futuros (trocar meio, valor e status; demo
    `.ai/demos/demo-confirmar-previsto.html`). Grava o lançamento no meio pagador, o liga à fatura, e a linha some.
  - O pagamento mexe no saldo do meio pagador e **não conta de novo como gasto**: o gasto já contou nos registros de `CREDITO`, que são
    reais, não afetam o saldo e não se pode perder o controle deles. Caso real: estorno no dia do corte, fatura de um valor e pagamento de outro.
  - **Pagamento mínimo ou parcial** (a maioria dos usuários usa; o dono não) fica aqui: a fatura passa a ter mais de um pagamento.
  - **Dados novos aprovados pelo dono em 2026-10-25:** `pagador` e `diaVencimento` no cartão e `pagador` e `vencimento` por fatura (tudo dentro da entrada do
    cartão em `CFG#CARTOES`). O atributo `pagaFatura` no lançamento aguarda a aprovação no spec. **O pagamento é um lançamento normal e a marca `pagaFatura` é só uma label** (sem limite de pagamentos, de valor ou de meio; pagamento mínimo e restante são da operadora, nada passa adiante). Depende dos itens 33a e 41,
    concluídos. Intent, plan e spec (aprovados em 2026-10-25; **concluído em 2026-10-25: fases A (backend, #228) e B (telas, #230), painel do cartão refeito em linhas compactas (#231) e o débito do mesmo banco pode pagar a fatura (#233); conferido em dev pelo dono**) em [`.ai/specs/pagar-fatura/`](.ai/specs/pagar-fatura/intent.md). Detalhes em `.ai/arquitetura.md`.
- 20. [ ] <a id="item-20"></a>**[Depois] Ideias** (seção no fim deste arquivo).

### UX/UI (usabilidade do que já existe)
- 30. [x] <a id="item-30"></a>**Formulário de Lançar no padrão de telas** (concluído; confirmado pelo dono em 2026-10-26): hoje o Lançar ainda mostra à vista os controles antigos de
  aviso e de repetição (17 controles, pouco usáveis no celular). Passar "Repetir" e "Avisar" para linhas-resumo que
  abrem as telas cheias do Prever (a lógica e os componentes já existem em `web/index.html`, bloco "Prever e Estender").
  Pelo padrão, começa com uma demo do formulário completo para aprovação.
- 21. [x] <a id="item-21"></a>**Próximos passos de UX** (encerrado em 2026-10-26: o dono considera bom o uso no celular como está; o que sobrou não será feito) (vindos de observar o uso real; detalhes em [`.ai/ux.md`](.ai/ux.md)): barra de navegação
  inferior no celular, botões pequenos de Cartão e Saldos, "Cancelar edição"/"Limpar formulário" perto do Salvar,
  acessibilidade das abas, unificar o seletor de tags (agora no item 42).
- 48. [x] <a id="item-48"></a>**Aviso padrão do cartão, copiado para a fatura ao definir o vencimento** (concluído em 2026-10-26 no #249; conferido em dev pelo dono): o cartão ganha `porBanco[meio].aviso` (mesmo formato do aviso da fatura, só um modelo). Quando o dono define o **primeiro vencimento** de uma fatura (o momento em que ela vira registro em Futuros) e a fatura ainda não tem aviso, o servidor **copia** o padrão para `faturas[ym].aviso` na mesma gravação; depois o dono muda pelo botão Avisar. Mudar o padrão não altera faturas já copiadas; nenhuma leitura grava; o passado não é tocado. Dado novo: `porBanco[meio].aviso` (aprovado pelo dono em 2026-10-26). Em Configurar cartão, linha **Aviso (padrão)** que abre a mesma tela de aviso (demo antes). Extensão do item 46, em [`.ai/specs/aviso-fatura/`](.ai/specs/aviso-fatura/intent.md).
- 47. [ ] <a id="item-47"></a>**[Próximo] Tela "Avisar no Telegram" como componente reutilizável (2026-10-26):** hoje a tela de aviso são funções soltas do Prever (`pvTelaAvisar`, `pvBalao`, o estado `pv`), amarradas a um lançamento. O dono quer que ela seja um **padrão**, reformulado para ser reaproveitado pelas buscas do Histórico, das Tendências e do Futuros (item 42) e pelo aviso da fatura (item 46). Pede um componente com entradas claras (título, vencimento de referência, textos, balão, o que o Salvar faz) e uma demo própria antes de qualquer tela, atualizando `.ai/padrao-de-telas.md`. O item 46 usa a tela como está, só parametrizada, para não esperar este item; este item depois troca os dois pelos mesmo componente. Junto do item 42.
- 42. [ ] <a id="item-42"></a>**[Próximo] Filtro único (tags, bancos, status e combos) e lista da consulta da Tendências** (guarda-chuva; raciocínio e regras em
  [`.ai/ux.md`](.ai/ux.md), seção "Filtro único e lista da consulta"). Hoje Tendências (filtro avançado de tags com E, OU e NÃO), Histórico (status, banco, tags e
  combos, só "todas" ou "qualquer uma") e Futuros têm cada um o seu filtro, e os números e as consultas divergem. A ideia é **um filtro compartilhado**: uma janela
  própria, aberta pela tela que a chama dizendo **quais blocos quer** (a Tendências só tags e combos; o Histórico tags, bancos e status), em duas partes, a lógica
  pura (testada) e a janela que só desenha. Cada tela é dona do seu estado, as opções vêm da tela que chama e nada é enviado ao servidor. Cada passo pede demo
  (janela nova ou tela mexida) e a janela nova entra no botão voltar (`camadasAbertas` e `fecharCamadaTopo`). Ordem:
  - [x] **a) Lista dos lançamentos da consulta, na Tendências** (implementada e aprovada em 2026-10-22; falta testar no `dev`) (decidido em 2026-10-22: ficar na Tendências e testar em produção). Janela só de leitura aberta
    por um botão ao lado do total; tocar numa barra do mês filtra a lista; a soma da lista bate com o gráfico e a lista respeita a visão (Gastos, Receitas ou
    Resultado); limite de linhas na tela (proposta: 200, com aviso). A Tendências é uma **simulação**: não edita, não oculta e não grava nada.
  - [ ] **b) Extrair o filtro de tags** que já existe (`consulta-tags.js` e o seletor de `tendencias.js`) para o componente compartilhado, sem mudar o que a Tendências faz.
  - [ ] **c) Histórico passa a usar o filtro**, ganhando o filtro avançado de tags. É a tela do saldo real: demo e cuidado extra.
  - [ ] **d) Futuros:** ler o que ele filtra hoje e decidir se adota o componente.
  - [ ] **e) "Abrir no Histórico" a partir da Tendências**, levando a consulta e o mês, para quem quiser editar ou ocultar. Depende do c.
  - [ ] **g) O total e a soma por mês do Histórico, de volta** (2026-10-24): o item 41 **retira** da tela do Histórico o total da lista e a soma por mês (pedido do dono: o front não soma). Este
    item decide como eles voltam: **calculados no servidor**, o que exige que o filtro de tags e combos (hoje só no navegador, "nada é enviado ao servidor") **passe a ser entendido pelo servidor**, e o
    resumo do filtro inteiro (`n`, `total`, por mês) volta junto. Demo antes do código.
  - [ ] **f) Detalhes já vistos** (o aviso de que o mês mais antigo pode estar incompleto foi feito em 2026-10-23; o corte em si continua): gráfico parado nos meses antigos ao girar o celular ou redimensionar a janela; decidir se Gastos/Receitas/Resultado deve ser lembrado
    (hoje volta a Gastos ao recarregar); o gráfico longo é limitado aos 1000 lançamentos mais recentes carregados.

### Processo e qualidade (testes, releases, docs)
- 10. [x] **Primeira release `v0.1.0`** (2026-10-04): PR `develop → main` (#6), tag anotada e release no GitHub.
- 16. [x] <a id="item-16"></a>**Testes automatizados** do front (descartado em 2026-10-26: a lógica de faturas e de eventos futuros foi para o servidor, que tem seus testes; o front já tem os `web/*.test.js` do que sobrou nele).
- 26. [x] **Tag e release automáticas após o deploy da produção**: o job `lancar-versao` (em `deploy-prod.yml`) cria a
  tag anotada e a release depois do deploy da `main`. Versão pelos commits (`feat:` sobe o número do meio, o resto
  sobe o último) com os rótulos `release:minor|patch` no PR de release por cima; o PR de release recebe um comentário
  com a versão calculada. No ar (PRs #44 e #45, release v0.7.0). Detalhes em `.ai/deploy-github-actions.md`.
- 25. [x] **PRs automáticos**: push em `feature/**` (e `fix/`, `bug/`, `docs/`, `chore/`) abre o PR para a
  `develop` (inicialmente em rascunho; desde 2026-10-20, já pronto para mergear); depois do deploy do dev, abre o PR
  `develop → main`. No ar (PR #44). O `CI` por `pull_request` do PR criado pelo robô esperava "Approve and run" (limite
  do `GITHUB_TOKEN`; vermelho sem efeito); por isso o CI passou a rodar só em `push`, e o check dele aparece no PR.
- 19. [x] **Script de release**: dispensado e substituído pela automação dos itens 25 e 26 (PR de release aberto pelo
  robô, tag e release criadas depois do deploy da produção). Funcionou na v0.7.0.
- 36. [x] <a id="item-36"></a>**Remover `financas-app.html`** (concluído em 2026-10-26: arquivo e linha do README apagados) (versão original em arquivo único, de quando o app rodava no ambiente
  do Claude, com 2188 linhas e cópia da mesma lógica que hoje está em `web/index.html`). Só o README o cita; nenhum
  deploy, workflow, template ou script o usa. Apagar o arquivo e a linha do README.

### Operação (tarefas de uso e dados, não de código)
- 6. [x] **Cartão: conferir as faturas** dos 4 cartões (MERCADO PAGO, NUBANK, INTER, SANTANDER) contra o banco e
  definir o **início do controle** de cada um (usando a aba Cartão).
- 1. [x] **Mergear a PR #3** (nome de exibição) e **a #4** (eventos futuros) em `develop`.
- 4. [x] **Convidar a esposa** (`scripts/seed.sh financas <dono> <email-dela>`) e instalar na tela inicial do celular.
- 5. [x] **Limpar os PREVISTO e TRANSFERINDO antigos** da importação: aparecem como atrasados em Futuros (Ocultar, ou
  Editar para o status que de fato aconteceu).

## Já feito (resumo)
- Backend AWS (SAM + Go + DynamoDB + Cognito), publicado; login com Cognito (Hosted UI + PKCE).
- Importação de backup em lote; exportar e importar lançamentos e configurações separadamente; importar fechamentos
  da planilha; importar notas do `.xlsx`; escape de HTML.
- Cartão: início do controle, fatura conferida, `CREDITO IN`/`EX` como na planilha; verificação de duplicidade
  sem o limite de 1000.
- Autorização por papel no backend (owner/member) com testes; front por papel; teste com um member de verdade.
- Nome de exibição (PR #3). Eventos futuros, Parte A (PR #4).
- Hospedagem do front em S3 privado + CloudFront (PR #5). Primeira release `v0.1.0` na `main`.
- UX de Lançar e Histórico (v0.2.0, PR #11): aviso ao salvar, botão ocupado contra clique duplo, erro visível, data na
  transferência, Status/Tipo/Valor na mesma linha, Histórico abre no mês corrente com setas e botão Mês atual, ordem
  por botão Data ↓/↑, Enter dispara a busca. Guarda de publicação nos scripts (PR #10).
- Filtro por combos e tags no Histórico (painel recolhido) e nas Tendências, "todas as tags" como padrão (v0.3.0,
  PR #13).
- Remover status migra os lançamentos para outro status; remover banco é bloqueado se em uso; X dos bancos só para o
  dono (v0.3.1, PR #15).
- Fechamento mensal por banco (v0.4.0, PR #20): o saldo parte do último fechamento válido e lê só os meses seguintes;
  conferência com o extrato com ajuste automático da diferença; mês conferido trava ele e os anteriores (só o dono
  confere e reabre, e só o mais recente reabre); meses anteriores à data de início são histórico e nunca travam.
  Item `SALDO#<banco>` no DynamoDB e rotas `/fechamentos`.
- Verificação arquivo x banco de dados na importação (v0.4.0, PR #21): compara o arquivo com o que ficou gravado, por
  ano e banco, com os meses ao tocar na linha; chave igual é o mesmo lançamento (o app não grava duplicados).
- Sincronização da tela por contador `seq`: cada escrita soma 1 no item `SEQ` do espaço; a escrita própria atualiza só
  aquele registro no cache (write-through, sem reler os 1000 lançamentos); a checagem a cada 3 min lê só o `seq`. Veio
  de excluir em série estourar a capacidade de leitura da tabela (5 RCU).
- Importação da planilha em lote (PR #27): 25 linhas por chamada, ids gerados na tela (reenviar nunca duplica), contador
  `N/total (%) · tempo` que continua andando enquanto espera o banco, tentativas automáticas em falha de rede/5xx e
  aviso claro de onde parou. Veio de importar 1413 lançamentos um a um estourar a capacidade de escrita da tabela.
- Scripts: `deploy.ps1`, `aws-mfa.ps1`, `publicar-front.ps1`, `seed.sh`, `definir-nome.sh`, `limpar-espaco.py`.

## Documentos de apoio
O que não é backlog vive em [`.ai/`](.ai/README.md): regras para a IA ([`guardrails.md`](.ai/guardrails.md)),
arquitetura, segurança e permissões, fluxo de versões, hospedagem, ambiente de teste, eventos futuros, colaboração e UX.

## Ideias
- `#Garantia` com filtro de garantias vigentes (data de vencimento na nota).
- Anexar foto/PDF da nota fiscal ao lançamento (bucket S3).
- Fatura: aviso quando uma compra `CREDITO EX`/`CREDITO IN` cair fora do esperado (a planilha as ignora).
- Aviso "já existe um igual" ao salvar manualmente usa só os 1000 mais recentes carregados (hoje só a importação
  confere o histórico inteiro).
- **Congelamento por data de corte** para aposentar status e termos sem migrar o passado: o servidor compara a data do
  lançamento com o corte e manda o front o que vale (vocabulário e regra antigos antes do corte, os novos depois). Uma
  data no lugar de um marcador "legado" por status. Candidato para o item 33 (`CREDITO IN`/`EX`, `TRANSFERINDO`,
  `PREVISTO`) e para o rótulo de `PAGO` (Pago/Recebido). A decidir: qual data compara (`dataEvento` ou `criadoEm`), corte
  em início de mês (alinhado ao fechamento), um corte global ou um por regra, e onde guardar o campo.
  **Superado em 2026-10-24:** não haverá corte global. O catálogo de status tem uma entrada por nome e a marca `selecionavel`; `desde` e `ate` só sugerem a lista de uma
  data; o passado é do usuário (pode reclassificar); a vigência dos meios é do item 43. Ver itens 41 e 43. (O item 44, troca em massa do admin, foi removido em 2026-10-27.)
