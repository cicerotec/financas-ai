# Plan: backup (item 11)

Como construir o que o [`intent.md`](intent.md) pede. Premissa: uma pessoa só, tabela pequena, custo perto de zero. Cada fase é um PR pequeno e deixa o
app mais protegido que o anterior.

**Estado:** rascunho para aprovação (2026-10-26). Nada implementado.

## Onde estamos hoje

| Fato | Onde | Consequência |
|---|---|---|
| Tabela única `FinancasApp`, `DeletionPolicy: Retain`, PITR só em prod | `template.yaml` | `Retain` protege do CloudFormation, não de a AWS apagar ou suspender a conta |
| Região `sa-east-1`, sessão MFA por `scripts/aws-mfa.ps1` | `samconfig.toml`, `scripts/` | O script de backup reaproveita a sessão MFA; sem credencial nova |
| Exportar gera `.json` v2 só com lançamentos e configurações | `README.md`, `padrao-de-telas.md` | Não serve como backup completo: faltam `HIST#`, `USER#`, `SALDO#*`, `SEQ` |
| A Lambda tem `Deny` de `UpdateItem` e `DeleteItem` em `HIST#*` | `template.yaml` (`HistoricoImutavel`) | O restaurador roda com role administrativa (só `PutItem`); o `Deny` não o impede |
| O vínculo `USER#<sub>` depende do `sub` do Cognito, que não se importa | `modelo-de-dados/README.md` | Se o pool sumir, o restaurador precisa remapear os `USER#` |
| Repositório público; `.gitignore` já exclui `/local/`, `web/config*.js`, `samconfig.toml` | `.gitignore` | Dump e arquivos ignorados nunca entram no git; o script recusa gravar dentro do repositório |

## Fases (cada uma é um PR)

### Fase 0: medir e decidir (sem código)
- `aws dynamodb describe-table` em prod: `ItemCount` e `TableSizeBytes`. Define se um `Scan` simples basta e quanto custa guardar.
- Fechar as decisões listadas no intent (cifra, destino, espelho, fase 3).

### Fase 1: dump manual cifrado e restauração testada (1 a 2 dias, custo zero)
Não toca `template.yaml`, não pede permissão nova, não mexe nas roles do Actions.
- `scripts/backup.py`: sessão MFA, `Scan` paginado com ritmo controlado (respeita os 5 RCU), JSON tipado do DynamoDB com **todos** os prefixos, cifra, hash
  e grava fora do repositório (recusa gravar dentro dele).
- `scripts/restaurar.py`: lê, decifra e faz `BatchWriteItem` na ordem `SPACE#`, `USER#`, `HIST#`; tem `--simular`; confere contagens por prefixo ao final.
  Em tabela com 5 WCU a carga é lenta: o roteiro diz como subir a capacidade (ou on-demand) só durante a restauração.
- **Primeiro teste** no `dev` (dados fictícios), depois com um dump real numa tabela temporária.
- Rotina: semanal e manual, copiando para o destino externo.

### Fase 2: o que a restauração não vê (meio dia a 1 dia)
- Exportar `list-users` do Cognito junto do dump.
- Roteiro de recuperação com o pool perdido: recriar a stack e o pool, convidar de novo, passo do `restaurar.py` que **remapeia `USER#<sub>`**.
- Documentar em `.ai/` o que não está no DynamoDB (SSM do Telegram, `config.js`, `samconfig.toml`, variáveis e `environments` do Actions) e onde guardar cada um.

### Fase 2b: código fora do GitHub e do PC (meio dia, custo zero)
- Remoto `espelho` num repositório **privado** em outro provedor (conta com outro e-mail e 2FA), com `git push --mirror espelho`.
- `git bundle create financas-ai.bundle --all` semanal, guardado junto do dump (repositório público: o bundle não precisa de cifra).
- Guardar `web/config.js`, `web/config.dev.js` e `samconfig.toml` no gerenciador de senhas ou num cofre cifrado.
- Opcional: Action diária de `push --mirror` (serve ao esquecimento, não à perda da conta do GitHub).

### Fase 3: automatizar na AWS (2 a 3 dias, centavos por mês; só se a rotina manual falhar)
Mexe no `template.yaml`: **atualizar as roles OIDC antes do merge** e passar pelo dev primeiro.
- EventBridge diário mais Lambda de backup (ou `ExportTableToPointInTime`, que usa o PITR e não gasta leitura).
- Bucket com versionamento e Object Lock **em outra região** (melhor ainda numa segunda conta); cifra no fluxo; a credencial de backup só escreve.
- Retenção por ciclo de vida: 30 diários e 12 mensais. A fase 1 continua como a cópia **fora da AWS**, agora puxada do bucket.

### Fase 4: vigília (1 dia)
- Alarme se não houver dump novo em 36 horas (SNS ou o Telegram que já existe).
- AWS Budget de US$ 1: já feito na fase 1 do item 49 (multiusuário); aqui só conferir que continua ativo.
- Lembrete trimestral para o simulado de restauração.

## Riscos

| Risco | Resposta |
|---|---|
| O dump é dado financeiro real | Cifrado antes de sair, nunca no git, chave fora da AWS |
| Perder a chave da cifra inutiliza o backup | Duas cópias independentes da chave, nunca junto do dump |
| O `Scan` consome leitura de uma tabela de 5 RCU | Ritmo controlado; na fase 3, o export por PITR |
| Mudança futura de schema quebra o restaurador | O restaurador não depende do código da Lambda; o simulado trimestral pega |
| O backup vira rotina esquecida | A fase 3 automatiza e a fase 4 alarma |
| O espelho fica desatualizado | `push --mirror` junto do backup semanal; Action opcional |

## Ordem recomendada

Fases 0, 1, 2 e 2b: cerca de uma semana de trabalho, custo zero, e já resolvem "a AWS apagou tudo" e "perdi o GitHub". As fases 3 e 4 entram depois que a
rotina manual provar o hábito.

## Decisões que o plan fixou e o dono pode contestar

1. A primeira entrega é um **script manual**, não infraestrutura.
2. O restaurador usa `PutItem` com role administrativa; o `Deny` de `HIST#*` da Lambda continua como está.
3. O repositório de backup do código, se existir, é **privado e separado** deste; o dump de dados nunca vai ao git.
4. AWS Backup entre regiões fica de fora; a fase 3 o substitui.
