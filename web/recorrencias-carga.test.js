// Teste da leitura das recorrencias ativas (GET /recorrencias) em web/js/app.js: ela e do Historico e de Futuros (a acao
// "Parar recorrencia"), nunca da subida do app nem do Lancar. Bug do item 61 do BACKLOG: o init lia /recorrencias na subida
// e cada gravacao relia em qualquer tela, contra a regra "cada tela le so o que e dela" (.ai/guardrails.md).
// O carga.test.js nao via porque so roda o api.js; este roda o trecho do app.js que guarda as recorrencias, com um db de mentira.
// Rode com: node web/recorrencias-carga.test.js
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const assert = require('assert');

const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const corpo = (inicio) => { const i = app.indexOf(inicio); assert.ok(i >= 0, inicio + ' nao existe'); return app.slice(i, app.indexOf('\n}', i)); };
// o trecho do app.js que guarda e le as recorrencias: do estado (recAtivas) ate a primeira funcao que so consulta
const ini = app.indexOf('let recAtivas'), fim = app.indexOf('function recorrenciaDe(');
assert.ok(ini >= 0 && fim > ini, 'o trecho das recorrencias mudou de lugar: ajuste este teste');

let leituras = 0, falhar = false, aba = 'lancar', desenhos = 0;
const ctx = {
  console: { warn() {}, error() {} },
  JSON, Promise,
  setTimeout: (f) => { f(); return 1; }, clearTimeout() {}, // o agendamento roda na hora
  db: { recorrencias: async () => { leituras++; if (falhar) throw new Error('rede'); return [{ id: 'r1', ultimoId: 'l1', intervalo: 1 }]; } },
  abaAtual: () => aba,
  renderHistorico: () => { desenhos++; }, renderFuturos: () => {}
};
vm.createContext(ctx);
vm.runInContext(app.slice(ini, fim), ctx);
const rodar = (codigo) => vm.runInContext(codigo, ctx);
const esperar = () => new Promise((r) => setImmediate(r));
const lidas = () => { const n = leituras; leituras = 0; return n; };

let falhas = 0;
const teste = async (nome, fn) => {
  try { await fn(); console.log('ok   ' + nome); } catch (e) { falhas++; console.error('FALHOU: ' + nome + '\n  ' + e.message); }
};

(async () => {
  await teste('a subida (init) nao le as recorrencias', async () => {
    const init = corpo('async function init(){');
    assert.strictEqual(/carregarRecorrencias\(|garantirRecorrencias\(/.test(init), false, 'o init chama a leitura das recorrencias');
    assert.strictEqual(init.includes('agendarRecorrencias()'), false, 'o init agenda a leitura a cada mudanca, em qualquer tela');
  });

  await teste('Historico e Futuros pedem as recorrencias ao abrir (irParaAba); as outras telas nao', async () => {
    assert.ok(corpo('function irParaAba(t){').includes('if(usaRecorrencias(t)) garantirRecorrencias();'), 'irParaAba nao pede as recorrencias');
    assert.deepStrictEqual(['lancar', 'historico', 'futuros', 'cartao', 'saldos', 'tendencias', 'listas', 'versoes', 'configuracoes'].filter((t) => rodar(`usaRecorrencias('${t}')`)), ['historico', 'futuros']);
  });

  await teste('abrir a tela le uma vez; abrir de novo, sem mudanca, nao le', async () => {
    aba = 'historico';
    await rodar('garantirRecorrencias()');
    assert.strictEqual(lidas(), 1);
    assert.strictEqual(rodar('recAtivas.length'), 1);
    assert.strictEqual(desenhos, 1, 'a tela e redesenhada com o que chegou');
    await rodar('garantirRecorrencias()');
    assert.strictEqual(lidas(), 0);
  });

  await teste('lancamento que muda com o Lancar aberto nao le nada; a leitura fica para quando a tela abrir', async () => {
    aba = 'lancar';
    rodar('recorrenciasMudaram()');
    await esperar();
    assert.strictEqual(lidas(), 0);
    aba = 'futuros';
    await rodar('garantirRecorrencias()');
    assert.strictEqual(lidas(), 1);
  });

  await teste('lancamento que muda com o Historico ou Futuros aberto relê na hora', async () => {
    aba = 'futuros';
    rodar('recorrenciasMudaram()');
    await esperar();
    assert.strictEqual(lidas(), 1);
  });

  await teste('leitura que falha e tentada de novo na proxima vez que a tela abrir', async () => {
    aba = 'lancar';
    rodar('recorrenciasMudaram()');
    falhar = true;
    await rodar('garantirRecorrencias()');
    assert.strictEqual(lidas(), 1);
    falhar = false;
    await rodar('garantirRecorrencias()');
    assert.strictEqual(lidas(), 1, 'a falha nao pode deixar a tela sem "Parar recorrencia" ate o proximo lancamento');
  });

  await teste('mudanca que chega durante a leitura nao se perde', async () => {
    aba = 'lancar';
    rodar('recorrenciasMudaram()');
    const p = rodar('garantirRecorrencias()'); // leitura em curso
    rodar('recorrenciasMudaram()');            // alguem gravou no meio, com o Lancar aberto
    await p;
    lidas();
    await rodar('garantirRecorrencias()');
    assert.strictEqual(lidas(), 1);
  });

  if (falhas) process.exit(1);
  console.log('recorrencias-carga.test.js: ok');
})();
