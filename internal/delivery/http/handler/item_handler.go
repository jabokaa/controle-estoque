package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"controle-estoque/internal/delivery/http/request"
	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
	"controle-estoque/internal/usecase"
)

// ItemUseCases agrupa os casos de uso que o ItemHandler usa,
// para o construtor não crescer um parâmetro a cada ação nova.
type ItemUseCases struct {
	List       *usecase.ListItems
	Get        *usecase.GetItem
	Create     *usecase.CreateItem
	Update     *usecase.UpdateItem
	Delete     *usecase.DeleteItem
	Categories *usecase.ListAllCategories // opções do campo "Categoria" e do filtro
	Summary    *usecase.GetStockSummary   // cartões de resumo do estoque
}

type ItemHandler struct {
	useCases ItemUseCases
}

func NewItemHandler(useCases ItemUseCases) *ItemHandler {
	return &ItemHandler{useCases: useCases}
}

// ---------- Listagem ----------

// itemColumns são as colunas da tabela de itens que podem ser ordenadas.
var itemColumns = []columnDefinition{
	{Field: "name", Label: "Item"},
	{Field: "category", Label: "Categoria"},
	{Field: "quantity", Label: "Quantidade"},
	{Field: "created_at", Label: "Criado em"},
}

// itemListView é tudo o que o template pages/items.html precisa.
type itemListView struct {
	Items           []entity.Item
	Filter          request.ListItemsRequest
	Errors          request.ValidationErrors
	Columns         []sortableColumn
	Pagination      paginationView
	PerPageOptions  []int
	CategoryOptions []selectOption
	Summary         repository.StockSummary
	Flash           *flashMessage
}

// List responde GET /items?name=&category_id=&sort=&direction=&page=&per_page=
func (itemHandler *ItemHandler) List(writer http.ResponseWriter, httpRequest *http.Request) {
	// 1. FormRequest: lê e valida os parâmetros da URL.
	var filter request.ListItemsRequest
	validationErrors := request.Parse(httpRequest, &filter)

	// 2. Casos de uso.
	page, err := itemHandler.useCases.List.Execute(httpRequest.Context(), filter.ToQuery())
	if err != nil {
		serverError(writer, "listar itens", err)
		return
	}
	filter.Page = page.Number // o caso de uso pode ter ajustado a página

	categoryOptions, err := itemHandler.categoryOptions(httpRequest, int64(filter.CategoryID))
	if err != nil {
		serverError(writer, "listar categorias", err)
		return
	}

	summary, err := itemHandler.useCases.Summary.Execute(httpRequest.Context())
	if err != nil {
		serverError(writer, "resumir estoque", err)
		return
	}

	// 3. Monta a tela.
	view := itemListView{
		Summary:         summary,
		Items:           page.Items,
		Filter:          filter,
		Errors:          validationErrors,
		PerPageOptions:  request.PerPageOptions,
		CategoryOptions: categoryOptions,
		Flash:           popFlash(writer, httpRequest),
		Columns: newSortableColumns(itemColumns, filter.Sort, filter.Direction, func(field, direction string) string {
			sorted := filter
			sorted.Sort, sorted.Direction, sorted.Page = field, direction, 1
			return itemsURL(sorted)
		}),
		Pagination: newPaginationView(page.Number, page.Size, page.TotalPages(), page.Total, func(pageNumber int) string {
			paged := filter
			paged.Page = pageNumber
			return itemsURL(paged)
		}),
	}
	render(writer, http.StatusOK, "items.html", view)
}

// itemsURL monta o link da listagem com os filtros, a ordenação e a página informados.
func itemsURL(filter request.ListItemsRequest) string {
	values := url.Values{}
	if filter.Name != "" {
		values.Set("name", filter.Name)
	}
	if filter.CategoryID > 0 {
		values.Set("category_id", strconv.Itoa(filter.CategoryID))
	}
	values.Set("sort", filter.Sort)
	values.Set("direction", filter.Direction)
	values.Set("per_page", strconv.Itoa(filter.PerPage))
	if filter.Page > 1 {
		values.Set("page", strconv.Itoa(filter.Page))
	}
	return "/items?" + values.Encode()
}

