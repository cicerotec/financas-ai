// Testes da seção "Segurança da conta" (web/js/seguranca-conta.js). Spec: .ai/specs/mfa-opcional/spec.md (R7 a R24, R37). Demo: .ai/demos/demo-listas-v10-mfa.html.
// Roda em Node, com elementos, sessão, Cognito e servidor de mentira. Rode com: node web/seguranca-conta.test.js
const assert = require('assert');
const criarSegurancaConta = require('./js/seguranca-conta.js');

const tique = () => new Promise((r) => setTimeout(r, 0));
const esperar = async () => { for (let i = 0; i < 12; i++) await tique(); };
const CODIGOS = ['AAAA-BBBB-CCCC-DDDD-EEEE', 'FFFF-GGGG-HHHH-JJJJ-KKKK'];
const CHAVE = 'JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP';

function elemento(id, extra) {
  const e = Object.assign({ id, hidden: false, textContent: '', checked: false, disabled: false, value: '', children: [], attrs: {}, classes: new Set(), onclick: null, onchange: null, onkeydown: null, focado: 0 }, extra || {});
  let html = '';
  Object.defineProperty(e, 'innerHTML', { get: () => html, set: (v) => { html = v; if (v === '') e.children = []; } });
  e.classList = { toggle: (c, sim) => { if (sim) e.classes.add(c); else e.classes.delete(c); } };
  e.setAttribute = (k, v) => { e.attrs[k] = String(v); };
  e.getAttribute = (k) => (k in e.attrs ? e.attrs[k] : null);
  e.appendChild = (f) => { e.children.push(f); return f; };
  e.focus = () => { e.focado++; };
  return e;
}

