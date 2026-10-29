# Guardrails: regras para a IA que programa neste repositório

Leia antes de mexer em qualquer coisa. Estas regras valem para toda tarefa, sem exceção. Se um pedido do dono
contrariar uma regra daqui (segurança, arquitetura ou uma decisão registrada), **aponte o conflito antes de fazer** em
vez de obedecer. Se a especificação não cobre um caso, **pergunte** em vez de decidir.

## Arquitetura e dados
Referência: [`arquitetura.md`](arquitetura.md) (decisões de 2026-10-10).
- **O front não decide.** Toda regra de negócio fica no backend e chega pronta pela API, nunca por etiqueta (nome de
  status ou de banco).
- **O front só exibe e não é fonte de número (2026-10-18).** Saldo, fechamento, total, fatura, "é futuro", "afeta o
  saldo" e "fora do total" são calculados e decididos pelo servidor. O front pode montar e até calcular **para mostrar**
  (a aba Tendências, por exemplo), mas **nunca envia um valor calculado como verdade**: o backend rejeita campo derivado
  que venha do cliente, e quem sabe forjar a requisição não ganha nada. Antes de aceitar um campo novo na API, pergunte
  se o servidor consegue recalculá-lo; se sim, ele não entra no corpo. Detalhes e o que falta fazer em
  [`arquitetura.md`](arquitetura.md) e [`security.md`](security.md).
- **Status é do backend; o front não trata status de forma nenhuma (2026-10-24).** O backend **manda a lista** de status; o front só a
  exibe, guarda a escolha da pessoa e a devolve como veio. Nada de nome de status, lista fixa, predicado por nome (`ehFuturo`,
  `afetaSaldo`, `ehCredito`...), cor ou regra escondida em nome, nem `excluirDoTotal` calculado no navegador. Ver
  [`decisoes.md`](decisoes.md).
- Camadas `controller` → `service` → `domain`, adaptadores atrás de interfaces e dependências por construtor, sem
  globais. Em `backend/` cada camada é uma pasta (`controller/`, `service/`, `repository/`, `auth/`, `domain/`) e o
  teste `arquitetura_test.go` falha se um pacote importar o que não pode (por exemplo, `service` importando a AWS).
  Código novo entra na pasta da sua responsabilidade; o que ainda está na raiz migra por fatia (item 32).
- **Não crie, renomeie nem remova chave ou campo no DynamoDB sem discutir com o dono.** Cada campo (como
  `faturaAjuste`) carrega uma decisão que não é óbvia no código.
