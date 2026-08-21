package db

import (
	"errors"
	"fmt"

	"github.com/candidato/faturamento-service/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations aplica todas as migrations pendentes na pasta ./migrations.
// Chamado uma vez na inicialização do serviço, antes do servidor HTTP subir.
func RunMigrations(cfg *config.Config) error {
	dbConn, err := NewPostgresConnection(cfg)
	if err != nil {
		return fmt.Errorf("migrate: falha ao conectar no banco: %w", err)
	}
	defer dbConn.Close()

	driver, err := postgres.WithInstance(dbConn.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate: falha ao criar driver postgres: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: falha ao inicializar migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: falha ao aplicar migrations: %w", err)
	}

	return nil
}
