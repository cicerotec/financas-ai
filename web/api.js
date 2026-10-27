// Substitui o ambiente do Claude: login (Cognito Hosted UI + PKCE) e uma camada com a mesma
// API de banco que o app ja usa (collection/doc/where/orderBy/limit/onSnapshot), falando
// com a Lambda. O app continua chamando claude.use("db") e claude.use("downloads").
(function () {
  const CFG = window.FINANCAS_CONFIG;
  const LS = 'financas.auth';
  const API = CFG.apiUrl.replace(/\/$/, '');

  // ---------- login ----------
  const b64url = (buf) => btoa(String.fromCharCode(...new Uint8Array(buf))).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  const redirectUri = () => location.origin + '/';
  const lerAuth = () => { try { return JSON.parse(localStorage.getItem(LS)); } catch (e) { return null; } };
  function salvarAuth(t) {
    const antigo = lerAuth() || {};
    const a = { id: t.id_token, refresh: t.refresh_token || antigo.refresh, exp: Date.now() + (t.expires_in - 60) * 1000 };
    localStorage.setItem(LS, JSON.stringify(a));
    return a;
  }
  function pedirToken(params) {
    return fetch('https://' + CFG.loginDomain + '/oauth2/token', {
      method: 'POST',
      headers: { 'content-type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams(params)
    });
  }
  async function irParaLogin() {
    const v = b64url(crypto.getRandomValues(new Uint8Array(32)));
    const c = b64url(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(v)));
    const state = b64url(crypto.getRandomValues(new Uint8Array(16)));
    sessionStorage.setItem('financas.pkce', JSON.stringify({ v, state }));
    location.href = 'https://' + CFG.loginDomain + '/oauth2/authorize?' + new URLSearchParams({
      response_type: 'code', client_id: CFG.clientId, redirect_uri: redirectUri(),
      scope: 'openid email', code_challenge: c, code_challenge_method: 'S256', state
    });
    return new Promise(() => {}); // a pagina vai embora
  }
  async function token() {
    const p = new URLSearchParams(location.search);
    if (p.get('code')) {
      const s = JSON.parse(sessionStorage.getItem('financas.pkce') || 'null');
      history.replaceState({}, '', location.pathname);
      if (!s || s.state !== p.get('state')) return irParaLogin();
      const r = await pedirToken({ grant_type: 'authorization_code', client_id: CFG.clientId, code: p.get('code'), redirect_uri: redirectUri(), code_verifier: s.v });
      if (!r.ok) return irParaLogin();
      return salvarAuth(await r.json()).id;
    }
    const a = lerAuth();
    if (a && a.exp > Date.now()) return a.id;
    if (a && a.refresh) {
      const r = await pedirToken({ grant_type: 'refresh_token', client_id: CFG.clientId, refresh_token: a.refresh });
      if (r.ok) return salvarAuth(await r.json()).id;
    }
    return irParaLogin();
  }
  function sair() {
    localStorage.removeItem(LS);
    location.href = 'https://' + CFG.loginDomain + '/logout?' + new URLSearchParams({ client_id: CFG.clientId, logout_uri: redirectUri() });
  }

  // ---------- chamadas HTTP ----------
  async function api(method, path, body) {
    const id = await token();
    const r = await fetch(API + path, {
      method,
      headers: Object.assign({ authorization: 'Bearer ' + id }, body ? { 'content-type': 'application/json' } : {}),
      body: body ? JSON.stringify(body) : undefined
    });
    if (r.status === 401) { localStorage.removeItem(LS); return irParaLogin(); }
    const j = await r.json().catch(() => ({}));
    if (!r.ok && r.status !== 409) {
      const e = new Error(j.erro || ('HTTP ' + r.status));
      e.code = r.status;
      throw e;
    }
    return { status: r.status, body: j };
  }

  // Escrita de lancamento recusada (409): mes conferido e travado. Vira erro com a mensagem do servidor.
  function escrita(r) {
    if (r.status === 409) {
      const e = new Error((r.body && r.body.erro) || 'Este mês está conferido e travado.');
      e.code = 409;
      e.codigo = r.body && r.body.codigo;
      throw e;
    }
    return r;
  }

  // Mes fechado por data (item 38): o servidor recusa com 409 mes_fechado e quem decide e ele. Aqui so se mostra a
  // pergunta (window.confirmarMesFechado, do app.js) e, se a pessoa confirmar, reenvia com ?confirmaMesFechado=1.
  // Ids ja confirmados nesta sessao reenviam direto (o Desfazer de uma edicao confirmada nao pergunta de novo).
  const confirmados = new Set();
  const comConfirmacao = (path, confirma) => path + (confirma ? (path.includes('?') ? '&' : '?') + 'confirmaMesFechado=1' : '');
  async function escreverConfirmando(id, acao, enviar) {
    let confirma = !!(id && confirmados.has(id));
    let r = await enviar(confirma);
    if (r.status === 409 && r.body && r.body.codigo === 'mes_fechado' && !confirma && window.confirmarMesFechado) {
      if (!(await window.confirmarMesFechado(r.body, acao))) {
        const e = new Error('Cancelado: nada foi alterado.');
        e.codigo = 'cancelado';
        throw e;
      }
      confirma = true;
      r = await enviar(true);
    }
    r = escrita(r);
    const ident = id || (r.body && r.body.id);
    if (confirma && ident) confirmados.add(ident);
    return r;
  }

  let ESPACO = null;
  let ESPACOS = [];
  let SUB = '';
  let EMAIL = '';
  async function espaco() {
    if (ESPACO) return ESPACO;
    const { body } = await api('GET', '/me');
    const lista = body.espacos || [];
    SUB = body.sub || '';
    EMAIL = body.email || '';
    if (!lista.length) throw new Error('Sua conta ainda nao tem um espaco. Rode scripts/seed.sh.');
    const guardado = localStorage.getItem('financas.space');
    ESPACO = lista.find((e) => e.id === guardado) || lista[0];
    ESPACOS = lista;
    return ESPACO;
  }
  // Nome de exibição: o apelido da pessoa; enquanto não definir, a parte do e-mail antes do @.
  function nomeExibicao() {
    return (ESPACO && ESPACO.apelido) || (EMAIL ? EMAIL.split('@')[0] : '');
  }
  const base = async () => '/spaces/' + (await espaco()).id;
  // Cada pessoa só altera o próprio nome (o servidor usa o sub do token). Vazio remove o apelido.
  async function definirApelido(nome) {
    const { body } = await api('PUT', (await base()) + '/perfil', { apelido: nome });
    ESPACO.apelido = body.apelido || '';
    return ESPACO.apelido;
  }

  // Chat id do Telegram de quem esta logado (para onde vao os avisos). Vazio desvincula.
  async function definirTelegram(chatId) {
    const { body } = await api('PUT', (await base()) + '/perfil', { telegramChatId: String(chatId || '').trim() });
    ESPACO.telegramChatId = body.telegramChatId || '';
    return ESPACO.telegramChatId;
  }
  async function testarAviso() {
    await api('POST', (await base()) + '/avisos/teste', {});
  }

  // ---------- configuracoes (documentos) ----------
  // A aparencia e pessoal (cada pessoa a sua); o resto de config/listas e compartilhado.
  const PESSOAL = ['tema', 'statusCor', 'estiloCor', 'layoutSaldos', 'ordemSaldos', 'ordemManual'];
  const versoes = {};
  const limpa = (b) => { const o = Object.assign({}, b); delete o.version; return o; };
  async function lerCfg(nome) {
    const b = await base();
    const { body } = await api('GET', b + '/cfg/' + nome);
    versoes[nome] = body.version || 0;
    let dados = limpa(body);
    if (nome === 'listas') {
      const ap = await api('GET', b + '/aparencia');
      dados = Object.assign(dados, limpa(ap.body));
    }
    return Object.keys(dados).length ? dados : undefined;
  }
  async function gravarCfg(nome, obj) {
    const b = await base();
    const membro = ESPACO && ESPACO.role === 'member';
    let comum = obj;
    if (nome === 'listas') {
      const pes = {}; comum = {};
      Object.keys(obj).forEach((k) => { (PESSOAL.includes(k) ? pes : comum)[k] = obj[k]; });
      await api('PUT', b + '/aparencia', pes);
      if (membro) return; // para o membro so a aparencia e dele; o resto de "listas" e do dono
    } else if (membro) {
      const e = new Error('Seu perfil não pode alterar ' + nome + '.');
      e.code = 403;
      throw e;
    }
    for (let tentativa = 0; tentativa < 2; tentativa++) {
      const r = await api('PUT', b + '/cfg/' + nome, Object.assign({}, comum, { version: versoes[nome] || 0 }));
      if (r.status === 409) { versoes[nome] = (r.body.atual && r.body.atual.version) || 0; continue; }
      versoes[nome] = r.body.version;
      return;
    }
    throw new Error('Nao consegui salvar ' + nome + ' (alterado por outra pessoa).');
  }

  // ---------- lancamentos ----------
  const cache = new Map(); // id -> ultimo documento visto (precisa da dataEvento para editar/excluir)
  const guardar = (d) => { cache.set(d.id, d); return d; };
  const snapDocs = (arr) => ({ docs: arr.map((d) => ({ id: d.id, data: () => Object.assign({}, d) })) });

  // Contador de alteracoes do espaco (seq). Cada escrita de lancamento soma 1 no servidor e devolve o valor em
  // "_seq". Se o que volta de uma escrita nossa e o ultimo que conhecemos + 1, ninguem gravou no meio; se pulou,
  // alguem gravou e recarregamos. A checagem periodica le so o seq (1 leitura) em vez dos 1000 lancamentos.
  let seqConhecido = null;
  let ultimaChecagem = 0;
  async function lerSeq() {
    const { body } = await api('GET', (await base()) + '/seq');
    return typeof body.seq === 'number' ? body.seq : null;
  }

  // Cache write-through: depois de uma escrita nossa, a copia que cada listener ja tem e ajustada so naquele
  // registro (e reentregue sem ir ao banco). So uma recarga completa busca tudo de novo.
  function casa(d, w) {
    return w.every(([f, op, v]) => {
      const x = d[f];
      if (op === '==') return x === v;
      if (op === '>=') return x >= v;
      if (op === '>') return x > v;
      if (op === '<') return x < v;
      if (op === '<=') return x <= v;
      if (op === 'array-contains') return (x || []).includes(v);
      return true;
    });
  }
  function ordenar(arr, ord) {
    const sinal = ord[1] === 'asc' ? 1 : -1;
    return arr.sort((a, b) => {
      const c = a[ord[0]] < b[ord[0]] ? -1 : a[ord[0]] > b[ord[0]] ? 1 : (a.id < b.id ? -1 : a.id > b.id ? 1 : 0);
      return c * sinal;
    });
  }
  function escreverNoCache(op, d) {
    avisarMudanca();
    ouvintes.forEach((L) => {
      if (L.tipo !== 'col' || !L.carregado) return; // ainda nao leu nada: a primeira leitura ja traz o registro novo
      const docs = L.docs.filter((x) => x.id !== d.id); // tira a versao antiga, se estava na lista
      if (op === 'put' && casa(d, L.q.w)) docs.push(Object.assign({}, d));
      ordenar(docs, L.q.ord);
      if (docs.length > L.q.lim) docs.length = L.q.lim;
      L.docs = docs;
      try { L.cb(snapDocs(docs)); } catch (e) { if (L.err) L.err(e); }
    });
  }
  function registrarSeq(n) {
    if (typeof n !== 'number') { seqConhecido = null; return; } // o servidor nao disse: a proxima checagem recarrega
    const esperado = seqConhecido !== null && n === seqConhecido + 1;
    const pulou = seqConhecido !== null && !esperado;
    seqConhecido = n;
    if (pulou) atualizarOuvintes('col'); // alguem gravou no meio
  }

  async function rodar(q) {
    const b = await base();
    const params = { ordem: q.ord[1] === 'asc' ? 'asc' : 'desc', limite: '1000' };
    const locais = [];
    q.w.forEach(([f, op, v]) => {
      if (f === 'dataEvento' && op === '>=') params.de = v;
      else if (f === 'dataEvento' && op === '>') params.de = v + '~';
      else if (f === 'dataEvento' && op === '<') params.ate = v;
      else if (f === 'dataEvento' && op === '<=') params.ate = v + '~';
      else if (f === 'banco' && op === '==') params.banco = v;
      else if (f === 'status' && op === '==') params.status = v;
      else locais.push([f, op, v]);
    });
    const passa = (d) => locais.every(([f, op, v]) =>
      op === 'array-contains' ? (d[f] || []).includes(v) : op === '==' ? d[f] === v : true);
    const res = [];
    let cursor = '';
    for (let i = 0; i < 200 && res.length < q.lim; i++) {
      const p = new URLSearchParams(params);
      if (cursor) p.set('cursor', cursor);
      const { body } = await api('GET', b + '/tx?' + p);
      (body.itens || []).forEach((d) => { if (passa(d)) res.push(guardar(d)); });
      if (!body.proximo) break;
      cursor = body.proximo;
    }
    return res.slice(0, q.lim);
  }

  // Cada ouvinte e uma leitura que uma tela precisa. `ativo === false` (leitura sob demanda) quer dizer que a tela que a usa
  // nao esta aberta: uma mudanca so a marca como `suja` e ela e relida quando a tela voltar a pedi-la. Nada e lido por
  // uma tela que ninguem esta vendo.
  const ouvintes = new Set();
  let agendado = null;
  const pendentes = [];
  let ultimaLeituraDocs = Date.now();
  const DOCS_TTL = 600000; // configuracoes lidas ha mais de 10 min sao relidas na proxima checagem
  const aoMudar = new Set(); // quem quer saber que os lancamentos mudaram (ex.: Futuros, que le por conta propria)
  function avisarMudanca() { aoMudar.forEach((f) => { try { f(); } catch (e) { console.error(e); } }); }
  // Uma leitura por vez em cada ouvinte: se pedirem de novo durante a leitura, le outra vez no fim (e nao duas ao mesmo tempo).
  function carregar(L) {
    if (L.emCurso) { L.refazer = true; return L.emCurso; }
    L.emCurso = L.run()
      .then((s) => { L.carregado = true; L.suja = false; if (L.tipo === 'doc') ultimaLeituraDocs = Date.now(); L.cb(s); })
      .catch((e) => { if (L.err) L.err(e); })
      .then(() => { L.emCurso = null; if (L.refazer) { L.refazer = false; carregar(L); } });
    return L.emCurso;
  }
  const disparar = carregar;
  // tipo ('col' ou 'doc') e nome (so 'doc': 'listas', 'tags'...) limitam o que e relido; sem nome, todos do tipo.
  function atualizarOuvintes(tipo, nome) {
    pendentes.push([tipo, nome]);
    clearTimeout(agendado);
    agendado = setTimeout(() => {
      const lote = pendentes.splice(0);
      ouvintes.forEach((L) => {
        if (!lote.some(([t, n]) => (!t || L.tipo === t) && (!n || L.nome === n))) return;
        if (L.ativo === false) { L.suja = true; return; }
        carregar(L);
      });
      if (lote.some(([t]) => !t || t === 'col')) avisarMudanca();
    }, 150);
  }
  // A cada poucos minutos (aba visivel) e ao voltar para a aba: so o seq (uma leitura minuscula) e so se algum lancamento ja
  // foi lido; os lancamentos so sao relidos se o seq mudou. As configuracoes nao tem seq: so envelhecem (DOCS_TTL).
  async function checar() {
    if (document.visibilityState !== 'visible') return;
    ultimaChecagem = Date.now();
    if (Date.now() - ultimaLeituraDocs > DOCS_TTL) atualizarOuvintes('doc');
    if (![...ouvintes].some((L) => L.tipo === 'col' && L.carregado)) return; // nenhuma tela leu lancamentos: nada a conferir
    try {
      const s = await lerSeq();
      if (s === null || s !== seqConhecido) atualizarOuvintes('col');
    } catch (e) { atualizarOuvintes('col'); }
  }
  setInterval(checar, 180000);
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible' && Date.now() - ultimaChecagem > 60000) checar();
  });

  function Consulta(w, ord, lim) { this.w = w || []; this.ord = ord || ['dataEvento', 'desc']; this.lim = lim || 1000; }
  Consulta.prototype.where = function (f, op, v) { return new Consulta(this.w.concat([[f, op, v]]), this.ord, this.lim); };
  Consulta.prototype.orderBy = function (f, d) { return new Consulta(this.w, [f, d || 'asc'], this.lim); };
  Consulta.prototype.limit = function (n) { return new Consulta(this.w, this.ord, n); };
  Consulta.prototype.get = async function () { return snapDocs(await rodar(this)); };
  function ouvinteDeLancamentos(q, cb, err, ativo) {
    const L = { tipo: 'col', q, docs: [], ativo, carregado: false, suja: false };
    L.run = async () => {
      const s = await lerSeq().catch(() => null); // lido ANTES dos lancamentos: nada gravado no meio se perde
      L.docs = await rodar(q);
      seqConhecido = s;
      return snapDocs(L.docs);
    };
    L.cb = cb; L.err = err;
    ouvintes.add(L);
    return L;
  }
  Consulta.prototype.onSnapshot = function (cb, err) {
    const L = ouvinteDeLancamentos(this, cb, err, true);
    disparar(L);
    return () => ouvintes.delete(L);
  };
  // Leitura sob demanda: nao le nada ate a tela pedir. `ativar()` le (se nunca leu ou se mudou desde a ultima vez) e passa a
  // reler a cada mudanca; `pausar()` para de reler (a tela saiu) e so marca que mudou; `garantir()` le uma vez, sem ativar,
  // para quem precisa dos dados num gesto (ex.: conferir duplicados). Os tres devolvem uma promessa da leitura.
  Consulta.prototype.sobDemanda = function (cb, err) {
    const L = ouvinteDeLancamentos(this, cb, err, false);
    const garantir = () => (!L.carregado || L.suja ? carregar(L) : Promise.resolve());
    return {
      ativar: () => { L.ativo = true; return garantir(); },
      pausar: () => { L.ativo = false; },
      garantir,
      carregado: () => L.carregado
    };
  };
  Consulta.prototype.add = async function (obj) {
    const b = await base();
    const { body } = await escreverConfirmando(obj && obj.id, 'Lançar', (c) => api('POST', comConfirmacao(b + '/tx', c), obj));
    const { _seq, ...d } = body;
    guardar(d); escreverNoCache('put', d); registrarSeq(_seq);
    return { id: d.id };
  };
  Consulta.prototype.doc = function (id) {
    return {
      update: async (patch) => {
        const antigo = cache.get(id);
        if (!antigo) throw new Error('Lancamento nao carregado: recarregue a pagina.');
        const b = await base();
        const { body } = await escreverConfirmando(id, 'Alterar', (c) => api('PUT', comConfirmacao(b + '/tx/' + id + '?de=' + encodeURIComponent(antigo.dataEvento), c), patch));
        const { _seq, ...d } = body;
        guardar(d); escreverNoCache('put', d); registrarSeq(_seq);
      },
      delete: async () => {
        const antigo = cache.get(id);
        if (!antigo) throw new Error('Lancamento nao carregado: recarregue a pagina.');
        const b = await base();
        const { body } = await escreverConfirmando(id, 'Excluir', (c) => api('DELETE', comConfirmacao(b + '/tx/' + id + '?de=' + encodeURIComponent(antigo.dataEvento), c)));
        cache.delete(id); escreverNoCache('del', { id }); registrarSeq(body._seq);
        return body; // traz desfazer: { versao } (a versao a restaurar para desfazer a exclusao)
      }
    };
  };

  // Importacao de backup: 25 por chamada; o servidor devolve o que o DynamoDB nao aceitou
  // a tempo (limite de escrita) e reenviamos ate acabar. Reimportar sobrescreve pelo id.
  // Envia um lote. Falha de rede ou erro 5xx/429 e tentada de novo (ate 3 vezes, com espera): reenviar e seguro
  // porque os itens levam id (o servidor sobrescreve pelo id em vez de duplicar). Erro 4xx de verdade sobe na hora.
  async function enviarLote(b, lote, aviso, confirma) {
    for (let t = 1; ; t++) {
      try {
        return (await api('POST', comConfirmacao(b + '/tx/batch', confirma), { itens: lote })).body;
      } catch (e) {
        const temporario = !e.code || e.code >= 500 || e.code === 429;
        if (!temporario || t >= 3) throw e;
        if (aviso) aviso(t);
        await new Promise((r) => setTimeout(r, 2000 * t));
      }
    }
  }
  // progresso(processados, total, info): info.espera = numero da tentativa quando esta esperando o banco liberar.
  // Se parar no meio, o erro leva gravados e processados (e.gravados, e.processados) para a tela dizer onde parou.
  async function importarLancamentos(itens, progresso) {
    const b = await base();
    const TAM = 25;
    let gravados = 0, processados = 0;
    const rejeitados = [];
    // Envia a lista em lotes; indiceDe(i) devolve a posicao original no arquivo do item i da lista.
    const enviarLista = async (lista, confirma, indiceDe, comProgresso) => {
      for (let i = 0; i < lista.length; i += TAM) {
        let lote = lista.slice(i, i + TAM);
        let primeira = true, tentativas = 0;
        while (lote.length) {
          const body = await enviarLote(b, lote, (t) => comProgresso && progresso && progresso(processados, itens.length, { espera: t }), confirma);
          gravados += body.gravados || 0;
          if (primeira) (body.rejeitados || []).forEach((r) => rejeitados.push({ indice: indiceDe(i + r.indice), motivo: r.motivo, codigo: r.codigo }));
          primeira = false;
          lote = body.pendentes || [];
          if (lote.length) {
            if (++tentativas > 10) throw new Error('O banco de dados recusou itens repetidamente (capacidade de escrita esgotada).');
            if (comProgresso && progresso) progresso(processados, itens.length, { espera: tentativas });
            await new Promise((r) => setTimeout(r, 1500));
          }
        }
        if (comProgresso) {
          processados = Math.min(i + TAM, itens.length);
          if (progresso) progresso(processados, itens.length);
        }
      }
    };
    try {
      await enviarLista(itens, false, (i) => i, true);
      // Os de mes fechado ficaram de fora: uma pergunta so, e reenvia apenas esses com a confirmacao.
      const fechados = rejeitados.filter((r) => r.codigo === 'mes_fechado');
      if (fechados.length && window.confirmarMesFechadoLote && (await window.confirmarMesFechadoLote(fechados.length))) {
        const demais = rejeitados.filter((r) => r.codigo !== 'mes_fechado');
        rejeitados.length = 0;
        demais.forEach((r) => rejeitados.push(r));
        await enviarLista(fechados.map((r) => itens[r.indice]), true, (i) => fechados[i].indice, false);
      }
    } catch (e) {
      e.gravados = gravados; e.processados = processados; e.rejeitados = rejeitados;
      atualizarOuvintes('col'); // o que ja entrou aparece no historico
      throw e;
    }
    atualizarOuvintes('col');
    return { gravados, rejeitados };
  }

  // Acrescenta tags que ainda nao existem (o servidor ignora as repetidas). Vale para dono e membro.
  async function adicionarTags(tags) {
    await api('POST', (await base()) + '/tags', { tags });
    atualizarOuvintes('doc', 'tags');
  }

  // Fechamento mensal por banco (so no app com servidor). Os 409 voltam como { status, body }
  // para a tela decidir (version velha = alguem gravou no meio; ordem = so o mais recente reabre).
  const saldos = async () => (await api('GET', (await base()) + '/saldos')).body;
  const conferirMes = async (dados) => api('POST', (await base()) + '/fechamentos/conferir', dados);
  // Futuros (item 41): o servidor devolve os previstos, o resumo, os Atrasados e os meses prontos. Cada item e guardado no cache do shim (precisa da dataEvento para editar).
  const futuros = async () => {
    const b = (await api('GET', (await base()) + '/futuros?ocultos=1')).body;
    (b.atrasados || []).forEach((a) => (a.meses || []).forEach((m) => (m.itens || []).forEach(guardar)));
    (b.meses || []).forEach((m) => (m.itens || []).forEach(guardar));
    return b;
  };
  // Atrasados de um ano anterior, so a pedido (item 45): o servidor devolve o grupo do ano pronto ({ano, n, valor, meses}).
  const futurosAno = async (ano) => {
    const b = (await api('GET', (await base()) + '/futuros?ocultos=1&ano=' + encodeURIComponent(ano))).body;
    (b.meses || []).forEach((m) => (m.itens || []).forEach(guardar));
    return b;
  };
  // Transferencia entre meios: o servidor cria as duas pontas numa transacao so (POST /transferencias).
  const transferir = async (origem, destino, valor, dataEvento) => {
    const b = await base();
    const r = await escreverConfirmando(null, 'Transferir',
      (confirma) => api('POST', comConfirmacao(b + '/transferencias', confirma), { origem, destino, valor, dataEvento }));
    registrarSeq(r.body._seq);
    atualizarOuvintes('col');
    return r.body;
  };
  // Lista de status que o servidor sugere para uma data (ISO); sem data, a de hoje.
  const statusSugeridos = async (data) =>
    (await api('GET', (await base()) + '/status' + (data ? '?data=' + encodeURIComponent(data) : ''))).body;
  // Confirma um previsto: { valor, dataEvento, status }. `de` e a dataEvento atual. Mes fechado por data pergunta e reenvia (como as outras escritas).
  const confirmarPrevisto = async (id, de, dados) => {
    const b = await base();
    const r = await escreverConfirmando(id, 'Confirmar',
      (confirma) => api('POST', comConfirmacao(b + '/tx/' + encodeURIComponent(id) + '/confirmar?de=' + encodeURIComponent(de), confirma), dados));
    registrarSeq(r.body._seq);
    atualizarOuvintes('col');
    return r.body;
  };
  const reabrirMes = async (banco, mes) => api('POST', (await base()) + '/fechamentos/reabrir', { banco, mes });

  // Faturas do cartao (item 33a): o servidor calcula periodo, fatura de cada compra, total, conferencia e sugestoes; os comandos so valem para o dono.
  // Os 409 (diferenca da conferencia) voltam como { status, body }; os outros erros viram excecao com a mensagem do servidor.
  const cartaoUrl = async (meio, resto) => (await base()) + '/cartoes' + (meio ? '/' + encodeURIComponent(meio) : '') + (resto || '');
  const cartoes = async () => (await api('GET', await cartaoUrl())).body;
  const cartaoFatura = async (meio, ym) => (await api('GET', await cartaoUrl(meio, '/faturas' + (ym ? '/' + ym : '')))).body;
  const cartaoAdicionar = async (meio, diaPadrao) => (await api('POST', await cartaoUrl(), { meio, diaPadrao })).body;
  const cartaoRemover = async (meio) => (await api('DELETE', await cartaoUrl(meio))).body;
  const cartaoInicio = async (meio, inicio) => (await api('PUT', await cartaoUrl(meio), { inicio })).body;
  // pagador e dia do vencimento do cartao ({ pagador?, diaVencimento? }; null remove)
  const cartaoConfig = async (meio, dados) => (await api('PUT', await cartaoUrl(meio), dados)).body;
  const cartaoFechamento = async (meio, ym, dados) => (await api('PUT', await cartaoUrl(meio, '/faturas/' + ym), dados)).body;
  const cartaoConferir = async (meio, ym, aceitarDiferenca) =>
    api('POST', await cartaoUrl(meio, '/faturas/' + ym + '/conferir'), { aceitarDiferenca: !!aceitarDiferenca });
  const cartaoDesconferir = async (meio, ym) => (await api('DELETE', await cartaoUrl(meio, '/faturas/' + ym + '/conferir'))).body;
  const cartaoDiferenca = async (meio, ym) => {
    const r = (await api('POST', await cartaoUrl(meio, '/faturas/' + ym + '/diferenca'), {})).body;
    atualizarOuvintes('col'); // o servidor criou um lancamento
    return r;
  };

  // Historico de versoes (.ai/specs/historico). Quem decide o que aparece, o diff e se uma versao pode ser restaurada e o servidor.
  // Busca por periodo da alteracao (e texto): periodo = {mes:'AAAA-MM'} | {dias:N} | {tudo:true}. Devolve { truncado, itens, mes|dias|tudo }.
  const historico = async (periodo, q) => {
    const p = new URLSearchParams();
    if (periodo.mes) p.set('mes', periodo.mes);
    else if (periodo.dias) p.set('dias', String(periodo.dias));
    else if (periodo.tudo) p.set('tudo', '1');
    if (q) p.set('q', q);
    return (await api('GET', (await base()) + '/historico?' + p)).body;
  };
  // Versoes de um lancamento, da mais nova para a mais antiga. de (dataEvento) so serve a lancamento sem nenhuma versao.
  const versoesDoLancamento = async (id, de) =>
    (await api('GET', (await base()) + '/tx/' + encodeURIComponent(id) + '/versoes' + (de ? '?de=' + encodeURIComponent(de) : ''))).body;
  // Restaura a versao n (grava uma versao nova). Os 409 voltam como { status, body } (mes conferido, ja esta nessa versao,
  // versao da exclusao, conflito); o 200 avisa as outras telas, que leem lancamentos, para recarregar.
  const restaurarVersao = async (id, n, confirma) => {
    // id ja confirmado nesta sessao (ex.: exclusao em mes fechado) leva a confirmacao tambem: o Desfazer nao pergunta de novo
    const r = await api('POST', comConfirmacao((await base()) + '/tx/' + encodeURIComponent(id) + '/versoes/' + n + '/restaurar', confirma || confirmados.has(id)));
    if (r.status === 200) { registrarSeq(r.body._seq); atualizarOuvintes('col'); }
    return r;
  };

  const dbShim = {
    importarLancamentos,
    adicionarTags,
    aoMudarLancamentos: (f) => { aoMudar.add(f); return () => aoMudar.delete(f); },
    saldos,
    conferirMes,
    futuros,
    futurosAno,
    transferir,
    statusSugeridos,
    confirmarPrevisto,
    cartoes,
    cartaoFatura,
    cartaoAdicionar,
    cartaoRemover,
    cartaoInicio,
    cartaoConfig,
    cartaoFechamento,
    cartaoConferir,
    cartaoDesconferir,
    cartaoDiferenca,
    reabrirMes,
    historico,
    versoesDoLancamento,
    restaurarVersao,
    collection: (nome) => {
      if (nome !== 'lancamentos') throw new Error('colecao desconhecida: ' + nome);
      return new Consulta();
    },
    doc: (caminho) => {
      const nome = caminho.replace(/^config\//, '');
      const snap = (d) => ({ exists: d !== undefined, data: () => d });
      return {
        get: async () => snap(await lerCfg(nome)),
        set: async (obj) => { await gravarCfg(nome, obj); atualizarOuvintes('doc', nome); }, // so este documento e relido
        onSnapshot: (cb, err) => {
          const L = { tipo: 'doc', nome, run: async () => snap(await lerCfg(nome)), cb, err };
          ouvintes.add(L); disparar(L);
          return () => ouvintes.delete(L);
        }
      };
    }
  };

  // ---------- downloads ----------
  const downloads = {
    save: async ({ filename, data }) => {
      const a = document.createElement('a');
      a.href = URL.createObjectURL(new Blob([data], { type: 'application/json' }));
      a.download = filename;
      document.body.appendChild(a); a.click(); a.remove();
      setTimeout(() => URL.revokeObjectURL(a.href), 5000);
      return true;
    }
  };

  window.claude = {
    papel: () => (ESPACO ? ESPACO.role : null),
    sub: () => SUB,
    apelido: () => (ESPACO && ESPACO.apelido) || '',
    // A conta de quem esta logado, para o menu (web/js/usuario.js): nome de exibicao, espaco atual e a lista de espacos.
    sessao: () => (ESPACO ? { nome: nomeExibicao(), email: EMAIL, apelido: ESPACO.apelido || '', espaco: { id: ESPACO.id, nome: ESPACO.nome, role: ESPACO.role }, espacos: ESPACOS.map((e) => ({ id: e.id, nome: e.nome })) } : null),
    trocarEspaco: (id) => { localStorage.setItem('financas.space', id); location.reload(); },
    sair,
    definirApelido,
    telegramChatId: () => (ESPACO && ESPACO.telegramChatId) || '',
    definirTelegram,
    testarAviso,
    use: async (nome) => {
      if (nome === 'db') {
        try { await espaco(); } catch (e) { alert(e.message); return null; }
        return dbShim;
      }
      if (nome === 'downloads') return downloads;
      return null; // "sample" (IA) ainda nao existe fora do Claude
    }
  };
})();
