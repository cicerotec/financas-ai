# Spec da fatia 4: avisos e Telegram

Quarta e última entrega do [`plan.md`](plan.md) do item 32 (fatias 1 a 3 no ar). **Refatoração: o contrato HTTP, a mensagem
enviada e o agendamento não mudam.** Ao fim dela a raiz só liga as peças: somem as globais `db`, `table`, `ssmCli`,
`enviarMensagem` e `ambienteApp`, e os arquivos `store.go`, `avisos.go`, `telegram.go`, `perfil.go`, `efeitos.go` e
`dominio_compat.go`.

**Estado:** rascunho para o dono aprovar. Nenhum código foi alterado.

## Escopo

| Sai de | O que | Vai para |
|---|---|---|
| `avisos.go` | a decisão pura (`decidirAviso`, `lerAviso`, `diaBR`, `ehStatusFuturo`, `statusFuturoPadrao`, `maxAtrasoAviso`) e os textos (`moeda`, `plural`, `dataBR`, `linhaAviso.texto`, `montarResumo`) | `domain/aviso.go` (funções puras, com a marca de ambiente recebida por parâmetro) e `domain.FusoBR` |
| `avisos.go` | o item auxiliar `AVISOS#ATIVOS` (`sincronizarAviso`, `removerAviso`, a leitura paginada dos ativos) | `repository/avisos.go` |
| `avisos.go` | a rotina diária `rodarAvisos` | `service/avisos.go` (`Avisos.Rodar`), que passa a ler o dia pelo `Relogio` |
| `avisos.go` | `marcaAmbiente` e `ambienteApp` (global) | campo `ambiente` do `service.Avisos`, vindo do `AMBIENTE` lido uma vez no `main` |
| `perfil.go` | `postAvisoTeste` | `service.Avisos.Testar` e `controller/avisos.go` (`PostAvisoTeste`) |
| `telegram.go` | envio, token no SSM, `errTelegramNaoConfigurado`, `reChatID` já saiu na fatia 2 | `adapter/notificacao/telegram.go` (implementa `service.Notificador`); o erro vira `domain.ErrCanalNaoConfigurado` |
| `store.go`, `main.go` | `initDB`, `db`, `table`, `ssmCli`, `getItem`, `key`, `marshal`, `spacePK`, `txSK`, `dynamoAPI` | o `main` cria os clients uma vez e os entrega ao `novoServidor`; o resto some (cada repository já tem o seu `Key`/`SpacePK`/`TxSK`) |
| `efeitos.go` | a ponte entre o service de lançamentos e os efeitos | `service/efeitos.go`: `EfeitosLancamento`, que junta `Fechamento`, `Tags`, `Contador` e `Avisos` e implementa a interface `Efeitos` |
| `dominio_compat.go` | apelidos (`doc`, `user`, `fusoApp`, `protegidos`...) | somem; os testes passam a usar `domain.*` e `auth.User` direto |
| `roteador.go` | o evento `{"acao":"avisos"}` chamava `rodarAvisos(ctx, time.Now())` | chama `controller.Avisos.Rodar`, que usa o `Relogio` |

Nada de despachante de canais aqui: o item 34 constrói o despachante no `service` em cima do `Notificador` desta fatia.

## Contrato que não muda (exemplos, dados fictícios)

Teste do aviso (cada pessoa, para o **seu** chat id):

```bash
curl -s -X POST "$API/spaces/s1/avisos/teste" -H "authorization: Bearer $TOKEN" -H "content-type: application/json" -d '{}'
```
`200 {"ok":true}` e o Telegram recebe `🔔 [DEV] Teste do financas.ai: se você recebeu esta mensagem, os avisos vão chegar aqui.`
(o `[DEV] ` só em dev). Erros preservados: `400 cadastre o seu chat id do Telegram primeiro`;
`503 o envio pelo Telegram ainda nao foi configurado no servidor`;
`502 o Telegram recusou a mensagem: confira o chat id e se voce ja iniciou a conversa com o bot`; `500 erro interno` (leitura).

