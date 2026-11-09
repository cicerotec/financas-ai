# Intent: o dono cria, desativa e remove meios de pagamento

Item 43 do BACKLOG. Este documento diz **o quê e por quê**; o como está no [`plan.md`](plan.md) e o roteiro verificável no [`spec.md`](spec.md).

**Estado:** aprovado pelo dono em 2026-11-03 (inclui o `ativoAte` em `CFG#LISTAS`); fases A (#347) e B (#348) concluídas em 2026-11-03 e conferidas em dev pelo dono. A demo [`demo-listas-v3-meios`](../../demos/demo-listas-v3-meios.html) foi **aprovada pelo dono em 2026-11-03**.
Decisões da conversa de 2026-11-03 estão marcadas como "decidido" abaixo.
**Revisado pelo item 58 (2026-11-08):** desativar passou a ser **encerrar a conta** (itens 1 e 6 abaixo). O backend e as telas foram mergeados no PR 386; a regra de custo da conferência está na branch `fix/desativar-sem-varrer-passado`.

## Por quê

O que o dono chama de **meio de pagamento** (conta, carteira, vale, cartão: `SODEXO-REF-IBM`, `DINHEIRO`, `CARTEIRA`, `CITIBANK`...) é o `banco` gravado em cada
lançamento. É dado do espaço dele, não do produto: ele precisa **colocar e tirar meios sem mexer no passado**. Exemplo: quando sair da IBM, o `SODEXO-REF-IBM`
deve deixar de ser sugerido, mas os lançamentos e saldos antigos continuam com ele.

Hoje a lista de bancos só tem "Adicionar" e um ✕. O ✕ **confere o uso no navegador** (`removerBanco` em `web/js/app.js`, lendo até 1000 lançamentos), o que
quebra a regra de que o front não decide, e não prova nada além do 1000º lançamento.

## O que queremos

1. **Desativar um meio = encerrar a conta (item 58).** Só vale **a partir de hoje** (não há "fim do mês" nem "outra data"). Antes de gravar, o servidor confere, sem alterar nada, se o meio
   ainda participa de algum evento (saldo diferente de zero; lançamento previsto ou futuro de hoje em diante; recorrência ativa; cartão com fatura sem pagamento registrado, nas três últimas; pagador de
   cartão ou de fatura) e, se sim, recusa com a lista do que falta. Depois de desativado, o meio **não recebe nada novo** (criar, copiar pela recorrência, Transferir, Confirmar, restaurar, trocar o banco para ele;
   editar só aceita descrição, nota e tags; importar só aceita registro anterior à `dataInicio`), **some de Saldos, Cartão e Futuros** e fica **só no Histórico, com a marca "desativado"**,
   escolhível no filtro de meio. **Reativar** sempre é possível.
2. **Remover um meio só se nunca foi usado.** A conferência é feita **pelo servidor, só quando o dono clica em Remover** (decidido): meio com lançamento é
   recusado e o app oferece Desativar. O front só pede e mostra a resposta.
3. **A etiqueta "crédito"** (decidido) vem da aba Cartão: é lá que um meio é ligado a um cartão de crédito. Listas só exibe, o servidor já deduz. Não há
   interruptor de cartão ao adicionar.
4. **Onde guardar** (decidido): o `ativoAte` vai em `CFG#LISTAS`, onde o meio nasce, e não em `CFG#SALDOS`: um cartão sem conta (um `CITIBANK` só de crédito) não tem saldo.
5. **Tela** (demo aprovada): o card "Bancos / contas" de Listas vira **Meios de pagamento**; tocar num meio abre a janela de ações (Desativar ou Reativar, Remover). Telas do item 58 (demos
   aprovadas em 2026-11-08): Listas v6, Histórico v3, Saldos v3, Cartão v2 e Lançar v7.
6. **Custo baixo, sem índice (decidido pelo dono, 2026-11-08).** A tabela fica em **5 RCU e 5 WCU**; **índice só em último caso e só depois de provado**. A conferência do Desativar **não lê o passado**:
   o saldo parte do **último fechamento** (mês conferido ou cache do fechamento automático; a `dataInicio` é só o piso) e, **sem nenhum ponto de partida**, a conferência recusa antes de ler lançamento algum;
   **previsto é só futuro** (previsto no passado não existe) e se lê só de hoje em diante. Cada conferência registra páginas e ms no log. Origem: a primeira versão varria o espaço duas vezes e deu 502 em dev
   (limite de 29 s do API Gateway).

## Decisões que ficaram de fora (e por quê)

- **Flag `jaUsado`** (marcar o uso na escrita do lançamento, para a remoção não varrer nada): proposta do dono, guardada em **Ideias** do BACKLOG. Entra se a
  varredura ficar cara. Por isso o servidor **registra no log o custo de cada remoção** (páginas e itens lidos), para o gatilho ser visível. **Observado em dev (2026-11-08):** remover um meio nunca usado
  lê o espaço inteiro (a tabela tem 5 RCU e o espaço tem ~12.770 lançamentos); fica registrado como ponto de custo conhecido, sem índice (ver o item 6).
- **Interruptor "é cartão de crédito" ao adicionar:** a ligação cartão-meio acontece só na aba Cartão (uma fonte só).
- **Migração de meios antigos:** nenhuma. O passado está correto; um meio sem `ativoAte` é ativo.

## Limites (o que fica de fora)

- **Um campo novo no DynamoDB:** `ativoAte`, um mapa `{ "<meio>": "AAAA-MM-DD" }` dentro do item `CFG#LISTAS` que já existe. Sem chave nem item novo. A aprovação de 2026-10-24
  era para `CFG#SALDOS.porBanco`; o lugar mudou em 2026-11-03 e **o dono reaprovou o novo lugar no mesmo dia**.
- ~~Desativar **não proíbe** lançar no meio (ele aparece em "ver todos"); só deixa de ser sugerido. O servidor não valida lançamento contra o `ativoAte`.~~
  **Revisado em 2026-11-08 pelo item 58:** desativar passa a ser "encerrar" e **proíbe** lançar (criar, Transferir, recorrência, restaurar, Confirmar previsto); só se desativa a partir de hoje, com
  saldo zero e sem nenhum evento de hoje em diante (o passado não é lido); importar só aceita registros anteriores à `dataInicio`. Detalhe em [`BACKLOG.md`](../../../BACKLOG.md#item-58). Isto aqui fica como registro do que valeu até então.
- ~~Saldos, faturas, fechamento e conferência não mudam: um meio inativo continua somando no passado.~~ **Revisado pelo item 58:** o meio desativado sai de Saldos, Cartão e Futuros (tem saldo zero e nenhum
  evento); o passado continua intacto e visível no Histórico, com a marca "desativado".
- `CRIPTO` e `DREX` são **formas** (status), não meios.
- Permissões por meio (níveis Visível, Resumido e Oculto, item 27) ficam fora.
