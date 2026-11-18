package repository

import (
	"context"

	"controle-estoque/internal/domain/entity"
)

// ItemSortField é um campo pelo qual a listagem de itens pode ser ordenada.
type ItemSortField string

const (
	ItemSortByName      ItemSortField = "name"
	ItemSortByCategory  ItemSortField = "category" // pelo nome da categoria
	ItemSortByQuantity  ItemSortField = "quantity"
	ItemSortByCreatedAt ItemSortField = "created_at"
)

// ItemListQuery reúne os filtros, a ordenação e a página da listagem de itens.
type ItemListQuery struct {
	Name          string // filtra pelos itens cujo nome contém este texto
	CategoryID    int64  // filtra pelos itens desta categoria (0 = todas)
	SortBy        ItemSortField
	SortDirection SortDirection
	Page          PageRequest
}

// StockSummary é o resumo do estoque (os números dos cartões de resumo).
type StockSummary struct {
	TotalUnits      int64 // soma das quantidades de todos os itens
	ItemCount       int64 // quantos itens estão cadastrados
	OutOfStockCount int64 // quantos itens estão com estoque zero
}

// ItemRepository é o contrato de acesso aos itens.
// As consultas devolvem os itens com a Category preenchida.
type ItemRepository interface {
	List(ctx context.Context, query ItemListQuery) (Page[entity.Item], error)

	// ListAll devolve todos os itens, por nome (ex.: para um campo de seleção).
	ListAll(ctx context.Context) ([]entity.Item, error)

	// Summary devolve o resumo do estoque.
	Summary(ctx context.Context) (StockSummary, error)

	// FindByID devolve ErrNotFound se o item não existir.
	FindByID(ctx context.Context, id int64) (entity.Item, error)

	// ExistsByName diz se já existe outro item com esse nome (sem diferenciar maiúsculas),
	// ignorando o item ignoreID (use 0 no cadastro, e o próprio ID na edição).
	ExistsByName(ctx context.Context, name string, ignoreID int64) (bool, error)

	// CountByCategory diz quantos itens pertencem à categoria.
	CountByCategory(ctx context.Context, categoryID int64) (int64, error)

	// Create grava o item e preenche ID, CreatedAt e UpdatedAt.
	Create(ctx context.Context, item *entity.Item) error

	// Update grava os dados cadastrais (não a quantidade, que muda por movimentação)
	// e atualiza UpdatedAt. Devolve ErrNotFound se o item não existir.
	Update(ctx context.Context, item *entity.Item) error

	// Delete devolve ErrNotFound se o item não existir.
	Delete(ctx context.Context, id int64) error
}
