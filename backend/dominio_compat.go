package main

import (
	"financas/auth"
	"financas/domain"
)

// Compatibilidade transitoria: o que ja mora em domain/ e auth/ e ainda tem uso na raiz (fechamento, tags, avisos,
// perfil, seq e os testes). Cada nome sai daqui quando a fatia que o usa migrar de pasta (item 32 do BACKLOG).

type (
	doc  = domain.Doc
	user = auth.User
)

var (
	fusoApp      = domain.FusoApp
	validarAviso = domain.ValidarAviso
	validarSerie = domain.ValidarSerie
	protegidos   = domain.Protegidos
	publico      = domain.Publico
	mesDe        = domain.MesDe
	errConflito  = domain.ErrConflitoConfig
)
