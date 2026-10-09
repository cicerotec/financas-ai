# Padrão de telas (celular primeiro, com versão para computador)

Aprovado pelo dono em 2026-10-08, em duas demos. **Toda tela nova ou alterada segue este padrão e vem com demo antes do
código** (regra no `BACKLOG.md`, seção "Telas"). Este guia existe para que qualquer pessoa ou IA reproduza o mesmo
resultado sem ter visto a conversa.

## Fonte da verdade
Os arquivos abaixo são páginas completas e independentes (abra no navegador; no celular, ou numa janela larga para a
visão Computador). Quando o guia e uma demo discordarem, vale a demo.

| Demo | Arquivo | O que mostra |
|---|---|---|
| Repetir e avisar | [`demos/demo-repeticao-e-aviso.html`](demos/demo-repeticao-e-aviso.html) | linha-resumo que abre tela cheia, lista que acumula, escolha única, interruptor, janela com rodas, contador de limite |
| Futuros | [`demos/demo-futuros.html`](demos/demo-futuros.html) | uma linha por evento, ações numa janela de baixo, resumo em faixa, e a **versão para computador** (tabela) |

## Princípios
1. **Celular primeiro.** O dono usa o app no Android. Desenhe a largura de 360 a 400 px e só depois amplie.
2. **Pouco à vista, detalhe sob demanda.** Uma linha com o resumo; o detalhe abre em tela cheia ou em janela de baixo.
   Nunca deixe mais de dois ou três controles à vista num formulário (hoje o Lançar passou de 17 e ficou poluído).
3. **Padrões nativos do celular**: lista que acumula com check redondo, rodas de rolar, interruptor, janela de baixo.
4. **Os limites aparecem antes de serem atingidos** ("2 de 6 prazos"), e o erro aparece na própria tela, sem fechar.
5. **O mesmo dado e as mesmas regras nos dois tamanhos.** Só o desenho muda com a largura da janela.
6. **Diga o custo.** Quando uma proposta simplifica e faz perder algo, escreva o que se perde na própria demo.

## Tokens
Use as variáveis que o app já tem (`web/index.html`, bloco `:root`). Onde a demo usa outro nome, o mapeamento é este:

| Demo | App | Claro (demo) | Escuro (demo) | Uso |
|---|---|---|---|---|
| `--bg` | `--bg` | `#f1f0ec` | `#121513` | fundo da página |
| `--card` | `--card` | `#ffffff` | `#1f2421` | cartões, linhas, janelas |
| `--fg` | `--text` | `#1d2220` | `#e8ebe7` | texto |
| `--muted` | `--muted` | `#5c645e` | `#a3aaa4` | texto de apoio (13 px) |
| `--line` | `--border` | `#e4e2dc` | `#2e3430` | divisórias de 1 px |
| `--accent` | `--accent` | `#0f6b4f` | `#5fc9a0` | check marcado, interruptor ligado, número da roda |
| `--onaccent` | `--accent-ink` | `#ffffff` | `#08231a` | texto sobre o acento |
| `--danger` | `--danger` | `#a3321f` | `#f08a76` | erro, atrasado, valor de saída, "Excluir" |
| `--ok` | `--income` | `#1d6b45` | `#6fd6a4` | valor de entrada |
| `--faint` | (novo) | `#8c938d` | `#7b827c` | datas, legendas, itens fora da seleção |
| `--ck` | (novo) | `#d3d6d2` | `#3b413d` | check vazio e interruptor desligado |
| `--accent-bg` | (novo) | `#dcefe5` | `#1c3a2e` | faixa de destaque da roda |

A demo usa `--bg2` (`#e9e8e3` claro, `#0c0f0d` escuro) só para o fundo do quadro de celular que fica atrás dos cartões; no
app esse papel é do próprio `--bg`, e os cartões continuam em `--card`. Nada de `--bg2` no app.

