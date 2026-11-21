// Avisos no Telegram, ligados por link (cartao Avisos de Listas). Demo: .ai/demos/demo-listas-v11-telegram-vinculo.html (aprovada em 2026-11-21).
// Regras: .ai/specs/telegram-vinculo/spec.md (R3, R14 e R17). A chave pede um link ao servidor (POST /spaces/{sid}/telegram/vinculo), abre a url e a pessoa
// toca em Iniciar no Telegram. O estado "ligado" vem SEMPRE do servidor (GET /me, canais.telegram), relido ao tocar em "Ja iniciei" e, uma vez, quando o app
// volta ao primeiro plano enquanto aguarda: nao ha sondagem por tempo. O front nao ve, mostra nem guarda chat id; desligar e o PUT /perfil com o Id vazio.
function criarTelegram(env){
  const FRASE = {
    padrao: 'Vale só para você, neste espaço.',
    aguardando: 'Falta tocar em Iniciar no Telegram.',
    ligado: 'Os avisos deste espaço chegam no seu Telegram.',
    pedindo: 'Pedindo o link…',
    redeLink: 'Não consegui pedir o link agora (sem conexão). Tente de novo.',
    naoConfigurado: 'O Telegram ainda não foi configurado no servidor.',
    linksDemais: 'Muitos links pedidos: tente de novo mais tarde.',
    venceu: 'O link venceu: toque em Ligar Telegram de novo.',
    ainda: 'Ainda não chegou. Toque em Iniciar no Telegram e volte aqui.',
    redeConferir: 'Não consegui conferir agora (sem conexão). Tente de novo.',
    ligou: 'Ligado ✓ Use "Enviar aviso de teste" para conferir.',
    desligando: 'Desligando…',
    redeDesligar: 'Não consegui desligar agora (sem conexão). Tente de novo.',
    enviando: 'Enviando...',
    enviado: 'Enviado ✓ Confira o Telegram.',
    redeTeste: 'Não consegui enviar o teste (sem conexão).'
  };
  const LIMITE = 10 * 60 * 1000; // o link vale 10 minutos e uma vez (R2)
  const K_ANTIGO = 'financas.telegram'; // o Id lembrado de antes: nunca mais se guarda um chat id no aparelho (R17)
  // Erro da API sem codigo e falha de rede; com codigo, a frase e do servidor (ou, nos poucos casos conhecidos, a do cartao).
  const semRede = e => !e || (e.code === undefined && e.codigo === undefined) || e.codigo === 'rede';
  const frase = (e, rede) => (semRede(e) ? rede : (e.message || rede));

  let estado = 'desligado'; // desligado | aguardando | ligado
  let ocupado = false, ligando = false, pronto = false, expira = 0, mostrarLigar = false, rotuloLigar = 'Ligar Telegram', tique = null;
  const sw = () => env.el('swTelegram'), msg = () => env.el('telegramMsg');
  const aviso = (t, ruim) => { const m = msg(); m.textContent = t || ''; m.classList.toggle('ruim', !!ruim); };

  function pararTique(){ if(tique !== null){ env.limpar(tique); tique = null; } }
  function tempo(){
    if(estado !== 'aguardando') return;
    const resta = Math.max(0, Math.ceil((expira - env.agora()) / 1000));
    env.el('tgTempo').textContent = 'O link vale por mais ' + Math.floor(resta / 60) + ':' + ('0' + (resta % 60)).slice(-2) + ' (10 minutos no total).';
  }
  function desenhar(){
    if(!pronto) return;
    const ligado = estado === 'ligado', aguarda = estado === 'aguardando';
    sw().checked = ligado || aguarda || ligando; sw().disabled = ocupado; // ligando: a chave fica marcada enquanto o link e pedido
    env.el('tgAguarda').hidden = !aguarda;
    env.el('tgApoio').textContent = ligado ? FRASE.ligado : (aguarda ? FRASE.aguardando : FRASE.padrao);
    env.el('btnTesteTelegram').hidden = !ligado;
    env.el('btnTgLigar').hidden = !mostrarLigar; env.el('btnTgLigar').textContent = rotuloLigar;
    env.el('tgAcoes').hidden = !(ligado || mostrarLigar);
    if(!aguarda) env.el('tgAbrir').hidden = true;
    tempo();
  }
  function venceu(){
    pararTique();
    estado = 'desligado'; mostrarLigar = true; rotuloLigar = 'Ligar Telegram';
    aviso(FRASE.venceu, true); desenhar();
  }
  function tic(){
    if(estado !== 'aguardando') return;
    if(env.agora() >= expira) venceu(); else tempo();
  }
  // Chamado quando o espaco e conhecido: a chave segue o que o servidor mandou (canais.telegram). Recarregar o app no meio do aguardando cai aqui.
  function atualizar(){
    pararTique();
    ocupado = false; ligando = false; mostrarLigar = false;
    estado = (env.canais() || {}).telegram ? 'ligado' : 'desligado';
    if(pronto){ aviso(''); desenhar(); }
    return estado === 'ligado';
  }
  function falhaLigar(texto){ ocupado = false; ligando = false; mostrarLigar = true; rotuloLigar = 'Tentar de novo'; aviso(texto, true); desenhar(); }
  function fraseDoLink(e){
    if(semRede(e)) return FRASE.redeLink;
    if(e.code === 503) return FRASE.naoConfigurado;
    if(e.code === 429) return FRASE.linksDemais;
    return e.message || FRASE.redeLink;
  }
  async function ligar(){
    ocupado = true; ligando = true; mostrarLigar = false; aviso(FRASE.pedindo); desenhar();
    let r;
    try { r = await env.gerarLink(); }
    catch (e) { falhaLigar(fraseDoLink(e)); return; }
    // o tempo restante e o do servidor (expiraEm); se o relogio do aparelho destoa, vale o prazo de 10 minutos a partir de agora
    const agora = env.agora(), t = Date.parse(r && r.expiraEm);
    expira = (t > agora && t - agora <= LIMITE + 60 * 1000) ? t : agora + LIMITE;
    ocupado = false; ligando = false; estado = 'aguardando'; aviso(''); desenhar();
    pararTique(); tique = env.intervalo(tic, 500);
    // abrir a url: se o navegador bloquear a janela (a chamada veio depois de uma espera), a propria tela oferece o link para tocar
    let abriu = false;
    try { abriu = !!env.abrir(r.url); } catch (e) { abriu = false; }
    if(!abriu){ const a = env.el('tgAbrir'); a.href = r.url; a.hidden = false; }
  }
  function cancelar(){ pararTique(); estado = 'desligado'; aviso(''); desenhar(); } // o link ja gerado continua valendo ate vencer
  async function desligar(){
    ocupado = true; aviso(FRASE.desligando); desenhar();
    try { await env.desligar(); }
    catch (e) { ocupado = false; aviso(frase(e, FRASE.redeDesligar), true); desenhar(); return; } // continua ligada
    ocupado = false; estado = 'desligado'; aviso(''); desenhar();
    env.toast('✓ Telegram desligado');
    env.aoMudar();
  }
  // Rele GET /me. silencioso: a releitura do primeiro plano nao reclama de rede nem diz "ainda nao chegou".
  async function reler(silencioso){
    let c;
    try { c = await env.recarregarCanais(); }
    catch (e) { if(!silencioso) aviso(frase(e, FRASE.redeConferir), true); return; }
    if(estado !== 'aguardando') return; // o estado mudou enquanto a leitura voltava
    if(c && c.telegram){
      pararTique(); estado = 'ligado'; mostrarLigar = false; aviso(FRASE.ligou); desenhar();
      env.toast('✓ Avisos no Telegram ligados');
      env.aoMudar();
      return;
    }
    if(!silencioso){ aviso(FRASE.ainda); tempo(); }
  }
  async function testar(){
    aviso(FRASE.enviando);
    try { await env.testar(); aviso(FRASE.enviado); }
    catch (e) { aviso('⚠️ ' + frase(e, FRASE.redeTeste), true); }
  }
  function ligarCartao(){
    if(!env.el('telegramCard') || !env.el('swTelegram')) return false;
    pronto = true;
    try { env.armazenamento.removeItem(K_ANTIGO); } catch (e) { /* sem armazenamento: nada a apagar */ }
    sw().onchange = () => { if(sw().checked) return ligar(); if(estado === 'aguardando') return cancelar(); if(estado === 'ligado') return desligar(); };
    env.el('btnTgIniciei').onclick = () => reler(false);
    env.el('btnTgCancelar').onclick = cancelar;
    env.el('btnTgLigar').onclick = ligar;
    env.el('btnTesteTelegram').onclick = testar;
    env.aoVoltar(() => { if(estado === 'aguardando') reler(true); }); // uma releitura ao voltar ao primeiro plano, so enquanto aguarda
    atualizar();
    return true;
  }
  return {ligarCartao, atualizar, ligar, cancelar, desligar, reler, testar, estadoAgora: () => estado, ligadoAgora: () => estado === 'ligado'};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarTelegram;
const TelegramApp = (typeof window !== 'undefined') ? criarTelegram({
  el: id => document.getElementById(id),
  canais: () => (window.financasApi && window.financasApi.canais && window.financasApi.canais()) || {},
  gerarLink: () => window.financasApi.gerarLinkTelegram(),
  recarregarCanais: () => window.financasApi.recarregarCanais(),
  desligar: () => window.financasApi.desligarTelegram(),
  testar: () => window.financasApi.testarAviso(),
  abrir: url => { const w = window.open(url, '_blank'); if(w) w.opener = null; return !!w; },
  armazenamento: window.localStorage,
  agora: () => Date.now(),
  intervalo: (f, ms) => setInterval(f, ms),
  limpar: id => clearInterval(id),
  aoVoltar: cb => document.addEventListener('visibilitychange', () => { if(document.visibilityState === 'visible') cb(); }),
  toast: (m, erro) => { if(typeof toast === 'function') toast(m, erro); },
  aoMudar: () => { if(typeof atualizarCampoAviso === 'function') atualizarCampoAviso(); } // o aviso do Lançar depende de ter o Telegram ligado
}) : null;
