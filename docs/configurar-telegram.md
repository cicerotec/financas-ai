# Configurar o Telegram (dev e produção)

Passo a passo, feito **uma vez por ambiente**, para o Finanças mandar avisos e vincular o Telegram de cada pessoa. Quem executa é o dono: os scripts de token e de webhook nunca são rodados pela IA.

## Antes de começar

| Regra | Por quê |
|---|---|
| **Um bot por ambiente** (um para o `dev`, outro para a produção) | O Telegram aceita **um webhook por bot**. Registrar o webhook do `dev` num bot da produção tira os `/start` da produção do ar |
| O token nunca vai ao repositório, ao chat, à linha de comando nem a um log | O token dá controle total do bot. Se vazar, revogue (seção "Se o token vazar") |
| Ordem fixa: **bot, token, deploy, webhook** | O webhook precisa da URL da API, que só existe depois do deploy |

Requisitos: AWS CLI, PowerShell, acesso à conta AWS com MFA, e o Telegram instalado.

Os parâmetros ficam no SSM, região `sa-east-1`, por ambiente:

| Parâmetro | Tipo | Quem grava |
|---|---|---|
| `/financas/<amb>/telegram-token` | SecureString | `guardar-telegram.ps1` |
| `/financas/<amb>/telegram-webhook-secret` | SecureString | `registrar-webhook-telegram.ps1` |
| `/financas/<amb>/telegram-bot-usuario` | String | `registrar-webhook-telegram.ps1` |

`<amb>` é `dev` ou `prod`. As stacks do CloudFormation são `financas-dev` (dev) e `financas` (prod).

## 1. Criar o bot no @BotFather

Faça duas vezes: um bot para o `dev` e outro para a produção.

1. No Telegram, abra o **@BotFather** (só o oficial, com selo de verificado).
2. Envie `/newbot`.
3. **Nome** (aparece no topo da conversa; pode ter espaço e acento):
   - prod: `Finanças AI`
   - dev: `Finanças AI [DEV]`
4. **Usuário** (único no Telegram, sem espaço, termina em `bot`). Escolha dois nomes distintos e anote-os, por exemplo `<nome>_financas_bot` para a produção e `<nome>_financas_dev_bot` para o dev.
5. O BotFather responde com o **token**, no formato `123456789:AAH...`. **Não o copie para lugar nenhum além do passo 2.** Ele pode ser consultado de novo a qualquer hora em `/mybots` → o bot → **API Token**.

### Alterar o nome do bot

O **nome** (o texto no topo da conversa) pode mudar a qualquer hora; o **usuário** (`@..._bot`) não muda depois de criado. Para trocar o nome:

1. No @BotFather, envie `/setname`.
2. Escolha o bot na lista.
3. Envie o novo nome (até 64 caracteres).

A troca vale na hora, não mexe no token nem no webhook, e não exige deploy. Se precisar de outro usuário, só criando um bot novo, o que obriga a refazer os passos 2 e 4 desse ambiente e a religar o Telegram de cada pessoa.

### Foto e textos do bot

Todos os comandos abaixo: envie ao @BotFather, escolha o bot na lista e responda ao que ele pedir. Cada ajuste vale para **um bot só**, então repita para os dois.

| Comando | O que é | Limite |
|---|---|---|
| `/setuserpic` | Foto do bot | Imagem quadrada, 512×512 |
| `/setdescription` | Texto que aparece **antes** de tocar em Iniciar | 512 caracteres |
| `/setabouttext` | Texto do perfil do bot | 120 caracteres |

**Fotos** (envie como foto, não como arquivo):

| Bot | Arquivo |
|---|---|
| prod | `web/icons/icon-512-v1.png` |
| dev | `web/icons/dev/icon-512-v1.png` (o mesmo ícone com a faixa laranja **DEV**) |

**Textos da produção**

`/setabouttext`:
```
Avisos de lançamentos futuros do Finanças AI. Só envio a contagem; os detalhes ficam no app.
```

`/setdescription`:
```
Este bot envia os avisos de lançamentos futuros do Finanças AI: quantos estão atrasados, vencem hoje ou estão a vencer, todo dia às 8h (Brasília).

Para receber, abra o app, vá em Listas → Avisos no Telegram e ligue a chave. O app abre esta conversa; toque em Iniciar.

A mensagem traz só a contagem. Descrições e valores ficam no app. Para parar, bloqueie o bot.
```

**Textos do dev**

`/setabouttext`:
```
[DEV] Bot de testes do Finanças AI. Os avisos daqui não são do ambiente real.
```

`/setdescription`:
```
[DEV] Ambiente de desenvolvimento do Finanças AI. Os avisos daqui são de teste.

Para vincular, use o app de desenvolvimento: Listas → Avisos no Telegram, ligue a chave e toque em Iniciar.
```

Para conferir, abra o bot numa conta que ainda não o tenha iniciado: a descrição só aparece antes do primeiro **Iniciar**.

## 2. Guardar o token no SSM

Em cada ambiente, no PowerShell, na raiz do repositório:

