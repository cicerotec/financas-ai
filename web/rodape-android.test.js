// Teste do rodape do app dentro do Android (web/css/app.css). No app Android (Capacitor) a pagina e desenhada ate a borda de baixo
// da tela, e os botoes do sistema (quadrado, circulo e voltar) ficam por cima do que estiver ali. Achado em 2026-11-12 no aparelho
// do dono: "Limpar" e "Salvar lançamento" do Lançar ficavam embaixo desses botoes (env(safe-area-inset-bottom) = 48 px, e o botao
// Salvar ia de 848 a 900 numa janela de 904). No navegador e na PWA a margem e 0, por isso la nao acontecia.
// Regra: todo elemento preso ao rodape da tela reserva env(safe-area-inset-bottom). Rode com: node web/rodape-android.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');

const css = fs.readFileSync(path.join(__dirname, 'css', 'app.css'), 'utf8').replace(/\/\*[\s\S]*?\*\//g, '');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
assert.ok(/viewport-fit=cover/.test(html), 'a pagina pede a tela inteira (viewport-fit=cover): por isso as margens sao por nossa conta');

// cada regra (inclusive as de dentro de @media) com o seletor e o corpo
const regras = [...css.matchAll(/([^{}@]+)\{([^{}]*)\}/g)].map((m) => ({ sel: m[1].trim(), corpo: m[2] }));
const presas = regras.filter((r) => /position:(fixed|sticky)/.test(r.corpo) && /(^|;)\s*bottom:/.test(r.corpo));
assert.ok(presas.length >= 5, 'esperava achar as barras e janelas presas ao rodape, achei ' + presas.length);

// a barra do Lançar (Limpar, Cancelar edição e Salvar): para acima dos botoes do sistema e cobre a faixa deles com a cor do fundo
const barra = regras.find((r) => r.sel === '.barra');
assert.ok(barra, 'regra .barra');
assert.ok(/position:sticky/.test(barra.corpo) && /bottom:env\(safe-area-inset-bottom,0px\)/.test(barra.corpo), 'a barra do Lançar fica acima dos botoes do sistema');
assert.ok(/box-shadow:0 env\(safe-area-inset-bottom,0px\) 0 var\(--bg\)/.test(barra.corpo), 'a faixa dos botoes do sistema e pintada com a cor do fundo');

// toda regra presa ao rodape (bottom:0 ou bottom:<algo>) reserva a margem, no bottom ou no padding
const semMargem = presas.filter((r) => !/env\(safe-area-inset-bottom/.test(r.corpo))
  // janelas que cobrem a tela inteira (inset:0 nao tem "bottom:") nao entram aqui; a lista abaixo e de quem tem bottom e nao precisa:
  .filter((r) => !['.dropdown-panel'].includes(r.sel));
assert.deepStrictEqual(semMargem.map((r) => r.sel), [], 'presos ao rodape sem reservar env(safe-area-inset-bottom)');

console.log('rodape-android.test.js: ok');
