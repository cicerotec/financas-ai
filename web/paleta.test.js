// Item 56: a paleta do seletor de cor da forma de pagamento (COR_PALETA em js/app.js) precisa ter 16 cores sem repetir e conter toda cor padrao
// de status do backend (CorPadrao em backend/domain/catalogo.go); senao a bolinha do seletor nao aparece marcada para aquele status.
// Rode com: node web/paleta.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');

const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const catalogo = fs.readFileSync(path.join(__dirname, '..', 'backend', 'domain', 'catalogo.go'), 'utf8');

const m = app.match(/const COR_PALETA = \[([^\]]*)\]/);
assert(m, 'COR_PALETA nao encontrada em js/app.js');
const paleta = m[1].match(/#[0-9a-fA-F]{6}/g).map((c) => c.toUpperCase());
assert.strictEqual(paleta.length, 16, 'a paleta tem 16 cores');
assert.strictEqual(new Set(paleta).size, 16, 'sem cor repetida');

// so as entradas selecionaveis precisam estar na paleta (as antigas, como CREDITO IN, so aparecem como texto)
const padroes = [...catalogo.matchAll(/\{Nome: "([^"]+)"[^}]*CorPadrao: "(#[0-9a-fA-F]{6})"[^}]*Selecionavel: true/g)].map((x) => [x[1], x[2].toUpperCase()]);
assert(padroes.length >= 2, 'achou as cores do catalogo');
for (const [nome, cor] of padroes) assert(paleta.includes(cor), `a cor padrao de ${nome} (${cor}) nao esta na paleta`);
console.log(`paleta ok: ${paleta.length} cores, ${padroes.length} status conferidos`);
