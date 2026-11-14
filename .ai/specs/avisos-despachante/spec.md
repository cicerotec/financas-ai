# Spec: avisos por despachante e vários canais (item 34)

O roteiro **verificável** do [`plan.md`](plan.md), que segue o [`intent.md`](intent.md). Regras com número (`R`), testes com id (`D-T<n>`), para o PR dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale o plan.

**Estado:** feito (PR 429, 2026-11-13) e **emendado pelo PR 0 da [spine de arquitetura](../../bmad/arquitetura-avisos-canais.md) (2026-11-14):** texto, limite e divisão do Telegram (`MontarResumo`, `PartesDoResumo`, `TextoDeTeste`, `LimiteTelegram`, `Moeda`) saíram do `domain/` para `adapter/notificacao/telegram_texto.go`; o `domain` guarda só o conteúdo e a ordem (`OrdenarLinhas`); `Testar` recebe o canal e o `service` não nomeia canal. Onde este texto diz "funções puras de `domain/`", leia "do canal". A revisão independente das premissas foi **dispensada pelo dono** (2026-11-13, sem agente para isso; pedir quando existir); as premissas levantadas por quem escreveu estão no `plan.md` e cada uma aponta a sua regra aqui. Convenção dos testes: os de lógica são `go test ./...` em `backend/` e **falham antes da mudança**; o que
depende do `dev` é marcado **[dev]** e entra como roteiro manual no PR, com o resultado escrito. Diga o que não deu para ver em vez de afirmar que funciona.

## Regras

- **R1. O `service` manda conteúdo estruturado.** A rotina e o teste de aviso entregam um `domain.ResumoAvisos{Hoje, Marca, Linhas, Teste}`; **nenhum texto de aviso é montado no `service`**. O texto é do canal, por funções puras
  de `domain/`.
- **R2. Camadas.** A interface `Canal` é declarada no `service`; o Telegram a implementa em `adapter/notificacao/` usando só tipos do `domain`. O `arquitetura_test.go` continua verde (o `service` não importa o adaptador, nem o
  adaptador o `service`).
- **R3. Despachante.** `service.Despachante` guarda canais por nome e entrega um `ResumoAvisos` a um `Destino{Canal, Endereco}`. Nome de canal repetido, na montagem, é erro. Destino de canal desconhecido é erro **de entrega**
  (logado, o grupo não conta), nunca pânico.
- **R4. Destinos.** `LeitorContatos.Destinos(sid, sub)` devolve os destinos da pessoa naquele espaço, por canal. Hoje só `telegram`, a partir de `telegramChatId` do vínculo. **Estar assinado é ter destino**; não há campo de
  "canais ligados". Nenhum campo, chave ou item novo no DynamoDB.
- **R5. Agrupamento.** A rotina junta as linhas por **`(canal, endereço)`**, como hoje (por chat id). Duas pessoas com o mesmo chat id recebem **um** resumo com as linhas das duas, e a **mesma linha** nunca aparece duas vezes
  no mesmo grupo (criador igual ao dono, ou mesmo chat). O mesmo destino registrado em dois espaços recebe **um** resumo só, com as linhas dos dois, **sem o nome do espaço** (decisão do dono, 2026-11-13: o nome apareceria na tela
  bloqueada). Os grupos são entregues em **ordem estável** (por canal e endereço).
- **R6. Isolamento de falha.** A falha de um grupo não impede os outros. `domain.ErrCanalNaoConfigurado` de um canal pula **o resto dos grupos desse canal**, com **uma** linha de log por canal por rodada, e a rotina continua
  com os outros canais. A rotina **não devolve erro** por falha de envio (só por falha de leitura, como hoje).
- **R7. Contagem.** O `enviados` do evento diário é o número de grupos entregues **por inteiro** (todas as partes). Sem canal configurado ou sem destino, `0` e sem erro.
- **R8. Divisão do Telegram.** Um resumo cujo texto cabe em **4000 caracteres** (`rune`) sai **idêntico** ao `MontarResumo` de hoje, numa mensagem. Acima disso é dividido em partes: cada uma começa com o cabeçalho
  (`🔔 <marca>Avisos de dd/mm/aaaa`), acrescido de ` (i/n)`; as linhas seguem a ordem de `MontarResumo` (atrasados, hoje, antes; do vencimento mais antigo ao mais novo); **uma linha nunca é cortada ao meio**; uma linha sozinha
  acima do limite é truncada com `…`. Nenhuma linha é perdida nem repetida entre as partes. A ordenação trabalha numa **cópia**: a lista de quem chamou não é alterada.
- **R9. Envio das partes.** As partes saem em ordem. Falha numa parte **não impede as seguintes**; o grupo conta como não entregue e o erro devolvido é o da **primeira** que falhou. `ErrCanalNaoConfigurado` na primeira parte
  encerra o envio daquele grupo (as outras falhariam igual).
