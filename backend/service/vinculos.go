package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"financas/domain"
)

// RepositorioVinculos e o que o service precisa do banco para o codigo de vinculo de um canal (spec do vinculo por link, R1 e R6; AD-11).
// So o repository sabe onde ele fica e em que campo do vinculo o endereco e gravado.
type RepositorioVinculos interface {
	// SalvarPendente grava o codigo pendente da pessoa neste canal, substituindo o anterior.
	SalvarPendente(ctx context.Context, sub, canal string, p domain.VinculoPendente) error
	// LerPendente le o codigo pendente (so leitura), ou nil se nao ha.
	LerPendente(ctx context.Context, sub, canal string) (*domain.VinculoPendente, error)
	// ConsumirPendente apaga o codigo, de forma condicional ao hash e a validade, e o devolve; nil se ja nao vale.
	ConsumirPendente(ctx context.Context, sub, canal, hash string, agora time.Time) (*domain.VinculoPendente, error)
	// LigarEndereco grava o endereco no vinculo (sub, sid) e devolve o apelido e o endereco de antes. domain.ErrVinculoNaoEncontrado sem vinculo.
	LigarEndereco(ctx context.Context, sid, sub, canal, endereco string) (domain.EnderecoLigado, error)
}

var (
	// ErrVinculoInvalido: o codigo esta fora do formato, nao existe, venceu, ja foi usado ou nao e deste vinculo. O webhook nao distingue (R9).
	ErrVinculoInvalido = errors.New("codigo de vinculo invalido")
	// ErrTentativasDemais: o mesmo endereco tentou demais na hora. A rota ignora, sem resposta (R6).
	ErrTentativasDemais = errors.New("tentativas demais deste endereco")
	// ErrLimiteDeLinks: a pessoa pediu links demais na hora (R3). A rota responde 429.
	ErrLimiteDeLinks = errors.New("links demais na ultima hora")
	// ErrNaoLigou: o codigo era valido e foi consumido, mas o endereco nao foi gravado (vinculo apagado, erro de gravacao).
	ErrNaoLigou = errors.New("nao consegui ligar")
)

const (
	// MaxLinksPorHora e MaxTentativasPorHora sao os limites da D5: gerar, por pessoa; usar, por endereco (o chat de quem escreve).
	MaxLinksPorHora      = 5
	MaxTentativasPorHora = 10
	janelaDoVinculo      = time.Hour
)

// Vinculos gera o codigo de vinculo e o consome quando o canal o devolve com o endereco provado. Trata o canal como valor opaco: nao sabe como o canal
// entrega o codigo, que URL a pessoa toca nem que segredo protege a rota (AD-2 e AD-11); isso e do controller e do adaptador. O sub vem sempre do token de quem pede o link.
type Vinculos struct {
	repo     RepositorioVinculos
	rel      Relogio
	fonte    io.Reader // bytes aleatorios; crypto/rand em producao
	gerar    *limiteDeJanela
	tentadas *limiteDeJanela
}

func NewVinculos(repo RepositorioVinculos, rel Relogio) *Vinculos {
	return &Vinculos{repo: repo, rel: rel, fonte: rand.Reader,
		gerar: newLimiteDeJanela(MaxLinksPorHora, janelaDoVinculo), tentadas: newLimiteDeJanela(MaxTentativasPorHora, janelaDoVinculo)}
}

// UsarFonte troca a fonte de bytes aleatorios (so para os testes serem deterministicos).
func (s *Vinculos) UsarFonte(f io.Reader) { s.fonte = f }

// Gerar cria o codigo de vinculo da pessoa neste canal, substituindo o pendente anterior, e o devolve com o momento em que vence. O codigo em
// texto so existe no retorno: nunca e gravado nem logado. ErrLimiteDeLinks na 6a geracao da hora; ErrInterno se o sub nao e um UUID (nada e gravado).
func (s *Vinculos) Gerar(ctx context.Context, sid, sub, canal string) (codigo string, expiraEm time.Time, err error) {
	agora := s.rel.Agora()
	if !s.gerar.Permite(sub, agora) {
		return "", time.Time{}, ErrLimiteDeLinks
	}
	codigo, hash, err := domain.NovoCodigoVinculo(sub, s.fonte)
	if err != nil {
		log.Println("vinculo: sortear o codigo:", err) // o erro nunca leva o sub nem o codigo
		return "", time.Time{}, ErrInterno
	}
	expira := agora.Add(domain.ValidadeVinculo)
	if err := s.repo.SalvarPendente(ctx, sub, canal, domain.VinculoPendente{Sid: sid, Hash: hash, ExpiraEm: expira, CriadoEm: agora}); err != nil {
		log.Println("vinculo: gravar o codigo:", err)
		return "", time.Time{}, ErrSalvar
	}
	return codigo, expira, nil
}