function pagina(op) {
  op = op || {};
  const d = { chamadas: [], toasts: [], avisos: [], janelas: [], copiados: [], impressos: [], armazem: new Map(), tokenMfa: op.semToken ? null : 'AT', irListas: 0 };
  const els = {};
  const ids = ['mfaCard', 'swMfa', 'mfaApoio', 'mfaMsg', 'mfaListaBloco', 'mfaListaTxt', 'mfaAcoes', 'btnMfaNova', 'mfaDeNovo', 'mfaNota',
    'mfLig', 'mfLigVolta', 'mfQr', 'mfChave', 'mfCodigo', 'mfLigMsg', 'mfVerificar',
    'mfLista', 'mfListaVolta', 'mfListaCtx', 'mfCodigos', 'mfCopiar', 'mfImprimir', 'mfEntendi', 'mfListaMsg', 'mfGuardei'];
  ids.forEach((id) => { els[id] = elemento(id, { hidden: ['mfLig', 'mfLista', 'mfaListaBloco', 'mfaAcoes', 'mfaDeNovo'].includes(id) }); });
  els.mfEntendi.attrs['aria-checked'] = 'false';
  els.mfGuardei.textContent = 'Guardei a lista';
  d.els = els;
  d.servidor = { estado: op.estado || { ligado: false, lista: { ativa: false } }, falharEstado: null, falharGerar: null, falharConfirmar: null, n: 0 };
  d.cognito = { falharAssociar: null, falharVerificar: null, falharLigar: null, falharDesligar: null };
  const reg = (x) => d.chamadas.push(x);
  const mfa = {
    associar: async (t) => { reg('associar:' + t); if (d.cognito.falharAssociar) throw d.cognito.falharAssociar; return { chave: CHAVE }; },
    verificar: async (t, c) => { reg('verificar:' + t + ':' + c); if (d.cognito.falharVerificar) throw d.cognito.falharVerificar; return true; },
    ligar: async (t) => { reg('ligar:' + t); if (d.cognito.falharLigar) throw d.cognito.falharLigar; d.servidor.estado = { ligado: true, lista: { ativa: true, geradaEm: '2026-11-19T10:00:00Z' } }; return true; },
    desligar: async (t) => { reg('desligar:' + t); if (d.cognito.falharDesligar) throw d.cognito.falharDesligar; d.servidor.estado = { ligado: false, lista: { ativa: false } }; return true; },
    uriOtpauth: (k, n) => 'otpauth://totp/' + n + '?secret=' + k,
    formatarChave: (k) => k.replace(/(.{4})/g, '$1 ').trim()
  };
  d.sessao = {
    modo: op.navegador ? 'navegador' : 'app',
    iniciarMfa: async () => { reg('iniciarMfa'); if (op.loginFalha) throw Object.assign(new Error('x'), { codigo: op.loginFalha }); if (op.navegador) return new Promise(() => {}); d.tokenMfa = 'AT'; return 'AT'; },
    tokenMfa: () => d.tokenMfa,
    descartarMfa: () => { reg('descartarMfa'); d.tokenMfa = null; },
    aoMfa: (f) => { d.retorno = f; }
  };
  d.respostasModal = op.respostas || [];
  d.env = {
    el: (id) => els[id] || null,
    documento: { createElement: () => elemento('li') },
    sessao: d.sessao,
    armazenamento: { getItem: (k) => (d.armazem.has(k) ? d.armazem.get(k) : null), setItem: (k, v) => { d.armazem.set(k, String(v)); }, removeItem: (k) => { d.armazem.delete(k); } },
    agora: () => 1000,
    agoraISO: () => '2026-11-19T10:00:00Z',
    nomeDoApp: () => 'Finanças',
    enderecoRecuperar: () => '',
    api: {
      estado: async () => { reg('GET estado'); if (d.servidor.falharEstado) throw d.servidor.falharEstado; return d.servidor.estado; },
      gerar: async () => { reg('POST gerar'); if (d.servidor.falharGerar) throw d.servidor.falharGerar; d.servidor.n++; return { listaId: 'L' + d.servidor.n, codigos: CODIGOS.slice() }; },
      confirmar: async (id) => { reg('POST confirmar:' + id); if (d.servidor.falharConfirmar) throw d.servidor.falharConfirmar; return true; }
    },
    mfa: async () => mfa,
    qr: async () => ({ svg: (t) => '<svg data-texto="' + t + '"></svg>' }),
    confirmar: async (msg, ok, cancelar, titulo) => { d.janelas.push({ msg, ok, cancelar, titulo }); const r = d.respostasModal.length ? d.respostasModal.shift() : true; return r; },
    toast: (m, erro) => { d.toasts.push({ m, erro: !!erro }); },
    aviso: (a, b) => { d.avisos.push([a, b]); },
    copiar: async (t) => { d.copiados.push(t); },
    imprimir: (t) => { d.impressos.push(t); },
    irParaListas: () => { d.irListas++; }
  };
  d.s = criarSegurancaConta(d.env);
  d.s.ligar();
  return d;
}

const marcarCheck = (d) => { d.els.mfEntendi.onclick(); };
let falhas = 0;
const fila = [];
const teste = (nome, fn) => fila.push([nome, fn]);

teste('ao abrir Listas, o estado vem do servidor e so isso e exibido (ligado e lista ativa com a data)', async () => {
  const d = pagina({ estado: { ligado: true, lista: { ativa: true, geradaEm: '2026-11-05T10:00:00Z' } } });
  assert.deepStrictEqual(d.chamadas, [], 'a subida e o ligar nao leem nada');
  await d.s.entrouEmListas();
  assert.deepStrictEqual(d.chamadas, ['GET estado']);
  assert.strictEqual(d.els.swMfa.checked, true);
  assert.strictEqual(d.els.mfaListaBloco.hidden, false);
  assert.ok(/05\/11\/2026/.test(d.els.mfaListaTxt.textContent), d.els.mfaListaTxt.textContent);
  assert.strictEqual(d.els.mfaAcoes.hidden, false);
});

teste('desligado: sem bloco da lista e sem "Gerar nova lista"; a frase de apoio e a de desligado', async () => {
  const d = pagina();
  await d.s.entrouEmListas();
  assert.strictEqual(d.els.swMfa.checked, false);
  assert.strictEqual(d.els.mfaListaBloco.hidden, true);
  assert.strictEqual(d.els.mfaAcoes.hidden, true);
  assert.ok(/só a senha protege/.test(d.els.mfaNota.textContent));
});