- **R10. Marca do ambiente.** O `[DEV] ` vem no cabeçalho de **toda** parte (e o teste de aviso mantém "🔔 [DEV] Teste...").
- **R11. Teste de aviso.** `POST /spaces/{sid}/avisos/teste` entrega `ResumoAvisos{Teste: true}` ao Telegram de quem chamou, pelo despachante. Respostas: `200`; `400` sem chat id; `503` canal não configurado; `502` o Telegram
  recusou; `403` fora do espaço. O texto do teste é o de hoje.
- **R12. Log.** Nada do que o despachante e os canais registram contém chat id, texto do resumo, descrição, valor ou token. Só contagens e o nome do canal. (O erro do Telegram já não vaza o token; continua assim.)
- **R13. O que não muda.** Quem recebe (criador e dono; a fatura, só o dono), o horário (8h), os filtros (previsto, oculto, excluído, mês fechado) e a recorrência da mesma rotina continuam como hoje.
- **R14. API.** O evento `{"acao":"avisos"}` e a resposta `{"enviados": n}`, as rotas e as ações de `auth/authz.go` não mudam. O front não muda.
- **R15. Fora do escopo, por escrito.** E-mail, WhatsApp, a **idempotência da entrega** (aviso em dobro se o agendamento disparar duas vezes) e o **tempo limite da Lambda** (30 s, com envio em série; risco aceito até o dono decidir) **não são tratados** neste item.

## Testes

| Id | O que prova |
|---|---|
| D-T1 | `PartesDoResumo` abaixo do limite devolve **uma** parte, igual a `MontarResumo`; no limite exato, uma; um caractere acima, duas (R8) |
| D-T2 | Em tabela: 1, 2 e 5 partes; cabeçalho com ` (i/n)` e marca `[DEV] ` em todas; ordem preservada; nenhuma linha perdida ou repetida (a união das linhas das partes é a lista original); a lista passada **não é reordenada** (R8, R10) |
| D-T3 | Linha sozinha acima do limite é truncada com `…` e ainda cabe; o texto com `🔔`, acentos e `—` é contado em letras (R8) |
| D-T4 | Rotina com **dois canais falsos**: um falha, o outro entrega; o `enviados` conta só o entregue; a rotina não devolve erro (R6, R7) |
| D-T5 | Rotina com um canal que devolve `ErrCanalNaoConfigurado` e outro que entrega: o segundo entrega; o primeiro tem **uma** linha de log por rodada e os grupos seguintes dele são pulados (R6) |
| D-T6 | Criador igual ao dono: um resumo, sem linha repetida; duas pessoas do mesmo espaço com o mesmo chat id: **um** resumo com as linhas das duas; o mesmo chat id em dois espaços: **um** resumo com as linhas dos dois; os grupos saem na mesma ordem em duas rodadas (R5) |
| D-T7 | Canal Telegram com servidor HTTP falso: 3 partes saem em ordem, na API do bot; a 2ª falha e a 3ª ainda sai; o destino conta como não entregue; o erro é o da 2ª (R9) |
| D-T8 | Sem token, a 1ª parte devolve `ErrCanalNaoConfigurado` e nenhuma outra é tentada (R9) |
| D-T9 | O `avisos_test.go` de hoje (rotina, criador e dono, insistir, oculto, excluído, sem vínculo, status futuro, teste de aviso, marca `[DEV]`) passa com o canal falso novo **sem mudar uma asserção** (R1, R7, R10, R11, R13) |
| D-T10 | `TestMensagemDeTeste`: `200`, `400`, `503`, `502`, `403` como hoje (R11) |
| D-T11 | `arquitetura_test.go` verde; o `service` não importa o adaptador (R2) |
| D-T12 | Nome de canal repetido no `NovoDespachante` é recusado; destino de canal desconhecido não derruba a rotina (R3) |
| D-T13 | Nenhuma linha de log da rotina e dos canais contém o chat id, o texto do resumo ou o token (R12) |
| D-T14 **[dev]** | Um resumo real de poucas linhas chega igual ao de hoje; um resumo de dados fictícios com mais de 4000 caracteres chega em partes, na ordem e com o ` (i/n)` (R8) |

**Quebrar de propósito:** tirar o limite (ou pô-lo em 10.000) tem de derrubar D-T1 e D-T7; fazer o canal que falha derrubar a rotina tem de derrubar D-T4 e D-T5; repetir a mesma linha no mesmo grupo, ou mandar dois resumos ao mesmo destino, tem de derrubar D-T6.

## Verificação no `dev`

1. Fazer o deploy e conferir o changeset e o status final da stack (guardrails): a tabela e o login **não** podem ser substituídos. Este item não toca o `template.yaml`.
2. Cadastrar o chat id do `dev`, criar um previsto com aviso para hoje e rodar o evento `{"acao":"avisos"}`; conferir que a mensagem chega no Telegram igual à de hoje.
3. Com dados fictícios, criar avisos até passar de 4000 caracteres e rodar o evento de novo; conferir as partes.
4. No Logs Insights de `/aws/lambda/financas-api-dev`, conferir que as linhas de log da rotina trazem só contagens e o nome do canal.

## Fora do escopo

E-mail, WhatsApp, idempotência da entrega, mudar quem recebe o aviso, o push (fase 4 do item 54, que depende deste), qualquer tela e qualquer campo novo no DynamoDB.