- Nada se sobrescreve sem versão no histórico (item 31 do backlog).
- **Cada tela lê só o que é dela, e só quando está aberta (2026-10-27).** A subida (tela Lançar) lê `listas`, `aparencia`,
  `tags` e `combos` e nada mais. Os 1000 lançamentos são do Histórico e da Tendências (`sobDemanda` em `web/api.js`,
  `irParaAba` em `web/js/app.js`); `/futuros` é da aba Futuros e dos gestos que olham a série (Prever, Estender, repetir
  ao lançar); scripts pesados (`tendencias.js`, `versoes.js`, `xlsx`) vêm na primeira visita. Recarregar é pelo nome
  (gravar `tags` relê só `tags`), nunca "tudo", e o timer só confere o `seq` se alguma tela já leu lançamentos.
  `web/carga.test.js` falha se a subida voltar a ler o que não é dela. Tela nova declara o que lê em `irParaAba`.
  Excluir ou desfazer um previsto de ano antigo (buscado a pedido, fora de `GET /futuros`) relê só aquele ano, nunca
  `/futuros` nem os outros anos: `GET /futuros/ano` não tem cache e estourou os 5 RCU em 2026-10-27 (`futuros ano:
  ProvisionedThroughputExceededException` no log).

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
    blocos e uma faixa de atraso (só quando há atraso), valores que nunca quebram; saldo do mês à direita do título
    de cada mês (só o valor, com sinal e cor); total da lista no Histórico à direita do contador (soma de renda
    menos gasto do que está exibido; **retirado até o item 42**); linhas de pelo menos 56 px; excluir e salvar uma edição pedem confirmação e
    depois mostram um aviso de 6 s com Desfazer (padrão 11b, provisório até o histórico de versões).
  - **Lançar** (demo `demo-lancar-layout.html`): valor grande primeiro, Data/Banco/Status com seletor nativo, Tags em janela de baixo e Salvar fixo no celular; IA e Tags à direita no computador; Transferência é um modo no topo.
  - **Histórico** (demos `demo-historico-layout.html` e `demo-historico-periodo.html`): período 30d · 90d · 180d · 📅 (um mês) no topo, abrindo em 30d; busca só no que está marcado, com "Buscar em tudo" e aviso; Filtros numa janela com fichas; linha como a de Futuros agrupada por dia (por mês, com soma, na busca e na tabela de períodos longos); nota só em ícone e tags em uma linha.
  - **Configurações** (demo `demo-configuracoes.html`, antes aba Importar, só do dono): três seções recolhidas, Cartão (fechamentos da planilha), Importar (planilha, colar dados, backup) e Exportar; a aba Cartão não guarda aviso do que saiu.
  - **Forma de pagamento** (demo `demo-forma-pagamento.html`): um cartão só para status e cores; interruptor "Cores nas linhas" (igual a "Mostrar ocultos") e, por status, a bolinha de cor com a linha de exemplo do Histórico e de Futuros; sem fichas de estilo em Aparência.
  - **Previstos em lote** (demo `demo-previstos-em-lote.html`): no Histórico, Selecionar marca os lançamentos e abre o fluxo do Prever com um mês de destino, Repetir e Avisar no Telegram; **nunca pergunta status** (o previsto sai com o status da origem); até 100 por vez.
  - **Botão voltar** (demo `demo-voltar-navegador.html`): troca de aba e cada janela ou tela cheia é uma entrada do histórico do navegador (`web/js/voltar.js`); o voltar fecha a camada de cima, o aviso de Desfazer fica fora da pilha, o formulário de edição com texto digitado pergunta antes de descartar. **Janela ou tela cheia nova entra em `camadasAbertas()` e `fecharCamadaTopo()` de `web/js/app.js`.**
  - **Computador:** a mesma tela vira uma tabela de largura total (colunas alinhadas, mínimo de 1120 px a partir de
    1160 px de janela, container de até 1200 px) com as ações na própria linha, **só ícones** em seis lugares fixos (demo `demo-icones-acoes.html`), apagadas até o mouse passar ou o foco
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
  direto em `develop` ou `main`: o push numa branch de trabalho abre o PR sozinho, já pronto para mergear.
- **Aprovado, entregue (2026-10-28):** depois que o dono aprova (a demo, a tela ou a correção), **sempre commite e abra o PR** na
  mesma tarefa, sem esperar que ele peça: teste, commite na branch certa e faça `git push origin <branch>`, que abre o PR.
  Antes da aprovação não se commita nem se envia nada. Um assunto por branch e por PR.
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
  o robô abrir o PR no primeiro push. Antes do mantra, confira o `git status`: se houver alterações não
  commitadas, guarde-as ou commite-as, em vez de arrastá-las para a branch nova. Pelo mesmo motivo, veja o
  `git status` antes de `git add -A` (um `.exe` de build já foi parar num commit).

## Segurança
- **Segredos:** nunca peça nem repita token, chave ou senha no chat, e nunca os passe na linha de comando; guarde no
  SSM e leia com `Read-Host -AsSecureString`. Se um segredo aparecer numa conversa, recomende revogar.
- **Permissões e autenticação:** leia [`security.md`](security.md) antes de mexer em rota, papel, token, CORS ou IAM. Rota
  nova exige ação nova em `auth/authz.go` com teste; o que não está liberado é negado. O front não é fronteira de segurança.
- **Dado real nunca no git** (o repositório é público): exemplos e fixtures só com dados fictícios.

## Testes e deploy
- **Testes:** rode `go test ./...` em `backend/` e cada `node web/*.test.js` (o CI e o deploy rodam todos os que existirem, sem lista) antes de commitar. Quebre o código de
  propósito uma vez para ver o teste falhar. Teste não pode depender de variável de ambiente do CI (o deploy roda com
  `AMBIENTE=dev`).
- **Deploy:** o job verde não basta: confira o changeset e o status final da stack. Permissão nova na infra exige
  atualizar as roles do Actions (`infra/github-oidc.yaml`, uma stack por ambiente, com `Repositorio` explícito) antes do
  merge.

## Conduta
- **Provas:** não escreva "provavelmente" sobre o que dá para ler no log ou no código; leia e mostre a evidência.
- **Windows:** edições com `\n` dentro de heredocs do shell se corrompem; use a ferramenta de edição de arquivos.
