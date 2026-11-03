package domain

import "time"

// Meios de pagamento (item 43): o `banco` gravado em cada lancamento. O dono desativa um meio a partir de uma data (CFG#LISTAS.ativoAte, um mapa
// nome -> AAAA-MM-DD, ausente = ativo) sem tocar no passado.

// DataValida diz se s e uma data AAAA-MM-DD (so o formato de 10 caracteres, sem hora).
func DataValida(s string) bool {
	if len(s) != len("2006-01-02") {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// HojeDe devolve o dia (AAAA-MM-DD) do instante dado, no fuso do app.
func HojeDe(t time.Time) string { return t.In(FusoApp).Format("2006-01-02") }

// MeioAtivo diz se o meio ainda e sugerido no dia hoje: sem ativoAte ou antes da data em que ele deixa de valer.
func MeioAtivo(ativoAte, hoje string) bool { return ativoAte == "" || hoje < ativoAte }
