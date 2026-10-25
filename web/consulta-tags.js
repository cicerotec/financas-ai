// Logica pura da consulta por tags da aba Tendencias: grupos (E e NAO dentro do grupo, OU entre grupos) e expressao livre
// (parenteses, E, OU, NAO). Sem DOM nem rede; testada em consulta-tags.test.js (node web/consulta-tags.test.js).
// O front so desenha e soma o que ja esta na tela: nenhum numero daqui vai para a API.
(function (raiz) {
  const MAX_GRUPOS = 4;      // grupos ligados por OU
  const MAX_TAGS = 8;        // tags em cada lista de um grupo ("precisa ter" e "nao pode ter")
  const MAX_FICHAS = 60;     // fichas da expressao livre
  const MAX_OUTRAS = 6;      // outras tags mostradas na lista

  // Ficha da expressao: {k:'tag',v} | {k:'op',v:'E'|'OU'|'NAO'} | {k:'('} | {k:')'}
  const nomeFicha = (x) => (x.k === 'tag' ? x.v : x.k === 'op' ? (x.v === 'NAO' ? 'NÃO' : x.v) : x.k);
  const clonar = (x) => JSON.parse(JSON.stringify(x));
  const grupoVazio = () => ({ tem: [], nao: [] });
  const grupoPreenchido = (g) => g.tem.length > 0 || g.nao.length > 0;
  const casaGrupo = (g, tags) => g.tem.every((t) => tags.includes(t)) && !g.nao.some((t) => tags.includes(t));

  // Leitor da expressao: NAO antes de E, E antes de OU (como em SQL). Devolve a arvore; erro lancado tem .msg.
  function analisar(fichas) {
    let i = 0;
    const fim = () => i >= fichas.length;
    const ant = () => (i > 0 ? fichas[i - 1] : null);
    const falha = (msg) => { const e = new Error(msg); e.msg = msg; throw e; };
    function operando() {
      if (fim()) { const a = ant(); falha(a ? 'Falta uma tag depois de "' + nomeFicha(a) + '".' : 'A expressão está vazia.'); }
      const x = fichas[i];
      if (x.k === 'tag') { i++; return { t: 'tag', v: x.v }; }
      if (x.k === 'op' && x.v === 'NAO') { i++; return { t: 'nao', a: operando() }; }
      if (x.k === '(') {
        i++;
        if (!fim() && fichas[i].k === ')') falha('Parêntese vazio: coloque tags dentro dele.');
        const e = ou();
        if (fim() || fichas[i].k !== ')') falha('Falta fechar um parêntese.');
        i++;
        return e;
      }
      if (x.k === ')') { const a = ant(); falha(a ? 'Falta uma tag depois de "' + nomeFicha(a) + '".' : 'Tem ")" sem "(" antes.'); }
      return falha('Falta uma tag antes de "' + nomeFicha(x) + '".');
    }
    function e_() {
      const xs = [operando()];
      while (!fim() && fichas[i].k === 'op' && fichas[i].v === 'E') { i++; xs.push(operando()); }
      return xs.length === 1 ? xs[0] : { t: 'e', xs };
    }
    function ou() {
      const xs = [e_()];
      while (!fim() && fichas[i].k === 'op' && fichas[i].v === 'OU') { i++; xs.push(e_()); }
      return xs.length === 1 ? xs[0] : { t: 'ou', xs };
    }
    const r = ou();
    if (!fim()) {
      const x = fichas[i], a = ant();
      if (x.k === ')') falha('Tem ")" sem "(" antes.');
      falha('Falta E ou OU entre "' + nomeFicha(a) + '" e "' + nomeFicha(x) + '".');
    }
    return r;
  }

  const avalia = (n, tags) => (n.t === 'tag' ? tags.includes(n.v) : n.t === 'nao' ? !avalia(n.a, tags)
    : n.t === 'e' ? n.xs.every((x) => avalia(x, tags)) : n.xs.some((x) => avalia(x, tags)));

  // "Lido como": a expressao com todos os parenteses, para a pessoa conferir como a precedencia foi entendida.
  function texto(n, topo) {
    if (topo === undefined) topo = true;
    if (n.t === 'tag') return n.v;
    if (n.t === 'nao') return 'NÃO ' + texto(n.a, false);
    const s = n.xs.map((x) => texto(x, false)).join(n.t === 'e' ? ' E ' : ' OU ');
    return topo && n.t === 'ou' ? s : '(' + s + ')';
  }

  // Frase de um grupo ("Lazer E Comida NÃO Presente") e da consulta de grupos.
  function fraseGrupo(g) {
    let s = g.tem.join(' E ');
    if (g.nao.length) s += (g.tem.length ? ' ' : '') + g.nao.map((t) => 'NÃO ' + t).join(' ');
    return s;
  }
  function fraseGrupos(grupos) {
    const a = grupos.filter(grupoPreenchido);
    if (!a.length) return '';
    return a.length === 1 ? fraseGrupo(a[0]) : a.map((g) => '(' + fraseGrupo(g) + ')').join(' OU ');
  }

  // A consulta de um estado {modo:'grupos'|'avancado', grupos, fichas}. Sem nada escolhido, tudo passa.
  // valida=false so no modo avancado, com a mensagem do que falta.
  function consulta(estado) {
    if (estado.modo !== 'avancado') {
      const a = estado.grupos.filter(grupoPreenchido);
      return { valida: true, vazia: !a.length, pred: (tags) => !a.length || a.some((g) => casaGrupo(g, tags)),
        mencionadas: new Set(a.flatMap((g) => g.tem.concat(g.nao))), no: null };
    }
    if (!estado.fichas.length) return { valida: true, vazia: true, pred: () => true, mencionadas: new Set(), no: null };
    try {
      const no = analisar(estado.fichas);
      return { valida: true, vazia: false, pred: (tags) => avalia(no, tags), no,
        mencionadas: new Set(estado.fichas.filter((x) => x.k === 'tag').map((x) => x.v)) };
    } catch (e) {
      if (!e.msg) throw e;
      return { valida: false, msg: e.msg };
    }
  }

  // Grupos -> fichas, ao abrir o modo avancado (a consulta aparece ja traduzida).
  function gruposParaFichas(grupos) {
    const a = grupos.filter(grupoPreenchido), out = [];
    a.forEach((g, i) => {
      const p = [];
      g.tem.forEach((t) => { if (p.length) p.push({ k: 'op', v: 'E' }); p.push({ k: 'tag', v: t }); });
      g.nao.forEach((t) => { if (p.length) p.push({ k: 'op', v: 'E' }); p.push({ k: 'op', v: 'NAO' }, { k: 'tag', v: t }); });
      if (i) out.push({ k: 'op', v: 'OU' });
      if (a.length > 1 && p.length > 1) out.push({ k: '(' }, ...p, { k: ')' }); else out.push(...p);
    });
    return out.slice(0, MAX_FICHAS);
  }

  // Insere uma ficha no cursor. Tag, "(" ou NAO logo depois de uma tag ou de ")" ganham um E implicito.
  function inserirFicha(fichas, pos, x) {
    if (fichas.length >= MAX_FICHAS) return { fichas, pos, cheio: true };
    const antes = pos > 0 ? fichas[pos - 1] : null;
    const fecha = antes && (antes.k === 'tag' || antes.k === ')');
    const abre = x.k === 'tag' || x.k === '(' || (x.k === 'op' && x.v === 'NAO');
    const novos = fecha && abre && fichas.length + 2 <= MAX_FICHAS ? [{ k: 'op', v: 'E' }, x] : [x];
    const f = fichas.slice();
    f.splice(pos, 0, ...novos);
    return { fichas: f, pos: pos + novos.length, cheio: false };
  }

  // A base da Tendencias: so o que o Historico tambem conta nos totais. Fica de fora o lancamento oculto (ocultar tira do saldo) e o
  // previsto (o servidor o marca; ele nem vem na lista do Historico). Entrada e tudo que nao e saida (como o valorAssinado do
  // Historico). O mes e o da data local do lancamento. A Tendencias e uma simulacao: le o que ja esta na tela e nunca grava nada.
  function base(lancamentos) {
    return lancamentos.filter((l) => !l.oculto && !l.previsto).map((l) => {
      const d = new Date(l.dataEvento);
      return { mes: d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0'), valor: Number(l.valor) || 0, tags: l.tags || [], entrada: l.tipo !== 'saida', ref: l };
    });
  }

  const VISOES = ['gastos', 'receitas', 'saldo'];
  const centavos = (x) => Math.round(x * 100) / 100 + 0; // + 0 troca -0 por 0

  // Resume os lancamentos da consulta. itens: [{mes, valor, tags, entrada}] (so os que contam; sem 'entrada' e saida).
  // visao: 'gastos' soma as saidas, 'receitas' soma as entradas e 'saldo' e renda menos gasto (a conta do total do Historico).
  // Uma barra do mes se divide em "so as tags da consulta" (total - ext) e "tem outras tags" (ext): por lancamento, nunca por
  // pedaco do valor. No saldo os valores levam sinal (entrada soma, saida subtrai).
  // vol/volExt: o movimento (valor sem sinal), a base das porcentagens no saldo, onde valores que se cancelam nao servem de base.
  // cot: as outras tags, com o movimento dos lancamentos da consulta que a tem (se sobrepoem: uma compra com 5 tags conta nas 5).
  function resumir(itens, q, visao) {
    if (!VISOES.includes(visao)) visao = 'gastos';
    const meses = new Map();
    const co = new Map();
    let n = 0, total = 0, totExt = 0, vol = 0, volExt = 0;
    const sel = []; // os lancamentos que entraram, para a lista da consulta
    itens.forEach((l) => {
      if (visao === 'gastos' && l.entrada) return;
      if (visao === 'receitas' && !l.entrada) return;
      const tags = l.tags || [];
      if (!q.pred(tags)) return;
      const val = visao === 'saldo' ? (l.entrada ? l.valor : -l.valor) : l.valor;
      n++; total += val; vol += l.valor; sel.push(l);
      const m = meses.get(l.mes) || { mes: l.mes, total: 0, ext: 0 };
      m.total += val;
      const outras = q.vazia ? [] : tags.filter((t) => !q.mencionadas.has(t)); // sem filtro nao ha 'outras tags'
      if (outras.length) { m.ext += val; totExt += val; volExt += l.valor; }
      outras.forEach((t) => co.set(t, (co.get(t) || 0) + l.valor));
      meses.set(l.mes, m);
    });
    const cot = [...co.entries()].map(([t, v]) => ({ t, v: centavos(v) }))
      .sort((x, y) => y.v - x.v || x.t.localeCompare(y.t, 'pt-BR'));
    const lista = [...meses.values()].map((m) => ({ mes: m.mes, total: centavos(m.total), ext: centavos(m.ext) }))
      .sort((x, y) => (x.mes < y.mes ? -1 : x.mes > y.mes ? 1 : 0));
    return { meses: lista, n, total: centavos(total), totExt: centavos(totExt), vol: centavos(vol), volExt: centavos(volExt), cot, vazia: !!q.vazia, visao, itens: sel };
  }

  // O app carrega so os N lancamentos mais recentes (limite). Se vieram N, o mes mais antigo pode estar incompleto: devolve esse mes
  // ('aaaa-mm', o primeiro do grafico) para o aviso, ou '' quando nao ha corte provavel. Nao resolve o corte: so evita a confusao.
  function corte(carregados, limite, r) {
    return carregados >= limite && r.meses.length ? r.meses[0].mes : '';
  }

  // A lista da consulta (janela so de leitura da Tendencias). r: o resultado de resumir; mes: '' (todos) ou 'aaaa-mm'; max: linhas a desenhar.
  // Ordem: do mais novo ao mais antigo. A soma segue a visao (no saldo, entrada soma e saida subtrai) e tem de bater com o grafico
  // (r.total, ou a barra do mes): "bate" e a conferencia dos numeros. Mes que nao tem lancamento cai em "todos".
  function listar(r, mes, max) {
    const todos = r.itens.slice().sort((a, b) => new Date(b.ref.dataEvento) - new Date(a.ref.dataEvento));
    const contagem = new Map();
    todos.forEach((l) => contagem.set(l.mes, (contagem.get(l.mes) || 0) + 1));
    const meses = [...contagem.entries()].map(([m, n]) => ({ mes: m, n })).sort((a, b) => (a.mes < b.mes ? -1 : a.mes > b.mes ? 1 : 0));
    if (!contagem.has(mes)) mes = '';
    const vis = mes ? todos.filter((l) => l.mes === mes) : todos;
    const val = (l) => (r.visao === 'saldo' ? (l.entrada ? l.valor : -l.valor) : l.valor);
    const soma = (arr) => centavos(arr.reduce((t, l) => t + val(l), 0));
    const somaMes = {};
    meses.forEach((m) => { somaMes[m.mes] = soma(todos.filter((l) => l.mes === m.mes)); });
    const esperado = mes ? r.meses.find((m) => m.mes === mes).total : r.total;
    const total = soma(vis);
    return { mes, meses, n: vis.length, linhas: vis.slice(0, max), corte: vis.length > max, soma: total, esperado, bate: Math.abs(total - esperado) < 0.005, somaMes };
  }

  // "Tirar (NAO)" numa outra tag: nos grupos, o NAO entra em todos os grupos preenchidos; na expressao, ela inteira vira
  // "(expressao) E NAO tag". Devolve {ok, estado} ou {ok:false, erro} sem mexer no estado recebido.
  function tirarTag(estado, tag) {
    const e = clonar(estado);
    if (e.modo !== 'avancado') {
      let cheio = false;
      e.grupos.filter(grupoPreenchido).forEach((g) => {
        if (g.nao.includes(tag)) return;
        if (g.nao.length >= MAX_TAGS) { cheio = true; return; }
        g.nao.push(tag);
      });
      return cheio ? { ok: false, erro: 'Algum grupo já tem ' + MAX_TAGS + ' tags em "Não pode ter". Libere uma vaga primeiro.' } : { ok: true, estado: e };
    }
    if (!consulta(e).valida) return { ok: false, erro: 'Corrija a expressão antes de tirar uma tag.' };
    const envolve = e.fichas.length > 1;
    if (e.fichas.length + (envolve ? 2 : 0) + 3 > MAX_FICHAS) return { ok: false, erro: 'A expressão passaria de ' + MAX_FICHAS + ' fichas.' };
    e.fichas = [].concat(envolve ? [{ k: '(' }] : [], e.fichas, envolve ? [{ k: ')' }] : [],
      [{ k: 'op', v: 'E' }, { k: 'op', v: 'NAO' }, { k: 'tag', v: tag }]);
    e.pos = e.fichas.length;
    return { ok: true, estado: e };
  }

  const api = { MAX_GRUPOS, MAX_TAGS, MAX_FICHAS, MAX_OUTRAS, nomeFicha, grupoVazio, grupoPreenchido, casaGrupo, analisar, avalia, texto,
    fraseGrupo, fraseGrupos, consulta, gruposParaFichas, inserirFicha, resumir, listar, corte, tirarTag, VISOES, base };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else raiz.ConsultaTags = api;
})(typeof window !== 'undefined' ? window : globalThis);
