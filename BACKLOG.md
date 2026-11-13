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
- 61. [x] **[Front] A subida do app lia `GET /recorrencias`, que não é do Lançar** (achado em 2026-11-10, ao analisar a demora
  da abertura). O `init` de `web/js/app.js` chamava `carregarRecorrencias()` e cada lançamento gravado relia a lista 0,6 s
  depois, **em qualquer tela**, contra a regra "cada tela lê só o que é dela" (`.ai/guardrails.md`). A lista só serve à ação
  "Parar recorrência" do Histórico e de Futuros; a linha "Recorrente" do Lançar não a usa (guarda a escolha em `formRec`).
  Entrou na fase B do item 51 (commit `3cdcf28`) e o `web/carga.test.js` não acusava porque roda só o `api.js`. Havia um
  segundo defeito: se a leitura da subida falhasse, ninguém tentava de novo e o "Parar recorrência" sumia até o próximo
  lançamento mudar. **Correção:** a leitura passa a ser pedida por `irParaAba` ao abrir o Histórico ou Futuros
  (`garantirRecorrencias`); com outra tela aberta, a mudança só marca `recSujo` e a leitura fica para a volta; falha marca
  de novo. Teste: `web/recorrencias-carga.test.js`.

### Seções (índice dos itens pendentes)
Só o que ainda não foi feito, na ordem de prioridade (**[Agora]**, depois **[Próximo]**, depois **[Depois]**), cada um
com o salto para o registro real. **Ao concluir um item, marque `[x]` no registro e tire a linha daqui.** Bugs
pendentes: nenhum.

