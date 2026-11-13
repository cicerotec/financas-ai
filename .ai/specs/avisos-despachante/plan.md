# Plan: avisos por despachante e vários canais (item 34)

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje (`backend/service/avisos.go`, `domain/aviso_envio.go`, `repository/pessoas.go`, `adapter/notificacao/telegram.go`, `servidor.go`, `main.go`
e `avisos_test.go`). O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** rascunho para aprovação (2026-11-13). Este plan **não cria chave nem campo no DynamoDB**. A **revisão independente das premissas** (guardrails, "Roteiros") **não foi feita**: o dono não tem agente para isso e a
**dispensou em 2026-11-13**; ela é pedida quando existir um. Enquanto isso, a seção "Premissas" traz só o que **quem escreveu** conseguiu levantar, e **não substitui** a revisão.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| A rotina agrupa as linhas **por chat id** (`porChat`) e manda `domain.MontarResumo(...)` como **texto** por `Notificador.Enviar(ctx, chat, texto)` | `service/avisos.go:140-272`, `service/avisos.go:19-23` | O `service` conhece o formato do texto. Para o push mostrar só a contagem, o conteúdo tem de sair **estruturado** e cada canal formatar |
| Recebem o aviso de um lançamento o **criador** e o **dono do espaço**; a fatura vai **só ao dono** | `service/avisos.go:200-205`, `236-248`, `.ai/avisos.md` | Regra de quem recebe **não muda** neste item |
| A mesma linha não vai duas vezes ao mesmo chat (`vistos[chat]`, por linha) quando criador e dono são a mesma pessoa ou usam o mesmo chat | `service/avisos.go:237-247`, teste `TestRotinaCriadorIgualAoDonoRecebeUmaSo` | O agrupamento por destino, com essa deduplicação, tem de continuar |
| `ErrCanalNaoConfigurado` faz a rotina inteira **parar** (`return enviados, nil`) | `service/avisos.go:261-264`, teste `TestRotinaSemVinculoOuSemTelegramNaoQuebra` | Com mais de um canal, isso derrubaria os outros. Vira "pula o resto deste canal" |
| `ChatID(sid, sub)` lê o vínculo (`USER#<sub>` / `SPACE#<sid>`) por `GetItem` e a rotina o guarda em cache por rodada | `repository/pessoas.go:36-47`, `service/avisos.go:148-163` | Vira `Destinos(sid, sub)`: o mesmo `GetItem`, devolvendo os destinos de todos os canais. Hoje só o Telegram |
| O resumo inteiro vai **numa mensagem**; o Telegram recusa acima de 4096 caracteres (uns 40 avisos de ~100 caracteres) | `.ai/avisos.md` (limites), `security.md` (pontos em aberto), `adapter/notificacao/telegram.go:67-89` | A divisão em partes é do canal Telegram, com funções puras em `domain/` |
| `Testar` manda um texto fixo ao Telegram de quem chamou; o controller traduz `ErrTelegramNaoConfigurado` em `503` e `ErrTelegramRecusou` em `502` | `service/avisos.go:275-293`, `controller/resposta.go:70-72`, teste `TestMensagemDeTeste` | O teste passa a ir pelo despachante, com as mesmas respostas HTTP |
| `novoServidor(..., notificador service.Notificador, ...)` recebe **um** canal; o `main` cria o `Telegram` | `servidor.go:30`, `main.go:25-26` | Passa a receber os canais (hoje um) e o `main` os registra |
| O teste usa um `notifFalso` que implementa `Notificador` e guarda `(chat, texto)` | `avisos_test.go:231-256` | Vira um canal falso. As **asserções** (texto, quem recebeu, contagem, status HTTP) continuam as mesmas |
| O `arquitetura.md` já decidiu: o despachante vive no `service`, canal novo é um arquivo em `adapter/notificacao/`, `adapter/*` não importa `service` | `arquitetura.md` (camadas e "Avisos: um despachante, vários canais") | A interface do canal é declarada no `service`; o adaptador a implementa só com tipos do `domain` |

## Desenho

