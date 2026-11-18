package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// GetCategory busca uma categoria pelo ID (usado, por exemplo, para preencher o formulário de edição).
type GetCategory struct {
	repo repository.CategoryRepository
}

func NewGetCategory(repo repository.CategoryRepository) *GetCategory {
	return &GetCategory{repo: repo}
}

// Execute devolve repository.ErrNotFound se a categoria não existir.
func (getCategory *GetCategory) Execute(ctx context.Context, id int64) (entity.Category, error) {
	return getCategory.repo.FindByID(ctx, id)
}
