// Testes da chave do Telegram em Listas / Avisos (demo-push-v2-telegram-chave): DOM e API de mentira.
// Rode com: node web/telegram.test.js
const assert = require('assert');
const criarTelegram = require('./js/telegram.js');

function mem(init){ const d = Object.assign({}, init); return {d, getItem: k => (k in d ? d[k] : null), setItem: (k, v) => { d[k] = String(v); }, removeItem: k => { delete d[k]; }}; }

function montar(op){
  op = op || {};
  const ev = [], dom = {}, toasts = [], arm = mem(op.mem);
  let salvo = op.chatId || '';
  const el = id => dom[id] || (dom[id] = {id, checked: false, disabled: false, hidden: false, value: '', textContent: '', focus(){ ev.push('foco'); }, classList: {_r: false, toggle(c, v){ this._r = !!v; }}});
  const env = {
    el,
    chatId: () => salvo,
    definir: async id => {
      ev.push('definir:' + id);
      if(op.aoDefinir) op.aoDefinir(dom);
      if(op.erro) throw op.erro;
      salvo = id; return id;
    },
    testar: async () => { ev.push('testar'); if(op.erroTeste) throw op.erroTeste; },
    armazenamento: arm, sid: () => 's1',
    toast: (m, e) => toasts.push([m, !!e]),
    aoMudar: () => ev.push('aoMudar')
  };
  const t = criarTelegram(env);
  return {t, ev, dom, toasts, arm, ruim: () => dom.telegramMsg.classList._r};
}
const erroServidor = (codigo, texto) => Object.assign(new Error(texto), {code: codigo});

