# Spec: consultas salvas da Tendências (item 53)

Roteiro **verificável** do [`intent.md`](intent.md) e do [`plan.md`](plan.md). Regras com número (`R`), testes com id (`C-T<n>`), para o PR
dizer exatamente o que cobriu. Quando este texto e o do intent divergirem, vale o plan.

**Estado:** rascunho para aprovação (2026-10-31). Duas fases: A (backend) e B (tela). A tela segue a demo aprovada em 2026-10-30.

## Regras

### Dado
- **R1.** Uma consulta é o item `PK = SPACE#<sid>`, `SK = CONSULTA#<id>`. O `<id>` tem 20 caracteres (minúsculas e dígitos) e é gerado no
  servidor. O `sid` vem do token, nunca do corpo.
- **R2.** Campos do item:

  | Campo | Quem grava | Regra |
  |---|---|---|
  | `id` | servidor | igual ao sufixo da `SK` |
  | `nome` | cliente | 1 a 40 caracteres, sem espaço nas pontas; único entre as **vivas** do espaço, sem diferenciar maiúsculas |
  | `modo` | cliente | `grupos` ou `avancado` |
  | `grupos` | cliente | só com `modo = grupos`: `[{ tem: [texto], nao: [texto] }]` |
  | `expressao` | cliente | só com `modo = avancado`: a expressão como lista de fichas, `[{k:'tag',v}`, `{k:'op',v:'E'\|'OU'\|'NAO'}`, `{k:'('}`, `{k:')'}]` |
  | `visao` | cliente | `gastos`, `receitas` ou `saldo` |
  | `criadoPor` | servidor | o `sub` do token; imutável |
  | `criadoEm` | servidor | ISO em UTC, com milissegundos |
  | `atualizadoEm`, `atualizadoPor` | servidor | quando e quem alterou por último (inclui o dono que mexe na de outra pessoa) |
  | `excluidoEm` | servidor | só quando excluída; removido ao restaurar |
  | `version` | servidor | controle otimista, como em `CFG#*` |

- **R2a.** No item o campo se chama **`expressao`** (o título "Expressão" do modo avançado); dentro do front o mesmo dado continua sendo `ten.fichas`, e só `paraSalvar` e `deSalvo` fazem a conversão. Na tela, "fichas" segue como está ("N de 60 fichas").
- **R3.** **Só o formato é validado** (decisão A). O servidor recusa (`400`) campo desconhecido, `modo` fora da lista, `grupos` junto de
  `expressao` (ou nenhum dos dois quando o modo o exige), `k` ou `v` fora dos valores aceitos, tag vazia ou com mais de 60 caracteres,
  e os limites: **4 grupos, 8 tags em cada lista de um grupo, 60 fichas**. Uma consulta vazia (sem tag) é recusada. **Não** valida a
  gramática da expressão nem se a tag existe.
- **R4.** **Nenhum número entra no item.** O corpo não aceita total, contagem nem meses; qualquer campo fora de R2 é `400`.
- **R5.** O `criadoPor` vem **sempre** do token; um `criadoPor` no corpo é `400` (campo desconhecido). O apelido do autor **não é guardado**:
  o servidor o resolve na leitura pelo vínculo `USER#<sub>` (vazio se a pessoa saiu do espaço, mostrado como "alguém do espaço").
- **R6.** Cada espaço tem no máximo **50** consultas **vivas**; as excluídas não contam. Passar disso é `400` ("limite de 50").

### Permissões
- **R7.** Três ações novas em `auth/authz.go`, tudo o que não está liberado é negado:

  | Ação | Owner | Member |
  |---|---|---|
  | `consulta.ler`, `consulta.criar` | Liberado | Liberado |
  | `consulta.editar` (inclui renomear, atualizar, excluir e restaurar) | Liberado | `SoDono` |

  `SoDono` = `criadoPor == sub`. Quem não é dono nem autor recebe `403`, em qualquer rota de escrita.
- **R8.** O **dono (papel owner) altera qualquer consulta** do seu espaço; nunca as de outro espaço (R16).

### Rotas

```
GET    /spaces/{sid}/consultas                  -> 200 [ { id, nome, modo, grupos|expressao, visao, autor, criadoEm, atualizadoEm, version, minha, podeAlterar } ]
POST   /spaces/{sid}/consultas                  { nome, modo, grupos|expressao, visao }            -> 201 { ...item }   | 400
PUT    /spaces/{sid}/consultas/{id}             { version, nome?, (modo, grupos|expressao, visao)? } -> 200 { ...item }   | 400 | 403 | 404 | 409
DELETE /spaces/{sid}/consultas/{id}                                                              -> 200 { id }        | 403 | 404
POST   /spaces/{sid}/consultas/{id}/restaurar                                                    -> 200 { ...item }   | 400 | 403 | 404
```

- **R9.** `GET` devolve só as **vivas**, da mais nova para a mais antiga (`criadoEm`). `minha` (`criadoPor == sub`) e `podeAlterar`
  (R7 e R8) vêm **calculados pelo servidor**; o front só os exibe. O `sub` não é devolvido.