```powershell
. .\scripts\aws-mfa.ps1 <código do MFA>
.\scripts\guardar-telegram.ps1 -Ambiente dev
.\scripts\guardar-telegram.ps1 -Ambiente prod
```

O script pede o token com o texto oculto: cole-o, **Enter**, e pronto. Ele:

- valida o formato (`número:letras`);
- **recusa o mesmo token nos dois ambientes**, o que protege contra colar o bot errado;
- grava como SecureString em `/financas/<amb>/telegram-token`, por um arquivo temporário que é sobrescrito e apagado;
- nunca imprime o token.

Termina com `Token gravado em /financas/<amb>/telegram-token`. Rodar de novo troca o token.

## 3. Fazer o deploy do ambiente

O webhook aponta para a API do ambiente, então a stack tem de existir e estar atualizada. O deploy segue o fluxo normal: [`.ai/deploy-github-actions.md`](../.ai/deploy-github-actions.md) e [`.ai/fluxo-de-versoes.md`](../.ai/fluxo-de-versoes.md). Sem a stack, o script do passo 4 falha com "Nao consegui ler o ApiUrl".

## 4. Registrar o webhook

**Depois** do deploy e do passo 2, com a sessão do MFA ativa:

```powershell
.\scripts\registrar-webhook-telegram.ps1 -Ambiente dev
.\scripts\registrar-webhook-telegram.ps1 -Ambiente prod
```

O script, nesta ordem:

1. lê o token do SSM e pergunta ao Telegram quem é o bot (`getMe`);
2. confere o webhook atual (`getWebhookInfo`) e **recusa** se o bot já tem um webhook de outra URL;
3. gera o segredo (32 bytes aleatórios, 43 caracteres);
4. chama `setWebhook` com a URL `<ApiUrl da stack>/telegram/webhook`, o segredo e `allowed_updates: ["message"]`;
5. só então grava o segredo em `/financas/<amb>/telegram-webhook-secret` e, por **último**, o usuário do bot em `/financas/<amb>/telegram-bot-usuario`.

O usuário vai por último de propósito: o app só gera o link do Telegram quando ele existe, então nunca aponta para um webhook que não está registrado.

Mensagens que podem aparecer:

| Mensagem | O que fazer |
|---|---|
| `Nao achei /financas/<amb>/telegram-token` | Rode o passo 2 desse ambiente |
| `Nao consegui ler o ApiUrl da stack` | Faça o deploy (passo 3) |
| `O Telegram recusou ... (HTTP 401)` | Token errado ou revogado: refaça o passo 2 |
| `Este bot ja tem um webhook de OUTRA URL` | Confirme que o bot é mesmo deste ambiente. Se for, rode de novo com `-Forcar`. Se não for, você colou o token do outro ambiente |
| `...mas nao consegui gravar o segredo` (ou o usuário) | A sessão do MFA expirou. Renove e rode de novo; é seguro repetir |

Rodar o script de novo gera **outro segredo**. O servidor passa a aceitar o novo em até 5 minutos; nesse intervalo o Telegram pode receber 401 e reenvia sozinho.

## 5. Conferir no app

Em cada ambiente, com o app do ambiente:

1. **Listas → Avisos no Telegram** e ligue **Receber avisos no Telegram**.
2. O app abre o bot. Toque em **Iniciar**.
3. Volte ao app: ele mostra **Ligado**. Se não mostrar, toque em **Já iniciei**. O link vale 10 minutos e uma só vez.
4. Toque em **Enviar aviso de teste**: a mensagem tem de chegar no bot certo. No `dev` ela traz `[DEV]`.

Se o app disser que o Telegram não está configurado, falta o passo 4 (o usuário do bot não foi gravado). Se o **Iniciar** não responde "Ligado ✓", o webhook não chegou: confira o passo 4 e o ambiente do app.

## Se o token vazar

1. No @BotFather: `/revoke`, escolha o bot. O token antigo morre na hora.
2. Pegue o novo token em `/mybots` → o bot → **API Token**.
3. Refaça o passo 2 e o passo 4 desse ambiente (o webhook registrado continua o mesmo, mas o segredo se renova).

Se o token apareceu num chat com a IA, num commit ou num log, trate como vazado.

## Parar os avisos

- Uma pessoa: desligar a chave em **Listas → Avisos no Telegram** apaga o destino no servidor. Para parar na hora, bloqueie o bot; o destino recusado é apagado na rotina das 8h seguinte.
- Um ambiente inteiro: `/deletebot` no @BotFather apaga o bot e **não tem volta**. Para só desativar, `/revoke` e não guarde o token novo.

## Referências

- Como os avisos funcionam: [`.ai/avisos.md`](../.ai/avisos.md)
- Desenho do vínculo: [`.ai/specs/telegram-vinculo/plan.md`](../.ai/specs/telegram-vinculo/plan.md)
- Scripts: [`scripts/guardar-telegram.ps1`](../scripts/guardar-telegram.ps1), [`scripts/registrar-webhook-telegram.ps1`](../scripts/registrar-webhook-telegram.ps1), [`scripts/aws-mfa.ps1`](../scripts/aws-mfa.ps1)