Agendamento diário (EventBridge): entrada `{"acao":"avisos"}`, resposta `{"enviados": 2}`. Chamada HTTP com esse corpo continua
caindo no handler e voltando `401` sem token. A rotina, para cada item de `AVISOS#ATIVOS`: descarta o vencido há mais de 60
dias; confere o lançamento de verdade (excluído ou inexistente remove o item; oculto não avisa); só avisa se o status é de
evento futuro (`statusFuturo` em `CFG#LISTAS`, padrão PREVISTO, TRANSFERINDO e CREDITANDO); manda para o criador e para o dono,
um resumo por chat (sem repetir chat). Telegram não configurado: log e sem erro; falha de envio de um chat: log e segue.

O resumo (um por chat):

```
🔔 Avisos de 10/10/2026
• Atrasado há 2 dias (08/10/2026): Aluguel — R$ 1.500,00 (ITAU)
• Vence hoje (10/10/2026): Internet — R$ 99,90 (NUBANK)
• Vence em 3 dias (13/10/2026): Condomínio — R$ 480,00 (ITAU)
• A receber em 1 dia (11/10/2026): Freela — R$ 2.000,00 (NUBANK)
```
Ordem: atrasados, hoje, depois os que ainda vão vencer; dentro de cada grupo, pelo vencimento.

Itens `AVISOS#ATIVOS` (a mesma forma de hoje): `PK=AVISOS#ATIVOS`, `SK=<sid>#<id>`, com `sid`, `id`, `dataEvento`, `venc`, `dias`,
`insistir`. **Nenhuma chave nova.** O token continua no SSM (`TELEGRAM_TOKEN_PARAM`), o template não muda.

## Desenho

- `domain/aviso.go`: ganha `FusoBR`, `LerAviso`, `DiaBR`, `DecidirAviso`, `EhStatusFuturo`, `Linha` (com `Texto()`), `MontarResumo(marca,
  hoje, linhas)` e os textos. Todas puras, testadas em tabela. `domain.ErrCanalNaoConfigurado` (erro-sentinela do envio).
- `repository/avisos.go` (novo): `Avisos{db, table}` com `Sincronizar(ctx, sid, item)` (PutItem), `Remover(ctx, sid, id)` (DeleteItem) e
  `Ativos(ctx)` (Query paginada de `AVISOS#ATIVOS`). Para a rotina também precisa de:
  - `Pessoas.ChatID(ctx, sid, sub)` (o `telegramChatId` do vínculo; vazio se não houver);
  - `Espacos.Dono(ctx, sid)` (o `ownerSub` do `META`).
  O lançamento vem de `Lancamentos.Buscar` (já existe) e a lista de status de `Config.Ler(sid, "listas")`.
- `service/avisos.go` (novo): `Avisos{repo, lancamentos, config, pessoas, espacos, notificador, rel, ambiente}` com `Sincronizar`,
  `Remover` (os efeitos de lançamento; falha só vai ao log, como hoje), `Rodar(ctx) (int, error)` e `Testar(ctx, sid, sub)`.
  Declara as interfaces `RepositorioAvisos` e `Notificador { Enviar(ctx, destino, texto string) error }`.
- `service/efeitos.go` (novo): `EfeitosLancamento` implementa `Efeitos` chamando `Fechamento`, `Tags`, `Contador` e `Avisos`.
- `adapter/notificacao/telegram.go` (novo): `Telegram{ssm, param, base, http}` com o token em cache **dentro do struct** (com mutex), sem
  global. `NovoTelegram(ssm SSM, param string)`; `base` e `http` ficam como campos para o teste apontar para um servidor falso. O erro
  de rede continua sem repassar o texto original (a URL leva o token).
- `controller/avisos.go` (novo): `PostAvisoTeste` e `Rodar` (a resposta `{"enviados": n}`). O `respostaErro` ganha o `503` e o `502`.
- `main.go`: lê `TABLE_NAME`, `AMBIENTE` e `TELEGRAM_TOKEN_PARAM` uma vez, carrega a config da AWS, cria os clients do DynamoDB e do SSM
  e chama `novoServidor(db, table, rel, telegram, ambiente)`. `rodarLocal` recebe a tabela por parâmetro.
- `arquitetura_test.go`: ganha `adapter/notificacao` (não importa `service`, `controller` nem `repository`); `domain` segue sem AWS.

## Testes

