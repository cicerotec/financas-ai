// Testes do gerador de QR code (web/js/qr.js). Spec mfa-opcional, R19. Roda em Node: node web/qr.test.js
// O QR guarda a chave do TOTP, entao o teste e rigoroso: o vetor oficial da norma (ISO 18004, Anexo I) para o Reed-Solomon e um DECODIFICADOR
// ESCRITO A PARTE, so a partir da norma (formato, mascara, zigue-zague, intercalacao, sindromes de Reed-Solomon e modo de bytes), que le a matriz
// pronta e tem de devolver o texto. Um leitor de terceiros (celular) e conferido a mao na entrega.
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const QR = require('./js/qr.js');

let falhas = 0, concluido = false;
process.on('exit', () => { if (!concluido) { console.error('qr.test.js: o teste nao terminou'); process.exitCode = 1; } });
const fila = [];
const teste = (nome, fn) => fila.push([nome, fn]);

// ---------- o decodificador independente ----------
const EXP = new Array(512), LOG = new Array(256);
(function () { let x = 1; for (let i = 0; i < 255; i++) { EXP[i] = x; LOG[x] = i; x <<= 1; if (x & 0x100) x ^= 0x11D; } for (let i = 255; i < 512; i++) EXP[i] = EXP[i - 255]; })();
const gmul = (a, b) => (a === 0 || b === 0 ? 0 : EXP[LOG[a] + LOG[b]]);
const bitDe = (x, i) => ((x >>> i) & 1) === 1;

// da norma: ECC por bloco e blocos (grupo 1 e grupo 2: [quantidade, bytes de dados]) do nivel M, versoes 1 a 10
const BLOCOS_M = {
  1: { ecc: 10, g: [[1, 16]] }, 2: { ecc: 16, g: [[1, 28]] }, 3: { ecc: 26, g: [[1, 44]] }, 4: { ecc: 18, g: [[2, 32]] }, 5: { ecc: 24, g: [[2, 43]] },
  6: { ecc: 16, g: [[4, 27]] }, 7: { ecc: 18, g: [[4, 31]] }, 8: { ecc: 22, g: [[2, 38], [2, 39]] }, 9: { ecc: 22, g: [[3, 36], [2, 37]] }, 10: { ecc: 26, g: [[4, 43], [1, 44]] }
};
const CENTROS = { 1: [], 2: [6, 18], 3: [6, 22], 4: [6, 26], 5: [6, 30], 6: [6, 34], 7: [6, 22, 38], 8: [6, 24, 42], 9: [6, 26, 46], 10: [6, 28, 50] };
const MASCARAS = [
  (x, y) => (x + y) % 2 === 0, (x, y) => y % 2 === 0, (x, y) => x % 3 === 0, (x, y) => (x + y) % 3 === 0,
  (x, y) => (Math.floor(x / 3) + Math.floor(y / 2)) % 2 === 0, (x, y) => (x * y) % 2 + (x * y) % 3 === 0,
  (x, y) => ((x * y) % 2 + (x * y) % 3) % 2 === 0, (x, y) => (((x + y) % 2) + (x * y) % 3) % 2 === 0
];
// os 32 formatos validos: 5 bits (nivel 2 + mascara 3), BCH(15,5) com 0x537, xor 0x5412
function formatoValido(dados) { let r = dados; for (let i = 0; i < 10; i++) r = (r << 1) ^ ((r >>> 9) * 0x537); return ((dados << 10) | r) ^ 0x5412; }

