// Painel do cartao aberto (aba Cartão de crédito): cada secao fica num .cc-sec, com uma linha fina entre elas (demo-cartao-v4-separadores).
// A linha e do CSS e fica FORA do trecho que recolhe: recolher uma secao nao tira nem duplica a linha.
const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const css = fs.readFileSync(path.join(__dirname, 'css', 'app.css'), 'utf8');

// 1. o CSS: linha de 1 px em --border, menos na primeira secao
assert.ok(/\.cc-sec\{[^}]*border-top:1px solid var\(--border\)/.test(css), '.cc-sec tem a linha fina nos tokens do app');
assert.ok(/\.cc-sec:first-child\{[^}]*border-top:0/.test(css), 'a primeira secao nao leva linha');

// 2. o codigo real de painelCartaoHtml, com um app de mentira
function funcao(nome) {
  const ini = app.indexOf('function ' + nome + '(');
  assert.ok(ini >= 0, 'nao achei ' + nome);
  let i = app.indexOf('{', app.indexOf(')', ini)), n = 0;
  for (; i < app.length; i++) { if (app[i] === '{') n++; else if (app[i] === '}' && --n === 0) break; }
  return app.slice(ini, i + 1);
}
const fonte = ['escHtml', 'esc', 'fmtMoeda', 'fmtDiaMes', 'fmtDataCompleta', 'rotuloMesCurto', 'linhaCfg', 'tituloRecolhivel', 'conferenciaHtml', 'painelCartaoHtml'].map(funcao).join('\n');

function painel({ dono, cfgAberto, comprasAbertas, pagAberto, pagamentos, historico }) {
  const ctx = {
    cartaoFatura: {
      meio: 'Bradesco', ym: '2026-11', periodo: { ini: '2026-10-25', fim: '2026-11-24' }, anterior: '2026-10', proxima: '2026-12',
      chips: [{ ym: '2026-10', rotulo: 'Anterior' }, { ym: '2026-11', rotulo: 'Atual' }, { ym: '2026-12', rotulo: 'Próxima' }],
      fechada: false, historico: !!historico, inicio: '', total: 100, valorBanco: null, diferenca: null, conferida: null, vencimento: '', pagador: '',
      lancamentos: [{ id: 'a', dia: '2026-11-03', descricao: 'Compra', valor: 100, ajuste: 0, fatura: '2026-11', base: '2026-11', origem: null, estimado: false, podeMexer: true }],
      pagamentos: pagamentos ? [{ dia: '2026-12-02', meio: 'Itau', valor: 100 }] : [], totalPago: 100,
    },
    cartoesResumo: { cartoes: [{ meio: 'Bradesco' }] },
    cartaoCfgAberto: cfgAberto, cartaoComprasAberto: comprasAbertas, cartaoPagAberto: pagAberto,
    cartaoRecSel: null, msgCartao: '', listas: { banco: [] }, ehOwner: () => dono,
    Recorrencia: { fraseAviso: () => '' }, marcaDesativado: () => '', MESES_NOME: ['janeiro','fevereiro','marco','abril','maio','junho','julho','agosto','setembro','outubro','novembro','dezembro'],
  };
  vm.createContext(ctx);
  vm.runInContext(fonte + '\nglobalThis.out = painelCartaoHtml();', ctx);
  return ctx.out;
}
const conta = (h, re) => (h.match(re) || []).length;

for (const dono of [true, false]) for (const aberto of [true, false]) {
  const h = painel({ dono, cfgAberto: aberto, comprasAbertas: aberto, pagAberto: aberto, pagamentos: true });
  const nome = `dono=${dono} aberto=${aberto}`;
  // tags de abrir e fechar balanceadas
  assert.strictEqual(conta(h, /<div[\s>]/g), conta(h, /<\/div>/g), 'div balanceado: ' + nome);
  // secoes: faturas, compras, total e pagamentos (+ configurar so para o dono)
  assert.strictEqual(conta(h, /class="cc-sec"/g), dono ? 5 : 4, 'numero de secoes: ' + nome);
  // o titulo que recolhe esta dentro da secao, e a secao fecha depois do corpo: a linha nao depende de estar aberto
  for (const chave of ['compras', 'pag'].concat(dono ? ['cfg'] : [])) {
    assert.ok(new RegExp('<div class="cc-sec"><button[^>]*data-tg="' + chave + '"').test(h), 'o titulo ' + chave + ' abre a secao: ' + nome);
  }
  // todo filho direto do painel e uma secao (nada solto entre as linhas)
  const fora = h.replace(/^<div class="sd-painel">/, '').replace(/<div class="cc-sec">[\s\S]*?(?=<div class="cc-sec">|<\/div>$)/g, '');
  assert.strictEqual(fora, '</div>', 'nada fora das secoes: ' + nome + ' -> ' + fora.slice(0, 80));
}

// 3. sem pagamentos a secao nao aparece (e nao deixa linha sobrando); fatura de historico tambem fecha direito
const semPag = painel({ dono: true, cfgAberto: false, comprasAbertas: true, pagAberto: true, pagamentos: false });
assert.strictEqual(conta(semPag, /class="cc-sec"/g), 4, 'sem pagamentos: 4 secoes');
assert.strictEqual(conta(semPag, /<div[\s>]/g), conta(semPag, /<\/div>/g), 'div balanceado sem pagamentos');
const hist = painel({ dono: true, cfgAberto: false, comprasAbertas: true, pagAberto: true, pagamentos: false, historico: true });
assert.strictEqual(conta(hist, /<div[\s>]/g), conta(hist, /<\/div>/g), 'div balanceado em fatura de historico');

console.log('cartao-separadores: ok');
