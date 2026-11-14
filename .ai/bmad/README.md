# .ai/bmad: o que os agentes do metodo-ai produziram para este projeto

O metodo-ai (versão pessoal do BMad Method, repositório à parte) planeja, desenha a arquitetura, revisa e testa; quem escreve o código deste projeto é o agente daqui. Esta pasta guarda só os documentos que o dono aprovou, como as premissas e as decisões de arquitetura. Rascunhos e memlogs ficam no metodo-ai.

## Regras
- **Precedência:** onde um documento daqui divergir de [`../guardrails.md`](../guardrails.md), [`../arquitetura.md`](../arquitetura.md) ou [`../decisoes.md`](../decisoes.md), valem estes; a divergência vai ao dono antes de qualquer código.
- Documento novo entra aqui por PR, com uma linha na tabela abaixo.

## Documentos

| Documento | Para quê |
|---|---|
| [`arquitetura-avisos-canais.md`](arquitetura-avisos-canais.md) | Como um canal de aviso (Telegram, push) entra no backend sem vazar para `domain` e `service`: dez decisões (AD-1 a AD-10), o PR 0 antes do push e as emendas à spec do push |
