# Padrão de telas (celular primeiro, com versão para computador)

Aprovado pelo dono em 2026-10-08, em duas demos. **Toda tela nova ou alterada segue este padrão e vem com demo antes do
código** (regra em `.ai/guardrails.md`, seção "Telas"). Este guia existe para que qualquer pessoa ou IA reproduza o mesmo
resultado sem ter visto a conversa.

## Fonte da verdade
Os arquivos abaixo são páginas completas e independentes (abra no navegador; no celular, ou numa janela larga para a
visão Computador). Quando o guia e uma demo discordarem, vale a demo.

| Demo | Arquivo | O que mostra |
|---|---|---|
| Repetir e avisar | [`demos/demo-repeticao-e-aviso.html`](demos/demo-repeticao-e-aviso.html) | linha-resumo que abre tela cheia, lista que acumula, escolha única, interruptor, janela com rodas, contador de limite, status das cópias e o botão de criar (é a base do **Prever** e do **Estender**) |
| Futuros | [`demos/demo-futuros.html`](demos/demo-futuros.html) | uma linha por evento, ações numa janela de baixo, resumo com faixa de atraso, e a **versão para computador** (tabela) |
| Futuros por largura | [`demos/demo-futuros-largura.html`](demos/demo-futuros-largura.html) | **vale sobre a demo acima onde diferirem**: prazo embaixo do valor, descrição em até 2 linhas, "⋯" no celular, botões no computador e realce na cor do status |
| Resumo da Futuros | [`demos/demo-resumo-futuros.html`](demos/demo-resumo-futuros.html) | a comparação que decidiu o resumo (opção A), com valores extremos e larguras de 320 a 390 px |
| Total do Histórico | [`demos/demo-historico-total.html`](demos/demo-historico-total.html) | a soma da lista (renda menos gasto) à direita da linha do contador, na busca por texto e no mês aberto |
| Atalhos no cabeçalho | [`demos/demo-atalho-saldos.html`](demos/demo-atalho-saldos.html) | Histórico e Saldos entre o nome da tela e o ⋮, em 320, 360 e 390 px, com o "como está hoje" (aprovado em 2026-10-15) |
| Histórico de versões | [`demos/demo-historico-versoes.html`](demos/demo-historico-versoes.html) | busca por mês, lista de excluídos e alterados, versões com diff e Restaurar (desabilitado com o motivo), estados vazio e truncado, papéis dono e membro e a visão de computador (aprovado em 2026-10-16) |
| Confirmação de mês fechado | [`demos/demo-confirmacao-mes-fechado.html`](demos/demo-confirmacao-mes-fechado.html) | segunda janela ao editar/lançar em mês fechado, restaurar com aviso no cartão, importar com uma pergunta no fim, mês conferido sem confirmação (aprovado em 2026-10-17) |
| Excluir na hora | [`demos/demo-exclusao-imediata.html`](demos/demo-exclusao-imediata.html) | DELETE imediato com o item na lista até a resposta, recusas antes de sumir (conferido, mês fechado, rede), Desfazer por restauração e seu erro (aprovado em 2026-10-17) |
| Tendências por tags | [`demos/demo-tendencias-tags.html`](demos/demo-tendencias-tags.html) | grupos com E/NÃO e OU entre eles (padrão), "Modo avançado" com expressão livre (parênteses, E, OU, NÃO), barra do mês em duas partes e lista de outras tags com "Tirar (NÃO)" (implementada em 2026-10-18) |
| Tendências, layout | [`demos/demo-tendencias-layout.html`](demos/demo-tendencias-layout.html) | **vale sobre a demo acima onde diferirem (layout)**: computador com resultado à esquerda e escolhas à direita, celular com "Editar consulta" em tela cheia, modo avançado por cima, barra de resumo e tags em botões (implementada em 2026-10-18) |
| Tendências, visão | [`demos/demo-tendencias-visao.html`](demos/demo-tendencias-visao.html) | seletor "Gastos \| Receitas \| Resultado" no topo do resultado (Gastos é o padrão; Resultado é entradas menos saídas das tags escolhidas, **não** o saldo da conta), faixa fixa "Simulação" (não é saldo, nada é gravado, ocultos ficam de fora), barra que desce quando o resultado é negativo e "outras tags" pesadas pelo movimento no Resultado (implementada e aprovada em 2026-10-22) |
| Tendências, lista | [`demos/demo-tendencias-lista.html`](demos/demo-tendencias-lista.html) | **em avaliação, não aprovada**: janela só de leitura com os lançamentos da consulta (botão "Ver os N lançamentos" ou toque numa barra do mês), fichas de mês, soma que bate com o gráfico, limite de 200 linhas e o topo compacto do app. Só desenha: nada vai para o código antes da aprovação |
| Lançar, layout | [`demos/demo-lancar-layout.html`](demos/demo-lancar-layout.html) | Lançamento e Transferência como modos no topo, valor grande, Data/Banco/Status em linhas com seletor nativo, Tags em janela de baixo (celular) ou na coluna da direita (computador), "Mais" para nota, fatura, repetir e avisar, Salvar fixo embaixo (implementada em 2026-10-19) |
| Histórico, layout | [`demos/demo-historico-layout.html`](demos/demo-historico-layout.html) | Lista de lançamentos: mês e busca no topo (o topo e a busca foram trocados pela demo de período abaixo), Filtros numa janela, linha como a de Futuros agrupada por dia, nota em ícone, ações numa janela de baixo, grupos por mês com soma na busca e tabela no computador (implementada em 2026-10-19) |
| Histórico, período | [`demos/demo-historico-periodo.html`](demos/demo-historico-periodo.html) | **vale sobre a demo acima no topo da tela e na busca**: grupo 30d · 90d · 180d · 📅 (um mês) no lugar do ‹ mês ›, abre em 30d, a busca olha só o que está marcado e "Buscar em tudo" pede confirmação (implementada em 2026-10-20) |
| Botão voltar | [`demos/demo-voltar-navegador.html`](demos/demo-voltar-navegador.html) | O voltar do navegador, do Android e do app instalado fecha uma camada por vez: pilha do histórico ao vivo, Editar, Excluir com Desfazer fora da pilha, pergunta ao descartar texto digitado e Esc (aprovada e implementada em 2026-10-19) |
| Desfazer | [`demos/demo-desfazer.html`](demos/demo-desfazer.html)(demos/demo-desfazer.html) | janela de confirmação antes de excluir e de salvar uma edição, e depois o aviso de 6 s com "Desfazer"; erro ao efetivar e o que se perde (aprovado em 2026-10-14) |

