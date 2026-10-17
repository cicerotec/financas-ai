package service

import (
	"context"
	"errors"
	"log"
	"math"
	"time"

	"financas/domain"
)

// Relogio e o instante "agora". O service nunca chama time.Now: quem monta o sistema escolhe o relogio (real ou fixo).
type Relogio interface{ Agora() time.Time }

// RepositorioSaldos e o que o service precisa do banco para o fechamento mensal (um item SALDO#<banco> por banco).
type RepositorioSaldos interface {
	// Ler devolve o item do banco (com a version), ou nil se ainda nao existe.
	Ler(ctx context.Context, sid, banco string) (domain.Doc, error)
	// Salvar grava com controle otimista (esperada = version lida). domain.ErrConflitoConfig se alguem gravou antes.
	Salvar(ctx context.Context, sid, banco string, body domain.Doc, esperada float64) (domain.Doc, error)
	// Todos devolve o item de cada banco do espaco.
	Todos(ctx context.Context, sid string) ([]domain.Doc, error)
}

var (
	// ErrMesNaoConferido: reabrir um mes que nao esta conferido (404).
	ErrMesNaoConferido = errors.New("esse mes nao esta conferido")
)

// ErroOrdemFechamento e o 409 de ordem: conferir ou reabrir fora da sequencia (codigo "ordem").
type ErroOrdemFechamento struct{ Msg string }

func (e *ErroOrdemFechamento) Error() string { return e.Msg }

// Fechamento mensal por banco.
//
// O saldo de um banco e uma cascata: o mes abre com o fechamento do anterior, desde a data inicial. Quem calcula e o
// navegador (so ele conhece quais status mexem no saldo); o servidor guarda e protege:
//
//   - caches: fechamento calculado de meses ja encerrados, so para nao reler o historico inteiro. Qualquer escrita em
//     lancamento marca "invalidoDe" e muda a version, entao o cache nunca sobrevive a uma alteracao e um cache calculado
//     antes dela nao consegue ser gravado depois.
//   - conferidos: o dono confirmou o mes com o extrato do banco. Mes conferido trava: nenhum lancamento que mexa no saldo
//     pode ser criado, alterado ou excluido nele nem nos meses anteriores, ate o dono reabrir (do mais recente para o
//     mais antigo).
//   - mes fechado por data: a trava seguinte, que a pessoa pode confirmar (domain.MesFechadoPorData).
//
// O mes e sempre o do fuso de Sao Paulo, pelo instante do Relogio.
type Fechamento struct {
	saldos RepositorioSaldos
	cfg    RepositorioConfig
	rel    Relogio
}

func NewFechamento(saldos RepositorioSaldos, cfg RepositorioConfig, rel Relogio) *Fechamento {
	return &Fechamento{saldos: saldos, cfg: cfg, rel: rel}
}

func versaoDe(d domain.Doc) float64 {
	v, _ := d["version"].(float64)
	return v
}

// conflito e o 409 de versao: outra pessoa gravou antes. Sem item, o estado atual e so {"version": 0}.
func conflito(atual domain.Doc) error {
	if atual == nil {
		atual = domain.Doc{"version": 0.0}
	}
	return &ErroConflitoConfig{Atual: domain.Publico(atual)}
}

// atualizar le, aplica fn e grava com controle otimista (tenta de novo se alguem gravou no meio).
func (s *Fechamento) atualizar(ctx context.Context, sid, banco string, fn func(d domain.Doc) error) (domain.Doc, error) {
	for i := 0; i < 5; i++ {
		d, err := s.saldos.Ler(ctx, sid, banco)
		if err != nil {
			return nil, err
		}
		if d == nil {
			d = domain.Doc{"banco": banco}
		}
		ver := versaoDe(d)
		if err := fn(d); err != nil {
			return nil, err
		}
		res, err := s.saldos.Salvar(ctx, sid, banco, d, ver)
		if errors.Is(err, domain.ErrConflitoConfig) {
			continue
		}
		return res, err
	}
	return nil, domain.ErrConflitoConfig
}

// ---------- trava dos meses ----------

// travasDaChamada consulta, ao longo de uma chamada, o ultimo mes conferido de cada banco e o inicio do controle,
// lendo cada banco (e a configuracao de saldos) uma vez so. Nao guarda nada entre chamadas: o Lambda reaproveita a
// memoria e um cache entre requisicoes serviria dado velho.
type travasDaChamada struct {
	ctx     context.Context
	sid     string
	s       *Fechamento
	ate     map[string]string
	inicio  map[string]string // banco -> mes da data de inicio ("" = sem data de inicio)
	cfgLida bool
	cfg     domain.Doc
}

// Travas devolve a consulta de trava de uma chamada (veja a interface Travas).
func (s *Fechamento) Travas(ctx context.Context, sid string) Travas { return s.novasTravas(ctx, sid) }

func (s *Fechamento) novasTravas(ctx context.Context, sid string) *travasDaChamada {
	return &travasDaChamada{ctx: ctx, sid: sid, s: s, ate: map[string]string{}, inicio: map[string]string{}}
}

