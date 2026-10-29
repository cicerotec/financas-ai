# Plan: log de cada requisição

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/roteador.go`, `backend/main.go`,
`backend/local.go`). O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** aprovado (as quatro recomendações) e implementado em 2026-10-29.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| Todo evento HTTP da Function URL passa por `Servidor.handler`; o agendamento sai antes, em `entrada` | `roteador.go:26`, `roteador.go:132` | O log vai no `handler`: o agendamento fica de fora sem código extra |
| `handler` tem o método e o `RequestContext.RequestID`, mas a **ação** só nasce dentro de `processar` (`auth.RotaParaAcao`) | `roteador.go:38-49` | `processar` precisa devolver a ação ao `handler` (ou o `handler` classifica a rota antes) |
| Três saídas não chegam à classificação: token inválido (401), `/me` e rota inexistente (404) | `roteador.go:27`, `39`, `42-48` | Cada uma ganha um nome fixo de ação: `sem-token`, `me`, `desconhecida` |
| O modo local (`local.go`) chama `processar` direto, sem passar pelo `handler` | `local.go` | Não loga em desenvolvimento, ou chama o mesmo ponto; a decidir (ver abaixo) |
| O projeto usa o `log` padrão (`log.Printf`) | `roteador.go:59`, services | Mantém o `log.Printf`, em formato `chave=valor` como a linha "negado" |
| A retenção do log group já é de 14 dias | `template.yaml:89` | Nada a mudar na infra |

## Desenho

- **`handler` mede e loga.** Guarda `t0 := time.Now()`, chama `processar`, e ao voltar escreve uma linha:
  `req rid=<requestId> m=<GET> acao=<tx.ler> status=<200> ms=<12>`.
- **A ação vem de `processar`.** A assinatura passa a devolver também a ação classificada (string vazia nas três saídas acima, que o
  `handler` troca pelos nomes fixos). Os ids (`sid`, `p[3]`) nunca entram na linha porque a ação é um nome de categoria.
- **O status** vem de `resp.StatusCode` da resposta. Se `processar` devolver erro Go (resposta nula), o status logado é `500`.
- **Uma linha por requisição, na saída.** A linha de entrada não acrescenta informação (a saída já tem rota e status) e dobraria o
  volume. Se a requisição travar até o timeout, o `REPORT` da AWS e o `Task timed out` já a denunciam.
- **Função pequena e testável:** `linhaDeLog(rid, m, acao string, status int, d time.Duration) string` em `roteador.go`, para o teste
  não depender do `log` global.

## Decisões a confirmar com o dono

1. **Só a linha de saída**, em vez de entrada e saída (recomendado: metade do volume e a mesma informação).
2. **Ação (`tx.ler`) no lugar do caminho** (recomendado: sem ids e agrupa bem no Logs Insights; o caminho cru `/spaces/<id>/tx` expõe
   o id do espaço).
3. **Sem `sub`** na linha (recomendado: privacidade; dá para ligar depois).
4. **Modo local** (`local.go`) também loga? (recomendado: sim, pelo mesmo ponto, para ver o log ao depurar.)

## Riscos

- **Mudar a assinatura de `processar`** mexe em quem a chama (`handler` e `local.go`); os testes do roteador precisam acompanhar.
- **Vazar dado sem querer:** por isso a linha é montada só com campos fixos e há teste que procura id, corpo e query string no texto.
- **Ordem das fases:** é uma fase só (backend), sem tela, sem demo e sem mudança de chave no DynamoDB.
