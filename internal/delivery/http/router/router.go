// Package router registra as rotas HTTP e liga cada uma ao seu handler.
package router

import (
	"controle-estoque/internal/app"
	"controle-estoque/internal/delivery/http/handler"
	"net/http"
)

func New(app *app.App) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/", handler.Home)

	// Categorias (formulários HTML só enviam GET e POST, por isso editar e excluir também são POST)
	mux.HandleFunc("GET /categories", app.CategoryHandler.List)
	mux.HandleFunc("GET /categories/new", app.CategoryHandler.New)
	mux.HandleFunc("POST /categories", app.CategoryHandler.Create)
	mux.HandleFunc("GET /categories/{id}/edit", app.CategoryHandler.Edit)
	mux.HandleFunc("POST /categories/{id}", app.CategoryHandler.Update)
	mux.HandleFunc("POST /categories/{id}/delete", app.CategoryHandler.Delete)

	// Itens
	mux.HandleFunc("GET /items", app.ItemHandler.List)
	mux.HandleFunc("GET /items/new", app.ItemHandler.New)
	mux.HandleFunc("POST /items", app.ItemHandler.Create)
	mux.HandleFunc("GET /items/{id}/edit", app.ItemHandler.Edit)
	mux.HandleFunc("POST /items/{id}", app.ItemHandler.Update)
	mux.HandleFunc("POST /items/{id}/delete", app.ItemHandler.Delete)

	// Proteção contra CSRF: recusa POST vindo de outro site (usa os cabeçalhos
	// Sec-Fetch-Site/Origin que o navegador envia). Equivale ao @csrf do Laravel.
	return http.NewCrossOriginProtection().Handler(mux)
}
