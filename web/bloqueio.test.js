// Testes da trava por biometria (web/js/bloqueio.js), item 54 fase 3. Spec: .ai/specs/app-android/spec.md (A-T23 a A-T28).
// Demo: .ai/demos/demo-bloqueio-v1.html. Roda em Node, com relogio, armazenamento, elementos e plugin Trava de mentira.
// Rode com: node web/bloqueio.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const criarBloqueio = require('./js/bloqueio.js');
const criarSessao = require('./js/sessao.js');
const criarNativo = require('./js/nativo.js');

const tique = () => new Promise((r) => setTimeout(r, 0));
function aparelho(op) {
  op = op || {};
  const m = new Map(Object.entries(op.guardado || {}));
  const d = { m, agora: 1000, pedidos: [], janela: [], toasts: [], fetches: 0, saiu: 0, coberto: false, temTela: op.temTela !== false, resposta: op.resposta || 'ok' };
  const els = {};
  const el = (id) => els[id] || (els[id] = { id, hidden: id === 'bloqueio' || id === 'apSeg', textContent: '', checked: false, classes: new Set(), onclick: null, onchange: null,
    classList: { toggle(c, sim) { if (sim) els[id].classes.add(c); else els[id].classes.delete(c); } } });
  ['bloqueio', 'bqT', 'bqS', 'bqAbrir', 'bqSair', 'apSeg', 'swTrava', 'swRecentes', 'segMsg'].forEach(el);
  d.els = els;
  const recusa = (code) => Object.assign(new Error('x'), { code });
  const Trava = {
    disponivel: async () => ({ disponivel: d.temTela }),
    confirmar: async ({ titulo }) => {
      d.pedidos.push(titulo);
      const r = typeof d.resposta === 'function' ? await d.resposta() : d.resposta;
      if (r !== 'ok') throw recusa(r);
    },
    protegerJanela: async ({ ligar }) => { d.janela.push(ligar); }
  };
  d.env = {
    Capacitor: op.navegador ? undefined : { isNativePlatform: () => true, Plugins: op.semPlugin ? {} : { Trava } },
    el: (id) => els[id] || null,
    armazenamento: { getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => { m.set(k, String(v)); }, removeItem: (k) => { m.delete(k); } },
    agora: () => d.agora,
    fundo: (sim) => { d.coberto = sim; },
    toast: (t) => { d.toasts.push(t); },
    sair: () => { d.saiu++; }
  };
  d.b = criarBloqueio(d.env);
  return d;
}
const fora = async (d, ms) => { d.b.visibilidade(false); d.agora += ms; d.b.visibilidade(true); await tique(); await tique(); };

let falhas = 0;
const fila = [];
const teste = (nome, fn) => fila.push([nome, fn]);

teste('A-T23: sem a trava ligada nunca ha bloqueio, ao abrir nem depois de qualquer tempo fora', async () => {
  const d = aparelho();
  assert.strictEqual(d.b.iniciar(), true);
  assert.deepStrictEqual([d.b.bloqueado(), d.els.bloqueio.hidden, d.coberto], [false, true, false]);
  for (const ms of [1000, 59999, 60000, 3600000, 86400000]) { await fora(d, ms); assert.strictEqual(d.b.bloqueado(), false, ms + ' ms'); }
  assert.deepStrictEqual(d.pedidos, []);
});

teste('A-T24: com a trava ligada, bloqueia ao abrir; voltar antes de 1 minuto nao bloqueia e a partir de 1 minuto bloqueia', async () => {
  const d = aparelho({ guardado: { 'financas.trava': '1' } });
  d.b.iniciar();
  assert.deepStrictEqual([d.b.bloqueado(), d.els.bloqueio.hidden, d.coberto], [true, false, true], 'bloqueado antes de qualquer resposta: nada do app e mostrado');
  await tique(); await tique();
  assert.deepStrictEqual([d.b.bloqueado(), d.els.bloqueio.hidden, d.coberto, d.pedidos.length], [false, true, false, 1], 'a confirmacao liberou');
  await fora(d, 59999);
  assert.deepStrictEqual([d.b.bloqueado(), d.pedidos.length], [false, 1]);
  d.resposta = 'cancelado';
  await fora(d, 60000);
  assert.deepStrictEqual([d.b.bloqueado(), d.els.bloqueio.hidden, d.pedidos.length], [true, false, 2]);
  // duas saidas seguidas sem voltar contam da primeira
  d.b.visibilidade(false); d.agora += 10; d.b.visibilidade(false);
  assert.strictEqual(d.b.bloqueado(), true);
});

