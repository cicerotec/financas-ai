// Push: avisos como notificacao do app Android (item 54, fase 4). Spec: .ai/specs/app-android-push/spec.md (R14 a R17, R19, R20).
// Padrao: .ai/padrao-de-telas.md ("Avisos neste aparelho"), demo .ai/demos/arquivo/demo-push-v1.html (aprovada em 2026-11-17).
// So existe no app (Capacitor.isNativePlatform()); no navegador e na PWA este modulo nao faz nada e a subsecao nao aparece.
// O token do aparelho vem do plugin Push (Java, SDK do Firebase) e SO passa por aqui rumo a API: nunca e guardado, mostrado nem escrito
// em log (R23). No aparelho fica apenas financas.push = {"<espaco>": {em: quando registrou, h: impressao curta do token}} (R15).
// O front nao decide quem recebe nem o que a notificacao diz: guarda a escolha da pessoa e chama a API (R24).
function criarPush(env){
  const K = 'financas.push', SETE_DIAS = 7 * 24 * 3600 * 1000, LIMITE_SAIR = 8000, ATRASO_TESTE = 10000;
  const cap = env.Capacitor;
  const nativo = !!(cap && typeof cap.isNativePlatform === 'function' && cap.isNativePlatform());
  const Plugin = (nativo && cap.Plugins && cap.Plugins.Push) || null;

  const FRASE = {
    padrao: 'Vale só neste espaço e neste aparelho.',
    indisponivel: 'Este aparelho não recebe avisos do app.',
    unico: 'Este é o único aparelho que recebe avisos neste espaço.',
    ligado: 'Este aparelho recebe os avisos neste espaço.',
    bloqueada: 'As notificações estão desligadas para o Finanças no Android. Ligue-as nas configurações do app para receber avisos.',
    negou: 'Sem a permissão do Android o aviso não chega. Nada foi ligado.',
    redeLigar: 'Não consegui ligar agora (sem conexão). Tente de novo.',
    redeDesligar: 'Não consegui desligar agora (sem conexão). Tente de novo.',
    redeTeste: 'Não consegui enviar o teste (sem conexão).',
    esperaTeste: 'O teste sai em 10 segundos. Aperte o botão Início do Android para ver a notificação.',
    erroLigar: 'Não consegui ligar agora. Tente de novo.',
    erroDesligar: 'Não consegui desligar agora. Tente de novo.'
  };
  const n = x => (x === 1 ? FRASE.unico : x + ' aparelhos recebem avisos neste espaço.');

  // ---------- o que fica no aparelho (so a escolha por espaco, a data do ultimo registro e a impressao do token) ----------
  const lerTudo = () => {
    try { const o = JSON.parse(env.armazenamento.getItem(K) || 'null'); return o && typeof o === 'object' && !Array.isArray(o) ? o : {}; } catch (e) { return {}; }
  };
  const gravarTudo = o => { if(Object.keys(o).length) env.armazenamento.setItem(K, JSON.stringify(o)); else env.armazenamento.removeItem(K); };
  const ligadoNo = sid => !!sid && !!lerTudo()[sid];
  const impressao = async token => (await env.digest(token)).slice(0, 16);
  async function guardar(sid, token){
    const t = lerTudo();
    t[sid] = {em: env.agora(), h: await impressao(token)};
    gravarTudo(t);
  }
  function esquecer(sid){ const t = lerTudo(); delete t[sid]; gravarTudo(t); }

  const falha = (motivo, texto, extra) => Object.assign({ok: false, motivo, texto}, extra || {});
  // Erro da API: 409 e o limite de aparelhos (a frase vem do servidor); sem codigo e falha de rede.
  function daApi(e, rede, outro){
    if(e && e.code === 409) return falha('limite', e.message);
    if(!e || (e.code === undefined && e.codigo === undefined) || e.codigo === 'rede') return falha('rede', rede, {denovo: true});
    return falha('erro', outro, {denovo: true});
  }
  async function tokenDoAparelho(){
    const r = await Plugin.token();
    if(!r || !r.token) throw new Error('sem token');
    return r.token;
  }

  // ---------- ligar, desligar e testar (R2, R3, R16, R17) ----------
  // Chamado DEPOIS da explicacao (a janela e da tela). Devolve {ok:true, aparelhos} ou {ok:false, motivo, texto, config?, denovo?}.
  async function ligar(sid){
    if(!Plugin || !sid) return falha('indisponivel', FRASE.indisponivel);
    try {
      const d = await Plugin.disponivel();
      if(!d || !d.disponivel) return falha('indisponivel', FRASE.indisponivel);
      let perm = (await Plugin.permissao()).permissao;
      if(perm === 'negada') return falha('negada', FRASE.bloqueada, {config: true}); // o app nao pede de novo sozinho (R17)
      if(perm === 'pergunte'){
        perm = (await Plugin.pedirPermissao()).permissao;
        if(perm !== 'concedida') return falha('recusada', FRASE.negou, {config: true});
      }
    } catch (e) { return falha('indisponivel', FRASE.indisponivel); }
    let token;
    try { token = await tokenDoAparelho(); } catch (e) { return falha('indisponivel', FRASE.indisponivel); }
    let aparelhos;
    try { aparelhos = await env.api.registrar(token, sid); } catch (e) { return daApi(e, FRASE.redeLigar, FRASE.erroLigar); }
    await guardar(sid, token).catch(() => {}); // so depois de o servidor aceitar (P-T19)
    try { await Plugin.marcarRegistrado(); } catch (e) { /* so deixa de limpar a marca; a proxima abertura registra de novo */ }
    return {ok: true, aparelhos};
  }
  async function desligar(sid){
    if(!Plugin || !sid) return falha('indisponivel', FRASE.indisponivel);
    let token;
    try { token = await tokenDoAparelho(); } catch (e) { return falha('erro', FRASE.erroDesligar, {denovo: true}); }
    try { await env.api.remover(token, sid); } catch (e) { return daApi(e, FRASE.redeDesligar, FRASE.erroDesligar); }
    esquecer(sid);
    return {ok: true};
  }
  async function testar(){
    try { await env.api.testar('push'); return {ok: true}; }
    catch (e) {
      if(e && e.code !== undefined && e.message) return falha('recusado', e.message); // a frase do servidor (sem aparelho, nao configurado, FCM recusou)
      return falha('rede', FRASE.redeTeste);
    }
  }

  // ---------- registrar de novo so quando precisa (R15) ----------
  // Com sessao, para cada espaco ligado neste aparelho: so se o token mudou (marca do onNewToken ou impressao diferente) ou se o ultimo
  // registro tem 7 dias ou mais. Fora disso a abertura nao escreve nada. Falha e silenciosa: tenta de novo na abertura seguinte.
  async function manterRegistro(){
    const tudo = lerTudo(), sids = Object.keys(tudo);
    if(!Plugin || !sids.length) return {registrados: 0};
    let token, marcou = false;
    try { token = await tokenDoAparelho(); } catch (e) { return {registrados: 0}; }
    try { marcou = !!((await Plugin.mudou()) || {}).mudou; } catch (e) { /* sem a marca, vale a impressao e a data */ }
    const h = await impressao(token), agora = env.agora();
    let feitos = 0, falhou = false;
    for(const sid of sids){
      const r = tudo[sid] || {};
      if(!(marcou || r.h !== h || !r.em || agora - r.em >= SETE_DIAS)) continue;
      try { await env.api.registrar(token, sid); await guardar(sid, token); feitos++; }
      catch (e) { falhou = true; if(e && e.code === 403) esquecer(sid); } // o vinculo acabou: a chave daquele espaco nao vale mais
    }
    if(marcou && !falhou) { try { await Plugin.marcarRegistrado(); } catch (e) {} }
    return {registrados: feitos};
  }

  // ---------- Sair (R19) ----------
  // Antes de apagar a sessao: tira o aparelho de TODOS os espacos e invalida o token no FCM. Cada passo falha sem impedir o Sair.
  async function sair(){
    const corrida = p => Promise.race([p, env.esperar(LIMITE_SAIR)]);
    let token = null;
    try { if(Plugin) token = await corrida(tokenDoAparelho()); } catch (e) { token = null; }
    if(token && typeof token === 'string'){ try { await corrida(env.api.sair(token)); } catch (e) { /* sem rede: o FCM invalida o token e a rotina o apaga */ } }
    try { if(Plugin) await corrida(Plugin.apagarToken()); } catch (e) {}
    try { env.armazenamento.removeItem(K); } catch (e) {}
  }

  // ---------- o cartao (Listas, Avisos, subsecao "Avisos neste aparelho") ----------
  function ligarCartao(){
    const sec = env.el('pushCard');
    if(!nativo || !sec) return false;
    const sw = env.el('swPush'), msg = env.el('pushMsg'), apoio = env.el('pushApoio'), acoes = env.el('pushAcoes');
    const bTeste = env.el('btnPushTeste'), bConfig = env.el('btnPushConfig'), bDeNovo = env.el('btnPushDeNovo');
    let ocupado = false, indisponivel = !Plugin, contagem = null;
    const aviso = (t, ruim) => { msg.textContent = t || ''; msg.classList.toggle('ruim', !!ruim); };
    const botoes = (teste, config, denovo) => { bTeste.hidden = !teste; bConfig.hidden = !config; bDeNovo.hidden = !denovo; acoes.hidden = !(teste || config || denovo); };
    const desenhar = () => {
      const sid = env.sid(), lig = !indisponivel && ligadoNo(sid);
      sw.checked = lig;
      sw.disabled = ocupado || indisponivel || !sid;
      apoio.textContent = indisponivel ? FRASE.indisponivel : lig ? (contagem === null ? FRASE.ligado : n(contagem)) : FRASE.padrao;
    };
    const mostrarFalha = r => {
      aviso(r.texto, true);
      botoes(false, !!r.config, !!r.denovo);
      desenhar();
    };
    async function ativar(){
      const sid = env.sid();
      if(!(await env.explicar())) return;
      ocupado = true; aviso('Ligando…'); botoes(false, false, false); desenhar();
      const r = await ligar(sid);
      ocupado = false;
      if(!r.ok){ contagem = null; mostrarFalha(r); return; }
      contagem = r.aparelhos; aviso(''); botoes(true, false, false); desenhar();
      env.toast('✓ Este aparelho recebe os avisos');
    }
    async function desativar(){
      const sid = env.sid();
      ocupado = true; aviso('Desligando…'); botoes(false, false, false); desenhar();
      const r = await desligar(sid);
      ocupado = false;
      if(!r.ok){ aviso(r.texto, true); botoes(ligadoNo(sid), false, false); desenhar(); return; } // continua ligada
      contagem = null; aviso(''); botoes(false, false, false); desenhar();
      env.toast('✓ Este aparelho não recebe mais avisos');
    }
    sec.hidden = false;
    sw.onchange = () => {
      const quer = sw.checked;
      sw.checked = ligadoNo(env.sid()); // so muda depois da resposta
      aviso(''); botoes(ligadoNo(env.sid()), false, false);
      return quer ? ativar() : desativar();
    };
    bDeNovo.onclick = () => { aviso(''); botoes(false, false, false); return ativar(); };
    bConfig.onclick = () => { try { if(Plugin && Plugin.abrirConfiguracoes) Promise.resolve(Plugin.abrirConfiguracoes()).catch(() => {}); } catch (e) {} };
    // Com o app aberto o Android nao mostra a notificacao (R12/R14). Por isso o teste espera 10 s: da tempo de sair do app.
    let testando = false;
    bTeste.onclick = async () => {
      if(testando) return;
      testando = true; bTeste.disabled = true; aviso(FRASE.esperaTeste);
      try {
        await env.esperar(ATRASO_TESTE);
        aviso('');
        const r = await testar();
        if(r.ok) env.toast('✓ Teste enviado: a notificação chega em alguns segundos');
        else env.toast(r.texto, true);
      } finally { testando = false; bTeste.disabled = false; aviso(''); }
    };
    desenhar(); botoes(ligadoNo(env.sid()), false, false);
    if(Plugin) Promise.resolve().then(() => Plugin.disponivel()).then(d => { indisponivel = !(d && d.disponivel); desenhar(); }, () => { indisponivel = true; desenhar(); });
    return true;
  }

  // Chamado quando o espaco ja esta conhecido (apos o acesso). A subida nao le nada do servidor por causa disto (so escreve, e so quando precisa).
  function iniciar(){
    if(!nativo) return false;
    const ok = ligarCartao();
    Promise.resolve().then(manterRegistro).catch(() => {});
    return ok;
  }
  return {iniciar, ligarCartao, ligar, desligar, testar, manterRegistro, sair, ligadoNo, disponivel: () => !!Plugin};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarPush;
const PushApp = (typeof window !== 'undefined') ? criarPush({
  Capacitor: window.Capacitor,
  el: id => document.getElementById(id),
  armazenamento: window.localStorage,
  agora: () => Date.now(),
  digest: async t => {
    const b = await window.crypto.subtle.digest('SHA-256', new TextEncoder().encode(t));
    return Array.from(new Uint8Array(b)).map(x => x.toString(16).padStart(2, '0')).join('');
  },
  esperar: ms => new Promise(r => setTimeout(r, ms)), // resolve sem valor: quem corre contra o tempo ve "nada"
  toast: (m, erro) => { if(typeof toast === 'function') toast(m, erro); },
  // a explicacao ANTES do pedido do Android (demo-push-v1; R17): janela de confirmacao do app
  explicar: () => (typeof confirmarModal === 'function'
    ? confirmarModal('Para mostrar a notificação, o Android vai pedir a sua permissão. A notificação diz só quantos lançamentos estão atrasados, vencem hoje ou vencem em breve: nunca descrição, valor nem meio de pagamento.', 'Continuar', 'Agora não', 'Receber avisos neste aparelho?')
    : Promise.resolve(true)),
  sid: () => (window.financasApi && window.financasApi.espacoId ? window.financasApi.espacoId() : ''),
  api: {
    registrar: (t, s) => window.financasApi.registrarAparelho(t, s),
    remover: (t, s) => window.financasApi.removerAparelho(t, s),
    sair: t => window.financasApi.sairAparelho(t),
    testar: c => window.financasApi.testarAviso(c)
  }
}) : null;
