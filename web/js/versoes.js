// Tela "Histórico de versões" (Configurar). Demo aprovada: .ai/demos/demo-historico-versoes.html. Spec: .ai/specs/historico/spec.md.
// O front só desenha: a busca por mês, o diff e se uma versão pode ser restaurada (com o motivo) vêm prontos da API.
// Carregado depois do app.js e usa os globais dele (db, esc, fmtMoeda, toast, confirmarModal, comBotaoOcupado, mesAtualApp, somaMes, rotuloMes).

let vsMes = '', vsBusca = '', vsLista = [], vsTrunc = false, vsCarregando = false, vsErro = '';
let vsPer = '30d', vsTudo = false, vsAsc = false; // periodo ('30d' | '90d' | '180d' | 'mes'), busca em todo o historico e ordem
let vsAberto = '', vsDados = null, vsErroVersoes = '', vsReqBusca = 0, vsReqVersoes = 0; // contadores separados: uma resposta velha nao pode apagar a outra

// ---------- formatação (só texto) ----------
const VS_CAMPOS = {descricao:'Descrição', valor:'Valor', tipo:'Tipo', status:'Status', banco:'Banco', dataEvento:'Data', nota:'Nota',
  tags:'Tags', oculto:'Oculto', excluirDoTotal:'Fora do total', faturaAjuste:'Fatura', excluidoEm:'Excluído em', aviso:'Aviso', serie:'Série'};
// insert e a versao 1: o estado de antes da primeira alteracao (ou, nas gravadas antes de 2026-10-23, a criacao/importacao).
const VS_ACAO = {insert:'Original', update:'Alterado', delete:'Excluído', restore:'Restaurado'};
function vsQuando(iso){
  if(!iso) return '';
  const d = new Date(iso);
  if(isNaN(d)) return String(iso);
  return d.toLocaleString('pt-BR', {timeZone:'America/Sao_Paulo', day:'2-digit', month:'2-digit', year:'numeric', hour:'2-digit', minute:'2-digit'}).replace(', ', ' ');
}
function vsData(iso){ return vsQuando(iso).slice(0, 10); }
function vsValor(campo, v){
  if(v === null || v === undefined) return '—';
  if(campo === 'valor') return fmtMoeda(v);
  if(campo === 'dataEvento') return vsData(v);
  if(campo === 'excluidoEm') return vsQuando(v);
  if(typeof v === 'boolean') return v ? 'sim' : 'não';
  if(Array.isArray(v)) return v.length ? v.join(', ') : '—';
  if(typeof v === 'object') return JSON.stringify(v);
  return String(v);
}
function vsDiffHtml(diff){
  return (diff || []).map(d=>`<li><span class="n">${esc(VS_CAMPOS[d.campo] || d.campo)}:</span> <s>${esc(vsValor(d.campo, d.antes))}</s> → ${esc(vsValor(d.campo, d.depois))}</li>`).join('');
}
function vsPilula(acao){
  const cl = acao === 'delete' ? ' ex' : acao === 'restore' ? ' rs' : '';
  return `<span class="vs-pill${cl}">${esc(VS_ACAO[acao] || acao)}</span>`;
}
function vsQuem(v){ return v.porNome || (v.porSub ? 'alguém' : 'autor desconhecido'); }

