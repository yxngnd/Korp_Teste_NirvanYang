// Package model contém as structs que espelham as tabelas do banco de dados.
// Não devem ser expostas diretamente pela API — para isso existem os DTOs.
package model

import "time"

// Produto representa a linha da tabela `produtos`.
type Produto struct {
	ID        int64     `db:"id"`
	Codigo    string    `db:"codigo"`
	Descricao string    `db:"descricao"`
	Saldo     int       `db:"saldo"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