teste('falha ao ler o estado: nada inventado, a chave fica desligada e travada, e ha "Tentar de novo"', async () => {
  const d = pagina();
  d.servidor.falharEstado = new Error('rede');
  await d.s.entrouEmListas();
  assert.strictEqual(d.els.swMfa.disabled, true);
  assert.strictEqual(d.els.mfaDeNovo.hidden, false);
  assert.ok(/Não consegui/.test(d.els.mfaMsg.textContent));
  assert.strictEqual(d.s.estadoAtual(), null);
  d.servidor.falharEstado = null;
  await d.els.mfaDeNovo.onclick();
  assert.strictEqual(d.els.mfaDeNovo.hidden, true);
  assert.strictEqual(d.els.swMfa.disabled, false);
});

teste('ligar, no app: confirma, segundo login, QR e chave, codigo, lista, "Guardei a lista" e SO ENTAO liga (ordem do R16)', async () => {
  const d = pagina();
  await d.s.entrouEmListas();
  await d.s.trocar(true); await esperar();
  assert.strictEqual(d.janelas[0].titulo, 'Confirmar que é você');
  assert.deepStrictEqual(d.chamadas.slice(1, 3), ['iniciarMfa', 'associar:AT']);
  assert.strictEqual(d.els.mfLig.hidden, false);
  assert.ok(d.els.mfQr.innerHTML.includes('otpauth://totp/Finanças?secret=' + CHAVE), 'o QR leva so o nome do app e a chave');
  assert.ok(!/@/.test(d.els.mfQr.innerHTML), 'sem e-mail no QR');
  assert.ok(/JBSW Y3DP/.test(d.els.mfChave.textContent));
  d.els.mfCodigo.value = '123 456';
  await d.s.verificar(); await esperar();
  assert.deepStrictEqual(d.chamadas.slice(3), ['verificar:AT:123 456', 'POST gerar']);
  assert.strictEqual(d.els.mfLig.hidden, true, 'a tela do QR fecha');
  assert.strictEqual(d.els.mfQr.innerHTML, '', 'o QR some (R19)');
  assert.strictEqual(d.els.mfChave.textContent, '', 'a chave some (R19)');
  assert.strictEqual(d.els.mfLista.hidden, false);
  assert.deepStrictEqual(d.els.mfCodigos.children.map((l) => l.textContent), CODIGOS);
  assert.ok(!d.chamadas.some((c) => c.startsWith('ligar')), 'ainda nao ligou');
  marcarCheck(d);
  await d.s.guardei(); await esperar();
  assert.deepStrictEqual(d.chamadas.slice(5), ['POST confirmar:L1', 'ligar:AT', 'descartarMfa', 'GET estado']);
  assert.strictEqual(d.els.mfLista.hidden, true);
  assert.strictEqual(d.els.mfCodigos.children.length, 0, 'a lista some (R20)');
  assert.strictEqual(d.s.listaAberta(), false);
  assert.strictEqual(d.els.swMfa.checked, true);
  assert.strictEqual(d.avisos.length, 1);
  assert.strictEqual(d.armazem.size, 0, 'nada fica guardado');
});

teste('codigo errado: nao gera lista, mostra o erro na propria tela e deixa tentar de novo', async () => {
  const d = pagina();
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  d.cognito.falharVerificar = Object.assign(new Error('x'), { codigo: 'codigo_errado' });
  d.els.mfCodigo.value = '000000';
  await d.s.verificar();
  assert.ok(/Código incorreto/.test(d.els.mfLigMsg.textContent));
  assert.ok(d.els.mfLigMsg.classes.has('ruim'));
  assert.ok(!d.chamadas.includes('POST gerar'));
  assert.strictEqual(d.els.mfLig.hidden, false);
  d.cognito.falharVerificar = null; d.els.mfCodigo.value = '123456';
  await d.s.verificar(); await esperar();
  assert.ok(d.chamadas.includes('POST gerar'));
});

