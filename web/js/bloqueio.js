// Bloqueio: a trava por biometria do app Android (item 54, fase 3). Spec: .ai/specs/app-android/spec.md, R19 a R23.
// Padrao: .ai/padrao-de-telas.md ("Bloqueio"), demo .ai/demos/demo-bloqueio-v1.html (aprovada em 2026-11-12).
// So existe no app (Capacitor.isNativePlatform()); no navegador e na PWA este modulo nao faz nada e a subsecao Seguranca nao aparece.
// A escolha da pessoa fica so neste aparelho (localStorage: financas.trava e financas.recentes); nada vai ao servidor.
// Com a trava ligada, a tela de bloqueio cobre o app ao abrir e ao voltar depois de FORA ms fora dele, e so sai com a
// confirmacao do Android (digital, rosto ou a senha do aparelho), pedida ao plugin Trava (Java). Cancelar, errar ou nao ter
// bloqueio de tela nao libera; a saida que sobra e "Sair da conta". A trava e so da tela: nao cifra nada (R23).
function criarBloqueio(env){
  const FORA = 60000, K_TRAVA = 'financas.trava', K_REC = 'financas.recentes';
  const cap = env.Capacitor;
  const nativo = !!(cap && typeof cap.isNativePlatform === 'function' && cap.isNativePlatform());
  const Trava = (nativo && cap.Plugins && cap.Plugins.Trava) || null;
  const raiz = nativo ? env.el('bloqueio') : null;
  const ler = k => { try { return env.armazenamento.getItem(k) === '1'; } catch (e) { return false; } };
  const gravar = (k, sim) => { if(sim) env.armazenamento.setItem(k, '1'); else env.armazenamento.removeItem(k); };
  let bloqueado = false, pedindo = null, saiuEm = null;

  const travaLigada = () => nativo && ler(K_TRAVA);
  const recentesLigado = () => nativo && ler(K_REC);

  // titulo vazio = sem aviso; soSair = o aparelho nao tem como confirmar
  function tela(titulo, frase, ruim, soSair){
    const t = env.el('bqT');
    t.textContent = titulo;
    t.classList.toggle('ruim', !!ruim);
    env.el('bqS').textContent = frase;
    raiz.classList.toggle('com-aviso', !!titulo);
    env.el('bqAbrir').hidden = !!soSair;
  }
  function bloquear(){
    bloqueado = true;
    tela('', 'Confirme que é você para abrir.');
    raiz.hidden = false;
    env.fundo(true);
  }
  function liberar(){
    bloqueado = false;
    raiz.hidden = true;
    env.fundo(false);
  }
  // Pergunta ao Android. Devolve 'ok', 'indisponivel', 'bloqueado' (muitas tentativas) ou 'cancelado'. Uma pergunta por vez.
  function confirmar(titulo){
    if(pedindo) return pedindo;
    const p = (async () => {
      if(!Trava) return 'indisponivel';
      try {
        const d = await Trava.disponivel();
        if(!d || !d.disponivel) return 'indisponivel';
        await Trava.confirmar({titulo});
        return 'ok';
      } catch (e) {
        const c = e && e.code;
        return c === 'indisponivel' || c === 'bloqueado' ? c : 'cancelado';
      }
    })().finally(() => { if(pedindo === p) pedindo = null; });
    pedindo = p;
    return p;
  }
  // Tenta liberar a tela bloqueada. So o 'ok' do Android libera.
  async function desbloquear(){
    if(!bloqueado) return;
    tela('', 'Confirme que é você para abrir.');
    const r = await confirmar('Confirme que é você');
    if(!bloqueado) return;
    if(r === 'ok') liberar();
    else if(r === 'indisponivel') tela('Este aparelho não tem mais digital, rosto nem senha de tela.', 'Sem isso não dá para confirmar que é você. Saia da conta e entre de novo.', false, true);
    else if(r === 'bloqueado') tela('Muitas tentativas.', 'Espere 30 segundos e tente de novo, ou use a senha do aparelho.', true);
    // cancelado: a tela fica como esta, com Desbloquear e Sair da conta
  }
  // O app saiu da frente ou voltou. Voltando depois de FORA ms com a trava ligada, bloqueia e pede de novo.
  function visibilidade(visivel){
    if(!nativo) return;
    if(!visivel){ if(saiuEm === null) saiuEm = env.agora(); return; }
    const fora = saiuEm === null ? 0 : env.agora() - saiuEm;
    saiuEm = null;
    if(bloqueado || pedindo || !travaLigada()) return; // a janela do Android tambem tira o app da frente: nao conta como saida
    if(fora >= FORA){ bloquear(); desbloquear(); }
  }

  // ---------- as duas chaves (Listas, Aparencia, Seguranca) ----------
  // Ligar e desligar a trava pedem a confirmacao na hora; sem ela, nada muda. Devolve {ligada, aviso, ruim}.
  async function mudarTrava(quer){
    const era = travaLigada();
    if(!nativo || quer === era) return {ligada: era, aviso: ''};
    const r = await confirmar(quer ? 'Confirme que é você para ligar a trava' : 'Confirme que é você para desligar a trava');
    if(r === 'ok'){ gravar(K_TRAVA, quer); return {ligada: quer, aviso: '', ok: true}; }
    if(r === 'indisponivel') return {ligada: era, aviso: 'Cadastre uma digital, um rosto ou uma senha de tela no Android para usar a trava.', ruim: true};
    if(r === 'bloqueado') return {ligada: era, aviso: 'Muitas tentativas. Tente de novo em 30 segundos.', ruim: true};
    return {ligada: era, aviso: 'Não confirmado: a trava continua ' + (era ? 'ligada' : 'desligada') + '.'};
  }
  async function mudarRecentes(quer){
    if(!nativo || !Trava) return recentesLigado();
    await Trava.protegerJanela({ligar: !!quer});
    gravar(K_REC, !!quer);
    return !!quer;
  }
  // Liga as chaves da tela (chamado quando o DOM esta pronto). No navegador a subsecao continua escondida.
  function ligarChaves(){
    const sec = env.el('apSeg');
    if(!nativo || !sec) return false;
    const sw = env.el('swTrava'), sr = env.el('swRecentes'), msg = env.el('segMsg');
    const aviso = (texto, ruim) => { msg.textContent = texto; msg.classList.toggle('ruim', !!ruim); };
    sec.hidden = false;
    sw.checked = travaLigada();
    sr.checked = recentesLigado();
    sw.onchange = async () => {
      const quer = sw.checked;
      sw.checked = travaLigada(); // so muda depois da confirmacao
      aviso('');
      const r = await mudarTrava(quer);
      sw.checked = r.ligada;
      aviso(r.aviso, r.ruim);
      if(r.ok) env.toast(r.ligada ? '✓ Trava ligada' : '✓ Trava desligada');
    };
    sr.onchange = async () => {
      const quer = sr.checked;
      aviso('');
      try {
        sr.checked = await mudarRecentes(quer);
        env.toast(sr.checked ? '✓ O app fica em branco nos apps recentes' : '✓ O app volta a aparecer nos apps recentes');
      } catch (e) { sr.checked = recentesLigado(); aviso('Não consegui mudar. Tente de novo.', true); }
    };
    return true;
  }

  // ---------- subida ----------
  function iniciar(){
    if(!nativo || !raiz) return false;
    env.el('bqAbrir').onclick = () => { desbloquear(); };
    env.el('bqSair').onclick = () => env.sair();
    if(recentesLigado() && Trava) Promise.resolve().then(() => Trava.protegerJanela({ligar: true})).catch(() => {});
    if(travaLigada()){ bloquear(); desbloquear(); }
    return true;
  }
  return {iniciar, visibilidade, ligarChaves, mudarTrava, mudarRecentes, desbloquear, bloqueado: () => bloqueado, travaLigada, recentesLigado};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarBloqueio;
const Bloqueio = (typeof window !== 'undefined') ? criarBloqueio({
  Capacitor: window.Capacitor,
  el: id=>document.getElementById(id),
  armazenamento: window.localStorage,
  agora: ()=>Date.now(),
  // enquanto bloqueado, nada do app recebe foco, toque nem leitor de tela (o conteudo fica por baixo, sem ser mostrado)
  fundo: sim=>{ document.documentElement.classList.toggle('bloqueado', sim); },
  toast: m=>{ if(typeof toast === 'function') toast(m); },
  sair: ()=>{ if(window.financasApi && window.financasApi.sair) window.financasApi.sair(); }
}) : null;
if(Bloqueio){
  Bloqueio.iniciar();
  document.addEventListener('visibilitychange', ()=>Bloqueio.visibilidade(document.visibilityState === 'visible'));
  // o Capacitor tambem avisa quando o app sai da frente e volta; os dois caminhos chegam ao mesmo lugar
  const App = window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.App;
  if(App && typeof App.addListener === 'function' && window.Capacitor.isNativePlatform && window.Capacitor.isNativePlatform()){
    App.addListener('pause', ()=>Bloqueio.visibilidade(false));
    App.addListener('resume', ()=>Bloqueio.visibilidade(true));
  }
  document.addEventListener('DOMContentLoaded', ()=>Bloqueio.ligarChaves());
}
