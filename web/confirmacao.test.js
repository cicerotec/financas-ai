// Teste da confirmação de mês fechado (item 38) na camada web/api.js, em Node, com um servidor de mentira.
// O servidor decide (409 mes_fechado); o front só pergunta e reenvia com ?confirmaMesFechado=1.
// Rode com: node web/confirmacao.test.js
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const assert = require('assert');

const pedidos = [];            // { metodo, caminho, corpo }
let servidor = () => ({ status: 200, body: {} });
const el = () => new Proxy(function () {}, { get: (t, k) => (k === 'style' || k === 'classList' || k === 'dataset' ? new Proxy({}, { get: () => () => {}, set: () => true }) : () => el()), set: () => true });
const ctx = {
  console, setTimeout, setInterval: () => 0, clearTimeout, Promise, Error, JSON, Math, Date, Number, String, Array, Set, Map, Object, URLSearchParams, encodeURIComponent, parseInt, isNaN,
  localStorage: { getItem: (k) => (k === 'financas.auth' ? JSON.stringify({ id: 'token-de-teste', exp: Date.now() + 3600000 }) : null), setItem() {}, removeItem() {} },
  sessionStorage: { getItem() { return null; }, setItem() {} },
  location: { search: '', origin: 'http://localhost', pathname: '/' },
  history: { replaceState() {} },
  document: { getElementById: () => el(), querySelector: () => el(), createElement: () => el(), addEventListener() {}, visibilityState: 'visible', body: el() },
  alert: (m) => { throw new Error('alert: ' + m); },
  fetch: async (url, op) => {
    const caminho = url.replace('https://api.teste', '');
    const corpo = op.body ? JSON.parse(op.body) : undefined;
    pedidos.push({ metodo: op.method, caminho, corpo });
    const r = caminho === '/me' ? { status: 200, body: { sub: 's1', email: 'a@b.c', espacos: [{ id: 'sp1', role: 'owner' }] } } : servidor(op.method, caminho, corpo);
    return { status: r.status, ok: r.status < 400, json: async () => r.body };
  },
  FINANCAS_CONFIG: { apiUrl: 'https://api.teste', loginDomain: 'x', clientId: 'y' }
};
ctx.window = ctx;
vm.createContext(ctx);
vm.runInContext(fs.readFileSync(path.join(__dirname, 'api.js'), 'utf8'), ctx);

let perguntas = [], respostaPergunta = true, perguntasLote = [];
ctx.confirmarMesFechado = async (info, acao) => { perguntas.push({ mes: info.mes, acao }); return respostaPergunta; };
ctx.confirmarMesFechadoLote = async (n) => { perguntasLote.push(n); return respostaPergunta; };

const fechado = (mes) => ({ status: 409, body: { erro: 'o mes ja fechou', codigo: 'mes_fechado', banco: 'Nubank', mes, confirmavel: true } });
const ID = 'abc123';
const antigo = { id: ID, dataEvento: '2026-08-10T12:00:00.000Z', descricao: 'Mercado', valor: 180, tipo: 'saida', banco: 'Nubank' };
const zera = () => { pedidos.length = 0; perguntas = []; perguntasLote = []; respostaPergunta = true; };