// ---------- peças ----------
// Período (demo: .ai/demos/demo-versoes-periodo.html): 30d, 90d, 180d ou um mês, sempre pela data da ALTERAÇÃO.
// A busca vale no que está marcado; "Buscar em tudo" amplia sem aviso de custo, porque a leitura do servidor é a mesma em qualquer período.
const VS_DIAS = {'30d':30, '90d':90, '180d':180};
const VS_CAL = '<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/></svg>';
function vsEmTudo(){ return !!(vsBusca && vsTudo); }
function vsEmMes(){ return vsPer === 'mes' && !vsEmTudo(); }
// o que vai para a API
function vsPeriodoApi(){
  if(vsEmTudo()) return {tudo:true};
  if(vsPer === 'mes') return {mes:vsMes};
  return {dias:VS_DIAS[vsPer]};
}
// "em setembro de 2026" / "nos últimos 90 dias" / "em todo o histórico"
function vsEmTxt(){
  if(vsEmTudo()) return 'em todo o histórico';
  if(vsPer === 'mes') return 'em ' + rotuloMes(vsMes).toLowerCase();
  return 'nos últimos ' + VS_DIAS[vsPer] + ' dias';
}
function vsMesAbrev(ym){ const [y, m] = ym.split('-'); return rotuloMes(ym).slice(0, 3).toLowerCase() + '/' + y; }
function vsBarraHtml(){
  const max = mesAtualApp(), emMes = vsEmMes(), tudo = vsEmTudo();
  const seg = '<div class="seg2" role="group" aria-label="Período">' +
    ['30d', '90d', '180d'].map(p=>{ const on = !tudo && vsPer === p; return `<button type="button" data-vs="per" data-p="${p}" aria-pressed="${on}" class="${on ? 'on' : ''}">${p}</button>`; }).join('') +
    `<label class="cal${emMes ? ' on' : ''}" title="Escolher o mês da alteração">${VS_CAL}${emMes ? '<span>' + esc(vsMesAbrev(vsMes)) + '</span>' : ''}<input type="month" id="vsCalIn" value="${esc(vsMes)}" max="${esc(max)}" aria-label="Escolher o mês da alteração"></label></div>`;
  const ord = `<button type="button" class="hm-ord" data-vs="ord" aria-label="Ordem: ${vsAsc ? 'mais antigas' : 'mais recentes'} primeiro. Toque para inverter.">Data ${vsAsc ? '↑' : '↓'}</button>`;
  const linhaMes = emMes ? `<div class="vs-mes"><button type="button" class="vs-nv" data-vs="ant" aria-label="Mês anterior">‹</button>
    <input type="month" id="vsMesIn" value="${esc(vsMes)}" max="${esc(max)}" aria-label="Mês da alteração">
    <button type="button" class="vs-nv" data-vs="prox" aria-label="Próximo mês"${vsMes >= max ? ' disabled' : ''}>›</button></div>` : '';
  const ph = vsLargo() ? 'Buscar neste ' + (vsPer === 'mes' ? 'mês' : 'período') + ' (descrição, nota ou tag)' : 'Descrição, nota ou tag';
  return `<div class="vs-per perbar">${seg}${ord}</div>${linhaMes}
    <div class="vs-busca"><input type="search" id="vsQIn" placeholder="${esc(ph)}" value="${esc(vsBusca)}" aria-label="Buscar"><button type="button" class="vs-bt" data-vs="buscar">Buscar</button></div>`;
}
// Faixa do escopo da busca: onde se buscou, com o botão para ampliar ou voltar.
function vsEscopoHtml(){
  if(!vsBusca) return '';
  if(vsTudo) return `<div class="escopo"><span class="t">Resultado de <b>todo o histórico</b></span><button type="button" data-vs="escopo-voltar">Voltar a ${vsPer === 'mes' ? esc(vsMesAbrev(vsMes)) : vsPer}</button></div>`;
  const onde = vsPer === 'mes' ? 'de <b>' + esc(rotuloMes(vsMes).toLowerCase()) + '</b>' : 'dos <b>últimos ' + VS_DIAS[vsPer] + ' dias</b>';
  return `<div class="escopo"><span class="t">Resultado ${onde}</span><button type="button" data-vs="tudo">Buscar em tudo</button></div>`;
}
function vsResumoHtml(){
  const n = vsLista.length;
  if(vsBusca) return `<div class="vs-cont">${n} lançamento(s) no resultado da busca</div>`;
  return `<div class="vs-cont">${n === 1 ? '1 lançamento mudou' : n + ' lançamentos mudaram'} ${esc(vsEmTxt())}</div>`;
}
// Celular: um título por dia da alteração (por mês, na busca). Computador: uma faixa por mês quando o período passa de um mês.
function vsDiaChave(x){ return vsData(x.em); }
function vsMesChave(x){ return vsData(x.em).slice(3); } // mm/aaaa
function vsTituloMes(chave, n){
  const [m, a] = chave.split('/');
  return rotuloMes(a + '-' + m).toLowerCase() + ' · ' + n + (n === 1 ? ' alteração' : ' alterações');
}
function vsLinhaHtml(x){
  const sinal = x.tipo === 'entrada' ? '+' : '-';
  return `<button type="button" class="vs-li${vsAberto === x.id ? ' sel' : ''}" data-vs="abrir" data-id="${esc(x.id)}">
    <span class="t"><b>${esc(x.descricao || '(sem descrição)')}</b>
    <span class="s">${esc(vsData(x.dataEvento))} · ${esc(x.banco || '')}</span>
    <span class="s">${vsPilula(x.acao)}${esc(vsQuem(x))} · ${esc(vsQuando(x.em))}</span></span>
    <span class="r"><span class="v ${x.tipo === 'entrada' ? 'in' : 'out'}">${sinal}${esc(fmtMoeda(x.valor))}</span><span class="vs">versão ${esc(x.versao)}</span></span></button>`;
}
function vsLinhaTabela(x){
  const sinal = x.tipo === 'entrada' ? '+' : '-';
  return `<tr class="${vsAberto === x.id ? 'sel' : ''}" tabindex="0" data-vs="abrir" data-id="${esc(x.id)}">
    <td>${esc(vsQuando(x.em))}<div class="mu">versão ${esc(x.versao)}</div></td>
    <td><div class="d">${esc(x.descricao || '(sem descrição)')}</div><div class="mu">${esc(vsData(x.dataEvento))} · ${esc(x.banco || '')}</div></td>
    <td>${vsPilula(x.acao)}</td><td>${esc(vsQuem(x))}</td>
    <td class="num ${x.tipo === 'entrada' ? 'in' : 'out'}">${sinal}${esc(fmtMoeda(x.valor))}</td></tr>`;
}
function vsVersaoHtml(v){
  let corpo;
  if(v.acao === 'insert' && !v.porSub) corpo = '<li>Estado antes da primeira alteração registrada.</li>';
  else if(v.acao === 'insert') corpo = `<li>Criado: ${esc((v.registro && v.registro.descricao) || '')} · ${esc(fmtMoeda(v.registro && v.registro.valor))}</li>`;
  else if(v.acao === 'delete') corpo = `<li>Excluído em ${esc(vsQuando(v.registro && v.registro.excluidoEm))}</li>`;
  else corpo = vsDiffHtml(v.diff) || '<li>Sem mudança nos campos.</li>';
  const q = 'Versão ' + v.versao + (v.acao === 'restore' && v.deVersao ? ' · restaurada da versão ' + v.deVersao : '');
  const rodape = v.restauravel
    ? `<button type="button" class="vs-rb" data-vs="restaurar" data-n="${esc(v.versao)}">Restaurar esta versão</button>${v.exigeConfirmacao ? '<div class="vs-mt">Mês fechado: vai pedir confirmação.</div>' : ''}`
    : `<button type="button" class="vs-rb" disabled>Restaurar esta versão</button><div class="vs-mt">${esc(vsMotivo(v.motivo))}</div>`;
  return `<div class="vs-ver"><div class="h"><b>${vsPilula(v.acao)}${esc(vsQuem(v))}</b><span class="vs-mt">${esc(vsQuando(v.em))}</span></div>
    <div class="q">${esc(q)}</div><ul>${corpo}</ul>${rodape}</div>`;
}
// O servidor escreve os motivos sem acento; so poe maiuscula na frente.
function vsMotivo(t){ t = String(t || ''); return t.charAt(0).toUpperCase() + t.slice(1) + (/[.!?]$/.test(t) ? '' : '.'); }
function vsVersoesHtml(){
  if(vsErroVersoes) return `<div class="vs-aviso">${esc(vsErroVersoes)}</div>`;
  if(!vsDados) return '<div class="vs-vazio">Carregando…</div>';
  if(vsDados.semHistorico) return '<div class="vs-vazio">Este lançamento ainda não tem versões.</div>';
  const it = vsLista.find(x=>x.id === vsAberto) || {};
  const ultima = (vsDados.versoes || [])[0] || {};
  const reg = ultima.registro || {};
  const cab = `<div class="vs-ctx"><b>${esc(reg.descricao || it.descricao || '')}</b> · ${esc(fmtMoeda(reg.valor))} · ${esc(vsData(reg.dataEvento))} · ${esc(reg.banco || '')}${vsDados.excluido ? ' <span class="vs-pill ex">Excluído</span>' : ''}</div>`;
  return cab + '<div class="vs-corpo">' + (vsDados.versoes || []).map(vsVersaoHtml).join('') + '</div>';
}
function vsLargo(){ return !!(window.matchMedia && window.matchMedia('(min-width:1160px)').matches); }

