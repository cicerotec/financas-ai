package domain

// ResumoAvisos e o conteudo de um aviso, sem formato: o dia, a marca do ambiente e as linhas (ou a marca de "mensagem de
// teste"). O service o entrega ao despachante; cada canal escolhe o texto (o Telegram, o resumo com cada linha; o push da
// fase 4 do item 54, so a contagem). Texto, limite e divisao de mensagem sao do canal, em adapter/notificacao/.
type ResumoAvisos struct {
	Hoje   string // AAAA-MM-DD
	Marca  string // MarcaAmbiente do servidor
	Linhas []LinhaAviso
	Teste  bool // a mensagem de teste do botao "Testar" (sem linhas)
}
