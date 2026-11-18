package usecase

import (
	"context"
	"errors"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// ErrCategoryHasItems: uma categoria com itens não pode ser excluída
// (os itens ficariam sem categoria).
var ErrCategoryHasItems = errors.New("a categoria tem itens cadastrados")

// DeleteCategory exclui uma categoria.
type DeleteCategory struct {
	repo     repository.CategoryRepository
	itemRepo repository.ItemRepository
}

func NewDeleteCategory(repo repository.CategoryRepository, itemRepo repository.ItemRepository) *DeleteCategory {
	return &DeleteCategory{repo: repo, itemRepo: itemRepo}
}

// Execute exclui a categoria e devolve os dados dela (para a mensagem "Categoria X excluída").
// Devolve repository.ErrNotFound se a categoria não existir,
// e ErrCategoryHasItems se ela ainda tiver itens.
func (deleteCategory *DeleteCategory) Execute(ctx context.Context, id int64) (entity.Category, error) {
	category, err := deleteCategory.repo.FindByID(ctx, id)
	if err != nil {
		return category, err
	}

	itemCount, err := deleteCategory.itemRepo.CountByCategory(ctx, id)
	if err != nil {
		return category, err
	}
	if itemCount > 0 {
		return category, ErrCategoryHasItems
	}

	return category, deleteCategory.repo.Delete(ctx, id)
}
