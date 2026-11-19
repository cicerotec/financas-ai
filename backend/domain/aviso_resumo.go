package domain

// ResumoAvisos e o conteudo de um aviso, sem formato: o dia, a marca do ambiente e as linhas (ou a marca de "mensagem de
// teste"). O service o entrega ao despachante; cada canal escolhe o texto (o Telegram, o resumo com cada linha; o push da
// fase 4 do item 54, so a contagem). Texto, limite e divisao de mensagem sao do canal, em adapter/notificacao/.
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
// tela bloqueada). Entradas (a receber) contam na situacao do vencimento, como as saidas.
type ContagemAvisos struct {
	Atrasados, Hoje, AVencer int
	Marca                    string
	Teste                    bool
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
			c.AVencer++
		}
	}
	return c
}
