// Package handler recebe as requisições HTTP, chama os casos de uso e renderiza o front (web/templates).
package handler

import "html/template"

// templates guarda todas as views de web/templates, lidas uma vez ao iniciar.
var templates = template.Must(template.ParseGlob("web/templates/*.html"))
