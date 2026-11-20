// As chamadas do MFA ao Cognito (spec mfa-opcional, R14 a R19): GetUser, AssociateSoftwareToken, VerifySoftwareToken e SetUserMFAPreference,
// direto do navegador ao endpoint do Cognito, SEM passar pela nossa Lambda. Sem regra de negocio: o front so exibe e devolve o que o Cognito responde.
//  - o access token vai no CORPO, nunca na URL nem em cabecalho (R14); o unico cabecalho alem do tipo e o X-Amz-Target, que a API do Cognito exige;
//  - o estado do MFA e o que o GetUser devolve (so UserMFASettingList e PreferredMfaSetting): nada e inferido, guardado nem exibido alem disso, e
//    nenhum outro atributo (como o e-mail) sai daqui (R15);
//  - a chave do TOTP e o texto otpauth ficam so em memoria de quem chama: este modulo nao usa armazenamento, area de transferencia nem log (R19);
//  - erro vira um codigo curto (codigo_errado, tentativas, permissao, rede, servico), nunca o texto nem o corpo do Cognito (R18).
// Os testes estao em web/mfa-cognito.test.js.
function criarMfaCognito(env){
  const CFG = env.cfg || {};
  const erro = (codigo, msg) => { const e = new Error(msg); e.codigo = codigo; return e; };
  const MENSAGENS = {
    codigo_errado: 'Código incorreto. Confira o aplicativo e tente de novo.',
    tentativas: 'Muitas tentativas. Aguarde alguns minutos e tente de novo.',
    permissao: 'Não foi possível continuar. Confirme que é você e tente de novo.',
    rede: 'Sem conexão com a internet. Nada foi alterado.',
    servico: 'Não foi possível concluir agora. Tente de novo em instantes.',
    config: 'O app não conseguiu falar com o login.'
  };
  const falha = codigo => erro(codigo, MENSAGENS[codigo] || MENSAGENS.servico);

  // a regiao vem do endereco do login (<prefixo>.auth.<regiao>.amazoncognito.com) ou de cfg.regiao
  function regiao(){
    const m = /\.auth\.([a-z0-9-]+)\.amazoncognito\.com$/.exec(String(CFG.loginDomain || ''));
    return (m && m[1]) || (/^[a-z0-9-]+$/.test(String(CFG.regiao || '')) ? CFG.regiao : '');
  }
  // o nome do erro do Cognito ("CodeMismatchException" ou "com.amazon...#CodeMismatchException") vira um codigo nosso
  function codigoDe(status, nomeBruto){
    const nome = String(nomeBruto || '').split('#').pop();
    if(nome === 'CodeMismatchException' || nome === 'EnableSoftwareTokenMFAException' || nome === 'ExpiredCodeException') return 'codigo_errado';
    if(nome === 'LimitExceededException' || nome === 'TooManyRequestsException' || nome === 'TooManyFailedAttemptsException') return 'tentativas';
    if(nome === 'NotAuthorizedException' || nome === 'ForbiddenException' || nome === 'AccessDeniedException' || nome === 'InvalidParameterException' || status === 401 || status === 403) return 'permissao';
    return 'servico';
  }
  async function chamar(alvo, corpo){
    const r = regiao();
    if(!r) throw falha('config');
    let resp;
    try {
      resp = await env.fetch('https://cognito-idp.' + r + '.amazonaws.com/', {
        method: 'POST',
        headers: {'content-type': 'application/x-amz-json-1.1', 'x-amz-target': 'AWSCognitoIdentityProviderService.' + alvo},
        body: JSON.stringify(corpo)
      });
    } catch (e) { throw falha('rede'); } // o erro do fetch pode levar a URL: o texto original nunca sobe
    let j = null;
    try { j = await resp.json(); } catch (e) { j = null; }
    if(!resp.ok) throw falha(codigoDe(resp.status, j && (j.__type || j.code)));
    return j || {};
  }
  const tokenValido = t => typeof t === 'string' && t.length > 0;

  // R15: so o que o GetUser devolve sobre o MFA; nenhum outro campo (e-mail, atributos) e repassado
  async function estado(token){
    if(!tokenValido(token)) throw falha('permissao');
    const j = await chamar('GetUser', {AccessToken: token});
    return {lista: Array.isArray(j.UserMFASettingList) ? j.UserMFASettingList.filter(x => typeof x === 'string') : [], preferido: typeof j.PreferredMfaSetting === 'string' ? j.PreferredMfaSetting : ''};
  }
  // so para exibir a chave do estado na tela (a chave "Verificacao em duas etapas"): o TOTP esta na lista do Cognito?
  const totpNaLista = est => !!est && Array.isArray(est.lista) && est.lista.includes('SOFTWARE_TOKEN_MFA');

  // R16 passo 1: o Cognito cria o segredo do TOTP. A chave volta so em memoria de quem chama (R19).
  async function associar(token){
    if(!tokenValido(token)) throw falha('permissao');
    const j = await chamar('AssociateSoftwareToken', {AccessToken: token});
    const chave = String(j.SecretCode || '').replace(/\s+/g, '').toUpperCase();
    if(!/^[A-Z2-7]{16,128}$/.test(chave)) throw falha('servico');
    return {chave};
  }
  // R16 passo 3: confere o primeiro codigo. So SUCCESS vale; qualquer outro resultado e codigo errado, e nada e ligado.
  async function verificar(token, codigo, nomeDoAparelho){
    if(!tokenValido(token)) throw falha('permissao');
    const c = String(codigo || '').replace(/\s+/g, '');
    if(!/^\d{6}$/.test(c)) throw falha('codigo_errado'); // 6 digitos: nem chama o Cognito
    const j = await chamar('VerifySoftwareToken', {AccessToken: token, UserCode: c, FriendlyDeviceName: nomeDoAparelho || 'Finanças'});
    if(j.Status !== 'SUCCESS') throw falha('codigo_errado');
    return true;
  }
  // R16 passo 7 e R17: liga ou desliga. So depois do 200 do confirmar da lista (quem chama garante a ordem).
  async function definir(token, ligado){
    if(!tokenValido(token)) throw falha('permissao');
    await chamar('SetUserMFAPreference', {AccessToken: token, SoftwareTokenMfaSettings: {Enabled: !!ligado, PreferredMfa: !!ligado}});
    return true;
  }
  const ligar = token => definir(token, true);
  const desligar = token => definir(token, false);

  // O texto do QR code: o nome do app, o nome de exibicao da pessoa (o autenticador mostra "App: nome") e a chave. NUNCA o e-mail: o nome
  // de exibicao e o apelido ou a parte do e-mail antes do @, e qualquer coisa a partir de um @ e cortada aqui (R15). Quem desenha e web/js/qr.js, no aparelho.
  const contaDoQr = c => String(c || '').split('@')[0].replace(/[:%\s]+/g, ' ').trim().slice(0, 40);
  function uriOtpauth(chave, nomeDoApp, conta){
    const k = String(chave || '').replace(/\s+/g, '').toUpperCase();
    if(!/^[A-Z2-7]{16,128}$/.test(k)) throw falha('servico');
    const nome = encodeURIComponent(String(nomeDoApp || 'Finanças'));
    const cta = contaDoQr(conta);
    return 'otpauth://totp/' + nome + (cta ? ':' + encodeURIComponent(cta) : '') + '?secret=' + k + '&issuer=' + nome;
  }
  // a chave em grupos de 4, para ler e digitar
  const formatarChave = chave => String(chave || '').replace(/\s+/g, '').replace(/(.{4})/g, '$1 ').trim();

  return {estado, totpNaLista, associar, verificar, ligar, desligar, uriOtpauth, formatarChave, MENSAGENS};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarMfaCognito;
if(typeof window !== 'undefined' && window.FINANCAS_CONFIG){
  window.MfaCognito = criarMfaCognito({cfg: window.FINANCAS_CONFIG, fetch: (u, o) => window.fetch(u, o)});
}
