package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// DeleteItem exclui um item.
type DeleteItem struct {
	repo repository.ItemRepository
}

func NewDeleteItem(repo repository.ItemRepository) *DeleteItem {
	return &DeleteItem{repo: repo}
}

// Execute exclui o item e devolve os dados dele (para a mensagem "Item X excluído").
// Devolve repository.ErrNotFound se o item não existir.
func (deleteItem *DeleteItem) Execute(ctx context.Context, id int64) (entity.Item, error) {
	item, err := deleteItem.repo.FindByID(ctx, id)
	if err != nil {
		return item, err
	}
	return item, deleteItem.repo.Delete(ctx, id)
}
