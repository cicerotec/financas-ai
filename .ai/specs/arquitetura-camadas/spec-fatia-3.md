# Spec da fatia 3: fechamento e saldos

Terceira entrega do [`plan.md`](plan.md) do item 32 (fatias 1 e 2: [`spec-fatia-1.md`](spec-fatia-1.md) e
[`spec-fatia-2.md`](spec-fatia-2.md), no ar). **Refatoração: o contrato HTTP não muda.** É a fatia de maior risco do item 32,
porque a trava do mês (conferido e fechado por data) é consultada por **toda** escrita de lançamento. Por isso o spec lista,
além do que migra, as garantias que os testes já existentes dão e os que entram novos.

**Estado:** rascunho para o dono aprovar. Nenhum código foi alterado.

## Escopo

| Sai de | O que | Vai para |
|---|---|---|
| `fechamento.go` | `lerSaldo`, `atualizarSaldo` (ler, aplicar, gravar com até 5 tentativas), `skSaldo`, `ultimoConferido`, `versaoDe` | `repository/saldos.go` (acesso) e `domain/saldo.go` (`UltimoConferido`, `MesDeInicio`, `MesValido`) |
| `fechamento.go` | `travas` (`checar`, `mesDeInicio`, `anteriorAoInicio`) e `invalidarSaldos` | `service/fechamento.go` (`Fechamento.Travas` e `Fechamento.InvalidarSaldos`) |
| `fechamento.go` | `getFechamentos`, `putFechamentos`, `conferirMes`, `reabrirMes`, `conflitoVersao` e as validações | `service.Fechamento` (`Listar`, `SalvarCaches`, `Conferir`, `Reabrir`) e `controller/fechamento.go` |
| `fechamento.go`, `efeitos.go` | a variável `agora` e a função `mesAtual()` | interface `service.Relogio` e o pacote `adapter/relogio/` (`Real` e `Fixo`) |
| `main.go` | a ponte `salvarCfg` e `errConflito` (dominio_compat) | somem: o fechamento passa a usar `repository.Config.SalvarItem` direto |
| `efeitos.go` | `Travas` e `InvalidarSaldos` chamavam a raiz | passam a chamar `service.Fechamento`, injetado em `efeitosRaiz` |

**Fica na raiz de propósito** (migra na fatia 4): `getItem`, `marshal`, `key`, `spacePK`, `txSK` e as globais `db`, `table`,
`ssmCli`, `enviarMensagem` (usados por avisos e telegram), `SincronizarAviso` e `RemoverAviso` em `efeitosRaiz`, e o
`rodarAvisos(ctx, time.Now())` em `entrada`. **Some a global `agora`** (só o fechamento a usava); o Relogio entra nos avisos na
fatia 4.

## Contrato que não muda (exemplos, dados fictícios)

Um item por banco (`PK=SPACE#s1`, `SK=SALDO#ITAU`), com `version`, `caches` (saldo de fechamento de meses encerrados),
`base`, `invalidoDe` e `conferidos` (por mês: `saldo`, `real`, `em`, `por`, `ajuste` opcional).

```bash
curl -s "$API/spaces/s1/fechamentos" -H "authorization: Bearer $TOKEN"
```
```json
{ "itens": [ { "banco": "ITAU", "version": 5, "caches": { "2026-08": 1500.25 }, "base": "2026-01", "invalidoDe": "2026-09",
               "conferidos": { "2026-08": { "saldo": 1500.25, "real": 1500.25, "em": "2026-09-03T12:00:00.000Z", "por": "a1b2..." } } } ],
  "mesAtual": "2026-10" }
```
O `mesAtual` é o do fuso de São Paulo, pelo relógio do servidor. Falha: `500 {"erro":"erro interno"}`.

Gravar o cache (o cliente manda a `version` que leu **antes** de ler os lançamentos):

```bash
curl -s -X PUT "$API/spaces/s1/fechamentos" -H "authorization: Bearer $TOKEN" -H "content-type: application/json" \
  -d '{"banco":"ITAU","version":5,"base":"2026-01","caches":{"2026-08":1500.25,"2026-09":1700}}'
```
`200` devolve o item gravado (`version: 6`, sem `invalidoDe`). O mês em andamento (e os futuros) **não** entra no cache: é
descartado em silêncio. Erros: `400 banco obrigatorio`; `400 caches invalidos` (mês fora de `AAAA-MM`, valor não numérico,
`NaN` ou infinito); `409 {"erro":"alterado por outra pessoa","atual":{...}}` se a versão mudou (sem item: `"atual":{"version":0}`);
`500 erro interno` (leitura) ou `500 erro ao salvar`.

Conferir um mês (só o dono): trava o mês e os anteriores até reabrir.

