// Testes do login (web/js/sessao.js), item 54 fase 2. Spec: .ai/specs/app-android/spec.md (A-T9 a A-T18 e A-T22a a A-T22j).
// Dois modos atras da mesma interface: navegador (como sempre) e app Android (aba do sistema, Cofre, ID token so em memoria).
// Roda em Node, com armazenamento, fetch, navegador do sistema e relogio de mentira. Rode com: node web/sessao.test.js
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const assert = require('assert');
const criarSessao = require('./js/sessao.js');

const ID = 'br.com.exemplo.app.dev', RET = ID + '://auth';
const CFG = { apiUrl: 'https://api.teste', loginDomain: 'login.teste', clientId: 'cliente' };
const esperar = (ms) => new Promise((r) => setTimeout(r, ms || 5));
const guarda = () => { const m = new Map(), escritas = []; return { m, escritas, getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => { m.set(k, String(v)); escritas.push([k, String(v)]); }, removeItem: (k) => { m.delete(k); } }; };
const resp = (status, body) => ({ status, ok: status >= 200 && status < 300, json: async () => { if (body === undefined) throw new Error('sem corpo'); return body; } });
const tokens = (extra) => Object.assign({ id_token: 'ID-1', refresh_token: 'REFRESH-1', expires_in: 3600 }, extra || {});

// Um aparelho de mentira: o Cofre (que sobrevive a "matar o app"), o App e o Browser do Capacitor, e o servidor de login.
function aparelho(op) {
  op = op || {};
  const cofre = op.cofre || new Map();
  const d = { cofre, abertas: [], pedidos: [], avisos: [], recargas: 0, ouvintes: {}, logs: [], agora: 1000000, ordem: [], ls: guarda(), ss: guarda(), launch: op.launch || null };
  d.login = op.login || (() => resp(200, tokens()));
  const Cofre = {
    ler: async ({ nome }) => { if (d.cofreQuebrado) throw new Error('keystore'); return { valor: cofre.has(nome) ? cofre.get(nome) : null }; },
    gravar: async ({ nome, valor }) => { if (d.cofreQuebrado) throw new Error('keystore'); cofre.set(nome, valor); d.ordem.push('gravar ' + nome); },
    apagar: async ({ nome }) => { if (d.apagarQuebrado) throw new Error('keystore'); cofre.delete(nome); d.ordem.push('apagar ' + nome); }
  };
  const App = { getInfo: async () => ({ id: ID }), getLaunchUrl: async () => (d.launch ? { url: d.launch } : undefined), addListener: (ev, f) => { d.ouvintes[ev] = f; } };
  const Browser = { open: async ({ url }) => { d.abertas.push(url); d.ordem.push('abrir'); } };
  const Plugins = Object.assign({ Cofre, App, Browser }, op.plugins || {});
  for (const k of Object.keys(Plugins)) if (Plugins[k] === null) delete Plugins[k];
  d.env = {
    cfg: CFG,
    Capacitor: op.navegador ? undefined : { isNativePlatform: () => true, Plugins },
    localStorage: d.ls, sessionStorage: d.ss,
    location: { origin: 'https://front.teste', search: op.search || '', pathname: '/', href: '' },
    history: { replaceState() { d.env.location.search = ''; } },
    crypto: require('crypto').webcrypto,
    btoa: (s) => Buffer.from(s, 'binary').toString('base64'),
    fetch: async (url, o) => {
      const p = Object.fromEntries(new URLSearchParams(o.body));
      d.pedidos.push({ url, p });
      if (d.semRede) throw new TypeError('Failed to fetch');
      return d.login(p);
    },
    agora: () => d.agora,
    recarregar: () => { d.recargas++; },
    aviso: (t) => { d.avisos.push(t); }
  };
  d.sessao = () => criarSessao(d.env);
  d.estadoDaAba = () => new URL(d.abertas[d.abertas.length - 1]).searchParams.get('state');
  return d;
}
const semSessao = async (p) => { let e = null; try { await p; } catch (x) { e = x; } assert.ok(e && e.codigo === 'sem_sessao', 'devia recusar com sem_sessao, veio ' + (e && (e.codigo || e.message))); };
const pendente = async (p) => { let fim = false; p.then(() => { fim = true; }, () => { fim = true; }); await esperar(20); assert.strictEqual(fim, false, 'a promessa devia ficar parada (a pagina recarrega)'); };

