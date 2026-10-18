# Decisões de produto fechadas e descartadas

Decisões pequenas que não pertencem a um documento de assunto. As de arquitetura estão em
[`arquitetura.md`](arquitetura.md), as de permissão em [`security.md`](security.md) e as de hospedagem em
[`hospedagem.md`](hospedagem.md).

## Futuros, status e números no servidor (2026-10-18; decididas, nada implementado)
O raciocínio completo está em [`arquitetura.md`](arquitetura.md), nas seções "Saldo e totais calculados no servidor" e
"Futuros, status e saldo no servidor". Resumo do que foi fechado com o dono:
- **O front não calcula verdade.** Mostra, e no máximo calcula para exibir (Tendências). O servidor rejeita número
  derivado vindo do cliente. Motivo: o navegador fica no computador de quem usa, e qualquer um pode trocar a conta.
- **Marcador `previsto` no lançamento** no lugar dos status `PREVISTO`, `TRANSFERINDO` e `CREDITANDO`. O lançamento
  futuro tem o status real; o marcador só o tira do saldo e do total do Histórico até ser confirmado. **Campo novo no
  Dynamo, nome a aprovar.** Foi descartado manter o status `PREVISTO` (a tela de Futuros não mostraria o status real) e
  descartado ter data de congelamento (a tela não precisa; o servidor entrega cada aba pronta).
- **Atraso** é o marcador ainda ligado depois da data, e só serve ao aviso do Telegram. Saem do front o resumo "A pagar e
  A receber" e o saldo por mês de Futuros que ele soma hoje; o servidor manda o que for mostrado.
- **Status e cores de status:** o catálogo é do backend, o app não cria nem edita status. A cor por status continua como
  está. `salvarListas` segue gravando bancos, cores, aparência e ordens; o servidor ignora o que mude em status.
- **Lançamentos antigos** com os três status antigos contam como marcador ligado até serem editados; sem migração em
  massa e sem gravar nada antes de o dono aprovar.
- **`CREDITO IN` e `CREDITO EX`** já estão resolvidos pelo `faturaAjuste`. O **`PAGO`** será desmembrado depois, pela
  junção de tipo e meio (entrada: recebimento, pix, drex, cripto; saída: débito, pix, drex, cripto). O **`CONTAS`**
  (transferência entre bancos, `transferParId`) entra nesse desmembramento, fora de Futuros.
- **Admin** é um complemento ao dono (poderes reduzidos em tela, acesso total internamente); não é decidido agora. Poder
  de mexer em catálogo de status e módulos (PIX, DREX, cripto) seria dele no futuro.
- **Em aberto para o spec:** quem liga o marcador na criação; como os previstos vencidos aparecem no Histórico; o gesto
  de confirmar; o nome do campo.

## Outras decisões
- Descartado: bloquear duplicidade por conteúdo (a mesma compra pode ter outro horário e outra descrição).
- Atualização da tela por gatilho (foco da aba, troca de aba, antes de salvar, botão Atualizar, checagem lenta de 3 a
  5 min). **Sem temporizador curto** (15 s foi descartado).
