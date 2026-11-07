// Esconder valores (olho), como nos apps de banco. Demo aprovada: .ai/demos/demo-valores-v1-esconder.html.
// Com o olho fechado, todo valor LIDO na tela vira "R$ ••••••" (o sinal e a cor ficam); nos gráficos somem os números e as barras ficam.
// Não esconde o que a pessoa digita no Lançar (valor grande e resumo da barra) nem as janelas e painéis em que ela decide olhando o
// número (confirmações e a conferência com o extrato). A escolha fica só neste aparelho (localStorage), nunca vai ao servidor.
// Um observador mascara o que o app desenha depois, no mesmo ciclo, antes de a tela pintar: nenhuma tela precisa saber do olho.
(function () {
  const PONTOS = 'R$ ••••••';
  const RX = /R\$[\s ]*\d[\d.]*(?:,\d+)?/g;
  const NUMERO_DE_GRAFICO = /^\s*(?:até\s+)?\d[\d.,]*(?:\s*mil)?\s*$/;
  // o que nunca é escondido: o que se digita e onde se decide olhando o valor
  const EXCECOES = '.valor-grande,#barraResumo,.modal-box,.fm-conf';
  const CHAVE = 'valoresEscondidos';

  // Texto já mascarado; noSvg diz que o texto é um rótulo de gráfico (só número, "12 mil" ou "até 500").
  function mascararTexto(t, noSvg) {
    if (noSvg && NUMERO_DE_GRAFICO.test(t)) return '••';
    return t.replace(RX, PONTOS);
  }
  window.ValoresEscondidos = { mascararTexto: mascararTexto };
  if (typeof document === 'undefined') return;

  let escondidos = false;
  try { escondidos = localStorage.getItem(CHAVE) === '1'; } catch (e) { /* sem armazenamento: vale só nesta sessão */ }
  const originais = new Map(); // nó de texto -> texto real, para o olho abrir de novo

  function ignorado(no) {
    const el = no.nodeType === 1 ? no : no.parentElement;
    if (!el) return true;
    if (/^(INPUT|TEXTAREA|SCRIPT|STYLE)$/.test(el.tagName)) return true;
    return !!el.closest(EXCECOES);
  }
  function mascararNo(n) {
    if (n.nodeType !== 3 || ignorado(n)) return;
    const pai = n.parentElement, svg = !!(pai && pai.tagName.toLowerCase() === 'text' && pai.closest('svg'));
    const t = n.nodeValue, novo = mascararTexto(t, svg);
    if (novo === t) return;
    originais.set(n, t);
    n.nodeValue = novo;
  }
  function mascararAtributos(el) {
    if (el.nodeType !== 1 || ignorado(el)) return;
    ['aria-label', 'title'].forEach((a) => {
      const v = el.getAttribute(a);
      if (!v) return;
      RX.lastIndex = 0;
      if (!RX.test(v)) { RX.lastIndex = 0; return; }
      RX.lastIndex = 0;
      el.setAttribute('data-vo-' + a, v);
      el.setAttribute(a, v.replace(RX, PONTOS));
    });
  }
  function mascararArvore(raiz) {
    if (raiz.nodeType === 3) { mascararNo(raiz); return; }
    if (raiz.nodeType !== 1) return;
    const w = document.createTreeWalker(raiz, NodeFilter.SHOW_TEXT, null), nos = [];
    for (let n = w.nextNode(); n; n = w.nextNode()) nos.push(n);
    nos.forEach(mascararNo);
    mascararAtributos(raiz);
    raiz.querySelectorAll('[aria-label],[title]').forEach(mascararAtributos);
  }
  function desmascarar() {
    originais.forEach((t, n) => { if (n.isConnected) n.nodeValue = t; });
    originais.clear();
    ['aria-label', 'title'].forEach((a) => {
      document.querySelectorAll('[data-vo-' + a + ']').forEach((el) => { el.setAttribute(a, el.getAttribute('data-vo-' + a)); el.removeAttribute('data-vo-' + a); });
    });
  }

  const observador = new MutationObserver((lote) => {
    if (!escondidos) return;
    lote.forEach((m) => {
      if (m.type === 'childList') m.addedNodes.forEach(mascararArvore);
      else if (m.type === 'characterData') mascararNo(m.target);
      else if (m.type === 'attributes') mascararAtributos(m.target);
    });
    observador.takeRecords(); // descarta o que as nossas próprias trocas acabaram de gerar
    if (originais.size > 5000) originais.forEach((_, n) => { if (!n.isConnected) originais.delete(n); });
  });

  function pintar() {
    document.querySelectorAll('.olho').forEach((b) => {
      b.setAttribute('aria-pressed', String(escondidos));
      b.setAttribute('aria-label', escondidos ? 'Mostrar valores' : 'Esconder valores');
    });
    desmascarar();
    if (escondidos) { mascararArvore(document.body); observador.takeRecords(); }
    try { localStorage.setItem(CHAVE, escondidos ? '1' : '0'); } catch (e) { /* ignora */ }
  }

  function iniciar() {
    document.querySelectorAll('.olho').forEach((b) => b.addEventListener('click', () => { escondidos = !escondidos; pintar(); }));
    observador.observe(document.body, { childList: true, subtree: true, characterData: true, attributes: true, attributeFilter: ['aria-label', 'title'] });
    pintar();
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', iniciar); else iniciar();
})();
