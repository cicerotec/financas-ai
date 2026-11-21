// A janela "Ligar um canal" e o aviso de "sem canal ativo" das telas de aviso. Demos: .ai/demos/demo-avisar-v3-canal.html, demo-lancar-v9-aviso-canal.html
// e demo-cartao-v5-aviso-canal.html (aprovadas em 2026-11-21). Regras: .ai/specs/telegram-vinculo/spec.md (R17 a R19).
// Um componente so: uma janela de baixo, por cima da tela atual, com um botao por canal. Telegram pede o link ao servidor e abre a url (como o cartao de
// Listas); "Neste aparelho" e o push e so aparece no app Android. Concluido (ou fechado) o canal, o app RELE o estado dos canais (GET /me) e a tela de baixo
// segue como estava: este modulo nunca mexe no que a pessoa digitou. O front so exibe o que o servidor diz (canais); nao decide quem recebe.
function criarCanais(env){
  const FRASE = {
    escolha: 'Escolha por onde receber os avisos. Eles dizem só quantos são, como “2 atrasados · 1 vence hoje”; o que é cada um você vê ao abrir o app.',
    pedindo: 'Pedindo o link…',
    redeLink: 'Não consegui pedir o link agora (sem conexão). Tente de novo.',
    naoConfigurado: 'O Telegram ainda não foi configurado no servidor.',
    linksDemais: 'Muitos links pedidos: tente de novo mais tarde.',
    venceu: 'O link venceu: toque em Telegram de novo.',
    ainda: 'Ainda não chegou. Toque em Iniciar no Telegram e volte aqui.',
    redeConferir: 'Não consegui conferir agora (sem conexão). Tente de novo.',
    tgJa: 'O Telegram já está ligado neste espaço.',
    pushJa: 'Este aparelho já recebe os avisos neste espaço.',
    ligado: 'Ligado ✓',
    redePush: 'Não consegui ligar agora (sem conexão). Tente de novo.',
    tgLigado: '✓ Avisos no Telegram ligados',
    pushLigado: '✓ Avisos neste aparelho ligados',
    dicaT: 'Você ainda não tem um canal ativo para receber este aviso',
    dicaS: 'O aviso é guardado, mas fica mudo até você ligar um canal.'
  };
  const LIMITE = 10 * 60 * 1000; // o link vale 10 minutos e uma vez (R2)
  const FECHA_DEPOIS = 900;      // a janela fecha sozinha um instante depois de o canal ligar
  const semRede = e => !e || (e.code === undefined && e.codigo === undefined) || e.codigo === 'rede';

  let passo = 'escolha'; // escolha | aguardando
  let aberta = false, ocupado = false, expira = 0, tique = null, pronto = false;
  const el = id => env.el(id);
  const canais = () => env.canais() || {};
  const semCanal = () => { const c = canais(); return !c.telegram && !c.push; };
  const msg = (t, ruim) => { const m = el('lcMsg'); m.textContent = t || ''; m.classList.toggle('ruim', !!ruim); };

  function pararTique(){ if(tique !== null){ env.limpar(tique); tique = null; } }
  function tempo(){
    if(passo !== 'aguardando') return;
    const resta = Math.max(0, Math.ceil((expira - env.agora()) / 1000));
    el('lcTempo').textContent = 'O link vale por mais ' + Math.floor(resta / 60) + ':' + ('0' + (resta % 60)).slice(-2) + ' (10 minutos no total).';
  }
  function desenhar(){
    if(!pronto) return;
    const c = canais();
    el('lcMm').textContent = FRASE.escolha;
    el('lcBotoes').hidden = passo !== 'escolha';
    el('lcPush').hidden = !env.noApp();
    el('lcTg').disabled = ocupado; el('lcPush').disabled = ocupado;
    el('lcTgS').textContent = c.telegram ? 'Ligado ✓' : 'Numa conversa do Telegram';
    el('lcPushS').textContent = c.push ? 'Ligado ✓' : 'Notificação do app';
    el('lcAguarda').hidden = passo !== 'aguardando';
    if(passo !== 'aguardando') el('lcAbrir').hidden = true;
    tempo();
  }
  function venceu(){ pararTique(); passo = 'escolha'; msg(FRASE.venceu, true); desenhar(); }
  function tic(){
    if(passo !== 'aguardando') return;
    if(env.agora() >= expira) venceu(); else tempo();
  }

  // ---------- abrir e fechar (a camada do botao voltar e a do toque fora) ----------
  function abrir(){
    if(!pronto) return;
    aberta = true; passo = 'escolha'; ocupado = false; msg(''); desenhar();
    el('lcOv').hidden = false;
    if(el('lcTg').focus) el('lcTg').focus();
  }
  // Fechar rele o estado dos canais (GET /me) e avisa a tela de baixo; sem rede, fica o que o app sabia.
  async function fechar(){
    if(!aberta) return;
    aberta = false; pararTique(); passo = 'escolha';
    el('lcOv').hidden = true;
    try { await env.recarregarCanais(); } catch (e) { /* sem conexao: a tela segue com o ultimo estado conhecido */ }
    env.aoMudar();
  }
  const concluir = texto => { env.toast(texto); env.depois(fechar, FECHA_DEPOIS); };

  // ---------- Telegram ----------
  function falhaLink(e){
    ocupado = false;
    msg(semRede(e) ? FRASE.redeLink : e.code === 503 ? FRASE.naoConfigurado : e.code === 429 ? FRASE.linksDemais : (e.message || FRASE.redeLink), true);
    desenhar();
  }
  async function ligarTelegram(){
    if(canais().telegram){ msg(FRASE.tgJa); return; }
    ocupado = true; msg(FRASE.pedindo); desenhar();
    let r;
    try { r = await env.gerarLink(); } catch (e) { falhaLink(e); return; }
    // o prazo e o do servidor (expiraEm); se o relogio do aparelho destoa, vale 10 minutos a partir de agora
    const agora = env.agora(), t = Date.parse(r && r.expiraEm);
    expira = (t > agora && t - agora <= LIMITE + 60 * 1000) ? t : agora + LIMITE;
    ocupado = false; passo = 'aguardando'; msg(''); desenhar();
    pararTique(); tique = env.intervalo(tic, 500);
    let abriu = false;
    try { abriu = !!env.abrir(r.url); } catch (e) { abriu = false; }
    if(!abriu){ const a = el('lcAbrir'); a.href = r.url; a.hidden = false; } // a janela foi bloqueada: a propria tela oferece o link
  }
  // Rele GET /me enquanto aguarda o Iniciar. silencioso: a releitura ao voltar ao primeiro plano nao reclama.
  async function reler(silencioso){
    let c;
    try { c = await env.recarregarCanais(); }
    catch (e) { if(!silencioso) msg(FRASE.redeConferir, true); return; }
    if(passo !== 'aguardando') return;
    if(c && c.telegram){
      pararTique(); passo = 'escolha'; msg(FRASE.ligado); desenhar();
      env.aoMudar();
      concluir(FRASE.tgLigado);
      return;
    }
    if(!silencioso){ msg(FRASE.ainda); tempo(); }
  }
  function cancelar(){ pararTique(); passo = 'escolha'; msg(''); desenhar(); } // o link ja gerado continua valendo ate vencer

  // ---------- Neste aparelho (so no app Android) ----------
  async function ligarAparelho(){
    if(canais().push){ msg(FRASE.pushJa); return; }
    ocupado = true; msg(''); desenhar();
    let r;
    try { r = await env.ligarAparelho(); } catch (e) { r = {ok: false, texto: FRASE.redePush}; }
    ocupado = false;
    if(r && r.cancelou){ desenhar(); return; } // "Agora nao" na explicacao: nada foi pedido
    if(!r || !r.ok){ msg((r && r.texto) || FRASE.redePush, true); desenhar(); return; }
    try { await env.recarregarCanais(); } catch (e) { /* o servidor ja registrou; a releitura de ao fechar corrige */ }
    msg(FRASE.ligado); desenhar();
    env.aoMudar();
    concluir(FRASE.pushLigado);
  }

  // ---------- o aviso de "sem canal ativo" das telas de aviso (R19) ----------
  // Pinta (ou limpa) a dica em `alvo`. Aparece quando a pessoa marcou o aviso e o espaco nao tem nenhum canal; "Continuar assim" a esconde (quem chama guarda isso
  // ate a pessoa marcar de novo) e "Ligar um canal" abre a janela. O salvar nunca e bloqueado: o aviso e guardado e fica mudo ate haver canal.
  function dica(alvo, mostrar, aoContinuar){
    if(!alvo) return;
    if(!(mostrar && semCanal())){ alvo.innerHTML = ''; return; }
    alvo.innerHTML = '<div class="lc-dica" role="note"><b>' + FRASE.dicaT + '</b>' + FRASE.dicaS +
      '<div class="lc-acoes"><button type="button" class="pri" data-lc="ligar">Ligar um canal</button><button type="button" data-lc="seguir">Continuar assim</button></div></div>';
    alvo.querySelector('[data-lc=ligar]').onclick = abrir;
    alvo.querySelector('[data-lc=seguir]').onclick = () => { alvo.innerHTML = ''; if(aoContinuar) aoContinuar(); };
  }

  function ligarJanela(){
    if(!env.el('lcOv') || !env.el('lcTg')) return false;
    pronto = true;
    el('lcTg').onclick = ligarTelegram;
    el('lcPush').onclick = ligarAparelho;
    el('lcFechar').onclick = fechar;
    el('lcIniciei').onclick = () => reler(false);
    el('lcCancelar').onclick = cancelar;
    el('lcOv').onclick = e => { if(e.target === el('lcOv')) fechar(); }; // o toque fora fecha
    env.aoVoltar(() => { if(aberta && passo === 'aguardando') reler(true); }); // uma releitura ao voltar ao primeiro plano, so enquanto aguarda
    return true;
  }
  return {ligarJanela, abrir, fechar, dica, semCanal, estaAberta: () => aberta, passoAgora: () => passo};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarCanais;
const CanaisApp = (typeof window !== 'undefined') ? criarCanais({
  el: id => document.getElementById(id),
  canais: () => (window.financasApi && window.financasApi.canais && window.financasApi.canais()) || {},
  recarregarCanais: () => window.financasApi.recarregarCanais(),
  gerarLink: () => window.financasApi.gerarLinkTelegram(),
  abrir: url => { const w = window.open(url, '_blank'); if(w) w.opener = null; return !!w; },
  noApp: () => typeof PushApp !== 'undefined' && !!PushApp && PushApp.disponivel(), // o push so existe no app Android: o front sabe pelo ambiente e so exibe
  ligarAparelho: () => PushApp.ligarDaJanela(),
  agora: () => Date.now(),
  intervalo: (f, ms) => setInterval(f, ms),
  limpar: id => clearInterval(id),
  depois: (f, ms) => setTimeout(f, ms),
  aoVoltar: cb => document.addEventListener('visibilitychange', () => { if(document.visibilityState === 'visible') cb(); }),
  toast: (m, erro) => { if(typeof toast === 'function') toast(m, erro); },
  aoMudar: () => { if(typeof atualizarCanais === 'function') atualizarCanais(); } // as telas de aviso e o cartao de Listas leem o estado de novo
}) : null;