function decodificar(m) {
  const n = m.length, v = (n - 17) / 4;
  assert.ok(Number.isInteger(v) && v >= 1 && v <= 10, 'tamanho invalido ' + n);
  const g = (x, y) => m[y][x];
  // 1. o formato, lido nas duas copias, tem de ser igual e valido
  const copia1 = [], copia2 = [];
  for (let i = 0; i <= 5; i++) copia1[i] = g(8, i);
  copia1[6] = g(8, 7); copia1[7] = g(8, 8); copia1[8] = g(7, 8);
  for (let i = 9; i < 15; i++) copia1[i] = g(14 - i, 8);
  for (let i = 0; i < 8; i++) copia2[i] = g(n - 1 - i, 8);
  for (let i = 8; i < 15; i++) copia2[i] = g(8, n - 15 + i);
  const num = (c) => c.reduce((a, b, i) => a | ((b ? 1 : 0) << i), 0);
  assert.strictEqual(num(copia1), num(copia2), 'as duas copias do formato diferem');
  let dados = -1;
  for (let d = 0; d < 32; d++) if (formatoValido(d) === num(copia1)) dados = d;
  assert.ok(dados >= 0, 'formato invalido (BCH)');
  assert.strictEqual(dados >> 3, 0, 'nivel de correcao diferente de M');
  const mascara = dados & 7;
  // 2. o que e padrao de funcao (nao carrega dados)
  const alin = CENTROS[v], pontos = [];
  alin.forEach((cx, i) => alin.forEach((cy, j) => { if (!((i === 0 && j === 0) || (i === 0 && j === alin.length - 1) || (i === alin.length - 1 && j === 0))) pontos.push([cx, cy]); }));
  const funcao = (x, y) => (x <= 8 && y <= 8) || (x >= n - 8 && y <= 8) || (x <= 8 && y >= n - 8) || x === 6 || y === 6 ||
    pontos.some(([cx, cy]) => Math.abs(x - cx) <= 2 && Math.abs(y - cy) <= 2) ||
    (v >= 7 && ((x >= n - 11 && x <= n - 9 && y <= 5) || (y >= n - 11 && y <= n - 9 && x <= 5)));
  // 3. a informacao de versao (v >= 7), BCH(18,6)
  if (v >= 7) {
    let r = v; for (let i = 0; i < 12; i++) r = (r << 1) ^ ((r >>> 11) * 0x1F25);
    const quer = (v << 12) | r;
    let lido = 0;
    for (let i = 0; i < 18; i++) lido |= (g(n - 11 + (i % 3), Math.floor(i / 3)) ? 1 : 0) << i;
    assert.strictEqual(lido, quer, 'informacao de versao');
  }
  // 4. os bits de dados, no zigue-zague, ja sem a mascara
  const bits = [];
  for (let direita = n - 1; direita >= 1; direita -= 2) {
    if (direita === 6) direita = 5;
    for (let vert = 0; vert < n; vert++) for (let j = 0; j < 2; j++) {
      const x = direita - j, y = ((direita + 1) & 2) === 0 ? n - 1 - vert : vert;
      if (!funcao(x, y)) bits.push(g(x, y) !== MASCARAS[mascara](x, y) ? 1 : 0);
    }
  }
  const bytes = [];
  for (let i = 0; i + 8 <= bits.length; i += 8) bytes.push(bits.slice(i, i + 8).reduce((a, b) => (a << 1) | b, 0));
  // 5. desintercala nos blocos e confere os sindromes de Reed-Solomon de cada um
  const { ecc, g: grupos } = BLOCOS_M[v];
  const tamanhos = [];
  grupos.forEach(([q, k]) => { for (let i = 0; i < q; i++) tamanhos.push(k); });
  const blocos = tamanhos.map(() => []);
  const maxDados = Math.max(...tamanhos);
  let p = 0;
  for (let i = 0; i < maxDados; i++) tamanhos.forEach((k, b) => { if (i < k) blocos[b].push(bytes[p++]); });
  for (let i = 0; i < ecc; i++) blocos.forEach((bl) => bl.push(bytes[p++]));
  assert.strictEqual(p, tamanhos.reduce((a, b) => a + b, 0) + ecc * tamanhos.length, 'quantidade de codigos');
  const dadosTodos = [];
  blocos.forEach((bl, b) => {
    for (let s = 0; s < ecc; s++) { // sindrome S_s = bloco(alfa^s); tem de ser zero
      let acc = 0;
      for (const c of bl) acc = gmul(acc, EXP[s]) ^ c;
      assert.strictEqual(acc, 0, 'sindrome ' + s + ' do bloco ' + b + ' diferente de zero');
    }
    dadosTodos.push(...bl.slice(0, tamanhos[b]));
  });
  // 6. o modo de bytes
  const fluxo = dadosTodos.flatMap((b) => [7, 6, 5, 4, 3, 2, 1, 0].map((i) => (b >> i) & 1));
  const le = (n0, ini) => fluxo.slice(ini, ini + n0).reduce((a, b) => (a << 1) | b, 0);
  assert.strictEqual(le(4, 0), 0b0100, 'modo de bytes');
  const bc = v >= 10 ? 16 : 8, qtd = le(bc, 4);
  const txt = [];
  for (let i = 0; i < qtd; i++) txt.push(le(8, 4 + bc + i * 8));
  return { versao: v, mascara, texto: decodeURIComponent(escape(String.fromCharCode(...txt))) };
}

