package main

import (
	"log"
	"net/http"

	"controle-estoque/internal/config"
	"controle-estoque/internal/delivery/http/router"
)

func main() {
	cfg := config.Load()

	// Aqui é o "composition root": no futuro, crie o banco, os repositórios
	// e os casos de uso e injete-os nos handlers.
	r := router.New()

	log.Printf("servidor rodando em http://localhost:%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}