function vsTabelaHtml(){
  const qtd = {};
  vsLista.forEach(x=>{ const k = vsMesChave(x); qtd[k] = (qtd[k] || 0) + 1; });
  let corpo = '', anterior = null;
  vsLista.forEach(x=>{
    const k = vsMesChave(x);
    if(!vsEmMes() && k !== anterior){ anterior = k; corpo += `<tr class="vs-mh"><td colspan="5">${esc(vsTituloMes(k, qtd[k]))}</td></tr>`; }
    corpo += vsLinhaTabela(x);
  });
  return `<table class="vs-tab"><thead><tr><th>Alterado em</th><th>Lançamento</th><th>O que houve</th><th>Quem</th><th class="num">Valor</th></tr></thead><tbody>${corpo}</tbody></table>`;
}
function vsTituloDia(x){
  const w = new Date(x.em).toLocaleDateString('pt-BR', {timeZone:'America/Sao_Paulo', weekday:'short'}).replace('.', '');
  return w.charAt(0).toUpperCase() + w.slice(1) + ' ' + vsData(x.em);
}
function vsCelularHtml(){
  const qtd = {};
  vsLista.forEach(x=>{ const k = vsMesChave(x); qtd[k] = (qtd[k] || 0) + 1; });
  let h = '', chave = null, aberto = false;
  vsLista.forEach(x=>{
    const k = vsBusca ? vsMesChave(x) : vsDiaChave(x);
    if(k !== chave){
      if(aberto) h += '</div>';
      chave = k; aberto = true;
      h += `<div class="fut-titulo"><span>${esc(vsBusca ? vsTituloMes(k, qtd[k]) : vsTituloDia(x))}</span></div><div class="card vs-card">`;
    }
    h += vsLinhaHtml(x);
  });
  if(aberto) h += '</div>';
  const prox = vsPer === '30d' ? '90d' : vsPer === '90d' ? '180d' : '';
  return h + '<div class="vs-fim">' + (!vsBusca && prox ? `<button type="button" data-vs="per" data-p="${prox}">Ver ${prox}</button>` : '') +
    '<button type="button" data-vs="topo">↑ Topo</button></div>';
}