| # | Prova | Como |
|---|---|---|
| F4-T1 | Decisão e texto idênticos: antes, hoje, atrasado, insistir, limite de 60 dias, ordem do resumo, moeda, plural, data com ano | os testes de `avisos_test.go` que hoje chamam as funções da raiz passam a chamar `domain.*`, **sem mudar asserções** |
| F4-T2 | Sincronizar e remover o item auxiliar ao criar, editar e excluir lançamento; falha do banco só vai ao log | testes de API existentes + falha injetada |
| F4-T3 | A rotina: envia para o criador e o dono sem repetir chat; ignora oculto, excluído e status resolvido; limpa o vencido há muito; Telegram não configurado não é erro; falha em um chat não impede os outros | `avisos_test.go` com um `Notificador` falso injetado no `Servidor` (no lugar de trocar `enviarMensagem`) |
| F4-T4 | A marca `[DEV]` aparece no teste e no resumo só em dev | dois `Servidor` de teste, um com `ambiente="dev"` |
| F4-T5 | `POST /avisos/teste`: sem chat id (`400`), não configurado (`503`), recusado (`502`), ok (`200`), estranho (`403`) | testes existentes |
| F4-T6 | O evento `{"acao":"avisos"}` vs HTTP sem token (`401`); o "hoje" vem do `Relogio` | `TestEntradaDistingueAgendamentoDeHTTP` + um com o relógio em 2026-10-10 |
| F4-T7 | O adaptador do Telegram: monta o corpo e a URL certos, lê o token do SSM uma vez e o guarda, trata `não configurado` (sem parâmetro, valor vazio), erro do SSM, resposta não 2xx (mensagem sem o token) e erro de rede | `adapter/notificacao/telegram_test.go` com `httptest` e um SSM falso (**hoje o envio real não tem teste**) |
| F4-T8 | A raiz perdeu tudo: nenhum desses arquivos existe; não há `var db`, `var table`, `ssmCli`, `enviarMensagem`; `main.go` só liga | inspeção do fonte, como F1-T6, F2-T7 e F3-T8 |
| F4-T9 | `service/` não importa o SDK da AWS e não chama `time.Now` em `avisos.go` | `arquitetura_test.go` + a checagem da fatia 3 estendida |
| F4-T10 | Mutação: tirar a deduplicação de chat, trocar o `Relogio` por `time.Now` e remover o filtro de `excluidoEm` derrubam os testes certos | verificada à mão |

## Risco e como o spec o controla

- É a fatia que mais mexe em arquivos (`main.go`, todos os testes de avisos, `local.go`), mas com o contrato fixo e sem dado novo.
- O envio real só aparece em produção (token no SSM): por isso o teste do adaptador (F4-T7) e a conferência em dev com o
  `POST /avisos/teste` e o agendamento rodando uma vez.
- **Sem migração de dados** e sem mudança em `template.yaml` (as mesmas três variáveis de ambiente).

## Pronto quando

- `go vet ./...` e `go test ./...` passam; `sam validate --lint` passa.
- No app de dev: salvar um lançamento futuro com aviso, ver o item aparecer em `AVISOS#ATIVOS`; "Testar aviso" no Perfil chega ao
  Telegram com `[DEV]`; o agendamento (ou o evento `{"acao":"avisos"}` pela Lambda) envia o resumo do dia.
- `store.go`, `avisos.go`, `telegram.go`, `perfil.go`, `efeitos.go` e `dominio_compat.go` não existem; não há variável global de
  dependência na raiz.
- `arquitetura.md`, `plan.md` e o item 32 do BACKLOG atualizados (o item 32 fecha), e o item 34 passa a apontar para o `Notificador`.

## Decisões para o dono (recomendação em negrito)

1. **A rotina diária lê o "hoje" pelo `Relogio`** em vez de receber `time.Now()` do `entrada`. O comportamento é o mesmo; os testes
   deixam de passar a data à mão.
2. **A junção dos efeitos de lançamento vira `service/efeitos.go`** (`EfeitosLancamento`), e não fica na raiz. Assim `efeitos.go` some.
3. **O adaptador do Telegram guarda o token em cache no próprio struct** (e não numa variável global), e ganha teste com servidor
   HTTP falso, que hoje não existe.
4. **A leitura do `ownerSub` e do chat id entra nos repositories que já existem** (`Espacos.Dono` e `Pessoas.ChatID`), e não num
   repository de avisos grande demais.