```bash
curl -s -X POST "$API/spaces/s1/fechamentos/conferir" -H "authorization: Bearer $TOKEN" -H "content-type: application/json" \
  -d '{"banco":"ITAU","mes":"2026-09","saldo":1700,"real":1700,"version":6,"ajuste":0}'
```
`200` devolve o item com o mês em `conferidos` (`em` = o instante pelo relógio do servidor, `por` = o `sub`; `ajuste` só se diferente
de zero). Erros: `400 informe banco, mes (AAAA-MM), saldo calculado e saldo do extrato`; `400 so da para conferir o mes atual ou
meses anteriores`; `400 o saldo calculado nao bate com o extrato: lance o ajuste antes de conferir` (diferença de 0,005 ou mais);
`409` de versão (como acima); `409 {"erro":"o mes 2026-08 nao pode ser conferido: o ultimo conferido e 2026-09","codigo":"ordem"}`;
`500`.

Reabrir (só o mais recente):

```bash
curl -s -X POST "$API/spaces/s1/fechamentos/reabrir" -H "authorization: Bearer $TOKEN" -H "content-type: application/json" \
  -d '{"banco":"ITAU","mes":"2026-09"}'
```
`200` devolve o item sem o mês. Erros: `400 informe banco e mes (AAAA-MM)`; `404 esse mes nao esta conferido`;
`409 {"erro":"reabra primeiro o mes 2026-09 (o mais recente conferido)","codigo":"ordem"}`; `500 erro ao salvar`.

A **trava** dentro de cada escrita de lançamento (já testada nas fases anteriores) segue igual: conferido primeiro, depois o
fechado por data, com a isenção do que é anterior ao início do controle (`dataInicio` em `CFG#SALDOS`), `409 mes_conferido` e
`409 mes_fechado` (ver o spec do [fechamento automático](../fechamento-automatico/spec.md)).

## Desenho

- `adapter/relogio/relogio.go` (pacote novo dentro de `adapter/`, a pasta dos adaptadores que não são banco; não importa nada do projeto): `Real{}` (`Agora()` = `time.Now()`) e `Fixo` com `Em time.Time`
  e `Definir(time.Time)` para os testes. É um adaptador, no mesmo papel de `repository/`.
- `service.Relogio`: `interface { Agora() time.Time }`, declarada em `service/fechamento.go`.
- `domain/saldo.go`: `MesValido(string) bool` (o `reMes`), `MesAtual(agora time.Time) string` (fuso de São Paulo),
  `UltimoConferido(Doc) string`, `MesDeInicio(cfgSaldos Doc, banco string) string` (o que o `mesDeInicio` extrai de `CFG#SALDOS`).
  Todas puras, com teste em tabela.
- `repository/saldos.go` (novo): `Saldos{db, table, cfg *Config}` com `Ler(ctx, sid, banco)`, `Todos(ctx, sid)` (a query
  `begins_with(SK, "SALDO#")` de hoje) e `Salvar(ctx, sid, banco, body, esperada)` (via `Config.SalvarItem`, mesma condição de
  versão). **Nenhuma chave nova**: `SALDO#<banco>` como hoje. A leitura de `CFG#SALDOS` reaproveita `Config.Ler(sid, "saldos")`.
- `service/fechamento.go` (novo): `Fechamento{saldos RepositorioSaldos, cfg RepositorioConfig, rel Relogio}` com
  - `Listar`, `SalvarCaches`, `Conferir`, `Reabrir`: as regras e mensagens de hoje, devolvendo os erros do service (`ErroValidacao`
    para os `400`, `*ErroConflitoConfig` para os `409` de versão, `ErroOrdemFechamento{Msg}` para o `409` com `codigo:"ordem"`,
    `ErrMesNaoConferido` para o `404`, `ErrInterno`/`ErrSalvar` para os `500`);
  - `atualizar` (o ler-aplicar-gravar com 5 tentativas), privado;
  - `Travas(ctx, sid) Travas`: devolve um objeto **por chamada** que guarda, como hoje, o último conferido de cada banco e o início
    do controle (cada banco e `CFG#SALDOS` lidos uma vez só por chamada) e implementa `Checar(alvo, confirmado)` com o instante
    do `Relogio` e `domain.MesFechadoPorData`. A ordem, a isenção e o `exigiu` são exatamente os de `travas.checar`;
  - `InvalidarSaldos(ctx, sid, alvos)`: marca `invalidoDe` por banco e muda a `version`; falha só vai ao log, como hoje.
- `controller/fechamento.go` (novo): `GetFechamentos`, `PutFechamentos`, `PostConferir`, `PostReabrir`. O `respostaErro` ganha o
  `409` de `ErroOrdemFechamento` e o `404` de `ErrMesNaoConferido`.