- [\[Infra\] - 49 - **\[Próximo\] App multiusuário (MFA, cadastro com convite, espaço próprio, isolamento):**](#item-49)
- [\[Backend\] - 34 - **\[Próximo\] Avisos por despachante e vários canais:**](#item-34)
- [\[UX/UI\] - 42 - **\[Próximo\] Filtro único (tags, bancos, status e combos) e lista da consulta da Tendências:**](#item-42)
- [\[UX/UI\] - 47 - **\[Próximo\] Tela de aviso como componente reutilizável:**](#item-47)
- [\[Front\] - 54 - **\[Próximo\] App Android com Capacitor:**](#item-54)
- [\[UX/UI\] - 63 - **\[Próximo\] Avisar a falta de rede em toda tela:**](#item-63)
- [\[Infra\] - 11 - **\[Depois\] Backup periódico:**](#item-11)
- [\[Backend\] - 59 - **\[Depois\] Congelar meio (continua existindo, mas não aceita eventos novos):**](#item-59)
- [\[Backend\] - 39 - **\[Depois\] Open Finance (visão):**](#item-39)
- [\[Backend\] - 12 - **\[Depois\] Colaboração, Fase 1:**](#item-12)
- [\[Backend\] - 13 - **\[Depois\] Colaboração, Fase 2:**](#item-13)
- [\[Backend\] - 15 - **\[Depois\] IA ("Preencher com IA"):**](#item-15)
- [\[Backend\] - 18 - **\[Depois\] Segurança e escala:**](#item-18)
- [\[Backend\] - 27 - **\[Depois\] Tela de permissões por usuário:**](#item-27)
- [\[Front\] - 64 - **\[Depois\] App Android: Play Store, produção aberta e uso sem internet:**](#item-64)
- [\[Front\] - 20 - **\[Depois\] Ideias:**](#item-20)

### Infra (AWS, SAM, ambientes, custo, backup)
- 51. [x] <a id="item-51"></a>**Recorrência automática de lançamentos (concluído em 2026-11-01, em produção; aprovado em 2026-10-29, roteiro em [`.ai/specs/recorrencia/`](.ai/specs/recorrencia/intent.md)):** hoje o dono cria as contas fixas (colégio, internet, água, luz, aluguel) como previstos mês a mês, pelo Prever ou pelo Repetir, que fazem um número fixo de cópias. Ele quer **marcar o lançamento como recorrente uma vez** e deixar o servidor copiá-lo sozinho, com todas as características (valor, banco, tags, nota, aviso do Telegram), até desligar. Decidido com o dono: a recorrência é um lançamento previsto com o atributo `recorrente: { intervalo }` (objeto, no `TX#`), **sem chave nem tipo de registro novo** (o `REC#` foi rejeitado); quem comanda a criação é o item auxiliar do `AVISOS#ATIVOS`, que guarda intervalo (1, 2, 3, 6, 12 meses ou outro), próxima data e o último registro; a cópia nasce **na hora de marcar** (aparece em Futuros com meses de antecedência, para IPVA, IPTU e seguro) e a rotina diária dos avisos cria a seguinte quando a anterior vence (sem janela de 30 dias); a cópia herda a marca e **não leva série** (o Repetir continua com série); **criar não grava histórico**; apagar uma cópia pula o mês; desligar deixa as cópias futuras como estão; religar cria na hora a próxima, sempre para frente. Fases: A (backend, rotina, rotas, testes) e B (tela, **demo antes**, no padrão do Prever). **Aprovado em 2026-10-30. Fase A (backend: marcar, copiar, rotina, rotas) implementada. Fase B (telas) implementada em 2026-10-30 sobre as demos aprovadas** ([Histórico](.ai/demos/arquivo/demo-historico-v2-recorrencia.html), [Lançar](.ai/demos/arquivo/demo-lancar-v2-recorrencia.html) e [Futuros](.ai/demos/arquivo/demo-futuros-v2-recorrencia.html)): "Tornar recorrente" e "Parar recorrência" nas ações do lançamento, linha Recorrente em "Mais" no Lançar, marca 🔁 nas linhas; padrão Mensal; **sem a lista "Recorrentes"** (o dono achou desnecessária). Limite conhecido: "Parar" aparece só na cópia mais recente de cada recorrência; se ela for excluída (pular o ciclo), a recorrência continua ativa e não há como pará-la pela tela. Em produção e em uso pelo dono desde 2026-11-01, com os ajustes do aviso (Editar mostra o aviso da data futura; linha de aviso em "Tornar recorrente").
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
- 58. [x] <a id="item-58"></a>**[Agora] Meio desativado: impacto, quando some e o que deixa de poder mudar (análise, 2026-11-07; primeira tarefa, antes de qualquer código) (concluído em 2026-11-09: backend e telas mergeados nos PRs 386 e 387 (custo da conferência corrigido) e 388 (log por meio do saldo); testado e aprovado pelo dono em dev; o que sobrou virou o item 60):**
  Veio do teste do dono (2026-11-07): ele criou o banco `Teste`, fez um lançamento, desativou (não dava para remover) e o meio continuou em Saldos. A correção que o escondia de Saldos e Cartão de crédito
  foi **revertida** (#370): um meio desativado pode ter saldo, e esconder esse dinheiro está errado. O item 43 entregou só "deixa de ser sugerido"; este item decide o que é, de verdade, desativar.
  **Ideia do dono:** o meio **não pode ser desativado enquanto houver lançamentos futuros para ele**; depois de desativado **não consta em nenhum evento** (Saldos, Cartão de crédito, Futuros) e **só aparece no Histórico**,
  de preferência com uma **marca de desativado**, **sem permitir edições que alterem chaves** como data e valor. A comparação é com o banco, que manda zerar o saldo para encerrar a conta.
  **O que analisar (nesta ordem, só levantamento e decisão, nada de código):**
  1. **O impacto de um cartão desativado hoje, tela por tela:** Cartão de crédito (lista, total, próximo fechamento, "disponíveis"); Futuros (faturas calculadas, previstos, atrasados, resumo do mês);
     Saldos; **avisos** (itens `AVISOS#ATIVOS` de fatura e de lançamento continuam disparando); **recorrências ativas** (a rotina diária cria a próxima cópia no meio desativado);
     Prever, Repetir e Estender; Transferência; **pagador** do cartão e das faturas (o pagador pode ser um meio desativado); conferência de mês e de fatura; importar, restaurar versão e Confirmar
     previsto; Tendências; exportar. Para cada um: o que acontece, o que deveria acontecer.
  2. **Quando um meio pode sair de Saldos, Cartão de crédito e Futuros:** as condições para **desativar**. Candidatas: saldo zero (banco) ou faturas abertas e fechadas todas pagas (cartão); nenhum
     lançamento futuro ou previsto (de qualquer data); nenhuma recorrência ativa; nenhuma fatura a pagar; nenhum aviso pendente. Definir o que é "saldo zero" para um meio **sem início
     do controle** e para um meio **fora do total**. A recusa explica o que falta, como o Remover faz hoje.
  3. **Desativar bloqueia lançar?** Hoje não (R9 do roteiro do item 43: o servidor não valida lançamento contra o `ativoAte`). Pela ideia do dono, passa a recusar criar, importar, restaurar e
     confirmar no meio desativado. Isso **reverte uma decisão do item 43** e precisa de confirmação.
  4. **O que se pode editar num lançamento de um meio desativado:** só o que não muda o saldo (descrição, nota e tags), na mesma lógica do `mudaSaldo` do item 38; recusar valor, tipo, status,
     banco, data, ocultar, excluir e restaurar. Conferir o que o Desfazer e o Histórico de versões fazem com esse meio.
  5. **Histórico:** marca "desativado" nas linhas e no filtro de meio (**demo antes**); o meio desativado continua sendo escolhível no filtro.
  6. **Desativar com data futura** ("fim do mês", "outra data"): conferir a trava na hora ou na data? Se a data for futura, ainda dá para lançar até lá e o saldo pode mudar.
  7. **Reativar:** sempre possível? O meio volta a aparecer em tudo com o que tem.
  8. **Remover versus desativar:** desativar passa a ser "encerrar"; remover continua só para o que nunca teve lançamento. **Furo achado na conversa:** um lançamento excluído logicamente conta como
     uso, então um meio criado por engano e usado uma vez (como o `Teste`) **nunca mais pode ser removido**. Decidir se o Remover aceita um meio cujos lançamentos estão todos excluídos.
  **Também entra aqui (lembrado na conversa):** o cache das faturas de Futuros acompanha a versão de `CFG#LISTAS` (já ficou assim na tentativa revertida); os documentos do item 43
  (`.ai/specs/meios-de-pagamento/`, o R9 e os Limites do `intent.md`, `padrao-de-telas.md`) precisam ser corrigidos quando a regra for decidida; o meio `Teste` do dono, que hoje está desativado com um
  lançamento, precisa de um fim (excluir o lançamento e remover, se o item 8 permitir). **Dado novo no DynamoDB:** nenhum previsto (o `ativoAte` basta); se a análise pedir um (por exemplo `desativadoEm`), só entra com a aprovação do dono.
  **DECIDIDO pelo dono em 2026-11-08 (análise concluída; falta implementar, com a demo do Histórico antes; este item vira o "Encerrar", o "congelar" é o item 59):**
  - **Desativar = encerrar a conta.** Só vale **a partir de hoje**: some "Fim do mês" e "Outra data", porque a conferência é feita na hora do pedido e quem desativa aceita a análise daquele momento.
  - **O servidor recusa desativar, explicando o que falta,** se houver: saldo diferente de zero (arredondado a centavos, também para meio sem início e fora do total); lançamento previsto ou
    com data futura no meio (inclui ponta de transferência futura); recorrência ativa cuja última cópia está no meio (checar à parte: a ponta vencida sem a próxima cópia não é previsto nem futura); cartão com fatura
    aberta ou fechada, total maior que zero e sem marca de paga; aviso pendente de fatura; meio usado como **pagador** de cartão ou de fatura em aberto (recusa até trocar o pagador). O meio não pode participar de nenhum evento.
  - **Depois de desativado, nada novo entra no meio:** o servidor recusa criar, Transferir, Confirmar previsto, recorrência, restaurar versão e trocar o banco de um lançamento para ele (código `MEIO_DESATIVADO`).
    Reverte o R9 do item 43. Três portas a cobrir: `criar`, `CriarCopia` (recorrência) e `Transferir` (que grava por `SalvarLote`).
  - **Importar é a exceção:** num meio desativado entra só o registro com `dataEvento` **anterior** à `dataInicio` do meio (a partir dela o saldo conta); sem `dataInicio`, recusa o registro. Vale só para desativados
    (nos ativos, importar segue livre); a recusa é por registro (os bons entram, os ruins voltam em `rejeitados`, com código novo); a trava de mês conferido continua por cima.
  - **Some de Saldos, Cartão de crédito e Futuros** (o saldo é zero e não há evento, então nada se perde); **só aparece no Histórico, com marca "desativado"** (demo antes) e continua escolhível no filtro de meio.
    Edição de lançamento de meio desativado: só descrição, nota e tags (lógica do `mudaSaldo` do item 38). **Reativar** sempre possível: o meio volta a aparecer com o que tem.
  - **Remover** aceita o meio cujos lançamentos estão **todos excluídos logicamente** (`UsaMeio` passa a ignorar o `excluidoEm`), o que resolve o `Teste`. **Restaurar versão passa a validar o meio contra a lista:** meio removido ou
    desativado é recusado ("cadastre o meio de novo para restaurar"), então o histórico não traz um meio removido de volta. `HIST#` continua imutável. Se um meio for recriado com o mesmo nome, os excluídos do antigo passam a apontar para o novo (não entram no saldo; só valem se restaurados): aceito.
  - **Nenhum dado novo no DynamoDB:** o `ativoAte` basta. Falta, ao implementar: o cache das faturas de Futuros acompanhar a versão de `CFG#LISTAS`; corrigir `padrao-de-telas.md` e o texto de Listas no `index.html`
    ("Desativar um meio não muda o passado…"). Os documentos do item 43 já ganharam a nota de revisão (R9 e Limites) em 2026-11-08.
  - **Ainda não lido (confirmar ao implementar):** Prever, Repetir, Estender, Tendências, exportar e conferência; pelo que se viu, todos passam por `criar`/`ValidarTx` ou leem por banco e herdam o mesmo comportamento.
  **ANDAMENTO (2026-11-08, branch `feature/meio-desativado-encerrar`, nada commitado ainda):** o **backend está pronto e testado** (conferência de pendências incluindo as três últimas faturas sem depender de vencimento, `MEIO_DESATIVADO` em todas as portas de escrita, `PUT /cfg/saldos` e `POST /cartoes` recusam meio desativado, `ativoAte: "hoje"`, `encerrado` em `/saldos` e `/cartoes`, Remover só com lançamentos excluídos). As demos de **Listas, Histórico, Saldos e Cartão de crédito foram aprovadas** pelo dono (ações apagadas com o motivo) e as telas estão implementadas (`web/meio-desativado.test.js`). A demo e o código do **Lançar editando** também foram aprovados e feitos. Prever, Repetir, Estender, Tendências, exportar e a conferência com o extrato passam pelo servidor (que recusa com mensagem) ou só leem; o `cartaoDiferenca` tratava o 409 como sucesso e foi corrigido. O cache das faturas de Futuros **não precisa** acompanhar `CFG#LISTAS` (meio desativado não tem fatura a pagar). **Correção de custo (2026-11-08, branch `fix/desativar-sem-varrer-passado`):** a conferência lia o espaço inteiro duas vezes e deu 502 em dev; agora meio sem nenhum ponto de partida do saldo (mês conferido, cache ou `dataInicio`) não desativa, o saldo parte do último fechamento e previsto/futuro é só de hoje em diante (R13 do spec). **Falta:** ver as telas no ambiente dev (o app real exige o login do Cognito e não foi aberto).
- 59. [ ] <a id="item-59"></a>**[Depois] Congelar meio (continua existindo, mas não aceita eventos novos; ideia do dono, 2026-11-08; só registrado, nada decidido nem feito):**
  Surgiu na análise do item 58. O 58 trata do **Encerrar** (a conta acabou: só com saldo zero e sem nenhum evento, inclusive futuro; o meio some de Saldos, Cartão de crédito e Futuros e fica só no
  Histórico, com marca). O **Congelar** é outro conceito: o meio **continua existindo e aparecendo em Saldos e Cartão de crédito, com a marca**, mas **não aceita eventos novos** (Lançar, Transferir,
  Avisos novos, Recorrência). Serve para o meio com saldo que não dá para zerar (o `SODEXO-REF-IBM` ao sair da empresa, o exemplo do item 43) e funciona como rampa para o Encerrar:
  congela, deixa o que existe se resolver, encerra. **A definir antes de qualquer código:** o nome na tela ("Congelar" ou "Pausar"; não usar "desativar" nem "bloquear" para não confundir
  com o Encerrar); o que já existe num meio congelado (confirmar previsto, pagar fatura, conferir mês: a sugestão é poder liquidar o que existe e recusar só o novo); recorrência ativa
  (sugestão: recusar congelar enquanto houver); avisos já existentes continuam. **Dado novo no DynamoDB:** precisa de um segundo estado por meio (hoje só há o `ativoAte` em `CFG#LISTAS`);
  só entra com a aprovação do dono. Importar no meio congelado segue a mesma regra do desativado (item 58): só registros anteriores à `dataInicio`. **Demo antes** da tela. Depende de o item 58 estar decidido.
- 60. [x] <a id="item-60"></a>**[Agora] Saldos: meio novo nasce com início do controle e meio sem início não é calculado (concluído em 2026-11-09: PR 392, testado e aprovado pelo dono em dev; decidido pelo dono em 2026-11-09; vem **depois** do item 58, separado de propósito: são duas coisas ao mesmo tempo e não dá para fazer juntas):**
  **Origem (provada no log de dev, 2026-11-09):** com 12 meios, o `GET /saldos` levava 15 a 18 s (13 páginas, 12.770 itens) porque dois meios (`Teste` e `T-LIVRE`) não tinham `dataInicio` nem fechamento e o servidor
  somava a história inteira por eles; os outros 10 já liam só do mês seguinte ao último fechamento (o fechamento automático funciona). Com `dataInicio` nos dois, caiu para 0,8 s (1 página, 866 itens). A tabela fica em 5 RCU,
  sem índice (`feedback_performance-consultas`). O `PUT cfg/listas` só acrescenta o nome em `CFG#LISTAS`; nada é criado em `CFG#SALDOS`, então **todo meio novo nasce sem início e reabre esse custo**.
  **Decidido (as duas abordagens valem):**
  1. **Meio novo nasce com `dataInicio` = a data de hoje** (a data de criação; saldo inicial 0), **no servidor**, quando o `PUT cfg/listas` recebe um nome que ainda não existia (a IA do Lançar e a importação da planilha também salvam a lista). Usa a `porBanco` de
     `CFG#SALDOS` que já existe: nenhum campo novo. **Avisar o usuário** de alguma forma (a definir na demo): "X começa a contar saldo hoje; mude em Saldos". Efeito a dizer: lançamento de data anterior à `dataInicio` não conta no saldo
     (vira histórico), inclusive o histórico importado de um meio novo, até a data ser ajustada. A variação "1º dia do mês" foi discutida e fica como alternativa.
  2. **Meio sem `dataInicio` (e sem fechamento) não é calculado** pelo `/saldos`: o servidor não lê lançamento algum por ele e a tela **avisa o motivo** ("sem a data de início o saldo exigiria somar todo o passado; defina a data de início"). Cobre os
     meios antigos sem início. Muda o item 40 (hoje, sem configuração, o saldo parte do primeiro movimento): os testes de saldo sem início mudam, e o total precisa dizer que não inclui o meio sem início.
  **Rejeitado:** assumir que a data vazia é "hoje" a cada leitura (a data mudaria todo dia e o saldo nunca seria estável).
  **Antes de qualquer código:** demo das duas telas (aviso ao criar o meio em Listas; Saldos com o meio não calculado, o motivo e o total), partindo de Listas v6 e Saldos v3, com o que cada uma faz perder. A decisão de escopo do desativar
  (item 58) já está resolvida e mergeada; este item não a reabre.
  **Fechado com o dono em 2026-11-09 (demos [`Listas v7`](.ai/demos/demo-listas-v7-meio-novo.html) e [`Saldos v4`](.ai/demos/demo-saldos-v4-sem-inicio.html) aprovadas):** o início do meio novo é **hoje**; o aviso é o texto da demo (cartão em Listas, aviso rápido na IA do Lançar, resumo no fim da importação);
  o total diz "N sem início do controle (fora do total)"; a importação de planilha também cria o meio com início hoje (o histórico importado só entra no saldo depois de ajustar a data).
  **Implementado (2026-11-09, PR 392, aprovado pelo dono em dev):** `PUT cfg/listas` cria `porBanco[nome] = {saldoInicial: 0, dataInicio: hoje}` para o nome novo e devolve `meiosNovos`; o `/saldos` devolve `naoCalculado` e `motivo` para o meio sem início (nenhum lançamento lido, sem cache, fora do total, `total.naoCalculados`);
  a conferência recusa com 409 `SEM_INICIO`. Testes: `backend/meio_novo_inicio_test.go`, `saldo_servidor_test.go` (S-T13 e correlatos), `web/meio-sem-inicio.test.js`.
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
  regravado. **Confirmar** (demo aprovada em 2026-10-24, `.ai/demos/arquivo/demo-confirmar-previsto.html`): janela com Valor,
  Data e Status, cada um já preenchido e alterável, textos "Confirmar", "Já paguei" e "Já recebi", botão verde na
  tabela, aviso de 6 s com Desfazer (restauração no servidor), data sugerida hoje em mês fechado; ao confirmar um
  antigo, a janela abre com o primeiro status da lista do servidor (hoje `PAGO`). Sai do front o que olha status ou soma: `ehFuturo`, `afetaSaldo`, `resumoFuturos`,
  `gruposFuturos`, a soma do Histórico, a inclusão automática de `CREDITANDO` em Listas e o "Status das cópias" do Prever
  (a decidir no spec). Fora daqui: o desmembramento do `PAGO` (PIX, DEBITO, DINHEIRO e TED, item 56), o `CONTAS` (**fica como está**, decidido em 2026-11-02: não é selecionável e só o servidor o grava) e o `excluirDoTotal`, e o papel de
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
  `GET /cfg/listas`, validação de `faturaAjuste`, `PUT /cfg/cartoes` removida), com a aba Cartão de crédito **conferida lado a lado**; a
  **fatia b** (o cartão como entidade própria, a trava da fatura conferida e a migração do `faturaAjuste` explícito) segue a discutir.
  Intent, plan e spec da fatia a (aprovados em 2026-10-24) em [`.ai/specs/fatura-servidor/`](.ai/specs/fatura-servidor/intent.md).
  **Fatia a concluída em 2026-10-24** (fases A, B e C: a aba Cartão de crédito e a linha Fatura do Lançar leem o servidor, conferidas pelo dono em dev e produção; o `PUT /cfg/cartoes` saiu). Sobra a fatia b.
  - [x] **Fatia a: faturas do cartão no servidor** (concluída em 2026-10-24: fases A, B e C).
  - [x] **Fatia b, com o escopo reduzido (2026-10-25; concluída em 2026-10-25, junto com o item 35):** só a **trava da fatura conferida pelo servidor**, feita junto com o item 35. **Fora, por decisão do dono:** o cartão como entidade própria (ele continua sendo uma entrada de `CFG#CARTOES`, `meios[].cartao`) e a migração do `faturaAjuste` explícito (o passado está correto). Intent, plan e spec (rascunho) em [`.ai/specs/pagar-fatura/`](.ai/specs/pagar-fatura/intent.md).
- 34. [ ] <a id="item-34"></a>**[Próximo] Avisos por despachante e vários canais:** o service só sabe o que avisar e usa um `Avisador`; um
  despachante lê os canais que a pessoa assinou (Telegram, e-mail, WhatsApp), formata, divide e entrega, e uma falha de
  canal não derruba as outras. Cada canal é uma strategy; canal novo é um adaptador novo. Resolve na origem o limite de
  4096 caracteres do item 28. **Ponto de partida (item 32, fatia 4):** o `Avisador` já existe com o nome `service.Notificador`
  (`Enviar(destino, texto)`), o canal Telegram é o primeiro arquivo de `adapter/notificacao/`, e a rotina diária está em
  `service.Avisos`; falta o despachante por assinatura e os outros canais.
  **Roteiro em [`.ai/specs/avisos-despachante/`](.ai/specs/avisos-despachante/intent.md) (rascunho de 2026-11-13).** Decidido em 2026-11-13: este item vem **inteiro antes** do push (fase 4 do [item 54](#item-54)); "inteiro" = despachante, divisão da mensagem longa do
  Telegram e isolamento de falha, com o Telegram como único canal real (e-mail e WhatsApp sem código agora); quem assina o quê é deduzido de quem tem destino; nenhum campo novo no DynamoDB.
  **Código pronto no PR da branch `feature/avisos-despachante` (2026-11-13); falta o item ser marcado depois de conferido no `dev` (D-T14 do spec).**
- 43. [x] <a id="item-43"></a>**Meios de pagamento: o dono cria, desativa e remove (concluído em 2026-11-03: fase A no #347 e fase B no #348; conferido em dev pelo dono; rascunho de 2026-10-24):** o que o dono chama de **meio de pagamento** (conta, carteira, vale, cartão:
  `SODEXO-REF-IBM`, `DINHEIRO`, `CARTEIRA`...) é o `banco` gravado, com um **de-para só de exibição em `domain/`** (o Dynamo não muda, o campo continua `banco`).
  **O dono cria e desativa meios pelo app** (dado do espaço, não do produto); o front nunca conhece os nomes. Um meio desativado (por exemplo, o `SODEXO` quando o dono sair da
  IBM) **continua nos lançamentos e no saldo do passado** e deixa de ser sugerido a partir de uma data; "ver todos" mostra também os inativos, porque o passado é do
  usuário. `CRIPTO` e `DREX` são **formas** (status), não meios. Pede **um atributo novo no Dynamo** (a data em que o meio deixa de valer, `ativoAte` (AAAA-MM-DD) em
  `CFG#SALDOS.porBanco[meio]`, ausente = ativo; **aprovado pelo dono em 2026-10-24**) e a lista `meios: [{nome, cartao, ativo}]` que o item 33a já entrega com `ativo` sempre verdadeiro.
  Intent, plan e spec depois da conversa sobre o 40, o 33a e o 41. Demo antes de qualquer tela.
  **Fases A (backend, #347) e B (tela, #348) concluídas em 2026-11-03 e conferidas em dev pelo dono. Roteiro (aprovado em 2026-11-03):** [`.ai/specs/meios-de-pagamento/`](.ai/specs/meios-de-pagamento/intent.md), em duas fases (A backend, B tela); o `ativoAte` em `CFG#LISTAS` foi reaprovado pelo dono.
  **Conversa de 2026-11-03:** o `ativoAte` vai em `CFG#LISTAS` (onde o meio nasce), não em `CFG#SALDOS` (o cartão sem conta não tem saldo); a etiqueta **crédito** vem da aba Cartão de crédito
  (Listas só exibe); **Remover** só confere o uso ao clicar, por varredura (a flag `jaUsado` fica em Ideias, para se a varredura ficar cara). Demo: [`demo-listas-v3-meios`](.ai/demos/demo-listas-v3-meios.html) (aprovada em 2026-11-03).
- 46. [x] <a id="item-46"></a>**Aviso no Telegram para a fatura do cartão** (concluído em 2026-10-26: fase A do backend no #244 e fase B da tela no #246; conferido em dev pelo dono): hoje só um lançamento tem aviso; a fatura, que é calculada pelo servidor, só tem o botão Já paguei. O aviso passa a ser ligado **na própria fatura** em Futuros (botão Avisar, mesma tela do Prever), só para aquela fatura (o cartão não tem aviso padrão) e só se ela tiver vencimento. Dado novo: `faturas[ym].aviso` (mesmo formato do lançamento) mais o item auxiliar já existente em `AVISOS#ATIVOS`; **desempenho: dias sem aviso não leem nada além da Query dos avisos**. A fatura paga some do resumo sozinha. Intent, plan e spec (rascunho) em [`.ai/specs/aviso-fatura/`](.ai/specs/aviso-fatura/intent.md); demo `.ai/demos/arquivo/demo-aviso-fatura.html`.
- 45. [x] <a id="item-45"></a>**Futuros: os atrasados antigos são buscados por ano, sob demanda** (concluído em 2026-10-26, em produção; demo `.ai/demos/arquivo/demo-atrasados-por-ano.html`; `GET /futuros` agrupa em `atrasados[{ano,n,valor,meses}]` e devolve `anosAnteriores`; `?ano=AAAA` busca um ano; anotado em 2026-10-25): hoje Futuros olha só **do mês passado em diante**
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
  - **Três níveis de visibilidade por banco ou cartão** (conversa de 2026-10-31; cada dono escolhe a política dele, e
    um mesmo espaço pode misturar níveis):
    | Nível | Saldos e Cartão de crédito | Lançamentos (Histórico, Futuros, Tendências) | Totais e somas |
    |---|---|---|---|
    | **Visível** | aparece | aparecem | incluem |
    | **Resumido** | não aparece | aparecem | incluem, com aviso "inclui bancos que você não visualiza" |
    | **Oculto** | não aparece | não aparecem | **não** incluem (como o "ocultar" do Histórico) |
    - Mais uma opção independente por meio: **pode lançar neste meio**. No nível Oculto fica sempre desligada.
    - Caso real que originou o Resumido: a esposa só tem o cartão Santander e os cartões Sodexo (alimentação e
      refeição); não faz sentido mostrar os outros cartões, mas ela precisa ver os totais para ter consciência dos
      gastos. No futuro o dono pode dar a ela outro cartão e passar saldo de outros bancos para ele (basta mudar o
      meio para Visível e pode lançar). Outro dono pode não querer mostrar nada ao membro: usa o Oculto.
  - **Coerência dos números:** todo número mostrado é a soma das linhas mostradas, com **uma única exceção declarada**,
    o Resumido, onde o total tem o aviso. O servidor filtra na origem e manda pronta a flag do aviso; o front só
    exibe (nada decidido por nome de banco). O Oculto se comporta como o "ocultar" do Histórico: nada dele entra em
    soma nenhuma na tela dela (saldos, faturas, Histórico, Futuros, Tendências, totais, filtros por tag).
    Diferença importante: o `oculto` do lançamento vale para todo mundo e tira o valor do saldo; a visibilidade por
    pessoa **não muda o saldo real**, só o que cada uma enxerga (o dono continua vendo tudo).
  - **Limite do Resumido:** com o total completo, quem tem os outros saldos consegue deduzir o do banco escondido
    por subtração. Só o Oculto esconde o valor de verdade; isso deve constar na documentação.
  A decidir (em reflexão do dono): modelo (permissão por usuário ou perfis nomeados); onde guardar (proposta: campo
  `permissoes` no item `USER#<sub> / SPACE#<sid>`, que já existe com `role`; é campo novo no DynamoDB e só entra com
  aprovação); padrão de meio novo (proposta: Oculto, com o dono podendo trocar por pessoa); transferência entre um
  meio visível e um Oculto (proposta: mostrar só a ponta visível); pagamento de fatura de cartão Oculto feito por banco
  visível (proposta: mostrar a linha normal, sem ligar à fatura); se o member pode lançar em meio que não enxerga
  (proposta: não); e as capacidades gerais por pessoa (transferência no Lançar, valor estimado). Aplicar no **backend**, não só esconder no front: a API não devolve lançamentos, saldos (itens
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
  - **Meio pagador:** escolhido na seção do cartão, na aba Cartão de crédito. Vale para o cartão inteiro como padrão e **pode ser trocado em cada fatura**,
    porque ninguém sabe onde haverá saldo (cartão do SANTANDER pago pelo NUBANK, em débito).
  - **Vencimento:** a data **de cada fatura** (o dia do cartão só sugere, não vale para nenhuma fatura; decisão do dono, 2026-10-25). É a data em que a fatura aparece em Futuros; sem ela, a fatura não aparece.
  - **Em Futuros**, no meio pagador, até duas linhas por cartão, **calculadas pelo servidor** (não são previstos gravados):
    **fatura fechada** (valor **congelado** no fechamento: só mudam o valor da operadora, o do boleto e ajuste ou estorno explícito;
    passado o vencimento sem pagar, fica como **atrasada**) e **fatura aberta** (total vivo, muda a cada compra no crédito). Ao fechar,
    no dia seguinte ao fechamento, a aberta vira fechada e a seguinte ganha a sua linha. Compra com data anterior lançada depois do
    fechamento segue a regra de hoje: o `faturaAjuste` de ±1 a leva para a fatura vizinha; o valor congelado só muda por ajuste explícito.
  - **Confirmar o pagamento** reaproveita o gesto de confirmar do Futuros (trocar meio, valor e status; demo
    `.ai/demos/arquivo/demo-confirmar-previsto.html`). Grava o lançamento no meio pagador, o liga à fatura, e a linha some.
  - O pagamento mexe no saldo do meio pagador e **não conta de novo como gasto**: o gasto já contou nos registros de `CREDITO`, que são
    reais, não afetam o saldo e não se pode perder o controle deles. Caso real: estorno no dia do corte, fatura de um valor e pagamento de outro.
  - **Pagamento mínimo ou parcial** (a maioria dos usuários usa; o dono não) fica aqui: a fatura passa a ter mais de um pagamento.
  - **Dados novos aprovados pelo dono em 2026-10-25:** `pagador` e `diaVencimento` no cartão e `pagador` e `vencimento` por fatura (tudo dentro da entrada do
    cartão em `CFG#CARTOES`). O atributo `pagaFatura` no lançamento aguarda a aprovação no spec. **O pagamento é um lançamento normal e a marca `pagaFatura` é só uma label** (sem limite de pagamentos, de valor ou de meio; pagamento mínimo e restante são da operadora, nada passa adiante). Depende dos itens 33a e 41,
    concluídos. Intent, plan e spec (aprovados em 2026-10-25; **concluído em 2026-10-25: fases A (backend, #228) e B (telas, #230), painel do cartão refeito em linhas compactas (#231) e o débito do mesmo banco pode pagar a fatura (#233); conferido em dev pelo dono**) em [`.ai/specs/pagar-fatura/`](.ai/specs/pagar-fatura/intent.md). Detalhes em `.ai/arquitetura.md`.
- 54. [ ] <a id="item-54"></a>**[Próximo] App Android com Capacitor (2026-11-01; rascunho para aprovação, roteiro em [`.ai/specs/app-android/`](.ai/specs/app-android/intent.md)):** hoje o app é uma página (`web/`) usada
  pelo navegador ou como PWA. O dono quer um app Android **fora do navegador**, com **capacidades nativas**, primeiro para duas pessoas e depois pela **Play Store**. Decidido: **Capacitor** com o
  `web/` empacotado dentro do app (sem reescrever telas, sem mudar regra nem backend); TWA e app nativo (Kotlin ou Flutter) foram descartados; `dev` e `prod` coexistem por `applicationId` distinto;
  **APK de debug** no desenvolvimento e **AAB assinado** na Play. Fases (cada uma um PR):
  - [x] **Fase 1, a casca** (2026-11-11, PR 415; o APK do `dev` compila, instala e abre no aparelho; sem login o app não mostra tela, então os testes A-T6 e A-T7 do `spec.md` passaram para o fim da fase 2): projeto em `app-android/` (`applicationId` `br.com.cicerotec.financas`, sabores `dev` e `prod`), `scripts/app-android.ps1`, CORS com a origem `https://localhost` da WebView (`AppOrigin`) e botão voltar físico (`web/js/nativo.js`).
  - [x] **Fase 2, login nativo e token seguro** (2026-11-12, PRs 419 e 420; login, refresh, Sair, a volta com o app morto, o botão voltar físico e o modo avião vistos no aparelho no `dev`, ver "Fase 2: o que foi verificado" no `spec.md`; o `prod` ainda não foi compilado nem instalado): Hosted UI no navegador do sistema (Custom Tabs), só no toque em "Entrar" (tela "sem sessão" da Abertura, demo v3), PKCE e retorno por esquema próprio, refresh token no Keystore (plugin `Cofre`, em Java), login extraído de `web/api.js` para `web/js/sessao.js`.
  - [x] **Fase 3a, trava por biometria** (2026-11-12, branch `feature/app-android-biometria`, demo `demo-bloqueio-v1` aprovada; conferida pelo dono no aparelho; ficam por ver o aparelho sem biometria e a medição de layout, ver o `spec.md`): duas chaves em Listas, Aparência, Segurança ("Bloquear com biometria" e "Esconder nos apps recentes"), guardadas só no aparelho; tela de bloqueio ao abrir e depois de 1 minuto fora; plugin `Trava` em Java.
  - [x] **Fase 3b, splash nativo e ícone adaptativo** (2026-11-12, branch `feature/app-android-icone-splash`, demo `demo-abertura-v4-splash-nativo` aprovada; medido no aparelho): o splash do Android mostra o ícone sobre o fundo do app e espera a página; a Abertura usa o ícone de 192 px no app; ícone adaptativo nos dois sabores. A barra de status já acompanhava o tema. Não visto: celular em tema claro e o sabor `prod`.
  - [ ] **Fase 4, push (FCM) como canal do despachante** (spec própria; **depende do [item 34](#item-34)**, que vem antes). Decidido em 2026-11-13: o token fica no campo `pushTokens` do vínculo da pessoa no espaço (como o `telegramChatId`, **campo novo no DynamoDB, aprovado**), a notificação mostra só a contagem, o plugin é nosso, em Java.
  - **As fases 5 (release e Play Store), 6 (produção aberta) e 7 (uso sem internet) passaram para o [item 64](#item-64)** (decisão do dono em 2026-11-13): este item se conclui com a fase 4.
  **Já decidido (no `intent.md` e no `spec.md`):** `applicationId` `br.com.cicerotec.financas`, retorno do login por esquema próprio (o App Link fica para o item 64), biometria opcional, plugins sem
  bundler. **A decidir antes da fase 4:** onde guardar o token de push de cada aparelho (dado novo no DynamoDB). **Fora:** iOS e as mudanças de regra de negócio.
- 64. [ ] <a id="item-64"></a>**[Depois] App Android: Play Store, produção aberta e uso sem internet (2026-11-13; continuação do [item 54](#item-54)):** o item 54 entrega o app Android para uso
  pela família (casca, login nativo, trava e, por fim, push) e se conclui na fase 4. O que vem depois é publicar e abrir o app, e ficou aqui por decisão do dono em 2026-11-13, para o item 54 não
  ficar aberto à espera de coisas que dependem de outros itens. As fases mantêm os números do roteiro ([`.ai/specs/app-android/plan.md`](.ai/specs/app-android/plan.md)); cada uma ganha spec própria.
  - [ ] **Fase 5, release e Play Store:** AAB assinado (keystore de upload fora do git, Play App Signing), `versionCode` e `versionName` ligados às tags, teste interno para as duas pessoas, política
    de privacidade, formulário de Segurança dos Dados e as demais exigências da loja (conferir no Play Console na hora). Inclui o **App Link por `https`** no retorno do login, que troca o esquema
    próprio da fase 2 (decisão de 2026-11-11), e compilar e instalar o sabor `prod`, que ainda não foi feito. **A decidir antes:** conta da Play (pessoal ou de organização) e a distribuição inicial
    (APK assinado direto entre as duas pessoas ou esperar o teste interno).
  - [ ] **Fase 6, produção aberta:** só depois das fases 1 a 5 do [item 49](#item-49) (MFA, limite de taxa, cadastro, espaço próprio, exportar e excluir conta) e da política de privacidade.
  - [ ] **Fase 7, uso sem internet** (futuro; segue [`.ai/offline-e-sincronizacao.md`](.ai/offline-e-sincronizacao.md)): demo e spec próprias; depende do gatilho "muitos usuários querendo offline".
  **Herdado do item 54, por ver:** `dev` e `prod` instalados juntos (A-T5), o celular em tema claro no splash, e desinstalar para ver o Cofre sumir (A-T16b).
- 55. [x] <a id="item-55"></a>**Novas visões dos gastos na Tendências (2026-11-01; concluído em 2026-11-02, fases A a F, roteiro em [`.ai/specs/tendencias-visoes/`](.ai/specs/tendencias-visoes/intent.md)):** a consulta continua
  sendo o filtro e as visões desenham a mesma consulta de outros ângulos, em **seções que recolhem pelo título** abaixo do resultado (Comparar, Ao longo do mês, Para onde foi), com o mês de referência (‹ ›) e o "Comparar com" no topo. Tudo calculado no navegador com os 1000 lançamentos da tela (simulação, nada vai ao servidor),
  menos o limite das Metas. O dono escolheu as visões depois de ver onze gráficos de exemplo. A primeira demo (seletor "Ver como") foi reprovada em 2026-11-01; as demos [`v2`](.ai/demos/demo-tendencias-v2.html) e [`v3`](.ai/demos/demo-tendencias-v3.html) foram **aprovadas em 2026-11-02**. Fases (cada uma um PR):
  - [x] **A, Calendário de calor e Maiores tags** (concluída em 2026-11-02; só front): seções "Ao longo do mês" e "Para onde foi", recolhíveis (a escolha fica só no aparelho), mês mais recente da consulta, só em Gastos e Receitas.
    Maiores tags comparava só com o mês anterior; o "Comparar com" e a navegação entre meses vieram na fase F.
  - [x] **B, Ritmo do mês** (concluída em 2026-11-02; só front): acumulado do mês contra o mês anterior inteiro e projeção pelo ritmo dos dias passados (a partir do 3º dia, só no mês atual); não lê Futuros. Fica na seção "Ao longo do mês", antes do calendário.
  - [x] **C, Fixos e variáveis** (concluída em 2026-11-02; só front): seis meses até o mais recente da consulta, fixo = lançamento marcado como recorrente (item 51), o resto é variável; nada migra nem é inferido. Fica numa seção "Comparar", no topo.
  - [x] **D, Forma de pagamento** (concluída em 2026-11-02; só front, o item 41 já está pronto): no mês de referência, barra repartida e uma linha por status, do maior ao menor; o nome é o texto do status e a cor vem do app (que a recebe do servidor). Na seção "Para onde foi", antes das tags.
  - [x] **E, Metas por consulta** (concluída em 2026-11-02): campo `limite` no item da consulta (aprovado pelo dono: número em reais de 0 a 1 bilhão, até 2 casas, opcional, dentro do item; quem altera a consulta altera o limite). Backend: validação só do formato, `null` tira, trocar a consulta não apaga. Tela: card "Metas por consulta" (dentro abaixo de 85%, perto de 85% a 100%, passou acima de 100%) e janela do limite.
  - [x] **F, comparar consultas e navegar** (concluída em 2026-11-02, demo [`v3`](.ai/demos/demo-tendencias-v3.html) aprovada): "Este mês contra o anterior", "Seis meses, por consulta", **Comparar com** (média de 3 meses e mesmo mês do ano passado) e **navegar entre meses**. Leem todas as consultas salvas uma vez, quando uma seção que precisa delas está aberta (guardrail e `carga.test.js` ajustados). Desvio da demo: a pilha de seis meses mostra as cinco consultas que mais gastaram e "Outros" (o que nenhuma pega), e cada lançamento entra na primeira consulta, para a pilha somar o total.
  - Previstos ficam fora das visões (decidido em 2026-11-02): repetições, parcelas e recorrências já viram lançamento real no dia. Só o previsto pontual (ex.: uma excursão) ficaria de fora; se o dono quiser vê-lo, seria uma janela aberta sob demanda, nunca carregada com a tela.
- 56. [x] <a id="item-56"></a>**Formas de pagamento PIX, DEBITO, DINHEIRO e TED (2026-11-02; aprovado e implementado no backend em 2026-11-02, roteiro em [`.ai/specs/formas-de-pagamento/`](.ai/specs/formas-de-pagamento/intent.md)):** é o desmembramento do `PAGO` que o item 41 deixou para depois. Decidido com o dono: entram **quatro** entradas selecionáveis no `Catalogo` do backend, contando no saldo como o `PAGO`, em **ordem de uso** (PIX, DEBITO, CREDITO, DINHEIRO, TED, PAGO); o `PAGO` continua selecionável **no fim da lista** (o padrão passa a ser o PIX; tirá-lo é uma linha, quando o dono quiser); **todas valem para entrada e saída, sem validação de tipo** (estorno no débito e no cartão são entradas; o salário do dono entra por TED, com a tag `salário`); **nada no DynamoDB muda**, **nenhuma tela muda** (sem demo) e o passado é do usuário. **Ficaram de fora de propósito:** BOLETO (é o documento, não o meio: paga-se com PIX, DINHEIRO ou CREDITO) e RECEBIMENTO (o tipo `entrada` já diz que entrou). **Não falta mais nada nesta frente:** DREX, CRIPTO e o que vier são linhas novas do catálogo em versões futuras, conforme o app crescer. Uma fase só (backend), feita: catálogo, ordem de uso, `domain.StatusGenerico` para o ajuste da conferência e testes; a base do cache de saldos muda uma vez (recalculada sozinha). **Conferido pelo dono no dev em 2026-11-02 e levado à produção.** Junto veio a **paleta de 16 cores** do seletor de cor da forma de pagamento (demo [`demo-listas-v2-novas-cores`](.ai/demos/demo-listas-v2-novas-cores.html), `web/paleta.test.js` exige que toda cor padrão de status esteja nela). **Fora desta entrega, para quando o dono quiser:** tirar o `PAGO` da lista (uma linha, `Selecionavel: false`, mais trocar o status de exemplo em 21 arquivos de teste) e mover o catálogo para um JSON embutido no backend (o `PAGO`, o `CONTAS` e os nomes antigos continuam no código).
- 20. [ ] <a id="item-20"></a>**[Depois] Ideias** (seção no fim deste arquivo).

### UX/UI (usabilidade do que já existe)
- 30. [x] <a id="item-30"></a>**Formulário de Lançar no padrão de telas** (concluído; confirmado pelo dono em 2026-10-26): hoje o Lançar ainda mostra à vista os controles antigos de
  aviso e de repetição (17 controles, pouco usáveis no celular). Passar "Repetir" e "Avisar" para linhas-resumo que
  abrem as telas cheias do Prever (a lógica e os componentes já existem em `web/index.html`, bloco "Prever e Estender").
  Pelo padrão, começa com uma demo do formulário completo para aprovação.
- 21. [x] <a id="item-21"></a>**Próximos passos de UX** (encerrado em 2026-10-26: o dono considera bom o uso no celular como está; o que sobrou não será feito) (vindos de observar o uso real; detalhes em [`.ai/ux.md`](.ai/ux.md)): barra de navegação
  inferior no celular, botões pequenos de Cartão de crédito e Saldos, "Cancelar edição"/"Limpar formulário" perto do Salvar,
  acessibilidade das abas, unificar o seletor de tags (agora no item 42).
- 48. [x] <a id="item-48"></a>**Aviso padrão do cartão, copiado para a fatura ao definir o vencimento** (concluído em 2026-10-26 no #249; conferido em dev pelo dono): o cartão ganha `porBanco[meio].aviso` (mesmo formato do aviso da fatura, só um modelo). Quando o dono define o **primeiro vencimento** de uma fatura (o momento em que ela vira registro em Futuros) e a fatura ainda não tem aviso, o servidor **copia** o padrão para `faturas[ym].aviso` na mesma gravação; depois o dono muda pelo botão Avisar. Mudar o padrão não altera faturas já copiadas; nenhuma leitura grava; o passado não é tocado. Dado novo: `porBanco[meio].aviso` (aprovado pelo dono em 2026-10-26). Em Configurar cartão, linha **Aviso (padrão)** que abre a mesma tela de aviso (demo antes). Extensão do item 46, em [`.ai/specs/aviso-fatura/`](.ai/specs/aviso-fatura/intent.md).
- 63. [ ] <a id="item-63"></a>**[Próximo] Avisar a falta de rede em toda tela (2026-11-12; achado pelo dono no app Android, em modo avião, ao fechar a fase 2 do item 54):** sem internet, cada tela
  reage de um jeito, e três delas não dizem nada. Não é defeito do app Android: o `web/js/app.js` não mudou na fase 2 e o navegador faz igual; o app só torna o caso comum, porque abre sem rede.
  O que o dono viu, com o app aberto e logado: **Histórico** e **Histórico de versões** mostram **"Failed to fetch"** (o texto cru do erro do `fetch`, em inglês, vai para a tela:
  `"Não consegui buscar esse período: " + err.message` e semelhantes); **Saldos**, **Cartão de crédito** e **Futuros** **não carregam e não avisam** (esperou 1 minuto); **Listas** (excluir
  tag) e **Lançar** (salvar) já têm mensagem própria e servem de modelo ("Não foi possível salvar. Confira a conexão e tente de novo."). O que o código faz hoje, lido em 2026-11-12:
  `carregarFuturos` só faz `console.error` e marca `futurosSujo`; `carregarSaldos` marca `fechFalhou`, mas a frase só aparece dentro do bloco "Fechamento mensal" de um meio aberto;
  `carregarCartoes` guarda `cartaoErro`, mas `renderCartao` só o mostra quando ainda não há resultado anterior. **Antes de corrigir, reproduzir** tela a tela (modo avião no celular e rede
  cortada no navegador), anotando o que aparece com e sem dado já lido, e conferir as telas que o dono não abriu (Tendências, Configurações, as janelas de Prever e de Confirmar).
  **Proposta:** uma função só que traduz o erro de rede para uma frase em português ("Sem conexão com a internet. Confira e tente de novo.") e um jeito único de a tela dizer que não
  conseguiu ler, com "Tentar de novo", sem apagar o que já estava na tela. **É mudança de tela: demo aprovada antes do código** (celular primeiro, dois temas, o que se perde). Fora do
  escopo: usar o app sem internet (fase 7 do item 54, `.ai/offline-e-sincronizacao.md`) e a Abertura, que já trata a falha. Cada tela ganha um teste que falha sem a correção.
- 47. [ ] <a id="item-47"></a>**[Próximo] Tela "Avisar no Telegram" como componente reutilizável (2026-10-26):** hoje a tela de aviso são funções soltas do Prever (`pvTelaAvisar`, `pvBalao`, o estado `pv`), amarradas a um lançamento. O dono quer que ela seja um **padrão**, reformulado para ser reaproveitado pelas buscas do Histórico, das Tendências e do Futuros (item 42) e pelo aviso da fatura (item 46). Pede um componente com entradas claras (título, vencimento de referência, textos, balão, o que o Salvar faz) e uma demo própria antes de qualquer tela, atualizando `.ai/padrao-de-telas.md`. O item 46 usa a tela como está, só parametrizada, para não esperar este item; este item depois troca os dois pelos mesmo componente. Junto do item 42.
  **Revisto em 2026-11-03 (vale este parágrafo, não o seguinte):** o roteiro foi refeito em [`.ai/specs/repetir-unificado/`](.ai/specs/repetir-unificado/intent.md) com um **inventário** de todos os pontos de entrada. Duas trilhas separadas: **A**, tirar a duplicação sem mudar nada que se vê (testada por igualdade de HTML e de chamadas, com um teste que impede a volta da cópia); **B**, redesenhar, só depois das decisões do dono e de uma demo que cubra o inventário. As demos de 2026-11-03 são exploração e **não valem como padrão**. Nada implementado.
  **Primeira leitura de 2026-11-03 (superada):** a duplicação é de **telas**, não de aviso em busca. O `pvTelaAvisar` serve a Prever, lote, Recorrência, fatura e aviso padrão do cartão (cinco modos no mesmo `pv` global); o Repetir tem três versões (`pvTelaRepetir`, `pvTelaRepetirLote`, `pvTelaRepetirRec`); o Lançar tem um segundo desenho do aviso (`htmlAviso`) e do repetir (`htmlRepeticao`), escondidos pela regra "data futura". Passa a ser **uma tela Repetir** com o seletor Só este · N vezes · Recorrente e o Avisar dentro, em casca única (`montarTc`) e seções com entradas explícitas. Demos de **exploração** de 2026-11-03, não aprovadas como padrão ([`demo-repetir-v2-seletor`](.ai/demos/demo-repetir-v2-seletor.html), [`demo-lancar-v3-repetir`](.ai/demos/demo-lancar-v3-repetir.html), [`demo-historico-v2-repetir`](.ai/demos/demo-historico-v2-repetir.html)); registro em [`.ai/padrao-de-telas.md`](.ai/padrao-de-telas.md) ("Repetir: exploração"). Fica de fora: o aviso da fatura e o do cartão, que seguem abrindo só a tela Avisar com o Salvar no rodapé. **Nada implementado ainda.**
- 52. [x] <a id="item-52"></a>**[Próximo] Demos versionadas por tela (2026-10-30; próxima ação, antes da tela da recorrência):** as demos em `.ai/demos/` estão confusas: cada uma nasceu de um assunto e mostra pedaços de telas, com a tela inteira repetida (e divergente) em várias. Plano: (1) **recuperar as telas como estão hoje no app** (`web/index.html`) e montar uma **demo-padrão de cada tela** (Lançar, Histórico, Futuros, Tendências, Configurações, menu do usuário, Repetir, Aviso etc.), cada uma completa e fiel ao app, com o nome `demo-<tela>-v1`; (2) **toda mudança nova parte da última versão da tela** e gera um arquivo novo `demo-<tela>-v<N>-<assunto>`, com o assunto acrescentado em uma palavra (ex.: `demo-historico-v1`, `demo-historico-v2-recorrencia`); (3) a versão aprovada vira a padrão da tela e o `.ai/padrao-de-telas.md` aponta para ela; (4) as demos antigas de assunto (`demo-*.html` atuais) são migradas para esse esquema ou arquivadas, sem apagar de uma vez. Primeiro uso: refazer a demo da recorrência (item 51, fase B) em cima de `demo-lancar-v1`, `demo-historico-v1` e `demo-futuros-v1`, em vez de montar telas soltas. **Decidido em 2026-11-02:** pasta plana com prefixo, antigas em `.ai/demos/arquivo/` (35 movidas, links atualizados), todas as telas. **Feito (aprovado pelo dono em 2026-11-02):** 25 capturas `demo-<tela>-v1` tiradas do app rodando com dados fictícios (telas, menus e janelas), a seção "Demos versionadas por tela" em `.ai/padrao-de-telas.md` e o resumo em `.ai/guardrails.md`. Concluído. O primeiro uso será a próxima tela que for mexida (por exemplo, as novas visões da Tendências, item 55), partindo da v1 dela.
- 53. [x] <a id="item-53"></a>**Consultas salvas da Tendências** (concluído em 2026-10-31, #301 e #303; conferido em dev pelo dono): salvar a consulta (grupos ou modo avançado, mais a visão) com um nome e usá-la de novo, **compartilhada no espaço**: todos veem e usam, cada pessoa altera e exclui só as suas e o dono modera. Demo aprovada em 2026-10-30 ([`demo-tendencias-consultas.html`](.ai/demos/arquivo/demo-tendencias-consultas.html)). Dado novo (aprovado em 2026-10-31): um item por consulta, `SPACE#<sid>` / `CONSULTA#<id>`, com `criadoPor` do login, exclusão lógica (`excluidoEm`) e `version`; validação só do formato no servidor; lê só quando a janela abre; nunca guarda número. Três ações novas em `authz.go` (`consulta.ler`, `consulta.criar`, `consulta.editar` com `SoDono`). Roteiro em [`.ai/specs/tendencias-consultas/`](.ai/specs/tendencias-consultas/intent.md) (aprovado em 2026-10-31). Fases: **A backend (#301)** e **B tela (#303)**, as duas conferidas em dev. Independe do item 42, mas a forma da consulta pode mudar na extração do filtro (42b).
- 57. [x] <a id="item-57"></a>**[Próximo] Combos sugeridos calculados pelo servidor (concluído em 2026-11-09: implementado em 2026-11-06, PR 366 do teclado mergeado, a rotina já rodou em produção (último cálculo 08/11/2026, 10 sugeridos em Listas) e o dono confirmou; roteiro em [`.ai/specs/combos-sugeridos/`](.ai/specs/combos-sugeridos/intent.md)):** hoje o navegador conta os combos sugeridos nos `lancamentosCache` (1000 lançamentos que só o Histórico e a Tendências carregam), então o Lançar não mostra sugerido ao abrir e, se lesse, seriam mil leituras por abertura. Proposta: uma rotina semanal do EventBridge (`{"acao":"combos"}`, separada da dos avisos) que conta os 1000 lançamentos mais recentes, **ligada pelo dono num interruptor em Listas > Combos (já ligado; cria o item auxiliar `combos` no `AVISOS#ATIVOS`, como a recorrência; demo antes)**, grava `sugeridos` (até 10, com `contagem`) e `geradoEm` dentro de `CFG#COMBOS`, sem chave nova e sem subir o `version`; o app só lê. **Campo novo no DynamoDB: só entra com a aprovação do dono.** Fora: contagem de uso por tag e sugestão pela descrição (dispensadas em 2026-11-06). Decisões D1 a D3 no `plan.md`.
- 62. [x] <a id="item-62"></a>**Abertura do app (splash)** (concluído em 2026-11-10, PRs 408 e 410; conferido no celular pelo dono): ao abrir, o Lançar aparecia na hora com Meio de pagamento, Forma de pagamento e Tags vazios e os botões sem ligar, por alguns segundos. Agora uma abertura (o ícone do app, "Finanças" e um indicador) cobre a tela até `listas`, `tags` e `combos` chegarem: fica no mínimo 0,6 s, avisa aos 4 s e, se o acesso ou uma leitura falhar, mostra o erro com "Tentar de novo" (e "Sair da conta" para conta sem espaço) no lugar do alerta do navegador. O ícone copia o do splash do Android (círculo de 224 px no centro da tela), porque esse splash é do sistema e o site só controla a imagem e a cor de fundo; em dev leva a faixa DEV. **A espera é a mesma**: a abertura só a cobre. Encurtar de verdade (ler a aparência junto com as listas, uma chamada só na subida, cópia local das listas no aparelho) e segurar o splash do próprio Android (só no app com Capacitor, item 54) **ficaram de fora**. Código: `web/js/abertura.js`, com `web/abertura.test.js`; demos `demo-abertura-v1` e `demo-abertura-v2-icone`; regras em [`.ai/padrao-de-telas.md`](.ai/padrao-de-telas.md), seção "Abertura (splash)". Na mesma análise saiu o bug do item 61.
- 42. [ ] <a id="item-42"></a>**[Próximo] Filtro único (tags, bancos, status e combos) e lista da consulta da Tendências** (guarda-chuva; raciocínio e regras em
  [`.ai/ux.md`](.ai/ux.md), seção "Filtro único e lista da consulta"). Hoje Tendências (filtro avançado de tags com E, OU e NÃO), Histórico (status, banco, tags e
  combos, só "todas" ou "qualquer uma") e Futuros têm cada um o seu filtro, e os números e as consultas divergem. A ideia é **um filtro compartilhado**: uma janela
  própria, aberta pela tela que a chama dizendo **quais blocos quer** (a Tendências só tags e combos; o Histórico tags, bancos e status), em duas partes, a lógica
  pura (testada) e a janela que só desenha. Cada tela é dona do seu estado, as opções vêm da tela que chama e nada é enviado ao servidor. Cada passo pede demo
  (janela nova ou tela mexida) e a janela nova entra no botão voltar (`camadasAbertas` e `fecharCamadaTopo`). **Nota de 2026-11-03:** a duplicação do aviso e do repetir (item 47) segue o mesmo método; o filtro de tags, combos e busca (copiado no Lançar, no Histórico, na Tendências e em Listas) ganha a mesma matriz de telas × diferenças antes da demo. **Nota de 2026-11-09 (candidato a base do componente):** o cartão Tags de Listas (#400, demo [`demo-listas-v9-tags`](.ai/demos/demo-listas-v9-tags.html), conferido em dev pelo dono) já nasceu como componente, `web/lista-tags.js` (`ListaTags.criar(el, {tags, aoExcluir, topo, pagina})`: estado próprio, sem id fixo, lógica pura testada em `web/lista-tags.test.js`), com busca, as criadas recentemente, "Ver todas as tags (N)" com ordem e "Mostrar mais". É **candidato a base da componentização das buscas de tags**: o que muda de uma tela para outra vira opção (o que o toque faz: marcar, excluir, escolher uma ou inserir; o adorno do chip; criar tag; a aba de Combos; "Marcadas primeiro"). Hoje só Listas o usa; a janela de Tags do Lançar (`renderTagPicker`), o filtro do Histórico (`criarSeletorFiltroTags`), a janela de tags da Tendências (`tendencias.js`) e a de migrar (`pedirNovaTagParaMigrar`) seguem com o código deles, **e há outros locais a levantar** (dono, 2026-11-09). Nada decidido: a matriz e a demo de cada tela continuam valendo antes de qualquer troca. Descrição em [`.ai/padrao-de-telas.md`](.ai/padrao-de-telas.md), "Tags (cartão de Listas)". Ordem:
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
  definir o **início do controle** de cada um (usando a aba Cartão de crédito).
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
- **Item 43, remover um meio: flag `jaUsado` no lugar da varredura (ideia do dono, conversa de 2026-11-03; guardada aqui para não se perder).**
  **Decisão de hoje:** no item 43, o botão **Remover** de um meio confere o uso **por varredura**, só ao clicar (nunca ao abrir a tela), parando no
  primeiro lançamento achado; o meio com lançamento é recusado e só pode ser desativado. **Se a varredura começar a custar caro** (leitura da tabela, 5 RCU,
  ou demora no Remover), troca-se pela abordagem abaixo, que o dono propôs e que fica pronta para ser retomada:
  - Todo lançamento tem um meio, então o uso pode ser **marcado na escrita**, sem busca nenhuma: `jaUsado` por meio (junto do `ativoAte`, num mapa
    `meios: { "<nome>": { ativoAte, jaUsado } }` em `CFG#LISTAS`; o `banco` continua `[]string`). Quando o servidor grava um lançamento num meio com
    `jaUsado: false`, muda para `true`, **uma vez só**, na mesma transação do lançamento. Remover passa a ser a leitura de um item.
  - **Três estados**, para não migrar o passado: ausente = "não sei" (uma única varredura, ao primeiro Remover, grava o resultado); `false` = nunca usado
    (como todo meio novo nasce); `true` = usado, **nunca volta a `false`** (o nome fica no histórico imutável, mesmo com o lançamento excluído).
  - **Cuidados:** todo caminho que grava `banco` marca a flag (criar, editar trocando o banco, importar em lote, restaurar, as duas pontas da transferência,
    recorrência, pagamento de fatura), com um teste por caminho: um esquecido deixa `false` num meio usado e o Remover apagaria um meio com histórico;
    marcar e remover são atômicos (Remover condicional a `jaUsado = false`, no mesmo item); a flag é do servidor, preservada como os campos de status
    (`CFG#LISTAS` é gravada inteira pelo front) e marcar não sobe o `version` da lista; custo: saber a flag em cada escrita (conferir se `CFG#LISTAS` já é lida ali).
  - Campo novo no DynamoDB: só entra com a aprovação do dono.
