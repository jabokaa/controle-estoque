package repository

import (
	"context"

	"controle-estoque/internal/domain/entity"
)

// CategorySortField é um campo pelo qual a listagem de categorias pode ser ordenada.
type CategorySortField string

const (
	CategorySortByName        CategorySortField = "name"
	CategorySortByDescription CategorySortField = "description"
	CategorySortByCreatedAt   CategorySortField = "created_at"
)

// CategoryListQuery reúne o filtro, a ordenação e a página da listagem de categorias.
type CategoryListQuery struct {
	Name          string // filtra pelas categorias cujo nome contém este texto
	SortBy        CategorySortField
	SortDirection SortDirection
	Page          PageRequest
}

// CategoryRepository é o contrato de acesso às categorias.
type CategoryRepository interface {
	List(ctx context.Context, query CategoryListQuery) (Page[entity.Category], error)
}
