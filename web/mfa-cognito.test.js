// Testes das chamadas do MFA ao Cognito (web/js/mfa-cognito.js). Spec mfa-opcional: M-T11, M-T12, M-T15, M-T16 (R14, R15, R16, R18, R19).
// Roda em Node, com um Cognito de mentira: node web/mfa-cognito.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const criar = require('./js/mfa-cognito.js');

let falhas = 0, concluido = false;
process.on('exit', () => { if (!concluido) { console.error('mfa-cognito.test.js: o teste nao terminou'); process.exitCode = 1; } });
const fila = [];
const teste = (nome, fn) => fila.push([nome, fn]);

const TOKEN = 'ACCESS-TOKEN-SECRETO.abc.def';
const CFG = { loginDomain: 'financas-dev.auth.sa-east-1.amazoncognito.com', clientId: 'c' };
const resp = (status, corpo) => ({ status, ok: status >= 200 && status < 300, json: async () => { if (corpo === undefined) throw new Error('sem corpo'); return corpo; } });
function cognito(handler, cfg) {
  const d = { chamadas: [], semRede: false };
  d.api = criar({
    cfg: cfg || CFG,
    fetch: async (url, o) => {
      d.chamadas.push({ url, o, alvo: (o.headers['x-amz-target'] || '').replace('AWSCognitoIdentityProviderService.', ''), corpo: JSON.parse(o.body) });
      if (d.semRede) throw new TypeError('Failed to fetch https://cognito-idp.sa-east-1.amazonaws.com/ ' + TOKEN);
      return handler(d.chamadas[d.chamadas.length - 1]);
    }
  });
  return d;
}
const rejeita = async (p, codigo, nome) => { let e = null; try { await p; } catch (x) { e = x; } assert.ok(e, (nome || '') + ': devia falhar'); assert.strictEqual(e.codigo, codigo, (nome || '') + ': codigo ' + (e && e.codigo)); return e; };

teste('M-T11, R14: toda chamada e um POST ao endpoint do Cognito da regiao, com o access token SO no corpo (nunca na URL nem em cabecalho)', async () => {
  const d = cognito((c) => ({
    GetUser: () => resp(200, { UserMFASettingList: [], Username: 'u' }),
    AssociateSoftwareToken: () => resp(200, { SecretCode: 'JBSWY3DPEHPK3PXP' }),
    VerifySoftwareToken: () => resp(200, { Status: 'SUCCESS' }),
    SetUserMFAPreference: () => resp(200, {})
  })[c.alvo]());
  await d.api.estado(TOKEN); await d.api.associar(TOKEN); await d.api.verificar(TOKEN, '123456'); await d.api.ligar(TOKEN); await d.api.desligar(TOKEN);
  assert.deepStrictEqual(d.chamadas.map((c) => c.alvo), ['GetUser', 'AssociateSoftwareToken', 'VerifySoftwareToken', 'SetUserMFAPreference', 'SetUserMFAPreference']);
  for (const c of d.chamadas) {
    assert.strictEqual(c.url, 'https://cognito-idp.sa-east-1.amazonaws.com/', 'a regiao vem do endereco do login');
    assert.strictEqual(c.o.method, 'POST');
    assert.deepStrictEqual(Object.keys(c.o.headers).sort(), ['content-type', 'x-amz-target'], 'so o tipo e o X-Amz-Target; nada de authorization');
    assert.strictEqual(c.o.headers['content-type'], 'application/x-amz-json-1.1');
    assert.strictEqual(c.corpo.AccessToken, TOKEN, 'o token vai no corpo');
    assert.ok(!c.url.includes(TOKEN) && !c.url.includes('?'), 'o token nunca na URL');
    assert.ok(!JSON.stringify(c.o.headers).includes(TOKEN), 'o token nunca em cabecalho');
  }
  const v = d.chamadas[2].corpo, s = d.chamadas[3].corpo, x = d.chamadas[4].corpo;
  assert.deepStrictEqual([v.UserCode, typeof v.FriendlyDeviceName], ['123456', 'string']);
  assert.deepStrictEqual(s.SoftwareTokenMfaSettings, { Enabled: true, PreferredMfa: true }, 'ligar');
  assert.deepStrictEqual(x.SoftwareTokenMfaSettings, { Enabled: false, PreferredMfa: false }, 'desligar');
  assert.deepStrictEqual(Object.keys(s).sort(), ['AccessToken', 'SoftwareTokenMfaSettings']);
});