teste('A-T25: cancelar, errar muitas vezes e nao ter bloqueio de tela mantem o bloqueio; so o ok libera', async () => {
  const d = aparelho({ guardado: { 'financas.trava': '1' }, resposta: 'cancelado' });
  d.b.iniciar(); await tique(); await tique();
  assert.deepStrictEqual([d.b.bloqueado(), d.coberto, d.els.bqT.textContent, d.els.bqS.textContent, d.els.bqAbrir.hidden], [true, true, '', 'Confirme que é você para abrir.', false]);
  d.resposta = 'bloqueado';
  d.els.bqAbrir.onclick(); await tique(); await tique();
  assert.deepStrictEqual([d.b.bloqueado(), d.els.bqT.textContent, d.els.bqT.classes.has('ruim'), d.els.bloqueio.classes.has('com-aviso')], [true, 'Muitas tentativas.', true, true]);
  d.temTela = false;
  d.els.bqAbrir.onclick(); await tique(); await tique();
  assert.deepStrictEqual([d.b.bloqueado(), d.els.bqT.textContent, d.els.bqAbrir.hidden, d.els.bqT.classes.has('ruim')], [true, 'Este aparelho não tem mais digital, rosto nem senha de tela.', true, false], 'so resta Sair da conta');
  assert.strictEqual(d.pedidos.length, 2, 'sem bloqueio de tela a janela do Android nem e pedida');
  d.els.bqSair.onclick();
  assert.strictEqual(d.saiu, 1);
  // um erro qualquer do plugin tambem nao libera; e sem o plugin, idem
  const q = aparelho({ guardado: { 'financas.trava': '1' }, resposta: () => { throw new Error('quebrou'); } });
  q.b.iniciar(); await tique(); await tique();
  assert.strictEqual(q.b.bloqueado(), true);
  const s = aparelho({ guardado: { 'financas.trava': '1' }, semPlugin: true });
  s.b.iniciar(); await tique(); await tique();
  assert.deepStrictEqual([s.b.bloqueado(), s.els.bqAbrir.hidden], [true, true], 'falha fechada');
  // enfim o ok
  d.temTela = true; d.resposta = 'ok';
  await d.b.desbloquear();
  assert.deepStrictEqual([d.b.bloqueado(), d.els.bloqueio.hidden, d.coberto], [false, true, false]);
});

teste('uma pergunta por vez: toques repetidos em Desbloquear e a volta ao app nao abrem duas janelas', async () => {
  let soltar;
  const d = aparelho({ guardado: { 'financas.trava': '1' }, resposta: () => new Promise((r) => { soltar = () => r('ok'); }) });
  d.b.iniciar(); await tique();
  d.els.bqAbrir.onclick(); d.els.bqAbrir.onclick();
  d.b.visibilidade(false); d.agora += 120000; d.b.visibilidade(true); // a janela do Android tira o app da frente
  await tique();
  assert.strictEqual(d.pedidos.length, 1);
  soltar(); await tique(); await tique();
  assert.strictEqual(d.b.bloqueado(), false);
});

