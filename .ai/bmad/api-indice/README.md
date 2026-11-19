# Índice de rotas (`.ai/api.md`) e o teste que o mantém

Desenho do metodo-ai, a pedido do dono (2026-11-19), para o agente deste projeto implementar. Não há API Gateway nem documento central de rotas; a regra "Seção API no `spec.md`" do guardrails dá o contrato **de cada feature**, e falta **uma visão única do que existe hoje**. Este desenho a cria sem deixá-la desatualizar: **o `go test ./...` falha se o índice e o código divergirem.**

## Onde estão o índice e o teste

O índice (`api.md`, com as **55 rotas de hoje**, em 48 linhas, porque `cfg/listas|tags|...` agrupa alternativas) e o teste (`api_doc_test.go`, 357 linhas) **foram instalados em 2026-11-19** em [`../../api.md`](../../api.md) e em `backend/api_doc_test.go`, e **já não são copiados nesta pasta**, para não ficarem desatualizados. Passaram juntos contra o código de 2026-11-19 (`go test -run TestIndiceDeRotas` verde, 0,05 s). Esta pasta guarda só o desenho.

## Como o teste funciona

Não tem lista de rotas própria. Usa **uma única porta para o código**: `rotular(m, p)` (`backend/roteador.go`), a mesma função que dá o rótulo do log.

1. **Ida (índice para código):** cada linha de `api.md` é expandida (`{x}` vira um valor qualquer; `a|b|c` vira uma rota para cada) e passada a `rotular`. Falha se: o código não reconhece a rota; a **ação** é outra; as colunas **Owner** e **Member** diferem de `auth.Permitido`; a coluna **Token** diz `ID token` mas a rota é pública (ou o contrário); a **spec** apontada não existe em `.ai/`; a linha está duplicada.
2. **Volta (código para índice):** o teste lê o próprio código (`auth/authz.go` e `roteador.go`, por `go/ast`) e colhe, para cada posição do caminho, os literais com que o código compara (`rest[0] == "tx"`, `switch p[2] { case "futuros": }`, as chaves de `cfgNomes`), mais um valor qualquer (`ID`). Varre todas as combinações (cerca de meio milhão, 4 métodos, em 50 ms) e junta o que `rotular` reconhece. **Toda rota reconhecida tem de estar no índice.** Um segmento que é só parâmetro (por exemplo o `batch` de `DELETE /tx/batch`, que é o id `batch`) é trocado por `ID` antes da comparação.
3. **A mensagem ajuda:** quando falta linha, o teste imprime a linha pronta para colar, com ação, owner, member e token já preenchidos. Sobra ajustar o nome dos `{parâmetros}` e a coluna Spec.

Conferido por "quebrar de propósito" (feito no desenho):

| Quebra | Resultado |
|---|---|
| Apagar uma linha do índice | Falha com a linha pronta |
| Trocar `não` por `sim` na coluna Member | Falha: "tem owner=sim member=não no código" |
| Acrescentar uma rota nova em `RotaParaAcao` | Falha com a linha da rota nova |
| Trocar a ação de uma linha | Falha: "tem a ação seq.ler no código" |
| Apontar uma spec que não existe | Falha por linha |
| Linha de rota que o código não tem | Falha: "não existe no código" |

## Limite conhecido

O teste só enxerga classificação por **comparação de `rest[i]` ou `p[i]` com um literal** e por `cfgNomes`. Se alguém classificar rota de outro jeito (prefixo, regex, tabela), a **volta** não a vê (a **ida** continua valendo). Quem introduzir um jeito novo estende `apiColher` no mesmo PR. O teste tem uma trava: se a varredura não achar nenhuma rota, falha, para o colhedor não parar de funcionar em silêncio.

Também confia em `rotular`: se `processar` passar a tratar uma rota que `rotular` não classifica, o índice não a vê. Endurecimento opcional, fora desta entrega: `processar` e `rotular` usarem a mesma função de classificação.

## O que o agente faz

1. **(Feito em 2026-11-19.)** Copiou `api.md` para `.ai/api.md` e `api_doc_test.go.txt` para `backend/api_doc_test.go` (as cópias desta pasta foram apagadas depois). Rodou `cd backend && go vet ./... && go test ./...`. Os nomes do teste começam com `api` para não colidir com os de `package main` (`chave` já existe em `handler_test.go`).
2. Acrescenta uma linha em [`.ai/README.md`](../../README.md) (índice) e cita `api.md` na tabela "Tarefa" do `AGENTS.md`, na linha "Backend, rota, permissão".
3. **`guardrails.md` já foi ajustado (OK do dono em 2026-11-19):** a regra "Seção API no `spec.md`" ganhou o parágrafo "Índice único". Falta o `security.md`: a regra "Rota nova exige ação nova" ganha "e uma linha em `api.md`". **O parágrafo do guardrails aponta para `api.md` e para o teste: copie os dois no mesmo PR (passo 1), para o link não ficar solto.**
4. **Junto com a Fase 3 do MFA** (AD-1 e AD-2 de [`../arquitetura-mfa-recuperacao.md`](../arquitetura-mfa-recuperacao.md)):
   - `apiEhPublica` passa a ser `_, ok := auth.RotaPublica(m, p); return ok` (hoje devolve sempre `false`; o comentário no código diz isso).
   - `rotular` ganha, **antes** da checagem de `/me`, `if a, ok := auth.RotaPublica(m, p); ok { return rotulo{string(a), p[0]} }`. A seção "API" do `spec.md` do MFA (R34) já pede isso para o log; o teste depende da mesma coisa.
   - As três linhas novas entram em `api.md` (o teste imprime a forma exata; as colunas esperadas são):

     | Método | Caminho | Ação | Owner | Member | Token |
     |---|---|---|---|---|---|
     | POST | `/me/mfa/codigos` | a de gerar | — | — | ID token |
     | POST | `/me/mfa/codigos/confirmar` | `mfa.confirmar` | — | — | ID token |
     | POST | `/mfa/recuperar` | `mfa.recuperar` | — | — | nenhum |

     Confirmado por simulação: com essas três rotas no código, o teste as acha, e exige `nenhum` só na pública.
5. Daí em diante, **toda rota nova ou removida** muda `api.md` no mesmo PR. O teste é o aviso; ninguém precisa lembrar.

## O que não faz

- Não documenta corpo nem resposta. Isso é a seção "API" do `spec.md` de cada feature.
- Não cobre os eventos do EventBridge (`avisos`, `combos`): não são rotas. Estão numa nota no fim do `api.md`.
- Não gera OpenAPI nem coleção do Postman. Se o dono quiser um contrato máquina-legível depois, o índice já dá a lista de rotas, ações e papéis para gerá-lo.
