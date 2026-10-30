# Plan: recorrência automática de lançamentos (item 51)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/service/avisos.go`, `repository/avisos.go`, `domain/aviso.go`, `service/lancamento.go`,
`web/recorrencia.js`, `web/js/app.js`). O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** rascunho para aprovação (2026-10-29). Este plan **não cria chave nem tipo de registro novo**; o que ele propõe de dado novo (o atributo `recorrente` e o item auxiliar da recorrência) está em
"Propostas que o dono pode contestar", porque a regra do projeto é discutir antes de mexer em campo ou item do DynamoDB.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| Cada lançamento com `aviso` tem um **item auxiliar** em `PK = AVISOS#ATIVOS`, `SK = <sid>#<id>`; a fatura do cartão tem outro, do `tipo: "fatura"`, **sem lançamento por trás** | `service.Avisos.Sincronizar`, `SincronizarFatura`, `repository/avisos.go` | A recorrência é mais um `tipo` na **mesma partição**, sem chave nova. Já há precedente de item auxiliar que não é espelho de lançamento |
| **Não existe índice secundário** na tabela: o `AVISOS#ATIVOS` existe justamente para a rotina diária não varrer a tabela | `repository/avisos.go` | Um atributo no lançamento sozinho não seria achado sem varredura; quem comanda a criação é o item auxiliar |
| A rotina diária faz **uma Query** em `AVISOS#ATIVOS` e decide pelas datas; só então lê o lançamento (`Buscar`), e o `Buscar` devolve também o **excluído** (`excluidoEm`), que a rotina descarta | `Avisos.Rodar` | A rotina de recorrência entra no mesmo evento `{"acao":"avisos"}` e no mesmo horário (08:00 em São Paulo), sem agendamento novo |
| **Criar** um lançamento não grava versão no `HIST#`; só alterar e excluir gravam (R4 do histórico) | `repository.Lancamentos.Salvar` | A cópia criada pela rotina é uma gravação só, sem histórico |
| Excluir pelo app apaga o item de aviso do lançamento (`RemoverAviso`) e deixa o lançamento com `excluidoEm` | `service/lancamento.go` | O item da recorrência é **outro item** (outra `SK`): excluir uma cópia não mexe nele, e é isso que faz o "pular o mês" funcionar |
| As cópias do Prever levam `status`, `tipo`, `valor`, `descricao`, `nota`, `dataEvento`, `banco`, `tags`, `aviso` e `serie`; dia que não existe no mês vira o último, sempre a partir do dia base | `web/recorrencia.js` (`montarCopias`, `gerarDatas`) | A cópia da recorrência leva os mesmos campos, **menos `serie`**, mais a marca. A conta de datas é a mesma e precisa existir no servidor (Go) |
| O servidor marca como previsto todo lançamento de data futura; a cópia leva o status da origem | `service/lancamento.go`, `domain` | Nenhuma tela pergunta status |
| Escrever lançamento passa pelas permissões `tx.criar` e `tx.editar` e pela trava do mês fechado | `auth/authz.go`, `Travas.Checar` | A recorrência usa as mesmas permissões; as cópias são de data futura, portanto livres da trava |

## Desenho

- **No lançamento (`TX#`):** `recorrente: { intervalo }` (inteiro de 1 a 120, em meses), um objeto como `serie` e `aviso`. Serve para a tela mostrar e para ligar e desligar. As cópias nascem com o
  mesmo objeto. O item auxiliar guarda o intervalo também, porque é ele que a rotina lê: ela não precisa abrir o lançamento só para saber de quanto em quanto tempo copiar.
- **No `AVISOS#ATIVOS`:** um item por recorrência, `SK = <sid>#recorrencia#<id do primeiro registro>` (estável, não muda a cada mês), com `tipo: "recorrencia"`, `id`, `ultimoId` (a ponta: o último
  registro criado), `dataUltimo` (a `dataEvento` dele, que é parte da chave do `TX#`), `proxima` (o dia ISO da próxima cópia), `intervalo` (meses, 1 a 120, como `serie.intervalo`) e `dia` (o dia do mês
  original, para o 31 não virar 28 para sempre). O item **não é espelho**: é a única fonte da regra, e por isso entra no backup.
- **Datas:** a ocorrência seguinte é o mês de `proxima` mais `intervalo` meses, no dia `dia` (ou o último do mês), com a hora do último registro. A conta é a mesma de `gerarDatas`, escrita em Go e testada
  em tabela (31/01 mensal: 28/02, 31/03, 30/04; 29/02 anual: 28/02/2029 e volta a 29/02 em 2032).
- **Rotina:** depois dos avisos, no mesmo evento, `Avisos.Rodar` percorre os itens `tipo: "recorrencia"` que a mesma Query já trouxe. Para cada um: se `proxima` está **no passado** (hoje depois dela),
  avança sem criar até `proxima` ≥ hoje; se `proxima − hoje ≤ 30 dias` (`JanelaRecorrencia`, constante em `domain/`, como o dia 7 do fechamento), cria a cópia.
