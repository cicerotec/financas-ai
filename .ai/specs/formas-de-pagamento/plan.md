# Plan: PIX, DEBITO, DINHEIRO e TED

Segue o [`intent.md`](intent.md); o roteiro verificável está no [`spec.md`](spec.md).

**Estado:** aprovado em 2026-11-02; fase A implementada.

## Decisões

1. **Quatro entradas novas no catálogo**, todas `Selecionavel` e `ContaNoSaldo: true`, sem `ForaDoTotal`, sem `Credito` e sem `AjustePadrao`: iguais
   ao `PAGO` no que o servidor calcula. Por isso a trava de data (efeito no saldo), a cascata, a conferência e o resumo do mês não mudam.
2. **Sem restrição de tipo.** Não há campo `Tipos` no catálogo nem validação por tipo: todas valem para entrada e saída (estorno no débito, estorno
   no cartão como entrada de `CREDITO`, salário por TED).
3. **Ordem do catálogo = ordem de uso (decisão do dono, 2026-11-02):** `PIX`, `DEBITO`, `CREDITO`, `DINHEIRO`, `TED`, `PAGO`, depois os não
   selecionáveis (`CONTAS`, `CREDITO IN`, `CREDITO EX`, `PREVISTO`, `TRANSFERINDO`, `CREDITANDO`). O primeiro da lista é o padrão: passa a ser o `PIX` (era
   o `PAGO`) em Lançar, no Confirmar de um antigo e no "volta ao primeiro". O `PAGO` fica selecionável no fim, como forma genérica.
4. **Cores padrão** (a pessoa troca em Listas): PIX `#1D9E75`, DEBITO `#D4537E`, DINHEIRO `#EF9F27`, TED `#5DCAA5`. *A confirmar com o dono ao ver na tela.*
5. **Nenhuma rota, ação de autorização ou tela nova.** `GET /status` e `GET /config/listas` já mandam a lista do catálogo; o front só a exibe
   (`web/sem-status.test.js` segue valendo e nenhum nome de status entra em `web/`).
6. **O status do ajuste da conferência não pode ser "o primeiro da lista":** com o `PIX` na frente, o ajuste viraria um PIX. `domain.StatusGenerico` (`PAGO`) fixa o status dos lançamentos criados pelo servidor.
7. **A base do cache de saldos muda uma vez** (ela lista os status que contam no saldo): os caches dos meses encerrados são recalculados e regravados na primeira leitura, sem perda de dado.
8. **Prever, Estender e Repetir** copiam o status do original; sem mudança.

## Fases

Uma só: **backend**. Catálogo, ordem, cores, testes e a conferência dos testes que dependem do primeiro da lista. Sem demo, porque nenhuma tela muda
de desenho.

## Riscos

- **O padrão muda de `PAGO` para `PIX`:** os testes que dependem do primeiro da lista (por exemplo, o do Confirmar de um antigo, que abre com o primeiro
  status) precisam ser conferidos. Os que usam `PAGO` como exemplo de lançamento novo continuam válidos, porque ele segue selecionável.
- **Conferir nas telas:** o nome mais longo (`DINHEIRO`, 8 letras) nas colunas do Histórico e de Futuros, no celular (≈ 390 px) e na janela larga, nos dois temas.
- **Importação de backup antigo** com `PAGO`: continua válida (R7 do `futuros-previsto`).
