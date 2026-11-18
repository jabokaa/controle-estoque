// Package router registra as rotas HTTP e liga cada uma ao seu handler.
package router

import (
	"net/http"

	"controle-estoque/internal/delivery/http/handler"
)

func New() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("GET /{$}", handler.Home)
	return mux
}
