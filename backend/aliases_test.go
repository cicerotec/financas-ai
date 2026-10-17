package main

import (
	"financas/auth"
	"financas/domain"
)

// Atalhos dos testes: nomes curtos para o que mora em domain/ e auth/ (a raiz de producao nao tem mais pontes).
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

	decidirAviso = domain.DecidirAviso
	diaBR        = domain.DiaBR
	moeda        = domain.Moeda
)

type kindAviso = domain.KindAviso

const (
	semAviso      = domain.SemAviso
	avisoAntes    = domain.AvisoAntes
	avisoHoje     = domain.AvisoHoje
	avisoAtrasado = domain.AvisoAtrasado
)

// pkAvisos e a particao do item auxiliar de cada lancamento com aviso (repository/avisos.go).
const pkAvisos = "AVISOS#ATIVOS"