- `servidor.go`: `novoServidor(db, table, rel service.Relogio)` cria o `Saldos`, o `Fechamento` e os injeta no controller e em
  `efeitosRaiz` (que deixa de chamar a raiz para travas e saldos). `main` passa `relogio.Real{}`.
- `arquitetura_test.go`: ganha o pacote `adapter/relogio` (não importa `financas/` nem AWS), e `service/fechamento.go` continua sem importar
  `time.Now` (o teste procura a chamada `time.Now(` em `service/`).

## Testes

| # | Prova | Como |
|---|---|---|
| F3-T1 | Todas as rotas de fechamento com as mesmas respostas, códigos e mensagens (as de cima): ler, gravar cache (inclusive descartar o mês atual e os futuros), conferir, reabrir | os testes de `fechamento_test.go` passam sem alterar as asserções; só o relógio muda (T7) |
| F3-T2 | Conflito de versão em `PUT`, `conferir` e `reabrir`; `409 ordem` nos dois sentidos; `404` ao reabrir mês não conferido | existentes + os que faltarem |
| F3-T3 | Validações: `caches` com `NaN`/infinito/mês inválido, `banco` vazio, mês futuro, diferença entre saldo e extrato (0,004 passa, 0,005 não) | existentes + borda do `0,005` |
| F3-T4 | A trava de lançamento: conferido absoluto, fechado por data com e sem confirmação, isenção do anterior ao início, leitura única de cada banco por chamada, mês exatamente no limite (dia 7 às 23h59 e dia 8 às 0h em SP) | `fechamento_auto_test.go` (A-T1 a A-T17) e `fechamento_test.go`, sem mudar asserções |
| F3-T5 | `InvalidarSaldos`: cada escrita marca `invalidoDe` com o menor mês do banco e muda a `version`; um cache calculado antes dela é recusado com `409`; falha ao invalidar não derruba o lançamento | existentes + falha injetada |
| F3-T6 | `domain.MesAtual`, `UltimoConferido`, `MesDeInicio`, `MesValido` | tabela em `domain/saldo_test.go` |
| F3-T7 | O relógio é injetado: o `cenario` monta o `Servidor` com um `relogio.Fixo` compartilhado; `relogioEm` e o relógio padrão (05/02/2026) passam a ajustá-lo; **não existe mais** a variável global `agora` | teste de inspeção do fonte + os testes acima rodando |
| F3-T8 | A raiz perdeu o fechamento: `fechamento.go` não existe; `main.go` e `efeitos.go` não têm `salvarCfg` nem `novasTravas` | inspeção, como os F1-T6 e F2-T7 |
| F3-T9 | Mutação: trocar a ordem (fechado antes do conferido), tirar a isenção do início do controle e usar `time.Now()` no lugar do relógio derrubam os testes certos | verificada à mão |

## Risco e como o spec o controla

- **A trava é transversal**: um erro aqui deixa passar edição em mês conferido, ou recusa uma legítima. Por isso nenhuma asserção
  dos testes de trava muda, e os de A-T1 a A-T17 e `fechamento_test.go` são a rede.
- **O relógio** já é parâmetro de `MesFechadoPorData`; só a origem do instante muda (da global para o `Relogio`). O instante lido é
  um por chamada de `Checar`, como hoje.
- **Sem migração de dados**: os itens `SALDO#*` e `CFG#SALDOS` são os mesmos.

## Pronto quando

- `go vet ./...` e `go test ./...` passam; `sam validate --lint` passa (o `template.yaml` não muda).
- A coleção Postman roda igual contra dev nas quatro rotas de fechamento.
- No app de dev: abrir Saldos, conferir um mês, tentar editar um lançamento daquele mês (recusa), reabrir o mês e editar; editar
  um lançamento de mês fechado por data (pede confirmação) e ver o saldo recalcular (o cache é invalidado).
- `fechamento.go` não existe mais; `agora`, `salvarCfg` e `errConflito` também não.
- `arquitetura.md` e o item 32 do BACKLOG atualizados no mesmo PR.

## Decisões para o dono (recomendação em negrito)

1. **O `Relogio` mora em `adapter/relogio/` (adaptadores `Real` e `Fixo`) e a interface em `service/`.** (Ajustado pelo dono em 2026-10-17: os adaptadores que não são banco ficam em `adapter/`; o banco continua em `repository/`.)
2. **A global `agora` e as pontes `salvarCfg`/`errConflito` somem já nesta fatia.** O `entrada` mantém `time.Now()` para os avisos até
   a fatia 4, quando o `Relogio` chega neles.
3. **A trava continua sendo um objeto por chamada** (cache de banco e de `CFG#SALDOS` dentro da requisição), em vez de um serviço
   com cache compartilhado entre requisições: o Lambda reaproveita a memória, e cache entre requisições serviria dado velho.
