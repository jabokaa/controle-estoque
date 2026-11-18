package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// DeleteCategory exclui uma categoria.
type DeleteCategory struct {
	repo repository.CategoryRepository
}

func NewDeleteCategory(repo repository.CategoryRepository) *DeleteCategory {
	return &DeleteCategory{repo: repo}
}

// Execute exclui a categoria e devolve os dados dela (para a mensagem "Categoria X excluída").
// Devolve repository.ErrNotFound se a categoria não existir.
//
// Quando existirem itens, é aqui que entra a regra "não excluir categoria com itens".
func (deleteCategory *DeleteCategory) Execute(ctx context.Context, id int64) (entity.Category, error) {
	category, err := deleteCategory.repo.FindByID(ctx, id)
	if err != nil {
		return category, err
	}
	return category, deleteCategory.repo.Delete(ctx, id)
}
