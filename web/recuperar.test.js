// Testes da tela "Recuperar o acesso" (web/js/recuperar.js). Spec: .ai/specs/mfa-opcional/spec.md (R27, R28, R31, R33, R34, M-T29b).
// Roda em Node, com elementos e fetch de mentira. Rode com: node web/recuperar.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const criarRecuperar = require('./js/recuperar.js');

const tique = () => new Promise((r) => setTimeout(r, 0));
function el(id, extra) {
  const e = Object.assign({ id, hidden: false, textContent: '', value: '', classes: new Set(), onclick: null, onkeydown: null, focado: 0 }, extra || {});
  e.classList = { toggle: (c, sim) => { if (sim) e.classes.add(c); else e.classes.delete(c); } };
  e.focus = () => { e.focado++; };
  return e;
}
function pagina(op) {
  op = op || {};
  const d = { fetches: [], entrou: 0, enderecoLimpo: 0, guardado: new Map() };
  const els = {};
  ['mfRec', 'mfRecVolta', 'mfRecForm', 'mfRecOk', 'mfRecEmail', 'mfRecCod', 'mfRecMsg', 'mfRecEnviar', 'mfRecEntrar', 'abPerdi'].forEach((id) => { els[id] = el(id, { hidden: ['mfRec', 'mfRecOk', 'mfRecEntrar'].includes(id) }); });
  d.els = els;
  d.resposta = op.resposta || { ok: true, status: 200 };
  d.env = {
    el: (id) => els[id] || null,
    url: 'http://api.test',
    fetch: async (u, o) => { d.fetches.push({ u, o }); if (d.resposta === 'rede') throw new TypeError('Failed to fetch http://api.test com ana@x.com'); return d.resposta; },
    endereco: () => op.hash || '',
    limparEndereco: () => { d.enderecoLimpo++; },
    entrar: () => { d.entrou++; }
  };
  d.r = criarRecuperar(d.env);
  d.r.ligar();
  return d;
}
const preencher = (d, e, c) => { d.els.mfRecEmail.value = e; d.els.mfRecCod.value = c; };

let falhas = 0;
const fila = [];
const teste = (nome, fn) => fila.push([nome, fn]);

teste('"Perdi o autenticador" abre a tela sem iniciar o login e sem chamar nada', async () => {
  const d = pagina();
  d.els.abPerdi.onclick();
  assert.strictEqual(d.els.mfRec.hidden, false);
  assert.strictEqual(d.els.mfRecForm.hidden, false);
  assert.strictEqual(d.entrou, 0, 'nunca inicia o login');
  assert.deepStrictEqual(d.fetches, []);
});

teste('o endereco #recuperar abre a tela direto; qualquer outro endereco nao', async () => {
  assert.strictEqual(pagina({ hash: '#recuperar' }).els.mfRec.hidden, false);
  assert.strictEqual(pagina({ hash: '#outra' }).els.mfRec.hidden, true);
  assert.strictEqual(pagina().els.mfRec.hidden, true);
  assert.strictEqual(pagina({ hash: '#recuperar' }).entrou, 0);
});

teste('envia so e-mail e codigo, sem cabecalho de autorizacao, por POST na rota publica', async () => {
  const d = pagina();
  d.els.abPerdi.onclick(); preencher(d, '  ana@exemplo.com ', ' AAAA-BBBB-CCCC-DDDD-EEEE ');
  await d.r.enviar();
  assert.strictEqual(d.fetches.length, 1);
  const { u, o } = d.fetches[0];
  assert.strictEqual(u, 'http://api.test/mfa/recuperar');
  assert.strictEqual(o.method, 'POST');
  assert.deepStrictEqual(Object.keys(o.headers).map((k) => k.toLowerCase()), ['content-type']);
  assert.deepStrictEqual(JSON.parse(o.body), { email: 'ana@exemplo.com', codigo: 'AAAA-BBBB-CCCC-DDDD-EEEE' });
  assert.ok(!u.includes('ana') && !u.includes('AAAA'), 'nada na URL');
});

teste('sucesso: mostra o aviso, troca o botao por "Entrar", limpa os campos; o login so abre no toque', async () => {
  const d = pagina();
  d.els.abPerdi.onclick(); preencher(d, 'ana@exemplo.com', 'AAAA-BBBB-CCCC-DDDD-EEEE');
  await d.r.enviar();
  assert.deepStrictEqual([d.els.mfRecForm.hidden, d.els.mfRecOk.hidden, d.els.mfRecEnviar.hidden, d.els.mfRecEntrar.hidden], [true, false, true, false]);
  assert.deepStrictEqual([d.els.mfRecEmail.value, d.els.mfRecCod.value], ['', ''], 'e-mail e codigo nao ficam nos campos');
  assert.strictEqual(d.entrou, 0);
  d.els.mfRecEntrar.onclick();
  assert.strictEqual(d.entrou, 1);
  assert.strictEqual(d.els.mfRec.hidden, true);
});

