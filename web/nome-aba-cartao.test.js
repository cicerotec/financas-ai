// A aba de faturas do cartao se chama "Cartão de crédito" na tela (decisao de 2026-11-09, .ai/decisoes.md).
// So o texto mudou: o id da aba continua "cartao" (data-tab, #tab-cartao, MENU_APP e o endereco #cartao).
const assert = require('assert');
const fs = require('fs');
const path = require('path');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');

// 1. o botao da aba (fonte do nome no menu e no titulo da tela) traz o nome novo e o id de sempre
assert.ok(/<button data-tab="cartao">Cartão de crédito<\/button>/.test(html), 'a aba se chama Cartão de crédito e o id continua cartao');
assert.ok(/<section id="tab-cartao"/.test(html), 'a secao da aba continua #tab-cartao');
assert.ok(/\['cartao','💳'\]/.test(app), 'o menu continua apontando para o id cartao');

// 2. a secao de Configuracoes acompanha o nome
assert.ok(/<b>Cartão de crédito<\/b><span>Fechamentos da planilha<\/span>/.test(html), 'a secao de Configuracoes se chama Cartão de crédito');

// 3. nenhuma frase do app cita a aba pelo nome antigo
for (const [nome, txt] of [['index.html', html], ['app.js', app]]) {
  assert.ok(!/aba Cartão(?! de crédito)/.test(txt), nome + ' ainda cita "aba Cartão" sem "de crédito"');
}

console.log('nome-aba-cartao: ok');
