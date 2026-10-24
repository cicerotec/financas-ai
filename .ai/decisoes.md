# Decisões de produto fechadas e descartadas

Decisões pequenas que não pertencem a um documento de assunto. As de arquitetura estão em
[`arquitetura.md`](arquitetura.md), as de permissão em [`security.md`](security.md) e as de hospedagem em
[`hospedagem.md`](hospedagem.md).

## Histórico de versões: a versão só nasce na alteração (2026-10-23)
Fechado com o dono; o raciocínio está em [`specs/historico/plan.md`](specs/historico/plan.md) (decisão 8).
- **Criar e importar `id` novo não gravam versão.** Um lançamento é um lançamento; quem criou e quando já estão nele.
- **A versão 1 nasce na primeira edição, exclusão ou sobrescrita**, com o estado de antes, junto da versão 2. Na tela ela
  chama-se **"Original"** (só o texto; o valor `insert` no banco não muda).
- Importar um `id` que já existe é sobrescrita e continua gravando versão.
- **Exceção (escolhida pelo dono):** a criação ou importação que exigiu confirmação de mês fechado grava a versão 1 com a confirmação.
- **Ocultar** continua sendo edição e gerando versão. O que ocultar significa para o saldo fica para o item 40 do BACKLOG.
- As versões `insert` de importações anteriores ficam; o dono pode apagá-las no DynamoDB, só as de lançamentos nunca alterados.
- Perde-se o rastro de quem importou e quando. Não cria chave nem campo no DynamoDB.

## Futuros, status e números no servidor (2026-10-18; decididas, nada implementado)
O raciocínio completo está em [`arquitetura.md`](arquitetura.md), nas seções "Saldo e totais calculados no servidor" e
"Futuros, status e saldo no servidor". Resumo do que foi fechado com o dono:
- **O front não calcula verdade.** Mostra, e no máximo calcula para exibir (Tendências). O servidor rejeita número
  derivado vindo do cliente. Motivo: o navegador fica no computador de quem usa, e qualquer um pode trocar a conta.
- **Marcador `previsto` no lançamento** no lugar dos status `PREVISTO`, `TRANSFERINDO` e `CREDITANDO`. O lançamento
  futuro tem o status real; o marcador só o tira do saldo e do total do Histórico até ser confirmado. **Campo novo no
  Dynamo, nome a aprovar.** Foi descartado manter o status `PREVISTO` (a tela de Futuros não mostraria o status real) e
  descartado ter data de congelamento (a tela não precisa; o servidor entrega cada aba pronta).
- **Quem liga o marcador (2026-10-24):** o servidor, em todo lançamento com data depois de hoje; o Prever e o "Repetir ao
  lançar" também criam previstos.
- **Atrasados (2026-10-24):** os previstos vencidos **continuam em Atrasados, dentro de Futuros**, como hoje. O Histórico só
  mostra o que já aconteceu (nunca um previsto). O mesmo critério alimenta o aviso do Telegram. Saem do front o resumo "A pagar e A receber"
  e o saldo por mês de Futuros que ele soma hoje; o servidor manda o que for mostrado.
- **Confirmar (demo aprovada em 2026-10-24):** gesto A, com janela de **Valor**, **Data** e **Status**, cada um já preenchido
  e alterável. Textos: **Confirmar**, "Já paguei" (saída) e "Já recebi" (entrada). Botão verde na tabela do computador.
  Aviso de 6 s com Desfazer. Em mês fechado a data sugerida é hoje. O gesto B (direto, um toque) foi descartado.
