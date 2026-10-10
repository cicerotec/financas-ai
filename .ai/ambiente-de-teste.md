# Ambiente de teste (`dev`)

Decidido: **opção A**, uma segunda stack inteira (`financas-dev`) com o mesmo template, em vez de só outra tabela ou de
prefixo na tabela real (isolamento fraco, risco de misturar com dados reais). **Dados fictícios**, ambiente **sempre
ligado**, **publicação manual** (automatizar a partir da `develop` fica para depois).
- Hoje há nomes fixos no `template.yaml`: tabela `FinancasApp`, função `financas-api`, pool `financas`; os scripts
  `seed.sh`, `definir-nome.sh` e `limpar-espaco.py` também fixam `FinancasApp`. Uma segunda stack colidiria.
- Parâmetro `Ambiente` (`prod` por padrão, `dev`). O padrão tem que gerar **exatamente os nomes atuais**, senão o
  CloudFormation tentaria substituir a tabela real (o `DeletionPolicy: Retain` já protege, mas conferir o changeset).
  Em dev: `FinancasApp-dev` etc.
- `samconfig.toml` com perfil `dev` (`sam deploy --config-env dev`), com `CognitoDomainPrefix` próprio.
- Scripts com `-Ambiente dev` (`deploy.ps1`, `publicar-front.ps1`, `seed.sh`, `definir-nome.sh`). A guarda de
  publicação passa a aceitar a `develop` quando o alvo for dev (hoje avisa para quem não está na `main`).
- Front: `web/config.dev.js` (URL da API e login do dev), enviado pelo `publicar-front.ps1 -Ambiente dev`; bucket e
  distribuição separados.
- Dados: script que cria um dono e um membro de teste (e-mail "+dev") e carrega dados fictícios com volume parecido
  (status, bancos, cartões, combos, centenas de tags) para testar as telas como a esposa usa.
- Custo: tabela de teste com 5/5 RCU/WCU; o always-free são 25 no total e a produção já usa 5/5, então cabe.
- **Concluído.** Parâmetro `Ambiente` no template, `-Ambiente dev` nos scripts e `--dev` no `limpar-espaco.py`. A
  publicação é **bloqueada** fora da branch do ambiente (`main` para prod, `develop` para dev), sem opção de pular. O
  perfil `dev` do `samconfig.toml` fica só na máquina (o arquivo é ignorado pelo git): stack `financas-dev`,
  `Ambiente="dev"`, `CognitoDomainPrefix="financas-cicero-dev"`. O `web/config.dev.js` (fora do git) leva
  `ambiente: "dev"`, que acende a faixa laranja "AMBIENTE DE TESTE" e o prefixo `[DEV]` no título da aba.
- Dados: `scripts/dados-ficticios.py` (grava só em `FinancasApp-dev`; ids fixos, rodar de novo sobrescreve; recomeçar do
  zero com `limpar-espaco.py <space-id> --dev --confirmar`). Dono e membro criados com `scripts/seed.sh financas-dev`.
- Fica de fora: configuração de **cartões** e **saldo inicial** por banco nos dados fictícios; as despesas superam as
  entradas, então os saldos ficam negativos (serve para testar esse caso). A URL é a do CloudFront; domínio próprio só
  se entrar domínio para a produção.
