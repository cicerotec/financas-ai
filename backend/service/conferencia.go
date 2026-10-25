package service

import (
	"context"
	"log"
	"math"

	"financas/domain"
)

// CriadorDeLancamento e o que a conferencia usa para registrar o lancamento de ajuste: o mesmo caminho de escrita de qualquer
// lancamento (com as travas de mes e o historico). Quem implementa e o service de lancamentos.
type CriadorDeLancamento interface {
	Criar(ctx context.Context, sid, sub string, body domain.Lancamento, confirma bool) (domain.Lancamento, error)
}

// Conferencia confere um mes de um meio com o extrato (item 40). O cliente manda so o saldo do extrato (real); o servidor compara com
// o saldo que ele mesmo calculou. Com diferenca, a pessoa escolhe entre cancelar e pedir o ajuste, que o servidor cria.
type Conferencia struct {
	saldos *Saldos
	fech   *Fechamento
	lanc   CriadorDeLancamento
}

func NewConferencia(saldos *Saldos, fech *Fechamento, lanc CriadorDeLancamento) *Conferencia {
	return &Conferencia{saldos: saldos, fech: fech, lanc: lanc}
}

// Conferir: POST /spaces/{sid}/fechamentos/conferir com { meio (ou banco), mes, real, ajustar? }. O saldo vindo do cliente e recusado (400);
// version e caches, se vierem, sao ignorados.
func (s *Conferencia) Conferir(ctx context.Context, sid, sub string, body domain.Doc) (domain.Doc, error) {
	if _, veio := body["saldo"]; veio {
		return nil, &ErroValidacao{"o saldo e calculado pelo servidor: informe so o extrato (real)"}
	}
	banco, _ := body["meio"].(string)
	if banco == "" {
		banco, _ = body["banco"].(string)
	}
	mes, _ := body["mes"].(string)
	real, temReal := body["real"].(float64)
	ajustar, _ := body["ajustar"].(bool)
	if banco == "" || !domain.MesValido(mes) || !temReal || math.IsNaN(real) || math.IsInf(real, 0) {
		return nil, &ErroValidacao{"informe meio, mes (AAAA-MM) e o saldo do extrato (real)"}
	}
	if mes > domain.MesAtual(s.fech.rel.Agora()) {
		return nil, &ErroValidacao{"so da para conferir o mes atual ou meses anteriores"}
	}
	calc, err := s.saldos.Calcular(ctx, sid, banco)
	if err != nil {
		return nil, err
	}
	// mes ja conferido (ou anterior ao ultimo conferido): nao ha o que conferir e o ajuste seria recusado; a ordem manda
	if u := domain.UltimoConferido(calc.Item); u != "" && mes <= u {
		if ajustar {
			return nil, &domain.ErroMesConferido{Banco: banco, Mes: mes, Ate: u}
		}
		return nil, &ErroOrdemFechamento{"o mes " + mes + " nao pode ser conferido: o ultimo conferido e " + u}
	}
	saldo, ok := calc.Mapa[mes]
	if !ok {
		return nil, &ErroValidacao{"mes fora do periodo calculado (anterior ao inicio do controle)"}
	}
	ajuste := 0.0
	if diff := domain.Arred(real - saldo); math.Abs(diff) >= 0.005 {
		if !ajustar {
			return nil, &ErroDiferenca{Calculado: saldo, Real: real, Diferenca: diff}
		}
		if err := s.criarAjuste(ctx, sid, sub, banco, mes, diff); err != nil {
			return nil, err
		}
		ajuste = diff
		calc, err = s.saldos.Calcular(ctx, sid, banco)
		if err != nil {
			return nil, err
		}
		saldo = calc.Mapa[mes]
		if novo := domain.Arred(real - saldo); math.Abs(novo) >= 0.005 {
			// alguem alterou lancamentos no meio: o ajuste ficou gravado; repetir a conferencia nao cria outro
			log.Printf("conferencia: o ajuste de %s %s ficou gravado, mas o saldo mudou no meio (diferenca %.2f)", banco, mes, novo)
			return nil, &ErroDiferenca{Calculado: saldo, Real: real, Diferenca: novo}
		}
	}
	return s.fech.RegistrarConferencia(ctx, sid, sub, banco, mes, saldo, real, ajuste)
}

// criarAjuste registra o lancamento de ajuste do mes: o ultimo dia as 12:00 em Sao Paulo, entrada se o extrato e maior que o
// calculado, saida se e menor. Pedir o ajuste vale como a confirmacao de mes fechado por data (item 38); o mes conferido segue
// absoluto (a trava recusa).
func (s *Conferencia) criarAjuste(ctx context.Context, sid, sub, banco, mes string, diff float64) error {
	status := domain.StatusDeAjuste()
	if status == "" {
		return &ErroValidacao{"nenhum status mexe no saldo: configure em Listas antes de criar o ajuste"}
	}
	data, _ := domain.UltimoDiaDoMes(mes)
	tipo := "saida"
	if diff > 0 {
		tipo = "entrada"
	}
	_, err := s.lanc.Criar(ctx, sid, sub, domain.Lancamento{
		"dataEvento": data, "status": status, "tipo": tipo, "valor": domain.Arred(math.Abs(diff)), "banco": banco,
		"descricao": "Ajuste de conferência " + mes, "nota": "Criado ao conferir o mês com o extrato do banco.",
		"tags": []any{}, "excluirDoTotal": false,
	}, true)
	return err
}