- **Criar a cópia:** lê o último registro (`Buscar(dataUltimo, ultimoId)`, que devolve também o excluído) e monta a cópia (campos acima, sem `serie`, com o mesmo `recorrente`, `criadoPor` do último). Entra
  pelo **caminho normal de criação** do service (validação, aviso, `Sincronizar` do aviso da própria cópia), sem versão no histórico. O `id` da cópia é **derivado** da recorrência e da data
  (`hash(id da recorrência, proxima)`, no formato de `ReID`), e a gravação é condicional (`attribute_not_exists`): rodar duas vezes, ou rodar depois de uma falha no meio, **não duplica**.
- **Avançar:** só depois de criada (ou de a cópia já existir), a rotina atualiza o item (`ultimoId`, `dataUltimo`, `proxima`). Se cair entre as duas, a próxima rodada tenta criar de novo, a condição acusa que já
  existe e ela só avança. Apagar a cópia depois (exclusão lógica) não desfaz nada: a `proxima` já está adiante, e o mês fica pulado.
- **Se o último registro não existe mais** (apagado direto no banco): o item é removido e a recorrência para, com uma linha no log. Excluído pelo app não é esse caso: ele continua lido.
- **Desligar:** `DELETE` do item auxiliar e tira o `recorrente` do registro vivo (uma alteração do dono, que grava versão como qualquer edição). Cópias futuras já criadas ficam como estão.
- **Religar:** marca de novo o lançamento mais recente com um novo item; `proxima` é calculada **a partir de hoje, sempre para frente**, sem recuperar os meses parados.
- **Marcar:** na criação (formulário) ou num lançamento existente (uma alteração: grava a versão do que mudou). O servidor valida o `intervalo`, recusa transferência, e cria o item na mesma operação.
- **Permissões:** quem pode criar ou editar lançamento pode marcar e desligar; a rotina escreve como o servidor, com o `criadoPor` do registro de origem.
- **Limite por rodada:** no máximo 50 cópias por execução (5 WCU, uma gravação de cada vez); o que sobrar vai para o dia seguinte. Com cerca de 20 recorrências por mês, nunca chega perto.
- **Desempenho:** dia sem recorrência a vencer = nenhuma leitura além da Query de `AVISOS#ATIVOS`, que já existe. Criar uma cópia = 1 leitura (o último registro), 1 gravação do lançamento, 1 do
  aviso dela (se tiver) e 1 do item.

## API (proposta)

```
POST   /spaces/{sid}/recorrencias        { "id": "<lançamento>", "intervalo": 1 }   -> 201 { id, intervalo, proxima }  (marca o lançamento e cria o item)
DELETE /spaces/{sid}/recorrencias/{id}                                              -> 200  (desliga: apaga o item e tira a marca do registro)
GET    /spaces/{sid}/recorrencias                                                   -> 200 [{ id, ultimoId, intervalo, proxima }]  (as ativas)
```

Ao criar um lançamento (`POST /tx`) com `recorrente: { intervalo }`, o item nasce na mesma operação (o formulário não faz duas chamadas). A leitura de lançamentos devolve `recorrente`.

## Fases (cada uma é um PR)

- **A (backend):** `ValidarRecorrencia` (intervalo e transferência), item auxiliar no `AVISOS#ATIVOS`, rotina de recorrência no evento `avisos`, rotas, constante da janela, testes. Só acrescenta; o app antigo
  não envia `recorrente` e nada muda para ele.
- **B (tela):** demo primeiro, no padrão do Prever e do Repetir (as rodas do intervalo, a lista e o interruptor), com o 🔁 na linha, a lista de recorrências ativas e o botão de parar. A tela não decide
  nada: o servidor calcula a próxima data e diz quais lançamentos são recorrentes.

O release para a `main` só depois da A **e** da B, porque o backend sozinho não dá como marcar.

## Propostas que o dono pode contestar

1. **Decidido pelo dono (2026-10-29):** `recorrente` no `TX#` é um **objeto** `{ intervalo }`, como `serie` e `aviso`. A próxima data e o último registro ficam só no item do `AVISOS#`, que repete o intervalo.
2. O item da recorrência é **estável** (SK pelo primeiro registro) e aponta para a ponta em `ultimoId`, em vez de ser recriado a cada mês.
3. A cópia tem **id derivado** da recorrência e da data, para a criação ser idempotente.
4. **Janela de 30 dias**, constante, igual para todas.
5. Marcar um lançamento que **já existe** é uma alteração e grava uma versão no histórico; marcar na criação, não.
6. A rota `GET /recorrencias` e a lista na tela: sem elas não há como saber o que está ativo para desligar.
7. Limite de 50 cópias por rodada.
8. As cópias herdam o `recorrente`; só a ponta (`ultimoId`) tem item auxiliar. A marca nas cópias antigas é só informação.

## Riscos

| Risco | Resposta |
|---|---|
| A rotina cria e cai antes de avançar o item | Id derivado + gravação condicional: a rodada seguinte só avança |
| Item auxiliar sumir (apagado à mão) | A recorrência para; é a única fonte da regra, então entra no backup e no modelo de dados |
| Esquecer uma recorrência ligada | A lista de ativas na tela e o botão de parar; a cópia aparece em Futuros 30 dias antes |
| Dia 31 derivar para 28 | O item guarda o `dia` original e a conta parte dele |
| Muitos lançamentos criados de uma vez | Limite de 50 por rodada e 5 WCU: o resto vai para o dia seguinte |
| Excluir pelo app apagar a recorrência | Não apaga: o item é outro, com outra `SK`, e a exclusão só mexe no aviso do lançamento |
