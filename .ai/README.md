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
| `specs/<feature>/{intent,plan,spec}.md` | O que, como e o roteiro de cada feature (**a criar**) |

A coleção do Postman continua em `docs/postman/`, porque é ferramenta de teste da API, não contexto para a IA.
