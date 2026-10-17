package domain

// Espaco e um espaco (familia) a que a pessoa pertence, com o papel dela e as preferencias que ficam no vinculo
// USER#<sub> / SPACE#<sid>.
type Espaco struct {
	ID             string
	Nome           string // o do META do espaco; se nao houver, o proprio id
	Role           any
	Apelido        any // como veio do banco (texto ou ausente): a resposta HTTP repete o que existe hoje
	TelegramChatID any
}
