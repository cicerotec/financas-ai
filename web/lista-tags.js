// Lista de tags com busca (demo-listas-v9-tags): um campo de busca, as criadas mais recentemente e "Ver todas as tags (N)"
// com ordem e "Mostrar mais". E um componente: recebe onde desenhar e o que fazer no toque, guarda o proprio estado e nao usa
// id fixo, para servir a mais de uma tela na mesma pagina. Hoje so o cartao Tags de Listas o usa; a janela de Tags do Lancar,
// o filtro do Historico e a Tendencias ainda tem o codigo deles. A logica (buscar, ordenar, recentes) e pura e testada em
// lista-tags.test.js.
(function (raiz) {
  const MAX_BUSCA = 40;
  const ORDENS = [['az', 'A → Z'], ['za', 'Z → A'], ['rec', 'Criadas recentemente']];

  function norm(s) { return String(s || '').normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase(); }
  function limpar(s) { return String(s || '').trim().replace(/^#/, ''); }
  function esc(t) { return String(t == null ? '' : t).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;'); }

  // A busca nao diferencia maiusculas nem acento e aceita "#" no comeco. Ordem do resultado: as que comecam com o texto,
  // depois as mais curtas, depois o alfabeto. `tags` vem na ordem de criacao e nao e alterada.
  function buscar(tags, texto) {
    const q = norm(limpar(texto));
    if (!q) return [];
    return tags.filter((t) => norm(t).includes(q))
      .sort((a, b) => (norm(a).startsWith(q) ? 0 : 1) - (norm(b).startsWith(q) ? 0 : 1) || a.length - b.length || a.localeCompare(b));
  }
  function recentes(tags, n) { return tags.slice().reverse().slice(0, n); }
  function ordenar(tags, ordem) {
    const l = tags.slice();
    if (ordem === 'rec') return l.reverse();
    l.sort((a, b) => a.localeCompare(b));
    return ordem === 'za' ? l.reverse() : l;
  }
  function destacar(tag, texto) {
    const q = norm(limpar(texto));
    const i = q ? norm(tag).indexOf(q) : -1;
    if (i < 0) return esc(tag);
    return esc(tag.slice(0, i)) + '<mark>' + esc(tag.slice(i, i + q.length)) + '</mark>' + esc(tag.slice(i + q.length));
  }

  // op: { tags(): lista na ordem de criacao; aoExcluir(tag): o toque no ×; topo: quantas recentes (8); pagina: tamanho do "Mostrar mais" (30) }
  function criar(el, op) {
    const topo = op.topo || 8, pagina = op.pagina || 30;
    const S = { busca: '', abertas: false, ordem: 'az', limite: pagina };
    el.innerHTML =
      '<div class="tg-busca"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7" fill="none" stroke="currentColor" stroke-width="2"/><path d="m20 20-3.5-3.5" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>' +
      '<input type="search" data-p="busca" placeholder="Buscar tag" aria-label="Buscar tag" autocomplete="off" autocapitalize="off" autocorrect="off" spellcheck="false" enterkeyhint="search"></div>' +
      '<div data-p="topo"></div>' +
      '<button type="button" class="tg-mais" data-p="todasBtn" aria-expanded="false"></button>' +
      '<div data-p="corpo" hidden><div class="tg-lista-topo"><h3 class="tg-sec">Todas as tags</h3><label class="tg-ord"><span>Ordenar</span>' +
      '<select data-p="ordem" aria-label="Ordenar tags">' + ORDENS.map((o) => '<option value="' + o[0] + '">' + o[1] + '</option>').join('') + '</select></label></div>' +
      '<div class="tg-res" data-p="todas" role="group" aria-label="Todas as tags"></div><button type="button" class="tg-mais" data-p="mais" hidden></button></div>';
    const p = (k) => el.querySelector('[data-p="' + k + '"]');
    const chip = (t) => '<span class="tg-del"><span class="nm">' + destacar(t, S.busca) + '</span><button type="button" data-del="' + esc(t) + '" aria-label="Excluir a tag ' + esc(t) + '" title="Excluir">×</button></span>';

    function render() {
      const tags = op.tags(), bruto = limpar(S.busca);
      if (!tags.length) {
        p('topo').innerHTML = '<p class="tg-vazio">Nenhuma tag ainda.</p>';
        p('todasBtn').hidden = true; p('corpo').hidden = true;
        return;
      }
      if (bruto) {
        const achou = buscar(tags, bruto), mostra = achou.slice(0, MAX_BUSCA);
        p('topo').innerHTML = achou.length
          ? '<div class="tg-sec">' + achou.length + (achou.length === 1 ? ' tag achada' : ' tags achadas') + '</div><div class="tg-res">' + mostra.map(chip).join('') + '</div>' +
            (achou.length > mostra.length ? '<p class="tg-vazio">Mostrando ' + MAX_BUSCA + '. Digite mais para refinar.</p>' : '')
          : '<p class="tg-vazio">Nenhuma tag com “' + esc(bruto) + '”.</p>';
      } else {
        p('topo').innerHTML = '<div class="tg-sec">Criadas recentemente</div><div class="tg-res">' + recentes(tags, topo).map(chip).join('') + '</div>';
      }
      // "Ver todas" e a busca sao caminhos separados: com texto no campo, a lista inteira some
      p('todasBtn').hidden = !!bruto;
      p('todasBtn').textContent = (S.abertas ? 'Esconder' : 'Ver') + ' todas as tags (' + tags.length + ')';
      p('todasBtn').setAttribute('aria-expanded', String(S.abertas));
      p('corpo').hidden = !!bruto || !S.abertas;
      if (S.abertas && !bruto) {
        const l = ordenar(tags, S.ordem), resto = l.length - S.limite;
        p('todas').innerHTML = l.slice(0, S.limite).map(chip).join('');
        p('mais').hidden = resto <= 0;
        p('mais').textContent = resto > 0 ? 'Mostrar mais ' + Math.min(pagina, resto) + ' de ' + resto + ' restantes' : '';
      }
    }
    p('busca').addEventListener('input', (e) => { S.busca = e.target.value; render(); });
    p('busca').addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); e.target.blur(); } }); // Enter fecha o teclado
    p('ordem').addEventListener('change', (e) => { S.ordem = e.target.value; S.limite = pagina; render(); });
    el.addEventListener('click', (e) => {
      const b = e.target.closest('button');
      if (!b || !el.contains(b)) return;
      if (b.dataset.del !== undefined) { op.aoExcluir(b.dataset.del); return; }
      if (b.dataset.p === 'todasBtn') { S.abertas = !S.abertas; render(); }
      else if (b.dataset.p === 'mais') { S.limite += pagina; render(); }
    });
    render();
    return { render }; // redesenha sem perder a busca, a ordem nem o "Ver todas" aberto
  }

  const api = { MAX_BUSCA, ORDENS, norm, buscar, recentes, ordenar, destacar, criar };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else raiz.ListaTags = api;
})(typeof window !== 'undefined' ? window : globalThis);
