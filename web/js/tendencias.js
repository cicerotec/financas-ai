// Tela Tendências: gasto por mês filtrado por tags, com grupos (E e NÃO dentro do grupo, OU entre grupos) e, sob demanda,
// uma expressão livre (parênteses, E, OU, NÃO). A barra do mês mostra quanto do valor tem outras tags além das da consulta.
// Layout (demo .ai/demos/demo-tendencias-layout.html): no computador (>= 1160 px) o resultado fica à esquerda e as escolhas à direita;
// no celular o resultado vem primeiro e as escolhas abrem em tela cheia por "Editar consulta" (o modo avançado abre por cima).
// Nada de lista de tags ou de combos solta na página: tudo abre numa janela (grupos, combos e "+ Tag" do modo avançado).
// O seletor Gastos | Receitas | Resultado (padrão: Gastos) escolhe o que somar; Resultado é entradas menos saídas das tags escolhidas.
// Isto é uma SIMULAÇÃO (a faixa no topo diz): nunca vale como saldo (o saldo real é o do Histórico) e nunca grava nada. Ocultos ficam de fora.
// Lógica pura e testada em web/consulta-tags.js. O front só desenha e soma o que já está na tela: nenhum número daqui vai para a API.
// Carregado depois do app.js e usa os globais dele (lancamentosCache, tagsUsadas, ehStatusReal, combosAtuaisLista, esc, fmtMoeda).
(function () {
  const CT = window.ConsultaTags;
  const ten = { modo: 'grupos', grupos: [CT.grupoVazio()], fichas: [], pos: 0 };
  let jan = null;        // janela de tags dos grupos: {g, campo}
  let desf = null;       // estado anterior, para o Desfazer do "Tirar"
  let avisoTxt = '', avisoRuim = false;
  let ultimo = null;     // último resumo de uma consulta válida
  let montado = false, mesesAntes = 0, rolarFim = true;
  let edAberto = false;  // celular: tela cheia de escolhas aberta
  let visao = 'gastos';  // o que somar: 'gastos' (saídas), 'receitas' (entradas) ou 'saldo' (renda menos gasto). Volta a Gastos ao recarregar.
  const $ = (id) => document.getElementById(id);
  const brl0 = (n) => n.toLocaleString('pt-BR', { maximumFractionDigits: 0 });
  const mq = window.matchMedia('(min-width:1160px)');
  const largo = () => mq.matches;

  function montar() {
    $('tendRaiz').innerHTML =
      '<div class="td-layout">' +
      // ---- resultado: esquerda no computador, primeiro no celular ----
      '<div class="td-res">' +
        '<div class="td-card td-topo">' +
          '<span class="td-pill" tabindex="0" title="Os números aqui servem para explorar: não são o seu saldo e nada é gravado. Lançamentos ocultos ficam de fora, como no Histórico.">Simulação</span>' +
          '<div class="td-seg" id="tdSeg" role="radiogroup" aria-label="O que somar">' +
            '<button type="button" role="radio" aria-checked="true" data-v="gastos">Gastos</button>' +
            '<button type="button" role="radio" aria-checked="false" data-v="receitas">Receitas</button>' +
            '<button type="button" role="radio" aria-checked="false" data-v="saldo">Resultado</button></div>' +
          '<div class="td-consulta"><span class="td-cons-rot">Consulta:</span><div class="td-frase" id="tdFrase"></div></div>' +
          '<button class="td-editar td-so-cel" id="tdEditar" type="button">Editar consulta</button></div>' +
        '<div class="td-card">' +
          '<div class="td-rtop"><span class="td-rtx" id="tdRtx"></span><span class="td-rtot" id="tdRtot"></span></div>' +
          '<div class="td-graf" id="tdGraf"></div><div class="td-leg" id="tdLeg"></div><div class="td-alerta" id="tdAlerta"></div>' +
          '<p class="td-nota-sim" id="tdSegNota"></p></div>' +
        '<div class="td-card"><div class="td-tit">Outras tags nesses lançamentos</div><div id="tdOutras"></div>' +
          '<div class="td-aviso" id="tdAviso" hidden></div></div>' +
      '</div>' +
      // ---- escolhas: direita no computador, tela cheia no celular ----
      '<div class="td-esc" id="tdEsc">' +
        '<div class="td-cab td-so-cel"><h2>Editar consulta</h2><button type="button" id="tdFechar" aria-label="Fechar">✕</button></div>' +
        '<button class="td-avanc" id="tdCombos" type="button" aria-haspopup="dialog" hidden>' +
          '<span><b>Combos</b><span class="td-sub" id="tdCombosSub"></span></span><span class="td-seta" aria-hidden="true">›</span></button>' +
        '<div id="tdGrupos">' +
          '<div id="tdLista"></div>' +
          '<button class="td-novo" id="tdNovo" type="button"></button>' +
          '<button class="td-avanc" id="tdAvanc" type="button" aria-controls="tdExpr">' +
            '<span><b>Modo avançado</b><span class="td-sub">Escrever a expressão com parênteses, E, OU e NÃO</span></span><span class="td-seta" aria-hidden="true">›</span></button>' +
        '</div>' +
        '<div id="tdExpr" hidden>' +
          '<div class="td-cab td-so-cel"><button type="button" id="tdVoltarCel">‹ Grupos</button><h2>Expressão</h2></div>' +
          '<div class="td-volta td-so-pc"><button type="button" id="tdVoltar">‹ Voltar aos grupos</button><span>Descarta a expressão</span></div>' +
          '<div class="td-card"><div class="td-tit">Gasto por mês de (expressão)</div>' +
            '<div class="td-expr" id="tdFichas" role="group" aria-label="Expressão"></div>' +
            '<div class="td-msg" id="tdMsg" role="status"></div><div class="td-lido" id="tdLido"></div>' +
            '<button type="button" class="td-btn-tag" id="tdBtnTag" aria-haspopup="dialog">+ Tag</button>' +
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
            '<p class="td-pe" id="tdCont"></p></div>' +
        '</div>' +
      '</div></div>' +
      // ---- barra de baixo (celular, com a tela de escolhas aberta) ----
      '<div class="td-rodape" id="tdRodape"><div class="td-rs" id="tdRs"></div><button type="button" id="tdPronto">Pronto</button></div>' +
      // ---- janela para escolher as tags de um grupo ----
      '<div class="td-veu" id="tdVeu" hidden><div class="td-jan" role="dialog" aria-modal="true" aria-labelledby="tdJt">' +
        '<h2 id="tdJt"></h2><p class="td-jsub" id="tdJs"></p>' +
        '<input type="search" class="td-jbusca" id="tdJb" placeholder="Buscar tag..." aria-label="Buscar tag">' +
        '<div class="td-jlista"><div class="td-tgs" id="tdJl" role="group"></div></div>' +
        '<p class="td-jcnt" id="tdJc"></p><p class="td-jdica" id="tdJd"></p>' +
        '<button class="td-jok" id="tdJok" type="button">Pronto</button></div></div>';
    $('tendRaiz').addEventListener('click', aoClicar);
    $('tdJb').addEventListener('input', () => { if (jan) renderJanela(); });
    document.addEventListener('keydown', (e) => {
      if (e.key !== 'Escape') return;
      if (jan) fecharJanela(); else if (edAberto && !largo()) { edAberto = false; render(); }
    });
    mq.addEventListener('change', () => { rolarFim = true; render(); });
    // a aba pode estar escondida quando o gráfico é desenhado: rola até os meses mais recentes assim que ele ganha largura
    if (window.ResizeObserver) new ResizeObserver(() => { const g = $('tdGraf'); if (rolarFim && g.clientWidth > 0) { g.scrollLeft = g.scrollWidth; rolarFim = false; } }).observe($('tdGraf'));
    montado = true;
  }

  // ---------- dados ----------
  // A base é de CT.base: sem ocultos (como o Histórico), só status real, entrada = tudo que não é saída. Quem escolhe o que somar
  // (gastos, receitas ou resultado) é CT.resumir, pela visão.
  function itens() { return CT.base(lancamentosCache, ehStatusReal); }
  function limparTagsInexistentes() { // tag excluída ou migrada sai da consulta (só depois que a lista de tags chegou)
    if (!tagsUsadas.length) return;
    const ok = (t) => tagsUsadas.includes(t);
    ten.grupos.forEach((g) => { g.tem = g.tem.filter(ok); g.nao = g.nao.filter(ok); });
    const antes = ten.fichas.length;
    ten.fichas = ten.fichas.filter((x) => x.k !== 'tag' || ok(x.v));
    if (ten.fichas.length !== antes) ten.pos = Math.min(ten.pos, ten.fichas.length);
  }

  // ---------- consulta (frase no topo do resultado) ----------
  function fraseHtml(g) {
    let h = g.tem.map(esc).join(' <span class="td-op e">E</span> ');
    if (g.nao.length) h += (g.tem.length ? ' ' : '') + g.nao.map((t) => '<span class="td-op nao">NÃO</span> ' + esc(t)).join(' ');
    return h;
  }
  function renderFrase(q) {
    let h;
    if (ten.modo === 'grupos') {
      const a = ten.grupos.filter(CT.grupoPreenchido);
      h = !a.length ? '<span class="td-vazio">Nenhuma tag escolhida: mostra tudo.</span>'
        : a.length === 1 ? fraseHtml(a[0]) : a.map((g) => '(' + fraseHtml(g) + ')').join(' <span class="td-op ou">OU</span> ');
    } else if (!q.valida) h = '<span class="td-vazio">Expressão com erro: corrija no modo avançado.</span>';
    else h = q.no ? esc(CT.texto(q.no)) : '<span class="td-vazio">Nenhuma tag escolhida: mostra tudo.</span>';
    $('tdFrase').innerHTML = h;
    $('tdFrase').title = $('tdFrase').textContent; // a linha corta com reticências quando é longa: o texto inteiro fica no title
  }

  // ---------- modo grupos ----------
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
  // combo em uso: o único grupo, sem NÃO, com as mesmas tags do combo
  const comboEmUso = () => (ten.modo === 'grupos' && ten.grupos.length === 1 && !ten.grupos[0].nao.length ? chaveTags(ten.grupos[0].tem) : null);
  function renderCombos() {
    const cs = combosAtuaisLista(), b = $('tdCombos');
    b.hidden = !cs.length;
    const u = comboEmUso(), c = u ? cs.find((x) => chaveTags(x.tags) === u) : null;
    $('tdCombosSub').textContent = c ? 'Em uso: ' + c.tags.join(' · ') : cs.length + (cs.length === 1 ? ' combo' : ' combos') + '. Toque para escolher.';
  }

  // janela de tags (grupos): as tags em botões que quebram em várias linhas, como no modo avançado
  function abrirJanela(g, campo) {
    jan = { tipo: 'tags', g, campo }; $('tdJb').value = '';
    $('tdJt').textContent = campo === 'tem' ? 'Tags que o lançamento tem' : 'Tags que ele não pode ter';
    $('tdJs').textContent = 'Grupo ' + (g + 1) + '. ' + (campo === 'tem' ? 'Precisa ter todas as marcadas.' : 'Fica de fora se tiver qualquer uma das marcadas.');
    $('tdVeu').hidden = false; travar(); renderJanela(); $('tdJb').focus();
  }
  function abrirCombos() {
    jan = { tipo: 'combos' }; $('tdJb').value = '';
    $('tdJt').textContent = 'Combos'; $('tdJs').textContent = 'Toque num combo para usá-lo. Tocar de novo no que está em uso desmarca.';
    $('tdVeu').hidden = false; travar(); renderJanela();
  }
  function abrirInserir() {
    jan = { tipo: 'inserir' }; $('tdJb').value = '';
    $('tdJt').textContent = 'Inserir tag'; $('tdJs').textContent = 'A tag entra onde está o cursor da expressão.';
    $('tdVeu').hidden = false; travar(); renderJanela(); $('tdJb').focus();
  }
  function renderJanela() {
    $('tdJb').style.display = jan.tipo === 'combos' ? 'none' : '';
    $('tdJok').textContent = jan.tipo === 'tags' ? 'Pronto' : 'Fechar';
    if (jan.tipo === 'combos') {
      const u = comboEmUso();
      $('tdJl').className = 'td-tgs';
      $('tdJl').innerHTML = combosAtuaisLista().map((c, i) => '<button type="button" class="' + (u && chaveTags(c.tags) === u ? 'on' : '') + '" data-combo="' + i + '" style="' + (c.salvo ? '' : 'border-style:dashed;') + '">' +
        esc(c.tags.join(' · ')) + (c.n ? ' <span class="td-n">' + c.n + 'x</span>' : '') + '</button>').join('');
      $('tdJc').textContent = 'Tracejado = sugerido pelo histórico.'; $('tdJd').textContent = '';
      return;
    }
    if (jan.tipo === 'inserir') {
      const q = normTxt($('tdJb').value.trim());
      $('tdJl').className = 'td-tgs';
      $('tdJl').innerHTML = tagsUsadas.filter((t) => !q || normTxt(t).includes(q)).map((t) => '<button type="button" data-tag="' + esc(t) + '">' + esc(t) + '</button>').join('')
        || '<p class="td-vazio-g">Nenhuma tag encontrada.</p>';
      $('tdJc').textContent = ''; $('tdJd').textContent = '';
      return;
    }
    const g = ten.grupos[jan.g], arr = g[jan.campo], outro = g[jan.campo === 'tem' ? 'nao' : 'tem'];
    const q = normTxt($('tdJb').value.trim());
    $('tdJl').className = 'td-tgs' + (jan.campo === 'nao' ? ' nao' : '');
    $('tdJl').innerHTML = tagsUsadas.filter((t) => !q || normTxt(t).includes(q)).map((t) => {
      const on = arr.includes(t), bloq = outro.includes(t);
      return '<button type="button" class="' + (on ? 'on' : '') + (bloq ? ' bloq' : '') + '" role="checkbox" aria-checked="' + on + '" data-t="' + esc(t) + '"' + (bloq ? ' aria-disabled="true"' : '') + '>' + (on ? '✓ ' : '') + esc(t) + '</button>';
    }).join('') || '<p class="td-vazio-g">Nenhuma tag encontrada.</p>';
    $('tdJc').textContent = arr.length + ' de ' + CT.MAX_TAGS + ' tags' + (arr.length >= CT.MAX_TAGS ? ' (limite atingido: desmarque uma para trocar)' : '');
    $('tdJd').textContent = outro.length ? 'As tags em cinza já estão em "' + (jan.campo === 'tem' ? 'Não pode ter' : 'Precisa ter') + '" deste grupo.' : '';
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
  // ---------- resultado ----------
  const NOME_VISAO = { gastos: 'Gastos', receitas: 'Receitas', saldo: 'Resultado' };
  const NOTA_VISAO = { gastos: 'Soma das saídas.', receitas: 'Soma das entradas.', saldo: 'Entradas menos saídas das tags escolhidas. Não é o saldo da conta.' };
  const sinal = (v, vis) => (vis === 'gastos' ? '-' : vis === 'receitas' ? '+' : v < 0 ? '-' : v > 0 ? '+' : '');
  const cents = (x) => Math.round(x * 100) / 100 + 0;
  function renderResultado(valida) {
    const a = ultimo, vis = a.visao;
    $('tdSegNota').innerHTML = '<b>Simulação:</b> não é o seu saldo, nada é gravado e ocultos ficam de fora, como no Histórico. ' + esc(NOTA_VISAO[vis]);
    document.querySelectorAll('#tdSeg [data-v]').forEach((b) => b.setAttribute('aria-checked', b.dataset.v === vis));
    $('tdRtx').textContent = a.n + ' lançamento(s)' + (valida ? '' : ' (último resultado válido)');
    const rt = $('tdRtot');
    rt.textContent = a.n ? sinal(a.total, vis) + fmtMoeda(Math.abs(a.total)) : '';
    rt.className = 'td-rtot' + (vis === 'receitas' || (vis === 'saldo' && a.total > 0) ? ' pos' : vis === 'saldo' && a.total === 0 ? ' zero' : '');
    const g = $('tdGraf');
    const rolada = g.scrollLeft;
    if (a.meses.length !== mesesAntes) rolarFim = true;
    g.classList.toggle('velho', !valida);
    g.setAttribute('role', 'img');
    // cada barra tem uma parte acima e outra abaixo do zero (só o saldo desce); "só as tags da consulta" + "tem outras tags" somam o mês
    const sol = (m) => cents(m.total - m.ext);
    const up = (m) => Math.max(sol(m), 0) + Math.max(m.ext, 0), dn = (m) => Math.max(-sol(m), 0) + Math.max(-m.ext, 0);
    const maxUp = Math.max(0, ...a.meses.map(up)), maxDn = Math.max(0, ...a.meses.map(dn));
    const alt = largo() ? 170 : 110, k = alt / Math.max(maxUp + maxDn, 0.01);
    const cor = (v) => (vis === 'saldo' && v < 0 ? 'var(--danger)' : 'var(--accent)');
    const rot = (v) => (vis === 'saldo' ? (v < 0 ? '-' : v > 0 ? '+' : '') : '') + brl0(Math.abs(v));
    g.setAttribute('aria-label', a.n ? NOME_VISAO[vis] + ' por mês: ' + a.meses.map((m) => m.mes.slice(5) + '/' + m.mes.slice(2, 4) + ' ' + (vis === 'saldo' ? sinal(m.total, vis) : '') + fmtMoeda(Math.abs(m.total)) +
      (m.ext ? ', sendo ' + fmtMoeda(Math.abs(m.ext)) + ' com outras tags' : '')).join('; ') : 'Sem lançamentos com essa consulta');
    const peca = (v, tipo) => (v ? '<div class="' + tipo + '" style="height:' + Math.abs(v) * k + 'px;--c:' + cor(v) + '"></div>' : '');
    g.innerHTML = a.n ? '<div class="td-cols" style="min-width:' + a.meses.length * 46 + 'px">' + a.meses.map((m) => {
      const s2 = sol(m), e = m.ext, liq = m.total, rotulo = '<span class="v">' + rot(liq) + '</span>';
      return '<div class="td-col"><div class="pos" style="height:' + (maxUp * k + 16) + 'px">' + (liq >= 0 ? rotulo : '') + (e > 0 ? peca(e, 'bx') : '') + (s2 > 0 ? peca(s2, 'bs') : '') + '</div>' +
        '<div class="neg" style="height:' + (maxDn ? maxDn * k + 16 : 0) + 'px">' + (s2 < 0 ? peca(s2, 'bs') : '') + (e < 0 ? peca(e, 'bx') : '') + (liq < 0 ? rotulo : '') + '</div>' +
        '<span class="m">' + m.mes.slice(5) + '/' + m.mes.slice(2, 4) + '</span></div>';
    }).join('') + '</div>' : '<p class="td-vazio-g">Sem lançamentos com essa consulta ainda.</p>';
    if (rolarFim && g.clientWidth > 0) { g.scrollLeft = g.scrollWidth; rolarFim = false; } else if (!rolarFim) g.scrollLeft = rolada; // começa nos meses mais recentes
    mesesAntes = a.meses.length;

    $('tdLeg').innerHTML = a.vazia || !a.n ? '' : '<span><i class="s"></i>Só as tags da consulta</span><span><i class="x"></i>Tem outras tags além das da consulta</span>';
    // porcentagem de "outras tags": do valor (gastos e receitas) ou do movimento, entradas mais saídas sem sinal (saldo: valores que se cancelam não servem de base)
    const base = vis === 'saldo' ? a.vol : a.total, baseExt = vis === 'saldo' ? a.volExt : a.totExt;
    const pct = a.n && baseExt ? Math.round(baseExt / base * 100) : 0;
    const al = $('tdAlerta');
    if (a.vazia) { al.className = 'td-alerta ok'; al.textContent = 'Sem filtro: todos os lançamentos, com todas as tags.'; }
    else if (!a.n) { al.className = 'td-alerta ok'; al.textContent = 'Nenhum lançamento encontrado.'; }
    else if (!baseExt) { al.className = 'td-alerta ok'; al.innerHTML = '<b>Total puro:</b> nenhum lançamento tem outras tags além das da consulta.'; }
    else {
      al.className = 'td-alerta';
      al.innerHTML = vis === 'saldo'
        ? '<b>' + pct + '% do movimento (' + esc(fmtMoeda(baseExt)) + ' entre entradas e saídas)</b> está em lançamentos que têm <b>outras tags</b> além das da consulta. O resultado pode estar misturando assuntos.'
        : '<b>' + pct + '% do valor (' + esc(fmtMoeda(baseExt)) + ')</b> está em lançamentos que têm <b>outras tags</b> além das da consulta. O total pode estar misturando assuntos.';
    }
    // resumo ao vivo da barra de baixo (celular)
    $('tdRs').innerHTML = a.n + ' lançamento(s) · <b>' + (a.n ? sinal(a.total, vis) + esc(fmtMoeda(Math.abs(a.total))) : esc(fmtMoeda(0))) + '</b><br>' +
      (a.vazia ? 'Sem filtro' : !a.n ? 'Nenhum lançamento' : pct ? pct + '% com outras tags' : 'Total puro') + (valida ? '' : ' (último válido)');
  }
  function renderOutras() {
    const a = ultimo, el = $('tdOutras');
    if (a.vazia) el.innerHTML = '<p class="td-vazio-g">Escolha tags na consulta para ver o que mais aparece junto com elas.</p>';
    else if (!a.cot.length) el.innerHTML = '<p class="td-vazio-g">Nenhuma outra tag nos lançamentos desta consulta.</p>';
    else {
      const top = a.cot.slice(0, CT.MAX_OUTRAS), resto = a.cot.length - top.length;
      el.innerHTML = '<p class="td-expl">Porcentagem do ' + (a.visao === 'saldo' ? 'movimento' : 'valor') + ' da consulta que está em lançamentos com a tag. Um lançamento com várias tags conta em cada uma, então não soma 100%.</p>' +
        top.map((c) => {
          const p = Math.round(c.v / (a.visao === 'saldo' ? a.vol : a.total) * 100);
          return '<div class="td-cot"><div class="info"><div class="nm"><b>' + esc(c.t) + '</b><span>' + p + '%</span></div><div class="trilho"><i style="width:' + p + '%"></i></div></div>' +
            '<button type="button" data-cot="' + esc(c.t) + '" aria-label="Tirar da consulta os lançamentos com ' + esc(c.t) + '">Tirar (NÃO)</button></div>';
        }).join('') + (resto > 0 ? '<p class="td-pe">e mais ' + resto + ' tag' + (resto > 1 ? 's' : '') + ' com peso menor.</p>' : '');
    }
    const av = $('tdAviso');
    av.hidden = !avisoTxt;
    av.className = 'td-aviso' + (avisoRuim ? ' bad' : '');
    av.innerHTML = avisoTxt ? '<span>' + esc(avisoTxt) + '</span>' + (desf && !avisoRuim ? '<button type="button" id="tdDesf">Desfazer</button>' : '') : '';
  }

  // celular: a tela de escolhas e a janela de tags travam a rolagem da página
  function travar() {
    const celular = !largo() && edAberto;
    document.body.classList.toggle('td-edit', celular);
    document.body.classList.toggle('td-trava', celular || !$('tdVeu').hidden);
    $('tdEsc').classList.toggle('aberto', edAberto);
  }

  function render() {
    if (!$('tendRaiz')) return;
    if (!montado) montar();
    limparTagsInexistentes();
    const q = CT.consulta(ten);
    if (q.valida) ultimo = CT.resumir(itens(), q, visao);
    else if (!ultimo || ultimo.visao !== visao) ultimo = CT.resumir(itens(), CT.consulta({ modo: 'grupos', grupos: [], fichas: [] }), visao);
    $('tdGrupos').hidden = ten.modo !== 'grupos';
    $('tdExpr').hidden = ten.modo !== 'avancado';
    renderCombos(); renderFrase(q);
    if (ten.modo === 'grupos') renderGrupos(); else { renderFichas(); renderMsgExpr(q); }
    renderResultado(q.valida); renderOutras(); travar();
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
    const t = e.target.closest('[data-v],[data-combo],#tdCombos,#tdBtnTag,[data-add],[data-rm],[data-del],#tdNovo,[data-t],#tdJok,[data-i],[data-k],[data-tag],#tdAvanc,#tdVoltar,#tdVoltarCel,[data-cot],#tdDesf,#tdEditar,#tdFechar,#tdPronto');
    if (!t) { if (e.target === $('tdVeu')) fecharJanela(); return; }
    if (t.dataset.v) { visao = t.dataset.v; return render(); }
    if (t.id === 'tdCombos') return abrirCombos();
    if (t.id === 'tdBtnTag') return abrirInserir();
    if (t.id === 'tdEditar') { edAberto = true; render(); const f = $('tdFechar'); if (f) f.focus(); return; }
    if (t.id === 'tdFechar' || t.id === 'tdPronto') { edAberto = false; render(); const b = $('tdEditar'); if (b && !largo()) b.focus(); return; }
    if (t.id === 'tdAvanc') { limparAviso(); ten.fichas = CT.gruposParaFichas(ten.grupos); ten.pos = ten.fichas.length; ten.modo = 'avancado'; return render(); }
    if (t.id === 'tdVoltar' || t.id === 'tdVoltarCel') { limparAviso(); ten.modo = 'grupos'; return render(); }
    if (t.id === 'tdDesf') { if (desf) Object.assign(ten, desf); limparAviso(); return render(); }
    if (t.dataset.cot) return tirar(t.dataset.cot);
    if (t.dataset.combo !== undefined) {
      const c = combosAtuaisLista()[+t.dataset.combo];
      if (!c) return;
      limparAviso();
      $('tdVeu').hidden = true; jan = null; // escolher um combo fecha a janela
      const mesmo = comboEmUso() === chaveTags(c.tags);
      const tags = mesmo ? [] : c.tags.slice(0, CT.MAX_TAGS); // tocar de novo no combo desmarca; o limite de 8 tags vale também para combo
      ten.grupos = [{ tem: tags, nao: [] }];
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
      if (t.getAttribute('aria-disabled') === 'true') return;
      const g = ten.grupos[jan.g], arr = g[jan.campo], tag = t.dataset.t, k = arr.indexOf(tag);
      if (k >= 0) arr.splice(k, 1); else if (arr.length < CT.MAX_TAGS) arr.push(tag);
      limparAviso(); renderJanela(); return render();
    }
    if (t.dataset.i !== undefined) { ten.pos = +t.dataset.i + 1; return render(); }
    if (t.dataset.tag !== undefined) { limparAviso(); $('tdVeu').hidden = true; jan = null; return inserir({ k: 'tag', v: t.dataset.tag }); } // a tag entra no cursor e a janela fecha
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
