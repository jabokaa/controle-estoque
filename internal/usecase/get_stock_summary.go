package usecase

import (
	"context"

	"controle-estoque/internal/domain/repository"
)

// GetStockSummary devolve o resumo do estoque: total de unidades, itens cadastrados e itens sem estoque.
type GetStockSummary struct {
	repo repository.ItemRepository
}

func NewGetStockSummary(repo repository.ItemRepository) *GetStockSummary {
	return &GetStockSummary{repo: repo}
}

func (getStockSummary *GetStockSummary) Execute(ctx context.Context) (repository.StockSummary, error) {
	return getStockSummary.repo.Summary(ctx)
}
