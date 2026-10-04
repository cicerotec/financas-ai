# Backlog

Atualizado em 2026-10-04. Ordem = prioridade. Marque `[x]` ao concluir.

## Próximos passos (liberar o app para a esposa)

1. [x] **Autorização por papel no backend (Go) + testes automatizados da matriz.**
   Tabela de permissões abaixo. A trava fica na Lambda, não no front. Feito: `authz.go`, `tags.go`,
   `authz_test.go`, `handler_test.go` (banco em memória). **Falta publicar** (`.\scripts\deploy.ps1 <codigo>`);
   a API nova `POST /spaces/{sid}/tags` precisa ser usada pelo front no item 2.
2. [x] **Front por papel.** `/me` devolve o papel; a tela esconde o que o `member` não pode usar
   (aba Importar, edição de saldos/cartão/listas, botões de editar nos registros alheios).
   Feito e verificado na tela com dados inventados (owner e member). Falta publicar o backend (item 1) e
   exercitar com um usuário member de verdade (item 3).
3. [ ] **Teste local com um usuário `member`** (janela anônima; `seed.sh` com e-mail `+alias`).
4. [ ] **Hospedagem do front em S3 + CloudFront** (bucket privado, HTTPS, domínio `*.cloudfront.net`):
   - `template.yaml`: bucket, acesso do CloudFront, distribuição; CORS da Lambda e URLs de retorno do
     Cognito aceitando CloudFront **e** `localhost`.
   - `scripts/publicar-front.ps1`: envia `web/` (inclui `config.js`, que fica fora do git) e limpa o cache.
5. [ ] **Convidar a esposa** (`scripts/seed.sh financas <dono> <email-dela>`), instalar na tela inicial do celular.

### Matriz de permissões (decidida)

| Ação | owner | member |
|---|---|---|
| Criar lançamento | sim | sim |
| Copiar lançamento | sim | sim |
| Editar / ajustar fatura | todos | **só os criados por ela** (`criadoPor == sub`) |
| Ocultar lançamento | todos | **só os dela** |
| Excluir lançamento | todos | **não** |
| Criar tag | sim | **só tags que não existem** (servidor só acrescenta) |
| Renomear/excluir tag, combos, bancos, status, flags | sim | não |
| Ver saldos, cartão, tendências | sim | sim (só leitura) |
| Alterar saldo inicial, fechamento, valor do banco, conferir fatura, início do controle | sim | não |
| Aparência (cores, layout, ordem) | sim | sim, **só a dela** |
| Importar / exportar / limpeza | sim | não |

Decisões: ela pode ler **todos** os lançamentos e notas (saldos/cartão/tendências são calculados no
navegador). Hospedagem: **S3 + CloudFront** (GitHub Pages descartado; localhost não serve para ela:
sem HTTPS o PKCE e o Cognito não funcionam).

## Visibilidade entre usuários (em discussão; só o nome foi implementado)

- [x] **Nome de exibição (apelido):** cada pessoa define o seu em Listas > Seu nome; `scripts/definir-nome.sh`
      define pelo e-mail. Guardado no vínculo `USER#sub / SPACE#id`. Falta publicar o backend.
- [ ] Atualização da tela **por gatilho** (foco da aba, entrar nas abas, antes de salvar, botão Atualizar) com um
      contador de revisão do espaço e uma checagem lenta de segurança (3 a 5 min). Sem temporizador curto.
- [ ] Mostrar quem criou/alterou/ocultou cada lançamento e quando (`atualizadoPor/Em`, `ocultoPor/Em`);
      lista de membros do espaço (item espelho `SPACE#id / MEMBER#sub`) para converter `sub` em nome.
- [ ] "Novo desde a sua última visita": etiquetas automáticas (novo/alterado/oculto) e contagem no Histórico.
- [ ] "Atividade recente": registro de eventos do servidor com validade (~90 dias).
- [ ] Idempotência de criação (id gerado na tela, travar o botão Salvar) e edição concorrente (409 se mudou).
- Descartado: bloquear duplicidade por conteúdo (a mesma compra pode ter outro horário e outra descrição).

## Confiabilidade e dados

- [ ] Conferir no navegador a aba Cartão: início do controle, fatura conferida e "mudou desde a conferência".
- [ ] Carregar anos anteriores (do mais novo para o mais antigo); definir o **início do controle** de cada cartão.
- [ ] Conferir totais de fatura de cada cartão contra o banco (MERCADO PAGO, NUBANK, INTER, SANTANDER).
- [ ] Importação do `.xlsx` usar o endpoint de lote (hoje grava um lançamento por requisição).
- [ ] Aviso "já existe um igual" ao salvar manualmente usa só os 1000 mais recentes carregados.
- [ ] Cartão: campos **valor pago** e **observação** por fatura (exceções como extrato ≠ pago).

## Arquitetura

- [ ] Mover **saldos e faturas do cartão** para a Lambda (hoje o navegador lê todo o histórico).
      Isso também permitiria esconder notas/lançamentos alheios do `member`, se um dia for preciso.
- [ ] **IA** ("Preencher com IA") via Lambda, com a chave da Anthropic no SSM Parameter Store.
- [ ] Testes automatizados (backend e a lógica de faturas do front).
- [ ] Salvamento simultâneo de configurações: hoje o front repete por cima no conflito (409); mesclar.
- [ ] Avaliar índice (GSI) por banco se o histórico crescer.

## Segurança e operação

- [ ] Revisar o armazenamento do token (`localStorage`) e adicionar limite de taxa na Function URL.
- [ ] AWS Budget de US$ 1 com alerta por e-mail (conferir se existe).
- [ ] Backup periódico (exportar JSON) e/ou point-in-time recovery do DynamoDB.

## Git e releases

- [ ] Fazer o merge da PR #1 em `develop`.
- [ ] Primeira release: PR `develop → main` e tag `v0.1.0`.

## Ideias

- `#Garantia` com filtro de garantias vigentes (data de vencimento na nota).
- Anexar foto/PDF da nota fiscal ao lançamento (bucket S3).
- Fatura: aviso quando uma compra `CREDITO EX`/`CREDITO IN` cair fora do esperado (a planilha as ignora).
