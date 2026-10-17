# Spec da fatia 2: configurações, tags, contador e perfil

Segunda entrega do [`plan.md`](plan.md) do item 32 (a primeira foi o [`spec-fatia-1.md`](spec-fatia-1.md), no ar).
**Refatoração: o contrato HTTP não muda.** Este spec lista o que sai da raiz, o que nasce em cada camada, o que fica de
ponte até a fatia 3 e os testes que provam que nada mudou.

**Estado:** rascunho para o dono aprovar. Nenhum código foi alterado.

## Escopo

| Sai de | O que | Vai para |
|---|---|---|
| `main.go` | `getCfg`, `putCfg`, `salvarCfg` (controle otimista por `version`), `putAparencia`, `errConflito`, `errDados` | `repository/config.go`, `service/config.go`, `controller/config.go` |
| `tags.go` | `adicionarTags` (com 3 tentativas), `garantirTagsServidor`, `postTags` | `service/tags.go` (regra) sobre `repository/config.go` (grava `CFG#TAGS`) |
| `seq.go` | `proximoSeq`, `comSeq`, `getSeq` | `repository/contador.go`, `service/contador.go` |
| `perfil.go` | `normalizarApelido`, `putPerfil` | `domain/perfil.go` (regras puras), `repository/pessoas.go` (`AtualizarPerfil`), `service/perfil.go`, `controller/perfil.go` |
| `telegram.go` | `reChatID` (só `putPerfil` o usa) | `domain.ChatIDValido` |
| `efeitos.go` | `GarantirTags` e `ComSeq` chamavam funções da raiz | passam a chamar `service.Tags` e `service.Contador`, injetados em `efeitosRaiz` |

**Fica na raiz de propósito** (migra nas fatias 3 e 4):
- `postAvisoTeste` (em `perfil.go`): fala com o Telegram. Migra na fatia 4 com o `Notificador`.
- `getItem`, `marshal`, `key`, `spacePK`, `txSK` e as globais `db`, `table`, `ssmCli`, `agora`.
- `salvarCfg(ctx, pk, sk, body, esperada)` continua existindo como **ponte de uma linha** para
  `repository.Config.SalvarItem`, porque `fechamento.go` grava os itens `SALDO#<banco>` com ele. Some na fatia 3, quando o
  fechamento ganha o seu repository. O mesmo vale para `errConflito`, que vira apelido de `domain.ErrConflitoConfig` em
  `dominio_compat.go`.

## Contrato que não muda (exemplos, dados fictícios)

Configuração compartilhada (`listas`, `tags`, `combos`, `saldos`, `cartoes`). Item `PK=SPACE#s1`, `SK=CFG#<NOME EM MAIÚSCULAS>`:

```bash
curl -s "$API/spaces/s1/cfg/listas" -H "authorization: Bearer $TOKEN"
```
```json
{ "banco": ["ITAU", "NUBANK"], "status": ["PAGO", "PREVISTO"], "version": 3 }
```
Sem o item: `{"version": 0}`. `PUT` (só o dono) manda o `version` que leu; o servidor grava com condição:

```bash
curl -s -X PUT "$API/spaces/s1/cfg/listas" -H "authorization: Bearer $TOKEN" -H "content-type: application/json" \
  -d '{"banco":["ITAU","NUBANK","C6"],"status":["PAGO","PREVISTO"],"version":3}'
```
`200` devolve o item com `version: 4`. Se outra pessoa gravou antes: `409`

```json
{ "erro": "alterado por outra pessoa", "atual": { "banco": ["ITAU", "NUBANK"], "status": ["PAGO"], "version": 4 } }
```
Corpo que não serializa: `400 {"erro":"dados invalidos"}`. Falha do banco: `500 {"erro":"erro ao salvar"}` (no `GET`:
`500 {"erro":"erro interno"}`). Chaves internas (`PK`, `SK`) e os campos protegidos nunca saem nem entram.

Aparência (por pessoa, `SK=USER#<sub>#CFG#APARENCIA` sob a `PK` do espaço, **sem** controle de versão):

```bash
curl -s -X PUT "$API/spaces/s1/aparencia" -H "authorization: Bearer $TOKEN" -H "content-type: application/json" -d '{"tema":"escuro"}'
```
`200 {"tema":"escuro"}`. `GET` sem item: `{"version": 0}`.

Tags: o membro pode acrescentar, nunca reescrever.

