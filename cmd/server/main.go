package main

import (
	"log"
	"net/http"

	"controle-estoque/internal/app"
	"controle-estoque/internal/config"
	"controle-estoque/internal/delivery/http/router"
	"controle-estoque/internal/infrastructure/database"
)

func main() {
	cfg := config.Load()

	// Infraestrutura: conexão com o banco e migrations
	db, err := database.Connect(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer database.Close(db)

	if err := database.Migrate(db); err != nil {
		log.Fatalf("erro ao rodar migrations: %v", err)
	}

	app, err := app.NewApp(db, cfg)
	if err != nil {
		log.Fatalf("erro na configuração: %v", err)
	}
	if cfg.AnthropicAPIKey == "" {
		log.Printf("assistente de IA desativado: defina ANTHROPIC_API_KEY no .env")
	} else {
		log.Printf("assistente de IA ativo: modelo %s, esforço %s", cfg.ClaudeModel, cfg.ClaudeEffort)
	}

	routes := router.New(app)

	log.Printf("servidor rodando em http://localhost:%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, routes))
}
