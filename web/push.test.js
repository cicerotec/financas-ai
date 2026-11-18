// Testes do push no app (item 54, fase 4; spec .ai/specs/app-android-push, P-T19 a P-T21): plugin, API, armazenamento e DOM de mentira.
// Rode com: node web/push.test.js
const assert = require('assert');
const criarPush = require('./js/push.js');

const TOKEN = 'tok-ficticio-do-aparelho-aaaaaaaaaaaa';
const TOKEN2 = 'tok-ficticio-do-aparelho-bbbbbbbbbbbb';

function mem(){ const d = {}; return {getItem: k => (k in d ? d[k] : null), setItem: (k, v) => { d[k] = String(v); }, removeItem: k => { delete d[k]; }, d}; }
const digest = async t => require('crypto').createHash('sha256').update(t).digest('hex'); // o mesmo SHA-256 que o app usa (crypto.subtle)

function montar(op){
  op = op || {};
  const ev = [], arm = mem(), dom = {};
  let agora = op.agora || 1700000000000;
  const plugin = {
    disponivel: async () => { ev.push('disponivel'); return {disponivel: op.disponivel !== false}; },
    permissao: async () => { ev.push('permissao'); return {permissao: op.permissao || 'concedida'}; },
    pedirPermissao: async () => { ev.push('pedirPermissao'); return {permissao: op.aoPedir || 'concedida'}; },
    token: async () => { ev.push('token'); if(op.semToken) throw new Error('x'); return {token: op.token || TOKEN}; },
    apagarToken: async () => { ev.push('apagarToken'); if(op.apagarFalha) throw new Error('x'); },
    mudou: async () => { ev.push('mudou'); return {mudou: !!op.mudou}; },
    marcarRegistrado: async () => { ev.push('marcarRegistrado'); },
    abrirConfiguracoes: async () => { ev.push('abrirConfiguracoes'); }
  };
  const api = {
    registrar: async (t, s) => { ev.push('api.registrar:' + s + ':' + (arm.d['financas.push'] ? 'gravado' : 'vazio')); if(op.apiErro) throw op.apiErro; return op.aparelhos || 1; },
    remover: async (t, s) => { ev.push('api.remover:' + s); if(op.apiErro) throw op.apiErro; return 0; },
    sair: async t => { ev.push('api.sair'); if(op.sairPendura) return new Promise(() => {}); if(op.sairFalha) throw new TypeError('rede'); return {removidos: 1}; },
    testar: async c => { ev.push('api.testar:' + c); if(op.apiErro) throw op.apiErro; }
  };
  const toasts = [];
  const el = id => dom[id] || (dom[id] = {id, checked: false, disabled: false, hidden: false, textContent: '', classList: {toggle(){}, add(){}, remove(){}}});
  const env = {
    Capacitor: {isNativePlatform: () => op.nativo !== false, Plugins: op.semPlugin ? {} : {Push: plugin}},
    armazenamento: arm, agora: () => agora, digest, el, api,
    esperar: ms => { if(ms === 10000) ev.push('esperar:10000'); return new Promise(r => setTimeout(r, op.espera === undefined ? 5 : op.espera)); },
    toast: (m, e) => toasts.push([m, !!e]),
    explicar: async () => { ev.push('explicar'); return op.explicar !== false; },
    sid: () => op.sid === undefined ? 's1' : op.sid
  };
  return {push: criarPush(env), ev, arm, dom, toasts, avancar: ms => { agora += ms; }};
}
const gravado = m => JSON.parse(m.arm.d['financas.push'] || 'null');
const semToken = m => assert.ok(!JSON.stringify(m.arm.d).includes('tok-ficticio'), 'o token nunca e guardado no aparelho');

