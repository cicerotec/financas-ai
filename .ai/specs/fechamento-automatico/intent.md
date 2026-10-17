# Intent: fechamento automático do mês anterior

Item 38 do BACKLOG. Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`) vêm
junto, para o dono aprovar antes de qualquer código.

**Estado:** decisões fechadas com o dono em 2026-10-16 (registradas no item 38 do BACKLOG); plan e spec aprovados. Fase A
(backend) implementada; as fases B e C (tela) esperam a demo.

## Por quê

O saldo de um banco é uma **cascata**: alterar um evento de um mês passado muda o saldo de todos os meses seguintes. Hoje só
o mês **conferido** com o extrato trava, e a conferência é uma manualidade herdada da planilha que, na prática, ninguém
vai fazer na tela de Saldos. O dono quer um sistema que flua sozinho (e, mais adiante, com o Open Finance, que nem os
lançamentos sejam digitados) **sem abrir mão da integridade dos dados**: o passado não pode mudar por acidente.

## O que queremos

1. **O mês fecha sozinho.** Um evento de um mês anterior só pode ser alterado com **confirmação explícita**. O mês anterior
   fica livre até o **dia 7** do mês seguinte; **a partir do dia 8** (fuso de São Paulo) exige confirmação. Meses mais antigos
   sempre exigem. O mês atual e os futuros são livres.
2. **A regra é calculada pela data.** Não grava estado nenhum nem campo novo no lançamento.
3. **Vale só para o que muda o saldo** (valor, tipo, status, banco, data, oculto), como a trava do conferido: descrição, nota e
   tags seguem livres. O que é anterior ao **início do controle** do banco também segue livre.
4. **O conferido manual fica como está, opcional e absoluto**: mês conferido continua travado, sem confirmação que o destrave
   (reabrir o mês, como hoje). A confirmação só vale para o mês fechado por data.
5. **O servidor recusa e a tela confirma.** O backend responde `409` com um código próprio (`mes_fechado`); a tela mostra a
   janela de confirmação e reenvia com o sinal de confirmação. O front só exibe, não decide.
6. Vale para **criar, editar, excluir, importar e restaurar** (a restauração da Fase 2 do histórico passa pelo mesmo ponto).
7. **A confirmação fica gravada na versão do histórico** (campo novo só no item `HIST#`, nunca no lançamento vivo).
8. **Excluir passa a ser imediato:** o `DELETE` vai na hora, a recusa aparece antes de o item sumir, e o **Desfazer vira uma
   restauração** (a resposta do `DELETE` diz qual versão restaurar). Depois dos 6 segundos ainda dá para recuperar pela tela
   Histórico de versões. Isso muda o padrão 11b de `padrao-de-telas.md` (demo antes do código).

## Limites (o que fica de fora)

- Não é conferência: não compara com extrato nem gera ajuste. A conferência automática com o saldo do banco é o item 39
  (Open Finance).
- Não bloqueia o mês atual nem o anterior até o dia 7, nem texto (descrição, nota, tags).
- Não muda o conferido nem a trava de **fatura** do cartão (item 33).
- Não calcula saldo no servidor (itens 14 e 33).

## Custo (vai na demo)

Um toque a mais ao editar, excluir ou importar algo de um mês fechado. Ao mesmo tempo, o servidor passa a recusar
imediatamente uma exclusão que hoje só seria recusada 6 segundos depois.

## Pronto quando

- O backend recusa, com `409 mes_fechado`, toda escrita que muda o saldo de um mês fechado por data sem confirmação, e grava
  com ela, registrando a confirmação na versão.
- A tela confirma (criar, editar, importar, restaurar), a exclusão é imediata e o Desfazer restaura, com a demo aprovada antes.
- `padrao-de-telas.md` (padrão 11b), o modelo de dados (`HIST#`) e o BACKLOG ficam em dia na mesma entrega.
