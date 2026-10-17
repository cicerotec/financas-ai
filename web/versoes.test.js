// Teste da tela Histórico de versões (web/js/versoes.js) em Node, sem navegador: um DOM mínimo de mentira e uma API falsa.
// Rode com: node web/versoes.test.js
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const assert = require('assert');

// ---------- ambiente de mentira ----------
const escuta = { sec: {}, doc: {} };
const elementos = {};
function elemento(id) {
  return (elementos[id] = elementos[id] || {
    id, innerHTML: '', value: '', dataset: {}, style: {}, attrs: {},
    classList: { _h: id === 'modalOverlay', contains() { return this._h; }, add(c) { if (c === 'hidden') this._h = true; }, remove(c) { if (c === 'hidden') this._h = false; } },
    addEventListener(tipo, fn) { (id === 'tab-versoes' ? escuta.sec : {})[tipo] = fn; },
    setAttribute(k, v) { this.attrs[k] = v; }, remove() { delete elementos[this.id]; }, querySelector() { return null; }
  });
}
let largo = false;
const toasts = [];
const chamadas = [];
const ctx = {
  console, setTimeout, Number, String, Array, JSON, Math, Date, Intl, RegExp, Promise, Error,
  document: {
    getElementById: (id) => elementos[id] || (id === 'tab-versoes' || id === 'modalOverlay' ? elemento(id) : null),
    createElement: () => ({ id: '', className: '', innerHTML: '', attrs: {}, setAttribute(k, v) { this.attrs[k] = v; }, querySelector() { return null; }, remove() { delete elementos[this.id]; } }),
    body: { style: {}, dataset: { aba: 'versoes' }, appendChild(el) { elementos[el.id] = el; el.remove = () => { delete elementos[el.id]; }; } },
    addEventListener(tipo, fn) { escuta.doc[tipo] = fn; }
  },
  window: { matchMedia: () => ({ matches: largo, addEventListener() {} }) },
  esc: (t) => String(t == null ? '' : t).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;'),
  fmtMoeda: (v) => 'R$ ' + Number(v || 0).toFixed(2).replace('.', ','),
  toast: (msg, erro) => toasts.push({ msg, erro: !!erro }),
  confirmarModal: async (msg, rotulo) => { chamadas.push({ confirmou: msg, rotulo }); return ctx.__confirma; },
  comBotaoOcupado: async (btn, rotulo, fn) => fn(),
  mesAtualApp: () => '2026-10',
  somaMes: (ym, n) => { const [y, m] = ym.split('-').map(Number); const t = y * 12 + (m - 1) + n; return Math.floor(t / 12) + '-' + String(t % 12 + 1).padStart(2, '0'); },
  rotuloMes: (ym) => { const N = ['janeiro','fevereiro','março','abril','maio','junho','julho','agosto','setembro','outubro','novembro','dezembro']; const [y, m] = ym.split('-').map(Number); return N[m - 1][0].toUpperCase() + N[m - 1].slice(1) + ' de ' + y; },
  __confirma: true
};
ctx.window.document = ctx.document;
vm.createContext(ctx);
vm.runInContext(fs.readFileSync(path.join(__dirname, 'js', 'versoes.js'), 'utf8') + '\nthis.__v = {abrirVersoes, vsPreparar, vsMudarMes, vsRender, get s(){return {vsMes,vsLista,vsAberto,vsDados,vsErro,vsTrunc};}};', ctx);
const V = ctx.__v;