let falhas = 0;
const fila = [];
const teste = (nome, fn) => fila.push([nome, fn]);

// ---------- app Android ----------
teste('A-T9, A-T11, A-T22a: retorno valido troca o code com o verifier guardado, grava o refresh, apaga o PKCE e pede a recarga', async () => {
  const d = aparelho(), s = d.sessao();
  await semSessao(s.token());
  assert.deepStrictEqual(d.avisos, ['nova']);
  assert.strictEqual(d.abertas.length, 0, 'sem sessao o login NAO abre sozinho: so no toque em Entrar');
  await s.entrar();
  assert.strictEqual(d.abertas.length, 1);
  const u = new URL(d.abertas[0]);
  assert.strictEqual(u.origin + u.pathname, 'https://login.teste/oauth2/authorize');
  assert.strictEqual(u.searchParams.get('redirect_uri'), RET, 'o retorno vem do applicationId do pacote (App.getInfo), R18f');
  assert.strictEqual(u.searchParams.get('code_challenge_method'), 'S256');
  const pkce = JSON.parse(d.cofre.get('pkce'));
  assert.ok(pkce.v.length >= 43 && pkce.state === u.searchParams.get('state'));
  const fim = d.ouvintes.appUrlOpen({ url: RET + '?code=C1&state=' + pkce.state });
  await esperar();
  assert.strictEqual(d.pedidos.length, 1);
  assert.deepStrictEqual([d.pedidos[0].p.grant_type, d.pedidos[0].p.code, d.pedidos[0].p.code_verifier, d.pedidos[0].p.redirect_uri], ['authorization_code', 'C1', pkce.v, RET]);
  assert.strictEqual(d.cofre.get('refresh'), 'REFRESH-1');
  assert.strictEqual(d.cofre.has('pkce'), false, 'verifier e state somem no fim do login');
  assert.strictEqual(d.recargas, 1, 'depois do login a pagina recarrega (R18a)');
  assert.ok(d.avisos.includes('entrando'));
  void fim;
  // a pagina nova (outra instancia, mesmo Cofre) renova e devolve o ID token
  const d2 = aparelho({ cofre: d.cofre, login: () => resp(200, tokens({ id_token: 'ID-2', refresh_token: undefined })) });
  assert.strictEqual(await d2.sessao().token(), 'ID-2');
});

teste('A-T10, A-T11: state diferente, ausente, sem code ou de outro esquema nao chama /oauth2/token; com login pendente vira "nao concluiu"', async () => {
  for (const [url, pendencia] of [[RET + '?code=C1&state=outro', true], [RET + '?code=C1', true], [RET + '?state=X', true], [RET + '?code=C1&state=', true], ['outro.app://auth?code=C1&state=X', false], [RET + '/x?code=C1&state=X', false]]) {
    const d = aparelho(), s = d.sessao();
    await s.entrar();
    const st = d.estadoDaAba();
    d.ouvintes.appUrlOpen({ url: url.replace('state=X', 'state=' + st) });
    await esperar();
    assert.strictEqual(d.pedidos.length, 0, url);
    assert.strictEqual(d.recargas, 0, url);
    if (pendencia) {
      assert.strictEqual(d.cofre.has('pkce'), false, 'PKCE apagado tambem no erro: ' + url);
      assert.ok(d.avisos.includes('falhou'), url);
    } else {
      assert.strictEqual(d.cofre.has('pkce'), true, 'retorno de outro endereco e ignorado e o login segue pendente: ' + url);
    }
    assert.strictEqual(d.cofre.has('refresh'), false);
  }
});

