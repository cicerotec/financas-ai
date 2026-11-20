// Seguranca da conta: a verificacao em duas etapas (MFA) em Listas. Spec: .ai/specs/mfa-opcional/spec.md (R7 a R24, R37). Demo aprovada: .ai/demos/demo-listas-v10-mfa.html.
// Padrao: .ai/padrao-de-telas.md ("Seguranca da conta"). O nome na tela e "Verificacao em duas etapas": a sigla MFA nunca aparece.
//
// O que este modulo faz, e o que NAO faz:
//  - O ESTADO (ligado, e se ha lista de codigos ativa) vem do servidor, por GET /me/mfa, quando Listas e aberta (R37, R23); o front so exibe, nunca guarda nem infere.
//  - Ligar e desligar pedem um SEGUNDO login (web/js/sessao.js), que entrega o access token so em memoria (R8, R9). Gerar uma lista nova nao pede: usa o ID token.
//  - Ligar segue a ordem do R16: associar -> QR e chave -> codigo de 6 digitos -> gerar a lista (pendente) -> "Guardei a lista" -> confirmar (200) -> so entao ligar.
//  - A chave do TOTP, o QR e a lista de codigos so existem em memoria e na tela, e somem ao fechar (R19, R20). Nada vai a armazenamento; so a INTENCAO do segundo login
//    (a acao e o prazo de 10 minutos, sem token) fica em sessionStorage, no navegador, que sai da pagina para o login (R12).
//  - O modulo nao decide regra nenhuma de negocio e nao escreve log (R13).
function criarSegurancaConta(env){
  const K_ACAO = 'financas.mfaAcao', TTL = 10 * 60 * 1000;
  const FRASE_LISTA = 'SE PERDER A LISTA, PERDE O ACESSO.';
  const TXT = {
    apoioOff: 'Pede um código do aplicativo autenticador, além da senha, ao entrar.',
    apoioOn: 'Ligada. O login pede o código do aplicativo autenticador depois da senha.',
    apoioLendo: 'Conferindo…',
    apoioErro: 'Não consegui conferir agora.',
    notaOff: 'Sem isso, só a senha protege a conta. Quem não ligar continua entrando só com e-mail e senha.',
    notaOn: 'Se você perder o autenticador, use um código da lista na tela "Recuperar o acesso", ao lado do Entrar.',
    listaSim: d => 'Há uma lista ativa' + (d ? ', gerada em ' + d : '') + '. Cada código vale uma vez.',
    listaNao: 'Não há lista ativa. Gere uma nova.',
    nada: 'Nada foi alterado.',
    venceu: 'A confirmação venceu. Tente de novo.',
    confirmouErro: 'Não consegui confirmar que é você. Nada foi alterado.'
  };
  const ERRO = { // R18: na propria janela, sem detalhe tecnico
    codigo_errado: 'Código incorreto. Confira o aplicativo e tente de novo; ele muda a cada 30 segundos.',
    tentativas: 'Muitas tentativas. Aguarde alguns minutos e tente de novo.',
    rede: 'Não consegui conferir agora (sem conexão). Nada foi alterado. Tente de novo.',
    permissao: 'Não foi possível continuar. Feche esta tela e tente outra vez.',
    servico: 'Não foi possível concluir agora. Tente de novo em instantes.'
  };
  const semRede = e => !e || (e.code === undefined && e.codigo === undefined) || e.codigo === 'rede';
  const codigoDe = e => (e && e.codigo) || (semRede(e) ? 'rede' : 'servico');

  let estado = null;        // o que o servidor disse: {ligado, lista:{ativa, geradaEm}}; so para desenhar
  let ocupado = false, lendo = false, fluxo = null; // fluxo: 'ligar' | 'nova' enquanto a lista esta na tela
  let lista = null;         // {listaId, codigos}: so em memoria, so enquanto a tela da lista esta aberta
  let preparando = false, enviando = false;

  const $ = id => env.el(id);
  const dataBR = iso => { // dd/mm/aaaa
    const d = new Date(iso);
    return isNaN(d) ? '' : String(d.getDate()).padStart(2, '0') + '/' + String(d.getMonth() + 1).padStart(2, '0') + '/' + d.getFullYear();
  };
  function msg(id, texto, ruim, ok){
    const e = $(id); if(!e) return;
    e.textContent = texto || '';
    e.classList.toggle('ruim', !!ruim);
    e.classList.toggle('ok', !!ok);
  }

  // ---------- o cartao em Listas ----------
  function desenhar(){
    const sw = $('swMfa'); if(!sw) return;
    const ligado = !!(estado && estado.ligado);
    sw.checked = ligado;
    sw.disabled = ocupado || lendo || !estado;
    $('mfaApoio').textContent = lendo ? TXT.apoioLendo : !estado ? TXT.apoioErro : ligado ? TXT.apoioOn : TXT.apoioOff;
    $('mfaNota').textContent = !estado ? '' : ligado ? TXT.notaOn : TXT.notaOff;
    const l = estado && estado.lista;
    $('mfaListaBloco').hidden = !(estado && ligado);
    $('mfaListaTxt').textContent = l && l.ativa ? TXT.listaSim(l.geradaEm ? dataBR(l.geradaEm) : '') : TXT.listaNao;
    $('mfaAcoes').hidden = !(estado && ligado);
    $('mfaDeNovo').hidden = !!estado || lendo;
  }
  async function ler(){
    if(lendo) return;
    lendo = true; msg('mfaMsg', ''); desenhar();
    try { estado = await env.api.estado(); }
    catch (e) { estado = null; msg('mfaMsg', 'Não consegui ler o estado agora. Toque em "Tentar de novo".', true); }
    lendo = false; desenhar();
  }
  // Chamado quando Listas e aberta (e depois de cada mudanca): e a unica leitura do cartao; a subida do app nao le nada (R23).
  const entrouEmListas = () => ler();

  // ---------- janelas de confirmacao (as do app) ----------
  const confirmarLigar = () => env.confirmar('Para ligar a verificação em duas etapas, o app abre o login do Cognito e pede a sua senha de novo. É só uma vez, para esta mudança.', 'Continuar', 'Agora não', 'Confirmar que é você');
  const confirmarDesligar = () => env.confirmar('O login volta a pedir só e-mail e senha. Por segurança, o app abre o login do Cognito e pede a senha e o código de novo.', 'Continuar', 'Cancelar', 'Desligar a verificação em duas etapas?');
  const confirmarNova = () => env.confirmar('A lista que você já tem continua valendo até você guardar a nova. Quando guardar, a antiga deixa de valer.', 'Gerar nova lista', 'Cancelar', 'Gerar uma nova lista?');
  const confirmarSair = () => env.confirmar(fluxo === 'nova' ? 'Esta lista nova será descartada e a que você já tem continua valendo.' : 'A verificação em duas etapas não será ligada, e esta lista será descartada. Dá para começar de novo quando quiser.', 'Sair sem guardar', 'Ficar', 'Sair sem guardar a lista?');

  // ---------- o segundo login (R8 a R12) ----------
  const lerAcao = () => { try { const o = JSON.parse(env.armazenamento.getItem(K_ACAO) || 'null'); return o && (o.acao === 'ligar' || o.acao === 'desligar') && o.ate > env.agora() ? o.acao : null; } catch (e) { return null; } };
  const limparAcao = () => { try { env.armazenamento.removeItem(K_ACAO); } catch (e) { /* sem armazenamento */ } };
  async function pedirLogin(acao){
    ocupado = true; desenhar(); msg('mfaMsg', '');
    try {
      if(env.sessao.modo === 'navegador'){ // a pagina vai embora; ao voltar, aoMfa retoma a acao
        try { env.armazenamento.setItem(K_ACAO, JSON.stringify({acao, ate: env.agora() + TTL})); } catch (e) { /* sem armazenamento: o retorno so nao retoma sozinho */ }
        await env.sessao.iniciarMfa(); // nao volta
        return;
      }
      await env.sessao.iniciarMfa(); // app: a promessa volta com o access token
    } catch (e) {
      ocupado = false; desenhar();
      if(e && e.codigo === 'cancelado') env.toast(TXT.nada); else env.toast(TXT.confirmouErro, true);
      return;
    }
    ocupado = false;
    continuar(acao);
  }
  // O retorno do segundo login no navegador: a pagina recarregou, o access token ja esta em memoria e a acao foi guardada antes de sair.
  function retomar(){
    const acao = lerAcao(); limparAcao();
    if(!acao || !env.sessao.tokenMfa()) return;
    env.irParaListas();
    ler().then(() => continuar(acao));
  }
  function continuar(acao){ if(acao === 'ligar') abrirLigar(); else desligar(); }

  async function trocar(quer){
    $('swMfa').checked = !!(estado && estado.ligado); // so muda depois do servidor e do Cognito
    if(ocupado || !estado) return;
    if(quer){ if(await confirmarLigar()) pedirLogin('ligar'); }
    else if(await confirmarDesligar()) pedirLogin('desligar');
  }

  // ---------- ligar: a tela do QR code e do codigo ----------
  function fecharLigar(){
    $('mfLig').hidden = true;
    $('mfQr').innerHTML = ''; $('mfChave').textContent = ''; $('mfCodigo').value = ''; msg('mfLigMsg', '');
    preparando = false; enviando = false;
  }
  function descartarAcesso(){ if(env.sessao.descartarMfa) env.sessao.descartarMfa(); }
  async function abrirLigar(){
    const token = env.sessao.tokenMfa();
    if(!token){ env.toast(TXT.venceu, true); return; }
    $('mfLig').hidden = false; $('mfQr').innerHTML = ''; $('mfChave').textContent = ''; $('mfCodigo').value = '';
    msg('mfLigMsg', 'Preparando…');
    preparando = true; enviando = false;
    try {
      const [mfa, qr] = await Promise.all([env.mfa(), env.qr()]);
      const { chave } = await mfa.associar(token);
      if($('mfLig').hidden) return; // fechou enquanto esperava
      $('mfQr').innerHTML = qr.svg(mfa.uriOtpauth(chave, env.nomeDoApp(), env.nomeDaConta())); // o QR e a chave, so neste aparelho e so nesta tela (R19)
      $('mfChave').textContent = mfa.formatarChave(chave);
      msg('mfLigMsg', '');
      $('mfCodigo').focus();
    } catch (e) { msg('mfLigMsg', ERRO[codigoDe(e)] || ERRO.servico, true); }
    preparando = false;
  }
  async function verificar(){
    if(preparando || enviando) return;
    const c = ($('mfCodigo').value || '').trim();
    if(!/^\d{6}$/.test(c.replace(/\s+/g, ''))){ msg('mfLigMsg', 'Digite os 6 números que o autenticador mostra.', true); return; }
    enviando = true; msg('mfLigMsg', 'Conferindo…');
    try {
      const mfa = await env.mfa();
      await mfa.verificar(env.sessao.tokenMfa(), c, env.nomeDoApp());
    } catch (e) { enviando = false; msg('mfLigMsg', ERRO[codigoDe(e)] || ERRO.servico, true); return; }
    msg('mfLigMsg', 'Código aceito ✓', false, true);
    enviando = false;
    gerarLista('ligar');
  }

  // ---------- a lista de codigos (a unica vez que ela aparece) ----------
  function fecharLista(){
    $('mfLista').hidden = true;
    $('mfCodigos').innerHTML = ''; msg('mfListaMsg', '');
    lista = null; fluxo = null; enviando = false;
    $('mfEntendi').setAttribute('aria-checked', 'false');
  }
  async function gerarLista(de){
    fluxo = de; enviando = true;
    try { lista = await env.api.gerar(); }
    catch (e) {
      enviando = false; fluxo = null;
      if(de === 'ligar') msg('mfLigMsg', ERRO[codigoDe(e)] || ERRO.servico, true); else env.toast(semRede(e) ? 'Não consegui gerar a lista (sem conexão). Nada foi alterado.' : TXT.confirmouErro, true);
      return;
    }
    enviando = false;
    fecharLigar(); // R19: o QR e a chave somem ao sair desta tela
    $('mfListaCtx').textContent = de === 'ligar' ? 'Passo 2 de 2: guarde a lista' : 'Lista nova: a atual continua valendo até você guardar esta';
    $('mfCodigos').innerHTML = '';
    (lista.codigos || []).forEach(c => { const li = env.documento.createElement('li'); li.textContent = c; $('mfCodigos').appendChild(li); });
    $('mfEntendi').setAttribute('aria-checked', 'false');
    $('mfGuardei').textContent = 'Guardei a lista';
    msg('mfListaMsg', '');
    $('mfLista').hidden = false;
  }
  const textoDaLista = () => ['Finanças: códigos de recuperação', 'Gerada em ' + dataBR(env.agoraISO()), ''].concat(lista.codigos || [], ['', FRASE_LISTA]).join('\n'); // sem endereco: uma lista achada nao pode dizer onde usar os codigos (decisao do dono, 2026-11-20)
  async function copiar(){
    if(!lista) return;
    try { await env.copiar(textoDaLista()); env.toast('✓ Lista copiada. Cole num lugar seguro.'); }
    catch (e) { env.toast('Não consegui copiar. Anote os códigos à mão ou use Imprimir.', true); }
  }
  function imprimir(){ if(lista) env.imprimir(textoDaLista()); }
  async function guardei(){
    if(enviando || !lista) return;
    if($('mfGuardei').textContent === 'Gerar nova lista'){ const de = fluxo; fecharLista(); gerarLista(de === 'nova' ? 'nova' : 'ligar'); return; }
    if($('mfEntendi').getAttribute('aria-checked') !== 'true'){ msg('mfListaMsg', 'Marque que guardou a lista e entendeu o aviso para continuar.', true); return; }
    enviando = true; msg('mfListaMsg', 'Confirmando a lista…');
    try { await env.api.confirmar(lista.listaId); }
    catch (e) {
      enviando = false;
      if(e && e.code === 409){ $('mfGuardei').textContent = 'Gerar nova lista'; msg('mfListaMsg', 'Esta lista foi substituída por outra (ou venceu). Ela não vale mais. Toque abaixo para gerar uma nova; a lista que você já tinha continua valendo.', true); return; }
      msg('mfListaMsg', semRede(e) ? 'Não consegui confirmar (sem conexão). A lista continua aberta: toque de novo em "Guardei a lista".' : ERRO.servico, true);
      return;
    }
    if(fluxo === 'nova'){ enviando = false; fecharLista(); descartarAcesso(); await ler(); env.aviso('Lista nova guardada', 'A lista anterior deixou de valer.'); return; }
    msg('mfListaMsg', 'Lista guardada ✓ Ligando a verificação…', false, true);
    try { const mfa = await env.mfa(); await mfa.ligar(env.sessao.tokenMfa()); }
    catch (e) { enviando = false; msg('mfListaMsg', 'A lista foi guardada, mas não consegui ligar a verificação. Toque de novo em "Guardei a lista".', true); return; }
    enviando = false; fecharLista(); descartarAcesso(); await ler();
    env.aviso('Verificação em duas etapas ligada', 'A partir de agora, entrar pede a senha e o código do autenticador.');
  }
  async function sairDaLista(){ if(!lista) return; if(await confirmarSair()){ const de = fluxo; fecharLista(); if(de === 'ligar') descartarAcesso(); env.toast(TXT.nada); } }
  function sairDoLigar(){ fecharLigar(); descartarAcesso(); env.toast(TXT.nada); }

  // ---------- desligar (R17) ----------
  async function desligar(){
    ocupado = true; desenhar(); msg('mfaMsg', 'Desligando…');
    try { const mfa = await env.mfa(); await mfa.desligar(env.sessao.tokenMfa()); }
    catch (e) { ocupado = false; descartarAcesso(); desenhar(); msg('mfaMsg', ERRO[codigoDe(e)] === ERRO.codigo_errado ? ERRO.servico : (codigoDe(e) === 'rede' ? 'Não consegui desligar agora (sem conexão). Nada foi alterado. Tente de novo.' : codigoDe(e) === 'permissao' ? 'Não foi possível continuar. Entre de novo e tente outra vez.' : ERRO[codigoDe(e)] || ERRO.servico), true); return; }
    ocupado = false; descartarAcesso(); msg('mfaMsg', '');
    await ler();
    env.aviso('Verificação em duas etapas desligada', 'Entrar volta a pedir só e-mail e senha. Ao ligar de novo você recebe uma lista nova.');
  }
  async function gerarNova(){ if(ocupado || enviando || !estado) return; if(await confirmarNova()) gerarLista('nova'); }

  // ---------- ligacao com a pagina ----------
  function ligar(){
    if(!$('mfaCard') || !$('swMfa')) return false;
    $('swMfa').onchange = () => trocar($('swMfa').checked);
    $('btnMfaNova').onclick = gerarNova;
    $('mfaDeNovo').onclick = ler;
    $('mfVerificar').onclick = verificar;
    $('mfCodigo').onkeydown = e => { if(e && e.key === 'Enter') verificar(); };
    $('mfLigVolta').onclick = sairDoLigar;
    $('mfListaVolta').onclick = sairDaLista;
    $('mfGuardei').onclick = guardei;
    $('mfCopiar').onclick = copiar;
    $('mfImprimir').onclick = imprimir;
    const ck = $('mfEntendi');
    const alterna = () => { const v = ck.getAttribute('aria-checked') === 'true'; ck.setAttribute('aria-checked', v ? 'false' : 'true'); if(!v) msg('mfListaMsg', ''); };
    ck.onclick = alterna; ck.onkeydown = e => { if(e && (e.key === ' ' || e.key === 'Enter')){ if(e.preventDefault) e.preventDefault(); alterna(); } };
    env.sessao.aoMfa(retomar); // o retorno do segundo login no navegador
    desenhar();
    return true;
  }
  return {ligar, entrouEmListas, trocar, verificar, guardei, gerarNova, sairDaLista, sairDoLigar, ler, estadoAtual: () => estado, listaAberta: () => !!lista};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarSegurancaConta;
