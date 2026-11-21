package domain

// Espaco e um espaco (familia) a que a pessoa pertence, com o papel dela e as preferencias que ficam no vinculo
// USER#<sub> / SPACE#<sid>.
type Espaco struct {
	ID             string
	Nome           string // o do META do espaco; se nao houver, o proprio id
	Role           any
	Apelido        any // como veio do banco (texto ou ausente): a resposta HTTP repete o que existe hoje
	TelegramChatID any
	// Canais diz, por canal, se o vinculo tem destino (spec telegram-vinculo, R14): chat id nao vazio, ao menos um aparelho. So booleanos: o front
	// so exibe, nunca decide quem recebe.
	Canais CanaisLigados
}

// CanaisLigados e o estado dos canais de aviso de uma pessoa num espaco.
type CanaisLigados struct{ Telegram, Push bool }