## Princípios
1. **Celular primeiro.** O dono usa o app no Android. Desenhe a largura de 360 a 400 px e só depois amplie.
2. **Pouco à vista, detalhe sob demanda.** Uma linha com o resumo; o detalhe abre em tela cheia ou em janela de baixo.
   Nunca deixe mais de dois ou três controles à vista num formulário (o Lançar passou de 17 e foi redesenhado em 2026-10-19).
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
| `--danger-bg` | (novo) | `#f8e6e2` | `#3a1f1a` | fundo da faixa de atraso |

A demo usa `--bg2` (`#e9e8e3` claro, `#0c0f0d` escuro) só para o fundo do quadro de celular que fica atrás dos cartões; no
app esse papel é do próprio `--bg`, e os cartões continuam em `--card`. Nada de `--bg2` no app.

Os quatro "novos" entram no `:root` do app, com os dois temas, no mesmo formato dos que já existem (claro em `:root`,
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

## Menu do app (navegação entre telas)
Aprovado pelo dono em 2026-10-14. Presente no topo de todas as demos de `demos/` (bloco `menu-app:inicio` a `menu-app:fim`).
- **Celular e janela abaixo de 1160 px:** cabeçalho de uma linha, com o **nome da tela atual** em destaque (sem o nome do
  app: o app se chama só "Finanças", e isso aparece na aba do navegador e no ícone) e o botão **⋮** (alvo de 48 px). O ⋮ abre um menu suspenso com as telas agrupadas em **Dia a dia** (Lançar,
  Histórico, Futuros), **Acompanhar** (Cartão, Saldos, Tendências) e **Configurar** (Listas, Importar). A tela atual leva
  destaque e "✓". Toque fora do menu ou Esc fecha, e o foco volta ao ⋮.
- **Atalhos Histórico e Saldos** (aprovado em 2026-10-15, `demos/demo-atalho-saldos.html`): dois botões de 56 × 52 px
  entre o nome da tela e o ⋮, com ícone e rótulo de 11 px embaixo (ícone sozinho é ambíguo). Ficam **sempre no mesmo
  lugar**; o da tela atual ganha fundo `--accent-bg` e `aria-current="page"`. O título continua sendo a fonte da verdade
  sobre onde se está. Os dois também seguem dentro do ⋮. Cabeçalho de 56 px (era 52). Em 320 px sobram ~120 px para o
  nome da tela ("Tendências" cabe justo) e o nome do app, acima, ganha reticências.
- **Conta no menu** (aprovado em 2026-10-17, `demos/demo-menu-usuario.html`; `web/js/usuario.js`): o menu abre com um bloco da
  conta no topo (inicial, **nome**, e embaixo o espaço e o papel, "Família Silva · dono"/"membro"), depois as telas, e termina
  com **Sair**, separado por uma linha e em `--danger`. Com mais de um espaço, o bloco leva um **seletor nativo** "Espaço" no lugar
  da linha do espaço (trocar recarrega). Enquanto não há apelido, uma faixa pequena "Defina seu nome em Listas" leva a Listas.
  A antiga linha de cima (nome, espaço, dica e Sair) deixou de existir; quem monta o bloco é `window.claude.sessao()` (api.js).
- **Janela larga (1160 px ou mais):** título "Finanças" (só o texto) e abas em botões, como sempre; o canto direito do
  cabeçalho ganha um **botão da conta** (inicial e nome) que abre o mesmo bloco, com Sair. O limite é o mesmo da tabela de
  Futuros.
- **No app**, as abas em botões (`nav`) continuam sendo a fonte das telas e dos nomes; o menu é montado a partir delas
  (`MENU_APP` em `web/js/app.js` só define a ordem, os grupos e os ícones). Membro não vê Importar, nem no menu. Configurar tem Listas,
  Histórico de versões e Importar.
- **Custo:** trocar de tela passa a pedir dois toques (⋮ e a tela) em vez de um, **exceto Histórico e Saldos** (um toque). Sair
  também pede dois (⋮ e Sair), e o nome só aparece com o menu aberto.
  Em troca, o cabeçalho cai de 2 linhas para 1. A barra inferior estilo C6 foi descartada: lá a barra some em toda tela
  empilhada, e o app é de navegação plana.

## Botão voltar (navegador, Android e app instalado)
Demo [`demos/demo-voltar-navegador.html`](demos/demo-voltar-navegador.html), aprovada pelo dono em 2026-10-19; implementação em `web/js/voltar.js` (o módulo, com `web/voltar.test.js` no CI) e `ligarVoltar` em `web/js/app.js`. Antes o app era uma página só: nada criava entrada no histórico do navegador, então o voltar fechava o app. Vale igual no navegador do computador, no do celular e no app instalado (PWA).
- **O que vira uma entrada do histórico:** cada troca de aba (decidido pelo dono: toda troca cria entrada) e cada **camada**, isto é, janela ou tela cheia que o "←", o toque fora ou o Cancelar já fechariam: menu ⋮, janela de ações e da nota, Filtros, Tags do Lançar, perguntas (`modalOverlay`), Prever e Estender (a tela cheia, as telas internas Repetir, Avisar e Status e a janela de rodas contam uma cada), versões em tela cheia, Editar consulta, modo avançado e janela de tags de Tendências.
- **O voltar fecha só a camada de cima.** Fechar pelo botão da própria janela (ou Esc, ou toque fora) tira a entrada sem fechar mais nada. Os dois caminhos usam o mesmo código: quem abre e quem fecha a janela continua sendo o de sempre, e `voltar.js` só acompanha a contagem (`camadasAbertas()` e `fecharCamadaTopo()` em `app.js`).
- **Editar e Copiar do Histórico:** a janela de ações dá lugar à aba Lançar na mesma entrada, e o voltar leva ao Histórico. Salvar ou Cancelar a edição **desfaz** a ida (`Voltar.retornar()`), em vez de empilhar de novo o Histórico.
- **Formulário de edição com texto digitado:** o voltar (e o "Cancelar edição") pergunta "Descartar o que foi digitado?" (decidido pelo dono). Sem alteração, sai direto. Compara com a foto tirada ao abrir a edição (`assinaturaFormulario`).
- **Desfazer fica fora da pilha** e o voltar nunca o fecha, nem desfaz ou refaz uma exclusão ou edição já gravada. Sem entrada extra para o aviso (decidido pelo dono): no primeiro nível, o voltar sai do app e o botão some; o lançamento já estava gravado e continua recuperável em Histórico de versões.
- **Esc** fecha janelas e telas cheias, nunca sai do app. **F5** reabre na mesma aba (o endereço leva `#historico`, `#lancar`...), sem janela aberta. "Avançar" (Alt + →) não reabre janela fechada.
- **Camada que não fecha** (criando eventos em Prever): o voltar não faz nada e a entrada volta ao histórico.
- **Custo:** alternar entre abas cria um passo a cada troca, então sair do app pode pedir vários voltar; o Voltar sem querer no formulário pede um toque a mais para confirmar; o botão voltar do app instalado no Windows não existe na janela (valem Alt + ←, o botão do mouse e o gesto do touchpad), por isso as telas cheias mantêm o "←".

## Histórico de versões (tela)
Aprovado pelo dono em 2026-10-16 ([`demos/demo-historico-versoes.html`](demos/demo-historico-versoes.html)). Fica em **Configurar**, depois de Listas
(`web/js/versoes.js`, seção `tab-versoes`). Só desenha: a busca, o diff, `restauravel` e `motivo` vêm da API.
- **Busca por mês da alteração** (não o da data do lançamento): ‹ mês › com o seletor nativo (`input type=month`, sem passar do mês
  atual, que é o padrão) e o campo de texto com o botão Buscar (Enter também busca). O texto olha a descrição e a nota de qualquer
  versão do mês.
- **Lista (padrão 8):** uma linha por lançamento com a última alteração do mês: descrição (até 2 linhas), data · banco, a pílula
  (Criado, Alterado, Excluído ou Restaurado), quem e quando, o valor e "versão N". Mês sem alteração mostra o estado vazio, e
  `truncado` mostra a faixa "Há mais versões do que a tela mostra. Refine a busca ou escolha outro mês."
- **Versões:** tocar abre a **tela cheia** com ← (padrão 2); a mais nova primeiro, cada uma só com o que mudou (valor antes → depois).
  A v1 de um lançamento antigo diz "Estado antes da primeira alteração registrada."
- **Restaurar:** botão de 52 px por versão, com janela de confirmação ("Restaurar a versão N?") que diz que será gravada uma versão
  nova e que o lançamento reaparece nas listas, se estava excluído. O botão vem **desabilitado com o motivo** que o servidor mandou
  (a versão é a exclusão, já está nessa versão, mês conferido, só o dono restaura um excluído). Todos os motivos aparecem na mesma cor
  de apoio: o front não decide pelo texto.
- **Computador (1160 px ou mais):** tabela (alterado em, lançamento, o que houve, quem, valor) com o **painel de versões ao lado**,
  no lugar da tela cheia.
- **Custo:** só aparece o mês escolhido; restaurar em mês conferido não deixa, nem para o dono; não há atalho "Ver versões" dentro
  do lançamento. A exclusão imediata com Desfazer por restauração é o item 38 do BACKLOG, não desta tela.

## Confirmação de mês fechado (aprovado em 2026-10-17)

Demo: [`demos/demo-confirmacao-mes-fechado.html`](demos/demo-confirmacao-mes-fechado.html). Regra do item 38: a partir do dia 8 o mês
anterior (e os mais antigos) fecha; mexer nele pede confirmação. **Quem decide é o servidor** (`409 mes_fechado`, `confirmavel: true`):
a tela não calcula nada, só pergunta e reenvia com `?confirmaMesFechado=1` (query, nunca o corpo).

- **Editar / Lançar / Excluir:** a janela de sempre vem primeiro; se o servidor recusar, uma segunda janela: "Este lançamento é de
  agosto de 2026, que já fechou. Alterar mesmo assim?", botão "Alterar mesmo assim" (ou "Lançar…", "Excluir…"). Cancelar não grava
  nada e mostra "Cancelado: nada foi alterado.". O **Desfazer** de uma edição já confirmada reenvia com a confirmação, sem perguntar
  de novo (o `api.js` guarda só os ids confirmados na sessão).
- **Restaurar:** o cartão da versão mostra "Mês fechado: vai pedir confirmação." (vem de `exigeConfirmacao` na API) e a janela de
  confirmação é **uma só**, com o aviso do mês e o botão "Restaurar mesmo assim".
- **Importar:** o arquivo vai sem confirmação; os de mês fechado voltam em `rejeitados` (`codigo: mes_fechado`) e, no fim, **uma**
  pergunta ("N lançamento(s) são de meses já fechados… Importar esses mesmo assim?") reenvia só esses. Recusar deixa-os rejeitados.
- **Mês conferido** continua sem confirmação possível: mensagem do servidor em vermelho.
- **Custo:** ao editar são dois toques e uma ida e volta a mais ao servidor; numa série de repetições cada mês fechado pergunta
  separadamente. Teste: `web/confirmacao.test.js` (CI).

## Tendências (tela)
**Layout** (demo [`demos/demo-tendencias-layout.html`](demos/demo-tendencias-layout.html)): no **computador** (1160 px ou mais, `main` até 1200 px) o **resultado fica à esquerda**
(Sua consulta, gráfico, Outras tags; a coluna acompanha a rolagem) e as **escolhas à direita** (Combos, Grupos, Modo avançado, 430 px). No **celular** o resultado vem
primeiro e "Editar consulta" abre as escolhas em **tela cheia**; o modo avançado abre por cima em outra tela cheia, e uma barra fixa embaixo mostra o resumo ao vivo
(lançamentos, total, % com outras tags) com o botão "Pronto". **Nada de lista de tags ou de combos solta na página:** tudo abre em janela, com as tags em **botões que quebram em várias linhas**: o "+ tag" dos grupos, a linha "Combos" (mostra o combo em uso; escolher um fecha a janela) e o botão "+ Tag" do modo avançado (a tag entra no cursor e a janela fecha). As colunas do computador **não têm barras de rolagem próprias**: a página rola e a da esquerda acompanha.
Demo [`demos/demo-tendencias-tags.html`](demos/demo-tendencias-tags.html); implementação em `web/js/tendencias.js` (tela) e `web/consulta-tags.js` (lógica pura, com
`web/consulta-tags.test.js` no CI). Só desenha e soma o que já está na tela (`lancamentosCache`): nenhum número vai para a API.
- **Grupos (padrão):** cada grupo tem "Precisa ter todas" (E) e "Não pode ter nenhuma" (NÃO); os grupos se ligam por **OU** e o lançamento
  que cai em dois conta uma vez. Tags escolhidas numa **janela de baixo** (lista que acumula com check redondo e busca). Limites à vista:
  4 grupos, 8 tags por lista. Grupo só com NÃO é "tudo, menos estas"; grupo vazio é ignorado; sem nada escolhido mostra tudo.
  Os **combos** (salvos e sugeridos) continuam no topo e carregam um grupo.
- **Modo avançado** (fechado até o toque): expressão livre em fichas com cursor, teclado `( ) E OU NÃO`, lista de tags com busca, E implícito
  depois de tag ou ")", erro em vermelho na própria tela e "Lido como" com todos os parênteses. Abre já com os grupos traduzidos;
  "Voltar aos grupos" descarta a expressão. Limite de 60 fichas.
- **A Tendências é uma simulação, não o saldo** (regra do dono, 2026-10-22): o **Histórico é o saldo real** (ocultar tira o lançamento do saldo); a Tendências serve para
  brincar com os números e fazer projeções, **nunca vale como saldo e nunca grava nada no backend** (só lê o que já está na tela). Uma **faixa fixa "Simulação"** no topo do
  resultado diz isso e avisa que **lançamentos ocultos ficam de fora**, como no Histórico (`CT.base`, com teste). Não há interruptor para contar ocultos.
  **Topo do resultado compacto** (pedido do dono, 2026-10-22, sem demo): no computador uma **linha só** com o selo "Simulação" (o texto inteiro fica no `title` e numa nota embaixo do gráfico), o seletor Gastos | Receitas | Resultado e a consulta (corta com reticências, texto inteiro no `title`), para o gráfico subir. No celular são três linhas: selo e "Editar consulta", a consulta e o seletor.
- **Seletor Gastos | Receitas | Resultado** (demo [`demos/demo-tendencias-visao.html`](demos/demo-tendencias-visao.html)): no topo do resultado, **Gastos é o padrão** (soma as saídas, a regra
  de antes), Receitas soma as entradas e **Resultado é entradas menos saídas das tags escolhidas** (entrada é tudo que não é saída, como em `valorAssinado`). O nome **não é "Saldo"**
  de propósito, para ninguém confundir com o saldo da conta. Uma frase embaixo diz o que está somando. No Resultado a barra do mês **desce** (em `--danger`) quando ele é
  negativo, e a porcentagem de "outras tags" e o peso da lista são do **movimento** (entradas mais saídas, sem sinal): valores que se cancelam não servem de base. A escolha não é
  guardada (volta a Gastos ao recarregar).
- **Lista dos lançamentos da consulta** (demo [`demos/demo-tendencias-lista.html`](demos/demo-tendencias-lista.html), item 42a do backlog; implementada e aprovada em 2026-10-22): o texto
  "Ver os N lançamentos ›" acima do total e cada **barra do mês** abrem uma **janela só de leitura** (tela cheia no celular, janela central de 700 px no computador; `#tdListaVeu`, entra em
  `camadasAbertas()` e `fecharCamadaTopo()` do botão voltar). Não edita, não oculta e não exclui: para isso, o Histórico. Linhas como as do Histórico (descrição, data com ano, status, banco, tags,
  valor com sinal e cor, sem ações), mais novo primeiro; **fichas de mês** (Todos e cada mês, com a contagem) e, em Todos, grupos por mês com a soma. A lista **segue a visão** e a base do gráfico
  (ocultos e futuros fora), e o **rodapé mostra a soma com "✓ Bate com o gráfico"** (ou com a barra do mês): `CT.listar` faz a conta, com teste. Desenha no máximo **200 linhas** ("Mostrando 200 de N.
  Escolha um mês acima para ver o resto."). **Custos:** não deixa agir no lançamento (a ponte "Abrir no Histórico" é o item 42e), sem busca por texto nem ordenação, e só enxerga os 1000 lançamentos
  mais recentes carregados.
- **Gráfico:** barra do mês dividida em "só as tags da consulta" (cheia) e "tem outras tags além das da consulta" (listrada), que somam o total
  do mês; faixa com a porcentagem do valor com outras tags; com erro na expressão o gráfico fica esmaecido com o último resultado válido.
  Meses em fila que rola na horizontal (46 px por mês), começando pelos mais recentes.
- **Outras tags:** as 6 mais pesadas, com a porcentagem do valor da consulta em lançamentos com a tag (**se sobrepõem**, não somam 100%) e
  "Tirar (NÃO)", que põe NÃO em todos os grupos (ou "(expressão) E NÃO tag") com "Desfazer". Sem filtro não há "outras tags".
- **Custo:** "Tirar" tira o lançamento inteiro da conta (o total fica "puro", não "real"); não há divisão do valor entre tags; no celular não se vê o gráfico
  mudar enquanto se edita (a barra de baixo resume) e editar custa um toque a mais ("Editar consulta"), o modo avançado outro; botões de tag ocupam mais
  altura que uma lista com centenas de tags (a busca fica no topo da janela); montar uma expressão custa um toque a mais por tag; a consulta não é guardada ao sair do app.

## Lançar (tela)
Demo [`demos/demo-lancar-layout.html`](demos/demo-lancar-layout.html), aprovada pelo dono em 2026-10-19; implementação em `web/index.html` (`#tab-lancar`), `web/css/app.css` (bloco "Aba Lançar") e `web/js/app.js` (`ligarLancar`). Todos os ids dos campos continuam os mesmos; só o desenho mudou.
- **Modo no topo:** "Lançamento | Transferência" (`#modoLancar`) troca o painel. Editar ou copiar do Histórico volta sempre para Lançamento.
- **Celular:** linha recolhida "Preencher com IA" (abre a caixa de texto); cartão com **Saída/Entrada** (o `<select id="fTipo">` escondido continua sendo a fonte do valor, e `definirTipo()` espelha os botões), **Valor em 32 px** e Descrição; depois linhas de 56 px: **Data, Banco e Status** são o próprio seletor nativo (`datetime-local` e `select`, sem janela nossa, valor à direita), **Tags** abre a janela de baixo, igual à de Tendências (título, busca, botões com ✓, "N tags marcadas" e o botão **Pronto** verde embaixo; as escolhas aparecem como chips na linha) e **Mais** abre nota, repetir, avisar, fatura e "valor estimado". Barra **fixa** embaixo com Limpar (cinza), Cancelar edição (só editando) e Salvar. A mensagem de resultado (`#iaStatus`) fica embaixo da barra, fora da IA recolhida.
- **Transferência:** valor em destaque como em Lançamento, De e Para, e **Data e hora** (`datetime-local`; vazio vale agora).
- **Computador (900 px ou mais):** formulário à esquerda (Valor, Descrição, Data, Banco, Status e Nota em grade de duas colunas, sem "Mais"; a linha **Tags** ocupa a largura toda e lista as tags escolhidas em chips com × para tirar, pois quem escolhe é a coluna da direita), **IA e Tags na coluna da direita**, sempre à vista; as Tags têm um só lugar (`#tagsCard`, que é janela de baixo no celular e coluna no computador). Transferência em coluna de 560 px com De e Para lado a lado.
- **Rótulo "Tags" depois dos Combos** (2026-10-20): nos seletores de tags do Lançar e do Histórico, um rótulo pequeno "Tags" marca onde acabam os combos e começa a lista de tags.
- **Custo:** Nota, repetir, avisar e fatura custam um toque a mais no celular (a tela abre "Mais" sozinha ao editar um lançamento com nota); Tags também; trocar para Transferência são dois toques em vez de rolar a página; a IA fica recolhida.

## Histórico (tela, lista de lançamentos)
Demo [`demos/demo-historico-layout.html`](demos/demo-historico-layout.html), aprovada pelo dono em 2026-10-19; implementação em `web/index.html` (`#tab-historico` e a janela `#histFilOv`), `web/css/app.css` (bloco "Historico") e `web/js/app.js` (`renderHistorico` e funções `*Hist`). Reaproveita as classes `.fut-*` de Futuros (linha, tabela, título de grupo, janela de ações); nada de regra de negócio mudou.
- **Topo (período, 2026-10-20, demo [`demos/demo-historico-periodo.html`](demos/demo-historico-periodo.html)):** o grupo **30d · 90d · 180d · 📅** e a ordem ("Data ↓"); embaixo, a busca por texto (Enter ou a lupa) e o botão **Filtros** com a contagem. Abre em **30d**: a lista corre sem cortar na virada do mês, então o que foi lançado ontem aparece mesmo no dia 1º. O 📅 escolhe **um mês** (seletor nativo `input type=month`, aberto por `showPicker()`): com um mês escolhido o botão mostra o nome dele ("set/2026"), os outros ficam desmarcados e aparece a linha ‹ mês ›. Não há "de/até". **Uma regra:** o que está marcado é o que a lista mostra e o que a busca olha. Os filtros ativos viram fichas com ×.
- **Busca no período:** olha só o período ou o mês marcado (a varredura é limitada, então responde rápido). Embaixo dos filtros, a faixa "Resultado dos últimos 90 dias" (ou "de setembro de 2026") traz **Buscar em tudo**, que abre um aviso (`confirmarModal`: pode demorar) e depois mostra "Resultado de todo o histórico" com um botão para voltar. Trocar o período com uma busca ativa **refaz a busca** no novo escopo. Código: `periodoHist`, `limitesEscopoHist`, `cacheCobre`, `carregarEscopoHist`, `aplicarEscopoHist` e `buscarTextoHistorico(tudo)` em `app.js`; a lista usa o cache (os 1000 mais recentes) quando ele chega até o início do período e consulta a API por `de`/`ate` quando não chega. **O que a busca olha (2026-10-21):** descrição, nota e **tags**, sem diferenciar maiúsculas nem acento (`termoBusca` e `textoBuscavel`, que usam `normTxt`); um `#` no começo é ignorado (quem digita `#mercado` acha a tag) e espaços sobrando também. Casa por parte da palavra ("merc" acha `mercado`), ao contrário do filtro de Tags, que é exato. O dado guardado não muda. Texto do campo: no computador "Buscar neste período (descrição, nota ou tag)" (ou "…neste mês…"); no celular "Descrição, nota ou tag", porque o campo é estreito e o período já aparece no grupo de cima e na faixa do resultado.
- **Filtros (janela de baixo no celular, janela central no computador):** Status e Banco em botões com ✓, Tags com o mesmo seletor de antes (combos, "todas" ou "qualquer uma"), chaves "Mostrar ocultos" e "Incluir eventos futuros" (só na busca), "Limpar" e "Ver N lançamento(s)" com a contagem ao vivo. No mês aberto os filtros valem só para o mês; na busca, para o resultado de todo o histórico.
- **Linha (celular):** descrição em até 2 linhas, hora (ou a data, na busca), status, banco com reticências e, embaixo, as tags em **uma linha** com reticências; valor sem quebrar e "⋯". **Nota não aparece no texto**: só o ícone de página, que no computador mostra a nota ao passar o mouse e, nos dois, o clique abre uma janela com o texto. Tocar no resto da linha abre as ações (`abrirAcoesFuturo`).
- **Grupos:** no celular, um título por dia (por mês, na busca). No computador, um título por **mês** com a quantidade e a **soma do mês** quando o período passa de um mês ou há busca, e nenhum título quando é um mês só. O total da lista continua à direita do contador (10b) e agora é o do **período mostrado**, não o do mês.
- **Fim da lista (celular):** "Ver 90d" (ou 180d) para ampliar o período sem subir, ou a troca de mês quando há um mês escolhido, e "↑ Topo".
- **Computador (1160 px ou mais):** mesma tabela de Futuros (Data com hora, Lançamento, Banco, Status, Valor, ações na linha apagadas até o mouse passar); "Data" no cabeçalho inverte a ordem; busca, Filtros e mês ficam numa faixa que acompanha a rolagem. O limite 1160 px é o mesmo de Futuros (`MQ_TABELA` em `app.js`).
- **Custo:** filtrar custa um toque a mais (abrir a janela); agir num lançamento custa um toque a mais no celular; no computador os filtros ficam atrás de um botão; só as tags que cabem em uma linha aparecem. Período: o total deixa de ser "o do mês" (para um mês exato, use o 📅); conferir o extrato de um mês custa um toque a mais; achar algo antigo custa dois toques e a espera do aviso; 180d no celular é uma lista longa; o seletor de mês é o do navegador (no computador, o calendário com ‹ ›).

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
dias", "atrasado 3 dias", este último em `--danger` e negrito). **O prazo fica embaixo do valor em todas as larguras;
nunca é coluna própria.** A **descrição ocupa até 2 linhas** (`-webkit-line-clamp:2`, com reticências; o texto inteiro
está na janela de ações e em Editar). O sino de aviso fica na linha de baixo (celular) ou ao lado do texto (computador),
fora do trecho cortado. Na ponta direita da linha vai um **"⋯"** (3 pontos, `--muted`) que mostra que a linha abre
ações; a linha toda continua tocável. Se o valor e o "⋯" deixam menos de 120 px para a descrição (valor de 7 dígitos
em 320 px), o valor desce para a segunda linha, à direita. **Realce:** ao passar o mouse (`@media (hover:hover)`),
focar ou tocar (`:active`), a linha ganha fundo `color-mix(in srgb, <cor do status> 16%, var(--card))`, a cor que o
usuário escolheu em Listas, Aparência. Grupos com título por mês e um grupo "Atrasados" no topo, em `--danger`.
**Saldo do mês** (aprovado em 2026-10-11): à direita do título de cada mês, só o valor (13 px, negrito, `nowrap`),
com sinal e cor (`+` em `--income`, `-` em `--danger`, zero em `--muted`) e o cálculo a receber menos a pagar, sem os
ocultos. "Atrasados" não tem saldo (o resumo do topo já mostra). Evento oculto fica a 50% de opacidade.

**9. Ações numa janela de baixo.** Tocar na linha do evento abre a janela com descrição, valor, data e banco no topo e as
ações em lista (Editar, Copiar, Prever ou Estender série, Ocultar ou Mostrar, Excluir). "Excluir" em `--danger`. Nada de
fileira de botões dentro da linha.

**10. Resumo.** No topo, "A pagar" e "A receber" lado a lado (metade da largura cada um), com o valor em **18 px**. "Atrasados"
é uma **faixa** de largura total embaixo, com fundo `--danger-bg`, o nome à esquerda e "N · valor" à direita em 16 px, e
**só aparece quando há atraso** (sem atraso o resumo volta a ter cerca de 60 px). Substitui o texto explicativo longo.
**O valor monetário nunca quebra** (`white-space:nowrap`): se faltar espaço, a linha cresce, e não o número. Três
blocos iguais lado a lado só no computador (padrão abaixo).

**10b. Total da lista no Histórico** (aprovado em 2026-10-11). Na linha do contador, à direita, só o valor (13 px,
negrito, `nowrap`) com sinal e cor (`+` em `--income`, `-` em `--danger`, zero em `--muted`): a **soma de renda menos
gasto** dos lançamentos exibidos, com o mesmo sinal de cada linha (500,00 de renda e 550,00 de gasto dão `-R$ 50,00`).
Vale na **busca por texto** e no **mês aberto**, e acompanha Status, Banco, Tags, "mostrar ocultos" e "incluir eventos
futuros". Some com a lista vazia e durante "Buscando...". É a soma do que está na tela, não o saldo do banco. O texto
do contador à esquerda quebra em duas linhas em 320 px; o valor nunca. A classe `.soma-valor` é a mesma do saldo do mês
em Futuros.

**11. Aviso rápido (toast).** Faixa escura na base da tela, 2 segundos, para confirmar ("Oculto: saiu dos totais.").

**11b. Excluir na hora e Desfazer depois de excluir ou editar** (editar: aprovado em 2026-10-14; excluir imediato: aprovado
em 2026-10-17, item 38 fase C, demo [`demos/demo-exclusao-imediata.html`](demos/demo-exclusao-imediata.html)). Duas camadas, de
propósito: a **janela de confirmação** pega o engano de toque e o **aviso com Desfazer** pega o arrependimento.
- **Excluir:** janela "Excluir este lançamento (descrição · valor)? Depois de excluir, dá para desfazer por 6 segundos." com
  Cancelar e Confirmar. Confirmado, o `DELETE` vai **ao servidor na hora** e o lançamento **continua na lista**, esmaecido e
  com "Excluindo…" (`.excluindo`), até a resposta. Só com o `200` ele some de todas as telas e abre o aviso "Excluído" com
  **Desfazer**. Recusas chegam **antes** de ele sumir: mês conferido → mensagem em vermelho; mês fechado por data → a janela da
  confirmação ("…Excluir mesmo assim?", ver a seção Confirmação de mês fechado) e reenvio; rede → aviso vermelho com "Tentar
  de novo", e o lançamento fica.
- **Desfazer a exclusão é uma restauração:** o app pede `POST /tx/{id}/versoes/{n}/restaurar` com a `desfazer.versao` que o
  `DELETE` devolveu (leva a confirmação do mês fechado, se houve). Volta com o mesmo id e uma versão nova no histórico. Se falhar
  (rede, mês conferido nesse meio tempo): aviso vermelho dizendo que o lançamento continua excluído e que dá para recuperá-lo
  em Histórico de versões.
- **Editar:** o botão Salvar da edição abre "Salvar a alteração?" com o que mudou (`Valor: R$ 50,00 → R$ 75,50`).
  Confirmado, a edição é gravada e o aviso mostra "Edição salva" com Desfazer, que reenvia os valores anteriores como
  uma edição normal (campo que não existia volta `null`). Nada muda de contrato: é o `PUT` de hoje.
- **Aviso:** faixa escura na base, 6 s, texto + linha de apoio + botão **Desfazer** (alvo de 48 px) e uma barra fina
  que mostra o tempo. **Um por vez:** abrir outro fecha o anterior (o Desfazer dele some). Erro: aviso vermelho, com "Tentar de
  novo" quando cabe. `role="status"`; a barra não anima com `prefers-reduced-motion`.
- **Custos que a demo diz:** um toque a mais por exclusão e edição; o lançamento só some depois da resposta do servidor (cerca
  de um segundo); excluir e desfazer deixam **duas versões** no histórico; o Desfazer vale só na sessão e só por 6 s (depois,
  o Histórico de versões); trocar de aba ou de app **não** muda a exclusão (já foi feita), só tira o botão Desfazer; ele não
  passa por cima da regra do backend (conferido, mês fechado) nem da lacuna da fatura do cartão (item 33).
- **Ocultar** continua sem confirmação e sem Desfazer (fica para uma decisão própria).
- Implementação (em `web/js/app.js`): `mostrarAvisoAcao` e `fecharAvisoAcao` (aviso), `iniciarExclusao` (DELETE na hora, set
  `excluindo`), `desfazerExclusao` (restauração), `esconderLocal` e `devolverLocal` (tiram e devolvem o item das listas) e
  `desfazerEdicao` (edição). Os `onSnapshot` e as buscas filtram `exclusaoPendente`, para um item excluído não reaparecer
  enquanto o Desfazer vale. `db.collection().doc().delete()` devolve o corpo (com `desfazer.versao`).

**12. Fluxo de criação em tela cheia (Prever e Estender).** Uma tela cheia por cima do app (fixa, com a rolagem da página
travada) que junta os padrões 1 a 7:
- **Cabeçalho** com `←` (fecha) e o título; **linha de contexto** em `--muted` com descrição, valor, status e data do
  evento de partida.
- **Cartão de linhas-resumo:** Status das cópias, Repetir e Avisar no Telegram, cada uma abrindo a sua tela cheia (com `←`
  para voltar). Parcelas (`2/10`) acrescentam um interruptor "Continuar a numeração das parcelas".
- **Prévia:** "Cria N eventos: d1, d2, d3, d4 e mais k" e "Total R$ x" (datas com ano).
- **Já existem:** aviso em `--danger` com as datas e uma escolha única "Pular os que já existem" (padrão) ou "Criar
  mesmo assim". Se tudo já existe, o botão não cria nada e a tela explica.
- **Botão fixo embaixo** "Criar N eventos", que durante a criação mostra "Criando 3/12…". Não use botão desabilitado: o
  problema aparece como texto na tela.
- **Falha no meio:** a tela fica aberta, diz "Parou depois de 2 de 3", recalcula a prévia (os criados passam a "já
  existem") e a nova tentativa cria só o que falta.
- **Esc** volta um nível: fecha a janela de rodas, depois a tela interna, depois o fluxo (nunca durante a criação).

## Padrão do computador (janela larga)
Mesma tela, mesmos dados, desenho de tabela.
- **Tabela de largura total** com cabeçalho cinza e uma linha por evento, em grade de colunas:
  `92px minmax(170px,1fr) 90px 140px 128px 316px` = data, evento, banco, status, valor (alinhado à direita, com o
  prazo embaixo), ações. A coluna do banco corta nomes compridos com reticências (o nome inteiro vai no `title`), e a
  coluna de status e o prazo não quebram linha. A descrição ocupa até 2 linhas e leva o nome inteiro no `title`. O espaçamento entre colunas é de 12 px. A tabela tem largura mínima de **1120 px** e rola na
  horizontal dentro do próprio quadro abaixo disso. A visão de tabela começa em **1160 px de janela** (1120 px de
  tabela mais as margens do container); abaixo disso vale a visão do celular. (A soma das colunas fixas mais o `minmax` tem de caber na largura
  mínima, senão a coluna do nome colapsa e as colunas se sobrepõem.)
- **Container largo:** o conteúdo passa a até 1600 px (era 1200; em monitor grande a descrição ficava presa em ~280 px). No app, o `main{max-width:640px}` é o que deixa a tela estreita
  no computador; a tela larga precisa de um modificador, e não de trocar o `main` para todas as telas.
- **Ações na própria linha**, no fim, em botões de texto pequenos, **apagadas (opacidade 0,25) até o mouse passar ou o
  foco entrar** (`:hover` e `:focus-within`, linha com `tabindex="0"`). Sem "⋯", sem janela de baixo e sem toque extra. É
  a **única diferença** entre as duas larguras: a mesma linha, com "⋯" abaixo de 1160 px e botões a partir dela.
- **Fluxos em tela cheia** (Prever e Estender) viram uma **coluna central de 560 px**, com o botão de criar da mesma
  largura. A tabela é só para listas.
- **Resumo:** três blocos lado a lado, até 600 px de largura (cada um com uns 195 px), valor em 16 px, e "Atrasados" sempre
  visível, inclusive com 0. O **interruptor de ocultos** fica no topo, como no celular; os totais continuam excluindo os
  ocultos.
- **Texto curto nas ações da tabela** ("Estender", "Prever") e o texto completo na janela de baixo do celular
  ("Estender série", "Prever próximas datas").
- Telas em largura média (tablet) usam a visão do celular até haver uma demo própria.

## Regras de conteúdo
- **Datas sempre com o ano**, `dd/mm/aaaa`, em tela e em mensagem. Moeda em pt-BR (`R$ 1.234,56`).
- Português com letra maiúscula só no começo da frase; verbos curtos nas ações ("Editar", "Estender série").
- Prazo relativo junto da data real, nunca no lugar dela (na Futuros, embaixo do valor).
- Todo limite (quantidade, dias, tamanho) aparece na tela antes de ser atingido e tem mensagem de erro própria.

## Acessibilidade e tema
- Controles com `role` e `aria-checked` corretos (`checkbox`, `radio`, `switch`) e `aria-label` quando não há texto.
- Alvos de toque de 56 px nas linhas; botões de janela com 52 px.
- Respeitar `prefers-reduced-motion` (a bolinha do interruptor e a seta animam só quando permitido).
- Tema claro e escuro **pelos tokens**, nunca por cor literal; testar os dois.

## Manter o padrão em dia (regra obrigatória)
As demos de `demos/` são a fonte da verdade deste padrão. Elas só continuam sendo a fonte da verdade se quem **implementa
ou testa** uma tela devolver o que descobriu. Sempre que a implementação, o teste ou o uso real mostrar que algo do
padrão estava errado ou incompleto (largura de coluna, rótulo, limite, comportamento, token, caso de borda), faça, **na
mesma entrega** (o mesmo PR, ou um PR de documentação aberto junto):
1. **Corrija a demo** de referência em `demos/` para refletir o que ficou certo.
2. **Atualize este guia**: as medidas e os padrões afetados, e uma linha em "Armadilhas já encontradas" dizendo o que
   aconteceu e como evitar.
3. **Atualize o resumo** da seção "Telas" do `.ai/guardrails.md` se o resumo mudou.
4. **Diga ao dono**, na resposta e na descrição do PR, o que mudou no padrão e por quê.

Não deixe a correção só no código do app: a próxima tela vai copiar a demo e repetir o erro. Antes de abrir o PR,
confira se `git diff` mostra `web/` e `.ai/` andando juntos quando o padrão mudou.

### Antes de entregar uma tela, meça o pior caso
Uma tela que funciona com os dados de teste bonitos ainda pode quebrar com os dados reais. Semeie e confira, no
celular (≈ 390 px) e numa janela larga (≥ 1280 px), nos dois temas:
- o rótulo mais longo de cada coluna e botão (status, ação, série);
- nome de banco ou de pessoa comprido, descrição longa, atraso de três dígitos, valor de cinco dígitos;
- itens com todos os adornos ao mesmo tempo (sino, série, oculto);
- lista vazia, com um item e com muitos itens.

Meça por script que nenhum conteúdo passa da própria coluna nem encosta no vizinho, em **todas** as colunas (não só nas
ações), e tire uma captura para olhar.

## Como reproduzir numa tela nova (receita)
1. Liste o que a tela precisa mostrar e decidir. O que não for essencial sai da vista (padrão 1).
2. Escolha os padrões desta lista; não invente um componente novo se um existente serve.
3. **Faça a demo** copiando a estrutura de uma das demos de referência (CSS e JS ficam num arquivo só). Publique como
   página com link que abre no celular (artefato privado); o widget do chat só aparece no desktop.
4. Inclua na demo o que se perde com a proposta e a visão "como está hoje" para comparar.
5. Espere o ok do dono. Só então implemente, com os tokens do app (tabela acima) e a lógica real por baixo.
6. Teste no navegador: celular (≈ 390 px) e janela larga (≥ 1280 px), tema claro e escuro, e os limites e os erros,
   com o pior caso de cada coluna (seção acima).
7. Devolva ao padrão o que descobriu (seção "Manter o padrão em dia"): demo, guia e recomendações, na mesma entrega.

## Armadilhas já encontradas
- Esconder um lançamento só da lista do Histórico deixa o item vivo em Futuros, saldos, gráfico e cartão, que leem
  `lancamentosCache`. A exclusão adiada tira o item das três listas (`lancamentosCache`, `lancamentosMes`,
  `lancamentosBusca`) e roda as mesmas atualizações do `onSnapshot`; e os `onSnapshot` e a busca filtram
  `exclusaoPendente`, senão uma atualização que chega nos 6 s traz o item de volta.
- `aplicarLocal` altera o objeto que está na lista: para o Desfazer de uma edição, copie o registro **antes** de gravar.
- O navegador de teste do painel não gera o evento `scroll` quando a página não está sendo desenhada: para testar as
  rodas por script, ajuste `scrollTop` **e** dispare `dispatchEvent(new Event('scroll'))`. No celular real o evento
  dispara sozinho (a roda usa `scroll-snap` e lê a posição 90 ms depois de parar).
- Grade de colunas cuja soma passa da largura da tabela faz a coluna `1fr` colapsar e sobrepõe o texto.
- Rótulo longo estoura a coluna (na Futuros, "Estender série" empurrava "Editar" para cima do valor, e a pílula
  CREDITANDO invadia o prazo). **Meça todas as colunas com o pior caso**: status mais longo (TRANSFERINDO), banco de nome
  comprido, atraso de três dígitos, valor de cinco dígitos, descrição longa com série, sino e oculto. Confira em janela
  larga que nenhum conteúdo passa da própria coluna nem encosta no vizinho, e que a primeira ação não toca o valor.
- Descrição livre vem com 100 caracteres ou mais (quem digita não sabe se vai conseguir ler a nota depois). Linha de
  lista sem limite de linhas vira bloco de 4 linhas, e coluna de tabela só para o prazo desperdiça largura que a
  descrição precisa. **Prazo embaixo do valor e descrição em até 2 linhas** resolvem nas duas larguras. Em demo,
  "como está hoje" tem de ser **igual ao app** (selo colorido do status, sem corte), senão a comparação engana.
- Container com `max-width` baixo em monitor grande: a coluna da descrição não cresce. Meça também a 1920 px.
- Data sem ano (`31/01`) é ambígua quando o intervalo passa de um ano.
- A demo cobria as telas "Repetir" e "Avisar" mas não o fluxo inteiro do Prever (status das cópias, botão de criar, "já
  existem", falha no meio). Uma IA nova teria de improvisar. **Defina na demo todos os estados do fluxo** (vazio, erro,
  limite, progresso, conclusão) antes de implementar.
- Depois de uma falha parcial, o que já foi criado só entra no cache da lista uns instantes depois: espere cerca de
  400 ms antes de recalcular o "já existem", ou a nova tentativa repete o que já foi criado.
- Corrigir só o código do app e esquecer a demo e o guia: a demo continua errada e a próxima tela copia o erro. Aconteceu
  duas vezes na tela Futuros (ações sobre o valor e pílula sobre o prazo). A regra está em "Manter o padrão em dia".
- Três colunas iguais para números de largura variável: o valor quebra no meio ("R$ 182.012,2 / 7") quando passa de
  uns R$ 99.999,99 num celular. Valor monetário leva `white-space:nowrap`, e o layout é que cede (faixa embaixo,
  linha que cresce). Meça com valores de seis e sete dígitos em 320, 360 e 390 px.
- Esconder um elemento cujo `display` foi definido por **id** com uma regra só de classe não funciona: o id vence.
  Use o mesmo id na regra (`#futMetricaAtras.sem-atraso`). Sempre confira por `getComputedStyle(...).display`.
- `scrollWidth` não acusa estouro para a **esquerda** em linha com `justify-content:flex-end` (as ações da tabela). Meça
  as extremidades com `getBoundingClientRect()` da coluna e do primeiro e do último botão.
- Dois elementos com o **mesmo id** (o container dos grupos e a janela da lista ficaram ambos `tdLista`) fazem o `$()` achar o primeiro e a janela nunca aparece (altura 0, `position:static`). A janela
  é `tdListaVeu`. Antes de criar um id novo, confira que ele não existe: `grep -n "id=\"x\"" web/js/tendencias.js`.
- `#tendRaiz button{color:inherit}` com id **vence** as cores de `.td-*` (o "Tirar" e o segmento marcado ficavam com a cor do texto). O reset usa `:where(#tendRaiz) button`
  (especificidade zero), e cada botão traz a sua cor.
- Gráfico que rola na horizontal desenhado com a aba **escondida** (`hidden`) tem `scrollWidth` 0: o "rolar até o fim" não pega. Em Tendências um
  `ResizeObserver` rola até os meses recentes quando o gráfico ganha largura.
- Esconder um elemento com uma regra de classe simples (`.lh-tags{display:none}`) dentro de `@media` não adianta se uma regra **posterior** de mesma especificidade (`.lh{display:block}`) o mostra de novo: no Lançar a linha "Tags" apareceu duas vezes no computador. Dobre a classe (`.lh.lh-tags`) e confira por `getComputedStyle`. O mesmo erro, ao contrário, escondeu os chips das tags na demo do computador: `.tela.pc .linhas .lh .v{display:none}` vencia a regra da linha Tags. Ao mostrar de novo o que uma regra geral escondeu, use um seletor **mais específico** que ela.
- **Testar uma tela sem login:** o app real exige Cognito (a página redireciona), então não dá para abri-lo no navegador do painel. Para conferir só o visual, copie `web/css` e um `index.html` sem as tags `<script>` para a pasta temporária, preencha selects e chips por um script curto de exemplo e sirva com `python -m http.server`. Isso **não cobre** salvar, editar, copiar, IA nem transferência: diga isso na entrega e peça o teste no ambiente dev.
- Texto de `placeholder` comprido é cortado no celular ("Buscar descrição, nota ou tag" não coube em 223 px). Meça o texto contra a largura do campo (`measureText`) e use uma versão curta no celular.
- Busca de texto que não normaliza acento falha com "farmacia" x "farmácia". Compare sempre os dois lados com `normTxt`, sem alterar o dado guardado.
- Elemento interativo dentro de `<button>` não vale: a linha do Histórico é um `div role="button"` (com Enter e Espaço tratados) para caber o ícone da nota, que também é clicável.
- Janela de baixo com texto livre (a nota do Histórico chega a dezenas de linhas) precisa de `max-height` e rolagem **dentro do texto**, com o botão fixo embaixo; sem isso a janela passa da tela e o botão some. Teste com uma nota de 60 linhas.
- **Janela ou tela cheia nova precisa entrar no botão voltar:** acrescente o teste de "está aberta" em `camadasAbertas()` e o jeito de fechar em `fecharCamadaTopo()` (`web/js/app.js`), na ordem de cima para baixo. O `MutationObserver` só enxerga mudança de `class`, `hidden` e filhos do `body`; se a camada mudar só por estado do JavaScript (como as telas internas de Prever), chame `Voltar.sync()` depois de redesenhar. Sem isso o voltar ignora a camada e fecha o app ou a aba por baixo.
- `history.go()` seguidos não se somam em todo navegador: `voltar.js` pede um por vez e espera o `popstate` do anterior antes de acertar o próximo. Não chame `history.back()` à mão no app; mude o estado da tela e deixe o módulo acompanhar.
- Este botão voltar só foi testado em Node (`web/voltar.test.js`) e num navegador de teste com uma página de mentira; o app real exige o login do Cognito. Confira no ambiente dev, no celular e instalado, antes de dar por fechado.
- `\n` dentro de heredoc do shell no Windows se corrompe: edite arquivos com a ferramenta de edição.