teste('A-T26, A-T28: ligar e desligar pedem a confirmacao; se falhar, nada muda; a escolha fica so no aparelho', async () => {
  const d = aparelho({ resposta: 'cancelado' });
  d.b.iniciar();
  assert.strictEqual(d.b.ligarChaves(), true);
  assert.deepStrictEqual([d.els.apSeg.hidden, d.els.swTrava.checked, d.els.swRecentes.checked], [false, false, false]);
  d.els.swTrava.checked = true; await d.els.swTrava.onchange();
  assert.deepStrictEqual([d.els.swTrava.checked, d.m.has('financas.trava'), d.els.segMsg.textContent], [false, false, 'Não confirmado: a trava continua desligada.']);
  assert.deepStrictEqual(d.pedidos, ['Confirme que é você para ligar a trava']);
  d.resposta = 'bloqueado';
  d.els.swTrava.checked = true; await d.els.swTrava.onchange();
  assert.deepStrictEqual([d.els.swTrava.checked, d.els.segMsg.textContent, d.els.segMsg.classes.has('ruim')], [false, 'Muitas tentativas. Tente de novo em 30 segundos.', true]);
  d.temTela = false;
  d.els.swTrava.checked = true; await d.els.swTrava.onchange();
  assert.deepStrictEqual([d.els.swTrava.checked, d.els.segMsg.textContent], [false, 'Cadastre uma digital, um rosto ou uma senha de tela no Android para usar a trava.']);
  d.temTela = true; d.resposta = 'ok';
  d.els.swTrava.checked = true; await d.els.swTrava.onchange();
  assert.deepStrictEqual([d.els.swTrava.checked, d.m.get('financas.trava'), d.els.segMsg.textContent, d.toasts], [true, '1', '', ['✓ Trava ligada']]);
  assert.strictEqual(d.b.bloqueado(), false, 'ligar nao bloqueia na hora');
  // desligar tambem pede; cancelado, continua ligada
  d.resposta = 'cancelado';
  d.els.swTrava.checked = false; await d.els.swTrava.onchange();
  assert.deepStrictEqual([d.els.swTrava.checked, d.m.get('financas.trava'), d.els.segMsg.textContent], [true, '1', 'Não confirmado: a trava continua ligada.']);
  assert.strictEqual(d.pedidos[d.pedidos.length - 1], 'Confirme que é você para desligar a trava');
  d.resposta = 'ok';
  d.els.swTrava.checked = false; await d.els.swTrava.onchange();
  assert.deepStrictEqual([d.els.swTrava.checked, d.m.has('financas.trava')], [false, false]);
  assert.strictEqual(d.fetches, 0);
  assert.strictEqual(/fetch\(|financasApi\.use|XMLHttpRequest/.test(fs.readFileSync(path.join(__dirname, 'js', 'bloqueio.js'), 'utf8')), false, 'o modulo nao fala com o servidor');
});

teste('esconder nos apps recentes: liga e desliga na hora, sem confirmacao, e a janela ja sobe protegida na proxima abertura', async () => {
  const d = aparelho();
  d.b.iniciar(); d.b.ligarChaves();
  d.els.swRecentes.checked = true; await d.els.swRecentes.onchange();
  assert.deepStrictEqual([d.janela, d.m.get('financas.recentes'), d.pedidos.length, d.els.swRecentes.checked], [[true], '1', 0, true]);
  d.els.swRecentes.checked = false; await d.els.swRecentes.onchange();
  assert.deepStrictEqual([d.janela, d.m.has('financas.recentes')], [[true, false], false]);
  const n = aparelho({ guardado: { 'financas.recentes': '1' } });
  n.b.iniciar(); await tique();
  assert.deepStrictEqual([n.janela, n.b.bloqueado()], [[true], false], 'a segunda chave nao depende da primeira');
});

teste('no navegador e na PWA nada aparece: sem bloqueio, sem a subsecao Seguranca, mesmo com as chaves guardadas', async () => {
  const d = aparelho({ navegador: true, guardado: { 'financas.trava': '1', 'financas.recentes': '1' } });
  assert.strictEqual(d.b.iniciar(), false);
  assert.strictEqual(d.b.ligarChaves(), false);
  await fora(d, 3600000);
  assert.deepStrictEqual([d.b.bloqueado(), d.els.bloqueio.hidden, d.els.apSeg.hidden, d.pedidos.length, d.janela.length], [false, true, true, 0, 0]);
});

teste('A-T27: sair apaga a preferencia da trava e a de esconder nos recentes', async () => {
  const m = new Map([['financas.trava', '1'], ['financas.recentes', '1'], ['financas.space', 'e1'], ['financas.tema', 'dark']]);
  const cofre = new Map([['refresh', 'R0']]);
  const s = criarSessao({
    cfg: { loginDomain: 'l', clientId: 'c' },
    Capacitor: { isNativePlatform: () => true, Plugins: {
      Cofre: { ler: async ({ nome }) => ({ valor: cofre.get(nome) || null }), gravar: async () => {}, apagar: async ({ nome }) => { cofre.delete(nome); } },
      App: { getInfo: async () => ({ id: 'x.y' }), getLaunchUrl: async () => undefined, addListener() {} },
      Browser: { open: async () => {} } } },
    localStorage: { getItem: (k) => m.get(k) || null, setItem: (k, v) => m.set(k, v), removeItem: (k) => m.delete(k) },
    recarregar() {}, agora: () => 0
  });
  await s.sair();
  assert.deepStrictEqual([...m.keys()], ['financas.tema']);
});

teste('com a tela de bloqueio na frente, o voltar fisico so tira o app da frente (nao mexe no que esta por baixo)', () => {
  const c = { ouv: {}, voltou: 0, min: 0, bloq: true };
  const App = { addListener(n, f) { c.ouv[n] = f; }, minimizeApp() { c.min++; } };
  criarNativo({ Capacitor: { isNativePlatform: () => true, Plugins: { App } }, history: { back() { c.voltou++; } }, bloqueado: () => c.bloq }).iniciar();
  c.ouv.backButton({ canGoBack: true });
  assert.deepStrictEqual([c.voltou, c.min], [0, 1]);
  c.bloq = false;
  c.ouv.backButton({ canGoBack: true });
  assert.deepStrictEqual([c.voltou, c.min], [1, 1]);
});

teste('a tela e o pacote: marcacao no comeco do body, subsecao escondida por padrao, CSS, plugin registrado e biblioteca de biometria', () => {
  const raiz = path.resolve(__dirname, '..');
  const ler = (...p) => fs.readFileSync(path.join(raiz, ...p), 'utf8').replace(/\r\n/g, '\n');
  const html = ler('web', 'index.html');
  const iBq = html.indexOf('id="bloqueio"'), iScript = html.indexOf('<script src="js/bloqueio.js"></script>'), iHeader = html.indexOf('<header>');
  assert.ok(iBq > html.indexOf('<script src="js/abertura.js"></script>') && iBq < iScript && iScript < iHeader, 'ordem: abertura, bloqueio, script do bloqueio, header');
  assert.ok(/<div class="bq" id="bloqueio"[^>]* hidden>/.test(html), 'a tela de bloqueio nasce escondida');
  assert.ok(html.includes('<div class="ap-sub" id="apSeg" hidden>'), 'a subsecao Seguranca nasce escondida (no navegador nunca aparece)');
  ['bqT', 'bqS', 'bqAbrir', 'bqSair', 'swTrava', 'swRecentes', 'segMsg'].forEach((id) => assert.ok(html.includes('id="' + id + '"'), 'falta #' + id));
  for (const frase of ['Bloquear com biometria', 'Esconder nos apps recentes', 'Vale só neste aparelho.', 'Também impede capturas e gravação de tela do app.']) assert.ok(html.includes(frase), frase);
  const css = ler('web', 'css', 'app.css');
  assert.ok(/\.bq\{[^}]*z-index:210/.test(css), 'acima da abertura');
  assert.ok(css.includes('html.bloqueado header,html.bloqueado main,html.bloqueado .ab{visibility:hidden;}'), 'bloqueado, nada do app e mostrado');
  assert.ok(css.includes('.ap-sub[hidden]{display:none;}'));
  const java = ['app-android', 'android', 'app', 'src', 'main', 'java', 'br', 'com', 'cicerotec', 'financas'];
  assert.match(ler(...java, 'MainActivity.java'), /registerPlugin\(TravaPlugin\.class\)/);
  const plugin = ler(...java, 'TravaPlugin.java');
  assert.match(plugin, /@CapacitorPlugin\(name = "Trava"\)/);
  assert.match(plugin, /BIOMETRIC_WEAK \| BiometricManager\.Authenticators\.DEVICE_CREDENTIAL/);
  assert.match(plugin, /FLAG_SECURE/);
  assert.deepStrictEqual((plugin.match(/@PluginMethod[\s\S]*?public void (\w+)/g) || []).map((x) => x.split(' ').pop()).sort(), ['confirmar', 'disponivel', 'protegerJanela']);
  assert.strictEqual(/Log\.|Logger\./.test(plugin), false, 'o plugin nao escreve log');
  assert.match(ler('app-android', 'android', 'app', 'build.gradle'), /androidx\.biometric:biometric:\$androidxBiometricVersion/);
  const pkg = JSON.parse(ler('app-android', 'package.json'));
  assert.deepStrictEqual(Object.keys(pkg.dependencies).sort(), ['@capacitor/android', '@capacitor/app', '@capacitor/browser', '@capacitor/core'], 'nenhum plugin de terceiros');
});

(async () => {
  for (const [nome, fn] of fila) {
    try { await fn(); console.log('ok   ' + nome); } catch (e) { falhas++; console.error('FALHOU: ' + nome + '\n  ' + (e.stack || e.message).split('\n').slice(0, 4).join('\n  ')); }
  }
  if (falhas) { console.error(falhas + ' teste(s) falharam'); process.exit(1); }
  console.log('bloqueio.test.js: ok');
})();
