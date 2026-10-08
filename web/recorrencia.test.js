// Testes da logica de repeticao dos eventos futuros. Rodar: node web/recorrencia.test.js
// Usa o node:test quando existe (Node 18+, o do CI) e um executor minimo no Node mais antigo.
const assert = require('node:assert/strict');
let test;
try { test = require('node:test'); } catch (e) {
  test = (nome, fn) => {
    try { fn(); console.log('ok     ' + nome); }
    catch (err) { console.error('FALHOU ' + nome + '\n' + err.stack); process.exitCode = 1; }
  };
}
const R = require('./recorrencia.js');

// data local legivel: 2026-10-31 12:30
const d = (a, m, dia, h = 12, mi = 30) => new Date(a, m - 1, dia, h, mi, 0, 0);
const txt = (dt) => R.diaLocal(dt) + ' ' + String(dt.getHours()).padStart(2, '0') + ':' + String(dt.getMinutes()).padStart(2, '0');

test('somarMeses: dia que nao existe vira o ultimo do mes', () => {
  assert.equal(R.diaLocal(R.somarMeses(d(2027, 1, 31), 1)), '2027-02-28');
  assert.equal(R.diaLocal(R.somarMeses(d(2028, 1, 31), 1)), '2028-02-29'); // bissexto
  assert.equal(R.diaLocal(R.somarMeses(d(2026, 1, 31), 2)), '2026-03-31');
  assert.equal(R.diaLocal(R.somarMeses(d(2026, 1, 31), 3)), '2026-04-30');
  assert.equal(R.diaLocal(R.somarMeses(d(2026, 10, 31), 1)), '2026-11-30');
});

test('somarMeses: virada de ano, anual e hora preservada', () => {
  assert.equal(txt(R.somarMeses(d(2026, 11, 15, 8, 5), 3)), '2027-02-15 08:05');
  assert.equal(R.diaLocal(R.somarMeses(d(2026, 6, 10), 12)), '2027-06-10');
  assert.equal(R.diaLocal(R.somarMeses(d(2028, 2, 29), 12)), '2029-02-28'); // 29/02 anual
  assert.equal(R.diaLocal(R.somarMeses(d(2028, 2, 29), 48)), '2032-02-29'); // volta a existir
  assert.equal(R.diaLocal(R.somarMeses(d(2026, 5, 20), 0)), '2026-05-20');
});

test('gerarDatas parte sempre da data original (31/01 mensal: 28/02, 31/03, 30/04)', () => {
  assert.deepEqual(R.gerarDatas(d(2026, 10, 31), 1, 3).map(R.diaLocal), ['2026-11-30', '2026-12-31', '2027-01-31']);
  assert.deepEqual(R.gerarDatas(d(2027, 1, 31), 1, 3).map(R.diaLocal), ['2027-02-28', '2027-03-31', '2027-04-30']);
  assert.deepEqual(R.gerarDatas(d(2026, 1, 15), 6, 3).map(R.diaLocal), ['2026-07-15', '2027-01-15', '2027-07-15']);
  assert.deepEqual(R.gerarDatas(d(2026, 3, 1), 12, 2).map(R.diaLocal), ['2027-03-01', '2028-03-01']);
  assert.equal(R.gerarDatas(d(2026, 1, 1), 1, 0).length, 0);
});

test('validar: limites do intervalo e da quantidade', () => {
  assert.equal(R.validar(1, 1), null);
  assert.equal(R.validar(120, R.MAX_OCORRENCIAS), null);
  for (const [i, q] of [[0, 3], [121, 3], [1.5, 3], [1, 0], [1, R.MAX_OCORRENCIAS + 1], [1, 2.5], ['1', 3], [NaN, 3]])
    assert.notEqual(R.validar(i, q), null, `(${i}, ${q}) deveria ser invalido`);
});

test('rotuloIntervalo', () => {
  assert.equal(R.rotuloIntervalo(1), 'mensal');
  assert.equal(R.rotuloIntervalo(3), 'trimestral');
  assert.equal(R.rotuloIntervalo(6), 'semestral');
  assert.equal(R.rotuloIntervalo(12), 'anual');
  assert.equal(R.rotuloIntervalo(2), 'a cada 2 meses');
});

test('parcelas: leitura, restantes e descricao numerada', () => {
  assert.deepEqual(R.lerParcela('Notebook 2/10'), { a: 2, b: 10 });
  assert.deepEqual(R.lerParcela('Geladeira 1 / 3 no cartao'), { a: 1, b: 3 });
  assert.equal(R.lerParcela('Aluguel'), null);
  assert.equal(R.lerParcela('Caso 5/3'), null);  // parcela maior que o total
  assert.equal(R.lerParcela('Caso 0/3'), null);
  assert.equal(R.lerParcela(''), null);
  assert.equal(R.copiasRestantesNaParcela('Notebook 2/10'), 8);
  assert.equal(R.copiasRestantesNaParcela('Notebook 10/10'), 0);
  assert.equal(R.copiasRestantesNaParcela('Aluguel'), null);
  assert.equal(R.descricaoNumerada('Notebook 2/10', 3), 'Notebook 5/10');
  assert.equal(R.descricaoNumerada('Geladeira 1 / 3 no cartao', 1), 'Geladeira 2/3 no cartao');
  assert.equal(R.descricaoNumerada('Aluguel', 4), 'Aluguel');
});

