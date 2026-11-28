// Conferencia das compras na aba Cartao de credito (demo-cartao-v6-conferencia): Selecionar / Selecionados · N, so em memoria.
// Regra: mudou algo na aba, apaga as marcas e desliga o modo; nao mudou, mantem. Roda o codigo real do app.js com um app de mentira.
const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const css = fs.readFileSync(path.join(__dirname, 'css', 'app.css'), 'utf8');

function funcao(nome) {
  let ini = app.indexOf('function ' + nome + '(');
  assert.ok(ini >= 0, 'nao achei ' + nome);
  if (app.slice(ini - 6, ini) === 'async ') ini -= 6;
  let i = app.indexOf('{', app.indexOf(')', ini)), n = 0;
  for (; i < app.length; i++) { if (app[i] === '{') n++; else if (app[i] === '}' && --n === 0) break; }
  return app.slice(ini, i + 1);
}
const fonte = ['escHtml', 'esc', 'fmtMoeda', 'fmtDiaMes', 'fmtDataCompleta', 'rotuloMesCurto', 'textoDias', 'linhaCfg', 'tituloRecolhivel', 'conferenciaHtml',
  'retratoCartao', 'apagarMarcas', 'conferirMarcas', 'alternarSelecaoCartao', 'alternarMarcaCompra', 'focarNoCartao', 'painelCartaoHtml', 'renderCartao',
  'carregarFatura', 'carregarCartoes', 'recarregarCartao', 'editarCampoCartao', 'valorAtualDoCampo', 'erroCartao', 'onCartaoClick'].map(funcao).join('\n');

const copia = x => JSON.parse(JSON.stringify(x));
function montar({ dono = true } = {}) {
  const compra = (id, extra) => Object.assign({ id, dia: '2026-11-03', descricao: 'Compra ' + id, valor: 10, ajuste: 0, fatura: '2026-11', base: '2026-11', origem: null, estimado: false, podeMexer: true }, extra);
  const fatura = (meio, ym, lancamentos) => ({
    meio, ym, periodo: { ini: '2026-10-25', fim: '2026-11-24' }, anterior: '2026-10', proxima: '2026-12', chips: [{ ym, rotulo: 'Atual' }],
    fechada: false, historico: false, inicio: null, total: 20, valorBanco: null, diferenca: null, conferida: null, vencimento: null, pagador: null,
    sugestoes: [], lancamentos, pagamentos: [], totalPago: 0,
  });
  const servidor = {
    cartoes: ['Bradesco', 'Outro'].map(meio => ({ meio, fatura: { fechamento: '2026-11-24', diasAteFechar: 3, total: 20 }, conferidasQueMudaram: 0 })),
    faturas: {
      'Bradesco|2026-11': fatura('Bradesco', '2026-11', [compra('a'), compra('b', { origem: 'foi', fatura: '2026-12', ocupaOutraFatura: '2026-12' })]),
      'Bradesco|2026-10': fatura('Bradesco', '2026-10', [compra('c')]),
      'Outro|2026-11': fatura('Outro', '2026-11', [compra('d')]),
    },
    falhar: false,
  };
  const els = {};
  const el = id => els[id] || (els[id] = { id, innerHTML: '', textContent: '', value: '', classList: { toggle() {}, add() {}, remove() {} } });
  const ctx = {
    console: { error() {} }, document: { getElementById: el, querySelectorAll: () => [] },
    cartoesResumo: null, cartaoFatura: null, cartaoAbertoMeio: '', cartaoYmSel: '', cartaoRecSel: null, cartaoMarcas: null,
    cartaoCfgAberto: false, cartaoComprasAberto: true, cartaoPagAberto: true, cartaoCarregando: false, cartaoUltimaCalc: 0, cartaoErro: '', msgCartao: '', rascunhoFech: null,
    listas: { banco: [] }, lancamentosCache: [], ehOwner: () => dono, marcaDesativado: () => '', marcarMeiosCartao() {}, aplicarLocal() {},
    Recorrencia: { fraseAviso: () => '' }, MESES_NOME: ['janeiro','fevereiro','marco','abril','maio','junho','julho','agosto','setembro','outubro','novembro','dezembro'],
    CAMPOS_CARTAO: { valorBanco: { titulo: 'Valor', tipo: 'numero' } },
    resposta: undefined, pedirCampoCartao: async () => ctx.resposta, avisar: async () => {}, confirmarModal: async () => true,
    db: {
      cartoes: async () => copia({ cartoes: servidor.cartoes, total: 40, proximoFechamentoEmDias: 3, disponiveis: [] }),
      cartaoFatura: async (meio, ym) => copia(servidor.faturas[meio + '|' + (ym || '2026-11')]),
      cartaoFechamento: async () => { if (servidor.falhar) throw new Error('sem rede'); },
      cartaoRemover: async () => {},
    },
  };
  vm.createContext(ctx);
  vm.runInContext(fonte, ctx);
  const clicar = dataset => ctx.onCartaoClick({ detail: 1, target: { closest: () => ({ dataset }) } });
  const tela = () => el('listaCartoes').innerHTML;
  return { ctx, servidor, clicar, tela };
}
const marcadas = ctx => ctx.cartaoMarcas ? [...ctx.cartaoMarcas.ids].sort().join(',') : null;
const conta = (h, re) => (h.match(re) || []).length;

