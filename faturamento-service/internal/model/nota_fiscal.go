package model

import "time"

// StatusNota representa os dois estados possíveis de uma nota fiscal.
type StatusNota string

const (
	StatusAberta  StatusNota = "Aberta"
	StatusFechada StatusNota = "Fechada"
)

// NotaFiscal representa a linha da tabela `notas_fiscais`.
type NotaFiscal struct {
	ID        int64      `db:"id"`
	Numero    int        `db:"numero"`
	Status    StatusNota `db:"status"`
	CreatedAt time.Time  `db:"created_at"`
	FechadaEm *time.Time `db:"fechada_em"`
}
