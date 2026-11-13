// Casca Android (Capacitor, item 54): liga o botao voltar fisico a pilha de js/voltar.js. No navegador e na PWA nao faz nada.
// Sem bundler: o Capacitor injeta window.Capacitor no app, e os plugins vem de Capacitor.Plugins.
// Regra (spec app-android R6): o voltar fecha a camada de cima ou troca de aba como o historico do navegador; na raiz (sem entrada
// anterior no historico da WebView) minimiza o app sem perder o estado, em vez de fechar a atividade.
function criarNativo(env){
  function iniciar(){
    const cap = env.Capacitor;
    if(!cap || typeof cap.isNativePlatform !== 'function' || !cap.isNativePlatform()) return false;
    const App = cap.Plugins && cap.Plugins.App;
    if(!App || typeof App.addListener !== 'function') return false;
    App.addListener('backButton', ev => {
      if(env.bloqueado && env.bloqueado()){ if(typeof App.minimizeApp === 'function') App.minimizeApp(); return; } // com a tela de bloqueio na frente, o voltar so tira o app da frente
      if(ev && ev.canGoBack) env.history.back();
      else if(typeof App.minimizeApp === 'function') App.minimizeApp();
    });
    return true;
  }
  return {iniciar};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarNativo;
if(typeof window !== 'undefined') criarNativo({Capacitor: window.Capacitor, history: window.history, bloqueado: ()=>typeof Bloqueio !== 'undefined' && !!Bloqueio && Bloqueio.bloqueado()}).iniciar();