- **Conteúdo (`domain`, puro).** `domain.ResumoAvisos{Hoje, Marca, Linhas, Teste}`: o dia, a marca do ambiente (`[DEV] `), as linhas (`LinhaAviso`, que já é estruturada) e a marca de "mensagem de teste". `MontarResumo` continua
  como está: é o texto do Telegram. Duas funções novas: `PartesDoResumo(r, limite)` (divide) e `TextoDeTeste(marca)` (o texto do teste, hoje no `service`).
- **Divisão.** Cabeçalho (`🔔 [DEV] Avisos de dd/mm/aaaa`) em toda parte, com ` (i/n)` quando há mais de uma; linhas na ordem de `MontarResumo` (atrasados, hoje, antes); nunca se corta uma linha; uma linha sozinha acima do limite é
  truncada com `…`. O limite é de **4000 caracteres** (contados em letras, `rune`), abaixo dos 4096 do Telegram, para sobrar margem.
- **Canal (`service`).** `type Canal interface { Enviar(ctx, destino string, r domain.ResumoAvisos) error }`. O Telegram o implementa: formata (`MontarResumo` ou as partes), envia cada parte pelo envio de texto que já existe (renomeado
  `EnviarTexto`, para o nome `Enviar` ser do canal) e devolve `domain.ErrCanalNaoConfigurado` quando falta o token, como hoje.
- **Despachante (`service`).** `Despachante` guarda os canais por nome (`"telegram"`) e `Entregar(ctx, Destino{Canal, Endereco}, r)`. É injetado no `Avisos` no lugar do `Notificador`. Canal desconhecido é erro de configuração, não pânico.
- **Destinos (`service` + `repository`).** `LeitorContatos.ChatID` vira `Destinos(ctx, sid, sub) (map[string][]string, error)`: `{"telegram": [chat id]}` quando a pessoa cadastrou o chat id. A **assinatura é implícita**: ter destino é estar
  assinado. Quando a fase 4 do item 54 chegar, o `push` entra no mesmo mapa, a partir do campo `pushTokens` do mesmo vínculo.
- **Rotina (`Avisos.Rodar`).** Para cada aviso que vale, para cada pessoa que deve recebê-lo, para cada destino dela: acrescenta a linha ao grupo `(canal, endereço)`, sem repetir a mesma linha no mesmo grupo. No fim, entrega cada
  grupo (em ordem estável) pelo despachante. Falha de um grupo vai ao log e a rotina segue; `ErrCanalNaoConfigurado` de um canal pula **o resto desse canal** (uma linha de log por canal por rodada) e segue com os outros.
  A contagem devolvida é a de grupos entregues por inteiro.
- **Agrupar por destino, como hoje (decidido em 2026-11-13).** O grupo é o `(canal, endereço)`, sem o espaço: o mesmo destino em dois espaços recebe **um** resumo, com as linhas dos dois e **sem o nome do espaço** (que apareceria na
  tela bloqueada). A pessoa escolhe de quais espaços o aparelho recebe aviso porque o registro é por espaço. Com um espaço, nada muda.
- **Teste de aviso.** `Testar` entrega `ResumoAvisos{Teste: true}` ao destino `telegram` de quem chamou, pelo despachante. As respostas (`200`, `400` sem chat id, `503`, `502`, `403`) continuam as mesmas.
- **Sem mudança de API.** O evento `{"acao":"avisos"}` e a resposta `{"enviados": n}` ficam; nenhuma rota nova; nenhuma ação nova em `auth/authz.go`.

## Fases (cada uma é um PR)

Uma só, **só backend**: o item não tem tela. Ela leva `domain` (conteúdo e divisão, com testes em tabela), `service` (`Canal`, `Despachante`, rotina e `Testar`), `adapter/notificacao` (o canal Telegram), `repository` (`Destinos`),
`servidor.go` e `main.go` (a ligação) e os testes. O release só depois de conferido no `dev`.

Ordem dentro do PR, para o teste de hoje continuar dizendo a mesma coisa em cada passo: (1) `domain` e testes; (2) canal Telegram com o envio renomeado; (3) `Despachante` e a rotina, com o `avisos_test.go` adaptado **sem mudar uma
asserção**; (4) a divisão no Telegram; (5) a ligação e o `Testar`.

## Premissas

Levantadas por quem escreveu o roteiro, lendo o código. **Não são a revisão independente** (dispensada pelo dono em 2026-11-13; pedir quando houver agente). Cada uma aponta a regra do `spec.md` que a cobre.

