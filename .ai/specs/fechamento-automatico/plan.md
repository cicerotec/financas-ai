# Plan: fechamento automático do mês anterior

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/service/`, `fechamento.go`,
`efeitos.go`, `web/js/app.js`). O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** rascunho de 2026-10-16, aguardando o ok do dono. Nada foi implementado.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| Toda escrita de lançamento pergunta à trava antes de gravar: `Travas.Checar(alvo)` | `service/lancamento.go`, `service/restaurar.go` | A regra nova entra **no mesmo ponto**, e vale para criar, editar, excluir, importar e restaurar |
| A trava do conferido mora em `travas.checar`, e o service só fala com a interface `Travas` | `fechamento.go`, `efeitos.go` | A interface ganha o sinal de confirmação; o service continua sem conhecer o relógio |
| A edição só chama a trava quando `MudaSaldo` | `service/lancamento.go` | O "só o que mexe no saldo" já existe e é reaproveitado |
| `anteriorAoInicio` já isenta o que é anterior ao início do controle | `fechamento.go` | A isenção é a mesma da trava do conferido |
| `agora()` é uma variável da raiz que os testes trocam | `fechamento.go` | A regra do dia 7 usa `agora()` na raiz, não `time.Now` no service |
| O front mostra `msgErro` para `mes_conferido` e reverte | `web/js/app.js` | O front precisa de um caminho novo para o `409` confirmável |
| A exclusão só chama o servidor depois dos 6 s (Desfazer) | `web/js/app.js` (`iniciarExclusao`) | Muda na fase C |

## Desenho

- **Regra pura em `domain/`:** `MesFechadoPorData(mes string, agora time.Time) bool`. O mês M é **aberto** se for o atual, um futuro,
  ou o anterior ao atual enquanto o dia do mês em São Paulo for ≤ 7; senão é fechado. Testada em tabela, com as bordas (dia 7 às
  23h59 e dia 8 às 0h em SP, o mesmo instante em UTC, virada de ano).
- **Erro novo `domain.ErroMesFechado{Banco, Mes}`**, irmão do `ErroMesConferido`, mas **confirmável**.
- **`Travas.Checar(alvo, confirmado bool)`** (interface do service, implementada em `efeitos.go`/`fechamento.go`): primeiro o
  conferido (absoluto, ignora a confirmação), depois, se `!confirmado`, o mês fechado por data. Devolve `*ErroMesConferido`,
  `*ErroMesFechado` ou nil. Quem usa a confirmação também precisa saber se ela foi **necessária** (para gravar na versão): a
  interface devolve `(exigiu bool, err error)`.
- **A confirmação viaja na query:** `?confirmaMesFechado=1`. Não vai no corpo, porque o `DELETE` e o `restaurar` não têm corpo e
  porque assim ela **nunca chega ao item vivo** (não há o que proteger em `protegidos`). O controller lê da query e passa ao service.
- **`409` próprio:** `{"erro": "...", "codigo": "mes_fechado", "banco": ..., "mes": ..., "confirmavel": true}`. O `mes_conferido`
  continua como está (sem `confirmavel`).
- **Registro na versão:** a `Versao` ganha `ConfirmouMesFechado bool`; o repository grava o atributo `confirmouMesFechado: true`
  **só** na versão em que a confirmação foi necessária e dada (nunca no item vivo). `/versoes` e `/historico` devolvem o booleano.
- **Importação:** o lote leva a confirmação para todos os itens; sem ela, cada item em mês fechado vai para `rejeitados` com
  `codigo: "mes_fechado"` (como já acontece com o conferido), e o resto grava.
- **Restauração:** `/versoes` ganha `exigeConfirmacao` por versão (a restauração cairia em mês fechado), para a tela avisar antes
  de enviar. Continua `restauravel: true`: confirmar é permitido.
- **Excluir imediato:** o `DELETE` passa a devolver, além de `ok`, `desfazer: {"versao": N}` (a versão de antes da exclusão), e
  o Desfazer da tela chama a restauração dessa versão.

## Fases (cada uma é um PR; o release para a `main` só depois da B)

O backend sozinho faria o app recusar edição do passado **sem a tela saber confirmar**. Por isso o PR de release `develop → main`
**não deve ser mergeado entre a A e a B**. (Alternativa, se preferir uma trava de segurança: um parâmetro `DiaFechamento` no
template que desliga a regra enquanto for 0. Não está no spec; é só dizer.)

- **Fase A, backend:** regra de domínio, `Travas.Checar` com confirmação, query `confirmaMesFechado`, `409 mes_fechado`, registro
  na versão, `exigeConfirmacao`, `desfazer` no `DELETE`, testes (o relógio do `cenario` passa a ser fixo) e docs (modelo de
  dados, security, spec do histórico).
- **Fase B, tela da confirmação:** demo primeiro. Janela de confirmação ao receber `mes_fechado` (criar, editar, importar,
  restaurar), com reenvio. Usa o padrão de janela de confirmação que já existe.
- **Fase C, excluir imediato:** demo primeiro (muda o padrão 11b). `DELETE` na hora; recusa antes de o item sumir; Desfazer =
  restauração; fim do "pendente" no servidor.

## Riscos

| Risco | Resposta |
|---|---|
| Release só com a A trava a edição do passado no app | A B vem antes do release; o plan avisa e a descrição do PR repete |
| Os testes atuais usam datas de janeiro de 2026 com o relógio real (outubro) e passariam a pedir confirmação | `cenario` fixa o relógio (uma data de fevereiro de 2026); é ajuste do ambiente de teste, não de expectativa |
| Importar uma planilha antiga pede a confirmação de uma vez | O lote leva a confirmação; a tela pergunta uma vez ("Importar lançamentos de meses anteriores?") |
| `anteriorAoInicio` depende da config de saldos, que pode não existir | Sem `dataInicio`, nada é isento: o passado todo exige confirmação |
| Confirmar vira hábito e perde o sentido | A confirmação fica gravada na versão (quem, quando): dá para auditar |

## Decisões que o plan fixou e o dono pode contestar

1. A confirmação é um **parâmetro de query** (`confirmaMesFechado=1`), não campo do corpo.
2. O código do erro é **`mes_fechado`**, com `confirmavel: true`.
3. O **dia 7 é inclusivo**: no dia 7 o mês anterior ainda é livre; no dia 8, às 0h de São Paulo, fecha.
4. A ordem é **conferido primeiro, depois fechado por data**; o conferido nunca é destravado por confirmação.
5. O registro na versão só existe quando a confirmação **foi necessária** (`confirmouMesFechado: true`); confirmar sem precisar não grava.
6. Importar: a confirmação vale para o lote inteiro.
