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
  const d = { cofre, abertas: [], pedidos: [], avisos: [], recargas: 0, ouvintes: {}, logs: [], agora: 1000000, ordem: [], ls: guarda(), ss: guarda(), launch: op.launch || null, timers: [] };
  d.login = op.login || (() => resp(200, tokens()));
  const Cofre = {
    ler: async ({ nome }) => { if (d.cofreQuebrado) throw new Error('keystore'); return { valor: cofre.has(nome) ? cofre.get(nome) : null }; },
    gravar: async ({ nome, valor }) => { if (d.cofreQuebrado) throw new Error('keystore'); cofre.set(nome, valor); d.ordem.push('gravar ' + nome); },
    apagar: async ({ nome }) => { if (d.apagarQuebrado) throw new Error('keystore'); cofre.delete(nome); d.ordem.push('apagar ' + nome); }
  };
  const App = { getInfo: async () => ({ id: ID }), getLaunchUrl: async () => (d.launch ? { url: d.launch } : undefined), addListener: (ev, f) => { d.ouvintes[ev] = f; } };
  const Browser = { open: async ({ url }) => { d.abertas.push(url); d.ordem.push('abrir'); }, addListener: (ev, f) => { d.ouvintes[ev] = f; } };
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
    setTimeout: (f) => { d.timers.push(f); },
    aviso: (t) => { d.avisos.push(t); }
  };
  d.sessao = () => criarSessao(d.env);
  d.estadoDaAba = () => new URL(d.abertas[d.abertas.length - 1]).searchParams.get('state');
  d.rodarTimers = async () => { const l = d.timers.splice(0); for (const f of l) await f(); await esperar(); };
  return d;
}
const semSessao = async (p) => { let e = null, parou = false; try { await Promise.race([p, esperar(200).then(() => { parou = true; })]); } catch (x) { e = x; } assert.ok(!parou, 'a promessa ficou parada: devia recusar com sem_sessao'); assert.ok(e && e.codigo === 'sem_sessao', 'devia recusar com sem_sessao, veio ' + (e && (e.codigo || e.message))); };
const pendente = async (p) => { let fim = false; p.then(() => { fim = true; }, () => { fim = true; }); await esperar(20); assert.strictEqual(fim, false, 'a promessa devia ficar parada (a pagina recarrega)'); };

let falhas = 0, concluido = false;
process.on('exit', () => { if (!concluido) { console.error('sessao.test.js: o teste nao terminou (uma promessa ficou parada)'); process.exitCode = 1; } });
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

