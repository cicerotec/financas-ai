# Intent: o app aberto a qualquer pessoa, cada uma no seu espaço

Este documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md).

**Estado:** rascunho para aprovação (2026-10-27). Nada foi implementado.

## O problema

O app nasceu para uma família: o Cognito só aceita convite (`AllowAdminCreateUserOnly: true`, `template.yaml`) e o espaço de cada pessoa é criado à
mão pelo `scripts/seed.sh`. Para uma pessoa qualquer usar o app, o dono teria de rodar um script por ela, e hoje ninguém de fora entra sozinho.
Além disso, abrir a porta muda o risco: o app guarda dados financeiros, não tem MFA, a Function URL não limita requisições e a tabela é
provisionada em 5 RCU/WCU.

## O que queremos

1. **Cadastro aberto, mas só com código de convite.** A pessoa cria a conta sozinha, informando um código que o dono distribui e pode revogar.
2. **Espaço próprio no primeiro acesso.** Quem entra e não tem espaço ganha um, vazio, onde é `owner`. Nenhum dado de outra pessoa aparece.
3. **App vazio utilizável.** A primeira tela leva a pessoa a criar o primeiro meio de pagamento; nada de nome de banco, tag ou status do dono.
4. **Proteção antes de abrir:** MFA por aplicativo autenticador (item 37), teto de custo e de uso (AWS Budget do item 11, limite de taxa).
5. **Isolamento entre espaços provado por teste**, e a pessoa pode **exportar e excluir a própria conta**.

## Decisões já tomadas (dono, 2026-10-27)

- Cadastro **aberto só com código de convite** (não livre, não só por convite individual).
- **MFA, limite de taxa e teto de custo vêm antes** de abrir o cadastro.
- **Ninguém lê dados de outra pessoa, nem o dono como admin.** Suporte só pelo JSON que a própria pessoa exporta.

## Limites (o que fica de fora)

- **"Nem o admin" é, nesta entrega, isolamento lógico**: a API nunca cruza espaços e o dono não tem tela nem rota para ler dado alheio. Quem tem o
  console da AWS ainda consegue ler a tabela. Impedir isso de verdade é a **cifragem por campo com KMS**, já discutida em
  [`.ai/privacidade-e-cifragem.md`](../../privacidade-e-cifragem.md) e **fora deste escopo**. Isto precisa ficar dito aos usuários na política de privacidade.
- Nenhuma chave nova no DynamoDB sem o dono aprovar (o modelo `USER#<sub>` / `SPACE#<id>` já basta para um espaço novo). Pontos que pedem conversa estão no
  plan.
- Login social (Google, Apple), recuperação de conta por suporte e planos pagos.
- Convidar outras pessoas para o **mesmo** espaço (o papel `member`) segue como hoje, por `seed.sh`; uma tela do dono para isso é o item 27 do BACKLOG.

## Custo

Depende de quantas pessoas entrarem. Cognito, Lambda e CloudFront têm faixa gratuita; a tabela em 5/5 vira gargalo antes de virar custo. O teto de custo
(AWS Budget) existe para o app avisar antes de a conta surpreender. Limite de taxa por WAF **não é grátis**: está como decisão no plan.

## Pronto quando

- Uma pessoa sem convite individual, só com o código, cria a conta no `dev`, confirma o e-mail, ativa o MFA e cai num app vazio com espaço próprio.
- Um teste cria dois espaços e prova que nenhuma rota de um lê, grava ou lista o outro; um código errado ou revogado não cria conta.
- Exportar e excluir conta funcionam e a exclusão cobre também o `HIST#` (que a Lambda não pode apagar hoje) e o usuário do Cognito.
- `security.md`, `BACKLOG.md` e o `README.md` ficam em dia; o MFA fecha o item 37.

## Para o dono decidir

- Como limitar taxa: teto de concorrência da Lambda (grátis, mas derruba todos juntos), WAF atrás do CloudFront (cobrado), ou nada além do teto.
- Capacidade da tabela quando houver mais gente: manter 5/5, autoscaling dentro das 25 unidades gratuitas ou sob demanda (cobrado por uso).
- Um código só para todos ou vários códigos com limite de usos (vários exigem registro, que seria **chave nova no DynamoDB**, ou parâmetros no SSM).
- MFA obrigatório para todos ou opcional para quem não é dono.
- Quem tem espaço próprio e depois é convidado para outro espaço: o app já devolve vários espaços em `GET /me`; confirmar como o front escolhe.
- Política de privacidade e termos de uso (texto, retenção, o aviso de que o dono tecnicamente alcança o console): quem redige e onde publica.
