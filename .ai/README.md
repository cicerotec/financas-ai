# .ai: contexto para quem constrói com IA

Tudo o que a IA precisa ler antes de agir fica aqui, separado do `BACKLOG.md`, que é só backlog (roadmap, o que já foi
feito e ideias).

| Documento | Para quê |
|---|---|
| [`guardrails.md`](guardrails.md) | Regras que valem para toda tarefa (arquitetura, telas, git, segurança, testes) |
| [`arquitetura.md`](arquitetura.md) | Camadas, regras no servidor, histórico imutável, domínio do cartão |
| [`security.md`](security.md) | Autenticação, papéis e permissões, dados, segredos, infraestrutura e pontos em aberto |
| [`privacidade-e-cifragem.md`](privacidade-e-cifragem.md) | Ideia para o futuro: cifrar os valores no DynamoDB (envelope encryption com KMS) para que nenhuma pessoa os veja ao consultar o banco; o service continua decifrando e calculando (nada decidido) |
| [`decisoes.md`](decisoes.md) | Decisões de produto fechadas e descartadas |
| [`padrao-de-telas.md`](padrao-de-telas.md) e [`demos/`](demos/) | Padrão de telas aprovado e as demos versionadas por tela (`demo-<tela>-v1`; as antigas, de assunto, em `demos/arquivo/`) |
| [`ux.md`](ux.md) | Observações de uso real e próximos passos de UX |
| [`eventos-futuros.md`](eventos-futuros.md) | Conceito e regras de Prever, Repetir e Estender |
| [`avisos.md`](avisos.md) | Avisos de lançamentos futuros (Telegram) |
| [`colaboracao-tempo-real.md`](colaboracao-tempo-real.md) | Proposta de colaboração em tempo real e o plano por fases |
| [`offline-e-sincronizacao.md`](offline-e-sincronizacao.md) | Ideia para o futuro: uso sem internet, níveis, erros da escrita offline, proposta de orquestrador local com fila de eventos e a tendência de projeção do saldo no aparelho, sem envio ao servidor (nada decidido) |
| [`hospedagem.md`](hospedagem.md) | S3 + CloudFront e a decisão sobre o `/api/*` |
| [`ambiente-de-teste.md`](ambiente-de-teste.md) | A stack `financas-dev` |
| [`deploy-github-actions.md`](deploy-github-actions.md) | Deploy, release e as roles OIDC |
| [`fluxo-de-versoes.md`](fluxo-de-versoes.md) | Branches, tags e releases |
| [`specs/modelo-de-dados/`](specs/modelo-de-dados/README.md) | As chaves do DynamoDB, com exemplos fictícios em JSON |
| [`specs/historico/intent.md`](specs/historico/intent.md) | Intent do histórico de versões e recuperação |
| [`specs/historico/plan.md`](specs/historico/plan.md) | Plan do histórico: fases, decisões a confirmar e riscos (decisões fechadas em 2026-10-15 e ajustadas em 2026-10-23) |
| [`specs/historico/spec.md`](specs/historico/spec.md) | Spec do histórico: dados, regras, contratos de API e testes por fase (implementado) |
| [`demos/arquivo/demo-historico-versoes.html`](demos/arquivo/demo-historico-versoes.html) | Demo aprovada da tela Histórico de versões (seção "Histórico de versões (tela)" de `padrao-de-telas.md`) |
| [`demos/arquivo/demo-desfazer.html`](demos/arquivo/demo-desfazer.html) | Demo aprovada do Desfazer depois de excluir ou editar (padrão 11b de `padrao-de-telas.md`; provisório até o histórico) |
| [`specs/fechamento-automatico/`](specs/fechamento-automatico/intent.md) | Intent, plan e spec do item 38: o mês anterior fecha sozinho e mexer no passado exige confirmação (implementado) |
| [`demos/arquivo/demo-tendencias-tags.html`](demos/arquivo/demo-tendencias-tags.html) | Demo da aba Tendências (implementada em `web/js/tendencias.js`): grupos com E, OU e NÃO, modo avançado de expressão livre e aviso de outras tags no gráfico |
| [`demos/arquivo/demo-tendencias-layout.html`](demos/arquivo/demo-tendencias-layout.html) | Demo do layout da aba Tendências: duas colunas no computador, escolhas em tela cheia no celular e tags em botões (vale sobre a demo acima no layout) |
| [`demos/arquivo/demo-tendencias-visao.html`](demos/arquivo/demo-tendencias-visao.html) | Demo do seletor Gastos, Receitas e Resultado da aba Tendências (Gastos é o padrão) e da faixa "Simulação" (implementada em `web/js/tendencias.js`; aprovada em 2026-10-22) |
| [`demos/arquivo/demo-tendencias-lista.html`](demos/arquivo/demo-tendencias-lista.html) | Demo (em avaliação, ainda sem aprovação) da lista dos lançamentos da consulta na Tendências: janela só de leitura aberta por "Ver os N lançamentos" ou por uma barra do mês, soma que bate com o gráfico e limite de linhas (item 42a do backlog) |
| [`demos/arquivo/demo-tendencias-consultas.html`](demos/arquivo/demo-tendencias-consultas.html) | Demo **aprovada em 2026-10-30 e implementada em 2026-10-31** da janela "Consultas salvas" da Tendências: compartilhada no espaço, Renomear e Excluir só nas próprias, o dono modera, Salvar como nova ou Atualizar, exclusão com Desfazer e limite de 50 |
| [`demos/arquivo/demo-lancar-layout.html`](demos/arquivo/demo-lancar-layout.html) | Demo do layout da aba Lançar: valor em destaque, seletores nativos, Tags em janela no celular e na coluna da direita no computador, Transferência como modo (implementada em 2026-10-19) |
| [`demos/arquivo/demo-historico-layout.html`](demos/arquivo/demo-historico-layout.html) | Demo do layout da aba Histórico (lista de lançamentos): filtros numa janela, lançamentos por dia, nota em ícone, ações numa janela de baixo, tabela por mês no computador (aprovada e implementada em 2026-10-19) |
| [`demos/arquivo/demo-voltar-navegador.html`](demos/arquivo/demo-voltar-navegador.html) | Demo do botão voltar do navegador, do Android e do app instalado: pilha do histórico ao vivo, Desfazer fora da pilha e pergunta ao descartar texto digitado (aprovada e implementada em 2026-10-19; seção "Botão voltar" de `padrao-de-telas.md`) |
| [`specs/arquitetura-camadas/`](specs/arquitetura-camadas/intent.md) | Intent, plan e as quatro specs de fatia do item 32: o que falta migrar da raiz do backend para as camadas (as quatro fatias estão implementadas; a 4 aguarda a conferência em dev) |
| [`demos/arquivo/demo-repeticao-e-aviso.html`](demos/arquivo/demo-repeticao-e-aviso.html) | Demo de Repetir e avisar no celular: linha-resumo que abre tela cheia, lista que acumula, janela com rodas; base do Prever e do Estender |
| [`demos/arquivo/demo-futuros.html`](demos/arquivo/demo-futuros.html) | Demo da tela Futuros: uma linha por evento, ações numa janela de baixo, resumo com faixa de atraso e a versão para computador |
| [`demos/arquivo/demo-futuros-largura.html`](demos/arquivo/demo-futuros-largura.html) | Demo de Futuros por largura da janela (vale sobre a demo acima onde diferirem): prazo embaixo do valor, "⋯" no celular, botões no computador |
| [`demos/arquivo/demo-resumo-futuros.html`](demos/arquivo/demo-resumo-futuros.html) | Demo da comparação que decidiu o resumo de Futuros, com valores extremos e larguras de 320 a 390 px |
| [`demos/arquivo/demo-historico-total.html`](demos/arquivo/demo-historico-total.html) | Demo do total no Histórico: soma da lista (renda menos gasto) à direita do contador (aprovada em 2026-10-11) |
| [`demos/arquivo/demo-atalho-saldos.html`](demos/arquivo/demo-atalho-saldos.html) | Demo dos atalhos Histórico e Saldos no cabeçalho, entre o nome da tela e o ⋮ (aprovada em 2026-10-15) |
| [`demos/arquivo/demo-confirmacao-mes-fechado.html`](demos/arquivo/demo-confirmacao-mes-fechado.html) | Demo da confirmação ao mexer em mês fechado: segunda janela, restaurar, importar e mês conferido (aprovada em 2026-10-17) |
| [`demos/arquivo/demo-exclusao-imediata.html`](demos/arquivo/demo-exclusao-imediata.html) | Demo de excluir na hora, com Desfazer por restauração e as recusas antes de sumir (aprovada em 2026-10-17) |
| [`demos/arquivo/demo-menu-usuario.html`](demos/arquivo/demo-menu-usuario.html) | Demo do menu do usuário: nome e Sair saem da linha de cima e vão para o menu (aprovada em 2026-10-17) |
| [`demos/arquivo/demo-historico-periodo.html`](demos/arquivo/demo-historico-periodo.html) | Demo da aba Histórico por período (30d, 90d, 180d e calendário de um mês, abre em 30d), com a busca limitada ao que está marcado e o aviso de "Buscar em tudo" (aprovada pelo dono e implementada em 2026-10-20) |
| [`demos/arquivo/demo-configuracoes.html`](demos/arquivo/demo-configuracoes.html) | Demo da aba Configurações (antes Importar), só do dono, com as seções recolhidas Cartão, Importar e Exportar (aprovada e implementada em 2026-10-23; seção "Configurações (tela)" de `padrao-de-telas.md`) |
| [`demos/arquivo/demo-versoes-periodo.html`](demos/arquivo/demo-versoes-periodo.html) | Demo do Histórico de versões por período (30d, 90d, 180d e calendário, pela data da alteração), busca por descrição, nota e tag, "Buscar em tudo" sem aviso de custo e o aviso de "mais versões"; só lançamentos alterados, com a v1 "Original" dentro das versões (aprovada pelo dono e implementada em 2026-10-23) |
| [`demos/arquivo/demo-confirmar-previsto.html`](demos/arquivo/demo-confirmar-previsto.html) | Demo **aprovada em 2026-10-24** (ainda não implementada) do gesto de confirmar um lançamento previsto em Futuros (item 41 do BACKLOG): janela com Valor, Data e Status, vencidos em Atrasados, data sugerida hoje em mês fechado |
| [`specs/saldo-servidor/`](specs/saldo-servidor/intent.md) | Intent, plan e spec do item 40: saldo, fechamento e conferência calculados e validados no servidor, o primeiro da fila (implementado) |
| [`specs/fatura-servidor/`](specs/fatura-servidor/intent.md) | Intent, plan e spec do item 33, fatia a: faturas do cartão, total, conferência e sugestões de ajuste no servidor, `bancos[].cartao` (implementado) |
| [`specs/futuros-previsto/`](specs/futuros-previsto/intent.md) | Intent, plan e spec do item 41: o marcador `previsto` no lugar dos status `PREVISTO`, `TRANSFERINDO` e `CREDITANDO`, o catálogo de status do backend sem data de corte global, o passado do usuário, Futuros e Histórico prontos pelo servidor e Confirmar (implementado) |
| [`demos/arquivo/demo-atrasados-por-ano.html`](demos/arquivo/demo-atrasados-por-ano.html) | Demo aprovada e implementada em 2026-10-26 (item 45): Atrasados em blocos Ano › Mês recolhíveis e seletor de anos anteriores na linha do título |
| [`demos/arquivo/demo-tema.html`](demos/arquivo/demo-tema.html) | Demo do tema Automático, Claro ou Escuro, como primeiro item do cartão Aparência de Listas, guardado na conta (aprovada e implementada em 2026-10-26) |
| [`specs/recorrencia/`](specs/recorrencia/intent.md) | Intent, plan e spec do item 51: o lançamento marcado como recorrente é copiado sozinho pela rotina diária dos avisos, até o dono desligar (implementado) |
| [`specs/log-de-requisicoes/`](specs/log-de-requisicoes/intent.md) | Intent, plan e spec do item 50: uma linha de log por requisição (ação, status e duração, sem conteúdo nem ids), para achar as leituras repetidas e decidir o índice do Histórico (implementado) |
| [`specs/tendencias-consultas/`](specs/tendencias-consultas/intent.md) | Intent, plan e spec do item 53: consultas salvas da Tendências, compartilhadas no espaço (cada pessoa altera só as suas, o dono modera), um item `CONSULTA#<id>` por consulta, exclusão lógica (implementado) |
| [`specs/app-android/`](specs/app-android/intent.md) | Intent, plan e spec (fases 1 a 3) do app Android com Capacitor: o `web/` empacotado, login pelo navegador do sistema com token no Keystore, biometria opcional, push e Play Store nas fases seguintes (rascunho, nada implementado) |
| [`specs/tendencias-visoes/`](specs/tendencias-visoes/intent.md) | Intent, plan e spec do item 55: o seletor Ver como da Tendências (Ritmo, Calendário, Fixos, Tags, Pagamento e Metas), tudo calculado na tela, com o limite das Metas como único dado novo (rascunho) |
| [`specs/formas-de-pagamento/`](specs/formas-de-pagamento/intent.md) | Intent, plan e spec do item 56: PIX, DEBITO, DINHEIRO e TED como entradas do catálogo de status, em ordem de uso com o `PAGO` no fim, valendo para entrada e saída (implementado no backend) |
| `specs/<feature>/{intent,plan,spec}.md` | O que, como e o roteiro de cada feature |

A coleção do Postman continua em `docs/postman/`, porque é ferramenta de teste da API, não contexto para a IA.
