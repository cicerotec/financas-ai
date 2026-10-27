(function(){
  // Tema claro, escuro ou automatico (segue o aparelho). A escolha da pessoa e guardada na conta (campo `tema` da aparencia); aqui fica so a
  // aplicacao e uma copia no aparelho, lida no carregamento da pagina para nao piscar o tema errado antes da API responder.
  var CHAVE = 'financas.tema';
  function aplicar(t){
    var r = document.documentElement;
    if(t === 'claro') r.setAttribute('data-theme', 'light');
    else if(t === 'escuro') r.setAttribute('data-theme', 'dark');
    else r.removeAttribute('data-theme');
  }
  function escolher(t){
    aplicar(t);
    try{ localStorage.setItem(CHAVE, t); }catch(e){}
  }
  try{ aplicar(localStorage.getItem(CHAVE)); }catch(e){}
  window.Tema = { escolher: escolher };
})();
