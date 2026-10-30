# Spec: recorrência automática de lançamentos (item 51)

Roteiro verificável do [`intent.md`](intent.md) e do [`plan.md`](plan.md).

**Estado:** rascunho para aprovação (2026-10-29). Nada implementado.

## Regras

- **R1.** `recorrente` no lançamento (`TX#`) é um objeto `{ intervalo }`, como `serie` e `aviso`. A recorrência em si é o item auxiliar `AVISOS#ATIVOS` / `<sid>#recorrencia#<id do primeiro registro>` com `tipo: "recorrencia"`, `id`, `ultimoId`,
  `dataUltimo`, `proxima` (dia ISO), `intervalo` (inteiro de 1 a 120, em meses) e `dia` (1 a 31, o dia do mês original).
- **R2.** Marcar (`POST /spaces/{sid}/recorrencias`) exige `tx.editar` (ou `tx.criar` na criação) e um `intervalo` válido. Recusa (`400`) intervalo inválido, lançamento inexistente ou excluído, e
  **transferência**. Cria o item e põe `recorrente: { intervalo }` no lançamento na mesma operação. Marcar um lançamento que já existe é uma alteração (grava versão); marcar na criação não grava.
- **R3.** A primeira `proxima` é a primeira data **depois** da data do lançamento marcado, somando `intervalo` meses a partir dele; `dia` é o dia do mês dele.
- **R4.** A rotina diária (evento `{"acao":"avisos"}`, 08:00 em São Paulo) cria a cópia quando `proxima − hoje ≤ 30 dias` (`JanelaRecorrencia`, constante em `domain/`). Dia sem recorrência a vencer faz
  **zero leituras além da Query de `AVISOS#ATIVOS`** (R9).
- **R5.** A cópia leva `status`, `tipo`, `valor`, `descricao`, `nota`, `banco`, `tags` e `aviso` do **último registro** (a ponta, mesmo excluído), na `proxima`, com a hora dele, e `recorrente` (o mesmo objeto); **não
  leva `serie`**. `criadoPor` é o do último registro; `criadoEm`, o da criação. Entra pelo caminho normal de criação do service (o aviso da cópia é sincronizado) e **não grava versão** no `HIST#`.
- **R6.** O `id` da cópia é derivado de `(id da recorrência, proxima)` e a gravação é condicional (`attribute_not_exists`): rodar duas vezes, ou depois de uma falha no meio, não duplica nem recria o que já
  existiu, nem o que foi apagado.
- **R7.** Depois de criar (ou constatar que a cópia já existe), a rotina avança o item: `ultimoId` e `dataUltimo` da cópia, `proxima` = o mês seguinte por `intervalo` meses, no dia `dia` (ou o último do mês).
- **R8.** **Pular o mês:** apagar a cópia (exclusão lógica) não altera o item, cuja `proxima` já está adiante: a rotina não a recria e segue no próximo vencimento. Excluir um lançamento não apaga o item da
  recorrência (outra `SK`).
- **R9.** **Sempre para frente:** se `proxima` ficou no passado (rotina parada, recorrência religada), a rotina avança `proxima` sem criar nada até ela ser ≥ hoje; nunca cria cópia de data passada. Fora
  disso, uma rodada cria no máximo **uma** cópia por recorrência e **50 no total**; o resto fica para o dia seguinte.
- **R10.** **Desligar** (`DELETE /spaces/{sid}/recorrencias/{id}`): apaga o item e tira o `recorrente` do registro vivo (alteração do dono, com versão). As cópias futuras já criadas ficam como estão.
  Desligar o que já está desligado é `200` sem efeito.
- **R11.** **Religar** é marcar de novo (R2); a `proxima` é a próxima data **depois de hoje** (R3 aplicado à data do registro, avançado até passar de hoje). Os meses parados não são recuperados.
- **R12.** Se o último registro **não existe** (apagado direto no banco), o item é removido e uma linha vai ao log; a recorrência para. Excluído pelo app não é esse caso.
- **R13.** `GET /spaces/{sid}/recorrencias` devolve as ativas (`id`, `ultimoId`, `intervalo`, `proxima`) do espaço do token; `GET /tx` e `GET /futuros` devolvem `recorrente` em cada lançamento.
- **R14.** O passado não é tocado: nenhum lançamento existente é alterado pela entrega; só ganha `recorrente` o que o dono marca.
- **R15.** Isolamento: cada item leva o `sid` e a rotina só cria no espaço dele; um espaço nunca lê nem cria no de outro.
- **R16.** Desempenho: criar uma cópia = 1 leitura do último registro, 1 gravação do lançamento, 1 do aviso da cópia (se tiver) e 1 do item. Nenhuma varredura de tabela em lugar nenhum.

## API

```
POST   /spaces/{sid}/recorrencias        { "id": "<lançamento>", "intervalo": 1 }  -> 201 { id, intervalo, proxima }   | 400 | 403
DELETE /spaces/{sid}/recorrencias/{id}                                             -> 200
GET    /spaces/{sid}/recorrencias                                                  -> 200 [{ id, ultimoId, intervalo, proxima }]
POST   /spaces/{sid}/tx                  { ..., "recorrente": { "intervalo": 2 } }   -> cria o lançamento e o item juntos
```

## Testes

| # | Cenário | Esperado |
|---|---|---|
| RC-T1 | Marcar com intervalo 1, 2, 6, 12; inválido (0, 121, texto); transferência; lançamento inexistente ou excluído | `201`; `400`; `400`; `400` |
| RC-T2 | Marcar um lançamento existente; marcar na criação | item criado e `recorrente: { intervalo }`; versão só no primeiro caso |
| RC-T3 | Rotina com `proxima` a 31 dias; a 30 dias; a 1 dia | não cria; cria; cria |
| RC-T4 | A cópia criada | campos de R5, sem `serie`, `recorrente` herdado, sem versão no `HIST#`, aviso da cópia sincronizado |
| RC-T5 | Dia 31 mensal; 29/02 anual | 28/02, 31/03, 30/04; 28/02/2029 e volta a 29/02 em 2032 |
| RC-T6 | Intervalos 2, 6 e 12 | datas a cada 2, 6 e 12 meses, no mesmo dia (ou o último do mês) |
| RC-T7 | Rodar a rotina duas vezes no mesmo dia | uma cópia só; o item avançou uma vez |
| RC-T8 | Falha depois de criar a cópia e antes de avançar o item; nova rodada | sem duplicar; o item só avança |
| RC-T9 | Apagar a cópia criada e rodar de novo | o mês fica pulado; a `proxima` já era a seguinte; nada é recriado |
| RC-T10 | Excluir pelo app o último registro; rodar no vencimento seguinte | o item permanece; a cópia nova sai com os dados do excluído |
| RC-T11 | Desligar e rodar | nada é criado; as cópias futuras já criadas ficam |
| RC-T12 | Religar depois de meses parada | `proxima` a partir de hoje, sem cópia de data passada |
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
- O front só mostra: quem calcula a próxima data e diz quais são ativas é o servidor.
- Testes do front: `web/*.test.js` sem chamada a função indefinida e sem nome de status.

## Fora desta entrega

Recorrência com fim ou com quantidade (é o Repetir); editar ou apagar as cópias em massa; janela de antecedência por recorrência; valor estimado; transferências recorrentes; recuperar meses perdidos.
