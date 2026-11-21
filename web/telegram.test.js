// Testes do cartao Avisos no Telegram por link (demo-listas-v11-telegram-vinculo, spec telegram-vinculo R17): DOM e API de mentira.
// Rode com: node web/telegram.test.js
const assert = require('assert');
const criarTelegram = require('./js/telegram.js');

const LIMITE = 10 * 60 * 1000;
function mem(init){ const d = Object.assign({}, init); return {d, gravou: [], getItem: k => (k in d ? d[k] : null), setItem(k, v){ d[k] = String(v); this.gravou.push(k); }, removeItem: k => { delete d[k]; }}; }

function montar(op){
  op = op || {};
  const ev = [], dom = {}, toasts = [], arm = mem(op.mem), timers = [], voltas = [];
  let canais = {telegram: !!op.ligado, push: false}, agora = 5000000;
  const el = id => dom[id] || (dom[id] = {id, checked: false, disabled: false, hidden: false, href: '', textContent: '', classList: {_r: false, toggle(c, v){ this._r = !!v; }}});
  const env = {
    el,
    canais: () => canais,
    gerarLink: async () => {
      ev.push('gerarLink');
      if(op.antesDoLink) op.antesDoLink(dom);
      if(op.erroLink) throw op.erroLink;
      return {url: 'https://t.me/bot_de_teste?start=CODIGO', expiraEm: op.expiraEm !== undefined ? op.expiraEm : new Date(agora + LIMITE).toISOString()};
    },
    recarregarCanais: async () => { ev.push('reler'); if(op.erroReler) throw op.erroReler; return Object.assign({}, canais); },
    desligar: async () => { ev.push('desligar'); if(op.erroDesligar) throw op.erroDesligar; canais = Object.assign({}, canais, {telegram: false}); },
    testar: async () => { ev.push('testar'); if(op.erroTeste) throw op.erroTeste; },
    abrir: url => { ev.push('abrir:' + url); return op.abre !== false; },
    armazenamento: arm,
    agora: () => agora,
    intervalo: f => { timers.push(f); ev.push('intervalo'); return timers.length; },
    limpar: id => { timers[id - 1] = null; ev.push('limpar'); },
    aoVoltar: cb => voltas.push(cb),
    toast: (m, e) => toasts.push([m, !!e]),
    aoMudar: () => ev.push('aoMudar')
  };
  const t = criarTelegram(env);
  return {
    t, ev, dom, toasts, arm,
    ruim: () => dom.telegramMsg.classList._r,
    servidorLigou: () => { canais = Object.assign({}, canais, {telegram: true}); },
    passa: ms => { agora += ms; },
    tique: () => timers.forEach(f => f && f()),
    voltaAoApp: () => voltas.forEach(f => f()),
    liga: async () => { dom.swTelegram.checked = true; await dom.swTelegram.onchange(); },
    desliga: async () => { dom.swTelegram.checked = false; await dom.swTelegram.onchange(); }
  };
}
const erroServidor = (codigo, texto) => Object.assign(new Error(texto), {code: codigo});
const sem = (m, ...ids) => ids.every(i => m.dom[i] === undefined);

