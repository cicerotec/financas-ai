package main

import (
	"context"
	"errors"

	"financas/controller"
	"financas/domain"
	"financas/repository"
	"financas/service"
)

// dynamoAPI e o que o codigo usa do DynamoDB; os testes trocam por um banco em memoria.
type dynamoAPI = repository.DynamoAPI

// Servidor e o conjunto montado uma vez, em main: um controller por assunto, ligado ao seu service e ao repository.
// E a unica ligacao das camadas; o roteador (roteador.go) so escolhe o controller de cada rota.
type Servidor struct {
	lanc      *controller.Lancamentos
	esp       *controller.Espacos
	cfg       *controller.Config
	perfil    *controller.Perfil
	fech      *controller.Fechamento
	avisos    *controller.Avisos
	aparelhos *controller.Aparelhos
	rec       *controller.Recorrencias
	consultas *controller.Consultas
	cartoes   *rotasCartao
	meios     *controller.Meios
	combos    *controller.Combos
	estadoMFA *controller.EstadoMFA            // GET /me/mfa: o estado do MFA e a lista de codigos, lidos do login e do banco (spec mfa-opcional, R37)
	codigos   *controller.CodigosDeRecuperacao // lista de codigos de recuperacao do MFA (gerar e confirmar; spec mfa-opcional)
	recup     *controller.Recuperacao          // rota PUBLICA: recuperacao do MFA (auth.RotaPublica; spec mfa-opcional)

	repoCodigos *repository.CodigosDeRecuperacao // para religar a recuperacao a outra identidade (usarIdentidade)
	relogio     service.Relogio
	segAvisos   *service.AvisosDeSeguranca
}

// novoServidor liga repository, service e controller. Recebe o que vem de fora (banco, tabela, relogio, despachante de avisos e o
// ambiente, "dev" ou "prod"). E o unico lugar que faz essa ligacao.
func novoServidor(db dynamoAPI, table string, rel service.Relogio, despachante *service.Despachante, ambiente string) *Servidor {
	cfgRepo := repository.NewConfig(db, table)
	pessoas := repository.NewPessoas(db, table)
	espacos := repository.NewEspacos(db, table)
	lancRepo := repository.NewLancamentos(db, table)
	tags := service.NewTags(cfgRepo)
	contador := service.NewContador(repository.NewContador(db, table))
	saldosRepo := repository.NewSaldos(db, table)
	fech := service.NewFechamento(saldosRepo, cfgRepo, rel)
	avisos := service.NewAvisos(repository.NewAvisos(db, table), lancRepo, cfgRepo, pessoas, espacos, despachante, pessoas, rel, ambiente)
	efeitos := service.NewEfeitos(fech, tags, avisos, contador)
	lancSvc := service.NewLancamentos(lancRepo, efeitos, pessoas, rel)
	saldos := service.NewSaldos(lancRepo, cfgRepo, saldosRepo, rel)
	cartoes := service.NewCartoes(lancRepo, cfgRepo, rel, lancSvc)
	lancSvc.UsarCartoes(cfgRepo, cartoes)
	cartoes.UsarAvisos(avisos)
	avisos.UsarFaturas(cartoes)
	recorrencias := service.NewRecorrencias(repository.NewAvisos(db, table), lancSvc, rel)
	lancSvc.UsarRecorrencias(recorrencias)
	avisos.UsarRecorrencias(recorrencias)
	encerramento := service.NewEncerramento(saldos, lancRepo, cfgRepo, repository.NewAvisos(db, table), lancSvc, cartoes, rel)
	combos := service.NewCombos(repository.NewAvisos(db, table), cfgRepo, lancRepo, rel)
	segAvisos := service.NewAvisosDeSeguranca(espacos, pessoas, despachante, ambiente) // aviso de mudanca do MFA (spec mfa-opcional, R35)
	cfgSvc := service.NewConfig(cfgRepo, rel)
	cfgSvc.UsarRotinaCombos(combos)
	s := &Servidor{
		lanc:      controller.NewLancamentos(lancSvc),
		esp:       controller.NewEspacos(service.NewEspacos(espacos)),
		cfg:       controller.NewConfig(cfgSvc, tags, contador),
		perfil:    controller.NewPerfil(service.NewPerfil(pessoas)),
		fech:      controller.NewFechamento(fech, saldos, service.NewConferencia(saldos, fech, lancSvc)),
		avisos:    controller.NewAvisos(avisos),
		aparelhos: controller.NewAparelhos(service.NewAparelhos(pessoas, espacos)),
		rec:       controller.NewRecorrencias(recorrencias),
		consultas: controller.NewConsultas(service.NewConsultas(repository.NewConsultas(db, table), pessoas, rel)),
		cartoes:   &rotasCartao{c: controller.NewCartoes(cartoes)},
		meios:     controller.NewMeios(service.NewMeios(cfgRepo, lancRepo, rel, encerramento)),
		combos:    controller.NewCombos(combos),
		estadoMFA: controller.NewEstadoMFA(service.NewEstadoMFA(identidadeIndisponivel{}, repository.NewCodigosDeRecuperacao(db, table))),
		codigos:   controller.NewCodigosDeRecuperacao(service.NewCodigosDeRecuperacao(repository.NewCodigosDeRecuperacao(db, table), segAvisos, rel)),
		recup:     controller.NewRecuperacao(service.NewRecuperacao(identidadeIndisponivel{}, repository.NewCodigosDeRecuperacao(db, table), segAvisos, rel)),
	}
	s.repoCodigos, s.relogio, s.segAvisos = repository.NewCodigosDeRecuperacao(db, table), rel, segAvisos
	return s
}

// usarIdentidade liga a recuperacao do MFA ao login (o Cognito, no main; um falso, nos testes). Ate la toda recuperacao falha.
func (s *Servidor) usarIdentidade(id service.Identidade) {
	s.recup = controller.NewRecuperacao(service.NewRecuperacao(id, s.repoCodigos, s.segAvisos, s.relogio))
	s.estadoMFA = controller.NewEstadoMFA(service.NewEstadoMFA(id, s.repoCodigos))
}

// identidadeIndisponivel e a identidade de antes de o main ligar o Cognito: nao acha ninguem, entao a recuperacao so responde a falha.
type identidadeIndisponivel struct{}

func (identidadeIndisponivel) BuscarPessoa(context.Context, string) (domain.PessoaMFA, bool, error) {
	return domain.PessoaMFA{}, false, nil
}
func (identidadeIndisponivel) DesligarMFA(context.Context, string) error {
	return errors.New("identidade indisponivel")
}
