// Teste do item 43 (.ai/specs/meios-de-pagamento): o front so exibe e pede. Quem diz se um meio esta ativo e se pode ser removido e o servidor
// (GET /cfg/listas: meios[].ativo; DELETE /meios/{meio}). Este teste falha se o navegador voltar a conferir o uso de um meio lendo lancamentos,
// se esquecer o meio de um lancamento antigo no select ou se perder o ativo ao atualizar a marca de cartao.
// Rode com: node web/meios.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');

const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const api = fs.readFileSync(path.join(__dirname, 'api.js'), 'utf8');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');

// 1. nao existe mais a conferencia de uso no navegador (lia ate 1000 lancamentos)
assert.ok(!/function removerBanco\b/.test(app), 'removerBanco saiu: a conferencia de uso e do servidor');
assert.ok(!/data-del-banco/.test(app) && !/data-del-banco/.test(html), 'o X antigo de banco saiu');

// 2. as chamadas novas existem e usam as rotas do servidor
assert.ok(/'\/meios\/'/.test(api), 'api.js chama /meios/{meio}');
assert.ok(/meioDefinir/.test(api) && /meioRemover/.test(api), 'api.js expoe meioDefinir e meioRemover');
assert.ok(/db\.meioRemover\(/.test(app) && /db\.meioDefinir\(/.test(app), 'a tela chama o servidor');

// 3. so os meios ativos sao sugeridos (Lancar e Transferencia), mas o meio de um lancamento em edicao ou copia continua no select
assert.ok(/refazer\(sel, meiosSugeridos\(\)\)/.test(app), 'os selects de banco usam so os ativos');
assert.strictEqual((app.match(/opcaoMeio\(document\.getElementById\('fBanco'\), l\.banco\)/g) || []).length, 2, 'editar e copiar garantem a opcao do meio antigo');

// 4. atualizar a marca de cartao nao zera o ativo que o servidor mandou
assert.ok(/Object\.assign\(\{\}, meioDe\(n\), \{nome:n, cartao:cartoes\.has\(n\)\}\)/.test(app), 'marcarMeiosCartao preserva ativo e ativoAte');
assert.ok(!/ativo:true\}\)\)/.test(app), 'nenhum meio e marcado ativo no navegador');

// 5. a etiqueta e "credito" (vem da aba Cartao), nao "cartao"
assert.ok(/class="sel">crédito</.test(app), 'etiqueta crédito');

// 6. a aba Saldos nao mostra o meio desativado (o servidor nem o calcula) e a contagem so conta os ativos
assert.ok(/listas\.banco\.map\(\(_,i\)=>i\)\.filter\(i=>meioAtivo\(listas\.banco\[i\]\)\)/.test(app), 'indicesOrdenados so lista os ativos');
assert.ok(/\$\{meiosSugeridos\(\)\.length\} banco/.test(app), 'o subtitulo de Saldos conta so os ativos');

assert.ok(/listas\.banco\.filter\(x=>meioAtivo\(x\) \|\| x===atual\)/.test(app), 'Paga com sugere so os ativos, mantendo o atual');

console.log('ok meios.test.js');