// ---------- Cadastro e edição ----------

// itemFormView é tudo o que o template pages/item_form.html precisa.
type itemFormView struct {
	IsEdit          bool
	Item            entity.Item // na edição: o item original (ID, datas)
	Form            request.SaveItemRequest
	Errors          request.ValidationErrors
	Action          string // para onde o formulário envia
	CategoryOptions []selectOption
}

// New responde GET /items/new: formulário vazio.
// Aceita ?category_id=3 para já vir com a categoria escolhida.
func (itemHandler *ItemHandler) New(writer http.ResponseWriter, httpRequest *http.Request) {
	view := itemFormView{Action: "/items"}
	view.Form.CategoryID, _ = strconv.Atoi(httpRequest.URL.Query().Get("category_id"))
	itemHandler.renderForm(writer, httpRequest, http.StatusOK, view)
}

// Create responde POST /items.
func (itemHandler *ItemHandler) Create(writer http.ResponseWriter, httpRequest *http.Request) {
	view := itemFormView{Action: "/items"}

	// 1. FormRequest
	if view.Errors = request.Parse(httpRequest, &view.Form); view.Errors != nil {
		itemHandler.renderForm(writer, httpRequest, http.StatusUnprocessableEntity, view)
		return
	}

	// 2. Caso de uso
	item, err := itemHandler.useCases.Create.Execute(httpRequest.Context(), view.Form.ToInput())
	if view.Errors = itemFieldErrors(err); view.Errors != nil {
		itemHandler.renderForm(writer, httpRequest, http.StatusUnprocessableEntity, view)
		return
	}
	if err != nil {
		serverError(writer, "cadastrar item", err)
		return
	}

	// 3. Redirect com mensagem (Post/Redirect/Get: atualizar a página não reenvia o formulário)
	setFlash(writer, "success", fmt.Sprintf("Item “%s” cadastrado.", item.Name))
	http.Redirect(writer, httpRequest, "/items", http.StatusSeeOther)
}

// Edit responde GET /items/{id}/edit: formulário preenchido.
func (itemHandler *ItemHandler) Edit(writer http.ResponseWriter, httpRequest *http.Request) {
	item, found := itemHandler.findItem(writer, httpRequest)
	if !found {
		return
	}
	itemHandler.renderForm(writer, httpRequest, http.StatusOK, itemFormView{
		IsEdit: true,
		Item:   item,
		Form:   request.NewSaveItemRequest(item),
		Action: fmt.Sprintf("/items/%d", item.ID),
	})
}

// Update responde POST /items/{id}.
func (itemHandler *ItemHandler) Update(writer http.ResponseWriter, httpRequest *http.Request) {
	item, found := itemHandler.findItem(writer, httpRequest)
	if !found {
		return
	}
	view := itemFormView{IsEdit: true, Item: item, Action: fmt.Sprintf("/items/%d", item.ID)}

	if view.Errors = request.Parse(httpRequest, &view.Form); view.Errors != nil {
		itemHandler.renderForm(writer, httpRequest, http.StatusUnprocessableEntity, view)
		return
	}

	updated, err := itemHandler.useCases.Update.Execute(httpRequest.Context(), item.ID, view.Form.ToInput())
	if view.Errors = itemFieldErrors(err); view.Errors != nil {
		itemHandler.renderForm(writer, httpRequest, http.StatusUnprocessableEntity, view)
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		itemNotFound(writer, httpRequest)
		return
	}
	if err != nil {
		serverError(writer, "atualizar item", err)
		return
	}

	setFlash(writer, "success", fmt.Sprintf("Item “%s” atualizado.", updated.Name))
	http.Redirect(writer, httpRequest, "/items", http.StatusSeeOther)
}

// ---------- Exclusão ----------

