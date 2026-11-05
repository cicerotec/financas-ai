// Valor do Lançar (demo-lancar-v4-atalhos): digita-se em reais, a vírgula só serve para os centavos.
// As funções moram em js/app.js (que não é módulo); o teste as extrai do código-fonte.
const fs = require('fs'), assert = require('assert');
const src = fs.readFileSync(__dirname + '/js/app.js', 'utf8');
const ini = src.indexOf('function valorNum(s)');
const fim = src.indexOf('function poeValorForm');
assert(ini > 0 && fim > ini, 'funcoes do valor nao encontradas em app.js');
const { valorNum, fmtValorCampo, limparDigitacaoValor } = new Function(src.slice(ini, fim) + '; return {valorNum, fmtValorCampo, limparDigitacaoValor};')();

// digitação: ponto vira vírgula, no máximo 2 casas, "-" só no começo
assert.strictEqual(limparDigitacaoValor('450'), '450');
assert.strictEqual(limparDigitacaoValor('450,5'), '450,5');
assert.strictEqual(limparDigitacaoValor('45,678'), '45,67');
assert.strictEqual(limparDigitacaoValor('12.5'), '12,5');
assert.strictEqual(limparDigitacaoValor('1,2,3'), '1,23');
assert.strictEqual(limparDigitacaoValor('-12'), '-12');
assert.strictEqual(limparDigitacaoValor('a1b'), '1');

// leitura: reais sem vírgula valem reais; vazio e "-" são inválidos; zero e negativo valem
assert.strictEqual(valorNum('450'), 450);
assert.strictEqual(valorNum('450,5'), 450.5);
assert.strictEqual(valorNum('1.234,50'), 1234.5);
assert.strictEqual(valorNum('-12,00'), -12);
assert.strictEqual(valorNum('0,00'), 0);
assert(Number.isNaN(valorNum('')));
assert(Number.isNaN(valorNum('-')));

// exibição ao sair do campo
assert.strictEqual(fmtValorCampo(450), '450,00');
assert.strictEqual(fmtValorCampo(450.5), '450,50');
assert.strictEqual(fmtValorCampo(0), '0,00');
assert.strictEqual(fmtValorCampo(NaN), '');
assert.strictEqual(fmtValorCampo(-12), '-12,00');
console.log('valor.test.js ok');
