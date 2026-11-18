package request

import (
	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/usecase"
)

// SaveCategoryRequest valida o formulário de cadastro e de edição de categoria:
// POST /categories e POST /categories/{id}
type SaveCategoryRequest struct {
	Name        string `form:"name"        label:"Nome"      validate:"required,max=100"`
	Description string `form:"description" label:"Descrição" validate:"max=255"`
}

// NewSaveCategoryRequest preenche o formulário com os dados de uma categoria existente (tela de edição).
func NewSaveCategoryRequest(category entity.Category) SaveCategoryRequest {
	return SaveCategoryRequest{Name: category.Name, Description: category.Description}
}

// ToInput converte o request (dados da tela) na entrada que os casos de uso entendem.
func (saveRequest SaveCategoryRequest) ToInput() usecase.CategoryInput {
	return usecase.CategoryInput{Name: saveRequest.Name, Description: saveRequest.Description}
}
