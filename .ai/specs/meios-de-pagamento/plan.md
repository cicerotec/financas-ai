# Plan: meios de pagamento (criar, desativar, remover)

Segue o [`intent.md`](intent.md); o roteiro verificável está no [`spec.md`](spec.md).

**Estado:** aprovado pelo dono em 2026-11-03 (inclui o `ativoAte` em `CFG#LISTAS`); fase A (backend) implementada no #347; fase B (tela) implementada em 2026-11-03, à espera da conferência do dono em dev.

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
