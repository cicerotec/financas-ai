// Teste do splash nativo e do icone adaptativo do app Android (item 54, fase 3b). Demo: .ai/demos/demo-abertura-v4-splash-nativo.html.
// O splash do Android mostra o icone do app no centro, sobre o fundo do app, e espera a pagina carregar; a Abertura, no app, usa o
// icone do mesmo tamanho (192 px) e no mesmo lugar, para a passagem nao dar salto nem ter tela vazia (medido no aparelho do dono
// em 2026-11-12: antes havia ~1 s de tela cinza vazia e um quadro preto). Rode com: node web/splash-android.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const criarAbertura = require('./js/abertura.js');

const raiz = path.resolve(__dirname, '..');
const res = (...p) => path.join(raiz, 'app-android', 'android', 'app', 'src', ...p);
const ler = (f) => fs.readFileSync(f, 'utf8').replace(/\r\n/g, '\n');
const css = ler(path.join(raiz, 'web', 'css', 'app.css'));

// as cores do splash sao o fundo do app nos dois temas (token --bg)
const bgClaro = css.match(/:root\{\s*--bg:(#[0-9a-f]{6})/i)[1];
const bgEscuro = css.match(/:root\[data-theme="dark"\]\{\s*--bg:(#[0-9a-f]{6})/i)[1];
assert.ok(ler(res('main', 'res', 'values', 'colors.xml')).includes('<color name="fundo">' + bgClaro + '</color>'), 'fundo claro igual ao --bg claro');
assert.ok(ler(res('main', 'res', 'values-night', 'colors.xml')).includes('<color name="fundo">' + bgEscuro + '</color>'), 'fundo escuro igual ao --bg escuro');

// o tema de abertura: icone no centro, fundo do app, e a janela seguinte nasce com a mesma cor (sem quadro preto)
const estilos = ler(res('main', 'res', 'values', 'styles.xml'));
const launch = estilos.slice(estilos.indexOf('AppTheme.NoActionBarLaunch'));
for (const item of ['<item name="windowSplashScreenBackground">@color/fundo</item>', '<item name="windowSplashScreenAnimatedIcon">@drawable/splash_icone</item>', '<item name="postSplashScreenTheme">@style/AppTheme.NoActionBar</item>']) assert.ok(launch.includes(item), item);
assert.ok(/AppTheme\.NoActionBar" parent[^]*?<item name="android:windowBackground">@color\/fundo<\/item>/.test(estilos), 'a janela do app nasce com o fundo do app');
assert.strictEqual(/@drawable\/splash</.test(estilos), false, 'o splash.png padrao do Capacitor nao e mais usado');
const sobras = fs.readdirSync(res('main', 'res')).filter((d) => fs.existsSync(res('main', 'res', d, 'splash.png')));
assert.deepStrictEqual(sobras, [], 'splash.png removido');

// o icone do splash: quadro de 288 dp, com o recorte da zona segura do icone (o da Abertura) no circulo de 192 dp; em dev leva a faixa
const html = ler(path.join(raiz, 'web', 'index.html'));
assert.ok(html.includes('id="abIc" viewBox="51.2 51.2 409.6 409.6"'), 'a Abertura mostra o recorte da zona segura');
for (const sabor of ['main', 'dev']) {
  const v = ler(res(sabor, 'res', 'drawable', 'splash_icone.xml'));
  assert.ok(/android:width="288dp" android:height="288dp" android:viewportWidth="614\.4" android:viewportHeight="614\.4"/.test(v), sabor + ': 409,6 unidades em 192 dp');
  assert.ok(v.includes('android:translateX="51.2" android:translateY="51.2"') && v.includes('<clip-path'), sabor + ': recorte em circulo no centro');
  for (const cor of ['#14213d', '#4cc9a0', '#ffd166']) assert.ok(v.includes(cor), sabor + ': cor ' + cor);
  assert.strictEqual(v.includes('#d97b2f'), sabor === 'dev', sabor + ': a faixa DEV so no ambiente de teste');
}

// o splash espera a pagina (senao sai antes e sobra tela vazia), com um limite para nao ficar preso
const main = ler(res('main', 'java', 'br', 'com', 'cicerotec', 'financas', 'MainActivity.java'));
assert.ok(/SplashScreen\.installSplashScreen\(this\)[^]*setKeepOnScreenCondition\(\(\) -> !paginaPronta\)[^]*super\.onCreate/.test(main), 'o splash e instalado antes do super.onCreate e fica ate a pagina');
assert.ok(/onPageLoaded[^]*paginaPronta = true/.test(main) && /postDelayed\(\(\) -> paginaPronta = true, ESPERA_MAXIMA_MS\)/.test(main), 'sai com a pagina carregada ou no limite');
assert.ok(main.includes('setBackgroundColor(ContextCompat.getColor(this, R.color.fundo))'), 'a WebView nasce com o fundo do app');

// icone adaptativo: fundo ate a borda e a frente (barras a 82%) na area que nenhum recorte corta; os PNG continuam para Android antigo
for (const nome of ['ic_launcher.xml', 'ic_launcher_round.xml']) {
  const a = ler(res('main', 'res', 'mipmap-anydpi-v26', nome));
  assert.ok(a.includes('<background android:drawable="@color/icone_fundo" />') && a.includes('<foreground android:drawable="@drawable/icone_frente" />'), nome);
}
assert.ok(ler(res('main', 'res', 'values', 'colors.xml')).includes('<color name="icone_fundo">#14213d</color>'));
for (const sabor of ['main', 'dev']) {
  const f = ler(res(sabor, 'res', 'drawable', 'icone_frente.xml'));
  assert.ok(f.includes('android:viewportWidth="512"') && f.includes('android:scaleX="0.82" android:scaleY="0.82"'), sabor + ': barras reduzidas');
  assert.strictEqual(f.includes('#14213d'), false, sabor + ': a frente nao leva o fundo');
  assert.strictEqual(f.includes('#d97b2f'), sabor === 'dev');
}
for (const d of ['mipmap-mdpi', 'mipmap-xxxhdpi']) assert.ok(fs.existsSync(res('main', 'res', d, 'ic_launcher.png')), 'PNG legado em ' + d);

// no app, a Abertura e o bloqueio usam o icone de 192 px (o do splash); no navegador a Abertura continua com 224 px
assert.ok(css.includes('.ab.nativo .ab-ic{width:min(192px,34vh);height:min(192px,34vh);}'));
assert.ok(/\.ab-ic\{[^}]*224px/.test(css), 'no navegador, 224 px');
assert.ok(/\.bq-ic\{width:min\(192px,34vh\);height:min\(192px,34vh\)/.test(css), 'o bloqueio so existe no app');
const amb = (nativo) => {
  const cl = new Set();
  const el = { hidden: false, textContent: '', classList: { add: (c) => cl.add(c), remove() {}, toggle() {} }, onclick: null };
  criarAbertura({ el: () => el, agora: () => 0, setTimeout() {}, fundo() {}, recarregar() {}, sair() {}, entrar() {}, nativo });
  return cl.has('nativo');
};
assert.strictEqual(amb(true), true, 'no app a Abertura leva a classe nativo');
assert.strictEqual(amb(false), false, 'no navegador nao');
// em dev, o icone da tela de bloqueio tambem leva a faixa DEV, como o do splash
assert.ok(/querySelectorAll\('#abIc, #bloqueio \.bq-ic'\)/.test(ler(path.join(raiz, 'web', 'js', 'ambiente.js'))), 'a faixa DEV vai aos dois icones');

console.log('splash-android.test.js: ok');
