package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// ListItems lista os itens com filtros (nome, categoria), ordenação e paginação.
type ListItems struct {
	repo repository.ItemRepository
}

func NewListItems(repo repository.ItemRepository) *ListItems {
	return &ListItems{repo: repo}
}

func (listItems *ListItems) Execute(ctx context.Context, query repository.ItemListQuery) (repository.Page[entity.Item], error) {
	// Regras gerais de paginação (pagination.go)
	query.Page = normalizePageRequest(query.Page)
	query.SortDirection = normalizeSortDirection(query.SortDirection)

	// Regra específica de itens: ordena por nome quando nada for pedido
	if query.SortBy == "" {
		query.SortBy = repository.ItemSortByName
	}

	return paginate(query.Page, func(pageRequest repository.PageRequest) (repository.Page[entity.Item], error) {
		query.Page = pageRequest
		return listItems.repo.List(ctx, query)
	})
}