teste('menos de 6 digitos: nem chama o Cognito', async () => {
  const d = pagina();
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  d.els.mfCodigo.value = '12345';
  await d.s.verificar();
  assert.ok(!d.chamadas.some((c) => c.startsWith('verificar')));
  assert.ok(d.els.mfLigMsg.classes.has('ruim'));
});

teste('sem marcar "Guardei a lista" nao confirma nem liga', async () => {
  const d = pagina();
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  d.els.mfCodigo.value = '123456'; await d.s.verificar(); await esperar();
  await d.s.guardei();
  assert.ok(!d.chamadas.some((c) => c.startsWith('POST confirmar') || c.startsWith('ligar')));
  assert.ok(d.els.mfListaMsg.classes.has('ruim'));
});

teste('409 lista_substituida: NAO liga, o botao vira "Gerar nova lista" e a antiga continua valendo', async () => {
  const d = pagina();
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  d.els.mfCodigo.value = '123456'; await d.s.verificar(); await esperar();
  marcarCheck(d);
  d.servidor.falharConfirmar = Object.assign(new Error('lista_substituida'), { code: 409 });
  await d.s.guardei();
  assert.ok(!d.chamadas.some((c) => c.startsWith('ligar')), 'nunca liga sem a confirmacao 200');
  assert.strictEqual(d.els.mfGuardei.textContent, 'Gerar nova lista');
  d.servidor.falharConfirmar = null;
  await d.s.guardei(); await esperar(); // gera outra
  assert.strictEqual(d.els.mfGuardei.textContent, 'Guardei a lista');
  assert.ok(d.chamadas.includes('POST confirmar:L1') && d.chamadas.filter((c) => c === 'POST gerar').length === 2);
});

teste('confirmar sem rede: a lista continua na tela e nada e ligado', async () => {
  const d = pagina();
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  d.els.mfCodigo.value = '123456'; await d.s.verificar(); await esperar();
  marcarCheck(d);
  d.servidor.falharConfirmar = new TypeError('Failed to fetch');
  await d.s.guardei();
  assert.strictEqual(d.els.mfLista.hidden, false);
  assert.ok(!d.chamadas.some((c) => c.startsWith('ligar')));
  assert.ok(/sem conexão/.test(d.els.mfListaMsg.textContent));
});

teste('falha ao ligar depois de confirmar: a lista continua e o botao tenta de novo', async () => {
  const d = pagina();
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  d.els.mfCodigo.value = '123456'; await d.s.verificar(); await esperar();
  marcarCheck(d);
  d.cognito.falharLigar = Object.assign(new Error('x'), { codigo: 'rede' });
  await d.s.guardei();
  assert.strictEqual(d.els.mfLista.hidden, false);
  assert.ok(/não consegui ligar/.test(d.els.mfListaMsg.textContent));
  d.cognito.falharLigar = null;
  await d.s.guardei(); await esperar();
  assert.strictEqual(d.els.mfLista.hidden, true);
  assert.strictEqual(d.s.estadoAtual().ligado, true);
});

teste('sair da lista pergunta antes, descarta tudo e nao liga; "Ficar" mantem a lista', async () => {
  const d = pagina({ respostas: [true, false, true] }); // ligar: sim; sair: "Ficar"; sair de novo: "Sair sem guardar"
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  d.els.mfCodigo.value = '123456'; await d.s.verificar(); await esperar();
  await d.s.sairDaLista();
  assert.strictEqual(d.els.mfLista.hidden, false, 'Ficar mantem');
  await d.s.sairDaLista();
  assert.strictEqual(d.els.mfLista.hidden, true);
  assert.strictEqual(d.els.mfCodigos.children.length, 0);
  assert.ok(d.chamadas.includes('descartarMfa'));
  assert.ok(!d.chamadas.some((c) => c.startsWith('ligar') || c.startsWith('POST confirmar')));
});

