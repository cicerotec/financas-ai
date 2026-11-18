// Avisos no Telegram com chave de ligar e desligar (cartao Avisos de Listas). Demo: .ai/demos/demo-push-v2-telegram-chave.html (aprovada em 2026-11-18).
// Padrao: .ai/padrao-de-telas.md ("Avisos"). Ligar = guardar o chat id (PUT /perfil); desligar = o mesmo com o Id vazio, que apaga o Id (nenhum campo
// novo no DynamoDB). O front so guarda a escolha da pessoa e mostra a resposta: quem valida o Id e o servidor (domain.ChatIDValido).
// Ao desligar, o ultimo Id fica so neste aparelho (financas.telegram = {"<espaco>": "<id>"}) para preencher o campo ao ligar de novo; o Sair apaga.
function criarTelegram(env){
  const FRASE = {
    padrao: 'Vale só para você, neste espaço.',
    ligado: 'Os avisos deste espaço chegam neste Telegram.',
    abrir: 'Cole o seu Id e toque em Salvar.',
    vazio: 'Cole o Id do Telegram ou desligue a chave.',
    salvo: 'Salvo ✓ Use "Enviar aviso de teste" para conferir.',
    enviado: 'Enviado ✓ Confira o Telegram.',
    redeLigar: 'Não consegui salvar agora (sem conexão). Tente de novo.',
    redeDesligar: 'Não consegui desligar agora (sem conexão). Tente de novo.',
    redeTeste: 'Não consegui enviar o teste (sem conexão).',
    lembrado: 'Usei o último Id deste aparelho. Toque em Salvar para ligar.'
  };
  const K = 'financas.telegram';
  const lerMem = () => { try { const o = JSON.parse(env.armazenamento.getItem(K) || 'null'); return o && typeof o === 'object' && !Array.isArray(o) ? o : {}; } catch (e) { return {}; } };
  const lembrado = () => { const v = lerMem()[env.sid()]; return typeof v === 'string' ? v : ''; };
  function lembrar(id){
    try {
      const o = lerMem(), sid = env.sid();
      if(!sid) return;
      if(id) o[sid] = id; else delete o[sid];
      if(Object.keys(o).length) env.armazenamento.setItem(K, JSON.stringify(o)); else env.armazenamento.removeItem(K);
    } catch (e) { /* sem armazenamento: so nao lembra */ }
  }
  // Erro da API sem codigo e falha de rede; com codigo, a frase e do servidor.
  const semRede = e => !e || (e.code === undefined && e.codigo === undefined) || e.codigo === 'rede';
  const frase = (e, rede) => (semRede(e) ? rede : (e.message || rede));

  let ligado = false, aberto = false, ocupado = false, pronto = false;
  const sw = () => env.el('swTelegram'), inp = () => env.el('telegramChatId'), msg = () => env.el('telegramMsg');
  const aviso = (t, ruim) => { const m = msg(); m.textContent = t || ''; m.classList.toggle('ruim', !!ruim); };

  function desenhar(){
    if(!pronto) return;
    sw().checked = ligado; sw().disabled = ocupado;
    env.el('tgCampos').hidden = !(ligado || aberto);
    env.el('tgAjuda').hidden = ligado || !aberto;
    env.el('btnTesteTelegram').hidden = !ligado;
    env.el('btnTgAgoraNao').hidden = ligado;
    env.el('tgApoio').textContent = ligado ? FRASE.ligado : FRASE.padrao;
  }
  // Chamado quando o espaco e conhecido (e depois de qualquer mudanca de espaco): a chave segue o Id que o servidor mandou.
  function atualizar(){
    const id = env.chatId() || '';
    ligado = !!id; aberto = false; ocupado = false;
    if(pronto){ inp().value = id; aviso(''); desenhar(); }
    return ligado;
  }
  async function trocar(quer){
    sw().checked = ligado; aviso('');
    if(quer){ const ult = lembrado(); aberto = true; inp().value = ult; desenhar(); aviso(ult ? FRASE.lembrado : FRASE.abrir); if(inp().focus) inp().focus(); return; }
    const id = (inp().value || '').trim() || env.chatId();
    ocupado = true; desenhar(); aviso('Desligando…');
    try { await env.definir(''); }
    catch (e) { ocupado = false; aviso(frase(e, FRASE.redeDesligar), true); desenhar(); return; } // continua ligada
    ocupado = false; ligado = false; aberto = false; inp().value = ''; aviso(''); desenhar();
    lembrar(id); // so neste aparelho: o servidor ficou sem o Id
    env.toast('✓ Telegram desligado (o Id fica só neste aparelho)');
    env.aoMudar();
  }
  async function salvar(){
    const v = (inp().value || '').trim();
    if(!v){ aviso(FRASE.vazio, true); return; } // vazio desvincularia: isso e o desligar
    aviso('Salvando...');
    let r;
    try { r = await env.definir(v); }
    catch (e) { aviso(frase(e, FRASE.redeLigar), true); return; }
    ligado = !!r; aberto = false; inp().value = r || '';
    if(r) lembrar(''); // agora o Id esta no servidor
    desenhar(); aviso(FRASE.salvo); env.toast('✓ Avisos no Telegram ligados');
    env.aoMudar();
  }
  async function testar(){
    aviso('Enviando...');
    try { await env.testar(); aviso(FRASE.enviado); }
    catch (e) { aviso('⚠️ ' + frase(e, FRASE.redeTeste), true); }
  }
  function ligarCartao(){
    if(!env.el('telegramCard') || !env.el('swTelegram')) return false;
    pronto = true;
    sw().onchange = () => trocar(sw().checked);
    env.el('btnSalvarTelegram').onclick = salvar;
    env.el('btnTesteTelegram').onclick = testar;
    env.el('btnTgAgoraNao').onclick = () => { aberto = false; aviso(''); desenhar(); };
    atualizar();
    return true;
  }
  return {ligarCartao, atualizar, trocar, salvar, testar, ligadoAgora: () => ligado};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarTelegram;
const TelegramApp = (typeof window !== 'undefined') ? criarTelegram({
  el: id => document.getElementById(id),
  chatId: () => (window.financasApi && window.financasApi.telegramChatId && window.financasApi.telegramChatId()) || '',
  definir: id => window.financasApi.definirTelegram(id),
  testar: () => window.financasApi.testarAviso(),
  armazenamento: window.localStorage,
  sid: () => (window.financasApi && window.financasApi.espacoId ? window.financasApi.espacoId() : ''),
  toast: (m, erro) => { if(typeof toast === 'function') toast(m, erro); },
  aoMudar: () => { if(typeof atualizarCampoAviso === 'function') atualizarCampoAviso(); } // o aviso do Lançar depende de ter o Telegram vinculado
}) : null;
