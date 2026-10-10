// Logica pura da repeticao de eventos futuros (botao Prever e "Repetir" ao lancar): datas, parcelas, duplicados e os
// lancamentos a criar. Sem DOM nem rede; testada em recorrencia.test.js (node --test web/).
(function (raiz) {
  const MAX_OCORRENCIAS = 36;      // copias por vez (Futuros carrega os 1000 lancamentos mais recentes)
  const MAX_INTERVALO_MESES = 120;
  const INTERVALOS = [
    { meses: 1, rotulo: 'Mensal' },
    { meses: 3, rotulo: 'Trimestral' },
    { meses: 6, rotulo: 'Semestral' },
    { meses: 12, rotulo: 'Anual' }
  ];

  const pad2 = (n) => String(n).padStart(2, '0');
  const diaLocal = (d) => d.getFullYear() + '-' + pad2(d.getMonth() + 1) + '-' + pad2(d.getDate());

  // Soma meses mantendo o dia da data ORIGINAL; dia que nao existe no mes vira o ultimo (31/01 + 1 mes = 28/02).
  // Sempre calculado a partir da data base, nunca da copia anterior (31/01, 28/02, 31/03, e nao 28/03).
  function somarMeses(base, meses) {
    const alvo = new Date(base.getFullYear(), base.getMonth() + meses, 1,
      base.getHours(), base.getMinutes(), base.getSeconds(), base.getMilliseconds());
    const ultimo = new Date(alvo.getFullYear(), alvo.getMonth() + 1, 0).getDate();
    alvo.setDate(Math.min(base.getDate(), ultimo));
    return alvo;
  }

  // Datas das `quantidade` copias depois da base, a cada `intervaloMeses`.
  function gerarDatas(base, intervaloMeses, quantidade) {
    const datas = [];
    for (let k = 1; k <= quantidade; k++) datas.push(somarMeses(base, intervaloMeses * k));
    return datas;
  }

  function rotuloIntervalo(meses) {
    const p = INTERVALOS.find((i) => i.meses === meses);
    if (p) return p.rotulo.toLowerCase();
    if (meses % 12 === 0) return 'a cada ' + (meses / 12) + ' anos';
    return 'a cada ' + meses + ' meses';
  }

  // null se valido; senao a mensagem para a tela.
  function validar(intervaloMeses, quantidade) {
    if (!Number.isInteger(intervaloMeses) || intervaloMeses < 1 || intervaloMeses > MAX_INTERVALO_MESES)
      return 'O intervalo deve ser de 1 a ' + MAX_INTERVALO_MESES + ' meses.';
    if (!Number.isInteger(quantidade) || quantidade < 1 || quantidade > MAX_OCORRENCIAS)
      return 'Repita de 1 a ' + MAX_OCORRENCIAS + ' vezes.';
    return null;
  }

  // "Notebook 2/10" -> { a: 2, b: 10 }. So o primeiro "n/m" da descricao conta.
  const RE_PARCELA = /(\d+)\s*\/\s*(\d+)/;
  function lerParcela(descricao) {
    const m = RE_PARCELA.exec(descricao || '');
    if (!m) return null;
    const a = parseInt(m[1], 10), b = parseInt(m[2], 10);
    return b >= 1 && a >= 1 && a <= b ? { a, b } : null;
  }
  // Quantas copias ainda cabem na numeracao (2/10 -> 8). null = a descricao nao tem numeracao.
  function copiasRestantesNaParcela(descricao) {
    const p = lerParcela(descricao);
    return p ? p.b - p.a : null;
  }
  // Descricao da copia k (1, 2, ...): "Notebook 2/10" -> "Notebook 3/10".
  function descricaoNumerada(descricao, k) {
    const p = lerParcela(descricao);
    if (!p) return descricao;
    return descricao.replace(RE_PARCELA, (p.a + k) + '/' + p.b);
  }

  // Chave para achar evento igual: status, descricao, valor, banco e DIA (sem a hora).
  function chaveDuplicado(l) {
    return [l.status, (l.descricao || '').trim().toLowerCase(), Number(l.valor).toFixed(2),
      (l.banco || '').trim().toLowerCase(), diaLocal(new Date(l.dataEvento))].join('|');
  }

  // Lancamentos a criar (ainda sem id). base: o registro de origem; op:
  //   { intervaloMeses, quantidade, status, aviso (obj|null), numerar (bool), serieId, indiceInicial }
  // A copia k carrega serie { id, indice, intervalo } (indice = indiceInicial + k - 1).
  function montarCopias(base, op) {
    const erro = validar(op.intervaloMeses, op.quantidade);
    if (erro) throw new Error(erro);
    let quantidade = op.quantidade;
    if (op.numerar) {
      const resta = copiasRestantesNaParcela(base.descricao);
      if (resta !== null) quantidade = Math.min(quantidade, resta);
    }
    const datas = gerarDatas(new Date(base.dataEvento), op.intervaloMeses, quantidade);
    return datas.map((data, i) => {
      const k = i + 1;
      const doc = {
        status: op.status,
        tipo: base.tipo,
        valor: base.valor,
        descricao: op.numerar ? descricaoNumerada(base.descricao, k) : base.descricao,
        nota: base.nota || '',
        dataEvento: data.toISOString(),
        banco: base.banco,
        tags: [...(base.tags || [])],
        excluirDoTotal: /^CONTAS$|^TRANSFERINDO$/i.test(op.status),
        serie: { id: op.serieId, indice: (op.indiceInicial || 1) + i, intervalo: op.intervaloMeses }
      };
      if (op.aviso) doc.aviso = { dias: [...op.aviso.dias], insistir: !!op.aviso.insistir };
      return doc;
    });
  }

  // Ultima ocorrencia de uma serie entre os lancamentos carregados (a base do "estender").
  function ultimaDaSerie(lancamentos, serieId) {
    let ultima = null;
    for (const l of lancamentos) {
      if (!l.serie || l.serie.id !== serieId) continue;
      if (!ultima || l.serie.indice > ultima.serie.indice) ultima = l;
    }
    return ultima;
  }

  // ---- avisos (prazos em dias antes do vencimento) e frases das telas de Prever e Estender ----
  const MAX_PRAZOS = 6;            // por lancamento: os quatro atalhos mais dois prazos proprios
  const MAX_DIAS_AVISO = 30;
  const PRAZOS_PADRAO = [0, 1, 3, 7];

  function rotuloPrazo(d) {
    if (d === 0) return 'No dia';
    if (d > 7 && d % 7 === 0) return (d / 7) + ' semanas antes';
    return d + (d === 1 ? ' dia antes' : ' dias antes');
  }
  function juntar(p) { return p.length < 2 ? p.join('') : p.slice(0, -1).join(', ') + ' e ' + p[p.length - 1]; }
  // av = { on, dias, insistir } -> "1 dia antes e no dia, insiste depois"
  function fraseAviso(av) {
    if (!av.on) return 'Desligado';
    const ds = [...av.dias].sort((a, b) => b - a);
    if (!ds.length && !av.insistir) return 'Escolha um prazo';
    const p = ds.map((d) => (d === 0 ? 'no dia' : rotuloPrazo(d).toLowerCase()));
    let f = p.length ? juntar(p) : '';
    if (av.insistir) f = (f ? f + ', ' : '') + 'insiste depois';
    return f.charAt(0).toUpperCase() + f.slice(1);
  }
  // Prazo novo das rodas (n dias ou n semanas). Devolve { erro } ou { valor (em dias), texto }.
  function validarPrazoNovo(dias, n, semanas) {
    const d = n * (semanas ? 7 : 1);
    if (d > MAX_DIAS_AVISO) return { erro: 'O máximo é ' + MAX_DIAS_AVISO + ' dias (4 semanas).' };
    if (dias.includes(d)) return { erro: 'Esse prazo já está na lista.' };
    if (dias.length >= MAX_PRAZOS) return { erro: 'O máximo é ' + MAX_PRAZOS + ' prazos por lançamento.' };
    return { valor: d, texto: rotuloPrazo(d) };
  }
  // Intervalo novo das rodas (n meses ou n anos). Devolve { erro } ou { valor (em meses), texto }.
  function validarIntervaloNovo(n, anos) {
    const m = n * (anos ? 12 : 1);
    if (m > MAX_INTERVALO_MESES) return { erro: 'O máximo é 10 anos (120 meses).' };
    const t = rotuloIntervalo(m);
    return { valor: m, texto: t.charAt(0).toUpperCase() + t.slice(1) };
  }
  function fraseRepeticao(qtd, meses) { return qtd + (qtd === 1 ? ' vez' : ' vezes') + ' além deste, ' + rotuloIntervalo(meses); }
  // Estado da tela -> o que vai no lancamento: null (sem aviso), 'incompleto' (ligado sem nenhum prazo) ou { dias, insistir }.
  function avisoDoEstado(av) {
    if (!av.on) return null;
    if (!av.dias.length && !av.insistir) return 'incompleto';
    return { dias: [...av.dias].sort((a, b) => b - a), insistir: !!av.insistir };
  }
  // Linhas da lista que acumula: os atalhos e os prazos proprios marcados, em ordem crescente.
  function listaDePrazos(dias) {
    return [...PRAZOS_PADRAO, ...dias.filter((d) => !PRAZOS_PADRAO.includes(d))].sort((a, b) => a - b);
  }

  const api = { MAX_PRAZOS, MAX_DIAS_AVISO, PRAZOS_PADRAO, rotuloPrazo, fraseAviso, validarPrazoNovo, validarIntervaloNovo,
    fraseRepeticao, avisoDoEstado, listaDePrazos, MAX_OCORRENCIAS, MAX_INTERVALO_MESES, INTERVALOS, somarMeses, gerarDatas, rotuloIntervalo, validar,
    lerParcela, copiasRestantesNaParcela, descricaoNumerada, chaveDuplicado, montarCopias, ultimaDaSerie, diaLocal };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else raiz.Recorrencia = api;
})(typeof window !== 'undefined' ? window : globalThis);
