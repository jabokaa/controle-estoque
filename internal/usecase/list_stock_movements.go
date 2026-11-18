package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// ListStockMovements lista o histórico de movimentações com filtros (item, tipo) e paginação.
type ListStockMovements struct {
	repo repository.StockMovementRepository
}

func NewListStockMovements(repo repository.StockMovementRepository) *ListStockMovements {
	return &ListStockMovements{repo: repo}
}

func (listStockMovements *ListStockMovements) Execute(ctx context.Context, query repository.StockMovementListQuery) (repository.Page[entity.StockMovement], error) {
	// Regras gerais de paginação (pagination.go)
	query.Page = normalizePageRequest(query.Page)

	// Regra específica: tipo desconhecido vira "todos"
	if query.Type != entity.MovementIn && query.Type != entity.MovementOut {
		query.Type = ""
	}

	return paginate(query.Page, func(pageRequest repository.PageRequest) (repository.Page[entity.StockMovement], error) {
		query.Page = pageRequest
		return listStockMovements.repo.List(ctx, query)
	})
}
