package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"controle-estoque/internal/delivery/http/request"
	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
	"controle-estoque/internal/usecase"
)

// CategoryUseCases agrupa os casos de uso que o CategoryHandler usa,
// para o construtor não crescer um parâmetro a cada ação nova.
type CategoryUseCases struct {
	List   *usecase.ListCategories
	Get    *usecase.GetCategory
	Create *usecase.CreateCategory
	Update *usecase.UpdateCategory
	Delete *usecase.DeleteCategory
}

type CategoryHandler struct {
	useCases CategoryUseCases
}

func NewCategoryHandler(useCases CategoryUseCases) *CategoryHandler {
	return &CategoryHandler{useCases: useCases}
}

// ---------- Listagem ----------

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
	Flash          *flashMessage
}

// List responde GET /categories?name=&sort=&direction=&page=&per_page=
func (categoryHandler *CategoryHandler) List(writer http.ResponseWriter, httpRequest *http.Request) {
	// 1. FormRequest: lê e valida os parâmetros da URL.
	var filter request.ListCategoriesRequest
	validationErrors := request.Parse(httpRequest, &filter)

	// 2. Caso de uso.
	page, err := categoryHandler.useCases.List.Execute(httpRequest.Context(), filter.ToQuery())
	if err != nil {
		serverError(writer, "listar categorias", err)
		return
	}
	filter.Page = page.Number // o caso de uso pode ter ajustado a página

	// 3. Monta a tela.
	view := categoryListView{
		Categories:     page.Items,
		Filter:         filter,
		Errors:         validationErrors,
		PerPageOptions: request.PerPageOptions,
		Flash:          popFlash(writer, httpRequest),
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
	render(writer, http.StatusOK, "categories.html", view)
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

// ---------- Cadastro e edição ----------

// categoryFormView é tudo o que o template pages/category_form.html precisa.
type categoryFormView struct {
	IsEdit   bool
	Category entity.Category // na edição: a categoria original (ID, datas)
	Form     request.SaveCategoryRequest
	Errors   request.ValidationErrors
	Action   string // para onde o formulário envia
}

// New responde GET /categories/new: formulário vazio.
func (categoryHandler *CategoryHandler) New(writer http.ResponseWriter, httpRequest *http.Request) {
	render(writer, http.StatusOK, "category_form.html", categoryFormView{Action: "/categories"})
}

// Create responde POST /categories.
func (categoryHandler *CategoryHandler) Create(writer http.ResponseWriter, httpRequest *http.Request) {
	view := categoryFormView{Action: "/categories"}

	// 1. FormRequest
	if view.Errors = request.Parse(httpRequest, &view.Form); view.Errors != nil {
		render(writer, http.StatusUnprocessableEntity, "category_form.html", view)
		return
	}

	// 2. Caso de uso
	category, err := categoryHandler.useCases.Create.Execute(httpRequest.Context(), view.Form.ToInput())
	if view.Errors = categoryFieldErrors(err); view.Errors != nil {
		render(writer, http.StatusUnprocessableEntity, "category_form.html", view)
		return
	}
	if err != nil {
		serverError(writer, "cadastrar categoria", err)
		return
	}

	// 3. Redirect com mensagem (Post/Redirect/Get: atualizar a página não reenvia o formulário)
	setFlash(writer, "success", fmt.Sprintf("Categoria “%s” cadastrada.", category.Name))
	http.Redirect(writer, httpRequest, "/categories", http.StatusSeeOther)
}

// Edit responde GET /categories/{id}/edit: formulário preenchido.
func (categoryHandler *CategoryHandler) Edit(writer http.ResponseWriter, httpRequest *http.Request) {
	category, found := categoryHandler.findCategory(writer, httpRequest)
	if !found {
		return
	}
	render(writer, http.StatusOK, "category_form.html", categoryFormView{
		IsEdit:   true,
		Category: category,
		Form:     request.NewSaveCategoryRequest(category),
		Action:   fmt.Sprintf("/categories/%d", category.ID),
	})
}

// Update responde POST /categories/{id}.
func (categoryHandler *CategoryHandler) Update(writer http.ResponseWriter, httpRequest *http.Request) {
	category, found := categoryHandler.findCategory(writer, httpRequest)
	if !found {
		return
	}
	view := categoryFormView{IsEdit: true, Category: category, Action: fmt.Sprintf("/categories/%d", category.ID)}

	if view.Errors = request.Parse(httpRequest, &view.Form); view.Errors != nil {
		render(writer, http.StatusUnprocessableEntity, "category_form.html", view)
		return
	}

	updated, err := categoryHandler.useCases.Update.Execute(httpRequest.Context(), category.ID, view.Form.ToInput())
	if view.Errors = categoryFieldErrors(err); view.Errors != nil {
		render(writer, http.StatusUnprocessableEntity, "category_form.html", view)
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		categoryNotFound(writer, httpRequest)
		return
	}
	if err != nil {
		serverError(writer, "atualizar categoria", err)
		return
	}

	setFlash(writer, "success", fmt.Sprintf("Categoria “%s” atualizada.", updated.Name))
	http.Redirect(writer, httpRequest, "/categories", http.StatusSeeOther)
}

// ---------- Exclusão ----------

// Delete responde POST /categories/{id}/delete.
// A confirmação acontece antes, na tela (modal); aqui a exclusão é feita de fato.
func (categoryHandler *CategoryHandler) Delete(writer http.ResponseWriter, httpRequest *http.Request) {
	returnTo := safeReturnPath(httpRequest.FormValue("return_to"), "/categories")

	id, err := strconv.ParseInt(httpRequest.PathValue("id"), 10, 64)
	if err != nil {
		categoryNotFound(writer, httpRequest)
		return
	}

	category, err := categoryHandler.useCases.Delete.Execute(httpRequest.Context(), id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		setFlash(writer, "danger", "Essa categoria não existe mais.")
	case err != nil:
		serverError(writer, "excluir categoria", err)
		return
	default:
		setFlash(writer, "success", fmt.Sprintf("Categoria “%s” excluída.", category.Name))
	}
	http.Redirect(writer, httpRequest, returnTo, http.StatusSeeOther)
}

// ---------- Auxiliares ----------

// findCategory lê o {id} da rota e busca a categoria. Se não achar, já responde 404.
func (categoryHandler *CategoryHandler) findCategory(writer http.ResponseWriter, httpRequest *http.Request) (entity.Category, bool) {
	id, err := strconv.ParseInt(httpRequest.PathValue("id"), 10, 64)
	if err != nil {
		categoryNotFound(writer, httpRequest)
		return entity.Category{}, false
	}
	category, err := categoryHandler.useCases.Get.Execute(httpRequest.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		categoryNotFound(writer, httpRequest)
		return category, false
	}
	if err != nil {
		serverError(writer, "buscar categoria", err)
		return category, false
	}
	return category, true
}

// categoryFieldErrors traduz os erros de regra de negócio em mensagens de campo do formulário.
// Devolve nil se o erro não for de nenhum campo.
func categoryFieldErrors(err error) request.ValidationErrors {
	switch {
	case errors.Is(err, entity.ErrCategoryNameRequired):
		return request.ValidationErrors{"name": "Nome é um campo obrigatório."}
	case errors.Is(err, usecase.ErrCategoryNameTaken):
		return request.ValidationErrors{"name": "Já existe uma categoria com esse nome."}
	}
	return nil
}

func categoryNotFound(writer http.ResponseWriter, httpRequest *http.Request) {
	setFlash(writer, "danger", "Categoria não encontrada.")
	http.Redirect(writer, httpRequest, "/categories", http.StatusSeeOther)
}

// safeReturnPath só aceita caminhos internos da listagem de categorias,
// para ninguém usar o return_to para mandar o usuário a outro site.
func safeReturnPath(path, fallback string) string {
	if path == "/categories" || strings.HasPrefix(path, "/categories?") {
		return path
	}
	return fallback
}
