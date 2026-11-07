# Plan: combos sugeridos calculados pelo servidor (item 57)

Como fazer o que o [`intent.md`](intent.md) pede. Segue a camada do backend (`controller` → `service` → `repository`, regra em `domain`).

## Onde guardar

Dentro do item que já existe, **sem chave nova**:

```
CFG#COMBOS
  lista:     [ { "tags": ["mercado","mercado-mensal"] } ]                    <- manuais (como hoje)
  sugeridos: [ { "tags": ["mercado","mercado-mensal"], "contagem": 5 } ]     <- automaticos (novo)
  geradoEm:  "2026-11-08"                                                    <- novo, informativo
```

Dois campos, não uma lista só: o botão de salvar combo grava o item inteiro e a tela de Listas apaga combo pela posição na `lista`. Com tudo numa lista o front reenviaria os automáticos a cada gravação manual.

Dois cuidados:

- **O PUT do `config/combos` substitui o item todo.** O servidor preserva `sugeridos` e `geradoEm` na gravação manual, como já faz com `ativoAte` e o catálogo de status em `listas`.
- **A rotina não sobe o `version`.** Se subisse, uma tela de Listas aberta tomaria "conflito" ao salvar. Ela faz uma atualização parcial (`SET sugeridos, geradoEm`), sem tocar em `version` nem em `lista`. Se o item ainda não existe, a rotina o cria com `version` 0.

## A rotina

- **Regra nova no `template.yaml`**, separada da dos avisos: `cron(0 6 ? * SUN *)` (domingo 03:00 em São Paulo), entrada `{"acao":"combos","espacos":["<sid>"]}`.
- **`roteador.entrada`** reconhece `acao == "combos"` e chama o service novo (`service/combos.go`), que devolve `{"espacos": n, "sugeridos": n}`.
- **Leitura:** `Listar` já pagina por SK; a rotina usa a faixa `TX#<hoje - 12 meses>` a `TX#<hoje>` em páginas pequenas, com uma espera curta entre elas (5 unidades de leitura). Ignora excluídos e previstos.
- **Conta** (`domain/combos.go`, função pura e testada): chave do conjunto = tags ordenadas; só conjuntos com 2 tags ou mais e contagem 3 ou mais; ordena por contagem (empate: ordem alfabética da chave) e fica com 10.

## O app

- `combosAtuaisLista()` passa a juntar `combosSalvos` (de `lista`) e `combosSugeridosServidor` (de `sugeridos`), tirando o sugerido que for igual a um manual e dando a contagem ao manual.
- `contagemConjuntos()` e o uso de `lancamentosCache` para combos deixam de existir no Lançar. O que o Histórico e a Tendências fazem com combos como **filtro** (item 42) não muda neste item.
- O `onSnapshot` de `config/combos` já lê o item inteiro: só passa a guardar `d.sugeridos`.

## Decisões para o dono

- **D1. Como a rotina descobre os espaços.** Não há lista global. Recomendo **passar o(s) espaço(s) na entrada da regra** (`"espacos":["financas"]`): zero consulta extra, nenhum campo novo. Quando o app abrir a outros usuários (item 49), troca-se por uma lista. A alternativa (varrer a tabela atrás de `META`) custa leitura demais.
- **D2. Janela de 12 meses** em vez dos 1000 últimos. Recomendo 12 meses.
- **D3. Guardar 10, mostrar 5.** Recomendo.

## Riscos

- Estourar a leitura da tabela na madrugada: páginas pequenas, espera entre elas e uma vez por semana.
- Combo sugerido ficar velho até uma semana.
- Esquecer de preservar `sugeridos` no PUT manual: coberto por teste (R9 da spec).
