# Marcador de conferência nas compras do cartão

Aprovado pelo dono em 2026-11-27. Demo aprovada: [`demo-cartao-conferencia.html`](demo-cartao-conferencia.html) (nesta pasta).

## Para quê

O dono confere as compras da fatura uma a uma com o app do banco e se perde nas que já viu. A marca é só orientação: não dispara ação, não altera lançamento, não é gravada.

## Onde

Aba Cartão de crédito, painel do cartão aberto, seção **Compras** (`painelCartaoHtml`, linhas `.fx-rec`).

## Comportamento

- Fora da seleção a lista fica como hoje, sem círculo. No cabeçalho de Compras, à direita do título, há o link **Selecionar**, com o estilo do `.hist-sel` do Histórico.
- Tocar em **Selecionar** liga o modo: aparece o círculo à direita do valor (`.lote-ck`, o mesmo do Histórico) e o link vira **Selecionados · N**.
- No modo seleção, tocar em qualquer parte da linha marca ou desmarca. As ações de ajuste de fatura ("Passar para a próxima" e as outras) não abrem.
- Tocar em **Selecionados** limpa todas as marcas e desliga o modo.
- Trocar de fatura ou de cartão, fechar o painel, sair da aba ou recarregar zera as marcas e desliga o modo.
- Compra marcada fica apagada (opacidade .5) com o círculo verde e ✓. Sem riscar: o riscado já significa "foi para outra fatura".
- Todas as compras da lista podem ser marcadas, inclusive as que estão `fora` da fatura.

## Fora do escopo

- Nenhuma gravação: nem localStorage, nem servidor, nem backend, nem DynamoDB.
- Sem barra de ação, sem "marcar todas", sem toque longo.
- O Histórico e a seleção em lote dele não mudam.

## Acessibilidade

O círculo tem `role="checkbox"` com `aria-checked`, como no Histórico. O estado é dado também por texto e ✓, não só pela cor. O círculo é acionável por teclado.

## Para quem implementa

- A demo é um recorte isolado da seção Compras, não parte da v5 da tela. Gere `.ai/demos/demo-cartao-v6-conferencia.html` a partir de [`demo-cartao-v5-aviso-canal.html`](../../demos/demo-cartao-v5-aviso-canal.html), com este comportamento, e atualize o `padrao-de-telas.md` e o resumo do `guardrails.md`.
- A marca é estado em memória ligado ao id do lançamento, como o `selecaoHist`, e sobrevive a uma nova renderização da mesma fatura.
- API: sem rotas novas.