teste('A-T22b, A-T20: app morto com a aba aberta; a instancia nova conclui pelo URL de lancamento, e o mesmo URL de novo nao faz nada', async () => {
  const d = aparelho();
  await d.sessao().entrar();
  const st = d.estadoDaAba();
  const d2 = aparelho({ cofre: d.cofre, launch: RET + '?code=C9&state=' + st });
  await pendente(d2.sessao().token());
  assert.strictEqual(d2.pedidos.length, 1);
  assert.strictEqual(d2.cofre.get('refresh'), 'REFRESH-1');
  assert.strictEqual(d2.recargas, 1);
  assert.strictEqual(d2.abertas.length, 0, 'nenhum login novo foi aberto');
  // depois da recarga o Android entrega o mesmo URL de lancamento: sem login pendente, e ignorado
  const d3 = aparelho({ cofre: d.cofre, launch: RET + '?code=C9&state=' + st, login: (p) => { assert.strictEqual(p.grant_type, 'refresh_token'); return resp(200, tokens({ id_token: 'ID-3' })); } });
  assert.strictEqual(await d3.sessao().token(), 'ID-3');
  assert.strictEqual(d3.recargas, 0);
});

teste('A-T22c: retorno sem login pendente (o do logout, por exemplo) nao chama /oauth2/token nem abre o navegador', async () => {
  const d = aparelho({ cofre: new Map([['refresh', 'R0']]) }), s = d.sessao();
  await s.token();
  d.pedidos.length = 0;
  d.ouvintes.appUrlOpen({ url: RET });
  d.ouvintes.appUrlOpen({ url: RET + '?code=C1&state=qualquer' });
  await esperar();
  assert.strictEqual(d.pedidos.length, 0);
  assert.strictEqual(d.abertas.length, 0);
  assert.strictEqual(d.recargas, 0);
});

teste('A-T12, A-T17: no app nada vai a localStorage nem a sessionStorage, o refresh fica no Cofre, o ID token so em memoria, e nada e logado', async () => {
  const d = aparelho(), s = d.sessao();
  const original = Object.assign({}, console);
  for (const k of ['log', 'info', 'warn', 'error', 'debug']) console[k] = (...a) => d.logs.push(a.join(' '));
  try {
    await semSessao(s.token());
    await s.entrar();
    d.ouvintes.appUrlOpen({ url: RET + '?code=C1&state=' + d.estadoDaAba() });
    await esperar();
    const d2 = aparelho({ cofre: d.cofre });
    d2.ls = d.ls; d2.env.localStorage = d.ls; d2.env.sessionStorage = d.ss;
    assert.strictEqual(await d2.sessao().token(), 'ID-1');
    await d2.sessao().recusado('ID-1').catch(() => {});
  } finally { Object.assign(console, original); }
  assert.deepStrictEqual(d.ls.escritas, [], 'localStorage');
  assert.deepStrictEqual(d.ss.escritas, [], 'sessionStorage (o verifier e o state vao ao Cofre)');
  assert.deepStrictEqual([...d.cofre.keys()], ['refresh']);
  assert.ok(![...d.cofre.values()].some((v) => v.includes('ID-1')), 'o ID token nao e guardado');
  assert.deepStrictEqual(d.logs, [], 'o modulo nao escreve log');
  assert.strictEqual(/console\./.test(fs.readFileSync(path.join(__dirname, 'js', 'sessao.js'), 'utf8')), false, 'sessao.js nao chama console');
});

teste('A-T13, A-T22h: sair apaga o Cofre e o espaco escolhido, espera, e so entao abre o logout com o logout_uri do app; depois recarrega', async () => {
  const d = aparelho({ cofre: new Map([['refresh', 'R0'], ['pkce', '{}']]) }), s = d.sessao();
  d.ls.m.set('financas.space', 'e1'); d.ls.m.set('financas.tema', 'dark');
  await s.sair();
  assert.strictEqual(d.cofre.size, 0);
  assert.deepStrictEqual(d.ordem, ['apagar refresh', 'apagar pkce', 'abrir'], 'apaga antes de abrir o logout');
  const u = new URL(d.abertas[0]);
  assert.strictEqual(u.origin + u.pathname, 'https://login.teste/logout');
  assert.strictEqual(u.searchParams.get('logout_uri'), RET);
  assert.strictEqual(d.ls.m.has('financas.space'), false);
  assert.strictEqual(d.ls.m.get('financas.tema'), 'dark', 'tema e as outras preferencias ficam');
  assert.strictEqual(d.recargas, 1);
  // apagar falhou: erro na tela, o logout nao abre e nada recarrega
  const q = aparelho({ cofre: new Map([['refresh', 'R0']]) });
  q.apagarQuebrado = true;
  await assert.rejects(q.sessao().sair());
  assert.deepStrictEqual([q.abertas.length, q.recargas, q.avisos], [0, 0, ['cofre']]);
});