// ---------- os testes ----------
teste('Reed-Solomon: o vetor oficial da norma (ISO 18004, Anexo I, "01234567", versao 1-M)', () => {
  const dados = [0x10, 0x20, 0x0C, 0x56, 0x61, 0x80, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11];
  assert.deepStrictEqual(QR.reedSolomon(dados, 10), [0xA5, 0x24, 0xD4, 0xC1, 0xED, 0x36, 0xC7, 0x87, 0x2C, 0x55]);
});

teste('as capacidades em bytes (nivel M) sao as da norma: 14, 26, 42, 62, 84, 106, 122, 152, 180 e 213', () => {
  assert.deepStrictEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map(QR._capacidade), [14, 26, 42, 62, 84, 106, 122, 152, 180, 213]);
});

teste('o decodificador independente le de volta o texto, em todas as versoes, nas fronteiras de capacidade, com UTF-8', () => {
  const textos = ['a', 'HELLO WORLD', 'otpauth://totp/Financas?secret=JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP&issuer=Financas',
    'otpauth://totp/Finan%C3%A7as%20DEV?secret=JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP&issuer=Finan%C3%A7as%20DEV', 'Finanças ação teste de utf-8 é ç ã'];
  for (const cap of [14, 26, 42, 62, 84, 106, 122, 152, 180, 213]) {
    textos.push('k'.repeat(cap)); // cabe exatamente
    if (cap < 213) textos.push('K'.repeat(cap + 1)); // so cabe na versao seguinte
  }
  const versoes = new Set();
  for (const t of textos) {
    const m = QR.matriz(t);
    const d = decodificar(m.modulos);
    assert.strictEqual(d.texto, t, 'texto de ' + t.length + ' letras');
    assert.deepStrictEqual([d.versao, d.mascara], [m.versao, m.mascara]);
    versoes.add(m.versao);
  }
  assert.deepStrictEqual([...versoes].sort((a, b) => a - b), [1, 2, 3, 4, 5, 6, 7, 8, 9, 10], 'todas as versoes foram exercitadas');
  assert.throws(() => QR.matriz('x'.repeat(214)), /grande demais/);
});

