# Guardrails: regras para a IA que programa neste repositório

Leia antes de mexer em qualquer coisa. Estas regras valem para toda tarefa, sem exceção. Se um pedido do dono
contrariar uma regra daqui (segurança, arquitetura ou uma decisão registrada), **aponte o conflito antes de fazer** em
vez de obedecer. Se a especificação não cobre um caso, **pergunte** em vez de decidir.

## Arquitetura e dados
Referência: [`arquitetura.md`](arquitetura.md) (decisões de 2026-10-10).
- **O front não decide.** Toda regra de negócio fica no backend e chega pronta pela API, nunca por etiqueta (nome de
  status ou de banco).
- Camadas `controller` → `service` → `domain`, adaptadores atrás de interfaces e dependências por construtor, sem
  globais.
- **Não crie, renomeie nem remova chave ou campo no DynamoDB sem discutir com o dono.** Cada campo (como
  `faturaAjuste`) carrega uma decisão que não é óbvia no código.
- Nada se sobrescreve sem versão no histórico (item 31 do backlog).

## Telas
Regra de 2026-10-08: **nenhuma alteração de tela vai para o código antes de uma demo aprovada** pelo dono. Ele vê só o
visual e decide; quem propõe diz o que faz sentido, o que não faz e o que a proposta faz perder. Vale para telas novas
e para ajustes em telas que já existem.
- A demo é uma página interativa publicada com link que abre no celular (artefato privado). O widget inline do chat só
  aparece no app de desktop, então não serve como entrega.
- **Padrão aprovado (celular primeiro, com versão para computador):** leia `.ai/padrao-de-telas.md` e abra as
  demos de `.ai/demos/` antes de desenhar ou alterar qualquer tela. Em resumo:
  - **Celular:** uma linha-resumo que abre tela cheia; lista que acumula com check redondo (ou escolha única);
    interruptor no topo; "Outro..." com rodas de número e unidade numa janela de baixo, com resumo ao vivo e erro na
    própria janela; contador de limite visível; um evento por linha, com as ações numa janela de baixo; resumo com dois
    blocos e uma faixa de atraso (só quando há atraso), valores que nunca quebram; linhas de pelo menos 56 px.
  - **Computador:** a mesma tela vira uma tabela de largura total (colunas alinhadas, mínimo de 1120 px a partir de
    1160 px de janela, container de até 1200 px) com as ações na própria linha, apagadas até o mouse passar ou o foco
    entrar. Mesmos dados e mesmas regras; só o desenho muda com a largura da janela.
  - **Sempre:** cores só pelos tokens do app (claro e escuro), datas com ano (`dd/mm/aaaa`), limites à vista, e a demo
    diz o que a proposta faz perder.
- **Mantenha o padrão em dia (obrigatório):** ao implementar ou testar uma tela e descobrir um ajuste (largura de
  coluna, rótulo, limite, comportamento, token), corrija **na mesma entrega** a demo em `.ai/demos/`, o guia
  `.ai/padrao-de-telas.md` (inclusive uma linha em "Armadilhas já encontradas") e este resumo, e diga ao dono o
  que mudou no padrão. Se só o código do app mudar, a demo deixa de ser a fonte da verdade e a próxima tela repete o
  erro. Antes do PR, confira que `web/` e `.ai/demos/` andam juntos quando o padrão mudou.
- **Antes de entregar uma tela, meça o pior caso:** rótulo mais longo de cada coluna (status, ação, série), banco ou
  descrição comprida, atraso de três dígitos, valor de cinco dígitos, item com todos os adornos, lista vazia e cheia;
  celular (≈ 390 px) e janela larga (≥ 1280 px), nos dois temas. Meça por script que nenhum conteúdo passa da própria
  coluna nem encosta no vizinho, em todas as colunas, e olhe uma captura.
- Demo antes do código, com limites e custos ditos na própria resposta. Se não consegue ver o resultado (toque real no
  celular, por exemplo), diga isso em vez de afirmar que funciona.
- A grande mudança de UX/UI prevista (item 21) segue a mesma regra, tela por tela. O Figma é a ferramenta do time de
  design; o conector ainda não está autorizado nesta conta.

## Git
- Rode `git branch --show-current` antes de commitar e use `git push origin <branch>` com o nome da branch. Não commite
  direto em `develop` ou `main`: o push numa branch de trabalho abre o PR rascunho sozinho.
- Depois que o PR de uma branch for mergeado, **não envie mais nada para ela**: o robô abre um PR novo só com esse
  commit (foi o #63, que colidiu com o #64) e o commit pode ficar de fora do merge que você esperava. Abra uma branch
  nova a partir da `develop`.
- **Mantra para começar qualquer trabalho** (sempre, nesta ordem):
  ```
  git fetch
  git checkout develop
  git pull
  git checkout -b feature/xxxx
  ```
- Use o prefixo da branch que combina com o assunto (`feature/`, `fix/`, `bug/`, `docs/` ou `chore/`), que é o que faz
  o robô abrir o PR rascunho no primeiro push. Antes do mantra, confira o `git status`: se houver alterações não
  commitadas, guarde-as ou commite-as, em vez de arrastá-las para a branch nova. Pelo mesmo motivo, veja o
  `git status` antes de `git add -A` (um `.exe` de build já foi parar num commit).

## Segurança
- **Segredos:** nunca peça nem repita token, chave ou senha no chat, e nunca os passe na linha de comando; guarde no
  SSM e leia com `Read-Host -AsSecureString`. Se um segredo aparecer numa conversa, recomende revogar.

## Testes e deploy
- **Testes:** rode `go test ./...` em `backend/` e `node web/recorrencia.test.js` antes de commitar. Quebre o código de
  propósito uma vez para ver o teste falhar. Teste não pode depender de variável de ambiente do CI (o deploy roda com
  `AMBIENTE=dev`).
- **Deploy:** o job verde não basta: confira o changeset e o status final da stack. Permissão nova na infra exige
  atualizar as roles do Actions (`infra/github-oidc.yaml`, uma stack por ambiente, com `Repositorio` explícito) antes do
  merge.

## Conduta
- **Provas:** não escreva "provavelmente" sobre o que dá para ler no log ou no código; leia e mostre a evidência.
- **Windows:** edições com `\n` dentro de heredocs do shell se corrompem; use a ferramenta de edição de arquivos.
