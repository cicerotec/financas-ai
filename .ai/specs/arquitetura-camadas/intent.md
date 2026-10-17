# Intent: arquitetura em camadas (o que falta da raiz)

Item 32 do BACKLOG. Este documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md). O roteiro verificável
(`spec.md`) de cada fatia é escrito **quando a fatia começa**, depois do ok do dono, como foi feito no histórico.

**Estado:** rascunho para o dono aprovar. Nenhum código foi alterado. Contexto das decisões já fechadas:
[`../../arquitetura.md`](../../arquitetura.md) (camadas, regra de dependência, estrutura de pastas).

## Por quê

Os lançamentos, o histórico e o `auth` já vivem em pastas por responsabilidade (`domain/`, `auth/`, `repository/`,
`service/`, `controller/`), com a regra de dependência testada em `arquitetura_test.go`. O resto do backend ainda mora na
raiz, de antes da regra:

- **Variáveis globais:** `db`, `table` (store.go), `ssmCli` (telegram.go), `agora` (fechamento.go) e `enviarMensagem`
  (telegram.go). Os testes as trocam diretamente, o que impede rodar dois cenários ao mesmo tempo e esconde de quem lê
  *do que* cada função depende.
- **Regra misturada com banco e HTTP:** `fechamento.go`, `perfil.go`, `tags.go`, `seq.go` e as rotas de configuração
  (`getCfg`, `putCfg`, `putAparencia`) montam expressões do DynamoDB e respostas HTTP no mesmo lugar. A leitura de
  lançamentos (`listarTx`, `getTxs`) e `getMe` também.
- **Pontes temporárias:** `efeitos.go` e `dominio_compat.go` existem só para o service de lançamentos conversar com o que
  ainda está na raiz.
- **`main.go` faz mais que ligar:** ele roteia, lê configuração e ainda tem handlers dentro.

Isso é o que o princípio "o front não decide, o domínio é claro e todos o usam" exige do servidor: quando a próxima
regra chegar (fatura do cartão no item 33, despachante de avisos no 34, Open Finance no 39), ela entra numa camada com
dono, e não em mais um arquivo da raiz.

## O que queremos

1. **A raiz só liga as peças.** `main.go` cria os clients (DynamoDB, SSM), monta repository, service e controller **uma
   vez**, e entrega ao roteador. Nenhuma regra e nenhum handler na raiz.
2. **Nenhuma variável global** de dependência: `db`, `table`, `ssmCli`, `agora` e `enviarMensagem` passam a ser
   injetados por construtor. O relógio vira uma interface (`Relogio`) declarada pelo service.
3. **Cada assunto restante migra para as camadas**, uma fatia vertical por vez, com o app no ar entre elas:
   leitura de lançamentos e espaços; configurações, tags, contador e perfil; fechamento e saldos; avisos e Telegram.
4. **O contrato HTTP não muda.** Mesmas rotas, mesmos JSONs, mesmos códigos. O front e o Postman não percebem a migração.
5. **Os testes passam a montar o sistema com falsos injetados**, sem trocar globais; o `arquitetura_test.go` continua
   impondo quem pode importar quem, e ganha as pastas novas.
6. **`efeitos.go` e `dominio_compat.go` somem** quando a última fatia migra.

## Limites (o que fica de fora)

- **Não muda comportamento.** É refatoração: nenhuma regra nova, nenhum campo novo no DynamoDB, nenhuma rota nova.
- **O despachante de avisos com vários canais é o item 34.** A fatia de avisos aqui só injeta as dependências e tira as
  globais; a pasta `dispatcher/` e as strategies de canal nascem com o 34.
- **Capacidade da tabela, GSI por data e Open Finance** seguem nos seus itens (depois da refatoração).
- **Não reescreve o que já migrou** (lançamentos, histórico, `auth`).

## Riscos

| Risco | Resposta |
|---|---|
| Refatorar muito de uma vez e quebrar produção | Uma fatia por PR, release entre elas; o contrato HTTP é fixo e coberto pelos testes de API que já existem |
| Os testes antigos trocam as globais | Cada fatia converte os seus testes para o cenário com falsos injetados, no mesmo PR; nenhum teste é apagado para passar |
| A fatia de fechamento mexe na trava dos lançamentos | É a que mais risco leva; vem depois de a fundação estar no ar e tem os testes da regra de mês (A-T1 a A-T17) como rede |
