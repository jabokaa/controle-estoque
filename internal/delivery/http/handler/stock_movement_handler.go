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

// StockMovementUseCases agrupa os casos de uso que o StockMovementHandler usa.
type StockMovementUseCases struct {
	List     *usecase.ListStockMovements
	Register *usecase.RegisterStockMovement
	Summary  *usecase.GetStockSummary // cartões de resumo do estoque
	Items    *usecase.ListAllItems    // opções do campo "Item" e do filtro
}

type StockMovementHandler struct {
	useCases StockMovementUseCases
}

func NewStockMovementHandler(useCases StockMovementUseCases) *StockMovementHandler {
	return &StockMovementHandler{useCases: useCases}
}

// ---------- Histórico ----------

// stockMovementListView é tudo o que o template pages/movements.html precisa.
type stockMovementListView struct {
	Movements      []entity.StockMovement
	Filter         request.ListStockMovementsRequest
	Errors         request.ValidationErrors
	Pagination     paginationView
	PerPageOptions []int
	ItemOptions    []selectOption
	Summary        repository.StockSummary
	Flash          *flashMessage
}

// List responde GET /movements?item_id=&type=&page=&per_page=
func (movementHandler *StockMovementHandler) List(writer http.ResponseWriter, httpRequest *http.Request) {
	// 1. FormRequest: lê e valida os parâmetros da URL.
	var filter request.ListStockMovementsRequest
	validationErrors := request.Parse(httpRequest, &filter)

	// 2. Casos de uso.
	page, err := movementHandler.useCases.List.Execute(httpRequest.Context(), filter.ToQuery())
	if err != nil {
		serverError(writer, "listar movimentações", err)
		return
	}
	filter.Page = page.Number // o caso de uso pode ter ajustado a página

	itemOptions, err := movementHandler.itemOptions(httpRequest, int64(filter.ItemID), false)
	if err != nil {
		serverError(writer, "listar itens", err)
		return
	}

	summary, err := movementHandler.useCases.Summary.Execute(httpRequest.Context())
	if err != nil {
		serverError(writer, "resumir estoque", err)
		return
	}

	// 3. Monta a tela.
	view := stockMovementListView{
		Movements:      page.Items,
		Filter:         filter,
		Errors:         validationErrors,
		PerPageOptions: request.PerPageOptions,
		ItemOptions:    itemOptions,
		Summary:        summary,
		Flash:          popFlash(writer, httpRequest),
		Pagination: newPaginationView(page.Number, page.Size, page.TotalPages(), page.Total, func(pageNumber int) string {
			paged := filter
			paged.Page = pageNumber
			return movementsURL(paged)
		}),
	}
	render(writer, http.StatusOK, "movements.html", view)
}

// movementsURL monta o link do histórico com os filtros e a página informados.
func movementsURL(filter request.ListStockMovementsRequest) string {
	values := url.Values{}
	if filter.ItemID > 0 {
		values.Set("item_id", strconv.Itoa(filter.ItemID))
	}
	if filter.Type != "" {
		values.Set("type", filter.Type)
	}
	values.Set("per_page", strconv.Itoa(filter.PerPage))
	if filter.Page > 1 {
		values.Set("page", strconv.Itoa(filter.Page))
	}
	return "/movements?" + values.Encode()
}

// ---------- Entrada / baixa ----------

// stockMovementFormView é tudo o que o template pages/movement_form.html precisa.
type stockMovementFormView struct {
	Form        request.RegisterStockMovementRequest
	Errors      request.ValidationErrors
	ItemOptions []selectOption
}

// New responde GET /movements/new: formulário de entrada/baixa.
// Aceita ?item_id=3&type=out para já vir com o item e o tipo escolhidos (botões da lista de itens).
func (movementHandler *StockMovementHandler) New(writer http.ResponseWriter, httpRequest *http.Request) {
	var view stockMovementFormView
	query := httpRequest.URL.Query()
	view.Form.ItemID, _ = strconv.Atoi(query.Get("item_id"))
	if movementType := query.Get("type"); movementType == string(entity.MovementOut) {
		view.Form.Type = movementType
	}
	view.Form.SetDefaults()
	movementHandler.renderForm(writer, httpRequest, http.StatusOK, view)
}

