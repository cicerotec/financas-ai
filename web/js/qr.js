// Gerador de QR code, SO para o que o MFA precisa (spec mfa-opcional, R19): um texto curto (o endereco otpauth:// do TOTP) em modo de bytes,
// correcao de erros nivel M, versoes 1 a 10 (ate 213 bytes). Escrito aqui, sem biblioteca de terceiros, porque a chave do TOTP passa por ele:
// o texto nunca sai deste modulo (sem rede, sem armazenamento, sem log) e o QR e desenhado no aparelho.
// Segue a ISO/IEC 18004: padroes de localizacao, alinhamento e sincronia, informacao de formato e de versao (BCH), Reed-Solomon sobre GF(256),
// intercalacao dos blocos, colocacao em zigue-zague e a escolha da mascara pela menor penalidade (as quatro regras).
// Uso: QR.matriz(texto) -> {tamanho, modulos[y][x] (true = escuro), versao}; QR.svg(texto, {modulo, margem}) -> texto do <svg>.
const QR = (function () {
  // ---------- tabelas do nivel M, versoes 1 a 10 ----------
  const ECC_POR_BLOCO = [0, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26];
  const NUM_BLOCOS = [0, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5];
  const ALINHAMENTO = [null, [], [6, 18], [6, 22], [6, 26], [6, 30], [6, 34], [6, 22, 38], [6, 24, 42], [6, 26, 46], [6, 28, 50]];
  const VERSAO_MAX = 10;
  const NIVEL_M = 0; // bits do nivel no formato: L=1, M=0, Q=3, H=2

  const bit = (x, i) => ((x >>> i) & 1) !== 0;

  // ---------- Reed-Solomon sobre GF(256), polinomio 0x11D ----------
  function mul(x, y) {
    let z = 0;
    for (let i = 7; i >= 0; i--) {
      z = (z << 1) ^ ((z >>> 7) * 0x11D);
      z ^= ((y >>> i) & 1) * x;
    }
    return z;
  }
  function divisor(grau) {
    const r = new Array(grau).fill(0);
    r[grau - 1] = 1;
    let raiz = 1;
    for (let i = 0; i < grau; i++) {
      for (let j = 0; j < grau; j++) {
        r[j] = mul(r[j], raiz);
        if (j + 1 < grau) r[j] ^= r[j + 1];
      }
      raiz = mul(raiz, 0x02);
    }
    return r;
  }
  function resto(dados, div) {
    const r = new Array(div.length).fill(0);
    for (const b of dados) {
      const fator = b ^ r.shift();
      r.push(0);
      div.forEach((c, i) => { r[i] ^= mul(c, fator); });
    }
    return r;
  }
  // os codigos de correcao de erros de um bloco de dados (exportado para o teste com o vetor da norma)
  const reedSolomon = (dados, nEcc) => resto(dados, divisor(nEcc));

  // ---------- capacidade ----------
  function modulosDeDados(v) { // quantos modulos sobram para dados e correcao, sem os padroes de funcao
    let r = (16 * v + 128) * v + 64;
    if (v >= 2) {
      const n = Math.floor(v / 7) + 2;
      r -= (25 * n - 10) * n - 55;
      if (v >= 7) r -= 36;
    }
    return r;
  }
  const codigosDeDados = v => Math.floor(modulosDeDados(v) / 8) - ECC_POR_BLOCO[v] * NUM_BLOCOS[v];

  function bytesUtf8(texto) { // UTF-8 por encodeURIComponent: roda igual no navegador, no app e no Node, sem TextEncoder nem globais legados
    const enc = encodeURIComponent(texto), r = [];
    for (let i = 0; i < enc.length; i++) {
      if (enc[i] === '%') { r.push(parseInt(enc.substr(i + 1, 2), 16)); i += 2; } else r.push(enc.charCodeAt(i));
    }
    return r;
  }

  // ---------- os dados: modo de bytes, terminador, enchimento ----------
  function codigosDeDadosDe(bytes, v) {
    const bits = [];
    const poe = (valor, n) => { for (let i = n - 1; i >= 0; i--) bits.push(bit(valor, i) ? 1 : 0); };
    poe(0x4, 4); // modo de bytes
    poe(bytes.length, v >= 10 ? 16 : 8);
    for (const b of bytes) poe(b, 8);
    const capBits = codigosDeDados(v) * 8;
    poe(0, Math.min(4, capBits - bits.length)); // terminador
    poe(0, (8 - (bits.length % 8)) % 8);
    for (let pad = 0xEC; bits.length < capBits; pad ^= 0xEC ^ 0x11) poe(pad, 8);
    const r = [];
    for (let i = 0; i < bits.length; i += 8) {
      let b = 0;
      for (let j = 0; j < 8; j++) b = (b << 1) | bits[i + j];
      r.push(b);
    }
    return r;
  }

  // ---------- correcao de erros e intercalacao ----------
  function comEccIntercalado(dados, v) {
    const nBlocos = NUM_BLOCOS[v], eccLen = ECC_POR_BLOCO[v];
    const bruto = Math.floor(modulosDeDados(v) / 8);
    const curtos = nBlocos - (bruto % nBlocos), tamCurto = Math.floor(bruto / nBlocos);
    const div = divisor(eccLen);
    const blocos = [];
    for (let i = 0, k = 0; i < nBlocos; i++) {
      const d = dados.slice(k, k + tamCurto - eccLen + (i < curtos ? 0 : 1));
      k += d.length;
      const ecc = resto(d, div);
      if (i < curtos) d.push(0); // enchimento so para alinhar a intercalacao
      blocos.push(d.concat(ecc));
    }
    const out = [];
    for (let i = 0; i < blocos[0].length; i++) {
      blocos.forEach((b, j) => { if (i !== tamCurto - eccLen || j >= curtos) out.push(b[i]); });
    }
    return out;
  }

  // ---------- a matriz ----------
  function montar(v, codigos) {
    const n = v * 4 + 17;
    const mod = Array.from({ length: n }, () => new Array(n).fill(false));
    const fun = Array.from({ length: n }, () => new Array(n).fill(false));
    const poeFuncao = (x, y, escuro) => { mod[y][x] = escuro; fun[y][x] = true; };

    // sincronia
    for (let i = 0; i < n; i++) { poeFuncao(6, i, i % 2 === 0); poeFuncao(i, 6, i % 2 === 0); }
    // localizacao (com o separador) nos tres cantos
    const localizador = (cx, cy) => {
      for (let dy = -4; dy <= 4; dy++) for (let dx = -4; dx <= 4; dx++) {
        const d = Math.max(Math.abs(dx), Math.abs(dy)), x = cx + dx, y = cy + dy;
        if (x >= 0 && x < n && y >= 0 && y < n) poeFuncao(x, y, d !== 2 && d !== 4);
      }
    };
    localizador(3, 3); localizador(n - 4, 3); localizador(3, n - 4);
    // alinhamento
    const pos = ALINHAMENTO[v];
    for (let i = 0; i < pos.length; i++) for (let j = 0; j < pos.length; j++) {
      if ((i === 0 && j === 0) || (i === 0 && j === pos.length - 1) || (i === pos.length - 1 && j === 0)) continue; // cobriria um localizador
      for (let dy = -2; dy <= 2; dy++) for (let dx = -2; dx <= 2; dx++) poeFuncao(pos[i] + dx, pos[j] + dy, Math.max(Math.abs(dx), Math.abs(dy)) !== 1);
    }
    // informacao de versao (v >= 7)
    if (v >= 7) {
      let r = v;
      for (let i = 0; i < 12; i++) r = (r << 1) ^ ((r >>> 11) * 0x1F25);
      const bits = (v << 12) | r;
      for (let i = 0; i < 18; i++) {
        const a = n - 11 + (i % 3), b = Math.floor(i / 3);
        poeFuncao(a, b, bit(bits, i)); poeFuncao(b, a, bit(bits, i));
      }
    }
    const formato = mascara => { // 15 bits: nivel + mascara, BCH(15,5), xor 0x5412
      const dados = (NIVEL_M << 3) | mascara;
      let r = dados;
      for (let i = 0; i < 10; i++) r = (r << 1) ^ ((r >>> 9) * 0x537);
      const bits = ((dados << 10) | r) ^ 0x5412;
      for (let i = 0; i <= 5; i++) poeFuncao(8, i, bit(bits, i));
      poeFuncao(8, 7, bit(bits, 6)); poeFuncao(8, 8, bit(bits, 7)); poeFuncao(7, 8, bit(bits, 8));
      for (let i = 9; i < 15; i++) poeFuncao(14 - i, 8, bit(bits, i));
      for (let i = 0; i < 8; i++) poeFuncao(n - 1 - i, 8, bit(bits, i));
      for (let i = 8; i < 15; i++) poeFuncao(8, n - 15 + i, bit(bits, i));
      poeFuncao(8, n - 8, true); // o modulo escuro fixo
    };
    formato(0); // reserva as posicoes (o valor certo entra depois da mascara)

    // dados, em zigue-zague, de baixo para cima, de duas em duas colunas (a coluna 6 e pulada)
    let k = 0;
    for (let direita = n - 1; direita >= 1; direita -= 2) {
      if (direita === 6) direita = 5;
      for (let vert = 0; vert < n; vert++) for (let j = 0; j < 2; j++) {
        const x = direita - j, sobe = ((direita + 1) & 2) === 0, y = sobe ? n - 1 - vert : vert;
        if (!fun[y][x] && k < codigos.length * 8) { mod[y][x] = bit(codigos[k >>> 3], 7 - (k & 7)); k++; }
      }
    }
    return { n, mod, fun, formato };
  }

  const MASCARAS = [
    (x, y) => (x + y) % 2 === 0,
    (x, y) => y % 2 === 0,
    (x, y) => x % 3 === 0,
    (x, y) => (x + y) % 3 === 0,
    (x, y) => (Math.floor(x / 3) + Math.floor(y / 2)) % 2 === 0,
    (x, y) => ((x * y) % 2) + ((x * y) % 3) === 0,
    (x, y) => (((x * y) % 2) + ((x * y) % 3)) % 2 === 0,
    (x, y) => (((x + y) % 2) + ((x * y) % 3)) % 2 === 0
  ];
  function aplicarMascara(m, i) {
    for (let y = 0; y < m.n; y++) for (let x = 0; x < m.n; x++) if (!m.fun[y][x] && MASCARAS[i](x, y)) m.mod[y][x] = !m.mod[y][x];
  }

  // as quatro regras de penalidade da norma
  function penalidade(mod, n) {
    let p = 0;
    const linhas = [];
    for (let y = 0; y < n; y++) linhas.push(mod[y]);
    for (let x = 0; x < n; x++) linhas.push(mod.map(l => l[x]));
    for (const l of linhas) { // R1: sequencias de 5 ou mais iguais; R3: o padrao 1:1:3:1:1 com 4 claros de um lado
      let corrida = 1;
      for (let i = 1; i < n; i++) {
        if (l[i] === l[i - 1]) { corrida++; if (corrida === 5) p += 3; else if (corrida > 5) p++; } else corrida = 1;
      }
      const s = l.map(b => (b ? '1' : '0')).join('');
      for (const padrao of ['10111010000', '00001011101']) { let from = 0, i; while ((i = s.indexOf(padrao, from)) >= 0) { p += 40; from = i + 1; } }
    }
    for (let y = 0; y < n - 1; y++) for (let x = 0; x < n - 1; x++) { // R2: blocos 2x2 da mesma cor
      const c = mod[y][x];
      if (c === mod[y][x + 1] && c === mod[y + 1][x] && c === mod[y + 1][x + 1]) p += 3;
    }
    let escuros = 0;
    for (const l of mod) for (const b of l) if (b) escuros++;
    const total = n * n;
    p += (Math.ceil(Math.abs(escuros * 20 - total * 10) / total) - 1) * 10; // R4: proporcao de escuros
    return p;
  }

  // mascaraForcada (0 a 7): so para o teste provar as oito mascaras; em uso normal a de menor penalidade e escolhida
  function matriz(texto, mascaraForcada) {
    const bytes = bytesUtf8(String(texto));
    let v = 1;
    // capacidade em bytes: os bits dos dados menos o modo (4) e o contador (8 ou 16), em bytes inteiros
    const capacidade = ver => Math.floor((codigosDeDados(ver) * 8 - 4 - (ver >= 10 ? 16 : 8)) / 8);
    while (v <= VERSAO_MAX && bytes.length > capacidade(v)) v++;
    if (v > VERSAO_MAX) throw new Error('texto grande demais para o QR code');
    const codigos = comEccIntercalado(codigosDeDadosDe(bytes, v), v);
    const m = montar(v, codigos);
    let melhor = 0, menor = Infinity;
    for (let i = 0; i < 8; i++) { // escolhe a mascara de menor penalidade
      aplicarMascara(m, i); m.formato(i);
      const p = penalidade(m.mod, m.n);
      if (p < menor) { menor = p; melhor = i; }
      aplicarMascara(m, i); // desfaz (a mascara e uma involucao)
    }
    if (mascaraForcada !== undefined) melhor = mascaraForcada;
    aplicarMascara(m, melhor); m.formato(melhor);
    return { tamanho: m.n, versao: v, mascara: melhor, modulos: m.mod };
  }

  // O QR sempre preto sobre branco, com a margem de silencio de 4 modulos: o leitor precisa dela e do contraste, em qualquer tema do app.
  function svg(texto, op) {
    op = op || {};
    const margem = op.margem === undefined ? 4 : op.margem, lado = (op.modulo || 4);
    const m = matriz(texto), t = m.tamanho + margem * 2;
    let d = '';
    for (let y = 0; y < m.tamanho; y++) for (let x = 0; x < m.tamanho; x++) if (m.modulos[y][x]) d += 'M' + (x + margem) + ' ' + (y + margem) + 'h1v1h-1z';
    return '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ' + t + ' ' + t + '" width="' + t * lado + '" height="' + t * lado + '" shape-rendering="crispEdges" role="img" aria-label="QR code">' +
      '<rect width="' + t + '" height="' + t + '" fill="#fff"/><path d="' + d + '" fill="#000"/></svg>';
  }

  return { matriz, svg, reedSolomon, _capacidade: v => Math.floor((codigosDeDados(v) * 8 - 4 - (v >= 10 ? 16 : 8)) / 8), _versaoMax: VERSAO_MAX };
})();

if (typeof module !== 'undefined' && module.exports) module.exports = QR;
