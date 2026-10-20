# Uso sem internet e sincronização (ideia para o futuro)

Registro de uma conversa de 2026-10-20. **Nada aqui foi decidido ou implementado.** É memória do que foi discutido, para
retomar depois. O documento continua aberto: novas perguntas e conclusões entram aqui.

**Quando isto entra em jogo (decisão do dono, 2026-10-20):** é trabalho **de futuro**, para o dia em que houver muitos
usuários querendo usar o app sem internet. Nada daqui é para fazer agora. O objetivo do documento é não ser pego de
calças curtas: ter o desenho pensado e saber quais passos são baratos e quais só valem a pena se o uso justificar.

Relacionados: [`arquitetura.md`](arquitetura.md) (front não decide, "Saldo e totais calculados no servidor"),
[`colaboracao-tempo-real.md`](colaboracao-tempo-real.md), [`hospedagem.md`](hospedagem.md).

## 1. Situação atual (levantada no código)

- Não há service worker, IndexedDB nem tratamento de `navigator.onLine`. O `localStorage` guarda só o token de login e o
  espaço escolhido (`web/api.js`).
- Existe `manifest.webmanifest` (instalável na tela inicial), mas isso não dá suporte offline: sem cache, a página nem abre.
- Cada leitura e escrita vai direto à Lambda; o servidor é a fonte de verdade. O login (Cognito, PKCE) também exige rede.
- Hoje o **navegador calcula** o saldo por banco (`calcularSaldoBanco`), o total, as faturas do cartão e os totais de
  Futuros e Histórico. A decisão de mover isso ao servidor está em `arquitetura.md`, ainda não implementada.

## 2. Armazenamento local (decidido em 2026-10-20)

O dado local fica no **IndexedDB** do navegador. SQLite local foi descartado por custo; o banco do servidor continua o
DynamoDB, sem novo adaptador.

## 3. Níveis de uso offline

| Nível | O que entrega | Esforço |
|---|---|---|
| 1. Abrir sem rede | Service worker com cache de `index.html`, CSS, JS e ícones; aviso de "sem conexão" | Baixo |
| 2. Leitura offline | Última resposta de lançamentos, saldos e futuros no IndexedDB; consulta sem alterar | Médio |
| 3. Escrita offline | Fila local enviada quando a rede volta | Alto |

### Níveis 1 e 2 e o cálculo de Saldos e Cartão

- **Nível 1:** o cálculo não muda; as abas Saldos e Cartão ficam sem número (`erro` ou `···`, total com `⚠`).
- **Nível 2, caminho A (não recomendado):** guardar os lançamentos e recalcular no navegador. Reforça o cálculo no
  cliente, que a decisão de arquitetura quer eliminar, e a data do aparelho passaria a decidir o que é futuro.
- **Nível 2, caminho B (recomendado):** guardar a **resposta pronta do servidor** e exibi-la com carimbo
  ("atualizado em 14:32, sem conexão"). Depende de o servidor já calcular saldo e fatura.
  - Meses fechados e conferidos são estáveis; o mês corrente e a fatura aberta podem estar defasados (carimbo obrigatório).
  - A fatura é independente por ciclo, então o snapshot de um ciclo não envelhece por causa de outro.
  - O cache é por espaço e por pessoa e é apagado no logout.
  - Conferir, fechar mês e pagar fatura continuam exigindo conexão.
- **Conclusão:** o nível 1 pode vir antes de tudo; o nível 2 deve vir depois, ou junto, da migração "Saldo e totais no
  servidor".

## 4. Erros possíveis no nível 3 (escrita offline)

### Duplicidade
| Cenário | Tratamento |
|---|---|
| Duas pessoas lançam a mesma coisa (uma online, outra offline) | Não há prevenção automática confiável (dois gastos iguais no mesmo dia podem ser legítimos). **Detectar e avisar** ("possível duplicado, manter os dois?") |
| Reenvio do mesmo pedido (rede caiu no meio, toque duplo) | **ID gerado no aparelho** (UUID); o servidor trata o segundo envio como o mesmo pedido (idempotência) |
| Importação de planilha feita offline e depois online | ID por linha mais detecção de parecidos |