| # | A regra só funciona porque... | Se deixar de valer | Regra |
|---|---|---|---|
| P1 | **Hoje há um espaço e um canal**: agrupar por chat id (`porChat`, `service/avisos.go:150`) equivale a agrupar por pessoa | Com um segundo espaço, as contas dos dois cairiam no mesmo resumo (**aceito pelo dono**: um resumo por destino, sem nome de espaço) | R5 |
| P2 | `ErrCanalNaoConfigurado` parar a rotina inteira é inofensivo **só porque há um canal** (`service/avisos.go:261-264`) | Com dois canais, um canal sem configuração deixaria o outro sem aviso | R6, D-T5 |
| P3 | A rotina **roda uma vez por dia**: não há registro de "já enviei" | Se o agendamento disparar duas vezes, todos os avisos saem em dobro (já era assim; o push torna mais visível) | R15 (fora do escopo; decisão do dono) |
| P4 | O `enviados` conta **resumos**, e quem o lê (o evento agendado, `avisos_test.go`: `n != 2`) supõe uma mensagem por pessoa | Com partes, contar mensagens mudaria o número sem mudar o que a pessoa recebeu | R7 (conta grupos entregues por inteiro) |
| P5 | `MontarResumo` **ordena a lista que recebe, no lugar** (`sort.SliceStable`, `domain/aviso_envio.go:189`) | Dividir um resumo ordenaria a lista de quem chamou, e uma lista compartilhada entre grupos sairia embaralhada | R8, D-T2 |
| P6 | O envio é **em série** e o cliente HTTP do Telegram espera até 10 s por chamada, numa Lambda de **30 s** (`template.yaml`, `Globals`; `telegram.go:43`) | Muitos destinos ou partes com o Telegram lento podem estourar os 30 s e a rotina morre no meio (hoje já é possível, com um resumo por chat) | R15 (risco aceito; este item não muda o tempo limite) |
| P7 | A iteração de mapa em Go é **aleatória**: hoje a ordem dos envios entre chats não é estável | Teste e log que dependam da ordem ficariam instáveis | R5 (grupos em ordem estável) |
| P8 | O limite de 4096 do Telegram vale em unidades que o roteiro **não conferiu** (letras, UTF-16 ou bytes) | Um resumo perto do limite pode ser recusado mesmo "cabendo" | R8 (margem de 96) e D-T3 |

## Propostas que o dono pode contestar

1. Interpretação de "item 34 inteiro": despachante, divisão e isolamento, com o Telegram como único canal real; e-mail e WhatsApp sem código (ver `intent.md`).
2. Assinatura implícita (ter destino = estar assinado), sem campo novo.
3. Agrupar por destino, como hoje (decidido), sem o nome do espaço.
4. Limite de 4000 caracteres por mensagem, com ` (i/n)` no cabeçalho.
5. Idempotência da entrega **fora** deste item (pede dado novo).
6. Revisão independente das premissas **dispensada** por falta de agente; a seção "Premissas" é só do autor.

## Riscos

| Risco | Resposta |
|---|---|
| Mudar a rotina que avisa de contas a pagar e deixar de avisar | O `avisos_test.go` passa **sem mudar asserção**; quebrar de propósito (limite, isolamento) precisa derrubar o teste certo; conferência no `dev` antes do release |
| Telegram limita o ritmo (cerca de uma mensagem por segundo por chat) e recusa a 2ª parte | Só ocorre com resumos acima de 4000 caracteres, que hoje nem chegam. O roteiro manda tentar todas as partes e registrar a falha; a rotina não repete. Se acontecer, a repetição com espera é um passo seguinte |
| Entrega parcial: a 1ª parte chegou e a 2ª não | O cabeçalho ` (1/2)` mostra que falta; o destino não conta como entregue e o log registra |
| O limite de 4096 do Telegram ser em unidades diferentes de `rune` (por exemplo UTF-16) | A margem de 96 caracteres cobre a diferença nos emojis do texto; o teste de limite usa o texto com `🔔` e acentos |
| Logar chat id, texto ou valor por descuido | Regra R12 do spec e teste que procura isso no log |
| Dois canais com o mesmo nome no `main` | `NovoDespachante` recusa nome repetido (erro na subida, não em produção silenciosa) |
