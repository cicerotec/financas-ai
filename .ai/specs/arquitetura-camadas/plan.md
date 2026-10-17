# Plan: arquitetura em camadas (o que falta da raiz)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje. Cada fatia vira uma entrega própria
(branch, spec curto, PR, release); o app fica no ar entre elas.

**Estado:** as quatro fatias estão implementadas (a 4, avisos e Telegram, esperando a conferência em dev). Specs: [`spec-fatia-1.md`](spec-fatia-1.md), [`spec-fatia-2.md`](spec-fatia-2.md), [`spec-fatia-3.md`](spec-fatia-3.md) e [`spec-fatia-4.md`](spec-fatia-4.md).

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
- **Os canais de envio** moram em `adapter/notificacao/`, um arquivo por canal (`telegram.go` agora; `email.go` e `whatsapp.go` quando
  existirem), como `repository/` tem um arquivo por assunto. Cada um implementa `service.Notificador` e recebe o que precisa
  (cliente SSM, HTTP) por construtor. **O despachante** (quem recebe, o quê, por qual canal) é regra e vive no `service`, no item 34;
  por isso a pasta `dispatcher/` do `arquitetura.md` deixa de ser necessária e esse documento é ajustado quando o 34 começar.
- **A raiz** (`main.go`) fica com: ler variáveis de ambiente, criar clients, montar tudo, entregar o handler à Lambda e ao modo
  local. `entrada` (HTTP ou `{"acao":"avisos"}`) continua lá, só delegando.
- **Testes:** o `cenario()` passa a construir o conjunto com o `bancoFake`, um relógio fixo e um notificador falso, sem
  globais. Cada fatia converte os testes que tocam o seu assunto.

## Fatias (ordem sugerida)

| # | Fatia | Sai da raiz | Entrega |
|---|---|---|---|
| 1 | **Espaços e leitura de lançamentos** | `store.go` (`papel`, `getItem` dos espaços, `listarTx`, cursor), `getMe`, `getTxs` | `repository/espacos.go`, `service.Espacos`, `Lancamentos.Listar`; o montador único (`Servidor`, montado uma vez em `main`) nasce aqui; `db` e `table` seguem globais para o resto da raiz até a fatia 4 |
| 2 | **Configurações, tags, contador e perfil** | `getCfg`, `putCfg`, `putAparencia`, `salvarCfg`, `tags.go`, `seq.go`, `perfil.go` | `repository/config.go`, `service.Config` e `service.Perfil`; some a ponte `GarantirTags` e `ComSeq` do `efeitos.go` |
| 3 | **Fechamento e saldos** | `fechamento.go` (saldos, conferir, reabrir, `travas`, `invalidarSaldos`) | `Relogio` (interface e adaptadores real e fixo), `repository/saldos.go`, `service.Fechamento`; a trava passa a ser do service (some a ponte `Travas` e `InvalidarSaldos`) |
| 4 | **Avisos e Telegram** | `avisos.go`, `telegram.go`, `enviarMensagem` | `repository/avisos.go`, `service.Avisos`, adaptador `adapter/notificacao/telegram.go`; **sem** despachante (item 34); somem as últimas globais e `efeitos.go`/`dominio_compat.go` |

Depois da 4, `main.go` só liga, e o item 34 constrói o despachante em cima de `service.Avisos`.

**Por que essa ordem:** a 1 é a mais simples e cria o que as outras reutilizam (o montador único); a 2 é
mecânica (CRUD de configuração); a 3 é a de maior risco (trava de mês, usada por todo lançamento) e por isso vem com a
fundação já provada; o `Relogio` nasce na 3 (onde `agora` é usado de fato); a 4 depende do relógio e do telegram injetados e fecha as globais.

## Como cada fatia é entregue

1. Branch pela mantra, **spec curto** da fatia (arquivos que saem da raiz, interfaces novas, testes que provam que o
   contrato HTTP não mudou) aprovado pelo dono.
2. Implementação com os testes antigos convertidos (não apagados) e o `arquitetura_test.go` estendido às pastas novas.
3. `go vet`, `go test ./...`, e **comparação do contrato**: a coleção Postman e os testes de API rodam iguais antes e depois.
4. Deploy em dev, conferência no app, PR para `develop`; release quando o dono quiser.
5. `arquitetura.md` e o BACKLOG atualizados no mesmo PR (a raiz encolhe a cada fatia).

## Decisões do dono (2026-10-17)

1. **Ordem das fatias:** 1 → 2 → 3 → 4.
2. **Avisos e Telegram:** migram na fatia 4, só com injeção; o despachante fica para o item 34.
3. **Relógio:** interface `Relogio` no service, com adaptador real e fixo.
4. **Um PR por fatia**, com release entre elas.
5. **Pasta dos canais:** `adapter/notificacao/`, com um arquivo por canal (telegram, e-mail, WhatsApp). O despachante vive no `service`.
