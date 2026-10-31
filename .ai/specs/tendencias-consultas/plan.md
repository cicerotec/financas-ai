# Plan: consultas salvas da Tendências

Como construir o que o [`intent.md`](intent.md) pede, a partir do código de hoje. O roteiro verificável é o [`spec.md`](spec.md).

**Estado:** rascunho para aprovação (2026-10-31). Decisões A a D do intent já fechadas; o ponto E abaixo é novo.

## Onde o código está hoje

| Fato | Onde | Consequência |
|---|---|---|
| A consulta é o estado `ten = {modo, grupos, fichas, pos}` e a `visao`, só em memória | `web/js/tendencias.js:12`, `:22` | Salvar é serializar `modo`, `grupos` **ou** `expressao` (as `fichas` do front), e `visao`; `pos` (o cursor) não entra |
| A lógica pura (limites, `gruposParaFichas`, `consulta`) é testada | `web/consulta-tags.js`, `consulta-tags.test.js` | Serializar e restaurar viram funções puras ali, com teste |
| Os combos moram num documento só (`CFG#COMBOS`), gravado inteiro e só pelo owner | `auth/authz.go` (`cfgNomes`), `app.js:1360` | **Não serve:** uma lista num item só deixaria qualquer gravação substituir a de todos, e o servidor não saberia o autor de cada uma |
| Já existe o padrão "um item por registro" com listagem por prefixo | `repository/saldos.go` (`SALDO#<banco>`) | `CONSULTA#<id>` segue o mesmo molde |
| Já existe `SoDono` (`criadoPor == sub`) para o member, e o owner liberado | `auth/authz.go` (`Permitido`) | Reaproveitado nas ações de editar e excluir |
| O nome de quem agiu é o apelido do vínculo `USER#<sub>`, resolvido pelo servidor | `repository/pessoas.go`, `service/historico.go` | O item guarda só o `sub`; o apelido atual é resolvido na leitura |
| Excluído logicamente some da lista e continua na tabela | `repository/lancamento.go:277` | Mesmo padrão: `excluidoEm` |
| O app não calcula permissão no front: o servidor decide | `guardrails.md` ("o front não decide") | `GET /consultas` devolve `minha` e `podeAlterar` já prontos |
| A subida lê só `listas`, `aparencia`, `tags` e `combos` | `web/carga.test.js` | `GET /consultas` só quando a janela abre |
| Janela nova entra no botão voltar | `camadasAbertas()` e `fecharCamadaTopo()` em `app.js` | As três janelas novas (lista, salvar, renomear) entram nas duas funções |

## Desenho

### Dado
Item novo no espaço: `PK = SPACE#<sid>`, `SK = CONSULTA#<id>` (id de 20 caracteres, gerado no servidor). Campos e regras em
[`spec.md`](spec.md) (R1 a R6). Listar é uma `Query` por `begins_with(SK, "CONSULTA#")`. Sem índice novo, sem `HIST#`, sem mexer no
`SEQ`.

### Backend, nas camadas
- `domain/consulta.go`: o tipo, os limites (4 grupos, 8 tags, 60 fichas, 40 do nome, 50 vivas) e `Validar` (só o **formato**, ponto A).
- `repository/consultas.go`: `Listar(sid)`, `Obter(sid, id)`, `Criar`, `Atualizar` (condicional por `version`) e `MarcarExcluida` /
  `Restaurar` (condicionais). Interface em `service/`, implementação aqui; `service` nunca importa a AWS.
- `service/consultas.go`: as regras (autoria, nome repetido, limite, versão, apelido do autor). Recebe um relógio e o resolvedor de
  pessoas por construtor, como o resto.
- Rotas em `roteador.go` e três ações novas em `auth/authz.go` com teste em `authz_test.go`. Nenhuma passa por `/cfg/{nome}`.

### Front
- `web/consulta-tags.js`: `paraSalvar(estado, visao)` (monta o corpo) e `deSalvo(item)` (devolve o estado do front, `{modo, grupos, fichas, pos, visao}`; o campo salvo `expressao` vira `ten.fichas` aqui e só aqui),
  puras e testadas. Os limites já existem ali; o servidor os repete porque o front não é fronteira de segurança.
- `web/api.js`: `db.consultas.listar/criar/atualizar/excluir/restaurar`, no mesmo estilo de `db.historico`.
- `web/js/tendencias.js`: o botão "Salvar consulta" (desabilitado com expressão inválida ou vazia), o botão "Consultas salvas" e as
  três janelas, **como na demo aprovada**. As ações vêm do servidor (`minha`, `podeAlterar`): o front só mostra ou esconde.
- `irParaAba` não muda: a lista é pedida ao abrir a janela, e a cada gravação o front relê **só** `consultas`.

### Fases
Uma branch e um PR por fase (um assunto por branch):
1. **Fase A, backend** (`feature/consultas-backend`): domain, repository, service, rotas, `authz.go`, testes, modelo de dados.
2. **Fase B, front** (`feature/consultas-tela`): `consulta-tags.js`, `api.js`, `tendencias.js`, `voltar`, testes, demo e
   `padrao-de-telas.md` atualizados para "implementada".
A fase B depende da A estar no `dev`. Entre as duas o app não muda de comportamento.

## Decisões a confirmar com o dono

1. **Ponto E: sem histórico de versões da consulta.** Editar sobrescreve o item (controle por `version`), e só a exclusão é lógica.
   É uma exceção ao "nada se sobrescreve sem versão" (item 31), que trata de lançamentos: a consulta é uma configuração sem efeito
   no saldo, como `CFG#TAGS`. Recomendado: sim, sem histórico.
2. **Nome repetido recusa com `400`** (e não `409`), para o `409` continuar sendo só "a versão mudou, releia". Recomendado.
3. **Restaurar pede `consulta.editar`** (a mesma ação de editar: o member só restaura a própria, o dono qualquer uma). Recomendado.
4. **O dono pode atualizar a consulta de outra pessoa**; o `criadoPor` não muda e `atualizadoPor` registra quem mexeu. Recomendado
   (é o que a demo aprovada mostra: "Atualizar ... (como dono)").

## Riscos

- **Forma da consulta mudar no item 42:** a extração do filtro (42b) pode alterar `grupos` e `expressao`. Itens já salvos teriam de
  ser lidos pela forma antiga. Mitigação: o servidor valida o formato em um só lugar (`domain.Validar`) e o `deSalvo` do front é
  o único ponto que interpreta o item; uma mudança de forma é uma migração planejada, e esta entrega não a antecipa.
- **Tags renomeadas ou excluídas** continuam nas consultas salvas; o front as limpa em silêncio ao carregar (como hoje). Fica
  como está; avisar é uma tela nova.
- **Corrida do limite e do nome** (ponto D): aceita.
- **Acúmulo de excluídas:** sem TTL. Aceito pelo volume; se um dia pesar, é um `Query` + `DeleteItem` administrativo.
- **Permissão só no front** seria o erro clássico: por isso os testes de `authz_test.go` e do serviço provam o `403` sem passar
  pela tela, e o `criadoPor` nunca é lido do corpo.
- **Permissão de infra:** conferir em `template.yaml` se a policy da Lambda já cobre `SK` com o prefixo novo. O `Deny` de
  `UpdateItem` e `DeleteItem` vale só para `HIST#*`, então não bloqueia; se houver lista de prefixos, atualizar as roles do Actions
  (`infra/github-oidc.yaml`) **antes** do merge.