teste('R14: a regiao vem do endereco do login ou de cfg.regiao; sem nenhuma, "config" e nenhuma chamada sai', async () => {
  const a = cognito(() => resp(200, {}), { loginDomain: 'x.auth.us-east-2.amazoncognito.com' });
  await a.api.estado(TOKEN);
  assert.strictEqual(a.chamadas[0].url, 'https://cognito-idp.us-east-2.amazonaws.com/');
  const b = cognito(() => resp(200, {}), { regiao: 'eu-west-1' });
  await b.api.estado(TOKEN);
  assert.strictEqual(b.chamadas[0].url, 'https://cognito-idp.eu-west-1.amazonaws.com/');
  const c = cognito(() => resp(200, {}), { loginDomain: 'login.invalido.teste', regiao: 'a/b?x' });
  await rejeita(c.api.estado(TOKEN), 'config');
  assert.strictEqual(c.chamadas.length, 0);
});

teste('M-T12, R15: o estado e so o que o GetUser devolve sobre o MFA; e-mail e outros atributos nunca saem do modulo', async () => {
  const d = cognito(() => resp(200, { Username: 'sub-1', UserAttributes: [{ Name: 'email', Value: 'segredo@exemplo.com' }, { Name: 'sub', Value: 's' }], UserMFASettingList: ['SOFTWARE_TOKEN_MFA'], PreferredMfaSetting: 'SOFTWARE_TOKEN_MFA', MFAOptions: [{ DeliveryMedium: 'SMS' }] }));
  const e = await d.api.estado(TOKEN);
  assert.deepStrictEqual(e, { lista: ['SOFTWARE_TOKEN_MFA'], preferido: 'SOFTWARE_TOKEN_MFA' });
  assert.ok(!JSON.stringify(e).includes('segredo@exemplo.com') && !JSON.stringify(e).includes('sub-1'));
  assert.strictEqual(d.api.totpNaLista(e), true);
  const sem = await cognito(() => resp(200, { UserMFASettingList: [] })).api.estado(TOKEN);
  assert.deepStrictEqual(sem, { lista: [], preferido: '' });
  assert.strictEqual(cognito(() => resp(200, {})).api.totpNaLista(sem), false);
  const lixo = await cognito(() => resp(200, { UserMFASettingList: 'x', PreferredMfaSetting: 5 })).api.estado(TOKEN);
  assert.deepStrictEqual(lixo, { lista: [], preferido: '' }, 'tipo inesperado: nada e inventado');
});

teste('R16, M-T13: so Status SUCCESS vale; outro resultado, ou codigo fora do formato, e "codigo_errado" e nada e ligado (nem chama o Cognito sem 6 digitos)', async () => {
  const d = cognito(() => resp(200, { Status: 'ERROR' }));
  await rejeita(d.api.verificar(TOKEN, '123456'), 'codigo_errado', 'Status ERROR');
  assert.strictEqual(d.chamadas.length, 1);
  for (const ruim of ['', '12345', '1234567', 'abcdef', '12 34 5', null, undefined, 123456.5]) {
    const e = cognito(() => resp(200, { Status: 'SUCCESS' }));
    await rejeita(e.api.verificar(TOKEN, ruim), 'codigo_errado', String(ruim));
    assert.strictEqual(e.chamadas.length, 0, 'formato invalido nao chama o Cognito: ' + ruim);
  }
  const ok = cognito(() => resp(200, { Status: 'SUCCESS' }));
  assert.strictEqual(await ok.api.verificar(TOKEN, ' 123 456 '), true, 'espacos no meio sao aceitos');
  assert.strictEqual(ok.chamadas[0].corpo.UserCode, '123456');
  assert.ok((await cognito(() => resp(200, { SecretCode: 'jbsw y3dp ehpk 3pxp' })).api.associar(TOKEN)).chave === 'JBSWY3DPEHPK3PXP', 'a chave e normalizada');
});

teste('R16, R19: a chave devolvida pelo Cognito e validada (base32) antes de virar QR; chave estranha vira erro, nunca um QR com lixo', async () => {
  for (const ruim of ['', 'curta', 'JBSWY3DPEHPK3PX!', 'JBSWY3DPEHPK3PX1', undefined]) {
    await rejeita(cognito(() => resp(200, { SecretCode: ruim })).api.associar(TOKEN), 'servico', String(ruim));
  }
  const api = cognito(() => resp(200, {})).api;
  assert.throws(() => api.uriOtpauth('curta', 'Finanças'), (e) => e.codigo === 'servico');
  assert.strictEqual(api.uriOtpauth('jbswy3dpehpk3pxp', 'Finanças'), 'otpauth://totp/Finan%C3%A7as?secret=JBSWY3DPEHPK3PXP&issuer=Finan%C3%A7as');
  assert.strictEqual(api.uriOtpauth('JBSWY3DPEHPK3PXP', 'Finanças DEV'), 'otpauth://totp/Finan%C3%A7as%20DEV?secret=JBSWY3DPEHPK3PXP&issuer=Finan%C3%A7as%20DEV');
  assert.ok(!/@/.test(api.uriOtpauth('JBSWY3DPEHPK3PXP')), 'R15: o texto do QR nunca leva e-mail');
  assert.strictEqual(api.formatarChave('JBSWY3DPEHPK3PXP'), 'JBSW Y3DP EHPK 3PXP');
});

