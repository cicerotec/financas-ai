// Tela Tendências: gasto por mês filtrado por tags, com grupos (E e NÃO dentro do grupo, OU entre grupos) e, sob demanda,
// uma expressão livre (parênteses, E, OU, NÃO). A barra do mês mostra quanto do valor tem outras tags além das da consulta.
// Demo: .ai/demos/demo-tendencias-tags.html. Lógica pura e testada em web/consulta-tags.js.
// O front só desenha e soma o que já está na tela: nenhum número daqui vai para a API.
// Carregado depois do app.js e usa os globais dele (lancamentosCache, tagsUsadas, ehStatusReal, combosAtuaisLista, esc, fmtMoeda).
(function () {
  const CT = window.ConsultaTags;
  const ten = { modo: 'grupos', grupos: [CT.grupoVazio()], fichas: [], pos: 0 };
  let jan = null;        // janela de tags dos grupos: {g, campo}
  let desf = null;       // estado anterior, para o Desfazer do "Tirar"
  let avisoTxt = '', avisoRuim = false;
  let ultimo = null;     // último resumo de uma consulta válida
  let montado = false, mesesAntes = 0, rolarFim = true;
  const $ = (id) => document.getElementById(id);
  const brl0 = (n) => n.toLocaleString('pt-BR', { maximumFractionDigits: 0 });

  function montar() {
    $('tendRaiz').innerHTML =
      '<div class="td-card" id="tdCombosCard" hidden><div class="td-tit">Combos (toque para usar)</div><div class="td-ex" id="tdCombos"></div></div>' +
      '<div id="tdGrupos">' +
        '<div class="td-card"><div class="td-tit">Sua consulta</div><div class="td-frase" id="tdFrase"></div></div>' +
        '<div id="tdLista"></div>' +
        '<button class="td-novo" id="tdNovo" type="button"></button>' +
        '<button class="td-avanc" id="tdAvanc" type="button" aria-expanded="false" aria-controls="tdExpr">' +
          '<span><b>Modo avançado</b><span class="td-sub">Escrever a expressão com parênteses, E, OU e NÃO</span></span><span class="td-seta" aria-hidden="true">›</span></button>' +
      '</div>' +
      '<div id="tdExpr" hidden>' +
        '<div class="td-volta"><button type="button" id="tdVoltar">‹ Voltar aos grupos</button><span>Descarta a expressão</span></div>' +
        '<div class="td-card"><div class="td-tit">Gasto por mês de (expressão)</div>' +
          '<div class="td-expr" id="tdFichas" role="group" aria-label="Expressão"></div>' +
          '<div class="td-msg" id="tdMsg" role="status"></div><div class="td-lido" id="tdLido"></div>' +
          '<div class="td-teclas">' +
            '<button type="button" class="k-par" data-k="(" aria-label="Abre parêntese">(</button>' +
            '<button type="button" class="k-par" data-k=")" aria-label="Fecha parêntese">)</button>' +
            '<button type="button" class="k-e" data-k="E">E</button><button type="button" data-k="OU">OU</button>' +
            '<button type="button" class="k-nao" data-k="NAO">NÃO</button></div>' +
          '<div class="td-teclas l2">' +
            '<button type="button" class="k-x" data-k="esq" aria-label="Cursor para a esquerda">◀</button>' +
            '<button type="button" class="k-x" data-k="dir" aria-label="Cursor para a direita">▶</button>' +
            '<button type="button" class="k-x" data-k="apagar" aria-label="Apagar a ficha antes do cursor">⌫ Apagar</button>' +
            '<button type="button" class="k-x" data-k="limpar">Limpar</button></div>' +
          '<p class="td-pe" id="tdCont"></p>' +
          '<input type="search" class="td-busca" id="tdBusca" placeholder="Buscar tag..." aria-label="Buscar tag">' +
          '<div class="td-tgs" id="tdTgs"></div></div>' +
      '</div>' +
      '<div class="td-card"><div class="td-rtop"><span class="td-rtx" id="tdRtx"></span><span class="td-rtot" id="tdRtot"></span></div>' +
        '<div class="td-graf" id="tdGraf"></div><div class="td-leg" id="tdLeg"></div><div class="td-alerta" id="tdAlerta"></div></div>' +
      '<div class="td-card"><div class="td-tit">Outras tags nesses lançamentos</div><div id="tdOutras"></div>' +
        '<div class="td-aviso" id="tdAviso" hidden></div></div>' +
      '<div class="td-veu" id="tdVeu" hidden><div class="td-jan" role="dialog" aria-modal="true" aria-labelledby="tdJt">' +
        '<h2 id="tdJt"></h2><p class="td-jsub" id="tdJs"></p>' +
        '<input type="search" class="td-jbusca" id="tdJb" placeholder="Buscar tag..." aria-label="Buscar tag">' +
        '<div class="td-jlista" id="tdJl" role="group"></div><p class="td-jcnt" id="tdJc"></p>' +
        '<button class="td-jok" id="tdJok" type="button">Pronto</button></div></div>';
    $('tendRaiz').addEventListener('click', aoClicar);
    $('tdBusca').addEventListener('input', renderTags);
    $('tdJb').addEventListener('input', () => { if (jan) renderJanela(); });
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape' && jan) fecharJanela(); });
    // a aba pode estar escondida quando o gráfico é desenhado: rola até os meses mais recentes assim que ele ganha largura
    if (window.ResizeObserver) new ResizeObserver(() => { const g = $('tdGraf'); if (rolarFim && g.clientWidth > 0) { g.scrollLeft = g.scrollWidth; rolarFim = false; } }).observe($('tdGraf'));
    montado = true;
  }

  // ---------- dados ----------
  // Gasto = saída com status real. O que entra é o mesmo que o gráfico antigo somava.
  function itens() {
    return lancamentosCache.filter((l) => l.tipo === 'saida' && ehStatusReal(l.status)).map((l) => {
      const d = new Date(l.dataEvento);
      return { mes: d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0'), valor: l.valor, tags: l.tags || [] };
    });
  }
  function limparTagsInexistentes() { // tag excluída ou migrada sai da consulta (só depois que a lista de tags chegou)
    if (!tagsUsadas.length) return;
    const ok = (t) => tagsUsadas.includes(t);
    ten.grupos.forEach((g) => { g.tem = g.tem.filter(ok); g.nao = g.nao.filter(ok); });
    const antes = ten.fichas.length;
    ten.fichas = ten.fichas.filter((x) => x.k !== 'tag' || ok(x.v));
    if (ten.fichas.length !== antes) ten.pos = Math.min(ten.pos, ten.fichas.length);
  }

  // ---------- modo grupos ----------
  function fraseHtml(g) {
    let h = g.tem.map(esc).join(' <span class="td-op e">E</span> ');
    if (g.nao.length) h += (g.tem.length ? ' ' : '') + g.nao.map((t) => '<span class="td-op nao">NÃO</span> ' + esc(t)).join(' ');
    return h;
  }
  function renderFrase() {
    const a = ten.grupos.filter(CT.grupoPreenchido);
    $('tdFrase').innerHTML = !a.length ? '<span class="td-vazio">Nenhuma tag escolhida: mostra tudo.</span>'
      : a.length === 1 ? fraseHtml(a[0]) : a.map((g) => '(' + fraseHtml(g) + ')').join(' <span class="td-op ou">OU</span> ');
  }
  function chips(g, i, campo) {
    const arr = g[campo], cls = campo === 'nao' ? ' nao' : '';
    let h = arr.map((t, k) => '<span class="td-chip' + cls + '">' + esc(t) + '<button class="x" type="button" data-rm="' + i + ':' + campo + ':' + k + '" aria-label="Tirar ' + esc(t) + '">✕</button></span>').join('');
    h += arr.length < CT.MAX_TAGS ? '<button class="td-add" type="button" data-add="' + i + ':' + campo + '">+ tag</button>'
      : '<span class="td-lim">Limite de ' + CT.MAX_TAGS + ' tags.</span>';
    return h;
  }
  function renderGrupos() {
    $('tdLista').innerHTML = ten.grupos.map((g, i) =>
      (i ? '<div class="td-ou">OU</div>' : '') +
      '<div class="td-grupo"><div class="td-gtop"><b>Grupo ' + (i + 1) + '</b>' +
        (ten.grupos.length > 1 ? '<button class="rm" type="button" data-del="' + i + '" aria-label="Remover grupo ' + (i + 1) + '">✕</button>' : '') + '</div>' +
      '<div class="td-lin"><div class="td-rot"><span class="e">E</span> Precisa ter todas (' + g.tem.length + ' de ' + CT.MAX_TAGS + ')</div><div class="td-chips">' + chips(g, i, 'tem') + '</div></div>' +
      '<div class="td-lin"><div class="td-rot"><span class="n">NÃO</span> Não pode ter nenhuma (' + g.nao.length + ' de ' + CT.MAX_TAGS + ')</div><div class="td-chips">' + chips(g, i, 'nao') + '</div></div></div>').join('');
    const cheio = ten.grupos.length >= CT.MAX_GRUPOS, b = $('tdNovo');
    b.className = 'td-novo' + (cheio ? ' lim' : '');
    b.innerHTML = '<span>+ Outro grupo (OU)</span><span class="td-sub">' + (cheio ? 'Limite de ' + CT.MAX_GRUPOS + ' grupos. Remova um para adicionar outro.' : ten.grupos.length + ' de ' + CT.MAX_GRUPOS + ' grupos') + '</span>';
  }
  function renderCombos() {
    const cs = combosAtuaisLista();
    $('tdCombosCard').hidden = !cs.length;
    const unico = ten.modo === 'grupos' && ten.grupos.length === 1 && !ten.grupos[0].nao.length ? chaveTags(ten.grupos[0].tem) : null;
    $('tdCombos').innerHTML = cs.map((c, i) => '<button type="button" class="' + (unico && chaveTags(c.tags) === unico ? 'on' : '') + '" data-combo="' + i + '" style="' + (c.salvo ? '' : 'border-style:dashed;') + '">' +
      esc(c.tags.join(' · ')) + (c.n ? ' <span class="td-n">' + c.n + 'x</span>' : '') + '</button>').join('');
  }

  // janela de tags (grupos): lista que acumula, com check redondo
  function abrirJanela(g, campo) {
    jan = { g, campo }; $('tdJb').value = '';
    $('tdJt').textContent = campo === 'tem' ? 'Tags que o lançamento tem' : 'Tags que ele não pode ter';
    $('tdJs').textContent = 'Grupo ' + (g + 1) + '. ' + (campo === 'tem' ? 'Precisa ter todas as marcadas.' : 'Fica de fora se tiver qualquer uma das marcadas.');
    $('tdVeu').hidden = false; renderJanela(); $('tdJb').focus();
  }
  function renderJanela() {
    const g = ten.grupos[jan.g], arr = g[jan.campo], outro = g[jan.campo === 'tem' ? 'nao' : 'tem'];
    const q = normTxt($('tdJb').value.trim());
    $('tdJl').innerHTML = tagsUsadas.filter((t) => !q || normTxt(t).includes(q)).map((t) => {
      const on = arr.includes(t), bloq = outro.includes(t);
      return '<button class="td-li ' + (jan.campo === 'nao' ? 'nao' : '') + '" type="button" role="checkbox" aria-checked="' + on + '" data-t="' + esc(t) + '"' + (bloq ? ' aria-disabled="true" style="opacity:.5"' : '') + '>' +
        '<span class="n">' + esc(t) + (bloq ? '<span class="o">Já está em "' + (jan.campo === 'tem' ? 'Não pode ter' : 'Precisa ter') + '" deste grupo</span>' : '') + '</span><span class="td-ck">' + (on ? '✓' : '') + '</span></button>';
    }).join('') || '<p class="td-vazio-g">Nenhuma tag encontrada.</p>';
    $('tdJc').textContent = arr.length + ' de ' + CT.MAX_TAGS + ' tags' + (arr.length >= CT.MAX_TAGS ? ' (limite atingido: desmarque uma para trocar)' : '');
  }
  function fecharJanela() { $('tdVeu').hidden = true; jan = null; render(); }

  // ---------- modo avançado ----------
  function renderFichas() {
    const cur = '<span class="td-cur" aria-hidden="true"></span>';
    let h = '';
    if (!ten.fichas.length) h = cur + '<span class="td-vaz">Toque numa tag abaixo para começar</span>';
    else {
      ten.fichas.forEach((x, i) => {
        if (i === ten.pos) h += cur;
        const cls = x.k === 'tag' ? '' : x.k === 'op' ? ' op ' + (x.v === 'E' ? 'e' : x.v === 'OU' ? 'ou' : 'nao') : ' par';
        h += '<button type="button" class="td-tk' + cls + '" data-i="' + i + '" aria-label="' + esc(CT.nomeFicha(x)) + ', toque para pôr o cursor depois">' + esc(CT.nomeFicha(x)) + '</button>';
      });
      if (ten.pos === ten.fichas.length) h += cur;
    }
    $('tdFichas').innerHTML = h;
  }
  function renderMsgExpr(q) {
    $('tdFichas').classList.toggle('erro', !q.valida);
    $('tdMsg').className = 'td-msg ' + (q.valida ? 'ok' : 'bad');
    $('tdMsg').textContent = !q.valida ? q.msg : q.vazia ? 'Sem expressão: mostra todos os lançamentos.' : 'Expressão válida.';
    $('tdLido').textContent = q.valida && q.no ? 'Lido como: ' + CT.texto(q.no) : '';
    $('tdCont').textContent = ten.fichas.length + ' de ' + CT.MAX_FICHAS + ' fichas' + (ten.fichas.length >= CT.MAX_FICHAS ? ' (limite atingido: apague uma para inserir outra)' : '');
  }
  function renderTags() {
    const q = normTxt($('tdBusca').value.trim());
    $('tdTgs').innerHTML = tagsUsadas.filter((t) => !q || normTxt(t).includes(q)).map((t) => '<button type="button" data-tag="' + esc(t) + '">' + esc(t) + '</button>').join('')
      || '<span class="td-vazio">Nenhuma tag encontrada.</span>';
  }

  // ---------- resultado ----------
  function renderResultado(valida) {
    const a = ultimo;
    const max = Math.max(1, ...a.meses.map((m) => m.total));
    $('tdRtx').textContent = a.n + ' lançamento(s)' + (valida ? '' : ' (último resultado válido)');
    $('tdRtot').textContent = a.n ? '-' + fmtMoeda(a.total) : '';
    const g = $('tdGraf');
    const rolada = g.scrollLeft;
    if (a.meses.length !== mesesAntes) rolarFim = true;
    g.classList.toggle('velho', !valida);
    g.setAttribute('role', 'img');
    g.setAttribute('aria-label', a.n ? 'Gasto por mês: ' + a.meses.map((m) => m.mes.slice(5) + '/' + m.mes.slice(2, 4) + ' ' + fmtMoeda(m.total) + (m.ext ? ', sendo ' + fmtMoeda(m.ext) + ' com outras tags' : '')).join('; ') : 'Sem gastos com essa consulta');
    g.innerHTML = a.n ? '<div class="td-cols" style="min-width:' + a.meses.length * 46 + 'px">' + a.meses.map((m) => {
      const h = m.total / max * 118, hx = m.ext / max * 118;
      return '<div class="td-col"><span class="v">' + brl0(m.total) + '</span><div class="b" style="height:' + h + 'px"><div class="bx" style="height:' + hx + 'px"></div><div class="bs" style="height:' + Math.max(h - hx, 0) + 'px"></div></div><span class="m">' + m.mes.slice(5) + '/' + m.mes.slice(2, 4) + '</span></div>';
    }).join('') + '</div>' : '<p class="td-vazio-g">Sem gastos com essa consulta ainda.</p>';
    if (rolarFim && g.clientWidth > 0) { g.scrollLeft = g.scrollWidth; rolarFim = false; } else if (!rolarFim) g.scrollLeft = rolada; // começa nos meses mais recentes
    mesesAntes = a.meses.length;

    $('tdLeg').innerHTML = a.vazia || !a.n ? '' : '<span><i class="s"></i>Só as tags da consulta</span><span><i class="x"></i>Tem outras tags além das da consulta</span>';
    const al = $('tdAlerta');
    if (a.vazia) { al.className = 'td-alerta ok'; al.textContent = 'Sem filtro: todos os gastos, com todas as tags.'; }
    else if (!a.n) { al.className = 'td-alerta ok'; al.textContent = 'Nenhum lançamento encontrado.'; }
    else if (!a.totExt) { al.className = 'td-alerta ok'; al.innerHTML = '<b>Total puro:</b> nenhum lançamento tem outras tags além das da consulta.'; }
    else {
      const pct = Math.round(a.totExt / a.total * 100);
      al.className = 'td-alerta';
      al.innerHTML = '<b>' + pct + '% do valor (' + esc(fmtMoeda(a.totExt)) + ')</b> está em lançamentos que têm <b>outras tags</b> além das da consulta. O total pode estar misturando assuntos.';
    }
  }
  function renderOutras() {
    const a = ultimo, el = $('tdOutras');
    if (a.vazia) el.innerHTML = '<p class="td-vazio-g">Escolha tags na consulta para ver o que mais aparece junto com elas.</p>';
    else if (!a.cot.length) el.innerHTML = '<p class="td-vazio-g">Nenhuma outra tag nos lançamentos desta consulta.</p>';
    else {
      const top = a.cot.slice(0, CT.MAX_OUTRAS), resto = a.cot.length - top.length;
      el.innerHTML = '<p class="td-expl">Porcentagem do valor da consulta que está em lançamentos com a tag. Um lançamento com várias tags conta em cada uma, então não soma 100%.</p>' +
        top.map((c) => {
          const p = Math.round(c.v / a.total * 100);
          return '<div class="td-cot"><div class="info"><div class="nm"><b>' + esc(c.t) + '</b><span>' + p + '%</span></div><div class="trilho"><i style="width:' + p + '%"></i></div></div>' +
            '<button type="button" data-cot="' + esc(c.t) + '" aria-label="Tirar da consulta os lançamentos com ' + esc(c.t) + '">Tirar (NÃO)</button></div>';
        }).join('') + (resto > 0 ? '<p class="td-pe">e mais ' + resto + ' tag' + (resto > 1 ? 's' : '') + ' com peso menor.</p>' : '');
    }
    const av = $('tdAviso');
    av.hidden = !avisoTxt;
    av.className = 'td-aviso' + (avisoRuim ? ' bad' : '');
    av.innerHTML = avisoTxt ? '<span>' + esc(avisoTxt) + '</span>' + (desf && !avisoRuim ? '<button type="button" id="tdDesf">Desfazer</button>' : '') : '';
  }

  function render() {
    if (!$('tendRaiz')) return;
    if (!montado) { montar(); renderTags(); }
    limparTagsInexistentes();
    const q = CT.consulta(ten);
    if (q.valida) ultimo = CT.resumir(itens(), q);
    else if (!ultimo) ultimo = CT.resumir(itens(), CT.consulta({ modo: 'grupos', grupos: [], fichas: [] }));
    $('tdGrupos').hidden = ten.modo !== 'grupos';
    $('tdExpr').hidden = ten.modo !== 'avancado';
    $('tdAvanc').setAttribute('aria-expanded', ten.modo === 'avancado');
    renderCombos();
    if (ten.modo === 'grupos') { renderFrase(); renderGrupos(); } else { renderFichas(); renderMsgExpr(q); renderTags(); }
    renderResultado(q.valida); renderOutras();
  }

  // ---------- toques ----------
  function limparAviso() { avisoTxt = ''; desf = null; }
  function tirar(tag) {
    const r = CT.tirarTag(ten, tag);
    if (!r.ok) { avisoTxt = r.erro; avisoRuim = true; desf = null; return render(); }
    desf = { modo: ten.modo, grupos: JSON.parse(JSON.stringify(ten.grupos)), fichas: JSON.parse(JSON.stringify(ten.fichas)), pos: ten.pos };
    Object.assign(ten, { grupos: r.estado.grupos, fichas: r.estado.fichas, pos: r.estado.pos });
    avisoTxt = 'Adicionado: NÃO ' + tag + '.'; avisoRuim = false; render();
  }
  function inserir(x) {
    const r = CT.inserirFicha(ten.fichas, ten.pos, x);
    ten.fichas = r.fichas; ten.pos = r.pos; render();
  }
  function aoClicar(e) {
    const t = e.target.closest('[data-combo],[data-add],[data-rm],[data-del],#tdNovo,[data-t],#tdJok,[data-i],[data-k],[data-tag],#tdAvanc,#tdVoltar,[data-cot],#tdDesf');
    if (!t) { if (e.target === $('tdVeu')) fecharJanela(); return; }
    if (t.id === 'tdAvanc') { limparAviso(); ten.fichas = CT.gruposParaFichas(ten.grupos); ten.pos = ten.fichas.length; ten.modo = 'avancado'; return render(); }
    if (t.id === 'tdVoltar') { limparAviso(); ten.modo = 'grupos'; return render(); }
    if (t.id === 'tdDesf') { if (desf) Object.assign(ten, desf); limparAviso(); return render(); }
    if (t.dataset.cot) return tirar(t.dataset.cot);
    if (t.dataset.combo !== undefined) {
      const c = combosAtuaisLista()[+t.dataset.combo];
      if (!c) return;
      limparAviso();
      const mesmo = ten.modo === 'grupos' && ten.grupos.length === 1 && !ten.grupos[0].nao.length && chaveTags(ten.grupos[0].tem) === chaveTags(c.tags);
      const tags = mesmo ? [] : c.tags.slice(0, CT.MAX_TAGS); // tocar de novo no combo desmarca
      ten.grupos = [{ tem: tags, nao: [] }]; // o limite de 8 tags por lista vale também para combo
      ten.fichas = []; tags.forEach((tg, k) => { if (k) ten.fichas.push({ k: 'op', v: 'E' }); ten.fichas.push({ k: 'tag', v: tg }); });
      ten.pos = ten.fichas.length;
      return render();
    }
    if (t.dataset.add) { const [i, c] = t.dataset.add.split(':'); return abrirJanela(+i, c); }
    if (t.dataset.rm) { limparAviso(); const [i, c, k] = t.dataset.rm.split(':'); ten.grupos[+i][c].splice(+k, 1); return render(); }
    if (t.dataset.del) { limparAviso(); ten.grupos.splice(+t.dataset.del, 1); return render(); }
    if (t.id === 'tdNovo') { if (ten.grupos.length < CT.MAX_GRUPOS) { limparAviso(); ten.grupos.push(CT.grupoVazio()); render(); abrirJanela(ten.grupos.length - 1, 'tem'); } return; }
    if (t.id === 'tdJok') return fecharJanela();
    if (t.dataset.t) {
      const g = ten.grupos[jan.g], arr = g[jan.campo], tag = t.dataset.t, k = arr.indexOf(tag);
      if (t.getAttribute('aria-disabled') === 'true') return;
      if (k >= 0) arr.splice(k, 1); else if (arr.length < CT.MAX_TAGS) arr.push(tag);
      limparAviso(); renderJanela(); return render();
    }
    if (t.dataset.i !== undefined) { ten.pos = +t.dataset.i + 1; return render(); }
    if (t.dataset.tag !== undefined) { limparAviso(); return inserir({ k: 'tag', v: t.dataset.tag }); }
    const k = t.dataset.k;
    if (k) limparAviso();
    if (k === '(' || k === ')') return inserir({ k });
    if (k === 'E' || k === 'OU' || k === 'NAO') return inserir({ k: 'op', v: k });
    if (k === 'esq') { ten.pos = Math.max(0, ten.pos - 1); return render(); }
    if (k === 'dir') { ten.pos = Math.min(ten.fichas.length, ten.pos + 1); return render(); }
    if (k === 'apagar') { if (ten.pos > 0) { ten.fichas.splice(ten.pos - 1, 1); ten.pos--; render(); } return; }
    if (k === 'limpar') { ten.fichas = []; ten.pos = 0; render(); }
  }

  window.tendRender = render;
})();