- **R10.** `POST` cria com `version = 1`. Nome repetido entre as vivas (R2) e passar do limite (R6) são `400`, com mensagem em português.
- **R11.** `PUT` aceita renomear (`nome`), trocar a consulta (`modo` + `grupos` ou `expressao` + `visao`, **em bloco**) ou os dois.
  Grava com condição `version = <enviada>`; se mudou, `409` ("alguém alterou antes, abra de novo"). `criadoPor` e `criadoEm` não mudam; `atualizadoEm` e
  `atualizadoPor` sim, e `version` sobe. Renomear para o nome de **outra** consulta viva é `400`; manter o próprio nome é permitido.
- **R12.** `DELETE` marca `excluidoEm` (decisão B), sem apagar o item. Excluir uma consulta já excluída é `200` (idempotente). O item
  excluído some de `GET`, não conta no limite e libera o nome.
- **R13.** `restaurar` remove `excluidoEm` do **mesmo** item, com o mesmo autor. Recusa (`400`) se o nome agora pertence a outra consulta viva
  ou se o limite de 50 foi atingido entre a exclusão e o Desfazer; restaurar uma que não está excluída é `200`.
- **R14.** Uma consulta de outro espaço, ou que não existe, é `404` (nunca `403`, para não revelar que existe).

### Leitura, isolamento e desempenho
- **R15.** **Só lê quando a janela "Consultas salvas" abre** (decisão C). A subida do app e a abertura da Tendências **não** chamam
  `/consultas`. Depois de cada gravação o front relê só `consultas`, nunca "tudo".
- **R16.** Isolamento: o `PK` é sempre `SPACE#<sid do token>`; nenhuma rota lê, altera ou lista consulta de outro espaço.
- **R17.** Custo por chamada: `GET` = 1 `Query`; `POST` = 1 `Query` (contagem e nomes) + 1 `PutItem`; `PUT` = 1 `GetItem` + (se renomear) 1 `Query` +
  1 `UpdateItem` condicional; `DELETE` e `restaurar` = 1 `GetItem` + 1 `UpdateItem` condicional (+ 1 `Query` na restauração). Nenhuma varredura de tabela.
- **R18.** A criação **não** mexe no `SEQ` do espaço (ele só conta alterações de lançamento) nem grava em `HIST#`.
- **R19.** Corrida (decisão D): dois `POST` simultâneos podem passar do limite em 1 ou repetir um nome; **aceito**, sem efeito em saldo ou permissão.
- **R20.** Cada rota deixa uma linha do log de requisição (item 50) com a ação e o status, sem nome, id nem conteúdo da consulta.

### Tela (fase B, como na demo aprovada)
- **R21.** Janela **Consultas salvas** (botão na Tendências): lista com nome, frase, visão, "por <autor>" (ou "por você") e data (`dd/mm/aaaa`), busca por nome e
  frase, fichas **Todas | Minhas** com contagem, e o contador "N de 50 consultas" com aviso no limite.
- **R22.** **Usar** (todos): carrega `modo`, `grupos` ou `expressao` e `visao`; a janela fecha. A consulta carregada mostra "Carregada de X (sua ou de <autor>)" e, se foi mexida,
  "alterada, ainda não salva".
- **R23.** **Renomear** e **Excluir** só aparecem com `podeAlterar`. Sem permissão, o texto "Só <autor> ou o dono podem alterar." no lugar dos botões. Nas do outro, o
  dono vê "como dono".
- **R24.** **Salvar consulta** (desabilitado com expressão inválida ou vazia): pede o nome (contador "N de 40"). Se a carregada é alterável e foi alterada, oferece
  **Atualizar "nome"** (`PUT`) ou **Salvar como nova** (`POST`); se é de outro e a pessoa não é dona, só **Salvar como nova**, com o aviso de que o original não muda.
- **R25.** **Excluir** chama `DELETE` na hora e mostra o aviso de 6 s com **Desfazer** (`restaurar`). Fora dos 6 s, não há Desfazer. Se a consulta excluída era a carregada, a consulta
  na tela fica e só deixa de ser "carregada de".
- **R26.** Erros do servidor (`400`, `403`, `409`, rede) aparecem **na própria janela**, em português, sem fechar e sem perder o nome digitado; `409` oferece reler a lista.
- **R27.** As janelas novas entram em `camadasAbertas()` e `fecharCamadaTopo()`: o voltar do navegador fecha a de cima. Esc fecha a de cima.
- **R28.** **A Tendências continua simulação:** a faixa "Simulação" não muda e nada que o resultado calcula é enviado ao servidor. Cores só por tokens, nos dois temas.

## Testes

Cada um falha antes da mudança. `C-T1` a `C-T20` são de Go (fase A); `C-T21` a `C-T26`, de front (fase B).

