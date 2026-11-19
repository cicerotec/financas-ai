# .ai/bmad: o que os agentes do metodo-ai produziram para este projeto

O metodo-ai (versão pessoal do BMad Method, repositório à parte) planeja, desenha a arquitetura, revisa e testa; quem escreve o código deste projeto é o agente daqui. Esta pasta guarda só os documentos que o dono aprovou, como as premissas e as decisões de arquitetura. Rascunhos e memlogs ficam no metodo-ai.

## Regras
- **Precedência:** onde um documento daqui divergir de [`../guardrails.md`](../guardrails.md), [`../arquitetura.md`](../arquitetura.md) ou [`../decisoes.md`](../decisoes.md), valem estes; a divergência vai ao dono antes de qualquer código.
- Documento novo entra aqui por PR, com uma linha na tabela abaixo.

## Documentos

| Documento | Para quê |
|---|---|
| [`arquitetura-avisos-canais.md`](arquitetura-avisos-canais.md) | Como um canal de aviso (Telegram, push) entra no backend sem vazar para `domain` e `service`: dez decisões (AD-1 a AD-10), o PR 0 antes do push e as emendas à spec do push |
| [`arquitetura-mfa-recuperacao.md`](arquitetura-mfa-recuperacao.md) | Duas decisões que a spec [`mfa-opcional`](../specs/mfa-opcional/spec.md) precisa absorver antes da Fase 3: a rota pública `POST /mfa/recuperar` como etapa do roteador antes do token (AD-1) e a lista de códigos gerada pendente e ativada só ao confirmar (AD-2), com os testes, as emendas à spec e a ordem de trabalho. Repassada ao agente do projeto em 2026-11-19; a conversa sobre o DynamoDB segue pendente |
| [`revisao-mfa-opcional-pendencias.md`](revisao-mfa-opcional-pendencias.md) | O resto da revisão de arquitetura da spec `mfa-opcional`, depois de A1 e A5: cinco correções à atualização (tempo do R28, "dano da própria pessoa", resíduos, ramo das rotas autenticadas novas, a regra "Seção API"), os ajustes de spec (segundo login x retorno, revoke, CORS, entrada da recuperação, ação de leitura, normalização do e-mail, TTL da intenção) e seis decisões do dono (D1 a D6) |
| [`api-indice/`](api-indice/README.md) | Desenho do índice de rotas `.ai/api.md` (as 55 rotas de hoje, em 48 linhas) e do teste `backend/api_doc_test.go` que falha o `go test ./...` quando o índice e o código divergem; os dois foram **instalados em 2026-11-19** (`.ai/api.md` e `backend/api_doc_test.go`) e não são mais copiados aqui; a pasta guarda o desenho e o que fazer junto com a Fase 3 do MFA |
