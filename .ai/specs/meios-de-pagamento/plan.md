# Plan: meios de pagamento (criar, desativar, remover)

Segue o [`intent.md`](intent.md); o roteiro verificável está no [`spec.md`](spec.md).

**Estado:** aprovado pelo dono em 2026-11-03 (inclui o `ativoAte` em `CFG#LISTAS`); fases A (#347) e B (#348) concluídas em 2026-11-03 e conferidas em dev pelo dono.

## Decisões

1. **`ativoAte` em `CFG#LISTAS`**, um mapa `{ nome: "AAAA-MM-DD" }`, ausente = ativo. Significa **a data em que o meio deixa de valer**: `ativo = hoje < ativoAte`
   (o "hoje" é o do servidor, fuso de São Paulo). Não vai cru ao front: `GET /cfg/listas` continua devolvendo `meios: [{nome, cartao, ativo}]` e ganha `ativoAte` por meio
   (para a tela escrever "Inativo desde…"). A lista `banco` continua `[]string`.
2. **O servidor protege o campo.** `CFG#LISTAS` é gravada inteira pelo front; o `ativoAte` passa a ser preservado como os campos de status (`statusDeListas` em
   `service/config.go`): o corpo do `PUT` é ignorado nele. Pelo mesmo motivo, o `PUT` de Listas **não pode mais tirar um nome de `banco`** (senão a regra de remoção
   seria contornada por uma requisição forjada); tirar passa só pela rota de remover.
3. **Duas rotas novas**, só do dono: `PUT /spaces/{sid}/meios/{meio}` (`{ ativoAte: "AAAA-MM-DD" | null }`) e `DELETE /spaces/{sid}/meios/{meio}`. Ação nova em
   `auth/authz.go`, com `SoDono` (nome a confirmar). **Adicionar** continua pelo `PUT` de Listas (acrescentar nome não muda).
4. **Remover = varredura com parada no primeiro achado**, só ao clicar. Usa `Lancamentos.Listar` com `Banco` e `Limite: 1`, que já filtra em memória e
   para no primeiro achado, lendo em páginas de 500 itens. O ponto fraco é o meio **realmente sem uso**: lê a partição inteira, uma vez. O servidor **grava no log**
   `meios: remover meio= paginas= itens= ms=` para ver o custo; é o gatilho de troca para a flag `jaUsado` (BACKLOG, Ideias).
5. **Lançamentos excluídos logicamente contam como uso.** `Listar` hoje os pula; a conferência precisa vê-los (campo novo no filtro, `ComExcluidos`, sem chave nova),
   porque restaurar uma versão traria de volta um lançamento num meio que não existe mais. O histórico (`HIST#`) guarda só o nome.
6. **Meio ligado a um cartão não se remove daqui:** se existe em `CFG#CARTOES.porBanco`, o servidor recusa e diz para tirar o cartão na aba Cartão antes.
7. **Sugestão no front:** Lançar e Transferência listam só os `ativo`; o meio de um lançamento que está sendo **editado** continua na lista mesmo inativo (senão o combo
   troca de valor sozinho, como no bug 25). Histórico, Filtros, Saldos e Tendências usam todos. A tela só lê `ativo`, sem comparar datas (o front não decide).
8. **Corrida entre a conferência e um lançamento novo:** a varredura e a gravação de `CFG#LISTAS` não são uma transação. Risco aceito: só o dono remove, e é raro lançar
   no mesmo instante num meio que nunca foi usado. A flag `jaUsado` fecha essa janela (BACKLOG, Ideias).

## Fases

- **A, backend:** `ativoAte` em `CFG#LISTAS` (leitura derivada, proteção no `PUT`), rotas `PUT` e `DELETE` de meio, conferência de uso, log de custo, `authz`, testes.
- **B, tela** (demo [`demo-listas-v3-meios`](../../demos/demo-listas-v3-meios.html) aprovada): card Meios de pagamento em Listas, janela de ações, Desativar com data, Reativar, Remover
  chamando o servidor, combos de Lançar e Transferência só com ativos. Sai o `removerBanco` do navegador. Sem desvio da demo sem avisar antes do PR.

## Riscos

- **Custo da varredura:** meio sem uso lê tudo. Mitigação: só ao clicar, uma vez, log do custo e a flag `jaUsado` pronta em Ideias.
- **O `PUT` de Listas deixa de poder tirar bancos:** conferir tudo no front que remove nome de `banco` (hoje só o `removerBanco`) e a importação de configurações,
  que não pode apagar nem recriar o `ativoAte`.
- **Exportar e importar configurações:** o `ativoAte` entra no JSON exportado de Listas; a importação o aceita só pelo caminho do servidor (a conferir no spec da fase A).
- **Combo na edição:** testar o meio inativo num lançamento antigo (Lançar em edição e Confirmar de um previsto).

## Documentos a atualizar na entrega

`.ai/specs/modelo-de-dados/` (exemplo de `space-cfg-listas.json` e o README), `.ai/arquitetura.md` (a linha de "meio de pagamento" e o `ativo`), `fatura-servidor/spec.md` (o `ativo` deixa de ser sempre verdadeiro) e `.ai/padrao-de-telas.md`.

## Fase C: encerrar o meio (item 58)

Decidido pelo dono em 2026-11-08 (ver o item 58 do [`BACKLOG.md`](../../../BACKLOG.md#item-58) e as regras R12 a R20 do [`spec.md`](spec.md)). O servidor passa a decidir; o front só exibe (demo antes).

- **Uma guarda só.** `service/` ganha a leitura de `CFG#LISTAS` (+ `ativoAte`) por chamada, usada por `criar`, `editar`, `Excluir`, `Confirmar`, `Restaurar`, `Importar` e `Transferir`. `CriarCopia` (recorrência),
  o ajuste da conferência e o pagamento de fatura passam por `criar`, então herdam a recusa.
- **Encerramento** é um service novo que só lê (saldo, lançamentos do meio, recorrências, cartões) e devolve a lista de pendências; `Meios.Definir` chama antes de gravar.
- **Sem dado novo no DynamoDB.** `encerrado` é derivado na leitura de `/saldos` e `/cartoes`.
- **Fases:** C1 backend (este plano); C2 telas, **depois da demo**: a marca "desativado" no Histórico e no filtro de meio, a janela Desativar só com "A partir de hoje" e a explicação das pendências, e o Saldos/Cartão
  escondendo o que o servidor marcou `encerrado`.

### Premissas (preencher com a revisão independente antes da aprovação)

Perguntas que a revisão independente responde por escrito: *"que regra desta fase só funciona por causa de outra, e o que acontece se essa outra deixar de valer?"*

Revisão independente feita em 2026-11-08 por um agente sem o contexto de quem escreveu (lendo o spec e o código). O que ela achou e o que foi feito:

1. **Comparação da edição (R15) dava falso positivo** com o formulário real (`null` onde o gravado não tem o campo, data sem milissegundos). **Corrigido:** comparação por equivalência; testes em `domain/meios_test.go` e `TestEditarLancamentoDeMeioDesativado`.
2. **Importar era porta de ressurreição e de troca de meio** (sobrescreve o `id` existente, inclusive excluído). **Corrigido:** R16 vale também para o registro gravado (`importavelSobre`); teste M-T25.
3. **A ponta da recorrência pode estar excluída** (a rotina continua copiando dela) e a recusa deixava a rotina tentando todo dia. **Corrigido:** a conferência conta a ponta excluída; a rotina encerra a recorrência (R14b; `TestRecorrenciaDeMeioDesativadoEncerra`).
4. **Previsto oculto, antigo ou de ano remoto** não podia escapar da conferência. **Garantido:** a conferência varre o meio inteiro por `Listar` (não usa a leitura de Futuros); teste com previsto oculto de 2019.
5. **Fatura (R13d):** a conferência dependia de Futuros (só cartão com vencimento). **Corrigido por decisão do dono (2026-11-08):** confere direto as três últimas faturas (`Cartoes.EmAberto`), com ou sem vencimento; valor maior que zero sem pagamento registrado bloqueia e a frase diz fatura e valor. Fatura mais antiga que as três é só papel e não conta. Teste M-T31. O `encerrado` do cartão em `GET /cartoes` continua olhando só a fatura atual (só esconde na tela; a desativação já foi conferida).
6. **O pagador** passa pelas duas rotas (`validarPagador` já é chamada por `PUT /cartoes/{meio}` e por `PUT .../faturas/{ym}`). **Coberto.** Transformar um meio desativado em cartão (`Adicionar`) agora é recusado (R22, M-T33).
7. **O saldo zero depende de `CFG#SALDOS` e do cache.** `PUT /cfg/saldos` não muda o saldo inicial nem a `dataInicio` de meio desativado (R21, M-T32). O item de Saldos com `erro` nunca vira `encerrado` (o código só marca no ramo sem erro).
8. **Corrida** entre a conferência e a gravação do `ativoAte` (alguém lança no meio). **Risco aceito**, como no Remover: só o dono desativa e a saída é Reativar.
9. **Dados antigos com `ativoAte` futuro** passariam a encerrar sozinhos na data, sem a conferência. **Resolvido:** o dono informou (2026-11-08) que não tem meio desativado; nada a conferir.
10. **O meio `Teste`:** como excluir também é recusado em meio desativado, a ordem é reativar, excluir o lançamento e remover.
11. **Guarda de meio fora da lista em todas as portas** (criar, importar, transferir): **não adotada nesta fase.** Muitos dados e testes têm `banco` fora de Listas e o backup importa nomes de outros espaços; só o Restaurar confere a lista. **Decidido pelo dono (2026-11-08): fica assim**, sem item no backlog. Backup (item 11) é outro assunto e restaura como foi guardado.
12. **Front (fase C2, depois da demo):** "A partir de hoje" usa a data do aparelho e o servidor exige a dele (perto da meia-noite dá 400); "Fim do mês" e "Outra data" passam a dar 400 se o C1 for publicado sozinho; `refazer` mantém o meio já escolhido mesmo desativado; Pagar fatura oferece o pagador sugerido desativado; Saldos itera `listas.banco` e conta "N bancos" sem tratar `encerrado`; um meio com `ativoAte` futuro mostra "Reativar". **Fase C2 (2026-11-08, demos aprovadas pelo dono e implementadas):** "A partir de hoje" pede `ativoAte: "hoje"` e o servidor usa o dia dele; "Fim do mês" e "Outra data" saíram; `refazer` só mantém o meio desativado ao editar; Pagar fatura só oferece meio ativo; Saldos e Cartão escondem o `encerrado`; o Lançar editando trava tudo menos descrição, nota e tags. **Futuros:** o cache das faturas não precisa acompanhar `CFG#LISTAS`, porque um meio desativado não tem fatura a pagar (a conferência bloqueia). **Por isso o C1 não vai sozinho para produção.**
