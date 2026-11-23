package domain

// ResumoAvisos e o conteudo de um aviso, sem formato: o dia, a marca do ambiente e as linhas (ou a marca de "mensagem de
// teste"). O service o entrega ao despachante; cada canal escolhe o texto (hoje os dois, o Telegram e o push, so a
// contagem: item 66 e item 54). Texto e limite de mensagem sao do canal, em adapter/notificacao/.
type ResumoAvisos struct {
	Hoje   string // AAAA-MM-DD
	Marca  string // MarcaAmbiente do servidor
	Linhas []LinhaAviso
	Teste  bool // a mensagem de teste do botao "Testar" (sem linhas)
	// Mensagem e um aviso de texto FIXO que nao e de lancamentos (seguranca da conta; spec mfa-opcional, R35). Sem linhas. Diz so o fato:
	// nunca codigo, e-mail nem lista. Cada canal escolhe como mostra (o Telegram, o texto; o aparelho, o corpo da notificacao).
	Mensagem string
}

// Frases fixas dos avisos de seguranca do MFA (spec mfa-opcional, R35). Dizem so o fato.
const (
	MensagemMFADesligado = "O MFA da sua conta foi desligado com um código de recuperação."
	MensagemListaGerada  = "Uma nova lista de códigos de recuperação foi gerada." // nao diz que o MFA foi ligado: nesse ponto ele ainda nao esta
)

// ContagemAvisos e tudo o que um canal que NAO mostra os detalhes (o push) pode ler de um resumo: so inteiros, a marca do ambiente e se
// e teste. Descricao, valor, meio de pagamento e data ficam de fora de proposito (decisao do dono, 2026-11-13: a notificacao aparece na
// tela bloqueada). Entradas (a receber) contam na situacao do vencimento, como as saidas. Os a vencer se dividem em Amanha (falta 1 dia) e
// Proximos (os demais): "1 a vencer" sozinho nao diz se e amanha ou daqui a uma semana, e "amanha" nao expoe a data.
type ContagemAvisos struct {
	Atrasados, Hoje, Amanha, Proximos int
	Marca                             string
	Teste                             bool
}

// Contagens conta as linhas do resumo por situacao do vencimento.
func Contagens(r ResumoAvisos) ContagemAvisos {
	c := ContagemAvisos{Marca: r.Marca, Teste: r.Teste}
	for _, l := range r.Linhas {
		switch l.Kind {
		case AvisoAtrasado:
			c.Atrasados++
		case AvisoHoje:
			c.Hoje++
		default:
			if l.N == 1 {
				c.Amanha++
			} else {
				c.Proximos++
			}
		}
	}
	return c
}
