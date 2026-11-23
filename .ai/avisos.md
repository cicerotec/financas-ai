# Avisos de lançamentos futuros (Telegram)

> **Mudança decidida em 2026-10-18, ainda não implementada.** O gatilho "status ainda é de evento futuro" passa a ser
> "marcador `previsto` ainda ligado" (e, depois da data, é isso que define o atraso). Mudar o status deixa de ser o gatilho do
> aviso; o encerramento passa a seguir o marcador (os detalhes saem no spec). O texto abaixo descreve o comportamento
> **atual**. Veja [`decisoes.md`](decisoes.md).

Cada lançamento pode ter um **aviso**: o app manda uma mensagem no Telegram nos dias combinados antes do vencimento e,
se marcado, **todo dia** depois dele até o lançamento deixar de ser evento futuro. Pensado para o `PREVISTO` com multa
(boleto, fatura), mas vale para qualquer status de evento futuro (`PREVISTO`, `TRANSFERINDO`, `CREDITANDO`).

## Como funciona
- No lançamento: `aviso: { dias: [3, 1, 0], insistir: true }`. `dias` = quantos dias antes do vencimento avisar (0 = no
  dia). `insistir` = avisa no dia e todo dia depois (até 60 dias de atraso). O vencimento é a data do lançamento no
  fuso de São Paulo.
- Todo dia às **8h** (horário de Brasília) uma regra do EventBridge chama a mesma Lambda da API
  (`{"acao":"avisos"}`). Ela junta o que vence e manda **um resumo por pessoa**: quem criou o lançamento e o dono do
  espaço (quem não vinculou o Telegram simplesmente não recebe).
- Só avisa se o status ainda for de evento futuro e o lançamento não estiver oculto. Mudar para `PAGO`, ocultar ou
  excluir para os avisos; voltar para `PREVISTO` os reativa.
- Para a rotina não varrer a tabela, cada lançamento com aviso tem um item auxiliar em `AVISOS#ATIVOS` (SK
  `<espaço>#<id>`), mantido ao criar, editar, excluir e importar.

## Configuração única (sua)
1. **Criar o bot:** no Telegram, converse com o `@BotFather`, envie `/newbot`, escolha nome e usuário. Ele devolve o
   **token** (`123456:ABC...`). Não o compartilhe nem o coloque no repositório.
2. **Guardar o token no SSM** (SecureString; **um bot por ambiente**: um bot aceita um webhook só, então o `dev` nunca usa o bot da produção). Use o script, que lê o token sem mostrá-lo e recusa o mesmo token nos dois ambientes (só o dono o roda):
   ```powershell
   .\scripts\guardar-telegram.ps1 -Ambiente prod
   .\scripts\guardar-telegram.ps1 -Ambiente dev
   ```
3. **Atualizar as roles do Actions** (nova permissão para criar a regra de agendamento) antes do próximo deploy:
   ```bash
   aws cloudformation deploy --stack-name financas-gha-dev  --template-file infra/github-oidc.yaml --capabilities CAPABILITY_NAMED_IAM --parameter-overrides Ambiente=dev  Repositorio=cicerotec@24279229/financas-ai@1394222228 --region sa-east-1
   aws cloudformation deploy --stack-name financas-gha-prod --template-file infra/github-oidc.yaml --capabilities CAPABILITY_NAMED_IAM --parameter-overrides Ambiente=prod Repositorio=cicerotec@24279229/financas-ai@1394222228 --region sa-east-1
   ```
4. **Registrar o webhook** de cada ambiente, **depois do deploy** (gera o segredo, registra o webhook e grava o segredo e o usuário do bot no SSM; recusa um bot que já tem outro webhook):
   ```powershell
   .\scripts\registrar-webhook-telegram.ps1 -Ambiente dev
   ```