// ---------- API falsa (formatos do spec) ----------
const ID = 'k3x9a1m7q2w8e5r4t6y1';
const reg = (valor, extra) => Object.assign({ id: ID, dataEvento: '2026-10-10T12:00:00.000Z', descricao: 'Mercado', valor, tipo: 'saida', banco: 'Bradesco' }, extra || {});
const itensOutubro = [
  { id: ID, descricao: 'Mercado <b>do bairro</b>', valor: 187.5, tipo: 'saida', dataEvento: '2026-10-10T12:00:00.000Z', banco: 'Bradesco', status: 'PAGO',
    excluido: false, excluidoEm: null, acao: 'restore', versao: 4, em: '2026-10-15T12:00:00.000Z', porSub: 's1', porNome: 'Ana' },
  { id: 'ing1', descricao: 'Ingresso cancelado', valor: 89.9, tipo: 'saida', dataEvento: '2026-10-04T12:00:00.000Z', banco: 'Itau', status: 'PAGO',
    excluido: true, excluidoEm: '2026-10-12T16:15:00.000Z', acao: 'delete', versao: 3, em: '2026-10-12T16:15:00.000Z', porSub: 's1', porNome: '' }
];
const versoesDoItem = {
  id: ID, excluido: false, semHistorico: false,
  versoes: [
    { versao: 4, acao: 'restore', porSub: 's1', porNome: 'Ana', em: '2026-10-15T12:00:00.000Z', deVersao: 2, diff: [{ campo: 'excluidoEm', antes: '2026-10-14T18:40:00.000Z', depois: null }], registro: reg(187.5), restauravel: false, motivo: 'o lancamento ja esta nessa versao' },
    { versao: 3, acao: 'delete', porSub: 's1', porNome: 'Ana', em: '2026-10-14T18:40:00.000Z', deVersao: null, diff: [], registro: reg(187.5, { excluidoEm: '2026-10-14T18:40:00.000Z' }), restauravel: false, motivo: 'esta versao e a exclusao: restaure uma versao anterior' },
    { versao: 2, acao: 'update', porSub: 's1', porNome: 'Ana', em: '2026-10-14T16:02:00.000Z', deVersao: null, diff: [{ campo: 'valor', antes: 180, depois: 187.5 }], registro: reg(187.5), restauravel: false, motivo: 'o lancamento ja esta nessa versao' },
    { versao: 1, acao: 'insert', porSub: '', porNome: '', em: '2026-10-10T15:05:00.000Z', deVersao: null, diff: [], registro: reg(180), restauravel: true, motivo: '' }
  ]
};
let respostaRestaurar = { status: 200, body: { id: ID, _seq: 9 } };
ctx.db = {
  async historicoMes(mes, q) { chamadas.push({ historico: [mes, q] }); return mes === '2026-10' ? { mes, truncado: false, itens: itensOutubro } : { mes, truncado: mes === '2026-07', itens: [] }; },
  async versoesDoLancamento(id) { chamadas.push({ versoes: id }); return versoesDoItem; },
  async restaurarVersao(id, n, confirma) { chamadas.push({ restaurar: [id, n], confirma }); return respostaRestaurar; }
};

const clique = (a, d) => escuta.sec.click({ target: { closest: () => ({ dataset: Object.assign({ vs: a }, d || {}) }) } });
const cliqueDoc = (a, d) => escuta.doc.click({ target: { closest: () => ({ dataset: Object.assign({ vs: a }, d || {}) }) } });
const html = () => elemento('tab-versoes').innerHTML;
const tela = () => (elementos.vsTc || {}).innerHTML || '';
const espera = () => new Promise((r) => setTimeout(r, 5));

