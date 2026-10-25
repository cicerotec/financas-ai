package main

import (
	"financas/controller"
	"financas/repository"
	"financas/service"
)

// dynamoAPI e o que o codigo usa do DynamoDB; os testes trocam por um banco em memoria.
type dynamoAPI = repository.DynamoAPI

// Servidor e o conjunto montado uma vez, em main: um controller por assunto, ligado ao seu service e ao repository.
// E a unica ligacao das camadas; o roteador (roteador.go) so escolhe o controller de cada rota.
type Servidor struct {
	lanc    *controller.Lancamentos
	esp     *controller.Espacos
	cfg     *controller.Config
	perfil  *controller.Perfil
	fech    *controller.Fechamento
	avisos  *controller.Avisos
	cartoes *rotasCartao
}

// novoServidor liga repository, service e controller. Recebe o que vem de fora (banco, tabela, relogio, canal de envio e o
// ambiente, "dev" ou "prod"). E o unico lugar que faz essa ligacao.
func novoServidor(db dynamoAPI, table string, rel service.Relogio, notificador service.Notificador, ambiente string) *Servidor {
	cfgRepo := repository.NewConfig(db, table)
	pessoas := repository.NewPessoas(db, table)
	espacos := repository.NewEspacos(db, table)
	lancRepo := repository.NewLancamentos(db, table)
	tags := service.NewTags(cfgRepo)
	contador := service.NewContador(repository.NewContador(db, table))
	saldosRepo := repository.NewSaldos(db, table)
	fech := service.NewFechamento(saldosRepo, cfgRepo, rel)
	avisos := service.NewAvisos(repository.NewAvisos(db, table), lancRepo, cfgRepo, pessoas, espacos, notificador, rel, ambiente)
	efeitos := service.NewEfeitos(fech, tags, avisos, contador)
	lancSvc := service.NewLancamentos(lancRepo, efeitos, pessoas, rel)
	saldos := service.NewSaldos(lancRepo, cfgRepo, saldosRepo, rel)
	return &Servidor{
		lanc:    controller.NewLancamentos(lancSvc),
		esp:     controller.NewEspacos(service.NewEspacos(espacos)),
		cfg:     controller.NewConfig(service.NewConfig(cfgRepo), tags, contador),
		perfil:  controller.NewPerfil(service.NewPerfil(pessoas)),
		fech:    controller.NewFechamento(fech, saldos, service.NewConferencia(saldos, fech, lancSvc)),
		avisos:  controller.NewAvisos(avisos),
		cartoes: &rotasCartao{c: controller.NewCartoes(service.NewCartoes(lancRepo, cfgRepo, rel, lancSvc))},
	}
}