```bash
curl -s -X POST "$API/spaces/s1/tags" -H "authorization: Bearer $TOKEN" -H "content-type: application/json" -d '{"tags":["Lazer","viagem"]}'
```
`200 {"adicionadas":["viagem"]}` (só as que não existiam; `[]` se nenhuma). Sem tag válida: `400 {"erro":"informe ao menos uma tag"}`.
Conflito persistente nas 3 tentativas: `500 {"erro":"erro ao salvar tags"}`. Ao salvar um lançamento, as tags novas entram
sozinhas em `CFG#TAGS` (falha só vai ao log).

Contador de alterações (`SPACE#s1` / `SEQ`, atributo `seq`):

```bash
curl -s "$API/spaces/s1/seq" -H "authorization: Bearer $TOKEN"
```
`200 {"seq": 41}` (`0` se não existe). Cada escrita de lançamento soma 1 e devolve `_seq` na resposta; se o contador falhar,
a resposta sai sem `_seq` e só o log registra.

Perfil (cada pessoa altera o **próprio** vínculo `USER#<sub>` / `SPACE#<sid>`):

```bash
curl -s -X PUT "$API/spaces/s1/perfil" -H "authorization: Bearer $TOKEN" -H "content-type: application/json" \
  -d '{"apelido":"Ana","telegramChatId":"123456789"}'
```
`200 {"apelido":"Ana","telegramChatId":"123456789"}`. Texto vazio remove o campo. Erros preservados: `400` sem nenhum dos dois
campos (`informe o apelido ou o telegramChatId`), apelido que não é texto (`informe o apelido`), com mais de 30 caracteres
(`o nome pode ter no maximo 30 caracteres`) ou com caractere de controle (`o nome tem caracteres invalidos`); chat id que
não é texto (`informe o telegramChatId como texto`) ou fora de `^-?[0-9]{5,20}$` (`o chat id do Telegram tem so numeros
(ex.: 123456789)`); `404 vinculo com o espaco nao encontrado` se o vínculo não existe; `500 erro ao salvar`.

## Desenho

- `domain/config.go` (novo): `ErrConflitoConfig` (alguém gravou antes). `ErrDadosInvalidos` já existe e é reaproveitado.
- `domain/perfil.go` (novo): `NormalizarApelido(string) (string, error)` (sem mudar as mensagens) e `ChatIDValido(string) bool`.
  Os testes de `normalizarApelido` mudam para `domain/perfil_test.go`.
- `repository/config.go` (novo): `Config{db, table}` com
  - `Ler(ctx, sid, nome)` e `Salvar(ctx, sid, nome, body, esperada)` (a chave é `CFG#` + nome em maiúsculas);
  - `LerAparencia(ctx, sid, sub)` e `GravarAparencia(ctx, sid, sub, body)` (a chave `USER#<sub>#CFG#APARENCIA`);
  - `LerItem(ctx, pk, sk)` e `SalvarItem(ctx, pk, sk, body, esperada)`: o `getItem`/`salvarCfg` de hoje, para a ponte e para o
    `Salvar`. `SalvarItem` mantém a condição (`attribute_not_exists(PK) OR attribute_not_exists(version)` quando `esperada` é 0;
    `version = :v` nos demais) e devolve `domain.ErrConflitoConfig`.
  **Nenhuma chave nova**: as mesmas de hoje.
- `repository/contador.go` (novo): `Contador{db, table}` com `Proximo(ctx, sid) (float64, error)` (`ADD seq 1` com
  `ReturnValues UPDATED_NEW`) e `Atual(ctx, sid) (float64, error)`.
- `repository/pessoas.go`: ganha `AtualizarPerfil(ctx, sid, sub string, m domain.MudancaPerfil) error` (mesmo `UpdateExpression`,
  `attribute_exists(PK)`, devolvendo `domain.ErrVinculoNaoEncontrado`).
- `service/config.go` (novo): `Config` com `Ler`, `Gravar` (separa o `version` do corpo, tira os campos protegidos, grava e,
  no conflito, devolve `*ErroConflitoConfig{Atual}` com o item atual já público), `LerAparencia`, `GravarAparencia`. Declara a
  interface `RepositorioConfig`.
- `service/tags.go` (novo): `Tags` com `Adicionar` (as 3 tentativas, `domain.Faltantes`, `domain.NormalizarTags`) e `Garantir`
  (o que `efeitosRaiz.GarantirTags` chamava; falha só vai ao log). Usa `RepositorioConfig`.
- `service/contador.go` (novo): `Contador` com `Atual`, `Proximo` e `ComSeq(ctx, sid, doc)` (se o repository falha, loga e devolve
  o doc sem `_seq`). Declara `RepositorioContador`.