(async () => {
  V.vsPreparar(); // o DOMContentLoaded do navegador

  // celular: lista do mes atual
  V.abrirVersoes(); await espera();
  assert.strictEqual(V.s.vsMes, '2026-10');
  assert.deepStrictEqual(chamadas[0].historico, ['2026-10', '']);
  assert(html().includes('2 lançamentos mudaram em outubro de 2026'), 'resumo');
  assert(html().includes('vs-pill ex">Excluído'), 'pilula de excluido');
  assert(html().includes('Restaurado') && html().includes('versão 4'), 'restaurado e versao');
  assert(html().includes('Mercado &lt;b&gt;do bairro&lt;/b&gt;') && !html().includes('<b>do bairro</b>'), 'descricao escapada');
  assert(html().includes('autor desconhecido') || html().includes('alguém'), 'quem, sem nome');
  assert(/data-vs="prox"[^>]*disabled/.test(html().replace(/\n/g, ' ')), 'o mes atual nao tem proximo');

  // celular: versoes em tela cheia
  clique('abrir', { id: ID }); await espera();
  assert(tela().includes('tc-hd') && tela().includes('Versões'), 'tela cheia com cabecalho');
  assert(tela().includes('Valor:</span> <s>R$ 180,00</s> → R$ 187,50'), 'diff do valor');
  assert(tela().includes('Estado antes da primeira alteração registrada.'), 'v1 preguicosa');
  assert(tela().includes('restaurada da versão 2'), 'restore aponta a versao de origem');
  assert(tela().includes('Esta versao e a exclusao: restaure uma versao anterior.'), 'motivo da exclusao');
  assert(tela().includes('O lancamento ja esta nessa versao.'), 'motivo ja igual');
  assert.strictEqual((tela().match(/data-vs="restaurar"/g) || []).length, 1, 'so a v1 pode restaurar');
  assert((tela().match(/class="vs-rb" disabled/g) || []).length === 3, 'as outras ficam desabilitadas');
  assert.strictEqual(elementos.vsTc.attrs.role, 'dialog');

  // restaurar com confirmacao: chama a API, avisa, recarrega as versoes e a lista
  chamadas.length = 0; toasts.length = 0;
  cliqueDoc('restaurar', { n: '1' }); await espera(); await espera();
  assert(chamadas[0].confirmou.includes('Restaurar a versão 1'), 'pediu confirmacao');
  assert.deepStrictEqual(chamadas[1].restaurar, [ID, 1]);
  assert(toasts[0].msg.includes('Restaurado da versão 1') && !toasts[0].erro, 'aviso de sucesso');
  assert(chamadas.some((c) => c.versoes === ID) && chamadas.some((c) => c.historico), 'recarrega versoes e lista');

  assert.strictEqual(chamadas[1].confirma, false, 'mes aberto: restaura sem confirmacao');
  assert(!tela().includes('vai pedir confirmacao') && chamadas[0].rotulo === undefined, 'mes aberto: janela normal');

  // mes fechado (o servidor avisa com exigeConfirmacao): aviso no cartao, uma janela so, restaura com a confirmacao
  versoesDoItem.versoes[3].exigeConfirmacao = true;
  await V.s.vsDados; chamadas.length = 0;
  clique('abrir', { id: ID }); await espera();
  assert(tela().includes('Mês fechado: vai pedir confirmação.'), 'aviso no cartao da versao');
  cliqueDoc('restaurar', { n: '1' }); await espera(); await espera();
  const jan = chamadas.find((c) => c.confirmou);
  assert(jan.confirmou.includes('já fechou') && jan.rotulo === 'Restaurar mesmo assim', 'janela unica com o aviso');
  assert.strictEqual(chamadas.filter((c) => c.confirmou).length, 1);
  assert.strictEqual(chamadas.find((c) => c.restaurar).confirma, true, 'envia a confirmacao');
  delete versoesDoItem.versoes[3].exigeConfirmacao;

  // cancelar a confirmacao nao chama a API
  chamadas.length = 0; ctx.__confirma = false;
  cliqueDoc('restaurar', { n: '1' }); await espera();
  assert(!chamadas.some((c) => c.restaurar), 'cancelou: nada foi restaurado');
  ctx.__confirma = true;

  // o servidor recusa (409): so mostra a mensagem dele
  respostaRestaurar = { status: 409, body: { erro: 'o mes 2026-09 do banco ITAU esta conferido (ate 2026-09) e travado: reabra o mes na aba Saldos para alterar', codigo: 'mes_conferido' } };
  chamadas.length = 0; toasts.length = 0;
  cliqueDoc('restaurar', { n: '1' }); await espera(); await espera();
  assert(toasts[0].erro && toasts[0].msg.includes('esta conferido'), 'mostra o erro do servidor');
  assert(!chamadas.some((c) => c.versoes), 'recusado: nao recarrega');
  respostaRestaurar = { status: 200, body: { id: ID, _seq: 10 } };

  // voltar fecha a tela cheia
  cliqueDoc('voltar'); await espera();
  assert(!elementos.vsTc, 'a tela cheia fechou');

  // mes anterior e mes vazio / truncado
  chamadas.length = 0;
  clique('ant'); await espera();
  assert.deepStrictEqual(chamadas[0].historico, ['2026-09', '']);
  assert(html().includes('Nenhuma alteração em setembro de 2026'), 'mes vazio');
  V.vsMudarMes('2026-07'); await espera();
  V.vsMudarMes('2026-11'); await espera(); // futuro: ignorado
  assert.strictEqual(V.s.vsMes, '2026-07');
  V.vsMudarMes('2026-10'); await espera();

  // busca de texto
  elemento('vsQIn').value = ' mercado ';
  global.document = ctx.document; ctx.document.getElementById = ((orig) => (id) => id === 'vsQIn' ? elementos.vsQIn : orig(id))(ctx.document.getElementById);
  chamadas.length = 0;
  clique('buscar'); await espera();
  assert.deepStrictEqual(chamadas[0].historico, ['2026-10', 'mercado']);

  // computador: tabela e painel
  largo = true; chamadas.length = 0;
  V.vsRender();
  assert(html().includes('vs-tab') && html().includes('vs-painel') && html().includes('Escolha um lançamento para ver as versões.'), 'tabela e painel vazio');
  clique('abrir', { id: ID }); await espera();
  assert(html().includes('Restaurar esta versão') && !elementos.vsTc, 'painel ao lado, sem tela cheia');
  largo = false;

  // erro de rede
  ctx.db.historicoMes = async () => { throw new Error('Failed to fetch'); };
  V.vsMudarMes('2026-08'); await espera();
  assert(html().includes('vs-aviso') && html().includes('Failed to fetch'), 'erro aparece na tela');

  // resposta velha nao apaga a nova (corrida)
  let soltarVelha;
  ctx.db.historicoMes = (mes) => mes === '2026-06'
    ? new Promise((r) => { soltarVelha = () => r({ mes, truncado: false, itens: itensOutubro }); })
    : Promise.resolve({ mes, truncado: false, itens: [] });
  V.vsMudarMes('2026-06');
  V.vsMudarMes('2026-05'); await espera();
  soltarVelha(); await espera();
  assert.strictEqual(V.s.vsMes, '2026-05');
  assert(html().includes('Nenhuma alteração em maio de 2026'), 'a resposta de junho chegou tarde e foi ignorada');

  console.log('versoes.test.js: ok');
})().catch((e) => { console.error(e); process.exit(1); });
