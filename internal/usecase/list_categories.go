package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// ListCategories lista as categorias com filtro, ordenação e paginação.
type ListCategories struct {
	repo repository.CategoryRepository
}

func NewListCategories(repo repository.CategoryRepository) *ListCategories {
	return &ListCategories{repo: repo}
}

func (listCategories *ListCategories) Execute(ctx context.Context, query repository.CategoryListQuery) (repository.Page[entity.Category], error) {
	// Regras gerais de paginação (pagination.go)
	query.Page = normalizePageRequest(query.Page)
	query.SortDirection = normalizeSortDirection(query.SortDirection)

	// Regra específica de categorias: ordena por nome quando nada for pedido
	if query.SortBy == "" {
		query.SortBy = repository.CategorySortByName
	}

	return paginate(query.Page, func(pageRequest repository.PageRequest) (repository.Page[entity.Category], error) {
		query.Page = pageRequest
		return listCategories.repo.List(ctx, query)
	})
}
