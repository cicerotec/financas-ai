// Teste do item 41 (.ai/specs/futuros-previsto): o front NAO trata status. Quem manda a lista de status, o que conta no saldo, o que e futuro,
// o que e credito e a cor padrao e o servidor; o app so exibe a lista e devolve o texto escolhido. Este teste falha se web/ citar o nome de um
// status ou um dos predicados que decidiam isso no navegador.
// Excecoes: uma linha com o marcador `sem-status:ok` (comentario na mesma linha), com o motivo ao lado. Comentarios de codigo nao contam.
// Rode com: node web/sem-status.test.js
const fs = require('fs');
const path = require('path');

const raiz = __dirname;
const arquivos = [path.join(raiz, 'index.html')];
for (const f of fs.readdirSync(path.join(raiz, 'js'))) if (f.endsWith('.js')) arquivos.push(path.join(raiz, 'js', f));
for (const f of fs.readdirSync(raiz)) if (f.endsWith('.js') && !f.includes('.test.') && !f.startsWith('config')) arquivos.push(path.join(raiz, f));

const NOMES = /\b(PAGO|CREDITO|CONTAS|PREVISTO|TRANSFERINDO|CREDITANDO)\b/;
const PREDICADOS = /\b(ehFuturo|ehStatusReal|afetaSaldo|afetaSaldoBanco|ehCredito|ehCreditoEX|ehCreditoIN|statusFuturo|statusReal)\b/;

// tira comentarios (// e /* */ nos .js, <!-- --> no .html), sem mexer em textos entre aspas
function semComentarios(src, html) {
  if (html) return src.replace(/<!--[\s\S]*?-->/g, (m) => m.replace(/[^\n]/g, ' ').replace(/sem-status:ok/g, ''));
  let out = '';
  let i = 0;
  const n = src.length;
  while (i < n) {
    const c = src[i], d = src[i + 1];
    if (c === '/' && d === '*') { const f = src.indexOf('*/', i + 2); const fim = f < 0 ? n : f + 2; out += src.slice(i, fim).replace(/[^\n]/g, ' '); i = fim; continue; }
    if (c === '/' && d === '/') { while (i < n && src[i] !== '\n') { out += ' '; i++; } continue; }
    if (c === '\'' || c === '"' || c === '`') {
      out += c; i++;
      while (i < n && src[i] !== c) {
        if (src[i] === '\\') { out += src[i] + (src[i + 1] || ''); i += 2; continue; }
        if (src[i] === '\n' && c !== '`') break;
        out += src[i]; i++;
      }
      out += src[i] || ''; i++;
      continue;
    }
    out += c; i++;
  }
  return out;
}

const achados = [];
for (const arq of arquivos) {
  const bruto = fs.readFileSync(arq, 'utf8');
  const brutas = bruto.split('\n');
  const limpas = semComentarios(bruto, arq.endsWith('.html')).split('\n');
  limpas.forEach((linha, i) => {
    if (/sem-status:ok/.test(brutas[i])) return;
    const m = linha.match(NOMES) || linha.match(PREDICADOS);
    if (m) achados.push(path.relative(raiz, arq) + ':' + (i + 1) + '  ' + m[0] + '  | ' + linha.trim().slice(0, 100));
  });
}

if (achados.length) {
  console.error('FALHOU: o front cita status ou decide por status (so o servidor faz isso):\n  ' + achados.join('\n  '));
  console.error('\nSe a linha for legitima (ex.: nome de coluna de planilha), ponha `sem-status:ok` num comentario na mesma linha, com o motivo.');
  process.exit(1);
}
console.log('ok   sem-status: web/ nao cita nome de status nem decide por status (' + arquivos.length + ' arquivos)');
