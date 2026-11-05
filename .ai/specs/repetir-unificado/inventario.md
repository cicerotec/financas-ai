# Inventário: onde Repetir, Recorrente e Avisar aparecem hoje

Levantado em 2026-11-03 a partir de `web/js/app.js`, `web/api.js` e `backend/` (service e domain). **É o que o código faz hoje**, sem proposta. Serve de base para o
[`plan.md`](plan.md): nenhuma demo nem decisão de tela deve ser feita sem cobrir cada linha daqui.

## 1. Pontos de entrada (quem abre o quê)

| # | Tela | Gesto | O que abre | Função | Quando aparece |
|---|---|---|---|---|---|
| E1 | Lançar (novo) | linha **Repetir** em "Mais" | blocos soltos no formulário (quantidade e intervalo) | `htmlRepeticao('fr')` | lançamento **novo** e **data futura** (`dataFuturaForm()`) |
| E2 | Lançar (novo) | linha **🔁 Recorrente** em "Mais" | tela cheia só com o Repetir, rodapé Concluir / Não repetir | `abrirRecorrenciaForm` | lançamento **novo** (qualquer data) |
| E3 | Lançar (novo e edição) | caixa **🔔 Avisar no Telegram** em "Mais" | caixa + dias + "Insistir" soltos no formulário | `htmlAviso('f')` | **data futura**, novo ou edição |
| E4 | Lançar (edição) | **Editar** de um lançamento | o formulário com o aviso preenchido; "Mais" abre sozinho se houver nota ou aviso | `iniciarEdicao` | Repetir e Recorrente **somem**; aviso só com data futura |
| E5 | Lançar (cópia) | **Copiar** do Histórico ou de Futuros | formulário novo; o aviso é **limpo** e a repetição volta a 1 | `iniciarCopia` | como E1 a E3 |
| E6 | Histórico e Futuros | ação **Prever próximas datas** | tela cheia: linhas Repetir e Avisar, prévia das datas, "Criar N eventos" | `preverLancamento` → `abrirPrever` | todo lançamento que **não é transferência** e **não tem série** |
| E7 | Histórico e Futuros | ação **Estender série** | a mesma tela, partindo da **última da série**, com a numeração de parcelas | `preverLancamento` → `abrirPrever({estender})` | lançamento **com série**; busca a última entre os lançamentos e os previstos carregados |
| E8 | Histórico e Futuros | ação **Tornar recorrente** | tela cheia: linhas Repetir (só intervalo) e Avisar, "Ativar recorrência" | `abrirRecorrencia` | não é transferência, a pessoa pode mexer e o lançamento **ainda não é recorrente** |
| E9 | Histórico e Futuros | ação **Parar recorrência** | só uma confirmação; **sem tela** | `pararRecorrencia` | só na **ponta** da recorrência (`recorrenciaDe`) |
| E10 | Histórico | **seleção em lote** → "Tornar previstos" | tela cheia com **Mês de destino**, Repetir (interruptor), Avisar, card de duplicados, limite de 100 | `abrirLote` | transferências ficam de fora (`podeNoLote`) |
| E11 | Futuros | ação **Avisar no Telegram** de uma **fatura** | só a tela Avisar, com **Salvar** no rodapé; seta só desiste | `abrirAvisoFatura` | só o dono; a fatura precisa ter vencimento |
| E12 | Cartão | linha **Aviso (padrão)** em "Configurar cartão" | só a tela Avisar, com **Salvar** | `abrirAvisoPadrao` | só o dono |
| E13 | Histórico e Futuros (linhas) | marcas **🔔** e **🔁** | só exibem | `chipAviso`, `chipRecorrente` | 🔔 só se o lançamento é **previsto**; 🔁 se é recorrente |

Em Transferência (aba Lançar) não há aviso nem repetição. Tendências não tem nenhum dos dois.

## 2. Os sete "modos" da mesma janela (`pv`, `#tc`)

A janela cheia e o estado global `pv` servem a **sete** variações. A coluna "o que a tela mostra" é o que existe de verdade:

| Modo | Aberto por | Repetir | Quantas vezes | Interruptor Repetir | Mês de destino | Numerar parcelas | Duplicados | Avisar | Rodapé |
|---|---|---|---|---|---|---|---|---|---|
| Prever | E6 | `pvTelaRepetir` | sim ("além deste") | não | não | se a descrição tem parcela | sim | sim | Criar N eventos |
| Estender | E7 | `pvTelaRepetir` (intervalo da série) | sim | não | não | sim | sim | sim | Criar N eventos |
| Lote | E10 | `pvTelaRepetirLote` | sim ("além do primeiro mês") | **sim** | **sim** | automática | sim | sim | Criar N previstos |
| Recorrência (existente) | E8 | `pvTelaRepetirRec` | **não** | não | não | não | não | sim (vale só para a 1ª cópia) | Ativar recorrência |
| Recorrência (Lançar) | E2 | `pvTelaRepetirRec` com contexto | **não** | não | não | não | não | **não** (o aviso é o do formulário) | Concluir / Não repetir |
| Fatura | E11 | não | não | não | não | não | não | sim | Salvar |
| Aviso padrão do cartão | E12 | não | não | não | não | não | não | sim | Salvar |