teste('sair da tela do QR descarta o acesso e some com QR e chave', async () => {
  const d = pagina();
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  d.s.sairDoLigar();
  assert.strictEqual(d.els.mfLig.hidden, true);
  assert.strictEqual(d.els.mfQr.innerHTML, '');
  assert.strictEqual(d.els.mfChave.textContent, '');
  assert.ok(d.chamadas.includes('descartarMfa'));
});

teste('cancelar a confirmacao ou o segundo login nao altera nada', async () => {
  const d = pagina({ respostas: [false] });
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  assert.ok(!d.chamadas.includes('iniciarMfa'));
  const e = pagina({ loginFalha: 'cancelado' });
  await e.s.entrouEmListas(); await e.s.trocar(true); await esperar();
  assert.strictEqual(e.els.mfLig.hidden, true);
  assert.ok(/Nada foi alterado/.test(e.toasts[0].m));
  assert.strictEqual(e.els.swMfa.checked, false);
});

teste('desligar: confirma, segundo login, desliga, relê o estado e avisa; a chave so muda depois do servidor', async () => {
  const d = pagina({ estado: { ligado: true, lista: { ativa: true, geradaEm: '2026-11-05T10:00:00Z' } } });
  await d.s.entrouEmListas();
  await d.s.trocar(false); await esperar();
  assert.ok(/Desligar a verificação/.test(d.janelas[0].titulo));
  assert.deepStrictEqual(d.chamadas.slice(1), ['iniciarMfa', 'desligar:AT', 'descartarMfa', 'GET estado']);
  assert.strictEqual(d.els.swMfa.checked, false);
  assert.strictEqual(d.avisos.length, 1);
  assert.ok(/desligada/.test(d.avisos[0][0]));
});

teste('desligar com erro: nada muda, a chave continua ligada e o erro aparece no cartao', async () => {
  const d = pagina({ estado: { ligado: true, lista: { ativa: true } } });
  await d.s.entrouEmListas();
  d.cognito.falharDesligar = Object.assign(new Error('x'), { codigo: 'rede' });
  await d.s.trocar(false); await esperar();
  assert.strictEqual(d.els.swMfa.checked, true);
  assert.ok(/sem conexão/.test(d.els.mfaMsg.textContent));
  assert.strictEqual(d.avisos.length, 0);
});

teste('gerar nova lista: sem segundo login; so confirma ao guardar; nao liga nem desliga nada', async () => {
  const d = pagina({ estado: { ligado: true, lista: { ativa: true, geradaEm: '2026-11-05T10:00:00Z' } } });
  await d.s.entrouEmListas();
  await d.s.gerarNova(); await esperar();
  assert.ok(!d.chamadas.includes('iniciarMfa'));
  assert.strictEqual(d.els.mfLista.hidden, false);
  assert.ok(/continua valendo/.test(d.els.mfListaCtx.textContent));
  assert.ok(!d.chamadas.includes('POST confirmar:L1'));
  marcarCheck(d);
  await d.s.guardei(); await esperar();
  assert.ok(d.chamadas.includes('POST confirmar:L1'));
  assert.ok(!d.chamadas.some((c) => c.startsWith('ligar') || c.startsWith('desligar')));
  assert.strictEqual(d.els.mfLista.hidden, true);
  assert.ok(/anterior deixou de valer/.test(d.avisos[0][1]));
});

teste('copiar e imprimir levam os codigos e a frase; a lista so existe em memoria', async () => {
  const d = pagina();
  await d.s.entrouEmListas(); await d.s.trocar(true); await esperar();
  d.els.mfCodigo.value = '123456'; await d.s.verificar(); await esperar();
  await d.els.mfCopiar.onclick();
  d.els.mfImprimir.onclick();
  for (const t of [d.copiados[0], d.impressos[0]]) {
    assert.ok(CODIGOS.every((c) => t.includes(c)));
    assert.ok(t.includes('SE PERDER A LISTA, PERDE O ACESSO.'));
  }
  assert.strictEqual(d.armazem.size, 0);
});

