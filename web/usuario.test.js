// Teste do bloco da conta (web/js/usuario.js) em Node, sem navegador: um DOM minimo de mentira e uma sessao falsa.
// Rode com: node web/usuario.test.js
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const assert = require('assert');

const escuta = { doc: {} };
const els = {};
function el(id) {
  return (els[id] = els[id] || {
    id, hidden: false, innerHTML: '', textContent: '', dataset: {}, attrs: {},
    setAttribute(k, v) { this.attrs[k] = v; }, focus() { this.focado = true; },
    addEventListener(tipo, fn) { (this.ouvintes = this.ouvintes || {})[tipo] = fn; }
  });
}
['usrBtn', 'usrDrop', 'usrAv', 'usrNome'].forEach(el);
els.usrDrop.hidden = true; // como no HTML

let sessao = null;
const chamadas = [];
const ctx = {
  console, String, Array, JSON, Object,
  esc: (t) => String(t == null ? '' : t).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;'),
  fecharMenuApp: (v) => chamadas.push(['fecharMenuApp', v]),
  irParaAba: (t) => chamadas.push(['irParaAba', t]),
  document: {
    getElementById: (id) => els[id] || null,
    addEventListener(tipo, fn) { escuta.doc[tipo] = fn; }
  },
  window: {},
  claude: {
    sessao: () => sessao,
    sair: () => chamadas.push(['sair']),
    trocarEspaco: (id) => chamadas.push(['trocarEspaco', id])
  }
};
ctx.window.claude = ctx.claude;
vm.createContext(ctx);
vm.runInContext(fs.readFileSync(path.join(__dirname, 'js', 'usuario.js'), 'utf8') +
  '\nthis.__u = {htmlConta, htmlContaTopo, htmlContaFim, usAtualizar, usAbrir, usFechar, usPreparar};', ctx);
const U = ctx.__u;

// um alvo de clique/mudanca com closest() simples por atributo data-conta
const alvo = (conta, extra) => Object.assign({ closest: (sel) => (sel.includes('data-conta') && conta ? { dataset: { conta }, value: (extra || {}).value } : (sel === '#usrDrop' && (extra || {}).dentro ? {} : null)) }, extra || {});
const clique = (conta, extra) => escuta.doc.click({ target: alvo(conta, extra), });
const limpa = () => { chamadas.length = 0; };

const unico = { nome: 'Cícero', email: 'c@x.com', apelido: 'Cícero', espaco: { id: 's1', nome: 'Família Silva', role: 'owner' }, espacos: [{ id: 's1', nome: 'Família Silva' }] };

// ---------- o bloco ----------
sessao = unico;
let h = U.htmlConta(sessao);
assert(h.includes('>C<'), 'inicial');
assert(h.includes('<b>Cícero</b>') && h.includes('Família Silva · dono'), 'nome, espaco e papel');
assert(!h.includes('cta-dica') && !h.includes('<select'), 'com apelido e um espaco: sem dica nem seletor');

sessao = Object.assign({}, unico, { espaco: { id: 's1', nome: 'Família Silva', role: 'member' } });
assert(U.htmlConta(sessao).includes('Família Silva · membro'), 'papel membro');

sessao = Object.assign({}, unico, { apelido: '', nome: 'cicero' });
h = U.htmlConta(sessao);
assert(h.includes('data-conta="listas"') && h.includes('Defina seu nome em <b>Listas</b>'), 'dica do apelido');
assert(h.includes('>C<'), 'a inicial vem em maiuscula');

sessao = Object.assign({}, unico, { espacos: [{ id: 's1', nome: 'Família Silva' }, { id: 's2', nome: 'Casa da praia' }] });
h = U.htmlConta(sessao);
assert(h.includes('<select data-conta="espaco"') && h.includes('value="s1" selected') && h.includes('Casa da praia'), 'seletor de espaco, com o atual marcado');
assert(!h.includes('Família Silva · dono'), 'com seletor, a linha do espaco some');

sessao = Object.assign({}, unico, { nome: '<img src=x onerror=alert(1)>' });
h = U.htmlConta(sessao);
assert(!h.includes('<img') && h.includes('&lt;img'), 'nome escapado');

// ---------- topo e fim do menu do celular ----------
sessao = unico;
assert(U.htmlContaTopo().includes('Cícero') && U.htmlContaTopo().endsWith('<hr class="mn-sep">'), 'topo termina com separador');
const fim = U.htmlContaFim();
assert(fim.startsWith('<hr class="mn-sep">') && fim.includes('data-conta="sair"') && fim.includes('mn-sair') && fim.includes('Sair'), 'Sair no fim, separado, em vermelho');
sessao = null;
assert.strictEqual(U.htmlContaTopo(), '', 'sem sessao: sem topo');
assert.strictEqual(U.htmlContaFim(), '', 'sem sessao: sem Sair');

// ---------- botao do computador ----------
sessao = unico;
U.usAtualizar();
assert.strictEqual(els.usrBtn.hidden, false);
assert.strictEqual(els.usrAv.textContent, 'C');
assert.strictEqual(els.usrNome.textContent, 'Cícero');
sessao = null; U.usAtualizar();
assert.strictEqual(els.usrBtn.hidden, true, 'sem sessao o botao some');
sessao = unico; U.usAtualizar();

U.usPreparar();
els.usrBtn.ouvintes.click({ stopPropagation() {} });
assert.strictEqual(els.usrDrop.hidden, false);
assert.strictEqual(els.usrBtn.attrs['aria-expanded'], 'true');
assert(els.usrDrop.innerHTML.includes('Cícero') && els.usrDrop.innerHTML.includes('data-conta="sair"'), 'o menu do computador tem nome e Sair');
assert(!els.usrDrop.innerHTML.includes('data-tab'), 'o menu do computador nao repete as telas (ja estao nas abas)');
els.usrBtn.ouvintes.click({ stopPropagation() {} });
assert.strictEqual(els.usrDrop.hidden, true, 'segundo toque fecha');

// Esc fecha e devolve o foco
els.usrBtn.ouvintes.click({ stopPropagation() {} });
els.usrBtn.focado = false;
escuta.doc.keydown({ key: 'Escape' });
assert.strictEqual(els.usrDrop.hidden, true); assert(els.usrBtn.focado, 'foco volta ao botao');

// clique fora fecha; clique dentro nao
els.usrBtn.ouvintes.click({ stopPropagation() {} });
clique(null, { dentro: true });
assert.strictEqual(els.usrDrop.hidden, false, 'clique dentro do menu nao fecha');
clique(null, {});
assert.strictEqual(els.usrDrop.hidden, true, 'clique fora fecha');

// ---------- as acoes ----------
limpa(); els.usrBtn.ouvintes.click({ stopPropagation() {} });
clique('sair');
assert.deepStrictEqual(JSON.parse(JSON.stringify(chamadas)), [['sair']], 'Sair so encerra a sessao: fechar o menu do app dispara o history.go(-1) do voltar.js e cancela a navegacao do logout');
assert.strictEqual(els.usrDrop.hidden, true);

limpa(); clique('listas');
assert.deepStrictEqual(JSON.parse(JSON.stringify(chamadas)), [['fecharMenuApp', false], ['irParaAba', 'listas']], 'a dica leva a Listas');

limpa();
escuta.doc.change({ target: alvo('espaco', { value: 's2' }) });
assert.deepStrictEqual(JSON.parse(JSON.stringify(chamadas)), [['trocarEspaco', 's2']], 'trocar de espaco');

console.log('usuario.test.js: ok');
