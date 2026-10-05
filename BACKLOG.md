# Backlog e roadmap

Atualizado em 2026-10-05. Uma única ordem de prioridade (as três faixas abaixo); os detalhes de cada item ficam nas
seções depois delas. Marque `[x]` ao concluir.

## Roadmap

### Agora: liberar o app para a esposa
1. [x] **Mergear a PR #3** (nome de exibição) e **a #4** (eventos futuros) em `develop`.
2. [x] **Decidir o `/api/*`: chamada direta à Function URL (opção B)**, front estático no CloudFront. Reavaliar rotear
   a API pelo CloudFront (OAC) quando entrar domínio próprio, WAF ou mais usuários. Detalhe em "Hospedagem".
3. [x] **Hospedar o front em S3 + CloudFront** (HTTPS, domínio `*.cloudfront.net`). No ar e testado no celular
   (PR #5). Detalhes em "Hospedagem".
4. [x] **Convidar a esposa** (`scripts/seed.sh financas <dono> <email-dela>`) e instalar na tela inicial do celular.
5. [x] **Limpar os PREVISTO e TRANSFERINDO antigos** da importação: aparecem como atrasados em Futuros (Ocultar, ou
   Editar para o status que de fato aconteceu).
6. [ ] **Conferir as faturas** dos 4 cartões (MERCADO PAGO, NUBANK, INTER, SANTANDER) contra o banco e definir o
   **início do controle** de cada um. Conferir no navegador a aba Cartão (início do controle, fatura conferida).

### Próximo
7. [ ] **Parte B dos eventos futuros: botão Prever** (cópias nos meses seguintes, parcelas, aviso de duplicado).
8. [ ] **Carregar os anos anteriores** (do mais novo para o mais antigo).
9. [ ] **Importação do `.xlsx` usando o endpoint de lote** (hoje grava um lançamento por requisição).
10. [x] **Primeira release `v0.1.0`** (2026-10-04): PR `develop → main` (#6), tag anotada e release no GitHub.
11. [ ] **AWS Budget de US$ 1** com alerta por e-mail (conferir se existe) e **backup periódico** (exportar JSON e/ou
    point-in-time recovery do DynamoDB).

### Depois
12. [ ] **Colaboração, Fase 1:** log de operações + atualização incremental da tela por gatilho; quem criou/alterou/
    ocultou; "novo desde a última visita"; atividade recente; idempotência e edição concorrente.
13. [ ] **Colaboração, Fase 2:** presença em tempo real ("fulano está mexendo neste registro agora").
14. [ ] **Saldos e faturas do cartão calculados na Lambda** (hoje o navegador lê todo o histórico).
15. [ ] **IA** ("Preencher com IA") via Lambda, com a chave da Anthropic no SSM Parameter Store.
16. [ ] **Testes automatizados** do front (a lógica de faturas e de eventos futuros).
17. [ ] **Cartão: valor pago e observação** por fatura (exceções como extrato diferente do pago).
18. [ ] **Segurança e escala:** revisar o `localStorage` do token, limite de taxa na Function URL, mesclar conflito
    de configuração (hoje o front repete por cima), avaliar índice (GSI) por banco.
19. [ ] **Script de release** (`scripts/lancar-versao.ps1 <versão>`): PR `develop → main`, merge, tag e release em um
    comando, para não depender de lembrar a sequência (ver "Fluxo de branches e versões").
20. [ ] **Ideias** (seção no fim).
21. [ ] **UX, próximos passos** (a partir de observar o uso real; detalhes em "UX"): barra de navegação inferior no
    celular, botões pequenos de Cartão e Saldos, "Cancelar edição"/"Limpar formulário" perto do Salvar.

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

Pronto na branch `feature/hospedagem-s3-cloudfront`:
- `template.yaml`: bucket S3 privado, OAC, distribuição (HTTPS, cache gerenciado, cabeçalhos de segurança), política do
  bucket; o CORS da Lambda e as URLs de retorno/saída do Cognito aceitam o CloudFront **e** `localhost`; parâmetro
  `ReservedConcurrency` (teto de execuções simultâneas da Lambda), **desligado (0)**: a conta tem limite de 10 execuções
  simultâneas no total (`aws lambda get-account-settings`) e a AWS exige manter 10 sem reserva, então nenhuma reserva
  é possível; esse limite de 10 da conta já funciona como teto natural. Para reservar de verdade, pedir aumento de cota
  em Service Quotas. Outputs novos: `FrontUrl`, `FrontBucket`, `DistributionId`.
- `scripts/publicar-front.ps1`: envia `web/` ao bucket (inclui `config.js`, que fica fora do git) e invalida o cache.
- Passos: sessão MFA, `.\scripts\deploy.ps1 <codigo>` (cria bucket e distribuição; leva alguns minutos), depois
  `.\scripts\publicar-front.ps1` (use `-Simular` antes) e abrir o `FrontUrl`.

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
Referência: `colaboracao-tempo-real.md` (gerado no chat do Claude; arquivo do usuário, não versionado). O problema de
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
- **Código:** muito `style` inline; `web/index.html` passa de 2.700 linhas (separar CSS e JS).
- **Seletor de tags:** o do Lançar e o dos filtros duplicam lógica; unificar em um componente.
- **Observado e já resolvido:** salvar sem resposta, transferência sem data, Enter sem efeito na busca, tudo na tela
  em Histórico, X de status apagando sem aviso.

### Ideias
- `#Garantia` com filtro de garantias vigentes (data de vencimento na nota).
- Anexar foto/PDF da nota fiscal ao lançamento (bucket S3).
- Fatura: aviso quando uma compra `CREDITO EX`/`CREDITO IN` cair fora do esperado (a planilha as ignora).
- Aviso "já existe um igual" ao salvar manualmente usa só os 1000 mais recentes carregados (hoje só a importação
  confere o histórico inteiro).