teste('a estrutura: localizadores, sincronia e modulo escuro fixo; cada uma das 8 mascaras produz uma matriz que o decodificador le', () => {
  const m = QR.matriz('otpauth://totp/Financas?secret=JBSWY3DPEHPK3PXP&issuer=Financas'), n = m.tamanho, g = (x, y) => m.modulos[y][x];
  for (const [cx, cy] of [[3, 3], [n - 4, 3], [3, n - 4]]) { // anel 3x3 escuro no centro, anel claro, anel escuro externo
    for (let dy = -3; dy <= 3; dy++) for (let dx = -3; dx <= 3; dx++) {
      const d = Math.max(Math.abs(dx), Math.abs(dy));
      assert.strictEqual(g(cx + dx, cy + dy), d !== 2, 'localizador em ' + cx + ',' + cy);
    }
  }
  for (let i = 8; i < n - 8; i++) { assert.strictEqual(g(6, i), i % 2 === 0); assert.strictEqual(g(i, 6), i % 2 === 0); }
  assert.strictEqual(g(8, n - 8), true, 'o modulo escuro fixo');
  for (const t of ['otpauth://totp/Financas?secret=JBSWY3DPEHPK3PXP&issuer=Financas', 'a', 'x'.repeat(150)]) {
    for (let k = 0; k < 8; k++) {
      const forcada = QR.matriz(t, k);
      assert.strictEqual(forcada.mascara, k);
      assert.strictEqual(decodificar(forcada.modulos).texto, t, 'mascara ' + k);
    }
  }
  const vistas = new Set();
  for (const t of ['a', 'bb', 'ccc', 'dddd', 'eeeee', 'ffffff', 'ggggggg', 'hhhhhhhh', 'iiiiiiiii', 'jjjjjjjjjj', '0123456789', 'AAAAAAAAAAAAAAAA', 'zxcvbnmasdfghjkl', 'qwertyuiop12345678', '9876543210abcdefghijklmnopqrstuvwxyz']) {
    const r = QR.matriz(t); vistas.add(r.mascara);
    assert.strictEqual(decodificar(r.modulos).texto, t);
  }
  assert.ok(vistas.size >= 2, 'a mascara e escolhida pela penalidade (varia com o texto)');
});

teste('um modulo invertido numa area de dados quebra os sindromes de Reed-Solomon (o decodificador detecta)', () => {
  const m = QR.matriz('otpauth://totp/Financas?secret=JBSWY3DPEHPK3PXP&issuer=Financas');
  const c = m.modulos.map((l) => l.slice());
  c[m.tamanho - 1][m.tamanho - 1] = !c[m.tamanho - 1][m.tamanho - 1];
  assert.throws(() => decodificar(c), /sindrome|formato|quantidade/);
});

teste('svg: sempre preto sobre branco com margem de 4 modulos, deterministico, e nao leva o texto em claro', () => {
  const t = 'otpauth://totp/Financas?secret=JBSWY3DPEHPK3PXPSEGREDO&issuer=Financas';
  const s = QR.svg(t), m = QR.matriz(t);
  assert.strictEqual(s, QR.svg(t), 'deterministico');
  assert.match(s, /^<svg [^>]*viewBox="0 0 (\d+) \1"/);
  assert.strictEqual(Number(/viewBox="0 0 (\d+)/.exec(s)[1]), m.tamanho + 8, 'margem de silencio de 4 modulos de cada lado');
  assert.ok(s.includes('<rect') && s.includes('fill="#fff"') && s.includes('fill="#000"'), 'preto sobre branco, em qualquer tema');
  assert.ok(!s.includes('SEGREDO') && !s.includes('JBSW'), 'o SVG nao carrega o texto, so os modulos');
  assert.ok(!/href|script|on[a-z]+=/i.test(s), 'o SVG nao tem link nem script');
});

teste('R19: o modulo nao usa rede, armazenamento, area de transferencia nem log', () => {
  const fonte = fs.readFileSync(path.join(__dirname, 'js', 'qr.js'), 'utf8');
  for (const proibido of [/\bfetch\b/, /XMLHttpRequest/, /localStorage/, /sessionStorage/, /indexedDB/, /clipboard/, /console\./, /\bimport\b/, /\brequire\(/, /document\./, /window\./, /sendBeacon/]) {
    assert.ok(!proibido.test(fonte.replace(/(^|\s)\/\/.*$/gm, '$1')), 'qr.js nao pode ter ' + proibido);
  }
});

(async () => {
  for (const [nome, fn] of fila) {
    try { await fn(); console.log('ok   ' + nome); } catch (e) { falhas++; console.error('FALHOU: ' + nome + '\n  ' + (e.stack || e.message).split('\n').slice(0, 4).join('\n  ')); }
  }
  concluido = true;
  if (falhas) { console.error(falhas + ' teste(s) falharam'); process.exit(1); }
  console.log('qr.test.js: ok');
})();
