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
  // a abertura (js/abertura.js) mostra o mesmo icone do splash do Android: em dev ele leva a faixa DEV, como o icone instalado
  // (a tela de bloqueio do app Android usa o mesmo icone e leva a mesma faixa)
  [].forEach.call(document.querySelectorAll('#abIc, #bloqueio .bq-ic'), function(ab){ ab.insertAdjacentHTML('beforeend', '<rect x="158" y="394" width="196" height="80" rx="34" fill="#d97b2f"/><text x="256" y="443" text-anchor="middle" fill="#fff" style="font:700 50px system-ui,sans-serif">DEV</text>'); });
  var f = document.createElement('div');
  f.className = 'faixa-ambiente';
  f.setAttribute('role', 'status');
  f.textContent = 'AMBIENTE DE TESTE — dados fictícios';
  document.body.insertBefore(f, document.body.firstChild);
  document.documentElement.style.setProperty('--faixa-h', '26px');
  document.title = '[DEV] ' + document.title;
})();
