package main

import (
	"financas/controller"
	"financas/repository"
	"financas/service"
)

// Servidor e o conjunto montado uma vez, em main: os controllers de cada assunto ja migrado para as camadas, ligados
// ao repository e ao service. Os assuntos que ainda moram na raiz (avisos e Telegram) seguem chamando as suas
// funcoes; migram na fatia 4 do item 32 do BACKLOG.
type Servidor struct {
	lanc   *controller.Lancamentos
	esp    *controller.Espacos
	cfg    *controller.Config
	perfil *controller.Perfil
	fech   *controller.Fechamento
}

// novoServidor liga repository, service e controller. E o unico lugar que faz essa ligacao.
func novoServidor(db dynamoAPI, table string, rel service.Relogio) *Servidor {
	cfgRepo := repository.NewConfig(db, table)
	pessoas := repository.NewPessoas(db, table)
	tags := service.NewTags(cfgRepo)
	contador := service.NewContador(repository.NewContador(db, table))
	fech := service.NewFechamento(repository.NewSaldos(db, table), cfgRepo, rel)
	return &Servidor{
		lanc:   controller.NewLancamentos(service.NewLancamentos(repository.NewLancamentos(db, table), efeitosRaiz{tags: tags, contador: contador, fech: fech}, pessoas)),
		esp:    controller.NewEspacos(service.NewEspacos(repository.NewEspacos(db, table))),
		cfg:    controller.NewConfig(service.NewConfig(cfgRepo), tags, contador),
		perfil: controller.NewPerfil(service.NewPerfil(pessoas)),
		fech:   controller.NewFechamento(fech),
	}
}
