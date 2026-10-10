# Colaboração em tempo real no app (presença + atualizações ao vivo)

## Contexto

App próprio usado por 2 pessoas (eu e minha esposa).

Stack atual:
- Front estático em **S3 + CloudFront**
- Autenticação com **Cognito**
- Backend em **Lambda (Go)**
- Banco **DynamoDB**

Restrição: **não usar API Gateway**. As chamadas ao backend passam pelo CloudFront (behavior `/api/*` → Lambda Function URL).

## Objetivo

Comportamento estilo Google Sheets / "digitando..." do WhatsApp, em escala doméstica:

1. A esposa abre o app e começa a criar um registro.
2. Eu abro o app minutos depois e vejo um alerta: "esposa está criando um registro".
3. Vou ao histórico e vejo o rascunho dela sendo preenchido.
4. Abro o registro e ela vê "Cicero está aqui".
5. Quando alguém sai ou fecha o app, a presença some sozinha.

Não há edição simultânea pesada. O foco é presença, rascunho visível e atualização sem refresh manual.

## Conceitos-chave (como Google/Microsoft fazem)

- Cada edição é uma **operação pequena**, nunca o documento inteiro.
- O **servidor define a ordem oficial** (número de sequência `seq` por "casa"/documento).
- O cliente aplica a mudança localmente na hora (otimista) e confirma quando a op volta do servidor.
- Conflito na mesma entidade/campo: **last write wins** pela ordem do servidor.
- Use **IDs estáveis (UUID)** para registros e campos, nunca posições ou índices. Isso elimina a necessidade de Operational Transformation.
- **Presença é efêmera e separada dos dados.** "Fulano está vendo/criando X" não é dado de negócio.
- Evento efêmero publicado sem ninguém ouvindo **se perde**. Por isso o rascunho precisa de **auto-save** no banco.

## Opção A — Polling via arquivo no CloudFront (sem novos serviços)

### Ideia

Separar "**mudou algo?**" (arquivo minúsculo no CloudFront) de "**o que mudou?**" (Lambda, só quando necessário).

### Arquivo de cabeçalho

A Lambda regrava `s3://<bucket>/casa/{casaId}/head.json` a cada gravação relevante (registro salvo, auto-save de rascunho, heartbeat de presença):

```json
{
  "seq": 4513,
  "ativos": [
    { "user": "esposa", "acao": "criando", "registro": "r_77", "visto_em": 1759600000 }
  ]
}
```

- Gere o JSON **a partir do DynamoDB** (fonte da verdade), nunca editando o arquivo anterior. Isso evita que duas Lambdas concorrentes apaguem dados uma da outra.
- Filtre de `ativos` quem tem `visto_em` mais antigo que ~90s.

### CloudFront

- Behavior para `casa/*/head.json` com cache policy de **TTL = 1s**.
- Behavior `/api/*` → Lambda Function URL (OAC), **sem cache**.

### Cliente

```js
async function tick() {
  const r = await fetch(`/casa/${casaId}/head.json`, { cache: 'no-store' });
  const head = await r.json();
  if (head.seq > meuSeq) await buscarMudancas(meuSeq); // GET /api/ops?after=meuSeq
  mostrarPresenca(head.ativos);
}
```

- **NÃO** usar `?t=timestamp` na URL (fura o cache do CloudFront). `cache: 'no-store'` só afeta o cache do navegador.
- Intervalo adaptativo: ~1s com outros ativos ou mudança recente, 5–10s ocioso, **parar** com a aba em segundo plano (`visibilitychange`).
- Atraso esperado: até ~2–3s.

### Limitação

Toda presença precisa passar por Lambda + DynamoDB + S3, porque o arquivo só muda quando alguém grava. Status ("criando", "vendo") funciona bem. "Digitando" letra a letra não é prático.

## Opção B — Push com AWS AppSync Events (WebSocket gerenciado)

### Ideia

O front continua em S3 + CloudFront. O browser abre **um** WebSocket para o AppSync Events (com auth Cognito) e assina canais. Não precisa de API Gateway nem de tabela de conexões.

### Canais

- `/casa/{casaId}`: avisos gerais (quem está online, quem está criando o quê).
- `/registros/{registroId}`: assinado só enquanto o registro está aberto (campos mudando, "fulano está aqui").

### Fluxo

- **Dados** (registro, rascunho): front → Lambda Go → grava no DynamoDB → Lambda publica no canal (POST HTTP assinado com IAM).
- **Presença/digitando**: o cliente publica direto no canal, sem Lambda e sem banco.

```js
import { events } from 'aws-amplify/data';

const ch = await events.connect(`/casa/${casaId}`);
ch.subscribe({ next: (msg) => tratar(msg) });

events.post(`/casa/${casaId}`, { tipo: 'quem-esta-ai?' });
```

### Ciclo de vida

1. **App abre**: conecta e assina `/casa/{id}`, depois publica `quem-esta-ai?`. Quem estiver online responde com o próprio status.
2. **Criando registro**: publica `{tipo:'criando', registro}` + auto-save do rascunho via Lambda.
3. **Abre registro**: assina `/registros/{id}` e publica `{tipo:'vendo'}`. Edições publicadas com debounce de ~300ms.
4. **Heartbeat** a cada ~30s. Os outros expiram a presença após ~60–90s sem notícia (cobre fechamento abrupto e app em segundo plano no celular).
5. **Sai/fecha**: unsubscribe + publica "saí". Em segundo plano, desconecta.
6. **Reconexão**: uma única chamada `GET /api/ops?after={lastSeq}` para recuperar o que perdeu.