5. **Cada pessoa:** em *Listas → Avisos no Telegram* liga a chave **Receber avisos no Telegram**. O app abre o Telegram; a pessoa toca em **Iniciar** e volta ao app, que mostra **Ligado** (o chat id nunca é digitado nem aparece). O link vale 10 minutos e uma vez; **Já iniciei** confere na hora e **Cancelar** desiste. O botão **Enviar aviso de teste** confirma que chegou. Quem tem mais de um espaço liga o Telegram em cada um. **Ao marcar um aviso num espaço sem nenhum canal ligado** (nem Telegram nem aparelho), a tela diz "Você ainda não tem um canal ativo para receber este aviso" com **Ligar um canal**, que abre uma janela com um botão por canal (Telegram e, só no app Android, Neste aparelho); o aviso é guardado mesmo sem canal e fica mudo até haver um. **Em Futuros e no Histórico**, o sino 🔔 dessas linhas (e a linha da fatura) passa a dizer **"Sem canal ativo"** enquanto o espaço não tiver nenhum canal. Desligar a chave apaga o destino do servidor; **para parar na hora, bloqueie o bot** (as mensagens cessam na rotina das 8h seguinte).

## Push no app (fase 4 do item 54)
- Segundo canal do despachante: a notificação do app Android, pelo Firebase Cloud Messaging. Cada aparelho registra o seu token (`PUT /spaces/{sid}/push`), guardado no campo `pushTokens` do vínculo da pessoa no espaço, e a rotina das 8h o recebe como destino, igual ao chat id do Telegram.
- **A notificação só traz a contagem** ("2 atrasados · 1 vence hoje"), nunca descrição, valor, meio de pagamento nem nome do espaço: ela aparece na tela bloqueada. O mesmo aparelho em dois espaços recebe uma notificação com a soma.
- O token que o FCM disser não existir mais (`UNREGISTERED`) é apagado dos vínculos pela própria rotina. A credencial do Firebase fica no SSM (`/financas/<ambiente>/fcm-credencial`, gravada por `scripts/guardar-fcm.ps1`); sem ela o push fica "não configurado" e o Telegram segue.
- Sem promessa de horário: a notificação sai às 8h do servidor e chega quando o aparelho a receber.

## Limites e decisões
- O Telegram é o único canal por enquanto; outros (Web Push, e-mail) entrariam como novos canais da mesma rotina.
- O horário é fixo (8h). Sem horário de verão no Brasil, a regra em UTC (`cron(0 11 * * ? *)`) não deriva.
- Se o token não existir no SSM, a rotina registra no log e encerra sem erro; o resto do app não é afetado.
- Visibilidade por pessoa (item 27 do BACKLOG): quando existir, os avisos devem respeitá-la.
- **O Telegram só traz a contagem (item 66, fase 1; decisão do dono, 2026-11-20):** a mensagem é uma linha, `🔔 Avisos de dd/mm/aaaa: 2 atrasados · 1 vence hoje · 1 vence amanhã · 2 vencem nos próximos dias. Abra o app para ver o que é cada um.`, com as mesmas frases do push (o que for zero some; entradas contam na situação do vencimento). **Os a vencer dizem o prazo (decisão do dono, 2026-11-22): "vence amanhã" quando falta 1 dia, "vence nos próximos dias" nos demais; "1 a vencer" sozinho não dizia quando.** Descrição, valor, meio de pagamento, data de vencimento e nome de espaço ou de pessoa **nunca** entram. Vale para todos, inclusive quem recebia o resumo com cada linha. O aviso de segurança do MFA e a mensagem de teste seguem com o texto fixo de sempre. Como a mensagem é curta, **não há mais divisão em partes nem limite de 4000 letras** (era o item 34).
- **Um resumo por pessoa e destino (item 66, D7):** os espaços da **mesma pessoa** no mesmo chat (ou no mesmo aparelho) recebem um resumo só, com a contagem somada e sem dizer de qual espaço. **Pessoas diferentes nunca se misturam**, mesmo dividindo o chat ou o token: cada uma recebe a sua mensagem, com a sua contagem. Vale para o Telegram e para o push. Antes o agrupamento era só por destino.
- **Destino recusado é apagado (item 66, fase 1):** `403` (bot bloqueado, conta apagada) e `400 chat not found` fazem a rotina apagar o chat id dos vínculos de onde ele saiu, como já fazia com o aparelho do push (`UNREGISTERED`). `401` do bot é credencial recusada: a rotina pula o canal e loga o texto próprio. Qualquer outro erro (`429`, `5xx`, rede) só vai ao log com o código HTTP, nunca com a URL, o token, o corpo da resposta nem o chat id, e não é repetido na rodada.
- **Falha de canal:** um canal sem configuração (ou que falha) não impede os outros; a rotina não devolve erro por falha de envio.
- As datas das mensagens trazem o ano (`dd/mm/aaaa`).
