# .ai: contexto para quem constrói com IA

Tudo o que a IA precisa ler antes de agir fica aqui, separado do `BACKLOG.md`, que é só backlog (roadmap, o que já foi
feito e ideias).

| Documento | Para quê |
|---|---|
| [`guardrails.md`](guardrails.md) | Regras que valem para toda tarefa (arquitetura, telas, git, segurança, testes) |
| [`arquitetura.md`](arquitetura.md) | Camadas, regras no servidor, histórico imutável, domínio do cartão |
| [`security.md`](security.md) | Autenticação, papéis e permissões, dados, segredos, infraestrutura e pontos em aberto |
| [`decisoes.md`](decisoes.md) | Decisões de produto fechadas e descartadas |
| [`padrao-de-telas.md`](padrao-de-telas.md) e [`demos/`](demos/) | Padrão de telas aprovado e as demos que são a fonte da verdade |
| [`ux.md`](ux.md) | Observações de uso real e próximos passos de UX |
| [`eventos-futuros.md`](eventos-futuros.md) | Conceito e regras de Prever, Repetir e Estender |
| [`avisos.md`](avisos.md) | Avisos de lançamentos futuros (Telegram) |
| [`colaboracao-tempo-real.md`](colaboracao-tempo-real.md) | Proposta de colaboração em tempo real e o plano por fases |
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
| `specs/<feature>/{intent,plan,spec}.md` | O que, como e o roteiro de cada feature |

A coleção do Postman continua em `docs/postman/`, porque é ferramenta de teste da API, não contexto para a IA.