teste('A-T14, A-T22i: refresh recusado (400) encerra a sessao; sem rede e 503 mantem; refresh sem refresh_token mantem o guardado', async () => {
  for (const erro of ['invalid_grant', 'invalid_client', 'invalid_request']) {
    const d = aparelho({ cofre: new Map([['refresh', 'R0']]), login: () => resp(400, { error: erro }) });
    await semSessao(d.sessao().token());
    assert.deepStrictEqual([d.cofre.has('refresh'), d.avisos, d.abertas.length], [false, ['fim'], 0], erro);
  }
  const rede = aparelho({ cofre: new Map([['refresh', 'R0']]) });
  rede.semRede = true;
  await assert.rejects(rede.sessao().token(), (e) => e.codigo === 'rede');
  const fora = aparelho({ cofre: new Map([['refresh', 'R0']]), login: () => resp(503) });
  await assert.rejects(fora.sessao().token(), (e) => e.codigo === 'rede');
  const estranho = aparelho({ cofre: new Map([['refresh', 'R0']]), login: () => resp(200, {}) });
  await assert.rejects(estranho.sessao().token(), (e) => e.codigo === 'rede');
  for (const d of [rede, fora, estranho]) assert.deepStrictEqual([d.cofre.get('refresh'), d.avisos, d.abertas.length], ['R0', [], 0], 'a sessao continua, sem tela de login');
  const bom = aparelho({ cofre: new Map([['refresh', 'R0']]), login: () => resp(200, { id_token: 'ID-9', expires_in: 3600 }) });
  assert.strictEqual(await bom.sessao().token(), 'ID-9');
  assert.strictEqual(bom.cofre.get('refresh'), 'R0', 'nunca grava vazio');
  const novo = aparelho({ cofre: new Map([['refresh', 'R0']]), login: () => resp(200, tokens({ refresh_token: 'R-NOVO' })) });
  await novo.sessao().token();
  assert.strictEqual(novo.cofre.get('refresh'), 'R-NOVO');
});

teste('A-T15, A-T22d: cinco chamadas juntas fazem um refresh so; o ID token vale ate vencer; depois de um 401 ha um refresh e nenhum login', async () => {
  let n = 0;
  const d = aparelho({ cofre: new Map([['refresh', 'R0']]), login: async () => { n++; await esperar(10); return resp(200, tokens({ id_token: 'ID-' + n, refresh_token: undefined })); } });
  const s = d.sessao();
  assert.deepStrictEqual(await Promise.all([1, 2, 3, 4, 5].map(() => s.token())), ['ID-1', 'ID-1', 'ID-1', 'ID-1', 'ID-1']);
  assert.strictEqual(n, 1);
  assert.strictEqual(await s.token(), 'ID-1'); assert.strictEqual(n, 1, 'em memoria, sem ir ao servidor');
  d.agora += 3600 * 1000;
  assert.strictEqual(await s.token(), 'ID-2', 'vencido: renova');
  // cinco chamadas levaram 401 com o mesmo token: um refresh para todas, inclusive a que chega depois de ele terminar
  const r = await Promise.all([1, 2, 3, 4].map(() => s.recusado('ID-2')));
  assert.deepStrictEqual(r, ['ID-3', 'ID-3', 'ID-3', 'ID-3']);
  assert.strictEqual(await s.recusado('ID-2'), 'ID-3');
  assert.strictEqual(n, 3);
  assert.strictEqual(d.abertas.length, 0);
  // 401 e o refresh recusado: sem sessao, uma vez, sem abrir a aba
  d.login = () => resp(400, { error: 'invalid_grant' });
  await semSessao(s.recusado('ID-3'));
  assert.deepStrictEqual([d.avisos, d.abertas.length], [['fim'], 0]);
});

