# Intent: consultas salvas da Tendências (item 53)

Item 53 do BACKLOG. Este documento diz **o quê e por quê**; o como (`plan.md`) e o roteiro verificável (`spec.md`) vêm junto,
para o dono aprovar antes de qualquer código.

**Estado:** rascunho para aprovação (2026-10-31). A demo da janela foi aprovada em 2026-10-30
([`demo-tendencias-consultas.html`](../../demos/demo-tendencias-consultas.html)); as decisões de dados A a D, também em 2026-10-31.

## Por quê

A consulta da Tendências (grupos com E e NÃO, OU entre grupos, ou a expressão livre do modo avançado) fica só na memória do
navegador (`ten` em `web/js/tendencias.js`): ao recarregar a página ela se perde. Quem monta uma consulta trabalhosa, como
"Mercado E NÃO Presente" ou "Casa E (fixo OU mensal)", refaz tudo na visita seguinte, e a outra pessoa do espaço não vê o que
a primeira montou.

## O que queremos

1. **Salvar a consulta com um nome** e **usá-la de novo** com um toque, numa janela "Consultas salvas" aberta pela Tendências.
2. **Compartilhada no espaço:** todos os membros veem e usam todas as consultas.
3. **Cada pessoa altera só as suas** (renomear, atualizar, excluir). **O dono modera:** altera e exclui as de qualquer pessoa.
4. **Quem não é o autor nunca sobrescreve a consulta de outro:** a cópia vira uma consulta nova, sua ("Salvar como nova").
5. **Excluir na hora, com Desfazer** (aviso de 6 s), restaurando a mesma consulta com o mesmo autor.
6. **A Tendências continua sendo simulação:** só a consulta é guardada (tags, operadores e visão), nunca um número. O resultado é
   sempre recalculado com os lançamentos que já estão na tela.

## Decisões fechadas (dono, 2026-10-31)

- **A. O servidor valida o formato**, não a gramática da expressão. O front já desabilita "Salvar" com expressão inválida; uma
  consulta malformada guardada por quem forja a requisição só mostra "Expressão com erro" na tela e não afeta saldo nem dado de
  ninguém. Reescrever em Go o leitor de `consulta-tags.js` duplicaria regra.
- **B. Exclusão lógica** (`excluidoEm`, o mesmo padrão do lançamento). O Desfazer restaura o mesmo item, com o mesmo autor.
- **C. Lê só quando a janela "Consultas salvas" abre**, nunca na subida do app (guardrail "cada tela lê só o que é dela").
- **D. Aceita a corrida** de duas pessoas criando ao mesmo tempo: o limite de 50 pode estourar em 1 e um nome pode repetir. O
  efeito é cosmético e não afeta saldo nem permissão; um contador transacional não compensa.

## Limites (o que fica de fora)

- **Histórico de versões da consulta:** editar sobrescreve (ver o ponto E do `plan.md`); só a exclusão é lógica.
- **Aviso de tag que não existe mais** numa consulta salva: o front continua limpando em silêncio (`limparTagsInexistentes`).
  Avisar a pessoa é uma tela nova e pede demo própria.
- **Salvar a consulta nas outras telas** (Histórico, Futuros): depende do item 42 (filtro único). Esta entrega é só da Tendências.
- Contar usos, data do último uso, favoritas, pastas, compartilhar fora do espaço.
- Salvar **números** do resultado (total, meses): nunca, por decisão e pelo guardrail "o front não é fonte de número".

## Custo

- **DynamoDB:** até 50 itens pequenos (≈ 1 KB) por espaço. `GET /consultas` é uma `Query` por prefixo, só quando a janela abre;
  escrever é uma gravação por ação. Muito abaixo de 1 RCU por abertura numa tabela provisionada de 5 RCU.
- **Armazenamento:** as consultas excluídas ficam na tabela (não há TTL ligado em `template.yaml`); com o teto de 50 vivas e
  uso familiar, o acúmulo é desprezível.

## Pronto quando

- A demo aprovada funciona no app, no celular e no computador, nos dois temas, com o pior caso medido (nome de 40 caracteres,
  consulta de 60 fichas, 50 consultas).
- Um membro não consegue editar, renomear, excluir nem restaurar a consulta de outro (`403`), e o dono consegue; o `criadoPor`
  vem sempre do login, nunca do corpo.
- `go test ./...` passa (inclusive `arquitetura_test.go` e `authz_test.go`) e cada `node web/*.test.js` passa; `carga.test.js`
  prova que a subida do app não lê `/consultas`.
- As chaves novas estão em [`modelo-de-dados/README.md`](../modelo-de-dados/README.md), com exemplo fictício nos dois JSON, e a
  demo e o `padrao-de-telas.md` dizem "implementada".
