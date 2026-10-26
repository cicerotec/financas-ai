# Intent: backup que sobrevive à perda da conta AWS e do GitHub (item 11)

Item 11 do BACKLOG (parte do backup periódico). Este documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md).

**Estado:** rascunho para aprovação (2026-10-26). Nada foi implementado.

## O problema

Hoje a única proteção dos dados é o PITR do DynamoDB (só em prod, 35 dias, mesma conta e mesma região) e o botão Exportar. Se a AWS apagar ou
suspender a conta, o PITR vai junto. O Exportar não restaura a tabela inteira: leva `colecoes.lancamentos` e os `documentos` de configuração, mas **não**
o histórico de versões (`HIST#`), os vínculos `USER#`, o `SALDO#*` nem o `SEQ`. Do lado do código, o repositório tem uma cópia no GitHub e outra na
máquina do dono: perder o acesso ao GitHub e ao PC deixa o projeto sem fonte.

## O que queremos

1. **Um dump completo da tabela**, com todos os tipos de item (`SPACE#`, `USER#`, `HIST#`), **cifrado** e guardado **fora da conta AWS**.
2. **Um `restaurar` testado**: recria a tabela vazia pelo `template.yaml`, recarrega o dump e confere as contagens. Backup nunca restaurado não conta.
3. **O que o DynamoDB não guarda** documentado e salvo: a lista de usuários do Cognito (e como remapear `USER#<sub>` se o pool também sumir), o token do
   Telegram no SSM e os arquivos ignorados pelo git (`web/config.js`, `web/config.dev.js`, `samconfig.toml`).
4. **O código-fonte** com uma cópia fora do GitHub e fora do PC: um espelho privado em outro provedor e um `git bundle` guardado junto do dump.
5. **Depois**, automação na AWS (dump diário para um bucket em outra região) e vigília (alarme de dump que não veio, AWS Budget).

## Decisões já tomadas (dono, 2026-10-26)

- **Dado real nunca no git deste repositório** (ele é público), nem cifrado: o git não esquece.
- A primeira entrega é **manual** (um script rodado pelo dono, com a sessão MFA que ele já usa): custo zero e sem permissão nova na infra.
- AWS Backup com cofre entre regiões fica **fora**: seria redundante com a automação da fase 3.

## Limites (o que fica de fora)

- Cifragem por campo com KMS (`privacidade-e-cifragem.md`) é outra decisão e não bloqueia o backup.
- RPO de minutos fora da janela do PITR.
- Nenhuma chave ou campo novo no DynamoDB: tudo é script e, mais tarde, infraestrutura.
- Issues e PRs do GitHub não vão no espelho (as decisões importantes já estão em `.ai/` e no `BACKLOG.md`).

## Custo

Fases 1 a 2b: zero (um `Scan` ocasional na tabela de 5 RCU e armazenamento no que o dono já tem). Fase 3: centavos por mês (S3, Lambda).

## Pronto quando

- Existe um dump cifrado de prod fora da AWS e um `restaurar` que já o devolveu numa tabela temporária, com as contagens por prefixo de chave iguais.
- O roteiro de recuperação (stack, Cognito, SSM, arquivos ignorados) está escrito e a cópia do código existe em dois lugares fora do GitHub e do PC.
- `security.md` (seção Infraestrutura) e o BACKLOG ficam em dia na mesma entrega.

## Para o dono decidir

- Ferramenta de cifra: `age` (simples no Windows) ou GPG.
- Destino fora da AWS: disco externo, Drive ou Backblaze.
- Espelho do código: GitLab ou Codeberg.
- Se a fase 3 (automação) entra agora ou depois, e se vale criar uma segunda conta AWS para o bucket.
