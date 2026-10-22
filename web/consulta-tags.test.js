// Testes da consulta por tags da aba Tendencias. Rodar: node web/consulta-tags.test.js
// Usa o node:test quando existe (Node 18+, o do CI) e um executor minimo no Node mais antigo.
const assert = require('node:assert/strict');
let test;
try { test = require('node:test'); } catch (e) {
  test = (nome, fn) => {
    try { fn(); console.log('ok     ' + nome); }
    catch (err) { console.error('FALHOU ' + nome + '\n' + err.stack); process.exitCode = 1; }
  };
}
const C = require('./consulta-tags.js');

const F = (...a) => a.map((x) => (x === 'E' || x === 'OU' || x === 'NAO' ? { k: 'op', v: x } : x === '(' || x === ')' ? { k: x } : { k: 'tag', v: x }));
const G = (tem, nao) => ({ tem, nao: nao || [] });
const grupos = (...g) => ({ modo: 'grupos', grupos: g, fichas: [] });
const expr = (...f) => ({ modo: 'avancado', grupos: [], fichas: F(...f) });
const li = (mes, valor, ...tags) => ({ mes, valor, tags });
const msg = (...f) => C.consulta(expr(...f)).msg;

test('precedencia: NAO antes de E, E antes de OU', () => {
  assert.equal(C.texto(C.analisar(F('A', 'E', 'B', 'OU', 'C'))), '(A E B) OU C');
  assert.equal(C.texto(C.analisar(F('A', 'OU', 'B', 'E', 'C'))), 'A OU (B E C)');
  assert.equal(C.texto(C.analisar(F('A', 'E', 'NAO', '(', 'B', 'OU', 'C', ')'))), '(A E NÃO (B OU C))');
  assert.equal(C.texto(C.analisar(F('NAO', 'NAO', 'A'))), 'NÃO NÃO A');
});

test('avalia a expressao sobre as tags do lancamento', () => {
  const n = C.analisar(F('A', 'E', 'B', 'OU', 'C'));
  assert.equal(C.avalia(n, ['A', 'B']), true);
  assert.equal(C.avalia(n, ['A']), false);
  assert.equal(C.avalia(n, ['C', 'X']), true);
  const m = C.analisar(F('A', 'E', 'NAO', '(', 'B', 'OU', 'C', ')'));
  assert.equal(C.avalia(m, ['A']), true);
  assert.equal(C.avalia(m, ['A', 'C']), false);
  assert.equal(C.avalia(m, ['B']), false);
});

test('erros dizem o que falta', () => {
  assert.equal(msg('(', 'A'), 'Falta fechar um parêntese.');
  assert.equal(msg('A', 'E'), 'Falta uma tag depois de "E".');
  assert.equal(msg('A', 'B'), 'Falta E ou OU entre "A" e "B".');
  assert.equal(msg('(', ')'), 'Parêntese vazio: coloque tags dentro dele.');
  assert.equal(msg('A', ')'), 'Tem ")" sem "(" antes.');
  assert.equal(msg('OU', 'A'), 'Falta uma tag antes de "OU".');
  assert.equal(msg('NAO'), 'Falta uma tag depois de "NÃO".');
  assert.equal(msg('A', 'E', ')'), 'Falta uma tag depois de "E".');
  assert.equal(msg('A', 'E', 'NAO'), 'Falta uma tag depois de "NÃO".');
});

test('sem nada escolhido tudo passa, nos dois modos', () => {
  for (const est of [grupos(), grupos(G([])), { modo: 'avancado', grupos: [], fichas: [] }]) {
    const q = C.consulta(est);
    assert.equal(q.valida, true); assert.equal(q.vazia, true);
    assert.equal(q.pred(['X']), true); assert.equal(q.pred([]), true);
  }
});

test('grupo: E nas tags, NAO nas proibidas, e tags a mais no lancamento nao atrapalham', () => {
  const q = C.consulta(grupos(G(['Lazer', 'Comida'], ['Moradia'])));
  assert.equal(q.pred(['Lazer', 'Comida']), true);
  assert.equal(q.pred(['Lazer', 'Comida', 'Variavel']), true);
  assert.equal(q.pred(['Lazer', 'Comida', 'Moradia']), false);
  assert.equal(q.pred(['Lazer']), false);
});

test('grupo so com NAO significa tudo menos essas; grupo vazio e ignorado', () => {
  const q = C.consulta(grupos(G([], ['Presente']), G([])));
  assert.equal(q.vazia, false);
  assert.equal(q.pred(['A']), true);
  assert.equal(q.pred(['Presente']), false);
});

