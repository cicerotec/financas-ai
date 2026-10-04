package main

import (
	"context"
	"errors"
	"log"
)

// Tags: a lista compartilhada (CFG#TAGS, campo "usadas") so cresce por aqui. O member nao
// consegue reescrever a lista (PUT em cfg/tags e so do owner), mas pode acrescentar tags novas.

func tagsDe(d doc) []string {
	var t []string
	if l, ok := d["tags"].([]any); ok {
		for _, x := range l {
			if s, ok := x.(string); ok {
				t = append(t, s)
			}
		}
	}
	return t
}

// adicionarTags acrescenta ao CFG#TAGS apenas o que ainda nao existe. Devolve o que foi adicionado.
func adicionarTags(ctx context.Context, sid string, novas []string) ([]string, error) {
	pk, sk := spacePK(sid), "CFG#TAGS"
	for tentativa := 0; tentativa < 3; tentativa++ {
		atual, err := getItem(ctx, pk, sk)
		if err != nil {
			return nil, err
		}
		var existentes []string
		versao := 0.0
		if atual != nil {
			if l, ok := atual["usadas"].([]any); ok {
				for _, x := range l {
					if s, ok := x.(string); ok {
						existentes = append(existentes, s)
					}
				}
			}
			versao, _ = atual["version"].(float64)
		}
		falta := faltantes(novas, existentes)
		if len(falta) == 0 {
			return nil, nil
		}
		todas := make([]any, 0, len(existentes)+len(falta))
		for _, t := range append(existentes, falta...) {
			todas = append(todas, t)
		}
		_, err = salvarCfg(ctx, pk, sk, doc{"usadas": todas}, versao)
		if errors.Is(err, errConflito) {
			continue // alguem gravou ao mesmo tempo: le de novo e tenta outra vez
		}
		if err != nil {
			return nil, err
		}
		return falta, nil
	}
	return nil, errors.New("conflito ao gravar tags")
}

// garantirTagsServidor e chamado ao salvar um lancamento: tags novas entram na lista sozinhas.
// Falha aqui nao derruba o salvamento do lancamento (so registra).
func garantirTagsServidor(ctx context.Context, sid string, tags []string) {
	novas := normalizarTags(tags)
	if len(novas) == 0 {
		return
	}
	if _, err := adicionarTags(ctx, sid, novas); err != nil {
		log.Println("garantir tags:", err)
	}
}

func postTags(ctx context.Context, sid string, body doc) (resp, error) {
	novas := normalizarTags(tagsDe(body))
	if len(novas) == 0 {
		return erro(400, "informe ao menos uma tag")
	}
	adicionadas, err := adicionarTags(ctx, sid, novas)
	if err != nil {
		log.Println("post tags:", err)
		return erro(500, "erro ao salvar tags")
	}
	if adicionadas == nil {
		adicionadas = []string{}
	}
	return out(200, doc{"adicionadas": adicionadas})
}
