// Recuperar o acesso (sem login): quem perdeu o aplicativo autenticador usa UM código da lista de recuperação para desligar a verificação em duas etapas.
// Spec: .ai/specs/mfa-opcional/spec.md (R27 a R31, R33). Demo: .ai/demos/demo-listas-v10-mfa.html (tela "Recuperar o acesso").
// Regras deste módulo:
//  - chama só POST /mfa/recuperar, SEM token e sem cabeçalho de autorização (a rota é pública, R27); nunca inicia o login do Cognito (R33);
//  - o front NÃO decide nada sobre o código: não confere, não conta tentativas e não distingue os erros (o servidor responde igual a todos, R28);
//  - e-mail e código só existem nos campos e no corpo da chamada: nunca em armazenamento, log, URL nem texto de erro (R34);
//  - a porta de entrada é o botão "Perdi o autenticador" da tela de entrada (Abertura) e o endereço #recuperar, que abre esta tela direto.
function criarRecuperar(env){
  const TXT = {
    vazio: 'Digite o e-mail e um código da lista.',
    conferindo: 'Conferindo…',
    erro: 'Confira o e-mail e o código. Depois de várias tentativas erradas, aguarde alguns minutos.',
    rede: 'Não consegui falar com o servidor (sem conexão). Tente de novo.'
  };
  const $ = id => env.el(id);
  let enviando = false;
  const msg = (t, ruim) => { const e = $('mfRecMsg'); e.textContent = t || ''; e.classList.toggle('ruim', !!ruim); };
  const limparCampos = () => { $('mfRecEmail').value = ''; $('mfRecCod').value = ''; };

  function abrir(){
    $('mfRecForm').hidden = false; $('mfRecOk').hidden = true;
    $('mfRecEnviar').hidden = false; $('mfRecEntrar').hidden = true;
    msg('');
    $('mfRec').hidden = false;
    $('mfRecEmail').focus();
  }
  function fechar(){
    $('mfRec').hidden = true;
    limparCampos(); msg(''); enviando = false;
    if(env.limparEndereco) env.limparEndereco();
  }
  async function enviar(){
    if(enviando) return;
    const email = ($('mfRecEmail').value || '').trim(), codigo = ($('mfRecCod').value || '').trim();
    if(!email || !codigo){ msg(TXT.vazio, true); return; }
    enviando = true; msg(TXT.conferindo);
    let r;
    try { r = await env.fetch(env.url + '/mfa/recuperar', {method: 'POST', headers: {'content-type': 'application/json'}, body: JSON.stringify({email, codigo})}); }
    catch (e) { enviando = false; msg(TXT.rede, true); return; } // o texto do erro do fetch nunca sobe
    enviando = false;
    if(!r || !r.ok){ msg(TXT.erro, true); return; } // qualquer resposta que não seja o sucesso: a mesma frase (R28)
    limparCampos(); msg('');
    $('mfRecForm').hidden = true; $('mfRecOk').hidden = false;
    $('mfRecEnviar').hidden = true; $('mfRecEntrar').hidden = false;
  }
  function entrar(){ fechar(); env.entrar(); } // o login só abre aqui, no toque (R33a)

  function ligar(){
    if(!$('mfRec')) return false;
    $('mfRecVolta').onclick = fechar;
    $('mfRecEnviar').onclick = enviar;
    $('mfRecEntrar').onclick = entrar;
    $('mfRecCod').onkeydown = e => { if(e && e.key === 'Enter') enviar(); };
    const perdi = $('abPerdi'); if(perdi) perdi.onclick = abrir; // "Perdi o autenticador", na tela de entrada
    if(env.endereco() === '#recuperar') abrir(); // o endereço próprio abre a recuperação direto, sem iniciar o login
    return true;
  }
  return {ligar, abrir, fechar, enviar, entrar};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarRecuperar;
if(typeof window !== 'undefined' && window.FINANCAS_CONFIG && window.document){
  window.Recuperar = criarRecuperar({
    el: id => document.getElementById(id),
    url: String(window.FINANCAS_CONFIG.apiUrl || '').replace(/\/$/, ''),
    fetch: (u, o) => window.fetch(u, o),
    endereco: () => window.location.hash,
    limparEndereco: () => { if(window.location.hash === '#recuperar' && window.history && window.history.replaceState) window.history.replaceState({}, '', window.location.pathname + window.location.search); },
    entrar: () => { if(window.financasApi && window.financasApi.entrar) window.financasApi.entrar(); }
  });
  window.Recuperar.ligar();
}
