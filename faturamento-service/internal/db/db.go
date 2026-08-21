package db

import (
	"fmt"
	"time"

	"github.com/candidato/faturamento-service/internal/config"
	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

// NewPostgresConnection abre o pool de conexões e testa a conectividade com Ping.
func NewPostgresConnection(cfg *config.Config) (*sqlx.DB, error) {
	dbConn, err := sqlx.Connect("postgres", cfg.PostgresDSN())
	if err != nil {
		return nil, fmt.Errorf("db: falha ao conectar no postgres: %w", err)
	}

	dbConn.SetMaxOpenConns(20)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(30 * time.Minute)

	return dbConn, nil
}
