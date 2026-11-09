// Cartao Tags de Listas (demo-listas-v9-tags): a logica da lista de tags com busca (web/lista-tags.js) e as duas regras do excluir:
// a janela abre no toque, antes de perguntar ao servidor quem usa a tag, e toda exclusao pede confirmacao.
// Rode com: node web/lista-tags.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const LT = require('./lista-tags.js');

// a ordem da lista e a ordem de criacao (a ultima e a mais nova)
const TAGS = ['mercado', 'casa', 'mercado-mensal', 'Farmácia', 'supermercado', 'viagem', 'Água', 'casa-mercado'];

// 1. busca: sem acento nem maiusculas, "#" no comeco e ignorado, casa por parte da palavra
assert.deepStrictEqual(LT.buscar(TAGS, 'farmacia'), ['Farmácia']);
assert.deepStrictEqual(LT.buscar(TAGS, '  #AGUA '), ['Água']);
assert.deepStrictEqual(LT.buscar(TAGS, 'zzz'), []);
assert.deepStrictEqual(LT.buscar(TAGS, ''), [], 'sem texto nao ha resultado: a tela mostra as recentes');
// ordem do resultado: as que comecam com o texto, depois as mais curtas, depois o alfabeto
assert.deepStrictEqual(LT.buscar(TAGS, 'merc'), ['mercado', 'mercado-mensal', 'casa-mercado', 'supermercado']);
assert.strictEqual(LT.MAX_BUSCA, 40);

// 2. recentes: as ultimas criadas, a mais nova primeiro, sem mexer na lista
assert.deepStrictEqual(LT.recentes(TAGS, 3), ['casa-mercado', 'Água', 'viagem']);
assert.deepStrictEqual(LT.recentes(['a'], 8), ['a']);
assert.strictEqual(TAGS[0], 'mercado', 'a lista original nao muda');

// 3. ordem de "Ver todas": A-Z, Z-A e criadas recentemente; nao ha "Marcadas primeiro" (Listas nao marca)
assert.deepStrictEqual(LT.ordenar(['b', 'c', 'a'], 'az'), ['a', 'b', 'c']);
assert.deepStrictEqual(LT.ordenar(['b', 'c', 'a'], 'za'), ['c', 'b', 'a']);
assert.deepStrictEqual(LT.ordenar(['b', 'c', 'a'], 'rec'), ['a', 'c', 'b']);
assert.deepStrictEqual(LT.ORDENS.map((o) => o[0]), ['az', 'za', 'rec']);

// 4. destaque do trecho achado, com o texto escapado
assert.strictEqual(LT.destacar('Farmácia', 'farm'), '<mark>Farm</mark>ácia');
assert.strictEqual(LT.destacar('a<b', ''), 'a&lt;b');
assert.strictEqual(LT.destacar('casa', 'x'), 'casa');

// 5. a tela usa o componente: o cartao tem o lugar dele, o script vem antes do app.js e a lista antiga saiu
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
assert.ok(html.includes('id="tagsListas"') && !html.includes('id="pillTags"'), 'o cartao Tags tem #tagsListas e nao tem mais #pillTags');
assert.ok(html.indexOf('src="lista-tags.js"') > 0 && html.indexOf('src="lista-tags.js"') < html.indexOf('src="js/app.js"'), 'lista-tags.js carrega antes do app.js');
assert.ok(/ListaTags\.criar\(el, \{tags:\(\)=>tagsUsadas, aoExcluir:excluirTag/.test(app), 'renderGerenciarTags cria o componente com o excluir da tela');

// 6. excluir: a janela abre antes da consulta ao servidor (o toque nao fica sem resposta) e a tag sem uso tambem pergunta
const ini = app.indexOf('async function excluirTag(');
const corpo = app.slice(ini, app.indexOf('\n}', ini) + 2);
assert.ok(ini > 0, 'excluirTag existe');
const abre = corpo.indexOf('janelaExcluirTag('), consulta = corpo.indexOf('await db.collection("lancamentos")');
assert.ok(abre > 0 && consulta > 0 && abre < consulta, 'a janela "Conferindo…" abre antes de perguntar ao servidor');
assert.ok(corpo.includes('Conferindo se algum lançamento usa essa tag'), 'a janela diz o que esta acontecendo');
assert.ok(corpo.includes('Nenhum lançamento usa essa tag.') && /if\(!ok\)\{ fecharJanelaTag\(\); return; \}/.test(corpo), 'tag sem uso pede confirmacao e Cancelar nao exclui');
assert.ok(corpo.indexOf('if(!ok)') < corpo.indexOf('db.doc("config/tags").set'), 'nada e gravado antes da confirmacao');
assert.ok(/Atualizando \$\{feitos\} de \$\{qtd\} lançamentos/.test(corpo), 'a migracao mostra o andamento');

console.log('lista-tags.test.js ok');