| Id | O que prova |
|---|---|
| C-T1 | `authz_test.go`: `consulta.ler` e `consulta.criar` liberados a owner e member; papel desconhecido e rota desconhecida negados (R7) |
| C-T2 | `authz_test.go`: `consulta.editar` é Liberado ao owner e `SoDono` ao member (R7, R8) |
| C-T3 | Member edita, exclui e restaura a **própria** consulta; recebe `403` ao tentar o mesmo na de outra pessoa (R7) |
| C-T4 | Owner edita, exclui e restaura a consulta de um member, e o `criadoPor` não muda (R8, R11) |
| C-T5 | `criadoPor` vem do token: um `criadoPor` no corpo é `400` e nunca é gravado (R5) |
| C-T6 | Campo desconhecido, `modo` inválido, `grupos` com `expressao`, ficha com `k` ou `v` inválido e consulta vazia são `400` (R3, R4) |
| C-T7 | Os limites: 5 grupos, 9 tags numa lista, 61 fichas, nome de 41 caracteres e nome vazio são `400`; 4, 8, 60 e 40 passam (R3, R2) |
| C-T8 | Uma expressão **malformada** mas no formato certo, como `( E`, é aceita (decisão A, R3) |
| C-T9 | Nome repetido entre as vivas é `400`, sem diferenciar maiúsculas; o nome de uma excluída pode ser reaproveitado (R2, R10, R12) |
| C-T10 | A 51ª consulta viva é `400`; excluir uma libera a vaga (R6, R12) |
| C-T11 | `GET` não traz excluídas, vem da mais nova para a mais antiga e traz `minha`, `podeAlterar` e `autor` (apelido), sem `sub` (R5, R9) |
| C-T12 | `PUT` com `version` antiga é `409` e não altera nada; com a atual sobe a `version` e grava `atualizadoPor` (R11) |
| C-T13 | `PUT` renomeia mantendo o próprio nome sem erro e recusa o nome de outra viva (R11) |
| C-T14 | `DELETE` é idempotente e o item continua na tabela com `excluidoEm` (R12) |
| C-T15 | `restaurar` devolve o mesmo item com o mesmo autor; recusa se o nome foi tomado ou o limite foi atingido (R13) |
| C-T16 | Consulta de outro espaço ou inexistente é `404` em `PUT`, `DELETE` e `restaurar` (R14, R16) |
| C-T17 | Duas pessoas em espaços diferentes não veem nem alteram as consultas uma da outra (R16) |
| C-T18 | `POST` não altera o `SEQ` nem cria item em `HIST#` (R18) |
| C-T19 | O número de leituras e gravações de cada rota é o de R17 (contado no repositório falso) (R17) |
| C-T20 | A linha de log de cada rota traz a ação e o status e **não** traz nome, id nem conteúdo da consulta (R20) |
| C-T21 | `consulta-tags.test.js`: `paraSalvar` e `deSalvo` fazem ida e volta sem perda, nos dois modos e nas três visões (R22) |
| C-T22 | `consulta-tags.test.js`: `paraSalvar` não leva `pos` nem campo derivado (R4) |
| C-T23 | `carga.test.js`: a subida do app e a abertura da Tendências não chamam `/consultas`; só a janela chama (R15) |
| C-T24 | Os botões Renomear e Excluir só existem quando o servidor manda `podeAlterar` (R23) |
| C-T25 | As três janelas estão em `camadasAbertas()` e `fecharCamadaTopo()` (R27) |
| C-T26 | O pior caso cabe sem estourar a coluna: nome de 40 caracteres, 60 fichas, 50 consultas, autor comprido, em ≈ 390 px e ≥ 1280 px, nos dois temas (medido por script, com captura) (R21, R28) |

**Quebrar de propósito (antes de commitar):**
- Aceitar `criadoPor` do corpo tem de fazer C-T5 falhar.
- Trocar `SoDono` por `Liberado` em `consulta.editar` tem de fazer C-T2 e C-T3 falharem.
- Fazer a subida chamar `/consultas` tem de fazer C-T23 falhar.

## Verificação no dev

1. Deploy no `financas-dev` e conferir o changeset e o status final da stack. Dados só fictícios.
2. Com duas pessoas (um dono e um membro): o membro salva uma consulta, o dono a vê e a usa, e o dono a atualiza como dono.
3. O membro tenta renomear e excluir a do dono: os botões não aparecem. Chamada direta com o token do membro deve dar `403`.
4. Excluir e Desfazer dentro dos 6 s: a consulta volta com o mesmo autor.
5. No Logs Insights (`/aws/lambda/financas-api-dev`): `filter @message like /^req /` e `acao like /consulta/`: só ações, sem id nem nome.
6. Recarregar a página sem abrir a janela: nenhuma chamada a `/consultas` no log.

## Fora do escopo

Histórico de versões da consulta, aviso de tag inexistente, usar em Histórico e Futuros (item 42), favoritas, pastas, contagem de uso,
TTL das excluídas e qualquer número do resultado no item (ver Limites do `intent.md`).
