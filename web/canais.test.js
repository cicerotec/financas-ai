// Testes da janela "Ligar um canal" e do aviso de "sem canal ativo" (demos demo-avisar-v3-canal, demo-lancar-v9-aviso-canal e demo-cartao-v5-aviso-canal;
// spec telegram-vinculo R18 a R20, V-T23 a V-T25): DOM e API de mentira. Rode com: node web/canais.test.js
const assert = require('assert');
const fs = require('fs');
const path = require('path');
const criarCanais = require('./js/canais.js');

const LIMITE = 10 * 60 * 1000;
const erroServidor = (codigo, texto) => Object.assign(new Error(texto), {code: codigo});

function montar(op){
  op = op || {};
  const ev = [], dom = {}, toasts = [], timers = [], depois = [], voltas = [];
  let canais = Object.assign({telegram: false, push: false}, op.canais), agora = 9000000, mudou = 0;
  const el = id => dom[id] || (dom[id] = {id, hidden: false, disabled: false, href: '', innerHTML: '', textContent: '', onclick: null, focus(){ ev.push('foco:' + id); }, classList: {_r: false, toggle(c, v){ this._r = !!v; }}});
  const env = {
    el,
    canais: () => canais,
    recarregarCanais: async () => { ev.push('reler'); if(op.erroReler) throw op.erroReler; if(op.depoisDeReler) op.depoisDeReler(m); return Object.assign({}, canais); },
    gerarLink: async () => {
      ev.push('gerarLink');
      if(op.erroLink) throw op.erroLink;
      return {url: 'https://t.me/bot_de_teste?start=CODIGO', expiraEm: op.expiraEm !== undefined ? op.expiraEm : new Date(agora + LIMITE).toISOString()};
    },
    abrir: url => { ev.push('abrir:' + url); return op.abre !== false; },
    noApp: () => !!op.noApp,
    ligarAparelho: async () => { ev.push('ligarAparelho'); if(op.erroAparelho) throw op.erroAparelho; return op.resultadoAparelho || {ok: true, aparelhos: 1}; },
    agora: () => agora,
    intervalo: f => { timers.push(f); return timers.length; },
    limpar: id => { timers[id - 1] = null; },
    depois: (f, ms) => depois.push([f, ms]),
    aoVoltar: cb => voltas.push(cb),
    toast: (m, e) => toasts.push([m, !!e]),
    aoMudar: () => { mudou++; ev.push('aoMudar'); }
  };
  const c = criarCanais(env);
  const m = {
    c, ev, dom, toasts,
    mudou: () => mudou,
    ruim: () => dom.lcMsg.classList._r,
    liga: v => { canais = Object.assign({}, canais, v); },
    passa: ms => { agora += ms; },
    tique: () => timers.forEach(f => f && f()),
    voltaAoApp: () => voltas.forEach(f => f()),
    fechaDepois: async () => { for(const [f] of depois.splice(0)) await f(); },
    atrasos: () => depois.map(d => d[1])
  };
  return m;
}
// a dica de mentira: o innerHTML tem o texto, e os botoes sao achados por querySelector
function dicaDe(m, mostrar, cont){
  const botoes = {'[data-lc=ligar]': {}, '[data-lc=seguir]': {}};
  const alvo = {innerHTML: '', querySelector: sel => botoes[sel]};
  m.c.dica(alvo, mostrar, cont);
  return {alvo, botoes};
}
const pronta = op => { const m = montar(op); assert.strictEqual(m.c.ligarJanela(), true); return m; };

