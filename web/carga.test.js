// Teste da carga por tela (web/api.js): cada tela le so o que e dela, e nada recarrega sem motivo.
// Nasceu de uma inspecao (2026-10-27): na tela de Lancar o app lia tambem os 1000 lancamentos, Futuros e as configuracoes de
// outras telas, e a cada gravacao ou a cada 3 minutos recarregava tudo. Falha se a subida voltar a ler o que nao e do Lancar.
// Roda em Node, sem navegador: um ambiente de mentira e um fetch que so anota as chamadas.
// Rode com: node web/carga.test.js
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const assert = require('assert');

const chamadas = [];
const resposta = (url) => {
  if (url.endsWith('/me')) return { sub: 's1', email: 'a@b.c', espacos: [{ id: 'e1', nome: 'Casa', role: 'owner' }] };
  if (url.includes('/seq')) return { seq: 5 };
  if (url.includes('/tx?')) return { itens: [], proximo: null };
  if (url.includes('/cfg/')) return { version: 1 };
  return {};
};
const ctx = {
  console,
  setTimeout, clearTimeout,
  URLSearchParams, URL, TextEncoder, Uint8Array, JSON, Promise, Set, Map, Date,
  btoa: (s) => Buffer.from(s, 'binary').toString('base64'),
  localStorage: { getItem: (k) => (k === 'financas.auth' ? JSON.stringify({ id: 't', exp: Date.now() + 3600000 }) : null), setItem() {}, removeItem() {} },
  sessionStorage: { getItem: () => null, setItem() {} },
  location: { search: '', origin: 'http://x', pathname: '/' },
  history: { replaceState() {} },
  fetch: async (url, opt) => {
    const u = String(url).replace('http://api.test', '');
    chamadas.push((opt && opt.method) + ' ' + u);
    return { status: 200, ok: true, json: async () => resposta(u) };
  }
};
let intervalo = null, aoVoltar = null;
ctx.setInterval = (f) => { intervalo = f; return 1; };
ctx.document = { visibilityState: 'visible', addEventListener: (t, f) => { if (t === 'visibilitychange') aoVoltar = f; } };
ctx.window = ctx;
ctx.FINANCAS_CONFIG = { apiUrl: 'http://api.test', loginDomain: 'l', clientId: 'c' };
vm.createContext(ctx);
vm.runInContext(fs.readFileSync(path.join(__dirname, 'api.js'), 'utf8'), ctx);

const esperar = (ms) => new Promise((r) => setTimeout(r, ms));
const quais = () => chamadas.splice(0).filter((c) => !c.endsWith('/me'));
let falhas = 0;
const teste = async (nome, fn) => {
  try { await fn(); console.log('ok   ' + nome); } catch (e) { falhas++; console.error('FALHOU: ' + nome + '\n  ' + e.message); }
};

(async () => {
  const db = await ctx.claude.use('db');
  const ignorar = () => {};

  await teste('a subida em Lancar le so listas, aparencia, tags e combos (nada de /tx, /seq nem /futuros)', async () => {
    chamadas.length = 0;
    db.doc('config/listas').onSnapshot(ignorar, ignorar);
    db.doc('config/tags').onSnapshot(ignorar, ignorar);
    db.doc('config/combos').onSnapshot(ignorar, ignorar);
    const lanc = db.collection('lancamentos').orderBy('dataEvento', 'desc').limit(1000).sobDemanda(ignorar, ignorar);
    await esperar(60);
    assert.deepStrictEqual(quais().sort(), ['GET /spaces/e1/aparencia', 'GET /spaces/e1/cfg/combos', 'GET /spaces/e1/cfg/listas', 'GET /spaces/e1/cfg/tags']);
    ctx.__lanc = lanc;
  });

  await teste('o timer de 3 minutos nao le nada enquanto nenhuma tela leu lancamentos', async () => {
    chamadas.length = 0;
    await intervalo();
    await esperar(260);
    assert.deepStrictEqual(quais(), []);
  });

  await teste('ativar le o seq e os lancamentos uma vez; ativar de novo nao le', async () => {
    chamadas.length = 0;
    await ctx.__lanc.ativar();
    assert.deepStrictEqual(quais(), ['GET /spaces/e1/seq', 'GET /spaces/e1/tx?ordem=desc&limite=1000']);
    await ctx.__lanc.ativar();
    assert.deepStrictEqual(quais(), []);
  });

  await teste('gravar tags relê so as tags', async () => {
    chamadas.length = 0;
    await db.adicionarTags(['x']);
    await esperar(260);
    assert.deepStrictEqual(quais(), ['POST /spaces/e1/tags', 'GET /spaces/e1/cfg/tags']);
  });

  await teste('com a tela fechada (pausar), uma mudanca nao le os lancamentos; ao voltar le uma vez', async () => {
    ctx.__lanc.pausar();
    chamadas.length = 0;
    await db.importarLancamentos([]); // avisa que os lancamentos mudaram
    await esperar(260);
    assert.deepStrictEqual(quais(), []);
    await ctx.__lanc.ativar();
    assert.deepStrictEqual(quais(), ['GET /spaces/e1/seq', 'GET /spaces/e1/tx?ordem=desc&limite=1000']);
  });

  await teste('o timer, com lancamentos lidos, confere so o seq (que nao mudou: nada mais)', async () => {
    chamadas.length = 0;
    await intervalo();
    await esperar(260);
    assert.deepStrictEqual(quais(), ['GET /spaces/e1/seq']);
  });

  await teste('quem quer saber das mudancas e avisado (Futuros)', async () => {
    let avisos = 0;
    db.aoMudarLancamentos(() => { avisos++; });
    await db.importarLancamentos([]);
    await esperar(260);
    assert.strictEqual(avisos, 1);
  });

  if (falhas) process.exit(1);
  console.log('carga.test.js: ok');
})();
