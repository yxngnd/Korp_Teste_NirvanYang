// Package config centraliza a leitura de variáveis de ambiente, para que
// nenhum outro pacote precise chamar os.Getenv diretamente.
package config

import (
	"fmt"
	"os"
)

// Config agrupa toda a configuração necessária para o serviço subir.
type Config struct {
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

// LoadConfig lê as variáveis de ambiente e retorna a configuração pronta.
// Entrada: nenhuma (lê o ambiente do processo).
// Saída: *Config preenchido, ou erro se alguma variável obrigatória faltar.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServerPort: getEnv("SERVER_PORT", "8081"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", ""),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	if cfg.DBUser == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("config: DB_USER e DB_NAME são obrigatórios")
	}

	return cfg, nil
}

// PostgresDSN monta a connection string no formato esperado pelo driver lib/pq.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
