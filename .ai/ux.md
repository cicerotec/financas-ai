# UX

Origem: observar a esposa usando o app (sem explicar nada) e anotar onde ela hesita. Repetir a cada rodada.
- **Barra inferior no celular: descartada** (2026-10-15). No C6 a barra some em toda tela empilhada e no teclado; aqui a
  navegação é plana. Resolvido com os atalhos Histórico e Saldos antes do ⋮ (`padrao-de-telas.md`, "Menu do app").
- **Alvos de toque** de Cartão e Saldos (`font-size:12px; padding:4px 10px`) abaixo de 44px; trocar por linha de ações.
- **Acessibilidade:** abas sem `role="tablist"`/`aria-selected`; `‹ ›` das janelas de faturas são `<span>`, não
  funcionam por teclado.
- **Código:** muito `style` inline; `web/index.html` passa de 3.200 linhas (separar CSS e JS).
- **Seletor de tags:** o do Lançar e o dos filtros duplicam lógica; unificar em um componente.
- **Bug (item 25): combo de bancos reseta após copiar pelo Histórico.** Ao copiar um lançamento pelo Histórico e
  lançar o registro, o combo de bancos volta sozinho para o primeiro elemento da lista, e isso pode gerar erro no
  cadastro (o lançamento sai no banco errado sem a pessoa perceber). Investigar onde o formulário de Lançar repopula
  ou reinicia os combos (cópia, salvar, atualização da tela por `seq`) e manter o banco escolhido.
  **Corrigido:** `renderListas()` refazia os `<option>` dos combos (Status, Banco e bancos da transferência) e o
  navegador voltava ao primeiro; agora a escolha atual é guardada e devolvida depois de refazer a lista.
- **Lançar redesenhado (2026-10-19):** o formulário passou de 17 controles à vista para 4 no celular; ver "Lançar (tela)" em `padrao-de-telas.md`. Observar o uso real: se Nota ou Tags forem usadas em todo lançamento, o toque a mais em "Mais" e "Tags" pode pedir ajuste.
- **Lançar, a testar no ambiente dev (PR 146):** salvar, editar, copiar do Histórico, IA e Transferência com data e hora só tiveram o visual conferido, não o fluxo. Conferir também o × das tags no computador e o "Mais" abrindo sozinho ao editar um lançamento com nota.
- **Observado e já resolvido:** salvar sem resposta, transferência sem data, Enter sem efeito na busca, tudo na tela
  em Histórico, X de status apagando sem aviso.