(async () => {
  // sem a pagina (sem a janela): nada acontece
  assert.strictEqual(criarCanais({el: () => null}).ligarJanela(), false);

  // abrir: a janela abre por cima, com a escolha; o aparelho so aparece no app Android
  let m = pronta({noApp: true});
  m.c.abrir();
  assert.deepStrictEqual([m.dom.lcOv.hidden, m.c.estaAberta(), m.dom.lcBotoes.hidden, m.dom.lcPush.hidden, m.dom.lcAguarda.hidden, m.c.passoAgora()], [false, true, false, false, true, 'escolha']);
  assert.ok(m.dom.lcMm.textContent.includes('Eles dizem só quantos são'), 'a janela diz que o aviso traz so a contagem');
  assert.strictEqual(m.dom.lcTgS.textContent, 'Numa conversa do Telegram');
  m = pronta({noApp: false});
  m.c.abrir();
  assert.strictEqual(m.dom.lcPush.hidden, true, 'no navegador e na PWA nao ha botao do aparelho');

  // canal ja ligado aparece como Ligado e nao liga de novo
  m = pronta({noApp: true, canais: {telegram: true, push: true}});
  m.c.abrir();
  assert.deepStrictEqual([m.dom.lcTgS.textContent, m.dom.lcPushS.textContent], ['Ligado ✓', 'Ligado ✓']);
  await m.dom.lcTg.onclick(); await m.dom.lcPush.onclick();
  assert.ok(!m.ev.includes('gerarLink') && !m.ev.includes('ligarAparelho'));

  // Telegram: pede o link so no toque, abre a url e fica aguardando com o tempo restante
  m = pronta();
  m.c.abrir();
  assert.ok(!m.ev.includes('gerarLink'), 'abrir a janela nao pede link');
  await m.dom.lcTg.onclick();
  assert.deepStrictEqual(m.ev.filter(e => e.startsWith('gerar') || e.startsWith('abrir')), ['gerarLink', 'abrir:https://t.me/bot_de_teste?start=CODIGO']);
  assert.deepStrictEqual([m.c.passoAgora(), m.dom.lcAguarda.hidden, m.dom.lcBotoes.hidden, m.dom.lcAbrir.hidden, m.dom.lcTempo.textContent],
    ['aguardando', false, true, true, 'O link vale por mais 10:00 (10 minutos no total).']);
  m.passa(125000); m.tique();
  assert.strictEqual(m.dom.lcTempo.textContent, 'O link vale por mais 7:55 (10 minutos no total).');

  // Ja iniciei sem o Iniciar: "Ainda nao chegou", continua aguardando
  await m.dom.lcIniciei.onclick();
  assert.deepStrictEqual([m.dom.lcMsg.textContent, m.c.passoAgora()], ['Ainda não chegou. Toque em Iniciar no Telegram e volte aqui.', 'aguardando']);
  // o Iniciar chegou: Ligado, a tela de baixo e avisada, o toast sai e a janela fecha sozinha um instante depois, relendo o estado
  m.liga({telegram: true});
  await m.dom.lcIniciei.onclick();
  assert.deepStrictEqual([m.dom.lcMsg.textContent, m.c.passoAgora(), m.toasts], ['Ligado ✓', 'escolha', [['✓ Avisos no Telegram ligados', false]]]);
  assert.ok(m.mudou() >= 1);
  assert.deepStrictEqual(m.atrasos(), [900]);
  const antes = m.ev.filter(e => e === 'reler').length;
  await m.fechaDepois();
  assert.deepStrictEqual([m.dom.lcOv.hidden, m.c.estaAberta()], [true, false]);
  assert.strictEqual(m.ev.filter(e => e === 'reler').length, antes + 1, 'fechar rele GET /me');

  // voltar ao primeiro plano enquanto aguarda: uma releitura silenciosa
  m = pronta(); m.c.abrir(); m.voltaAoApp();
  assert.ok(!m.ev.includes('reler'), 'escolhendo, voltar ao app nao le nada');
  await m.dom.lcTg.onclick();
  m.voltaAoApp(); await Promise.resolve(); await Promise.resolve();
  assert.strictEqual(m.ev.filter(e => e === 'reler').length, 1);
  assert.strictEqual(m.dom.lcMsg.textContent, '', 'ainda nao chegou: em silencio');
  m.liga({telegram: true});
  m.voltaAoApp(); await Promise.resolve(); await Promise.resolve();
  assert.strictEqual(m.dom.lcMsg.textContent, 'Ligado ✓');
  // sem sondagem por tempo
  m = pronta(); m.c.abrir(); await m.dom.lcTg.onclick();
  for(let i = 0; i < 5; i++){ m.passa(30000); m.tique(); }
  assert.ok(!m.ev.includes('reler'));

  // o link vence: volta para a escolha com a frase em vermelho
  m = pronta(); m.c.abrir(); await m.dom.lcTg.onclick();
  m.passa(LIMITE); m.tique();
  assert.deepStrictEqual([m.c.passoAgora(), m.dom.lcMsg.textContent, m.ruim(), m.dom.lcBotoes.hidden], ['escolha', 'O link venceu: toque em Telegram de novo.', true, false]);

  // Cancelar volta a escolha sem falar com o servidor
  m = pronta(); m.c.abrir(); await m.dom.lcTg.onclick();
  m.ev.length = 0;
  await m.dom.lcCancelar.onclick();
  assert.deepStrictEqual([m.c.passoAgora(), m.dom.lcMsg.textContent, m.ev], ['escolha', '', []]);

  // erros do link
  const casos = [
    [erroServidor(503, 'x'), 'O Telegram ainda não foi configurado no servidor.'],
    [erroServidor(429, 'x'), 'Muitos links pedidos: tente de novo mais tarde.'],
    [new Error('Failed to fetch'), 'Não consegui pedir o link agora (sem conexão). Tente de novo.'],
    [erroServidor(500, 'erro interno'), 'erro interno']
  ];
  for(const [erro, texto] of casos){
    m = pronta({erroLink: erro}); m.c.abrir(); await m.dom.lcTg.onclick();
    assert.deepStrictEqual([m.dom.lcMsg.textContent, m.ruim(), m.c.passoAgora(), m.dom.lcTg.disabled], [texto, true, 'escolha', false], erro.message);
  }

  // a janela do navegador foi bloqueada: o link fica na propria janela
  m = pronta({abre: false}); m.c.abrir(); await m.dom.lcTg.onclick();
  assert.deepStrictEqual([m.dom.lcAbrir.hidden, m.dom.lcAbrir.href], [false, 'https://t.me/bot_de_teste?start=CODIGO']);

  // o prazo e o do servidor (expiraEm), e relogio que destoa cai nos 10 minutos
  m = pronta({expiraEm: new Date(9000000 + 3 * 60 * 1000).toISOString()}); m.c.abrir(); await m.dom.lcTg.onclick();
  assert.strictEqual(m.dom.lcTempo.textContent, 'O link vale por mais 3:00 (10 minutos no total).');
  m = pronta({expiraEm: 'lixo'}); m.c.abrir(); await m.dom.lcTg.onclick();
  assert.strictEqual(m.dom.lcTempo.textContent, 'O link vale por mais 10:00 (10 minutos no total).');

  // Neste aparelho: a explicacao e do push (o modulo so chama); "Agora nao" nao liga nem diz nada
  m = pronta({noApp: true, resultadoAparelho: {ok: false, cancelou: true}}); m.c.abrir();
  await m.dom.lcPush.onclick();
  assert.deepStrictEqual([m.dom.lcMsg.textContent, m.dom.lcPush.disabled, m.toasts, m.c.estaAberta()], ['', false, [], true]);
  // ligou: rele o estado, avisa a tela de baixo, toast e fecha sozinha
  m = pronta({noApp: true}); m.c.abrir();
  m.liga({push: true});
  await m.dom.lcPush.onclick(); // o servidor ja diz push:true
  assert.ok(!m.ev.includes('ligarAparelho'), 'ja ligado: nao chama');
  m = pronta({noApp: true, depoisDeReler: () => {}}); m.c.abrir();
  await m.dom.lcPush.onclick();
  assert.deepStrictEqual([m.dom.lcMsg.textContent, m.toasts, m.atrasos()], ['Ligado ✓', [['✓ Avisos neste aparelho ligados', false]], [900]]);
  assert.ok(m.ev.includes('reler') && m.mudou() >= 1);
  // recusado (permissao negada, limite de aparelhos, rede): a frase do push, em vermelho, e nada fecha
  m = pronta({noApp: true, resultadoAparelho: {ok: false, texto: 'Sem a permissão do Android o aviso não chega. Nada foi ligado.', config: true}}); m.c.abrir();
  await m.dom.lcPush.onclick();
  assert.deepStrictEqual([m.dom.lcMsg.textContent, m.ruim(), m.dom.lcPush.disabled, m.c.estaAberta()], ['Sem a permissão do Android o aviso não chega. Nada foi ligado.', true, false, true]);
  m = pronta({noApp: true, erroAparelho: new Error('x')}); m.c.abrir();
  await m.dom.lcPush.onclick();
  assert.deepStrictEqual([m.dom.lcMsg.textContent, m.ruim()], ['Não consegui ligar agora (sem conexão). Tente de novo.', true]);

  // fechar (botao, toque fora, voltar): rele GET /me e avisa a tela de baixo; sem rede, nao quebra e a tela segue
  m = pronta(); m.c.abrir();
  await m.dom.lcFechar.onclick();
  assert.deepStrictEqual([m.dom.lcOv.hidden, m.c.estaAberta(), m.ev.includes('reler'), m.mudou()], [true, false, true, 1]);
  m = pronta({erroReler: new Error('sem rede')}); m.c.abrir();
  await m.dom.lcFechar.onclick();
  assert.deepStrictEqual([m.dom.lcOv.hidden, m.mudou()], [true, 1]);
  m = pronta(); m.c.abrir();
  await m.dom.lcOv.onclick({target: m.dom.lcOv});
  assert.strictEqual(m.dom.lcOv.hidden, true, 'o toque fora fecha');
  m = pronta(); m.c.abrir();
  await m.dom.lcOv.onclick({target: m.dom.lcTg});
  assert.strictEqual(m.dom.lcOv.hidden, false, 'o toque dentro da janela nao fecha');
  m = pronta(); await m.c.fechar();
  assert.ok(!m.ev.includes('reler'), 'fechar sem estar aberta nao faz nada');

  // abrir e fechar nao tocam em nada da tela de baixo: este modulo nunca le nem escreve campo de formulario (o que foi digitado fica)
  m = pronta(); m.c.abrir(); await m.dom.lcFechar.onclick();
  assert.deepStrictEqual(Object.keys(m.dom).filter(k => !/^lc/.test(k)), [], 'so os elementos da propria janela');

  // ---------- o aviso de "sem canal ativo" (R19) ----------
  m = pronta();
  let d = dicaDe(m, true);
  assert.ok(d.alvo.innerHTML.includes('Você ainda não tem um canal ativo para receber este aviso') && d.alvo.innerHTML.includes('Ligar um canal') && d.alvo.innerHTML.includes('Continuar assim'));
  // so aparece quando a pessoa marcou o aviso
  assert.strictEqual(dicaDe(m, false).alvo.innerHTML, '');
  // com QUALQUER canal ligado, inclusive so o push, nao aparece
  for(const canais of [{telegram: true}, {push: true}, {telegram: true, push: true}]){
    m = pronta({canais});
    assert.strictEqual(dicaDe(m, true).alvo.innerHTML, '', 'canais ' + JSON.stringify(canais));
    assert.strictEqual(m.c.semCanal(), false);
  }
  // Ligar um canal abre a janela; Continuar assim esconde a dica e avisa quem chamou (para nao voltar ate marcar de novo)
  m = pronta(); let seguiu = 0;
  d = dicaDe(m, true, () => { seguiu++; });
  d.botoes['[data-lc=ligar]'].onclick();
  assert.strictEqual(m.c.estaAberta(), true);
  d.botoes['[data-lc=seguir]'].onclick();
  assert.deepStrictEqual([d.alvo.innerHTML, seguiu], ['', 1]);
  // alvo ausente nao quebra
  m.c.dica(null, true);

  // ---------- rotulos neutros e previa fixa (R19 e R20), no codigo do app ----------
  const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
  const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
  const semComentario = app.split('\n').filter(l => !/^\s*\/\//.test(l)).join('\n').replace(/\s\/\/\s.*$/gm, ''); // sem os comentarios de linha inteira nem os do fim da linha
  assert.ok(!/Telegram/.test(semComentario.replace(/TelegramApp/g, '').replace(/getElementById\('telegramCard'\)/g, '')), 'nenhum texto de tela de aviso diz "Telegram" (so o cartao de Listas e a janela dizem o nome de um canal)');
  assert.ok(!app.includes('telegramVinculado'), 'telegramVinculado() saiu');
  assert.ok(!app.includes('Você ainda não vinculou o Telegram') && !app.includes('cadastrar o chat id'), 'a frase antiga saiu');
  assert.ok(!app.includes('Assim chega no Telegram'), 'a previa antiga saiu');
  const balao = app.slice(app.indexOf('function pvBalao('), app.indexOf('function pvTelaAvisar'));
  assert.ok(balao.includes('A contagem, como “1 vence hoje”. O que é cada um você vê ao abrir o app.'), 'a previa e a frase fixa');
  assert.ok(!/descricao|valorAssinado|fmtMoeda|banco|ref\./.test(balao), 'o front nao monta texto de aviso a partir do lancamento');
  for(const rotulo of ["tcCabeca('Avisar','voltar')", "tcLinha('avi','Avisar',", '🔔 Avisar</label>', ">Avisar</button>"]) assert.ok(app.includes(rotulo), 'falta o rotulo ' + rotulo);
  // a janela existe no HTML com tudo que o modulo usa, e entra nas camadas do botao voltar
  for(const id of ['lcOv', 'lcMm', 'lcBotoes', 'lcTg', 'lcTgS', 'lcPush', 'lcPushS', 'lcAguarda', 'lcTempo', 'lcIniciei', 'lcCancelar', 'lcAbrir', 'lcMsg', 'lcFechar']) assert.ok(html.includes('id="' + id + '"'), id + ' nao esta no index.html');
  const corpo = nome => { const i = app.indexOf('function ' + nome + '('); assert.ok(i >= 0, nome); return app.slice(i, app.indexOf('\n}', i) + 2); };
  assert.ok(corpo('camadasAbertas').includes("camadaVisivel('lcOv')"), 'a janela conta como camada');
  const fechar = corpo('fecharCamadaTopo');
  assert.ok(fechar.indexOf("camadaVisivel('lcOv')") > fechar.indexOf("camadaVisivel('modalOverlay')") && fechar.indexOf("clicar('#lcFechar')") > fechar.indexOf("camadaVisivel('lcOv')"), 'o voltar fecha a janela pelo botao dela, logo abaixo do modal de confirmacao');
  assert.ok(fechar.indexOf("camadaVisivel('lcOv')") < fechar.indexOf('if(pv)'), 'a janela fecha antes da tela cheia que ela cobre');

  console.log('canais.test.js ok');
})().catch(e => { console.error(e); process.exit(1); });