teste('M-T15, R18: o erro do Cognito vira um codigo curto e uma frase nossa; o texto, o corpo e o token do Cognito nunca sobem', async () => {
  const casos = [
    [400, 'CodeMismatchException', 'codigo_errado'], [400, 'EnableSoftwareTokenMFAException', 'codigo_errado'], [400, 'ExpiredCodeException', 'codigo_errado'],
    [400, 'LimitExceededException', 'tentativas'], [429, 'TooManyRequestsException', 'tentativas'], [400, 'com.amazon.coral.availability#TooManyRequestsException', 'tentativas'],
    [400, 'NotAuthorizedException', 'permissao'], [403, 'AccessDeniedException', 'permissao'], [400, 'InvalidParameterException', 'permissao'], [401, undefined, 'permissao'],
    [500, 'InternalErrorException', 'servico'], [503, undefined, 'servico'], [400, 'AlgoNovoException', 'servico']
  ];
  for (const [status, nome, codigo] of casos) {
    const d = cognito(() => resp(status, { __type: nome, message: 'detalhe tecnico com ' + TOKEN + ' e segredo@exemplo.com' }));
    const e = await rejeita(d.api.estado(TOKEN), codigo, status + ' ' + nome);
    assert.ok(!e.message.includes(TOKEN) && !e.message.includes('segredo') && !e.message.includes('tecnico') && !e.message.includes('Exception'), 'a mensagem e nossa: ' + e.message);
    assert.strictEqual(e.message, d.api.MENSAGENS[codigo]);
  }
  const semCorpo = cognito(() => resp(502));
  await rejeita(semCorpo.api.estado(TOKEN), 'servico', 'sem corpo');
  // sem internet: o erro do fetch leva a URL e ate o token; nada disso sobe
  const rede = cognito(() => resp(200, {}));
  rede.semRede = true;
  const e = await rejeita(rede.api.estado(TOKEN), 'rede');
  assert.ok(!e.message.includes('http') && !e.message.includes(TOKEN) && !e.message.includes('fetch'), e.message);
  for (const f of ['associar', 'ligar', 'desligar']) await rejeita(rede.api[f](TOKEN), 'rede', f);
  await rejeita(rede.api.verificar(TOKEN, '123456'), 'rede', 'verificar');
  // sem token na mao: recusa antes de sair
  const sem = cognito(() => resp(200, {}));
  for (const t of ['', null, undefined, 5]) { await rejeita(sem.api.estado(t), 'permissao'); await rejeita(sem.api.ligar(t), 'permissao'); }
  assert.strictEqual(sem.chamadas.length, 0);
});

teste('R19, M-T16: o modulo nao usa armazenamento, area de transferencia, log, nem outra rede alem do endpoint do Cognito; a chave so volta ao chamador', async () => {
  const fonte = fs.readFileSync(path.join(__dirname, 'js', 'mfa-cognito.js'), 'utf8').replace(/(^|\s)\/\/.*$/gm, '$1'); // tira os comentarios sem comer o https:// das strings
  for (const proibido of [/localStorage/, /sessionStorage/, /indexedDB/, /clipboard/, /console\./, /sendBeacon/, /XMLHttpRequest/, /WebSocket/, /document\./, /\.setItem\(/]) {
    assert.ok(!proibido.test(fonte), 'mfa-cognito.js nao pode ter ' + proibido);
  }
  assert.strictEqual((fonte.match(/env\.fetch\(/g) || []).length, 1, 'um unico ponto de rede');
  assert.ok(fonte.includes("'https://cognito-idp.'"), 'o unico destino e o endpoint do Cognito');
  // a chave nao e guardada no modulo: duas chamadas devolvem valores independentes
  let n = 0;
  const d = cognito(() => resp(200, { SecretCode: n++ === 0 ? 'AAAAAAAAAAAAAAAA' : 'BBBBBBBBBBBBBBBB' }));
  assert.strictEqual((await d.api.associar(TOKEN)).chave, 'AAAAAAAAAAAAAAAA');
  assert.strictEqual((await d.api.associar(TOKEN)).chave, 'BBBBBBBBBBBBBBBB');
});

(async () => {
  for (const [nome, fn] of fila) {
    try { await fn(); console.log('ok   ' + nome); } catch (e) { falhas++; console.error('FALHOU: ' + nome + '\n  ' + (e.stack || e.message).split('\n').slice(0, 4).join('\n  ')); }
  }
  concluido = true;
  if (falhas) { console.error(falhas + ' teste(s) falharam'); process.exit(1); }
  console.log('mfa-cognito.test.js: ok');
})();
