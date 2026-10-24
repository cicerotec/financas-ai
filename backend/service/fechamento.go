package service

import (
	"context"
	"errors"
	"log"
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

// SalvarCaches: PUT /spaces/{sid}/fechamentos. **O cache do cliente nao e mais aceito** (item 40): quem calcula e grava o cache dos meses
// encerrados e o servidor (Saldos.gravarCache); um numero vindo do navegador nao vira ponto de partida da cascata de ninguem. A rota
// continua so para o app de hoje nao quebrar: responde 200 com o item como esta e nao grava nada. Some na Fase C.
func (s *Fechamento) SalvarCaches(ctx context.Context, sid string, body domain.Doc) (domain.Doc, error) {
	banco, _ := body["banco"].(string)
	if banco == "" {
		return nil, &ErroValidacao{"banco obrigatorio"}
	}
	d, err := s.saldos.Ler(ctx, sid, banco)
	if err != nil {
		log.Println("put fechamentos: ler:", err)
		return nil, ErrInterno
	}
	log.Printf("fechamentos: cache vindo do cliente ignorado sid=%s banco=%s", sid, banco)
	if d == nil {
		return domain.Doc{"banco": banco, "version": 0.0}, nil
	}
	return domain.Publico(d), nil
}

// RegistrarConferencia grava a conferencia de um mes com o saldo que o SERVIDOR calculou (e o extrato informado), e trava o mes e os
// anteriores. So confere a ordem e o mes; quem compara o extrato com o saldo e a Conferencia. ajuste e o lancamento de ajuste que
// ela criou (0 se nao criou). Tenta de novo se outra pessoa gravou o item no meio (atualizar).
func (s *Fechamento) RegistrarConferencia(ctx context.Context, sid, sub, banco, mes string, saldo, real, ajuste float64) (domain.Doc, error) {
	agora := s.rel.Agora()
	if !domain.MesValido(mes) {
		return nil, &ErroValidacao{"mes invalido (use AAAA-MM)"}
	}
	if mes > domain.MesAtual(agora) {
		return nil, &ErroValidacao{"so da para conferir o mes atual ou meses anteriores"}
	}
	res, err := s.atualizar(ctx, sid, banco, func(d domain.Doc) error {
		if u := domain.UltimoConferido(d); u != "" && mes <= u {
			return &ErroOrdemFechamento{"o mes " + mes + " nao pode ser conferido: o ultimo conferido e " + u}
		}
		conf, _ := d["conferidos"].(map[string]any)
		if conf == nil {
			conf = map[string]any{}
		}
		reg := map[string]any{"saldo": saldo, "real": real, "em": agora.UTC().Format("2006-01-02T15:04:05.000Z"), "por": sub}
		if ajuste != 0 {
			reg["ajuste"] = ajuste
		}
		conf[mes] = reg
		d["conferidos"] = conf
		return nil
	})
	var ordem *ErroOrdemFechamento
	switch {
	case errors.As(err, &ordem):
		return nil, err
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
