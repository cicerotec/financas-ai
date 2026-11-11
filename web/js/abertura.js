// Abertura (splash): a tela que cobre o app ate o Lancar ter o que precisa para ser usado (listas, tags e combos).
// Padrao: .ai/padrao-de-telas.md ("Abertura"), demo .ai/demos/demo-abertura-v1.html (aprovada em 2026-11-10).
// Quem diz o que chegou e o init do web/js/app.js (Abertura.chegou e Abertura.falhou); este modulo so decide quando sair:
// fica no minimo MIN ms (nao pisca), avisa depois de AVISO ms e, se algo falhar, mostra o erro com "Tentar de novo" e nao sai.
// Nao le nada do servidor e nao e camada do botao voltar (voltar durante a abertura sai do app).
function criarAbertura(env){
  const MIN = 600, AVISO = 4000, PARTES = ['listas', 'tags', 'combos'];
  const raiz = env.el('abertura');
  const chegaram = new Set();
  const inicio = env.agora();
  let estado = raiz ? 'abrindo' : 'fora'; // abrindo | erro | fora

  function sair(){
    if(estado !== 'abrindo') return;
    estado = 'fora';
    env.fundo(false);
    raiz.classList.add('saindo');
    env.setTimeout(()=>{ raiz.hidden = true; }, 200);
  }
  // Uma parte do Lancar chegou. Com todas, sai (respeitando o tempo minimo). Releituras depois disso nao fazem nada.
  function chegou(parte){
    if(estado !== 'abrindo' || !PARTES.includes(parte)) return;
    chegaram.add(parte);
    if(chegaram.size < PARTES.length) return;
    const falta = MIN - (env.agora() - inicio);
    if(falta > 0) env.setTimeout(sair, falta); else sair();
  }
  // tipo: 'conta' (a pessoa nao tem espaco) ou 'rede' (qualquer outra falha). Devolve false se a abertura ja saiu (quem chama avisa de outro jeito).
  function falhou(tipo){
    if(estado === 'fora') return false;
    estado = 'erro';
    const conta = tipo === 'conta';
    env.el('abErroT').textContent = conta ? 'Sua conta ainda não tem um espaço.' : 'Não consegui carregar seus dados.';
    env.el('abErroS').textContent = conta ? 'Peça o convite a quem cuida do espaço da família.' : 'Confira a conexão com a internet.';
    env.el('abSair').hidden = !conta;
    raiz.classList.add('erro');
    return true;
  }

  if(raiz){
    env.fundo(true);
    env.el('abDeNovo').onclick = ()=>env.recarregar();
    env.el('abSair').onclick = ()=>env.sair();
    env.setTimeout(()=>{ if(estado === 'abrindo') env.el('abMsg').textContent = 'Está demorando mais que o normal. Continuo tentando…'; }, AVISO);
  }
  return {chegou, falhou, estado: ()=>estado};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarAbertura;
const Abertura = (typeof window !== 'undefined') ? criarAbertura({
  el: id=>document.getElementById(id),
  agora: ()=>Date.now(),
  setTimeout: (fn, ms)=>window.setTimeout(fn, ms),
  // enquanto a abertura cobre a tela, o que esta atras nao recebe foco nem toque (o cabecalho e o main ainda nao existem quando este script roda)
  fundo: sim=>{ document.documentElement.classList.toggle('abrindo', sim); },
  recarregar: ()=>window.location.reload(),
  sair: ()=>{ if(window.financasApi && window.financasApi.sair) window.financasApi.sair(); }
}) : null;
