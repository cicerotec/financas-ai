// Teste do botao voltar (web/js/voltar.js) em Node, sem navegador: um historico de mentira que dispara o popstate como o navegador.
// Rode com: node web/voltar.test.js
const assert = require('assert');
const criarVoltar = require('./js/voltar.js');

// historico falso: entradas {state}, indice, pushState corta o que estava adiante e go() anda de forma assincrona
function novoCenario(abaInicial){
  const h = { entradas: [{state: null, url: '/#' + abaInicial}], i: 0, ouvinte: null, filas: [] };
  const history = {
    get state(){ return h.entradas[h.i].state; },
    pushState(s, t, u){ h.entradas.length = h.i + 1; h.entradas.push({state: s, url: u}); h.i++; },
    replaceState(s, t, u){ h.entradas[h.i] = {state: s, url: u}; },
    go(d){ h.filas.push(d); setTimeout(()=>{ const d2 = h.filas.shift(); const n = h.i + d2; if(n < 0 || n >= h.entradas.length) return; h.i = n; h.ouvinte({state: h.entradas[h.i].state}); }, 0); }
  };
  const location = { pathname: '/', search: '', get hash(){ return h.entradas[h.i].url.replace(/^.*(#.*)$/, '$1'); } };
  const app = { aba: abaInicial, camadas: [], editandoSujo: false, perguntou: 0, impedirFechar: false };
  const V = criarVoltar({ history, location, ouvir: fn => { h.ouvinte = fn; }, microtask: fn => Promise.resolve().then(fn), setTimeout: (fn, ms) => setTimeout(fn, ms) });
  const hooks = {
    abaAtual: () => app.aba,
    abaValida: a => ['lancar', 'historico', 'saldos'].includes(a),
    irParaAba: a => { app.aba = a; V.sync(); },
    contarAbertas: () => app.camadas.length,
    fecharTopo: () => { if(!app.impedirFechar) app.camadas.pop(); },
    precisaConfirmar: () => app.editandoSujo,
    perguntarSaida: () => { app.perguntou++; }
  };
  V.iniciar(hooks);
  const abrir = nome => { app.camadas.push(nome); V.sync(); };
  const fechar = () => { app.camadas.pop(); V.sync(); };
  const voltarDoNavegador = () => history.go(-1);
  const espera = () => new Promise(r => setTimeout(r, 20));
  return { h, app, V, abrir, fechar, voltarDoNavegador, espera, hooks };
}

(async function(){
  // 1. abrir uma janela cria entrada; o voltar do navegador fecha a janela e fica na aba
  let c = novoCenario('historico');
  await c.espera();
  c.abrir('acoes'); await c.espera();
  assert.strictEqual(c.h.entradas.length, 2);
  c.voltarDoNavegador(); await c.espera();
  assert.deepStrictEqual(c.app.camadas, []);
  assert.strictEqual(c.app.aba, 'historico');
  assert.strictEqual(c.h.i, 0);

  // 2. fechar a janela pelo proprio botao tira a entrada, sem fechar mais nada
  c = novoCenario('historico'); await c.espera();
  c.abrir('filtros'); await c.espera();
  c.fechar(); await c.espera();
  assert.strictEqual(c.h.i, 0);
  assert.strictEqual(c.h.entradas.length, 2, 'a entrada de adiante continua no navegador');
  assert.strictEqual(c.app.aba, 'historico');

  // 3. cada troca de aba cria uma entrada e o voltar volta para a aba anterior
  c = novoCenario('historico'); await c.espera();
  c.hooks.irParaAba('saldos'); await c.espera();
  c.hooks.irParaAba('lancar'); await c.espera();
  assert.strictEqual(c.h.entradas.length, 3);
  c.voltarDoNavegador(); await c.espera();
  assert.strictEqual(c.app.aba, 'saldos');
  c.voltarDoNavegador(); await c.espera();
  assert.strictEqual(c.app.aba, 'historico');

  // 4. Editar: a janela de acoes fecha e a aba muda no mesmo instante: uma entrada so (a janela cede o lugar)
  c = novoCenario('historico'); await c.espera();
  c.abrir('acoes'); await c.espera();
  c.app.camadas.pop(); c.hooks.irParaAba('lancar'); await c.espera();
  assert.strictEqual(c.h.entradas.length, 2);
  assert.deepStrictEqual(c.V.estado(), ['historico:0', 'lancar:0']);
  c.voltarDoNavegador(); await c.espera();
  assert.strictEqual(c.app.aba, 'historico');

  // 5. salvar ou cancelar a edicao (retorno) desfaz a ida: nao sobra a aba Lancar no historico
  c = novoCenario('historico'); await c.espera();
  c.abrir('acoes'); await c.espera();
  c.app.camadas.pop(); c.hooks.irParaAba('lancar'); await c.espera();
  c.V.retornar(); c.hooks.irParaAba('historico'); await c.espera();
  assert.deepStrictEqual(c.V.estado(), ['historico:0']);
  assert.strictEqual(c.h.i, 0);

  // 6. retorno depois de fechar uma pergunta (modal) na mesma aba: dois ajustes seguidos, um de cada vez
  c = novoCenario('historico'); await c.espera();
  c.abrir('acoes'); await c.espera();
  c.app.camadas.pop(); c.hooks.irParaAba('lancar'); await c.espera();
  c.abrir('modal'); await c.espera();
  c.app.camadas.pop(); c.V.sync();
  c.V.retornar(); c.hooks.irParaAba('historico'); await c.espera(); await c.espera();
  assert.deepStrictEqual(c.V.estado(), ['historico:0']);
  assert.strictEqual(c.app.aba, 'historico');
  assert.deepStrictEqual(c.app.camadas, []);

  // 7. texto digitado: o voltar nao sai da aba, a entrada volta e o app pergunta
  c = novoCenario('historico'); await c.espera();
  c.abrir('acoes'); await c.espera();
  c.app.camadas.pop(); c.hooks.irParaAba('lancar'); await c.espera();
  c.app.editandoSujo = true;
  c.voltarDoNavegador(); await c.espera();
  assert.strictEqual(c.app.aba, 'lancar');
  assert.strictEqual(c.app.perguntou, 1);
  assert.strictEqual(c.h.entradas.length, 2);
  assert.strictEqual(c.h.i, 1);
  // sem texto digitado, sai direto
  c.app.editandoSujo = false;
  c.voltarDoNavegador(); await c.espera();
  assert.strictEqual(c.app.aba, 'historico');

  // 8. o voltar dentro da mesma aba nao pergunta, mesmo com texto digitado (fecha so a janela de tags)
  c = novoCenario('lancar'); await c.espera();
  c.abrir('tags'); await c.espera();
  c.app.editandoSujo = true;
  c.voltarDoNavegador(); await c.espera();
  assert.deepStrictEqual(c.app.camadas, []);
  assert.strictEqual(c.app.perguntou, 0);

  // 9. camada que nao fecha (ex.: criando eventos): a entrada volta e a tela continua aberta
  c = novoCenario('historico'); await c.espera();
  c.abrir('prever'); await c.espera();
  c.app.impedirFechar = true;
  c.voltarDoNavegador(); await c.espera();
  assert.deepStrictEqual(c.app.camadas, ['prever']);
  assert.strictEqual(c.h.i, 1);
  assert.strictEqual(c.h.entradas.length, 2);
  c.app.impedirFechar = false;
  c.voltarDoNavegador(); await c.espera();
  assert.deepStrictEqual(c.app.camadas, []);

  // 10. varias camadas: um voltar fecha uma de cada vez
  c = novoCenario('historico'); await c.espera();
  c.abrir('prever'); c.abrir('janela'); await c.espera();
  assert.strictEqual(c.h.entradas.length, 3);
  c.voltarDoNavegador(); await c.espera();
  assert.deepStrictEqual(c.app.camadas, ['prever']);
  c.voltarDoNavegador(); await c.espera();
  assert.deepStrictEqual(c.app.camadas, []);

  // 11. fechar duas camadas de uma vez (confirmar uma pergunta aberta sobre a janela de acoes) volta as duas entradas
  c = novoCenario('historico'); await c.espera();
  c.abrir('acoes'); c.abrir('modal'); await c.espera();
  c.app.camadas.length = 0; c.V.sync(); await c.espera();
  assert.strictEqual(c.h.i, 0);
  assert.deepStrictEqual(c.V.estado(), ['historico:0']);

  // 12. F5: o endereco leva a aba (#saldos) e o app reabre nela
  c = novoCenario('lancar');
  c.h.entradas[0].url = '/#saldos';
  const c2 = (function(){ // novo carregamento com o mesmo historico
    const app = { aba: 'lancar', camadas: [] };
    const V = criarVoltar({
      history: { get state(){ return c.h.entradas[c.h.i].state; }, pushState(){}, replaceState(s, t, u){ c.h.entradas[c.h.i] = {state: s, url: u}; }, go(){} },
      location: { pathname: '/', search: '', hash: '#saldos' },
      ouvir(){}, microtask: fn => Promise.resolve().then(fn), setTimeout
    });
    V.iniciar({ abaAtual: () => app.aba, abaValida: a => a === 'saldos', irParaAba: a => { app.aba = a; }, contarAbertas: () => 0, fecharTopo(){}, precisaConfirmar: () => false, perguntarSaida(){} });
    return app;
  })();
  assert.strictEqual(c2.aba, 'saldos');

  console.log('voltar.test.js: tudo certo');
  process.exit(0);
})().catch(e => { console.error(e); process.exit(1); });
