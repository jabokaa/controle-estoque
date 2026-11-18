package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// GetItem busca um item pelo ID (usado, por exemplo, para preencher o formulário de edição).
type GetItem struct {
	repo repository.ItemRepository
}

func NewGetItem(repo repository.ItemRepository) *GetItem {
	return &GetItem{repo: repo}
}

// Execute devolve repository.ErrNotFound se o item não existir.
func (getItem *GetItem) Execute(ctx context.Context, id int64) (entity.Item, error) {
	return getItem.repo.FindByID(ctx, id)
}