### Conflito de edição
- A edita um campo e B edita outro do mesmo lançamento offline: o backend já confere a **versão esperada**; o envio de B
  chega com versão antiga e é recusado. A tela mostra as duas versões.
- A exclui e B edita: devolver "excluído por A" e deixar B recriar ou descartar.
- Mesmo campo editado pelos dois: a pessoa escolhe. O histórico imutável guarda a versão rejeitada.

### Ordem e tempo
- Não confiar na hora do aparelho: o servidor carimba o recebimento e guarda a data informada pela pessoa à parte.
- Eventos do mesmo lançamento sobem em ordem (criar, editar, excluir). Decidir se um item rejeitado para a fila ou não.

### Regras do domínio que quebram
- **Mês fechado ou conferido enquanto estava offline:** o servidor devolve `ErroMesConferido`; o item fica "precisa de
  confirmação", que exige uma pessoa.
- **Fatura conferida e travada:** mesmo problema para lançamentos de cartão.
- **Transferência** (dois lançamentos com o mesmo `transferParId`): as duas pontas sobem **juntas**, num único evento.
- **Pagamento de fatura:** não deve ser permitido offline (a fatura pode ter mudado).
- **Banco, tag ou lista apagada ou renomeada** por outra pessoa; **papel alterado** (membro removido); **token expirado**.
- Saldo e totais são do servidor: o lançamento pendente aparece como "aguardando envio" e **não mexe em saldo ou fatura**
  até sincronizar.

### Falhas da fila local
- O navegador pode limpar o armazenamento (pouco espaço, iOS/Safari, limpeza pelo usuário): o que não subiu some. Mostrar
  sempre "N lançamentos aguardando envio".
- Troca de aparelho ou logout perde a fila; dois aparelhos da mesma pessoa têm filas diferentes.
- Falha parcial no envio: exige ID e idempotência para saber o que já subiu.
- Telas mostram dados do último snapshot, sem o que a outra pessoa lançou.

## 5. Como o Google faz o modo offline (referência)

- **Docs e Sheets:** o offline é ativado de forma explícita por arquivo; as edições ficam numa fila local e são
  **mescladas** pelo servidor (Operational Transformation/CRDT, edições tratadas como operações). Na mesma célula do
  Sheets vale a última que chegou, e o histórico de versões guarda a outra.
- **Gmail:** lê o que já foi baixado e escreve na caixa de saída; enviar não conflita com nada.
- **O que dá para copiar:** avisar o estado ("Salvando", "Salvo", "Offline, será sincronizado"), ativar o offline de
  forma explícita, ter histórico de versões e limitar o offline às operações que não conflitam.
- **O que não dá:** a mesclagem automática só funciona onde a ordem das edições não muda o resultado. Dinheiro não é
  assim: lançamento em mês recém-fechado exige decisão de uma pessoa.

## 6. Versão restrita sugerida do nível 3

1. Só **criar lançamento simples** offline (sem editar, excluir, transferir ou pagar fatura).
2. UUID gerado no aparelho em cada criação.
3. Aparece como "pendente", sem efeito em saldo e fatura até sincronizar.
4. Ao sincronizar, o servidor aplica todas as regras e devolve um relatório ("4 enviados, 1 precisa da sua atenção").
5. Aviso de possível duplicado (mesmo valor, data e banco) em vez de bloqueio.
6. Contador visível de itens aguardando envio.

## 7. Proposta do dono: orquestrador local + fila de eventos (em discussão)

Ideia: o front é **só front** (exibe, não decide, não manda regra a ninguém). Um **orquestrador local**, separado do
front, gera os registros com **UUID + chave de geração conhecida** e mantém uma **fila de eventos**; o servidor aplica
todas as regras. É o padrão *outbox* local com comandos/eventos (parecido com CQRS) e está alinhado ao princípio "o
front não decide".

O que resolve: duplicidade por reenvio (idempotência), regras num só lugar, front simples (manda a intenção e recebe o
estado calculado).

