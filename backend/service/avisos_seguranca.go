package service

import (
	"context"
	"log"
	"time"

	"financas/domain"
)

// Avisador e o que a recuperacao e a lista de codigos pedem para avisar a pessoa de uma mudanca de seguranca (spec mfa-opcional, R35).
// O aviso e de melhor esforco: nunca falha a operacao.
type Avisador interface {
	Avisar(ctx context.Context, sub, mensagem string)
}

// esperaMaximaDoAviso e quanto a operacao espera o envio antes de responder (o aviso nunca atrasa a resposta alem disso). Na Lambda o
// que ainda estiver em curso depois da resposta pode nao terminar: o aviso e de melhor esforco.
const esperaMaximaDoAviso = 1500 * time.Millisecond

// AvisosDeSeguranca avisa a pessoa, pelo canal que ela JA tem ligado (o que o despachante conhece), de que o MFA dela mudou. So duas frases
// fixas (domain.MensagemMFADesligado e domain.MensagemListaGerada): nunca codigo, e-mail nem lista. O servico trata o canal como valor opaco.
type AvisosDeSeguranca struct {
	espacos     RepositorioEspacos
	contatos    LeitorContatos
	despachante *Despachante
	ambiente    string
}

func NewAvisosDeSeguranca(espacos RepositorioEspacos, contatos LeitorContatos, despachante *Despachante, ambiente string) *AvisosDeSeguranca {
	return &AvisosDeSeguranca{espacos: espacos, contatos: contatos, despachante: despachante, ambiente: ambiente}
}

// Avisar junta os destinos da pessoa em todos os espacos dela (um aviso por destino, sem repetir), entrega a frase e volta. Sem destino
// ligado nada e enviado, e isso nao e erro. Espera no maximo esperaMaximaDoAviso. Nunca loga destino, texto nem sub.
func (a *AvisosDeSeguranca) Avisar(ctx context.Context, sub, mensagem string) {
	if a == nil || a.despachante == nil || sub == "" || mensagem == "" {
		return
	}
	feito := make(chan struct{})
	go func() {
		defer close(feito)
		a.enviar(ctx, sub, mensagem)
	}()
	select {
	case <-feito:
	case <-time.After(esperaMaximaDoAviso):
	}
}

func (a *AvisosDeSeguranca) enviar(ctx context.Context, sub, mensagem string) {
	espacos, err := a.espacos.EspacosDe(ctx, sub)
	if err != nil {
		log.Println("aviso de seguranca: listar espacos falhou")
		return
	}
	vistos := map[Destino]bool{}
	var destinos []Destino
	for _, e := range espacos {
		porCanal, err := a.contatos.Destinos(ctx, e.ID, sub)
		if err != nil {
			log.Println("aviso de seguranca: ler destinos falhou")
			continue
		}
		for canal, enderecos := range porCanal {
			for _, end := range enderecos {
				d := Destino{Canal: canal, Endereco: end}
				if !vistos[d] { // os destinos iguais de varios espacos recebem um aviso so
					vistos[d] = true
					destinos = append(destinos, d)
				}
			}
		}
	}
	resumo := domain.ResumoAvisos{Marca: domain.MarcaAmbiente(a.ambiente), Mensagem: mensagem}
	falhas := 0
	for _, d := range destinos {
		if err := a.despachante.Entregar(ctx, d, resumo); err != nil {
			falhas++
		}
	}
	if falhas > 0 {
		log.Printf("aviso de seguranca: %d de %d entregas falharam", falhas, len(destinos)) // so contagens
	}
}
