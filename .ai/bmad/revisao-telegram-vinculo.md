---
name: 'Revisão da spec telegram-vinculo'
type: review
status: aplicada em 2026-11-20 em intent.md, plan.md, spec.md e na spine; D2 a D7 decididas, P13 conferido (UUID) e emendas à spine aprovadas pelo dono em 2026-11-20; nada pendente
created: '2026-11-20'
target: '.ai/specs/telegram-vinculo/{intent,plan,spec}.md'
sources: ['.ai/bmad/arquitetura-avisos-canais.md', 'backend/service/avisos.go (lido em 2026-11-20)', 'backend/repository/ (consultas, MFACODIGOS#)', 'template.yaml (sem TTL, Deny em HIST#*)', 'app-android/capacitor.config.json']
---

# Revisão da spec telegram-vinculo (visão do arquiteto)

Feita pelo Winston (arquiteto do metodo-ai) em 2026-11-20, com as lentes adversarial e edge-case. **Não substitui a revisão independente das premissas** que o `plan.md` registra como dispensada. As correções abaixo **já estão aplicadas** em `intent.md`, `plan.md` e `spec.md`; este documento diz o que mudou e por quê. Rascunho original e histórico: `metodo-ai/_bmad-output/initiative-financas-ai/review-telegram-vinculo/`.

## Decisões do dono (2026-11-20)

- **Agrupamento da rotina de avisos: por pessoa** (plan D7, spec R13). A chave do `porDestino` em `service/avisos.go:165` passa de `Destino{canal, endereço}` para `(Destino, sub)`. Spaces da mesma pessoa no mesmo chat viram uma mensagem só, com a contagem somada e sem dizer de qual space; pessoas diferentes nunca se misturam. Vale ao push também. Motivo: "é melhor estranhar do que ferir regra de privacidade".
- **Chat id digitado antes da feature (A4):** só existe o do dono. A exceção de isolamento fecha quando ele refizer o vínculo pelo link (V-T31); a partir da fase 3 o servidor recusa Id digitado, então nenhum Id sem prova nasce de novo. Sem marcador e sem campo novo.
- **Dado novo do código de vínculo (D1): opção C.** Um item por pessoa e por canal em `USER#<sub>` / `LINK#telegram`, no molde do `MFACODIGOS#PENDENTE`. Gerar substitui; usar apaga; o teto é o número de pessoas. O link leva o `sub` empacotado (16 bytes) mais 16 bytes aleatórios. Descartadas: **A.** `TGLINK#<hash>` (partição nova por código: código nunca usado não tem quem o ache para apagar, e a tabela não tem TTL) e **B.** campos no vínculo (o link teria de levar `sub` e `sid`, sem folga nos 64 caracteres do `?start=`). Depende de **P13**: o `sub` do pool precisa ser UUID.
- **Critério para chave ou campo novo:** as quatro perguntas (já existe? quem lê e como? como cresce e como sai? há precedente?) estão em `.ai/guardrails.md` e resumidas no `AGENTS.md`.

## O que a revisão achou e onde foi corrigido

| # | Achado | Onde ficou |
|---|---|---|
| A1 | A exclusão condicional que falha gasta WCU; a promessa "nenhuma escrita sem código válido" era falsa | spec R6 (lê antes, `GetItem`; escrita só com código que existe) |
| A2 | Religar trocava o chat id em silêncio | spec R6a (aviso fixo ao chat antigo) |
| A3 | A confirmação não dizia a quem o chat foi ligado; `/parar` não desliga | spec R7 e R8; plan D6 (apelido, **decidido**) |
| A4 | Chat ids sem prova | dissolvido (ver decisões); `intent.md` tabela de exceções |
| A5 | Faltavam as camadas do serviço do vínculo | spec R28; plan "Desenho" |
| A6 | A spine contradizia a feature | `plan.md` "Emendas à spine"; spine amendada e **copiada para `.ai/bmad/arquitetura-avisos-canais.md` em 2026-11-20** (AD-1, AD-2, AD-4, AD-9, AD-10 e a nova AD-11) |
| A7 | Segredo do webhook em cache quebra a rotação | spec R4 (cache de no máximo 5 minutos), V-T3a |
| A8 | O link podia ser gerado antes de o webhook existir | spec R3 (503 sem o segredo) e R24 (script grava por último) |
| A9 | Reentrega do Telegram gerava mensagens contraditórias | spec R5 (`update_id`), V-T5a |
| A10 | Janela entre consumir e gravar | spec R6 ("alternativa avaliada e não adotada": a transação não devolve o valor antigo) |
| A11 | Código nunca usado acumulava | resolvido pela opção C do D1 |
| A12 | Webhook e API dividem a Lambda | plan "Riscos" e spec "Fora do escopo" (aceito agora, medido no item 65) |
| E1 | Mistura de pessoas no mesmo chat | decisão do agrupamento por pessoa; V-T12a |
| E2 a E8 | "Já iniciei" com falso, recarga no Aguardando, "outro ambiente", vínculo recriado, ordem do script, bundle Android, cabeçalhos | spec R17, R6, R24, R28; plan P9 |

## O que o código mostrou (lido em 2026-11-20)

- A rotina agrupa por destino sem `sid` nem `sub` na chave (`service/avisos.go:165-166`).
- Não há `Scan` nem índice nos repositórios; as consultas usam `PK = :pk`. Um item em `USER#<sub>` com `SK` fora de `SPACE#` não aparece em `GET /me` (`repository/espacos.go:49`).
- A tabela não tem TTL (`template.yaml`) e tem 5 RCU e 5 WCU. O `Deny` de `UpdateItem` e `DeleteItem` vale só em `HIST#*`.
- O app Android embute o bundle web (`webDir: "www"`, sem `server.url`): um app desatualizado vê o Telegram como desligado depois da fase 3 (plan P9, aceito).

## Pendências

1. **Plan D2 a D6:** decididas pelo dono em 2026-11-20 (`/parar` sem índice, o 66 antes do 47, `GET /me` com `canais` e sem `telegramChatId`, limites do plan, apelido na confirmação).
2. **P13:** conferido pelo dono em 2026-11-20: o `sub` do `dev` é UUID.
3. **Não verificado:** como o roteador repassa os cabeçalhos do Function URL ao controller (spec R28 e V-T3).
