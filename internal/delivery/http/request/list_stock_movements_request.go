package request

import (
	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// ListStockMovementsRequest valida os parâmetros do histórico de movimentações:
// GET /movements?item_id=3&type=out&page=2&per_page=25
type ListStockMovementsRequest struct {
	ItemID  int    `form:"item_id"  label:"Item"             validate:"min=0"`
	Type    string `form:"type"     label:"Tipo"             validate:"omitempty,oneof=in out"`
	Page    int    `form:"page"     label:"Página"           validate:"min=1"`
	PerPage int    `form:"per_page" label:"Itens por página" validate:"oneof=10 25 50"`
}

// ResetsInvalidFields: num filtro de listagem, valor inválido vira o padrão em vez de bloquear a tela.
func (listRequest *ListStockMovementsRequest) ResetsInvalidFields() {}

// SetDefaults preenche o que não veio na URL.
func (listRequest *ListStockMovementsRequest) SetDefaults() {
	if listRequest.Page == 0 {
		listRequest.Page = 1
	}
	if listRequest.PerPage == 0 {
		listRequest.PerPage = PerPageOptions[0]
	}
}

// ToQuery converte o request (dados da tela) na consulta que o caso de uso entende.
func (listRequest ListStockMovementsRequest) ToQuery() repository.StockMovementListQuery {
	return repository.StockMovementListQuery{
		ItemID: int64(listRequest.ItemID),
		Type:   entity.MovementType(listRequest.Type),
		Page:   repository.PageRequest{Number: listRequest.Page, Size: listRequest.PerPage},
	}
}
