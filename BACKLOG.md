# Backlog e roadmap

Atualizado em 2026-10-05. O roadmap é separado por **área de responsabilidade**; dentro de cada área, os itens estão em
ordem de prioridade e levam a etiqueta **[Agora]**, **[Próximo]** ou **[Depois]**. A numeração é fixa (não renumerar:
outros trechos referenciam os números). Os detalhes ficam nas seções depois do roadmap. Marque `[x]` ao concluir.

## Roadmap

### Bugs (têm prioridade sobre o resto)
Cada item começa com a área correspondente entre colchetes. A numeração segue a mesma sequência do roadmap.
- 25. [x] **[UX/UI] Combo de bancos volta sozinho ao primeiro item ao lançar após copiar pelo Histórico**, o que pode
  gerar erro no cadastro. Detalhe em "UX".

### Infra (AWS, SAM, ambientes, custo, backup)
- 2. [x] **Decidir o `/api/*`: chamada direta à Function URL (opção B)**, front estático no CloudFront. Reavaliar
  rotear a API pelo CloudFront (OAC) quando entrar domínio próprio, WAF ou mais usuários. Detalhe em "Hospedagem".
- 3. [x] **Hospedar o front em S3 + CloudFront** (HTTPS, `*.cloudfront.net`). No ar (PR #5). Detalhes em "Hospedagem".
- 22. [x] **Ambiente de teste (`dev`)**: segunda stack `financas-dev` com tabela, login, API e front próprios,
  dados fictícios, sempre ligada, publicada à mão a partir da `develop`. No ar (PRs #33, #34 e #35). Detalhes em
  "Ambiente de teste".
- 24. [x] **Deploy por GitHub Actions (OIDC)**: `develop` publica o dev, `main` publica a prod com aprovação manual,
  PR só valida; roles IAM por OIDC separadas por ambiente, sem chave nos secrets; o deploy falha se o changeset remover
  ou substituir a tabela ou o login. No ar (PRs #38, #39 e #40, release v0.6.0). Guia em
  `docs/deploy-github-actions.md`.
- 11. [ ] **[Depois] AWS Budget de US$ 1** com alerta por e-mail (conferir se existe) e **backup periódico**
  (exportar JSON e/ou point-in-time recovery do DynamoDB). Adiado por decisão: não será feito agora.

### Backend (Go, DynamoDB, Cognito, API)
- 14. [ ] **[Depois] Saldos e faturas do cartão calculados na Lambda** (o saldo do banco já lê só o mês aberto, graças
  ao fechamento mensal; faltam as faturas do cartão e as tendências).
- 12. [ ] **[Depois] Colaboração, Fase 1:** log de operações + atualização incremental da tela por gatilho (o contador
  `seq` e a checagem barata já existem; falta o log do que mudou, para atualizar só os registros afetados); quem
  criou/alterou/ocultou; "novo desde a última visita"; atividade recente; idempotência e edição concorrente
  (exige trabalho no front também).
- 13. [ ] **[Depois] Colaboração, Fase 2:** presença em tempo real ("fulano está mexendo neste registro agora").
- 15. [ ] **[Depois] IA** ("Preencher com IA") via Lambda, com a chave da Anthropic no SSM Parameter Store.
- 18. [ ] **[Depois] Segurança e escala:** revisar o `localStorage` do token (front), limite de taxa na Function URL
  (infra), mesclar conflito de configuração (hoje o front repete por cima), avaliar índice (GSI) por banco.
- 27. [ ] **[Depois] Tela de permissões por usuário** (hoje só há dois papéis fixos, `owner` e `member`, na matriz de
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
- 7. [ ] **[Próximo] Parte B dos eventos futuros: botão Prever** (cópias nos meses seguintes, parcelas, aviso de
  duplicado).
- 8. [x] **Carregar os anos anteriores** (2025 carregado e conferido com a verificação arquivo x banco de dados). Novas
  cargas virão: meses anteriores à data de início do banco são histórico e não são travados pelo fechamento mensal.
  Antes de importar, conferir na planilha as datas fora do mês da aba (no app o mês é o da data, não o da aba).
- 9. [x] **Importação do `.xlsx` usando o endpoint de lote** (25 por chamada, com contador de progresso e erro tratado).
- 23. [ ] **[Depois] Fechamento mensal, melhorias:** "ver mais" na lista (hoje mostra os últimos 12 meses); aviso quando
  um mês conferido deixa de bater com o que foi guardado; "reabrir tudo" de uma vez (hoje reabre do mais novo para o
  mais antigo, um por vez); conferir meses anteriores ao último conferido sem reabrir os mais novos.
- 17. [ ] **[Depois] Cartão: valor pago e observação** por fatura (exceções como extrato diferente do pago).
- 20. [ ] **[Depois] Ideias** (seção no fim).

### UX/UI (usabilidade do que já existe)
- 21. [ ] **[Próximo] Próximos passos de UX** (vindos de observar o uso real; detalhes em "UX"): barra de navegação
  inferior no celular, botões pequenos de Cartão e Saldos, "Cancelar edição"/"Limpar formulário" perto do Salvar,
  acessibilidade das abas, unificar o seletor de tags.

### Processo e qualidade (testes, releases, docs)
- 10. [x] **Primeira release `v0.1.0`** (2026-10-04): PR `develop → main` (#6), tag anotada e release no GitHub.
- 16. [ ] **[Depois] Testes automatizados** do front (a lógica de faturas e de eventos futuros).
- 26. [x] **Tag e release automáticas após o deploy da produção**: o job `lancar-versao` (em `deploy-prod.yml`) cria a
  tag anotada e a release depois do deploy da `main`. Versão pelos commits (`feat:` sobe o número do meio, o resto
  sobe o último) com os rótulos `release:minor|patch` no PR de release por cima; o PR de release recebe um comentário
  com a versão calculada. No ar (PRs #44 e #45, release v0.7.0). Detalhes em `docs/deploy-github-actions.md`.
- 25. [x] **PRs automáticos**: push em `feature/**` (e `fix/`, `bug/`, `docs/`, `chore/`) abre o PR rascunho para a
  `develop`; depois do deploy do dev, abre o PR `develop → main`. O CI roda também em `push`. No ar (PR #44). O
  `CI` por `pull_request` do PR criado pelo robô espera "Approve and run" (limite do `GITHUB_TOKEN`); não bloqueia.
- 19. [ ] **[Depois] Script de release** (`scripts/lancar-versao.ps1 <versão>`): PR `develop → main`, merge, tag e
  release em um comando (ver "Fluxo de branches e versões"). Pode ser dispensado pelos itens 25 e 26.

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

## Fluxo de branches e versões (convenção simples)
- **Trabalho:** uma branch nova a partir da `develop` por assunto (`feature/...`, `docs/...`) e PR para a `develop`.
- **Lançar uma versão:** PR `develop → main`, merge (commit de merge), depois a **tag anotada** `vX.Y.Z` no commit da
  `main` e a release no GitHub. A tag marca **o que foi para produção**. A release é só a página de registro: não
  publica nada.
  ```
  git fetch origin
  git tag -a v0.2.0 -m "v0.2.0: resumo" origin/main
  git push origin v0.2.0
  gh release create v0.2.0 --verify-tag --title "v0.2.0" --notes "..."
  ```
- **Número:** `0.x.y` até ficar estável. Correção = sobe o último número (`0.1.1`); funcionalidade nova = sobe o do meio
  (`0.2.0`); `1.0.0` quando estável.
- **Bug em produção:** se a `develop` só tem coisas já lançadas, corrige na `develop` como qualquer assunto e lança a
  versão seguinte. Se a `develop` já tem coisa que não foi lançada, cria `hotfix/...` a partir da `main`, PR para a
  `main`, tag da correção e PR de volta para a `develop` para a correção não se perder. Só ir à tag antiga se for
  preciso reproduzir o bug (`git checkout v0.1.0`).
- **Sem branches `release/x.y`** por enquanto; só se for preciso congelar uma leva grande para testar enquanto a
  `develop` segue em frente.
- **Publicar na AWS é separado** da release: backend com `deploy.ps1`, front com `publicar-front.ps1`. Para a versão
  no ar bater com a tag, publicar a partir da `main` depois de lançar.

## Decisões fechadas

| Ação | owner | member |
|---|---|---|
| Criar lançamento | sim | sim |
| Copiar lançamento | sim | sim |
| Editar / ajustar fatura | todos | **só os criados por ela** (`criadoPor == sub`) |
| Ocultar lançamento | todos | **só os dela** |
| Excluir lançamento | todos | **não** |
| Criar tag | sim | **só tags que não existem** (o servidor só acrescenta) |
| Renomear/excluir tag, combos, bancos, status, flags | sim | não |
| Ver saldos, cartão, tendências, futuros | sim | sim (só leitura) |
| Alterar saldo inicial, fechamento, valor do banco, conferir fatura, início do controle | sim | não |
| Conferir e reabrir mês de um banco (fechamento mensal) | sim | não (só vê) |
| Aparência (cores, layout, ordem) e o próprio nome | sim | sim, **só a dela** |
| Importar / exportar / limpeza | sim | não |

- Ela lê **todos** os lançamentos e notas (saldos, cartão e tendências são calculados no navegador).
- Hospedagem: **S3 + CloudFront** (GitHub Pages descartado; `localhost` não serve para ela: sem HTTPS o PKCE e o
  Cognito não funcionam).
- Descartado: bloquear duplicidade por conteúdo (a mesma compra pode ter outro horário e outra descrição).
- Atualização da tela por gatilho (foco da aba, troca de aba, antes de salvar, botão Atualizar, checagem lenta de 3 a
  5 min). **Sem temporizador curto** (15 s foi descartado).

## Detalhes

### Hospedagem (S3 + CloudFront) e decisão sobre o `/api/*`
**Decidido: opção B**, a API continua sendo chamada direto na Function URL (com CORS), e o CloudFront serve só o front.
Motivos: o token do Cognito já protege os dados; rotear a API pelo CloudFront (OAC) exige `AuthType: AWS_IAM`, hash do
corpo em `x-amz-content-sha256` nos `POST`/`PUT`, token em outro cabeçalho (o OAC sobrescreve o `Authorization`) e
complica o desenvolvimento em `localhost`, e o ganho de segurança é pequeno (o CloudFront continua público, então um
anônimo ainda chega à Lambda por ele; limitar abuso de verdade pede WAF, que custa). Reavaliar com domínio próprio,
WAF ou mais usuários; a troca é localizada (URL base em `config.js`, cabeçalho em `api.js` e `auth.go`, template).

Pronto (PR #5, no ar):
- `template.yaml`: bucket S3 privado, OAC, distribuição (HTTPS, cache gerenciado, cabeçalhos de segurança), política do
  bucket; o CORS da Lambda e as URLs de retorno/saída do Cognito aceitam o CloudFront **e** `localhost`; parâmetro
  `ReservedConcurrency` (teto de execuções simultâneas da Lambda), **desligado (0)**: a conta tem limite de 10 execuções
  simultâneas no total (`aws lambda get-account-settings`) e a AWS exige manter 10 sem reserva, então nenhuma reserva
  é possível; esse limite de 10 da conta já funciona como teto natural. Para reservar de verdade, pedir aumento de cota
  em Service Quotas. Outputs novos: `FrontUrl`, `FrontBucket`, `DistributionId`.
- `scripts/publicar-front.ps1`: envia `web/` ao bucket (inclui `config.js`, que fica fora do git) e invalida o cache.
- Passos: sessão MFA, `.\scripts\deploy.ps1 <codigo>` (cria bucket e distribuição; leva alguns minutos), depois
  `.\scripts\publicar-front.ps1` (use `-Simular` antes) e abrir o `FrontUrl`.

### Ambiente de teste (`dev`)
Decidido: **opção A**, uma segunda stack inteira (`financas-dev`) com o mesmo template, em vez de só outra tabela ou de
prefixo na tabela real (isolamento fraco, risco de misturar com dados reais). **Dados fictícios**, ambiente **sempre
ligado**, **publicação manual** (automatizar a partir da `develop` fica para depois).
- Hoje há nomes fixos no `template.yaml`: tabela `FinancasApp`, função `financas-api`, pool `financas`; os scripts
  `seed.sh`, `definir-nome.sh` e `limpar-espaco.py` também fixam `FinancasApp`. Uma segunda stack colidiria.
- Parâmetro `Ambiente` (`prod` por padrão, `dev`). O padrão tem que gerar **exatamente os nomes atuais**, senão o
  CloudFormation tentaria substituir a tabela real (o `DeletionPolicy: Retain` já protege, mas conferir o changeset).
  Em dev: `FinancasApp-dev` etc.
- `samconfig.toml` com perfil `dev` (`sam deploy --config-env dev`), com `CognitoDomainPrefix` próprio.
- Scripts com `-Ambiente dev` (`deploy.ps1`, `publicar-front.ps1`, `seed.sh`, `definir-nome.sh`). A guarda de
  publicação passa a aceitar a `develop` quando o alvo for dev (hoje avisa para quem não está na `main`).
- Front: `web/config.dev.js` (URL da API e login do dev), enviado pelo `publicar-front.ps1 -Ambiente dev`; bucket e
  distribuição separados.
- Dados: script que cria um dono e um membro de teste (e-mail "+dev") e carrega dados fictícios com volume parecido
  (status, bancos, cartões, combos, centenas de tags) para testar as telas como a esposa usa.
- Custo: tabela de teste com 5/5 RCU/WCU; o always-free são 25 no total e a produção já usa 5/5, então cabe.
- **Concluído.** Parâmetro `Ambiente` no template, `-Ambiente dev` nos scripts e `--dev` no `limpar-espaco.py`. A
  publicação é **bloqueada** fora da branch do ambiente (`main` para prod, `develop` para dev), sem opção de pular. O
  perfil `dev` do `samconfig.toml` fica só na máquina (o arquivo é ignorado pelo git): stack `financas-dev`,
  `Ambiente="dev"`, `CognitoDomainPrefix="financas-cicero-dev"`. O `web/config.dev.js` (fora do git) leva
  `ambiente: "dev"`, que acende a faixa laranja "AMBIENTE DE TESTE" e o prefixo `[DEV]` no título da aba.
- Dados: `scripts/dados-ficticios.py` (grava só em `FinancasApp-dev`; ids fixos, rodar de novo sobrescreve; recomeçar do
  zero com `limpar-espaco.py <space-id> --dev --confirmar`). Dono e membro criados com `scripts/seed.sh financas-dev`.
- Fica de fora: configuração de **cartões** e **saldo inicial** por banco nos dados fictícios; as despesas superam as
  entradas, então os saldos ficam negativos (serve para testar esse caso). A URL é a do CloudFront; domínio próprio só
  se entrar domínio para a produção.

### Eventos futuros
Conceito: status marcados como "evento futuro" (PREVISTO, TRANSFERINDO, CREDITANDO) ainda não aconteceram: não contam
como reais, não mexem no saldo, saem do Histórico e aparecem na aba **Futuros**. O marcador manda por cima das caixas
"conta como real" e "afeta saldo" sem apagar os valores guardados. A Parte A está feita (PR #4).

**Parte B, botão Prever** em cada registro do Histórico: cria cópias como PREVISTO, CREDITANDO ou TRANSFERINDO com a
mesma data e hora nos meses seguintes.
- "Repetir N meses" além do original (compras parceladas).
- Dia que não existe no mês vira o último dia, calculado a partir da data original (31/10 gera 30/11, 31/12, 31/01).
- Numeração `1/3` vira `2/3`, `3/3` na descrição.
- Aviso antes de criar se já existe evento igual (status, descrição, valor, banco e data, sem a hora): pular, criar
  mesmo assim ou cancelar.
- Pode ser oferecido também nas linhas de Futuros (estender a recorrência).

Limite conhecido: Futuros e o aviso usam os 1000 lançamentos mais recentes carregados; um evento futuro muito antigo
pode ficar de fora.

### Colaboração em tempo real
Referência: [`docs/colaboracao-tempo-real.md`](docs/colaboracao-tempo-real.md) (gerado no chat do Claude). O problema de
origem: a tela de uma pessoa não percebe o que a outra gravou, alterou ou ocultou (cópia em memória só renovada por
gravação própria ou a cada 2 min).

**Fase 1: log de operações + sincronização incremental**
- Log no servidor: cada criação, edição, ocultar/mostrar, exclusão e mudança de configuração vira uma operação com
  número de sequência (`seq`) por espaço, gravada na mesma transação (`TransactWriteItems`: contador no item `META`,
  item `OP#<seq>` e o registro). Validade de ~180 dias (TTL).
- `GET /spaces/{sid}/ops?after=N`: devolve só o que mudou desde N (inclui exclusões e ocultações) e o `seq` atual. Sem
  `after`, devolve só o `seq`. Leitura liberada para owner e member (entra na matriz de permissões).
- A tela aplica as operações recebidas em vez de recarregar os 1000 lançamentos.
- Em cada lançamento: quem criou/alterou/ocultou e quando (`atualizadoPor/Em`, `ocultoPor/Em`); lista de membros do
  espaço (item espelho `SPACE#id / MEMBER#sub`) para converter `sub` em nome.
- "Novo desde a sua última visita" (etiquetas automáticas novo/alterado/oculto + contagem) e "Atividade recente",
  ambos a partir do log, sem nada para o usuário marcar ou revisar.
- Idempotência e edição concorrente: id do lançamento gerado na tela (criar duas vezes devolve o mesmo registro),
  travar o botão Salvar, e 409 "alterado por outra pessoa" se o registro mudou desde que foi aberto.
- Cuidados: a importação em lote não deve gerar uma operação por item (emitir uma operação-resumo); a transação
  consome ~2× de escrita (capacidade atual 5 por segundo); registros antigos não têm `atualizadoEm`.

**Fase 2: presença** ("esposa está criando um registro"; no registro aberto, "fulano está aqui"; rascunho com
auto-save visível no Histórico). Presença é efêmera, separada dos dados e expira sozinha (~60 a 90 s sem sinal).
- Caminho A (sem serviço novo): polling de um `head.json` no CloudFront com cache de 1 s (atraso de 2 a 3 s).
  Caminho B (depois): AWS AppSync Events (WebSocket gerenciado, centenas de ms, centavos por mês). O modelo de
  operações é o mesmo, então dá para começar em A e migrar.
- Cuidados: o `head.json` público deve ter só o `seq` (presença e nomes vêm da API autenticada); a presença não pode
  incrementar o `seq` dos dados; no AppSync os canais precisam de autorização por espaço, o exemplo usa `aws-amplify`
  (exige bundler; o front não tem build) e é preciso confirmar a disponibilidade em `sa-east-1`.

### UX
Origem: observar a esposa usando o app (sem explicar nada) e anotar onde ela hesita. Repetir a cada rodada.
- **Barra inferior no celular** com 4 ou 5 destinos (Lançar, Histórico, Cartão, Saldos, Mais); Listas, Tendências,
  Futuros e Importar ficam em "Mais". Hoje são 8 abas numa barra que rola na horizontal.
- **Alvos de toque** de Cartão e Saldos (`font-size:12px; padding:4px 10px`) abaixo de 44px; trocar por linha de ações.
- **Acessibilidade:** abas sem `role="tablist"`/`aria-selected`; `‹ ›` das janelas de faturas são `<span>`, não
  funcionam por teclado.
- **Código:** muito `style` inline; `web/index.html` passa de 3.200 linhas (separar CSS e JS).
- **Seletor de tags:** o do Lançar e o dos filtros duplicam lógica; unificar em um componente.
- **Bug (item 25): combo de bancos reseta após copiar pelo Histórico.** Ao copiar um lançamento pelo Histórico e
  lançar o registro, o combo de bancos volta sozinho para o primeiro elemento da lista, e isso pode gerar erro no
  cadastro (o lançamento sai no banco errado sem a pessoa perceber). Investigar onde o formulário de Lançar repopula
  ou reinicia os combos (cópia, salvar, atualização da tela por `seq`) e manter o banco escolhido.
  **Corrigido:** `renderListas()` refazia os `<option>` dos combos (Status, Banco e bancos da transferência) e o
  navegador voltava ao primeiro; agora a escolha atual é guardada e devolvida depois de refazer a lista.
- **Observado e já resolvido:** salvar sem resposta, transferência sem data, Enter sem efeito na busca, tudo na tela
  em Histórico, X de status apagando sem aviso.

### Ideias
- `#Garantia` com filtro de garantias vigentes (data de vencimento na nota).
- Anexar foto/PDF da nota fiscal ao lançamento (bucket S3).
- Fatura: aviso quando uma compra `CREDITO EX`/`CREDITO IN` cair fora do esperado (a planilha as ignora).
- Aviso "já existe um igual" ao salvar manualmente usa só os 1000 mais recentes carregados (hoje só a importação
  confere o histórico inteiro).
