# API: índice das rotas

O que a API tem **hoje**, uma linha por rota. Não há API Gateway nem documento central, e o contrato de cada rota (corpo, resposta, todos os status) vive na seção **"API"** do `spec.md` da feature que a criou (regra do [`guardrails.md`](guardrails.md)). Este arquivo é o **índice**: diz que rotas existem, a ação de cada uma em `backend/auth/authz.go`, quem pode chamar e onde está o contrato.

**Não é escrito de memória:** o teste `backend/api_doc_test.go` lê este arquivo e o compara com o código. Rota que o código reconhece e não está aqui, linha que o código não reconhece, ação diferente ou decisão de papel diferente de `auth.Permitido` **falham o `go test ./...`** (e, por isso, o CI e o deploy).

## Como ler

| Coluna | O que é |
|---|---|
| Método e Caminho | `{nome}` é um valor qualquer (o nome só ajuda a ler); `a\|b\|c` são alternativas, uma rota para cada. `{sid}` é o espaço |
| Ação | A `Acao` de `auth/authz.go` (o mesmo nome que vai ao log, `acao=...`). `me` é a única que não é uma `Acao`: `GET /me` é tratada no roteador |
| Owner e Member | A decisão de `auth.Permitido` para o papel: **sim**, **não** ou **só o que criou** (`criadoPor == sub`). Rota fora de `/spaces/{sid}` não tem papel (`—`): vale o `sub` do token |
| Token | **ID token** (validado por `auth.VerificarToken`) ou **nenhum** (rota pública; só existe a de `auth.RotaPublica`) |
| Spec | Onde está o contrato, relativo a `.ai/`; `—` se ainda não há |

Rota de espaço (`/spaces/{sid}/...`) também exige vínculo da pessoa com o espaço: sem ele, `403`.

## Como este índice se mantém

- **Rota nova, mudada ou removida:** o PR altera `auth/authz.go` (ou `roteador.go`), o teste de `authz_test.go` **e** uma linha aqui. Rodar `go test ./...` na pasta `backend/`: se faltar linha, a mensagem do teste traz a linha pronta para colar (ajuste os `{nomes}` e a coluna Spec).
- **Spec da feature:** a rota nova ganha a seção "API" no `spec.md` (guardrails); a coluna Spec aponta para ele.
- **Papel mudou:** atualizar a linha **e** a tabela de ações de [`security.md`](security.md) na mesma entrega (já era a regra).
- O teste só enxerga o que o roteamento compara com literais (`rest[0] == "tx"`, `switch p[2]`, as chaves de `cfgNomes`). Quem introduzir outro jeito de classificar rota (prefixo, regex, tabela) estende o colhedor de literais do teste no mesmo PR.

## Rotas