Pontos levantados, ainda sem decisão:
1. **A chave não impede o duplicado entre duas pessoas** (cada aparelho gera uma chave diferente). Manter dois
   mecanismos separados: a chave do evento (idempotência de envio) e a detecção de parecidos no servidor (aviso).
   Chave derivada do conteúdo bloquearia gastos legítimos iguais no mesmo dia.
2. **Forma do orquestrador local** (decisão que destrava as demais):
   - service worker + IndexedDB no próprio navegador (mais barato, mesmo código e instalação);
   - app separado (Tauri, Capacitor, nativo): isolamento, mas dois produtos para manter;
   - processo local no computador: não serve ao celular.
   O isolamento é de **responsabilidade**, não de segurança: quem garante a regra é o servidor.
3. **O front manda comandos, não estados.** Hoje monta o lançamento inteiro (com `status` e `faturaAjuste`). No novo
   modelo manda só o que a pessoa digitou e o servidor deriva status, fatura e saldo. Pede rotas de comando novas.
4. **Cada evento volta com resultado:** `aceito`, `rejeitado` (com motivo) ou `precisa de decisão`. A decisão da pessoa
   vira outro comando, nunca regra no cliente.
5. **Ordem:** eventos do mesmo lançamento em sequência; lançamentos diferentes em paralelo; transferência como um único
   evento com as duas pontas.
6. **Pendente não altera saldo nem fatura** até o servidor confirmar (consequência direta de o front não calcular).
7. **Edição e exclusão** levam a **versão que a pessoa viu**; se mudou, o servidor devolve conflito.

Efeito no projeto: é uma mudança de contrato da API (comandos com ID de evento, respostas com resultado por evento),
não um acréscimo. Faz mais sentido junto da migração "Saldo e totais calculados no servidor", para não refazer o front
duas vezes.

## 8. Comunicação: status do comando e canal de mensagens (em discussão)

Conflito no mesmo registro já acontece online e a defesa é a versão esperada (o backend recusa a versão antiga); a fila
não cria o problema nem o piora. O que ela dá é **visibilidade**: o conflito vira um estado do comando na tela.

Ciclo de vida de cada comando, com o ID de correlação em todas as transições:

```
na fila (local) → enviado → recebido pelo servidor → processando
                                      ↓
              aceito | rejeitado (motivo) | precisa de decisão
```

**Sem loop de requisições.** O padrão é publish/subscribe com request-reply assíncrono:
1. O worker envia o comando por HTTP e recebe só `202 Accepted` com o ID.
2. O servidor **publica o resultado** num tópico do espaço (ou da pessoa).
3. O worker está inscrito por uma conexão persistente e recebe o resultado quando ele existe; ninguém fica perguntando.

**Obstáculo na infra atual.** A Lambda não mantém conexão: `colaboracao-tempo-real.md` fixa "sem API Gateway" e
`hospedagem.md` registra o limite de 10 execuções simultâneas na conta (uma conexão SSE na Lambda ocuparia uma execução
durante toda a sessão). A conexão persistente teria de viver fora da Lambda.

| Opção | Observação |
|---|---|
| AWS IoT Core (MQTT sobre WebSocket) | Pub/sub gerenciado, cobrança por mensagem, autenticação via Cognito (Identity Pool), sem API Gateway. Inclinação inicial |
| AppSync Events | Parecido; mais um serviço para aprender |
| Servidor próprio de WebSocket (container) | Custo fixo e mais operação |
| Web Push | Não é sessão aberta; só complemento ("1 lançamento precisa da sua atenção") |

Exigências do padrão:
- **Mensagem perdida:** ao reconectar, o worker faz **uma** consulta pontual "o que mudou desde o cursor X?". O contador
  `_seq` do backend serve de cursor. Não é loop.
- **Idempotência no recebimento:** o resultado pode chegar duas vezes (canal e reconciliação); o worker ignora pelo ID.
- **App fechado:** navegadores matam a conexão e até o service worker. Sem app aberto não há sessão; os resultados ficam
  no servidor e são recuperados pela reconciliação ao abrir.
