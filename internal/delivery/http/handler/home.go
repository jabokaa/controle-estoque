// Package handler recebe as requisições HTTP, chama os casos de uso e renderiza o front (web/templates).
package handler

import (
	"html/template"
	"net/http"
)

var templates = template.Must(template.ParseGlob("web/templates/*.html"))

func Home(w http.ResponseWriter, r *http.Request) {
	if err := templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
