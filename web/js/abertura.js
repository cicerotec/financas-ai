// Abertura (splash): a tela que cobre o app ate o Lancar ter o que precisa para ser usado (listas, tags e combos).
// Padrao: .ai/padrao-de-telas.md ("Abertura"), demos .ai/demos/demo-abertura-v2-icone.html e demo-abertura-v3-sem-sessao.html
// (a v3, aprovada em 2026-11-12, e o estado "sem sessao" do app Android).
// Quem diz o que chegou e o init do web/js/app.js (Abertura.chegou e Abertura.falhou); este modulo so decide quando sair:
// fica no minimo MIN ms (nao pisca), avisa depois de AVISO ms e, se algo falhar, mostra o erro com "Tentar de novo" e nao sai.
// No app Android, quem cuida do login (web/js/sessao.js) avisa por Abertura.sessao: sem sessao a tela mostra o botao "Entrar"
// (o login so abre no toque), e pode voltar a cobrir o app depois de ter saido (a sessao terminou com o app aberto).
// Nao le nada do servidor e nao e camada do botao voltar (voltar durante a abertura sai do app).
function criarAbertura(env){
  const MIN = 600, AVISO = 4000, PARTES = ['listas', 'tags', 'combos'];
  const ERROS = {
    conta: ['Sua conta ainda não tem um espaço.', 'Peça o convite a quem cuida do espaço da família.'],
    cofre: ['Não consegui usar o armazenamento seguro deste aparelho.', 'Feche o app e abra de novo.'],
    rede: ['Não consegui carregar seus dados.', 'Confira a conexão com a internet.']
  };
  const SEM = { // titulo (vazio = sem aviso), frase, titulo em vermelho?
    nova: ['', 'Entre com a sua conta para continuar.', false],
    fim: ['Sua sessão terminou.', 'Entre de novo para continuar.', false],
    falhou: ['Não consegui concluir o login.', 'Confira a conexão com a internet e entre de novo.', true]
  };
  const raiz = env.el('abertura');
  const chegaram = new Set();
  const inicio = env.agora();
  let estado = raiz ? 'abrindo' : 'fora'; // abrindo | erro | sem | entrando | fora
  let erroAtual = '', comAviso = false;

  function sair(){
    if(estado !== 'abrindo') return;
    estado = 'fora';
    env.fundo(false);
    raiz.classList.add('saindo');
    env.setTimeout(()=>{ if(estado === 'fora') raiz.hidden = true; }, 200);
  }
  // volta a cobrir a tela (mesmo depois de ter saido) e zera as marcas de estado
  function cobrir(){
    raiz.hidden = false;
    raiz.classList.remove('saindo', 'erro', 'sem-sessao', 'com-aviso');
    env.fundo(true);
  }
  // Uma parte do Lancar chegou. Com todas, sai (respeitando o tempo minimo). Releituras depois disso nao fazem nada.
  function chegou(parte){
    if(estado !== 'abrindo' || !PARTES.includes(parte)) return;
    chegaram.add(parte);
    if(chegaram.size < PARTES.length) return;
    const falta = MIN - (env.agora() - inicio);
    if(falta > 0) env.setTimeout(sair, falta); else sair();
  }
  function mostrarErro(tipo){
    cobrir();
    estado = 'erro'; erroAtual = tipo;
    const t = ERROS[tipo] || ERROS.rede;
    env.el('abErroT').textContent = t[0];
    env.el('abErroS').textContent = t[1];
    env.el('abSair').hidden = tipo !== 'conta';
    raiz.classList.add('erro');
  }
  // tipo: 'conta' (a pessoa nao tem espaco) ou 'rede' (qualquer outra falha). Devolve false se a abertura ja saiu (quem chama avisa de outro jeito).
  // Sem sessao e o erro do armazenamento seguro ja estao na tela e ficam: a falha de leitura e consequencia deles.
  function falhou(tipo){
    if(estado === 'fora') return false;
    if(estado === 'sem' || estado === 'entrando' || (estado === 'erro' && erroAtual === 'cofre')) return true;
    mostrarErro(tipo === 'conta' ? 'conta' : 'rede');
    return true;
  }
  // O login do app Android (web/js/sessao.js). tipo: 'nova' | 'fim' | 'falhou' (sem sessao, com o botao Entrar), 'entrando' ou 'cofre'.
  function sessao(tipo){
    if(!raiz) return;
    if(tipo === 'cofre'){ mostrarErro('cofre'); return; }
    if(estado === 'erro' && erroAtual === 'cofre') return; // sem armazenamento seguro nao ha Entrar
    if(tipo === 'entrando'){
      cobrir();
      estado = 'entrando'; // o login esta terminando: a pagina recarrega em seguida (ou volta a 'sem')
      env.el('abMsg').textContent = 'Entrando…';
      return;
    }
    const t = SEM[tipo];
    if(!t) return;
    if(estado === 'sem' && tipo === 'nova' && comAviso) return; // um aviso na tela nao e trocado pela frase comum
    cobrir();
    estado = 'sem'; comAviso = !!t[0];
    const tit = env.el('abEntT');
    tit.textContent = t[0];
    tit.classList.toggle('ruim', t[2]);
    env.el('abEntS').textContent = t[1];
    raiz.classList.add('sem-sessao');
    if(comAviso) raiz.classList.add('com-aviso');
  }

  if(raiz){
    env.fundo(true);
    env.el('abDeNovo').onclick = ()=>env.recarregar();
    env.el('abSair').onclick = ()=>env.sair();
    const entrar = env.el('abEntrar');
    if(entrar) entrar.onclick = ()=>env.entrar();
    env.setTimeout(()=>{ const m = env.el('abMsg'); if(estado === 'abrindo' && !m.textContent) m.textContent = 'Está demorando mais que o normal. Continuo tentando…'; }, AVISO);
  }
  return {chegou, falhou, sessao, estado: ()=>estado};
}

if(typeof module !== 'undefined' && module.exports) module.exports = criarAbertura;
const Abertura = (typeof window !== 'undefined') ? criarAbertura({
  el: id=>document.getElementById(id),
  agora: ()=>Date.now(),
  setTimeout: (fn, ms)=>window.setTimeout(fn, ms),
  // enquanto a abertura cobre a tela, o que esta atras nao recebe foco nem toque (o cabecalho e o main ainda nao existem quando este script roda)
  fundo: sim=>{ document.documentElement.classList.toggle('abrindo', sim); },
  recarregar: ()=>window.location.reload(),
  sair: ()=>{ if(window.financasApi && window.financasApi.sair) window.financasApi.sair(); },
  entrar: ()=>{ if(window.financasApi && window.financasApi.entrar) window.financasApi.entrar(); }
}) : null;
