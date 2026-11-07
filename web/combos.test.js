// Teste do item 57 (.ai/specs/combos-sugeridos): quem conta os combos sugeridos e o servidor (uma vez por semana); o navegador so junta e exibe.
// Falha se alguma tela voltar a contar combos nos lancamentos do navegador, ou se o interruptor da rotina deixar de pedir ao servidor.
// Rode com: node web/combos.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');

const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const api = fs.readFileSync(path.join(__dirname, 'api.js'), 'utf8');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');

// 1. a conta saiu do navegador
assert.ok(!/function contagemConjuntos\b/.test(app), 'contagemConjuntos saiu: o servidor conta os combos');
const ini = app.indexOf('function combosAtuaisLista(');
const corpo = app.slice(ini, app.indexOf('\n}', ini) + 2);
assert.ok(ini > 0 && !/lancamentosCache/.test(corpo), 'combosAtuaisLista nao le lancamentos');

// 2. o comportamento: salvo primeiro (com a contagem do servidor), sugerido igual ao salvo some, o resto segue a ordem do servidor
const chaveTags = a => [...a].sort().join('|');
const montar = (combosSalvos, combosSugeridos) => new Function('chaveTags', 'combosSalvos', 'combosSugeridos', corpo + '; return combosAtuaisLista();')(chaveTags, combosSalvos, combosSugeridos);
const r = montar(
  [{ tags: ['b', 'a'] }, { tags: ['x', 'y'] }],
  [{ tags: ['a', 'b'], contagem: 7 }, { tags: ['c', 'd'], contagem: 5 }, { tags: ['e', 'f'], contagem: 3 }]);
assert.deepStrictEqual(r.map(c => [c.tags.join(','), c.n, c.salvo]), [['b,a', 7, true], ['x,y', 0, true], ['c,d', 5, false], ['e,f', 3, false]], 'salvos primeiro; o salvo igual ao sugerido leva a contagem e o sugerido some');
assert.deepStrictEqual(montar([], []), [], 'sem sugeridos e sem salvos, nada');

// 3. o app guarda o que o servidor manda e o interruptor pede ao servidor
assert.ok(/combosSugeridos = \(d && Array\.isArray\(d\.sugeridos\)\)/.test(app) && /rotinaCombosAtiva = !!\(d && d\.rotinaAtiva\)/.test(app), 'o snapshot de config/combos guarda sugeridos e rotinaAtiva');
assert.ok(/'\/combos\/rotina'/.test(api) && /combosRotina,/.test(api), 'api.js chama PUT /combos/rotina');
assert.ok(/db\.combosRotina\(/.test(app), 'o interruptor chama o servidor');
assert.ok(/id="swRotinaCombos"/.test(html), 'o interruptor esta em Listas');

console.log('combos.test.js ok');
