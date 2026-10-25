// Teste de referencias: toda funcao chamada pelo JavaScript do app precisa estar definida em algum lugar de web/.
// Nasceu de uma regressao real (2026-10-24): ao trocar a aba Cartao pelo servidor, `chaveDia` e `diasAte` foram apagados junto com o
// codigo do cartao, mas Historico e Futuros ainda os chamavam, e o app quebrava so quando a pessoa abria essas telas.
// E uma checagem aproximada (sem parser): ve nomes seguidos de "(" no codigo (sem comentarios, textos entre aspas e texto de template)
// e confere se aparecem como funcao, variavel, parametro ou nome importado.
// Rode com: node web/referencias.test.js
const fs = require('fs');
const path = require('path');

const raiz = __dirname;
const arquivos = [];
for (const f of fs.readdirSync(path.join(raiz, 'js'))) if (f.endsWith('.js')) arquivos.push(path.join(raiz, 'js', f));
for (const f of fs.readdirSync(raiz)) if (f.endsWith('.js') && !f.includes('.test.') && !f.startsWith('config')) arquivos.push(path.join(raiz, f));

const fonte = arquivos.map((f) => fs.readFileSync(f, 'utf8')).join('\n');

// Passa pelo codigo uma vez, caractere a caractere, separando comentarios, textos entre aspas e templates do que e codigo.
function soCodigo(src) {
  let out = '';
  let i = 0;
  const n = src.length;
  function template() { // dentro de `...`: guarda so o que esta em ${...}
    i++;
    while (i < n && src[i] !== '`') {
      if (src[i] === '\\') { i += 2; continue; }
      if (src[i] === '$' && src[i + 1] === '{') {
        i += 2;
        let fundo = 1;
        out += ' ';
        while (i < n && fundo > 0) {
          const c = src[i];
          if (c === '{') fundo++;
          else if (c === '}') { fundo--; if (!fundo) break; }
          if (c === '`') { template(); continue; }
          if (c === '\'' || c === '"') { texto(c); continue; }
          out += c; i++;
        }
        i++; out += ' ';
        continue;
      }
      i++;
    }
    i++;
  }
  function texto(aspa) {
    i++;
    while (i < n && src[i] !== aspa && src[i] !== '\n') { if (src[i] === '\\') i++; i++; }
    i++;
    out += '""';
  }
  while (i < n) {
    const c = src[i], d = src[i + 1];
    if (c === '/' && d === '*') { const f = src.indexOf('*/', i + 2); i = f < 0 ? n : f + 2; out += ' '; continue; }
    if (c === '/' && d === '/') { while (i < n && src[i] !== '\n') i++; continue; }
    if (c === '`') { template(); out += '""'; continue; }
    if (c === '\'' || c === '"') { texto(c); continue; }
    out += c; i++;
  }
  return out;
}
const limpo = soCodigo(fonte);

const declarados = new Set();
const ident = '[A-Za-z_$][\\w$]*';
for (const m of limpo.matchAll(new RegExp('\\bfunction\\s*\\*?\\s*(' + ident + ')', 'g'))) declarados.add(m[1]);
for (const m of limpo.matchAll(new RegExp('\\b(?:const|let|var|class)\\s+(' + ident + ')', 'g'))) declarados.add(m[1]);
for (const m of limpo.matchAll(/\b(?:const|let|var)\s*[{[]([^}\]=]*)[}\]]/g)) {
  m[1].split(',').forEach((x) => { const nome = x.split(':').pop().trim().split('=')[0].trim(); if (nome) declarados.add(nome); });
}
for (const m of limpo.matchAll(/\(([^()]*)\)\s*(?:=>|\{)/g)) {
  m[1].split(',').forEach((x) => { const nome = x.replace(/[{}[\].]/g, '').split('=')[0].trim(); if (new RegExp('^' + ident + '$').test(nome)) declarados.add(nome); });
}
for (const m of limpo.matchAll(new RegExp('\\b(' + ident + ')\\s*=>', 'g'))) declarados.add(m[1]);
for (const m of limpo.matchAll(new RegExp('\\bcatch\\s*\\(\\s*(' + ident + ')', 'g'))) declarados.add(m[1]);
// metodo de objeto: nome(params) {
for (const m of limpo.matchAll(new RegExp('(?<![.\\w$])(' + ident + ')\\s*\\([^()]*\\)\\s*\\{', 'g'))) declarados.add(m[1]);

const nativos = new Set(('if for while switch catch return typeof new await async function of in else do throw void delete super import export ' +
  'Math JSON Object Array String Number Boolean Date Promise Set Map WeakMap RegExp Error parseInt parseFloat isNaN isFinite setTimeout clearTimeout ' +
  'setInterval clearInterval fetch alert confirm requestAnimationFrame cancelAnimationFrame encodeURIComponent decodeURIComponent atob btoa ' +
  'structuredClone Intl URL URLSearchParams FileReader Blob File Uint8Array AbortController require module console window document navigator ' +
  'localStorage sessionStorage history location XLSX Symbol Reflect Proxy queueMicrotask getComputedStyle matchMedia IntersectionObserver ' +
  'MutationObserver ResizeObserver Notification crypto TextEncoder TextDecoder Image Event CustomEvent ' +
  'tendRender').split(' ')); // tendRender: chamada protegida por typeof (js/tendencias.js carrega depois do app.js)

const chamados = new Set();
for (const m of limpo.matchAll(new RegExp('(?<![.\\w$])(' + ident + ')\\s*\\(', 'g'))) chamados.add(m[1]);
const faltam = [...chamados].filter((nome) => !declarados.has(nome) && !nativos.has(nome)).sort();

if (faltam.length) {
  console.error('FALHOU: funcoes chamadas e nunca definidas: ' + faltam.join(', '));
  process.exit(1);
}
console.log('ok   referencias: toda funcao chamada esta definida (' + chamados.size + ' nomes conferidos)');
