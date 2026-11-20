// Sessao (login): quem entrega o token do Cognito ao web/api.js. Spec: .ai/specs/app-android/spec.md, fase 2 (R9 a R18o).
// Duas implementacoes atras da mesma interface (token, recusado, sair, limpar, entrar), escolhidas por Capacitor.isNativePlatform():
//  - navegador e PWA: exatamente como sempre (Hosted UI na mesma aba, PKCE em sessionStorage, token em localStorage; qualquer recusa
//    do refresh e todo 401 levam ao login, e a pagina vai embora);
//  - app Android: o login abre na aba do sistema (Custom Tabs) e so no toque em "Entrar"; o retorno chega por <applicationId>://auth;
//    o refresh token e o PKCE ficam no plugin Cofre (Java, cifrado por chave do Android Keystore) e o ID token so em memoria.
//    Nada de token em localStorage ou sessionStorage. Rede e 5xx nao sao logout; 400 do refresh e.
// MFA (spec mfa-opcional, Fase 4): o login normal pede so "openid email" (R7). A secao do MFA pede um SEGUNDO login, com o escopo de autoatendimento
// (R8), cujo access token vive so em memoria (R9) e nunca substitui a sessao do app (R10). O PKCE pendente leva um tipo ("normal" ou "mfa") e o
// retorno do tipo "mfa" tem ramo proprio: nao grava refresh, nao chama salvarAuth e nao recarrega a pagina.
// Sem sessao, o navegador tambem mostra a tela de entrada (R33a) em vez de ir direto ao login do Cognito: o login so abre no toque em "Entrar".
// Este modulo nao escreve log nenhum (R18): token, code, verifier e a URL de retorno nunca sao impressos.
// Sair no app chama env.aoSair() (opcional) antes de apagar a sessao: e por onde web/js/push.js tira o aparelho do servidor (push, R19).
function criarSessao(env){
  const CFG = env.cfg;
  const b64url = buf => env.btoa(String.fromCharCode(...new Uint8Array(buf))).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  const base = 'https://' + CFG.loginDomain;
  const pedirToken = params => env.fetch(base + '/oauth2/token', {
    method: 'POST',
    headers: {'content-type': 'application/x-www-form-urlencoded'},
    body: new URLSearchParams(params)
  });
  async function novoPkce(){
    const v = b64url(env.crypto.getRandomValues(new Uint8Array(32)));
    const c = b64url(await env.crypto.subtle.digest('SHA-256', new TextEncoder().encode(v)));
    const state = b64url(env.crypto.getRandomValues(new Uint8Array(16)));
    return {v, c, state};
  }
  const ESCOPO_NORMAL = 'openid email';                                   // R7: o login normal NUNCA pede o escopo de autoatendimento
  const ESCOPO_MFA = 'openid email aws.cognito.signin.user.admin';        // R8: so o segundo login, ao abrir a secao do MFA
  const TTL_INTENCAO = 10 * 60 * 1000;                                    // R12: o login da secao do MFA vale 10 minutos
  // R8 [PENDENTE no dev]: o parametro que obriga a senha de novo no Managed login (prompt=login; a alternativa e max_age). Fica num lugar so.
  const REAUTENTICAR = {prompt: 'login'};
  const urlLogin = (redirect, p, mfa) => base + '/oauth2/authorize?' + new URLSearchParams(Object.assign({
    response_type: 'code', client_id: CFG.clientId, redirect_uri: redirect,
    scope: mfa ? ESCOPO_MFA : ESCOPO_NORMAL, code_challenge: p.c, code_challenge_method: 'S256', state: p.state
  }, mfa ? REAUTENTICAR : {}));
  // R10: o refresh do segundo login continua valido por 30 dias no Cognito; descartar no cliente nao basta, entao ele e revogado.
  // PROVADO no dev (2026-11-19, Verificacao passo 9): revogar o refresh REVOGA TAMBEM o access token emitido com ele ("Access Token has been revoked"),
  // entao a revogacao NAO pode ser logo depois da troca: o refresh fica so em memoria, junto do access token, e e revogado quando a secao termina
  // (descartarMfa), quando o access token vence ou quando a pagina e fechada.
  const revogar = refresh => env.fetch(base + '/oauth2/revoke', {
    method: 'POST',
    headers: {'content-type': 'application/x-www-form-urlencoded'},
    body: new URLSearchParams({token: refresh, client_id: CFG.clientId}),
    keepalive: true // a revogacao feita ao fechar a pagina ainda sai
  });
  const urlLogout = redirect => base + '/logout?' + new URLSearchParams({client_id: CFG.clientId, logout_uri: redirect});
  const erro = (msg, codigo) => { const e = new Error(msg); e.codigo = codigo; return e; };

  // ---------- o access token da secao do MFA: so em memoria, nos dois modos (R9) ----------
  let mfa = null;          // {acesso, refresh, exp}: nunca vai a localStorage, sessionStorage, Cofre nem a outro modulo
  const ouvintesMfa = [];
  // Descartar = esquecer os dois tokens e revogar o refresh (que revoga tambem o access token). A falha da revogacao nunca sobe.
  const descartarMfa = () => { const m = mfa; mfa = null; if(m && m.refresh) Promise.resolve().then(() => revogar(m.refresh)).catch(() => {}); };
  const tokenMfa = () => { if(mfa && mfa.exp <= env.agora()) descartarMfa(); return mfa ? mfa.acesso : null; };
  const aoMfa = f => { ouvintesMfa.push(f); if(tokenMfa()) { try { f(); } catch (e) {} } };
  // Troca o code do segundo login. Entrega SO o access token a secao (R10): o ID token e descartado; o refresh fica so em memoria e e revogado ao descartar (ver acima).
  async function trocarMfa(code, verifier, redirect){
    let r;
    try { r = await pedirToken({grant_type: 'authorization_code', client_id: CFG.clientId, code, redirect_uri: redirect, code_verifier: verifier}); }
    catch (e) { throw erro('Não consegui confirmar que é você.', 'mfa_falhou'); } // sem rede: o erro do fetch nunca sobe (nem seu texto)
    const t = r.ok ? await r.json().catch(() => null) : null;
    if(!t || !t.access_token) throw erro('Não consegui confirmar que é você.', 'mfa_falhou');
    mfa = {acesso: t.access_token, refresh: t.refresh_token || null, exp: env.agora() + ((t.expires_in || 3600) - 60) * 1000};
    for(const f of ouvintesMfa.slice()) { try { f(); } catch (e) {} }
    return mfa.acesso;
  }
  if(env.aoFecharPagina) env.aoFecharPagina(descartarMfa); // fechar ou sair da pagina no meio da secao: o refresh e revogado
  const aviso = tipo => { if(env.aviso) env.aviso(tipo); }; // 'nova' | 'fim' | 'falhou' | 'entrando' | 'cofre' (quem desenha e a Abertura)
  const semSessao = tipo => { aviso(tipo); return erro('Sem sessão: entre de novo.', 'sem_sessao'); };

  // ---------- navegador e PWA ----------
  function navegador(){
    const LS = 'financas.auth', PK = 'financas.pkce', ls = env.localStorage, ss = env.sessionStorage, loc = env.location;
    const redirect = () => loc.origin + '/';
    const lerAuth = () => { try { return JSON.parse(ls.getItem(LS)); } catch (e) { return null; } };
    const lerPkce = () => { try { return JSON.parse(ss.getItem(PK)); } catch (e) { return null; } };
    function salvarAuth(t){
      const antigo = lerAuth() || {};
      const a = {id: t.id_token, refresh: t.refresh_token || antigo.refresh, exp: env.agora() + (t.expires_in - 60) * 1000};
      ls.setItem(LS, JSON.stringify(a));
      return a;
    }
    // O toque em "Entrar" (e so ele): o login normal, na mesma aba. A pagina vai embora.
    async function irParaLogin(){
      const p = await novoPkce();
      ss.setItem(PK, JSON.stringify({v: p.v, state: p.state, tipo: 'normal'}));
      loc.href = urlLogin(redirect(), p, false);
      return new Promise(() => {}); // a pagina vai embora
    }
    // R8 e R12: o segundo login da secao do MFA. Guarda so a intencao (o PKCE com o tipo e o prazo de 10 minutos), nunca token.
    async function iniciarMfa(){
      const p = await novoPkce();
      ss.setItem(PK, JSON.stringify({v: p.v, state: p.state, tipo: 'mfa', ate: env.agora() + TTL_INTENCAO}));
      loc.href = urlLogin(redirect(), p, true);
      return new Promise(() => {}); // a pagina vai embora; ao voltar, aoMfa avisa que a secao pode abrir
    }
    // A sessao do app, sem login automatico: sem sessao a Abertura mostra a tela de entrada (R33a) e quem chama recebe "sem_sessao".
    async function tokenNormal(){
      const a = lerAuth();
      if(a && a.exp > env.agora()) return a.id;
      if(a && a.refresh){
        let r;
        try { r = await pedirToken({grant_type: 'refresh_token', client_id: CFG.clientId, refresh_token: a.refresh}); }
        catch (e) { throw erro('Sem conexão com a internet.', 'rede'); }
        if(r.ok) return salvarAuth(await r.json()).id;
        if(r.status === 400){ ls.removeItem(LS); throw semSessao('fim'); } // so a recusa do Cognito encerra a sessao
        throw erro('O login não respondeu. Tente de novo em instantes.', 'rede');
      }
      throw semSessao('nova');
    }
    async function token(){
      const p = new URLSearchParams(loc.search);
      const pend = lerPkce();
      if(p.get('code') || p.get('error')){ // o retorno do Cognito
        env.history.replaceState({}, '', loc.pathname);
        ss.removeItem(PK); // o PKCE sai ja, antes de qualquer espera: chamadas simultaneas nao o veem
        if(pend && pend.tipo === 'mfa'){ // R10: ramo proprio do segundo login; a sessao do app segue como estava
          if(p.get('code') && pend.state === p.get('state') && pend.ate > env.agora()){
            try { await trocarMfa(p.get('code'), pend.v, redirect()); } catch (e) { /* a secao nao abre; o app continua logado */ }
          }
          return tokenNormal();
        }
        if(!p.get('code') || !pend || pend.state !== p.get('state')) throw semSessao('falhou');
        const r = await pedirToken({grant_type: 'authorization_code', client_id: CFG.clientId, code: p.get('code'), redirect_uri: redirect(), code_verifier: pend.v});
        if(!r.ok) throw semSessao('falhou');
        return salvarAuth(await r.json()).id;
      }
      if(pend && pend.tipo === 'mfa') ss.removeItem(PK); // R12: voltou sem code (cancelou o login): a intencao e apagada e a secao nao reabre
      return tokenNormal();
    }
    const limpar = async () => { ls.removeItem(LS); };
    return {
      modo: 'navegador',
      token,
      recusado: async () => { ls.removeItem(LS); throw semSessao('fim'); }, // 401 da API: sem sessao, na tela de entrada (nao vai mais direto ao login)
      limpar,
      entrar: irParaLogin,
      iniciarMfa, tokenMfa, descartarMfa, aoMfa,
      sair: () => { descartarMfa(); ls.removeItem(LS); ss.removeItem(PK); loc.href = urlLogout(redirect()); }
    };
  }

  // ---------- app Android ----------
  function app(){
    const P = (env.Capacitor && env.Capacitor.Plugins) || {};
    const Cofre = P.Cofre, App = P.App, Browser = P.Browser;
    const parado = () => new Promise(() => {}); // depois de pedir a recarga, nada mais continua nesta pagina
    let mem = null;          // {id, exp}: o ID token so existe aqui
    let redirect = '';       // <applicationId>://auth, lido do pacote (App.getInfo), nao do config
    let pronto = null, renovando = null, retorno = null, entrando = false;
    let mfaEspera = null;    // {res, rej}: quem espera o segundo login terminar (R11: fechar a aba sem concluir rejeita com "cancelado")

    const ler = async nome => { const r = await Cofre.ler({nome}); return (r && r.valor) || null; };
    const gravar = (nome, valor) => Cofre.gravar({nome, valor});
    const apagar = nome => Cofre.apagar({nome});
    async function limpar(){
      mem = null;
      await apagar('refresh');
      await apagar('pkce');
    }
    // R18d: no pacote do app, faltar um plugin e erro na tela; nunca o fluxo do navegador.
    function conferir(){
      const falta = [Cofre, App, Browser].some(p => !p);
      if(falta){ aviso('cofre'); throw erro('O app não conseguiu usar o armazenamento seguro deste aparelho.', 'sem_cofre'); }
    }
    // O retorno do login (R11, R12, R18b): so vale o nosso endereco, com login pendente e o state guardado.
    async function tratarRetorno(url){
      if(typeof url !== 'string' || !redirect) return;
      const u = url.split('#')[0], i = u.indexOf('?');
      if((i < 0 ? u : u.slice(0, i)).replace(/\/$/, '') !== redirect) return; // outro esquema ou caminho: ignorado
      let pend = null;
      try { pend = JSON.parse(await ler('pkce') || 'null'); } catch (e) { pend = null; }
      if(!pend || !pend.v || !pend.state) return; // sem login pendente (retorno do logout, URL de lancamento repetida): ignorado
      await apagar('pkce'); // o verifier e o state somem no fim do login, com sucesso ou erro
      const q = new URLSearchParams(i < 0 ? '' : u.slice(i + 1));
      if(pend.tipo === 'mfa') return retornoMfa(q, pend); // R10: ramo proprio; nao grava refresh, nao recarrega e nao mexe na sessao do app
      if(!q.get('code') || !q.get('state') || q.get('state') !== pend.state) throw semSessao('falhou');
      aviso('entrando');
      let r;
      try { r = await pedirToken({grant_type: 'authorization_code', client_id: CFG.clientId, code: q.get('code'), redirect_uri: redirect, code_verifier: pend.v}); }
      catch (e) { throw semSessao('falhou'); }
      const t = r.ok ? await r.json().catch(() => null) : null;
      if(!t || !t.refresh_token || !t.id_token) throw semSessao('falhou');
      await gravar('refresh', t.refresh_token);
      env.recarregar(); // R18a: a pagina recomeca do zero com a sessao nova
      return parado();
    }
    // O retorno do segundo login (secao do MFA): troca o code, guarda SO o access token em memoria e avisa quem esperava. Erro aqui nunca vira
    // "sem sessao": o app continua logado e a secao so nao abre (R11).
    async function retornoMfa(q, pend){
      const espera = mfaEspera; mfaEspera = null;
      try {
        if(!q.get('code') || q.get('state') !== pend.state || !(pend.ate > env.agora())) throw erro('Não consegui confirmar que é você.', 'mfa_falhou');
        const t = await trocarMfa(q.get('code'), pend.v, redirect);
        if(espera) espera.res(t);
      } catch (e) { if(espera) espera.rej(e); }
    }
    const comRetorno = url => { // um retorno por vez; quem pede token nesse meio tempo espera por ele
      const p = tratarRetorno(url).finally(() => { if(retorno === p) retorno = null; });
      retorno = p;
      p.catch(() => {});
      return p;
    };
    function iniciar(){
      if(pronto) return pronto;
      pronto = (async () => {
        conferir();
        redirect = (await App.getInfo()).id + '://auth';
        App.addListener('appUrlOpen', ev => { comRetorno(ev && ev.url); });
        if(typeof Browser.addListener === 'function') Browser.addListener('browserFinished', () => { // R11: a aba do sistema fechou
          // espera um instante: no sucesso o retorno chega logo depois do fechamento; sem ele, a pessoa cancelou
          env.setTimeout(async () => {
            if(!mfaEspera || retorno) return;
            const espera = mfaEspera; mfaEspera = null;
            try { await apagar('pkce'); } catch (e) {}
            espera.rej(erro('Cancelado.', 'cancelado'));
          }, 1500);
        });
        const l = await App.getLaunchUrl(); // partida a frio pelo retorno: o Android matou o app com a aba de login aberta
        if(l && l.url) await comRetorno(l.url).catch(() => {});
      })();
      return pronto;
    }
    // R15, R18g, R18k: so a recusa (400) do Cognito encerra a sessao; rede, 5xx e o resto mantem.
    async function renovar(){
      let refresh;
      try { refresh = await ler('refresh'); }
      catch (e) { await limpar().catch(() => {}); throw semSessao('nova'); } // nao deu para ler ou decifrar: sem sessao, nunca um beco
      if(!refresh) throw semSessao('nova');
      let r;
      try { r = await pedirToken({grant_type: 'refresh_token', client_id: CFG.clientId, refresh_token: refresh}); }
      catch (e) { throw erro('Sem conexão com a internet.', 'rede'); }
      if(r.status === 400){ await limpar(); throw semSessao('fim'); }
      const t = r.ok ? await r.json().catch(() => null) : null;
      if(!t || !t.id_token) throw erro('O login não respondeu. Tente de novo em instantes.', 'rede');
      if(t.refresh_token) await gravar('refresh', t.refresh_token); // o Cognito nao costuma mandar um novo: o guardado continua valendo
      mem = {id: t.id_token, exp: env.agora() + (t.expires_in - 60) * 1000};
      return mem.id;
    }
    const umaRenovacao = () => { // R18c: um refresh por vez; as chamadas simultaneas esperam o mesmo resultado
      if(!renovando){
        const p = renovar().finally(() => { if(renovando === p) renovando = null; });
        renovando = p;
      }
      return renovando;
    };
    async function token(){
      await iniciar();
      if(retorno) await retorno; // login terminando agora: se der certo a pagina recarrega; se falhar, o erro e "sem sessao"
      if(mem && mem.exp > env.agora()) return mem.id;
      return umaRenovacao();
    }
    // R16: 401 da API apaga o ID token da memoria e tenta um refresh. Quem repete a chamada (uma vez so) e o api.js.
    async function recusado(usado){
      await iniciar();
      if(mem && usado && mem.id !== usado && mem.exp > env.agora()) return mem.id; // outra chamada ja renovou enquanto esta esperava a resposta
      mem = null;
      return umaRenovacao();
    }
    // O toque em "Entrar": guarda o PKCE no Cofre (sobrevive a morte do processo) e abre o login na aba do sistema.
    async function entrar(){
      if(entrando) return;
      entrando = true;
      try {
        await iniciar();
        const p = await novoPkce();
        await gravar('pkce', JSON.stringify({v: p.v, state: p.state, tipo: 'normal'}));
        await Browser.open({url: urlLogin(redirect, p)});
      } catch (e) {
        if(e && e.codigo !== 'sem_cofre') aviso('cofre');
        throw e;
      } finally { entrando = false; }
    }
    // R8, R11, R12: o segundo login da secao do MFA, na aba do sistema. Resolve com o access token (so em memoria); rejeita com "cancelado" se a aba
    // fechar sem concluir e com "mfa_falhou" se o retorno nao valer. A sessao do app nunca muda.
    async function iniciarMfa(){
      await iniciar();
      if(mfaEspera) throw erro('Já há um login em andamento.', 'ocupado');
      const p = await novoPkce();
      await gravar('pkce', JSON.stringify({v: p.v, state: p.state, tipo: 'mfa', ate: env.agora() + TTL_INTENCAO}));
      const espera = new Promise((res, rej) => { mfaEspera = {res, rej}; });
      espera.catch(() => {});
      try { await Browser.open({url: urlLogin(redirect, p, true)}); }
      catch (e) { mfaEspera = null; try { await apagar('pkce'); } catch (x) {} throw erro('Não consegui abrir o login.', 'mfa_falhou'); }
      return espera;
    }
    // R14, R18h, R18n: apaga e espera; so entao abre o logout. Se apagar falhar, erro na tela e o logout nao abre.
    async function sair(){
      descartarMfa(); // o access token da secao do MFA nao sobrevive ao Sair
      try {
        await iniciar();
        // Push (R19): ANTES de apagar a sessao, tira o aparelho do servidor e invalida o token no FCM. Falha aqui nunca impede o Sair.
        try { if(env.aoSair) await env.aoSair(); } catch (e) {}
        await limpar();
      } catch (e) { aviso('cofre'); throw e; }
      try { for(const k of ['financas.space', 'financas.trava', 'financas.recentes', 'financas.push', 'financas.telegram']) env.localStorage.removeItem(k); } catch (e) {} // o espaco escolhido e a trava (R14); tema e o olho ficam
      try { await Browser.open({url: urlLogout(redirect)}); } catch (e) {} // sem aba do sistema o Cognito nao e avisado; a sessao local ja foi apagada
      env.recarregar();
    }
    return {modo: 'app', token, recusado, limpar, entrar, sair, iniciarMfa, tokenMfa, descartarMfa, aoMfa};
  }

  const nativo = !!(env.Capacitor && typeof env.Capacitor.isNativePlatform === 'function' && env.Capacitor.isNativePlatform());
  return nativo ? app() : navegador();
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarSessao;
if(typeof window !== 'undefined' && window.FINANCAS_CONFIG){
  window.Sessao = criarSessao({
    cfg: window.FINANCAS_CONFIG,
    Capacitor: window.Capacitor,
    localStorage: window.localStorage,
    sessionStorage: window.sessionStorage,
    location: window.location,
    history: window.history,
    crypto: window.crypto,
    fetch: (u, o) => window.fetch(u, o),
    btoa: s => window.btoa(s),
    agora: () => Date.now(),
    recarregar: () => window.location.reload(),
    setTimeout: (f, ms) => window.setTimeout(f, ms),
    aoFecharPagina: f => { if(typeof window.addEventListener === 'function') window.addEventListener('pagehide', f); },
    aoSair: () => (typeof PushApp !== 'undefined' && PushApp ? PushApp.sair() : undefined), // web/js/push.js (so age no app)
    aviso: tipo => { if(typeof Abertura !== 'undefined' && Abertura && Abertura.sessao) Abertura.sessao(tipo); }
  });
}
