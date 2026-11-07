// Teste de "esconder valores" (olho), demo-valores-v1-esconder: so o front, a escolha fica no aparelho. Rode com: node web/valores.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');

const codigo = fs.readFileSync(path.join(__dirname, 'js', 'valores.js'), 'utf8');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');

// 1. a mascara (funcao pura): valor em reais vira pontos, o sinal fica; rotulo de grafico so se for numero
const w = {};
new Function('window', codigo)(w);
const m = w.ValoresEscondidos.mascararTexto;
assert.strictEqual(m('R$\u00a0105.029,54'), 'R$ ••••••');
assert.strictEqual(m('-R$ 45,00'), '-R$ ••••••');
assert.strictEqual(m('Soma +R$ 1.200 e R$ 3,50'), 'Soma +R$ •••••• e R$ ••••••');
assert.strictEqual(m('30% do movimento'), '30% do movimento', 'percentual nao e valor');
assert.strictEqual(m('jan', true), 'jan');
assert.strictEqual(m('12 mil', true), '••');
assert.strictEqual(m('até 500', true), '••');
assert.strictEqual(m('1.234', true), '••');
assert.strictEqual(m('12 mil', false), '12 mil', 'fora de grafico so o R$ conta');

// 2. nunca esconde o que se digita nem onde se decide olhando o numero
assert.ok(/EXCECOES = '\.valor-grande,#barraResumo,\.modal-box,\.fm-conf'/.test(codigo), 'excecoes: valor digitado, resumo da barra, janelas e conferencia');

// 3. a escolha fica so no aparelho (nunca vai ao servidor)
assert.ok(/localStorage\.setItem\(CHAVE/.test(codigo) && !/fetch\(|XMLHttpRequest|api\(/.test(codigo), 'localStorage, sem chamada ao servidor');

// 4. o olho esta no celular (barra) e no computador (topo) e o script e carregado
assert.strictEqual((html.match(/class="mn-ib(?: olho-topo)? olho"/g) || []).length, 2, 'dois botoes de olho');
assert.ok(/<script src="js\/valores\.js"><\/script>/.test(html), 'valores.js carregado');
console.log('valores.test.js ok');
