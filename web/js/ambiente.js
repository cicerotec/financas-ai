(function(){
  var c = window.FINANCAS_CONFIG || {};
  if(c.ambiente !== 'dev') return;
  // instalar o app do ambiente de teste: manifest e icones com a faixa DEV, para nao confundir com o de producao
  var m = document.querySelector('link[rel="manifest"]');
  if(m) m.href = 'manifest.dev.webmanifest';
  var ic = document.querySelector('link[rel="icon"]');
  if(ic) ic.href = 'icons/dev/icon-192-v1.png';
  var at = document.querySelector('link[rel="apple-touch-icon"]');
  if(at) at.href = 'icons/dev/apple-touch-icon-v1.png';
  var tc = document.querySelector('meta[name="theme-color"]');
  if(tc) tc.content = '#e8710a';
  var f = document.createElement('div');
  f.className = 'faixa-ambiente';
  f.setAttribute('role', 'status');
  f.textContent = 'AMBIENTE DE TESTE — dados fictícios';
  document.body.insertBefore(f, document.body.firstChild);
  document.documentElement.style.setProperty('--faixa-h', '26px');
  document.title = '[DEV] ' + document.title;
})();
