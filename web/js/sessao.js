// Sessao (login): quem entrega o token do Cognito ao web/api.js. Spec: .ai/specs/app-android/spec.md, fase 2 (R9 a R18o).
// Duas implementacoes atras da mesma interface (token, recusado, sair, limpar, entrar), escolhidas por Capacitor.isNativePlatform():
//  - navegador e PWA: exatamente como sempre (Hosted UI na mesma aba, PKCE em sessionStorage, token em localStorage; qualquer recusa
//    do refresh e todo 401 levam ao login, e a pagina vai embora);
//  - app Android: o login abre na aba do sistema (Custom Tabs) e so no toque em "Entrar"; o retorno chega por <applicationId>://auth;
//    o refresh token e o PKCE ficam no plugin Cofre (Java, cifrado por chave do Android Keystore) e o ID token so em memoria.
//    Nada de token em localStorage ou sessionStorage. Rede e 5xx nao sao logout; 400 do refresh e.
// Este modulo nao escreve log nenhum (R18): token, code, verifier e a URL de retorno nunca sao impressos.
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
  const urlLogin = (redirect, p) => base + '/oauth2/authorize?' + new URLSearchParams({
    response_type: 'code', client_id: CFG.clientId, redirect_uri: redirect,
    scope: 'openid email', code_challenge: p.c, code_challenge_method: 'S256', state: p.state
  });
  const urlLogout = redirect => base + '/logout?' + new URLSearchParams({client_id: CFG.clientId, logout_uri: redirect});
  const erro = (msg, codigo) => { const e = new Error(msg); e.codigo = codigo; return e; };

  // ---------- navegador e PWA: o fluxo de sempre ----------
  function navegador(){
    const LS = 'financas.auth', ls = env.localStorage, ss = env.sessionStorage, loc = env.location;
    const redirect = () => loc.origin + '/';
    const lerAuth = () => { try { return JSON.parse(ls.getItem(LS)); } catch (e) { return null; } };
    function salvarAuth(t){
      const antigo = lerAuth() || {};
      const a = {id: t.id_token, refresh: t.refresh_token || antigo.refresh, exp: env.agora() + (t.expires_in - 60) * 1000};
      ls.setItem(LS, JSON.stringify(a));
      return a;
    }
    async function irParaLogin(){
      const p = await novoPkce();
      ss.setItem('financas.pkce', JSON.stringify({v: p.v, state: p.state}));
      loc.href = urlLogin(redirect(), p);
      return new Promise(() => {}); // a pagina vai embora
    }
    async function token(){
      const p = new URLSearchParams(loc.search);
      if(p.get('code')){
        const s = JSON.parse(ss.getItem('financas.pkce') || 'null');
        env.history.replaceState({}, '', loc.pathname);
        if(!s || s.state !== p.get('state')) return irParaLogin();
        const r = await pedirToken({grant_type: 'authorization_code', client_id: CFG.clientId, code: p.get('code'), redirect_uri: redirect(), code_verifier: s.v});
        if(!r.ok) return irParaLogin();
        return salvarAuth(await r.json()).id;
      }
      const a = lerAuth();
      if(a && a.exp > env.agora()) return a.id;
      if(a && a.refresh){
        const r = await pedirToken({grant_type: 'refresh_token', client_id: CFG.clientId, refresh_token: a.refresh});
        if(r.ok) return salvarAuth(await r.json()).id;
      }
      return irParaLogin();
    }
    const limpar = async () => { ls.removeItem(LS); };
    return {
      modo: 'navegador',
      token,
      recusado: () => { ls.removeItem(LS); return irParaLogin(); }, // 401 da API: login, como sempre
      limpar,
      entrar: irParaLogin,
      sair: () => { ls.removeItem(LS); loc.href = urlLogout(redirect()); }
    };
  }

  // ---------- app Android ----------
  function app(){
    const P = (env.Capacitor && env.Capacitor.Plugins) || {};
    const Cofre = P.Cofre, App = P.App, Browser = P.Browser;
    const aviso = (tipo) => { if(env.aviso) env.aviso(tipo); }; // 'nova' | 'fim' | 'falhou' | 'entrando' | 'cofre' (quem desenha e a Abertura)
    const semSessao = (tipo) => { aviso(tipo); return erro('Sem sessão: entre de novo.', 'sem_sessao'); };
    const parado = () => new Promise(() => {}); // depois de pedir a recarga, nada mais continua nesta pagina
    let mem = null;          // {id, exp}: o ID token so existe aqui
    let redirect = '';       // <applicationId>://auth, lido do pacote (App.getInfo), nao do config
    let pronto = null, renovando = null, retorno = null, entrando = false;

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
        await gravar('pkce', JSON.stringify({v: p.v, state: p.state}));
        await Browser.open({url: urlLogin(redirect, p)});
      } catch (e) {
        if(e && e.codigo !== 'sem_cofre') aviso('cofre');
        throw e;
      } finally { entrando = false; }
    }
    // R14, R18h, R18n: apaga e espera; so entao abre o logout. Se apagar falhar, erro na tela e o logout nao abre.
    async function sair(){
      try {
        await iniciar();
        await limpar();
      } catch (e) { aviso('cofre'); throw e; }
      try { env.localStorage.removeItem('financas.space'); } catch (e) {}
      try { await Browser.open({url: urlLogout(redirect)}); } catch (e) {} // sem aba do sistema o Cognito nao e avisado; a sessao local ja foi apagada
      env.recarregar();
    }
    return {modo: 'app', token, recusado, limpar, entrar, sair};
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
    aviso: tipo => { if(typeof Abertura !== 'undefined' && Abertura && Abertura.sessao) Abertura.sessao(tipo); }
  });
}