| Método | Caminho | Ação | Owner | Member | Token | Spec |
|---|---|---|---|---|---|---|
| GET | `/me` | `me` | — | — | ID token | specs/app-multiusuario/intent.md |
| POST | `/push/sair` | `push.sair` | — | — | ID token | specs/app-android-push/spec.md |
| GET | `/spaces/{sid}/aparencia` | `aparencia.ler` | sim | sim | ID token | arquitetura.md |
| PUT | `/spaces/{sid}/aparencia` | `aparencia.escrever` | sim | sim | ID token | arquitetura.md |
| POST | `/spaces/{sid}/avisos/teste` | `avisos.testar` | sim | sim | ID token | avisos.md |
| GET | `/spaces/{sid}/cartoes` | `cartao.ler` | sim | sim | ID token | specs/fatura-servidor/spec.md |
| POST | `/spaces/{sid}/cartoes` | `cartao.escrever` | sim | não | ID token | specs/fatura-servidor/spec.md |
| PUT | `/spaces/{sid}/cartoes/{banco}` | `cartao.escrever` | sim | não | ID token | specs/fatura-servidor/spec.md |
| DELETE | `/spaces/{sid}/cartoes/{banco}` | `cartao.escrever` | sim | não | ID token | specs/fatura-servidor/spec.md |
| GET | `/spaces/{sid}/cartoes/{banco}/faturas` | `cartao.ler` | sim | sim | ID token | specs/fatura-servidor/spec.md |
| GET | `/spaces/{sid}/cartoes/{banco}/faturas/{ym}` | `cartao.ler` | sim | sim | ID token | specs/fatura-servidor/spec.md |
| PUT | `/spaces/{sid}/cartoes/{banco}/faturas/{ym}` | `cartao.escrever` | sim | não | ID token | specs/fatura-servidor/spec.md |
| POST | `/spaces/{sid}/cartoes/{banco}/faturas/{ym}/conferir` | `cartao.escrever` | sim | não | ID token | specs/fatura-servidor/spec.md |
| DELETE | `/spaces/{sid}/cartoes/{banco}/faturas/{ym}/conferir` | `cartao.escrever` | sim | não | ID token | specs/fatura-servidor/spec.md |
| POST | `/spaces/{sid}/cartoes/{banco}/faturas/{ym}/diferenca` | `cartao.escrever` | sim | não | ID token | specs/fatura-servidor/spec.md |
| GET | `/spaces/{sid}/cfg/listas\|tags\|combos\|saldos\|cartoes` | `cfg.ler` | sim | sim | ID token | arquitetura.md |
| PUT | `/spaces/{sid}/cfg/listas\|tags\|combos\|saldos` | `cfg.escrever` | sim | não | ID token | arquitetura.md |
| PUT | `/spaces/{sid}/combos/rotina` | `combos.rotina` | sim | não | ID token | specs/combos-sugeridos/spec.md |
| GET | `/spaces/{sid}/consultas` | `consulta.ler` | sim | sim | ID token | specs/tendencias-consultas/spec.md |
| POST | `/spaces/{sid}/consultas` | `consulta.criar` | sim | sim | ID token | specs/tendencias-consultas/spec.md |
| PUT | `/spaces/{sid}/consultas/{id}` | `consulta.editar` | sim | só o que criou | ID token | specs/tendencias-consultas/spec.md |
| DELETE | `/spaces/{sid}/consultas/{id}` | `consulta.editar` | sim | só o que criou | ID token | specs/tendencias-consultas/spec.md |
| POST | `/spaces/{sid}/consultas/{id}/restaurar` | `consulta.editar` | sim | só o que criou | ID token | specs/tendencias-consultas/spec.md |
| POST | `/spaces/{sid}/fechamentos/conferir` | `fechamento.conferir` | sim | não | ID token | specs/saldo-servidor/spec.md |
| POST | `/spaces/{sid}/fechamentos/reabrir` | `fechamento.reabrir` | sim | não | ID token | specs/saldo-servidor/spec.md |
| GET | `/spaces/{sid}/futuros` | `tx.ler` | sim | sim | ID token | specs/futuros-previsto/spec.md |
| GET | `/spaces/{sid}/historico` | `historico.ler` | sim | só o que criou | ID token | specs/historico/spec.md |
| PUT | `/spaces/{sid}/meios/{meio}` | `meio.escrever` | sim | não | ID token | specs/meios-de-pagamento/spec.md |
| DELETE | `/spaces/{sid}/meios/{meio}` | `meio.escrever` | sim | não | ID token | specs/meios-de-pagamento/spec.md |
| PUT | `/spaces/{sid}/perfil` | `perfil.editar` | sim | sim | ID token | specs/app-multiusuario/intent.md |
| PUT | `/spaces/{sid}/push` | `push.registrar` | sim | sim | ID token | specs/app-android-push/spec.md |
| POST | `/spaces/{sid}/push/remover` | `push.remover` | sim | sim | ID token | specs/app-android-push/spec.md |
| GET | `/spaces/{sid}/recorrencias` | `tx.ler` | sim | sim | ID token | specs/recorrencia/spec.md |
| POST | `/spaces/{sid}/recorrencias` | `tx.editar` | sim | só o que criou | ID token | specs/recorrencia/spec.md |
| DELETE | `/spaces/{sid}/recorrencias/{id}` | `tx.editar` | sim | só o que criou | ID token | specs/recorrencia/spec.md |
| GET | `/spaces/{sid}/saldos` | `fechamento.ler` | sim | sim | ID token | specs/saldo-servidor/spec.md |
| GET | `/spaces/{sid}/seq` | `seq.ler` | sim | sim | ID token | arquitetura.md |
| GET | `/spaces/{sid}/status` | `tx.ler` | sim | sim | ID token | specs/futuros-previsto/spec.md |
| POST | `/spaces/{sid}/tags` | `tags.adicionar` | sim | sim | ID token | arquitetura.md |
| POST | `/spaces/{sid}/transferencias` | `tx.criar` | sim | sim | ID token | arquitetura.md |
| GET | `/spaces/{sid}/tx` | `tx.ler` | sim | sim | ID token | arquitetura.md |
| POST | `/spaces/{sid}/tx` | `tx.criar` | sim | sim | ID token | arquitetura.md |
| POST | `/spaces/{sid}/tx/batch` | `tx.importar` | sim | não | ID token | arquitetura.md |
| PUT | `/spaces/{sid}/tx/{id}` | `tx.editar` | sim | só o que criou | ID token | arquitetura.md |
| DELETE | `/spaces/{sid}/tx/{id}` | `tx.excluir` | sim | não | ID token | arquitetura.md |
| POST | `/spaces/{sid}/tx/{id}/confirmar` | `tx.confirmar` | sim | só o que criou | ID token | specs/futuros-previsto/spec.md |
| GET | `/spaces/{sid}/tx/{id}/versoes` | `historico.ler` | sim | só o que criou | ID token | specs/historico/spec.md |
| POST | `/spaces/{sid}/tx/{id}/versoes/{versao}/restaurar` | `historico.restaurar` | sim | só o que criou | ID token | specs/historico/spec.md |

## Fora da API HTTP

A mesma Lambda recebe dois eventos do EventBridge (`{"acao":"avisos"}`, todo dia às 08:00 de São Paulo, e `{"acao":"combos"}`, aos domingos), tratados em `entrada` (`backend/roteador.go`). Não são rotas, não passam por token nem por papel, e não estão no teste.
