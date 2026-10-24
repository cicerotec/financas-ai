// Compara o saldo que o NAVEGADOR calcula (a tela antiga) com o que o SERVIDOR calcula (GET /saldos) e mede a latencia.
// Item 40, fase B: so depois de dar IGUAL no dev e na producao a tela nova de Saldos entra (.ai/specs/saldo-servidor/plan.md).
//
// Como usar (passo a passo completo em .ai/ambiente-de-teste.md, secao "Comparar e medir o saldo do servidor"):
//   1. Abra o app (dev ou producao), entre como DONO e deixe a aba Saldos carregar.
//   2. F12 > Console, cole este arquivo inteiro e aperte Enter.
// O token nunca sai do navegador: o script le a sessao que o proprio app ja guardou. Ele so le (GET); a unica gravacao e o cache de
// fechamento que o servidor guarda por conta propria na primeira chamada.
//
// Modo SO SERVIDOR (para o teste de volume, com muitos anos de dados): antes de colar o script, digite
//   window.COMPARAR_SO_SERVIDOR = true
// e ele NAO roda a tela antiga (que le o historico inteiro uma vez por banco e estoura a leitura da tabela), so mede o GET /saldos.
//
// Em Node (sem navegador) so a funcao compararSaldos existe, para o teste scripts/comparar-saldos-console.test.js.

var TOLERANCIA = 0.005; // meio centavo: o saldo e arredondado a 2 casas a cada mes (var: da para colar o script de novo no mesmo console)

// antigo: { banco: { saldo, mapa: { 'AAAA-MM': valor } } } (o saldosAtuais do app); totalAntigo: o total da tela antiga;
// resposta: o corpo do GET /saldos ({ total: { valor }, meios: [{ meio, saldo, meses: [{ mes, valor }] }] }).
function compararSaldos(antigo, totalAntigo, resposta) {
  const iguais = (a, b) => Math.abs(a - b) < TOLERANCIA;
  const novos = {};
  (resposta.meios || []).forEach((m) => { novos[m.meio] = m; });
  const linhas = [], soAntigo = [], soNovo = [];
  Object.keys(antigo).forEach((banco) => {
    const a = antigo[banco], n = novos[banco];
    if (!n) { soAntigo.push(banco); return; }
    const mesesDiferentes = [];
    (n.meses || []).forEach((mn) => {
      const va = a.mapa ? a.mapa[mn.mes] : undefined;
      if (va === undefined || va === null) return; // o servidor tem mais meses que a tela antiga (ela so calcula a partir do ponto de partida)
      if (!iguais(va, mn.valor)) mesesDiferentes.push({ mes: mn.mes, antigo: va, novo: mn.valor });
    });
    const dif = Math.round((n.saldo - a.saldo) * 100) / 100;
    linhas.push({ meio: banco, saldoAntigo: a.saldo, saldoNovo: n.saldo, diferenca: dif, mesesDiferentes, ok: iguais(a.saldo, n.saldo) && mesesDiferentes.length === 0 });
  });
  Object.keys(novos).forEach((banco) => { if (!(banco in antigo)) soNovo.push(banco); });
  const difTotal = Math.round((resposta.total.valor - totalAntigo) * 100) / 100;
  const total = { antigo: totalAntigo, novo: resposta.total.valor, diferenca: difTotal, ok: iguais(totalAntigo, resposta.total.valor) };
  return { igual: linhas.every((l) => l.ok) && total.ok && soAntigo.length === 0 && soNovo.length === 0, linhas, soAntigo, soNovo, total };
}