// Push, P-T20 (spec app-android-push, R19): o aparelho sai do servidor ANTES de a sessao ser apagada; falhar nao impede o Sair
teste('P-T20: sair chama aoSair antes de apagar a sessao, e falha ou demora de aoSair nao impede o Sair', async () => {
  const d = aparelho({ cofre: new Map([['refresh', 'R0']]) });
  d.ls.m.set('financas.push', '{"s1":{}}');
  d.ls.m.set('financas.telegram', '{"s1":"123456789"}');
  d.env.aoSair = async () => { d.ordem.push('aoSair'); assert.strictEqual(d.cofre.size, 1, 'a sessao ainda existe quando o aparelho sai do servidor'); };
  await d.sessao().sair();
  assert.deepStrictEqual(d.ordem, ['aoSair', 'apagar refresh', 'apagar pkce', 'abrir'], 'o aparelho sai antes de apagar a sessao');
  assert.strictEqual(d.ls.m.has('financas.push'), false, 'financas.push some no Sair');
  assert.strictEqual(d.ls.m.has('financas.telegram'), false, 'o Id lembrado do Telegram some no Sair');
  assert.strictEqual(d.recargas, 1);
  const q = aparelho({ cofre: new Map([['refresh', 'R0']]) });
  q.env.aoSair = async () => { throw new Error('sem rede'); };
  await q.sessao().sair();
  assert.deepStrictEqual([q.cofre.size, q.abertas.length, q.recargas], [0, 1, 1], 'falha em aoSair: o Sair segue');
  const n = aparelho({ navegador: true });
  assert.strictEqual(typeof n.sessao().sair, 'function'); // no navegador e na PWA nada de push (o aoSair nem existe la)
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

// ---------- navegador e PWA ----------
teste('A-T16, A-T16a, M-T29b: no navegador, sem sessao, o login NAO abre sozinho: a tela de entrada aparece e so o toque em Entrar abre o login (R33a)', async () => {
  const d = aparelho({ navegador: true }), s = d.sessao();
  assert.strictEqual(s.modo, 'navegador');
  await semSessao(s.token());
  assert.deepStrictEqual([d.avisos, d.env.location.href, d.pedidos.length], [['nova'], '', 0], 'sem sessao: so o aviso da tela de entrada, nenhum redirecionamento');
  await pendente(s.entrar()); // o toque em Entrar: o login na mesma aba, e a pagina vai embora
  const u = new URL(d.env.location.href);
  assert.strictEqual(u.origin + u.pathname, 'https://login.teste/oauth2/authorize');
  assert.strictEqual(u.searchParams.get('redirect_uri'), 'https://front.teste/');
  const pk = JSON.parse(d.ss.m.get('financas.pkce'));
  assert.deepStrictEqual([pk.state, pk.tipo], [u.searchParams.get('state'), 'normal']);
  // a volta com o code
  const v = aparelho({ navegador: true, search: '?code=C1&state=' + pk.state });
  v.ss.m.set('financas.pkce', JSON.stringify(pk));
  assert.strictEqual(await v.sessao().token(), 'ID-1');
  assert.deepStrictEqual([v.pedidos[0].p.code_verifier, v.pedidos[0].p.redirect_uri], [pk.v, 'https://front.teste/']);
  const a = JSON.parse(v.ls.m.get('financas.auth'));
  assert.deepStrictEqual([a.id, a.refresh], ['ID-1', 'REFRESH-1']);
  assert.strictEqual(v.ss.m.has('financas.pkce'), false, 'o PKCE some no retorno');
  assert.strictEqual(await v.sessao().token(), 'ID-1', 'valido: nao vai ao servidor');
  assert.strictEqual(v.pedidos.length, 1);
  // PKCE de antes desta versao (sem tipo) vale como login normal
  const legado = aparelho({ navegador: true, search: '?code=C1&state=ST' });
  legado.ss.m.set('financas.pkce', JSON.stringify({ v: 'V', state: 'ST' }));
  assert.strictEqual(await legado.sessao().token(), 'ID-1', 'M-T7a');
  // refresh recusado (400): sessao encerrada e tela de entrada ("Sua sessao terminou"); 503 e rede mantem a sessao; nada redireciona
  const r400 = aparelho({ navegador: true, login: () => resp(400, { error: 'invalid_grant' }) });
  r400.ls.m.set('financas.auth', JSON.stringify({ id: 'velho', refresh: 'R0', exp: 0 }));
  await semSessao(r400.sessao().token());
  assert.deepStrictEqual([r400.avisos, r400.ls.m.has('financas.auth'), r400.env.location.href], [['fim'], false, '']);
  const r503 = aparelho({ navegador: true, login: () => resp(503) });
  r503.ls.m.set('financas.auth', JSON.stringify({ id: 'velho', refresh: 'R0', exp: 0 }));
  await assert.rejects(r503.sessao().token(), (e) => e.codigo === 'rede');
  assert.deepStrictEqual([r503.avisos, r503.ls.m.has('financas.auth'), r503.env.location.href], [[], true, '']);
  // 401 da API: sem sessao, tela de entrada, token apagado
  const q = aparelho({ navegador: true });
  q.ls.m.set('financas.auth', JSON.stringify({ id: 'x', refresh: 'R0', exp: q.agora + 1000 }));
  await semSessao(q.sessao().recusado('x'));
  assert.deepStrictEqual([q.avisos, q.ls.m.has('financas.auth'), q.env.location.href], [['fim'], false, '']);
  // a troca do code que falha, ou o retorno sem code, ou o state que nao bate: tela de entrada com "nao concluiu"; nunca um redirecionamento
  for (const [search, pkce, login] of [['?code=C1&state=ST', { v: 'V', state: 'ST', tipo: 'normal' }, () => resp(400, {})], ['?code=C1&state=OUTRO', { v: 'V', state: 'ST', tipo: 'normal' }], ['?error=access_denied&state=ST', { v: 'V', state: 'ST', tipo: 'normal' }], ['?code=C1&state=ST', null]]) {
    const f = aparelho({ navegador: true, search, login });
    if (pkce) f.ss.m.set('financas.pkce', JSON.stringify(pkce));
    await semSessao(f.sessao().token());
    assert.deepStrictEqual([f.avisos, f.env.location.href], [['falhou'], ''], search);
  }
  // sair: apaga o token e vai ao logout com o endereco do site (o Cognito volta ao site, que mostra a tela de entrada)
  const o = aparelho({ navegador: true });
  o.ls.m.set('financas.auth', '{}');
  o.sessao().sair();
  assert.strictEqual(o.ls.m.has('financas.auth'), false);
  assert.strictEqual(new URL(o.env.location.href).searchParams.get('logout_uri'), 'https://front.teste/');
  assert.strictEqual(d.cofre.size + v.cofre.size + o.cofre.size, 0, 'o Cofre nunca e tocado no navegador');
});

// ---------- MFA: o segundo login (spec mfa-opcional, R7 a R12) ----------
const comAcesso = (extra) => tokens(Object.assign({ access_token: 'ACESSO-1' }, extra || {}));
const loginMfa = (p) => (p.token ? resp(200, {}) : p.grant_type === 'refresh_token' ? resp(200, tokens({ refresh_token: undefined })) : resp(200, comAcesso())); // o revoke responde 200 vazio; a renovacao nao traz refresh novo
const revogacoes = (d) => d.pedidos.filter((x) => x.url.endsWith('/oauth2/revoke'));
const trocas = (d) => d.pedidos.filter((x) => x.url.endsWith('/oauth2/token') && x.p.grant_type === 'authorization_code');
const escopoDe = (url) => new URL(url).searchParams.get('scope');
const AUTH_OK = (d) => JSON.stringify({ id: 'ID-N', refresh: 'R-N', exp: d.agora + 3600 * 1000 });
// o navegador logo depois de voltar do segundo login: sessao normal guardada, PKCE do tipo mfa e o code na URL
function voltaMfa(op) {
  op = op || {};
  const d = aparelho({ navegador: true, search: op.search === undefined ? '?code=C2&state=SM' : op.search, login: op.login || loginMfa });
  d.ls.m.set('financas.auth', AUTH_OK(d));
  if (op.pkce !== null) d.ss.m.set('financas.pkce', JSON.stringify(op.pkce || { v: 'VM', state: 'SM', tipo: 'mfa', ate: d.agora + 5 * 60 * 1000 }));
  return d;
}

teste('M-T4, R7: o login NORMAL pede exatamente "openid email" (nunca o escopo de autoatendimento), no navegador e no app', async () => {
  const n = aparelho({ navegador: true });
  await pendente(n.sessao().entrar());
  const a = aparelho(), s = a.sessao();
  await s.entrar();
  for (const url of [n.env.location.href, a.abertas[0]]) {
    assert.strictEqual(escopoDe(url), 'openid email');
    assert.strictEqual(new URL(url).searchParams.get('prompt'), null, 'o login normal nao forca a senha');
    assert.ok(!url.includes('aws.cognito'), 'o escopo de autoatendimento nao aparece');
  }
  // 401 e sessao expirada tambem nao abrem login com o escopo novo (nem abrem login)
  const q = aparelho({ navegador: true });
  q.ls.m.set('financas.auth', JSON.stringify({ id: 'x', refresh: 'R0', exp: 0 }));
  q.login = () => resp(400, {});
  await semSessao(q.sessao().token());
  assert.strictEqual(q.env.location.href, '');
});

teste('M-T5, R8: o segundo login pede o escopo novo, com PKCE e state NOVOS a cada vez e o parametro que obriga a senha de novo; guarda so a intencao com prazo de 10 minutos', async () => {
  const d = aparelho({ navegador: true });
  await pendente(d.sessao().iniciarMfa());
  const u1 = new URL(d.env.location.href);
  assert.strictEqual(u1.origin + u1.pathname, 'https://login.teste/oauth2/authorize');
  assert.strictEqual(u1.searchParams.get('scope'), 'openid email aws.cognito.signin.user.admin');
  assert.strictEqual(u1.searchParams.get('prompt'), 'login', 'R8: reautenticar (confirmar no dev, Verificacao passo 2)');
  assert.strictEqual(u1.searchParams.get('code_challenge_method'), 'S256');
  const k1 = JSON.parse(d.ss.m.get('financas.pkce'));
  assert.deepStrictEqual([k1.tipo, k1.state === u1.searchParams.get('state'), k1.ate], ['mfa', true, d.agora + 10 * 60 * 1000]);
  assert.deepStrictEqual(Object.keys(k1).sort(), ['ate', 'state', 'tipo', 'v'], 'R12: a intencao nao leva token');
  const d2 = aparelho({ navegador: true });
  await pendente(d2.sessao().iniciarMfa());
  const u2 = new URL(d2.env.location.href);
  assert.notStrictEqual(u2.searchParams.get('state'), u1.searchParams.get('state'));
  assert.notStrictEqual(u2.searchParams.get('code_challenge'), u1.searchParams.get('code_challenge'));
  // no app: a aba do sistema, o tipo no Cofre
  const a = aparelho({ cofre: new Map([['refresh', 'R0']]) }), s = a.sessao();
  s.iniciarMfa().catch(() => {});
  await esperar();
  assert.strictEqual(escopoDe(a.abertas[0]), 'openid email aws.cognito.signin.user.admin');
  assert.strictEqual(new URL(a.abertas[0]).searchParams.get('prompt'), 'login');
  assert.strictEqual(JSON.parse(a.cofre.get('pkce')).tipo, 'mfa');
  assert.strictEqual(a.cofre.get('refresh'), 'R0', 'a sessao do app nao muda');
});

teste('M-T6, M-T7, R9, R10, R12: no navegador, o retorno do segundo login guarda SO o access token em memoria, nao toca na sessao, revoga o refresh e avisa a secao', async () => {
  const d = voltaMfa(), s = d.sessao();
  let abriu = 0;
  s.aoMfa(() => { abriu++; });
  const antes = d.ls.m.get('financas.auth');
  assert.strictEqual(await s.token(), 'ID-N', 'a sessao do app segue como estava');
  assert.strictEqual(abriu, 1, 'a secao e avisada para abrir');
  assert.strictEqual(s.tokenMfa(), 'ACESSO-1');
  assert.strictEqual(trocas(d).length, 1);
  assert.deepStrictEqual([trocas(d)[0].p.code, trocas(d)[0].p.code_verifier], ['C2', 'VM']);
  assert.strictEqual(d.ls.m.get('financas.auth'), antes, 'financas.auth nao foi tocado (nem o ID token nem o refresh do segundo login entram)');
  assert.deepStrictEqual(d.ls.escritas, [], 'nada gravado em localStorage');
  assert.deepStrictEqual(d.ss.escritas, [], 'nada gravado em sessionStorage');
  assert.strictEqual(d.ss.m.has('financas.pkce'), false, 'a intencao e apagada no retorno');
  assert.ok([...d.ls.m.values(), ...d.ss.m.values()].every((v) => !v.includes('ACESSO-1') && !v.includes('REFRESH-1')), 'nenhum armazenamento recebe o access token nem o refresh do segundo login');
  assert.strictEqual(revogacoes(d).length, 1, 'o refresh do segundo login e revogado');
  assert.deepStrictEqual([revogacoes(d)[0].p.token, revogacoes(d)[0].p.client_id], ['REFRESH-1', 'cliente']);
  assert.strictEqual(d.recargas, 0, 'a pagina nao recarrega');
  assert.deepStrictEqual(d.avisos, []);
  // a secao que ja estava aberta e se registra depois tambem e avisada; fechar a secao descarta o token
  let depois = 0;
  s.aoMfa(() => { depois++; });
  assert.strictEqual(depois, 1);
  s.descartarMfa();
  assert.strictEqual(s.tokenMfa(), null);
});

teste('M-T6, R9: o access token vence em memoria (1 hora menos a folga) e some ao descartar; chamadas juntas na volta trocam o code uma vez so', async () => {
  const d = voltaMfa(), s = d.sessao();
  await Promise.all([1, 2, 3, 4, 5].map(() => s.token()));
  assert.strictEqual(trocas(d).length, 1, 'o PKCE sai antes da espera: as outras chamadas nao o veem');
  assert.strictEqual(s.tokenMfa(), 'ACESSO-1');
  d.agora += 3600 * 1000;
  assert.strictEqual(s.tokenMfa(), null, 'vencido: descartado');
});

teste('M-T8, M-T9, R11, R12: cancelar, o state que nao bate, a intencao vencida e o erro do login deixam o app logado, sem token, e a secao NAO reabre', async () => {
  const casos = [
    ['voltou sem code (cancelou)', { search: '' }],
    ['erro do Cognito', { search: '?error=access_denied&state=SM' }],
    ['state que nao bate', { search: '?code=C2&state=OUTRO' }],
    ['intencao vencida (mais de 10 minutos)', { search: '?code=C2&state=SM', pkce: { v: 'VM', state: 'SM', tipo: 'mfa', ate: 1000000 - 1 } }],
    ['a troca do code falha', { login: () => resp(400, { error: 'invalid_grant' }) }],
    ['a resposta nao traz o access token', { login: () => resp(200, tokens()) }],
    ['sem rede na troca', { semRede: true }]
  ];
  for (const [nome, op] of casos) {
    const d = voltaMfa(op), s = d.sessao();
    d.semRede = !!op.semRede;
    let abriu = 0;
    s.aoMfa(() => { abriu++; });
    assert.strictEqual(await s.token(), 'ID-N', nome + ': a sessao do app segue logada');
    assert.deepStrictEqual([abriu, s.tokenMfa(), d.ss.m.has('financas.pkce'), d.avisos, d.env.location.href, d.recargas], [0, null, false, [], '', 0], nome);
    assert.strictEqual(revogacoes(d).length, 0, nome + ': sem token novo, nada a revogar');
  }
  // reabrir o app depois de cancelar nao reabre a secao: o PKCE do tipo mfa so sobrevive uma carga
  const d = voltaMfa({ search: '' });
  await d.sessao().token();
  const outra = aparelho({ navegador: true });
  outra.ls.m.set('financas.auth', AUTH_OK(outra));
  outra.ss = d.ss; outra.env.sessionStorage = d.ss;
  let abriu = 0;
  const s2 = outra.sessao();
  s2.aoMfa(() => { abriu++; });
  await s2.token();
  assert.deepStrictEqual([abriu, s2.tokenMfa()], [0, null]);
});

teste('M-T7a, R10: o login NORMAL continua como hoje; e o retorno "mfa" nunca grava refresh nem chama salvarAuth, mesmo sem sessao previa', async () => {
  // sem sessao previa, o retorno do segundo login nao cria sessao: cai na tela de entrada
  const d = aparelho({ navegador: true, search: '?code=C2&state=SM', login: loginMfa });
  d.ss.m.set('financas.pkce', JSON.stringify({ v: 'VM', state: 'SM', tipo: 'mfa', ate: d.agora + 1000 }));
  await semSessao(d.sessao().token());
  assert.deepStrictEqual([d.ls.escritas, d.avisos], [[], ['nova']], 'o segundo login nao serve de login normal');
});

// ---------- MFA no app Android ----------
teste('M-T5, M-T6, M-T7, R10, R11: no app, o retorno "mfa" entrega so o access token, nao grava refresh, nao recarrega, nao diz "entrando" e revoga o refresh', async () => {
  const d = aparelho({ cofre: new Map([['refresh', 'R0']]), login: loginMfa }), s = d.sessao();
  assert.strictEqual(await s.token(), 'ID-1');
  const espera = s.iniciarMfa();
  espera.catch(() => {});
  await esperar();
  const st = d.estadoDaAba();
  assert.strictEqual(escopoDe(d.abertas[0]), 'openid email aws.cognito.signin.user.admin');
  assert.strictEqual(JSON.parse(d.cofre.get('pkce')).tipo, 'mfa');
  d.ouvintes.appUrlOpen({ url: RET + '?code=C5&state=' + st });
  assert.strictEqual(await espera, 'ACESSO-1');
  assert.strictEqual(s.tokenMfa(), 'ACESSO-1');
  assert.deepStrictEqual([d.cofre.get('refresh'), d.cofre.has('pkce'), d.recargas, d.avisos], ['R0', false, 0, []], 'a sessao do app nao muda; o PKCE some');
  assert.deepStrictEqual([d.ls.escritas, d.ss.escritas], [[], []]);
  assert.ok(![...d.cofre.values()].some((v) => v.includes('ACESSO-1') || v.includes('REFRESH-1')), 'o Cofre nao recebe o access token nem o refresh do segundo login');
  assert.deepStrictEqual([trocas(d).length, revogacoes(d).length, revogacoes(d)[0].p.token], [1, 1, 'REFRESH-1']);
  assert.strictEqual(await s.token(), 'ID-1', 'a sessao do app segue valida');
  // sair descarta o access token da secao
  await s.sair();
  assert.strictEqual(s.tokenMfa(), null);
});

teste('M-T8, R11: no app, fechar a aba sem concluir cancela; o retorno invalido ou a troca que falha nao viram "sem sessao" nem encerram a sessao', async () => {
  // fechar a aba sem retorno
  const d = aparelho({ cofre: new Map([['refresh', 'R0']]), login: loginMfa }), s = d.sessao();
  await s.token();
  const espera = s.iniciarMfa();
  espera.catch(() => {});
  await esperar();
  d.ouvintes.browserFinished();
  await d.rodarTimers();
  await assert.rejects(espera, (e) => e.codigo === 'cancelado');
  assert.deepStrictEqual([d.cofre.has('pkce'), d.cofre.get('refresh'), d.avisos, s.tokenMfa()], [false, 'R0', [], null]);
  // a aba fecha MAS o retorno chega logo depois (o sucesso): nao cancela
  const e = aparelho({ cofre: new Map([['refresh', 'R0']]), login: loginMfa }), s2 = e.sessao();
  await s2.token();
  const espera2 = s2.iniciarMfa();
  espera2.catch(() => {});
  await esperar();
  e.ouvintes.browserFinished();
  e.ouvintes.appUrlOpen({ url: RET + '?code=C5&state=' + e.estadoDaAba() });
  await e.rodarTimers();
  assert.strictEqual(await espera2, 'ACESSO-1');
  // retorno invalido (state), troca recusada, sem rede: "mfa_falhou", sem aviso de sessao e sem mexer no refresh guardado
  for (const [nome, estado, login, semRede] of [['state errado', 'outro', loginMfa], ['troca recusada', null, () => resp(400, { error: 'invalid_grant' })], ['sem access token', null, () => resp(200, tokens())], ['sem rede', null, loginMfa, true]]) {
    const f = aparelho({ cofre: new Map([['refresh', 'R0']]), login: loginMfa }), t = f.sessao();
    await t.token();
    f.login = login; f.semRede = !!semRede;
    const ep = t.iniciarMfa();
    ep.catch(() => {});
    await esperar();
    f.ouvintes.appUrlOpen({ url: RET + '?code=C5&state=' + (estado || f.estadoDaAba()) });
    await assert.rejects(ep, (x) => x.codigo === 'mfa_falhou', nome);
    assert.deepStrictEqual([f.cofre.get('refresh'), f.cofre.has('pkce'), f.avisos, f.recargas, t.tokenMfa()], ['R0', false, [], 0, null], nome);
    f.login = loginMfa; f.semRede = false;
    assert.strictEqual(await t.token(), 'ID-1', nome + ': continua logado');
  }
  // dois segundos logins juntos: o segundo e recusado; a aba que nao abre vira "mfa_falhou" e limpa o PKCE
  const g = aparelho({ cofre: new Map([['refresh', 'R0']]), login: loginMfa }), u = g.sessao();
  await u.token();
  const primeiro = u.iniciarMfa();
  primeiro.catch(() => {});
  await esperar();
  await assert.rejects(u.iniciarMfa(), (x) => x.codigo === 'ocupado');
  const h = aparelho({ cofre: new Map([['refresh', 'R0']]), plugins: { Browser: { open: async () => { throw new Error('sem navegador'); } } } });
  await h.sessao().token();
  await assert.rejects(h.sessao().iniciarMfa(), (x) => x.codigo === 'mfa_falhou');
  assert.strictEqual(h.cofre.has('pkce'), false);
});

teste('M-T10, R13: o modulo nao escreve log e nenhum erro do segundo login leva token, code ou verifier', async () => {
  const d = voltaMfa(), s = d.sessao();
  const original = Object.assign({}, console);
  for (const k of ['log', 'info', 'warn', 'error', 'debug']) console[k] = (...a) => d.logs.push(a.join(' '));
  try {
    await s.token();
    const q = aparelho({ cofre: new Map([['refresh', 'R0']]), login: () => resp(400, {}) });
    const esp = q.sessao().iniciarMfa(); esp.catch(() => {});
    await esperar();
    q.ouvintes.appUrlOpen({ url: RET + '?code=SEGREDO&state=' + q.estadoDaAba() });
    await esp.catch((e) => { assert.ok(!/SEGREDO|REFRESH|ACESSO|VM/.test(e.message), 'o erro nao leva segredo'); });
  } finally { Object.assign(console, original); }
  assert.deepStrictEqual(d.logs, []);
  const fonte = fs.readFileSync(path.join(__dirname, 'js', 'sessao.js'), 'utf8');
  assert.strictEqual(/console\./.test(fonte), false);
  assert.strictEqual(/\.setItem\([^)]*(acesso|access_token)/i.test(fonte), false, 'o access token nunca vai a um armazenamento');
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
  concluido = true;
  if (falhas) { console.error(falhas + ' teste(s) falharam'); process.exit(1); }
  console.log('sessao.test.js: ok');
})();
