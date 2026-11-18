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
	mux.HandleFunc("/categories", app.CategoryHandler.List)
	return mux
}