async function rodarComparacao() {
  const cfg = window.FINANCAS_CONFIG;
  const auth = JSON.parse(localStorage.getItem('financas.auth') || 'null');
  if (!auth || Date.now() > auth.exp) { console.error('A sessao expirou: recarregue a pagina (F5) e cole o script de novo.'); return; }
  const sessao = window.claude && window.claude.sessao && window.claude.sessao();
  if (!sessao || !sessao.espaco) { console.error('Nao achei o espaco da sessao: espere o app carregar e tente de novo.'); return; }
  if (sessao.espaco.role !== 'owner') { console.error('Entre como DONO: o membro nao ve a aba Saldos completa.'); return; }
  const url = cfg.apiUrl.replace(/\/$/, '') + '/spaces/' + sessao.espaco.id + '/saldos';
  const soServidor = window.COMPARAR_SO_SERVIDOR === true;

  let antigo = {}, totalAntigo = 0, msAntigo = 0;
  if (soServidor) {
    console.log('Modo so servidor: a tela antiga nao sera calculada.');
  } else {
    console.log('1/3 Calculando no navegador (tela antiga)...');
    const t0 = performance.now();
    await carregarSaldos(true);
    msAntigo = Math.round(performance.now() - t0);
    const comErro = listas.banco.filter((b) => saldosAtuais[b] && saldosAtuais[b].erro);
    if (comErro.length) {
      console.error('A tela antiga falhou em ' + comErro.join(', ') + ' (a leitura do historico deu erro: veja a aba Rede). Nao da para comparar assim; use o modo so servidor (window.COMPARAR_SO_SERVIDOR = true).');
      return;
    }
    listas.banco.forEach((b) => { if (saldosAtuais[b]) antigo[b] = { saldo: saldosAtuais[b].saldo, mapa: saldosAtuais[b].mapa }; });
    totalAntigo = Math.round(totalSaldos().t * 100) / 100;
  }

  console.log((soServidor ? '' : '2/3 ') + 'Chamando GET /saldos tres vezes (a primeira pode ser a sem cache)...');
  const tempos = [];
  let resposta = null;
  for (let i = 1; i <= 3; i++) {
    const t = performance.now();
    const r = await fetch(url, { headers: { authorization: 'Bearer ' + auth.id } });
    const corpo = await r.json().catch(() => ({}));
    tempos.push({ chamada: i, status: r.status, ms: Math.round(performance.now() - t) });
    if (!r.ok) { console.table(tempos); console.error('O servidor respondeu ' + r.status + ': ' + (corpo.erro || '')); return; }
    resposta = corpo;
  }

  if (soServidor) {
    console.table(resposta.meios.map((m) => ({ meio: m.meio, saldo: m.saldo, 'meses devolvidos': m.meses.length, 'sem inicio': m.semInicio })));
    console.log('Total do servidor: ' + resposta.total.valor + ' (' + resposta.total.meios + ' meios, ' + resposta.total.fora + ' fora do total).');
    console.log('Tempos (navegador ate a resposta; a Lambda e o DynamoDB ficam no CloudWatch):');
    console.table(tempos);
    return resposta;
  }

  console.log('3/3 Comparando...');
  const res = compararSaldos(antigo, totalAntigo, resposta);
  console.table(res.linhas.map((l) => ({ meio: l.meio, 'saldo (tela antiga)': l.saldoAntigo, 'saldo (servidor)': l.saldoNovo, diferenca: l.diferenca, 'meses diferentes': l.mesesDiferentes.length, ok: l.ok ? 'OK' : 'DIFERE' })));
  console.log('Total: tela antiga ' + res.total.antigo + ' | servidor ' + res.total.novo + ' | diferenca ' + res.total.diferenca + (res.total.ok ? ' OK' : ' DIFERE'));
  res.linhas.filter((l) => l.mesesDiferentes.length).forEach((l) => { console.log('Meses que diferem em ' + l.meio + ':'); console.table(l.mesesDiferentes); });
  if (res.soAntigo.length) console.warn('So na tela antiga: ' + res.soAntigo.join(', '));
  if (res.soNovo.length) console.warn('So no servidor: ' + res.soNovo.join(', '));
  console.log('Tempos (navegador ate a resposta; a Lambda e o DynamoDB ficam no CloudWatch):');
  console.table(tempos);
  console.log('Tela antiga, calculando tudo no navegador: ' + msAntigo + ' ms (lancamentos no cache do app: ' + (typeof lancamentosCache !== 'undefined' ? lancamentosCache.length : '?') + ').');
  console.log(res.igual ? '%c✓ IGUAL: os saldos e os meses batem.' : '%c✗ DIFERENTE: nao ligue a tela nova; copie as tabelas acima e investigue.', res.igual ? 'color:#0a7d33;font-weight:bold' : 'color:#b3261e;font-weight:bold');
  return res;
}

if (typeof module !== 'undefined' && module.exports) module.exports = { compararSaldos };
else rodarComparacao();
