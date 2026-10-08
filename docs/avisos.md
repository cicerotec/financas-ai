# Avisos de lançamentos futuros (Telegram)

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
2. **Guardar o token no SSM** (SecureString; um por ambiente, pode ser o mesmo bot ou bots diferentes):
   ```bash
   aws ssm put-parameter --region sa-east-1 --type SecureString --name /financas/prod/telegram-token --value "<token>"
   aws ssm put-parameter --region sa-east-1 --type SecureString --name /financas/dev/telegram-token  --value "<token>"
   ```
3. **Atualizar as roles do Actions** (nova permissão para criar a regra de agendamento) antes do próximo deploy:
   ```bash
   aws cloudformation deploy --stack-name financas-gha-dev  --template-file infra/github-oidc.yaml --capabilities CAPABILITY_NAMED_IAM --parameter-overrides Ambiente=dev  Repositorio=cicerotec@24279229/financas-ai@1394222228 --region sa-east-1
   aws cloudformation deploy --stack-name financas-gha-prod --template-file infra/github-oidc.yaml --capabilities CAPABILITY_NAMED_IAM --parameter-overrides Ambiente=prod Repositorio=cicerotec@24279229/financas-ai@1394222228 --region sa-east-1
   ```
4. **Cada pessoa:** abre o bot no Telegram e toca em **Iniciar** (o bot só consegue escrever depois disso), descobre o
   próprio **chat id** (converse com o `@userinfobot`, que responde com o `Id`) e cola no app em *Configurações → Avisos*.
   O botão **Enviar aviso de teste** confirma que chegou.

## Limites e decisões
- O Telegram é o único canal por enquanto; outros (Web Push, e-mail) entrariam como novos canais da mesma rotina.
- O horário é fixo (8h). Sem horário de verão no Brasil, a regra em UTC (`cron(0 11 * * ? *)`) não deriva.
- Se o token não existir no SSM, a rotina registra no log e encerra sem erro; o resto do app não é afetado.
- Visibilidade por pessoa (item 27 do BACKLOG): quando existir, os avisos devem respeitá-la.
