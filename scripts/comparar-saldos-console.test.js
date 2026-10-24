// Teste da comparacao de saldos (scripts/comparar-saldos-console.js) em Node, sem navegador.
// Rode com: node scripts/comparar-saldos-console.test.js
const assert = require('assert');
const { compararSaldos } = require('./comparar-saldos-console.js');

const antigo = {
  Itau: { saldo: 1250.5, mapa: { '2026-08': 1000, '2026-09': 1100.25, '2026-10': 1250.5 } },
  Nubank: { saldo: -40, mapa: { '2026-09': -10, '2026-10': -40 } },
};
const resposta = (extra) => Object.assign({
  total: { valor: 1210.5 },
  meios: [
    { meio: 'Itau', saldo: 1250.5, meses: [{ mes: '2026-10', valor: 1250.5 }, { mes: '2026-09', valor: 1100.25 }, { mes: '2026-07', valor: 900 }] },
    { meio: 'Nubank', saldo: -40, meses: [{ mes: '2026-10', valor: -40 }, { mes: '2026-09', valor: -10 }] },
  ],
}, extra || {});

// igual: o servidor ter um mes a mais (2026-07) nao e diferenca
let r = compararSaldos(antigo, 1210.5, resposta());
assert.strictEqual(r.igual, true, 'saldos e meses iguais');
assert.strictEqual(r.linhas.length, 2);

// diferenca de meio centavo ainda e igual (arredondamento); de um centavo, nao
assert.strictEqual(compararSaldos(antigo, 1210.504, resposta()).igual, true);
r = compararSaldos(antigo, 1210.5, resposta({ total: { valor: 1210.51 } }));
assert.strictEqual(r.igual, false);
assert.strictEqual(r.total.ok, false);
assert.strictEqual(r.total.diferenca, 0.01);

// saldo diferente em um meio
r = compararSaldos(antigo, 1210.5, resposta({ meios: [
  { meio: 'Itau', saldo: 1251, meses: [{ mes: '2026-10', valor: 1251 }] },
  { meio: 'Nubank', saldo: -40, meses: [{ mes: '2026-10', valor: -40 }] },
] }));
assert.strictEqual(r.igual, false);
assert.strictEqual(r.linhas[0].ok, false);
assert.strictEqual(r.linhas[0].diferenca, 0.5);
assert.deepStrictEqual(r.linhas[0].mesesDiferentes, [{ mes: '2026-10', antigo: 1250.5, novo: 1251 }]);

// so o mes do meio, com o saldo final igual
r = compararSaldos(antigo, 1210.5, resposta({ meios: [
  { meio: 'Itau', saldo: 1250.5, meses: [{ mes: '2026-10', valor: 1250.5 }, { mes: '2026-09', valor: 1090 }] },
  { meio: 'Nubank', saldo: -40, meses: [] },
] }));
assert.strictEqual(r.igual, false, 'um mes diferente reprova mesmo com o saldo final igual');
assert.strictEqual(r.linhas[0].mesesDiferentes.length, 1);

// meio so de um lado
r = compararSaldos(antigo, 1210.5, resposta({ meios: [{ meio: 'Itau', saldo: 1250.5, meses: [] }, { meio: 'Outro', saldo: 1, meses: [] }] }));
assert.deepStrictEqual(r.soAntigo, ['Nubank']);
assert.deepStrictEqual(r.soNovo, ['Outro']);
assert.strictEqual(r.igual, false);

console.log('ok   comparar-saldos: igual, tolerancia de meio centavo, saldo, mes e meio so de um lado');
