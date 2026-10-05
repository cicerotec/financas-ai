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

  let ESPACO = null;
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
    mostrarSessao(body.email, lista);
    return ESPACO;
  }
  function mostrarSessao(email, lista) {
    const h = document.querySelector('header');
    if (!h || document.getElementById('sessao')) return;
    const d = document.createElement('div');
    d.id = 'sessao';
    d.className = 'muted';
    d.style.cssText = 'margin:0 0 6px;display:flex;gap:8px;align-items:center;flex-wrap:wrap;';
    const t = document.createElement('span');
    t.id = 'sessaoNome';
    t.textContent = nomeExibicao() + ' · ';
    d.appendChild(t);
    if (lista.length > 1) {
      const s = document.createElement('select');
      s.style.cssText = 'width:auto;padding:2px 6px;font-size:12px;';
      lista.forEach((e) => { const o = document.createElement('option'); o.value = e.id; o.textContent = e.nome; s.appendChild(o); });
      s.value = ESPACO.id;
      s.onchange = () => { localStorage.setItem('financas.space', s.value); location.reload(); };
      d.appendChild(s);
    } else {
      const n = document.createElement('span');
      n.textContent = ESPACO.nome + (ESPACO.role === 'member' ? ' · membro' : '');
      d.appendChild(n);
    }
    const dica = document.createElement('span');
    dica.id = 'sessaoDica';
    dica.style.cssText = 'font-size:11px;';
    dica.textContent = '(defina seu nome em Listas)';
    dica.style.display = ESPACO && ESPACO.apelido ? 'none' : '';
    d.appendChild(dica);
    const b = document.createElement('button');
    b.className = 'ghost';
    b.style.cssText = 'padding:2px 8px;font-size:12px;';
    b.textContent = 'Sair';
    b.onclick = sair;
    d.appendChild(b);
    h.insertBefore(d, h.firstChild);
  }
  // Nome de exibição: o apelido da pessoa; enquanto não definir, a parte do e-mail antes do @.
  function nomeExibicao() {
    return (ESPACO && ESPACO.apelido) || (EMAIL ? EMAIL.split('@')[0] : '');
  }
  function atualizarSessao() {
    const t = document.getElementById('sessaoNome');
    if (t) t.textContent = nomeExibicao() + ' · ';
    const d = document.getElementById('sessaoDica');
    if (d) d.style.display = ESPACO && ESPACO.apelido ? 'none' : '';
  }
  const base = async () => '/spaces/' + (await espaco()).id;
  // Cada pessoa só altera o próprio nome (o servidor usa o sub do token). Vazio remove o apelido.
  async function definirApelido(nome) {
    const { body } = await api('PUT', (await base()) + '/perfil', { apelido: nome });
    ESPACO.apelido = body.apelido || '';
    atualizarSessao();
    return ESPACO.apelido;
  }

  // ---------- configuracoes (documentos) ----------
  // A aparencia e pessoal (cada pessoa a sua); o resto de config/listas e compartilhado.
  const PESSOAL = ['statusCor', 'estiloCor', 'layoutSaldos', 'ordemSaldos', 'ordemManual'];
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
    ouvintes.forEach((L) => {
      if (L.tipo !== 'col' || !L.docs) return;
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

  const ouvintes = new Set();
  let agendado = null;
  function disparar(L) {
    L.run().then(L.cb).catch((e) => { if (L.err) L.err(e); });
  }
  function atualizarOuvintes(tipo) {
    clearTimeout(agendado);
    agendado = setTimeout(() => ouvintes.forEach((L) => { if (!tipo || L.tipo === tipo) disparar(L); }), 150);
  }
  // A cada poucos minutos (aba visivel) e ao voltar para a aba: configuracoes sao poucos itens pequenos, entao
  // recarregam; os lancamentos so recarregam se o seq mudou (ou se nao deu para ler).
  async function checar() {
    if (document.visibilityState !== 'visible') return;
    ultimaChecagem = Date.now();
    atualizarOuvintes('doc');
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
  Consulta.prototype.onSnapshot = function (cb, err) {
    const q = this;
    const L = { tipo: 'col', q, docs: [] };
    L.run = async () => {
      const s = await lerSeq().catch(() => null); // lido ANTES dos lancamentos: nada gravado no meio se perde
      L.docs = await rodar(q);
      seqConhecido = s;
      return snapDocs(L.docs);
    };
    L.cb = cb; L.err = err;
    ouvintes.add(L); disparar(L);
    return () => ouvintes.delete(L);
  };
  Consulta.prototype.add = async function (obj) {
    const { body } = escrita(await api('POST', (await base()) + '/tx', obj));
    const { _seq, ...d } = body;
    guardar(d); escreverNoCache('put', d); registrarSeq(_seq);
    return { id: d.id };
  };
  Consulta.prototype.doc = function (id) {
    return {
      update: async (patch) => {
        const antigo = cache.get(id);
        if (!antigo) throw new Error('Lancamento nao carregado: recarregue a pagina.');
        const { body } = escrita(await api('PUT', (await base()) + '/tx/' + id + '?de=' + encodeURIComponent(antigo.dataEvento), patch));
        const { _seq, ...d } = body;
        guardar(d); escreverNoCache('put', d); registrarSeq(_seq);
      },
      delete: async () => {
        const antigo = cache.get(id);
        if (!antigo) throw new Error('Lancamento nao carregado: recarregue a pagina.');
        const { body } = escrita(await api('DELETE', (await base()) + '/tx/' + id + '?de=' + encodeURIComponent(antigo.dataEvento)));
        cache.delete(id); escreverNoCache('del', { id }); registrarSeq(body._seq);
      }
    };
  };

  // Importacao de backup: 25 por chamada; o servidor devolve o que o DynamoDB nao aceitou
  // a tempo (limite de escrita) e reenviamos ate acabar. Reimportar sobrescreve pelo id.
  // Envia um lote. Falha de rede ou erro 5xx/429 e tentada de novo (ate 3 vezes, com espera): reenviar e seguro
  // porque os itens levam id (o servidor sobrescreve pelo id em vez de duplicar). Erro 4xx de verdade sobe na hora.
  async function enviarLote(b, lote, aviso) {
    for (let t = 1; ; t++) {
      try {
        return (await api('POST', b + '/tx/batch', { itens: lote })).body;
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
    try {
      for (let i = 0; i < itens.length; i += TAM) {
        let lote = itens.slice(i, i + TAM);
        let primeira = true, tentativas = 0;
        while (lote.length) {
          const body = await enviarLote(b, lote, (t) => progresso && progresso(processados, itens.length, { espera: t }));
          gravados += body.gravados || 0;
          if (primeira) (body.rejeitados || []).forEach((r) => rejeitados.push({ indice: i + r.indice, motivo: r.motivo }));
          primeira = false;
          lote = body.pendentes || [];
          if (lote.length) {
            if (++tentativas > 10) throw new Error('O banco de dados recusou itens repetidamente (capacidade de escrita esgotada).');
            if (progresso) progresso(processados, itens.length, { espera: tentativas });
            await new Promise((r) => setTimeout(r, 1500));
          }
        }
        processados = Math.min(i + TAM, itens.length);
        if (progresso) progresso(processados, itens.length);
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
    atualizarOuvintes('doc');
  }

  // Fechamento mensal por banco (so no app com servidor). Os 409 voltam como { status, body }
  // para a tela decidir (version velha = alguem gravou no meio; ordem = so o mais recente reabre).
  const fechamentos = async () => (await api('GET', (await base()) + '/fechamentos')).body;
  const salvarCaches = async (banco, version, baseCfg, caches) =>
    api('PUT', (await base()) + '/fechamentos', { banco, version, base: baseCfg, caches });
  const conferirMes = async (dados) => api('POST', (await base()) + '/fechamentos/conferir', dados);
  const reabrirMes = async (banco, mes) => api('POST', (await base()) + '/fechamentos/reabrir', { banco, mes });

  const dbShim = {
    importarLancamentos,
    adicionarTags,
    fechamentos,
    salvarCaches,
    conferirMes,
    reabrirMes,
    collection: (nome) => {
      if (nome !== 'lancamentos') throw new Error('colecao desconhecida: ' + nome);
      return new Consulta();
    },
    doc: (caminho) => {
      const nome = caminho.replace(/^config\//, '');
      const snap = (d) => ({ exists: d !== undefined, data: () => d });
      return {
        get: async () => snap(await lerCfg(nome)),
        set: async (obj) => { await gravarCfg(nome, obj); atualizarOuvintes('doc'); },
        onSnapshot: (cb, err) => {
          const L = { tipo: 'doc', run: async () => snap(await lerCfg(nome)), cb, err };
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
    definirApelido,
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
