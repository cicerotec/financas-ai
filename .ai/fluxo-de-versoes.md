# Fluxo de branches e versões (convenção simples)

Convenção de trabalho e de versões. Depois dela, o PR de release e a tag passaram a sair automáticos (itens 25 e 26 do
backlog; veja [`deploy-github-actions.md`](deploy-github-actions.md)). Os comandos manuais abaixo ficam como referência
e para emergência. As regras de branch para a IA (mantra, prefixos, não reaproveitar branch) estão em
[`guardrails.md`](guardrails.md).

- **Trabalho:** uma branch nova a partir da `develop` por assunto (`feature/...`, `docs/...`) e PR para a `develop`.
- **Lançar uma versão:** PR `develop → main`, merge (commit de merge), depois a **tag anotada** `vX.Y.Z` no commit da
  `main` e a release no GitHub. A tag marca **o que foi para produção**. A release é só a página de registro: não
  publica nada.
  ```
  git fetch origin
  git tag -a v0.2.0 -m "v0.2.0: resumo" origin/main
  git push origin v0.2.0
  gh release create v0.2.0 --verify-tag --title "v0.2.0" --notes "..."
  ```
- **Número:** `0.x.y` até ficar estável. Correção = sobe o último número (`0.1.1`); funcionalidade nova = sobe o do meio
  (`0.2.0`); `1.0.0` quando estável.
- **Bug em produção:** se a `develop` só tem coisas já lançadas, corrige na `develop` como qualquer assunto e lança a
  versão seguinte. Se a `develop` já tem coisa que não foi lançada, cria `hotfix/...` a partir da `main`, PR para a
  `main`, tag da correção e PR de volta para a `develop` para a correção não se perder. Só ir à tag antiga se for
  preciso reproduzir o bug (`git checkout v0.1.0`).
- **Sem branches `release/x.y`** por enquanto; só se for preciso congelar uma leva grande para testar enquanto a
  `develop` segue em frente.
- **Publicar na AWS é separado** da release: backend com `deploy.ps1`, front com `publicar-front.ps1`. Para a versão
  no ar bater com a tag, publicar a partir da `main` depois de lançar.
