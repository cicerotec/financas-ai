# Spec: recorrência automática de lançamentos (item 51)

Roteiro verificável do [`intent.md`](intent.md) e do [`plan.md`](plan.md).

**Estado:** aprovado em 2026-10-30. **Fase A (backend) implementada** (`service/recorrencias.go`, `domain/recorrencia.go`, testes em `recorrencia_test.go`); **Fase B (telas) implementada** sobre as demos `demo-historico-v2-recorrencia`, `demo-lancar-v2-recorrencia` e `demo-futuros-v2-recorrencia` (`web/js/app.js`, `web/api.js`, `web/recorrencia.js`): sem lista de recorrentes; "Parar" só aparece na cópia mais recente (a que o `GET /recorrencias` traz como `ultimoId`).

## Regras

- **R1.** `recorrente` no lançamento (`TX#`) é um objeto `{ intervalo }`, como `serie` e `aviso`. A recorrência em si é o item auxiliar `AVISOS#ATIVOS` / `<sid>#recorrencia#<id do primeiro registro>` com `tipo: "recorrencia"`, `id` (`recorrencia#<id do primeiro registro>`, como a fatura usa `fatura#...`), `ultimoId` (a ponta: a cópia futura mais recente), `dataUltimo` (a data dela, dia ISO), `intervalo` (inteiro de 1 a 120, em meses) e `dia` (1 a 31, o dia do mês original). Não há campo `proxima` nem janela de dias: a próxima cópia é `dataUltimo` mais `intervalo` meses.
- **R2.** Marcar (`POST /spaces/{sid}/recorrencias`) exige `tx.editar` (ou `tx.criar` na criação) e um `intervalo` válido. Recusa (`400`) intervalo inválido, lançamento inexistente ou excluído, e **transferência**. Cria o item, põe `recorrente: { intervalo }` no lançamento e **cria na hora a primeira cópia** (R3), tudo na mesma operação. Marcar um lançamento que já existe é uma alteração (grava versão); marcar na criação não grava. A cópia criada nunca grava versão.
- **R3.** **A cópia nasce na hora de marcar:** a primeira cópia fica na data do lançamento marcado mais `intervalo` meses (no dia `dia`, ou no último do mês) e aparece em Futuros na hora. Se essa data não é futura (lançamento marcado antigo), avança de `intervalo` em `intervalo` até ser **depois de hoje**: nunca nasce cópia de data passada. Semestral marcado em janeiro cria a de julho agora.
- **R4.** A rotina diária (evento `{"acao":"avisos"}`, 08:00 em São Paulo) cria a **seguinte** quando a ponta venceu: `dataUltimo ≤ hoje`. A nova fica em `dataUltimo` mais `intervalo` meses (R9 se a rotina esteve parada). Não existe janela de antecedência: há sempre uma cópia futura à vista, a um intervalo de distância. Dia em que nenhuma ponta venceu faz **zero leituras além da Query de `AVISOS#ATIVOS`** (R16).
- **R5.** A cópia leva `status`, `tipo`, `valor`, `descricao`, `nota`, `banco`, `tags` e `aviso` do **último registro** (a ponta, mesmo excluído), com a hora dele, e `recorrente` (o mesmo objeto); **não leva `serie`**. `criadoPor` é o do último registro; `criadoEm`, o da criação. Entra pelo caminho normal de criação do service (o aviso da cópia é sincronizado) e **não grava versão** no `HIST#`. O valor segue a ponta **no dia em que ela vence**: ajustar o valor da ponta antes disso vale para a seguinte; depois, só para a outra.
- **R6.** O `id` da cópia é derivado de `(id da recorrência, data da cópia)` e a gravação é condicional (`attribute_not_exists`): rodar duas vezes, ou depois de uma falha no meio, não duplica nem recria o que já existiu, nem o que foi apagado.
- **R7.** Depois de criar (ou constatar que a cópia já existe), avança o item: `ultimoId` e `dataUltimo` passam a ser os da cópia. Vale para a criação na hora de marcar e para a rotina.
- **R8.** **Pular o ciclo:** apagar a cópia (exclusão lógica) não altera o item. Quando a data dela chega, a rotina cria a seguinte normalmente, com os dados da ponta excluída. Excluir um lançamento não apaga o item da recorrência (outra `SK`).
- **R9.** **Sempre para frente:** se `dataUltimo` ficou no passado (rotina parada), a rotina cria **uma** cópia, na primeira data `dataUltimo + n × intervalo` depois de hoje; nunca cria cópia de data passada e não recupera os ciclos perdidos. Uma rodada cria no máximo **uma** cópia por recorrência e **50 no total**; o resto fica para o dia seguinte.
- **R10.** **Desligar** (`DELETE /spaces/{sid}/recorrencias/{id}`): apaga o item e tira o `recorrente` do registro vivo (alteração do dono, com versão). As cópias futuras já criadas, inclusive a que está à vista, ficam como estão. Desligar o que já está desligado é `200` sem efeito.
- **R11.** **Religar** é marcar de novo (R2): cria na hora a próxima cópia, sempre depois de hoje (R3). Os ciclos parados não são recuperados.
- **R12.** Se o último registro **não existe** (apagado direto no banco), o item é removido e uma linha vai ao log; a recorrência para. Excluído pelo app não é esse caso.
- **R13.** `GET /spaces/{sid}/recorrencias` devolve as ativas (`id`, `ultimoId`, `intervalo`, `dataUltimo`) do espaço do token; `GET /tx` e `GET /futuros` devolvem `recorrente` em cada lançamento.
- **R14.** O passado não é tocado: nenhum lançamento existente é alterado pela entrega; só ganha `recorrente` o que o dono marca.
- **R15.** Isolamento: cada item leva o `sid` e a rotina só cria no espaço dele; um espaço nunca lê nem cria no de outro.
- **R16.** Desempenho: criar uma cópia (ao marcar ou pela rotina) = 1 leitura do registro de origem, 1 gravação do lançamento, 1 do aviso da cópia (se tiver) e 1 do item. Nenhuma varredura de tabela em lugar nenhum.
- **R17.** Ao marcar, `aviso` é opcional e vale só para a **primeira cópia** (validado como o de qualquer lançamento; `null` = sem aviso, mesmo que o original tenha; ausente = herda o do original, como em R5). O lançamento original não muda. As cópias seguintes herdam o aviso da ponta (R5). Recusa (`400`) com aviso inválido.

