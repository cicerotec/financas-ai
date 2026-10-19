// Botao voltar do navegador e do celular: cada tela ou janela que o "←" fecharia vira uma entrada do historico do navegador.
// Padrao: .ai/padrao-de-telas.md ("Botão voltar"), demo .ai/demos/demo-voltar-navegador.html (aprovada em 2026-10-19).
// Modelo: o historico guarda {aba, n, i}: a aba, quantas camadas (janelas e telas cheias) estao abertas e o indice da entrada.
// A tela e a fonte da verdade: o app diz qual aba e quantas camadas estao abertas (hooks) e este modulo so ajusta o historico.
// O aviso de Desfazer nao e camada e nunca entra aqui. Sem entrada extra para ele: no primeiro nivel, voltar sai do app.
function criarVoltar(env){
  const hist = env.history;
  let H = null;            // hooks do app, entregues em iniciar()
  let ent = [];            // espelho das entradas do historico: {aba, n}
  const pend = [];         // history.go() pedidos por nos: o popstate deles nao e a pessoa voltando
  let emPop = false, agendado = false, retorno = false, token = 0;

  const url = aba => env.location.pathname + env.location.search + '#' + aba;
  const topo = () => ent[ent.length - 1];
  function empurrar(aba, n){ const i = ent.length; ent.push({aba, n}); hist.pushState({aba, n, i}, '', url(aba)); }
  function trocarTopo(aba, n){ const i = ent.length - 1; ent[i] = {aba, n}; hist.replaceState({aba, n, i}, '', url(aba)); }
  function ir(delta){ // delta < 0: volta entradas pedindo ao navegador; o popstate que isso causa e nosso
    const t = ++token;
    pend.push(t);
    ent.length += delta;
    env.setTimeout(()=>{ const k = pend.indexOf(t); if(k >= 0){ pend.splice(k, 1); agendar(); } }, 1500); // go() que nao andou nao trava
    hist.go(delta);
  }
  function agendar(){
    if(agendado) return;
    agendado = true;
    env.microtask(aplicar);
  }

  // Compara o que a tela mostra com a ultima entrada do historico e acerta a diferenca.
  function aplicar(){
    agendado = false;
    if(!H || emPop) return;
    if(pend.length) return; // espera o popstate do go() anterior: o aoVoltar chama de novo
    const aba = H.abaAtual(), n = H.contarAbertas(), t = topo();
    if(!t) return;
    if(aba === t.aba){
      if(n > t.n) for(let i = t.n + 1; i <= n; i++) empurrar(aba, i);
      else if(n < t.n) ir(n - t.n);
      return;
    }
    // a aba mudou. O retorno (salvar ou cancelar a edicao) desfaz a ida em vez de empilhar de novo.
    const prev = ent[ent.length - 2 - t.n];
    if(retorno && prev && prev.aba === aba){
      retorno = false;
      ir(-(t.n + 1));
      return;
    }
    retorno = false;
    if(t.n > 0) trocarTopo(aba, 0); // a camada que fechou cede o lugar a aba nova (ex.: Editar na janela de acoes)
    else empurrar(aba, 0);
    for(let i = 1; i <= n; i++) empurrar(aba, i);
  }

  // A pessoa apertou voltar (ou avancar): fecha o que a entrada de destino nao tem.
  function aoVoltar(e){
    if(pend.length){ pend.shift(); if(!pend.length) agendar(); return; }
    const s = e.state, velho = topo();
    if(!H || !velho || !s || typeof s.aba !== 'string') return;
    if(s.aba !== velho.aba && H.precisaConfirmar && H.precisaConfirmar()){
      // texto digitado: devolve a entrada que a pessoa acabou de sair e pergunta antes de descartar
      hist.pushState({aba: velho.aba, n: velho.n, i: ent.length - 1}, '', url(velho.aba));
      Promise.resolve(H.perguntarSaida()).catch(()=>{});
      return;
    }
    const i = typeof s.i === 'number' ? s.i : ent.length - 1;
    if(i < ent.length - 1) ent.length = i + 1;
    else if(i > ent.length - 1){ ent.length = i; ent.push({aba: s.aba, n: s.n}); }
    ent[ent.length - 1] = {aba: s.aba, n: s.n};
    emPop = true;
    try{
      if(s.aba !== H.abaAtual()) H.irParaAba(s.aba);
      let guarda = 20;
      while(H.contarAbertas() > s.n && guarda-- > 0){
        const antes = H.contarAbertas();
        H.fecharTopo();
        if(H.contarAbertas() >= antes) break; // nao fechou (ex.: criando eventos): a entrada volta abaixo
      }
    }finally{ emPop = false; }
    const aba = H.abaAtual(), n = H.contarAbertas(), t = topo();
    if(aba === t.aba && n > t.n) for(let k = t.n + 1; k <= n; k++) empurrar(aba, k);
    else if(aba !== t.aba || n < t.n) trocarTopo(aba, n);
  }

  // hooks: abaAtual(), abaValida(aba), irParaAba(aba), contarAbertas(), fecharTopo(), precisaConfirmar(), perguntarSaida()
  function iniciar(hooks){
    H = hooks;
    let aba = H.abaAtual();
    const h = String(env.location.hash || '').replace(/^#/, '');
    if(h && h !== aba && H.abaValida(h)){ H.irParaAba(h); aba = H.abaAtual(); } // F5 reabre na mesma aba
    const st = hist.state;
    const i = st && typeof st.i === 'number' ? st.i : 0;
    ent = []; ent.length = i; ent.push({aba, n: 0});
    hist.replaceState({aba, n: 0, i}, '', url(aba));
    env.ouvir(aoVoltar);
    agendar();
  }
  // Marca que a proxima troca de aba e um retorno (salvar ou cancelar a edicao): volta uma entrada em vez de empilhar.
  function retornar(){ retorno = true; env.setTimeout(()=>{ retorno = false; }, 1000); }

  return {iniciar, sync: agendar, retornar, estado: ()=>ent.map(x=>x ? x.aba + ':' + x.n : '-')};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarVoltar;
const Voltar = (typeof window !== 'undefined') ? criarVoltar({
  history: window.history,
  location: window.location,
  ouvir: fn => window.addEventListener('popstate', fn),
  microtask: fn => queueMicrotask(fn),
  setTimeout: (fn, ms) => window.setTimeout(fn, ms)
}) : null;