// Delete responde POST /items/{id}/delete.
// A confirmação acontece antes, na tela (modal); aqui a exclusão é feita de fato.
func (itemHandler *ItemHandler) Delete(writer http.ResponseWriter, httpRequest *http.Request) {
	returnTo := safeReturnPath(httpRequest.FormValue("return_to"), "/items")

	id, err := strconv.ParseInt(httpRequest.PathValue("id"), 10, 64)
	if err != nil {
		itemNotFound(writer, httpRequest)
		return
	}

	item, err := itemHandler.useCases.Delete.Execute(httpRequest.Context(), id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		setFlash(writer, "danger", "Esse item não existe mais.")
	case errors.Is(err, usecase.ErrItemHasMovements):
		setFlash(writer, "danger", fmt.Sprintf(
			"O item “%s” não pode ser excluído porque tem movimentações de estoque (o histórico seria perdido).", item.Name))
	case err != nil:
		serverError(writer, "excluir item", err)
		return
	default:
		setFlash(writer, "success", fmt.Sprintf("Item “%s” excluído.", item.Name))
	}
	http.Redirect(writer, httpRequest, returnTo, http.StatusSeeOther)
}

// ---------- Auxiliares ----------

// renderForm completa a tela do formulário com as opções de categoria e desenha a página.
func (itemHandler *ItemHandler) renderForm(writer http.ResponseWriter, httpRequest *http.Request, status int, view itemFormView) {
	categoryOptions, err := itemHandler.categoryOptions(httpRequest, int64(view.Form.CategoryID))
	if err != nil {
		serverError(writer, "listar categorias", err)
		return
	}
	view.CategoryOptions = categoryOptions
	render(writer, status, "item_form.html", view)
}

// categoryOptions monta as opções do campo de categoria, marcando a selecionada.
func (itemHandler *ItemHandler) categoryOptions(httpRequest *http.Request, selectedID int64) ([]selectOption, error) {
	categories, err := itemHandler.useCases.Categories.Execute(httpRequest.Context())
	if err != nil {
		return nil, err
	}
	options := make([]selectOption, 0, len(categories))
	for _, category := range categories {
		options = append(options, selectOption{
			Value:    category.ID,
			Label:    category.Name,
			Selected: category.ID == selectedID,
		})
	}
	return options, nil
}

// findItem lê o {id} da rota e busca o item. Se não achar, já responde com redirect e aviso.
func (itemHandler *ItemHandler) findItem(writer http.ResponseWriter, httpRequest *http.Request) (entity.Item, bool) {
	id, err := strconv.ParseInt(httpRequest.PathValue("id"), 10, 64)
	if err != nil {
		itemNotFound(writer, httpRequest)
		return entity.Item{}, false
	}
	item, err := itemHandler.useCases.Get.Execute(httpRequest.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		itemNotFound(writer, httpRequest)
		return item, false
	}
	if err != nil {
		serverError(writer, "buscar item", err)
		return item, false
	}
	return item, true
}

// itemFieldErrors traduz os erros de regra de negócio em mensagens de campo do formulário.
// Devolve nil se o erro não for de nenhum campo.
func itemFieldErrors(err error) request.ValidationErrors {
	switch {
	case errors.Is(err, entity.ErrItemNameRequired):
		return request.ValidationErrors{"name": "Nome é um campo obrigatório."}
	case errors.Is(err, usecase.ErrItemNameTaken):
		return request.ValidationErrors{"name": "Já existe um item com esse nome."}
	case errors.Is(err, entity.ErrItemCategoryRequired):
		return request.ValidationErrors{"category_id": "Categoria é um campo obrigatório."}
	case errors.Is(err, usecase.ErrItemCategoryNotFound):
		return request.ValidationErrors{"category_id": "A categoria escolhida não existe mais. Escolha outra."}
	case errors.Is(err, entity.ErrItemQuantityNegative):
		return request.ValidationErrors{"initial_quantity": "Quantidade inicial não pode ser negativa."}
	}
	return nil
}

func itemNotFound(writer http.ResponseWriter, httpRequest *http.Request) {
	setFlash(writer, "danger", "Item não encontrado.")
	http.Redirect(writer, httpRequest, "/items", http.StatusSeeOther)
}
