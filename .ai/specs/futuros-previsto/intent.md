# Intent: Futuros com o marcador `previsto` e status do backend

Item 41 do BACKLOG. Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`) vêm
junto, para o dono aprovar antes de qualquer código.

**Estado:** rascunho para aprovação, **revisto em 2026-10-24** depois da regra do dono: **o front não trata status de forma
nenhuma**. As decisões vêm das conversas de 2026-10-18 e 2026-10-24 (registradas em [`decisoes.md`](../../decisoes.md) e em
[`arquitetura.md`](../../arquitetura.md)); a demo do gesto de confirmar foi aprovada em 2026-10-24
([`demo-confirmar-previsto.html`](../../demos/demo-confirmar-previsto.html)). **Ordem de execução:** este item vem **depois** dos
itens 40 ([`saldo-servidor`](../saldo-servidor/intent.md)) e 33 fatiado ([`fatura-servidor`](../fatura-servidor/intent.md)), que
tiram do navegador as contas que ainda usam status. **Pendente do dono:** o nome e o tipo do campo novo no DynamoDB (proposta:
`previsto`, booleano gravado só quando ligado).

## A regra que organiza tudo

**O backend manda a lista de status; o front só a exibe, guarda a escolha da pessoa e a devolve.** O front não sabe o que há na
lista, não decide nada por nome, não monta lista nenhuma e não faz conta com status. Hoje a lista é a de compatibilidade com o
passado; no corte desta entrega ela passa a ser `["PAGO", "CREDITO"]`; no futuro, `["PIX", "DREX", "DEBITO", ...]`, **sem tocar
no front**. A lista fixa de `web/js/app.js` (linha 5) deixa de existir.

## Por quê

Hoje "ser futuro" é uma **etiqueta**: o status `PREVISTO`, `TRANSFERINDO` ou `CREDITANDO`, mais uma caixa `statusFuturo` em
Listas que o app edita. Quem decide o que é futuro, o que entra no saldo e o que aparece em cada aba é o navegador. Isso tem
problemas que o dono apontou:

1. **O lançamento futuro não tem status real.** A tela de Futuros mostra `PREVISTO`, e o status que o lançamento terá (`PAGO`,
   `CREDITO`) só aparece depois de trocar o status à mão.
2. **A regra mora no front** (`ehFuturo`, `afetaSaldo`, `resumoFuturos`, `gruposFuturos`), contra o princípio "o front não
   decide" (`arquitetura.md`, princípio 1 e o corolário de 2026-10-18).
3. **O status é editável pelo app.** Quem cria um status como "evento futuro" muda o comportamento do sistema inteiro. Isso é
   regra de negócio, não preferência de tela.

## O que queremos

1. **Um marcador `previsto` no lançamento** no lugar dos três status. O lançamento futuro nasce com o **status real** e o
   marcador ligado. **Enquanto o marcador existir, o lançamento fica fora do saldo e fora do Histórico.**
2. **O servidor liga o marcador:** em todo lançamento criado (ou editado) com data **depois de hoje**, no fuso de São Paulo.
   O Prever e o "Repetir ao lançar" também criam previstos, pela mesma regra. O cliente não manda o marcador.
3. **Só uma ação desliga o marcador: Confirmar.** A pessoa informa **valor, data e status** (já preenchidos e alteráveis; o
   status vem da lista do servidor e a data não pode ser futura). O servidor tira o marcador, grava a versão e devolve como
   **Desfazer** uma restauração.
4. **Cada aba recebe do servidor o que deve mostrar.** Futuros traz todos os previstos, com os vencidos em **Atrasados** (como
   hoje) e o resumo (a pagar, a receber, atrasados) já calculado. O Histórico só traz o que já aconteceu (nunca um previsto), com cada linha resolvida pelas regras da data do evento. O front não olha
   status nem `statusFuturo`.
5. **Lançamentos antigos ficam como estão.** Os que já estão gravados com `PREVISTO`, `TRANSFERINDO` ou `CREDITANDO` continuam
   em Futuros (os do passado em Atrasados) até alguém tratar. O servidor os reconhece por uma única função; **nada é regravado**
   no Dynamo e não há migração em massa. O texto do status deles continua aparecendo como está gravado.
6. **O status é uma lista do backend, com vigência.** O app não cria nem edita status nem as caixas `statusReal`, `afetaSaldoBanco` e
   `statusFuturo`; `salvarListas` continua gravando o resto (bancos, cores, aparência, ordens), e o servidor ignora qualquer mudança em
   status. O front deixa de ler `statusFuturo`, `statusReal` e `afetaSaldoBanco`. O catálogo guarda **períodos** (`desde`, `ate`): nenhum
   status sai dele, e o que deixa de valer só deixa de ser **oferecido** a partir de uma data. A **data do corte** é o dia em que este
   item entra em produção. Ao gravar, o servidor aceita só um status válido **para a data do evento**, e só quando o que muda mexe em saldo
   (valor, tipo, banco, data, status, oculto); descrição, nota e tags são livres.
7. **O aviso do Telegram** passa a seguir o marcador (e o mesmo critério de atraso), em vez do status.
8. **O Histórico sai resolvido pela data do evento**, inclusive os filtros: cada linha com o rótulo e a cor da época, a lista de filtros de
   status do período que se está vendo (em 2019, `CREDITO IN` e `CREDITO EX`; em 2027, o que existir) e o resumo (quantidade e total) do
   filtro inteiro, tudo calculado no servidor. A base vai de 2016 em diante.

## Limites (o que fica de fora)

- **Saldo, fechamento e totais** são do item 40, e **faturas e cartões**, do item 33 fatiado. Eles vêm **antes**: quando este item
  começar, o navegador já não soma nada com status, e o previsto só precisa ficar fora da conta que o **servidor** faz.
- **O desmembramento do `PAGO`** (tipo e meio: recebimento, débito, pix, drex, cripto), o destino do **`CONTAS`** e do
  `excluirDoTotal`, e o papel de **Admin** ficam para depois.
- Não cria chave, índice nem item novo no DynamoDB: só **um atributo opcional** no lançamento (e nas versões do histórico, que
  guardam o lançamento inteiro). O nome e o tipo dependem da aprovação do dono (princípio 3 da arquitetura).
- Não muda a trava de mês conferido nem a de mês fechado por data (item 38): confirmar um previsto passa pelas duas.
- Não muda o aviso por Telegram além do gatilho (`avisos.md`).

## Custo (vai nas demos que faltam)

- Confirmar custa três toques no celular (a linha, Confirmar, Confirmar na janela), mais um por campo trocado.
- Quem esquece de confirmar deixa o lançamento em Atrasados e **fora do saldo**: o saldo pode destoar do extrato até alguém
  confirmar (o aviso do Telegram continua).
- O Histórico deixa de mostrar previstos, e o filtro "Incluir eventos futuros" da busca some.
- Prever e Estender precisam achar os previstos existentes (duplicados e última ocorrência da série) pela lista de Futuros, não
  mais pelos 1000 lançamentos carregados.
- A lista de Futuros lê o espaço inteiro a cada chamada (sem chave nova). Se pesar, um índice é decisão do dono.

## Pronto quando

- O servidor liga o marcador pela data, rejeita o marcador vindo do cliente, entrega a lista de status e os dados de Futuros e
  Histórico prontos, confirma com valor, data e status (passando pelas travas) e devolve o Desfazer, tudo com testes que tentam
  forjar cada campo.
- O front não contém nome de status, `ehFuturo`, `afetaSaldo`, `ehStatusReal`, `ehCredito` nem `statusFuturo`, e **um teste de CI
  falha se algum deles voltar** a `web/`.
- A aba Futuros usa só a API nova e a janela de confirmar da demo aprovada.
- O aviso do Telegram segue o marcador, a tela de Listas só exibe a lista de status, e os lançamentos antigos continuam
  funcionando sem migração.
- `modelo-de-dados`, `security.md`, `padrao-de-telas.md`, `avisos.md`, `eventos-futuros.md` e o BACKLOG ficam em dia na
  mesma entrega.