teste('A-T22e, A-T16a: no app, faltando o Cofre, o App ou o Browser, e erro: nada em localStorage, nenhum /oauth2/token, nenhum login do navegador', async () => {
  for (const falta of ['Cofre', 'App', 'Browser']) {
    const d = aparelho({ plugins: { [falta]: null } }), s = d.sessao();
    assert.strictEqual(s.modo, 'app');
    await assert.rejects(s.token(), (e) => e.codigo === 'sem_cofre', falta);
    await assert.rejects(s.entrar(), (e) => e.codigo === 'sem_cofre', falta);
    assert.deepStrictEqual([d.ls.escritas, d.ss.escritas, d.pedidos, d.abertas, d.env.location.href], [[], [], [], [], ''], falta);
    assert.ok(d.avisos.includes('cofre'));
  }
});

teste('A-T22g: o Cofre que falha ao ler vira "sem sessao" (nunca um beco) e o que havia e apagado', async () => {
  const d = aparelho({ cofre: new Map([['refresh', 'R0']]) });
  const s = d.sessao();
  d.cofreQuebrado = true;
  await semSessao(s.token());
  assert.deepStrictEqual([d.avisos, d.cofre.size, d.pedidos.length], [['nova'], 0, 0]);
});

teste('o toque em Entrar duas vezes seguidas abre uma aba so; um segundo login troca o state e o retorno do primeiro nao vale', async () => {
  const d = aparelho(), s = d.sessao();
  await Promise.all([s.entrar(), s.entrar()]);
  assert.strictEqual(d.abertas.length, 1);
  const primeiro = d.estadoDaAba();
  await s.entrar();
  assert.notStrictEqual(d.estadoDaAba(), primeiro);
  d.ouvintes.appUrlOpen({ url: RET + '?code=C1&state=' + primeiro });
  await esperar();
  assert.strictEqual(d.pedidos.length, 0);
});

teste('a troca do code que falha (rede ou recusa) nao deixa sessao pela metade: "nao concluiu", PKCE apagado, sem recarga', async () => {
  for (const modo of ['rede', 'recusa', 'sem refresh']) {
    const d = aparelho({ login: () => (modo === 'recusa' ? resp(400, { error: 'invalid_grant' }) : resp(200, { id_token: 'X', expires_in: 60 })) });
    d.semRede = modo === 'rede';
    const s = d.sessao();
    await s.entrar();
    d.ouvintes.appUrlOpen({ url: RET + '?code=C1&state=' + d.estadoDaAba() });
    await esperar();
    assert.deepStrictEqual([d.cofre.size, d.recargas, d.avisos.slice(-1)], [0, 0, ['falhou']], modo);
    await semSessao(s.token());
  }
});