// Register responde POST /movements.
func (movementHandler *StockMovementHandler) Register(writer http.ResponseWriter, httpRequest *http.Request) {
	var view stockMovementFormView

	// 1. FormRequest
	if view.Errors = request.Parse(httpRequest, &view.Form); view.Errors != nil {
		movementHandler.renderForm(writer, httpRequest, http.StatusUnprocessableEntity, view)
		return
	}

	// 2. Caso de uso (transação: atualiza o estoque e grava a movimentação)
	movement, err := movementHandler.useCases.Register.Execute(httpRequest.Context(), view.Form.ToInput())
	if view.Errors = stockMovementFieldErrors(err); view.Errors != nil {
		movementHandler.renderForm(writer, httpRequest, http.StatusUnprocessableEntity, view)
		return
	}
	if err != nil {
		serverError(writer, "registrar movimentação", err)
		return
	}

	// 3. Redirect com mensagem
	action := "Entrada"
	if !movement.IsEntry() {
		action = "Baixa"
	}
	setFlash(writer, "success", fmt.Sprintf("%s de %s un. em “%s” registrada. Estoque atual: %s.",
		action, formatNumber(movement.Quantity), movement.Item.Name, formatNumber(movement.BalanceAfter)))
	http.Redirect(writer, httpRequest, "/movements", http.StatusSeeOther)
}

// ---------- Auxiliares ----------

// renderForm completa a tela do formulário com as opções de item e desenha a página.
func (movementHandler *StockMovementHandler) renderForm(writer http.ResponseWriter, httpRequest *http.Request, status int, view stockMovementFormView) {
	itemOptions, err := movementHandler.itemOptions(httpRequest, int64(view.Form.ItemID), true)
	if err != nil {
		serverError(writer, "listar itens", err)
		return
	}
	view.ItemOptions = itemOptions
	render(writer, status, "movement_form.html", view)
}

// itemOptions monta as opções do campo de item, marcando o selecionado.
// Com withStock, o texto mostra o estoque atual: "Cabo HDMI 2m (estoque: 15)".
func (movementHandler *StockMovementHandler) itemOptions(httpRequest *http.Request, selectedID int64, withStock bool) ([]selectOption, error) {
	items, err := movementHandler.useCases.Items.Execute(httpRequest.Context())
	if err != nil {
		return nil, err
	}
	options := make([]selectOption, 0, len(items))
	for _, item := range items {
		label := item.Name
		if withStock {
			label = fmt.Sprintf("%s (estoque: %s)", item.Name, formatNumber(item.Quantity))
		}
		options = append(options, selectOption{Value: item.ID, Label: label, Selected: item.ID == selectedID})
	}
	return options, nil
}

// stockMovementFieldErrors traduz os erros de regra de negócio em mensagens de campo do formulário.
// Devolve nil se o erro não for de nenhum campo.
func stockMovementFieldErrors(err error) request.ValidationErrors {
	var insufficientStock entity.InsufficientStockError
	switch {
	case errors.As(err, &insufficientStock):
		return request.ValidationErrors{"quantity": fmt.Sprintf(
			"Estoque insuficiente: há apenas %s unidade(s) desse item.", formatNumber(insufficientStock.Available))}
	case errors.Is(err, repository.ErrNotFound):
		return request.ValidationErrors{"item_id": "O item escolhido não existe mais. Escolha outro."}
	case errors.Is(err, entity.ErrMovementQuantityInvalid):
		return request.ValidationErrors{"quantity": "Quantidade deve ser maior que zero."}
	case errors.Is(err, entity.ErrMovementTypeInvalid):
		return request.ValidationErrors{"type": "Escolha entrada ou saída."}
	}
	return nil
}
