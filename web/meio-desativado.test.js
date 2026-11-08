// Teste do item 58 (.ai/specs/meios-de-pagamento, Fase C2): desativar um meio e encerrar a conta. O front so exibe e pede: pede ativoAte "hoje" (o servidor diz que
// dia e hoje), mostra a lista de pendencias que o servidor devolve, esconde o que ele marca encerrado e apaga as acoes que ele recusaria.
// Rode com: node web/meio-desativado.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');

const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const api = fs.readFileSync(path.join(__dirname, 'api.js'), 'utf8');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
const css = fs.readFileSync(path.join(__dirname, 'css', 'app.css'), 'utf8');

// 1. Desativar so vale a partir de hoje: some "Fim do mês" e "Outra data", e a data e a do servidor (o app pede "hoje", nao manda a data do aparelho)
assert.ok(!/Fim do mês/.test(app) && !/Outra data/.test(app) && !/meioOutraData/.test(app), 'a janela Desativar so tem "a partir de hoje"');
assert.ok(/definirMeio\(nome, 'hoje'/.test(app), 'o app pede ativoAte "hoje"');
assert.ok(!/definirMeio\(nome, data,/.test(app), 'o app nao calcula a data de desativar');

// 2. o 409 MEIO_COM_PENDENCIA volta como { status, body } e a tela mostra as frases do servidor (nao decide por conta propria)
assert.ok(/if \(r\.status === 200\) atualizarOuvintes\('doc', 'listas'\); return r;/.test(api), 'meioDefinir devolve a resposta inteira (o 409 nao e sucesso)');
assert.ok(/r\.status===409/.test(app) && /body\.pendencias/.test(app), 'a tela le pendencias do 409');
assert.ok(/mostrarPendenciasMeio/.test(app) && /class="pend"/.test(app), 'a lista de pendencias aparece na janela');

// 3. o texto de Listas fala em encerrar a conta
assert.ok(/Desativar encerra a conta/.test(html), 'o texto de Listas diz que desativar encerra a conta');
assert.ok(!/Ele só deixa de ser sugerido ao lançar/.test(html), 'o texto antigo saiu');

// 4. o meio desativado so e mantido no select quando se edita um lancamento dele; o pagador de fatura nunca e um meio desativado
assert.ok(/if\(editandoId && atual && !itens\.includes\(atual\)/.test(app), 'refazer so mantem o meio desativado ao editar');
assert.ok(/const meios = listas\.banco\.filter\(meioAtivo\);/.test(app), 'Pagar fatura so oferece meio ativo');

// 5. Saldos e Cartao escondem o que o servidor marca encerrado, e mostram a marca no desativado que continua
assert.ok(/encerrado:!!m\.encerrado/.test(app) && /saldoEncerrado/.test(app), 'Saldos le o encerrado do servidor');
assert.ok(/filter\(c=>!c\.encerrado\)/.test(app), 'Cartao esconde o encerrado');
assert.ok(/marcaDesativado\(b\)/.test(app) && /marcaDesativado\(c\.meio\)/.test(app), 'a marca aparece em Saldos e Cartao');

// 6. Historico: marca na linha e no filtro; as acoes que o servidor recusaria aparecem apagadas, com o motivo, e somem dos icones da tabela
assert.ok(/marcaDesativado\(l\.banco/.test(app) && /\$\{marcaDesativado\(x\)\}/.test(app), 'a marca aparece na linha e no filtro de meio');
assert.ok(/const off = meioDesativado\(l\.banco\)/.test(app) && /desab:true/.test(app), 'as acoes de um meio desativado ficam apagadas');
assert.ok(/acoesDoEvento\(l\)\.filter\(a=>!a\.desab\)/.test(app), 'na tabela so ficam os icones que valem');
assert.ok(/aria-disabled="true">\$\{a\.t\}<small>\$\{esc\(a\.motivo\)\}/.test(app), 'a janela mostra o motivo de cada acao apagada');

// 7. a marca usa so tokens
assert.ok(/\.desat\{[^}]*var\(--border\)[^}]*var\(--muted\)/.test(css) && /\.pend\{[^}]*var\(--danger-bg\)/.test(css), 'css do item 58 so com tokens');

// 8. Lancar editando um lancamento de meio desativado: faixa com o motivo e so descricao, nota e tags livres (demo-lancar-v7); sair da edicao destrava
assert.ok(/id="avisoDesat"/.test(html), 'o formulario tem a faixa do meio desativado');
assert.ok(/travarFormulario\(meioDesativado\(l\.banco\) \? l\.banco : ''\)/.test(app), 'editar trava quando o meio esta desativado');
assert.ok(/function iniciarCopia\(id\)\{[\s\S]*?travarFormulario\(''\)/.test(app), 'copiar destrava');
assert.ok(/function limparFormulario\(\)\{\s*travarFormulario\(''\)/.test(app), 'limpar destrava');
const corpoTrava = app.match(/function travarFormulario[\s\S]*?\n\}/)[0];
assert.ok(/#tipoLancar button, #dtAtalhos button, #maisAbrir, #fValor, #fData, #fBanco, #fStatus/.test(corpoTrava), 'trava tipo, valor, data, meio, forma e Mais');
assert.ok(!/#fDesc|#fNota|tagsAbrir/.test(corpoTrava), 'deixa descricao, nota e tags livres');
assert.ok(/#formCard\.travado #tipoLancar/.test(css), 'css da trava');

// 9. registrar a diferenca da conferencia: o 409 (meio desativado) e erro, nao sucesso silencioso
assert.ok(/escrita\(await api\('POST', await cartaoUrl\(meio, '\/faturas\/' \+ ym \+ '\/diferenca'\)/.test(api), 'cartaoDiferenca passa o 409 por escrita()');

console.log('ok meio-desativado.test.js');