// ---------- navegador e PWA: como sempre ----------
teste('A-T16, A-T16a: no navegador o fluxo e o de antes e nenhum plugin e chamado', async () => {
  const d = aparelho({ navegador: true }), s = d.sessao();
  assert.strictEqual(s.modo, 'navegador');
  await pendente(s.token()); // sem sessao: vai ao login na mesma aba e a pagina vai embora
  const u = new URL(d.env.location.href);
  assert.strictEqual(u.origin + u.pathname, 'https://login.teste/oauth2/authorize');
  assert.strictEqual(u.searchParams.get('redirect_uri'), 'https://front.teste/');
  const pk = JSON.parse(d.ss.m.get('financas.pkce'));
  assert.strictEqual(pk.state, u.searchParams.get('state'));
  // a volta com o code
  const v = aparelho({ navegador: true, search: '?code=C1&state=' + pk.state });
  v.ss.m.set('financas.pkce', JSON.stringify(pk));
  assert.strictEqual(await v.sessao().token(), 'ID-1');
  assert.deepStrictEqual([v.pedidos[0].p.code_verifier, v.pedidos[0].p.redirect_uri], [pk.v, 'https://front.teste/']);
  const a = JSON.parse(v.ls.m.get('financas.auth'));
  assert.deepStrictEqual([a.id, a.refresh], ['ID-1', 'REFRESH-1']);
  assert.strictEqual(await v.sessao().token(), 'ID-1', 'valido: nao vai ao servidor');
  assert.strictEqual(v.pedidos.length, 1);
  // refresh recusado (400), fora do ar (503) e 401 da API levam ao login, como hoje
  for (const st of [400, 503]) {
    const r = aparelho({ navegador: true, login: () => resp(st, { error: 'x' }) });
    r.ls.m.set('financas.auth', JSON.stringify({ id: 'velho', refresh: 'R0', exp: 0 }));
    await pendente(r.sessao().token());
    assert.ok(r.env.location.href.includes('/oauth2/authorize'), 'refresh ' + st);
  }
  const q = aparelho({ navegador: true });
  q.ls.m.set('financas.auth', JSON.stringify({ id: 'x', refresh: 'R0', exp: q.agora + 1000 }));
  await pendente(q.sessao().recusado('x'));
  assert.strictEqual(q.ls.m.has('financas.auth'), false);
  assert.ok(q.env.location.href.includes('/oauth2/authorize'));
  // sair: apaga o token e vai ao logout com o endereco do site
  const o = aparelho({ navegador: true });
  o.ls.m.set('financas.auth', '{}');
  o.sessao().sair();
  assert.strictEqual(o.ls.m.has('financas.auth'), false);
  assert.strictEqual(new URL(o.env.location.href).searchParams.get('logout_uri'), 'https://front.teste/');
  assert.strictEqual(d.cofre.size + v.cofre.size + o.cofre.size, 0, 'o Cofre nunca e tocado no navegador');
});

// ---------- o api.js por cima do modulo ----------
function comApi(sessao, servidor) {
  const ctx = { console, setTimeout, clearTimeout, setInterval: () => 0, URLSearchParams, JSON, Promise, Set, Map, Date, Error, Object,
    localStorage: guarda(), document: { visibilityState: 'visible', addEventListener() {} },
    fetch: async (url, o) => { const r = servidor(String(url).replace('https://api.teste', ''), o); return { status: r.status, ok: r.status < 400, json: async () => r.body || {} }; },
    FINANCAS_CONFIG: CFG, Sessao: sessao };
  ctx.window = ctx;
  vm.createContext(ctx);
  vm.runInContext(fs.readFileSync(path.join(__dirname, 'api.js'), 'utf8'), ctx);
  return ctx.financasApi;
}
teste('A-T15, A-T22j: 401 da API faz um refresh e repete a chamada uma vez; um segundo 401 vira erro na tela, sem login', async () => {
  let n = 0;
  const d = aparelho({ cofre: new Map([['refresh', 'R0']]), login: () => { n++; return resp(200, tokens({ id_token: 'ID-' + n, refresh_token: undefined })); } });
  const vistos = [];
  let api = comApi(d.sessao(), (u, o) => { vistos.push(o.headers.authorization); return o.headers.authorization === 'Bearer ID-1' ? { status: 401 } : { status: 200, body: { sub: 's', email: 'a@b.c', espacos: [{ id: 'e1', role: 'owner' }] } }; });
  assert.ok(await api.use('db'), 'a chamada repetida passou');
  assert.deepStrictEqual(vistos, ['Bearer ID-1', 'Bearer ID-2']);
  // o servidor recusa tambem o token novo: erro, sem terceira tentativa e sem aba de login
  const d2 = aparelho({ cofre: new Map([['refresh', 'R0']]) });
  let pedidos = 0;
  api = comApi(d2.sessao(), () => { pedidos++; return { status: 401 }; });
  assert.strictEqual(await api.use('db'), null);
  assert.strictEqual(api.erroDeAcesso().code, 401);
  assert.deepStrictEqual([pedidos, d2.abertas.length, d2.avisos, d2.cofre.get('refresh')], [2, 0, [], 'REFRESH-1'], 'a sessao continua guardada');
});

