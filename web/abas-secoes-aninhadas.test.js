// Trocar de aba nao pode esconder as <section> de dentro de uma aba (ex.: as da janela de Tags do Lancar).
// Bug: irParaAba fazia querySelectorAll('main section') + classList.add('hidden'); isso pegava tgTagsSec, tgTodasSec e tgCombosSec,
// e so a aba voltava a ser mostrada, entao depois da primeira troca a lista de tags e de combos ficava em display:none para sempre.
// Rode com: node web/abas-secoes-aninhadas.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');

// o seletor que irParaAba usa para esconder todas as abas
const ini = app.indexOf('function irParaAba(');
assert.ok(ini > 0, 'irParaAba existe');
const corpo = app.slice(ini, app.indexOf('\n}', ini));
const m = corpo.match(/document\.querySelectorAll\('([^']+)'\)\.forEach\(sec=>sec\.classList\.add\('hidden'\)\)/);
assert.ok(m, 'irParaAba esconde as abas com querySelectorAll(...).forEach(...add(hidden))');
const seletor = m[1];

// as <section> de dentro de <main> e quantas estao aninhadas em outra <section>
const main = html.slice(html.indexOf('<main>'), html.indexOf('</main>'));
let pilha = 0; const abas = [], aninhadas = [];
for (const t of main.matchAll(/<(\/?)section\b([^>]*)>/g)) {
  if (t[1]) { pilha--; continue; }
  const id = (t[2].match(/\bid="([^"]+)"/) || [])[1];
  (pilha === 0 ? abas : aninhadas).push(id);
  pilha++;
}
assert.ok(abas.length >= 9 && abas.every(id => /^tab-/.test(id)), 'toda <section> de primeiro nivel e uma aba (tab-*): ' + abas);
assert.ok(aninhadas.includes('tgTagsSec') && aninhadas.includes('tgCombosSec'), 'a janela de Tags tem <section> aninhadas (o teste so vale enquanto elas existirem)');

// "main section" pega tambem as aninhadas; so as abas (filhas diretas de main) devem ser escondidas
assert.strictEqual(seletor, 'main > section', 'irParaAba esconde so as abas (main > section), nao as <section> de dentro delas; usa: ' + seletor);
console.log('abas-secoes-aninhadas.test.js ok');
