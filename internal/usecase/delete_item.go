package usecase

import (
	"context"
	"errors"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// ErrItemHasMovements: um item com movimentações não pode ser excluído
// (o histórico de entradas e saídas seria perdido).
var ErrItemHasMovements = errors.New("o item tem movimentações de estoque")

// DeleteItem exclui um item.
type DeleteItem struct {
	repo         repository.ItemRepository
	movementRepo repository.StockMovementRepository
}

func NewDeleteItem(repo repository.ItemRepository, movementRepo repository.StockMovementRepository) *DeleteItem {
	return &DeleteItem{repo: repo, movementRepo: movementRepo}
}

// Execute exclui o item e devolve os dados dele (para a mensagem "Item X excluído").
// Devolve repository.ErrNotFound se o item não existir,
// e ErrItemHasMovements se ele tiver movimentações.
func (deleteItem *DeleteItem) Execute(ctx context.Context, id int64) (entity.Item, error) {
	item, err := deleteItem.repo.FindByID(ctx, id)
	if err != nil {
		return item, err
	}

	movementCount, err := deleteItem.movementRepo.CountByItem(ctx, id)
	if err != nil {
		return item, err
	}
	if movementCount > 0 {
		return item, ErrItemHasMovements
	}

	return item, deleteItem.repo.Delete(ctx, id)
}
