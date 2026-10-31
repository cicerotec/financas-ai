// Teste das janelas das consultas salvas da Tendencias (item 53, C-T25): janela nova entra em camadasAbertas() e fecharCamadaTopo() de js/app.js,
// senao o botao voltar do celular fecha o app em vez da janela (padrao em .ai/padrao-de-telas.md, "Botao voltar").
// Confere o codigo (sem navegador): cada janela, o botao que a fecha e a ordem de cima para baixo. Rode com: node web/consultas-camadas.test.js
const fs = require('fs');
const path = require('path');
const assert = require('assert');

const app = fs.readFileSync(path.join(__dirname, 'js', 'app.js'), 'utf8');
const tend = fs.readFileSync(path.join(__dirname, 'js', 'tendencias.js'), 'utf8');
const corpo = (nome) => {
  const i = app.indexOf('function ' + nome + '(');
  assert.ok(i >= 0, nome + ' nao existe em app.js');
  const f = app.indexOf('\n}', i);
  return app.slice(i, f + 2);
};
const abertas = corpo('camadasAbertas'), fechar = corpo('fecharCamadaTopo');

// janela -> botao que a fecha, de cima para baixo (a de renomear e a de salvar ficam por cima da lista, que fica por cima da lista de lancamentos)
const janelas = [['tdRnVeu', 'tdRnFechar'], ['tdSvVeu', 'tdSvFechar'], ['tdCsVeu', 'tdCsFechar'], ['tdListaVeu', 'tdLFechar']];
let falhas = 0;
const teste = (nome, fn) => { try { fn(); console.log('ok   ' + nome); } catch (e) { falhas++; console.error('FALHOU: ' + nome + '\n  ' + e.message); } };

teste('cada janela das consultas existe no HTML da Tendencias, com o botao que a fecha', () => {
  for (const [veu, botao] of janelas) {
    assert.ok(tend.includes('id="' + veu + '"'), veu + ' nao esta em tendencias.js');
    assert.ok(tend.includes('id="' + botao + '"'), botao + ' nao esta em tendencias.js');
  }
});

teste('camadasAbertas() conta cada janela (o voltar so fecha o que a contagem enxerga)', () => {
  for (const [veu] of janelas) assert.ok(abertas.includes("camadaVisivel('" + veu + "')"), veu + ' falta em camadasAbertas()');
});

teste('fecharCamadaTopo() fecha cada janela pelo botao dela, na ordem de cima para baixo', () => {
  let ult = -1;
  for (const [veu, botao] of janelas) {
    const i = fechar.indexOf("camadaVisivel('" + veu + "')");
    assert.ok(i >= 0, veu + ' falta em fecharCamadaTopo()');
    assert.ok(fechar.indexOf("clicar('#" + botao + "')", i) > i, 'o botao #' + botao + ' nao e clicado para ' + veu);
    assert.ok(i > ult, veu + ' esta fora de ordem (a de cima fecha primeiro)');
    ult = i;
  }
});

teste('o botao de cada janela fecha a janela por um clique (o voltar clica nele)', () => {
  for (const botao of ['tdRnFechar', 'tdSvFechar', 'tdCsFechar']) {
    assert.ok(tend.includes("t.id === '" + botao + "'"), 'aoClicar nao trata #' + botao);
  }
});

if (falhas) process.exit(1);
console.log('consultas-camadas.test.js: ok');
