package request

import (
	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/usecase"
)

// SaveItemRequest valida o formulário de cadastro e de edição de item:
// POST /items e POST /items/{id}
type SaveItemRequest struct {
	Name        string `form:"name"        label:"Nome"       validate:"required,max=100"`
	CategoryID  int    `form:"category_id" label:"Categoria"  validate:"required"`
	Description string `form:"description" label:"Descrição"  validate:"max=255"`
	// Só no cadastro: vira uma movimentação de entrada "Estoque inicial". Na edição é ignorada.
	InitialQuantity int `form:"initial_quantity" label:"Quantidade inicial" validate:"min=0,max=1000000"`
}

// NewSaveItemRequest preenche o formulário com os dados de um item existente (tela de edição).
func NewSaveItemRequest(item entity.Item) SaveItemRequest {
	return SaveItemRequest{
		Name:        item.Name,
		CategoryID:  int(item.CategoryID),
		Description: item.Description,
	}
}

// ToInput converte o request (dados da tela) na entrada que os casos de uso entendem.
func (saveRequest SaveItemRequest) ToInput() usecase.ItemInput {
	return usecase.ItemInput{
		CategoryID:      int64(saveRequest.CategoryID),
		Name:            saveRequest.Name,
		Description:     saveRequest.Description,
		InitialQuantity: saveRequest.InitialQuantity,
	}
}
