# Backlog e roadmap

Atualizado em 2026-10-10. O roadmap é separado por **área de responsabilidade**; dentro de cada área, os itens estão em
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

- [\[Backend\] - **\[Agora\] Histórico imutável e versionado:**](#item-31)
- [\[Backend\] - **\[Agora\] Arquitetura em camadas:**](#item-32)
- [\[Infra\] - **\[Próximo\] MFA no login do app (Cognito):**](#item-37)
- [\[Backend\] - **\[Próximo\] Regras no backend, front só exibe:**](#item-33)
- [\[Backend\] - **\[Próximo\] Avisos por despachante e vários canais:**](#item-34)
- [\[UX/UI\] - **\[Próximo\] Formulário de Lançar no padrão de telas:**](#item-30)
- [\[UX/UI\] - **\[Próximo\] Próximos passos de UX:**](#item-21)
- [\[Processo\] - **\[Próximo\] Remover `financas-app.html`:**](#item-36)
- [\[Infra\] - **\[Depois\] AWS Budget de US$ 1 e backup periódico:**](#item-11)
- [\[Backend\] - **\[Depois\] Saldos e faturas do cartão calculados na Lambda:**](#item-14)
- [\[Backend\] - **\[Depois\] Colaboração, Fase 1:**](#item-12)
- [\[Backend\] - **\[Depois\] Colaboração, Fase 2:**](#item-13)
- [\[Backend\] - **\[Depois\] IA ("Preencher com IA"):**](#item-15)
- [\[Backend\] - **\[Depois\] Segurança e escala:**](#item-18)
- [\[Backend\] - **\[Depois\] Tela de permissões por usuário:**](#item-27)
- [\[Front\] - **\[Depois\] Pagar fatura: valor do boleto e vínculo com o pagamento:**](#item-35)
- [\[Front\] - **\[Depois\] Cartão: valor pago e observação:**](#item-17)
- [\[Front\] - **\[Depois\] Fechamento mensal, melhorias:**](#item-23)
- [\[Front\] - **\[Depois\] Ideias:**](#item-20)
- [\[Processo\] - **\[Depois\] Testes automatizados:**](#item-16)

### Infra (AWS, SAM, ambientes, custo, backup)
- 37. [ ] <a id="item-37"></a>**[Próximo] MFA no login do app (Cognito):** hoje o login é só e-mail e senha; num app de finanças, uma senha
  vazada basta para ver tudo (e quem é member lê todos os lançamentos). Ligar o segundo fator no pool, com TOTP por
  aplicativo autenticador (sem o custo do SMS), no `template.yaml`, para os dois ambientes. O backend continua
  validando o ID token como hoje. **A decidir:** obrigatório para todos ou só para o dono; como a pessoa cadastra o
  aplicativo no primeiro acesso (confirmar no `dev` antes da produção); **recuperação** se o celular for perdido (reset
  pelo dono, por script, para ninguém ficar trancada de fora); ajuste no convite de `scripts/seed.sh`. Ao concluir,
  atualizar a seção de autenticação de `.ai/security.md`. Detalhes de contexto em `.ai/security.md`.
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
- 11. [ ] <a id="item-11"></a>**[Depois] AWS Budget de US$ 1** com alerta por e-mail (conferir se existe) e **backup periódico**
  (exportar JSON e/ou point-in-time recovery do DynamoDB). Adiado por decisão: não será feito agora.
  - O **AWS Budget** controla o **custo da própria conta AWS**, não é um orçamento do app de finanças. Define-se um
    limite mensal em dólar (aqui, US$ 1, já que o projeto fica no free tier) e a AWS manda e-mail ao chegar em 80%
    do gasto real e quando a previsão passar de 100%. Protege contra cobrança inesperada (enxurrada de requisições na
    Function URL, recurso esquecido). Console: Billing > Budgets > Create budget > Cost budget; os 2 primeiros são grátis.
  - **Backup:** o PITR do DynamoDB está ligado **só em prod** pelo `template.yaml` (`!If [EhDev, false, true]`; ligar
    no console sem mudar o template seria desfeito no próximo deploy). Falta o backup periódico; enquanto isso, o
    botão Exportar (`.json`) é a cópia manual.

### Backend (Go, DynamoDB, Cognito, API)
Os itens 31 a 34 vêm das decisões de 2026-10-10 (erro grave: uma edição sobrescreveu um registro e o dado se perdeu).
O documento de referência é [`.ai/arquitetura.md`](.ai/arquitetura.md); leia antes de mexer. **Nenhuma chave nova no
DynamoDB sem discutir com o dono.**
- 31. [ ] <a id="item-31"></a>**[Agora] Histórico imutável e versionado:** todo dado que entra vai também para o histórico, inteiro; toda
  criação, edição, ocultação e exclusão gera uma versão (quem, quando, o registro completo), gravada na mesma transação
  do registro atual e com checagem de versão contra edição concorrente. Excluir passa a ser lógico; restaurar cria
  versão nova; o histórico nunca é editado nem apagado (sem método no repositório e `Deny` por IAM). Lançamentos
  primeiro, depois configurações e fechamentos; o estado atual vira a versão 1. **A decidir antes de gravar qualquer
  chave:** o formato dos itens, ligar o PITR do DynamoDB já como proteção (item 11 está adiado) e confirmar a
  exclusão lógica.
- 32. [ ] <a id="item-32"></a>**[Agora] Arquitetura em camadas:** `main` só liga as peças; `controller` recebe o request; `service` decide o
  que o front recebe e recebe tudo por construtor; `domain` com tipos e regras puras que todos usam; adaptadores
  (DynamoDB, Telegram, auth, relógio) atrás de interfaces declaradas pelo service; sem variáveis globais (`db`,
  `ssmCli`, `table`, `agora()`). Entra por fatias verticais, uma regra por vez, com o app no ar. Pré-requisito do 31.
- 33. [ ] <a id="item-33"></a>**[Próximo] Regras no backend, front só exibe:** fatura do cartão (fechamento, base, ajuste e total prontos na
  API, preservando `faturaAjuste` e o fechamento escolhido por mês), início do controle (`historico: true` vindo da API),
  conferência de fatura **travada pelo backend** como o mês do banco (com forma de reabrir), e futuro/real/afeta saldo
  aplicados pelo servidor com as caixas que já existem em Listas (o saldo do mês da aba Futuros, hoje somado no front em
`gruposFuturos`, também passa a vir pronto; o `excluirDoTotal`, hoje gravado pelo front a partir
  do nome `CONTAS`/`TRANSFERINDO`, é outra decisão por etiqueta: a regra passa a ser do servidor, e o que fazer com o
  campo gravado é discussão). Saem do front `ehCredito*`, `ajustePadrao`,
  `faturaBase`, `ehFuturo`, `afetaSaldo`. Os status `CREDITO IN`/`EX` deixam de ser necessários; migração dos
  lançamentos antigos sem `faturaAjuste` fica a discutir. Absorve o item 14.
- 34. [ ] <a id="item-34"></a>**[Próximo] Avisos por despachante e vários canais:** o service só sabe o que avisar e usa um `Avisador`; um
  despachante lê os canais que a pessoa assinou (Telegram, e-mail, WhatsApp), formata, divide e entrega, e uma falha de
  canal não derruba as outras. Cada canal é uma strategy; canal novo é um adaptador novo. Resolve na origem o limite de
  4096 caracteres do item 28.
- 14. [ ] <a id="item-14"></a>**[Depois] Saldos e faturas do cartão calculados na Lambda** (o saldo do banco já lê só o mês aberto, graças
  ao fechamento mensal; faltam as faturas do cartão e as tendências). Absorvido pelo item 33.
- 12. [ ] <a id="item-12"></a>**[Depois] Colaboração, Fase 1:** log de operações + atualização incremental da tela por gatilho (o contador
  `seq` e a checagem barata já existem; falta o log do que mudou, para atualizar só os registros afetados); quem
  criou/alterou/ocultou; "novo desde a última visita"; atividade recente; idempotência e edição concorrente
  (exige trabalho no front também).
- 13. [ ] <a id="item-13"></a>**[Depois] Colaboração, Fase 2:** presença em tempo real ("fulano está mexendo neste registro agora").
- 15. [ ] <a id="item-15"></a>**[Depois] IA** ("Preencher com IA") via Lambda, com a chave da Anthropic no SSM Parameter Store.
- 18. [ ] <a id="item-18"></a>**[Depois] Segurança e escala:** revisar o `localStorage` do token (front), limite de taxa na Function URL
  (infra), mesclar conflito de configuração (hoje o front repete por cima), avaliar índice (GSI) por banco.
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
  `authz.go`): centralizar numa tela do dono o que cada pessoa pode ver e fazer. Motivos vindos do uso real: a esposa
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
- 23. [ ] <a id="item-23"></a>**[Depois] Fechamento mensal, melhorias:** "ver mais" na lista (hoje mostra os últimos 12 meses); aviso quando
  um mês conferido deixa de bater com o que foi guardado; "reabrir tudo" de uma vez (hoje reabre do mais novo para o
  mais antigo, um por vez); conferir meses anteriores ao último conferido sem reabrir os mais novos.
- 17. [ ] <a id="item-17"></a>**[Depois] Cartão: valor pago e observação** por fatura (exceções como extrato diferente do pago).
  Evoluído no item 35.
- 35. [ ] <a id="item-35"></a>**[Depois] Pagar fatura: valor do boleto e vínculo com o pagamento.** A fatura tem três valores (calculado,
  da operadora e do boleto), e o pagamento nasce **da fatura**, não da soma dos lançamentos: o sistema propõe o valor da
  fatura, o dono confirma ou corrige para o do boleto, e o lançamento do pagamento no banco fica ligado à fatura, com a
  diferença à vista (hoje isso vira um lançamento de valor zero como nota solta). O **banco pagador é escolhido**
  (cartão do SANTANDER pode ser pago pelo NUBANK), o pagamento mexe no saldo dele e **não conta de novo como gasto**: o
  gasto já contou nos registros de `CREDITO`, que são reais, não afetam o saldo e não se pode perder o controle deles.
  Caso real: estorno no dia do corte, fatura de um valor e pagamento de outro. **Pagamento mínimo ou parcial** (a maioria dos usuários usa; o dono não)
  fica aqui: a fatura passa a ter mais de um pagamento. Formato dos dados a discutir antes de criar chave. Detalhes em
  `.ai/arquitetura.md`.
- 20. [ ] <a id="item-20"></a>**[Depois] Ideias** (seção no fim deste arquivo).

### UX/UI (usabilidade do que já existe)
- 30. [ ] <a id="item-30"></a>**[Próximo] Formulário de Lançar no padrão de telas**: hoje o Lançar ainda mostra à vista os controles antigos de
  aviso e de repetição (17 controles, pouco usáveis no celular). Passar "Repetir" e "Avisar" para linhas-resumo que
  abrem as telas cheias do Prever (a lógica e os componentes já existem em `web/index.html`, bloco "Prever e Estender").
  Pelo padrão, começa com uma demo do formulário completo para aprovação.
- 21. [ ] <a id="item-21"></a>**[Próximo] Próximos passos de UX** (vindos de observar o uso real; detalhes em [`.ai/ux.md`](.ai/ux.md)): barra de navegação
  inferior no celular, botões pequenos de Cartão e Saldos, "Cancelar edição"/"Limpar formulário" perto do Salvar,
  acessibilidade das abas, unificar o seletor de tags.

### Processo e qualidade (testes, releases, docs)
- 10. [x] **Primeira release `v0.1.0`** (2026-10-04): PR `develop → main` (#6), tag anotada e release no GitHub.
- 16. [ ] <a id="item-16"></a>**[Depois] Testes automatizados** do front (a lógica de faturas e de eventos futuros).
- 26. [x] **Tag e release automáticas após o deploy da produção**: o job `lancar-versao` (em `deploy-prod.yml`) cria a
  tag anotada e a release depois do deploy da `main`. Versão pelos commits (`feat:` sobe o número do meio, o resto
  sobe o último) com os rótulos `release:minor|patch` no PR de release por cima; o PR de release recebe um comentário
  com a versão calculada. No ar (PRs #44 e #45, release v0.7.0). Detalhes em `.ai/deploy-github-actions.md`.
- 25. [x] **PRs automáticos**: push em `feature/**` (e `fix/`, `bug/`, `docs/`, `chore/`) abre o PR rascunho para a
  `develop`; depois do deploy do dev, abre o PR `develop → main`. O CI roda também em `push`. No ar (PR #44). O
  `CI` por `pull_request` do PR criado pelo robô espera "Approve and run" (limite do `GITHUB_TOKEN`); não bloqueia.
- 19. [x] **Script de release**: dispensado e substituído pela automação dos itens 25 e 26 (PR de release aberto pelo
  robô, tag e release criadas depois do deploy da produção). Funcionou na v0.7.0.
- 36. [ ] <a id="item-36"></a>**[Próximo] Remover `financas-app.html`** (versão original em arquivo único, de quando o app rodava no ambiente
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