// mesDeInicio devolve o mes da data de inicio do banco (config de saldos), "" se nao houver.
func (t *travasDaChamada) mesDeInicio(banco string) (string, error) {
	if m, ok := t.inicio[banco]; ok {
		return m, nil
	}
	if !t.cfgLida {
		d, err := t.s.cfg.Ler(t.ctx, t.sid, "saldos")
		if err != nil {
			return "", err
		}
		t.cfg, t.cfgLida = d, true
	}
	m := domain.MesDeInicio(t.cfg, banco)
	t.inicio[banco] = m
	return m, nil
}

// anteriorAoInicio: lancamento de mes anterior ao inicio do controle do banco nao entra no saldo (e historico), entao
// nao precisa de trava nem invalida cache.
func (t *travasDaChamada) anteriorAoInicio(a domain.Alvo) (bool, error) {
	ini, err := t.mesDeInicio(a.Banco)
	return ini != "" && a.Mes < ini, err
}

// Checar decide se uma escrita no alvo pode seguir, na ordem: (1) mes conferido com o extrato, absoluto, devolve
// *domain.ErroMesConferido e a confirmacao nao o destrava; (2) mes fechado por data (domain.MesFechadoPorData no instante
// do Relogio), que devolve *domain.ErroMesFechado se a pessoa nao confirmou. O que e anterior ao inicio do controle do
// banco nao entra no saldo e e livre nos dois casos. exigiu = a confirmacao era necessaria e foi dada.
func (t *travasDaChamada) Checar(a domain.Alvo, confirmado bool) (exigiu bool, err error) {
	ate, ok := t.ate[a.Banco]
	if !ok {
		d, err := t.s.saldos.Ler(t.ctx, t.sid, a.Banco)
		if err != nil {
			return false, err
		}
		ate = domain.UltimoConferido(d)
		t.ate[a.Banco] = ate
	}
	fechadoPorData := domain.MesFechadoPorData(a.Mes, t.s.rel.Agora())
	if (ate != "" && a.Mes <= ate) || fechadoPorData {
		antes, err := t.anteriorAoInicio(a)
		if err != nil {
			return false, err
		}
		if antes {
			return false, nil
		}
	}
	if ate != "" && a.Mes <= ate {
		return false, &domain.ErroMesConferido{Banco: a.Banco, Mes: a.Mes, Ate: ate}
	}
	if fechadoPorData {
		if !confirmado {
			return false, &domain.ErroMesFechado{Banco: a.Banco, Mes: a.Mes}
		}
		return true, nil
	}
	return false, nil
}

// InvalidarSaldos marca que os caches do banco deixaram de valer a partir do mes do alvo e muda a version (barra caches
// calculados antes desta escrita). Falha so vai ao log: o lancamento ja foi gravado.
func (s *Fechamento) InvalidarSaldos(ctx context.Context, sid string, alvos []domain.Alvo) {
	minimo := map[string]string{}
	tr := s.novasTravas(ctx, sid)
	for _, a := range alvos {
		if antes, err := tr.anteriorAoInicio(a); err == nil && antes {
			continue // historico anterior ao inicio nao muda nenhum saldo
		}
		if cur, ok := minimo[a.Banco]; !ok || a.Mes < cur {
			minimo[a.Banco] = a.Mes
		}
	}
	for banco, mes := range minimo {
		_, err := s.atualizar(ctx, sid, banco, func(d domain.Doc) error {
			if cur, _ := d["invalidoDe"].(string); cur == "" || mes < cur {
				d["invalidoDe"] = mes
			}
			return nil
		})
		if err != nil {
			// o lancamento ja foi gravado; sem isto o cache do banco pode ficar velho ate o proximo ajuste
			log.Printf("ERRO invalidar saldo sid=%s banco=%s mes=%s: %v", sid, banco, mes, err)
		}
	}
}

// ---------- rotas ----------

// Listar: GET /spaces/{sid}/fechamentos. Devolve o item de cada banco e o mes atual (fuso de Sao Paulo).
func (s *Fechamento) Listar(ctx context.Context, sid string) ([]domain.Doc, string, error) {
	itens, err := s.saldos.Todos(ctx, sid)
	if err != nil {
		log.Println("fechamentos:", err)
		return nil, "", ErrInterno
	}
	return itens, domain.MesAtual(s.rel.Agora()), nil
}

