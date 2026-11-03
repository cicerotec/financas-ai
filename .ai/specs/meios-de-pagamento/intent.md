# Intent: o dono cria, desativa e remove meios de pagamento

Item 43 do BACKLOG. Este documento diz **o quê e por quê**; o como está no [`plan.md`](plan.md) e o roteiro verificável no [`spec.md`](spec.md).

**Estado:** aprovado pelo dono em 2026-11-03 (inclui o `ativoAte` em `CFG#LISTAS`); nada implementado. A demo [`demo-listas-v3-meios`](../../demos/demo-listas-v3-meios.html) foi **aprovada pelo dono em 2026-11-03**.
Decisões da conversa de 2026-11-03 estão marcadas como "decidido" abaixo.

## Por quê

O que o dono chama de **meio de pagamento** (conta, carteira, vale, cartão: `SODEXO-REF-IBM`, `DINHEIRO`, `CARTEIRA`, `CITIBANK`...) é o `banco` gravado em cada
lançamento. É dado do espaço dele, não do produto: ele precisa **colocar e tirar meios sem mexer no passado**. Exemplo: quando sair da IBM, o `SODEXO-REF-IBM`
deve deixar de ser sugerido, mas os lançamentos e saldos antigos continuam com ele.

Hoje a lista de bancos só tem "Adicionar" e um ✕. O ✕ **confere o uso no navegador** (`removerBanco` em `web/js/app.js`, lendo até 1000 lançamentos), o que
quebra a regra de que o front não decide, e não prova nada além do 1000º lançamento.

## O que queremos

1. **Desativar um meio a partir de uma data** (hoje, fim do mês ou outra). Depois da data ele deixa de ser **sugerido** (Lançar, Transferência), mas
   continua nos lançamentos, no saldo e no histórico, e aparece em "Mostrar meios inativos". **Reativar** a qualquer momento.
2. **Remover um meio só se nunca foi usado.** A conferência é feita **pelo servidor, só quando o dono clica em Remover** (decidido): meio com lançamento é
   recusado e o app oferece Desativar. O front só pede e mostra a resposta.
3. **A etiqueta "crédito"** (decidido) vem da aba Cartão: é lá que um meio é ligado a um cartão de crédito. Listas só exibe, o servidor já deduz. Não há
   interruptor de cartão ao adicionar.
4. **Onde guardar** (decidido): o `ativoAte` vai em `CFG#LISTAS`, onde o meio nasce, e não em `CFG#SALDOS`: um cartão sem conta (um `CITIBANK` só de crédito) não tem saldo.
5. **Tela** (demo aprovada): o card "Bancos / contas" de Listas vira **Meios de pagamento**; tocar num meio abre a janela de ações (Desativar ou Reativar, Remover).

## Decisões que ficaram de fora (e por quê)

- **Flag `jaUsado`** (marcar o uso na escrita do lançamento, para a remoção não varrer nada): proposta do dono, guardada em **Ideias** do BACKLOG. Entra se a
  varredura ficar cara. Por isso o servidor **registra no log o custo de cada remoção** (páginas e itens lidos), para o gatilho ser visível.
- **Interruptor "é cartão de crédito" ao adicionar:** a ligação cartão-meio acontece só na aba Cartão (uma fonte só).
- **Migração de meios antigos:** nenhuma. O passado está correto; um meio sem `ativoAte` é ativo.

## Limites (o que fica de fora)

- **Um campo novo no DynamoDB:** `ativoAte`, um mapa `{ "<meio>": "AAAA-MM-DD" }` dentro do item `CFG#LISTAS` que já existe. Sem chave nem item novo. A aprovação de 2026-10-24
  era para `CFG#SALDOS.porBanco`; o lugar mudou em 2026-11-03, então **o dono reaprovou o novo lugar em 2026-11-03.**
- Desativar **não proíbe** lançar no meio (ele aparece em "ver todos"); só deixa de ser sugerido. O servidor não valida lançamento contra o `ativoAte`.
- Saldos, faturas, fechamento e conferência não mudam: um meio inativo continua somando no passado.
- `CRIPTO` e `DREX` são **formas** (status), não meios.
- Permissões por meio (níveis Visível, Resumido e Oculto, item 27) ficam fora.
