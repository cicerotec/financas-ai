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
| [`padrao-de-telas.md`](padrao-de-telas.md) e [`demos/`](demos/) | Padrão de telas aprovado e as demos que são a fonte da verdade |
| [`ux.md`](ux.md) | Observações de uso real e próximos passos de UX |
| [`eventos-futuros.md`](eventos-futuros.md) | Conceito e regras de Prever, Repetir e Estender |
| [`avisos.md`](avisos.md) | Avisos de lançamentos futuros (Telegram) |
| [`colaboracao-tempo-real.md`](colaboracao-tempo-real.md) | Proposta de colaboração em tempo real e o plano por fases |
| [`offline-e-sincronizacao.md`](offline-e-sincronizacao.md) | Ideia para o futuro: uso sem internet, níveis, erros da escrita offline e proposta de orquestrador local com fila de eventos (nada decidido) |
| [`hospedagem.md`](hospedagem.md) | S3 + CloudFront e a decisão sobre o `/api/*` |
| [`ambiente-de-teste.md`](ambiente-de-teste.md) | A stack `financas-dev` |
| [`deploy-github-actions.md`](deploy-github-actions.md) | Deploy, release e as roles OIDC |
| [`fluxo-de-versoes.md`](fluxo-de-versoes.md) | Branches, tags e releases |
| [`specs/modelo-de-dados/`](specs/modelo-de-dados/README.md) | As chaves do DynamoDB, com exemplos fictícios em JSON |
| [`specs/historico/intent.md`](specs/historico/intent.md) | Intent do histórico de versões e recuperação |
| [`specs/historico/plan.md`](specs/historico/plan.md) | Plan do histórico: fases, decisões a confirmar e riscos (decisões fechadas em 2026-10-15) |
| [`specs/historico/spec.md`](specs/historico/spec.md) | Spec do histórico: dados, regras, contratos de API e testes por fase (rascunho) |
| [`demos/demo-historico-versoes.html`](demos/demo-historico-versoes.html) | Demo aprovada da tela Histórico de versões (seção "Histórico de versões (tela)" de `padrao-de-telas.md`) |
| [`demos/demo-desfazer.html`](demos/demo-desfazer.html) | Demo aprovada do Desfazer depois de excluir ou editar (padrão 11b de `padrao-de-telas.md`; provisório até o histórico) |
| [`specs/fechamento-automatico/`](specs/fechamento-automatico/intent.md) | Intent, plan e spec do item 38: o mês anterior fecha sozinho e mexer no passado exige confirmação (rascunho) |
| [`demos/demo-tendencias-tags.html`](demos/demo-tendencias-tags.html) | Demo da aba Tendências (implementada em `web/js/tendencias.js`): grupos com E, OU e NÃO, modo avançado de expressão livre e aviso de outras tags no gráfico |
| [`demos/demo-tendencias-layout.html`](demos/demo-tendencias-layout.html) | Demo do layout da aba Tendências: duas colunas no computador, escolhas em tela cheia no celular e tags em botões (vale sobre a demo acima no layout) |
| [`demos/demo-lancar-layout.html`](demos/demo-lancar-layout.html) | Demo do layout da aba Lançar: valor em destaque, seletores nativos, Tags em janela no celular e na coluna da direita no computador, Transferência como modo (implementada em 2026-10-19) |
| [`demos/demo-historico-layout.html`](demos/demo-historico-layout.html) | Demo do layout da aba Histórico (lista de lançamentos): filtros numa janela, lançamentos por dia, nota em ícone, ações numa janela de baixo, tabela por mês no computador (aprovada e implementada em 2026-10-19) |
| [`demos/demo-voltar-navegador.html`](demos/demo-voltar-navegador.html) | Demo do botão voltar do navegador, do Android e do app instalado: pilha do histórico ao vivo, Desfazer fora da pilha e pergunta ao descartar texto digitado (aprovada e implementada em 2026-10-19; seção "Botão voltar" de `padrao-de-telas.md`) |
| [`specs/arquitetura-camadas/`](specs/arquitetura-camadas/intent.md) | Intent, plan e as quatro specs de fatia do item 32: o que falta migrar da raiz do backend para as camadas (as quatro fatias estão implementadas; a 4 aguarda a conferência em dev) |
| [`demos/demo-repeticao-e-aviso.html`](demos/demo-repeticao-e-aviso.html) | Demo de Repetir e avisar no celular: linha-resumo que abre tela cheia, lista que acumula, janela com rodas; base do Prever e do Estender |
| [`demos/demo-futuros.html`](demos/demo-futuros.html) | Demo da tela Futuros: uma linha por evento, ações numa janela de baixo, resumo com faixa de atraso e a versão para computador |
| [`demos/demo-futuros-largura.html`](demos/demo-futuros-largura.html) | Demo de Futuros por largura da janela (vale sobre a demo acima onde diferirem): prazo embaixo do valor, "⋯" no celular, botões no computador |
| [`demos/demo-resumo-futuros.html`](demos/demo-resumo-futuros.html) | Demo da comparação que decidiu o resumo de Futuros, com valores extremos e larguras de 320 a 390 px |
| [`demos/demo-historico-total.html`](demos/demo-historico-total.html) | Demo do total no Histórico: soma da lista (renda menos gasto) à direita do contador (aprovada em 2026-10-11) |
| [`demos/demo-atalho-saldos.html`](demos/demo-atalho-saldos.html) | Demo dos atalhos Histórico e Saldos no cabeçalho, entre o nome da tela e o ⋮ (aprovada em 2026-10-15) |
| [`demos/demo-confirmacao-mes-fechado.html`](demos/demo-confirmacao-mes-fechado.html) | Demo da confirmação ao mexer em mês fechado: segunda janela, restaurar, importar e mês conferido (aprovada em 2026-10-17) |
| [`demos/demo-exclusao-imediata.html`](demos/demo-exclusao-imediata.html) | Demo de excluir na hora, com Desfazer por restauração e as recusas antes de sumir (aprovada em 2026-10-17) |
| [`demos/demo-menu-usuario.html`](demos/demo-menu-usuario.html) | Demo do menu do usuário: nome e Sair saem da linha de cima e vão para o menu (aprovada em 2026-10-17) |
| [`demos/demo-historico-periodo.html`](demos/demo-historico-periodo.html) | Demo da aba Histórico por período (30d, 90d, 180d e calendário de um mês, abre em 30d), com a busca limitada ao que está marcado e o aviso de "Buscar em tudo" (aprovada pelo dono em 2026-10-20; ainda não implementada) |
| `specs/<feature>/{intent,plan,spec}.md` | O que, como e o roteiro de cada feature |

A coleção do Postman continua em `docs/postman/`, porque é ferramenta de teste da API, não contexto para a IA.
