# Marcador de conferência nas compras do cartão

Aprovado pelo dono em 2026-11-27. Demo aprovada: [`demo-cartao-v6-conferencia.html`](../../demos/demo-cartao-v6-conferencia.html), que substitui o recorte [`demo-cartao-conferencia.html`](demo-cartao-conferencia.html) (nesta pasta).

## Para quê

O dono confere as compras da fatura uma a uma com o app do banco e se perde nas que já viu. A marca é só orientação: não dispara ação, não altera lançamento, não é gravada.

## Onde

Aba Cartão de crédito, painel do cartão aberto, seção **Compras** (`painelCartaoHtml`, linhas `.fx-rec`).

## Comportamento

- Fora da seleção a lista fica como hoje, sem círculo. No cabeçalho de Compras, à direita do título, há o link **Selecionar**, com o estilo do `.hist-sel` do Histórico.
- Tocar em **Selecionar** liga o modo: aparece o círculo à direita do valor (`.lote-ck`, o mesmo do Histórico) e o link vira **Selecionados · N**.
- No modo seleção, tocar em qualquer parte da linha marca ou desmarca. As ações de ajuste de fatura ("Passar para a próxima" e as outras) não abrem.
- Compra marcada muda só o círculo: verde com ✓. Sem opacidade, sem riscar e sem fundo colorido.
- Todas as compras da lista podem ser marcadas, inclusive as que estão `fora` da fatura, que mantêm a aparência de hoje.
- Com **Compras** recolhida ou sem nenhuma compra o link não aparece; ao abrir, volta com a contagem.

## O que apaga e o que mantém as marcas

Revisto pelo dono em 2026-11-27. Regra: **mudou algo na aba Cartão de crédito, apaga as marcas e desliga o modo; não mudou, mantém.** A exceção é tocar em **Selecionados · N**, que apaga sempre.

| Apaga | Mantém |
|---|---|
| Tocar em **Selecionados · N** | Recolher ou abrir **Compras** e qualquer outra seção do painel |
| Tocar em **Atualizar ⟳** | Fechar o painel e reabrir o mesmo cartão, na mesma fatura e sem mudança |
| Trocar de fatura ou de cartão | Sair da aba e voltar sem mudança |
| Salvar um campo, conferir ou desfazer a conferência, corrigir um valor estimado, registrar a diferença, adicionar ou remover cartão | Abrir a janela de um campo e cancelar |
| Voltar à fatura com dado diferente (por exemplo, compra editada no Histórico) | Marcar e desmarcar |
| Recarregar a página | |

O painel reaberto mostra a fatura atual, como hoje: se as marcas eram de outra fatura, elas são apagadas.

## Fora do escopo

- Nenhuma gravação: nem localStorage, nem servidor, nem backend, nem DynamoDB.
- Sem barra de ação, sem "marcar todas", sem toque longo.
- O Histórico e a seleção em lote dele não mudam.

## Acessibilidade

Como no Histórico: no modo seleção a **linha** tem `role="checkbox"`, `aria-checked` e foco (Enter e Espaço marcam), e o círculo é só visual (`aria-hidden`). O estado é dado pelo ✓, não só pela cor.

## Para quem implementa

- A tela segue a [`demo-cartao-v6-conferencia.html`](../../demos/demo-cartao-v6-conferencia.html), gerada da v5; o `padrao-de-telas.md` e o resumo do `guardrails.md` já a registram. Implementado em `web/` em 2026-11-27 (teste: `web/cartao-conferencia.test.js`).
- A marca é estado em memória ligado ao id do lançamento, como o `selecaoHist`, e só sobrevive a uma nova renderização se o cartão, a fatura e os dados forem os mesmos.
- A demo `demo-cartao-conferencia.html` desta pasta é o recorte aprovado primeiro; onde divergir deste texto (opacidade, acessibilidade), vale o texto e a v6.
- API: sem rotas novas.
