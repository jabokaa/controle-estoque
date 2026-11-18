package handler

import (
	"log"
	"net/http"

	"controle-estoque/internal/usecase"
)

type CategoryHandler struct {
	listCategories *usecase.ListCategories
}

func NewCategoryHandler(listCategories *usecase.ListCategories) *CategoryHandler {
	return &CategoryHandler{listCategories: listCategories}
}

// List responde GET /categories.
func (categoryHandler *CategoryHandler) List(writer http.ResponseWriter, request *http.Request) {
	categories, err := categoryHandler.listCategories.Execute(request.Context())
	if err != nil {
		log.Printf("erro ao listar categorias: %v", err)
		http.Error(writer, "Não foi possível carregar as categorias.", http.StatusInternalServerError)
		return
	}

	data := map[string]any{"Categories": categories}
	if err := templates.ExecuteTemplate(writer, "categories.html", data); err != nil {
		log.Printf("erro ao renderizar categorias: %v", err)
	}
}
