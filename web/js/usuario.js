// A conta de quem esta logado: nome, espaco e Sair. No celular entram no menu do app (abrirMenuApp, em app.js); na janela larga
// (1160 px ou mais) ficam num botao no canto direito do cabecalho. Padrao: .ai/padrao-de-telas.md ("Menu do app"), demo
// .ai/demos/demo-menu-usuario.html. So desenha: nome, espaco, papel e lista de espacos vem do api.js (window.claude.sessao()).
// Carregado depois do app.js e usa os globais dele (esc, fecharMenuApp, irParaAba).

function usSessao(){ return (window.claude && claude.sessao && claude.sessao()) || null; }
function usPapel(role){ return role === 'member' ? 'membro' : 'dono'; }
function usInicial(nome){ return String(nome || '?').trim().charAt(0).toUpperCase() || '?'; }

// Bloco do topo do menu: inicial, nome, espaco (ou o seletor de espaco, se ha mais de um) e a dica do apelido.
function htmlConta(s){
  const esp = s.espaco || {};
  const dois = (s.espacos || []).length > 1;
  const sub = dois ? '' : `<small>${esc(esp.nome || '')} · ${usPapel(esp.role)}</small>`;
  const seletor = dois
    ? `<div class="cta-esp"><label>Espaço<select data-conta="espaco" aria-label="Espaço">${s.espacos.map(e=>`<option value="${esc(e.id)}"${e.id === esp.id ? ' selected' : ''}>${esc(e.nome)}</option>`).join('')}</select></label></div>`
    : '';
  const dica = s.apelido ? '' : '<button type="button" class="cta-dica" data-conta="listas">Defina seu nome em <b>Listas</b></button>';
  return `<div class="cta-eu"><span class="cta-av" aria-hidden="true">${esc(usInicial(s.nome))}</span><div><b>${esc(s.nome)}</b>${sub}</div></div>${dica}${seletor}`;
}
const US_SAIR = '<hr class="mn-sep"><button type="button" class="mn-it mn-sair" role="menuitem" data-conta="sair"><span class="mn-ic">🚪</span>Sair</button>';

// Topo do menu do celular (antes das telas) e fim dele (depois das telas). Sem sessao (ambiente sem login), nada.
function htmlContaTopo(){ const s = usSessao(); return s ? htmlConta(s) + '<hr class="mn-sep">' : ''; }
function htmlContaFim(){ return usSessao() ? US_SAIR : ''; }

// Botao do computador: a inicial e o nome.
function usAtualizar(){
  const s = usSessao(), btn = document.getElementById('usrBtn');
  if(!btn) return;
  btn.hidden = !s;
  if(!s) return;
  document.getElementById('usrAv').textContent = usInicial(s.nome);
  document.getElementById('usrNome').textContent = s.nome;
}
function usFechar(voltarFoco){
  const btn = document.getElementById('usrBtn'), drop = document.getElementById('usrDrop');
  if(!drop || drop.hidden) return;
  drop.hidden = true;
  btn.setAttribute('aria-expanded', 'false');
  if(voltarFoco) btn.focus();
}
function usAbrir(){
  const s = usSessao(); if(!s) return;
  const drop = document.getElementById('usrDrop');
  drop.innerHTML = htmlConta(s) + US_SAIR;
  drop.hidden = false;
  document.getElementById('usrBtn').setAttribute('aria-expanded', 'true');
}
function usPreparar(){
  const btn = document.getElementById('usrBtn'), drop = document.getElementById('usrDrop');
  if(!btn || btn.dataset.pronto) return;
  btn.dataset.pronto = '1';
  btn.addEventListener('click', e=>{ e.stopPropagation(); if(drop.hidden) usAbrir(); else usFechar(false); });
  document.addEventListener('click', e=>{
    // as acoes do bloco valem no menu do celular e no menu do computador
    const a = e.target.closest && e.target.closest('[data-conta]');
    if(a && a.dataset.conta === 'sair'){ if(typeof fecharMenuApp === 'function') fecharMenuApp(false); usFechar(false); if(window.claude && claude.sair) claude.sair(); return; }
    if(a && a.dataset.conta === 'listas'){ if(typeof fecharMenuApp === 'function') fecharMenuApp(false); usFechar(false); irParaAba('listas'); return; }
    if(!drop.hidden && !(e.target.closest && e.target.closest('#usrDrop'))) usFechar(false); // clique fora
  });
  document.addEventListener('change', e=>{
    const sel = e.target.closest && e.target.closest('[data-conta="espaco"]');
    if(sel && window.claude && claude.trocarEspaco) claude.trocarEspaco(sel.value);
  });
  document.addEventListener('keydown', e=>{ if(e.key === 'Escape' && !drop.hidden) usFechar(true); });
}
document.addEventListener('DOMContentLoaded', usPreparar);
