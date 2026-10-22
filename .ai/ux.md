# UX

Origem: observar a esposa usando o app (sem explicar nada) e anotar onde ela hesita. Repetir a cada rodada.
- **Barra inferior no celular: descartada** (2026-10-15). No C6 a barra some em toda tela empilhada e no teclado; aqui a
  navegação é plana. Resolvido com os atalhos Histórico e Saldos antes do ⋮ (`padrao-de-telas.md`, "Menu do app").
- **Alvos de toque** de Cartão e Saldos (`font-size:12px; padding:4px 10px`) abaixo de 44px; trocar por linha de ações.
- **Acessibilidade:** abas sem `role="tablist"`/`aria-selected`; `‹ ›` das janelas de faturas são `<span>`, não
  funcionam por teclado.
- **Código:** muito `style` inline; `web/index.html` passa de 3.200 linhas (separar CSS e JS).
- **Seletor de tags:** o do Lançar e o dos filtros duplicam lógica; unificar em um componente (ver "Filtro único e lista da consulta", item 42 do backlog).
- **Bug (item 25): combo de bancos reseta após copiar pelo Histórico.** Ao copiar um lançamento pelo Histórico e
  lançar o registro, o combo de bancos volta sozinho para o primeiro elemento da lista, e isso pode gerar erro no
  cadastro (o lançamento sai no banco errado sem a pessoa perceber). Investigar onde o formulário de Lançar repopula
  ou reinicia os combos (cópia, salvar, atualização da tela por `seq`) e manter o banco escolhido.
  **Corrigido:** `renderListas()` refazia os `<option>` dos combos (Status, Banco e bancos da transferência) e o
  navegador voltava ao primeiro; agora a escolha atual é guardada e devolvida depois de refazer a lista.
- **Lançar redesenhado (2026-10-19):** o formulário passou de 17 controles à vista para 4 no celular; ver "Lançar (tela)" em `padrao-de-telas.md`. Observar o uso real: se Nota ou Tags forem usadas em todo lançamento, o toque a mais em "Mais" e "Tags" pode pedir ajuste.
- **Lançar, a testar no ambiente dev (PR 146):** salvar, editar, copiar do Histórico, IA e Transferência com data e hora só tiveram o visual conferido, não o fluxo. Conferir também o × das tags no computador e o "Mais" abrindo sozinho ao editar um lançamento com nota.
- **Histórico redesenhado (2026-10-19):** filtros numa janela, linha como a de Futuros e nota em ícone; ver "Histórico (tela, lista de lançamentos)" em `padrao-de-telas.md`. A testar no ambiente dev: busca por texto com dados reais, ações (editar, copiar, prever, ocultar, excluir), mudança de mês, faixa fixa no computador e o pior caso (valor de 7 dígitos, banco comprido).
- **Histórico por período (2026-10-20):** o mês deu lugar a 30d · 90d · 180d · 📅 porque, na virada do mês, o lançamento de ontem sumia da lista (a esposa estranhou também a busca "em todo o histórico", que escondia o período). A testar no dev: busca no período e "Buscar em tudo" com dados reais, seletor de mês no navegador e no celular, e o tempo da varredura em 180d e em tudo.
- **Busca por tag e sem acento (2026-10-21):** a busca do Histórico passou a olhar também as tags e a ignorar acento e maiúsculas. A testar no dev com dados reais: tags com espaço ou acento, e o termo digitado com `#`.
- **Botão voltar (2026-10-19):** o voltar do navegador e do celular fechava o app, porque nada criava entrada no histórico. Agora cada aba e cada janela ou tela cheia é uma entrada; ver "Botão voltar" em `padrao-de-telas.md`. **A testar no ambiente dev, no celular e instalado:** Histórico → Editar → voltar; salvar e cancelar edição; voltar com texto digitado (pergunta); excluir e voltar com o Desfazer aberto; Prever com janela de rodas; Tendências (Editar consulta, modo avançado, janela de tags); versões em tela cheia; F5 em cada aba.
- **Observado e já resolvido:** salvar sem resposta, transferência sem data, Enter sem efeito na busca, tudo na tela
  em Histórico, X de status apagando sem aviso.

## Filtro único e lista da consulta (2026-10-22; item 42 do backlog)
Conversa com o dono depois de a Tendências ganhar a consulta com E, OU e NÃO. **Nada disto está implementado**; é o roteiro.

**Por que.** Tendências, Histórico e Futuros usam tags (e o Histórico também banco, status e combos), cada um com o seu filtro: três cópias do mesmo assunto. Os números e as
consultas divergem (a Tendências tem tags avançadas, o Histórico só "todas" ou "qualquer uma"; ocultos e status contam diferente). Papéis, ditos pelo dono: **o Histórico é o saldo
real** (ocultar tira do saldo); a **Tendências é uma simulação** para brincar com os números e fazer projeções, nunca vale como saldo e nunca grava no backend.

**Decisões.**
- **A lista dos lançamentos da consulta fica na Tendências** (não "jogar" a consulta no Histórico, que ainda não entende tags avançadas) e será testada em produção. É só
  leitura: o risco é baixo, porque não grava nada. Passar pelo `dev` (dados fictícios) antes, porque é visual.
- **Filtro compartilhado como "classe própria com tela própria".** A tela é uma **janela** aberta por cima de quem chama (não uma aba, para não perder o contexto nem quebrar o
  botão voltar). Quem chama diz o que quer ("só tags e combos", "tags, bancos e status") e recebe o estado ao tocar em "Pronto".
- **Duas partes:** lógica pura (estado do filtro e "este lançamento casa?", testada; é a `consulta-tags.js` crescendo) e a janela, que só desenha. **Cada tela é dona do seu
  estado** (o Histórico lembra os filtros dele; a Tendências, os dela). **As opções** (tags, bancos, status) vêm da tela que chama (status é do backend). **Só filtra o que já está
  na tela:** nunca envia nada ao servidor.
- **Ordem:** a) lista na Tendências; b) extrair o filtro de tags; c) Histórico adota (tela do saldo real, demo e cuidado); d) Futuros avalia (ler antes o que filtra hoje);
  e) "Abrir no Histórico" a partir da Tendências, para editar ou ocultar (depende do c).

**Lista da consulta na Tendências (regras propostas).**
- Janela aberta por um botão ao lado do total ("Ver os N lançamentos"); linhas como as do Histórico (descrição, data, status, banco, tags, valor com sinal e cor), **sem ações**.
- Tocar numa barra do mês filtra a lista para aquele mês; no topo, a **soma da lista**, que bate com o total do gráfico na visão escolhida (conferência dos números).
- A lista respeita a visão (Gastos só saídas, Receitas só entradas, Resultado as duas) e a mesma base do gráfico (ocultos e futuros fora), então pode mostrar menos lançamentos que a
  busca do Histórico para as mesmas tags; a janela diz isso.
- Limite de linhas desenhadas (proposta: 200, com "mostrando 200 de N, filtre por mês") para não pesar no celular. A janela entra em `camadasAbertas()` e `fecharCamadaTopo()`.

**Observados na Tendências, ainda sem tratar.** Gráfico parado nos meses antigos ao girar o celular ou redimensionar a janela (o rolar até o fim só roda quando a largura passa a
valer); a visão Gastos/Receitas/Resultado não é guardada ao recarregar; o gráfico longo só enxerga os 1000 lançamentos mais recentes carregados.
