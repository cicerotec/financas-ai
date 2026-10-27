let db = null, sample = null;
let papelAtual = 'owner', meuSub = '';
let lancamentosParaImportar = [];
let listas = { status: [], banco: [] }; // status: a lista que o servidor manda (forma de pagamento); o app so a exibe
let statusCor = {};
let tema = 'auto'; // da aparencia da pessoa: auto (segue o aparelho), claro ou escuro (js/tema.js aplica)
let estiloCor = 'ambos';
let corAberta = null;
const COR_PALETA = ['#639922','#BA7517','#378ADD','#7F77DD','#888780','#D85A30','#1D9E75','#D4537E','#E24B4A'];
let coresPadrao = {}; // cor padrao de cada status, vinda do servidor (GET /cfg/listas); a cor que a pessoa escolheu (statusCor) vale por cima
function corStatus(s){ const c = statusCor[s] || coresPadrao[s] || '#888780'; return /^#[0-9a-fA-F]{3,8}$/.test(c) ? c : '#888780'; }
function estiloItemCor(status){
  const c = corStatus(status);
  const tint = `background:color-mix(in srgb,${c} 16%,transparent);`;
  const base = 'border:1px solid var(--border);padding:10px;margin-bottom:6px;';
  if(estiloCor==='linha') return `${tint}${base}border-radius:8px;`;
  if(estiloCor==='faixa') return `${base}border-left:4px solid ${c};border-radius:0;`;
  if(estiloCor==='ambos') return `${tint}${base}border-left:4px solid ${c};border-radius:0;`;
  return '';
}
function seloStatus(status){
  if(estiloCor==='nenhum') return esc(status);
  const c = corStatus(status);
  return `<span style="background:color-mix(in srgb,${c} 32%,transparent);padding:1px 7px;border-radius:8px;font-weight:600;color:var(--text);">${esc(status)}</span>`;
}
let tagsUsadas = [];
let saldosConfig = {};
let layoutSaldos = 'linhas';
let ordemSaldos = 'lista';
let ordemManual = [];
let timerOrdem = null;
let saldosAtuais = {};
let saldosCarregando = false;
let saldosUltimaCalc = 0;
let saldoAberto = -1;
let rascunhoSaldo = null;
let msgSaldo = '';
// O que e cartao e o ajuste de fatura que cada status de credito traz vem pronto do servidor (GET /cfg/listas: meios e faturaPadrao);
// o navegador so consulta, sem olhar o nome do status.
let meiosLista = [];   // [{ nome, cartao, ativo }]
let faturaPadrao = {}; // status de credito -> ajuste padrao da fatura (-1, 0 ou +1)
function ehMeioCartao(nome){ return meiosLista.some(m=>m.nome===nome && m.cartao); }
function ajustePadrao(status){ return faturaPadrao[status] || 0; }
function ajusteEfetivo(l){ if(typeof l.faturaAjuste==='number') return l.faturaAjuste; return ajustePadrao(l.status); }
function chipFatura(l){
  if(!ehMeioCartao(l.banco)) return '';
  const a = ajusteEfetivo(l);
  if(!a) return '';
  return ` <span style="font-size:11px;border:1px solid var(--border);border-radius:8px;padding:0 6px;">foi para a ${a>0?'próxima':'anterior'} fatura</span>`;
}
function chipPagaFatura(l){
  const m = l.pagaFatura;
  if(!m || !m.meio || !m.ym) return '';
  return ` <span style="font-size:11px;border:1px solid var(--accent);color:var(--accent);border-radius:8px;padding:0 6px;">Fatura ${esc(rotuloMesCurto(m.ym))} · ${esc(m.meio)}</span>`;
}
function chipEstimado(l){
  if(l.valorEstimado!==true) return '';
  return ' <span style="font-size:11px;border:1px solid var(--border);border-radius:8px;padding:0 6px;">valor estimado</span>';
}
// A data escolhida no formulario e depois de hoje? So serve para mostrar o aviso e a repeticao; quem decide se o lancamento e previsto e o servidor.
function dataFuturaForm(){ const v = document.getElementById('fData').value; return !!v && v.slice(0,10) > chaveDia(new Date()); }
function atualizarCampoFatura(){
  const w = document.getElementById('fFaturaWrap');
  if(!w) return;
  const st = document.getElementById('fStatus').value;
  const fd = document.getElementById('fFuturoDica');
  if(fd) fd.textContent = dataFuturaForm() ? 'Data futura: aparece na aba Futuros, não no Histórico, e não mexe no saldo.' : '';
  w.classList.toggle('hidden', !ehMeioCartao(document.getElementById('fBanco').value));
  atualizarCampoAviso();
  const padrao = ajustePadrao(st);
  document.getElementById('fFaturaDica').textContent = padrao>0 ? 'Registros com este status já vão para a próxima fatura.' : (padrao<0 ? 'Registros com este status já entram na fatura anterior (a que fecha neste mês).' : '');
}
// ---------- aviso (Telegram): controles reutilizados pelo formulario (prefixo f) e pela janela de repeticao (pv) ----------
function telegramVinculado(){ return !!(window.claude && claude.telegramChatId && claude.telegramChatId()); }
const avisoVisivel = { f: ()=>dataFuturaForm(), pv: ()=>true };
function htmlAviso(p){
  const dias = [[7,'7 dias antes'],[3,'3 dias antes'],[1,'1 dia antes'],[0,'No dia']];
  return `<div id="${p}AvisoWrap" class="hidden">
    <label style="display:flex;align-items:center;gap:8px;cursor:pointer;margin-top:8px;"><input type="checkbox" id="${p}Avisar" style="width:auto;"> 🔔 Avisar no Telegram</label>
    <div id="${p}AvisoOpcoes" class="hidden" style="margin:6px 0 0 4px;">
      <div id="${p}AvisoDias" style="display:flex;flex-wrap:wrap;gap:6px 14px;">
        ${dias.map(([v,t])=>`<label style="display:flex;align-items:center;gap:4px;cursor:pointer;"><input type="checkbox" value="${v}" style="width:auto;"> ${t}</label>`).join('')}
      </div>
      <label style="display:flex;align-items:center;gap:8px;cursor:pointer;margin-top:6px;"><input type="checkbox" id="${p}AvisoInsistir" style="width:auto;"> Insistir todo dia depois do vencimento, até pagar</label>
    </div>
    <p class="muted" id="${p}AvisoDica"></p>
  </div>`;
}
function atualizarAviso(p){
  const w = document.getElementById(p+'AvisoWrap');
  if(!w) return;
  const mostrar = avisoVisivel[p]();
  const marcado = document.getElementById(p+'Avisar').checked;
  w.classList.toggle('hidden', !mostrar);
  document.getElementById(p+'AvisoOpcoes').classList.toggle('hidden', !marcado);
  document.getElementById(p+'AvisoDica').textContent = (mostrar && marcado && !telegramVinculado())
    ? 'Você ainda não vinculou o Telegram: o aviso é guardado, mas só chega depois que você cadastrar o chat id em Listas.' : '';
}
// null = sem aviso; 'incompleto' = marcado sem nenhum dia nem "insistir"
function lerAviso(p){
  if(document.getElementById(p+'AvisoWrap').classList.contains('hidden') || !document.getElementById(p+'Avisar').checked) return null;
  const dias = [...document.querySelectorAll('#'+p+'AvisoDias input:checked')].map(i=>parseInt(i.value,10));
  const insistir = document.getElementById(p+'AvisoInsistir').checked;
  if(!dias.length && !insistir) return 'incompleto';
  return {dias, insistir};
}
function preencherAviso(p, aviso){
  const tem = !!(aviso && typeof aviso === 'object');
  document.getElementById(p+'Avisar').checked = tem;
  const dias = tem && Array.isArray(aviso.dias) ? aviso.dias : [];
  document.querySelectorAll('#'+p+'AvisoDias input').forEach(i=>{ i.checked = dias.includes(parseInt(i.value,10)); });
  document.getElementById(p+'AvisoInsistir').checked = tem && aviso.insistir === true;
  atualizarAviso(p);
}
// ao ligar "Avisar" sem nada marcado: sugere 1 dia antes, no dia, e insistir depois (o caso da multa)
function ligarAviso(p, aoMudar){
  document.getElementById(p+'Avisar').addEventListener('change', ()=>{
    const algum = document.querySelectorAll('#'+p+'AvisoDias input:checked').length || document.getElementById(p+'AvisoInsistir').checked;
    if(document.getElementById(p+'Avisar').checked && !algum){
      document.querySelectorAll('#'+p+'AvisoDias input').forEach(i=>{ i.checked = (i.value==='1' || i.value==='0'); });
      document.getElementById(p+'AvisoInsistir').checked = true;
    }
    atualizarAviso(p);
    if(aoMudar) aoMudar();
  });
  document.getElementById(p+'AvisoWrap').addEventListener('change', e=>{ if(e.target.id!==p+'Avisar' && aoMudar) aoMudar(); });
}
function atualizarCampoAviso(){ atualizarAviso('f'); atualizarCampoRepetir(); }
function lerAvisoDoFormulario(){ return lerAviso('f'); }
function preencherAvisoNoFormulario(aviso){ preencherAviso('f', aviso); }

// ---------- repeticao de eventos futuros (Prever, Estender e "Repetir" ao lancar) ----------
// Controles de quantidade e periodicidade; a logica de datas fica em recorrencia.js (testada em node).
function htmlRepeticao(p, o){
  const radios = Recorrencia.INTERVALOS.map(i=>`<label style="display:flex;align-items:center;gap:4px;cursor:pointer;"><input type="radio" name="${p}Meses" value="${i.meses}" ${i.meses===1?'checked':''} style="width:auto;"> ${i.rotulo}</label>`).join('');
  return `<div id="${p}Repeticao">
    <div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-top:8px;">
      <label for="${p}Qtd" style="margin:0;">${o.rotuloQtd}</label>
      <input type="number" id="${p}Qtd" min="1" max="${Recorrencia.MAX_OCORRENCIAS}" value="${o.qtd}" inputmode="numeric" style="width:76px;">
      ${[1,3,6,12].map(n=>`<button type="button" class="ghost" data-qtd-rapida="${n}" data-p="${p}" style="padding:2px 10px;">${n}</button>`).join('')}
    </div>
    <label style="margin-top:8px;">A cada</label>
    <div style="display:flex;flex-wrap:wrap;gap:6px 14px;align-items:center;">
      ${radios}
      <label style="display:flex;align-items:center;gap:4px;cursor:pointer;"><input type="radio" name="${p}Meses" value="0" style="width:auto;"> Outro</label>
      <input type="number" id="${p}Outro" min="1" max="${Recorrencia.MAX_INTERVALO_MESES}" placeholder="meses" inputmode="numeric" class="hidden" style="width:90px;">
    </div>
  </div>`;
}
function ligarRepeticao(p, aoMudar){
  const raiz = document.getElementById(p+'Repeticao');
  raiz.addEventListener('change', e=>{
    if(e.target.name === p+'Meses') document.getElementById(p+'Outro').classList.toggle('hidden', e.target.value !== '0');
    if(aoMudar) aoMudar();
  });
  raiz.addEventListener('input', ()=>{ if(aoMudar) aoMudar(); });
  raiz.addEventListener('click', e=>{
    const b = e.target.closest('button[data-qtd-rapida]');
    if(b && b.dataset.p === p){ document.getElementById(p+'Qtd').value = b.dataset.qtdRapida; if(aoMudar) aoMudar(); }
  });
}
function lerRepeticao(p){
  const quantidade = parseInt(document.getElementById(p+'Qtd').value, 10);
  const sel = document.querySelector(`input[name="${p}Meses"]:checked`);
  let meses = sel ? parseInt(sel.value, 10) : 1;
  if(meses === 0) meses = parseInt(document.getElementById(p+'Outro').value, 10);
  return {quantidade, intervaloMeses: meses};
}
function definirIntervalo(p, meses){
  const radio = [...document.querySelectorAll(`input[name="${p}Meses"]`)].find(r=>parseInt(r.value,10)===meses && meses>0);
  if(radio){ radio.checked = true; document.getElementById(p+'Outro').classList.add('hidden'); return; }
  document.querySelector(`input[name="${p}Meses"][value="0"]`).checked = true;
  const o = document.getElementById(p+'Outro'); o.classList.remove('hidden'); o.value = meses;
}
function novoSerieId(){
  return 'ser' + Date.now().toString(36) + Math.random().toString(36).slice(2, 10).padEnd(8, '0');
}
function dataCurta(d){ return d.toLocaleDateString('pt-BR', {weekday:'short', day:'2-digit', month:'2-digit', year:'numeric'}); }
// Prever, Estender e repetir ao lancar comparam com os lancamentos e os previstos: ninguem os le na subida, entao sao lidos aqui, a pedido.
async function lerDadosParaDuplicados(){
  await Promise.all([lancsOuv ? lancsOuv.garantir() : null, garantirFuturos()]);
}
// Aviso de duplicado ao salvar: uma consulta pontual pela data do lancamento (nao precisa dos 1000 carregados).
async function jaExisteIgual(doc, ignorarId){
  try{
    const snap = await db.collection("lancamentos").where("dataEvento", ">=", doc.dataEvento).where("dataEvento", "<=", doc.dataEvento).get();
    return snap.docs.some(d=>d.id!==ignorarId && chaveLancamento(d.data())===chaveLancamento(doc));
  }catch(e){
    console.warn('Não consegui conferir duplicados', e); // o aviso e uma ajuda: sem ele, salva
    return false;
  }
}
function chavesExistentes(){ return new Set(lancamentosCache.concat(futurosLista()).filter(x=>!x.oculto).map(Recorrencia.chaveDuplicado)); }

// Repeticao ao lancar: so em evento futuro novo (na edicao nao aparece). "Total de ocorrencias" inclui o proprio lancamento.
function atualizarCampoRepetir(){
  const w = document.getElementById('fRepetirWrap');
  if(!w) return;
  const mostrar = !editandoId && dataFuturaForm();
  w.classList.toggle('hidden', !mostrar);
  if(mostrar) atualizarDicaRepeticao();
}
function atualizarDicaRepeticao(){
  const dica = document.getElementById('fRepetirDica');
  const r = lerRepeticao('fr');
  if(!(r.quantidade > 1)){ dica.textContent = 'Total 1 = só este evento. Aumente para criar as repetições já com o aviso acima.'; return; }
  const erro = Recorrencia.validar(r.intervaloMeses, r.quantidade - 1);
  if(erro){ dica.textContent = erro; return; }
  const base = new Date(document.getElementById('fData').value || Date.now());
  if(isNaN(base)){ dica.textContent = ''; return; }
  const datas = Recorrencia.gerarDatas(base, r.intervaloMeses, r.quantidade - 1);
  dica.textContent = `Cria este e mais ${datas.length} (${Recorrencia.rotuloIntervalo(r.intervaloMeses)}), até ${datas[datas.length-1].toLocaleDateString('pt-BR')}.`;
}

// Cria os eventos um a um (o membro tambem pode: e o mesmo POST de um lancamento). Devolve quantos foram criados;
// se um falhar, lanca o erro com .criados para a tela dizer onde parou.
async function criarEventos(docs, aoProgredir){
  let criados = 0;
  for(const doc of docs){
    try{ await db.collection("lancamentos").add({...doc, criadoEm: new Date().toISOString()}); }
    catch(err){ err.criados = criados; throw err; }
    criados++;
    if(aoProgredir) aoProgredir(criados, docs.length);
  }
  return criados;
}

// ---------- Prever e Estender no padrao de telas (.ai/padrao-de-telas.md) ----------
// Tela cheia com tres linhas-resumo (Status das copias, Repetir, Avisar no Telegram), a previa das datas e o botao de
// criar. Cada linha abre uma tela cheia propria; "Outro..." abre rodas numa janela de baixo. As datas, os prazos e as
// frases vem de recorrencia.js (testado em node). Demo: .ai/demos/demo-repeticao-e-aviso.html.
let pv = null;

function tcCk(on){ return `<span class="tc-ck${on?' on':''}" aria-hidden="true">${on?'✓':''}</span>`; }
function tcCabeca(titulo, acao){ return `<div class="tc-hd"><button type="button" data-a="${acao}" aria-label="Voltar">←</button><h2>${esc(titulo)}</h2></div>`; }
function tcLinha(para, titulo, valor){
  return `<button type="button" class="tc-li" data-a="ir" data-to="${para}"><span class="t"><b>${titulo}</b><span class="s">${valor}</span></span><span class="go" aria-hidden="true">›</span></button>`;
}
function tcInterruptor(a, titulo, sub, ligado){
  return `<div class="tc-li"><span class="t"><b>${titulo}</b>${sub?`<span class="s">${sub}</span>`:''}</span><button type="button" class="tc-sw" role="switch" aria-checked="${ligado}" data-a="${a}" aria-label="${esc(titulo)}"><i></i></button></div>`;
}
function tcOpcao(papel, a, rotulo, marcado, dado){
  return `<button type="button" class="tc-li" role="${papel}" aria-checked="${marcado}" data-a="${a}" ${dado}><span class="t">${rotulo}</span>${tcCk(marcado)}</button>`;
}
function pvData(iso){ return new Date(iso).toLocaleDateString('pt-BR'); }

// Copias a criar, com o que ja existe marcado, a partir do estado das telas.
function pvCalcular(){
  if(pv.fatura) return {docs:[pv.base], iguais:new Set(), erro:''}; // o aviso da fatura nao cria copias
  const aviso = Recorrencia.avisoDoEstado(pv.aviso);
  let erro = Recorrencia.validar(pv.meses, pv.qtd);
  if(!erro && aviso === 'incompleto') erro = 'Escolha pelo menos um prazo de aviso ou ligue "Insistir depois do vencimento".';
  let docs = [];
  if(!erro) docs = Recorrencia.montarCopias(pv.base, {intervaloMeses:pv.meses, quantidade:pv.qtd, status:pv.status,
    aviso: aviso || null, numerar:pv.numerar, serieId:pv.ctx.serieId, indiceInicial:pv.ctx.indiceInicial});
  const existentes = chavesExistentes();
  const iguais = new Set();
  docs.forEach((d,i)=>{ if(existentes.has(Recorrencia.chaveDuplicado(d))) iguais.add(i); });
  return {docs, iguais, erro};
}

function pvTelaPrincipal(){
  const c = pv.calc = pvCalcular(), b = pv.base, qtd = c.docs.length;
  const valor = Math.abs(valorAssinado(b));
  const datas = c.docs.slice(0,4).map(d=>pvData(d.dataEvento)).join(', ') + (qtd>4 ? ` e mais ${qtd-4}` : '');
  let h = tcCabeca(pv.ctx.estender ? 'Estender a série' : 'Prever próximas datas', 'fechar');
  h += `<p class="tc-ctx">${pv.ctx.estender?'A partir de: ':''}${esc(b.descricao)} · ${fmtMoeda(valor)} · ${esc(b.status)} · ${dataBR(b)}</p>`;
  h += `<div class="tc-card">${tcLinha('rep','Repetir',Recorrencia.fraseRepeticao(pv.qtd,pv.meses))}${tcLinha('avi','Avisar no Telegram',Recorrencia.fraseAviso(pv.aviso))}</div>`;
  if(pv.temParcela) h += `<div class="tc-card">${tcInterruptor('num','Continuar a numeração das parcelas','2/10 vira 3/10, 4/10…',pv.numerar)}</div>`;
  if(!c.erro) h += `<p class="tc-nota">Cria <b>${qtd}</b> evento${qtd===1?'':'s'}: <b>${datas}</b><br>Total <b>${fmtMoeda(valor*qtd)}</b>${pv.numerar && pv.qtd>qtd ? ` · a numeração só comporta mais ${qtd}` : ''}</p>`;
  if(c.iguais.size){
    const lista = [...c.iguais].slice(0,4).map(i=>pvData(c.docs[i].dataEvento)).join(', ');
    h += `<p class="tc-nota" style="color:var(--danger)"><b style="color:var(--danger)">${c.iguais.size}</b> já exist${c.iguais.size===1?'e':'em'}: ${lista}</p>`+
      `<div class="tc-card">${tcOpcao('radio','dup','Pular os que já existem',pv.dup==='pular','data-v="pular"')}${tcOpcao('radio','dup','Criar mesmo assim',pv.dup==='criar','data-v="criar"')}</div>`;
  }
  const msg = pv.erro || c.erro;
  if(msg) h += `<div class="tc-erro" role="alert">${esc(msg)}</div>`;
  return h;
}
function pvBarra(){
  const n = pv.calc ? pv.calc.docs.length : 0;
  const rotulo = pv.ocupado ? pv.progresso : (n ? `Criar ${n} evento${n===1?'':'s'}` : 'Criar');
  return `<div class="tc-barra"><button type="button" class="tc-criar" id="tcCriar" data-a="criar" aria-busy="${pv.ocupado}">${rotulo}</button></div>`;
}
function pvTelaRepetir(){
  const ehPre = Recorrencia.INTERVALOS.some(i=>i.meses===pv.meses);
  let linhas = Recorrencia.INTERVALOS.map(i=>tcOpcao('radio','int',i.rotulo,i.meses===pv.meses,`data-m="${i.meses}"`)).join('');
  if(!ehPre){ const t = Recorrencia.rotuloIntervalo(pv.meses); linhas += tcOpcao('radio','int',t.charAt(0).toUpperCase()+t.slice(1),true,`data-m="${pv.meses}"`); }
  const c = pvCalcular();
  const datas = c.docs.map(d=>pvData(d.dataEvento)).join(', ');
  return tcCabeca('Repetir','voltar')+
    `<div class="tc-card">${linhas}</div>`+
    '<div class="tc-card"><button type="button" class="tc-li" data-a="janela" data-tipo="int"><span class="t">Outro intervalo</span><span class="go" aria-hidden="true">›</span></button></div>'+
    `<div class="tc-card"><button type="button" class="tc-li" data-a="janela" data-tipo="qtd"><span class="t"><b>Repetir quantas vezes</b><span class="s">Além deste lançamento</span></span><span style="color:var(--muted)">${pv.qtd}</span><span class="go" aria-hidden="true">›</span></button></div>`+
    (c.erro ? '' : `<p class="tc-nota">Datas: <b>${datas}</b><br>Dia que não existe no mês vira o último (31/01 mensal: 28/02, 31/03).</p>`);
}
function pvBalao(c){
  const ds = [...pv.aviso.dias].sort((a,b)=>b-a);
  if(!ds.length) return '';
  const d = ds[0], ref = c.docs[0] || pv.base, venc = new Date(ref.dataEvento);
  const aviso = new Date(venc); aviso.setDate(aviso.getDate()-d);
  const saida = ref.tipo !== 'entrada';
  const quando = d===0 ? (saida?'Vence hoje':'A receber hoje') : `${saida?'Vence em':'A receber em'} ${d} ${d===1?'dia':'dias'}`;
  return `<div class="tc-card" style="margin-top:10px"><div style="padding:12px 16px;font-size:13px"><div style="color:var(--faint);margin:0 0 6px">Assim chega no Telegram</div>`+
    `<div>🔔 Avisos de ${pvData(aviso)}<br>• ${quando} (${pvData(venc)}): ${esc(ref.descricao)} — ${fmtMoeda(Math.abs(valorAssinado(ref)))} (${esc(ref.banco||'-')})</div></div></div>`;
}
function pvTelaAvisar(){
  const av = pv.aviso, c = pvCalcular();
  const lista = Recorrencia.listaDePrazos(av.dias).map(d=>tcOpcao('checkbox','dia',Recorrencia.rotuloPrazo(d),av.dias.includes(d),`data-d="${d}"`)).join('');
  const n = av.dias.length, max = Recorrencia.MAX_PRAZOS;
  const outro = n >= max
    ? `<button type="button" class="tc-li cheio" data-a="cheio"><span class="t">Outro prazo<span class="s">Limite de ${max} prazos. Desmarque um para adicionar outro.</span></span></button>`
    : `<button type="button" class="tc-li" data-a="janela" data-tipo="prazo"><span class="t">Outro prazo<span class="s">${n} de ${max} prazos</span></span><span class="go" aria-hidden="true">›</span></button>`;
  const venc = new Date((c.docs[0] || pv.base).dataEvento);
  const datas = [...av.dias].sort((a,b)=>b-a).map(d=>{ const x = new Date(venc); x.setDate(x.getDate()-d); return `<b>${pvData(x)}</b>`; }).join(', ');
  return tcCabeca('Avisar no Telegram','voltar')+
    (pv.fatura ? `<p class="tc-ctx">${pv.fatura.padrao ? 'Aviso padrão' : 'Fatura ' + esc(pv.fatura.rotulo)} · ${esc(pv.fatura.meio)}</p>` : '')+
    `<div class="tc-card">${tcInterruptor('sw','Avisar','',av.on)}</div>`+
    (av.on
      ? `<div class="tc-card">${lista}</div><div class="tc-card">${outro}</div>`+
        `<div class="tc-card">${tcInterruptor('ins','Insistir depois do vencimento','Avisa todo dia, até pagar',av.insistir)}</div>`+
        (pv.fatura && pv.fatura.padrao
          ? `<p class="tc-nota">Para cada fatura, avisa ${datas||'nenhum prazo'}${av.insistir?' e todo dia depois':''} do vencimento. Vale quando a fatura ganha o vencimento.</p>${pvBalao(c)}`
          : `<p class="tc-nota">Para um vencimento em ${pvData(venc)}, avisa em ${datas||'nenhum dia'}${av.insistir?' e todo dia depois':''}.</p>${pvBalao(c)}`)
      : `<p class="tc-nota">${pv.fatura && pv.fatura.padrao ? 'Sem aviso padrão, as faturas novas ficam sem aviso até você ligar em Futuros.' : 'Sem aviso, o lançamento só aparece em Futuros.'}</p>`)+
    (pv.erro ? `<div class="tc-erro" role="alert">${esc(pv.erro)}</div>` : '');
}

// Janela de baixo com rodas (Outro prazo, Outro intervalo, Repetir quantas vezes).
const PV_JANELAS = {prazo:'Outro prazo', int:'Outro intervalo', qtd:'Repetir quantas vezes'};
function pvHtmlJanela(){
  const j = pv.sheet;
  if(!j) return '';
  return `<div class="tc-ov"><div class="tc-sheet" role="dialog" aria-modal="true" aria-label="${PV_JANELAS[j.tipo]}"><h3>${PV_JANELAS[j.tipo]}</h3>`+
    '<div class="tc-live" id="tcLive"></div><div class="tc-wheels" id="tcWheels"></div>'+
    '<div class="tc-btns"><button type="button" class="no" data-a="no">Cancelar</button><button type="button" class="ok" data-a="ok">OK</button></div></div></div>';
}
function pvValidarJanela(j){
  if(j.tipo === 'prazo') return Recorrencia.validarPrazoNovo(pv.aviso.dias, j.n, j.un===1);
  if(j.tipo === 'int') return Recorrencia.validarIntervaloNovo(j.n, j.un===1);
  return {valor:j.n, texto:`Repetir ${j.n} ${j.n===1?'vez':'vezes'}, além deste lançamento`};
}
function pvPintarJanela(){
  const r = pvValidarJanela(pv.sheet), el = document.getElementById('tcLive');
  if(!el) return;
  el.className = 'tc-live' + (r.erro ? ' err' : '');
  el.textContent = r.erro || r.texto;
}
function tcRoda(itens, sel, aoMudar){
  const caixa = document.createElement('div');
  caixa.className = 'tc-wh';
  caixa.innerHTML = '<div class="bar"></div><div class="sc"></div>';
  const sc = caixa.querySelector('.sc');
  itens.forEach((t,i)=>{
    const d = document.createElement('div');
    d.className = 'it' + (i===sel ? ' on' : ''); d.textContent = t;
    d.onclick = ()=>sc.scrollTo({top:i*44, behavior:'smooth'});
    sc.appendChild(d);
  });
  let espera;
  sc.addEventListener('scroll', ()=>{
    clearTimeout(espera);
    espera = setTimeout(()=>{
      const i = Math.max(0, Math.min(itens.length-1, Math.round(sc.scrollTop/44)));
      sc.querySelectorAll('.it').forEach((e,k)=>e.classList.toggle('on', k===i));
      aoMudar(i);
    }, 90);
  });
  setTimeout(()=>{ sc.scrollTop = sel*44; }, 0);
  return caixa;
}
function pvMontarRodas(){
  const j = pv && pv.sheet, alvo = document.getElementById('tcWheels');
  if(!j || !alvo) return;
  const nums = (a,b)=>Array.from({length:b-a+1}, (_,i)=>String(a+i));
  const maximo = j.tipo==='qtd' ? Recorrencia.MAX_OCORRENCIAS : (j.tipo==='prazo' ? Recorrencia.MAX_DIAS_AVISO : 60);
  alvo.appendChild(tcRoda(nums(1,maximo), j.n-1, i=>{ j.n = i+1; pvPintarJanela(); }));
  if(j.tipo==='prazo') alvo.appendChild(tcRoda(['dia','semana'], j.un, i=>{ j.un = i; pvPintarJanela(); }));
  if(j.tipo==='int') alvo.appendChild(tcRoda(['mês','ano'], j.un, i=>{ j.un = i; pvPintarJanela(); }));
  pvPintarJanela();
}

function pvRender(topo){
  const el = document.getElementById('tc');
  if(!el || !pv) return;
  const antes = el.querySelector('.tc-scr');
  const y = (!topo && antes) ? antes.scrollTop : 0;
  const tela = pv.tela==='rep' ? pvTelaRepetir() : pv.tela==='avi' ? pvTelaAvisar() : pvTelaPrincipal();
  el.innerHTML = `<div class="tc-scr"><div class="tc-col">${tela}</div></div>${pv.fatura ? pvBarraFatura() : pv.tela==='main' ? pvBarra() : ''}${pvHtmlJanela()}`;
  el.querySelector('.tc-scr').scrollTop = y;
  pvMontarRodas();
  Voltar.sync(); // as telas internas (Repetir, Avisar, Status) e a janela de rodas contam como camadas do botao voltar
}
function pvVoltar(){
  if(!pv || pv.ocupado) return;
  if(pv.sheet){ pv.sheet = null; return pvRender(false); }
  if(pv.tela !== 'main' && !pv.fatura){ pv.tela = 'main'; return pvRender(true); }
  fecharPrever();
}
function fecharPrever(){
  const el = document.getElementById('tc');
  if(el) el.remove();
  document.documentElement.style.overflow = '';
  pv = null;
}
document.addEventListener('keydown', e=>{ if(e.key === 'Escape' && pv) pvVoltar(); });

async function pvCriar(){
  if(!pv || pv.ocupado) return;
  const c = pvCalcular();
  if(c.erro){ pv.erro = c.erro; return pvRender(false); }
  const pular = c.iguais.size > 0 && pv.dup === 'pular';
  const aCriar = c.docs.filter((_,i)=>!(pular && c.iguais.has(i)));
  if(!aCriar.length){ pv.erro = 'Todos os eventos já existem: nada a criar.'; return pvRender(false); }
  pv.ocupado = true; pv.erro = ''; pv.progresso = `Criando 0/${aCriar.length}…`; pvRender(false);
  try{
    const n = await criarEventos(aCriar, (feitos,total)=>{
      if(!pv) return;
      pv.progresso = `Criando ${feitos}/${total}…`;
      const b = document.getElementById('tcCriar'); if(b) b.textContent = pv.progresso;
    });
    fecharPrever();
    toast(`✓ ${n} evento${n===1?'':'s'} criado${n===1?'':'s'}`);
  }catch(err){
    console.error(err);
    await new Promise(r=>setTimeout(r, 400));   // o que ja foi criado entra no cache; a previa volta a apontar o "ja existe"
    if(!pv) return;
    pv.ocupado = false; pv.dup = 'pular';
    pv.erro = `Parou depois de ${err.criados||0} de ${aCriar.length}: ${msgErro(err,'erro ao salvar')} Os que foram criados aparecem como "já existem"; deixe "Pular os que já existem" e tente de novo.`;
    pvRender(false);
  }
}

function pvClique(e){
  const b = e.target.closest('[data-a]');
  if(!b || !pv) return;
  const a = b.dataset.a, av = pv.aviso;
  if(a==='fechar') return pvVoltar();
  if(a==='voltar' && pv.fatura) return pvVoltar();
  if(a==='salvarFatura') return pvSalvarFatura();
  if(a==='voltar'){ pv.tela = 'main'; pv.sheet = null; return pvRender(true); }
  if(a==='ir'){ pv.tela = b.dataset.to; pv.erro = ''; return pvRender(true); }
  if(a==='criar') return pvCriar();
  if(a==='num'){ pv.numerar = !pv.numerar; return pvRender(false); }
  if(a==='dup'){ pv.dup = b.dataset.v; return pvRender(false); }
  if(a==='int'){ pv.meses = parseInt(b.dataset.m,10); return pvRender(false); }
  if(a==='sw'){ av.on = !av.on; return pvRender(false); }
  if(a==='ins'){ av.insistir = !av.insistir; return pvRender(false); }
  if(a==='dia'){
    const d = parseInt(b.dataset.d,10), i = av.dias.indexOf(d);
    if(i >= 0) av.dias.splice(i,1); else av.dias.push(d);
    return pvRender(false);
  }
  if(a==='cheio') return;
  if(a==='janela'){
    pv.sheet = {tipo:b.dataset.tipo, n:b.dataset.tipo==='qtd' ? pv.qtd : (b.dataset.tipo==='prazo' ? 5 : 4), un:0};
    return pvRender(false);
  }
  if(a==='no'){ pv.sheet = null; return pvRender(false); }
  if(a==='ok'){
    const j = pv.sheet, r = pvValidarJanela(j);
    if(r.erro) return;
    if(j.tipo==='prazo') av.dias.push(r.valor);
    if(j.tipo==='int') pv.meses = r.valor;
    if(j.tipo==='qtd') pv.qtd = r.valor;
    pv.sheet = null; return pvRender(false);
  }
}

// Abre o fluxo. base = o evento de partida (o ultimo da serie, no Estender); ctx = { estender, serieId, indiceInicial, intervaloPadrao }.
function abrirPrever(base, ctx){
  if(!listas.status.length){ avisar('Não há forma de pagamento disponível para prever.'); return; }
  fecharPrever();
  const aviso = base.aviso && typeof base.aviso === 'object' ? base.aviso : null;
  const temParcela = Recorrencia.lerParcela(base.descricao) !== null;
  pv = {
    base, ctx, temParcela, numerar: temParcela, dup:'pular', tela:'main', sheet:null, erro:'', ocupado:false, progresso:'',
    status: listas.status.includes(base.status) ? base.status : listas.status[0], // as copias saem com o status do original; a data futura e que as faz previstas
    qtd: 3, meses: ctx.intervaloPadrao || 1,
    aviso: {on: !!aviso, dias: aviso && Array.isArray(aviso.dias) ? [...aviso.dias] : [1,0], insistir: aviso ? aviso.insistir === true : true}
  };
  const el = document.createElement('div');
  el.className = 'tc'; el.id = 'tc';
  el.setAttribute('role','dialog'); el.setAttribute('aria-modal','true');
  el.setAttribute('aria-label', ctx.estender ? 'Estender a série' : 'Prever próximas datas');
  el.addEventListener('click', pvClique);
  document.body.appendChild(el);
  document.documentElement.style.overflow = 'hidden';
  pvRender(true);
  const voltar = el.querySelector('.tc-hd button');
  if(voltar) voltar.focus();
}

// Aviso da fatura do cartao (item 46): a MESMA tela "Avisar no Telegram" do Prever, sem a tela principal por tras; o Salvar fica no rodape e a seta so desiste.
// Demo: .ai/demos/demo-aviso-fatura.html. O servidor valida o aviso e exige o vencimento da fatura (que ja existe: so assim a fatura aparece em Futuros).
function pvBarraFatura(){
  return `<div class="tc-barra"><button type="button" class="tc-criar" id="tcCriar" data-a="salvarFatura" aria-busy="${pv.ocupado}">${pv.ocupado ? 'Salvando…' : 'Salvar'}</button></div>`;
}
function abrirAvisoFatura(f){
  if(!f) return;
  fecharPrever();
  const aviso = f.aviso && typeof f.aviso === 'object' ? f.aviso : null;
  pv = {
    fatura:{meio:f.meio, ym:f.ym, rotulo:f.rotulo},
    base:{dataEvento:f.vencimento + 'T12:00:00.000Z', descricao:'Fatura ' + f.rotulo + ' · ' + f.meio, valor:f.valor, tipo:'saida', banco:f.meio},
    ctx:{}, tela:'avi', sheet:null, erro:'', ocupado:false, progresso:'', qtd:1, meses:1,
    aviso:{on:!!aviso, dias:aviso && Array.isArray(aviso.dias) ? [...aviso.dias] : [1,0], insistir:aviso ? aviso.insistir === true : true}
  };
  const el = document.createElement('div');
  el.className = 'tc'; el.id = 'tc';
  el.setAttribute('role','dialog'); el.setAttribute('aria-modal','true');
  el.setAttribute('aria-label','Avisar no Telegram');
  el.addEventListener('click', pvClique);
  document.body.appendChild(el);
  document.documentElement.style.overflow = 'hidden';
  pvRender(true);
  const voltar = el.querySelector('.tc-hd button');
  if(voltar) voltar.focus();
}
// O aviso padrao do cartao (item 48): a mesma tela, aberta pela linha "Aviso (padrao)" de Configurar cartao. E so o modelo que o servidor copia para a fatura
// quando ela ganha o primeiro vencimento; nao muda nenhuma fatura existente.
function abrirAvisoPadrao(meio){
  const cfg = ((cartoesResumo && cartoesResumo.cartoes) || []).find(c=>c.meio===meio) || {};
  const f = cartaoFatura;
  if(!f) return;
  fecharPrever();
  const aviso = cfg.aviso && typeof cfg.aviso === 'object' ? cfg.aviso : null;
  pv = {
    fatura:{meio, ym:null, rotulo:f.rotulo, padrao:true},
    base:{dataEvento:(f.vencimento || f.periodo.fim) + 'T12:00:00.000Z', descricao:'Fatura ' + f.rotulo + ' · ' + meio, valor:f.total, tipo:'saida', banco:meio},
    ctx:{}, tela:'avi', sheet:null, erro:'', ocupado:false, progresso:'', qtd:1, meses:1,
    aviso:{on:!!aviso, dias:aviso && Array.isArray(aviso.dias) ? [...aviso.dias] : [1,0], insistir:aviso ? aviso.insistir === true : true}
  };
  const el = document.createElement('div');
  el.className = 'tc'; el.id = 'tc';
  el.setAttribute('role','dialog'); el.setAttribute('aria-modal','true');
  el.setAttribute('aria-label','Avisar no Telegram');
  el.addEventListener('click', pvClique);
  document.body.appendChild(el);
  document.documentElement.style.overflow = 'hidden';
  pvRender(true);
  const voltar = el.querySelector('.tc-hd button');
  if(voltar) voltar.focus();
}
async function pvSalvarFatura(){
  if(!pv || !pv.fatura || pv.ocupado) return;
  const aviso = Recorrencia.avisoDoEstado(pv.aviso);
  if(aviso === 'incompleto'){ pv.erro = 'Escolha pelo menos um prazo de aviso ou ligue "Insistir depois do vencimento".'; return pvRender(false); }
  const {meio, ym, padrao} = pv.fatura;
  pv.ocupado = true; pv.erro = ''; pvRender(false);
  try{
    if(padrao){
      await db.cartaoConfig(meio, {aviso: aviso || null});
      fecharPrever();
      toast(aviso ? '✓ Aviso padrão salvo. Vale para as próximas faturas que ganharem vencimento' : '✓ Aviso padrão desligado');
      msgCartao = 'Salvo ✓';
      await recarregarCartao();
      return;
    }
    await db.cartaoFechamento(meio, ym, {aviso: aviso || null});
    fecharPrever();
    toast(aviso ? '✓ Aviso da fatura salvo' : '✓ Aviso da fatura desligado');
    carregarFuturos();
  }catch(err){
    console.error(err);
    if(!pv) return;
    pv.ocupado = false; pv.erro = msgErro(err, 'Não foi possível salvar o aviso. Tente de novo.');
    pvRender(false);
  }
}

// Botao Prever (registro sem serie) ou Estender (registro de uma serie: parte da ultima ocorrencia carregada).
async function preverLancamento(id){
  const l = acharLancamento(id);
  if(!l || !db) return;
  await lerDadosParaDuplicados(); // a serie e os duplicados olham os lancamentos e os previstos: lidos so agora
  let base = l, ctx;
  if(l.serie){
    const ultima = Recorrencia.ultimaDaSerie(lancamentosCache.concat(futurosLista()), l.serie.id) || l;
    base = ultima;
    ctx = {estender:true, serieId:l.serie.id, indiceInicial:(ultima.serie ? ultima.serie.indice : l.serie.indice)+1, intervaloPadrao:(ultima.serie||l.serie).intervalo};
  } else {
    ctx = {estender:false, serieId:novoSerieId(), indiceInicial:1};
  }
  abrirPrever(base, ctx);
}

let lancamentosCache = [];
let modoMesEspecifico = false;
function mesAtualYM(){ const d=new Date(); return d.getFullYear()+'-'+String(d.getMonth()+1).padStart(2,'0'); }
let mesSel = mesAtualYM();
let tokenMes = 0;
let lancamentosMes = [];
let modoBuscaTexto = false;
let lancamentosBusca = [];
let periodoHist = '30d';  // '30d' | '90d' | '180d' | 'mes' (demo: .ai/demos/demo-historico-periodo.html)
let tudoHist = false;     // a busca atual olhou todo o histórico
let escopoBuscaTxt = '';  // onde a busca atual olhou, para a faixa do resultado
const DIAS_PERIODO = {'30d':30, '90d':90, '180d':180};
let tagsSelecionadas = new Set();
function clonar(x){ return x===undefined ? x : JSON.parse(JSON.stringify(x)); }
let combosSalvos = [];
let filaCombos = Promise.resolve();
let combosVisiveis = [];
let tagBusca = '';
let tagsTodasAbertas = false;
let seletorHist = null;
let filStatusSelecionados = new Set();
let filBancoSelecionados = new Set();
let editandoId = null;

// Valor do ponto de vista do caixa: saída é negativa e entrada positiva; um valor negativo inverte o
// sentido (reembolso/estorno lançado como saída de -R$ 100 aparece como +R$ 100).
function valorAssinado(l){ return (l.tipo==='saida'?-1:1)*(Number(l.valor)||0); }
function htmlValorLinha(l){
  const v = valorAssinado(l);
  return `<span class="valor ${v<0?'saida':'entrada'}">${v<0?'-':'+'}${fmtMoeda(Math.abs(v))}</span>`;
}
// Papel do usuário (no ambiente do Claude não existe, então vale 'owner'). A trava de verdade está
// na API; aqui só se esconde o que o papel não pode usar.
function ehOwner(){ return papelAtual === 'owner'; }
// Escrita recusada porque o mês está conferido: a mensagem do servidor já diz o que fazer.
function msgErro(err, padrao){ return err && (err.codigo === 'mes_conferido' || err.codigo === 'fatura_conferida' || err.codigo === 'cancelado' || err.code === 400) ? err.message : padrao; } // 400: o servidor explica o que esta errado
// Mes fechado por data (item 38): o servidor recusou com 409 mes_fechado; aqui so se pergunta. As funcoes sao chamadas pelo api.js.
window.confirmarMesFechado = (info, acao)=>confirmarModal(`Este lançamento é de ${rotuloMes(info.mes)}, que já fechou. ${acao} mesmo assim? O saldo desse mês e dos seguintes muda. A confirmação fica gravada no histórico.`, acao + ' mesmo assim');
window.confirmarMesFechadoLote = (n)=>confirmarModal(`${n} lançamento(s) são de meses já fechados e ficaram de fora. Importar esses mesmo assim? O saldo desses meses muda. A confirmação fica gravada no histórico de cada um.`, 'Importar mesmo assim');
window.addEventListener('unhandledrejection', ev=>{
  if(ev.reason && (ev.reason.codigo === 'mes_conferido' || ev.reason.codigo === 'fatura_conferida')){ ev.preventDefault(); toast(ev.reason.message, true); }
});
function podeMexer(l){ return ehOwner() || (!!l.criadoPor && l.criadoPor === meuSub); }
function fmtMoeda(v){ return (v||0).toLocaleString('pt-BR',{style:'currency',currency:'BRL'}); }
function nowLocalInput(){
  const d = new Date(); d.setMinutes(d.getMinutes()-d.getTimezoneOffset());
  return d.toISOString().slice(0,16);
}

let toastTimer;
function toast(msg, erro){
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.classList.toggle('erro', !!erro);
  el.classList.add('on');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(()=>el.classList.remove('on'), erro ? 6000 : 3500);
}
// Aviso com ação ("Desfazer"), com barra de tempo. Um por vez: abrir outro encerra o anterior na hora (onFim).
// Padrão da demo .ai/demos/demo-desfazer.html; a exclusão imediata é da demo .ai/demos/demo-exclusao-imediata.html.
const AVISO_ACAO_MS = 6000;
let avisoAcaoAtual = null, avisoAcaoTimer;
function fecharAvisoAcao(porAcao){
  const a = avisoAcaoAtual;
  if(!a) return;
  avisoAcaoAtual = null;
  clearTimeout(avisoAcaoTimer);
  document.getElementById('avisoAcao').classList.remove('on');
  if(!porAcao && a.onFim) a.onFim();
}
function mostrarAvisoAcao({txt, sub, rotulo, onAcao, onFim, erro, ms}){
  fecharAvisoAcao(false);
  const el = document.getElementById('avisoAcao');
  const dur = ms || AVISO_ACAO_MS;
  el.classList.toggle('erro', !!erro);
  el.innerHTML = '<div class="aa-linha"><span class="aa-tx">' + esc(txt) + (sub ? '<small>' + esc(sub) + '</small>' : '') +
    '</span><button type="button" id="avisoAcaoBtn">' + esc(rotulo) + '</button></div>' +
    '<div class="aa-barra" style="animation-duration:' + dur + 'ms"></div>';
  const a = {onAcao, onFim};
  avisoAcaoAtual = a;
  document.getElementById('avisoAcaoBtn').onclick = ()=>{ if(avisoAcaoAtual !== a) return; fecharAvisoAcao(true); if(onAcao) onAcao(); };
  el.classList.add('on');
  avisoAcaoTimer = setTimeout(()=>fecharAvisoAcao(false), dur);
}
// Sair da aba ou do app encerra o aviso na hora (o Desfazer some; a exclusão já tinha sido feita no servidor).
document.addEventListener('visibilitychange', ()=>{ if(document.visibilityState === 'hidden') fecharAvisoAcao(false); });
window.addEventListener('pagehide', ()=>fecharAvisoAcao(false));

// Desabilita o botão e mostra "Salvando…" enquanto roda; impede clique duplo.
async function comBotaoOcupado(btn, rotulo, fn){
  if(btn.disabled) return;
  const antes = btn.textContent;
  btn.disabled = true; btn.textContent = rotulo;
  try{ return await fn(); }
  finally{
    btn.disabled = false;
    // só restaura se a ação não trocou o texto (ex.: cancelarEdicao põe "Salvar lançamento")
    if(btn.textContent === rotulo) btn.textContent = antes;
  }
}

async function avisar(msg){
  return new Promise(resolve=>{
    document.getElementById('modalMsg').textContent = msg;
    document.getElementById('modalExtra').innerHTML = '';
    document.getElementById('modalBotoes').innerHTML = '<button class="primary" id="modalOk" style="width:100%;">OK</button>';
    document.getElementById('modalOverlay').classList.remove('hidden');
    document.getElementById('modalOk').onclick = ()=>{ document.getElementById('modalOverlay').classList.add('hidden'); resolve(); };
  });
}
function confirmarModal(msg, rotuloOk){
  return new Promise(resolve=>{
    document.getElementById('modalMsg').textContent = msg;
    document.getElementById('modalExtra').innerHTML = '';
    document.getElementById('modalBotoes').innerHTML = '<button class="ghost" id="modalCancelar">Cancelar</button><button class="primary" id="modalConfirmar"></button>';
    document.getElementById('modalConfirmar').textContent = rotuloOk || 'Confirmar';
    document.getElementById('modalOverlay').classList.remove('hidden');
    const fechar = (v)=>{ document.getElementById('modalOverlay').classList.add('hidden'); resolve(v); };
    document.getElementById('modalCancelar').onclick = ()=>fechar(false);
    document.getElementById('modalConfirmar').onclick = ()=>fechar(true);
  });
}

// Pede o que fazer com uma tag em uso: retorna null (cancelou), '' (só remover a tag,
// sem substituir) ou o nome da tag nova pra migrar os lançamentos.
function pedirNovaTagParaMigrar(tag, qtd, outrasTags){
  return new Promise(resolve=>{
    document.getElementById('modalMsg').textContent =
      `${qtd} lançamento(s) usam a tag "${tag}". Escolha uma tag pra substituir (ou deixe em branco pra só remover essa tag desses lançamentos):`;
    document.getElementById('modalExtra').innerHTML = `
      <input type="text" id="modalNovaTag" list="modalListaTags" placeholder="nova tag (opcional)" style="margin-bottom:10px;">
      <datalist id="modalListaTags">${outrasTags.map(t=>`<option value="${esc(t)}">`).join('')}</datalist>
    `;
    document.getElementById('modalBotoes').innerHTML = '<button class="ghost" id="modalCancelar">Cancelar</button><button class="primary" id="modalConfirmar">Continuar</button>';
    document.getElementById('modalOverlay').classList.remove('hidden');
    const fechar = (v)=>{ document.getElementById('modalOverlay').classList.add('hidden'); resolve(v); };
    document.getElementById('modalCancelar').onclick = ()=>fechar(null);
    document.getElementById('modalConfirmar').onclick = ()=>{
      const v = document.getElementById('modalNovaTag').value.trim().replace(/^#/,'');
      fechar(v);
    };
  });
}

async function excluirTag(tag){
  if(!db) return;
  const snap = await db.collection("lancamentos").where("tags","array-contains",tag).limit(1000).get();
  const qtd = snap.docs.length;
  let migrada = '';
  if(qtd>0){
    const novaTag = await pedirNovaTagParaMigrar(tag, qtd, tagsUsadas.filter(t=>t!==tag));
    if(novaTag===null) return; // cancelou
    migrada = novaTag || '';
    for(const d of snap.docs){
      const l = d.data();
      const restante = new Set((l.tags||[]).filter(t=>t!==tag));
      if(novaTag) restante.add(novaTag);
      await db.collection("lancamentos").doc(d.id).update({tags:[...restante]});
    }
    if(novaTag && !tagsUsadas.includes(novaTag)) tagsUsadas.push(novaTag);
  }
  tagsUsadas = tagsUsadas.filter(t=>t!==tag);
  await db.doc("config/tags").set({usadas:tagsUsadas});
  combosSalvos = combosSalvos
    .map(c=>({tags:[...new Set(c.tags.map(t=>t===tag ? migrada : t).filter(Boolean))]}))
    .filter(c=>c.tags.length>=2);
  await salvarCombos();
  renderTagPicker(); renderTagFilter(); renderTagChartSelect();
  renderGerenciarTags(); renderCombosLista();
}

async function init(){
  db = await claude.use("db");
  sample = await claude.use("sample");
  if(window.claude && claude.papel){
    papelAtual = claude.papel() || 'owner';
    meuSub = (claude.sub && claude.sub()) || '';
  }
  document.body.classList.toggle('papel-member', papelAtual !== 'owner');
  if(window.claude && claude.definirApelido){
    document.getElementById('perfilNome').value = (claude.apelido && claude.apelido()) || '';
    usAtualizar();
    document.getElementById('telegramChatId').value = (claude.telegramChatId && claude.telegramChatId()) || '';
  } else {
    document.getElementById('perfilCard').classList.add('hidden'); // ambiente sem perfil (artifact do Claude)
    document.getElementById('telegramCard').classList.add('hidden');
  }
  if(papelAtual !== 'owner'){
    const dica = document.getElementById('saldoDica');
    if(dica) dica.textContent = 'Saldos por banco';
  }
  document.getElementById('fData').value = nowLocalInput();

  if(db){
    db.doc("config/listas").onSnapshot(snap=>{
      const d = snap.data();
      if(d && d.status) listas.status = clonar(d.status);
      if(d && d.banco) listas.banco = clonar(d.banco);
      if(d && Array.isArray(d.meios)) meiosLista = clonar(d.meios);
      if(d && d.faturaPadrao) faturaPadrao = clonar(d.faturaPadrao);
      if(d && d.statusCorPadrao) coresPadrao = clonar(d.statusCorPadrao);
      if(d && d.statusCor) statusCor = clonar(d.statusCor);
      tema = (d && d.tema) || 'auto'; Tema.escolher(tema);
      if(d && d.estiloCor) estiloCor = d.estiloCor;
      if(d && d.layoutSaldos) layoutSaldos = d.layoutSaldos;
      if(d && d.ordemSaldos) ordemSaldos = d.ordemSaldos;
      if(d && Array.isArray(d.ordemManual)) ordemManual = clonar(d.ordemManual);
      // A lista de status e as caixas sao do servidor (item 41): o app nao completa nem regrava nada aqui.
      renderListas();
      renderSaldos();
      if(abaAtual()==='historico'){ try{ renderHistorico(); }catch(e){} }
    }, ()=>{ renderListas(); });

    db.doc("config/tags").onSnapshot(snap=>{
      const d = snap.data();
      tagsUsadas = clonar((d && d.usadas) || []);
      renderTagPicker();
      renderTagFilter();
      renderTagChartSelect();
      renderGerenciarTags();
    }, ()=>{});

    db.doc("config/combos").onSnapshot(snap=>{
      const d = snap.data();
      combosSalvos = (d && Array.isArray(d.lista)) ? clonar(d.lista) : [];
      renderTagPicker();
      renderCombosLista();
    }, ()=>{});

    // Cada tela le o que e dela, so quando esta aberta (irParaAba): Lancar usa so as configuracoes acima. Os lancamentos
    // (1000 mais recentes) sao lidos sob demanda por Historico e Tendencias; Futuros le por conta propria (carregarFuturos).
    lancsOuv = db.collection("lancamentos").orderBy("dataEvento","desc").limit(1000).sobDemanda(snap=>{
      lancamentosCache = snap.docs.map(d=>({id:d.id, ...d.data()})).filter(l=>!exclusaoPendente.has(l.id) && !l.previsto); // previsto e de Futuros
      try{ renderTagPicker(); }catch(e){}
      cartaoUltimaCalc = 0;
      const aba = abaAtual();
      if(aba==='historico') renderHistorico();
      if(aba==='tendencias'){ try{ desenharGrafico(); }catch(e){} }
    }, ()=>{ document.getElementById('listaLancamentos').innerHTML = '<p class="muted">Sem acesso ao histórico nesta visualização.</p>'; });
    // Futuros muda quando um lancamento muda (confirmar, excluir, editar): se a tela esta aberta le de novo; se nao, so marca.
    if(db.aoMudarLancamentos) db.aoMudarLancamentos(()=>{ if(abaAtual()==='futuros') agendarFuturos(); else futurosSujo = true; });
  } else {
    document.getElementById('iaStatus').textContent = "Banco de dados indisponível nesta visualização.";
  }
  renderListas();
  setupTabs();
  setupHandlers();
  ligarVoltar();
}

// ---------- botao voltar do navegador e do celular (web/js/voltar.js; padrao em .ai/padrao-de-telas.md) ----------
// Camadas = janelas e telas cheias que o "←" ou o toque fora fecharia. Quem abre ou fecha continua sendo o codigo de sempre;
// voltar.js so acompanha a contagem (MutationObserver nos atributos e nos filhos do body, e Voltar.sync() onde o DOM nao muda).
function camadaVisivel(id){ const e = document.getElementById(id); return !!e && !e.hidden && !e.classList.contains('hidden'); }
function camadasAbertas(){
  let n = 0;
  if(camadaVisivel('modalOverlay')) n++;
  if(camadaVisivel('mnDrop')) n++;
  if(camadaVisivel('tdListaVeu')) n++;
  if(camadaVisivel('tdVeu')) n++;
  if(document.body.classList.contains('td-edit')){ n++; if(camadaVisivel('tdExpr') && window.matchMedia('(max-width:1159px)').matches) n++; }
  if(pv) n += 1 + (pv.tela !== 'main' ? 1 : 0) + (pv.sheet ? 1 : 0);
  if(document.getElementById('vsTc')) n++;
  if(document.getElementById('futOv')) n++;
  if(camadaVisivel('histFilOv')) n++;
  if(document.getElementById('tagsCard').classList.contains('aberto')) n++;
  return n;
}
function fecharCamadaTopo(){ // na ordem de cima para baixo; o que nao existe e pulado
  const clicar = sel=>{ const e = document.querySelector(sel); if(e) e.click(); return !!e; };
  if(camadaVisivel('modalOverlay')){ clicar('#modalCancelar') || clicar('#modalOk'); return; }
  if(camadaVisivel('mnDrop')){ fecharMenuApp(false); return; }
  if(camadaVisivel('tdListaVeu')){ clicar('#tdLFechar'); return; }
  if(camadaVisivel('tdVeu')){ clicar('#tdJok'); return; }
  if(document.body.classList.contains('td-edit')){
    if(camadaVisivel('tdExpr') && window.matchMedia('(max-width:1159px)').matches){ clicar('#tdVoltarCel') || clicar('#tdVoltar'); return; }
    clicar('#tdFechar'); return;
  }
  if(pv){ pvVoltar(); return; }
  if(document.getElementById('vsTc')){ vsFechar(); return; }
  if(document.getElementById('futOv')){ fecharAcoesFuturo(); return; }
  if(camadaVisivel('histFilOv')){ fecharFiltrosHist(); return; }
  if(document.getElementById('tagsCard').classList.contains('aberto')) lancarTags(false);
}
// Edicao com texto digitado: o voltar pergunta antes de sair do formulario.
let assinaturaEdicao = null;
function assinaturaFormulario(){
  const v = id=>document.getElementById(id).value;
  return JSON.stringify([v('fTipo'), v('fValor'), v('fDesc'), v('fNota'), v('fData'), v('fBanco'), v('fStatus'), v('fFatura'),
    document.getElementById('fEstimado').checked, [...tagsSelecionadas].sort()]);
}
function formularioSujo(){ return !!editandoId && assinaturaEdicao !== null && abaAtual() === 'lancar' && assinaturaFormulario() !== assinaturaEdicao; }
async function perguntarDescartarEdicao(){
  if(await confirmarModal('Descartar o que foi digitado? A alteração ainda não foi salva e será perdida.', 'Descartar')) cancelarEdicao();
}
function ligarVoltar(){
  if(!Voltar || !window.history || !history.pushState) return;
  Voltar.iniciar({
    abaAtual,
    abaValida: t=>{
      const b = document.querySelector(`nav button[data-tab="${t}"]`);
      return !!b && !b.hidden && !b.classList.contains('hidden') && b.style.display !== 'none' && !!document.getElementById('tab-'+t);
    },
    irParaAba,
    contarAbertas: camadasAbertas,
    fecharTopo: fecharCamadaTopo,
    precisaConfirmar: formularioSujo,
    perguntarSaida: perguntarDescartarEdicao
  });
  const ouvir = ()=>Voltar.sync();
  new MutationObserver(ouvir).observe(document.body, {subtree:true, attributes:true, attributeFilter:['class','hidden']});
  new MutationObserver(ouvir).observe(document.body, {childList:true});
}

let abaOrigemEdicao = null;
function abaAtual(){ const b = document.querySelector('nav button.active'); return b ? b.dataset.tab : 'lancar'; }
function irParaAba(t){
  document.body.dataset.aba = t;
  const nomeAba = document.querySelector(`nav button[data-tab="${t}"]`);
  if(nomeAba) document.getElementById('mnTela').textContent = nomeAba.textContent;
  document.querySelectorAll('nav button').forEach(x=>x.classList.toggle('active', x.dataset.tab===t));
  document.querySelectorAll('.mn-at').forEach(x=>{
    if(x.dataset.atalho===t) x.setAttribute('aria-current','page'); else x.removeAttribute('aria-current');
  });
  document.querySelectorAll('main section').forEach(sec=>sec.classList.add('hidden'));
  document.getElementById('tab-'+t).classList.remove('hidden');
  // Cada tela le so o que e dela. Historico e Tendencias leem os lancamentos (sob demanda); as outras saem deles.
  const usaLancamentos = t==='historico' || t==='tendencias';
  if(lancsOuv && !usaLancamentos) lancsOuv.pausar();
  if(t==='saldos') carregarSaldosConfig().then(()=>carregarSaldos(false));
  if(t==='cartao') carregarCartoes(false);
  if(t==='futuros') carregarFuturos();
  if(t==='versoes') carregarScript('js/versoes.js').then(()=>{ if(abaAtual()==='versoes') abrirVersoes(); });
  if(t==='configuracoes') carregarScript(XLSX_URL).catch(e=>console.error(e)); // a leitura de planilhas (~1 MB) so serve a importacao
  if(t==='historico') entrarHistorico();
  if(t==='tendencias') entrarTendencias();
  Voltar.sync();
}
// scripts de telas que nem toda sessao abre: baixados na primeira visita
const XLSX_URL = 'https://cdnjs.cloudflare.com/ajax/libs/xlsx/0.18.5/xlsx.full.min.js';
const scriptsCarregados = new Map();
function carregarScript(src){
  if(!scriptsCarregados.has(src)){
    scriptsCarregados.set(src, new Promise((ok, falha)=>{
      const s = document.createElement('script');
      s.src = src; s.onload = ok;
      s.onerror = ()=>{ scriptsCarregados.delete(src); falha(new Error('Não consegui carregar ' + src)); };
      document.body.appendChild(s);
    }));
  }
  return scriptsCarregados.get(src);
}
let lancsOuv = null; // leitura sob demanda dos 1000 lancamentos mais recentes (ver o init)
async function entrarHistorico(){
  if(lancsOuv && !lancsOuv.carregado()){
    document.getElementById('contadorHistorico').textContent = 'Buscando...';
    document.getElementById('somaHistorico').innerHTML = '';
    document.getElementById('listaLancamentos').innerHTML = '';
  }
  if(lancsOuv) await lancsOuv.ativar();
  if(abaAtual()==='historico') renderHistorico();
}
async function entrarTendencias(){
  const raiz = document.getElementById('tendRaiz');
  if(raiz && !raiz.children.length) raiz.innerHTML = '<p class="muted">Carregando...</p>';
  try{
    await carregarScript('js/tendencias.js');
    if(lancsOuv) await lancsOuv.ativar();
  }catch(e){ console.error(e); if(raiz) raiz.innerHTML = '<p class="muted">Não consegui carregar a tela. Tente de novo.</p>'; return; }
  if(abaAtual()==='tendencias') tendRender();
}
function setupTabs(){
  document.querySelectorAll('nav button').forEach(b=>{
    b.addEventListener('click', ()=>irParaAba(b.dataset.tab));
  });
  setupMenuApp();
}

// ---------- menu do app no celular (⋮): a lista vem das abas em botoes, que continuam sendo a fonte das telas ----------
const MENU_APP = [
  ['Dia a dia', [['lancar','✏️'],['historico','🕘'],['futuros','📅']]],
  ['Acompanhar', [['cartao','💳'],['saldos','🏦'],['tendencias','📈']]],
  ['Configurar', [['listas','🏷️'],['versoes','🧾'],['configuracoes','⚙️']]]
];
function fecharMenuApp(voltarFoco){
  const btn = document.getElementById('mnBtn');
  document.getElementById('mnDrop').hidden = true;
  document.getElementById('mnShade').hidden = true;
  btn.setAttribute('aria-expanded', 'false');
  if(voltarFoco) btn.focus();
}
function abrirMenuApp(){
  const atual = abaAtual();
  const nomeDe = t=>{ const b = document.querySelector(`nav button[data-tab="${t}"]`); return b ? b.textContent : t; };
  // topo: a conta (nome, espaco); meio: as telas; fim: Sair (usuario.js)
  document.getElementById('mnDrop').innerHTML = htmlContaTopo() + MENU_APP.map(([grupo, itens], i)=>
    (i ? '<hr class="mn-sep">' : '') + `<div class="mn-grp">${grupo}</div>` +
    itens.map(([t, ic])=>`<button type="button" class="mn-it${t===atual?' on':''}" role="menuitem" data-tab="${t}"><span class="mn-ic">${ic}</span>${esc(nomeDe(t))}${t===atual?'<span class="mn-ck">✓</span>':''}</button>`).join('')
  ).join('') + htmlContaFim();
  document.getElementById('mnDrop').hidden = false;
  document.getElementById('mnShade').hidden = false;
  document.getElementById('mnBtn').setAttribute('aria-expanded', 'true');
  const foco = document.querySelector('#mnDrop .mn-it.on') || document.querySelector('#mnDrop .mn-it');
  if(foco) foco.focus();
}
function setupMenuApp(){
  const btn = document.getElementById('mnBtn'), drop = document.getElementById('mnDrop');
  btn.addEventListener('click', ()=>{ if(drop.hidden) abrirMenuApp(); else fecharMenuApp(false); });
  document.querySelectorAll('.mn-at').forEach(a=>a.addEventListener('click', ()=>irParaAba(a.dataset.atalho)));
  document.getElementById('mnShade').addEventListener('click', ()=>fecharMenuApp(false));
  drop.addEventListener('click', e=>{
    const it = e.target.closest('.mn-it[data-tab]'); if(!it) return; // Sair e o bloco da conta (usuario.js) tem os seus
    fecharMenuApp(false);
    irParaAba(it.dataset.tab);
  });
  document.addEventListener('keydown', e=>{ if(e.key === 'Escape' && !drop.hidden) fecharMenuApp(true); });
}

function renderListas(){
  const fStatus=document.getElementById('fStatus'), fBanco=document.getElementById('fBanco');
  const tOrig=document.getElementById('tBancoOrigem'), tDest=document.getElementById('tBancoDestino');
  // reconstruir os <option> zera o select para o primeiro item; guarda e devolve a escolha atual (se ainda existir)
  const refazer = (sel, itens)=>{
    const atual = sel.value;
    sel.innerHTML = itens.map(s=>`<option>${esc(s)}</option>`).join('');
    if(atual && itens.includes(atual)) sel.value = atual;
  };
  refazer(fStatus, listas.status);
  renderMarcaFatura();
  atualizarCampoFatura();
  [fBanco,tOrig,tDest].forEach(sel=>refazer(sel, listas.banco));
  // tira da seleção qualquer status que não existe mais
  filStatusSelecionados = new Set([...filStatusSelecionados]);
  renderFiltrosHistCampos();
  filBancoSelecionados = new Set([...filBancoSelecionados].filter(b=>listas.banco.includes(b)));

  document.getElementById('pillStatus').innerHTML = listas.status.map(s=>
    `<span class="pill">
      <span style="display:flex;align-items:center;gap:6px;"><button data-cor-status="${esc(s)}" aria-label="Escolher cor de ${esc(s)}" style="width:22px;height:22px;min-width:22px;border-radius:50%;background:${corStatus(s)};border:2px solid var(--border);padding:0;cursor:pointer;"></button><b>${esc(s)}</b></span>
      ${corAberta===s?`<div style="display:flex;flex-wrap:wrap;gap:6px;">${COR_PALETA.map(p=>`<button data-cor-pick="${p}" data-cor-st="${esc(s)}" aria-label="Cor ${p}" style="width:26px;height:26px;min-width:26px;border-radius:50%;background:${p};border:${p===corStatus(s)?'2px solid var(--text)':'1px solid var(--border)'};padding:0;cursor:pointer;"></button>`).join('')}</div>`:''}
    </span>`).join('');
  document.getElementById('pillBanco').innerHTML = listas.banco.map(s=>
    `<span class="pill">${esc(s)} ${ehOwner()?`<button data-del-banco="${esc(s)}" aria-label="Remover ${esc(s)}">✕</button>`:''}</span>`).join('');

  document.querySelectorAll('[data-del-banco]').forEach(b=>b.onclick=()=>removerBanco(b.dataset.delBanco));
  document.querySelectorAll('[data-cor-status]').forEach(b=>b.onclick=()=>{
    corAberta = (corAberta===b.dataset.corStatus) ? null : b.dataset.corStatus;
    renderListas();
  });
  document.querySelectorAll('[data-cor-pick]').forEach(b=>b.onclick=()=>{
    statusCor[b.dataset.corSt] = b.dataset.corPick;
    corAberta = null;
    salvarListas(); renderListas(); renderHistorico();
  });
  renderAparencia();
  renderCombosLista();
}

function renderAparencia(){
  const NOMES_TEMA = {auto:'Automático', claro:'Claro', escuro:'Escuro'};
  document.getElementById('temaBtns').innerHTML = [['auto','🌗','segue o aparelho'],['claro','☀️','sempre claro'],['escuro','🌙','sempre escuro']].map(([k,ic,sub])=>
    `<button type="button" data-tema="${k}" aria-pressed="${tema===k}"><span>${ic} ${NOMES_TEMA[k]}</span><small>${sub}</small></button>`).join('');
  document.querySelectorAll('[data-tema]').forEach(b=>b.onclick=()=>{
    tema = b.dataset.tema; Tema.escolher(tema);
    salvarListas(); renderAparencia(); toast('Tema: '+NOMES_TEMA[tema]+'.');
  });
  const op = [['linha','Linha inteira'],['faixa','Faixa + selo'],['ambos','Os dois juntos'],['nenhum','Sem cor']];
  document.getElementById('estiloCorBtns').innerHTML = op.map(([k,t])=>
    `<span class="tag ${estiloCor===k?'on':''}" data-estilo-cor="${k}">${t}</span>`).join('');
  document.getElementById('prevCor').innerHTML = listas.status.slice(0,3).map(st=>
    `<div class="item" style="${estiloItemCor(st)}"><div class="item-top"><span class="item-desc">Exemplo de lançamento</span><span class="valor saida">-R$ 10,00</span></div><div class="item-meta">${seloStatus(st)} · Banco</div></div>`).join('');
  document.getElementById('layoutSaldosBtns').innerHTML = [['linhas','Uma linha por banco'],['tiles','Quadradinhos']].map(([k,t])=>
    `<span class="tag ${layoutSaldos===k?'on':''}" data-layout-saldos="${k}">${t}</span>`).join('');
  document.getElementById('ordemSaldosBtns').innerHTML = [['lista','Ordem da lista'],['valor','Maior saldo'],['manual','Manual']].map(([k,t])=>
    `<span class="tag ${ordemSaldos===k?'on':''}" data-ordem-saldos="${k}">${t}</span>`).join('');
  const manualEl = document.getElementById('ordemManualLista');
  if(ordemSaldos==='manual'){
    const ef = ordemManualEfetiva();
    manualEl.innerHTML = '<p class="muted" style="margin:8px 0 2px;">Use as setas para mover cada banco.</p>' + ef.map((n,i)=>
      `<div class="item" style="display:flex;align-items:center;gap:8px;"><span style="flex:1;font-size:14px;">${esc(n)}</span>` +
      `<button class="ghost" data-ord-up="${i}" aria-label="Subir ${esc(n)}" style="padding:4px 12px;" ${i===0?'disabled':''}>▲</button>` +
      `<button class="ghost" data-ord-dn="${i}" aria-label="Descer ${esc(n)}" style="padding:4px 12px;" ${i===ef.length-1?'disabled':''}>▼</button></div>`).join('');
  } else manualEl.innerHTML = '';
  document.querySelectorAll('[data-ordem-saldos]').forEach(b=>b.onclick=()=>{
    const novo = b.dataset.ordemSaldos;
    if(novo==='manual' && !ordemManual.length) ordemManual = indicesOrdenados().map(i=>listas.banco[i]);
    ordemSaldos = novo;
    salvarListas(); renderAparencia(); renderSaldos();
  });
  const mover = (i,d)=>{
    const ef = ordemManualEfetiva();
    const j = i+d;
    if(j<0 || j>=ef.length) return;
    [ef[i], ef[j]] = [ef[j], ef[i]];
    ordemManual = ef;
    agendarSalvarOrdem(); renderAparencia(); renderSaldos();
  };
  document.querySelectorAll('[data-ord-up]').forEach(b=>b.onclick=()=>mover(parseInt(b.dataset.ordUp,10),-1));
  document.querySelectorAll('[data-ord-dn]').forEach(b=>b.onclick=()=>mover(parseInt(b.dataset.ordDn,10),1));
  document.querySelectorAll('[data-layout-saldos]').forEach(b=>b.onclick=()=>{
    layoutSaldos = b.dataset.layoutSaldos;
    saldoAberto = -1;
    salvarListas(); renderAparencia(); renderSaldos();
  });
  document.querySelectorAll('[data-estilo-cor]').forEach(b=>b.onclick=()=>{
    estiloCor = b.dataset.estiloCor;
    salvarListas(); renderAparencia(); renderHistorico();
  });
}

async function salvarListas(){
  if(!db) return;
  await db.doc("config/listas").set({banco: listas.banco, statusCor, tema, estiloCor, layoutSaldos, ordemSaldos, ordemManual});
}
async function removerBanco(banco){
  if(!db) return;
  let snap;
  try{ snap = await db.collection("lancamentos").where("banco","==",banco).limit(1000).get(); }
  catch(err){ console.error(err); toast('Não consegui conferir os lançamentos desse banco. Tente de novo.', true); return; }
  const qtd = snap.docs.length;
  if(qtd>0){
    await avisar(`Não dá para remover "${banco}": ${qtd>=1000?'pelo menos 1000':qtd} lançamento(s) usam esse banco. Para tirá-lo da lista, mova ou exclua esses lançamentos antes.`);
    return;
  }
  if(!(await confirmarModal(`Nenhum lançamento usa o banco "${banco}". Remover da lista?`))) return;
  listas.banco = listas.banco.filter(x=>x!==banco);
  await salvarListas(); renderListas(); renderHistorico();
  toast(`✓ Banco "${banco}" removido`);
}

function normTxt(s){ return String(s||'').normalize('NFD').replace(/[\u0300-\u036f]/g,'').toLowerCase(); }
function chaveTags(a){ return [...a].sort().join('|'); }
function contagemConjuntos(){
  const m = new Map();
  lancamentosCache.forEach(l=>{
    const t = l.tags || [];
    if(t.length < 2) return;
    const k = chaveTags(t);
    const e = m.get(k);
    if(e) e.n++; else m.set(k, {tags:[...t], n:1});
  });
  return m;
}
function combosAtuaisLista(){
  const conj = contagemConjuntos();
  const salvos = combosSalvos.map(c=>({tags:c.tags, n:(conj.get(chaveTags(c.tags))||{}).n||0, salvo:true}));
  const ks = new Set(salvos.map(c=>chaveTags(c.tags)));
  const sug = [...conj.values()].filter(x=>x.n>=3 && !ks.has(chaveTags(x.tags)))
    .sort((a,b)=>b.n-a.n).slice(0,5).map(x=>({tags:x.tags, n:x.n, salvo:false}));
  return [...salvos, ...sug];
}
function salvarCombos(){
  if(!db) return Promise.resolve();
  const foto = clonar(combosSalvos);
  filaCombos = filaCombos.then(()=>db.doc("config/combos").set({lista:foto})).catch(err=>{
    console.error('Falha ao salvar combos', err);
    avisar('Não consegui salvar os combos agora (' + ((err && err.code) || 'erro') + '). Tente de novo.');
  });
  return filaCombos;
}
function chipTag(t){
  const on = tagsSelecionadas.has(t);
  return `<span class="tag ${on?'on':''}" role="checkbox" aria-checked="${on}" tabindex="0" data-tag="${esc(t)}">${on?'✓ ':''}${esc(t)}</span>`;
}
function renderTagPicker(){
  if(!document.getElementById('tagArea')) return;
  combosVisiveis = combosAtuaisLista();
  const sel = [...tagsSelecionadas];
  const kSel = chaveTags(sel);
  const temSugerido = combosVisiveis.some(c=>!c.salvo);
  let h = '';
  if(combosVisiveis.length){
    h += `<div class="muted">Combos${temSugerido?' (tracejado = sugerido pelo histórico)':''}</div><div class="tags">` +
      combosVisiveis.map((c,i)=>{
        const on = sel.length===c.tags.length && chaveTags(c.tags)===kSel;
        return `<span class="tag ${on?'on':''}" data-combo="${i}" style="border-radius:8px;${c.salvo?'':'border-style:dashed;'}">${esc(c.tags.join(' · '))}${c.n?` <span style="font-size:11px;">${c.n}x</span>`:''}</span>`;
      }).join('') + '</div>';
  }
  const jaSalvo = combosSalvos.some(c=>chaveTags(c.tags)===kSel);
  if(sel.length>=2 && !jaSalvo && ehOwner()) h += '<button class="ghost" id="btnSalvarCombo" style="margin-top:8px;">Salvar seleção como combo</button>';
  document.getElementById('tagCombos').innerHTML = h;
  const q = normTxt(tagBusca.trim());
  const lista = [...sel.filter(t=>!tagsUsadas.includes(t)), ...tagsUsadas].filter(t=>!q || normTxt(t).includes(q));
  const l = lista.length ? lista.map(chipTag).join('') : '<span class="muted">Nenhuma tag encontrada. Digite abaixo e Enter para criar.</span>';
  document.getElementById('tagsCnt').textContent = sel.length + (sel.length===1 ? ' tag marcada' : ' tags marcadas');
  document.getElementById('tagPicker').innerHTML = l;
  document.getElementById('tagsResumo').innerHTML = sel.length ? sel.map(t=>`<span class="chip" data-rm-tag="${esc(t)}">${esc(t)}</span>`).join('') : '<span class="tags-vazio">Nenhuma</span>';
  if(seletorHist) seletorHist.render();
}
function renderCombosLista(){
  const el = document.getElementById('listaCombos');
  if(!el) return;
  el.innerHTML = combosSalvos.map((c,i)=>
    `<div class="item" style="display:flex;align-items:center;gap:8px;"><span style="flex:1;font-size:13px;">${esc(c.tags.join(' · '))}</span><button class="ghost" data-del-combo="${i}">Excluir</button></div>`).join('')
    || '<p class="muted">Nenhum combo salvo ainda.</p>';
  el.querySelectorAll('[data-del-combo]').forEach(b=>b.onclick=()=>{
    combosSalvos.splice(parseInt(b.dataset.delCombo,10),1);
    salvarCombos(); renderCombosLista(); renderTagPicker();
  });
}
// Seletor de tags só para filtrar o Histórico: mesmos combos e chips do Lançar, sem criar nada. (Tendências tem o seu em js/tendencias.js.)
function criarSeletorFiltroTags(raiz, aoMudar){
  const S = {sel:new Set(), busca:'', abertas:false, modo:'todas', combos:[]};
  raiz.innerHTML = '<div data-p="combos"></div><div class="tags" data-p="marcadas"></div>' +
    '<input type="text" data-p="busca" placeholder="Buscar tag..." style="margin-top:8px;">' +
    '<div class="muted tags-rot" style="margin-top:8px;">Tags</div><div class="tags" data-p="lista"></div><div class="tags" data-p="modo"></div>';
  const q = k=>raiz.querySelector('[data-p="'+k+'"]');
  const chip = t=>`<span class="tag ${S.sel.has(t)?'on':''}" data-tag="${esc(t)}">${esc(t)}</span>`;
  function render(){
    [...S.sel].forEach(t=>{ if(!tagsUsadas.includes(t)) S.sel.delete(t); }); // tag excluída/migrada
    S.combos = combosAtuaisLista();
    const sel = [...S.sel], kSel = chaveTags(sel);
    const sug = S.combos.some(c=>!c.salvo);
    q('combos').innerHTML = S.combos.length
      ? `<div class="muted">Combos${sug?' (tracejado = sugerido pelo histórico)':''}</div><div class="tags">` +
        S.combos.map((c,i)=>{
          const on = sel.length===c.tags.length && chaveTags(c.tags)===kSel;
          return `<span class="tag ${on?'on':''}" data-combo="${i}" style="border-radius:8px;${c.salvo?'':'border-style:dashed;'}">${esc(c.tags.join(' · '))}${c.n?` <span style="font-size:11px;">${c.n}x</span>`:''}</span>`;
        }).join('') + '</div>' : '';
    q('marcadas').innerHTML = sel.length
      ? '<span class="muted" style="align-self:center;">Marcadas:</span>' + sel.map(chip).join('') +
        '<span class="tag" data-limpar="1">Limpar</span>' : '';
    const b = normTxt(S.busca.trim());
    q('lista').innerHTML = b
      ? (tagsUsadas.filter(t=>normTxt(t).includes(b)).map(chip).join('') || '<span class="muted">Nenhuma tag encontrada.</span>')
      : S.abertas
        ? '<span data-todas="1" style="cursor:pointer;color:var(--accent);font-size:13px;width:100%;">Recolher ▴</span>' + tagsUsadas.map(chip).join('')
        : `<span data-todas="1" style="cursor:pointer;color:var(--accent);font-size:13px;">Todas as tags (${tagsUsadas.length}) ▾</span>`;
    q('modo').innerHTML = sel.length>=2
      ? '<span class="muted" style="align-self:center;">Mostrar lançamentos que têm:</span>' +
        `<span class="tag ${S.modo==='todas'?'on':''}" data-modo="todas">todas as tags</span>` +
        `<span class="tag ${S.modo==='qualquer'?'on':''}" data-modo="qualquer">qualquer uma</span>` : '';
  }
  const descricaoSel = ()=>[...S.sel].join(S.modo==='todas' ? ' + ' : ' ou ');
  function mudou(){ render(); aoMudar(); }
  q('busca').addEventListener('input', e=>{ S.busca = e.target.value; render(); });
  raiz.addEventListener('click', e=>{
    const el = e.target.closest('[data-tag],[data-combo],[data-todas],[data-modo],[data-limpar]');
    if(!el) return;
    if(el.dataset.tag!==undefined){ const t = el.dataset.tag; S.sel.has(t) ? S.sel.delete(t) : S.sel.add(t); return mudou(); }
    if(el.dataset.combo!==undefined){
      const c = S.combos[parseInt(el.dataset.combo,10)];
      if(!c) return;
      S.sel = chaveTags([...S.sel])===chaveTags(c.tags) ? new Set() : new Set(c.tags); // tocar de novo no combo desmarca
      return mudou();
    }
    if(el.dataset.modo){ S.modo = el.dataset.modo; return mudou(); }
    if(el.dataset.limpar){ S.sel = new Set(); return mudou(); }
    if(el.dataset.todas){ S.abertas = !S.abertas; render(); }
  });
  render();
  return {
    render,
    // sem seleção, tudo passa; com seleção, vale "todas" (padrão) ou "qualquer uma"
    casa(l){
      if(!S.sel.size) return true;
      const tg = l.tags||[], sel = [...S.sel];
      return S.modo==='todas' ? sel.every(t=>tg.includes(t)) : sel.some(t=>tg.includes(t));
    },
    descricao(){ return [...S.sel].join(S.modo==='todas' ? ' + ' : ' ou '); },
    n(){ return S.sel.size; },
    limpar(){ S.sel = new Set(); render(); },
    resumo(){
      const n = S.sel.size;
      if(!n) return 'todas';
      return n<=2 ? descricaoSel() : n+' tags';
    }
  };
}
function renderTagFilter(){
  if(!seletorHist) seletorHist = criarSeletorFiltroTags(document.getElementById('histTags'), renderHistorico);
  else seletorHist.render();
}
function renderTagChartSelect(){ desenharGrafico(); }
function renderGerenciarTags(){
  const el = document.getElementById('pillTags');
  if(!el) return;
  el.innerHTML = tagsUsadas.map(t=>
    `<span class="pill" style="flex-direction:row;min-width:0;">#${esc(t)} <button data-del-tag="${esc(t)}">✕</button></span>`).join('')
    || '<p class="muted">Nenhuma tag ainda.</p>';
  el.querySelectorAll('[data-del-tag]').forEach(b=>b.onclick=()=>excluirTag(b.dataset.delTag));
}

async function garantirTags(novasTags){
  const set = new Set(tagsUsadas);
  const novas = [];
  novasTags.forEach(t=>{ if(t && !set.has(t)){ set.add(t); novas.push(t); } });
  if(novas.length && db){
    tagsUsadas = [...set];
    // o servidor só acrescenta o que falta (vale para o membro); reescrever a lista é do dono
    if(db.adicionarTags) await db.adicionarTags(novas);
    else await db.doc("config/tags").set({usadas:tagsUsadas});
  }
}

// Aba Lançar: o <select id="fTipo"> continua sendo a fonte do valor; os botões só o espelham.
function definirTipo(v){
  document.getElementById('fTipo').value = v;
  document.querySelectorAll('#tipoLancar [data-tipo]').forEach(b=>b.setAttribute('aria-checked', b.dataset.tipo===v ? 'true' : 'false'));
}
function lancarModo(m){
  document.querySelectorAll('#modoLancar [data-modo]').forEach(b=>b.setAttribute('aria-pressed', b.dataset.modo===m ? 'true' : 'false'));
  document.getElementById('painelLancar').classList.toggle('hidden', m!=='lanc');
  document.getElementById('painelTransf').classList.toggle('hidden', m!=='transf');
}
function lancarMais(abrir){
  document.getElementById('maisDet').classList.toggle('aberto', abrir);
  document.getElementById('maisAbrir').setAttribute('aria-expanded', String(abrir));
}
function lancarTags(abrir){
  document.getElementById('tagsCard').classList.toggle('aberto', abrir);
  document.getElementById('tagsVeu').classList.toggle('aberto', abrir);
}
function ligarLancar(){
  document.getElementById('modoLancar').addEventListener('click', e=>{ const b = e.target.closest('[data-modo]'); if(b) lancarModo(b.dataset.modo); });
  document.getElementById('tipoLancar').addEventListener('click', e=>{ const b = e.target.closest('[data-tipo]'); if(b) definirTipo(b.dataset.tipo); });
  document.getElementById('iaAbrir').addEventListener('click', ()=>{
    const aberto = document.getElementById('iaCard').classList.toggle('aberto');
    document.getElementById('iaAbrir').setAttribute('aria-expanded', String(aberto));
    if(aberto) document.getElementById('notaLivre').focus();
  });
  document.getElementById('maisAbrir').addEventListener('click', ()=>lancarMais(!document.getElementById('maisDet').classList.contains('aberto')));
  document.getElementById('tagsAbrir').addEventListener('click', e=>{
    const x = e.target.closest('[data-rm-tag]');
    if(x && matchMedia('(min-width:900px)').matches){ tagsSelecionadas.delete(x.dataset.rmTag); renderTagPicker(); return; } // computador: o × tira a tag
    if(!matchMedia('(min-width:900px)').matches) lancarTags(true);
  });
  document.getElementById('tagsPronto').addEventListener('click', ()=>lancarTags(false));
  document.getElementById('tagsVeu').addEventListener('click', ()=>lancarTags(false));
  document.getElementById('tagArea').addEventListener('keydown', e=>{
    if((e.key==='Enter' || e.key===' ') && e.target.dataset && e.target.dataset.tag!==undefined){ e.preventDefault(); e.target.click(); }
  });
  document.addEventListener('keydown', e=>{ if(e.key==='Escape' && document.getElementById('tagsCard').classList.contains('aberto')) lancarTags(false); });
  definirTipo(document.getElementById('fTipo').value);
}

function setupHandlers(){
  ligarLancar();
  document.getElementById('fAvisoMount').innerHTML = htmlAviso('f');
  document.getElementById('fRepetirMount').innerHTML = htmlRepeticao('fr', {qtd:1, rotuloQtd:'Total de ocorrências (com este)'});
  ligarAviso('f');
  ligarRepeticao('fr', atualizarDicaRepeticao);
  document.getElementById('fData').addEventListener('change', atualizarDicaRepeticao);
  document.getElementById('fData').addEventListener('change', atualizarCampoFatura);
  document.getElementById('fStatus').addEventListener('change', ()=>{
    document.getElementById('fFatura').value = String(ajustePadrao(document.getElementById('fStatus').value));
    atualizarCampoFatura();
  });
  document.getElementById('fBanco').addEventListener('change', atualizarCampoFatura);
  document.getElementById('fPagaCartao').addEventListener('change', ()=>{
    const mes = document.getElementById('fPagaMes');
    mes.classList.toggle('hidden', !document.getElementById('fPagaCartao').value);
    if(!mes.value) mes.value = document.getElementById('fData').value.slice(0,7);
  });
  document.getElementById('listaCartoes').addEventListener('click', onCartaoClick);
  document.getElementById('btnAtualizarCartao').addEventListener('click', ()=>carregarCartoes(true));
  document.getElementById('btnAddCartao').addEventListener('click', adicionarCartaoUI);
  document.getElementById('tagBusca').addEventListener('input', e=>{ tagBusca = e.target.value; renderTagPicker(); });
  document.getElementById('tagArea').addEventListener('click', e=>{
    const el = e.target.closest('[data-tag],[data-combo],[data-todas],#btnSalvarCombo');
    if(!el) return;
    if(el.dataset.tag!==undefined){
      const t = el.dataset.tag;
      tagsSelecionadas.has(t) ? tagsSelecionadas.delete(t) : tagsSelecionadas.add(t);
      renderTagPicker(); return;
    }
    if(el.dataset.combo!==undefined){
      const c = combosVisiveis[parseInt(el.dataset.combo,10)];
      if(c){ tagsSelecionadas = new Set(c.tags); renderTagPicker(); }
      return;
    }
    if(el.dataset.todas){ tagsTodasAbertas = !tagsTodasAbertas; renderTagPicker(); return; }
    if(el.id==='btnSalvarCombo'){
      combosSalvos.push({tags:[...tagsSelecionadas]});
      salvarCombos(); renderTagPicker(); renderCombosLista();
    }
  });
  document.getElementById('novaTagInput').addEventListener('keydown', e=>{
    if(e.key==='Enter'){
      e.preventDefault();
      const v = e.target.value.trim().replace(/^#/,'');
      if(v){ tagsSelecionadas.add(v); garantirTags([v]); renderTagPicker(); e.target.value=''; }
    }
  });

  document.getElementById('btnAddBanco').onclick = ()=>{
    const v = document.getElementById('novoBanco').value.trim();
    if(v && !listas.banco.includes(v)){ listas.banco.push(v); salvarListas(); renderListas(); document.getElementById('novoBanco').value=''; }
  };

  document.getElementById('btnIA').onclick = preencherComIA;
  document.getElementById('btnSalvar').onclick = salvarLancamento;
  document.getElementById('btnCancelarEdicao').onclick = ()=>{ if(formularioSujo()) perguntarDescartarEdicao(); else cancelarEdicao(); };
  document.getElementById('btnLimpar').onclick = limparFormulario;
  document.getElementById('btnTransferir').onclick = registrarTransferencia;
  document.getElementById('tData').value = nowLocalInput();
}

async function preencherComIA(){
  const nota = document.getElementById('notaLivre').value.trim();
  const statusEl = document.getElementById('iaStatus');
  if(!nota){ statusEl.textContent = "Escreva algo na nota primeiro."; return; }
  if(!sample){ statusEl.textContent = "IA indisponível nesta visualização — preencha manualmente abaixo."; return; }
  statusEl.textContent = "Pensando...";
  try{
    const hoje = nowLocalInput().replace('T',' ');
    const prompt = `Extraia dados estruturados de um lançamento financeiro pessoal em português (Brasil).
Texto do usuário: "${nota}"
Data/hora local do usuário agora (já é hora local, sem fuso, não é UTC): ${hoje}
Status possíveis: ${listas.status.join(", ")}
Bancos conhecidos: ${listas.banco.join(", ") || "(nenhum ainda)"}
Tags já usadas antes (reaproveite se fizer sentido, senão crie novas simples em minúsculas): ${tagsUsadas.join(", ") || "(nenhuma ainda)"}
Conjuntos de tags frequentes (se o lançamento combinar com um deles, use o conjunto inteiro): ${combosAtuaisLista().map(c=>'['+c.tags.join(', ')+']').join(' ') || "(nenhum ainda)"}
Responda SOMENTE em JSON, sem markdown, no formato:
{"status":"...", "tipo":"entrada"|"saida", "valor":number, "descricao":"...", "banco":"...", "tags":["..."], "dataEvento":"AAAA-MM-DDTHH:MM:SS (hora local do usuário, SEM Z, SEM fuso)"}
Se não souber o banco com certeza, deixe banco como "".`;
    const res = await sample.json(prompt, {modelTier:"quick"});
    if(res.status){ document.getElementById('fStatus').value = res.status; document.getElementById('fFatura').value = String(ajustePadrao(res.status)); atualizarCampoFatura(); }
    if(res.tipo) definirTipo(res.tipo);
    if(res.valor) document.getElementById('fValor').value = res.valor;
    if(res.descricao) document.getElementById('fDesc').value = res.descricao;
    if(res.banco){
      if(!listas.banco.includes(res.banco)){ listas.banco.push(res.banco); salvarListas(); renderListas(); }
      document.getElementById('fBanco').value = res.banco;
    }
    if(res.dataEvento){
      // já é hora local do usuário (sem Z) — só normaliza pro formato do campo
      const s = String(res.dataEvento).replace('Z','').slice(0,16);
      document.getElementById('fData').value = s;
    }
    document.getElementById('fNota').value = nota;
    if(Array.isArray(res.tags)){
      tagsSelecionadas = new Set(res.tags);
      garantirTags(res.tags).then(renderTagPicker);
      renderTagPicker();
    }
    statusEl.textContent = "Confira os campos abaixo e salve.";
  }catch(err){
    statusEl.textContent = "Não consegui interpretar agora (" + (err.code||'erro') + "). Preencha manualmente.";
  }
}

function salvarLancamento(){
  const btn = document.getElementById('btnSalvar');
  return comBotaoOcupado(btn, 'Salvando…', async ()=>{
    try{ await salvarLancamentoInterno(); }
    catch(err){ console.error(err); toast(msgErro(err, 'Não foi possível salvar. Confira a conexão e tente de novo.'), true); }
  });
}
async function salvarLancamentoInterno(){
  if(!db){ await avisar("Banco de dados indisponível nesta visualização."); return; }
  const valor = parseFloat(document.getElementById('fValor').value);
  // zero (lançamento só de nota) e negativo (estorno/reembolso) são aceitos de propósito
  if(!Number.isFinite(valor)){ await avisar("Informe um valor válido."); return; }
  const tags = [...tagsSelecionadas];
  await garantirTags(tags);
  const statusEscolhido = document.getElementById('fStatus').value;
  const doc = {
    status: statusEscolhido,
    tipo: document.getElementById('fTipo').value,
    valor: valor,
    descricao: document.getElementById('fDesc').value || "(sem descrição)",
    nota: document.getElementById('fNota').value || "",
    dataEvento: new Date(document.getElementById('fData').value).toISOString(),
    banco: document.getElementById('fBanco').value,
    tags: tags
  };
  if(ehMeioCartao(doc.banco)){
    doc.faturaAjuste = parseInt(document.getElementById('fFatura').value,10) || 0;
    doc.valorEstimado = document.getElementById('fEstimado').checked;
  }
  const marca = marcaDoFormulario();
  if(marca) doc.pagaFatura = marca; else if(editandoId) doc.pagaFatura = null; // na edicao, sem escolha tira a marca
  const aviso = lerAvisoDoFormulario();
  if(aviso === 'incompleto'){ await avisar("Marque pelo menos um dia de aviso ou \"Insistir todo dia\"."); return; }
  if(aviso) doc.aviso = aviso;
  else if(editandoId && dataFuturaForm()) doc.aviso = null; // desligou o aviso na edicao
  // repeticao (so evento futuro novo): este + as copias, ligados por uma serie
  let copias = [];
  if(!editandoId && dataFuturaForm()){
    const rep = lerRepeticao('fr');
    if(rep.quantidade > 1){
      const erroRep = Recorrencia.validar(rep.intervaloMeses, rep.quantidade - 1);
      if(erroRep){ await avisar(erroRep); return; }
      const serieId = novoSerieId();
      doc.serie = {id: serieId, indice: 1, intervalo: rep.intervaloMeses};
      copias = Recorrencia.montarCopias(doc, {intervaloMeses:rep.intervaloMeses, quantidade:rep.quantidade-1, status:statusEscolhido,
        aviso: doc.aviso || null, numerar:false, serieId, indiceInicial:2});
      await lerDadosParaDuplicados();
      const existentes = chavesExistentes();
      const iguais = copias.filter(c=>existentes.has(Recorrencia.chaveDuplicado(c))).length;
      if(iguais && !(await confirmarModal(`${iguais} das repetições já existem (mesmo status, descrição, valor, banco e dia). Criar mesmo assim?`))) return;
    }
  }
  const jaExiste = await jaExisteIgual(doc, editandoId);
  if(jaExiste && !(await confirmarModal("Já existe um lançamento igual (mesmo status, descrição, valor, banco e data). Salvar mesmo assim?"))) return;
  if(editandoId){
    const idEditado = editandoId;
    const antes = JSON.parse(JSON.stringify(acharLancamento(idEditado) || {})); // copia: aplicarLocal altera o objeto da lista
    if(!(await confirmarModal(`Salvar a alteração? ${resumoEdicao(antes, doc)}. Depois de salvar, dá para desfazer por 6 segundos.`))) return;
    await db.collection("lancamentos").doc(idEditado).update(doc);
    aplicarLocal(idEditado, doc);
    document.getElementById('iaStatus').textContent = "Edição salva ✓";
    cancelarEdicao();
    mostrarAvisoAcao({
      txt: 'Edição salva', sub: cortar(doc.descricao, 40) + ' · ' + fmtMoeda(doc.valor), rotulo: 'Desfazer',
      onAcao: ()=>desfazerEdicao(idEditado, antes, doc)
    });
  }else{
    await db.collection("lancamentos").add({...doc, criadoEm: new Date().toISOString()});
    let extra = '';
    if(copias.length){
      try{ const n = await criarEventos(copias); extra = ` + ${n} repetição${n===1?'':'ões'}`; }
      catch(err){
        console.error(err);
        await avisar(`O lançamento foi salvo, mas as repetições pararam depois de ${err.criados||0} de ${copias.length}: ${msgErro(err, 'erro ao salvar')} Use Prever no lançamento para completar a série.`);
      }
    }
    document.getElementById('iaStatus').textContent = "Lançamento salvo ✓" + extra;
    toast('✓ Lançamento salvo: ' + doc.descricao + ' · ' + fmtMoeda(doc.valor) + extra);
    limparFormulario();
  }
}

// A marca "Paga a fatura" (itens 33b e 35): uma label opcional, o cartao e o mes da fatura que este lancamento pagou. So ajuda a achar depois.
function renderMarcaFatura(){
  const sel = document.getElementById('fPagaCartao');
  if(!sel) return;
  const atual = sel.value;
  const cartoes = meiosLista.filter(m=>m.cartao).map(m=>m.nome);
  sel.innerHTML = '<option value="">Nenhuma</option>' + cartoes.map(n=>`<option value="${esc(n)}">${esc(n)}</option>`).join('');
  if(cartoes.includes(atual)) sel.value = atual;
  document.getElementById('fPagaWrap').classList.toggle('hidden', !cartoes.length);
  document.getElementById('fPagaMes').classList.toggle('hidden', !sel.value);
}
function marcaDoFormulario(){
  const meio = document.getElementById('fPagaCartao').value, ym = document.getElementById('fPagaMes').value;
  return meio && /^\d{4}-\d{2}$/.test(ym) ? {meio, ym} : null;
}
function definirMarcaDoFormulario(m){
  const sel = document.getElementById('fPagaCartao');
  if(m && ![...sel.options].some(o=>o.value===m.meio)){ const o = document.createElement('option'); o.value = o.textContent = m.meio; sel.appendChild(o); }
  sel.value = m ? m.meio : '';
  document.getElementById('fPagaMes').value = m ? m.ym : '';
  document.getElementById('fPagaMes').classList.toggle('hidden', !m);
}
// Um lancamento antigo pode ter um status que o servidor nao oferece mais (ex.: PREVISTO, CREDITO IN). Ao editar, ele aparece como uma opcao a mais e volta inalterado.
function definirStatusDoFormulario(status, soDaLista){
  const sel = document.getElementById('fStatus');
  [...sel.querySelectorAll('option[data-extra]')].forEach(o=>o.remove());
  if(status && ![...sel.options].some(o=>o.value===status)){
    if(soDaLista){ status = sel.options.length ? sel.options[0].value : ''; }
    else { const o = document.createElement('option'); o.value = status; o.textContent = status; o.dataset.extra = '1'; sel.appendChild(o); }
  }
  sel.value = status;
}
function limparFormulario(){
  definirMarcaDoFormulario(null);
  [...document.querySelectorAll('#fStatus option[data-extra]')].forEach(o=>o.remove());
  document.getElementById('notaLivre').value='';
  document.getElementById('fValor').value='';
  document.getElementById('fDesc').value='';
  document.getElementById('fNota').value='';
  document.getElementById('fData').value = nowLocalInput();
  tagsSelecionadas = new Set();
  tagBusca = ''; document.getElementById('tagBusca').value = '';
  document.getElementById('fFatura').value = String(ajustePadrao(document.getElementById('fStatus').value));
  document.getElementById('fEstimado').checked = false;
  preencherAvisoNoFormulario(null);
  if(document.getElementById('frQtd')){ document.getElementById('frQtd').value = '1'; definirIntervalo('fr', 1); }
  atualizarCampoFatura();
  renderTagPicker();
}

function registrarTransferencia(){
  return comBotaoOcupado(document.getElementById('btnTransferir'), 'Registrando…', async ()=>{
    try{ await registrarTransferenciaInterno(); }
    catch(err){ console.error(err); toast(msgErro(err, 'Não foi possível registrar a transferência. Tente de novo.'), true); }
  });
}
async function registrarTransferenciaInterno(){
  if(!db){ await avisar("Banco de dados indisponível nesta visualização."); return; }
  const origem = document.getElementById('tBancoOrigem').value;
  const destino = document.getElementById('tBancoDestino').value;
  const valor = parseFloat(document.getElementById('tValor').value);
  if(!origem || !destino || origem===destino || !valor || valor<=0){ await avisar("Confira os bancos e o valor."); return; }
  // data e hora escolhidas pela pessoa (datetime-local); vazio vale agora. O servidor cria as duas pontas, numa transacao so.
  const quando = new Date(document.getElementById('tData').value || nowLocalInput()).toISOString();
  await db.transferir(origem, destino, valor, quando);
  document.getElementById('tValor').value='';
  document.getElementById('tData').value = nowLocalInput();
  toast(`✓ Transferência registrada: ${origem} → ${destino} · ${fmtMoeda(valor)}`);
}

const MESES_NOME = ['janeiro','fevereiro','março','abril','maio','junho','julho','agosto','setembro','outubro','novembro','dezembro'];
function rotuloMes(ym){
  const [y,m] = ym.split('-').map(Number);
  const nome = MESES_NOME[m-1];
  return nome.charAt(0).toUpperCase() + nome.slice(1) + ' de ' + y;
}
function chaveDia(d){ return d.getFullYear()+'-'+pad2(d.getMonth()+1)+'-'+pad2(d.getDate()); } // AAAA-MM-DD no fuso do navegador
function diasAte(k){ // dias de hoje até o dia k (AAAA-MM-DD): negativo se já passou
  const h = new Date(); const a = new Date(h.getFullYear(), h.getMonth(), h.getDate());
  const [y,m,d] = k.split('-').map(Number);
  return Math.round((new Date(y,m-1,d) - a)/86400000);
}
function diaLocal(l){ return chaveDia(new Date(l.dataEvento)); }
// Eventos futuros ordenados por data. Ocultos só entram se pedido.
// ---------- Futuros: o servidor manda tudo pronto (GET /futuros): previstos, resumo, Atrasados e meses ----------
let futurosDados = null, futurosCarregando = false, futurosRepetir = false;
let futurosTimer;
let futurosSujo = true; // algo mudou desde a ultima leitura (ou nunca leu): quem precisa dos previstos le de novo
// Futuros so e lido por quem precisa: a aba Futuros e os gestos que olham a serie ou os duplicados (Prever, Estender, repetir ao lancar).
async function garantirFuturos(){
  if(!db || typeof db.futuros !== 'function') return;
  if(futurosDados && !futurosSujo && !futurosCarregando) return;
  await carregarFuturos();
  while(futurosCarregando) await new Promise(r=>setTimeout(r, 50)); // uma leitura que ja estava em curso
}
// Atrasados por ano (item 45): os anos antigos buscados a pedido e quais blocos (ano ou mes) a pessoa abriu ou fechou. O servidor agrupa e conta; aqui so se guarda a escolha.
let futurosAnosExtra = [], futurosBlocos = {}, futurosAnoSel = '', futurosAnoBuscando = false;
// varias leituras de lancamentos seguidas (a abertura do app, uma gravacao) viram uma leitura so de Futuros
function agendarFuturos(){
  clearTimeout(futurosTimer);
  const soAnos = futurosSoAnos; futurosSoAnos = null; // vale so para a leitura agendada por esta gravacao; outra mudanca no meio cancela e le tudo
  futurosTimer = setTimeout(()=>carregarFuturos(soAnos), 400);
}
// Excluir (ou desfazer) um previsto de um ano antigo buscado a pedido: ele nao esta em GET /futuros, entao so aquele ano e lido de novo.
// anoDoItemAntigo guarda o ano de cada item excluido, para o Desfazer saber onde ele estava.
let futurosSoAnos = null;
const anoDoItemAntigo = new Map();
function anoAntigoDoItem(id){
  const g = futurosAnosExtra.find(g=>g.meses.some(m=>m.itens.some(i=>i.id===id)));
  return g ? String(g.ano) : '';
}
function lerSoOAnoAntigo(ano){ if(ano) (futurosSoAnos = futurosSoAnos || new Set()).add(ano); }
async function carregarFuturos(soAnos){
  if(!db || typeof db.futuros !== 'function') return;
  if(futurosCarregando){ futurosRepetir = true; return; } // alguem gravou durante a leitura: le de novo no fim
  const soOsAnos = !!(soAnos && soAnos.size && futurosDados && !futurosSujo);
  futurosCarregando = true;
  futurosSujo = false;
  try{
    if(soOsAnos){
      futurosAnosExtra = await Promise.all(futurosAnosExtra.map(g=>soAnos.has(String(g.ano)) ? db.futurosAno(g.ano) : g));
    }else{
      futurosDados = await db.futuros();
      // os anos antigos ja buscados sao lidos de novo, para nao mostrar o que acabou de ser confirmado ou excluido
      if(futurosAnosExtra.length) futurosAnosExtra = await Promise.all(futurosAnosExtra.map(g=>db.futurosAno(g.ano)));
    }
  }
  catch(e){ console.error('Falha ao ler os Futuros', e); futurosSujo = true; }
  finally{
    futurosCarregando = false;
    renderFuturos();
    if(futurosRepetir){ futurosRepetir = false; carregarFuturos(); }
  }
}
// todos os previstos que o servidor mandou (sem os que acabaram de ser excluidos aqui)
function futurosLista(){
  if(!futurosDados) return [];
  const atrasados = [...futurosDados.atrasados, ...futurosAnosExtra].flatMap(a=>a.meses.flatMap(m=>m.itens));
  return [...atrasados, ...futurosDados.meses.flatMap(m=>m.itens)].filter(l=>l.tipo!=='fatura' && !exclusaoPendente.has(l.id));
}
// os grupos para mostrar: Atrasados (anos e meses recolhiveis) e um por mes. O servidor ja agrupou e contou; aqui so se esconde o oculto (a chave "Mostrar ocultos") e o que acabou de ser excluido.
function futurosGrupos(){
  const mostrar = document.getElementById('futMostrarOcultos').checked;
  const ok = l=> (mostrar || !l.oculto) && !exclusaoPendente.has(l.id);
  const grupos = [];
  const anos = [];
  // os anos buscados vem do mais novo para o mais velho, depois do ano da janela
  [...futurosDados.atrasados.map(a=>({a, busca:false})), ...futurosAnosExtra.slice().sort((x,y)=>y.ano-x.ano).map(a=>({a, busca:true}))].forEach(({a, busca})=>{
    const meses = a.meses.map(m=>({mes:m.mes, n:m.n, itens:m.itens.filter(ok)})).filter(m=>m.itens.length);
    if(meses.length) anos.push({ano:a.ano, n:a.n, valor:a.valor, busca, meses});
  });
  grupos.push({titulo:'Atrasados', atrasado:true, anos});
  futurosDados.meses.forEach(m=>{ const itens = m.itens.filter(ok); if(itens.length) grupos.push({titulo:rotuloMes(m.mes), atrasado:false, itens}); });
  return grupos;
}
// aberto por padrao: o ano da janela e todos os meses; o ano buscado chega recolhido
function futurosBlocoAberto(chave, padrao){ return chave in futurosBlocos ? futurosBlocos[chave] : padrao; }
function htmlBlocoAtrasados(a){
  const ka = 'a'+a.ano, aberto = futurosBlocoAberto(ka, !a.busca);
  let h = `<button type="button" class="fut-bloco" data-fut-bloco="${ka}" aria-expanded="${aberto}"><span>${esc(a.ano)} ${aberto?'▴':'▾'}</span><span class="fut-bloco-n">${a.n} · ${fmtMoeda(a.valor)}</span></button>`;
  if(!aberto) return h;
  a.meses.forEach(m=>{
    const km = 'm'+m.mes, ab = futurosBlocoAberto(km, true);
    h += `<button type="button" class="fut-bloco mes" data-fut-bloco="${km}" aria-expanded="${ab}"><span>${esc(MESES_NOME[+m.mes.slice(5)-1])} ${ab?'▴':'▾'}</span><span class="fut-bloco-n">${m.n}</span></button>`;
    if(ab) h += m.itens.map(l=>l.tipo==='fatura'?htmlFaturaFuturo(l):htmlEventoFuturo(l)).join('');
  });
  return h;
}
// a linha do titulo Atrasados leva o seletor dos anos anteriores (a lista de anos vem do servidor)
function htmlTituloAtrasados(){
  const anos = futurosDados.anosAnteriores || [];
  if(!anos.length) return '<div class="fut-titulo atras"><span>Atrasados</span></div>';
  const sel = futurosAnoSel || String(anos[0]);
  return `<div class="fut-titulo atras"><span>Atrasados</span><span class="fut-anos"><label for="futAno">Anos anteriores</label>`+
    `<select id="futAno">${anos.map(a=>`<option ${String(a)===sel?'selected':''}>${a}</option>`).join('')}</select>`+
    `<button type="button" id="futAnoVer" ${futurosAnoBuscando?'disabled':''}>${futurosAnoBuscando?'Buscando...':'Ver'}</button></span></div>`;
}
async function buscarAtrasadosDoAno(){
  const ano = (document.getElementById('futAno')||{}).value;
  if(!ano || futurosAnoBuscando) return;
  futurosAnoSel = ano;
  if(futurosAnosExtra.some(g=>String(g.ano)===ano)){ toast(ano+' já está na tela.'); return; }
  futurosAnoBuscando = true; renderFuturos();
  try{
    const g = await db.futurosAno(ano);
    if(g.n || g.meses.length){ futurosAnosExtra.push(g); toast(`${ano}: ${g.n} atrasado${g.n===1?'':'s'}. Toque no ano para abrir.`); }
    else toast('Nenhum atrasado em '+ano+'.');
  }catch(e){ console.error('Falha ao buscar os atrasados do ano', e); toast('Não foi possível buscar os atrasados de '+ano+'.', true); }
  finally{ futurosAnoBuscando = false; renderFuturos(); }
}
function chipAviso(l){
  const a = l && l.aviso;
  if(!a || typeof a !== 'object' || !l.previsto) return '';
  const dias = (Array.isArray(a.dias) ? a.dias : []).map(d=>d===0?'no dia':d+'d antes');
  if(a.insistir) dias.push('todo dia depois');
  return ` <span class="chip" title="Aviso no Telegram: ${esc(dias.join(', '))}">🔔</span>`;
}

function htmlItemLancamento(l, o){
  o = o || {};
  const fut = l.previsto === true;
  const dias = diasAte(diaLocal(l));
  const atrasado = !!o.prazo && fut && !l.oculto && dias < 0;
  let prazo = '';
  if(o.prazo && fut && !l.oculto){
    if(dias < 0) prazo = ` · <span class="chip atrasado">atrasado ${-dias} dia${-dias===1?'':'s'}</span>`;
    else if(dias === 0) prazo = ' · <span class="chip">hoje</span>';
    else if(dias <= 30) prazo = ` · <span class="muted">em ${dias} dia${dias===1?'':'s'}</span>`;
  }
  const seu = (o.prazo && !ehOwner() && l.criadoPor && l.criadoPor === meuSub) ? ' · <span class="muted">criado por você</span>' : '';
  const marca = (o.marcarFuturo && fut) ? '<div class="item-meta marca-futuro">📅 Evento futuro · aparece também em Futuros</div>' : '';
  return `
    <div class="item ${l.oculto?'oculto':''} ${fut?'futuro':''} ${atrasado?'atrasado':''} ${excluindo.has(l.id)?'excluindo':''}" style="${estiloItemCor(l.status)}">
      <div class="item-top"><span class="item-desc">${esc(l.descricao)}</span>${htmlValorLinha(l)}</div>
      <div class="item-meta">${new Date(l.dataEvento).toLocaleString('pt-BR')} · ${seloStatus(l.status)}${chipFatura(l)}${chipPagaFatura(l)}${chipEstimado(l)}${chipAviso(l)} · ${esc(l.banco||'-')} ${l.tags&&l.tags.length? '· '+l.tags.map(t=>'#'+esc(t)).join(' '):''}${prazo}${seu}</div>
      ${marca}
      ${l.nota?`<div class="item-meta">${escHtml(l.nota)}</div>`:''}
      <div class="item-acoes">
        ${podeMexer(l)?`<button data-acao="editar" data-id="${esc(l.id)}">✏️ Editar</button>`:''}
        <button data-acao="copiar" data-id="${esc(l.id)}">📋 Copiar</button>
        ${l.transferParId?'':`<button data-acao="prever" data-id="${esc(l.id)}">${l.serie?'➕ Estender série':'🔮 Prever'}</button>`}
        ${podeMexer(l)?`<button data-acao="ocultar" data-id="${esc(l.id)}">${l.oculto?'👁️ Mostrar':'🙈 Ocultar'}</button>`:''}
        ${ehOwner()?`<button data-acao="excluir" data-id="${esc(l.id)}">🗑️ Excluir</button>`:''}
      </div>
    </div>`;
}

function renderFuturos(){
  const el = document.getElementById('listaFuturos');
  if(!el) return;
  if(!futurosDados){ el.innerHTML = '<p class="muted">Carregando...</p>'; return; }
  const r = futurosDados.resumo;
  document.getElementById('futPagar').textContent = fmtMoeda(r.pagar);
  document.getElementById('futReceber').textContent = fmtMoeda(r.receber);
  document.getElementById('futAtrasados').textContent = r.atrasados.n ? `${r.atrasados.n} · ${fmtMoeda(r.atrasados.valor)}` : '0';
  document.getElementById('futMetricaAtras').classList.toggle('alerta', r.atrasados.n > 0);
  document.getElementById('futMetricaAtras').classList.toggle('sem-atraso', r.atrasados.n === 0);
  const grupos = futurosGrupos();
  // o grupo Atrasados sempre existe (leva o seletor dos anos anteriores); sem nada nele, ele some da lista, mas o seletor continua
  const vazio = grupos.every(g=>g.atrasado ? !g.anos.length : !g.itens.length);
  const semAtrasados = grupos[0].anos.length === 0;
  el.innerHTML = (futurosDados.truncado ? '<p class="muted">Mostrando os 1000 primeiros eventos.</p>' : '') +
    (vazio ? '<p class="muted">Nenhum evento futuro. Crie um lançamento na aba Lançar com uma data futura.</p>' : '') +
    '<div class="fut-tabela"><div class="fut-tabela-in">'+
    (vazio ? '' : '<div class="fut-cab"><span>Data</span><span>Evento</span><span>Banco</span><span>Status</span><span style="text-align:right">Valor</span><span></span></div>')+
    grupos.map(g=> g.atrasado
      ? htmlTituloAtrasados() + (semAtrasados ? '' : `<div class="fut-grupo">${g.anos.map(htmlBlocoAtrasados).join('')}</div>`)
      : `<div class="fut-titulo"><span>${esc(g.titulo)}</span></div><div class="fut-grupo">${g.itens.map(l=>l.tipo==='fatura'?htmlFaturaFuturo(l):htmlEventoFuturo(l)).join('')}</div>`).join('')+
    '</div></div>';
}

// ---------- Futuros: uma linha por evento (celular) e tabela (janela larga) ----------
function prazoFuturo(l){
  if(l.oculto) return {t:'', atras:false};
  const n = diasAte(diaLocal(l));
  if(n < 0) return {t:`atrasado ${-n} dia${n===-1?'':'s'}`, atras:true};
  if(n === 0) return {t:'hoje', atras:false};
  return {t:`em ${n} dia${n===1?'':'s'}`, atras:false};
}
function dataBR(l){ return new Date(l.dataEvento).toLocaleDateString('pt-BR'); }
// Acoes de um evento, na ordem e com as mesmas permissoes de antes. t = texto da janela; c = texto curto da tabela.
function acoesDoEvento(l){
  const a = [];
  // Confirmar vale so para previsto (o servidor manda previsto: true); em saida o texto e "Já paguei", em entrada "Já recebi"
  if(l.previsto && podeMexer(l)) a.push({k:'confirmar', t:l.tipo==='saida'?'Já paguei':'Já recebi', c:'Confirmar', pri:true});
  if(podeMexer(l)) a.push({k:'editar', t:'Editar', c:'Editar'});
  a.push({k:'copiar', t:'Copiar', c:'Copiar'});
  if(!l.transferParId) a.push({k:'prever', t:l.serie?'Estender série':'Prever próximas datas', c:l.serie?'Estender':'Prever'});
  if(podeMexer(l)) a.push({k:'ocultar', t:l.oculto?'Mostrar':'Ocultar', c:l.oculto?'Mostrar':'Ocultar'});
  if(ehOwner()) a.push({k:'excluir', t:'Excluir', c:'Excluir', perigo:true});
  return a;
}
const PONTOS_FUT = '<svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true"><circle cx="4" cy="10" r="1.7" fill="currentColor"/><circle cx="10" cy="10" r="1.7" fill="currentColor"/><circle cx="16" cy="10" r="1.7" fill="currentColor"/></svg>';
function htmlEventoFuturo(l){
  const p = prazoFuturo(l), id = esc(l.id);
  const serie = l.serie && l.serie.indice ? `série ${l.serie.indice}` : '';
  const prazo = `<span class="fut-prazo ${p.atras?'atras':''}">${p.t}</span>`;
  const botoes = acoesDoEvento(l).map(a=>`<button type="button" class="${a.perigo?'perigo':''}${a.pri?'pri':''}" data-acao="${a.k}" data-id="${id}">${a.c}</button>`).join('');
  const cor = estiloCor==='nenhum' ? 'var(--faint)' : corStatus(l.status);
  const sub = [serie, l.oculto?'oculto':''].filter(Boolean).join(' · ');
  return `<div class="fut-row ${l.oculto?'oculto':''} ${excluindo.has(l.id)?'excluindo':''}" style="--cor:${cor}">
    <button type="button" class="fut-m" data-fut-abrir="${id}" aria-haspopup="dialog">
      <span class="l"><span class="d">${esc(l.descricao)}</span><span class="mt">${chipAviso(l).trim()} ${dataBR(l)} · ${esc(l.status)} · ${esc(l.banco||'-')}${serie?' · '+serie:''}${l.oculto?' · oculto':''}</span></span>
      <span class="r">${htmlValorLinha(l)}${prazo}</span>
      <span class="fut-mais" aria-hidden="true">${PONTOS_FUT}</span>
    </button>
    <div class="fut-dk" tabindex="0">
      <span>${dataBR(l)}</span>
      <span class="dd"><span class="dl"><span class="d" title="${esc(l.descricao)}">${esc(l.descricao)}</span>${chipAviso(l)}</span>${sub?`<span class="sub">${esc(sub)}</span>`:''}</span>
      <span class="bk" title="${esc(l.banco||'-')}">${esc(l.banco||'-')}</span>
      <span class="st">${seloStatus(l.status)}</span>
      <span class="v">${htmlValorLinha(l)}${prazo}</span>
      <span class="fut-acoes">${botoes}</span>
    </div>
  </div>`;
}
// Janela de baixo com as acoes do evento (celular).
function abrirAcoesFuturo(id){
  if(id.startsWith('fatura|')) return abrirAcoesFatura(id);
  const l = acharLancamento(id);
  if(!l) return;
  fecharAcoesFuturo();
  const ov = document.createElement('div');
  ov.className = 'fut-ov'; ov.id = 'futOv';
  ov.innerHTML = `<div class="fut-sheet" role="dialog" aria-modal="true" aria-label="Ações de ${esc(l.descricao)}">
    <h3>${esc(l.descricao)}</h3><p class="mm">${fmtMoeda(Math.abs(valorAssinado(l)))} · ${dataBR(l)} · ${esc(l.banco||'-')}</p>
    ${acoesDoEvento(l).map(a=>`<button type="button" class="fut-ac${a.perigo?' perigo':''}${a.pri?' pri':''}" data-acao="${a.k}" data-id="${esc(l.id)}">${a.t}</button>`).join('')}</div>`;
  ov.addEventListener('click', e=>{
    const b = e.target.closest('button[data-acao]');
    if(b){ fecharAcoesFuturo(); executarAcaoRegistro(b.dataset.acao, b.dataset.id); return; }
    if(e.target === ov) fecharAcoesFuturo();
  });
  document.body.appendChild(ov);
  const primeiro = ov.querySelector('.fut-ac');
  if(primeiro) primeiro.focus();
}
function fecharAcoesFuturo(){ const o = document.getElementById('futOv'); if(o) o.remove(); }
document.addEventListener('keydown', e=>{ if(e.key === 'Escape') fecharAcoesFuturo(); });

// No mês buscado os eventos futuros não aparecem na lista; se o mês tiver algum, avisa e leva para a aba Futuros.
// ---------- Histórico: filtros numa janela, lançamentos por dia ou por mês, tabela no computador ----------
// Demo: .ai/demos/demo-historico-layout.html. Reaproveita as classes de Futuros (.fut-*): mesma linha, mesma tabela, mesma janela de ações.
const MQ_TABELA = window.matchMedia('(min-width:1160px)');
MQ_TABELA.addEventListener('change', ()=>renderHistorico());
let termoBuscaHist = '';
const ICONE_NOTA = '<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 3h9l4 4v14H6z"/><path d="M9 12h7M9 16h5"/></svg>';
function htmlNotaIcone(l){
  return l.nota ? `<span class="hist-nota" role="button" tabindex="0" data-hist-nota="${esc(l.id)}" title="${esc(cortar(l.nota,200))}" aria-label="Ver nota">${ICONE_NOTA}</span>` : '';
}
function horaBR(l){ return new Date(l.dataEvento).toLocaleTimeString('pt-BR',{hour:'2-digit',minute:'2-digit'}); }
function tituloDiaHist(l){
  const w = new Date(l.dataEvento).toLocaleDateString('pt-BR',{weekday:'short'}).replace('.','');
  return w.charAt(0).toUpperCase() + w.slice(1) + ' ' + dataBR(l);
}
function htmlLinhaHist(l, busca){
  const id = esc(l.id);
  const cor = estiloCor==='nenhum' ? 'var(--faint)' : corStatus(l.status);
  const tags = (l.tags||[]).map(t=>'#'+esc(t)).join(' ');
  const fut = '';
  const botoes = acoesDoEvento(l).map(a=>`<button type="button" class="${a.perigo?'perigo':''}${a.pri?'pri':''}" data-acao="${a.k}" data-id="${id}">${a.c}</button>`).join('');
  return `<div class="fut-row hist-row ${l.oculto?'oculto':''} ${excluindo.has(l.id)?'excluindo':''}" style="--cor:${cor}">
    <div class="fut-m" role="button" tabindex="0" data-fut-abrir="${id}" aria-haspopup="dialog">
      <span class="l"><span class="d">${esc(l.descricao)}</span>
        <span class="mt"><span>${busca?dataBR(l):horaBR(l)}</span>${seloStatus(l.status)}${chipPagaFatura(l)}<span class="bk">${esc(l.banco||'-')}${l.oculto?' · oculto':''}${fut}</span>${htmlNotaIcone(l)}</span>
        ${tags?`<span class="hist-tags">${tags}</span>`:''}</span>
      <span class="r">${htmlValorLinha(l)}</span>
      <span class="fut-mais" aria-hidden="true">${PONTOS_FUT}</span>
    </div>
    <div class="fut-dk" tabindex="0">
      <span class="dt">${dataBR(l)}<small>${horaBR(l)}</small></span>
      <span class="dd"><span class="dl"><span class="d" title="${esc(l.descricao)}">${esc(l.descricao)}</span></span>${(tags||l.nota)?`<span class="sub hist-sub"><span class="hist-tags-t">${tags}</span>${htmlNotaIcone(l)}</span>`:''}</span>
      <span class="bk" title="${esc(l.banco||'-')}">${esc(l.banco||'-')}${l.oculto?' · oculto':''}</span>
      <span class="st">${seloStatus(l.status)}</span>
      <span class="v">${htmlValorLinha(l)}</span>
      <span class="fut-acoes">${botoes}</span>
    </div>
  </div>`;
}
function htmlListaHist(lista, busca, asc){
  const tabela = MQ_TABELA.matches;
  const porMes = busca || (tabela && periodoHist!=='mes');
  const grupos = [];
  lista.forEach(l=>{
    const k = porMes ? diaLocal(l).slice(0,7) : (tabela ? '' : diaLocal(l));
    let g = grupos[grupos.length-1];
    if(!g || g.k!==k){ g = {k, l, itens:[]}; grupos.push(g); }
    g.itens.push(l);
  });
  const corpo = grupos.map(g=>{
    let tit = '';
    if(porMes) tit = `<div class="fut-titulo"><span>${esc(rotuloMes(g.k))} · ${g.itens.length} lançamento(s)</span></div>`;
    else if(!tabela) tit = `<div class="fut-titulo"><span>${esc(tituloDiaHist(g.l))}</span></div>`;
    return tit + `<div class="fut-grupo">${g.itens.map(l=>htmlLinhaHist(l,busca)).join('')}</div>`;
  }).join('');
  const cab = `<div class="fut-cab"><span><button type="button" class="hist-th" data-hist-ordem>Data ${asc?'↑':'↓'}</button></span><span>Lançamento</span><span>Banco</span><span>Status</span><span style="text-align:right">Valor</span><span></span></div>`;
  return `<div class="fut-tabela"><div class="fut-tabela-in">${cab}${corpo}</div></div>`;
}
function nFiltrosHist(){
  return filStatusSelecionados.size + filBancoSelecionados.size + (seletorHist ? seletorHist.n() : 0)
    + (document.getElementById('filMostrarOcultos').checked?1:0);
}
function renderChipsHist(){
  const chip = (k,v,t)=>`<button type="button" class="hist-chip" data-hist-x="${k}" data-v="${esc(v||'')}" aria-label="Tirar: ${esc(t)}">${esc(t)}<i aria-hidden="true">×</i></button>`;
  const h = [];
  if(modoBuscaTexto) h.push(chip('q','','Busca: '+cortar(termoBuscaHist,30)));
  filStatusSelecionados.forEach(s=>h.push(chip('st',s,s)));
  filBancoSelecionados.forEach(b=>h.push(chip('bk',b,b)));
  if(seletorHist && seletorHist.n()) h.push(chip('tg','','Tags: '+seletorHist.descricao()));
  if(document.getElementById('filMostrarOcultos').checked) h.push(chip('oc','','Com ocultos'));
  if(modoBuscaTexto && document.getElementById('buscaIncluiFuturos').checked) h.push(chip('fu','','Com futuros'));
  document.getElementById('histChips').innerHTML = h.join('');
}
function renderFiltrosHistCampos(){
  const st = document.getElementById('histFilStatus'), bk = document.getElementById('histFilBanco');
  if(!st || !bk) return;
  const b = (k,v,on)=>`<button type="button" class="tag ${on?'on':''}" aria-pressed="${on}" data-hist-f="${k}" data-v="${esc(v)}">${esc(v)}</button>`;
  const presentes = new Set([...lancamentosCache, ...lancamentosMes, ...lancamentosBusca].map(l=>l.status).filter(Boolean));
  const opcoes = [...listas.status, ...[...presentes].filter(x=>!listas.status.includes(x)).sort()]; // inclui os antigos que aparecem no periodo (ex.: CREDITO IN)
  st.innerHTML = opcoes.filter(x=>listas.status.includes(x) || presentes.has(x)).map(x=>b('st',x,filStatusSelecionados.has(x))).join('');
  bk.innerHTML = listas.banco.map(x=>b('bk',x,filBancoSelecionados.has(x))).join('');
}

function renderHistorico(){
  renderFuturos();
  const el = document.getElementById('listaLancamentos');
  const fsList = [...filStatusSelecionados];
  const fbList = [...filBancoSelecionados];
  const mostrarOcultos = document.getElementById('filMostrarOcultos').checked;
  const iniDia = periodoHist==='mes' ? '' : chaveDia(inicioPeriodoHist(periodoHist));
  const noEscopo = l=> periodoHist==='mes' ? diaLocal(l).slice(0,7)===mesSel : diaLocal(l) >= iniDia;
  const fonte = modoBuscaTexto ? lancamentosBusca : (modoMesEspecifico ? lancamentosMes : lancamentosCache).filter(noEscopo);
  document.getElementById('filMesEspecifico').value = mesSel;
  document.getElementById('mesRotulo').textContent = rotuloMes(mesSel);
  document.getElementById('calPeriodo').value = mesSel;
  const tudoAgora = modoBuscaTexto && tudoHist;
  document.querySelectorAll('#histPer [data-hist-per]').forEach(b=>{ const on = !tudoAgora && periodoHist===b.dataset.histPer; b.classList.toggle('on', on); b.setAttribute('aria-pressed', on); });
  const emMes = !tudoAgora && periodoHist==='mes';
  document.getElementById('histCal').classList.toggle('on', emMes);
  document.getElementById('calRotulo').textContent = emMes ? mesAbrev(mesSel) : '';
  document.getElementById('navMes').classList.toggle('hidden', !emMes);
  // No celular o campo é estreito: o período já aparece no grupo de cima e na faixa do resultado.
  document.getElementById('buscaTexto').placeholder = MQ_TABELA.matches
    ? `Buscar neste ${periodoHist==='mes'?'mês':'período'} (descrição, nota ou tag)` : 'Descrição, nota ou tag';
  const hdr = document.querySelector('header');
  document.documentElement.style.setProperty('--hdr-h', (hdr ? hdr.offsetHeight : 0) + 'px');
  // Os previstos ficam na aba Futuros: o servidor nem os manda para o Historico.
  let lista = fonte.filter(l=>
    !l.previsto &&
    (mostrarOcultos || !l.oculto) &&
    (!fsList.length || fsList.includes(l.status)) && (!fbList.length || fbList.includes(l.banco)) && (!seletorHist || seletorHist.casa(l)));
  const ordem = document.getElementById('filOrdenacao').value;
  lista = [...lista].sort((a,b)=> ordem==='asc'
    ? new Date(a.dataEvento) - new Date(b.dataEvento)
    : new Date(b.dataEvento) - new Date(a.dataEvento));
  document.getElementById('contadorHistorico').textContent = modoBuscaTexto
    ? `${lista.length} lançamento(s) exibido(s) do resultado da busca (${lancamentosBusca.length} no total)`
    : periodoHist==='mes' ? `${lista.length} lançamento(s) em ${rotuloMes(mesSel)}` : `${lista.length} lançamento(s) nos últimos ${DIAS_PERIODO[periodoHist]} dias`;
  // Sem total da lista nem soma por mes ate o item 42 (o servidor calcula junto com o filtro de tags).
  document.getElementById('somaHistorico').innerHTML = '';
  const n = nFiltrosHist();
  const bf = document.getElementById('btnFiltros');
  bf.innerHTML = 'Filtros' + (n ? `<span class="n">${n}</span>` : '');
  bf.classList.toggle('on', n>0);
  document.getElementById('histFilOk').textContent = `Ver ${lista.length} lançamento(s)`;
  renderChipsHist();
  document.getElementById('histEscopo').innerHTML = !modoBuscaTexto ? '' : (tudoHist
    ? `<div class="escopo"><span class="t">Resultado de <b>todo o histórico</b></span><button type="button" data-hist-voltar>Voltar a ${periodoHist==='mes'?esc(mesAbrev(mesSel)):periodoHist}</button></div>`
    : `<div class="escopo"><span class="t">Resultado <b>${esc(escopoBuscaTxt)}</b></span><button type="button" data-hist-tudo>Buscar em tudo</button></div>`);
  renderFiltrosHistCampos();
  const asc = ordem==='asc';
  document.getElementById('btnOrdem').textContent = asc ? 'Data ↑' : 'Data ↓';
  const fim = document.getElementById('histFim');
  if(!lista.length){
    el.innerHTML = `<div class="hist-vazio"><b>Nenhum lançamento encontrado.</b>${(n||modoBuscaTexto)?'<button type="button" data-hist-limpar>Limpar busca e filtros</button>':''}</div>`;
    fim.innerHTML = '';
    return;
  }
  el.innerHTML = htmlListaHist(lista, modoBuscaTexto, asc);
  const prox = periodoHist==='30d' ? '90d' : periodoHist==='90d' ? '180d' : '';
  fim.innerHTML = (modoBuscaTexto ? '' : (periodoHist==='mes'
    ? `<div class="hm-fimbar"><button type="button" class="hm-b" data-hist-mes="-1" aria-label="Mês anterior">‹</button><label class="hm-m"><span>${esc(rotuloMes(mesSel))}</span><input type="month" value="${esc(mesSel)}" aria-label="Mês exibido" data-hist-mes-pick></label><button type="button" class="hm-b" data-hist-mes="1" aria-label="Próximo mês">›</button></div>`
    : (prox ? `<button type="button" class="hm-amp" data-hist-per="${prox}">Ver ${prox}</button>` : '')))
    + '<button type="button" class="hm-topo" data-hist-topo>↑ Topo</button>';
}

// Janelas: filtros (estática, em index.html) e nota (montada como a janela de ações de Futuros).
function abrirFiltrosHist(){ document.getElementById('histFilOv').classList.remove('hidden'); document.getElementById('histFilOk').focus(); }
function fecharFiltrosHist(){ document.getElementById('histFilOv').classList.add('hidden'); }
function limparFiltrosHist(){
  filStatusSelecionados = new Set(); filBancoSelecionados = new Set();
  if(seletorHist) seletorHist.limpar();
  document.getElementById('filMostrarOcultos').checked = false;
  document.getElementById('buscaIncluiFuturos').checked = false;
}
function abrirNotaLancamento(id){
  const l = acharLancamento(id);
  if(!l || !l.nota) return;
  fecharAcoesFuturo();
  const ov = document.createElement('div');
  ov.className = 'fut-ov'; ov.id = 'futOv';
  ov.innerHTML = `<div class="fut-sheet hist-nota-sheet" role="dialog" aria-modal="true" aria-label="Nota de ${esc(l.descricao)}">
    <h3>${esc(l.descricao)}</h3><p class="mm">${dataBR(l)} · ${esc(l.banco||'-')}</p>
    <p class="hist-nota-txt">${escHtml(l.nota)}</p><button type="button" class="hf-ok" data-fechar>Fechar</button></div>`;
  ov.addEventListener('click', e=>{ if(e.target === ov || e.target.closest('[data-fechar]')) fecharAcoesFuturo(); });
  document.body.appendChild(ov);
  ov.querySelector('[data-fechar]').focus();
}

function acharLancamento(id){
  return lancamentosCache.find(x=>x.id===id) || lancamentosMes.find(x=>x.id===id) || lancamentosBusca.find(x=>x.id===id) || futurosLista().find(x=>x.id===id);
}
function aplicarLocal(id, patch){
  [lancamentosMes, lancamentosBusca, futurosLista()].forEach(arr=>{ const it = arr.find(x=>x.id===id); if(it) Object.assign(it, patch); });
  renderHistorico();
  renderFuturos();
}
function removerLocal(id){
  lancamentosMes = lancamentosMes.filter(x=>x.id!==id);
  lancamentosBusca = lancamentosBusca.filter(x=>x.id!==id);
  renderHistorico();
}

function iniciarEdicao(id){
  const l = acharLancamento(id);
  if(!l) return;
  const origem = abaAtual();
  abaOrigemEdicao = origem === 'lancar' ? null : origem;
  editandoId = id;
  definirMarcaDoFormulario(l.pagaFatura || null);
  definirStatusDoFormulario(l.status, false);
  document.getElementById('fFatura').value = String(ajusteEfetivo(l));
  document.getElementById('fEstimado').checked = l.valorEstimado === true;
  preencherAvisoNoFormulario(l.aviso);
  atualizarCampoFatura();
  definirTipo(l.tipo);
  document.getElementById('fValor').value = l.valor;
  document.getElementById('fDesc').value = l.descricao;
  document.getElementById('fNota').value = l.nota || '';
  lancarModo('lanc'); lancarMais(!!l.nota);
  const d = new Date(l.dataEvento); d.setMinutes(d.getMinutes()-d.getTimezoneOffset());
  document.getElementById('fData').value = d.toISOString().slice(0,16);
  document.getElementById('fBanco').value = l.banco || '';
  tagsSelecionadas = new Set(l.tags||[]);
  renderTagPicker();
  document.getElementById('btnSalvar').textContent = 'Salvar edição';
  document.getElementById('btnCancelarEdicao').classList.remove('hidden');
  assinaturaEdicao = assinaturaFormulario();
  irParaAba('lancar');
}

function iniciarCopia(id){
  const l = acharLancamento(id);
  if(!l) return;
  editandoId = null; assinaturaEdicao = null;
  document.getElementById('btnSalvar').textContent = 'Salvar lançamento';
  document.getElementById('btnCancelarEdicao').classList.add('hidden');
  document.getElementById('notaLivre').value = '';
  definirStatusDoFormulario(l.status, true); // a copia e um lancamento novo: so um status que o servidor oferece
  document.getElementById('fFatura').value = String(ajustePadrao(document.getElementById('fStatus').value));
  document.getElementById('fEstimado').checked = false;
  preencherAvisoNoFormulario(null); // a copia tem outra data: o aviso e escolhido de novo
  atualizarCampoFatura();
  definirTipo(l.tipo);
  document.getElementById('fValor').value = l.valor;
  document.getElementById('fDesc').value = l.descricao;
  document.getElementById('fNota').value = l.nota || '';
  lancarModo('lanc'); lancarMais(!!l.nota);
  document.getElementById('fData').value = nowLocalInput();
  document.getElementById('fBanco').value = l.banco || '';
  tagsSelecionadas = new Set(l.tags||[]);
  renderTagPicker();
  document.getElementById('iaStatus').textContent = 'Copiado do histórico. Ajuste o que precisar e salve como novo lançamento.';
  irParaAba('lancar');
  window.scrollTo({top:0});
}

function cancelarEdicao(){
  editandoId = null; assinaturaEdicao = null;
  document.getElementById('btnSalvar').textContent = 'Salvar lançamento';
  document.getElementById('btnCancelarEdicao').classList.add('hidden');
  limparFormulario();
  if(abaOrigemEdicao){ const t = abaOrigemEdicao; abaOrigemEdicao = null; Voltar.retornar(); irParaAba(t); } // volta para onde a edição começou (e desfaz a entrada do historico do navegador)
}

async function ocultarLancamento(id){
  if(!db) return;
  const l = acharLancamento(id);
  if(!l) return;
  const novo = !l.oculto;
  await db.collection("lancamentos").doc(id).update({oculto: novo});
  aplicarLocal(id, {oculto: novo});
}

// Exclusão imediata (item 38, fase C): depois do Confirmar o DELETE vai ao servidor na hora e o lançamento fica na lista
// ("Excluindo…") até a resposta; as recusas (mês conferido, mês fechado, rede) chegam antes de ele sumir. Só então some
// de todas as telas e o aviso oferece Desfazer por 6 s, que é uma restauração da versão que o DELETE devolveu.
const excluindo = new Set(); // ids com DELETE em andamento
const exclusaoPendente = new Map(); // id -> {c, m, b}: o item guardado de cada lista enquanto está excluído e o Desfazer vale
function atualizarVistas(){
  try{ renderTagPicker(); }catch(e){}
  cartaoUltimaCalc = 0;
  renderHistorico();
  try{ desenharGrafico(); }catch(e){}
  renderCartao();
  renderFuturos();
}
function esconderLocal(id){
  exclusaoPendente.set(id, {
    c: lancamentosCache.find(x=>x.id===id),
    m: lancamentosMes.find(x=>x.id===id),
    b: lancamentosBusca.find(x=>x.id===id)
  });
  lancamentosCache = lancamentosCache.filter(x=>x.id!==id);
  lancamentosMes = lancamentosMes.filter(x=>x.id!==id);
  lancamentosBusca = lancamentosBusca.filter(x=>x.id!==id);
  atualizarVistas();
}
function devolverLocal(id){
  const g = exclusaoPendente.get(id);
  if(!g) return;
  exclusaoPendente.delete(id);
  const porData = (a, b)=> a.dataEvento < b.dataEvento ? 1 : (a.dataEvento > b.dataEvento ? -1 : 0);
  if(g.c) lancamentosCache = lancamentosCache.concat([g.c]).sort(porData);
  if(g.m) lancamentosMes = lancamentosMes.concat([g.m]).sort(porData);
  if(g.b) lancamentosBusca = lancamentosBusca.concat([g.b]).sort(porData);
  atualizarVistas();
}
function cortar(t, n){ t = String(t || ''); return t.length > n ? t.slice(0, n) + '…' : t; }
async function iniciarExclusao(id, desc){
  if(excluindo.has(id)) return;
  excluindo.add(id); atualizarVistas();
  const anoAntigo = anoAntigoDoItem(id);
  if(anoAntigo) anoDoItemAntigo.set(id, anoAntigo);
  lerSoOAnoAntigo(anoAntigo);
  let r;
  try{
    r = await db.collection("lancamentos").doc(id).delete();
  }catch(err){
    console.error(err);
    futurosSoAnos = null; anoDoItemAntigo.delete(id);
    excluindo.delete(id); atualizarVistas();
    if(err && err.codigo === 'cancelado'){ toast(err.message); return; }
    if(err && err.codigo === 'mes_conferido'){ toast(err.message, true); return; }
    mostrarAvisoAcao({
      txt: 'Não foi possível excluir "' + cortar(desc, 40) + '".',
      sub: 'Confira a conexão. O lançamento continua na lista.',
      rotulo: 'Tentar de novo', erro: true, ms: 8000,
      onAcao: ()=>iniciarExclusao(id, desc)
    });
    return;
  }
  excluindo.delete(id);
  esconderLocal(id);
  const versao = r && r.desfazer && r.desfazer.versao;
  mostrarAvisoAcao({
    txt: 'Excluído: ' + cortar(desc, 40), sub: 'Dá para desfazer por 6 segundos.', rotulo: 'Desfazer',
    onAcao: ()=>desfazerExclusao(id, versao),
    onFim: ()=>exclusaoPendente.delete(id)
  });
}
// Desfazer = restaurar a versão anterior à exclusão (a API a devolve no DELETE). Se falhar, o lançamento continua excluído.
async function desfazerExclusao(id, versao){
  const falha = (m)=>{ exclusaoPendente.delete(id); toast((m || 'Não foi possível desfazer.') + ' O lançamento continua excluído: recupere em Histórico de versões.', true); };
  if(!versao){ falha(); return; }
  lerSoOAnoAntigo(anoDoItemAntigo.get(id));
  try{
    const r = await db.restaurarVersao(id, versao);
    if(r.status !== 200){ futurosSoAnos = null; falha(r.body && r.body.erro); return; }
    devolverLocal(id);
    toast('Exclusão desfeita');
  }catch(err){
    console.error(err);
    futurosSoAnos = null;
    falha(msgErro(err));
  }
}
async function excluirLancamento(id){
  if(!db) return;
  const l = acharLancamento(id);
  if(!l) return;
  const resumo = cortar(l.descricao, 40) + ' · ' + fmtMoeda(l.valor);
  if(!(await confirmarModal(`Excluir este lançamento (${resumo})? Depois de excluir, dá para desfazer por 6 segundos.`))) return;
  await iniciarExclusao(id, l.descricao);
}

// Desfazer de uma edição: reenvia, como edição normal, os valores que a edição trocou (o PUT mescla campo a campo).
// Campo que não existia antes volta como null.
async function desfazerEdicao(id, antes, doc){
  const patch = {};
  Object.keys(doc).forEach(k=>{ patch[k] = antes[k] !== undefined ? antes[k] : null; });
  try{
    await db.collection("lancamentos").doc(id).update(patch);
    aplicarLocal(id, patch);
    toast('Edição desfeita');
  }catch(err){
    console.error(err);
    toast(msgErro(err, 'Não foi possível desfazer. Edite o lançamento de novo.'), true);
  }
}
function resumoEdicao(antes, doc){
  const campos = [['valor','Valor',fmtMoeda],['descricao','Descrição',v=>cortar(v,30)],['banco','Banco',v=>v],['status','Status',v=>v],
    ['tipo','Tipo',v=>v],['dataEvento','Data',v=>fmtDataCompleta(diaLocal({dataEvento:v}))]];
  const igual = (k)=> k === 'dataEvento' ? new Date(antes[k]).getTime() === new Date(doc[k]).getTime() : String(antes[k]) === String(doc[k]);
  const m = campos.filter(([k])=>antes[k] !== undefined && !igual(k))
    .map(([k,rot,f])=>`${rot}: ${f(antes[k])} → ${f(doc[k])}`);
  return m.length ? m.join('; ') : 'Nenhum campo principal mudou.';
}

// ---------- A fatura do cartao em Futuros (itens 33b e 35; demo .ai/demos/demo-pagar-fatura.html) ----------
// O servidor manda a fatura pronta (vencimento, valor, pagador sugerido, atraso). "Ja paguei" lanca um pagamento normal, ja marcado com a fatura.
function faturaPorChave(chave){
  const [, meio, ym] = chave.split('|');
  return ((futurosDados && futurosDados.faturas) || []).find(f=>f.meio===meio && f.ym===ym);
}
function htmlFaturaFuturo(f){
  const chave = esc('fatura|' + f.meio + '|' + f.ym), dia = fmtDataCompleta(f.vencimento);
  const n = diasAte(f.vencimento);
  const p = n < 0 ? {t:`atrasado ${-n} dia${n===-1?'':'s'}`, atras:true} : (n === 0 ? {t:'hoje', atras:false} : {t:`em ${n} dia${n===1?'':'s'}`, atras:false});
  const prazo = `<span class="fut-prazo ${p.atras?'atras':''}">${p.t}</span>`;
  const valor = htmlValorLinha({tipo:'saida', valor:f.valor});
  const pagador = f.pagador ? esc(f.pagador) : 'a definir';
  const nome = `Fatura ${esc(f.rotulo)} · ${esc(f.meio)}`;
  const botao = ehOwner() ? `<button type="button" data-acao="avisarFatura" data-id="${chave}">Avisar</button><button type="button" class="pri" data-acao="pagarFatura" data-id="${chave}">Já paguei</button>` : '';
  const sino = f.aviso ? `🔔 ${esc(Recorrencia.fraseAviso({on:true, dias:f.aviso.dias, insistir:f.aviso.insistir}))}` : '';
  return `<div class="fut-row" style="--cor:var(--faint)">
    <button type="button" class="fut-m" data-fut-abrir="${chave}" aria-haspopup="dialog">
      <span class="l"><span class="d">${nome}</span><span class="mt">${dia} · ${pagador} · ${f.estado}</span>${sino?`<span class="mt fut-sino">${sino}</span>`:''}</span>
      <span class="r">${valor}${prazo}</span>
      <span class="fut-mais" aria-hidden="true">${PONTOS_FUT}</span>
    </button>
    <div class="fut-dk" tabindex="0">
      <span>${dia}</span>
      <span class="dd"><span class="dl"><span class="d" title="${nome}">${nome}</span></span><span class="sub">${f.estado==='aberta'?'ainda cresce a cada compra':'fechada'}</span>${sino?`<span class="sub fut-sino">${sino}</span>`:''}</span>
      <span class="bk" title="${pagador}">${pagador}</span>
      <span class="st"><span class="muted">${f.estado}</span></span>
      <span class="v">${valor}${prazo}</span>
      <span class="fut-acoes">${botao}</span>
    </div>
  </div>`;
}
function abrirAcoesFatura(chave){
  const f = faturaPorChave(chave);
  if(!f) return;
  fecharAcoesFuturo();
  const ov = document.createElement('div');
  ov.className = 'fut-ov'; ov.id = 'futOv';
  ov.innerHTML = `<div class="fut-sheet" role="dialog" aria-modal="true" aria-label="Ações da fatura ${esc(f.rotulo)}">
    <h3>Fatura ${esc(f.rotulo)} · ${esc(f.meio)}</h3><p class="mm">${fmtMoeda(f.valor)} · vence ${fmtDataCompleta(f.vencimento)} · ${f.estado}</p>
    ${ehOwner() ? `<button type="button" class="fut-ac pri" data-acao="pagarFatura" data-id="${esc(chave)}">Já paguei</button><button type="button" class="fut-ac" data-acao="avisarFatura" data-id="${esc(chave)}">Avisar no Telegram</button>` : '<p class="muted" style="margin:8px">Só o dono lança o pagamento da fatura.</p>'}</div>`;
  ov.addEventListener('click', e=>{
    const b = e.target.closest('button[data-acao]');
    if(b){ fecharAcoesFuturo(); executarAcaoRegistro(b.dataset.acao, b.dataset.id); return; }
    if(e.target === ov) fecharAcoesFuturo();
  });
  document.body.appendChild(ov);
}
// A janela de pagar: valor (o que a operadora cobra, mas pode ser qualquer valor), data, status e o meio de saida; tudo ja vem sugerido.
function pedirPagamentoFatura(f){
  return new Promise(async resolve=>{
    const hoje = chaveDia(new Date());
    const sugerir = async (dia)=>{
      try{ return (await db.statusSugeridos(new Date(dia + 'T12:00:00').toISOString())).status || []; }
      catch(e){ return [...listas.status]; }
    };
    let lista = await sugerir(hoje);
    const opcoes = (itens, atual)=>itens.map(x=>`<option value="${esc(x)}" ${x===atual?'selected':''}>${esc(x)}</option>`).join('');
    const meios = listas.banco;
    const meioInicial = f.sugestao.pagador && meios.includes(f.sugestao.pagador) ? f.sugestao.pagador : meios[0];
    document.getElementById('modalMsg').textContent = 'Já paguei a fatura ' + f.rotulo + ' · ' + f.meio;
    document.getElementById('modalExtra').innerHTML =
      `<label>Valor</label><input type="number" step="0.01" id="pfValor" value="${Number(f.sugestao.valor) || ''}">
       <label>Data do pagamento</label><input type="date" id="pfData" value="${hoje}" max="${hoje}">
       <label>Status</label><select id="pfStatus">${opcoes(lista, lista[0])}</select>
       <label>Meio de saída</label><select id="pfMeio">${opcoes(meios, meioInicial)}</select>
       <p class="muted" style="margin:6px 0 0;">O lançamento sai marcado com <b>Fatura ${esc(f.rotulo)} · ${esc(f.meio)}</b>. Pode ser o total, o mínimo ou qualquer valor.</p>
       <p class="muted" id="pfErro" style="color:var(--danger);margin:6px 0 0;"></p>`;
    document.getElementById('modalBotoes').innerHTML = '<button class="ghost" id="modalCancelar">Cancelar</button><button class="primary" id="modalConfirmar">Confirmar</button>';
    document.getElementById('modalOverlay').classList.remove('hidden');
    const fechar = (v)=>{ document.getElementById('modalOverlay').classList.add('hidden'); document.getElementById('modalExtra').innerHTML = ''; resolve(v); };
    document.getElementById('pfData').addEventListener('change', async ()=>{
      const dia = document.getElementById('pfData').value;
      if(!dia) return;
      const atual = document.getElementById('pfStatus').value;
      lista = await sugerir(dia);
      document.getElementById('pfStatus').innerHTML = opcoes(lista, lista.includes(atual) ? atual : lista[0]);
    });
    document.getElementById('modalCancelar').onclick = ()=>fechar(null);
    document.getElementById('modalConfirmar').onclick = ()=>{
      const valor = parseFloat(document.getElementById('pfValor').value);
      const dia = document.getElementById('pfData').value;
      const status = document.getElementById('pfStatus').value;
      const meio = document.getElementById('pfMeio').value;
      const erro = document.getElementById('pfErro');
      if(!(valor > 0)){ erro.textContent = 'Informe um valor maior que zero.'; return; }
      if(!dia){ erro.textContent = 'Informe a data.'; return; }
      if(!status || !meio){ erro.textContent = 'Escolha o status e o meio de saída.'; return; }
      fechar({valor, status, banco: meio, dataEvento: new Date(dia + 'T12:00:00').toISOString()});
    };
  });
}
async function pagarFatura(chave){
  const f = faturaPorChave(chave);
  if(!f || !db) return;
  const d = await pedirPagamentoFatura(f);
  if(!d) return;
  let r;
  try{
    r = await db.collection("lancamentos").add({status:d.status, tipo:'saida', valor:d.valor, descricao:f.sugestao.descricao, nota:'', dataEvento:d.dataEvento,
      banco:d.banco, tags:[], pagaFatura:{meio:f.meio, ym:f.ym}, criadoEm:new Date().toISOString()});
  }catch(err){
    console.error(err);
    toast((err && err.message) || 'Não foi possível lançar o pagamento. Tente de novo.', err && err.codigo !== 'cancelado');
    return;
  }
  mostrarAvisoAcao({
    txt: 'Pagamento lançado: Fatura ' + f.rotulo + ' · ' + f.meio, sub: 'Está no Histórico, marcado com a fatura. Dá para desfazer por 6 segundos.', rotulo: 'Desfazer',
    onAcao: ()=>desfazerPagamentoFatura(r.id)
  });
}
// Desfazer = excluir o lancamento do pagamento (a fatura volta a Futuros sozinha).
async function desfazerPagamentoFatura(id){
  try{ await db.collection("lancamentos").doc(id).delete(); toast('Pagamento desfeito'); }
  catch(err){ console.error(err); toast((err && err.message) || 'Não foi possível desfazer. Exclua o lançamento no Histórico.', true); }
}
// ---------- Confirmar um previsto (demo .ai/demos/demo-confirmar-previsto.html): valor, data e status; o servidor decide o resto ----------
// A janela pergunta o valor (ja vem o previsto; da para trocar: multa e juros), a data (hoje; nunca depois de hoje) e o status, da lista do servidor para aquela data.
function pedirConfirmacaoPrevisto(l){
  return new Promise(async resolve=>{
    const saida = l.tipo === 'saida';
    const hoje = chaveDia(new Date());
    const sugerir = async (dia)=>{
      try{ return (await db.statusSugeridos(new Date(dia + 'T12:00:00').toISOString())).status || []; }
      catch(e){ return [...listas.status]; }
    };
    let lista = await sugerir(hoje);
    const opcoes = (lista, atual)=>lista.map(x=>`<option value="${esc(x)}" ${x===atual?'selected':''}>${esc(x)}</option>`).join('');
    const inicial = lista.includes(l.status) ? l.status : lista[0];
    document.getElementById('modalMsg').textContent = (saida ? 'Confirmar pagamento: ' : 'Confirmar recebimento: ') + cortar(l.descricao, 60);
    document.getElementById('modalExtra').innerHTML =
      `<label>Valor</label><input type="number" step="0.01" id="cpValor" value="${Number(l.valor) || ''}">
       <label>Data ${saida ? 'do pagamento' : 'do recebimento'}</label><input type="date" id="cpData" value="${hoje}" max="${hoje}">
       <label>Status</label><select id="cpStatus">${opcoes(lista, inicial)}</select>
       <p class="muted" id="cpErro" style="color:var(--danger);margin:6px 0 0;"></p>`;
    document.getElementById('modalBotoes').innerHTML = '<button class="ghost" id="modalCancelar">Cancelar</button><button class="primary" id="modalConfirmar">Confirmar</button>';
    document.getElementById('modalOverlay').classList.remove('hidden');
    const fechar = (v)=>{ document.getElementById('modalOverlay').classList.add('hidden'); document.getElementById('modalExtra').innerHTML = ''; resolve(v); };
    document.getElementById('cpData').addEventListener('change', async ()=>{
      const dia = document.getElementById('cpData').value;
      if(!dia) return;
      const atual = document.getElementById('cpStatus').value;
      lista = await sugerir(dia);
      document.getElementById('cpStatus').innerHTML = opcoes(lista, lista.includes(atual) ? atual : lista[0]);
    });
    document.getElementById('modalCancelar').onclick = ()=>fechar(null);
    document.getElementById('modalConfirmar').onclick = ()=>{
      const valor = parseFloat(document.getElementById('cpValor').value);
      const dia = document.getElementById('cpData').value;
      const status = document.getElementById('cpStatus').value;
      const erro = document.getElementById('cpErro');
      if(!(valor > 0)){ erro.textContent = 'Informe um valor maior que zero.'; return; }
      if(!dia){ erro.textContent = 'Informe a data.'; return; }
      if(!status){ erro.textContent = 'Escolha o status.'; return; }
      fechar({valor, status, dataEvento: new Date(dia + 'T12:00:00').toISOString()});
    };
  });
}
async function confirmarPrevisto(id){
  const l = acharLancamento(id);
  if(!l || !db) return;
  const dados = await pedirConfirmacaoPrevisto(l);
  if(!dados) return;
  let r;
  try{ r = await db.confirmarPrevisto(id, l.dataEvento, dados); }
  catch(err){
    console.error(err);
    toast((err && err.message) || 'Não foi possível confirmar. Tente de novo.', err && err.codigo !== 'cancelado');
    return;
  }
  const versao = r && r.desfazer && r.desfazer.versao;
  mostrarAvisoAcao({
    txt: (l.tipo === 'saida' ? 'Pagamento confirmado: ' : 'Recebimento confirmado: ') + cortar(l.descricao, 40),
    sub: 'Agora está no Histórico e conta no saldo. Dá para desfazer por 6 segundos.', rotulo: 'Desfazer',
    onAcao: ()=>desfazerConfirmacao(id, versao)
  });
}
// Desfazer = restaurar a versao de antes da confirmacao (a API a devolve): o lancamento volta a Futuros como estava.
async function desfazerConfirmacao(id, versao){
  if(!versao){ toast('Não foi possível desfazer.', true); return; }
  try{
    const r = await db.restaurarVersao(id, versao);
    if(r.status !== 200){ toast((r.body && r.body.erro) || 'Não foi possível desfazer.', true); return; }
    toast('Confirmação desfeita');
  }catch(err){
    console.error(err);
    toast((err && err.message) || 'Não foi possível desfazer.', true);
  }
}
function executarAcaoRegistro(acao, id){
  if(acao==='pagarFatura') return pagarFatura(id);
  if(acao==='avisarFatura') return abrirAvisoFatura(faturaPorChave(id));
  if(acao==='confirmar') confirmarPrevisto(id);
  if(acao==='editar') iniciarEdicao(id);
  if(acao==='copiar') iniciarCopia(id);
  if(acao==='prever') preverLancamento(id);
  if(acao==='ocultar') ocultarLancamento(id);
  if(acao==='excluir') excluirLancamento(id);
}
function cliqueAcaoRegistro(e){
  const btn = e.target.closest('button[data-acao]');
  if(!btn) return;
  executarAcaoRegistro(btn.dataset.acao, btn.dataset.id);
}
document.getElementById('listaLancamentos').addEventListener('click', cliqueAcaoRegistro);
document.getElementById('listaFuturos').addEventListener('click', cliqueAcaoRegistro);
document.getElementById('listaLancamentos').addEventListener('click', e=>{
  const n = e.target.closest('[data-hist-nota]');
  if(n){ abrirNotaLancamento(n.dataset.histNota); return; }
  const b = e.target.closest('[data-fut-abrir]');
  if(b){ abrirAcoesFuturo(b.dataset.futAbrir); return; }
  if(e.target.closest('[data-hist-ordem]')){ document.getElementById('btnOrdem').click(); return; }
  if(e.target.closest('[data-hist-limpar]')) limparTudoHist();
});
document.getElementById('listaLancamentos').addEventListener('keydown', e=>{
  if((e.key==='Enter' || e.key===' ') && e.target.matches('.fut-m,.hist-nota')){ e.preventDefault(); e.target.click(); }
});
document.getElementById('histFim').addEventListener('click', e=>{
  const m = e.target.closest('[data-hist-mes]');
  if(m){ deslocarMes(+m.dataset.histMes); return; }
  const pp = e.target.closest('[data-hist-per]');
  if(pp){ irParaPeriodo(pp.dataset.histPer); return; }
  const pk = e.target.closest('[data-hist-mes-pick]');
  if(pk){ try{ pk.showPicker(); }catch(_){} return; }
  if(e.target.closest('[data-hist-topo]')) window.scrollTo({top:0, behavior:'smooth'});
});
document.getElementById('histFim').addEventListener('change', e=>{
  if(e.target.matches('[data-hist-mes-pick]') && e.target.value) irParaMes(e.target.value);
});
document.getElementById('btnFiltros').addEventListener('click', abrirFiltrosHist);
document.getElementById('histFilOk').addEventListener('click', fecharFiltrosHist);
document.getElementById('histFilOv').addEventListener('click', e=>{ if(e.target.id==='histFilOv') fecharFiltrosHist(); });
document.getElementById('histFilLimpar').addEventListener('click', ()=>{ limparFiltrosHist(); renderHistorico(); });
document.getElementById('histFilJan').addEventListener('click', e=>{
  const f = e.target.closest('[data-hist-f]');
  if(!f) return;
  const set = f.dataset.histF==='st' ? filStatusSelecionados : filBancoSelecionados;
  set.has(f.dataset.v) ? set.delete(f.dataset.v) : set.add(f.dataset.v);
  renderHistorico();
});
document.getElementById('histChips').addEventListener('click', e=>{
  const c = e.target.closest('[data-hist-x]');
  if(!c) return;
  const k = c.dataset.histX;
  if(k==='q'){ limparBuscaTexto(); return; }
  if(k==='st') filStatusSelecionados.delete(c.dataset.v);
  if(k==='bk') filBancoSelecionados.delete(c.dataset.v);
  if(k==='tg' && seletorHist) seletorHist.limpar();
  if(k==='oc') document.getElementById('filMostrarOcultos').checked = false;
  if(k==='fu') document.getElementById('buscaIncluiFuturos').checked = false;
  renderHistorico();
});
document.addEventListener('keydown', e=>{ if(e.key==='Escape') fecharFiltrosHist(); });
function limparTudoHist(){
  limparFiltrosHist();
  if(modoBuscaTexto){ limparBuscaTexto(); return; }
  renderHistorico();
}
document.getElementById('listaFuturos').addEventListener('click', e=>{
  const b = e.target.closest('[data-fut-abrir]');
  if(b) abrirAcoesFuturo(b.dataset.futAbrir);
  const bl = e.target.closest('[data-fut-bloco]');
  if(bl){ futurosBlocos[bl.dataset.futBloco] = bl.getAttribute('aria-expanded') !== 'true'; renderFuturos(); }
  if(e.target.closest('#futAnoVer')) buscarAtrasadosDoAno();
});
document.getElementById('listaFuturos').addEventListener('change', e=>{ if(e.target.id === 'futAno') futurosAnoSel = e.target.value; });
document.getElementById('futMostrarOcultos').addEventListener('change', renderFuturos);
document.getElementById('buscaIncluiFuturos').addEventListener('change', renderHistorico);
function inicioPeriodoHist(p){ const d = new Date(); d.setHours(0,0,0,0); d.setDate(d.getDate() - DIAS_PERIODO[p]); return d; }
function mesAbrev(ym){ const [y,m] = ym.split('-').map(Number); return MESES_NOME[m-1].slice(0,3) + '/' + y; }
// Limites (ISO) do que a tela mostra agora; fim vazio = sem limite superior.
function limitesEscopoHist(){
  if(periodoHist==='mes'){
    const [a, m] = mesSel.split('-').map(Number);
    return {ini:new Date(a, m-1, 1).toISOString(), fim:new Date(a, m, 1).toISOString()};
  }
  return {ini:inicioPeriodoHist(periodoHist).toISOString(), fim:''};
}
// O cache guarda os 1000 lançamentos mais recentes: serve se for tudo o que existe ou se já chega até o início do escopo.
function cacheCobre(ini){
  const c = lancamentosCache;
  return c.length < 1000 || (c.length > 0 && new Date(c[c.length-1].dataEvento) <= new Date(ini));
}
async function carregarEscopoHist(){
  const {ini, fim} = limitesEscopoHist();
  if(cacheCobre(ini)){ modoMesEspecifico = false; renderHistorico(); return; }
  if(!db) return;
  const meu = ++tokenMes;
  document.getElementById('contadorHistorico').textContent = "Buscando...";
  document.getElementById('somaHistorico').innerHTML = '';
  try{
    let q = db.collection("lancamentos").where("dataEvento", ">=", ini);
    if(fim) q = q.where("dataEvento", "<", fim);
    const snap = await q.orderBy("dataEvento", "desc").limit(1000).get();
    if(meu !== tokenMes) return; // o usuário já foi para outro período
    lancamentosMes = snap.docs.map(d=>({id:d.id, ...d.data()})).filter(l=>!exclusaoPendente.has(l.id));
    modoMesEspecifico = true;
    renderHistorico();
  }catch(err){
    if(meu === tokenMes){
      document.getElementById('contadorHistorico').textContent = "Não consegui buscar esse período: " + err.message;
      document.getElementById('somaHistorico').innerHTML = '';
    }
  }
}
// Trocar o período ou o mês recarrega a lista e, havendo busca, refaz a busca no novo escopo (o que está marcado é o que se busca).
async function aplicarEscopoHist(){
  await carregarEscopoHist();
  if(modoBuscaTexto) buscarTextoHistorico(false, termoBuscaHist);
}
function irParaMes(ym){
  if(!/^\d{4}-\d{2}$/.test(ym)) return;
  mesSel = ym; periodoHist = 'mes'; tudoHist = false;
  return aplicarEscopoHist();
}
function irParaPeriodo(p){
  if(!DIAS_PERIODO[p]) return;
  periodoHist = p; tudoHist = false;
  return aplicarEscopoHist();
}
function deslocarMes(delta){
  const [a, m] = mesSel.split('-').map(Number);
  const d = new Date(a, m-1+delta, 1);
  irParaMes(d.getFullYear()+'-'+String(d.getMonth()+1).padStart(2,'0'));
}
document.getElementById('mesAnterior').addEventListener('click', ()=>deslocarMes(-1));
document.getElementById('mesProximo').addEventListener('click', ()=>deslocarMes(1));
document.getElementById('histPer').addEventListener('click', e=>{
  const b = e.target.closest('[data-hist-per]');
  if(b){ irParaPeriodo(b.dataset.histPer); return; }
  const p = e.target.closest('[data-hist-pick]');
  if(p){ try{ p.showPicker(); }catch(_){} } // abre o seletor de mês mesmo quando o campo está escondido sob o botão
});
document.getElementById('histPer').addEventListener('change', e=>{ if(e.target.matches('[data-hist-pick]') && e.target.value) irParaMes(e.target.value); });
document.getElementById('btnOrdem').addEventListener('click', ()=>{
  const inp = document.getElementById('filOrdenacao');
  const asc = inp.value !== 'asc';
  inp.value = asc ? 'asc' : 'desc';
  const b = document.getElementById('btnOrdem');
  b.textContent = asc ? 'Data ↑' : 'Data ↓';
  const t = asc ? 'Mais antigos primeiro' : 'Mais recentes primeiro';
  b.title = t + '. Toque para inverter.';
  b.setAttribute('aria-label', 'Ordem: ' + t.toLowerCase() + '. Toque para inverter.');
  renderHistorico();
});

// Texto da busca: sem acento, sem maiúsculas, sem "#" no começo (quem digita #mercado procura a tag) e sem espaços sobrando.
function termoBusca(s){ return normTxt(s).replace(/^[#\s]+/, '').replace(/\s+/g, ' ').trim(); }
// Onde a busca olha: descrição, nota e tags, também sem acento. O dado guardado não muda.
function textoBuscavel(l){ return normTxt((l.descricao||'') + ' ' + (l.nota||'') + ' ' + (Array.isArray(l.tags) ? l.tags.join(' ') : '')); }
// Busca por texto no que está marcado (período ou mês) ou, pedindo "Buscar em tudo", em todo o histórico.
// A API não tem operador de "contém" no banco: varremos o intervalo em blocos de 1000, do mais recente pro mais antigo,
// e comparamos o texto no código. No período marcado é rápido; em tudo cresce com o histórico, por isso há o aviso.
async function buscarTextoHistorico(tudo, termoFixo){
  const entrada = (termoFixo !== undefined ? termoFixo : document.getElementById('buscaTexto').value).trim();
  const termo = termoBusca(entrada);
  const statusEl = document.getElementById('buscaTextoStatus');
  const btn = document.getElementById('btnBuscarTexto');
  if(!termo){ statusEl.textContent = "Digite algo pra buscar."; return; }
  if(!db){ statusEl.textContent = "Banco de dados indisponível nesta visualização."; return; }
  btn.disabled = true;
  const lim = tudo ? {ini:'', fim:''} : limitesEscopoHist();
  let encontrados = [];
  let totalVarrido = 0;
  let cursor = null;
  try{
    while(true){
      let q = db.collection("lancamentos").orderBy("dataEvento","desc");
      if(lim.ini) q = q.where("dataEvento",">=",lim.ini);
      const ate = cursor || lim.fim;
      if(ate) q = q.where("dataEvento","<",ate);
      const snap = await q.limit(1000).get();
      if(snap.docs.length===0) break;
      snap.docs.forEach(d=>{
        const l = {id:d.id, ...d.data()};
        totalVarrido++;
        if(textoBuscavel(l).includes(termo)) encontrados.push(l);
      });
      statusEl.textContent = `Varrendo... ${totalVarrido} lançamento(s) verificados, ${encontrados.length} encontrado(s) até agora.`;
      cursor = snap.docs[snap.docs.length-1].data().dataEvento;
      if(snap.docs.length<1000) break;
    }
    lancamentosBusca = encontrados.filter(l=>!exclusaoPendente.has(l.id));
    modoBuscaTexto = true;
    tudoHist = !!tudo;
    escopoBuscaTxt = tudo ? 'de todo o histórico' : (periodoHist==='mes' ? 'de ' + rotuloMes(mesSel).toLowerCase() : `dos últimos ${DIAS_PERIODO[periodoHist]} dias`);
    termoBuscaHist = entrada;
    renderHistorico();
    statusEl.textContent = `Busca concluída: ${encontrados.length} resultado(s) em ${totalVarrido} lançamento(s) verificados.`;
  }catch(e){
    statusEl.textContent = "Erro na busca: " + e.message;
  }
  btn.disabled = false;
}
function limparBuscaTexto(){
  modoBuscaTexto = false;
  tudoHist = false;
  document.getElementById('buscaTexto').value = '';
  document.getElementById('buscaTextoStatus').textContent = '';
  renderHistorico();
}
document.getElementById('btnBuscarTexto').addEventListener('click', ()=>buscarTextoHistorico(false));
document.getElementById('buscaTexto').addEventListener('keydown', e=>{ if(e.key==='Enter'){ e.preventDefault(); buscarTextoHistorico(false); } });
document.getElementById('histEscopo').addEventListener('click', async e=>{
  if(e.target.closest('[data-hist-tudo]')){
    const ok = await confirmarModal('Isso olha todos os lançamentos, de todos os anos, e pode demorar alguns segundos, principalmente no celular. Se o que você procura é recente, escolha 180d ou um mês no calendário: é bem mais rápido.', 'Buscar em tudo');
    if(ok) buscarTextoHistorico(true, termoBuscaHist);
    return;
  }
  if(e.target.closest('[data-hist-voltar]')) buscarTextoHistorico(false, termoBuscaHist);
});
document.getElementById('filMostrarOcultos').addEventListener('change', renderHistorico);
document.getElementById('filOrdenacao').addEventListener('change', renderHistorico);

// O grafico de Tendencias e a consulta por tags vivem em js/tendencias.js (tendRender).
function desenharGrafico(){ if(abaAtual()==='tendencias' && typeof tendRender==='function') tendRender(); }

async function carregarSaldosConfig(){
  if(!db) return;
  try{
    const snap = await db.doc("config/saldos").get();
    const d = snap.data();
    saldosConfig = clonar((d && d.porBanco) || {});
  }catch(e){
    console.error('Falha ao carregar config/saldos', e);
  }
  renderSaldos();
}

function ordemManualEfetiva(){
  const nomes = listas.banco;
  const base = ordemManual.filter(n=>nomes.includes(n));
  nomes.forEach(n=>{ if(!base.includes(n)) base.push(n); });
  return base;
}
function indicesOrdenados(){
  const idx = listas.banco.map((_,i)=>i);
  if(ordemSaldos==='valor'){
    const v = i=>{ const r = saldosAtuais[listas.banco[i]]; return r ? r.saldo : -Infinity; };
    return idx.sort((a,b)=> (v(b)-v(a)) || (a-b));
  }
  if(ordemSaldos==='manual'){
    const ef = ordemManualEfetiva();
    return idx.sort((a,b)=> ef.indexOf(listas.banco[a]) - ef.indexOf(listas.banco[b]));
  }
  return idx;
}
function agendarSalvarOrdem(){
  clearTimeout(timerOrdem);
  timerOrdem = setTimeout(()=>salvarListas(), 600);
}
function totalSaldos(){
  const t = saldosTotal;
  return t ? {t:t.valor, n:t.meios, fora:t.fora, incompleto:t.erros>0} : {t:0, n:0, fora:0, incompleto:false};
}
function classeVal(r){ return 'sd-val' + (!r ? ' carr' : (saldosCarregando ? ' carr' : (r.saldo<0 ? ' neg' : ''))); }
function textoVal(r){ return r ? (r.erro ? 'erro' : fmtMoeda(r.saldo)) + (r.incompleto ? ' ⚠' : '') : '···'; }
function painelSaldoHtml(i){
  const b = listas.banco[i];
  const cfg = saldosConfig[b] || {saldoInicial:0, dataInicio:''};
  const d = (rascunhoSaldo && rascunhoSaldo.idx===i) ? rascunhoSaldo : {si:String(cfg.saldoInicial||0), di:cfg.dataInicio||'', entra:!cfg.foraDoTotal};
  return `<div class="sd-painel">
    <b>Configurar ${esc(b)}</b>
    <div class="row" style="margin-top:8px;">
      <div><label>Saldo inicial (o que a conta tinha na data abaixo)</label><input type="number" step="0.01" id="sInicial" value="${esc(d.si)}"></div>
      <div><label>Data de início (AAAA-MM-DD)</label><input type="text" inputmode="numeric" placeholder="2026-09-01" id="sData" value="${esc(d.di)}"></div>
    </div>
    <label style="display:flex;align-items:center;gap:8px;margin-top:10px;cursor:pointer;"><input type="checkbox" id="sEntra" style="width:auto;" ${d.entra?'checked':''}> Entra no total</label>
    <p class="muted" id="sMsg" style="margin-top:6px;">${esc(msgSaldo||'')}</p>
    <button class="primary" id="btnSalvarSaldo" data-i="${i}">Salvar</button>
    ${ultimoConf(saldosAtuais[b]) ? `<p class="muted" style="margin-top:6px;">Os meses até ${rotuloMesCurto(ultimoConf(saldosAtuais[b]))} estão conferidos e não mudam: o saldo inicial só vale para os meses depois deles.</p>` : ''}
    ${blocoFechamentoHtml(b)}
  </div>`;
}
function renderSaldos(){
  const el = document.getElementById('listaSaldos');
  if(!el) return;
  if(!ehOwner()) saldoAberto = -1;
  const fr = document.getElementById('fmReal');
  if(fr && confRasc) confRasc.val = fr.value;
  const sI = document.getElementById('sInicial');
  if(sI && saldoAberto>=0){
    rascunhoSaldo = {idx:saldoAberto, si:sI.value, di:document.getElementById('sData').value, entra:document.getElementById('sEntra').checked};
  }
  if(!listas.banco.length){
    el.innerHTML = '<p class="muted">Cadastre bancos na aba Listas primeiro.</p>';
    document.getElementById('saldoTotal').textContent = '···';
    document.getElementById('saldoSub').textContent = '';
    return;
  }
  const tot = totalSaldos();
  const te = document.getElementById('saldoTotal');
  te.textContent = tot.n ? fmtMoeda(tot.t) : '···';
  te.className = (tot.n && !saldosCarregando && tot.t<0) ? 'sd-val neg' : '';
  te.style.fontSize = '32px';
  const hora = saldosUltimaCalc ? new Date(saldosUltimaCalc).toLocaleTimeString('pt-BR',{hour:'2-digit',minute:'2-digit'}) : '';
  document.getElementById('saldoSub').textContent = saldosCarregando ? 'Calculando...' :
    `${listas.banco.length} banco${listas.banco.length>1?'s':''}${tot.fora?` · ${tot.fora} fora do total`:''}${tot.incompleto?' · ⚠ confira':''}${hora?` · atualizado ${hora}`:''}`;
  let h = '';
  if(layoutSaldos==='tiles'){
    h += '<div class="sd-tiles">' + indicesOrdenados().map(i=>{
      const b = listas.banco[i];
      const cfg = saldosConfig[b], r = saldosAtuais[b];
      return `<div class="sd-tile ${saldoAberto===i?'aberto':''} ${cfg&&cfg.foraDoTotal?'sd-fora':''}" data-sd="${i}"><div class="sd-nome">${esc(b)}${cfg&&cfg.foraDoTotal?' · fora':''}</div><div class="${classeVal(r)}">${textoVal(r)}</div></div>`;
    }).join('') + '</div>';
    if(saldoAberto>=0 && listas.banco[saldoAberto]!==undefined) h += painelSaldoHtml(saldoAberto);
  } else {
    h += indicesOrdenados().map(i=>{
      const b = listas.banco[i];
      const cfg = saldosConfig[b], r = saldosAtuais[b];
      const dica = !cfg ? 'defina o saldo inicial' : (cfg.foraDoTotal ? 'fora do total' : '');
      return `<div class="sd-row ${cfg&&cfg.foraDoTotal?'sd-fora':''}" data-sd="${i}"><div class="sd-nome">${esc(b)}${dica?`<div class="sd-sub">${dica}</div>`:''}</div><div class="${classeVal(r)}">${textoVal(r)}</div><span class="muted">${saldoAberto===i?'▴':'▾'}</span></div>` +
        (saldoAberto===i ? painelSaldoHtml(i) : '');
    }).join('');
  }
  el.innerHTML = h;
  if(ehOwner()) el.querySelectorAll('[data-sd]').forEach(x=>x.addEventListener('click', ()=>{
    const i = parseInt(x.dataset.sd,10);
    saldoAberto = (saldoAberto===i) ? -1 : i;
    rascunhoSaldo = null; msgSaldo = '';
    renderSaldos();
  }));
  const bs = document.getElementById('btnSalvarSaldo');
  if(bs) bs.addEventListener('click', ()=>salvarConfigSaldo(parseInt(bs.dataset.i,10)));
  el.querySelectorAll('[data-fm]').forEach(x=>x.addEventListener('click', ()=>acaoFechamento(x.dataset.fm, x.dataset.banco, x.dataset.mes)));
  const ci = document.getElementById('fmReal');
  if(ci){
    ci.addEventListener('input', ()=>{ confRasc.val = ci.value; });
  }
  const fo = document.getElementById('fmOk');
  if(fo) fo.addEventListener('click', conferirMesFluxo);
}
async function salvarConfigSaldo(i){
  if(!db) return;
  const banco = listas.banco[i];
  if(banco===undefined) return;
  const saldoInicial = parseFloat(document.getElementById('sInicial').value) || 0;
  const dataInicioTexto = document.getElementById('sData').value.trim();
  const entra = document.getElementById('sEntra').checked;
  const msgEl = document.getElementById('sMsg');
  let dataInicio = '';
  if(dataInicioTexto){
    const m = dataInicioTexto.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/);
    if(!m){ msgEl.textContent = '⚠️ Data inválida. Digite exatamente no formato AAAA-MM-DD, ex: 2026-09-01.'; return; }
    const [_, ano, mes, dia] = m;
    dataInicio = `${ano}-${mes.padStart(2,'0')}-${dia.padStart(2,'0')}`;
  }
  try{
    msgEl.textContent = 'Salvando...';
    const ref = db.doc("config/saldos");
    // lê o que REALMENTE está no banco e mescla só este banco
    const atual = await ref.get();
    const porBancoAtual = clonar((atual.data() && atual.data().porBanco) || {});
    const nova = {saldoInicial, dataInicio};
    if(!entra) nova.foraDoTotal = true;
    await ref.set({porBanco: {...porBancoAtual, [banco]: nova}});
    // confere lendo de volta
    const conferido = await ref.get();
    const g = conferido.data() && conferido.data().porBanco && conferido.data().porBanco[banco];
    if(!g || g.dataInicio!==dataInicio || g.saldoInicial!==saldoInicial || !!g.foraDoTotal!==!entra){
      msgEl.textContent = '⚠️ O banco de dados NÃO guardou o valor enviado. Enviado: ' + JSON.stringify(nova) + ' | Guardado: ' + JSON.stringify(g);
      return;
    }
    saldosConfig = clonar(conferido.data().porBanco);
    rascunhoSaldo = null;
    msgSaldo = 'Salvo ✓';
    await carregarSaldos(true);
  }catch(e){
    msgEl.textContent = '⚠️ Erro ao salvar: ' + (e && (e.code||'')) + ' ' + (e && (e.message||e));
  }
}
// ===== FECHAMENTO MENSAL =====
// O servidor calcula o saldo de cada meio (cascata mensal desde o início do controle) e guarda a conferência com o extrato.
const FUSO_APP = 'America/Sao_Paulo';
const fmtMesSP = new Intl.DateTimeFormat('en-CA', {timeZone:FUSO_APP, year:'numeric', month:'2-digit'});
function mesDeISO(iso){
  const p = fmtMesSP.formatToParts(new Date(iso));
  return p.find(x=>x.type==='year').value + '-' + p.find(x=>x.type==='month').value;
}
function somaMes(ym, n){
  const [y,m] = ym.split('-').map(Number);
  const t = y*12 + (m-1) + n;
  return Math.floor(t/12) + '-' + String(t%12+1).padStart(2,'0');
}
function rotuloMesCurto(ym){ const [y,m] = ym.split('-'); return MESES_NOME[+m-1].slice(0,3) + '/' + y.slice(2); }

let saldosTotal = null; // o total que o servidor calculou (GET /saldos)
let mesAtualSrv = '';
let confRasc = null;   // conferência em andamento { banco, mes, val, msg }
let fechFalhou = false; // a leitura de GET /saldos falhou
let fechAberto = false; // a lista de meses começa recolhida, como as tags em Lançar
function mesAtualApp(){ return mesAtualSrv || mesDeISO(new Date().toISOString()); }
function ultimoConf(r){ return ((r && r.historico) || []).filter(x=>x.estado==='conferido').map(x=>x.mes).sort().pop() || ''; }

// O servidor calcula o saldo, a cascata dos meses e o fechamento (GET /saldos); aqui só se mostra o que ele mandou.
async function carregarSaldos(forcar){
  if(!db || saldosCarregando) return;
  if(!forcar && Date.now()-saldosUltimaCalc<30000) return;
  saldosCarregando = true; fechFalhou = false; renderSaldos();
  try{
    const r = await db.saldos();
    if(r.mesAtual) mesAtualSrv = r.mesAtual;
    saldosTotal = r.total || null;
    const novos = {};
    (r.meios||[]).forEach(m=>{
      novos[m.meio] = m.erro ? {saldo:0, erro:true} : {saldo:m.saldo, historico:(m.meses||[]).map(x=>({mes:x.mes, valor:x.valor, conf:x.conferido||null, estado:x.estado}))};
    });
    saldosAtuais = novos;
    saldosUltimaCalc = Date.now();
  }catch(e){
    console.error('Falha ao ler os saldos do servidor', e);
    fechFalhou = true;
  } finally {
    saldosCarregando = false; renderSaldos();
  }
}
function recalcularTodosSaldos(){ return carregarSaldos(true); }
document.getElementById('btnRecalcularSaldos').addEventListener('click', recalcularTodosSaldos);

// ----- conferência com o extrato -----
function blocoFechamentoHtml(banco){
  if(fechFalhou) return '<div class="fm"><b>Fechamento mensal</b><p class="muted" style="margin:6px 0;">Indisponível agora: não consegui ler os saldos do servidor. Toque em Atualizar para tentar de novo.</p></div>';
  const r = saldosAtuais[banco];
  if(!r || !r.historico || !r.historico.length) return '';
  const ult = ultimoConf(r);
  const pend = r.historico.filter(x=>x.estado==='calculado').length;
  const aberto = fechAberto || (confRasc && confRasc.banco===banco); // conferindo: não esconde o painel
  const resumo = pend ? ` · <span class="muted">${pend} sem conferir</span>` : '';
  const alternar = `<span data-fm="alternar" style="cursor:pointer;color:var(--accent);font-size:13px;">${aberto?'Recolher ▴':'Ver meses ▾'}</span>`;
  let h = `<div class="fm"><div style="display:flex;justify-content:space-between;align-items:center;gap:8px;"><span><b>Fechamento mensal</b>${resumo}</span>${alternar}</div>`;
  if(!aberto) return h + '</div>';
  h += '<p class="muted" style="margin:6px 0;">Cada mês fecha sozinho. Confira com o extrato do banco: o mês conferido (e os anteriores) fica travado.</p>';
  r.historico.forEach(x=>{
    const extra = x.conf && x.conf.ajuste ? ` · ajuste ${fmtMoeda(x.conf.ajuste)}` : '';
    let acao = '';
    if(x.estado==='calculado' || x.estado==='aberto') acao = `<button class="ghost" data-fm="abrir" data-banco="${esc(banco)}" data-mes="${x.mes}">Conferir</button>`;
    else if(x.mes===ult) acao = `<button class="ghost" data-fm="reabrir" data-banco="${esc(banco)}" data-mes="${x.mes}">Reabrir</button>`;
    h += `<div class="fm-lin"><span class="fm-mes">${rotuloMesCurto(x.mes)}</span><span class="fm-pill fm-${x.estado}">${x.estado}</span><span class="fm-val">${fmtMoeda(x.valor)}<span class="muted" style="font-size:11px;">${extra}</span></span>${acao}</div>`;
    if(confRasc && confRasc.banco===banco && confRasc.mes===x.mes){
      h += `<div class="fm-conf"><label>Saldo no extrato do banco ao fim de ${rotuloMesCurto(x.mes)}</label>
        <input type="number" step="0.01" id="fmReal" value="${esc(confRasc.val)}" placeholder="0,00">
        <p class="muted" style="margin:4px 0;">O sistema calculou ${fmtMoeda(x.valor)}.</p>
        <p class="muted" style="margin:4px 0;color:var(--expense);">${esc(confRasc.msg||'')}</p>
        <div style="display:flex;gap:8px;"><button class="primary" id="fmOk">Conferir</button><button class="ghost" data-fm="cancelar" data-banco="${esc(banco)}" data-mes="${x.mes}">Cancelar</button></div></div>`;
    }
  });
  return h + '</div>';
}
async function conferirMesFluxo(){
  const c = confRasc;
  if(!c) return;
  const real = parseFloat(String(c.val).replace(',','.'));
  if(!Number.isFinite(real)){ c.msg = 'Digite o saldo do extrato primeiro'; renderSaldos(); return; }
  await comBotaoOcupado(document.getElementById('fmOk'), 'Conferindo…', async ()=>{
    try{
      // o servidor compara o extrato com o saldo que ele calculou; se não bater, devolve 409 com os números e a pessoa decide pelo ajuste
      let resp = await db.conferirMes({meio:c.banco, mes:c.mes, real});
      if(resp.status===409 && resp.body && resp.body.codigo==='diferenca'){
        const d = resp.body;
        const ok = await confirmarModal(`O sistema calculou ${fmtMoeda(d.calculado)} e o extrato diz ${fmtMoeda(d.real)} (diferença de ${fmtMoeda(d.diferenca)}). Criar um lançamento de ajuste de ${fmtMoeda(Math.abs(d.diferenca))} em ${rotuloMesCurto(c.mes)} para o saldo bater?`);
        if(!ok){ c.msg = 'O mês segue sem conferir. Revise os lançamentos e tente de novo.'; return; }
        resp = await db.conferirMes({meio:c.banco, mes:c.mes, real, ajustar:true});
      }
      if(resp.status!==200){ c.msg = (resp.body && resp.body.erro) || 'Não foi possível conferir. Tente de novo.'; return; }
      confRasc = null;
      toast('✓ ' + rotuloMesCurto(c.mes) + ' conferido: ' + fmtMoeda(real));
      await carregarSaldos(true);
    }catch(e){ console.error(e); c.msg = e && e.message ? e.message : 'Não foi possível conferir. Tente de novo.'; }
    finally{ renderSaldos(); }
  });
}
async function reabrirMesFluxo(banco, mes){
  if(!(await confirmarModal(`Reabrir ${rotuloMesCurto(mes)} de ${banco}? Os lançamentos de ${rotuloMesCurto(mes)} e dos meses anteriores voltam a poder ser alterados.`))) return;
  try{
    const resp = await db.reabrirMes(banco, mes);
    if(resp.status!==200){ await avisar((resp.body && resp.body.erro) || 'Não foi possível reabrir o mês.'); return; }
    toast('✓ ' + rotuloMesCurto(mes) + ' reaberto');
    await carregarSaldos(true);
  }catch(e){ console.error(e); await avisar('Não foi possível reabrir o mês. Tente de novo.'); }
}
function acaoFechamento(acao, banco, mes){
  if(acao==='alternar') fechAberto = !fechAberto;
  else if(acao==='abrir') confRasc = {banco, mes, val:'', msg:''};
  else if(acao==='cancelar') confRasc = null;
  else if(acao==='reabrir') return reabrirMesFluxo(banco, mes);
  renderSaldos();
}

// ===== CARTÃO: faturas calculadas pelo servidor =====
// O servidor calcula o período, a fatura de cada compra, o ajuste, o total, a conferência e as sugestões (GET /cartoes e
// GET /cartoes/{meio}/faturas/{ym}). Aqui só se mostra o que ele mandou e se manda o que a pessoa escolheu.
let cartoesResumo = null;  // resposta de GET /cartoes
let cartaoFatura = null;   // resposta da fatura do cartão aberto
let cartaoAbertoMeio = ''; // nome do cartão aberto
let cartaoYmSel = '';      // fatura pedida ('' = a atual)
let cartaoRecSel = null;
let cartaoCfgAberto = false, cartaoComprasAberto = true, cartaoPagAberto = true; // os blocos recolhiveis do painel do cartao
let cartaoCarregando = false;
let cartaoUltimaCalc = 0;
let cartaoErro = '';
let msgCartao = '';
let rascunhoFech = null;
const MESES_ABREV = ['jan','fev','mar','abr','mai','jun','jul','ago','set','out','nov','dez'];
function pad2(n){ return String(n).padStart(2,'0'); }
function diasNoMes(ym){ const [y,m]=ym.split('-').map(Number); return new Date(y,m,0).getDate(); }
function fmtDiaMes(k){ return k.slice(8,10)+'/'+k.slice(5,7); }
function fmtDataCompleta(k){ return k.slice(8,10)+'/'+k.slice(5,7)+'/'+k.slice(0,4); }
function textoDias(n){ return n===0 ? 'hoje' : 'em '+n+' dia'+(Math.abs(n)===1?'':'s'); }

// os meios que são cartão, para o Lançar: o servidor diz quais são (a lista chega com o GET /cfg/listas; aqui só se atualiza depois de mexer nos cartões)
function marcarMeiosCartao(resumo){
  const cartoes = new Set(((resumo && resumo.cartoes) || []).map(c=>c.meio));
  meiosLista = listas.banco.map(n=>({nome:n, cartao:cartoes.has(n), ativo:true}));
  atualizarCampoFatura();
  renderMarcaFatura();
}

async function carregarFatura(meio, ym){
  cartaoFatura = await db.cartaoFatura(meio, ym || '');
}
async function carregarCartoes(forcar){
  if(!db || cartaoCarregando) return;
  if(!forcar && cartoesResumo && Date.now()-cartaoUltimaCalc<30000){ renderCartao(); return; }
  cartaoCarregando = true; cartaoErro = ''; renderCartao();
  try{
    cartoesResumo = await db.cartoes();
    marcarMeiosCartao(cartoesResumo);
    cartaoUltimaCalc = Date.now();
    if(cartaoAbertoMeio && !cartoesResumo.cartoes.some(c=>c.meio===cartaoAbertoMeio)){ cartaoAbertoMeio = ''; cartaoFatura = null; }
    if(cartaoAbertoMeio) await carregarFatura(cartaoAbertoMeio, cartaoYmSel);
  }catch(e){
    console.error('Falha ao ler os cartões do servidor', e);
    cartaoErro = 'Não consegui ler os cartões. Toque em Atualizar para tentar de novo.';
  } finally { cartaoCarregando = false; renderCartao(); }
}
// pede de novo ao servidor depois de um comando; a tela mostra o que ele devolver
async function recarregarCartao(){ cartaoUltimaCalc = 0; await carregarCartoes(true); }
async function ajustarFatura(id, n){
  await db.collection("lancamentos").doc(id).update({faturaAjuste:n});
  const c = lancamentosCache.find(x=>x.id===id); if(c) c.faturaAjuste = n;
  aplicarLocal(id, {faturaAjuste:n});
}

function conferenciaHtml(f){
  if(f.valorBanco===null) return '<p class="muted">Digite o valor do app do cartão e salve para conferir.</p>';
  const d = f.diferenca;
  if(d===0) return '<p style="color:var(--income);font-weight:600;margin:6px 0;">Bateu ✓</p>';
  let h = `<p style="color:var(--expense);font-weight:600;margin:6px 0;">Diferença de ${fmtMoeda(Math.abs(d))} (${d>0?'o calculado é maior':'o calculado é menor'})</p>`;
  const nEst = f.lancamentos.filter(r=>r.fatura===f.ym && r.estimado).length;
  if(nEst) h += `<p class="muted">Há ${nEst} compra${nEst>1?'s':''} com valor estimado. Corrija ${nEst>1?'os valores':'o valor'} abaixo antes de procurar outras causas.</p>`;
  if(!f.fechada) return h + '<p class="muted">Fatura ainda aberta: a conferência fica mais confiável depois do fechamento.</p>';
  if(f.sugestoes.length){
    h += '<div class="muted" style="margin-top:6px;">Registros perto do corte que explicam a diferença:</div>';
    h += f.sugestoes.map((c,i)=>`<div class="sd-painel" style="margin:6px 0;padding:8px;"><div style="font-size:13px;">${c.lancamentos.map(r=>`${esc(r.descricao)} <span class="muted">${fmtDiaMes(r.dia)} · ${fmtMoeda(r.valor)}</span>`).join(' + ')}</div><button class="ghost" data-cand="${i}" style="margin-top:6px;font-size:12px;padding:4px 10px;">${esc(c.rotulo)}</button></div>`).join('');
  } else if(f.semSugestao){
    h += `<p class="muted">Nenhum registro perto do corte bate com ${fmtMoeda(Math.abs(d))}. ${esc(f.semSugestao)}</p>`;
    if(f.podeRegistrarDiferenca) h += `<button class="ghost" data-regdif="${f.ym}" style="font-size:12px;padding:4px 10px;">Registrar a diferença como lançamento</button>`;
  }
  return h;
}

// as opcoes de "Paga com": nenhum e os meios da lista, menos o proprio cartao
function opcoesPagador(cartao, atual){
  return '<option value="">(nenhum)</option>' + listas.banco.map(x=>`<option value="${esc(x)}" ${x===atual?'selected':''}>${esc(x)}</option>`).join('');
}
function linhaCfg(chave, rotulo, valor, ym){
  return `<button type="button" class="cfg-lin" data-ed="${chave}" data-ym="${ym||''}"><span>${rotulo}</span><span class="v">${esc(valor)} ›</span></button>`;
}
function tituloRecolhivel(chave, texto, aberto){
  return `<button type="button" class="cfg-tit" data-tg="${chave}" aria-expanded="${aberto}">${texto} ${aberto?'▴':'▾'}</button>`;
}
function painelCartaoHtml(){
  const f = cartaoFatura;
  if(!f) return '<div class="sd-painel"><p class="muted">Carregando faturas...</p></div>';
  const b = f.meio, ym = f.ym, per = f.periodo;
  const dono = ehOwner();
  const conferida = f.conferida;
  const cfg = ((cartoesResumo && cartoesResumo.cartoes) || []).find(c=>c.meio===b) || {};
  let h = '<div class="sd-painel">';
  // 1. o cartao: so para o dono, recolhido
  if(dono){
    h += tituloRecolhivel('cfg', 'Configurar cartão', cartaoCfgAberto);
    if(cartaoCfgAberto){
      h += '<div class="cfg-grupo">'
        + linhaCfg('inicio', 'Início do controle', f.inicio || 'sem limite', '')
        + linhaCfg('pagadorCartao', 'Paga com (padrão)', cfg.pagador || '(nenhum)', '')
        + linhaCfg('vencDia', 'Vence dia', cfg.diaVencimento==null ? '(nenhum)' : String(cfg.diaVencimento), '')
        + linhaCfg('avisoPadrao', 'Aviso (padrão)', cfg.aviso ? Recorrencia.fraseAviso({on:true, dias:cfg.aviso.dias, insistir:cfg.aviso.insistir}) : '(nenhum)', '')
        + `<div style="margin-top:8px;"><button class="ghost" data-rm="${esc(b)}" style="font-size:12px;">Remover este cartão</button></div></div>`;
    }
  }
  // 2. a fatura
  h += '<div class="tags" style="align-items:center;margin-top:10px;">'
    + `<span class="tag" data-nav="${f.anterior}" aria-label="Faturas mais antigas">‹</span>`
    + f.chips.map(c=>`<span class="tag ${c.ym===ym?'on':''}" data-chip="${c.ym}">${esc(c.rotulo)}</span>`).join('')
    + `<span class="tag" data-nav="${f.proxima}" aria-label="Faturas mais novas">›</span></div>`;
  h += `<p class="muted" style="margin:8px 0 4px;">Período: ${fmtDataCompleta(per.ini)} a ${fmtDataCompleta(per.fim)} · ${f.fechada?'fechada':'aberta'}</p>`;
  if(f.historico) h += `<p class="muted">📁 Histórico: antes do início do controle deste cartão (${esc(f.inicio||'')}). Os lançamentos aparecem, mas a fatura não é conferida.</p>`;
  h += `<p class="muted" id="cMsg">${esc(msgCartao||'')}</p>`;
  // 3. as compras (recolhivel)
  h += tituloRecolhivel('compras', 'Compras', cartaoComprasAberto);
  if(cartaoComprasAberto){
    if(!f.lancamentos.length) h += '<p class="muted">Nenhuma compra nesta fatura.</p>';
    f.lancamentos.forEach(r=>{
    const fora = r.origem==='foi';
    let tag = '';
    if(r.origem==='foi') tag = `<span class="fx-tag">foi para a ${r.ocupaOutraFatura>ym?'próxima':'anterior'} fatura</span>`;
    else if(r.origem==='veio') tag = `<span class="fx-tag">veio da ${r.ocupaOutraFatura<ym?'anterior':'próxima'} fatura</span>`;
    const estTag = r.estimado ? '<span class="fx-tag" style="margin-left:6px;">valor estimado</span>' : '';
    h += `<div class="fx-rec ${fora?'fora':''}" data-rec="${esc(r.id)}"><div class="fx-lin"><span class="fx-d">${fmtDiaMes(r.dia)}</span><span class="fx-t" style="${fora?'text-decoration:line-through;':''}">${esc(r.descricao)}</span><span class="fx-v" style="${fora?'text-decoration:line-through;':''}">${fmtMoeda(r.valor)}</span></div>${(tag||estTag)?'<div>'+tag+estTag+'</div>':''}`;
    if(cartaoRecSel===r.id && r.podeMexer){
      const ops = [[0,'Pela data (normal)'],[1,'Passar para a próxima fatura'],[-1,'Trazer para a anterior']].filter(o=>o[0]!==r.ajuste);
      h += '<div class="fx-acoes">' + ops.map(o=>`<button class="ghost" data-ajuste="${esc(r.id)}|${o[0]}" style="font-size:12px;padding:4px 10px;">${o[1]}</button>`).join('') + '</div>';
    }
    h += '</div>';
  });
  }
  // 4. o total e os dados da fatura
  h += `<div style="display:flex;justify-content:space-between;align-items:baseline;margin:10px 0 4px;"><span class="muted">Total da fatura</span><span class="sd-val">${fmtMoeda(f.total)}</span></div>`;
  if(!f.historico){
    const venc = f.vencimento ? fmtDataCompleta(f.vencimento) : '(nenhum)';
    const pag = f.pagador || '(nenhum)';
    if(dono){
      h += '<div class="cfg-grupo">'
        + linhaCfg('fechamento', 'Fechamento', fmtDataCompleta(per.fim), ym)
        + linhaCfg('vencimento', 'Vencimento', venc, ym)
        + linhaCfg('pagadorFatura', 'Paga com', pag, ym)
        + linhaCfg('valorBanco', 'Valor no app do cartão', f.valorBanco===null ? '(nenhum)' : fmtMoeda(f.valorBanco), ym) + '</div>';
    } else {
      h += `<p class="muted" style="margin:4px 0;">Fechamento ${fmtDataCompleta(per.fim)} · Vencimento ${esc(venc)} · Paga com ${esc(pag)}</p>`;
    }
  }
  if(!f.historico && !dono){
    if(f.valorBanco!==null) h += `<p class="muted" style="margin:4px 0;">Valor no app do cartão: ${fmtMoeda(f.valorBanco)}</p>`;
    if(conferida){
      h += f.mudouDesdeConferencia
        ? `<p style="color:var(--expense);font-weight:600;margin:6px 0;">Mudou desde a conferência (era ${fmtMoeda(conferida.total)})</p>`
        : `<p style="color:var(--income);font-weight:600;margin:6px 0;">🔒 Conferida em ${fmtDataCompleta(String(conferida.em).slice(0,10))}</p><p class="muted" style="margin:0 0 6px;">Travada: mudar valor, data, meio ou ajuste de uma compra desta fatura só depois de o dono desfazer a conferência.</p>`;
    }
  }
  if(!f.historico && dono){
    h += `<div id="cConf">${conferenciaHtml(f)}</div>`;
    if(conferida){
      const quando = fmtDataCompleta(String(conferida.em).slice(0,10));
      if(f.mudouDesdeConferencia){
        const dif = Math.round((f.total - conferida.total)*100)/100; // so a diferenca mostrada: o servidor ja disse que mudou
        h += `<div class="sd-painel" style="border-color:var(--expense);margin:6px 0;"><p style="color:var(--expense);font-weight:600;margin:0 0 4px;">Mudou desde a conferência</p><p class="muted" style="margin:0 0 6px;">Conferida em ${quando} com ${fmtMoeda(conferida.total)}; agora está ${fmtMoeda(f.total)} (${dif>0?'+':'-'}${fmtMoeda(Math.abs(dif))}). Algum lançamento foi alterado, ocultado ou importado depois.</p><button class="ghost" data-conferir="${ym}" style="font-size:12px;padding:4px 10px;">Aceitar o novo total</button> <button class="ghost" data-desconferir="${ym}" style="font-size:12px;padding:4px 10px;">Desfazer conferência</button></div>`;
      } else {
        h += `<p style="color:var(--income);font-weight:600;margin:6px 0;">🔒 Conferida em ${quando} (${fmtMoeda(conferida.total)}) <button class="ghost" data-desconferir="${ym}" style="font-size:11px;padding:2px 8px;margin-left:6px;">Desfazer</button></p>`;
      }
    } else if(f.fechada){
      h += `<button class="ghost" data-conferir="${ym}" style="font-size:12px;padding:4px 10px;margin:4px 0;">Marcar como conferida</button>`;
    }
  }
  const est = f.lancamentos.filter(r=>r.fatura===ym && r.estimado);
  if(est.length && dono){
    h += '<div style="margin:8px 0;"><div class="muted">Valores estimados (corrija com o valor real em reais):</div>' + est.map(r=>`<div class="fx-lin" style="padding:4px 0;"><span class="fx-d">${fmtDiaMes(r.dia)}</span><span class="fx-t">${esc(r.descricao)}</span><input type="number" step="0.01" id="est-${esc(r.id)}" value="${Math.abs(r.valor)}" style="flex:0 0 96px;width:96px;"><button class="ghost" data-est="${esc(r.id)}" style="padding:4px 10px;">Corrigir</button></div>`).join('') + '</div>';
  }
  // 5. os pagamentos marcados (recolhivel)
  if(f.pagamentos && f.pagamentos.length){
    h += tituloRecolhivel('pag', 'Pagamentos desta fatura', cartaoPagAberto);
    if(cartaoPagAberto){
      h += '<div style="margin:4px 0 10px;">' + f.pagamentos.map(p=>`<div class="fx-lin" style="padding:4px 0;"><span class="fx-d">${fmtDiaMes(p.dia)}</span><span class="fx-t">${esc(p.meio)} <span class="fx-tag">Fatura ${esc(rotuloMesCurto(ym))}</span></span><span class="fx-v">${fmtMoeda(p.valor)}</span></div>`).join('')
        + `<p class="muted" style="margin:4px 0;">Soma ${fmtMoeda(f.totalPago)}</p></div>`;
    }
  }
  return h + '</div>';
}
function renderCartao(){
  const el = document.getElementById('listaCartoes');
  if(!el) return;
  const r = cartoesResumo, lista = (r && r.cartoes) || [], n = lista.length;
  document.getElementById('cartaoTotal').textContent = (n && r) ? fmtMoeda(r.total) : '···';
  document.getElementById('cartaoSub').textContent = !r ? (cartaoErro || (cartaoCarregando ? 'Calculando...' : '')) :
    (!n ? 'Adicione um cartão abaixo' : (cartaoCarregando ? 'Calculando...' :
    `${n} ${n>1?'cartões':'cartão'} · próximo fechamento ${textoDias(r.proximoFechamentoEmDias)}`));
  let h = '';
  lista.forEach(c=>{
    const fat = c.fatura, mudou = c.conferidasQueMudaram;
    h += `<div class="sd-row" data-cartao="${esc(c.meio)}"><div class="sd-nome">${esc(c.meio)}<div class="sd-sub">fatura atual · fecha ${fmtDiaMes(fat.fechamento)} · ${textoDias(fat.diasAteFechar)}${mudou?` · <span style="color:var(--expense);">⚠ ${mudou} fatura${mudou>1?'s conferidas mudaram':' conferida mudou'}</span>`:''}</div></div><div class="sd-val ${cartaoCarregando?'carr':''}">${fmtMoeda(fat.total)}</div><span class="muted">${cartaoAbertoMeio===c.meio?'▴':'▾'}</span></div>`;
    if(cartaoAbertoMeio===c.meio) h += painelCartaoHtml();
  });
  if(!n) h = '<p class="muted" style="padding:6px 4px;">Nenhum cartão cadastrado ainda.</p>';
  el.innerHTML = h;
  // opções para adicionar cartão
  const sel = document.getElementById('ncBanco');
  if(sel){
    const antes = sel.value;
    const livres = (r && r.disponiveis) || [];
    sel.innerHTML = livres.map(x=>`<option value="${esc(x.meio)}">${esc(x.meio)}${x.temComprasNoCredito?' (tem compras no crédito)':''}</option>`).join('');
    if(livres.some(x=>x.meio===antes)) sel.value = antes;
  }
}

// Editar um campo do cartao ou da fatura: cada linha abre uma janelinha com o campo, Salvar, Limpar e Cancelar. O servidor valida e a tela mostra o que ele responder.
const CAMPOS_CARTAO = {
  inicio:        {titulo:'Início do controle (AAAA-MM)', tipo:'texto'},
  pagadorCartao: {titulo:'Paga com (padrão do cartão)', tipo:'meio'},
  vencDia:       {titulo:'Vence dia (1 a 31)', tipo:'numero'},
  fechamento:    {titulo:'Fechamento desta fatura (AAAA-MM-DD)', tipo:'texto'},
  vencimento:    {titulo:'Vencimento desta fatura (AAAA-MM-DD)', tipo:'texto'},
  pagadorFatura: {titulo:'Paga com (só esta fatura)', tipo:'meio'},
  valorBanco:    {titulo:'Valor no app do cartão', tipo:'numero', passo:'0.01'}
};
function valorAtualDoCampo(chave){
  const f = cartaoFatura, cfg = ((cartoesResumo && cartoesResumo.cartoes) || []).find(c=>c.meio===f.meio) || {};
  switch(chave){
    case 'inicio': return f.inicio || '';
    case 'pagadorCartao': return cfg.pagador || '';
    case 'vencDia': return cfg.diaVencimento==null ? '' : String(cfg.diaVencimento);
    case 'fechamento': return f.periodo.fim;
    case 'vencimento': return f.vencimento || (f.preencher && f.preencher.vencimento) || ''; // o dia do cartao so sugere
    case 'pagadorFatura': return f.pagador || (f.preencher && f.preencher.pagador) || ''; // o padrao do cartao so sugere
    case 'valorBanco': return f.valorBanco===null ? '' : String(f.valorBanco);
  }
  return '';
}
function pedirCampoCartao(c, atual, meio){
  return new Promise(resolve=>{
    const campo = c.tipo==='meio'
      ? `<select id="edCampo">${opcoesPagador(meio, atual)}</select>`
      : `<input id="edCampo" type="${c.tipo==='numero'?'number':'text'}" ${c.passo?`step="${c.passo}"`:''} ${c.tipo==='texto'?'inputmode="numeric"':''} value="${esc(atual)}">`;
    document.getElementById('modalMsg').textContent = c.titulo;
    document.getElementById('modalExtra').innerHTML = campo;
    document.getElementById('modalBotoes').innerHTML = '<button class="ghost" id="modalCancelar">Cancelar</button><button class="ghost" id="edLimpar">Limpar</button><button class="primary" id="modalConfirmar">Salvar</button>';
    document.getElementById('modalOverlay').classList.remove('hidden');
    const fechar = (v)=>{ document.getElementById('modalOverlay').classList.add('hidden'); document.getElementById('modalExtra').innerHTML = ''; resolve(v); };
    document.getElementById('modalCancelar').onclick = ()=>fechar(undefined);
    document.getElementById('edLimpar').onclick = ()=>fechar(null);
    document.getElementById('modalConfirmar').onclick = ()=>{ const v = document.getElementById('edCampo').value.trim(); fechar(v===''?null:v); };
    const el = document.getElementById('edCampo'); if(el && el.focus) el.focus();
  });
}
async function editarCampoCartao(chave, ym, meio){
  if(chave === 'avisoPadrao') return abrirAvisoPadrao(meio);
  const c = CAMPOS_CARTAO[chave];
  if(!c || !cartaoFatura) return;
  const v = await pedirCampoCartao(c, valorAtualDoCampo(chave), meio);
  if(v===undefined) return; // cancelou
  try{
    if(chave==='inicio') await db.cartaoInicio(meio, v);
    else if(chave==='pagadorCartao') await db.cartaoConfig(meio, {pagador: v});
    else if(chave==='vencDia') await db.cartaoConfig(meio, {diaVencimento: v===null ? null : parseInt(v,10)});
    else if(chave==='fechamento') await db.cartaoFechamento(meio, ym, {fechamento: v});
    else if(chave==='vencimento') await db.cartaoFechamento(meio, ym, {vencimento: v});
    else if(chave==='pagadorFatura') await db.cartaoFechamento(meio, ym, {pagador: v});
    else if(chave==='valorBanco'){
      const n = v===null ? null : parseFloat(v);
      if(n!==null && (isNaN(n) || n<0)){ await avisar('Valor inválido.'); return; }
      await db.cartaoFechamento(meio, ym, {valorBanco: n});
    }
    msgCartao = 'Salvo ✓';
    await recarregarCartao();
  }catch(err){ await avisar(erroCartao(err)); }
}
function erroCartao(e){ return (e && (e.message || e.code)) || String(e); }
async function onCartaoClick(e){
  const el = e.target.closest('[data-cartao],[data-chip],[data-nav],[data-rec],[data-ajuste],[data-ed],[data-tg],[data-rm],[data-est],[data-cand],[data-regdif],[data-conferir],[data-desconferir]');
  if(!el) return;
  const meio = cartaoAbertoMeio;
  if(el.dataset.conferir || el.dataset.desconferir){
    const ym = el.dataset.conferir || el.dataset.desconferir;
    if(!meio) return;
    try{
      if(el.dataset.desconferir){
        await db.cartaoDesconferir(meio, ym);
      } else {
        // quem decide se bate e o servidor: com diferenca (ou sem o valor do app) ele responde 409 e a pessoa escolhe conferir mesmo assim
        const jaConferida = !!(cartaoFatura && cartaoFatura.conferida);
        let resp = await db.cartaoConferir(meio, ym, jaConferida);
        if(resp.status===409 && resp.body && resp.body.codigo==='diferenca'){
          const d = resp.body;
          const txt = d.valorBanco===null ? `Esta fatura não tem o valor do app do cartão para comparar. Marcar como conferida mesmo assim, com o total ${fmtMoeda(d.total)}?`
            : `O total calculado (${fmtMoeda(d.total)}) difere do valor informado do banco (${fmtMoeda(d.valorBanco)}). Marcar como conferida mesmo assim?`;
          if(!(await confirmarModal(txt))) return;
          resp = await db.cartaoConferir(meio, ym, true);
        }
        if(resp.status!==200){ await avisar((resp.body && resp.body.erro) || 'Não consegui conferir a fatura.'); return; }
      }
    }catch(err){ await avisar('Não consegui salvar: ' + erroCartao(err)); }
    await recarregarCartao(); return;
  }
  if(el.dataset.tg){
    const k = el.dataset.tg;
    if(k==='cfg') cartaoCfgAberto = !cartaoCfgAberto;
    else if(k==='compras') cartaoComprasAberto = !cartaoComprasAberto;
    else if(k==='pag') cartaoPagAberto = !cartaoPagAberto;
    renderCartao(); return;
  }
  if(el.dataset.ed){ await editarCampoCartao(el.dataset.ed, el.dataset.ym, meio); return; }
  if(el.dataset.est){
    const id = el.dataset.est;
    const n = parseFloat(document.getElementById('est-'+id).value);
    if(!n || n<=0){ await avisar('Informe um valor válido.'); return; }
    try{
      await db.collection("lancamentos").doc(id).update({valor:n, valorEstimado:false});
      const c = lancamentosCache.find(x=>x.id===id); if(c){ c.valor = n; c.valorEstimado = false; }
      aplicarLocal(id, {valor:n, valorEstimado:false});
    }catch(err){ await avisar('Não consegui corrigir o valor: ' + erroCartao(err)); }
    await recarregarCartao(); return;
  }
  if(el.dataset.cand!==undefined){
    const c = cartaoFatura && cartaoFatura.sugestoes[parseInt(el.dataset.cand,10)];
    if(!c) return;
    try{ for(const a of c.acoes){ await ajustarFatura(a.id, a.faturaAjuste); } }
    catch(err){ await avisar('Não consegui ajustar a fatura: ' + erroCartao(err)); }
    cartaoRecSel = null; await recarregarCartao(); return;
  }
  if(el.dataset.regdif){
    const ym = el.dataset.regdif, f = cartaoFatura;
    if(!f) return;
    if(!(await confirmarModal(`Registrar um lançamento de ${fmtMoeda(Math.abs(f.diferenca))} em ${meio}, em ${fmtDataCompleta(f.periodo.fim)} (fechamento)?`))) return;
    try{ await db.cartaoDiferenca(meio, ym); }
    catch(err){ await avisar('Não consegui registrar: ' + erroCartao(err)); }
    await recarregarCartao(); return;
  }
  if(el.dataset.ajuste){
    const [id, n] = el.dataset.ajuste.split('|');
    cartaoRecSel = null;
    try{ await ajustarFatura(id, parseInt(n,10)); }catch(err){ await avisar('Não consegui ajustar a fatura: ' + erroCartao(err)); }
    await recarregarCartao(); return;
  }
  if(el.dataset.rec){ cartaoRecSel = (cartaoRecSel===el.dataset.rec) ? null : el.dataset.rec; renderCartao(); return; }
  if(el.dataset.chip!==undefined || el.dataset.nav!==undefined){
    // o servidor mandou o mes de cada chip e dos botões ‹ ›: aqui só se pede a fatura escolhida
    cartaoYmSel = el.dataset.chip || el.dataset.nav;
    cartaoRecSel = null; msgCartao = ''; rascunhoFech = null;
    try{ cartaoCarregando = true; renderCartao(); await carregarFatura(meio, cartaoYmSel); }
    catch(err){ await avisar('Não consegui ler a fatura: ' + erroCartao(err)); }
    finally{ cartaoCarregando = false; renderCartao(); }
    return;
  }
  if(el.dataset.cartao!==undefined){
    const nome = el.dataset.cartao;
    cartaoRecSel = null; msgCartao = ''; rascunhoFech = null; cartaoYmSel = '';
    if(cartaoAbertoMeio===nome){ cartaoAbertoMeio = ''; cartaoFatura = null; renderCartao(); return; }
    cartaoAbertoMeio = nome; cartaoFatura = null; renderCartao();
    try{ await carregarFatura(nome, ''); }
    catch(err){ await avisar('Não consegui ler a fatura: ' + erroCartao(err)); cartaoAbertoMeio = ''; }
    renderCartao();
    return;
  }
  if(el.dataset.rm!==undefined){
    const b = el.dataset.rm;
    if(!b) return;
    if(!(await confirmarModal('Remover ' + b + ' da lista de cartões? Os lançamentos não são apagados, só as datas de fechamento configuradas.'))) return;
    try{
      await db.cartaoRemover(b);
      cartaoAbertoMeio = ''; cartaoFatura = null;
      await recarregarCartao();
    }catch(err){ await avisar('Não consegui remover: ' + erroCartao(err)); }
  }
}
async function adicionarCartaoUI(){
  const banco = document.getElementById('ncBanco').value;
  const dia = parseInt(document.getElementById('ncDia').value,10);
  const msgEl = document.getElementById('ncMsg');
  if(!banco){ msgEl.textContent = '⚠️ Escolha um banco ou conta.'; return; }
  if(!dia || dia<1 || dia>31){ msgEl.textContent = '⚠️ Informe o dia do fechamento (1 a 31).'; return; }
  try{
    msgEl.textContent = 'Salvando...';
    await db.cartaoAdicionar(banco, dia);
    document.getElementById('ncDia').value = '';
    msgEl.textContent = 'Cartão adicionado ✓';
    await recarregarCartao();
  }catch(err){ msgEl.textContent = '⚠️ Erro ao salvar: ' + erroCartao(err); }
}

function normalizaCabecalho(s){
  return String(s).trim().toUpperCase().normalize('NFD').replace(/[\u0300-\u036f]/g,'');
}
function parseValorCell(v){
  if(v===undefined||v===null||v==='') return null;
  if(typeof v === 'number') return isNaN(v) ? null : v;
  let s = String(v).trim();
  if(!s) return null;
  if(/,\d{1,2}$/.test(s)) s = s.replace(/\./g,'').replace(',','.');
  const n = parseFloat(s);
  return isNaN(n) ? null : n;
}
function parseDataCell(v){
  if(v===undefined||v===null||v==='') return null;
  if(v instanceof Date) return isNaN(v.getTime()) ? null : v;
  if(typeof v === 'number'){
    const d = XLSX.SSF.parse_date_code(v);
    if(d) return new Date(d.y, d.m-1, d.d, d.H||0, d.M||0, d.S||0);
    return null;
  }
  const s = String(v).trim();
  if(!s) return null;
  // formato brasileiro dd/mm/aaaa[ hh:mm[:ss]]
  const m = s.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})(?:[ T](\d{1,2}):(\d{2})(?::(\d{2}))?)?$/);
  if(m){
    const [,dd,mm,yyyy,hh,mi,ss] = m;
    return new Date(+yyyy, +mm-1, +dd, +(hh||0), +(mi||0), +(ss||0));
  }
  const d2 = new Date(s);
  return isNaN(d2.getTime()) ? null : d2;
}

const CABECALHOS = {
  status: ['ST'], descricao: ['DESCRICAO'], dataHora: ['DATA HORA'], pago: ['PAGO'], // sem-status:ok (nome da coluna da planilha)
  renda: ['RENDA'], gastos: ['GASTOS'], banco: ['BANCO'], tag: ['TAG']
};
function mapaColunas(linhaCabecalho){
  const norm = linhaCabecalho.map(c=>c!=null?normalizaCabecalho(c):'');
  const mapa = {};
  Object.keys(CABECALHOS).forEach(chave=>{
    const idx = norm.findIndex(c=>CABECALHOS[chave].includes(c));
    mapa[chave] = idx;
  });
  return mapa;
}

function escHtml(t){ return String(t==null?'':t).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;'); }
function esc(t){ return escHtml(t); }
// Notas/comentários de célula (Google Sheets exporta as "notas" como comentários no .xlsx)
function limparNotaComentario(c){
  const partes = (c||[]).map(x=>{
    let t = String(x && x.t || '').replace(/\r\n?/g,'\n');
    if(/^\[Threaded comment\]/i.test(t.trim())){
      const m = t.match(/Comment:\s*\n?([\s\S]*)$/i);
      t = m ? m[1] : t;
    }
    return t.trim();
  }).filter(Boolean);
  return partes.join('\n').trim();
}
function notasDaAba(ws){
  const ref = ws && ws['!ref'];
  if(!ref) return ()=>'';
  const r = XLSX.utils.decode_range(ref);
  return (i,j)=>{
    if(j==null || j<0) return '';
    const cel = ws[XLSX.utils.encode_cell({r:r.s.r+i, c:r.s.c+j})];
    return (cel && cel.c) ? limparNotaComentario(cel.c) : '';
  };
}

function parseLinhas(linhas, notaDe){
  let headerIdx = -1, mapa = null;
  for(let i=0;i<linhas.length;i++){
    const linha = linhas[i];
    if(!linha) continue;
    const norm = linha.map(c=>c!=null?normalizaCabecalho(c):'');
    if(norm.includes('ST') && norm.includes('DESCRICAO')){ headerIdx = i; mapa = mapaColunas(linha); break; }
  }
  if(headerIdx===-1) return {erro:'cabecalho'};
  if(mapa.dataHora===-1 || (mapa.renda===-1 && mapa.gastos===-1)) return {erro:'colunas', mapa};

  let resultado = [];
  let ignoradas = 0;
  for(let i=headerIdx+1;i<linhas.length;i++){
    const r = linhas[i];
    if(!r) continue;
    const status = mapa.status>=0 && r[mapa.status]!=null ? String(r[mapa.status]).trim() : '';
    if(!status || status.toUpperCase()==='NAN') continue;
    const desc = mapa.descricao>=0 && r[mapa.descricao]!=null ? String(r[mapa.descricao]).trim() : '(sem descrição)';
    let dataEv = mapa.dataHora>=0 ? parseDataCell(r[mapa.dataHora]) : null;
    if(!dataEv && mapa.pago>=0) dataEv = parseDataCell(r[mapa.pago]);
    const renda = mapa.renda>=0 ? parseValorCell(r[mapa.renda]) : null;
    const gasto = mapa.gastos>=0 ? parseValorCell(r[mapa.gastos]) : null;
    const banco = mapa.banco>=0 && r[mapa.banco]!=null ? String(r[mapa.banco]).trim() : '';
    const tagCell = mapa.tag>=0 && r[mapa.tag]!=null ? String(r[mapa.tag]) : '';
    const tags = (tagCell.match(/#[^\s#]+/g)||[]).map(t=>t.slice(1));
    const valor = renda!=null ? renda : gasto;
    if(valor==null || !dataEv){ ignoradas++; continue; }
    const tipo = renda!=null ? 'entrada' : 'saida';
    resultado.push({
      status, tipo, valor, descricao: desc, nota: notaDe ? notaDe(i, mapa.descricao) : '', banco, tags,
      dataEvento: dataEv.toISOString()
    });
  }
  return {resultado, ignoradas};
}

// Verifica duplicidade consultando o BANCO DE DADOS diretamente (não o cache limitado
// da tela, que só guarda os 1000 lançamentos mais recentes e por isso "esquece" de
// meses antigos assim que o total passa disso).
// Chave igual (status, descrição, valor, banco, data e tipo) é o mesmo lançamento: o app não grava duplicados, então
// linhas iguais no arquivo, ou que já estão no banco, entram uma vez só.
async function deduplicarContraBanco(resultado){
  if(!db) return {resultado, duplicadas:0};
  const bancosNoLote = [...new Set(resultado.map(l=>l.banco||''))];
  const datas = resultado.map(l=>l.dataEvento).filter(Boolean).sort();
  const existentes = new Set();
  if(datas.length){
    // Um duplicado só pode existir dentro do período do lote (a chave inclui a data exata).
    const de = datas[0], ate = datas[datas.length-1];
    for(const banco of bancosNoLote){
      let snap;
      try{
        snap = await db.collection("lancamentos").where("banco","==",banco)
          .where("dataEvento",">=",de).where("dataEvento","<=",ate).limit(1000000).get();
      }catch(e){
        // sem conferir, a importação poderia duplicar: melhor parar do que seguir
        throw new Error(`não consegui conferir duplicidade em "${banco}" (${(e && e.message) || e})`);
      }
      snap.docs.forEach(d=>existentes.add(chaveVerif(d.data())));
    }
  }
  const vistos = new Set();
  let duplicadas = 0;
  const filtrado = resultado.filter(l=>{
    const k = chaveVerif(l);
    if(existentes.has(k) || vistos.has(k)){ duplicadas++; return false; }
    vistos.add(k);
    return true;
  });
  return {resultado: filtrado, duplicadas};
}
function chaveLancamento(l){
  return [l.status, (l.descricao||'').trim().toLowerCase(), Number(l.valor).toFixed(2), (l.banco||'').trim().toLowerCase(), l.dataEvento].join('|');
}

const MESES_PT = {janeiro:1,fevereiro:2,marco:3,abril:4,maio:5,junho:6,julho:7,agosto:8,setembro:9,outubro:10,novembro:11,dezembro:12};
function ymDaAba(nome){
  const m = normTxt(nome).match(/(janeiro|fevereiro|marco|abril|maio|junho|julho|agosto|setembro|outubro|novembro|dezembro)\D*(\d{4})/);
  return m ? m[2]+'-'+pad2(MESES_PT[m[1]]) : null;
}
// Em cada aba mensal da planilha, a linha do cartão tem o dia de fechamento na célula à esquerda de uma célula
// "FECHAMENTO" e o banco na coluna BANCO. Devolve { banco: { 'AAAA-MM': dia } }.
function extrairFechamentos(wb){
  const achados = {}, semMes = [];
  wb.SheetNames.forEach(nome=>{
    const linhas = XLSX.utils.sheet_to_json(wb.Sheets[nome], {header:1, raw:true, defval:null});
    let h = -1, mapa = null;
    for(let i=0;i<linhas.length;i++){
      const norm = (linhas[i]||[]).map(c=>c!=null?normalizaCabecalho(c):'');
      if(norm.includes('ST') && norm.includes('DESCRICAO')){ h = i; mapa = mapaColunas(linhas[i]); break; }
    }
    if(h<0 || mapa.banco<0) return;
    const ym = ymDaAba(nome);
    for(let i=h+1;i<linhas.length;i++){
      const r = linhas[i];
      if(!r) continue;
      const j = r.findIndex(c=>typeof c==='string' && /^\s*FECHAMENTO/i.test(c));
      if(j<1) continue;
      const dia = Number(r[j-1]);
      const banco = r[mapa.banco]!=null ? String(r[mapa.banco]).trim() : '';
      if(!banco || !Number.isInteger(dia) || dia<1 || dia>31) continue;
      if(!ym){ if(!semMes.includes(nome)) semMes.push(nome); continue; }
      (achados[banco] = achados[banco] || {})[ym] = dia;
    }
  });
  return {achados, semMes};
}
async function importarFechamentos(){
  const st = document.getElementById('fechStatus'), inp = document.getElementById('arquivoFech');
  if(!db){ st.textContent = 'Banco de dados indisponível.'; return; }
  if(!inp.files.length){ st.textContent = 'Escolha a planilha (.xlsx) primeiro.'; return; }
  st.textContent = 'Lendo...';
  try{
    await carregarScript(XLSX_URL);
    const wb = XLSX.read(new Uint8Array(await inp.files[0].arrayBuffer()), {type:'array'});
    const {achados, semMes} = extrairFechamentos(wb);
    const todos = Object.keys(achados);
    if(!todos.length){ st.textContent = 'Não encontrei linhas "FECHAMENTO" (o dia na célula à esquerda e o banco na coluna BANCO).'; return; }
    const bancos = todos.filter(b=>listas.banco.includes(b));
    const ignorados = todos.filter(b=>!listas.banco.includes(b));
    if(!bancos.length){ st.textContent = 'Os bancos da planilha (' + todos.join(', ') + ') não estão na lista de bancos do app.'; return; }
    const resumo = bancos.map(b=>`${b} (${Object.keys(achados[b]).length} meses)`).join(', ');
    if(!(await confirmarModal(`Gravar os dias de fechamento: ${resumo}?` + (ignorados.length ? ` Fora da lista de bancos, serão ignorados: ${ignorados.join(', ')}.` : '')))){ st.textContent = ''; return; }
    // cada fechamento vira um comando do servidor (que valida a data); o cartão que ainda não existe é criado com o dia do último mês da planilha
    const atuais = new Set(((await db.cartoes()).cartoes || []).map(c=>c.meio));
    let feitos = 0;
    const total = bancos.reduce((n,b)=>n+Object.keys(achados[b]).length, 0);
    for(const b of bancos){
      const yms = Object.keys(achados[b]).sort();
      if(!atuais.has(b)) await db.cartaoAdicionar(b, achados[b][yms[yms.length-1]]);
      for(const ym of yms){
        const dia = Math.min(achados[b][ym], diasNoMes(ym));
        await db.cartaoFechamento(b, ym, {fechamento: ym + '-' + pad2(dia)});
        st.textContent = `Gravando... ${++feitos} de ${total}`;
      }
    }
    st.textContent = `Fechamentos gravados: ${resumo}.` + (semMes.length ? ` Abas ignoradas (mês não identificado): ${semMes.join(', ')}.` : '');
    recarregarCartao();
  }catch(e){
    st.textContent = '⚠️ ' + ((e && (e.message||e.code)) || e);
  }
}

function tratarResultadoImport(res, statusEl){
  if(res.erro==='cabecalho'){ statusEl.textContent = "Não encontrei a linha de cabeçalho (ST/DESCRIÇÃO/...). Confira se colou a partir dessa linha."; return; }
  if(res.erro==='colunas'){ statusEl.textContent = "Encontrei o cabeçalho, mas faltou a coluna DATA HORA ou RENDA/GASTOS. Confira se colou todas as colunas."; return; }
  lancamentosParaImportar = res.resultado;
  mostrarPreview();
  const partes = [];
  const comNota = (res.resultado||[]).filter(l=>l.nota).length;
  if(comNota) partes.push(`${comNota} lançamento(s) com nota`);
  if(res.ignoradas) partes.push(`${res.ignoradas} linha(s) sem data/valor ignoradas`);
  if(res.duplicadas) partes.push(`${res.duplicadas} já existente(s) no app, não repetidas`);
  statusEl.textContent = partes.join(' · ');
}

async function processarArquivo(){
  const inp = document.getElementById('arquivoImport');
  const statusEl = document.getElementById('importStatus');
  if(!inp.files.length){ statusEl.textContent = "Escolha um arquivo primeiro."; return; }
  statusEl.textContent = "Lendo...";
  const arrayBuffer = await inp.files[0].arrayBuffer();
  try{
    await carregarScript(XLSX_URL);
    const wb = XLSX.read(new Uint8Array(arrayBuffer), {type:'array', cellDates:false});
    let resultadoFinal = [], ignoradasFinal = 0, achouAlguma = false;
    wb.SheetNames.forEach(nome=>{
      const linhas = XLSX.utils.sheet_to_json(wb.Sheets[nome], {header:1, raw:true, defval:null});
      const res = parseLinhas(linhas, notasDaAba(wb.Sheets[nome]));
      if(!res.erro){ achouAlguma = true; resultadoFinal = resultadoFinal.concat(res.resultado); ignoradasFinal += res.ignoradas; }
    });
    if(!achouAlguma){ tratarResultadoImport({erro:'cabecalho'}, statusEl); return; }
    guardarLoteArquivo(resultadoFinal, ignoradasFinal, false);
    statusEl.textContent = "Conferindo duplicidade no banco de dados...";
    const {resultado, duplicadas} = await deduplicarContraBanco(resultadoFinal);
    tratarResultadoImport({resultado, ignoradas: ignoradasFinal, duplicadas}, statusEl);
  }catch(err){
    statusEl.textContent = "Não consegui ler esse arquivo: " + err.message;
  }
}

function mostrarPreview(){
  document.getElementById('qtdPreview').textContent = lancamentosParaImportar.length;
  document.getElementById('listaPreview').innerHTML = lancamentosParaImportar.slice(0,300).map(l=>`
    <div class="item">
      <div class="item-top"><span class="item-desc">${esc(l.descricao)}</span>${htmlValorLinha(l)}</div>
      <div class="item-meta">${new Date(l.dataEvento).toLocaleDateString('pt-BR')} · ${esc(l.status)} · ${esc(l.banco||'-')}</div>
      ${l.nota?`<div class="item-meta">📝 ${escHtml(l.nota.length>140 ? l.nota.slice(0,140)+'…' : l.nota)}</div>`:''}
    </div>`).join('');
  document.getElementById('cardPreview').classList.remove('hidden');
}

function fmtDuracao(seg){ return seg < 60 ? seg + 's' : Math.floor(seg/60) + 'min ' + String(seg%60).padStart(2,'0') + 's'; }
function novoIdLinha(){
  const alfa = 'abcdefghijklmnopqrstuvwxyz0123456789';
  return Array.from(crypto.getRandomValues(new Uint8Array(20)), b=>alfa[b % alfa.length]).join('');
}
// Importa a planilha em lotes de 25 (endpoint de lote). Cada linha leva um id gerado aqui, então reenviar um lote
// nunca duplica. Se parar no meio, a prévia é descartada: processar o mesmo arquivo de novo importa só o que faltou.
async function confirmarImportacao(){
  const btn = document.getElementById('btnConfirmarImport');
  const statusEl = document.getElementById('importStatus');
  if(!db || !db.importarLancamentos){ await avisar("Importação em lote indisponível nesta visualização."); return; }
  const itens = lancamentosParaImportar, total = itens.length;
  btn.disabled = true;
  const inicio = Date.now(), est = {feitos:0, espera:0};
  const desenhar = ()=>{
    const pct = total ? Math.floor(est.feitos*100/total) : 100;
    btn.textContent = `Importando ${est.feitos}/${total}…`;
    statusEl.textContent = `Importando… ${est.feitos}/${total} (${pct}%) · ${fmtDuracao(Math.round((Date.now()-inicio)/1000))}`
      + (est.espera ? ` · aguardando o banco de dados liberar capacidade (tentativa ${est.espera})…` : '')
      + ' Não feche esta página.';
  };
  desenhar();
  const tique = setInterval(desenhar, 1000); // o tempo corre mesmo quando está esperando o banco
  window.onbeforeunload = e=>{ e.preventDefault(); e.returnValue = ''; };
  let r = null, erro = null;
  try{
    // listas e tags ANTES: se a importação parar no meio, o que já entrou não fica com banco, status ou tag desconhecidos
    const bancosNovos = new Set(listas.banco), statusNovos = new Set(listas.status), tagsNovas = new Set(tagsUsadas);
    itens.forEach(l=>{
      if(l.banco) bancosNovos.add(l.banco);
      if(l.status) statusNovos.add(l.status);
      (l.tags||[]).forEach(t=>tagsNovas.add(t));
    });
    if(bancosNovos.size!==listas.banco.length || statusNovos.size!==listas.status.length){
      listas.banco = [...bancosNovos]; listas.status = [...statusNovos];
      await salvarListas();
    }
    if(tagsNovas.size!==tagsUsadas.length) await db.doc("config/tags").set({usadas:[...tagsNovas]});
    const agora = new Date().toISOString();
    r = await db.importarLancamentos(itens.map(l=>({...l, id:novoIdLinha(), criadoEm:agora})), (feitos, tot, info)=>{
      est.feitos = feitos; est.espera = info && info.espera ? info.espera : 0; desenhar();
    });
  }catch(e){
    console.error(e);
    erro = e;
  }finally{
    clearInterval(tique);
    window.onbeforeunload = null;
    btn.disabled = false;
    btn.textContent = 'Confirmar importação';
  }
  const dur = fmtDuracao(Math.round((Date.now()-inicio)/1000));
  if(erro){
    const gravados = erro.gravados != null ? erro.gravados : 0;
    statusEl.textContent = `⚠️ A importação parou em ${est.feitos}/${total} (${gravados} gravados): ${(erro && erro.message) || erro}. Nada se perde nem duplica: processe o mesmo arquivo de novo e confirme, que só o que faltou é importado.`;
    toast('A importação parou. Veja a mensagem em Configurações › Importar.', true);
    document.getElementById('cardPreview').classList.add('hidden');
    lancamentosParaImportar = [];
  }else{
    statusEl.textContent = `Importação concluída: ${r.gravados} lançamento(s) em ${dur}.`
      + (r.rejeitados.length ? ` ${r.rejeitados.length} rejeitado(s): ` + r.rejeitados.slice(0,5).map(x=>`#${x.indice+1} ${x.motivo}`).join('; ') + (r.rejeitados.length>5?'…':'') + '.' : '');
    toast('✓ Importação concluída: ' + r.gravados + ' lançamentos');
    document.getElementById('cardPreview').classList.add('hidden');
    lancamentosParaImportar = [];
  }
  await verificarArquivoXBanco();
}

async function processarTextoColado(){
  const texto = document.getElementById('textoColado').value;
  const statusEl = document.getElementById('importStatus');
  if(!texto.trim()){ statusEl.textContent = "Cole os dados primeiro."; return; }
  try{
    const linhas = texto.split(/\r?\n/).map(l=>l.split('\t'));
    const res = parseLinhas(linhas);
    if(res.erro){ tratarResultadoImport(res, statusEl); return; }
    guardarLoteArquivo(res.resultado, res.ignoradas, false);
    statusEl.textContent = "Conferindo duplicidade no banco de dados...";
    const {resultado, duplicadas} = await deduplicarContraBanco(res.resultado);
    tratarResultadoImport({resultado, ignoradas: res.ignoradas, duplicadas}, statusEl);
  }catch(err){
    statusEl.textContent = "Não consegui interpretar o texto colado: " + err.message;
  }
}

async function lerTodosLancamentos(aoProgredir){
  const todos = new Map();
  let cursor = null;
  for(let i=0;i<500;i++){
    let q = db.collection("lancamentos").orderBy("dataEvento","desc");
    if(cursor) q = q.where("dataEvento","<=",cursor);
    const snap = await q.limit(1000).get();
    let novos = 0;
    snap.docs.forEach(d=>{
      if(!todos.has(d.id)){ todos.set(d.id, {id:d.id, ...clonar(d.data())}); novos++; }
    });
    if(aoProgredir) aoProgredir(todos.size);
    if(snap.docs.length<1000) break;
    if(novos===0) throw new Error('Mais de 1000 lançamentos com exatamente a mesma data e hora. A exportação ficaria incompleta e foi interrompida.');
    cursor = snap.docs[snap.docs.length-1].data().dataEvento;
  }
  return [...todos.values()];
}
async function exportarDados(){
  const statusEl = document.getElementById('exportStatus');
  const downloads = await claude.use("downloads");
  if(!downloads){ statusEl.textContent = "Exportação indisponível nesta visualização."; return; }
  if(!db){ statusEl.textContent = "Banco de dados indisponível nesta visualização."; return; }
  try{
    const comLanc = document.getElementById('expLanc').checked, comCfg = document.getElementById('expCfg').checked;
    if(!comLanc && !comCfg){ statusEl.textContent = "Marque ao menos uma opção para exportar."; return; }
    let lancamentos = [];
    if(comLanc){
      statusEl.textContent = "Lendo lançamentos...";
      lancamentos = await lerTodosLancamentos(n=>{ statusEl.textContent = `Lendo lançamentos... ${n}`; });
    }
    const documentos = {};
    if(comCfg){
      statusEl.textContent = "Lendo configurações...";
      for(const caminho of ["config/listas","config/tags","config/combos","config/saldos","config/cartoes"]){
        const snap = await db.doc(caminho).get();
        documentos[caminho] = snap.data() ? clonar(snap.data()) : null;
      }
    }
    const payload = {
      app: "Finanças pessoais",
      versaoExportacao: 2,
      exportadoEm: new Date().toISOString(),
      contagem: { lancamentos: lancamentos.length },
      colecoes: comLanc ? { lancamentos } : {},
      documentos
    };
    const tipo = comLanc && comCfg ? 'export' : (comLanc ? 'lancamentos' : 'configuracoes');
    const ok = await downloads.save({ filename: `financas-${tipo}-${new Date().toISOString().slice(0,10)}.json`, data: JSON.stringify(payload, null, 2) });
    const nDocs = Object.values(documentos).filter(Boolean).length;
    statusEl.textContent = ok ? `Exportado ✓ (${lancamentos.length} lançamentos e ${nDocs} documentos de configuração)` : "Exportação cancelada.";
  }catch(e){
    statusEl.textContent = "⚠️ Erro ao exportar: " + ((e && (e.code||'')) + ' ' + (e && (e.message||e)));
  }
}

// ===== VERIFICAÇÃO: arquivo x banco de dados =====
// Depois de importar (ou a qualquer momento, com o arquivo já processado), lê de volta o que está gravado no mesmo
// período e bancos e compara com o arquivo, por ano e banco: quantidade, entradas e saídas. O que está no arquivo
// e não está no banco aparece linha a linha. O que já existia no banco e não está no arquivo só é contado.
let loteArquivo = null; // { itens, ignoradas, porId } do último arquivo processado
function chaveVerif(l){ return chaveLancamento(l) + '|' + l.tipo; }
function totalVazio(){ return {n:0, ent:0, sai:0}; }
function somarTotal(t, l){ t.n++; if(l.tipo==='entrada') t.ent += Number(l.valor); else t.sai += Number(l.valor); }

async function lerBancoParaVerificar(itens){
  const bancos = [...new Set(itens.map(l=>l.banco).filter(Boolean))];
  const datas = itens.map(l=>l.dataEvento).sort();
  const docs = [];
  for(const b of bancos){
    const snap = await db.collection("lancamentos").where("banco","==",b)
      .where("dataEvento",">=",datas[0]).where("dataEvento","<=",datas[datas.length-1]).limit(1000000).get();
    snap.docs.forEach(d=>docs.push({id:d.id, ...d.data()}));
  }
  return docs;
}

function compararArquivoBanco(itens, docs, porId){
  const grupos = {};
  const g = l=>{
    const mes = mesDeISO(l.dataEvento), ano = mes.slice(0,4), banco = l.banco || '(sem banco)';
    return grupos[mes+'|'+banco] = grupos[mes+'|'+banco] || {ano, mes, banco, arq:totalVazio(), grav:totalVazio(), faltando:[], diferentes:[], extras:0, repetidas:0};
  };
  if(porId){
    const mapa = new Map(docs.map(d=>[d.id, d]));
    const ids = new Set(itens.map(l=>l.id));
    itens.forEach(l=>{
      const gr = g(l); somarTotal(gr.arq, l);
      const d = mapa.get(l.id);
      if(!d) gr.faltando.push(l);
      else if(chaveVerif(d)!==chaveVerif(l)) gr.diferentes.push({arquivo:l, banco:d});
      else somarTotal(gr.grav, l);
    });
    docs.forEach(d=>{ if(!ids.has(d.id)) g(d).extras++; });
  }else{
    // chave igual é o mesmo lançamento (o app não grava duplicados): linha repetida no arquivo conta uma vez
    const noBanco = new Set(docs.map(chaveVerif)), vistas = new Set();
    itens.forEach(l=>{
      const gr = g(l), k = chaveVerif(l);
      if(vistas.has(k)){ gr.repetidas++; return; }
      vistas.add(k);
      somarTotal(gr.arq, l);
      if(noBanco.has(k)) somarTotal(gr.grav, l); else gr.faltando.push(l);
    });
    docs.forEach(d=>{ if(!vistas.has(chaveVerif(d))) g(d).extras++; });
  }
  return Object.values(grupos).sort((a,b)=>(a.ano+'|'+a.banco+'|'+a.mes).localeCompare(b.ano+'|'+b.banco+'|'+b.mes));
}

function htmlVerificacao(grupos, lote){
  const faltam = grupos.reduce((s,x)=>s+x.faltando.length, 0), dif = grupos.reduce((s,x)=>s+x.diferentes.length, 0);
  const total = grupos.reduce((s,x)=>s+x.arq.n, 0), ok = grupos.reduce((s,x)=>s+x.grav.n, 0), rep = grupos.reduce((s,x)=>s+x.repetidas, 0);
  let h = (!faltam && !dif)
    ? `<p style="color:var(--income);font-weight:600;">✓ Tudo certo: os ${total} lançamentos do arquivo estão no banco de dados.</p>`
    : `<p style="color:var(--expense);font-weight:600;">⚠ ${ok} de ${total} lançamentos do arquivo conferem` + (faltam?`; ${faltam} não estão no banco`:'') + (dif?`; ${dif} estão com dados diferentes`:'') + '.</p>';
  if(rep) h += `<p class="muted">${rep} linha(s) repetida(s) no arquivo (mesmo dia, valor, banco, descrição, status e tipo) contam como o mesmo lançamento: o app não grava duplicados.</p>`;
  if(lote.ignoradas) h += `<p class="muted">${lote.ignoradas} linha(s) do arquivo foram ignoradas na leitura (sem data ou sem valor) e não entram nesta conta.</p>`;
  // uma linha por ano e banco (recolhida); clicar nela abre os meses. Quem tem divergência já vem aberto.
  const porAnoBanco = [];
  grupos.forEach(x=>{
    let p = porAnoBanco.find(y=>y.ano===x.ano && y.banco===x.banco);
    if(!p){ p = {ano:x.ano, banco:x.banco, arq:totalVazio(), grav:totalVazio(), faltando:[], diferentes:[], extras:0, repetidas:0, meses:[]}; porAnoBanco.push(p); }
    p.meses.push(x);
    p.arq.n += x.arq.n; p.arq.ent += x.arq.ent; p.arq.sai += x.arq.sai; p.grav.n += x.grav.n;
    p.faltando = p.faltando.concat(x.faltando); p.diferentes = p.diferentes.concat(x.diferentes);
    p.extras += x.extras; p.repetidas += x.repetidas;
  });
  const situacao = x=>{
    const n = x.faltando.length + x.diferentes.length;
    return (n ? `<span style="color:var(--expense);">⚠ ${n} divergente(s)</span>` : '<span style="color:var(--income);">✓ confere</span>')
      + (x.extras ? ` <span class="muted">(+${x.extras} só no banco)</span>` : '')
      + (x.repetidas ? ` <span class="muted">(+${x.repetidas} repetida${x.repetidas>1?'s':''} no arquivo)</span>` : '');
  };
  const num = 'text-align:right;';
  h += '<p class="muted" style="margin:4px 0;">Toque numa linha de ano e banco para ver os meses.</p>';
  h += `<div style="overflow-x:auto;"><table style="width:100%;font-size:13px;border-collapse:collapse;"><tr class="muted"><td>Ano · mês</td><td>Banco</td><td style="${num}">Arquivo</td><td style="${num}">Entradas</td><td style="${num}">Saídas</td><td style="${num}">No banco</td><td>Situação</td></tr>`;
  porAnoBanco.forEach((p, i)=>{
    const aberto = p.faltando.length > 0 || p.diferentes.length > 0;
    h += `<tr data-vf="${i}" style="border-top:1px solid var(--border);cursor:pointer;font-weight:600;"><td><span class="vf-seta">${aberto?'▾':'▸'}</span> ${esc(p.ano)}</td><td>${esc(p.banco)}</td><td style="${num}">${p.arq.n}</td><td style="${num}">${fmtMoeda(p.arq.ent)}</td><td style="${num}">${fmtMoeda(p.arq.sai)}</td><td style="${num}">${p.grav.n}</td><td style="font-weight:400;">${situacao(p)}</td></tr>`;
    p.meses.forEach(m=>{
      h += `<tr data-pai="${i}" style="${aberto?'':'display:none;'}"><td class="muted" style="padding-left:18px;">${esc(rotuloMesCurto(m.mes))}</td><td></td><td style="${num}">${m.arq.n}</td><td style="${num}">${fmtMoeda(m.arq.ent)}</td><td style="${num}">${fmtMoeda(m.arq.sai)}</td><td style="${num}">${m.grav.n}</td><td>${situacao(m)}</td></tr>`;
    });
  });
  h += '</table></div>';
  const itens = [];
  grupos.forEach(x=>{
    x.faltando.forEach(l=>itens.push(`<div class="item-meta">Não está no banco: ${new Date(l.dataEvento).toLocaleDateString('pt-BR')} · ${esc(l.banco||'-')} · ${esc(l.descricao||'')} · ${fmtMoeda(l.valor)} (${l.tipo})</div>`));
    x.diferentes.forEach(p=>itens.push(`<div class="item-meta">Dados diferentes (id ${esc(p.arquivo.id)}): arquivo ${esc(p.arquivo.descricao||'')} ${fmtMoeda(p.arquivo.valor)} · banco ${esc(p.banco.descricao||'')} ${fmtMoeda(p.banco.valor)}</div>`));
  });
  if(itens.length){
    h += `<div style="max-height:240px;overflow-y:auto;margin-top:8px;">${itens.slice(0,200).join('')}${itens.length>200?`<div class="item-meta">… e mais ${itens.length-200}.</div>`:''}</div>`;
    if(!lote.porId) h += '<p class="muted">Se alguma linha acima já deveria ter entrado, confira se ela mudou de descrição, valor, banco, data ou tipo no Histórico depois da importação, e importe de novo.</p>';
  }
  return h;
}

async function verificarArquivoXBanco(lote){
  const card = document.getElementById('cardVerif'), corpo = document.getElementById('verifCorpo');
  lote = lote || loteArquivo;
  if(!db || !lote || !lote.itens.length){ return; }
  card.classList.remove('hidden');
  corpo.innerHTML = '<p class="muted">Conferindo o arquivo com o banco de dados...</p>';
  try{
    const docs = await lerBancoParaVerificar(lote.itens);
    corpo.innerHTML = htmlVerificacao(compararArquivoBanco(lote.itens, docs, !!lote.porId), lote);
  }catch(e){
    console.error(e);
    corpo.innerHTML = '<p style="color:var(--expense);">⚠ Não consegui conferir: ' + esc((e && e.message) || String(e)) + '</p>';
  }
}
function guardarLoteArquivo(itens, ignoradas, porId){
  loteArquivo = {itens: itens.filter(l=>l.dataEvento), ignoradas: ignoradas||0, porId: !!porId};
  document.getElementById('btnVerificar').classList.toggle('hidden', !loteArquivo.itens.length);
  document.getElementById('cardVerif').classList.add('hidden');
}
document.getElementById('btnVerificar').addEventListener('click', ()=>verificarArquivoXBanco());
document.getElementById('verifCorpo').addEventListener('click', e=>{
  const tr = e.target.closest('tr[data-vf]');
  if(!tr) return;
  const filhos = document.querySelectorAll('#verifCorpo tr[data-pai="' + tr.dataset.vf + '"]');
  if(!filhos.length) return;
  const abrir = filhos[0].style.display === 'none';
  filhos.forEach(f=>{ f.style.display = abrir ? '' : 'none'; });
  tr.querySelector('.vf-seta').textContent = abrir ? '▾' : '▸';
});

function limparPreview(){
  lancamentosParaImportar = [];
  loteArquivo = null;
  document.getElementById('btnVerificar').classList.add('hidden');
  document.getElementById('cardVerif').classList.add('hidden');
  document.getElementById('cardPreview').classList.add('hidden');
  document.getElementById('listaPreview').innerHTML = '';
  document.getElementById('arquivoImport').value = '';
  document.getElementById('textoColado').value = '';
  document.getElementById('importStatus').textContent = '';
}
document.getElementById('btnLimparPreview').addEventListener('click', limparPreview);
document.getElementById('btnProcessarTexto').addEventListener('click', processarTextoColado);
document.getElementById('btnProcessar').addEventListener('click', processarArquivo);
document.getElementById('btnConfirmarImport').addEventListener('click', confirmarImportacao);
async function importarBackup(){
  const st = document.getElementById('backupStatus'), inp = document.getElementById('arquivoBackup'), btn = document.getElementById('btnImportarBackup');
  if(!db || !db.importarLancamentos){ st.textContent = 'Importação de backup indisponível nesta versão.'; return; }
  const querLanc = document.getElementById('impLanc').checked, querCfg = document.getElementById('impCfg').checked;
  if(!querLanc && !querCfg){ st.textContent = 'Marque ao menos uma opção para importar.'; return; }
  if(!inp.files.length){ st.textContent = 'Escolha o arquivo .json do backup.'; return; }
  let dados;
  try{ dados = JSON.parse(await inp.files[0].text()); }
  catch(e){ st.textContent = 'Arquivo inválido: ' + e.message; return; }
  const itens = (dados && dados.colecoes && dados.colecoes.lancamentos) || null;
  const docs = Object.entries((dados && dados.documentos) || {}).filter(([k,v])=>v);
  if(querLanc && !Array.isArray(itens)){ st.textContent = 'Este arquivo não tem lançamentos. Desmarque "Importar lançamentos" ou escolha outro arquivo.'; return; }
  if(querCfg && !docs.length){ st.textContent = 'Este arquivo não tem configurações. Desmarque "Importar configurações" ou escolha outro arquivo.'; return; }
  const partes = [];
  if(querLanc) partes.push(`${itens.length} lançamento(s)`);
  if(querCfg) partes.push(`${docs.length} configuração(ões)`);
  if(!(await confirmarModal(`Importar ${partes.join(' e ')}? Itens com o mesmo id (ou a mesma configuração) serão sobrescritos; nada é apagado.`))) return;
  btn.disabled = true;
  try{
    let msg = 'Concluído:';
    if(querLanc){
      const r = await db.importarLancamentos(itens, (feitos,total)=>{ st.textContent = `Importando lançamentos... ${feitos}/${total}`; });
      msg += ` ${r.gravados} lançamento(s) gravado(s) de ${itens.length} no arquivo` + (dados.contagem && dados.contagem.lancamentos!=null ? ` (contagem declarada: ${dados.contagem.lancamentos})` : '') + '.';
      guardarLoteArquivo(itens.filter(x=>x && x.id && !Number.isNaN(new Date(x.dataEvento).getTime())).map(x=>({...x, dataEvento:new Date(x.dataEvento).toISOString()})), 0, true);
      if(r.rejeitados.length) msg += ` ${r.rejeitados.length} rejeitado(s): ` + r.rejeitados.slice(0,5).map(x=>`#${x.indice+1} ${x.motivo}`).join('; ') + (r.rejeitados.length>5?'…':'') + '.';
    }
    if(querCfg){
      for(const [caminho,obj] of docs){ st.textContent = `Gravando ${caminho}...`; await db.doc(caminho).set(obj); }
      msg += ` ${docs.length} configuração(ões) gravada(s).`;
    }
    st.textContent = msg;
    if(querLanc) await verificarArquivoXBanco();
  }catch(e){
    st.textContent = '⚠️ Erro na importação: ' + (e && e.message || e) + ' Você pode tentar de novo: o que já foi gravado não duplica.';
  }finally{ btn.disabled = false; }
}
document.getElementById('btnImportarBackup').addEventListener('click', importarBackup);
document.getElementById('btnExportar').addEventListener('click', exportarDados);
document.getElementById('btnImportarFech').addEventListener('click', importarFechamentos);

async function salvarNomePerfil(){
  const msg = document.getElementById('perfilMsg'), inp = document.getElementById('perfilNome');
  if(!window.claude || !claude.definirApelido){ msg.textContent = 'Indisponível nesta versão.'; return; }
  msg.textContent = 'Salvando...';
  try{
    const nome = await claude.definirApelido(inp.value);
    inp.value = nome;
    usAtualizar();
    msg.textContent = nome ? 'Salvo ✓' : 'Nome removido ✓';
  }catch(e){
    msg.textContent = '⚠️ ' + ((e && e.message) || e);
  }
}
document.getElementById('btnSalvarNome').addEventListener('click', salvarNomePerfil);

async function salvarTelegram(){
  const msg = document.getElementById('telegramMsg'), inp = document.getElementById('telegramChatId');
  if(!window.claude || !claude.definirTelegram){ msg.textContent = 'Indisponível nesta versão.'; return; }
  msg.textContent = 'Salvando...';
  try{
    const chat = await claude.definirTelegram(inp.value);
    inp.value = chat;
    msg.textContent = chat ? 'Salvo ✓ Use "Enviar aviso de teste" para conferir.' : 'Telegram desvinculado ✓';
    atualizarCampoAviso();
  }catch(e){ msg.textContent = '⚠️ ' + ((e && e.message) || e); }
}
async function testarTelegram(){
  const msg = document.getElementById('telegramMsg');
  if(!window.claude || !claude.testarAviso){ msg.textContent = 'Indisponível nesta versão.'; return; }
  msg.textContent = 'Enviando...';
  try{ await claude.testarAviso(); msg.textContent = 'Enviado ✓ Confira o Telegram.'; }
  catch(e){ msg.textContent = '⚠️ ' + ((e && e.message) || e); }
}
document.getElementById('btnSalvarTelegram').addEventListener('click', salvarTelegram);
document.getElementById('btnTesteTelegram').addEventListener('click', testarTelegram);

window.addEventListener('unhandledrejection', ev=>{
  const r = ev.reason;
  if(r && r.code === 403){ ev.preventDefault(); avisar('Sem permissão: ' + (r.message || 'ação não permitida para o seu perfil.')); }
});
init();