(async () => {
  // P-T19: ligar chama permissao, token e o PUT, e SO DEPOIS grava financas.push (so a data e a impressao, nunca o token)
  let m = montar({aparelhos: 3});
  let r = await m.push.ligar('s1');
  assert.deepStrictEqual([r.ok, r.aparelhos], [true, 3]);
  assert.deepStrictEqual(m.ev, ['disponivel', 'permissao', 'token', 'api.registrar:s1:vazio', 'marcarRegistrado']);
  assert.deepStrictEqual(Object.keys(gravado(m)), ['s1']);
  assert.ok(gravado(m).s1.em > 0 && gravado(m).s1.h && gravado(m).s1.h.length === 16);
  semToken(m);
  assert.ok(m.push.ligadoNo('s1') && !m.push.ligadoNo('s2'));

  // Android 13 ou mais, ainda sem perguntar: pede uma vez e liga
  m = montar({permissao: 'pergunte'});
  r = await m.push.ligar('s1');
  assert.ok(r.ok && m.ev.filter(e => e === 'pedirPermissao').length === 1);

  // permissao negada pelo Android: nao pede de novo, nao le o token, nao chama a API, nao grava (R17)
  m = montar({permissao: 'negada'});
  r = await m.push.ligar('s1');
  assert.deepStrictEqual([r.ok, r.motivo, r.config], [false, 'negada', true]);
  assert.ok(/nas configurações do app/.test(r.texto));
  assert.deepStrictEqual(m.ev, ['disponivel', 'permissao']);
  assert.strictEqual(m.arm.d['financas.push'], undefined);

  // a pessoa nega no pedido do sistema: nada e ligado
  m = montar({permissao: 'pergunte', aoPedir: 'negada'});
  r = await m.push.ligar('s1');
  assert.deepStrictEqual([r.ok, r.motivo, r.config], [false, 'recusada', true]);
  assert.ok(!m.ev.some(e => e.startsWith('api.') || e === 'token') && m.arm.d['financas.push'] === undefined);

  // R16: indisponivel (sem Google Play ou sem google-services.json) nao grava nem chama a API; o plugin ausente tambem
  m = montar({disponivel: false});
  r = await m.push.ligar('s1');
  assert.deepStrictEqual([r.ok, r.motivo, r.texto], [false, 'indisponivel', 'Este aparelho não recebe avisos do app.']);
  assert.deepStrictEqual(m.ev, ['disponivel']);
  m = montar({semPlugin: true});
  assert.strictEqual((await m.push.ligar('s1')).motivo, 'indisponivel');
  m = montar({semToken: true});
  assert.strictEqual((await m.push.ligar('s1')).motivo, 'indisponivel');
  assert.strictEqual(m.arm.d['financas.push'], undefined);

  // limite de 5 aparelhos (409): a frase e a do servidor; nada gravado
  const limite = Object.assign(new Error('Você já tem 5 aparelhos neste espaço. Desligue um deles para ligar este.'), {code: 409});
  m = montar({apiErro: limite});
  r = await m.push.ligar('s1');
  assert.deepStrictEqual([r.ok, r.motivo, r.texto], [false, 'limite', limite.message]);
  assert.strictEqual(m.arm.d['financas.push'], undefined);

  // falha de rede (sem codigo) e erro do servidor: a chave volta desligada, com "Tentar de novo"
  m = montar({apiErro: new TypeError('Failed to fetch')});
  r = await m.push.ligar('s1');
  assert.deepStrictEqual([r.ok, r.motivo, r.denovo], [false, 'rede', true]);
  assert.strictEqual(r.texto, 'Não consegui ligar agora (sem conexão). Tente de novo.');
  assert.strictEqual(m.arm.d['financas.push'], undefined);
  m = montar({apiErro: Object.assign(new Error('HTTP 500'), {code: 500})});
  assert.strictEqual((await m.push.ligar('s1')).motivo, 'erro');

  // desligar: tira do servidor e do aparelho; falha deixa ligada
  m = montar();
  await m.push.ligar('s1');
  r = await m.push.desligar('s1');
  assert.ok(r.ok && m.ev.includes('api.remover:s1') && !m.push.ligadoNo('s1') && m.arm.d['financas.push'] === undefined);
  m = montar();
  await m.push.ligar('s1');
  m.ev.length = 0;
  const m2 = montar({apiErro: new TypeError('x')}); // outro aparelho de mentira, ja ligado, que nao consegue desligar
  m2.arm.setItem('financas.push', JSON.stringify({s1: {em: 1, h: 'x'}}));
  r = await m2.push.desligar('s1');
  assert.deepStrictEqual([r.ok, r.texto], [false, 'Não consegui desligar agora (sem conexão). Tente de novo.']);
  assert.ok(m2.push.ligadoNo('s1'), 'falha ao desligar: a chave continua ligada');

  // teste de aviso: a frase do servidor aparece; sem codigo e rede
  m = montar();
  assert.ok((await m.push.testar()).ok && m.ev.includes('api.testar:push'));
  m = montar({apiErro: Object.assign(new Error('Nenhum aparelho registrado.'), {code: 400})});
  assert.strictEqual((await m.push.testar()).texto, 'Nenhum aparelho registrado.');
  m = montar({apiErro: new TypeError('x')});
  assert.strictEqual((await m.push.testar()).texto, 'Não consegui enviar o teste (sem conexão).');

  // P-T19 / R15: registrar de novo SO quando precisa
  const SETE = 7 * 24 * 3600 * 1000;
  m = montar();
  await m.push.ligar('s1');
  m.ev.length = 0;
  m.avancar(SETE - 1000);
  assert.strictEqual((await m.push.manterRegistro()).registrados, 0); // mesmo token, menos de 7 dias, sem marca: nao escreve nada
  assert.ok(!m.ev.some(e => e.startsWith('api.')), 'a abertura comum nao escreve');
  m.avancar(2000); // 7 dias
  assert.strictEqual((await m.push.manterRegistro()).registrados, 1);
  assert.ok(m.ev.includes('api.registrar:s1:gravado'));
  // token diferente do guardado (impressao)
  m = montar();
  await m.push.ligar('s1');
  const outro = montar({token: TOKEN2});
  outro.arm.setItem('financas.push', m.arm.d['financas.push']);
  assert.strictEqual((await outro.push.manterRegistro()).registrados, 1);
  // marca do onNewToken: registra e limpa a marca; so se tudo deu certo
  m = montar({mudou: true});
  await m.push.ligar('s1');
  m.ev.length = 0;
  assert.strictEqual((await m.push.manterRegistro()).registrados, 1);
  assert.ok(m.ev.includes('marcarRegistrado'));
  const falha = montar({mudou: true, apiErro: new TypeError('x')});
  falha.arm.setItem('financas.push', JSON.stringify({s1: {em: 1, h: 'x'}}));
  assert.strictEqual((await falha.push.manterRegistro()).registrados, 0); // falha de rede: silenciosa
  assert.ok(!falha.ev.includes('marcarRegistrado') && falha.push.ligadoNo('s1'), 'a chave nao some por falha de rede');
  // varios espacos: cada um por si; sem nenhum espaco ligado nao le nem o token
  const vazio = montar();
  assert.strictEqual((await vazio.push.manterRegistro()).registrados, 0);
  assert.deepStrictEqual(vazio.ev, []);

  // P-T20 (parte do front): Sair tira o aparelho de todos os espacos, invalida o token e limpa o aparelho, nesta ordem
  m = montar();
  await m.push.ligar('s1');
  m.ev.length = 0;
  await m.push.sair();
  assert.deepStrictEqual(m.ev, ['token', 'api.sair', 'apagarToken']);
  assert.strictEqual(m.arm.d['financas.push'], undefined);
  // falha de rede no /push/sair e no apagarToken nao impedem o resto
  m = montar({sairFalha: true, apagarFalha: true});
  await m.push.ligar('s1');
  m.ev.length = 0;
  await m.push.sair();
  assert.deepStrictEqual(m.ev, ['token', 'api.sair', 'apagarToken']);
  assert.strictEqual(m.arm.d['financas.push'], undefined);
  // sem rede de verdade (a chamada nunca volta): o Sair espera pouco e segue
  m = montar({sairPendura: true, espera: 5});
  await m.push.ligar('s1');
  await m.push.sair();
  assert.strictEqual(m.arm.d['financas.push'], undefined);
  // sem token (plugin falha): ainda invalida e limpa
  m = montar({semToken: true});
  m.arm.setItem('financas.push', JSON.stringify({s1: {em: 1, h: 'x'}}));
  await m.push.sair();
  assert.ok(m.ev.includes('apagarToken') && !m.ev.includes('api.sair') && m.arm.d['financas.push'] === undefined);

  // P-T21: no navegador e na PWA nada existe nem acontece
  m = montar({nativo: false});
  assert.strictEqual(m.push.iniciar(), false);
  assert.strictEqual(m.push.ligarCartao(), false);
  assert.strictEqual((await m.push.manterRegistro()).registrados, 0);
  assert.deepStrictEqual(m.ev, []);
  assert.strictEqual(m.push.disponivel(), false);

  // o cartao: ligar passa pela explicacao, depois pela permissao; "Agora nao" nao faz nada; o texto de apoio diz quantos aparelhos
  m = montar({aparelhos: 2});
  assert.strictEqual(m.push.ligarCartao(), true);
  const d = m.dom;
  assert.strictEqual(d.pushCard.hidden, false);
  await new Promise(r => setImmediate(r));
  assert.strictEqual(d.swPush.checked, false);
  assert.strictEqual(d.pushApoio.textContent, 'Vale só neste espaço e neste aparelho.');
  d.swPush.checked = true;
  await d.swPush.onchange();
  assert.ok(m.ev.indexOf('explicar') >= 0 && m.ev.indexOf('explicar') < m.ev.indexOf('permissao'), 'a explicacao vem antes do pedido');
  assert.strictEqual(d.swPush.checked, true);
  assert.strictEqual(d.pushApoio.textContent, '2 aparelhos recebem avisos neste espaço.');
  assert.strictEqual(d.btnPushTeste.hidden, false);
  assert.ok(m.toasts.some(t => t[0] === '✓ Este aparelho recebe os avisos'));
  // o teste espera 10 s ANTES de chamar a API (com o app aberto o Android nao mostra a notificacao) e avisa a pessoa
  const antes = m.ev.length;
  const clique = d.btnPushTeste.onclick();
  assert.ok(/10 segundos/.test(d.pushMsg.textContent) && d.btnPushTeste.disabled, 'avisa e trava o botao durante a espera');
  await d.btnPushTeste.onclick(); // segundo clique durante a espera: ignorado
  await clique;
  assert.deepStrictEqual(m.ev.slice(antes), ['esperar:10000', 'api.testar:push'], 'espera 10 s e so entao chama a API, uma vez');
  assert.ok(!d.btnPushTeste.disabled && d.pushMsg.textContent === '');
  assert.ok(m.toasts.some(t => /Teste enviado/.test(t[0])));
  d.swPush.checked = false;
  await d.swPush.onchange();
  assert.strictEqual(d.swPush.checked, false);
  assert.strictEqual(d.btnPushTeste.hidden, true);
  m = montar({explicar: false});
  m.push.ligarCartao();
  m.dom.swPush.checked = true;
  await m.dom.swPush.onchange();
  assert.ok(!m.ev.includes('permissao') && !m.ev.includes('token') && !m.ev.some(e => e.startsWith('api.')) && m.dom.swPush.checked === false, 'Agora nao: nada acontece');
  // aparelho indisponivel: a chave fica desabilitada com a frase (R16)
  m = montar({disponivel: false});
  m.push.ligarCartao();
  await new Promise(r => setImmediate(r));
  assert.ok(m.dom.swPush.disabled);
  assert.strictEqual(m.dom.pushApoio.textContent, 'Este aparelho não recebe avisos do app.');
  // permissao bloqueada: a frase em vermelho e o botao das configuracoes
  m = montar({permissao: 'negada'});
  m.push.ligarCartao();
  m.dom.swPush.checked = true;
  await m.dom.swPush.onchange();
  assert.strictEqual(m.dom.btnPushConfig.hidden, false);
  assert.ok(/desligadas para o Finanças/.test(m.dom.pushMsg.textContent));
  await m.dom.btnPushConfig.onclick();
  assert.ok(m.ev.includes('abrirConfiguracoes'));
  assert.strictEqual(m.dom.swPush.checked, false);

  console.log('push.test.js: ok');
})().catch(e => { console.error(e); process.exit(1); });