test('OU entre grupos conta o lancamento uma vez so', () => {
  const est = grupos(G(['Lazer', 'Comida']), G(['Variavel']));
  const r = C.resumir([li('2026-07', 100, 'Lazer', 'Comida', 'Variavel'), li('2026-07', 50, 'Variavel'), li('2026-07', 9, 'Fixo')], C.consulta(est));
  assert.equal(r.n, 2); assert.equal(r.total, 150);
});

test('mencionadas: tudo o que a consulta cita, inclusive o NAO', () => {
  assert.deepEqual([...C.consulta(grupos(G(['A', 'B'], ['C']), G(['D']))).mencionadas].sort(), ['A', 'B', 'C', 'D']);
  assert.deepEqual([...C.consulta(expr('A', 'E', 'NAO', '(', 'B', 'OU', 'C', ')')).mencionadas].sort(), ['A', 'B', 'C']);
});

test('resumir: barra do mes = so as tags da consulta + com outras tags, ordenada por mes', () => {
  const itens = [
    li('2026-08', 60, 'Lazer', 'Comida'),
    li('2026-07', 70, 'Lazer', 'Comida', 'Variavel'),
    li('2026-07', 1350, 'Lazer', 'Comida', 'Moradia', 'Limpeza'),
    li('2026-07', 30, 'Lazer', 'Comida'),
    li('2026-07', 500, 'Mercado')
  ];
  const r = C.resumir(itens, C.consulta(grupos(G(['Lazer', 'Comida']))));
  assert.deepEqual(r.meses.map((m) => m.mes), ['2026-07', '2026-08']);
  assert.deepEqual(r.meses[0], { mes: '2026-07', total: 1450, ext: 1420 });
  assert.deepEqual(r.meses[1], { mes: '2026-08', total: 60, ext: 0 });
  assert.equal(r.n, 4); assert.equal(r.total, 1510); assert.equal(r.totExt, 1420);
  // as outras tags se sobrepoem: nao somam o total
  assert.deepEqual(r.cot, [{ t: 'Limpeza', v: 1350 }, { t: 'Moradia', v: 1350 }, { t: 'Variavel', v: 70 }]);
});

test('resumir: sem filtro, lancamento sem tag e vazia', () => {
  const r = C.resumir([li('2026-07', 10), li('2026-07', 5, 'A')], C.consulta(grupos()));
  assert.equal(r.vazia, true); assert.equal(r.n, 2); assert.equal(r.total, 15);
  assert.equal(r.totExt, 0); // sem filtro nao ha 'outras tags'
  assert.deepEqual(r.cot, []);
});

test('resumir: nada casa devolve zeros', () => {
  const r = C.resumir([li('2026-07', 10, 'A')], C.consulta(grupos(G(['Z']))));
  assert.equal(r.n, 0); assert.equal(r.total, 0); assert.deepEqual(r.meses, []); assert.deepEqual(r.cot, []);
});

test('gruposParaFichas traduz a consulta de grupos', () => {
  const nomes = (fs) => fs.map(C.nomeFicha).join(' ');
  assert.equal(nomes(C.gruposParaFichas([G(['A', 'B'], ['C'])])), 'A E B E NÃO C');
  assert.equal(nomes(C.gruposParaFichas([G(['A', 'B']), G(['C']), G([])])), '( A E B ) OU C');
  assert.equal(nomes(C.gruposParaFichas([G([], ['C'])])), 'NÃO C');
  // a traducao tem o mesmo significado que os grupos
  const est = grupos(G(['A', 'B'], ['C']), G(['D']));
  const igual = C.consulta({ modo: 'avancado', grupos: [], fichas: C.gruposParaFichas(est.grupos) });
  const base = C.consulta(est);
  [['A', 'B'], ['A', 'B', 'C'], ['D'], ['A'], ['D', 'C'], []].forEach((t) => assert.equal(igual.pred(t), base.pred(t), t.join()));
});

test('inserirFicha: E implicito depois de tag ou de ")"', () => {
  let r = C.inserirFicha([], 0, { k: 'tag', v: 'A' });
  r = C.inserirFicha(r.fichas, r.pos, { k: 'tag', v: 'B' });
  assert.equal(r.fichas.map(C.nomeFicha).join(' '), 'A E B');
  assert.equal(r.pos, 3);
  r = C.inserirFicha(r.fichas, r.pos, { k: 'op', v: 'OU' });
  r = C.inserirFicha(r.fichas, r.pos, { k: 'op', v: 'NAO' });
  assert.equal(r.fichas.map(C.nomeFicha).join(' '), 'A E B OU NÃO'); // depois de OU nao entra E
  const p = C.inserirFicha(F('(', 'A', ')'), 3, { k: 'tag', v: 'B' });
  assert.equal(p.fichas.map(C.nomeFicha).join(' '), '( A ) E B');
});