## API

```
POST   /spaces/{sid}/recorrencias?de=<dataEvento atual>  { "id": "<lançamento>", "intervalo": 1, "aviso"?: {...} | null }  -> 201 { id, intervalo, dataUltimo }   | 400 | 403   (cria também a primeira cópia)
DELETE /spaces/{sid}/recorrencias/{id}                                             -> 200
GET    /spaces/{sid}/recorrencias                                                  -> 200 [{ id, ultimoId, intervalo, dataUltimo }]
POST   /spaces/{sid}/tx                  { ..., "recorrente": { "intervalo": 2 } }   -> cria o lançamento, o item e a primeira cópia juntos
```

## Testes

| # | Cenário | Esperado |
|---|---|---|
| RC-T1 | Marcar com intervalo 1, 2, 6, 12; inválido (0, 121, texto); transferência; lançamento inexistente ou excluído | `201`; `400`; `400`; `400` |
| RC-T2 | Marcar um lançamento existente; marcar na criação | item criado e `recorrente: { intervalo }`; versão só no primeiro caso |
| RC-T3 | Marcar mensal, semestral, anual; marcar lançamento antigo | a cópia nasce na hora, em +1, +6, +12 meses; no antigo, na primeira data depois de hoje |
| RC-T3b | Rotina com `dataUltimo` amanhã; hoje; ontem (rotina parada) | não cria; cria a seguinte; cria uma só, depois de hoje |
| RC-T4 | A cópia criada | campos de R5, sem `serie`, `recorrente` herdado, sem versão no `HIST#`, aviso da cópia sincronizado |
| RC-T5 | Dia 31 mensal; 29/02 anual | 28/02, 31/03, 30/04; 28/02/2029 e volta a 29/02 em 2032 |
| RC-T6 | Intervalos 2, 6 e 12 | datas a cada 2, 6 e 12 meses, no mesmo dia (ou o último do mês) |
| RC-T7 | Rodar a rotina duas vezes no mesmo dia | uma cópia só; o item avançou uma vez |
| RC-T8 | Falha depois de criar a cópia e antes de avançar o item; nova rodada | sem duplicar; o item só avança |
| RC-T9 | Apagar a cópia à vista e rodar na data dela | a rotina cria a seguinte normalmente; nada é recriado no lugar da apagada |
| RC-T10 | Excluir pelo app o último registro; rodar no vencimento seguinte | o item permanece; a cópia nova sai com os dados do excluído |
| RC-T11 | Desligar e rodar | nada é criado; as cópias futuras já criadas ficam |
| RC-T12 | Religar depois de meses parada | cria na hora a próxima depois de hoje, sem cópia de data passada |
| RC-T13 | Rotina sem nada a vencer | só a Query de `AVISOS#ATIVOS` (contagem de leituras no fake) |
| RC-T14 | Último registro apagado direto no banco | item removido, linha no log, nada criado |
| RC-T15 | Rodada com mais de 50 a vencer | 50 criadas; as demais na rodada seguinte |
| RC-T16 | Edição do valor do último registro antes da rotina | a cópia seguinte sai com o valor novo |
| RC-T17 | Dois espaços com recorrência | cada cópia vai para o seu espaço (`sid`) |
| RC-T18 | `authz`: owner e member com `tx.editar`; leitor sem | `201`/`403` como em R2 |
| RC-T19 | `GET /recorrencias`; `GET /tx` | só as ativas do espaço; `recorrente` no lançamento |
| RC-T20 | Lançamentos antigos | não ganham `recorrente`; nada é gravado ao ler |

## Telas (fase B, depois da demo aprovada)

- **Reaproveita a tela **Repetir** que já existe (Mensal, Trimestral, Semestral, Anual e Outro intervalo), **sem** a linha "Repetir quantas vezes", porque a recorrência não tem fim** (`.ai/padrao-de-telas.md`): o dono escolhe o intervalo ali e confirma; **não há campo de status**. A demo vai em
  `.ai/demos/` e mostra a tela inteira (celular e computador).
- A linha do lançamento mostra o 🔁 quando `recorrente`; as cópias de recorrência **não** mostram "série N". Há uma lista das recorrências ativas, com o botão de parar.
- O front só mostra: quem calcula a data da cópia e diz quais são ativas é o servidor.
- Testes do front: `web/*.test.js` sem chamada a função indefinida e sem nome de status.

## Fora desta entrega

Recorrência com fim ou com quantidade (é o Repetir); editar ou apagar as cópias em massa; janela de antecedência (a cópia nasce na hora de marcar, sem prazo); valor estimado; transferências recorrentes; recuperar meses perdidos.