Os três "novos" entram no `:root` do app, com os dois temas, no mesmo formato dos que já existem (claro em `:root`,
escuro em `prefers-color-scheme` com `:root:not([data-theme="light"])` e em `:root[data-theme="dark"]`). Nunca use cor
literal num componente.

## Medidas
| Elemento | Medida |
|---|---|
| Fonte | `system-ui`; 15 px corpo, 13 px apoio, 12 a 12,5 px legendas, 18 px título de tela |
| Alvo de toque | linha de lista com no mínimo **56 px**; botão de janela 52 px |
| Cartão | raio 16 px, sem borda, fundo `--card`, 10 px de espaço entre cartões |
| Linha de lista | padding 8 × 16 px, divisória de 1 px em `--line` entre linhas |
| Check redondo | 28 × 28 px; vazio em `--ck`; marcado em `--accent` com "✓" em `--onaccent` |
| Interruptor | 52 × 30 px (44 × 26 na faixa de filtros), bolinha branca, `role="switch"` |
| Janela de baixo | raio 22 px nos cantos de cima, fundo `--card`, sobre véu preto a 50% |
| Roda | item de 44 px; área de 132 px (3 itens); faixa de destaque no item do meio |
| Botões de janela | dois botões lado a lado, raio 14 px; Cancelar em `--ck`, OK em `--accent` |

## Padrões do celular
**1. Linha-resumo que abre tela cheia.** Cartão com título em negrito e, embaixo, o valor atual em `--muted`
(ex.: "Repetir / 3 vezes além deste, mensal"), seta `›` à direita. Tocar abre a tela cheia daquela escolha.

**2. Tela cheia.** Cabeçalho com seta de voltar (`←`) e o título; abaixo, cartões. Não há botão "Salvar": cada escolha
vale na hora e o resumo da linha de origem se atualiza.

**3. Lista que acumula (vários valores).** Cada opção é uma linha com `role="checkbox"` e o check redondo à direita.
Tocar marca e desmarca. Valores próprios (criados pela pessoa) entram na mesma lista, ordenados, e desmarcá-los os remove.

**4. Escolha única.** Igual à lista que acumula, com `role="radio"`: marcar um desmarca os outros.

**5. Interruptor no topo** liga e desliga o recurso inteiro. Desligado, a lista some e aparece uma frase curta explicando o
que acontece sem ele.

**6. "Outro..." com rodas.** Linha própria, depois da lista. Abre uma janela de baixo com título, **resumo ao vivo** em
cima das rodas (ex.: "2 semanas antes"), uma roda de número e, quando faz sentido, uma de unidade (dia ou semana; mês
ou ano). Botões Cancelar e OK. **Erro de validação aparece no lugar do resumo, em `--danger`, e o OK não fecha a janela.**
Mensagens do tipo "O máximo é 30 dias (4 semanas)." ou "Esse prazo já está na lista."

**7. Contador e limite.** O subtítulo da linha "Outro..." mostra "N de MÁX prazos". No limite, a linha troca para uma
frase em `--danger` ("Limite de 6 prazos. Desmarque um para adicionar outro.") e o toque deixa de abrir as rodas.

**8. Lista de eventos.** Uma linha por evento: à esquerda descrição em negrito e, embaixo, "data · status · banco ·
série"; à direita valor (saída em `--danger` com "-", entrada em `--ok` com "+") e, embaixo, o prazo ("hoje", "em 3
dias", "atrasado 3 dias", este último em `--danger` e negrito). Ícone de sino quando há aviso. Grupos com título por mês
e um grupo "Atrasados" no topo, em `--danger`. Evento oculto fica a 50% de opacidade.

**9. Ações numa janela de baixo.** Tocar na linha do evento abre a janela com descrição, valor, data e banco no topo e as
ações em lista (Editar, Copiar, Prever ou Estender série, Ocultar ou Mostrar, Excluir). "Excluir" em `--danger`. Nada de
fileira de botões dentro da linha.

**10. Resumo em faixa.** Três blocos lado a lado no topo (a pagar, a receber, atrasados), número em 16 px. Substitui o
texto explicativo longo.

