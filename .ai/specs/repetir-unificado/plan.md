# Plan: tela Repetir única

Segue o [`intent.md`](intent.md); o roteiro verificável está no [`spec.md`](spec.md).

**Estado:** rascunho para aprovação do dono (2026-11-03). Demos aprovadas (PR #350).

## Decisões

1. **Só front.** Nenhuma mudança em `backend/`. Quem decide se um lançamento é previsto continua sendo o servidor; o front deixa de esconder controles por data (`dataFuturaForm()` sai da regra de
   mostrar Repetir e Avisar).
2. **Lógica pura em `web/recorrencia.js`** (já testado em node): estado inicial do aviso, frase do aviso, posições do seletor por contexto e o **plano de criação** do Lançar (`planoDeRepeticao`: dado
   o lançamento e a escolha, devolve o documento, as cópias, a marca `recorrente` e onde o aviso vale). A tela em si vai para um arquivo novo `web/js/repetir.js`, como `versoes.js` e `tendencias.js`
   (o `app.js` tem 4.256 linhas). Nome do arquivo a confirmar.
3. **Casca e seções com entradas explícitas.** A casca (`montarTc`) abre a janela uma vez; cada seção (Repetir, Avisar) recebe o estado e devolve o que o usuário escolheu, sem ler `pv`. Quem chama
   passa: título, contexto, vencimento de referência, textos da nota e do balão, rodapé (Concluir, Criar N eventos, Ativar recorrência ou Salvar) e o que o rodapé faz.
4. **O Avisar do Cartão é o terceiro uso, com Salvar.** O aviso padrão do cartão e o aviso de cada fatura abrem **só a tela Avisar**, com Salvar no rodapé, como hoje; a seta só desiste. Isso fica como
   está visualmente e passa pela mesma seção, sem ramos `pv.fatura` dentro dela.
5. **Posições do seletor por contexto:** Lançar tem as três (Só este, N vezes, Recorrente); Histórico e Futuros têm duas (N vezes e Recorrente), porque o lançamento já existe.
6. **Avisar em "Só este":** liga só com data futura; com data de hoje aparece desligado, com a dica da demo. O formulário só mostra isso; o servidor segue sem validar o aviso contra a data.
7. **"N vezes" no Lançar passa a ser "além deste"** (como no Prever). Hoje o Lançar pede o "Total de ocorrências (com este)". A demo e o Prever já usam "além deste"; o Lançar é alinhado a eles.
8. **Camadas do botão voltar:** a tela continua contando em `camadasAbertas()` e `fecharCamadaTopo()` (casca, subtela e janela de rodas). O `consultas-camadas.test.js` e o `voltar.test.js` provam.

## Fases (cada uma é um PR, uma branch)

- **Fase 1, limpeza sem mudar o visual:** `montarTc` nos quatro abridores, um `avisoInicial()` no lugar das quatro leituras do aviso salvo e fora o ramo morto `avisoVisivel.pv` / prefixo `pv` de `htmlAviso`.
  Testes de front iguais aos de hoje. Nada muda na tela.
- **Fase 2, componente interno, sem mudar o visual:** Repetir (três versões → uma, com parâmetros) e Avisar (cinco modos → entradas explícitas) em `web/js/repetir.js`; Prever, lote, Recorrência,
  fatura e aviso padrão do cartão passam a usá-los; some o estado global `pv`. A tela fica **igual às v1** (conferida lado a lado).
- **Fase 3, o seletor e o botão Repetir no Histórico e em Futuros** (demos `demo-repetir-v2-seletor` e `demo-historico-v2-repetir`): a tela ganha o seletor; Prever e Tornar recorrente viram o botão
  **Repetir**; ícone pilha em N vezes. Depende de decidir Futuros, lote e Estender (ver Pendências).
- **Fase 4, o Lançar** (demo `demo-lancar-v3-repetir`): a linha Repetir em "Mais"; saem `htmlRepeticao`, `htmlAviso`, `formRec`, `fRepetirWrap`, `fRecWrap` e `fAvisoMount`; `salvarLancamentoInterno`
  usa o `planoDeRepeticao`. É a fase mais arriscada (é a tela mais usada): conferir salvar, editar, copiar do Histórico, IA e Transferência.

## Pendências (o dono decide antes da fase que depende delas)

1. **Aviso no Lançar com data de hoje e Recorrente.** `POST /tx` com `recorrente` faz o servidor copiar o lançamento **inclusive o aviso** (`CopiaDeRecorrencia`, `backend/domain/recorrencia.go:124`,
   chamada por `Recorrencias.abrir` sem aviso explícito); já `marcarRecorrencia` (Tornar recorrente) põe o
   aviso só na primeira cópia e deixa o original sem aviso. Com data de hoje o lançamento original não deveria ter aviso. Saídas: salvar o lançamento e depois chamar `marcarRecorrencia` (duas chamadas, sem
   mexer no backend; precisa do `id` devolvido) ou aceitar o aviso no original. **A confirmar na fase 4.**
2. **Lote ("Tornar previstos"):** hoje tem a linha extra **Mês de destino**, o interruptor Repetir, o card de duplicados e o limite de 100. Não está nas demos. Proposta: usa a mesma tela, com as
   posições Só este (sem repetir) e N vezes, sem Recorrente, e a linha Mês de destino. **Precisa de demo antes da fase 3** ou fica com a tela atual até lá.
3. **Estender a série:** tem "Continuar a numeração das parcelas" e parte da última da série. Proposta: só N vezes, com a numeração como linha extra. **Precisa de demo ou confirmação.**
4. **Menu de ações de Futuros** (Prever, Estender, Tornar recorrente): a demo aprovada cobre só o Histórico. Proposta: o mesmo botão Repetir; **confirmar ou fazer demo curta.**
5. **Aviso de duplicados** ("Pular os que já existem" / "Criar mesmo assim"): continua como está, abaixo da nota. Só confirmar que cabe no desenho novo.

## Riscos

- **Lançar é a tela mais usada:** regressão em salvar. Mitigação: o `planoDeRepeticao` é função pura com teste para cada posição, e a fase 4 só entra depois das fases 1 a 3.
- **Mudar o "total" para "além deste" no Lançar** muda o significado de um número que o dono já usa. Mitigação: a nota da tela mostra as datas e o total, como no Prever.
- **O botão voltar:** tela cheia nova que não entra em `camadasAbertas()` fecha o app em vez da janela (já aconteceu). Mitigação: o teste de camadas cobre a casca e a subtela.
- **Eventos em lote por loop no navegador** (`criarEventos`): continua como hoje, com o mesmo tratamento de falha no meio ("Parou depois de N de M").

## Documentos a atualizar na entrega

`.ai/padrao-de-telas.md` (a seção "Repetir: uma tela só" passa de "ainda não implementada" a implementada, e a linha em "Armadilhas já encontradas"), `.ai/guardrails.md` (resumo de Telas),
`.ai/demos/` (a v2/v3 aprovada vira a padrão da tela; corrigir se houver desvio) e o `BACKLOG.md` (itens 47 e 42).
