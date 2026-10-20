# Uso sem internet e sincronização (ideia para o futuro)

Registro de uma conversa de 2026-10-20. **Nada aqui foi decidido ou implementado.** É memória do que foi discutido, para
retomar depois. O documento continua aberto: novas perguntas e conclusões entram aqui.

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

## Em aberto
- Forma do orquestrador local (item 7.2).
- Critério de "possível duplicado" (campos e tolerância).
- Se um item rejeitado para a fila ou ela segue.
- O que a pessoa vê quando um evento precisa de decisão e ela já saiu do app.
- Prazo/ordem em relação à migração de saldo e totais para o servidor.
