package usecase

import (
	"context"
	"errors"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// Regras que dependem do banco, por isso ficam no caso de uso e não na entidade.
var (
	ErrItemNameTaken        = errors.New("já existe um item com esse nome")
	ErrItemCategoryNotFound = errors.New("a categoria escolhida não existe")
)

// initialStockNote é a observação da entrada lançada automaticamente no cadastro do item.
const initialStockNote = "Estoque inicial"

// ItemInput são os dados que o usuário informa ao cadastrar ou editar um item.
type ItemInput struct {
	CategoryID  int64
	Name        string
	Description string
	// InitialQuantity só vale no cadastro: vira uma movimentação de entrada.
	// Na edição é ignorada (a quantidade muda por movimentação).
	InitialQuantity int
}

func (input ItemInput) toItemData() entity.ItemData {
	return entity.ItemData{
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Description: input.Description,
	}
}

// ensureItemNameIsAvailable devolve ErrItemNameTaken se o nome já estiver em uso
// por outro item (ignoreID é o próprio item, na edição).
func ensureItemNameIsAvailable(ctx context.Context, repo repository.ItemRepository, name string, ignoreID int64) error {
	exists, err := repo.ExistsByName(ctx, name, ignoreID)
	if err != nil {
		return err
	}
	if exists {
		return ErrItemNameTaken
	}
	return nil
}

// findItemCategory busca a categoria escolhida; devolve ErrItemCategoryNotFound se ela não existir.
func findItemCategory(ctx context.Context, categoryRepo repository.CategoryRepository, categoryID int64) (entity.Category, error) {
	category, err := categoryRepo.FindByID(ctx, categoryID)
	if errors.Is(err, repository.ErrNotFound) {
		return category, ErrItemCategoryNotFound
	}
	return category, err
}

// CreateItem cadastra um item novo e lança o estoque inicial como movimentação de entrada.
type CreateItem struct {
	repo         repository.ItemRepository
	categoryRepo repository.CategoryRepository
	movementRepo repository.StockMovementRepository
}

func NewCreateItem(repo repository.ItemRepository, categoryRepo repository.CategoryRepository, movementRepo repository.StockMovementRepository) *CreateItem {
	return &CreateItem{repo: repo, categoryRepo: categoryRepo, movementRepo: movementRepo}
}

func (createItem *CreateItem) Execute(ctx context.Context, input ItemInput) (entity.Item, error) {
	if input.InitialQuantity < 0 {
		return entity.Item{}, entity.ErrItemQuantityNegative
	}
	item, err := entity.NewItem(input.toItemData())
	if err != nil {
		return item, err
	}
	category, err := findItemCategory(ctx, createItem.categoryRepo, item.CategoryID)
	if err != nil {
		return item, err
	}
	if err := ensureItemNameIsAvailable(ctx, createItem.repo, item.Name, 0); err != nil {
		return item, err
	}
	if err := createItem.repo.Create(ctx, &item); err != nil {
		return item, err
	}
	item.Category = category

	// Estoque inicial: entra como movimentação, para o saldo bater com o histórico.
	if input.InitialQuantity > 0 {
		movement, err := createItem.movementRepo.Register(ctx, item.ID, func(stored *entity.Item) (entity.StockMovement, error) {
			return entity.NewStockMovement(stored, entity.MovementIn, input.InitialQuantity, initialStockNote)
		})
		if err != nil {
			return item, err
		}
		item.Quantity = movement.BalanceAfter
	}
	return item, nil
}

// UpdateItem altera os dados cadastrais de um item existente.
type UpdateItem struct {
	repo         repository.ItemRepository
	categoryRepo repository.CategoryRepository
}

func NewUpdateItem(repo repository.ItemRepository, categoryRepo repository.CategoryRepository) *UpdateItem {
	return &UpdateItem{repo: repo, categoryRepo: categoryRepo}
}

func (updateItem *UpdateItem) Execute(ctx context.Context, id int64, input ItemInput) (entity.Item, error) {
	item, err := updateItem.repo.FindByID(ctx, id)
	if err != nil {
		return item, err
	}
	if err := item.Change(input.toItemData()); err != nil {
		return item, err
	}
	category, err := findItemCategory(ctx, updateItem.categoryRepo, item.CategoryID)
	if err != nil {
		return item, err
	}
	if err := ensureItemNameIsAvailable(ctx, updateItem.repo, item.Name, item.ID); err != nil {
		return item, err
	}
	if err := updateItem.repo.Update(ctx, &item); err != nil {
		return item, err
	}
	item.Category = category
	return item, nil
}