- `service/perfil.go` (novo): `Perfil.Editar(ctx, sid, sub, body)` com as validações de hoje (usa `domain.NormalizarApelido` e
  `domain.ChatIDValido`) e a `domain.MudancaPerfil`. Declara `RepositorioPerfil`.
- `controller/config.go` e `controller/perfil.go` (novos): `GetCfg`, `PutCfg`, `GetAparencia`, `PutAparencia`, `PostTags`, `GetSeq`,
  `PutPerfil`. O `respostaErro` ganha os erros novos (`*ErroConflitoConfig` → `409` com `atual`; `ErrSalvarConfig`, `ErrSalvarTags`
  → `500` com a mensagem de hoje).
- `servidor.go`: `novoServidor` cria `Config`, `Tags`, `Contador` e `Perfil` e injeta `Tags` e `Contador` em `efeitosRaiz`
  (que passa a ter campos; continua ponte para saldos e avisos até as fatias 3 e 4). `processar` chama os controllers nesses `case`.
- `arquitetura_test.go`: nada a proibir de novo (as pastas já existem).

## Testes

| # | Prova | Como |
|---|---|---|
| F2-T1 | `GET /cfg/{nome}`: item existente, ausente (`{"version":0}`), sem `PK`/`SK` na resposta, nome inválido (`404`), erro de leitura (`500 erro interno`) | testes de API + um novo para o erro |
| F2-T2 | `PUT /cfg/{nome}`: primeira gravação (`version` 0 → 1), gravação com a versão certa, conflito (`409` com o `atual` público), corpo com `version` e campos protegidos ignorados, erro do banco (`500 erro ao salvar`) | testes de API existentes + conflito e erro novos |
| F2-T3 | Papéis: membro não escreve cfg (`403`), lê cfg e escreve a própria aparência | testes existentes (`TestMemberNaoMexeEmConfiguracoesCompartilhadas`, `TestAparenciaEPorUsuario`) |
| F2-T4 | Tags: acrescentar só as novas, nenhuma nova (`[]`), sem tag (`400`), membro não reescreve a lista, lançamento com tag nova a inclui, 3 conflitos seguidos → `500 erro ao salvar tags` | existentes + conflito persistente (gancho no fake) |
| F2-T5 | Contador: cada escrita soma 1 e devolve `_seq`, não sobe quando nada foi gravado, importação em lote, número grande, `GET /seq`, falha do contador não derruba a escrita (sai sem `_seq`) | `seq_test.go` existentes + falha injetada |
| F2-T6 | Perfil: apelido e chat id, vazio remove, cada pessoa só o próprio, todas as validações e mensagens, vínculo inexistente (`404`), erro (`500`), `GET /me` devolve o apelido | `perfil_test.go` existentes; as de `normalizarApelido` vão para `domain/` |
| F2-T7 | A raiz perdeu `tags.go`, `seq.go` e `putCfg`/`getCfg`/`putAparencia`/`putPerfil`; `main.go` e `perfil.go` não têm `UpdateItemInput` nem `PutItemInput` (só a ponte `salvarCfg` chama o repository) | teste de inspeção do fonte, como o F1-T6 |
| F2-T8 | Mutação: tirar a condição de versão do `SalvarItem` e remover a normalização de tags derrubam os testes certos | verificada à mão |

Todos os testes de API que existem hoje passam **sem alterar as asserções**.

## Pronto quando

- `go vet ./...` e `go test ./...` passam; `sam validate --lint` passa (o `template.yaml` não muda).
- A coleção Postman roda igual contra dev: `cfg/listas` (leitura e escrita), `aparencia`, `tags`, `seq` e `perfil`.
- No app de dev: salvar uma lista (Listas), mudar o apelido (Perfil), lançar com tag nova e ver o contador funcionando (a tela
  recarrega quando outra pessoa grava).
- `tags.go` e `seq.go` não existem mais; `perfil.go` só tem `postAvisoTeste`; `main.go` não tem handlers de configuração.
- `arquitetura.md` e o item 32 do BACKLOG atualizados no mesmo PR.

## Decisões para o dono (recomendação em negrito)

1. **`postAvisoTeste` fica na raiz até a fatia 4** (depende do Telegram) em vez de migrar junto com o perfil.
2. **A ponte `salvarCfg` fica até a fatia 3** (o fechamento grava `SALDO#*` com ela) em vez de antecipar o repository de saldos.
3. **`Tags` e `Contador` ficam como serviços próprios** (`service/tags.go`, `service/contador.go`) e não dentro de `Config`: o
   contador é regra de lançamento, não de configuração, e a fatia 3 e o item 34 o reutilizam.
