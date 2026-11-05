# Plan: Repetir, Recorrente e Avisar sem duplicação

Segue o [`intent.md`](intent.md) a partir do [`inventario.md`](inventario.md); o roteiro verificável da trilha A está no [`spec.md`](spec.md).

**Estado:** refeito em 2026-11-03, **sem aprovação do dono**. A trilha A não tem decisão de produto; a trilha B só começa depois das respostas da seção "Perguntas ao dono".

## Trilha A, tirar a duplicação sem mudar nada que se vê

Mesmo visual, mesmas regras, mesmas chamadas. Só front. O Lançar **não muda** nesta trilha (o `htmlAviso` e o `htmlRepeticao` ficam até a trilha B).

1. **A0, travar o comportamento de hoje (nenhuma linha de produção muda).** Capturar o HTML de cada tela por modo (Prever, Estender, Lote, Recorrência pelo histórico, Recorrência pelo Lançar, Fatura, Aviso padrão;
   telas principal, Repetir, Avisar, Mês de destino; com e sem erro, com e sem duplicados, com 6 prazos) e o corpo das chamadas que cada gesto faz (`POST /tx`, `POST /recorrencias`, `PUT` de fatura e de cartão).
   Isso vira um teste de node (`web/*.test.js`) com os arquivos de referência no repositório, no mesmo estilo dos testes que já leem `app.js` como texto. O método de captura (funções carregadas por texto
   com stubs, ou HTML copiado do app rodando com dados fictícios) é escolhido aqui. Sem A0, as outras fases não têm como provar "igual".
2. **A1, abertura e estado inicial.** `montarTc` passa a ser o único lugar que cria a janela (hoje `abrirPrever`, `abrirLote`, `abrirAvisoFatura` e `abrirAvisoPadrao` repetem o bloco); uma função
   `avisoInicial()` no lugar das quatro leituras de `base.aviso`; sai o ramo morto `avisoVisivel.pv` e o prefixo `pv` de `htmlAviso`. O teste da A0 continua verde.
3. **A2, uma tela Repetir e uma tela Avisar parametrizadas.** Arquivo novo `web/js/repetir.js` (nome a confirmar), como `versoes.js` e `tendencias.js`. O Repetir (três versões hoje) vira **uma** função com
   parâmetros (tem quantidade? tem interruptor? texto da quantidade e da nota); o Avisar recebe **entradas explícitas** (título, contexto, vencimento de referência, aviso, texto da nota, dados do balão) e
   devolve o aviso escolhido, **sem ler `pv`** nem decidir por modo. O Cartão e a fatura continuam abrindo só a tela Avisar com Salvar no rodapé, como hoje. O estado global `pv` sai.
4. **A3, o que impede a volta da duplicação.** Um teste (`web/telas-unicas.test.js`) que falha se (a) algo além de `montarTc` criar a janela cheia, (b) o título "Avisar no Telegram" ou "Repetir quantas
   vezes" for montado em mais de um lugar, (c) a seção Avisar ou Repetir ler estado global. E uma linha em `padrao-de-telas.md` e em `guardrails.md`: **tela que já existe se reaproveita; copiar é recusado
   pelo teste**.

Cada fase é um PR, uma branch a partir da `develop`. A trilha A não precisa de demo (não muda o que se vê), e cada PR diz isso e mostra o teste de igualdade verde.

## Trilha B, o redesenho (só depois das decisões e de uma demo completa)

Nada é decidido aqui. O que existe é uma exploração (as três demos de 2026-11-03), que cobre só uma parte do inventário.

### Perguntas ao dono (cada uma aponta as linhas do inventário que ela mexe)

1. **Repetir e Recorrente:** hoje são dois botões (E6/E7 e E8) e duas telas. O dono quer um seletor único dentro de uma tela, dois botões que abrem a mesma tela, ou deixar como está? (mexe em E2, E6 a E9, P10)
2. **Estender série e Parar recorrência** (E7 e E9) existem hoje e **não estavam nas demos**. Onde ficam, e o Estender continua sendo uma ação à parte?
3. **Tornar previstos em lote** (E10): usa a mesma tela, com a linha Mês de destino, ou continua uma tela própria?
4. **Lançar (novo):** Repetir, Recorrente e Avisar ficam sempre visíveis ou só com data futura (E1 a E3)? E o que acontece com o aviso quando a data é de hoje (P1, P2, P4)? E "N vezes" é "com este" (como hoje no
   Lançar) ou "além deste" (como no Prever) (P3)?
5. **Lançar (edição)** (E4): a edição mostra Repetir e Recorrente? O aviso na edição segue só com data futura?
6. **Lançar (cópia)** (E5): continua limpando o aviso e a repetição?
7. **Ícones e colunas** (E13, P10): o conjunto de botões das linhas muda? O padrão fixa seis lugares na tabela do computador.
8. **Cartão e fatura** (E11, E12): confirmar que seguem como hoje, só a tela Avisar com Salvar.

### O que a demo da trilha B precisa cobrir antes de qualquer decisão virar código
Uma demo por entrada do inventário, partindo da última demo-padrão da tela: Lançar (novo, edição, cópia), menu de ações do Histórico e de Futuros (com Estender e Parar recorrência), a tela Repetir em **todos** os
modos, a tela Avisar em **todos** os modos, o lote, a fatura e o aviso padrão do Cartão, e as marcas 🔔 e 🔁 das linhas. Cada demo diz o que a proposta faz perder.

## Premissas (guardrail de 2026-11-03)

As dez premissas escondidas estão no [`inventario.md`](inventario.md), seção 5 (P1 a P10), cada uma com "se a outra regra deixar de valer". Para a trilha A **nenhuma** muda de valor (não se mexe em nenhuma
das regras de que elas dependem), mas P5, P6 e P9 pedem cuidado: P5 e P6 porque a A2 mexe no estado `pv` e no que conta em `camadasAbertas()`, P9 porque o Estender precisa dos dados carregados. Para a trilha B,
P1 a P4 e P7 a P10 são decisões que o dono precisa ver antes da demo. **Revisão independente do roteiro (agente sem o contexto de quem escreveu): ainda não feita.**

## Riscos

- **A0 é o que dá segurança e também o que custa:** capturar o HTML de cada modo exige cuidado (datas relativas a "hoje" precisam de relógio fixo no teste). Sem isso, "igual" fica só na palavra de quem fez.
- **A2 mexe no estado global:** o botão voltar do celular depende do que `camadasAbertas()` conta (P6). O teste de camadas precisa cobrir a casca, a subtela e a janela de rodas antes.
- **A trilha A não corrige P1 a P4:** o Lançar continua com a regra de data futura e a herança do aviso. Isso é aceito e fica para a B.
- **Reaparecer a duplicação:** o teste da A3 é a única proteção; precisa rodar no CI (o CI roda todo `web/*.test.js` sem lista).

## Documentos a atualizar na entrega

`.ai/padrao-de-telas.md` (uma seção "Componentes reaproveitáveis" com a tela Repetir e a tela Avisar e a regra de reaproveitar) e `.ai/guardrails.md` (resumo). Na trilha B, também as demos e a tabela de demos.