test('inserirFicha: respeita o limite de fichas', () => {
  const cheio = Array.from({ length: C.MAX_FICHAS }, () => ({ k: 'tag', v: 'A' }));
  const r = C.inserirFicha(cheio, cheio.length, { k: 'tag', v: 'B' });
  assert.equal(r.cheio, true); assert.equal(r.fichas.length, C.MAX_FICHAS);
});

test('tirarTag nos grupos: NAO em todos os grupos preenchidos, sem repetir', () => {
  const est = grupos(G(['A']), G(['B'], ['X']), G([]));
  const r = C.tirarTag(est, 'X');
  assert.equal(r.ok, true);
  assert.deepEqual(r.estado.grupos.map((g) => g.nao), [['X'], ['X'], []]);
  assert.deepEqual(est.grupos[0].nao, []); // o estado recebido nao muda
});

test('tirarTag nos grupos: lista de proibidas cheia nao deixa e nao muda nada', () => {
  const cheia = Array.from({ length: C.MAX_TAGS }, (_, i) => 'n' + i);
  const est = grupos(G(['A'], cheia));
  const r = C.tirarTag(est, 'X');
  assert.equal(r.ok, false);
  assert.match(r.erro, /8 tags/);
  assert.equal(est.grupos[0].nao.length, C.MAX_TAGS);
});

test('tirarTag na expressao: (expressao) E NAO tag, e tag unica nao leva parenteses', () => {
  const r = C.tirarTag(expr('A', 'OU', 'B'), 'X');
  assert.equal(r.estado.fichas.map(C.nomeFicha).join(' '), '( A OU B ) E NÃO X');
  assert.equal(C.tirarTag(expr('A'), 'X').estado.fichas.map(C.nomeFicha).join(' '), 'A E NÃO X');
  const q = C.consulta(r.estado);
  assert.equal(q.valida, true);
  assert.equal(q.pred(['A']), true); assert.equal(q.pred(['B', 'X']), false);
});

test('tirarTag na expressao: invalida ou no limite nao deixa', () => {
  assert.equal(C.tirarTag(expr('A', 'E'), 'X').ok, false);
  const longa = { modo: 'avancado', grupos: [], fichas: Array.from({ length: C.MAX_FICHAS - 2 }, (_, i) => (i % 2 ? { k: 'op', v: 'E' } : { k: 'tag', v: 'A' })) };
  longa.fichas.pop(); longa.fichas.push({ k: 'tag', v: 'A' });
  assert.equal(C.tirarTag(longa, 'X').ok, false);
});

test('frases', () => {
  assert.equal(C.fraseGrupos([G(['A', 'B'], ['C'])]), 'A E B NÃO C');
  assert.equal(C.fraseGrupos([G(['A', 'B']), G(['C'])]), '(A E B) OU (C)');
  assert.equal(C.fraseGrupos([G([])]), '');
});

// ---- visoes: gastos, receitas e saldo ----
const en = (mes, valor, ...tags) => ({ mes, valor, tags, entrada: true });
const qG = (...tem) => C.consulta(grupos(G(tem)));

test('visao padrao e gastos: soma so as saidas e ignora as entradas', () => {
  const itens = [li('2026-07', 100, 'A'), en('2026-07', 100, 'A'), li('2026-07', 50, 'A')];
  const r = C.resumir(itens, qG('A'));
  assert.equal(r.visao, 'gastos'); assert.equal(r.n, 2); assert.equal(r.total, 150);
  assert.equal(C.resumir(itens, qG('A'), 'qualquer coisa').visao, 'gastos'); // visao desconhecida cai em gastos
});

test('receitas: soma so as entradas', () => {
  const itens = [li('2026-07', 100, 'A'), en('2026-07', 70, 'A'), en('2026-08', 30, 'A')];
  const r = C.resumir(itens, qG('A'), 'receitas');
  assert.equal(r.n, 2); assert.equal(r.total, 100);
  assert.deepEqual(r.meses.map((m) => [m.mes, m.total]), [['2026-07', 70], ['2026-08', 30]]);
});

test('saldo: renda menos gasto, como o total do Historico (reembolso que cancela a compra da zero)', () => {
  const itens = [li('2026-09', 310, 'A', 'B'), en('2026-09', 310, 'A', 'B'), li('2026-09', 40, 'A')];
  const r = C.resumir(itens, qG('A'), 'saldo');
  assert.equal(r.visao, 'saldo'); assert.equal(r.n, 3); assert.equal(r.total, -40);
  const g = C.resumir(itens, qG('A'), 'gastos');
  assert.equal(g.total, 350); // em gastos o reembolso nao aparece
});