- **Status (2026-10-24, regra do dono): o front não trata status de forma nenhuma.** O backend manda a lista (`["PAGO",
  "CREDITO"]` no corte; no futuro `PIX`, `DREX`, `DEBITO`...); o front só exibe a lista, guarda a escolha e a devolve como veio
  (o texto da opção). A lista fixa de `app.js:5` deixa de existir e um teste de CI barra nomes de status em `web/`. O servidor
  manda pronto o que o front deduzia: `previsto`, `bancos[].cartao`, `excluirDoTotal`, a fatura efetiva. `CREDITO IN` e
  `CREDITO EX` não entram na lista de escolha (o `faturaAjuste` os substitui) e só aparecem, como texto, nos registros antigos.
- **Status, meios e o passado (2026-10-24):** **não há data de corte global**; cada coisa tem a sua vigência. O catálogo de status tem uma entrada por nome
  e a marca **selecionável** (nenhum sai dele); `desde` e `ate` só sugerem a lista de uma data. **O passado é do usuário**: qualquer status selecionável
  vale em qualquer data (um `PAGO` de 2018 pode virar `PIX`), por um registro por vez pelo app; a trava de data segue o **efeito no saldo** (`PAGO` para `PIX`
  não é travado) e o histórico de versões torna tudo reversível. A **troca em massa é só do admin** (item 44). O status é mostrado como **forma de pagamento**
  (`PIX`, `DEBITO`, `DREX`, `CRIPTO` são formas; o saldo mora no meio). O `banco` é mostrado como **meio de pagamento** por um de-para em `domain/`, sem mudar o
  Dynamo; **o dono cria e desativa meios** pelo app (item 43). O dono **não criou status além dos sete padrão**. O Histórico sai do servidor pronto: rótulo e cor, filtros
  de status **do uso no período** e resumo do filtro inteiro. A base vai de 2016 em diante, então o item 40 exige um teste de volume de 10 anos. **O `PREVISTO`
  antigo continua previsto**, em Atrasados.
- **Status e cores de status:** o app não cria nem edita status. A cor por status continua como está. `salvarListas` segue
  gravando bancos, cores, aparência e ordens; o servidor ignora o que mude em status.
- **Ordem do trabalho (2026-10-24):** primeiro o item 40 (saldo, fechamento e conferência no servidor), depois o 33a (faturas do
  cartão) e por último o 41 (Futuros e status). O front só deixa de tratar status quando para de somar. O 33 é fatiado: a fatia b
  (cartão como entidade própria, trava da fatura conferida, migração do `faturaAjuste`) fica para depois. Specs em
  `specs/saldo-servidor/`, `specs/fatura-servidor/` e `specs/futuros-previsto/`.
- **Lançamentos antigos** com os três status antigos **ficam como estão** (eram só previsões): continuam em Futuros, os do
  passado em Atrasados, até alguém tratar (2026-10-24). Sem migração em massa; nada é regravado. Ao confirmar um deles, a
  janela abre com o primeiro status da lista do servidor escolhido (hoje `PAGO`).
- **`CREDITO IN` e `CREDITO EX`** já estão resolvidos pelo `faturaAjuste`. O **`PAGO`** será desmembrado depois, pela
  junção de tipo e meio (entrada: recebimento, pix, drex, cripto; saída: débito, pix, drex, cripto). O **`CONTAS`**
  (transferência entre bancos, `transferParId`) entra nesse desmembramento, fora de Futuros.
- **Admin** é um complemento ao dono (poderes reduzidos em tela, acesso total internamente); não é decidido agora. Poder
  de mexer em catálogo de status e módulos (PIX, DREX, cripto) seria dele no futuro.
- **Em aberto para o spec:** o nome do campo (proposta `previsto`, a aprovar) e como o servidor encontra todos os
  previstos sem depender dos 1000 lançamentos mais recentes; o que o Prever faz com o "Status das cópias".

## Outras decisões
- Descartado: bloquear duplicidade por conteúdo (a mesma compra pode ter outro horário e outra descrição).
- Atualização da tela por gatilho (foco da aba, troca de aba, antes de salvar, botão Atualizar, checagem lenta de 3 a
  5 min). **Sem temporizador curto** (15 s foi descartado).
