// Package config carrega as configurações da aplicação a partir das variáveis de ambiente.
// Em desenvolvimento, as variáveis podem vir de um arquivo .env na raiz do projeto.
package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     string
	DBDriver string
	DBDSN    string
}

func Load() Config {
	// Lê o .env se existir; em produção as variáveis vêm do próprio ambiente.
	_ = godotenv.Load()

	return Config{
		Port:     getEnv("PORT", "8080"),
		DBDriver: getEnv("DB_DRIVER", "sqlite"),
		DBDSN:    getEnv("DB_DSN", "./estoque.db"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