(async () => {
  const db = await ctx.claude.use('db');
  const col = db.collection('lancamentos');
  zera();

  // lançar em mês fechado: 409, pergunta, reenvia com a confirmação
  servidor = (m, c) => (c.includes('confirmaMesFechado=1') ? { status: 201, body: Object.assign({}, antigo, { _seq: 1 }) } : fechado('2026-08'));
  const r = await col.add(Object.assign({}, antigo));
  assert.strictEqual(r.id, ID);
  assert.deepStrictEqual(pedidos.map((p) => p.caminho), ['/me', '/spaces/sp1/tx', '/spaces/sp1/tx?confirmaMesFechado=1'].slice(pedidos[0].caminho === '/me' ? 0 : 1));
  assert.deepStrictEqual(perguntas, [{ mes: '2026-08', acao: 'Lançar' }]);
  assert(!JSON.stringify(pedidos[pedidos.length - 1].corpo).includes('confirma'), 'a confirmacao vai na query, nunca no corpo');

  // mês aberto: sem pergunta
  zera();
  servidor = () => ({ status: 201, body: { id: 'novo1', _seq: 2 } });
  await col.add({ id: 'novo1', dataEvento: '2026-10-10T12:00:00.000Z' });
  assert.strictEqual(perguntas.length, 0);
  assert.strictEqual(pedidos.length, 1);

  // cancelar: uma tentativa só, erro "cancelado", nada reenviado
  zera(); respostaPergunta = false;
  servidor = () => fechado('2026-08');
  await assert.rejects(col.add({ id: 'outro', dataEvento: '2026-08-10T12:00:00.000Z' }), (e) => e.codigo === 'cancelado' && /nada foi alterado/.test(e.message));
  assert.strictEqual(pedidos.length, 1);

  servidor = () => ({ status: 201, body: Object.assign({}, antigo, { id: 'ed1', _seq: 2 }) });
  await col.add({ id: 'ed1' });
  // editar: o ?de= e a confirmação juntos; o Desfazer do mesmo lançamento reenvia direto, sem perguntar de novo
  zera();
  servidor = (m, c) => (c.includes('confirmaMesFechado=1') ? { status: 200, body: Object.assign({}, antigo, { _seq: 3 }) } : fechado('2026-08'));
  await col.doc('ed1').update({ valor: 170 });
  assert.strictEqual(pedidos[0].caminho, '/spaces/sp1/tx/ed1?de=' + encodeURIComponent(antigo.dataEvento));
  assert.strictEqual(pedidos[1].caminho, '/spaces/sp1/tx/ed1?de=' + encodeURIComponent(antigo.dataEvento) + '&confirmaMesFechado=1');
  assert.deepStrictEqual(perguntas, [{ mes: '2026-08', acao: 'Alterar' }]);
  zera();
  await col.doc('ed1').update({ valor: 180 });
  assert.strictEqual(pedidos.length, 1, 'desfazer: um pedido so');
  assert(pedidos[0].caminho.endsWith('&confirmaMesFechado=1'));
  assert.strictEqual(perguntas.length, 0);

  // excluir: o DELETE devolve o corpo (com desfazer.versao); em mes fechado pergunta uma vez, e o Desfazer (restaurar) leva a confirmacao
  zera();
  servidor = () => ({ status: 201, body: Object.assign({}, antigo, { id: 'ex1', _seq: 4 }) });
  await col.add({ id: 'ex1' });
  zera();
  servidor = (m, c) => (c.includes('confirmaMesFechado=1') ? { status: 200, body: { ok: true, desfazer: { versao: 2 }, _seq: 5 } } : fechado('2026-08'));
  const del = await col.doc('ex1').delete();
  assert.strictEqual(del.desfazer.versao, 2);
  assert.deepStrictEqual(perguntas, [{ mes: '2026-08', acao: 'Excluir' }]);
  assert(pedidos[1].caminho.endsWith('?de=' + encodeURIComponent(antigo.dataEvento) + '&confirmaMesFechado=1'));
  zera();
  servidor = () => ({ status: 200, body: { id: 'ex1', _seq: 6 } });
  await db.restaurarVersao('ex1', del.desfazer.versao);
  assert.strictEqual(pedidos[0].caminho, '/spaces/sp1/tx/ex1/versoes/2/restaurar?confirmaMesFechado=1');
  assert.strictEqual(perguntas.length, 0);
  // excluir em mes aberto: sem pergunta e o Desfazer restaura sem confirmacao
  zera();
  servidor = () => ({ status: 201, body: Object.assign({}, antigo, { id: 'ex2', _seq: 7 }) });
  await col.add({ id: 'ex2' });
  servidor = () => ({ status: 200, body: { ok: true, desfazer: { versao: 3 }, _seq: 8 } });
  zera();
  assert.strictEqual((await col.doc('ex2').delete()).desfazer.versao, 3);
  assert.strictEqual(perguntas.length, 0);
  await db.restaurarVersao('ex2', 3);
  assert.strictEqual(pedidos[1].caminho, '/spaces/sp1/tx/ex2/versoes/3/restaurar');
  // excluir cancelado na pergunta: nada excluido
  servidor = () => ({ status: 201, body: Object.assign({}, antigo, { id: 'ex3', _seq: 9 }) });
  await col.add({ id: 'ex3' });
  zera(); respostaPergunta = false;
  servidor = (m, c) => (m === 'DELETE' ? fechado('2026-08') : { status: 200, body: {} });
  await assert.rejects(col.doc('ex3').delete(), (e) => e.codigo === 'cancelado');
  assert.strictEqual(pedidos.filter((p) => p.metodo === 'DELETE').length, 1);
  respostaPergunta = true;

  // mês conferido continua sem confirmação possível
  zera();
  servidor = () => ({ status: 409, body: { erro: 'o mes esta conferido', codigo: 'mes_conferido' } });
  await assert.rejects(col.add({ id: 'c1', dataEvento: '2026-07-10T12:00:00.000Z' }), (e) => e.codigo === 'mes_conferido');
  assert.strictEqual(perguntas.length, 0);
  assert.strictEqual(pedidos.length, 1);

  // restaurar: a query só vai quando a tela manda confirmar
  zera();
  servidor = () => ({ status: 200, body: { id: ID, _seq: 9 } });
  await db.restaurarVersao(ID, 1, true);
  await db.restaurarVersao('semconf', 1, false);
  assert.strictEqual(pedidos[0].caminho, '/spaces/sp1/tx/' + ID + '/versoes/1/restaurar?confirmaMesFechado=1');
  assert.strictEqual(pedidos[1].caminho, '/spaces/sp1/tx/semconf/versoes/1/restaurar');

  // importar: o arquivo vai sem confirmação; os de mês fechado ficam de fora e uma pergunta só reenvia apenas esses
  const itens = [0, 1, 2, 3].map((i) => ({ id: 'i' + i, dataEvento: '2026-0' + (i + 5) + '-10T12:00:00.000Z' }));
  const rej = [{ indice: 1, motivo: 'mes fechado', codigo: 'mes_fechado' }, { indice: 3, motivo: 'mes fechado', codigo: 'mes_fechado' }, { indice: 2, motivo: 'sem data', codigo: '' }];
  zera();
  servidor = (m, c, corpo) => (c.includes('confirmaMesFechado=1') ? { status: 200, body: { gravados: corpo.itens.length, rejeitados: [] } } : { status: 200, body: { gravados: 1, rejeitados: rej } });
  let res = await db.importarLancamentos(itens);
  const lotes = pedidos.filter((p) => p.caminho.includes('/tx/batch'));
  assert.strictEqual(lotes.length, 2);
  assert(!lotes[0].caminho.includes('confirma') && lotes[0].corpo.itens.length === 4);
  assert(lotes[1].caminho.endsWith('/tx/batch?confirmaMesFechado=1'));
  assert.deepStrictEqual(lotes[1].corpo.itens.map((x) => x.id), ['i1', 'i3']);
  assert.strictEqual(JSON.stringify(perguntasLote), "[2]");
  assert.strictEqual(res.gravados, 3);
  assert.strictEqual(JSON.stringify(res.rejeitados.map((x) => x.indice)), JSON.stringify([2]), 'sobra so o que nao e de mes fechado');

  // importar e recusar a pergunta: os de mês fechado continuam rejeitados e nada é reenviado
  zera(); respostaPergunta = false;
  res = await db.importarLancamentos(itens);
  assert.strictEqual(pedidos.filter((p) => p.caminho.includes('/tx/batch')).length, 1);
  assert.strictEqual(JSON.stringify(res.rejeitados.map((x) => x.indice).sort()), JSON.stringify([1, 2, 3]));

  // importar sem nenhum de mês fechado: não pergunta
  zera();
  servidor = () => ({ status: 200, body: { gravados: 4, rejeitados: [] } });
  res = await db.importarLancamentos(itens);
  assert.strictEqual(perguntasLote.length, 0);
  assert.strictEqual(res.gravados, 4);

  console.log('confirmacao.test.js: ok');
})().catch((e) => { console.error(e); process.exit(1); });
