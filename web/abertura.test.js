// Teste da abertura (splash, web/js/abertura.js): cobre o app ate listas, tags e combos chegarem, fica no minimo 0,6 s, avisa aos 4 s
// e, se algo falhar, mostra o erro e nao sai. Demo: .ai/demos/demo-abertura-v1.html. Roda em Node, com relogio e elementos de mentira.
// Rode com: node web/abertura.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const criarAbertura = require('./js/abertura.js');

function ambiente(){
  let agora = 0;
  const fila = [];
  const els = {};
  const el = (id) => els[id] || (els[id] = { id, hidden: false, textContent: '', classes: new Set(), classList: null, onclick: null });
  ['abertura', 'abMsg', 'abErroT', 'abErroS', 'abDeNovo', 'abSair'].forEach((id) => { const e = el(id); e.classList = { add: (...c) => c.forEach((x) => e.classes.add(x)) }; });
  els.abSair.hidden = true;
  const env = {
    el: (id) => els[id] || null,
    agora: () => agora,
    setTimeout: (f, ms) => { fila.push({ em: agora + ms, f }); },
    fundo: (sim) => { env.coberto = sim; },
    recarregar: () => { env.recarregou = true; },
    sair: () => { env.saiu = true; }
  };
  const passar = (ms) => { // anda o relogio e roda o que venceu, na ordem
    const fim = agora + ms;
    for (;;) {
      const prox = fila.filter((t) => t.em <= fim).sort((a, b) => a.em - b.em)[0];
      if (!prox) break;
      fila.splice(fila.indexOf(prox), 1);
      agora = prox.em; prox.f();
    }
    agora = fim;
  };
  return { env, els, passar };
}
const tudo = (a) => ['listas', 'tags', 'combos'].forEach((p) => a.chegou(p));

let falhas = 0;
const teste = (nome, fn) => { try { fn(); console.log('ok   ' + nome); } catch (e) { falhas++; console.error('FALHOU: ' + nome + '\n  ' + e.message); } };

teste('rede boa: tudo chega em 0,3 s e a abertura so sai aos 0,6 s (nao pisca)', () => {
  const { env, els, passar } = ambiente();
  const a = criarAbertura(env);
  assert.strictEqual(env.coberto, true, 'o que esta atras fica coberto');
  passar(300); tudo(a);
  assert.strictEqual(a.estado(), 'abrindo');
  passar(299);
  assert.strictEqual(a.estado(), 'abrindo');
  passar(1);
  assert.strictEqual(a.estado(), 'fora');
  assert.ok(els.abertura.classes.has('saindo'));
  assert.strictEqual(env.coberto, false, 'o Lancar volta a receber toque assim que a abertura comeca a sair');
  assert.strictEqual(els.abertura.hidden, false, 'ainda esta apagando');
  passar(200);
  assert.strictEqual(els.abertura.hidden, true);
});

teste('rede lenta: sai na hora em que a ultima parte chega; faltando uma, fica', () => {
  const { env, passar } = ambiente();
  const a = criarAbertura(env);
  passar(1000); a.chegou('listas'); a.chegou('tags');
  passar(2000);
  assert.strictEqual(a.estado(), 'abrindo', 'sem os combos o Lancar nao esta pronto');
  a.chegou('combos');
  assert.strictEqual(a.estado(), 'fora');
});

teste('a mesma parte chegando tres vezes nao vale por tres', () => {
  const { env, passar } = ambiente();
  const a = criarAbertura(env);
  passar(700); a.chegou('listas'); a.chegou('listas'); a.chegou('listas'); a.chegou('outra');
  assert.strictEqual(a.estado(), 'abrindo');
});

teste('aos 4 s avisa que esta demorando, e continua esperando', () => {
  const { env, els, passar } = ambiente();
  const a = criarAbertura(env);
  passar(3999);
  assert.strictEqual(els.abMsg.textContent, '');
  passar(1);
  assert.strictEqual(els.abMsg.textContent, 'Está demorando mais que o normal. Continuo tentando…');
  passar(5000); tudo(a);
  assert.strictEqual(a.estado(), 'fora');
});

