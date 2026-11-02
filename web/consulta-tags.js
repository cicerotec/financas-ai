// Logica pura da consulta por tags da aba Tendencias: grupos (E e NAO dentro do grupo, OU entre grupos) e expressao livre
// (parenteses, E, OU, NAO). Sem DOM nem rede; testada em consulta-tags.test.js (node web/consulta-tags.test.js).
// O front so desenha e soma o que ja esta na tela: nenhum numero daqui vai para a API.
(function (raiz) {
  const MAX_GRUPOS = 4;      // grupos ligados por OU
  const MAX_TAGS = 8;        // tags em cada lista de um grupo ("precisa ter" e "nao pode ter")
  const MAX_FICHAS = 60;     // fichas da expressao livre
  const MAX_OUTRAS = 6;      // outras tags mostradas na lista
  const MAX_CONSULTAS = 50;  // consultas salvas vivas por espaco (o servidor tambem recusa)
  const MAX_NOME_CONSULTA = 40;

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
      return { mes: d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0'), dia: d.getDate(), valor: Number(l.valor) || 0, tags: l.tags || [], entrada: l.tipo !== 'saida', ref: l };
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

  // ---- Visoes dos gastos (item 55): Calendario de calor e Maiores tags do mes. Mesma consulta, mesma base: so muda o desenho. ----
  // So Gastos e Receitas (no Resultado, entrada e saida se cancelam e o dia ou a tag perde o sentido): devolve null.
  const mesAnterior = (mes) => { const a = Number(mes.slice(0, 4)), m = Number(mes.slice(5, 7)); return m === 1 ? (a - 1) + '-12' : a + '-' + String(m - 1).padStart(2, '0'); };
  function itensDoMes(itens, q, visao, mes) {
    return itens.filter((l) => l.mes === mes && (visao === 'gastos' ? !l.entrada : l.entrada) && q.pred(l.tags || []));
  }
  // Uma celula por dia do mes. As faixas 1 a 5 saem do maior dia do mes (ate 20%, 40%, 60%, 80% e 100% dele); 0 e dia sem lancamento.
  // inicio: o dia da semana do dia 1, com a segunda-feira = 0. Mes que nao tem lancamento da consulta: max 0 e todos os dias em 0.
  function calendario(itens, q, visao, mes) {
    if (visao !== 'gastos' && visao !== 'receitas') return null;
    const a = Number(mes.slice(0, 4)), m = Number(mes.slice(5, 7));
    const dias = new Date(a, m, 0).getDate();
    const valores = new Array(dias).fill(0);
    itensDoMes(itens, q, visao, mes).forEach((l) => { valores[l.dia - 1] += l.valor; });
    const v = valores.map(centavos);
    const max = Math.max(0, ...v);
    const faixa = (x) => (x <= 0 ? 0 : Math.min(5, Math.max(1, Math.ceil(x / max * 5 - 1e-9))));
    return { mes, dias, inicio: (new Date(a, m - 1, 1).getDay() + 6) % 7, valores: v, faixas: v.map(faixa), max,
      cortes: [1, 2, 3, 4, 5].map((k) => centavos(max * k / 5)), total: centavos(v.reduce((t, x) => t + x, 0)) };
  }
  // As tags dos lancamentos da consulta no mes, do maior ao menor (empate: ordem alfabetica). Um lancamento com varias tags conta em cada uma.
  // Variacao contra o mes anterior: pct inteiro, 'novo' (antes era 0), 'sem' (o mes anterior e o cortado pelo limite de carga ou anterior a ele:
  // pode estar incompleto) e 'igual'. mesCorte: o retorno de corte(), ou ''.
  function maioresTags(itens, q, visao, mes, mesCorte, max) {
    if (visao !== 'gastos' && visao !== 'receitas') return null;
    const soma = (ls) => { const t = new Map(); ls.forEach((l) => (l.tags || []).forEach((x) => t.set(x, (t.get(x) || 0) + l.valor))); return t; };
    const atual = soma(itensDoMes(itens, q, visao, mes)), ant = soma(itensDoMes(itens, q, visao, mesAnterior(mes)));
    const semBase = !!mesCorte && mesAnterior(mes) <= mesCorte;
    const todas = [...atual.entries()].map(([t, v]) => ({ t, v: centavos(v) })).sort((x, y) => y.v - x.v || x.t.localeCompare(y.t, 'pt-BR'));
    const lim = max || 8;
    const linhas = todas.slice(0, lim).map((x) => {
      const b = centavos(ant.get(x.t) || 0);
      const dif = semBase ? 'sem' : b === 0 ? 'novo' : Math.round((x.v - b) / b * 100);
      return { t: x.t, v: x.v, dif: dif === 0 ? 'igual' : dif };
    });
    const resto = todas.slice(lim).reduce((t, x) => t + x.v, 0);
    return { mes, linhas, outras: centavos(resto), nOutras: Math.max(0, todas.length - lim) };
  }

  // Ritmo do mes: o gasto acumulado dia a dia ate hoje (mes atual) ou ate o ultimo dia (mes passado), contra o mes anterior inteiro.
  // Projecao (so no mes atual, com 3 dias passados ou mais): o ritmo medio dos dias passados repetido ate o fim do mes. E uma simulacao:
  // nao soma previstos nem le Futuros. base: null quando o mes anterior nao tem lancamento da consulta ou pode estar incompleto (mesCorte, como em maioresTags).
  function ritmo(itens, q, visao, mes, hoje, mesCorte) {
    if (visao !== 'gastos' && visao !== 'receitas') return null;
    const acumular = (m, n) => {
      const d = new Array(n).fill(0);
      itensDoMes(itens, q, visao, m).forEach((l) => { if (l.dia <= n) d[l.dia - 1] += l.valor; });
      let t = 0;
      return d.map((v) => (t += v, centavos(t)));
    };
    const nDias = (m) => new Date(Number(m.slice(0, 4)), Number(m.slice(5, 7)), 0).getDate();
    const dias = nDias(mes), atual = hoje.getFullYear() + '-' + String(hoje.getMonth() + 1).padStart(2, '0') === mes;
    const ate = atual ? Math.min(hoje.getDate(), dias) : dias;
    const acum = acumular(mes, ate);
    const mb = mesAnterior(mes), semBase = !!mesCorte && mb <= mesCorte;
    const base = semBase ? null : acumular(mb, nDias(mb));
    const baseTotal = base ? base[base.length - 1] : 0;
    return { mes, dias, ate, atual, acum, total: acum.length ? acum[acum.length - 1] : 0, mesBase: mb, base: base && baseTotal > 0 ? base : null, semBase,
      projecao: atual && ate >= 3 ? centavos(acum[ate - 1] / ate * dias) : null };
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

  // ---- consultas salvas (item 53, .ai/specs/tendencias-consultas) ----
  // O que vai para o servidor e o que volta dele. So a consulta (tags, operadores e visao): nunca numero, nunca o cursor (pos).
  // O campo salvo da expressao livre se chama "expressao"; no front ele e o ten.fichas (so estas duas funcoes fazem a troca).
  const norm = (t) => String(t || '').normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase();
  const visaoValida = (v) => (VISOES.includes(v) ? v : 'gastos');

  // Corpo da consulta para POST e PUT: { modo, visao, grupos } ou { modo, visao, expressao }. Grupos vazios ficam de fora.
  function paraSalvar(estado, visao) {
    if (estado.modo === 'avancado') return { modo: 'avancado', visao: visaoValida(visao), expressao: clonar(estado.fichas) };
    return { modo: 'grupos', visao: visaoValida(visao),
      grupos: estado.grupos.filter(grupoPreenchido).map((g) => ({ tem: g.tem.slice(), nao: g.nao.slice() })) };
  }

  // O estado da tela a partir de uma consulta salva ({modo, grupos|expressao, visao}). Respeita os limites da tela, como se vindo de um combo.
  function deSalvo(item) {
    const visao = visaoValida(item.visao);
    if (item.modo === 'avancado') {
      const fichas = clonar(item.expressao || []).slice(0, MAX_FICHAS);
      return { modo: 'avancado', grupos: [grupoVazio()], fichas, pos: fichas.length, visao };
    }
    const grupos = clonar(item.grupos || []).slice(0, MAX_GRUPOS).map((g) => ({ tem: (g.tem || []).slice(0, MAX_TAGS), nao: (g.nao || []).slice(0, MAX_TAGS) }));
    const fichas = gruposParaFichas(grupos);
    return { modo: 'grupos', grupos: grupos.length ? grupos : [grupoVazio()], fichas, pos: fichas.length, visao };
  }

  // Salvar so faz sentido com alguma tag e, no modo avancado, uma expressao valida (o servidor so confere o formato).
  function podeSalvar(estado) {
    const q = consulta(estado);
    return q.valida && !q.vazia;
  }

  // A frase de uma consulta salva, para a lista. Expressao com erro de gramatica (o servidor nao a confere) mostra as fichas cruas.
  function fraseSalva(item) {
    if (item.modo !== 'avancado') return fraseGrupos(item.grupos || []);
    const f = item.expressao || [];
    try { return texto(analisar(f)); } catch (e) { return f.map(nomeFicha).join(' '); }
  }

  // Duas consultas (corpo de paraSalvar) sao a mesma? Serve para saber se a consulta carregada foi mexida.
  const mesmaConsulta = (a, b) => JSON.stringify(a) === JSON.stringify(b);

  // O que a lista mostra para cada consulta: Usar e de todos; Renomear e Excluir so com podeAlterar, que o SERVIDOR decide (o front nao calcula permissao).
  function acoesDaConsulta(item) {
    const pode = !!item.podeAlterar;
    return { usar: true, renomear: pode, excluir: pode, comoDono: pode && !item.minha,
      motivo: pode ? '' : 'Só ' + (item.autor || 'o autor') + ' ou o dono podem alterar.' };
  }

  // Filtra a lista da janela: ficha Todas | Minhas e busca (sem acento nem caixa) no nome e na frase. Devolve tambem as contagens das fichas.
  function filtrarConsultas(lista, filtro, busca) {
    const q = norm(busca).trim();
    const itens = lista.filter((c) => (filtro !== 'minhas' || c.minha) && (!q || norm(c.nome).includes(q) || norm(fraseSalva(c)).includes(q)));
    return { itens, todas: lista.length, minhas: lista.filter((c) => c.minha).length };
  }

  const api = { MAX_GRUPOS, MAX_TAGS, MAX_FICHAS, MAX_OUTRAS, nomeFicha, grupoVazio, grupoPreenchido, casaGrupo, analisar, avalia, texto,
    fraseGrupo, fraseGrupos, consulta, gruposParaFichas, inserirFicha, resumir, listar, corte, tirarTag, VISOES, base, mesAnterior, calendario, maioresTags, ritmo,
    MAX_CONSULTAS, MAX_NOME_CONSULTA, paraSalvar, deSalvo, podeSalvar, fraseSalva, mesmaConsulta, acoesDaConsulta, filtrarConsultas };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else raiz.ConsultaTags = api;
})(typeof window !== 'undefined' ? window : globalThis);
