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
- **Bug em produção:** veja o roteiro [abaixo](#bug-em-produção-ordem-de-preferência). Só ir à tag antiga se for
  preciso reproduzir o bug (`git checkout v0.1.0`).
- **Sem branches `release/x.y`** por enquanto; só se for preciso congelar uma leva grande para testar enquanto a
  `develop` segue em frente.
- **Publicar na AWS é feito pelo GitHub Actions**, não pela release: o push na `develop` publica o dev e o push na
  `main` publica a produção (depois da aprovação do environment `prod`); a tag e a release saem depois desse deploy
  (veja [`deploy-github-actions.md`](deploy-github-actions.md)). Os scripts `deploy.ps1` e `publicar-front.ps1` ficam
  como alternativa manual, para emergência, e só publicam a partir da branch do ambiente (`main` ou `develop`).

## Bug em produção: ordem de preferência

Do mais simples ao mais pesado. Use o primeiro que resolve; hotfix é o último recurso (ninguém gosta dele: é um
segundo caminho para a produção, que ainda precisa descer de volta para a `develop`).

1. **Revert, pelo fluxo normal** (quando o bug veio de um PR que dá para apontar).
   - No PR mergeado na `develop`, use o botão **Revert**: o GitHub abre o PR de reversão (o robô não precisa fazer nada),
     você mergeia com **merge** e o PR de release leva a reversão para a `main` com a aprovação do `prod`.
   - Pelo terminal, de uma branch nova a partir da `develop`: `git revert -m 1 <sha-do-merge>` (o `-m 1` é obrigatório
     para um commit de merge).
   - **Para reintroduzir a mudança depois,** reverta o revert (senão o Git acha que ela já está lá) ou abra uma branch
     nova a partir da `develop` com a correção.
   - A correção de verdade vem depois, com calma, num PR normal.
2. **Correção na `develop` e release imediato,** se a `develop` só tem o que já foi lançado: a correção entra como qualquer
   assunto e o PR de release sai logo em seguida. Como o release leva tudo o que está na `develop`, confira antes se não
   há outra coisa não lançada ali.
3. **Hotfix a partir da `main`,** só quando a `develop` tem coisa não lançada e o revert não serve:
   ```
   git fetch
   git checkout main
   git pull
   git checkout -b hotfix/descricao
   # corrige, commita (use `fix:` para a versão subir o último número) e:
   git push origin hotfix/descricao
   gh pr create --base main --fill
   ```
   - O robô **não** abre PR para `hotfix/**` (o destino é a `main`, não a `develop`); o `ci.yml` roda no push e dá o
     check `validar`, que a `main` exige. Mergear com **merge**.
   - O hotfix **não passa pelo ambiente de dev** (o dev só publica a `develop`): teste a correção antes de pedir a
     aprovação do `prod`.
   - Depois do deploy, a tag (`vX.Y.Z`, só o último número sobe) e a release saem sozinhas.
   - **A descida:** abra o PR da correção de volta para a `develop`, também com **merge** e a mesma branch/commits (nunca
     squash nem rebase, que criam commits novos e geram conflito no próximo release):
     `gh pr create --base develop --head hotfix/descricao --fill`. Se a branch já foi apagada, use
     `gh pr create --base develop --head main` e mergeie.
   - Sem a descida, a correção se perde no próximo release.

Reforço: tanto o revert quanto o hotfix vão por PR e **merge**, porque o PR de release lista os commits
`Merge pull request #N` e a descida do hotfix depende dos mesmos commits nas duas branches.