(async () => {
  // sem a pagina (sem o cartao): nada acontece
  assert.strictEqual(criarTelegram({el: () => null, chatId: () => ''}).ligarCartao(), false);

  // sem Id salvo: chave desligada e so ela a mostra (sem campo, sem teste)
  let m = montar();
  assert.strictEqual(m.t.ligarCartao(), true);
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.tgCampos.hidden, m.dom.btnTesteTelegram.hidden], [false, true, true]);
  assert.strictEqual(m.dom.tgApoio.textContent, 'Vale só para você, neste espaço.');

  // com Id salvo (o servidor mandou): chave ligada, o Id no campo, teste a vista, "Agora nao" escondido
  m = montar({chatId: '123456789'});
  m.t.ligarCartao();
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.tgCampos.hidden, m.dom.btnTesteTelegram.hidden, m.dom.btnTgAgoraNao.hidden, m.dom.telegramChatId.value], [true, false, false, true, '123456789']);
  assert.strictEqual(m.dom.tgApoio.textContent, 'Os avisos deste espaco chegam neste Telegram.'.replace('espaco', 'espaço'));

  // ligar: abre a explicacao e o campo; a chave NAO liga e nada vai ao servidor antes do Salvar
  m = montar();
  m.t.ligarCartao();
  m.dom.swTelegram.checked = true;
  await m.dom.swTelegram.onchange();
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.tgCampos.hidden, m.dom.tgAjuda.hidden, m.dom.btnTgAgoraNao.hidden], [false, false, false, false]);
  assert.strictEqual(m.dom.telegramMsg.textContent, 'Cole o seu Id e toque em Salvar.');
  assert.ok(!m.ev.some(e => e.startsWith('definir')), 'nada vai ao servidor antes do Salvar');

  // Salvar vazio nao desvincula por engano
  await m.dom.btnSalvarTelegram.onclick();
  assert.ok(!m.ev.some(e => e.startsWith('definir')) && m.ruim());
  assert.strictEqual(m.dom.telegramMsg.textContent, 'Cole o Id do Telegram ou desligue a chave.');

  // Agora nao fecha sem ligar
  await m.dom.btnTgAgoraNao.onclick();
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.tgCampos.hidden, m.dom.telegramMsg.textContent], [false, true, '']);

  // Salvar: o servidor recebe o Id como foi digitado (sem o front validar), so depois a chave liga
  m = montar();
  m.t.ligarCartao();
  m.dom.swTelegram.checked = true; await m.dom.swTelegram.onchange();
  m.dom.telegramChatId.value = ' 987654321 ';
  await m.dom.btnSalvarTelegram.onclick();
  assert.ok(m.ev.includes('definir:987654321'));
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.btnTesteTelegram.hidden, m.dom.tgAjuda.hidden, m.dom.btnTgAgoraNao.hidden], [true, false, true, true]);
  assert.strictEqual(m.dom.telegramMsg.textContent, 'Salvo ✓ Use "Enviar aviso de teste" para conferir.');
  assert.ok(m.toasts.some(t => t[0] === '✓ Avisos no Telegram ligados') && m.ev.includes('aoMudar'));

  // o servidor recusa o Id (frase dele) ou nao ha rede: a chave continua desligada, a frase em vermelho
  m = montar({erro: erroServidor(400, 'o chat id do Telegram tem so numeros (ex.: 123456789)')});
  m.t.ligarCartao();
  m.dom.swTelegram.checked = true; await m.dom.swTelegram.onchange();
  m.dom.telegramChatId.value = 'abc';
  await m.dom.btnSalvarTelegram.onclick();
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.ruim(), m.dom.telegramMsg.textContent], [false, true, 'o chat id do Telegram tem so numeros (ex.: 123456789)']);
  assert.ok(!m.ev.includes('aoMudar'));
  m = montar({erro: new TypeError('Failed to fetch')});
  m.t.ligarCartao();
  m.dom.swTelegram.checked = true; await m.dom.swTelegram.onchange();
  m.dom.telegramChatId.value = '12345';
  await m.dom.btnSalvarTelegram.onclick();
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.ruim(), m.dom.telegramMsg.textContent], [false, true, 'Não consegui salvar agora (sem conexão). Tente de novo.']);

  // desligar: espera o servidor (a chave so muda depois), apaga o Id e some o campo
  let durante = null;
  m = montar({chatId: '123456789', aoDefinir: dom => { durante = [dom.swTelegram.checked, dom.swTelegram.disabled]; }});
  m.t.ligarCartao();
  m.dom.swTelegram.checked = false;
  await m.dom.swTelegram.onchange();
  assert.deepStrictEqual(durante, [true, true], 'durante a chamada a chave continua ligada e travada');
  assert.ok(m.ev.includes('definir:'), 'desligar manda o Id vazio');
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.swTelegram.disabled, m.dom.tgCampos.hidden, m.dom.telegramChatId.value], [false, false, true, '']);
  assert.ok(m.toasts.some(t => t[0] === '✓ Telegram desligado (o Id fica só neste aparelho)') && m.ev.includes('aoMudar'));
  assert.deepStrictEqual(JSON.parse(m.arm.d['financas.telegram']), {s1: '123456789'}, 'o Id fica so neste aparelho, por espaco');
  // ligar de novo: o campo vem preenchido com o ultimo Id, a chave NAO liga e nada vai ao servidor antes do Salvar
  m.dom.swTelegram.checked = true; await m.dom.swTelegram.onchange();
  assert.deepStrictEqual([m.dom.telegramChatId.value, m.dom.swTelegram.checked, m.dom.telegramMsg.textContent], ['123456789', false, 'Usei o último Id deste aparelho. Toque em Salvar para ligar.']);
  const antes = m.ev.length;
  await m.dom.btnSalvarTelegram.onclick();
  assert.deepStrictEqual(m.ev.slice(antes, antes + 1), ['definir:123456789']);
  assert.strictEqual(m.dom.swTelegram.checked, true);
  assert.strictEqual(m.arm.d['financas.telegram'], undefined, 'com o Id no servidor, o aparelho nao guarda copia');

  // quem ja tem outro Id lembrado em outro espaco nao o perde ao desligar este
  m = montar({chatId: '111111', mem: {'financas.telegram': '{"s2":"222222"}'}});
  m.t.ligarCartao(); m.dom.swTelegram.checked = false; await m.dom.swTelegram.onchange();
  assert.deepStrictEqual(JSON.parse(m.arm.d['financas.telegram']), {s1: '111111', s2: '222222'});
  // sem Id lembrado, ligar abre o campo vazio com a explicacao de sempre; armazenamento quebrado nao atrapalha
  m = montar({mem: {'financas.telegram': 'lixo{'}});
  m.t.ligarCartao(); m.dom.swTelegram.checked = true; await m.dom.swTelegram.onchange();
  assert.deepStrictEqual([m.dom.telegramChatId.value, m.dom.telegramMsg.textContent], ['', 'Cole o seu Id e toque em Salvar.']);

  // desligar sem rede: continua ligada, com a frase em vermelho (e nada e lembrado)
  m = montar({chatId: '123456789', erro: new TypeError('Failed to fetch')});
  m.t.ligarCartao();
  m.dom.swTelegram.checked = false;
  await m.dom.swTelegram.onchange();
  assert.strictEqual(m.arm.d['financas.telegram'], undefined);
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.swTelegram.disabled, m.dom.tgCampos.hidden, m.ruim(), m.t.ligadoAgora()], [true, false, false, true, true]);
  assert.strictEqual(m.dom.telegramMsg.textContent, 'Não consegui desligar agora (sem conexão). Tente de novo.');
  assert.strictEqual(m.dom.telegramChatId.value, '123456789');

  // o teste
  m = montar({chatId: '123456789'});
  m.t.ligarCartao();
  await m.dom.btnTesteTelegram.onclick();
  assert.ok(m.ev.includes('testar') && m.dom.telegramMsg.textContent === 'Enviado ✓ Confira o Telegram.' && !m.ruim());
  m = montar({chatId: '123456789', erroTeste: erroServidor(400, 'cadastre o seu chat id do Telegram primeiro')});
  m.t.ligarCartao();
  await m.dom.btnTesteTelegram.onclick();
  assert.deepStrictEqual([m.ruim(), m.dom.telegramMsg.textContent], [true, '⚠️ cadastre o seu chat id do Telegram primeiro']);
  m = montar({chatId: '123456789', erroTeste: new TypeError('x')});
  m.t.ligarCartao();
  await m.dom.btnTesteTelegram.onclick();
  assert.strictEqual(m.dom.telegramMsg.textContent, '⚠️ Não consegui enviar o teste (sem conexão).');

  // atualizar: a chave segue o Id que o servidor manda (troca de espaco, nova leitura)
  m = montar();
  m.t.ligarCartao();
  assert.strictEqual(m.dom.swTelegram.checked, false);
  m = montar({chatId: '555555'});
  m.t.ligarCartao();
  assert.strictEqual(m.t.atualizar(), true);

  console.log('telegram.test.js: ok');
})().catch(e => { console.error(e); process.exit(1); });