### Custo (referência da página de preços da AWS)

- US$ 1,00 por milhão de operações (publicação, entrega a cada inscrito, conexão, inscrição, ping). Cobrança por bloco de 5 KB de payload.
- US$ 0,08 por milhão de minutos de conexão.
- Estimativa para 2–5 usuários: centavos por mês. Só há conexão (e custo) enquanto o app está aberto.
- Confirmar se o Event API está disponível na região usada.

## Backend comum às duas opções

### Sequenciamento na Lambda Go (ordem total sem buracos)

1. Ler `seq` atual da casa (N).
2. `TransactWriteItems`:
   - `Update casa SET seq = N+1` **com condição** `seq = N`
   - `Put op` (PK `casaId`, SK `N+1`) com `opId`
   - `Put/Update` do registro ou rascunho com `version = N+1`
3. Se a condição falhar (`TransactionCanceledException`), reler e tentar de novo.
4. Opção A: regravar `head.json`. Opção B: publicar no AppSync.

### Operação

```json
{ "opId": "uuid-do-cliente", "casaId": "123", "tipo": "salvarCampo",
  "registroId": "r_77", "campo": "valor", "valor": 150 }
```

O `opId` garante **idempotência**: um reenvio do cliente não é aplicado duas vezes.

### Endpoints

- `POST /api/ops`: aplica uma operação e retorna o `seq`.
- `GET /api/ops?after=N`: ops com `seq > N` (DynamoDB `Query` com `SK > N`).
- `POST /api/presenca`: heartbeat (só na Opção A).
- Snapshot periódico para não reprocessar o log inteiro.

### Front (as duas opções)

- Estado em mapa `id → valor`. Ao receber uma op, atualizar e re-renderizar **só** aquele item (componente memoizado).
- Fila de ops pendentes. Ao receber o próprio `opId` de volta = ack, remove da fila.
- Presença com throttle (máx. 1 evento a cada 2–3s). Status "digitando" expira localmente em ~5s sem novo evento. Status "vendo/criando" expira em ~60–90s sem heartbeat.

## Decisão pendente

- **A (polling + head.json)**: zero serviços novos, atraso de ~2–3s, presença via banco.
- **B (AppSync Events)**: tempo real de verdade (dezenas a centenas de ms), presença leve sem banco, custo de centavos.
- O modelo de ops, `seq` e auto-save é o mesmo nas duas, então dá para começar com A e migrar para B sem refazer o backend.

---

## Plano acordado por fases (vindo do backlog)

A proposta acima foi gerada no chat do Claude. O problema de origem: a tela de uma pessoa não percebe o que a outra gravou, alterou ou ocultou (cópia em memória só renovada por
gravação própria ou a cada 2 min).

**Fase 1: log de operações + sincronização incremental**
- Log no servidor: cada criação, edição, ocultar/mostrar, exclusão e mudança de configuração vira uma operação com
  número de sequência (`seq`) por espaço, gravada na mesma transação (`TransactWriteItems`: contador no item `META`,
  item `OP#<seq>` e o registro). Validade de ~180 dias (TTL).
- `GET /spaces/{sid}/ops?after=N`: devolve só o que mudou desde N (inclui exclusões e ocultações) e o `seq` atual. Sem
  `after`, devolve só o `seq`. Leitura liberada para owner e member (entra na matriz de permissões).
- A tela aplica as operações recebidas em vez de recarregar os 1000 lançamentos.
- Em cada lançamento: quem criou/alterou/ocultou e quando (`atualizadoPor/Em`, `ocultoPor/Em`); lista de membros do
  espaço (item espelho `SPACE#id / MEMBER#sub`) para converter `sub` em nome.
- "Novo desde a sua última visita" (etiquetas automáticas novo/alterado/oculto + contagem) e "Atividade recente",
  ambos a partir do log, sem nada para o usuário marcar ou revisar.
- Idempotência e edição concorrente: id do lançamento gerado na tela (criar duas vezes devolve o mesmo registro),
  travar o botão Salvar, e 409 "alterado por outra pessoa" se o registro mudou desde que foi aberto.
- Cuidados: a importação em lote não deve gerar uma operação por item (emitir uma operação-resumo); a transação
  consome ~2× de escrita (capacidade atual 5 por segundo); registros antigos não têm `atualizadoEm`.

**Fase 2: presença** ("esposa está criando um registro"; no registro aberto, "fulano está aqui"; rascunho com
auto-save visível no Histórico). Presença é efêmera, separada dos dados e expira sozinha (~60 a 90 s sem sinal).
- Caminho A (sem serviço novo): polling de um `head.json` no CloudFront com cache de 1 s (atraso de 2 a 3 s).
  Caminho B (depois): AWS AppSync Events (WebSocket gerenciado, centenas de ms, centavos por mês). O modelo de
  operações é o mesmo, então dá para começar em A e migrar.
- Cuidados: o `head.json` público deve ter só o `seq` (presença e nomes vêm da API autenticada); a presença não pode
  incrementar o `seq` dos dados; no AppSync os canais precisam de autorização por espaço, o exemplo usa `aws-amplify`
  (exige bundler; o front não tem build) e é preciso confirmar a disponibilidade em `sa-east-1`.
