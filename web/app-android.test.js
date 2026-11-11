// Testes do pacote do app Android (item 54, fase 1; spec .ai/specs/app-android): montagem do www/ por ambiente, CORS e .gitignore.
// Rode com: node web/app-android.test.js
const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');
const montar = require('../app-android/montar-www.js');

const aqui = path.resolve(__dirname, '..');
const cfg = (api, id, extra) => 'window.FINANCAS_CONFIG = {\n  apiUrl: "https://' + api + '.lambda-url.sa-east-1.on.aws/",\n  clientId: "' + id + '",\n  loginDomain: "x.auth.sa-east-1.amazoncognito.com"' + (extra || '') + '\n};\n';

// raiz de mentira: um web/ minimo, sem dado real
function raizFalsa(comDev, comProd){
  const raiz = fs.mkdtempSync(path.join(os.tmpdir(), 'financas-app-'));
  fs.mkdirSync(path.join(raiz, 'web', 'js'), {recursive: true});
  fs.mkdirSync(path.join(raiz, 'app-android'));
  fs.writeFileSync(path.join(raiz, 'web', 'index.html'), '<html></html>');
  fs.writeFileSync(path.join(raiz, 'web', 'js', 'app.js'), '//');
  fs.writeFileSync(path.join(raiz, 'web', 'js', 'app.test.js'), '//');
  fs.writeFileSync(path.join(raiz, 'web', 'config.example.js'), '//');
  if(comDev) fs.writeFileSync(path.join(raiz, 'web', 'config.dev.js'), cfg('IDDEV', 'CLIENTEDEV', ',\n  ambiente: "dev"'));
  if(comProd) fs.writeFileSync(path.join(raiz, 'web', 'config.js'), cfg('IDPROD', 'CLIENTEPROD'));
  return raiz;
}

// A-T1: sem o config do ambiente, erro claro e nenhum www/ gerado (mesmo que o do outro ambiente exista)
let r = raizFalsa(false, true);
assert.throws(() => montar({raiz: r, ambiente: 'dev'}), /web\/config\.dev\.js nao existe/);
assert.strictEqual(fs.existsSync(path.join(r, 'app-android', 'www')), false);
r = raizFalsa(true, false);
assert.throws(() => montar({raiz: r, ambiente: 'prod'}), /web\/config\.js nao existe/);
assert.strictEqual(fs.existsSync(path.join(r, 'app-android', 'www')), false);
assert.throws(() => montar({raiz: r, ambiente: 'teste'}), /Ambiente invalido/);

// o config errado de ambiente tambem e recusado (faixa de teste so no dev)
r = raizFalsa(true, true);
fs.writeFileSync(path.join(r, 'web', 'config.js'), cfg('IDPROD', 'CLIENTEPROD', ',\n  ambiente: "dev"'));
assert.throws(() => montar({raiz: r, ambiente: 'prod'}), /nao pode ser o do dev/);
fs.writeFileSync(path.join(r, 'web', 'config.dev.js'), cfg('IDDEV', 'CLIENTEDEV'));
assert.throws(() => montar({raiz: r, ambiente: 'dev'}), /precisa ter ambiente/);

// A-T2: cada pacote tem o apiUrl e o clientId do seu ambiente e nenhum do outro
r = raizFalsa(true, true);
const lerWww = (...p) => fs.readFileSync(path.join(r, 'app-android', 'www', ...p), 'utf8');
montar({raiz: r, ambiente: 'dev'});
let c = lerWww('config.js');
assert.ok(c.includes('IDDEV') && c.includes('CLIENTEDEV') && c.includes('ambiente: "dev"'));
assert.ok(!c.includes('IDPROD') && !c.includes('CLIENTEPROD'));
montar({raiz: r, ambiente: 'prod'}); // refaz o www/ do zero: nada do dev sobra
c = lerWww('config.js');
assert.ok(c.includes('IDPROD') && c.includes('CLIENTEPROD'));
assert.ok(!c.includes('IDDEV') && !c.includes('CLIENTEDEV') && !c.includes('"dev"'));
// o pacote leva as telas e deixa de fora testes, exemplo e o config do outro ambiente
assert.ok(fs.existsSync(path.join(r, 'app-android', 'www', 'index.html')));
assert.ok(fs.existsSync(path.join(r, 'app-android', 'www', 'js', 'app.js')));
for(const f of ['config.example.js', 'config.dev.js', path.join('js', 'app.test.js')]) assert.ok(!fs.existsSync(path.join(r, 'app-android', 'www', f)), f);

// com o web/ de verdade: o config de mentira entra e sai sem sobrar no repositorio (web/config*.js ficam fora do git)
const real = fs.mkdtempSync(path.join(os.tmpdir(), 'financas-real-'));
fs.cpSync(path.join(aqui, 'web'), path.join(real, 'web'), {recursive: true});
fs.mkdirSync(path.join(real, 'app-android'));
fs.writeFileSync(path.join(real, 'web', 'config.js'), cfg('IDPROD', 'CLIENTEPROD'));
const www = montar({raiz: real, ambiente: 'prod'});
for(const f of ['index.html', 'api.js', 'js/nativo.js', 'js/voltar.js', 'css/app.css']) assert.ok(fs.existsSync(path.join(www, f)), f);
assert.ok(!fs.readdirSync(www).some(f => f.endsWith('.test.js')));

// A-T3 (parte local; o `sam validate --lint` roda no CI): AppOrigin entra no CORS da Function URL e as outras origens seguem
const t = fs.readFileSync(path.join(aqui, 'template.yaml'), 'utf8');
assert.match(t, /\n  AppOrigin:\n    Type: String\n    Default: https:\/\/localhost\n/);
assert.ok(t.includes("AllowOrigins: [!Ref FrontOrigin, !Ref AppOrigin, !Sub 'https://${Distribution.DomainName}']"));
assert.ok(t.includes("CallbackURLs: [!Sub '${FrontOrigin}/', !Sub 'https://${Distribution.DomainName}/']"), 'o login do navegador nao muda na fase 1');

// R3: dois sabores, dev com sufixo .dev (instalam juntos) e nome proprio; o app carrega o pacote local (sem server.url)
const g = fs.readFileSync(path.join(aqui, 'app-android', 'android', 'app', 'build.gradle'), 'utf8');
assert.match(g, /applicationId "br\.com\.cicerotec\.financas"/);
assert.match(g, /dev \{[^}]*applicationIdSuffix "\.dev"/);
assert.match(fs.readFileSync(path.join(aqui, 'app-android', 'android', 'app', 'src', 'dev', 'res', 'values', 'strings.xml'), 'utf8'), /Finanças DEV/);
const cap = JSON.parse(fs.readFileSync(path.join(aqui, 'app-android', 'capacitor.config.json'), 'utf8'));
assert.deepStrictEqual([cap.webDir, cap.server.androidScheme, cap.server.url], ['www', 'https', undefined]);

// A-T4 / R8: o que o build gera e a assinatura ficam fora do git
const gi = fs.readFileSync(path.join(aqui, '.gitignore'), 'utf8').split(/\r?\n/);
for(const l of ['app-android/www/', 'app-android/android/app/build/', 'app-android/android/.gradle/', 'local.properties', '*.jks', '*.keystore', '*.apk', '*.aab']) assert.ok(gi.includes(l), l);

console.log('app-android.test.js: ok');
