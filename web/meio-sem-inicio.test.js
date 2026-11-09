// Teste do item 60 (demos demo-listas-v7-meio-novo e demo-saldos-v4-sem-inicio): meio novo nasce com inicio do controle hoje (o SERVIDOR cria; o front so avisa) e
// meio sem inicio nao e calculado (o servidor marca naoCalculado e diz o motivo; o front so exibe).
// Rode com: node web/meio-sem-inicio.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');

const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const api = fs.readFileSync(path.join(__dirname, 'api.js'), 'utf8');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
const css = fs.readFileSync(path.join(__dirname, 'css', 'app.css'), 'utf8');

// 1. o front nao cria o inicio do controle: so le meiosNovos da resposta do PUT de Listas
assert.ok(/return r\.body; \/\/ em listas, traz meiosNovos/.test(api), 'gravarCfg devolve o corpo da resposta');
assert.ok(/const r = await gravarCfg\(nome, obj\); atualizarOuvintes\('doc', nome\); return r;/.test(api), 'doc().set devolve a resposta');
assert.ok(/const novos = \(r && r\.meiosNovos\) \|\| \[\];/.test(app), 'salvarListas le meiosNovos');
assert.ok(!/dataInicio:\s*(hoje|new Date|nowLocal)/.test(app), 'o front nao decide a data de inicio de um meio novo');
assert.ok(/saldosUltimaCalc = 0;/.test(app.match(/async function salvarListas[\s\S]*?\n\}/)[0]), 'meio novo forca o recalculo dos saldos');

// 2. os tres caminhos que criam meio avisam: Listas (cartao fixo), IA do Lancar (aviso rapido) e importacao (resumo)
assert.ok(/id="avisoNovo"/.test(html) && /id="avisoNovoImport"/.test(html), 'os dois cartoes de aviso existem');
assert.ok(/salvarListas\(\)\.then\(avisoMeioNovo\)/.test(app), 'Adicionar mostra o aviso do servidor');
assert.ok(/começa a contar saldo hoje \(\$\{esc\(fmtDataBR\(n\.dataInicio\)\)\}\)/.test(app), 'texto do aviso (demo)');
assert.ok(/Lançamentos de antes dessa data ficam no histórico e não entram no saldo\. Para mudar, defina a data de início em Saldos\./.test(app), 'o aviso explica o que se perde');
assert.ok(/Ir para Saldos/.test(app) && /Entendi/.test(app), 'o aviso tem Ir para Saldos e Entendi');
assert.ok(/Meio \$\{novos\[0\]\.nome\} criado\. Começa a contar saldo hoje/.test(app), 'IA do Lancar avisa com toast');
assert.ok(/meiosNovosImport = await salvarListas\(\)/.test(app) && /avisoMeiosNovosImport\(meiosNovosImport\)/.test(app), 'a importacao mostra o resumo');

// 3. Saldos: o servidor marca naoCalculado; a tela mostra a marca, "—", o motivo dele e deixa o meio de fora do total
assert.ok(/m\.naoCalculado \? \{saldo:0, naoCalculado:true, motivo:m\.motivo\|\|''\}/.test(app), 'Saldos le naoCalculado e o motivo do servidor');
assert.ok(!/m\.erro/.test(app), 'sumiu o erro por meio (o servidor nao le mais o passado)');
assert.ok(/r\.naoCalculado \? '—'/.test(app), 'o valor do meio nao calculado e "—"');
assert.ok(/class=\\"semini\\">sem início|class="semini">sem início/.test(app), 'marca "sem início"');
assert.ok(/<b>Saldo não calculado<\/b>\$\{esc\(rb\.motivo \|\| ''\)\}/.test(app), 'o painel mostra o motivo que o servidor mandou');
assert.ok(/sem início do controle \(fora do total\)/.test(app), 'o total diz que nao inclui o meio sem inicio');
assert.ok(/t\.naoCalculados\|\|0/.test(app) && !/incompleto:/.test(app.match(/function totalSaldos[\s\S]*?\n\}/)[0]), 'o total usa naoCalculados do servidor');
assert.ok(/r && !r\.naoCalculado \? r\.saldo : -Infinity/.test(app), 'na ordenacao por valor o nao calculado vai para o fim');

// 4. o css so usa tokens
for (const cls of ['semini', 'sd-val\\.nc', 'nc-aviso', 'aviso-novo']) {
  const m = css.match(new RegExp('\\.' + cls + '\\{([^}]*)\\}'));
  assert.ok(m, cls + ' existe no css');
  assert.ok(!/#[0-9a-fA-F]{3,6}\b/.test(m[1]), cls + ' so com tokens');
}

console.log('ok meio-sem-inicio.test.js');
