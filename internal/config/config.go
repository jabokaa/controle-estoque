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

	// Assistente de IA. Sem chave, o chat aparece desativado com instruções.
	AnthropicAPIKey string
	ClaudeModel     string // CLAUDE_MODELO
	ClaudeEffort    string // CLAUDE_ESFORCO: low, medium, high, xhigh ou max
}

func Load() Config {
	// Lê o .env se existir; em produção as variáveis vêm do próprio ambiente.
	_ = godotenv.Load()

	return Config{
		Port:            getEnv("PORT", "8080"),
		DBDriver:        getEnv("DB_DRIVER", "sqlite"),
		DBDSN:           getEnv("DB_DSN", "./estoque.db"),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
		ClaudeModel:     getEnv("CLAUDE_MODELO", "claude-sonnet-5-5"),
		ClaudeEffort:    getEnv("CLAUDE_ESFORCO", "medium"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