- **Autorização do tópico:** por espaço; só dono e membro assinam, e o servidor confere o papel antes de emitir a credencial.
- O mesmo canal pode servir à **presença e atualizações ao vivo** de `colaboracao-tempo-real.md`: decidir as duas juntas.

## 9. Core independente da Lambda (em discussão; só se houver demanda)

Princípio do dono: **não ficar refém da Lambda.** O core (`domain` + `service`) deve poder receber comandos de várias
entradas: a Lambda HTTP (online) e consumidores de fila (online e offline).

**O que já existe.** É uma arquitetura hexagonal: o `service` declara as interfaces e não importa AWS, HTTP nem Telegram
(o `arquitetura_test.go` garante); relógio e repositório entram por construtor; o `local.go` já é uma segunda entrada.
O que prende à Lambda é só a borda: o `roteador.go` recebe o evento da Lambda e o `controller` pensa em rota, token e JSON.

**O que faltaria.**
1. **Porta de comandos independente de HTTP:** `Comando{id, tipo, autor, espaço, dados}` que devolve
   `Resultado{status, motivo}`.
2. **Entradas como adaptadores finos** que só traduzem e chamam o core: Lambda HTTP (a atual), consumidor de fila, modo local.
3. **Porta de saída para resultados:** interface `Publicador` declarada no core e implementada por um adaptador (IoT,
   WebSocket...), no mesmo molde do `Avisador`. O core não sabe que canal existe.
4. **Empacotamento:** vários executáveis no mesmo módulo Go (`cmd/lambda`, `cmd/worker`, `cmd/local`) ligando o mesmo
   core. Separar em módulos distintos só mais tarde, se compensar.

**Pontos de atenção.**
- **Duas filas diferentes:** a fila **local** no aparelho (outbox, para o offline) e a fila de **ingestão no servidor**.
  Têm papéis e falhas distintos.
- **Várias entradas ao mesmo tempo:** a proteção continua sendo a transação atômica + a checagem de versão esperada.
- **Ordem por espaço:** uma fila FIFO particionada por espaço (SQS FIFO com grupo por espaço, por exemplo) dá ordem e
  deduplicação por ID. É escolha de infraestrutura que o core não deve conhecer.
- **Idempotência no core:** o ID do comando precisa ser lembrado em algum lugar. É um **dado novo no banco**, então cai
  no princípio 3 de `arquitetura.md` (nenhuma chave ou campo novo no DynamoDB sem discutir antes).
- **Processo de longa duração:** se o core puder rodar como container, ele mesmo pode manter o canal aberto, sem IoT
  Core. A troca é custo fixo de servidor ligado contra serviço gerenciado por mensagem.

**Ressalva honesta.** Para duas pessoas, vários executáveis e fila de ingestão não se justificam. A separação em camadas
já existe, então o custo real está só nas portas de comando e de saída. **Não começar** extraindo o core em módulo nem
trocando a infra: começar definindo o contrato `Comando`/`Resultado`, com a Lambda como primeiro adaptador a usá-lo.

**Gatilho sugerido para retomar:** quando houver demanda real de uso offline por muitos usuários. Até lá, o que é barato
e não compromete nada: manter o `service` livre de AWS/HTTP (já garantido por teste), não colocar regra no front e
fazer a migração "Saldo e totais calculados no servidor", que é pré-requisito do nível 2 e do modelo de comandos.

## Em aberto
- Forma do orquestrador local (item 7.2).
- Critério de "possível duplicado" (campos e tolerância).
- Se um item rejeitado para a fila ou ela segue.
- O que a pessoa vê quando um evento precisa de decisão e ela já saiu do app.
- Prazo/ordem em relação à migração de saldo e totais para o servidor.
- Serviço de pub/sub (inclinação: IoT Core) ou core de longa duração com o canal próprio.
- Se o Web Push entra ou basta o aviso ao abrir o app.
- Formato exato de `Comando`/`Resultado` e onde guardar o ID para idempotência (campo novo: discutir antes).
- O que medir para decidir que "há muitos usuários querendo offline" (o gatilho de retomada).