// ---------- o que nao e JavaScript: pacote, manifest, plugin e template ----------
teste('A-T22f, A-T18: log do Capacitor desligado, esquema por applicationId, Cofre fora do backup e registrado, um URL de retorno por ambiente', () => {
  const raiz = path.resolve(__dirname, '..');
  const ler = (...p) => fs.readFileSync(path.join(raiz, ...p), 'utf8').replace(/\r\n/g, '\n');
  const cap = JSON.parse(ler('app-android', 'capacitor.config.json'));
  assert.strictEqual(cap.loggingBehavior, 'none', 'R18e');
  const pkg = JSON.parse(ler('app-android', 'package.json'));
  assert.ok(pkg.dependencies['@capacitor/browser'], 'Custom Tabs');
  assert.deepStrictEqual(Object.keys(pkg.dependencies).sort(), ['@capacitor/android', '@capacitor/app', '@capacitor/browser', '@capacitor/core'], 'nenhum plugin de terceiros (R9a)');
  const man = ler('app-android', 'android', 'app', 'src', 'main', 'AndroidManifest.xml');
  assert.match(man, /android:allowBackup="false"/);
  assert.match(man, /android:dataExtractionRules="@xml\/regras_de_extracao"/);
  assert.match(man, /android:fullBackupContent="@xml\/regras_de_backup"/);
  assert.match(man, /<data android:scheme="\$\{applicationId\}" android:host="auth" ?\/>/);
  assert.strictEqual(/custom_url_scheme/.test(man), false, 'R18f');
  for (const f of ['regras_de_extracao.xml', 'regras_de_backup.xml']) assert.match(ler('app-android', 'android', 'app', 'src', 'main', 'res', 'xml', f), /domain="sharedpref" path="cofre\.xml"/, f);
  const java = ['app-android', 'android', 'app', 'src', 'main', 'java', 'br', 'com', 'cicerotec', 'financas'];
  assert.match(ler(...java, 'MainActivity.java'), /registerPlugin\(CofrePlugin\.class\)/, 'R18d');
  const plugin = ler(...java, 'CofrePlugin.java');
  assert.match(plugin, /@CapacitorPlugin\(name = "Cofre"\)/);
  assert.match(plugin, /AndroidKeyStore/); assert.match(plugin, /AES\/GCM\/NoPadding/);
  assert.deepStrictEqual((plugin.match(/@PluginMethod[\s\S]*?public void (\w+)/g) || []).map((m) => m.split(' ').pop()).sort(), ['apagar', 'gravar', 'ler'], 'so tres operacoes');
  assert.strictEqual(/Log\.|Logger\./.test(plugin), false, 'o plugin nao escreve log');
  const t = ler('template.yaml');
  const app = "!If [EhDev, 'br.com.cicerotec.financas.dev://auth', 'br.com.cicerotec.financas://auth']";
  assert.ok(t.includes("CallbackURLs: [!Sub '${FrontOrigin}/', !Sub 'https://${Distribution.DomainName}/', " + app + ']'), 'CallbackURLs');
  assert.ok(t.includes("LogoutURLs: [!Sub '${FrontOrigin}/', !Sub 'https://${Distribution.DomainName}/', " + app + ']'), 'LogoutURLs');
  const html = ler('web', 'index.html');
  assert.ok(html.indexOf('<script src="js/sessao.js"></script>') > 0 && html.indexOf('<script src="js/sessao.js"></script>') < html.indexOf('<script src="api.js"></script>'), 'sessao.js antes do api.js (R18n)');
});

(async () => {
  for (const [nome, fn] of fila) {
    try { await fn(); console.log('ok   ' + nome); } catch (e) { falhas++; console.error('FALHOU: ' + nome + '\n  ' + (e.stack || e.message).split('\n').slice(0, 4).join('\n  ')); }
  }
  if (falhas) { console.error(falhas + ' teste(s) falharam'); process.exit(1); }
  console.log('sessao.test.js: ok');
})();
