# Aprendendo a API no Postman

Esta pasta tem uma coleção e um ambiente do Postman para conhecer a API do app **com os dados da própria pessoa**.

| Arquivo | Para quê |
|---|---|
| `financas.postman_collection.json` | As requisições, em 5 pastas: **0** quem sou eu, **1** leitura (segura), **2** escrita (cria dados de verdade), **3** erros de propósito e **4** histórico de versões (busca por mês, versões com diff e restaurar), com exemplos de resposta em cada requisição |
| `financas.postman_environment.json` | As variáveis (`baseUrl`, `token`, `spaceId`, `bancoTeste`, e as do histórico: `mesHistorico`, `histId`, `versaoRestaurar`, `ultimaVersao`) |

## Primeiros passos
1. No Postman: **Import** e escolha os dois arquivos. Depois, no canto superior direito, selecione o ambiente **Finanças da Família (produção)**.
2. **`baseUrl`:** a URL da API, sem barra no final. É o valor `apiUrl` do `web/config.js` (ou o Output `ApiUrl` do deploy).
3. **`token`:** entre no app pelo navegador, abra as ferramentas do desenvolvedor (F12) → **Console** e rode:
   ```js
   JSON.parse(localStorage.getItem('financas.auth')).id
   ```
   Copie o texto (sem aspas) para a variável `token` do ambiente. Ele vale cerca de **1 hora**; quando as requisições passarem a dar 401, copie de novo.
4. Rode **0. Quem sou eu**. Ela grava o `spaceId` sozinha, e você já vê o seu papel (`owner` ou `member`).
5. Rode as requisições da pasta **1. Leitura**.

## Cuidados
- **O token é você.** Quem tiver o token age como você por 1 hora. Não envie, não cole em conversas nem em capturas de tela. A variável `token` está como `secret` no ambiente, mas **não exporte o ambiente preenchido** nem o commite.
- **Consumo da tabela:** a tabela do app é pequena (5 leituras por segundo, compartilhadas com quem estiver usando o app). Uma chamada com `limite=1000` lê muito e pode deixar o app lento para todo mundo. Use `limite` baixo (20 a 50) e intervalos de datas curtos (`de` e `ate`).
- **A pasta 2 grava de verdade.** Rode na ordem (criar, editar, ocultar) e termine com **Ocultar**, para o lançamento de teste sumir do Histórico e dos totais. Antes, preencha `bancoTeste` com um banco que exista nas Listas. O `statusTeste` padrão é `PREVISTO` (evento futuro: não entra no saldo).
- **`member` tem limites:** só edita e oculta o que ela mesma criou e não pode excluir nem alterar configurações. É de propósito, e a pasta 3 mostra isso na prática.

## As rotas da API
Todas levam `Authorization: Bearer <token>`. `{id}` é o `spaceId`.

| Método e caminho | O que faz | Quem |
|---|---|---|
| `GET /me` | Seu usuário e seus espaços | todos |
| `GET /spaces/{id}/tx` | Lista lançamentos (`de`, `ate`, `banco`, `status`, `q`, `limite`, `ordem`, `cursor`) | todos |
| `POST /spaces/{id}/tx` | Cria um lançamento | todos |
| `PUT /spaces/{id}/tx/{txId}?de=<dataEvento atual>` | Altera campos do lançamento | dono: todos; membro: só os dela |
| `DELETE /spaces/{id}/tx/{txId}?de=<dataEvento atual>` | Exclui (lógico: some das telas e fica no histórico) | só dono |
| `POST /spaces/{id}/tx/batch` | Importa até 25 lançamentos | só dono |
| `GET /spaces/{id}/seq` | Contador de alterações | todos |
| `GET /spaces/{id}/cfg/{nome}` | Configuração (`listas`, `tags`, `combos`, `saldos`, `cartoes`) | todos |
| `GET /spaces/{id}/fechamentos` | Fechamento mensal por banco | todos |
| `GET /spaces/{id}/historico?mes=AAAA-MM&q=` | Lançamentos que mudaram no mês (excluídos e alterados), por mês da alteração | dono: todos; membro: só os dela |
| `GET /spaces/{id}/tx/{txId}/versoes` | Versões do lançamento, com diff e se cada uma pode ser restaurada (`?de=` para lançamento antigo, nunca alterado) | dono: todos; membro: só os dela |
| `POST /spaces/{id}/tx/{txId}/versoes/{n}/restaurar` | Restaura a versão `n` (grava uma versão nova; corpo vazio) | dono: todos; membro: só os dela e que existem |

A `dataEvento` é ISO 8601 em UTC (ex.: `2026-09-28T22:08:00.000Z`) e faz parte da chave do lançamento, por isso editar ou excluir pede a data atual em `?de=`.

## Exercícios para aprender
1. Rode `0. Quem sou eu` e descubra o seu papel e o `spaceId`.
2. Liste os 20 lançamentos mais recentes e identifique `valor`, `tipo` e `banco` de um deles.
3. Busque só um mês, pelo intervalo de datas. O que muda se você trocar `ordem=asc` por `desc`?
4. Veja `cfg/listas` e compare com a aba **Listas** do app.
5. Crie o lançamento de teste, confira o `_seq` na resposta, ache-o no app (aba **Futuros**, se o status for PREVISTO) e depois rode **Ocultar**.
6. Rode a pasta **3** e explique por que cada requisição devolveu 401, 403, 404, 400 ou 403.
7. Depois de criar e editar o lançamento de teste (pasta 2), rode **Busca do mês** da pasta **4** e ache-o. Abra **Versões de um lançamento** e leia o `diff` de cada versão.
8. Rode **Restaurar uma versão** e veja que a lista de versões ganhou uma versão nova (`restore`), sem apagar as antigas. Tente de novo e leia o `409`.