test('chaveDuplicado ignora hora, caixa e espacos, e separa dias e valores', () => {
  const a = { status: 'PREVISTO', descricao: ' Luz ', valor: 10, banco: 'ITAU', dataEvento: d(2026, 10, 10, 9, 0).toISOString() };
  const b = { ...a, descricao: 'luz', banco: 'itau', dataEvento: d(2026, 10, 10, 21, 45).toISOString() };
  assert.equal(R.chaveDuplicado(a), R.chaveDuplicado(b));
  assert.notEqual(R.chaveDuplicado(a), R.chaveDuplicado({ ...b, dataEvento: d(2026, 10, 11, 9, 0).toISOString() }));
  assert.notEqual(R.chaveDuplicado(a), R.chaveDuplicado({ ...b, valor: 10.01 }));
  assert.notEqual(R.chaveDuplicado(a), R.chaveDuplicado({ ...b, status: 'PAGO' }));
});

const base = () => ({ status: 'PAGO', tipo: 'saida', valor: 99.9, descricao: 'Seguro', nota: 'apolice 7', banco: 'ITAU',
  tags: ['Casa'], dataEvento: d(2026, 10, 31, 8, 0).toISOString() });

test('montarCopias: campos, datas, serie e aviso', () => {
  const aviso = { dias: [3, 0], insistir: true };
  const c = R.montarCopias(base(), { intervaloMeses: 1, quantidade: 3, status: 'PREVISTO', aviso, numerar: false, serieId: 'serie1abc', indiceInicial: 1 });
  assert.equal(c.length, 3);
  assert.deepEqual(c.map((x) => R.diaLocal(new Date(x.dataEvento))), ['2026-11-30', '2026-12-31', '2027-01-31']);
  assert.ok(c.every((x) => x.status === 'PREVISTO' && x.tipo === 'saida' && x.valor === 99.9 && x.banco === 'ITAU' && x.nota === 'apolice 7'));
  assert.deepEqual(c.map((x) => x.serie), [
    { id: 'serie1abc', indice: 1, intervalo: 1 }, { id: 'serie1abc', indice: 2, intervalo: 1 }, { id: 'serie1abc', indice: 3, intervalo: 1 }]);
  assert.deepEqual(c[0].aviso, aviso);
  c[0].aviso.dias.push(9); c[0].tags.push('x');
  assert.deepEqual(c[1].aviso.dias, [3, 0], 'cada copia tem o seu proprio aviso');
  assert.deepEqual(c[1].tags, ['Casa'], 'cada copia tem as suas proprias tags');
  assert.deepEqual(aviso.dias, [3, 0], 'o aviso do original nao e alterado');
  assert.equal(c[0].excluirDoTotal, false);
});

test('montarCopias: sem aviso nao leva o campo; TRANSFERINDO fica fora do total; indice continua', () => {
  const c = R.montarCopias(base(), { intervaloMeses: 12, quantidade: 2, status: 'TRANSFERINDO', aviso: null, numerar: false, serieId: 's', indiceInicial: 5 });
  assert.ok(c.every((x) => !('aviso' in x) && x.excluirDoTotal === true));
  assert.deepEqual(c.map((x) => x.serie.indice), [5, 6]);
  assert.deepEqual(c.map((x) => R.diaLocal(new Date(x.dataEvento))), ['2027-10-31', '2028-10-31']);
});

test('montarCopias: numeracao de parcelas continua e para no total', () => {
  const b = { ...base(), descricao: 'Notebook 8/10' };
  const c = R.montarCopias(b, { intervaloMeses: 1, quantidade: 6, status: 'PREVISTO', aviso: null, numerar: true, serieId: 's', indiceInicial: 1 });
  assert.deepEqual(c.map((x) => x.descricao), ['Notebook 9/10', 'Notebook 10/10']); // so cabem 2
  const sem = R.montarCopias(b, { intervaloMeses: 1, quantidade: 3, status: 'PREVISTO', aviso: null, numerar: false, serieId: 's', indiceInicial: 1 });
  assert.deepEqual(sem.map((x) => x.descricao), ['Notebook 8/10', 'Notebook 8/10', 'Notebook 8/10']);
  const semParcela = R.montarCopias(base(), { intervaloMeses: 1, quantidade: 2, status: 'PREVISTO', aviso: null, numerar: true, serieId: 's', indiceInicial: 1 });
  assert.equal(semParcela.length, 2, 'sem "n/m" na descricao, numerar nao limita');
});

test('montarCopias recusa parametros invalidos', () => {
  const op = { intervaloMeses: 1, quantidade: 3, status: 'PREVISTO', aviso: null, numerar: false, serieId: 's', indiceInicial: 1 };
  assert.throws(() => R.montarCopias(base(), { ...op, quantidade: 0 }));
  assert.throws(() => R.montarCopias(base(), { ...op, quantidade: R.MAX_OCORRENCIAS + 1 }));
  assert.throws(() => R.montarCopias(base(), { ...op, intervaloMeses: 0 }));
});

test('ultimaDaSerie acha a de maior indice, so da serie pedida', () => {
  const l = [
    { id: 'a', serie: { id: 'x', indice: 2 } }, { id: 'b', serie: { id: 'x', indice: 5 } },
    { id: 'c', serie: { id: 'y', indice: 9 } }, { id: 'd' }, { id: 'e', serie: { id: 'x', indice: 3 } }];
  assert.equal(R.ultimaDaSerie(l, 'x').id, 'b');
  assert.equal(R.ultimaDaSerie(l, 'y').id, 'c');
  assert.equal(R.ultimaDaSerie(l, 'z'), null);
});