// Ligar usa o codigo que o canal devolveu: grava o endereco no vinculo de quem gerou o link. A ordem vai da checagem mais barata a mais cara (R6):
// formato, limite por endereco, leitura (sem escrita: codigo que nao existe nao gasta gravacao), consumo condicional e so entao a gravacao. O codigo
// e consumido ANTES de gravar, de proposito: nao vale duas vezes, mesmo sob concorrencia. Nenhum log leva sub, sid, codigo nem endereco.
func (s *Vinculos) Ligar(ctx context.Context, canal, codigo, endereco string) (domain.EnderecoLigado, error) {
	sub, hash, ok := domain.LerCodigoVinculo(codigo)
	if !ok {
		return domain.EnderecoLigado{}, ErrVinculoInvalido
	}
	agora := s.rel.Agora()
	if !s.tentadas.Permite(endereco, agora) {
		return domain.EnderecoLigado{}, ErrTentativasDemais
	}
	pend, err := s.repo.LerPendente(ctx, sub, canal)
	if err != nil {
		log.Println("vinculo: ler o codigo:", err)
		return domain.EnderecoLigado{}, ErrNaoLigou
	}
	if pend == nil || !agora.Before(pend.ExpiraEm) || subtle.ConstantTimeCompare([]byte(pend.Hash), []byte(hash)) != 1 {
		return domain.EnderecoLigado{}, ErrVinculoInvalido
	}
	usado, err := s.repo.ConsumirPendente(ctx, sub, canal, pend.Hash, agora)
	if err != nil {
		log.Println("vinculo: consumir o codigo:", err)
		return domain.EnderecoLigado{}, ErrNaoLigou
	}
	if usado == nil { // outra execucao o consumiu, ou a pessoa gerou outro, entre a leitura e a exclusao
		return domain.EnderecoLigado{}, ErrVinculoInvalido
	}
	ligado, err := s.repo.LigarEndereco(ctx, usado.Sid, sub, canal, endereco)
	if err != nil {
		if !errors.Is(err, domain.ErrVinculoNaoEncontrado) {
			log.Println("vinculo: gravar o endereco:", err)
		}
		return domain.EnderecoLigado{}, ErrNaoLigou
	}
	return ligado, nil
}

// limiteDeJanela conta eventos por chave numa janela deslizante, em memoria da Lambda: de melhor esforco, zera na partida a frio (R3 e R6).
type limiteDeJanela struct {
	mu      sync.Mutex
	max     int
	janela  time.Duration
	eventos map[string][]time.Time
}

const tetoChavesDoLimite = 10000

func newLimiteDeJanela(max int, janela time.Duration) *limiteDeJanela {
	return &limiteDeJanela{max: max, janela: janela, eventos: map[string][]time.Time{}}
}

// Permite conta o evento e diz se ainda cabe na janela. O evento que passa do limite nao e contado (quem insiste nao se prende mais).
func (l *limiteDeJanela) Permite(chave string, agora time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.eventos) >= tetoChavesDoLimite {
		if _, ja := l.eventos[chave]; !ja {
			l.limpar(agora)
		}
	}
	recentes := l.eventos[chave][:0]
	for _, t := range l.eventos[chave] {
		if agora.Sub(t) < l.janela {
			recentes = append(recentes, t)
		}
	}
	if len(recentes) >= l.max {
		l.eventos[chave] = recentes
		return false
	}
	l.eventos[chave] = append(recentes, agora)
	return true
}

func (l *limiteDeJanela) limpar(agora time.Time) {
	for k, ts := range l.eventos {
		if len(ts) == 0 || agora.Sub(ts[len(ts)-1]) >= l.janela {
			delete(l.eventos, k)
		}
	}
	if len(l.eventos) >= tetoChavesDoLimite { // ainda cheio de chaves correntes: recomeca (melhor esforco)
		l.eventos = map[string][]time.Time{}
	}
}