teste('falha de rede: mostra o erro com Tentar de novo, sem Sair, e nao sai nem que os dados cheguem depois', () => {
  const { env, els, passar } = ambiente();
  const a = criarAbertura(env);
  passar(2500);
  assert.strictEqual(a.falhou('rede'), true);
  assert.ok(els.abertura.classes.has('erro'));
  assert.strictEqual(els.abErroT.textContent, 'Não consegui carregar seus dados.');
  assert.strictEqual(els.abErroS.textContent, 'Confira a conexão com a internet.');
  assert.strictEqual(els.abSair.hidden, true);
  tudo(a); passar(5000);
  assert.strictEqual(a.estado(), 'erro');
  assert.strictEqual(els.abMsg.textContent, '', 'o aviso de demora nao aparece por cima do erro');
  assert.strictEqual(env.coberto, true);
  els.abDeNovo.onclick();
  assert.strictEqual(env.recarregou, true);
});

teste('conta sem espaco: a mensagem propria e o botao Sair da conta', () => {
  const { env, els } = ambiente();
  const a = criarAbertura(env);
  a.falhou('conta');
  assert.strictEqual(els.abErroT.textContent, 'Sua conta ainda não tem um espaço.');
  assert.strictEqual(els.abSair.hidden, false);
  els.abSair.onclick();
  assert.strictEqual(env.saiu, true);
});

teste('depois que a abertura saiu, uma falha de releitura nao a traz de volta (quem chama avisa de outro jeito)', () => {
  const { env, els, passar } = ambiente();
  const a = criarAbertura(env);
  passar(700); tudo(a);
  assert.strictEqual(a.falhou('rede'), false);
  assert.strictEqual(els.abertura.classes.has('erro'), false);
});

teste('pagina sem o elemento da abertura: nada quebra', () => {
  const a = criarAbertura({ el: () => null, agora: () => 0, setTimeout() {}, fundo() { throw new Error('nao devia cobrir nada'); }, recarregar() {}, sair() {} });
  tudo(a);
  assert.strictEqual(a.estado(), 'fora');
  assert.strictEqual(a.falhou('rede'), false);
});

teste('o index.html traz a abertura no comeco do body, antes do cabecalho, com o script logo depois', () => {
  const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
  const iAb = html.indexOf('id="abertura"'), iScript = html.indexOf('<script src="js/abertura.js"></script>'), iHeader = html.indexOf('<header>');
  assert.ok(iAb > html.indexOf('<body>') && iAb < iScript && iScript < iHeader, 'ordem: body, abertura, script, header');
  ['abMsg', 'abErroT', 'abErroS', 'abDeNovo', 'abSair'].forEach((id) => assert.ok(html.includes('id="' + id + '"'), 'falta #' + id));
});

teste('o icone da abertura e o do splash do Android: mesmo recorte, em circulo, no centro; em dev leva a faixa DEV (demo v2)', () => {
  const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
  const css = fs.readFileSync(path.join(__dirname, 'css', 'app.css'), 'utf8');
  assert.ok(html.includes('<svg class="ab-ic" id="abIc" viewBox="51.2 51.2 409.6 409.6"'), 'o recorte e a zona segura do icone (80% do quadro de 512)');
  assert.ok(/\.ab-ic\{[^}]*224px[^}]*border-radius:50%/.test(css), 'circulo de 224 px');
  assert.ok(/\.ab\{[^}]*grid-template-rows:1fr auto 1fr/.test(css), 'o icone fica na linha do meio: centro exato da tela');
  assert.ok(html.indexOf('class="ab-baixo"') > html.indexOf('id="abIc"') && html.indexOf('class="ab-nome"') > html.indexOf('class="ab-baixo"'), 'nome e indicador ficam na metade de baixo, fora do centro');
  const amb = fs.readFileSync(path.join(__dirname, 'js', 'ambiente.js'), 'utf8');
  assert.ok(amb.includes("getElementById('abIc')") && amb.includes('>DEV</text>'), 'em dev o icone da abertura leva a faixa DEV');
});

teste('o init avisa a abertura das tres leituras do Lancar e das falhas', () => {
  const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
  const i = app.indexOf('async function init(){'), init = app.slice(i, app.indexOf('\n}', i));
  ['listas', 'tags', 'combos'].forEach((p) => {
    assert.ok(init.includes("Abertura.chegou('" + p + "')"), 'falta chegou: ' + p);
    assert.ok(init.includes("falhaNaAbertura('" + p + "')"), 'a falha de ' + p + ' nao chega a abertura');
  });
  assert.ok(init.includes("Abertura.falhou("), 'sem banco (conta sem espaco ou rede) a abertura precisa mostrar o erro');
  const api = fs.readFileSync(path.join(__dirname, 'api.js'), 'utf8');
  assert.strictEqual(/\balert\(/.test(api), false, 'o api.js nao abre mais alerta do navegador: o erro aparece na abertura');
});

if (falhas) process.exit(1);
console.log('abertura.test.js: ok');
