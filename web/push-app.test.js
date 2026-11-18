// Testes estaticos do lado Android do push (item 54, fase 4; spec .ai/specs/app-android-push, R14, R16, R21 e R23): o que da para
// provar lendo o projeto, sem aparelho. O que depende do aparelho e roteiro manual do PR (P-T22 a P-T29).
// Rode com: node web/push-app.test.js
const assert = require('assert');
const fs = require('fs');
const path = require('path');

const android = path.resolve(__dirname, '..', 'app-android', 'android');
const ler = (...p) => fs.readFileSync(path.join(android, ...p), 'utf8').replace(/\r\n/g, '\n'); // no Windows o git entrega CRLF
const java = (nome) => ler('app', 'src', 'main', 'java', 'br', 'com', 'cicerotec', 'financas', nome);
const gradle = ler('app', 'build.gradle');
const manifest = ler('app', 'src', 'main', 'AndroidManifest.xml');
const plugin = java('PushPlugin.java');
const servico = java('FinancasMessagingService.java');

// R14: o SDK do Firebase Messaging (pelo BoM) e a unica dependencia nova de terceiros; nada de Analytics, Crashlytics, Auth etc.
assert.match(gradle, /implementation platform\("com\.google\.firebase:firebase-bom:\$firebaseBomVersion"\)/);
assert.match(gradle, /implementation "com\.google\.firebase:firebase-messaging"/);
const firebase = gradle.match(/com\.google\.firebase:[\w-]+/g).filter((x, i, l) => l.indexOf(x) === i).sort();
assert.deepStrictEqual(firebase, ['com.google.firebase:firebase-bom', 'com.google.firebase:firebase-messaging']);
assert.match(ler('variables.gradle'), /firebaseBomVersion = '\d+\.\d+\.\d+'/);

// P2/R16: o plugin google-services so e aplicado quando existe o arquivo (sem ele o app compila e o plugin diz "indisponivel")
assert.match(gradle, /if \(servicesJSON\.text\) \{\s*apply plugin: 'com\.google\.gms\.google-services'/);
assert.match(ler('build.gradle'), /classpath 'com\.google\.gms:google-services:\d+\.\d+\.\d+'/);

// R14: o manifest declara a permissao, o servico (fechado a outros apps) e o canal que o servidor pede
assert.match(manifest, /<uses-permission android:name="android\.permission\.POST_NOTIFICATIONS" \/>/);
assert.match(manifest, /<service\s+android:name="\.FinancasMessagingService"\s+android:exported="false">\s*<intent-filter>\s*<action android:name="com\.google\.firebase\.MESSAGING_EVENT" \/>/);
assert.match(manifest, /com\.google\.firebase\.messaging\.default_notification_channel_id"\s+android:value="avisos"/);
assert.match(manifest, /<queries>\s*<package android:name="com\.google\.android\.gms" \/>/);

// o canal do app e o que o servidor pede na mensagem (backend/adapter/notificacao/fcm.go)
assert.match(plugin, /CANAL_AVISOS = "avisos"/);
const fcm = fs.readFileSync(path.resolve(__dirname, '..', 'backend', 'adapter', 'notificacao', 'fcm.go'), 'utf8');
assert.match(fcm, /canalDeNotificacao = "avisos"/);

// R14: plugin registrado na MainActivity (antes do super.onCreate) e as operacoes que o JavaScript usa
const main = java('MainActivity.java');
assert.ok(main.indexOf('registerPlugin(PushPlugin.class)') > 0 && main.indexOf('registerPlugin(PushPlugin.class)') < main.indexOf('super.onCreate'));
assert.match(plugin, /@CapacitorPlugin\(\s*name = "Push"/);
for (const op of ['disponivel', 'permissao', 'pedirPermissao', 'token', 'apagarToken', 'abrirConfiguracoes', 'mudou', 'marcarRegistrado']) {
  assert.match(plugin, new RegExp('@PluginMethod\\s+public void ' + op + '\\(', 'm'), 'falta a operacao ' + op);
}

// R14: o servico ignora a mensagem e so marca "o token mudou" (um booleano); o token nunca e gravado
assert.match(servico, /onMessageReceived\([^)]*\) \{\s*\/\/[^\n]*\n\s*\}/, 'onMessageReceived tem de ficar vazio');
assert.match(servico, /putBoolean\(PushPlugin\.MUDOU, true\)/);
assert.ok(!/putString/.test(servico) && !/putString/.test(plugin), 'o token nao pode ser gravado no aparelho pelo codigo nativo');

// R23: nada escreve log nem imprime o que quer que seja
for (const [nome, fonte] of [['PushPlugin', plugin], ['FinancasMessagingService', servico]]) {
  const codigo = fonte.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/.*$/gm, '');
  assert.ok(!/\bLog\.|System\.(out|err)|printStackTrace|Timber|Toast/.test(codigo), nome + ' nao escreve log nem mostra o token');
}

console.log('push-app.test.js: ok');
