package request

import "controle-estoque/internal/domain/repository"

// ListCategoriesRequest valida os parâmetros da listagem de categorias:
// GET /categories?name=ele&sort=created_at&direction=desc&page=2&per_page=25
type ListCategoriesRequest struct {
	Name      string `form:"name"      label:"Nome"             validate:"max=100"`
	Sort      string `form:"sort"      label:"Ordenação"        validate:"oneof=name description created_at"`
	Direction string `form:"direction" label:"Direção"          validate:"oneof=asc desc"`
	Page      int    `form:"page"      label:"Página"           validate:"min=1"`
	PerPage   int    `form:"per_page"  label:"Itens por página" validate:"oneof=10 25 50"`
}

// PerPageOptions são os tamanhos de página que a tela oferece (os mesmos da regra oneof acima).
var PerPageOptions = []int{10, 25, 50}

// SetDefaults preenche o que não veio na URL.
func (listRequest *ListCategoriesRequest) SetDefaults() {
	if listRequest.Sort == "" {
		listRequest.Sort = string(repository.CategorySortByName)
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
func (listRequest ListCategoriesRequest) ToQuery() repository.CategoryListQuery {
	return repository.CategoryListQuery{
		Name:          listRequest.Name,
		SortBy:        repository.CategorySortField(listRequest.Sort),
		SortDirection: repository.SortDirection(listRequest.Direction),
		Page:          repository.PageRequest{Number: listRequest.Page, Size: listRequest.PerPage},
	}
}