A tela de Avisar (`pvTelaAvisar`) é a mesma nos cinco modos que a têm; só a **nota de baixo** muda por modo (recorrência, fatura, padrão ou normal), e o balão "Assim chega no Telegram" usa
descrição e valor do `pv.base` (que, na fatura, é um lançamento montado só para isso).

## 3. O que cada gesto grava (o servidor não muda)

| Gesto | Chamada | O que o servidor faz com o aviso |
|---|---|---|
| Lançar sem repetir | `POST /tx` com `aviso` no corpo | `Sincronizar` grava o item do aviso **sem olhar a data** do lançamento |
| Lançar, repetição (E1) | `POST /tx` do lançamento (com `serie`) e **um `POST /tx` por cópia**, em loop no navegador (`criarEventos`) | igual; cada cópia leva o aviso e a série |
| Lançar, Recorrente (E2) | `POST /tx` com `recorrente: {intervalo}` | cria a **primeira cópia** copiando o original **inclusive o `aviso`** (`CopiaDeRecorrencia`, `backend/domain/recorrencia.go:124`; `Recorrencias.abrir` sem aviso explícito) |
| Tornar recorrente (E8) | `POST /recorrencias?de=` com `{id, intervalo, aviso?}` | aviso **só na primeira cópia** (`null` = sem aviso, omitido = herda o do original); o **original não muda** (R17 do spec do item 51) |
| Prever, Estender, Lote | um `POST /tx` por cópia, em loop no navegador | igual ao Lançar |
| Parar recorrência (E9) | `DELETE /recorrencias/{id}` | copias já criadas ficam |
| Aviso da fatura (E11) | `PUT` de `faturas/{ym}` com `{aviso}` | item auxiliar da fatura |
| Aviso padrão (E12) | `PUT` do cartão com `{aviso}` | vale para as próximas faturas que ganharem vencimento |

`Consulta.add` (o `POST /tx`) **devolve o `id`** do lançamento criado (`web/api.js:364`).

## 4. Ícones e colunas da tabela do computador (padrão de 6 lugares fixos)

`LUGAR_ACAO` (`app.js:2088`): confirmar 1, editar 2, copiar 3, **prever e avisar 4**, **recorrente e parar recorrência 5**, ocultar 6, excluir 7. O **Estender** usa o lugar do Prever e o
ícone muda (`estender`). Mudar o conjunto de botões muda essas colunas, que o `padrao-de-telas.md` fixa ("seis lugares fixos").

## 5. Regras escondidas (premissas) que só valem por causa de outra regra

| # | Premissa | Depende de | Se a outra deixar de valer |
|---|---|---|---|
| P1 | O Recorrente criado pelo Lançar herda o aviso do original | o Lançar **só mostra o aviso com data futura** | com a data de hoje o original ganharia um aviso sem sentido; o servidor o grava sem olhar a data |
| P2 | A marca 🔔 só aparece para previsto (`chipAviso`) | o servidor **marca `previsto` pela data futura** | um aviso gravado num lançamento de hoje não aparece em lugar nenhum da tela |
| P3 | O Lançar pede "Total de ocorrências (**com este**)" e o Prever "quantas vezes **além deste**" | cada tela tem o seu texto | trocar o texto muda o sentido de um número que o dono já usa |
| P4 | `iniciarEdicao` abre "Mais" sozinho quando há aviso e confia em `atualizarCampoFatura()` para mostrar o aviso só com data futura | a regra de data futura | sem ela, a edição de um lançamento de hoje mostraria o aviso |
| P5 | `limparFormulario`, `iniciarEdicao` e `iniciarCopia` mexem em `formRec`, `frQtd` e `preencherAvisoNoFormulario` | os blocos `htmlRepeticao` e `htmlAviso` existirem no formulário | remover os blocos quebra essas três funções |
| P6 | `camadasAbertas()` conta `pv`, `pv.tela` e `pv.sheet`; o botão voltar do celular depende disso | o estado global `pv` | mudar o estado sem mudar a contagem faz o voltar fechar o app |
| P7 | Prever e Lote ignoram transferência; Recorrência **recusa** transferência no servidor | `transferParId` | uma tela nova que mostre a opção para transferência quebra |
| P8 | "Parar recorrência" só existe na **ponta** | `recorrenciaDe(l)` (lê `recAtivas`) | excluir a cópia mais recente tira o botão e a recorrência não pode mais ser parada pela tela (limite já registrado no item 51) |
| P9 | Estender precisa dos lançamentos **e** dos previstos carregados | `lerDadosParaDuplicados()` | sem ela, a "última da série" é a errada |
| P10 | A tabela do computador tem seis lugares fixos para ícones | `LUGAR_ACAO` | botão novo ou fundido muda a coluna dos outros |

## 6. Testes de front que tocam isto hoje

`recorrencia.test.js` (datas, intervalos, avisos, cópias), `consultas-camadas.test.js` e `voltar.test.js` (camadas do botão voltar), `carga.test.js` (a subida não lê o que não é dela) e
`sem-status.test.js` (nenhum nome de status em `web/`). **Nenhum teste cobre o HTML das telas Repetir e Avisar**, nem o corpo do `POST /tx` montado pelo Lançar com repetição.
