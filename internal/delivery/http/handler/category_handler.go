package handler

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"controle-estoque/internal/delivery/http/request"
	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/usecase"
)

type CategoryHandler struct {
	listCategories *usecase.ListCategories
}

func NewCategoryHandler(listCategories *usecase.ListCategories) *CategoryHandler {
	return &CategoryHandler{listCategories: listCategories}
}

// categoryColumns são as colunas da tabela de categorias que podem ser ordenadas.
var categoryColumns = []columnDefinition{
	{Field: "name", Label: "Categoria"},
	{Field: "description", Label: "Descrição"},
	{Field: "created_at", Label: "Criada em"},
}

// categoryListView é tudo o que o template pages/categories.html precisa.
type categoryListView struct {
	Categories     []entity.Category
	Filter         request.ListCategoriesRequest
	Errors         request.ValidationErrors
	Columns        []sortableColumn
	Pagination     paginationView
	PerPageOptions []int
}

// List responde GET /categories?name=&sort=&direction=&page=&per_page=
func (categoryHandler *CategoryHandler) List(writer http.ResponseWriter, httpRequest *http.Request) {
	// 1. FormRequest: lê e valida os parâmetros da URL.
	var filter request.ListCategoriesRequest
	validationErrors := request.Parse(httpRequest, &filter)

	// 2. Caso de uso.
	page, err := categoryHandler.listCategories.Execute(httpRequest.Context(), filter.ToQuery())
	if err != nil {
		log.Printf("erro ao listar categorias: %v", err)
		http.Error(writer, "Não foi possível carregar as categorias.", http.StatusInternalServerError)
		return
	}
	filter.Page = page.Number // o caso de uso pode ter ajustado a página

	// 3. Monta a tela.
	view := categoryListView{
		Categories:     page.Items,
		Filter:         filter,
		Errors:         validationErrors,
		PerPageOptions: request.PerPageOptions,
		Columns: newSortableColumns(categoryColumns, filter.Sort, filter.Direction, func(field, direction string) string {
			sorted := filter
			sorted.Sort, sorted.Direction, sorted.Page = field, direction, 1
			return categoriesURL(sorted)
		}),
		Pagination: newPaginationView(page.Number, page.Size, page.TotalPages(), page.Total, func(pageNumber int) string {
			paged := filter
			paged.Page = pageNumber
			return categoriesURL(paged)
		}),
	}

	if err := templates.ExecuteTemplate(writer, "categories.html", view); err != nil {
		log.Printf("erro ao renderizar categorias: %v", err)
	}
}

// categoriesURL monta o link da listagem com o filtro, a ordenação e a página informados.
func categoriesURL(filter request.ListCategoriesRequest) string {
	values := url.Values{}
	if filter.Name != "" {
		values.Set("name", filter.Name)
	}
	values.Set("sort", filter.Sort)
	values.Set("direction", filter.Direction)
	values.Set("per_page", strconv.Itoa(filter.PerPage))
	if filter.Page > 1 {
		values.Set("page", strconv.Itoa(filter.Page))
	}
	return "/categories?" + values.Encode()
}