**11. Aviso rápido (toast).** Faixa escura na base da tela, 2 segundos, para confirmar ("Oculto: saiu dos totais.").

## Padrão do computador (janela larga)
Mesma tela, mesmos dados, desenho de tabela.
- **Tabela de largura total** com cabeçalho cinza e uma linha por evento, em grade de colunas:
  `100px minmax(190px,1fr) 80px 90px 118px 110px 310px` = data, evento, banco, status, prazo, valor (alinhado à
  direita), ações. O espaçamento entre colunas é de 12 px. A tabela tem largura mínima de **1100 px** e rola na
  horizontal dentro do próprio quadro abaixo disso. (A soma das colunas fixas mais o `minmax` tem de caber na largura
  mínima, senão a coluna do nome colapsa e as colunas se sobrepõem.)
- **Container largo:** o conteúdo passa a até 1200 px. No app, o `main{max-width:640px}` é o que deixa a tela estreita
  no computador; a tela larga precisa de um modificador, e não de trocar o `main` para todas as telas.
- **Ações na própria linha**, no fim, em botões de texto pequenos, **apagadas (opacidade 0,25) até o mouse passar ou o
  foco entrar** (`:hover` e `:focus-within`, linha com `tabindex="0"`). Sem janela de baixo e sem toque extra.
- **Resumo e interruptor de ocultos** ficam no topo, como no celular; os totais continuam excluindo os ocultos.
- Quando existirem telas em largura média (tablet), tratar como a visão do celular até haver uma demo própria.

## Regras de conteúdo
- **Datas sempre com o ano**, `dd/mm/aaaa`, em tela e em mensagem. Moeda em pt-BR (`R$ 1.234,56`).
- Português com letra maiúscula só no começo da frase; verbos curtos nas ações ("Editar", "Estender série").
- Prazo relativo ao lado da data real, nunca no lugar dela.
- Todo limite (quantidade, dias, tamanho) aparece na tela antes de ser atingido e tem mensagem de erro própria.

## Acessibilidade e tema
- Controles com `role` e `aria-checked` corretos (`checkbox`, `radio`, `switch`) e `aria-label` quando não há texto.
- Alvos de toque de 56 px nas linhas; botões de janela com 52 px.
- Respeitar `prefers-reduced-motion` (a bolinha do interruptor e a seta animam só quando permitido).
- Tema claro e escuro **pelos tokens**, nunca por cor literal; testar os dois.

## Como reproduzir numa tela nova (receita)
1. Liste o que a tela precisa mostrar e decidir. O que não for essencial sai da vista (padrão 1).
2. Escolha os padrões desta lista; não invente um componente novo se um existente serve.
3. **Faça a demo** copiando a estrutura de uma das demos de referência (CSS e JS ficam num arquivo só). Publique como
   página com link que abre no celular (artefato privado); o widget do chat só aparece no desktop.
4. Inclua na demo o que se perde com a proposta e a visão "como está hoje" para comparar.
5. Espere o ok do dono. Só então implemente, com os tokens do app (tabela acima) e a lógica real por baixo.
6. Teste no navegador: celular (≈ 390 px) e janela larga (≥ 1280 px), tema claro e escuro, e os limites e os erros.

## Armadilhas já encontradas
- O navegador de teste do painel não gera o evento `scroll` quando a página não está sendo desenhada: para testar as
  rodas por script, ajuste `scrollTop` **e** dispare `dispatchEvent(new Event('scroll'))`. No celular real o evento
  dispara sozinho (a roda usa `scroll-snap` e lê a posição 90 ms depois de parar).
- Grade de colunas cuja soma passa da largura da tabela faz a coluna `1fr` colapsar e sobrepõe o texto.
- Data sem ano (`31/01`) é ambígua quando o intervalo passa de um ano.
- `\n` dentro de heredoc do shell no Windows se corrompe: edite arquivos com a ferramenta de edição.
