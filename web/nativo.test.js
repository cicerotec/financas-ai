// Teste do botao voltar fisico da casca Android (web/js/nativo.js), sem aparelho: Capacitor e historico de mentira.
// Rode com: node web/nativo.test.js
const assert = require('assert');
const criarNativo = require('./js/nativo.js');

function cenario(nativo){
  const c = { ouvintes: {}, voltou: 0, minimizou: 0 };
  const App = { addListener(nome, fn){ c.ouvintes[nome] = fn; }, minimizeApp(){ c.minimizou++; } };
  const Capacitor = nativo === null ? undefined : { isNativePlatform: () => nativo, Plugins: { App } };
  const history = { back(){ c.voltou++; } };
  c.ligou = criarNativo({Capacitor, history}).iniciar();
  return c;
}

// no navegador e na PWA (sem Capacitor, ou Capacitor da web) nada e ligado
assert.strictEqual(cenario(null).ligou, false);
const web = cenario(false);
assert.strictEqual(web.ligou, false);
assert.strictEqual(web.ouvintes.backButton, undefined);

// no app: com entrada anterior no historico, o voltar anda no historico (voltar.js fecha a camada ou troca a aba)
const app = cenario(true);
assert.strictEqual(app.ligou, true);
app.ouvintes.backButton({canGoBack: true});
assert.deepStrictEqual([app.voltou, app.minimizou], [1, 0]);
// na raiz, minimiza o app e nao mexe no historico
app.ouvintes.backButton({canGoBack: false});
assert.deepStrictEqual([app.voltou, app.minimizou], [1, 1]);

// sem o plugin App (app antigo ou plugin nao sincronizado) nao quebra
assert.strictEqual(criarNativo({Capacitor: {isNativePlatform: () => true, Plugins: {}}, history: {}}).iniciar(), false);

console.log('nativo.test.js: ok');