async function aberto(opcoes) {
  const t = montar(opcoes);
  await t.ctx.carregarCartoes(true);
  await t.clicar({ cartao: 'Bradesco' });
  return t;
}
async function comMarcas() {
  const t = await aberto();
  await t.clicar({ marcar: '1' });
  await t.clicar({ rec: 'a' });
  await t.clicar({ rec: 'b' });
  assert.strictEqual(marcadas(t.ctx), 'a,b');
  return t;
}

(async () => {
  // fora da selecao a lista e a de sempre: sem circulo, sem papel de caixa, tocar abre as acoes
  for (const dono of [true, false]) {
    const t = await aberto({ dono });
    assert.ok(/data-marcar="1">Selecionar</.test(t.tela()), 'link Selecionar, dono=' + dono);
    assert.ok(!/lote-ck|role="checkbox"/.test(t.tela()), 'sem circulo fora do modo');
    await t.clicar({ rec: 'a' });
    assert.ok(/fx-acoes/.test(t.tela()), 'tocar abre as acoes');
    // entrar no modo fecha as acoes e cada linha vira caixa; tocar marca, tocar de novo desmarca
    await t.clicar({ marcar: '1' });
    assert.ok(/data-marcar="1">Selecionados · 0</.test(t.tela()));
    assert.strictEqual(conta(t.tela(), /role="checkbox" aria-checked="false" tabindex="0"/g), 2);
    assert.strictEqual(conta(t.tela(), /class="lote-ck" aria-hidden="true"/g), 2);
    assert.ok(!/fx-acoes/.test(t.tela()));
    await t.clicar({ rec: 'a' });
    assert.ok(/Selecionados · 1</.test(t.tela()) && /aria-checked="true"/.test(t.tela()) && /class="lote-ck on"/.test(t.tela()));
    assert.ok(!/fx-acoes/.test(t.tela()), 'no modo as acoes nao abrem');
    await t.clicar({ rec: 'a' });
    assert.strictEqual(marcadas(t.ctx), '');
    // a compra fora da fatura tambem marca e continua riscada
    await t.clicar({ rec: 'b' });
    assert.ok(/class="fx-rec fora sel"/.test(t.tela()) && /line-through/.test(t.tela()));
  }

  // MANTEM: recolher e abrir secoes, fechar e reabrir o mesmo cartao, recarregar sem mudanca, cancelar a janela de um campo, salvar que falha
  let t = await comMarcas();
  await t.clicar({ tg: 'compras' });
  assert.ok(!/data-marcar/.test(t.tela()), 'recolhida: sem o link');
  assert.strictEqual(marcadas(t.ctx), 'a,b', 'recolher mantem');
  await t.clicar({ tg: 'compras' });
  assert.ok(/Selecionados · 2</.test(t.tela()), 'ao abrir volta com a contagem');
  await t.clicar({ tg: 'cfg' }); await t.clicar({ tg: 'pag' });
  await t.clicar({ cartao: 'Bradesco' }); await t.clicar({ cartao: 'Bradesco' });
  assert.strictEqual(marcadas(t.ctx), 'a,b', 'fechar e reabrir mantem');
  await t.ctx.carregarCartoes(true);
  assert.strictEqual(marcadas(t.ctx), 'a,b', 'voltar sem mudanca mantem');
  await t.clicar({ chip: '2026-11' });
  assert.strictEqual(marcadas(t.ctx), 'a,b', 'tocar na fatura ja aberta mantem');
  t.ctx.resposta = undefined; await t.clicar({ ed: 'valorBanco', ym: '2026-11' });
  assert.strictEqual(marcadas(t.ctx), 'a,b', 'cancelar mantem');
  t.ctx.resposta = '5'; t.servidor.falhar = true; await t.clicar({ ed: 'valorBanco', ym: '2026-11' });
  assert.strictEqual(marcadas(t.ctx), 'a,b', 'salvar que falhou nao mudou nada');

  // APAGA: Selecionados, salvar um campo, trocar de fatura ou de cartao, dado diferente, remover o cartao
  const apaga = async (nome, acao) => {
    const x = await comMarcas();
    await acao(x);
    assert.strictEqual(x.ctx.cartaoMarcas, null, 'apaga: ' + nome);
    assert.ok(!/lote-ck/.test(x.tela()), 'desliga o modo: ' + nome);
  };
  await apaga('Selecionados', x => x.clicar({ marcar: '1' }));
  await apaga('salvar um campo, mesmo sem mudar o dado', async x => { x.ctx.resposta = '5'; await x.clicar({ ed: 'valorBanco', ym: '2026-11' }); });
  await apaga('trocar de fatura', x => x.clicar({ nav: '2026-10' }));
  await apaga('trocar de cartao', x => x.clicar({ cartao: 'Outro' }));
  await apaga('compra alterada em outra tela', async x => { x.servidor.faturas['Bradesco|2026-11'].lancamentos[0].valor = 11; await x.ctx.carregarCartoes(true); });
  await apaga('configuracao do cartao alterada', async x => { x.servidor.cartoes[0].pagador = 'Itau'; await x.ctx.carregarCartoes(true); });
  await apaga('remover o cartao', x => x.clicar({ rm: 'Bradesco' }));
  await apaga('reabrir com outra fatura marcada', async x => {
    await x.clicar({ nav: '2026-10' }); await x.clicar({ marcar: '1' }); await x.clicar({ rec: 'c' });
    await x.clicar({ cartao: 'Bradesco' }); await x.clicar({ cartao: 'Bradesco' });
  });

  // sem compras nao ha link
  t = await aberto();
  t.servidor.faturas['Bradesco|2026-11'].lancamentos = [];
  await t.ctx.carregarCartoes(true);
  assert.ok(!/data-marcar/.test(t.tela()), 'sem compras: sem o link');

  // Atualizar, teclado e os outros comandos que gravam
  assert.ok(/btnAtualizarCartao'\)\.addEventListener\('click', \(\)=>\{ apagarMarcas\(\); carregarCartoes\(true\); \}\)/.test(app), 'Atualizar apaga');
  assert.ok(/e\.target\.matches\('\.fx-rec\[role="checkbox"\]'\)\)\{ e\.preventDefault\(\); e\.target\.click\(\); \}/.test(app), 'Enter e Espaco marcam');
  for (const comando of ['cartaoDesconferir(meio, ym)', 'cartaoDiferenca(meio, ym)', 'cartaoAdicionar(banco, dia)', 'cartaoConfig(meio, {aviso: aviso || null})']) {
    const i = app.indexOf('await db.' + comando);
    assert.ok(i >= 0 && app.slice(i, i + 200).includes('apagarMarcas()'), 'apaga depois de ' + comando);
  }
  // nada e gravado: o marcador nao toca no aparelho nem no servidor
  const doMarcador = ['retratoCartao', 'apagarMarcas', 'conferirMarcas', 'alternarSelecaoCartao', 'alternarMarcaCompra', 'focarNoCartao'].map(funcao).join('\n');
  assert.ok(!/localStorage|sessionStorage|indexedDB|\bdb\.|fetch\(|console\./.test(doMarcador), 'so memoria');

  // CSS: so o circulo muda; vazio sem o ✓; na compra fora da fatura o circulo fica com a cor inteira
  assert.ok(/\.fx-rec \.lote-ck\{color:transparent;\}/.test(css));
  assert.ok(/\.fx-rec\.fora\.sel\{opacity:1;\}/.test(css));
  assert.ok(!/\.fx-rec\.marcada/.test(css));

  console.log('cartao-conferencia: ok');
})().catch(e => { console.error(e); process.exit(1); });
