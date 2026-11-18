package repository

import (
	"context"
	"errors"

	"controle-estoque/internal/domain/entity"
)

// ErrNotFound é devolvido quando o registro procurado não existe.
var ErrNotFound = errors.New("registro não encontrado")

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

	// ListAll devolve todas as categorias, por nome (ex.: para um campo de seleção).
	ListAll(ctx context.Context) ([]entity.Category, error)

	// FindByID devolve ErrNotFound se a categoria não existir.
	FindByID(ctx context.Context, id int64) (entity.Category, error)

	// ExistsByName diz se já existe outra categoria com esse nome (sem diferenciar maiúsculas),
	// ignorando a categoria ignoreID (use 0 no cadastro, e o próprio ID na edição).
	ExistsByName(ctx context.Context, name string, ignoreID int64) (bool, error)

	// Create grava a categoria e preenche ID, CreatedAt e UpdatedAt.
	Create(ctx context.Context, category *entity.Category) error

	// Update grava as alterações e atualiza UpdatedAt. Devolve ErrNotFound se a categoria não existir.
	Update(ctx context.Context, category *entity.Category) error

	// Delete devolve ErrNotFound se a categoria não existir.
	Delete(ctx context.Context, id int64) error
}