// SalvarCaches: PUT /spaces/{sid}/fechamentos. Grava o cache dos meses encerrados. Exige a version que o cliente leu ANTES
// de ler os lancamentos: se qualquer escrita aconteceu desde entao, 409 e o cliente recalcula.
func (s *Fechamento) SalvarCaches(ctx context.Context, sid string, body domain.Doc) (domain.Doc, error) {
	banco, _ := body["banco"].(string)
	if banco == "" {
		return nil, &ErroValidacao{"banco obrigatorio"}
	}
	ver, _ := body["version"].(float64)
	base, _ := body["base"].(string)
	bruto, _ := body["caches"].(map[string]any)
	mesAtual := domain.MesAtual(s.rel.Agora())
	caches := map[string]any{}
	for m, v := range bruto {
		n, ok := v.(float64)
		if !domain.MesValido(m) || !ok || math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, &ErroValidacao{"caches invalidos"}
		}
		if m < mesAtual { // o mes em andamento nunca e cache
			caches[m] = n
		}
	}
	d, err := s.saldos.Ler(ctx, sid, banco)
	if err != nil {
		log.Println("put fechamentos: ler:", err)
		return nil, ErrInterno
	}
	if versaoDe(d) != ver {
		return nil, conflito(d)
	}
	if d == nil {
		d = domain.Doc{"banco": banco}
	}
	d["caches"], d["base"] = caches, base
	delete(d, "invalidoDe")
	res, err := s.saldos.Salvar(ctx, sid, banco, d, ver)
	switch {
	case errors.Is(err, domain.ErrConflitoConfig):
		atual, _ := s.saldos.Ler(ctx, sid, banco)
		return nil, conflito(atual)
	case err != nil:
		log.Println("put fechamentos:", err)
		return nil, ErrSalvar
	}
	return res, nil
}

// Conferir: POST /spaces/{sid}/fechamentos/conferir. Registra a conferencia com o extrato e trava o mes (e os anteriores).
// O saldo calculado precisa bater com o extrato: a diferenca se resolve lancando o ajuste antes.
func (s *Fechamento) Conferir(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error) {
	banco, _ := body["banco"].(string)
	mes, _ := body["mes"].(string)
	saldo, ok1 := body["saldo"].(float64)
	real, ok2 := body["real"].(float64)
	ver, _ := body["version"].(float64)
	if banco == "" || !domain.MesValido(mes) || !ok1 || !ok2 {
		return nil, &ErroValidacao{"informe banco, mes (AAAA-MM), saldo calculado e saldo do extrato"}
	}
	agora := s.rel.Agora()
	if mes > domain.MesAtual(agora) {
		return nil, &ErroValidacao{"so da para conferir o mes atual ou meses anteriores"}
	}
	if math.Abs(saldo-real) >= 0.005 {
		return nil, &ErroValidacao{"o saldo calculado nao bate com o extrato: lance o ajuste antes de conferir"}
	}
	d, err := s.saldos.Ler(ctx, sid, banco)
	if err != nil {
		log.Println("conferir: ler:", err)
		return nil, ErrInterno
	}
	if versaoDe(d) != ver {
		return nil, conflito(d)
	}
	if u := domain.UltimoConferido(d); u != "" && mes <= u {
		return nil, &ErroOrdemFechamento{"o mes " + mes + " nao pode ser conferido: o ultimo conferido e " + u}
	}
	if d == nil {
		d = domain.Doc{"banco": banco}
	}
	conf, _ := d["conferidos"].(map[string]any)
	if conf == nil {
		conf = map[string]any{}
	}
	reg := map[string]any{"saldo": saldo, "real": real, "em": agora.UTC().Format("2006-01-02T15:04:05.000Z"), "por": sub}
	if aj, ok := body["ajuste"].(float64); ok && aj != 0 {
		reg["ajuste"] = aj
	}
	conf[mes] = reg
	d["conferidos"] = conf
	res, err := s.saldos.Salvar(ctx, sid, banco, d, ver)
	switch {
	case errors.Is(err, domain.ErrConflitoConfig):
		atual, _ := s.saldos.Ler(ctx, sid, banco)
		return nil, conflito(atual)
	case err != nil:
		log.Println("conferir:", err)
		return nil, ErrSalvar
	}
	return res, nil
}

var errNaoUltimo = errors.New("nao e o ultimo conferido")

// Reabrir: POST /spaces/{sid}/fechamentos/reabrir. Destrava so o ultimo mes conferido, para nunca sobrar conferencia por
// cima de mes aberto.
func (s *Fechamento) Reabrir(ctx context.Context, sid string, body domain.Doc) (domain.Doc, error) {
	banco, _ := body["banco"].(string)
	mes, _ := body["mes"].(string)
	if banco == "" || !domain.MesValido(mes) {
		return nil, &ErroValidacao{"informe banco e mes (AAAA-MM)"}
	}
	var naoUltimo string
	res, err := s.atualizar(ctx, sid, banco, func(d domain.Doc) error {
		u := domain.UltimoConferido(d)
		if u == "" {
			return ErrMesNaoConferido
		}
		if mes != u {
			naoUltimo = u
			return errNaoUltimo
		}
		conf, _ := d["conferidos"].(map[string]any)
		delete(conf, mes)
		d["conferidos"] = conf
		return nil
	})
	switch {
	case naoUltimo != "":
		return nil, &ErroOrdemFechamento{"reabra primeiro o mes " + naoUltimo + " (o mais recente conferido)"}
	case errors.Is(err, ErrMesNaoConferido):
		return nil, err
	case err != nil:
		log.Println("reabrir:", err)
		return nil, ErrSalvar
	}
	return res, nil
}
