package request

import "controle-estoque/internal/domain/repository"

// ListItemsRequest valida os parâmetros da listagem de itens:
// GET /items?name=cab&category_id=3&sort=quantity&direction=desc&page=2&per_page=25
type ListItemsRequest struct {
	Name       string `form:"name"        label:"Nome"             validate:"max=100"`
	CategoryID int    `form:"category_id" label:"Categoria"        validate:"min=0"`
	Sort       string `form:"sort"        label:"Ordenação"        validate:"oneof=name category quantity created_at"`
	Direction  string `form:"direction"   label:"Direção"          validate:"oneof=asc desc"`
	Page       int    `form:"page"        label:"Página"           validate:"min=1"`
	PerPage    int    `form:"per_page"    label:"Itens por página" validate:"oneof=10 25 50"`
}

// ResetsInvalidFields: num filtro de listagem, valor inválido vira o padrão em vez de bloquear a tela.
func (listRequest *ListItemsRequest) ResetsInvalidFields() {}

// SetDefaults preenche o que não veio na URL.
func (listRequest *ListItemsRequest) SetDefaults() {
	if listRequest.Sort == "" {
		listRequest.Sort = string(repository.ItemSortByName)
	}
	if listRequest.Direction == "" {
		listRequest.Direction = string(repository.SortAscending)
	}
	if listRequest.Page == 0 {
		listRequest.Page = 1
	}
	if listRequest.PerPage == 0 {
		listRequest.PerPage = PerPageOptions[0]
	}
}

// ToQuery converte o request (dados da tela) na consulta que o caso de uso entende.
func (listRequest ListItemsRequest) ToQuery() repository.ItemListQuery {
	return repository.ItemListQuery{
		Name:          listRequest.Name,
		CategoryID:    int64(listRequest.CategoryID),
		SortBy:        repository.ItemSortField(listRequest.Sort),
		SortDirection: repository.SortDirection(listRequest.Direction),
		Page:          repository.PageRequest{Number: listRequest.Page, Size: listRequest.PerPage},
	}
}