teste('navegador: a intencao (so a acao e o prazo, sem token) vai ao sessionStorage antes do login e o retorno retoma', async () => {
  const d = pagina({ navegador: true, semToken: true });
  await d.s.entrouEmListas();
  await d.s.trocar(true); // a promessa do login nao volta (a pagina vai embora)
  await esperar();
  const guardado = d.armazem.get('financas.mfaAcao');
  assert.deepStrictEqual(JSON.parse(guardado), { acao: 'ligar', ate: 1000 + 600000 });
  assert.ok(!/AT/.test(guardado), 'nenhum token');
  // a pagina recarrega: o access token ja esta em memoria e a acao e retomada
  const e = pagina({ navegador: true });
  e.armazem.set('financas.mfaAcao', guardado);
  e.env.armazenamento.setItem('financas.mfaAcao', guardado);
  await e.retorno && e.retorno();
  await esperar();
  assert.strictEqual(e.armazem.has('financas.mfaAcao'), false, 'a intencao e consumida');
  assert.ok(e.irListas >= 1);
  assert.strictEqual(e.els.mfLig.hidden, false);
});

teste('navegador: intencao vencida ou sem token em memoria nao retoma nada', async () => {
  const d = pagina({ navegador: true, semToken: true });
  d.armazem.set('financas.mfaAcao', JSON.stringify({ acao: 'ligar', ate: 1000 + 1 }));
  d.retorno();
  await esperar();
  assert.strictEqual(d.els.mfLig.hidden, true);
  const e = pagina({ navegador: true });
  e.armazem.set('financas.mfaAcao', JSON.stringify({ acao: 'ligar', ate: 5 }));
  e.retorno(); await esperar();
  assert.strictEqual(e.els.mfLig.hidden, true, 'vencida');
  assert.strictEqual(e.armazem.size, 0);
});

teste('as duas telas entram em camadasAbertas() e fecharCamadaTopo() (o voltar do celular fecha a de cima)', async () => {
  const fs = require('fs'), path = require('path');
  const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8').replace(/\r\n/g, '\n');
  const corpo = (nome) => { const i = app.indexOf('function ' + nome + '('); assert.ok(i >= 0, nome); return app.slice(i, app.indexOf('\n}\n', i)); };
  for (const [veu, fechar] of [['mfLista', '#mfListaVolta'], ['mfLig', '#mfLigVolta']]) {
    assert.ok(corpo('camadasAbertas').includes("camadaVisivel('" + veu + "')"), veu + ' falta em camadasAbertas()');
    assert.ok(corpo('fecharCamadaTopo').includes("camadaVisivel('" + veu + "')") && corpo('fecharCamadaTopo').includes("'" + fechar + "'"), veu + ' falta em fecharCamadaTopo()');
  }
  const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
  for (const id of ['mfaCard', 'swMfa', 'mfLig', 'mfLista', 'mfLigVolta', 'mfListaVolta', 'mfEntendi', 'mfGuardei']) assert.ok(html.includes('id="' + id + '"'), id + ' falta no index.html');
  assert.ok(html.includes('SE PERDER A LISTA, PERDE O ACESSO.'));
  assert.ok(!/\bMFA\b/.test(html.slice(html.indexOf('id="mfaCard"'), html.indexOf('id="mfaCard"') + 6000)), 'a sigla MFA nao aparece na tela');
});

(async () => {
  for (const [nome, fn] of fila) {
    const sem = setTimeout(() => { console.log('FALHOU (promessa parada): ' + nome); process.exit(1); }, 5000);
    try { await fn(); console.log('ok   ' + nome); }
    catch (e) { falhas++; console.log('FALHOU ' + nome + '\n' + (e && e.stack)); }
    clearTimeout(sem);
  }
  if (falhas) { console.log(falhas + ' falha(s)'); process.exit(1); }
  console.log('todos passaram (' + fila.length + ')');
})();
