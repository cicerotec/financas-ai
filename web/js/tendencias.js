// Tela Tendências: gasto por mês filtrado por tags, com grupos (E e NÃO dentro do grupo, OU entre grupos) e, sob demanda,
// uma expressão livre (parênteses, E, OU, NÃO). A barra do mês mostra quanto do valor tem outras tags além das da consulta.
// Layout (demo .ai/demos/demo-tendencias-layout.html): no computador (>= 1160 px) o resultado fica à esquerda e as escolhas à direita;
// no celular o resultado vem primeiro e as escolhas abrem em tela cheia por "Editar consulta" (o modo avançado abre por cima).
// Nada de lista de tags ou de combos solta na página: tudo abre numa janela (grupos, combos e "+ Tag" do modo avançado).
// O seletor Gastos | Receitas | Resultado (padrão: Gastos) escolhe o que somar; Resultado é entradas menos saídas das tags escolhidas.
// Isto é uma SIMULAÇÃO (a faixa no topo diz): nunca vale como saldo (o saldo real é o do Histórico) e nunca grava nada. Ocultos ficam de fora.
// Consultas salvas (item 53, demo .ai/demos/demo-tendencias-consultas.html): "Consultas salvas" e "Salvar consulta" abrem janelas; as consultas são do espaço (todos usam; cada
// pessoa altera só as suas, o dono modera: quem decide é o servidor, que manda minha e podeAlterar). Só a consulta é guardada, nunca um número; a lista só é lida quando a janela abre.
// Lógica pura e testada em web/consulta-tags.js. O front só desenha e soma o que já está na tela: nenhum número daqui vai para a API.
// Carregado depois do app.js e usa os globais dele (lancamentosCache, tagsUsadas, combosAtuaisLista, esc, fmtMoeda).
(function () {
  const CT = window.ConsultaTags;
  const ten = { modo: 'grupos', grupos: [CT.grupoVazio()], fichas: [], pos: 0 };
  let jan = null;        // janela de tags dos grupos: {g, campo}
  let desf = null;       // estado anterior, para o Desfazer do "Tirar"
  let avisoTxt = '', avisoRuim = false;
  let ultimo = null;     // último resumo de uma consulta válida
  let montado = false, mesesAntes = 0, rolarFim = true;
  let edAberto = false;  // celular: tela cheia de escolhas aberta
  let listaAberta = false, listaMes = ''; // janela da lista dos lançamentos da consulta e o mês que ela mostra ('' = todos)
  const LIMITE_CARGA = 1000; // quantos lançamentos o app carrega (limit(1000) em app.js; o backend também limita a 1000)
  const MAX_LISTA = 200;  // linhas desenhadas na lista (o resto pede um mês)
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
          '<div class="td-orig" id="tdOrig" hidden></div>' +
          '<div class="td-cacoes"><button type="button" class="td-bt" id="tdBtnSalvas" aria-haspopup="dialog">Consultas salvas</button>' +
            '<button type="button" class="td-bt pri" id="tdBtnSalvar" aria-haspopup="dialog">Salvar consulta</button></div>' +
          '<button class="td-editar td-so-cel" id="tdEditar" type="button">Editar consulta</button></div>' +
        '<div class="td-card">' +
          '<div class="td-rtop"><button type="button" class="td-ver-lista" id="tdBtnLista" aria-haspopup="dialog"><span id="tdRtx"></span><span aria-hidden="true"> ›</span></button><span class="td-rtot" id="tdRtot"></span></div>' +
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
        '<button class="td-jok" id="tdJok" type="button">Pronto</button></div></div>' +
      // ---- janela da lista dos lançamentos da consulta (só leitura) ----
      '<div class="td-veu td-lista" id="tdListaVeu" hidden><div class="td-jan td-ljan" role="dialog" aria-modal="true" aria-labelledby="tdLt">' +
        '<div class="td-lcab"><h2 id="tdLt">Lançamentos da consulta</h2><button type="button" id="tdLFechar" aria-label="Fechar">✕</button></div>' +
        '<p class="td-lsub" id="tdLsub"></p><div class="td-lchips" id="tdLMeses" role="group" aria-label="Mês"></div>' +
        '<div class="td-lcorpo" id="tdLCorpo"></div><div class="td-lrodape" id="tdLRodape"></div></div></div>' +
      // ---- janela das consultas salvas (do espaço), a de salvar e a de renomear ----
      '<div class="td-veu" id="tdCsVeu" hidden><div class="td-jan" role="dialog" aria-modal="true" aria-labelledby="tdCsT">' +
        '<div class="td-lcab"><h2 id="tdCsT">Consultas salvas</h2><button type="button" id="tdCsFechar" aria-label="Fechar">✕</button></div>' +
        '<p class="td-jsub">Do espaço: todos usam. Cada pessoa altera só as suas; o dono altera todas.</p>' +
        '<input type="search" class="td-jbusca" id="tdCsBusca" placeholder="Buscar consulta..." aria-label="Buscar consulta" autocomplete="off">' +
        '<div class="td-lchips" id="tdCsFichas" role="group" aria-label="Filtrar"></div>' +
        '<div class="td-msg bad" id="tdCsErro" role="alert" hidden></div>' +
        '<div class="td-cs-lista" id="tdCsLista"></div>' +
        '<div class="td-cs-rod"><span id="tdCsCnt"></span><span class="td-cs-lim" id="tdCsLim"></span></div></div></div>' +
      '<div class="td-veu td-alto" id="tdSvVeu" hidden><div class="td-jan" role="dialog" aria-modal="true" aria-labelledby="tdSvT">' +
        '<div class="td-lcab"><h2 id="tdSvT">Salvar consulta</h2><button type="button" id="tdSvFechar" aria-label="Fechar">✕</button></div>' +
        '<p class="td-jsub" id="tdSvSub"></p>' +
        '<input type="text" class="td-cs-nome" id="tdSvNome" maxlength="' + CT.MAX_NOME_CONSULTA + '" placeholder="Nome da consulta" aria-label="Nome da consulta" autocomplete="off">' +
        '<div class="td-cs-cnt" id="tdSvCnt"></div><div class="td-msg bad" id="tdSvErro" role="alert"></div>' +
        '<button class="td-jok" id="tdSvOk" type="button">Salvar</button><button class="td-cs-sec" id="tdSvAtual" type="button" hidden></button></div></div>' +
      '<div class="td-veu td-alto" id="tdRnVeu" hidden><div class="td-jan" role="dialog" aria-modal="true" aria-labelledby="tdRnT">' +
        '<div class="td-lcab"><h2 id="tdRnT">Renomear</h2><button type="button" id="tdRnFechar" aria-label="Fechar">✕</button></div>' +
        '<p class="td-jsub" id="tdRnSub"></p>' +
        '<input type="text" class="td-cs-nome" id="tdRnNome" maxlength="' + CT.MAX_NOME_CONSULTA + '" aria-label="Novo nome" autocomplete="off">' +
        '<div class="td-cs-cnt" id="tdRnCnt"></div><div class="td-msg bad" id="tdRnErro" role="alert"></div>' +
        '<button class="td-jok" id="tdRnOk" type="button">Renomear</button></div></div>';
    $('tendRaiz').addEventListener('click', aoClicar);
    $('tdJb').addEventListener('input', () => { if (jan) renderJanela(); });
    $('tdCsBusca').addEventListener('input', () => { cs.busca = $('tdCsBusca').value; renderConsultas(); });
    $('tdSvNome').addEventListener('input', () => { $('tdSvCnt').textContent = $('tdSvNome').value.length + ' de ' + CT.MAX_NOME_CONSULTA; $('tdSvErro').textContent = ''; });
    $('tdSvNome').addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); salvarNova(); } });
    $('tdRnNome').addEventListener('input', () => { $('tdRnCnt').textContent = $('tdRnNome').value.length + ' de ' + CT.MAX_NOME_CONSULTA; $('tdRnErro').textContent = ''; });
    $('tdRnNome').addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); renomear(); } });
    document.addEventListener('keydown', (e) => {
      if (e.key !== 'Escape') return;
      if (cs.rn) { fecharRenomear(); return; }
      if (cs.sv) { fecharSalvar(); return; }
      if (cs.aberta) { fecharConsultas(); return; }
      if (listaAberta) { fecharLista(); return; }
      if (jan) fecharJanela(); else if (edAberto && !largo()) { edAberto = false; render(); }
    });
    $('tendRaiz').addEventListener('keydown', (e) => { // as barras do mês são botões: Enter e espaço abrem a lista daquele mês
      if ((e.key === 'Enter' || e.key === ' ') && e.target.matches && e.target.matches('.td-col[data-mes]')) { e.preventDefault(); abrirLista(e.target.dataset.mes); }
    });
    mq.addEventListener('change', () => { rolarFim = true; render(); });
    // a aba pode estar escondida quando o gráfico é desenhado: rola até os meses mais recentes assim que ele ganha largura
    if (window.ResizeObserver) new ResizeObserver(() => { const g = $('tdGraf'); if (rolarFim && g.clientWidth > 0) { g.scrollLeft = g.scrollWidth; rolarFim = false; } }).observe($('tdGraf'));
    montado = true;
  }

  // ---------- dados ----------
  // A base é de CT.base: sem ocultos (como o Histórico), só status real, entrada = tudo que não é saída. Quem escolhe o que somar
  // (gastos, receitas ou resultado) é CT.resumir, pela visão.
  function itens() { return CT.base(lancamentosCache); }
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
    const mesCorte = CT.corte(lancamentosCache.length, LIMITE_CARGA, a); // chegou ao limite: o mês mais antigo pode estar incompleto
    $('tdSegNota').innerHTML = '<b>Simulação:</b> não é o seu saldo, nada é gravado e ocultos ficam de fora, como no Histórico. ' + esc(NOTA_VISAO[vis]) +
      (mesCorte ? ' <span class="td-corte">O app carrega só os ' + LIMITE_CARGA + ' lançamentos mais recentes, então o mês mais antigo (' + esc(mesCorte.slice(5) + '/' + mesCorte.slice(2, 4)) + ') pode estar incompleto.</span>' : '');
    document.querySelectorAll('#tdSeg [data-v]').forEach((b) => b.setAttribute('aria-checked', b.dataset.v === vis));
    $('tdRtx').textContent = (a.n ? (a.n === 1 ? 'Ver o lançamento' : 'Ver os ' + a.n + ' lançamentos') : 'Nenhum lançamento') + (valida ? '' : ' (último resultado válido)');
    $('tdBtnLista').classList.toggle('sem', !a.n);
    const rt = $('tdRtot');
    rt.textContent = a.n ? sinal(a.total, vis) + fmtMoeda(Math.abs(a.total)) : '';
    rt.className = 'td-rtot' + (vis === 'receitas' || (vis === 'saldo' && a.total > 0) ? ' pos' : vis === 'saldo' && a.total === 0 ? ' zero' : '');
    const g = $('tdGraf');
    const rolada = g.scrollLeft;
    if (a.meses.length !== mesesAntes) rolarFim = true;
    g.classList.toggle('velho', !valida);
    g.setAttribute('role', 'group');
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
      const rm = m.mes.slice(5) + '/' + m.mes.slice(2, 4);
      return '<div class="td-col" data-mes="' + m.mes + '" role="button" tabindex="0" aria-label="Ver os lançamentos de ' + rm + '"><div class="pos" style="height:' + (maxUp * k + 16) + 'px">' + (liq >= 0 ? rotulo : '') + (e > 0 ? peca(e, 'bx') : '') + (s2 > 0 ? peca(s2, 'bs') : '') + '</div>' +
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
    document.body.classList.toggle('td-trava', celular || !$('tdVeu').hidden || listaAberta || cs.aberta || cs.sv || !!cs.rn);
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
    renderCombos(); renderFrase(q); renderBarraConsulta(q);
    if (ten.modo === 'grupos') renderGrupos(); else { renderFichas(); renderMsgExpr(q); }
    renderResultado(q.valida); renderOutras();
    if (listaAberta) { if (ultimo.n) renderLista(); else fecharLista(); }
    travar();
  }

  // ---------- lista dos lançamentos da consulta (só leitura) ----------
  // A Tendências é uma simulação: a lista não edita, não oculta e não exclui (para isso, o Histórico). Segue a visão e a base do
  // gráfico, e a soma do rodapé bate com ele (CT.listar faz a conta e diz se bate).
  function linhaLista(l) {
    const x = l.ref;
    return '<div class="td-lrow"><div class="ld"><b>' + esc(x.descricao || '(sem descrição)') + '</b><span class="lm">' + esc(dataBR(x)) + ' · ' + esc(x.status || '') + ' · ' + esc(x.banco || '-') +
      '</span><span class="ltg">' + esc((x.tags || []).map((t) => '#' + t).join(' ')) + '</span></div>' +
      '<span class="lv ' + (l.entrada ? 'pos' : 'neg') + '">' + (l.entrada ? '+' : '-') + esc(fmtMoeda(l.valor)) + '</span></div>';
  }
  function renderLista() {
    const a = ultimo, r = CT.listar(a, listaMes, MAX_LISTA);
    listaMes = r.mes;
    const textoSoma = (v) => sinal(v, a.visao) + fmtMoeda(Math.abs(v));
    $('tdLsub').textContent = 'Só leitura: aqui não dá para editar, ocultar nem excluir; para isso, use o Histórico. ' + NOTA_VISAO[a.visao] + ' Ocultos e futuros ficam de fora, como no gráfico.';
    $('tdLMeses').innerHTML = '<button type="button" class="' + (listaMes === '' ? 'on' : '') + '" data-lm="">Todos <span class="td-n">' + a.n + '</span></button>' +
      r.meses.map((m) => '<button type="button" class="' + (listaMes === m.mes ? 'on' : '') + '" data-lm="' + m.mes + '">' + m.mes.slice(5) + '/' + m.mes.slice(2, 4) + ' <span class="td-n">' + m.n + '</span></button>').join('');
    const ficha = $('tdLMeses').querySelector('.on'); // a ficha do mês escolhido fica à vista, mesmo com muitos meses
    if (ficha && ficha.scrollIntoView) ficha.scrollIntoView({ inline: 'center', block: 'nearest' });
    let h = '', mesAtual = '';
    if (!r.n) h = '<p class="td-vazio-g">Nenhum lançamento neste mês com essa consulta.</p>';
    r.linhas.forEach((l) => {
      if (!listaMes && l.mes !== mesAtual) { mesAtual = l.mes; h += '<div class="td-lmes"><span>' + esc(rotuloMes(l.mes)) + '</span><span>' + esc(textoSoma(r.somaMes[l.mes])) + '</span></div>'; }
      h += linhaLista(l);
    });
    if (r.corte) h += '<div class="td-laviso">Mostrando ' + MAX_LISTA + ' de ' + r.n + ' lançamentos. Escolha um mês acima para ver o resto.</div>';
    $('tdLCorpo').innerHTML = h;
    $('tdLRodape').innerHTML = '<span>' + r.n + ' lançamento(s) · soma <b>' + esc(textoSoma(r.soma)) + '</b></span>' +
      (r.bate ? '<span class="bate">✓ Bate com ' + (listaMes ? 'a barra de ' + listaMes.slice(5) + '/' + listaMes.slice(2, 4) : 'o gráfico') + '</span>' : '<span class="nbate">⚠ Não bate com o gráfico</span>');
  }
  function abrirLista(mes) {
    if (!ultimo || !ultimo.n) return;
    listaMes = mes || ''; listaAberta = true; $('tdListaVeu').hidden = false; renderLista(); $('tdLCorpo').scrollTop = 0; travar();
  }
  function fecharLista() { listaAberta = false; $('tdListaVeu').hidden = true; travar(); const b = $('tdBtnLista'); if (b) b.focus(); }

  // ---------- consultas salvas (item 53) ----------
  // A lista vem do servidor só quando a janela abre. Quem decide o autor, minha, podeAlterar, os limites e o conflito (409) é o servidor.
  const cs = { aberta: false, lista: null, carregando: false, erro: '', filtro: 'todas', busca: '', origem: null, base: null, sv: false, rn: null, ocupado: false };
  const dataCurta = (iso) => { const d = new Date(iso); return isNaN(d) ? '' : d.toLocaleDateString('pt-BR'); };
  const consultaDaTela = () => CT.paraSalvar(ten, visao);
  const achar = (id) => (cs.lista || []).find((c) => c.id === id) || null;
  const origemItem = () => (cs.origem ? achar(cs.origem) : null);
  const alterada = () => !!origemItem() && !CT.mesmaConsulta(consultaDaTela(), cs.base);
  const ordenar = () => { cs.lista.sort((a, b) => (a.criadoEm < b.criadoEm ? 1 : a.criadoEm > b.criadoEm ? -1 : 0)); };
  const substituir = (c) => { cs.lista = (cs.lista || []).map((x) => (x.id === c.id ? c : x)); };
  function mensagemDe(e) { // o texto do servidor (400, 403, 404), ou um genérico quando nem chegou lá (rede)
    const m = e && e.code && e.message ? e.message : 'Não consegui agora. Confira a conexão e tente de novo.';
    return m.charAt(0).toUpperCase() + m.slice(1) + (/[.!?]$/.test(m) ? '' : '.');
  }
  function erroDoNome(nome, exceto) {
    if (!nome) return 'Dê um nome para a consulta.';
    if ((cs.lista || []).some((c) => c.id !== exceto && c.nome.trim().toLowerCase() === nome.toLowerCase())) return 'Já existe uma consulta com esse nome.';
    return '';
  }
  function erroJanela(id, e) {
    if (e && e.codigo === 'conflito_versao') { // alguém gravou antes: a lista é relida e a pessoa confere
      $(id).textContent = 'Alguém alterou esta consulta antes de você. A lista foi atualizada: confira e tente de novo.';
      carregarConsultas();
    } else $(id).textContent = mensagemDe(e);
  }

  function renderBarraConsulta(q) {
    const o = origemItem(), el = $('tdOrig');
    el.hidden = !o;
    if (o) el.innerHTML = 'Carregada de <b>' + esc(o.nome) + '</b> (' + (o.minha ? 'sua' : 'de ' + esc(o.autor)) + ')' + (alterada() ? ' · <b>alterada, ainda não salva</b>' : '') + '.';
    $('tdBtnSalvar').disabled = !(q.valida && !q.vazia);
  }

  function linhaConsulta(c) {
    const a = CT.acoesDaConsulta(c), uso = cs.origem === c.id, id = esc(c.id);
    return '<div class="td-cs-item' + (uso ? ' em-uso' : '') + '"><div class="nm"><b>' + esc(c.nome) + '</b>' + (c.minha ? '<span class="tagu">Sua</span>' : '') + '</div>' +
      '<div class="fr" title="' + esc(CT.fraseSalva(c)) + '">' + esc(CT.fraseSalva(c)) + '</div>' +
      '<div class="por">' + esc(NOME_VISAO[c.visao] || '') + ' · por ' + (c.minha ? 'você' : esc(c.autor)) + ' em ' + dataCurta(c.criadoEm) + '</div>' +
      '<div class="ac"><button type="button" class="td-bt pri" data-cs-usar="' + id + '">' + (uso ? 'Em uso' : 'Usar') + '</button>' +
      (a.renomear ? '<button type="button" class="td-bt" data-cs-ren="' + id + '">Renomear</button>' : '') +
      (a.excluir ? '<button type="button" class="td-bt perigo" data-cs-del="' + id + '">Excluir</button>' : '') +
      (a.comoDono ? '<span class="so">como dono</span>' : '') + (a.motivo ? '<span class="so">' + esc(a.motivo) + '</span>' : '') + '</div></div>';
  }
  function renderConsultas() {
    const lista = cs.lista || [], r = CT.filtrarConsultas(lista, cs.filtro, cs.busca);
    $('tdCsFichas').innerHTML = ['todas', 'minhas'].map((f) => '<button type="button" class="' + (cs.filtro === f ? 'on' : '') + '" data-cs-f="' + f + '">' + (f === 'todas' ? 'Todas' : 'Minhas') + ' <span class="td-n">' + r[f] + '</span></button>').join('');
    const er = $('tdCsErro');
    er.hidden = !cs.erro;
    er.innerHTML = cs.erro ? esc(cs.erro) + ' <button type="button" class="td-cs-link" id="tdCsRetry">Tentar de novo</button>' : '';
    let h = '';
    if (!cs.lista) h = cs.erro ? '' : '<p class="td-vazio-g">Carregando...</p>';
    else if (!lista.length) h = '<div class="td-vazio-g">Nenhuma consulta salva ainda.<br>Monte uma consulta e toque em <b>Salvar consulta</b>.</div>';
    else if (!r.itens.length) h = '<p class="td-vazio-g">' + (cs.filtro === 'minhas' && !cs.busca.trim() ? 'Você ainda não salvou nenhuma consulta.' : 'Nenhuma consulta encontrada.') + '</p>';
    $('tdCsLista').innerHTML = h + r.itens.map(linhaConsulta).join('');
    $('tdCsCnt').textContent = cs.lista ? lista.length + ' de ' + CT.MAX_CONSULTAS + ' consultas' : '';
    $('tdCsLim').textContent = lista.length >= CT.MAX_CONSULTAS ? 'Limite atingido: exclua uma para salvar outra.' : '';
  }
  async function carregarConsultas() {
    cs.carregando = true; cs.erro = ''; renderConsultas();
    try { cs.lista = await db.consultas(); } catch (e) { cs.erro = 'Não consegui carregar as consultas agora.'; }
    cs.carregando = false; renderConsultas(); render();
  }
  function abrirConsultas() {
    cs.aberta = true; cs.erro = ''; $('tdCsBusca').value = cs.busca;
    $('tdCsVeu').hidden = false; travar(); renderConsultas(); carregarConsultas(); $('tdCsBusca').focus();
  }
  function fecharConsultas() { cs.aberta = false; $('tdCsVeu').hidden = true; travar(); const b = $('tdBtnSalvas'); if (b) b.focus(); }

  function usarConsulta(id) {
    const c = achar(id);
    if (!c) return;
    const e = CT.deSalvo(c);
    Object.assign(ten, { modo: e.modo, grupos: e.grupos, fichas: e.fichas, pos: e.pos }); visao = e.visao;
    limparAviso(); limparTagsInexistentes();
    cs.origem = id; cs.base = consultaDaTela();
    fecharConsultas(); render();
  }

  // salvar
  function abrirSalvar() {
    const q = CT.consulta(ten);
    if (!q.valida || q.vazia) return;
    const o = origemItem(), mexida = alterada(), podeAtualizar = !!o && mexida && o.podeAlterar;
    cs.sv = true; $('tdSvNome').value = ''; $('tdSvErro').textContent = ''; $('tdSvCnt').textContent = '0 de ' + CT.MAX_NOME_CONSULTA;
    $('tdSvSub').innerHTML = '<b>' + esc(CT.fraseSalva(consultaDaTela())) + '</b><br>Visão: ' + esc(NOME_VISAO[visao]) + '. Todos do espaço poderão usar; só você altera (e o dono modera).' +
      (o && mexida && !o.podeAlterar ? '<br>A consulta de <b>' + esc(o.autor) + '</b> não muda: isto cria uma cópia sua.' : '');
    const at = $('tdSvAtual');
    at.hidden = !podeAtualizar;
    if (podeAtualizar) at.textContent = 'Atualizar "' + o.nome + '"' + (o.minha ? '' : ' (como dono)');
    $('tdSvOk').textContent = o && mexida ? 'Salvar como nova' : 'Salvar';
    const cheio = !!cs.lista && cs.lista.length >= CT.MAX_CONSULTAS;
    $('tdSvOk').disabled = cheio;
    if (cheio) $('tdSvErro').textContent = 'Limite de ' + CT.MAX_CONSULTAS + ' consultas. Exclua uma antes de salvar outra.';
    $('tdSvVeu').hidden = false; travar(); $('tdSvNome').focus();
  }
  function fecharSalvar() { cs.sv = false; $('tdSvVeu').hidden = true; travar(); const b = $('tdBtnSalvar'); if (b) b.focus(); }
  async function salvarNova() {
    if (cs.ocupado || !cs.sv) return;
    const nome = $('tdSvNome').value.trim(), er = erroDoNome(nome, null);
    if (er) { $('tdSvErro').textContent = er; return; }
    cs.ocupado = true; $('tdSvOk').disabled = true;
    try {
      const c = await db.criarConsulta(Object.assign({ nome }, consultaDaTela()));
      cs.lista = [c].concat((cs.lista || []).filter((x) => x.id !== c.id));
      cs.origem = c.id; cs.base = consultaDaTela();
      fecharSalvar(); render(); toast('✓ Consulta "' + c.nome + '" salva');
      if (cs.aberta) carregarConsultas();
    } catch (e) { erroJanela('tdSvErro', e); }
    cs.ocupado = false; $('tdSvOk').disabled = false;
  }
  async function atualizarAtual() {
    const o = origemItem();
    if (!o || cs.ocupado || !cs.sv) return;
    cs.ocupado = true; $('tdSvAtual').disabled = true;
    try {
      const c = await db.atualizarConsulta(o.id, Object.assign({ version: o.version }, consultaDaTela()));
      substituir(c); cs.base = consultaDaTela();
      fecharSalvar(); render(); toast('✓ Consulta "' + c.nome + '" atualizada');
      if (cs.aberta) carregarConsultas();
    } catch (e) { erroJanela('tdSvErro', e); }
    cs.ocupado = false; $('tdSvAtual').disabled = false;
  }

  // renomear
  function abrirRenomear(id) {
    const c = achar(id);
    if (!c || !CT.acoesDaConsulta(c).renomear) return;
    cs.rn = id; $('tdRnNome').value = c.nome; $('tdRnErro').textContent = ''; $('tdRnCnt').textContent = c.nome.length + ' de ' + CT.MAX_NOME_CONSULTA;
    $('tdRnSub').textContent = c.minha ? 'Só o nome muda.' : 'Consulta de ' + c.autor + '. Como dono, você pode renomear.';
    $('tdRnVeu').hidden = false; travar(); $('tdRnNome').focus(); $('tdRnNome').select();
  }
  function fecharRenomear() { cs.rn = null; $('tdRnVeu').hidden = true; travar(); }
  async function renomear() {
    const c = achar(cs.rn);
    if (!c || cs.ocupado) return;
    const nome = $('tdRnNome').value.trim(), er = erroDoNome(nome, c.id);
    if (er) { $('tdRnErro').textContent = er; return; }
    cs.ocupado = true; $('tdRnOk').disabled = true;
    try {
      substituir(await db.atualizarConsulta(c.id, { version: c.version, nome }));
      fecharRenomear(); renderConsultas(); render();
    } catch (e) { erroJanela('tdRnErro', e); }
    cs.ocupado = false; $('tdRnOk').disabled = false;
  }

  // excluir na hora, com Desfazer por 6 s (o Desfazer é restaurar a mesma consulta, com o mesmo autor)
  async function excluirConsulta(id) {
    const c = achar(id);
    if (!c || cs.ocupado || !CT.acoesDaConsulta(c).excluir) return;
    cs.ocupado = true;
    try { await db.excluirConsulta(id); } catch (e) { cs.ocupado = false; cs.erro = mensagemDe(e); renderConsultas(); return; }
    cs.ocupado = false;
    const eraOrigem = cs.origem === id;
    cs.lista = cs.lista.filter((x) => x.id !== id);
    if (eraOrigem) cs.origem = null; // a consulta na tela fica; só deixa de ser "carregada de"
    renderConsultas(); render();
    const aviso = $('avisoAcao');
    const abaixar = () => aviso.classList.remove('acima');
    $('toast').classList.remove('on'); // o aviso de Desfazer ocupa o mesmo lugar
    mostrarAvisoAcao({ txt: 'Excluída: ' + c.nome, sub: 'Dá para desfazer por 6 segundos.', rotulo: 'Desfazer', onFim: abaixar,
      onAcao: async () => {
        abaixar();
        try {
          const r = await db.restaurarConsulta(id);
          cs.lista = [r].concat((cs.lista || []).filter((x) => x.id !== id)); ordenar();
          if (eraOrigem) cs.origem = id;
          renderConsultas(); render(); toast('✓ Consulta "' + r.nome + '" de volta');
        } catch (e) { toast(mensagemDe(e), true); }
      } });
    aviso.classList.add('acima'); // a janela das consultas (z 80) cobriria o aviso
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
    const t = e.target.closest('#tdBtnSalvas,#tdCsFechar,#tdCsRetry,[data-cs-f],[data-cs-usar],[data-cs-ren],[data-cs-del],#tdBtnSalvar,#tdSvFechar,#tdSvOk,#tdSvAtual,#tdRnFechar,#tdRnOk,#tdBtnLista,[data-mes],[data-lm],#tdLFechar,[data-v],[data-combo],#tdCombos,#tdBtnTag,[data-add],[data-rm],[data-del],#tdNovo,[data-t],#tdJok,[data-i],[data-k],[data-tag],#tdAvanc,#tdVoltar,#tdVoltarCel,[data-cot],#tdDesf,#tdEditar,#tdFechar,#tdPronto');
    if (!t) {
      if (e.target === $('tdVeu')) fecharJanela(); else if (e.target === $('tdListaVeu')) fecharLista();
      else if (e.target === $('tdCsVeu')) fecharConsultas(); else if (e.target === $('tdSvVeu')) fecharSalvar(); else if (e.target === $('tdRnVeu')) fecharRenomear();
      return;
    }
    if (t.id === 'tdBtnSalvas') return abrirConsultas();
    if (t.id === 'tdCsFechar') return fecharConsultas();
    if (t.id === 'tdCsRetry') return carregarConsultas();
    if (t.dataset.csF) { cs.filtro = t.dataset.csF; return renderConsultas(); }
    if (t.dataset.csUsar) return usarConsulta(t.dataset.csUsar);
    if (t.dataset.csRen) return abrirRenomear(t.dataset.csRen);
    if (t.dataset.csDel) return excluirConsulta(t.dataset.csDel);
    if (t.id === 'tdBtnSalvar') return abrirSalvar();
    if (t.id === 'tdSvFechar') return fecharSalvar();
    if (t.id === 'tdSvOk') return salvarNova();
    if (t.id === 'tdSvAtual') return atualizarAtual();
    if (t.id === 'tdRnFechar') return fecharRenomear();
    if (t.id === 'tdRnOk') return renomear();
    if (t.id === 'tdBtnLista') return abrirLista('');
    if (t.dataset.mes) return abrirLista(t.dataset.mes);
    if (t.dataset.lm !== undefined) { listaMes = t.dataset.lm; renderLista(); $('tdLCorpo').scrollTop = 0; return; }
    if (t.id === 'tdLFechar') return fecharLista();
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
