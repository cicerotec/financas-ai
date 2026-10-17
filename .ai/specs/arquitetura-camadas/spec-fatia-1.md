# Spec da fatia 1: espaços e leitura de lançamentos

Primeira entrega do [`plan.md`](plan.md) do item 32. **Refatoração: o contrato HTTP não muda.** Este spec lista o que sai da
raiz, o que nasce em cada camada e os testes que provam que nada mudou.

**Estado:** rascunho para o dono aprovar. Nenhum código foi alterado.

## Escopo

| Sai de | O que | Vai para |
|---|---|---|
| `store.go` | `papel` (papel da pessoa no espaço) | `repository/espacos.go` (`Papel`) e `service/espacos.go` |
| `main.go` | `getMe` (espaços da pessoa, com nome do `META`) | `repository/espacos.go` (`EspacosDe`), `service/espacos.go` (`Meus`) e `controller/espacos.go` (`GetMe`) |
| `store.go` | `listarTx`, `filtroTx`, `codCursor`, `decCursor` | `repository/lancamento.go` (`Listar`), `domain.FiltroTx` e o cursor no `repository` |
| `main.go` | `getTxs` | `service.Lancamentos.Listar` e `controller.Lancamentos.GetTxs` |
| `main.go` | montagem do service **a cada requisição** (`controller.NewLancamentos(service.NewLancamentos(...))` dentro de `processar`) | `servidor.go`: um `Servidor` montado **uma vez** em `main` |

**Fica na raiz de propósito** (migra nas fatias 2 a 4): `getItem`, `marshal`, `key`, `spacePK`, `txSK` (usados por
configuração, fechamento, tags, seq, perfil e avisos) e as globais `db`, `table`, `ssmCli`, `agora`. O `Servidor` recebe o
mesmo `db` e `table` que as globais apontam, de modo que as duas formas convivem até a fatia 4.

## Contrato que não muda (exemplos, dados fictícios)

```bash
curl -s "$API/me" -H "authorization: Bearer $TOKEN"
```
```json
{ "sub": "a1b2c3d4-...", "email": "ana@exemplo.com",
  "espacos": [ { "id": "s1", "nome": "Familia", "role": "owner", "apelido": "Ana", "telegramChatId": "1234..." } ] }
```
Sem vínculo, `espacos` é `[]`. Espaço sem `META` usa o próprio id como `nome`. Falha de leitura: `500 {"erro":"erro interno"}`.

```bash
curl -s "$API/spaces/s1/tx?de=2026-10-01&ate=2026-10-31T99&banco=ITAU&status=PAGO&q=mercado&ordem=asc&limite=50&cursor=$C" \
  -H "authorization: Bearer $TOKEN"
```
```json
{ "itens": [ { "id": "k3x9...", "dataEvento": "2026-10-10T12:00:00.000Z", "descricao": "Mercado", "valor": 187.5,
               "tipo": "saida", "banco": "ITAU", "status": "PAGO", "tags": ["Lazer"] } ],
  "proximo": "VFgjMjAyNi0xMC0xMC..." }
```
Regras que a migração preserva **exatamente**: `limite` padrão 200, aceito de 1 a 1000, senão 200; páginas de 500 no
DynamoDB e filtro em memória por `banco`, `status` e `q` (sem acento, minúsculas, em descrição e nota); itens com
`excluidoEm` nunca saem; `PK`/`SK` e campos internos não saem (`domain.Publico`); `proximo` é a `SK` em base64 (só
aceita `TX#...`). `itens` nunca é `null`. **Cursor inválido continua devolvendo `500 {"erro":"erro ao listar"}`**, como
hoje (é um achado: o certo seria `400`; fica registrado no BACKLOG para outra entrega, para esta não mudar comportamento).

## Desenho

- `domain/lancamento.go`: `type FiltroTx struct { De, Ate, Banco, Q, Status string; Asc bool; Limite int; Cursor string }`.
- `repository/lancamento.go`: `func (r *Lancamentos) Listar(ctx, sid, f domain.FiltroTx) (itens []domain.Doc, proximo string, err error)`
  com o corpo atual de `listarTx`, e `ErrCursorInvalido`. O cursor (`codCursor`/`decCursor`) mora ao lado.
