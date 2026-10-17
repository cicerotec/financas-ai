// Tela "Histórico de versões" (Configurar). Demo aprovada: .ai/demos/demo-historico-versoes.html. Spec: .ai/specs/historico/spec.md.
// O front só desenha: a busca por mês, o diff e se uma versão pode ser restaurada (com o motivo) vêm prontos da API.
// Carregado depois do app.js e usa os globais dele (db, esc, fmtMoeda, toast, confirmarModal, comBotaoOcupado, mesAtualApp, somaMes, rotuloMes).

let vsMes = '', vsBusca = '', vsLista = [], vsTrunc = false, vsCarregando = false, vsErro = '';
let vsAberto = '', vsDados = null, vsErroVersoes = '', vsReqBusca = 0, vsReqVersoes = 0; // contadores separados: uma resposta velha nao pode apagar a outra

// ---------- formatação (só texto) ----------
const VS_CAMPOS = {descricao:'Descrição', valor:'Valor', tipo:'Tipo', status:'Status', banco:'Banco', dataEvento:'Data', nota:'Nota',
  tags:'Tags', oculto:'Oculto', excluirDoTotal:'Fora do total', faturaAjuste:'Fatura', excluidoEm:'Excluído em', aviso:'Aviso', serie:'Série'};
const VS_ACAO = {insert:'Criado', update:'Alterado', delete:'Excluído', restore:'Restaurado'};
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
function vsBarraHtml(){
  const max = mesAtualApp();
  return `<div class="vs-mes"><button type="button" class="vs-nv" data-vs="ant" aria-label="Mês anterior">‹</button>
    <input type="month" id="vsMesIn" value="${esc(vsMes)}" max="${esc(max)}" aria-label="Mês da alteração">
    <button type="button" class="vs-nv" data-vs="prox" aria-label="Próximo mês"${vsMes >= max ? ' disabled' : ''}>›</button></div>
    <div class="vs-busca"><input type="search" id="vsQIn" placeholder="Buscar na descrição ou nota" value="${esc(vsBusca)}" aria-label="Buscar"><button type="button" class="vs-bt" data-vs="buscar">Buscar</button></div>`;
}
function vsResumoHtml(){
  const n = vsLista.length;
  return `<div class="vs-cont">${n === 1 ? '1 lançamento mudou' : n + ' lançamentos mudaram'} em ${esc(rotuloMes(vsMes).toLowerCase())}</div>`;
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
    ? `<button type="button" class="vs-rb" data-vs="restaurar" data-n="${esc(v.versao)}">Restaurar esta versão</button>`
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

// ---------- tela ----------
function vsRender(){
  const sec = document.getElementById('tab-versoes');
  if(!sec) return;
  let lista;
  if(vsErro) lista = `<div class="vs-aviso">${esc(vsErro)}</div>`;
  else if(vsCarregando) lista = '<div class="vs-vazio">Buscando…</div>';
  else if(!vsLista.length) lista = `<div class="card"><div class="vs-vazio">Nenhuma alteração em ${esc(rotuloMes(vsMes).toLowerCase())}${vsBusca ? ' para essa busca' : ''}.</div></div>`;
  else {
    const aviso = vsTrunc ? '<div class="vs-aviso" role="status">Há mais versões do que a tela mostra. Refine a busca ou escolha outro mês.</div>' : '';
    lista = vsResumoHtml() + aviso + (vsLargo()
      ? `<table class="vs-tab"><thead><tr><th>Alterado em</th><th>Lançamento</th><th>O que houve</th><th>Quem</th><th class="num">Valor</th></tr></thead><tbody>${vsLista.map(vsLinhaTabela).join('')}</tbody></table>`
      : `<div class="card vs-card">${vsLista.map(vsLinhaHtml).join('')}</div>`);
  }
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
    if(!db || typeof db.historicoMes !== 'function') throw new Error('O histórico de versões só funciona no app com servidor.');
    const r = await db.historicoMes(vsMes, vsBusca);
    if(req !== vsReqBusca) return;
    vsLista = r.itens || []; vsTrunc = !!r.truncado;
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
  const reg = v.registro || {};
  const ok = await confirmarModal(`Restaurar a versão ${n} de "${reg.descricao || ''}"? O lançamento volta para o conteúdo dessa versão (${fmtMoeda(reg.valor)} · ${vsData(reg.dataEvento)})${vsDados.excluido ? ' e reaparece nas listas' : ''}. Será gravada uma versão nova; as anteriores continuam.`);
  if(!ok) return;
  const id = vsAberto;
  await comBotaoOcupado(btn, 'Restaurando…', async ()=>{
    try{
      const r = await db.restaurarVersao(id, n);
      if(r.status !== 200){ toast((r.body && r.body.erro) || 'Não foi possível restaurar.', true); return; }
      toast(`Restaurado da versão ${n}. Uma versão nova foi gravada.`);
      await vsCarregarVersoes();
      vsBuscar();
    }catch(err){ console.error(err); toast((err && err.message) || 'Não foi possível restaurar. Confira a conexão e tente de novo.', true); }
  });
}
function vsMudarMes(ym){
  if(!/^\d{4}-\d{2}$/.test(ym) || ym > mesAtualApp()) return;
  vsMes = ym; vsAberto = ''; vsDados = null;
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
    const b = e.target.closest('[data-vs]'); if(!b) return;
    const a = b.dataset.vs;
    if(a === 'ant') return vsMudarMes(somaMes(vsMes, -1));
    if(a === 'prox') return vsMudarMes(somaMes(vsMes, 1));
    if(a === 'buscar'){ vsBusca = document.getElementById('vsQIn').value.trim(); vsAberto = ''; vsDados = null; return vsBuscar(); }
    if(a === 'abrir'){ vsAberto = b.dataset.id; return vsCarregarVersoes(); }
  });
  sec.addEventListener('change', e=>{ if(e.target.id === 'vsMesIn') vsMudarMes(e.target.value); });
  sec.addEventListener('keydown', e=>{
    if(e.key !== 'Enter') return;
    if(e.target.id === 'vsQIn'){ vsBusca = e.target.value.trim(); vsAberto = ''; vsDados = null; vsBuscar(); }
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