(async () => {
  // sem a pagina (sem o cartao): nada acontece
  assert.strictEqual(criarTelegram({el: () => null, canais: () => ({})}).ligarCartao(), false);

  // desligado: so a chave; sem campo de Id, sem Salvar, sem "Agora nao"
  let m = montar();
  assert.strictEqual(m.t.ligarCartao(), true);
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.tgAguarda.hidden, m.dom.tgAcoes.hidden, m.dom.btnTesteTelegram.hidden], [false, true, true, true]);
  assert.strictEqual(m.dom.tgApoio.textContent, 'Vale só para você, neste espaço.');
  assert.ok(sem(m, 'telegramChatId', 'tgCampos', 'btnSalvarTelegram', 'btnTgAgoraNao', 'tgAjuda'), 'o cartao nao tem mais o campo do Id nem o Salvar');

  // ligado (o servidor mandou canais.telegram): chave ligada e o teste a vista
  m = montar({ligado: true});
  m.t.ligarCartao();
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.tgAguarda.hidden, m.dom.tgAcoes.hidden, m.dom.btnTesteTelegram.hidden, m.dom.btnTgLigar.hidden], [true, true, false, false, true]);
  assert.strictEqual(m.dom.tgApoio.textContent, 'Os avisos deste espaço chegam no seu Telegram.');

  // ligar: pede o link (so no toque), abre a url e fica aguardando com o tempo restante; a chave fica marcada enquanto o link e pedido
  m = montar({antesDoLink: dom => { assert.deepStrictEqual([dom.swTelegram.checked, dom.swTelegram.disabled, dom.telegramMsg.textContent], [true, true, 'Pedindo o link…']); }});
  m.t.ligarCartao();
  assert.ok(!m.ev.includes('gerarLink'), 'nenhum link e pedido antes do toque');
  await m.liga();
  assert.deepStrictEqual(m.ev.filter(e => e !== 'intervalo'), ['gerarLink', 'abrir:https://t.me/bot_de_teste?start=CODIGO']);
  assert.deepStrictEqual([m.dom.swTelegram.checked, m.dom.swTelegram.disabled, m.dom.tgAguarda.hidden, m.dom.tgAcoes.hidden, m.t.estadoAgora()], [true, false, false, true, 'aguardando']);
  assert.strictEqual(m.dom.tgApoio.textContent, 'Falta tocar em Iniciar no Telegram.');
  assert.strictEqual(m.dom.tgTempo.textContent, 'O link vale por mais 10:00 (10 minutos no total).');
  assert.strictEqual(m.dom.tgAbrir.hidden, true, 'a janela abriu: nao precisa do link na tela');
  m.passa(2 * 60 * 1000 + 3000); m.tique();
  assert.strictEqual(m.dom.tgTempo.textContent, 'O link vale por mais 7:57 (10 minutos no total).');

  // Ja iniciei com o Iniciar ainda nao chegado: "Ainda nao chegou" e CONTINUA aguardando, com o tempo restante
  await m.dom.btnTgIniciei.onclick();
  assert.deepStrictEqual([m.dom.telegramMsg.textContent, m.t.estadoAgora(), m.dom.tgAguarda.hidden], ['Ainda não chegou. Toque em Iniciar no Telegram e volte aqui.', 'aguardando', false]);
  assert.strictEqual(m.ev.filter(e => e === 'reler').length, 1);

  // o Iniciar chegou: Ja iniciei le o servidor e a chave vira "Ligado"
  m.servidorLigou();
  await m.dom.btnTgIniciei.onclick();
  assert.strictEqual(m.t.estadoAgora(), 'ligado');
  assert.deepStrictEqual([m.dom.tgAguarda.hidden, m.dom.btnTesteTelegram.hidden, m.dom.tgApoio.textContent], [true, false, 'Os avisos deste espaço chegam no seu Telegram.']);
  assert.deepStrictEqual(m.toasts, [['✓ Avisos no Telegram ligados', false]]);
  assert.ok(m.ev.includes('aoMudar') && m.ev.includes('limpar'), 'a tela de baixo e avisada e o relogio para');

  // sem conexao ao conferir: a frase em vermelho e continua aguardando
  m = montar({erroReler: new Error('x')});
  m.t.ligarCartao(); await m.liga();
  await m.dom.btnTgIniciei.onclick();
  assert.deepStrictEqual([m.dom.telegramMsg.textContent, m.ruim(), m.t.estadoAgora()], ['Não consegui conferir agora (sem conexão). Tente de novo.', true, 'aguardando']);

  // voltar ao primeiro plano enquanto aguarda: UMA releitura, e em silencio quando ainda nao chegou
  m = montar();
  m.t.ligarCartao();
  m.voltaAoApp();
  assert.ok(!m.ev.includes('reler'), 'desligado: voltar ao app nao le nada');
  await m.liga();
  m.voltaAoApp(); await Promise.resolve(); await Promise.resolve();
  assert.strictEqual(m.ev.filter(e => e === 'reler').length, 1);
  assert.strictEqual(m.dom.telegramMsg.textContent, '', 'ainda nao chegou: a releitura silenciosa nao diz nada');
  m.servidorLigou();
  m.voltaAoApp(); await Promise.resolve(); await Promise.resolve();
  assert.strictEqual(m.t.estadoAgora(), 'ligado');
  m.voltaAoApp(); await Promise.resolve();
  assert.strictEqual(m.ev.filter(e => e === 'reler').length, 2, 'ligado: nao rele mais ao voltar');
  // sem sondagem por tempo: o relogio andar nunca le o servidor
  m = montar(); m.t.ligarCartao(); await m.liga();
  for(let i = 0; i < 5; i++){ m.passa(30000); m.tique(); }
  assert.ok(!m.ev.includes('reler'), 'o tempo passar nao rele o servidor');

  // o link vence (10 minutos): "O link venceu", chave desligada e o botao Ligar Telegram
  m = montar();
  m.t.ligarCartao(); await m.liga();
  m.passa(LIMITE); m.tique();
  assert.deepStrictEqual([m.t.estadoAgora(), m.dom.swTelegram.checked, m.dom.tgAguarda.hidden, m.dom.btnTgLigar.hidden, m.dom.btnTgLigar.textContent, m.dom.tgAcoes.hidden],
    ['desligado', false, true, false, 'Ligar Telegram', false]);
  assert.deepStrictEqual([m.dom.telegramMsg.textContent, m.ruim()], ['O link venceu: toque em Ligar Telegram de novo.', true]);
  m.ev.length = 0;
  await m.dom.btnTgLigar.onclick(); // ligar de novo
  assert.deepStrictEqual([m.ev[0], m.t.estadoAgora(), m.dom.btnTgLigar.hidden], ['gerarLink', 'aguardando', true]);

  // Cancelar sai do aguardando sem falar com o servidor (o link ja gerado vale ate vencer)
  m = montar(); m.t.ligarCartao(); await m.liga();
  m.ev.length = 0;
  await m.dom.btnTgCancelar.onclick();
  assert.deepStrictEqual([m.t.estadoAgora(), m.dom.swTelegram.checked, m.dom.tgAguarda.hidden, m.dom.telegramMsg.textContent], ['desligado', false, true, '']);
  assert.ok(!m.ev.some(e => e === 'gerarLink' || e === 'desligar' || e === 'reler'));
  // tocar na chave ligada durante o aguardando tambem cancela
  m = montar(); m.t.ligarCartao(); await m.liga(); await m.desliga();
  assert.strictEqual(m.t.estadoAgora(), 'desligado');

  // erros do link: 503 (nao configurado), 429 (limite), rede e a frase do servidor para outro codigo; o botao vira "Tentar de novo"
  const casos = [
    [erroServidor(503, 'o Telegram ainda nao foi configurado no servidor'), 'O Telegram ainda não foi configurado no servidor.'],
    [erroServidor(429, 'muitos links pedidos'), 'Muitos links pedidos: tente de novo mais tarde.'],
    [new Error('Failed to fetch'), 'Não consegui pedir o link agora (sem conexão). Tente de novo.'],
    [erroServidor(500, 'erro interno'), 'erro interno']
  ];
  for(const [erro, texto] of casos){
    m = montar({erroLink: erro});
    m.t.ligarCartao(); await m.liga();
    assert.deepStrictEqual([m.dom.telegramMsg.textContent, m.ruim(), m.t.estadoAgora(), m.dom.swTelegram.checked, m.dom.swTelegram.disabled, m.dom.btnTgLigar.textContent, m.dom.btnTgLigar.hidden],
      [texto, true, 'desligado', false, false, 'Tentar de novo', false], 'erro ' + erro.message);
    assert.ok(!m.ev.some(e => e.startsWith('abrir')), 'sem link nada abre');
  }

  // a janela foi bloqueada pelo navegador: a propria tela oferece o link para tocar
  m = montar({abre: false});
  m.t.ligarCartao(); await m.liga();
  assert.deepStrictEqual([m.dom.tgAbrir.hidden, m.dom.tgAbrir.href, m.t.estadoAgora()], [false, 'https://t.me/bot_de_teste?start=CODIGO', 'aguardando']);
  m.servidorLigou(); await m.dom.btnTgIniciei.onclick();
  assert.strictEqual(m.dom.tgAbrir.hidden, true);

  // o prazo e o do servidor (expiraEm); relogio do aparelho que destoa cai nos 10 minutos a partir de agora
  m = montar({expiraEm: new Date(5000000 + 3 * 60 * 1000).toISOString()});
  m.t.ligarCartao(); await m.liga();
  assert.strictEqual(m.dom.tgTempo.textContent, 'O link vale por mais 3:00 (10 minutos no total).');
  for(const ruim of [new Date(5000000 - 60000).toISOString(), 'lixo', undefined, null, new Date(5000000 + 5 * 3600 * 1000).toISOString()]){
    m = montar({expiraEm: ruim});
    m.t.ligarCartao(); await m.liga();
    assert.strictEqual(m.dom.tgTempo.textContent, 'O link vale por mais 10:00 (10 minutos no total).', 'expiraEm ' + ruim);
  }

  // recarregar o app no meio: o estado vem do servidor (atualizar), o aguardando nao sobrevive
  m = montar(); m.t.ligarCartao(); await m.liga();
  m.servidorLigou();
  assert.strictEqual(m.t.atualizar(), true);
  assert.deepStrictEqual([m.t.estadoAgora(), m.dom.tgAguarda.hidden, m.dom.swTelegram.checked], ['ligado', true, true]);
  m = montar(); m.t.ligarCartao(); await m.liga();
  assert.strictEqual(m.t.atualizar(), false);
  assert.deepStrictEqual([m.t.estadoAgora(), m.dom.tgAguarda.hidden, m.dom.swTelegram.checked], ['desligado', true, false]);

  // desligar: o servidor apaga o destino; so depois a chave desliga; nenhum chat id em lugar nenhum
  m = montar({ligado: true}); m.t.ligarCartao();
  await m.desliga();
  assert.deepStrictEqual([m.ev.includes('desligar'), m.t.estadoAgora(), m.dom.swTelegram.checked, m.dom.btnTesteTelegram.hidden, m.dom.tgAcoes.hidden], [true, 'desligado', false, true, true]);
  assert.deepStrictEqual(m.toasts, [['✓ Telegram desligado', false]]);
  assert.ok(m.ev.includes('aoMudar'));
  // sem conexao: continua ligada, frase em vermelho
  m = montar({ligado: true, erroDesligar: new Error('x')}); m.t.ligarCartao();
  await m.desliga();
  assert.deepStrictEqual([m.t.estadoAgora(), m.dom.swTelegram.checked, m.dom.telegramMsg.textContent, m.ruim()], ['ligado', true, 'Não consegui desligar agora (sem conexão). Tente de novo.', true]);
  m = montar({ligado: true, erroDesligar: erroServidor(500, 'erro interno')}); m.t.ligarCartao();
  await m.desliga();
  assert.strictEqual(m.dom.telegramMsg.textContent, 'erro interno');

  // aviso de teste
  m = montar({ligado: true}); m.t.ligarCartao();
  await m.dom.btnTesteTelegram.onclick();
  assert.deepStrictEqual([m.dom.telegramMsg.textContent, m.ruim(), m.ev.includes('testar')], ['Enviado ✓ Confira o Telegram.', false, true]);
  m = montar({ligado: true, erroTeste: new Error('x')}); m.t.ligarCartao();
  await m.dom.btnTesteTelegram.onclick();
  assert.deepStrictEqual([m.dom.telegramMsg.textContent, m.ruim()], ['⚠️ Não consegui enviar o teste (sem conexão).', true]);
  m = montar({ligado: true, erroTeste: erroServidor(400, 'ligue o Telegram em Listas, Avisos, primeiro')}); m.t.ligarCartao();
  await m.dom.btnTesteTelegram.onclick();
  assert.strictEqual(m.dom.telegramMsg.textContent, '⚠️ ligue o Telegram em Listas, Avisos, primeiro');

  // nenhum chat id e lido, mostrado, guardado ou lembrado: o Id lembrado de antes e APAGADO ao subir e nada novo e gravado em nenhum fluxo
  m = montar({mem: {'financas.telegram': '{"s1":"123456789"}', 'outra': 'fica'}});
  m.t.ligarCartao();
  assert.strictEqual(m.arm.d['financas.telegram'], undefined, 'o Id lembrado de antes some ao subir');
  assert.strictEqual(m.arm.d.outra, 'fica');
  await m.liga(); m.servidorLigou(); await m.dom.btnTgIniciei.onclick(); await m.desliga();
  assert.deepStrictEqual(m.arm.gravou, [], 'nenhum fluxo grava nada no aparelho');
  assert.ok(Object.values(m.dom).every(d => !('value' in d) || d.value === undefined), 'nenhum campo de Id');
  m = montar({mem: {'financas.telegram': 'lixo{'}}); m.t.ligarCartao();
  assert.strictEqual(m.arm.d['financas.telegram'], undefined);

  console.log('telegram.test.js ok');
})().catch(e => { console.error(e); process.exit(1); });
