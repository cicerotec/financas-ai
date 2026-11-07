// A janela de Tags do celular fica acima do teclado (visualViewport), senao a lista some atras dele (Android). Rode com: node web/janela-tags-teclado.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const ini = app.indexOf('function ajustarJanelaTags(');
const corpo = app.slice(ini, app.indexOf('\n}', ini) + 2);
assert.ok(ini > 0, 'ajustarJanelaTags existe');

function rodar({ innerHeight, vv, aberto = true, largo = false }) {
  const card = { style: { bottom: 'x', maxHeight: 'x' }, classList: { contains: c => c === 'aberto' && aberto } };
  const win = { innerHeight, visualViewport: vv };
  new Function('document', 'window', 'matchMedia', corpo + '; ajustarJanelaTags();')({ getElementById: () => card }, win, () => ({ matches: largo }));
  return card.style;
}
// teclado aberto: a janela sobe para o topo do teclado e cabe na area visivel
let st = rodar({ innerHeight: 900, vv: { height: 500, offsetTop: 0 } });
assert.strictEqual(st.bottom, '400px'); assert.strictEqual(st.maxHeight, '480px');
// a pagina rolou (offsetTop): o teclado continua sendo o que sobra embaixo
st = rodar({ innerHeight: 900, vv: { height: 400, offsetTop: 100 } });
assert.strictEqual(st.bottom, '400px');
// sem teclado, janela fechada ou computador: o CSS manda (estilos limpos)
for (const o of [{ innerHeight: 900, vv: { height: 900, offsetTop: 0 } }, { innerHeight: 900, vv: { height: 500, offsetTop: 0 }, aberto: false }, { innerHeight: 900, vv: { height: 500, offsetTop: 0 }, largo: true }, { innerHeight: 900, vv: undefined }]) {
  st = rodar(o); assert.strictEqual(st.bottom, ''); assert.strictEqual(st.maxHeight, '');
}
// o app liga o ajuste ao foco, ao desfoco e ao visualViewport
assert.ok(/visualViewport\.addEventListener\('resize', ajustarJanelaTags\)/.test(app) && /addEventListener\('focus', \(\)=>\{ cardTags\.classList\.add\('buscando'\); ajustarJanelaTags/.test(app));
console.log('janela-tags-teclado.test.js ok');