- `repository/espacos.go` (novo): `Espacos{db, table}` com `Papel(ctx, sid, sub) (string, error)` e
  `EspacosDe(ctx, sub) ([]domain.Espaco, error)` (lê `USER#<sub>` / `SPACE#*` e o `META` de cada espaço). **Nenhuma chave
  nova**: as mesmas leituras de hoje.
- `domain/espaco.go` (novo): `type Espaco struct { ID, Nome, Role, Apelido, TelegramChatID string }`. A resposta HTTP continua
  com as chaves `id`, `nome`, `role`, `apelido`, `telegramChatId` (o controller monta o JSON).
- `service/espacos.go` (novo): `Espacos` com `Papel` e `Meus`, declarando a interface `RepositorioEspacos`.
  `service.Lancamentos` ganha `Listar` (delegando ao `Repositorio`, que ganha o método `Listar`).
- `controller/espacos.go` (novo): `GetMe(ctx, sub, email)`. `controller/lancamento.go`: `GetTxs(ctx, sid, qs)` faz a leitura do
  `limite` e das demais queries.
- `servidor.go` (raiz, novo): `type Servidor struct { espacos *controller.Espacos; lanc *controller.Lancamentos; papel func(...) }`,
  `func novoServidor(db dynamoAPI, table string) *Servidor` e `func (s *Servidor) processar(...)`, que é o `processar` de hoje
  usando `s.lanc` e `s.espacos`. `main()` cria o `Servidor` uma vez depois do `initDB`; `entrada` e `local.go` chamam
  `s.handler`. As rotas de configuração, fechamento, tags, seq, perfil e avisos continuam chamando as funções da raiz.
- `arquitetura_test.go`: nada novo a proibir; os pacotes já existem. O teste ganha uma verificação de que `main.go` e
  `servidor.go` **não contêm** `dynamodb.QueryInput` nem `GetItemInput` de espaços (a leitura de espaços e de lançamentos
  não está mais na raiz).

## Testes

| # | Prova | Como |
|---|---|---|
| F1-T1 | `GET /me` igual ao de hoje: um espaço com `META`, um sem `META`, nenhum vínculo (`[]`), erro de leitura (`500`) | converter os testes atuais de `/me` para o `Servidor`; manter as asserções |
| F1-T2 | `GET /tx`: filtros `de`, `ate`, `banco`, `status`, `q`, `ordem=asc`, `limite` (padrão, 1, 1000, 1001, `abc`), paginação por cursor completa sem repetir nem perder item | testes de API existentes + um de paginação com 3 páginas |
| F1-T3 | Excluídos logicamente não aparecem; `PK`, `SK` e `excluidoEm` não vazam | teste existente + asserção |
| F1-T4 | Cursor inválido: `500 erro ao listar` (comportamento preservado) | teste novo |
| F1-T5 | O papel vem do repository: dono, membro e sem vínculo (`403 sem acesso a esse espaco`) | testes de autorização existentes |
| F1-T6 | O `Servidor` é montado uma vez: duas requisições usam o mesmo `service` (não há `NewLancamentos` dentro de `processar`) | teste de `servidor.go` por inspeção do fonte, como o de arquitetura |
| F1-T7 | Mutação: trocar `limite` padrão de 200 e remover o filtro de `excluidoEm` derrubam os testes certos | verificada à mão, como nas fases anteriores |

Todos os testes de API que existem hoje (`handler_test.go`, `historico*_test.go`, `fechamento*_test.go`, etc.) passam **sem
alterar as asserções**. O que muda neles é só o `chamar`, que passa a usar o `Servidor` do `cenario`.

## Pronto quando

- `go vet ./...` e `go test ./...` passam; `sam validate --lint` passa (o `template.yaml` não muda).
- A coleção Postman (`docs/postman/`) roda igual contra dev: `GET /me` e `GET /spaces/{id}/tx` com os mesmos corpos.
- `store.go` não tem mais `papel`, `listarTx`, `filtroTx` nem cursor; `main.go` não tem `getMe` nem `getTxs`.
- `arquitetura.md` (estrutura de pastas e "o que ainda não migrou") e o item 32 do BACKLOG atualizados no mesmo PR.

## Ajuste no plan

O `Relogio` não é usado nesta fatia (o service de lançamentos não lê relógio; quem lê é a trava, na raiz). Ele **nasce na
fatia 3** (fechamento), onde `agora` é usado de fato. O `plan.md` foi corrigido.
