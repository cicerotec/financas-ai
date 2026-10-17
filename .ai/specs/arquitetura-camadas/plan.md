# Plan: arquitetura em camadas (o que falta da raiz)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje. Cada fatia vira uma entrega própria
(branch, spec curto, PR, release); o app fica no ar entre elas.

**Estado:** rascunho para o dono aprovar.

## Onde o código está hoje (o que falta migrar)

| Arquivo da raiz | O que tem | Depende de |
|---|---|---|
| `store.go` | `initDB`, `getItem`, `papel`, `listarTx` e cursor de paginação | globais `db`, `table`, `ssmCli` |
| `main.go` | roteamento, `getMe`, `getTxs`, `getCfg`, `putCfg`, `putAparencia`, `salvarCfg`, `entrada` (HTTP e agendamento) | `db`, `table` |
| `tags.go` | adicionar tags, `garantirTagsServidor` | `db`, `table` |
| `seq.go` | contador de alterações (`proximoSeq`, `comSeq`, `getSeq`) | `db`, `table` |
| `perfil.go` | apelido, telegram, aviso de teste | `db`, `table`, `enviarMensagem` |
| `fechamento.go` | saldos, conferir/reabrir, `travas`, `invalidarSaldos` | `db`, `table`, `agora` |
| `avisos.go`, `telegram.go` | resumo diário, envio ao Telegram, token no SSM | `db`, `table`, `ssmCli`, `enviarMensagem` |
| `efeitos.go`, `dominio_compat.go` | pontes para o service de lançamentos | tudo acima |

## Desenho

- **Relógio:** interface `Relogio { Agora() time.Time }` declarada no `service`. Adaptador real e adaptador fixo para teste.
  O fuso de São Paulo e o `domain.MesFechadoPorData` continuam como estão (puros, recebem o instante).
- **Repositories por assunto** em `repository/`: `config.go` (`CFG#*`, aparência, tags, contador), `espacos.go` (`META`,
  papéis, espaços do usuário), `saldos.go` (`SALDO#*`), `avisos.go` (`AVISO#*`), além do `lancamento.go` e `pessoas.go` que
  já existem. **Nenhuma chave nova**: só o código que já grava com PK/SK passa a morar aqui.
- **Services por assunto** em `service/`: `Espacos`, `Config` (inclui tags e contador), `Perfil`, `Fechamento` (saldos e
  trava), `Avisos`. Cada um declara as interfaces de que precisa (`Repositorio…`, `Relogio`, `Notificador`).
- **Controllers** em `controller/` traduzem requests e erros, como o dos lançamentos. O roteador deixa de ter `case` com
  lógica: cada rota chama um método.
- **O telegram** vira um adaptador `telegram/` que implementa `service.Notificador` e recebe o cliente SSM e o HTTP por construtor.
- **A raiz** (`main.go`) fica com: ler variáveis de ambiente, criar clients, montar tudo, entregar o handler à Lambda e ao modo
  local. `entrada` (HTTP ou `{"acao":"avisos"}`) continua lá, só delegando.
- **Testes:** o `cenario()` passa a construir o conjunto com o `bancoFake`, um relógio fixo e um notificador falso, sem
  globais. Cada fatia converte os testes que tocam o seu assunto.

## Fatias (ordem sugerida)

| # | Fatia | Sai da raiz | Entrega |
|---|---|---|---|
| 1 | **Espaços e leitura de lançamentos** | `store.go` (`papel`, `getItem` dos espaços, `listarTx`, cursor), `getMe`, `getTxs` | `repository/espacos.go`, `service.Espacos`, `Lancamentos.Listar`; o `Relogio` e o montador único nascem aqui, e o `main` passa a injetar `db` e `table` |
| 2 | **Configurações, tags, contador e perfil** | `getCfg`, `putCfg`, `putAparencia`, `salvarCfg`, `tags.go`, `seq.go`, `perfil.go` | `repository/config.go`, `service.Config` e `service.Perfil`; some a ponte `GarantirTags` e `ComSeq` do `efeitos.go` |
| 3 | **Fechamento e saldos** | `fechamento.go` (saldos, conferir, reabrir, `travas`, `invalidarSaldos`) | `repository/saldos.go`, `service.Fechamento`; a trava passa a ser do service (some a ponte `Travas` e `InvalidarSaldos`) |
| 4 | **Avisos e Telegram** | `avisos.go`, `telegram.go`, `enviarMensagem` | `repository/avisos.go`, `service.Avisos`, adaptador `telegram/`; **sem** despachante (item 34); somem as últimas globais e `efeitos.go`/`dominio_compat.go` |

Depois da 4, `main.go` só liga, e o item 34 constrói o despachante em cima de `service.Avisos`.

**Por que essa ordem:** a 1 é a mais simples e cria o que as outras reutilizam (relógio, montador, injeção de `db`); a 2 é
mecânica (CRUD de configuração); a 3 é a de maior risco (trava de mês, usada por todo lançamento) e por isso vem com a
fundação já provada; a 4 depende do relógio e do telegram injetados e fecha as globais.

## Como cada fatia é entregue

1. Branch pela mantra, **spec curto** da fatia (arquivos que saem da raiz, interfaces novas, testes que provam que o
   contrato HTTP não mudou) aprovado pelo dono.
2. Implementação com os testes antigos convertidos (não apagados) e o `arquitetura_test.go` estendido às pastas novas.
3. `go vet`, `go test ./...`, e **comparação do contrato**: a coleção Postman e os testes de API rodam iguais antes e depois.
4. Deploy em dev, conferência no app, PR para `develop`; release quando o dono quiser.
5. `arquitetura.md` e o BACKLOG atualizados no mesmo PR (a raiz encolhe a cada fatia).

## Decisões para o dono (recomendação em negrito)

1. **Ordem das fatias:** **1 → 2 → 3 → 4**, como acima. Alternativa: fechamento (3) primeiro, por ser o mais ligado ao item 33.
2. **Avisos e Telegram:** **migrar na fatia 4, só com injeção, e deixar o despachante para o item 34**. Alternativa: tirar a
   fatia 4 daqui e fazê-la dentro do 34, mantendo `db`/`ssmCli` globais até lá.
3. **Relógio:** **interface `Relogio` no service, com adaptador real e fixo**. Alternativa: continuar passando `time.Time` por
   parâmetro em todo método.
4. **Um PR por fatia, com release entre elas** (**recomendado**) ou agrupar 1 e 2 num PR só, por serem pequenas.
5. **Nome dos pacotes novos:** `telegram/` para o adaptador de envio (**recomendado**) ou `notificacao/`, já pensando nos
   outros canais do item 34.
