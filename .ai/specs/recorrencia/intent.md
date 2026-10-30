# Intent: recorrência automática de lançamentos (item 51)

Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`) vêm junto, para o dono aprovar antes de qualquer código ou demo.

**Estado:** rascunho para aprovação (2026-10-29). As decisões abaixo saíram da conversa com o dono em 2026-10-29; o que ainda é proposta está marcado em "Propostas que o dono pode contestar" no [`plan.md`](plan.md).

## Por quê

O dono tem contas que se repetem para sempre: colégio, internet, água, luz, aluguel, e outras a cada dois, seis ou doze meses. Hoje ele consegue criá-las como previstos, mas **mês a mês**:
o **Prever** e o **Repetir** criam um número fixo de cópias, e quando elas acabam é preciso estender a série ou criar tudo de novo. A ideia do dono: **marcar o lançamento como recorrente uma vez** e deixar o
sistema copiá-lo sozinho, com todas as características (valor, banco, tags, nota, aviso do Telegram), **até ele desligar**. É o mesmo espírito do saldo e do fechamento do mês, que o dono já não confere
à mão: o servidor faz sozinho, pela data.

## O que queremos

1. **Marcar uma vez.** O lançamento ganha o atributo `recorrente: { intervalo }` (no `TX#`, o próprio lançamento), com o intervalo em meses (1, 2, 3, 6, 12 ou outro). O intervalo é escolhido na tela **Repetir** que já existe, sem a linha "Repetir quantas vezes". Não existe "tipo novo" de lançamento: a recorrência é um
   lançamento previsto qualquer com essa marca.
2. **O servidor cria as cópias.** A rotina diária que já manda os avisos do Telegram passa a criar também a próxima cópia de cada recorrência, **30 dias antes** do vencimento, para ela aparecer em Futuros com
   tempo. A cópia leva os mesmos dados do último registro (valor, banco, tags, nota, status da origem, aviso) e **herda a marca**, para a recorrência seguir sozinha.
3. **Quem comanda é o `AVISOS#`.** A rotina não varre a tabela: ela lê o item auxiliar da recorrência no `AVISOS#ATIVOS`, que guarda o que ela precisa (intervalo, próxima data, último registro). O lançamento
   (`TX#`) só carrega a marca e o intervalo, para a tela mostrar e para ligar e desligar.
4. **Recorrência não é série.** O **Repetir** continua com série (finito: o dono pede N repetições). A recorrência não tem fim, e as cópias dela **não levam `serie`**.
5. **Pular o mês é apagar a cópia.** Se o dono apaga a cópia que a rotina criou, aquele mês fica pulado e a recorrência segue no próximo. A rotina sabe disso porque a "próxima data" já avançou ao criar a cópia.
6. **Desligar e religar.** Desligar não apaga nada que já foi criado: as cópias futuras ficam como estão. Religar continua do **próximo vencimento, sempre para frente**, sem recuperar os meses em que esteve desligada.
7. **Criar não grava histórico.** A cópia nasce como qualquer lançamento criado: sem versão no `HIST#` (regra R4 do histórico). Versão só existe quando alguém altera ou exclui.
8. **O valor segue o último registro.** Água e luz mudam; se o dono ajusta o valor do mês, a cópia seguinte sai com esse valor.
9. **O passado não é tocado.** Nenhum lançamento existente é alterado nem migrado.

## Limites (o que fica de fora)

- **Sem chave nova no DynamoDB.** Nada de `REC#` nem de tipo de registro à parte: o item auxiliar da recorrência é mais um `tipo` dentro do `AVISOS#ATIVOS`, como o da fatura do cartão. O que o item guarda é
  a recorrência, e não uma cópia do lançamento.
- **Sem recorrência com fim ou com quantidade** (isso é o Repetir) e **sem edição em massa** das cópias.
- **Sem janela própria por recorrência:** os 30 dias são um só para todo o sistema, uma constante no código, sem tela.
- **Não vale para transferências** (as pontas de uma transferência não entram, como no Prever).
- **Não decide status:** a cópia sai com o status da origem, e o servidor marca como previsto o que tem data futura. Nenhuma tela pergunta status.
- **Não faz o lançamento ser "real" antes da hora:** a cópia é um previsto e só pesa no saldo quando o dono a confirmar, como qualquer previsto.

## Custo (vai na demo)

Uma recorrência esquecida continua criando cópias todo mês até alguém desligá-la. Para isso a tela precisa mostrar **quais lançamentos são recorrentes** e permitir parar. Apagar uma cópia **não** para a
recorrência (só pula o mês); parar é uma ação à parte.

## Pronto quando

- A rotina diária cria a próxima cópia de cada recorrência na janela, sem duplicar e sem varrer a tabela, e os testes cobrem intervalo 1, 2, 6 e 12, dia 31, apagar e pular, desligar e religar, e rodar duas vezes.
- A tela marca, mostra e desliga a recorrência, com a demo aprovada antes (reaproveitando a tela Repetir, sem "Repetir quantas vezes").
- `modelo-de-dados`, `eventos-futuros.md` (a "regra recorrente sem fim" deixa de ficar de fora), `padrao-de-telas.md`, o guardrails e o BACKLOG ficam em dia na mesma entrega.