// ---------- tela ----------
function vsRender(){
  const sec = document.getElementById('tab-versoes');
  if(!sec) return;
  let lista;
  if(vsErro) lista = `<div class="vs-aviso">${esc(vsErro)}</div>`;
  else if(vsCarregando) lista = '<div class="vs-vazio">Buscando…</div>';
  else if(!vsLista.length) lista = `<div class="card"><div class="vs-vazio">Nenhuma alteração ${esc(vsEmTxt())}${vsBusca ? ' para essa busca' : ''}.</div></div>`;
  else {
    const aviso = vsTrunc ? '<div class="vs-aviso" role="status">Há mais versões do que a tela mostra. Refine a busca ou escolha outro mês.</div>' : '';
    lista = vsResumoHtml() + aviso + (vsLargo() ? vsTabelaHtml() : vsCelularHtml());
  }
  if(!vsErro && !vsCarregando) lista = vsEscopoHtml() + lista; // a faixa do escopo aparece também quando a busca não acha nada
  const painel = vsLargo()
    ? `<div class="vs-painel"><div class="vs-ph">Versões</div>${vsAberto ? vsVersoesHtml() : '<div class="vs-vazio">Escolha um lançamento para ver as versões.</div>'}</div>` : '';
  sec.innerHTML = vsBarraHtml() + (vsLargo() ? `<div class="vs-desk"><div>${lista}</div>${painel}</div>` : lista);
  vsTelaCheia();
}
// No celular, as versões abrem em tela cheia por cima do app (padrão 2); no computador ficam no painel ao lado.
function vsTelaCheia(){
  let tc = document.getElementById('vsTc');
  if(!vsAberto || vsLargo()){ if(tc) tc.remove(); document.body.style.overflow = ''; return; }
  if(!tc){
    tc = document.createElement('div');
    tc.id = 'vsTc'; tc.className = 'tc';
    tc.setAttribute('role', 'dialog'); tc.setAttribute('aria-label', 'Versões do lançamento');
    document.body.appendChild(tc);
    document.body.style.overflow = 'hidden';
  }
  const rolagem = tc.querySelector('.tc-scr') ? tc.querySelector('.tc-scr').scrollTop : 0;
  tc.innerHTML = `<div class="tc-scr"><div class="tc-col"><div class="tc-hd"><button type="button" data-vs="voltar" aria-label="Voltar">←</button><h2>Versões</h2></div>${vsVersoesHtml()}</div></div>`;
  if(rolagem) tc.querySelector('.tc-scr').scrollTop = rolagem;
}
async function vsBuscar(){
  const req = ++vsReqBusca;
  vsCarregando = true; vsErro = '';
  vsRender();
  try{
    if(!db || typeof db.historico !== 'function') throw new Error('O histórico de versões só funciona no app com servidor.');
    const r = await db.historico(vsPeriodoApi(), vsBusca);
    if(req !== vsReqBusca) return;
    vsLista = r.itens || []; if(vsAsc) vsLista = vsLista.slice().reverse(); // o servidor devolve a mais nova primeiro
    vsTrunc = !!r.truncado;
  }catch(err){
    if(req !== vsReqBusca) return;
    console.error(err); vsLista = []; vsTrunc = false;
    vsErro = (err && err.message) || 'Não foi possível buscar o histórico. Confira a conexão e tente de novo.';
  }
  vsCarregando = false;
  vsRender();
}
async function vsCarregarVersoes(){
  const id = vsAberto, req = ++vsReqVersoes;
  vsDados = null; vsErroVersoes = '';
  vsRender();
  try{
    const r = await db.versoesDoLancamento(id);
    if(id !== vsAberto || req !== vsReqVersoes) return;
    vsDados = r;
  }catch(err){
    if(id !== vsAberto) return;
    console.error(err);
    vsErroVersoes = (err && err.message) || 'Não foi possível ler as versões. Tente de novo.';
  }
  vsRender();
}
function vsFechar(){ vsAberto = ''; vsDados = null; vsErroVersoes = ''; vsRender(); }
async function vsRestaurar(n, btn){
  const v = ((vsDados && vsDados.versoes) || []).find(x=>x.versao === n);
  if(!v) return;
  const id = vsAberto; // guardado antes da janela: se a tela cheia fechar enquanto a pessoa decide, o id nao se perde
  const reg = v.registro || {};
  const exige = !!v.exigeConfirmacao; // o servidor avisa; a confirmacao e esta mesma janela
  const ok = await confirmarModal(`Restaurar a versão ${n} de "${reg.descricao || ''}"? O lançamento volta para o conteúdo dessa versão (${fmtMoeda(reg.valor)} · ${vsData(reg.dataEvento)})${vsDados.excluido ? ' e reaparece nas listas' : ''}.${exige ? ' O mês desse lançamento já fechou: restaurar muda o saldo dele.' : ''} Será gravada uma versão nova; as anteriores continuam.`, exige ? 'Restaurar mesmo assim' : undefined);
  if(!ok) return;
  await comBotaoOcupado(btn, 'Restaurando…', async ()=>{
    try{
      const r = await db.restaurarVersao(id, n, exige);
      if(r.status !== 200){ toast((r.body && r.body.erro) || 'Não foi possível restaurar.', true); return; }
      toast(`Restaurado da versão ${n}. Uma versão nova foi gravada.`);
      await vsCarregarVersoes();
      vsBuscar();
    }catch(err){ console.error(err); toast((err && err.message) || 'Não foi possível restaurar. Confira a conexão e tente de novo.', true); }
  });
}
function vsMudarMes(ym){
  if(!/^\d{4}-\d{2}$/.test(ym) || ym > mesAtualApp()) return;
  vsMes = ym; vsPer = 'mes'; vsTudo = false; vsAberto = ''; vsDados = null;
  vsBuscar();
}
function abrirVersoes(){
  if(!vsMes) vsMes = mesAtualApp();
  vsBuscar();
}
function vsPreparar(){
  const sec = document.getElementById('tab-versoes');
  if(!sec || sec.dataset.pronto) return;
  sec.dataset.pronto = '1';
  sec.addEventListener('click', e=>{
    // abre o seletor de mês mesmo quando o campo está escondido sob o botão do calendário
    if(e.target.matches && e.target.matches('input[type=month]')){ try{ e.target.showPicker(); }catch(_){} return; }
    const b = e.target.closest('[data-vs]'); if(!b) return;
    const a = b.dataset.vs;
    if(a === 'ant') return vsMudarMes(somaMes(vsMes, -1));
    if(a === 'prox') return vsMudarMes(somaMes(vsMes, 1));
    if(a === 'buscar'){ vsBusca = document.getElementById('vsQIn').value.trim(); if(!vsBusca) vsTudo = false; vsAberto = ''; vsDados = null; return vsBuscar(); }
    if(a === 'per'){ vsPer = b.dataset.p; vsTudo = false; vsAberto = ''; vsDados = null; return vsBuscar(); }
    if(a === 'tudo'){ vsTudo = true; vsAberto = ''; vsDados = null; return vsBuscar(); }
    if(a === 'escopo-voltar'){ vsTudo = false; vsAberto = ''; vsDados = null; return vsBuscar(); }
    if(a === 'ord'){ vsAsc = !vsAsc; vsLista = vsLista.slice().reverse(); return vsRender(); }
    if(a === 'topo'){ if(window.scrollTo) window.scrollTo({top:0, behavior:'smooth'}); return; }
    if(a === 'abrir'){ vsAberto = b.dataset.id; return vsCarregarVersoes(); }
  });
  sec.addEventListener('change', e=>{ if(e.target.id === 'vsMesIn' || e.target.id === 'vsCalIn') vsMudarMes(e.target.value); });

  sec.addEventListener('keydown', e=>{
    if(e.key !== 'Enter') return;
    if(e.target.id === 'vsQIn'){ vsBusca = e.target.value.trim(); if(!vsBusca) vsTudo = false; vsAberto = ''; vsDados = null; vsBuscar(); }
    else if(e.target.matches && e.target.matches('tr[data-vs="abrir"]')){ vsAberto = e.target.dataset.id; vsCarregarVersoes(); }
  });
  // a tela cheia (celular) e o painel (computador) tem os mesmos botoes de versao
  document.addEventListener('click', e=>{
    const b = e.target.closest('#vsTc [data-vs], #tab-versoes [data-vs="restaurar"]'); if(!b) return;
    if(b.dataset.vs === 'voltar') return vsFechar();
    if(b.dataset.vs === 'restaurar') return vsRestaurar(Number(b.dataset.n), b);
  });
  document.addEventListener('keydown', e=>{ if(e.key === 'Escape' && vsAberto && document.getElementById('modalOverlay').classList.contains('hidden')) vsFechar(); });
  if(window.matchMedia) window.matchMedia('(min-width:1160px)').addEventListener('change', ()=>{ if(document.body.dataset.aba === 'versoes') vsRender(); });
}
document.addEventListener('DOMContentLoaded', vsPreparar);