teste('qualquer resposta que nao seja o sucesso mostra a MESMA frase generica (o front nao distingue nem conta tentativas)', async () => {
  const frases = new Set();
  for (const status of [400, 401, 404, 429, 500, 503]) {
    const d = pagina({ resposta: { ok: false, status } });
    d.els.abPerdi.onclick(); preencher(d, 'ana@exemplo.com', 'AAAA-BBBB-CCCC-DDDD-EEEE');
    await d.r.enviar();
    frases.add(d.els.mfRecMsg.textContent);
    assert.ok(d.els.mfRecMsg.classes.has('ruim'));
    assert.strictEqual(d.els.mfRecForm.hidden, false, 'continua no formulario');
  }
  assert.deepStrictEqual([...frases], ['Confira o e-mail e o código. Depois de várias tentativas erradas, aguarde alguns minutos.']);
  const d = pagina({ resposta: { ok: false, status: 400 } });
  d.els.abPerdi.onclick(); preencher(d, 'a@b.c', 'X');
  await d.r.enviar(); await d.r.enviar(); await d.r.enviar();
  assert.strictEqual(d.fetches.length, 3, 'o servidor decide; o front nao bloqueia sozinho');
});

teste('sem conexao: frase propria e o texto do erro do fetch (que leva e-mail e URL) nunca aparece', async () => {
  const d = pagina({ resposta: 'rede' });
  d.els.abPerdi.onclick(); preencher(d, 'ana@exemplo.com', 'AAAA-BBBB-CCCC-DDDD-EEEE');
  await d.r.enviar();
  assert.ok(/sem conexão/.test(d.els.mfRecMsg.textContent));
  assert.ok(!/ana@|api\.test/.test(d.els.mfRecMsg.textContent));
});

teste('campo vazio: avisa na tela e nem chama o servidor', async () => {
  const d = pagina();
  d.els.abPerdi.onclick(); preencher(d, 'ana@exemplo.com', '   ');
  await d.r.enviar();
  assert.deepStrictEqual(d.fetches, []);
  assert.ok(d.els.mfRecMsg.classes.has('ruim'));
});

teste('clique duplo durante o envio manda uma chamada so', async () => {
  const d = pagina();
  d.els.abPerdi.onclick(); preencher(d, 'ana@exemplo.com', 'AAAA-BBBB-CCCC-DDDD-EEEE');
  const a = d.r.enviar(), b = d.r.enviar();
  await a; await b;
  assert.strictEqual(d.fetches.length, 1);
});

teste('voltar fecha a tela, apaga o que foi digitado e limpa o endereco #recuperar', async () => {
  const d = pagina({ hash: '#recuperar' });
  preencher(d, 'ana@exemplo.com', 'AAAA');
  d.els.mfRecVolta.onclick();
  assert.strictEqual(d.els.mfRec.hidden, true);
  assert.deepStrictEqual([d.els.mfRecEmail.value, d.els.mfRecCod.value], ['', '']);
  assert.strictEqual(d.enderecoLimpo, 1);
  assert.strictEqual(d.entrou, 0);
});

teste('nada e guardado: o modulo nao usa armazenamento, log nem console', async () => {
  const src = fs.readFileSync(path.join(__dirname, 'js', 'recuperar.js'), 'utf8').replace(/(^|\s)\/\/.*$/gm, '');
  for (const proibido of ['localStorage', 'sessionStorage', 'console.', 'document.cookie']) assert.ok(!src.includes(proibido), proibido);
});

teste('a tela de entrada tem "Perdi o autenticador" e a tela de recuperacao entra nas camadas do voltar', async () => {
  const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
  for (const id of ['abPerdi', 'mfRec', 'mfRecVolta', 'mfRecEmail', 'mfRecCod', 'mfRecEnviar', 'mfRecEntrar']) assert.ok(html.includes('id="' + id + '"'), id);
  assert.ok(html.includes('js/recuperar.js'));
  const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8').replace(/\r\n/g, '\n');
  const corpo = (nome) => { const i = app.indexOf('function ' + nome + '('); return app.slice(i, app.indexOf('\n}\n', i)); };
  assert.ok(corpo('camadasAbertas').includes("camadaVisivel('mfRec')"));
  assert.ok(corpo('fecharCamadaTopo').includes("'#mfRecVolta'"));
  const css = fs.readFileSync(path.join(__dirname, 'css', 'app.css'), 'utf8');
  assert.ok(/\.mf-rec\{z-index:210;\}/.test(css), 'acima da Abertura (200)');
});

(async () => {
  for (const [nome, fn] of fila) {
    const sem = setTimeout(() => { console.log('FALHOU (promessa parada): ' + nome); process.exit(1); }, 5000);
    try { await fn(); await tique(); console.log('ok   ' + nome); }
    catch (e) { falhas++; console.log('FALHOU ' + nome + '\n' + (e && e.stack)); }
    clearTimeout(sem);
  }
  if (falhas) { console.log(falhas + ' falha(s)'); process.exit(1); }
  console.log('todos passaram (' + fila.length + ')');
})();