test('saldo: mes que se cancela aparece com zero; mes so de renda fica positivo', () => {
  const itens = [li('2026-06', 100, 'A'), en('2026-06', 100, 'A'), en('2026-07', 500, 'A')];
  const r = C.resumir(itens, qG('A'), 'saldo');
  assert.deepEqual(r.meses.map((m) => [m.mes, m.total]), [['2026-06', 0], ['2026-07', 500]]);
  assert.equal(r.total, 500);
});

test('saldo: a barra se divide por lancamento com sinal; o movimento (sem sinal) e a base das porcentagens', () => {
  const itens = [li('2026-07', 100, 'A'), li('2026-07', 300, 'A', 'X'), en('2026-07', 200, 'A', 'X')];
  const r = C.resumir(itens, qG('A'), 'saldo');
  assert.deepEqual(r.meses[0], { mes: '2026-07', total: -200, ext: -100 }); // sem outras tags: -100; com outras: -300 + 200
  assert.equal(r.totExt, -100);
  assert.equal(r.vol, 600); assert.equal(r.volExt, 500);
  assert.deepEqual(r.cot, [{ t: 'X', v: 500 }]); // peso da outra tag = movimento, nao o saldo
});

test('receitas e gastos: porcentagem e peso seguem o valor da propria visao', () => {
  const itens = [li('2026-07', 100, 'A'), li('2026-07', 100, 'A', 'X'), en('2026-07', 999, 'A', 'X')];
  const g = C.resumir(itens, qG('A'), 'gastos');
  assert.equal(g.total, 200); assert.equal(g.totExt, 100); assert.deepEqual(g.cot, [{ t: 'X', v: 100 }]);
  const rc = C.resumir(itens, qG('A'), 'receitas');
  assert.equal(rc.total, 999); assert.deepEqual(rc.cot, [{ t: 'X', v: 999 }]);
});

test('visoes: arredonda centavos (soma de decimais nao vaza erro de ponto flutuante)', () => {
  const itens = [li('2026-07', 0.1, 'A'), li('2026-07', 0.2, 'A'), en('2026-07', 0.3, 'A')];
  assert.equal(C.resumir(itens, qG('A'), 'gastos').total, 0.3);
  assert.equal(C.resumir(itens, qG('A'), 'saldo').total, 0); // 0.3 - 0.1 - 0.2
});

test('visoes: sem filtro mostra tudo e nao ha outras tags', () => {
  const itens = [li('2026-07', 10, 'A'), en('2026-07', 4)];
  const r = C.resumir(itens, C.consulta(grupos()), 'saldo');
  assert.equal(r.vazia, true); assert.equal(r.total, -6); assert.equal(r.totExt, 0); assert.deepEqual(r.cot, []);
});

// ---- base da Tendencias: o que conta ----
const lan = (extra) => Object.assign({ tipo: 'saida', status: 'PAGO', dataEvento: '2026-07-15T12:00:00', valor: 10, tags: ['A'] }, extra);
const real = (st) => st !== 'PREVISTO';

test('base: lancamento oculto fica de fora (ocultar tira do total, como no Historico)', () => {
  const r = C.base([lan(), lan({ oculto: true }), lan({ oculto: false })], real);
  assert.equal(r.length, 2);
});

test('base: status que nao e real fica de fora, quem decide e o ehReal do app', () => {
  assert.equal(C.base([lan(), lan({ status: 'PREVISTO' })], real).length, 1);
  assert.equal(C.base([lan(), lan({ status: 'PREVISTO' })], () => true).length, 2);
});

test('base: entrada e tudo que nao e saida; mes, valor e tags saem prontos', () => {
  const r = C.base([lan({ tipo: 'entrada', valor: '25.5', tags: undefined, dataEvento: '2026-09-02T09:00:00' }), lan({ tipo: 'saida' })], real);
  assert.deepEqual(r[0], { mes: '2026-09', valor: 25.5, tags: [], entrada: true });
  assert.equal(r[1].entrada, false);
});

test('base: o saldo da consulta nao conta o oculto (mesma regra do total do Historico)', () => {
  const itens = C.base([lan({ valor: 100 }), lan({ valor: 40, oculto: true }), lan({ tipo: 'entrada', valor: 30 })], real);
  assert.equal(C.resumir(itens, qG('A'), 'saldo').total, -70);
  assert.equal(C.resumir(itens, qG('A'), 'gastos').total, 100);
});
