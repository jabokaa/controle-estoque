package request

import (
	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/usecase"
)

// RegisterStockMovementRequest valida o formulário de entrada/baixa de estoque: POST /movements
type RegisterStockMovementRequest struct {
	ItemID   int    `form:"item_id"  label:"Item"       validate:"required"`
	Type     string `form:"type"     label:"Tipo"       validate:"required,oneof=in out"`
	Quantity int    `form:"quantity" label:"Quantidade" validate:"required,min=1,max=1000000"`
	Note     string `form:"note"     label:"Observação" validate:"max=255"`
}

// SetDefaults: sem tipo informado, o formulário começa em "entrada".
func (registerRequest *RegisterStockMovementRequest) SetDefaults() {
	if registerRequest.Type == "" {
		registerRequest.Type = string(entity.MovementIn)
	}
}

// ToInput converte o request (dados da tela) na entrada que o caso de uso entende.
func (registerRequest RegisterStockMovementRequest) ToInput() usecase.StockMovementInput {
	return usecase.StockMovementInput{
		ItemID:   int64(registerRequest.ItemID),
		Type:     entity.MovementType(registerRequest.Type),
		Quantity: registerRequest.Quantity,
		Note:     registerRequest.Note,
	}
}
