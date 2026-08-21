// Package db cuida da abertura e configuração do pool de conexões com o Postgres.
package db

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yxngnd/estoque-service/internal/config"

	_ "github.com/lib/pq" // driver Postgres, registrado via side-effect import
)

// NewPostgresConnection abre o pool de conexões e testa a conectividade com Ping.
// Entrada: configuração com dados de conexão.
// Saída: *sqlx.DB pronto para uso, ou erro se não conseguir conectar.
func NewPostgresConnection(cfg *config.Config) (*sqlx.DB, error) {
	dbConn, err := sqlx.Connect("postgres", cfg.PostgresDSN())
	if err != nil {
		return nil, fmt.Errorf("db: falha ao conectar no postgres: %w", err)
	}

	// Limites de pool conservadores, adequados a um microsserviço pequeno.
	dbConn.SetMaxOpenConns(20)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(30 * time.Minute)

	return dbConn, nil
}
